package workers

import (
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
	PromptSegmentedStep(steps []string, index int, summary string) []models.Message
	TaskInformation() string
}

type Base interface {
	SetTask(*Task)
	GetTask() *Task
	GetFolder() string
	GetLockFolder() bool
	GetToolsPreset() string
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

func (w *Worker) SetTask(task *Task) {
	task.ID = uuid.New()
	w.Task = task
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

func (w *Worker) TaskInformation() string {
	var sb strings.Builder
	sb.WriteString("Task Information:\n")
	sb.WriteString(fmt.Sprintf("Main Task: %s\n", w.Task.Task))
	if len(w.Task.AcceptConditions) > 0 {
		sb.WriteString(fmt.Sprintf("Accepted Conditions: %s\n", strings.Join(w.Task.AcceptConditions, ", ")))
	}
	return sb.String()
}

// planSystemPrompt builds the strict planning system prompt used by all workers.
func planSystemPrompt(preamble string) string {
	var rules []string
	rules = append(rules,
		"HARD FORMAT RULES:",
		"- Output ONLY a numbered list of steps.",
		"- Each step MUST be exactly: `N. [imperative verb, brief actionable description]`.",
		"- Start at 1 and increment by 1.",
		"- The description MUST be inside `[]` with no nested brackets.",
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
	sys := planSystemPrompt()
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
		"You are a meticulous validator.",
		"Decide if the plan has been executed COMPLETELY and CORRECTLY based on the execution summary.",
		"ALL criteria must be satisfied:",
		"- Every step in the plan is covered with no omissions.",
		"- Execution respects logical order or provides equivalent justified progression.",
		"- No pending tasks, TODOs, errors, blockers, or implied future work remain.",
		"- Key outputs/artifacts are indicated as completed.",
		"OUTPUT FORMAT: respond with EXACTLY `true` or `false` in lowercase. No extra text.",
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

func (w *Worker) PromptSegmentedStep(steps []string, index int, summary string) []models.Message {
	if index < 0 || index >= len(steps) {
		systemMsg := models.Message{
			Role:    "system",
			Content: "Step index out of range. Respond with `DONE`.",
		}
		userMsg := models.Message{Role: "user", Content: "DONE"}
		return []models.Message{systemMsg, userMsg}
	}

	var sb strings.Builder
	sb.WriteString("You are executing a segmented plan.\n")
	sb.WriteString("FULL PLAN (numbered list):\n")
	for i, step := range steps {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
	}
	sb.WriteString("\nFOCUS STEP:\n")
	sb.WriteString(fmt.Sprintf("%d. %s\n", index+1, steps[index]))

	if summary == "" {
		sb.WriteString("\nCurrent Execution Summary: (not available)\n")
	} else {
		sb.WriteString("\nCurrent Execution Summary:\n" + summary + "\n")
	}

	sb.WriteString(strings.Join([]string{
		"\nGOAL:",
		"- If the focus step is already complete per the summary: respond `DONE`.",
		"- Otherwise: return the single next minimal action to advance this step.",
		"REQUIRED OUTPUT (exactly one line):",
		"- `ACTION: <imperative command, <= 25 words>` or `DONE`.",
		"- No explanations, no additional lines.",
	}, "\n"))

	systemMsg := models.Message{Role: "system", Content: sb.String()}
	userMsg := models.Message{Role: "user", Content: "Return the output in the required format."}
	return []models.Message{systemMsg, userMsg}
}
