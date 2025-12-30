package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShellNode_Name(t *testing.T) {
	node := &ShellNode{}
	assert.Equal(t, "exec/shell", node.Name())
}

func TestShellNode_Version(t *testing.T) {
	node := &ShellNode{}
	assert.Equal(t, "v1", node.Version())
}

func TestShellNode_Metadata(t *testing.T) {
	node := &ShellNode{}
	metadata := node.Metadata()

	assert.Equal(t, "Execute shell commands on the agent server", metadata.Description)
	assert.Equal(t, "exec", metadata.Category)

	// Validate InputSchema
	require.Contains(t, metadata.InputSchema, "command")
	assert.True(t, metadata.InputSchema["command"].Required)
	assert.Equal(t, "string", metadata.InputSchema["command"].Type)

	require.Contains(t, metadata.InputSchema, "timeout")
	assert.False(t, metadata.InputSchema["timeout"].Required)
	assert.Equal(t, 60, metadata.InputSchema["timeout"].Default)
	require.NotNil(t, metadata.InputSchema["timeout"].MinValue)
	assert.Equal(t, float64(1), *metadata.InputSchema["timeout"].MinValue)
	require.NotNil(t, metadata.InputSchema["timeout"].MaxValue)
	assert.Equal(t, float64(3600), *metadata.InputSchema["timeout"].MaxValue)

	// Validate OutputSchema
	require.Contains(t, metadata.OutputSchema, "stdout")
	require.Contains(t, metadata.OutputSchema, "stderr")
	require.Contains(t, metadata.OutputSchema, "exit_code")
}

func TestShellNode_Params(t *testing.T) {
	node := &ShellNode{}
	params := node.Params()

	// Params should return the same as Metadata().InputSchema
	assert.Equal(t, node.Metadata().InputSchema, params)
}

func TestShellNode_Execute_BasicCommand(t *testing.T) {
	node := &ShellNode{}

	inputs := map[string]interface{}{
		"command": "echo hello",
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "hello\n", result.Outputs["stdout"])
	assert.Equal(t, "", result.Outputs["stderr"])
	assert.Equal(t, 0, result.Outputs["exit_code"])
	assert.Greater(t, result.Duration, time.Duration(0))
	assert.NotEmpty(t, result.Logs)
	assert.Contains(t, result.Logs[0], "Executing: echo hello")
}

func TestShellNode_Execute_WithArgs(t *testing.T) {
	node := &ShellNode{}

	inputs := map[string]interface{}{
		"command": "echo",
		"args":    []interface{}{"hello", "world"},
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Contains(t, result.Outputs["stdout"], "hello world")
	assert.Equal(t, 0, result.Outputs["exit_code"])
}

func TestShellNode_Execute_WithEnv(t *testing.T) {
	node := &ShellNode{}

	inputs := map[string]interface{}{
		"command": "echo $TEST_VAR",
		"env": map[string]interface{}{
			"TEST_VAR": "test_value",
		},
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Contains(t, result.Outputs["stdout"], "test_value")
	assert.Equal(t, 0, result.Outputs["exit_code"])
}

func TestShellNode_Execute_WithWorkdir(t *testing.T) {
	node := &ShellNode{}

	inputs := map[string]interface{}{
		"command": "pwd",
		"workdir": "/tmp",
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Contains(t, result.Outputs["stdout"], "/tmp")
	assert.Equal(t, 0, result.Outputs["exit_code"])
}

func TestShellNode_Execute_Timeout(t *testing.T) {
	node := &ShellNode{}

	inputs := map[string]interface{}{
		"command": "sleep 10",
		"timeout": 1, // 1 second timeout
	}

	start := time.Now()
	result, err := node.Execute(context.Background(), inputs)
	duration := time.Since(start)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	assert.Less(t, duration, 2*time.Second)
	assert.NotNil(t, result) // Even on error, result should be returned
	assert.Equal(t, -1, result.Outputs["exit_code"])
}

func TestShellNode_Execute_CommandFailed(t *testing.T) {
	node := &ShellNode{}

	inputs := map[string]interface{}{
		"command": "false", // Always returns exit code 1
	}

	result, err := node.Execute(context.Background(), inputs)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "exit code 1")
	assert.Equal(t, 1, result.Outputs["exit_code"])
}

func TestShellNode_Execute_CommandNotFound(t *testing.T) {
	node := &ShellNode{}

	inputs := map[string]interface{}{
		"command": "nonexistent-command-12345",
	}

	result, err := node.Execute(context.Background(), inputs)

	require.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 127, result.Outputs["exit_code"]) // sh returns 127 for command not found
}

func TestShellNode_Execute_EmptyCommand(t *testing.T) {
	node := &ShellNode{}

	inputs := map[string]interface{}{
		"command": "",
	}

	result, err := node.Execute(context.Background(), inputs)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "command is required")
	assert.Nil(t, result)
}

func TestShellNode_Execute_MissingCommand(t *testing.T) {
	node := &ShellNode{}

	inputs := map[string]interface{}{}

	result, err := node.Execute(context.Background(), inputs)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "command is required")
	assert.Nil(t, result)
}

func TestShellNode_Execute_ConcurrentSafe(t *testing.T) {
	node := &ShellNode{}

	// Run 10 concurrent executions
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			inputs := map[string]interface{}{
				"command": "echo test",
			}
			result, err := node.Execute(context.Background(), inputs)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, 0, result.Outputs["exit_code"])
			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestShellNode_Execute_StderrCapture(t *testing.T) {
	node := &ShellNode{}

	inputs := map[string]interface{}{
		"command": "echo error >&2", // Write to stderr
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Equal(t, "", result.Outputs["stdout"])
	assert.Contains(t, result.Outputs["stderr"], "error")
	assert.Equal(t, 0, result.Outputs["exit_code"])
}

func TestShellNode_Execute_DurationTracking(t *testing.T) {
	node := &ShellNode{}

	inputs := map[string]interface{}{
		"command": "sleep 0.1",
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, result.Outputs["duration_ms"].(int64), int64(100))
	assert.Greater(t, result.Duration, 100*time.Millisecond)
}

func TestRegister(t *testing.T) {
	node := Register()
	require.NotNil(t, node)
	assert.Equal(t, "exec/shell", node.Name())
	assert.Equal(t, "v1", node.Version())
}

func TestShellNode_Execute_Float64Timeout(t *testing.T) {
	node := &ShellNode{}

	inputs := map[string]interface{}{
		"command": "echo test",
		"timeout": float64(10), // Test float64 timeout parsing
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Equal(t, 0, result.Outputs["exit_code"])
}

func TestShellNode_Execute_MultipleEnvVars(t *testing.T) {
	node := &ShellNode{}

	inputs := map[string]interface{}{
		"command": "echo $VAR1-$VAR2",
		"env": map[string]interface{}{
			"VAR1": "value1",
			"VAR2": "value2",
		},
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Contains(t, result.Outputs["stdout"], "value1-value2")
}

func TestShellNode_Execute_ContextCancellation(t *testing.T) {
	node := &ShellNode{}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	inputs := map[string]interface{}{
		"command": "sleep 10",
	}

	result, err := node.Execute(ctx, inputs)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
	assert.NotNil(t, result)
	assert.Equal(t, -1, result.Outputs["exit_code"])
}
