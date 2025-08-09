package storage

import (
	"context"
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
