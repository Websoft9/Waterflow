# Waterflow CLI

Waterflow CLI 是 Waterflow 工作流编排引擎的命令行工具,提供工作流管理、验证、提交和监控功能。

## 安装

### 从源码构建

```bash
# 克隆仓库
git clone https://github.com/Websoft9/waterflow.git
cd waterflow

# 构建 CLI
make cli

# 安装到系统
sudo make install-cli
```

## 快速开始

```bash
# 显示帮助
waterflow --help

# 显示版本
waterflow --version
waterflow version

# 后续 Story 将添加:
# - validate: 验证工作流 YAML (Story 5.2)
# - submit: 提交工作流 (Story 5.3)
# - status: 查询工作流状态 (Story 5.4)
# - logs: 获取工作流日志 (Story 5.5)
# - node: 管理节点 (Story 5.6)
```

## 全局参数

```bash
--server string         Waterflow server URL (env: WATERFLOW_SERVER)
--api-key string        API key for authentication (env: WATERFLOW_API_KEY)
--debug                 Enable debug logging
--config string         Config file path (default: ~/.waterflow/config.yaml)
--output-format string  Output format: text, json, or yaml (default: text)
```

## 配置文件

CLI 支持通过配置文件持久化设置,默认路径为 `~/.waterflow/config.yaml`。

配置文件示例:

```yaml
# ~/.waterflow/config.yaml

# Server 地址 (必需)
server: http://localhost:8088

# API 认证密钥 (可选)
api_key: my-secret-key

# 调试模式 (可选,默认 false)
debug: false

# 请求超时时间(秒) (可选,默认 30)
timeout: 60

# 输出格式 (可选,默认 text)
# 可选值: text, json, yaml
output_format: text
```

### 配置优先级

配置项的优先级顺序 (从高到低):

1. 命令行参数 (最高优先级)
2. 环境变量
3. 配置文件
4. 默认值 (最低优先级)

示例:

```bash
# 1. 命令行参数 (优先级最高)
waterflow submit --server http://localhost:8080 workflow.yaml

# 2. 环境变量
export WATERFLOW_SERVER=http://localhost:8080
waterflow submit workflow.yaml

# 3. 配置文件
cat > ~/.waterflow/config.yaml <<EOF
server: http://localhost:8080
EOF
waterflow submit workflow.yaml

# 4. 默认值 (http://localhost:8088)
waterflow submit workflow.yaml
```

## Commands

### validate

Validate workflow YAML syntax and structure.

**Usage:**
```bash
waterflow validate [flags] <workflow-file>...
```

**Flags:**
- `--remote` - Validate against server (checks node availability)
- `--verbose, -v` - Show detailed validation information
- `--format <format>` - Output format (text, json, yaml)
- `--recursive, -r` - Recursively validate YAML files in directory

**Examples:**

```bash
# Local validation (default)
waterflow validate workflow.yaml

# Validate multiple files
waterflow validate workflow1.yaml workflow2.yaml

# Server validation (checks node availability)
waterflow validate --remote workflow.yaml

# Recursive directory validation
waterflow validate --recursive ./workflows/

# Verbose output with detailed information
waterflow validate --verbose workflow.yaml

# JSON output for automation
waterflow validate --format json workflow.yaml | jq '.valid'

# YAML output
waterflow validate --format yaml workflow.yaml
```

**Exit Codes:**
- `0` - All workflows valid
- `1` - Validation failed
- `2` - Usage error

**Validation Modes:**

1. **Local Validation (default)**: Fast offline validation using DSL parser
   - YAML syntax validation
   - Schema structure validation  
   - Basic semantic validation
   - No network connection required

2. **Server Validation (--remote)**: Complete validation with node availability check
   - All local validations
   - Node plugin availability verification
   - Server-side semantic validation
   - Requires running Waterflow server

### submit

Submit a workflow to the Waterflow server for execution.

**Usage:**
```bash
waterflow submit [flags] <workflow-file>
```

