# Echo Node Plugin

A simple example node that echoes input to output.

## Building

```bash
export CGO_ENABLED=1
make build
```

This creates `echo.so` plugin file.

## Testing

```bash
make test
```

Expected output:
- All tests pass
- Coverage >80%

## Loading

The plugin can be loaded by PluginManager:

```go
pm := agent.NewPluginManager("/path/to/plugins", logger)
pm.LoadPlugins()
```

## Usage in Workflow

```yaml
steps:
  - name: Echo Message
    node: flow/echo@v1
    with:
      message: "Hello, Waterflow!"
```

Output:
```json
{
  "message": "Hello, Waterflow!"
}
```
