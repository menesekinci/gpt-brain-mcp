package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Event represents a single audited action.
type Event struct {
	TS            string `json:"ts"`
	Tool          string `json:"tool"`
	ProjectID     string `json:"project_id,omitempty"`
	Path          string `json:"path,omitempty"`
	Decision      string `json:"decision"`
	Reason        string `json:"reason,omitempty"`
	BytesReturned int    `json:"bytes_returned,omitempty"`
}

// Logger writes audit events to a JSONL file.
type Logger struct {
	mu     sync.Mutex
	file   *os.File
	enc    *json.Encoder
	enabled bool
}

// NewLogger creates an audit logger.
func NewLogger(enabled bool, logDir string) (*Logger, error) {
	if !enabled {
		return &Logger{enabled: false}, nil
	}
	if err := os.MkdirAll(logDir, 0o750); err != nil {
		return nil, fmt.Errorf("create audit log dir: %w", err)
	}
	path := filepath.Join(logDir, "audit.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return nil, fmt.Errorf("open audit log: %w", err)
	}
	return &Logger{file: f, enc: json.NewEncoder(f), enabled: true}, nil
}

// Log records an audit event.
func (l *Logger) Log(e Event) {
	if !l.enabled {
		return
	}
	if e.TS == "" {
		e.TS = time.Now().UTC().Format(time.RFC3339)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_ = l.enc.Encode(e)
}

// Close closes the underlying file.
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
