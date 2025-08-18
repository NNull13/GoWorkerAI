package runtime

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type AuditLogger struct {
	*log.Logger
	mu      sync.RWMutex
	buf     []string
	cap     int
	start   int
	size    int
	lineBuf bytes.Buffer
}

const (
	colorReset = "\033[0m"
)

type colorWriter struct {
	w     io.Writer
	color string
}

func NewWorkerLogger(worker, color string, capacity int) (*AuditLogger, error) {
	if capacity <= 0 {
		capacity = 1
	}
	audit := &AuditLogger{
		buf: make([]string, capacity),
		cap: capacity,
	}

	if err := os.MkdirAll("logs", 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(filepath.Join("logs", fmt.Sprintf("%s.log", worker)), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	cw := colorWriter{w: os.Stdout, color: color}
	mw := io.MultiWriter(cw, file, audit)
	logger := log.New(mw, fmt.Sprintf("[%s] ", worker), log.LstdFlags)
	audit.Logger = logger
	return audit, nil
}

func (cw colorWriter) Write(p []byte) (int, error) {
	if cw.color == "" {
		return cw.w.Write(p)
	}
	colored := append([]byte(cw.color), p...)
	colored = append(colored, []byte(colorReset)...)
	return cw.w.Write(colored)
}

func (a *AuditLogger) Write(p []byte) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	n, _ := a.lineBuf.Write(p)
	for {
		b := a.lineBuf.Bytes()
		idx := bytes.IndexByte(b, '\n')
		if idx < 0 {
			break
		}
		line := string(b[:idx])
		a.lineBuf.Next(idx + 1)
		a.push(line)
	}
	return n, nil
}

func (a *AuditLogger) push(s string) {
	if a.size < a.cap {
		pos := (a.start + a.size) % a.cap
		a.buf[pos] = s
		a.size++
		return
	}
	a.buf[a.start] = s
	a.start = (a.start + 1) % a.cap
}

func (a *AuditLogger) GetLastLogs(n int) []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if n > a.size {
		n = a.size
	}
	out := make([]string, 0, n)
	for i := a.size - n; i < a.size; i++ {
		pos := (a.start + i) % a.cap
		out = append(out, a.buf[pos])
	}
	return out
}
