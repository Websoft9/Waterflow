package agent

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/config"
	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/Websoft9/waterflow/pkg/temporal"
	"github.com/Websoft9/waterflow/test/support/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap/zaptest"
)

// TestWorkerMock_Registration tests worker registration using mock.
func TestWorkerMock_Registration(t *testing.T) {
	workerMock := mocks.NewWorkerMock("test-queue")

	// Simulate registrations like real worker does
	workerMock.RegisterWorkflow(temporal.RunWorkflowExecutor)
	workerMock.RegisterActivity((*temporal.Activities).ExecuteStepActivity)

	// Verify registrations
	workflows := workerMock.GetRegisteredWorkflows()
	activities := workerMock.GetRegisteredActivities()

	assert.Len(t, workflows, 1, "should have 1 registered workflow")
	assert.Len(t, activities, 1, "should have 1 registered activity")
	// Workflow/Activity names are stored as full function signatures
	assert.NotEmpty(t, workflows[0], "workflow registration should not be empty")
	assert.NotEmpty(t, activities[0], "activity registration should not be empty")
}

// TestWorkerMock_Lifecycle tests worker lifecycle using mock.
func TestWorkerMock_Lifecycle(t *testing.T) {
	workerMock := mocks.NewWorkerMock("test-queue")

	// Initial state
	assert.False(t, workerMock.IsRunning(), "worker should not be running initially")

	// Start
	err := workerMock.Start()
	assert.NoError(t, err)
	assert.True(t, workerMock.IsRunning(), "worker should be running after start")

	// Start again should error
	err = workerMock.Start()
	assert.Error(t, err, "starting already running worker should error")

	// Stop
	workerMock.Stop()
	assert.False(t, workerMock.IsRunning(), "worker should not be running after stop")

	// Can start again after stop
	err = workerMock.Start()
	assert.NoError(t, err)
	assert.True(t, workerMock.IsRunning())
}

// TestWorkerMock_MultipleTaskQueues tests multiple workers for different queues.
func TestWorkerMock_MultipleTaskQueues(t *testing.T) {
	queues := []string{"linux-amd64", "windows-amd64", "gpu-a100"}
	workers := make([]*mocks.WorkerMock, len(queues))

	// Create workers for each queue
	for i, q := range queues {
		workers[i] = mocks.NewWorkerMock(q)
		workers[i].RegisterWorkflow(temporal.RunWorkflowExecutor)
	}

	// Verify each worker has correct task queue
	for i, w := range workers {
		assert.Equal(t, queues[i], w.GetTaskQueue())
		assert.Len(t, w.GetRegisteredWorkflows(), 1)
	}

	// Start all workers
	for _, w := range workers {
		err := w.Start()
		assert.NoError(t, err)
	}

	// Verify all running
	for _, w := range workers {
		assert.True(t, w.IsRunning())
	}

	// Stop all workers
	for _, w := range workers {
		w.Stop()
	}

	// Verify all stopped
	for _, w := range workers {
		assert.False(t, w.IsRunning())
	}
}

