package audit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileAuditStore(t *testing.T) {
	t.Run("create with valid config", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := &FileAuditStoreConfig{
			Path:       tmpDir,
			MaxSize:    50,
			MaxAge:     30,
			MaxBackups: 10,
			Compress:   true,
		}

		store, err := NewFileAuditStore(config)
		require.NoError(t, err)
		require.NotNil(t, store)
		defer func() { _ = store.Close() }()

		// Verify directory was created
		_, err = os.Stat(tmpDir)
		assert.NoError(t, err)
	})

	t.Run("create with defaults", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := &FileAuditStoreConfig{
			Path: tmpDir,
		}

		store, err := NewFileAuditStore(config)
		require.NoError(t, err)
		require.NotNil(t, store)
		defer func() { _ = store.Close() }()

		// Defaults should be applied
		assert.Equal(t, 100, store.writer.MaxSize)
		assert.Equal(t, 90, store.writer.MaxAge)
		assert.Equal(t, 30, store.writer.MaxBackups)
	})

	t.Run("error on nil config", func(t *testing.T) {
		store, err := NewFileAuditStore(nil)
		assert.Error(t, err)
		assert.Nil(t, store)
		assert.Contains(t, err.Error(), "config is required")
	})

	t.Run("error on empty path", func(t *testing.T) {
		config := &FileAuditStoreConfig{
			Path: "",
		}
		store, err := NewFileAuditStore(config)
		assert.Error(t, err)
		assert.Nil(t, store)
		assert.Contains(t, err.Error(), "path is required")
	})
}

func TestFileAuditStore_Log(t *testing.T) {
	tmpDir := t.TempDir()
	config := &FileAuditStoreConfig{
		Path: tmpDir,
	}

	store, err := NewFileAuditStore(config)
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	t.Run("log simple entry", func(t *testing.T) {
		entry := NewAuditLogEntry(EventWorkflowSubmit, CategoryWorkflow).
			WithUser(&UserContext{ID: "user1", Name: "Alice"}).
			WithResource(&ResourceContext{Type: "workflow", ID: "wf1"}).
			WithAction("submit").
			WithResult(ResultSuccess)

		err := store.Log(context.Background(), entry)
		assert.NoError(t, err)

		// Verify file exists
		logFile := filepath.Join(tmpDir, "audit.log")
		_, err = os.Stat(logFile)
		assert.NoError(t, err)
	})

	t.Run("log sets timestamp", func(t *testing.T) {
		entry := &AuditLogEntry{
			EventType:     EventWorkflowCancel,
			EventCategory: CategoryWorkflow,
			Action:        "cancel",
			Result:        ResultSuccess,
		}

		// Timestamp should be zero
		assert.True(t, entry.Timestamp.IsZero())

		err := store.Log(context.Background(), entry)
		assert.NoError(t, err)

		// After logging, entry timestamp should be set
		assert.False(t, entry.Timestamp.IsZero())
	})

	t.Run("log multiple entries", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			entry := NewAuditLogEntry(EventWorkflowSubmit, CategoryWorkflow).
				WithAction("submit").
				WithResult(ResultSuccess).
				WithDetails(map[string]interface{}{"index": i})

			err := store.Log(context.Background(), entry)
			assert.NoError(t, err)
		}
	})

	t.Run("error when store is closed", func(t *testing.T) {
		tmpDir2 := t.TempDir()
		config2 := &FileAuditStoreConfig{Path: tmpDir2}
		store2, _ := NewFileAuditStore(config2)
		_ = store2.Close()

		entry := NewAuditLogEntry(EventWorkflowSubmit, CategoryWorkflow).
			WithAction("submit").
			WithResult(ResultSuccess)

		err := store2.Log(context.Background(), entry)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "closed")
	})
}

