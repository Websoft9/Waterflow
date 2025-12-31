//go:build integration
// +build integration

package main

import (
	"context"
	"os"
	"path/filepath"
	"plugin"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDockerComposeNode_Execute_Up_Integration tests actual up operation
func TestDockerComposeNode_Execute_Up_Integration(t *testing.T) {
	// Skip if Docker Compose not available
	if _, err := checkDockerComposeAvailable(); err != nil {
		t.Skip("Docker Compose not available, skipping integration test")
	}

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "compose-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test docker-compose.yml
	composeContent := `version: '3.8'

services:
  test:
    image: busybox:latest
    command: sleep 3600
`
	composeFile := filepath.Join(tmpDir, "docker-compose.yml")
	err = os.WriteFile(composeFile, []byte(composeContent), 0644)
	require.NoError(t, err)

	node := &DockerComposeNode{}
	ctx := context.Background()

	// Test up operation
	inputs := map[string]interface{}{
		"action":       "up",
		"file":         "docker-compose.yml",
		"workdir":      tmpDir,
		"project_name": "test-compose",
		"detach":       true,
	}

	result, err := node.Execute(ctx, inputs)
	require.NoError(t, err)
	require.NotNil(t, result)

	outputs := result.Outputs
	assert.Equal(t, "up", outputs["action"])
	assert.Equal(t, 0, outputs["exit_code"])
	assert.NotEmpty(t, outputs["containers"])

	// Cleanup: down operation
	downInputs := map[string]interface{}{
		"action":       "down",
		"file":         "docker-compose.yml",
		"workdir":      tmpDir,
		"project_name": "test-compose",
		"volumes":      true,
	}

	downResult, err := node.Execute(ctx, downInputs)
	require.NoError(t, err)
	require.NotNil(t, downResult)

	downOutputs := downResult.Outputs
	assert.Equal(t, "down", downOutputs["action"])
	assert.Equal(t, 0, downOutputs["exit_code"])
}

// TestDockerComposeNode_Execute_Down_Integration tests actual down operation
func TestDockerComposeNode_Execute_Down_Integration(t *testing.T) {
	// Skip if Docker Compose not available
	if _, err := checkDockerComposeAvailable(); err != nil {
		t.Skip("Docker Compose not available, skipping integration test")
	}

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "compose-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test docker-compose.yml
	composeContent := `version: '3.8'

services:
  web:
    image: nginx:alpine
    ports:
      - "8888:80"
`
	composeFile := filepath.Join(tmpDir, "docker-compose.yml")
	err = os.WriteFile(composeFile, []byte(composeContent), 0644)
	require.NoError(t, err)

	node := &DockerComposeNode{}
	ctx := context.Background()

	// First up
	upInputs := map[string]interface{}{
		"action":       "up",
		"file":         "docker-compose.yml",
		"workdir":      tmpDir,
		"project_name": "test-down",
		"detach":       true,
	}

	upResult, err := node.Execute(ctx, upInputs)
	require.NoError(t, err)
	require.NotNil(t, upResult)

	// Test down with volumes
	downInputs := map[string]interface{}{
		"action":       "down",
		"file":         "docker-compose.yml",
		"workdir":      tmpDir,
		"project_name": "test-down",
		"volumes":      true,
		"rmi":          "local",
	}

	result, err := node.Execute(ctx, downInputs)
	require.NoError(t, err)
	require.NotNil(t, result)

	outputs := result.Outputs
	assert.Equal(t, "down", outputs["action"])
	assert.Equal(t, 0, outputs["exit_code"])
}

// TestPluginLoading tests plugin loading
func TestPluginLoading(t *testing.T) {
	// This test requires building the plugin first
	// Run: make build
	pluginPath := "./compose.so"

	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		t.Skip("Plugin not built, run 'make build' first")
	}

	// Load plugin
	p, err := plugin.Open(pluginPath)
	require.NoError(t, err)

	// Lookup Register symbol
	symbol, err := p.Lookup("Register")
	require.NoError(t, err)

	// Call Register function
	registerFunc, ok := symbol.(func() interface{})
	if !ok {
		t.Fatal("Register function has wrong signature")
	}

	node := registerFunc()
	require.NotNil(t, node)

	// Verify node implements basic interface
	t.Log("Plugin loaded successfully:", pluginPath)
}
