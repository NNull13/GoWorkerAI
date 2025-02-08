package workers

import (
	"github.com/google/uuid"

	"GoWorkerAI/app/models"
)

type Interface interface {
	Base
	PromptPlan() []models.Message
	PromptNextAction(plan, resume string) []models.Message
	PromptValidation(plan, resume string) []models.Message
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

type Worker struct {
	Task       *Task
	Rules      []string
	LockFolder bool
	Folder     string
}

func (w *Worker) SetTask(task *Task) {
	task.ID = uuid.New()
	w.Task = task
}

func (w *Worker) GetTask() (task *Task) {
	if w != nil {
		task = w.Task
	}
	return
}

func (w *Worker) GetFolder() (folder string) {
	if w != nil {
		folder = w.Folder
	}
	return
}
func (w *Worker) GetLockFolder() bool {
	return w != nil && w.LockFolder
}
