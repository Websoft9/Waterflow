package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewNodeExecutionError tests NodeExecutionError construction
func TestNewNodeExecutionError(t *testing.T) {
	tests := []struct {
		name            string
		nodeType        string
		stepName        string
		cause           error
		expectRetryable bool
		expectType      string
	}{
		{
			name:            "retryable network error",
			nodeType:        "http/request",
			stepName:        "api-call",
			cause:           errors.New("connection timeout"),
			expectRetryable: true,
			expectType:      "deadline_exceeded",
		},
		{
			name:            "non-retryable validation error",
			nodeType:        "shell/command",
			stepName:        "build",
			cause:           errors.New("validation failed"),
			expectRetryable: false,
			expectType:      "validation_error",
		},
		{
			name:            "service unavailable",
			nodeType:        "docker/exec",
			stepName:        "deploy",
			cause:           errors.New("service unavailable"),
			expectRetryable: true,
			expectType:      "service_unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewNodeExecutionError(tt.nodeType, tt.stepName, tt.cause)

			assert.NotNil(t, err)
			assert.Equal(t, tt.expectType, err.ErrorType())
			assert.Equal(t, tt.expectRetryable, err.IsRetryable())
			assert.Equal(t, tt.nodeType, err.NodeType)
			assert.Equal(t, tt.stepName, err.StepName)
			assert.Contains(t, err.Error(), tt.nodeType)
			assert.True(t, errors.Is(err, tt.cause))

			// Check context
			ctx := err.ErrorContext()
			assert.Equal(t, tt.nodeType, ctx["node_type"])
			assert.Equal(t, tt.stepName, ctx["step_name"])
		})
	}
}

// TestNewNodeNotFoundError tests NodeNotFoundError construction
func TestNewNodeNotFoundError(t *testing.T) {
	nodeType := "custom/unknown"
	err := NewNodeNotFoundError(nodeType)

	assert.NotNil(t, err)
	assert.Equal(t, "node_not_registered", err.ErrorType())
	assert.Equal(t, nodeType, err.NodeType)
	assert.False(t, err.IsRetryable())
	assert.Contains(t, err.Error(), nodeType)
	assert.Equal(t, nodeType, err.ErrorContext()["node_type"])
}

// TestNewNodeParameterError tests NodeParameterError construction
func TestNewNodeParameterError(t *testing.T) {
	err := NewNodeParameterError("http/request", "method", "string", 123)

	assert.NotNil(t, err)
	assert.Equal(t, "invalid_argument", err.ErrorType())
	assert.Equal(t, "http/request", err.NodeType)
	assert.Equal(t, "method", err.ParamName)
	assert.Equal(t, "string", err.Expected)
	assert.Equal(t, 123, err.Actual)
	assert.False(t, err.IsRetryable())
	assert.Contains(t, err.Error(), "method")
	assert.Contains(t, err.Error(), "http/request")

	// Check context
	ctx := err.ErrorContext()
	assert.Equal(t, "http/request", ctx["node_type"])
	assert.Equal(t, "method", ctx["param_name"])
	assert.Equal(t, "string", ctx["expected"])
	assert.Equal(t, 123, ctx["actual"])
}

// TestNodeErrors_InterfaceCompliance tests interface compliance
func TestNodeErrors_InterfaceCompliance(t *testing.T) {
	errors := []WaterflowError{
		NewNodeExecutionError("shell", "test", errors.New("test")),
		NewNodeNotFoundError("custom/unknown"),
		NewNodeParameterError("http/request", "url", "string", nil),
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

// TestNodeExecutionError_WithInputs tests adding inputs context
func TestNodeExecutionError_WithInputs(t *testing.T) {
	err := NewNodeExecutionError("http/request", "api-call", errors.New("timeout"))
	err.Inputs = map[string]interface{}{
		"url":    "https://api.example.com",
		"method": "POST",
	}

	assert.NotNil(t, err.Inputs)
	assert.Equal(t, "https://api.example.com", err.Inputs["url"])
	assert.Equal(t, "POST", err.Inputs["method"])
}
