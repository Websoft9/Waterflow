# ADR-0009: 工作流定义与执行分离架构

**状态:** ✅ 已采纳  
**日期:** 2026-02-03  
**决策者:** 架构团队  
**相关 Story:** Story 1.9 (API 重构), Story 1.10 (Schedule API), Story 1.11 (Webhook Trigger), Story 6.4 (模板 API)

## 背景

在实现 Story 1-10 (Schedule API) 和 Story 1-11 (Webhook) 过程中，发现当前架构存在以下问题：

### 问题 1: YAML 无持久存储

当前 `POST /v1/workflows` API 将 YAML 仅存储在 Temporal Memo 中：

```go
// internal/api/workflow_handler.go
startOptions := client.StartWorkflowOptions{
    Memo: map[string]interface{}{
        "workflow_yaml": string(yamlBytes),  // 仅存在 Memo
    },
}
```

**问题：**
- Memo 与 Execution 生命周期绑定，Execution 结束（或超过保留期）后丢失
- Schedule/Webhook 无法引用已有工作流定义
- 无法查询"有哪些工作流定义"

### 问题 2: API 语义混淆

当前 API 既是"定义"又是"执行"：

```
POST /v1/workflows  # 提交 YAML 并立即执行
                    # 没有"仅保存定义"的能力
```

### 问题 3: 模板与定义概念分离

Epic 6 的工作流模板 (`/v1/templates`) 和用户工作流定义 (`/v1/workflows`) 已经是两套独立的 API：

```
/v1/templates   # 系统预置模板，文件系统存储（只读）
/v1/workflows   # 用户工作流，需要增加持久存储
```

**经验证：这是正确的架构设计**
- 模板 = 可复用的蓝图（参考资料，只读）
- 工作流定义 = 可执行的实例（持久化配置，可读写）
- 两者职责清晰，无需合并

### 问题 4: 参数化不完整

当前 `${{ vars.xxx }}` 仅支持 Step 层级，不支持结构化字段：

```yaml
# ❌ 当前不支持
runs-on: ${{ vars.agent }}
timeout: ${{ vars.timeout }}
matrix:
  servers: ${{ vars.servers }}
```

---

## 决策

### 决策 1: Definition 与 Execution 分离

采用两层架构：

```
┌─────────────────────────────────────────────────────────────────┐
│                    两层工作流体系                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ Layer 1: Definition (工作流定义)                           │ │
│  │ ─────────────────────────────────────────────────────────  │ │
│  │ • 持久存储的工作流 YAML                                     │ │
│  │ • 可参数化 (${{ vars.xxx }})                               │ │
│  │ • 可被多次执行、被触发器引用                                 │ │
│  └───────────────────────┬───────────────────────────────────┘ │
│                          │                                      │
│                          ▼                                      │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ Layer 2: Execution (工作流执行)                            │ │
│  │ ─────────────────────────────────────────────────────────  │ │
│  │ • 一次具体的运行实例                                        │ │
│  │ • 可通过 API / Schedule / Webhook 触发                      │ │
│  │ • 执行时传入 vars 参数覆盖                                  │ │
│  └───────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### 决策 2: 混合存储架构

**模板与工作流定义是两个独立的概念：**

```
┌─────────────────────────────────────────────────────┐
│  模板 (Template) - 可复用的蓝图                      │
│  ──────────────────────────────────────────────────  │
│  API: /v1/templates (独立 API)                      │
│  存储: 文件系统 (examples/workflows/)                │
│  访问: 只读                                          │
│  用途: 用户参考、复制、学习最佳实践                    │
│  实现: Story 6.4 已完成 (TemplateHandler)           │
└─────────────────────────────────────────────────────┘
                        ↓ 用户复制内容
