package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

// DatabaseDefinitionStore implements DefinitionStore using GORM
type DatabaseDefinitionStore struct {
	db *gorm.DB
}

// NewDatabaseDefinitionStore creates a new database-backed definition store
func NewDatabaseDefinitionStore(db *gorm.DB) *DatabaseDefinitionStore {
	return &DatabaseDefinitionStore{
		db: db,
	}
}

// Create creates a new workflow definition
func (s *DatabaseDefinitionStore) Create(ctx context.Context, def *WorkflowDefinition) error {
	// Generate content hash
	def.ContentHash = s.generateContentHash(def.Content)

	// Encode JSON fields
	if err := s.encodeJSONFields(def); err != nil {
		return fmt.Errorf("failed to encode fields: %w", err)
	}

	// Create record
	result := s.db.WithContext(ctx).Create(def)
	if result.Error != nil {
		if isDuplicateKeyError(result.Error) {
			return fmt.Errorf("workflow definition '%s' already exists", def.Name)
		}
		return fmt.Errorf("failed to create workflow definition: %w", result.Error)
	}

	// Decode JSON fields for return
	_ = s.decodeJSONFields(def)

	return nil
}

// Get retrieves a workflow definition by name
func (s *DatabaseDefinitionStore) Get(ctx context.Context, name string) (*WorkflowDefinition, error) {
	var def WorkflowDefinition
	result := s.db.WithContext(ctx).Where("name = ?", name).First(&def)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("workflow definition '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to get workflow definition: %w", result.Error)
	}

	// Decode JSON fields
	_ = s.decodeJSONFields(&def)

	return &def, nil
}

// List retrieves workflow definitions with filtering and pagination
func (s *DatabaseDefinitionStore) List(ctx context.Context, filter DefinitionFilter) ([]*WorkflowDefinition, int, error) {
	// Build query
	query := s.db.WithContext(ctx).Model(&WorkflowDefinition{})

	// Apply filters
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.NamePrefix != "" {
		query = query.Where("name LIKE ?", filter.NamePrefix+"%")
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count workflow definitions: %w", err)
	}

	// Apply pagination
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 20
	}
	offset := (filter.Page - 1) * filter.Limit

	// Get records
	var defs []*WorkflowDefinition
	result := query.
		Order("updated_at DESC").
		Offset(offset).
		Limit(filter.Limit).
		Find(&defs)

	if result.Error != nil {
		return nil, 0, fmt.Errorf("failed to list workflow definitions: %w", result.Error)
	}

	// Decode JSON fields for each definition
	for _, def := range defs {
		_ = s.decodeJSONFields(def)
	}

	return defs, int(total), nil
}

// Update updates an existing workflow definition
func (s *DatabaseDefinitionStore) Update(ctx context.Context, def *WorkflowDefinition) error {
	// Generate new content hash
	def.ContentHash = s.generateContentHash(def.Content)

	// Encode JSON fields
	if err := s.encodeJSONFields(def); err != nil {
		return fmt.Errorf("failed to encode fields: %w", err)
	}

	// Update record
	result := s.db.WithContext(ctx).
		Model(&WorkflowDefinition{}).
		Where("name = ?", def.Name).
		Updates(map[string]interface{}{
			"display_name": def.DisplayName,
			"description":  def.Description,
			"category":     def.Category,
			"tags":         def.Tags,
			"parameters":   def.Parameters,
			"content":      def.Content,
			"content_hash": def.ContentHash,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update workflow definition: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("workflow definition '%s' not found", def.Name)
	}

	// Decode JSON fields for return
	_ = s.decodeJSONFields(def)

	return nil
}

// Delete deletes a workflow definition by name
func (s *DatabaseDefinitionStore) Delete(ctx context.Context, name string) error {
	result := s.db.WithContext(ctx).Where("name = ?", name).Delete(&WorkflowDefinition{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete workflow definition: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("workflow definition '%s' not found", name)
	}

	return nil
}

// Exists checks if a workflow definition exists
func (s *DatabaseDefinitionStore) Exists(ctx context.Context, name string) (bool, error) {
	var count int64
	result := s.db.WithContext(ctx).Model(&WorkflowDefinition{}).Where("name = ?", name).Count(&count)
	if result.Error != nil {
		return false, fmt.Errorf("failed to check workflow definition existence: %w", result.Error)
	}

	return count > 0, nil
}

// generateContentHash generates SHA-256 hash of the content
func (s *DatabaseDefinitionStore) generateContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// encodeJSONFields encodes JSON data fields to string columns
func (s *DatabaseDefinitionStore) encodeJSONFields(def *WorkflowDefinition) error {
	// Encode Tags
	if def.TagsData != nil {
		data, err := json.Marshal(def.TagsData)
		if err != nil {
			return fmt.Errorf("failed to encode tags: %w", err)
		}
		def.Tags = string(data)
	}

	// Encode Parameters
	if def.ParamsData != nil {
		data, err := json.Marshal(def.ParamsData)
		if err != nil {
			return fmt.Errorf("failed to encode parameters: %w", err)
		}
		def.Parameters = string(data)
	}

	return nil
}

// decodeJSONFields decodes string columns to JSON data fields
func (s *DatabaseDefinitionStore) decodeJSONFields(def *WorkflowDefinition) error {
	// Decode Tags
	if def.Tags != "" && def.Tags != "null" {
		var tags map[string]interface{}
		if err := json.Unmarshal([]byte(def.Tags), &tags); err != nil {
			return fmt.Errorf("failed to decode tags: %w", err)
		}
		def.TagsData = tags
	}

	// Decode Parameters
	if def.Parameters != "" && def.Parameters != "null" {
		var params []Parameter
		if err := json.Unmarshal([]byte(def.Parameters), &params); err != nil {
			return fmt.Errorf("failed to decode parameters: %w", err)
		}
		def.ParamsData = params
	}

	return nil
}

// isDuplicateKeyError checks if the error is a duplicate key violation
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	// PostgreSQL unique violation
	errMsg := err.Error()
	return contains(errMsg, "duplicate key") || contains(errMsg, "UNIQUE constraint")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
