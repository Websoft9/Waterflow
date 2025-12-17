# Story 1.9: 工作流取消 API

Status: ready-for-dev

## Story

As a **工作流用户**,  
I want **取消正在运行的工作流**,  
So that **停止不需要的执行释放资源**。

## Acceptance Criteria

**Given** 工作流正在运行  
**When** POST `/v1/workflows/{id}/cancel` 请求取消  
**Then** 工作流标记为 cancelled 状态  
**And** Temporal Workflow 收到取消信号  
**And** 正在执行的 Step 优雅停止  
**And** 取消已完成的工作流返回 409  
**And** 取消成功返回 202  
**And** 重复取消请求返回 202 (幂等性)

## Technical Context

### Architecture Constraints

根据 [docs/architecture.md](docs/architecture.md) §3.1.1 REST API Handler 和 §3.1.5 Temporal Client 设计:

1. **核心职责**
   - 处理 `POST /v1/workflows/{id}/cancel` 请求
   - 通过 Temporal Client 取消工作流
   - 向 Temporal 发送 Cancel 信号
   - 返回取消状态和结果

2. **Event Sourcing 架构** (参考 ADR-0001)
   - 取消操作记录在 Temporal Event History
   - 支持优雅取消:正在执行的 Activity 会接收取消信号
   - 已完成的 Step 不会回滚,仅停止后续执行

3. **功能需求映射**
   - **FR3 工作流管理 API**: 本 Story 实现 POST /v1/workflows/{id}/cancel 端点
   - **FR7 实时状态跟踪**: 取消状态实时更新

2. **取消流程**

```
User Request                 Waterflow Server              Temporal Server
     │                              │                             │
     ├─ POST /cancel ──────────────→│                             │
     │                              │                             │
     │                              ├─ 1. 查询状态 ──────────────→│
     │                              │←─ running ─────────────────┤
     │                              │                             │
     │                              ├─ 2. 发送取消信号 ──────────→│
     │                              │   CancelWorkflow()          │
     │←─ 202 Accepted ──────────────┤                             │
     │   (取消请求已接受)            │                             │
     │                              │                             │
     │                              │                 Workflow执行中收到取消:
     │                              │                 ctx.Err() == Canceled
     │                              │                             │
     │                              │←─ WorkflowCanceled ─────────┤
```

3. **响应格式**

**成功取消 (202 Accepted):**
```json
{
  "workflow_id": "wf-550e8400-e29b-41d4-a716-446655440000",
  "status": "canceling",
  "message": "Workflow cancellation requested"
}
```

**工作流不存在 (404):**
```json
{
  "type": "https://waterflow.io/errors/not-found",
  "title": "Workflow Not Found",
  "status": 404,
  "detail": "Workflow with ID 'wf-xxx' does not exist"
}
```

**工作流已完成 (409 Conflict):**
```json
{
  "type": "https://waterflow.io/errors/conflict",
  "title": "Cannot Cancel Completed Workflow",
  "status": 409,
  "detail": "Workflow 'wf-xxx' has already completed",
  "current_status": "completed"
}
```

4. **状态验证规则**

| 当前状态 | 可否取消 | HTTP响应 |
|---------|---------|----------|
| running | ✅ 可取消 | 202 Accepted |
| completed | ❌ 不可取消 | 409 Conflict |
| failed | ❌ 不可取消 | 409 Conflict |
| canceled | ❌ 已取消 | 409 Conflict |
| timeout | ❌ 不可取消 | 409 Conflict |

### Dependencies

**前置 Story:**
- ✅ Story 1.4: Temporal SDK 集成
  - 使用: `CancelWorkflow` API
- ✅ Story 1.5: 工作流提交 API
  - 使用: WorkflowID 验证
- ✅ Story 1.6: 基础工作流执行引擎
  - 使用: Workflow Context 取消处理
- ✅ Story 1.7: 工作流状态查询 API
  - 使用: 状态查询逻辑验证是否可取消

**后续 Story 依赖本 Story:**
- Story 2.x: Agent 执行取消 - 需优雅停止正在执行的 Activity

### Technology Stack

**Temporal Cancel API:**

```go
import (
    "go.temporal.io/sdk/client"
)

// 1. CancelWorkflow - 发送取消信号
err := temporalClient.CancelWorkflow(ctx, workflowID, runID)

// 特点:
// - 异步操作,立即返回
// - Workflow会收到context.Canceled错误
// - Activity可能仍在执行,需优雅处理
```

**Workflow 中处理取消:**

```go
// internal/temporal/workflow.go (Story 1.6 已实现)

func WaterflowWorkflow(ctx workflow.Context, def *WorkflowDefinition) error {
    for _, job := range def.Jobs {
        for _, step := range job.Steps {
            // 检查取消信号
            if ctx.Err() != nil {
                return ctx.Err() // 返回 context.Canceled
            }
            
            // 执行 Step
            err := workflow.ExecuteActivity(ctx, ExecuteStepActivity, step).Get(ctx, nil)
            if err != nil {
                return err
            }
        }
    }
    return nil
}
```

**Activity 优雅停止:**

```go
// Activity 应定期检查 Context
func ExecuteStepActivity(ctx context.Context, step StepInput) error {
    // 长时间运行的操作应分段检查
    for i := 0; i < 100; i++ {
        // 检查取消
        if ctx.Err() != nil {
            return ctx.Err() // 立即返回
        }
        
        // 执行部分工作
        doWork()
    }
    return nil
}
```

**CancelWorkflow vs TerminateWorkflow:**

| API | 行为 | 使用场景 |
|-----|------|---------|
| CancelWorkflow | 发送取消信号,Workflow可优雅处理 | 用户主动取消 |
| TerminateWorkflow | 强制终止,不执行清理逻辑 | 异常终止,紧急停止 |

**本 Story 使用:** `CancelWorkflow` (优雅取消)

### Project Structure Updates

