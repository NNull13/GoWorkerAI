package runtime

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"GoWorkerAI/app/actions"
	"GoWorkerAI/app/models"
	"GoWorkerAI/app/storage"
	"GoWorkerAI/app/workers"
)

type Runtime struct {
	mu         sync.Mutex
	worker     workers.Interface
	model      models.Interface
	actions    []actions.Action
	events     chan Event
	activeTask bool
	cancelFunc context.CancelFunc
	db         storage.Interface
}

func NewRuntime(worker workers.Interface, model models.Interface, initialActions []actions.Action, db storage.Interface, activeTask bool) *Runtime {
	return &Runtime{
		worker:     worker,
		model:      model,
		events:     make(chan Event, 100),
		actions:    initialActions,
		activeTask: activeTask,
		db:         db,
	}
}

func (r *Runtime) Start() {
	r.mu.Lock()
	if r.activeTask {
		ctx, cancel := context.WithCancel(context.Background())
		r.cancelFunc = cancel
		go r.runTask(ctx)
	}
	r.mu.Unlock()

	for {
		select {
		case ev := <-r.events:
			r.handleEvent(ev)
		default:
			time.Sleep(1 * time.Second)
		}
	}
}

func (r *Runtime) AddActions(newActions []actions.Action) {
	r.mu.Lock()
	r.actions = append(r.actions, newActions...)
	r.mu.Unlock()
}

func (r *Runtime) QueueEvent(event Event) {
	select {
	case r.events <- event:
	default:
		log.Print("⚠️ Event queue is full, dropping event")
	}
}

func (r *Runtime) runTask(ctx context.Context) {
	currentWorker, folderPath, plan := r.initializeTask(ctx)
	if currentWorker == nil {
		return
	}

	task := currentWorker.GetTask()
	if task == nil {
		return
	}

	r.executeIterations(ctx, currentWorker, folderPath, plan, task)
}

func (r *Runtime) initializeTask(ctx context.Context) (workers.Interface, string, string) {
	r.mu.Lock()
	currentWorker := r.worker
	r.activeTask = false
	r.mu.Unlock()

	if currentWorker == nil {
		return nil, "", ""
	}

	plan, err := r.model.GenerateResponse(ctx, currentWorker.PromptPlan(), 0.66, -1)
	if err != nil {
		return nil, "", ""
	}

	folderPath := r.prepareFolders(currentWorker)
	return currentWorker, folderPath, plan
}

func (r *Runtime) executeIterations(ctx context.Context, worker workers.Interface, folderPath, plan string, task *workers.Task) {
	taskID := task.ID.String()
	maxIterations := task.MaxIterations

	resume := "Not started the task yet"
	for i := 0; i <= maxIterations; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			actionTask, err := r.processAction(ctx, worker, plan, folderPath, resume)
			if err != nil {
				log.Printf("❌ Error executing action: %v", err)
				break
			}
			log.Printf("✅ Action executed: %+v", actionTask)

			r.mu.Lock()
			err = r.db.SaveRecord(storage.Record{
				TaskID:    taskID,
				Action:    actionTask.Action,
				Filename:  actionTask.Filename,
				Response:  actionTask.Result,
				Iteration: i,
			})
			r.mu.Unlock()

			if err != nil {
				break
			}

			var validationResult bool
			if resume, validationResult = r.validateAction(ctx, worker, plan, task.Task); validationResult {
				log.Printf("🎉 Task successfully completed: %s", task.Task)
				return
			}
		}
	}
	log.Printf("🚧 Maximum iterations (%d) reached for task: %s", maxIterations, task.Task)
}

func (r *Runtime) processAction(ctx context.Context, worker workers.Interface, plan, folderPath, resume string) (*models.ActionTask, error) {
	prompt := worker.PromptNextAction(plan, resume, r.actions)

	actionTask, err := r.model.Process(ctx, prompt)
	if err != nil {
		return nil, err
	}

	actionTask.Result, err = actions.ExecuteFileAction(actionTask, folderPath)
	if err != nil {
		return nil, err
	}

	return actionTask, nil
}

func (r *Runtime) validateAction(ctx context.Context, worker workers.Interface, plan, task string) (string, bool) {
	records, err := r.db.GetRecords(task)
	if err != nil {
		return "", false
	}

	var recordHistory strings.Builder
	for j, record := range records {
		recordHistory.WriteString(fmt.Sprintf("- Record: `%d` Info: %v\n", j, record))
	}

	resume, err := r.model.GenerateResponse(ctx, []models.Message{}, 0.1, 500)
	if err != nil {
		return resume, false
	}

	log.Printf("✅ Resume generated: %s", resume)

	validationResult, err := r.model.YesOrNo(ctx, worker.PromptValidation(plan, resume))
	if err != nil {
		return resume, false
	}

	return resume, validationResult
}

func (r *Runtime) prepareFolders(w workers.Interface) string {
	folder := w.GetFolder()
	if w.GetLockFolder() {
		folder = filepath.Join("generations", folder+time.Now().Format("20060102_150405"))
		os.MkdirAll(folder, os.ModePerm)
	}
	os.MkdirAll("logs", os.ModePerm)
	return folder
}

func (r *Runtime) handleEvent(ev Event) {
	r.mu.Lock()
	log.Printf("🆕 New Event received: %s Task: %v\n", ev.HandlerFunc(r, ev), ev.Task)
	r.mu.Unlock()
}
