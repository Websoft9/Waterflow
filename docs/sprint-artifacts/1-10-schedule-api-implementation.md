# Story 1.10: Schedule API 实现（基于 Temporal Schedules）

Status: not-started

> **⚠️ 架构变更通知 (ADR-0009, 2026-02-03)**
>
> 本 Story 基于 [ADR-0009](../adr/0009-workflow-definition-execution-separation.md) 进行了 API 重设计：
>
> **核心变更:**
> - Schedule API 挂载到 `/v1/workflows/{name}/schedules` 下
> - Schedule 引用已存储的工作流定义 (Definition)，而非 YAML 文件
> - 支持 vars 参数绑定，执行时覆盖 YAML 默认值
>
> **新 API 结构:**
> ```
> POST   /v1/workflows/{name}/schedules              # 创建 Schedule
> GET    /v1/workflows/{name}/schedules              # 列出 Schedules
> GET    /v1/workflows/{name}/schedules/{schedule_id} # 获取详情
> PUT    /v1/workflows/{name}/schedules/{schedule_id} # 更新 Schedule
> DELETE /v1/workflows/{name}/schedules/{schedule_id} # 删除 Schedule
> POST   /v1/workflows/{name}/schedules/{schedule_id}/trigger # 手动触发
> ```
>
> **参数三层覆盖机制:**
> 1. YAML 中的 vars 默认值
> 2. Schedule 创建时绑定的 vars
> 3. 手动触发时传入的 vars（最高优先级）
>
> **详见:** [api-inventory.md](../api-inventory.md), [ADR-0009](../adr/0009-workflow-definition-execution-separation.md)

## Story

As a **工作流用户**,  
I want **通过 API 为已创建的工作流配置定时调度**,  
so that **工作流可以按 cron 表达式自动执行，支持灵活的调度策略**。

## Context

这是 Epic 1 的第十个 Story，实现基于 Temporal Schedules API 的完整定时调度功能。在 Story 1.1-1.9 的基础上，本 Story 提供独立的 Schedule 管理 API，实现工作流的定时自动执行。

**设计理念（API 驱动）:**
- 工作流定义（YAML）与触发配置（Schedule）分离
- 同一工作流可配置多个不同的 Schedule
- Schedule 独立管理：创建/暂停/恢复/删除

