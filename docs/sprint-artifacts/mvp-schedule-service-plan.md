# MVP 阶段 Schedule Service 实施方案

**决策日期:** 2026-01-27  
**决策者:** Websoft9  
**状态:** 待实施

## 📋 决策概述

**原计划:** MVP 阶段使用纯 API 触发 + 外部 cron，`on` 字段作为预留配置  
**新决策:** MVP 阶段实现基于 Temporal Schedules API 的完整调度功能  

**理由:**
- ✅ Temporal v1.38.0 的 Schedules API 已经非常成熟
- ✅ 一次性实现完整调度，避免后续重构
- ✅ 提供生产级的定时任务管理能力
- ✅ `on` 字段立即发挥实际作用，不只是文档

---

## 🎯 架构设计

### 完整架构图

```
┌─────────────────────────────────────────────────────────┐
│  用户/外部系统                                           │
│  - CLI 工具                                              │
│  - Web UI (未来)                                         │
│  - GitHub Webhook                                        │
│  - 其他集成系统                                          │
└─────────────────────────────────────────────────────────┘
                          ↓ HTTP/JSON
┌─────────────────────────────────────────────────────────┐
│  Waterflow Server (REST API)                            │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Workflow API (Epic 1 已有)                       │  │
│  │  - POST   /v1/workflows         一次性执行        │  │
│  │  - GET    /v1/workflows/:id     查询状态          │  │
│  │  - DELETE /v1/workflows/:id     取消执行          │  │
│  │  - GET    /v1/workflows/:id/logs 查看日志         │  │
│  └───────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Schedule API (Story 1.9 新增)                    │  │
│  │  - POST   /v1/schedules         创建定时工作流     │  │
│  │  - GET    /v1/schedules         列表查询          │  │
│  │  - GET    /v1/schedules/:id     详情查询          │  │
│  │  - PATCH  /v1/schedules/:id     更新配置          │  │
│  │  - POST   /v1/schedules/:id/pause   暂停          │  │
│  │  - POST   /v1/schedules/:id/resume  恢复          │  │
│  │  - DELETE /v1/schedules/:id     删除              │  │
│  │  - POST   /v1/schedules/:id/trigger 手动触发      │  │
│  └───────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Webhook API (Story 1.10 新增)                    │  │
│  │  - POST /webhooks/:schedule-id  外部系统触发      │  │
│  │  - POST /webhooks/:schedule-id/validate 验证配置  │  │
│  └───────────────────────────────────────────────────┘  │
│                                                          │
│  组件:                                                   │
│  - pkg/schedule/          Schedule 管理逻辑            │
│  - internal/api/schedule_handler.go  API 处理器        │
│  - internal/api/webhook_handler.go   Webhook 处理器    │
│  - internal/storage/      Schedule 元数据存储(SQLite)  │
└─────────────────────────────────────────────────────────┘
                          ↓ gRPC
┌─────────────────────────────────────────────────────────┐
│  Temporal Server (v1.38.0)                              │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Schedules API (原生支持)                         │  │
│  │  - Schedule.Create()     创建调度                 │  │
│  │  - Schedule.Pause()      暂停                     │  │
│  │  - Schedule.Unpause()    恢复                     │  │
│  │  - Schedule.Trigger()    手动触发                 │  │
│  │  - Schedule.Delete()     删除                     │  │
│  │  - Schedule.Describe()   查询详情                 │  │
│  │  - Schedule.Update()     更新配置                 │  │
│  └───────────────────────────────────────────────────┘  │
│  - 持久化调度配置和状态                                  │
│  - 自动在 cron 时间触发 Workflow                        │
│  - 提供完整的管理接口                                    │
└─────────────────────────────────────────────────────────┘
                          ↓ Task Queue
┌─────────────────────────────────────────────────────────┐
│  Waterflow Agents                                       │
│  - 执行 Workflow 的 Step                                │
└─────────────────────────────────────────────────────────┘
```

---

## 📦 新增 Story 规格

### Story 1.9: Schedule API 实现

**优先级:** P0 (MVP 必须)  
**估算:** 3-4 天  
**依赖:** Story 1.1, 1.2, 1.3, 1.8

#### Story 描述

As a **工作流用户**,  
I want **通过 API 注册定时工作流**,  
so that **工作流可以按 cron 表达式自动执行，无需外部调度器**。

#### Acceptance Criteria

**AC1: 创建 Schedule**
```bash
POST /v1/schedules
Content-Type: application/json

{
  "name": "nightly-backup",
  "workflow_yaml": "name: Backup\non:\n  schedule:\n    cron: \"0 2 * * *\"\njobs:...",
  "paused": false,
  "overlap_policy": "skip"
}

# 响应
201 Created
{
  "schedule_id": "nightly-backup",
  "status": "active",
  "next_run_time": "2026-01-28T02:00:00Z",
  "created_at": "2026-01-27T10:00:00Z"
}
```