**Flags:**
- `--wait, -w` - Wait for workflow completion
- `--follow, -f` - Follow logs in real-time (implies --wait)
- `--validate` - Validate workflow before submitting
- `--quiet, -q` - Only output workflow ID (for scripts)
- `--format <format>` - Output format (text, json)
- `--var <key=value>` - Override variables (can be used multiple times)

**Examples:**

```bash
# Quick submit (returns workflow ID)
waterflow submit workflow.yaml

# Submit with variable overrides
waterflow submit deployment.yaml \
  --var env=production \
  --var region=us-east-1

# Wait for completion
waterflow submit --wait workflow.yaml

# Follow logs in real-time
waterflow submit --follow workflow.yaml

# Validate before submitting
waterflow submit --validate workflow.yaml

# JSON output for automation
waterflow submit --format json workflow.yaml | jq '.id'

# Quiet mode for scripts
WORKFLOW_ID=$(waterflow submit --quiet workflow.yaml)
echo "Submitted: $WORKFLOW_ID"
waterflow status $WORKFLOW_ID
```

**Exit Codes:**
- `0` - Workflow submitted successfully (or completed successfully with --wait)
- `1` - Submission failed or workflow failed (with --wait)
- `2` - Usage error (invalid flags or arguments)

**Variable Type Inference:**

The `--var` flag intelligently parses values to appropriate types:
- Numbers: `--var timeout=300` → integer
- Floats: `--var ratio=0.75` → float
- Booleans: `--var debug=true` → boolean
- JSON: `--var config='{"key":"value"}'` → object/array
- Strings: `--var name=production` → string (default)

To force string type for numeric values, use JSON string syntax:
```bash
# Force "2023" as string (not integer)
waterflow submit --var version='"2023"' workflow.yaml
```

**Error Scenarios:**

```bash
# File not found
$ waterflow submit missing.yaml
Error: File not found
  Path: missing.yaml

Suggestion: Check file path or use 'waterflow validate --help'
Exit code: 2

# Server connection failed
$ waterflow submit workflow.yaml
Error: Failed to connect to server
  URL: http://localhost:8088
  Cause: connection refused

Suggestion:
  1. Check if Waterflow server is running
  2. Verify server URL with --server flag
  3. Check network connectivity
Exit code: 1

# Validation failed (server-side)
$ waterflow submit invalid.yaml
Error: Workflow validation failed
  Status: 422 Unprocessable Entity

Validation errors:
  1. Line 10: jobs.build.steps is required
     Suggestion: Add at least one step to the job

Suggestion: Run 'waterflow validate invalid.yaml' locally
Exit code: 1
```

### status

Query the status of a workflow execution.

**Usage:**
```bash
waterflow status [flags] <workflow-id>
```

**Flags:**
- `--watch, -w` - Watch status updates continuously
- `--interval <duration>` - Refresh interval for --watch (default: 2s)
- `--compact` - Compact mode (hide steps)
- `--format <format>` - Output format (text, json, yaml)
- `--quiet, -q` - Only output status value (for scripts)
- `--no-color` - Disable colored output

**Examples:**

```bash
# Get current status
waterflow status 550e8400-e29b-41d4-a716-446655440000

# Watch status updates
waterflow status --watch 550e8400-e29b-41d4-a716-446655440000

# Custom refresh interval (5 seconds)
waterflow status --watch --interval 5s <workflow-id>

# Compact mode (hide steps)
waterflow status --compact <workflow-id>

# JSON output for automation
waterflow status --format json <workflow-id> | jq '.status'

# YAML output
waterflow status --format yaml <workflow-id>

# Quiet mode for scripts
STATUS=$(waterflow status --quiet <workflow-id>)
if [ "$STATUS" = "completed" ]; then
  echo "Workflow succeeded"
fi

# Disable colors (for CI/CD)
waterflow status --no-color <workflow-id>
NO_COLOR=1 waterflow status <workflow-id>
```

**Output Example (Text):**

