package errors_test

import (
	"encoding/json"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Websoft9/waterflow/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRFC7807Integration tests RFC 7807 integration in HTTP handlers
func TestRFC7807Integration(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedType   string
		expectedTitle  string
	}{
		{
			name:           "ValidationError returns 400",
			err:            errors.NewValidationError("Invalid YAML", nil),
			expectedStatus: 400,
			expectedType:   "validation_error",
			expectedTitle:  "Validation Failed",
		},
		{
			name:           "WorkflowNotFoundError returns 404",
			err:            errors.NewWorkflowNotFoundError("wf-123"),
			expectedStatus: 404,
			expectedType:   "not_found",
			expectedTitle:  "Resource Not Found",
		},
		{
			name:           "NodeNotFoundError returns non-retryable",
			err:            errors.NewNodeNotFoundError("shell/exec"),
			expectedStatus: 500, // node_not_registered maps to 500
			expectedType:   "node_not_registered",
			expectedTitle:  "Node Not Registered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create HTTP handler that uses ToRFC7807
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				rfc7807 := errors.ToRFC7807(tt.err, r.URL.Path)

				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(rfc7807.Status)
				_ = json.NewEncoder(w).Encode(rfc7807)
			})

			// Create request
			req := httptest.NewRequest("GET", "/test/path", nil)
			w := httptest.NewRecorder()

			// Call handler
			handler.ServeHTTP(w, req)

			// Verify response
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))

			// Parse response
			var rfc7807 errors.RFC7807Response
			err := json.NewDecoder(w.Body).Decode(&rfc7807)
			require.NoError(t, err)

			// Verify RFC 7807 fields
			assert.Equal(t, tt.expectedType, rfc7807.Type)
			assert.Equal(t, tt.expectedTitle, rfc7807.Title)
			assert.Equal(t, tt.expectedStatus, rfc7807.Status)
			assert.NotEmpty(t, rfc7807.Detail)
			assert.Equal(t, "/test/path", rfc7807.Instance)
		})
	}
}

// TestErrorRetryabilityIntegration tests error classification for retry logic
func TestErrorRetryabilityIntegration(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "Validation error is non-retryable",
			err:       errors.NewValidationError("syntax error", nil),
			retryable: false,
		},
		{
			name:      "Workflow timeout is retryable",
			err:       errors.NewWorkflowTimeoutError("wf-123", 0),
			retryable: true,
		},
		{
			name:      "Node not found is non-retryable",
			err:       errors.NewNodeNotFoundError("missing/node"),
			retryable: false,
		},
		{
			name:      "Workflow cancelled is non-retryable",
			err:       errors.NewWorkflowCancelledError("wf-123", "user request"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if wfErr, ok := tt.err.(errors.WaterflowError); ok {
				assert.Equal(t, tt.retryable, wfErr.IsRetryable())
			} else {
				t.Errorf("Error does not implement WaterflowError interface")
			}

			// Also test global helper
			assert.Equal(t, tt.retryable, errors.IsRetryableError(tt.err))
		})
	}
}

// TestErrorContextPropagation tests context propagation through error wrapping
func TestErrorContextPropagation(t *testing.T) {
	// Simulate nested error propagation (like in actual workflow execution)
	originalErr := errors.NewNodeNotFoundError("shell/exec")

	// Wrap at node execution level
	nodeErr := errors.NewNodeExecutionError("shell/exec", "build", originalErr)
	assert.Equal(t, "node_not_registered", nodeErr.ErrorType())
	assert.False(t, nodeErr.IsRetryable())
	assert.Contains(t, nodeErr.ErrorContext(), "node_type")
	assert.Contains(t, nodeErr.ErrorContext(), "step_name")

	// Wrap at workflow level
	workflowErr := errors.NewWorkflowExecutionError("wf-123", "job-456", "build", nodeErr)
	assert.Equal(t, "node_not_registered", workflowErr.ErrorType())
	assert.False(t, workflowErr.IsRetryable())
	assert.Contains(t, workflowErr.ErrorContext(), "workflow_id")
	assert.Contains(t, workflowErr.ErrorContext(), "job_id")

	// Verify error chain
	assert.Equal(t, nodeErr, workflowErr.Cause)

	// Verify JSON serialization preserves context
	jsonData, err := workflowErr.ToJSON()
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(jsonData, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "node_not_registered", parsed["type"])
	assert.Contains(t, parsed, "context")
	assert.Contains(t, parsed, "cause")
}

