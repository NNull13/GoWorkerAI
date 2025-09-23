package workers

import (
	"fmt"

	"GoWorkerAI/app/models"
	"GoWorkerAI/app/tools"
)

const maxSteps = 20

type Worker interface {
	Prompt(task, context string) []models.Message
	GetToolsOptions() []string
	GetToolsPreset() string
	AddTools(tool []tools.Tool)
	SetToolKit(tk map[string]tools.Tool)
	GetToolKit() map[string]tools.Tool
}

type Base struct {
	Prompt      string
	Rules       []string
	ToolsPreset string
	Toolkit     map[string]tools.Tool
}

func (w *Base) SetToolKit(tk map[string]tools.Tool) {
	if w != nil {
		w.Toolkit = tk
	}
}

func (w *Base) GetToolKit() map[string]tools.Tool {
	if w == nil {
		return nil
	}
	return w.Toolkit
}

func (w *Base) GetToolsOptions() []string {
	if w == nil {
		return nil
	}
	options := make([]string, len(w.Toolkit))
	for _, tool := range w.Toolkit {
		options = append(options, fmt.Sprintf("name: %s { description: %s } ", tool.Name, tool.Description))
	}
	return options
}

func (w *Base) GetToolsPreset() string {
	return w.ToolsPreset
}

func (w *Base) AddTools(list []tools.Tool) {
	if w == nil {
		return
	}
	if w.Toolkit == nil {
		w.Toolkit = map[string]tools.Tool{}
	}
	for _, tool := range list {
		w.Toolkit[tool.Name] = tool
	}
}
