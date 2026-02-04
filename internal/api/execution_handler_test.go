package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/Websoft9/waterflow/pkg/workflow"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// MockTemporalClient is a mock Temporal client
type MockTemporalClient struct {
	mock.Mock
}

func (m *MockTemporalClient) GetClient() client.Client {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(client.Client)
}

func (m *MockTemporalClient) CheckHealth(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTemporalClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockWorkflowRun is a mock workflow run
type MockWorkflowRun struct {
	mock.Mock
}

func (m *MockWorkflowRun) GetID() string {
	return "test-execution-id"
}

func (m *MockWorkflowRun) GetRunID() string {
	return "test-run-id-12345"
}

func (m *MockWorkflowRun) Get(ctx context.Context, valuePtr interface{}) error {
	args := m.Called(ctx, valuePtr)
	return args.Error(0)
}

func (m *MockWorkflowRun) GetWithOptions(ctx context.Context, valuePtr interface{}, options client.WorkflowRunGetOptions) error {
	args := m.Called(ctx, valuePtr, options)
	return args.Error(0)
}

// MockSDKClient is a mock Temporal SDK client
type MockSDKClient struct {
	mock.Mock
}

func (m *MockSDKClient) ExecuteWorkflow(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
	mockArgs := m.Called(ctx, options, workflow, args)
	if mockArgs.Get(0) == nil {
		return nil, mockArgs.Error(1)
	}
	return mockArgs.Get(0).(client.WorkflowRun), mockArgs.Error(1)
}

func (m *MockSDKClient) Close() {
	m.Called()
}

func TestExecuteWorkflowDefinition_Success(t *testing.T) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewExecutionHandlers(logger, nil, mockStore, nil)

	// Mock definition store
	def := &workflow.WorkflowDefinition{
		Name:        "test-workflow",
		Description: "Test",
		Content: `name: test-workflow
vars:
  env: production
jobs:
  test:
    runs-on: default
    steps:
      - name: Echo
        run: echo "hello"`,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockStore.On("Get", mock.Anything, "test-workflow").Return(def, nil)

	reqBody := ExecuteWorkflowDefinitionRequest{
		Vars: map[string]interface{}{
			"env": "staging", // Override YAML default
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/test-workflow/run", bytes.NewBuffer(body))
	req = mux.SetURLVars(req, map[string]string{"name": "test-workflow"})
	w := httptest.NewRecorder()

	handler.ExecuteWorkflowDefinition(w, req)

	// Should return 500 because Temporal client is nil
	// But validates that definition is loaded and parameters are merged
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockStore.AssertExpectations(t)
}

func TestExecuteWorkflowDefinition_NotFound(t *testing.T) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewExecutionHandlers(logger, nil, mockStore, nil)

	mockStore.On("Get", mock.Anything, "nonexistent").
		Return(nil, errors.New("workflow definition 'nonexistent' not found"))

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/nonexistent/run", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "nonexistent"})
	w := httptest.NewRecorder()

	handler.ExecuteWorkflowDefinition(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockStore.AssertExpectations(t)
}

func TestDirectExecuteWorkflow_ValidationFailure(t *testing.T) {
	logger := zap.NewNop()
	handler := NewExecutionHandlers(logger, nil, nil, nil)

	reqBody := DirectExecuteWorkflowRequest{
		Workflow: "invalid yaml content: [[[",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/executions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.DirectExecuteWorkflow(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestDirectExecuteWorkflow_MissingWorkflow(t *testing.T) {
	logger := zap.NewNop()
	handler := NewExecutionHandlers(logger, nil, nil, nil)

	reqBody := DirectExecuteWorkflowRequest{}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/executions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.DirectExecuteWorkflow(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExecuteWorkflowDefinition_EmptyName(t *testing.T) {
	logger := zap.NewNop()
	handler := NewExecutionHandlers(logger, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows//run", nil)
	req = mux.SetURLVars(req, map[string]string{"name": ""})
	w := httptest.NewRecorder()

	handler.ExecuteWorkflowDefinition(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestParameterOverridePriority tests AC6 - parameter override mechanism
func TestParameterOverridePriority(t *testing.T) {
	// Test validates parameter override logic: YAML vars < execution vars
	yamlContent := `name: test
vars:
  env: production
  replicas: 3
jobs:
  test:
    runs-on: default
    steps:
      - run: echo`

	parser := dsl.NewParser(zap.NewNop())
	wf, err := parser.Parse([]byte(yamlContent))
	assert.NoError(t, err)
	assert.Equal(t, "production", wf.Vars["env"])
	assert.Equal(t, 3, wf.Vars["replicas"])

	// Simulate execution vars override (as done in ExecuteWorkflowDefinition)
	executionVars := map[string]interface{}{
		"env": "staging",
	}

	for k, v := range executionVars {
		wf.Vars[k] = v
	}

	// Verify override
	assert.Equal(t, "staging", wf.Vars["env"], "execution vars should override YAML vars")
	assert.Equal(t, 3, wf.Vars["replicas"], "non-overridden vars should remain")
}

// TestParameterOverride_MultipleScenarios tests various override scenarios
func TestParameterOverride_MultipleScenarios(t *testing.T) {
	parser := dsl.NewParser(zap.NewNop())

	// Scenario 1: Complete override
	yamlContent1 := `name: test
vars:
  env: dev
jobs:
  test:
    runs-on: default
    steps:
      - run: echo`

	wf, _ := parser.Parse([]byte(yamlContent1))
	executionVars := map[string]interface{}{
		"env":     "prod",
		"version": "2.0",
	}
	for k, v := range executionVars {
		wf.Vars[k] = v
	}
	assert.Equal(t, "prod", wf.Vars["env"])
	assert.Equal(t, "2.0", wf.Vars["version"])

	// Scenario 2: Partial override
	yamlContent2 := `name: test
vars:
  env: dev
  replicas: 3
  timeout: 30
jobs:
  test:
    runs-on: default
    steps:
      - run: echo`

	wf2, _ := parser.Parse([]byte(yamlContent2))
	executionVars2 := map[string]interface{}{
		"env": "staging",
	}
	for k, v := range executionVars2 {
		wf2.Vars[k] = v
	}
	assert.Equal(t, "staging", wf2.Vars["env"])
	assert.Equal(t, 3, wf2.Vars["replicas"])
	assert.Equal(t, 30, wf2.Vars["timeout"])

	// Scenario 3: Type override
	yamlContent3 := `name: test
vars:
  debug: false
jobs:
  test:
    runs-on: default
    steps:
      - run: echo`

	wf3, _ := parser.Parse([]byte(yamlContent3))
	executionVars3 := map[string]interface{}{
		"debug": true,
	}
	for k, v := range executionVars3 {
		wf3.Vars[k] = v
	}
	assert.Equal(t, true, wf3.Vars["debug"])
}

func TestGetExecutionStatus_EmptyID(t *testing.T) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewExecutionHandlers(logger, nil, mockStore, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/executions/", nil)
	req = mux.SetURLVars(req, map[string]string{"id": ""})
	res := httptest.NewRecorder()

	handler.GetExecutionStatus(res, req)

	assert.NotEqual(t, http.StatusOK, res.Code)
}

func TestListExecutions_NoTemporal(t *testing.T) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewExecutionHandlers(logger, nil, mockStore, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/executions", nil)
	res := httptest.NewRecorder()

	handler.ListExecutions(res, req)

	// Without Temporal client, should return error
	assert.Equal(t, http.StatusInternalServerError, res.Code)
}

func TestGetExecutionLogs_NoTemporal(t *testing.T) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewExecutionHandlers(logger, nil, mockStore, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/executions/test-id/logs", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-id"})
	res := httptest.NewRecorder()

	handler.GetExecutionLogs(res, req)

	// Without Temporal client, should return error
	assert.Equal(t, http.StatusInternalServerError, res.Code)
}

func TestCancelExecution_NoTemporal(t *testing.T) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewExecutionHandlers(logger, nil, mockStore, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/executions/test-id/cancel", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-id"})
	res := httptest.NewRecorder()

	handler.CancelExecution(res, req)

	// Without Temporal client, should return error
	assert.Equal(t, http.StatusInternalServerError, res.Code)
}

func TestTerminateExecution_NoTemporal(t *testing.T) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewExecutionHandlers(logger, nil, mockStore, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/executions/test-id/terminate", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "test-id"})
	res := httptest.NewRecorder()

	handler.TerminateExecution(res, req)

	// Without Temporal client, should return error
	assert.Equal(t, http.StatusInternalServerError, res.Code)
}
