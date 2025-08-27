package workers

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"GoWorkerAI/app/models"
)

type Coder struct {
	Worker
	Language string
	Tests    bool
}

type coderInfo struct {
	workerInfo
	Language   string   `json:"language,omitempty"`
	CodeStyles []string `json:"code_styles,omitempty"`
	Tests      bool     `json:"tests,omitempty"`
}

func NewCoder(language, task, toolPreset string, rules []string, maxIterations int) *Coder {
	return &Coder{
		Worker: Worker{
			Task: &Task{
				ID:            uuid.New(),
				Task:          task,
				MaxIterations: maxIterations,
			},
			ToolsPreset: toolPreset,
			Rules:       rules,
		},
		Language: language,
	}
}

func (c *Coder) buildCoderInfo() coderInfo {
	if c == nil {
		return coderInfo{}
	}
	base := c.buildWorkerInfo()
	return coderInfo{
		workerInfo: base,
		Language:   c.Language,
		Tests:      c.Tests,
	}
}

func (c *Coder) TaskInformation() string {
	taskInformation, _ := json.Marshal(c.buildCoderInfo())
	return string(taskInformation)
}

func (c *Coder) PromptPlan(taskInformation string) []models.Message {
	sys := planSystemPrompt(c.GetPreamble())
	return []models.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: strings.TrimSpace(taskInformation)},
	}
}

func (c *Coder) GetPreamble() []string {
	base := []string{
		"You are a senior software engineer and strategic planner.",
		"Write correct, minimal, maintainable code that compiles and runs.",
		"Prefer small, idempotent changes with clear file operations.",
		"Deterministic output; no filler, no extra commentary.",
		"When unsure, choose the safest, reversible change.",
		"Keep diffs minimal; do not rewrite unrelated code.",
	}
	return append(base, c.Rules...)
}
