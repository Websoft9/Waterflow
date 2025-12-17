# Story 1.8: 工作流日志输出

Status: ready-for-dev

## Story

As a **工作流用户**,  
I want **获取工作流执行日志**,  
So that **调试失败原因和验证执行过程**。

## Acceptance Criteria

**Given** 工作流正在执行或已完成  
**When** GET `/v1/workflows/{id}/logs` 请求日志  
**Then** 返回结构化日志 (JSON 格式)  
**And** 日志包含时间戳、级别、Job/Step 信息、消息  
**And** 支持日志级别过滤 (info, warn, error)  
**And** 支持实时日志流 (SSE 或 WebSocket)  
**And** 历史日志从 Temporal Event History 获取

## Technical Context

### Architecture Constraints

根据 [docs/architecture.md](docs/architecture.md) §3.1.1 REST API Handler设计:

1. **核心职责**
   - 处理 `GET /v1/workflows/{id}/logs` 请求
   - 从 Temporal Event History 提取日志事件
   - 返回结构化的日志数据 (JSON)
   - MVP 阶段支持实时查询,后续可支持流式输出

2. **集成接口** (参考 architecture.md 集成接口)
   - **FR17 LogHandler 接口**: 支持集成外部日志系统 (ELK/Loki/CloudWatch)
   - MVP 阶段日志存储在 Temporal Event History
   - 后续 Story 实现 LogHandler 接口,支持实时日志流

3. **功能需求映射**
   - **FR7 实时状态跟踪和日志**: 本 Story 实现日志查询 API
   - **FR3 工作流管理 API**: GET /v1/workflows/{id}/logs 端点

2. **日志来源**

**Temporal Event History日志:**
```
Event Type                     → Log Entry
─────────────────────────────────────────────────────
WorkflowExecutionStarted      → [info] Workflow started
ActivityTaskScheduled         → [info] Step 'Build' started
ActivityTaskCompleted         → [info] Step 'Build' completed (duration: 2.3s)
ActivityTaskFailed            → [error] Step 'Build' failed: exit code 1
WorkflowExecutionCompleted    → [info] Workflow completed successfully
WorkflowExecutionFailed       → [error] Workflow failed: <reason>
```

**Activity日志 (未来扩展):**
```json
{
  "timestamp": "2025-12-16T10:30:15.123Z",
  "level": "info",
  "job": "build",
  "step": "Build",
  "message": "Running command: go build -o bin/server",
  "source": "activity"
}
```

3. **响应格式**

**JSON日志数组 (默认):**
```json
{
  "workflow_id": "wf-550e8400-e29b-41d4-a716-446655440000",
  "logs": [
    {
      "timestamp": "2025-12-16T10:30:00.000Z",
      "level": "info",
      "type": "workflow.started",
      "message": "Workflow 'Deploy Application' started"
    },
    {
      "timestamp": "2025-12-16T10:30:02.123Z",
      "level": "info",
      "type": "step.started",
      "job": "build",
      "step": "Build",
      "message": "Step 'Build' started"
    },
    {
      "timestamp": "2025-12-16T10:30:05.456Z",
      "level": "info",
      "type": "step.completed",
      "job": "build",
      "step": "Build",
      "duration": 3333,
      "message": "Step 'Build' completed in 3.3s"
    },
    {
      "timestamp": "2025-12-16T10:30:10.789Z",
      "level": "error",
      "type": "step.failed",
      "job": "build",
      "step": "Test",
      "error": "exit code 1",
      "message": "Step 'Test' failed: exit code 1"
    }
  ],
  "total": 4,
  "filtered": true
}
```

**SSE流 (实时日志):**
```
GET /v1/workflows/{id}/logs?stream=true

Response:
Content-Type: text/event-stream

event: log
data: {"timestamp":"2025-12-16T10:30:00Z","level":"info","message":"Workflow started"}

event: log
data: {"timestamp":"2025-12-16T10:30:02Z","level":"info","message":"Step 'Build' started"}

event: log
data: {"timestamp":"2025-12-16T10:30:05Z","level":"info","message":"Step 'Build' completed"}

event: close
data: {"reason":"workflow_completed"}
```

4. **过滤参数**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `level` | string | all | 过滤级别: all, info, warn, error |
| `stream` | boolean | false | 是否启用实时流 |
| `limit` | int | 1000 | 最大日志条数 (非流模式) |
| `since` | timestamp | - | 仅返回指定时间后的日志 |

**示例请求:**
```bash
# 获取所有日志
GET /v1/workflows/wf-123/logs

# 仅获取错误日志
GET /v1/workflows/wf-123/logs?level=error

# 实时日志流
GET /v1/workflows/wf-123/logs?stream=true

# 最近100条info日志
GET /v1/workflows/wf-123/logs?level=info&limit=100
```

### Dependencies

**前置Story:**
- ✅ Story 1.4: Temporal SDK集成
  - 使用: `GetWorkflowHistory` API
- ✅ Story 1.6: 基础工作流执行引擎
  - 使用: Workflow产生的Event History
- ✅ Story 1.7: 工作流状态查询API
  - 复用: Event History遍历逻辑

**后续Story依赖本Story:**
- Story 2.x: Agent执行日志 - 需将Agent输出注入到日志流

### Technology Stack

**Temporal Event History API:**

```go
import (
    "go.temporal.io/sdk/client"
    "go.temporal.io/api/enums/v1"
    "go.temporal.io/api/history/v1"
)

// 获取完整Event History
iter := temporalClient.GetWorkflowHistory(
    ctx,
    workflowID,
    runID,
    false, // waitNewEvent = false (历史日志)
    enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
)

// 实时日志流 (Long Polling)
iter := temporalClient.GetWorkflowHistory(
    ctx,
    workflowID,
    runID,
    true, // waitNewEvent = true (实时流)
    enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
)

for iter.HasNext() {
    event, err := iter.Next()
    if err != nil {
        break
    }
    
    logEntry := convertEventToLog(event)
    // 发送到SSE流或添加到日志数组
}
```

**日志转换逻辑:**

