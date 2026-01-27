# Waterflow Workflow Management API 调研报告

**版本:** 1.0  
**日期:** 2026-01-27  
**状态:** 已完成  
**调研人员:** 架构师团队

---

## 文档说明

本文档为 **Story 1.9: Workflow Management API** 提供完整的技术调研,包括:
- Temporal Go SDK 原生能力分析
- 行业最佳实践对比(GitHub Actions, GitLab CI, Argo Workflows)
- Waterflow 架构特性适配分析
- API 设计建议和验收标准更新

本调研基于 Waterflow 的 Event Sourcing 架构和 Temporal SDK v1.38.0 的实际能力。

---

## 1. Temporal SDK 能力分析

### 1.1 工作流查询和状态获取

#### 核心 API

**1. `client.GetWorkflow(ctx, workflowID, runID)` - 获取工作流句柄**

```go
// 获取工作流句柄,用于后续操作
run := client.GetWorkflow(ctx, workflowID, runID)

// 获取执行结果(阻塞直到完成)
var result interface{}
err := run.Get(ctx, &result)

// 获取 WorkflowID 和 RunID
wfID := run.GetID()
runID := run.GetRunID()
```

**能力:**
- ✅ 获取工作流引用,即使工作流已完成
- ✅ 支持通过 WorkflowID 查询(runID 传空字符串使用最新 run)
- ✅ Get() 方法阻塞等待结果,支持超时控制

**限制:**
- ⚠️ Get() 只能获取最终结果,无法获取中间进度
- ⚠️ 不提供详细的执行状态信息

**2. `client.DescribeWorkflowExecution(ctx, workflowID, runID)` - 详细执行信息**

```go
// 查询工作流执行详情
desc, err := client.DescribeWorkflowExecution(ctx, workflowID, "")
if err != nil {
    return fmt.Errorf("workflow not found: %w", err)
}

// 获取状态
status := desc.WorkflowExecutionInfo.Status
// enums.WORKFLOW_EXECUTION_STATUS_RUNNING
// enums.WORKFLOW_EXECUTION_STATUS_COMPLETED
// enums.WORKFLOW_EXECUTION_STATUS_FAILED
// enums.WORKFLOW_EXECUTION_STATUS_CANCELED
// enums.WORKFLOW_EXECUTION_STATUS_TERMINATED
// enums.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW
// enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT

// 获取时间信息
startTime := desc.WorkflowExecutionInfo.StartTime
closeTime := desc.WorkflowExecutionInfo.CloseTime
executionTime := desc.WorkflowExecutionInfo.ExecutionTime

// 获取工作流类型
workflowType := desc.WorkflowExecutionInfo.Type.Name
```

**能力:**
- ✅ 实时获取工作流状态(运行中/已完成/失败/取消等)
- ✅ 获取开始时间、结束时间、执行持续时间
- ✅ 获取工作流类型名称
- ✅ 获取父工作流信息(如果是子工作流)
- ✅ 获取任务队列信息
- ✅ 性能优秀,延迟 < 50ms

**限制:**
- ⚠️ 不包含 Job/Step 级别的详细进度
- ⚠️ 不包含变量值和上下文信息

**3. `client.GetWorkflowHistory(ctx, workflowID, runID, isLongPoll, filterType)` - 获取事件历史**

```go
// 获取完整事件历史
iter := client.GetWorkflowHistory(ctx, workflowID, runID, false, 0)

var events []*history.HistoryEvent
for iter.HasNext() {
    event, err := iter.Next()
    if err != nil {
        return err
    }
    events = append(events, event)
}

// 解析 Activity 事件获取 Step 状态
for _, event := range events {
    switch event.EventType {
    case enums.EVENT_TYPE_ACTIVITY_TASK_STARTED:
        // Step 开始
    case enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
        // Step 完成
    case enums.EVENT_TYPE_ACTIVITY_TASK_FAILED:
        // Step 失败
    }
}
```

**能力:**
- ✅ 获取完整的事件历史(Event Sourcing 源数据)
- ✅ 支持过滤类型(仅获取特定事件)
- ✅ 支持长轮询模式获取实时更新
- ✅ 可重建任意时刻的工作流状态(时间旅行)
- ✅ 包含所有 Activity(Step)的执行详情

**限制:**
- ⚠️ Event History 有大小限制(默认 50MB,可配置到 500MB)
- ⚠️ 需要解析事件来提取 Job/Step 状态(无直接 API)
- ⚠️ 长时运行工作流可能产生数千个事件

**Waterflow 实现示例:**

```go
// pkg/temporal/history_parser.go
type HistoryParser struct{}

func (p *HistoryParser) ParseJobsFromHistory(events []*history.HistoryEvent) []JobStatus {
    jobs := make([]JobStatus, 0)
    currentSteps := make(map[string]*StepStatus)
    
    for _, event := range events {
        switch event.EventType {
        case enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED:
            // 记录 Step 调度
        case enums.EVENT_TYPE_ACTIVITY_TASK_STARTED:
            // 更新 Step 为 running
        case enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
            // 更新 Step 为 completed
        case enums.EVENT_TYPE_ACTIVITY_TASK_FAILED:
            // 更新 Step 为 failed,提取错误信息
        }
    }
    
    return jobs
}
```

**性能考虑:**
- 小型工作流(< 100 events): < 100ms
- 中型工作流(100-1000 events): 100-500ms
- 大型工作流(1000+ events): 500ms-2s

---

### 1.2 工作流控制操作

#### 核心 API

**1. `client.CancelWorkflow(ctx, workflowID, runID)` - 取消工作流**

```go
// 发送取消信号
err := client.CancelWorkflow(ctx, workflowID, "")
if err != nil {
    return fmt.Errorf("cancel failed: %w", err)
}

// 工作流代码中检测取消
func MyWorkflow(ctx workflow.Context) error {
    // 检查是否被取消
    if err := ctx.Err(); err != nil {
        return err // workflow.ErrCanceled
    }
    
    // Activity 执行时自动检测取消
    err := workflow.ExecuteActivity(ctx, MyActivity).Get(ctx, nil)
    if err != nil {
        // 如果工作流被取消,Activity 会被自动取消
        return err
    }
}
```

**能力:**
- ✅ 优雅取消,给工作流机会执行清理逻辑
- ✅ 自动传播到所有 Activity
- ✅ 工作流可以捕获取消信号执行补偿操作
- ✅ 状态变为 WORKFLOW_EXECUTION_STATUS_CANCELED

**限制:**
- ⚠️ 取消是协作式的,工作流代码必须检查 ctx.Err()
- ⚠️ 如果工作流代码不检查取消,工作流会继续运行
- ⚠️ 无法取消已完成的工作流

**使用场景:**
- ✅ 用户手动取消长时运行的部署任务
- ✅ 超时后的自动取消
- ✅ 依赖失败后取消下游任务

**2. `client.TerminateWorkflow(ctx, workflowID, runID, reason, details)` - 终止工作流**

```go
// 强制终止工作流
err := client.TerminateWorkflow(
    ctx,
    workflowID,
    "",
    "manual termination by admin",
    nil, // 可选的详细信息
)
```

**能力:**
- ✅ 立即终止,不执行任何清理逻辑
- ✅ 状态变为 WORKFLOW_EXECUTION_STATUS_TERMINATED
- ✅ 可以终止任何状态的工作流(包括卡住的工作流)
- ✅ 记录终止原因到 Event History

**限制:**
- ⚠️ 无法执行补偿逻辑
- ⚠️ 可能导致资源泄漏(如果工作流持有外部资源)
- ⚠️ 不可恢复

**使用场景:**
- ✅ 工作流代码有 bug 导致卡住
- ✅ 紧急情况需要立即停止
- ✅ 工作流无响应(不检查取消信号)

#### Cancel vs Terminate 对比

| 维度 | Cancel | Terminate |
|------|--------|-----------|
| **执行方式** | 协作式,发送取消信号 | 强制式,立即终止 |
| **清理逻辑** | ✅ 可以执行 | ❌ 不执行 |
| **资源释放** | ✅ 工作流负责 | ⚠️ 可能泄漏 |
| **状态** | CANCELED | TERMINATED |
| **Event History** | 包含取消处理逻辑 | 仅记录终止原因 |
| **使用场景** | 正常取消操作 | 紧急情况/工作流卡住 |
| **推荐** | ✅ 首选 | ⚠️ 谨慎使用 |

**Waterflow 实践建议:**
- 默认使用 `CancelWorkflow` (用户主动取消)
- `TerminateWorkflow` 仅用于管理员强制终止
- API 应该提供两个端点:
  - `POST /v1/workflows/{id}/cancel` - 普通用户可用
  - `POST /v1/workflows/{id}/terminate` - 需要管理员权限

---

### 1.3 工作流信号和查询

#### 核心 API

**1. `client.SignalWorkflow(ctx, workflowID, runID, signalName, arg)` - 发送信号**

```go
// 发送信号到运行中的工作流
err := client.SignalWorkflow(
    ctx,
    workflowID,
    "",
    "pause", // 信号名称
    map[string]interface{}{"reason": "maintenance"}, // 信号参数
)
```

**工作流代码中接收信号:**

```go
func MyWorkflow(ctx workflow.Context) error {
    var paused bool
    
    // 注册信号处理器
    pauseChannel := workflow.GetSignalChannel(ctx, "pause")
    workflow.Go(ctx, func(ctx workflow.Context) {
        var pauseReason map[string]interface{}
        pauseChannel.Receive(ctx, &pauseReason)
        paused = true
        logger.Info("Workflow paused", pauseReason)
    })
    
    // 等待恢复信号
    resumeChannel := workflow.GetSignalChannel(ctx, "resume")
    if paused {
        resumeChannel.Receive(ctx, nil)
        paused = false
    }
}
```

**能力:**
- ✅ 向运行中的工作流发送外部事件
- ✅ 支持任意 JSON 序列化参数
- ✅ 异步操作,不阻塞
- ✅ 信号持久化到 Event History

**限制:**
- ⚠️ 工作流必须预先注册信号处理器
- ⚠️ 信号可能丢失(如果工作流未监听)
- ⚠️ 无法获取信号处理结果

**使用场景:**
- ✅ 暂停/恢复工作流
- ✅ 更新工作流配置
- ✅ 触发工作流内部事件

**2. `client.QueryWorkflow(ctx, workflowID, runID, queryType, args)` - 查询内部状态**

```go
// 查询工作流内部状态
resp, err := client.QueryWorkflow(
    ctx,
    workflowID,
    "",
    "getProgress", // 查询类型
)

var progress map[string]interface{}
err = resp.Get(&progress)
```

**工作流代码中处理查询:**

```go
func MyWorkflow(ctx workflow.Context) error {
    var completedSteps int
    var totalSteps int = 10
    
    // 注册查询处理器
    err := workflow.SetQueryHandler(ctx, "getProgress", func() (map[string]interface{}, error) {
        return map[string]interface{}{
            "completed": completedSteps,
            "total":     totalSteps,
            "percentage": float64(completedSteps) / float64(totalSteps) * 100,
        }, nil
    })
}
```

