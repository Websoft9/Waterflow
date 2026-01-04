# Node Development Guide

Complete guide for developing Waterflow node plugins.

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
