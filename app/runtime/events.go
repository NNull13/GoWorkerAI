package runtime

import (
	"context"
	"log"

	"GoWorkerAI/app/models"
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

var EventsHandlerFuncDefault = map[string]func(r *Runtime, ev Event) string{
	NewTask: func(r *Runtime, ev Event) string {
		r.worker.SetTask(ev.Task)
		if r.cancelFunc != nil {
			log.Println("🛑 Canceling current task before starting a new one.")
			r.cancelFunc()
		}
		ctx, cancel := context.WithCancel(context.Background())
		r.cancelFunc = cancel
		r.activeTask = true
		r.pastActions = []models.ActionTask{}
		go r.runTask(ctx)
		return NewTask
	},
	CancelTask: func(r *Runtime, ev Event) string {
		if r.activeTask {
			log.Println("🛑 Canceling active task.")
			r.activeTask = false
			r.worker.SetTask(nil)
			if r.cancelFunc != nil {
				r.cancelFunc() // Stops the current `runTask`
			}
			r.pastActions = []models.ActionTask{}
		} else {
			log.Println("⚠️ No active task to cancel.")
		}
		return CancelTask
	},
}
