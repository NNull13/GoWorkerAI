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

var task = "Create a new minimal app with gin framework and a calculator service to resolve operations from a endpoint request from a string like `2 + (5 + 2 x 4)`  "

var members = []*teams.Member{
	{
		Key: "leader", //Reserved key
		Worker: &workers.Leader{
			Base: workers.Base{
				ToolsPreset: tools.PresetFileOpsBasic,
				Rules: []string{
					"Always avoid using commands that are not available in the tool kit.",
					"Never suggest using go commands, still not supported.",
				},
			},
		},
	},
	{
		Key:    "event_handler", //Reserved key
		Worker: &workers.EventHandler{},
	},
	teams.NewMember("coder", "This worker should be called every time is needed programming code.", &workers.Coder{
		Base: workers.Base{
			ToolsPreset: tools.PresetFileOpsBasic,
			Rules: []string{
				"You are an golang expert",
				"You should use gin framework for the web server",
				"You should use postgresql for the database",
				"Always avoid using commands that are not available in the tool kit. Discard as it was done",
				"Never use go commands, still not supported.",
				"Avoid partial updates on files, always try to write the entire file",
			},
		},
	}),
}
var team = teams.NewTeam(members, task)

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
