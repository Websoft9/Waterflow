# Story 1.6: 基础工作流执行引擎

Status: ready-for-dev

## Story

As a **系统**,  
I want **将解析的 YAML 工作流转换为 Temporal Workflow 执行**,  
So that **工作流可以持久化运行**。

## Acceptance Criteria

**Given** 工作流已通过 API 提交  
**When** 工作流开始执行  
**Then** 创建 Temporal Workflow 实例  
**And** 工作流状态 100% 持久化到 Temporal Event History (Event Sourcing 架构)  
**And** 支持单 Job、单 Step 的简单工作流  
**And** 每个 Step = 1 个 Activity 调用 (单节点执行模式, ADR-0002)  
**And** Step 执行结果记录到 Temporal Event History  
**And** Server/Worker 崩溃后可从 Event History 完全恢复执行  
**And** 工作流执行失败时状态正确记录  
**And** 单个 Step 执行启动延迟 < 100ms (Activity调度性能)

## Technical Context

### Architecture Constraints

根据 [docs/architecture.md](docs/architecture.md) §3.2 Agent内部组件设计:

1. **核心职责**
   - 定义Temporal Workflow函数
   - 将YAML工作流定义转换为Temporal Workflow执行
   - 编排Job和Step的执行顺序
   - 调用Activity执行具体Step (ADR-0002单节点模式)
   - 处理执行错误和状态持久化

2. **执行流程**

```
Story 1.5提交 → Temporal启动Workflow → Story 1.6执行引擎
                                          ↓
                            遍历Jobs → 遍历Steps → ExecuteActivity
                                                      ↓
                                                   (Story 2.x Agent执行)
```

3. **关键设计约束**
   
   **Event Sourcing 架构** (参考 ADR-0001):
   - 所有工作流状态存储在 Temporal Event History
   - Server 完全无状态,状态 100% 持久化
   - 进程崩溃后从 Event History 自动恢复
   - 支持时间旅行查询和完整审计日志
   
   **单节点执行模式** (参考 ADR-0002):
   - **每个 Step = 1 个 Activity 调用**,精确控制粒度
   - **细粒度超时**: 每个 Step 独立配置 timeout-minutes
   - **独立重试**: 每个 Step 可配置不同的重试策略
   - **精确故障定位**: Temporal UI 清晰展示每个 Step 的执行历史
   - **MVP简化**: 仅支持单 Job、串行执行 Steps

### Dependencies

**前置Story:**
- ✅ Story 1.1: Waterflow Server框架搭建
- ✅ Story 1.2: REST API服务框架
- ✅ Story 1.3: YAML DSL解析器
  - 使用: `WorkflowDefinition`, `Job`, `Step` 数据结构
- ✅ Story 1.4: Temporal SDK集成
  - 使用: Temporal Workflow和Activity API
- ✅ Story 1.5: 工作流提交API
  - 使用: 提交时传递的`WorkflowDefinition`

**后续Story依赖本Story:**
- Story 1.7: 状态查询API - 查询Workflow执行状态
- Story 2.x: Agent系统 - 实现真实的Activity执行逻辑
- Story 3.x: 并行执行 - 扩展支持多Job并行

**外部依赖:**
- Temporal Server (Worker注册和执行)

### Technology Stack

**Temporal Workflow SDK:**

```go
import (
    "go.temporal.io/sdk/workflow"
    "go.temporal.io/sdk/worker"
)

// Workflow函数签名
func WaterflowWorkflow(ctx workflow.Context, def *parser.WorkflowDefinition) error {
    // Workflow代码必须是确定性的 (Deterministic)
    // 不能使用: time.Now(), random, goroutines
    // 必须使用: workflow.Now(), workflow.Go()
}
```

**核心API:**

1. **workflow.ExecuteActivity** - 调用Activity
   ```go
   ctx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
       StartToCloseTimeout: 10 * time.Minute,
   })
   
   var result string
   err := workflow.ExecuteActivity(ctx, "ExecuteStepActivity", step).Get(ctx, &result)
   ```

2. **workflow.GetLogger** - 日志记录
   ```go
   logger := workflow.GetLogger(ctx)
   logger.Info("Starting job", "job_name", job.Name)
   ```

3. **workflow.Sleep** - 等待
   ```go
   workflow.Sleep(ctx, 5*time.Second)
   ```

**Worker注册:**

```go
// Waterflow Server需要启动Worker监听Task Queue
worker := worker.New(temporalClient, "default", worker.Options{})

// 注册Workflow
worker.RegisterWorkflow(WaterflowWorkflow)

// 注册Activity (MVP暂时使用Mock)
worker.RegisterActivity(ExecuteStepActivity)

// 启动Worker
worker.Run(worker.InterruptCh())
```

### Temporal Workflow 确定性要求

**关键约束 (Temporal强制要求):**

1. **禁止操作**
   - ❌ `time.Now()` → 使用 `workflow.Now(ctx)`
   - ❌ `rand.Intn()` → 使用 `workflow.SideEffect()`
   - ❌ `go func()` → 使用 `workflow.Go()`
   - ❌ 直接IO操作 → 必须通过Activity

2. **原因**: Workflow重放机制
   - Temporal通过重放Event History恢复状态
   - 非确定性代码导致重放结果不一致
   - Worker崩溃后无法恢复执行

**示例:**

```go
// ❌ 错误: 非确定性
func BadWorkflow(ctx workflow.Context) error {
    start := time.Now() // 重放时每次不同!
    time.Sleep(1 * time.Second) // 不会真正等待
    return nil
}

// ✅ 正确: 确定性
func GoodWorkflow(ctx workflow.Context) error {
    start := workflow.Now(ctx) // 重放时保持一致
    workflow.Sleep(ctx, 1*time.Second) // 使用Temporal Timer
    return nil
}
```

### MVP Scope Definition

**支持的功能:**

✅ 单个Job执行  
✅ 串行Steps执行  
✅ Step超时控制 (timeout-minutes)  
✅ 基本错误处理  
✅ 执行状态持久化  

**不支持 (后续Story):**

❌ 多Job并行 (Story 3.x)  
❌ Step重试策略 (Story 4.x)  
❌ 条件执行 (if表达式, Story 5.x)  
❌ outputs传递 (Story 5.x)  
❌ 真实Agent执行 (Story 2.x - MVP使用Mock Activity)  

**MVP工作流示例:**

