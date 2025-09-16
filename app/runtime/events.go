package runtime

import (
	"context"
	"fmt"
	"log"
	"time"

	"GoWorkerAI/app/models"
	"GoWorkerAI/app/storage"
	"GoWorkerAI/app/workers"
)

const (
	NewTask    = "new_task"
	CancelTask = "cancel_task"

	noTaskID   = "no_task_id"
	maxHistory = 33
	maxTokens  = -1
)

type Event struct {
	Task        *workers.Task
	HandlerFunc func(r *Runtime, ev Event) string
}

func (r *Runtime) SaveEventOnHistory(ctx context.Context, content, role string) error {
	task := r.worker.GetTask()

	var taskID string
	if task != nil {
		taskID = task.ID.String()
	}
	return r.db.SaveHistory(ctx, storage.Record{
		TaskID:    taskID,
		StepID:    0,
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
	})
}

func (r *Runtime) handleEvent(ev Event) {
	msg := ev.HandlerFunc(r, ev)
	r.audits.Printf("🆕 New Event received: %s Task: %v\n", msg, ev.Task)
}

var EventsHandlerFuncDefault = map[string]func(r *Runtime, ev Event) string{
	NewTask: func(r *Runtime, ev Event) string {
		if ev.Task == nil {
			r.audits.Println("⚠️ NewTask called with nil task.")
			return NewTask
		}

		r.mu.Lock()
		worker := r.worker
		prevCancel := r.cancelFunc

		if prevCancel != nil {
			r.audits.Println("🛑 Canceling current task before starting a new one.")
			prevCancel()
		}

		ctx, cancel := context.WithCancel(context.Background())

		if worker != nil {
			worker.SetTask(ev.Task)
		} else {
			r.audits.Println("⚠️ No worker configured; task will not run.")
		}
		r.activeTask.Store(true)
		r.mu.Unlock()

		go func() {
			if err := r.runTask(ctx, cancel); err != nil {
				r.audits.Printf("Error running task: %v", err)
			}
		}()

		return NewTask
	},

	CancelTask: func(r *Runtime, ev Event) string {
		if !r.activeTask.CompareAndSwap(true, false) {
			r.audits.Println("⚠️ No active task to cancel.")
			return CancelTask
		}

		r.StopRuntime()

		r.audits.Println("🛑 Canceling active task.")
		return CancelTask
	},
}

func (r *Runtime) ProcessQuickEvent(ctx context.Context, message string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	history, _ := r.db.GetHistoryByTaskID(ctx, noTaskID, -1)
	var messages []models.Message
	if len(history) > 0 {
		var summary string
		if len(history) > maxHistory {
			history = history[len(history)-maxHistory:]
		}
		for _, v := range history {
			summary = summary + fmt.Sprintf("%s Role: %s Message: %s \n", v.CreatedAt, v.Role, v.Content)
		}
		messages = append(messages, models.Message{
			Role:    models.SystemRole,
			Content: r.worker.GetPreamble() + "\nPrevious conversation:\n" + summary,
		})
	}

	messages = append(messages, models.Message{
		Role:    models.UserRole,
		Content: message,
	})

	response, err := r.model.Think(ctx, messages, 0.666, maxTokens)
	if err != nil {
		response = "Couldn't process your message. Something went wrong"
	}

	if err = r.db.SaveHistory(ctx, storage.Record{
		TaskID:    noTaskID,
		StepID:    0,
		Role:      models.AssistantRole,
		Content:   response,
		CreatedAt: time.Now(),
	}); err != nil {
		log.Printf("⚠️ Error saving history for task %s: %v", noTaskID, err)
	}

	return response
}
