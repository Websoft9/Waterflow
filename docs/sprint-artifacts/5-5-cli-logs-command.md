# Story 5.5: CLI logs 命令

Status: ready-for-dev

## Story

As a **工作流用户**,  
I want **查看工作流日志**,  
So that **调试执行问题**。

## Context

这是 Epic 5 (客户端工具和 SDK) 的第五个 Story,在 submit(5.3) 和 status(5.4) 命令之后,实现**日志查看命令**。该命令让用户能够通过命令行快速查看工作流执行日志,是故障排查和调试的核心工具。

**前置依赖:**
- ✅ Story 5.1 - CLI 基础框架 (HTTP 客户端、输出格式化)
- ✅ Story 5.4 - status 命令 (状态查询和颜色显示逻辑可复用)
- ✅ Story 1.9 - REST API 服务 (`GET /v1/workflows/{id}/logs` 端点)
- ✅ Story 1.8 - Temporal SDK 集成 (Event History 日志重建)

**Epic 背景:**  
Epic 5 专注于**客户端工具和 SDK**。logs 命令是工作流调试的核心工具,支持多种模式:
1. **历史日志** - 查询已完成工作流的所有日志 (默认)
2. **实时跟踪** - `--follow` 参数实时显示新日志 (类似 `tail -f`)
3. **日志过滤** - 按级别、Job、Step 过滤日志
4. **彩色输出** - 根据日志级别高亮显示

本 Story 实现日志查看功能,配合 submit 和 status 命令形成完整的工作流管理工具链。

**业务价值:**
- 🎯 **快速调试** - 立即查看工作流执行日志
- 🎯 **问题定位** - 根据错误日志快速定位问题
- 🎯 **实时监控** - --follow 模式实时跟踪执行
- 🎯 **精准过滤** - 按级别/Job/Step 过滤日志

**技术定位:**
- **不是** 日志聚合系统 (仅显示单个工作流日志)
- **不是** 状态查询工具 (status 命令负责)
- **是** Server REST API 的便捷封装
- **是** 故障排查和调试的主要入口

**复用现有组件:**
- Story 5.1: HTTP 客户端、配置管理、输出格式化
- Story 5.4: 颜色显示逻辑、终端检测
- Story 1.9: `GET /v1/workflows/{id}/logs` API

**与其他命令的关系:**
- `submit` → 提交工作流,返回 ID
- `status` → 查询工作流状态
- `logs` → 查看工作流执行日志

**日志来源:**
- Temporal Event History 重建 (历史日志)
- 从 ActivityTaskScheduled/Started/Completed/Failed 事件提取
- 包含时间戳、级别、Job/Step 信息

## Acceptance Criteria

### AC1: 基础日志查询

**Given** 存在有效的工作流 ID  
**When** 执行 `waterflow logs <workflow-id>`  
**Then** 显示工作流执行日志  
**And** 日志包含时间戳、级别、Job、Step、消息  
**And** 按时间顺序显示  
**And** 默认显示最后 100 行 (可通过 --tail 配置)  
**And** 返回退出码 0

**说明:**
- 使用 `GET /v1/workflows/{id}/logs` (Story 1.9 API)
- 响应为 JSON Array,客户端逐条格式化输出

**日志查询成功示例:**
```bash
$ waterflow logs 550e8400-e29b-41d4-a716-446655440000
[2026-01-04 10:30:45] INFO  Workflow started: Deploy Application
[2026-01-04 10:30:46] INFO  [build] Job started on queue: linux-amd64
[2026-01-04 10:30:47] INFO  [build.checkout] Step started
[2026-01-04 10:30:48] INFO  [build.checkout] Cloning repository...
[2026-01-04 10:30:50] INFO  [build.checkout] Step completed (3s)
[2026-01-04 10:30:51] INFO  [build.compile] Step started
[2026-01-04 10:30:52] INFO  [build.compile] Running: make build
[2026-01-04 10:30:55] INFO  [build.compile] Build successful
[2026-01-04 10:30:56] INFO  [build.compile] Step completed (5s)
[2026-01-04 10:30:57] INFO  [build] Job completed (11s)
[2026-01-04 10:30:58] INFO  [deploy] Job started on queue: linux-amd64
[2026-01-04 10:30:59] INFO  [deploy.push] Step started
[2026-01-04 10:31:00] ERROR [deploy.push] Failed to push image
[2026-01-04 10:31:00] ERROR [deploy.push] Error: connection timeout
[2026-01-04 10:31:01] ERROR [deploy.push] Step failed (2s)
[2026-01-04 10:31:02] ERROR [deploy] Job failed (4s)
[2026-01-04 10:31:03] ERROR Workflow failed: deployment error

$ echo $?
0
```

**限制行数:**
```bash
# 显示最后 50 行
$ waterflow logs --tail 50 550e8400-e29b-41d4-a716-446655440000

# 显示所有日志
$ waterflow logs --tail -1 550e8400-e29b-41d4-a716-446655440000
```

### AC2: 日志级别过滤

**Given** 工作流产生不同级别的日志  
**When** 使用 `--level` 参数  
**Then** 仅显示指定级别的日志  
**And** 支持多个级别 (逗号分隔)  
**And** 支持级别: info, warn, error, debug

**级别过滤示例:**
```bash
# 仅显示错误日志
$ waterflow logs --level error 550e8400-e29b-41d4-a716-446655440000
[2026-01-04 10:31:00] ERROR [deploy.push] Failed to push image
[2026-01-04 10:31:00] ERROR [deploy.push] Error: connection timeout
[2026-01-04 10:31:01] ERROR [deploy.push] Step failed (2s)
[2026-01-04 10:31:02] ERROR [deploy] Job failed (4s)
[2026-01-04 10:31:03] ERROR Workflow failed: deployment error

# 显示错误和警告
$ waterflow logs --level error,warn 550e8400-e29b-41d4-a716-446655440000
[2026-01-04 10:30:55] WARN  [build.compile] Using cached dependencies
[2026-01-04 10:31:00] ERROR [deploy.push] Failed to push image
[2026-01-04 10:31:00] ERROR [deploy.push] Error: connection timeout

# 调试日志 (包含详细信息)
$ waterflow logs --level debug 550e8400-e29b-41d4-a716-446655440000
[2026-01-04 10:30:47] DEBUG [build.checkout] Git clone command: git clone https://...
[2026-01-04 10:30:48] DEBUG [build.checkout] Using SSH key: ~/.ssh/id_rsa
```

