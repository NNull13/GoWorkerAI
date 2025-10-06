package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

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
	for _, m := range team.Members {
		toolsPreset := tools.NewToolkitFromPreset(m.GetToolsPreset())
		m.SetToolKit(toolsPreset)
	}

	auditLogger, err := utils.NewWorkerLogger("team_logs", colors[0], 10000)
	if err != nil {
		log.Fatalf("failed to create logger for the team: %v", err)
	}
	team.Audits = auditLogger

	r := runtime.NewRuntime(team, model, db)
	for _, client := range clients {
		client.Subscribe(r)
	}
	go r.Start(appCtx)

	log.Println("All runtimes started. Waiting for clients indefinitely...")

	// Wait for signal to exit
	<-appCtx.Done()
}
