package models

import "context"

type Interface interface {
	Think(context.Context, []Message) (string, error)
	Process(context.Context, []Message) (*ActionTask, error)
	YesOrNo(context.Context, []Message) (bool, error)
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
