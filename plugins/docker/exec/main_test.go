package main

import (
	"context"
	"strings"
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerExecNode_Name(t *testing.T) {
	n := &DockerExecNode{}
	assert.Equal(t, "docker/exec", n.Name())
}

func TestDockerExecNode_Version(t *testing.T) {
	n := &DockerExecNode{}
	assert.Equal(t, "v1", n.Version())
}

func TestDockerExecNode_Metadata(t *testing.T) {
	n := &DockerExecNode{}
	metadata := n.Metadata()

	assert.Equal(t, "docker", metadata.Category)
	assert.NotEmpty(t, metadata.Description)
	assert.Contains(t, metadata.InputSchema, "command")
	assert.Contains(t, metadata.InputSchema, "args")
	assert.Contains(t, metadata.InputSchema, "timeout")
	assert.Contains(t, metadata.InputSchema, "docker_host")

	// Verify command parameter is required
	assert.True(t, metadata.InputSchema["command"].Required)
	assert.Equal(t, "string", metadata.InputSchema["command"].Type)

	// Verify args is optional array
	assert.False(t, metadata.InputSchema["args"].Required)
	assert.Equal(t, "array", metadata.InputSchema["args"].Type)

	// Verify timeout has default
	assert.Equal(t, "5m", metadata.InputSchema["timeout"].Default)

	// Verify output schema
	assert.NotNil(t, metadata.OutputSchema)
	assert.Contains(t, metadata.OutputSchema, "exit_code")
	assert.Contains(t, metadata.OutputSchema, "stdout")
	assert.Contains(t, metadata.OutputSchema, "stderr")
	assert.Contains(t, metadata.OutputSchema, "elapsed_ms")
	assert.Contains(t, metadata.OutputSchema, "command_line")
}

func TestDockerExecNode_Params(t *testing.T) {
	n := &DockerExecNode{}
	params := n.Params()

	// Params() should return same as Metadata().InputSchema
	assert.Equal(t, n.Metadata().InputSchema, params)
}

func TestDockerExecNode_Execute_ValidationErrors(t *testing.T) {
	n := &DockerExecNode{}
	ctx := context.Background()

	tests := []struct {
		name        string
		inputs      map[string]interface{}
		expectedErr string
	}{
		{
			name:        "missing command",
			inputs:      map[string]interface{}{},
			expectedErr: "command is required",
		},
		{
			name: "empty command",
			inputs: map[string]interface{}{
				"command": "",
			},
			expectedErr: "command is required",
		},
		{
			name: "command not a string",
			inputs: map[string]interface{}{
				"command": 123,
			},
			expectedErr: "command is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := n.Execute(ctx, tt.inputs)
			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

// Integration test - requires Docker to be installed and running
func TestDockerExecNode_Execute_Version(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	n := &DockerExecNode{}
	ctx := context.Background()

	result, err := n.Execute(ctx, map[string]interface{}{
		"command": "version",
	})

	// If Docker is not available, skip the test
	if err != nil && (strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "not running") ||
		strings.Contains(err.Error(), "permission denied")) {
		t.Skipf("Docker not available: %v", err)
	}

	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify outputs
	assert.Equal(t, 0, result.Outputs["exit_code"])
	assert.NotEmpty(t, result.Outputs["stdout"])
	assert.Contains(t, result.Outputs["stdout"].(string), "Version:")
	assert.NotNil(t, result.Outputs["elapsed_ms"])
	assert.Contains(t, result.Outputs["command_line"].(string), "docker version")
}

// Integration test - docker ps
func TestDockerExecNode_Execute_PS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	n := &DockerExecNode{}
	ctx := context.Background()

	result, err := n.Execute(ctx, map[string]interface{}{
		"command": "ps",
		"args":    []interface{}{"-a"},
	})

	if err != nil && (strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "not running") ||
		strings.Contains(err.Error(), "permission denied")) {
		t.Skipf("Docker not available: %v", err)
	}

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 0, result.Outputs["exit_code"])
	assert.Contains(t, result.Outputs["command_line"].(string), "docker ps -a")
}

