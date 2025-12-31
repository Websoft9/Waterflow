// Package main implements the Docker Compose node plugin
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"go.temporal.io/sdk/temporal"
)

// DockerComposeNode implements Docker Compose operations (up/down)
type DockerComposeNode struct{}

// Name returns the node name
func (n *DockerComposeNode) Name() string {
	return "docker/compose"
}

// Version returns the node version
func (n *DockerComposeNode) Version() string {
	return "v1"
}

// Params returns parameter specifications
func (n *DockerComposeNode) Params() map[string]node.ParamSpec {
	return n.Metadata().InputSchema
}

// Metadata returns node metadata with input/output schemas
func (n *DockerComposeNode) Metadata() node.NodeMetadata {
	return node.NodeMetadata{
		Description: "Manage Docker Compose stacks (up/down)",
		Category:    "docker",
		InputSchema: map[string]node.ParamSpec{
			"action": {
				Type:        "string",
				Required:    true,
				Description: "Compose action: 'up' or 'down'",
				Enum:        []interface{}{"up", "down"},
			},
			"file": {
				Type:        "string",
				Required:    false,
				Default:     "docker-compose.yml",
				Description: "Compose file path",
			},
			"project_name": {
				Type:        "string",
				Required:    false,
				Description: "Project name (-p flag)",
			},
			"workdir": {
				Type:        "string",
				Required:    false,
				Description: "Working directory",
			},
			"env": {
				Type:        "object",
				Required:    false,
				Description: "Environment variables",
			},
			"timeout": {
				Type:        "string",
				Required:    false,
				Default:     "10m",
				Description: "Operation timeout",
			},
			"detach": {
				Type:        "bool",
				Required:    false,
				Default:     true,
				Description: "Detached mode (-d)",
			},
			"build": {
				Type:        "bool",
				Required:    false,
				Default:     false,
				Description: "Build images before starting (--build)",
			},
			"force_recreate": {
				Type:        "bool",
				Required:    false,
				Default:     false,
				Description: "Recreate containers (--force-recreate)",
			},
			"volumes": {
				Type:        "bool",
				Required:    false,
				Default:     false,
				Description: "Remove volumes (-v)",
			},
			"rmi": {
				Type:        "string",
				Required:    false,
				Description: "Remove images: 'local' or 'all'",
				Enum:        []interface{}{"", "local", "all"},
			},
			"remove_orphans": {
				Type:        "bool",
				Required:    false,
				Default:     true,
				Description: "Remove orphan containers (--remove-orphans)",
			},
		},
		OutputSchema: map[string]interface{}{
			"action":     "string",
			"containers": "array",
			"services":   "array",
			"exit_code":  "int",
			"stdout":     "string",
			"stderr":     "string",
			"elapsed_ms": "int",
		},
	}
}

// Execute performs Docker Compose operations
func (n *DockerComposeNode) Execute(
	ctx context.Context,
	inputs map[string]interface{},
) (*node.NodeResult, error) {
	startTime := time.Now()

	composeCmd, err := checkDockerComposeAvailable()
	if err != nil {
		return nil, err
	}

	action, ok := inputs["action"].(string)
	if !ok || (action != "up" && action != "down") {
		return nil, temporal.NewNonRetryableApplicationError(
			"action must be 'up' or 'down'",
			"InvalidParameter",
			nil,
		)
	}

	file := "docker-compose.yml"
	if f, ok := inputs["file"].(string); ok && f != "" {
		file = f
	}

	workdir := ""
	if w, ok := inputs["workdir"].(string); ok && w != "" {
		workdir = w
	}

	if err := validateComposeFile(file, workdir); err != nil {
		return nil, err
	}

	timeout := 10 * time.Minute
	if timeoutStr, ok := inputs["timeout"].(string); ok && timeoutStr != "" {
		d, err := time.ParseDuration(timeoutStr)
		if err != nil {
			return nil, temporal.NewNonRetryableApplicationError(
				fmt.Sprintf("invalid timeout format: %s (expected: 10m, 1h, etc.)", timeoutStr),
				"InvalidTimeout",
				nil,
			)
		}
		timeout = d
	}

	var exitCode int
	var stdout, stderr string
	var containers, services []string

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if action == "up" {
		exitCode, stdout, stderr, err = executeComposeUp(execCtx, composeCmd, inputs)
	} else {
		exitCode, stdout, stderr, err = executeComposeDown(execCtx, composeCmd, inputs)
	}

	if err != nil {
		return nil, err
	}

	containers, services, _ = getComposeContainers(composeCmd, file, inputs)

	elapsed := time.Since(startTime)

	return &node.NodeResult{
		Outputs: map[string]interface{}{
			"action":     action,
			"containers": containers,
			"services":   services,
			"exit_code":  exitCode,
			"stdout":     stdout,
			"stderr":     stderr,
			"elapsed_ms": elapsed.Milliseconds(),
		},
		Logs: []string{
			fmt.Sprintf("Docker Compose %s: %s", action, file),
			fmt.Sprintf("%d containers, %d services, exit_code=%d, duration=%dms",
				len(containers), len(services), exitCode, elapsed.Milliseconds()),
		},
		Duration: elapsed,
	}, nil
}

