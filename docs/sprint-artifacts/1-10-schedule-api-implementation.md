# Story 1.10: Schedule API 实现（基于 Temporal Schedules）

Status: not-started

## Story

As a **工作流用户**,  
I want **通过 API 注册定时工作流**,  
so that **工作流可以按 cron 表达式自动执行，无需配置外部调度器**。

## Context

这是 Epic 1 的第十个 Story，实现基于 Temporal Schedules API 的完整定时调度功能。在 Story 1.1-1.9 的基础上，本 Story 提供独立的 Schedule 管理 API，实现工作流的定时自动执行。

**设计理念（API 驱动）:**
- 工作流定义（YAML）与触发配置（Schedule）分离
- 同一工作流可配置多个不同的 Schedule
- Schedule 独立管理：创建/暂停/恢复/删除

**前置依赖:**
- Story 1.1 (Server 框架、日志系统) 已完成
- Story 1.2 (REST API 框架、错误处理) 已完成
- Story 1.3 (YAML 解析、Workflow 数据结构) 已完成
- Story 1.8 (Temporal SDK 集成) 已完成

**Epic 背景:**  
根据 [mvp-schedule-service-plan.md](./mvp-schedule-service-plan.md)，Waterflow 在 MVP 阶段就实现完整的调度能力，充分利用 Temporal v1.38.0 的 Schedules API，提供一站式的定时任务管理体验。

**业务价值:**
- API 驱动架构：工作流定义与触发配置分离，更灵活
- 同一工作流可配置多个 Schedule（不同 cron 表达式）
- 基于 Temporal 的持久化调度，服务器重启后继续执行
- 完整的管理 API（暂停/恢复/查询/手动触发）
- 为 Webhook 触发（Story 1.11）奠定架构基础

**技术价值:**
- 充分利用 Temporal Schedules API（v1.17+）的成熟能力
- 避免自己实现调度器，降低复杂度
- 继承 Temporal 的可靠性和可观测性

## Acceptance Criteria

### AC1: 创建 Schedule

**Given** 用户已有工作流定义  
**When** 调用 POST /v1/schedules 配置定时触发  
**Then** 创建 Temporal Schedule 并返回详情

**请求示例:**
```bash
POST /v1/schedules
Content-Type: application/json

{
  "name": "nightly-backup",
  "workflow_name": "Backup",
  "cron": "0 2 * * *",
  "paused": false,
  "overlap_policy": "skip",
  "timezone": "UTC"
}
```

**响应示例:**
```json
{
  "schedule_id": "nightly-backup",
  "workflow_name": "Backup",
  "cron": "0 2 * * *",
  "status": "active",
  "next_run_time": "2026-01-28T02:00:00Z",
  "created_at": "2026-01-27T10:00:00Z",
  "overlap_policy": "skip"
}
```

**And** Schedule 成功注册到 Temporal Server  
**And** 元数据存储到 SQLite  
**And** 返回 201 Created 状态码

**错误处理:**
- Workflow 不存在 → 404 Not Found ("workflow 'Backup' not found")
- Cron 表达式无效 → 400 Bad Request ("invalid cron expression")
- Schedule ID 已存在 → 409 Conflict
- Temporal 连接失败 → 503 Service Unavailable

### AC2: 列出所有 Schedules

**Given** 系统中有多个已注册的 Schedules  
**When** 调用 GET /v1/schedules  
**Then** 返回 Schedule 列表（分页）

**请求示例:**
```bash
GET /v1/schedules?status=active&limit=20&offset=0

# 支持过滤参数
GET /v1/schedules?status=paused
GET /v1/schedules?workflow_name=Backup
```

**响应示例:**
```json
{
  "schedules": [
    {
      "id": "nightly-backup",
      "workflow_name": "Backup",
      "cron": "0 2 * * *",
      "status": "active",
      "next_run_time": "2026-01-28T02:00:00Z",
      "last_run_time": "2026-01-27T02:00:00Z",
      "last_run_id": "nightly-backup-1735696800",
      "total_runs": 365,
      "created_at": "2025-01-27T10:00:00Z"
    },
    {
      "id": "weekly-report",
      "workflow_name": "Generate Report",
      "cron": "0 9 * * 1",
      "status": "active",
      "next_run_time": "2026-02-03T09:00:00Z",
      "last_run_time": "2026-01-27T09:00:00Z",
      "total_runs": 52
    }
  ],
  "total": 2,
  "limit": 20,
  "offset": 0
}
```

**And** 支持分页（默认 limit=20, offset=0）  
**And** 支持按状态过滤（active, paused）  
**And** 支持按工作流名称过滤

