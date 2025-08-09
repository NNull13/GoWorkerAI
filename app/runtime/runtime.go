package runtime

import (
	"context"
	"log"
	"math/rand"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"GoWorkerAI/app/models"
	"GoWorkerAI/app/storage"
	"GoWorkerAI/app/tools"
	"GoWorkerAI/app/utils"
	"GoWorkerAI/app/workers"
)

type Runtime struct {
	mu         sync.RWMutex
	worker     workers.Interface
	model      models.Interface
	toolkit    map[string]tools.Tool
	db         storage.Interface
	events     chan Event
	activeTask atomic.Bool
	cancelFunc context.CancelFunc
}

func NewRuntime(w workers.Interface, m models.Interface, initial map[string]tools.Tool,
	db storage.Interface, startTask bool) *Runtime {
	rt := &Runtime{
		worker:  w,
		model:   m,
		events:  make(chan Event, 1024),
		toolkit: make(map[string]tools.Tool, len(initial)),
		db:      db,
	}
	for k, v := range initial {
		rt.toolkit[k] = v
	}
	rt.activeTask.Store(startTask)
	return rt
}

func (r *Runtime) Start(ctx context.Context) {
	runtimeCtx, runtimeCancel := context.WithCancel(ctx)
	r.mu.Lock()
	r.cancelFunc = runtimeCancel
	startTask := r.activeTask.Load()
	worker := r.worker
	r.mu.Unlock()

	// Lanza tarea si corresponde, fuera del lock
	if startTask && worker != nil {
		r.mu.Lock()
		r.activeTask.Store(true)
		r.mu.Unlock()

		go func() {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("❌ Panic recovered in runTask: %v\nStack trace:\n%s", rec, debug.Stack())
				}
				r.StopRuntime()
			}()
			if err := r.runTask(runtimeCtx); err != nil {
				log.Printf("Error running task: %v", err)
			}
		}()
	}

	for {
		select {
		case <-runtimeCtx.Done():
			return
		case ev, ok := <-r.events:
			if !ok {
				return
			}
			func() {
				defer func() {
					if rec := recover(); rec != nil {
						log.Printf("❌ Panic recovered in handleEvent: %v\nStack trace:\n%s", rec, debug.Stack())
					}
				}()
				r.handleEvent(ev)
			}()
		}
	}
}

func (r *Runtime) StopRuntime() {
	r.mu.Lock()
	r.worker.SetTask(nil)
	if r.cancelFunc != nil {
		r.cancelFunc()
		r.cancelFunc = nil
	}
	r.mu.Unlock()
}

func (r *Runtime) AddTools(newActions []tools.Tool) {
	r.mu.Lock()
	if r.toolkit == nil {
		r.toolkit = make(map[string]tools.Tool)
	}
	for _, newAction := range newActions {
		r.toolkit[newAction.Name] = newAction
	}
	r.mu.Unlock()
}

func (r *Runtime) Toolkit() map[string]tools.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copy := make(map[string]tools.Tool, len(r.toolkit))
	for k, v := range r.toolkit {
		copy[k] = v
	}
	return copy
}

func (r *Runtime) QueueEvent(event Event) {
	select {
	case r.events <- event:
	case <-time.After(10 * time.Millisecond):
		log.Printf("⚠️ Event queue is full, dropping event, task: %v", event.Task)
	}
}

