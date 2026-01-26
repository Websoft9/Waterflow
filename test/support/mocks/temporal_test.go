package mocks

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// Example Workflows and Activities for Testing
// ============================================================================

// SampleWorkflow is a simple workflow for testing.
func SampleWorkflow(ctx workflow.Context, input string) (string, error) {
	// Set activity options (required by Temporal)
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Execute activity
	var result string
	err := workflow.ExecuteActivity(ctx, SampleActivity, input).Get(ctx, &result)
	if err != nil {
		return "", err
	}
	return result, nil
}

// SampleActivity is a simple activity for testing.
func SampleActivity(ctx context.Context, input string) (string, error) {
	return "processed: " + input, nil
}

// ============================================================================
// Tests for TemporalClientWrapper
// ============================================================================

func TestTemporalClientWrapper_ExecuteWorkflow(t *testing.T) {
	wrapper := NewTemporalClientWrapper()

	ctx := context.Background()
	options := client.StartWorkflowOptions{
		ID:        "test-workflow-1",
		TaskQueue: "test-queue",
	}

	// Execute workflow
	exec, err := wrapper.ExecuteWorkflow(ctx, options, SampleWorkflow, "input-data")
	require.NoError(t, err)
	assert.Equal(t, "test-workflow-1", exec.WorkflowID)
	assert.Equal(t, "test-queue", exec.TaskQueue)
	assert.NotEmpty(t, exec.RunID)

	// Verify tracking
	executed := wrapper.GetExecutedWorkflows()
	require.Len(t, executed, 1)
	assert.Equal(t, "test-workflow-1", executed[0].WorkflowID)
}

func TestTemporalClientWrapper_SignalWorkflow(t *testing.T) {
	wrapper := NewTemporalClientWrapper()
	ctx := context.Background()

	// Send signal
	err := wrapper.SignalWorkflow(ctx, "wf-123", "run-456", "pause", map[string]string{"reason": "test"})
	require.NoError(t, err)

	// Verify tracking
	signals := wrapper.GetSignalsSent()
	require.Len(t, signals, 1)
	assert.Equal(t, "wf-123", signals[0].WorkflowID)
	assert.Equal(t, "pause", signals[0].SignalName)
}

func TestTemporalClientWrapper_CancelWorkflow(t *testing.T) {
	wrapper := NewTemporalClientWrapper()
	ctx := context.Background()

	// Cancel workflow
	err := wrapper.CancelWorkflow(ctx, "wf-to-cancel", "run-123")
	require.NoError(t, err)

	// Verify tracking
	cancelled := wrapper.GetCancelledWorkflows()
	require.Len(t, cancelled, 1)
	assert.Equal(t, "wf-to-cancel", cancelled[0])
}

func TestTemporalClientWrapper_TerminateWorkflow(t *testing.T) {
	wrapper := NewTemporalClientWrapper()
	ctx := context.Background()

	// Terminate workflow
	err := wrapper.TerminateWorkflow(ctx, "wf-to-terminate", "run-123", "test termination")
	require.NoError(t, err)

	// Verify tracking
	terminated := wrapper.GetTerminatedWorkflows()
	require.Len(t, terminated, 1)
	assert.Equal(t, "wf-to-terminate", terminated[0])
}

func TestTemporalClientWrapper_CheckHealth(t *testing.T) {
	wrapper := NewTemporalClientWrapper()
	ctx := context.Background()

	// Test healthy
	wrapper.SetHealthy(true)
	err := wrapper.CheckHealth(ctx)
	require.NoError(t, err)

	// Test unhealthy
	wrapper.SetHealthy(false)
	err = wrapper.CheckHealth(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unhealthy")
}

func TestTemporalClientWrapper_Reset(t *testing.T) {
	wrapper := NewTemporalClientWrapper()
	ctx := context.Background()

	// Add some data
	_, _ = wrapper.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: "wf-1"}, SampleWorkflow)
	_ = wrapper.SignalWorkflow(ctx, "wf-1", "", "signal", nil)

	// Verify data exists
	assert.Len(t, wrapper.GetExecutedWorkflows(), 1)
	assert.Len(t, wrapper.GetSignalsSent(), 1)

	// Reset
	wrapper.Reset()

	// Verify cleared
	assert.Empty(t, wrapper.GetExecutedWorkflows())
	assert.Empty(t, wrapper.GetSignalsSent())
}

