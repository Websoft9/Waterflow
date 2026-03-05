package trigger

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// GORMStorage implements Storage interface using GORM and PostgreSQL
type GORMStorage struct {
	db *gorm.DB
}

// NewGORMStorage creates a new GORM-based storage
func NewGORMStorage(db *gorm.DB) (*GORMStorage, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	return &GORMStorage{db: db}, nil
}

// Save saves a trigger to database (create or update)
func (s *GORMStorage) Save(trigger *Trigger) error {
	model := FromTrigger(trigger)

	// Use Create for new triggers
	result := s.db.Create(model)
	if result.Error != nil {
		return fmt.Errorf("failed to save trigger: %w", result.Error)
	}

	return nil
}

// Get retrieves a trigger by ID
func (s *GORMStorage) Get(id string) (*Trigger, error) {
	var model TriggerModel

	result := s.db.Where("id = ?", id).First(&model)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("trigger not found")
		}
		return nil, fmt.Errorf("failed to get trigger: %w", result.Error)
	}

	return model.ToTrigger(), nil
}

// List retrieves triggers with filtering and pagination
func (s *GORMStorage) List(filter *Filter) ([]*Trigger, int, error) {
	var models []TriggerModel
	var total int64

	query := s.db.Model(&TriggerModel{})

	// Apply filters
	if filter != nil {
		if filter.WorkflowName != "" {
			query = query.Where("workflow_name = ?", filter.WorkflowName)
		}
		if filter.Type != "" {
			query = query.Where("type = ?", filter.Type)
		}
		if filter.Status != "" {
			query = query.Where("status = ?", filter.Status)
		}
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count triggers: %w", err)
	}

	// Apply pagination
	if filter != nil {
		if filter.Limit > 0 {
			query = query.Limit(filter.Limit)
		}
		if filter.Offset > 0 {
			query = query.Offset(filter.Offset)
		}
	}

	// Execute query
	if err := query.Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list triggers: %w", err)
	}

	// Convert to business logic types
	triggers := make([]*Trigger, len(models))
	for i, model := range models {
		triggers[i] = model.ToTrigger()
	}

	return triggers, int(total), nil
}

// Update updates an existing trigger
func (s *GORMStorage) Update(id string, trigger *Trigger) error {
	model := FromTrigger(trigger)

	// Update using struct (only updates non-zero fields)
	result := s.db.Model(&TriggerModel{}).Where("id = ?", id).Updates(model)
	if result.Error != nil {
		return fmt.Errorf("failed to update trigger: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("trigger not found")
	}

	return nil
}

// Delete deletes a trigger by ID
func (s *GORMStorage) Delete(id string) error {
	result := s.db.Where("id = ?", id).Delete(&TriggerModel{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete trigger: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("trigger not found")
	}

	return nil
}

// IncrementTriggerCount increments trigger count statistics
func (s *GORMStorage) IncrementTriggerCount(id string, success bool) error {
	updates := map[string]interface{}{
		"total_triggers": gorm.Expr("total_triggers + 1"),
	}

	if success {
		updates["successful_triggers"] = gorm.Expr("successful_triggers + 1")
	} else {
		updates["failed_triggers"] = gorm.Expr("failed_triggers + 1")
	}

	result := s.db.Model(&TriggerModel{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to increment trigger count: %w", result.Error)
	}

	return nil
}

// UpdateLastTrigger updates the last trigger information
func (s *GORMStorage) UpdateLastTrigger(id string, workflowID string) error {
	now := time.Now()

	updates := map[string]interface{}{
		"last_triggered_at": now,
		"last_workflow_id":  workflowID,
	}

	result := s.db.Model(&TriggerModel{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update last trigger: %w", result.Error)
	}

	return nil
}

// SaveWebhookLog saves a webhook audit log entry
func (s *GORMStorage) SaveWebhookLog(log *WebhookLog) error {
	model := &WebhookLogModel{
		TriggerID:         log.TriggerID,
		Timestamp:         log.Timestamp,
		SourceIP:          log.SourceIP,
		UserAgent:         log.UserAgent,
		EventType:         log.EventType,
		Ref:               log.Ref,
		SignatureValid:    log.SignatureValid,
		FilterMatched:     log.FilterMatched,
		WorkflowTriggered: log.WorkflowTriggered,
		WorkflowID:        log.WorkflowID,
		Reason:            log.Reason,
		ProcessingTimeMs:  int64(log.ProcessingTimeMs), // Convert int to int64
	}

	result := s.db.Create(model)
	if result.Error != nil {
		return fmt.Errorf("failed to save webhook log: %w", result.Error)
	}

	return nil
}

// GetWebhookLogs retrieves webhook logs for a trigger
func (s *GORMStorage) GetWebhookLogs(triggerID string, limit int) ([]*WebhookLog, error) {
	var models []WebhookLogModel

	query := s.db.Where("trigger_id = ?", triggerID).
		Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get webhook logs: %w", err)
	}

	// Convert to business logic types
	logs := make([]*WebhookLog, len(models))
	for i, model := range models {
		logs[i] = model.ToWebhookLog()
	}

	return logs, nil
}
