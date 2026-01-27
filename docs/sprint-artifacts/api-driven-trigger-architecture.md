# API 驱动的触发器架构设计

**创建时间:** 2026-01-27  
**版本:** 1.0  
**状态:** Approved

## 背景

Waterflow 采用 **API 驱动架构**，将工作流定义与触发配置分离，实现更灵活的触发器管理。

## 设计理念

### 关注点分离 (Separation of Concerns)

```
┌──────────────────────────────────────────────────────┐
│              Waterflow 架构分层                        │
├──────────────────────────────────────────────────────┤
│                                                       │
│  Layer 1: 工作流定义 (YAML)                           │
│  ┌──────────────────────────────────┐                │
│  │ name: Backup                      │                │
│  │ jobs:                             │ ← 纯逻辑定义   │
│  │   backup:                         │                │
│  │     steps: [...]                  │                │
│  └──────────────────────────────────┘                │
│                                                       │
│  Layer 2: 触发器管理 (REST API)                       │
│  ┌──────────────────────────────────┐                │
│  │ POST /v1/schedules                │                │
│  │ {                                 │                │
│  │   "workflow_name": "Backup",      │ ← 独立配置     │
│  │   "cron": "0 2 * * *"             │                │
│  │ }                                 │                │
│  └──────────────────────────────────┘                │
│                                                       │
│  Layer 3: 执行引擎 (Temporal)                         │
│  ┌──────────────────────────────────┐                │
│  │ Temporal Schedules API            │ ← 持久化执行   │
│  │ Temporal Workflows                │                │
│  └──────────────────────────────────┘                │
│                                                       │
└──────────────────────────────────────────────────────┘
```

### 核心原则

1. **YAML 专注逻辑**
   - 定义工作流的执行逻辑（jobs, steps）
   - 不包含触发配置（cron, webhook）
   - 可选的 `on` 字段仅作元数据标记

2. **API 管理触发**
   - Schedule API 管理定时触发
   - Webhook API 管理事件触发
   - 支持同一工作流多种触发方式

3. **Temporal 负责执行**
   - 持久化调度（Schedules API）
   - 可靠执行（Workflow Engine）
   - 完整可观测性（Event History）

## 架构对比

### 方案 A: API 驱动（✅ 采用）

```yaml
# workflows/backup.yaml - 纯工作流定义
name: Backup
on: ["schedule", "manual"]  # 可选，仅作文档说明
jobs:
  backup:
    runs-on: default
    steps:
      - uses: run@v1
        with:
          command: ./backup.sh
```

```bash
# 通过 API 配置触发器

# 1. 创建工作流
POST /v1/workflows
{
  "name": "Backup",
  "yaml": "..."
}

# 2. 配置定时触发
POST /v1/schedules
{
  "name": "nightly-backup",
  "workflow_name": "Backup",
  "cron": "0 2 * * *"
}

# 3. 配置 Webhook 触发
POST /v1/triggers
{
  "name": "backup-on-push",
  "workflow_name": "Backup",
  "type": "webhook",
  "filters": {"branches": ["main"]}
}

# 4. 手动触发
POST /v1/workflows/executions
{
  "workflow_name": "Backup"
}
```

**优点：**
- ✅ 同一工作流支持多种触发方式
- ✅ 修改 cron 无需更新 YAML
- ✅ 触发配置独立版本控制
- ✅ 符合云原生 API 设计

### 方案 B: 声明式（❌ 未采用）

```yaml
# workflows/backup.yaml - 包含触发配置
name: Backup
on:
  schedule:
    cron: "0 2 * * *"
  push:
    branches: [main]
jobs:
  backup:
    steps: [...]
```

```bash
# 通过 API 提交 YAML
POST /v1/workflows
{
  "yaml": "..."  # YAML 包含一切
}
# API 自动解析 on 字段，创建 Schedule/Webhook
```

**缺点：**
- ❌ 同一工作流难以配置多个 Schedule
- ❌ 修改 cron 需要重新部署 YAML
- ❌ 触发配置与逻辑耦合