// ============================================================================
// Tests for WorkerMock
// ============================================================================

func TestWorkerMock_Registration(t *testing.T) {
	worker := NewWorkerMock("my-queue")

	// Register workflows and activities
	worker.RegisterWorkflow(SampleWorkflow)
	worker.RegisterActivity(SampleActivity)

	// Verify registrations
	workflows := worker.GetRegisteredWorkflows()
	assert.Len(t, workflows, 1)

	activities := worker.GetRegisteredActivities()
	assert.Len(t, activities, 1)

	// Verify task queue
	assert.Equal(t, "my-queue", worker.GetTaskQueue())
}

func TestWorkerMock_Lifecycle(t *testing.T) {
	worker := NewWorkerMock("test-queue")

	// Initially not running
	assert.False(t, worker.IsRunning())

	// Start
	err := worker.Start()
	require.NoError(t, err)
	assert.True(t, worker.IsRunning())

	// Cannot start twice
	err = worker.Start()
	require.Error(t, err)

	// Stop
	worker.Stop()
	assert.False(t, worker.IsRunning())
}

// ============================================================================
// Tests for TemporalTestEnv
// ============================================================================

func TestTemporalTestEnv_SimpleWorkflow(t *testing.T) {
	env := NewTemporalTestEnv(t)

	// Mock activity
	env.MockActivity(SampleActivity, "mocked result", nil)

	// Execute workflow
	env.ExecuteWorkflow(SampleWorkflow, "test input")

	// Assert completion
	env.AssertWorkflowCompleted()

	// Check result
	var result string
	err := env.GetWorkflowResult(&result)
	require.NoError(t, err)
	assert.Equal(t, "mocked result", result)
}

func TestTemporalTestEnv_WorkflowWithError(t *testing.T) {
	env := NewTemporalTestEnv(t)

	// Mock activity to return error
	env.MockActivity(SampleActivity, "", assert.AnError)

	// Execute workflow
	env.ExecuteWorkflow(SampleWorkflow, "test input")

	// Assert failure
	env.AssertWorkflowFailed()
}

func TestTemporalTestEnv_MockActivityWithDelay(t *testing.T) {
	env := NewTemporalTestEnv(t)

	// Mock activity with delay
	env.MockActivityWithDelay(SampleActivity, "delayed result", nil, 100*time.Millisecond)

	// Execute workflow
	env.ExecuteWorkflow(SampleWorkflow, "test input")

	// Assert completion
	env.AssertWorkflowCompleted()
}

func TestTemporalTestEnv_SetStartTime(t *testing.T) {
	env := NewTemporalTestEnv(t)

	// Set custom start time
	customTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	env.SetStartTime(customTime)

	// Mock activity
	env.MockActivity(SampleActivity, "result", nil)

	// Execute workflow
	env.ExecuteWorkflow(SampleWorkflow, "input")
	env.AssertWorkflowCompleted()
}

// ============================================================================
// Example: Using TemporalTestEnv for Real Workflow Testing
// ============================================================================

func ExampleTemporalTestEnv() {
	// This example shows how to use TemporalTestEnv in your tests
	t := &testing.T{} // In real tests, this comes from the test function

	// Create test environment
	env := NewTemporalTestEnv(t)

	// Mock external activity
	env.MockActivity(SampleActivity, "mocked output", nil)

	// Execute the workflow
	env.ExecuteWorkflow(SampleWorkflow, "input data")

	// Verify the workflow completed successfully
	env.AssertWorkflowCompleted()

	// Get and verify result
	var result string
	_ = env.GetWorkflowResult(&result)
	// result == "mocked output"
}

func ExampleTemporalClientWrapper() {
	// This example shows how to use TemporalClientWrapper in your tests

	// Create wrapper
	wrapper := NewTemporalClientWrapper()

	// Configure health
	wrapper.SetHealthy(true)

	// Execute a workflow (tracking only)
	ctx := context.Background()
	options := client.StartWorkflowOptions{
		ID:        "my-workflow",
		TaskQueue: "my-queue",
	}
	_, _ = wrapper.ExecuteWorkflow(ctx, options, SampleWorkflow, "input")

	// Send a signal
	_ = wrapper.SignalWorkflow(ctx, "my-workflow", "", "pause", nil)

	// Verify operations
	executed := wrapper.GetExecutedWorkflows()
	// executed[0].WorkflowID == "my-workflow"
	_ = executed
}
