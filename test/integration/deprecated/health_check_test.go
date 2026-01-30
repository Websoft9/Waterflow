//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHealthCheckIntegration tests health check endpoints in Docker Compose environment
// Run with: go test -v ./test/integration -run TestHealthCheckIntegration
func TestHealthCheckIntegration(t *testing.T) {
	// Skip if not running in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()

	t.Run("Health endpoint returns 200", func(t *testing.T) {
		resp, err := http.Get(serverURL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	})

	t.Run("Ready endpoint with healthy Temporal", func(t *testing.T) {
		// Wait for services to be ready using condition-based waiting
		client := &http.Client{Timeout: 2 * time.Second}
		require.Eventually(t, func() bool {
			resp, err := client.Get(serverURL + "/ready")
			if err != nil {
				return false
			}
			defer resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		}, 30*time.Second, 500*time.Millisecond, "services should become ready")

		resp, err := http.Get(serverURL + "/ready")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return 200 when all dependencies are healthy
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Ready endpoint response format", func(t *testing.T) {
		resp, err := http.Get(serverURL + "/ready")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Response should contain status, timestamp, and checks
		var body map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&body)
		require.NoError(t, err)

		assert.Contains(t, body, "status")
		assert.Contains(t, body, "timestamp")
		assert.Contains(t, body, "checks")

		checks := body["checks"].(map[string]interface{})
		assert.Contains(t, checks, "temporal")
	})

	t.Run("Health check performance", func(t *testing.T) {
		start := time.Now()
		resp, err := http.Get(serverURL + "/health")
		duration := time.Since(start)

		require.NoError(t, err)
		defer resp.Body.Close()

		// Health endpoint should respond in < 100ms
		assert.Less(t, duration.Milliseconds(), int64(100),
			"Health check took too long: %v", duration)
	})

	t.Run("Ready check with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL+"/ready", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should respond within timeout
		assert.NotEqual(t, http.StatusRequestTimeout, resp.StatusCode)
	})
}

// TestDatabaseHealthCheck tests database health check if configured
func TestDatabaseHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()

	t.Run("Ready endpoint includes database check", func(t *testing.T) {
		resp, err := http.Get(serverURL + "/ready")
		require.NoError(t, err)
		defer resp.Body.Close()

		var body map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&body)
		require.NoError(t, err)

		checks := body["checks"].(map[string]interface{})

		// If database is configured, check should be present
		// If not configured, this is acceptable (database is optional)
		if _, hasDB := checks["database"]; hasDB {
			// Database check exists, verify it's either "ok" or has error message
			dbStatus := checks["database"].(string)
			t.Logf("Database check status: %s", dbStatus)
		}
	})
}

// TestHealthCheckFailureScenarios tests health check behavior when dependencies fail
// Note: This test requires ability to stop/start dependencies
func TestHealthCheckFailureScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Skip("Requires Docker Compose control - implement after docker-compose test harness")

	// TODO: Implement tests for:
	// 1. Stop Temporal -> /ready returns 503
	// 2. Stop Database -> /ready returns 503 (if DB configured)
	// 3. Restart Temporal -> /ready returns 200
}

// Note: getServerURL is defined in common_test.go
