package main

import (
	"context"
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEchoNode_Name(t *testing.T) {
	n := &EchoNode{}
	assert.Equal(t, "flow/echo", n.Name())
}

func TestEchoNode_Version(t *testing.T) {
	n := &EchoNode{}
	assert.Equal(t, "v1", n.Version())
}

func TestEchoNode_Params(t *testing.T) {
	n := &EchoNode{}
	params := n.Params()
	assert.Len(t, params, 1)
	assert.True(t, params["message"].Required)
}

func TestEchoNode_Execute(t *testing.T) {
	n := &EchoNode{}

	result, err := n.Execute(context.Background(), map[string]interface{}{
		"message": "hello world",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "hello world", result.Outputs["message"])
	assert.Greater(t, len(result.Logs), 0)
}

func TestEchoNode_Metadata(t *testing.T) {
	n := &EchoNode{}
	meta := n.Metadata()

	assert.Equal(t, "Echoes input message to output", meta.Description)
	assert.Equal(t, "flow", meta.Category)
	assert.NotNil(t, meta.InputSchema)
	assert.NotNil(t, meta.OutputSchema)
}

func TestEchoNode_Validation(t *testing.T) {
	n := &EchoNode{}
	err := node.ValidateNode(n)
	assert.NoError(t, err)
}

func TestRegister(t *testing.T) {
	n := Register()
	assert.NotNil(t, n)
	assert.Equal(t, "flow/echo", n.Name())
}