基于 Story 1.1-1.8 的结构,本 Story 新增:

```
internal/
├── service/
│   ├── workflow_cancel_service.go      # 取消服务层 (新建)
│   └── workflow_cancel_service_test.go (新建)
├── server/handlers/
│   └── workflow.go                     # 修改 - 添加CancelWorkflow方法
├── models/
│   └── workflow_cancel.go              # 取消响应模型 (新建)

api/
└── openapi.yaml                        # 更新 - 添加POST /v1/workflows/{id}/cancel
```

## Tasks / Subtasks

### Task 0: 验证依赖 (AC: 开发环境就绪)

- [ ] 0.1 验证Temporal连接
  ```bash
  # 确保Temporal Server运行
  curl -s localhost:7233 > /dev/null && echo "✅ Temporal running" || echo "❌ Temporal not running"
  ```

- [ ] 0.2 验证Go环境
  ```bash
  go version | grep "go1.21" && echo "✅ Go 1.21+" || echo "❌ Go version mismatch"
  ```

- [ ] 0.3 验证前置Story依赖文件
  ```bash
  # test/verify-dependencies-story-1-9.sh
  #!/bin/bash
  
  set -e
  
  echo "=== Story 1.9 依赖验证 ==="
  
  # 函数: 检查文件是否存在
  check_file() {
      local file=$1
      local story=$2
      
      if [ -f "$file" ]; then
          echo "✅ $story: $file"
      else
          echo "❌ $story: $file NOT FOUND"
          exit 1
      fi
  }
  
  # Story 1.1: Server Framework
  check_file "internal/config/config.go" "Story 1.1"
  check_file "cmd/server/main.go" "Story 1.1"
  
  # Story 1.2: REST API Framework
  check_file "internal/server/server.go" "Story 1.2"
  check_file "internal/server/router.go" "Story 1.2"
  
  # Story 1.3: YAML Parser
  check_file "internal/parser/yaml_parser.go" "Story 1.3"
  check_file "internal/models/workflow_definition.go" "Story 1.3"
  
  # Story 1.4: Temporal SDK Integration (CancelWorkflow API)
  check_file "internal/temporal/client.go" "Story 1.4"
  
  # Story 1.5: Workflow Submission API
  check_file "internal/service/workflow_service.go" "Story 1.5"
  check_file "internal/server/handlers/workflow.go" "Story 1.5"
  
  # Story 1.6: Workflow Execution Engine (to be enhanced with cancel checks)
  check_file "internal/workflow/waterflow_workflow.go" "Story 1.6"
  check_file "internal/workflow/activities.go" "Story 1.6"
  check_file "internal/workflow/worker.go" "Story 1.6"
  
  # Story 1.7: Workflow Status Query API (for isCancelable validation)
  check_file "internal/service/workflow_query_service.go" "Story 1.7"
  check_file "internal/models/workflow_status.go" "Story 1.7"
  
  # 验证Temporal连接
  echo ""
  echo "检查Temporal Server连接..."
  if curl -s localhost:7233 > /dev/null 2>&1; then
      echo "✅ Temporal Server运行中 (localhost:7233)"
  else
      echo "❌ Temporal Server未运行 - 请启动Temporal"
      echo "   提示: make dev-env 或 docker-compose up temporal"
      exit 1
  fi
  
  echo ""
  echo "✅ Story 1.9 所有依赖验证通过"
  ```

- [ ] 0.4 运行验证脚本
  ```bash
  chmod +x test/verify-dependencies-story-1-9.sh
  ./test/verify-dependencies-story-1-9.sh
  ```

### Task 1: 定义取消响应模型 (AC: 返回 202/409)

- [ ] 1.1 创建 `internal/models/workflow_cancel.go`
  ```go
  package models
  
  // CancelWorkflowResponse 取消响应
  type CancelWorkflowResponse struct {
      WorkflowID string `json:"workflow_id"`
      Status     string `json:"status"` // canceling
      Message    string `json:"message"`
  }
  
  // ConflictError 409冲突错误
  type ConflictError struct {
      Type          string `json:"type"`
      Title         string `json:"title"`
      Status        int    `json:"status"`
      Detail        string `json:"detail"`
      CurrentStatus string `json:"current_status,omitempty"`
  }
  
  // NewConflictError 创建409错误
  func NewConflictError(workflowID, currentStatus string) *ConflictError {
      return &ConflictError{
          Type:          "https://waterflow.io/errors/conflict",
          Title:         "Cannot Cancel Workflow",
          Status:        409,
          Detail:        fmt.Sprintf("Workflow '%s' cannot be canceled in current state", workflowID),
          CurrentStatus: currentStatus,
      }
  }
  ```

### Task 2: 实现取消服务层 (AC: Temporal取消信号)

