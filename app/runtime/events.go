package runtime

import (
	"context"
	"fmt"
	"log"
	"time"

	"GoWorkerAI/app/models"
	"GoWorkerAI/app/storage"
	"GoWorkerAI/app/teams"
)

const (
	noTaskID   = "no_task_id"
	maxHistory = 33
	maxTokens  = -1
)

type Event struct {
	Origin      string
	Task        *teams.Task
	HandlerFunc func(r *Runtime, ev Event) string
}

func (r *Runtime) SaveEventOnHistory(ctx context.Context, content, role string) error {
	task := r.team.Task

	var taskID string
	if task != nil {
		taskID = task.ID.String()
	}
	return r.db.SaveHistory(ctx, storage.Record{
		TaskID:    taskID,
		SubTaskID: 0,
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
	})
}

func (r *Runtime) handleEvent(ev Event) {
	msg := ev.HandlerFunc(r, ev)
	log.Printf("🆕 New Event received: %s Description: %v\n", msg, ev.Task)
}

func (r *Runtime) ProcessQuickEvent(ctx context.Context, message string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	history, _ := r.db.GetHistoryByTaskID(ctx, noTaskID, -1)
	var summary string
	if len(history) > 0 {
		if len(history) > maxHistory {
			history = history[len(history)-maxHistory:]
		}
		for _, v := range history {
			summary = summary + fmt.Sprintf("%s Role: %s Message: %s \n", v.CreatedAt, v.Role, v.Content)
		}
	}

	response, err := r.model.Think(ctx, r.team.GetEventHandler().Prompt(message, summary), 0.666, maxTokens)
	if err != nil {
		response = "Couldn't process your message. Something went wrong"
	}

	if err = r.db.SaveHistory(ctx, storage.Record{
		TaskID:    noTaskID,
		SubTaskID: 0,
		Role:      models.AssistantRole,
		Content:   response,
		CreatedAt: time.Now(),
	}); err != nil {
		log.Printf("⚠️ Error saving history for task %s: %v", noTaskID, err)
	}

	return response
}
