package configs

import (
	"context"
	"fmt"
	"log"

	"GoWorkerAI/app/clients"
	"GoWorkerAI/app/mcps"
	"GoWorkerAI/app/runtime"
	"GoWorkerAI/app/teams"
)

func (tc TeamConfig) BuildTeam(ctx context.Context, mcpRegistry *mcps.Registry) (*teams.Team, error) {
	var members []*teams.Member

	for _, mc := range tc.Members {
		worker, err := mc.BuildWorker()
		if err != nil {
			return nil, fmt.Errorf("build worker %s: %w", mc.Key, err)
		}

		for _, mcpCfg := range mc.MCPs {
			log.Printf("🔧 Starting MCP '%s' for worker '%s'\n", mcpCfg.Name, mc.Key)
			if err = mcpRegistry.Start(ctx, mcpCfg); err != nil {
				return nil, fmt.Errorf("start MCP %s for worker %s: %w", mcpCfg.Name, mc.Key, err)
			}
		}

		member := teams.NewMember(mc.Key, mc.System, mc.WhenCall, worker)
		members = append(members, member)
	}

	return teams.NewTeam(members, tc.Task), nil
}

func (c *Config) BuildTeamByName(ctx context.Context, teamName string, mcpRegistry *mcps.Registry) (*teams.Team, error) {
	teamCfg, ok := c.Teams[teamName]
	if !ok {
		return nil, fmt.Errorf("team %s not found in configs", teamName)
	}

	return teamCfg.BuildTeam(ctx, mcpRegistry)
}

func (c *Config) StartGlobalMCPs(ctx context.Context, mcpRegistry *mcps.Registry) error {
	for _, mcpCfg := range c.GlobalMCPs {
		log.Printf("🌍 Starting global MCP '%s'\n", mcpCfg.Name)
		if err := mcpRegistry.Start(ctx, mcpCfg); err != nil {
			return fmt.Errorf("start global MCP %s: %w", mcpCfg.Name, err)
		}
	}
	return nil
}

func (c *Config) InitializeClients(clientRegistry *clients.Registry, rt *runtime.Runtime) error {
	if len(c.Clients) == 0 {
		log.Println("ℹ️ No clients configured")
		return nil
	}

	for _, clientCfg := range c.Clients {
		if !clientCfg.Enabled {
			log.Printf("⏭️ Client %s is disabled, skipping\n", clientCfg.Type)
			continue
		}

		log.Printf("🔌 Initializing %s client...\n", clientCfg.Type)
		client, err := clients.CreateClient(clientCfg)
		if err != nil {
			return fmt.Errorf("failed to create %s client: %w", clientCfg.Type, err)
		}

		if err := clientRegistry.Register(client, rt); err != nil {
			return fmt.Errorf("failed to register %s client: %w", clientCfg.Type, err)
		}

		log.Printf("✅ %s client initialized\n", clientCfg.Type)
	}

	return nil
}
