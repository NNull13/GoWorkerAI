package storage

import (
	"context"
	"fmt"
	"time"
)

type Interface interface {
	SaveHistory(ctx context.Context, iteration Record) error
	GetHistoryByTaskID(ctx context.Context, taskID string, stepID int) ([]Record, error)
}

type Record struct {
	ID         int64     `json:"id" db:"id"`
	TaskID     string    `json:"task_id" db:"task_id"`
	StepID     int64     `json:"step_id" db:"step_id"`
	Role       string    `json:"role" db:"role"`
	Tool       string    `json:"tool" db:"tool"`
	Parameters string    `json:"parameters" db:"parameters"`
	Content    string    `json:"content" db:"content"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

func RecordListToString(records []Record, countSteps int) string {
	recordsSliced := records
	var historySummary string
	if len(records) > 0 {
		if len(records) > countSteps {
			recordsSliced = records[:countSteps]
		}
		for _, entry := range recordsSliced {
			if entry.Role == "tool" || entry.Role == "assistant" {
				historySummary += fmt.Sprintf("\nRole: %s | Content: %s | Tool: %s | Step: %d | ID: %d",
					entry.Role, entry.Content, entry.Tool, entry.StepID, entry.ID)
			}
		}
	}
	return historySummary
}
