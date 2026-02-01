package clients

import (
	"fmt"
	"log"
	"sync"

	"GoWorkerAI/app/runtime"
)

// Config defines the configuration for a client connector
type Config struct {
	Type    string            `yaml:"type" json:"type"`
	Enabled bool              `yaml:"enabled" json:"enabled"`
	Config  map[string]string `yaml:"config,omitempty" json:"config,omitempty"`
}

type Registry struct {
	mu      sync.RWMutex
	clients []Interface
}

func NewRegistry() *Registry {
	return &Registry{
		clients: make([]Interface, 0),
	}
}

func (r *Registry) Register(client Interface, rt *runtime.Runtime) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.clients = append(r.clients, client)
	client.Subscribe(rt)

	return nil
}

func (r *Registry) GetAll() []Interface {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Interface, len(r.clients))
	copy(result, r.clients)
	return result
}

func (r *Registry) CloseAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, client := range r.clients {
		if closer, ok := client.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				log.Printf("⚠️ Error closing client: %v\n", err)
			}
		}
	}
	r.clients = make([]Interface, 0)
}

func CreateClient(cfg Config) (Interface, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("client %s is disabled", cfg.Type)
	}

	switch cfg.Type {
	case "discord":
		return NewDiscordClientFromConfig(cfg.Config)
	// Add more client types here in the future:
	// case "slack":
	//     return NewSlackClient(cfg.Config)
	// case "telegram":
	//     return NewTelegramClient(cfg.Config)
	// case "http":
	//     return NewHTTPClient(cfg.Config)
	default:
		return nil, fmt.Errorf("unknown client type: %s", cfg.Type)
	}
}
