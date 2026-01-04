# Story 5.4: CLI status 命令

Status: ready-for-dev

## Story

As a **工作流用户**,  
I want **查询工作流状态**,  
So that **了解执行进度**。

## Context

这是 Epic 5 (客户端工具和 SDK) 的第四个 Story,在 submit 命令(5.3)之后,实现**状态查询命令**。该命令让用户能够通过命令行快速查看工作流的执行状态和进度,是工作流监控的核心功能。

**前置依赖:**
- ✅ Story 5.1 - CLI 基础框架 (HTTP 客户端、输出格式化)
- ✅ Story 5.3 - submit 命令 (已实现 GetWorkflowStatus 客户端方法)
- ✅ Story 1.9 - REST API 服务 (`GET /v1/workflows/{id}` 端点)
- ✅ Story 1.8 - Temporal SDK 集成 (状态查询)

**Epic 背景:**  
Epic 5 专注于**客户端工具和 SDK**。status 命令是工作流生命周期监控的核心,支持多种模式:
1. **一次性查询** - 查询当前状态并退出 (默认)
2. **持续监控** - `--watch` 参数实时刷新状态
3. **详细信息** - 显示 Jobs/Steps 执行进度
4. **多种输出格式** - text/json/yaml 格式支持

本 Story 实现状态查询功能,为用户提供工作流执行的实时可见性,配合 submit 和 logs 命令形成完整的工作流管理工具链。

**业务价值:**
- 🎯 **执行可见性** - 实时了解工作流执行状态
- 🎯 **进度跟踪** - 查看当前执行的 Job/Step
- 🎯 **快速诊断** - 快速判断工作流是否成功/失败
- 🎯 **自动化集成** - 支持脚本轮询状态

**技术定位:**
- **不是** 日志查看工具 (logs 命令负责)
- **不是** 实时跟踪工具 (submit --follow 负责)
- **是** 状态查询和监控的入口
- **是** Server REST API 的便捷封装

**复用现有组件:**
- Story 5.1: HTTP 客户端、配置管理、输出格式化
- Story 5.3: GetWorkflowStatus() 方法 (已在 submit --wait 中实现)
- Story 1.9: `GET /v1/workflows/{id}` API

**与其他命令的关系:**
- `submit` → 提交工作流,返回 ID
- `status` → 查询工作流状态和进度
- `logs` → 查看工作流执行日志

## Acceptance Criteria

### AC1: 基础状态查询

**Given** 存在有效的工作流 ID  
**When** 执行 `waterflow status <workflow-id>`  
**Then** 显示工作流基本状态信息  
**And** 显示工作流名称和 ID  
**And** 显示当前状态 (running/completed/failed/cancelled)  
**And** 显示结论 (success/failure/cancelled/timeout)  
**And** 显示时间信息 (创建/开始/完成时间,持续时间)  
**And** 返回退出码 0

**状态查询成功示例:**
```bash
$ waterflow status 550e8400-e29b-41d4-a716-446655440000
Workflow: Deploy Application
ID:       550e8400-e29b-41d4-a716-446655440000
Run ID:   temporal-run-id-123

Status:      running
Conclusion:  -
Created:     2026-01-04T10:30:45Z
Started:     2026-01-04T10:30:46Z
Duration:    2m 15s

$ echo $?
0
```

**已完成工作流示例:**
```bash
$ waterflow status abc-456
Workflow: Deploy Application
ID:       abc-456
Run ID:   temporal-run-id-456

Status:      completed
Conclusion:  success
Created:     2026-01-04T10:00:00Z
Started:     2026-01-04T10:00:01Z
Completed:   2026-01-04T10:05:30Z
Duration:    5m 29s

$ echo $?
0
```

**失败工作流示例:**
```bash
$ waterflow status def-789
Workflow: Deploy Application
ID:       def-789
Run ID:   temporal-run-id-789

Status:      failed
Conclusion:  failure
Created:     2026-01-04T09:00:00Z
Started:     2026-01-04T09:00:01Z
Completed:   2026-01-04T09:02:15Z
Duration:    2m 14s

View logs:   waterflow logs def-789

$ echo $?
0
```

### AC2: 显示执行进度

**Given** 工作流正在执行  
**When** 查询状态  
**Then** 显示当前执行的 Jobs 和 Steps  
**And** 显示每个 Job 的状态  
**And** 显示当前执行的 Step  
**And** 使用缩进表示层级关系

**进度显示示例:**
```bash
$ waterflow status 550e8400-e29b-41d4-a716-446655440000
Workflow: Multi-Step Deployment
ID:       550e8400-e29b-41d4-a716-446655440000

Status:      running
Duration:    3m 20s

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

**紧凑模式 (无 Steps):**
```bash
$ waterflow status --compact 550e8400-e29b-41d4-a716-446655440000
Workflow: Multi-Step Deployment
ID:       550e8400-e29b-41d4-a716-446655440000

Status:      running
Duration:    3m 20s

Jobs:
  ✓ build    completed  (2m 10s)
  → deploy   running    (1m 10s)
  ○ verify   pending
```

### AC3: 持续监控模式 (--watch)

**Given** 工作流正在执行  
**When** 使用 `--watch` 参数  
**Then** 每隔指定时间刷新状态  
**And** 清屏后重新显示完整状态  
**And** 工作流完成后自动退出  
**And** 支持 Ctrl+C 中断

**持续监控示例:**
```bash
$ waterflow status --watch 550e8400-e29b-41d4-a716-446655440000
# 每 2 秒刷新一次 (可通过 --interval 配置)

Workflow: Deploy Application
ID:       550e8400-e29b-41d4-a716-446655440000

Status:      running
Duration:    45s

