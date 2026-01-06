// Package client provides HTTP client for Waterflow server
// NOTE: This is a temporary implementation for CLI framework validation.
//
//	Story 5.7 will implement production-grade Go SDK (pkg/sdk/client.go)
//	which will replace this temporary client.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Client is a temporary HTTP client for Waterflow server
// TODO: Replace with pkg/sdk/client.go in Story 5.7
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	debug      bool
}

// New creates a new client instance
func New(baseURL, apiKey string, timeout time.Duration, debug bool) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		debug: debug,
	}
}

// NOTE: The generic `do` method below is reserved for future use in Story 5.7 (Go SDK).
// Currently, specific methods (SubmitWorkflow, GetWorkflowStatus) are used directly.
// This method will be utilized once the comprehensive SDK client is implemented.

// do performs an HTTP request (reserved for Story 5.7)
//
//nolint:unused // Reserved for Story 5.7 SDK implementation
func (c *Client) do(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	if c.debug {
		fmt.Printf("DEBUG: %s %s\n", method, url)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("server error (status %d): %v", resp.StatusCode, errResp)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// SubmitWorkflowRequest represents a workflow submission request
type SubmitWorkflowRequest struct {
	YAML string                 `json:"yaml"`
	Vars map[string]interface{} `json:"vars,omitempty"`
}

// SubmitWorkflowResult represents the workflow submission result
type SubmitWorkflowResult struct {
	ID        string    `json:"id"`
	RunID     string    `json:"run_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	URL       string    `json:"url"`
}

// WorkflowStatus represents workflow status information
// Updated in Story 5.4 for complete status command support
type WorkflowStatus struct {
	ID              string                 `json:"id"`
	RunID           string                 `json:"run_id"`
	Name            string                 `json:"name"`
	Status          string                 `json:"status"` // pending, running, completed, failed, cancelled, timeout
	CreatedAt       string                 `json:"created_at"`
	StartedAt       string                 `json:"started_at,omitempty"`
	CompletedAt     string                 `json:"completed_at,omitempty"`
	DurationSeconds *int                   `json:"duration_seconds,omitempty"`
	Vars            map[string]interface{} `json:"vars,omitempty"`
	Jobs            []JobStatus            `json:"jobs,omitempty"`
	Error           string                 `json:"error,omitempty"` // 失败时的错误信息
}

// JobStatus represents job execution status
type JobStatus struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Status      string       `json:"status"`
	StartedAt   string       `json:"started_at,omitempty"`
	CompletedAt string       `json:"completed_at,omitempty"`
	RunsOn      string       `json:"runs_on,omitempty"`
	Steps       []StepStatus `json:"steps,omitempty"`
}

// StepStatus represents step execution status
type StepStatus struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Conclusion  string `json:"conclusion,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
}

// ServerError represents an error from the server
type ServerError struct {
	StatusCode int
	Code       string
	Message    string
	Details    map[string]interface{}
}

func (e *ServerError) Error() string {
	return fmt.Sprintf("server error (status %d): %s - %s", e.StatusCode, e.Code, e.Message)
}

// SubmitWorkflow submits a workflow to the server
// IMPORTANT: This implementation uses direct HTTP calls
// Story 5.7 will replace this with the Go SDK
func (c *Client) SubmitWorkflow(ctx context.Context, yamlContent string, vars map[string]interface{}) (*SubmitWorkflowResult, error) {
	req := SubmitWorkflowRequest{
		YAML: yamlContent,
		Vars: vars,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/v1/workflows",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	if c.debug {
		fmt.Printf("DEBUG: POST %s/v1/workflows\n", c.baseURL)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result SubmitWorkflowResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetWorkflowStatus gets the current status of a workflow
// Updated in Story 5.4 for complete status command with Jobs/Steps details
func (c *Client) GetWorkflowStatus(workflowID string) (*WorkflowStatus, error) {
	httpReq, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("%s/v1/workflows/%s", c.baseURL, workflowID),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	if c.debug {
		fmt.Printf("DEBUG: GET %s/v1/workflows/%s\n", c.baseURL, workflowID)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var status WorkflowStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &status, nil
}

// parseError parses server error responses
func (c *Client) parseError(resp *http.Response) error {
	var errResp struct {
		Error struct {
			Code    string                 `json:"code"`
			Message string                 `json:"message"`
			Details map[string]interface{} `json:"details"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return fmt.Errorf("server error (status %d)", resp.StatusCode)
	}

	return &ServerError{
		StatusCode: resp.StatusCode,
		Code:       errResp.Error.Code,
		Message:    errResp.Error.Message,
		Details:    errResp.Error.Details,
	}
}

// LogEntry represents a single workflow log entry
// Added in Story 5.5 for logs command
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Job       string `json:"job,omitempty"`
	Step      string `json:"step,omitempty"`
	Message   string `json:"message"`
	Error     string `json:"error,omitempty"`
}

// LogsQuery represents query parameters for logs
type LogsQuery struct {
	Tail  int    // Number of lines (1-1000, 0 for all)
	Level string // Filter by level (info,warn,error,debug)
	Job   string // Filter by job name
	Step  string // Filter by step name
}

// GetWorkflowLogs retrieves logs for a workflow
// Server returns NDJSON format (newline-delimited JSON), one log entry per line
// Client decodes NDJSON stream and returns as slice of LogEntry
// Added in Story 5.5 for logs command
func (c *Client) GetWorkflowLogs(workflowID string, query LogsQuery) ([]LogEntry, error) {
	url := fmt.Sprintf("%s/v1/workflows/%s/logs", c.baseURL, workflowID)

	// 构建查询参数
	params := []string{}
	if query.Tail > 0 {
		params = append(params, fmt.Sprintf("tail=%d", query.Tail))
	}
	if query.Level != "" {
		params = append(params, fmt.Sprintf("level=%s", query.Level))
	}
	if query.Job != "" {
		params = append(params, fmt.Sprintf("job=%s", query.Job))
	}
	if query.Step != "" {
		params = append(params, fmt.Sprintf("step=%s", query.Step))
	}

	if len(params) > 0 {
		url += "?" + joinParams(params)
	}

	httpReq, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	if c.debug {
		fmt.Printf("DEBUG: GET %s\n", url)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	// Decode NDJSON stream (each line is a separate JSON object)
	logs := make([]LogEntry, 0)
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

// StreamWorkflowLogs streams logs in real-time using Server-Sent Events (SSE)
// Returns channels for log entries and errors
// Used for --follow mode in logs command (Story 5.5 AC4)
func (c *Client) StreamWorkflowLogs(workflowID string, query LogsQuery) (<-chan LogEntry, <-chan error, error) {
	// 构建查询参数 (添加 stream=true)
	params := []string{"stream=true"}
	if query.Level != "" {
		params = append(params, fmt.Sprintf("level=%s", query.Level))
	}
	if query.Job != "" {
		params = append(params, fmt.Sprintf("job=%s", query.Job))
	}
	if query.Step != "" {
		params = append(params, fmt.Sprintf("step=%s", query.Step))
	}

	url := fmt.Sprintf("%s/v1/workflows/%s/logs?%s", c.baseURL, workflowID, joinParams(params))

	httpReq, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Accept", "text/event-stream")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	if c.debug {
		fmt.Printf("DEBUG: GET %s (SSE)\n", url)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}

	// Check if SSE is supported (server returns 200 with text/event-stream)
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		// If server doesn't support SSE, return error to trigger polling fallback
		if resp.StatusCode == http.StatusNotImplemented || resp.StatusCode == http.StatusMethodNotAllowed {
			return nil, nil, fmt.Errorf("SSE not supported by server")
		}
		return nil, nil, c.parseError(resp)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/event-stream" && contentType != "application/x-ndjson" {
		_ = resp.Body.Close()
		return nil, nil, fmt.Errorf("SSE not supported: unexpected content-type %s", contentType)
	}

	logChan := make(chan LogEntry, 10)
	errChan := make(chan error, 1)

	// Start goroutine to read SSE stream
	go func() {
		defer func() { _ = resp.Body.Close() }()
		defer close(logChan)
		defer close(errChan)

		decoder := json.NewDecoder(resp.Body)
		for decoder.More() {
			var entry LogEntry
			if err := decoder.Decode(&entry); err != nil {
				if c.debug {
					fmt.Fprintf(os.Stderr, "WARN: Invalid log entry, skipping\n")
				}
				continue
			}
			logChan <- entry
		}
	}()

	return logChan, errChan, nil
}

func joinParams(params []string) string {
	result := ""
	for i, p := range params {
		if i > 0 {
			result += "&"
		}
		result += p
	}
	return result
}

// Specific methods will be added in subsequent stories:
// - ValidateWorkflow (Story 5.2)
// - ListNodes (Story 5.6)

// NodeInfo represents node metadata
type NodeInfo struct {
	Name         string                            `json:"name"`
	Version      string                            `json:"version"`
	Category     string                            `json:"category"`
	Description  string                            `json:"description"`
	InputSchema  map[string]map[string]interface{} `json:"input_schema"`
	OutputSchema map[string]interface{}            `json:"output_schema"`
}

// ListNodes retrieves list of available nodes
// Added in Story 5.6 for node list command
func (c *Client) ListNodes(category, search string) ([]NodeInfo, error) {
	url := fmt.Sprintf("%s/v1/nodes", c.baseURL)

	// Build query parameters
	params := []string{}
	if category != "" {
		params = append(params, fmt.Sprintf("category=%s", category))
	}
	if search != "" {
		params = append(params, fmt.Sprintf("search=%s", search))
	}

	if len(params) > 0 {
		url += "?" + joinParams(params)
	}

	httpReq, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	if c.debug {
		fmt.Printf("DEBUG: GET %s\n", url)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result struct {
		Nodes []NodeInfo `json:"nodes"`
		Total int        `json:"total"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Nodes, nil
}
