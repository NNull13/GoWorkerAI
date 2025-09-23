package teams

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"GoWorkerAI/app/teams/workers"
	"GoWorkerAI/app/utils"
)

const (
	leaderKey       = "leader"
	eventHandlerKey = "event_handler"
)

type Team struct {
	Members map[string]*Member
	Task    *Task
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

func (t *Team) Close() {
	for _, member := range t.Members {
		if member.Key != eventHandlerKey {
			member.Audits.ClearBuffer()
			member.Audits.ClearFile()
		}
	}
}

func (t *Team) GetMembersOptions() []string {
	if t == nil {
		return nil
	}
	options := make([]string, len(t.Members))
	for _, member := range t.Members {
		if len(member.WhenCall) > 0 {
			options = append(options, fmt.Sprintf("team member name: %s { when call : %s | tools : [ %v ] } ",
				member.Key, member.WhenCall, strings.Join(member.GetToolsOptions(), " | ")))
		}
	}
	return options
}

type Member struct {
	Key      string
	WhenCall string
	Task     *Task
	Audits   *utils.AuditLogger
	workers.Worker
}

func NewMember(key, whenCall string, worker workers.Worker) *Member {
	return &Member{
		Key:      key,
		WhenCall: whenCall,
		Worker:   worker,
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
