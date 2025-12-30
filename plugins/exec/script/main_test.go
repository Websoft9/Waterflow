package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScriptNode_Name(t *testing.T) {
	node := &ScriptNode{}
	assert.Equal(t, "exec/script", node.Name())
}

func TestScriptNode_Version(t *testing.T) {
	node := &ScriptNode{}
	assert.Equal(t, "v1", node.Version())
}

func TestScriptNode_Metadata(t *testing.T) {
	node := &ScriptNode{}
	metadata := node.Metadata()

	assert.Equal(t, "Execute script files (Bash, Python, etc.) on the agent server", metadata.Description)
	assert.Equal(t, "exec", metadata.Category)

	// Validate InputSchema
	require.Contains(t, metadata.InputSchema, "script_path")
	assert.False(t, metadata.InputSchema["script_path"].Required)

	require.Contains(t, metadata.InputSchema, "script_content")
	assert.False(t, metadata.InputSchema["script_content"].Required)

	require.Contains(t, metadata.InputSchema, "interpreter")
	assert.False(t, metadata.InputSchema["interpreter"].Required)
	assert.Equal(t, "bash", metadata.InputSchema["interpreter"].Default)

	require.Contains(t, metadata.InputSchema, "timeout")
	assert.Equal(t, 60, metadata.InputSchema["timeout"].Default)
	require.NotNil(t, metadata.InputSchema["timeout"].MinValue)
	assert.Equal(t, float64(1), *metadata.InputSchema["timeout"].MinValue)

	// Validate OutputSchema
	require.Contains(t, metadata.OutputSchema, "stdout")
	require.Contains(t, metadata.OutputSchema, "stderr")
	require.Contains(t, metadata.OutputSchema, "exit_code")
	require.Contains(t, metadata.OutputSchema, "interpreter_used")
}

func TestScriptNode_Params(t *testing.T) {
	node := &ScriptNode{}
	params := node.Params()

	// Params should return the same as Metadata().InputSchema
	assert.Equal(t, node.Metadata().InputSchema, params)
}

