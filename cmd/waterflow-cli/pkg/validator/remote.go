package validator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
)

// RemoteValidator validates workflows using Server API
type RemoteValidator struct {
	client    *http.Client
	serverURL string
	logger    *zap.Logger
}

// NewRemoteValidator creates a new remote validator
func NewRemoteValidator(client *http.Client, serverURL string, logger *zap.Logger) *RemoteValidator {
	return &RemoteValidator{
		client:    client,
		serverURL: serverURL,
		logger:    logger,
	}
}

// Validate validates a workflow file using Server API
func (v *RemoteValidator) Validate(filepath string) (*ValidationResult, error) {
	start := time.Now()

	// Read file
	content, err := os.ReadFile(filepath) //nolint:gosec // File path from user input
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", filepath)
		}
		if os.IsPermission(err) {
			return nil, fmt.Errorf("permission denied: %s", filepath)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Call Server API (using submit endpoint with dry-run mode)
	// Note: Story 1.9 does not implement standalone /v1/workflows/validate endpoint
	// Using POST /v1/workflows with dry_run parameter instead
	reqBody := map[string]interface{}{
		"yaml": string(content),
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/workflows?dry_run=true", v.serverURL)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := v.client.Do(req)
	if err != nil {
		v.logger.Debug("Server request failed", zap.Error(err))
		// Return network error for fallback handling
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	duration := time.Since(start)

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp map[string]interface{}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Parse response
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		// Validation successful
		result := &ValidationResult{
			File:           filepath,
			Valid:          true,
			Remote:         true,
			ValidationTime: duration,
		}

		// Extract workflow info if available
		if workflow, ok := apiResp["workflow"].(map[string]interface{}); ok {
			if name, ok := workflow["name"].(string); ok {
				result.WorkflowName = name
			}
			if jobs, ok := workflow["jobs"].(map[string]interface{}); ok {
				result.JobsCount = len(jobs)
			}
		}

		return result, nil
	}

	// Validation failed
	return v.parseServerErrors(filepath, apiResp, duration), nil
}

// parseServerErrors parses Server error response
func (v *RemoteValidator) parseServerErrors(filepath string, resp map[string]interface{}, duration time.Duration) *ValidationResult {
	result := &ValidationResult{
		File:           filepath,
		Valid:          false,
		Remote:         true,
		ValidationTime: duration,
	}

	// Parse RFC 7807 error response
	if detail, ok := resp["detail"].(string); ok {
		// Check for errors array
		if errors, ok := resp["errors"].([]interface{}); ok {
			result.Errors = make([]ValidationError, len(errors))
			for i, e := range errors {
				if errMap, ok := e.(map[string]interface{}); ok {
					result.Errors[i] = ValidationError{
						Type:       getString(errMap, "type"),
						Field:      getString(errMap, "field"),
						Line:       getInt(errMap, "line"),
						Message:    getString(errMap, "error"),
						Suggestion: getString(errMap, "suggestion"),
					}
				}
			}
		} else {
			// Single error message
			result.Errors = []ValidationError{{
				Type:    "server_error",
				Message: detail,
			}}
		}
	} else if msg, ok := resp["message"].(string); ok {
		// Alternative error format
		result.Errors = []ValidationError{{
			Type:    "server_error",
			Message: msg,
		}}
	} else {
		// Unknown error format
		result.Errors = []ValidationError{{
			Type:    "server_error",
			Message: "Unknown server error",
		}}
	}

	return result
}

// getString safely extracts string from map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

// getInt safely extracts int from map
func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return 0
}
