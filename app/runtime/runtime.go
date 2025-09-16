package runtime

import (
	"context"
	"log"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"

	"GoWorkerAI/app/models"
	"GoWorkerAI/app/storage"
	"GoWorkerAI/app/tools"
	"GoWorkerAI/app/workers"
)

type Runtime struct {
	mu         sync.RWMutex
	model      models.Interface
	worker     workers.Interface
	toolkit    map[string]tools.Tool
	db         storage.Interface
	events     chan Event
	activeTask atomic.Bool
	cancelFunc context.CancelFunc
	context    context.Context
	audits     *AuditLogger
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
	db storage.Interface, audits *AuditLogger) *Runtime {
	rt := &Runtime{
		worker:  w,
		model:   m,
		events:  make(chan Event, 1024),
		toolkit: make(map[string]tools.Tool, len(initial)),
		db:      db,
		audits:  audits,
	}
	for k, v := range initial {
		rt.toolkit[k] = v
	}
	return rt
}

func handlePanic(r *Runtime) {
	if rec := recover(); rec != nil {
		r.audits.Printf("❌ Panic recovered in runTask: %v\nStack trace:\n%s", rec, debug.Stack())
	}
	os.Exit(666)
}

func (r *Runtime) Start(ctx context.Context) {
	runtimeCtx, runtimeCancel := context.WithCancel(ctx)

	r.mu.RLock()
	taskRunning := r.activeTask.Load()
	worker := r.worker
	r.mu.RUnlock()

	if !taskRunning && worker != nil {
		task := worker.GetTask()
		if task.Task != "" {
			r.activeTask.Store(true)
			go func() {
				defer func() {
					r.activeTask.Store(false)
					defer handlePanic(r)
				}()
				if err := r.runTask(runtimeCtx, runtimeCancel); err != nil {
					r.audits.Printf("Error running task: %v", err)
				}
			}()
		}
	}

	for {
		select {
		case <-runtimeCtx.Done():
			return
		case ev, ok := <-r.events:
			if !ok {
				log.Println("runtime: Event channel closed.")
				return
			}
			defer handlePanic(r)
			r.handleEvent(ev)
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
	r.events <- event
}

func (r *Runtime) runTask(ctx context.Context, cancel context.CancelFunc) error {
	r.mu.Lock()
	r.context = ctx
	r.cancelFunc = cancel
	currentWorker := r.worker
	tk := cloneToolkit(r.toolkit)
	r.mu.Unlock()

	if currentWorker == nil {
		r.audits.Println("⚠️ No worker assigned, cannot run task.")
		return nil
	}
	task := currentWorker.GetTask()
	if task == nil {
		r.audits.Println("⚠️ Worker returned nil task.")
		return nil
	}

	taskID := task.ID.String()
	log.Printf("▶️ Starting task: %s", task.Task)

	planText, err := r.model.Think(ctx, currentWorker.PromptPlan(currentWorker.TaskInformation()), 0.25, -1)
	if err != nil {
		r.audits.Printf("❌ Error generating initial plan: %v\n", err)
		return err
	}
	log.Printf("✅ Plan generated:\n%s\n", planText)

	steps := strings.Split(planText, "\n")
	if len(steps) == 0 {
		log.Println("⚠️ Could not split plan into steps, aborting.")
		return nil
	}
	log.Printf("✅ Detected %d step(s) in plan.\n", len(steps))

	var runningSummary string
	for i, step := range steps {
		stepData := stepCtx{
			TaskID:      taskID,
			Task:        step,
			Index:       i + 1,
			Plan:        steps,
			PrevSummary: runningSummary,
			MaxAttempts: task.MaxIterations,
		}
		ok, newSummary := r.runStepWithValidation(stepData, currentWorker, tk)
		if !ok {
			r.audits.Printf("❌ Step %d could not be completed, continuing.\n", i+1)
		}
		runningSummary = newSummary

		r.audits.file.Sync()
		if r.validateTaskCompletion(ctx, stepData, planText) {
			return nil
		}
	}

	r.audits.Printf("🚧 Task is not fully completed according to validation: %s\n", task.Task)
	r.audits.Close()
	return nil
}

func (r *Runtime) GetTaskStatus(ctx context.Context) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	task := r.worker.GetTask()
	if task == nil {
		return "No task assigned to worker."
	}
	auditLogs := r.audits.GetLastLogs(10000)
	history, _ := r.db.GetHistoryByTaskID(ctx, task.ID.String(), -1)
	finalSummary, _ := r.model.GenerateSummary(ctx, task.Task, auditLogs, history)
	return finalSummary
}

func (r *Runtime) validateTaskCompletion(ctx context.Context, stepData stepCtx, planText string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	currentWorker := r.worker
	auditLogs := r.audits.GetLastLogs(10000)
	history, _ := r.db.GetHistoryByTaskID(ctx, stepData.TaskID, stepData.Index)
	finalSummary, _ := r.model.GenerateSummary(ctx, stepData.Task, auditLogs, history)

	finalOK, reason, err := r.model.TrueOrFalse(ctx, currentWorker.PromptValidation(planText, finalSummary))
	if err != nil {
		r.audits.Printf("❌ Error in final validation: %v\n", err)
		return false
	}

	if finalOK {
		r.audits.Printf("🎉 Task successfully completed: \ntask:%s\nreason:%s", stepData.Task, reason)
		return true
	} else {
		r.audits.Printf("ℹ️ Task not fully completed yet, reason: %s", reason)
		return false
	}
}

func (r *Runtime) runStepWithValidation(sc stepCtx, worker workers.Interface,
	toolkit map[string]tools.Tool) (bool, string) {
	summary := sc.PrevSummary
	ctx := r.context

	for attempt := 1; attempt <= sc.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			r.audits.Println("⚠️ Context canceled, stopping task.")
			return false, summary
		default:
		}
		step := sc.Plan[sc.Index-1] // -1 because the index starts from 0

		// todo: Maybe it would be good to check if is necessary to run the step
		r.audits.Printf("▶️ Executing step %d (attempt %d): %s\n", sc.Index, attempt, step)

		prompt := worker.PromptSegmentedStep(step, summary, worker.GetPreamble())

		resp, err := r.model.Process(ctx, r.audits.Logger, prompt, toolkit, sc.TaskID, sc.Index)
		if err != nil {
			r.audits.Printf("❌ Error processing step %d attempt %d: %v\n", sc.Index, attempt, err)
			continue
		}

		r.audits.Printf("✅ Step %d response: %s", sc.Index, resp)

		auditLogs := r.audits.GetLastLogs(10000)
		history, _ := r.db.GetHistoryByTaskID(ctx, sc.TaskID, sc.Index)
		summary, err = r.model.GenerateSummary(ctx, step, auditLogs, history)
		if err != nil {
			r.audits.Printf("❌ Error generating summary for step %d: %v\n", sc.Index, err)
			continue
		}

		r.audits.Printf("ℹ️ Current step %d summary: %s\n", sc.Index, summary)

		ok, reason, err := r.model.TrueOrFalse(ctx, worker.PromptValidation(sc.Task, summary))
		if err != nil {
			r.audits.Printf("❌ Error validating step %d (LLM): %v\n", sc.Index, err)
			continue
		}

		if ok {
			r.audits.Printf("✅ Step %d completed, task: %s\n, reason: %s", sc.Index, sc.Task, reason)
			return true, summary
		} else {
			r.audits.Printf("❌ Step %d not completed, task: %s\n, reason: %s", sc.Index, sc.Task, reason)
		}
	}

	return false, summary
}

func cloneToolkit(src map[string]tools.Tool) map[string]tools.Tool {
	tk := make(map[string]tools.Tool, len(src))
	for k, v := range src {
		tk[k] = v
	}
	return tk
}
