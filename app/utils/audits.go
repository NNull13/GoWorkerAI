package utils

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

const (
	colorReset = "\033[0m"
)

type AuditLogger struct {
	*log.Logger
	file    *os.File
	mu      sync.RWMutex
	buf     []string
	cap     int
	start   int
	size    int
	lineBuf bytes.Buffer
}

type colorWriter struct {
	w     io.Writer
	color string
}

func newLogger(audit *AuditLogger, worker, color string) (*log.Logger, *os.File, error) {
	if err := os.MkdirAll("logs", 0o755); err != nil {
		return nil, nil, err
	}
	file, err := os.OpenFile(filepath.Join("logs", fmt.Sprintf("%s.log", worker)),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, err
	}
	cw := colorWriter{w: os.Stdout, color: color}
	mw := io.MultiWriter(cw, file, audit)
	logger := log.New(mw, fmt.Sprintf("[%s] ", worker), log.LstdFlags)
	return logger, file, nil
}

func NewWorkerLogger(worker, color string, capacity int) (*AuditLogger, error) {
	if capacity <= 0 {
		capacity = 1
	}
	audit := &AuditLogger{
		buf: make([]string, capacity),
		cap: capacity,
	}
	logger, file, err := newLogger(audit, worker, color)
	if err != nil {
		return nil, err
	}
	audit.Logger = logger
	audit.file = file
	return audit, nil
}

func (a *AuditLogger) Close() error {
	if a.file != nil {
		return a.file.Close()
	}
	return nil
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

func (a *AuditLogger) Printf(text string, v ...any) {
	text = fmt.Sprintf(text, v...)
	a.Write([]byte(text))
	a.Logger.Print(text)
}

func (a *AuditLogger) Print(v ...any) {
	text := fmt.Sprint(v...)
	a.Write([]byte(text))
	a.Logger.Print(text)
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

func (a *AuditLogger) ClearBuffer() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.start = 0
	a.size = 0
	a.buf = make([]string, a.cap)
	a.lineBuf.Reset()
	a.ClearFile()
}

func (a *AuditLogger) ClearFile() error {
	if a.file == nil {
		return nil
	}
	// Truncar a 0
	return a.file.Truncate(0)
}
