package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

const (
	cmdTimeout     = 120 * time.Second
	maxOutputBytes = 2 << 20
)

type CommandAction struct {
	Command string `json:"command"`
}

type CommandExecError struct {
	ExitCode int
	Output   string
}

var allowedCmds = map[string]map[string]struct{}{
	"go": {"test": {}, "fmt": {}, "mod": {}},
}

func (e *CommandExecError) Error() string {
	return fmt.Sprintf("command failed with exit code %d", e.ExitCode)
}

const ToolRunCommand = "run_command"

var commandDispatch = map[string]func(any) (string, error){
	ToolRunCommand: func(p any) (string, error) {
		return withParsed[CommandAction](p, ToolRunCommand, func(ca CommandAction) (string, error) {
			return runWhitelistedSimple(ca.Command)
		})
	},
}

func executeCommandAction(action ToolTask) (string, error) {
	h, ok := commandDispatch[action.Key]
	if !ok {
		log.Printf("❌ Unknown tool key: %s\n", action.Key)
		return "", fmt.Errorf("unknown tool key: %s", action.Key)
	}
	return h(action.Parameters)
}

func runWhitelistedSimple(cmdline string) (string, error) {
	cmdline = strings.TrimSpace(cmdline)
	if cmdline == "" {
		return "", errors.New("command is required")
	}
	if err := forbidShellMeta(cmdline); err != nil {
		return "", err
	}

	tokens := strings.Fields(cmdline)
	if len(tokens) == 0 {
		return "", errors.New("empty command")
	}
	exe := tokens[0]
	var sub string
	if len(tokens) > 1 {
		sub = tokens[1]
	}

	cmdAllowed, exist := allowedCmds[exe]
	if !exist {
		return "", fmt.Errorf("unknown command: %s", exe)
	}

	if _, exist = cmdAllowed[sub]; !exist {
		return "", fmt.Errorf("unknown command: %s", exe)
	}

	if _, err := exec.LookPath(exe); err != nil {
		return "", fmt.Errorf("%q not found in PATH", exe)
	}

	root, _ := getRoot()

	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, exe, tokens[1:]...)
	if root != "" {
		cmd.Dir = root
	}

	cw := &cappedWriter{max: maxOutputBytes}
	cmd.Stdout = cw
	cmd.Stderr = cw

	if err := cmd.Start(); err != nil {
		return "", err
	}
	waitErr := cmd.Wait()

	out := cw.String()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return out, fmt.Errorf("command timed out after %s", cmdTimeout)
	}
	if waitErr != nil {
		var ee *exec.ExitError
		if errors.As(waitErr, &ee) {
			return out, &CommandExecError{ExitCode: exitCodeFromError(ee), Output: out + "\n" + waitErr.Error()}
		}
		return out, waitErr
	}

	log.Printf("✅ %s executed successfully.\n", exe)
	return out, nil
}

func forbidShellMeta(s string) error {
	if strings.ContainsAny(s, `"'`+"\n\r`$()<>|;&") {
		return errors.New("shell metacharacters are not allowed")
	}
	if strings.Contains(s, "&&") || strings.Contains(s, "||") || strings.Contains(s, ";;") {
		return errors.New("shell operators are not allowed")
	}
	return nil
}

type cappedWriter struct {
	buf bytes.Buffer
	max int64
	n   int64
}

func (w *cappedWriter) Write(p []byte) (int, error) {
	remain := w.max - w.n
	if remain <= 0 {
		return len(p), nil
	}
	if int64(len(p)) > remain {
		p = p[:remain]
	}
	n, _ := w.buf.Write(p)
	w.n += int64(n)
	return len(p), nil
}

func (w *cappedWriter) String() string {
	return w.buf.String()
}

func exitCodeFromError(e *exec.ExitError) int {
	if st, ok := e.Sys().(interface{ ExitStatus() int }); ok {
		return st.ExitStatus()
	}
	return -1
}
