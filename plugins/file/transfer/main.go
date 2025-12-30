// Package main implements a file transfer node for cross-server file operations
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"github.com/websoft9/waterflow/pkg/dsl/node"
	"go.temporal.io/sdk/temporal"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// FileTransferNode implements file transfer functionality via SFTP/SSH
type FileTransferNode struct{}

// Name returns the node identifier
func (n *FileTransferNode) Name() string {
	return "file/transfer"
}

// Version returns the node version
func (n *FileTransferNode) Version() string {
	return "v1"
}

// Params returns the parameter specifications
func (n *FileTransferNode) Params() map[string]node.ParamSpec {
	return n.Metadata().InputSchema
}

// Metadata returns the node metadata including input/output schemas
func (n *FileTransferNode) Metadata() node.NodeMetadata {
	return node.NodeMetadata{
		Description: "Transfer files between servers using SFTP/SSH (upload/download)",
		Category:    "file",
		InputSchema: map[string]node.ParamSpec{
			"mode": {
				Type:        "string",
				Required:    true,
				Description: "Transfer mode: 'upload' (local→remote) or 'download' (remote→local)",
			},
			"protocol": {
				Type:        "string",
				Required:    false,
				Default:     "sftp",
				Description: "Transfer protocol: 'sftp' (recommended) or 'scp' (Note: SCP internally uses SFTP for reliability)",
			},
			"host": {
				Type:        "string",
				Required:    true,
				Description: "Remote host (IP or domain)",
			},
			"port": {
				Type:        "int",
				Required:    false,
				Default:     22,
				Description: "SSH port (default: 22)",
			},
			"user": {
				Type:        "string",
				Required:    true,
				Description: "SSH username",
			},
			"password": {
				Type:        "string",
				Required:    false,
				Description: "SSH password (alternative to private_key)",
			},
			"private_key": {
				Type:        "string",
				Required:    false,
				Description: "SSH private key (file path or inline PEM content)",
			},
			"source_path": {
				Type:        "string",
				Required:    true,
				Description: "Source file path (local for upload, remote for download)",
			},
			"target_path": {
				Type:        "string",
				Required:    true,
				Description: "Target file path (remote for upload, local for download)",
			},
			"permissions": {
				Type:        "string",
				Required:    false,
				Description: "File permissions for upload (e.g., '0644', '0755')",
			},
			"timeout": {
				Type:        "string",
				Required:    false,
				Default:     "30s",
				Description: "Connection timeout (e.g., '30s', '1m')",
			},
			"host_key_check": {
				Type:        "bool",
				Required:    false,
				Default:     true,
				Description: "Verify SSH host key using known_hosts file (~/.ssh/known_hosts or /etc/ssh/ssh_known_hosts)",
			},
		},
		OutputSchema: map[string]interface{}{
			"transferred_bytes": "int - Number of bytes transferred",
			"file_size":         "int - File size in bytes",
			"elapsed_ms":        "int - Transfer duration in milliseconds",
			"mode":              "string - Transfer mode used (upload/download)",
			"remote_path":       "string - Remote file path",
		},
	}
}

