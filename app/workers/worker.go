package workers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"GoWorkerAI/app/models"
)

type Interface interface {
	Base
	PromptPlan(taskInformation string) []models.Message
	PromptNextAction(plan, resume string) []models.Message
	PromptValidation(plan, summary string) []models.Message
	PromptSegmentedStep(steps []string, index int, summary, preamble string) []models.Message
	TaskInformation() string
}

type Base interface {
	SetTask(*Task)
	GetTask() *Task
	GetFolder() string
	GetLockFolder() bool
	GetToolsPreset() string
	GetPreamble() string
}

type Task struct {
	ID               uuid.UUID
	Task             string
	AcceptConditions []string
	MaxIterations    int
}

type Worker struct {
	Task        *Task
	ToolsPreset string
	Rules       []string
	LockFolder  bool
	Folder      string
}

type workerInfo struct {
	ID               string   `json:"id,omitempty"`
	MainTask         string   `json:"main_task,omitempty"`
	AcceptConditions []string `json:"accept_conditions,omitempty"`
	MaxIterations    int      `json:"max_iterations,omitempty"`
	ToolsPreset      string   `json:"tools_preset,omitempty"`
	Rules            []string `json:"rules,omitempty"`
	Folder           string   `json:"folder,omitempty"`
	LockFolder       bool     `json:"lock_folder,omitempty"`
}

func (w *Worker) SetTask(task *Task) {
	w.Task = task
	if w == nil {
		return
	}
	task.ID = uuid.New()
}

func (w *Worker) GetTask() *Task {
	if w == nil {
		return nil
	}
	return w.Task
}

func (w *Worker) GetFolder() string {
	if w == nil {
		return ""
	}
	return w.Folder
}

func (w *Worker) GetLockFolder() bool {
	if w == nil {
		return false
	}
	return w.LockFolder
}

func (w *Worker) GetToolsPreset() string {
	if w == nil {
		return ""
	}
	return w.ToolsPreset
}

func (w *Worker) buildWorkerInfo() workerInfo {
	if w == nil {
		return workerInfo{}
	}
	info := workerInfo{
		ToolsPreset: w.ToolsPreset,
		Rules:       append([]string(nil), w.Rules...),
		Folder:      w.Folder,
		LockFolder:  w.LockFolder,
	}
	if w.Task != nil {
		info.ID = w.Task.ID.String()
		info.MainTask = w.Task.Task
		info.AcceptConditions = append([]string(nil), w.Task.AcceptConditions...)
		info.MaxIterations = w.Task.MaxIterations
	}
	return info
}

func (w *Worker) GetPreamble() string {
	var sb strings.Builder
	sb.WriteString(strings.Join([]string{
		"You are the best assistant for any task. All tasks are important.",
		"Goals:",
		"- Complete the task as quickly as possible.",
		"- Complete the task without errors.",
		"- Complete the task without breaking any rules.",
	}, "\n"))
	return sb.String()
}

func (w *Worker) TaskInformation() string {
	taskInformation, _ := json.Marshal(w.buildWorkerInfo())
	return string(taskInformation)
}

func planSystemPrompt(preamble string) string {
	var rules []string
	rules = append(rules,
		"You are an expert strategic planner. Your role is to create a highly detailed, ",
		"step-by-step plan for the task described below. The plan MUST adhere to the format exactly as specified.",
		"HARD FORMAT RULES:",
		"- Output ONLY a numbered list of steps.",
		"- Required Output Format is:",
		"`1. [Description of the first step]\n2. [Description of the next step]\n...\nN. [Final step]\n`.",
		"- Start at 1 and increment by 1.",
		"- Exactly one line per step. No text before, between, or after steps.",
	)

	rules = append(rules, fmt.Sprintf("- Max %d steps. No sub-steps, no explanations.", 20))

	content := []string{
		"CONTENT RULES:",
		"- Steps must be actionable, verifiable, and as small as reasonably possible.",
		"- If information is missing, include explicit clarification steps (e.g., `[Request X from stakeholder]`).",
	}

	var parts []string
	if strings.TrimSpace(preamble) != "" {
		parts = append(parts, strings.TrimSpace(preamble))
	}
	parts = append(parts, strings.Join(rules, "\n"))
	parts = append(parts, strings.Join(content, "\n"))
	return strings.Join(parts, "\n")
}