```
Workflow: Deploy Application
ID:       550e8400-e29b-41d4-a716-446655440000
Run ID:   temporal-run-id-123

Status:      running
Created:     2026-01-05T10:30:45Z
Started:     2026-01-05T10:30:46Z
Duration:    2m 15s

Jobs:
  ✓ build          completed  (2m 10s)
    ✓ checkout     completed
    ✓ compile      completed
    ✓ test         completed
  
  → deploy         running    (1m 10s)
    ✓ prepare      completed
    → push-image   running
    ○ restart      pending
  
  ○ verify         pending
    ○ health-check pending

Legend: ✓ completed  → running  ✗ failed  ○ pending
```

**Exit Codes:**
- `0` - Status query successful
- `1` - Query failed or workflow not found
- `2` - Usage error (invalid flags or arguments)

### logs

View execution logs from a workflow.

**Usage:**
```bash
waterflow logs [flags] <workflow-id>
```

**Flags:**
- `-f, --follow` - Follow logs in real-time (like `tail -f`)
- `--tail <n>` - Number of lines to show from the end (1-1000, -1 for all, default: 100)
- `--level <level>` - Filter by log level (info, warn, error, debug)
- `--job <name>` - Filter by job name
- `--step <name>` - Filter by step name
- `--format <format>` - Output format (text, json, compact, default: text)
- `--no-color` - Disable colored output
- `--no-timestamps` - Hide timestamps
- `--poll-interval <duration>` - Polling interval for `--follow` (default: 2s)

**Examples:**

```bash
# View recent logs (last 100 lines)
waterflow logs <workflow-id>

# Follow logs in real-time
waterflow logs --follow <workflow-id>

# Show only errors
waterflow logs --level error <workflow-id>

# Filter by job
waterflow logs --job deploy <workflow-id>

# Filter by step
waterflow logs --step checkout <workflow-id>

# Combine filters
waterflow logs --job deploy --level error <workflow-id>

# Show last 50 lines
waterflow logs --tail 50 <workflow-id>

# Show all logs
waterflow logs --tail -1 <workflow-id>

# JSON output for automation
waterflow logs --format json <workflow-id> | jq '.[] | select(.level=="error")'

# Compact format (no timestamps, no log level)
waterflow logs --format compact <workflow-id>

# No timestamps, no colors (for CI/CD)
waterflow logs --no-timestamps --no-color <workflow-id>

# Real-time monitoring with custom interval
waterflow logs --follow --poll-interval 5s <workflow-id>
```

**Output Example (Text):**

```
[2026-01-05 10:30:45] INFO  Workflow started
[2026-01-05 10:30:46] INFO  [build] Job started
[2026-01-05 10:30:47] INFO  [build.checkout] Checking out code from main branch
[2026-01-05 10:30:50] INFO  [build.checkout] Checkout completed
[2026-01-05 10:30:51] INFO  [build.compile] Compiling application
[2026-01-05 10:31:15] INFO  [build.compile] Compilation successful
[2026-01-05 10:31:16] INFO  [build.test] Running test suite
[2026-01-05 10:31:45] INFO  [build.test] All tests passed (127/127)
[2026-01-05 10:31:46] INFO  [build] Job completed
[2026-01-05 10:31:47] INFO  [deploy] Job started
[2026-01-05 10:31:48] INFO  [deploy.prepare] Preparing deployment environment
[2026-01-05 10:32:00] INFO  [deploy.push-image] Pushing Docker image
[2026-01-05 10:33:15] ERROR [deploy.push-image] Failed to authenticate Error: invalid credentials
```

**Output Example (JSON):**

```json
{"timestamp":"2026-01-05T10:30:45Z","level":"info","message":"Workflow started"}
{"timestamp":"2026-01-05T10:30:46Z","level":"info","job":"build","message":"Job started"}
{"timestamp":"2026-01-05T10:30:47Z","level":"info","job":"build","step":"checkout","message":"Checking out code from main branch"}
```

**Output Example (Compact):**

```
Workflow started
[build] Job started
[build.checkout] Checking out code from main branch
[build.checkout] Checkout completed
```

**Log Filtering:**

You can combine multiple filters to narrow down logs:

