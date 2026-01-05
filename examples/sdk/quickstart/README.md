# Waterflow Go SDK Quickstart

5-minute guide to get started with the Waterflow Go SDK.

## Prerequisites

- Go 1.21+
- Waterflow Server running on `localhost:8080`

## Quick Start

1. Run the example:

```bash
cd examples/sdk/quickstart
go run main.go
```

Expected output:

```
=== Waterflow Go SDK Quickstart ===
Creating client...
✓ Client created

Submitting workflow...
✓ Workflow submitted: wf-abc123

Checking status...
✓ Status: completed

Fetching logs...
✓ Retrieved 5 log entries
  [info] Workflow started
  [info] Hello from Waterflow Go SDK!
  [info] Mon Jan  6 10:30:00 UTC 2026
  [info] Step completed
  [info] Workflow completed

✅ Quickstart complete!
```

## What This Example Does

1. **Creates a client** - Connects to Waterflow Server at localhost:8080
2. **Reads workflow YAML** - Loads workflow.yaml from current directory
3. **Submits workflow** - Sends workflow for execution
4. **Checks status** - Queries current workflow status
5. **Fetches logs** - Retrieves execution logs

## Next Steps

- See `../basic/` for more detailed examples
- Read `../../../pkg/sdk/README.md` for full API reference
- Check workflow YAML syntax in documentation

## Common Issues

**Connection refused:**
```
Failed to create client: connection refused
```
→ Make sure Waterflow Server is running on localhost:8080

**Workflow validation failed:**
```
Failed to submit workflow: validation error
```
→ Check your workflow.yaml syntax

## Customization

Edit `workflow.yaml` to try different workflows:

```yaml
name: my-custom-workflow

jobs:
  my-job:
    runs-on: default
    steps:
      - name: My step
        uses: exec/shell@v1
        with:
          command: echo "Hello, World!"
```

Then run `go run main.go` again.
