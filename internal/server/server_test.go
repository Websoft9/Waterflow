package server

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/config"
	"github.com/Websoft9/waterflow/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerStartAndShutdown(t *testing.T) {
	// Initialize logger for tests
	err := logger.Init("info", "json")
	require.NoError(t, err)

	// Create test config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:            "localhost",
			Port:            18080,
			ReadTimeout:     10 * time.Second,
			WriteTimeout:    10 * time.Second,
			ShutdownTimeout: 5 * time.Second,
		},
	}

	// Create server
	srv := New(cfg, logger.Log, "v1.0.0-test", "abc123", "2025-12-19")
	assert.NotNil(t, srv)

	// Start server in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Test health endpoint
	resp, err := http.Get("http://localhost:18080/health")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	if err := resp.Body.Close(); err != nil {
		t.Logf("failed to close response body: %v", err)
	}

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = srv.Shutdown(ctx)
	assert.NoError(t, err)

	// Verify server stopped
	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("Server did not stop in time")
	}
}

func TestHealthEndpoint(t *testing.T) {
	err := logger.Init("info", "json")
	require.NoError(t, err)

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:            "localhost",
			Port:            18081,
			ReadTimeout:     10 * time.Second,
			WriteTimeout:    10 * time.Second,
			ShutdownTimeout: 5 * time.Second,
		},
	}

	srv := New(cfg, logger.Log, "v1.0.0-test", "abc123", "2025-12-19")

	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("server stopped: %v", err)
		}
	}()
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://localhost:18081/health")
	require.NoError(t, err)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("failed to close response body: %v", err)
		}
	}()
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Logf("server shutdown failed: %v", err)
	}
}