```go
package service

import (
    "time"
    "go.temporal.io/api/enums/v1"
    "go.temporal.io/api/history/v1"
)

type LogEntry struct {
    Timestamp time.Time              `json:"timestamp"`
    Level     string                 `json:"level"` // info, warn, error
    Type      string                 `json:"type"`  // workflow.started, step.started, etc.
    Job       string                 `json:"job,omitempty"`
    Step      string                 `json:"step,omitempty"`
    Duration  int64                  `json:"duration,omitempty"` // 毫秒
    Error     string                 `json:"error,omitempty"`
    Message   string                 `json:"message"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func convertEventToLog(event *historypb.HistoryEvent) *LogEntry {
    entry := &LogEntry{
        Timestamp: event.EventTime.AsTime(),
        Metadata:  make(map[string]interface{}),
    }
    
    switch event.EventType {
    case enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED:
        entry.Level = "info"
        entry.Type = "workflow.started"
        entry.Message = "Workflow started"
        
    case enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED:
        attrs := event.GetActivityTaskScheduledEventAttributes()
        entry.Level = "info"
        entry.Type = "step.started"
        entry.Step = attrs.ActivityType.Name
        entry.Message = fmt.Sprintf("Step '%s' started", attrs.ActivityType.Name)
        
    case enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
        attrs := event.GetActivityTaskCompletedEventAttributes()
        entry.Level = "info"
        entry.Type = "step.completed"
        
        // 查找对应的Scheduled事件获取Step名称
        scheduledEvent := findScheduledEvent(event.EventId, allEvents)
        if scheduledEvent != nil {
            scheduledAttrs := scheduledEvent.GetActivityTaskScheduledEventAttributes()
            entry.Step = scheduledAttrs.ActivityType.Name
            
            // 计算持续时间
            startTime := scheduledEvent.EventTime.AsTime()
            endTime := event.EventTime.AsTime()
            entry.Duration = endTime.Sub(startTime).Milliseconds()
            
            entry.Message = fmt.Sprintf("Step '%s' completed in %.1fs",
                entry.Step,
                float64(entry.Duration)/1000.0,
            )
        }
        
    case enums.EVENT_TYPE_ACTIVITY_TASK_FAILED:
        attrs := event.GetActivityTaskFailedEventAttributes()
        entry.Level = "error"
        entry.Type = "step.failed"
        
        scheduledEvent := findScheduledEvent(attrs.ScheduledEventId, allEvents)
        if scheduledEvent != nil {
            scheduledAttrs := scheduledEvent.GetActivityTaskScheduledEventAttributes()
            entry.Step = scheduledAttrs.ActivityType.Name
        }
        
        entry.Error = attrs.Failure.GetMessage()
        entry.Message = fmt.Sprintf("Step '%s' failed: %s", entry.Step, entry.Error)
        
    case enums.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED:
        entry.Level = "info"
        entry.Type = "workflow.completed"
        entry.Message = "Workflow completed successfully"
        
    case enums.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED:
        attrs := event.GetWorkflowExecutionFailedEventAttributes()
        entry.Level = "error"
        entry.Type = "workflow.failed"
        entry.Error = attrs.Failure.GetMessage()
        entry.Message = fmt.Sprintf("Workflow failed: %s", entry.Error)
        
    default:
        // 不记录其他类型事件
        return nil
    }
    
    return entry
}
```

**SSE (Server-Sent Events) 实现:**

```go
// Gin框架SSE支持
func (h *WorkflowHandler) StreamLogs(c *gin.Context) {
    workflowID := c.Param("id")
    
    // 设置SSE Headers
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Header("X-Accel-Buffering", "no") // Nginx: 禁用缓冲
    
    ctx := c.Request.Context()
    
    // Long Polling获取实时Event
    iter := h.workflowLogService.GetRealtimeLogs(ctx, workflowID)
    
    for {
        select {
        case <-ctx.Done():
            // 客户端断开连接
            return
        default:
            if !iter.HasNext() {
                // 工作流已完成,发送close事件
                c.SSEvent("close", map[string]string{
                    "reason": "workflow_completed",
                })
                c.Writer.Flush()
                return
            }
            
            logEntry, err := iter.Next()
            if err != nil {
                c.SSEvent("error", map[string]string{
                    "message": err.Error(),
                })
                c.Writer.Flush()
                return
            }
            
            // 发送日志事件
            c.SSEvent("log", logEntry)
            c.Writer.Flush()
        }
    }
}
```

### Project Structure Updates

基于Story 1.1-1.7的结构,本Story新增:

```
internal/
├── service/
│   ├── workflow_log_service.go        # 日志服务层 (新建)
│   └── workflow_log_service_test.go   (新建)
├── server/handlers/
│   └── workflow.go                    # 修改 - 添加GetLogs/StreamLogs方法
├── models/
│   └── workflow_log.go                # 日志模型 (新建)