**架构设计原则：**
> ✅ **完全遵循 Temporal 原生模式 - 无需额外存储**  
> 
> **Temporal Schedules 的工作原理：**
> 1. 创建 Schedule 时，Temporal **不验证** Workflow Type 是否存在
> 2. Temporal 只存储对 Workflow Type 的**字符串引用**
> 3. 只有在实际触发时，才会查找 Worker 上注册的 Workflow
> 4. 如果找不到 Workflow，Schedule 执行失败（记录在 Temporal 历史中）
> 
> **Waterflow 的简化架构：**
> - **无需 Workflow Definition API**
> - **无需 Workflow Registry 数据库**
> - **只需文件系统存储 YAML**（/opt/waterflow/workflows/*.yaml）
> - Worker 启动时扫描目录，自动注册所有 Workflow

**前置依赖:**
- Story 1.1 (Server 框架、日志系统) 已完成
- Story 1.2 (REST API 框架、错误处理) 已完成
- Story 1.3 (YAML 解析、Workflow 数据结构) 已完成
- Story 1.8 (Temporal SDK 集成) 已完成
- **Workflow 管理：** 文件系统存储（/opt/waterflow/workflows/）
  - 用户手动上传 YAML 文件到该目录
  - 或通过简单的文件上传 API（可选）
  - Worker 启动时自动扫描并注册

**Epic 背景:**  
根据 [mvp-schedule-service-plan.md](./mvp-schedule-service-plan.md)，Waterflow 在 MVP 阶段就实现完整的调度能力，充分利用 Temporal v1.38.0 的 Schedules API，提供一站式的定时任务管理体验。

**业务价值:**
- **API 驱动架构：** 工作流定义与触发配置分离，更灵活
- **多调度支持：** 同一工作流可配置多个 Schedule（不同 cron、不同参数）
- **基于 Temporal 的持久化调度：** 服务器重启后继续执行
- **完整的管理 API：** 暂停/恢复/查询/手动触发已创建的 Schedule
- **符合 Temporal 原生模型：** Workflow 与 Schedule 分离，清晰的职责划分

**架构决策 - Git仓库存储 + 声明式触发 + API 增强：**
- ✅ **YAML 存储在 Git 仓库** - 版本控制、Code Review、GitOps 天然支持
- ✅ **可选 `on` 字段声明触发器** - Server 解析后自动创建 Temporal Schedule（借鉴 GitHub Actions）
- ✅ **Schedule 只存储配置** - 不存储完整 YAML，通过 `workflowName` 引用
- ✅ **API 动态管理触发器** - 支持运行时创建/修改/删除 Schedule（超越 GitHub Actions）
- ✅ **一对多关系** - 一个 Workflow 可以有多个 Schedule/Webhook/手动触发
- ✅ **符合 Temporal 设计哲学** - Workflow Definition (YAML in Git) + Schedule (API/声明式) 分离

**与 GitHub Actions 的对比：**
| 特性 | GitHub Actions | Waterflow |
|------|----------------|-----------|
| YAML 存储 | Git 仓库 (.github/workflows/) | **Git 仓库** (推荐) or 文件系统/数据库 |
| 触发器定义 | YAML 内 `on` 字段（强制） | **`on` 字段（可选）+ REST API（灵活）** |
| 运行时加载 | GitHub 自动检测 YAML 变更 | **Server 主动 Clone/Pull Git 仓库** |
| 动态管理 | ❌ 无法运行时修改 Schedule | **✅ REST API 动态创建/更新/删除 Schedule** |
| 适用场景 | CI/CD 单一场景 | **复杂企业工作流 + DevOps 自动化** |

**Waterflow 的优势：**
- ✅ **GitOps 最佳实践** - YAML 版本控制 + Code Review
- ✅ **声明式 + 命令式双模式** - 既支持 `on` 字段声明，也支持 API 动态管理
- ✅ **灵活的触发器管理** - 同一 Workflow 可以有多个独立 Schedule
- ✅ **SaaS 友好** - 可选数据库存储支持多租户隔离

**完整工作流（GitOps 模式）：**
```
阶段 1：提交 Workflow 到 Git 仓库
──────────────────────────────────────────────────
1. 用户创建 YAML 文件（可选包含 on 字段）
   workflows/backup.yaml:
   ───────────────────────────────────
   name: Backup Database
   on:
     schedule:
       - cron: "0 2 * * *"
   jobs:
     backup:
       runs-on: default
       steps:
         - name: Backup PostgreSQL
           uses: shell@v1
           with:
             cmd: pg_dump mydb > backup.sql
   ───────────────────────────────────

2. 用户提交到 Git 仓库
   git add workflows/backup.yaml
   git commit -m "Add backup workflow"
   git push origin main

阶段 2：Waterflow Server 同步和处理
──────────────────────────────────────────────────
3. Server 检测到 Git 仓库变更（Webhook 或定时 Pull）
   
4. Server 克隆/拉取最新代码
   git clone https://github.com/org/workflows.git /opt/waterflow/workflows/
   
5. Server 解析 YAML
   - 验证语法和语义
   - 检查 on 字段
   
6. Server 自动创建 Schedule（如果有 on.schedule）
   scheduleClient.Create(ctx, &client.ScheduleOptions{
     ID: "workflow-backup-auto",  // 自动生成 ID
     Spec: &client.ScheduleSpec{
       CronExpressions: []string{"0 2 * * *"},
     },
     Action: &client.ScheduleWorkflowAction{
       Workflow: temporal.RunWorkflowExecutor,
       Args: []interface{}{"Backup Database"},  // 引用 workflow name
       TaskQueue: "default",
     },
   })

阶段 3：运行时执行
──────────────────────────────────────────────────
7. Temporal Schedule 定时触发
   
8. RunWorkflowExecutor 接收参数 "Backup Database"
   
9. 根据 name 从 Git 仓库加载 YAML
   yamlPath := "/opt/waterflow/workflows/backup.yaml"
   yamlContent := os.ReadFile(yamlPath)
   
10. 解析并执行工作流

阶段 4：动态管理（API 增强）
──────────────────────────────────────────────────
11. 用户也可以通过 API 动态创建额外 Schedule
    POST /v1/schedules
    {
      "workflowName": "Backup Database",
      "cron": "0 14 * * *",  // 额外的下午备份
      "timezone": "America/New_York"
    }
    
12. Server 创建第二个 Schedule
    scheduleClient.Create(ctx, &client.ScheduleOptions{
      ID: "backup-afternoon",  // 用户指定或自动生成
      Spec: &client.ScheduleSpec{
        CronExpressions: []string{"0 14 * * *"},
        Timezone: "America/New_York",
      },
      Action: &client.ScheduleWorkflowAction{
        Workflow: temporal.RunWorkflowExecutor,
        Args: []interface{}{"Backup Database"},  // 引用同一个 workflow
        TaskQueue: "default",
      },
    })
```

**存储策略对比：**

| 方案 | YAML 存储 | Schedule 存储 | 优势 | 适用场景 |
|------|-----------|--------------|------|---------|
| **Git 仓库模式（推荐）** | Git 仓库 | Temporal + SQLite 元数据 | 版本控制、Code Review、GitOps | DevOps 团队、企业级 |
| **文件系统模式** | /opt/waterflow/workflows/ | Temporal + SQLite 元数据 | 简单直接、零依赖 | 小团队、快速部署 |
| **数据库模式** | SQLite/PostgreSQL | Temporal + 同一数据库 | 集中管理、多租户隔离 | SaaS 平台、多租户 |

**关键设计原则：**
1. **Schedule 绝对不存储完整 YAML** - 只存储 `workflowName` 引用，避免数据冗余
2. **YAML 是单一数据源（Single Source of Truth）** - 统一从 Git/文件系统/数据库读取
3. **运行时动态加载** - Temporal 触发时，根据 `workflowName` 实时加载 YAML 并解析执行
4. **声明式 + 命令式双模式** - 既支持 YAML `on` 字段自动创建，也支持 API 手动管理

---
4. Schedule 触发
   - Temporal 根据 cron 时间触发
   - 查找 Worker 上注册的 "Backup" Workflow
   - 如果找到 → 创建 Workflow Execution
   - 如果找不到 → 执行失败，记录错误
```
```

**关键优势：**
- ✅ **零存储开销** - Waterflow Server 完全无状态
- ✅ **简化部署** - 只需复制 YAML 文件
- ✅ **热重载** - Worker 监听文件变化（fsnotify）
- ✅ **GitOps 友好** - YAML 文件可版本控制
- ✅ **Temporal 原生** - 完全遵循 Temporal 设计哲学

**与 Temporal UI 的对应关系：**
- Waterflow `workflow_name` = Temporal UI "Workflow Type"
- Waterflow Registry = Worker 注册的 Workflow 函数集合
- 不能在 Schedule 中直接定义 Workflow 代码

**技术价值:**
- 充分利用 Temporal Schedules API（v1.17+）的成熟能力
- 避免自己实现调度器，降低复杂度
- 继承 Temporal 的可靠性和可观测性

## Acceptance Criteria

### AC1: 提交包含 schedule 的 Workflow 时自动创建 Schedule

**Given** 用户有一个包含 `on.schedule` 的 YAML 文件  
**When** 调用 POST /v1/workflows 提交 Workflow  
**Then** 自动创建 Temporal Schedule 并返回 Workflow 详情

**请求示例:**
```bash
POST /v1/workflows
Content-Type: application/json

{
  "yaml": "name: Backup\non:\n  schedule:\n    cron: '0 2 * * *'\n    timezone: 'UTC'\n    overlap_policy: 'skip'\njobs:\n  backup-db:\n    runs-on: default\n    steps:\n      - name: Backup Database\n        uses: shell\n        with:\n          cmd: pg_dump -U postgres mydb > /backups/db.sql"
}
```

**YAML 格式（格式化后）:**
```yaml
name: Backup
on:
  schedule:
    cron: "0 2 * * *"
    timezone: "UTC"           # 可选，默认 UTC
    overlap_policy: "skip"    # 可选，默认 allow_all
jobs:
  backup-db:
    runs-on: default
    steps:
      - name: Backup Database
        uses: shell
        with:
          cmd: pg_dump -U postgres mydb > /backups/db.sql
```

**Server 处理逻辑:**
1. 解析 YAML（使用 Story 1.3 的解析器）
2. 保存到 /opt/waterflow/workflows/{name}.yaml
3. **检查是否存在 `on.schedule` 字段**
4. **如果有 schedule:**
   a. 调用 Temporal ScheduleClient.Create()
   b. Schedule ID = "workflow-{name}"（例如：workflow-backup）
   c. WorkflowType = YAML 中的 name 字段
   d. Spec.CronExpressions = on.schedule.cron
5. 通知 Worker 热重载（或等待定期扫描）
6. 返回 Workflow 详情

**响应示例:**
```json
{
  "workflow_id": "backup",
  "name": "Backup",
  "created_at": "2026-01-27T10:00:00Z",
  "file_path": "/opt/waterflow/workflows/Backup.yaml",
  "schedule": {
    "id": "workflow-backup",
    "cron": "0 2 * * *",
    "status": "active",
    "next_run_time": "2026-01-28T02:00:00Z",
    "timezone": "UTC",
    "overlap_policy": "skip"
  }
}
```

**And** Workflow YAML 保存到文件系统  
**And** Temporal Schedule 创建成功  
**And** 返回 201 Created 状态码

**错误处理:**
- YAML 格式错误 → 400 Bad Request ("invalid YAML")
- Cron 表达式无效 → 400 Bad Request ("invalid cron expression: {details}")
- Workflow 名称重复 → 409 Conflict ("{name}.yaml already exists")
- Schedule ID 冲突 → 409 Conflict ("schedule workflow-{name} already exists")
- Temporal 连接失败 → 503 Service Unavailable

**重要说明:**
- ✅ YAML 中的 `on.schedule` 是可选的
- ✅ 如果没有 `on.schedule`，只创建 Workflow 文件，不创建 Schedule
- ✅ 如果有 `on.schedule`，自动创建 Schedule，用户无需额外操作
- ✅ Schedule ID 使用 "workflow-{name}" 格式，与 Workflow 一一对应

**架构说明:**
- **Schedule 调度数据**完全存储在 Temporal Server（cron、状态、执行历史）
- **Workflow 定义（YAML）**存储在文件系统（/opt/waterflow/workflows/）
- Waterflow Server **完全无状态**，不存储任何数据
- **职责分离**：
  - Temporal：管理调度和执行状态
  - 文件系统：存储 Workflow 定义
  - Worker：加载 YAML 并注册 Workflow

**数据分层：**
```
配置层：Workflow YAML → 文件系统 /opt/waterflow/workflows/*.yaml
注册层：Worker 启动 → 扫描目录 → 注册到 Temporal Worker
执行层：Schedule 状态 + Execution 历史 → Temporal Server
```

### AC2: 获取 Workflow 的 Schedule 信息

**Given** Workflow 已创建并包含 Schedule  
**When** 调用 GET /v1/workflows/:id/schedule  
**Then** 返回该 Workflow 的 Schedule 详情

**请求示例:**
```bash
GET /v1/workflows/backup/schedule
```

**响应示例:**
```json
{
  "schedule_id": "workflow-backup",
  "workflow_name": "Backup",
  "spec": {
    "cron": "0 2 * * *"
  },
  "status": "active",
  "policy": {
    "overlap": "skip",
    "timezone": "UTC",
    "pause_on_failure": false
  },
  "info": {
    "next_run_time": "2026-01-28T02:00:00Z",
    "last_run_time": "2026-01-27T02:00:00Z",
    "last_run_id": "workflow-backup-1735696800",
    "last_run_status": "completed",
    "total_runs": 365,
    "recent_runs": [
      {
        "workflow_id": "workflow-backup-1735696800",
        "start_time": "2026-01-27T02:00:00Z",
        "end_time": "2026-01-27T02:05:32Z",
        "status": "completed"
      }
    ]
  },
  "created_at": "2025-01-27T10:00:00Z"
}
```

**And** 包含 Temporal Schedule 的实时状态  
**And** 包含最近执行记录（最多 10 条）

**错误处理:**
- Workflow 不存在 → 404 Not Found
- Workflow 没有 Schedule → 404 Not Found ("workflow has no schedule")
- Temporal 连接失败 → 503 Service Unavailable

### AC3: 列出所有 Schedules

**Given** 系统中有多个 Workflow 包含 Schedule  
**When** 调用 GET /v1/schedules  
**Then** 返回所有 Schedule 列表（分页）

**请求示例:**
```bash
GET /v1/schedules?status=active&limit=20&offset=0

# 支持过滤参数
GET /v1/schedules?status=paused
```

**响应示例:**
```json
{
  "schedules": [
    {
      "schedule_id": "workflow-backup",
      "workflow_name": "Backup",
      "cron": "0 2 * * *",
      "status": "active",
      "next_run_time": "2026-01-28T02:00:00Z",
      "last_run_time": "2026-01-27T02:00:00Z",
      "total_runs": 365
    },
    {
      "schedule_id": "workflow-weekly-report",
      "workflow_name": "WeeklyReport",
      "cron": "0 9 * * 1",
      "status": "active",
      "next_run_time": "2026-02-03T09:00:00Z",
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

**数据源:**
- 调用 Temporal ScheduleClient.List() API
- 过滤 schedule_id 以 "workflow-" 开头的 Schedule
- 客户端过滤，保证数据实时性
- Schedule 不存在 → 404 Not Found (Temporal 返回 EntityNotExistsError)

### AC4: 暂停和恢复 Schedule

**Given** Schedule 处于 active 状态  
**When** 调用 POST /v1/schedules/:id/pause  
**Then** Schedule 被暂停，不再自动触发

### AC4: 暂停/恢复 Schedule

**Given** Workflow 的 Schedule 处于 active 状态  
**When** 调用 POST /v1/workflows/:id/schedule/pause  
**Then** Schedule 暂停，不再触发新的 Workflow

**暂停请求:**
```bash
POST /v1/workflows/backup/schedule/pause
Content-Type: application/json

{
  "reason": "System maintenance"
}
```

**暂停响应:**
```json
{
  "schedule_id": "workflow-backup",
  "workflow_name": "Backup",
  "status": "paused",
  "paused_at": "2026-01-27T10:00:00Z",
  "reason": "System maintenance"
}
```

**恢复请求:**
```bash
POST /v1/workflows/backup/schedule/resume
Content-Type: application/json

{
  "reason": "Maintenance complete"
}
```

**恢复响应:**
```json
{
  "schedule_id": "workflow-backup",
  "workflow_name": "Backup",
  "status": "active",
  "resumed_at": "2026-01-27T12:00:00Z",
  "next_run_time": "2026-01-28T02:00:00Z"
}
```

**And** 暂停后的 Schedule 不会触发新的 Workflow  
**And** 已运行的 Workflow 不受影响（继续执行）  
**And** 恢复后的 Schedule 计算下次执行时间

**实现方式:**
- 调用 Temporal ScheduleClient.GetHandle("workflow-{id}").Pause()
- 状态变更立即生效，无需本地存储
- Waterflow Server 重启不影响 Schedule 状态

### AC5: 手动触发 Schedule

**Given** Workflow 有 Schedule（无论 active 或 paused）  
**When** 调用 POST /v1/workflows/:id/schedule/trigger  
**Then** 立即启动一个 Workflow Run（不等待 cron 时间）

**请求示例:**
```bash
POST /v1/workflows/backup/schedule/trigger
Content-Type: application/json

{
  "overlap": "allow_all"
}
```

**响应示例:**
```json
{
  "schedule_id": "workflow-backup",
  "workflow_id": "workflow-backup-manual-1706356800",
  "run_id": "abc123...",
  "triggered_at": "2026-01-27T10:00:00Z",
  "status": "running"
}
```

**And** 触发不影响正常的 cron 调度  
**And** 支持 overlap 策略配置（allow_all, skip, cancel_other）  
**And** 返回新启动的 Workflow ID

### AC6: 更新 Schedule 配置

**Given** Workflow 已有 Schedule  
**When** 用户修改 YAML 中的 `on.schedule` 并重新提交  
**Then** Schedule 配置更新

**请求示例:**
```bash
PUT /v1/workflows/backup
Content-Type: application/json

{
  "yaml": "name: Backup\non:\n  schedule:\n    cron: '0 3 * * *'\njobs: ..."
}
```

**Server 处理逻辑:**
1. 解析新的 YAML
2. 检查是否存在 `on.schedule`
3. **如果有 schedule:**
   - 调用 Temporal ScheduleClient.Update() 更新配置
   - 更新 cron、timezone、overlap_policy 等
4. **如果删除了 schedule:**
   - 调用 ScheduleClient.Delete() 删除 Schedule
5. 保存新的 YAML 到文件系统

**响应:**
```json
{
  "workflow_id": "backup",
  "name": "Backup",
  "updated_at": "2026-01-27T10:00:00Z",
  "schedule": {
    "id": "workflow-backup",
    "cron": "0 3 * * *",
    "status": "active",
    "next_run_time": "2026-01-28T03:00:00Z"
  }
}
```

### AC7: 删除 Workflow 时自动删除 Schedule

**Given** Workflow 有 Schedule  
**When** 调用 DELETE /v1/workflows/:id  
**Then** Workflow 和 Schedule 都被删除

**请求示例:**
```bash
DELETE /v1/workflows/backup
```

**响应:**
```
204 No Content
```

**Server 处理逻辑:**
1. 检查 Workflow 是否有 Schedule（查询 Temporal）
2. 如果有 → 调用 ScheduleClient.Delete("workflow-{id}")
3. 删除文件系统中的 YAML 文件
4. 通知 Worker 卸载 Workflow（可选）

**And** Temporal Schedule 被永久删除  
**And** 已运行的 Workflow 不受影响（继续执行）  
**And** 删除后无法恢复（需要重新创建）

**重要说明 - 删除行为:**
- ❌ **不会删除执行历史** - 过往的 Workflow Executions 记录保留
- ❌ **不会终止运行中的实例** - 已启动的 Workflow 继续执行直到完成
- ✅ **删除 Workflow YAML 文件** - /opt/waterflow/workflows/{name}.yaml
- ✅ **删除 Schedule** - 不再按 cron 表达式自动创建新的 Workflow
- ✅ **Worker 卸载 Workflow** - 新的 Workflow 请求会失败

**架构关系:**
```
Workflow 定义 (workflows 表)
    ↑ 引用关系
Schedule (Temporal Schedules)  ← 删除这个
    ↓ 创建
Workflow Executions (运行实例) ← 不受影响
```

**实现方式:**
- 调用 Temporal ScheduleClient.GetHandle(id).Delete()
- Schedule 从 Temporal Server 永久删除
- 无本地数据清理，简化实现
- **不涉及 Workflow 定义和执行历史的任何操作**

### AC8: Overlap 策略支持

**Given** YAML 中配置了 `on.schedule.overlap_policy`  
**When** 上次执行尚未完成，新的调度时间到达  
**Then** 根据策略处理

**支持的策略:**
- **skip**: 跳过新的执行（默认）
- **allow_all**: 允许并发执行
- **cancel_other**: 取消上次执行，启动新的
- **buffer_one**: 缓存一个待执行的任务
- **buffer_all**: 缓存所有待执行的任务

**配置示例:**
```yaml
name: LongRunningJob
on:
  schedule:
    cron: "*/5 * * * *"  # 每 5 分钟
    overlap_policy: skip  # 如果上次还在运行，跳过本次