// Execute performs the file transfer operation
func (n *FileTransferNode) Execute(
	ctx context.Context,
	inputs map[string]interface{},
) (*node.NodeResult, error) {
	startTime := time.Now()

	// 1. Parse and validate mode
	mode, ok := inputs["mode"].(string)
	if !ok || mode == "" {
		return nil, temporal.NewNonRetryableApplicationError(
			"mode is required",
			"InvalidParameter",
			nil,
		)
	}
	mode = strings.ToLower(mode)
	if mode != "upload" && mode != "download" {
		return nil, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("invalid mode '%s': must be 'upload' or 'download'", mode),
			"InvalidParameter",
			nil,
		)
	}

	// 2. Parse protocol
	protocol := "sftp"
	if p, ok := inputs["protocol"].(string); ok && p != "" {
		protocol = strings.ToLower(p)
	}
	if protocol != "sftp" && protocol != "scp" {
		return nil, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("invalid protocol '%s': must be 'sftp' or 'scp'", protocol),
			"InvalidParameter",
			nil,
		)
	}

	// 3. Parse required parameters
	host, _ := inputs["host"].(string)
	user, _ := inputs["user"].(string)
	sourcePath, _ := inputs["source_path"].(string)
	targetPath, _ := inputs["target_path"].(string)

	if host == "" || user == "" || sourcePath == "" || targetPath == "" {
		return nil, temporal.NewNonRetryableApplicationError(
			"host, user, source_path, and target_path are required",
			"InvalidParameter",
			nil,
		)
	}

	// 4. Create SSH client
	client, err := createSSHClient(ctx, inputs)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// 5. Execute transfer based on mode
	var transferredBytes int64
	var remotePath string

	if mode == "upload" {
		remotePath = targetPath
		transferredBytes, err = sftpUpload(client, sourcePath, targetPath, inputs)
	} else { // download
		remotePath = sourcePath
		transferredBytes, err = sftpDownload(client, sourcePath, targetPath)
	}

	if err != nil {
		return nil, classifyTransferError(err)
	}

	elapsed := time.Since(startTime)

	// 6. Return result
	actualProtocol := protocol
	if protocol == "scp" {
		actualProtocol = "SFTP (via SCP)" // Clarify that SCP uses SFTP implementation
	}
	logs := []string{
		fmt.Sprintf("%s %s: %s → %s",
			strings.ToUpper(actualProtocol), mode, sourcePath, targetPath),
		fmt.Sprintf("Transferred %d bytes in %dms",
			transferredBytes, elapsed.Milliseconds()),
	}

	return &node.NodeResult{
		Outputs: map[string]interface{}{
			"transferred_bytes": transferredBytes,
			"file_size":         transferredBytes,
			"elapsed_ms":        elapsed.Milliseconds(),
			"mode":              mode,
			"remote_path":       remotePath,
		},
		Logs:     logs,
		Duration: elapsed,
	}, nil
}

// createSSHClient establishes an SSH connection to the remote server
func createSSHClient(ctx context.Context, inputs map[string]interface{}) (*ssh.Client, error) {
	host, _ := inputs["host"].(string)
	user, _ := inputs["user"].(string)

	// Parse port
	port := 22
	if p, ok := inputs["port"].(int); ok && p > 0 {
		port = p
	} else if p, ok := inputs["port"].(float64); ok && p > 0 {
		port = int(p)
	}

	// Parse timeout
	timeout := 30 * time.Second
	if timeoutStr, ok := inputs["timeout"].(string); ok && timeoutStr != "" {
		if d, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = d
		}
	}

	// Configure authentication methods
	var authMethods []ssh.AuthMethod

	// Password authentication
	if password, ok := inputs["password"].(string); ok && password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}

	// Private key authentication
	if privateKey, ok := inputs["private_key"].(string); ok && privateKey != "" {
		signer, err := parsePrivateKey(privateKey)
		if err != nil {
			return nil, temporal.NewNonRetryableApplicationError(
				fmt.Sprintf("failed to parse private key: %v", err),
				"AuthenticationError",
				nil,
			)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	if len(authMethods) == 0 {
		return nil, temporal.NewNonRetryableApplicationError(
			"either password or private_key must be provided",
			"InvalidParameter",
			nil,
		)
	}

	// Configure host key checking
	var hostKeyCallback ssh.HostKeyCallback
	hostKeyCheck := true
	if check, ok := inputs["host_key_check"].(bool); ok {
		hostKeyCheck = check
	}

	if hostKeyCheck {
		// Production: Verify host key using known_hosts file
		// Try multiple common locations for known_hosts
		knownHostsPaths := []string{
			filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"),
			"/etc/ssh/ssh_known_hosts",
		}

		var knownHostsErr error
		for _, khPath := range knownHostsPaths {
			if _, err := os.Stat(khPath); err == nil {
				hostKeyCallback, knownHostsErr = knownhosts.New(khPath)
				if knownHostsErr == nil {
					break
				}
			}
		}

		if hostKeyCallback == nil {
			// Fallback: If no known_hosts file found, use InsecureIgnoreHostKey with warning
			fmt.Fprintf(os.Stderr, "Warning: host_key_check=true but no known_hosts file found, using insecure mode\n")
			hostKeyCallback = ssh.InsecureIgnoreHostKey()
		}
	} else {
		// Development: Skip host key verification (insecure)
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	// Create SSH client config
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         timeout,
	}

	// Connect to SSH server
	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		// Classify SSH connection errors
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "authentication") ||
			strings.Contains(errMsg, "unable to authenticate") ||
			strings.Contains(errMsg, "permission denied") {
			return nil, temporal.NewNonRetryableApplicationError(
				fmt.Sprintf("SSH authentication failed: %v", err),
				"AuthenticationError",
				nil,
			)
		}
		return nil, temporal.NewApplicationError(
			fmt.Sprintf("SSH connection failed: %v", err),
			"ConnectionError",
		)
	}

	return client, nil
}

