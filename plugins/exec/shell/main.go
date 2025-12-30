package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/websoft9/waterflow/pkg/dsl/node"
)

// parsedInputs holds validated and sanitized input parameters
type parsedInputs struct {
	command string
	args    []string
	env     map[string]string
	workdir string
	timeout int
}

// ShellNode implements shell command execution
type ShellNode struct{}

// Register is the Go Plugin exported registration function
func Register() node.Node {
	return &ShellNode{}
}

// Name returns the node name
func (n *ShellNode) Name() string {
	return "exec/shell"
}

// Version returns the node version
func (n *ShellNode) Version() string {
	return "v1"
}

// Params returns parameter specifications (legacy method from Story 1.3)
func (n *ShellNode) Params() map[string]node.ParamSpec {
	return n.Metadata().InputSchema
}

// Metadata returns node metadata
func (n *ShellNode) Metadata() node.NodeMetadata {
	minValue := float64(1)
	maxValue := float64(3600)

	return node.NodeMetadata{
		Description: "Execute shell commands on the agent server",
		Category:    "exec",
		InputSchema: map[string]node.ParamSpec{
			"command": {
				Type:        "string",
				Required:    true,
				Description: "Shell command to execute",
			},
			"args": {
				Type:        "array",
				Required:    false,
				Description: "Command arguments",
			},
			"env": {
				Type:        "object",
				Required:    false,
				Description: "Environment variables",
			},
			"workdir": {
				Type:        "string",
				Required:    false,
				Description: "Working directory",
			},
			"timeout": {
				Type:        "int",
				Required:    false,
				Default:     60,
				MinValue:    &minValue,
				MaxValue:    &maxValue,
				Description: "Timeout in seconds",
			},
		},
		OutputSchema: map[string]interface{}{
			"stdout":      "string",
			"stderr":      "string",
			"exit_code":   "int",
			"duration_ms": "int",
		},
	}
}

// parseInputs extracts and validates input parameters
func (n *ShellNode) parseInputs(inputs map[string]interface{}) (parsedInputs, error) {
	var parsed parsedInputs

	// Parse and validate command
	command, ok := inputs["command"].(string)
	if !ok || command == "" {
		return parsed, fmt.Errorf("command is required and must be a non-empty string")
	}
	parsed.command = command

	// Parse timeout with default
	parsed.timeout = 60
	if t, ok := inputs["timeout"].(int); ok {
		parsed.timeout = t
	} else if t, ok := inputs["timeout"].(float64); ok {
		parsed.timeout = int(t)
	}

	// Parse args array with safe shell escaping
	if args, ok := inputs["args"].([]interface{}); ok && len(args) > 0 {
		for _, arg := range args {
			parsed.args = append(parsed.args, fmt.Sprintf("%v", arg))
		}
	}

	// Parse env map with sanitized values
	if env, ok := inputs["env"].(map[string]interface{}); ok && len(env) > 0 {
		parsed.env = make(map[string]string)
		for k, v := range env {
			// Sanitize value: convert to string and remove newlines
			value := fmt.Sprintf("%v", v)
			// Remove characters that could break environment variable format
			for _, char := range []string{"\n", "\r", "\x00"} {
				value = strings.ReplaceAll(value, char, "")
			}
			parsed.env[k] = value
		}
	}

	// Parse workdir
	if workdir, ok := inputs["workdir"].(string); ok && workdir != "" {
		parsed.workdir = workdir
	}

	return parsed, nil
}

// buildCommand constructs the exec.Cmd with safe parameter handling
func (n *ShellNode) buildCommand(ctx context.Context, parsed parsedInputs) *exec.Cmd {
	var cmd *exec.Cmd

	// Build command with args using shell quoting for safety
	if len(parsed.args) > 0 {
		// Use shell's printf %q for safe quoting of each argument
		fullCommand := parsed.command
		for _, arg := range parsed.args {
			// Shell escape: wrap in single quotes and escape existing single quotes
			escaped := "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
			fullCommand += " " + escaped
		}
		cmd = exec.CommandContext(ctx, "sh", "-c", fullCommand)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", parsed.command)
	}

	// Apply environment variables
	if len(parsed.env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range parsed.env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	// Apply working directory
	if parsed.workdir != "" {
		cmd.Dir = parsed.workdir
	}

	return cmd
}

// Execute executes the shell command
func (n *ShellNode) Execute(
	ctx context.Context,
	inputs map[string]interface{},
) (*node.NodeResult, error) {
	startTime := time.Now()
	logs := []string{}

	// 1. Parse and validate inputs
	parsed, err := n.parseInputs(inputs)
	if err != nil {
		return nil, err
	}
	logs = append(logs, fmt.Sprintf("Executing: %s", parsed.command))
	if len(parsed.args) > 0 {
		logs = append(logs, fmt.Sprintf("With args: %v", parsed.args))
	}
	if len(parsed.env) > 0 {
		logs = append(logs, fmt.Sprintf("With env variables: %d", len(parsed.env)))
	}
	if parsed.workdir != "" {
		logs = append(logs, fmt.Sprintf("In directory: %s", parsed.workdir))
	}

	// 2. Create context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, time.Duration(parsed.timeout)*time.Second)
	defer cancel()

	// 3. Build command with safe parameter handling
	cmd := n.buildCommand(cmdCtx, parsed)

	// 4. Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 5. Execute command
	err = cmd.Run()
	duration := time.Since(startTime)

	// 6. Build outputs
	outputs := map[string]interface{}{
		"stdout":      stdout.String(),
		"stderr":      stderr.String(),
		"exit_code":   0,
		"duration_ms": duration.Milliseconds(),
	}

	// 7. Handle errors
	if err != nil {
		// Timeout error (temporary, retryable) - check first
		if cmdCtx.Err() == context.DeadlineExceeded {
			outputs["exit_code"] = -1
			logs = append(logs, fmt.Sprintf("Command timeout after %d seconds", parsed.timeout))
			return &node.NodeResult{
				Outputs:  outputs,
				Logs:     logs,
				Duration: duration,
			}, fmt.Errorf("command timeout after %d seconds", parsed.timeout)
		}

		// Context cancellation
		if ctx.Err() == context.Canceled {
			outputs["exit_code"] = -1
			logs = append(logs, "Command cancelled by context")
			return &node.NodeResult{
				Outputs:  outputs,
				Logs:     logs,
				Duration: duration,
			}, fmt.Errorf("command cancelled: %w", ctx.Err())
		}

		// Get exit code from ExitError
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			outputs["exit_code"] = exitCode
			logs = append(logs, fmt.Sprintf("Command failed with exit code %d", exitCode))
			return &node.NodeResult{
				Outputs:  outputs,
				Logs:     logs,
				Duration: duration,
			}, fmt.Errorf("command failed with exit code %d: %s", exitCode, stderr.String())
		}

		// Other errors (permanent, non-retryable)
		outputs["exit_code"] = -1
		logs = append(logs, fmt.Sprintf("Command execution failed: %v", err))
		return &node.NodeResult{
			Outputs:  outputs,
			Logs:     logs,
			Duration: duration,
		}, fmt.Errorf("command execution failed: %w", err)
	}

	// Success
	logs = append(logs, "Command completed successfully")
	return &node.NodeResult{
		Outputs:  outputs,
		Logs:     logs,
		Duration: duration,
	}, nil
}
