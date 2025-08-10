package runtime

import (
	"bytes"
	"sync"
)

var AuditInstance *AuditLogger

type AuditLogger struct {
	mu      sync.RWMutex
	buf     []string
	cap     int
	start   int
	size    int
	lineBuf bytes.Buffer
}

func NewAuditLogger(capacity int) *AuditLogger {
	if capacity <= 0 {
		capacity = 1
	}
	return &AuditLogger{
		buf: make([]string, capacity),
		cap: capacity,
	}
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

func init() {
	//AuditInstance = NewAuditLogger(10000)
	//log.SetOutput(io.MultiWriter(os.Stderr, AuditInstance))
	//log.SetFlags(log.LstdFlags)
}
