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
			"I need you to generate both unit and integration tests for my Go project.First of all get the list of files in app folder, Must review only the code in the folder use list_files in app .Never update code files, only create and edit test files Please ensure the tests meet the following requirements and constraints:\n\nTask: Execute Comprehensive Automated Testing\nObjective: Validate the project's functionality through robust unit and integration tests.\n\nRequirements:\n\nWrite tests that cover all critical code paths.\nFollow best practices for unit and integration testing in Go.\nEnsure proper error handling in test cases.\nAll tests must pass before deployment.\nContinuous integration should verify the test results.\nTest logs must include relevant debugging information.\nTechnical Constraints:\n\nUse the Go testing package for unit tests.\nUtilize httptest for HTTP handler tests.\nImplement table-driven tests and organize them using t.Run() with descriptive scenario names.\nUse the testify package for assertions when appropriate.\nInclude proper test teardown and cleanup routines.\nIntegrate with CI/CD pipelines for automated testing.\nImportant: Do not modify the go.mod file or any existing code files—only create or modify test files (i.e., *_test.go).\nAdditional Guidance:\n\nIf the code involves external dependencies (such as databases or external APIs), simulate these dependencies using mocks (for example, with gomock).\nEnsure the tests cover scenarios including successful execution, error cases, integration issues, and edge cases like flaky tests.\nPlease generate complete test files that are ready to run with go test.",
			[]string{
				"Tests failing due to unexpected errors.",
				"Integration issues between modules.",
				"Incomplete test coverage.",
				"Flaky tests causing inconsistent results.",
			},
			[]string{
				"Write tests that cover all critical code paths.",
				"Follow best practices for unit and integration testing in Go.",
				"Ensure proper error handling in test cases.",
			},
			[]string{
				"All tests must pass before deployment.",
				"Continuous integration should verify test results.",
				"Test logs must include relevant debugging information.",
			},
			[]string{
				"Use the `testing` package for unit tests.",
				"Utilize `httptest` for HTTP handler tests.",
				"Implement test teardown and cleanup routines.",
				"Never update go.mod",
				"First of all get the list of files in app folder",
			},
			[]string{
				"Integrate with CI/CD pipelines for automated testing.",
				"Use testify and test scenarios",
				"Table driven tests",
				"r.Run() using name in scenarios",
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
