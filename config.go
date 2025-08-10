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
		"Golang",
		"You are a Golang engineer. Create a new uniquely named folder like `seed_<yyyy-mm_dd_hh_mm>` in the current directory and work only inside it. "+
			"Inside that folder, initialize a minimal Go module (e.g., `module example.com/seed`). "+
			"Build a small Golang application that demonstrates your skills by implementing three functions in a package (e.g., `utils`): "+
			"1) ReverseString(s string) string — reverse a string in a Unicode-safe way (use runes). "+
			"2) Factorial(n int) int — return 0 for n<0; otherwise compute n!. "+
			"3) IsPrime(n int) bool — return true only for prime n; for n<2 return false. "+
			"Write table-driven tests in `_test.go` with clear case names and `t.Run` subtests covering normal, edge, and error-like inputs. "+
			"Ensure the code compiles, is idiomatic, and formatted. "+
			"At the end, list all created files (paths) and provide a brief summary of what you built. "+
			"Do not modify anything outside the seed folder.",
		"file_basic", // toolPreset
		[]string{
			"Write table-driven tests with descriptive case names for clarity.",
			"Organize tests using `t.Run` subtests for each case.",
			"Rely only on Go's standard library; avoid external dependencies.",
			"Add clear, meaningful doc comments for all exported identifiers.",
			"Follow idiomatic Go naming conventions for packages, functions, and variables.",
			"Keep functions small, focused, and easy to read.",
			"Ensure all code is formatted using `go fmt` before completion.",
		},          // code style
		[]string{}, // accept conditions
		[]string{
			"Must be a Golang application.",
			"Do not touch files outside the new working folder.",
		}, // rules
		5,
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
	return models.NewLLMClient(db, model, embModel)
}
