package builtin

import (
	"context"

	"github.com/Websoft9/waterflow/pkg/dsl/node"
)

type CheckoutNode struct{}

func (n *CheckoutNode) Name() string    { return "checkout" }
func (n *CheckoutNode) Version() string { return "v1" }
func (n *CheckoutNode) Params() map[string]node.ParamSpec {
	return map[string]node.ParamSpec{
		"repository": {Type: "string", Required: true, Description: "Git repository URL"},
		"branch":     {Type: "string", Required: false, Description: "Branch name", Default: "main"},
	}
}

func (n *CheckoutNode) Execute(ctx context.Context, inputs map[string]interface{}) (*node.NodeResult, error) {
	// Stub implementation - will be implemented in future stories
	result := node.NewNodeResult()
	result.SetOutput("status", "checkout executed (stub)")
	result.AddLog("CheckoutNode.Execute called (stub implementation)")
	return result, nil
}

func (n *CheckoutNode) Metadata() node.NodeMetadata {
	return node.NodeMetadata{
		Description: "Checks out code from a Git repository",
		Category:    "exec",
		InputSchema: n.Params(),
		OutputSchema: map[string]interface{}{
			"status": "string",
		},
	}
}

type RunNode struct{}

func (n *RunNode) Name() string    { return "run" }
func (n *RunNode) Version() string { return "v1" }
func (n *RunNode) Params() map[string]node.ParamSpec {
	return map[string]node.ParamSpec{
		"command": {Type: "string", Required: true, Description: "Shell command"},
	}
}

func (n *RunNode) Execute(ctx context.Context, inputs map[string]interface{}) (*node.NodeResult, error) {
	// Stub implementation - will be implemented in future stories
	result := node.NewNodeResult()
	result.SetOutput("status", "run executed (stub)")
	result.AddLog("RunNode.Execute called (stub implementation)")
	return result, nil
}

func (n *RunNode) Metadata() node.NodeMetadata {
	return node.NodeMetadata{
		Description: "Executes a shell command",
		Category:    "exec",
		InputSchema: n.Params(),
		OutputSchema: map[string]interface{}{
			"status": "string",
		},
	}
}

// RegisterBuiltinNodes registers all builtin nodes to the registry
func RegisterBuiltinNodes(registry *node.Registry) error {
	// Register checkout@v1
	if err := registry.Register(&CheckoutNode{}); err != nil {
		return err
	}

	// Register run@v1
	if err := registry.Register(&RunNode{}); err != nil {
		return err
	}

	return nil
}
