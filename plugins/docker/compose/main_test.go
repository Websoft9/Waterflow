package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerComposeNode_Metadata(t *testing.T) {
	node := &DockerComposeNode{}

	assert.Equal(t, "docker/compose", node.Name())
	assert.Equal(t, "v1", node.Version())

	metadata := node.Metadata()
	assert.Equal(t, "docker", metadata.Category)
	assert.Contains(t, metadata.Description, "Docker Compose")

	// Check required parameters
	assert.True(t, metadata.InputSchema["action"].Required)
	assert.Equal(t, "string", metadata.InputSchema["action"].Type)
	assert.Equal(t, []interface{}{"up", "down"}, metadata.InputSchema["action"].Enum)

	// Check optional parameters
	assert.False(t, metadata.InputSchema["file"].Required)
	assert.Equal(t, "docker-compose.yml", metadata.InputSchema["file"].Default)

	// Check output schema
	assert.Contains(t, metadata.OutputSchema, "action")
	assert.Contains(t, metadata.OutputSchema, "containers")
	assert.Contains(t, metadata.OutputSchema, "services")
}

func TestDockerComposeNode_Execute_MissingAction(t *testing.T) {
	node := &DockerComposeNode{}

	result, err := node.Execute(context.Background(), map[string]interface{}{})
	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "action must be")
}

func TestDockerComposeNode_Execute_InvalidAction(t *testing.T) {
	node := &DockerComposeNode{}

	inputs := map[string]interface{}{
		"action": "invalid",
	}

	result, err := node.Execute(context.Background(), inputs)
	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "action must be")
}

func TestDockerComposeNode_Params(t *testing.T) {
	node := &DockerComposeNode{}
	params := node.Params()

	assert.NotNil(t, params)
	assert.Contains(t, params, "action")
	assert.Contains(t, params, "file")
	assert.Contains(t, params, "project_name")
	assert.Contains(t, params, "detach")
	assert.Contains(t, params, "build")
	assert.Contains(t, params, "volumes")
	assert.Contains(t, params, "rmi")
}

func TestBuildComposeArgs(t *testing.T) {
	tests := []struct {
		name       string
		composeCmd []string
		inputs     map[string]interface{}
		action     string
		expected   []string
	}{
		{
			name:       "docker compose with default file",
			composeCmd: []string{"docker", "compose"},
			inputs:     map[string]interface{}{},
			action:     "up",
			expected:   []string{"compose", "-f", "docker-compose.yml", "up"},
		},
		{
			name:       "docker-compose with custom file",
			composeCmd: []string{"docker-compose"},
			inputs: map[string]interface{}{
				"file": "docker-compose.prod.yml",
			},
			action:   "down",
			expected: []string{"-f", "docker-compose.prod.yml", "down"},
		},
		{
			name:       "with project name",
			composeCmd: []string{"docker", "compose"},
			inputs: map[string]interface{}{
				"project_name": "myapp",
			},
			action:   "up",
			expected: []string{"compose", "-f", "docker-compose.yml", "-p", "myapp", "up"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildComposeArgs(tt.composeCmd, tt.inputs, tt.action)
			assert.Equal(t, tt.expected, args)
		})
	}
}

func TestClassifyComposeError(t *testing.T) {
	tests := []struct {
		name         string
		exitCode     int
		stderr       string
		errType      string
		nonRetryable bool
	}{
		{
			name:         "file not found",
			exitCode:     1,
			stderr:       "no such file or directory",
			errType:      "ComposeFileNotFound",
			nonRetryable: true,
		},
		{
			name:         "YAML parse error",
			exitCode:     1,
			stderr:       "YAML parse error",
			errType:      "ComposeConfigError",
			nonRetryable: true,
		},
		{
			name:         "port conflict",
			exitCode:     1,
			stderr:       "port 8080 is already allocated",
			errType:      "PortConflict",
			nonRetryable: true,
		},
		{
			name:         "timeout error",
			exitCode:     1,
			stderr:       "context deadline exceeded",
			errType:      "TimeoutError",
			nonRetryable: false,
		},
		{
			name:         "network error",
			exitCode:     1,
			stderr:       "network connection failed",
			errType:      "NetworkError",
			nonRetryable: false,
		},
		{
			name:         "image pull error",
			exitCode:     1,
			stderr:       "failed to pull image nginx:latest",
			errType:      "ImagePullError",
			nonRetryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := classifyComposeError(tt.exitCode, tt.stderr)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errType)
		})
	}
}

func TestValidateComposeFile(t *testing.T) {
	t.Run("file not found", func(t *testing.T) {
		err := validateComposeFile("nonexistent.yml", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "compose file not found")
	})

	t.Run("file not found with workdir", func(t *testing.T) {
		err := validateComposeFile("nonexistent.yml", "/tmp")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "compose file not found")
	})
}

