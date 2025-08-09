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
		"You are an expert in Go unit testing. Your mission is to read each file and make the tests that cover all"+
			" public functions and relevant behaviors in this directory without modifying any existing logic files. "+
			"Use Go's native testing library and avoid unnecessary external frameworks. Leverage table-driven tests and "+
			"sub-tests where appropriate, ensure each file is formatted with go fmt, and do not introduce any compilation"+
			" warnings. Process by 1 file per once, first list all, then each by each read file,"+
			"write new file, read both and next",
		[]string{
			"Use table-driven tests with well-defined structs for inputs and expected outputs.",
			"Leverage sub-tests via t.Run for each scenario in your table-driven tests.",
			"Optionally use testify (assert/require) if already permitted or present in the project, but don't introduce new dependencies without necessity.",
			"Include descriptive test case names and clear error messages.",
		},
		[]string{
			"Do not overwrite or modify existing tests.",
			"Generate new `_test.go` files for each set of public functions or methods that need coverage.",
			"Each test must use the signature `func TestXxx(t *testing.T)` and provide clear error messages.",
			"Avoid external dependencies unless strictly necessary.",
		},
		[]string{
			"Follow Go style guidelines and apply go fmt before finishing.",
			"Use sub-tests (t.Run) and table-driven tests to organize your tests.",
		},
		5,
		"",
		true,
		false,
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
	return models.NewLMStudioClient(db, model, embModel)
}
