package configs

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"GoWorkerAI/app/clients"
	"GoWorkerAI/app/mcps"
	"GoWorkerAI/app/teams"
	"GoWorkerAI/app/tools"
)

type Config struct {
	Teams      map[string]TeamConfig `yaml:"teams"`
	Clients    []clients.Config      `yaml:"clients,omitempty"`
	GlobalMCPs []mcps.Config         `yaml:"global_mcps,omitempty"`
}

type TeamConfig struct {
	Task    string         `yaml:"task"`
	Members []MemberConfig `yaml:"members"`
}

type MemberConfig struct {
	Key         string        `yaml:"key"`
	System      string        `yaml:"system,omitempty"`
	WhenCall    string        `yaml:"when_call,omitempty"`
	ToolsPreset string        `yaml:"tools_preset,omitempty"`
	Rules       []string      `yaml:"rules,omitempty"`
	MCPs        []mcps.Config `yaml:"mcps,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read configs file: %w", err)
	}

	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	return &cfg, nil
}

func (mc MemberConfig) BuildWorker() (teams.Interface, error) {
	worker := &teams.Worker{
		ToolsPreset: mc.ToolsPreset,
		Rules:       mc.Rules,
		System:      mc.System,
	}

	switch mc.Key {
	case "leader":
		worker.ToolsPreset = tools.PresetDelegate
	case "reviewer":
		worker.ToolsPreset = tools.PresetApprover
	case "event_handler":
		worker.ToolsPreset = "" // No tools
	}
	return worker, nil
}

func (c *Config) Validate() error {
	if len(c.Teams) == 0 {
		return fmt.Errorf("no teams defined in configs")
	}

	for teamName, teamCfg := range c.Teams {
		if err := teamCfg.Validate(); err != nil {
			return fmt.Errorf("team %s: %w", teamName, err)
		}
	}

	return nil
}

func (tc TeamConfig) Validate() error {
	if tc.Task == "" {
		return fmt.Errorf("task cannot be empty")
	}

	if len(tc.Members) == 0 {
		return fmt.Errorf("no members defined")
	}

	hasLeader := false
	for _, member := range tc.Members {
		if member.Key == "leader" {
			hasLeader = true
			break
		}
	}

	if !hasLeader {
		return fmt.Errorf("team must have a 'leader' member")
	}

	return nil
}
