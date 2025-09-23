package workers

import (
	"strings"

	"GoWorkerAI/app/models"
)

var _ Worker = &Coder{}

type Coder struct {
	Base
}

func (c Coder) Prompt(task, context string) []models.Message {
	sys := "You are an expert software engineer. Your mission is to complete the given task.\n" +
		"RULES:" +
		strings.Join(c.Rules, "\n") +
		"CONTEXT:" + context

	return []models.Message{
		{Role: models.SystemRole, Content: sys},
		{Role: models.UserRole, Content: task},
	}
}