// checkDockerComposeAvailable checks if Docker Compose is available
func checkDockerComposeAvailable() ([]string, error) {
	cmd := exec.Command("docker", "compose", "version")
	if err := cmd.Run(); err == nil {
		return []string{"docker", "compose"}, nil
	}

	if _, err := exec.LookPath("docker-compose"); err == nil {
		return []string{"docker-compose"}, nil
	}

	return nil, temporal.NewNonRetryableApplicationError(
		"docker-compose not found - please install Docker Compose",
		"DockerComposeNotInstalled",
		nil,
	)
}

// validateComposeFile validates the Compose file exists
func validateComposeFile(file, workdir string) error {
	// Prevent path traversal attacks
	if strings.Contains(file, "..") {
		return temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("invalid file path (path traversal detected): %s", file),
			"InvalidFilePath",
			nil,
		)
	}

	filePath := file
	if workdir != "" {
		// Validate workdir doesn't contain path traversal
		if strings.Contains(workdir, "..") {
			return temporal.NewNonRetryableApplicationError(
				fmt.Sprintf("invalid workdir (path traversal detected): %s", workdir),
				"InvalidWorkdir",
				nil,
			)
		}
		filePath = filepath.Join(workdir, file)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("compose file not found: %s", filePath),
			"ComposeFileNotFound",
			nil,
		)
	}

	return nil
}

// executeComposeUp executes docker-compose up
func executeComposeUp(
	ctx context.Context,
	composeCmd []string,
	inputs map[string]interface{},
) (int, string, string, error) {
	args := buildComposeArgs(composeCmd, inputs, "up")

	if detach, ok := inputs["detach"].(bool); !ok || detach {
		args = append(args, "-d")
	}

	if build, ok := inputs["build"].(bool); ok && build {
		args = append(args, "--build")
	}

	if forceRecreate, ok := inputs["force_recreate"].(bool); ok && forceRecreate {
		args = append(args, "--force-recreate")
	}

	return executeComposeCommand(ctx, composeCmd[0], args, inputs)
}

// executeComposeDown executes docker-compose down
func executeComposeDown(
	ctx context.Context,
	composeCmd []string,
	inputs map[string]interface{},
) (int, string, string, error) {
	args := buildComposeArgs(composeCmd, inputs, "down")

	if volumes, ok := inputs["volumes"].(bool); ok && volumes {
		args = append(args, "-v")
	}

	if rmi, ok := inputs["rmi"].(string); ok && rmi != "" {
		args = append(args, "--rmi", rmi)
	}

	if removeOrphans, ok := inputs["remove_orphans"].(bool); !ok || removeOrphans {
		args = append(args, "--remove-orphans")
	}

	return executeComposeCommand(ctx, composeCmd[0], args, inputs)
}

// buildComposeArgs builds common Compose arguments
func buildComposeArgs(
	composeCmd []string,
	inputs map[string]interface{},
	action string,
) []string {
	var args []string

	if len(composeCmd) > 1 {
		args = append(args, composeCmd[1:]...)
	}

	if file, ok := inputs["file"].(string); ok && file != "" {
		args = append(args, "-f", file)
	} else {
		args = append(args, "-f", "docker-compose.yml")
	}

	if projectName, ok := inputs["project_name"].(string); ok && projectName != "" {
		args = append(args, "-p", projectName)
	}

	args = append(args, action)

	return args
}

