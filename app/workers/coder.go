package workers

import (
	"fmt"
	"strings"

	"GoWorkerAI/app/actions"
	"GoWorkerAI/app/models"
)

type Coder struct {
	Worker
	Language   string
	CodeStyles []string
	Tests      bool
	TestStyles []string
}

func NewCoder(language, task string, codeStyles, acceptConditions, rules, testStyles []string, tests bool,
	maxIterations int, folder string, lockFolder bool) *Coder {
	return &Coder{
		Worker: Worker{
			Task:  &Task{Task: task, AcceptConditions: acceptConditions, MaxIterations: maxIterations},
			Rules: rules, LockFolder: lockFolder, Folder: folder},
		Language: language, CodeStyles: codeStyles, Tests: tests, TestStyles: testStyles,
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
	sb.WriteString(fmt.Sprintf("- **Requires Tests?** %v\n", c.Tests))
	sb.WriteString(fmt.Sprintf("- **Test Styles:** %v (If tests are required)\n", c.TestStyles))
	return sb.String()
}

func (c Coder) PromptPlan() []models.Message {
	var sysBuilder strings.Builder
	sysBuilder.WriteString("You are an expert software engineer tasked with planning the implementation of a coding task. ")
	sysBuilder.WriteString("Analyze the given problem and generate a structured, step-by-step development plan.\n\n")
	sysBuilder.WriteString("### **Instructions:**\n")
	sysBuilder.WriteString("1. Break down the task into small, actionable steps.\n")
	sysBuilder.WriteString("2. Consider potential risks and outline measures to mitigate them.\n")
	sysBuilder.WriteString("3. Ensure adherence to the specified code style and development rules.\n")
	if c.Tests {
		sysBuilder.WriteString("4. Include a plan for writing and executing tests following the provided test styles.\n")
	}
	sysBuilder.WriteString("\n### **Output Format (Numbered List of Steps Only):**\n")
	sysBuilder.WriteString("1. [Step 1]\n2. [Step 2]\n... \nN. [Final step]\n\n")
	sysBuilder.WriteString("Do **NOT** write any code at this stage; focus solely on planning.")

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

func (c Coder) PromptNextAction(plan string, actions []actions.Action, executedActions []models.ActionTask) []models.Message {
	var actionsDescBuilder strings.Builder
	for _, action := range actions {
		actionsDescBuilder.WriteString(fmt.Sprintf("- `%s`: %s\n", action.Key, action.Description))
	}

	var executedActionsBuilder strings.Builder
	for i, act := range executedActions {
		executedActionsBuilder.WriteString(fmt.Sprintf("Iteration %d:\n   - ActionTask: %v\n", i+1, act))
	}

	systemPrompt := fmt.Sprintf(
		`You are a software engineer AI tasked with generating code based on the given plan.
		Your response must strictly adhere to a JSON format and use one of the available actions to move the task forward.
		
		### Available Actions:
		%s
		
		### JSON Output Format:
		{
		  "action": "<selected_action>",
		  "filename": "<file>",
		  "content": "<code or file content>"
		}
		
		Ensure your response is a valid JSON object. Do not return plain text.
		Default action list_files with empty filename to search root`,
		actionsDescBuilder.String(),
	)

	userPrompt := fmt.Sprintf(
		`**Plan:**
		%s
		
		### Executed Actions (Past Iterations):
		%s
		
		Review the last steps executed
		What should be the next action needed to complete the task?`,
		plan, executedActionsBuilder.String(),
	)

	return []models.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
}

func (c Coder) PromptValidation(plan string, actions []models.ActionTask) []models.Message {
	var actionsSummaryBuilder strings.Builder
	for i, act := range actions {
		actionsSummaryBuilder.WriteString(fmt.Sprintf("Iteration %d:\n", i+1))
		actionsSummaryBuilder.WriteString(fmt.Sprintf("  - ActionTask: %s\n", act.Action))
		if act.Filename != "" {
			actionsSummaryBuilder.WriteString(fmt.Sprintf("  - Filename: %s\n", act.Filename))
		}
		if act.Content != "" {
			actionsSummaryBuilder.WriteString(fmt.Sprintf("  - Content: %s\n", act.Content))
		}
		actionsSummaryBuilder.WriteString("\n")
	}

	systemMessage := models.Message{
		Role: "system",
		Content: "You are a strict validation AI responsible for verifying if the generated code fully meets all task requirements.\n\n" +
			"### **Output Format (One Character Response Only):**\n" +
			"- \"true\" → The task meets all criteria.\n" +
			"- \"false\" → The task is incomplete or incorrect.\n\n" +
			"If the code fails validation (\"false\"), the system will iterate until all conditions are met.",
	}

	userMessage := models.Message{
		Role: "user",
		Content: fmt.Sprintf(
			"### **Development Plan:**\n%s\n\n### **Context / Current Status / Iterations:**\n%s\n\n"+
				"Based on the above information, have all necessary steps been completed to finalize the task plan?",
			plan, actionsSummaryBuilder.String(),
		),
	}

	return []models.Message{
		systemMessage,
		userMessage,
	}
}