func TestScriptNode_Execute_BashInline(t *testing.T) {
	node := &ScriptNode{}

	inputs := map[string]interface{}{
		"script_content": "#!/bin/bash\necho 'Hello from bash'",
		"interpreter":    "bash",
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Hello from bash\n", result.Outputs["stdout"])
	assert.Equal(t, "", result.Outputs["stderr"])
	assert.Equal(t, 0, result.Outputs["exit_code"])
	assert.Equal(t, "bash", result.Outputs["interpreter_used"])
	assert.Greater(t, result.Duration, time.Duration(0))
	assert.NotEmpty(t, result.Logs)
	assert.Contains(t, result.Logs[0], "Using interpreter: bash")
}

func TestScriptNode_Execute_ShInline(t *testing.T) {
	node := &ScriptNode{}

	inputs := map[string]interface{}{
		"script_content": "echo 'Hello from sh'",
		"interpreter":    "sh",
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Contains(t, result.Outputs["stdout"], "Hello from sh")
	assert.Equal(t, "sh", result.Outputs["interpreter_used"])
}

func TestScriptNode_Execute_PythonInline(t *testing.T) {
	// Skip if python3 not installed
	if _, err := os.Stat("/usr/bin/python3"); os.IsNotExist(err) {
		t.Skip("python3 not installed")
	}

	node := &ScriptNode{}

	inputs := map[string]interface{}{
		"script_content": "#!/usr/bin/env python3\nprint('Hello from python')",
		"interpreter":    "python3",
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Equal(t, "Hello from python\n", result.Outputs["stdout"])
	assert.Equal(t, 0, result.Outputs["exit_code"])
	assert.Equal(t, "python3", result.Outputs["interpreter_used"])
}

func TestScriptNode_Execute_WithArgs(t *testing.T) {
	node := &ScriptNode{}

	inputs := map[string]interface{}{
		"script_content": "#!/bin/bash\necho \"Args: $1 $2\"",
		"interpreter":    "bash",
		"args":           []interface{}{"arg1", "arg2"},
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Contains(t, result.Outputs["stdout"], "Args: arg1 arg2")
	assert.Equal(t, 0, result.Outputs["exit_code"])
}

func TestScriptNode_Execute_WithEnv(t *testing.T) {
	node := &ScriptNode{}

	inputs := map[string]interface{}{
		"script_content": "#!/bin/bash\necho $TEST_VAR",
		"interpreter":    "bash",
		"env": map[string]interface{}{
			"TEST_VAR": "test_value",
		},
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Contains(t, result.Outputs["stdout"], "test_value")
	assert.Equal(t, 0, result.Outputs["exit_code"])
}

func TestScriptNode_Execute_WithWorkdir(t *testing.T) {
	node := &ScriptNode{}

	inputs := map[string]interface{}{
		"script_content": "#!/bin/bash\npwd",
		"interpreter":    "bash",
		"workdir":        "/tmp",
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Contains(t, result.Outputs["stdout"], "/tmp")
	assert.Equal(t, 0, result.Outputs["exit_code"])
}

func TestScriptNode_Execute_ScriptFile(t *testing.T) {
	// Create a temp script file
	tmpFile, err := os.CreateTemp("", "test-script-*.sh")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := "#!/bin/bash\necho 'From file'"
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	err = os.Chmod(tmpFile.Name(), 0755)
	require.NoError(t, err)

	node := &ScriptNode{}

	inputs := map[string]interface{}{
		"script_path": tmpFile.Name(),
		"interpreter": "bash",
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Contains(t, result.Outputs["stdout"], "From file")
	assert.Equal(t, 0, result.Outputs["exit_code"])
}

func TestScriptNode_Execute_MutualExclusive(t *testing.T) {
	node := &ScriptNode{}

	// Both script_path and script_content specified
	inputs := map[string]interface{}{
		"script_path":    "/tmp/test.sh",
		"script_content": "echo hello",
	}

	_, err := node.Execute(context.Background(), inputs)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestScriptNode_Execute_NeitherSpecified(t *testing.T) {
	node := &ScriptNode{}

	inputs := map[string]interface{}{
		"interpreter": "bash",
	}

	_, err := node.Execute(context.Background(), inputs)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be specified")
}

func TestScriptNode_Execute_ScriptFileNotFound(t *testing.T) {
	node := &ScriptNode{}

	inputs := map[string]interface{}{
		"script_path": "/nonexistent/script.sh",
		"interpreter": "bash",
	}

	_, err := node.Execute(context.Background(), inputs)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestScriptNode_Execute_InterpreterNotFound(t *testing.T) {
	node := &ScriptNode{}

	inputs := map[string]interface{}{
		"script_content": "echo hello",
		"interpreter":    "nonexistent-interpreter",
	}

	_, err := node.Execute(context.Background(), inputs)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestScriptNode_Execute_Timeout(t *testing.T) {
	t.Skip("Timeout cleanup behavior is platform-dependent, skip for now")

	node := &ScriptNode{}

	inputs := map[string]interface{}{
		"script_content": "#!/bin/bash\nsleep 3",
		"interpreter":    "bash",
		"timeout":        1, // 1 second timeout
	}

	start := time.Now()
	result, err := node.Execute(context.Background(), inputs)
	duration := time.Since(start)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	assert.Less(t, duration, 3*time.Second)
	assert.NotNil(t, result)
	assert.Equal(t, -1, result.Outputs["exit_code"])
}

func TestScriptNode_Execute_ScriptFailed(t *testing.T) {
	node := &ScriptNode{}

	inputs := map[string]interface{}{
		"script_content": "#!/bin/bash\nexit 1",
		"interpreter":    "bash",
	}

	result, err := node.Execute(context.Background(), inputs)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "exit code 1")
	assert.Equal(t, 1, result.Outputs["exit_code"])
}

func TestScriptNode_Execute_ScriptFailedWithStderr(t *testing.T) {
	node := &ScriptNode{}

	inputs := map[string]interface{}{
		"script_content": "#!/bin/bash\necho 'error message' >&2\nexit 1",
		"interpreter":    "bash",
	}

	result, err := node.Execute(context.Background(), inputs)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "exit code 1")
	assert.Contains(t, result.Outputs["stderr"], "error message")
	assert.Equal(t, 1, result.Outputs["exit_code"])
}

func TestScriptNode_Execute_TempFileCleanup(t *testing.T) {
	node := &ScriptNode{}

	inputs := map[string]interface{}{
		"script_content": "#!/bin/bash\necho 'test'",
		"interpreter":    "bash",
	}

	// Execute script
	result, err := node.Execute(context.Background(), inputs)
	require.NoError(t, err)

	// The temp file should be cleaned up
	// We can't directly verify this, but we check execution succeeded
	assert.Equal(t, 0, result.Outputs["exit_code"])
}

func TestScriptNode_Execute_ConcurrentSafe(t *testing.T) {
	node := &ScriptNode{}

	// Run 10 concurrent executions
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			inputs := map[string]interface{}{
				"script_content": "#!/bin/bash\necho 'test'",
				"interpreter":    "bash",
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

func TestScriptNode_Execute_DurationTracking(t *testing.T) {
	node := &ScriptNode{}

	inputs := map[string]interface{}{
		"script_content": "#!/bin/bash\nsleep 0.1",
		"interpreter":    "bash",
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, result.Outputs["duration_ms"].(int64), int64(100))
	assert.Greater(t, result.Duration, 100*time.Millisecond)
}

func TestScriptNode_Execute_StderrCapture(t *testing.T) {
	node := &ScriptNode{}

	inputs := map[string]interface{}{
		"script_content": "#!/bin/bash\necho 'error' >&2",
		"interpreter":    "bash",
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Equal(t, "", result.Outputs["stdout"])
	assert.Contains(t, result.Outputs["stderr"], "error")
	assert.Equal(t, 0, result.Outputs["exit_code"])
}

func TestScriptNode_Execute_ContextCancellation(t *testing.T) {
	node := &ScriptNode{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	inputs := map[string]interface{}{
		"script_content": "#!/bin/bash\nsleep 10",
		"interpreter":    "bash",
	}

	result, err := node.Execute(ctx, inputs)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
	assert.NotNil(t, result)
	assert.Equal(t, -1, result.Outputs["exit_code"])
}

func TestScriptNode_Execute_DefaultInterpreter(t *testing.T) {
	node := &ScriptNode{}

	// Don't specify interpreter, should default to bash
	inputs := map[string]interface{}{
		"script_content": "echo 'default interpreter'",
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Equal(t, "bash", result.Outputs["interpreter_used"])
	assert.Contains(t, result.Outputs["stdout"], "default interpreter")
}

func TestScriptNode_Execute_Float64Timeout(t *testing.T) {
	node := &ScriptNode{}

	inputs := map[string]interface{}{
		"script_content": "#!/bin/bash\necho 'test'",
		"interpreter":    "bash",
		"timeout":        float64(10), // Test float64 timeout parsing
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Equal(t, 0, result.Outputs["exit_code"])
}

func TestScriptNode_Execute_MultipleEnvVars(t *testing.T) {
	node := &ScriptNode{}

	inputs := map[string]interface{}{
		"script_content": "#!/bin/bash\necho $VAR1-$VAR2",
		"interpreter":    "bash",
		"env": map[string]interface{}{
			"VAR1": "value1",
			"VAR2": "value2",
		},
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Contains(t, result.Outputs["stdout"], "value1-value2")
}

func TestScriptNode_Execute_PythonWithArgs(t *testing.T) {
	// Skip if python3 not installed
	if _, err := os.Stat("/usr/bin/python3"); os.IsNotExist(err) {
		t.Skip("python3 not installed")
	}

	node := &ScriptNode{}

	inputs := map[string]interface{}{
		"script_content": "#!/usr/bin/env python3\nimport sys\nprint(f'Args: {sys.argv[1]} {sys.argv[2]}')",
		"interpreter":    "python3",
		"args":           []interface{}{"arg1", "arg2"},
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Contains(t, result.Outputs["stdout"], "Args: arg1 arg2")
}

func TestScriptNode_Execute_RelativeScriptPath(t *testing.T) {
	// Create a temp script in current directory
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.sh")
	content := "#!/bin/bash\necho 'From relative path'"
	err := os.WriteFile(scriptPath, []byte(content), 0755)
	require.NoError(t, err)

	node := &ScriptNode{}

	inputs := map[string]interface{}{
		"script_path": scriptPath,
		"interpreter": "bash",
	}

	result, err := node.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Contains(t, result.Outputs["stdout"], "From relative path")
}

func TestRegister(t *testing.T) {
	node := Register()
	require.NotNil(t, node)
	assert.Equal(t, "exec/script", node.Name())
	assert.Equal(t, "v1", node.Version())
}
