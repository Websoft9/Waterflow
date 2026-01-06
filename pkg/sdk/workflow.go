package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// SubmitWorkflow submits a workflow for execution.
//
// The workflow YAML content is sent to the Waterflow server for validation and execution.
// On success, returns a SubmitWorkflowResponse containing the workflow ID and metadata.
//
// Example:
//
//	yamlContent := `
//	name: deploy-app
//	jobs:
//	  build:
//	    runs-on: default
//	    steps:
//	      - name: Build
//	        uses: exec/shell@v1
//	        with:
//	          command: make build
//	`
//
//	resp, err := client.SubmitWorkflow(ctx, &sdk.SubmitWorkflowRequest{
//	    YAML: yamlContent,
//	})
//	if err != nil {
//	    if sdk.IsValidationError(err) {
//	        fmt.Println("Invalid YAML syntax")
//	    }
//	    return err
//	}
//
//	fmt.Printf("Workflow ID: %s\n", resp.ID)
//
// Returns:
//   - ServerError if YAML validation fails or server error occurs
//   - context.DeadlineExceeded if request times out
func (c *Client) SubmitWorkflow(ctx context.Context, req *SubmitWorkflowRequest) (*SubmitWorkflowResponse, error) {
	if req == nil || req.YAML == "" {
		return nil, fmt.Errorf("YAML content is required")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var result SubmitWorkflowResponse
	if err := c.doRequest(ctx, http.MethodPost, "/v1/workflows", bytes.NewReader(body), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetWorkflowStatus retrieves the current status of a workflow.
//
// Returns detailed workflow information including status, jobs, steps, and execution times.
// The Jobs field is an array that should be iterated using range.
//
// Example:
//
//	status, err := client.GetWorkflowStatus(ctx, workflowID)
//	if err != nil {
//	    if sdk.IsNotFound(err) {
//	        fmt.Println("Workflow not found")
//	        return
//	    }
//	    return err
//	}
//
//	fmt.Printf("Status: %s\n", status.Status)
//	for _, job := range status.Jobs {
//	    fmt.Printf("Job %s: %s\n", job.Name, job.Status)
//	    for _, step := range job.Steps {
//	        fmt.Printf("  Step %s: %s\n", step.Name, step.Status)
//	    }
//	}
//
// Returns:
//   - ServerError with StatusCode 404 if workflow not found
//   - context.DeadlineExceeded if request times out
func (c *Client) GetWorkflowStatus(ctx context.Context, workflowID string) (*WorkflowStatus, error) {
	if workflowID == "" {
		return nil, fmt.Errorf("workflow ID is required")
	}

	path := fmt.Sprintf("/v1/workflows/%s", workflowID)

	var result WorkflowStatus
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CancelWorkflow cancels a running workflow.
//
// Sends a cancellation request to the server. The workflow will be marked as cancelled
// and no further steps will be executed. Already running steps may complete or be interrupted.
//
// Example:
//
//	err := client.CancelWorkflow(ctx, workflowID)
//	if err != nil {
//	    if sdk.IsNotFound(err) {
//	        fmt.Println("Workflow not found")
//	        return
//	    }
//	    return err
//	}
//
//	fmt.Println("Workflow cancelled successfully")
//
// Returns:
//   - ServerError with StatusCode 404 if workflow not found
//   - ServerError with StatusCode 400 if workflow already completed
func (c *Client) CancelWorkflow(ctx context.Context, workflowID string) error {
	if workflowID == "" {
		return fmt.Errorf("workflow ID is required")
	}

	path := fmt.Sprintf("/v1/workflows/%s/cancel", workflowID)
	return c.doRequest(ctx, http.MethodPost, path, nil, nil)
}

// GetWorkflowLogs retrieves logs for a workflow with optional filtering.
//
// Returns log entries in chronological order. The API returns logs in NDJSON
// (newline-delimited JSON) format which is automatically parsed.
//
// Example:
//
//	logs, err := client.GetWorkflowLogs(ctx, &sdk.GetLogsRequest{
//	    WorkflowID: workflowID,
//	    Level:      "error,warn",
//	    Job:        "deploy",
//	    Tail:       100,
//	})
//	if err != nil {
//	    return err
//	}
//
//	for _, log := range logs {
//	    fmt.Printf("[%s] %s: %s\n", log.Level, log.Timestamp, log.Message)
//	    if log.Job != "" {
//	        fmt.Printf("  Job: %s, Step: %s\n", log.Job, log.Step)
//	    }
//	}
//
// Returns:
//   - []LogEntry: Slice of log entries in chronological order
//   - ServerError with StatusCode 404 if workflow not found
func (c *Client) GetWorkflowLogs(ctx context.Context, req *GetLogsRequest) ([]LogEntry, error) {
	if req == nil || req.WorkflowID == "" {
		return nil, fmt.Errorf("workflow ID is required")
	}

	path := fmt.Sprintf("/v1/workflows/%s/logs", req.WorkflowID)

	// Build query parameters
	queryParams := []string{}
	if req.Level != "" {
		queryParams = append(queryParams, fmt.Sprintf("level=%s", req.Level))
	}
	if req.Job != "" {
		queryParams = append(queryParams, fmt.Sprintf("job=%s", req.Job))
	}
	if req.Step != "" {
		queryParams = append(queryParams, fmt.Sprintf("step=%s", req.Step))
	}
	if req.Tail > 0 {
		queryParams = append(queryParams, fmt.Sprintf("tail=%d", req.Tail))
	}

	if len(queryParams) > 0 {
		path += "?" + queryParams[0]
		for i := 1; i < len(queryParams); i++ {
			path += "&" + queryParams[i]
		}
	}

	// Note: API returns NDJSON format
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	// Parse NDJSON (newline-delimited JSON)
	var logs []LogEntry
	decoder := json.NewDecoder(resp.Body)
	for decoder.More() {
		var log LogEntry
		if err := decoder.Decode(&log); err != nil {
			// Skip invalid lines
			continue
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// ListWorkflows retrieves a paginated list of workflows.
//
// Supports filtering by status and name, as well as pagination control.
//
// Example:
//
//	resp, err := client.ListWorkflows(ctx, &sdk.ListWorkflowsRequest{
//	    Page:   1,
//	    Limit:  20,
//	    Status: "running",
//	    Name:   "deploy",
//	})
//	if err != nil {
//	    return err
//	}
//
//	fmt.Printf("Total workflows: %d\n", resp.Pagination.Total)
//	for _, wf := range resp.Workflows {
//	    fmt.Printf("- %s: %s\n", wf.Name, wf.Status)
//	}
//
// Returns:
//   - ListWorkflowsResponse: Paginated workflow list with metadata
//   - ServerError if server error occurs
func (c *Client) ListWorkflows(ctx context.Context, req *ListWorkflowsRequest) (*ListWorkflowsResponse, error) {
	if req == nil {
		req = &ListWorkflowsRequest{} // Use defaults
	}

	path := "/v1/workflows"

	// Build query parameters
	queryParams := []string{}
	if req.Page > 0 {
		queryParams = append(queryParams, fmt.Sprintf("page=%d", req.Page))
	}
	if req.Limit > 0 {
		queryParams = append(queryParams, fmt.Sprintf("limit=%d", req.Limit))
	}
	if req.Status != "" {
		queryParams = append(queryParams, fmt.Sprintf("status=%s", req.Status))
	}
	if req.Name != "" {
		queryParams = append(queryParams, fmt.Sprintf("name=%s", req.Name))
	}

	if len(queryParams) > 0 {
		path += "?" + queryParams[0]
		for i := 1; i < len(queryParams); i++ {
			path += "&" + queryParams[i]
		}
	}

	var result ListWorkflowsResponse
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// RerunWorkflow reruns a completed or failed workflow with optional variable overrides.
//
// Submits the same workflow definition with optional new variable values.
// Returns a new workflow ID for the rerun execution.
//
// Example:
//
//	resp, err := client.RerunWorkflow(ctx, &sdk.RerunWorkflowRequest{
//	    WorkflowID: originalWorkflowID,
//	    Vars: map[string]interface{}{
//	        "environment": "staging",
//	        "timeout":     600,
//	    },
//	})
//	if err != nil {
//	    return err
//	}
//
//	fmt.Printf("Workflow rerun: %s\n", resp.ID)
//
// Returns:
//   - SubmitWorkflowResponse: New workflow execution metadata
//   - ServerError with StatusCode 404 if original workflow not found
func (c *Client) RerunWorkflow(ctx context.Context, req *RerunWorkflowRequest) (*SubmitWorkflowResponse, error) {
	if req == nil || req.WorkflowID == "" {
		return nil, fmt.Errorf("workflow ID is required")
	}

	path := fmt.Sprintf("/v1/workflows/%s/rerun", req.WorkflowID)

	var body []byte
	var err error
	if len(req.Vars) > 0 {
		body, err = json.Marshal(map[string]interface{}{"vars": req.Vars})
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	var result SubmitWorkflowResponse
	if err := c.doRequest(ctx, http.MethodPost, path, bytes.NewReader(body), &result); err != nil {
		return nil, err
	}

	return &result, nil
}