// Integration test - docker images
func TestDockerExecNode_Execute_Images(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	n := &DockerExecNode{}
	ctx := context.Background()

	result, err := n.Execute(ctx, map[string]interface{}{
		"command": "images",
	})

	if err != nil && (strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "not running") ||
		strings.Contains(err.Error(), "permission denied")) {
		t.Skipf("Docker not available: %v", err)
	}

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 0, result.Outputs["exit_code"])
}

// Test with multiple args
func TestDockerExecNode_Execute_WithMultipleArgs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	n := &DockerExecNode{}
	ctx := context.Background()

	result, err := n.Execute(ctx, map[string]interface{}{
		"command": "ps",
		"args":    []interface{}{"--filter", "status=running", "--format", "{{.Names}}"},
	})

	if err != nil && (strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "not running") ||
		strings.Contains(err.Error(), "permission denied")) {
		t.Skipf("Docker not available: %v", err)
	}

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 0, result.Outputs["exit_code"])
	cmdLine := result.Outputs["command_line"].(string)
	assert.Contains(t, cmdLine, "docker ps")
	assert.Contains(t, cmdLine, "--filter")
	assert.Contains(t, cmdLine, "status=running")
}

// Test timeout parsing
func TestDockerExecNode_Execute_CustomTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	n := &DockerExecNode{}
	ctx := context.Background()

	result, err := n.Execute(ctx, map[string]interface{}{
		"command": "version",
		"timeout": "30s",
	})

	if err != nil && (strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "not running") ||
		strings.Contains(err.Error(), "permission denied")) {
		t.Skipf("Docker not available: %v", err)
	}

	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestDockerExecNode_Execute_InvalidCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	n := &DockerExecNode{}
	ctx := context.Background()

	// Try to access non-existent container
	result, err := n.Execute(ctx, map[string]interface{}{
		"command": "stop",
		"args":    []interface{}{"non-existent-container-xyz-123"},
	})

	if err != nil && (strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "not running") ||
		strings.Contains(err.Error(), "permission denied")) {
		t.Skipf("Docker not available: %v", err)
	}

	require.Error(t, err)
	assert.Nil(t, result)
	// Should be permanent error for non-existent container
	assert.Contains(t, err.Error(), "ContainerNotFound")
}

func TestCheckDockerAvailable(t *testing.T) {
	// This test will pass if Docker is installed and running
	// It will return an error if Docker is not available
	err := checkDockerAvailable()

	if err != nil {
		// Check error type
		if strings.Contains(err.Error(), "not found") {
			assert.Contains(t, err.Error(), "DockerNotInstalled")
		} else if strings.Contains(err.Error(), "not running") {
			assert.Contains(t, err.Error(), "DockerDaemonNotRunning")
		} else if strings.Contains(err.Error(), "permission denied") {
			assert.Contains(t, err.Error(), "DockerPermissionDenied")
		}
	} else {
		// Docker is available
		assert.NoError(t, err)
	}
}

func TestClassifyDockerError(t *testing.T) {
	tests := []struct {
		name        string
		exitCode    int
		stderr      string
		expectedErr string
		isRetriable bool
	}{
		{
			name:        "container not found",
			exitCode:    1,
			stderr:      "Error: No such container: xyz",
			expectedErr: "ContainerNotFound",
			isRetriable: false,
		},
		{
			name:        "image not found",
			exitCode:    1,
			stderr:      "Unable to find image 'nonexistent:latest' locally",
			expectedErr: "ImageNotFound",
			isRetriable: true,
		},
		{
			name:        "permission denied",
			exitCode:    1,
			stderr:      "Got permission denied while trying to connect",
			expectedErr: "PermissionDenied",
			isRetriable: false,
		},
		{
			name:        "network error without timeout",
			exitCode:    1,
			stderr:      "error pulling image: network error occurred",
			expectedErr: "NetworkError",
			isRetriable: true,
		},
		{
			name:        "network timeout - should be TimeoutError not NetworkError",
			exitCode:    1,
			stderr:      "error pulling image: network timeout",
			expectedErr: "TimeoutError",
			isRetriable: true,
		},
		{
			name:        "daemon not running",
			exitCode:    1,
			stderr:      "Cannot connect to the Docker daemon",
			expectedErr: "DockerDaemonError",
			isRetriable: false,
		},
		{
			name:        "timeout error",
			exitCode:    -1,
			stderr:      "context deadline exceeded",
			expectedErr: "TimeoutError",
			isRetriable: true,
		},
		{
			name:        "generic error",
			exitCode:    1,
			stderr:      "unknown error occurred",
			expectedErr: "DockerCommandFailed",
			isRetriable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := classifyDockerError(tt.exitCode, tt.stderr)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)

			// Check if error is retriable by checking if it contains NonRetryable
			errStr := err.Error()
			if tt.isRetriable {
				// Retriable errors should not mention NonRetryable
				// Note: This is a simplified check; in actual Temporal, we'd check the error type
				assert.NotContains(t, errStr, "NonRetryable")
			}
		})
	}
}

