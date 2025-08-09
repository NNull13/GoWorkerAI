package runtime

import (
	"context"
	"log"
	"runtime/debug"
	"sync"

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
	events     chan Event
	activeTask bool
	cancelFunc context.CancelFunc
	db         storage.Interface
}

func NewRuntime(worker workers.Interface, model models.Interface, initialActions map[string]tools.Tool,
	db storage.Interface, activeTask bool) *Runtime {
	return &Runtime{
		worker:     worker,
		model:      model,
		events:     make(chan Event, 100),
		toolkit:    initialActions,
		activeTask: activeTask,
		db:         db,
	}
}

func (r *Runtime) Start(ctx context.Context) {
	r.mu.Lock()
	if r.activeTask {
		cctx, cancel := context.WithCancel(ctx)
		r.cancelFunc = cancel
		go r.runTask(cctx)
	}
	r.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-r.events:
			if !ok {
				return
			}
			r.handleEvent(ev)
		}
	}
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
	default:
		log.Print("⚠️ Event queue is full, dropping event")
	}
}

func (r *Runtime) runTask(ctx context.Context) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("❌ Panic recovered in runTask: %v\nStack trace:\n%s", rec, debug.Stack())
		}
	}()

	r.mu.Lock()
	currentWorker := r.worker
	r.activeTask = false
	r.mu.Unlock()

	if currentWorker == nil {
		log.Printf("⚠️ No worker assigned, cannot run task.\n")
		return
	}

	task := currentWorker.GetTask()
	if task == nil {
		log.Printf("⚠️ Worker returned nil task.\n")
		return
	}
	taskID := task.ID.String()

	log.Printf("▶️ Starting task: %s\n", task.Task)
	taskInfo := currentWorker.TaskInformation()
	plan, err := r.model.Think(ctx, currentWorker.PromptPlan(taskInfo), 0.25, -1)
	if err != nil {
		log.Printf("❌ Error generating initial plan: %v\n", err)
		return
	}
	log.Printf("✅ Plan generated:\n%s\n", plan)

	steps := utils.SplitPlanIntoSteps(plan)
	if len(steps) == 0 {
		log.Printf("⚠️ Could not split plan into steps, aborting.\n")
		return
	}
	log.Printf("✅ Detected %d step(s) in plan.\n", len(steps))

	var summary string
	for stepIndex, step := range steps {
		completed, newSummary := r.executeStep(ctx, currentWorker, taskID, stepIndex, step, task.MaxIterations, summary, plan, steps)
		if !completed {
			log.Printf("❌ Step %d could not be completed, continue with task execution.\n", stepIndex+1)
		}
		summary = newSummary
	}

	history, err := r.db.GetHistoryByTaskID(ctx, taskID, 0)
	if err != nil {
		log.Printf("❌ Error final GetHistoryByTaskID: %s\n", err.Error())
	}
	finalSummary, _ := r.model.GenerateSummary(ctx, history)
	log.Printf("✅ Final summary: %s\n", finalSummary)

	decision, err := r.model.YesOrNo(ctx, currentWorker.PromptValidation(plan, finalSummary))
	if err != nil {
		log.Printf("❌ Error in final validation: %v\n", err)
		return
	}
	if decision {
		log.Printf("🎉 Task successfully completed: %s\n", task.Task)
	} else {
		log.Printf("🚧 Task is not fully completed according to validation: %s\n", task.Task)
	}
}

func (r *Runtime) executeStep(ctx context.Context, worker workers.Interface, taskID string, stepIndex int, step string,
	maxIterations int, currentSummary, plan string, steps []string) (bool, string) {
	for attempt := 0; attempt < maxIterations; attempt++ {
		select {
		case <-ctx.Done():
			log.Printf("⚠️ Context canceled, stopping task.\n")
			return false, currentSummary
		default:
			log.Printf("▶️ Executing step %d (attempt %d): %s\n", stepIndex+1, attempt+1, step)
			prompt := worker.PromptSegmentedStep(steps, stepIndex, currentSummary)
			response, perr := r.model.Process(ctx, prompt, r.Toolkit(), taskID)
			if perr != nil {
				log.Printf("❌ Error processing step %d attempt %d: %v\n", stepIndex+1, attempt+1, perr)
				continue
			}
			log.Printf("✅ Step %d response:\n%s\n", stepIndex+1, response)

			history, err := r.db.GetHistoryByTaskID(ctx, taskID, stepIndex)
			if err != nil {
				log.Printf("❌ Error retrieving history for step %d: %v\n", stepIndex+1, err)
				continue
			}

			currentSummary, _ = r.model.GenerateSummary(ctx, history)
			log.Printf("ℹ️ Current step %d summary: %s\n", stepIndex+1, currentSummary)

			stepCompleted, err := r.model.YesOrNo(ctx, worker.PromptValidation(plan, currentSummary))
			if err != nil {
				log.Printf("❌ Error validating step %d: %v\n", stepIndex+1, err)
				continue
			}
			if stepCompleted {
				log.Printf("✅ Step %d completed successfully\n", stepIndex+1)
				return true, currentSummary
			}
		}
	}
	return false, currentSummary
}