```yaml
name: Simple Build
on: push
jobs:
  build:
    runs-on: linux-amd64
    timeout-minutes: 30
    steps:
      - name: Checkout
        uses: checkout@v1
        timeout-minutes: 5
      
      - name: Build
        uses: run@v1
        with:
          command: echo "Building..."
        timeout-minutes: 10
```

### Project Structure Updates

基于Story 1.1-1.5的结构,本Story新增:

```
internal/
├── workflow/
│   ├── waterflow_workflow.go      # Temporal Workflow实现 (新建)
│   ├── activities.go              # Activity定义 (新建)
│   ├── worker.go                  # Worker管理 (新建)
│   ├── waterflow_workflow_test.go # Workflow测试 (新建)
│   └── activities_test.go         (新建)
├── server/
│   └── server.go                  # 修改 - 启动Worker

cmd/server/
└── main.go                        # 修改 - 初始化Worker
```

## Tasks / Subtasks

### Task 0: 验证依赖 (AC: 开发环境就绪)

- [ ] 0.1 确认Temporal SDK已安装 (Story 1.4)
  ```bash
  # 验证依赖
  go list -m go.temporal.io/sdk
  # 期望输出: go.temporal.io/sdk v1.25.0
  ```

- [ ] 0.2 确认Temporal Server运行中
  ```bash
  # 方法1: 健康检查
  curl http://localhost:7233/health
  
  # 方法2: Docker Compose状态
  cd deployments
  docker-compose ps temporal
  # 期望: State=Up
  ```

- [ ] 0.3 验证前置Story (1.1-1.5) 产出文件存在
  ```bash
  #!/bin/bash
  # test/verify-dependencies.sh
  
  echo "=== Verifying Story 1.1-1.5 Dependencies ==="
  
  # Story 1.1-1.2: Server框架和REST API
  test -f cmd/server/main.go || { echo "ERROR: Story 1.1未完成 - 缺少cmd/server/main.go"; exit 1; }
  test -f internal/server/server.go || { echo "ERROR: Story 1.2未完成 - 缺少internal/server/server.go"; exit 1; }
  
  # Story 1.3: YAML解析器
  test -f internal/parser/parser.go || { echo "ERROR: Story 1.3未完成 - 缺少internal/parser/parser.go"; exit 1; }
  go list -m gopkg.in/yaml.v3 > /dev/null 2>&1 || { echo "ERROR: Story 1.3未完成 - YAML库未安装"; exit 1; }
  
  # Story 1.4: Temporal集成
  test -f internal/temporal/client.go || { echo "ERROR: Story 1.4未完成 - 缺少internal/temporal/client.go"; exit 1; }
  go list -m go.temporal.io/sdk > /dev/null 2>&1 || { echo "ERROR: Story 1.4未完成 - Temporal SDK未安装"; exit 1; }
  
  # Story 1.5: 工作流提交API
  test -f internal/service/workflow_service.go || { echo "ERROR: Story 1.5未完成 - 缺少workflow_service.go"; exit 1; }
  test -f internal/models/request.go || { echo "ERROR: Story 1.5未完成 - 缺少models/request.go"; exit 1; }
  
  echo "✅ All dependencies verified - Story 1.6 can proceed"
  ```

- [ ] 0.4 说明本Story无新增依赖
  ```markdown
  本Story复用Story 1.4安装的Temporal SDK,无需安装新依赖。
  主要开发工作: 实现Workflow函数、Activity定义、Worker管理。
  
  **开始开发前务必运行:**
  ```bash
  chmod +x test/verify-dependencies.sh
  ./test/verify-dependencies.sh
  ```
  
  如果验证失败,请先完成对应的前置Story。
  ```

### Task 1: 定义Temporal Workflow函数 (AC: 创建Temporal Workflow实例)

- [ ] 1.1 创建`internal/workflow/waterflow_workflow.go`
  ```go
  package workflow
  
  import (
      "fmt"
      "time"
      
      "go.temporal.io/sdk/workflow"
      
      "waterflow/internal/parser"
  )
  
  // WaterflowWorkflow 主工作流函数
  func WaterflowWorkflow(ctx workflow.Context, def *parser.WorkflowDefinition) error {
      logger := workflow.GetLogger(ctx)
      
      logger.Info("Workflow started",
          "name", def.Name,
          "job_count", len(def.Jobs),
      )
      
      // MVP: 仅支持单个Job
      if len(def.Jobs) == 0 {
          return fmt.Errorf("workflow must have at least one job")
      }
      
      if len(def.Jobs) > 1 {
          return fmt.Errorf("MVP only supports single job (found %d jobs)", len(def.Jobs))
      }
      
      // 获取第一个Job (Go map遍历顺序不确定,但MVP只有一个)
      var job parser.Job
      var jobName string
      for name, j := range def.Jobs {
          jobName = name
          job = j
          break
      }
      
      logger.Info("Starting job", "job_name", jobName)
      
      // 执行Job
      err := executeJob(ctx, jobName, job)
      if err != nil {
          logger.Error("Job failed", "job_name", jobName, "error", err)
          return fmt.Errorf("job %s failed: %w", jobName, err)
      }
      
      logger.Info("Workflow completed successfully")
      return nil
  }
  
  // executeJob 执行单个Job
  func executeJob(ctx workflow.Context, jobName string, job parser.Job) error {
      logger := workflow.GetLogger(ctx)
      
      // Job级别超时 (如果配置)
      if job.TimeoutMinutes > 0 {
          var cancel workflow.CancelFunc
          ctx, cancel = workflow.WithCancel(ctx)
          defer cancel()
          
          workflow.Go(ctx, func(ctx workflow.Context) {
              workflow.Sleep(ctx, time.Duration(job.TimeoutMinutes)*time.Minute)
              cancel()
          })
      }
      
      // 串行执行Steps
      for i, step := range job.Steps {
          logger.Info("Starting step",
              "job", jobName,
              "step_index", i,
              "step_name", step.Name,
          )
          
          err := executeStep(ctx, step)
          if err != nil {
              logger.Error("Step failed",
                  "step_name", step.Name,
                  "error", err,
              )
              return fmt.Errorf("step %s failed: %w", step.Name, err)
          }
          
          logger.Info("Step completed", "step_name", step.Name)
      }
      
      return nil
  }
  ```