func TestContains(t *testing.T) {
	slice := []string{"web", "db", "cache"}

	assert.True(t, contains(slice, "web"))
	assert.True(t, contains(slice, "db"))
	assert.False(t, contains(slice, "api"))
	assert.False(t, contains([]string{}, "web"))
}

func TestRegister(t *testing.T) {
	node := Register()
	require.NotNil(t, node)
	assert.Equal(t, "docker/compose", node.Name())
	assert.Equal(t, "v1", node.Version())
}

// TestValidateComposeFile_PathTraversal tests path traversal protection
func TestValidateComposeFile_PathTraversal(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		workdir string
		wantErr bool
		errType string
	}{
		{
			name:    "path traversal in file",
			file:    "../../../etc/passwd",
			workdir: "",
			wantErr: true,
			errType: "InvalidFilePath",
		},
		{
			name:    "path traversal in workdir",
			file:    "docker-compose.yml",
			workdir: "../../etc",
			wantErr: true,
			errType: "InvalidWorkdir",
		},
		{
			name:    "safe path",
			file:    "docker-compose.yml",
			workdir: "/tmp",
			wantErr: true, // File doesn't exist, but path is safe
			errType: "ComposeFileNotFound",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateComposeFile(tt.file, tt.workdir)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errType)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestExecute_InvalidTimeout tests invalid timeout handling
func TestExecute_InvalidTimeout(t *testing.T) {
	// Create a temporary valid compose file
	tmpDir, err := os.MkdirTemp("", "compose-test-*")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tmpDir) // Cleanup, ignore error
	}()

	composeContent := "version: '3.8'\nservices:\n  test:\n    image: busybox\n"
	composeFile := filepath.Join(tmpDir, "docker-compose.yml")
	err = os.WriteFile(composeFile, []byte(composeContent), 0600)
	require.NoError(t, err)

	node := &DockerComposeNode{}

	inputs := map[string]interface{}{
		"action":  "up",
		"file":    "docker-compose.yml",
		"workdir": tmpDir,
		"timeout": "invalid-duration",
	}

	result, err := node.Execute(context.Background(), inputs)
	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "InvalidTimeout")
	assert.Contains(t, err.Error(), "invalid timeout format")
}

// TestExecuteComposeUp_DefaultDetach tests up with default detach
func TestExecuteComposeUp_DefaultDetach(t *testing.T) {
	inputs := map[string]interface{}{
		"file": "docker-compose.yml",
	}

	args := buildComposeArgs([]string{"docker", "compose"}, inputs, "up")

	// Should not include -d yet (added in executeComposeUp)
	assert.NotContains(t, args, "-d")
	assert.Contains(t, args, "up")
}

// TestExecuteComposeDown_DefaultRemoveOrphans tests down with default remove_orphans
func TestExecuteComposeDown_DefaultRemoveOrphans(t *testing.T) {
	inputs := map[string]interface{}{
		"file": "docker-compose.yml",
	}

	args := buildComposeArgs([]string{"docker", "compose"}, inputs, "down")

	// Should not include --remove-orphans yet (added in executeComposeDown)
	assert.NotContains(t, args, "--remove-orphans")
	assert.Contains(t, args, "down")
}

// TestCheckDockerComposeAvailable tests Compose availability detection
func TestCheckDockerComposeAvailable(t *testing.T) {
	// This test will vary based on environment
	cmd, err := checkDockerComposeAvailable()

	if err != nil {
		// Docker Compose not installed
		assert.Contains(t, err.Error(), "docker-compose not found")
	} else {
		// Docker Compose is available
		assert.NotNil(t, cmd)
		// Should be either ["docker", "compose"] or ["docker-compose"]
		assert.True(t, len(cmd) > 0)
	}
}

// TestClassifyComposeError_AllTypes tests all error classifications
func TestClassifyComposeError_AllTypes(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		stderr   string
		errType  string
	}{
		{"context deadline", 1, "context deadline exceeded", "TimeoutError"},
		{"timeout", 1, "operation timeout", "TimeoutError"},
		{"no such file", 1, "no such file or directory", "ComposeFileNotFound"},
		{"file not found", 1, "file not found: docker-compose.yml", "ComposeFileNotFound"},
		{"yaml error", 1, "yaml: line 5: mapping values", "ComposeConfigError"},
		{"parse error", 1, "parse error in config", "ComposeConfigError"},
		{"invalid syntax", 1, "invalid service definition", "ComposeConfigError"},
		{"port allocated", 1, "port 8080 is already allocated", "PortConflict"},
		{"network timeout", 1, "network timeout occurred", "NetworkError"},
		{"connection failed", 1, "connection refused", "NetworkError"},
		{"pull failed", 1, "failed to pull image", "ImagePullError"},
		{"image not found", 1, "image nginx:latest not found", "ImagePullError"},
		{"unknown error", 1, "something went wrong", "ComposeCommandFailed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := classifyComposeError(tt.exitCode, tt.stderr)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errType)
		})
	}
}