### AC3: Job/Step 过滤

**Given** 工作流包含多个 Jobs 和 Steps  
**When** 使用 `--job` 或 `--step` 参数  
**Then** 仅显示指定 Job 或 Step 的日志  
**And** 支持按 Job 名称过滤  
**And** 支持按 Step 名称过滤

**Job 过滤示例:**
```bash
# 仅显示 deploy Job 的日志
$ waterflow logs --job deploy 550e8400-e29b-41d4-a716-446655440000
[2026-01-04 10:30:58] INFO  [deploy] Job started on queue: linux-amd64
[2026-01-04 10:30:59] INFO  [deploy.push] Step started
[2026-01-04 10:31:00] ERROR [deploy.push] Failed to push image
[2026-01-04 10:31:00] ERROR [deploy.push] Error: connection timeout
[2026-01-04 10:31:01] ERROR [deploy.push] Step failed (2s)
[2026-01-04 10:31:02] ERROR [deploy] Job failed (4s)
```

**Step 过滤示例:**
```bash
# 仅显示特定 Step 的日志
$ waterflow logs --step checkout 550e8400-e29b-41d4-a716-446655440000
[2026-01-04 10:30:47] INFO  [build.checkout] Step started
[2026-01-04 10:30:48] INFO  [build.checkout] Cloning repository...
[2026-01-04 10:30:50] INFO  [build.checkout] Step completed (3s)
```

**组合过滤:**
```bash
# 显示 deploy Job 的错误日志
$ waterflow logs --job deploy --level error 550e8400-e29b-41d4-a716-446655440000
[2026-01-04 10:31:00] ERROR [deploy.push] Failed to push image
[2026-01-04 10:31:00] ERROR [deploy.push] Error: connection timeout
[2026-01-04 10:31:01] ERROR [deploy.push] Step failed (2s)
[2026-01-04 10:31:02] ERROR [deploy] Job failed (4s)
```

### AC4: 实时日志跟踪 (--follow)

**Given** 工作流正在执行  
**When** 使用 `--follow` 参数  
**Then** 实时显示新日志 (类似 tail -f)  
**And** 优先使用 SSE 流式获取 (Story 1.9 `stream=true`)  
**And** SSE 不可用时降级为轮询 (2s 间隔)  
**And** 持续跟踪直到工作流完成  
**And** 工作流完成后自动退出  
**And** 支持 Ctrl+C 中断

**实时跟踪示例:**
```bash
$ waterflow logs --follow 550e8400-e29b-41d4-a716-446655440000
[2026-01-04 10:30:45] INFO  Workflow started: Deploy Application
[2026-01-04 10:30:46] INFO  [build] Job started on queue: linux-amd64
[2026-01-04 10:30:47] INFO  [build.checkout] Step started
[2026-01-04 10:30:48] INFO  [build.checkout] Cloning repository...

# 持续显示新日志...

[2026-01-04 10:30:50] INFO  [build.checkout] Step completed (3s)
[2026-01-04 10:30:51] INFO  [build.compile] Step started

# 工作流完成后

[2026-01-04 10:31:20] INFO  Workflow completed successfully

$ echo $?
0
```

**失败场景:**
```bash
$ waterflow logs --follow 550e8400-e29b-41d4-a716-446655440000
[2026-01-04 10:30:45] INFO  Workflow started
...
[2026-01-04 10:31:00] ERROR [deploy.push] Failed to push image
[2026-01-04 10:31:00] ERROR [deploy.push] Error: connection timeout
[2026-01-04 10:31:03] ERROR Workflow failed: deployment error

$ echo $?
1
```

**自定义轮询间隔:**
```bash
$ waterflow logs --follow --poll-interval 1s 550e8400-e29b-41d4-a716-446655440000
# 每 1 秒查询新日志 (默认 2 秒)
```

### AC5: 彩色日志输出

**Given** 在终端显示日志  
**When** 输出到 TTY  
**Then** 根据日志级别使用颜色高亮  
**And** 支持 `--no-color` 参数禁用颜色  
**And** 支持 NO_COLOR 环境变量

**颜色定义:**
- `DEBUG` - 灰色/白色
- `INFO` - 蓝色
- `WARN` - 黄色
- `ERROR` - 红色

**彩色输出示例:**
```bash
$ waterflow logs 550e8400-e29b-41d4-a716-446655440000
[2026-01-04 10:30:45] INFO  Workflow started  (蓝色)
[2026-01-04 10:30:55] WARN  Using cached dependencies  (黄色)
[2026-01-04 10:31:00] ERROR Failed to push image  (红色)
```

**禁用颜色:**
```bash
$ waterflow logs --no-color 550e8400-e29b-41d4-a716-446655440000
# 纯文本输出,无 ANSI 颜色码

# 或通过环境变量
$ NO_COLOR=1 waterflow logs 550e8400-e29b-41d4-a716-446655440000
```

### AC6: 格式化输出选项

**Given** 需要集成到自动化流程  
**When** 使用输出格式参数  
**Then** 支持 `--format text` (默认,人类可读)  
**And** 支持 `--format json` (JSON Lines 格式,机器可读)  
**And** 支持 `--timestamps` 参数控制时间戳显示

**JSON 输出示例:**
```bash
$ waterflow logs --format json 550e8400-e29b-41d4-a716-446655440000
{"timestamp":"2026-01-04T10:30:45Z","level":"info","message":"Workflow started: Deploy Application"}
{"timestamp":"2026-01-04T10:30:46Z","level":"info","job":"build","message":"Job started on queue: linux-amd64"}
{"timestamp":"2026-01-04T10:30:47Z","level":"info","job":"build","step":"checkout","message":"Step started"}
{"timestamp":"2026-01-04T10:31:00Z","level":"error","job":"deploy","step":"push","message":"Failed to push image","error":"connection timeout"}

$ echo $?
0
```

**禁用时间戳:**
```bash
$ waterflow logs --no-timestamps 550e8400-e29b-41d4-a716-446655440000
INFO  Workflow started: Deploy Application
INFO  [build] Job started on queue: linux-amd64
INFO  [build.checkout] Step started
```

**紧凑格式 (仅消息):**
```bash
$ waterflow logs --format compact 550e8400-e29b-41d4-a716-446655440000
Workflow started: Deploy Application
[build] Job started on queue: linux-amd64
[build.checkout] Step started
[build.checkout] Cloning repository...
```