- [ ] 1.2 实现executeStep函数 (调用Activity)
  ```go
  // executeStep 执行单个Step (调用Activity)
  func executeStep(ctx workflow.Context, step parser.Step) error {
      // Step超时配置
      timeout := 10 * time.Minute // 默认10分钟
      if step.TimeoutMinutes > 0 {
          timeout = time.Duration(step.TimeoutMinutes) * time.Minute
      }
      
      // 配置Activity选项
      activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
          StartToCloseTimeout: timeout,
          HeartbeatTimeout:    30 * time.Second, // 心跳超时
          RetryPolicy: &temporal.RetryPolicy{
              MaximumAttempts:    1, // MVP暂不重试
              InitialInterval:    time.Second,
              BackoffCoefficient: 2.0,
          },
      })
      
      // 准备Activity输入
      input := ExecuteStepInput{
          Name: step.Name,
          Uses: step.Uses,
          With: step.With,
      }
      
      // 调用Activity
      var result ExecuteStepResult
      err := workflow.ExecuteActivity(activityCtx, "ExecuteStepActivity", input).Get(activityCtx, &result)
      
      if err != nil {
          return err
      }
      
      // 记录输出 (MVP暂不处理,Story 5.x实现)
      workflow.GetLogger(ctx).Info("Step output", "output", result.Output)
      
      return nil
  }
  ```

### Task 2: 定义Activity接口和Mock实现 (AC: Step执行结果记录)

- [ ] 2.1 创建`internal/workflow/activities.go`
  ```go
  package workflow
  
  import (
      "context"
      "fmt"
      "time"
      
      "go.temporal.io/sdk/activity"
  )
  
  // ExecuteStepInput Activity输入参数
  type ExecuteStepInput struct {
      Name string                    `json:"name"`
      Uses string                    `json:"uses"`
      With map[string]interface{}    `json:"with"`
  }
  
  // ExecuteStepResult Activity返回结果
  type ExecuteStepResult struct {
      Output   string            `json:"output"`
      ExitCode int               `json:"exit_code"`
      Duration time.Duration     `json:"duration"`
  }
  
  // ExecuteStepActivity 执行单个Step (MVP Mock实现)
  // TODO: Story 2.x 将实现真实的Agent调用逻辑 (run@v1, checkout@v1等)
  // 当前仅用于验证Workflow编排和Event Sourcing架构
  func ExecuteStepActivity(ctx context.Context, input ExecuteStepInput) (ExecuteStepResult, error) {
      logger := activity.GetLogger(ctx)
      
      logger.Info("[MOCK] Executing step",
          "name", input.Name,
          "uses", input.Uses,
      )
      
      // MVP: Mock执行,返回成功
      // Story 2.x 将实现真实的Agent调用
      
      // 模拟执行时间
      time.Sleep(1 * time.Second)
      
      // 根据节点类型模拟输出
      output := fmt.Sprintf("[MOCK] Executed %s with args: %v", input.Uses, input.With)
      
      result := ExecuteStepResult{
          Output:   output,
          ExitCode: 0,
          Duration: 1 * time.Second,
      }
      
      logger.Info("Step executed successfully",
          "name", input.Name,
          "exit_code", result.ExitCode,
      )
      
      return result, nil
  }
  ```

- [ ] 2.2 添加增强的心跳支持 (支持取消检测和进度报告)
  ```go
  func ExecuteStepActivity(ctx context.Context, input ExecuteStepInput) (ExecuteStepResult, error) {
      logger := activity.GetLogger(ctx)
      
      // 检查Activity是否在启动前就被取消
      if err := ctx.Err(); err != nil {
          logger.Warn("Activity cancelled before execution", zap.Error(err))
          return ExecuteStepResult{}, err
      }
      
      logger.Info("[MOCK] Executing step",
          "name", input.Name,
          "uses", input.Uses,
      )
      
      // 心跳Ticker (每10秒发送一次)
      heartbeatTicker := time.NewTicker(10 * time.Second)
      defer heartbeatTicker.Stop()
      
      done := make(chan ExecuteStepResult)
      errCh := make(chan error)
      
      // 异步执行 (模拟长时间运行的任务)
      go func() {
          // MVP: Mock执行,模拟10秒的工作
          for i := 0; i < 10; i++ {
              time.Sleep(1 * time.Second)
              
              // 定期检查取消信号
              select {
              case <-ctx.Done():
                  logger.Warn("Execution cancelled mid-flight")
                  errCh <- ctx.Err()
                  return
              default:
                  // 继续执行
              }
          }
          
          // 完成执行
          done <- ExecuteStepResult{
              Output:   fmt.Sprintf("[MOCK] Executed %s with args: %v", input.Uses, input.With),
              ExitCode: 0,
              Duration: 10 * time.Second,
          }
      }()
      
      // 心跳循环
      for {
          select {
          case <-ctx.Done():
              logger.Warn("Activity cancelled during execution")
              return ExecuteStepResult{}, ctx.Err()
          
          case <-heartbeatTicker.C:
              // 发送心跳并报告进度
              activity.RecordHeartbeat(ctx, fmt.Sprintf("executing: %s", input.Name))
              logger.Debug("Heartbeat sent", zap.String("step", input.Name))
          
          case result := <-done:
              logger.Info("Step executed successfully",
                  "name", input.Name,
                  "exit_code", result.ExitCode,
                  "duration", result.Duration,
              )
              return result, nil
          
          case err := <-errCh:
              logger.Error("Step execution failed", zap.Error(err))
              return ExecuteStepResult{}, err
          }
      }
  }
  ```

### Task 3: 实现Worker管理 (AC: 工作流状态持久化到Temporal)

- [ ] 3.1 创建`internal/workflow/worker.go`
  ```go
  package workflow
  
  import (
      "go.temporal.io/sdk/client"
      "go.temporal.io/sdk/worker"
      "go.uber.org/zap"
  )
  
  // WorkerManager 管理Temporal Worker
  type WorkerManager struct {
      client client.Client
      worker worker.Worker
      logger *zap.Logger
  }
  
  // NewWorkerManager 创建Worker管理器
  func NewWorkerManager(c client.Client, taskQueue string, logger *zap.Logger) *WorkerManager {
      // 创建Worker
      w := worker.New(c, taskQueue, worker.Options{
          MaxConcurrentActivityExecutionSize: 10,
          MaxConcurrentWorkflowTaskExecutionSize: 10,
      })
      
      // 注册Workflow
      w.RegisterWorkflow(WaterflowWorkflow)
      
      // 注册Activity
      w.RegisterActivity(ExecuteStepActivity)
      
      logger.Info("Worker registered",
          "task_queue", taskQueue,
          "workflows", "WaterflowWorkflow",
          "activities", "ExecuteStepActivity",
      )
      
      return &WorkerManager{
          client: c,
          worker: w,
          logger: logger,
      }
  }
  
  // Start 启动Worker
  func (wm *WorkerManager) Start() error {
      wm.logger.Info("Starting Temporal Worker...")
      
      // Run会阻塞直到Stop被调用
      err := wm.worker.Run(worker.InterruptCh())
      if err != nil {
          wm.logger.Error("Worker failed", zap.Error(err))
          return err
      }
      
      return nil
  }
  
  // Stop 优雅停止Worker (等待正在执行的Activity完成)
  func (wm *WorkerManager) Stop() {
      wm.logger.Info("Stopping Temporal Worker gracefully...")
      
      // Worker.Stop()会阻塞直到所有Activity完成
      // 添加超时保护避免无限等待
      done := make(chan struct{})
      
      go func() {
          wm.worker.Stop() // 阻塞直到Activity完成
          close(done)
      }()
      
      // 等待最多30秒
      select {
      case <-done:
          wm.logger.Info("Worker stopped gracefully - all activities completed")
      case <-time.After(30 * time.Second):
          wm.logger.Warn("Worker stop timeout - forcing shutdown after 30s")
      }
  }
  ```