**能力:**
- ✅ 同步读取工作流内部状态
- ✅ 不影响工作流执行(只读操作)
- ✅ 支持复杂查询逻辑
- ✅ 延迟低(< 50ms)

**限制:**
- ⚠️ 工作流必须预先注册查询处理器
- ⚠️ 只能查询当前状态,不能修改
- ⚠️ 查询不持久化到 Event History

**使用场景:**
- ✅ 获取工作流执行进度
- ✅ 获取当前变量值
- ✅ 调试工作流状态

#### 通过 Signal 实现暂停/恢复

**设计方案:**

```go
// Workflow 实现
func DeployWorkflow(ctx workflow.Context, input DeployInput) error {
    var paused bool
    var pauseReason string
    
    // 注册暂停信号
    pauseChannel := workflow.GetSignalChannel(ctx, "pause")
    workflow.Go(ctx, func(ctx workflow.Context) {
        for {
            var req PauseRequest
            pauseChannel.Receive(ctx, &req)
            paused = true
            pauseReason = req.Reason
        }
    })
    
    // 注册恢复信号
    resumeChannel := workflow.GetSignalChannel(ctx, "resume")
    workflow.Go(ctx, func(ctx workflow.Context) {
        for {
            resumeChannel.Receive(ctx, nil)
            paused = false
            pauseReason = ""
        }
    })
    
    // 注册查询处理器
    workflow.SetQueryHandler(ctx, "isPaused", func() (bool, error) {
        return paused, nil
    })
    
    // 执行 Steps,检查暂停状态
    for _, step := range input.Steps {
        // 等待恢复
        for paused {
            workflow.Sleep(ctx, time.Second)
        }
        
        // 执行 Step
        err := workflow.ExecuteActivity(ctx, ExecuteStep, step).Get(ctx, nil)
        if err != nil {
            return err
        }
    }
    
    return nil
}
```

**API 设计:**

```http
# 暂停工作流
POST /v1/workflows/{id}/pause
{
  "reason": "manual pause for debugging"
}

# 恢复工作流
POST /v1/workflows/{id}/resume

# 查询暂停状态
GET /v1/workflows/{id}/pause-status
Response:
{
  "paused": true,
  "reason": "manual pause for debugging",
  "paused_at": "2026-01-27T10:30:00Z"
}
```

**实现限制:**
- ⚠️ 需要 Workflow 代码支持暂停逻辑
- ⚠️ 暂停点只能在 Activity 之间(不能暂停 Activity 执行)
- ⚠️ 长时间暂停可能触发工作流超时

---

### 1.4 工作流重试和重置

#### 核心 API

**1. `client.ResetWorkflowExecution(ctx, request)` - 重置到历史点**

```go
// 重置工作流到指定事件
resetResp, err := client.ResetWorkflowExecution(ctx, &workflowservice.ResetWorkflowExecutionRequest{
    Namespace: "default",
    WorkflowExecution: &commonpb.WorkflowExecution{
        WorkflowId: workflowID,
        RunId:      runID, // 必须指定 RunID
    },
    Reason: "Reset to before failed activity",
    // 重置到指定事件 ID
    WorkflowTaskFinishEventId: eventID,
    // 或者重置到第一个 Workflow Task
    // RequestId: uuid.New().String(),
})

// 新的 RunID
newRunID := resetResp.RunId
```

**能力:**
- ✅ 重置到 Event History 中的任意 Workflow Task 完成点
- ✅ 保留原始 Event History(创建新的 Run)
- ✅ 可以用于修复失败的工作流
- ✅ 支持重置到第一个任务(完全重新执行)

**限制:**
- ⚠️ 只能重置已完成/失败的工作流(不能重置运行中的)
- ⚠️ 必须重置到 Workflow Task 完成点(不能重置到 Activity 中间)
- ⚠️ 工作流代码必须是确定性的(否则重放可能失败)
- ⚠️ 复杂度高,需要理解 Event History 结构

**使用场景:**
- ✅ 修复失败的工作流(例如 Activity 代码有 bug)
- ✅ 重新执行特定分支
- ⚠️ 不推荐作为常规重试机制

**Waterflow 实践建议:**
- ❌ MVP 阶段不实现 Reset API
- ✅ 使用 "重新运行" 代替(创建新工作流执行)
- ✅ Reset 留作高级功能(管理员使用)

**2. 从特定 Activity 重试的可行性**

**问题:** Waterflow 用户期望能够 "从失败的 Step 重新开始",但 Temporal 不直接支持。

**Temporal 限制:**
- Activity 重试是自动的(由 RetryPolicy 控制)
- 手动重试只能重置整个工作流
- 不能跳过成功的 Activity 直接重试失败的

**Waterflow 解决方案:**

**方案 A: 重新运行整个工作流(推荐 MVP)**

```http
POST /v1/workflows/{id}/rerun
{
  "skip_steps": ["build", "test"], // 可选:跳过某些 Step
  "from_step": "deploy" // 可选:从特定 Step 开始
}
```

实现:创建新的工作流执行,但允许指定起始 Step。

**方案 B: 使用 Workflow 级别的条件跳过**

```go
func DeployWorkflow(ctx workflow.Context, input DeployInput) error {
    // 检查是否是重试,哪些 Step 需要跳过
    if input.SkipSteps != nil {
        // 跳过已成功的 Step
    }
    
    for _, step := range input.Steps {
        if shouldSkip(step, input.SkipSteps) {
            continue
        }
        
        // 执行 Step
        err := workflow.ExecuteActivity(ctx, ExecuteStep, step).Get(ctx, nil)
    }
}
```

**方案 C: 使用 Signal 动态跳过**

```go
// 发送信号跳过失败的 Step
client.SignalWorkflow(ctx, workflowID, "", "skipStep", "deploy")
```

**推荐方案:** 方案 A (重新运行) for MVP
- 简单易懂
- 不需要复杂的 Workflow 逻辑
- 符合用户预期

---

### 1.5 工作流列表和搜索

#### 核心 API

**`client.ListWorkflow(ctx, request)` - 列表查询**

```go
// 列出工作流执行
listResp, err := client.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
    Namespace: "default",
    PageSize:  20,
    // 下一页 token
    NextPageToken: nextPageToken,
    // 可见性查询(SQL-like)
    Query: "WorkflowType='DeployWorkflow' AND ExecutionStatus='Running'",
})

for _, exec := range listResp.Executions {
    fmt.Printf("Workflow: %s, Status: %s\n", 
        exec.Execution.WorkflowId,
        exec.Status,
    )
}

// 获取下一页
nextPageToken = listResp.NextPageToken
```

**支持的过滤条件:**

**1. 基本字段:**
```sql
-- 工作流类型
WorkflowType = 'DeployWorkflow'

-- 工作流 ID
WorkflowId = 'deploy-12345'

-- 执行状态
ExecutionStatus = 'Running'
ExecutionStatus IN ('Running', 'Completed')

-- 开始时间范围
StartTime BETWEEN '2026-01-01T00:00:00Z' AND '2026-01-31T23:59:59Z'

-- 结束时间范围
CloseTime > '2026-01-27T00:00:00Z'
```

**2. 自定义搜索属性:**

需要先在 Workflow 中设置搜索属性:

```go
// Workflow 代码中设置搜索属性
workflow.UpsertSearchAttributes(ctx, map[string]interface{}{
    "CustomKeywordField": "deploy-production",
    "CustomIntField": 123,
})
```

Temporal Server 配置搜索属性:

```bash
# 注册自定义搜索属性
tctl admin cluster add-search-attributes \
    --name WorkflowName --type Keyword \
    --name Environment --type Keyword \
    --name Priority --type Int
```

查询示例:

```sql
-- 按自定义属性过滤
WorkflowName = 'Deploy App' AND Environment = 'production'

-- 复合查询
WorkflowType = 'DeployWorkflow' 
  AND ExecutionStatus = 'Running' 
  AND Environment = 'production'
  AND Priority > 5
  ORDER BY StartTime DESC
```

**能力:**
- ✅ SQL-like 查询语法
- ✅ 支持分页(PageSize, NextPageToken)
- ✅ 支持排序(ORDER BY)
- ✅ 支持自定义搜索属性
- ✅ 索引优化,性能好(< 200ms for 1M+ workflows)

**限制:**
- ⚠️ 查询语法有限(不支持 JOIN, GROUP BY 等)
- ⚠️ 自定义搜索属性需要预先注册
- ⚠️ 搜索属性类型有限(Keyword, Int, Double, Bool, Datetime)
- ⚠️ 文本搜索不支持模糊匹配(必须精确匹配或前缀匹配)
- ⚠️ PageSize 最大 1000

**性能特征:**
- 小规模(< 1K workflows): < 50ms
- 中等规模(1K-100K): 50-200ms
- 大规模(100K-1M+): 200ms-1s (取决于查询复杂度)

**Waterflow 实现建议:**

```go
// 构建 Temporal 可见性查询
func buildTemporalVisibilityQuery(apiParams url.Values) string {
    var conditions []string
    
    // 工作流类型(固定)
    conditions = append(conditions, "WorkflowType = 'waterflow-executor'")
    
    // 状态过滤
    if status := apiParams.Get("status"); status != "" {
        statuses := strings.Split(status, ",")
        statusConditions := make([]string, len(statuses))
        for i, s := range statuses {
            statusConditions[i] = fmt.Sprintf("'%s'", mapToTemporalStatus(s))
        }
        conditions = append(conditions, 
            fmt.Sprintf("ExecutionStatus IN (%s)", strings.Join(statusConditions, ",")))
    }
    
    // 时间范围
    if createdAfter := apiParams.Get("created_after"); createdAfter != "" {
        conditions = append(conditions, 
            fmt.Sprintf("StartTime > '%s'", createdAfter))
    }
    
    // 工作流名称(需要自定义搜索属性)
    if name := apiParams.Get("name"); name != "" {
        conditions = append(conditions, 
            fmt.Sprintf("WorkflowName = '%s'", name))
    }
    
    query := strings.Join(conditions, " AND ")
    query += " ORDER BY StartTime DESC"
    
    return query
}
```

**搜索属性设计:**

建议在 Workflow 启动时设置以下搜索属性:

```go
// Workflow 代码
workflow.UpsertSearchAttributes(ctx, map[string]interface{}{
    "WorkflowName": workflow.Name, // 来自 YAML
    "Environment": workflow.Vars["env"], // 环境变量
    "SubmittedBy": workflow.Metadata["user"], // 提交用户
})
```

这样用户可以按 Workflow 名称搜索:

```http
GET /v1/workflows?name=Deploy App&status=running
```

---

### 1.6 日志和 Event History

#### 核心 API

**`client.GetWorkflowHistory(ctx, workflowID, runID, isLongPoll, filterType)` - 获取历史**

```go
// 获取完整 Event History
iter := client.GetWorkflowHistory(ctx, workflowID, runID, false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)

var events []*history.HistoryEvent
for iter.HasNext() {
    event, err := iter.Next()
    if err != nil {
        break
    }
    events = append(events, event)
}
```

**Event History 过滤类型:**