### AC3: 查询 Schedule 详情

**Given** Schedule 已创建  
**When** 调用 GET /v1/schedules/:id  
**Then** 返回完整的 Schedule 详情

**请求示例:**
```bash
GET /v1/schedules/nightly-backup
```

**响应示例:**
```json
{
  "id": "nightly-backup",
  "workflow_name": "Backup",
  "workflow_yaml": "name: Backup\non:\n  schedule:\n    cron: \"0 2 * * *\"\njobs:...",
  "spec": {
    "cron": "0 2 * * *"
  },
  "status": "active",
  "policy": {
    "overlap": "skip",
    "catchup_window": "1h0m0s",
    "pause_on_failure": false
  },
  "info": {
    "next_run_time": "2026-01-28T02:00:00Z",
    "last_run_time": "2026-01-27T02:00:00Z",
    "last_run_id": "nightly-backup-1735696800",
    "last_run_status": "completed",
    "total_runs": 365,
    "recent_runs": [
      {
        "workflow_id": "nightly-backup-1735696800",
        "start_time": "2026-01-27T02:00:00Z",
        "end_time": "2026-01-27T02:05:32Z",
        "status": "completed"
      },
      {
        "workflow_id": "nightly-backup-1735610400",
        "start_time": "2026-01-26T02:00:00Z",
        "end_time": "2026-01-26T02:04:18Z",
        "status": "completed"
      }
    ]
  },
  "created_at": "2025-01-27T10:00:00Z",
  "updated_at": "2026-01-27T02:00:00Z"
}
```

**And** 包含完整的 YAML 定义  
**And** 包含 Temporal Schedule 的实时状态  
**And** 包含最近执行记录（最多 10 条）

**错误处理:**
- Schedule 不存在 → 404 Not Found

### AC4: 暂停和恢复 Schedule

**Given** Schedule 处于 active 状态  
**When** 调用 POST /v1/schedules/:id/pause  
**Then** Schedule 被暂停，不再自动触发

**暂停请求:**
```bash
POST /v1/schedules/nightly-backup/pause
Content-Type: application/json

{
  "reason": "System maintenance"
}
```

**暂停响应:**
```json
{
  "schedule_id": "nightly-backup",
  "status": "paused",
  "paused_at": "2026-01-27T10:00:00Z",
  "paused_by": "admin",
  "reason": "System maintenance"
}
```

**恢复请求:**
```bash
POST /v1/schedules/nightly-backup/resume
Content-Type: application/json

{
  "reason": "Maintenance complete"
}
```

**恢复响应:**
```json
{
  "schedule_id": "nightly-backup",
  "status": "active",
  "resumed_at": "2026-01-27T12:00:00Z",
  "next_run_time": "2026-01-28T02:00:00Z"
}
```

**And** 暂停后的 Schedule 不会触发新的 Workflow  
**And** 已运行的 Workflow 不受影响（继续执行）  
**And** 恢复后的 Schedule 计算下次执行时间

### AC5: 手动触发 Schedule

**Given** Schedule 已创建（无论 active 或 paused）  
**When** 调用 POST /v1/schedules/:id/trigger  
**Then** 立即启动一个 Workflow Run（不等待 cron 时间）

**请求示例:**
```bash
POST /v1/schedules/nightly-backup/trigger
Content-Type: application/json

{
  "overlap": "allow_all"
}
```

**响应示例:**
```json
{
  "schedule_id": "nightly-backup",
  "workflow_id": "nightly-backup-manual-1706356800",
  "run_id": "abc123...",
  "triggered_at": "2026-01-27T10:00:00Z",
  "status": "running"
}
```

**And** 触发不影响正常的 cron 调度  
**And** 支持 overlap 策略配置（allow_all, skip, cancel_other）  
**And** 返回新启动的 Workflow ID

### AC6: 更新 Schedule 配置

**Given** Schedule 已创建  
**When** 调用 PATCH /v1/schedules/:id  
**Then** 更新 Schedule 配置

