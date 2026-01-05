# Waterflow Go SDK

Go client library for [Waterflow](https://github.com/Websoft9/waterflow) workflow orchestration engine.

[![Go Reference](https://pkg.go.dev/badge/github.com/Websoft9/waterflow/pkg/sdk.svg)](https://pkg.go.dev/github.com/Websoft9/waterflow/pkg/sdk)
[![Go Report Card](https://goreportcard.com/badge/github.com/Websoft9/waterflow)](https://goreportcard.com/report/github.com/Websoft9/waterflow)

## Features

- ✅ **Type-safe** - Full Go type definitions for all API operations
- ✅ **Context-aware** - Timeout and cancellation support via context
- ✅ **Error handling** - Typed errors with detailed messages  
- ✅ **Production-ready** - Connection pooling, configurable timeouts
- ✅ **Simple API** - Idiomatic Go interface following best practices

## Installation

```bash
go get github.com/Websoft9/waterflow/pkg/sdk
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/Websoft9/waterflow/pkg/sdk"
)

func main() {
    // Create client
    client, err := sdk.NewClient(&sdk.ClientConfig{
        ServerURL: "http://localhost:8080",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Submit workflow
    yamlContent := `
name: hello-world
jobs:
  greet:
    runs-on: default
    steps:
      - name: Say hello
        uses: exec/shell@v1
        with:
          command: echo "Hello from Waterflow SDK"
`
    
    resp, err := client.SubmitWorkflow(context.Background(), &sdk.SubmitWorkflowRequest{
        YAML: yamlContent,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Workflow submitted: %s\n", resp.ID)
    
    // Get status
    status, err := client.GetWorkflowStatus(context.Background(), resp.ID)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Status: %s\n", status.Status)
}
```

For a complete runnable example, see [examples/sdk/quickstart](../../examples/sdk/quickstart).

## API Reference

### Client Creation

#### NewClient

```go
func NewClient(cfg *ClientConfig) (*Client, error)
```

Creates a new Waterflow client with custom configuration.

```go
client, err := sdk.NewClient(&sdk.ClientConfig{
    ServerURL:  "https://waterflow.example.com",
    APIKey:     "your-api-key",
    Timeout:    60 * time.Second,
})
```

#### NewDefaultClient

```go
func NewDefaultClient() (*Client, error)
```

Creates a client using environment variables:
- `WATERFLOW_SERVER_URL` - Server URL (default: http://localhost:8080)
- `WATERFLOW_API_KEY` - Optional API key

```go
export WATERFLOW_SERVER_URL=http://localhost:8080
export WATERFLOW_API_KEY=your-api-key

client, err := sdk.NewDefaultClient()
```

### Workflow Operations

#### SubmitWorkflow

```go
func (c *Client) SubmitWorkflow(ctx context.Context, req *SubmitWorkflowRequest) (*SubmitWorkflowResponse, error)
```

Submits a workflow for execution.

```go
resp, err := client.SubmitWorkflow(ctx, &sdk.SubmitWorkflowRequest{
    YAML: yamlContent,
    Vars: map[string]interface{}{
        "environment": "production",
    },
})
```

#### GetWorkflowStatus

```go
func (c *Client) GetWorkflowStatus(ctx context.Context, workflowID string) (*WorkflowStatus, error)
```

Gets detailed workflow status including jobs and steps.

```go
status, err := client.GetWorkflowStatus(ctx, workflowID)
if err != nil {
    return err
}

// Jobs and Steps are arrays
for _, job := range status.Jobs {
    fmt.Printf("Job %s: %s\n", job.Name, job.Status)
    for _, step := range job.Steps {
        fmt.Printf("  Step %s: %s\n", step.Name, step.Status)
    }
}
```

### Log Operations

#### GetWorkflowLogs

```go
func (c *Client) GetWorkflowLogs(ctx context.Context, req *GetLogsRequest) ([]LogEntry, error)
```

Retrieves workflow logs with optional filtering.

```go
logs, err := client.GetWorkflowLogs(ctx, &sdk.GetLogsRequest{
    WorkflowID: workflowID,
    Level:      "error,warn",  // Filter by log level
    Job:        "deploy",       // Filter by job name
    Step:       "Build",        // Filter by step name
    Tail:       100,            // Last 100 entries
})
for _, log := range logs {
    fmt.Printf("[%s] %s\n", log.Level, log.Message)
}
```

#### ListWorkflows

```go
func (c *Client) ListWorkflows(ctx context.Context, req *ListWorkflowsRequest) (*ListWorkflowsResponse, error)
```

Lists workflows with pagination and filtering.

```go
resp, err := client.ListWorkflows(ctx, &sdk.ListWorkflowsRequest{
    Page:   1,
    Limit:  20,
    Status: "running",
    Name:   "deploy",
})
if err != nil {
    return err
}

fmt.Printf("Total: %d workflows\n", resp.Pagination.Total)
for _, wf := range resp.Workflows {
    fmt.Printf("- %s: %s\n", wf.Name, wf.Status)
}
```

### Control Operations

#### CancelWorkflow

```go
func (c *Client) CancelWorkflow(ctx context.Context, workflowID string) error
```

Cancels a running workflow.

```go
err := client.CancelWorkflow(ctx, workflowID)
```

#### RerunWorkflow

```go
func (c *Client) RerunWorkflow(ctx context.Context, req *RerunWorkflowRequest) (*SubmitWorkflowResponse, error)
```

Reruns a workflow with optional variable overrides.

```go
resp, err := client.RerunWorkflow(ctx, &sdk.RerunWorkflowRequest{
    WorkflowID: originalWorkflowID,
    Vars: map[string]interface{}{
        "environment": "staging",
        "timeout":     600,
    },
})
if err != nil {
    return err
}

fmt.Printf("Workflow rerun: %s\n", resp.ID)
```

## Configuration

### Basic Configuration

```go
client, err := sdk.NewClient(&sdk.ClientConfig{
    ServerURL: "http://localhost:8080",
})
```

### With Authentication

```go
client, err := sdk.NewClient(&sdk.ClientConfig{
    ServerURL: "https://waterflow.example.com",
    APIKey:    "your-api-key-here",
})
```

The API key is sent in the `Authorization` header as `Bearer your-api-key-here`.

### Custom Timeout

```go
client, err := sdk.NewClient(&sdk.ClientConfig{
    ServerURL: "http://localhost:8080",
    Timeout:   60 * time.Second,  // 60 second timeout
})
```

Default timeout is 30 seconds.

### Per-Request Timeout

Use context for request-specific timeouts:

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

status, err := client.GetWorkflowStatus(ctx, workflowID)
```

### Custom HTTP Client

For advanced configuration (TLS, proxies, connection pooling):

```go
import (
    "crypto/tls"
    "net/http"
)

httpClient := &http.Client{
    Timeout: 60 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        TLSClientConfig: &tls.Config{
            InsecureSkipVerify: false,
        },
    },
}

client, err := sdk.NewClient(&sdk.ClientConfig{
    ServerURL:  "https://waterflow.example.com",
    HTTPClient: httpClient,
})
```

### Environment Variables

```bash
export WATERFLOW_SERVER_URL=http://localhost:8080
export WATERFLOW_API_KEY=your-api-key
```

```go
client, err := sdk.NewDefaultClient()
```

## Error Handling

The SDK provides typed errors for different scenarios.

### Error Types

| Type | Helper | Description |
|------|--------|-------------|
| `ServerError` | `IsNotFound()` | 404 Not Found |
| `ServerError` | `IsValidationError()` | 400/422 Validation errors |
| `context.DeadlineExceeded` | `errors.Is()` | Request timeout |

### Handling Not Found Errors

```go
status, err := client.GetWorkflowStatus(ctx, workflowID)
if err != nil {
    if sdk.IsNotFound(err) {
        fmt.Println("Workflow not found")
        return
    }
    return err
}
```

### Handling Validation Errors

```go
resp, err := client.SubmitWorkflow(ctx, req)
if err != nil {
    if sdk.IsValidationError(err) {
        fmt.Println("Invalid workflow YAML")
        if serverErr, ok := err.(*sdk.ServerError); ok {
            fmt.Printf("Message: %s\n", serverErr.Message)
        }
        return
    }
    return err
}
```

### Complete Error Handling Example

```go
func submitWorkflow(client *sdk.Client, yamlContent string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    resp, err := client.SubmitWorkflow(ctx, &sdk.SubmitWorkflowRequest{
        YAML: yamlContent,
    })
    if err != nil {
        if sdk.IsValidationError(err) {
            fmt.Println("❌ Workflow validation failed")
            return err
        }
        
        if sdk.IsNotFound(err) {
            fmt.Println("❌ Resource not found")
            return err
        }
        
        if errors.Is(err, context.DeadlineExceeded) {
            fmt.Println("❌ Request timed out")
            return err
        }
        
        fmt.Printf("❌ Unexpected error: %v\n", err)
        return err
    }

    fmt.Printf("✅ Workflow submitted: %s\n", resp.ID)
    return nil
}
```

See [examples/sdk/basic/error_handling.go](../../examples/sdk/basic/error_handling.go) for more examples.

## Concurrency

The Client is safe for concurrent use by multiple goroutines. Share a single client instance:

```go
var globalClient *sdk.Client

func init() {
    var err error
    globalClient, err = sdk.NewDefaultClient()
    if err != nil {
        panic(err)
    }
}

// Use in multiple goroutines safely
func submitWorkflows(workflows []string) {
    var wg sync.WaitGroup
    for _, yaml := range workflows {
        wg.Add(1)
        go func(y string) {
            defer wg.Done()
            resp, err := globalClient.SubmitWorkflow(ctx, &sdk.SubmitWorkflowRequest{
                YAML: y,
            })
            // ...
        }(yaml)
    }
    wg.Wait()
}
```

## Examples

See the [examples/sdk](../../examples/sdk) directory:

- **quickstart/** - 5-minute getting started guide
- **basic/** - Basic usage patterns (submit, status, logs, errors)

## License

Apache 2.0