api/
└── openapi.yaml                       # 更新 - 添加GET /v1/workflows/{id}/logs
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
  # test/verify-dependencies-story-1-8.sh
  #!/bin/bash
  
  set -e
  
  echo "=== Story 1.8 依赖验证 ==="
  
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
  
  # Story 1.4: Temporal SDK Integration
  check_file "internal/temporal/client.go" "Story 1.4"
  
  # Story 1.5: Workflow Submission API
  check_file "internal/service/workflow_service.go" "Story 1.5"
  check_file "internal/server/handlers/workflow.go" "Story 1.5"
  
  # Story 1.6: Workflow Execution Engine
  check_file "internal/workflow/waterflow_workflow.go" "Story 1.6"
  check_file "internal/workflow/activities.go" "Story 1.6"
  check_file "internal/workflow/worker.go" "Story 1.6"
  
  # Story 1.7: Workflow Status Query API
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
  echo "✅ Story 1.8 所有依赖验证通过"
  ```

- [ ] 0.4 运行验证脚本
  ```bash
  chmod +x test/verify-dependencies-story-1-8.sh
  ./test/verify-dependencies-story-1-8.sh
  ```

### Task 1: 定义日志模型 (AC: 结构化日志)

- [ ] 1.1 创建`internal/models/workflow_log.go`
  ```go
  package models
  
  import "time"
  
  // WorkflowLogsResponse 日志响应
  type WorkflowLogsResponse struct {
      WorkflowID string      `json:"workflow_id"`
      Logs       []*LogEntry `json:"logs"`
      Total      int         `json:"total"`
      Filtered   bool        `json:"filtered"` // 是否应用了过滤
  }
  
  // LogEntry 单条日志
  type LogEntry struct {
      Timestamp time.Time              `json:"timestamp"`
      Level     string                 `json:"level"` // info, warn, error
      Type      string                 `json:"type"`  // workflow.started, step.started, etc.
      Job       string                 `json:"job,omitempty"`
      Step      string                 `json:"step,omitempty"`
      Duration  int64                  `json:"duration,omitempty"` // 毫秒
      Error     string                 `json:"error,omitempty"`
      Message   string                 `json:"message"`
      Metadata  map[string]interface{} `json:"metadata,omitempty"`
  }
  
  // LogFilter 日志过滤参数
  type LogFilter struct {
      Level  string     // all, info, warn, error
      Limit  int        // 最大条数
      Since  *time.Time // 时间过滤
  }
  ```

### Task 2: 实现Event转Log逻辑 (AC: 从Event History获取)

- [ ] 2.1 创建`internal/service/workflow_log_service.go`
  ```go
  package service
  
  import (
      "context"
      "fmt"
      "sync"
      "time"
      
      "go.temporal.io/api/enums/v1"
      "go.temporal.io/api/history/v1"
      "go.uber.org/zap"
      
      "waterflow/internal/models"
      "waterflow/internal/temporal"
  )
  
  // LogCache 为已完成工作流提供日志缓存
  type LogCache struct {
      mu      sync.RWMutex
      entries map[string]*CachedLogs
      ttl     time.Duration
  }
  
  type CachedLogs struct {
      Logs      *models.WorkflowLogsResponse
      CachedAt  time.Time
      ExpiresAt time.Time
  }
  
  func NewLogCache(ttl time.Duration) *LogCache {
      return &LogCache{
          entries: make(map[string]*CachedLogs),
          ttl:     ttl,
      }
  }
  
  func (lc *LogCache) Get(workflowID string) *models.WorkflowLogsResponse {
      lc.mu.RLock()
      defer lc.mu.RUnlock()
      
      cached, exists := lc.entries[workflowID]
      if !exists || time.Now().After(cached.ExpiresAt) {
          return nil
      }
      return cached.Logs
  }
  
  func (lc *LogCache) Set(workflowID string, logs *models.WorkflowLogsResponse) {
      lc.mu.Lock()
      defer lc.mu.Unlock()
      
      lc.entries[workflowID] = &CachedLogs{
          Logs:      logs,
          CachedAt:  time.Now(),
          ExpiresAt: time.Now().Add(lc.ttl),
      }
  }
  
  type WorkflowLogService struct {
      temporalClient *temporal.Client
      logger         *zap.Logger
      cache          *LogCache
  }
  
  func NewWorkflowLogService(tc *temporal.Client, logger *zap.Logger) *WorkflowLogService {
      return &WorkflowLogService{
          temporalClient: tc,
          logger:         logger,
          cache:          NewLogCache(1 * time.Hour), // 缓存1小时
      }
  }
  
  // GetLogs 获取历史日志
  func (wls *WorkflowLogService) GetLogs(ctx context.Context, workflowID string, filter *models.LogFilter) (*models.WorkflowLogsResponse, error) {
      // 1. 检查缓存 (仅适用于已完成工作流)
      if cached := wls.cache.Get(workflowID); cached != nil {
          wls.logger.Debug("Cache hit for completed workflow",
              zap.String("workflow_id", workflowID),
          )
          // 应用过滤器到缓存结果
          return wls.applyFilterToResponse(cached, filter), nil
      }
      
      // 2. 获取Event History
      iter := wls.temporalClient.GetClient().GetWorkflowHistory(
          ctx,
          workflowID,
          "", // runID为空则查询最新run
          false, // waitNewEvent = false (历史日志)
          enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
      )
      
      // 3. 收集所有事件 (需要完整列表用于关联)
      var allEvents []*historypb.HistoryEvent
      for iter.HasNext() {
          event, err := iter.Next()
          if err != nil {
              return nil, fmt.Errorf("failed to iterate history: %w", err)
          }
          allEvents = append(allEvents, event)
      }
      
      // 4. 转换为日志 (不应用过滤,缓存原始数据)
      var logs []*models.LogEntry
      var workflowStatus string
      
      for _, event := range allEvents {
          logEntry := wls.convertEventToLog(event, allEvents)
          if logEntry == nil {
              continue // 跳过不关心的事件类型
          }
          logs = append(logs, logEntry)
          
          // 检测工作流状态
          if event.EventType == enums.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED {
              workflowStatus = "completed"
          } else if event.EventType == enums.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED {
              workflowStatus = "failed"
          } else if event.EventType == enums.EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED {
              workflowStatus = "canceled"
          }
      }
      
      // 5. 创建响应对象
      response := &models.WorkflowLogsResponse{
          WorkflowID: workflowID,
          Logs:       logs,
          Total:      len(logs),
          Filtered:   false,
      }
      
      // 6. 缓存已完成的工作流日志
      if workflowStatus == "completed" || workflowStatus == "failed" || workflowStatus == "canceled" {
          wls.cache.Set(workflowID, response)
          wls.logger.Debug("Cached completed workflow logs",
              zap.String("workflow_id", workflowID),
              zap.String("status", workflowStatus),
              zap.Int("log_count", len(logs)),
          )
      }
      
      // 7. 应用过滤器
      return wls.applyFilterToResponse(response, filter), nil
  }
  
  // applyFilterToResponse 对响应应用过滤条件
  func (wls *WorkflowLogService) applyFilterToResponse(response *models.WorkflowLogsResponse, filter *models.LogFilter) *models.WorkflowLogsResponse {
      filteredLogs := []*models.LogEntry{}
      
      for _, log := range response.Logs {
          if !wls.matchesFilter(log, filter) {
              continue
          }
          filteredLogs = append(filteredLogs, log)
      }
      
      // 限制数量
      total := len(filteredLogs)
      filtered := false
      if filter.Limit > 0 && len(filteredLogs) > filter.Limit {
          filteredLogs = filteredLogs[:filter.Limit]
          filtered = true
      }
      
      return &models.WorkflowLogsResponse{
          WorkflowID: response.WorkflowID,
          Logs:       filteredLogs,
          Total:      total,
          Filtered:   filtered,
      }
  }
  
  // convertEventToLog 转换单个事件为日志
  func (wls *WorkflowLogService) convertEventToLog(event *historypb.HistoryEvent, allEvents []*historypb.HistoryEvent) *models.LogEntry {
      entry := &models.LogEntry{
          Timestamp: event.EventTime.AsTime(),
          Metadata:  make(map[string]interface{}),
      }
      
      switch event.EventType {
      case enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED:
          entry.Level = "info"
          entry.Type = "workflow.started"
          entry.Message = "Workflow started"
          
      case enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED:
          attrs := event.GetActivityTaskScheduledEventAttributes()
          entry.Level = "info"
          entry.Type = "step.started"
          entry.Step = attrs.ActivityType.Name
          entry.Job = "build" // MVP: 单Job
          entry.Message = fmt.Sprintf("Step '%s' started", attrs.ActivityType.Name)
          
      case enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
          // 查找对应的Scheduled事件
          scheduledEvent := wls.findEventByID(event.GetActivityTaskCompletedEventAttributes().ScheduledEventId, allEvents)
          if scheduledEvent != nil {
              scheduledAttrs := scheduledEvent.GetActivityTaskScheduledEventAttributes()
              stepName := scheduledAttrs.ActivityType.Name
              
              entry.Level = "info"
              entry.Type = "step.completed"
              entry.Step = stepName
              entry.Job = "build"
              
              // 计算持续时间
              startTime := scheduledEvent.EventTime.AsTime()
              endTime := event.EventTime.AsTime()
              entry.Duration = endTime.Sub(startTime).Milliseconds()
              
              entry.Message = fmt.Sprintf("Step '%s' completed in %.1fs",
                  stepName,
                  float64(entry.Duration)/1000.0,
              )
          } else {
              return nil // 无法关联Step信息
          }
          
      case enums.EVENT_TYPE_ACTIVITY_TASK_FAILED:
          attrs := event.GetActivityTaskFailedEventAttributes()
          scheduledEvent := wls.findEventByID(attrs.ScheduledEventId, allEvents)
          if scheduledEvent != nil {
              scheduledAttrs := scheduledEvent.GetActivityTaskScheduledEventAttributes()
              stepName := scheduledAttrs.ActivityType.Name
              
              entry.Level = "error"
              entry.Type = "step.failed"
              entry.Step = stepName
              entry.Job = "build"
              entry.Error = attrs.Failure.GetMessage()
              entry.Message = fmt.Sprintf("Step '%s' failed: %s", stepName, entry.Error)
          } else {
              return nil
          }
          
      case enums.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED:
          entry.Level = "info"
          entry.Type = "workflow.completed"
          entry.Message = "Workflow completed successfully"
          
      case enums.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED:
          attrs := event.GetWorkflowExecutionFailedEventAttributes()
          entry.Level = "error"
          entry.Type = "workflow.failed"
          entry.Error = attrs.Failure.GetMessage()
          entry.Message = fmt.Sprintf("Workflow failed: %s", entry.Error)
          
      case enums.EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED:
          entry.Level = "warn"
          entry.Type = "workflow.canceled"
          entry.Message = "Workflow was canceled"
          
      default:
          // 不记录其他事件类型
          return nil
      }
      
      return entry
  }
  
  // findEventByID 查找指定ID的事件
  func (wls *WorkflowLogService) findEventByID(eventID int64, allEvents []*historypb.HistoryEvent) *historypb.HistoryEvent {
      for _, event := range allEvents {
          if event.EventId == eventID {
              return event
          }
      }
      return nil
  }
  
  // matchesFilter 检查日志是否匹配过滤条件
  func (wls *WorkflowLogService) matchesFilter(log *models.LogEntry, filter *models.LogFilter) bool {
      // 级别过滤
      if filter.Level != "" && filter.Level != "all" {
          if log.Level != filter.Level {
              return false
          }
      }
      
      // 时间过滤
      if filter.Since != nil {
          if log.Timestamp.Before(*filter.Since) {
              return false
          }
      }
      
      return true
  }
  ```

### Task 3: 实现HTTP Handler (AC: GET /v1/workflows/{id}/logs)

- [ ] 3.1 更新`internal/server/handlers/workflow.go`
  ```go
  // GetLogs - GET /v1/workflows/:id/logs
  func (h *WorkflowHandler) GetLogs(c *gin.Context) {
      workflowID := c.Param("id")
      
      // 验证WorkflowID
      if workflowID == "" || !strings.HasPrefix(workflowID, "wf-") {
          c.JSON(http.StatusBadRequest, models.NewBadRequestError(
              "Invalid workflow ID format",
          ))
          return
      }
      
      // 解析查询参数
      filter := &models.LogFilter{
          Level: c.DefaultQuery("level", "all"),
          Limit: 1000, // 默认限制
      }
      
      if limitStr := c.Query("limit"); limitStr != "" {
          if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
              filter.Limit = limit
          }
      }
      
      if sinceStr := c.Query("since"); sinceStr != "" {
          if sinceTime, err := time.Parse(time.RFC3339, sinceStr); err == nil {
              filter.Since = &sinceTime
          }
      }
      
      // 检查是否请求流模式
      if c.Query("stream") == "true" {
          h.StreamLogs(c)
          return
      }
      
      // 获取历史日志
      ctx := c.Request.Context()
      logs, err := h.workflowLogService.GetLogs(ctx, workflowID, filter)
      
      if err != nil {
          if strings.Contains(err.Error(), "not found") {
              c.JSON(http.StatusNotFound, &models.ErrorResponse{
                  Type:   "https://waterflow.io/errors/not-found",
                  Title:  "Workflow Not Found",
                  Status: 404,
                  Detail: fmt.Sprintf("Workflow with ID '%s' does not exist", workflowID),
              })
              return
          }
          
          h.logger.Error("Failed to get workflow logs",
              zap.String("workflow_id", workflowID),
              zap.Error(err),
          )
          c.JSON(http.StatusInternalServerError, &models.ErrorResponse{
              Type:   "https://waterflow.io/errors/internal-error",
              Title:  "Internal Server Error",
              Status: 500,
              Detail: "Failed to retrieve workflow logs",
          })
          return
      }
      
      c.JSON(http.StatusOK, logs)
  }
  ```

### Task 4: 实现SSE实时日志流 (AC: 支持实时日志流)

- [ ] 4.1 实现流式日志服务
  ```go
  // internal/service/workflow_log_service.go
  
  // LogIterator 日志迭代器
  type LogIterator struct {
      historyIter workflow.HistoryEventIterator
      allEvents   []*historypb.HistoryEvent
      wls         *WorkflowLogService
  }
  
  func (li *LogIterator) HasNext() bool {
      return li.historyIter.HasNext()
  }
  
  func (li *LogIterator) Next() (*models.LogEntry, error) {
      for li.historyIter.HasNext() {
          event, err := li.historyIter.Next()
          if err != nil {
              return nil, err
          }
          
          // 添加到事件列表 (用于关联)
          li.allEvents = append(li.allEvents, event)
          
          // 转换为日志
          logEntry := li.wls.convertEventToLog(event, li.allEvents)
          if logEntry != nil {
              return logEntry, nil
          }
          // 继续循环直到找到有效日志
      }
      return nil, nil
  }
  
  // GetRealtimeLogs 获取实时日志流
  func (wls *WorkflowLogService) GetRealtimeLogs(ctx context.Context, workflowID string) *LogIterator {
      iter := wls.temporalClient.GetClient().GetWorkflowHistory(
          ctx,
          workflowID,
          "",
          true, // waitNewEvent = true (实时流)
          enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
      )
      
      return &LogIterator{
          historyIter: iter,
          allEvents:   []*historypb.HistoryEvent{},
          wls:         wls,
      }
  }
  ```

- [ ] 4.2 实现SSE Handler
  ```go
  // internal/server/handlers/workflow.go
  
  // StreamLogs - GET /v1/workflows/:id/logs?stream=true
  func (h *WorkflowHandler) StreamLogs(c *gin.Context) {
      workflowID := c.Param("id")
      
      // 设置SSE Headers
      c.Header("Content-Type", "text/event-stream")
      c.Header("Cache-Control", "no-cache")
      c.Header("Connection", "keep-alive")
      c.Header("X-Accel-Buffering", "no") // Nginx: 禁用缓冲
      
      ctx := c.Request.Context()
      logIter := h.workflowLogService.GetRealtimeLogs(ctx, workflowID)
      
      // 发送初始化消息
      c.SSEvent("connected", map[string]string{
          "workflow_id": workflowID,
      })
      c.Writer.Flush()
      
      // 流式发送日志
      for {
          select {
          case <-ctx.Done():
              // 客户端断开
              h.logger.Info("Client disconnected from log stream",
                  zap.String("workflow_id", workflowID),
              )
              return
              
          default:
              if !logIter.HasNext() {
                  // 工作流已完成
                  c.SSEvent("close", map[string]string{
                      "reason": "workflow_completed",
                  })
                  c.Writer.Flush()
                  return
              }
              
              logEntry, err := logIter.Next()
              if err != nil {
                  c.SSEvent("error", map[string]string{
                      "message": err.Error(),
                  })
                  c.Writer.Flush()
                  return
              }
              
              if logEntry != nil {
                  c.SSEvent("log", logEntry)
                  c.Writer.Flush()
              }
          }
      }
  }
  ```

### Task 5: 注册路由端点

- [ ] 5.1 更新`internal/server/router.go`
  ```go
  func SetupRouter(logger *zap.Logger, tc *temporal.Client, workflowService *service.WorkflowService) *gin.Engine {
      router := gin.New()
      
      // ... 中间件 ...
      
      v1 := router.Group("/v1")
      {
          // 工作流端点
          workflowQueryService := service.NewWorkflowQueryService(tc, logger)
          workflowLogService := service.NewWorkflowLogService(tc, logger)
          workflowHandler := handlers.NewWorkflowHandler(
              workflowService,
              workflowQueryService,
              workflowLogService, // 新增
              logger,
          )
          
          v1.POST("/workflows", workflowHandler.SubmitWorkflow)
          v1.GET("/workflows/:id", workflowHandler.GetWorkflow)
          v1.GET("/workflows/:id/logs", workflowHandler.GetLogs) // 新增
      }
      
      return router
  }
  ```

- [ ] 5.2 更新Handler构造函数
  ```go
  // internal/server/handlers/workflow.go
  
  type WorkflowHandler struct {
      workflowService      *service.WorkflowService
      workflowQueryService *service.WorkflowQueryService
      workflowLogService   *service.WorkflowLogService // 新增
      logger               *zap.Logger
  }
  
  func NewWorkflowHandler(
      ws *service.WorkflowService,
      wqs *service.WorkflowQueryService,
      wls *service.WorkflowLogService, // 新增
      logger *zap.Logger,
  ) *WorkflowHandler {
      return &WorkflowHandler{
          workflowService:      ws,
          workflowQueryService: wqs,
          workflowLogService:   wls,
          logger:               logger,
      }
  }
  ```

### Task 6: 添加单元测试

- [ ] 6.1 创建`internal/service/workflow_log_service_test.go`
  ```go
  package service
  
  import (
      "context"
      "testing"
      "time"
      
      "github.com/stretchr/testify/assert"
      "go.temporal.io/api/enums/v1"
      "go.uber.org/zap"
  )
  
  func TestConvertEventToLog_WorkflowStarted(t *testing.T) {
      wls := &WorkflowLogService{logger: zap.NewNop()}
      
      event := &historypb.HistoryEvent{
          EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
          EventTime: timestamppb.New(time.Now()),
      }
      
      log := wls.convertEventToLog(event, nil)
      
      assert.NotNil(t, log)
      assert.Equal(t, "info", log.Level)
      assert.Equal(t, "workflow.started", log.Type)
      assert.Contains(t, log.Message, "started")
  }
  
  func TestConvertEventToLog_StepCompleted(t *testing.T) {
      wls := &WorkflowLogService{logger: zap.NewNop()}
      
      startTime := time.Now()
      endTime := startTime.Add(3 * time.Second)
      
      // Scheduled事件
      scheduledEvent := &historypb.HistoryEvent{
          EventId:   100,
          EventType: enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
          EventTime: timestamppb.New(startTime),
          Attributes: &historypb.HistoryEvent_ActivityTaskScheduledEventAttributes{
              ActivityTaskScheduledEventAttributes: &historypb.ActivityTaskScheduledEventAttributes{
                  ActivityType: &commonpb.ActivityType{Name: "Build"},
              },
          },
      }
      
      // Completed事件
      completedEvent := &historypb.HistoryEvent{
          EventType: enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
          EventTime: timestamppb.New(endTime),
          Attributes: &historypb.HistoryEvent_ActivityTaskCompletedEventAttributes{
              ActivityTaskCompletedEventAttributes: &historypb.ActivityTaskCompletedEventAttributes{
                  ScheduledEventId: 100,
              },
          },
      }
      
      allEvents := []*historypb.HistoryEvent{scheduledEvent, completedEvent}
      log := wls.convertEventToLog(completedEvent, allEvents)
      
      assert.NotNil(t, log)
      assert.Equal(t, "info", log.Level)
      assert.Equal(t, "step.completed", log.Type)
      assert.Equal(t, "Build", log.Step)
      assert.Equal(t, int64(3000), log.Duration)
      assert.Contains(t, log.Message, "3.0s")
  }
  
  func TestMatchesFilter_LevelFilter(t *testing.T) {
      wls := &WorkflowLogService{}
      
      errorLog := &models.LogEntry{Level: "error"}
      infoLog := &models.LogEntry{Level: "info"}
      
      filter := &models.LogFilter{Level: "error"}
      
      assert.True(t, wls.matchesFilter(errorLog, filter))
      assert.False(t, wls.matchesFilter(infoLog, filter))
  }
  
  func TestGetLogs_Success(t *testing.T) {
      mockClient := &MockTemporalClient{}
      wls := NewWorkflowLogService(mockClient, zap.NewNop())
      
      // Mock Event History
      mockClient.On("GetWorkflowHistory", mock.Anything, "wf-123", "", false, mock.Anything).Return(
          &MockHistoryIterator{
              events: []*historypb.HistoryEvent{
                  {EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED},
                  {EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED},
              },
          },
      )
      
      filter := &models.LogFilter{Level: "all", Limit: 100}
      logs, err := wls.GetLogs(context.Background(), "wf-123", filter)
      
      assert.NoError(t, err)
      assert.Equal(t, 2, logs.Total)
      assert.Equal(t, "wf-123", logs.WorkflowID)
  }
  ```

- [ ] 6.2 测试Handler
  ```go
  // internal/server/handlers/workflow_test.go
  
  func TestGetLogs_Success(t *testing.T) {
      gin.SetMode(gin.TestMode)
      
      mockLogService := &MockWorkflowLogService{}
      mockLogService.On("GetLogs", mock.Anything, "wf-123", mock.Anything).Return(
          &models.WorkflowLogsResponse{
              WorkflowID: "wf-123",
              Logs: []*models.LogEntry{
                  {Level: "info", Message: "Workflow started"},
              },
              Total: 1,
          }, nil,
      )
      
      handler := NewWorkflowHandler(nil, nil, mockLogService, zap.NewNop())
      
      router := gin.New()
      router.GET("/workflows/:id/logs", handler.GetLogs)
      
      req := httptest.NewRequest("GET", "/workflows/wf-123/logs?level=info", nil)
      w := httptest.NewRecorder()
      router.ServeHTTP(w, req)
      
      assert.Equal(t, http.StatusOK, w.Code)
      
      var resp models.WorkflowLogsResponse
      json.Unmarshal(w.Body.Bytes(), &resp)
      assert.Equal(t, "wf-123", resp.WorkflowID)
      assert.Equal(t, 1, resp.Total)
  }
  
  func TestGetLogs_LevelFilter(t *testing.T) {
      // 测试日志级别过滤
      mockLogService := &MockWorkflowLogService{}
      mockLogService.On("GetLogs", mock.Anything, "wf-123", mock.MatchedBy(func(f *models.LogFilter) bool {
          return f.Level == "error"
      })).Return(&models.WorkflowLogsResponse{
          Logs: []*models.LogEntry{
              {Level: "error", Message: "Step failed"},
          },
      }, nil)
      
      handler := NewWorkflowHandler(nil, nil, mockLogService, zap.NewNop())
      
      router := gin.New()
      router.GET("/workflows/:id/logs", handler.GetLogs)
      
      req := httptest.NewRequest("GET", "/workflows/wf-123/logs?level=error", nil)
      w := httptest.NewRecorder()
      router.ServeHTTP(w, req)
      
      assert.Equal(t, http.StatusOK, w.Code)
  }
  ```

- [ ] 6.3 运行测试
  ```bash
  go test -v ./internal/service -run TestWorkflowLog
  go test -v ./internal/server/handlers -run TestGetLogs
  ```

- [ ] 6.4 性能基准测试
  ```go
  // internal/service/workflow_log_service_benchmark_test.go
  package service
  
  import (
      "context"
      "testing"
      "time"
      
      "github.com/stretchr/testify/mock"
      "go.temporal.io/api/enums/v1"
      "go.temporal.io/api/history/v1"
      "go.uber.org/zap"
      
      "waterflow/internal/models"
  )
  
  // BenchmarkGetLogs_SmallHistory 测试小型工作流(100事件)的查询性能
  func BenchmarkGetLogs_SmallHistory(b *testing.B) {
      wls := setupMockServiceWithEvents(100)
      filter := &models.LogFilter{Level: "all", Limit: 1000}
      ctx := context.Background()
      
      b.ResetTimer()
      for i := 0; i < b.N; i++ {
          _, err := wls.GetLogs(ctx, "wf-test-100", filter)
          if err != nil {
              b.Fatal(err)
          }
      }
  }
  
  // BenchmarkGetLogs_MediumHistory 测试中型工作流(1000事件)的查询性能
  func BenchmarkGetLogs_MediumHistory(b *testing.B) {
      wls := setupMockServiceWithEvents(1000)
      filter := &models.LogFilter{Level: "all", Limit: 1000}
      ctx := context.Background()
      
      b.ResetTimer()
      for i := 0; i < b.N; i++ {
          _, err := wls.GetLogs(ctx, "wf-test-1000", filter)
          if err != nil {
              b.Fatal(err)
          }
      }
  }
  
  // BenchmarkGetLogs_LargeHistory 测试大型工作流(5000事件)的查询性能
  func BenchmarkGetLogs_LargeHistory(b *testing.B) {
      wls := setupMockServiceWithEvents(5000)
      filter := &models.LogFilter{Level: "all", Limit: 1000}
      ctx := context.Background()
      
      b.ResetTimer()
      for i := 0; i < b.N; i++ {
          _, err := wls.GetLogs(ctx, "wf-test-5000", filter)
          if err != nil {
              b.Fatal(err)
          }
      }
  }
  
  // BenchmarkGetLogs_WithCache 测试缓存性能
  func BenchmarkGetLogs_WithCache(b *testing.B) {
      wls := setupMockServiceWithEvents(1000)
      filter := &models.LogFilter{Level: "all", Limit: 1000}
      ctx := context.Background()
      
      // 预热缓存
      wls.GetLogs(ctx, "wf-cached", filter)
      
      b.ResetTimer()
      for i := 0; i < b.N; i++ {
          _, err := wls.GetLogs(ctx, "wf-cached", filter)
          if err != nil {
              b.Fatal(err)
          }
      }
  }
  
  // BenchmarkConvertEventToLog 测试单个事件转换性能
  func BenchmarkConvertEventToLog(b *testing.B) {
      wls := &WorkflowLogService{logger: zap.NewNop()}
      
      event := &historypb.HistoryEvent{
          EventId:   100,
          EventType: enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
          EventTime: timestamppb.New(time.Now()),
          Attributes: &historypb.HistoryEvent_ActivityTaskCompletedEventAttributes{
              ActivityTaskCompletedEventAttributes: &historypb.ActivityTaskCompletedEventAttributes{
                  ScheduledEventId: 99,
              },
          },
      }
      
      scheduledEvent := &historypb.HistoryEvent{
          EventId:   99,
          EventType: enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
          EventTime: timestamppb.New(time.Now().Add(-3 * time.Second)),
          Attributes: &historypb.HistoryEvent_ActivityTaskScheduledEventAttributes{
              ActivityTaskScheduledEventAttributes: &historypb.ActivityTaskScheduledEventAttributes{
                  ActivityType: &commonpb.ActivityType{Name: "Build"},
              },
          },
      }
      
      allEvents := []*historypb.HistoryEvent{scheduledEvent, event}
      
      b.ResetTimer()
      for i := 0; i < b.N; i++ {
          wls.convertEventToLog(event, allEvents)
      }
  }
  
  // BenchmarkMatchesFilter 测试过滤器性能
  func BenchmarkMatchesFilter(b *testing.B) {
      wls := &WorkflowLogService{}
      
      log := &models.LogEntry{
          Level:     "info",
          Timestamp: time.Now(),
      }
      
      filter := &models.LogFilter{
          Level: "info",
          Since: func() *time.Time { t := time.Now().Add(-1 * time.Hour); return &t }(),
      }
      
      b.ResetTimer()
      for i := 0; i < b.N; i++ {
          wls.matchesFilter(log, filter)
      }
  }
  
  // setupMockServiceWithEvents 创建包含指定数量事件的Mock服务
  func setupMockServiceWithEvents(eventCount int) *WorkflowLogService {
      mockClient := &MockTemporalClient{}
      
      events := make([]*historypb.HistoryEvent, eventCount)
      for i := 0; i < eventCount; i++ {
          eventType := enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED
          if i%10 == 0 {
              eventType = enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED
          } else if i%10 == 5 {
              eventType = enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED
          }
          
          events[i] = &historypb.HistoryEvent{
              EventId:   int64(i + 1),
              EventType: eventType,
              EventTime: timestamppb.New(time.Now().Add(time.Duration(i) * time.Second)),
          }
      }
      
      mockClient.On("GetWorkflowHistory", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
          &MockHistoryIterator{events: events},
      )
      
      return NewWorkflowLogService(mockClient, zap.NewNop())
  }
  ```
  
  **性能目标:**
  - 100事件工作流: <50ms p95
  - 1000事件工作流: <200ms p95
  - 5000事件工作流: <500ms p95
  - 缓存命中: <5ms p95
  - convertEventToLog: <1ms per event
  
  **运行基准测试:**
  ```bash
  # 运行所有基准测试
  go test -bench=. -benchmem ./internal/service
  
  # 运行特定基准测试
  go test -bench=BenchmarkGetLogs_SmallHistory -benchtime=10s ./internal/service
  
  # 生成CPU Profile
  go test -bench=BenchmarkGetLogs_LargeHistory -cpuprofile=cpu.prof ./internal/service
  go tool pprof cpu.prof
  ```
  
  **集成到CI:**
  ```yaml
  # .github/workflows/performance.yml
  - name: Run Performance Benchmarks
    run: |
      go test -bench=. -benchmem ./internal/service | tee benchmark.txt
      
      # 验证性能目标
      if grep -q "BenchmarkGetLogs_SmallHistory.*[0-9]\{3,\} ns/op" benchmark.txt; then
        echo "❌ Small history benchmark exceeded 100ms"
        exit 1
      fi
  ```

### Task 7: 集成测试

- [ ] 7.1 创建集成测试脚本
  ```bash
  # test/integration/test_workflow_logs.sh
  #!/bin/bash
  
  set -e
  
  echo "=== Workflow Logs API Integration Test ==="
  
  # 1. 启动环境
  make dev-env
  go run ./cmd/server &
  SERVER_PID=$!
  sleep 3
  
  # 2. 提交工作流
  WORKFLOW_ID=$(curl -s -X POST http://localhost:8080/v1/workflows \
    -H "Content-Type: application/json" \
    -d '{"workflow":"name: Test\non: push\njobs:\n  build:\n    runs-on: linux\n    steps:\n      - name: Build"}' \
    | jq -r '.workflow_id')
  
  echo "Workflow ID: $WORKFLOW_ID"
  
  # 3. 等待执行
  sleep 5
  
  # 4. 获取日志
  echo "Fetching logs..."
  LOGS=$(curl -s http://localhost:8080/v1/workflows/$WORKFLOW_ID/logs)
  echo "Logs: $LOGS"
  
  # 验证日志包含关键字段
  echo $LOGS | jq -e '.workflow_id' > /dev/null
  echo $LOGS | jq -e '.logs' > /dev/null
  echo $LOGS | jq -e '.total' > /dev/null
  
  # 验证至少有2条日志 (started + completed)
  LOG_COUNT=$(echo $LOGS | jq '.total')
  if [ "$LOG_COUNT" -ge 2 ]; then
      echo "✅ Log count validated: $LOG_COUNT"
  else
      echo "❌ Expected >= 2 logs, got $LOG_COUNT"
      exit 1
  fi
  
  # 5. 测试级别过滤
  echo "Testing level filter..."
  ERROR_LOGS=$(curl -s "http://localhost:8080/v1/workflows/$WORKFLOW_ID/logs?level=error")
  echo "Error logs: $ERROR_LOGS"
  
  # 6. 测试SSE流 (简单验证)
  echo "Testing SSE stream..."
  timeout 5s curl -N -s "http://localhost:8080/v1/workflows/$WORKFLOW_ID/logs?stream=true" | head -n 5
  echo "✅ SSE stream working"
  
  # 清理
  kill $SERVER_PID
  make dev-env-stop
  
  echo "✅ Integration test completed"
  ```

- [ ] 7.2 测试SSE客户端
  ```html
  <!-- test/integration/sse_client.html -->
  <!DOCTYPE html>
  <html>
  <head>
      <title>Waterflow SSE Log Stream</title>
  </head>
  <body>
      <h1>Workflow Logs</h1>
      <div id="logs"></div>
      
      <script>
      const workflowId = 'wf-123'; // 替换为实际WorkflowID
      const eventSource = new EventSource(`http://localhost:8080/v1/workflows/${workflowId}/logs?stream=true`);
      
      const logsDiv = document.getElementById('logs');
      
      eventSource.addEventListener('connected', (e) => {
          console.log('Connected:', e.data);
      });
      
      eventSource.addEventListener('log', (e) => {
          const log = JSON.parse(e.data);
          const logEntry = document.createElement('div');
          logEntry.textContent = `[${log.level.toUpperCase()}] ${log.timestamp}: ${log.message}`;
          logsDiv.appendChild(logEntry);
      });
      
      eventSource.addEventListener('close', (e) => {
          const reason = JSON.parse(e.data);
          console.log('Stream closed:', reason.reason);
          eventSource.close();
      });
      
      eventSource.addEventListener('error', (e) => {
          console.error('SSE Error:', e);
          eventSource.close();
      });
      </script>
  </body>
  </html>
  ```

### Task 8: 更新OpenAPI文档

- [ ] 8.1 更新`api/openapi.yaml`
  ```yaml
  paths:
    /v1/workflows/{id}/logs:
      get:
        summary: Get workflow logs
        parameters:
          - name: id
            in: path
            required: true
            schema:
              type: string
            example: wf-550e8400-e29b-41d4-a716-446655440000
          - name: level
            in: query
            schema:
              type: string
              enum: [all, info, warn, error]
              default: all
            description: Filter by log level
          - name: limit
            in: query
            schema:
              type: integer
              default: 1000
            description: Maximum number of logs to return
          - name: stream
            in: query
            schema:
              type: boolean
              default: false
            description: Enable real-time log streaming (SSE)
          - name: since
            in: query
            schema:
              type: string
              format: date-time
            description: Return logs after this timestamp
        responses:
          '200':
            description: Logs retrieved successfully
            content:
              application/json:
                schema:
                  $ref: '#/components/schemas/WorkflowLogsResponse'
              text/event-stream:
                schema:
                  description: Server-Sent Events stream
          '404':
            description: Workflow not found
  
  components:
    schemas:
      WorkflowLogsResponse:
        type: object
        properties:
          workflow_id:
            type: string
          logs:
            type: array
            items:
              $ref: '#/components/schemas/LogEntry'
          total:
            type: integer
          filtered:
            type: boolean
      
      LogEntry:
        type: object
        properties:
          timestamp:
            type: string
            format: date-time
            example: 2025-12-16T10:30:00Z
          level:
            type: string
            enum: [info, warn, error]
            example: info
          type:
            type: string
            example: step.started
          job:
            type: string
            example: build
          step:
            type: string
            example: Build
          duration:
            type: integer
            description: Duration in milliseconds
            example: 3000
          error:
            type: string
            example: exit code 1
          message:
            type: string
            example: "Step 'Build' completed in 3.0s"
          metadata:
            type: object
            additionalProperties: true
  ```

## Dev Notes

### Critical Implementation Guidelines

**1. Event关联 - 避免信息丢失**

```go
// ✅ 正确: 收集所有事件再转换
var allEvents []*historypb.HistoryEvent
for iter.HasNext() {
    allEvents = append(allEvents, iter.Next())
}
for _, event := range allEvents {
    log := convertEventToLog(event, allEvents) // 可关联Scheduled事件
}

// ❌ 错误: 边遍历边转换
for iter.HasNext() {
    event := iter.Next()
    log := convertEventToLog(event, nil) // 无法关联,丢失Step名称
}
```

**2. SSE连接管理 - 优雅关闭**

```go
// ✅ 监听客户端断开
select {
case <-ctx.Done():
    logger.Info("Client disconnected")
    return
default:
    // 继续发送日志
}

// ❌ 不检查断开,持续消耗资源
for logIter.HasNext() {
    c.SSEvent("log", logEntry)
}
```

**3. 内存控制 - 限制日志数量**

```go
// ✅ 默认限制
if filter.Limit > 0 && len(logs) > filter.Limit {
    logs = logs[:filter.Limit]
}

// ❌ 无限制可能OOM
var logs []*LogEntry
for _, event := range allEvents {
    logs = append(logs, convertEventToLog(event))
}
```

**4. SSE Header设置 - 禁用缓冲**

```go
// ✅ 完整SSE Headers
c.Header("Content-Type", "text/event-stream")
c.Header("Cache-Control", "no-cache")
c.Header("Connection", "keep-alive")
c.Header("X-Accel-Buffering", "no") // Nginx代理必需

// ❌ 缺少Headers导致缓冲
c.Header("Content-Type", "text/event-stream")
```

**5. 时间过滤 - 使用索引优化**

```go
// ✅ 提前过滤
for _, event := range allEvents {
    if filter.Since != nil && event.EventTime.AsTime().Before(*filter.Since) {
        continue // 跳过早期事件
    }
    // 转换日志
}

// ❌ 转换后再过滤
for _, event := range allEvents {
    log := convertEventToLog(event)
    if filter.Since != nil && log.Timestamp.Before(*filter.Since) {
        continue // 浪费转换时间
    }
}
```

**6. 长连接超时控制**

```go
// ✅ 设置超时防止僵尸连接
ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Minute)
defer cancel()

