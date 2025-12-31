//go:build integration
// +build integration

package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_DockerVersion tests docker version command
func TestIntegration_DockerVersion(t *testing.T) {
	n := &DockerExecNode{}
	ctx := context.Background()

	result, err := n.Execute(ctx, map[string]interface{}{
		"command": "version",
		"timeout": "30s",
	})

	require.NoError(t, err, "docker version should succeed")
	require.NotNil(t, result)

	assert.Equal(t, 0, result.Outputs["exit_code"])
	stdout := result.Outputs["stdout"].(string)
	assert.Contains(t, stdout, "Version:")
	assert.NotEmpty(t, result.Outputs["command_line"])
	assert.Greater(t, result.Outputs["elapsed_ms"].(int64), int64(0))
}

// TestIntegration_DockerPS tests docker ps command
func TestIntegration_DockerPS(t *testing.T) {
	n := &DockerExecNode{}
	ctx := context.Background()

	result, err := n.Execute(ctx, map[string]interface{}{
		"command": "ps",
		"args":    []interface{}{"-a"},
	})

	require.NoError(t, err)
	assert.Equal(t, 0, result.Outputs["exit_code"])
	assert.Contains(t, result.Outputs["command_line"].(string), "docker ps -a")
}

// TestIntegration_DockerImages tests docker images command
func TestIntegration_DockerImages(t *testing.T) {
	n := &DockerExecNode{}
	ctx := context.Background()

	result, err := n.Execute(ctx, map[string]interface{}{
		"command": "images",
	})

	require.NoError(t, err)
	assert.Equal(t, 0, result.Outputs["exit_code"])
}

// TestIntegration_DockerInfo tests docker info command
func TestIntegration_DockerInfo(t *testing.T) {
	n := &DockerExecNode{}
	ctx := context.Background()

	result, err := n.Execute(ctx, map[string]interface{}{
		"command": "info",
		"timeout": "30s",
	})

	require.NoError(t, err)
	assert.Equal(t, 0, result.Outputs["exit_code"])
	stdout := result.Outputs["stdout"].(string)
	assert.Contains(t, stdout, "Server")
}

