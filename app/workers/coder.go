package workers

import (
	"fmt"
	"strings"

	"GoWorkerAI/app/models"
	"github.com/google/uuid"
)

type Coder struct {
	Worker
	Language   string
	CodeStyles []string
	Tests      bool
}

func NewCoder(
	language, task, toolPreset string,
	codeStyles, acceptConditions, rules []string,
	maxIterations int,
	folder string,
	tests, lockFolder bool,
) *Coder {
	return &Coder{
		Worker: Worker{
			Task: &Task{
				ID:               uuid.New(),
				Task:             task,
				AcceptConditions: acceptConditions,
				MaxIterations:    maxIterations,
			},
			ToolsPreset: toolPreset,
			Rules:       rules,
			LockFolder:  lockFolder,
			Folder:      folder,
		},
		Language:   language,
		CodeStyles: codeStyles,
		Tests:      tests,
	}
}

// ------- Improved TaskInformation (more context, still compact)

func (c *Coder) TaskInformation() string {
	baseInfo := c.Worker.TaskInformation()
	var sb strings.Builder
	sb.WriteString(baseInfo)
	sb.WriteString(fmt.Sprintf("Programming Language: %s\n", c.Language))
	if len(c.CodeStyles) > 0 {
		sb.WriteString(fmt.Sprintf("Code Styles: %s\n", strings.Join(c.CodeStyles, ", ")))
	}
	sb.WriteString(fmt.Sprintf("Testing Required: %t\n", c.Tests))

	// Extra helpful context for coding tasks
	if c.ToolsPreset != "" {
		sb.WriteString(fmt.Sprintf("Tools Preset: %s\n", c.ToolsPreset))
	}
	if len(c.Rules) > 0 {
		sb.WriteString(fmt.Sprintf("Rules: %s\n", strings.Join(c.Rules, " | ")))
	}
	if c.Folder != "" {
		sb.WriteString(fmt.Sprintf("Working Folder: %s\n", c.Folder))
	}
	sb.WriteString(fmt.Sprintf("Lock Folder: %t\n", c.LockFolder))
	return sb.String()
}

// ------- Shared coder preamble (consistent behavior across prompts)

func (c *Coder) coderPreamble() string {
	var sb strings.Builder
	sb.WriteString(strings.Join([]string{
		"You are a senior software engineer and careful editor.",
		"Goals:",
		"- Write correct, minimal, maintainable code that compiles and runs.",
		"- Prefer small, idempotent changes with clear file operations.",
		"- Follow the specified language and code styles.",
		"- If information is missing, add explicit TODOs or clarification steps (but keep outputs in the required format).",
		"General Rules:",
		"- Deterministic output. No filler text, no extra commentary.",
		"- If you are unsure, choose the safest, reversible change.",
		"- Keep diffs minimal; do not rewrite unrelated code.",
	}, "\n"))
	return sb.String()
}

func (c *Coder) PromptPlan(taskInformation string) []models.Message {
	sys := planSystemPrompt(c.coderPreamble())
	user := strings.Join([]string{
		taskInformation,
		"Generate the technical development plan, strictly following the format rules.",
	}, "\n\n")
	return []models.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: user},
	}
}

// ------- Implementation: produce ONLY JSON file operations

// Schema (documented inside the prompt):
// [
//   {
//     "op": "create|replace|append|edit_region|delete",
//     "path": "relative/file/path",
//     "language": "go|ts|py|...",
//     "description": "≤ 15 words",
//     "code": "full file content or snippet (required for create/replace/append)",
//     "region": {"start": "marker or line", "end": "marker or line"} // required for edit_region
//   }
// ]

func (c *Coder) PromptCodeImplement(context, plan, repoState string) []models.Message {
	sys := strings.Join([]string{
		c.coderPreamble(),
		"Implement the next changes as a list of file operations.",
		"OUTPUT FORMAT (MANDATORY):",
		"- Return ONLY a JSON array of operations as per the schema below. No prose.",
		"SCHEMA:",
		`- op: "create" | "replace" | "append" | "edit_region" | "delete"`,
		"- path: relative POSIX path within the working folder.",
		"- language: programming language (e.g., go, ts, py).",
		"- description: brief rationale (<= 15 words).",
		"- code: required for create/replace/append. For replace, provide the full new file content.",
		"- region: object with 'start' and 'end' (required for edit_region).",
		"RULES:",
		"- Prefer minimal diffs (edit_region or append) over full replacements when feasible.",
		"- Ensure compilable/buildable state after operations if possible.",
		"- Follow language and code style settings.",
		"- Do not touch files outside scope.",
		"- Deterministic ordering of operations by dependency.",
		"IMPORTANT:",
		"- Output MUST be valid JSON. No trailing commas. No comments. No extra text.",
	}, "\n")
	user := strings.Join([]string{
		"Task & Context:\n" + context,
		"\nPlan (numbered):\n" + plan,
		"\nRepository State (files, key snippets, constraints):\n" + repoState,
		"\nReturn ONLY the JSON operations array.",
	}, "\n")
	return []models.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: user},
	}
}

