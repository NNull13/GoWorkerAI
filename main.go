package main

import (
	"log"

	"GoWorkerAI/app/actions"
	"GoWorkerAI/app/models"
	"GoWorkerAI/app/runtime"
)

func main() {
	modelClient := models.NewLMStudioClient()
	for _, worker := range customWorkers {
		r := runtime.NewRuntime(worker, modelClient, actions.WorkerActions, worker.GetTask() != nil)
		for _, client := range customClients {
			client.Subscribe(r)
		}
		go r.Start()
	}
	log.Println("All runtimes started. Waiting for clients indefinitely...")
	select {}
}
