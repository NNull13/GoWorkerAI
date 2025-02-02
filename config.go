package main

import (
	"GoWorkerAI/app/clients"
	"GoWorkerAI/app/workers"
)

var customWorkers = []workers.Interface{
	workers.NewCoder(
		"Go",
		`Your task is to review the code in root and then generate complete test files (i.e., *_test.go) without modifying any existing code files.
		List files of root and read each for each to create the test go file covering all the code to test the functionality.`,
		[]string{},
		[]string{
			"List all the files and validate the project's functionality with robust and exhaustive tests",
		},
		[]string{},
		[]string{},
		true,
		33,
		"",
		false,
	),
}

var customClients = []clients.Interface{
	clients.NewDiscordClient(),
}