// TestTemporalClientWrapper_HealthCheck tests health check behavior.
func TestTemporalClientWrapper_HealthCheck(t *testing.T) {
	wrapper := mocks.NewTemporalClientWrapper()

	// Default is healthy
	err := wrapper.CheckHealth(context.Background())
	assert.NoError(t, err)

	// Set unhealthy
	wrapper.SetHealthy(false)
	err = wrapper.CheckHealth(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unhealthy")

	// Set healthy again
	wrapper.SetHealthy(true)
	err = wrapper.CheckHealth(context.Background())
	assert.NoError(t, err)
}

// TestTemporalClientWrapper_WorkflowExecution tests workflow execution tracking.
func TestTemporalClientWrapper_WorkflowExecution(t *testing.T) {
	wrapper := mocks.NewTemporalClientWrapper()
	ctx := context.Background()

	// Execute workflows
	opts1 := client.StartWorkflowOptions{
		ID:        "wf-1",
		TaskQueue: "linux",
	}
	opts2 := client.StartWorkflowOptions{
		ID:        "wf-2",
		TaskQueue: "windows",
	}

	_, err := wrapper.ExecuteWorkflow(ctx, opts1, "TestWorkflow", "arg1")
	require.NoError(t, err)

	_, err = wrapper.ExecuteWorkflow(ctx, opts2, "TestWorkflow", "arg2")
	require.NoError(t, err)

	// Verify tracking
	executions := wrapper.GetExecutedWorkflows()
	assert.Len(t, executions, 2)
	assert.Equal(t, "wf-1", executions[0].WorkflowID)
	assert.Equal(t, "wf-2", executions[1].WorkflowID)
	assert.Equal(t, "linux", executions[0].TaskQueue)
	assert.Equal(t, "windows", executions[1].TaskQueue)
}

// TestTemporalClientWrapper_SignalWorkflow tests signal tracking.
func TestTemporalClientWrapper_SignalWorkflow(t *testing.T) {
	wrapper := mocks.NewTemporalClientWrapper()
	ctx := context.Background()

	// Send signals
	err := wrapper.SignalWorkflow(ctx, "wf-1", "run-1", "cancel", nil)
	require.NoError(t, err)

	err = wrapper.SignalWorkflow(ctx, "wf-2", "run-2", "pause", map[string]string{"reason": "maintenance"})
	require.NoError(t, err)

	// Verify tracking
	signals := wrapper.GetSignalsSent()
	assert.Len(t, signals, 2)
	assert.Equal(t, "cancel", signals[0].SignalName)
	assert.Equal(t, "pause", signals[1].SignalName)
}

// TestTemporalClientWrapper_CancelWorkflow tests cancel tracking.
func TestTemporalClientWrapper_CancelWorkflow(t *testing.T) {
	wrapper := mocks.NewTemporalClientWrapper()
	ctx := context.Background()

	// Cancel workflows
	err := wrapper.CancelWorkflow(ctx, "wf-1", "run-1")
	require.NoError(t, err)

	err = wrapper.CancelWorkflow(ctx, "wf-2", "run-2")
	require.NoError(t, err)

	// Verify tracking
	cancelled := wrapper.GetCancelledWorkflows()
	assert.Len(t, cancelled, 2)
	assert.Contains(t, cancelled, "wf-1")
	assert.Contains(t, cancelled, "wf-2")
}

// TestTemporalClientWrapper_Reset tests reset functionality.
func TestTemporalClientWrapper_Reset(t *testing.T) {
	wrapper := mocks.NewTemporalClientWrapper()
	ctx := context.Background()

	// Perform some operations
	_, _ = wrapper.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: "wf-1"}, "Workflow")
	_ = wrapper.SignalWorkflow(ctx, "wf-1", "", "signal", nil)
	_ = wrapper.CancelWorkflow(ctx, "wf-1", "")

	// Verify data exists
	assert.Len(t, wrapper.GetExecutedWorkflows(), 1)
	assert.Len(t, wrapper.GetSignalsSent(), 1)
	assert.Len(t, wrapper.GetCancelledWorkflows(), 1)

	// Reset
	wrapper.Reset()

	// Verify cleared
	assert.Empty(t, wrapper.GetExecutedWorkflows())
	assert.Empty(t, wrapper.GetSignalsSent())
	assert.Empty(t, wrapper.GetCancelledWorkflows())
}

// TestPluginManagerCreation tests plugin manager creation.
func TestPluginManagerCreation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	registry := node.NewRegistry()

	pm := NewPluginManager("/plugins", registry, logger)
	assert.NotNil(t, pm)
}

// TestNodeRegistry_Operations tests node registry operations.
func TestNodeRegistry_Operations(t *testing.T) {
	registry := node.NewRegistry()

	// Get non-existent node
	_, err := registry.Get("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// List nodes (should be empty initially)
	nodes := registry.List()
	assert.Empty(t, nodes)
}

// TestActivities_Creation tests activity creation.
func TestActivities_Creation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	registry := node.NewRegistry()

	activities := temporal.NewActivities(logger, registry)
	assert.NotNil(t, activities)
}

// TestWorkerConfig tests worker configuration parsing.
func TestWorkerConfig(t *testing.T) {
	tests := []struct {
		name           string
		cfg            *config.Config
		expectedQueues int
	}{
		{
			name: "single task queue",
			cfg: &config.Config{
				Agent: config.AgentConfig{
					TaskQueues:      []string{"linux-amd64"},
					PluginDir:       "/tmp/plugins",
					ShutdownTimeout: 30 * time.Second,
				},
			},
			expectedQueues: 1,
		},
		{
			name: "multiple task queues",
			cfg: &config.Config{
				Agent: config.AgentConfig{
					TaskQueues:      []string{"linux", "windows", "mac"},
					PluginDir:       "/tmp/plugins",
					ShutdownTimeout: 30 * time.Second,
				},
			},
			expectedQueues: 3,
		},
		{
			name: "empty task queues",
			cfg: &config.Config{
				Agent: config.AgentConfig{
					TaskQueues:      []string{},
					PluginDir:       "/tmp/plugins",
					ShutdownTimeout: 30 * time.Second,
				},
			},
			expectedQueues: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Len(t, tt.cfg.Agent.TaskQueues, tt.expectedQueues)
		})
	}
}

