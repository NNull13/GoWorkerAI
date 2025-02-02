package workers

import (
	"GoEngineerAI/app/models"
)

type Interface interface {
	Base
	PromptPlan() []models.Message
	PromptNextAction(plan string, executedActions []models.Action) []models.Message
	PromptValidation(plan string, actions []models.Action) []models.Message
}

type Base interface {
	GetTask() string
	GetFolder() string
	GetMaxIterations() int
	GetLockFolder() bool
}
