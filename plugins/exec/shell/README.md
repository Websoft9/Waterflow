# exec/shell Node Plugin

Shell command execution node for Waterflow workflows.

## Description

The `exec/shell` node executes shell commands on the agent server, capturing stdout, stderr, and exit codes. It's the most fundamental node in the Waterflow plugin system.

## Category

`exec` - Command execution

## Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `command` | string | ✅ Yes | - | Shell command to execute |
| `args` | array | No | - | Command arguments (string array) |
| `env` | object | No | - | Environment variables (key-value pairs) |
| `workdir` | string | No | - | Working directory path |
| `timeout` | int | No | 60 | Timeout in seconds (1-3600) |

## Outputs

| Output | Type | Description |
|--------|------|-------------|
| `stdout` | string | Standard output |
| `stderr` | string | Standard error |
| `exit_code` | int | Process exit code |
| `duration_ms` | int | Execution duration in milliseconds |

## Examples

### Basic Command

```yaml
steps:
  - name: Simple echo
    uses: exec/shell@v1
    with:
      command: echo "Hello Waterflow"
```

### Command with Arguments

```yaml
steps:
  - name: List files
    uses: exec/shell@v1
    with:
      command: ls
      args: ["-la", "/tmp"]
      timeout: 10
```

### Environment Variables

```yaml
steps:
  - name: Use custom env
    uses: exec/shell@v1
    with:
      command: echo "API_KEY=$API_KEY"
      env:
        API_KEY: "secret-key-123"
        ENV: "production"
```

### Working Directory

```yaml
steps:
  - name: Create temp file
    uses: exec/shell@v1
    with:
      command: touch test.txt
      workdir: /tmp
```

### Error Handling

```yaml
steps:
  - name: Timeout example
    uses: exec/shell@v1
    with:
      command: sleep 100
      timeout: 5
    continue-on-error: true
  
  - name: Command failure
    uses: exec/shell@v1
    with:
      command: exit 1
    continue-on-error: true
```

## Security Considerations

### Command Injection Protection

The shell node implements safety measures against command injection:

1. **Arguments Escaping**: All `args` values are shell-escaped using single quotes
   - Input: `args: ["file.txt; rm -rf /"]`
   - Safe output: Command receives `'file.txt; rm -rf /'` as literal string

2. **Environment Variable Sanitization**: Newlines and null bytes are removed from env values
   - Prevents environment variable injection attacks

3. **User Responsibility**: The `command` parameter is executed as-is
   - ⚠️ **WARNING**: Only use trusted input in `command` parameter
   - ❌ **NEVER** pass unsanitized user input directly to `command`
   - ✅ **DO** validate and whitelist commands in your application layer

### Best Practices

```yaml
# ❌ UNSAFE: Direct user input
steps:
  - uses: exec/shell@v1
    with:
      command: ${{ user.input }}  # NEVER DO THIS!

# ✅ SAFE: Parameterized with args
steps:
  - uses: exec/shell@v1
    with:
      command: cat
      args: [${{ user.filename }}]  # Safe - properly escaped

# ✅ SAFE: Whitelisted commands
steps:
  - uses: exec/shell@v1
    with:
      command: |
        case "$ACTION" in
          start) systemctl start app ;;
          stop)  systemctl stop app ;;
          *) exit 1 ;;
        esac
      env:
        ACTION: ${{ user.action }}  # Limited to start/stop
```

## Error Handling

### Exit Code ≠ 0

When a command fails (exit code ≠ 0), the node returns an error containing stdout and stderr. Use `continue-on-error: true` to ignore failures.

### Timeout

Commands exceeding the timeout are automatically terminated. The timeout error is classified as a temporary error (retryable).

### Command Not Found

If the command doesn't exist, a permanent error (non-retryable) is returned.

## Building

```bash
# Build plugin
make build

# Run tests
make test

# Install to agent
make install
```

## Requirements

- Go 1.22+
- CGO_ENABLED=1
- Linux or macOS

## FAQ

**Q: How do I pass arguments with spaces?**  
A: Use the `args` array:
```yaml
with:
  command: echo
  args: ["hello world", "test"]
```

**Q: Can I chain multiple commands?**  
A: Yes, use shell operators:
```yaml
with:
  command: "cd /tmp && ls -la && pwd"
```

**Q: How do I handle command failures?**  
A: Add `continue-on-error: true` to the step, or check the exit code in a subsequent step.

**Q: What happens if timeout is reached?**  
A: The process is automatically terminated, and a timeout error is returned. This is a temporary error and can be retried.

## Version

v1

## License

Same as Waterflow project
