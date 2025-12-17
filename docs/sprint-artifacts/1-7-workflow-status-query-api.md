# Story 1.7: 工作流状态查询 API

Status: ready-for-dev

## Story

As a **工作流用户**,  
I want **查询工作流的执行状态**,  
So that **了解工作流进度和结果**。

## Acceptance Criteria

**Given** 工作流已提交并执行  
**When** GET `/v1/workflows/{id}` 查询工作流  
**Then** 返回工作流状态 (running, completed, failed)  
**And** 返回执行进度 (当前 Job/Step)  
**And** 返回开始时间和持续时间  
**And** 工作流不存在返回 404  
**And** 响应时间 <200ms

## Technical Context

### Architecture Constraints

根据 [docs/architecture.md](docs/architecture.md) §3.1.1 REST API Handler设计:

1. **核心职责**
   - 处理 `GET /v1/workflows/{id}` 请求
   - 通过 Temporal Client 查询工作流状态
   - 从 Temporal Event History 获取执行信息
   - 返回结构化的状态数据 (JSON)

2. **Event Sourcing 架构** (参考 ADR-0001)
   - 所有状态从 Temporal Event History 查询,确保数据一致性
   - 支持时间旅行查询:可查看历史任意时刻的状态
   - Server 无状态,直接从 Temporal 获取最新状态

3. **功能需求映射**
   - **FR7 实时状态跟踪和日志**: 本 Story 实现状态查询 API
   - **FR3 工作流管理 API**: GET /v1/workflows/{id} 端点

2. **响应格式**

**成功响应 (200 OK):**
```json
{
  "workflow_id": "wf-550e8400-e29b-41d4-a716-446655440000",
  "run_id": "temporal-generated-uuid",
  "name": "Deploy Application",
  "status": "running",
  "start_time": "2025-12-16T10:30:00Z",
  "close_time": null,
  "duration": 125000,
  "progress": {
    "current_job": "build",
    "current_step": "Build",
    "total_steps": 3,
    "completed_steps": 1
  },
  "result": null
}
```

**工作流完成:**
```json
{
  "workflow_id": "wf-xxx",
  "status": "completed",
  "close_time": "2025-12-16T10:32:05Z",
  "duration": 125000,
  "result": {
    "success": true,
    "outputs": {}
  }
}
```

**工作流失败:**
```json
{
  "workflow_id": "wf-xxx",
  "status": "failed",
  "close_time": "2025-12-16T10:31:30Z",
  "error": {
    "type": "StepExecutionError",
    "message": "Step 'Build' failed: exit code 1",
    "failed_step": "Build"
  }
}
```

**404响应:**
```json
{
  "type": "https://waterflow.io/errors/not-found",
  "title": "Workflow Not Found",
  "status": 404,
  "detail": "Workflow with ID 'wf-xxx' does not exist"
}
```

3. **性能要求**
   - 查询响应时间 p95 < 200ms
   - 支持并发查询 (100+ req/s)
   - 缓存Workflow基本信息 (可选优化)

### Dependencies

**前置Story:**
- ✅ Story 1.1: Waterflow Server框架搭建
- ✅ Story 1.2: REST API服务框架
- ✅ Story 1.4: Temporal SDK集成
  - 使用: `DescribeWorkflowExecution` API
- ✅ Story 1.5: 工作流提交API
  - 使用: WorkflowID生成和存储
- ✅ Story 1.6: 基础工作流执行引擎
  - 使用: Workflow执行产生的Event History

**后续Story依赖本Story:**
- Story 1.8: 日志输出 - 复用状态查询逻辑
- Story 1.9: 取消API - 需要先查询状态判断是否可取消

### Technology Stack

**Temporal Client API:**

```go
import (
    "go.temporal.io/sdk/client"
    "go.temporal.io/api/enums/v1"
)

// 1. DescribeWorkflowExecution - 获取Workflow基本信息
describe, err := client.DescribeWorkflowExecution(ctx, workflowID, runID)

// 2. 获取状态
status := describe.WorkflowExecutionInfo.Status
// 枚举值:
// - enums.WORKFLOW_EXECUTION_STATUS_RUNNING
// - enums.WORKFLOW_EXECUTION_STATUS_COMPLETED
// - enums.WORKFLOW_EXECUTION_STATUS_FAILED
// - enums.WORKFLOW_EXECUTION_STATUS_CANCELED
// - enums.WORKFLOW_EXECUTION_STATUS_TERMINATED

// 3. 获取时间信息
startTime := describe.WorkflowExecutionInfo.StartTime
closeTime := describe.WorkflowExecutionInfo.CloseTime

// 4. GetWorkflowHistory - 获取Event History (进度分析)
historyIter := client.GetWorkflowHistory(ctx, workflowID, runID, false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)

for historyIter.HasNext() {
    event, err := historyIter.Next()
    // 分析事件类型
    switch event.EventType {
    case enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED:
        // Step开始
    case enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
        // Step完成
    }
}
```

**状态映射:**

| Temporal Status | Waterflow Status | 说明 |
|----------------|------------------|------|
| RUNNING | running | 正在执行 |
| COMPLETED | completed | 成功完成 |
| FAILED | failed | 执行失败 |
| CANCELED | canceled | 用户取消 |
| TERMINATED | terminated | 强制终止 |
| TIMED_OUT | timeout | 超时 |

### Progress Calculation Strategy

**从Event History提取进度:**

```go
type ProgressInfo struct {
    CurrentJob      string `json:"current_job"`
    CurrentStep     string `json:"current_step"`
    TotalSteps      int    `json:"total_steps"`
    CompletedSteps  int    `json:"completed_steps"`
}

func calculateProgress(history []*historypb.HistoryEvent) *ProgressInfo {
    var completedSteps int
    var currentStep string
    
    for _, event := range history {
        switch event.EventType {
        case enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
            completedSteps++
        
        case enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED:
            // 从Activity名称提取Step信息
            attrs := event.GetActivityTaskScheduledEventAttributes()
            currentStep = attrs.ActivityType.Name
        }
    }
    
    return &ProgressInfo{
        CurrentJob:      "build", // MVP: 单Job
        CurrentStep:     currentStep,
        CompletedSteps:  completedSteps,
        TotalSteps:      wqs.getTotalSteps(ctx, workflowID), // 见辅助方法
    }
}

// getTotalSteps 获取工作流总步数
func (wqs *WorkflowQueryService) getTotalSteps(ctx context.Context, workflowID string) int {
    // 方法1: 查询Workflow Input (WorkflowDefinition) - 最准确
    // 方法2: 从Event History遍历ActivityTaskScheduled事件 - 适用于运行中
    // 方法3: 缓存提交时的步数 - 最快但需额外存储
    
    // MVP实现: 返回固定值或从缓存获取
    // TODO: Story后续优化 - 从Workflow Input解析
    return 3 // 临时返回
}
```

