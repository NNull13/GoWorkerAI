package main

import (
	"GoWorkerAI/app/clients"
	"GoWorkerAI/app/workers"
)

var customWorkers = []workers.Interface{
	workers.NewCoder(
		"Go",
		"Implement a monolithic HTTP server in a single file.Develop a single-file Go HTTP server that handles multiple routes and logs all requests",
		[]string{},
		[]string{
			"Server must run on port 8080.",
			"Must support at least two routes (`/` and `/status`).",
			"Logs must include timestamp and request details.",
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
