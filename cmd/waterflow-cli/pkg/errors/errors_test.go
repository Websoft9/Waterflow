package errors

import (
	"errors"
	"testing"
)

func TestCLIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *CLIError
		contains []string
	}{
		{
			name: "error with cause and suggestion",
			err: &CLIError{
				Code:       ExitGeneralError,
				Message:    "test error",
				Cause:      errors.New("underlying error"),
				Suggestion: "try this fix",
			},
			contains: []string{"Error: test error", "Cause: underlying error", "Suggestion: try this fix"},
		},
		{
			name: "error without cause",
			err: &CLIError{
				Code:       ExitUsageError,
				Message:    "usage error",
				Suggestion: "check usage",
			},
			contains: []string{"Error: usage error", "Suggestion: check usage"},
		},
		{
			name: "error without suggestion",
			err: &CLIError{
				Code:    ExitGeneralError,
				Message: "simple error",
				Cause:   errors.New("cause"),
			},
			contains: []string{"Error: simple error", "Cause: cause"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, substr := range tt.contains {
				if !contains(errStr, substr) {
					t.Errorf("Error() string should contain %q, got: %s", substr, errStr)
				}
			}
		})
	}
}

func TestCLIError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &CLIError{
		Code:    ExitGeneralError,
		Message: "wrapper",
		Cause:   cause,
	}

	if unwrapped := err.Unwrap(); unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}

	errNoCause := &CLIError{
		Code:    ExitGeneralError,
		Message: "no cause",
	}
	if unwrapped := errNoCause.Unwrap(); unwrapped != nil {
		t.Errorf("Unwrap() with no cause = %v, want nil", unwrapped)
	}
}

func TestNewConfigError(t *testing.T) {
	cause := errors.New("invalid YAML")
	err := NewConfigError(cause, "/path/to/config.yaml")

	if err.Code != ExitGeneralError {
		t.Errorf("Code = %d, want %d", err.Code, ExitGeneralError)
	}
	if !contains(err.Message, "config.yaml") {
		t.Errorf("Message should contain config file path")
	}
	if err.Cause == nil {
		t.Error("Cause should not be nil")
	}
	if err.Suggestion == "" {
		t.Error("Suggestion should not be empty")
	}
}

func TestNewConnectionError(t *testing.T) {
	cause := errors.New("connection refused")
	url := "http://localhost:8080"
	err := NewConnectionError(url, cause)

	if err.Code != ExitGeneralError {
		t.Errorf("Code = %d, want %d", err.Code, ExitGeneralError)
	}
	if !contains(err.Message, "connect") {
		t.Errorf("Message should mention connection")
	}
	if !contains(err.Cause.Error(), url) {
		t.Errorf("Cause should contain URL %s", url)
	}
	if err.Suggestion == "" {
		t.Error("Suggestion should not be empty")
	}
}

func TestNewAuthError(t *testing.T) {
	err := NewAuthError()

	if err.Code != ExitGeneralError {
		t.Errorf("Code = %d, want %d", err.Code, ExitGeneralError)
	}
	if !contains(err.Message, "authentication") {
		t.Errorf("Message should mention authentication")
	}
	if err.Suggestion == "" {
		t.Error("Suggestion should not be empty")
	}
}

func TestNewValidationError(t *testing.T) {
	cause := errors.New("invalid node")
	file := "workflow.yaml"
	err := NewValidationError(file, cause)

	if err.Code != ExitGeneralError {
		t.Errorf("Code = %d, want %d", err.Code, ExitGeneralError)
	}
	if !contains(err.Message, file) {
		t.Errorf("Message should contain file name")
	}
	if err.Cause != cause {
		t.Errorf("Cause = %v, want %v", err.Cause, cause)
	}
	if err.Suggestion == "" {
		t.Error("Suggestion should not be empty")
	}
}

func TestNewUsageError(t *testing.T) {
	message := "missing required argument"
	err := NewUsageError(message)

	if err.Code != ExitUsageError {
		t.Errorf("Code = %d, want %d", err.Code, ExitUsageError)
	}
	if err.Message != message {
		t.Errorf("Message = %q, want %q", err.Message, message)
	}
	if err.Suggestion == "" {
		t.Error("Suggestion should not be empty")
	}
}

func TestExitCodes(t *testing.T) {
	if ExitSuccess != 0 {
		t.Errorf("ExitSuccess = %d, want 0", ExitSuccess)
	}
	if ExitGeneralError != 1 {
		t.Errorf("ExitGeneralError = %d, want 1", ExitGeneralError)
	}
	if ExitUsageError != 2 {
		t.Errorf("ExitUsageError = %d, want 2", ExitUsageError)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
