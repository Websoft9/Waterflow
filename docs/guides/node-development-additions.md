## Core Concepts

This section provides a deep dive into the fundamental concepts of the Waterflow node system.

### Node Architecture

Nodes are the building blocks of Waterflow workflows. Each node:

- **Is a Go Plugin** - Compiled as `.so` file, dynamically loaded by Agent
- **Implements Node Interface** - 5 required methods (Name, Version, Params, Execute, Metadata)
- **Runs in Agent Process** - Shares same Go runtime with Agent
- **Is Stateless** - Execute() can be called multiple times concurrently
- **Has Unique Identity** - Identified by `category/name@version`

**Architecture Diagram**:

```
┌─────────────────────────────────────────────┐
│ Waterflow Server (Parse & Schedule)         │
└───────────────────┬─────────────────────────┘
                    │ gRPC
        ┌───────────▼───────────┐
        │  Temporal Server      │
        └───────────┬───────────┘
                    │ Long Polling
        ┌───────────▼───────────┐
        │  Waterflow Agent      │
        │  ┌─────────────────┐  │
        │  │ Plugin Manager  │  │
        │  ├─────────────────┤  │
        │  │ exec/shell.so   │  │ ← Node Plugins
        │  │ http/request.so │  │
        │  │ custom/greeter.so│ │
        │  └─────────────────┘  │
        └───────────────────────┘
```

**Reference**: [ADR-0003: Plugin-Based Node System](../adr/0003-plugin-based-node-system.md)

### Node Interface Deep Dive

#### 1. Name() string

Returns the unique identifier for this node in `category/name` format.

**Format**:
- Pattern: `^[a-z]+/[a-z][a-z0-9-]*$`
- Example: `exec/shell`, `docker/compose`, `custom/greeter`

**Valid Categories**:
- `exec` - Execution (shell commands, scripts)
- `docker` - Docker operations
- `http` - HTTP/REST API calls
- `file` - File operations
- `flow` - Control flow (loops, conditions)
- `custom` - User-defined nodes

**Naming Best Practices**:
- Use lowercase, hyphenated names
- Be descriptive but concise
- Avoid abbreviations unless well-known
- Examples: `http/request` (not `http/req`), `file/transfer` (not `file/xfer`)

#### 2. Version() string

Returns the semantic version of this node.

**Format**: `vX.Y.Z` or `vX` (where X, Y, Z are integers)

**Examples**:
- `v1` - Major version only (recommended for most nodes)
- `v1.2.3` - Full semantic versioning

**Version Strategy**:
- **v1** - Initial stable release
- **v2** - Breaking changes to parameters or behavior
- **v1.1.0** - New optional parameters (backward compatible)
- **v1.0.1** - Bug fixes (no interface changes)

**Multiple Versions**:
- Nodes can have multiple versions simultaneously loaded
- Workflows specify version: `uses: http/request@v2`
- Allows gradual migration to new versions

#### 3. Params() map[string]ParamSpec

Defines the input parameters accepted by this node.

**ParamSpec Fields**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `Type` | string | ✅ | Data type: `string`, `int`, `float`, `bool`, `object`, `array` |
| `Required` | bool | ❌ | Whether parameter must be provided (default: false) |
| `Description` | string | ❌ | Human-readable description |
| `Default` | interface{} | ❌ | Default value if not provided |
| `Pattern` | string | ❌ | Regex pattern (for `string` type) |
| `Enum` | []interface{} | ❌ | Allowed values (any type) |
| `MinValue` | *float64 | ❌ | Minimum value (for `int`/`float`) |
| `MaxValue` | *float64 | ❌ | Maximum value (for `int`/`float`) |

**Example - Complex Parameter Definition**:

```go
func (n *HTTPRequestNode) Params() map[string]node.ParamSpec {
    min, max := 1.0, 3600.0
    return map[string]node.ParamSpec{
        "url": {
            Type:        "string",
            Required:    true,
            Description: "HTTP request URL",
            Pattern:     `^https?://[^\s]+$`, // URL格式验证
        },
        "method": {
            Type:        "string",
            Required:    false,
            Default:     "GET",
            Description: "HTTP method",
            Enum:        []interface{}{"GET", "POST", "PUT", "DELETE", "PATCH"},
        },
        "timeout": {
            Type:        "int",
            Required:    false,
            Default:     30,
            Description: "Request timeout in seconds",
            MinValue:    &min,
            MaxValue:    &max,
        },
        "headers": {
            Type:        "object",
            Required:    false,
            Description: "HTTP headers as key-value pairs",
        },
        "retry_count": {
            Type:        "int",
            Required:    false,
            Default:     3,
            Description: "Number of retry attempts",
        },
    }
}
```

#### 4. Execute(ctx, inputs) (*NodeResult, error)

Executes the node's logic with given inputs.

**Parameters**:
- `ctx context.Context` - Cancellation, timeout, and deadline control
- `inputs map[string]interface{}` - Validated parameter values

**Return Values**:
- `*NodeResult` - Execution result (outputs, logs, duration)
- `error` - Execution error (nil if successful)

**Execution Flow**:

```go
func (n *MyNode) Execute(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error) {
    // 1. Start timing
    start := time.Now()
    
    // 2. Extract inputs (already validated)
    param1 := inputs["param1"].(string)
    
    // 3. Create result object
    result := node.NewNodeResult()
    result.AddLog("Execution started")
    
    // 4. Check context cancellation
    select {
    case <-ctx.Done():
        return nil, ctx.Err() // Workflow cancelled or timed out
    default:
    }
    
    // 5. Perform actual work
    output, err := doWork(ctx, param1)
    if err != nil {
        // Classify error (permanent vs temporary)
        if isPermanentError(err) {
            return nil, fmt.Errorf("permanent error: %w", err)
        }
        return nil, err // Temporary, will retry
    }
    
    // 6. Set outputs
    result.SetOutput("result", output)
    result.SetOutput("status", "success")
    
    // 7. Record duration and final log
    result.Duration = time.Since(start)
    result.AddLog("Execution completed successfully")
    
    return result, nil
}
```

**Context Handling**:
```go
// Respect context timeout
ctx, cancel := context.WithTimeout(ctx, timeout)
defer cancel()

