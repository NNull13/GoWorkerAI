package main

import (
	"log"
	"sync"

	"GoWorkerAI/app/actions"
	"GoWorkerAI/app/runtime"
)

func main() {
	var wg sync.WaitGroup
	for _, worker := range customWorkers {
		wg.Add(1)
		r := runtime.NewRuntime(worker, modelClient, actions.WorkerActions, db, worker.GetTask() != nil)
		for _, client := range customClients {
			client.Subscribe(r)
		}
		go func() {
			defer wg.Done()
			r.Start()
		}()
	}
	log.Println("All runtimes started. Waiting for clients indefinitely...")
	wg.Wait()
}
