package errors

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBaseError_Error tests the Error() method
func TestBaseError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *BaseError
		expected string
	}{
		{
			name: "error without cause",
			err: &BaseError{
				Type:    "validation_error",
				Message: "Invalid YAML syntax",
			},
			expected: "validation_error: Invalid YAML syntax",
		},
		{
			name: "error with cause",
			err: &BaseError{
				Type:    "node_error",
				Message: "Node execution failed",
				Cause:   errors.New("connection timeout"),
			},
			expected: "node_error: Node execution failed (caused by: connection timeout)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

// TestBaseError_Unwrap tests error chain support
func TestBaseError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	wrappedErr := &BaseError{
		Type:    "wrapped_error",
		Message: "Wrapped error",
		Cause:   originalErr,
	}

	// Test Unwrap
	assert.Equal(t, originalErr, wrappedErr.Unwrap())

	// Test errors.Is
	assert.True(t, errors.Is(wrappedErr, originalErr))

	// Test errors.As
	var baseErr *BaseError
	assert.True(t, errors.As(wrappedErr, &baseErr))
	assert.Equal(t, "wrapped_error", baseErr.Type)
}

// TestBaseError_WaterflowErrorInterface tests WaterflowError interface implementation
func TestBaseError_WaterflowErrorInterface(t *testing.T) {
	err := &BaseError{
		Type:      "test_error",
		Message:   "Test error message",
		Retryable: true,
		Context: map[string]interface{}{
			"workflow_id": "wf-123",
			"step_name":   "deploy",
		},
	}

	// Test interface methods
	assert.Equal(t, "test_error", err.ErrorType())
	assert.Equal(t, "Test error message", err.ErrorMessage())
	assert.True(t, err.IsRetryable())
	assert.Equal(t, "wf-123", err.ErrorContext()["workflow_id"])
	assert.Equal(t, "deploy", err.ErrorContext()["step_name"])
}

// TestBaseError_WithContext tests context addition
func TestBaseError_WithContext(t *testing.T) {
	err := &BaseError{
		Type:    "test_error",
		Message: "Test error",
	}

	// Add context (chainable)
	_ = err.WithContext("workflow_id", "wf-123").
		WithContext("job_id", "job-456").
		WithContext("step_name", "deploy")

	assert.Equal(t, "wf-123", err.Context["workflow_id"])
	assert.Equal(t, "job-456", err.Context["job_id"])
	assert.Equal(t, "deploy", err.Context["step_name"])
}

// TestBaseError_WithCause tests cause wrapping
func TestBaseError_WithCause(t *testing.T) {
	originalErr := errors.New("original error")
	err := &BaseError{
		Type:    "wrapped_error",
		Message: "Wrapped error",
	}

	_ = err.WithCause(originalErr)

	assert.Equal(t, originalErr, err.Cause)
	assert.True(t, errors.Is(err, originalErr))
}

// TestBaseError_ToJSON tests JSON serialization
func TestBaseError_ToJSON(t *testing.T) {
	err := &BaseError{
		Type:    "validation_error",
		Message: "YAML syntax error",
		Context: map[string]interface{}{
			"workflow_id": "wf-123",
			"line":        10,
		},
		Retryable: false,
	}

	data, jsonErr := err.ToJSON()
	require.NoError(t, jsonErr)
	require.NotEmpty(t, data)

	// Check JSON contains expected fields
	jsonStr := string(data)
	assert.Contains(t, jsonStr, "validation_error")
	assert.Contains(t, jsonStr, "YAML syntax error")
	assert.Contains(t, jsonStr, "wf-123")
}

// TestBaseError_ToJSON_WithCause tests JSON serialization with cause
func TestBaseError_ToJSON_WithCause(t *testing.T) {
	originalErr := errors.New("connection refused")
	err := &BaseError{
		Type:      "node_error",
		Message:   "Node execution failed",
		Cause:     originalErr,
		Retryable: true,
	}

	data, jsonErr := err.ToJSON()
	require.NoError(t, jsonErr)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, "node_error")
	assert.Contains(t, jsonStr, "connection refused")
}

// TestBaseError_WithStackTrace tests stack trace collection
func TestBaseError_WithStackTrace(t *testing.T) {
	err := &BaseError{
		Type:    "test_error",
		Message: "Test error with stack trace",
	}

	_ = err.WithStackTrace()

	assert.NotEmpty(t, err.StackTrace)
	// Stack trace should contain test runner or runtime frames
	assert.Contains(t, err.StackTrace, "testing.tRunner")
}

// TestBaseError_StackTrace_InJSON tests stack trace in JSON output
func TestBaseError_StackTrace_InJSON(t *testing.T) {
	err := &BaseError{
		Type:    "test_error",
		Message: "Test error",
	}
	_ = err.WithStackTrace()

	data, jsonErr := err.ToJSON()
	require.NoError(t, jsonErr)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, "stack_trace")
}

// TestShouldCollectStackTrace tests debug mode detection
func TestShouldCollectStackTrace(t *testing.T) {
	// Test DEBUG=true
	_ = os.Setenv("DEBUG", "true")
	assert.True(t, shouldCollectStackTrace())

	// Test DEBUG=1
	_ = os.Setenv("DEBUG", "1")
	assert.True(t, shouldCollectStackTrace())

	// Test DEBUG=false
	_ = os.Setenv("DEBUG", "false")
	assert.False(t, shouldCollectStackTrace())

	// Test DEBUG not set
	_ = os.Unsetenv("DEBUG")
	assert.False(t, shouldCollectStackTrace())
}

