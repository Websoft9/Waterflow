package trigger

import (
	"errors"
	"time"
)

// Sentinel errors for trigger operations
var (
	ErrNotFound        = errors.New("trigger not found")
	ErrAlreadyExists   = errors.New("trigger already exists")
	ErrInvalidSignature = errors.New("invalid signature")
	ErrWorkflowNotFound = errors.New("workflow not found")
)

// Type represents the type of trigger
type Type string

const (
	// TypeWebhook represents a webhook trigger
	TypeWebhook Type = "webhook"
)

// Status represents the status of a trigger
type Status string

const (
	// StatusEnabled indicates the trigger is active
	StatusEnabled Status = "enabled"
	// StatusDisabled indicates the trigger is inactive
	StatusDisabled Status = "disabled"
)

// Trigger represents a workflow trigger configuration
type Trigger struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	WorkflowName  string                 `json:"workflow_name"`
	Type          Type                   `json:"type"`
	WebhookURL    string                 `json:"webhook_url,omitempty"`
	Secret        string                 `json:"-"` // Never expose in JSON responses
	Filters       *FilterConfig          `json:"filters,omitempty"`
	Status        Status                 `json:"status"`
	TotalTriggers int                    `json:"total_triggers"`
	SuccessCount  int                    `json:"successful_triggers"`
	FailedCount   int                    `json:"failed_triggers"`
	LastTriggered *time.Time             `json:"last_triggered_at,omitempty"`
	LastWorkflow  string                 `json:"last_workflow_id,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     *time.Time             `json:"updated_at,omitempty"`
	DisabledAt    *time.Time             `json:"disabled_at,omitempty"`
	DisableReason string                 `json:"disable_reason,omitempty"`
	Vars          map[string]interface{} `json:"vars,omitempty"`
}

// FilterConfig represents webhook filter configuration
type FilterConfig struct {
	Branches       []string `json:"branches,omitempty"`
	BranchesIgnore []string `json:"branches_ignore,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	TagsIgnore     []string `json:"tags_ignore,omitempty"`
	Paths          []string `json:"paths,omitempty"`
	PathsIgnore    []string `json:"paths_ignore,omitempty"` // HIGH-3: exclude paths
	EventTypes     []string `json:"event_types,omitempty"`
}

// TriggerResponse is the API response for Create — includes webhook_secret once
type TriggerResponse struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	WorkflowName  string                 `json:"workflow_name"`
	Type          Type                   `json:"type"`
	WebhookURL    string                 `json:"webhook_url,omitempty"`
	WebhookSecret string                 `json:"webhook_secret,omitempty"` // Only on Create
	Filters       *FilterConfig          `json:"filters,omitempty"`
	Status        Status                 `json:"status"`
	TotalTriggers int                    `json:"total_triggers"`
	SuccessCount  int                    `json:"successful_triggers"`
	FailedCount   int                    `json:"failed_triggers"`
	LastTriggered *time.Time             `json:"last_triggered_at,omitempty"`
	LastWorkflow  string                 `json:"last_workflow_id,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     *time.Time             `json:"updated_at,omitempty"`
	DisabledAt    *time.Time             `json:"disabled_at,omitempty"`
	DisableReason string                 `json:"disable_reason,omitempty"`
	Vars          map[string]interface{} `json:"vars,omitempty"`
}

// ToResponse converts Trigger to TriggerResponse (exposeSecret=true only on Create)
func (t *Trigger) ToResponse(exposeSecret bool) *TriggerResponse {
	r := &TriggerResponse{
		ID:            t.ID,
		Name:          t.Name,
		WorkflowName:  t.WorkflowName,
		Type:          t.Type,
		WebhookURL:    t.WebhookURL,
		Filters:       t.Filters,
		Status:        t.Status,
		TotalTriggers: t.TotalTriggers,
		SuccessCount:  t.SuccessCount,
		FailedCount:   t.FailedCount,
		LastTriggered: t.LastTriggered,
		LastWorkflow:  t.LastWorkflow,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
		DisabledAt:    t.DisabledAt,
		DisableReason: t.DisableReason,
		Vars:          t.Vars,
	}
	if exposeSecret {
		r.WebhookSecret = t.Secret
	}
	return r
}

// CreateRequest represents a trigger creation request
type CreateRequest struct {
	Name         string                 `json:"name"`
	WorkflowName string                 `json:"workflow_name"`
	Type         Type                   `json:"type"`
	Filters      *FilterConfig          `json:"filters,omitempty"`
	Secret       string                 `json:"secret,omitempty"`
	Enabled      bool                   `json:"enabled"`
	Vars         map[string]interface{} `json:"vars,omitempty"`
}

// UpdateRequest represents a trigger update request
type UpdateRequest struct {
	Filters *FilterConfig          `json:"filters,omitempty"`
	Secret  string                 `json:"secret,omitempty"`
	Vars    map[string]interface{} `json:"vars,omitempty"`
}

// WebhookEvent represents a parsed webhook event
type WebhookEvent struct {
	Type         string   `json:"type"`
	Ref          string   `json:"ref"`
	Branch       string   `json:"branch"`
	Tag          string   `json:"tag"`
	CommitID     string   `json:"commit_id"`
	CommitMsg    string   `json:"commit_message"`
	Author       string   `json:"author"`
	ChangedFiles []string `json:"changed_files"`
	Repository   string   `json:"repository"`
	SourceIP     string   `json:"source_ip"`
}

// WebhookResponse represents the response to a webhook trigger
type WebhookResponse struct {
	TriggerID  string `json:"trigger_id"`
	WorkflowID string `json:"workflow_id,omitempty"`
	RunID      string `json:"run_id,omitempty"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

// WebhookLog represents an audit log entry for webhook events
type WebhookLog struct {
	ID                int64     `json:"id"`
	TriggerID         string    `json:"trigger_id"`
	Timestamp         time.Time `json:"timestamp"`
	SourceIP          string    `json:"source_ip"`
	UserAgent         string    `json:"user_agent"`
	EventType         string    `json:"event_type"`
	Ref               string    `json:"ref"`
	SignatureValid    bool      `json:"signature_valid"`
	FilterMatched     bool      `json:"filter_matched"`
	WorkflowTriggered bool      `json:"workflow_triggered"`
	WorkflowID        string    `json:"workflow_id,omitempty"`
	Reason            string    `json:"reason,omitempty"`
	ProcessingTimeMs  int       `json:"processing_time_ms"`
	CreatedAt         time.Time `json:"created_at"`
}

// Filter represents query filter for triggers
type Filter struct {
	Type         Type
	Status       Status
	WorkflowName string
	Limit        int
	Offset       int
}
