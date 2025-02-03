package storage

type Interface interface {
	SaveRecord(record Record) error
	GetRecords(taskID string) ([]Record, error)
}

type Record struct {
	TaskID    string
	Iteration int
	Action    string
	Filename  string
	Response  string
}
