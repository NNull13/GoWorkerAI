package main

import (
	"GoEngineerAI/app/workers"
)

var coderSample = workers.NewCoder(
	"Go",
	"Implement a monolithic HTTP server in a single file.Develop a single-file Go HTTP server that handles multiple routes and logs all requests",
	[]string{
		"Server crashes on invalid requests.",
		"Routes not handled correctly.",
		"Logs not formatted properly.",
		"Server does not close properly on shutdown.",
	},
	[]string{
		"Keep all functionality in a single Go file.",
		"Use idiomatic Go practices.",
		"Handle errors properly to prevent crashes.",
	},
	[]string{
		"Server must run on port 8080.",
		"Must support at least two routes (`/` and `/status`).",
		"Logs must include timestamp and request details.",
	},
	[]string{
		"Use the `net/http` package for HTTP handling.",
		"Implement structured logging using `log` package.",
		"Ensure the server shuts down properly using `context` package.",
	},
	[]string{"Use `httptest` package for testing the `/status` route."},
	true,
	10,
	"",
	false,
)

var coderCustom = workers.NewCoder(
	"Go",
	`Your task is to review the code in root and then generate complete test files (i.e., *_test.go) without modifying any existing code files.
	List files of root and read each for each to create the test go file covering all the code to test the functionality
	Objective: List all the files and validate the project's functionality with robust and exhaustive tests.`,
	[]string{},
	[]string{},
	[]string{},
	[]string{},
	[]string{},
	true,
	33,
	"",
	false,
)