- [ ] 2.1 创建 `internal/service/workflow_cancel_service.go`
  ```go
  package service
  
  import (
      "context"
      "fmt"
      
      "go.temporal.io/sdk/client"
      "go.uber.org/zap"
      
      "waterflow/internal/models"
      "waterflow/internal/temporal"
  )
  
  type WorkflowCancelService struct {
      temporalClient *temporal.Client
      queryService   *WorkflowQueryService // 复用状态查询
      logger         *zap.Logger
  }
  
  func NewWorkflowCancelService(
      tc *temporal.Client,
      qs *WorkflowQueryService,
      logger *zap.Logger,
  ) *WorkflowCancelService {
      return &WorkflowCancelService{
          temporalClient: tc,
          queryService:   qs,
          logger:         logger,
      }
  }
  
  // CancelWorkflow 取消工作流
  func (wcs *WorkflowCancelService) CancelWorkflow(ctx context.Context, workflowID string) (*models.CancelWorkflowResponse, error) {
      // 1. 查询当前状态
      status, err := wcs.queryService.GetWorkflowStatus(ctx, workflowID)
      if err != nil {
          wcs.logger.Error("Failed to get workflow status for cancellation",
              zap.String("workflow_id", workflowID),
              zap.Error(err),
          )
          return nil, fmt.Errorf("workflow not found: %w", err)
      }
      
      // 2. 检查是否已在取消中 (幂等性)
      if status.Status == "canceling" {
          wcs.logger.Info("Workflow already canceling, request is idempotent",
              zap.String("workflow_id", workflowID),
          )
          return &models.CancelWorkflowResponse{
              WorkflowID: workflowID,
              Status:     "canceling",
              Message:    "Workflow cancellation already in progress",
          }, nil
      }
      
      // 3. 验证状态是否可取消
      if !wcs.isCancelable(status.Status) {
          wcs.logger.Warn("Attempted to cancel non-cancelable workflow",
              zap.String("workflow_id", workflowID),
              zap.String("current_status", status.Status),
          )
          return nil, &CancelNotAllowedError{
              WorkflowID:    workflowID,
              CurrentStatus: status.Status,
          }
      }
      
      // 4. 发送取消信号
      err = wcs.temporalClient.GetClient().CancelWorkflow(ctx, workflowID, status.RunID)
      if err != nil {
          wcs.logger.Error("Failed to cancel workflow",
              zap.String("workflow_id", workflowID),
              zap.Error(err),
          )
          return nil, fmt.Errorf("failed to cancel workflow: %w", err)
      }
      
      wcs.logger.Info("Workflow cancellation requested",
          zap.String("workflow_id", workflowID),
          zap.String("run_id", status.RunID),
      )
      
      // 5. 返回成功响应
      return &models.CancelWorkflowResponse{
          WorkflowID: workflowID,
          Status:     "canceling",
          Message:    "Workflow cancellation requested",
      }, nil
  }
  
  // isCancelable 检查状态是否可取消
  func (wcs *WorkflowCancelService) isCancelable(status string) bool {
      cancelableStates := map[string]bool{
          "running":   true,
          "canceling": true,  // 支持幂等性: 重复取消返回202
      }
      return cancelableStates[status]
  }
  
  // CancelNotAllowedError 不可取消错误
  type CancelNotAllowedError struct {
      WorkflowID    string
      CurrentStatus string
  }
  
  func (e *CancelNotAllowedError) Error() string {
      return fmt.Sprintf("workflow %s cannot be canceled in state %s", e.WorkflowID, e.CurrentStatus)
  }
  ```

### Task 3: 实现 HTTP Handler (AC: POST /v1/workflows/{id}/cancel)

- [ ] 3.1 更新 `internal/server/handlers/workflow.go`
  ```go
  // CancelWorkflow - POST /v1/workflows/:id/cancel
  func (h *WorkflowHandler) CancelWorkflow(c *gin.Context) {
      workflowID := c.Param("id")
      
      // 验证 WorkflowID
      if workflowID == "" || !strings.HasPrefix(workflowID, "wf-") {
          c.JSON(http.StatusBadRequest, models.NewBadRequestError(
              "Invalid workflow ID format",
          ))
          return
      }
      
      // 执行取消
      ctx := c.Request.Context()
      response, err := h.workflowCancelService.CancelWorkflow(ctx, workflowID)
      
      if err != nil {
          // 检查是否为 not found 错误
          if strings.Contains(err.Error(), "not found") {
              c.JSON(http.StatusNotFound, &models.ErrorResponse{
                  Type:   "https://waterflow.io/errors/not-found",
                  Title:  "Workflow Not Found",
                  Status: 404,
                  Detail: fmt.Sprintf("Workflow with ID '%s' does not exist", workflowID),
              })
              return
          }
          
          // 检查是否为状态冲突错误
          if cancelErr, ok := err.(*service.CancelNotAllowedError); ok {
              c.JSON(http.StatusConflict, models.NewConflictError(
                  workflowID,
                  cancelErr.CurrentStatus,
              ))
              return
          }
          
          // 其他错误
          h.logger.Error("Failed to cancel workflow",
              zap.String("workflow_id", workflowID),
              zap.Error(err),
          )
          c.JSON(http.StatusInternalServerError, &models.ErrorResponse{
              Type:   "https://waterflow.io/errors/internal-error",
              Title:  "Internal Server Error",
              Status: 500,
              Detail: "Failed to cancel workflow",
          })
          return
      }
      
      // 返回 202 Accepted
      c.JSON(http.StatusAccepted, response)
  }
  ```

### Task 4: 增强 Workflow 取消处理 (AC: 优雅停止)

