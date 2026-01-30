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

// TestIntegration_MultiStep tests multi-step workflow execution
func TestIntegration_MultiStep(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	t.Run("Sequential step execution", func(t *testing.T) {
		yaml := `
name: multi-step-sequential
on: workflow_dispatch
jobs:
  process:
    runs-on: linux-amd64
    steps:
      - name: Step 1
        uses: shell@v1
        with:
          command: echo "Step 1 executed"
      
      - name: Step 2
        uses: shell@v1
        with:
          command: echo "Step 2 executed"
      
      - name: Step 3
        uses: shell@v1
        with:
          command: echo "Step 3 executed"
`
		resp, err := submitWorkflow(ctx, serverURL, yaml)
		require.NoError(t, err)

		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.GetID(), 90*time.Second)
		require.NoError(t, err)

		assert.Equal(t, "completed", strings.ToLower(status.Status))
	})

	t.Run("Step output passing", func(t *testing.T) {
		workflowYAML, err := loadTestWorkflow("multi-step-outputs.yaml")
		require.NoError(t, err)

		resp, err := submitWorkflow(ctx, serverURL, workflowYAML)
		require.NoError(t, err)

		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.GetID(), 90*time.Second)
		require.NoError(t, err)

		assert.Equal(t, "completed", strings.ToLower(status.Status),
			"Multi-step workflow with outputs should complete successfully")
	})
}

// TestIntegration_ConditionalExecution tests if conditions
func TestIntegration_ConditionalExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	t.Run("Conditional steps", func(t *testing.T) {
		workflowYAML, err := loadTestWorkflow("conditional.yaml")
		require.NoError(t, err)

		resp, err := submitWorkflow(ctx, serverURL, workflowYAML)
		require.NoError(t, err)

		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.GetID(), 90*time.Second)
		require.NoError(t, err)

		assert.Equal(t, "completed", strings.ToLower(status.Status),
			"Conditional workflow should complete successfully")
	})

	t.Run("Continue on error", func(t *testing.T) {
		yaml := `
name: continue-on-error-test
on: workflow_dispatch
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Failing step
        continue-on-error: true
        uses: shell@v1
        with:
          command: exit 1
      
      - name: Should still run
        uses: shell@v1
        with:
          command: echo "Continued after error"
`
		resp, err := submitWorkflow(ctx, serverURL, yaml)
		require.NoError(t, err)

		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.GetID(), 90*time.Second)
		require.NoError(t, err)

		// Workflow should complete (not fail) because continue-on-error is set
		assert.Equal(t, "completed", strings.ToLower(status.Status),
			"Workflow with continue-on-error should complete")
	})
}

// TestIntegration_MatrixStrategy tests matrix parallel execution
func TestIntegration_MatrixStrategy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	t.Run("Matrix expansion", func(t *testing.T) {
		workflowYAML, err := loadTestWorkflow("matrix.yaml")
		require.NoError(t, err)

		resp, err := submitWorkflow(ctx, serverURL, workflowYAML)
		require.NoError(t, err)

		// Matrix workflows take longer
		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.GetID(), 180*time.Second)
		require.NoError(t, err)

		assert.Equal(t, "completed", strings.ToLower(status.Status),
			"Matrix workflow should complete successfully")
	})

	t.Run("Simple matrix", func(t *testing.T) {
		yaml := `
name: simple-matrix
on: workflow_dispatch
jobs:
  test:
    runs-on: linux-amd64
    strategy:
      matrix:
        value: [1, 2, 3]
    steps:
      - name: Print value
        uses: shell@v1
        with:
          command: echo "Value is ${{ matrix.value }}"
`
		resp, err := submitWorkflow(ctx, serverURL, yaml)
		require.NoError(t, err)

		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.GetID(), 120*time.Second)
		require.NoError(t, err)

		assert.Equal(t, "completed", strings.ToLower(status.Status))
	})
}

// TestIntegration_JobDependencies tests job needs
func TestIntegration_JobDependencies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	t.Run("Job dependencies", func(t *testing.T) {
		workflowYAML, err := loadTestWorkflow("job-dependencies.yaml")
		require.NoError(t, err)

		resp, err := submitWorkflow(ctx, serverURL, workflowYAML)
		require.NoError(t, err)

		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.GetID(), 120*time.Second)
		require.NoError(t, err)

		assert.Equal(t, "completed", strings.ToLower(status.Status),
			"Job dependency workflow should complete successfully")
	})

	t.Run("Simple dependency chain", func(t *testing.T) {
		yaml := `
name: simple-dependency
on: workflow_dispatch
jobs:
  first:
    runs-on: linux-amd64
    steps:
      - name: First job
        uses: shell@v1
        with:
          command: echo "First"

  second:
    runs-on: linux-amd64
    needs: first
    steps:
      - name: Second job
        uses: shell@v1
        with:
          command: echo "Second - runs after first"
`
		resp, err := submitWorkflow(ctx, serverURL, yaml)
		require.NoError(t, err)

		status, err := waitForWorkflowCompletion(ctx, serverURL, resp.GetID(), 90*time.Second)
		require.NoError(t, err)

		assert.Equal(t, "completed", strings.ToLower(status.Status))
	})
}