logIter := workflowLogService.GetRealtimeLogs(ctx, workflowID)

// ❌ 无超时可能永久占用
logIter := workflowLogService.GetRealtimeLogs(c.Request.Context(), workflowID)
```

### Integration with Previous Stories

**与Story 1.6 Workflow执行集成:**

```go
// Story 1.6产生Event History
workflow.ExecuteActivity(ctx, "ExecuteStepActivity", input)

// Story 1.8读取Event History转换为日志
iter := client.GetWorkflowHistory(ctx, workflowID, runID, ...)
for iter.HasNext() {
    event := iter.Next()
    log := convertEventToLog(event)
}
```

**与Story 1.7 状态查询集成:**

```go
// Story 1.7已实现Event History遍历
// Story 1.8复用同样的API,但提取不同信息

// Story 1.7: 提取进度
case EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
    completedSteps++

// Story 1.8: 提取日志
case EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
    log := &LogEntry{
        Type:    "step.completed",
        Message: fmt.Sprintf("Step '%s' completed", stepName),
    }
```

**为Story 2.x准备 (Agent日志):**

```go
// 未来扩展: Activity可附加自定义日志
type StepExecutionResult struct {
    Success bool     `json:"success"`
    Logs    []string `json:"logs"` // Agent输出的日志
}

