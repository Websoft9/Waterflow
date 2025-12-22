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
	handlers := NewWorkflowHandlers(logger, nil) // nil client for unit test

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	handlers.SubmitWorkflow(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

func TestSubmitWorkflow_EmptyYAML(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil)

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
	handlers := NewWorkflowHandlers(logger, nil)

	reqBody := SubmitWorkflowRequest{
		YAML: "invalid: yaml: content:",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handlers.SubmitWorkflow(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestGetWorkflowStatus_MissingID(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handlers := NewWorkflowHandlers(logger, nil)

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
