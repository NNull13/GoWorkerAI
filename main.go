package main

import (
	"fmt"
	"sync"

	"GoEngineerAI/app/models"
	"GoEngineerAI/app/runtime"
	"GoEngineerAI/app/workers"
)

func main() {
	var runtimes []*runtime.Runtime
	modelClient := models.NewLMStudioClient()
	coders := []workers.Coder{
		coderCustom,
	}

	var wg sync.WaitGroup

	for _, coder := range coders {
		runtimes = append(runtimes, runtime.NewRuntime(&coder, modelClient))
	}

	for _, r := range runtimes {
		wg.Add(1)
		go func(rt *runtime.Runtime) {
			defer wg.Done()
			rt.Run()
		}(r)
	}

	fmt.Println("All runtimes started. Running indefinitely...")

	wg.Wait()

}
