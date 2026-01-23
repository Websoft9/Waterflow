//go:build integration

// Package integration contains end-to-end integration tests for Waterflow.
// These tests require a running Waterflow environment (Server + Temporal + Agent).
//
// Run with: go test -v -tags integration ./test/integration/...
// Or use: make integration-test
package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
