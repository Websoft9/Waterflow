// Package sdk provides Go SDK for Waterflow workflow orchestration engine.
//
// The Waterflow SDK allows Go applications to submit workflows, query execution
// status, retrieve logs, and control workflow execution through a simple API.
//
// # Quick Start
//
// Create a client and submit a workflow:
//
//	client, err := sdk.NewClient(&sdk.ClientConfig{
//	    ServerURL: "http://localhost:8080",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	yamlContent := `
//	name: hello-world
//	jobs:
//	  greet:
//	    runs-on: default
//	    steps:
//	      - name: Say hello
//	        uses: exec/shell@v1
//	        with:
//	          command: echo "Hello from Waterflow SDK"
//	`
//
//	resp, err := client.SubmitWorkflow(ctx, &sdk.SubmitWorkflowRequest{
//	    YAML: yamlContent,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Workflow ID: %s\n", resp.ID)
//
// # Configuration
//
// The client can be configured using ClientConfig:
//
//	client, err := sdk.NewClient(&sdk.ClientConfig{
//	    ServerURL:  "https://waterflow.example.com",
//	    APIKey:     "your-api-key",
//	    Timeout:    60 * time.Second,
//	})
//
// Or using environment variables:
//
//	export WATERFLOW_SERVER_URL=http://localhost:8080
//	export WATERFLOW_API_KEY=your-api-key
//
//	client, err := sdk.NewDefaultClient()
//
// # Error Handling
//
// The SDK provides typed errors for different failure scenarios:
//
//	_, err := client.SubmitWorkflow(ctx, req)
//	if err != nil {
//	    if sdk.IsValidationError(err) {
//	        fmt.Println("Validation failed")
//	    } else if sdk.IsNotFound(err) {
//	        fmt.Println("Resource not found")
//	    } else {
//	        fmt.Printf("Error: %v\n", err)
//	    }
//	}
//
// # Context and Timeouts
//
// All methods accept context.Context for timeout and cancellation:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	status, err := client.GetWorkflowStatus(ctx, workflowID)
//
// For more examples, see https://github.com/Websoft9/waterflow/tree/main/examples/sdk
package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client represents Waterflow API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// ClientConfig contains client configuration
type ClientConfig struct {
	ServerURL  string        // Waterflow Server URL (required)
	APIKey     string        // API Key for authentication (optional)
	Timeout    time.Duration // Request timeout (default: 30s)
	HTTPClient *http.Client  // Custom HTTP client (optional)
}

// NewClient creates a new Waterflow client with the provided configuration.
//
// The client is safe for concurrent use by multiple goroutines and should be reused
// across your application rather than creating new clients for each request.
//
// Example:
//
//	client, err := sdk.NewClient(&sdk.ClientConfig{
//	    ServerURL: "http://localhost:8080",
//	    APIKey:    "your-api-key",
//	    Timeout:   60 * time.Second,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// For production use with custom HTTP client:
//
//	httpClient := &http.Client{
//	    Timeout: 60 * time.Second,
//	    Transport: &http.Transport{
//	        MaxIdleConns:        100,
//	        MaxIdleConnsPerHost: 10,
//	    },
//	}
//
//	client, err := sdk.NewClient(&sdk.ClientConfig{
//	    ServerURL:  "https://waterflow.prod.example.com",
//	    APIKey:     os.Getenv("WATERFLOW_API_KEY"),
//	    HTTPClient: httpClient,
//	})
//
// Returns an error if cfg is nil or ServerURL is empty.
func NewClient(cfg *ClientConfig) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("client config is required")
	}

	if cfg.ServerURL == "" {
		return nil, fmt.Errorf("server URL is required")
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		timeout := cfg.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		httpClient = &http.Client{
			Timeout: timeout,
		}
	}

	return &Client{
		baseURL:    strings.TrimSuffix(cfg.ServerURL, "/"),
		httpClient: httpClient,
		apiKey:     cfg.APIKey,
	}, nil
}

// NewDefaultClient creates a client with configuration from environment variables.
//
// It reads the following environment variables:
//   - WATERFLOW_SERVER_URL: Server URL (defaults to http://localhost:8080)
//   - WATERFLOW_API_KEY: Optional API key for authentication
//
// Example:
//
//	export WATERFLOW_SERVER_URL=http://localhost:8080
//	export WATERFLOW_API_KEY=your-api-key
//
//	client, err := sdk.NewDefaultClient()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// This is the recommended way to configure the client in production environments
// where configuration is managed through environment variables.
func NewDefaultClient() (*Client, error) {
	serverURL := os.Getenv("WATERFLOW_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080" // Default
	}

	return NewClient(&ClientConfig{
		ServerURL: serverURL,
		APIKey:    os.Getenv("WATERFLOW_API_KEY"),
	})
}

// doRequest performs HTTP request with error handling
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader, result interface{}) error {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle error responses
	if resp.StatusCode >= 400 {
		return parseErrorResponse(resp)
	}

	// Decode success response
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// parseErrorResponse parses error response from server
func parseErrorResponse(resp *http.Response) error {
	var errResp struct {
		Error struct {
			Code    string                 `json:"code"`
			Message string                 `json:"message"`
			Details map[string]interface{} `json:"details"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return &ServerError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("HTTP %d", resp.StatusCode),
		}
	}

	return &ServerError{
		StatusCode: resp.StatusCode,
		Code:       errResp.Error.Code,
		Message:    errResp.Error.Message,
		Details:    errResp.Error.Details,
	}
}
