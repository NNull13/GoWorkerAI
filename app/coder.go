package app

import (
	"fmt"
	"strings"
)

type Coder struct {
	Language         string
	Task             string
	ProblemToSolve   string
	Risks            []string
	CodeStyles       []string
	AcceptConditions []string
	Rules            []string
	Tests            bool
	TestStyles       []string
	MaxIterations    int
	LockFolder       bool
	Folder           string
	Actions          map[string]string
}

func NewCoder(language, task string, risks, codeStyles, acceptConditions, rules, testStyles []string, tests bool,
	maxIterations int, folder string, lockFolder bool) Coder {
	return Coder{
		Language:         language,
		Task:             task,
		Risks:            risks,
		CodeStyles:       codeStyles,
		AcceptConditions: acceptConditions,
		Rules:            rules,
		Tests:            tests,
		TestStyles:       testStyles,
		MaxIterations:    maxIterations,
		Folder:           folder,
		LockFolder:       lockFolder,
		Actions:          defaultActions,
	}
}

func (c Coder) TaskInformation() string {
	var sb strings.Builder
	sb.WriteString("### **Task Information:**\n")
	sb.WriteString(fmt.Sprintf("- **Programming Language:** %s\n", c.Language))
	sb.WriteString(fmt.Sprintf("- **Main Task:** %s\n", c.Task))
	sb.WriteString(fmt.Sprintf("- **Potential Risks:** %v (Challenges to be considered)\n", c.Risks))
	sb.WriteString(fmt.Sprintf("- **Code Style Preferences:** %v (Standards to follow)\n", c.CodeStyles))
	sb.WriteString(fmt.Sprintf("- **Accepted Conditions:** %v (Requirements to be met)\n", c.AcceptConditions))
	sb.WriteString(fmt.Sprintf("- **Development Rules:** %v (Mandatory constraints)\n", c.Rules))
	sb.WriteString(fmt.Sprintf("- **Requires Tests?** %v\n", c.Tests))
	sb.WriteString(fmt.Sprintf("- **Test Styles:** %v (If tests are required)\n", c.TestStyles))
	return sb.String()
}

func (c Coder) PromptPlan() []Message {
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

	systemMessage := Message{
		Role:    "system",
		Content: sysBuilder.String(),
	}

	userMessage := Message{
		Role:    "user",
		Content: c.TaskInformation(),
	}

	return []Message{systemMessage, userMessage}
}

func (c Coder) PromptCodeGeneration(plan string, executedActions []Action) []Message {
	var actionsDescBuilder strings.Builder
	for action, description := range c.Actions {
		actionsDescBuilder.WriteString(fmt.Sprintf("- `%s`: %s\n", action, description))
	}

	var executedActionsBuilder strings.Builder
	for i, act := range executedActions {
		executedActionsBuilder.WriteString(fmt.Sprintf("Iteration %d:\n   - Action: %v\n", i+1, act))
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

	return []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
}

func (c Coder) PromptValidation(plan string, actions []Action) []Message {
	var actionsSummaryBuilder strings.Builder
	for i, act := range actions {
		actionsSummaryBuilder.WriteString(fmt.Sprintf("Iteration %d:\n", i+1))
		actionsSummaryBuilder.WriteString(fmt.Sprintf("  - Action: %s\n", act.Action))
		if act.Filename != "" {
			actionsSummaryBuilder.WriteString(fmt.Sprintf("  - Filename: %s\n", act.Filename))
		}
		if act.Content != "" {
			actionsSummaryBuilder.WriteString(fmt.Sprintf("  - Content: %s\n", act.Content))
		}
		actionsSummaryBuilder.WriteString("\n")
	}

	systemMessage := Message{
		Role: "system",
		Content: "You are a strict validation AI responsible for verifying if the generated code fully meets all task requirements.\n\n" +
			"### **Output Format (One Character Response Only):**\n" +
			"- \"true\" → The task meets all criteria.\n" +
			"- \"false\" → The task is incomplete or incorrect.\n\n" +
			"If the code fails validation (\"false\"), the system will iterate until all conditions are met.",
	}

	userMessage := Message{
		Role: "user",
		Content: fmt.Sprintf(
			"### **Development Plan:**\n%s\n\n### **Context / Current Status / Iterations:**\n%s\n\n"+
				"Based on the above information, have all necessary steps been completed to finalize the task plan?",
			plan, actionsSummaryBuilder.String(),
		),
	}

	return []Message{
		systemMessage,
		userMessage,
	}
}
