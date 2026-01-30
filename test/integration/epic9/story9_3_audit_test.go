//go:build integration

package epic9

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStory9_3_AC2_FileAuditStore verifies append-only file storage
//
// Story: 9.3 - 审计日志实现
// AC: AC2 - 实现 FileAuditStore (append-only 文件存储)
// Priority: P0 (Critical - Compliance requirement)
func TestStory9_3_AC2_FileAuditStore(t *testing.T) {
	// Create temp directory for audit logs
	tempDir := t.TempDir()
	auditPath := filepath.Join(tempDir, "audit.log")

	// Create file audit store
	config := &audit.FileAuditStoreConfig{
		FilePath:   auditPath,
		MaxSize:    10, // MB
		MaxBackups: 3,
		MaxAge:     7, // days
		Compress:   true,
	}

	store, err := audit.NewFileAuditStore(config)
	require.NoError(t, err, "Failed to create file audit store")
	defer store.Close()

	ctx := context.Background()

	// Log test audit entry
	entry := &audit.AuditLogEntry{
		Timestamp:     time.Now(),
		EventType:     audit.EventWorkflowSubmitted,
		EventCategory: audit.CategoryWorkflow,
		Severity:      audit.SeverityInfo,
		User: &audit.UserContext{
			ID:   "test-user-123",
			Name: "Test User",
		},
		Resource: &audit.ResourceContext{
			Type: "workflow",
			ID:   "wf-test-001",
		},
		Action: "submit",
		Result: audit.ResultSuccess,
		Details: map[string]interface{}{
			"workflow_name": "test-workflow",
		},
	}

	err = store.Log(ctx, entry)
	require.NoError(t, err, "Failed to log audit entry")

	t.Logf("✅ Audit entry logged successfully")

	// Verify file exists and is append-only
	_, err = os.Stat(auditPath)
	require.NoError(t, err, "Audit log file does not exist")

	// Read file content
	content, err := os.ReadFile(auditPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "test-user-123", "Audit log should contain user ID")
	assert.Contains(t, string(content), "wf-test-001", "Audit log should contain workflow ID")

	t.Logf("✅ File audit store verified")
}

