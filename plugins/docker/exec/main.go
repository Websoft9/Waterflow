package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"go.temporal.io/sdk/temporal"
)

// DockerExecNode implements Docker CLI command execution
type DockerExecNode struct{}

// Register is the Go Plugin exported registration function
func Register() node.Node {
	return &DockerExecNode{}
}

// Name returns the node name
func (n *DockerExecNode) Name() string {
	return "docker/exec"
}

// Version returns the node version
func (n *DockerExecNode) Version() string {
	return "v1"
}

// Params returns parameter specifications (legacy method from Story 1.3)
func (n *DockerExecNode) Params() map[string]node.ParamSpec {
	return n.Metadata().InputSchema
}

// Metadata returns node metadata
func (n *DockerExecNode) Metadata() node.NodeMetadata {
	return node.NodeMetadata{
		Description: "Execute Docker CLI commands on the agent server",
		Category:    "docker",
		InputSchema: map[string]node.ParamSpec{
			"command": {
				Type:        "string",
				Required:    true,
				Description: "Docker subcommand (e.g., 'run', 'ps', 'stop', 'rm', 'images', 'pull', 'exec', 'logs')",
			},
			"args": {
				Type:        "array",
				Required:    false,
				Description: "Command arguments (passed after the subcommand)",
			},
			"timeout": {
				Type:        "string",
				Required:    false,
				Default:     "5m",
				Description: "Command timeout (e.g., '30s', '5m', '10m' - use longer for image pulls)",
			},
			"docker_host": {
				Type:        "string",
				Required:    false,
				Description: "Docker daemon socket (default: unix:///var/run/docker.sock)",
			},
		},
		OutputSchema: map[string]interface{}{
			"exit_code":    "int",
			"stdout":       "string",
			"stderr":       "string",
			"elapsed_ms":   "int",
			"command_line": "string",
		},
	}
}

// Execute runs the Docker command
func (n *DockerExecNode) Execute(
	ctx context.Context,
	inputs map[string]interface{},
) (*node.NodeResult, error) {
	startTime := time.Now()

	// 1. Check Docker availability
	if err := checkDockerAvailable(); err != nil {
		return nil, err
	}

	// 2. Parse and validate parameters
	command, ok := inputs["command"].(string)
	if !ok || command == "" {
		return nil, temporal.NewNonRetryableApplicationError(
			"command is required and must be a non-empty string",
			"InvalidParameter",
			nil,
		)
	}

	// Parse args
	var args []string
	if argsInput, ok := inputs["args"].([]interface{}); ok {
		for _, arg := range argsInput {
			if strArg, ok := arg.(string); ok {
				args = append(args, strArg)
			}
		}
	}

	// Parse timeout
	timeout := 5 * time.Minute // default
	if timeoutStr, ok := inputs["timeout"].(string); ok && timeoutStr != "" {
		if d, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = d
		}
	}

	// 3. Build command
	cmdArgs := append([]string{command}, args...)
	cmdLine := "docker " + strings.Join(cmdArgs, " ")

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// #nosec G204 - Docker command with validated subcommand and args
	cmd := exec.CommandContext(execCtx, "docker", cmdArgs...)

	// Set DOCKER_HOST if provided
	if dockerHost, ok := inputs["docker_host"].(string); ok && dockerHost != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("DOCKER_HOST=%s", dockerHost))
	}

	// 4. Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 5. Execute command
	err := cmd.Run()

	elapsed := time.Since(startTime)
	exitCode := 0

	if err != nil {
		// Get exit code
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Non-exit error (e.g., timeout, cannot start)
			return nil, classifyDockerError(-1, err.Error())
		}
	}

	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	// 6. Check for errors
	if exitCode != 0 {
		return nil, classifyDockerError(exitCode, stderrStr)
	}

	// 7. Return result
	result := node.NewNodeResult()
	result.SetOutput("exit_code", exitCode)
	result.SetOutput("stdout", stdoutStr)
	result.SetOutput("stderr", stderrStr)
	result.SetOutput("elapsed_ms", elapsed.Milliseconds())
	result.SetOutput("command_line", cmdLine)

	result.AddLog(fmt.Sprintf("Executing: %s", cmdLine))
	result.AddLog(fmt.Sprintf("Exit code: %d, Duration: %dms", exitCode, elapsed.Milliseconds()))

	result.Duration = elapsed

	return result, nil
}

// checkDockerAvailable checks if Docker is installed and daemon is running
func checkDockerAvailable() error {
	// Check if docker command exists
	if _, err := exec.LookPath("docker"); err != nil {
		return temporal.NewNonRetryableApplicationError(
			"docker command not found - please install Docker Engine on the agent server",
			"DockerNotInstalled",
			nil,
		)
	}

	// Check Docker daemon status
	cmd := exec.Command("docker", "info")
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)

		// Daemon not running
		if strings.Contains(outputStr, "Cannot connect to the Docker daemon") {
			return temporal.NewNonRetryableApplicationError(
				"Docker daemon is not running - please start Docker service (e.g., 'systemctl start docker')",
				"DockerDaemonNotRunning",
				nil,
			)
		}

		// Permission denied
		if strings.Contains(outputStr, "permission denied") {
			return temporal.NewNonRetryableApplicationError(
				"permission denied - agent user needs to be in docker group (run 'sudo usermod -aG docker <user>')",
				"DockerPermissionDenied",
				nil,
			)
		}

		return temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("failed to check Docker availability: %v", err),
			"DockerCheckFailed",
			nil,
		)
	}

	return nil
}

// classifyDockerError categorizes Docker errors for Temporal retry logic
func classifyDockerError(exitCode int, stderr string) error {
	stderrLower := strings.ToLower(stderr)

	// Timeout errors - check first to avoid being caught by network check
	if strings.Contains(stderrLower, "context deadline exceeded") ||
		strings.Contains(stderrLower, "timeout") ||
		strings.Contains(stderrLower, "dial") {
		return temporal.NewApplicationError(
			stderr,
			"TimeoutError",
		)
	}

	// Container not found - permanent error
	if strings.Contains(stderrLower, "no such container") {
		return temporal.NewNonRetryableApplicationError(
			stderr,
			"ContainerNotFound",
			nil,
		)
	}

	// Image not found - temporary error (might need pull or network issue)
	if strings.Contains(stderrLower, "no such image") ||
		strings.Contains(stderrLower, "unable to find image") {
		return temporal.NewApplicationError(
			stderr,
			"ImageNotFound",
		)
	}

	// Permission errors - permanent error
	if strings.Contains(stderrLower, "permission denied") ||
		strings.Contains(stderrLower, "access denied") {
		return temporal.NewNonRetryableApplicationError(
			stderr,
			"PermissionDenied",
			nil,
		)
	}

	// Daemon connection errors - permanent error
	if strings.Contains(stderrLower, "cannot connect to the docker daemon") {
		return temporal.NewNonRetryableApplicationError(
			stderr,
			"DockerDaemonError",
			nil,
		)
	}

	// Network errors - temporary error (retriable)
	if strings.Contains(stderrLower, "network") {
		return temporal.NewApplicationError(
			stderr,
			"NetworkError",
		)
	}

	// Other errors - permanent error by default
	return temporal.NewNonRetryableApplicationError(
		fmt.Sprintf("docker command failed (exit code %d): %s", exitCode, stderr),
		"DockerCommandFailed",
		nil,
	)
}
