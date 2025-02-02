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
		coderCustom,
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
