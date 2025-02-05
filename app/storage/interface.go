package storage

import (
	"context"
	"time"
)

type Interface interface {
	SaveIteration(ctx context.Context, iteration Iteration) error
	GetHistoryByTaskID(ctx context.Context, taskID string) ([]Iteration, error)
}

type Iteration struct {
	ID        int64
	TaskID    string
	Role      string
	Tool      string
	Content   string
	CreatedAt time.Time
}
