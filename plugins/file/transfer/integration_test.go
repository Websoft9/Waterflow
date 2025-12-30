//go:build integration
// +build integration

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

// Integration tests require a running SSH/SFTP server
// Run with: go test -v -tags=integration
//
// Setup SSH server using Docker:
//   docker run -d --name sftp-test -p 2222:22 \
//     -v /tmp/sftp-upload:/home/testuser/upload \
//     atmoz/sftp testuser:testpass:::upload

const (
	testHost     = "localhost"
	testPort     = 2222
	testUser     = "testuser"
	testPassword = "testpass"
)

// TestIntegration_SFTPUpload tests real file upload using SFTP
func TestIntegration_SFTPUpload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	node := &FileTransferNode{}

	// Create test file
	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "test-upload.txt")
	testContent := "Hello from Waterflow file transfer node!"
	err := os.WriteFile(sourceFile, []byte(testContent), 0644)
	require.NoError(t, err)

	inputs := map[string]interface{}{
		"mode":           "upload",
		"protocol":       "sftp",
		"host":           testHost,
		"port":           testPort,
		"user":           testUser,
		"password":       testPassword,
		"source_path":    sourceFile,
		"target_path":    "/upload/test-upload.txt",
		"permissions":    "0644",
		"timeout":        "10s",
		"host_key_check": false, // Disable for test environment
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := node.Execute(ctx, inputs)
	require.NoError(t, err, "Upload should succeed")
	require.NotNil(t, result)

	// Verify outputs
	assert.Equal(t, int64(len(testContent)), result.Outputs["transferred_bytes"])
	assert.Equal(t, int64(len(testContent)), result.Outputs["file_size"])
	assert.Equal(t, "upload", result.Outputs["mode"])
	assert.Equal(t, "/upload/test-upload.txt", result.Outputs["remote_path"])
	assert.Greater(t, result.Outputs["elapsed_ms"].(int64), int64(0))
	assert.NotEmpty(t, result.Logs)
}

// TestIntegration_SFTPDownload tests real file download using SFTP
func TestIntegration_SFTPDownload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	node := &FileTransferNode{}

	// First upload a file to download later
	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "test-source.txt")
	testContent := "Test content for download"
	err := os.WriteFile(sourceFile, []byte(testContent), 0644)
	require.NoError(t, err)

	// Upload first
	uploadInputs := map[string]interface{}{
		"mode":           "upload",
		"host":           testHost,
		"port":           testPort,
		"user":           testUser,
		"password":       testPassword,
		"source_path":    sourceFile,
		"target_path":    "/upload/download-test.txt",
		"host_key_check": false,
	}

	ctx := context.Background()
	_, err = node.Execute(ctx, uploadInputs)
	require.NoError(t, err, "Upload should succeed before download test")

	// Now download
	targetFile := filepath.Join(tmpDir, "downloaded.txt")
	downloadInputs := map[string]interface{}{
		"mode":           "download",
		"protocol":       "sftp",
		"host":           testHost,
		"port":           testPort,
		"user":           testUser,
		"password":       testPassword,
		"source_path":    "/upload/download-test.txt",
		"target_path":    targetFile,
		"timeout":        "10s",
		"host_key_check": false,
	}

	result, err := node.Execute(ctx, downloadInputs)
	require.NoError(t, err, "Download should succeed")
	require.NotNil(t, result)

	// Verify file downloaded
	assert.FileExists(t, targetFile)
	content, err := os.ReadFile(targetFile)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(content))

	// Verify outputs
	assert.Equal(t, int64(len(testContent)), result.Outputs["transferred_bytes"])
	assert.Equal(t, "download", result.Outputs["mode"])
	assert.Equal(t, "/upload/download-test.txt", result.Outputs["remote_path"])
}