Jobs:
  → build    running    (45s)
    ✓ checkout  completed
    → compile   running

Refreshing every 2s... (Press Ctrl+C to stop)

# 2 秒后刷新显示

Workflow: Deploy Application
ID:       550e8400-e29b-41d4-a716-446655440000

Status:      running
Duration:    47s

Jobs:
  → build    running    (47s)
    ✓ checkout  completed
    ✓ compile   completed
    → test      running

Refreshing every 2s... (Press Ctrl+C to stop)

# 工作流完成后退出

Workflow: Deploy Application
ID:       550e8400-e29b-41d4-a716-446655440000

Status:      completed
Conclusion:  success
Duration:    5m 30s

Jobs:
  ✓ build    completed  (2m 10s)
  ✓ deploy   completed  (3m 20s)

✓ Workflow completed successfully

$ echo $?
0
```

**自定义刷新间隔:**
```bash
$ waterflow status --watch --interval 5s 550e8400-e29b-41d4-a716-446655440000
# 每 5 秒刷新一次
```

### AC4: 格式化输出选项

**Given** 需要集成到自动化流程  
**When** 使用输出格式参数  
**Then** 支持 `--format text` (默认,人类可读)  
**And** 支持 `--format json` (JSON 格式,机器可读)  
**And** 支持 `--format yaml` (YAML 格式)  
**And** 支持 `--quiet` 参数仅输出状态值

**JSON 输出示例:**
```bash
$ waterflow status --format json 550e8400-e29b-41d4-a716-446655440000
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "run_id": "temporal-run-id-123",
  "name": "Deploy Application",
  "status": "running",
  "conclusion": "",
  "created_at": "2026-01-04T10:30:45Z",
  "started_at": "2026-01-04T10:30:46Z",
  "completed_at": "",
  "duration_seconds": 135,
  "jobs": [
    {
      "id": "build",
      "name": "build",
      "status": "completed",
      "started_at": "2026-01-04T10:30:46Z",
      "completed_at": "2026-01-04T10:32:56Z",
      "steps": [
        {
          "name": "checkout",
          "status": "completed",
          "conclusion": "success",
          "started_at": "2026-01-04T10:30:46Z",
          "completed_at": "2026-01-04T10:31:00Z"
        }
      ]
    }
  ]
}

$ echo $?
0
```

**YAML 输出示例:**
```bash
$ waterflow status --format yaml 550e8400-e29b-41d4-a716-446655440000
id: 550e8400-e29b-41d4-a716-446655440000
run_id: temporal-run-id-123
name: Deploy Application
status: running
created_at: "2026-01-04T10:30:45Z"
started_at: "2026-01-04T10:30:46Z"
duration_seconds: 135
jobs:
  - id: build
    name: build
    status: completed
    started_at: "2026-01-04T10:30:46Z"
    completed_at: "2026-01-04T10:32:56Z"
```

**Quiet 模式 (脚本友好):**
```bash
$ waterflow status --quiet 550e8400-e29b-41d4-a716-446655440000
running

# 仅输出状态,便于脚本使用
STATUS=$(waterflow status --quiet $WORKFLOW_ID)
if [ "$STATUS" = "completed" ]; then
  echo "Workflow succeeded"
fi
```

### AC5: 友好的错误处理

**Given** 查询状态时遇到错误  
**When** 发生各种错误情况  
**Then** 显示清晰的错误信息和建议

**错误场景覆盖:**

**1. 工作流不存在:**
```bash
$ waterflow status nonexistent-id
Error: Workflow not found
  Workflow ID: nonexistent-id

Suggestion: Check workflow ID or use 'waterflow list' to see all workflows
Exit code: 1
```

**2. Server 连接失败:**
```bash
$ waterflow status 550e8400-e29b-41d4-a716-446655440000
Error: Failed to connect to server
  URL: http://localhost:8088
  Cause: dial tcp 127.0.0.1:8088: connect: connection refused

Suggestion:
  1. Check if Waterflow server is running (docker-compose up)
  2. Verify server URL with --server flag or config file
  3. Check network connectivity

Exit code: 1
```

**3. API 认证失败:**
```bash
$ waterflow status 550e8400-e29b-41d4-a716-446655440000
Error: API authentication failed
  Status: 401 Unauthorized
  Message: Invalid or missing API key

Suggestion: Set API key using --api-key flag or WATERFLOW_API_KEY env variable

Exit code: 1
```

**4. Server 内部错误:**
```bash
$ waterflow status 550e8400-e29b-41d4-a716-446655440000
Error: Server internal error
  Status: 500 Internal Server Error
  Message: Failed to query workflow status

Suggestion: Check server logs for details or contact administrator

Exit code: 1
```

**5. 无效的工作流 ID 格式:**
```bash
$ waterflow status invalid
Error: Invalid workflow ID format
  Value: 'invalid'
  Expected: UUID v4 format (e.g., 550e8400-e29b-41d4-a716-446655440000)

Exit code: 2
```

### AC6: 状态符号和颜色

**Given** 在终端显示状态  
**When** 输出到 TTY  
**Then** 使用符号表示状态  
**And** 使用颜色高亮 (可通过 --no-color 禁用)  
**And** 状态符号统一

**状态符号定义:**
- `✓` - 成功/已完成 (绿色)
- `→` - 正在执行 (蓝色)
- `✗` - 失败 (红色)
- `○` - 待执行 (灰色)
- `⊗` - 已取消 (黄色)

**颜色示例:**
```bash
$ waterflow status 550e8400-e29b-41d4-a716-446655440000
Workflow: Deploy Application
Status:      running  (蓝色)

Jobs:
  ✓ build    completed  (绿色)
  → deploy   running    (蓝色)
  ○ verify   pending    (灰色)
