package teams

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"GoWorkerAI/app/utils"
)

const (
	leaderKey       = "leader"
	eventHandlerKey = "event_handler"
)

type Team struct {
	Members map[string]*Member
	Task    *Task
	Audits  *utils.AuditLogger
}

func (t *Team) GetLeader() *Member {
	return t.Members[leaderKey]
}

func (t *Team) GetEventHandler() *Member {
	return t.Members[eventHandlerKey]
}

func (t *Team) GetMember(key string) *Member {
	return t.Members[key]
}

func (t *Team) Close() error {
	// Close the audit logger but DO NOT clear the logs
	// Logs must persist for debugging and task resumption
	if t.Audits != nil {
		return t.Audits.Close()
	}
	return nil
}

func (t *Team) GetMembersOptions() []string {
	if t == nil {
		return nil
	}
	options := make([]string, 0, len(t.Members))
	for _, member := range t.Members {
		if member.Key == leaderKey || member.Key == eventHandlerKey {
			continue
		}

		var whenToUseSection string
		if member.WhenCall != "" {
			whenToUseSection = fmt.Sprintf("\n  WHEN_TO_USE: %s", member.WhenCall)
		}

		toolsList := strings.Join(member.GetToolsOptions(), ", ")

		memberOption := fmt.Sprintf("WORKER_NAME: \"%s\"%s\n  AVAILABLE_TOOLS: [%s]",
			member.Key,
			whenToUseSection,
			toolsList,
		)

		options = append(options, memberOption)
	}
	return options
}

type Member struct {
	Key          string
	SystemPrompt string
	WhenCall     string
	Task         *Task
	Interface
}

func NewMember(key, systemPrompt, whenCall string, worker Interface) *Member {
	return &Member{
		Key:          key,
		SystemPrompt: systemPrompt,
		WhenCall:     whenCall,
		Interface:    worker,
	}
}

type Task struct {
	ID          uuid.UUID
	Description string
}

func (m *Member) SetTask(task *Task) {
	if m == nil {
		return
	}
	m.Task = task
	task.ID = uuid.New()
}

func (t Team) RemoveTeamMember(key string) {
	delete(t.Members, key)
}

func (t Team) AddTeamMember(member *Member) {
	t.Members[member.Key] = member
}

func (t Team) GetTeamMemberByKey(key string) *Member {
	m := t.Members[key]
	if len(m.Key) == 0 {
		return nil
	}
	return m
}

func NewTeam(members []*Member, task string) *Team {
	memberMap := make(map[string]*Member)
	for _, member := range members {
		memberMap[member.Key] = member
	}
	return &Team{
		Members: memberMap,
		Task: &Task{
			ID:          uuid.New(),
			Description: task,
		},
	}
}
