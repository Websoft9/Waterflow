package trigger

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// TriggerModel represents a trigger stored in database (GORM model)
type TriggerModel struct {
	ID            string     `gorm:"column:id;primaryKey;type:varchar(255)" json:"id"`
	Name          string     `gorm:"column:name;type:varchar(255);not null" json:"name"`
	WorkflowName  string     `gorm:"column:workflow_name;type:varchar(255);not null;index" json:"workflow_name"`
	Type          string     `gorm:"column:type;type:varchar(50);not null;default:'webhook';index" json:"type"`
	WebhookURL    string     `gorm:"column:webhook_url;type:text" json:"webhook_url,omitempty"`
	Secret        string     `gorm:"column:secret;type:text;not null" json:"-"` // Never expose in JSON
	Filters       JSONMap    `gorm:"column:filters;type:jsonb" json:"filters,omitempty"`
	Status        string     `gorm:"column:status;type:varchar(50);not null;default:'enabled';index" json:"status"`
	TotalTriggers int        `gorm:"column:total_triggers;default:0" json:"total_triggers"`
	SuccessCount  int        `gorm:"column:successful_triggers;default:0" json:"successful_triggers"`
	FailedCount   int        `gorm:"column:failed_triggers;default:0" json:"failed_triggers"`
	LastTriggered *time.Time `gorm:"column:last_triggered_at" json:"last_triggered_at,omitempty"`
	LastWorkflow  string     `gorm:"column:last_workflow_id;type:varchar(255)" json:"last_workflow_id,omitempty"`
	CreatedAt     time.Time  `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt     *time.Time `gorm:"column:updated_at" json:"updated_at,omitempty"`
	DisabledAt    *time.Time `gorm:"column:disabled_at" json:"disabled_at,omitempty"`
	DisableReason string     `gorm:"column:disable_reason;type:text" json:"disable_reason,omitempty"`
	Vars          JSONMap    `gorm:"column:vars;type:jsonb" json:"vars,omitempty"`
}

// TableName specifies the table name for GORM
func (TriggerModel) TableName() string {
	return "triggers"
}

// ToTrigger converts TriggerModel to business logic Trigger type
func (m *TriggerModel) ToTrigger() *Trigger {
	trigger := &Trigger{
		ID:            m.ID,
		Name:          m.Name,
		WorkflowName:  m.WorkflowName,
		Type:          Type(m.Type),
		WebhookURL:    m.WebhookURL,
		Secret:        m.Secret,
		Status:        Status(m.Status),
		TotalTriggers: m.TotalTriggers,
		SuccessCount:  m.SuccessCount,
		FailedCount:   m.FailedCount,
		LastTriggered: m.LastTriggered,
		LastWorkflow:  m.LastWorkflow,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		DisabledAt:    m.DisabledAt,
		DisableReason: m.DisableReason,
	}

	// Convert filters from JSONMap to FilterConfig
	if len(m.Filters) > 0 {
		filterBytes, _ := json.Marshal(m.Filters)
		var filters FilterConfig
		if err := json.Unmarshal(filterBytes, &filters); err == nil {
			trigger.Filters = &filters
		}
	}

	// Convert vars
	if len(m.Vars) > 0 {
		trigger.Vars = map[string]interface{}(m.Vars)
	}

	return trigger
}

// FromTrigger converts business logic Trigger to TriggerModel for database storage
func FromTrigger(t *Trigger) *TriggerModel {
	model := &TriggerModel{
		ID:            t.ID,
		Name:          t.Name,
		WorkflowName:  t.WorkflowName,
		Type:          string(t.Type),
		WebhookURL:    t.WebhookURL,
		Secret:        t.Secret,
		Status:        string(t.Status),
		TotalTriggers: t.TotalTriggers,
		SuccessCount:  t.SuccessCount,
		FailedCount:   t.FailedCount,
		LastTriggered: t.LastTriggered,
		LastWorkflow:  t.LastWorkflow,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
		DisabledAt:    t.DisabledAt,
		DisableReason: t.DisableReason,
	}

	// Convert filters to JSONMap
	if t.Filters != nil {
		filterBytes, _ := json.Marshal(t.Filters)
		var filterMap map[string]interface{}
		if err := json.Unmarshal(filterBytes, &filterMap); err == nil {
			model.Filters = JSONMap(filterMap)
		}
	}

	// Convert vars to JSONMap
	if t.Vars != nil {
		model.Vars = JSONMap(t.Vars)
	}

	return model
}

// WebhookLogModel represents audit log for webhook events (GORM model)
type WebhookLogModel struct {
	gorm.Model
	TriggerID         string    `gorm:"column:trigger_id;type:varchar(255);not null;index" json:"trigger_id"`
	Timestamp         time.Time `gorm:"column:timestamp;not null;index" json:"timestamp"`
	SourceIP          string    `gorm:"column:source_ip;type:varchar(255)" json:"source_ip,omitempty"`
	UserAgent         string    `gorm:"column:user_agent;type:text" json:"user_agent,omitempty"`
	EventType         string    `gorm:"column:event_type;type:varchar(100)" json:"event_type,omitempty"`
	Ref               string    `gorm:"column:ref;type:varchar(500)" json:"ref,omitempty"`
	SignatureValid    bool      `gorm:"column:signature_valid;default:false" json:"signature_valid"`
	FilterMatched     bool      `gorm:"column:filter_matched;default:false" json:"filter_matched"`
	WorkflowTriggered bool      `gorm:"column:workflow_triggered;default:false" json:"workflow_triggered"`
	WorkflowID        string    `gorm:"column:workflow_id;type:varchar(255)" json:"workflow_id,omitempty"`
	Reason            string    `gorm:"column:reason;type:text" json:"reason,omitempty"`
	ProcessingTimeMs  int64     `gorm:"column:processing_time_ms" json:"processing_time_ms"`
}

// TableName specifies the table name for GORM
func (WebhookLogModel) TableName() string {
	return "webhook_logs"
}

// ToWebhookLog converts WebhookLogModel to business logic WebhookLog type
func (m *WebhookLogModel) ToWebhookLog() *WebhookLog {
	return &WebhookLog{
		ID:                int64(m.ID),
		TriggerID:         m.TriggerID,
		Timestamp:         m.Timestamp,
		SourceIP:          m.SourceIP,
		UserAgent:         m.UserAgent,
		EventType:         m.EventType,
		Ref:               m.Ref,
		SignatureValid:    m.SignatureValid,
		FilterMatched:     m.FilterMatched,
		WorkflowTriggered: m.WorkflowTriggered,
		WorkflowID:        m.WorkflowID,
		Reason:            m.Reason,
		ProcessingTimeMs:  int(m.ProcessingTimeMs), // Convert int64 to int
		CreatedAt:         m.CreatedAt,
	}
}

// JSONMap is a custom type for storing JSON data in PostgreSQL JSONB column
type JSONMap map[string]interface{}

// Scan implements sql.Scanner interface for reading from database
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
	}

	result := make(map[string]interface{})
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*j = JSONMap(result)
	return nil
}

// Value implements driver.Valuer interface for writing to database
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}
