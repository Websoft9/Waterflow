# Waterflow Go SDK Examples

This directory contains examples demonstrating how to use the Waterflow Go SDK.

## Quick Navigation

| Example | Description | Difficulty |
|---------|-------------|------------|
| [quickstart/](quickstart/) | 5-minute getting started guide | ⭐ Beginner |
| [basic/error_handling.go](basic/error_handling.go) | Error handling patterns | ⭐⭐ Intermediate |

## Getting Started

### Prerequisites

- Go 1.21 or later
- Waterflow Server running on `localhost:8080`

Start Waterflow Server:

```bash
cd deployments
docker compose up -d
```

Verify server is running:

```bash
curl http://localhost:8080/health
```

### Running Examples

#### Quickstart (Recommended for Beginners)

```bash
cd quickstart
go run main.go
```

This demonstrates:
- Creating a client
- Submitting a workflow
- Checking status
- Retrieving logs

#### Error Handling

```bash
cd basic
go run error_handling.go
```

This demonstrates:
- Validation errors
- Not found errors
- Using IsValidationError() and IsNotFound() helpers

## Example Structure

```
examples/sdk/
├── README.md              # This file
├── quickstart/            # 5-minute getting started
│   ├── main.go
│   ├── workflow.yaml
│   └── README.md
└── basic/                 # Basic usage patterns
    └── error_handling.go
```

## Common Patterns

### Creating a Client

```go
// With explicit configuration
client, err := sdk.NewClient(&sdk.ClientConfig{
    ServerURL: "http://localhost:8080",
    Timeout:   30 * time.Second,
})

// From environment variables
client, err := sdk.NewDefaultClient()
```

### Submitting a Workflow

```go
yamlContent := `
name: my-workflow
jobs:
  build:
    runs-on: default
    steps:
      - name: Build
        uses: exec/shell@v1
        with:
          command: make build
`

resp, err := client.SubmitWorkflow(ctx, &sdk.SubmitWorkflowRequest{
    YAML: yamlContent,
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Workflow ID: %s\n", resp.ID)
```

### Getting Status

```go
status, err := client.GetWorkflowStatus(ctx, workflowID)
if err != nil {
    if sdk.IsNotFound(err) {
        fmt.Println("Workflow not found")
        return
    }
    log.Fatal(err)
}

fmt.Printf("Status: %s\n", status.Status)

// Jobs and Steps are arrays
for _, job := range status.Jobs {
    fmt.Printf("Job %s: %s\n", job.Name, job.Status)
}
```

### Error Handling

```go
resp, err := client.SubmitWorkflow(ctx, req)
if err != nil {
    if sdk.IsValidationError(err) {
        fmt.Println("Invalid YAML syntax")
    } else if sdk.IsNotFound(err) {
        fmt.Println("Resource not found")
    } else {
        fmt.Printf("Error: %v\n", err)
    }
    return
}
```

## Documentation

- **API Reference**: [pkg/sdk/README.md](../../pkg/sdk/README.md)
- **GoDoc**: https://pkg.go.dev/github.com/Websoft9/waterflow/pkg/sdk
- **Project Homepage**: https://github.com/Websoft9/waterflow

## Troubleshooting

**Connection refused:**
```
Failed to create client: dial tcp 127.0.0.1:8080: connect: connection refused
```
→ Make sure Waterflow Server is running. Run `docker compose up -d` in the `deployments/` directory.

**Import error:**
```
cannot find package "github.com/Websoft9/waterflow/pkg/sdk"
```
→ Run `go get github.com/Websoft9/waterflow/pkg/sdk`

## Contributing

Want to add more examples? Please submit a pull request!

## License

Apache 2.0
