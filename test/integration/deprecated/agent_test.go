//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_AgentConnection tests agent connectivity by verifying
// workflow execution (Agent doesn't expose HTTP endpoint directly)
func TestIntegration_AgentConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Run("Agent is connected and processing tasks", func(t *testing.T) {
		// Verify agent connectivity by executing a simple workflow
		// This proves the agent is registered and accepting tasks
		yaml := `
name: agent-connectivity-test
on: workflow_dispatch
jobs:
  verify:
    runs-on: linux-amd64
    steps:
      - name: Verify agent is alive
        uses: shell@v1
        with:
          command: echo "Agent is connected and processing"
`
		resp, err := submitWorkflow(ctx, serverURL, yaml)
		require.NoError(t, err, "Should be able to submit workflow")
		assert.NotEmpty(t, resp.GetID(), "Should receive workflow ID")

		// If workflow completes successfully, agent is connected
		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.GetID(), 60*time.Second)
		require.NoError(t, err, "Workflow should complete")
		assert.Equal(t, "completed", strings.ToLower(status.Status),
			"Agent should execute workflow successfully, proving it's connected")
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

		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.GetID(), 60*time.Second)
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

		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.GetID(), 60*time.Second)
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

		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.GetID(), 60*time.Second)
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

		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.GetID(), 60*time.Second)
		require.NoError(t, err)
		assert.Equal(t, "completed", strings.ToLower(status.Status))

		// Give logs time to be collected
		time.Sleep(2 * time.Second)

		// Query logs
		logs, err := getWorkflowLogs(ctx, serverURL, resp.GetID())
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
			workflowIDs[i] = resp.GetID()
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
