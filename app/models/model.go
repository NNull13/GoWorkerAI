package models

import (
	"context"

	"GoWorkerAI/app/storage"
	"GoWorkerAI/app/tools"
)

type Interface interface {
	Think(context.Context, []Message, float64, int) (string, error)
	Process(context.Context, []Message, map[string]tools.Tool, string, int) (string, error)
	YesOrNo(context.Context, []Message) (bool, error)
	GenerateSummary(context.Context, []storage.Record) (string, error)
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
}
