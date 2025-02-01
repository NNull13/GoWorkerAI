package main

import (
	"fmt"
	"sync"
)

func main() {
	var runtimes []Runtime
	modelClient := NewModelClient()
	coders := []Coder{
		NewCoder(
			"Go",
			"Implement a monolithic HTTP server in a single file.",
			"Develop a single-file Go HTTP server that handles multiple routes and logs all requests.",
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
	}

	var wg sync.WaitGroup

	for _, coder := range coders {
		runtimes = append(runtimes, Runtime{coder: coder, model: modelClient})
	}

	for _, r := range runtimes {
		wg.Add(1)
		go func(rt Runtime) {
			defer wg.Done()
			rt.run()
		}(r)
	}

	fmt.Println("All runtimes started. Running indefinitely...")

	wg.Wait()

}
