package app

import (
	"fmt"
	"log"
	"os"
	"time"
)

func appendActionLog(filename, task string, action *Action, validation bool) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening log file: %s", err)
		return
	}
	defer file.Close()

	logEntry := fmt.Sprintf(
		"Timestamp: %s\n--- Task: %s ---\nAction:\n%v\nValidation Result: %v\n\n",
		time.Now().Format(time.RFC3339), task, action, validation,
	)

	if _, err = file.WriteString(logEntry); err != nil {
		log.Printf("Error writing to log file: %v", err)
	}
}

func appendLLMLog(filename, llmOutput string) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening log file: %v", err)
		return
	}
	defer file.Close()

	logEntry := fmt.Sprintf(
		"Timestamp: %s\n--- LLM Output:\n%s\n\n",
		time.Now().Format(time.RFC3339), llmOutput,
	)

	if _, err = file.WriteString(logEntry); err != nil {
		log.Printf("Error writing to log file: %v", err)
	}
}