### Project Structure Updates

基于Story 1.1-1.6的结构,本Story新增:

```
internal/
├── service/
│   ├── workflow_query_service.go      # 查询服务层 (新建)
│   └── workflow_query_service_test.go (新建)
├── server/handlers/
│   └── workflow.go                    # 修改 - 添加GetWorkflow方法
├── models/
│   └── workflow_status.go             # 状态响应模型 (新建)

api/
└── openapi.yaml                       # 更新 - 添加GET /v1/workflows/{id}
```

## Tasks / Subtasks

### Task 0: 验证依赖 (AC: 开发环境就绪)

- [ ] 0.1 确认依赖已安装 (Story 1.1-1.6)
  ```bash
  # 验证Temporal SDK (Story 1.4)
  go list -m go.temporal.io/sdk
  # 期望输出: go.temporal.io/sdk v1.25.0
  
  # 本Story无新增依赖,复用现有Temporal Client API
  # 主要开发工作: 实现查询服务层、HTTP Handler、响应模型
  ```

- [ ] 0.2 验证前置Story (1.1-1.6) 产出文件存在
  ```bash
  #!/bin/bash
  # test/verify-dependencies-story-1-7.sh
  
  echo "=== Verifying Story 1.1-1.6 Dependencies for Story 1-7 ==="
  
  # Story 1.1-1.2: Server框架和REST API
  test -f cmd/server/main.go || { echo "ERROR: Story 1.1未完成 - 缺少cmd/server/main.go"; exit 1; }
  test -f internal/server/server.go || { echo "ERROR: Story 1.2未完成 - 缺少internal/server/server.go"; exit 1; }
  test -f internal/server/router.go || { echo "ERROR: Story 1.2未完成 - 缺少router.go"; exit 1; }
  
  # Story 1.4: Temporal集成
  test -f internal/temporal/client.go || { echo "ERROR: Story 1.4未完成 - 缺少temporal/client.go"; exit 1; }
  go list -m go.temporal.io/sdk > /dev/null 2>&1 || { echo "ERROR: Temporal SDK未安装"; exit 1; }
  
  # Story 1.5: 工作流提交API
  test -f internal/service/workflow_service.go || { echo "ERROR: Story 1.5未完成 - 缺少workflow_service.go"; exit 1; }
  test -f internal/models/request.go || { echo "ERROR: Story 1.5未完成 - 缺少models/request.go"; exit 1; }
  test -f internal/server/handlers/workflow.go || { echo "ERROR: Story 1.5未完成 - 缺少handlers/workflow.go"; exit 1; }
  
  # Story 1.6: 工作流执行引擎
  test -f internal/workflow/waterflow_workflow.go || { echo "ERROR: Story 1.6未完成 - 缺少waterflow_workflow.go"; exit 1; }
  test -f internal/workflow/activities.go || { echo "ERROR: Story 1.6未完成 - 缺少activities.go"; exit 1; }
  test -f internal/workflow/worker.go || { echo "ERROR: Story 1.6未完成 - 缺少worker.go"; exit 1; }
  
  echo "✅ All dependencies verified - Story 1.7 can proceed"
  ```

- [ ] 0.3 确认Temporal Server运行中
  ```bash
  curl http://localhost:7233/health
  # 期望: 200 OK
  ```

- [ ] 0.4 开发前运行验证脚本
  ```bash
  chmod +x test/verify-dependencies-story-1-7.sh
  ./test/verify-dependencies-story-1-7.sh
  
  # 如果验证失败,请先完成对应的前置Story
  ```

### Task 1: 定义状态查询响应模型 (AC: 返回工作流状态)

- [ ] 1.1 创建`internal/models/workflow_status.go`
  ```go
  package models
  
  import "time"
  
  // WorkflowStatusResponse 工作流状态响应
  type WorkflowStatusResponse struct {
      WorkflowID string        `json:"workflow_id"`
      RunID      string        `json:"run_id"`
      Name       string        `json:"name"`
      Status     string        `json:"status"` // running, completed, failed, canceled
      StartTime  time.Time     `json:"start_time"`
      CloseTime  *time.Time    `json:"close_time,omitempty"`
      Duration   int64         `json:"duration"` // 毫秒
      Progress   *ProgressInfo `json:"progress,omitempty"`
      Result     *WorkflowResult `json:"result,omitempty"`
      Error      *WorkflowError  `json:"error,omitempty"`
  }
  
  // ProgressInfo 执行进度信息
  type ProgressInfo struct {
      CurrentJob     string `json:"current_job"`
      CurrentStep    string `json:"current_step,omitempty"`
      TotalSteps     int    `json:"total_steps"`
      CompletedSteps int    `json:"completed_steps"`
  }
  
  // WorkflowResult 执行结果
  type WorkflowResult struct {
      Success bool                   `json:"success"`
      Outputs map[string]interface{} `json:"outputs,omitempty"`
  }
  
  // WorkflowError 执行错误
  type WorkflowError struct {
      Type       string `json:"type"`
      Message    string `json:"message"`
      FailedStep string `json:"failed_step,omitempty"`
  }
  ```

### Task 2: 实现查询服务层 (AC: 响应时间<200ms)

