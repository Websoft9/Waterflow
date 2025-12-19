package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewRouter(t *testing.T) {
	logger := zap.NewNop()
	router := NewRouter(logger, "v1.0.0", "abc123", "2025-12-19")
	require.NotNil(t, router)

	// Should be a mux.Router
	_, ok := router.(*mux.Router)
	assert.True(t, ok, "Router should be *mux.Router")
}

func TestRouterHealthEndpoint(t *testing.T) {
	logger := zap.NewNop()
	router := NewRouter(logger, "v1.0.0", "abc123", "2025-12-19")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
	assert.Contains(t, w.Body.String(), "timestamp")
}

func TestRouterReadyEndpoint(t *testing.T) {
	logger := zap.NewNop()
	router := NewRouter(logger, "v1.0.0", "abc123", "2025-12-19")

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Initially ready without Temporal
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "checks")
	assert.Contains(t, w.Body.String(), "temporal")
}

func TestRouterVersionEndpoint(t *testing.T) {
	logger := zap.NewNop()
	router := NewRouter(logger, "v1.0.0", "abc123", "2025-12-19")

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "version")
	assert.Contains(t, w.Body.String(), "go_version")
}

func TestRouterMetricsEndpoint(t *testing.T) {
	logger := zap.NewNop()
	router := NewRouter(logger, "v1.0.0", "abc123", "2025-12-19")

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Prometheus metrics format
	assert.Contains(t, w.Body.String(), "# HELP")
}

func TestRouterNotFoundHandler(t *testing.T) {
	logger := zap.NewNop()
	router := NewRouter(logger, "v1.0.0", "abc123", "2025-12-19")

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	// Should return RFC 7807 format
	assert.Contains(t, w.Body.String(), "type")
	assert.Contains(t, w.Body.String(), "title")
	assert.Contains(t, w.Body.String(), "status")
}

func TestRouterMethodNotAllowedHandler(t *testing.T) {
	logger := zap.NewNop()
	router := NewRouter(logger, "v1.0.0", "abc123", "2025-12-19")

	// POST to health (only GET allowed)
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	// Should return RFC 7807 format
	assert.Contains(t, w.Body.String(), "type")
	assert.Contains(t, w.Body.String(), "title")
	assert.Contains(t, w.Body.String(), "status")
}
