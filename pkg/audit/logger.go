// Package audit provides audit logging functionality for tracking security-relevant events.
//
// The audit system records all critical operations including workflow submissions,
// secret access, configuration changes, and authentication events. Audit logs are
// immutable and support compliance requirements like SOC2 and ISO27001.
package audit

import (
	"context"
	"time"
)

// AuditLogger defines the interface for audit logging.
//
// Implementations should ensure:
// - Thread-safe concurrent writes
// - Immutable log entries (append-only)
// - Proper error handling and recovery
type AuditLogger interface {
	// Log records an audit event
	Log(ctx context.Context, entry *AuditLogEntry) error

	// Query retrieves audit logs based on filter
	Query(ctx context.Context, filter AuditLogFilter) ([]*AuditLogEntry, error)

	// Close flushes and closes the logger
	Close() error
}

// AuditLogEntry represents a single audit log entry.
type AuditLogEntry struct {
	// Timestamp is when the event occurred (ISO 8601)
	Timestamp time.Time `json:"timestamp"`

	// EventType is the specific event (e.g., "workflow.submit")
	EventType string `json:"event_type"`

	// EventCategory is the high-level category
	EventCategory EventCategory `json:"event_category"`

	// Severity indicates the importance of the event
	Severity Severity `json:"severity"`

	// User contains information about who performed the action
	User *UserContext `json:"user,omitempty"`

	// Resource identifies what was acted upon
	Resource *ResourceContext `json:"resource,omitempty"`

	// Action is the operation performed
	Action string `json:"action"`

	// Result indicates success/failure
	Result Result `json:"result"`

	// Details contains event-specific additional information
	Details map[string]interface{} `json:"details,omitempty"`

	// Metadata contains request tracking information
	Metadata map[string]string `json:"metadata,omitempty"`
}

// EventCategory represents the category of audit events.
type EventCategory string

const (
	CategoryWorkflow EventCategory = "workflow"
	CategorySecret   EventCategory = "secret"
	CategoryAuth     EventCategory = "auth"
	CategoryConfig   EventCategory = "config"
	CategoryAgent    EventCategory = "agent"
	CategoryAdmin    EventCategory = "admin"
)

// Severity represents the severity level of an audit event.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarn     Severity = "warn"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// Result represents the outcome of an operation.
type Result string

const (
	ResultSuccess          Result = "success"
	ResultFailure          Result = "failure"
	ResultError            Result = "error"
	ResultPermissionDenied Result = "permission_denied"
	ResultNotFound         Result = "not_found"
)

// UserContext contains information about the user who performed the action.
type UserContext struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	IP        string `json:"ip,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
}

// ResourceContext identifies the resource being acted upon.
type ResourceContext struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// AuditLogFilter defines criteria for querying audit logs.
type AuditLogFilter struct {
	StartTime     *time.Time
	EndTime       *time.Time
	EventTypes    []string
	EventCategory *EventCategory
	UserID        string
	ResourceType  string
	ResourceID    string
	Result        *Result
	Limit         int
	Offset        int
}

// AuditLogConfig holds the configuration for audit logging.
type AuditLogConfig struct {
	// Enabled turns on audit logging (default: true)
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`

	// StoreType specifies the storage backend (file, database)
	StoreType string `mapstructure:"store_type" yaml:"store_type"`

	// File configuration for FileAuditStore
	File *FileAuditStoreConfig `mapstructure:"file" yaml:"file,omitempty"`
}

// FileAuditStoreConfig configures the file-based audit store.
type FileAuditStoreConfig struct {
	// Path is the directory for audit log files
	Path string `mapstructure:"path" yaml:"path"`

	// MaxSize is the maximum size in MB before rotation (default: 100)
	MaxSize int `mapstructure:"max_size" yaml:"max_size"`

	// MaxAge is the maximum days to retain old logs (default: 90)
	MaxAge int `mapstructure:"max_age" yaml:"max_age"`

	// MaxBackups is the maximum number of old log files to retain (default: 30)
	MaxBackups int `mapstructure:"max_backups" yaml:"max_backups"`

	// Compress enables gzip compression of rotated files (default: true)
	Compress bool `mapstructure:"compress" yaml:"compress"`
}
