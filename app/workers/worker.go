package workers

import "github.com/google/uuid"

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