- [ ] 2.1 创建`internal/service/workflow_query_service.go`
  ```go
  package service
  
  import (
      "context"
      "fmt"
      "time"
      
      "go.temporal.io/api/enums/v1"
      "go.temporal.io/sdk/client"
      "go.uber.org/zap"
      "golang.org/x/time/rate"  // 并发限流
      
      "waterflow/internal/models"
      "waterflow/internal/temporal"
  )
  
  type WorkflowQueryService struct {
      temporalClient *temporal.Client
      logger         *zap.Logger
      rateLimiter    *rate.Limiter  // 并发限流保护
  }
  
  func NewWorkflowQueryService(tc *temporal.Client, logger *zap.Logger) *WorkflowQueryService {
      return &WorkflowQueryService{
          temporalClient: tc,
          logger:         logger,
          rateLimiter:    rate.NewLimiter(100, 200), // 100 qps, burst 200
      }
  }
  
  // GetWorkflowStatus 查询工作流状态 (带并发限流保护)
  func (wqs *WorkflowQueryService) GetWorkflowStatus(ctx context.Context, workflowID string) (*models.WorkflowStatusResponse, error) {
      // 0. 并发限流检查 (防止高并发场景下Temporal过载)
      if err := wqs.rateLimiter.Wait(ctx); err != nil {
          wqs.logger.Warn("Rate limit exceeded",
              zap.String("workflow_id", workflowID),
              zap.Error(err),
          )
          return nil, fmt.Errorf("too many requests, please retry later: %w", err)
      }
      
      // 1. 查询Workflow基本信息
      describe, err := wqs.temporalClient.GetClient().DescribeWorkflowExecution(ctx, workflowID, "")
      if err != nil {
          wqs.logger.Error("Failed to describe workflow",
              zap.String("workflow_id", workflowID),
              zap.Error(err),
          )
          return nil, fmt.Errorf("workflow not found: %w", err)
      }
      
      info := describe.WorkflowExecutionInfo
      
      // 2. 映射状态
      status := wqs.mapStatus(info.Status)
      
      // 3. 计算持续时间
      duration := wqs.calculateDuration(info.StartTime, info.CloseTime)
      
      // 4. 构建响应
      response := &models.WorkflowStatusResponse{
          WorkflowID: info.Execution.WorkflowId,
          RunID:      info.Execution.RunId,
          Status:     status,
          StartTime:  *info.StartTime,
          Duration:   duration,
      }
      
      // 5. 设置关闭时间 (如果已完成)
      if info.CloseTime != nil {
          response.CloseTime = info.CloseTime
      }
      
      // 6. 获取进度信息 (仅running状态)
      if status == "running" {
          progress, err := wqs.getProgress(ctx, workflowID, info.Execution.RunId)
          if err != nil {
              wqs.logger.Warn("Failed to get progress",
                  zap.String("workflow_id", workflowID),
                  zap.Error(err),
              )
          } else {
              response.Progress = progress
          }
      }
      
      // 7. 获取结果或错误
      if status == "completed" {
          response.Result = &models.WorkflowResult{
              Success: true,
              Outputs: map[string]interface{}{}, // MVP暂不实现outputs
          }
      } else if status == "failed" {
          response.Error = wqs.extractError(info)
      }
      
      return response, nil
  }
  
  // mapStatus 映射Temporal状态到Waterflow状态
  func (wqs *WorkflowQueryService) mapStatus(status enums.WorkflowExecutionStatus) string {
      switch status {
      case enums.WORKFLOW_EXECUTION_STATUS_RUNNING:
          return "running"
      case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
          return "completed"
      case enums.WORKFLOW_EXECUTION_STATUS_FAILED:
          return "failed"
      case enums.WORKFLOW_EXECUTION_STATUS_CANCELED:
          return "canceled"
      case enums.WORKFLOW_EXECUTION_STATUS_TERMINATED:
          return "terminated"
      case enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
          return "timeout"
      default:
          return "unknown"
      }
  }
  
  // calculateDuration 计算执行时长 (毫秒)
  func (wqs *WorkflowQueryService) calculateDuration(startTime *time.Time, closeTime *time.Time) int64 {
      if startTime == nil {
          return 0
      }
      
      endTime := time.Now()
      if closeTime != nil {
          endTime = *closeTime
      }
      
      return endTime.Sub(*startTime).Milliseconds()
  }
  ```

- [ ] 2.2 实现进度提取逻辑
  ```go
  // getProgress 从Event History提取进度
  func (wqs *WorkflowQueryService) getProgress(ctx context.Context, workflowID, runID string) (*models.ProgressInfo, error) {
      // 获取Event History
      iter := wqs.temporalClient.GetClient().GetWorkflowHistory(
          ctx,
          workflowID,
          runID,
          false, // waitNewEvent
          enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
      )
      
      var completedSteps int
      var currentStep string
      
      for iter.HasNext() {
          event, err := iter.Next()
          if err != nil {
              return nil, err
          }
          
          // 统计完成的Steps (ActivityTaskCompleted事件)
          if event.EventType == enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED {
              completedSteps++
          }
          
          // 获取当前Step (最近的ActivityTaskStarted)
          if event.EventType == enums.EVENT_TYPE_ACTIVITY_TASK_STARTED {
              // 从ActivityID提取Step名称
              currentStep = event.GetActivityTaskStartedEventAttributes().GetActivityId()
          }
      }
      
      return &models.ProgressInfo{
          CurrentJob:      "build", // MVP: 单Job
          CurrentStep:     currentStep,
          CompletedSteps:  completedSteps,
          TotalSteps:      wqs.getTotalSteps(ctx, workflowID), // 见辅助方法
      }, nil
  }
  
  // getTotalSteps 获取工作流总步数 (优化实现)
  func (wqs *WorkflowQueryService) getTotalSteps(ctx context.Context, workflowID, runID string) int {
      // 方法1: 从SearchAttributes获取 (推荐,需Story 1.5配合)
      // 方法2: 从Event History统计 (回退方案)
      // 方法3: 返回固定值 (MVP临时方案)
      
      // 尝试从SearchAttributes获取 (如果Story 1.5已实现存储)
      describe, err := wqs.temporalClient.GetClient().DescribeWorkflowExecution(ctx, workflowID, runID)
      if err == nil {
          if searchAttrs := describe.WorkflowExecutionInfo.GetSearchAttributes(); searchAttrs != nil {
              if totalStepsPayload, ok := searchAttrs.GetIndexedFields()["TotalSteps"]; ok {
                  // 解析Payload获取总步数
                  var totalSteps int
                  if err := totalStepsPayload.Get(&totalSteps); err == nil {
                      return totalSteps
                  }
              }
          }
      }
      
      // 回退方案: 从Event History统计ActivityTaskScheduled事件
      count := wqs.countStepsFromHistory(ctx, workflowID, runID)
      if count > 0 {
          return count
      }
      
      // 最后回退: 返回固定值 (MVP临时方案)
      wqs.logger.Debug("Using fallback total steps", zap.String("workflow_id", workflowID))
      return 3
  }
  
  // countStepsFromHistory 从Event History统计总步数
  func (wqs *WorkflowQueryService) countStepsFromHistory(ctx context.Context, workflowID, runID string) int {
      iter := wqs.temporalClient.GetClient().GetWorkflowHistory(
          ctx,
          workflowID,
          runID,
          false, // waitNewEvent
          enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
      )
      
      count := 0
      maxEvents := 1000 // 限制遍历数量防止超时
      
      for iter.HasNext() && count < maxEvents {
          event, err := iter.Next()
          if err != nil {
              wqs.logger.Warn("Failed to iterate history for total steps", zap.Error(err))
              break
          }
          
          // 统计ActivityTaskScheduled事件
          if event.EventType == enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED {
              count++
          }
      }
      
      return count
  }
  
  // extractError 提取失败错误信息
  func (wqs *WorkflowQueryService) extractError(info *workflowpb.WorkflowExecutionInfo) *models.WorkflowError {
      // 从Failure字段提取错误
      if info.GetFailure() != nil {
          return &models.WorkflowError{
              Type:    "WorkflowExecutionError",
              Message: info.GetFailure().GetMessage(),
          }
      }
      
      return &models.WorkflowError{
          Type:    "UnknownError",
          Message: "Workflow failed with unknown error",
      }
  }
  ```