```go
// 所有事件(包括内部事件)
enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT

// 仅关闭事件(工作流完成/失败)
enums.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT
```

**能力:**
- ✅ 完整的事件历史,包含所有状态变更
- ✅ 分页迭代器,自动处理大型历史
- ✅ 支持长轮询(实时接收新事件)
- ✅ 事件包含时间戳、属性、关联事件 ID

**Event History 大小限制:**

| 配置项 | 默认值 | 最大值 | 说明 |
|--------|--------|--------|------|
| `history.maxPageSize` | 1000 events/page | 5000 | 单次查询返回的最大事件数 |
| `history.maxSize` | 50 MB | 500 MB | 单个工作流 Event History 总大小 |
| `history.sizeWarningThreshold` | 10 MB | - | 超过此大小会警告 |

**超过限制时的行为:**
- 工作流继续执行,但无法添加新事件
- 新事件尝试会失败,工作流被终止
- 状态变为 `WORKFLOW_EXECUTION_STATUS_FAILED`
- 错误信息: `Event history size limit exceeded`

**如何避免超限:**

1. **使用 Continue-As-New**
```go
func LongRunningWorkflow(ctx workflow.Context, iteration int) error {
    // 每 1000 次迭代重置历史
    if iteration >= 1000 {
        return workflow.NewContinueAsNewError(ctx, LongRunningWorkflow, 0)
    }
    
    // 执行任务
    for i := 0; i < 100; i++ {
        workflow.ExecuteActivity(ctx, Task, i).Get(ctx, nil)
    }
    
    return LongRunningWorkflow(ctx, iteration+1)
}
```

2. **减少 Activity 数量**
   - 合并小任务到单个 Activity
   - 使用 Local Activity(不记录到 History)

3. **优化重试策略**
   - 减少最大重试次数
   - 每次重试都会增加 3-4 个事件

**日志获取实现:**

Temporal 不直接提供日志 API,需要从 Event History 重建:

```go
// 从 Event History 重建日志
func extractLogsFromHistory(events []*history.HistoryEvent) []LogEntry {
    logs := make([]LogEntry, 0)
    
    for _, event := range events {
        var logEntry *LogEntry
        
        switch event.EventType {
        case enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED:
            logEntry = &LogEntry{
                Timestamp: event.EventTime.AsTime(),
                Level:     "info",
                Message:   "Workflow started",
            }
            
        case enums.EVENT_TYPE_ACTIVITY_TASK_STARTED:
            attrs := event.GetActivityTaskStartedEventAttributes()
            logEntry = &LogEntry{
                Timestamp: event.EventTime.AsTime(),
                Level:     "info",
                Job:       extractJobName(attrs.ActivityId),
                Step:      extractStepName(attrs.ActivityId),
                Message:   "Step started",
            }
            
        case enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
            attrs := event.GetActivityTaskCompletedEventAttributes()
            logEntry = &LogEntry{
                Timestamp: event.EventTime.AsTime(),
                Level:     "info",
                Job:       extractJobName(attrs.ActivityId),
                Step:      extractStepName(attrs.ActivityId),
                Message:   "Step completed",
            }
            
        case enums.EVENT_TYPE_ACTIVITY_TASK_FAILED:
            attrs := event.GetActivityTaskFailedEventAttributes()
            logEntry = &LogEntry{
                Timestamp: event.EventTime.AsTime(),
                Level:     "error",
                Job:       extractJobName(attrs.ActivityId),
                Step:      extractStepName(attrs.ActivityId),
                Message:   "Step failed",
                Error:     attrs.Failure.Message,
            }
        }
        
        if logEntry != nil {
            logs = append(logs, *logEntry)
        }
    }
    
    return logs
}
```

**实时日志流(Server-Sent Events):**

```go
// 使用长轮询获取新事件
func streamLogs(w http.ResponseWriter, workflowID string) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    flusher, _ := w.(http.Flusher)
    
    // 长轮询模式
    iter := client.GetWorkflowHistory(ctx, workflowID, "", true, 0)
    
    for iter.HasNext() {
        event, err := iter.Next()
        if err != nil {
            break
        }
        
        // 提取日志
        logEntry := extractLogFromEvent(event)
        if logEntry != nil {
            // 发送 SSE
            data, _ := json.Marshal(logEntry)
            fmt.Fprintf(w, "data: %s\n\n", data)
            flusher.Flush()
        }
    }
}
```

**性能考虑:**
- 小型工作流(< 100 events): < 50ms
- 中型工作流(100-1000 events): 100-500ms
- 大型工作流(1000-5000 events): 500ms-2s
- 巨型工作流(5000+ events): 2s+

**优化建议:**
- 使用分页,每次只获取最新的 N 条日志
- 缓存历史日志,仅获取增量
- 对于长时运行工作流,考虑外部日志存储

---

## 2. 行业最佳实践对比

### 2.1 GitHub Actions API

**核心 API 端点:**

```bash
# 1. 列出 workflow runs
GET /repos/{owner}/{repo}/actions/runs
# 查询参数: status, actor, branch, event, created, per_page, page

# 2. 获取单个 workflow run
GET /repos/{owner}/{repo}/actions/runs/{run_id}

# 3. 列出 run 的 jobs
GET /repos/{owner}/{repo}/actions/runs/{run_id}/jobs

# 4. 获取 job 日志
GET /repos/{owner}/{repo}/actions/jobs/{job_id}/logs

# 5. 取消 workflow run
POST /repos/{owner}/{repo}/actions/runs/{run_id}/cancel

# 6. 重新运行 workflow run
POST /repos/{owner}/{repo}/actions/runs/{run_id}/rerun

# 7. 重新运行失败的 jobs
POST /repos/{owner}/{repo}/actions/runs/{run_id}/rerun-failed-jobs

# 8. 删除 workflow run
DELETE /repos/{owner}/{repo}/actions/runs/{run_id}

# 9. 下载 artifacts
GET /repos/{owner}/{repo}/actions/runs/{run_id}/artifacts
```

**API 响应示例:**

```json
// GET /repos/{owner}/{repo}/actions/runs/{run_id}
{
  "id": 30433642,
  "name": "Build",
  "node_id": "MDEwOldvcmtmbG93IFJ1bjMwNDMzNjQy",
  "head_branch": "main",
  "head_sha": "acb5820ced9479c074f688cc328bf03f341a511d",
  "run_number": 562,
  "event": "push",
  "status": "completed",
  "conclusion": "success",
  "workflow_id": 159038,
  "url": "https://api.github.com/repos/octo-org/octo-repo/actions/runs/30433642",
  "html_url": "https://github.com/octo-org/octo-repo/actions/runs/30433642",
  "created_at": "2020-01-22T19:33:08Z",
  "updated_at": "2020-01-22T19:33:08Z",
  "run_started_at": "2020-01-22T19:33:08Z",
  "jobs_url": "https://api.github.com/repos/octo-org/octo-repo/actions/runs/30433642/jobs",
  "logs_url": "https://api.github.com/repos/octo-org/octo-repo/actions/runs/30433642/logs",
  "check_suite_url": "https://api.github.com/repos/octo-org/octo-repo/check-suites/414944374",
  "artifacts_url": "https://api.github.com/repos/octo-org/octo-repo/actions/runs/30433642/artifacts",
  "cancel_url": "https://api.github.com/repos/octo-org/octo-repo/actions/runs/30433642/cancel",
  "rerun_url": "https://api.github.com/repos/octo-org/octo-repo/actions/runs/30433642/rerun"
}
```

**API 设计特点:**

1. **RESTful 资源层次**
   - `/runs/{run_id}` - Run 资源
   - `/runs/{run_id}/jobs` - 嵌套 Jobs 资源
   - `/jobs/{job_id}/logs` - Job 日志资源

2. **状态和结论分离**
   - `status`: queued, in_progress, completed
   - `conclusion`: success, failure, cancelled, skipped, timed_out

3. **HATEOAS 链接**
   - 响应包含相关资源的 URL (jobs_url, logs_url, cancel_url)

4. **操作端点**
   - 取消: `POST /runs/{run_id}/cancel`
   - 重新运行: `POST /runs/{run_id}/rerun`
   - 重新运行失败: `POST /runs/{run_id}/rerun-failed-jobs`

5. **删除历史**
   - 支持删除 Run 记录 (DELETE /runs/{run_id})

---

### 2.2 GitLab CI/CD API

**核心 API 端点:**

```bash
# 1. 列出 pipelines
GET /projects/{id}/pipelines
# 查询参数: status, ref, sha, username, updated_after, updated_before

# 2. 获取单个 pipeline
GET /projects/{id}/pipelines/{pipeline_id}

# 3. 列出 pipeline 的 jobs
GET /projects/{id}/pipelines/{pipeline_id}/jobs

# 4. 获取单个 job
GET /projects/{id}/jobs/{job_id}

# 5. 获取 job 日志
GET /projects/{id}/jobs/{job_id}/trace

# 6. 重试 pipeline
POST /projects/{id}/pipelines/{pipeline_id}/retry

# 7. 取消 pipeline
POST /projects/{id}/pipelines/{pipeline_id}/cancel

# 8. 重试单个 job
POST /projects/{id}/jobs/{job_id}/retry

# 9. 取消单个 job
POST /projects/{id}/jobs/{job_id}/cancel

# 10. 触发(手动) job
POST /projects/{id}/jobs/{job_id}/play

# 11. 删除 pipeline
DELETE /projects/{id}/pipelines/{pipeline_id}
```

**API 响应示例:**

```json
// GET /projects/{id}/pipelines/{pipeline_id}
{
  "id": 46,
  "iid": 11,
  "project_id": 1,
  "status": "success",
  "ref": "main",
  "sha": "a91957a858320c0e17f3a0eca7cfacbff50ea29a",
  "before_sha": "a91957a858320c0e17f3a0eca7cfacbff50ea29a",
  "tag": false,
  "yaml_errors": null,
  "user": {
    "name": "Administrator",
    "username": "root",
    "id": 1
  },
  "created_at": "2016-08-11T11:28:34.085Z",
  "updated_at": "2016-08-11T11:32:35.169Z",
  "started_at": "2016-08-11T11:32:35.169Z",
  "finished_at": "2016-08-11T11:32:35.145Z",
  "committed_at": null,
  "duration": 123,
  "coverage": "30.0",
  "web_url": "https://gitlab.example.com/foo/bar/pipelines/46"
}
```

**API 设计特点:**

1. **粒度控制**
   - Pipeline 级别操作(retry, cancel)
   - Job 级别操作(retry, cancel, play)

2. **手动触发**
   - `POST /jobs/{job_id}/play` - 触发手动 job
   - 支持需要审批的部署流程

3. **多状态管理**
   - Pipeline 状态: created, waiting_for_resource, preparing, pending, running, success, failed, canceled, skipped, manual
   - Job 状态: created, pending, running, success, failed, canceled, skipped, manual

4. **覆盖率数据**
   - Pipeline 包含 `coverage` 字段

5. **用户关联**
   - 记录触发用户信息

---

### 2.3 Argo Workflows API

**核心 API 端点:**