### AC7: 友好的错误处理

**Given** 查询日志时遇到错误  
**When** 发生各种错误情况  
**Then** 显示清晰的错误信息和建议

**错误场景覆盖:**

**1. 工作流不存在:**
```bash
$ waterflow logs nonexistent-id
Error: Workflow not found
  Workflow ID: nonexistent-id

Suggestion: Check workflow ID or use 'waterflow status' to verify workflow exists
Exit code: 1
```

**2. Server 连接失败:**
```bash
$ waterflow logs 550e8400-e29b-41d4-a716-446655440000
Error: Failed to connect to server
  URL: http://localhost:8088
  Cause: dial tcp 127.0.0.1:8088: connect: connection refused

Suggestion:
  1. Check if Waterflow server is running (docker-compose up)
  2. Verify server URL with --server flag or config file

Exit code: 1
```

**3. 无效的参数:**
```bash
$ waterflow logs --tail 2000 550e8400-e29b-41d4-a716-446655440000
Error: Invalid parameter
  Parameter: tail
  Value: 2000
  Reason: tail must be between 1 and 1000

Suggestion: Use --tail 1000 or omit --tail for default (100 lines)

Exit code: 2
```

**4. 无日志可用:**
```bash
$ waterflow logs --level error 550e8400-e29b-41d4-a716-446655440000
No logs found matching filter criteria

Filters applied:
  Level: error

Suggestion: Try removing filters or check workflow status

$ echo $?
0
```

## Tasks / Subtasks

### Task 1: logs 子命令框架 (AC1)
- [ ] 创建 `cmd/waterflow-cli/cmd/logs.go`
  ```go
  package cmd
  
  import (
      "fmt"
      "os"
      "strings"
      "time"
      
      "github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/client"
      "github.com/spf13/cobra"
      "go.uber.org/zap"
  )
  
  var (
      logsFollow       bool
      logsTail         int
      logsLevel        string
      logsJob          string
      logsStep         string
      logsFormat       string
      logsNoColor      bool
      logsNoTimestamps bool
      logsPollInterval string
  )
  
  func newLogsCmd() *cobra.Command {
      cmd := &cobra.Command{
          Use:   "logs <workflow-id>",
          Short: "View workflow execution logs",
          Long: `View execution logs from a workflow.
  
  Displays logs from all jobs and steps in chronological order.
  Use --follow to stream logs in real-time.
  
  Examples:
    # View logs
    waterflow logs 550e8400-e29b-41d4-a716-446655440000
    
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
    
    # JSON output
    waterflow logs --format json <workflow-id>`,
          Args: cobra.ExactArgs(1),
          RunE: runLogs,
      }
      
      cmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow logs in real-time (like tail -f)")
      cmd.Flags().IntVar(&logsTail, "tail", 100, "Number of lines to show from the end (1-1000, -1 for all)")
      cmd.Flags().StringVar(&logsLevel, "level", "", "Filter by log level (info,warn,error,debug)")
      cmd.Flags().StringVar(&logsJob, "job", "", "Filter by job name")
      cmd.Flags().StringVar(&logsStep, "step", "", "Filter by step name")
      cmd.Flags().StringVar(&logsFormat, "format", "text", "Output format (text, json, compact)")
      cmd.Flags().BoolVar(&logsNoColor, "no-color", false, "Disable colored output")
      cmd.Flags().BoolVar(&logsNoTimestamps, "no-timestamps", false, "Hide timestamps")
      cmd.Flags().StringVar(&logsPollInterval, "poll-interval", "2s", "Polling interval for --follow")
      
      return cmd
  }
  
  func runLogs(cmd *cobra.Command, args []string) error {
      workflowID := args[0]
      
      // 验证参数
      if logsTail < -1 || logsTail == 0 || logsTail > 1000 {
          return fmt.Errorf("invalid --tail value: %d (must be -1 or 1-1000)", logsTail)
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
      noColor := logsNoColor || os.Getenv("NO_COLOR") != "" || !isTerminal()
      
      // 创建日志格式化器
      formatter := newLogsFormatter(logsFormat, noColor, logsNoTimestamps)
      
      // 跟踪模式
      if logsFollow {
          return followLogs(httpClient, workflowID, formatter)
      }
      
      // 一次性查询
      logs, err := httpClient.GetWorkflowLogs(workflowID, buildLogsQuery())
      if err != nil {
          return formatLogsError(err, logsFormat)
      }
      
      // 输出日志
      if len(logs) == 0 {
          return displayNoLogsMessage()
      }
      
      for _, log := range logs {
          if err := formatter.PrintLog(log); err != nil {
              return err
          }
      }
      
      return nil
  }
  
  // buildLogsQuery 构建查询参数
  func buildLogsQuery() client.LogsQuery {
      tail := logsTail
      if tail == -1 {
          tail = 1000 // Story 1.9 API 最大值
      }
      
      return client.LogsQuery{
          Tail:  tail,
          Level: logsLevel,
          Job:   logsJob,
          Step:  logsStep,
      }
  }
  ```

- [ ] 定义命令参数
  - `--follow, -f` - 实时跟踪
  - `--tail` - 显示行数
  - `--level` - 级别过滤
  - `--job` - Job 过滤
  - `--step` - Step 过滤
  - `--format` - 输出格式
  - `--no-color` - 禁用颜色
  - `--no-timestamps` - 隐藏时间戳
  - `--poll-interval` - 轮询间隔

- [ ] 注册到根命令

### Task 2: HTTP 客户端 GetWorkflowLogs 方法 (AC1)

**Developer Context:**
本方法在 Story 5.5 中首次实现。包含两个方法:
1. `GetWorkflowLogs()` - 普通查询,返回 JSON 数组
2. `StreamWorkflowLogs()` - SSE 流式查询,用于 --follow 模式

根据 Story 1.9 API:
- 普通查询返回 JSON Array: `[{...}, {...}]`
- SSE 流返回 Server-Sent Events 格式

