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

func TestScriptPlugin_Load(t *testing.T) {
	// 1. Load plugin
	p, err := plugin.Open("script.so")
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
	assert.Equal(t, "exec/script", nodeInstance.Name())
	assert.Equal(t, "v1", nodeInstance.Version())
}

func TestScriptPlugin_Execute_BashInline(t *testing.T) {
	// Load plugin
	p, _ := plugin.Open("script.so")
	registerSymbol, _ := p.Lookup("Register")
	register := registerSymbol.(func() node.Node)
	nodeInstance := register()

	// Execute inline bash script
	inputs := map[string]interface{}{
		"script_content": "#!/bin/bash\necho 'Hello from plugin'",
		"interpreter":    "bash",
	}

	result, err := nodeInstance.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Contains(t, result.Outputs["stdout"], "Hello from plugin")
	assert.Equal(t, 0, result.Outputs["exit_code"])
	assert.Equal(t, "bash", result.Outputs["interpreter_used"])
}

func TestScriptPlugin_Execute_Python(t *testing.T) {
	// Load plugin
	p, _ := plugin.Open("script.so")
	registerSymbol, _ := p.Lookup("Register")
	register := registerSymbol.(func() node.Node)
	nodeInstance := register()

	// Execute Python script
	inputs := map[string]interface{}{
		"script_content": "#!/usr/bin/env python3\nprint('Python from plugin')",
		"interpreter":    "python3",
	}

	result, err := nodeInstance.Execute(context.Background(), inputs)

	// Skip if python3 not available
	if err != nil && result == nil {
		t.Skip("python3 not available")
	}

	require.NoError(t, err)
	assert.Contains(t, result.Outputs["stdout"], "Python from plugin")
	assert.Equal(t, 0, result.Outputs["exit_code"])
}

func TestScriptPlugin_SystemCommands(t *testing.T) {
	// Load plugin
	p, _ := plugin.Open("script.so")
	registerSymbol, _ := p.Lookup("Register")
	register := registerSymbol.(func() node.Node)
	nodeInstance := register()

	tests := []struct {
		name   string
		script string
		check  func(*testing.T, *node.NodeResult)
	}{
		{
			name:   "whoami",
			script: "#!/bin/bash\nwhoami",
			check: func(t *testing.T, result *node.NodeResult) {
				assert.NotEmpty(t, result.Outputs["stdout"])
			},
		},
		{
			name:   "pwd",
			script: "#!/bin/bash\npwd",
			check: func(t *testing.T, result *node.NodeResult) {
				assert.Contains(t, result.Outputs["stdout"], "/")
			},
		},
		{
			name:   "date",
			script: "#!/bin/bash\ndate",
			check: func(t *testing.T, result *node.NodeResult) {
				assert.NotEmpty(t, result.Outputs["stdout"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputs := map[string]interface{}{
				"script_content": tt.script,
				"interpreter":    "bash",
			}

			result, err := nodeInstance.Execute(context.Background(), inputs)

			require.NoError(t, err)
			assert.Equal(t, 0, result.Outputs["exit_code"])
			tt.check(t, result)
		})
	}
}
