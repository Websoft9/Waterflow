package main

import (
	"context"
	"time"

	"github.com/Websoft9/waterflow/pkg/dsl/node"
)

type EchoNode struct{}

func (n *EchoNode) Name() string {
	return "flow/echo"
}

func (n *EchoNode) Version() string {
	return "v1"
}

func (n *EchoNode) Params() map[string]node.ParamSpec {
	return map[string]node.ParamSpec{
		"message": {
			Type:        "string",
			Required:    true,
			Description: "Message to echo",
		},
	}
}

func (n *EchoNode) Execute(ctx context.Context, inputs map[string]interface{}) (*node.NodeResult, error) {
	start := time.Now()

	if err := node.ValidateInputs(inputs, n.Params()); err != nil {
		return nil, err
	}

	result := node.NewNodeResult()
	result.SetOutput("message", inputs["message"])
	result.AddLog("Echoed message")
	result.Duration = time.Since(start)

	return result, nil
}

func (n *EchoNode) Metadata() node.NodeMetadata {
	return node.NodeMetadata{
		Description: "Echoes input message to output",
		Category:    "flow",
		InputSchema: n.Params(),
		OutputSchema: map[string]interface{}{
			"message": "string",
		},
	}
}

func Register() node.Node {
	return &EchoNode{}
}
