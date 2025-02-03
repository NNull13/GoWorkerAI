package workers

import (
	"github.com/google/uuid"

	"GoWorkerAI/app/actions"
	"GoWorkerAI/app/models"
)

type Interface interface {
	Base
	PromptPlan() []models.Message
	PromptNextAction(plan, resume string, actions []actions.Action) []models.Message
	PromptValidation(plan string, recordsResume string) []models.Message
}

type Base interface {
	SetTask(*Task)
	GetTask() *Task
	GetFolder() string
	GetLockFolder() bool
}

type Task struct {
	ID               uuid.UUID
	Task             string
	AcceptConditions []string
	MaxIterations    int
}
