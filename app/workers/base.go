package workers

type Worker struct {
	Task             string
	AcceptConditions []string
	Rules            []string
	MaxIterations    int
	LockFolder       bool
	Folder           string
	Actions          map[string]string
}

func (w *Worker) GetTask() (task string) {
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

func (w *Worker) GetMaxIterations() (maxIterations int) {
	if w != nil {
		maxIterations = w.MaxIterations
	}
	return
}
