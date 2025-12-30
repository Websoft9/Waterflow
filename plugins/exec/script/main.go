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

// ScriptNode implements script file execution
type ScriptNode struct{}

// parsedInputs holds validated and sanitized input parameters
type parsedInputs struct {
	scriptPath    string
	scriptContent string
	interpreter   string
	args          []string
	env           map[string]string
	workdir       string
	timeout       int
	hasPath       bool
	hasContent    bool
}

// Register is the Go Plugin exported registration function
func Register() node.Node {
	return &ScriptNode{}
}

// Name returns the node name
func (n *ScriptNode) Name() string {
	return "exec/script"
}

// Version returns the node version
func (n *ScriptNode) Version() string {
	return "v1"
}

// Params returns parameter specifications (legacy method from Story 1.3)
func (n *ScriptNode) Params() map[string]node.ParamSpec {
	return n.Metadata().InputSchema
}

// Metadata returns node metadata
func (n *ScriptNode) Metadata() node.NodeMetadata {
	minValue := float64(1)
	maxValue := float64(3600)

	return node.NodeMetadata{
		Description: "Execute script files (Bash, Python, etc.) on the agent server",
		Category:    "exec",
		InputSchema: map[string]node.ParamSpec{
			"script_path": {
				Type:        "string",
				Required:    false,
				Description: "Path to script file (mutually exclusive with script_content)",
			},
			"script_content": {
				Type:        "string",
				Required:    false,
				Description: "Inline script content (mutually exclusive with script_path)",
			},
			"interpreter": {
				Type:        "string",
				Required:    false,
				Default:     "bash",
				Description: "Script interpreter (bash, sh, python3, python, node, ruby)",
			},
			"args": {
				Type:        "array",
				Required:    false,
				Description: "Script arguments",
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
			"stdout":           "string",
			"stderr":           "string",
			"exit_code":        "int",
			"duration_ms":      "int",
			"interpreter_used": "string",
		},
	}
}

// parseInputs extracts and validates input parameters
func (n *ScriptNode) parseInputs(inputs map[string]interface{}) (parsedInputs, error) {
	var parsed parsedInputs

	// Parse script_path and script_content
	if path, ok := inputs["script_path"].(string); ok && path != "" {
		parsed.scriptPath = path
		parsed.hasPath = true
	}
	if content, ok := inputs["script_content"].(string); ok && content != "" {
		parsed.scriptContent = content
		parsed.hasContent = true
	}

	// Validate mutual exclusivity
	if !parsed.hasPath && !parsed.hasContent {
		return parsed, fmt.Errorf("either script_path or script_content must be specified")
	}
	if parsed.hasPath && parsed.hasContent {
		return parsed, fmt.Errorf("script_path and script_content are mutually exclusive")
	}

	// Parse interpreter with default
	parsed.interpreter = "bash"
	if interp, ok := inputs["interpreter"].(string); ok && interp != "" {
		parsed.interpreter = interp
	}

	// Parse timeout with default
	parsed.timeout = 60
	if t, ok := inputs["timeout"].(int); ok {
		parsed.timeout = t
	} else if t, ok := inputs["timeout"].(float64); ok {
		parsed.timeout = int(t)
	}

	// Parse args array
	if args, ok := inputs["args"].([]interface{}); ok && len(args) > 0 {
		for _, arg := range args {
			parsed.args = append(parsed.args, fmt.Sprintf("%v", arg))
		}
	}

	// Parse env map with sanitized values
	if env, ok := inputs["env"].(map[string]interface{}); ok && len(env) > 0 {
		parsed.env = make(map[string]string)
		for k, v := range env {
			// Sanitize value: convert to string and remove dangerous characters
			value := fmt.Sprintf("%v", v)
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

// checkInterpreter verifies the interpreter exists on the system
func (n *ScriptNode) checkInterpreter(interpreter string) (string, error) {
	interpreterPath, err := exec.LookPath(interpreter)
	if err != nil {
		return "", fmt.Errorf("interpreter '%s' not found on system", interpreter)
	}
	return interpreterPath, nil
}

// createTempScript creates a temporary script file from content
func (n *ScriptNode) createTempScript(content, interpreter string) (string, error) {
	// Determine file extension based on interpreter
	ext := ".sh"
	switch interpreter {
	case "python", "python3":
		ext = ".py"
	case "node":
		ext = ".js"
	case "ruby":
		ext = ".rb"
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "waterflow-script-*"+ext)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	scriptFile := tmpFile.Name()

	// Write script content
	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(scriptFile)
		return "", fmt.Errorf("failed to write script: %w", err)
	}
	tmpFile.Close()

	// Set executable permissions
	if err := os.Chmod(scriptFile, 0755); err != nil {
		os.Remove(scriptFile)
		return "", fmt.Errorf("failed to chmod script: %w", err)
	}

	return scriptFile, nil
}

// buildCommand constructs the exec.Cmd with interpreter and script
func (n *ScriptNode) buildCommand(ctx context.Context, interpreterPath, scriptFile string, parsed parsedInputs) *exec.Cmd {
	// Build command: interpreter scriptFile args...
	cmdArgs := []string{scriptFile}
	cmdArgs = append(cmdArgs, parsed.args...)

	cmd := exec.CommandContext(ctx, interpreterPath, cmdArgs...)

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

// Execute executes the script
func (n *ScriptNode) Execute(
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

	// 2. Check interpreter exists
	interpreterPath, err := n.checkInterpreter(parsed.interpreter)
	if err != nil {
		return nil, err
	}
	logs = append(logs, fmt.Sprintf("Using interpreter: %s (%s)", parsed.interpreter, interpreterPath))

	// 3. Prepare script file
	var scriptFile string
	var cleanupFunc func()

	if parsed.hasContent {
		// Create temporary script file
		scriptFile, err = n.createTempScript(parsed.scriptContent, parsed.interpreter)
		if err != nil {
			return nil, err
		}
		cleanupFunc = func() {
			os.Remove(scriptFile)
		}
		defer cleanupFunc()
		logs = append(logs, fmt.Sprintf("Created temp script: %s", scriptFile))
	} else {
		// Use specified script file
		scriptFile = parsed.scriptPath
		// Verify file exists
		if _, err := os.Stat(scriptFile); err != nil {
			return nil, fmt.Errorf("script file not found: %s", scriptFile)
		}
		logs = append(logs, fmt.Sprintf("Executing script file: %s", scriptFile))
	}

	if len(parsed.args) > 0 {
		logs = append(logs, fmt.Sprintf("With args: %v", parsed.args))
	}
	if len(parsed.env) > 0 {
		logs = append(logs, fmt.Sprintf("With env variables: %d", len(parsed.env)))
	}
	if parsed.workdir != "" {
		logs = append(logs, fmt.Sprintf("In directory: %s", parsed.workdir))
	}

	// 4. Create context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, time.Duration(parsed.timeout)*time.Second)
	defer cancel()

	// 5. Build command
	cmd := n.buildCommand(cmdCtx, interpreterPath, scriptFile, parsed)

	// 6. Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 7. Execute command
	err = cmd.Run()
	duration := time.Since(startTime)

	// 8. Build outputs
	outputs := map[string]interface{}{
		"stdout":           stdout.String(),
		"stderr":           stderr.String(),
		"exit_code":        0,
		"duration_ms":      duration.Milliseconds(),
		"interpreter_used": parsed.interpreter,
	}

	// 9. Handle errors
	if err != nil {
		// Timeout error (temporary, retryable)
		if cmdCtx.Err() == context.DeadlineExceeded {
			outputs["exit_code"] = -1
			logs = append(logs, fmt.Sprintf("Script timeout after %d seconds", parsed.timeout))
			if cleanupFunc != nil {
				logs = append(logs, "Temporary script file cleaned up")
			}
			return &node.NodeResult{
				Outputs:  outputs,
				Logs:     logs,
				Duration: duration,
			}, fmt.Errorf("script timeout after %d seconds", parsed.timeout)
		}

		// Context cancellation
		if ctx.Err() == context.Canceled {
			outputs["exit_code"] = -1
			logs = append(logs, "Script cancelled by context")
			return &node.NodeResult{
				Outputs:  outputs,
				Logs:     logs,
				Duration: duration,
			}, fmt.Errorf("script cancelled: %w", ctx.Err())
		}

		// Get exit code from ExitError
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			outputs["exit_code"] = exitCode
			logs = append(logs, fmt.Sprintf("Script failed with exit code %d", exitCode))
			return &node.NodeResult{
				Outputs:  outputs,
				Logs:     logs,
				Duration: duration,
			}, fmt.Errorf("script failed with exit code %d: %s", exitCode, stderr.String())
		}

		// Other errors (permanent, non-retryable)
		outputs["exit_code"] = -1
		logs = append(logs, fmt.Sprintf("Script execution failed: %v", err))
		return &node.NodeResult{
			Outputs:  outputs,
			Logs:     logs,
			Duration: duration,
		}, fmt.Errorf("script execution failed: %w", err)
	}

	// Success
	logs = append(logs, fmt.Sprintf("Script completed successfully in %dms", duration.Milliseconds()))
	return &node.NodeResult{
		Outputs:  outputs,
		Logs:     logs,
		Duration: duration,
	}, nil
}