// 从Activity Result提取日志
case EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
    result := event.GetActivityTaskCompletedEventAttributes().Result
    var stepResult StepExecutionResult
    json.Unmarshal(result.Payloads[0].Data, &stepResult)
    
    // 注入Agent日志到日志流
    for _, line := range stepResult.Logs {
        logs = append(logs, &LogEntry{
            Level:   "info",
            Source:  "agent",
            Message: line,
        })
    }
```

### Testing Strategy

**单元测试覆盖:**

| 组件 | 测试场景 |
|------|---------|
| convertEventToLog | 所有Event类型映射 |
| findEventByID | 正常查找、未找到 |
| matchesFilter | 级别过滤、时间过滤 |
| GetLogs | 成功、404、过滤 |
| LogIterator | 遍历、提前结束 |

**集成测试:**

```bash
# 1. 提交工作流
WORKFLOW_ID=$(curl -X POST /v1/workflows -d @workflow.json | jq -r '.workflow_id')

# 2. 等待执行
sleep 5

# 3. 获取所有日志
curl /v1/workflows/$WORKFLOW_ID/logs
# 期望: {"total": 4, "logs": [...]}

# 4. 过滤错误日志
curl /v1/workflows/$WORKFLOW_ID/logs?level=error
# 期望: 仅返回error级别

