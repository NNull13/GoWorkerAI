package runtime

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"GoWorkerAI/app/models"
	"GoWorkerAI/app/storage"
	"GoWorkerAI/app/tools"
	"GoWorkerAI/app/workers"
)

type Runtime struct {
	mu         sync.Mutex
	worker     workers.Interface
	model      models.Interface
	toolkit    map[string]tools.Tool
	events     chan Event
	activeTask bool
	cancelFunc context.CancelFunc
	db         storage.Interface
}

func NewRuntime(worker workers.Interface, model models.Interface, initialActions map[string]tools.Tool, db storage.Interface, activeTask bool) *Runtime {
	return &Runtime{
		worker:     worker,
		model:      model,
		events:     make(chan Event, 100),
		toolkit:    initialActions,
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

func (r *Runtime) AddTools(newActions []tools.Tool) {
	r.mu.Lock()
	for _, newAction := range newActions {
		r.toolkit[newAction.Name] = newAction
	}
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
	currentWorker, plan := r.initializeTask(ctx)
	if currentWorker == nil {
		return
	}

	task := currentWorker.GetTask()
	if task == nil {
		return
	}

	maxIterations := task.MaxIterations

	var resume string
	for i := 0; i <= maxIterations; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			prompt := currentWorker.PromptNextAction(plan, resume)
			response, err := r.model.Process(ctx, prompt, r.toolkit, task.ID.String(), maxIterations)
			if err != nil {
				log.Printf("❌ Error executing action: %v", err)
				break
			}
			log.Printf("✅ Process response: %s", response)

			var validationResult bool
			if resume, validationResult = r.validateAction(ctx, currentWorker, plan, resume); validationResult {
				log.Printf("🎉 Task successfully completed: %s", task.Task)
				return
			}
		}
	}
	log.Printf("🚧 Maximum iterations (%d) reached for task: %s", maxIterations, task.Task)
}

func (r *Runtime) initializeTask(ctx context.Context) (workers.Interface, string) {
	r.mu.Lock()
	currentWorker := r.worker
	r.activeTask = false
	r.mu.Unlock()

	if currentWorker == nil {
		return nil, ""
	}

	plan, err := r.model.Think(ctx, currentWorker.PromptPlan(), 0.66, -1)
	if err != nil {
		return nil, ""
	}

	return currentWorker, plan
}

func (r *Runtime) validateAction(ctx context.Context, worker workers.Interface, plan, task string) (string, bool) {
	resume, err := r.model.GenerateSummary(ctx, task)
	if err != nil {
		return "", false
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
