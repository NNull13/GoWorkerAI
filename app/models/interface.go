package models

type Interface interface {
	Think(messages []Message) (string, error)
	Process(messages []Message) (*ActionTask, error)
	YesOrNo(messages []Message, retry int) (bool, error)
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ActionTask struct {
	Action   string `json:"action"`
	Filename string `json:"filename,omitempty"`
	Content  string `json:"content,omitempty"`
	Result   string `json:"result,omitempty"`
}
