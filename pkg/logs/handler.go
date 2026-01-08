package logs

import (
	"context"
	"time"
)

// LogLevel represents log severity level.
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

// LogEntry represents a workflow execution log entry.
type LogEntry struct {
	Timestamp  time.Time              `json:"timestamp"`
	Level      LogLevel               `json:"level"`
	WorkflowID string                 `json:"workflow_id"`
	RunID      string                 `json:"run_id,omitempty"`
	JobID      string                 `json:"job_id,omitempty"`
	StepID     string                 `json:"step_id,omitempty"`
	NodeType   string                 `json:"node_type,omitempty"`
	Message    string                 `json:"message"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// LogHandler handles workflow execution logs.
type LogHandler interface {
	OnLog(ctx context.Context, entry *LogEntry) error
	Close() error
}
