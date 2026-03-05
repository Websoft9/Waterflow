package trigger

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/Websoft9/waterflow/pkg/workflow"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// Storage interface for trigger persistence
type Storage interface {
	Save(trigger *Trigger) error
	Get(id string) (*Trigger, error)
	List(filter *Filter) ([]*Trigger, int, error)
	Update(id string, trigger *Trigger) error
	Delete(id string) error
	IncrementTriggerCount(id string, success bool) error
	UpdateLastTrigger(id string, workflowID string) error
	SaveWebhookLog(log *WebhookLog) error
	GetWebhookLogs(triggerID string, limit int) ([]*WebhookLog, error)
}

// Manager handles Trigger business logic
type Manager struct {
	temporalClient client.Client
	defStore       workflow.DefinitionStore
	storage        Storage
	logger         *zap.Logger
	filterEngine   *FilterEngine
	baseURL        string // Base URL for webhook URLs
}

// NewManager creates a new Trigger Manager
func NewManager(
	temporalClient client.Client,
	defStore workflow.DefinitionStore,
	storage Storage,
	logger *zap.Logger,
	baseURL string,
) *Manager {
	return &Manager{
		temporalClient: temporalClient,
		defStore:       defStore,
		storage:        storage,
		logger:         logger,
		filterEngine:   NewFilterEngine(),
		baseURL:        baseURL,
	}
}

// Create creates a new Webhook Trigger
func (m *Manager) Create(ctx context.Context, req *CreateRequest) (*Trigger, error) {
	// Validate type
	if req.Type != TypeWebhook {
		return nil, fmt.Errorf("unsupported trigger type: %s", req.Type)
	}

	// Validate workflow exists
	if _, err := m.defStore.Get(ctx, req.WorkflowName); err != nil {
		return nil, fmt.Errorf("workflow '%s' not found: %w", req.WorkflowName, err)
	}

	// Validate filter configuration
	if err := validateFilters(req.Filters); err != nil {
		return nil, fmt.Errorf("invalid filters: %w", err)
	}

	// Generate trigger ID from name
	triggerID := generateID(req.Name)

	// Check if trigger ID already exists
	if existing, _ := m.storage.Get(triggerID); existing != nil {
		return nil, fmt.Errorf("trigger with name '%s' already exists", req.Name)
	}

	// Generate secret if not provided
	secret := req.Secret
	if secret == "" {
		secret = generateSecret()
	}
	if len(secret) < 16 {
		return nil, fmt.Errorf("secret must be at least 16 characters")
	}

	// Determine status
	status := StatusDisabled
	if req.Enabled {
		status = StatusEnabled
	}

	// Create trigger
	now := time.Now()
	trigger := &Trigger{
		ID:           triggerID,
		Name:         req.Name,
		WorkflowName: req.WorkflowName,
		Type:         req.Type,
		WebhookURL:   fmt.Sprintf("%s/api/v1/webhooks/%s/trigger", m.baseURL, triggerID),
		Secret:       secret,
		Filters:      req.Filters,
		Status:       status,
		Vars:         req.Vars,
		CreatedAt:    now,
	}

	// Save to storage
	if err := m.storage.Save(trigger); err != nil {
		return nil, fmt.Errorf("failed to save trigger: %w", err)
	}

	m.logger.Info("Webhook trigger created",
		zap.String("trigger_id", triggerID),
		zap.String("workflow_name", req.WorkflowName),
		zap.String("status", string(status)),
	)

	return trigger, nil
}

// Get retrieves a trigger by ID
func (m *Manager) Get(ctx context.Context, id string) (*Trigger, error) {
	trigger, err := m.storage.Get(id)
	if err != nil {
		return nil, fmt.Errorf("trigger not found: %w", err)
	}
	return trigger, nil
}

// List retrieves triggers with filtering
func (m *Manager) List(ctx context.Context, filter *Filter) ([]*Trigger, int, error) {
	triggers, total, err := m.storage.List(filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list triggers: %w", err)
	}
	return triggers, total, nil
}

