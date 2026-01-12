package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/audit"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockAuditLogger struct {
	entries []*audit.AuditLogEntry
}

func (m *mockAuditLogger) Log(ctx context.Context, entry *audit.AuditLogEntry) error {
	m.entries = append(m.entries, entry)
	return nil
}

func (m *mockAuditLogger) Query(ctx context.Context, filter audit.AuditLogFilter) ([]*audit.AuditLogEntry, error) {
	return m.entries, nil
}

func (m *mockAuditLogger) Close() error {
	return nil
}

func TestAuditMiddleware(t *testing.T) {
	t.Run("audits API request", func(t *testing.T) {
		mockLogger := &mockAuditLogger{}
		logger, _ := zap.NewDevelopment()

		handler := AuditMiddleware(mockLogger, logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}))

		req := httptest.NewRequest("GET", "/v1/workflows", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		// Wait for async audit logging to complete
		assert.Eventually(t, func() bool {
			return len(mockLogger.entries) == 1
		}, 100*time.Millisecond, 10*time.Millisecond, "audit entry should be logged")

		entry := mockLogger.entries[0]
		assert.Equal(t, audit.EventAPIRequest, entry.EventType)
		assert.Equal(t, audit.CategoryWorkflow, entry.EventCategory)
		assert.Equal(t, audit.ResultSuccess, entry.Result)
		assert.Equal(t, "/v1/workflows", entry.Resource.ID)
	})

	t.Run("skips health check", func(t *testing.T) {
		mockLogger := &mockAuditLogger{}
		logger, _ := zap.NewDevelopment()

		handler := AuditMiddleware(mockLogger, logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}))

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Len(t, mockLogger.entries, 0) // No audit entry
	})

	t.Run("categorizes paths correctly", func(t *testing.T) {
		tests := []struct {
			path     string
			category audit.EventCategory
		}{
			{"/v1/workflows", audit.CategoryWorkflow},
			{"/v1/audit/logs", audit.CategoryAdmin},
			{"/v1/agents", audit.CategoryAgent},
		}

		for _, tt := range tests {
			assert.Equal(t, tt.category, categorizeAPIPath(tt.path))
		}
	})

	t.Run("sets severity from status", func(t *testing.T) {
		tests := []struct {
			code     int
			severity audit.Severity
		}{
			{200, audit.SeverityInfo},
			{400, audit.SeverityWarn},
			{500, audit.SeverityError},
		}

		for _, tt := range tests {
			assert.Equal(t, tt.severity, severityFromStatus(tt.code))
		}
	})

	t.Run("sets result from status", func(t *testing.T) {
		tests := []struct {
			code   int
			result audit.Result
		}{
			{200, audit.ResultSuccess},
			{404, audit.ResultNotFound},
			{403, audit.ResultPermissionDenied},
			{401, audit.ResultPermissionDenied},
			{500, audit.ResultError},
		}

		for _, tt := range tests {
			assert.Equal(t, tt.result, resultFromStatus(tt.code))
		}
	})
}