┌─────────────────────────────────────────────────────┐
│  工作流定义 (Workflow Definition) - 可执行的实例     │
│  ──────────────────────────────────────────────────  │
│  API: /v1/workflows (独立 API)                      │
│  存储: 数据库 (workflow_definitions 表)              │
│  访问: CRUD                                          │
│  用途: 持久化配置，可被 Schedule/Webhook 执行         │
│  实现: 使用 DatabaseDefinitionStore                  │
└─────────────────────────────────────────────────────┘
```

**优势：**
- 模板和定义是两个独立的一等公民，职责清晰
- 模板用文件系统（简单、版本控制友好、随代码发布）
- 定义用数据库（支持动态 CRUD、查询、关联）
- 两个 API 独立演进，互不干扰

### 决策 3: 工作流定义存储实现

**工作流定义使用数据库存储（GORM + PostgreSQL）：**

```
┌─────────────────────────────────────────────────────────────────┐
│              工作流定义存储架构                                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │   数据库 (Temporal DB / PostgreSQL)                      │   │
│  ├─────────────────────────────────────────────────────────┤   │
│  │ workflow_definitions 表                                  │   │
│  │                                                          │   │
│  │ • 用户动态创建工作流定义                                  │   │
│  │ • 支持完整 CRUD 操作                                      │   │
│  │ • 并发安全（GORM 事务支持）                               │   │
│  │ • 可扩展元数据字段                                        │   │
│  │ • 支持查询、过滤、分页                                    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │        DatabaseDefinitionStore (GORM 实现)               │   │
│  │  ┌──────────────────────────────────────────────────┐      │   │
│  │  │ - Create(def) error                            │      │   │
│  │  │ - Get(name) (*WorkflowDefinition, error)       │      │   │
│  │  │ - List(filter) ([]*WorkflowDefinition, error)  │      │   │
│  │  │ - Update(def) error                            │      │   │
│  │  │ - Delete(name) error                           │      │   │
│  │  │ - Exists(name) (bool, error)                   │      │   │
│  │  └──────────────────────────────────────────────────┘      │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘

注：模板 API (/v1/templates) 由 Story 6.4 已实现的 TemplateHandler 处理，
使用文件系统存储 (examples/workflows/)，与工作流定义存储完全独立。
```

**选择理由：**

| 特性 | 数据库存储 (GORM) |
|------|------------------|
| **并发安全** | ✅ GORM 事务支持 |
| **CRUD 支持** | ✅ 完整支持 |
| **查询能力** | ✅ 支持复杂过滤、分页 |
| **扩展性** | ✅ 易于添加新字段 |
| **运维依赖** | 复用 Temporal 数据库 |

### 决策 4: DSL 参数化增强

扩展 `${{ vars.xxx }}` 支持范围：

```yaml
name: deploy-workflow
runs-on: ${{ vars.agent }}           # ✅ 支持动态 Agent
timeout: ${{ vars.timeout }}         # ✅ 支持动态超时

vars:
  agent: default-agent               # 默认值
  timeout: 30m
  env: production
  servers:
    - web1
    - web2

jobs:
  deploy:
    runs-on: ${{ vars.agent }}
    timeout: ${{ vars.timeout }}
    strategy:
      matrix:
        server: ${{ vars.servers }}  # ✅ 支持动态 Matrix
    steps:
      - name: Deploy to ${{ matrix.server }}
        uses: exec@v1
        with:
          command: deploy --env=${{ vars.env }}
```

### 决策 5: 三层参数覆盖机制

```
优先级 (低 → 高):
┌─────────────────────────────────────────────────────────────┐
│ 1. YAML 默认值         vars:                                │
│                          env: dev                           │
│                          timeout: 30m                       │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│ 2. 触发器绑定参数       Schedule/Webhook 创建时指定的 vars   │
│                          { "env": "staging" }               │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. 执行时覆盖           API 调用或 Webhook payload 传入      │
│                          { "env": "prod", "debug": true }   │
└─────────────────────────────────────────────────────────────┘

最终合并: { env: "prod", timeout: "30m", debug: true }
```

---

## API 设计

### 工作流定义 API

```
# 创建定义
POST /v1/workflows
{
    "name": "my-deploy",
    "description": "My deployment workflow",
    "content": "<YAML 内容>"
}
→ 201 Created

# 列表（支持过滤）
GET /v1/workflows
    ?category=deployment          # 按类别过滤
    ?limit=20&offset=0            # 分页
→ 200 OK
{
    "workflows": [...],
    "count": 10
}

# 获取详情
GET /v1/workflows/{name}
→ 200 OK
{
    "name": "my-deploy",
    "content": "<YAML>",
    "description": "My deployment workflow",
    ...
}

# 更新定义
PUT /v1/workflows/{name}
{
    "content": "<新 YAML 内容>"
}
→ 200 OK

# 删除定义
DELETE /v1/workflows/{name}
→ 204 No Content
```

### 工作流执行 API

```
# 执行定义（引用已有定义）
POST /v1/workflows/{name}/run
{
    "vars": {                     # 可选，参数覆盖
        "env": "prod",
        "version": "1.2.3"
    }
}
→ 202 Accepted
{
    "execution_id": "my-deploy-abc123",
    "status": "running"
}

