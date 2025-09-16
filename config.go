package main

import (
	"os"

	"GoWorkerAI/app/clients"
	"GoWorkerAI/app/models"
	"GoWorkerAI/app/storage"
	"GoWorkerAI/app/tools"
	"GoWorkerAI/app/workers"
)

const (
	gptModel       = "openai/gpt-oss-20b"
	embeddingModel = "text-embedding-nomic-embed-text-v1.5@q8_0"
	//  ...
)

var customWorkers = []workers.Interface{
	workers.NewCoder(
		"Golang",
		//"You are a Golang engineer. Build a small Golang application that demonstrates your skills by implementing three functions in a package (e.g., `utils`): ",
		"",
		tools.PresetFileOpsBasic, // toolPreset
		[]string{
			"Write the most clean and efficient code.",
			"Use Go's best practices and idiomatic code.",
			"Use idiomatic Go naming conventions.",
			"Write table-driven tests with descriptive case names for clarity.",
			"Organize tests using `t.Run` subtests for each case.",
			"Rely only on Go's standard library; avoid external dependencies.",
			"Add clear, meaningful doc comments for all exported identifiers.",
			"Follow idiomatic Go naming conventions for packages, functions, and variables.",
			"Keep functions small, focused, and easy to read.",
			"Ensure the code compiles, is idiomatic, and formatted. ",
		}, //rules
		5,
	),
}

func getClients() []clients.Interface {
	return []clients.Interface{clients.NewDiscordClient()}
}

func getDB() storage.Interface {
	return storage.NewSQLiteStorage()
}

func getModel(db storage.Interface) models.Interface {
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = gptModel
	}
	embModel := os.Getenv("LLM_EMBEDDINGS_MODEL")
	if embModel == "" {
		embModel = embeddingModel
	}
	return models.NewLLMClient(db, model, embModel)
}
