package workers

import (
	"GoWorkerAI/app/actions"
	"GoWorkerAI/app/models"
)

type Interface interface {
	Base
	PromptPlan() []models.Message
	PromptNextAction(plan string, actions []actions.Action, executedActions []models.ActionTask) []models.Message
	PromptValidation(plan string, actions []models.ActionTask) []models.Message
}

type Base interface {
	SetTask(*Task)
	GetTask() *Task
	GetFolder() string
	GetLockFolder() bool
}