### Task 2.3: 实现getTotalSteps优化方案 (可选,提升进度精度)

- [ ] 2.3.1 实现多层回退策略 (已在Task 2.2中实现)
  ```go
  // 优先级:
  // 1. SearchAttributes (最准确,需Story 1.5配合)
  // 2. Event History统计 (回退方案)
  // 3. 固定值3 (MVP临时方案)
  
  // 代码已在Task 2.2的getTotalSteps方法中实现
  ```

- [ ] 2.3.2 (可选) 在Story 1.5中存储TotalSteps到SearchAttributes
  ```go
  // internal/service/workflow_service.go (Story 1.5修改)
  
  func (ws *WorkflowService) SubmitWorkflow(ctx context.Context, yamlContent string, idempotencyKey string) (*models.SubmitWorkflowResponse, error) {
      // 1. 解析YAML
      wf, err := ws.parser.Parse(yamlContent)
      if err != nil {
          return nil, fmt.Errorf("parse error: %w", err)
      }
      
      // 2. 计算总步数
      totalSteps := ws.calculateTotalSteps(wf)
      
      // 3. 生成WorkflowID
      workflowID := ws.GenerateWorkflowID()
      
      // 4. 提交到Temporal并存储总步数
      workflowOptions := client.StartWorkflowOptions{
          ID:                 workflowID,
          TaskQueue:          "default",
          WorkflowRunTimeout: 1 * time.Hour,
          // 存储总步数到SearchAttributes (供Story 1.7查询使用)
          SearchAttributes: map[string]interface{}{
              "TotalSteps": totalSteps,
          },
      }
      
      we, err := ws.temporalClient.GetClient().ExecuteWorkflow(
          ctx,
          workflowOptions,
          "WaterflowWorkflow",
          wf,
      )
      // ... 原有逻辑 ...
  }
  
  // calculateTotalSteps 计算工作流总步数
  func (ws *WorkflowService) calculateTotalSteps(wf *parser.WorkflowDefinition) int {
      total := 0
      for _, job := range wf.Jobs {
          total += len(job.Steps)
      }
      return total
  }
  ```

- [ ] 2.3.3 添加getTotalSteps单元测试
  ```go
  func TestGetTotalSteps_FromSearchAttributes(t *testing.T) {
      mockClient := &MockTemporalClient{}
      
      // Mock DescribeWorkflowExecution返回SearchAttributes
      mockClient.On("DescribeWorkflowExecution", mock.Anything, "wf-123", "").Return(
          &workflowservice.DescribeWorkflowExecutionResponse{
              WorkflowExecutionInfo: &workflowpb.WorkflowExecutionInfo{
                  SearchAttributes: &commonpb.SearchAttributes{
                      IndexedFields: map[string]*commonpb.Payload{
                          "TotalSteps": {Data: []byte("5")}, // 5个步骤
                      },
                  },
              },
          }, nil,
      )
      
      wqs := NewWorkflowQueryService(mockClient, zap.NewNop())
      total := wqs.getTotalSteps(context.Background(), "wf-123", "run-456")
      
      assert.Equal(t, 5, total)
  }
  
  func TestGetTotalSteps_FromHistory(t *testing.T) {
      // 测试Event History回退方案
      // ... 实现 ...
  }
  ```

- [ ] 2.3.4 更新Dev Notes说明MVP权衡
  ```markdown
  **getTotalSteps实现策略:**
  
  - **方案1 (推荐):** Story 1.5提交时存储TotalSteps到SearchAttributes
    - 优势: 精确、快速,无需遍历Event History
    - 实施: 修改Story 1.5的SubmitWorkflow方法
  
  - **方案2 (回退):** 从Event History统计ActivityTaskScheduled事件
    - 优势: 无需修改Story 1.5,适用于已运行的工作流
    - 劣势: 需要遍历Event History,性能稍差
  
  - **方案3 (MVP):** 返回固定值3
    - 优势: 简单,适用于演示
    - 劣势: 进度信息不准确
  
  **当前实现:** Task 2.2已实现三层回退策略,优先使用方案1,回退到方案2和3。
  **后续优化:** Epic 2完成后,统一在Story 1.5中实现SearchAttributes存储。
  ```

### Task 2.5: 添加WorkflowID校验工具函数 (复用代码)

- [ ] 2.5.1 创建`internal/utils/validation.go`
  ```go
  package utils
  
  import (
      "fmt"
      "strings"
      
      "github.com/google/uuid"
  )
  
  // ValidateWorkflowID 验证WorkflowID格式
  // 格式: wf-{uuid}
  // 复用于Story 1.5 (提交) 和 Story 1.7 (查询)
  func ValidateWorkflowID(id string) error {
      if id == "" {
          return fmt.Errorf("workflow ID is required")
      }
      
      if !strings.HasPrefix(id, "wf-") {
          return fmt.Errorf("workflow ID must start with 'wf-'")
      }
      
      // 验证UUID部分
      uuidPart := strings.TrimPrefix(id, "wf-")
      if _, err := uuid.Parse(uuidPart); err != nil {
          return fmt.Errorf("invalid workflow ID format: %w", err)
      }
      
      return nil
  }
  ```

### Task 3: 实现HTTP Handler (AC: GET /v1/workflows/{id})