- [ ] 扩展 `cmd/waterflow-cli/pkg/client/client.go`
  ```go
  package client
  
  import (
      "bufio"
      "context"
      "encoding/json"
      "fmt"
      "io"
      "net/http"
      "net/url"
      "os"
      "strconv"
      "strings"
  )
  
  // LogEntry 日志条目 (对应 Story 1.9 API response schema)
  type LogEntry struct {
      Timestamp string `json:"timestamp"`
      Level     string `json:"level"`
      Job       string `json:"job,omitempty"`
      Step      string `json:"step,omitempty"`
      Message   string `json:"message"`
      Error     string `json:"error,omitempty"`
  }
  
  // LogsQuery 日志查询参数
  type LogsQuery struct {
      Tail  int
      Level string
      Job   string
      Step  string
  }
  
  // GetWorkflowLogs 查询工作流日志
  func (c *Client) GetWorkflowLogs(workflowID string, query LogsQuery) ([]LogEntry, error) {
      // 构建查询参数
      queryParams := url.Values{}
      if query.Tail > 0 {
          queryParams.Set("tail", strconv.Itoa(query.Tail))
      }
      if query.Level != "" {
          queryParams.Set("level", query.Level)
      }
      if query.Job != "" {
          queryParams.Set("job", query.Job)
      }
      if query.Step != "" {
          queryParams.Set("step", query.Step)
      }
      
      logsURL := fmt.Sprintf("%s/v1/workflows/%s/logs", c.baseURL, workflowID)
      if len(queryParams) > 0 {
          logsURL += "?" + queryParams.Encode()
      }
      
      httpReq, err := http.NewRequestWithContext(
          context.Background(),
          http.MethodGet,
          logsURL,
          nil,
      )
      if err != nil {
          return nil, fmt.Errorf("failed to create request: %w", err)
      }
      
      if c.apiKey != "" {
          httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
      }
      
      if c.debug {
          fmt.Printf("DEBUG: GET %s\n", logsURL)
      }
      
      resp, err := c.httpClient.Do(httpReq)
      if err != nil {
          return nil, fmt.Errorf("request failed: %w", err)
      }
      defer resp.Body.Close()
      
      if resp.StatusCode != http.StatusOK {
          return nil, c.parseError(resp)
      }
      
      // 解析 JSON 数组格式 (根据 Story 1.9 API 定义)
      var logs []LogEntry
      if err := json.NewDecoder(resp.Body).Decode(&logs); err != nil {
          return nil, fmt.Errorf("failed to parse logs: %w", err)
      }
      
      return logs, nil
  }
  
  // StreamWorkflowLogs 实时流式获取日志 (SSE)
  // NOTE: 用于 --follow 模式,使用 Story 1.9 的 SSE 特性
  func (c *Client) StreamWorkflowLogs(workflowID string, query LogsQuery) (<-chan LogEntry, <-chan error, error) {
      // 构建查询参数
      queryParams := url.Values{}
      queryParams.Set("stream", "true")
      if query.Level != "" {
          queryParams.Set("level", query.Level)
      }
      if query.Job != "" {
          queryParams.Set("job", query.Job)
      }
      if query.Step != "" {
          queryParams.Set("step", query.Step)
      }
      
      logsURL := fmt.Sprintf("%s/v1/workflows/%s/logs?%s", c.baseURL, workflowID, queryParams.Encode())
      
      httpReq, err := http.NewRequestWithContext(
          context.Background(),
          http.MethodGet,
          logsURL,
          nil,
      )
      if err != nil {
          return nil, nil, fmt.Errorf("failed to create request: %w", err)
      }
      
      httpReq.Header.Set("Accept", "text/event-stream")
      if c.apiKey != "" {
          httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
      }
      
      if c.debug {
          fmt.Printf("DEBUG: GET %s (SSE)\n", logsURL)
      }
      
      resp, err := c.httpClient.Do(httpReq)
      if err != nil {
          return nil, nil, fmt.Errorf("request failed: %w", err)
      }
      
      if resp.StatusCode != http.StatusOK {
          resp.Body.Close()
          return nil, nil, c.parseError(resp)
      }
      
      logChan := make(chan LogEntry, 10)
      errChan := make(chan error, 1)
      
      go func() {
          defer resp.Body.Close()
          defer close(logChan)
          defer close(errChan)
          
          reader := bufio.NewReader(resp.Body)
          for {
              line, err := reader.ReadString('\n')
              if err != nil {
                  if err != io.EOF {
                      errChan <- fmt.Errorf("failed to read SSE stream: %w", err)
                  }
                  return
              }
              
              line = strings.TrimSpace(line)
              if strings.HasPrefix(line, "data: ") {
                  data := line[6:] // 移除 "data: " 前缀
                  var entry LogEntry
                  if err := json.Unmarshal([]byte(data), &entry); err != nil {
                      if c.debug {
                          fmt.Fprintf(os.Stderr, "WARN: Invalid SSE log entry: %s\n", data)
                      }
                      continue
                  }
                  logChan <- entry
              }
          }
      }()
      
      return logChan, errChan, nil
  }
  ```

- [ ] 实现 GET /v1/workflows/{id}/logs 调用
- [ ] 解析 JSON Lines 响应
- [ ] 处理查询参数
- [ ] 单元测试 `pkg/client/logs_test.go`

