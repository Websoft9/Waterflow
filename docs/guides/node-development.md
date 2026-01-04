# Node Development Guide

**版本:** 2.0 (Epic 4 完整版)  
**最后更新:** 2026-01-04  
**前置 Stories:** 3.1, 4.1, 4.2, 4.3, 4.4

Complete guide for developing Waterflow node plugins - from basics to production.

> **本指南基于 Epic 4 (Node Extension System) 完成后的最新实现。**  
> 如需查看旧版本,请参考 [v1 备份](node-development-v1-backup.md)。

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start) (30 分钟上手)
3. [Core Concepts](#core-concepts) (核心概念)
4. [Node Examples](#node-examples) (分层示例)
   - [Example 1: Echo Node](#example-1-echo-node-simple) (简单)
   - [Example 2: Greeter Node](#example-2-greeter-node-intermediate) (中等)
   - [Example 3: HTTP Request](#example-3-http-request-production) (生产级)
5. [Parameter Validation](#parameter-validation) (参数验证)
6. [Error Handling and Logging](#error-handling-and-logging) (错误处理和日志)
7. [Testing Best Practices](#testing-best-practices) (测试最佳实践)
8. [Build and Deploy](#build-and-deploy) (编译和部署)
9. [Best Practices](#best-practices) (最佳实践)
10. [Troubleshooting](#troubleshooting) (故障排除)
11. [FAQ](#faq)
12. [References](#references)

---

## Overview

Nodes are the fundamental execution units in Waterflow workflows. Each node is implemented as a Go plugin (.so file) that conforms to the `node.Node` interface.

**Real-World Examples**:
- **Simple**: [Echo Node](../../examples/plugins/echo/) - Basic echo functionality (入门示例)
- **Advanced**: [Greeter Node](../../examples/plugins/greeter/) - Multi-language greetings with parameter validation (实战示例, 推荐参考)

## Prerequisites

- **Go Version**: 1.22.0+ (must match Agent's Go version exactly)
- **CGO**: Must be enabled (`export CGO_ENABLED=1`)
- **Platform**: Linux or macOS (Windows does not support Go plugins)

## Quick Start

### 1. Copy the Template

```bash
cp -r examples/plugins/template examples/plugins/mynode
cd examples/plugins/mynode
```

### 2. Implement the Node Interface

All nodes must implement 5 methods from `pkg/dsl/node.Node`:

```go
type Node interface {
    Name() string
    Version() string
    Params() map[string]ParamSpec
    Execute(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error)
    Metadata() NodeMetadata
}
```

### 3. Define Node Identity

```go
func (n *MyNode) Name() string {
    return "category/name"  // e.g., "exec/shell", "docker/compose"
}

func (n *MyNode) Version() string {
    return "v1"  // or "v1.2.3"
}
```

**Valid Categories**: `exec`, `docker`, `http`, `file`, `flow`

### 4. Specify Parameters

```go
func (n *MyNode) Params() map[string]node.ParamSpec {
    min, max := 1.0, 100.0
    return map[string]node.ParamSpec{
        "command": {
            Type:        "string",
            Required:    true,
            Description: "Command to execute",
            Pattern:     `^[a-zA-Z0-9\s\-_]+$`,  // Regex validation
        },
        "timeout": {
            Type:        "int",
            Required:    false,
            Default:     60,
            Description: "Timeout in seconds",
            MinValue:    &min,  // Numeric range
            MaxValue:    &max,
        },
        "log_level": {
            Type:        "string",
            Required:    false,
            Default:     "info",
            Description: "Logging level",
            Enum:        []interface{}{"debug", "info", "warn", "error"},  // Enum validation
        },
    }
}
```

**Supported Types**: `string`, `int`, `float`, `bool`, `object`, `array`

**Advanced Validation**:
- `Pattern` - Regular expression (for strings)
- `Enum` - Allowed values
- `MinValue` / `MaxValue` - Numeric ranges (for int/float)

#### Parameter Validation Details

**Automatic Validation**: Waterflow automatically validates all parameters against your `ParamSpec` definitions:

1. **Submission Time** (Static Values): DSL parser validates non-expression parameters when workflow is submitted
2. **Runtime** (All Values): Activity validates all parameters (including expression results) before calling `Execute()`

**Validation Rules**:

| Constraint | Applies To | Example |
|------------|-----------|---------|
| `Required` | All types | Missing required parameter → Error |
| `Type` | All types | String expected, int provided → Error |
| `Pattern` | `string` | Email regex `^[a-z]+@[a-z]+\.[a-z]+$` |
| `Enum` | All types | Must be one of `["GET", "POST", "PUT"]` |
| `MinValue`/`MaxValue` | `int`, `float` | Value must be in range `[1, 100]` |
| `Default` | Optional params | Applied if value not provided |

**Type Compatibility** (JSON/YAML Parsing):
- Numbers in JSON/YAML are parsed as `float64`
- `Type: "int"` accepts `float64` if no decimal part (e.g., `30.0` → valid int)
- `Type: "float"` accepts both `int` and `float64`

**Expression Parameters**:
- Expressions like `${{ vars.timeout }}` skip validation at submission time
- Validated after expression evaluation at runtime
- Expression results must match the declared type

**Error Handling**:
```go
// Validation errors are NonRetryableError (permanent)
err := node.ValidateInputs(inputs, n.Params())
if err != nil {
    // Error type: *node.InputValidationError
    // Properties:
    //   - NodeName: "exec/shell@v1"
    //   - Errors: []ParameterError
    //     - ParamName: "timeout"
    //     - ErrorType: "RangeViolation"
    //     - Expected: "<= 60"
    //     - Actual: 120
    //     - Message: "value 120 exceeds maximum 60"
    return nil, err
}
```

**Validation Error Types**:
- `Missing` - Required parameter not provided
- `TypeMismatch` - Wrong type (e.g., string instead of int)
- `PatternMismatch` - String doesn't match regex pattern
- `EnumViolation` - Value not in allowed list
- `RangeViolation` - Number outside min/max range

**Best Practices**:
- Use `Pattern` for formats (emails, URLs, file paths)
- Use `Enum` for limited choices (methods, log levels)
- Use `MinValue`/`MaxValue` for sensible ranges (timeouts, counts)
- Provide `Default` values for optional parameters
- Write descriptive `Description` to help users

### 5. Implement Execute Logic

```go
func (n *MyNode) Execute(ctx context.Context, inputs map[string]interface{}) (*node.NodeResult, error) {
    start := time.Now()
    
    // ⚠️ 参数已由 Temporal Activity 验证，无需再次验证
    // Waterflow 保证传入的 inputs 100% 符合 ParamSpec 定义
    
    // 1. 提取参数 (类型断言安全，因为已验证)
    command := inputs["command"].(string)
    timeout := 60 // Default value
    if t, ok := inputs["timeout"]; ok {
        // Type already validated - safe to assert
        switch v := t.(type) {
        case int:
            timeout = v
        case float64:
            timeout = int(v) // JSON numbers are float64
        }
    }
    
    // 2. Create result
    result := node.NewNodeResult()
    result.AddLog("Execution started")
    
    // 3. Perform work (check context cancellation)
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // Your logic here
        output, err := executeCommand(command)
        if err != nil {
            return nil, err
        }
        
        result.SetOutput("stdout", output)
        result.SetOutput("exit_code", 0)
    }
    
    // 4. Record duration and logs
    result.Duration = time.Since(start)
    result.AddLog("Execution completed")
    
    return result, nil
}
```

**Best Practices**:
- **Do NOT validate inputs** - Already validated by Activity layer
- Handle `ctx.Done()` for cancellation support
- Log execution progress for debugging
- Return structured outputs in OutputSchema format
- Record execution duration for metrics

### 6. Provide Metadata

```go
func (n *MyNode) Metadata() node.NodeMetadata {
    return node.NodeMetadata{
        Description: "Executes shell commands",
        Category:    "exec",
        InputSchema: n.Params(),
        OutputSchema: map[string]interface{}{
            "stdout":    "string - Command output",
            "exit_code": "int - Exit code",
            "duration":  "duration - Execution time",
        },
    }
}
```

### 7. Export Register Function

```go
func Register() node.Node {
    return &MyNode{}
}
```

This function is the plugin entry point. The plugin loader calls it to instantiate your node.

## Building

### Set Environment

```bash
export CGO_ENABLED=1
```

### Compile Plugin

```bash
go build -buildmode=plugin -o mynode.so main.go
```

### Verify Build

```bash
ls -lh mynode.so  # Should be >100KB
```

## Testing

### Unit Tests

Create `main_test.go`:

```go
package main

import (
    "context"
    "testing"
    
    "github.com/Websoft9/waterflow/pkg/dsl/node"
    "github.com/stretchr/testify/assert"
)

func TestMyNode_Execute(t *testing.T) {
    n := &MyNode{}
    
    result, err := n.Execute(context.Background(), map[string]interface{}{
        "command": "echo hello",
    })
    
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, "hello\n", result.Outputs["stdout"])
}

func TestMyNode_Validation(t *testing.T) {
    n := &MyNode{}
    err := node.ValidateNode(n)
    assert.NoError(t, err)
}
```

### Run Tests

```bash
go test -v -cover
```

**Coverage Target**: >80%

## Error Handling

### Distinguish Error Types

```go
// Temporary error (retriable)
if isNetworkTimeout(err) {
    return nil, fmt.Errorf("temporary network error: %w", err)
}

// Permanent error (not retriable)
if isInvalidParameter(err) {
    return nil, fmt.Errorf("invalid parameter: %w", err)
}
```

### Use Structured Errors

```go
type NodeExecutionError struct {
    Code    string
    Message string
    Cause   error
}

func (e *NodeExecutionError) Error() string {
    return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
}
```

## Performance Considerations

### Target Performance

- **Execute Call Overhead**: <1ms
- **Parameter Validation**: <100μs
- **Memory per Instance**: <1KB

### Optimization Tips

1. **Avoid Repeated Allocations**: Reuse buffers
2. **Cache Expensive Operations**: Compile regex once
3. **Use Context Timeouts**: Prevent hanging
4. **Log Strategically**: Too many logs hurt performance

### Benchmark Your Node

```go
func BenchmarkMyNode_Execute(b *testing.B) {
    n := &MyNode{}
    ctx := context.Background()
    inputs := map[string]interface{}{"command": "echo test"}
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = n.Execute(ctx, inputs)
    }
}
```

Run:
```bash
go test -bench=. -benchmem
```

## Common Issues

### Plugin Version Mismatch

**Error**: `plugin was built with a different version of package`

**Solution**: Rebuild plugin with exact same Go version as Agent:
```bash
go version  # Check Agent's Go version
# Use same version to build plugin
go build -buildmode=plugin -o node.so
```

### CGO Not Enabled

**Error**: `CGO_ENABLED must be 1 for plugin builds`

**Solution**:
```bash
export CGO_ENABLED=1
go build -buildmode=plugin -o node.so
```

### Undefined node.Node

**Error**: `undefined: node.Node`

**Solution**: Check import path:
```go
import "github.com/Websoft9/waterflow/pkg/dsl/node"  // Capital 'W'
```

### Windows Build Failure

**Error**: `Go plugins not supported on Windows`

**Solution**: Go plugins only work on Linux/macOS. Use WSL or Docker for Windows development.

## Advanced Topics

### Stateful Nodes

If your node requires initialization/cleanup (e.g., database connections):

```go
type StatefulNode struct {
    conn *sql.DB
}

func (n *StatefulNode) Init(ctx context.Context) error {
    var err error
    n.conn, err = sql.Open("postgres", connStr)
    return err
}

func (n *StatefulNode) Execute(ctx context.Context, inputs map[string]interface{}) (*node.NodeResult, error) {
    // Use n.conn
    rows, err := n.conn.QueryContext(ctx, query)
    // ...
}

func (n *StatefulNode) Close() error {
    return n.conn.Close()
}
```

### Concurrent Safety

Nodes may be called concurrently. Protect shared state:

```go
type ConcurrentNode struct {
    mu    sync.Mutex
    cache map[string]string
}

func (n *ConcurrentNode) Execute(ctx context.Context, inputs map[string]interface{}) (*node.NodeResult, error) {
    n.mu.Lock()
    defer n.mu.Unlock()
    
    // Safe access to n.cache
}
```

### Custom Validation

Beyond `ValidateInputs`, add domain-specific checks:

```go
func (n *MyNode) validateBusinessRules(inputs map[string]interface{}) error {
    if inputs["count"].(int) > inputs["limit"].(int) {
        return errors.New("count cannot exceed limit")
    }
    return nil
}
```

## Resources

- **Interface Definition**: [pkg/dsl/node/interface.go](../../pkg/dsl/node/interface.go)
- **Example Nodes**: [examples/plugins/](../../examples/plugins/)
- **ADR-0003**: [Plugin-Based Node System](../adr/0003-plugin-based-node-system.md)
- **Go Plugin Documentation**: https://pkg.go.dev/plugin

## Checklist

Before releasing your node:

- [ ] All 5 interface methods implemented
- [ ] Name follows `category/name` format
- [ ] Version follows `vX.Y.Z` format
- [ ] All parameters documented
- [ ] Input validation implemented
- [ ] Context cancellation handled
- [ ] Unit tests written (>80% coverage)
- [ ] Plugin compiles successfully
- [ ] README.md created
- [ ] Example usage documented


---

## Core Concepts

> This section provides an in-depth exploration of the Waterflow node system fundamentals.

### Node Architecture Overview

Nodes are the building blocks of Waterflow workflows. Understanding their architecture is key to effective development.

**Key Characteristics**:

1. **Go Plugins** - Nodes are compiled as `.so` files, dynamically loaded by Agents
2. **Stateless** - Each Execute() call is independent, safe for concurrent execution
3. **Interface-Based** - All nodes implement the same `node.Node` interface
4. **Versioned** - Support multiple versions simultaneously (v1, v2, etc.)
5. **Category-Organized** - Grouped by functionality (exec, docker, http, etc.)

**System Architecture**:

```
┌──────────────────────────────────────┐
│  Waterflow Server                     │
│  ├─ YAML Parser                      │
│  ├─ Workflow Compiler                │
│  └─ Temporal Client                  │
└──────────────┬───────────────────────┘
               │ gRPC
    ┌──────────▼────────────┐
    │  Temporal Server       │
    │  ├─ Event Sourcing    │
    │  ├─ Task Routing      │
    │  └─ State Persistence │
    └──────────┬────────────┘
               │ Long Polling
    ┌──────────▼────────────┐
    │  Waterflow Agent       │
    │  ┌──────────────────┐ │
    │  │ Plugin Manager   │ │
    │  ├──────────────────┤ │
    │  │ Node Registry    │ │
    │  │  exec/shell@v1   │ │ ← Your Custom Nodes
    │  │  http/request@v2 │ │
    │  │  custom/greeter@v1 │
    │  └──────────────────┘ │
    └───────────────────────┘
```

**References**:
- [ADR-0003: Plugin-Based Node System](../adr/0003-plugin-based-node-system.md)
- [Story 3.1: Node Interface Design](../sprint-artifacts/3-1-node-interface-design.md)

### The Node Interface - Method by Method

All nodes must implement these 5 methods:

```go
type Node interface {
    Name() string
    Version() string
    Params() map[string]ParamSpec
    Execute(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error)
    Metadata() NodeMetadata
}
```

#### Method 1: Name() string

**Purpose**: Returns unique identifier in `category/name` format.

**Format**:
- Pattern: `^[a-z]+/[a-z][a-z0-9-]*$`
- Example: `exec/shell`, `docker/compose`, `custom/greeter`

**Valid Categories**:

| Category | Purpose | Example Nodes |
|----------|---------|---------------|
| `exec` | Command execution | `exec/shell`, `exec/script` |
| `docker` | Container operations | `docker/exec`, `docker/compose` |
| `http` | HTTP/REST API | `http/request`, `http/webhook` |
| `file` | File operations | `file/transfer`, `file/read` |
| `flow` | Control flow | `flow/sleep`, `flow/condition` |
| `custom` | User-defined | `custom/greeter`, `custom/*` |

**Naming Best Practices**:
- Use lowercase with hyphens: `file-processor` not `fileProcessor`
- Be descriptive: `http/request` not `http/req`
- Use singular nouns: `docker/container` not `docker/containers`
- Keep concise: 5-15 characters

**Example**:

```go
func (n *GreeterNode) Name() string {
    return "custom/greeter"  // ✅ Correct
}

// ❌ Invalid examples:
// return "Greeter"           // Missing category
// return "custom/Greeter"    // Capital letter
// return "custom_greeter"    // Underscore instead of slash
```

#### Method 2: Version() string

**Purpose**: Returns semantic version for compatibility tracking.

**Format**: `vX.Y.Z` or `vX` (simplified)

**Version Strategy**:

| Change Type | Version Bump | Example | Notes |
|-------------|--------------|---------|-------|
| Breaking changes | Major (v1 → v2) | Remove parameter, change behavior | Users must update workflows |
| New features | Minor (v1 → v1.1) | Add optional parameter | Backward compatible |
| Bug fixes | Patch (v1.0 → v1.0.1) | Fix validation logic | No interface changes |

**Multiple Versions**:
- Agents can load multiple versions simultaneously
- Workflows specify: `uses: http/request@v2`
- Omitting version defaults to latest: `uses: http/request`

**Example**:

```go
func (n *GreeterNode) Version() string {
    return "v1"  // Recommended for most nodes
}

// Or full semantic versioning:
func (n *EnhancedNode) Version() string {
    return "v2.1.3"  // Major.Minor.Patch
}
```

#### Method 3: Params() map[string]ParamSpec

**Purpose**: Defines input parameters and validation rules.

**ParamSpec Structure**:

```go
type ParamSpec struct {
    Type        string        // Required: string, int, float, bool, object, array
    Required    bool          // Optional: default false
    Description string        // Optional: human-readable description
    Default     interface{}   // Optional: default value
    Pattern     string        // Optional: regex for string validation
    Enum        []interface{} // Optional: allowed values
    MinValue    *float64      // Optional: minimum for int/float
    MaxValue    *float64      // Optional: maximum for int/float
}
```

**Parameter Types**:

| Type | Go Type | JSON/YAML Examples | Notes |
|------|---------|-------------------|-------|
| `string` | string | `"hello"` | Text values |
| `int` | int/float64* | `42`, `30.0` | JSON numbers are float64 |
| `float` | float64 | `3.14`, `10` | Accepts integers too |
| `bool` | bool | `true`, `false` | Boolean values |
| `object` | map[string]interface{} | `{key: value}` | Nested structures |
| `array` | []interface{} | `[1, 2, 3]` | Lists |

\* Type `int` accepts `float64` if no decimal part (e.g., `30.0` → valid int)

**Complete Example**:

```go
func (n *HTTPRequestNode) Params() map[string]node.ParamSpec {
    minTimeout, maxTimeout := 1.0, 3600.0
    minRetry, maxRetry := 0.0, 10.0
    
    return map[string]node.ParamSpec{
        "url": {
            Type:        "string",
            Required:    true,
            Description: "HTTP request URL",
            Pattern:     `^https?://[^\s]+$`,  // URL format validation
        },
        "method": {
            Type:        "string",
            Required:    false,
            Default:     "GET",
            Description: "HTTP method",
            Enum:        []interface{}{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"},
        },
        "timeout": {
            Type:        "int",
            Required:    false,
            Default:     30,
            Description: "Request timeout in seconds",
            MinValue:    &minTimeout,
            MaxValue:    &maxTimeout,
        },
        "headers": {
            Type:        "object",
            Required:    false,
            Description: "HTTP headers as key-value pairs",
        },
        "body": {
            Type:        "string",
            Required:    false,
            Description: "Request body",
        },
        "retry_count": {
            Type:        "int",
            Required:    false,
            Default:     3,
            Description: "Number of retry attempts on failure",
            MinValue:    &minRetry,
            MaxValue:    &maxRetry,
        },
        "follow_redirects": {
            Type:        "bool",
            Required:    false,
            Default:     true,
            Description: "Follow HTTP redirects",
        },
    }
}
```

#### Method 4: Execute(ctx, inputs) (*NodeResult, error)

**Purpose**: Executes the node's core logic.

**Signature**:

```go
func (n *MyNode) Execute(
    ctx context.Context,              // Cancellation, timeout control
    inputs map[string]interface{},    // Validated parameters
) (*NodeResult, error) {
    // Implementation
}
```

**Implementation Pattern**:

```go
func (n *MyNode) Execute(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error) {
    // Phase 1: Timing & Setup
    start := time.Now()
    result := node.NewNodeResult()
    result.AddLog("Execution started")
    
    // Phase 2: Extract Inputs (already validated, safe to assert)
    url := inputs["url"].(string)
    method := "GET"
    if m, ok := inputs["method"].(string); ok {
        method = m
    }
    
    // Phase 3: Context-Aware Execution
    select {
    case <-ctx.Done():
        return nil, ctx.Err()  // Cancelled or timed out
    default:
    }
    
    // Phase 4: Perform Work
    response, err := doHTTPRequest(ctx, url, method)
    if err != nil {
        // Classify error (permanent vs temporary)
        if isClientError(err) {
            return nil, fmt.Errorf("client error: %w", err)  // Don't retry
        }
        return nil, err  // Temporary, will retry
    }
    
    // Phase 5: Build Result
    result.SetOutput("status_code", response.StatusCode)
    result.SetOutput("body", response.Body)
    result.SetOutput("headers", response.Headers)
    
    // Phase 6: Finalize
    result.Duration = time.Since(start)
    result.AddLog(fmt.Sprintf("Completed with status %d", response.StatusCode))
    
    return result, nil
}
```

**Best Practices**:

1. **NO Parameter Validation** - Inputs are pre-validated by Activity layer
2. **Respect Context** - Check `ctx.Done()` for cancellation
3. **Type Assertion Safety** - Inputs match ParamSpec types exactly
4. **Error Classification** - Distinguish permanent vs temporary errors
5. **Structured Logging** - Use `AddLog()` for execution trail
6. **Record Duration** - Set `result.Duration` for metrics
7. **Return Structured Outputs** - Match OutputSchema

**Context Handling**:

```go
// Set timeout from parameter
timeout := inputs["timeout"].(int)
ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
defer cancel()

// Check cancellation in loops
for i := 0; i < iterations; i++ {
    select {
    case <-ctx.Done():
        return nil, fmt.Errorf("cancelled after %d iterations: %w", i, ctx.Err())
    default:
        // Continue processing
    }
}

// Pass context to downstream operations
resp, err := http.NewRequestWithContext(ctx, "GET", url, nil)
```

#### Method 5: Metadata() NodeMetadata

**Purpose**: Provides descriptive information for documentation and introspection.

**NodeMetadata Structure**:

```go
type NodeMetadata struct {
    Description  string                      // Human-readable description
    Category     string                      // Node category
    InputSchema  map[string]ParamSpec        // Copy of Params()
    OutputSchema map[string]interface{}      // Output descriptions
}
```

**Complete Example**:

```go
func (n *GreeterNode) Metadata() node.NodeMetadata {
    return node.NodeMetadata{
        Description: "Generates multi-language greetings based on time of day. " +
                    "Supports English, Chinese, Spanish, and French with automatic " +
                    "time detection for morning, afternoon, and evening.",
        Category: "custom",
        InputSchema: n.Params(),  // Reuse Params() definition
        OutputSchema: map[string]interface{}{
            "greeting": "string - The generated greeting message in the specified language",
            "language": "string - The language code used (en, zh, es, fr)",
            "time_of_day": "string - The detected or specified time period (morning, afternoon, evening)",
        },
    }
}
```

**OutputSchema Best Practices**:
- Describe each output field
- Include type and purpose
- Document possible values for enums
- Example: `"status_code": "int - HTTP status code (200-599)"`

### NodeResult Deep Dive

`NodeResult` carries the execution results back to the workflow.

**Structure**:

```go
type NodeResult struct {
    Outputs  map[string]interface{}  // Data produced by the node
    Logs     []string                // Execution logs (appears in Temporal UI)
    Duration time.Duration           // How long Execute() took
    Metadata map[string]interface{}  // Optional metadata
}
```

**Creating and Populating**:

```go
// Create result
result := node.NewNodeResult()

// Set outputs (accessed in workflow as ${{ steps.step_id.outputs.key }})
result.SetOutput("count", 42)
result.SetOutput("status", "success")
result.SetOutput("items", []string{"a", "b", "c"})

// Add logs (visible in Temporal UI and agent logs)
result.AddLog("Starting processing")
result.AddLog(fmt.Sprintf("Processed %d items", count))
result.AddLog("Completed successfully")

// Set optional metadata (for debugging, metrics)
result.SetMetadata("hostname", os.Hostname())
result.SetMetadata("execution_id", uuid.New().String())

// Record duration (usually set automatically)
result.Duration = time.Since(start)
```

**Output Access in Workflows**:

```yaml
steps:
  - name: Count Items
    uses: custom/counter@v1
    with:
      input: data.json
  
  - name: Display Count
    uses: exec/shell@v1
    with:
      command: echo "Counted ${{ steps['Count Items'].outputs.count }} items"
```

---

## Node Examples

This section provides three progressively complex examples demonstrating node development.

### Example 1: Echo Node (Simple)

**Complexity**: <50 LOC  
**Learning Focus**: Basic interface implementation, parameter validation, output handling

**Full Implementation**:

```go
// examples/plugins/echo/main.go
package main

import (
    "context"
    "time"
    
    "github.com/Websoft9/waterflow/pkg/dsl/node"
)

// EchoNode returns input message as output
type EchoNode struct{}

func (n *EchoNode) Name() string {
    return "flow/echo"
}

func (n *EchoNode) Version() string {
    return "v1"
}

func (n *EchoNode) Params() map[string]node.ParamSpec {
    return map[string]node.ParamSpec{
        "message": {
            Type:        "string",
            Required:    true,
            Description: "Message to echo",
        },
    }
}

func (n *EchoNode) Execute(ctx context.Context, inputs map[string]interface{}) (*node.NodeResult, error) {
    start := time.Now()
    
    result := node.NewNodeResult()
    result.SetOutput("message", inputs["message"])
    result.AddLog("Echoed message")
    result.Duration = time.Since(start)
    
    return result, nil
}

func (n *EchoNode) Metadata() node.NodeMetadata {
    return node.NodeMetadata{
        Description: "Echoes input message to output",
        Category:    "flow",
        InputSchema: n.Params(),
        OutputSchema: map[string]interface{}{
            "message": "string - The echoed message",
        },
    }
}

func Register() node.Node {
    return &EchoNode{}
}
```

**Unit Tests**:

```go
// examples/plugins/echo/main_test.go
package main

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestEchoNode_Name(t *testing.T) {
    n := &EchoNode{}
    assert.Equal(t, "flow/echo", n.Name())
}

func TestEchoNode_Execute(t *testing.T) {
    n := &EchoNode{}
    inputs := map[string]interface{}{
        "message": "Hello, World!",
    }
    
    result, err := n.Execute(context.Background(), inputs)
    
    require.NoError(t, err)
    assert.Equal(t, "Hello, World!", result.Outputs["message"])
    assert.Greater(t, len(result.Logs), 0)
}
```

**Usage in Workflow**:

```yaml
steps:
  - name: Echo Test
    uses: flow/echo@v1
    with:
      message: "Hello from Waterflow!"
  
  - name: Print Echo
    uses: exec/shell@v1
    with:
      command: echo "${{ steps['Echo Test'].outputs.message }}"
```

### Example 2: Greeter Node (Intermediate)

**Complexity**: ~150 LOC  
**Learning Focus**: Enum validation, default values, multiple outputs, time handling

**Full Implementation**: See [examples/plugins/greeter/](../../examples/plugins/greeter/)

**Key Features**:
- 4 languages (English, Chinese, Spanish, French)
- 3 parameters with validation
- Automatic time detection
- 3 outputs
- 93.8% test coverage

**Highlights**:

```go
func (n *GreeterNode) Params() map[string]node.ParamSpec {
    return map[string]node.ParamSpec{
        "name": {
            Type:        "string",
            Required:    true,
            Description: "Person's name to greet",
        },
        "language": {
            Type:        "string",
            Required:    false,
            Default:     "en",
            Enum:        []interface{}{"en", "zh", "es", "fr"},  // Enum validation
        },
        "time_of_day": {
            Type:        "string",
            Required:    false,
            Default:     "auto",
            Enum:        []interface{}{"morning", "afternoon", "evening", "auto"},
        },
    }
}

func (n *GreeterNode) Execute(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error) {
    // Auto-detect time if needed
    timeOfDay := inputs["time_of_day"].(string)
    if timeOfDay == "auto" {
        hour := time.Now().Hour()
        switch {
        case hour < 12:
            timeOfDay = "morning"
        case hour < 18:
            timeOfDay = "afternoon"
        default:
            timeOfDay = "evening"
        }
    }
    
    greeting := generateGreeting(inputs["name"].(string), inputs["language"].(string), timeOfDay)
    
    result := node.NewNodeResult()
    result.SetOutput("greeting", greeting)
    result.SetOutput("language", inputs["language"])
    result.SetOutput("time_of_day", timeOfDay)
    
    return result, nil
}
```

**Usage**:

```yaml
steps:
  - name: Greet User
    uses: custom/greeter@v1
    with:
      name: 小明
      language: zh
      time_of_day: evening
  # Output: "晚上好,小明!"
```

### Example 3: HTTP Request (Production-Grade)

**Complexity**: ~300 LOC  
**Learning Focus**: Context management, timeout handling, error classification, retry logic, complex parameters

**Reference Implementation**: [Story 3.5: HTTP Request Node](../sprint-artifacts/3-5-http-request-node-implementation.md)

**Key Features**:
- 8+ parameters (URL, method, headers, body, timeout, retry, etc.)
- Context-aware HTTP client
- Error classification (4xx vs 5xx)
- Retry logic with exponential backoff
- TLS certificate validation
- Request/response logging

**Simplified Highlights**:

```go
func (n *HTTPRequestNode) Params() map[string]node.ParamSpec {
    minTimeout, maxTimeout := 1.0, 3600.0
    return map[string]node.ParamSpec{
        "url": {
            Type:        "string",
            Required:    true,
            Pattern:     `^https?://[^\s]+$`,
        },
        "method": {
            Type:        "string",
            Default:     "GET",
            Enum:        []interface{}{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"},
        },
        "timeout": {
            Type:        "int",
            Default:     30,
            MinValue:    &minTimeout,
            MaxValue:    &maxTimeout,
        },
        "headers": {Type: "object"},
        "body": {Type: "string"},
        "retry_count": {Type: "int", Default: 3},
        "follow_redirects": {Type: "bool", Default: true},
        "verify_tls": {Type: "bool", Default: true},
    }
}

func (n *HTTPRequestNode) Execute(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error) {
    // Create context with timeout
    timeout := time.Duration(inputs["timeout"].(int)) * time.Second
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    // Build HTTP request
    req, err := http.NewRequestWithContext(ctx, inputs["method"].(string), inputs["url"].(string), bodyReader)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    // Add headers
    if headers, ok := inputs["headers"].(map[string]interface{}); ok {
        for k, v := range headers {
            req.Header.Set(k, fmt.Sprint(v))
        }
    }
    
    // Execute with retry logic
    var resp *http.Response
    for attempt := 0; attempt <= retryCount; attempt++ {
        resp, err = client.Do(req)
        if err == nil && resp.StatusCode < 500 {
            break  // Success or client error (don't retry)
        }
        if attempt < retryCount {
            time.Sleep(time.Duration(1<<attempt) * time.Second)  // Exponential backoff
        }
    }
    
    if err != nil {
        // Classify error
        if isNetworkError(err) {
            return nil, err  // Temporary, will retry at workflow level
        }
        return nil, fmt.Errorf("permanent error: %w", err)  // Don't retry
    }
    
    // Read response
    body, _ := ioutil.ReadAll(resp.Body)
    defer resp.Body.Close()
    
    // Build result
    result := node.NewNodeResult()
    result.SetOutput("status_code", resp.StatusCode)
    result.SetOutput("body", string(body))
    result.SetOutput("headers", resp.Header)
    result.AddLog(fmt.Sprintf("HTTP %s %s -> %d", inputs["method"], inputs["url"], resp.StatusCode))
    
    // Classify as error if status >= 400
    if resp.StatusCode >= 400 {
        if resp.StatusCode < 500 {
            return result, fmt.Errorf("client error %d: %s", resp.StatusCode, resp.Status)  // Don't retry
        }
        return result, fmt.Errorf("server error %d: %s", resp.StatusCode, resp.Status)  // Retry
    }
    
    return result, nil
}
```

**Usage**:

```yaml
steps:
  - name: Call API
    uses: http/request@v1
    with:
      url: https://api.example.com/users
      method: POST
      headers:
        Content-Type: application/json
        Authorization: Bearer ${{ secrets.API_TOKEN }}
      body: '{"name": "Alice", "email": "alice@example.com"}'
      timeout: 60
      retry_count: 3
  
  - name: Process Response
    uses: exec/shell@v1
    with:
      command: echo "Status: ${{ steps['Call API'].outputs.status_code }}"
```

---


## Parameter Validation

> **Note**: The Activity layer handles parameter validation **before** Execute() is called.  
> This section explains the validation rules for ParamSpec configuration.

### Type Validation Rules

Each parameter type has specific validation logic:

#### 1. String Type

```go
"message": {
    Type:        "string",
    Required:    true,
    Pattern:     `^[A-Za-z0-9\s]+$`,  // Optional regex
    Enum:        []interface{}{"low", "medium", "high"},  // Optional allowed values
}
```

**Validation**:
- Must be JSON string or number (converted to string)
- If `Pattern` specified: Must match regex
- If `Enum` specified: Must be in allowed list
- Empty string `""` is valid unless `Required: true` and missing

**Examples**:

```yaml
# ✅ Valid
message: "Hello World"
message: "123"  # Converted to string
message: ""     # Valid if Required: false

# ❌ Invalid
message: 42     # Number (if Pattern specified, conversion may fail validation)
message: null   # Null (unless not Required)
```

#### 2. Int Type

```go
"count": {
    Type:        "int",
    Required:    true,
    MinValue:    &[]float64{0}[0],   // Must be >= 0
    MaxValue:    &[]float64{100}[0], // Must be <= 100
}
```

**Validation**:
- JSON numbers (int or float64)
- Float values accepted if no fractional part (e.g., `10.0` → `10`)
- If `MinValue` specified: Must be >= MinValue
- If `MaxValue` specified: Must be <= MaxValue

**Conversion Rules**:
```go
// In Execute(), safe to cast:
count := int(inputs["count"].(float64))  // JSON numbers are float64

// Or use type assertion helper:
count, ok := inputs["count"].(float64)
if ok {
    countInt := int(count)
}
```

**Examples**:

```yaml
# ✅ Valid
count: 42
count: 10.0      # Accepted as 10
count: 0         # Valid if MinValue: 0

# ❌ Invalid
count: "42"      # String (type mismatch)
count: 10.5      # Fractional part (for int type)
count: -1        # Below MinValue
count: 101       # Above MaxValue
```

#### 3. Float Type

```go
"threshold": {
    Type:        "float",
    Required:    false,
    Default:     0.5,
    MinValue:    &[]float64{0.0}[0],
    MaxValue:    &[]float64{1.0}[0],
}
```

**Validation**:
- JSON numbers (int or float64)
- Integers auto-converted to float: `10` → `10.0`
- Range checks applied

**Examples**:

```yaml
# ✅ Valid
threshold: 0.75
threshold: 1      # Converted to 1.0
threshold: 0.0

# ❌ Invalid
threshold: "0.5"  # String
threshold: -0.1   # Below MinValue
threshold: 1.5    # Above MaxValue
```

#### 4. Bool Type

```go
"enabled": {
    Type:        "bool",
    Required:    false,
    Default:     true,
}
```

**Validation**:
- Only JSON boolean values accepted
- No string conversion (`"true"` is invalid)

**Examples**:

```yaml
# ✅ Valid
enabled: true
enabled: false

# ❌ Invalid
enabled: "true"   # String
enabled: 1        # Number
enabled: yes      # YAML boolean (depends on parser)
```

#### 5. Object Type

```go
"config": {
    Type:        "object",
    Required:    false,
    Default:     map[string]interface{}{},
}
```

**Validation**:
- Must be JSON object: `{"key": "value"}`
- Nested validation not enforced (validate in Execute() if needed)

**Usage in Execute()**:

```go
config := inputs["config"].(map[string]interface{})
if apiKey, ok := config["api_key"].(string); ok {
    // Use apiKey
}
```

**Examples**:

```yaml
# ✅ Valid
config:
  api_key: secret123
  timeout: 30

config: {}  # Empty object

# ❌ Invalid
config: "string"   # Not an object
config: [1, 2]     # Array, not object
```

#### 6. Array Type

```go
"items": {
    Type:        "array",
    Required:    false,
    Default:     []interface{}{},
}
```

**Validation**:
- Must be JSON array: `[1, 2, 3]`
- Element type not enforced (validate in Execute())

**Usage in Execute()**:

```go
items := inputs["items"].([]interface{})
for i, item := range items {
    // Process each item
    // Type assert as needed: item.(string), item.(float64), etc.
}
```

**Examples**:

```yaml
# ✅ Valid
items:
  - apple
  - banana
  - cherry

items: [1, 2, 3]
items: []  # Empty array

# ❌ Invalid
items: "apple,banana"  # String, not array
items: {a: 1}          # Object, not array
```

### Advanced Validation Examples

#### Example 1: Email Validation

```go
"email": {
    Type:        "string",
    Required:    true,
    Pattern:     `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
    Description: "Valid email address",
}
```

#### Example 2: Port Number

```go
"port": {
    Type:        "int",
    Required:    false,
    Default:     8080,
    MinValue:    &[]float64{1}[0],
    MaxValue:    &[]float64{65535}[0],
    Description: "TCP port number (1-65535)",
}
```

#### Example 3: Log Level

```go
"log_level": {
    Type:        "string",
    Required:    false,
    Default:     "INFO",
    Enum:        []interface{}{"DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"},
    Description: "Logging level",
}
```

#### Example 4: Percentage

```go
"success_rate": {
    Type:        "float",
    Required:    true,
    MinValue:    &[]float64{0.0}[0],
    MaxValue:    &[]float64{100.0}[0],
    Description: "Success rate percentage (0-100)",
}
```

#### Example 5: File Path

```go
"file_path": {
    Type:        "string",
    Required:    true,
    Pattern:     `^(/[^/ ]+)+/?$`,  // Unix absolute path
    Description: "Absolute file path on target server",
}
```

### Custom Validation in Execute()

For complex validation beyond ParamSpec capabilities:

```go
func (n *MyNode) Execute(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error) {
    // Inputs are pre-validated by Activity layer
    
    // Additional business logic validation
    startDate := inputs["start_date"].(string)
    endDate := inputs["end_date"].(string)
    
    start, _ := time.Parse("2006-01-02", startDate)
    end, _ := time.Parse("2006-01-02", endDate)
    
    if end.Before(start) {
        // Return permanent error (don't retry)
        return nil, fmt.Errorf("end_date (%s) must be after start_date (%s)", endDate, startDate)
    }
    
    // Proceed with execution
    // ...
}
```

---

## Error Handling and Logging

### Error Classification

Waterflow distinguishes between **permanent** and **temporary** errors for retry logic.

#### Permanent Errors (Non-Retryable)

Return when:
- Invalid input (should be caught by validation, but business logic errors)
- Resource not found (404)
- Permission denied (403)
- Client errors (4xx HTTP status)

**How to Return**:

```go
// Option 1: Plain error (treated as permanent by default if no retry hint)
return nil, fmt.Errorf("invalid configuration: %s", reason)

// Option 2: Wrap with context
return nil, fmt.Errorf("resource not found: %w", err)

// Option 3: Custom error type
type ValidationError struct {
    Field string
    Reason string
}
func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Reason)
}
return nil, &ValidationError{Field: "email", Reason: "invalid format"}
```

**Examples**:

```go
// File not found
if os.IsNotExist(err) {
    return nil, fmt.Errorf("file %s does not exist", filePath)
}

// HTTP 4xx
if resp.StatusCode >= 400 && resp.StatusCode < 500 {
    return nil, fmt.Errorf("client error %d: %s", resp.StatusCode, resp.Status)
}

// Invalid date range
if endDate.Before(startDate) {
    return nil, fmt.Errorf("end_date must be after start_date")
}
```

#### Temporary Errors (Retryable)

Return when:
- Network timeout
- Service unavailable (503)
- Connection refused
- Database deadlock
- Server errors (5xx HTTP status)

**How to Return**:

```go
// Return error directly (Temporal will retry)
return nil, err

// With context
return nil, fmt.Errorf("connection failed: %w", err)
```

**Examples**:

```go
// Network error
if err, ok := err.(net.Error); ok && err.Timeout() {
    return nil, fmt.Errorf("network timeout: %w", err)
}

// HTTP 5xx
if resp.StatusCode >= 500 {
    return nil, fmt.Errorf("server error %d: %s", resp.StatusCode, resp.Status)
}

// Database error
if isDatabaseDeadlock(err) {
    return nil, fmt.Errorf("database deadlock, will retry: %w", err)
}
```

### Logging Best Practices

Use `NodeResult.AddLog()` to record execution trail.

#### Log Levels (by Convention)

```go
// Informational (always include)
result.AddLog("Starting HTTP request to " + url)
result.AddLog(fmt.Sprintf("Completed in %s", duration))

// Warning (potential issues)
result.AddLog("WARNING: Using default timeout (30s)")
result.AddLog("WARNING: Retry attempt 2/3")

// Error (before returning error)
result.AddLog("ERROR: Connection refused: " + err.Error())
```

#### What to Log

**DO Log**:
- Execution start/end
- Important parameters (avoid secrets!)
- Progress milestones
- Retry attempts
- Warnings
- Error details

**DON'T Log**:
- Secrets, passwords, API keys
- PII (Personally Identifiable Information)
- Excessive debug information
- Binary data

**Example**:

```go
func (n *HTTPRequestNode) Execute(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error) {
    result := node.NewNodeResult()
    
    // Log start with key parameters (redact secrets)
    url := inputs["url"].(string)
    method := inputs["method"].(string)
    result.AddLog(fmt.Sprintf("Sending %s request to %s", method, url))
    
    // Log headers (redact Authorization)
    if headers, ok := inputs["headers"].(map[string]interface{}); ok {
        for k := range headers {
            if k == "Authorization" {
                result.AddLog("Header: Authorization = [REDACTED]")
            } else {
                result.AddLog(fmt.Sprintf("Header: %s = %v", k, headers[k]))
            }
        }
    }
    
    // Execute request
    resp, err := client.Do(req)
    if err != nil {
        result.AddLog("ERROR: Request failed: " + err.Error())
        return result, err
    }
    
    // Log result
    result.AddLog(fmt.Sprintf("Received response: %d %s", resp.StatusCode, resp.Status))
    result.AddLog(fmt.Sprintf("Response size: %d bytes", len(body)))
    
    return result, nil
}
```

### Error Handling Patterns

#### Pattern 1: Retry with Backoff

```go
const maxRetries = 3

for attempt := 0; attempt <= maxRetries; attempt++ {
    resp, err := doRequest(ctx)
    if err == nil {
        return resp, nil
    }
    
    if attempt < maxRetries {
        backoff := time.Duration(1<<attempt) * time.Second
        result.AddLog(fmt.Sprintf("Attempt %d/%d failed, retrying in %s", attempt+1, maxRetries, backoff))
        time.Sleep(backoff)
    }
}

return nil, fmt.Errorf("max retries exceeded: %w", err)
```

#### Pattern 2: Context Timeout Handling

```go
select {
case <-ctx.Done():
    result.AddLog("ERROR: Operation cancelled or timed out")
    return result, ctx.Err()
case result := <-resultChan:
    return result, nil
}
```

#### Pattern 3: Partial Success

```go
var errors []string
successCount := 0

for _, item := range items {
    if err := processItem(item); err != nil {
        errors = append(errors, fmt.Sprintf("Item %s: %v", item, err))
    } else {
        successCount++
    }
}

result.SetOutput("success_count", successCount)
result.SetOutput("failed_count", len(errors))

if len(errors) > 0 {
    result.AddLog(fmt.Sprintf("Partial failure: %d/%d succeeded", successCount, len(items)))
    for _, errMsg := range errors {
        result.AddLog("ERROR: " + errMsg)
    }
    return result, fmt.Errorf("%d items failed", len(errors))
}

return result, nil
```

---

## Testing Best Practices

### Unit Testing Methodology

All nodes must have comprehensive unit tests.

**Required Coverage**:
- ✅ Interface methods (Name, Version, Params, Metadata)
- ✅ Parameter validation (via Params())
- ✅ Execute() with valid inputs
- ✅ Execute() with edge cases
- ✅ Error conditions

**Recommended Coverage**: >80%

### Table-Driven Tests

Use table-driven tests for multiple scenarios.

**Template**:

```go
func TestMyNode_Execute(t *testing.T) {
    tests := []struct {
        name        string                      // Test case name
        inputs      map[string]interface{}      // Input parameters
        want        map[string]interface{}      // Expected outputs
        wantErr     bool                        // Expect error?
        errContains string                      // Error message substring
    }{
        {
            name: "Valid Request",
            inputs: map[string]interface{}{
                "url":    "https://api.example.com/users",
                "method": "GET",
            },
            want: map[string]interface{}{
                "status_code": 200,
            },
            wantErr: false,
        },
        {
            name: "Timeout Error",
            inputs: map[string]interface{}{
                "url":     "https://slow.example.com",
                "timeout": 1,
            },
            wantErr:     true,
            errContains: "timeout",
        },
        // More cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            n := &MyNode{}
            result, err := n.Execute(context.Background(), tt.inputs)
            
            if tt.wantErr {
                require.Error(t, err)
                if tt.errContains != "" {
                    assert.Contains(t, err.Error(), tt.errContains)
                }
                return
            }
            
            require.NoError(t, err)
            for key, expected := range tt.want {
                assert.Equal(t, expected, result.Outputs[key])
            }
        })
    }
}
```

### Test Case Design Guidelines

#### 1. Happy Path

```go
{
    name: "Successful Execution",
    inputs: map[string]interface{}{
        "param1": "valid_value",
        "param2": 42,
    },
    want: map[string]interface{}{
        "output1": "expected_result",
    },
    wantErr: false,
}
```

#### 2. Edge Cases

```go
{
    name: "Empty String Input",
    inputs: map[string]interface{}{
        "message": "",
    },
    // Define expected behavior
},
{
    name: "Minimum Value",
    inputs: map[string]interface{}{
        "count": 0,
    },
},
{
    name: "Maximum Value",
    inputs: map[string]interface{}{
        "count": 1000000,
    },
},
```

#### 3. Error Conditions

```go
{
    name: "Invalid URL Format",
    inputs: map[string]interface{}{
        "url": "not-a-url",
    },
    wantErr:     true,
    errContains: "invalid URL",
},
{
    name: "Network Timeout",
    inputs: map[string]interface{}{
        "url":     "https://slow.example.com",
        "timeout": 1,
    },
    wantErr:     true,
    errContains: "timeout",
},
```

#### 4. Boundary Conditions

```go
{
    name: "Just Below Max Length",
    inputs: map[string]interface{}{
        "message": strings.Repeat("a", 999),
    },
},
{
    name: "Exactly Max Length",
    inputs: map[string]interface{}{
        "message": strings.Repeat("a", 1000),
    },
},
```

### Integration Testing

Test node loading and registration.

**Example**:

```go
func TestNodePluginLoading(t *testing.T) {
    // Build plugin
    cmd := exec.Command("make", "build")
    err := cmd.Run()
    require.NoError(t, err, "Failed to build plugin")
    
    // Load plugin
    plug, err := plugin.Open("./greeter.so")
    require.NoError(t, err, "Failed to load plugin")
    
    // Lookup Register function
    symbol, err := plug.Lookup("Register")
    require.NoError(t, err, "Register function not found")
    
    // Call Register()
    registerFunc, ok := symbol.(func() node.Node)
    require.True(t, ok, "Register has wrong signature")
    
    n := registerFunc()
    
    // Verify interface
    assert.Equal(t, "custom/greeter", n.Name())
    assert.Equal(t, "v1", n.Version())
    assert.NotNil(t, n.Params())
}
```

### Coverage Targets

Measure coverage with:

```bash
make coverage
```

**Target**: >80% coverage

**Coverage Breakdown**:
- Interface methods: 100% (trivial to test)
- Execute() logic: >90%
- Error handling: >70%
- Edge cases: >60%

**Viewing Coverage**:

```bash
make coverage
# Opens coverage.html in browser
```

**Example Output**:

```
custom/greeter/main.go:15:  Name            100.0%
custom/greeter/main.go:19:  Version         100.0%
custom/greeter/main.go:23:  Params          100.0%
custom/greeter/main.go:48:  Execute         95.2%
custom/greeter/main.go:78:  Metadata        100.0%
custom/greeter/main.go:95:  generateGreeting 91.7%
------------------------------------------------------
Total Coverage: 93.8%
```

---


## Build and Deploy

### Compilation Requirements

#### System Requirements

| Requirement | Value | Verification Command |
|-------------|-------|---------------------|
| **Go Version** | 1.22+ | `go version` |
| **CGO** | Enabled | `go env CGO_ENABLED` → `1` |
| **Platform** | Linux/macOS | `uname -s` |
| **Architecture** | amd64/arm64 | `uname -m` |

**Windows Limitation**: Go plugins are **not supported** on Windows. Use Linux/macOS or WSL2.

#### Verify Environment

```bash
#!/bin/bash
# scripts/verify-build-env.sh

echo "=== Verifying Build Environment ==="

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
if [[ "$GO_VERSION" < "1.22" ]]; then
    echo "❌ Go version must be 1.22+, found: $GO_VERSION"
    exit 1
fi
echo "✅ Go version: $GO_VERSION"

# Check CGO
if [[ "$(go env CGO_ENABLED)" != "1" ]]; then
    echo "❌ CGO must be enabled"
    echo "   Run: export CGO_ENABLED=1"
    exit 1
fi
echo "✅ CGO enabled"

# Check platform
PLATFORM=$(uname -s)
if [[ "$PLATFORM" != "Linux" && "$PLATFORM" != "Darwin" ]]; then
    echo "❌ Unsupported platform: $PLATFORM"
    echo "   Go plugins require Linux or macOS"
    exit 1
fi
echo "✅ Platform: $PLATFORM"

echo "✅ Build environment ready"
```

### Build Process

#### Makefile Template

```makefile
# Makefile for Custom Node Plugin

# ===== Configuration =====
PLUGIN_NAME := greeter
CATEGORY := custom
GO_FILES := $(shell find . -name '*.go' -not -name '*_test.go')
TEST_FILES := $(shell find . -name '*_test.go')

# Build paths
BUILD_DIR := .
PLUGIN_FILE := $(BUILD_DIR)/$(PLUGIN_NAME).so
INSTALL_DIR := /opt/waterflow/plugins/$(CATEGORY)

# Go build flags
GO := go
CGO_ENABLED := 1
BUILD_FLAGS := -buildmode=plugin
LDFLAGS := -w -s  # Strip debug info for smaller size

# ===== Targets =====

.PHONY: all check build test coverage integration-test install clean

all: check build test

# Check environment
check:
@echo "=== Checking Build Environment ==="
@which go > /dev/null || (echo "❌ Go not installed"; exit 1)
@echo "✅ Go version: $$(go version)"
@[ "$(shell go env CGO_ENABLED)" = "1" ] || (echo "❌ CGO disabled. Run: export CGO_ENABLED=1"; exit 1)
@echo "✅ CGO enabled"
@echo "✅ Environment ready"

# Build plugin
build: check
@echo "=== Building Plugin ==="
CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(PLUGIN_FILE) .
@echo "✅ Built: $(PLUGIN_FILE) ($$(du -h $(PLUGIN_FILE) | cut -f1))"

# Run unit tests
test:
@echo "=== Running Unit Tests ==="
$(GO) test -v -race -coverprofile=coverage.out ./...
@echo "✅ Unit tests passed"

# Generate coverage report
coverage: test
@echo "=== Generating Coverage Report ==="
$(GO) tool cover -html=coverage.out -o coverage.html
@$(GO) tool cover -func=coverage.out | grep total
@echo "✅ Coverage report: coverage.html"

# Run integration tests (plugin loading)
integration-test: build
@echo "=== Running Integration Tests ==="
$(GO) test -v -tags=integration -run TestPluginLoading ./...
@echo "✅ Integration tests passed"

# Install to Agent plugins directory
install: build
@echo "=== Installing Plugin ==="
@sudo mkdir -p $(INSTALL_DIR)
@sudo cp $(PLUGIN_FILE) $(INSTALL_DIR)/
@echo "✅ Installed to $(INSTALL_DIR)/$(PLUGIN_NAME).so"
@echo "🔄 Restart waterflow-agent to load: sudo systemctl restart waterflow-agent"

# Clean build artifacts
clean:
@echo "=== Cleaning Build Artifacts ==="
rm -f $(PLUGIN_FILE) coverage.out coverage.html
@echo "✅ Cleaned"

# Help
help:
@echo "Available targets:"
@echo "  make check            - Verify build environment"
@echo "  make build            - Compile plugin (.so file)"
@echo "  make test             - Run unit tests"
@echo "  make coverage         - Generate coverage report"
@echo "  make integration-test - Test plugin loading"
@echo "  make install          - Install to Agent (requires sudo)"
@echo "  make clean            - Remove build artifacts"
@echo "  make all              - check + build + test"
```

#### Build Steps

```bash
# Step 1: Verify environment
make check

# Step 2: Build plugin
make build
# Output: greeter.so (typically 3-10 MB)

# Step 3: Run tests
make test

# Step 4: Check coverage
make coverage
# Opens coverage.html

# Step 5: Integration test
make integration-test

# Step 6: Install
sudo make install
```

### Deployment Flow to Agent

#### Deployment Paths

```
Plugin Development → Build → Deploy → Agent Load

examples/plugins/greeter/
├── main.go              (Source code)
├── main_test.go         (Tests)
├── Makefile             (Build automation)
└── greeter.so           (Built plugin) ──┐
                                           │
                                           ▼
                         /opt/waterflow/plugins/custom/
                         └── greeter.so    (Deployed) ──┐
                                                         │
                                                         ▼
                         Agent Memory:
                         NodeRegistry["custom/greeter@v1"] = GreeterNode
```

#### Deployment Script

```bash
#!/bin/bash
# scripts/deploy-plugin.sh

set -e

PLUGIN_NAME="greeter"
CATEGORY="custom"
BUILD_DIR="examples/plugins/${PLUGIN_NAME}"
PLUGIN_FILE="${BUILD_DIR}/${PLUGIN_NAME}.so"
INSTALL_DIR="/opt/waterflow/plugins/${CATEGORY}"
AGENT_SERVICE="waterflow-agent"

echo "=== Deploying ${CATEGORY}/${PLUGIN_NAME} Plugin ==="

# Step 1: Build
cd "$BUILD_DIR"
echo "📦 Building plugin..."
make build

# Step 2: Verify plugin file
if [[ ! -f "$PLUGIN_FILE" ]]; then
    echo "❌ Plugin file not found: $PLUGIN_FILE"
    exit 1
fi
echo "✅ Plugin built: $(du -h $PLUGIN_FILE | cut -f1)"

# Step 3: Create install directory
echo "📁 Creating installation directory..."
sudo mkdir -p "$INSTALL_DIR"

# Step 4: Copy plugin
echo "📋 Copying plugin to $INSTALL_DIR..."
sudo cp "$PLUGIN_FILE" "$INSTALL_DIR/"
sudo chmod 644 "${INSTALL_DIR}/${PLUGIN_NAME}.so"

# Step 5: Verify installation
if [[ -f "${INSTALL_DIR}/${PLUGIN_NAME}.so" ]]; then
    echo "✅ Installed: ${INSTALL_DIR}/${PLUGIN_NAME}.so"
else
    echo "❌ Installation failed"
    exit 1
fi

# Step 6: Restart Agent (optional, for non-hot-reload deployments)
read -p "🔄 Restart waterflow-agent service? [y/N] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    sudo systemctl restart "$AGENT_SERVICE"
    sleep 2
    sudo systemctl status "$AGENT_SERVICE" --no-pager
    echo "✅ Agent restarted"
else
    echo "⏭️  Skipped restart (hot-reload will detect new plugin)"
fi

echo "✅ Deployment complete"
echo "📝 Use in workflows: uses: ${CATEGORY}/${PLUGIN_NAME}@v1"
```

### Hot-Reload Mechanism

Waterflow Agent supports hot-reloading plugins without restarts.

#### How It Works

```
┌──────────────────────────────────────┐
│  Plugin Directory Watcher (fsnotify) │
│  Monitors: /opt/waterflow/plugins/** │
└──────────┬───────────────────────────┘
           │
           ▼ (File Change Event)
    ┌──────────────────┐
    │ Plugin Manager   │
    ├──────────────────┤
    │ 1. Unload old    │ ← Close plugin handle
    │ 2. Load new .so  │ ← plugin.Open()
    │ 3. Call Register │ ← Get node.Node
    │ 4. Update registry│
    └──────────────────┘
           │
           ▼
    NodeRegistry Updated
    (Next workflow uses new version)
```

#### Manual Hot-Reload

```bash
# Deploy plugin
sudo cp greeter.so /opt/waterflow/plugins/custom/

# Agent detects change automatically (check logs)
sudo journalctl -u waterflow-agent -f

# Expected log:
# INFO Plugin updated: custom/greeter@v1
# INFO Reloaded plugin: /opt/waterflow/plugins/custom/greeter.so
```

#### Disable Hot-Reload

In `config.yaml`:

```yaml
agent:
  plugin_hot_reload: false  # Require restart for plugin changes
```

### Version Management

#### Strategy 1: Multiple Versions in Separate Files

```
/opt/waterflow/plugins/http/
├── request-v1.so    # http/request@v1
├── request-v2.so    # http/request@v2
└── request.so       # Symlink → request-v2.so (latest)
```

**Workflow Usage**:

```yaml
# Explicit version
- uses: http/request@v2

# Latest version (follows symlink)
- uses: http/request
```

#### Strategy 2: Same File, Different Interface

```go
// Version returns current version
func (n *HTTPRequestNode) Version() string {
    return "v2"
}
```

Rebuild and replace `request.so`. Old workflows still reference `@v1` but get v2 behavior (breaking change!).

**Recommendation**: Use Strategy 1 for breaking changes.

#### Version Upgrade Guide

```markdown
## Upgrading from v1 to v2

### Breaking Changes
- Parameter `url` now requires HTTPS (HTTP removed for security)
- Parameter `retry` renamed to `retry_count`
- Output `status` removed (use `status_code` instead)

### Migration

**Before (v1)**:
\```yaml
- uses: http/request@v1
  with:
    url: http://example.com
    retry: 3
\```

**After (v2)**:
\```yaml
- uses: http/request@v2
  with:
    url: https://example.com  # ← HTTPS required
    retry_count: 3            # ← Renamed parameter
\```

### Accessing Outputs

**Before (v1)**:
\```yaml
\${{ steps.my_step.outputs.status }}
\```

**After (v2)**:
\```yaml
\${{ steps.my_step.outputs.status_code }}
\```
```

---

## Best Practices and Troubleshooting

### Code Organization

#### Recommended Project Structure

```
examples/plugins/my-node/
├── main.go              # Node implementation
├── main_test.go         # Unit tests
├── integration_test.go  # Integration tests (plugin loading)
├── Makefile             # Build automation
├── README.md            # User documentation
├── go.mod               # Dependencies
├── .gitignore           # Ignore greeter.so, coverage files
└── testdata/            # Test fixtures
    ├── input.json
    └── expected-output.json
```

#### File Naming Conventions

- **Main**: `main.go` (required for plugins, package must be `main`)
- **Tests**: `*_test.go`
- **Build tags**: `// +build integration` for integration tests
- **Makefile**: Uppercase `Makefile` (GNU make convention)

#### Package Structure

```go
// ✅ Correct: package main (required for Go plugins)
package main

import (
    "github.com/Websoft9/waterflow/pkg/dsl/node"
    // Other imports
)

// Node implementation
type MyNode struct{}

// Register exports the node (required for plugin system)
func Register() node.Node {
    return &MyNode{}
}
```

### Performance Optimization

#### 1. Reuse Resources

```go
type HTTPRequestNode struct {
    client *http.Client  // Shared HTTP client
}

func init() {
    // Initialize once (when plugin loads)
    defaultNode = &HTTPRequestNode{
        client: &http.Client{
            Timeout: 30 * time.Second,
            Transport: &http.Transport{
                MaxIdleConns:        100,
                MaxIdleConnsPerHost: 10,
            },
        },
    }
}

func Register() node.Node {
    return defaultNode
}
```

#### 2. Context-Aware Operations

```go
// ✅ Always use ctx parameter for long operations
resp, err := http.NewRequestWithContext(ctx, "GET", url, nil)

// ❌ Don't ignore context
resp, err := http.Get(url)  // Can't be cancelled
```

#### 3. Avoid Memory Leaks

```go
// ✅ Close resources
defer resp.Body.Close()
defer file.Close()

// ✅ Limit buffer sizes
bodyBytes, err := ioutil.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))  // Max 10MB
```

### Common Issues and Solutions

#### Issue 1: Plugin Fails to Load

**Symptom**:
```
ERROR Failed to load plugin: plugin.Open("greeter"): plugin was built with a different version of package ...
```

**Cause**: Plugin built with different Go version or dependencies than Agent.

**Solution**:
```bash
# Check Agent's Go version
/opt/waterflow/bin/waterflow-agent version

# Rebuild plugin with matching version
go version  # Should match Agent
make clean build
```

#### Issue 2: CGO Disabled

**Symptom**:
```
ERROR -buildmode=plugin not supported when -buildmode=pie
```

**Cause**: `CGO_ENABLED=0`

**Solution**:
```bash
export CGO_ENABLED=1
make build
```

#### Issue 3: Type Assertion Panic

**Symptom**:
```
panic: interface conversion: interface {} is float64, not int
```

**Cause**: JSON numbers are `float64`, not `int`.

**Solution**:
```go
// ❌ Wrong
count := inputs["count"].(int)

// ✅ Correct
count := int(inputs["count"].(float64))

// ✅ Or safe assertion
if countFloat, ok := inputs["count"].(float64); ok {
    count := int(countFloat)
}
```

#### Issue 4: Plugin Not Found in Workflow

**Symptom**:
```
ERROR Node not found: custom/greeter@v1
```

**Cause**: Plugin not installed or Agent not restarted.

**Solution**:
```bash
# Check installation
ls -lh /opt/waterflow/plugins/custom/greeter.so

# Check Agent logs
sudo journalctl -u waterflow-agent | grep greeter

# Restart Agent
sudo systemctl restart waterflow-agent
```

#### Issue 5: Test Failures After Build

**Symptom**:
```
--- FAIL: TestGreeterNode_Execute (0.00s)
    Error: Not equal: expected "Good morning, Alice!", actual "Buenos días, Alice!"
```

**Cause**: Test expectations don't match updated implementation.

**Solution**:
- Review changes to `Execute()` logic
- Update test expectations in `main_test.go`
- Run `make coverage` to identify untested paths

### Debugging Techniques

#### 1. Add Debug Logs

```go
func (n *MyNode) Execute(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error) {
    result := node.NewNodeResult()
    
    // Log all inputs (for debugging)
    for k, v := range inputs {
        result.AddLog(fmt.Sprintf("DEBUG: Input %s = %v (type: %T)", k, v, v))
    }
    
    // Implementation
    // ...
    
    return result, nil
}
```

#### 2. Use Test-Driven Development

```bash
# Write failing test first
go test -v -run TestMyFeature
# FAIL: TestMyFeature

# Implement feature
# Edit main.go

# Rerun test
go test -v -run TestMyFeature
# PASS: TestMyFeature
```

#### 3. Inspect Plugin Loading

```go
// integration_test.go
func TestPluginInspection(t *testing.T) {
    plug, err := plugin.Open("./greeter.so")
    require.NoError(t, err)
    
    symbol, err := plug.Lookup("Register")
    require.NoError(t, err)
    
    registerFunc := symbol.(func() node.Node)
    n := registerFunc()
    
    // Inspect node
    t.Logf("Name: %s", n.Name())
    t.Logf("Version: %s", n.Version())
    t.Logf("Params: %+v", n.Params())
    t.Logf("Metadata: %+v", n.Metadata())
}
```

#### 4. Check Agent Logs

```bash
# Real-time logs
sudo journalctl -u waterflow-agent -f

# Search for errors
sudo journalctl -u waterflow-agent | grep ERROR

# Filter by node name
sudo journalctl -u waterflow-agent | grep greeter
```

---

## Advanced Topics

### Dependency Management

**Add external packages**:

```bash
cd examples/plugins/my-node
go mod init github.com/Websoft9/waterflow-plugin-my-node
go get github.com/some/library@v1.2.3
```

**go.mod Example**:

```go
module github.com/Websoft9/waterflow-plugin-greeter

go 1.22

require (
    github.com/Websoft9/waterflow v0.1.0
)

replace github.com/Websoft9/waterflow => ../../..
```

### Sharing Code Between Nodes

Create shared library:

```
plugins/shared/
├── utils.go
└── validators.go

plugins/my-node/
└── main.go  (imports ../shared)
```

**Caution**: All plugins must use same version of shared code.

### CI/CD Integration

**GitHub Actions Example**:

```yaml
name: Build Plugin

on:
  push:
    paths:
      - 'examples/plugins/greeter/**'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Build Plugin
        run: |
          cd examples/plugins/greeter
          make check build test coverage
      
      - name: Upload Artifact
        uses: actions/upload-artifact@v3
        with:
          name: greeter-plugin
          path: examples/plugins/greeter/greeter.so
```

---

## Summary

**Key Takeaways**:

1. ✅ **Interface First**: Implement all 5 methods (Name, Version, Params, Execute, Metadata)
2. ✅ **Validation**: Define ParamSpec thoroughly, let Activity layer validate
3. ✅ **Error Handling**: Classify permanent vs temporary errors for retry logic
4. ✅ **Testing**: Achieve >80% coverage with table-driven tests
5. ✅ **Build**: Use CGO_ENABLED=1, -buildmode=plugin
6. ✅ **Deploy**: Install to /opt/waterflow/plugins/<category>/, hot-reload supported
7. ✅ **Logging**: Use AddLog() for execution trail, avoid logging secrets
8. ✅ **Context**: Respect ctx for cancellation and timeouts

**Next Steps**:

- Review [Greeter Plugin Example](../../examples/plugins/greeter/) for complete reference
- Read [ADR-0003: Plugin-Based Node System](../adr/0003-plugin-based-node-system.md)
- Check [Node Development Quickstart](../quick-start.md#developing-custom-nodes)

---

**Document Version**: 2.0  
**Last Updated**: 2024-01-16  
**Maintainers**: Waterflow Core Team

