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
}

type Task struct {
	ID               uuid.UUID
	Task             string
	AcceptConditions []string
	MaxIterations    int
}

type Worker struct {
	Task       *Task
	Rules      []string
	LockFolder bool
	Folder     string
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

func (w *Worker) TaskInformation() string {
	var sb strings.Builder
	sb.WriteString("Task Information:\n")
	sb.WriteString(fmt.Sprintf("Main Task: %s\n", w.Task.Task))
	if len(w.Task.AcceptConditions) > 0 {
		sb.WriteString(fmt.Sprintf("Accepted Conditions: %s\n", strings.Join(w.Task.AcceptConditions, ", ")))
	}
	return sb.String()
}

func (w *Worker) PromptPlan(taskInformation string) []models.Message {
	var sb strings.Builder
	sb.WriteString("You are an expert strategic planner. Your role is to create a highly detailed, step-by-step plan for the task described below. The plan MUST adhere to the format exactly as specified, with each step in a numbered list, and no additional text or explanations outside the steps.\n")
	sb.WriteString("\n### Required Output Format:\n")
	sb.WriteString("```\n1. [Description of the first step]\n2. [Description of the next step]\n...\nN. [Final step]\n```\n")
	sb.WriteString("⚠ **Important Guidelines:**\n")
	sb.WriteString("- Ensure each step description is enclosed in square brackets `[]`.\n")
	sb.WriteString("- Do not include any text outside the numbered steps.\n")
	sb.WriteString("- Avoid any introductory or concluding remarks, comments, or code.\n")
	sb.WriteString("- Maintain brevity and clarity in each step.\n")
	sb.WriteString("- Follow the exact numbering format (`1.`, `2.`, etc.).\n")
	sb.WriteString(taskInformation)
	systemMsg := models.Message{
		Role:    "system",
		Content: sb.String(),
	}
	userMsg := models.Message{
		Role:    "user",
		Content: "Please provide the plan strictly in the required format. Ensure full adherence to the guidelines.",
	}
	return []models.Message{systemMsg, userMsg}
}

func (w *Worker) PromptNextAction(plan, resume string) []models.Message {
	sys := "You are an AI Worker executing a task plan. Based on the plan and the current execution history, determine the next immediate action."
	user := fmt.Sprintf("Plan:\n%s\n\nExecution History:\n%s\n\nWhat is the next immediate action? Please respond exactly as required.", plan, resume)
	return []models.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: user},
	}
}

func (w *Worker) PromptValidation(plan, summary string) []models.Message {
	sys := "You are a meticulous task validator. Evaluate the following development plan against the execution summary. Determine if every step has been completely and correctly executed and if all guidelines have been followed. Your response MUST be exactly either 'true' or 'false' (in lowercase) with no additional text."
	user := fmt.Sprintf("Plan:\n%s\n\nExecution Summary:\n%s\n\nHas the task been fully and correctly executed? Respond only with 'true' or 'false'.", plan, summary)
	return []models.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: user},
	}
}

func (w *Worker) PromptSegmentedStep(steps []string, index int, summary string) []models.Message {
	var sb strings.Builder
	sb.WriteString("You are executing a segmented development plan. The full plan is provided below as a numbered list:\n")
	for i, step := range steps {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
	}
	sb.WriteString("\nCurrently, focus on step ")
	sb.WriteString(fmt.Sprintf("%d: %s\n", index+1, steps[index]))
	if summary == "" {
		sb.WriteString("\nNo execution summary is available at this time.\n")
	} else {
		sb.WriteString("\nCurrent Execution Summary:\n" + summary + "\n")
	}
	sb.WriteString("\nYour response MUST provide the next immediate action for this step\n")
	sb.WriteString("Action: <description>\n")
	systemMsg := models.Message{
		Role:    "system",
		Content: sb.String(),
	}
	userMsg := models.Message{
		Role:    "user",
		Content: "What is your next action for this step?",
	}
	return []models.Message{systemMsg, userMsg}
}