- [ ] 3.2 集成到Server
  ```go
  // internal/server/server.go
  
  type Server struct {
      config          *config.Config
      logger          *zap.Logger
      router          *gin.Engine
      httpServer      *http.Server
      temporalClient  *temporal.Client
      workflowService *service.WorkflowService
      workerManager   *workflow.WorkerManager  // 新增
  }
  
  func New(cfg *config.Config, logger *zap.Logger, tc *temporal.Client) *Server {
      // ... 现有代码 ...
      
      // 创建WorkerManager
      workerManager := workflow.NewWorkerManager(
          tc.GetClient(),
          "default", // Task Queue名称
          logger,
      )
      
      return &Server{
          config:          cfg,
          logger:          logger,
          temporalClient:  tc,
          workflowService: workflowService,
          workerManager:   workerManager,
          router:          router,
          httpServer:      &http.Server{...},
      }
  }
  
  func (s *Server) Start() error {
      // Worker启动重试配置
      const (
          maxRetries = 5
          retryDelay = 5 * time.Second
      )
      
      // 创建错误通道用于Worker启动检测
      errChan := make(chan error, 1)
      
      // 启动Worker (带重试机制)
      go func() {
          for attempt := 1; attempt <= maxRetries; attempt++ {
              s.logger.Info("Starting Temporal Worker",
                  zap.Int("attempt", attempt),
                  zap.Int("max_retries", maxRetries),
              )
              
              err := s.workerManager.Start()
              if err == nil {
                  return // 启动成功
              }
              
              s.logger.Warn("Worker failed to start",
                  zap.Int("attempt", attempt),
                  zap.Error(err),
              )
              
              // 检查是否为连接错误 (可重试) vs 配置错误 (不可重试)
              if isConnectionError(err) && attempt < maxRetries {
                  s.logger.Info("Connection error detected - retrying",
                      zap.Duration("retry_delay", retryDelay),
                  )
                  time.Sleep(retryDelay)
                  continue
              }
              
              // 最后一次尝试或不可重试的错误
              errChan <- fmt.Errorf("worker failed after %d attempts: %w", attempt, err)
              return
          }
      }()
      
      // 等待Worker启动或失败 (最多等待30秒)
      select {
      case err := <-errChan:
          return err
      case <-time.After(30 * time.Second):
          s.logger.Info("Worker started successfully")
      }
      
      // 启动HTTP Server
      s.logger.Info("Starting HTTP server", zap.String("addr", s.httpServer.Addr))
      return s.httpServer.ListenAndServe()
  }
  
  // isConnectionError 判断是否为连接错误 (可重试)
  func isConnectionError(err error) bool {
      if err == nil {
          return false
      }
      
      errMsg := err.Error()
      // Temporal连接错误通常包含这些关键词
      connectionErrors := []string{
          "connection refused",
          "no such host",
          "timeout",
          "temporary failure",
      }
      
      for _, keyword := range connectionErrors {
          if strings.Contains(strings.ToLower(errMsg), keyword) {
              return true
          }
      }
      
      return false
  }
  
  func (s *Server) Shutdown(ctx context.Context) error {
      // 停止Worker
      s.workerManager.Stop()
      
      // 停止HTTP Server
      return s.httpServer.Shutdown(ctx)
  }
  ```

### Task 4: 更新工作流提交逻辑 (AC: 支持单Job、单Step的简单工作流)

- [ ] 4.1 更新`internal/service/workflow_service.go`
  ```go
  func (ws *WorkflowService) SubmitWorkflow(ctx context.Context, yamlContent string) (*models.SubmitWorkflowResponse, error) {
      // 1. 解析YAML
      wf, err := ws.parser.Parse(yamlContent)
      if err != nil {
          return nil, fmt.Errorf("parse error: %w", err)
      }
      
      // 2. MVP验证: 单Job
      if len(wf.Jobs) == 0 {
          return nil, fmt.Errorf("workflow must have at least one job")
      }
      if len(wf.Jobs) > 1 {
          return nil, fmt.Errorf("MVP only supports single job workflow")
      }
      
      // 3. 生成WorkflowID
      workflowID := ws.GenerateWorkflowID()
      
      // 4. 提交到Temporal
      workflowOptions := client.StartWorkflowOptions{
          ID:                 workflowID,
          TaskQueue:          "default",
          WorkflowRunTimeout: 1 * time.Hour,
      }
      
      // 调用WaterflowWorkflow (Story 1.6实现)
      we, err := ws.temporalClient.GetClient().ExecuteWorkflow(
          ctx,
          workflowOptions,
          "WaterflowWorkflow", // Workflow名称
          wf,                  // 传递WorkflowDefinition
      )
      if err != nil {
          ws.logger.Error("Failed to submit workflow to Temporal",
              zap.Error(err),
              zap.String("workflow_id", workflowID),
          )
          return nil, fmt.Errorf("temporal submission error: %w", err)
      }
      
      // 5. 返回响应
      response := &models.SubmitWorkflowResponse{
          WorkflowID:  we.GetID(),
          RunID:       we.GetRunID(),
          Status:      "running",
          SubmittedAt: time.Now(),
      }
      
      ws.logger.Info("Workflow submitted successfully",
          zap.String("workflow_id", response.WorkflowID),
          zap.String("run_id", response.RunID),
      )
      
      return response, nil
  }
  ```

