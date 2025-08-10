package workers

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"GoWorkerAI/app/models"
)

type Coder struct {
	Worker
	Language   string
	CodeStyles []string
	Tests      bool
}

type coderInfo struct {
	workerInfo
	Language   string   `json:"language,omitempty"`
	CodeStyles []string `json:"code_styles,omitempty"`
	Tests      bool     `json:"tests,omitempty"`
}

func NewCoder(
	language, task, toolPreset string,
	codeStyles, acceptConditions, rules []string,
	maxIterations int,
	folder string,
	tests, lockFolder bool,
) *Coder {
	return &Coder{
		Worker: Worker{
			Task: &Task{
				ID:               uuid.New(),
				Task:             task,
				AcceptConditions: acceptConditions,
				MaxIterations:    maxIterations,
			},
			ToolsPreset: toolPreset,
			Rules:       rules,
			LockFolder:  lockFolder,
			Folder:      folder,
		},
		Language:   language,
		CodeStyles: codeStyles,
		Tests:      tests,
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
		CodeStyles: append([]string(nil), c.CodeStyles...),
		Tests:      c.Tests,
	}
}

func (c *Coder) TaskInformation() string {
	taskInformation, _ := json.Marshal(c.buildCoderInfo())
	return string(taskInformation)
}

// ------- Shared coder preamble (consistent behavior across prompts)

func (c *Coder) GetPreamble() string {
	var sb strings.Builder
	sb.WriteString(strings.Join([]string{
		"You are an expert software engineer and careful editor.",
		"Goals:",
		"- Write correct, minimal, maintainable code that compiles and runs.",
		"- Prefer small, idempotent changes with clear file operations.",
		"- Follow the specified language and code styles.",
		"- If information is missing, add explicit TODOs or clarification steps (but keep outputs in the required format).",
		"General Rules:",
		"- Deterministic output. No filler text, no extra commentary.",
		"- If you are unsure, choose the safest, reversible change.",
		"- Keep diffs minimal; do not rewrite unrelated code.",
	}, "\n"))
	return sb.String()
}

func (c *Coder) PromptPlan(taskInformation string) []models.Message {
	sys := planSystemPrompt(c.GetPreamble())
	user := strings.Join([]string{
		taskInformation,
		"Generate a precise, step-by-step technical development plan, strictly following the format rules.",
	}, "\n\n")
	return []models.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: user},
	}
}