## 数据模型

### workflows 表
```sql
CREATE TABLE workflows (
    name TEXT PRIMARY KEY,
    yaml_content TEXT NOT NULL,
    version INTEGER DEFAULT 1,
    created_at DATETIME NOT NULL,
    updated_at DATETIME
);
```

### schedules 表
```sql
CREATE TABLE schedules (
    id TEXT PRIMARY KEY,
    workflow_name TEXT NOT NULL,  -- 引用 workflows.name
    cron TEXT NOT NULL,
    timezone TEXT DEFAULT 'UTC',
    status TEXT NOT NULL CHECK(status IN ('active', 'paused')),
    overlap_policy TEXT NOT NULL DEFAULT 'skip',
    created_at DATETIME NOT NULL,
    FOREIGN KEY (workflow_name) REFERENCES workflows(name) ON DELETE CASCADE
);
```

### triggers 表
```sql
CREATE TABLE triggers (
    id TEXT PRIMARY KEY,
    workflow_name TEXT NOT NULL,  -- 引用 workflows.name
    type TEXT NOT NULL CHECK(type IN ('webhook')),
    webhook_url TEXT,
    secret TEXT NOT NULL,
    filters TEXT,  -- JSON: {"branches": ["main"]}
    status TEXT NOT NULL CHECK(status IN ('enabled', 'disabled')),
    created_at DATETIME NOT NULL,
    FOREIGN KEY (workflow_name) REFERENCES workflows(name) ON DELETE CASCADE
);
```

## API 设计

### Workflow API (Story 1.9)

```bash
# 创建工作流
POST /v1/workflows
{
  "name": "Backup",
  "yaml": "name: Backup\njobs:..."
}

# 查询工作流
GET /v1/workflows/{name}

# 手动执行工作流
POST /v1/workflows/{name}/executions
```

### Schedule API (Story 1.10)

```bash
# 创建定时触发器
POST /v1/schedules
{
  "name": "nightly-backup",
  "workflow_name": "Backup",
  "cron": "0 2 * * *",
  "overlap_policy": "skip"
}

# 暂停/恢复
POST /v1/schedules/{id}/pause
POST /v1/schedules/{id}/resume

# 手动触发
POST /v1/schedules/{id}/trigger
```

### Webhook API (Story 1.11)

```bash
# 创建 Webhook 触发器
POST /v1/triggers
{
  "name": "deploy-on-push",
  "workflow_name": "Deploy",
  "type": "webhook",
  "filters": {
    "branches": ["main"],
    "paths": ["src/**"]
  },
  "secret": "my-secret"
}

# 接收 Webhook
POST /api/v1/webhooks/{trigger_id}
X-Hub-Signature-256: sha256=...
{
  "ref": "refs/heads/main",
  "commits": [...]
}
```

## 使用场景

### 场景 1: 单一定时触发

```bash
# 1. 创建工作流
POST /v1/workflows {"name": "Backup", "yaml": "..."}

# 2. 配置定时触发
POST /v1/schedules {"workflow_name": "Backup", "cron": "0 2 * * *"}
```

### 场景 2: 多个定时触发

```bash
# 同一工作流，不同时间触发
POST /v1/schedules {"workflow_name": "Backup", "cron": "0 2 * * *"}  # 每天 2 点
POST /v1/schedules {"workflow_name": "Backup", "cron": "0 */6 * * *"}  # 每 6 小时
```

### 场景 3: 定时 + Webhook 混合

```bash
# 定时触发
POST /v1/schedules {"workflow_name": "Deploy", "cron": "0 0 * * 0"}

# Webhook 触发（代码推送时）
POST /v1/triggers {
  "workflow_name": "Deploy",
  "type": "webhook",
  "filters": {"branches": ["main"]}
}
```

### 场景 4: 手动触发

```bash
# 不配置任何自动触发器，仅手动执行
POST /v1/workflows/Deploy/executions
```

## 实现细节

### Workflow Store