- [ ] 4.1 更新 `internal/temporal/workflow.go` (Story 1.6)
  ```go
  // WaterflowWorkflow - 主工作流函数
  func WaterflowWorkflow(ctx workflow.Context, def *WorkflowDefinition) error {
      logger := workflow.GetLogger(ctx)
      
      logger.Info("Workflow started", "name", def.Name)
      
      // 执行每个 Job (MVP: 单 Job)
      for jobName, job := range def.Jobs {
          // 检查取消信号
          if err := ctx.Err(); err != nil {
              logger.Warn("Workflow canceled before job execution",
                  "job", jobName,
                  "error", err,
              )
              return err // 返回 context.Canceled
          }
          
          err := executeJob(ctx, jobName, job)
          if err != nil {
              logger.Error("Job execution failed", "job", jobName, "error", err)
              return err
          }
      }
      
      logger.Info("Workflow completed successfully")
      return nil
  }
  
  // executeJob - 执行单个 Job
  func executeJob(ctx workflow.Context, jobName string, job JobDefinition) error {
      logger := workflow.GetLogger(ctx)
      
      // 串行执行 Steps (MVP)
      for _, step := range job.Steps {
          // 每个 Step 执行前检查取消
          if err := ctx.Err(); err != nil {
              logger.Warn("Job canceled before step execution",
                  "job", jobName,
                  "step", step.Name,
                  "error", err,
              )
              return err
          }
          
          err := executeStep(ctx, step)
          if err != nil {
              return fmt.Errorf("step '%s' failed: %w", step.Name, err)
          }
      }
      
      return nil
  }
  
  // executeStep - 执行单个 Step
  func executeStep(ctx workflow.Context, step StepDefinition) error {
      logger := workflow.GetLogger(ctx)
      logger.Info("Step started", "step", step.Name)
      
      // Activity 配置 (含取消处理)
      activityOptions := workflow.ActivityOptions{
          StartToCloseTimeout: 5 * time.Minute,
          HeartbeatTimeout:    30 * time.Second,  // 心跳超时
          RetryPolicy: &temporal.RetryPolicy{
              MaximumAttempts: 3,
          },
          // 取消配置: 等待Activity完成清理
          WaitForCancellation: true,
          CancellationType:    enums.CANCEL_TYPE_WAIT_CANCELLATION_COMPLETED,
      }
      
      ctx = workflow.WithActivityOptions(ctx, activityOptions)
      
      // 执行 Activity (取消会传递到 Activity Context)
      input := StepExecutionInput{
          Name: step.Name,
          Uses: step.Uses,
      }
      
      err := workflow.ExecuteActivity(ctx, ExecuteStepActivity, input).Get(ctx, nil)
      if err != nil {
          // 检查是否为取消错误
          if temporal.IsCanceledError(err) {
              logger.Warn("Step canceled", "step", step.Name)
              return err
          }
          
          logger.Error("Step failed", "step", step.Name, "error", err)
          return err
      }
      
      logger.Info("Step completed", "step", step.Name)
      return nil
  }
  ```

- [ ] 4.2 更新 Activity 实现支持取消
  ```go
  // internal/temporal/activities.go
  
  // ExecuteStepActivity - 执行单个 Step (Mock 实现)
  func ExecuteStepActivity(ctx context.Context, input StepExecutionInput) error {
      logger := activity.GetLogger(ctx)
      logger.Info("Activity started", "step", input.Name)
      
      // 模拟长时间运行的操作
      // 分段检查取消信号
      for i := 0; i < 10; i++ {
          // 检查 Context 取消
          select {
          case <-ctx.Done():
              logger.Warn("Activity canceled", "step", input.Name)
              return ctx.Err() // 返回 context.Canceled
          default:
              // 继续执行
          }
          
          // 模拟工作
          time.Sleep(500 * time.Millisecond)
          
          // 发送心跳 (让 Temporal 知道 Activity 还活着)
          activity.RecordHeartbeat(ctx, i)
      }
      
      logger.Info("Activity completed", "step", input.Name)
      return nil
  }
  ```

### Task 5: 注册路由端点

- [ ] 5.1 更新 `internal/server/router.go`
  ```go
  func SetupRouter(logger *zap.Logger, tc *temporal.Client, workflowService *service.WorkflowService) *gin.Engine {
      router := gin.New()
      
      // ... 中间件 ...
      
      v1 := router.Group("/v1")
      {
          // 工作流端点
          workflowQueryService := service.NewWorkflowQueryService(tc, logger)
          workflowLogService := service.NewWorkflowLogService(tc, logger)
          workflowCancelService := service.NewWorkflowCancelService(tc, workflowQueryService, logger)
          
          workflowHandler := handlers.NewWorkflowHandler(
              workflowService,
              workflowQueryService,
              workflowLogService,
              workflowCancelService, // 新增
              logger,
          )
          
          v1.POST("/workflows", workflowHandler.SubmitWorkflow)
          v1.GET("/workflows/:id", workflowHandler.GetWorkflow)
          v1.GET("/workflows/:id/logs", workflowHandler.GetLogs)
          v1.POST("/workflows/:id/cancel", workflowHandler.CancelWorkflow) // 新增
      }
      
      return router
  }
  ```

- [ ] 5.2 更新 Handler 构造函数
  ```go
  // internal/server/handlers/workflow.go
  
  type WorkflowHandler struct {
      workflowService       *service.WorkflowService
      workflowQueryService  *service.WorkflowQueryService
      workflowLogService    *service.WorkflowLogService
      workflowCancelService *service.WorkflowCancelService // 新增
      logger                *zap.Logger
  }
  
  func NewWorkflowHandler(
      ws *service.WorkflowService,
      wqs *service.WorkflowQueryService,
      wls *service.WorkflowLogService,
      wcs *service.WorkflowCancelService, // 新增
      logger *zap.Logger,
  ) *WorkflowHandler {
      return &WorkflowHandler{
          workflowService:       ws,
          workflowQueryService:  wqs,
          workflowLogService:    wls,
          workflowCancelService: wcs,
          logger:                logger,
      }
  }
  ```

### Task 6: 添加单元测试

