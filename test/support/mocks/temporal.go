// Package mocks provides mock implementations for testing.
//
// # Temporal Mocks
//
// This package provides mock implementations for testing code that interacts
// with Temporal workflow engine. Due to the complexity of Temporal's internal
// interfaces, we provide wrapper-based mocks that can be used for unit testing
// without requiring a real Temporal server.
//
// # Key Components
//
//   - TemporalClientWrapper: Wraps Temporal client operations for testing
//   - TemporalTestEnv: Complete test environment using Temporal's testsuite
//   - WorkerMock: Mock worker for registration testing
//
// # Usage
//
//	func TestMyWorkflow(t *testing.T) {
//	    env := mocks.NewTemporalTestEnv(t)
//	    env.MockActivity(MyActivity, "result", nil)
//	    env.ExecuteWorkflow(MyWorkflow, input)
//	    env.AssertWorkflowCompleted()
//	}
package mocks

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
)

// ============================================================================
// Temporal Client Wrapper for Testing
// ============================================================================

// TemporalClientWrapper wraps a Temporal client for testing purposes.
// It tracks all operations and allows verification in tests.
type TemporalClientWrapper struct {
	mock.Mock
	mu sync.Mutex

	// Real client (optional, for integration tests)
	realClient client.Client

	// Configuration
	healthy   bool
	namespace string

	// Tracking
	executedWorkflows   []WorkflowExecution
	signalsSent         []SignalRecord
	queriesSent         []QueryRecord
	cancelledWorkflows  []string
	terminatedWorkflows []string
}

// WorkflowExecution records a workflow execution.
type WorkflowExecution struct {
	WorkflowType string
	WorkflowID   string
	RunID        string
	TaskQueue    string
	Args         []interface{}
	StartTime    time.Time
	Result       interface{}
	Error        error
	Completed    bool
}

// SignalRecord records a signal sent to a workflow.
type SignalRecord struct {
	WorkflowID string
	RunID      string
	SignalName string
	Arg        interface{}
	SentTime   time.Time
}

// QueryRecord records a query sent to a workflow.
type QueryRecord struct {
	WorkflowID string
	RunID      string
	QueryType  string
	Args       []interface{}
	QueryTime  time.Time
	Result     interface{}
}

// NewTemporalClientWrapper creates a new client wrapper for testing.
func NewTemporalClientWrapper() *TemporalClientWrapper {
	return &TemporalClientWrapper{
		healthy:             true,
		namespace:           "default",
		executedWorkflows:   make([]WorkflowExecution, 0),
		signalsSent:         make([]SignalRecord, 0),
		queriesSent:         make([]QueryRecord, 0),
		cancelledWorkflows:  make([]string, 0),
		terminatedWorkflows: make([]string, 0),
	}
}

// WithRealClient sets a real client for integration testing.
func (w *TemporalClientWrapper) WithRealClient(c client.Client) *TemporalClientWrapper {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.realClient = c
	return w
}

// SetHealthy sets whether health checks should succeed.
func (w *TemporalClientWrapper) SetHealthy(healthy bool) *TemporalClientWrapper {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.healthy = healthy
	return w
}

// SetNamespace sets the namespace.
func (w *TemporalClientWrapper) SetNamespace(ns string) *TemporalClientWrapper {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.namespace = ns
	return w
}

// ExecuteWorkflow tracks a workflow execution.
func (w *TemporalClientWrapper) ExecuteWorkflow(
	ctx context.Context,
	options client.StartWorkflowOptions,
	workflow interface{},
	args ...interface{},
) (*WorkflowExecution, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check mock expectations
	if len(w.ExpectedCalls) > 0 {
		mockArgs := w.Called(ctx, options, workflow, args)
		if mockArgs.Error(1) != nil {
			return nil, mockArgs.Error(1)
		}
	}

	workflowType := fmt.Sprintf("%T", workflow)
	runID := fmt.Sprintf("mock-run-%d", len(w.executedWorkflows)+1)

	exec := WorkflowExecution{
		WorkflowType: workflowType,
		WorkflowID:   options.ID,
		RunID:        runID,
		TaskQueue:    options.TaskQueue,
		Args:         args,
		StartTime:    time.Now(),
		Completed:    false,
	}

	w.executedWorkflows = append(w.executedWorkflows, exec)
	return &exec, nil
}