```bash
# 1. 列出 workflows
GET /api/v1/workflows/{namespace}
# 查询参数: listOptions.labelSelector, listOptions.fieldSelector

# 2. 获取单个 workflow
GET /api/v1/workflows/{namespace}/{name}

# 3. 暂停 workflow
PUT /api/v1/workflows/{namespace}/{name}/suspend

# 4. 恢复 workflow
PUT /api/v1/workflows/{namespace}/{name}/resume

# 5. 重试 workflow
PUT /api/v1/workflows/{namespace}/{name}/retry

# 6. 停止 workflow
PUT /api/v1/workflows/{namespace}/{name}/stop

# 7. 终止 workflow
PUT /api/v1/workflows/{namespace}/{name}/terminate

# 8. 删除 workflow
DELETE /api/v1/workflows/{namespace}/{name}

# 9. 获取 workflow 日志
GET /api/v1/workflows/{namespace}/{name}/log
# 查询参数: podName, containerName, follow

# 10. 重新提交 workflow
POST /api/v1/workflows/{namespace}/{name}/resubmit
```

**API 响应示例:**

```json
// GET /api/v1/workflows/{namespace}/{name}
{
  "metadata": {
    "name": "hello-world-kzm2d",
    "namespace": "default",
    "uid": "a1b2c3d4",
    "creationTimestamp": "2020-01-22T19:33:08Z"
  },
  "spec": {
    "entrypoint": "whalesay",
    "templates": [...]
  },
  "status": {
    "phase": "Succeeded",
    "startedAt": "2020-01-22T19:33:08Z",
    "finishedAt": "2020-01-22T19:33:30Z",
    "estimatedDuration": 22,
    "progress": "1/1",
    "nodes": {
      "hello-world-kzm2d": {
        "id": "hello-world-kzm2d",
        "name": "hello-world-kzm2d",
        "displayName": "hello-world-kzm2d",
        "type": "Pod",
        "phase": "Succeeded",
        "startedAt": "2020-01-22T19:33:08Z",
        "finishedAt": "2020-01-22T19:33:30Z"
      }
    }
  }
}
```

**API 设计特点:**

1. **暂停/恢复**
   - `PUT /suspend` - 暂停工作流
   - `PUT /resume` - 恢复工作流
   - 支持长时运行的人工审批流程

2. **停止 vs 终止**
   - `PUT /stop` - 优雅停止(完成当前 step)
   - `PUT /terminate` - 立即终止

3. **节点级别状态**
   - `status.nodes` - 每个节点的详细状态
   - `progress` - 进度字符串(如 "3/5")

4. **日志流式传输**
   - `follow=true` - 实时流式日志

5. **重新提交**
   - `POST /resubmit` - 使用相同参数重新提交
   - 与 "重试" 不同,创建新的工作流实例

---

### 2.4 对比矩阵

| 功能 | GitHub Actions | GitLab CI | Argo Workflows | Temporal 原生支持 | Waterflow 实现难度 |
|------|----------------|-----------|----------------|-------------------|-------------------|
| **基础操作** |
| 提交/创建工作流 | ✅ Webhook 触发 | ✅ Webhook 触发 | ✅ POST /workflows | ✅ ExecuteWorkflow | 🟢 简单 (已实现) |
| 获取单个状态 | ✅ GET /runs/{id} | ✅ GET /pipelines/{id} | ✅ GET /workflows/{name} | ✅ DescribeWorkflowExecution | 🟢 简单 (已实现) |
| 列表查询 | ✅ GET /runs | ✅ GET /pipelines | ✅ GET /workflows | ✅ ListWorkflow | 🟢 简单 (已实现) |
| 获取日志 | ✅ GET /jobs/{id}/logs | ✅ GET /jobs/{id}/trace | ✅ GET /log?follow=true | ⚠️ 从 History 重建 | 🟡 中等 (已实现) |
| **控制操作** |
| 取消工作流 | ✅ POST /runs/{id}/cancel | ✅ POST /pipelines/{id}/cancel | ✅ PUT /stop | ✅ CancelWorkflow | 🟢 简单 (已实现) |
| 终止工作流 | ❌ 无 | ❌ 无 | ✅ PUT /terminate | ✅ TerminateWorkflow | 🟢 简单 (未实现) |
| 暂停工作流 | ❌ 无 | ⚠️ 手动 Job | ✅ PUT /suspend | ⚠️ 需 Signal | 🔴 困难 (需工作流代码支持) |
| 恢复工作流 | ❌ 无 | ❌ 无 | ✅ PUT /resume | ⚠️ 需 Signal | 🔴 困难 (需工作流代码支持) |
| **重试操作** |
| 重新运行整个工作流 | ✅ POST /runs/{id}/rerun | ✅ POST /pipelines/{id}/retry | ✅ PUT /retry | ⚠️ 重新提交 | 🟢 简单 (已实现) |
| 重新运行失败的 Jobs | ✅ POST /rerun-failed-jobs | ✅ POST /jobs/{id}/retry | ⚠️ 需手动 | ❌ 无 | 🔴 困难 (需工作流支持) |
| 从特定 Step 重试 | ❌ 无 | ❌ 无 | ⚠️ 修改 YAML | ❌ 无 | 🔴 困难 (架构限制) |
| **高级功能** |
| 删除工作流记录 | ✅ DELETE /runs/{id} | ✅ DELETE /pipelines/{id} | ✅ DELETE /workflows/{name} | ❌ 无 (Event Sourcing) | 🔴 不推荐 (违背 Event Sourcing) |
| 获取 Artifacts | ✅ GET /artifacts | ✅ GET /jobs/{id}/artifacts | ✅ 通过 S3/Minio | ❌ 需自行存储 | 🟡 中等 (未来功能) |
| 手动审批 | ⚠️ 环境保护规则 | ✅ Manual Jobs | ✅ Suspend/Resume | ⚠️ Signal + Query | 🔴 困难 (需复杂设计) |
| 实时日志流 | ❌ 无 (轮询) | ✅ WebSocket | ✅ follow=true | ⚠️ 长轮询 History | 🟡 中等 (SSE 实现) |
| 按名称/标签搜索 | ✅ 丰富过滤 | ✅ 丰富过滤 | ✅ labelSelector | ⚠️ 需搜索属性 | 🟡 中等 (需配置) |

**图例:**
- ✅ 原生支持
- ⚠️ 部分支持或需额外工作
- ❌ 不支持
- 🟢 简单 (< 1 天)
- 🟡 中等 (1-3 天)
- 🔴 困难 (> 3 天 或 需架构变更)

---

## 3. Waterflow 架构适配分析

### 3.1 Event Sourcing 架构影响

#### 核心特征

Waterflow 完全依赖 Temporal 的 Event Sourcing 架构:
- **所有状态存储在 Event History** - 无额外数据库
- **状态从事件重建** - 每次查询都重放事件
- **不可变历史** - 事件一旦写入不可修改

#### 对 API 设计的影响

**1. 删除操作的不可行性**

```
问题: 用户期望 DELETE /v1/workflows/{id} 删除工作流记录
现实: Temporal Event History 不可删除(设计原则)
```

**影响:**
- ❌ 不能提供 DELETE API
- ⚠️ 可以提供 "归档" 功能(修改可见性查询)
- ✅ Event History 保留完整审计追踪

**替代方案:**

```go
// 方案 A: 软删除(使用搜索属性)
workflow.UpsertSearchAttributes(ctx, map[string]interface{}{
    "Archived": true,
})

// 列表查询时过滤
query := "WorkflowType = 'waterflow-executor' AND Archived != true"
```

```http
# 方案 B: 归档 API (不删除,只隐藏)
POST /v1/workflows/{id}/archive
Response: 204 No Content

# 列表查询默认不显示归档的
GET /v1/workflows
# 显示归档的
GET /v1/workflows?show_archived=true
```

**推荐:** 方案 B (归档 API)
- 符合 Event Sourcing 原则
- 保留完整历史
- 满足用户"清理"需求

**2. 状态持久化方式**

```
传统系统: 数据库存储 { workflow_id, status, created_at, ... }
Waterflow: Event History 重建状态
```

**优势:**
- ✅ 无状态 Server,易扩展
- ✅ 完整的执行历史可追溯
- ✅ 时间旅行调试(查看任意时刻状态)

**挑战:**
- ⚠️ 查询性能依赖 Temporal 索引
- ⚠️ 复杂查询需要遍历 Event History
- ⚠️ 大型工作流的 History 解析开销

**API 设计建议:**

```go
// 状态查询 - 快速路径(DescribeWorkflowExecution)
GET /v1/workflows/{id}
Response Time: < 100ms
Data: 基本状态(status, start_time, end_time)

// 详细状态 - 慢速路径(GetWorkflowHistory + 解析)
GET /v1/workflows/{id}?include=jobs,steps
Response Time: 100-500ms
Data: Job/Step 级别详情

// 日志查询 - 最慢路径(完整 History 解析)
GET /v1/workflows/{id}/logs
Response Time: 500ms-2s
Data: 重建的日志条目
```

**3. 历史数据的不可变性**

**场景:** 用户希望修改已完成工作流的元数据

```http
# 期望: 修改工作流名称
PATCH /v1/workflows/{id}
{
  "name": "New Name"
}
```

**现实:**
- ❌ Event History 不可修改
- ⚠️ 可以通过 Memo 存储元数据(但仅限工作流启动时)

**替代方案:**

```go
// 方案 A: 启动时设置 Memo (不可变)
client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
    Memo: map[string]interface{}{
        "user_label": "Deploy Production",
        "environment": "production",
    },
})

// 方案 B: 外部元数据存储(如果确实需要可变元数据)
// 在 Server 维护一个轻量级 KV 存储
// workflow_id -> { user_label, tags, notes }
```

**推荐:** 方案 A (不可变 Memo) for MVP
- 符合 Event Sourcing 原则
- 简化架构
- 大多数场景不需要修改元数据

---

### 3.2 单节点执行模式

#### 架构特征

Waterflow 的执行模式:
- **每个 Step = 1 个 Temporal Activity**
- **单个 Agent Worker 执行 Activity**
- **Activity 是最小执行单元**

#### 对重试功能的影响

**1. Step 级别重试的可行性**

```yaml
jobs:
  deploy:
    steps:
      - name: Build       # Step 1 - 成功
      - name: Test        # Step 2 - 成功
      - name: Deploy      # Step 3 - 失败
```

**用户期望:** "只重试失败的 Deploy Step"

**Temporal 限制:**
- Activity 重试是自动的(RetryPolicy 控制)
- 手动重试只能重新执行整个 Workflow
- 无法跳过成功的 Activity 直接执行失败的

**解决方案对比:**

| 方案 | 实现难度 | 用户体验 | 架构影响 |
|------|----------|----------|----------|
| **A. 重新运行整个工作流** | 🟢 简单 | ⚠️ 重复执行成功的 Step | 无影响 |
| **B. Workflow 支持跳过已成功 Step** | 🟡 中等 | ✅ 只执行失败的 Step | 需修改 Workflow 代码 |
| **C. 拆分 Job 为多个 Child Workflow** | 🔴 困难 | ✅ 可单独重试 Job | 架构复杂化 |

