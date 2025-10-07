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
	"GoWorkerAI/app/teams"
)

type Runtime struct {
	team       *teams.Team
	mu         sync.RWMutex
	model      models.Interface
	db         storage.Interface
	events     chan Event
	activeTask atomic.Bool
	cancelFunc context.CancelFunc
	context    context.Context
}

func NewRuntime(t *teams.Team, m models.Interface, db storage.Interface) *Runtime {
	rt := &Runtime{
		team:   t,
		model:  m,
		events: make(chan Event, 1024),
		db:     db,
	}
	return rt
}

func handlePanic() {
	if rec := recover(); rec != nil {
		log.Printf("❌ Panic recovered in runTask: %v\nStack trace:\n%s", rec, debug.Stack())
	}
	os.Exit(666)
}

func (r *Runtime) Start(ctx context.Context) {
	runtimeCtx, runtimeCancel := context.WithCancel(ctx)
	taskRunning := r.activeTask.Load()
	defer handlePanic()

	if !taskRunning && r.team.Task != nil && r.team.Task.Description != "" {
		r.activeTask.Store(true)
		go func() {
			defer r.activeTask.Store(false)

			if err := r.runTask(runtimeCtx, runtimeCancel); err != nil {
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
				log.Println("runtime: Event channel closed.")
			}
			r.handleEvent(ev)
		}
	}
}

func (r *Runtime) StopRuntime() {
	r.mu.Lock()
	r.team.Task = nil
	for _, member := range r.team.Members {
		member.Task = nil
	}
	if r.cancelFunc != nil {
		r.cancelFunc()
		r.cancelFunc = nil
	}
	r.mu.Unlock()
}

func (r *Runtime) QueueEvent(event Event) {
	r.events <- event
}

func (r *Runtime) runTask(ctx context.Context, cancel context.CancelFunc) error {
	r.mu.Lock()
	r.context = ctx
	r.cancelFunc = cancel
	team := r.team
	task := team.Task
	leader := team.GetLeader()
	teamOptions := r.team.GetMembersOptions()
	r.mu.Unlock()

	if task == nil {
		log.Println("⚠️ Worker returned nil task.")
		return nil
	}

	log.Printf("▶️ Starting task: %s", task.Description)
	messages := models.CreateMessages(task.Description, leader.Prompt(models.PlanSystemPrompt))
	planText, err := r.model.Think(ctx, messages, 0.25, -1)
	if err != nil {
		log.Printf("❌ Error generating plan: %v\n", err)
		return err
	}

	team.Audits.Printf("✅ Plan generated:\n%s\n", planText)

	steps := strings.Split(planText, "\n")
	if len(steps) == 0 {
		log.Println("⚠️ Could not split plan into steps, aborting.")
		return nil
	}
	log.Printf("✅ Detected %d step(s) in plan.\n", len(steps))

	var i int
	var summary string
	var history []storage.Record
	for _, step := range steps {
		i++
		var delegateAction *models.DelegateAction
		prompt := leader.Prompt(summary)
		delegateAction, err = r.model.Delegate(ctx, teamOptions, step, prompt)
		if err != nil || delegateAction == nil {
			log.Printf("❌ Skipping step %d. Error delegating: %v\n", i, err)
			continue
		}

		worker := r.team.GetMember(delegateAction.Worker)
		if worker == nil {
			log.Printf("❌ Skipping step %d. Worker %s not found.\n", i, delegateAction.Worker)
			continue
		}
		log.Printf("Worker %s found.\n", delegateAction.Worker)
		team.Audits.Printf("▶️ Delegating step %d: %s to: %s", i, step, delegateAction.Worker)
		team.Audits.Printf("✅ Task assigned: %s\n", delegateAction.Task)

		prompt = worker.Prompt(delegateAction.Context)
		messages = models.CreateMessages(delegateAction.Task, prompt)
		_, err = r.model.Process(ctx, worker.Key, team.Audits.Logger, messages, worker.GetToolKit(), task.ID.String(), i)
		if err != nil {
			log.Printf("❌ Skipping step %d. Error processing: %v\n", i, err)
			continue
		}

		history, _ = r.db.GetHistoryByTaskID(ctx, task.ID.String(), -1)
		summary = storage.RecordListToString(history, 100)
		log.Printf("Summary after step%d: %s\n", i, summary)
	}

	log.Printf("✅ Description completed: %s\n", task.Description)
	r.team.Close()

	return nil
}

func (r *Runtime) GetTaskStatus(ctx context.Context) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	task := r.team.Task
	if task == nil {
		return "No task assigned to the team."
	}
	history, _ := r.db.GetHistoryByTaskID(ctx, task.ID.String(), -1)
	finalSummary, _ := r.model.GenerateSummary(ctx, task.Description, history)
	return finalSummary
}