### Task 5: 添加错误处理和状态记录 (AC: 工作流执行失败时状态正确记录)

- [ ] 5.1 在Workflow中处理错误
  ```go
  func executeStep(ctx workflow.Context, step parser.Step) error {
      // ... Activity调用 ...
      
      err := workflow.ExecuteActivity(activityCtx, "ExecuteStepActivity", input).Get(activityCtx, &result)
      
      if err != nil {
          // 记录失败原因
          workflow.GetLogger(ctx).Error("Step execution failed",
              "step_name", step.Name,
              "error", err,
          )
          
          // 检查错误类型
          if temporal.IsApplicationError(err) {
              // 应用错误 (节点执行失败)
              return fmt.Errorf("application error in step %s: %w", step.Name, err)
          }
          
          if temporal.IsTimeoutError(err) {
              // 超时错误
              return fmt.Errorf("step %s timeout after %d minutes", step.Name, step.TimeoutMinutes)
          }
          
          // 其他错误
          return fmt.Errorf("step %s failed: %w", step.Name, err)
      }
      
      return nil
  }
  ```

- [ ] 5.2 在Activity中返回应用错误
  ```go
  func ExecuteStepActivity(ctx context.Context, input ExecuteStepInput) (ExecuteStepResult, error) {
      // ... 执行逻辑 ...
      
      // 模拟失败场景
      if input.Uses == "fail@v1" {
          return ExecuteStepResult{}, temporal.NewApplicationError(
              "Mock failure",
              "MockError",
              nil,
          )
      }
      
      // 正常返回
      return result, nil
  }
  ```

### Task 6: 添加Workflow单元测试 (代码质量保障)

- [ ] 6.1 创建`internal/workflow/waterflow_workflow_test.go`
  ```go
  package workflow
  
  import (
      "testing"
      "time"
      
      "github.com/stretchr/testify/assert"
      "github.com/stretchr/testify/mock"
      "go.temporal.io/sdk/testsuite"
      
      "waterflow/internal/parser"
  )
  
  func TestWaterflowWorkflow_Success(t *testing.T) {
      testSuite := &testsuite.WorkflowTestSuite{}
      env := testSuite.NewTestWorkflowEnvironment()
      
      // Mock Activity
      env.OnActivity("ExecuteStepActivity", mock.Anything, mock.Anything).Return(
          ExecuteStepResult{
              Output:   "success",
              ExitCode: 0,
              Duration: 1 * time.Second,
          }, nil,
      )
      
      // 准备测试数据
      def := &parser.WorkflowDefinition{
          Name: "Test Workflow",
          On:   "push",
          Jobs: map[string]parser.Job{
              "build": {
                  RunsOn: "linux",
                  Steps: []parser.Step{
                      {
                          Name: "Test Step",
                          Uses: "run@v1",
                          With: map[string]interface{}{
                              "command": "echo hello",
                          },
                      },
                  },
              },
          },
      }
      
      // 执行Workflow
      env.ExecuteWorkflow(WaterflowWorkflow, def)
      
      // 验证结果
      assert.True(t, env.IsWorkflowCompleted())
      assert.NoError(t, env.GetWorkflowError())
  }
  
  func TestWaterflowWorkflow_StepFailure(t *testing.T) {
      testSuite := &testsuite.WorkflowTestSuite{}
      env := testSuite.NewTestWorkflowEnvironment()
      
      // Mock Activity返回错误
      env.OnActivity("ExecuteStepActivity", mock.Anything, mock.Anything).Return(
          ExecuteStepResult{}, fmt.Errorf("step failed"),
      )
      
      def := &parser.WorkflowDefinition{
          Name: "Test Workflow",
          On:   "push",
          Jobs: map[string]parser.Job{
              "build": {
                  RunsOn: "linux",
                  Steps: []parser.Step{
                      {Name: "Failing Step", Uses: "fail@v1"},
                  },
              },
          },
      }
      
      env.ExecuteWorkflow(WaterflowWorkflow, def)
      
      assert.True(t, env.IsWorkflowCompleted())
      assert.Error(t, env.GetWorkflowError())
  }
  
  func TestWaterflowWorkflow_MultipleJobs_Error(t *testing.T) {
      testSuite := &testsuite.WorkflowTestSuite{}
      env := testSuite.NewTestWorkflowEnvironment()
      
      def := &parser.WorkflowDefinition{
          Name: "Multi Job Workflow",
          On:   "push",
          Jobs: map[string]parser.Job{
              "build": {RunsOn: "linux", Steps: []parser.Step{{Name: "s1", Uses: "run@v1"}}},
              "test":  {RunsOn: "linux", Steps: []parser.Step{{Name: "s2", Uses: "run@v1"}}},
          },
      }
      
      env.ExecuteWorkflow(WaterflowWorkflow, def)
      
      assert.True(t, env.IsWorkflowCompleted())
      assert.Error(t, env.GetWorkflowError())
      assert.Contains(t, env.GetWorkflowError().Error(), "MVP only supports single job")
  }
  ```

- [ ] 6.2 测试Activity
  ```go
  // internal/workflow/activities_test.go
  
  func TestExecuteStepActivity_Success(t *testing.T) {
      testSuite := &testsuite.WorkflowTestSuite{}
      env := testSuite.NewTestActivityEnvironment()
      
      input := ExecuteStepInput{
          Name: "Test Step",
          Uses: "run@v1",
          With: map[string]interface{}{"command": "echo test"},
      }
      
      val, err := env.ExecuteActivity(ExecuteStepActivity, input)
      
      assert.NoError(t, err)
      
      var result ExecuteStepResult
      val.Get(&result)
      assert.Equal(t, 0, result.ExitCode)
      assert.NotEmpty(t, result.Output)
  }
  ```

- [ ] 6.3 运行测试
  ```bash
  go test -v ./internal/workflow
  # 期望: 所有测试通过
  ```

### Task 7: 端到端集成测试 (验证完整流程)

