package workers

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Coder struct {
	Worker
	Language   string
	CodeStyles []string
	Tests      bool
}

func NewCoder(
	language, task string,
	codeStyles, acceptConditions, rules []string,
	maxIterations int,
	folder string,
	tests, lockFolder bool,
) *Coder {
	return &Coder{
		Worker: Worker{
			Task: &Task{
				ID:               uuid.New(),
				Task:             task,
				AcceptConditions: acceptConditions,
				MaxIterations:    maxIterations,
			},
			Rules:      rules,
			LockFolder: lockFolder,
			Folder:     folder,
		},
		Language:   language,
		CodeStyles: codeStyles,
		Tests:      tests,
	}
}

func (c *Coder) TaskInformation() string {
	baseInfo := c.Worker.TaskInformation()
	var sb strings.Builder
	sb.WriteString(baseInfo)
	sb.WriteString(fmt.Sprintf("Programming Language: %s\n", c.Language))
	if len(c.CodeStyles) > 0 {
		sb.WriteString(fmt.Sprintf("Code Styles: %s\n", strings.Join(c.CodeStyles, ", ")))
	}
	sb.WriteString(fmt.Sprintf("Testing Required: %t\n", c.Tests))
	return sb.String()
}