func (r *Runtime) runTask(ctx context.Context) error {
	r.mu.RLock()
	currentWorker := r.worker

	tk := make(map[string]tools.Tool, len(r.toolkit))
	for k, v := range r.toolkit {
		tk[k] = v
	}
	r.mu.RUnlock()

	if currentWorker == nil {
		log.Printf("⚠️ No worker assigned, cannot run task.\n")
		return nil
	}

	task := currentWorker.GetTask()
	if task == nil {
		log.Printf("⚠️ Worker returned nil task.\n")
		return nil
	}
	taskID := task.ID.String()
	log.Printf("▶️ Starting task: %s\n", task.Task)

	thinkCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	plan, err := r.model.Think(thinkCtx, currentWorker.PromptPlan(currentWorker.TaskInformation()), 0.25, -1)
	cancel()
	if err != nil {
		log.Printf("❌ Error generating initial plan: %v\n", err)
		return err
	}
	log.Printf("✅ Plan generated:\n%s\n", plan)

	steps := utils.SplitPlanIntoSteps(plan)
	if len(steps) == 0 {
		log.Printf("⚠️ Could not split plan into steps, aborting.\n")
		return nil
	}
	log.Printf("✅ Detected %d step(s) in plan.\n", len(steps))

	var summary string
	for i, step := range steps {
		completed, newSummary := r.executeStep(ctx, currentWorker, taskID, i, step, task.MaxIterations, summary, plan, steps, tk)
		if !completed {
			log.Printf("❌ Step %d could not be completed, continuing.\n", i+1)
		}
		summary = newSummary
	}

	history, err := r.db.GetHistoryByTaskID(ctx, taskID, 0)
	if err != nil {
		log.Printf("❌ Error final GetHistoryByTaskID: %s\n", err.Error())
	}
	sumCtx, sCancel := context.WithTimeout(ctx, 20*time.Second)
	finalSummary, _ := r.model.GenerateSummary(sumCtx, history)
	sCancel()
	log.Printf("✅ Final summary: %s\n", finalSummary)

	valCtx, vCancel := context.WithTimeout(ctx, 20*time.Second)
	decision, err := r.model.YesOrNo(valCtx, currentWorker.PromptValidation(plan, finalSummary))
	vCancel()
	if err != nil {
		log.Printf("❌ Error in final validation: %v\n", err)
		return err
	}
	if decision {
		log.Printf("🎉 Task successfully completed: %s\n", task.Task)
	} else {
		log.Printf("🚧 Task is not fully completed according to validation: %s\n", task.Task)
	}
	return nil
}

func (r *Runtime) executeStep(
	ctx context.Context,
	worker workers.Interface,
	taskID string,
	stepIndex int,
	step string,
	maxIterations int,
	currentSummary, plan string,
	steps []string,
	toolkit map[string]tools.Tool,
) (bool, string) {

	backoff := 250 * time.Millisecond

	for attempt := 0; attempt < maxIterations; attempt++ {
		select {
		case <-ctx.Done():
			log.Printf("⚠️ Context canceled, stopping task.\n")
			return false, currentSummary
		default:
		}

		log.Printf("▶️ Executing step %d (attempt %d): %s\n", stepIndex+1, attempt+1, step)
		prompt := worker.PromptSegmentedStep(steps, stepIndex, currentSummary)

		attemptCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		response, perr := r.model.Process(attemptCtx, prompt, toolkit, taskID)
		cancel()
		if perr != nil {
			log.Printf("❌ Error processing step %d attempt %d: %v\n", stepIndex+1, attempt+1, perr)
			time.Sleep(backoff + time.Duration(rand.Intn(100))*time.Millisecond)
			if backoff < 2*time.Second {
				backoff *= 2
			}
			continue
		}
		log.Printf("✅ Step %d response:\n%s\n", stepIndex+1, response)

		history, err := r.db.GetHistoryByTaskID(ctx, taskID, stepIndex)
		if err != nil {
			log.Printf("❌ Error retrieving history for step %d: %v\n", stepIndex+1, err)
			continue
		}

		sumCtx, sCancel := context.WithTimeout(ctx, 15*time.Second)
		currentSummary, _ = r.model.GenerateSummary(sumCtx, history)
		sCancel()
		log.Printf("ℹ️ Current step %d summary: %s\n", stepIndex+1, currentSummary)

		valCtx, vCancel := context.WithTimeout(ctx, 10*time.Second)
		stepCompleted, err := r.model.YesOrNo(valCtx, worker.PromptValidation(plan, currentSummary))
		vCancel()
		if err != nil {
			log.Printf("❌ Error validating step %d: %v\n", stepIndex+1, err)
			continue
		}
		if stepCompleted {
			log.Printf("✅ Step %d completed successfully\n", stepIndex+1)
			return true, currentSummary
		}
	}
	return false, currentSummary
}