- [ ] 7.1 创建集成测试脚本
  ```bash
  # test/integration/test_workflow_execution.sh
  #!/bin/bash
  
  set -e
  
  echo "=== Waterflow Workflow Execution Integration Test ==="
  
  # 1. 启动Temporal
  echo "Starting Temporal..."
  cd deployments
  docker-compose up -d temporal
  sleep 10
  
  # 2. 启动Waterflow Server
  echo "Starting Waterflow Server..."
  cd ..
  go run ./cmd/server &
  SERVER_PID=$!
  sleep 3
  
  # 3. 提交工作流
  echo "Submitting workflow..."
  RESPONSE=$(curl -s -X POST http://localhost:8080/v1/workflows \
    -H "Content-Type: application/json" \
    -d '{
      "workflow": "name: Test Build\non: push\njobs:\n  build:\n    runs-on: linux-amd64\n    steps:\n      - name: Build\n        uses: run@v1\n        with:\n          command: echo Building"
    }')
  
  echo "Response: $RESPONSE"
  
  # 4. 提取WorkflowID
  WORKFLOW_ID=$(echo $RESPONSE | jq -r '.workflow_id')
  echo "Workflow ID: $WORKFLOW_ID"
  
  # 5. 等待执行完成
  echo "Waiting for workflow to complete..."
  sleep 5
  
  # 6. 查询工作流状态 (使用Story 1.7 API - 如果已实现)
  echo "Querying workflow status..."
  STATUS=$(curl -s http://localhost:8080/v1/workflows/$WORKFLOW_ID || echo '{"status":"unknown"}')
  echo "Status Response: $STATUS"
  
  # 验证状态 (如果API可用)
  if command -v jq &> /dev/null; then
      WORKFLOW_STATUS=$(echo "$STATUS" | jq -r '.status // "unknown"')
      if [ "$WORKFLOW_STATUS" = "completed" ]; then
          echo "✅ Workflow completed successfully"
      elif [ "$WORKFLOW_STATUS" = "running" ]; then
          echo "⏳ Workflow still running (may need more time)"
      else
          echo "⚠️  Workflow status: $WORKFLOW_STATUS"
      fi
  fi
  
  # 7. 在Temporal UI查看 (手动验证)
  echo "Check Temporal UI: http://localhost:8233"
  echo "  → Workflow ID: $WORKFLOW_ID"
  
  # 8. 测试失败场景
  echo "=== Testing Workflow Failure Scenario ==="
  
  FAILURE_RESPONSE=$(curl -s -X POST http://localhost:8080/v1/workflows \
    -H "Content-Type: application/json" \
    -d '{
      "workflow": "name: Failing Workflow\\non: push\\njobs:\\n  fail:\\n    runs-on: default\\n    steps:\\n      - name: Fail Step\\n        uses: fail@v1"
    }')
  
  FAILED_WF_ID=$(echo $FAILURE_RESPONSE | jq -r '.workflow_id')
  echo "Failed Workflow ID: $FAILED_WF_ID"
  
  # 等待失败
  sleep 10
  
  # 验证状态为failed
  FAILED_STATUS=$(curl -s http://localhost:8080/v1/workflows/$FAILED_WF_ID | jq -r '.status')
  if [ "$FAILED_STATUS" = "failed" ]; then
      echo "✅ Workflow failure handling works correctly"
  else
      echo "❌ Expected status=failed, got $FAILED_STATUS"
      # 不退出,继续清理
  fi
  
  # 9. 测试工作流取消
  echo "=== Testing Workflow Cancellation ==="
  
  # 提交一个长时间运行的工作流
  CANCEL_RESPONSE=$(curl -s -X POST http://localhost:8080/v1/workflows \
    -H "Content-Type: application/json" \
    -d '{
      "workflow": "name: Long Running\\non: push\\njobs:\\n  long:\\n    runs-on: default\\n    timeout-minutes: 60\\n    steps:\\n      - name: Sleep\\n        uses: sleep@v1\\n        with:\\n          duration: 300"
    }')
  
  CANCEL_WF_ID=$(echo $CANCEL_RESPONSE | jq -r '.workflow_id')
  echo "Long-running Workflow ID: $CANCEL_WF_ID"
  
  # 等待开始执行
  sleep 3
  
  # 取消工作流 (如果Story 1.9已实现)
  if curl -s -X POST http://localhost:8080/v1/workflows/$CANCEL_WF_ID/cancel > /dev/null 2>&1; then
      echo "✅ Workflow cancellation API available"
      sleep 2
      CANCELLED_STATUS=$(curl -s http://localhost:8080/v1/workflows/$CANCEL_WF_ID | jq -r '.status')
      echo "Cancelled workflow status: $CANCELLED_STATUS"
  else
      echo "⚠️  Workflow cancellation API not yet implemented (Story 1.9)"
  fi
  
  # 10. 清理
  echo "Cleaning up..."
  kill $SERVER_PID
  cd deployments
  docker-compose down
  
  echo "✅ Integration test completed (success + failure scenarios)"
  ```

- [ ] 7.2 添加到CI
  ```yaml
  # .github/workflows/ci.yml
  
  - name: Integration Test
    run: |
      chmod +x test/integration/test_workflow_execution.sh
      ./test/integration/test_workflow_execution.sh
  ```

### Task 8: 性能基准测试 (AC: 单个Step执行启动延迟<100ms)

- [ ] 8.1 创建`internal/workflow/benchmark_test.go`
  ```go
  package workflow
  
  import (
      "testing"
      "time"
      
      "github.com/stretchr/testify/assert"
      "github.com/stretchr/testify/mock"
      "go.temporal.io/sdk/testsuite"
      
      "waterflow/internal/parser"
  )
  
  // BenchmarkActivitySchedulingLatency 测试Activity调度延迟
  // 验证AC: 单个Step执行启动延迟 < 100ms
  func BenchmarkActivitySchedulingLatency(b *testing.B) {
      testSuite := &testsuite.WorkflowTestSuite{}
      env := testSuite.NewTestWorkflowEnvironment()
      
      // Mock Activity (立即返回)
      env.OnActivity("ExecuteStepActivity", mock.Anything, mock.Anything).Return(
          ExecuteStepResult{
              Output:   "benchmark",
              ExitCode: 0,
              Duration: 1 * time.Millisecond,
          }, nil,
      )
      
      // 准备最小工作流
      def := &parser.WorkflowDefinition{
          Name: "Benchmark Workflow",
          On:   "push",
          Jobs: map[string]parser.Job{
              "bench": {
                  RunsOn: "default",
                  Steps: []parser.Step{
                      {
                          Name: "Benchmark Step",
                          Uses: "run@v1",
                          With: map[string]interface{}{"command": "echo test"},
                      },
                  },
              },
          },
      }
      
      b.ResetTimer()
      
      // 运行基准测试
      for i := 0; i < b.N; i++ {
          start := time.Now()
          env.ExecuteWorkflow(WaterflowWorkflow, def)
          latency := time.Since(start)
          
          // 验证AC: 启动延迟必须<100ms
          if latency > 100*time.Millisecond {
              b.Errorf("Step启动延迟超过100ms: %v (第%d次)", latency, i+1)
          }
      }
  }
  
  // BenchmarkMultiStepLatency 测试多Step场景的总延迟
  func BenchmarkMultiStepLatency(b *testing.B) {
      testSuite := &testsuite.WorkflowTestSuite{}
      env := testSuite.NewTestWorkflowEnvironment()
      
      env.OnActivity("ExecuteStepActivity", mock.Anything, mock.Anything).Return(
          ExecuteStepResult{ExitCode: 0}, nil,
      )
      
      // 10个Steps的工作流
      steps := make([]parser.Step, 10)
      for i := 0; i < 10; i++ {
          steps[i] = parser.Step{
              Name: fmt.Sprintf("Step %d", i+1),
              Uses: "run@v1",
          }
      }
      
      def := &parser.WorkflowDefinition{
          Name: "Multi-Step Benchmark",
          On:   "push",
          Jobs: map[string]parser.Job{
              "multi": {
                  RunsOn: "default",
                  Steps:  steps,
              },
          },
      }
      
      b.ResetTimer()
      
      for i := 0; i < b.N; i++ {
          start := time.Now()
          env.ExecuteWorkflow(WaterflowWorkflow, def)
          totalLatency := time.Since(start)
          
          avgLatency := totalLatency / 10
          
          // 平均每个Step<100ms
          if avgLatency > 100*time.Millisecond {
              b.Errorf("平均Step延迟超过100ms: %v", avgLatency)
          }
      }
  }
  ```

