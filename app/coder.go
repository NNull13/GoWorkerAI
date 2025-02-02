package app

import (
	"fmt"
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
	Actions          map[string]string
	LockFolder       bool
}

func NewCoder(language, task string, risks, codeStyles, acceptConditions, rules, testStyles []string, tests bool, maxIterations int, actions map[string]string) Coder {
	for key, action := range defaultActions {
		if _, exists := actions[key]; !exists {
			actions[key] = action
		}
	}

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
		Actions:          actions,
	}
}

func (c Coder) TaskInformation() string {
	return fmt.Sprint(
		"### **Task Information:**\n"+
			"- **Programming Language:** ", c.Language, "\n"+
			"- **Main Task:** ", c.Task, "\n"+
			"- **Potential Risks:** ", c.Risks, " (List of challenges that must be considered)\n"+
			"- **Code Style Preferences:** ", c.CodeStyles, " (Standards to follow)\n"+
			"- **Accepted Conditions:** ", c.AcceptConditions, " (Requirements that must be met)\n"+
			"- **Development Rules:** ", c.Rules, " (Mandatory constraints)\n"+
			"- **Requires Tests?** ", c.Tests, "\n"+
			"- **Test Styles:** ", c.TestStyles, " (If Tests are required)\n")
}

func (c Coder) PromptPlan() []Message {
	return []Message{
		{
			Role: "system",
			Content: fmt.Sprint(
				"You are an expert software engineer responsible for planning the correct implementation of a coding Task. Your job is to **analyze the given problem and generate a structured, step-by-step development plan**"+
					"### **Instructions:**\n"+
					"1. Analyze the Task and break it down into **small, actionable steps** for implementation.\n"+
					"2. Consider potential Risks and ensure each step minimizes them.\n"+
					"3. Ensure that the coding style follows `", c.CodeStyles, "` and the Rules `", c.Rules, "`.\n"+
					"4. If `", c.Tests, "` = true, include a plan for writing and running `", c.TestStyles, "`.\n"+
					"### **Output Format (Only return a numbered list of steps):**\n"+
					"1. [Step 1]\n"+
					"2. [Step 2]\n"+
					"...\n"+
					"N. [Final step]\n\n"+
					"Do **NOT** write code at this stage. Focus **only** on planning the implementation correctly."),
		},
		{
			Role:    "user",
			Content: c.TaskInformation(),
		},
	}
}

func (c Coder) PromptCodeGeneration(plan, generatedCode string) []Message {
	var actionsDescription string
	for action, description := range c.Actions {
		actionsDescription += fmt.Sprintf("- `%s`: %s\n", action, description)
	}

	systemPrompt := fmt.Sprintf(
		`You are a software engineer AI. Generate Go code based on the given plan.
		Your responses must be in JSON format, using the correct action according to the task.
			
		### Available Actions:
		%s
			
		### JSON Output Format:
		{
		  "action": "<selected_action>",
		  "filename": "<file>.go",
		  "content": "<code or file content>"
		}
		
		### Task Details:
		- **Plan:** 
		%s
		
		Ensure that your response is structured as a valid JSON object. Never return plain text. 
		If unsure about which action to use, default to "write_file".`,
		actionsDescription, plan)

	return []Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Current Code:\n%s", generatedCode),
		},
	}
}

func (c Coder) PromptValidation(plan, generatedCode string) []Message {
	return []Message{
		{
			Role: "system",
			Content: fmt.Sprint("You are a strict validation AI responsible for **verifying if the generated code fully meets all the Task requirements**.\n\n"+
				c.TaskInformation()+
				"### **Development Plan :**\n", plan, "\n\n"+
				"### **Instructions:**\n"+
				"1. Compare the `", generatedCode, "` against the Task and **Development Plan**, ensuring all conditions are met.\n"+
				"2. Validate that the code:\n"+
				"   - **Fulfills the Task**: `", c.Task, "`\n"+
				"   - **Mitigates all Risks**: `", c.Risks, "`\n"+
				"   - **Follows the coding style**: `", c.CodeStyles, "`\n"+
				"   - **Respects all mandatory Rules**: `", c.Rules, "`\n"+
				"   - **Implements required Tests** (if `", c.Tests, "` = true and follows `", c.TestStyles, "`)\n"+
				"### **Output Format (Strictly One Character Response):**\n"+
				"- \"true\" → The implementation fully meets all criteria.\n"+
				"- \"false\" → The implementation is incorrect or incomplete.\n\n"+
				"If the code fails validation (`\"false\"`), the system will **iterate** until the code meets all conditions.\n"),
		},
		{
			Role:    "user",
			Content: "### **Generated Code / Current status / Current code:**\n" + generatedCode,
		},
	}
}