### Task 3: 实时日志跟踪 (AC4)
- [ ] 实现跟踪函数
  ```go
  // cmd/waterflow-cli/cmd/logs.go
  
  import (
      "context"
      "os"
      "os/signal"
      "syscall"
      "time"
  )
  
  // followLogs 实时跟踪日志 (SSE 优先)
  func followLogs(c *client.Client, workflowID string, formatter *LogsFormatter) error {
      // 优先使用 SSE 流式获取 (Story 1.9 支持)
      logChan, errChan, err := c.StreamWorkflowLogs(workflowID, buildLogsQuery())
      if err == nil {
          return followLogsWithSSE(c, workflowID, logChan, errChan, formatter)
      }
      
      // SSE 不可用时降级为轮询模式
      if logsFormat != "json" {
          fmt.Fprintf(os.Stderr, "WARN: SSE not available, using polling mode\n")
      }
      return followLogsWithPolling(c, workflowID, formatter)
  }
  
  // followLogsWithSSE 使用 SSE 流式跟踪日志
  func followLogsWithSSE(c *client.Client, workflowID string, logChan <-chan client.LogEntry, errChan <-chan error, formatter *LogsFormatter) error {
      // 设置信号处理
      ctx, cancel := context.WithCancel(context.Background())
      defer cancel()
      
      sigChan := make(chan os.Signal, 1)
      signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
      defer signal.Stop(sigChan)
      
      go func() {
          <-sigChan
          fmt.Fprintln(os.Stderr, "\nInterrupted by user")
          cancel()
      }()
      
      // 定期检查工作流状态
      statusTicker := time.NewTicker(5 * time.Second)
      defer statusTicker.Stop()
      
      for {
          select {
          case <-ctx.Done():
              return nil
              
          case log, ok := <-logChan:
              if !ok {
                  // SSE stream 关闭,检查工作流状态
                  return checkWorkflowCompletion(c, workflowID)
              }
              if err := formatter.PrintLog(log); err != nil {
                  return err
              }
              
          case err := <-errChan:
              if err != nil {
                  return fmt.Errorf("SSE stream error: %w", err)
              }
              
          case <-statusTicker.C:
              // 定期检查工作流是否完成
              status, err := c.GetWorkflowStatus(workflowID)
              if err == nil && isTerminalStatus(status.Status) {
                  // 等待剩余日志
                  time.Sleep(1 * time.Second)
                  return checkWorkflowCompletion(c, workflowID)
              }
          }
      }
  }
  
  // followLogsWithPolling 轮询模式跟踪日志 (降级方案)
  func followLogsWithPolling(c *client.Client, workflowID string, formatter *LogsFormatter) error {
      // 解析轮询间隔
      interval, err := time.ParseDuration(logsPollInterval)
      if err != nil {
          return fmt.Errorf("invalid poll-interval: %s (expected: 1s, 2s, 5s)", logsPollInterval)
      }
      
      if interval < 1*time.Second {
          return fmt.Errorf("poll-interval too short: %s (minimum: 1s)", logsPollInterval)
      }
      
      // 设置信号处理
      ctx, cancel := context.WithCancel(context.Background())
      defer cancel()
      
      sigChan := make(chan os.Signal, 1)
      signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
      defer signal.Stop(sigChan)
      
      go func() {
          <-sigChan
          fmt.Fprintln(os.Stderr, "\nInterrupted by user")
          cancel()
      }()
      
      // 使用计数器而非时间戳去重 (API 不支持 since 参数)
      lastSeenCount := 0
      ticker := time.NewTicker(interval)
      defer ticker.Stop()
      
      // 首次查询
      logs, err := c.GetWorkflowLogs(workflowID, buildLogsQuery())
      if err != nil {
          return formatLogsError(err, logsFormat)
      }
      
      for _, log := range logs {
          if err := formatter.PrintLog(log); err != nil {
              return err
          }
      }
      lastSeenCount = len(logs)
      
      for {
          select {
          case <-ctx.Done():
              return nil
              
          case <-ticker.C:
              // 查询新日志 (使用计数器去重)
              // NOTE: Story 1.9 API 不支持 since/offset 参数,需要客户端去重
              newLogs, err := c.GetWorkflowLogs(workflowID, buildLogsQuery())
              if err != nil {
                  return formatLogsError(err, logsFormat)
              }
              
              // 仅显示新增的日志
              if len(newLogs) > lastSeenCount {
                  for _, log := range newLogs[lastSeenCount:] {
                      if err := formatter.PrintLog(log); err != nil {
                          return err
                      }
                  }
                  lastSeenCount = len(newLogs)
              }
              
              // 检查工作流是否完成
              status, err := c.GetWorkflowStatus(workflowID)
              if err == nil && isTerminalStatus(status.Status) {
                  return checkWorkflowCompletion(c, workflowID)
              }
          }
      }
  }
  
  // checkWorkflowCompletion 检查工作流完成状态并显示消息
  func checkWorkflowCompletion(c *client.Client, workflowID string) error {
      status, err := c.GetWorkflowStatus(workflowID)
      if err != nil {
          return err
      }
      
      if logsFormat == "json" {
          return nil // JSON 模式不显示完成消息
      }
      
      fmt.Fprintln(os.Stderr, "")
      if status.Status == "completed" {
          fmt.Fprintf(os.Stderr, "✓ Workflow completed successfully\n")
          return nil
      } else {
          fmt.Fprintf(os.Stderr, "✗ Workflow %s\n", status.Status)
          return &ExitError{Code: 1}
      }
  }
  ```

- [ ] 实现 SSE 流式跟踪 (优先方案)
- [ ] 实现轮询降级逻辑
- [ ] 使用计数器去重 (API 无 since 参数)
- [ ] 检测工作流完成
- [ ] 信号处理 (Ctrl+C)
- [ ] SSE stream 解析和错误处理

### Task 4: 日志格式化 - Text 格式 (AC1, AC5)

**Developer Context:**
复用 Story 5.4 的颜色逻辑:
- `getLevelColor()` 方法 - 级别颜色映射 (可直接复用或参考实现)
- `isTerminal()` 方法 - TTY 检测
- 颜色定义保持一致: error=红色, warn=黄色, info=蓝色, debug=灰色

