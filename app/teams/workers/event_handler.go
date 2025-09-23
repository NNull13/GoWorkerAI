package workers

import (
	"strings"

	"GoWorkerAI/app/models"
)

var _ Worker = &Coder{}

type EventHandler struct {
	Base
}

func (eh EventHandler) Prompt(task, context string) []models.Message {
	sys := "You are the event hanlder assistant of the team. Your mission is to handle the events and evaluate if is needed" +
		"to create a new task on the backlog of the team or if is possible to quick solve and/or answer to mitigate the situation " +
		"also you should quick scape from situations that are not part of our mission as a team.\n" +
		"RULES:" +
		strings.Join(eh.Rules, "\n") +
		"CONTEXT:" + context

	return []models.Message{
		{Role: models.SystemRole, Content: sys},
		{Role: models.UserRole, Content: task},
	}
}