- [ ] 3.1 更新`internal/server/handlers/workflow.go`
  ```go
  package handlers
  
  import (
      "errors"
      "fmt"
      "net/http"
      
      "github.com/gin-gonic/gin"
      "go.temporal.io/api/serviceerror"
      "go.uber.org/zap"
      
      "waterflow/internal/models"
      "waterflow/internal/service"
      "waterflow/internal/utils"
  )
  
  type WorkflowHandler struct {
      workflowService      *service.WorkflowService
      workflowQueryService *service.WorkflowQueryService
      logger               *zap.Logger
  }
  
  func NewWorkflowHandler(
      ws *service.WorkflowService,
      wqs *service.WorkflowQueryService,
      logger *zap.Logger,
  ) *WorkflowHandler {
      return &WorkflowHandler{
          workflowService:      ws,
          workflowQueryService: wqs,
          logger:               logger,
      }
  }
  
  // GetWorkflow - GET /v1/workflows/:id
  func (h *WorkflowHandler) GetWorkflow(c *gin.Context) {
      workflowID := c.Param("id")
      
      // 验证WorkflowID格式 (复用工具函数)
      if err := utils.ValidateWorkflowID(workflowID); err != nil {
          c.JSON(http.StatusBadRequest, models.NewBadRequestError(
              err.Error(),
          ))
          return
      }
      
      // 查询状态
      ctx := c.Request.Context()
      status, err := h.workflowQueryService.GetWorkflowStatus(ctx, workflowID)
      
      if err != nil {
          // 使用Temporal错误类型判断404 (更可靠)
          var notFoundErr *serviceerror.NotFound
          if errors.As(err, &notFoundErr) {
              c.JSON(http.StatusNotFound, &models.ErrorResponse{
                  Type:   "https://waterflow.io/errors/not-found",
                  Title:  "Workflow Not Found",
                  Status: 404,
                  Detail: fmt.Sprintf("Workflow with ID '%s' does not exist", workflowID),
              })
              return
          }
          
          // 其他错误
          h.logger.Error("Failed to get workflow status",
              zap.String("workflow_id", workflowID),
              zap.Error(err),
          )
          c.JSON(http.StatusInternalServerError, &models.ErrorResponse{
              Type:   "https://waterflow.io/errors/internal-error",
              Title:  "Internal Server Error",
              Status: 500,
              Detail: "Failed to retrieve workflow status",
          })
          return
      }
      
      // 返回成功响应
      c.JSON(http.StatusOK, status)
  }
  ```

### Task 4: 注册路由端点 (集成到Router)

- [ ] 4.1 更新`internal/server/router.go`
  ```go
  func SetupRouter(logger *zap.Logger, tc *temporal.Client, workflowService *service.WorkflowService) *gin.Engine {
      router := gin.New()
      
      // ... 中间件配置 ...
      
      // 健康检查
      healthHandler := handlers.NewHealthHandler(tc)
      router.GET("/health", healthHandler.HealthCheck)
      router.GET("/ready", healthHandler.ReadinessCheck)
      
      // API v1
      v1 := router.Group("/v1")
      {
          v1.GET("/", handlers.APIVersionInfo)
          
          // 验证端点
          validateHandler := handlers.NewValidateHandler()
          v1.POST("/validate", validateHandler.Validate)
          
          // 工作流端点
          workflowQueryService := service.NewWorkflowQueryService(tc, logger)
          workflowHandler := handlers.NewWorkflowHandler(workflowService, workflowQueryService, logger)
          
          v1.POST("/workflows", workflowHandler.SubmitWorkflow)
          v1.GET("/workflows/:id", workflowHandler.GetWorkflow) // 新增
      }
      
      return router
  }
  ```

- [ ] 4.2 更新Server构造函数
  ```go
  // internal/server/server.go
  
  func New(cfg *config.Config, logger *zap.Logger, tc *temporal.Client) *Server {
      // 创建Services
      parserInstance := parser.New()
      workflowService := service.NewWorkflowService(parserInstance, tc, logger)
      
      // 创建Router
      router := SetupRouter(logger, tc, workflowService)
      
      // ... 创建Server ...
  }
  ```

### Task 5: 添加缓存优化 (可选,提升性能)

- [ ] 5.1 实现简单的内存缓存
  ```go
  package service
  
  import (
      "sync"
      "time"
  )
  
  // StatusCache 状态缓存(带TTL过期策略)
  type StatusCache struct {
      cache map[string]*CacheEntry
      mu    sync.RWMutex
  }
  
  // CacheEntry 缓存条目(带过期时间)
  type CacheEntry struct {
      Status    *models.WorkflowStatusResponse
      ExpiresAt time.Time
  }
  
  func NewStatusCache() *StatusCache {
      return &StatusCache{
          cache: make(map[string]*CacheEntry),
      }
  }
  
  func (sc *StatusCache) Get(workflowID string) (*models.WorkflowStatusResponse, bool) {
      sc.mu.RLock()
      defer sc.mu.RUnlock()
      
      entry, ok := sc.cache[workflowID]
      if !ok {
          return nil, false
      }
      
      // 检查是否过期
      if time.Now().After(entry.ExpiresAt) {
          return nil, false
      }
      
      return entry.Status, true
  }
  
  func (sc *StatusCache) Set(workflowID string, status *models.WorkflowStatusResponse) {
      sc.mu.Lock()
      defer sc.mu.Unlock()
      
      // 根据状态设置不同的TTL
      var ttl time.Duration
      if status.Status == "completed" || status.Status == "failed" || status.Status == "canceled" {
          // 终态状态缓存10分钟 (不会再改变)
          ttl = 10 * time.Minute
      } else if status.Status == "running" {
          // 运行中状态缓存5秒 (频繁变化)
          ttl = 5 * time.Second
      } else {
          return // 其他状态不缓存
      }
      
      sc.cache[workflowID] = &CacheEntry{
          Status:    status,
          ExpiresAt: time.Now().Add(ttl),
      }
  }
  ```

- [ ] 5.2 在QueryService中使用缓存
  ```go
  type WorkflowQueryService struct {
      temporalClient *temporal.Client
      logger         *zap.Logger
      cache          *StatusCache // 新增
  }
  
  func NewWorkflowQueryService(tc *temporal.Client, logger *zap.Logger) *WorkflowQueryService {
      return &WorkflowQueryService{
          temporalClient: tc,
          logger:         logger,
          cache:          NewStatusCache(5 * time.Minute), // 缓存5分钟
      }
  }
  
  func (wqs *WorkflowQueryService) GetWorkflowStatus(ctx context.Context, workflowID string) (*models.WorkflowStatusResponse, error) {
      // 1. 尝试从缓存获取
      if cached, ok := wqs.cache.Get(workflowID); ok {
          wqs.logger.Debug("Cache hit", zap.String("workflow_id", workflowID))
          return cached, nil
      }
      
      // 2. 查询Temporal
      status, err := wqs.queryFromTemporal(ctx, workflowID)
      if err != nil {
          return nil, err
      }
      
      // 3. 缓存结果
      wqs.cache.Set(workflowID, status)
      
      return status, nil
  }
  ```

