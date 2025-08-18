package main

import (
	"context"
	"fmt"
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
	colors := []string{"\033[31m", "\033[32m", "\033[33m", "\033[34m", "\033[35m", "\033[36m"}
	for i, worker := range customWorkers {
		wg.Add(1)
		toolsPreset := tools.NewToolkitFromPreset(worker.GetToolsPreset())
		_, logger, err := runtime.NewWorkerLogger(fmt.Sprintf("worker_%d", i+1), colors[i%len(colors)], 10000)
		if err != nil {
			log.Fatalf("failed to create logger for worker %d: %v", i+1, err)
		}
		r := runtime.NewRuntime(worker, model, toolsPreset, db, worker.GetTask() != nil, logger)
		for _, client := range clients {
			client.Subscribe(r)
		}
		go func(rt *runtime.Runtime) {
			defer wg.Done()
			rt.Start(context.Background())
		}(r)
	}
	log.Println("All runtimes started. Waiting for clients indefinitely...")
	wg.Wait()
}