jobs:
  process:
    runs-on: default
    steps:
      - name: Heavy Processing
        uses: shell
        with:
          cmd: process-data.sh
```

**And** overlap 策略在 YAML 中声明  
**And** 通过修改 YAML 并重新提交可更新策略  
**And** Trigger API 可临时覆盖默认策略
```

**实现逻辑:**
```go
func (m *Manager) ListByWorkflow(ctx context.Context, workflowName string) ([]*Schedule, error) {
    scheduleClient := m.temporalClient.GetClient().ScheduleClient()
    
    // 1. 查询所有 Schedules
    iter, err := scheduleClient.List(ctx, client.ScheduleListOptions{
        PageSize: 1000, // 获取所有数据
    })
    if err != nil {
        return nil, fmt.Errorf("failed to list schedules: %w", err)
    }
    
    // 2. 过滤出引用指定工作流的 Schedules
    var matchedSchedules []*Schedule
    for iter.HasNext() {
        entry, err := iter.Next()
        if err != nil {
            return nil, err
        }
        
        // 从 Schedule Action 提取 workflow name
        if action, ok := entry.Schedule.Action.(*client.ScheduleWorkflowAction); ok {
            if len(action.Args) > 0 {
                if wfName, ok := action.Args[0].(string); ok && wfName == workflowName {
                    schedule := convertFromListEntry(entry)
                    matchedSchedules = append(matchedSchedules, schedule)
                }
            }
        }
    }
    
    m.logger.Info("Listed schedules by workflow",
        zap.String("workflow_name", workflowName),
        zap.Int("count", len(matchedSchedules)),
    )
    
    return matchedSchedules, nil
}
```

