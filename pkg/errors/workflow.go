package errors

import (
	"fmt"
	"time"
)

// WorkflowNotFoundError represents a workflow that doesn't exist.
// This is a permanent error (non-retryable).
type WorkflowNotFoundError struct {
	*BaseError
	WorkflowID string
}

// NewWorkflowNotFoundError creates a new WorkflowNotFoundError.
func NewWorkflowNotFoundError(workflowID string) *WorkflowNotFoundError {
	return &WorkflowNotFoundError{
		BaseError: &BaseError{
			Type:      "not_found",
			Message:   fmt.Sprintf("Workflow not found: %s", workflowID),
			Retryable: false,
			Context: map[string]interface{}{
				"workflow_id": workflowID,
			},
		},
		WorkflowID: workflowID,
	}
}

// WorkflowExecutionError represents a workflow execution failure.
// Retryability depends on the underlying cause.
type WorkflowExecutionError struct {
	*BaseError
	WorkflowID string
	JobID      string
	StepName   string
}

// NewWorkflowExecutionError creates a new WorkflowExecutionError.
// It automatically classifies the error and determines retryability based on the cause.
func NewWorkflowExecutionError(workflowID, jobID, stepName string, cause error) *WorkflowExecutionError {
	// Classify the cause to determine retryability
	retryable := IsRetryableError(cause)
	classifier := NewErrorClassifier()
	errType := classifier.ClassifyError(cause)

	err := &WorkflowExecutionError{
		BaseError: &BaseError{
			Type:      errType,
			Message:   fmt.Sprintf("Workflow execution failed at step %s", stepName),
			Cause:     cause,
			Retryable: retryable,
			Context: map[string]interface{}{
				"workflow_id": workflowID,
				"job_id":      jobID,
				"step_name":   stepName,
			},
		},
		WorkflowID: workflowID,
		JobID:      jobID,
		StepName:   stepName,
	}

	// Collect stack trace if in debug mode
	if shouldCollectStackTrace() {
		err.WithStackTrace()
	}

	return err
}

// WorkflowTimeoutError represents a workflow timeout.
// This is a temporary error (retryable).
type WorkflowTimeoutError struct {
	*BaseError
	WorkflowID      string
	TimeoutDuration time.Duration
}

// NewWorkflowTimeoutError creates a new WorkflowTimeoutError.
func NewWorkflowTimeoutError(workflowID string, timeout time.Duration) *WorkflowTimeoutError {
	return &WorkflowTimeoutError{
		BaseError: &BaseError{
			Type:      "deadline_exceeded",
			Message:   fmt.Sprintf("Workflow timed out after %v", timeout),
			Retryable: true,
			Context: map[string]interface{}{
				"workflow_id":      workflowID,
				"timeout_duration": timeout.String(),
			},
		},
		WorkflowID:      workflowID,
		TimeoutDuration: timeout,
	}
}

// WorkflowCancelledError represents a cancelled workflow.
// This is a permanent error (non-retryable).
type WorkflowCancelledError struct {
	*BaseError
	WorkflowID string
	Reason     string
}

// NewWorkflowCancelledError creates a new WorkflowCancelledError.
func NewWorkflowCancelledError(workflowID, reason string) *WorkflowCancelledError {
	return &WorkflowCancelledError{
		BaseError: &BaseError{
			Type:      "cancelled",
			Message:   fmt.Sprintf("Workflow cancelled: %s", reason),
			Retryable: false,
			Context: map[string]interface{}{
				"workflow_id": workflowID,
				"reason":      reason,
			},
		},
		WorkflowID: workflowID,
		Reason:     reason,
	}
}
