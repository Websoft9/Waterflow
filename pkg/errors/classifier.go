package errors

import (
	"errors"
	"strings"
)

// ErrorClassifier classifies errors into types and determines retryability.
// It provides a configurable strategy for handling unknown errors.
type ErrorClassifier struct {
	// nonRetryableErrors maps error types that should never be retried
	nonRetryableErrors map[string]bool
	// unknownErrorIsRetryable determines default behavior for unknown errors
	unknownErrorIsRetryable bool
}

// ErrorClassifierConfig holds configuration for error classification.
type ErrorClassifierConfig struct {
	// UnknownErrorIsRetryable determines whether unknown errors are retryable
	// Default: false (safe strategy - avoid infinite retries)
	UnknownErrorIsRetryable bool
}

// NewErrorClassifier creates an ErrorClassifier with default config.
// Unknown errors are not retryable by default (safe strategy).
func NewErrorClassifier() *ErrorClassifier {
	return NewErrorClassifierWithConfig(ErrorClassifierConfig{
		UnknownErrorIsRetryable: false,
	})
}

// NewErrorClassifierWithConfig creates an ErrorClassifier with custom config.
func NewErrorClassifierWithConfig(config ErrorClassifierConfig) *ErrorClassifier {
	return &ErrorClassifier{
		nonRetryableErrors: map[string]bool{
			// Permanent errors - should never retry
			"validation_error":    true,
			"schema_error":        true,
			"not_found":           true,
			"permission_denied":   true,
			"invalid_argument":    true,
			"node_not_registered": true,
			"plugin_load_error":   true,
			"cancelled":           true,
		},
		unknownErrorIsRetryable: config.UnknownErrorIsRetryable,
	}
}

// IsRetryable determines whether an error type should be retried.
// It checks the non-retryable list first, then applies unknown error strategy.
func (c *ErrorClassifier) IsRetryable(errType string) bool {
	// Check if in explicit non-retryable list
	if c.nonRetryableErrors[errType] {
		return false
	}

	// unknown_error follows configured strategy
	if errType == "unknown_error" {
		return c.unknownErrorIsRetryable
	}

	// Other known types default to retryable (temporary errors)
	return true
}

// ClassifyError infers error type from an error object.
// It uses the following priority:
// 1. WaterflowError interface (most accurate)
// 2. Type assertion for known error types
// 3. Heuristic pattern matching on error message (fallback)
func (c *ErrorClassifier) ClassifyError(err error) string {
	if err == nil {
		return "unknown_error"
	}

	// Priority 1: Check WaterflowError interface
	if wfErr, ok := err.(WaterflowError); ok {
		return wfErr.ErrorType()
	}

	// Priority 2: Check known concrete error types
	// Note: ValidationError and other types defined in this package
	// will be checked after they are defined (circular reference handled)
	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return "validation_error"
	}

	var nodeNotFoundErr *NodeNotFoundError
	if errors.As(err, &nodeNotFoundErr) {
		return "node_not_registered"
	}

	var workflowNotFoundErr *WorkflowNotFoundError
	if errors.As(err, &workflowNotFoundErr) {
		return "not_found"
	}

	// Priority 3: Heuristic pattern matching (fallback)
	errMsg := strings.ToLower(err.Error())

	// Timeout/deadline errors
	if strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "deadline exceeded") ||
		strings.Contains(errMsg, "context deadline exceeded") {
		return "deadline_exceeded"
	}

	// Connection errors
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "connection reset") ||
		strings.Contains(errMsg, "broken pipe") {
		return "connection_refused"
	}

	// Service unavailable
	if strings.Contains(errMsg, "503") ||
		strings.Contains(errMsg, "service unavailable") ||
		strings.Contains(errMsg, "temporarily unavailable") {
		return "service_unavailable"
	}

	// Validation errors
	if strings.Contains(errMsg, "validation") ||
		strings.Contains(errMsg, "invalid yaml") ||
		strings.Contains(errMsg, "syntax error") {
		return "validation_error"
	}

	// Not found errors
	if strings.Contains(errMsg, "not found") ||
		strings.Contains(errMsg, "404") {
		return "not_found"
	}

	// Permission denied
	if strings.Contains(errMsg, "permission denied") ||
		strings.Contains(errMsg, "forbidden") ||
		strings.Contains(errMsg, "403") {
		return "permission_denied"
	}

	// Default: unknown error
	return "unknown_error"
}

// IsRetryableError is a global helper to check if an error should be retried.
// It creates a default classifier and checks retryability.
func IsRetryableError(err error) bool {
	classifier := NewErrorClassifier()
	errType := classifier.ClassifyError(err)
	return classifier.IsRetryable(errType)
}