// TestIntegration_SCPProtocol tests SCP protocol (which uses SFTP internally)
func TestIntegration_SCPProtocol(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	node := &FileTransferNode{}

	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "scp-test.txt")
	err := os.WriteFile(sourceFile, []byte("SCP test"), 0644)
	require.NoError(t, err)

	inputs := map[string]interface{}{
		"mode":           "upload",
		"protocol":       "scp", // Should use SFTP internally
		"host":           testHost,
		"port":           testPort,
		"user":           testUser,
		"password":       testPassword,
		"source_path":    sourceFile,
		"target_path":    "/upload/scp-test.txt",
		"host_key_check": false,
	}

	result, err := node.Execute(context.Background(), inputs)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify logs mention SFTP (via SCP)
	assert.NotEmpty(t, result.Logs)
	assert.Contains(t, result.Logs[0], "SFTP (VIA SCP)", "Should clarify SCP uses SFTP")
}

// TestIntegration_FilePermissions tests file permission setting
func TestIntegration_FilePermissions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	node := &FileTransferNode{}

	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "perms-test.txt")
	err := os.WriteFile(sourceFile, []byte("permissions test"), 0600)
	require.NoError(t, err)

	inputs := map[string]interface{}{
		"mode":           "upload",
		"host":           testHost,
		"port":           testPort,
		"user":           testUser,
		"password":       testPassword,
		"source_path":    sourceFile,
		"target_path":    "/upload/perms-test.txt",
		"permissions":    "0755", // Executable permissions
		"host_key_check": false,
	}

	result, err := node.Execute(context.Background(), inputs)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Note: Actual permission verification would require SSH command to stat the file
	// For now, we verify the upload succeeded without errors
	assert.Greater(t, result.Outputs["transferred_bytes"].(int64), int64(0))
}

// TestIntegration_InvalidPermissions tests that invalid permissions format doesn't break upload
func TestIntegration_InvalidPermissions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	node := &FileTransferNode{}

	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "bad-perms.txt")
	err := os.WriteFile(sourceFile, []byte("test"), 0644)
	require.NoError(t, err)

	inputs := map[string]interface{}{
		"mode":           "upload",
		"host":           testHost,
		"port":           testPort,
		"user":           testUser,
		"password":       testPassword,
		"source_path":    sourceFile,
		"target_path":    "/upload/bad-perms.txt",
		"permissions":    "644", // Invalid format (missing leading 0)
		"host_key_check": false,
	}

	// Should succeed with warning, not fail
	result, err := node.Execute(context.Background(), inputs)
	require.NoError(t, err, "Upload should succeed even with invalid permissions format")
	require.NotNil(t, result)
	assert.Greater(t, result.Outputs["transferred_bytes"].(int64), int64(0))
}

// TestIntegration_LargeFile tests large file transfer
func TestIntegration_LargeFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	node := &FileTransferNode{}

	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "large-file.bin")

	// Create 1MB file
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	err := os.WriteFile(sourceFile, largeData, 0644)
	require.NoError(t, err)

	inputs := map[string]interface{}{
		"mode":           "upload",
		"host":           testHost,
		"port":           testPort,
		"user":           testUser,
		"password":       testPassword,
		"source_path":    sourceFile,
		"target_path":    "/upload/large-file.bin",
		"timeout":        "30s",
		"host_key_check": false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	result, err := node.Execute(ctx, inputs)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, int64(len(largeData)), result.Outputs["transferred_bytes"])
	assert.Greater(t, result.Outputs["elapsed_ms"].(int64), int64(0))
}

// TestIntegration_ContextCancellation tests context cancellation during transfer
func TestIntegration_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	node := &FileTransferNode{}

	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "cancel-test.bin")

	// Create large file to ensure transfer takes time
	largeData := make([]byte, 10*1024*1024) // 10MB
	err := os.WriteFile(sourceFile, largeData, 0644)
	require.NoError(t, err)

	inputs := map[string]interface{}{
		"mode":           "upload",
		"host":           testHost,
		"port":           testPort,
		"user":           testUser,
		"password":       testPassword,
		"source_path":    sourceFile,
		"target_path":    "/upload/cancel-test.bin",
		"host_key_check": false,
	}

	// Cancel context almost immediately
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = node.Execute(ctx, inputs)
	// Should fail due to context cancellation (connection timeout or transfer interrupted)
	assert.Error(t, err, "Should fail when context is cancelled")
}
