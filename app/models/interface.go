package models

import "context"

type Interface interface {
	GenerateResponse(ctx context.Context, messages []Message, temperature float64, maxTokens int) (string, error)
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