```go
// pkg/workflow/store.go
type WorkflowStore interface {
    Create(name string, yaml string) error
    GetByName(name string) (*Workflow, error)
    List(filters ...Filter) ([]*Workflow, error)
    Delete(name string) error
}
```

### Schedule Manager

```go
// pkg/schedule/manager.go
func (m *Manager) Create(ctx context.Context, req *CreateScheduleRequest) (*Schedule, error) {
    // 1. 验证 Workflow 存在
    workflow, err := m.workflowStore.GetByName(req.WorkflowName)
    if err != nil {
        return nil, fmt.Errorf("workflow not found: %w", err)
    }
    
    // 2. 创建 Temporal Schedule
    handle, err := m.temporalClient.ScheduleClient().Create(ctx, client.ScheduleOptions{
        ID: req.Name,
        Spec: client.ScheduleSpec{
            CronExpressions: []string{req.Cron},
        },
        Action: &client.ScheduleWorkflowAction{
            Workflow: temporal.RunWorkflowExecutor,
            Args:     []interface{}{workflow},  // ← 传入已存在的 Workflow
        },
    })
    
    // 3. 存储元数据
    return m.storage.Save(&Schedule{
        ID:           req.Name,
        WorkflowName: req.WorkflowName,
        Cron:         req.Cron,
    })
}
```

### Webhook Manager

```go
// pkg/trigger/manager.go
func (m *Manager) HandleWebhook(ctx context.Context, triggerID string, payload []byte) error {
    // 1. 查询 Trigger 配置
    trigger, _ := m.storage.Get(triggerID)
    
    // 2. 应用过滤规则
    if !m.matchFilters(trigger.Filters, payload) {
        return nil  // 不匹配，忽略
    }
    
    // 3. 获取 Workflow
    workflow, _ := m.workflowStore.GetByName(trigger.WorkflowName)
    
    // 4. 触发 Temporal Workflow
    return m.temporalClient.ExecuteWorkflow(ctx, workflow)
}
```

## 迁移路径

对于已有的包含 `on` 字段的 YAML：

### 兼容性处理

```go
// pkg/dsl/parser.go
func Parse(yaml []byte) (*Workflow, error) {
    var wf Workflow
    yaml.Unmarshal(yaml, &wf)
    
    // 如果 YAML 中有 on 字段，输出警告（不阻塞）
    if wf.On != nil && len(wf.On.SupportedTypes) > 0 {
        log.Warn("'on' field is deprecated. Use Schedule/Webhook API instead.")
    }
    
    return &wf, nil
}
```

### 自动迁移工具（未来）

```bash
# 扫描 YAML 中的 on 字段，自动创建对应的 Schedule/Webhook
waterflow migrate-triggers --workflow-dir ./workflows
```

## 相关文档

- [Story 1.3: YAML DSL 解析](./1-3-yaml-dsl-parsing-and-validation.md)
- [Story 1.9: 工作流管理 API](./1-9-workflow-management-api.md)
- [Story 1.10: Schedule API 实现](./1-10-schedule-api-implementation.md)
- [Story 1.11: Webhook Trigger 实现](./1-11-webhook-trigger-implementation.md)
- [ADR-0001: 使用 Temporal 作为工作流引擎](../adr/0001-use-temporal-workflow-engine.md)

## 总结

**API 驱动架构优势：**
1. ✅ **灵活性**: 同一工作流支持多种触发方式
2. ✅ **解耦**: 工作流定义与触发配置分离
3. ✅ **可维护性**: 独立管理触发器，无需修改 YAML
4. ✅ **云原生**: 符合 Kubernetes、Airflow 等现代平台设计

**适用场景：**
- 企业级工作流平台
- 需要频繁调整触发策略的系统
- 多租户环境（不同租户不同触发配置）
- 复杂的触发器管理需求

---

**决策日期:** 2026-01-27  
**决策人:** Scrum Master & Tech Lead  
**状态:** ✅ Approved, 已应用于 Stories 1.10, 1.11