- [ ] 6.1 创建 `internal/service/workflow_cancel_service_test.go`
  ```go
  package service
  
  import (
      "context"
      "testing"
      
      "github.com/stretchr/testify/assert"
      "github.com/stretchr/testify/mock"
      "go.uber.org/zap"
  )
  
  func TestCancelWorkflow_Success(t *testing.T) {
      mockClient := &MockTemporalClient{}
      mockQueryService := &MockWorkflowQueryService{}
      
      // Mock 状态查询返回 running
      mockQueryService.On("GetWorkflowStatus", mock.Anything, "wf-123").Return(
          &models.WorkflowStatusResponse{
              WorkflowID: "wf-123",
              RunID:      "run-456",
              Status:     "running",
          }, nil,
      )
      
      // Mock 取消成功
      mockClient.On("CancelWorkflow", mock.Anything, "wf-123", "run-456").Return(nil)
      
      wcs := NewWorkflowCancelService(mockClient, mockQueryService, zap.NewNop())
      
      response, err := wcs.CancelWorkflow(context.Background(), "wf-123")
      
      assert.NoError(t, err)
      assert.Equal(t, "wf-123", response.WorkflowID)
      assert.Equal(t, "canceling", response.Status)
      
      mockClient.AssertExpectations(t)
  }
  
  func TestCancelWorkflow_AlreadyCompleted(t *testing.T) {
      mockQueryService := &MockWorkflowQueryService{}
      
      // Mock 状态查询返回 completed
      mockQueryService.On("GetWorkflowStatus", mock.Anything, "wf-123").Return(
          &models.WorkflowStatusResponse{
              WorkflowID: "wf-123",
              Status:     "completed",
          }, nil,
      )
      
      wcs := NewWorkflowCancelService(nil, mockQueryService, zap.NewNop())
      
      _, err := wcs.CancelWorkflow(context.Background(), "wf-123")
      
      assert.Error(t, err)
      assert.IsType(t, &CancelNotAllowedError{}, err)
      
      cancelErr := err.(*CancelNotAllowedError)
      assert.Equal(t, "completed", cancelErr.CurrentStatus)
  }
  
  func TestCancelWorkflow_NotFound(t *testing.T) {
      mockQueryService := &MockWorkflowQueryService{}
      
      // Mock 状态查询返回 not found
      mockQueryService.On("GetWorkflowStatus", mock.Anything, "wf-nonexist").Return(
          nil, errors.New("workflow not found"),
      )
      
      wcs := NewWorkflowCancelService(nil, mockQueryService, zap.NewNop())
      
      _, err := wcs.CancelWorkflow(context.Background(), "wf-nonexist")
      
      assert.Error(t, err)
      assert.Contains(t, err.Error(), "not found")
  }
  
  func TestIsCancelable(t *testing.T) {
      wcs := &WorkflowCancelService{}
      
      tests := []struct {
          status     string
          cancelable bool
      }{
          {"running", true},
          {"completed", false},
          {"failed", false},
          {"canceled", false},
          {"timeout", false},
      }
      
      for _, tt := range tests {
          result := wcs.isCancelable(tt.status)
          assert.Equal(t, tt.cancelable, result, "status: %s", tt.status)
      }
  }
  ```

- [ ] 6.2 测试 Handler
  ```go
  // internal/server/handlers/workflow_test.go
  
  func TestCancelWorkflow_Success(t *testing.T) {
      gin.SetMode(gin.TestMode)
      
      mockCancelService := &MockWorkflowCancelService{}
      mockCancelService.On("CancelWorkflow", mock.Anything, "wf-123").Return(
          &models.CancelWorkflowResponse{
              WorkflowID: "wf-123",
              Status:     "canceling",
              Message:    "Workflow cancellation requested",
          }, nil,
      )
      
      handler := NewWorkflowHandler(nil, nil, nil, mockCancelService, zap.NewNop())
      
      router := gin.New()
      router.POST("/workflows/:id/cancel", handler.CancelWorkflow)
      
      req := httptest.NewRequest("POST", "/workflows/wf-123/cancel", nil)
      w := httptest.NewRecorder()
      router.ServeHTTP(w, req)
      
      assert.Equal(t, http.StatusAccepted, w.Code)
      
      var resp models.CancelWorkflowResponse
      json.Unmarshal(w.Body.Bytes(), &resp)
      assert.Equal(t, "wf-123", resp.WorkflowID)
      assert.Equal(t, "canceling", resp.Status)
  }
  
  func TestCancelWorkflow_Conflict(t *testing.T) {
      mockCancelService := &MockWorkflowCancelService{}
      mockCancelService.On("CancelWorkflow", mock.Anything, "wf-123").Return(
          nil, &service.CancelNotAllowedError{
              WorkflowID:    "wf-123",
              CurrentStatus: "completed",
          },
      )
      
      handler := NewWorkflowHandler(nil, nil, nil, mockCancelService, zap.NewNop())
      
      router := gin.New()
      router.POST("/workflows/:id/cancel", handler.CancelWorkflow)
      
      req := httptest.NewRequest("POST", "/workflows/wf-123/cancel", nil)
      w := httptest.NewRecorder()
      router.ServeHTTP(w, req)
      
      assert.Equal(t, http.StatusConflict, w.Code)
      
      var errorResp models.ConflictError
      json.Unmarshal(w.Body.Bytes(), &errorResp)
      assert.Equal(t, 409, errorResp.Status)
      assert.Equal(t, "completed", errorResp.CurrentStatus)
  }
  ```

- [ ] 6.3 运行测试
  ```bash
  go test -v ./internal/service -run TestCancelWorkflow
  go test -v ./internal/server/handlers -run TestCancelWorkflow
  ```

### Task 7: 集成测试

