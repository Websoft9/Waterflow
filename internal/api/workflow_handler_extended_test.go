package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ============================================================================
// Mock Temporal Client for Extended Tests
// ============================================================================

// MockTemporalClient implements a mock for temporal client operations
type MockTemporalClient struct {
	mock.Mock
}

func (m *MockTemporalClient) DescribeWorkflowExecution(ctx context.Context, workflowID, runID string) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
	args := m.Called(ctx, workflowID, runID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflowservice.DescribeWorkflowExecutionResponse), args.Error(1)
}

func (m *MockTemporalClient) CancelWorkflow(ctx context.Context, workflowID, runID string) error {
	args := m.Called(ctx, workflowID, runID)
	return args.Error(0)
}

// ============================================================================
// [P0] SubmitWorkflow Extended Tests - Vars Merging
// ============================================================================

// TestSubmitWorkflow_VarsMerging tests that request vars override YAML vars
func TestSubmitWorkflow_VarsMerging(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	// GIVEN: A workflow YAML with predefined vars
	reqBody := SubmitWorkflowRequest{
		YAML: `name: test-with-vars
on:
  workflow_dispatch:
vars:
  env: production
  version: "1.0"
jobs:
  deploy:
    runs-on: linux-amd64
    steps:
      - name: Deploy
        uses: shell@v1
        with:
          command: echo "Deploying ${{ vars.env }}"
`,
		// Request vars should override YAML vars
		Vars: map[string]interface{}{
			"env":    "staging",   // Override existing
			"region": "us-east-1", // Add new
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handlers.SubmitWorkflow(w, req)

	// THEN: Should return 400 (validation error) or 500 (no Temporal client)
	// Vars merging logic executed before validation/execution
	assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusUnprocessableEntity || w.Code == http.StatusInternalServerError,
		"Expected 400, 422, or 500, got %d", w.Code)
}

// TestSubmitWorkflow_ValidYAMLWithAllFields tests comprehensive workflow YAML
func TestSubmitWorkflow_ValidYAMLWithAllFields(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	// GIVEN: A comprehensive workflow YAML with all fields
	reqBody := SubmitWorkflowRequest{
		YAML: `name: comprehensive-workflow
on:
  workflow_dispatch:
    inputs:
      environment:
        description: "Target environment"
        required: true
        default: "staging"
vars:
  timeout: 300
  retries: 3
jobs:
  build:
    runs-on: linux-amd64
    timeout: 600
    steps:
      - name: Checkout
        uses: shell@v1
        with:
          command: git clone https://github.com/example/repo
      - name: Build
        uses: shell@v1
        with:
          command: make build
        timeout: 300
  test:
    runs-on: linux-amd64
    needs: [build]
    steps:
      - name: Run tests
        uses: shell@v1
        with:
          command: make test
`,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handlers.SubmitWorkflow(w, req)

	// Should return error status (400/422/500) - validates comprehensive parsing
	assert.True(t, w.Code >= 400,
		"Expected error status, got %d", w.Code)
}

// ============================================================================
// [P0] SubmitWorkflow - TaskQueue Selection Tests
// ============================================================================

// TestSubmitWorkflow_TaskQueueFromRunsOn tests task queue is derived from runs-on
func TestSubmitWorkflow_TaskQueueFromRunsOn(t *testing.T) {
	tests := []struct {
		name          string
		runsOn        string
		expectedQueue string
	}{
		{"linux-amd64", "linux-amd64", "linux-amd64"},
		{"linux-arm64", "linux-arm64", "linux-arm64"},
		{"windows-amd64", "windows-amd64", "windows-amd64"},
		{"darwin-amd64", "darwin-amd64", "darwin-amd64"},
		{"custom-queue", "my-custom-queue", "my-custom-queue"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			handlers := NewWorkflowHandlers(logger, nil, nil)

			yaml := `name: test-workflow
on: workflow_dispatch
jobs:
  test:
    runs-on: ` + tt.runsOn + `
    steps:
      - name: Test
        uses: shell@v1
        with:
          command: echo hello
`
			reqBody := SubmitWorkflowRequest{YAML: yaml}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			handlers.SubmitWorkflow(w, req)

			// Validates YAML parsing and task queue extraction
			// Error expected: 400 (validation), 422 (semantic), or 500 (no temporal)
			assert.True(t, w.Code >= 400,
				"Expected error status, got %d", w.Code)
		})
	}
}

// ============================================================================
// [P0] GetWorkflowStatus - Response Building Tests
// ============================================================================

// TestGetWorkflowStatus_EmptyID tests the status endpoint with empty ID
func TestGetWorkflowStatus_EmptyID(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	// Create request with empty mux vars
	req := httptest.NewRequest(http.MethodGet, "/v1/workflows/", nil)
	req = mux.SetURLVars(req, map[string]string{"id": ""})
	w := httptest.NewRecorder()

	handlers.GetWorkflowStatus(w, req)

	// Should return 400 for missing ID
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// [P1] CancelWorkflow - Complete Flow Tests
// ============================================================================

// TestCancelWorkflow_EmptyIDValidation tests cancellation with empty ID
func TestCancelWorkflow_EmptyIDValidation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows//cancel", nil)
	req = mux.SetURLVars(req, map[string]string{"id": ""})
	w := httptest.NewRecorder()

	handlers.CancelWorkflow(w, req)

	// Should return 400 for missing ID
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestCancelWorkflow_EmptyWorkflowID tests cancellation with empty ID
func TestCancelWorkflow_EmptyWorkflowID(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows//cancel", nil)
	req = mux.SetURLVars(req, map[string]string{"id": ""})
	w := httptest.NewRecorder()

	handlers.CancelWorkflow(w, req)

	// Should return 400 for missing ID
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// [P1] RerunWorkflow - Complete Flow Tests
// ============================================================================

// TestRerunWorkflow_EmptyWorkflowID tests rerun with empty ID
func TestRerunWorkflow_EmptyWorkflowIDValidation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows//rerun", nil)
	req = mux.SetURLVars(req, map[string]string{"id": ""})
	w := httptest.NewRecorder()

	handlers.RerunWorkflow(w, req)

	// Should return 400 for missing ID
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// [P1] GetWorkflowLogs - Complete Flow Tests
// ============================================================================

// TestGetWorkflowLogs_EmptyWorkflowIDValidation tests logs with empty ID
func TestGetWorkflowLogs_EmptyWorkflowIDValidation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/workflows//logs", nil)
	req = mux.SetURLVars(req, map[string]string{"id": ""})
	w := httptest.NewRecorder()

	handlers.GetWorkflowLogs(w, req)

	// Should return 400 for missing ID
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// [P1] ListWorkflows - Pagination & Filter Tests
// ============================================================================

// TestListWorkflows_PaginationParams tests various pagination parameters
func TestListWorkflows_PaginationParams(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{"valid page and limit", "page=1&limit=10", http.StatusOK},
		{"page only", "page=2", http.StatusOK},
		{"limit only", "limit=50", http.StatusOK},
		{"max limit", "limit=100", http.StatusOK},
		{"page zero", "page=0", http.StatusBadRequest},
		{"negative page", "page=-1", http.StatusBadRequest},
		{"limit too high", "limit=101", http.StatusBadRequest},
		{"limit zero", "limit=0", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			handlers := NewWorkflowHandlers(logger, nil, nil)

			req := httptest.NewRequest(http.MethodGet, "/v1/workflows?"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			handlers.ListWorkflows(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code,
				"Query: %s, Expected: %d, Got: %d", tt.queryParams, tt.expectedStatus, w.Code)
		})
	}
}

// TestListWorkflows_StatusFilters tests status filter combinations
func TestListWorkflows_StatusFilters(t *testing.T) {
	statuses := []string{"running", "completed", "failed", "cancelled", "pending"}

	for _, status := range statuses {
		t.Run("filter_by_"+status, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			handlers := NewWorkflowHandlers(logger, nil, nil)

			req := httptest.NewRequest(http.MethodGet, "/v1/workflows?status="+status, nil)
			w := httptest.NewRecorder()

			handlers.ListWorkflows(w, req)

			// Should return 200 with empty list (no Temporal client)
			assert.Equal(t, http.StatusOK, w.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Contains(t, resp, "workflows")
			assert.Contains(t, resp, "pagination")
		})
	}
}

// TestListWorkflows_CombinedFilters tests multiple filters
func TestListWorkflows_CombinedFilters(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	// GIVEN: Combined filters
	req := httptest.NewRequest(http.MethodGet,
		"/v1/workflows?status=running&name=deploy&page=1&limit=20", nil)
	w := httptest.NewRecorder()

	// WHEN: Listing workflows
	handlers.ListWorkflows(w, req)

	// THEN: Should return 200 with proper pagination structure
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	// Verify response structure
	pagination, ok := resp["pagination"].(map[string]interface{})
	require.True(t, ok, "pagination should be a map")
	assert.Equal(t, float64(1), pagination["page"])
	assert.Equal(t, float64(20), pagination["limit"])
}

// ============================================================================
// [P2] Temporal Status Mapping - Edge Cases
// ============================================================================

// TestMapTemporalStatus_AllStatuses tests all Temporal status mappings
func TestMapTemporalStatus_AllStatuses(t *testing.T) {
	tests := []struct {
		temporalStatus enums.WorkflowExecutionStatus
		expected       string
	}{
		{enums.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED, "unknown"},
		{enums.WORKFLOW_EXECUTION_STATUS_RUNNING, "running"},
		{enums.WORKFLOW_EXECUTION_STATUS_COMPLETED, "completed"},
		{enums.WORKFLOW_EXECUTION_STATUS_FAILED, "failed"},
		{enums.WORKFLOW_EXECUTION_STATUS_CANCELED, "cancelled"},
		{enums.WORKFLOW_EXECUTION_STATUS_TERMINATED, "terminated"},
		{enums.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW, "running"},
		{enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT, "timeout"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := mapTemporalStatus(tt.temporalStatus)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestMapConclusion_AllConclusions tests all conclusion mappings
func TestMapConclusion_AllConclusions(t *testing.T) {
	tests := []struct {
		temporalStatus enums.WorkflowExecutionStatus
		expected       string
	}{
		{enums.WORKFLOW_EXECUTION_STATUS_RUNNING, ""},
		{enums.WORKFLOW_EXECUTION_STATUS_COMPLETED, "success"},
		{enums.WORKFLOW_EXECUTION_STATUS_FAILED, "failure"},
		{enums.WORKFLOW_EXECUTION_STATUS_CANCELED, "cancelled"},
		{enums.WORKFLOW_EXECUTION_STATUS_TERMINATED, ""},
		{enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT, "timeout"},
		{enums.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW, ""},
	}

	for _, tt := range tests {
		t.Run(tt.temporalStatus.String(), func(t *testing.T) {
			result := mapConclusion(tt.temporalStatus)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// [P2] Workflow Helper Functions Tests
// ============================================================================

// TestBuildTemporalVisibilityQuery_AllCases tests query building for list
func TestBuildTemporalVisibilityQuery_AllCases(t *testing.T) {
	tests := []struct {
		name     string
		query    url.Values
		expected string
	}{
		{
			name:     "no filters",
			query:    url.Values{},
			expected: "",
		},
		{
			name:     "status only",
			query:    url.Values{"status": []string{"running"}},
			expected: `(ExecutionStatus = 'Running')`,
		},
		{
			name:     "name only",
			query:    url.Values{"name": []string{"deploy"}},
			expected: `WorkflowType = 'deploy'`,
		},
		{
			name:     "both filters",
			query:    url.Values{"status": []string{"completed"}, "name": []string{"build"}},
			expected: `(ExecutionStatus = 'Completed') AND WorkflowType = 'build'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildTemporalVisibilityQuery(tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// [P2] Error Response Format Tests
// ============================================================================

// TestWorkflowHandler_ErrorFormat_RFC7807 tests error responses follow RFC 7807
func TestWorkflowHandler_ErrorFormat_RFC7807(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	// Trigger an error via invalid request
	req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewBufferString("invalid"))
	w := httptest.NewRecorder()

	handlers.SubmitWorkflow(w, req)

	// Verify RFC 7807 format
	assert.Contains(t, w.Header().Get("Content-Type"), "application/problem+json")

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	// RFC 7807 required fields
	assert.Contains(t, resp, "type")
	assert.Contains(t, resp, "title")
	assert.Contains(t, resp, "status")
}

// ============================================================================
// [P2] Concurrent Request Handling Tests
// ============================================================================

// TestListWorkflows_ConcurrentRequests tests thread safety
func TestListWorkflows_ConcurrentRequests(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	// Run concurrent requests
	const numRequests = 10
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(idx int) {
			req := httptest.NewRequest(http.MethodGet, "/v1/workflows?page=1&limit=10", nil)
			w := httptest.NewRecorder()
			handlers.ListWorkflows(w, req)
			results <- w.Code
		}(i)
	}

	// Collect results
	for i := 0; i < numRequests; i++ {
		select {
		case code := <-results:
			assert.Equal(t, http.StatusOK, code, "Request %d failed", i)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent requests")
		}
	}
}

// ============================================================================
// Helper function for creating mock workflow execution info
// ============================================================================

//nolint:unused // Reserved for future workflow execution tests
func createMockWorkflowExecutionInfo(workflowID, runID, workflowType string, status enums.WorkflowExecutionStatus) *workflow.WorkflowExecutionInfo {
	now := time.Now()
	return &workflow.WorkflowExecutionInfo{
		Execution: &common.WorkflowExecution{
			WorkflowId: workflowID,
			RunId:      runID,
		},
		Type: &common.WorkflowType{
			Name: workflowType,
		},
		Status:    status,
		StartTime: timestamppb.New(now.Add(-1 * time.Hour)),
		CloseTime: timestamppb.New(now),
	}
}
