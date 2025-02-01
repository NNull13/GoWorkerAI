package main

import "fmt"

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
	MinIterations    int
	MaxIterations    int
}

func (c Coder) TaskInformation() string {
	return fmt.Sprint(
		"### **Task Information:**\n"+
			"- **Programming Language:** ", c.Language, "\n"+
			"- **Main Task:** ", c.Task, "\n"+
			"- **Problem to Solve:** ", c.ProblemToSolve, "\n"+
			"- **Potential Risks:** ", c.Risks, " (List of challenges that must be considered)\n"+
			"- **Code Style Preferences:** ", c.CodeStyles, " (Standards to follow)\n"+
			"- **Accepted Conditions:** ", c.AcceptConditions, " (Requirements that must be met)\n"+
			"- **Development Rules:** ", c.Rules, " (Mandatory constraints)\n"+
			"- **Requires Tests?** ", c.Tests, "\n"+
			"- **Test Styles:** ", c.TestStyles, " (If tests are required)\n"+
			"- **Minimum Iterations Required:** ", c.MinIterations, "\n\n")
}

func (c Coder) PromptPlan() []Message {
	return []Message{
		{
			Role: "system",
			Content: fmt.Sprint(
				"You are an expert software engineer responsible for planning the correct implementation of a coding task. Your job is to **analyze the given problem and generate a structured, step-by-step development plan**"+
					"### **Instructions:**\n"+
					"1. Analyze the task and break it down into **small, actionable steps** for implementation.\n"+
					"2. Consider potential risks and ensure each step minimizes them.\n"+
					"3. Ensure that the coding style follows `", c.CodeStyles, "` and the rules `", c.Rules, "`.\n"+
					"4. If `", c.Tests, "` = true, include a plan for writing and running `", c.TestStyles, "`.\n"+
					"5. The plan should guarantee that the minimum iterations `", c.MinIterations, "` are met.\n\n"+
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
	return []Message{
		{
			Role: "system",
			Content: fmt.Sprint(
				"You are an expert software engineer and your job is to **implement the given plan by writing high-quality code**.\n\n",
				c.TaskInformation(),
				"### **Development Plan :**\n", plan, "\n\n"+
					"### **Instructions:**\n"+
					"1. Implement the task in `", c.Language, "` by following the **step-by-step development plan**.\n"+
					"2. Ensure that the code strictly adheres to the `", c.CodeStyles, "` and `", c.Rules, "`.\n"+
					"3. If `", c.Tests, "` = true, include test cases following `", c.TestStyles, "`.\n"+
					"4. The code must handle **all risks** identified in `", c.Risks, "`.\n"+
					"5. Ensure that the implementation meets all **accepted conditions**: `", c.AcceptConditions, "`.\n\n"+
					"### **Output Format:**\n"+
					"- **Return only the final, complete code.**\n"+
					"- If tests are required, include a separate test file or function.\n"+
					"- Do **NOT** explain the code, only provide the implementation.\n\n"+
					"If any step cannot be completed due to missing information, **clearly indicate what is needed**.\n"),
		},
		{
			Role:    "user",
			Content: "### **Generated Code / Current status / Current code:**\n" + generatedCode,
		},
	}
}

func (c Coder) PromptValidation(plan, generatedCode string) []Message {
	return []Message{
		{
			Role: "system",
			Content: fmt.Sprint("You are a strict validation AI responsible for **verifying if the generated code fully meets all the task requirements**.\n\n"+
				c.TaskInformation()+
				"### **Development Plan :**\n", plan, "\n\n"+
				"### **Instructions:**\n"+
				"1. Compare the `", generatedCode, "` against the task and **Development Plan**, ensuring all conditions are met.\n"+
				"2. Validate that the code:\n"+
				"   - **Fulfills the task**: `", c.Task, "`\n"+
				"   - **Solves the problem correctly**: `", c.ProblemToSolve, "`\n"+
				"   - **Mitigates all risks**: `", c.Risks, "`\n"+
				"   - **Follows the coding style**: `", c.CodeStyles, "`\n"+
				"   - **Respects all mandatory rules**: `", c.Rules, "`\n"+
				"   - **Implements required tests** (if `", c.Tests, "` = true and follows `", c.TestStyles, "`)\n"+
				"   - **Includes at least `", c.MinIterations, "` iterations (if applicable)**\n\n"+
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