- [ ] 7.1 创建集成测试脚本
  ```bash
  # test/integration/test_workflow_cancel.sh
  #!/bin/bash
  
  set -e
  
  echo "=== Workflow Cancel API Integration Test ==="
  
  # 1. 启动环境
  make dev-env
  go run ./cmd/server &
  SERVER_PID=$!
  sleep 3
  
  # 2. 提交长时间运行的工作流
  echo "Submitting long-running workflow..."
  WORKFLOW_ID=$(curl -s -X POST http://localhost:8080/v1/workflows \
    -H "Content-Type: application/json" \
    -d '{
      "workflow": "name: Long Running\non: push\njobs:\n  build:\n    runs-on: linux\n    steps:\n      - name: LongStep\n        uses: sleep@v1"
    }' | jq -r '.workflow_id')
  
  echo "Workflow ID: $WORKFLOW_ID"
  
  # 3. 等待工作流开始执行
  sleep 2
  
  # 4. 验证状态为 running
  STATUS=$(curl -s http://localhost:8080/v1/workflows/$WORKFLOW_ID | jq -r '.status')
  if [ "$STATUS" != "running" ]; then
      echo "❌ Expected status 'running', got '$STATUS'"
      exit 1
  fi
  echo "✅ Workflow is running"
  
  # 5. 发送取消请求
  echo "Canceling workflow..."
  CANCEL_RESPONSE=$(curl -s -X POST http://localhost:8080/v1/workflows/$WORKFLOW_ID/cancel)
  echo "Cancel response: $CANCEL_RESPONSE"
  
  # 验证返回 202
  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:8080/v1/workflows/$WORKFLOW_ID/cancel)
  if [ "$HTTP_CODE" = "202" ]; then
      echo "✅ Received 202 Accepted"
  else
      echo "❌ Expected 202, got $HTTP_CODE"
      exit 1
  fi
  
  # 6. 等待取消生效
  sleep 3
  
  # 7. 验证最终状态为 canceled
  FINAL_STATUS=$(curl -s http://localhost:8080/v1/workflows/$WORKFLOW_ID | jq -r '.status')
  if [ "$FINAL_STATUS" = "canceled" ]; then
      echo "✅ Workflow canceled successfully"
  else
      echo "⚠️  Final status: $FINAL_STATUS (取消可能仍在进行中)"
  fi
  
  # 8. 测试重复取消 (应返回 409)
  echo "Testing duplicate cancel..."
  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:8080/v1/workflows/$WORKFLOW_ID/cancel)
  if [ "$HTTP_CODE" = "409" ]; then
      echo "✅ Duplicate cancel returned 409 Conflict"
  else
      echo "❌ Expected 409, got $HTTP_CODE"
      exit 1
  fi
  
  # 9. 测试取消不存在的工作流 (404)
  echo "Testing cancel non-existent workflow..."
  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:8080/v1/workflows/wf-nonexist/cancel)
  if [ "$HTTP_CODE" = "404" ]; then
      echo "✅ Non-existent workflow returned 404"
  else
      echo "❌ Expected 404, got $HTTP_CODE"
      exit 1
  fi
  
  # 清理
  kill $SERVER_PID
  make dev-env-stop
  
  echo "✅ Integration test completed"
  ```

- [ ] 7.2 测试取消传播
  ```bash
  # test/integration/test_cancel_propagation.sh
  #!/bin/bash
  
  # 验证取消信号是否正确传递到 Workflow 和 Activity
  
  set -e
  
  echo "=== Cancel Propagation Test ==="
  
  # 1. 提交工作流
  WORKFLOW_ID=$(curl -s -X POST http://localhost:8080/v1/workflows \
    -d @test/fixtures/multi_step_workflow.yaml | jq -r '.workflow_id')
  
  sleep 1
  
  # 2. 取消
  curl -s -X POST http://localhost:8080/v1/workflows/$WORKFLOW_ID/cancel
  
  sleep 2
  
  # 3. 检查日志中是否有取消记录
  LOGS=$(curl -s http://localhost:8080/v1/workflows/$WORKFLOW_ID/logs)
  
  # 应包含 "Workflow canceled" 或 "Step canceled"
  if echo "$LOGS" | jq -r '.logs[].message' | grep -q "canceled"; then
      echo "✅ Cancel propagation verified in logs"
  else
      echo "❌ No cancel message found in logs"
      exit 1
  fi
  ```

### Task 8: 更新 OpenAPI 文档

- [ ] 8.1 更新 `api/openapi.yaml`
  ```yaml
  paths:
    /v1/workflows/{id}/cancel:
      post:
        summary: Cancel a running workflow
        parameters:
          - name: id
            in: path
            required: true
            schema:
              type: string
            example: wf-550e8400-e29b-41d4-a716-446655440000
        responses:
          '202':
            description: Cancellation request accepted
            content:
              application/json:
                schema:
                  $ref: '#/components/schemas/CancelWorkflowResponse'
          '404':
            description: Workflow not found
            content:
              application/json:
                schema:
                  $ref: '#/components/schemas/ErrorResponse'
          '409':
            description: Workflow cannot be canceled in current state
            content:
              application/json:
                schema:
                  $ref: '#/components/schemas/ConflictError'
          '500':
            description: Internal server error
  
  components:
    schemas:
      CancelWorkflowResponse:
        type: object
        properties:
          workflow_id:
            type: string
            example: wf-550e8400-e29b-41d4-a716-446655440000
          status:
            type: string
            enum: [canceling]
            example: canceling
          message:
            type: string
            example: Workflow cancellation requested
      
      ConflictError:
        type: object
        properties:
          type:
            type: string
            example: https://waterflow.io/errors/conflict
          title:
            type: string
            example: Cannot Cancel Workflow
          status:
            type: integer
            example: 409
          detail:
            type: string
            example: "Workflow 'wf-xxx' cannot be canceled in current state"
          current_status:
            type: string
            example: completed
  ```

## Dev Notes

### Critical Implementation Guidelines

**1. 状态验证 - 先查后取**

```go
// ✅ 正确: 先查询状态再取消
status, _ := queryService.GetWorkflowStatus(ctx, workflowID)
if !isCancelable(status.Status) {
    return ConflictError
}
client.CancelWorkflow(ctx, workflowID, runID)

// ❌ 错误: 直接取消不验证
client.CancelWorkflow(ctx, workflowID, "") // 可能取消已完成的Workflow
```

**2. 异步响应 - 立即返回 202**

```go
// ✅ 正确: 取消是异步操作
client.CancelWorkflow(ctx, workflowID, runID)
c.JSON(202, CancelResponse{Status: "canceling"})

// ❌ 错误: 等待取消完成
client.CancelWorkflow(ctx, workflowID, runID)
for {
    status := getStatus()
    if status == "canceled" {
        break
    }
}
c.JSON(200, ...) // 可能长时间阻塞
```

**3. Workflow 取消检查 - 每个 Step 前检查**

```go
// ✅ 正确: 每个 Step 执行前检查
for _, step := range steps {
    if ctx.Err() != nil {
        return ctx.Err() // 立即返回
    }
    executeStep(ctx, step)
}

// ❌ 错误: 不检查取消,继续执行所有 Step
for _, step := range steps {
    executeStep(ctx, step) // 取消信号被忽略
}
```

