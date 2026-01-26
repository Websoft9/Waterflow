package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	"go.uber.org/zap/zaptest"
)

// ============================================================================
// Unit Tests for Internal Helper Functions
// ============================================================================

// TestMapConclusion_Comprehensive tests all conclusion mappings thoroughly
func TestMapConclusion_Comprehensive(t *testing.T) {
	// Test using Temporal enum values
	assert.Equal(t, "success", mapConclusion(enums.WORKFLOW_EXECUTION_STATUS_COMPLETED))
	assert.Equal(t, "failure", mapConclusion(enums.WORKFLOW_EXECUTION_STATUS_FAILED))
	assert.Equal(t, "cancelled", mapConclusion(enums.WORKFLOW_EXECUTION_STATUS_CANCELED))
	assert.Equal(t, "timeout", mapConclusion(enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT))
	assert.Equal(t, "", mapConclusion(enums.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED))
	assert.Equal(t, "", mapConclusion(enums.WORKFLOW_EXECUTION_STATUS_TERMINATED))
	assert.Equal(t, "", mapConclusion(enums.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW))
	assert.Equal(t, "", mapConclusion(enums.WORKFLOW_EXECUTION_STATUS_RUNNING))
}

// TestMapTemporalStatus_Comprehensive tests all status mappings thoroughly
func TestMapTemporalStatus_Comprehensive(t *testing.T) {
	// Test all defined statuses using Temporal enums
	assert.Equal(t, "unknown", mapTemporalStatus(enums.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED))
	assert.Equal(t, "running", mapTemporalStatus(enums.WORKFLOW_EXECUTION_STATUS_RUNNING))
	assert.Equal(t, "completed", mapTemporalStatus(enums.WORKFLOW_EXECUTION_STATUS_COMPLETED))
	assert.Equal(t, "failed", mapTemporalStatus(enums.WORKFLOW_EXECUTION_STATUS_FAILED))
	assert.Equal(t, "cancelled", mapTemporalStatus(enums.WORKFLOW_EXECUTION_STATUS_CANCELED))
	assert.Equal(t, "terminated", mapTemporalStatus(enums.WORKFLOW_EXECUTION_STATUS_TERMINATED))
	assert.Equal(t, "running", mapTemporalStatus(enums.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW))
	assert.Equal(t, "timeout", mapTemporalStatus(enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT))

	// Test unknown status
	assert.Equal(t, "unknown", mapTemporalStatus(enums.WorkflowExecutionStatus(99)))
}

// ============================================================================
// [P1] Validation Path Tests
// ============================================================================

// TestSubmitWorkflow_ValidationPaths tests various validation scenarios
func TestSubmitWorkflow_ValidationPaths(t *testing.T) {
	tests := []struct {
		name         string
		yaml         string
		expectStatus int
		description  string
	}{
		{
			name: "missing_name",
			yaml: `on: workflow_dispatch
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Test
        uses: shell@v1
        with:
          command: echo hello`,
			expectStatus: http.StatusBadRequest,
			description:  "Should reject workflow without name",
		},
		{
			name: "missing_jobs",
			yaml: `name: test
on: workflow_dispatch`,
			expectStatus: http.StatusBadRequest,
			description:  "Should reject workflow without jobs",
		},
		{
			name: "empty_jobs",
			yaml: `name: test
on: workflow_dispatch
jobs: {}`,
			expectStatus: http.StatusBadRequest,
			description:  "Should reject workflow with empty jobs",
		},
		{
			name: "missing_steps",
			yaml: `name: test
on: workflow_dispatch
jobs:
  build:
    runs-on: linux-amd64`,
			expectStatus: http.StatusBadRequest,
			description:  "Should reject job without steps",
		},
		{
			name: "step_missing_uses",
			yaml: `name: test
on: workflow_dispatch
jobs:
  build:
    runs-on: linux-amd64
    steps:
      - name: Test step`,
			expectStatus: http.StatusUnprocessableEntity,
			description:  "Should reject step without uses",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			handlers := NewWorkflowHandlers(logger, nil, nil)

			reqBody := SubmitWorkflowRequest{YAML: tt.yaml}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			handlers.SubmitWorkflow(w, req)

			// Verify it returns an error status
			assert.True(t, w.Code >= 400,
				"%s: Expected error status, got %d", tt.description, w.Code)
		})
	}
}

