package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.temporal.io/api/enums/v1"
	"go.uber.org/zap/zaptest"
)

func TestSubmitWorkflow_InvalidJSON(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil) // nil client for unit test

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	handlers.SubmitWorkflow(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	// Response uses RFC 7807 format
	assert.Contains(t, w.Header().Get("Content-Type"), "application/problem+json")
}

func TestSubmitWorkflow_EmptyYAML(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	reqBody := SubmitWorkflowRequest{
		YAML: "",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handlers.SubmitWorkflow(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubmitWorkflow_InvalidYAML(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	reqBody := SubmitWorkflowRequest{
		YAML: "invalid: yaml: content:",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handlers.SubmitWorkflow(w, req)

	// Malformed YAML returns 400 Bad Request with validation_error type
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetWorkflowStatus_MissingID(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/workflows", nil)
	w := httptest.NewRecorder()

	handlers.GetWorkflowStatus(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMapTemporalStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   enums.WorkflowExecutionStatus
		expected string
	}{
		{"running", enums.WORKFLOW_EXECUTION_STATUS_RUNNING, "running"},
		{"completed", enums.WORKFLOW_EXECUTION_STATUS_COMPLETED, "completed"},
		{"failed", enums.WORKFLOW_EXECUTION_STATUS_FAILED, "failed"},
		{"cancelled", enums.WORKFLOW_EXECUTION_STATUS_CANCELED, "cancelled"},
		{"terminated", enums.WORKFLOW_EXECUTION_STATUS_TERMINATED, "terminated"},
		{"continued", enums.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW, "running"},
		{"timeout", enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT, "timeout"},
		{"unknown", enums.WorkflowExecutionStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapTemporalStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestMapConclusion tests conclusion mapping for workflow status
func TestMapConclusion(t *testing.T) {
	tests := []struct {
		name     string
		status   enums.WorkflowExecutionStatus
		expected string
	}{
		{"completed", enums.WORKFLOW_EXECUTION_STATUS_COMPLETED, "success"},
		{"failed", enums.WORKFLOW_EXECUTION_STATUS_FAILED, "failure"},
		{"cancelled", enums.WORKFLOW_EXECUTION_STATUS_CANCELED, "cancelled"},
		{"timeout", enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT, "timeout"},
		{"running", enums.WORKFLOW_EXECUTION_STATUS_RUNNING, ""},
		{"terminated", enums.WORKFLOW_EXECUTION_STATUS_TERMINATED, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapConclusion(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestListWorkflows_InvalidPage tests invalid page parameter
func TestListWorkflows_InvalidPage(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/workflows?page=0", nil)
	w := httptest.NewRecorder()

	handlers.ListWorkflows(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestListWorkflows_InvalidLimitTooLow tests invalid limit parameter (too low)
func TestListWorkflows_InvalidLimitTooLow(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/workflows?limit=0", nil)
	w := httptest.NewRecorder()

	handlers.ListWorkflows(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestListWorkflows_InvalidLimitTooHigh tests invalid limit parameter (too high)
func TestListWorkflows_InvalidLimitTooHigh(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/workflows?limit=101", nil)
	w := httptest.NewRecorder()

	handlers.ListWorkflows(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestListWorkflows_NoTemporal tests list when Temporal is not available
func TestListWorkflows_NoTemporal(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/workflows?page=1&limit=10", nil)
	w := httptest.NewRecorder()

	handlers.ListWorkflows(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp, "workflows")
	assert.Contains(t, resp, "pagination")
	// Should return empty list when no Temporal
	workflows := resp["workflows"].([]interface{})
	assert.Empty(t, workflows)
}

// TestListWorkflows_WithStatusFilter tests list with status filter
func TestListWorkflows_WithStatusFilter(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/workflows?status=running", nil)
	w := httptest.NewRecorder()

	handlers.ListWorkflows(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestListWorkflows_WithNameFilter tests list with name filter
func TestListWorkflows_WithNameFilter(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/workflows?name=test-workflow", nil)
	w := httptest.NewRecorder()

	handlers.ListWorkflows(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestCancelWorkflow_MissingID tests cancel with missing workflow ID
func TestCancelWorkflow_MissingID(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows//cancel", nil)
	w := httptest.NewRecorder()

	handlers.CancelWorkflow(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestRerunWorkflow_MissingID tests rerun with missing workflow ID
func TestRerunWorkflow_MissingID(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows//rerun", nil)
	w := httptest.NewRecorder()

	handlers.RerunWorkflow(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestGetWorkflowLogs_MissingID tests logs with missing workflow ID
func TestGetWorkflowLogs_MissingID(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/workflows//logs", nil)
	w := httptest.NewRecorder()

	handlers.GetWorkflowLogs(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestListTaskQueues_NoTemporal tests legacy WorkflowHandlers.ListTaskQueues (superseded by AgentHandlers)
func TestListTaskQueues_NoTemporal(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/task-queues", nil)
	w := httptest.NewRecorder()

	handlers.ListTaskQueues(w, req)

	// Legacy placeholder returns 200 with message
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp, "message")
	assert.Contains(t, resp, "task_queues")
}

// TestSetAuditLogger tests setting audit logger
func TestSetAuditLogger(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	assert.Nil(t, handlers.auditLogger)

	// Set a mock audit logger would require implementing the interface
	// For now, just test that nil is handled
	handlers.SetAuditLogger(nil)
	assert.Nil(t, handlers.auditLogger)
}

// TestSubmitWorkflow_InvalidWorkflowNoName tests workflow validation - missing name
func TestSubmitWorkflow_InvalidWorkflowNoName(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	reqBody := SubmitWorkflowRequest{
		YAML: `on:
  workflow_dispatch:
jobs:
  test:
    runs-on: linux
    steps:
      - name: Test
        uses: exec/shell@v1`,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handlers.SubmitWorkflow(w, req)

	// Validation error returns 400 (invalid YAML structure)
	assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusUnprocessableEntity)
}

// TestSubmitWorkflow_InvalidWorkflowNoJobs tests workflow validation - missing jobs
func TestSubmitWorkflow_InvalidWorkflowNoJobs(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	reqBody := SubmitWorkflowRequest{
		YAML: `name: test-workflow
on:
  workflow_dispatch:`,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handlers.SubmitWorkflow(w, req)

	// Validation error returns 400 (missing required fields)
	assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusUnprocessableEntity)
}

// TestSubmitWorkflow_InvalidWorkflowNoRunsOn tests workflow validation - missing runs-on
func TestSubmitWorkflow_InvalidWorkflowNoRunsOn(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	reqBody := SubmitWorkflowRequest{
		YAML: `name: test-workflow
on:
  workflow_dispatch:
jobs:
  test:
    steps:
      - name: Test
        uses: exec/shell@v1
        with:
          command: echo hello`,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handlers.SubmitWorkflow(w, req)

	// Validation error returns 400 or 422 (missing runs-on)
	assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusUnprocessableEntity)
}