```

**失败状态颜色:**
```bash
$ waterflow status abc-456
Workflow: Deploy Application
Status:      failed  (红色)
Conclusion:  failure (红色)

Jobs:
  ✓ build    completed  (绿色)
  ✗ deploy   failed     (红色)
```

**禁用颜色:**
```bash
$ waterflow status --no-color 550e8400-e29b-41d4-a716-446655440000
# 纯文本输出,无 ANSI 颜色码

# 或通过环境变量
$ NO_COLOR=1 waterflow status 550e8400-e29b-41d4-a716-446655440000
```

## Tasks / Subtasks

### Task 1: status 子命令框架 (AC1)
- [ ] 创建 `cmd/waterflow-cli/cmd/status.go`
  ```go
  package cmd
  
  import (
      "fmt"
      "os"
      "time"
      
      "github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/client"
      "github.com/spf13/cobra"
      "go.uber.org/zap"
  )
  
  var (
      statusWatch    bool
      statusInterval string
      statusCompact  bool
      statusFormat   string
      statusQuiet    bool
      statusNoColor  bool
  )
  
  func newStatusCmd() *cobra.Command {
      cmd := &cobra.Command{
          Use:   "status <workflow-id>",
          Short: "Get workflow execution status",
          Long: `Query the status of a workflow execution.
  
  Displays workflow status, execution progress, jobs, and steps.
  Use --watch to continuously monitor a running workflow.
  
  Examples:
    # Get current status
    waterflow status 550e8400-e29b-41d4-a716-446655440000
    
    # Watch status updates
    waterflow status --watch 550e8400-e29b-41d4-a716-446655440000
    
    # Custom refresh interval
    waterflow status --watch --interval 5s <workflow-id>
    
    # Compact mode (no steps)
    waterflow status --compact <workflow-id>
    
    # JSON output for automation
    waterflow status --format json <workflow-id>
    
    # Quiet mode (only status value)
    STATUS=$(waterflow status --quiet <workflow-id>)`,
          Args: cobra.ExactArgs(1),
          RunE: runStatus,
      }
      
      cmd.Flags().BoolVarP(&statusWatch, "watch", "w", false, "Watch status updates continuously")
      cmd.Flags().StringVar(&statusInterval, "interval", "2s", "Refresh interval for --watch (e.g., 2s, 5s)")
      cmd.Flags().BoolVar(&statusCompact, "compact", false, "Compact mode (hide steps)")
      cmd.Flags().StringVar(&statusFormat, "format", "text", "Output format (text, json, yaml)")
      cmd.Flags().BoolVarP(&statusQuiet, "quiet", "q", false, "Only output status value (for scripts)")
      cmd.Flags().BoolVar(&statusNoColor, "no-color", false, "Disable colored output")
      
      return cmd
  }
  
  func runStatus(cmd *cobra.Command, args []string) error {
      workflowID := args[0]
      
      // 验证工作流 ID 格式
      if err := validateWorkflowID(workflowID); err != nil {
          return err
      }
      
      // 从根命令获取配置
      cfg, err := loadConfig()
      if err != nil {
          return fmt.Errorf("failed to load config: %w", err)
      }
      
      // 创建 logger
      var logger *zap.Logger
      if cfg.Debug {
          logger, _ = zap.NewDevelopment()
      } else {
          logger = zap.NewNop()
      }
      defer logger.Sync()
      
      // 创建 HTTP 客户端
      httpClient := client.New(
          cfg.Server,
          cfg.APIKey,
          cfg.Timeout,
          cfg.Debug,
      )
      
      // 检查颜色支持
      noColor := statusNoColor || os.Getenv("NO_COLOR") != "" || !isTerminal()
      
      // 创建输出格式化器
      formatter := newOutputFormatter(statusFormat, noColor)
      
      // 监控模式
      if statusWatch {
          return watchStatus(httpClient, workflowID, statusInterval, formatter)
      }
      
      // 一次性查询
      status, err := httpClient.GetWorkflowStatus(workflowID)
      if err != nil {
          return formatStatusError(err, statusFormat)
      }
      
      // 输出结果
      if statusQuiet {
          fmt.Println(status.Status)
          return nil
      }
      
      if err := formatter.PrintWorkflowStatus(status, statusCompact); err != nil {
          return err
      }
      
      return nil
  }
  ```

- [ ] 定义命令参数
  - `--watch, -w` - 持续监控
  - `--interval` - 刷新间隔
  - `--compact` - 紧凑模式
  - `--format` - 输出格式
  - `--quiet, -q` - 仅输出状态值
  - `--no-color` - 禁用颜色

- [ ] 注册到根命令

### Task 2: 工作流 ID 验证 (AC5)
- [ ] 实现 ID 验证函数
  ```go
  // cmd/waterflow-cli/cmd/status.go
  
  import (
      "fmt"
      "regexp"
  )
  
  var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
  
  // validateWorkflowID 验证工作流 ID 格式
  func validateWorkflowID(id string) error {
      if id == "" {
          return fmt.Errorf("workflow ID is required")
      }
      
      // UUID v4 格式验证 (可选,Server 也会验证)
      // 这里宽松处理,允许任意字符串,Server 端会返回 404
      // 但可以提供友好提示
      if !uuidRegex.MatchString(id) {
          fmt.Fprintf(os.Stderr, "Warning: Workflow ID '%s' is not a valid UUID format\n", id)
          fmt.Fprintf(os.Stderr, "Expected format: 550e8400-e29b-41d4-a716-446655440000\n\n")
      }
      
      return nil
  }
  ```

- [ ] UUID 格式验证
- [ ] 友好提示信息
- [ ] 单元测试验证逻辑

### Task 3: HTTP 客户端 GetWorkflowStatus 方法 (AC1)

**Developer Context:**
本方法在 Story 5.4 中首次实现。Story 5.3 虽然提到此方法,但实际使用临时 HTTP 调用。本 Story 提供完整实现,供后续 Stories 复用。

- [ ] 扩展 `cmd/waterflow-cli/pkg/client/client.go`
  ```go
  package client
  
  import (
      "context"
      "encoding/json"
      "fmt"
      "net/http"
      "time"
  )
  
  // WorkflowStatus 工作流状态响应 (对应 Story 1.9 API schema)
  type WorkflowStatus struct {
      ID              string                 `json:"id"`
      RunID           string                 `json:"run_id"`
      Name            string                 `json:"name"`
      Status          string                 `json:"status"` // 值: pending, running, completed, failed, cancelled, timeout
      // NOTE: Story 1.9 API 没有 conclusion 字段,状态信息包含在 status 中
      CreatedAt       string                 `json:"created_at"`
      StartedAt       string                 `json:"started_at,omitempty"`
      CompletedAt     string                 `json:"completed_at,omitempty"`
      DurationSeconds *int                   `json:"duration_seconds,omitempty"`
      Vars            map[string]interface{} `json:"vars,omitempty"`
      Jobs            []JobStatus            `json:"jobs,omitempty"`
      Error           string                 `json:"error,omitempty"` // 失败时的错误信息
  }
  
  // JobStatus Job 执行状态
  type JobStatus struct {
      ID          string       `json:"id"`
      Name        string       `json:"name"`
      Status      string       `json:"status"`
      StartedAt   string       `json:"started_at,omitempty"`
      CompletedAt string       `json:"completed_at,omitempty"`
      RunsOn      string       `json:"runs_on,omitempty"`
      Steps       []StepStatus `json:"steps,omitempty"`
  }
  
  // StepStatus Step 执行状态
  type StepStatus struct {
      Name        string `json:"name"`
      Status      string `json:"status"`
      Conclusion  string `json:"conclusion,omitempty"`
      StartedAt   string `json:"started_at,omitempty"`
      CompletedAt string `json:"completed_at,omitempty"`
  }
  
  // GetWorkflowStatus 查询工作流状态
  func (c *Client) GetWorkflowStatus(workflowID string) (*WorkflowStatus, error) {
      httpReq, err := http.NewRequestWithContext(
          context.Background(),
          http.MethodGet,
          fmt.Sprintf("%s/v1/workflows/%s", c.baseURL, workflowID),
          nil,
      )
      if err != nil {
          return nil, fmt.Errorf("failed to create request: %w", err)
      }
      
      if c.apiKey != "" {
          httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
      }
      
      if c.debug {
          fmt.Printf("DEBUG: GET %s/v1/workflows/%s\n", c.baseURL, workflowID)
      }
      
      resp, err := c.httpClient.Do(httpReq)
      if err != nil {
          return nil, fmt.Errorf("request failed: %w", err)
      }
      defer resp.Body.Close()
      
      if resp.StatusCode != http.StatusOK {
          return nil, c.parseError(resp)
      }
      
      var status WorkflowStatus
      if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
          return nil, fmt.Errorf("failed to decode response: %w", err)
      }
      
      return &status, nil
  }
  ```

- [ ] 实现 GET /v1/workflows/{id} 调用
- [ ] 处理 Server 错误响应
- [ ] 单元测试 `pkg/client/status_test.go`

### Task 4: 持续监控逻辑 (AC3)
- [ ] 实现监控函数
  ```go
  // cmd/waterflow-cli/cmd/status.go
  
  import (
      "context"
      "fmt"
      "os"
      "os/signal"
      "syscall"
      "time"
  )
  
  // watchStatus 持续监控工作流状态
  func watchStatus(c *client.Client, workflowID string, intervalStr string, formatter *Formatter) error {
      // 解析刷新间隔
      interval, err := time.ParseDuration(intervalStr)
      if err != nil {
          return fmt.Errorf("invalid interval format: %s (expected: 2s, 5s, 1m)", intervalStr)
      }
      
      if interval < 1*time.Second {
          return fmt.Errorf("interval too short: %s (minimum: 1s)", intervalStr)
      }
      
      // 设置信号处理 (Ctrl+C 退出)
      ctx, cancel := context.WithCancel(context.Background())
      defer cancel()
      
      sigChan := make(chan os.Signal, 1)
      signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
      
      go func() {
          <-sigChan
          fmt.Println("\n\nInterrupted by user")
          cancel()
      }()
      
      ticker := time.NewTicker(interval)
      defer ticker.Stop()
      
      // 首次查询
      if err := displayStatus(c, workflowID, formatter, true); err != nil {
          return err
      }
      
      for {
          select {
          case <-ctx.Done():
              return nil
              
          case <-ticker.C:
              // 清屏 (仅在 TTY 时)
              if isTerminal() {
                  clearScreen()
              }
              
              // 查询并显示状态
              completed, err := displayStatus(c, workflowID, formatter, true)
              if err != nil {
                  return err
              }
              
              // 工作流完成后退出
              if completed {
                  return nil
              }
          }
      }
  }
  
  // displayStatus 查询并显示状态
  func displayStatus(c *client.Client, workflowID string, formatter *Formatter, showRefreshHint bool) (bool, error) {
      status, err := c.GetWorkflowStatus(workflowID)
      if err != nil {
          return false, formatStatusError(err, formatter.format)
      }
      
      if err := formatter.PrintWorkflowStatus(status, statusCompact); err != nil {
          return false, err
      }
      
      // 显示刷新提示
      if showRefreshHint && !isTerminalStatus(status.Status) {
          fmt.Printf("\nRefreshing every %s... (Press Ctrl+C to stop)\n", statusInterval)
      }
      
      // 判断是否完成
      return isTerminalStatus(status.Status), nil
  }
  
  // clearScreen 清屏 (ANSI 转义码)
  func clearScreen() {
      fmt.Print("\033[H\033[2J")
  }
  
  // isTerminal 判断是否为终端
  func isTerminal() bool {
      fileInfo, _ := os.Stdout.Stat()
      return (fileInfo.Mode() & os.ModeCharDevice) != 0
  }
  
  // isTerminalStatus 判断是否为终止状态
  func isTerminalStatus(status string) bool {
      return status == "completed" || status == "failed" || status == "cancelled"
  }
  ```

- [ ] 实现轮询逻辑
- [ ] 清屏和刷新显示
- [ ] 信号处理 (Ctrl+C)
- [ ] 工作流完成后退出

### Task 5: 输出格式化 - Text 格式 (AC2, AC6)
- [ ] 创建 `cmd/waterflow-cli/pkg/output/status.go`
  ```go
  package output
  
  import (
      "fmt"
      "strings"
      "time"
      
      "github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/client"
      "github.com/fatih/color"
  )
  
  // PrintWorkflowStatus 打印工作流状态
  func (f *Formatter) PrintWorkflowStatus(status *client.WorkflowStatus, compact bool) error {
      switch f.format {
      case FormatJSON:
          return f.printStatusJSON(status)
      case FormatYAML:
          return f.printStatusYAML(status)
      default:
          return f.printStatusText(status, compact)
      }
  }
  
  func (f *Formatter) printStatusText(status *client.WorkflowStatus, compact bool) error {
      // 标题
      fmt.Printf("Workflow: %s\n", status.Name)
      fmt.Printf("ID:       %s\n", status.ID)
      if status.RunID != "" {
          fmt.Printf("Run ID:   %s\n", status.RunID)
      }
      fmt.Println()
      
      // 状态 (根据 Story 1.9 API,只有 status 字段)
      statusColor := f.getStatusColor(status.Status)
      fmt.Printf("Status:      %s\n", statusColor.Sprint(status.Status))
      
      // 失败时显示错误信息
      if status.Error != "" {
          fmt.Printf("Error:       %s\n", color.RedString(status.Error))
      }
      
      // 时间信息
      if status.CreatedAt != "" {
          fmt.Printf("Created:     %s\n", status.CreatedAt)
      }
      if status.StartedAt != "" {
          fmt.Printf("Started:     %s\n", status.StartedAt)
      }
      if status.CompletedAt != "" {
          fmt.Printf("Completed:   %s\n", status.CompletedAt)
      }
      if status.DurationSeconds != nil {
          fmt.Printf("Duration:    %s\n", formatDuration(*status.DurationSeconds))
      }
      
      // Jobs
      if len(status.Jobs) > 0 {
          fmt.Println()
          fmt.Println("Jobs:")
          for _, job := range status.Jobs {
              f.printJob(job, compact)
          }
          
          if !compact {
              fmt.Println()
              f.printLegend()
          }
      }
      
      // 失败提示
      if status.Status == "failed" {
          fmt.Printf("\nView logs:   waterflow logs %s\n", status.ID)
      }
      
      return nil
  }
  
  func (f *Formatter) printJob(job client.JobStatus, compact bool) {
      symbol := f.getStatusSymbol(job.Status)
      statusColor := f.getStatusColor(job.Status)
      
      duration := ""
      if job.StartedAt != "" && job.CompletedAt != "" {
          d := calculateDuration(job.StartedAt, job.CompletedAt)
          duration = fmt.Sprintf("  (%s)", formatDuration(d))
      } else if job.StartedAt != "" && job.Status == "running" {
          d := calculateDurationSince(job.StartedAt)
          duration = fmt.Sprintf("  (%s)", formatDuration(d))
      }
      
      fmt.Printf("  %s %s", symbol, job.Name)
      fmt.Printf("  %s", statusColor.Sprintf("%-10s", job.Status))
      fmt.Printf("%s\n", duration)
      
      // Steps (仅在非紧凑模式下)
      if !compact && len(job.Steps) > 0 {
          for _, step := range job.Steps {
              f.printStep(step)
          }
          fmt.Println()
      }
  }
  
  func (f *Formatter) printStep(step client.StepStatus) {
      symbol := f.getStatusSymbol(step.Status)
      statusColor := f.getStatusColor(step.Status)
      
      fmt.Printf("    %s %s", symbol, step.Name)
      fmt.Printf("  %s\n", statusColor.Sprint(step.Status))
  }
  
  func (f *Formatter) getStatusSymbol(status string) string {
      if f.noColor {
          // 无颜色模式使用简单字符
          switch status {
          case "completed":
              return "[✓]"
          case "running":
              return "[→]"
          case "failed":
              return "[✗]"
          case "cancelled":
              return "[⊗]"
          case "timeout":
              return "[⊗]" // timeout 使用相同符号
          default:
              return "[○]"
          }
      }
      
      // 带颜色的符号
      switch status {
      case "completed":
          return color.GreenString("✓")
      case "running":
          return color.BlueString("→")
      case "failed":
          return color.RedString("✗")
      case "cancelled":
          return color.YellowString("⊗")
      case "timeout":
          return color.YellowString("⊗") // timeout 使用黄色
      default:
          return color.New(color.FgHiBlack).Sprint("○")
      }
  }
  
  func (f *Formatter) getStatusColor(status string) *color.Color {
      if f.noColor {
          return color.New()
      }
      
      switch status {
      case "completed":
          return color.New(color.FgGreen)
      case "running":
          return color.New(color.FgBlue)
      case "failed":
          return color.New(color.FgRed)
      case "cancelled":
          return color.New(color.FgYellow)
      case "timeout":
          return color.New(color.FgYellow)
      default:
          return color.New(color.FgHiBlack)
      }
  }
  
  func (f *Formatter) printLegend() {
      if f.noColor {
          fmt.Println("Legend: [✓] completed  [→] running  [✗] failed  [○] pending  [⊗] cancelled/timeout")
      } else {
          fmt.Printf("Legend: %s completed  %s running  %s failed  %s pending  %s cancelled/timeout\n",
              color.GreenString("✓"),
              color.BlueString("→"),
              color.RedString("✗"),
              color.New(color.FgHiBlack).Sprint("○"),
              color.YellowString("⊗"),
          )
      }
  }
  
  // formatDuration 格式化持续时间 (秒 → 人类可读)
  func formatDuration(seconds int) string {
      d := time.Duration(seconds) * time.Second
      
      if d < time.Minute {
          return fmt.Sprintf("%ds", seconds)
      }
      
      minutes := int(d.Minutes())
      remainingSeconds := seconds - (minutes * 60)
      
      if d < time.Hour {
          return fmt.Sprintf("%dm %ds", minutes, remainingSeconds)
      }
      
      hours := int(d.Hours())
      remainingMinutes := minutes - (hours * 60)
      return fmt.Sprintf("%dh %dm", hours, remainingMinutes)
  }
  
  // calculateDuration 计算两个时间戳之间的持续时间
  func calculateDuration(startStr, endStr string) int {
      start, _ := time.Parse(time.RFC3339, startStr)
      end, _ := time.Parse(time.RFC3339, endStr)
      return int(end.Sub(start).Seconds())
  }
  
  // calculateDurationSince 计算从开始到现在的持续时间
  func calculateDurationSince(startStr string) int {
      start, _ := time.Parse(time.RFC3339, startStr)
      return int(time.Since(start).Seconds())
  }
  
  // printJobsTable 使用 tablewriter 显示 Jobs (可选实现,增强可读性)
  func (f *Formatter) printJobsTable(jobs []client.JobStatus) {
      table := tablewriter.NewWriter(os.Stdout)
      table.SetHeader([]string{"Job", "Status", "Duration", "Queue"})
      table.SetBorder(false)
      table.SetColumnSeparator(" ")
      
      for _, job := range jobs {
          symbol := f.getStatusSymbol(job.Status)
          duration := "--"
          if job.StartedAt != "" && job.CompletedAt != "" {
              d := calculateDuration(job.StartedAt, job.CompletedAt)
              duration = formatDuration(d)
          }
          
          table.Append([]string{
              fmt.Sprintf("%s %s", symbol, job.Name),
              job.Status,
              duration,
              job.RunsOn,
          })
      }
      
      table.Render()
  }
  ```

- [ ] 实现 text 格式输出
- [ ] 实现状态符号和颜色
- [ ] 实现 Jobs/Steps 层级显示
- [ ] 实现时间格式化
- [ ] (可选) 使用 tablewriter 增强表格显示

### Task 6: 输出格式化 - JSON/YAML 格式 (AC4)
- [ ] 扩展 `cmd/waterflow-cli/pkg/output/status.go`
  ```go
  package output
  
  import (
      "encoding/json"
      "os"
      
      "github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/client"
      "gopkg.in/yaml.v3"
  )
  
  func (f *Formatter) printStatusJSON(status *client.WorkflowStatus) error {
      enc := json.NewEncoder(os.Stdout)
      enc.SetIndent("", "  ")
      return enc.Encode(status)
  }
  
  func (f *Formatter) printStatusYAML(status *client.WorkflowStatus) error {
      enc := yaml.NewEncoder(os.Stdout)
      enc.SetIndent(2)
      return enc.Encode(status)
  }
  ```

- [ ] 实现 JSON 格式输出
- [ ] 实现 YAML 格式输出
- [ ] 单元测试输出格式

### Task 7: 错误处理和建议 (AC5)
- [ ] 实现错误格式化函数
  ```go
  // cmd/waterflow-cli/cmd/status.go
  
  import (
      "github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/client"
  )
  
  // formatStatusError 格式化状态查询错误
  func formatStatusError(err error, format string) error {
      if format == "json" {
          return formatStatusErrorJSON(err)
      }
      
      if serverErr, ok := err.(*client.ServerError); ok {
          return formatServerError(serverErr)
      }
      
      // 其他错误
      return err
  }
  
  func formatServerError(err *client.ServerError) error {
      fmt.Println("Error: Workflow status query failed")
      fmt.Printf("  Status: %d %s\n\n", err.StatusCode, http.StatusText(err.StatusCode))
      
      switch err.StatusCode {
      case 404:
          fmt.Println("Workflow not found")
          if workflowID, ok := err.Details["workflow_id"].(string); ok {
              fmt.Printf("  Workflow ID: %s\n", workflowID)
          }
          fmt.Println("\nSuggestion: Check workflow ID or use 'waterflow list' to see all workflows")
          
      case 401:
          fmt.Printf("Message: %s\n", err.Message)
          fmt.Println("\nSuggestion: Set API key using --api-key flag or WATERFLOW_API_KEY env variable")
          
      default:
          fmt.Printf("Message: %s\n", err.Message)
          fmt.Println("\nSuggestion: Check server logs for details or contact administrator")
      }
      
      return &ExitError{Code: 1}
  }
  
  func formatStatusErrorJSON(err error) error {
      if serverErr, ok := err.(*client.ServerError); ok {
          enc := json.NewEncoder(os.Stdout)
          enc.SetIndent("", "  ")
          _ = enc.Encode(map[string]interface{}{
              "error": map[string]interface{}{
                  "code":    serverErr.Code,
                  "message": serverErr.Message,
                  "details": serverErr.Details,
              },
          })
      }
      return &ExitError{Code: 1}
  }
  ```

- [ ] 格式化 Server 错误
- [ ] 提供针对性建议
- [ ] 支持 JSON 错误输出
- [ ] 测试各种错误场景

### Task 8: 集成测试 (AC1-AC6)
- [ ] 创建测试脚本 `cmd/waterflow-cli/integration_status_test.sh`
  ```bash
  #!/bin/bash
  # CLI status 命令集成测试
  
  set -e
  
  CLI="./bin/waterflow"
  SERVER_URL="http://localhost:8088"
  
  # 检查 Server 是否运行
  if ! curl -sf "$SERVER_URL/health" > /dev/null; then
      echo "ERROR: Waterflow server not running at $SERVER_URL"
      exit 1
  fi
  
  echo "=== Test 1: Submit workflow first ==="
  WORKFLOW_ID=$($CLI submit --quiet testdata/valid/simple.yaml)
  echo "Submitted workflow: $WORKFLOW_ID"
  echo
  
  echo "=== Test 2: Basic status query (AC1) ==="
  $CLI status $WORKFLOW_ID | grep -q "Workflow:"
  $CLI status $WORKFLOW_ID | grep -q "Status:"
  echo "PASS"
  echo
  
  echo "=== Test 3: JSON output (AC4) ==="
  OUTPUT=$($CLI status --format json $WORKFLOW_ID)
  echo "$OUTPUT" | jq -e '.id' > /dev/null
  echo "$OUTPUT" | jq -e '.status' > /dev/null
  echo "PASS"
  echo
  
  echo "=== Test 4: YAML output (AC4) ==="
  $CLI status --format yaml $WORKFLOW_ID | grep -q "id:"
  echo "PASS"
  echo
  
  echo "=== Test 5: Quiet mode (AC4) ==="
  STATUS=$($CLI status --quiet $WORKFLOW_ID)
  if [ -z "$STATUS" ]; then
      echo "FAIL: No status returned"
      exit 1
  fi
  echo "PASS: Status = $STATUS"
  echo
  
  echo "=== Test 6: Workflow not found (AC5) ==="
  $CLI status nonexistent-id 2>&1 | grep -q "not found"
  if [ $? -ne 0 ]; then
      echo "FAIL: Should report not found error"
      exit 1
  fi
  echo "PASS"
  echo
  
  echo "=== Test 7: Watch mode (AC3) - manual test ==="
  echo "Run manually: $CLI status --watch --interval 2s $WORKFLOW_ID"
  echo "SKIP (requires manual verification)"
  echo
  
  echo "All automated tests passed!"
  ```

- [ ] 测试基础状态查询
- [ ] 测试输出格式
- [ ] 测试 quiet 模式
- [ ] 测试错误场景
- [ ] 手动测试 --watch 模式

### Task 9: 添加外部库依赖
- [ ] 更新 `go.mod`
  ```bash
  go get github.com/fatih/color@latest
  go get github.com/olekukonko/tablewriter@v0.0.5
  ```

- [ ] 验证依赖安装
- [ ] 更新 go.sum
- [ ] 确认版本符合 architecture.md 要求

### Task 10: 文档更新
- [ ] 更新 `cmd/waterflow-cli/README.md`
  ```markdown
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
  waterflow status --watch <workflow-id>
  
  # Custom refresh interval
  waterflow status --watch --interval 5s <workflow-id>
  
  # Compact mode (no steps)
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
  
  # Disable colors
  waterflow status --no-color <workflow-id>
  NO_COLOR=1 waterflow status <workflow-id>
  ```
  
  **Exit Codes:**
  - `0` - Status query successful
  - `1` - Query failed or workflow not found
  - `2` - Usage error (invalid flags or arguments)
  ```