// TestSensitiveDataSanitization tests that sensitive data is properly redacted
func TestSensitiveDataSanitization(t *testing.T) {
	// Create error with sensitive context
	err := &errors.BaseError{
		Type:    "test_error",
		Message: "Test error with sensitive data",
		Context: map[string]interface{}{
			"workflow_id": "wf-123",
			"password":    "super-secret",
			"api_key":     "sk-1234567890",
			"token":       "bearer-token",
			"username":    "admin", // Not sensitive
		},
	}

	// Convert to JSON (should sanitize)
	jsonData, jsonErr := err.ToJSON()
	require.NoError(t, jsonErr)

	var parsed map[string]interface{}
	unmarshalErr := json.Unmarshal(jsonData, &parsed)
	require.NoError(t, unmarshalErr)

	context := parsed["context"].(map[string]interface{})

	// Verify sensitive fields are redacted
	assert.Equal(t, "***REDACTED***", context["password"])
	assert.Equal(t, "***REDACTED***", context["api_key"])
	assert.Equal(t, "***REDACTED***", context["token"])

	// Verify non-sensitive fields are preserved
	assert.Equal(t, "wf-123", context["workflow_id"])
	assert.Equal(t, "admin", context["username"])

	// Also test RFC 7807 conversion
	rfc7807 := errors.ToRFC7807(err, "/test")
	assert.Equal(t, "***REDACTED***", rfc7807.Metadata["password"])
	assert.Equal(t, "wf-123", rfc7807.Metadata["workflow_id"])
}

// TestErrorClassifierIntegration tests error classification in real scenarios
func TestErrorClassifierIntegration(t *testing.T) {
	classifier := errors.NewErrorClassifier()

	tests := []struct {
		name      string
		err       error
		expected  string
		retryable bool
	}{
		{
			name:      "WaterflowError interface priority",
			err:       errors.NewValidationError("test", nil),
			expected:  "validation_error",
			retryable: false,
		},
		{
			name:      "Heuristic matching - timeout",
			err:       stderrors.New("context deadline exceeded"),
			expected:  "deadline_exceeded",
			retryable: true,
		},
		{
			name:      "Heuristic matching - connection",
			err:       stderrors.New("connection refused"),
			expected:  "connection_refused",
			retryable: true,
		},
		{
			name:      "Unknown error defaults to unknown_error",
			err:       stderrors.New("some random error"),
			expected:  "unknown_error",
			retryable: false, // Default classifier config
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errType := classifier.ClassifyError(tt.err)
			assert.Equal(t, tt.expected, errType)
			assert.Equal(t, tt.retryable, classifier.IsRetryable(errType))
		})
	}
}

// TestErrorWrappingPreservesRetryability tests that error wrapping preserves retryability
func TestErrorWrappingPreservesRetryability(t *testing.T) {
	// Create a retryable error
	timeoutErr := errors.NewWorkflowTimeoutError("wf-123", 0)
	assert.True(t, timeoutErr.IsRetryable())

	// Wrap it
	wrapped := errors.WrapWithContext(timeoutErr, "Operation timed out", map[string]interface{}{
		"attempt": 3,
	})

	// Should preserve retryability
	assert.True(t, wrapped.IsRetryable())
	assert.Equal(t, "deadline_exceeded", wrapped.ErrorType())

	// Create a non-retryable error
	notFoundErr := errors.NewWorkflowNotFoundError("wf-456")
	assert.False(t, notFoundErr.IsRetryable())

	// Wrap it
	wrappedNotFound := errors.WrapWithContext(notFoundErr, "Workflow missing", map[string]interface{}{
		"search_path": "/workflows",
	})

	// Should preserve non-retryability
	assert.False(t, wrappedNotFound.IsRetryable())
	assert.Equal(t, "not_found", wrappedNotFound.ErrorType())
}
