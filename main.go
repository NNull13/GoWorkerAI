package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"GoWorkerAI/app/runtime"
	"GoWorkerAI/app/tools"
	"GoWorkerAI/app/utils"
)

func main() {
	appCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	log.SetOutput(os.Stdout)
	db := getDB()
	model := getModel(db)
	clients := getClients()
	colors := utils.GetColors()
	var i int
	for _, m := range team.Members {
		toolsPreset := tools.NewToolkitFromPreset(m.GetToolsPreset())
		auditLogger, err := utils.NewWorkerLogger(fmt.Sprintf("worker_%d_%d", i, time.Now().Unix()),
			colors[i%len(colors)], 10000)
		if err != nil {
			log.Fatalf("failed to create logger for worker %d: %v", i+1, err)
		}
		m.Audits = auditLogger
		m.SetToolKit(toolsPreset)
		i++
	}

	r := runtime.NewRuntime(team, model, db)
	for _, client := range clients {
		client.Subscribe(r)
	}
	go r.Start(appCtx)

	log.Println("All runtimes started. Waiting for clients indefinitely...")

	// Wait for signal to exit
	<-appCtx.Done()
}