**返回数据示例:**
```go
[
    {
        ID:          "nightly-backup",
        WorkflowName: "Backup",
        Cron:        "0 2 * * *",
        Status:      "active",
        NextRunTime: time.Parse(...),
    },
    {
        ID:          "weekly-backup",
        WorkflowName: "Backup",
        Cron:        "0 3 * * 0",
        Status:      "paused",
        NextRunTime: time.Time{}, // zero value for paused
    },
]
```

**性能要求:**
- 查询时间: < 500ms (1000 schedules)
- 客户端过滤无需额外 RPC 调用
- 支持缓存优化（可选，TTL 60s）

**错误处理:**
- Temporal 连接失败 → 返回错误
- 工作流不存在 → 返回空列表（不是错误）

**And** 此方法主要供内部使用，不直接暴露为 REST API  
**And** Story 1.9 的删除工作流 API 依赖此方法  
**And** 返回结果按 Schedule ID 排序

**与 Story 1.9 的集成:**
```go
// Story 1.9 中使用此方法
func (h *WorkflowHandler) DeleteWorkflowDefinition(c *gin.Context) {
    workflowName := c.Param("name")
    
    // 检查 Schedule 引用
    schedules, err := h.scheduleManager.ListByWorkflow(c.Request.Context(), workflowName)
    if err != nil {
        // 处理错误
    }
    
    if len(schedules) > 0 {
        // 返回 409 Conflict，列出所有引用的 Schedules
        c.JSON(409, gin.H{...})
        return
    }
    
    // 无引用，允许删除
    h.workflowStore.Delete(c.Request.Context(), workflowName)
    c.Status(204)
}
```

