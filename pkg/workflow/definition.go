package workflow

import (
	"context"
	"time"
)

// WorkflowDefinition represents a stored workflow definition
type WorkflowDefinition struct {
	Name        string                 `gorm:"column:name;primaryKey" json:"name"`
	DisplayName string                 `gorm:"column:display_name" json:"display_name,omitempty"`
	Description string                 `gorm:"column:description" json:"description,omitempty"`
	Category    string                 `gorm:"column:category" json:"category,omitempty"`
	Tags        string                 `gorm:"column:tags;type:text" json:"-"` // JSON encoded
	TagsData    map[string]interface{} `gorm:"-" json:"tags,omitempty"`
	Parameters  string                 `gorm:"column:parameters;type:text" json:"-"` // JSON encoded
	ParamsData  []Parameter            `gorm:"-" json:"parameters,omitempty"`
	Content     string                 `gorm:"column:content" json:"content"`
	ContentHash string                 `gorm:"column:content_hash" json:"content_hash,omitempty"`
	CreatedAt   time.Time              `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time              `gorm:"column:updated_at" json:"updated_at"`
}

// TableName specifies the table name for GORM
func (WorkflowDefinition) TableName() string {
	return "workflow_definitions"
}

// Parameter represents a workflow parameter extracted from YAML
type Parameter struct {
	Name    string      `json:"name"`
	Type    string      `json:"type"`
	Default interface{} `json:"default,omitempty"`
}

// DefinitionFilter represents filters for querying workflow definitions
type DefinitionFilter struct {
	Category   string
	NamePrefix string
	Page       int
	Limit      int
}

// DefinitionStore defines the interface for workflow definition storage
type DefinitionStore interface {
	// Create creates a new workflow definition
	Create(ctx context.Context, def *WorkflowDefinition) error

	// Get retrieves a workflow definition by name
	Get(ctx context.Context, name string) (*WorkflowDefinition, error)

	// List retrieves workflow definitions with filtering and pagination
	List(ctx context.Context, filter DefinitionFilter) ([]*WorkflowDefinition, int, error)

	// Update updates an existing workflow definition
	Update(ctx context.Context, def *WorkflowDefinition) error

	// Delete deletes a workflow definition by name
	Delete(ctx context.Context, name string) error

	// Exists checks if a workflow definition exists
	Exists(ctx context.Context, name string) (bool, error)
}