### Task 6: 添加单元测试 (代码质量保障)

- [ ] 6.1 创建`internal/service/workflow_query_service_test.go`
  ```go
  package service
  
  import (
      "context"
      "testing"
      "time"
      
      "github.com/stretchr/testify/assert"
      "github.com/stretchr/testify/mock"
      "go.temporal.io/api/enums/v1"
      "go.uber.org/zap"
  )
  
  func TestGetWorkflowStatus_Success(t *testing.T) {
      // Mock Temporal Client
      mockClient := &MockTemporalClient{}
      
      mockClient.On("DescribeWorkflowExecution", mock.Anything, "wf-123", "").Return(
          &workflowservice.DescribeWorkflowExecutionResponse{
              WorkflowExecutionInfo: &workflowpb.WorkflowExecutionInfo{
                  Execution: &commonpb.WorkflowExecution{
                      WorkflowId: "wf-123",
                      RunId:      "run-456",
                  },
                  Status:    enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
                  StartTime: timestamppb.New(time.Now().Add(-5 * time.Minute)),
              },
          }, nil,
      )
      
      wqs := NewWorkflowQueryService(mockClient, zap.NewNop())
      
      status, err := wqs.GetWorkflowStatus(context.Background(), "wf-123")
      
      assert.NoError(t, err)
      assert.Equal(t, "wf-123", status.WorkflowID)
      assert.Equal(t, "running", status.Status)
      assert.NotZero(t, status.Duration)
  }
  
  func TestGetWorkflowStatus_NotFound(t *testing.T) {
      mockClient := &MockTemporalClient{}
      
      mockClient.On("DescribeWorkflowExecution", mock.Anything, "wf-nonexist", "").Return(
          nil, errors.New("workflow not found"),
      )
      
      wqs := NewWorkflowQueryService(mockClient, zap.NewNop())
      
      _, err := wqs.GetWorkflowStatus(context.Background(), "wf-nonexist")
      
      assert.Error(t, err)
      assert.Contains(t, err.Error(), "not found")
  }
  
  func TestMapStatus(t *testing.T) {
      wqs := &WorkflowQueryService{}
      
      tests := []struct {
          input    enums.WorkflowExecutionStatus
          expected string
      }{
          {enums.WORKFLOW_EXECUTION_STATUS_RUNNING, "running"},
          {enums.WORKFLOW_EXECUTION_STATUS_COMPLETED, "completed"},
          {enums.WORKFLOW_EXECUTION_STATUS_FAILED, "failed"},
          {enums.WORKFLOW_EXECUTION_STATUS_CANCELED, "canceled"},
      }
      
      for _, tt := range tests {
          result := wqs.mapStatus(tt.input)
          assert.Equal(t, tt.expected, result)
      }
  }
  ```

- [ ] 6.2 测试Handler
  ```go
  // internal/server/handlers/workflow_test.go
  
  func TestGetWorkflow_Success(t *testing.T) {
      gin.SetMode(gin.TestMode)
      
      mockQueryService := &MockWorkflowQueryService{}
      mockQueryService.On("GetWorkflowStatus", mock.Anything, "wf-123").Return(
          &models.WorkflowStatusResponse{
              WorkflowID: "wf-123",
              Status:     "running",
              StartTime:  time.Now(),
          }, nil,
      )
      
      handler := NewWorkflowHandler(nil, mockQueryService, zap.NewNop())
      
      router := gin.New()
      router.GET("/workflows/:id", handler.GetWorkflow)
      
      req := httptest.NewRequest("GET", "/workflows/wf-123", nil)
      w := httptest.NewRecorder()
      router.ServeHTTP(w, req)
      
      assert.Equal(t, http.StatusOK, w.Code)
      
      var resp models.WorkflowStatusResponse
      json.Unmarshal(w.Body.Bytes(), &resp)
      assert.Equal(t, "wf-123", resp.WorkflowID)
  }
  
  func TestGetWorkflow_NotFound(t *testing.T) {
      mockQueryService := &MockWorkflowQueryService{}
      mockQueryService.On("GetWorkflowStatus", mock.Anything, "wf-nonexist").Return(
          nil, errors.New("workflow not found"),
      )
      
      handler := NewWorkflowHandler(nil, mockQueryService, zap.NewNop())
      
      router := gin.New()
      router.GET("/workflows/:id", handler.GetWorkflow)
      
      req := httptest.NewRequest("GET", "/workflows/wf-nonexist", nil)
      w := httptest.NewRecorder()
      router.ServeHTTP(w, req)
      
      assert.Equal(t, http.StatusNotFound, w.Code)
  }
  ```

- [ ] 6.3 运行测试
  ```bash
  go test -v ./internal/service
  go test -v ./internal/server/handlers
  ```

### Task 7: 集成测试 (端到端验证)