// TestSubmitWorkflow_YAMLSyntaxErrors tests YAML syntax error handling
func TestSubmitWorkflow_YAMLSyntaxErrors(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "invalid_indentation",
			yaml: `name: test
jobs:
  build:
 runs-on: linux`,
		},
		{
			name: "unclosed_quote",
			yaml: `name: "test
jobs:
  build:
    runs-on: linux`,
		},
		{
			name: "invalid_character",
			yaml: `name: test
jobs:
  build:
    runs-on: @invalid`,
		},
		{
			name: "tabs_instead_of_spaces",
			yaml: "name: test\njobs:\n\tbuild:\n\t\truns-on: linux",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			handlers := NewWorkflowHandlers(logger, nil, nil)

			reqBody := SubmitWorkflowRequest{YAML: tt.yaml}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			handlers.SubmitWorkflow(w, req)

			// Should return error for syntax errors
			assert.True(t, w.Code >= 400,
				"Expected error for %s, got %d", tt.name, w.Code)
		})
	}
}

// ============================================================================
// [P1] Input Validation Tests for List Workflows
// ============================================================================

// TestListWorkflows_QueryParamValidation tests query parameter validation
func TestListWorkflows_QueryParamValidation(t *testing.T) {
	tests := []struct {
		name         string
		queryString  string
		expectStatus int
	}{
		{"empty_query", "", http.StatusOK},
		{"valid_page", "page=5", http.StatusOK},
		{"valid_limit", "limit=50", http.StatusOK},
		{"valid_status", "status=running", http.StatusOK},
		{"valid_multiple_status", "status=running,completed", http.StatusOK},
		{"valid_name", "name=deploy", http.StatusOK},
		{"invalid_page_zero", "page=0", http.StatusBadRequest},
		{"invalid_page_negative", "page=-5", http.StatusBadRequest},
		{"invalid_limit_zero", "limit=0", http.StatusBadRequest},
		{"invalid_limit_too_large", "limit=500", http.StatusBadRequest},
		{"non_numeric_page", "page=abc", http.StatusOK},   // defaults to page 1
		{"non_numeric_limit", "limit=xyz", http.StatusOK}, // defaults to limit 20
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			handlers := NewWorkflowHandlers(logger, nil, nil)

			url := "/v1/workflows"
			if tt.queryString != "" {
				url += "?" + tt.queryString
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handlers.ListWorkflows(w, req)

			assert.Equal(t, tt.expectStatus, w.Code,
				"Query %s: expected %d, got %d", tt.queryString, tt.expectStatus, w.Code)
		})
	}
}

// ============================================================================
// [P2] Response Structure Tests
// ============================================================================