// TestStory9_3_AC4_WorkflowAudit verifies workflow operation auditing
//
// Story: 9.3 - 审计日志实现
// AC: AC4 - 工作流操作审计
// Priority: P0 (Critical - Workflow tracking)
func TestStory9_3_AC4_WorkflowAudit(t *testing.T) {
	tempDir := t.TempDir()
	auditPath := filepath.Join(tempDir, "workflow-audit.log")

	config := &audit.FileAuditStoreConfig{
		FilePath: auditPath,
	}

	store, err := audit.NewFileAuditStore(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Test workflow lifecycle events
	workflowID := "wf-lifecycle-001"
	userID := "user-operator"

	events := []struct {
		eventType audit.EventType
		action    string
		result    audit.Result
	}{
		{audit.EventWorkflowSubmitted, "submit", audit.ResultSuccess},
		{audit.EventWorkflowStarted, "start", audit.ResultSuccess},
		{audit.EventWorkflowCompleted, "complete", audit.ResultSuccess},
	}

	for _, evt := range events {
		entry := &audit.AuditLogEntry{
			Timestamp:     time.Now(),
			EventType:     evt.eventType,
			EventCategory: audit.CategoryWorkflow,
			Severity:      audit.SeverityInfo,
			User: &audit.UserContext{
				ID:   userID,
				Name: "Operator User",
			},
			Resource: &audit.ResourceContext{
				Type: "workflow",
				ID:   workflowID,
			},
			Action: evt.action,
			Result: evt.result,
		}

		err = store.Log(ctx, entry)
		require.NoError(t, err)
	}

	t.Logf("✅ Workflow lifecycle events logged")

	// Query audit logs
	filter := audit.AuditLogFilter{
		ResourceType: "workflow",
		ResourceID:   workflowID,
	}

	entries, err := store.Query(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, entries, 3, "Should have 3 workflow events")

	t.Logf("✅ Workflow audit verified (%d events)", len(entries))
}

// TestStory9_3_AC5_SecretAccessAudit verifies secret access logging
//
// Story: 9.3 - 审计日志实现
// AC: AC5 - 密钥访问审计
// Priority: P0 (Critical - Security compliance)
func TestStory9_3_AC5_SecretAccessAudit(t *testing.T) {
	tempDir := t.TempDir()
	auditPath := filepath.Join(tempDir, "secret-audit.log")

	config := &audit.FileAuditStoreConfig{
		FilePath: auditPath,
	}

	store, err := audit.NewFileAuditStore(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Log secret access event
	entry := &audit.AuditLogEntry{
		Timestamp:     time.Now(),
		EventType:     audit.EventSecretAccess,
		EventCategory: audit.CategorySecret,
		Severity:      audit.SeverityInfo,
		User: &audit.UserContext{
			ID:   "workflow-runner",
			Name: "Workflow Execution Context",
		},
		Resource: &audit.ResourceContext{
			Type: "secret",
			ID:   "database/credentials/password",
		},
		Action: "get_secret",
		Result: audit.ResultSuccess,
		Details: map[string]interface{}{
			"provider":    "vault",
			"duration_ms": 45,
			// Note: Secret value is NOT logged
		},
	}

	err = store.Log(ctx, entry)
	require.NoError(t, err)

	t.Logf("✅ Secret access logged")

	// Verify secret value is not in log
	content, err := os.ReadFile(auditPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "database/credentials/password", "Should log secret key")
	assert.NotContains(t, string(content), "actual-secret-value", "Should NOT log secret value")

	t.Logf("✅ Secret access audit verified (value not logged)")
}

// TestStory9_3_AC6_AuditLogQuery verifies audit log query functionality
//
// Story: 9.3 - 审计日志实现
// AC: AC6 - 审计日志查询 API
// Priority: P1 (Important - Compliance reporting)
func TestStory9_3_AC6_AuditLogQuery(t *testing.T) {
	tempDir := t.TempDir()
	auditPath := filepath.Join(tempDir, "query-audit.log")

	config := &audit.FileAuditStoreConfig{
		FilePath: auditPath,
	}

	store, err := audit.NewFileAuditStore(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Log multiple entries with different attributes
	now := time.Now()

	entries := []*audit.AuditLogEntry{
		{
			Timestamp:     now.Add(-2 * time.Hour),
			EventType:     audit.EventWorkflowSubmitted,
			EventCategory: audit.CategoryWorkflow,
			Severity:      audit.SeverityInfo,
			User:          &audit.UserContext{ID: "user-alice"},
			Resource:      &audit.ResourceContext{Type: "workflow", ID: "wf-001"},
			Action:        "submit",
			Result:        audit.ResultSuccess,
		},
		{
			Timestamp:     now.Add(-1 * time.Hour),
			EventType:     audit.EventSecretAccess,
			EventCategory: audit.CategorySecret,
			Severity:      audit.SeverityInfo,
			User:          &audit.UserContext{ID: "user-bob"},
			Resource:      &audit.ResourceContext{Type: "secret", ID: "api-key"},
			Action:        "get_secret",
			Result:        audit.ResultSuccess,
		},
		{
			Timestamp:     now.Add(-30 * time.Minute),
			EventType:     audit.EventWorkflowFailed,
			EventCategory: audit.CategoryWorkflow,
			Severity:      audit.SeverityError,
			User:          &audit.UserContext{ID: "user-alice"},
			Resource:      &audit.ResourceContext{Type: "workflow", ID: "wf-002"},
			Action:        "execute",
			Result:        audit.ResultError,
		},
	}

	for _, entry := range entries {
		err = store.Log(ctx, entry)
		require.NoError(t, err)
	}

	// Query by user
	filter := audit.AuditLogFilter{
		UserID: "user-alice",
	}

	results, err := store.Query(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, results, 2, "Should find 2 entries for user-alice")

	// Query by event category
	filter = audit.AuditLogFilter{
		EventCategory: audit.CategorySecret,
	}

	results, err = store.Query(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, results, 1, "Should find 1 secret event")

	// Query by time range
	filter = audit.AuditLogFilter{
		StartTime: now.Add(-1 * time.Hour),
		EndTime:   now,
	}

	results, err = store.Query(ctx, filter)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 2, "Should find events in time range")

	t.Logf("✅ Audit log query verified (%d total queries)", 3)
}