# 直接执行 (一次性，不存储定义)
POST /v1/executions
{
    "workflow": "<YAML 内容>",
    "vars": { ... }
}
→ 202 Accepted

# 查询执行列表
GET /v1/executions
    ?workflow_name=my-deploy      # 按工作流过滤
    ?status=running               # 按状态过滤
→ 200 OK

# 获取执行详情
GET /v1/executions/{id}
→ 200 OK

# 取消执行
POST /v1/executions/{id}/cancel
→ 200 OK

# 重试执行
POST /v1/executions/{id}/retry
→ 202 Accepted
```

### 触发器 API

```
# Schedule 管理
POST   /v1/workflows/{name}/schedules
{
    "schedule_id": "nightly-deploy",
    "cron": "0 2 * * *",
    "timezone": "Asia/Shanghai",
    "vars": { "env": "prod" },
    "enabled": true
}
→ 201 Created

GET    /v1/workflows/{name}/schedules
GET    /v1/workflows/{name}/schedules/{schedule_id}
PUT    /v1/workflows/{name}/schedules/{schedule_id}
DELETE /v1/workflows/{name}/schedules/{schedule_id}
POST   /v1/workflows/{name}/schedules/{schedule_id}/pause
POST   /v1/workflows/{name}/schedules/{schedule_id}/resume

# Webhook 管理
POST   /v1/workflows/{name}/webhooks
{
    "webhook_id": "github-push",
    "vars": { "branch": "main" },
    "secret": "xxx",
    "enabled": true
}
→ 201 Created

GET    /v1/workflows/{name}/webhooks
DELETE /v1/workflows/{name}/webhooks/{webhook_id}