func TestFileAuditStore_Query(t *testing.T) {
	tmpDir := t.TempDir()
	config := &FileAuditStoreConfig{Path: tmpDir}
	store, err := NewFileAuditStore(config)
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	// Log test entries
	now := time.Now().UTC()
	entries := []*AuditLogEntry{
		NewAuditLogEntry(EventWorkflowSubmit, CategoryWorkflow).
			WithUser(&UserContext{ID: "user1"}).
			WithResource(&ResourceContext{Type: "workflow", ID: "wf1"}).
			WithAction("submit").
			WithResult(ResultSuccess),
		NewAuditLogEntry(EventSecretAccess, CategorySecret).
			WithUser(&UserContext{ID: "user2"}).
			WithResource(&ResourceContext{Type: "secret", ID: "secret1"}).
			WithAction("access").
			WithResult(ResultSuccess),
		NewAuditLogEntry(EventWorkflowCancel, CategoryWorkflow).
			WithUser(&UserContext{ID: "user1"}).
			WithResource(&ResourceContext{Type: "workflow", ID: "wf2"}).
			WithAction("cancel").
			WithResult(ResultSuccess),
	}

	for _, entry := range entries {
		entry.Timestamp = now
		err := store.Log(context.Background(), entry)
		require.NoError(t, err)
	}

	t.Run("query all entries", func(t *testing.T) {
		results, err := store.Query(context.Background(), AuditLogFilter{})
		assert.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("query by event category", func(t *testing.T) {
		cat := CategoryWorkflow
		results, err := store.Query(context.Background(), AuditLogFilter{
			EventCategory: &cat,
		})
		assert.NoError(t, err)
		assert.Len(t, results, 2)
		for _, r := range results {
			assert.Equal(t, CategoryWorkflow, r.EventCategory)
		}
	})

	t.Run("query by user ID", func(t *testing.T) {
		results, err := store.Query(context.Background(), AuditLogFilter{
			UserID: "user1",
		})
		assert.NoError(t, err)
		assert.Len(t, results, 2)
		for _, r := range results {
			assert.Equal(t, "user1", r.User.ID)
		}
	})

	t.Run("query by resource type", func(t *testing.T) {
		results, err := store.Query(context.Background(), AuditLogFilter{
			ResourceType: "secret",
		})
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "secret", results[0].Resource.Type)
	})

	t.Run("query with limit", func(t *testing.T) {
		results, err := store.Query(context.Background(), AuditLogFilter{
			Limit: 2,
		})
		assert.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("query with offset", func(t *testing.T) {
		results, err := store.Query(context.Background(), AuditLogFilter{
			Offset: 1,
		})
		assert.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("query with limit and offset", func(t *testing.T) {
		results, err := store.Query(context.Background(), AuditLogFilter{
			Limit:  1,
			Offset: 1,
		})
		assert.NoError(t, err)
		assert.Len(t, results, 1)
	})

	t.Run("query empty file", func(t *testing.T) {
		tmpDir2 := t.TempDir()
		config2 := &FileAuditStoreConfig{Path: tmpDir2}
		store2, _ := NewFileAuditStore(config2)
		defer func() { _ = store2.Close() }()

		results, err := store2.Query(context.Background(), AuditLogFilter{})
		assert.NoError(t, err)
		assert.Len(t, results, 0)
	})
}

func TestFileAuditStore_Close(t *testing.T) {
	t.Run("close successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := &FileAuditStoreConfig{Path: tmpDir}
		store, err := NewFileAuditStore(config)
		require.NoError(t, err)

		err = store.Close()
		assert.NoError(t, err)
		assert.True(t, store.closed)
	})

	t.Run("close twice is idempotent", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := &FileAuditStoreConfig{Path: tmpDir}
		store, err := NewFileAuditStore(config)
		require.NoError(t, err)

		err = store.Close()
		assert.NoError(t, err)

		err = store.Close()
		assert.NoError(t, err)
	})
}

func TestMatchesFilter(t *testing.T) {
	now := time.Now().UTC()
	before := now.Add(-1 * time.Hour)
	after := now.Add(1 * time.Hour)

	entry := &AuditLogEntry{
		Timestamp:     now,
		EventType:     EventWorkflowSubmit,
		EventCategory: CategoryWorkflow,
		User:          &UserContext{ID: "user1"},
		Resource:      &ResourceContext{Type: "workflow", ID: "wf1"},
		Result:        ResultSuccess,
	}

	t.Run("matches empty filter", func(t *testing.T) {
		assert.True(t, matchesFilter(entry, AuditLogFilter{}))
	})

	t.Run("matches start time", func(t *testing.T) {
		assert.True(t, matchesFilter(entry, AuditLogFilter{StartTime: &before}))
		assert.False(t, matchesFilter(entry, AuditLogFilter{StartTime: &after}))
	})

	t.Run("matches end time", func(t *testing.T) {
		assert.True(t, matchesFilter(entry, AuditLogFilter{EndTime: &after}))
		assert.False(t, matchesFilter(entry, AuditLogFilter{EndTime: &before}))
	})

	t.Run("matches event types", func(t *testing.T) {
		assert.True(t, matchesFilter(entry, AuditLogFilter{
			EventTypes: []string{EventWorkflowSubmit, EventWorkflowCancel},
		}))
		assert.False(t, matchesFilter(entry, AuditLogFilter{
			EventTypes: []string{EventSecretAccess},
		}))
	})

	t.Run("matches event category", func(t *testing.T) {
		cat := CategoryWorkflow
		assert.True(t, matchesFilter(entry, AuditLogFilter{EventCategory: &cat}))

		catSecret := CategorySecret
		assert.False(t, matchesFilter(entry, AuditLogFilter{EventCategory: &catSecret}))
	})

	t.Run("matches result", func(t *testing.T) {
		result := ResultSuccess
		assert.True(t, matchesFilter(entry, AuditLogFilter{Result: &result}))

		resultFailure := ResultFailure
		assert.False(t, matchesFilter(entry, AuditLogFilter{Result: &resultFailure}))
	})
}