**4. Activity 心跳 - 支持长时间运行**

```go
// ✅ 正确: 定期发送心跳
for i := 0; i < 100; i++ {
    activity.RecordHeartbeat(ctx, i)
    doWork()
}

// ❌ 错误: 无心跳,Temporal 无法检测超时
for i := 0; i < 100; i++ {
    doWork() // 长时间运行无心跳
}
```

**5. 错误类型判断 - 区分取消和失败**

```go
// ✅ 正确: 区分取消错误
if temporal.IsCanceledError(err) {
    logger.Warn("Workflow canceled")
    return err
}
logger.Error("Workflow failed")

// ❌ 错误: 统一处理
if err != nil {
    logger.Error("Workflow failed") // 取消也记录为失败
}
```

**6. RunID 传递 - 避免取消错误的 Run**

```go
// ✅ 正确: 使用最新的 RunID
status, _ := getStatus(workflowID)
client.CancelWorkflow(ctx, workflowID, status.RunID)

// ❌ 错误: RunID 为空可能取消错误的 Run
client.CancelWorkflow(ctx, workflowID, "") // Temporal 会选择最新 Run,但不安全
```

### Integration with Previous Stories

**与 Story 1.7 状态查询集成:**

```go
// Story 1.7 提供状态查询
status, _ := workflowQueryService.GetWorkflowStatus(ctx, workflowID)

// Story 1.9 使用状态判断是否可取消
if status.Status != "running" {
    return ConflictError
}
```

**与 Story 1.6 Workflow 执行集成:**

```go
// Story 1.6 实现的 Workflow
func WaterflowWorkflow(ctx workflow.Context, def *WorkflowDefinition) error {
    for _, step := range steps {
        // Story 1.9 添加: 检查取消
        if ctx.Err() != nil {
            return ctx.Err()
        }
        executeStep(ctx, step)
    }
}
```

**与 Story 1.8 日志输出集成:**

```go
// Story 1.9 取消后,Story 1.8 可查看取消日志
GET /v1/workflows/wf-123/logs

// 日志包含:
{
  "level": "warn",
  "type": "workflow.canceled",
  "message": "Workflow was canceled"
}
```

**为 Story 2.x 准备 (Agent 取消):**

```go
// 未来 Agent 需要处理 Activity 取消
func ExecuteStepActivity(ctx context.Context, input StepInput) error {
    // 启动 Agent 执行
    cmd := exec.CommandContext(ctx, "agent", "run", input.Command)
    
    // ctx.Done() 会在取消时触发
    // exec.CommandContext 会自动终止进程
    err := cmd.Run()
    
    if ctx.Err() != nil {
        return ctx.Err() // 返回取消错误
    }
    return err
}
```

### Testing Strategy

**单元测试覆盖:**

| 组件 | 测试场景 |
|------|---------|
| WorkflowCancelService | 成功取消、状态冲突、404 |
| isCancelable | 所有状态枚举 |
| CancelWorkflow Handler | 202/404/409 响应 |
| Workflow 取消检查 | ctx.Err() 检测 |

**集成测试:**

```bash
# 1. 提交工作流
WORKFLOW_ID=$(curl -X POST /v1/workflows -d @workflow.json | jq -r '.workflow_id')

# 2. 验证 running
curl /v1/workflows/$WORKFLOW_ID | jq -r '.status'
# 期望: running

# 3. 取消
curl -X POST /v1/workflows/$WORKFLOW_ID/cancel
# 期望: 202 Accepted

# 4. 验证最终状态
sleep 3
curl /v1/workflows/$WORKFLOW_ID | jq -r '.status'
# 期望: canceled

# 5. 重复取消
curl -X POST /v1/workflows/$WORKFLOW_ID/cancel
# 期望: 409 Conflict
```

**取消传播测试:**

```go
// 测试取消信号是否传递到 Workflow 和 Activity
func TestCancelPropagation(t *testing.T) {
    // 启动 Workflow
    workflowRun := client.ExecuteWorkflow(ctx, workflowID, WaterflowWorkflow, def)
    
    // 等待开始执行
    time.Sleep(1 * time.Second)
    
    // 取消
    client.CancelWorkflow(ctx, workflowID, workflowRun.GetRunID())
    
    // 等待 Workflow 结束
    err := workflowRun.Get(ctx, nil)
    
    // 验证是取消错误
    assert.True(t, temporal.IsCanceledError(err))
}
```

### Performance Considerations

**1. 取消响应时间**

```go
// 取消操作应立即返回,不等待 Workflow 完成
err := client.CancelWorkflow(ctx, workflowID, runID)
// 立即返回 202,不等待取消完成
```

**2. 状态查询性能**

```go
// 复用 Story 1.7 的状态查询逻辑
// 如果状态已缓存,查询非常快速
status, _ := queryService.GetWorkflowStatus(ctx, workflowID)
```

**3. Activity 优雅停止时间**

```go
// Activity 应快速响应取消 (< HeartbeatTimeout)
activityOptions := workflow.ActivityOptions{
    StartToCloseTimeout:    5 * time.Minute,
    HeartbeatTimeout:       30 * time.Second,  // 心跳超时
    WaitForCancellation:    true,              // 等待Activity完成取消清理
    CancellationType:       enums.CANCEL_TYPE_WAIT_CANCELLATION_COMPLETED,
}
```

**优雅停止配置说明:**

| 选项 | 值 | 作用 |
|------|------|------|
| WaitForCancellation | true | 等待Activity完成清理工作 |
| CancellationType | WAIT_CANCELLATION_COMPLETED | 不放弃Activity,等待完成 |
| HeartbeatTimeout | 30s | Activity清理超时限制 |

**超时行为:**
- Activity在收到取消后有最多`HeartbeatTimeout`(30秒)完成清理
- 超过超时,Temporal强制终止Activity
- 确保取消操作不会无限挂起

