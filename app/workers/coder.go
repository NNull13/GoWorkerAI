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
	sb.WriteString("### **Task Information:**\n")
	sb.WriteString(fmt.Sprintf("- **Programming Language:** %s\n", c.Language))
	sb.WriteString(fmt.Sprintf("- **Main Task:** %s\n", c.Task.Task))
	sb.WriteString(fmt.Sprintf("- **Code Style Preferences:** %v (Standards to follow)\n", c.CodeStyles))
	sb.WriteString(fmt.Sprintf("- **Accepted Conditions:** %v (Requirements to be met)\n", c.Task.AcceptConditions))
	sb.WriteString(fmt.Sprintf("- **Development Rules:** %v (Mandatory constraints)\n", c.Rules))
	return sb.String()
}

func (c Coder) PromptPlan() []models.Message {
	var sysBuilder strings.Builder
	sysBuilder.WriteString("You are an expert software engineer specializing in structured software planning.\n")
	sysBuilder.WriteString("Your task is to create a detailed and actionable development plan before any code is written.\n\n")

	sysBuilder.WriteString("### **Reasoning Approach:**\n")
	sysBuilder.WriteString("1. Think step by step about the dependencies and order of operations.\n")
	sysBuilder.WriteString("2. Identify potential risks and mitigation strategies.\n")
	sysBuilder.WriteString("3. Ensure all steps comply with:\n")
	if c.Tests {
		sysBuilder.WriteString("4. Include a structured plan for testing, following the specified test styles.\n")
	}
	sysBuilder.WriteString("   - **Code style guidelines**\n")
	sysBuilder.WriteString("   - **Development rules**\n")
	sysBuilder.WriteString("   - **Accepted conditions**\n\n")

	sysBuilder.WriteString("\n### **Strict Output Format:**\n")
	sysBuilder.WriteString("Provide the steps in a numbered list **without explanations**:\n")
	sysBuilder.WriteString("```\n1. [Step 1]\n2. [Step 2]\n...\nN. [Final step]\n```\n")
	sysBuilder.WriteString("⚠ **DO NOT generate any code at this stage. Only planning.**")

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
		`You are an AI-powered software engineer tasked with executing coding tasks step by step.
		Your task is to determine the next action strictly based on the given plan and the steps executed before`,
	)

	userPrompt := fmt.Sprintf(
		`### **Development Plan:**\n%s\n\n### **Executions resume:**\n%s\n\n
		What should be the next action to move the task forward?`,
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
		Content: "You are a strict validation AI responsible for determining if the task has been fully completed.\n\n" +
			"### **Validation Rules:**\n" +
			"- Check if all required steps in the plan have been executed.\n" +
			"- Ensure the output follows the required coding styles and rules.\n" +
			"- Confirm that any test requirements have been met.\n\n" +
			"### **Strict Output Format:**\n" +
			"- `true` → The task meets all criteria and is complete.\n" +
			"- `false` → The task is incomplete or incorrect.\n\n" +
			"⚠ **Do not provide explanations. Respond strictly with `true` or `false`.**",
	}

	userMessage := models.Message{
		Role: "user",
		Content: fmt.Sprintf(
			"### **Development Plan:**\n%s\n\n### **Executed Actions: **\n%s\n\n"+
				"Has the task been fully completed according to the plan? Respond with `true` or `false`.",
			plan, recordsResume,
		),
	}

	return []models.Message{
		systemMessage,
		userMessage,
	}
}
