# exec/script Node Plugin

Script file execution node for Waterflow workflows. Supports multiple interpreters including Bash, Python, Node.js, and Ruby.

## Description

The `exec/script` node executes script files or inline script content on the agent server. It provides a more flexible alternative to the `exec/shell` node for complex, multi-line scripts and multi-language support.

## Difference from Shell Node

| Feature | Shell Node (exec/shell) | Script Node (exec/script) |
|---------|------------------------|---------------------------|
| **Purpose** | Execute single commands | Execute script files or inline scripts |
| **Input** | `command` (single line) | `script_path` or `script_content` (multi-line) |
| **Interpreter** | Fixed: `sh -c` | Configurable: bash, python3, node, ruby, etc. |
| **Inline Support** | No | Yes (`script_content` parameter) |
| **File Support** | No | Yes (`script_path` parameter) |
| **Use Case** | Simple commands (`ls`, `echo`) | Complex logic, multi-language scripts |
| **Example** | `command: ls -la` | `script_content: "#!/bin/bash\n..."` |

**When to use:**
- ✅ **Use Script Node** for: Multi-line scripts, Python/Ruby/Node scripts, complex logic
- ✅ **Use Shell Node** for: Single commands, simple operations

## Category

`exec` - Command execution

## Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `script_path` | string | No* | - | Path to script file (mutually exclusive with script_content) |
| `script_content` | string | No* | - | Inline script content (mutually exclusive with script_path) |
| `interpreter` | string | No | `bash` | Script interpreter (bash, sh, python3, python, node, ruby) |
| `args` | array | No | - | Script arguments |
| `env` | object | No | - | Environment variables (key-value pairs) |
| `workdir` | string | No | - | Working directory path |
| `timeout` | int | No | 60 | Timeout in seconds (1-3600) |

**\* Either `script_path` or `script_content` must be specified (mutually exclusive)**

## Supported Interpreters

| Interpreter | Binary | File Extension | Notes |
|------------|--------|----------------|-------|
| `bash` | `/bin/bash` | `.sh` | Default interpreter |
| `sh` | `/bin/sh` | `.sh` | POSIX shell |
| `python3` | `/usr/bin/python3` | `.py` | Python 3.x |
| `python` | `/usr/bin/python` | `.py` | Python 2.x (legacy) |
| `node` | `/usr/bin/node` | `.js` | Node.js |
| `ruby` | `/usr/bin/ruby` | `.rb` | Ruby |

The node automatically checks if the interpreter exists on the system before execution.

## Outputs

| Output | Type | Description |
|--------|------|-------------|
| `stdout` | string | Standard output |
| `stderr` | string | Standard error |
| `exit_code` | int | Process exit code |
| `duration_ms` | int | Execution duration in milliseconds |
| `interpreter_used` | string | Interpreter that was used |

## Examples

### 1. Inline Bash Script

```yaml
steps:
  - name: Bash inline script
    uses: exec/script@v1
    with:
      script_content: |
        #!/bin/bash
        echo "Current user: $(whoami)"
        echo "Current directory: $(pwd)"
        echo "System info: $(uname -a)"
      interpreter: bash
```

### 2. Bash Script File

```yaml
steps:
  - name: Run deployment script
    uses: exec/script@v1
    with:
      script_path: /app/scripts/deploy.sh
      interpreter: bash
      timeout: 300
```

### 3. Python Script with Arguments

```yaml
steps:
  - name: Python data processing
    uses: exec/script@v1
    with:
      script_content: |
        #!/usr/bin/env python3
        import sys
        import os
        
        input_file = sys.argv[1]
        output_file = sys.argv[2]
        
        print(f"Processing {input_file} -> {output_file}")
        print(f"Environment: {os.getenv('ENV_NAME')}")
      interpreter: python3
      args: ["input.csv", "output.json"]
      env:
        ENV_NAME: "production"
```

### 4. Script with Environment Variables

```yaml
steps:
  - name: Configure application
    uses: exec/script@v1
    with:
      script_content: |
        #!/bin/bash
        echo "Database: $DB_HOST:$DB_PORT"
        echo "API Key: $API_KEY"
        echo "Environment: $APP_ENV"
      interpreter: bash
      env:
        DB_HOST: "db.example.com"
        DB_PORT: "5432"
        API_KEY: "secret-key-123"
        APP_ENV: "production"
```

### 5. Script in Custom Working Directory

```yaml
steps:
  - name: Build project
    uses: exec/script@v1
    with:
      script_content: |
        #!/bin/bash
        npm install
        npm run build
        ls -la dist/
      interpreter: bash
      workdir: /app/frontend
      timeout: 600
```

