package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewErrorClassifier tests default classifier creation
func TestNewErrorClassifier(t *testing.T) {
	classifier := NewErrorClassifier()

	assert.NotNil(t, classifier)
	assert.False(t, classifier.unknownErrorIsRetryable)
}

// TestNewErrorClassifierWithConfig tests custom config
func TestNewErrorClassifierWithConfig(t *testing.T) {
	config := ErrorClassifierConfig{
		UnknownErrorIsRetryable: true,
	}
	classifier := NewErrorClassifierWithConfig(config)

	assert.NotNil(t, classifier)
	assert.True(t, classifier.unknownErrorIsRetryable)
}

// TestErrorClassifier_IsRetryable tests retryability classification
func TestErrorClassifier_IsRetryable(t *testing.T) {
	classifier := NewErrorClassifier()

	tests := []struct {
		name      string
		errType   string
		retryable bool
	}{
		// Non-retryable errors
		{"validation error", "validation_error", false},
		{"schema error", "schema_error", false},
		{"not found", "not_found", false},
		{"permission denied", "permission_denied", false},
		{"invalid argument", "invalid_argument", false},
		{"node not registered", "node_not_registered", false},
		{"plugin load error", "plugin_load_error", false},
		{"cancelled", "cancelled", false},

		// Retryable errors
		{"deadline exceeded", "deadline_exceeded", true},
		{"connection refused", "connection_refused", true},
		{"service unavailable", "service_unavailable", true},
		{"internal error", "internal_error", true},

		// Unknown error (default not retryable)
		{"unknown error", "unknown_error", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.IsRetryable(tt.errType)
			assert.Equal(t, tt.retryable, result)
		})
	}
}

// TestErrorClassifier_ClassifyError tests error classification
func TestErrorClassifier_ClassifyError(t *testing.T) {
	classifier := NewErrorClassifier()

	tests := []struct {
		name         string
		err          error
		expectedType string
	}{
		{
			name:         "nil error",
			err:          nil,
			expectedType: "unknown_error",
		},
		{
			name:         "timeout error",
			err:          errors.New("connection timeout"),
			expectedType: "deadline_exceeded",
		},
		{
			name:         "deadline exceeded error",
			err:          errors.New("context deadline exceeded"),
			expectedType: "deadline_exceeded",
		},
		{
			name:         "connection refused",
			err:          errors.New("connection refused"),
			expectedType: "connection_refused",
		},
		{
			name:         "broken pipe",
			err:          errors.New("broken pipe"),
			expectedType: "connection_refused",
		},
		{
			name:         "service unavailable",
			err:          errors.New("503 service unavailable"),
			expectedType: "service_unavailable",
		},
		{
			name:         "validation error",
			err:          errors.New("validation failed"),
			expectedType: "validation_error",
		},
		{
			name:         "not found",
			err:          errors.New("resource not found"),
			expectedType: "not_found",
		},
		{
			name:         "permission denied",
			err:          errors.New("permission denied"),
			expectedType: "permission_denied",
		},
		{
			name:         "unknown error",
			err:          errors.New("some random error"),
			expectedType: "unknown_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.ClassifyError(tt.err)
			assert.Equal(t, tt.expectedType, result)
		})
	}
}

// TestErrorClassifier_ClassifyWaterflowError tests WaterflowError classification
func TestErrorClassifier_ClassifyWaterflowError(t *testing.T) {
	classifier := NewErrorClassifier()

	// Test ValidationError
	valErr := NewValidationError("test", []FieldError{})
	assert.Equal(t, "validation_error", classifier.ClassifyError(valErr))

	// Test NodeNotFoundError
	nodeErr := NewNodeNotFoundError("test")
	assert.Equal(t, "node_not_registered", classifier.ClassifyError(nodeErr))

	// Test WorkflowNotFoundError
	wfErr := NewWorkflowNotFoundError("test")
	assert.Equal(t, "not_found", classifier.ClassifyError(wfErr))
}

// TestIsRetryableError tests global helper function
func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "timeout error (retryable)",
			err:       errors.New("connection timeout"),
			retryable: true,
		},
		{
			name:      "validation error (non-retryable)",
			err:       errors.New("validation failed"),
			retryable: false,
		},
		{
			name:      "service unavailable (retryable)",
			err:       errors.New("503 service unavailable"),
			retryable: true,
		},
		{
			name:      "not found (non-retryable)",
			err:       errors.New("not found"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			assert.Equal(t, tt.retryable, result)
		})
	}
}

// TestErrorClassifier_UnknownErrorStrategy tests unknown error configuration
func TestErrorClassifier_UnknownErrorStrategy(t *testing.T) {
	// Default: unknown errors not retryable
	defaultClassifier := NewErrorClassifier()
	assert.False(t, defaultClassifier.IsRetryable("unknown_error"))

	// Custom: unknown errors retryable
	customClassifier := NewErrorClassifierWithConfig(ErrorClassifierConfig{
		UnknownErrorIsRetryable: true,
	})
	assert.True(t, customClassifier.IsRetryable("unknown_error"))
}
