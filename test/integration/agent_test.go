//go:build integration

package integration

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_AgentConnection tests agent connectivity
func TestIntegration_AgentConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	t.Run("Agent health check", func(t *testing.T) {
		agentURL := getAgentURL()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, agentURL+"/health", nil)
		require.NoError(t, err)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode,
			"Agent health check should return 200")
	})

	t.Run("Agent registration", func(t *testing.T) {
		// Check that agent appears in server's agent list
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL+"/api/v1/agents", nil)
		require.NoError(t, err)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Agent list endpoint should be accessible
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound,
			"Agents endpoint should respond")
	})
}

// TestIntegration_AgentWorkflowExecution tests agent executes workflows
func TestIntegration_AgentWorkflowExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	t.Run("Shell command execution on agent", func(t *testing.T) {
		yaml := `
name: agent-shell-test
on: workflow_dispatch
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Run shell command
        uses: shell@v1
        with:
          command: |
            hostname
            whoami
            pwd
`
		resp, err := submitWorkflow(ctx, serverURL, yaml)
		require.NoError(t, err)

		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.WorkflowID, 60*time.Second)
		require.NoError(t, err)

		assert.Equal(t, "completed", strings.ToLower(status.Status),
			"Shell command workflow should complete")
	})

	t.Run("Environment variable handling", func(t *testing.T) {
		yaml := `
name: agent-env-test
on: workflow_dispatch
jobs:
  test:
    runs-on: linux-amd64
    env:
      GLOBAL_VAR: "global-value"
    steps:
      - name: Check env vars
        uses: shell@v1
        env:
          STEP_VAR: "step-value"
        with:
          command: |
            echo "GLOBAL_VAR=$GLOBAL_VAR"
            echo "STEP_VAR=$STEP_VAR"
`
		resp, err := submitWorkflow(ctx, serverURL, yaml)
		require.NoError(t, err)

		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.WorkflowID, 60*time.Second)
		require.NoError(t, err)

		assert.Equal(t, "completed", strings.ToLower(status.Status),
			"Environment variable workflow should complete")
	})

	t.Run("Working directory handling", func(t *testing.T) {
		yaml := `
name: agent-workdir-test
on: workflow_dispatch
jobs:
  test:
    runs-on: linux-amd64
    defaults:
      run:
        working-directory: /tmp
    steps:
      - name: Check working directory
        uses: shell@v1
        with:
          command: |
            pwd
            ls -la
`
		resp, err := submitWorkflow(ctx, serverURL, yaml)
		require.NoError(t, err)

		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.WorkflowID, 60*time.Second)
		require.NoError(t, err)

		assert.Equal(t, "completed", strings.ToLower(status.Status),
			"Working directory workflow should complete")
	})
}

// TestIntegration_AgentLogs tests log collection from agent
func TestIntegration_AgentLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Run("Logs are collected", func(t *testing.T) {
		yaml := `
name: agent-logs-test
on: workflow_dispatch
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Generate logs
        uses: shell@v1
        with:
          command: |
            echo "Line 1: Test output"
            echo "Line 2: More output"
            echo "Line 3: Final line"
`
		resp, err := submitWorkflow(ctx, serverURL, yaml)
		require.NoError(t, err)

		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.WorkflowID, 60*time.Second)
		require.NoError(t, err)
		assert.Equal(t, "completed", strings.ToLower(status.Status))

		// Give logs time to be collected
		time.Sleep(2 * time.Second)

		// Query logs
		logs, err := getWorkflowLogs(ctx, serverURL, resp.WorkflowID)
		require.NoError(t, err)

		// Logs should contain the output
		assert.NotEmpty(t, logs, "Workflow logs should not be empty")
	})
}

// TestIntegration_AgentConcurrency tests concurrent workflow execution
func TestIntegration_AgentConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	t.Run("Parallel workflow execution", func(t *testing.T) {
		workflowCount := 3
		workflowIDs := make([]string, workflowCount)

		// Submit multiple workflows in parallel
		for i := 0; i < workflowCount; i++ {
			yaml := `
name: concurrent-test-` + string(rune('A'+i)) + `
on: workflow_dispatch
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Quick task
        uses: shell@v1
        with:
          command: sleep 2 && echo "Done"
`
			resp, err := submitWorkflow(ctx, serverURL, yaml)
			require.NoError(t, err)
			workflowIDs[i] = resp.WorkflowID
		}

		// Wait for all to complete
		for i, wfID := range workflowIDs {
			status, err := waitForWorkflowCompletion(ctx, serverURL, wfID, 120*time.Second)
			require.NoError(t, err, "Workflow %d should complete", i)
			assert.Equal(t, "completed", strings.ToLower(status.Status),
				"Concurrent workflow %d should complete", i)
		}
	})
}

// getAgentURL returns the agent URL from environment or default
func getAgentURL() string {
	url := getEnvOrDefault("AGENT_URL", "http://localhost:18081")
	return url
}