**请求示例:**
```bash
PATCH /v1/schedules/nightly-backup
Content-Type: application/json

{
  "cron": "0 3 * * *",
  "overlap_policy": "cancel_other",
  "timezone": "Asia/Shanghai"
}
```
```

**响应示例:**
```json
{
  "schedule_id": "nightly-backup",
  "workflow_name": "Backup",
  "cron": "0 3 * * *",
  "status": "active",
  "overlap_policy": "cancel_other",
  "updated_at": "2026-01-27T10:00:00Z"
}
```

**And** 更新会立即生效  
**And** 重新计算下次执行时间  
**And** 已运行的 Workflow 不受影响

### AC7: 删除 Schedule

**Given** Schedule 已创建  
**When** 调用 DELETE /v1/schedules/:id  
**Then** Schedule 被删除

**请求示例:**
```bash
DELETE /v1/schedules/nightly-backup
```

**响应:**
```
204 No Content
```

**And** Temporal Schedule 被删除  
**And** 本地元数据被删除  
**And** 已运行的 Workflow 不受影响（继续执行）  
**And** 删除后无法恢复（需要重新创建）

### AC8: Overlap 策略支持

**Given** Schedule 配置了 overlap 策略  
**When** 上次执行尚未完成，新的调度时间到达  
**Then** 根据策略处理

**支持的策略:**
- **skip**: 跳过新的执行（默认）
- **allow_all**: 允许并发执行
- **cancel_other**: 取消上次执行，启动新的
- **buffer_one**: 缓存一个待执行的任务

**配置示例:**
```yaml
name: Long Running Job
on:
  schedule:
    cron: "*/5 * * * *"  # 每 5 分钟
    overlap: skip  # 如果上次还在运行，跳过本次
```

**And** overlap 策略在创建时配置  
**And** 可以通过 PATCH 更新

## Tasks / Subtasks

### Task 1: Schedule Manager 核心逻辑
- [ ] 创建 pkg/schedule/types.go - Schedule 数据结构
- [ ] 创建 pkg/schedule/manager.go - Schedule 管理器
  - [ ] Create() - 创建 Schedule
  - [ ] List() - 列出 Schedules
  - [ ] Get() - 查询详情
  - [ ] Pause() - 暂停
  - [ ] Resume() - 恢复
  - [ ] Trigger() - 手动触发
  - [ ] Update() - 更新配置
  - [ ] Delete() - 删除
- [ ] 集成 Temporal Schedules API

**Schedule Manager 实现示例:**
```go
// pkg/schedule/manager.go
package schedule

import (
    "context"
    "fmt"
    "time"
    
    "github.com/Websoft9/waterflow/pkg/dsl"
    "github.com/Websoft9/waterflow/pkg/temporal"
    "go.temporal.io/sdk/client"
    "go.uber.org/zap"
)

type Manager struct {
    temporalClient *temporal.Client
    storage        *Storage
    logger         *zap.Logger
}

func NewManager(temporalClient *temporal.Client, storage *Storage, logger *zap.Logger) *Manager {
    return &Manager{
        temporalClient: temporalClient,
        storage:        storage,
        logger:         logger,
    }
}

// Create 创建新的 Schedule
func (m *Manager) Create(ctx context.Context, req *CreateScheduleRequest) (*Schedule, error) {
    // 1. 验证 Workflow 是否存在
    workflow, err := m.workflowStore.GetByName(req.WorkflowName)
    if err != nil {
        return nil, fmt.Errorf("workflow '%s' not found: %w", req.WorkflowName, err)
    }
    
    // 2. 验证 Cron 表达式
    if _, err := cron.ParseStandard(req.Cron); err != nil {
        return nil, fmt.Errorf("invalid cron expression: %w", err)
    }
    
    // 3. 创建 Temporal Schedule
    scheduleClient := m.temporalClient.GetClient().ScheduleClient()
    
    handle, err := scheduleClient.Create(ctx, client.ScheduleOptions{
        ID: req.Name,
        
        Spec: client.ScheduleSpec{
            CronExpressions: []string{req.Cron},
            Timezone:        req.Timezone,
        },
        
        Action: &client.ScheduleWorkflowAction{
            ID:        fmt.Sprintf("%s-{{.ScheduledTime.Unix}}", req.Name),
            Workflow:  temporal.RunWorkflowExecutor,
            Args:      []interface{}{workflow}, // 传入已存在的 Workflow
            TaskQueue: "default",
            
            WorkflowExecutionTimeout: 24 * time.Hour,
        },
        
        Policy: &client.SchedulePolicy{
            Overlap:       req.OverlapPolicy,
            CatchupWindow: 1 * time.Hour,
        },
        
        Paused: req.Paused,
    })
    
    if err != nil {
        return nil, fmt.Errorf("failed to create Temporal schedule: %w", err)
    }
    
    // 4. 存储元数据到 SQLite
    schedule := &Schedule{
        ID:            req.Name,
        WorkflowName:  req.WorkflowName,
        Cron:          req.Cron,
        Timezone:      req.Timezone,
        Status:        ternary(req.Paused, "paused", "active"),
        OverlapPolicy: string(req.OverlapPolicy),
        CreatedAt:     time.Now(),
    }
    
    if err := m.storage.Save(schedule); err != nil {
        // 回滚 Temporal Schedule
        handle.Delete(ctx)
        return nil, fmt.Errorf("failed to save schedule metadata: %w", err)
    }
    
    // 5. 查询下次执行时间
    desc, _ := handle.Describe(ctx)
    if desc.Schedule.State != nil && desc.Schedule.State.NextActionTime != nil {
        schedule.NextRunTime = *desc.Schedule.State.NextActionTime
    }
    
    m.logger.Info("Schedule created successfully",
        zap.String("schedule_id", req.Name),
        zap.String("workflow_name", req.WorkflowName),
        zap.String("cron", req.Cron),
        zap.String("status", schedule.Status),
    )
    
    return schedule, nil
}

