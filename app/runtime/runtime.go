package runtime

import (
	"context"
	"log"
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

type stepCtx struct {
	TaskID      string
	Task        string
	Index       int
	Plan        []string
	PrevSummary string
	MaxAttempts int
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
	case <-time.After(100 * time.Millisecond):
		log.Printf("⚠️ Event queue is full, dropping event, task: %v", event.Task)
	}
}

func (r *Runtime) runTask(ctx context.Context) error {
	r.mu.RLock()
	currentWorker := r.worker
	tk := cloneToolkit(r.toolkit)
	r.mu.RUnlock()

	if currentWorker == nil {
		log.Println("⚠️ No worker assigned, cannot run task.")
		return nil
	}
	task := currentWorker.GetTask()
	if task == nil {
		log.Println("⚠️ Worker returned nil task.")
		return nil
	}

	taskID := task.ID.String()
	log.Printf("▶️ Starting task: %s\n", task.Task)

	planText, err := r.model.Think(ctx, currentWorker.PromptPlan(currentWorker.TaskInformation()), 0.25, -1)
	if err != nil {
		log.Printf("❌ Error generating initial plan: %v\n", err)
		return err
	}
	log.Printf("✅ Plan generated:\n%s\n", planText)

	steps := utils.SplitPlanIntoSteps(planText)
	if len(steps) == 0 {
		log.Println("⚠️ Could not split plan into steps, aborting.")
		return nil
	}
	log.Printf("✅ Detected %d step(s) in plan.\n", len(steps))

	var runningSummary string
	for i, step := range steps {
		ok, newSummary := r.runStepWithValidation(ctx, stepCtx{
			TaskID:      taskID,
			Task:        step,
			Index:       i,
			Plan:        steps,
			PrevSummary: runningSummary,
			MaxAttempts: task.MaxIterations,
		}, currentWorker, tk)
		if !ok {
			log.Printf("❌ Step %d could not be completed, continuing.\n", i+1)
		}
		runningSummary = newSummary
	}

	history, err := r.db.GetHistoryByTaskID(ctx, taskID, -1) //all steps
	if err != nil {
		log.Printf("❌ Error final GetHistoryByTaskID: %s\n", err)
	}
	finalSummary, _ := r.model.GenerateSummary(ctx, history)
	log.Printf("✅ Final summary: %s\n", finalSummary)

	finalOK, err := r.model.YesOrNo(ctx, currentWorker.PromptValidation(planText, finalSummary))
	if err != nil {
		log.Printf("❌ Error in final validation: %v\n", err)
		return err
	}

	if finalOK {
		log.Printf("🎉 Task successfully completed: %s\n", task.Task)
	} else {
		log.Printf("🚧 Task is not fully completed according to validation: %s\n", task.Task)
	}
	return nil
}

func (r *Runtime) runStepWithValidation(ctx context.Context, sc stepCtx, worker workers.Interface,
	toolkit map[string]tools.Tool) (bool, string) {
	summary := sc.PrevSummary

	for attempt := 1; attempt <= sc.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			log.Println("⚠️ Context canceled, stopping task.")
			return false, summary
		default:
		}

		step := sc.Plan[sc.Index]
		log.Printf("▶️ Executing step %d (attempt %d): %s\n", sc.Index+1, attempt, step)

		prompt := worker.PromptSegmentedStep(sc.Plan, sc.Index, summary, worker.GetPreamble())

		resp, err := r.model.Process(ctx, prompt, toolkit, sc.TaskID, sc.Index)
		if err != nil {
			log.Printf("❌ Error processing step %d attempt %d: %v\n", sc.Index+1, attempt, err)
			continue
		}

		log.Printf("✅ Step %d response: %s", sc.Index+1, resp)

		if summary, err = r.stepSummary(ctx, sc.TaskID, sc.Index); err != nil {
			continue
		}

		log.Printf("ℹ️ Current step %d summary: %s\n", sc.Index+1, summary)

		ok, err := r.model.YesOrNo(ctx, worker.PromptValidation(sc.Task, summary))
		if err != nil {
			log.Printf("❌ Error validating step %d (LLM): %v\n", sc.Index+1, err)
			continue
		}

		if ok {
			log.Printf("✅ Step %d completed, task: %s\n", sc.Index+1, sc.Task)
			return true, summary
		}

		log.Printf("❌ Step %d not completed, task: %s\n", sc.Index+1, sc.Task)
	}

	return false, summary
}

func (r *Runtime) stepSummary(ctx context.Context, taskID string, stepIdx int) (string, error) {
	history, err := r.db.GetHistoryByTaskID(ctx, taskID, stepIdx)
	if err != nil {
		return "", err
	}
	if len(history) == 0 {
		return "", nil
	}
	return r.model.GenerateSummary(ctx, history)
}

func cloneToolkit(src map[string]tools.Tool) map[string]tools.Tool {
	tk := make(map[string]tools.Tool, len(src))
	for k, v := range src {
		tk[k] = v
	}
	return tk
}
