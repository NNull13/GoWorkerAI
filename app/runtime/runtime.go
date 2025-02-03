package runtime

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"GoWorkerAI/app/actions"
	"GoWorkerAI/app/models"
	"GoWorkerAI/app/workers"
)

type Runtime struct {
	mu          sync.Mutex
	worker      workers.Interface
	model       models.Interface
	pastActions []models.ActionTask
	actions     []actions.Action
	events      chan Event
	activeTask  bool
	cancelFunc  context.CancelFunc
}

func NewRuntime(worker workers.Interface, model models.Interface, initialActions []actions.Action, activeTask bool) *Runtime {
	return &Runtime{
		worker:      worker,
		model:       model,
		events:      make(chan Event, 100),
		pastActions: []models.ActionTask{},
		actions:     initialActions,
		activeTask:  activeTask,
	}
}

func (r *Runtime) AddActions(newActions []actions.Action) {
	r.mu.Lock()
	r.actions = append(r.actions, newActions...)
	r.mu.Unlock()
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

func (r *Runtime) QueueEvent(event Event) {
	select {
	case r.events <- event:
	default:
		log.Print("⚠️ Event queue is full, dropping event")
	}
}

func (r *Runtime) handleEvent(ev Event) {
	r.mu.Lock()
	log.Printf("🆕 New Event received: %s Task: %v\n", ev.HandlerFunc(r, ev), ev.Task)
	defer r.mu.Unlock()
}

func (r *Runtime) runTask(ctx context.Context) {
	r.mu.Lock()
	currentWorker := r.worker
	r.mu.Unlock()

	if currentWorker == nil {
		log.Println("❌ Error: Worker is not initialized")
		return
	}

	log.Println("✅ Worker initialized. Generating plan...")

	defer func() {
		r.mu.Lock()
		r.activeTask = false
		r.pastActions = []models.ActionTask{}
		r.mu.Unlock()
		log.Println("🛑 runTask() completed.")
	}()

	plan, err := r.model.Think(ctx, currentWorker.PromptPlan())
	if err != nil {
		log.Printf("❌ Error generating plan: %v", err)
		return
	}
	log.Printf("✅ Plan generated: %s", plan)

	folderPath := r.prepareFolders(currentWorker)
	log.Println("📂 Working directory:", folderPath)

	var task *workers.Task
	if task = currentWorker.GetTask(); task == nil {
		return
	}

	maxIterations := task.MaxIterations
	for i := 0; i <= maxIterations; i++ {
		select {
		case <-ctx.Done(): // Check if the task was canceled
			log.Println("🛑 Task was canceled mid-execution.")
			return
		default:
			pastActionsSnapshot := r.getPastActionsSnapshot()
			log.Printf("📝 Iteration %d. Generating next action...", i)

			prompt := currentWorker.PromptNextAction(plan, r.actions, pastActionsSnapshot)
			actionTask, err := r.model.Process(ctx, prompt)
			if err != nil {
				log.Printf("❌ Error processing action: %v", err)
				break
			}

			log.Println("🚀 Executing action...")
			actionTask.Result, err = actions.ExecuteFileAction(actionTask, folderPath)
			if err != nil {
				log.Printf("❌ Error executing action: %v", err)
				break
			}
			log.Printf("✅ Action executed: %+v", actionTask)

			r.mu.Lock()
			r.pastActions = append(r.pastActions, *actionTask)
			r.mu.Unlock()

			log.Println("🔍 Validating action...")
			validationResult, err := r.model.YesOrNo(ctx, currentWorker.PromptValidation(plan, pastActionsSnapshot))
			if err != nil {
				log.Printf("❌ Validation error: %v", err)
				break
			}

			log.Printf("✅ Validation Result: %v", validationResult)
			AppendActionLog(filepath.Join("logs", time.Now().Format("20060102")+".log"), task.Task, actionTask, validationResult)

			if validationResult {
				log.Printf("🎉 Task successfully completed: %s", task.Task)
				return
			}
		}
	}

	log.Printf("🚧 Maximum iterations (%d) reached for task: %s", maxIterations, task.Task)
}

func (r *Runtime) getPastActionsSnapshot() []models.ActionTask {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]models.ActionTask{}, r.pastActions...)
}

func (r *Runtime) prepareFolders(w workers.Interface) string {
	folder := w.GetFolder()
	if w.GetLockFolder() {
		folder = filepath.Join("generations", folder+time.Now().Format("20060102_150405"))
		if err := os.MkdirAll(folder, os.ModePerm); err != nil {
			log.Printf("Error creating generation directory %s: %v", folder, err)
		}
	}
	if err := os.MkdirAll("logs", os.ModePerm); err != nil {
		log.Printf("Error creating logs directory: %v", err)
	}
	return folder
}
