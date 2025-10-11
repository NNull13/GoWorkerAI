package runtime

import (
	"context"
	"fmt"
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
	mu         sync.RWMutex
	team       *teams.Team
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
	teamOptions := strings.Join(r.team.GetMembersOptions(), "\n")
	r.mu.Unlock()

	if task == nil {
		log.Println("⚠️ Worker returned nil task.")
		return nil
	}

	log.Printf("▶️ Starting task: %s", task.Description)
	messages := models.CreateMessages(task.Description, leader.Prompt(models.PlanSystemPrompt+"\n"+teamOptions))

	planText, err := r.model.Think(ctx, messages, 0.25, -1)
	if err != nil {
		log.Printf("❌ Error generating plan: %v\n", err)
		return err
	}

	//planText := task.Description
	team.Audits.Printf("✅ Plan generated:\n%s\n", planText)

	var i int
	var summarizedRecords int
	var history []storage.Record
	for {
		i++
		var delegateAction *models.DelegateAction
		summary := strings.Join(team.Audits.GetLastLogs(10), "\n")
		prompt := leader.Prompt(summary + "\nLast actions logs:\n" + storage.RecordListToString(history, 10))
		delegateAction, err = r.model.Delegate(ctx, teamOptions, planText, prompt)
		if err != nil || delegateAction == nil {
			log.Printf("❌ Skipping step %d. Error delegating: %v", i, err)
			continue
		}
		if delegateAction.Worker == "none" && delegateAction.Task == "finish" {
			team.Audits.Printf("✅ Plan finished: %s", delegateAction.Context)
			break
		}

		worker := r.team.GetMember(delegateAction.Worker)
		if worker == nil {
			log.Printf("❌ Skipping step %v. Worker %s not found.", delegateAction, delegateAction.Worker)
			continue
		}

		team.Audits.Printf("✅ Task assigned: %v", delegateAction)
		prompt = worker.Prompt(delegateAction.Context)
		messages = models.CreateMessages(delegateAction.Task, prompt)
		_, err = r.model.Process(ctx, worker.Key, team.Audits.Logger, messages, worker.GetToolKit(), task.ID.String(), i)
		if err != nil {
			log.Printf("❌ Skipping step %d. Error processing: %v", i, err)
			continue
		}

		var finish bool
		var newSummary string
		var reason string
		history, _ = r.db.GetHistoryByTaskID(ctx, task.ID.String(), -1)
		history = history[summarizedRecords:]
		newSummary = storage.RecordListToString(history, 100)
		messages = models.CreateMessages(newSummary, leader.Prompt(models.SummarySystemPrompt))
		newSummary, err = r.model.Think(ctx, messages, 0.1, 1000)
		if err != nil {
			log.Printf("❌ Skipping step %d. Error summarizing: %v", i, err)
			continue
		}
		team.Audits.Print(newSummary)
		summary += "\n" + newSummary
		summarizedRecords += len(history)

		messages = models.CreateMessages(fmt.Sprintf("Task : %s\n Summary: %s", planText, summary),
			leader.Prompt(models.TaskDoneBoolPrompt))
		finish, reason, err = r.model.TrueOrFalse(ctx, messages)
		if finish {
			team.Audits.Printf("✅ Plan finished: %s", reason)
			break
		} else {
			team.Audits.Printf("❌Plan still not finished, reason: %s", reason)
		}

	}

	r.team.Close()

	return nil
}

func (r *Runtime) GetTaskStatus() string {
	return strings.Join(r.team.Audits.GetLastLogs(100), "\n")
}
