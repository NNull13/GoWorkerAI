package main

import (
	"os"

	"GoWorkerAI/app/clients"
	"GoWorkerAI/app/models"
	"GoWorkerAI/app/storage"
	"GoWorkerAI/app/teams"
	"GoWorkerAI/app/teams/workers"
	"GoWorkerAI/app/tools"
)

const (
	gptModel       = "openai/gpt-oss-20b"
	embeddingModel = "text-embedding-nomic-embed-text-v1.5@q8_0"
	//  ...
)

var task = "Create a new folder with a main.go file that implements a function with a method named HelloWorld and the function should print 'Hello World' 13 times to the console. Also you should create the main_test.go "

var members = []*teams.Member{
	{
		Key: "leader", //Reserved key
		Worker: &workers.Leader{
			Base: workers.Base{
				ToolsPreset: tools.PresetDelegate,
				Rules:       []string{},
			},
		},
	},
	{
		Key: "event_handler", //Reserved key
		Worker: &workers.EventHandler{
			Base: workers.Base{
				ToolsPreset: tools.PresetDelegate,
				Rules:       []string{},
			},
		},
	},
	teams.NewMember("coder", "This worker should be called every time is needed programming code.", &workers.Coder{
		Base: workers.Base{
			Rules: []string{
				"Write the most clean and efficient code.",
				"Use Go's best practices and idiomatic code.",
				"Use idiomatic Go naming conventions.",
				"Write table-driven tests with descriptive case names for clarity.",
				"Organize tests using `t.Run` subtests for each case.",
				"Rely only on Go's standard library; avoid external dependencies.",
				"Add clear, meaningful doc comments for all exported identifiers.",
				"Follow idiomatic Go naming conventions for packages, functions, and variables.",
				"Keep functions small, focused, and easy to read.",
				"Ensure the code compiles, is idiomatic, and formatted. "},
		},
	}),
	teams.NewMember("file_manager", "This worker should be called when is necessary to work with the local files", &workers.FileManager{
		Base: workers.Base{
			ToolsPreset: tools.PresetFileOpsBasic,
			Rules: []string{
				"Never delete or override system files",
			},
		},
	}),
}
var team = teams.NewTeam(members, task)

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
