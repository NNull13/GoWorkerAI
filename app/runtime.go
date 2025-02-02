package app

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

type Runtime struct {
	coder   Coder
	model   *ModelClient
	actions []Action
}

func NewRuntime(coder Coder, model *ModelClient) *Runtime {
	return &Runtime{
		coder: coder,
		model: model,
	}
}

func (r *Runtime) Run() {
	promptPlan := r.coder.PromptPlan()
	plan, err := r.model.Think(promptPlan)
	if err != nil {
		log.Panicf("Error generating plan: %v\n", err)
	}

	runtimeFolder := r.coder.Folder
	if r.coder.LockFolder {
		folderName := r.coder.Folder + time.Now().Format("20060102_150405")
		runtimeFolder = filepath.Join("generations", folderName)
		if err = os.MkdirAll(runtimeFolder, os.ModePerm); err != nil {
			log.Fatalf("Error creating generation directory %s: %v", runtimeFolder, err)
		}
	}

	logsFolder := "logs"
	if err = os.MkdirAll(logsFolder, os.ModePerm); err != nil {
		log.Fatalf("Error creating logs directory %s: %v", logsFolder, err)
	}
	logFile := filepath.Join(logsFolder, time.Now().Format("20060102")+".log")
	appendLLMLog(logFile, plan)

	var validationResult bool
	for i := 0; i <= r.coder.MaxIterations; i++ {
		var action *Action
		promptWorker := r.coder.PromptCodeGeneration(plan, r.actions)
		if action, err = r.model.Process(promptWorker); err != nil {
			log.Printf("Error processing action: %s | Action: %v", err, action)
			continue
		}

		log.Printf("Processing action: %s | Action: %v", action.Action, action)
		action.Result, err = ExecuteAction(action, runtimeFolder)
		if err != nil {
			log.Panicf("Error executing action: %s | Action: %v", err, action)
		}
		r.actions = append(r.actions, *action)

		promptValidation := r.coder.PromptValidation(plan, r.actions)
		if validationResult, err = r.model.YesOrNo(promptValidation, 3); err != nil {
			log.Printf("Validation error: %v", err)
		}

		log.Printf("Validation Result: %v, Action %v", validationResult, action)
		appendActionLog(logFile, r.coder.Task, action, validationResult)

		if validationResult {
			log.Printf("✅ Task completed successfully: %s", r.coder.Task)
			break
		}
	}
}