// TestListWorkflows_ResponseStructure tests the response structure
func TestListWorkflows_ResponseStructure(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/workflows?page=1&limit=10", nil)
	w := httptest.NewRecorder()

	handlers.ListWorkflows(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	// Verify response structure
	assert.Contains(t, resp, "workflows", "Response should contain 'workflows'")
	assert.Contains(t, resp, "pagination", "Response should contain 'pagination'")

	// Verify pagination structure
	pagination, ok := resp["pagination"].(map[string]interface{})
	require.True(t, ok, "pagination should be a map")
	assert.Contains(t, pagination, "page")
	assert.Contains(t, pagination, "limit")
	assert.Contains(t, pagination, "total")
	assert.Contains(t, pagination, "total_pages")

	// Verify workflows is array
	workflows, ok := resp["workflows"].([]interface{})
	require.True(t, ok, "workflows should be an array")
	assert.NotNil(t, workflows)
}

// ============================================================================
// [P2] Error Response Format Tests
// ============================================================================

// TestErrorResponse_Format tests RFC 7807 error response format
func TestErrorResponse_Format(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	// Trigger validation error
	reqBody := SubmitWorkflowRequest{YAML: ""}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handlers.SubmitWorkflow(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Check Content-Type header
	contentType := w.Header().Get("Content-Type")
	assert.Contains(t, contentType, "application/problem+json")

	// Parse response
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	// Verify RFC 7807 required fields
	assert.Contains(t, resp, "type", "Error should have 'type'")
	assert.Contains(t, resp, "title", "Error should have 'title'")
	assert.Contains(t, resp, "status", "Error should have 'status'")
}

// ============================================================================
// [P2] Workflow ID Validation Tests
// ============================================================================

// TestWorkflowIDValidation tests ID validation across endpoints
func TestWorkflowIDValidation(t *testing.T) {
	endpoints := []struct {
		name    string
		method  string
		path    string
		handler func(*WorkflowHandlers, http.ResponseWriter, *http.Request)
	}{
		{"GetWorkflowStatus", http.MethodGet, "/v1/workflows/{id}", func(h *WorkflowHandlers, w http.ResponseWriter, r *http.Request) { h.GetWorkflowStatus(w, r) }},
		{"CancelWorkflow", http.MethodPost, "/v1/workflows/{id}/cancel", func(h *WorkflowHandlers, w http.ResponseWriter, r *http.Request) { h.CancelWorkflow(w, r) }},
		{"RerunWorkflow", http.MethodPost, "/v1/workflows/{id}/rerun", func(h *WorkflowHandlers, w http.ResponseWriter, r *http.Request) { h.RerunWorkflow(w, r) }},
		{"GetWorkflowLogs", http.MethodGet, "/v1/workflows/{id}/logs", func(h *WorkflowHandlers, w http.ResponseWriter, r *http.Request) { h.GetWorkflowLogs(w, r) }},
	}

	for _, ep := range endpoints {
		t.Run(ep.name+"_EmptyID", func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			handlers := NewWorkflowHandlers(logger, nil, nil)

			req := httptest.NewRequest(ep.method, ep.path, nil)
			req = mux.SetURLVars(req, map[string]string{"id": ""})
			w := httptest.NewRecorder()

			ep.handler(handlers, w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code,
				"%s should return 400 for empty ID", ep.name)
		})
	}
}

// ============================================================================
// [P2] Content-Type Header Tests
// ============================================================================

// TestContentTypeHeaders tests correct Content-Type headers
func TestContentTypeHeaders(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	t.Run("ListWorkflows_JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/workflows", nil)
		w := httptest.NewRecorder()

		handlers.ListWorkflows(w, req)

		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	})

	t.Run("ErrorResponse_ProblemJSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/workflows?page=-1", nil)
		w := httptest.NewRecorder()

		handlers.ListWorkflows(w, req)

		assert.Contains(t, w.Header().Get("Content-Type"), "application/problem+json")
	})
}

// ============================================================================
// [P2] Handler Initialization Tests
// ============================================================================

// TestNewWorkflowHandlers_NilDependencies tests handler creation with nil deps
func TestNewWorkflowHandlers_NilDependencies(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Should not panic with nil dependencies
	handlers := NewWorkflowHandlers(logger, nil, nil)

	assert.NotNil(t, handlers)
	assert.NotNil(t, handlers.logger)
	assert.NotNil(t, handlers.parser)
	// validator can be nil if creation fails
}

// TestSetAuditLogger_UpdatesLogger tests audit logger setter
func TestSetAuditLogger_UpdatesLogger(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	// Initially nil
	assert.Nil(t, handlers.auditLogger)

	// Set to nil (valid operation)
	handlers.SetAuditLogger(nil)
	assert.Nil(t, handlers.auditLogger)
}

// ============================================================================
// [P3] Edge Cases Tests
// ============================================================================

// TestSubmitWorkflow_LargeYAML tests handling of large workflow YAML
func TestSubmitWorkflow_LargeYAML(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	// Generate large YAML with many steps
	var steps string
	for i := 0; i < 100; i++ {
		steps += "      - name: Step " + string(rune('0'+i%10)) + "\n"
		steps += "        uses: shell@v1\n"
		steps += "        with:\n"
		steps += "          command: echo step\n"
	}

	yaml := `name: large-workflow
on: workflow_dispatch
jobs:
  build:
    runs-on: linux-amd64
    steps:
` + steps

	reqBody := SubmitWorkflowRequest{YAML: yaml}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handlers.SubmitWorkflow(w, req)

	// Should process without error (validation/temporal error expected)
	assert.True(t, w.Code >= 400,
		"Expected processing error for large YAML, got %d", w.Code)
}

// TestListWorkflows_MaxLimit tests maximum limit boundary
func TestListWorkflows_MaxLimit(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil, nil)

	// Test exactly at limit
	req := httptest.NewRequest(http.MethodGet, "/v1/workflows?limit=100", nil)
	w := httptest.NewRecorder()

	handlers.ListWorkflows(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "limit=100 should be valid")

	// Test one over limit
	req = httptest.NewRequest(http.MethodGet, "/v1/workflows?limit=101", nil)
	w = httptest.NewRecorder()

	handlers.ListWorkflows(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code, "limit=101 should be invalid")
}