**推荐方案 B - 条件跳过:**

```go
// Workflow 实现
func DeployWorkflow(ctx workflow.Context, input DeployInput) error {
    // 获取重试上下文(如果是重试)
    retryContext := input.RetryContext
    
    for _, step := range input.Steps {
        // 检查是否需要跳过
        if shouldSkipStep(step, retryContext) {
            logger.Info("Skipping already succeeded step", step.Name)
            continue
        }
        
        // 执行 Step
        err := workflow.ExecuteActivity(ctx, ExecuteStep, step).Get(ctx, nil)
        if err != nil {
            return err
        }
    }
}

func shouldSkipStep(step Step, retryContext *RetryContext) bool {
    if retryContext == nil {
        return false // 不是重试,正常执行
    }
    
    // 检查 step 是否在成功列表中
    for _, succeededStep := range retryContext.SucceededSteps {
        if step.Name == succeededStep {
            return true
        }
    }
    
    return false
}
```

**API 设计:**

```http
# 重新运行工作流(跳过成功的 Step)
POST /v1/workflows/{id}/rerun
{
  "from_step": "deploy",  // 从此 Step 开始
  "skip_steps": ["build", "test"]  // 或显式跳过
}

Response:
{
  "id": "new-workflow-id",  // 新的工作流 ID
  "original_id": "original-workflow-id",
  "skipped_steps": ["build", "test"],
  "status": "running"
}
```

**实现步骤:**

1. 查询原工作流的 Event History
2. 提取成功的 Step 列表
3. 使用原 YAML + RetryContext 提交新工作流
4. 新工作流跳过成功的 Step

**限制:**
- ⚠️ 幂等性要求:跳过的 Step 必须是幂等的
- ⚠️ 依赖关系:如果 Step 有副作用,跳过可能导致问题
- ⚠️ 用户理解成本:需要明确说明这是创建新工作流

**2. Activity 级别的控制**

**Temporal Activity 特性:**
- **超时控制** - StartToCloseTimeout, ScheduleToCloseTimeout
- **重试策略** - RetryPolicy (最大重试次数、重试间隔)
- **心跳机制** - Activity.RecordHeartbeat()
- **取消传播** - Activity 自动接收工作流取消信号

**Waterflow 如何利用:**

```yaml
# YAML DSL 映射到 Activity Options
jobs:
  deploy:
    timeout: 30m  # Workflow ExecutionTimeout
    steps:
      - name: Deploy
        uses: docker/exec@v1
        timeout: 10m  # Activity StartToCloseTimeout
        retry:
          max_attempts: 3  # RetryPolicy.MaximumAttempts
          backoff: 1m  # RetryPolicy.InitialInterval
```

**生成的 Activity Options:**

```go
activityOptions := workflow.ActivityOptions{
    StartToCloseTimeout: 10 * time.Minute,
    RetryPolicy: &temporal.RetryPolicy{
        MaximumAttempts: 3,
        InitialInterval: 1 * time.Minute,
    },
}
```

**API 可见性:**

```http
# 查询 Step 状态时,显示重试信息
GET /v1/workflows/{id}
{
  "jobs": [{
    "steps": [{
      "name": "Deploy",
      "status": "failed",
      "attempts": 3,  // 从 Event History 提取
      "last_error": "Connection timeout",
      "next_retry_at": "2026-01-27T10:35:00Z"
    }]
  }]
}
```

---

### 3.3 YAML DSL 特性

#### 核心特性

**1. Job 依赖图 (DAG)**

```yaml
jobs:
  build:
    runs-on: default
  test:
    needs: [build]  # 依赖 build
  deploy:
    needs: [test]   # 依赖 test
```

**Temporal 实现:**
- Job = Child Workflow
- DAG 通过父 Workflow 编排
- 依赖关系通过 `workflow.ExecuteChildWorkflow` 的顺序控制

**对重试的影响:**

```
场景: deploy Job 失败,用户希望重试
问题: build 和 test Job 已成功,是否需要重新执行?
```

**解决方案:**

```go
// 方案 A: 重新执行依赖链(简单但低效)
重试 deploy -> 重新执行 build, test, deploy

// 方案 B: 仅重试失败的 Job(推荐)
重试 deploy -> 跳过 build, test,仅执行 deploy

// 实现: 在 Workflow Input 中传递已完成的 Job 列表
type WorkflowInput struct {
    Jobs            []Job
    CompletedJobs   []string  // ["build", "test"]
}

// 执行时检查
for _, job := range input.Jobs {
    if contains(input.CompletedJobs, job.Name) {
        continue  // 跳过
    }
    
    // 执行 Job
    workflow.ExecuteChildWorkflow(ctx, JobWorkflow, job)
}
```

**2. Matrix 并行执行**

```yaml
jobs:
  deploy:
    strategy:
      matrix:
        env: [dev, staging, prod]
        region: [us, eu]
    runs-on: server-${{ matrix.env }}
```

**Temporal 实现:**
- 每个 Matrix 组合 = 1 个 Child Workflow
- 并发执行,等待全部完成

**对重试的影响:**

```
场景: Matrix 中某个组合失败
用户期望: 只重试失败的组合
```

**解决方案:**

```go
// 记录失败的 Matrix 组合
type MatrixStatus struct {
    Combination map[string]string  // {env: "prod", region: "eu"}
    Status      string
    Error       string
}

// 重试时只执行失败的
for _, combo := range matrixCombinations {
    if isCompleted(combo, retryContext) {
        continue
    }
    
    // 执行
    workflow.ExecuteChildWorkflow(ctx, JobWorkflow, combo)
}
```

**API 设计:**

```http
# 重试失败的 Matrix 组合
POST /v1/workflows/{id}/rerun
{
  "failed_only": true,  // 仅重试失败的组合
  "matrix_filter": {
    "env": "prod",  // 可选:仅重试特定组合
    "region": "eu"
  }
}
```

**3. 从特定 Job 重新执行的需求**

**用户场景:**

```yaml
jobs:
  build:    # 成功
  test:     # 成功
  deploy:   # 失败
  notify:   # 未执行
```

**期望:** "从 deploy Job 开始重新执行,跳过 build 和 test"

**实现难度:** 🟡 中等

**解决方案:**

```http
POST /v1/workflows/{id}/rerun
{
  "from_job": "deploy"
}
```

**实现逻辑:**

1. 查询原工作流的 Event History
2. 提取 Job 依赖图
3. 确定需要执行的 Job 列表:
   - 包含 `from_job` 及其所有下游 Job
   - 排除 `from_job` 的上游 Job
4. 使用原 YAML + `skip_jobs` 提交新工作流

```go
// 计算需要执行的 Job
func calculateJobsToRun(dag DAG, fromJob string) []string {
    // 1. 找到 from_job 在 DAG 中的位置
    // 2. 提取所有下游节点(包括 from_job 自己)
    // 3. 排除上游节点
    
    downstream := dag.GetDownstreamJobs(fromJob)
    downstream = append(downstream, fromJob)
    return downstream
}
```

**限制:**
- ⚠️ 依赖关系必须明确定义(needs 字段)
- ⚠️ 如果 Job 有副作用,跳过上游可能导致问题
- ⚠️ 用户需要理解这是创建新工作流

---

### 3.4 多触发方式统一

#### 触发方式

Waterflow 支持 3 种触发方式:

**1. 手动提交 (workflow_dispatch)**

```yaml
name: Deploy App
on:
  workflow_dispatch:
    inputs:
      environment:
        description: 'Target environment'
        required: true
        default: 'staging'
```

**API:**
```http
POST /v1/workflows
{
  "yaml": "...",
  "vars": {
    "environment": "production"
  }
}
```

**Temporal 实现:**
```go
workflowID := uuid.New().String()
client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
    ID: workflowID,
    TaskQueue: "default",
}, DeployWorkflow, input)
```

**2. Schedule 定时触发**

```yaml
name: Daily Backup
on:
  schedule:
    - cron: "0 2 * * *"  # 每天凌晨 2 点
```

**API:**
```http
# Story 1.10 - Schedule API
POST /v1/schedules
{
  "workflow_yaml": "...",
  "cron": "0 2 * * *",
  "timezone": "America/Los_Angeles"
}
```

**Temporal 实现:**
```go
// 使用 Temporal Schedules
client.ScheduleClient().Create(ctx, client.ScheduleOptions{
    ID: "daily-backup",
    Spec: client.ScheduleSpec{
        CronExpressions: []string{"0 2 * * *"},
    },
    Action: &client.ScheduleWorkflowAction{
        Workflow: DeployWorkflow,
        Args:     []interface{}{input},
    },
})
```

**3. Webhook 事件触发**

```yaml
name: Deploy on Push
on:
  webhook:
    events: [push]
    filters:
      branch: main
```

**API:**
```http
# Story 1.11 - Webhook API
POST /v1/webhooks/{webhook_id}
{
  "event": "push",
  "branch": "main",
  "commit": "abc123"
}
```

**Temporal 实现:**
```go
// Webhook Handler 触发工作流
workflowID := fmt.Sprintf("webhook-%s-%s", event.Type, event.CommitSHA)
client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
    ID: workflowID,
    TaskQueue: "default",
}, DeployWorkflow, input)
```

#### API 统一设计

**问题:** 不同触发方式如何统一管理?

**解决方案: 统一的工作流元数据**

```go
type WorkflowMetadata struct {
    TriggerType   string  // "manual", "schedule", "webhook"
    TriggerSource string  // schedule_id, webhook_id, user_id
    TriggerEvent  map[string]interface{}  // 触发事件详情
}

// 存储到 Temporal SearchAttributes
workflow.UpsertSearchAttributes(ctx, map[string]interface{}{
    "TriggerType":   metadata.TriggerType,
    "TriggerSource": metadata.TriggerSource,
})
```

**API 查询:**

```http
# 列出所有工作流(不区分触发方式)
GET /v1/workflows

# 仅列出手动触发的
GET /v1/workflows?trigger_type=manual

# 仅列出定时触发的
GET /v1/workflows?trigger_type=schedule&trigger_source=daily-backup
```

