package workers

import (
	"fmt"
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

func NewCoder(language, task string, codeStyles, acceptConditions, rules []string, maxIterations int, folder string, tests, lockFolder bool) *Coder {
	return &Coder{
		Worker: Worker{
			Task:  &Task{ID: uuid.New(), Task: task, AcceptConditions: acceptConditions, MaxIterations: maxIterations},
			Rules: rules, LockFolder: lockFolder, Folder: folder},
		Language: language, CodeStyles: codeStyles, Tests: tests,
	}
}

func (c Coder) TaskInformation() string {
	var sb strings.Builder
	sb.WriteString("### 📝 **Task Information**\n\n")
	sb.WriteString("#### 📌 **General Details:**\n")
	sb.WriteString(fmt.Sprintf("- **Programming Language:** `%s`\n", c.Language))
	sb.WriteString(fmt.Sprintf("- **Main Task:** %s\n", c.Task.Task))
	sb.WriteString(fmt.Sprintf("- **Working Directory:** `%s` (Lock: %t)\n", c.Folder, c.LockFolder))
	sb.WriteString("\n#### 🎨 **Code Style & Constraints:**\n")

	if len(c.CodeStyles) > 0 {
		sb.WriteString(fmt.Sprintf("- **Code Styles:** %s\n", strings.Join(c.CodeStyles, ", ")))
	}

	if len(c.Task.AcceptConditions) > 0 {
		sb.WriteString(fmt.Sprintf("- **Accepted Conditions:** %s\n", strings.Join(c.Task.AcceptConditions, ", ")))
	}

	if len(c.Rules) > 0 {
		sb.WriteString(fmt.Sprintf("- **Development Rules:** %s\n", strings.Join(c.Rules, ", ")))
	}

	sb.WriteString("\n#### 🛠️ **Testing & Validation:**\n")
	sb.WriteString(fmt.Sprintf("- **Testing Required:** %t\n", c.Tests))

	return sb.String()
}

func (c Coder) PromptPlan() []models.Message {
	var sysBuilder strings.Builder
	sysBuilder.WriteString("You are an expert software engineer specializing in structured development planning.\n")
	sysBuilder.WriteString("Your task is to create a highly detailed, executable plan for the given coding task **before any code is written**.\n\n")
	sysBuilder.WriteString("### **Key Guidelines:**\n")
	sysBuilder.WriteString("- Think **step by step**, considering dependencies and execution order.\n")
	sysBuilder.WriteString("- Identify **potential risks** and propose mitigations.\n")
	sysBuilder.WriteString("- The plan **must be actionable**, avoiding generalities.\n")
	sysBuilder.WriteString("- Ensure **full compliance with**:\n")
	sysBuilder.WriteString("  - **Coding style guidelines**\n")
	sysBuilder.WriteString("  - **Development rules and constraints**\n")
	sysBuilder.WriteString("  - **Accepted conditions and test requirements**\n")

	if c.Tests {
		sysBuilder.WriteString("- **Include a structured test plan** covering the necessary test types.\n")
	}

	sysBuilder.WriteString("\n### **Strict Output Format:**\n")
	sysBuilder.WriteString("```\n1. [First action]\n2. [Next action]\n...\nN. [Final step]\n```\n")
	sysBuilder.WriteString("⚠ **DO NOT generate any code yet. This is only a structured plan.**")

	systemMessage := models.Message{
		Role:    "system",
		Content: sysBuilder.String(),
	}

	userMessage := models.Message{
		Role:    "user",
		Content: c.TaskInformation(),
	}

	return []models.Message{systemMessage, userMessage}
}

func (c Coder) PromptNextAction(plan, resume string) []models.Message {
	systemPrompt := fmt.Sprintf(
		`You are an AI-powered software engineer responsible for step-by-step software execution. 
		Your task is to determine the next coding action strictly based on the provided plan and execution history.
		### **Guidelines:**
		- Max 5 iterations with assistant by once
		- Only suggest the next immediate step from the plan.
		- Do NOT skip steps or introduce actions outside the plan.
		- Ensure the next action aligns with dependencies in the project.`,
	)

	userPrompt := fmt.Sprintf(
		`### **Development Plan:**\n%s\n\n### **Execution History:**\n%s\n\n
		What should be the next action? Follow the output format strictly.`,
		plan, resume,
	)

	return []models.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
}

func (c Coder) PromptValidation(plan string, recordsResume string) []models.Message {
	systemMessage := models.Message{
		Role: "system",
		Content: `You are an AI tasked with strict validation of task completion.
		### **Validation Rules:**
		- Confirm that **every step in the plan** has been executed.
		- Validate that all **code style rules** and **development constraints** have been followed.
		- Ensure that **test requirements** (if applicable) have been satisfied.
		### **Strict Output Format:**
		- ` + "`true` → The task meets ALL criteria and is fully complete.`" + `
		- ` + "`false` → The task is incomplete, incorrect, or missing steps.`" + `
		⚠ **Do NOT provide explanations. Respond strictly with true or false.**`,
	}

	userMessage := models.Message{
		Role: "user",
		Content: fmt.Sprintf(
			"### **Development Plan:**\n%s\n\n### **Executed Actions:**\n%s\n\n"+
				"Has the task been fully completed according to the plan? Respond with `true` or `false`.",
			plan, recordsResume,
		),
	}

	return []models.Message{
		systemMessage,
		userMessage,
	}
}
