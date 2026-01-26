package errors

import "fmt"

// NodeExecutionError represents a node execution failure.
// Retryability depends on the underlying cause.
type NodeExecutionError struct {
	*BaseError
	NodeType string
	StepName string
	Inputs   map[string]interface{}
}

// NewNodeExecutionError creates a new NodeExecutionError.
// It automatically classifies the error and determines retryability based on the cause.
func NewNodeExecutionError(nodeType, stepName string, cause error) *NodeExecutionError {
	retryable := IsRetryableError(cause)
	classifier := NewErrorClassifier()
	errType := classifier.ClassifyError(cause)

	err := &NodeExecutionError{
		BaseError: &BaseError{
			Type:      errType,
			Message:   fmt.Sprintf("Node execution failed: %s", nodeType),
			Cause:     cause,
			Retryable: retryable,
			Context: map[string]interface{}{
				"node_type": nodeType,
				"step_name": stepName,
			},
		},
		NodeType: nodeType,
		StepName: stepName,
	}

	// Collect stack trace if in debug mode
	if shouldCollectStackTrace() {
		_ = err.WithStackTrace()
	}

	return err
}

// NodeNotFoundError represents a node type that is not registered.
// This is a permanent error (non-retryable).
type NodeNotFoundError struct {
	*BaseError
	NodeType string
}

// NewNodeNotFoundError creates a new NodeNotFoundError.
func NewNodeNotFoundError(nodeType string) *NodeNotFoundError {
	return &NodeNotFoundError{
		BaseError: &BaseError{
			Type:      "node_not_registered",
			Message:   fmt.Sprintf("Node not found: %s", nodeType),
			Retryable: false,
			Context: map[string]interface{}{
				"node_type": nodeType,
			},
		},
		NodeType: nodeType,
	}
}

// NodeParameterError represents invalid node parameters.
// This is a permanent error (non-retryable).
type NodeParameterError struct {
	*BaseError
	NodeType  string
	ParamName string
	Expected  string
	Actual    interface{}
}

// NewNodeParameterError creates a new NodeParameterError.
func NewNodeParameterError(nodeType, paramName, expected string, actual interface{}) *NodeParameterError {
	return &NodeParameterError{
		BaseError: &BaseError{
			Type:      "invalid_argument",
			Message:   fmt.Sprintf("Invalid parameter %s for node %s", paramName, nodeType),
			Retryable: false,
			Context: map[string]interface{}{
				"node_type":  nodeType,
				"param_name": paramName,
				"expected":   expected,
				"actual":     actual,
			},
		},
		NodeType:  nodeType,
		ParamName: paramName,
		Expected:  expected,
		Actual:    actual,
	}
}