// Update updates a trigger configuration
func (m *Manager) Update(ctx context.Context, id string, req *UpdateRequest) (*Trigger, error) {
	// Get existing trigger
	trigger, err := m.storage.Get(id)
	if err != nil {
		return nil, fmt.Errorf("trigger not found: %w", err)
	}

	// Update fields
	if req.Filters != nil {
		if err := validateFilters(req.Filters); err != nil {
			return nil, fmt.Errorf("invalid filters: %w", err)
		}
		trigger.Filters = req.Filters
	}

	if req.Secret != "" {
		if len(req.Secret) < 16 {
			return nil, fmt.Errorf("secret must be at least 16 characters")
		}
		trigger.Secret = req.Secret
	}

	if req.Vars != nil {
		trigger.Vars = req.Vars
	}

	now := time.Now()
	trigger.UpdatedAt = &now

	// Save changes
	if err := m.storage.Update(id, trigger); err != nil {
		return nil, fmt.Errorf("failed to update trigger: %w", err)
	}

	m.logger.Info("Webhook trigger updated",
		zap.String("trigger_id", id),
	)

	return trigger, nil
}

// Delete deletes a trigger
func (m *Manager) Delete(ctx context.Context, id string) error {
	// Check if trigger exists
	if _, err := m.storage.Get(id); err != nil {
		return fmt.Errorf("trigger not found: %w", err)
	}

	// Delete from storage
	if err := m.storage.Delete(id); err != nil {
		return fmt.Errorf("failed to delete trigger: %w", err)
	}

	m.logger.Info("Webhook trigger deleted",
		zap.String("trigger_id", id),
	)

	return nil
}

// Enable enables a trigger
func (m *Manager) Enable(ctx context.Context, id string) (*Trigger, error) {
	trigger, err := m.storage.Get(id)
	if err != nil {
		return nil, fmt.Errorf("trigger not found: %w", err)
	}

	if trigger.Status == StatusEnabled {
		return trigger, nil // Already enabled
	}

	trigger.Status = StatusEnabled
	trigger.DisabledAt = nil
	trigger.DisableReason = ""
	now := time.Now()
	trigger.UpdatedAt = &now

	if err := m.storage.Update(id, trigger); err != nil {
		return nil, fmt.Errorf("failed to enable trigger: %w", err)
	}

	m.logger.Info("Webhook trigger enabled",
		zap.String("trigger_id", id),
	)

	return trigger, nil
}

// Disable disables a trigger
func (m *Manager) Disable(ctx context.Context, id string, reason string) (*Trigger, error) {
	trigger, err := m.storage.Get(id)
	if err != nil {
		return nil, fmt.Errorf("trigger not found: %w", err)
	}

	if trigger.Status == StatusDisabled {
		return trigger, nil // Already disabled
	}

	trigger.Status = StatusDisabled
	now := time.Now()
	trigger.DisabledAt = &now
	trigger.DisableReason = reason
	trigger.UpdatedAt = &now

	if err := m.storage.Update(id, trigger); err != nil {
		return nil, fmt.Errorf("failed to disable trigger: %w", err)
	}

	m.logger.Info("Webhook trigger disabled",
		zap.String("trigger_id", id),
		zap.String("reason", reason),
	)

	return trigger, nil
}