# 5. 测试SSE流
curl -N /v1/workflows/$WORKFLOW_ID/logs?stream=true
# 期望: 持续输出event: log事件
```

**SSE流测试:**

```bash
# 使用curl测试SSE
curl -N -H "Accept: text/event-stream" \
  http://localhost:8080/v1/workflows/wf-123/logs?stream=true

# 期望输出:
event: connected
data: {"workflow_id":"wf-123"}

event: log
data: {"timestamp":"...","level":"info","message":"Workflow started"}

event: log
data: {"timestamp":"...","level":"info","message":"Step 'Build' started"}

event: close
data: {"reason":"workflow_completed"}
```

### Performance Considerations

**1. Event History大小**

```go
// 对于超长运行的Workflow (>10000 events)
// 考虑分页或限制返回数量
if len(allEvents) > 10000 {
    logger.Warn("Large event history detected",
        zap.Int("event_count", len(allEvents)),
    )
    // 只处理最近N个事件
    allEvents = allEvents[len(allEvents)-5000:]
}
```

**2. SSE连接数限制**

```go
// 使用连接池限制并发SSE连接
var activeStreams int32

func (h *WorkflowHandler) StreamLogs(c *gin.Context) {
    if atomic.LoadInt32(&activeStreams) > 100 {
        c.JSON(503, gin.H{"error": "Too many active streams"})
        return
    }
    
    atomic.AddInt32(&activeStreams, 1)
    defer atomic.AddInt32(&activeStreams, -1)
    
    // ... SSE逻辑 ...
}
```

**3. 日志缓存策略**

```go
// 已完成的Workflow日志可缓存
if workflowStatus == "completed" || workflowStatus == "failed" {
    cache.Set(workflowID, logs, 1*time.Hour)
}
```

### References

**架构设计:**
- [docs/architecture.md §3.1.1](docs/architecture.md) - REST API Handler设计

**技术文档:**
- [Temporal GetWorkflowHistory](https://pkg.go.dev/go.temporal.io/sdk/client#Client.GetWorkflowHistory)
- [SSE Specification](https://html.spec.whatwg.org/multipage/server-sent-events.html)
- [Gin SSE Example](https://github.com/gin-gonic/examples/tree/master/server-sent-event)

**项目上下文:**
- [docs/epics.md Story 1.6](docs/epics.md) - Workflow执行引擎
- [docs/epics.md Story 1.7](docs/epics.md) - 状态查询 (复用Event History逻辑)

### Dependency Graph

```
Story 1.4 (Temporal Client) ──┐
Story 1.6 (执行引擎)         ──┤
Story 1.7 (状态查询)         ──┤
                              ↓
