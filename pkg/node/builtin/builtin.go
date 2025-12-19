package builtin

import "github.com/Websoft9/waterflow/pkg/node"

type CheckoutNode struct{}

func (n *CheckoutNode) Name() string    { return "checkout" }
func (n *CheckoutNode) Version() string { return "v1" }
func (n *CheckoutNode) Params() map[string]node.ParamSpec {
	return map[string]node.ParamSpec{
		"repository": {Type: "string", Required: true, Description: "Git repository URL"},
		"branch":     {Type: "string", Required: false, Description: "Branch name", Default: "main"},
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