// SignalWorkflow tracks a signal.
func (w *TemporalClientWrapper) SignalWorkflow(ctx context.Context, workflowID, runID, signalName string, arg interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.ExpectedCalls) > 0 {
		mockArgs := w.Called(ctx, workflowID, runID, signalName, arg)
		if mockArgs.Error(0) != nil {
			return mockArgs.Error(0)
		}
	}

	w.signalsSent = append(w.signalsSent, SignalRecord{
		WorkflowID: workflowID,
		RunID:      runID,
		SignalName: signalName,
		Arg:        arg,
		SentTime:   time.Now(),
	})

	return nil
}

// QueryWorkflow tracks a query.
func (w *TemporalClientWrapper) QueryWorkflow(ctx context.Context, workflowID, runID, queryType string, args ...interface{}) (interface{}, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	var result interface{}
	var err error

	if len(w.ExpectedCalls) > 0 {
		mockArgs := w.Called(ctx, workflowID, runID, queryType, args)
		result = mockArgs.Get(0)
		err = mockArgs.Error(1)
	}

	w.queriesSent = append(w.queriesSent, QueryRecord{
		WorkflowID: workflowID,
		RunID:      runID,
		QueryType:  queryType,
		Args:       args,
		QueryTime:  time.Now(),
		Result:     result,
	})

	return result, err
}

// CancelWorkflow tracks a cancellation.
func (w *TemporalClientWrapper) CancelWorkflow(ctx context.Context, workflowID, runID string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.ExpectedCalls) > 0 {
		mockArgs := w.Called(ctx, workflowID, runID)
		if mockArgs.Error(0) != nil {
			return mockArgs.Error(0)
		}
	}

	w.cancelledWorkflows = append(w.cancelledWorkflows, workflowID)
	return nil
}

// TerminateWorkflow tracks a termination.
func (w *TemporalClientWrapper) TerminateWorkflow(ctx context.Context, workflowID, runID, reason string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.ExpectedCalls) > 0 {
		mockArgs := w.Called(ctx, workflowID, runID, reason)
		if mockArgs.Error(0) != nil {
			return mockArgs.Error(0)
		}
	}

	w.terminatedWorkflows = append(w.terminatedWorkflows, workflowID)
	return nil
}

// CheckHealth checks if the mock is configured as healthy.
func (w *TemporalClientWrapper) CheckHealth(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.healthy {
		return fmt.Errorf("temporal connection unhealthy (mock)")
	}
	return nil
}

// GetExecutedWorkflows returns all executed workflows.
func (w *TemporalClientWrapper) GetExecutedWorkflows() []WorkflowExecution {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]WorkflowExecution{}, w.executedWorkflows...)
}

// GetSignalsSent returns all sent signals.
func (w *TemporalClientWrapper) GetSignalsSent() []SignalRecord {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]SignalRecord{}, w.signalsSent...)
}

// GetQueriesSent returns all sent queries.
func (w *TemporalClientWrapper) GetQueriesSent() []QueryRecord {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]QueryRecord{}, w.queriesSent...)
}

// GetCancelledWorkflows returns all cancelled workflow IDs.
func (w *TemporalClientWrapper) GetCancelledWorkflows() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string{}, w.cancelledWorkflows...)
}

// GetTerminatedWorkflows returns all terminated workflow IDs.
func (w *TemporalClientWrapper) GetTerminatedWorkflows() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string{}, w.terminatedWorkflows...)
}

