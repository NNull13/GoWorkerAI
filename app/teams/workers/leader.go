package workers

import (
	"strings"

	"GoWorkerAI/app/models"
)

var _ Worker = &Leader{}

type Leader struct {
	Base
}

func (l Leader) Prompt(task, context string) []models.Message {
	sys := "You are an high tech team leader. Your mission is to complete the given task with the help of your team members.\n" +
		"You should delegate tasks to your team members and monitor their progress.\n" +
		"You are the responsible of the team and you should make sure that the team is working as expected.\n" +
		"RULES:" +
		strings.Join(l.Rules, "\n") +
		"CONTEXT:" + context

	return []models.Message{
		{Role: models.SystemRole, Content: sys},
		{Role: models.UserRole, Content: task},
	}
}
