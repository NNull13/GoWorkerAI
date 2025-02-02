package runtime

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"GoEngineerAI/app/actions"
	"GoEngineerAI/app/models"
	"GoEngineerAI/app/workers"
)

type Runtime struct {
	worker  workers.Interface
	model   models.Interface
	actions []models.Action
}

func NewRuntime(worker workers.Interface, model models.Interface) *Runtime {
	return &Runtime{
		worker: worker,
		model:  model,
	}
}

func (r *Runtime) Run() {
	promptPlan := r.worker.PromptPlan()
	plan, err := r.model.Think(promptPlan)
	if err != nil {
		log.Panicf("Error generating plan: %v\n", err)
	}

	runtimeFolder := r.worker.GetFolder()
	if r.worker.GetLockFolder() {
		folderName := r.worker.GetFolder() + time.Now().Format("20060102_150405")
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
	AppendLLMLog(logFile, plan)

	var validationResult bool
	for i := 0; i <= r.worker.GetMaxIterations(); i++ {
		var action *models.Action
		promptWorker := r.worker.PromptNextAction(plan, r.actions)
		if action, err = r.model.Process(promptWorker); err != nil {
			log.Printf("Error processing action: %s | Action: %v", err, action)
			continue
		}

		log.Printf("Processing action: %s | Action: %v", action.Action, action)
		action.Result, err = actions.ExecuteAction(action, runtimeFolder)
		if err != nil {
			log.Panicf("Error executing action: %s | Action: %v", err, action)
		}
		r.actions = append(r.actions, *action)

		promptValidation := r.worker.PromptValidation(plan, r.actions)
		if validationResult, err = r.model.YesOrNo(promptValidation, 3); err != nil {
			log.Printf("Validation error: %v", err)
		}

		log.Printf("Validation Result: %v, Action %v", validationResult, action)
		AppendActionLog(logFile, r.worker.GetTask(), action, validationResult)

		if validationResult {
			log.Printf("✅ Task completed successfully: %s", r.worker.GetTask())
			break
		}
	}
}