## Tasks / Subtasks

### Task 0: Worker Workflow 加载机制（非本 Story，但需说明）
**注：此功能应在 Worker 相关 Story 中实现，这里仅说明依赖**

- [ ] Worker 启动时扫描 /opt/waterflow/workflows/ 目录
- [ ] 解析所有 *.yaml 文件
- [ ] 注册 Workflow 到 Temporal Worker
- [ ] 可选：监听文件变化（fsnotify）实现热重载

**示例实现：**
```go
// cmd/agent/main.go
func main() {
    // ... Temporal Worker 初始化
    
    // 扫描并加载 Workflow
    workflowDir := "/opt/waterflow/workflows"
    files, _ := filepath.Glob(filepath.Join(workflowDir, "*.yaml"))
    
    for _, file := range files {
        yamlContent, _ := os.ReadFile(file)
        workflowDef := parseYAML(yamlContent)
        
        // 注册 Workflow
        worker.RegisterWorkflow(workflowDef.Name, func(ctx workflow.Context) error {
            // 执行 Workflow 逻辑
            return executeWorkflow(ctx, workflowDef)
        })
        
        log.Printf("Registered workflow: %s", workflowDef.Name)
    }
    
    // 启动 Worker
    worker.Run(worker.InterruptCh())
}
```

### Task 1: Workflow Handler 中的 Schedule 自动创建逻辑

**目标:** 在 POST /v1/workflows 时，如果 YAML 包含 `on.schedule`，自动创建 Temporal Schedule

- [ ] 修改 internal/api/workflow_handler.go
  - [ ] CreateWorkflow() 方法中增加 Schedule 创建逻辑
  - [ ] UpdateWorkflow() 方法中增加 Schedule 更新/删除逻辑
  - [ ] DeleteWorkflow() 方法中增加 Schedule 删除逻辑

**实现示例:**
```go
// internal/api/workflow_handler.go
func (h *WorkflowHandler) CreateWorkflow(c *gin.Context) {
    var req CreateWorkflowRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "invalid request"})
        return
    }
    
    // 1. 解析 YAML
    workflowDef, err := dsl.Parse(req.YAML)
    if err != nil {
        c.JSON(400, gin.H{"error": fmt.Sprintf("invalid YAML: %v", err)})
        return
    }
    
    // 2. 保存 YAML 到文件系统
    filePath := fmt.Sprintf("/opt/waterflow/workflows/%s.yaml", workflowDef.Name)
    if err := os.WriteFile(filePath, []byte(req.YAML), 0644); err != nil {
        c.JSON(500, gin.H{"error": "failed to save workflow"})
        return
    }
    
    // 3. 如果有 on.schedule → 创建 Schedule
    var scheduleInfo *ScheduleInfo
    if workflowDef.On != nil && workflowDef.On.Schedule != nil {
        scheduleClient := h.temporalClient.GetClient().ScheduleClient()
        
        scheduleID := fmt.Sprintf("workflow-%s", workflowDef.Name)
        schedule, err := scheduleClient.Create(c.Request.Context(), client.ScheduleOptions{
            ID: scheduleID,
            Spec: client.ScheduleSpec{
                CronExpressions: []string{workflowDef.On.Schedule.Cron},
                Timezone:        workflowDef.On.Schedule.Timezone, // 默认 "UTC"
            },
            Action: &client.ScheduleWorkflowAction{
                WorkflowType: workflowDef.Name,
                TaskQueue:    "default", // 从 jobs[0].runs_on 获取
            },
            Overlap: parseOverlapPolicy(workflowDef.On.Schedule.OverlapPolicy),
            Paused:  false,
        })
        
        if err != nil {
            // Schedule 创建失败，清理 YAML 文件
            os.Remove(filePath)
            c.JSON(500, gin.H{"error": fmt.Sprintf("failed to create schedule: %v", err)})
            return
        }
        
        scheduleInfo = &ScheduleInfo{
            ID:          scheduleID,
            Cron:        workflowDef.On.Schedule.Cron,
            Status:      "active",
            NextRunTime: schedule.GetNextRunTime(),
        }
    }
    
    // 4. 通知 Worker 热重载（可选）
    // h.notifyWorker(workflowDef.Name)
    
    // 5. 返回响应
    c.JSON(201, gin.H{
        "workflow_id": workflowDef.Name,
        "name":        workflowDef.Name,
        "file_path":   filePath,
        "schedule":    scheduleInfo, // 可能为 nil
        "created_at":  time.Now().UTC(),
    })
}
```

### Task 2: Schedule 查询和管理 API

**目标:** 实现 Schedule 的查询、暂停/恢复、触发等管理功能

- [ ] 创建 internal/api/schedule_handler.go
  - [ ] GET /v1/workflows/:id/schedule - 查询 Workflow 的 Schedule
  - [ ] GET /v1/schedules - 列出所有 Schedules
  - [ ] POST /v1/workflows/:id/schedule/pause - 暂停
  - [ ] POST /v1/workflows/:id/schedule/resume - 恢复
  - [ ] POST /v1/workflows/:id/schedule/trigger - 手动触发

