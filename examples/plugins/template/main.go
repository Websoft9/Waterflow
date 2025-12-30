package main

import (
	"context"
	"time"

	"github.com/Websoft9/waterflow/pkg/dsl/node"
)

type TemplateNode struct{}

func (n *TemplateNode) Name() string {
	return "flow/template"
}

func (n *TemplateNode) Version() string {
	return "v1"
}

func (n *TemplateNode) Params() map[string]node.ParamSpec {
	min, max := 1.0, 10.0
	return map[string]node.ParamSpec{
		"message": {
			Type:        "string",
			Required:    true,
			Description: "Message to process",
		},
		"repeat": {
			Type:        "int",
			Required:    false,
			Default:     1,
			Description: "Number of times to repeat",
			MinValue:    &min,
			MaxValue:    &max,
		},
	}
}

func (n *TemplateNode) Execute(ctx context.Context, inputs map[string]interface{}) (*node.NodeResult, error) {
	start := time.Now()

	if err := node.ValidateInputs(inputs, n.Params()); err != nil {
		return nil, err
	}

	message := inputs["message"].(string)
	repeat := 1
	if r, ok := inputs["repeat"]; ok {
		repeat = r.(int)
	}

	result := node.NewNodeResult()
	result.AddLog("Execution started")

	output := ""
	for i := 0; i < repeat; i++ {
		if i > 0 {
			output += " "
		}
		output += message
	}

	result.SetOutput("result", output)
	result.SetOutput("repeat_count", repeat)
	result.Duration = time.Since(start)

	return result, nil
}

func (n *TemplateNode) Metadata() node.NodeMetadata {
	return node.NodeMetadata{
		Description: "Template node for development",
		Category:    "flow",
		InputSchema: n.Params(),
		OutputSchema: map[string]interface{}{
			"result":       "string",
			"repeat_count": "int",
		},
	}
}

func Register() node.Node {
	return &TemplateNode{}
}