**AC2: 列出所有 Schedules**
```bash
GET /v1/schedules?status=active&limit=20

# 响应
{
  "schedules": [
    {
      "id": "nightly-backup",
      "workflow_name": "Backup",
      "cron": "0 2 * * *",
      "status": "active",
      "next_run_time": "2026-01-28T02:00:00Z",
      "last_run_time": "2026-01-27T02:00:00Z",
      "total_runs": 365
    }
  ],
  "total": 1
}
```

**AC3: 查询 Schedule 详情**
```bash
GET /v1/schedules/nightly-backup

# 响应
{
  "id": "nightly-backup",
  "workflow_yaml": "...",
  "spec": {
    "cron": "0 2 * * *"
  },
  "status": "active",
  "policy": {
    "overlap": "skip",
    "catchup_window": "1h"
  },
  "info": {
    "next_run_time": "2026-01-28T02:00:00Z",
    "last_run_time": "2026-01-27T02:00:00Z",
    "last_run_id": "backup-1735696800",
    "total_runs": 365,
    "recent_runs": [...]
  }
}
```

**AC4: 暂停/恢复 Schedule**
```bash
# 暂停
POST /v1/schedules/nightly-backup/pause
{
  "reason": "maintenance"
}

# 恢复
POST /v1/schedules/nightly-backup/resume
```

**AC5: 手动触发**
```bash
POST /v1/schedules/nightly-backup/trigger

# 响应
{
  "workflow_id": "nightly-backup-manual-1706356800",
  "triggered_at": "2026-01-27T10:00:00Z"
}
```

**AC6: 删除 Schedule**
```bash
DELETE /v1/schedules/nightly-backup

# 响应
204 No Content
```

#### 实现文件

- `pkg/schedule/manager.go` - Schedule 管理器
- `pkg/schedule/types.go` - Schedule 数据结构
- `internal/api/schedule_handler.go` - REST API 处理器
- `internal/storage/schedule_store.go` - SQLite 存储
- `pkg/schedule/manager_test.go` - 单元测试
- `test/integration/schedule_api_test.go` - 集成测试

---

### Story 1.10: Webhook Trigger 实现

**优先级:** P1 (MVP 推荐)  
**估算:** 2-3 天  
**依赖:** Story 1.9

#### Story 描述

As a **系统集成者**,  
I want **通过 Webhook 触发定时工作流**,  
so that **外部系统（如 GitHub）可以触发工作流执行**。

#### Acceptance Criteria

**AC1: Webhook 端点**
```bash
POST /webhooks/deployment-schedule
Content-Type: application/json
X-Webhook-Token: <secret-token>

{
  "event": "push",
  "ref": "refs/heads/main",
  "repository": {
    "name": "my-app",
    "url": "https://github.com/org/my-app"
  }
}

# 响应
202 Accepted
{
  "schedule_id": "deployment-schedule",
  "workflow_id": "deployment-1706356800",
  "status": "triggered"
}
```

**AC2: Token 验证**
- 支持 Bearer Token 验证
- 支持 HMAC 签名验证（GitHub/GitLab 风格）
- 验证失败返回 401 Unauthorized

**AC3: GitHub Webhook 集成示例**
```yaml
# GitHub Webhook 配置
name: Deploy on Push
on:
  schedule:
    # 基础调度（每天）
    cron: "0 2 * * *"
  webhook:
    # Webhook 触发
    events: [push, pull_request]

# Waterflow 创建 Schedule 时:
# 1. 注册 cron 调度
# 2. 生成 webhook URL: /webhooks/<schedule-id>
# 3. 用户在 GitHub 配置此 URL
```

#### 实现文件

- `internal/api/webhook_handler.go` - Webhook 处理器
- `pkg/webhook/validator.go` - Token/签名验证
- `pkg/webhook/github.go` - GitHub Webhook 适配器
- `internal/api/webhook_handler_test.go` - 单元测试

---

## 🔄 Story 1.3 调整内容

### 关键变更

1. **AC1 增加说明**
```markdown
**And** 解析 `on` 字段配置:
- `on: push` - 简单触发器（预留，MVP 不实现）
- `on.schedule.cron` - Cron 调度（Story 1.9 实现）
- `on.webhook.events` - Webhook 触发（Story 1.10 实现）

**MVP 阶段:** `on.schedule` 将用于创建 Temporal Schedule（Story 1.9）
```

2. **TriggerConfig 注释更新**
```go
// TriggerConfig 触发器配置（MVP 实现 schedule 和 webhook）
type TriggerConfig struct {
    Push     *PushTrigger     `yaml:"push,omitempty" json:"push,omitempty"`     // 预留
    Schedule *ScheduleTrigger `yaml:"schedule,omitempty" json:"schedule,omitempty"` // Story 1.9
    Webhook  *WebhookTrigger  `yaml:"webhook,omitempty" json:"webhook,omitempty"`  // Story 1.10
}
```