func (w *Worker) PromptPlan(taskInformation string) []models.Message {
	sys := planSystemPrompt("") // no preamble
	user := strings.Join([]string{
		taskInformation,
		"Generate the plan, strictly following the format rules.",
	}, "\n\n")
	return []models.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: user},
	}
}

func (w *Worker) PromptNextAction(plan, resume string) []models.Message {
	sys := strings.Join([]string{
		"You are an AI worker executing a numbered plan. Given the plan and execution history, decide the single next immediate action.",
		"RULES:",
		"- Identify the lowest-numbered step that is not fully completed.",
		"- If that step is already in progress, return the minimal next sub-action to advance it.",
		"- No explanations or lists.",
	}, "\n")
	user := fmt.Sprintf(
		"Plan (numbered list):\n%s\n\nExecution History:\n%s\n\nReturn only the required output.",
		plan, resume,
	)
	return []models.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: user},
	}
}

func (w *Worker) PromptValidation(plan, summary string) []models.Message {
	sys := strings.Join([]string{
		"You are a meticulous yet practical validator.",
		"Goal: Decide if more work is REQUIRED to finish the task, based on the execution summary and the plan.",
		"Important:",
		"- Answer `true` if further work is needed (incomplete, unclear, or incorrect).",
		"- Answer `false` if the task is sufficiently complete and correct.",
		"Use LIMITED, CRITICAL JUDGMENT:",
		"- You may make minor, reasonable assumptions only when strongly implied by the summary.",
		"- Treat such links as ASSUMED and be conservative: if the assumption touches a critical output/correctness criterion, prefer `true`.",
		"- Do NOT invent artifacts, IDs, results, or success states that are not implied.",
		"Pass criteria (all must be satisfied or safely assumed without touching critical outputs):",
		"- Each plan step is addressed with an explicit or strongly implied result.",
		"- Logical order respected or equivalently justified.",
		"- No critical TODOs, UNKNOWNs, blockers, or unresolved retries.",
		"- Key outputs/artifacts exist with enough detail to be verifiable (IDs/URLs/files/status/counters).",
		"- No contradictions.",
		"Decision policy:",
		"- If any critical gap, ambiguity, or error remains → `true` (more work needed).",
		"- If all critical criteria are clearly met and only non-critical details are missing → `false`.",
		"- When uncertain about critical completeness → `true`.",
	}, "\n")

	user := fmt.Sprintf(
		"Plan:\n%s\n\nExecution Summary:\n%s\n\nHas the plan been fully and correctly executed?",
		plan, summary,
	)
	return []models.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: user},
	}
}

func (w *Worker) PromptSegmentedStep(steps []string, index int, summary, preamble string) []models.Message {
	if index < 0 || index >= len(steps) {
		return nil
	}

	stepText := steps[index]
	total := len(steps)

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("FOCUS TASK (%d/%d):\n", index, total))
	sb.WriteString(stepText + "\n")

	sb.WriteString(preamble + "\n")

	if summary == "" {
		sb.WriteString("\nCURRENT EXECUTION SUMMARY: not started\n")
	} else {
		sb.WriteString("\nCURRENT EXECUTION SUMMARY:\n")
		sb.WriteString(summary)
		sb.WriteString("\n")
	}

	systemMsg := models.Message{Role: "system", Content: sb.String()}
	userMsg := models.Message{Role: "user", Content: steps[index]}
	return []models.Message{systemMsg, userMsg}
}
