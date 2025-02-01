package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

type Runtime struct {
	coder Coder
	model *ModelClient
}

func (r Runtime) run() {
	promptPlan := r.coder.PromptPlan()
	plan, err := r.model.Think(promptPlan)
	if err != nil {
		log.Panicf("Error generating plan: %v\n", err)
	}
	fmt.Println("Generated Plan:\n", plan)

	folderName := time.Now().Format("20060102_150405")
	generationFolder := filepath.Join("generations", folderName)
	if err := os.MkdirAll(generationFolder, os.ModePerm); err != nil {
		log.Fatalf("Error creating generation directory %s: %v", generationFolder, err)
	}

	logsFolder := "logs"
	if err := os.MkdirAll(logsFolder, os.ModePerm); err != nil {
		log.Fatalf("Error creating logs directory %s: %v", logsFolder, err)
	}
	logFile := filepath.Join(logsFolder, time.Now().Format("20060102")+".log")

	var validationResult bool
	var generatedCode string
	for i := 0; i <= r.coder.MaxIterations; i++ {
		fmt.Println("Processing:", r.coder.Task)

		promptWorker := r.coder.PromptCodeGeneration(plan, generatedCode)
		action, err := r.model.Process(promptWorker)
		if err != nil {
			log.Printf("Error processing action: %s | Action: %v", err, action)
			continue
		}

		switch action.Action {
		case "write_file":
			writeToFile(generationFolder, action.Filename, action.Content)
			generatedCode = action.Content
		case "read_file":
			generatedCode = readFile(generationFolder, action.Filename)
		case "edit_file":
			editFile(generationFolder, action.Filename, action.Content)
			generatedCode = readFile(generationFolder, action.Filename)
		case "delete_file":
			deleteFile(generationFolder, action.Filename)
		}

		promptValidation := r.coder.PromptValidation(plan, generatedCode)
		validationResult, err = r.model.YesOrNo(promptValidation, 3)
		if err != nil {
			log.Printf("Validation error: %v", err)
		}

		fmt.Println("Validation Result:", validationResult)
		appendToFile(logFile, r.coder.Task, action, validationResult)

		if validationResult {
			fmt.Println("✅ Task completed successfully:", r.coder.Task)
			break
		}
	}
}

func writeToFile(baseDir, filename, content string) {
	path := filepath.Join(baseDir, filename)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		log.Printf("Error writing file %s: %v\n", path, err)
	} else {
		log.Printf("✅ File %s written successfully.\n", path)
	}
}

func readFile(baseDir, filename string) string {
	path := filepath.Join(baseDir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Error reading file %s: %v\n", path, err)
		return ""
	}
	return string(content)
}

func editFile(baseDir, filename, newContent string) {
	existingContent := readFile(baseDir, filename)
	mergedContent := existingContent + "\n" + newContent
	writeToFile(baseDir, filename, mergedContent)
}

func deleteFile(baseDir, filename string) {
	path := filepath.Join(baseDir, filename)
	err := os.Remove(path)
	if err != nil {
		log.Printf("Error deleting file %s: %v\n", path, err)
	} else {
		log.Printf("✅ File %s deleted successfully.\n", path)
	}
}

func appendToFile(filename, task string, action *Action, validation bool) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer file.Close()

	logEntry := fmt.Sprintf(
		"Timestamp: %s\n--- Task: %s ---\nAction:\n%v\nValidation Result: %v\n\n",
		time.Now().Format(time.RFC3339), task, action, validation,
	)

	if _, err = file.WriteString(logEntry); err != nil {
		fmt.Println("Error writing to log file:", err)
	}
}
