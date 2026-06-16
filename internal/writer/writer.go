package writer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shiyisir/claude-proxy/internal/logging"
	"github.com/shiyisir/claude-proxy/internal/parser"
	"github.com/shiyisir/claude-proxy/internal/security"
)

const (
	flushInterval = 500 * time.Millisecond
	flushSize     = 64 * 1024 // 64KB
)

// EventWriter writes normalized events to JSONL files asynchronously.
type EventWriter struct {
	mu       sync.Mutex
	dir      string
	sessions map[string]*sessionFile
	stopped  bool
	written  atomic.Int32
	noSid    atomic.Int32
}

// Stats returns write statistics.
func (w *EventWriter) Stats() (written, noSid int32) {
	return w.written.Load(), w.noSid.Load()
}

type sessionFile struct {
	f      *os.File
	bw     *bufio.Writer
	lastFlush time.Time
}

// NewEventWriter creates an EventWriter.
func NewEventWriter() (*EventWriter, error) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".cc-connect", "claude-proxy", "events")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("writer: mkdir %s: %w", dir, err)
	}
	return &EventWriter{dir: dir, sessions: make(map[string]*sessionFile)}, nil
}

// Write writes a normalized event. Never blocks — if file I/O fails, event is dropped.
func (w *EventWriter) Write(event *parser.NormalizedEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopped {
		return
	}

	sid := event.SessionID
	if sid == "" {
		w.noSid.Add(1)
		return // don't write events without session_id
	}

	sf, ok := w.sessions[sid]
	if !ok {
		path := filepath.Join(w.dir, fmt.Sprintf("session-%s.jsonl", sid))
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			logging.Warn("writer: cannot open file", "path", path, "error", err)
			return
		}
		sf = &sessionFile{f: f, bw: bufio.NewWriterSize(f, 64*1024)}
		w.sessions[sid] = sf
		logging.Info("writer: opened session file", "session", sid, "path", path)
	}
	w.written.Add(1)

	// Redact before marshaling
	if event.Raw != nil {
		redactMap(event.Raw)
	}
	if event.Payload != nil {
		redactMap(event.Payload)
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	data = append(data, '\n')

	if _, err := sf.bw.Write(data); err != nil {
		logging.Warn("writer: write failed", "session", sid, "error", err)
		return
	}

	// Flush periodically
	now := time.Now()
	if sf.bw.Buffered() >= flushSize || now.Sub(sf.lastFlush) >= flushInterval {
		sf.bw.Flush()
		sf.lastFlush = now
	}
}

// Close flushes and closes all session files.
func (w *EventWriter) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.stopped = true
	for sid, sf := range w.sessions {
		sf.bw.Flush()
		sf.f.Close()
		delete(w.sessions, sid)
	}
}

func redactMap(m map[string]any) {
	for k, v := range m {
		switch val := v.(type) {
		case string:
			m[k] = security.RedactString(val)
		case map[string]any:
			redactMap(val)
		}
	}
}
