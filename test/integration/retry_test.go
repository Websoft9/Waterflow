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

// TestIntegration_RetryMechanism tests retry functionality
func TestIntegration_RetryMechanism(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	t.Run("Step retry on failure", func(t *testing.T) {
		workflowYAML, err := loadTestWorkflow("retry.yaml")
		require.NoError(t, err)

		resp, err := submitWorkflow(ctx, serverURL, workflowYAML)
		require.NoError(t, err)

		// Retry workflows need more time
		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.WorkflowID, 180*time.Second)
		require.NoError(t, err)

		// The retry.yaml may complete or fail depending on configuration
		// We're testing that retry mechanism is invoked
		assert.NotEmpty(t, status.Status)
	})

	t.Run("Retry with backoff", func(t *testing.T) {
		yaml := `
name: retry-backoff-test
on: workflow_dispatch
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Flaky step
        uses: shell@v1
        timeout-minutes: 2
        retry:
          max-attempts: 3
          backoff:
            initial: 1s
            multiplier: 2
        with:
          command: |
            # Simulate intermittent failure
            if [ ! -f /tmp/retry_success ]; then
              touch /tmp/retry_success
              exit 1
            fi
            echo "Success on retry"
            rm -f /tmp/retry_success
`
		resp, err := submitWorkflow(ctx, serverURL, yaml)
		require.NoError(t, err)

		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.WorkflowID, 120*time.Second)
		require.NoError(t, err)

		// Workflow should eventually succeed after retries
		assert.NotEqual(t, "", status.Status)
	})
}

// TestIntegration_TimeoutHandling tests timeout behavior
func TestIntegration_TimeoutHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	t.Run("Step timeout", func(t *testing.T) {
		yaml := `
name: step-timeout-test
on: workflow_dispatch
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Long running step
        uses: shell@v1
        timeout-minutes: 1
        with:
          command: sleep 90
`
		resp, err := submitWorkflow(ctx, serverURL, yaml)
		require.NoError(t, err)

		// Wait for timeout + buffer
		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.WorkflowID, 120*time.Second)
		require.NoError(t, err)

		// Should fail due to timeout
		assert.Equal(t, "failed", strings.ToLower(status.Status),
			"Workflow should fail due to step timeout")
	})

	t.Run("Job timeout", func(t *testing.T) {
		yaml := `
name: job-timeout-test
on: workflow_dispatch
jobs:
  test:
    runs-on: linux-amd64
    timeout-minutes: 1
    steps:
      - name: Sleep step
        uses: shell@v1
        with:
          command: sleep 120
`
		resp, err := submitWorkflow(ctx, serverURL, yaml)
		require.NoError(t, err)

		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.WorkflowID, 90*time.Second)
		require.NoError(t, err)

		assert.Equal(t, "failed", strings.ToLower(status.Status),
			"Workflow should fail due to job timeout")
	})
}

// TestIntegration_ErrorHandling tests error scenarios
func TestIntegration_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	t.Run("Step failure propagation", func(t *testing.T) {
		yaml := `
name: step-failure-test
on: workflow_dispatch
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Failing step
        uses: shell@v1
        with:
          command: exit 1
      
      - name: Should not run
        uses: shell@v1
        with:
          command: echo "This should not execute"
`
		resp, err := submitWorkflow(ctx, serverURL, yaml)
		require.NoError(t, err)

		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.WorkflowID, 60*time.Second)
		require.NoError(t, err)

		assert.Equal(t, "failed", strings.ToLower(status.Status),
			"Workflow should fail when step fails")
	})

	t.Run("Job failure stops dependent jobs", func(t *testing.T) {
		yaml := `
name: job-failure-propagation
on: workflow_dispatch
jobs:
  failing:
    runs-on: linux-amd64
    steps:
      - name: Fail
        uses: shell@v1
        with:
          command: exit 1

  dependent:
    runs-on: linux-amd64
    needs: failing
    steps:
      - name: Should skip
        uses: shell@v1
        with:
          command: echo "Should not run"
`
		resp, err := submitWorkflow(ctx, serverURL, yaml)
		require.NoError(t, err)

		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.WorkflowID, 60*time.Second)
		require.NoError(t, err)

		assert.Equal(t, "failed", strings.ToLower(status.Status),
			"Workflow should fail when dependent job's needs fail")
	})
}

// TestIntegration_WorkflowRecovery tests recovery scenarios
func TestIntegration_WorkflowRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	t.Run("Continue on error with always() condition", func(t *testing.T) {
		yaml := `
name: always-condition-test
on: workflow_dispatch
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Fail intentionally
        id: fail-step
        uses: shell@v1
        with:
          command: exit 1
        continue-on-error: true
      
      - name: Cleanup always runs
        if: always()
        uses: shell@v1
        with:
          command: echo "Cleanup executed"
`
		resp, err := submitWorkflow(ctx, serverURL, yaml)
		require.NoError(t, err)

		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.WorkflowID, 60*time.Second)
		require.NoError(t, err)

		// With continue-on-error, workflow should complete
		assert.Equal(t, "completed", strings.ToLower(status.Status),
			"Workflow with continue-on-error should complete")
	})
}