// Pause 暂停 Schedule
func (m *Manager) Pause(ctx context.Context, id string, reason string) error {
    scheduleClient := m.temporalClient.GetClient().ScheduleClient()
    handle := scheduleClient.GetHandle(ctx, id)
    
    err := handle.Pause(ctx, client.SchedulePauseOptions{
        Note: reason,
    })
    
    if err != nil {
        return fmt.Errorf("failed to pause schedule: %w", err)
    }
    
    // 更新本地状态
    return m.storage.UpdateStatus(id, "paused")
}

// Trigger 手动触发
func (m *Manager) Trigger(ctx context.Context, id string, overlap client.ScheduleOverlapPolicy) (string, error) {
    scheduleClient := m.temporalClient.GetClient().ScheduleClient()
    handle := scheduleClient.GetHandle(ctx, id)
    
    err := handle.Trigger(ctx, client.ScheduleTriggerOptions{
        Overlap: overlap,
    })
    
    if err != nil {
        return "", fmt.Errorf("failed to trigger schedule: %w", err)
    }
    
    workflowID := fmt.Sprintf("%s-manual-%d", id, time.Now().Unix())
    
    m.logger.Info("Schedule triggered manually",
        zap.String("schedule_id", id),
        zap.String("workflow_id", workflowID),
    )
    
    return workflowID, nil
}
```

### Task 2: SQLite 存储实现
- [ ] 创建 internal/storage/schedule_store.go
- [ ] 设计 schedules 表结构
- [ ] 实现 CRUD 操作
- [ ] 添加索引优化查询

**表结构设计:**
```sql
CREATE TABLE schedules (
    id TEXT PRIMARY KEY,
    workflow_name TEXT NOT NULL,  -- 引用的工作流名称
    cron TEXT NOT NULL,
    timezone TEXT DEFAULT 'UTC',
    status TEXT NOT NULL CHECK(status IN ('active', 'paused')),
    overlap_policy TEXT NOT NULL DEFAULT 'skip',
    next_run_time DATETIME,
    last_run_time DATETIME,
    last_run_id TEXT,
    total_runs INTEGER DEFAULT 0,
    created_at DATETIME NOT NULL,
    updated_at DATETIME,
    paused_at DATETIME,
    paused_by TEXT,
    pause_reason TEXT,
    FOREIGN KEY (workflow_name) REFERENCES workflows(name) ON DELETE CASCADE
);