- [ ] 添加使用示例到 `examples/cli/`
- [ ] 更新主项目 README

## Dev Notes

### 架构设计要点

**1. 命令参数组合:**
```go
// 参数组合
--watch         → 持续监控
--watch + --interval → 自定义刷新间隔
--compact       → 隐藏 Steps,仅显示 Jobs
--quiet         → 仅输出状态值
--no-color      → 禁用 ANSI 颜色
NO_COLOR=1      → 环境变量禁用颜色
```

**2. 工作流程:**
```
1. 验证工作流 ID (可选,宽松处理)
2. 创建 HTTP 客户端
3. 查询工作流状态 (GET /v1/workflows/{id})
4. 格式化输出 (text/json/yaml)
5. (可选) 持续监控 (--watch)
```

**3. 状态映射:**
```go
// Server 状态 → 显示符号/颜色
"running"    → "→" (蓝色)
"completed"  → "✓" (绿色)
"failed"     → "✗" (红色)
"cancelled"  → "⊗" (黄色)
"pending"    → "○" (灰色)
```

**4. 输出模式:**
- **text** (默认) - 人类可读,带符号和颜色
- **json** - 机器可读,完整 JSON
- **yaml** - YAML 格式
- **quiet** - 仅状态值,脚本友好

### 技术约束

