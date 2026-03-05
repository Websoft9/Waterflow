//go:build integration

package integration

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
	"testing"
	"time"
)

// WorkflowSubmitResponse represents the response from workflow submission
type WorkflowSubmitResponse struct {
	ID        string `json:"id"`         // API returns "id" not "workflow_id"
	RunID     string `json:"run_id"`     // Temporal run ID
	Name      string `json:"name"`       // Workflow name
	Status    string `json:"status"`     // Workflow status
	CreatedAt string `json:"created_at"` // Creation timestamp
	URL       string `json:"url"`        // Workflow URL
	// Legacy field for compatibility
	WorkflowID string `json:"workflow_id,omitempty"` // Deprecated: use ID
	Message    string `json:"message,omitempty"`     // Optional message
}

// GetID returns the workflow ID, supporting both new (id) and legacy (workflow_id) fields
func (r *WorkflowSubmitResponse) GetID() string {
	if r.ID != "" {
		return r.ID
	}
	return r.WorkflowID
}

// WorkflowStatus represents the workflow status response
type WorkflowStatus struct {
	WorkflowID string `json:"workflow_id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at"`
	Error      string `json:"error,omitempty"`
}

// getServerURL returns the server URL from environment or default
// Supports both WATERFLOW_TEST_URL (preferred) and SERVER_URL (legacy)
func getServerURL() string {
	if url := os.Getenv("WATERFLOW_TEST_URL"); url != "" {
		return url
	}
	return getEnvOrDefault("SERVER_URL", "http://localhost:18080")
}

// getAgentURL returns the agent URL from environment or default
func getAgentURL() string {
	return getEnvOrDefault("WATERFLOW_AGENT_URL", "http://localhost:18081")
}

// getEnvOrDefault returns the environment variable value or a default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// submitWorkflowRequest represents the JSON request body for workflow submission
type submitWorkflowRequestBody struct {
	YAML string                 `json:"yaml"`
	Vars map[string]interface{} `json:"vars,omitempty"`
}

// submitWorkflow submits a workflow YAML to the server
func submitWorkflow(ctx context.Context, serverURL, yamlContent string) (*WorkflowSubmitResponse, error) {
	// API expects JSON with yaml field, not raw YAML
	reqBody := submitWorkflowRequestBody{YAML: yamlContent}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		serverURL+"/v1/workflows",
		bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

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
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("timeout waiting for workflow %s to complete", workflowID)
			}

			status, err := getWorkflowStatus(ctx, serverURL, workflowID)
			if err != nil {
				// Retry on transient errors
				continue
			}

			// Check if workflow has reached a terminal state
			switch status.Status {
			case "completed", "Completed", "COMPLETED":
				return status, nil
			case "failed", "Failed", "FAILED":
				return status, nil
			case "cancelled", "Cancelled", "CANCELLED":
				return status, nil
			case "timed_out", "TimedOut", "TIMED_OUT":
				return status, nil
			}
			// Still running, continue polling
		}
	}
}

// cancelWorkflow cancels a running workflow
func cancelWorkflow(ctx context.Context, serverURL, workflowID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		serverURL+"/v1/workflows/"+workflowID+"/cancel",
		nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to cancel workflow: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cancel failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
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

// rerunWorkflow triggers a rerun of a completed workflow
func rerunWorkflow(ctx context.Context, serverURL, workflowID string) (*WorkflowSubmitResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		serverURL+"/v1/workflows/"+workflowID+"/rerun",
		nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to rerun workflow: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("rerun failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result WorkflowSubmitResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// waitForWorkflowRunning polls until the workflow is in running state
func waitForWorkflowRunning(ctx context.Context, serverURL, workflowID string, timeout time.Duration) (*WorkflowStatus, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("timeout waiting for workflow %s to start running", workflowID)
			}

			status, err := getWorkflowStatus(ctx, serverURL, workflowID)
			if err != nil {
				continue
			}

			switch status.Status {
			case "running", "Running", "RUNNING":
				return status, nil
			case "completed", "Completed", "COMPLETED",
				"failed", "Failed", "FAILED",
				"cancelled", "Cancelled", "CANCELLED":
				// Already finished
				return status, nil
			}
		}
	}
}

// skipDatabaseTests checks if database tests should be skipped
func skipDatabaseTests(t *testing.T) bool {
	if getEnvOrDefault("SKIP_DATABASE_TESTS", "") == "true" {
		t.Skip("Database tests disabled (SKIP_DATABASE_TESTS=true)")
		return true
	}
	return false
}

// skipTemporalTests checks if Temporal tests should be skipped
func skipTemporalTests(t *testing.T) bool {
	if getEnvOrDefault("SKIP_TEMPORAL_TESTS", "") == "true" {
		t.Skip("Temporal tests disabled (SKIP_TEMPORAL_TESTS=true)")
		return true
	}
	return false
}
