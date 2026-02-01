package mcps

import (
	"context"
	"fmt"
	"log"
	"sync"

	"GoWorkerAI/app/tools"
)

type Registry struct {
	mu      sync.RWMutex
	servers map[string]*Client
}

func NewRegistry() *Registry {
	return &Registry{
		servers: make(map[string]*Client),
	}
}

func (r *Registry) Start(ctx context.Context, cfg Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.servers[cfg.Name]; exists {
		log.Printf("⚠️ MCP server %s already running, skipping\n", cfg.Name)
		return nil
	}

	client, err := NewClient(ctx, cfg)
	if err != nil {
		return fmt.Errorf("start MCP %s: %w", cfg.Name, err)
	}

	r.servers[cfg.Name] = client

	for name, tool := range client.Tools() {
		if err = tools.Register(tool); err != nil {
			log.Printf("⚠️ Failed to register tool %s: %v\n", name, err)
		}
	}

	return nil
}

func (r *Registry) Stop(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	client, ok := r.servers[name]
	if !ok {
		return fmt.Errorf("MCP server %s not found", name)
	}

	// Unregister tools
	for toolName := range client.Tools() {
		tools.Unregister(toolName)
	}

	delete(r.servers, name)
	return client.Close()
}

func (r *Registry) Get(name string) (*Client, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	client, ok := r.servers[name]
	return client, ok
}

func (r *Registry) GetAllTools() map[string]tools.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]tools.Tool)
	for _, client := range r.servers {
		for name, tool := range client.Tools() {
			result[name] = tool
		}
	}
	return result
}

func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.servers))
	for name := range r.servers {
		names = append(names, name)
	}
	return names
}

func (r *Registry) CloseAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for name, client := range r.servers {
		if err := client.Close(); err != nil {
			log.Printf("⚠️ Error closing MCP %s: %v\n", name, err)
		}
	}
	r.servers = make(map[string]*Client)
}