// Reset clears all tracked data.
func (w *TemporalClientWrapper) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.executedWorkflows = make([]WorkflowExecution, 0)
	w.signalsSent = make([]SignalRecord, 0)
	w.queriesSent = make([]QueryRecord, 0)
	w.cancelledWorkflows = make([]string, 0)
	w.terminatedWorkflows = make([]string, 0)
}

// ============================================================================
// Temporal Test Environment (using SDK testsuite)
// ============================================================================

// TemporalTestEnv provides a complete test environment for Temporal workflows.
// It wraps the official Temporal SDK testsuite for deterministic testing.
type TemporalTestEnv struct {
	t         *testing.T
	testSuite testsuite.WorkflowTestSuite
	env       *testsuite.TestWorkflowEnvironment

	// Activity mocking
	activityMocks map[string]ActivityMock
}

// ActivityMock defines a mocked activity response.
type ActivityMock struct {
	Result interface{}
	Error  error
	Delay  time.Duration
	Called int
}

// NewTemporalTestEnv creates a new test environment.
func NewTemporalTestEnv(t *testing.T) *TemporalTestEnv {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	return &TemporalTestEnv{
		t:             t,
		testSuite:     suite,
		env:           env,
		activityMocks: make(map[string]ActivityMock),
	}
}

// MockActivity sets up a mock for an activity.
// Note: Activity functions have signature func(ctx context.Context, args...) so
// we need to match all arguments including context.
func (e *TemporalTestEnv) MockActivity(activity interface{}, result interface{}, err error) *TemporalTestEnv {
	e.env.OnActivity(activity, mock.Anything, mock.Anything).Return(result, err)
	activityName := fmt.Sprintf("%T", activity)
	e.activityMocks[activityName] = ActivityMock{
		Result: result,
		Error:  err,
	}
	return e
}

// MockActivityWithDelay sets up a mock with simulated delay.
func (e *TemporalTestEnv) MockActivityWithDelay(activity interface{}, result interface{}, err error, delay time.Duration) *TemporalTestEnv {
	e.env.OnActivity(activity, mock.Anything, mock.Anything).After(delay).Return(result, err)
	activityName := fmt.Sprintf("%T", activity)
	e.activityMocks[activityName] = ActivityMock{
		Result: result,
		Error:  err,
		Delay:  delay,
	}
	return e
}

// MockActivityFunc mocks an activity with custom logic.
func (e *TemporalTestEnv) MockActivityFunc(activity interface{}, fn interface{}) *TemporalTestEnv {
	e.env.OnActivity(activity, mock.Anything).Return(fn)
	return e
}

// RegisterWorkflow registers a workflow for testing.
func (e *TemporalTestEnv) RegisterWorkflow(workflow interface{}) *TemporalTestEnv {
	e.env.RegisterWorkflow(workflow)
	return e
}

// RegisterActivity registers an activity for testing.
func (e *TemporalTestEnv) RegisterActivity(activity interface{}) *TemporalTestEnv {
	e.env.RegisterActivity(activity)
	return e
}

// ExecuteWorkflow executes a workflow and stores the result.
func (e *TemporalTestEnv) ExecuteWorkflow(workflow interface{}, args ...interface{}) *TemporalTestEnv {
	e.env.ExecuteWorkflow(workflow, args...)
	return e
}

// GetWorkflowResult retrieves the workflow result.
func (e *TemporalTestEnv) GetWorkflowResult(resultPtr interface{}) error {
	return e.env.GetWorkflowResult(resultPtr)
}

// GetWorkflowError returns the workflow error if any.
func (e *TemporalTestEnv) GetWorkflowError() error {
	return e.env.GetWorkflowError()
}

// AssertWorkflowCompleted asserts the workflow completed successfully.
func (e *TemporalTestEnv) AssertWorkflowCompleted() {
	require.True(e.t, e.env.IsWorkflowCompleted(), "workflow should be completed")
	require.NoError(e.t, e.env.GetWorkflowError(), "workflow should not have error")
}

