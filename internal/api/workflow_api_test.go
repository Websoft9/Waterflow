package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Websoft9/waterflow/pkg/config"
	"github.com/Websoft9/waterflow/pkg/logger"
	"github.com/Websoft9/waterflow/pkg/temporal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupTestRouter creates a test router with or without Temporal client
func setupTestRouter(t *testing.T, withTemporal bool) (http.Handler, *temporal.Client) {
	_ = logger.Init("error", "json")
	testLogger := zap.NewNop()

	var temporalClient *temporal.Client
	if withTemporal {
		cfg := &config.TemporalConfig{
			Host:          "localhost:7233",
			Namespace:     "default", // Use default namespace
			TaskQueue:     "test-queue",
			MaxRetries:    1,
			RetryInterval: 0,
		}
		var err error
		temporalClient, err = temporal.NewClient(cfg, testLogger)
		if err != nil {
			t.Logf("Temporal not available, test will be limited: %v", err)
			temporalClient = nil
		}
	}

	router := NewRouter(testLogger, temporalClient, "test", "test", "test")
	return router, temporalClient
}

// TestSubmitWorkflow_Success tests successful workflow submission
func TestSubmitWorkflow_Success(t *testing.T) {
	router, temporalClient := setupTestRouter(t, true)
	if temporalClient == nil {
		t.Skip("Temporal not available, skipping integration test")
	}
	defer temporalClient.Close()

	// Prepare request
	reqBody := SubmitWorkflowRequest{
		YAML: `name: test-workflow
on:
  workflow_dispatch:
vars:
  env: dev
jobs:
  test:
    runs-on: test-queue
    steps:
      - name: Test Step
        uses: echo@v1
        with:
          message: "Hello"`,
		Vars: map[string]interface{}{
			"override": "value",
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusCreated {
		t.Logf("Response body: %s", w.Body.String())
	}
	require.Equal(t, http.StatusCreated, w.Code, "Response body: %s", w.Body.String())

	var resp SubmitWorkflowResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.ID)
	assert.NotEmpty(t, resp.RunID)
	assert.Equal(t, "test-workflow", resp.Name)
	assert.Equal(t, "running", resp.Status)
	assert.NotEmpty(t, resp.CreatedAt)
	assert.Equal(t, "/v1/workflows/"+resp.ID, resp.URL)
}

// TestSubmitWorkflow_MissingYAML tests error when YAML is missing
func TestSubmitWorkflow_MissingYAMLField(t *testing.T) {
	logger := zap.NewNop()
	handlers := NewWorkflowHandlers(logger, nil)

	reqBody := SubmitWorkflowRequest{
		YAML: "",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handlers.SubmitWorkflow(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp, "error")
	errorObj := errResp["error"].(map[string]interface{})
	assert.Equal(t, "invalid_request", errorObj["code"])
}

// TestSubmitWorkflow_MalformedYAML tests error when YAML is malformed
func TestSubmitWorkflow_MalformedYAML(t *testing.T) {
	logger := zap.NewNop()
	handlers := NewWorkflowHandlers(logger, nil)

	reqBody := SubmitWorkflowRequest{
		YAML: "invalid: [unclosed",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handlers.SubmitWorkflow(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var errResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp, "error")
	errorObj := errResp["error"].(map[string]interface{})
	assert.Equal(t, "validation_error", errorObj["code"])
}

// TestSubmitWorkflow_InvalidJSONFormat tests error when request JSON is invalid
func TestSubmitWorkflow_InvalidJSONFormat(t *testing.T) {
	logger := zap.NewNop()
	handlers := NewWorkflowHandlers(logger, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewReader([]byte("{invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handlers.SubmitWorkflow(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp, "error")
}

// TestGetWorkflowStatus_NotFound tests 404 when workflow doesn't exist
func TestGetWorkflowStatus_NotFound(t *testing.T) {
	router, temporalClient := setupTestRouter(t, true)
	if temporalClient == nil {
		t.Skip("Temporal not available, skipping integration test")
	}
	defer temporalClient.Close()

	req := httptest.NewRequest(http.MethodGet, "/v1/workflows/nonexistent-id", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp, "error")
	errorObj := errResp["error"].(map[string]interface{})
	assert.Equal(t, "not_found", errorObj["code"])
}

// TestListWorkflows_Success tests successful list query
func TestListWorkflows_Success(t *testing.T) {
	logger := zap.NewNop()
	handlers := NewWorkflowHandlers(logger, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/workflows?page=1&limit=20", nil)

	w := httptest.NewRecorder()
	handlers.ListWorkflows(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp, "workflows")
	assert.Contains(t, resp, "pagination")
}

// TestListWorkflows_InvalidParameters tests parameter validation
func TestListWorkflows_InvalidParameters(t *testing.T) {
	logger := zap.NewNop()
	handlers := NewWorkflowHandlers(logger, nil)

	// Test invalid limit
	req := httptest.NewRequest(http.MethodGet, "/v1/workflows?limit=500", nil)
	w := httptest.NewRecorder()
	handlers.ListWorkflows(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Contains(t, errResp, "error")
	errorObj := errResp["error"].(map[string]interface{})
	assert.Equal(t, "invalid_parameter", errorObj["code"])
}

// TestCancelWorkflow_NotRunning tests conflict error when canceling non-running workflow
func TestCancelWorkflow_NotRunning(t *testing.T) {
	router, temporalClient := setupTestRouter(t, true)
	if temporalClient == nil {
		t.Skip("Temporal not available, skipping integration test")
	}
	defer temporalClient.Close()

	// Try to cancel a non-existent workflow
	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/test-id/cancel", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 404 (not found) or 409 (conflict)
	assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusConflict)
}