**实现示例:**
```go
// internal/api/schedule_handler.go
package api

import (
    "fmt"
    "github.com/gin-gonic/gin"
    "go.temporal.io/sdk/client"
)

type ScheduleHandler struct {
    temporalClient *client.Client
    logger         *zap.Logger
}

// GetWorkflowSchedule - GET /v1/workflows/:id/schedule
func (h *ScheduleHandler) GetWorkflowSchedule(c *gin.Context) {
    workflowID := c.Param("id")
    scheduleID := fmt.Sprintf("workflow-%s", workflowID)
    
    scheduleClient := h.temporalClient.ScheduleClient()
    handle := scheduleClient.GetHandle(c.Request.Context(), scheduleID)
    
    desc, err := handle.Describe(c.Request.Context())
    if err != nil {
        if isScheduleNotFound(err) {
            c.JSON(404, gin.H{"error": "workflow has no schedule"})
            return
        }
        c.JSON(500, gin.H{"error": fmt.Sprintf("failed to get schedule: %v", err)})
        return
    }
    
    c.JSON(200, convertScheduleDescToResponse(desc))
}

// ListSchedules - GET /v1/schedules
func (h *ScheduleHandler) ListSchedules(c *gin.Context) {
    scheduleClient := h.temporalClient.ScheduleClient()
    
    iter := scheduleClient.List(c.Request.Context(), client.ScheduleListOptions{
        PageSize: 100,
    })
    
    var schedules []gin.H
    for iter.HasNext() {
        entry, err := iter.Next()
        if err != nil {
            c.JSON(500, gin.H{"error": "failed to list schedules"})
            return
        }
        
        // 只返回 Waterflow 创建的 Schedule (ID 以 "workflow-" 开头)
        if strings.HasPrefix(entry.ID, "workflow-") {
            schedules = append(schedules, gin.H{
                "schedule_id":    entry.ID,
                "workflow_name":  strings.TrimPrefix(entry.ID, "workflow-"),
                "cron":           entry.Spec.CronExpressions[0],
                "status":         entry.Info.Paused ? "paused" : "active",
                "next_run_time":  entry.Info.NextActionTime,
            })
        }
    }
    
    c.JSON(200, gin.H{
        "schedules": schedules,
        "total":     len(schedules),
    })
}

// PauseSchedule - POST /v1/workflows/:id/schedule/pause
func (h *ScheduleHandler) PauseSchedule(c *gin.Context) {
    workflowID := c.Param("id")
    scheduleID := fmt.Sprintf("workflow-%s", workflowID)
    
    var req PauseRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        req.Reason = "" // 可选参数
    }
    
    handle := h.temporalClient.ScheduleClient().GetHandle(c.Request.Context(), scheduleID)
    if err := handle.Pause(c.Request.Context(), client.SchedulePauseOptions{
        Note: req.Reason,
    }); err != nil {
        c.JSON(500, gin.H{"error": fmt.Sprintf("failed to pause: %v", err)})
        return
    }
    
    c.JSON(200, gin.H{
        "schedule_id": scheduleID,
        "workflow_name": workflowID,
        "status": "paused",
        "paused_at": time.Now().UTC(),
        "reason": req.Reason,
    })
}

// TriggerSchedule - POST /v1/workflows/:id/schedule/trigger
func (h *ScheduleHandler) TriggerSchedule(c *gin.Context) {
    workflowID := c.Param("id")
    scheduleID := fmt.Sprintf("workflow-%s", workflowID)
    
    var req TriggerRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        req.Overlap = "allow_all" // 默认值
    }
    
    handle := h.temporalClient.ScheduleClient().GetHandle(c.Request.Context(), scheduleID)
    if err := handle.Trigger(c.Request.Context(), client.ScheduleTriggerOptions{
        Overlap: parseOverlapPolicy(req.Overlap),
    }); err != nil {
        c.JSON(500, gin.H{"error": fmt.Sprintf("failed to trigger: %v", err)})
        return
    }
    
    c.JSON(200, gin.H{
        "schedule_id": scheduleID,
        "workflow_id": fmt.Sprintf("%s-manual-%d", scheduleID, time.Now().Unix()),
        "triggered_at": time.Now().UTC(),
        "status": "running",
    })
}
```

### Task 3: 数据结构和类型定义

- [ ] 创建 pkg/dsl/types.go (如不存在)
  - [ ] 定义 `On` 结构体，包含 `Schedule *ScheduleConfig`
  - [ ] 定义 `ScheduleConfig` 结构体（cron, timezone, overlap_policy）
  
**示例:**
```go
// pkg/dsl/types.go
type WorkflowDefinition struct {
    Name        string  `yaml:"name"`
    On          *On     `yaml:"on"`
    Jobs        map[string]*Job `yaml:"jobs"`
}

type On struct {
    Schedule *ScheduleConfig `yaml:"schedule"`
    Webhook  *WebhookConfig  `yaml:"webhook"`
    Manual   bool            `yaml:"manual"`
}

type ScheduleConfig struct {
    Cron          string `yaml:"cron"`
    Timezone      string `yaml:"timezone"`       // 可选，默认 UTC
    OverlapPolicy string `yaml:"overlap_policy"` // 可选，默认 skip
}
```

### Task 4: 路由注册

- [ ] 修改 internal/api/router.go
  - [ ] 注册 POST /v1/workflows (修改现有)
  - [ ] 注册 PUT /v1/workflows/:id (修改现有)
  - [ ] 注册 DELETE /v1/workflows/:id (修改现有)
  - [ ] 注册 GET /v1/workflows/:id/schedule
  - [ ] 注册 GET /v1/schedules
  - [ ] 注册 POST /v1/workflows/:id/schedule/pause
  - [ ] 注册 POST /v1/workflows/:id/schedule/resume
  - [ ] 注册 POST /v1/workflows/:id/schedule/trigger