**复用 Story 5.1:**
- ✅ HTTP 客户端基础 (`pkg/client/client.go`)
- ✅ 配置管理 (`pkg/config/config.go`)
- ✅ 输出格式化 (`pkg/output/formatter.go`)

**本 Story 新增:**
- 🆕 GetWorkflowStatus() 方法 (本 Story 实现,Story 5.3 暂未实现)
- 🆕 状态格式化和颜色支持
- 🆕 持续监控逻辑

**新增功能:**
- 持续监控 (--watch)
- 状态符号和颜色
- 紧凑模式显示
- YAML 输出格式

**依赖 Server API:**
- `GET /v1/workflows/{id}` - 查询工作流状态

**外部依赖:**
- `github.com/fatih/color` - 终端颜色支持
- `github.com/olekukonko/tablewriter@v0.0.5` - 表格格式化 (architecture.md 要求)
  - 用于 Jobs/Steps 列表的表格显示 (可选,增强可读性)
  - 示例用法见 Task 5 扩展实现

### 性能考虑

**轮询间隔:**
- 默认: 2 秒 (--watch)
- 最小: 1 秒
- 推荐: 2-5 秒 (平衡实时性和服务器负载)

**终端检测:**
- 检测是否为 TTY
- 非 TTY 自动禁用清屏和颜色
- 支持 NO_COLOR 环境变量

