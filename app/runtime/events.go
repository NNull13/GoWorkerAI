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
	Type        string
	Task        *workers.Task
	HandlerFunc func(r *Runtime, ev Event)
}

var EventsHandlerFuncDefault = map[string]func(r *Runtime, ev Event){
	NewTask: func(r *Runtime, ev Event) {
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
	},
	CancelTask: func(r *Runtime, ev Event) {
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
	},
}