- [ ] 7.1 创建集成测试脚本
  ```bash
  # test/integration/test_workflow_query.sh
  #!/bin/bash
  
  set -e
  
  echo "=== Workflow Query API Integration Test ==="
  
  # 1. 启动环境
  make dev-env
  go run ./cmd/server &
  SERVER_PID=$!
  sleep 3
  
  # 2. 提交工作流
  echo "Submitting workflow..."
  RESPONSE=$(curl -s -X POST http://localhost:8080/v1/workflows \
    -H "Content-Type: application/json" \
    -d '{
      "workflow": "name: Test\non: push\njobs:\n  build:\n    runs-on: linux\n    steps:\n      - name: Build\n        uses: run@v1"
    }')
  
  WORKFLOW_ID=$(echo $RESPONSE | jq -r '.workflow_id')
  echo "Workflow ID: $WORKFLOW_ID"
  
  # 3. 查询状态
  echo "Querying workflow status..."
  sleep 2
  
  STATUS=$(curl -s http://localhost:8080/v1/workflows/$WORKFLOW_ID)
  echo "Status: $STATUS"
  
  # 验证响应包含必需字段
  echo $STATUS | jq -e '.workflow_id' > /dev/null
  echo $STATUS | jq -e '.status' > /dev/null
  echo $STATUS | jq -e '.start_time' > /dev/null
  
  echo "✅ Status fields validated"
  
  # 4. 性能测试 (AC要求<200ms)
  echo "Running performance test..."
  if command -v ab &> /dev/null; then
      ab -n 1000 -c 50 http://localhost:8080/v1/workflows/$WORKFLOW_ID > /tmp/ab_result.txt 2>&1
      
      # 提取P95延迟
      P95=$(grep "95%" /tmp/ab_result.txt | awk '{print $2}')
      echo "P95 Latency: ${P95}ms"
      
      # 验证P95 < 200ms
      if [ $(echo "$P95 < 200" | bc) -eq 1 ]; then
          echo "✅ Performance test passed (P95: ${P95}ms < 200ms)"
      else
          echo "⚠️  Performance warning (P95: ${P95}ms >= 200ms)"
      fi
  else
      echo "⚠️  Apache Bench (ab) not installed, skipping performance test"
  fi
  
  # 5. 测试404错误
  echo "Testing 404 response..."
  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/v1/workflows/wf-nonexist)
  if [ "$HTTP_CODE" = "404" ]; then
      echo "✅ 404 test passed"
  else
      echo "❌ Expected 404, got $HTTP_CODE"
      exit 1
  fi
  
  # 5. 测试404错误
  echo "Testing 404 response..."
  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/v1/workflows/wf-nonexist)
  if [ "$HTTP_CODE" = "404" ]; then
      echo "✅ 404 test passed"
  else
      echo "❌ Expected 404, got $HTTP_CODE"
      exit 1
  fi
  
  # 6. 清理
  kill $SERVER_PID
  make dev-env-stop
  
  echo "✅ Integration test completed"
  ```

### Task 8: 更新OpenAPI文档 (API文档化)

- [ ] 8.1 更新`api/openapi.yaml`
  ```yaml
  paths:
    /v1/workflows/{id}:
      get:
        summary: Get workflow status
        description: Query the current status and progress of a workflow execution
        tags:
          - Workflows
        parameters:
          - name: id
            in: path
            required: true
            schema:
              type: string
              pattern: '^wf-[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$'
            description: Workflow ID (format wf-{uuid})
            example: wf-550e8400-e29b-41d4-a716-446655440000
        responses:
          '200':
            description: Workflow status retrieved successfully
            content:
              application/json:
                schema:
                  $ref: '#/components/schemas/WorkflowStatusResponse'
                examples:
                  running:
                    summary: Running workflow
                    value:
                      workflow_id: wf-550e8400-e29b-41d4-a716-446655440000
                      run_id: temporal-run-uuid
                      name: Deploy Application
                      status: running
                      start_time: "2025-12-16T10:30:00Z"
                      duration: 125000
                      progress:
                        current_job: build
                        current_step: Build
                        total_steps: 3
                        completed_steps: 1
                  completed:
                    summary: Completed workflow
                    value:
                      workflow_id: wf-550e8400-e29b-41d4-a716-446655440000
                      status: completed
                      close_time: "2025-12-16T10:32:05Z"
                      duration: 125000
                      result:
                        success: true
          '400':
            description: Invalid workflow ID format
            content:
              application/json:
                schema:
                  $ref: '#/components/schemas/ErrorResponse'
          '404':
            description: Workflow not found
            content:
              application/json:
                schema:
                  $ref: '#/components/schemas/ErrorResponse'
          '500':
            description: Internal server error
  
  components:
    schemas:
      WorkflowStatusResponse:
        type: object
        properties:
          workflow_id:
            type: string
            example: wf-550e8400-e29b-41d4-a716-446655440000
          run_id:
            type: string
            example: temporal-uuid-12345
          name:
            type: string
            example: Deploy Application
          status:
            type: string
            enum: [running, completed, failed, canceled, timeout]
            example: running
          start_time:
            type: string
            format: date-time
            example: 2025-12-16T10:30:00Z
          close_time:
            type: string
            format: date-time
            example: 2025-12-16T10:32:05Z
            nullable: true
          duration:
            type: integer
            description: Duration in milliseconds
            example: 125000
          progress:
            $ref: '#/components/schemas/ProgressInfo'
          result:
            $ref: '#/components/schemas/WorkflowResult'
          error:
            $ref: '#/components/schemas/WorkflowError'
      
      ProgressInfo:
        type: object
        properties:
          current_job:
            type: string
            example: build
          current_step:
            type: string
            example: Build
          total_steps:
            type: integer
            example: 3
          completed_steps:
            type: integer
            example: 1
      
      WorkflowResult:
        type: object
        properties:
          success:
            type: boolean
            example: true
          outputs:
            type: object
            additionalProperties: true
      
      WorkflowError:
        type: object
        properties:
          type:
            type: string
            example: StepExecutionError
          message:
            type: string
            example: "Step 'Build' failed: exit code 1"
          failed_step:
            type: string
            example: Build
  ```

## Dev Notes

### Critical Implementation Guidelines

**1. 错误处理 - 区分404和500**

```go
// ✅ 正确: 检查错误类型
if strings.Contains(err.Error(), "not found") {
    c.JSON(404, ...)
    return
}
c.JSON(500, ...)

// ❌ 错误: 所有错误返回500
c.JSON(500, gin.H{"error": err.Error()})
```

**2. 进度计算 - 避免阻塞**

```go
// ✅ 超时控制
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()

progress, err := wqs.getProgress(ctx, workflowID, runID)

// ❌ 无超时可能导致长时间阻塞
progress, err := wqs.getProgress(context.Background(), ...)
```

**3. 缓存策略 - 只缓存终态**

```go
// ✅ 只缓存已完成的Workflow
if status.Status == "completed" || status.Status == "failed" {
    cache.Set(workflowID, status)
}

// ❌ 缓存running状态会导致数据过时
cache.Set(workflowID, status) // 包括running状态
```

**4. 时间处理 - 处理nil值**

```go
// ✅ 安全处理CloseTime
if info.CloseTime != nil {
    response.CloseTime = info.CloseTime
}

// ❌ 直接赋值可能panic
response.CloseTime = *info.CloseTime
```

**5. Event History遍历 - 限制数量**

```go
// ✅ 限制遍历数量
maxEvents := 1000
count := 0
for iter.HasNext() && count < maxEvents {
    event, _ := iter.Next()
    count++
}

// ❌ 无限遍历可能超时
for iter.HasNext() {
    event, _ := iter.Next()
}
```

**6. 性能优化 - 并发查询**