### 集成点

**与现有系统集成:**

1. **Server REST API (Story 1.9)**
   - 端点: `GET /v1/workflows/{id}`
   - 响应: WorkflowStatus (含 Jobs/Steps)

2. **CLI 基础 (Story 5.1)**
   - HTTP 客户端
   - 配置加载
   - 输出格式化

3. **submit 命令 (Story 5.3)**
   - 复用 GetWorkflowStatus() 方法
   - 类似的错误处理逻辑

### 测试策略

**单元测试:**
- ID 验证逻辑
- 时间格式化
- 状态符号映射
- 输出格式化

**集成测试 (需要 Server):**
- 基础状态查询
- 输出格式验证
- 错误场景测试

**手动测试:**
- --watch 持续监控
- 颜色显示效果
- Ctrl+C 中断处理
- 用户体验测试

### 后续 Story 依赖

**Story 5.5 (logs 命令) 可能需要:**
- 类似的输出格式化逻辑
- 类似的 --follow 持续监控模式

### 参考文档

**内部文档:**
- [Story 5.1 - CLI 基础框架](./5-1-cli-framework.md)
- [Story 5.3 - submit 命令](./5-3-submit-command.md)
- [Story 1.9 - REST API](./1-9-workflow-management-api.md)

**外部参考:**
- kubectl get: https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#get
- docker ps: https://docs.docker.com/engine/reference/commandline/ps/
- github.com/fatih/color: https://github.com/fatih/color

