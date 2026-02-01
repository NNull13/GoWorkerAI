package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"GoWorkerAI/app/clients"
	"GoWorkerAI/app/configs"
	"GoWorkerAI/app/mcps"
	"GoWorkerAI/app/models"
	"GoWorkerAI/app/rag"
	"GoWorkerAI/app/runtime"
	"GoWorkerAI/app/storage"
	"GoWorkerAI/app/tools"
	"GoWorkerAI/app/utils"
)

func main() {
	appCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	log.SetOutput(os.Stdout)

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config.yaml"
	}

	log.Printf("📋 Loading configuration from: %s\n", configPath)
	cfg, err := configs.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("❌ Failed to load configs: %v", err)
	}

	if err = cfg.Validate(); err != nil {
		log.Fatalf("❌ Invalid configs: %v", err)
	}

	tools.InitializeBuiltinTools()

	mcpRegistry := mcps.NewRegistry()
	defer mcpRegistry.CloseAll()

	if err = cfg.StartGlobalMCPs(appCtx, mcpRegistry); err != nil {
		log.Fatalf("❌ Failed to start global MCPs: %v", err)
	}

	teamName := os.Getenv("TEAM_NAME")
	if teamName == "" {
		teamName = "default"
	}

	log.Printf("🏗️ Building team: %s\n", teamName)
	team, err := cfg.BuildTeamByName(appCtx, teamName, mcpRegistry)
	if err != nil {
		log.Fatalf("❌ Failed to build team: %v", err)
	}

	db := getDB()
	model := getModel(db)
	colors := utils.GetColors()

	for _, m := range team.Members {
		toolsPreset := tools.NewToolkitFromPreset(m.GetToolsPreset())

		for _, tool := range tools.AllRegisteredTools() {
			if _, exists := toolsPreset[tool.Name]; !exists {
				toolsPreset[tool.Name] = tool
			}
		}

		m.SetToolKit(toolsPreset)
	}

	auditLogger, err := utils.NewWorkerLogger("team_logs_"+time.Now().Format("20060102_150405"), colors[0], 10000)
	if err != nil {
		log.Fatalf("❌ Failed to create logger for the team: %v", err)
	}
	team.Audits = auditLogger

	ragClient := rag.NewClient(model)
	if err = ragClient.InitContext(appCtx); err != nil {
		log.Fatalf("❌ Failed to init rag: %v", err)
	}

	r := runtime.NewRuntime(team, model, db, ragClient)

	clientRegistry := clients.NewRegistry()
	defer clientRegistry.CloseAll()

	if err = cfg.InitializeClients(clientRegistry, r); err != nil {
		log.Fatalf("❌ Failed to initialize clients: %v", err)
	}

	go r.Start(appCtx)

	log.Println("✅ All systems started. Press Ctrl+C to exit...")
	log.Printf("📊 Active MCPs: %v\n", mcpRegistry.List())
	log.Printf("🔧 Total tools available: %d\n", len(tools.AllRegisteredTools()))
	log.Printf("🔌 Active clients: %d\n", len(clientRegistry.GetAll()))

	<-appCtx.Done()
	log.Println("🛑 Shutting down gracefully...")
}

func getDB() storage.Interface {
	return storage.NewSQLiteStorage()
}

func getModel(db storage.Interface) models.Interface {
	const (
		defaultModel          = "openai/gpt-oss-20b"
		defaultEmbeddingModel = "text-embedding-qwen3-embedding-4b"
	)

	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = defaultModel
	}

	embModel := os.Getenv("LLM_EMBEDDINGS_MODEL")
	if embModel == "" {
		embModel = defaultEmbeddingModel
	}

	return models.NewLLMClient(db, model, embModel)
}