### 6. Node.js Script

```yaml
steps:
  - name: Run Node.js script
    uses: exec/script@v1
    with:
      script_content: |
        const fs = require('fs');
        console.log('Node.js version:', process.version);
        console.log('Platform:', process.platform);
      interpreter: node
```

### 7. Ruby Script

```yaml
steps:
  - name: Run Ruby script
    uses: exec/script@v1
    with:
      script_content: |
        #!/usr/bin/env ruby
        puts "Ruby version: #{RUBY_VERSION}"
        puts "Hello from Ruby"
      interpreter: ruby
```

## Error Handling

### Exit Code ≠ 0

Scripts that exit with non-zero status code will fail the workflow step:

```yaml
steps:
  - name: Risky operation
    uses: exec/script@v1
    with:
      script_content: |
        #!/bin/bash
        if [ ! -f /tmp/required.txt ]; then
          echo "Error: Required file missing" >&2
          exit 1
        fi
      interpreter: bash
    continue-on-error: true  # Continue workflow even if script fails
```

### Timeout Handling

Scripts that exceed the timeout will be automatically terminated:

```yaml
steps:
  - name: Long-running task
    uses: exec/script@v1
    with:
      script_content: |
        #!/bin/bash
        # This will timeout after 60 seconds
        sleep 100
      interpreter: bash
      timeout: 60
    continue-on-error: true
```

### Interpreter Not Found

If the specified interpreter doesn't exist on the system, the node will return an error:

```
Error: interpreter 'python3' not found on system
```

**Solution**: Install the required interpreter on the agent server.

### Mutual Exclusivity Error

You cannot specify both `script_path` and `script_content`:

```yaml
# ❌ INVALID - will fail
with:
  script_path: /tmp/script.sh
  script_content: "echo hello"  # Error: mutually exclusive

# ✅ VALID - choose one
with:
  script_content: "echo hello"
```

## Security Considerations

### Temporary File Management

For inline scripts (`script_content`):
- Temporary files are created with unique names
- Files are automatically cleaned up after execution
- Files are set to executable permissions (0755)
- Cleanup occurs even on timeout or error

### Environment Variable Sanitization

Environment variable values are sanitized to remove dangerous characters:
- Newlines (`\n`, `\r`)
- Null bytes (`\x00`)

### Script Arguments

Script arguments are passed directly without shell escaping. The interpreter handles argument parsing:
- **Bash**: Arguments available as `$1`, `$2`, etc.
- **Python**: Arguments in `sys.argv[1:]`
- **Node.js**: Arguments in `process.argv.slice(2)`

## Common Issues

### Q: Script works locally but fails in Waterflow

**A:** Check:
1. Interpreter availability: `which bash`, `which python3`
2. File permissions for `script_path`
3. Absolute paths instead of relative paths
4. Required environment variables are set

### Q: How to pass complex data to scripts?

**A:** Use environment variables or write data to temp files:

```yaml
with:
  script_content: |
    #!/usr/bin/env python3
    import json
    import os
    
    # Read JSON from environment
    data = json.loads(os.getenv('INPUT_DATA'))
    print(f"Received: {data}")
  interpreter: python3
  env:
    INPUT_DATA: '{"key": "value", "count": 42}'
```

### Q: Can I use shebang lines?

**A:** Yes, but they are ignored. The `interpreter` parameter determines execution:

```yaml
with:
  script_content: |
    #!/usr/bin/env python3  # Ignored
    print("Hello")
  interpreter: bash  # This is what actually runs (will fail!)
```

Always ensure `interpreter` matches your script language.

### Q: How to capture only stdout without stderr?

**A:** Both are captured separately in outputs:

```yaml
steps:
  - name: Capture outputs
    uses: exec/script@v1
    with:
      script_content: |
        echo "This goes to stdout"
        echo "This goes to stderr" >&2
      interpreter: bash
    # Access with: result.outputs.stdout and result.outputs.stderr
```

## Performance Notes

- **Temporary file overhead**: ~1-5ms per inline script execution
- **Memory usage**: Proportional to script output size
- **Concurrent execution**: Fully supported, temp files use unique names

## Building and Testing

```bash
# Build plugin
make build

# Run tests
make test

# Run integration tests
make integration-test

# Clean build artifacts
make clean

# Install to /opt/waterflow/plugins/
make install
```

## See Also

- [Shell Node (exec/shell)](../shell/README.md) - For simple single-line commands
- [Waterflow Node Development Guide](../../../docs/guides/node-development.md)
- [Epic 3: Core Node Plugin Library](../../../docs/epics.md#epic-3)
