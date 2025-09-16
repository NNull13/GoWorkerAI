package workers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"GoWorkerAI/app/models"
)

const maxSteps = 20

type Interface interface {
	Base
	PromptPlan(taskInformation string) []models.Message
	PromptNextAction(plan, resume string) []models.Message
	PromptValidation(plan, summary string) []models.Message
	PromptSegmentedStep(step, summary, preamble string) []models.Message
	TaskInformation() string
}

type Base interface {
	SetTask(*Task)
	GetTask() *Task
	GetToolsPreset() string
	GetPreamble() string
}

type Task struct {
	ID            uuid.UUID
	Task          string
	MaxIterations int
}

type Worker struct {
	Task        *Task
	ToolsPreset string
	Rules       []string
}

type workerInfo struct {
	ID            string   `json:"id,omitempty"`
	MainTask      string   `json:"main_task,omitempty"`
	MaxIterations int      `json:"max_iterations,omitempty"`
	ToolsPreset   string   `json:"tools_preset,omitempty"`
	Rules         []string `json:"rules,omitempty"`
	Folder        string   `json:"folder,omitempty"`
	LockFolder    bool     `json:"lock_folder,omitempty"`
}

func NewWorker(task, toolPreset string, rules []string, maxIterations int) *Worker {
	return &Worker{
		Task: &Task{
			ID:            uuid.New(),
			Task:          task,
			MaxIterations: maxIterations,
		},
		ToolsPreset: toolPreset,
		Rules:       rules,
	}
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
	}
	if w.Task != nil {
		info.ID = w.Task.ID.String()
		info.MainTask = w.Task.Task
		info.MaxIterations = w.Task.MaxIterations
	}
	return info
}

func (w *Worker) TaskInformation() string {
	taskInformation, _ := json.Marshal(w.buildWorkerInfo())
	return string(taskInformation)
}

func planSystemPrompt(preamble string) string {
	core := []string{
		"You are an expert strategic planner. Create a precise, step-by-step plan for the task described below.",
		"Objectives:",
		"- Deterministic output; no filler, no extra commentary.",
		"- Follow the specified language and style when applicable.",
		"- If information is missing, include explicit clarification or TODO steps.",
		"CONTENT RULES:",
		"- Steps must be actionable, testable, and as small as reasonably possible.",
		"- If inputs are missing, include `[Request X from stakeholder]` or equivalent.",
		"HARD OUTPUT FORMAT:",
		"- Output ONLY a numbered list of steps.",
		"- Format exactly as:",
		"1. [First step]",
		"2. [Next step]",
		"...",
		"N. [Final step]",
		"- Start at 1 and increment by 1.",
		"- Exactly one line per step. No text before, between, or after steps.",
		fmt.Sprintf("- Max %d steps. No sub-steps, no explanations.", maxSteps),
	}

	return preamble + "\n" + strings.Join(core, "\n")
}

func (w *Worker) PromptNextAction(plan, resume string) []models.Message {
	sys := strings.Join([]string{
		"You are an AI worker executing a numbered plan. Decide the single next immediate action.",
		"RULES:",
		"- Identify the lowest-numbered step not fully completed given the execution history.",
		"- If that step is in progress, output the minimal next sub-action to advance it.",
		"- If all steps are completed, output `DONE`.",
		"- If the next action cannot proceed due to a concrete blocker, output `BLOCKED:<reason>`.",
		"OUTPUT:",
		"- If an action is possible, Return single atomic next action ",
		"- No lists, no explanations, no code fences, no surrounding text.",
	}, "\n")

	user := fmt.Sprintf(
		"Plan (numbered list):\n%s\n\nExecution History:\n%s\n\nReturn only one line as specified.",
		strings.TrimSpace(plan),
		strings.TrimSpace(resume),
	)

	return []models.Message{
		{Role: models.SystemRole, Content: sys},
		{Role: models.UserRole, Content: user},
	}
}

func (w *Worker) PromptValidation(plan, summary string) []models.Message {
	sys := strings.Join([]string{
		"You are a meticulous yet practical validator.",
		"Goal: Decide if the current task/step is sufficiently complete and correct based on the execution summary and the plan.",
		"CRITERIA (all must be satisfied or safely assumed without touching critical outputs):",
		"- The relevant plan step(s) are addressed with explicit or strongly implied results.",
		"- Logical order respected or equivalently justified.",
		"- No critical TODOs, UNKNOWNs, blockers, or unresolved tasks.",
		"- Key outputs/artifacts exist and are verifiable.",
		"- No contradictions.",
		"ASSUMPTIONS:",
		"- Make minor, reasonable assumptions only when strongly implied by the summary.",
		"- If an assumption touches correctness of critical outputs, prefer `false`.",
		"DECISION POLICY:",
		"- Any critical gap, ambiguity, or error → false.",
		"- All critical criteria clearly met; only non-critical details missing → true.",
		"OUTPUT:",
		"- Return exactly `true` or `false` in lowercase, with no punctuation or extra text.",
	}, "\n")

	user := fmt.Sprintf(
		"Plan:\n%s\n\nExecution Summary:\n%s\n\nReturn only `true` or `false`.",
		strings.TrimSpace(plan),
		strings.TrimSpace(summary),
	)

	return []models.Message{
		{Role: models.SystemRole, Content: sys},
		{Role: models.UserRole, Content: user},
	}
}

func (w *Worker) PromptSegmentedStep(step, summary, preamble string) []models.Message {
	var sb strings.Builder
	sb.WriteString(preamble)
	sb.WriteString("FOCUS TASK:\n")
	sb.WriteString(step + "\n")
	sb.WriteString("RULE: Must use only the tools provided for the task")

	if summary == "" {
		sb.WriteString("\nCURRENT EXECUTION SUMMARY: not started\n")
	} else {
		sb.WriteString("\nCURRENT EXECUTION SUMMARY:\n")
		sb.WriteString(summary)
		sb.WriteString("\n")
	}

	systemMsg := models.Message{Role: models.SystemRole, Content: sb.String()}
	userMsg := models.Message{Role: models.UserRole, Content: step}
	return []models.Message{systemMsg, userMsg}
}

func (w *Worker) PromptPlan(taskInformation string) []models.Message {
	sys := planSystemPrompt(w.GetPreamble())
	return []models.Message{
		{Role: models.SystemRole, Content: sys},
		{Role: models.UserRole, Content: strings.TrimSpace(taskInformation)},
	}
}

func (w *Worker) GetPreamble() string {
	base := []string{
		"You are the best assistant for any task. All tasks are important.",
		"Complete tasks quickly, without errors, and without breaking rules.",
		"Prefer clear, verifiable, reversible steps.",
	}
	return strings.Join(append(base, w.Rules...), "\n")
}
