package errors

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewWorkflowNotFoundError tests WorkflowNotFoundError construction
func TestNewWorkflowNotFoundError(t *testing.T) {
	workflowID := "wf-123"
	err := NewWorkflowNotFoundError(workflowID)

	assert.NotNil(t, err)
	assert.Equal(t, "not_found", err.ErrorType())
	assert.Equal(t, workflowID, err.WorkflowID)
	assert.False(t, err.IsRetryable())
	assert.Contains(t, err.Error(), workflowID)
	assert.Equal(t, workflowID, err.ErrorContext()["workflow_id"])
}

// TestNewWorkflowExecutionError tests WorkflowExecutionError construction
func TestNewWorkflowExecutionError(t *testing.T) {
	tests := []struct {
		name            string
		cause           error
		expectRetryable bool
		expectType      string
	}{
		{
			name:            "retryable cause (timeout)",
			cause:           errors.New("connection timeout"),
			expectRetryable: true,
			expectType:      "deadline_exceeded",
		},
		{
			name:            "non-retryable cause (validation)",
			cause:           errors.New("validation error"),
			expectRetryable: false,
			expectType:      "validation_error",
		},
		{
			name:            "temporary error",
			cause:           errors.New("503 service unavailable"),
			expectRetryable: true,
			expectType:      "service_unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewWorkflowExecutionError("wf-123", "job-456", "deploy", tt.cause)

			assert.NotNil(t, err)
			assert.Equal(t, tt.expectType, err.ErrorType())
			assert.Equal(t, tt.expectRetryable, err.IsRetryable())
			assert.Equal(t, "wf-123", err.WorkflowID)
			assert.Equal(t, "job-456", err.JobID)
			assert.Equal(t, "deploy", err.StepName)
			assert.Contains(t, err.Error(), "deploy")
			assert.True(t, errors.Is(err, tt.cause))
		})
	}
}

// TestNewWorkflowTimeoutError tests WorkflowTimeoutError construction
func TestNewWorkflowTimeoutError(t *testing.T) {
	workflowID := "wf-123"
	timeout := 5 * time.Minute
	err := NewWorkflowTimeoutError(workflowID, timeout)

	assert.NotNil(t, err)
	assert.Equal(t, "deadline_exceeded", err.ErrorType())
	assert.Equal(t, workflowID, err.WorkflowID)
	assert.Equal(t, timeout, err.TimeoutDuration)
	assert.True(t, err.IsRetryable())
	assert.Contains(t, err.Error(), "5m0s")
	assert.Equal(t, "5m0s", err.ErrorContext()["timeout_duration"])
}

// TestNewWorkflowCancelledError tests WorkflowCancelledError construction
func TestNewWorkflowCancelledError(t *testing.T) {
	workflowID := "wf-123"
	reason := "user requested cancellation"
	err := NewWorkflowCancelledError(workflowID, reason)

	assert.NotNil(t, err)
	assert.Equal(t, "cancelled", err.ErrorType())
	assert.Equal(t, workflowID, err.WorkflowID)
	assert.Equal(t, reason, err.Reason)
	assert.False(t, err.IsRetryable())
	assert.Contains(t, err.Error(), reason)
	assert.Equal(t, reason, err.ErrorContext()["reason"])
}

// TestWorkflowErrors_InterfaceCompliance tests interface compliance
func TestWorkflowErrors_InterfaceCompliance(t *testing.T) {
	errors := []WaterflowError{
		NewWorkflowNotFoundError("wf-123"),
		NewWorkflowExecutionError("wf-123", "job-456", "deploy", errors.New("test")),
		NewWorkflowTimeoutError("wf-123", time.Minute),
		NewWorkflowCancelledError("wf-123", "test"),
	}

	for _, err := range errors {
		assert.NotEmpty(t, err.ErrorType())
		assert.NotEmpty(t, err.ErrorMessage())
		assert.NotNil(t, err.ErrorContext())

		// Test ToJSON
		data, jsonErr := err.ToJSON()
		assert.NoError(t, jsonErr)
		assert.NotEmpty(t, data)
	}
}