**示例:**
```go
// internal/api/router.go
func SetupRouter(temporal *client.Client) *gin.Engine {
    r := gin.Default()
    
    workflowHandler := &WorkflowHandler{temporalClient: temporal}
    scheduleHandler := &ScheduleHandler{temporalClient: temporal}
    
    v1 := r.Group("/v1")
    {
        // Workflow APIs (修改现有)
        v1.POST("/workflows", workflowHandler.CreateWorkflow)
        v1.PUT("/workflows/:id", workflowHandler.UpdateWorkflow)
        v1.DELETE("/workflows/:id", workflowHandler.DeleteWorkflow)
        
        // Schedule APIs (新增)
        v1.GET("/workflows/:id/schedule", scheduleHandler.GetWorkflowSchedule)
        v1.POST("/workflows/:id/schedule/pause", scheduleHandler.PauseSchedule)
        v1.POST("/workflows/:id/schedule/resume", scheduleHandler.ResumeSchedule)
        v1.POST("/workflows/:id/schedule/trigger", scheduleHandler.TriggerSchedule)
        v1.GET("/schedules", scheduleHandler.ListSchedules)
    }
    
    return r
}
```
    handle, err := scheduleClient.Create(ctx, client.ScheduleOptions{
        ID: req.Name,
        
        Spec: client.ScheduleSpec{
            CronExpressions: []string{req.Cron},
            Timezone:        req.Timezone,
        },
        
        Action: &client.ScheduleWorkflowAction{
            ID:        fmt.Sprintf("%s-{{.ScheduledTime.Unix}}", req.Name),
            Workflow:  temporal.RunWorkflowExecutor,
            Args:      []interface{}{req.WorkflowName}, // 传入工作流名称
            TaskQueue: "default",
            
            WorkflowExecutionTimeout: 24 * time.Hour,
        },
        
        Policy: &client.SchedulePolicy{
            Overlap:       parseOverlapPolicy(req.OverlapPolicy), // 转换为 enums.ScheduleOverlapPolicy
            CatchupWindow: 1 * time.Hour,
        },
        
        Paused: req.Paused,
    })
    
    if err != nil {
        return nil, fmt.Errorf("failed to create Temporal schedule: %w", err)
    }
    
    // 3. 查询实时状态（从 Temporal 获取权威数据）
    desc, err := handle.Describe(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to describe schedule: %w", err)
    }
    
    // 4. 转换为响应格式
    schedule := convertFromDescription(desc, req.WorkflowName)
    
    m.logger.Info("Schedule created successfully",
        zap.String("schedule_id", req.Name),
        zap.String("workflow_name", req.WorkflowName),
        zap.String("cron", req.Cron),
    )
    
    return schedule, nil
}

// List 列出所有 Schedules（直接查询 Temporal）
func (m *Manager) List(ctx context.Context, filter *Filter) ([]*Schedule, int, error) {
    scheduleClient := m.temporalClient.GetClient().ScheduleClient()
    
    // 查询 Temporal Schedule 列表
    iter, err := scheduleClient.List(ctx, client.ScheduleListOptions{
        PageSize: 1000, // 客户端过滤需要获取更多数据
    })
    if err != nil {
        return nil, 0, fmt.Errorf("failed to list schedules: %w", err)
    }
    
    var allSchedules []*Schedule
    for iter.HasNext() {
        entry, err := iter.Next()
        if err != nil {
            return nil, 0, err
        }
        
        schedule := convertFromListEntry(entry)
        
        // 客户端过滤
        if filter.Status != "" && schedule.Status != filter.Status {
            continue
        }
        if filter.WorkflowName != "" && schedule.WorkflowName != filter.WorkflowName {
            continue
        }
        
        allSchedules = append(allSchedules, schedule)
    }
    
    // 客户端分页
    total := len(allSchedules)
    start := filter.Offset
    end := start + filter.Limit
    if end > total {
        end = total
    }
    
    return allSchedules[start:end], total, nil
}

// Get 查询 Schedule 详情（直接从 Temporal 获取）
func (m *Manager) Get(ctx context.Context, id string) (*Schedule, error) {
    scheduleClient := m.temporalClient.GetClient().ScheduleClient()
    handle := scheduleClient.GetHandle(ctx, id)
    
    desc, err := handle.Describe(ctx)
    if err != nil {
        return nil, fmt.Errorf("schedule not found: %w", err)
    }
    
    return convertFromDescription(desc, "" /* workflow name from desc */), nil
}

// Pause 暂停 Schedule（直接调用 Temporal）
func (m *Manager) Pause(ctx context.Context, id string, reason string) error {
    scheduleClient := m.temporalClient.GetClient().ScheduleClient()
    handle := scheduleClient.GetHandle(ctx, id)
    
    err := handle.Pause(ctx, client.SchedulePauseOptions{
        Note: reason,
    })
    
    if err != nil {
        return fmt.Errorf("failed to pause schedule: %w", err)
    }
    
    m.logger.Info("Schedule paused",
        zap.String("schedule_id", id),
        zap.String("reason", reason),
    )
    
    return nil
}

// Resume 恢复 Schedule（直接调用 Temporal）
func (m *Manager) Resume(ctx context.Context, id string, reason string) error {
    scheduleClient := m.temporalClient.GetClient().ScheduleClient()
    handle := scheduleClient.GetHandle(ctx, id)
    
    err := handle.Unpause(ctx, client.ScheduleUnpauseOptions{
        Note: reason,
    })
    
    if err != nil {
        return fmt.Errorf("failed to resume schedule: %w", err)
    }
    
    m.logger.Info("Schedule resumed",
        zap.String("schedule_id", id),
        zap.String("reason", reason),
    )
    
    return nil
}

// Update 更新 Schedule 配置（使用 Temporal Update API）
func (m *Manager) Update(ctx context.Context, id string, req *UpdateScheduleRequest) (*Schedule, error) {
    scheduleClient := m.temporalClient.GetClient().ScheduleClient()
    handle := scheduleClient.GetHandle(ctx, id)
    
    // 使用 Update 方法和 DoUpdate 回调
    err := handle.Update(ctx, client.ScheduleUpdateOptions{
        DoUpdate: func(input client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
            // 修改 Schedule 配置
            schedule := input.Description.Schedule
            
            if req.Cron != "" {
                schedule.Spec.CronExpressions = []string{req.Cron}
            }
            if req.Timezone != "" {
                schedule.Spec.Timezone = req.Timezone
            }
            if req.OverlapPolicy != "" {
                schedule.Policy.Overlap = parseOverlapPolicy(req.OverlapPolicy)
            }
            
            return &client.ScheduleUpdate{
                Schedule: &schedule,
            }, nil
        },
    })
    
    if err != nil {
        return nil, fmt.Errorf("failed to update schedule: %w", err)
    }
    
    // 获取更新后的详情
    desc, err := handle.Describe(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to describe updated schedule: %w", err)
    }
    
    m.logger.Info("Schedule updated",
        zap.String("schedule_id", id),
    )
    
    return convertFromDescription(desc, "" /* extract from desc */), nil
}

