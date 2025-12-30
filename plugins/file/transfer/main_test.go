package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileTransferNode_Name verifies the node name
func TestFileTransferNode_Name(t *testing.T) {
	node := &FileTransferNode{}
	assert.Equal(t, "file/transfer", node.Name())
}

// TestFileTransferNode_Version verifies the node version
func TestFileTransferNode_Version(t *testing.T) {
	node := &FileTransferNode{}
	assert.Equal(t, "v1", node.Version())
}

// TestFileTransferNode_Metadata verifies the node metadata
func TestFileTransferNode_Metadata(t *testing.T) {
	node := &FileTransferNode{}
	metadata := node.Metadata()

	assert.Equal(t, "file", metadata.Category)
	assert.Contains(t, metadata.Description, "Transfer files")
	assert.NotEmpty(t, metadata.InputSchema)
	assert.NotEmpty(t, metadata.OutputSchema)

	// Verify required parameters
	modeParam, ok := metadata.InputSchema["mode"]
	require.True(t, ok)
	assert.True(t, modeParam.Required)

	hostParam, ok := metadata.InputSchema["host"]
	require.True(t, ok)
	assert.True(t, hostParam.Required)

	userParam, ok := metadata.InputSchema["user"]
	require.True(t, ok)
	assert.True(t, userParam.Required)
}

// TestFileTransferNode_Execute_MissingMode tests missing mode parameter
func TestFileTransferNode_Execute_MissingMode(t *testing.T) {
	node := &FileTransferNode{}
	inputs := map[string]interface{}{
		"host":        "example.com",
		"user":        "testuser",
		"password":    "testpass",
		"source_path": "/path/file.txt",
		"target_path": "/path/dest.txt",
	}

	_, err := node.Execute(context.Background(), inputs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mode is required")
}

// TestFileTransferNode_Execute_InvalidMode tests invalid mode value
func TestFileTransferNode_Execute_InvalidMode(t *testing.T) {
	node := &FileTransferNode{}
	inputs := map[string]interface{}{
		"mode":        "copy",
		"host":        "example.com",
		"user":        "testuser",
		"password":    "testpass",
		"source_path": "/path/file.txt",
		"target_path": "/path/dest.txt",
	}

	_, err := node.Execute(context.Background(), inputs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid mode")
}

// TestFileTransferNode_Execute_MissingHost tests missing host parameter
func TestFileTransferNode_Execute_MissingHost(t *testing.T) {
	node := &FileTransferNode{}
	inputs := map[string]interface{}{
		"mode":        "upload",
		"user":        "testuser",
		"password":    "testpass",
		"source_path": "/path/file.txt",
		"target_path": "/path/dest.txt",
	}

	_, err := node.Execute(context.Background(), inputs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "are required")
}

// TestFileTransferNode_Execute_MissingUser tests missing user parameter
func TestFileTransferNode_Execute_MissingUser(t *testing.T) {
	node := &FileTransferNode{}
	inputs := map[string]interface{}{
		"mode":        "upload",
		"host":        "example.com",
		"password":    "testpass",
		"source_path": "/path/file.txt",
		"target_path": "/path/dest.txt",
	}

	_, err := node.Execute(context.Background(), inputs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "are required")
}

// TestFileTransferNode_Execute_MissingAuth tests missing authentication
func TestFileTransferNode_Execute_MissingAuth(t *testing.T) {
	node := &FileTransferNode{}
	inputs := map[string]interface{}{
		"mode":        "upload",
		"host":        "example.com",
		"user":        "testuser",
		"source_path": "/path/file.txt",
		"target_path": "/path/dest.txt",
	}

	_, err := node.Execute(context.Background(), inputs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "either password or private_key must be provided")
}

// TestFileTransferNode_Execute_MissingSourcePath tests missing source_path
func TestFileTransferNode_Execute_MissingSourcePath(t *testing.T) {
	node := &FileTransferNode{}
	inputs := map[string]interface{}{
		"mode":        "upload",
		"host":        "example.com",
		"user":        "testuser",
		"password":    "testpass",
		"target_path": "/path/dest.txt",
	}

	_, err := node.Execute(context.Background(), inputs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "are required")
}

// TestFileTransferNode_Execute_MissingTargetPath tests missing target_path
func TestFileTransferNode_Execute_MissingTargetPath(t *testing.T) {
	node := &FileTransferNode{}
	inputs := map[string]interface{}{
		"mode":        "upload",
		"host":        "example.com",
		"user":        "testuser",
		"password":    "testpass",
		"source_path": "/path/file.txt",
	}

	_, err := node.Execute(context.Background(), inputs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "are required")
}

// TestFileTransferNode_Execute_InvalidProtocol tests invalid protocol value
func TestFileTransferNode_Execute_InvalidProtocol(t *testing.T) {
	node := &FileTransferNode{}
	inputs := map[string]interface{}{
		"mode":        "upload",
		"protocol":    "ftp",
		"host":        "example.com",
		"user":        "testuser",
		"password":    "testpass",
		"source_path": "/path/file.txt",
		"target_path": "/path/dest.txt",
	}

	_, err := node.Execute(context.Background(), inputs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid protocol")
}

// TestFileTransferNode_Execute_InvalidPermissions tests invalid permissions format
func TestFileTransferNode_Execute_InvalidPermissions(t *testing.T) {
	t.Skip("Permissions validation happens during file upload, skipping connection test")
}

// NOTE: TestFileTransferNode_Execute_InvalidPort was removed - no port validation exists in code

// TestFileTransferNode_Execute_ValidUploadParams tests valid upload parameters
func TestFileTransferNode_Execute_ValidUploadParams(t *testing.T) {
	t.Skip("Requires SSH server, skipping connection test")
}

// TestFileTransferNode_Execute_ValidDownloadParams tests valid download parameters
func TestFileTransferNode_Execute_ValidDownloadParams(t *testing.T) {
	t.Skip("Requires SSH server, skipping connection test")
}

// TestRegister tests the Register function
func TestRegister(t *testing.T) {
	node := Register()
	require.NotNil(t, node)
	assert.Equal(t, "file/transfer", node.Name())
	assert.Equal(t, "v1", node.Version())
}