- [ ] 创建 `cmd/waterflow-cli/pkg/output/logs.go`
  ```go
  package output
  
  import (
      "fmt"
      "strings"
      "time"
      
      "github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/client"
      "github.com/fatih/color"
  )
  
  // LogsFormatter 日志格式化器
  type LogsFormatter struct {
      format       string
      noColor      bool
      noTimestamps bool
  }
  
  // NewLogsFormatter 创建日志格式化器
  func newLogsFormatter(format string, noColor bool, noTimestamps bool) *LogsFormatter {
      return &LogsFormatter{
          format:       format,
          noColor:      noColor,
          noTimestamps: noTimestamps,
      }
  }
  
  // PrintLog 打印日志条目
  func (f *LogsFormatter) PrintLog(log client.LogEntry) error {
      switch f.format {
      case "json":
          return f.printLogJSON(log)
      case "compact":
          return f.printLogCompact(log)
      default:
          return f.printLogText(log)
      }
  }
  
  func (f *LogsFormatter) printLogText(log client.LogEntry) error {
      var parts []string
      
      // 时间戳
      if !f.noTimestamps {
          ts := formatTimestamp(log.Timestamp)
          parts = append(parts, fmt.Sprintf("[%s]", ts))
      }
      
      // 日志级别 (带颜色)
      levelColor := f.getLevelColor(log.Level)
      levelStr := fmt.Sprintf("%-5s", strings.ToUpper(log.Level))
      if f.noColor {
          parts = append(parts, levelStr)
      } else {
          parts = append(parts, levelColor.Sprint(levelStr))
      }
      
      // Job/Step 信息
      if log.Job != "" {
          if log.Step != "" {
              parts = append(parts, fmt.Sprintf("[%s.%s]", log.Job, log.Step))
          } else {
              parts = append(parts, fmt.Sprintf("[%s]", log.Job))
          }
      }
      
      // 消息
      message := log.Message
      if log.Error != "" {
          message = fmt.Sprintf("%s: %s", message, log.Error)
      }
      
      // 应用级别颜色到消息
      if !f.noColor && log.Level == "error" {
          message = levelColor.Sprint(message)
      }
      
      parts = append(parts, message)
      
      fmt.Println(strings.Join(parts, " "))
      return nil
  }
  
  func (f *LogsFormatter) printLogCompact(log client.LogEntry) error {
      // 仅显示消息和 Job/Step
      var parts []string
      
      if log.Job != "" {
          if log.Step != "" {
              parts = append(parts, fmt.Sprintf("[%s.%s]", log.Job, log.Step))
          } else {
              parts = append(parts, fmt.Sprintf("[%s]", log.Job))
          }
      }
      
      message := log.Message
      if log.Error != "" {
          message = fmt.Sprintf("%s: %s", message, log.Error)
      }
      parts = append(parts, message)
      
      fmt.Println(strings.Join(parts, " "))
      return nil
  }
  
  func (f *LogsFormatter) printLogJSON(log client.LogEntry) error {
      enc := json.NewEncoder(os.Stdout)
      return enc.Encode(log)
  }
  
  func (f *LogsFormatter) getLevelColor(level string) *color.Color {
      if f.noColor {
          return color.New()
      }
      
      switch strings.ToLower(level) {
      case "error":
          return color.New(color.FgRed)
      case "warn", "warning":
          return color.New(color.FgYellow)
      case "info":
          return color.New(color.FgBlue)
      case "debug":
          return color.New(color.FgHiBlack)
      default:
          return color.New()
      }
  }
  
  // formatTimestamp 格式化时间戳 (ISO 8601 → 本地时间)
  func formatTimestamp(ts string) string {
      t, err := time.Parse(time.RFC3339, ts)
      if err != nil {
          return ts
      }
      
      // 格式: 2026-01-04 10:30:45
      return t.Local().Format("2006-01-02 15:04:05")
  }
  ```

- [ ] 实现 text 格式输出
- [ ] 实现级别颜色
- [ ] 实现时间戳格式化
- [ ] 实现 Job/Step 显示

### Task 5: 日志格式化 - JSON/Compact 格式 (AC6)
- [ ] 扩展 `cmd/waterflow-cli/pkg/output/logs.go`
  ```go
  package output
  
  import (
      "encoding/json"
      "os"
  )
  
  // printLogJSON 已在 Task 4 中实现
  // printLogCompact 已在 Task 4 中实现
  ```

- [ ] JSON Lines 格式输出
- [ ] Compact 格式输出
- [ ] 单元测试输出格式

### Task 6: 日志过滤逻辑 (AC2, AC3)
- [ ] 实现过滤参数验证
  ```go
  // cmd/waterflow-cli/cmd/logs.go
  
  // validateLogsParams 验证日志参数
  func validateLogsParams() error {
      // 验证 level 参数
      if logsLevel != "" {
          levels := strings.Split(logsLevel, ",")
          validLevels := map[string]bool{
              "debug": true,
              "info":  true,
              "warn":  true,
              "error": true,
          }
          
          for _, level := range levels {
              level = strings.TrimSpace(strings.ToLower(level))
              if !validLevels[level] {
                  return fmt.Errorf("invalid log level: %s (valid: debug, info, warn, error)", level)
              }
          }
      }
      
      return nil
  }
  ```

- [ ] 级别参数验证
- [ ] 查询参数构建
- [ ] 单元测试过滤逻辑

### Task 7: 错误处理和建议 (AC7)
- [ ] 实现错误格式化函数
  ```go
  // cmd/waterflow-cli/cmd/logs.go
  
  import (
      "github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/client"
  )
  
  // formatLogsError 格式化日志查询错误
  func formatLogsError(err error, format string) error {
      if format == "json" {
          return formatLogsErrorJSON(err)
      }
      
      if serverErr, ok := err.(*client.ServerError); ok {
          return formatServerError(serverErr)
      }
      
      // 其他错误
      fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
      return &ExitError{Code: 1}
  }
  
  func formatLogsErrorJSON(err error) error {
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
  
  // displayNoLogsMessage 显示无日志消息
  func displayNoLogsMessage() error {
      fmt.Println("No logs found matching filter criteria")
      
      if logsLevel != "" || logsJob != "" || logsStep != "" {
          fmt.Println("\nFilters applied:")
          if logsLevel != "" {
              fmt.Printf("  Level: %s\n", logsLevel)
          }
          if logsJob != "" {
              fmt.Printf("  Job: %s\n", logsJob)
          }
          if logsStep != "" {
              fmt.Printf("  Step: %s\n", logsStep)
          }
          fmt.Println("\nSuggestion: Try removing filters or check workflow status")
      }
      
      return nil
  }
  ```

- [ ] 格式化 Server 错误
- [ ] 无日志提示
- [ ] 提供针对性建议
- [ ] 测试错误场景

### Task 8: 集成测试 (AC1-AC7)
- [ ] 创建测试脚本 `cmd/waterflow-cli/integration_logs_test.sh`
  ```bash
  #!/bin/bash
  # CLI logs 命令集成测试
  
  set -e
  
  CLI="./bin/waterflow"
  SERVER_URL="http://localhost:8088"
  
  # 检查 Server 是否运行
  if ! curl -sf "$SERVER_URL/health" > /dev/null; then
      echo "ERROR: Waterflow server not running at $SERVER_URL"
      exit 1
  fi
  
  echo "=== Test 1: Submit workflow first ==="
  WORKFLOW_ID=$($CLI submit --quiet testdata/valid/multi-step.yaml)
  echo "Submitted workflow: $WORKFLOW_ID"
  
  # 等待工作流执行生成日志
  sleep 5
  echo
  
  echo "=== Test 2: Basic logs query (AC1) ==="
  $CLI logs $WORKFLOW_ID | grep -q "Workflow started"
  echo "PASS"
  echo
  
  echo "=== Test 3: Level filter (AC2) ==="
  $CLI logs --level info $WORKFLOW_ID > /dev/null
  echo "PASS"
  echo
  
  echo "=== Test 4: Job filter (AC3) ==="
  $CLI logs --job build $WORKFLOW_ID > /dev/null
  echo "PASS"
  echo
  
  echo "=== Test 5: JSON output (AC6) ==="
  OUTPUT=$($CLI logs --format json $WORKFLOW_ID | head -1)
  echo "$OUTPUT" | jq -e '.timestamp' > /dev/null
  echo "$OUTPUT" | jq -e '.level' > /dev/null
  echo "$OUTPUT" | jq -e '.message' > /dev/null
  echo "PASS"
  echo
  
  echo "=== Test 6: Tail limit (AC1) ==="
  LINES=$($CLI logs --tail 10 $WORKFLOW_ID | wc -l)
  if [ $LINES -gt 10 ]; then
      echo "FAIL: Expected <= 10 lines, got $LINES"
      exit 1
  fi
  echo "PASS"
  echo
  
  echo "=== Test 7: Workflow not found (AC7) ==="
  $CLI logs nonexistent-id 2>&1 | grep -q "not found"
  if [ $? -ne 0 ]; then
      echo "FAIL: Should report not found error"
      exit 1
  fi
  echo "PASS"
  echo
  
  echo "=== Test 8: Follow mode (AC4) - manual test ==="
  echo "Run manually: $CLI logs --follow $WORKFLOW_ID"
  echo "SKIP (requires manual verification)"
  echo
  
  echo "All automated tests passed!"
  ```

