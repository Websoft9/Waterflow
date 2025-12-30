// Package node provides the core plugin interface and types for Waterflow nodes.
//
// A node is the fundamental unit of execution in Waterflow workflows.
// Each node performs a specific task (e.g., run shell command, make HTTP request)
// and is loaded as a Go plugin (.so file).
//
// Example node implementation:
//
//	type EchoNode struct{}
//
//	func (n *EchoNode) Name() string { return "flow/echo" }
//	func (n *EchoNode) Version() string { return "v1" }
//	func (n *EchoNode) Params() map[string]node.ParamSpec {
//	    return map[string]node.ParamSpec{
//	        "message": {Type: "string", Required: true},
//	    }
//	}
//	func (n *EchoNode) Execute(ctx context.Context, inputs map[string]interface{}) (*node.NodeResult, error) {
//	    result := node.NewNodeResult()
//	    result.SetOutput("message", inputs["message"])
//	    return result, nil
//	}
//	func (n *EchoNode) Metadata() node.NodeMetadata {
//	    return node.NodeMetadata{
//	        Description: "Echoes input message to output",
//	        Category: "flow",
//	    }
//	}
//
//	func Register() node.Node {
//	    return &EchoNode{}
//	}
package node

import "context"

// Node is the core interface that all Waterflow nodes must implement.
// Nodes are loaded as plugins and execute workflow steps.
//
// The interface extends the base definition from Story 1.3 with execution
// and metadata capabilities required for plugin-based architecture.
type Node interface {
	// Name returns the unique identifier for this node.
	// Format: <category>/<name> (e.g., "exec/shell", "docker/compose")
	// Required by Story 1.3 - DO NOT MODIFY signature.
	Name() string

	// Version returns the semantic version of this node.
	// Format: vX.Y.Z or vX (e.g., "v1", "v1.2.3")
	// Required by Story 1.3 - DO NOT MODIFY signature.
	Version() string

	// Params returns the parameter specifications for this node.
	// Keys are parameter names, values define their types and constraints.
	// Required by Story 1.3 - DO NOT MODIFY signature.
	Params() map[string]ParamSpec

	// Execute runs the node's logic with the given context and inputs.
	// ctx provides cancellation, timeout, and deadline control.
	// inputs contains parameter values parsed from workflow YAML.
	// Returns NodeResult with outputs, logs, and duration on success.
	// Returns error for execution failures (distinguish retriable vs permanent).
	//
	// Added in Story 3.1 for plugin execution support.
	Execute(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error)

	// Metadata returns descriptive information about the node.
	// Includes description, category, input/output schemas.
	// Used for documentation generation and validation.
	//
	// Added in Story 3.1 for comprehensive node introspection.
	Metadata() NodeMetadata
}