3. **移除"外部 cron"说明**
- 删除所有提到"外部 crontab"的内容
- 更新为"基于 Temporal Schedules 的内置调度"

---

## 💻 核心实现示例

### Schedule Manager

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
    // 1. 解析 YAML
    workflow, err := dsl.Parse([]byte(req.WorkflowYAML))
    if err != nil {
        return nil, fmt.Errorf("invalid workflow YAML: %w", err)
    }
    
    // 2. 验证必须有 on.schedule
    if workflow.On.Schedule == nil {
        return nil, fmt.Errorf("workflow must have on.schedule configuration")
    }
    
    // 3. 创建 Temporal Schedule
    scheduleClient := m.temporalClient.GetClient().ScheduleClient()
    
    handle, err := scheduleClient.Create(ctx, client.ScheduleOptions{
        ID: req.Name,
        
        Spec: client.ScheduleSpec{
            CronExpressions: []string{workflow.On.Schedule.Cron},
        },
        
        Action: &client.ScheduleWorkflowAction{
            ID:        fmt.Sprintf("%s-{{.ScheduledTime.Unix}}", req.Name),
            Workflow:  temporal.RunWorkflowExecutor,
            Args:      []interface{}{workflow},
            TaskQueue: "default",
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
        ID:           req.Name,
        WorkflowYAML: req.WorkflowYAML,
        WorkflowName: workflow.Name,
        Cron:         workflow.On.Schedule.Cron,
        Status:       "active",
        CreatedAt:    time.Now(),
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
        zap.String("cron", workflow.On.Schedule.Cron),
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
func (m *Manager) Trigger(ctx context.Context, id string) (string, error) {
    scheduleClient := m.temporalClient.GetClient().ScheduleClient()
    handle := scheduleClient.GetHandle(ctx, id)
    
    err := handle.Trigger(ctx, client.ScheduleTriggerOptions{
        Overlap: client.ScheduleOverlapPolicyAllowAll,
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

---

## 📊 实施计划

### Phase 1: Story 1.3 调整（1 天）
- [ ] 更新 AC1 说明（`on` 字段用途）
- [ ] 更新 TriggerConfig 注释
- [ ] 移除"外部 cron"相关说明
- [ ] 更新文档引用

### Phase 2: Story 1.9 实现（3-4 天）
- [ ] Day 1: Schedule Manager 核心逻辑
- [ ] Day 2: REST API 端点实现
- [ ] Day 3: SQLite 存储 + 单元测试
- [ ] Day 4: 集成测试 + 文档

### Phase 3: Story 1.10 实现（2-3 天）
- [ ] Day 1: Webhook Handler + Token 验证
- [ ] Day 2: GitHub Webhook 适配器
- [ ] Day 3: 测试 + 文档

### Phase 4: 集成验证（1 天）
- [ ] 端到端测试
- [ ] 性能测试
- [ ] 文档更新

**总计: 7-9 天**

---

## ✅ Definition of Done

- [ ] Story 1.9 完成并通过所有测试
- [ ] Story 1.10 完成并通过所有测试
- [ ] Story 1.3 文档更新完成
- [ ] 集成测试覆盖所有 Schedule API
- [ ] Webhook 集成测试通过
- [ ] API 文档更新（OpenAPI）
- [ ] 用户文档更新（快速开始指南）
- [ ] 性能测试通过（1000+ Schedules）
- [ ] Code Review 通过
- [ ] 生产环境部署验证

---

## 🎯 成功标准

**功能验收:**
- ✅ 用户可以通过 API 创建定时工作流
- ✅ Cron 调度自动执行（无需外部 cron）
- ✅ Webhook 触发正常工作
- ✅ Schedule 管理功能完整（暂停/恢复/删除）

**性能要求:**
- ✅ 创建 Schedule < 100ms
- ✅ 触发延迟 < 1s
- ✅ 支持 1000+ 并发 Schedules

**可靠性:**
- ✅ Temporal Server 重启后 Schedule 继续工作
- ✅ Waterflow Server 重启后元数据完整
- ✅ 错误自动重试和告警

---

## 📚 参考文档

- [Temporal Schedules Documentation](https://docs.temporal.io/workflows#schedule)
- [Temporal Go SDK v1.38.0 API](https://pkg.go.dev/go.temporal.io/sdk@v1.38.0/client#ScheduleClient)
- [ADR-0004: YAML DSL 语法设计](../adr/0004-yaml-dsl-syntax.md)
- [Story 1.8: Temporal SDK 集成](./1-8-temporal-sdk-integration.md)

---

**最后更新:** 2026-01-27  
**下一步:** 开始实施 Story 1.3 调整
