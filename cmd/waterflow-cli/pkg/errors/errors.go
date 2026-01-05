// Package errors provides CLI-specific error types and handling
package errors

import (
	"fmt"
)

// Exit codes
const (
	ExitSuccess      = 0
	ExitGeneralError = 1
	ExitUsageError   = 2
)

// CLIError represents a CLI error with exit code and suggestions
type CLIError struct {
	Code       int
	Message    string
	Cause      error
	Suggestion string
}

// Error implements the error interface
func (e *CLIError) Error() string {
	msg := fmt.Sprintf("Error: %s\n", e.Message)
	if e.Cause != nil {
		msg += fmt.Sprintf("  Cause: %v\n", e.Cause)
	}
	if e.Suggestion != "" {
		msg += fmt.Sprintf("\nSuggestion: %s\n", e.Suggestion)
	}
	return msg
}

// Unwrap returns the wrapped error
func (e *CLIError) Unwrap() error {
	return e.Cause
}

// NewConfigError creates an error for configuration issues
func NewConfigError(cause error, configFile string) *CLIError {
	return &CLIError{
		Code:       ExitGeneralError,
		Message:    fmt.Sprintf("Failed to load config file: %s", configFile),
		Cause:      cause,
		Suggestion: "Check YAML syntax in config file or use --config to specify a different file",
	}
}

// NewConnectionError creates an error for network connection issues
func NewConnectionError(url string, cause error) *CLIError {
	return &CLIError{
		Code:    ExitGeneralError,
		Message: "Failed to connect to Waterflow server",
		Cause:   fmt.Errorf("URL: %s, %w", url, cause),
		Suggestion: `1. Check if Waterflow server is running
  2. Verify server URL in config or use --server flag
  3. Check network connectivity`,
	}
}

// NewAuthError creates an error for authentication failures
func NewAuthError() *CLIError {
	return &CLIError{
		Code:       ExitGeneralError,
		Message:    "API authentication failed",
		Suggestion: "Set API key using --api-key flag or WATERFLOW_API_KEY environment variable",
	}
}

// NewValidationError creates an error for workflow validation failures
func NewValidationError(file string, cause error) *CLIError {
	return &CLIError{
		Code:       ExitGeneralError,
		Message:    fmt.Sprintf("Workflow validation failed: %s", file),
		Cause:      cause,
		Suggestion: "Run 'waterflow node list' to see available nodes",
	}
}

// NewUsageError creates an error for incorrect command usage
func NewUsageError(message string) *CLIError {
	return &CLIError{
		Code:       ExitUsageError,
		Message:    message,
		Suggestion: "Run command with --help for usage information",
	}
}