```bash
# Errors in deploy job
waterflow logs --job deploy --level error <workflow-id>

# Warnings and errors in checkout step
waterflow logs --step checkout --level warn <workflow-id>
waterflow logs --step checkout --level error <workflow-id>

# All logs from specific job
waterflow logs --job build <workflow-id>
```

**Follow Mode:**

When using `--follow`, the command will continuously poll for new logs and display them as they arrive. The workflow will automatically exit when the workflow completes.

```bash
# Follow logs with 2-second polling (default)
waterflow logs --follow <workflow-id>

# Follow logs with 5-second polling (reduce server load)
waterflow logs --follow --poll-interval 5s <workflow-id>

# Press Ctrl+C to stop following
```

**Exit Codes:**
- `0` - Logs retrieved successfully (or workflow completed successfully in follow mode)
- `1` - Query failed, workflow not found, or workflow failed (in follow mode)
- `2` - Usage error (invalid flags or arguments)

**Notes:**
- Logs are extracted from Temporal workflow event history
- Maximum tail limit is 1000 lines (API constraint)
- In `--follow` mode, the CLI polls the server at regular intervals (default: 2s)
- Log timestamps are in RFC3339 format (YYYY-MM-DDTHH:MM:SSZ)
- Color codes are automatically disabled when output is not a terminal

## 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `WATERFLOW_SERVER` | Server 地址 | `http://localhost:8088` |
| `WATERFLOW_API_KEY` | API 认证密钥 | (空) |

### node

List and inspect available workflow nodes.

**Usage:**
```bash
waterflow node list [flags] [node-name]
```

**Flags:**
- `--category <category>` - Filter by category (exec, flow, http, file, docker)
- `--search <keyword>` - Search nodes by name
- `--format <format>` - Output format (text, json, yaml, simple, default: text)
- `--no-group` - Disable category grouping

**Examples:**

```bash
# List all nodes
waterflow node list

# Show node details
waterflow node list exec/shell

# Filter by category
waterflow node list --category exec

# Search nodes
waterflow node list --search docker

# JSON output
waterflow node list --format json

# Simple list (names only)
waterflow node list --format simple
```

**Output Example:**

```
Available Nodes (7):

Execution:
  exec/shell@v1           Execute shell commands
  exec/script@v1          Run script files (bash, python, node)

Flow Control:
  flow/sleep@v1           Delay execution for specified duration

HTTP:
  http/request@v1         Make HTTP requests (GET, POST, PUT, DELETE)

File Transfer:
  file/transfer@v1        Transfer files via SCP/SFTP

Docker:
  docker/exec@v1          Execute Docker commands
  docker/compose@v1       Manage Docker Compose services

Use 'waterflow node list <name>' to see details for a specific node.
```

## 环境变量

## 开发

### 运行测试

```bash
# 单元测试
make test-cli

# 集成测试
./cmd/waterflow-cli/integration_test.sh
```

### 构建

```bash
# 开发构建
make cli

# 生产构建 (包含版本信息)
make cli VERSION=v1.0.0
```

## 错误处理

CLI 提供清晰的错误信息和建议,帮助快速定位和解决问题。

退出码:

- `0` - 成功
- `1` - 一般错误 (配置错误、网络错误等)
- `2` - 使用错误 (命令错误、参数错误)

调试模式:

```bash
# 启用详细日志
waterflow --debug submit workflow.yaml
```

## 后续功能

本 Story (5.1) 实现了 CLI 基础框架。后续 Story 将添加:

- **Story 5.2** - `validate` 命令 (本地 YAML 验证)
- **Story 5.3** - `submit` 命令 (提交工作流)
- **Story 5.4** - `status` 命令 (查询工作流状态)
- **Story 5.5** - `logs` 命令 (获取工作流日志)
- **Story 5.6** - `node` 命令 (节点管理)
- **Story 5.7** - Go SDK Client (替换当前临时 HTTP 客户端)

## 参考文档

- [Waterflow 架构文档](../../docs/architecture.md)
- [PRD - Epic 5](../../docs/prd.md#epic-5-客户端工具和-sdk)
- [Story 5.1 完整说明](../../docs/sprint-artifacts/5-1-cli-framework.md)