// executeComposeCommand executes a Compose command
func executeComposeCommand(
	ctx context.Context,
	cmdName string,
	args []string,
	inputs map[string]interface{},
) (int, string, string, error) {
	// #nosec G204 - cmdName is validated by checkDockerComposeAvailable
	cmd := exec.CommandContext(ctx, cmdName, args...)

	if workdir, ok := inputs["workdir"].(string); ok && workdir != "" {
		cmd.Dir = workdir
	}

	cmd.Env = os.Environ()
	if envMap, ok := inputs["env"].(map[string]interface{}); ok {
		for k, v := range envMap {
			if strVal, ok := v.(string); ok {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, strVal))
			}
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Non-exit error (e.g., context canceled, command not found)
			errMsg := fmt.Sprintf("command execution failed: %s (cmd: %s %v)", err.Error(), cmdName, args)
			return -1, stdoutStr, stderrStr, classifyComposeError(-1, errMsg)
		}
	}

	if exitCode != 0 {
		return exitCode, stdoutStr, stderrStr, classifyComposeError(exitCode, stderrStr)
	}

	return exitCode, stdoutStr, stderrStr, nil
}

// getComposeContainers retrieves the list of Compose containers
func getComposeContainers(
	composeCmd []string,
	file string,
	inputs map[string]interface{},
) ([]string, []string, error) {
	args := []string{}

	if len(composeCmd) > 1 {
		args = append(args, composeCmd[1:]...)
	}

	args = append(args, "-f", file)

	if projectName, ok := inputs["project_name"].(string); ok && projectName != "" {
		args = append(args, "-p", projectName)
	}

	args = append(args, "ps", "--format", "json")

	// #nosec G204 - composeCmd is validated by checkDockerComposeAvailable
	cmd := exec.Command(composeCmd[0], args...)

	if workdir, ok := inputs["workdir"].(string); ok && workdir != "" {
		cmd.Dir = workdir
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, nil, err
	}

	var containers []string
	var services []string

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var container struct {
			Name    string `json:"Name"`
			Service string `json:"Service"`
		}

		if err := json.Unmarshal([]byte(line), &container); err == nil {
			containers = append(containers, container.Name)
			if container.Service != "" && !contains(services, container.Service) {
				services = append(services, container.Service)
			}
		}
	}

	return containers, services, nil
}

// classifyComposeError classifies Compose errors
func classifyComposeError(exitCode int, stderr string) error {
	stderrLower := strings.ToLower(stderr)

	// Check specific errors before generic timeout
	// Network errors (check before timeout as some network errors mention timeout)
	if strings.Contains(stderrLower, "network") ||
		strings.Contains(stderrLower, "connection") {
		return temporal.NewApplicationError(
			stderr,
			"NetworkError",
		)
	}

	// Context deadline and timeout errors
	if strings.Contains(stderrLower, "context deadline exceeded") ||
		strings.Contains(stderrLower, "timeout") {
		return temporal.NewApplicationError(
			stderr,
			"TimeoutError",
		)
	}

	if strings.Contains(stderrLower, "no such file") ||
		strings.Contains(stderrLower, "file not found") {
		return temporal.NewNonRetryableApplicationError(
			stderr,
			"ComposeFileNotFound",
			nil,
		)
	}

	if strings.Contains(stderrLower, "yaml") ||
		strings.Contains(stderrLower, "parse") ||
		strings.Contains(stderrLower, "invalid") {
		return temporal.NewNonRetryableApplicationError(
			stderr,
			"ComposeConfigError",
			nil,
		)
	}

	if strings.Contains(stderrLower, "port") &&
		strings.Contains(stderrLower, "already") {
		return temporal.NewNonRetryableApplicationError(
			stderr,
			"PortConflict",
			nil,
		)
	}

	if strings.Contains(stderrLower, "pull") ||
		strings.Contains(stderrLower, "image") {
		return temporal.NewApplicationError(
			stderr,
			"ImagePullError",
		)
	}

	return temporal.NewNonRetryableApplicationError(
		fmt.Sprintf("compose command failed (exit code %d): %s", exitCode, stderr),
		"ComposeCommandFailed",
		nil,
	)
}

// contains checks if a slice contains an item
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Register is the Go Plugin entry point
func Register() node.Node {
	return &DockerComposeNode{}
}