## Definition of Done

### 代码完成标准

- [ ] 所有 Task 完成并测试通过
- [ ] 单元测试覆盖率 >80%
- [ ] 代码通过 `golangci-lint` 检查
- [ ] 无编译警告和错误

### 功能验证标准

- [ ] AC1: 基础状态查询
  - [ ] `waterflow status <id>` 查询成功
  - [ ] 显示工作流名称、状态、时间
  - [ ] 返回退出码 0
- [ ] AC2: 显示执行进度
  - [ ] 显示 Jobs 列表
  - [ ] 显示 Steps 层级
  - [ ] --compact 模式隐藏 Steps
- [ ] AC3: 持续监控模式
  - [ ] `--watch` 持续刷新
  - [ ] 自定义刷新间隔
  - [ ] 工作流完成后退出
  - [ ] Ctrl+C 正常退出
- [ ] AC4: 格式化输出
  - [ ] text 格式 (默认)
  - [ ] json 格式
  - [ ] yaml 格式
  - [ ] quiet 模式
- [ ] AC5: 友好错误处理
  - [ ] 工作流不存在提示
  - [ ] Server 连接失败提示
  - [ ] 每种错误有建议
- [ ] AC6: 状态符号和颜色
  - [ ] 状态符号显示正确
  - [ ] 颜色高亮正常
  - [ ] --no-color 禁用颜色
  - [ ] NO_COLOR 环境变量支持