**统一的工作流响应:**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Deploy App",
  "status": "running",
  "trigger": {
    "type": "webhook",
    "source": "github-push",
    "event": {
      "branch": "main",
      "commit": "abc123",
      "author": "user@example.com"
    }
  },
  "created_at": "2026-01-27T10:30:00Z"
}
```

**好处:**
- ✅ API 一致性:所有工作流使用相同的端点查询
- ✅ 易于过滤:按触发类型/来源筛选
- ✅ 完整元数据:保留触发上下文

---

## 4. API 设计建议

### 4.1 推荐的 API 端点清单

#### MVP 优先级(必须实现)

| 优先级 | HTTP 方法 | 端点 | 用途 | 状态 |
|--------|-----------|------|------|------|
| **P0** | POST | `/v1/workflows` | 提交工作流 | ✅ 已实现 |
| **P0** | GET | `/v1/workflows/{id}` | 查询工作流状态 | ✅ 已实现 |
| **P0** | GET | `/v1/workflows` | 列出工作流 | ✅ 已实现 |
| **P0** | GET | `/v1/workflows/{id}/logs` | 获取工作流日志 | ✅ 已实现 |
| **P0** | POST | `/v1/workflows/{id}/cancel` | 取消工作流 | ✅ 已实现 |
| **P0** | POST | `/v1/workflows/{id}/rerun` | 重新运行工作流 | ✅ 已实现 |

#### Post-MVP 扩展(增强功能)

| 优先级 | HTTP 方法 | 端点 | 用途 | 实现难度 |
|--------|-----------|------|------|----------|
| **P1** | POST | `/v1/workflows/{id}/terminate` | 强制终止工作流 | 🟢 简单 |
| **P1** | POST | `/v1/workflows/{id}/pause` | 暂停工作流 | 🔴 困难 (需 Workflow 支持) |
| **P1** | POST | `/v1/workflows/{id}/resume` | 恢复工作流 | 🔴 困难 (需 Workflow 支持) |
| **P1** | GET | `/v1/workflows/{id}/pause-status` | 查询暂停状态 | 🔴 困难 (需 Query 支持) |
| **P2** | POST | `/v1/workflows/{id}/archive` | 归档工作流 | 🟡 中等 (需搜索属性) |
| **P2** | GET | `/v1/workflows/{id}/events` | 获取原始 Event History | 🟢 简单 |
| **P2** | POST | `/v1/workflows/{id}/signal` | 发送自定义信号 | 🟡 中等 (需 Workflow 支持) |
| **P3** | POST | `/v1/workflows/{id}/reset` | 重置到历史点 | 🔴 困难 (高级功能) |
| **P3** | POST | `/v1/workflows/{id}/retry-failed-jobs` | 仅重试失败的 Job | 🔴 困难 (需复杂逻辑) |

#### 完整 API 设计

**1. 提交工作流 [P0 - 已实现]**

```http
POST /v1/workflows
Content-Type: application/json

Request:
{
  "yaml": "name: Deploy\njobs:\n  build:\n    steps:...",
  "vars": {
    "environment": "production"
  }
}

Response: 201 Created
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "run_id": "temporal-run-id-123",
  "name": "Deploy",
  "status": "running",
  "created_at": "2026-01-27T10:30:00Z",
  "url": "/v1/workflows/550e8400-e29b-41d4-a716-446655440000"
}
```

**2. 查询工作流状态 [P0 - 已实现]**

```http
GET /v1/workflows/{id}
GET /v1/workflows/{id}?include=jobs,steps

Response: 200 OK
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "run_id": "temporal-run-id-123",
  "name": "Deploy",
  "status": "running",
  "created_at": "2026-01-27T10:30:00Z",
  "started_at": "2026-01-27T10:30:01Z",
  "completed_at": null,
  "duration_seconds": 125,
  "trigger": {
    "type": "manual",
    "source": "user@example.com"
  },
  "jobs": [
    {
      "id": "build",
      "name": "build",
      "status": "completed",
      "started_at": "2026-01-27T10:30:02Z",
      "completed_at": "2026-01-27T10:31:00Z",
      "runs_on": "default",
      "steps": [
        {
          "name": "Build",
          "status": "completed",
          "started_at": "2026-01-27T10:30:02Z",
          "completed_at": "2026-01-27T10:31:00Z"
        }
      ]
    }
  ]
}
```

**3. 列出工作流 [P0 - 已实现]**

```http
GET /v1/workflows?page=1&limit=20&status=running&name=Deploy&created_after=2026-01-01T00:00:00Z

Response: 200 OK
{
  "workflows": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Deploy",
      "status": "running",
      "created_at": "2026-01-27T10:30:00Z",
      "started_at": "2026-01-27T10:30:01Z",
      "duration_seconds": 125
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 42,
    "total_pages": 3
  }
}
```

**查询参数:**
- `page` - 页码(默认 1)
- `limit` - 每页数量(默认 20,最大 100)
- `status` - 状态过滤(pending, running, completed, failed, cancelled)
- `name` - 工作流名称(精确匹配)
- `created_after` - 创建时间下界(ISO 8601)
- `created_before` - 创建时间上界(ISO 8601)
- `trigger_type` - 触发类型(manual, schedule, webhook)

**4. 获取工作流日志 [P0 - 已实现]**

```http
GET /v1/workflows/{id}/logs
GET /v1/workflows/{id}/logs?level=error&job=deploy&tail=100

Response: 200 OK
Content-Type: application/x-ndjson

{"timestamp":"2026-01-27T10:30:01Z","level":"info","message":"Workflow started"}
{"timestamp":"2026-01-27T10:30:02Z","level":"info","job":"build","step":"Build","message":"Step started"}
{"timestamp":"2026-01-27T10:31:00Z","level":"info","job":"build","step":"Build","message":"Step completed"}
```

**查询参数:**
- `level` - 日志级别(info, warn, error, debug)
- `job` - Job 名称过滤
- `step` - Step 名称过滤
- `tail` - 返回最后 N 行(默认 100,最大 1000)

**实时日志流:**

```http
GET /v1/workflows/{id}/logs?stream=true
Accept: text/event-stream

Response: 200 OK
Content-Type: text/event-stream

data: {"timestamp":"2026-01-27T10:30:01Z","level":"info","message":"Workflow started"}

data: {"timestamp":"2026-01-27T10:30:02Z","level":"info","job":"build","message":"Step started"}
```

**5. 取消工作流 [P0 - 已实现]**

```http
POST /v1/workflows/{id}/cancel

Response: 204 No Content
```

**错误响应:**

```json
// 404 Not Found - 工作流不存在
{
  "error": {
    "code": "not_found",
    "message": "Workflow not found",
    "details": {
      "workflow_id": "invalid-id"
    }
  }
}

// 409 Conflict - 工作流已完成
{
  "error": {
    "code": "conflict",
    "message": "Cannot cancel completed workflow",
    "details": {
      "workflow_id": "550e8400-e29b-41d4-a716-446655440000",
      "current_status": "completed"
    }
  }
}
```

**6. 重新运行工作流 [P0 - 已实现]**

```http
POST /v1/workflows/{id}/rerun
Content-Type: application/json

Request:
{
  "skip_steps": ["build", "test"],  // 可选:跳过特定 Step
  "from_job": "deploy",  // 可选:从特定 Job 开始
  "vars": {  // 可选:覆盖变量
    "environment": "staging"
  }
}

Response: 201 Created
{
  "id": "new-workflow-id",
  "original_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Deploy",
  "status": "running",
  "created_at": "2026-01-27T11:00:00Z",
  "url": "/v1/workflows/new-workflow-id"
}
```

**7. 强制终止工作流 [P1 - 未实现]**

```http
POST /v1/workflows/{id}/terminate
Content-Type: application/json

Request:
{
  "reason": "Workflow stuck, manual termination required"
}

Response: 204 No Content
```

**权限要求:** 管理员权限

**8. 暂停/恢复工作流 [P1 - 未实现]**

```http
# 暂停
POST /v1/workflows/{id}/pause
{
  "reason": "Manual pause for debugging"
}
Response: 204 No Content

# 恢复
POST /v1/workflows/{id}/resume
Response: 204 No Content

# 查询暂停状态
GET /v1/workflows/{id}/pause-status
Response: 200 OK
{
  "paused": true,
  "reason": "Manual pause for debugging",
  "paused_at": "2026-01-27T10:35:00Z",
  "paused_by": "user@example.com"
}
```

**前提条件:** Workflow 代码必须实现暂停逻辑

**9. 归档工作流 [P2 - 未实现]**

```http
POST /v1/workflows/{id}/archive
Response: 204 No Content

# 取消归档
POST /v1/workflows/{id}/unarchive
Response: 204 No Content

# 列表默认不显示归档的
GET /v1/workflows
# 显示归档的
GET /v1/workflows?show_archived=true
```

**10. 获取原始 Event History [P2 - 未实现]**

```http
GET /v1/workflows/{id}/events
GET /v1/workflows/{id}/events?page_size=100&filter=activity

Response: 200 OK
{
  "events": [
    {
      "event_id": 1,
      "event_type": "WorkflowExecutionStarted",
      "event_time": "2026-01-27T10:30:00Z",
      "attributes": {...}
    },
    {
      "event_id": 2,
      "event_type": "ActivityTaskScheduled",
      "event_time": "2026-01-27T10:30:01Z",
      "attributes": {...}
    }
  ],
  "next_page_token": "..."
}
```

**用途:** 高级调试,查看 Temporal 原始事件

---

### 4.2 完整验收标准

#### AC1: 工作流提交 API

**Given** REST API 服务和 Temporal 集成已完成  
**When** POST `/v1/workflows` 请求带有 YAML 内容和可选变量  
**Then** 返回 201 Created 和工作流信息(id, run_id, name, status, created_at, url)  
**And** 工作流 ID 使用 UUID v4(全局唯一)  
**And** 工作流提交到 Temporal 执行队列  
**And** 响应时间 < 500ms  
**And** 支持变量覆盖(`vars` 字段)  

**错误处理:**
- 请求体缺失 → 400 Bad Request
- JSON 格式错误 → 400 Bad Request
- YAML 验证失败 → 422 Unprocessable Entity
- Temporal 不可用 → 503 Service Unavailable

**测试用例:**

```go
func TestSubmitWorkflow_Success(t *testing.T) {
    req := SubmitWorkflowRequest{
        YAML: validYAML,
        Vars: map[string]interface{}{"env": "production"},
    }
    
    resp := POST("/v1/workflows", req)
    
    assert.Equal(t, 201, resp.StatusCode)
    assert.NotEmpty(t, resp.Body.ID)
    assert.Equal(t, "Deploy App", resp.Body.Name)
    assert.Equal(t, "running", resp.Body.Status)
}

func TestSubmitWorkflow_InvalidYAML(t *testing.T) {
    req := SubmitWorkflowRequest{YAML: "invalid: yaml: :"}
    
    resp := POST("/v1/workflows", req)
    
    assert.Equal(t, 422, resp.StatusCode)
    assert.Equal(t, "validation_error", resp.Body.Error.Code)
}
```

#### AC2: 工作流查询 API

**Given** 工作流已提交并执行  
**When** GET `/v1/workflows/{id}` 查询工作流  
**Then** 返回 200 和完整状态  
**And** status 字段取值: pending, running, completed, failed, cancelled, timeout  
**And** conclusion 字段取值(仅 status=completed): success, failure, cancelled, timeout  
**And** 返回执行进度(当前 Job/Step)  
**And** 返回时间信息(created_at, started_at, completed_at, duration_seconds)  
**And** 工作流不存在 → 404 Not Found  
**And** 响应时间 < 200ms  

**可选参数:**
- `include=jobs` - 包含 Job 级别详情
- `include=jobs,steps` - 包含 Step 级别详情

**测试用例:**

```go
func TestGetWorkflowStatus_Running(t *testing.T) {
    workflowID := submitTestWorkflow(t)
    
    resp := GET(fmt.Sprintf("/v1/workflows/%s", workflowID))
    
    assert.Equal(t, 200, resp.StatusCode)
    assert.Equal(t, "running", resp.Body.Status)
    assert.NotEmpty(t, resp.Body.StartedAt)
    assert.Nil(t, resp.Body.CompletedAt)
}