// Check cancellation periodically in long-running operations
for i := 0; i < iterations; i++ {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // Continue work
    }
}
```

#### 5. Metadata() NodeMetadata

Returns descriptive information about the node.

**NodeMetadata Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `Description` | string | Human-readable description of what the node does |
| `Category` | string | Node category (exec, docker, http, file, flow, custom) |
| `InputSchema` | map[string]ParamSpec | Copy of Params() (for documentation) |
| `OutputSchema` | map[string]interface{} | Description of outputs produced |

**Example**:

```go
func (n *GreeterNode) Metadata() node.NodeMetadata {
    return node.NodeMetadata{
        Description: "Generates multi-language greetings based on time of day",
        Category:    "custom",
        InputSchema: n.Params(),
        OutputSchema: map[string]interface{}{
            "greeting":    "string - The generated greeting message",
            "language":    "string - The language used (en, zh, es, fr)",
            "time_of_day": "string - The detected or specified time period",
        },
    }
}
```

### NodeResult Structure

`NodeResult` is the structured return value from `Execute()`.

**Fields**:

```go
type NodeResult struct {
    Outputs  map[string]interface{} // Data produced by node
    Logs     []string               // Execution logs
    Duration time.Duration          // Execution time
    Metadata map[string]interface{} // Optional metadata
}
```

**Usage**:

```go
// Create result
result := node.NewNodeResult()

// Add outputs
result.SetOutput("key", "value")
result.SetOutput("count", 42)
result.SetOutput("success", true)

// Add logs
result.AddLog("Step 1 completed")
result.AddLog(fmt.Sprintf("Processed %d items", count))

// Set metadata (optional)
result.SetMetadata("execution_id", uuid.New().String())
result.SetMetadata("hostname", hostname)

// Record duration (automatically set by framework, but can override)
result.Duration = time.Since(start)
```

### Node Categories and Naming

**Built-in Categories**:

| Category | Purpose | Examples |
|----------|---------|----------|
| `exec` | Execute commands and scripts | `exec/shell`, `exec/script` |
| `docker` | Docker and container operations | `docker/exec`, `docker/compose` |
| `http` | HTTP/REST API calls | `http/request`, `http/webhook` |
| `file` | File and filesystem operations | `file/transfer`, `file/read` |
| `flow` | Control flow (loops, conditions, sleep) | `flow/sleep`, `flow/condition` |
| `custom` | User-defined custom nodes | `custom/greeter`, `custom/processor` |

**Naming Guidelines**:
- Use singular nouns: `http/request` not `http/requests`
- Use verbs for actions: `file/transfer`, `docker/exec`
- Keep names short (2-15 characters)
- Use hyphens for multi-word: `file/batch-process`

### Go Plugin Mechanism

Nodes are compiled as Go plugins using `-buildmode=plugin`.

**Register() Function**:

Every node plugin **must** export a `Register()` function:

```go
func Register() node.Node {
    return &MyNode{}
}
```

**Plugin Loading Process**:

1. **Discovery**: Agent scans `/opt/waterflow/plugins/` directory
2. **Load**: `plugin.Open("path/to/node.so")` loads the plugin
3. **Lookup**: `plugin.Lookup("Register")` finds the Register function
4. **Instantiate**: Calls `Register()` to get node instance
5. **Validate**: Calls `ValidateNode()` to check interface implementation
6. **Register**: Adds to NodeRegistry with key `category/name@version`

**Example - Plugin Loading**:

```go
// In Agent's Plugin Manager
p, err := plugin.Open("/opt/waterflow/plugins/custom/greeter.so")
if err != nil {
    return err
}

symRegister, err := p.Lookup("Register")
if err != nil {
    return err
}

registerFunc, ok := symRegister.(func() node.Node)
if !ok {
    return errors.New("invalid Register function signature")
}

nodeInstance := registerFunc()

// Validate node implements interface correctly
if err := node.ValidateNode(nodeInstance); err != nil {
    return err
}

// Add to registry
registry.Register(nodeInstance)
```

**Hot Reload**:

Agent uses `fsnotify` to watch plugin directory:

```go
// Agent watches /opt/waterflow/plugins/
watcher.Add("/opt/waterflow/plugins/")

for {
    select {
    case event := <-watcher.Events:
        if event.Op&fsnotify.Create == fsnotify.Create {
            // New plugin detected
            loadPlugin(event.Name)
        }
    }
}
```

### Version Management

**Multiple Versions**:
- Agents can load multiple versions of the same node
- Workflows specify version: `uses: http/request@v2`
- Default to latest if version omitted

**Version Compatibility**:
- **v1 → v2**: Breaking changes allowed
- **v1 → v1.1**: Must be backward compatible
- **v1.1 → v1.1.1**: Bug fixes only

**File Naming** (optional but recommended):
- `greeter-v1.so`
- `greeter-v2.so`
- Allows multiple versions in same directory

---