**Activity最佳实践:**
```go
func ExecuteStepActivity(ctx context.Context, input StepInput) error {
    // 确保清理总是执行
    defer cleanup()
    
    for {
        select {
        case <-ctx.Done():
            logger.Info("Cancellation received, cleaning up...")
            // 执行清理逻辑 (必须在HeartbeatTimeout内完成)
            return ctx.Err()
        default:
            doWork()
            activity.RecordHeartbeat(ctx, progress)
        }
    }
}
```

### Edge Cases

**1. 取消正在启动的 Workflow**

```go
// Workflow 刚提交,还未真正开始执行
// CancelWorkflow 仍然有效,Workflow 会在启动时检测到取消
```

**2. 取消已在取消中的 Workflow**

```go
// 重复取消同一个 Workflow
// Temporal 会忽略重复的取消请求,不报错
// 但我们的 API 在查询状态时会返回 409
```

**3. Workflow 正好完成时取消**

```go
// 存在竞态条件: 查询时是 running,取消时已 completed
// Temporal 会忽略对已完成 Workflow 的取消,不报错
// 我们的状态查询会最终显示 completed
```

### References

**架构设计:**
- [docs/architecture.md §3.1.1](docs/architecture.md) - REST API Handler 设计
- [docs/architecture.md §3.1.5](docs/architecture.md) - Temporal Client 取消功能

**技术文档:**
- [Temporal CancelWorkflow](https://pkg.go.dev/go.temporal.io/sdk/client#Client.CancelWorkflow)
- [Temporal Workflow Cancellation](https://docs.temporal.io/workflows#cancellation)
- [Temporal Activity Heartbeat](https://docs.temporal.io/activities#heartbeat)

**项目上下文:**
- [docs/epics.md Story 1.6](docs/epics.md) - Workflow 执行引擎
- [docs/epics.md Story 1.7](docs/epics.md) - 状态查询 (状态验证)

### Dependency Graph

```
Story 1.4 (Temporal Client) ──┐
Story 1.6 (执行引擎)         ──┤
Story 1.7 (状态查询)         ──┤
                              ↓
Story 1.9 (取消 API) ← 当前 Story
    ↓
    └→ Story 2.x (Agent 取消) - Agent 需处理 Activity 取消
```

## Dev Agent Record

### Context Reference

**Source Documents Analyzed:**
1. [docs/epics.md](docs/epics.md) (lines 410-428) - Story 1.9 需求定义
2. [docs/architecture.md](docs/architecture.md) (§3.1.1, §3.1.5) - REST API Handler, Temporal Client 设计

**Previous Stories:**
- Story 1.1-1.8: 全部 drafted (框架、API、解析、Temporal、提交、执行、状态查询、日志)

### Agent Model Used

Claude 3.5 Sonnet (BMM Scrum Master Agent - Bob)

### Estimated Effort

**开发时间:** 5-7 小时  
**复杂度:** 中等

**时间分解:**
- 取消响应模型: 0.5 小时
- 取消服务层实现: 1.5 小时
- HTTP Handler: 1 小时
- Workflow 取消处理增强: 1.5 小时
- 单元测试: 1 小时
- 集成测试: 1 小时
- OpenAPI 文档: 0.5 小时

**技能要求:**
- Temporal CancelWorkflow API
- Workflow Context 取消处理
- Activity 心跳机制
- HTTP 状态码设计 (202/409)

### Debug Log References

<!-- Will be populated during implementation -->

### Completion Notes List

<!-- Developer 填写完成时的笔记 -->

### File List

**预期创建/修改的文件清单:**

```
新建文件 (~3 个):
├── internal/models/
│   └── workflow_cancel.go                # 取消响应模型
├── internal/service/
│   ├── workflow_cancel_service.go        # 取消服务层
│   └── workflow_cancel_service_test.go   # 单元测试
├── test/integration/
│   ├── test_workflow_cancel.sh           # 集成测试
│   └── test_cancel_propagation.sh        # 取消传播测试

修改文件 (~4 个):
├── internal/temporal/workflow.go             # 增强取消检查
├── internal/temporal/activities.go           # Activity 心跳和取消处理
├── internal/server/handlers/workflow.go      # 添加 CancelWorkflow 方法
├── internal/server/router.go                 # 注册 POST /v1/workflows/:id/cancel
└── api/openapi.yaml                          # 添加取消端点文档
```

**关键代码片段:**

**workflow_cancel_service.go (核心):**
```go
func (wcs *WorkflowCancelService) CancelWorkflow(ctx context.Context, workflowID string) (*CancelWorkflowResponse, error) {
    // 1. 查询状态
    status, _ := wcs.queryService.GetWorkflowStatus(ctx, workflowID)
    
    // 2. 验证可取消
    if !wcs.isCancelable(status.Status) {
        return nil, &CancelNotAllowedError{CurrentStatus: status.Status}
    }
    
    // 3. 发送取消信号
    wcs.temporalClient.GetClient().CancelWorkflow(ctx, workflowID, status.RunID)
    
    // 4. 返回 202
    return &CancelWorkflowResponse{Status: "canceling"}, nil
}
```

**workflow.go (增强):**
```go
func WaterflowWorkflow(ctx workflow.Context, def *WorkflowDefinition) error {
    for _, step := range steps {
        // 检查取消
        if ctx.Err() != nil {
            return ctx.Err()
        }
        executeStep(ctx, step)
    }
}
```

**Handler:**
```go
func (h *WorkflowHandler) CancelWorkflow(c *gin.Context) {
    response, err := h.workflowCancelService.CancelWorkflow(ctx, workflowID)
    
    if cancelErr, ok := err.(*CancelNotAllowedError); ok {
        c.JSON(409, ConflictError)
        return
    }
    
    c.JSON(202, response)
}
```

---

**Story Ready for Development** ✅

开发者可基于 Story 1.1-1.8 的成果,实现工作流取消功能。
本 Story 完成后,用户可以取消正在运行的工作流,释放资源。
