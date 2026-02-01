package teams

import (
	"fmt"
	"strings"

	"GoWorkerAI/app/tools"
)

type Interface interface {
	Prompt(context string) string
	GetToolsOptions() []string
	GetToolsPreset() string
	AddTools(tool []tools.Tool)
	SetToolKit(tk map[string]tools.Tool)
	GetToolKit() map[string]tools.Tool
}

type Worker struct {
	System      string
	Rules       []string
	ToolsPreset string
	Toolkit     map[string]tools.Tool
}

func (w *Worker) SetToolKit(tk map[string]tools.Tool) {
	if w != nil {
		w.Toolkit = tk
	}
}

func (w *Worker) GetToolKit() map[string]tools.Tool {
	if w == nil {
		return nil
	}
	return w.Toolkit
}

func (w *Worker) GetToolsOptions() []string {
	if w == nil {
		return nil
	}
	options := make([]string, len(w.Toolkit))
	for _, tool := range w.Toolkit {
		options = append(options, fmt.Sprintf("name: %s { description: %s } ", tool.Name, tool.Description))
	}
	return options
}

func (w *Worker) GetToolsPreset() string {
	return w.ToolsPreset
}

func (w *Worker) AddTools(list []tools.Tool) {
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

func (w *Worker) Prompt(context string) string {
	var sys string
	if w.Rules != nil {
		sys = w.System + "\nRULES:" + strings.Join(w.Rules, "\n")
		sys += "\n\nMOST IMPORTANT RULE: If a tool returns status=success or done=true, do not call any tool again for the same task. Respond to the user."
	}
	if context != "" {
		sys = sys + "CONTEXT:" + context
	}

	return sys
}
