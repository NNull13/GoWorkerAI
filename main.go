package main

import (
	"fmt"
	"sync"

	"GoEngineerAI/app"
)

func main() {
	var runtimes []*app.Runtime
	modelClient := app.NewModelClient()
	coders := []app.Coder{
		/*
			app.NewCoder(
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
				3,
				10,
				map[string]string{},
			),
		*/
		app.NewCoder(
			"Go",
			`Generate comprehensive unit and integration tests for my Go project. The project's entire code resides in the "app" folder. Your task is to review only the code within this folder (using the "list_files" functionality) and then generate complete test files (i.e., *_test.go) without modifying any existing code files or the go.mod file.
			Task: Execute Comprehensive Automated Testing  
			Objective: Validate the project's functionality with robust and exhaustive tests.
			
			Requirements:
			- Write tests that cover all critical code paths.
			- Follow best practices for unit and integration testing in Go.
			- Ensure robust error handling in test cases.
			- All tests must pass before deployment.
			- Continuous integration should automatically verify test results.
			- Test logs must include detailed debugging information.
			
			Technical Constraints:
			- Use Go's built-in testing package for unit tests.
			- Utilize httptest for testing HTTP handlers.
			- Implement table-driven tests and organize them using t.Run() with descriptive scenario names.
			- Use the testify package for assertions where appropriate.
			- Include proper test teardown and cleanup routines.
			- Integrate with CI/CD pipelines for automated testing.
			
			Important:
			- Do not modify the go.mod file or any existing code files—only create or update test files.
			
			Additional Guidance:
			- Ensure tests cover scenarios including successful execution, error cases, integration issues, and edge cases such as flaky tests.
			- The tests should be ready to run with 'go test'.
			
			Please generate complete test files that fully meet these requirements.`,
			[]string{
				"Tests are failing due to unexpected errors.",
				"Integration issues between modules are present.",
				"Incomplete test coverage.",
				"Flaky tests causing inconsistent results.",
			},
			[]string{
				"Write tests that cover all critical code paths.",
				"Follow best practices for unit and integration testing in Go.",
				"Implement robust error handling in test cases.",
			},
			[]string{
				"All tests must pass before deployment.",
				"Continuous integration should verify test results.",
				"Test logs must include detailed debugging information.",
			},
			[]string{
				"Utilize the testing package for unit tests.",
				"Use httptest for HTTP handler testing.",
				"Implement proper test teardown and cleanup routines.",
				"Do not modify go.mod.",
				"Review only files in the 'app' folder using the list_files feature.",
			},
			[]string{
				"Integrate with CI/CD pipelines for automated testing.",
				"Use testify for assertions and organize tests using table-driven scenarios.",
				"Leverage t.Run() with descriptive scenario names.",
			},
			true,
			5,
			map[string]string{},
		),
	}

	var wg sync.WaitGroup

	for _, coder := range coders {
		runtimes = append(runtimes, app.NewRuntime(coder, modelClient))
	}

	for _, r := range runtimes {
		wg.Add(1)
		go func(rt *app.Runtime) {
			defer wg.Done()
			rt.Run()
		}(r)
	}

	fmt.Println("All runtimes started. Running indefinitely...")

	wg.Wait()

}
