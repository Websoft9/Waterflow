# Template Node Plugin

This is a template for creating Waterflow node plugins.

## Quick Start (5 minutes)

1. Copy this template:
```bash
cp -r examples/plugins/template examples/plugins/mynode
cd examples/plugins/mynode
```

2. Modify `main.go`:
   - Change type name from `TemplateNode` to `MyNode`
   - Update `Name()` to return your node identifier (e.g., "exec/mycommand")
   - Define your parameters in `Params()`
   - Implement your logic in `Execute()`

3. Build the plugin:
```bash
export CGO_ENABLED=1
make build
```

4. Test it:
```bash
make test
```

## Development Guide

### Node Interface

All nodes must implement these 5 methods:

- `Name() string` - Unique identifier (format: category/name)
- `Version() string` - Semantic version (e.g., "v1", "v1.2.3")
- `Params() map[string]ParamSpec` - Input parameter definitions
- `Execute(ctx, inputs) (*NodeResult, error)` - Main logic
- `Metadata() NodeMetadata` - Description and schemas

### Parameter Types

Supported types: string, int, float, bool, object, array

Advanced validation:
- `Enum` - Restrict to specific values
- `MinValue` / `MaxValue` - Numeric ranges
- `Pattern` - Regex validation for strings

### Best Practices

1. **Always validate inputs** using `node.ValidateInputs()`
2. **Handle context cancellation** with `ctx.Done()`
3. **Log execution steps** using `result.AddLog()`
4. **Record duration** with `result.Duration`
5. **Return structured outputs** via `result.SetOutput()`

## Compilation Requirements

- **Go Version**: Must match Agent's Go version
- **CGO**: Must be enabled (`export CGO_ENABLED=1`)
- **Platform**: Linux or macOS (Windows not supported)

## Common Issues

**Q: "plugin was built with a different version of package"**
A: Rebuild with exact same Go version as Agent

**Q: "CGO_ENABLED must be 1"**  
A: Run `export CGO_ENABLED=1` before building

**Q: "undefined: node.Node"**
A: Ensure import path is correct: `github.com/Websoft9/waterflow/pkg/dsl/node`
