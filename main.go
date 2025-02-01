package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

func main() {
	var runtimes []Runtime
	modelClient := NewModelClient()

	coders := []Coder{
		{
			Language:       "Go",
			Task:           "Implement a monolithic HTTP server in a single file.",
			ProblemToSolve: "Develop a single-file Go HTTP server that handles multiple routes and logs all requests.",
			Risks: []string{
				"Server crashes on invalid requests.",
				"Routes not handled correctly.",
				"Logs not formatted properly.",
				"Server does not close properly on shutdown.",
			},
			CodeStyles: []string{
				"Keep all functionality in a single Go file.",
				"Use idiomatic Go practices.",
				"Handle errors properly to prevent crashes.",
			},
			AcceptConditions: []string{
				"Server must run on port 8080.",
				"Must support at least two routes (`/` and `/status`).",
				"Logs must include timestamp and request details.",
			},
			Rules: []string{
				"Use the `net/http` package for HTTP handling.",
				"Implement structured logging using `log` package.",
				"Ensure the server shuts down properly using `context` package.",
			},
			Tests:         true,
			TestStyles:    []string{"Use `httptest` package for testing the `/status` route."},
			MinIterations: 3,
			MaxIterations: 10,
		},
	}

	var wg sync.WaitGroup

	for _, coder := range coders {
		runtimes = append(runtimes, Runtime{coder: coder, model: modelClient})
	}

	for _, r := range runtimes {
		wg.Add(1)
		go func(rt Runtime) {
			defer wg.Done()
			rt.run()
		}(r)
	}

	fmt.Println("All runtimes started. Running indefinitely...")

	wg.Wait()

}

type Runtime struct {
	coder Coder
	model *ModelClient
}

func (r Runtime) run() {
	promptPlan := r.coder.PromptPlan()
	plan, err := r.model.Process(promptPlan)
	if err != nil {
		log.Panicf("error processing %v: %v\n", r.coder, err)
	}
	fmt.Println("Generated Plan:", plan)

	ticker := time.NewTicker(1 * time.Second) //(5 * time.Minute)
	defer ticker.Stop()

	folderName := time.Now().Format("20060102_150405")
	logFile := folderName + "/execution_log.txt"

	var validationResult bool
	var generatedCode string
	for i := 0; i < r.coder.MaxIterations && !validationResult && i >= r.coder.MinIterations; i++ {
		select {
		case <-ticker.C:
			fmt.Println("Processing:", r.coder.Task)

			promptWorker := r.coder.PromptCodeGeneration(plan, generatedCode)
			generatedCode, err = r.model.Process(promptWorker)
			if err != nil {
				panic(err)
			}
			fmt.Println("Generated Code:\n", generatedCode)

			promptValidation := r.coder.PromptValidation(plan, generatedCode)
			validationResult, err = r.model.YesOrNo(promptValidation, 3)
			fmt.Println("Validation Result:", validationResult)

			appendToFile(fmt.Sprint(logFile, i), r.coder.Task, generatedCode, validationResult)

			if validationResult {
				fmt.Println("Task completed successfully:", r.coder.Task)
			} else {
				fmt.Println("Task failed, retrying:", r.coder.Task)
			}
		}
	}
}

func appendToFile(filename, task, code string, validation bool) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	logEntry := fmt.Sprintf(
		"--- Task: %s ---\nGenerated Code:\n%s\nValidation Result: %v\n\n",
		task, code, validation,
	)

	if _, err = file.WriteString(logEntry); err != nil {
		fmt.Println("Error writing to file:", err)
	}
}
