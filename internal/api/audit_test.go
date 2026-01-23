package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockAuditLogger is a mock implementation of audit.AuditLogger for testing
type mockAuditLogger struct {
	entries []*audit.AuditLogEntry
}

func (m *mockAuditLogger) Log(ctx context.Context, entry *audit.AuditLogEntry) error {
	m.entries = append(m.entries, entry)
	return nil
}

func (m *mockAuditLogger) Query(ctx context.Context, filter audit.AuditLogFilter) ([]*audit.AuditLogEntry, error) {
	// Simple implementation: return all entries that match filter
	var results []*audit.AuditLogEntry
	for _, e := range m.entries {
		// Apply basic filtering
		if filter.UserID != "" && (e.User == nil || e.User.ID != filter.UserID) {
			continue
		}
		if filter.ResourceType != "" && (e.Resource == nil || e.Resource.Type != filter.ResourceType) {
			continue
		}
		if filter.ResourceID != "" && (e.Resource == nil || e.Resource.ID != filter.ResourceID) {
			continue
		}
		results = append(results, e)
	}

	// Apply pagination
	total := len(results)
	if filter.Offset >= total {
		return []*audit.AuditLogEntry{}, nil
	}
	end := filter.Offset + filter.Limit
	if end > total || filter.Limit == 0 {
		end = total
	}

	return results[filter.Offset:end], nil
}

func (m *mockAuditLogger) Close() error {
	return nil
}

func newMockAuditLogger() *mockAuditLogger {
	return &mockAuditLogger{
		entries: []*audit.AuditLogEntry{
			{
				Timestamp:     time.Now().Add(-1 * time.Hour),
				EventType:     "workflow.submitted",
				EventCategory: audit.CategoryWorkflow,
				Severity:      audit.SeverityInfo,
				User:          &audit.UserContext{ID: "user-123", Name: "test-user"},
				Resource:      &audit.ResourceContext{Type: "workflow", ID: "wf-001"},
				Action:        "submit",
				Result:        audit.ResultSuccess,
			},
			{
				Timestamp:     time.Now().Add(-30 * time.Minute),
				EventType:     "workflow.cancelled",
				EventCategory: audit.CategoryWorkflow,
				Severity:      audit.SeverityInfo,
				User:          &audit.UserContext{ID: "user-456", Name: "other-user"},
				Resource:      &audit.ResourceContext{Type: "workflow", ID: "wf-002"},
				Action:        "cancel",
				Result:        audit.ResultSuccess,
			},
			{
				Timestamp:     time.Now().Add(-15 * time.Minute),
				EventType:     "secret.accessed",
				EventCategory: audit.CategorySecret,
				Severity:      audit.SeverityInfo,
				User:          &audit.UserContext{ID: "user-123", Name: "test-user"},
				Resource:      &audit.ResourceContext{Type: "secret", ID: "sec-001"},
				Action:        "access",
				Result:        audit.ResultSuccess,
			},
		},
	}
}

func TestAuditHandler_NewAuditHandler(t *testing.T) {
	logger := zap.NewNop()
	mockLogger := newMockAuditLogger()

	h := NewAuditHandler(logger, mockLogger)

	assert.NotNil(t, h)
	assert.NotNil(t, h.logger)
	assert.NotNil(t, h.auditLogger)
}

func TestAuditHandler_QueryLogs(t *testing.T) {
	logger := zap.NewNop()
	mockLogger := newMockAuditLogger()
	h := NewAuditHandler(logger, mockLogger)

	t.Run("query all logs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/audit/logs", nil)
		w := httptest.NewRecorder()

		h.QueryLogs(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response QueryLogsResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, 3, response.Total)
		assert.NotNil(t, response.Entries)
	})

	t.Run("query with user_id filter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/audit/logs?user_id=user-123", nil)
		w := httptest.NewRecorder()

		h.QueryLogs(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response QueryLogsResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, 2, response.Total)
		for _, entry := range response.Entries {
			assert.Equal(t, "user-123", entry.User.ID)
		}
	})

	t.Run("query with resource_type filter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/audit/logs?resource_type=workflow", nil)
		w := httptest.NewRecorder()

		h.QueryLogs(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response QueryLogsResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, 2, response.Total)
	})

	t.Run("query with pagination", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/audit/logs?limit=2&offset=0", nil)
		w := httptest.NewRecorder()

		h.QueryLogs(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response QueryLogsResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.LessOrEqual(t, len(response.Entries), 2)
		assert.Equal(t, 2, response.Limit)
		assert.Equal(t, 0, response.Offset)
	})

	t.Run("query with time range", func(t *testing.T) {
		startTime := time.Now().Add(-2 * time.Hour).Format(time.RFC3339)
		endTime := time.Now().Format(time.RFC3339)
		req := httptest.NewRequest(http.MethodGet, "/v1/audit/logs?start_time="+startTime+"&end_time="+endTime, nil)
		w := httptest.NewRecorder()

		h.QueryLogs(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("query with event_type filter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/audit/logs?event_type=workflow.submitted", nil)
		w := httptest.NewRecorder()

		h.QueryLogs(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("query with event_category filter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/audit/logs?event_category=workflow", nil)
		w := httptest.NewRecorder()

		h.QueryLogs(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("query with result filter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/audit/logs?result=success", nil)
		w := httptest.NewRecorder()

		h.QueryLogs(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestParseIntParam(t *testing.T) {
	tests := []struct {
		name         string
		queryParam   string
		queryValue   string
		defaultValue int
		expected     int
	}{
		{"empty value returns default", "limit", "", 10, 10},
		{"valid number", "limit", "5", 10, 5},
		{"invalid number returns default", "limit", "invalid", 10, 10},
		{"negative number", "limit", "-1", 10, -1},
		{"zero", "limit", "0", 10, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			url := "/test"
			if tc.queryValue != "" {
				url = "/test?" + tc.queryParam + "=" + tc.queryValue
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			result := parseIntParam(req, tc.queryParam, tc.defaultValue)
			assert.Equal(t, tc.expected, result)
		})
	}
}
