//go:build acceptance

package acceptance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// WorkflowSubmitResponse represents the response from workflow submission
type WorkflowSubmitResponse struct {
	WorkflowID string `json:"workflow_id"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

// WorkflowStatus represents the workflow status response
type WorkflowStatus struct {
	WorkflowID string `json:"workflow_id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at"`
	Error      string `json:"error,omitempty"`
	Jobs       []Job  `json:"jobs,omitempty"`
}

// Job represents a job in the workflow
type Job struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Steps  []Step `json:"steps,omitempty"`
}

// Step represents a step in a job
type Step struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// ScenarioResult stores the result of an acceptance test scenario
type ScenarioResult struct {
	Name       string
	Status     string // "passed", "failed", "skipped"
	Duration   time.Duration
	Error      error
	Details    map[string]interface{}
	Logs       string
	WorkflowID string
	StartTime  time.Time
	EndTime    time.Time
}

// getServerURL returns the server URL from environment or default
func getServerURL() string {
	if url := os.Getenv("SERVER_URL"); url != "" {
		return url
	}
	return "http://localhost:18080"
}

// getTemporalHost returns the Temporal host from environment or default
func getTemporalHost() string {
	if host := os.Getenv("TEMPORAL_HOST"); host != "" {
		return host
	}
	return "localhost:17233"
}

// submitWorkflow submits a workflow YAML to the server
func submitWorkflow(ctx context.Context, serverURL, yamlContent string) (*WorkflowSubmitResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		serverURL+"/v1/workflows",
		bytes.NewBufferString(yamlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-yaml")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to submit workflow: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("workflow submission failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result WorkflowSubmitResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// getWorkflowStatus retrieves the current status of a workflow
func getWorkflowStatus(ctx context.Context, serverURL, workflowID string) (*WorkflowStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		serverURL+"/v1/workflows/"+workflowID,
		nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow status: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get status with status %d: %s", resp.StatusCode, string(body))
	}

	var status WorkflowStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("failed to parse status: %w", err)
	}

	return &status, nil
}

// waitForWorkflowCompletion polls the workflow status until it completes or times out
func waitForWorkflowCompletion(ctx context.Context, serverURL, workflowID string, timeout time.Duration) (*WorkflowStatus, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("timeout waiting for workflow %s to complete after %v", workflowID, timeout)
			}

			status, err := getWorkflowStatus(ctx, serverURL, workflowID)
			if err != nil {
				// Retry on transient errors
				continue
			}

			// Check if workflow has reached a terminal state
			switch strings.ToLower(status.Status) {
			case "completed", "succeeded":
				return status, nil
			case "failed":
				return status, nil
			case "cancelled":
				return status, nil
			case "timed_out":
				return status, nil
			}
			// Still running, continue polling
		}
	}
}

// getWorkflowLogs retrieves the logs for a workflow
func getWorkflowLogs(ctx context.Context, serverURL, workflowID string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		serverURL+"/v1/workflows/"+workflowID+"/logs",
		nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get logs with status %d: %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

// loadTestWorkflow loads a test workflow from the testdata directory
func loadTestWorkflow(filename string) (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get current file path")
	}

	testdataDir := filepath.Join(filepath.Dir(currentFile), "testdata", "workflows")
	filePath := filepath.Join(testdataDir, filename)

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read test workflow %s: %w", filename, err)
	}

	return string(content), nil
}

// isTerminalStatus checks if a workflow status is terminal
func isTerminalStatus(status string) bool {
	switch strings.ToLower(status) {
	case "completed", "succeeded", "failed", "cancelled", "timed_out":
		return true
	default:
		return false
	}
}

// isSuccessStatus checks if a workflow status indicates success
func isSuccessStatus(status string) bool {
	switch strings.ToLower(status) {
	case "completed", "succeeded":
		return true
	default:
		return false
	}
}

// checkServerHealth verifies the server is healthy
func checkServerHealth(ctx context.Context, serverURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL+"/health", nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}