// TestIntegration_ContainerLifecycle tests full container lifecycle
func TestIntegration_ContainerLifecycle(t *testing.T) {
	n := &DockerExecNode{}
	ctx := context.Background()
	containerName := "waterflow-test-" + time.Now().Format("20060102-150405")

	// Cleanup function
	cleanup := func() {
		// Try to remove container if it exists
		n.Execute(ctx, map[string]interface{}{
			"command": "rm",
			"args":    []interface{}{"-f", containerName},
		})
	}
	defer cleanup()

	// Step 1: Pull alpine image (small and fast)
	t.Run("pull_image", func(t *testing.T) {
		result, err := n.Execute(ctx, map[string]interface{}{
			"command": "pull",
			"args":    []interface{}{"alpine:latest"},
			"timeout": "5m",
		})

		require.NoError(t, err)
		assert.Equal(t, 0, result.Outputs["exit_code"])
	})

	// Step 2: Run container
	t.Run("run_container", func(t *testing.T) {
		result, err := n.Execute(ctx, map[string]interface{}{
			"command": "run",
			"args": []interface{}{
				"-d",
				"--name", containerName,
				"alpine:latest",
				"sleep", "30",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, result.Outputs["exit_code"])
		stdout := result.Outputs["stdout"].(string)
		assert.NotEmpty(t, stdout) // Container ID
	})

	// Step 3: Verify container is running
	t.Run("verify_running", func(t *testing.T) {
		result, err := n.Execute(ctx, map[string]interface{}{
			"command": "ps",
			"args":    []interface{}{"--filter", "name=" + containerName},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, result.Outputs["exit_code"])
		stdout := result.Outputs["stdout"].(string)
		assert.Contains(t, stdout, containerName)
	})

	// Step 4: Execute command in container
	t.Run("exec_in_container", func(t *testing.T) {
		result, err := n.Execute(ctx, map[string]interface{}{
			"command": "exec",
			"args":    []interface{}{containerName, "echo", "Hello from container"},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, result.Outputs["exit_code"])
		stdout := result.Outputs["stdout"].(string)
		assert.Contains(t, stdout, "Hello from container")
	})

	// Step 5: View logs
	t.Run("view_logs", func(t *testing.T) {
		result, err := n.Execute(ctx, map[string]interface{}{
			"command": "logs",
			"args":    []interface{}{containerName},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, result.Outputs["exit_code"])
	})

	// Step 6: Stop container
	t.Run("stop_container", func(t *testing.T) {
		result, err := n.Execute(ctx, map[string]interface{}{
			"command": "stop",
			"args":    []interface{}{containerName},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, result.Outputs["exit_code"])
	})

	// Step 7: Remove container
	t.Run("remove_container", func(t *testing.T) {
		result, err := n.Execute(ctx, map[string]interface{}{
			"command": "rm",
			"args":    []interface{}{containerName},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, result.Outputs["exit_code"])
	})
}

// TestIntegration_ErrorHandling tests error classification
func TestIntegration_ErrorHandling(t *testing.T) {
	n := &DockerExecNode{}
	ctx := context.Background()

	// Test 1: Non-existent container (permanent error)
	t.Run("container_not_found", func(t *testing.T) {
		result, err := n.Execute(ctx, map[string]interface{}{
			"command": "stop",
			"args":    []interface{}{"non-existent-container-xyz-12345"},
		})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "ContainerNotFound")
	})

	// Test 2: Invalid command
	t.Run("invalid_command", func(t *testing.T) {
		result, err := n.Execute(ctx, map[string]interface{}{
			"command": "invalid-docker-command",
		})

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

// TestIntegration_NetworkOperations tests network commands
func TestIntegration_NetworkOperations(t *testing.T) {
	n := &DockerExecNode{}
	ctx := context.Background()
	networkName := "waterflow-test-net-" + time.Now().Format("20060102-150405")

	// Cleanup
	defer func() {
		n.Execute(ctx, map[string]interface{}{
			"command": "network",
			"args":    []interface{}{"rm", networkName},
		})
	}()

	// Create network
	t.Run("create_network", func(t *testing.T) {
		result, err := n.Execute(ctx, map[string]interface{}{
			"command": "network",
			"args":    []interface{}{"create", networkName},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, result.Outputs["exit_code"])
	})

	// List networks
	t.Run("list_networks", func(t *testing.T) {
		result, err := n.Execute(ctx, map[string]interface{}{
			"command": "network",
			"args":    []interface{}{"ls"},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, result.Outputs["exit_code"])
		stdout := result.Outputs["stdout"].(string)
		assert.Contains(t, stdout, networkName)
	})

	// Inspect network
	t.Run("inspect_network", func(t *testing.T) {
		result, err := n.Execute(ctx, map[string]interface{}{
			"command": "network",
			"args":    []interface{}{"inspect", networkName},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, result.Outputs["exit_code"])
	})

	// Remove network
	t.Run("remove_network", func(t *testing.T) {
		result, err := n.Execute(ctx, map[string]interface{}{
			"command": "network",
			"args":    []interface{}{"rm", networkName},
		})

		require.NoError(t, err)
		assert.Equal(t, 0, result.Outputs["exit_code"])
	})
}

// TestIntegration_Timeout tests timeout handling
func TestIntegration_Timeout(t *testing.T) {
	n := &DockerExecNode{}
	ctx := context.Background()

	// This should complete quickly
	result, err := n.Execute(ctx, map[string]interface{}{
		"command": "version",
		"timeout": "10s",
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify elapsed time is less than timeout
	elapsed := result.Outputs["elapsed_ms"].(int64)
	assert.Less(t, elapsed, int64(10000), "Should complete in less than 10 seconds")
}

// TestIntegration_StdoutStderr tests output capturing
func TestIntegration_StdoutStderr(t *testing.T) {
	n := &DockerExecNode{}
	ctx := context.Background()

	// docker version outputs to stdout
	result, err := n.Execute(ctx, map[string]interface{}{
		"command": "version",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.Outputs["stdout"])

	// stderr should be empty for successful command
	stderr := result.Outputs["stderr"].(string)
	// Note: Some Docker versions may output warnings to stderr even on success
	// So we just check it's a string, not necessarily empty
	assert.IsType(t, "", stderr)
}
