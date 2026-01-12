package audit

import (
	"context"
	"testing"

	"github.com/Websoft9/waterflow/pkg/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditSecretProvider_GetSecret(t *testing.T) {
	tmpDir := t.TempDir()
	auditStore, err := NewFileAuditStore(&FileAuditStoreConfig{Path: tmpDir})
	require.NoError(t, err)
	defer func() { _ = auditStore.Close() }()

	mockProvider := secrets.NewMockSecretProvider(map[string]string{
		"api_key": "secret123",
	})

	auditedProvider := NewAuditSecretProvider(mockProvider, auditStore)

	t.Run("audit successful access", func(t *testing.T) {
		value, err := auditedProvider.GetSecret(context.Background(), "api_key")
		assert.NoError(t, err)
		assert.Equal(t, "secret123", value)

		// Verify audit log
		logs, err := auditStore.Query(context.Background(), AuditLogFilter{
			EventTypes: []string{EventSecretAccess},
		})
		assert.NoError(t, err)
		assert.Len(t, logs, 1)
		assert.Equal(t, ResultSuccess, logs[0].Result)
		assert.Equal(t, "get_secret", logs[0].Action)
		assert.Equal(t, "secret", logs[0].Resource.Type)
		assert.Equal(t, "api_key", logs[0].Resource.Name)
	})

	t.Run("audit not found", func(t *testing.T) {
		_, err := auditedProvider.GetSecret(context.Background(), "nonexistent")
		assert.Error(t, err)

		// Verify audit log
		logs, err := auditStore.Query(context.Background(), AuditLogFilter{
			EventTypes: []string{EventSecretAccess},
		})
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(logs), 2) // At least 2 (success + not_found)

		// Find the not_found log
		var notFoundLog *AuditLogEntry
		for _, log := range logs {
			if log.Result == ResultNotFound {
				notFoundLog = log
				break
			}
		}
		require.NotNil(t, notFoundLog)
		assert.Equal(t, SeverityWarn, notFoundLog.Severity)
		assert.Equal(t, "nonexistent", notFoundLog.Resource.Name)
	})
}

func TestAuditSecretProvider_GetSecretMap(t *testing.T) {
	tmpDir := t.TempDir()
	auditStore, err := NewFileAuditStore(&FileAuditStoreConfig{Path: tmpDir})
	require.NoError(t, err)
	defer func() { _ = auditStore.Close() }()

	mockProvider := secrets.NewMockSecretProvider(map[string]string{
		"api_key":     "secret123",
		"db_password": "pass456",
	})

	auditedProvider := NewAuditSecretProvider(mockProvider, auditStore)

	t.Run("audit batch access", func(t *testing.T) {
		prefix := ""
		results, err := auditedProvider.GetSecretMap(context.Background(), prefix)
		assert.NoError(t, err)
		assert.Len(t, results, 2) // All keys

		// Verify audit log
		logs, err := auditStore.Query(context.Background(), AuditLogFilter{
			EventTypes: []string{EventSecretList},
		})
		assert.NoError(t, err)
		assert.Len(t, logs, 1)
		assert.Equal(t, ResultSuccess, logs[0].Result)
		assert.Equal(t, "get_secret_map", logs[0].Action)
		assert.Equal(t, float64(2), logs[0].Details["retrieved_count"])
	})
}

func TestAuditSecretProvider_Close(t *testing.T) {
	tmpDir := t.TempDir()
	auditStore, err := NewFileAuditStore(&FileAuditStoreConfig{Path: tmpDir})
	require.NoError(t, err)
	defer func() { _ = auditStore.Close() }()

	mockProvider := secrets.NewMockSecretProvider(map[string]string{})
	auditedProvider := NewAuditSecretProvider(mockProvider, auditStore)

	err = auditedProvider.Close()
	assert.NoError(t, err)
}