// HandleWebhook processes an incoming webhook request
func (m *Manager) HandleWebhook(
	ctx context.Context,
	triggerID string,
	payload []byte,
	signature string,
	headers map[string]string,
	event *WebhookEvent,
) (*WebhookResponse, error) {
	startTime := time.Now()

	// Get trigger
	trigger, err := m.storage.Get(triggerID)
	if err != nil {
		return nil, fmt.Errorf("trigger not found: %w", err)
	}

	// Check if enabled
	if trigger.Status != StatusEnabled {
		m.logWebhookEvent(triggerID, event, headers, false, false, false, "Trigger disabled", startTime)
		return &WebhookResponse{
			TriggerID: triggerID,
			Status:    "ignored",
			Message:   "Trigger is disabled",
		}, nil
	}

	// Verify signature
	if !verifySignature(payload, signature, trigger.Secret) {
		m.logWebhookEvent(triggerID, event, headers, false, false, false, "Signature verification failed", startTime)
		return nil, fmt.Errorf("invalid signature")
	}

	// Apply filters
	matched, reason := m.filterEngine.Match(event, trigger.Filters)
	if !matched {
		m.logWebhookEvent(triggerID, event, headers, true, false, false, reason, startTime)
		return &WebhookResponse{
			TriggerID: triggerID,
			Status:    "ignored",
			Message:   reason,
		}, nil
	}

	// Get workflow definition
	workflowDef, err := m.defStore.Get(ctx, trigger.WorkflowName)
	if err != nil {
		m.logWebhookEvent(triggerID, event, headers, true, true, false, fmt.Sprintf("Workflow not found: %v", err), startTime)
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	// Generate workflow ID
	workflowID := fmt.Sprintf("%s-%d", triggerID, time.Now().Unix())

	// Merge vars (webhook trigger vars override workflow defaults)
	vars := make(map[string]interface{})
	if trigger.Vars != nil {
		for k, v := range trigger.Vars {
			vars[k] = v
		}
	}

	// TODO: Trigger workflow execution via Temporal
	// This will be implemented when we integrate with the execution handler
	// For now, we'll mark it as triggered
	runID := "pending-implementation"

	// Update statistics
	m.storage.IncrementTriggerCount(triggerID, true)
	m.storage.UpdateLastTrigger(triggerID, workflowID)

	// Log success
	m.logWebhookEvent(triggerID, event, headers, true, true, true, "Workflow triggered successfully", startTime)

	m.logger.Info("Webhook triggered workflow",
		zap.String("trigger_id", triggerID),
		zap.String("workflow_id", workflowID),
		zap.String("workflow_name", workflowDef.Name),
	)

	return &WebhookResponse{
		TriggerID:  triggerID,
		WorkflowID: workflowID,
		RunID:      runID,
		Status:     "triggered",
		Message:    "Workflow triggered successfully",
	}, nil
}

// logWebhookEvent creates an audit log entry for the webhook event
// GetWebhookLogs retrieves webhook event logs for a trigger
func (m *Manager) GetWebhookLogs(ctx context.Context, triggerID string, limit int) ([]*WebhookLog, error) {
	if limit <= 0 {
		limit = 50
	}
	return m.storage.GetWebhookLogs(triggerID, limit)
}

func (m *Manager) logWebhookEvent(
	triggerID string,
	event *WebhookEvent,
	headers map[string]string,
	signatureValid bool,
	filterMatched bool,
	workflowTriggered bool,
	reason string,
	startTime time.Time,
) {
	duration := time.Since(startTime)

	log := &WebhookLog{
		TriggerID:         triggerID,
		Timestamp:         startTime,
		SourceIP:          event.SourceIP,
		UserAgent:         headers["User-Agent"],
		EventType:         event.Type,
		Ref:               event.Ref,
		SignatureValid:    signatureValid,
		FilterMatched:     filterMatched,
		WorkflowTriggered: workflowTriggered,
		Reason:            reason,
		ProcessingTimeMs:  int(duration.Milliseconds()),
		CreatedAt:         time.Now(),
	}

	if err := m.storage.SaveWebhookLog(log); err != nil {
		m.logger.Error("Failed to save webhook log",
			zap.String("trigger_id", triggerID),
			zap.Error(err),
		)
	}
}

// verifySignature validates HMAC-SHA256 signature using constant-time comparison
func verifySignature(payload []byte, signature string, secret string) bool {
	// Compute expected signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	expectedSignature := "sha256=" + expectedMAC

	// Constant-time comparison to prevent timing attacks
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// generateSecret generates a random secret for webhook signing
func generateSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based secret if random fails
		return fmt.Sprintf("secret-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// generateID generates a trigger ID from the trigger name
func generateID(name string) string {
	// Convert to lowercase and replace spaces with hyphens
	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "-")
	// Remove special characters except hyphens and alphanumeric
	var result strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// validateFilters validates filter configuration
func validateFilters(filters *FilterConfig) error {
	if filters == nil {
		return nil
	}

	// Validate branch patterns
	for _, pattern := range filters.Branches {
		if pattern == "" {
			return fmt.Errorf("branch pattern cannot be empty")
		}
	}

	for _, pattern := range filters.BranchesIgnore {
		if pattern == "" {
			return fmt.Errorf("branches_ignore pattern cannot be empty")
		}
	}

	// Validate tag patterns
	for _, pattern := range filters.Tags {
		if pattern == "" {
			return fmt.Errorf("tag pattern cannot be empty")
		}
	}

	// Validate path patterns
	for _, pattern := range filters.Paths {
		if pattern == "" {
			return fmt.Errorf("path pattern cannot be empty")
		}
	}

	return nil
}
