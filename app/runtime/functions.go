package runtime

import (
	"context"
	"log"
)

const (
	NewTask    = "new_task"
	CancelTask = "cancel_task"
)

var EventsHandlerFuncDefault = map[string]func(r *Runtime, ev Event) string{
	NewTask: func(r *Runtime, ev Event) string {
		if ev.Task == nil {
			return "No new task detected to start."
		}

		r.mu.Lock()
		prevCancel := r.cancelFunc

		if prevCancel != nil {
			log.Println("🛑 Canceling current task before starting a new one.")
			prevCancel()
		}

		ctx, cancel := context.WithCancel(context.Background())

		r.team.Task = ev.Task
		r.activeTask.Store(true)
		r.mu.Unlock()

		go func() {
			if err := r.runTask(ctx, cancel); err != nil {
				log.Printf("Error running task: %v", err)
			}
		}()

		return NewTask
	},

	CancelTask: func(r *Runtime, ev Event) string {
		if !r.activeTask.CompareAndSwap(true, false) {
			log.Println("⚠️ No active task to cancel.")
			return CancelTask
		}

		r.StopRuntime()

		log.Println("🛑 Canceling active task.")
		return CancelTask
	},
}