// TestWrapError tests error wrapping
func TestWrapError(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		message         string
		expectedType    string
		expectedCause   bool
		expectRetryable bool
	}{
		{
			name:            "wrap nil error",
			err:             nil,
			message:         "wrapped message",
			expectedType:    "",
			expectedCause:   false,
			expectRetryable: false,
		},
		{
			name: "wrap WaterflowError",
			err: &BaseError{
				Type:      "validation_error",
				Message:   "original message",
				Retryable: false,
			},
			message:         "wrapped message",
			expectedType:    "validation_error",
			expectedCause:   true,
			expectRetryable: false,
		},
		{
			name:            "wrap standard error (timeout)",
			err:             errors.New("connection timeout"),
			message:         "operation failed",
			expectedType:    "deadline_exceeded",
			expectedCause:   true,
			expectRetryable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := WrapError(tt.err, "", tt.message)

			if tt.err == nil {
				assert.Nil(t, wrapped)
				return
			}

			assert.NotNil(t, wrapped)
			assert.Equal(t, tt.expectedType, wrapped.Type)
			assert.Equal(t, tt.message, wrapped.Message)
			assert.Equal(t, tt.expectRetryable, wrapped.Retryable)

			if tt.expectedCause {
				assert.NotNil(t, wrapped.Cause)
				assert.True(t, errors.Is(wrapped, tt.err))
			}
		})
	}
}

// TestWrapWithContext tests context wrapping
func TestWrapWithContext(t *testing.T) {
	originalErr := errors.New("original error")
	context := map[string]interface{}{
		"workflow_id": "wf-123",
		"step_name":   "deploy",
	}

	wrapped := WrapWithContext(originalErr, "operation failed", context)

	require.NotNil(t, wrapped)
	assert.Equal(t, "operation failed", wrapped.Message)
	assert.Equal(t, "wf-123", wrapped.Context["workflow_id"])
	assert.Equal(t, "deploy", wrapped.Context["step_name"])
	assert.True(t, errors.Is(wrapped, originalErr))
}

// TestWrapWithContext_MergeContext tests context merging
func TestWrapWithContext_MergeContext(t *testing.T) {
	originalErr := &BaseError{
		Type:    "test_error",
		Message: "original error",
		Context: map[string]interface{}{
			"existing_key": "existing_value",
		},
	}

	newContext := map[string]interface{}{
		"new_key":      "new_value",
		"existing_key": "should_not_overwrite",
	}

	wrapped := WrapWithContext(originalErr, "wrapped error", newContext)

	require.NotNil(t, wrapped)
	// Existing key should not be overwritten
	assert.Equal(t, "existing_value", wrapped.Context["existing_key"])
	// New key should be added
	assert.Equal(t, "new_value", wrapped.Context["new_key"])
}

// TestSanitizeContext tests sensitive field sanitization
func TestSanitizeContext(t *testing.T) {
	tests := []struct {
		name     string
		context  map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:     "nil context",
			context:  nil,
			expected: nil,
		},
		{
			name: "no sensitive fields",
			context: map[string]interface{}{
				"workflow_id": "wf-123",
				"step_name":   "deploy",
			},
			expected: map[string]interface{}{
				"workflow_id": "wf-123",
				"step_name":   "deploy",
			},
		},
		{
			name: "password field",
			context: map[string]interface{}{
				"workflow_id": "wf-123",
				"password":    "secret123",
			},
			expected: map[string]interface{}{
				"workflow_id": "wf-123",
				"password":    "***REDACTED***",
			},
		},
		{
			name: "multiple sensitive fields",
			context: map[string]interface{}{
				"workflow_id": "wf-123",
				"api_key":     "sk-12345",
				"token":       "bearer xyz",
				"secret":      "my-secret",
			},
			expected: map[string]interface{}{
				"workflow_id": "wf-123",
				"api_key":     "***REDACTED***",
				"token":       "***REDACTED***",
				"secret":      "***REDACTED***",
			},
		},
		{
			name: "case insensitive matching",
			context: map[string]interface{}{
				"workflow_id": "wf-123",
				"PASSWORD":    "secret123",
				"Api_Key":     "sk-12345",
			},
			expected: map[string]interface{}{
				"workflow_id": "wf-123",
				"PASSWORD":    "***REDACTED***",
				"Api_Key":     "***REDACTED***",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeContext(tt.context)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSanitizeContextWithConfig tests custom sensitive field configuration
func TestSanitizeContextWithConfig(t *testing.T) {
	context := map[string]interface{}{
		"workflow_id": "wf-123",
		"custom_key":  "custom_secret",
		"password":    "secret123",
	}

	config := SensitiveFieldsConfig{
		AdditionalFields: []string{"custom_key"},
	}

	result := SanitizeContextWithConfig(context, config)

	assert.Equal(t, "wf-123", result["workflow_id"])
	assert.Equal(t, "***REDACTED***", result["password"])
	assert.Equal(t, "***REDACTED***", result["custom_key"])
}