```go
// ✅ 使用Temporal Client连接池
// Client内部已实现连接池,无需手动管理

// 支持高并发查询
for i := 0; i < 100; i++ {
    go func(id string) {
        status, _ := queryService.GetWorkflowStatus(ctx, id)
    }(workflowIDs[i])
}
```

### Integration with Previous Stories

**与Story 1.4 Temporal Client集成:**

```go
// Story 1.4提供的Client
temporalClient.GetClient()

// Story 1.7使用Client查询
describe, err := temporalClient.GetClient().DescribeWorkflowExecution(ctx, workflowID, "")
```

**与Story 1.5 WorkflowID集成:**

```go
// Story 1.5生成的WorkflowID格式
workflowID := fmt.Sprintf("wf-%s", uuid.New().String())

// Story 1.7使用WorkflowID查询
status, err := queryService.GetWorkflowStatus(ctx, workflowID)
```

**与Story 1.6 Workflow执行集成:**

```go
// Story 1.6执行产生Event History
workflow.ExecuteActivity(ctx, "ExecuteStepActivity", input)

// Story 1.7从Event History提取进度
iter := client.GetWorkflowHistory(ctx, workflowID, runID, ...)
```

**为Story 1.8准备:**

```go
// Story 1.8将扩展日志功能
// 复用Event History遍历逻辑
for iter.HasNext() {
    event, _ := iter.Next()
    // 提取日志信息
}
```

### Testing Strategy

**单元测试覆盖:**

| 组件 | 测试场景 |
|------|---------|
| WorkflowQueryService | 状态查询、进度计算、错误提取 |
| StatusCache | 缓存命中、过期、终态缓存 |
| WorkflowHandler | 成功响应、404、500错误 |
| mapStatus | 所有Temporal状态枚举 |

**集成测试:**

```bash
# 1. 提交工作流
curl -X POST /v1/workflows -d @workflow.json
# 返回: {"workflow_id":"wf-xxx"}

# 2. 查询状态 (running)
curl /v1/workflows/wf-xxx
# {"status":"running","progress":{...}}

# 3. 等待完成后查询
sleep 10
curl /v1/workflows/wf-xxx
# {"status":"completed","result":{...}}

# 4. 测试404
curl /v1/workflows/wf-nonexist
# {"status":404,"title":"Workflow Not Found"}
```

**性能测试:**

```bash
# 使用Apache Bench测试并发查询
ab -n 1000 -c 50 http://localhost:8080/v1/workflows/wf-xxx

# 期望: p95 < 200ms
```

### References

**架构设计:**
- [docs/architecture.md §3.1.1](docs/architecture.md) - REST API Handler设计

**技术文档:**
- [Temporal DescribeWorkflowExecution](https://pkg.go.dev/go.temporal.io/sdk/client#Client.DescribeWorkflowExecution)
- [Temporal GetWorkflowHistory](https://pkg.go.dev/go.temporal.io/sdk/client#Client.GetWorkflowHistory)
- [RFC 7807: Problem Details](https://tools.ietf.org/html/rfc7807)

**项目上下文:**
- [docs/epics.md Story 1.1-1.6](docs/epics.md) - 前置Stories
- [docs/epics.md Story 1.8](docs/epics.md) - 日志输出 (扩展本Story)

### Dependency Graph

```
Story 1.4 (Temporal Client) ──┐
Story 1.5 (提交API)         ──┤
Story 1.6 (执行引擎)         ──┤
                              ↓
Story 1.7 (状态查询API) ← 当前Story
    ↓
    ├→ Story 1.8 (日志输出) - 扩展查询功能
    └→ Story 1.9 (取消API) - 依赖状态查询判断是否可取消
```

## Dev Agent Record

### Context Reference

**Source Documents Analyzed:**
1. [docs/epics.md](docs/epics.md) (lines 377-394) - Story 1.7需求定义
2. [docs/architecture.md](docs/architecture.md) (§3.1.1) - REST API Handler设计

**Previous Stories:**
- Story 1.1-1.6: 全部drafted (框架、API、解析器、Temporal、提交、执行)

### Agent Model Used

Claude 3.5 Sonnet (BMM Scrum Master Agent - Bob)

### Estimated Effort

**开发时间:** 6-8小时  
**复杂度:** 中等

**时间分解:**
- 状态响应模型: 1小时
- 查询服务层实现: 2小时
- 进度提取逻辑: 1.5小时
- HTTP Handler: 1小时
- 缓存优化: 1小时
- 单元测试: 1.5小时
- 集成测试: 1小时
- OpenAPI文档: 1小时

**技能要求:**
- Temporal Client API
- Event History分析
- REST API设计
- 缓存策略

### Debug Log References

<!-- Will be populated during implementation -->

### Completion Notes List

<!-- Developer填写完成时的笔记 -->

### File List

**预期创建/修改的文件清单:**

```
新建文件 (~4个):
├── internal/models/
│   └── workflow_status.go          # 详见Task 1
├── internal/service/
│   ├── workflow_query_service.go   # 详见Task 2
│   └── workflow_query_service_test.go  # 详见Task 6
├── internal/utils/
│   └── validation.go               # 详见Task 2.5 (WorkflowID校验)
├── test/integration/
│   └── test_workflow_query.sh      # 详见Task 7

修改文件 (~3个):
├── internal/server/handlers/workflow.go  # 详见Task 3 (添加GetWorkflow)
├── internal/server/router.go             # 详见Task 4 (注册GET端点)
└── api/openapi.yaml                      # 详见Task 8 (API文档)
```

**详细实现代码请参考Tasks 0-8各小节,此处省略以节省token。**

**关键技术要点:**
- Event Sourcing查询 (从Temporal Event History读取状态)
- 404错误精确判断 (serviceerror.NotFound类型)
- WorkflowID校验复用 (utils.ValidateWorkflowID)
- 缓存TTL策略 (completed:10分钟, running:5秒)
- 性能优化 (AC要求<200ms,P95测试验证)

---

**Story Ready for Development** ✅

开发者可基于Story 1.1-1.6的成果,实现工作流状态查询API。
本Story完成后,用户可以实时查看工作流执行状态和进度。
        }
        c.JSON(500, models.NewInternalError())
        return
    }
    
    c.JSON(200, status)
}
```

---

**Story Ready for Development** ✅

开发者可基于Story 1.1-1.6的成果,实现工作流状态查询API。
本Story完成后,用户可以实时查看工作流执行状态和进度。