- [ ] 测试基础日志查询
- [ ] 测试过滤功能
- [ ] 测试输出格式
- [ ] 测试错误场景
- [ ] 手动测试 --follow 模式

### Task 9: 文档更新
- [ ] 更新 `cmd/waterflow-cli/README.md`
  ```markdown
  ### logs
  
  View execution logs from a workflow.
  
  **Usage:**
  ```bash
  waterflow logs [flags] <workflow-id>
  ```
  
  **Flags:**
  - `--follow, -f` - Follow logs in real-time (like tail -f)
  - `--tail <n>` - Number of lines to show (1-1000, -1 for all, default: 100)
  - `--level <levels>` - Filter by log level (info,warn,error,debug)
  - `--job <name>` - Filter by job name
  - `--step <name>` - Filter by step name
  - `--format <format>` - Output format (text, json, compact)
  - `--no-color` - Disable colored output
  - `--no-timestamps` - Hide timestamps
  - `--poll-interval <duration>` - Polling interval for --follow (default: 2s)
  
  **Examples:**
  ```bash
  # View logs
  waterflow logs 550e8400-e29b-41d4-a716-446655440000
  
  # Follow logs in real-time
  waterflow logs --follow <workflow-id>
  
  # Show only errors
  waterflow logs --level error <workflow-id>
  
  # Show errors and warnings
  waterflow logs --level error,warn <workflow-id>
  
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
  
  # JSON output
  waterflow logs --format json <workflow-id>
  
  # Compact format (no timestamps/levels)
  waterflow logs --format compact <workflow-id>
  
  # No timestamps
  waterflow logs --no-timestamps <workflow-id>
  
  # Disable colors
  waterflow logs --no-color <workflow-id>
  NO_COLOR=1 waterflow logs <workflow-id>
  ```
  
  **Exit Codes:**
  - `0` - Logs query successful (or workflow completed successfully with --follow)
  - `1` - Query failed, workflow not found, or workflow failed (with --follow)
  - `2` - Usage error (invalid flags or arguments)
  ```

- [ ] 添加使用示例到 `examples/cli/`
- [ ] 更新主项目 README

### Task 10: 性能优化建议
- [ ] 文档化性能考虑
  ```markdown
  ## Performance Considerations
  
  **Polling Interval:**
  - Default: 2 seconds (--follow mode)
  - Minimum: 1 second
  - Recommended: 2-5 seconds for balance between real-time and server load
  
  **Tail Limit:**
  - Default: 100 lines
  - Maximum: 1000 lines (server enforced)
  - Use -1 for all logs (may be slow for long-running workflows)
  
  **Filtering:**
  - Client-side deduplication for --follow mode
  - Server-side filtering for level/job/step (more efficient)
  ```

- [ ] 性能测试场景
- [ ] 优化建议文档

## Dev Notes

### 架构设计要点

**1. 命令参数组合:**
```go
// 参数组合
--follow          → 实时跟踪
--follow + --poll-interval → 自定义轮询间隔
--tail N          → 限制显示行数
--level error     → 仅错误日志
--job deploy      → 仅 deploy Job 日志
--step checkout   → 仅 checkout Step 日志
--no-color        → 禁用 ANSI 颜色
--no-timestamps   → 隐藏时间戳
```

**2. 工作流程:**
```
1. 验证参数 (tail, level)
2. 构建查询参数 (LogsQuery)
3. 查询日志 (GET /v1/workflows/{id}/logs)
4. 格式化输出 (text/json/compact)
5. (可选) 持续跟踪 (--follow)
```

**3. 日志级别颜色 (与 Story 5.4 保持一致):**
```go
// 级别 → 颜色
"debug" → 灰色/白色
"info"  → 蓝色
"warn"  → 黄色
"error" → 红色
```

**4. --follow 实现策略:**
```go
// SSE 优先 (Story 1.9 支持)
try {
    StreamWorkflowLogs(stream=true) → SSE 实时流
} catch {
    fallback → 轮询模式 (2s 间隔)
}

// 去重方案 (轮询模式):
// API 不支持 since/offset,使用计数器
lastSeenCount = 0
while running {
    logs = GetWorkflowLogs(tail=1000)
    newLogs = logs[lastSeenCount:]
    display(newLogs)
    lastSeenCount = len(logs)
}
```

**5. 输出模式:**
- **text** (默认) - 人类可读,带时间戳、级别、颜色
- **json** - 每行一个 JSON 对象,机器可读
- **compact** - 仅消息和 Job/Step,无时间戳/级别

### 技术约束

**复用 Story 5.1/5.4:**
- ✅ HTTP 客户端 (`pkg/client/client.go`)
- ✅ 配置管理 (`pkg/config/config.go`)
- ✅ 颜色支持 (`github.com/fatih/color`)
- ✅ 终端检测 (Story 5.4)

**新增功能:**
- JSON Lines 解析
- 日志去重 (--follow 模式)
- 级别/Job/Step 过滤
- 时间戳格式化

**依赖 Server API (Story 1.9):**
- `GET /v1/workflows/{id}/logs` - 查询日志
  - 查询参数: `level`, `job`, `step`, `tail`
  - 响应格式: JSON Array `[{...}, {...}]`
