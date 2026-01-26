//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_API_Validation tests the validation endpoint
func TestIntegration_API_Validation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Validate valid workflow", func(t *testing.T) {
		yaml := `
name: valid-workflow
on: workflow_dispatch
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Test
        uses: shell@v1
        with:
          command: echo hello
`
		reqBody, _ := json.Marshal(map[string]string{"yaml": yaml})
		req, err := http.NewRequestWithContext(ctx, "POST", serverURL+"/v1/validate", bytes.NewReader(reqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Validate invalid workflow", func(t *testing.T) {
		yaml := `
name: invalid
# Missing required fields
`
		reqBody, _ := json.Marshal(map[string]string{"yaml": yaml})
		req, err := http.NewRequestWithContext(ctx, "POST", serverURL+"/v1/validate", bytes.NewReader(reqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return 400 or 422 for validation errors
		assert.True(t, resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnprocessableEntity,
			"Expected 400 or 422, got %d", resp.StatusCode)
	})

	t.Run("Validate malformed YAML", func(t *testing.T) {
		yaml := `
name: test
jobs:
  - invalid yaml structure
    indentation: broken
`
		reqBody, _ := json.Marshal(map[string]string{"yaml": yaml})
		req, err := http.NewRequestWithContext(ctx, "POST", serverURL+"/v1/validate", bytes.NewReader(reqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.True(t, resp.StatusCode >= 400, "Should return error status")
	})
}

// TestIntegration_API_Nodes tests the nodes endpoint
func TestIntegration_API_Nodes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("List all nodes", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET", serverURL+"/v1/nodes", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var nodes []map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&nodes)
		require.NoError(t, err)

		// Should have at least some built-in nodes
		assert.NotEmpty(t, nodes, "Should have registered nodes")

		// Check for expected nodes
		nodeNames := make([]string, 0)
		for _, node := range nodes {
			if name, ok := node["name"].(string); ok {
				nodeNames = append(nodeNames, name)
			}
		}

		t.Logf("Registered nodes: %v", nodeNames)
	})
}

// TestIntegration_API_WorkflowList tests workflow listing
func TestIntegration_API_WorkflowList(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Submit a workflow first
	yaml := `
name: list-test
on: workflow_dispatch
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Quick task
        uses: shell@v1
        with:
          command: echo "test"
`
	_, err := submitWorkflow(ctx, serverURL, yaml)
	require.NoError(t, err)

	t.Run("List workflows", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET", serverURL+"/v1/workflows", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return 200 with workflow list
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		t.Logf("Workflow list response: %s", string(body))
	})

	t.Run("List with pagination", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET", serverURL+"/v1/workflows?page_size=10", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestIntegration_API_HealthEndpoints tests health-related endpoints
func TestIntegration_API_HealthEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	endpoints := []struct {
		name     string
		path     string
		wantCode int
	}{
		{"Health", "/health", http.StatusOK},
		{"Ready", "/ready", http.StatusOK},
		{"Version", "/version", http.StatusOK},
	}

	for _, ep := range endpoints {
		t.Run(ep.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(ctx, "GET", serverURL+ep.path, nil)
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, ep.wantCode, resp.StatusCode, "Endpoint %s returned unexpected status", ep.path)

			// All health endpoints should return JSON
			assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
		})
	}
}

// TestIntegration_API_WorkflowRerun tests workflow rerun functionality
func TestIntegration_API_WorkflowRerun(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Submit and wait for a workflow to complete
	yaml := `
name: rerun-test
on: workflow_dispatch
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Quick task
        uses: shell@v1
        with:
          command: echo "original run"
`
	resp, err := submitWorkflow(ctx, serverURL, yaml)
	require.NoError(t, err)

	// Wait for completion
	_, err = waitForWorkflowCompletion(ctx, serverURL, resp.GetID(), 60*time.Second)
	require.NoError(t, err)

	t.Run("Rerun completed workflow", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "POST", serverURL+"/v1/workflows/"+resp.GetID()+"/rerun", nil)
		require.NoError(t, err)

		rerunResp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer rerunResp.Body.Close()

		// Should accept rerun request
		assert.True(t, rerunResp.StatusCode == http.StatusOK || rerunResp.StatusCode == http.StatusAccepted,
			"Expected 200 or 202, got %d", rerunResp.StatusCode)

		if rerunResp.StatusCode == http.StatusOK || rerunResp.StatusCode == http.StatusAccepted {
			var result WorkflowResponse
			err = json.NewDecoder(rerunResp.Body).Decode(&result)
			if err == nil && result.WorkflowID != "" {
				t.Logf("Rerun created workflow: %s", result.WorkflowID)
			}
		}
	})
}