func TestGetWorkflowStatus_NotFound(t *testing.T) {
    resp := GET("/v1/workflows/invalid-id")
    
    assert.Equal(t, 404, resp.StatusCode)
    assert.Equal(t, "not_found", resp.Body.Error.Code)
}
```

#### AC3: 工作流列表查询 API

**Given** 系统中存在多个工作流  
**When** GET `/v1/workflows?page=1&limit=20&status=running&name=Deploy` 查询列表  
**Then** 返回 200 和分页结果  
**And** 支持查询参数: page, limit, status, name, created_after, created_before  
**And** 默认按创建时间倒序排列(最新在前)  
**And** 参数验证: page >= 1, 1 <= limit <= 100  
**And** 参数错误 → 400 Bad Request  
**And** 响应时间 < 300ms  

**测试用例:**

```go
func TestListWorkflows_Pagination(t *testing.T) {
    // 创建 50 个工作流
    for i := 0; i < 50; i++ {
        submitTestWorkflow(t)
    }
    
    resp := GET("/v1/workflows?page=1&limit=20")
    
    assert.Equal(t, 200, resp.StatusCode)
    assert.Len(t, resp.Body.Workflows, 20)
    assert.Equal(t, 50, resp.Body.Pagination.Total)
    assert.Equal(t, 3, resp.Body.Pagination.TotalPages)
}

func TestListWorkflows_FilterByStatus(t *testing.T) {
    resp := GET("/v1/workflows?status=running")
    
    assert.Equal(t, 200, resp.StatusCode)
    for _, wf := range resp.Body.Workflows {
        assert.Equal(t, "running", wf.Status)
    }
}
```

#### AC4: 工作流日志查询 API

**Given** 工作流正在执行或已完成  
**When** GET `/v1/workflows/{id}/logs` 请求日志  
**Then** 返回 200 和 JSON Lines 格式日志  
**And** 日志包含字段: timestamp, level, job, step, message, error  
**And** 支持查询参数: level, job, step, tail  
**And** 历史日志从 Temporal Event History 重建  
**And** 工作流不存在 → 404 Not Found  
**And** 响应时间 < 500ms(历史日志)  

**实时日志流:**
- `GET /v1/workflows/{id}/logs?stream=true`
- 返回 `text/event-stream` 格式
- 使用 Server-Sent Events 推送新日志

**测试用例:**

```go
func TestGetWorkflowLogs_Success(t *testing.T) {
    workflowID := submitTestWorkflow(t)
    waitForCompletion(t, workflowID)
    
    resp := GET(fmt.Sprintf("/v1/workflows/%s/logs", workflowID))
    
    assert.Equal(t, 200, resp.StatusCode)
    assert.Greater(t, len(resp.Body), 0)
    
    // 验证日志格式
    lines := strings.Split(resp.Body, "\n")
    for _, line := range lines {
        var entry LogEntry
        json.Unmarshal([]byte(line), &entry)
        assert.NotEmpty(t, entry.Timestamp)
        assert.NotEmpty(t, entry.Level)
    }
}

func TestGetWorkflowLogs_Filter(t *testing.T) {
    workflowID := submitTestWorkflow(t)
    
    resp := GET(fmt.Sprintf("/v1/workflows/%s/logs?level=error", workflowID))
    
    // 验证所有日志都是 error 级别
    lines := strings.Split(resp.Body, "\n")
    for _, line := range lines {
        var entry LogEntry
        json.Unmarshal([]byte(line), &entry)
        assert.Equal(t, "error", entry.Level)
    }
}
```

#### AC5: 取消工作流 API

**Given** 工作流正在运行  
**When** POST `/v1/workflows/{id}/cancel` 请求取消  
**Then** 返回 204 No Content  
**And** 工作流接收到取消信号  
**And** 工作流状态变为 cancelled  
**And** 工作流不存在 → 404 Not Found  
**And** 工作流已完成 → 409 Conflict  

**测试用例:**

```go
func TestCancelWorkflow_Success(t *testing.T) {
    workflowID := submitLongRunningWorkflow(t)
    
    resp := POST(fmt.Sprintf("/v1/workflows/%s/cancel", workflowID), nil)
    
    assert.Equal(t, 204, resp.StatusCode)
    
    // 验证状态变为 cancelled
    status := GET(fmt.Sprintf("/v1/workflows/%s", workflowID))
    assert.Equal(t, "cancelled", status.Body.Status)
}

func TestCancelWorkflow_AlreadyCompleted(t *testing.T) {
    workflowID := submitTestWorkflow(t)
    waitForCompletion(t, workflowID)
    
    resp := POST(fmt.Sprintf("/v1/workflows/%s/cancel", workflowID), nil)
    
    assert.Equal(t, 409, resp.StatusCode)
    assert.Equal(t, "conflict", resp.Body.Error.Code)
}
```

#### AC6: 重新运行工作流 API

**Given** 工作流已完成(成功或失败)  
**When** POST `/v1/workflows/{id}/rerun` 请求重新运行  
**Then** 返回 201 Created 和新工作流信息  
**And** 新工作流 ID 与原工作流不同  
**And** 响应包含 original_id 字段  
**And** 支持可选参数: skip_steps, from_job, vars  
**And** 工作流不存在 → 404 Not Found  
**And** 工作流正在运行 → 409 Conflict  

**测试用例:**

```go
func TestRerunWorkflow_Success(t *testing.T) {
    workflowID := submitTestWorkflow(t)
    waitForCompletion(t, workflowID)
    
    resp := POST(fmt.Sprintf("/v1/workflows/%s/rerun", workflowID), nil)
    
    assert.Equal(t, 201, resp.StatusCode)
    assert.NotEqual(t, workflowID, resp.Body.ID)
    assert.Equal(t, workflowID, resp.Body.OriginalID)
    assert.Equal(t, "running", resp.Body.Status)
}

