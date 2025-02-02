package workers

type Task struct {
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
