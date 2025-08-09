package main

import (
	"os"

	"GoWorkerAI/app/clients"
	"GoWorkerAI/app/models"
	"GoWorkerAI/app/storage"
	"GoWorkerAI/app/workers"
)

const (
	gptModel       = "openai/gpt-oss-20b"
	embeddingModel = "text-embedding-nomic-embed-text-v1.5@q8_0"
	//  ...
)

var customWorkers = []workers.Interface{
	workers.NewCoder(
		"Go",
		"You are a Go engineer. Your mission is to create a brand-new workspace **in a new folder** and work **only inside it**.\n\n"+
			"Steps:\n"+
			"1) Create a unique subfolder `seed_<yyyy-mm-dd_hhmmss>` in the current directory and operate ONLY there.\n"+
			"2) Initialize a minimal Go module (e.g., `module example.com/seed`).\n"+
			"3) Create a small package (e.g., `calc`) with 1–2 public functions (e.g., Add, Avg) and documentation comments.\n"+
			"4) Write table-driven tests in `_test.go` covering normal, edge, and error paths.\n"+
			"5) Run formatting and basic checks (`go fmt ./...` conceptually; ensure files are formatted). Do not introduce dependencies.\n"+
			"6) List the resulting files and summarize what was created.\n\n"+
			"Constraints:\n"+
			"- Never modify files outside the seed folder.\n"+
			"- Never overwrite existing tests; only create new ones if not present.\n"+
			"- Keep the code compilable and idiomatic Go.\n",
		"file_basic", // toolPreset
		[]string{
			"Use table-driven tests with clear case names.",
			"Use `t.Run` subtests per scenario.",
			"Prefer pure stdlib; no external deps.",
			"Ensure package-level doc comments for exported symbols.",
		},
		[]string{
			"Do not touch files outside the new seed folder.",
			"Do not modify existing non-test logic files if any are present.",
			"Do not add external dependencies.",
		},
		[]string{
			"Apply go fmt style before finishing.",
			"Keep functions small and testable.",
		},
		5,
		"",
		true,
		false,
	),
}

func getClients() []clients.Interface {
	return nil //[]clients.Interface{clients.NewDiscordClient()}
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
	return models.NewLMStudioClient(db, model, embModel)
}
