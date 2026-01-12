package audit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAuditLogEntry(t *testing.T) {
	entry := NewAuditLogEntry(EventWorkflowSubmit, CategoryWorkflow)

	assert.NotNil(t, entry)
	assert.Equal(t, EventWorkflowSubmit, entry.EventType)
	assert.Equal(t, CategoryWorkflow, entry.EventCategory)
	assert.Equal(t, SeverityInfo, entry.Severity)
	assert.Equal(t, ResultSuccess, entry.Result)
	assert.NotZero(t, entry.Timestamp)
	assert.NotNil(t, entry.Details)
	assert.NotNil(t, entry.Metadata)
}

func TestAuditLogEntry_FluentAPI(t *testing.T) {
	entry := NewAuditLogEntry(EventWorkflowSubmit, CategoryWorkflow).
		WithUser(&UserContext{
			ID:   "user-123",
			Name: "john.doe",
			IP:   "192.168.1.100",
		}).
		WithResource(&ResourceContext{
			Type: "workflow",
			ID:   "wf-abc123",
			Name: "deploy-production",
		}).
		WithAction("submit").
		WithResult(ResultSuccess).
		WithSeverity(SeverityInfo).
		WithDetail("workflow_size", 1024).
		WithMetadata("request_id", "req-xyz789")

	assert.Equal(t, "user-123", entry.User.ID)
	assert.Equal(t, "john.doe", entry.User.Name)
	assert.Equal(t, "192.168.1.100", entry.User.IP)

	assert.Equal(t, "workflow", entry.Resource.Type)
	assert.Equal(t, "wf-abc123", entry.Resource.ID)
	assert.Equal(t, "deploy-production", entry.Resource.Name)

	assert.Equal(t, "submit", entry.Action)
	assert.Equal(t, ResultSuccess, entry.Result)
	assert.Equal(t, SeverityInfo, entry.Severity)

	assert.Equal(t, 1024, entry.Details["workflow_size"])
	assert.Equal(t, "req-xyz789", entry.Metadata["request_id"])
}

func TestEventCategories(t *testing.T) {
	categories := []EventCategory{
		CategoryWorkflow,
		CategorySecret,
		CategoryAuth,
		CategoryConfig,
		CategoryAgent,
		CategoryAdmin,
	}

	for _, cat := range categories {
		assert.NotEmpty(t, string(cat))
	}
}

func TestSeverityLevels(t *testing.T) {
	severities := []Severity{
		SeverityInfo,
		SeverityWarn,
		SeverityError,
		SeverityCritical,
	}

	for _, sev := range severities {
		assert.NotEmpty(t, string(sev))
	}
}

func TestResults(t *testing.T) {
	results := []Result{
		ResultSuccess,
		ResultFailure,
		ResultError,
		ResultPermissionDenied,
		ResultNotFound,
	}

	for _, res := range results {
		assert.NotEmpty(t, string(res))
	}
}

func TestAuditLogFilter(t *testing.T) {
	now := time.Now()
	category := CategoryWorkflow
	result := ResultSuccess

	filter := AuditLogFilter{
		StartTime:     &now,
		EndTime:       &now,
		EventTypes:    []string{EventWorkflowSubmit, EventWorkflowCancel},
		EventCategory: &category,
		UserID:        "user-123",
		ResourceType:  "workflow",
		ResourceID:    "wf-abc",
		Result:        &result,
		Limit:         100,
		Offset:        0,
	}

	assert.NotNil(t, filter.StartTime)
	assert.NotNil(t, filter.EndTime)
	assert.Len(t, filter.EventTypes, 2)
	assert.Equal(t, CategoryWorkflow, *filter.EventCategory)
	assert.Equal(t, "user-123", filter.UserID)
	assert.Equal(t, 100, filter.Limit)
}

func TestStandardEventTypes(t *testing.T) {
	eventTypes := []string{
		EventWorkflowSubmit,
		EventWorkflowCancel,
		EventSecretAccess,
		EventAuthSuccess,
		EventAuthFailure,
		EventConfigUpdate,
		EventAgentRegister,
		EventAdminUserCreate,
	}

	for _, et := range eventTypes {
		assert.NotEmpty(t, et)
		assert.Contains(t, et, ".")
	}
}
