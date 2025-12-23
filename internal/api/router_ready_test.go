package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Websoft9/waterflow/pkg/config"
	"github.com/Websoft9/waterflow/pkg/temporal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestRouterWithTemporalClient_ReadyEndpoint(t *testing.T) {
	logger := zap.NewNop()

	// Create Temporal config
	cfg := &config.TemporalConfig{
		Host:          "localhost:7233",
		Namespace:     "default",
		MaxRetries:    1,
		RetryInterval: 0,
	}

	// Try to create client (may fail if Temporal not running)
	temporalClient, err := temporal.NewClient(cfg, logger)
	if err != nil {
		t.Skip("Temporal server not available, skipping integration test")
		return
	}
	defer temporalClient.Close()

	router := NewRouter(logger, temporalClient, "v1.0.0", "abc123", "2025-12-19")

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return 200 if Temporal is healthy
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "checks")
	assert.Contains(t, w.Body.String(), "temporal")
}

func TestRouterWithoutTemporalClient_ReadyEndpoint(t *testing.T) {
	logger := zap.NewNop()
	router := NewRouter(logger, nil, "v1.0.0", "abc123", "2025-12-19")

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should still return 200 with fallback handler
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ready")
	assert.Contains(t, w.Body.String(), "temporal")
}

func TestRouterWithTemporalClient_WorkflowEndpoints(t *testing.T) {
	logger := zap.NewNop()

	cfg := &config.TemporalConfig{
		Host:          "localhost:7233",
		Namespace:     "default",
		MaxRetries:    1,
		RetryInterval: 0,
	}

	temporalClient, err := temporal.NewClient(cfg, logger)
	if err != nil {
		t.Skip("Temporal server not available, skipping integration test")
		return
	}
	defer temporalClient.Close()

	router := NewRouter(logger, temporalClient, "v1.0.0", "abc123", "2025-12-19")

	// Test that basic workflow endpoints respond (not 404)
	tests := []struct {
		method   string
		endpoint string
	}{
		{http.MethodPost, "/v1/workflows"},
		{http.MethodGet, "/v1/workflows"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.endpoint, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Endpoint should be registered (not 404)
		// May return 400/500 due to missing body, but should not be 404
		assert.NotEqual(t, http.StatusNotFound, w.Code, "Endpoint %s %s should be registered", tt.method, tt.endpoint)
	}
}

func TestRouterWithoutTemporalClient_WorkflowEndpoints(t *testing.T) {
	logger := zap.NewNop()
	router := NewRouter(logger, nil, "v1.0.0", "abc123", "2025-12-19")

	// Test that workflow endpoints are NOT registered without Temporal client
	endpoints := []string{
		"/v1/workflows",
		"/v1/workflows/test-id",
	}

	for _, endpoint := range endpoints {
		req := httptest.NewRequest(http.MethodPost, endpoint, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Should return 404 (endpoints not registered)
		assert.Equal(t, http.StatusNotFound, w.Code, "Endpoint %s should not be registered without Temporal", endpoint)
	}
}