func TestRerunWorkflow_WithSkipSteps(t *testing.T) {
    workflowID := submitTestWorkflow(t)
    waitForCompletion(t, workflowID)
    
    req := RerunRequest{
        SkipSteps: []string{"build", "test"},
        FromJob:   "deploy",
    }
    
    resp := POST(fmt.Sprintf("/v1/workflows/%s/rerun", workflowID), req)
    
    assert.Equal(t, 201, resp.StatusCode)
    assert.Contains(t, resp.Body.SkippedSteps, "build")
    assert.Contains(t, resp.Body.SkippedSteps, "test")
}
```

---

### 4.3 技术实现建议

#### 直接调用 Temporal API 的功能

| API 功能 | Temporal API | 实现方式 | 性能 |
|---------|--------------|----------|------|
| 提交工作流 | `client.ExecuteWorkflow()` | 直接调用 | < 100ms |
| 查询基本状态 | `client.DescribeWorkflowExecution()` | 直接调用 | < 50ms |
| 取消工作流 | `client.CancelWorkflow()` | 直接调用 | < 50ms |
| 终止工作流 | `client.TerminateWorkflow()` | 直接调用 | < 50ms |
| 列表查询 | `client.ListWorkflow()` | 构建可见性查询后调用 | < 200ms |
| 发送信号 | `client.SignalWorkflow()` | 直接调用 | < 50ms |
| 查询状态 | `client.QueryWorkflow()` | 直接调用 | < 50ms |

**实现示例:**

```go
// 取消工作流 - 直接调用
func (h *WorkflowHandler) CancelWorkflow(w http.ResponseWriter, r *http.Request) {
    workflowID := mux.Vars(r)["id"]
    
    err := h.temporalClient.GetClient().CancelWorkflow(r.Context(), workflowID, "")
    if err != nil {
        h.writeError(w, r, http.StatusInternalServerError, "cancel_failed", err.Error())
        return
    }
    
    w.WriteHeader(http.StatusNoContent)
}
```

#### 需要 Waterflow 层封装的功能

| API 功能 | 封装原因 | 实现复杂度 |
|---------|----------|------------|
| 获取 Job/Step 状态 | 需解析 Event History | 🟡 中等 |
| 获取日志 | 需从 Event History 重建 | 🟡 中等 |
| 重新运行(跳过 Step) | 需查询历史 + 构建新输入 | 🟡 中等 |
| 列表查询(按名称) | 需设置搜索属性 | 🟢 简单 |
| 暂停/恢复 | 需 Workflow 代码支持 | 🔴 困难 |

**实现示例 - 获取 Job/Step 状态:**

```go
func (h *WorkflowHandler) GetWorkflowStatus(w http.ResponseWriter, r *http.Request) {
    workflowID := mux.Vars(r)["id"]
    
    // 1. 获取基本信息(快速)
    desc, err := h.temporalClient.GetClient().DescribeWorkflowExecution(r.Context(), workflowID, "")
    if err != nil {
        h.writeError(w, r, http.StatusNotFound, "not_found", "Workflow not found")
        return
    }
    
    // 2. 如果需要详细信息,解析 Event History
    includeJobs := r.URL.Query().Get("include") == "jobs" || r.URL.Query().Get("include") == "jobs,steps"
    
    var jobs []JobStatus
    if includeJobs {
        historyIter := h.temporalClient.GetClient().GetWorkflowHistory(
            r.Context(), workflowID, desc.WorkflowExecutionInfo.Execution.RunId, false, 0)
        
        var events []*history.HistoryEvent
        for historyIter.HasNext() {
            event, _ := historyIter.Next()
            events = append(events, event)
        }
        
        jobs = h.historyParser.ParseJobsFromHistory(events)
    }
    
    // 3. 构建响应
    response := WorkflowStatusResponse{
        ID:     workflowID,
        Status: mapTemporalStatus(desc.WorkflowExecutionInfo.Status),
        Jobs:   jobs,
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(response)
}
```

#### 需要额外状态管理的功能

| 功能 | 为什么需要额外状态 | 解决方案 |
|------|-------------------|----------|
| 归档工作流 | Temporal 无"删除" | 使用搜索属性标记 |
| 用户自定义标签 | Temporal 不支持运行时修改 | 外部 KV 存储(可选) |
| 触发关系追踪 | Temporal 无原生支持 | 通过 Memo/SearchAttributes |

**实现示例 - 归档工作流:**

```go
// 归档工作流(标记为已归档)
func (h *WorkflowHandler) ArchiveWorkflow(w http.ResponseWriter, r *http.Request) {
    workflowID := mux.Vars(r)["id"]
    
    // 通过 Signal 更新搜索属性
    err := h.temporalClient.GetClient().SignalWorkflow(
        r.Context(),
        workflowID,
        "",
        "archive",
        true,
    )
    
    if err != nil {
        h.writeError(w, r, http.StatusInternalServerError, "archive_failed", err.Error())
        return
    }
    
    w.WriteHeader(http.StatusNoContent)
}

// Workflow 代码中处理归档信号
func DeployWorkflow(ctx workflow.Context, input DeployInput) error {
    var archived bool
    
    archiveChannel := workflow.GetSignalChannel(ctx, "archive")
    workflow.Go(ctx, func(ctx workflow.Context) {
        archiveChannel.Receive(ctx, &archived)
        
        // 更新搜索属性
        workflow.UpsertSearchAttributes(ctx, map[string]interface{}{
            "Archived": archived,
        })
    })
    
    // ... 工作流逻辑
}

// 列表查询时过滤归档的
func (h *WorkflowHandler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
    showArchived := r.URL.Query().Get("show_archived") == "true"
    
    query := "WorkflowType = 'waterflow-executor'"
    if !showArchived {
        query += " AND Archived != true"
    }
    
    // ... 调用 ListWorkflow
}
```

---

### 4.4 不推荐实现的功能

#### 1. 删除工作流记录

**功能描述:** `DELETE /v1/workflows/{id}` - 永久删除工作流记录

**不推荐原因:**
- ❌ 违背 Event Sourcing 原则
- ❌ Temporal Event History 不可删除(架构限制)
- ❌ 丢失审计追踪
- ❌ 可能导致数据不一致

**替代方案:** 归档功能(标记为不可见)

**ROI:** 极低(破坏架构,用户价值有限)

---

#### 2. 从任意 Activity 中间重试

**功能描述:** "暂停失败的 Activity,修复后从失败点继续"

**不推荐原因:**
- ❌ Temporal 不支持 Activity 级别的断点续传
- ❌ 需要复杂的状态持久化
- ❌ 破坏确定性重放
- ❌ 实现成本极高(数周开发)

**替代方案:** 重新运行工作流 + 跳过成功的 Step

**ROI:** 低(技术复杂度高,用户价值有限)

---

#### 3. 修改运行中工作流的 YAML

**功能描述:** "动态修改工作流定义,添加/删除 Step"

**不推荐原因:**
- ❌ Temporal 工作流代码是不可变的
- ❌ 破坏确定性重放
- ❌ 可能导致状态不一致
- ❌ 实现成本极高

**替代方案:** 
- 取消当前工作流,提交新的修改后的工作流
- 使用 Signal 动态控制工作流行为(如跳过某些 Step)

**ROI:** 极低(架构不兼容)

---

#### 4. 跨工作流的事务支持

**功能描述:** "原子性执行多个工作流,全部成功或全部回滚"

**不推荐原因:**
- ❌ 分布式事务极其复杂
- ❌ 需要 Saga 模式或 2PC
- ❌ Temporal 无原生支持
- ❌ 实现成本极高(数月开发)

**替代方案:**
- 使用父工作流编排多个子工作流
- 手动实现补偿逻辑

**ROI:** 低(复杂度高,大多数场景不需要)

---

#### 5. 实时修改 Activity 超时时间

**功能描述:** "运行中的 Activity,动态延长超时时间"

**不推荐原因:**
- ❌ Temporal Activity Options 在调度时确定,不可修改
- ❌ 需要取消并重新调度 Activity
- ❌ 破坏确定性

**替代方案:**
- 使用 Heartbeat 机制检测 Activity 进度
- 设置合理的初始超时时间

**ROI:** 低(技术限制,用户价值有限)

---

## 5. Story 1.9 更新建议

### 5.1 验收标准更新

**原有 AC 保持不变,增加以下细化标准:**

#### AC1 补充: 工作流提交 API

**新增子项:**

**AC1.1: 变量优先级**
- YAML 中的 `vars` 作为默认值
- API 请求的 `vars` 覆盖 YAML 中的值
- 合并规则:浅合并(仅顶层 key)

**AC1.2: 触发元数据**
- 记录触发类型(manual)
- 记录提交用户(从认证信息提取)
- 记录提交时间
- 设置 Temporal SearchAttributes

**AC1.3: 性能要求**
- P50 延迟 < 200ms
- P95 延迟 < 500ms
- P99 延迟 < 1s

---

#### AC2 补充: 工作流查询 API

**新增子项:**

**AC2.1: 分层查询**
- 默认:仅返回基本状态(status, times)
- `?include=jobs`:包含 Job 级别状态
- `?include=jobs,steps`:包含 Step 级别状态

**AC2.2: 性能要求**
- 基本状态: P95 < 100ms
- 包含 Jobs: P95 < 300ms
- 包含 Steps: P95 < 500ms

**AC2.3: 错误处理**
- Temporal 不可用 → 503 Service Unavailable
- 工作流不存在 → 404 Not Found
- 内部错误 → 500 Internal Server Error

---

#### AC3 补充: 工作流列表查询 API

**新增子项:**

**AC3.1: 搜索属性配置**
- 注册自定义搜索属性: WorkflowName, Environment, Archived
- Workflow 启动时设置搜索属性
- 支持按 WorkflowName 精确搜索

**AC3.2: 分页限制**
- page >= 1
- 1 <= limit <= 100
- 默认 limit = 20

**AC3.3: 排序**
- 默认按 StartTime DESC
- 未来支持自定义排序(Post-MVP)

---

#### AC4 补充: 工作流日志查询 API

**新增子项:**

**AC4.1: 日志重建规则**
- WorkflowExecutionStarted → "Workflow started"
- ActivityTaskStarted → "Step started"
- ActivityTaskCompleted → "Step completed"
- ActivityTaskFailed → "Step failed" + error message
- ActivityTaskTimedOut → "Step timeout"

**AC4.2: 实时日志流(可选)**
- 支持 Server-Sent Events
- `?stream=true` 启用
- 长轮询 Temporal Event History

**AC4.3: 性能要求**
- 小型工作流(< 100 events): < 100ms
- 中型工作流(100-1000 events): 100-500ms
- 大型工作流(1000+ events): 500ms-2s

---

#### AC5 补充: 取消工作流 API

**新增子项:**

**AC5.1: 取消传播**
- 工作流接收取消信号
- 自动取消所有运行中的 Activity
- 子工作流接收取消信号

**AC5.2: 审计日志**
- 记录取消操作到审计日志
- 包含操作用户、时间、原因

---

#### AC6 补充: 重新运行工作流 API

**新增子项:**

**AC6.1: 跳过逻辑**
- 支持 `skip_steps`: 跳过指定 Step
- 支持 `from_job`: 从指定 Job 开始
- 提取原工作流的成功 Step 列表

**AC6.2: 变量覆盖**
- 支持 `vars`: 覆盖原工作流变量
- 保留原工作流其他配置

**AC6.3: 关联追踪**
- 新工作流包含 `original_id` 字段
- 通过 Memo 记录重新运行关系

---

### 5.2 新增验收标准

#### AC7: 强制终止工作流 API (Post-MVP)

**Given** 工作流正在运行或卡住  
**When** POST `/v1/workflows/{id}/terminate` 请求终止  
**Then** 返回 204 No Content  
**And** 工作流立即终止,不执行清理逻辑  
**And** 工作流状态变为 terminated  
**And** 需要管理员权限  
**And** 记录终止原因到 Event History  

---

#### AC8: 归档工作流 API (Post-MVP)

**Given** 工作流已完成  
**When** POST `/v1/workflows/{id}/archive` 请求归档  
**Then** 返回 204 No Content  
**And** 工作流标记为已归档(SearchAttribute Archived=true)  
**And** 列表查询默认不显示归档的工作流  
**And** `?show_archived=true` 可显示归档的工作流  
**And** 支持取消归档: POST `/v1/workflows/{id}/unarchive`  

---

#### AC9: 获取原始 Event History API (Post-MVP)

**Given** 工作流存在  
**When** GET `/v1/workflows/{id}/events` 查询事件历史  
**Then** 返回 200 和 Temporal Event History  
**And** 支持分页(`page_size`, `next_page_token`)  
**And** 支持过滤(`filter=activity`)  
**And** 用于高级调试  

---

### 5.3 技术约束补充

#### 约束 1: Event Sourcing 架构

- **不可删除工作流记录** - Temporal Event History 不可变
- **状态从事件重建** - 每次查询都解析 Event History
- **完整审计追踪** - 所有操作记录到 Event History

#### 约束 2: 单节点执行模式

- **Activity 是最小单元** - 无法从 Activity 中间重试
- **重试需重新运行** - 创建新工作流执行
- **跳过逻辑需 Workflow 支持** - 在 Workflow 代码中实现

#### 约束 3: Temporal 性能限制

- **Event History 大小限制** - 默认 50MB,最大 500MB
- **ListWorkflow PageSize 限制** - 最大 1000
- **SearchAttributes 类型限制** - Keyword, Int, Double, Bool, Datetime

---

### 5.4 文档更新建议

#### 新增文档

**1. API 完整参考文档**
- 文件: `docs/api-reference.md`
- 内容:每个端点的详细说明、参数、响应、错误码

**2. Event Sourcing 执行模型**
- 文件: `docs/concepts/event-sourcing-execution-model.md`
- 内容:解释为什么不能删除工作流,如何查询历史状态

**3. 重试策略最佳实践**
- 文件: `docs/guides/retry-strategies.md`
- 内容:如何设计幂等的 Step,如何使用重新运行功能

#### 更新文档

**1. 快速开始指南**
- 文件: `docs/quick-start.md`
- 新增:重新运行工作流示例

**2. API 使用指南**
- 文件: `docs/api-guide.md`
- 新增:所有新端点的使用示例

**3. 故障排查指南**
- 文件: `docs/troubleshooting.md`
- 新增:如何查看日志调试失败的工作流

---

## 总结

本调研报告为 Waterflow Workflow Management API 提供了全面的技术分析和设计建议:

### 关键发现

1. **Temporal SDK 能力**
   - ✅ 提供完整的工作流管理 API (Describe, List, Cancel, Terminate)
   - ⚠️ 日志和详细状态需要从 Event History 重建
   - ⚠️ 部分功能需要 Workflow 代码配合(Signal, Query, 暂停/恢复)

2. **行业最佳实践**
   - ✅ GitHub Actions: 简单易用,但功能有限
   - ✅ GitLab CI: 粒度控制好,支持手动审批
   - ✅ Argo Workflows: 功能最完整,支持暂停/恢复

3. **Waterflow 架构适配**
   - ✅ Event Sourcing 提供完整审计,但不能删除记录
   - ✅ 单节点执行模式简化架构,但重试需重新运行
   - ✅ YAML DSL 支持 DAG 和 Matrix,重试逻辑需精心设计

### 推荐实施

**MVP 阶段(P0):**
- ✅ 已实现的 6 个核心 API 保持不变
- ✅ 优化日志解析性能
- ✅ 完善错误处理和验证

**Post-MVP 扩展(P1-P2):**
- 🟢 强制终止 API (简单,1 天)
- 🟡 归档工作流 API (中等,2-3 天)
- 🟡 原始 Event History API (简单,1 天)
- 🔴 暂停/恢复 API (困难,需 Workflow 重构)

**不推荐:**
- ❌ 删除工作流记录(违背架构)
- ❌ Activity 中间重试(技术限制)
- ❌ 修改运行中工作流(架构不兼容)

---

**报告完成日期:** 2026-01-27  
**下一步行动:** 根据本报告更新 Story 1.9 验收标准,准备开始 Post-MVP 功能开发
