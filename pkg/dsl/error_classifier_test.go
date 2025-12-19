package dsl

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorClassifier_IsRetryable(t *testing.T) {
	classifier := NewErrorClassifier()

	tests := []struct {
		name      string
		errType   string
		retryable bool
	}{
		{
			name:      "validation_error is not retryable",
			errType:   "validation_error",
			retryable: false,
		},
		{
			name:      "schema_error is not retryable",
			errType:   "schema_error",
			retryable: false,
		},
		{
			name:      "not_found is not retryable",
			errType:   "not_found",
			retryable: false,
		},
		{
			name:      "permission_denied is not retryable",
			errType:   "permission_denied",
			retryable: false,
		},
		{
			name:      "invalid_argument is not retryable",
			errType:   "invalid_argument",
			retryable: false,
		},
		{
			name:      "node_not_registered is not retryable",
			errType:   "node_not_registered",
			retryable: false,
		},
		{
			name:      "network_timeout is retryable",
			errType:   "network_timeout",
			retryable: true,
		},
		{
			name:      "connection_refused is retryable",
			errType:   "connection_refused",
			retryable: true,
		},
		{
			name:      "service_unavailable is retryable",
			errType:   "service_unavailable",
			retryable: true,
		},
		{
			name:      "internal_error is retryable",
			errType:   "internal_error",
			retryable: true,
		},
		{
			name:      "deadline_exceeded is retryable",
			errType:   "deadline_exceeded",
			retryable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.IsRetryable(tt.errType)
			assert.Equal(t, tt.retryable, result)
		})
	}
}

func TestErrorClassifier_ClassifyError(t *testing.T) {
	classifier := NewErrorClassifier()

	tests := []struct {
		name         string
		err          error
		expectedType string
	}{
		{
			name:         "Nil error",
			err:          nil,
			expectedType: "",
		},
		{
			name:         "Validation error",
			err:          errors.New("validation failed: invalid YAML syntax"),
			expectedType: "validation_error",
		},
		{
			name:         "Schema error",
			err:          errors.New("schema validation failed"),
			expectedType: "schema_error",
		},
		{
			name:         "Not found error",
			err:          errors.New("resource not found"),
			expectedType: "not_found",
		},
		{
			name:         "404 error",
			err:          errors.New("HTTP 404: page not found"),
			expectedType: "not_found",
		},
		{
			name:         "Permission denied",
			err:          errors.New("permission denied: access forbidden"),
			expectedType: "permission_denied",
		},
		{
			name:         "403 error",
			err:          errors.New("HTTP 403 Forbidden"),
			expectedType: "permission_denied",
		},
		{
			name:         "Invalid argument",
			err:          errors.New("invalid argument: missing required field"),
			expectedType: "invalid_argument",
		},
		{
			name:         "400 error",
			err:          errors.New("HTTP 400 Bad Request"),
			expectedType: "invalid_argument",
		},
		{
			name:         "Node not registered",
			err:          errors.New("node not registered: unknown@v1"),
			expectedType: "node_not_registered",
		},
		{
			name:         "Timeout error",
			err:          errors.New("operation timed out"),
			expectedType: "deadline_exceeded",
		},
		{
			name:         "Context deadline exceeded",
			err:          errors.New("context deadline exceeded"),
			expectedType: "deadline_exceeded",
		},
		{
			name:         "Connection refused",
			err:          errors.New("connection refused"),
			expectedType: "connection_refused",
		},
		{
			name:         "Dial error",
			err:          errors.New("dial tcp: connection refused"),
			expectedType: "connection_refused",
		},
		{
			name:         "503 error",
			err:          errors.New("HTTP 503 Service Unavailable"),
			expectedType: "service_unavailable",
		},
		{
			name:         "500 error",
			err:          errors.New("HTTP 500 Internal Server Error"),
			expectedType: "internal_error",
		},
		{
			name:         "Unknown error",
			err:          errors.New("something went wrong"),
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

func TestErrorClassifier_Integration(t *testing.T) {
	classifier := NewErrorClassifier()

	tests := []struct {
		name               string
		err                error
		shouldRetry        bool
		expectedClassified string
	}{
		{
			name:               "Retryable network error",
			err:                errors.New("connection refused"),
			shouldRetry:        true,
			expectedClassified: "connection_refused",
		},
		{
			name:               "Non-retryable validation error",
			err:                errors.New("validation failed"),
			shouldRetry:        false,
			expectedClassified: "validation_error",
		},
		{
			name:               "Retryable timeout",
			err:                errors.New("request timeout"),
			shouldRetry:        true,
			expectedClassified: "deadline_exceeded",
		},
		{
			name:               "Non-retryable permission error",
			err:                errors.New("permission denied"),
			shouldRetry:        false,
			expectedClassified: "permission_denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errType := classifier.ClassifyError(tt.err)
			assert.Equal(t, tt.expectedClassified, errType)

			retryable := classifier.IsRetryable(errType)
			assert.Equal(t, tt.shouldRetry, retryable)
		})
	}
}
