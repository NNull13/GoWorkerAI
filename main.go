package main

import (
	"context"
	"log"
	"os"
	"sync"

	"GoWorkerAI/app/runtime"
	"GoWorkerAI/app/tools"
)

func main() {
	log.SetOutput(os.Stdout)
	db := getDB()
	model := getModel(db)
	clients := getClients()
	var wg sync.WaitGroup
	for _, worker := range customWorkers {
		wg.Add(1)
		toolsPreset := tools.NewToolkitFromPreset(worker.GetToolsPreset())
		r := runtime.NewRuntime(worker, model, toolsPreset, db, worker.GetTask() != nil)
		for _, client := range clients {
			client.Subscribe(r)
		}
		go func() {
			defer wg.Done()
			r.Start(context.Background())
		}()
	}
	log.Println("All runtimes started. Waiting for clients indefinitely...")
	wg.Wait()
}
