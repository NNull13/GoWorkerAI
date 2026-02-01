package tools

import (
	"errors"
	"log"

	"GoWorkerAI/app/utils"
)

// Presets
const (
	PresetDelegate = "delegate"
	PresetApprover = "approver"
	PresetAll      = "all"
)

// Tools
const (
	delegate_task = "delegate_task"
	report_issue  = "report_issue"
	true_or_false = "true_or_false"
)

type Tool struct {
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	Parameters  Parameter                      `json:"parameters"`
	HandlerFunc func(ToolTask) (string, error) `json:"-"`
}

type Parameter struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
	Required   []string       `json:"required"`
}

type ToolTask struct {
	Key        string         `json:"key"`
	Parameters map[string]any `json:"parameters"`
}

var allTools = map[string]Tool{
	report_issue: {
		Name:        report_issue,
		Description: "Report a problem, limitation, or need for assistance to the leader.",
		Parameters: Parameter{
			Type: "object",
			Properties: map[string]any{
				"reason": map[string]any{"type": "string"},
			},
			Required: []string{"reason"},
		},
	},
	delegate_task: {
		Name:        delegate_task,
		Description: "Assign a clear, atomic task required to complete the main task to a worker from the team.",
		Parameters: Parameter{
			Type: "object",
			Properties: map[string]any{
				"worker": map[string]any{
					"type":        "string",
					"description": "The worker to delegate the task to.",
				},
				"task": map[string]any{
					"type":        "string",
					"description": "A single, focused goal describing the exact action or deliverable expected.",
					"maxLength":   100,
				},
				"context": map[string]any{
					"type":        "string",
					"description": "Optional brief context or background needed for the worker to execute the task effectively (avoid redundancy).",
					"maxLength":   500,
				},
			},
			Required: []string{"worker_id", "objective"},
		},
	},
	true_or_false: {
		Name:        true_or_false,
		Description: "Binary decision with a brief reason.",
		Parameters: Parameter{
			Type: "object",
			Properties: map[string]any{
				"answer": map[string]any{
					"type": "string",
					"enum": []string{"true", "false"},
				},
				"reason": map[string]any{
					"type":      "string",
					"maxLength": 333,
				},
			},
			Required: []string{"answer", "reason"},
		},
		HandlerFunc: executeReviewerAction,
	},
}

func NewToolkitFromPreset(preset string) map[string]Tool {
	switch preset {
	case PresetDelegate:
		return pick(
			delegate_task,
		)
	case PresetApprover:
		return pick(
			true_or_false,
		)
	case PresetAll:
		keys := make([]string, 0, len(allTools))
		for k := range allTools {
			keys = append(keys, k)
		}
		return pick(keys...)
	default:
		return make(map[string]Tool)
	}
}

func pick(names ...string) map[string]Tool {
	m := make(map[string]Tool, len(names))
	for _, n := range names {
		if t, ok := allTools[n]; ok {
			m[n] = t
		}
	}
	return m
}

func withParsed[T any](params any, op string, f func(T) (string, error)) (string, error) {
	v, err := utils.CastAny[T](params)
	if err != nil {
		log.Printf("❌ Error parsing %s action: %v\n", op, err)
		return "", err
	}
	if v == nil {
		log.Printf("❌ %s action is nil\n", op)
		return "", errors.New("action is nil")
	}
	return f(*v)
}