- `GET /v1/workflows/{id}/logs?stream=true` - SSE 实时流
  - 响应格式: Server-Sent Events (text/event-stream)
  - 每个事件格式: `data: {...}\n\n`
- 注意: **不支持** `since` 或 `offset` 参数 (需客户端去重)

**外部依赖:**
- `github.com/fatih/color` - 终端颜色支持 (已有)

### 性能考虑

**轮询间隔:**
- 默认: 2 秒 (--follow)
- 最小: 1 秒
- 推荐: 2-5 秒 (平衡实时性和服务器负载)

**Tail 限制:**
- 默认: 100 行
- 最大: 1000 行 (服务器强制)
- -1 表示所有日志 (可能很慢)

**去重策略:**
- **SSE 模式**: 无需去重,Server 推送新日志
- **轮询模式**: 使用计数器去重 (lastSeenCount)
  - 记录已显示日志数量
  - 仅显示 logs[lastSeenCount:] 新日志
  - 不使用时间戳比较 (可能有重复的时间戳)

**SSE vs 轮询:**
- SSE: 实时推送,延迟 <100ms,无重复请求
- 轮询: 2-5s 延迟,重复请求,需客户端去重

### 集成点

**与现有系统集成:**

1. **Server REST API (Story 1.9)**
   - 端点: `GET /v1/workflows/{id}/logs`
   - 响应: JSON Array 格式 `[{...}]`
   - 查询参数: level, job, step, tail
   - SSE 流: `?stream=true` + `Accept: text/event-stream`

2. **CLI 基础 (Story 5.1)**
   - HTTP 客户端
   - 配置加载
   - 输出格式化

3. **status 命令 (Story 5.4)**
   - 复用颜色逻辑
   - 复用终端检测
   - 复用状态判断

### 测试策略

**单元测试:**
- 日志格式化
- 时间戳解析
- 级别颜色映射
- 参数验证

**集成测试 (需要 Server):**
- 基础日志查询
- 过滤功能验证
- 输出格式验证
- 错误场景测试

**手动测试:**
- --follow 实时跟踪
- 颜色显示效果
- Ctrl+C 中断处理
- 长日志性能测试

### 后续 Story 依赖

**Story 5.6 (node list 命令):**
- 不依赖本 Story

**Story 5.7 (Go SDK):**
- 可能需要类似的 GetWorkflowLogs() 方法

### 参考文档

**内部文档:**
- [Story 5.1 - CLI 基础框架](./5-1-cli-framework.md)
- [Story 5.4 - status 命令](./5-4-cli-status-command.md)
- [Story 1.9 - REST API](./1-9-workflow-management-api.md)

**外部参考:**
- kubectl logs: https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#logs
- docker logs: https://docs.docker.com/engine/reference/commandline/logs/
- stern (多 pod 日志): https://github.com/stern/stern

## Definition of Done

### 代码完成标准

- [ ] 所有 Task 完成并测试通过
- [ ] 单元测试覆盖率 >80%
- [ ] 代码通过 `golangci-lint` 检查
- [ ] 无编译警告和错误

### 功能验证标准

- [ ] AC1: 基础日志查询
  - [ ] `waterflow logs <id>` 查询成功
  - [ ] 显示时间戳、级别、Job、Step、消息
  - [ ] --tail 参数限制行数
- [ ] AC2: 日志级别过滤
  - [ ] `--level error` 仅显示错误
  - [ ] 支持多级别 (逗号分隔)
- [ ] AC3: Job/Step 过滤
  - [ ] `--job deploy` 过滤 Job
  - [ ] `--step checkout` 过滤 Step
  - [ ] 组合过滤正常工作
- [ ] AC4: 实时日志跟踪
  - [ ] `--follow` 持续显示新日志
  - [ ] 工作流完成后退出
  - [ ] Ctrl+C 正常退出
- [ ] AC5: 彩色日志输出
  - [ ] 根据级别显示颜色
  - [ ] --no-color 禁用颜色
  - [ ] NO_COLOR 环境变量支持
- [ ] AC6: 格式化输出
  - [ ] text 格式 (默认)
  - [ ] json 格式
  - [ ] compact 格式
  - [ ] --no-timestamps 隐藏时间戳
- [ ] AC7: 友好错误处理
  - [ ] 工作流不存在提示
  - [ ] Server 连接失败提示
  - [ ] 每种错误有建议

### 测试验证标准

- [ ] 所有单元测试通过
- [ ] 集成测试脚本通过 (需要 Server)
- [ ] 手动测试所有 AC
- [ ] 与 Server 端到端测试

### 文档完成标准

- [ ] README 包含 logs 命令文档
- [ ] `--help` 输出清晰完整
- [ ] 使用示例完整
- [ ] 错误信息文档完整

### 交付标准

- [ ] logs 命令可执行
- [ ] 基础查询正常工作
- [ ] 过滤功能正常工作
- [ ] 实时跟踪功能正常
- [ ] 所有输出格式正常工作
- [ ] 颜色显示正常工作
- [ ] 代码已合并到主分支
- [ ] Sprint status 更新为 `done`

## References

### 源文档
- [Source: docs/epics.md#Story 5.5](../epics.md) - Epic 分解中的 Story 5.5
- [Source: docs/prd.md#Epic 5](../prd.md) - PRD Epic 5 定义

### 前置 Stories
- [Source: docs/sprint-artifacts/5-1-cli-framework.md](./5-1-cli-framework.md) - CLI 基础框架
- [Source: docs/sprint-artifacts/5-4-cli-status-command.md](./5-4-cli-status-command.md) - status 命令
- [Source: docs/sprint-artifacts/1-9-workflow-management-api.md](./1-9-workflow-management-api.md) - REST API
- [Source: docs/sprint-artifacts/1-8-temporal-sdk-integration.md](./1-8-temporal-sdk-integration.md) - Temporal SDK

### 代码参考
- [Source: internal/api/workflow_handler.go](../../internal/api/workflow_handler.go) - GetWorkflowLogs API
- [Source: pkg/temporal/history_parser.go](../../pkg/temporal/history_parser.go) - Event History 解析

### 外部参考
- kubectl logs: https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#logs
- docker logs: https://docs.docker.com/engine/reference/commandline/logs/
- stern: https://github.com/stern/stern

## Dev Agent Record

### Agent Model Used

待 Dev Agent 执行时填写

### Completion Notes List

待 Dev Agent 执行时填写

### File List

待 Dev Agent 执行时填写
