package runtime

import (
	"context"
	"time"

	"GoWorkerAI/app/storage"
	"GoWorkerAI/app/workers"
)

const (
	NewTask    = "new_task"
	CancelTask = "cancel_task"
)

type Event struct {
	Task        *workers.Task
	HandlerFunc func(r *Runtime, ev Event) string
}

func (r *Runtime) SaveEventOnHistory(ctx context.Context, content string) error {
	task := r.worker.GetTask()

	var taskID string
	if task != nil {
		taskID = task.ID.String()
	}
	return r.db.SaveHistory(ctx, storage.Record{
		TaskID:    taskID,
		StepID:    0,
		Role:      "event",
		Content:   content,
		CreatedAt: time.Now(),
	})
}

func (r *Runtime) handleEvent(ev Event) {
	msg := ev.HandlerFunc(r, ev)
	r.logger.Printf("🆕 New Event received: %s Task: %v\n", msg, ev.Task)
}

var EventsHandlerFuncDefault = map[string]func(r *Runtime, ev Event) string{
	NewTask: func(r *Runtime, ev Event) string {
		if ev.Task == nil {
			r.logger.Println("⚠️ NewTask called with nil task.")
			return NewTask
		}

		r.mu.Lock()
		worker := r.worker
		prevCancel := r.cancelFunc
		r.mu.Unlock()

		if prevCancel != nil {
			r.logger.Println("🛑 Canceling current task before starting a new one.")
			prevCancel()
		}

		ctx, cancel := context.WithCancel(context.Background())

		r.mu.Lock()
		r.cancelFunc = cancel
		if worker != nil {
			worker.SetTask(ev.Task)
		} else {
			r.logger.Println("⚠️ No worker configured; task will not run.")
		}
		r.activeTask.Store(true)
		r.mu.Unlock()

		go func() {
			if err := r.runTask(ctx); err != nil {
				r.logger.Printf("Error running task: %v", err)
			}
		}()

		return NewTask
	},

	CancelTask: func(r *Runtime, ev Event) string {
		if !r.activeTask.CompareAndSwap(true, false) {
			r.logger.Println("⚠️ No active task to cancel.")
			return CancelTask
		}

		r.mu.Lock()
		r.StopRuntime()
		r.mu.Unlock()

		r.logger.Println("🛑 Canceling active task.")
		return CancelTask
	},
}
