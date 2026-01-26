package api

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/config"
	"github.com/Websoft9/waterflow/pkg/temporal"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	router := NewRouter(logger, temporalClient, nil, "v1.0.0", "abc123", "2025-12-19")

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
	router := NewRouter(logger, nil, nil, "v1.0.0", "abc123", "2025-12-19")

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

	router := NewRouter(logger, temporalClient, nil, "v1.0.0", "abc123", "2025-12-19")

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
	router := NewRouter(logger, nil, nil, "v1.0.0", "abc123", "2025-12-19")

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

// TestReadyEndpoint_WithDatabase tests /ready endpoint with database health check (Story 8-4 AC2)
func TestReadyEndpoint_WithDatabase(t *testing.T) {
	logger := zap.NewNop()

	// Create in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("Failed to close database: %v", err)
		}
	}()

	// Create health config with custom timeouts
	cfg := &config.Config{
		Server: config.ServerConfig{
			Health: config.HealthConfig{
				TemporalTimeout: 2 * time.Second,
				DatabaseTimeout: 1 * time.Second,
			},
		},
	}

	router := NewRouterWithDB(logger, nil, nil, db, cfg, "v1.0.0", "abc123", "2025-12-19", nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Database should be healthy (in-memory DB is always available)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"database":"ok"`)
	assert.Contains(t, w.Body.String(), `"status":"ready"`)
}

// TestReadyEndpoint_WithClosedDatabase tests 503 response when database is unavailable (Story 8-4 AC3)
func TestReadyEndpoint_WithClosedDatabase(t *testing.T) {
	logger := zap.NewNop()

	// Create and immediately close database to simulate failure
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}

	router := NewRouterWithDB(logger, nil, nil, db, nil, "v1.0.0", "abc123", "2025-12-19", nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Story 8-4 AC3: Should return 503 when database is unavailable
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"not_ready"`)
	assert.Contains(t, w.Body.String(), `"database"`)
}

// TestReadyEndpoint_ConfigurableTimeouts tests custom health check timeouts (Story 8-4 AC6)
func TestReadyEndpoint_ConfigurableTimeouts(t *testing.T) {
	logger := zap.NewNop()

	// Create in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("Failed to close database: %v", err)
		}
	}()

	// Config with very short timeout (100ms)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Health: config.HealthConfig{
				DatabaseTimeout: 100 * time.Millisecond,
			},
		},
	}

	router := NewRouterWithDB(logger, nil, nil, db, cfg, "v1.0.0", "abc123", "2025-12-19", nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Even with short timeout, in-memory DB should respond quickly
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"database":"ok"`)
}