// ------- Tests: generate minimal, runnable tests only

func (c *Coder) PromptWriteTests(context, repoState string) []models.Message {
	sys := strings.Join([]string{
		c.coderPreamble(),
		"Write minimal, high-value automated tests for the specified language and project.",
		"OUTPUT FORMAT (MANDATORY):",
		"- Return ONLY a JSON array of file operations (same schema as implementation).",
		"TEST RULES:",
		"- Favor fast, deterministic unit tests over slow integration tests unless specified.",
		"- Cover critical paths and edge cases; avoid over-mocking.",
		"- If the project uses a specific framework (e.g., Go testing, Jest, Pytest), follow conventions.",
		"- Include any necessary test scaffolding and fixtures.",
		"- Keep CI compatibility in mind (no environment secrets, no network unless allowed).",
	}, "\n")
	user := strings.Join([]string{
		"Task Context:\n" + context,
		"\nRepository State:\n" + repoState,
		"\nReturn ONLY the JSON operations array.",
	}, "\n")
	return []models.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: user},
	}
}

// ------- Error-driven fixes: consume errors/logs and output exact patches

func (c *Coder) PromptFixFromErrors(errorsLog, repoState string) []models.Message {
	sys := strings.Join([]string{
		c.coderPreamble(),
		"Fix the issues indicated by the logs.",
		"OUTPUT FORMAT: ONLY JSON array of file operations (same schema as implementation).",
		"RULES:",
		"- Address the root cause(s) indicated by the errors.",
		"- Keep diffs minimal and localized.",
		"- If uncertain, add TODOs and safest guards.",
		"- Ensure the code compiles/builds after changes if possible.",
	}, "\n")
	user := strings.Join([]string{
		"Errors/Logs:\n" + errorsLog,
		"\nRepository State:\n" + repoState,
		"\nReturn ONLY the JSON operations array.",
	}, "\n")
	return []models.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: user},
	}
}

// ------- Review: crisp verdict with optional minimal patches

// Output is a tiny JSON object to make it easy to parse:
// {
//   "approved": true|false,
//   "notes": ["short item", "..."],
//   "ops": [ ... // optional file operations, same schema as above ]
// }

func (c *Coder) PromptCodeReview(diffSummary, guidelines string) []models.Message {
	sys := strings.Join([]string{
		c.coderPreamble(),
		"Perform a strict code review against the guidelines.",
		"OUTPUT FORMAT (MANDATORY):",
		"- Return ONLY a JSON object with fields: approved (bool), notes (array of short strings), ops (optional array of file ops).",
		"REVIEW CRITERIA:",
		"- Correctness, safety (errors, panics, concurrency), style, naming, tests, docs.",
		"- Avoid bike-shedding; only block on real defects or policy violations.",
	}, "\n")
	user := strings.Join([]string{
		"Guidelines:\n" + guidelines,
		"\nProposed Changes (diff or summary):\n" + diffSummary,
		"\nReturn ONLY the JSON object.",
	}, "\n")
	return []models.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: user},
	}
}

func (c *Coder) PromptNextCommand(goal, history, repoState string) []models.Message {
	sys := strings.Join([]string{
		c.coderPreamble(),
		"Decide the single next shell command to execute to make progress toward the goal.",
		"OUTPUT FORMAT (one line):",
		"- If a command is needed: `CMD: <shell command>`",
		"- If nothing else is needed: `DONE`",
		"RULES:",
		"- Prefer read/verify commands (build, test, lint) before destructive operations.",
		"- Use short, safe commands; avoid chained risky ops.",
	}, "\n")
	user := strings.Join([]string{
		"Goal:\n" + goal,
		"\nHistory (previous commands & results):\n" + history,
		"\nRepository State:\n" + repoState,
		"\nRespond only with `CMD: ...` or `DONE`.",
	}, "\n")
	return []models.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: user},
	}
}