### 测试验证标准

- [ ] 所有单元测试通过
- [ ] 集成测试脚本通过 (需要 Server)
- [ ] 手动测试所有 AC
- [ ] 与 Server 端到端测试

### 文档完成标准

- [ ] README 包含 status 命令文档
- [ ] `--help` 输出清晰完整
- [ ] 使用示例完整
- [ ] 错误信息文档完整

### 交付标准

- [ ] status 命令可执行
- [ ] 基础查询正常工作
- [ ] 持续监控功能正常
- [ ] 所有输出格式正常工作
- [ ] 颜色显示正常工作
- [ ] 代码已合并到主分支
- [ ] Sprint status 更新为 `done`

## References

### 源文档
- [Source: docs/epics.md#Story 5.4](../epics.md) - Epic 分解中的 Story 5.4
- [Source: docs/prd.md#Epic 5](../prd.md) - PRD Epic 5 定义

### 前置 Stories
- [Source: docs/sprint-artifacts/5-1-cli-framework.md](./5-1-cli-framework.md) - CLI 基础框架
- [Source: docs/sprint-artifacts/5-3-cli-submit-command.md](./5-3-cli-submit-command.md) - submit 命令
- [Source: docs/sprint-artifacts/1-9-workflow-management-api.md](./1-9-workflow-management-api.md) - REST API
- [Source: docs/sprint-artifacts/1-8-temporal-sdk-integration.md](./1-8-temporal-sdk-integration.md) - Temporal SDK

### 代码参考
- [Source: internal/api/workflow_handler.go](../../internal/api/workflow_handler.go) - GetWorkflowStatus API
- [Source: pkg/temporal/history_parser.go](../../pkg/temporal/history_parser.go) - Event History 解析

### 外部参考
- kubectl get: https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#get
- docker ps: https://docs.docker.com/engine/reference/commandline/ps/
- fatih/color: https://github.com/fatih/color

## Dev Agent Record

### Agent Model Used

待 Dev Agent 执行时填写

### Completion Notes List

待 Dev Agent 执行时填写

### File List

待 Dev Agent 执行时填写