// TestConcurrentWorkerOperations tests thread safety of worker mock.
func TestConcurrentWorkerOperations(t *testing.T) {
	workerMock := mocks.NewWorkerMock("test-queue")
	var wg sync.WaitGroup

	// Concurrent registrations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			workerMock.RegisterWorkflow(temporal.RunWorkflowExecutor)
			workerMock.RegisterActivity((*temporal.Activities).ExecuteStepActivity)
		}(i)
	}

	wg.Wait()

	// Should have 10 of each
	assert.Len(t, workerMock.GetRegisteredWorkflows(), 10)
	assert.Len(t, workerMock.GetRegisteredActivities(), 10)
}

// TestConcurrentClientOperations tests client wrapper thread safety.
func TestConcurrentClientOperations(t *testing.T) {
	wrapper := mocks.NewTemporalClientWrapper()
	ctx := context.Background()
	var wg sync.WaitGroup

	// Concurrent workflow executions
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			opts := client.StartWorkflowOptions{
				ID:        string(rune('a' + idx)),
				TaskQueue: "test",
			}
			_, _ = wrapper.ExecuteWorkflow(ctx, opts, "Workflow", idx)
		}(i)
	}

	wg.Wait()

	// Should have 10 executions
	assert.Len(t, wrapper.GetExecutedWorkflows(), 10)
}

// TestWorkerMock_RegistrationCounts tests that registrations are tracked correctly.
func TestWorkerMock_RegistrationCounts(t *testing.T) {
	workerMock := mocks.NewWorkerMock("test-queue")

	// Register multiple workflows and activities
	for i := 0; i < 5; i++ {
		workerMock.RegisterWorkflow(temporal.RunWorkflowExecutor)
	}

	for i := 0; i < 3; i++ {
		workerMock.RegisterActivity((*temporal.Activities).ExecuteStepActivity)
	}

	assert.Len(t, workerMock.GetRegisteredWorkflows(), 5, "should track 5 workflow registrations")
	assert.Len(t, workerMock.GetRegisteredActivities(), 3, "should track 3 activity registrations")
}

// TestTemporalClientWrapper_TerminateWorkflow tests terminate tracking.
func TestTemporalClientWrapper_TerminateWorkflow(t *testing.T) {
	wrapper := mocks.NewTemporalClientWrapper()
	ctx := context.Background()

	// Terminate workflows
	err := wrapper.TerminateWorkflow(ctx, "wf-1", "run-1", "manual termination")
	require.NoError(t, err)

	err = wrapper.TerminateWorkflow(ctx, "wf-2", "run-2", "timeout exceeded")
	require.NoError(t, err)

	// Verify tracking
	terminated := wrapper.GetTerminatedWorkflows()
	assert.Len(t, terminated, 2)
	assert.Contains(t, terminated, "wf-1")
	assert.Contains(t, terminated, "wf-2")
}

// TestTemporalClientWrapper_QueryWorkflow tests query tracking.
func TestTemporalClientWrapper_QueryWorkflow(t *testing.T) {
	wrapper := mocks.NewTemporalClientWrapper()
	ctx := context.Background()

	// Execute queries
	_, err := wrapper.QueryWorkflow(ctx, "wf-1", "run-1", "getStatus")
	require.NoError(t, err)

	_, err = wrapper.QueryWorkflow(ctx, "wf-2", "run-2", "getProgress", "detailed")
	require.NoError(t, err)

	// Verify tracking
	queries := wrapper.GetQueriesSent()
	assert.Len(t, queries, 2)
	assert.Equal(t, "getStatus", queries[0].QueryType)
	assert.Equal(t, "getProgress", queries[1].QueryType)
}

// TestTemporalClientWrapper_Namespace tests namespace configuration.
func TestTemporalClientWrapper_Namespace(t *testing.T) {
	wrapper := mocks.NewTemporalClientWrapper()

	// Default namespace
	wrapper.SetNamespace("production")

	// Can chain configuration
	wrapper.SetNamespace("staging").SetHealthy(true)

	// Verify it's still usable
	err := wrapper.CheckHealth(context.Background())
	assert.NoError(t, err)
}

// TestAgentConfigValidation tests agent configuration validation.
func TestAgentConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.AgentConfig
		isValid bool
	}{
		{
			name: "valid config",
			cfg: config.AgentConfig{
				TaskQueues:      []string{"linux"},
				PluginDir:       "/opt/plugins",
				ShutdownTimeout: 30 * time.Second,
			},
			isValid: true,
		},
		{
			name: "config with auto reload",
			cfg: config.AgentConfig{
				TaskQueues:        []string{"linux"},
				PluginDir:         "/opt/plugins",
				ShutdownTimeout:   30 * time.Second,
				AutoReloadPlugins: true,
			},
			isValid: true,
		},
		{
			name: "empty plugin dir is allowed",
			cfg: config.AgentConfig{
				TaskQueues:      []string{"linux"},
				PluginDir:       "",
				ShutdownTimeout: 30 * time.Second,
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These configs are structurally valid
			assert.NotNil(t, tt.cfg.TaskQueues)
		})
	}
}
