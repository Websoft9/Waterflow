//go:build integration

// Package integration contains end-to-end integration tests for Waterflow.
// These tests require a running Waterflow environment (Server + Temporal + Agent).
//
// Run with: go test -v ./test/integration/...
// Or use: make integration-test
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// WorkflowResponse represents the API response for workflow operations
type WorkflowResponse struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id,omitempty"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
}

// WorkflowStatus represents workflow status response
type WorkflowStatus struct {
	WorkflowID string                 `json:"workflow_id"`
	RunID      string                 `json:"run_id"`
	Status     string                 `json:"status"`
	StartTime  string                 `json:"start_time,omitempty"`
	EndTime    string                 `json:"end_time,omitempty"`
	Jobs       map[string]interface{} `json:"jobs,omitempty"`
}

// TestIntegration_WorkflowSubmitAndComplete tests the full workflow lifecycle
func TestIntegration_WorkflowSubmitAndComplete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Load test workflow
	workflowYAML, err := loadTestWorkflow("simple-echo.yaml")
	require.NoError(t, err, "Failed to load test workflow")

	t.Run("Submit workflow", func(t *testing.T) {
		// Submit workflow
		resp, err := submitWorkflow(ctx, serverURL, workflowYAML)
		require.NoError(t, err, "Failed to submit workflow")
		assert.NotEmpty(t, resp.WorkflowID, "Workflow ID should not be empty")

		t.Logf("Submitted workflow: %s", resp.WorkflowID)

		// Wait for completion
		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.WorkflowID, 60*time.Second)
		require.NoError(t, err, "Workflow failed to complete")
		assert.Equal(t, "completed", strings.ToLower(status.Status), "Workflow should complete successfully")

		t.Logf("Workflow completed with status: %s", status.Status)
	})
}

// TestIntegration_WorkflowStatusQuery tests workflow status query API
func TestIntegration_WorkflowStatusQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	workflowYAML, err := loadTestWorkflow("simple-echo.yaml")
	require.NoError(t, err)

	// Submit workflow
	resp, err := submitWorkflow(ctx, serverURL, workflowYAML)
	require.NoError(t, err)

	t.Run("Query workflow status", func(t *testing.T) {
		status, err := getWorkflowStatus(ctx, serverURL, resp.WorkflowID)
		require.NoError(t, err)

		assert.Equal(t, resp.WorkflowID, status.WorkflowID)
		assert.NotEmpty(t, status.Status)
		assert.NotEmpty(t, status.RunID)
	})

	t.Run("Query non-existent workflow", func(t *testing.T) {
		_, err := getWorkflowStatus(ctx, serverURL, "non-existent-workflow-id")
		assert.Error(t, err, "Should return error for non-existent workflow")
	})
}

// TestIntegration_WorkflowCancel tests workflow cancellation
func TestIntegration_WorkflowCancel(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Use a workflow with sleep to give time for cancellation
	workflowYAML := `
name: cancel-test
on: workflow_dispatch
jobs:
  slow:
    runs-on: linux-amd64
    steps:
      - name: Long running task
        uses: sleep@v1
        with:
          duration: 60s
`

	// Submit workflow
	resp, err := submitWorkflow(ctx, serverURL, workflowYAML)
	require.NoError(t, err)

	// Wait a bit for workflow to start
	time.Sleep(2 * time.Second)

	t.Run("Cancel running workflow", func(t *testing.T) {
		err := cancelWorkflow(ctx, serverURL, resp.WorkflowID)
		require.NoError(t, err, "Failed to cancel workflow")

		// Verify workflow is cancelled
		time.Sleep(2 * time.Second)
		status, err := getWorkflowStatus(ctx, serverURL, resp.WorkflowID)
		require.NoError(t, err)

		// Status should be cancelled or terminated
		assert.True(t,
			strings.Contains(strings.ToLower(status.Status), "cancel") ||
				strings.Contains(strings.ToLower(status.Status), "terminated"),
			"Workflow should be cancelled, got: %s", status.Status)
	})
}

// TestIntegration_WorkflowLogs tests workflow log retrieval
func TestIntegration_WorkflowLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	workflowYAML, err := loadTestWorkflow("simple-echo.yaml")
	require.NoError(t, err)

	// Submit and wait for completion
	resp, err := submitWorkflow(ctx, serverURL, workflowYAML)
	require.NoError(t, err)

	_, err = waitForWorkflowCompletion(ctx, serverURL, resp.WorkflowID, 60*time.Second)
	require.NoError(t, err)

	t.Run("Get workflow logs", func(t *testing.T) {
		logs, err := getWorkflowLogs(ctx, serverURL, resp.WorkflowID)
		require.NoError(t, err)

		assert.NotEmpty(t, logs, "Logs should not be empty")
		t.Logf("Retrieved %d bytes of logs", len(logs))
	})
}

// Helper functions

func getServerURL() string {
	url := os.Getenv("WATERFLOW_TEST_URL")
	if url == "" {
		url = "http://localhost:18080"
	}
	return url
}

func loadTestWorkflow(name string) (string, error) {
	// Try multiple paths
	paths := []string{
		filepath.Join("testdata", "workflows", name),
		filepath.Join("test", "integration", "testdata", "workflows", name),
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			return string(data), nil
		}
	}

	return "", fmt.Errorf("workflow file not found: %s", name)
}

func submitWorkflow(ctx context.Context, serverURL, yaml string) (*WorkflowResponse, error) {
	reqBody, _ := json.Marshal(map[string]string{"yaml": yaml})
	req, err := http.NewRequestWithContext(ctx, "POST", serverURL+"/v1/workflows", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to submit workflow: %d - %s", resp.StatusCode, string(body))
	}

	var result WorkflowResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func getWorkflowStatus(ctx context.Context, serverURL, workflowID string) (*WorkflowStatus, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", serverURL+"/v1/workflows/"+workflowID, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get workflow status: %d - %s", resp.StatusCode, string(body))
	}

	var status WorkflowStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}

	return &status, nil
}

func waitForWorkflowCompletion(ctx context.Context, serverURL, workflowID string, timeout time.Duration) (*WorkflowStatus, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("timeout waiting for workflow completion")
			}

			status, err := getWorkflowStatus(ctx, serverURL, workflowID)
			if err != nil {
				continue // Retry on error
			}

			statusLower := strings.ToLower(status.Status)
			if statusLower == "completed" || statusLower == "succeeded" ||
				statusLower == "failed" || statusLower == "cancelled" ||
				statusLower == "terminated" {
				return status, nil
			}
		}
	}
}

func cancelWorkflow(ctx context.Context, serverURL, workflowID string) error {
	req, err := http.NewRequestWithContext(ctx, "POST", serverURL+"/v1/workflows/"+workflowID+"/cancel", nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to cancel workflow: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

func getWorkflowLogs(ctx context.Context, serverURL, workflowID string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", serverURL+"/v1/workflows/"+workflowID+"/logs", nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get logs: %d - %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