CREATE INDEX idx_schedules_status ON schedules(status);
CREATE INDEX idx_schedules_workflow_name ON schedules(workflow_name);
CREATE INDEX idx_schedules_next_run_time ON schedules(next_run_time);
```

### Task 3: REST API Handler 实现
- [ ] 创建 internal/api/schedule_handler.go
- [ ] 实现所有 API 端点
  - [ ] POST /v1/schedules
  - [ ] GET /v1/schedules
  - [ ] GET /v1/schedules/:id
  - [ ] POST /v1/schedules/:id/pause
  - [ ] POST /v1/schedules/:id/resume
  - [ ] POST /v1/schedules/:id/trigger
  - [ ] PATCH /v1/schedules/:id
  - [ ] DELETE /v1/schedules/:id
- [ ] 集成到 Router

**Router 集成:**
```go
// internal/api/router.go
func NewRouter(/* ... */, scheduleHandler *ScheduleHandler) *mux.Router {
    r := mux.NewRouter()
    
    // ... 已有路由
    
    // Schedule API (Story 1.9)
    r.HandleFunc("/v1/schedules", scheduleHandler.CreateSchedule).Methods("POST")
    r.HandleFunc("/v1/schedules", scheduleHandler.ListSchedules).Methods("GET")
    r.HandleFunc("/v1/schedules/{id}", scheduleHandler.GetSchedule).Methods("GET")
    r.HandleFunc("/v1/schedules/{id}/pause", scheduleHandler.PauseSchedule).Methods("POST")
    r.HandleFunc("/v1/schedules/{id}/resume", scheduleHandler.ResumeSchedule).Methods("POST")
    r.HandleFunc("/v1/schedules/{id}/trigger", scheduleHandler.TriggerSchedule).Methods("POST")
    r.HandleFunc("/v1/schedules/{id}", scheduleHandler.UpdateSchedule).Methods("PATCH")
    r.HandleFunc("/v1/schedules/{id}", scheduleHandler.DeleteSchedule).Methods("DELETE")
    
    return r
}
```

### Task 4: 单元测试
- [ ] pkg/schedule/manager_test.go - Manager 单元测试
- [ ] internal/storage/schedule_store_test.go - 存储测试
- [ ] internal/api/schedule_handler_test.go - API 测试

### Task 5: 集成测试
- [ ] test/integration/schedule_api_test.go
- [ ] 测试完整的创建→查询→暂停→恢复→删除流程
- [ ] 测试手动触发
- [ ] 测试 overlap 策略

### Task 6: 文档更新
- [ ] 更新 API 文档（OpenAPI）
- [ ] 更新用户文档（快速开始指南）
- [ ] 添加 Schedule API 示例

## Technical Requirements

### Technology Stack
- **Temporal SDK:** go.temporal.io/sdk v1.38.0 (Schedules API 支持)
- **存储:** SQLite 3 (Schedule 元数据)
- **HTTP Router:** gorilla/mux v1.8+ (已有)
- **日志:** uber-go/zap v1.26+ (已有)

### Architecture Constraints

**ADR 遵循:**
- [ADR-0001: 使用 Temporal 作为工作流引擎](../adr/0001-use-temporal-workflow-engine.md)
- [ADR-0004: YAML DSL 语法设计](../adr/0004-yaml-dsl-syntax.md)

**设计原则:**
- 充分利用 Temporal Schedules API，不重复造轮子
- Schedule 元数据本地存储（快速查询），状态同步自 Temporal
- 事务性操作：创建/删除时确保 Temporal 和本地存储一致
- 错误处理：Temporal 操作失败时回滚本地状态

### Performance Requirements

**创建/删除性能:**
- 创建 Schedule: < 100ms (P95)
- 删除 Schedule: < 50ms (P95)

**查询性能:**
- 列表查询: < 50ms (1000 schedules)
- 详情查询: < 20ms

**并发性能:**
- 支持 1000+ 并发 Schedules
- 支持 100+ 并发 API 请求

### Security Requirements

- **身份验证:** 集成现有的认证机制（如有）
- **授权:** Schedule 操作需要管理员权限
- **输入验证:** 验证 cron 表达式格式
- **SQL 注入防护:** 使用参数化查询

## Definition of Done

- [ ] 所有 Acceptance Criteria 验收通过
- [ ] 所有 Tasks 完成并测试通过
- [ ] 单元测试覆盖率 ≥85%
- [ ] 集成测试通过（完整流程）
- [ ] 性能测试通过（1000+ Schedules）
- [ ] 代码通过 golangci-lint 检查
- [ ] API 文档更新（OpenAPI）
- [ ] 用户文档更新
- [ ] Code Review 通过
- [ ] 与 Temporal Server 集成测试通过
- [ ] 生产环境验证（至少运行 24 小时）

## References

### Architecture Documents
- [mvp-schedule-service-plan.md](./mvp-schedule-service-plan.md) - 完整实施方案
- [ADR-0001: 使用 Temporal 作为工作流引擎](../adr/0001-use-temporal-workflow-engine.md)

### PRD Requirements
- [PRD - Epic 1: 核心工作流引擎](../prd.md)
- [PRD - Schedule Service](../prd.md#schedule-service)

### Previous Stories
- [Story 1.3: YAML DSL 解析和验证](./1-3-yaml-dsl-parsing-and-validation.md) - on.schedule 数据结构
- [Story 1.8: Temporal SDK 集成](./1-8-temporal-sdk-integration.md) - Temporal Client

### External Resources
- [Temporal Schedules Documentation](https://docs.temporal.io/workflows#schedule)
- [Temporal Go SDK v1.38.0 Schedules API](https://pkg.go.dev/go.temporal.io/sdk@v1.38.0/client#ScheduleClient)
- [Cron Expression Format](https://pkg.go.dev/github.com/robfig/cron#hdr-CRON_Expression_Format)

---

**Story 创建时间:** 2026-01-27  
**预计工作量:** 3-4 天  
**优先级:** P0 (MVP 必须)
