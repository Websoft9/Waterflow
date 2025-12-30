//go:build integration
// +build integration

package main

import (
	"context"
	"plugin"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/websoft9/waterflow/pkg/dsl/node"
)

func TestShellPlugin_Load(t *testing.T) {
	// 1. Load plugin
	p, err := plugin.Open("shell.so")
	require.NoError(t, err, "Failed to load plugin")

	// 2. Lookup Register function
	registerSymbol, err := p.Lookup("Register")
	require.NoError(t, err, "Register function not found")

	// 3. Call Register
	register, ok := registerSymbol.(func() node.Node)
	require.True(t, ok, "Register has wrong signature")

	nodeInstance := register()
	require.NotNil(t, nodeInstance)

	// 4. Verify node properties
	assert.Equal(t, "exec/shell", nodeInstance.Name())
	assert.Equal(t, "v1", nodeInstance.Version())
}

func TestShellPlugin_Execute(t *testing.T) {
	// Load plugin
	p, _ := plugin.Open("shell.so")
	registerSymbol, _ := p.Lookup("Register")
	register := registerSymbol.(func() node.Node)
	nodeInstance := register()

	// Execute command
	inputs := map[string]interface{}{
		"command": "whoami",
	}

	result, err := nodeInstance.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.NotEmpty(t, result.Outputs["stdout"])
	assert.Equal(t, 0, result.Outputs["exit_code"])
}

func TestShellPlugin_SystemCommands(t *testing.T) {
	// Load plugin
	p, _ := plugin.Open("shell.so")
	registerSymbol, _ := p.Lookup("Register")
	register := registerSymbol.(func() node.Node)
	nodeInstance := register()

	tests := []struct {
		name    string
		command string
		check   func(*testing.T, *node.NodeResult)
	}{
		{
			name:    "pwd",
			command: "pwd",
			check: func(t *testing.T, result *node.NodeResult) {
				assert.Contains(t, result.Outputs["stdout"], "/")
			},
		},
		{
			name:    "date",
			command: "date",
			check: func(t *testing.T, result *node.NodeResult) {
				assert.NotEmpty(t, result.Outputs["stdout"])
			},
		},
		{
			name:    "uname",
			command: "uname -s",
			check: func(t *testing.T, result *node.NodeResult) {
				stdout := result.Outputs["stdout"].(string)
				assert.True(t, len(stdout) > 0, "uname should return OS name")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputs := map[string]interface{}{
				"command": tt.command,
			}

			result, err := nodeInstance.Execute(context.Background(), inputs)

			require.NoError(t, err)
			assert.Equal(t, 0, result.Outputs["exit_code"])
			tt.check(t, result)
		})
	}
}

func TestShellPlugin_FileOperations(t *testing.T) {
	// Load plugin
	p, _ := plugin.Open("shell.so")
	registerSymbol, _ := p.Lookup("Register")
	register := registerSymbol.(func() node.Node)
	nodeInstance := register()

	// Create a temp file
	inputs := map[string]interface{}{
		"command": "touch /tmp/waterflow-test.txt && echo 'test content' > /tmp/waterflow-test.txt",
	}

	result, err := nodeInstance.Execute(context.Background(), inputs)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Outputs["exit_code"])

	// Read the file
	inputs = map[string]interface{}{
		"command": "cat /tmp/waterflow-test.txt",
	}

	result, err = nodeInstance.Execute(context.Background(), inputs)
	require.NoError(t, err)
	assert.Contains(t, result.Outputs["stdout"], "test content")

	// Clean up
	inputs = map[string]interface{}{
		"command": "rm /tmp/waterflow-test.txt",
	}

	result, err = nodeInstance.Execute(context.Background(), inputs)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Outputs["exit_code"])
}
