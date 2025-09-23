package workers

import (
	"strings"

	"GoWorkerAI/app/models"
)

var _ Worker = &FileManager{}

type FileManager struct {
	Base
}

func (fm FileManager) Prompt(task, context string) []models.Message {
	sys := "You are the fime manager of the team. Your mission is to handle operations on the files of the system.\n" +
		"RULES:" +
		strings.Join(fm.Rules, "\n") +
		"CONTEXT:" + context

	return []models.Message{
		{Role: models.SystemRole, Content: sys},
		{Role: models.UserRole, Content: task},
	}
}