// Trigger 手动触发（直接调用 Temporal）
func (m *Manager) Trigger(ctx context.Context, id string, overlap enums.ScheduleOverlapPolicy) (string, error) {
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

// Delete 删除 Schedule（直接调用 Temporal）
func (m *Manager) Delete(ctx context.Context, id string) error {
    scheduleClient := m.temporalClient.GetClient().ScheduleClient()
    handle := scheduleClient.GetHandle(ctx, id)
    
    err := handle.Delete(ctx)
    if err != nil {
        return fmt.Errorf("failed to delete schedule: %w", err)
    }
    
    m.logger.Info("Schedule deleted",
        zap.String("schedule_id", id),
    )
    
    return nil
}

// convertFromDescription 将 Temporal ScheduleDescription 转换为 REST 响应
func convertFromDescription(desc *client.ScheduleDescription, workflowName string) *Schedule {
    schedule := &Schedule{
        ID:           desc.ID,
        WorkflowName: workflowName, // 从 Action Args 提取
        Cron:         desc.Schedule.Spec.CronExpressions[0],
        Timezone:     desc.Schedule.Spec.Timezone,
        Status:       getScheduleStatus(desc.Schedule.State),
        CreatedAt:    desc.Info.CreateTime,
    }
    
    if desc.Schedule.State != nil {
        if desc.Schedule.State.NextActionTime != nil {
            schedule.NextRunTime = *desc.Schedule.State.NextActionTime
        }
    }
    
    return schedule
}
```

### Task 5: 单元测试和集成测试

**单元测试:**
- [ ] internal/api/workflow_handler_test.go - 测试 Workflow 创建时的 Schedule 自动创建
- [ ] internal/api/schedule_handler_test.go - 测试 Schedule 管理 API
- [ ] pkg/dsl/parser_test.go - 测试 YAML 中 `on.schedule` 字段解析

**集成测试:**
- [ ] test/integration/workflow_schedule_test.go
  - [ ] 测试提交包含 schedule 的 YAML → Schedule 自动创建
  - [ ] 测试提交不含 schedule 的 YAML → 只创建 Workflow
  - [ ] 测试更新 YAML 修改 cron → Schedule 更新
  - [ ] 测试删除 YAML → Schedule 自动删除
  - [ ] 测试暂停/恢复/触发功能
  - [ ] 验证与真实 Temporal Server 的集成

**测试环境:**
- 需要运行 Temporal Server（Docker Compose）
- 验证 Schedule 实际创建和执行
- 测试 Waterflow Server 重启后仍能查询 Schedule

### Task 6: 文档更新
- [ ] 更新 API 文档（api/openapi.yaml）- POST /v1/workflows 接口说明
- [ ] 更新用户文档（docs/quick-start.md）- 添加定时任务示例
- [ ] 更新 YAML DSL 参考（docs/yaml-dsl-reference.md）- `on.schedule` 字段说明
- [ ] 添加 Schedule API 示例

## Technical Requirements

### Technology Stack
- **Temporal SDK:** go.temporal.io/sdk v1.38.0 (Schedules API 支持)
- **存储:** Temporal Server (唯一数据源，分布式持久化)
- **HTTP Router:** gorilla/mux v1.8+ (已有)
- **日志:** uber-go/zap v1.26+ (已有)
- **Cron 解析:** github.com/robfig/cron v3 (验证 cron 表达式)

### Architecture Constraints

**ADR 遵循:**
- [ADR-0001: 使用 Temporal 作为工作流引擎](../adr/0001-use-temporal-workflow-engine.md)
- [ADR-0004: YAML DSL 语法设计](../adr/0004-yaml-dsl-syntax.md)

**设计原则:**
- **Temporal 是唯一数据源** - 不引入额外数据库，避免数据一致性问题
- **Waterflow Server 完全无状态** - 所有数据存储在 Temporal，重启不影响功能
- **充分利用 Temporal Schedules API** - 不重复造轮子，直接使用成熟方案
- **客户端过滤和分页** - List API 在 Waterflow 层实现过滤逻辑
- **实时数据** - 每次查询都从 Temporal 获取最新状态，无缓存延迟

### Performance Requirements

**创建/删除性能:**
- 创建 Schedule: < 200ms (P95, 包含 Temporal RPC 调用)
- 删除 Schedule: < 100ms (P95, 包含 Temporal RPC 调用)

**查询性能:**
- 列表查询: < 500ms (1000 schedules, 包含客户端过滤)
- 详情查询: < 100ms (单次 Temporal Describe 调用)

**并发性能:**
- 支持 1000+ 并发 Schedules（Temporal 分布式存储）
- 支持 100+ 并发 API 请求

**性能说明:**
- 查询性能取决于 Temporal Server 响应时间
- 无本地缓存，数据实时性优先于性能
- 如未来性能成为瓶颈，可在 Waterflow 层添加内存缓存（TTL 30s）

### Security Requirements

- **身份验证:** 集成现有的认证机制（如有）
- **授权:** Schedule 操作需要管理员权限
- **输入验证:** 验证 cron 表达式格式
- **SQL 注入防护:** 使用参数化查询

## Definition of Done

- [ ] 所有 Acceptance Criteria 验收通过
- [ ] 所有 Tasks 完成并测试通过
- [ ] 单元测试覆盖率 ≥85%（Mock Temporal ScheduleClient 测试）
- [ ] 集成测试通过（与真实 Temporal Server 集成）
- [ ] 代码通过 golangci-lint 检查
- [ ] **验证架构原则 - 一体化设计**:
  - [ ] POST /v1/workflows 时，解析 YAML 中的 `on.schedule` 字段
  - [ ] 如果有 `on.schedule`，自动调用 Temporal ScheduleClient 创建 Schedule
  - [ ] Schedule ID 格式为 "workflow-{name}"
  - [ ] Schedule 数据完全存储在 Temporal Server（无本地状态）
  - [ ] Workflow YAML 文件存储在 /opt/waterflow/workflows/
  - [ ] Worker 启动时扫描目录并注册所有 Workflow
  - [ ] Waterflow Server 重启后仍能正常查询和管理 Schedule
- [ ] **验证 API 设计**:
  - [ ] GET /v1/workflows/:id/schedule 返回 Workflow 的 Schedule 信息
  - [ ] GET /v1/schedules 列出所有 Schedule（过滤 "workflow-" 前缀）
  - [ ] POST /v1/workflows/:id/schedule/pause/resume/trigger 正常工作
  - [ ] PUT /v1/workflows/:id 更新 YAML 时自动更新 Schedule
  - [ ] DELETE /v1/workflows/:id 删除 Workflow 时自动删除 Schedule
- [ ] API 文档更新（api/openapi.yaml）
- [ ] YAML DSL 文档更新（docs/yaml-dsl-reference.md）- `on.schedule` 说明
- [ ] 用户文档更新（docs/quick-start.md）- 定时任务示例
- [ ] Code Review 通过
- [ ] 生产环境验证（至少运行 24 小时，包括 Server 重启测试）

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