Story 1.8 (日志输出) ← 当前Story
    ↓
    └→ Story 2.x (Agent日志) - 扩展Activity日志支持
```

## Dev Agent Record

### Context Reference

**Source Documents Analyzed:**
1. [docs/epics.md](docs/epics.md) (lines 394-410) - Story 1.8需求定义
2. [docs/architecture.md](docs/architecture.md) (§3.1.1, §6.3) - REST API Handler, 可观测性设计

**Previous Stories:**
- Story 1.1-1.7: 全部drafted (框架、API、解析、Temporal、提交、执行、状态查询)

### Agent Model Used

Claude 3.5 Sonnet (BMM Scrum Master Agent - Bob)

### Estimated Effort

**开发时间:** 8-10小时  
**复杂度:** 中高

**时间分解:**
- 日志模型定义: 1小时
- Event转Log逻辑: 2.5小时
- HTTP Handler (历史日志): 1.5小时
- SSE流实现: 2小时
- 单元测试: 1.5小时
- 集成测试: 1小时
- OpenAPI文档: 0.5小时

**技能要求:**
- Temporal Event History API
- SSE协议
- 事件关联逻辑
- 流式响应处理

### Debug Log References

<!-- Will be populated during implementation -->

### Completion Notes List

<!-- Developer填写完成时的笔记 -->

### File List

**预期创建/修改的文件清单:**

```
新建文件 (~4个):
├── internal/models/
│   └── workflow_log.go                # 日志模型
├── internal/service/
│   ├── workflow_log_service.go        # 日志服务层
│   └── workflow_log_service_test.go   # 单元测试
├── test/integration/
│   ├── test_workflow_logs.sh          # 集成测试
│   └── sse_client.html                # SSE客户端测试

