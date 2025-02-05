package models

import (
	"context"

	"GoWorkerAI/app/tools"
)

type Interface interface {
	Think(context.Context, []Message, float64, int) (string, error)
	Process(context.Context, []Message, map[string]tools.Tool, string, int) (string, error)
	YesOrNo(context.Context, []Message) (bool, error)
	GenerateSummary(context.Context, string) (string, error)
}

type Message struct {
	Role      string     `json:"role"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}