# Webhook 触发端点
POST /v1/webhooks/{webhook_id}/trigger
{
    "vars": { "commit": "abc123" }
}
→ 202 Accepted
```

**注：** 模板 API (`/v1/templates`) 是独立的 API，由 Story 6.4 实现的 TemplateHandler 处理，
与工作流定义 API (`/v1/workflows`) 职责分离，互不依赖。

---

## 数据模型

### workflow_definitions 表

```sql
CREATE TABLE workflow_definitions (
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(255) NOT NULL UNIQUE,
    
    -- 元数据
    display_name    VARCHAR(255),
    description     TEXT,
    category        VARCHAR(64),
    tags            JSONB,
    
    -- 参数定义（可选，用于 UI 展示）
    parameters      JSONB,
    
    -- 内容
    content         TEXT NOT NULL,
    content_hash    VARCHAR(64),
    
    -- 时间戳
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_wf_def_name ON workflow_definitions(name);
CREATE INDEX idx_wf_def_category ON workflow_definitions(category);
```

### workflow_schedules 表

```sql
CREATE TABLE workflow_schedules (
    id              SERIAL PRIMARY KEY,
    schedule_id     VARCHAR(255) NOT NULL UNIQUE,
    workflow_name   VARCHAR(255) NOT NULL,
    
    -- Schedule 配置
    cron            VARCHAR(100) NOT NULL,
    timezone        VARCHAR(64) DEFAULT 'UTC',
    
    -- 绑定参数
    vars            JSONB,
    
    -- 状态
    enabled         BOOLEAN DEFAULT TRUE,
    
    -- Temporal Schedule ID（内部）
    temporal_id     VARCHAR(255),
    
    -- 时间戳
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (workflow_name) REFERENCES workflow_definitions(name)
);

CREATE INDEX idx_wf_sched_workflow ON workflow_schedules(workflow_name);
```

### workflow_webhooks 表

```sql
CREATE TABLE workflow_webhooks (
    id              SERIAL PRIMARY KEY,
    webhook_id      VARCHAR(255) NOT NULL UNIQUE,
    workflow_name   VARCHAR(255) NOT NULL,
    
    -- Webhook 配置
    secret          VARCHAR(255),
    
    -- 绑定参数
    vars            JSONB,
    
    -- 状态
    enabled         BOOLEAN DEFAULT TRUE,
    
    -- 时间戳
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (workflow_name) REFERENCES workflow_definitions(name)
);

CREATE INDEX idx_wf_webhook_workflow ON workflow_webhooks(workflow_name);
```

---

## 存储接口设计

```go
// DefinitionStore 工作流定义存储接口
type DefinitionStore interface {
    // CRUD
    Create(ctx context.Context, def *WorkflowDefinition) error
    Get(ctx context.Context, name string) (*WorkflowDefinition, error)
    List(ctx context.Context, filter DefinitionFilter) ([]*WorkflowDefinition, error)
    Update(ctx context.Context, def *WorkflowDefinition) error
    Delete(ctx context.Context, name string) error
    
    // 检查
    Exists(ctx context.Context, name string) (bool, error)
}

// DefinitionFilter 查询过滤器
type DefinitionFilter struct {
    Category   string
    Tags       []string
    NamePrefix string
    Limit      int
    Offset     int
}

// WorkflowDefinition 工作流定义模型
type WorkflowDefinition struct {
    Name         string
    DisplayName  string
    Description  string
    Category     string
    Tags         []string
    Parameters   map[string]interface{}
    Content      string
    ContentHash  string
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

// DatabaseDefinitionStore 数据库存储实现（GORM）
type DatabaseDefinitionStore struct {
    db *gorm.DB
}

func (s *DatabaseDefinitionStore) Create(ctx context.Context, def *WorkflowDefinition) error {
    return s.db.WithContext(ctx).Create(def).Error
}

func (s *DatabaseDefinitionStore) Get(ctx context.Context, name string) (*WorkflowDefinition, error) {
    var def WorkflowDefinition
    err := s.db.WithContext(ctx).Where("name = ?", name).First(&def).Error
    if err != nil {
        return nil, err
    }
    return &def, nil
}

func (s *DatabaseDefinitionStore) List(ctx context.Context, filter DefinitionFilter) ([]*WorkflowDefinition, error) {
    var defs []*WorkflowDefinition
    query := s.db.WithContext(ctx)
    
    if filter.Category != "" {
        query = query.Where("category = ?", filter.Category)
    }
    if filter.NamePrefix != "" {
        query = query.Where("name LIKE ?", filter.NamePrefix+"%")
    }
    if filter.Limit > 0 {
        query = query.Limit(filter.Limit)
    }
    if filter.Offset > 0 {
        query = query.Offset(filter.Offset)
    }
    
    err := query.Find(&defs).Error
    return defs, err
}

func (s *DatabaseDefinitionStore) Update(ctx context.Context, def *WorkflowDefinition) error {
    return s.db.WithContext(ctx).Model(&WorkflowDefinition{}).Where("name = ?", def.Name).Updates(def).Error
}

func (s *DatabaseDefinitionStore) Delete(ctx context.Context, name string) error {
    return s.db.WithContext(ctx).Where("name = ?", name).Delete(&WorkflowDefinition{}).Error
}

func (s *DatabaseDefinitionStore) Exists(ctx context.Context, name string) (bool, error) {
    var count int64
    err := s.db.WithContext(ctx).Model(&WorkflowDefinition{}).Where("name = ?", name).Count(&count).Error
    return count > 0, err
}
```

**注：** 模板存储由 Story 6.4 已实现的 TemplateHandler 处理，使用文件系统存储，与工作流定义存储完全独立。

---

## 执行流程

```
┌─────────────────────────────────────────────────────────────────┐
│                      完整执行流程                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  用户创建定义                                                    │
│  POST /v1/workflows { name: "my-deploy", content: "..." }       │
│         │                                                       │
│         ▼                                                       │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ workflow_definitions (DB)                                │   │
│  │   name: my-deploy                                        │   │

│  │   content: <YAML>                                        │   │
│  └─────────────────────────────────────────────────────────┘   │
│         │                                                       │
│         │  可选：创建触发器                                      │
│         ├──────────────────────────────────────────┐           │
│         │                                          │           │
│         ▼                                          ▼           │
│  ┌─────────────────┐                    ┌─────────────────┐   │
│  │ Schedule        │                    │ Webhook         │   │
│  │ cron: 0 2 * * * │                    │ id: github-push │   │
│  │ vars: {env:prod}│                    │ vars: {br:main} │   │
│  └────────┬────────┘                    └────────┬────────┘   │
│           │                                      │             │
│           │  定时触发                             │ 事件触发    │
│           │                                      │             │
│           └──────────────┬───────────────────────┘             │
│                          │                                      │
│                          │  或 API 手动触发                      │
│                          │  POST /v1/workflows/my-deploy/run    │
│                          │  { vars: { version: "1.2.3" } }      │
│                          │                                      │
│                          ▼                                      │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 参数合并                                                 │   │
│  │   YAML vars + 触发器 vars + 执行时 vars                  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                          │                                      │
│                          ▼                                      │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Temporal Workflow Execution                              │   │
│  │   WorkflowID: my-deploy-<uuid>                           │   │
│  │   Memo: { yaml: <resolved>, definition_name: "my-deploy"}│   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Story 影响分析

| Story | 当前状态 | 影响 | 修改内容 |
|-------|---------|------|---------|
| **1-9** | Done | ✅ 重构 | API 重构为 Definition + Execution 分离 |
| **1-10** | Not Started | ✅ 更新 | 基于新 API 设计 Schedule |
| **1-11** | Not Started | ✅ 更新 | 基于新 API 设计 Webhook |
| **6-4** | Done | ✅ 无影响 | 模板 API 保持独立，无需修改 |

---

## 后果

### 正面影响

✅ **概念统一** - 用户只需理解"工作流定义"一个概念  
✅ **持久存储** - 工作流定义可被查询、复用、被触发器引用  
✅ **API 清晰** - Definition API 和 Execution API 职责分明  
✅ **灵活参数化** - 支持 runs-on、timeout、matrix 动态配置  
✅ **三层覆盖** - YAML 默认 → 触发器绑定 → 执行时覆盖  
✅ **扩展性好** - DefinitionStore 接口支持未来扩展存储方式

### 负面影响

⚠️ **API 变更** - 需要重构 Story 1-9 的 API 实现  
⚠️ **数据库依赖** - 用户定义需要数据库存储  
⚠️ **迁移成本** - 现有使用方式需要适配

### 风险缓解

**风险 1: API 兼容性**
- 保留 `POST /v1/executions` 支持一次性执行（不存储定义）
- `/v1/templates` 和 `/v1/workflows` 是两个独立的 API，职责清晰

**风险 2: 数据库依赖**
- 复用 Temporal 已有的数据库连接
- 仅增加 workflow_definitions、workflow_schedules、workflow_webhooks 三个表
- 不增加额外运维复杂度

---

## 验收标准

### AC1: Definition CRUD

```bash
# 创建
POST /v1/workflows
{ "name": "my-deploy", "content": "name: my-deploy\njobs:..." }
→ 201 Created

# 查询
GET /v1/workflows
→ 200 OK { "workflows": [{"name": "my-deploy", ...}] }

# 详情
GET /v1/workflows/my-deploy
→ 200 OK { "name": "my-deploy", "content": "..." }

# 更新
PUT /v1/workflows/my-deploy
{ "content": "..." }
→ 200 OK

# 删除
DELETE /v1/workflows/my-deploy
→ 204 No Content
```

### AC2: Execution 引用 Definition

```bash
POST /v1/workflows/my-deploy/run
{ "vars": { "env": "prod" } }
→ 202 Accepted { "execution_id": "my-deploy-xxx" }
```

### AC3: Schedule 引用 Definition

```bash
POST /v1/workflows/my-deploy/schedules
{ "schedule_id": "nightly", "cron": "0 2 * * *", "vars": { "env": "prod" } }
→ 201 Created

# Schedule 触发时自动执行 my-deploy 定义
```

### AC4: 参数化执行

```bash
# 执行时传入参数覆盖
POST /v1/workflows/my-deploy/run
{ "vars": { "env": "prod", "version": "1.2.3" } }
→ 202 Accepted

# 验证参数合并优先级
# YAML 默认值 < Schedule 绑定参数 < 执行时参数
```

---

## 参考资料

- [ADR-0007: Workflow 存储与触发器架构设计](0007-workflow-storage-and-trigger-architecture.md)
- [Story 1.9: 工作流管理 API](../sprint-artifacts/1-9-workflow-management-api.md)
- [Story 1.10: Schedule API 实现](../sprint-artifacts/1-10-schedule-api-implementation.md)
- [Story 1.11: Webhook 触发器实现](../sprint-artifacts/1-11-webhook-trigger-implementation.md)
- [Story 6.4: 模板 API 端点](../sprint-artifacts/6-4-template-api-endpoint.md)
- [Temporal Schedules Documentation](https://docs.temporal.io/workflows#schedule)

---

**决策状态:** ✅ 已采纳  
**生效日期:** 2026-02-03
