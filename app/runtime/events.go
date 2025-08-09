package runtime

import (
	"context"
	"log"
	"time"

	"GoWorkerAI/app/storage"
	"GoWorkerAI/app/workers"
)

const (
	NewTask    = "new_task"
	CancelTask = "cancel_task"
)

type Event struct {
	Context     string
	Task        *workers.Task
	HandlerFunc func(r *Runtime, ev Event) string
}

func (r *Runtime) SaveEventOnHistory(ctx context.Context, content string) error {
	task := r.worker.GetTask()
	if task == nil {
		return nil
	}
	return r.db.SaveHistory(ctx, storage.Record{
		TaskID:    task.ID.String(),
		StepID:    0,
		Role:      "event",
		Content:   content,
		CreatedAt: time.Now(),
	})
}

func (r *Runtime) handleEvent(ev Event) {
	msg := ev.HandlerFunc(r, ev)
	log.Printf("🆕 New Event received: %s Task: %v\n", msg, ev.Task)
}

var EventsHandlerFuncDefault = map[string]func(r *Runtime, ev Event) string{
	NewTask: func(r *Runtime, ev Event) string {
		r.mu.Lock()
		r.worker.SetTask(ev.Task)
		if r.cancelFunc != nil {
			log.Println("🛑 Canceling current task before starting a new one.")
			r.cancelFunc()
		}
		ctx, cancel := context.WithCancel(context.Background())
		r.cancelFunc = cancel
		r.activeTask = true
		r.mu.Unlock()
		go r.runTask(ctx)
		return NewTask
	},
	CancelTask: func(r *Runtime, ev Event) string {
		r.mu.Lock()
		if r.activeTask {
			log.Println("🛑 Canceling active task.")
			r.activeTask = false
			r.worker.SetTask(nil)
			if r.cancelFunc != nil {
				r.cancelFunc()
			}
		} else {
			log.Println("⚠️ No active task to cancel.")
		}
		r.mu.Unlock()
		return CancelTask
	},
}
