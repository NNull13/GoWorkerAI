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
	Process(context.Context, string, *log.Logger, []Message, map[string]tools.Tool, string, int) (string, error)
	Delegate(context.Context, string, string, string) (*DelegateAction, error)
	TrueOrFalse(context.Context, []Message) (bool, string, error)
	GenerateSummary(context.Context, string, []storage.Record) (string, error)
	EmbedText(context.Context, string) ([]float32, error)
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
}

type DelegateAction struct {
	Worker  string `json:"worker,validate:required"`
	Task    string `json:"task,validate:required"`
	Context string `json:"context"`
}

func CreateMessages(userPrompt, sysPrompt string) []Message {
	return []Message{
		{Role: SystemRole, Content: sysPrompt},
		{Role: UserRole, Content: userPrompt},
	}
}
