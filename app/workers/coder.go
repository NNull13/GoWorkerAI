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
	sysBuilder.WriteString("You are an expert software engineer and strategic planner. Your task is to develop a highly detailed, step-by-step development plan for the following coding task. **Do not generate any code at this stage.**\n\n")
	sysBuilder.WriteString("### Key Instructions:\n")
	sysBuilder.WriteString("- Break down the task into clear, logical steps, considering all dependencies and execution order.\n")
	sysBuilder.WriteString("- Identify potential risks and propose mitigation strategies.\n")
	sysBuilder.WriteString("- Ensure that each step is actionable and adheres to industry best practices.\n")
	sysBuilder.WriteString("- If any part of the task is unclear, specify what additional information you need before proceeding.\n")
	if c.Tests {
		sysBuilder.WriteString("- Include a structured test plan covering critical test cases and validation steps.\n")
	}
	sysBuilder.WriteString("\n### Required Output Format:\n")
	sysBuilder.WriteString("```\n1. [Description of the first step]\n2. [Description of the next step]\n...\nN. [Final step]\n```\n")
	sysBuilder.WriteString("⚠ **Important:** Only provide the plan. Do not include any code.\n")

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
		`You are an AI-powered software engineer tasked with executing development steps iteratively and precisely. Your current goal is to identify **the one and only next immediate action** that should be taken, based solely on the provided development plan and execution history.
		
		### Guidelines:
		- Propose **only one action** per response.
		- Do not skip or rearrange the steps outlined in the plan.
		- Ensure that the proposed action respects all dependencies and established guidelines.
		- If the execution history is ambiguous or incomplete, specify what additional details are needed before proceeding.
		`,
	)

	userPrompt := fmt.Sprintf(
		`### **Development Plan:**\n%s\n\n### **Execution History:**\n%s\n\n
		What is the next immediate action to perform? Please respond in the format provided above.`,
		plan, resume,
	)

	return []models.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
}

func (c Coder) PromptValidation(plan, recordsResume string) []models.Message {
	systemMessage := models.Message{
		Role: "system",
		Content: `
		You are an expert task validator. Your mission is to determine if **every step** in the development plan has been executed correctly and all coding guidelines, development rules, and testing requirements have been met.
		
		### Validation Rules:
		- Verify that **all steps** in the plan have been fully completed.
		- Confirm adherence to coding style guidelines, development constraints, and test requirements (if applicable).
		
		### Strict Output Format:
		Respond **only** with:
		- ` + "`true`" + ` → if the task is completely and correctly executed.
		- ` + "`false`" + ` → if any step is missing or any discrepancy is found.

		⚠ **Do not include any additional explanation or commentary.**
		`,
	}

	userMessage := models.Message{
		Role: "user",
		Content: fmt.Sprintf(
			"### Development Plan:\n%s\n\n### Executed Actions:\n%s\n\nHas the task been fully completed as per the plan? Respond only with `true` or `false`.",
			plan, recordsResume,
		),
	}

	return []models.Message{
		systemMessage,
		userMessage,
	}
}