// AssertWorkflowFailed asserts the workflow failed with an error.
func (e *TemporalTestEnv) AssertWorkflowFailed() {
	require.True(e.t, e.env.IsWorkflowCompleted(), "workflow should be completed")
	require.Error(e.t, e.env.GetWorkflowError(), "workflow should have error")
}

// SignalWorkflow sends a signal to the running workflow.
func (e *TemporalTestEnv) SignalWorkflow(signalName string, arg interface{}) {
	e.env.SignalWorkflow(signalName, arg)
}

// QueryWorkflow queries the running workflow.
func (e *TemporalTestEnv) QueryWorkflow(queryType string, args ...interface{}) (interface{}, error) {
	return e.env.QueryWorkflow(queryType, args...)
}

// SetStartTime sets the start time for the test environment.
func (e *TemporalTestEnv) SetStartTime(t time.Time) *TemporalTestEnv {
	e.env.SetStartTime(t)
	return e
}

// Env returns the underlying test environment for advanced usage.
func (e *TemporalTestEnv) Env() *testsuite.TestWorkflowEnvironment {
	return e.env
}

// ============================================================================
// Worker Mock for Registration Testing
// ============================================================================

// WorkerMock mocks a Temporal worker for testing.
type WorkerMock struct {
	mu sync.Mutex

	taskQueue            string
	running              bool
	registeredWorkflows  []string
	registeredActivities []string
}

// NewWorkerMock creates a new worker mock.
func NewWorkerMock(taskQueue string) *WorkerMock {
	return &WorkerMock{
		taskQueue:            taskQueue,
		running:              false,
		registeredWorkflows:  make([]string, 0),
		registeredActivities: make([]string, 0),
	}
}

// RegisterWorkflow records a workflow registration.
func (w *WorkerMock) RegisterWorkflow(wf interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.registeredWorkflows = append(w.registeredWorkflows, fmt.Sprintf("%T", wf))
}

// RegisterActivity records an activity registration.
func (w *WorkerMock) RegisterActivity(a interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.registeredActivities = append(w.registeredActivities, fmt.Sprintf("%T", a))
}

// Start starts the mock worker.
func (w *WorkerMock) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running {
		return fmt.Errorf("worker already running")
	}
	w.running = true
	return nil
}

// Stop stops the mock worker.
func (w *WorkerMock) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.running = false
}

// IsRunning returns whether the worker is running.
func (w *WorkerMock) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

// GetTaskQueue returns the task queue name.
func (w *WorkerMock) GetTaskQueue() string {
	return w.taskQueue
}

// GetRegisteredWorkflows returns registered workflow names.
func (w *WorkerMock) GetRegisteredWorkflows() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string{}, w.registeredWorkflows...)
}

// GetRegisteredActivities returns registered activity names.
func (w *WorkerMock) GetRegisteredActivities() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string{}, w.registeredActivities...)
}

// ============================================================================
// Test Helpers
// ============================================================================

// CreateTestTemporalClient creates a real Temporal client for integration tests.
// This requires a running Temporal server.
func CreateTestTemporalClient(t *testing.T, hostPort, namespace string) client.Client {
	c, err := client.Dial(client.Options{
		HostPort:  hostPort,
		Namespace: namespace,
	})
	require.NoError(t, err, "failed to create Temporal client")

	t.Cleanup(func() {
		c.Close()
	})

	return c
}

// CreateTestWorker creates a real Temporal worker for integration tests.
func CreateTestWorker(t *testing.T, c client.Client, taskQueue string, workflows []interface{}, activities []interface{}) worker.Worker {
	w := worker.New(c, taskQueue, worker.Options{})

	for _, wf := range workflows {
		w.RegisterWorkflow(wf)
	}

	for _, a := range activities {
		w.RegisterActivity(a)
	}

	err := w.Start()
	require.NoError(t, err, "failed to start worker")

	t.Cleanup(func() {
		w.Stop()
	})

	return w
}
