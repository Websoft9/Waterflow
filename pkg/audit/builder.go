package audit

import (
	"time"
)

// NewAuditLogEntry creates a new audit log entry with default values.
func NewAuditLogEntry(eventType string, category EventCategory) *AuditLogEntry {
	return &AuditLogEntry{
		Timestamp:     time.Now().UTC(),
		EventType:     eventType,
		EventCategory: category,
		Severity:      SeverityInfo,
		Result:        ResultSuccess,
		Details:       make(map[string]interface{}),
		Metadata:      make(map[string]string),
	}
}

// WithUser sets the user context.
func (e *AuditLogEntry) WithUser(user *UserContext) *AuditLogEntry {
	e.User = user
	return e
}

// WithResource sets the resource context.
func (e *AuditLogEntry) WithResource(resource *ResourceContext) *AuditLogEntry {
	e.Resource = resource
	return e
}

// WithAction sets the action.
func (e *AuditLogEntry) WithAction(action string) *AuditLogEntry {
	e.Action = action
	return e
}

// WithResult sets the result.
func (e *AuditLogEntry) WithResult(result Result) *AuditLogEntry {
	e.Result = result
	return e
}

// WithSeverity sets the severity.
func (e *AuditLogEntry) WithSeverity(severity Severity) *AuditLogEntry {
	e.Severity = severity
	return e
}

// WithDetail adds a detail field.
func (e *AuditLogEntry) WithDetail(key string, value interface{}) *AuditLogEntry {
	e.Details[key] = value
	return e
}

// WithDetails sets all details at once.
func (e *AuditLogEntry) WithDetails(details map[string]interface{}) *AuditLogEntry {
	e.Details = details
	return e
}

// WithMetadata adds a metadata field.
func (e *AuditLogEntry) WithMetadata(key, value string) *AuditLogEntry {
	e.Metadata[key] = value
	return e
}

// Standard event types for common operations
const (
	// Workflow events
	EventWorkflowSubmit = "workflow.submit"
	EventWorkflowCancel = "workflow.cancel"
	EventWorkflowRerun  = "workflow.rerun"
	EventWorkflowStatus = "workflow.status"
	EventWorkflowList   = "workflow.list"

	// Secret events
	EventSecretAccess   = "secret.access"
	EventSecretList     = "secret.list"
	EventSecretNotFound = "secret.not_found"

	// Auth events
	EventAuthSuccess        = "auth.success"
	EventAuthFailure        = "auth.failure"
	EventAuthLogout         = "auth.logout"
	EventAuthSessionExpired = "auth.session_expired"

	// Config events
	EventConfigUpdate   = "config.update"
	EventConfigReload   = "config.reload"
	EventConfigValidate = "config.validate"

	// Agent events
	EventAgentRegister   = "agent.register"
	EventAgentHeartbeat  = "agent.heartbeat"
	EventAgentDisconnect = "agent.disconnect"

	// Admin events
	EventAdminUserCreate   = "admin.user_create"
	EventAdminRoleAssign   = "admin.role_assign"
	EventAdminPolicyUpdate = "admin.policy_update"

	// API events
	EventAPIRequest = "api.request"
)
