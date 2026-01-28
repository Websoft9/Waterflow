package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestHandlers_Health(t *testing.T) {
	logger := zap.NewNop()
	h := NewHandlers(logger, "v1.0.0", "abc123", "2025-12-19")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.NotEmpty(t, response["timestamp"])
}

func TestHandlers_Ready(t *testing.T) {
	logger := zap.NewNop()
	h := NewHandlers(logger, "v1.0.0", "abc123", "2025-12-19")

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	h.Ready(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "ready", response["status"])
	assert.NotEmpty(t, response["timestamp"])
	assert.NotNil(t, response["checks"])

	checks := response["checks"].(map[string]interface{})
	assert.Equal(t, "not_configured", checks["temporal"])
}

func TestHandlers_Version(t *testing.T) {
	logger := zap.NewNop()
	h := NewHandlers(logger, "v1.2.3", "abc123", "2025-12-19_10:30:00")

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	h.Version(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "v1.2.3", response["version"])
	assert.Equal(t, "abc123", response["commit"])
	assert.Equal(t, "2025-12-19_10:30:00", response["build_time"])
	assert.NotEmpty(t, response["go_version"])
}

func TestHandlers_NotFound(t *testing.T) {
	logger := zap.NewNop()
	h := NewHandlers(logger, "v1.0.0", "abc123", "2025-12-19")

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	h.NotFound(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))

	var response ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	// Implementation uses error type as RFC 7807 type field
	assert.Equal(t, "not_found", response.Type)
	assert.Equal(t, "Resource Not Found", response.Title)
	assert.Equal(t, http.StatusNotFound, response.Status)
	assert.Equal(t, "/nonexistent", response.Instance)
}

func TestHandlers_GetWorkflowSchema(t *testing.T) {
	logger := zap.NewNop()
	h := NewHandlers(logger, "v1.0.0", "abc123", "2025-12-19")

	req := httptest.NewRequest(http.MethodGet, "/schema/workflow.json", nil)
	w := httptest.NewRecorder()

	h.GetWorkflowSchema(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var schema map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&schema)
	require.NoError(t, err)

	// 验证 JSON Schema 结构
	assert.Equal(t, "http://json-schema.org/draft-07/schema#", schema["$schema"])
	assert.Equal(t, "Waterflow Workflow Schema", schema["title"])
	assert.Contains(t, schema["description"], "API-driven architecture")

	// 验证必填字段
	required, ok := schema["required"].([]interface{})
	require.True(t, ok)
	assert.Contains(t, required, "name")
	assert.Contains(t, required, "jobs")

	// 验证不包含已移除的 'on' 字段
	properties, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok)
	_, hasOnField := properties["on"]
	assert.False(t, hasOnField, "Schema should not contain 'on' field in API-driven architecture")
}

func TestHandlers_MethodNotAllowed(t *testing.T) {
	logger := zap.NewNop()
	h := NewHandlers(logger, "v1.0.0", "abc123", "2025-12-19")

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()

	h.MethodNotAllowed(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))

	var response ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	// Implementation uses error type as RFC 7807 type field
	assert.Equal(t, "method_not_allowed", response.Type)
	assert.Equal(t, "Method Not Allowed", response.Title)
	assert.Equal(t, http.StatusMethodNotAllowed, response.Status)
	assert.Equal(t, "/health", response.Instance)
}

func TestHandlers_Metrics(t *testing.T) {
	logger := zap.NewNop()
	h := NewHandlers(logger, "v1.0.0", "abc123", "2025-12-19")

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	h.Metrics(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Prometheus metrics contain HELP and TYPE comments
	assert.Contains(t, w.Body.String(), "# HELP")
	assert.Contains(t, w.Body.String(), "# TYPE")
}