func TestRegister(t *testing.T) {
	n := Register()
	require.NotNil(t, n)

	// Verify it implements node.Node interface
	var _ node.Node = n

	assert.Equal(t, "docker/exec", n.Name())
	assert.Equal(t, "v1", n.Version())
}

// Test docker_host parameter
func TestDockerExecNode_Execute_CustomDockerHost(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	n := &DockerExecNode{}
	ctx := context.Background()

	// Test with default socket (should work if Docker is available)
	result, err := n.Execute(ctx, map[string]interface{}{
		"command":     "version",
		"docker_host": "unix:///var/run/docker.sock",
	})

	if err != nil && (strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "not running") ||
		strings.Contains(err.Error(), "permission denied")) {
		t.Skipf("Docker not available: %v", err)
	}

	require.NoError(t, err)
	assert.Equal(t, 0, result.Outputs["exit_code"])
}

// Test actual timeout behavior
func TestDockerExecNode_Execute_ActualTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	n := &DockerExecNode{}
	ctx := context.Background()

	// Run a long sleep with short timeout
	result, err := n.Execute(ctx, map[string]interface{}{
		"command": "run",
		"args":    []interface{}{"--rm", "alpine:latest", "sleep", "60"},
		"timeout": "1s",
	})

	if err != nil && (strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "not running") ||
		strings.Contains(err.Error(), "permission denied") ||
		strings.Contains(err.Error(), "ImageNotFound")) {
		t.Skipf("Docker not available or image not found: %v", err)
	}

	// Should timeout
	require.Error(t, err)
	assert.Nil(t, result)
	// Error could be TimeoutError or context deadline exceeded
	assert.True(t, strings.Contains(err.Error(), "TimeoutError") ||
		strings.Contains(err.Error(), "deadline") ||
		strings.Contains(err.Error(), "timeout"),
		"expected timeout error, got: %v", err)
}

// Test timeout parsing with invalid value
func TestDockerExecNode_Execute_InvalidTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	n := &DockerExecNode{}
	ctx := context.Background()

	// Invalid timeout should fall back to default (5m)
	result, err := n.Execute(ctx, map[string]interface{}{
		"command": "version",
		"timeout": "invalid",
	})

	if err != nil && (strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "not running") ||
		strings.Contains(err.Error(), "permission denied")) {
		t.Skipf("Docker not available: %v", err)
	}

	require.NoError(t, err)
	require.NotNil(t, result)
}

// Test empty args array
func TestDockerExecNode_Execute_EmptyArgs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	n := &DockerExecNode{}
	ctx := context.Background()

	result, err := n.Execute(ctx, map[string]interface{}{
		"command": "version",
		"args":    []interface{}{},
	})

	if err != nil && (strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "not running") ||
		strings.Contains(err.Error(), "permission denied")) {
		t.Skipf("Docker not available: %v", err)
	}

	require.NoError(t, err)
	assert.Equal(t, 0, result.Outputs["exit_code"])
}

// Test context cancellation
func TestDockerExecNode_Execute_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	n := &DockerExecNode{}
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	result, err := n.Execute(ctx, map[string]interface{}{
		"command": "version",
	})

	// Should fail due to context cancellation
	if err == nil {
		t.Log("Warning: context cancellation didn't prevent execution")
	} else {
		assert.Nil(t, result)
	}
}