修改文件 (~3个):
├── internal/server/handlers/workflow.go  # 添加GetLogs/StreamLogs方法
├── internal/server/router.go             # 注册GET /v1/workflows/:id/logs
└── api/openapi.yaml                      # 添加日志端点文档
```

**关键代码片段:**

**workflow_log_service.go (核心):**
```go
func (wls *WorkflowLogService) GetLogs(ctx context.Context, workflowID string, filter *LogFilter) (*WorkflowLogsResponse, error) {
    // 1. 获取Event History
    iter := wls.temporalClient.GetClient().GetWorkflowHistory(ctx, workflowID, "", false, ALL_EVENT)
    
    // 2. 收集所有事件
    var allEvents []*HistoryEvent
    for iter.HasNext() {
        allEvents = append(allEvents, iter.Next())
    }
    
    // 3. 转换为日志
    var logs []*LogEntry
    for _, event := range allEvents {
        log := wls.convertEventToLog(event, allEvents)
        if log != nil && wls.matchesFilter(log, filter) {
            logs = append(logs, log)
        }
    }
    
    return &WorkflowLogsResponse{Logs: logs}, nil
}
```

**SSE Handler:**
```go
func (h *WorkflowHandler) StreamLogs(c *gin.Context) {
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    
    logIter := h.workflowLogService.GetRealtimeLogs(ctx, workflowID)
    
    for logIter.HasNext() {
        log := logIter.Next()
        c.SSEvent("log", log)
        c.Writer.Flush()
    }
}
```

---

**Story Ready for Development** ✅

开发者可基于Story 1.1-1.7的成果,实现工作流日志查询和实时流功能。
本Story完成后,用户可以查看工作流执行的详细日志,支持级别过滤和实时流。