// parsePrivateKey parses an SSH private key from file path or inline content
func parsePrivateKey(key string) (ssh.Signer, error) {
	// Try as file path first
	if _, err := os.Stat(key); err == nil {
		keyBytes, err := os.ReadFile(key)
		if err != nil {
			return nil, fmt.Errorf("failed to read key file: %w", err)
		}
		return ssh.ParsePrivateKey(keyBytes)
	}

	// Try as inline content
	return ssh.ParsePrivateKey([]byte(key))
}

// sftpUpload uploads a file to remote server using SFTP
func sftpUpload(client *ssh.Client, source, target string, inputs map[string]interface{}) (int64, error) {
	// Create SFTP client
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return 0, fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Open local source file
	srcFile, err := os.Open(source)
	if err != nil {
		return 0, fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Create remote target directory if needed
	targetDir := filepath.Dir(target)
	if targetDir != "." && targetDir != "/" {
		sftpClient.MkdirAll(targetDir)
	}

	// Create remote target file
	dstFile, err := sftpClient.Create(target)
	if err != nil {
		return 0, fmt.Errorf("failed to create remote file: %w", err)
	}
	defer dstFile.Close()

	// Copy file content
	n, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return 0, fmt.Errorf("failed to transfer file: %w", err)
	}

	// Set file permissions if specified
	if permsStr, ok := inputs["permissions"].(string); ok && permsStr != "" {
		perms, err := strconv.ParseUint(permsStr, 8, 32)
		if err != nil {
			// Non-fatal: invalid format, warn but don't interrupt transfer
			fmt.Fprintf(os.Stderr, "Warning: invalid permissions format '%s' (expected octal like '0644'), skipping chmod\n", permsStr)
		} else {
			if err := sftpClient.Chmod(target, os.FileMode(perms)); err != nil {
				// Non-fatal: file was uploaded successfully, just permissions failed
				fmt.Fprintf(os.Stderr, "Warning: failed to set permissions %s on %s: %v\n", permsStr, target, err)
			}
		}
	}

	return n, nil
}

// sftpDownload downloads a file from remote server using SFTP
func sftpDownload(client *ssh.Client, source, target string) (int64, error) {
	// Create SFTP client
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return 0, fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Open remote source file
	srcFile, err := sftpClient.Open(source)
	if err != nil {
		return 0, fmt.Errorf("failed to open remote file: %w", err)
	}
	defer srcFile.Close()

	// Create local target directory
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return 0, fmt.Errorf("failed to create target directory: %w", err)
	}

	// Create local target file
	dstFile, err := os.Create(target)
	if err != nil {
		return 0, fmt.Errorf("failed to create local file: %w", err)
	}
	defer dstFile.Close()

	// Copy file content
	n, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return 0, fmt.Errorf("failed to transfer file: %w", err)
	}

	return n, nil
}

// classifyTransferError categorizes transfer errors for Temporal retry logic
func classifyTransferError(err error) error {
	errMsg := strings.ToLower(err.Error())

	// Authentication errors - permanent
	if containsAny(errMsg, "authentication", "unable to authenticate", "permission denied") {
		return temporal.NewNonRetryableApplicationError(
			err.Error(),
			"AuthenticationError",
			nil,
		)
	}

	// File not found errors - permanent
	if containsAny(errMsg, "no such file", "file not found", "does not exist") {
		return temporal.NewNonRetryableApplicationError(
			err.Error(),
			"FileNotFoundError",
			nil,
		)
	}

	// Permission errors - permanent
	if containsAny(errMsg, "permission denied", "access denied", "operation not permitted") {
		return temporal.NewNonRetryableApplicationError(
			err.Error(),
			"PermissionError",
			nil,
		)
	}

	// Network/timeout/I/O errors - temporary (retryable)
	return temporal.NewApplicationError(
		err.Error(),
		"TransferError",
	)
}

// containsAny checks if string contains any of the substrings
func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// Register is the exported function for Go plugin system
func Register() node.Node {
	return &FileTransferNode{}
}
