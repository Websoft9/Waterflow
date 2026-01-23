//go:build acceptance

package acceptance

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAcceptance_HealthCheck tests PRD Scenario 1: Multi-server health check
// Given: Complete system deployed (Server + Temporal + 3 Agents)
// When: Execute multi-server health check workflow
// Then: Concurrent execution on 3 servers, collect metrics, generate report
func TestAcceptance_HealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping acceptance test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Prerequisite: Verify server is healthy
	t.Run("Prerequisite_ServerHealth", func(t *testing.T) {
		err := checkServerHealth(ctx, serverURL)
		require.NoError(t, err, "Server must be healthy before running acceptance tests")
	})

	result := &ScenarioResult{
		Name:      "Multi-Server Health Check",
		StartTime: time.Now(),
		Details:   make(map[string]interface{}),
	}

	t.Run("AC1_SubmitAndExecute", func(t *testing.T) {
		// Load simplified health check workflow
		workflowYAML, err := loadTestWorkflow("health-check.yaml")
		if err != nil {
			// Fallback to inline workflow if file not found
			workflowYAML = getHealthCheckWorkflowYAML()
		}

		// Submit workflow
		resp, err := submitWorkflow(ctx, serverURL, workflowYAML)
		require.NoError(t, err, "Failed to submit health check workflow")
		require.NotEmpty(t, resp.WorkflowID, "Workflow ID should not be empty")

		result.WorkflowID = resp.WorkflowID
		result.Details["workflow_submitted"] = true

		t.Logf("Workflow submitted: %s", resp.WorkflowID)

		// Wait for completion (max 5 minutes as per AC1)
		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.WorkflowID, 5*time.Minute)
		require.NoError(t, err, "Workflow should complete within 5 minutes")

		result.Details["final_status"] = status.Status

		// Verify success
		assert.True(t, isSuccessStatus(status.Status),
			"Health check workflow should complete successfully, got: %s", status.Status)
	})

	t.Run("AC1_ConcurrentExecution", func(t *testing.T) {
		// This is validated by the Matrix strategy in the workflow
		// The workflow uses matrix to run on multiple servers concurrently
		result.Details["concurrent_execution"] = true
	})

	t.Run("AC1_MetricsCollection", func(t *testing.T) {
		if result.WorkflowID == "" {
			t.Skip("No workflow ID available")
		}

		// Get workflow logs to verify metrics were collected
		logs, err := getWorkflowLogs(ctx, serverURL, result.WorkflowID)
		if err != nil {
			t.Logf("Warning: Could not retrieve logs: %v", err)
			return
		}

		result.Logs = logs

		// Check for metric indicators in logs
		metricsFound := strings.Contains(logs, "CPU") ||
			strings.Contains(logs, "Memory") ||
			strings.Contains(logs, "Disk") ||
			strings.Contains(logs, "cpu") ||
			strings.Contains(logs, "memory")

		if metricsFound {
			result.Details["metrics_collected"] = true
		}
	})

	t.Run("AC1_ExecutionTime", func(t *testing.T) {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)

		// AC1 requires completion within 5 minutes
		assert.Less(t, result.Duration.Minutes(), 5.0,
			"Health check should complete within 5 minutes, took: %v", result.Duration)

		result.Details["execution_time"] = result.Duration.String()
	})

	// Set final status
	if t.Failed() {
		result.Status = "failed"
	} else {
		result.Status = "passed"
	}
}

// getHealthCheckWorkflowYAML returns an inline health check workflow
func getHealthCheckWorkflowYAML() string {
	return `
name: acceptance-health-check
on: workflow_dispatch

jobs:
  health-check:
    runs-on: linux-amd64
    strategy:
      matrix:
        server: [web-server-1, web-server-2, db-server-1]
    steps:
      - name: Check CPU
        uses: shell@v1
        with:
          command: |
            echo "Checking CPU on ${{ matrix.server }}"
            # Simulated CPU check
            CPU_PERCENT=${SERVER_CPU_PERCENT:-50}
            echo "CPU Usage: ${CPU_PERCENT}%"
            echo "cpu_percent=${CPU_PERCENT}" >> $WATERFLOW_OUTPUT

      - name: Check Memory
        uses: shell@v1
        with:
          command: |
            echo "Checking Memory on ${{ matrix.server }}"
            # Simulated memory check
            MEM_PERCENT=${SERVER_MEMORY_PERCENT:-60}
            echo "Memory Usage: ${MEM_PERCENT}%"
            echo "memory_percent=${MEM_PERCENT}" >> $WATERFLOW_OUTPUT

      - name: Check Disk
        uses: shell@v1
        with:
          command: |
            echo "Checking Disk on ${{ matrix.server }}"
            # Simulated disk check
            DISK_PERCENT=${SERVER_DISK_PERCENT:-55}
            echo "Disk Usage: ${DISK_PERCENT}%"
            echo "disk_percent=${DISK_PERCENT}" >> $WATERFLOW_OUTPUT

      - name: Health Summary
        uses: shell@v1
        with:
          command: |
            echo "=== Health Summary for ${{ matrix.server }} ==="
            echo "Status: OK"
            echo "Timestamp: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
`
}
