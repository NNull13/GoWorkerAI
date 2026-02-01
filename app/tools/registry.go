package tools

import (
	"fmt"
	"log"
	"sync"
)

var (
	globalRegistry = &toolRegistry{
		tools: make(map[string]Tool),
	}
)

type toolRegistry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func Register(tool Tool) error {
	return globalRegistry.register(tool)
}

func Unregister(name string) {
	globalRegistry.unregister(name)
}

func GetTool(name string) (Tool, bool) {
	return globalRegistry.get(name)
}

func AllRegisteredTools() map[string]Tool {
	return globalRegistry.all()
}

func (r *toolRegistry) register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tool.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if _, exists := r.tools[tool.Name]; exists {
		log.Printf("⚠️ Tool %s already registered, overwriting\n", tool.Name)
	}

	if tool.HandlerFunc == nil {
		log.Printf("tool %s has no handler", tool.Name)
	}

	r.tools[tool.Name] = tool
	return nil
}

func (r *toolRegistry) unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
}

func (r *toolRegistry) get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

func (r *toolRegistry) all() map[string]Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]Tool, len(r.tools))
	for k, v := range r.tools {
		result[k] = v
	}
	return result
}

func InitializeBuiltinTools() {
	for name, tool := range allTools {
		if err := Register(tool); err != nil {
			log.Printf("⚠️ Failed to register builtin tool %s: %v\n", name, err)
		}
	}
	log.Printf("✅ Registered %d builtin tools\n", len(allTools))
}