- [ ] 8.2 运行基准测试并记录结果
  ```bash
  # 运行性能基准测试
  go test -bench=BenchmarkActivitySchedulingLatency \
          -benchtime=100x \
          -benchmem \
          ./internal/workflow
  
  # 期望输出示例:
  # BenchmarkActivitySchedulingLatency-8    100    45000 ns/op    2048 B/op    25 allocs/op
  #                                               ^^^^^^^ <100ms = <100,000,000 ns
  
  # 记录基准结果
  go test -bench=. -benchmem ./internal/workflow > docs/benchmark-story-1-6.txt
  
  echo "✅ Performance benchmarks recorded"
  ```

- [ ] 8.3 添加到CI流程作为性能门禁
  ```yaml
  # .github/workflows/ci.yml (添加到现有workflow)
  
  - name: Performance Benchmark Tests
    run: |
      # 运行基准测试,如果延迟超标则失败
      go test -bench=BenchmarkActivitySchedulingLatency \
              -benchtime=50x \
              ./internal/workflow || {
        echo "❌ Performance regression detected - Step latency exceeds 100ms"
        exit 1
      }
      
      echo "✅ Performance benchmarks passed"
  ```

### Task 9: 更新文档和日志 (可观测性)

- [ ] 8.1 添加日志输出
  ```go
  func WaterflowWorkflow(ctx workflow.Context, def *parser.WorkflowDefinition) error {
      logger := workflow.GetLogger(ctx)
      
      // 记录Workflow信息
      info := workflow.GetInfo(ctx)
      logger.Info("Workflow execution info",
          "workflow_id", info.WorkflowExecution.ID,
          "run_id", info.WorkflowExecution.RunID,
          "task_queue", info.TaskQueueName,
      )
      
      // ... 执行逻辑 ...
      
      logger.Info("Workflow metrics",
          "total_steps", getTotalSteps(def),
          "duration", workflow.Now(ctx).Sub(info.WorkflowStartTime),
      )
      
      return nil
  }
  ```

- [ ] 8.2 更新README
  ```markdown
  # Waterflow
  
  ## Quick Start
  
  1. Start Temporal:
     ```bash
     make dev-env
     ```
  
  2. Start Waterflow Server:
     ```bash
     go run ./cmd/server
     ```
  
  3. Submit a workflow:
     ```bash
     curl -X POST http://localhost:8080/v1/workflows \
       -H "Content-Type: application/json" \
       -d @examples/simple-build.json
     ```
  
  4. View in Temporal UI:
     ```
     http://localhost:8233
     ```
  ```

## Dev Notes

### Critical Implementation Guidelines

**1. Temporal Workflow确定性**

```go
// ❌ 错误: 非确定性操作
func BadWorkflow(ctx workflow.Context) error {
    time.Sleep(1 * time.Second) // 不会生效!
    rand.Seed(time.Now().Unix()) // 重放时不一致!
    return nil
}

// ✅ 正确: 使用Temporal API
func GoodWorkflow(ctx workflow.Context) error {
    workflow.Sleep(ctx, 1*time.Second) // Temporal Timer
    // 随机数使用SideEffect
    var random int
    workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
        return rand.Intn(100)
    }).Get(&random)
    return nil
}
```

**2. Activity超时配置**

```go
// ✅ 合理的超时配置
activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
    StartToCloseTimeout: 10 * time.Minute,  // Activity总超时
    ScheduleToStartTimeout: 1 * time.Minute, // 调度超时
    HeartbeatTimeout: 30 * time.Second,      // 心跳超时
})

// ❌ 避免: 无限超时
activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
    StartToCloseTimeout: 0, // 危险!
})
```

**3. 错误处理最佳实践**

```go
// ✅ 区分错误类型
err := workflow.ExecuteActivity(ctx, "Activity", input).Get(ctx, &result)
if err != nil {
    if temporal.IsTimeoutError(err) {
        // 超时错误 - 可能需要重试
        workflow.GetLogger(ctx).Warn("Activity timeout", "error", err)
    } else if temporal.IsApplicationError(err) {
        // 应用错误 - 不应重试
        return err
    }
}
```

**4. Worker注册顺序**

```go
// ✅ 先注册再启动
worker := worker.New(client, "taskQueue", worker.Options{})
worker.RegisterWorkflow(MyWorkflow) // 必须在Run之前
worker.RegisterActivity(MyActivity)
worker.Run(worker.InterruptCh())

// ❌ 启动后注册无效
worker.Run(...) // 阻塞
worker.RegisterWorkflow(...) // 永远不会执行
```

**5. MVP范围控制**

```go
// ✅ 明确MVP限制,添加验证
if len(def.Jobs) > 1 {
    return fmt.Errorf("MVP only supports single job (found %d)", len(def.Jobs))
}

// 添加TODO注释
// TODO(Story 3.x): Support parallel job execution
for jobName, job := range def.Jobs {
    break // MVP只执行第一个
}
```

**6. 测试策略**

