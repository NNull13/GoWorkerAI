package models

import (
	"context"
	"log"

	"GoWorkerAI/app/storage"
	"GoWorkerAI/app/tools"
)

const (
	SystemRole    = "system"
	UserRole      = "user"
	AssistantRole = "assistant"
	ToolRole      = "tool"
)

type Interface interface {
	Think(context.Context, []Message, float64, int) (string, error)
	Process(context.Context, *log.Logger, []Message, map[string]tools.Tool, string, int) (string, error)
	TrueOrFalse(context.Context, []Message) (bool, string, error)
	GenerateSummary(context.Context, string, []string, []storage.Record) (string, error)
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
}