```go
// ✅ 使用Temporal测试框架
testSuite := &testsuite.WorkflowTestSuite{}
env := testSuite.NewTestWorkflowEnvironment()

// Mock Activity行为
env.OnActivity("MyActivity", mock.Anything).Return("result", nil)

// 执行Workflow
env.ExecuteWorkflow(MyWorkflow, input)

// 验证结果
assert.True(t, env.IsWorkflowCompleted())
assert.NoError(t, env.GetWorkflowError())
```

### Integration with Previous Stories

**与Story 1.3 YAML解析器集成:**

```go
// Story 1.3提供的数据结构
type WorkflowDefinition struct {
    Name string
    Jobs map[string]Job
}

// Story 1.6直接使用
func WaterflowWorkflow(ctx workflow.Context, def *parser.WorkflowDefinition) error {
    for jobName, job := range def.Jobs {
        // 执行job
    }
}
```

**与Story 1.4 Temporal Client集成:**

```go
// Story 1.4提供的Client
temporalClient.GetClient()

// Story 1.6使用Client创建Worker
worker := worker.New(temporalClient.GetClient(), "default", ...)
```

**与Story 1.5 工作流提交集成:**

```go
// Story 1.5提交Workflow
we, err := client.ExecuteWorkflow(ctx, options, "WaterflowWorkflow", def)

// Story 1.6实现WaterflowWorkflow
func WaterflowWorkflow(ctx workflow.Context, def *parser.WorkflowDefinition) error {
    // 执行逻辑
}
```

**为Story 1.7准备:**

```go
// Story 1.7将查询Workflow状态
describe, err := client.DescribeWorkflowExecution(ctx, workflowID, runID)
status := describe.WorkflowExecutionInfo.Status
// 状态已由Story 1.6的Workflow执行自动持久化
```

### Testing Strategy

**单元测试 (Temporal TestSuite):**

| 测试场景 | 目的 |
|---------|-----|
| 单Job单Step成功 | 验证基本执行流程 |
| Step失败处理 | 验证错误传播 |
| Step超时 | 验证超时控制 |
| 多Job拒绝 | 验证MVP限制 |
| Activity Mock | 验证Activity调用参数 |

**集成测试:**

```bash
# 1. 启动Temporal + Waterflow
make dev-env
go run ./cmd/server

# 2. 提交工作流
curl -X POST http://localhost:8080/v1/workflows -d @test.json

# 3. 在Temporal UI观察执行
open http://localhost:8233

# 4. 验证Event History
# - WorkflowExecutionStarted
# - ActivityTaskScheduled (每个Step)
# - ActivityTaskCompleted
# - WorkflowExecutionCompleted
```

### References

**架构设计:**
- [docs/architecture.md §3.2](docs/architecture.md) - Agent内部组件
- [docs/adr/0002-single-node-execution-pattern.md](docs/adr/0002-single-node-execution-pattern.md) - 单节点执行模式

**技术文档:**
- [Temporal Workflow API](https://docs.temporal.io/docs/go/workflows/)
- [Temporal Activity API](https://docs.temporal.io/docs/go/activities/)
- [Temporal Testing](https://docs.temporal.io/docs/go/testing/)

**项目上下文:**
- [docs/epics.md Story 1.1-1.5](docs/epics.md) - 前置Stories
- [docs/epics.md Story 1.7](docs/epics.md) - 状态查询API
- [docs/epics.md Story 2.x](docs/epics.md) - Agent系统 (实现真实Activity)

### Dependency Graph

```
Story 1.3 (YAML解析) ──┐
Story 1.4 (Temporal)  ─┤
Story 1.5 (提交API)   ─┤
                       ↓
Story 1.6 (执行引擎) ← 当前Story
    ↓
    ├→ Story 1.7 (状态查询) - 查询Workflow执行状态
    ├→ Story 1.8 (日志输出) - 获取Activity日志
    └→ Story 2.x (Agent系统) - 实现真实Activity执行
```

## Dev Agent Record

### Context Reference

**Source Documents Analyzed:**
1. [docs/epics.md](docs/epics.md) (lines 362-377) - Story 1.6需求定义
2. [docs/architecture.md](docs/architecture.md) (§3.2) - Agent组件设计
3. [docs/adr/0002-single-node-execution-pattern.md](docs/adr/0002-single-node-execution-pattern.md) - 单节点执行模式

**Previous Stories:**
- Story 1.1-1.5: 全部drafted (框架、API、解析器、Temporal、提交)

### Agent Model Used

Claude 3.5 Sonnet (BMM Scrum Master Agent - Bob)

### Estimated Effort

**开发时间:** 8-10小时  
**复杂度:** 中高

**时间分解:**
- Workflow函数实现: 2.5小时
- Activity定义和Mock: 1.5小时
- Worker管理集成: 1.5小时
- 错误处理和状态: 1小时
- Workflow单元测试: 2小时
- 集成测试: 1.5小时
- 文档更新: 1小时

**技能要求:**
- Temporal Workflow编程 (确定性要求)
- Temporal Activity API
- Temporal测试框架
- 异步编程和错误处理

### Debug Log References

<!-- Will be populated during implementation -->

### Completion Notes List

<!-- Developer填写完成时的笔记 -->

### File List

**预期创建/修改的文件清单:**

```
新建文件 (~5个):
├── internal/workflow/
│   ├── waterflow_workflow.go       # 详见Task 1
│   ├── activities.go               # 详见Task 2
│   ├── worker.go                   # 详见Task 3
│   ├── waterflow_workflow_test.go  # 详见Task 6
│   └── activities_test.go          # 详见Task 6
├── test/integration/
│   └── test_workflow_execution.sh  # 详见Task 7

修改文件 (~3个):
├── internal/server/server.go       # 详见Task 3.2 (Worker集成)
├── internal/service/workflow_service.go  # 详见Task 4 (更新提交逻辑)
└── cmd/server/main.go              # Worker生命周期管理
```

**详细实现代码请参考Tasks 0-7各小节,此处省略以节省token。**

**关键技术要点:**
- Event Sourcing架构 (状态100%持久化到Temporal Event History)
- Workflow确定性要求 (禁止time.Now()、rand、直接IO)
- 单节点执行模式 (每个Step = 1个Activity)
- MVP Mock实现 (Story 2.x将替换为真实Agent调用)

---

**Story Ready for Development** ✅

开发者可基于Story 1.1-1.5的成果,实现Temporal Workflow执行引擎。
本Story实现端到端MVP流程: 提交 → 执行 → 持久化状态。
