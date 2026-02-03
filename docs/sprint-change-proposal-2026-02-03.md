# Sprint 变更提案：工作流定义与执行分离架构

**日期：** 2026-02-03  
**提案人：** John (Product Manager)  
**状态：** ✅ 已批准  
**批准日期：** 2026-02-03  
**批准人：** Websoft9  
**变更范围：** 架构重构 - Definition/Execution 分离  
**影响级别：** 中等 (Moderate)

---

## 1. 问题摘要

### 1.1 触发因素

在实现以下 Stories 时发现架构缺陷：
- **Story 1-10**: Schedule API 实现
- **Story 1-11**: Webhook 触发器实现
- **Story 1-9**: 工作流管理 API（已完成，需重构）

### 1.2 核心问题

**问题分类：** 技术限制在实施过程中被发现

**四个关键架构问题：**

#### 问题1：YAML 无持久存储
当前工作流 YAML 仅存储在 Temporal Memo 中，与 Execution 生命周期绑定。

**证据：**
```go
// internal/api/workflow_handler.go
startOptions := client.StartWorkflowOptions{
    Memo: map[string]interface{}{
        "workflow_yaml": string(yamlBytes),  // ❌ 仅存在 Memo
    },
}
```

**影响：**
- Execution 结束后 YAML 丢失
- Schedule/Webhook 无法引用已有工作流定义
- 无法查询"有哪些工作流定义"

#### 问题2：API 语义混淆
当前 `POST /v1/workflows` 既是"定义"又是"执行"，无法满足"仅保存定义"的需求。

**证据：**
```
POST /v1/workflows  # ❌ 提交 YAML 并立即执行
                    # ❌ 没有"仅保存定义"的能力
```

**影响：**
- Schedule/Webhook 需要引用定义，但 API 不支持
- 用户无法预先保存工作流模板供后续使用

#### 问题3：模板与定义分离
Epic 6 的工作流模板和用户工作流是两套独立体系。

**证据：**
```
/v1/templates   # 系统预置模板，文件系统存储
/v1/workflows   # 用户工作流，无持久存储
```

**影响：**
- 概念重复，用户需要理解两套 API
- 系统模板无法被 Schedule/Webhook 引用

#### 问题4：参数化不完整
当前 `${{ vars.xxx }}` 仅支持 Step 层级，不支持结构化字段。

**证据：**
```yaml
# ❌ 当前不支持
runs-on: ${{ vars.agent }}
timeout: ${{ vars.timeout }}
matrix:
  servers: ${{ vars.servers }}
```

**影响：**
- 工作流灵活性受限
- 无法动态配置执行环境

### 1.3 发现背景

**时间：** 2026-02-03  
**情况：** 在设计 Story 1-10 (Schedule API) 的实现方案时，发现无法引用已有的工作流定义，因为当前架构不支持工作流持久存储。

---

## 2. 影响分析

### 2.1 Epic 影响评估

#### Epic 1: 核心工作流引擎基础

**当前状态：** 部分完成  
**影响级别：** ⚠️ 重大影响

| Story | 当前状态 | 影响类型 | 需要的变更 |
|-------|---------|---------|-----------|
| **1-9** | ✅ Done | ⚠️ **重构** | • API 重构为 Definition/Execution 分离<br>• 新增 DefinitionStore 接口<br>• 保留 DSL Parser 和 Temporal Client |
| **1-10** | 📄 仅文档 | ✅ **更新设计** | • Schedule API 挂载到 `/v1/workflows/{name}/schedules`<br>• 支持 vars 参数绑定<br>• 直接按新架构实现 |
| **1-11** | 📄 仅文档 | ✅ **更新设计** | • Webhook API 挂载到 `/v1/workflows/{name}/webhooks`<br>• 支持 vars 参数覆盖<br>• 直接按新架构实现 |
| **1-3** | ✅ Done | ⚠️ **增强** | • runs-on 支持 `${{ vars.xxx }}`<br>• timeout 支持变量<br>• matrix 支持变量数组 |
| **1-4** | ✅ Done | ⚠️ **增强** | • 扩展表达式引擎求值范围<br>• 支持数组和对象变量 |

**新增内容：**
- DefinitionStore 接口实现（文件系统 + 数据库）
- 三层参数覆盖机制
- 数据库表设计：workflow_definitions, workflow_schedules, workflow_webhooks

#### Epic 6: 工作流模板库

**当前状态：** Story 6-4 已完成  
**影响级别：** ✅ 无需调整

| Story | 当前状态 | 影响类型 | 说明 |
|-------|---------|---------|------|
| **6-4** | ✅ Done | ✅ **无需调整** | • 已有独立的模板 API (`/v1/templates`)<br>• 存储在文件系统 `examples/workflows/`<br>• 元数据文件 `templates-metadata.json`<br>• 与工作流定义 API 清晰分离 |

#### 其他 Epic

| Epic | 影响 | 说明 |
|------|-----|------|
| Epic 2 (Agent) | ❌ 无影响 | Task Queue 路由机制不变 |
| Epic 3 (节点库) | ❌ 无影响 | 节点插件系统不变 |
| Epic 4 (扩展) | ❌ 无影响 | Plugin Manager 不变 |
| Epic 5 (客户端) | ✅ 轻微 | CLI submit 命令支持两种模式 |
| Epic 7 (可靠性) | ❌ 无影响 | Event Sourcing 架构不变 |
| Epic 8 (部署) | ❌ 无影响 | 部署架构不变 |
| Epic 9 (安全) | ❌ 无影响 | 安全机制不变 |
| Epic 10 (文档) | ✅ 需更新 | API 文档、快速开始指南 |
| Epic 11 (测试) | ✅ 需更新 | API 测试用例 |

### 2.2 工件冲突分析

#### PRD 影响

**核心目标：** ✅ 无冲突（架构改进增强价值主张）

**需要新增的需求：**

| 需求 ID | 新增内容 | 理由 |
|---------|---------|------|
| **FR3.1** | 工作流定义管理 API | 支持 Definition/Execution 分离 |
| **FR3.2** | 工作流执行 API | 明确执行语义 |
| **FR19** | 工作流定时触发（Schedule） | 完善触发机制 |
| **FR20** | 工作流事件触发（Webhook） | 完善触发机制 |

**MVP 影响：**
- ✅ MVP 范围保持不变
- ⚠️ MVP 交付时间增加约 12 天（重构工作）
- ✅ MVP 质量提升（架构更清晰）

#### 架构文档影响

**需要更新的架构章节：**

| 章节 | 影响级别 | 更新内容 |
|------|---------|---------|
| Container View | ⚠️ 重大 | Server 组件新增 DefinitionStore |
| Component View | ⚠️ 重大 | 新增：DefinitionStore (GORM + PostgreSQL) |
| API 设计 | ⚠️ 重大 | 完全重构 API 端点设计 |
| 数据模型 | ⚠️ 重大 | 新增数据表：workflow_definitions, workflow_schedules, workflow_webhooks |
| 执行流程 | ✅ 轻微 | 增加"定义查询"步骤 |

**架构决策记录：**
- ✅ **ADR-0009 已存在** - 工作流定义与执行分离架构

#### 其他工件影响

| 工件 | 影响级别 | 更新内容 |
|------|---------|---------|
| OpenAPI 规范 | ⚠️ 重大 | 完全重构 API 定义 |
| CLI 工具 | ✅ 轻微 | submit 命令支持 `--save` 和 `--run` |
| Go SDK | ✅ 轻微 | 新增方法：CreateDefinition(), RunWorkflow(), CreateSchedule() |
| 测试用例 | ⚠️ 需更新 | API 测试需要重构 |
| 文档 | ⚠️ 需更新 | 快速开始、API 参考、概念文档 |

---

## 3. 推荐方案

### 3.1 方案选择

**推荐：选项1 - 直接调整**

### 3.2 方案对比

#### 选项1：直接调整（推荐）✅

**方法：** 在现有 Epic 框架内修改和新增 Stories

**工作量：** 中等（Medium） - 约 19.5 天
- Story 1-9 重构：5 天
- 数据库设计和实现：2 天
- DefinitionStore 接口实现：3 天
- Story 1-10 和 1-11 实现：10 天
- Epic 6 验证：0.5 天
- 测试和文档更新：4 天

**风险：** 低（Low）
- 架构更清晰，长期收益大
- Story 1-10 和 1-11 未开始，无返工成本
- 核心 Temporal 集成不变

**优势：**
- ✅ 解决 Schedule/Webhook 无法实现的根本问题
- ✅ 统一模板和用户定义概念
- ✅ API 语义更清晰，用户体验更好
- ✅ 为未来扩展（工作流版本管理）奠定基础

**劣势：**
- ⚠️ Story 1-9 已完成的代码需要重构
- ⚠️ 增加数据库依赖（可复用 Temporal 的 PostgreSQL）

#### 选项2：潜在回滚 ❌

**方法：** 回滚 Story 1-9，按新架构重新实现

**工作量：** 高（High） - 约 10 天
**风险：** 中等（Medium）

**结论：** ❌ 不推荐（成本不比重构低，且影响团队士气）

#### 选项3：缩减 MVP 范围 ❌

**方法：** 延后 Schedule/Webhook，仅实现 Definition/Execution 分离

**工作量：** 低（Low） - 约 8 天
**风险：** 低（Low）

**结论：** ❌ 不推荐（产品价值降低，与 PRD 承诺不符）

### 3.3 选择理由

**为什么选择选项1？**

1. **实施努力合理** - 12 天工作量可接受
2. **技术风险低** - 核心架构（Temporal、Event Sourcing）不变
3. **长期收益大** - 清晰的 API 语义，统一的概念模型
4. **时机最佳** - Story 1-10 和 1-11 还未开始开发
5. **团队士气** - 是架构改进，而非失败返工
6. **业务价值** - 完整交付 Schedule/Webhook 能力

---

## 4. 详细变更提案

### 4.1 Story 1-9 重构

**原始范围：**
```
POST /v1/workflows        # 提交并执行
GET  /v1/workflows/:id    # 查询状态
DELETE /v1/workflows/:id  # 取消执行
```

**新范围：**

#### Definition API
```
POST   /v1/workflows                    # 创建定义
GET    /v1/workflows                    # 列表定义（支持 namespace 过滤）
GET    /v1/workflows/{name}             # 获取定义详情
PUT    /v1/workflows/{name}             # 更新定义
DELETE /v1/workflows/{name}             # 删除定义（仅 user namespace）
```

#### Execution API
```
POST   /v1/workflows/{name}/run         # 基于定义执行
POST   /v1/executions                   # 一次性执行（不存储定义）
GET    /v1/executions                   # 列表执行
GET    /v1/executions/{id}              # 查询执行状态
POST   /v1/executions/{id}/cancel       # 取消执行
POST   /v1/executions/{id}/retry        # 重试执行
GET    /v1/executions/{id}/logs         # 获取日志
```

**实现变更：**

**保留（无需改动）：**
- ✅ DSL Parser
- ✅ Validator
- ✅ Expression Engine
- ✅ Temporal Client

**新增：**
- ✅ DefinitionStore 接口
- ✅ DatabaseStore 实现（GORM + PostgreSQL）

**重构：**
- ⚠️ API Handler（按新端点重构）
- ⚠️ 请求/响应模型

### 4.2 数据库表设计

#### workflow_definitions 表

```sql
CREATE TABLE workflow_definitions (
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    namespace       VARCHAR(64) NOT NULL DEFAULT 'user',
    
    -- 元数据
    display_name    VARCHAR(255),
    description     TEXT,
    category        VARCHAR(64),
    tags            JSONB,
    
    -- 参数定义（用于 UI 展示）
    parameters      JSONB,
    
    -- 内容
    content         TEXT NOT NULL,
    content_hash    VARCHAR(64),
    
    -- 时间戳
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE (namespace, name)
);

CREATE INDEX idx_wf_def_namespace ON workflow_definitions(namespace);
CREATE INDEX idx_wf_def_category ON workflow_definitions(category);
```

#### workflow_schedules 表

```sql
CREATE TABLE workflow_schedules (
    id              SERIAL PRIMARY KEY,
    schedule_id     VARCHAR(255) NOT NULL UNIQUE,
    workflow_name   VARCHAR(255) NOT NULL,
    workflow_ns     VARCHAR(64) NOT NULL DEFAULT 'user',
    
    -- Schedule 配置
    cron            VARCHAR(100) NOT NULL,
    timezone        VARCHAR(64) DEFAULT 'UTC',
    
    -- 绑定参数
    vars            JSONB,
    
    -- 状态
    enabled         BOOLEAN DEFAULT TRUE,
    
    -- Temporal Schedule ID
    temporal_id     VARCHAR(255),
    
    -- 时间戳
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_wf_sched_workflow ON workflow_schedules(workflow_ns, workflow_name);
```

#### workflow_webhooks 表

```sql
CREATE TABLE workflow_webhooks (
    id              SERIAL PRIMARY KEY,
    webhook_id      VARCHAR(255) NOT NULL UNIQUE,
    workflow_name   VARCHAR(255) NOT NULL,
    workflow_ns     VARCHAR(64) NOT NULL DEFAULT 'user',
    
    -- Webhook 配置
    secret          VARCHAR(255),
    
    -- 绑定参数
    vars            JSONB,
    
    -- 状态
    enabled         BOOLEAN DEFAULT TRUE,
    
    -- 时间戳
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_wf_webhook_workflow ON workflow_webhooks(workflow_ns, workflow_name);
```

#### 4.2.2 数据库方案评估

**方案对比**

| 维度 | 迁移脚本 (Recommended) | ORM Auto-Migration |
|------|----------------------|--------------------|
| **版本控制** | ✅ SQL 文件可追踪 | ❌ 难以审查代码生成的 SQL |
| **可回滚性** | ✅ Down 脚本精确回滚 | ⚠️ 依赖 ORM 实现 |
| **多环境一致性** | ✅ 相同 SQL 保证一致 | ⚠️ ORM 版本差异可能导致不一致 |
| **复杂变更支持** | ✅ 数据迁移/索引重建可精确控制 | ❌ 复杂逻辑难以表达 |
| **生产安全性** | ✅ 可提前 Dry-Run | ⚠️ Auto-Migration 风险高 |
| **团队协作** | ✅ Code Review 友好 | ❌ 自动生成难以审查 |
| **学习曲线** | ⚠️ 需掌握迁移工具 | ✅ ORM 开发者熟悉 |
| **开发速度** | ⚠️ 需手写 SQL | ✅ 自动生成快速 |
| **行业标准** | ✅ Rails/Django/Laravel 标准实践 | ⚠️ 仅适用于小型项目 |
| **错误处理** | ✅ SQL 语法错误在开发时发现 | ❌ 运行时才发现问题 |
| **性能优化** | ✅ 可精细调优索引/约束 | ⚠️ ORM 生成可能不优化 |
| **合规审计** | ✅ 变更记录完整 | ❌ 难以追溯谁改了什么 |

**行业最佳实践**

**成熟框架的选择：**
- **Rails (ActiveRecord):** 迁移脚本 (`db/migrate/*.rb`)
- **Django:** 迁移脚本 (`migrations/*.py`)
- **Laravel:** 迁移脚本 (`database/migrations/*.php`)
- **Go 生态推荐：** `golang-migrate/migrate` + GORM（仅用于查询）

**示例：Temporal 项目（Go）**

Temporal 使用独立的 SQL 迁移脚本：

**项目结构：**
```
schema/
  postgresql/
    v1.0/
      temporal/
        versioned/
          v1.0/
            base.sql
          v1.1/
            add_tasks_table.sql
          v1.2/
            add_index.sql
```

**迁移脚本示例 (Up):**
```sql
-- v1.1/add_workflow_templates.up.sql
CREATE TABLE workflow_templates (
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(255) NOT NULL UNIQUE,
    category        VARCHAR(32) NOT NULL DEFAULT 'custom',
    version         VARCHAR(32) NOT NULL DEFAULT '1.0',
    author          VARCHAR(255),
    description     TEXT,
    parameters      JSONB,
    content         TEXT NOT NULL,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_wf_tmpl_category ON workflow_templates(category);
```

**迁移脚本示例 (Down):**
```sql
-- v1.1/add_workflow_templates.down.sql
DROP TABLE IF EXISTS workflow_templates;
```

**GORM Model（仅用于应用层查询）:**
```go
// internal/store/models/template.go
type WorkflowTemplate struct {
    ID          uint           `gorm:"primaryKey"`
    Name        string         `gorm:"uniqueIndex;size:255;not null"`
    Category    string         `gorm:"size:32;not null;default:'custom'"`
    Version     string         `gorm:"size:32;not null;default:'1.0'"`
    Author      string         `gorm:"size:255"`
    Description string         `gorm:"type:text"`
    Parameters  datatypes.JSON `gorm:"type:jsonb"`
    Content     string         `gorm:"type:text;not null"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// 注意：禁用 AutoMigrate
// db.AutoMigrate(&WorkflowTemplate{})  // ❌ 不使用
```

**迁移执行代码:**
```go
// cmd/server/migrate.go
func runMigrations(cfg *config.Config) error {
    m, err := migrate.New(
        "file://schema/postgresql",
        cfg.Database.URL,
    )
    if err != nil {
        return err
    }
    
    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return err
    }
    
    version, dirty, _ := m.Version()
    log.Printf("Current schema version: %d (dirty: %v)", version, dirty)
    return nil
}
```

**其他成熟 Go 项目参考：**

| 项目 | Star 数 | 迁移方案 | 链接 |
|------|---------|---------|------|
| **Temporal** | 10k+ | SQL 迁移脚本 | [schema/postgresql](https://github.com/temporalio/temporal/tree/master/schema/postgresql) |
| **Gitea** | 40k+ | `xorm/migrate` + SQL | [models/migrations](https://github.com/go-gitea/gitea/tree/main/models/migrations) |
| **Drone CI** | 30k+ | `golang-migrate` | [store/database/migrate](https://github.com/harness/drone/tree/master/store/database/migrate) |

**推荐方案：golang-migrate + GORM**

**理由：**
1. **生产级稳定性：** 迁移脚本经过 Code Review，SQL 变更可控
2. **开发效率：** GORM 用于日常 CRUD，减少样板代码
3. **最佳实践对齐：** 符合 Rails/Django/Temporal 等成熟项目的模式
4. **维护友好：** Schema 变更历史清晰，易于追踪和回滚

**实施建议：**

1. **阶段1（立即）：** 创建 `schema/postgresql/` 目录和初始迁移脚本
2. **阶段2（开发）：** 使用 GORM Model 定义，但禁用 `AutoMigrate()`
3. **阶段3（部署）：** 集成 `golang-migrate` 到启动流程
4. **阶段4（运维）：** 建立迁移脚本的测试和回滚流程

**不推荐的方案：**
- ❌ **纯 ORM AutoMigrate：** 生产环境风险高，缺乏审计能力
- ❌ **手动 SQL 执行：** 无版本控制，容易出错

### 4.3 Story 1-10 和 1-11 实现

**Story 1-10: Schedule API**

```
POST   /v1/workflows/{name}/schedules              # 创建 Schedule
GET    /v1/workflows/{name}/schedules              # 列表 Schedules
GET    /v1/workflows/{name}/schedules/{id}         # 获取详情
PUT    /v1/workflows/{name}/schedules/{id}         # 更新配置
DELETE /v1/workflows/{name}/schedules/{id}         # 删除 Schedule
POST   /v1/workflows/{name}/schedules/{id}/pause   # 暂停
POST   /v1/workflows/{name}/schedules/{id}/resume  # 恢复
```

**请求示例：**
```json
POST /v1/workflows/my-deploy/schedules
{
  "schedule_id": "nightly-deploy",
  "cron": "0 2 * * *",
  "timezone": "Asia/Shanghai",
  "vars": {
    "env": "prod",
    "version": "latest"
  },
  "enabled": true
}
```

**Story 1-11: Webhook API**

```
POST   /v1/workflows/{name}/webhooks        # 创建 Webhook
GET    /v1/workflows/{name}/webhooks        # 列表 Webhooks
DELETE /v1/workflows/{name}/webhooks/{id}   # 删除 Webhook

POST   /v1/webhooks/{id}/trigger            # 触发端点
```

**请求示例：**
```json
POST /v1/workflows/my-deploy/webhooks
{
  "webhook_id": "github-push",
  "vars": {
    "branch": "main"
  },
  "secret": "my-webhook-secret",
  "enabled": true
}
```

**触发示例：**
```json
POST /v1/webhooks/github-push/trigger
{
  "vars": {
    "commit": "abc123",
    "author": "websoft9"
  }
}
```

### 4.4 Epic 6 评估：无需调整

**结论：Story 6-4 已按正确架构实现，无需任何修改**

#### 4.4.1 当前实现验证

Story 6-4 已经实现了完整的独立模板 API，架构设计合理：

**已实现的 API：**
```go
// 路由配置（internal/api/router.go）
router.HandleFunc("/v1/templates", th.ListTemplates).Methods(http.MethodGet)
router.HandleFunc("/v1/templates/{name}", th.GetTemplate).Methods(http.MethodGet)
```

**存储架构：**
```
examples/workflows/
├── single-server-deployment.yaml      # 模板 YAML
├── multi-server-health-check.yaml
├── distributed-stack-deployment.yaml
└── templates-metadata.json            # 元数据文件
```

**架构优势：**

| 特性 | 模板 API | 工作流定义 API | 评价 |
|------|---------|---------------|------|
| **API 端点** | `/v1/templates` | `/v1/workflows` | ✅ 清晰分离 |
| **概念定位** | 可复用的蓝图（只读） | 可执行的实例（可读写） | ✅ 职责明确 |
| **存储方式** | 文件系统 | 数据库（待实现） | ✅ 技术选型合理 |
| **元数据** | 丰富（category, author, examples） | 简单（name, vars） | ✅ 各取所需 |

#### 4.4.2 模板与工作流定义的交互示例

```bash
# 步骤1：浏览可用模板
GET /v1/templates

# 步骤2：查看模板详情
GET /v1/templates/single-server-deployment

# 步骤3：用户基于模板创建工作流定义
POST /v1/workflows
{
  "name": "prod-nginx-deploy",
  "content": "...(基于模板修改的 YAML)..."
}

# 步骤4：执行或配置 Schedule
POST /v1/workflows/prod-nginx-deploy/run
POST /v1/workflows/prod-nginx-deploy/schedules
```

**关键点：** 模板和定义是**两个独立的资源**，模板为只读参考，用户基于模板创建自己的工作流定义。

#### 4.4.3 结论

Story 6-4 当前实现已经符合最佳实践：
- ✅ 模板和工作流定义是两个独立的一等公民
- ✅ API 端点清晰分离（`/v1/templates` vs `/v1/workflows`）
- ✅ 存储架构合理（文件系统 vs 数据库）
- ✅ 支持 Definition/Execution 分离架构

**无需任何调整。**
    default: production
    enum: [dev, staging, production]

# 模板内容（标准工作流 YAML）
workflow:
  jobs:
    deploy:
      runs-on: ${{ template.agent }}
      strategy:
        matrix:
          server: ${{ template.servers }}
      steps:
        - name: Deploy ${{ template.app_name }}
          uses: exec@v1
          with:
            command: |
              deploy --app=${{ template.app_name }} \
                     --env=${{ template.environment }} \
                     --server=${{ matrix.server }}
```

**模板分类体系**

| Category | 存储位置 | 权限 | 示例 |
|----------|---------|------|------|
| **system** | 文件系统 (`/etc/waterflow/templates/`) | 只读（内置） | `deploy-to-servers`, `backup-mysql`, `install-wordpress` |
| **custom** | 数据库 (`workflow_templates`) | 用户创建/修改 | 用户自定义的可复用模板 |
| **community** | 远程仓库（未来） | 只读（下载） | GitHub Community Templates |

**API 实现示例**

```go
// internal/api/template_handler.go
type TemplateHandler struct {
    templateStore store.TemplateStore
}

// GET /v1/templates
func (h *TemplateHandler) ListTemplates(c *gin.Context) {
    category := c.Query("category") // system, custom, community
    
    templates, err := h.templateStore.List(category)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{
        "templates": templates,
        "total": len(templates),
    })
}

// POST /v1/templates/{name}/use
func (h *TemplateHandler) UseTemplate(c *gin.Context) {
    var req struct {
        WorkflowName string                 `json:"workflow_name"` // 目标工作流名称
        Parameters   map[string]interface{} `json:"parameters"`     // 参数值
    }
    
    if err := c.BindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    templateName := c.Param("name")
    
    // 1. 获取模板
    template, err := h.templateStore.Get(templateName)
    if err != nil {
        c.JSON(404, gin.H{"error": "template not found"})
        return
    }
    
    // 2. 渲染模板（参数替换）
    workflowYAML, err := h.renderTemplate(template, req.Parameters)
    if err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // 3. 创建工作流定义
    definition := &store.WorkflowDefinition{
        Name:      req.WorkflowName,
        Namespace: "user",
        Content:   workflowYAML,
    }
    
    if err := h.definitionStore.Save(definition); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(201, gin.H{
        "workflow_name": req.WorkflowName,
        "message": "Workflow created from template",
    })
}
```

**存储架构**

```go
// internal/store/template_store.go
type TemplateStore interface {
    List(category string) ([]*Template, error)
    Get(name string) (*Template, error)
    Save(template *Template) error  // 仅支持 custom category
    Delete(name string) error       // 仅支持 custom category
}

type CompositeTemplateStore struct {
    systemStore   *FileSystemTemplateStore  // 读取 /etc/waterflow/templates/*.yaml
    customStore   *DatabaseTemplateStore     // 读写 workflow_templates 表
}

func (s *CompositeTemplateStore) Get(name string) (*Template, error) {
    // 优先查找 system 模板
    if tmpl, err := s.systemStore.Get(name); err == nil {
        return tmpl, nil
    }
    
    // 回退到 custom 模板
    return s.customStore.Get(name)
}
```

**数据库表设计**

```sql
CREATE TABLE workflow_templates (
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(255) NOT NULL UNIQUE,
    category        VARCHAR(32) NOT NULL DEFAULT 'custom',
    version         VARCHAR(32) NOT NULL DEFAULT '1.0',
    author          VARCHAR(255),
    description     TEXT,
    
    -- 模板参数定义（JSON Schema）
    parameters      JSONB,
    
    -- 模板内容（YAML with ${{ template.xxx }} placeholders）
    content         TEXT NOT NULL,
    
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CHECK (category IN ('custom'))  -- 仅允许 custom，system 在文件系统
);

CREATE INDEX idx_wf_tmpl_category ON workflow_templates(category);

-- 示例数据
INSERT INTO workflow_templates (name, category, version, author, description, parameters, content)
VALUES (
    'my-deploy-template',
    'custom',
    '1.0',
    'user@example.com',
    '我的自定义部署模板',
    '{"parameters": [{"name": "target", "type": "string", "required": true}]}',
    'jobs:\n  deploy:\n    steps:\n      - uses: exec@v1\n        with:\n          command: deploy ${{ template.target }}'
);
```

#### 4.4.3 模板 vs 工作流定义对比

| 维度 | 模板 (Template) | 工作流定义 (Workflow Definition) |
|------|----------------|----------------------------------|
| **用途** | 可复用的蓝图 | 具体的编排逻辑 |
| **参数** | `${{ template.xxx }}` 占位符 | `${{ vars.xxx }}` 运行时变量 |
| **存储** | 文件系统 + 数据库 | 数据库（workflow_definitions） |
| **分类** | system/custom/community | user/system namespace |
| **元数据** | version, author, parameters | content_hash, category |
| **生命周期** | 长期存在，版本化 | 用户创建，按需删除 |
| **使用方式** | `POST /v1/templates/{name}/use` | `POST /v1/workflows/{name}/execute` |
| **API 路径** | `/v1/templates` | `/v1/workflows` |

#### 4.4.4 用户工作流示例

**场景：** 用户使用系统模板创建自定义工作流

**步骤1：浏览模板库**
```bash
GET /v1/templates?category=system

Response:
{
  "templates": [
    {
      "name": "deploy-to-servers",
      "category": "system",
      "version": "1.0",
      "description": "批量部署应用到多台服务器",
      "parameters": [
        {"name": "servers", "type": "array", "required": true},
        {"name": "app_name", "type": "string", "required": true}
      ]
    }
  ]
}
```

**步骤2：使用模板创建工作流**
```bash
POST /v1/templates/deploy-to-servers/use
{
  "workflow_name": "deploy-myapp",
  "parameters": {
    "servers": ["web1", "web2", "web3"],
    "app_name": "myapp",
    "environment": "production"
  }
}

Response:
{
  "workflow_name": "deploy-myapp",
  "message": "Workflow created from template"
}
```

**步骤3：执行工作流**
```bash
POST /v1/workflows/deploy-myapp/execute
{
  "vars": {
    "version": "v1.2.3"  # 可选的运行时参数
  }
}
```

#### 4.4.5 优势总结

| 优势 | 说明 |
|------|------|
| **概念清晰** | 模板和工作流定义职责分离，用户容易理解 |
| **扩展性强** | 支持多种模板来源（system/custom/community） |
| **用户体验好** | RESTful API 语义明确，`/v1/templates` 直观 |
| **参数化完整** | `parameters` 定义 + `${{ template.xxx }}` 渲染 |
| **元数据丰富** | version, author, description 支持模板管理 |
| **技术架构合理** | 文件系统（system）+ 数据库（custom）混合存储 |

### 4.5 DSL 参数化增强

**扩展支持：**

```yaml
name: deploy-workflow

# ✅ 新增：支持顶层变量定义
vars:
  agent: default-agent
  timeout: 30m
  env: production
  servers:
    - web1
    - web2

jobs:
  deploy:
    # ✅ 新增：runs-on 支持变量
    runs-on: ${{ vars.agent }}
    
    # ✅ 新增：timeout 支持变量
    timeout: ${{ vars.timeout }}
    
    # ✅ 新增：matrix 支持变量
    strategy:
      matrix:
        server: ${{ vars.servers }}
    
    steps:
      - name: Deploy to ${{ matrix.server }}
        uses: exec@v1
        with:
          command: deploy --env=${{ vars.env }}
```

**三层参数覆盖机制：**

```
优先级（低 → 高）：

1. YAML 默认值
   vars:
     env: dev
     timeout: 30m

     ↓

2. 触发器绑定参数（Schedule/Webhook 创建时指定）
   { "env": "staging" }

     ↓

3. 执行时覆盖（API 调用或 Webhook payload）
   { "env": "prod", "debug": true }

最终合并结果：
   { env: "prod", timeout: "30m", debug: true }
```

---

## 5. 实施计划

### 5.1 阶段划分

#### 阶段1：重构 Story 1-9（5 天）

**任务：**
1. 设计和实现 DefinitionStore 接口
2. 实现 DatabaseDefinitionStore（GORM + PostgreSQL）
3. 重构 API Handler 为 Definition/Execution 分离
4. 更新单元测试

**交付物：**
- ✅ DefinitionStore 接口及实现
- ✅ Definition API 完整实现
- ✅ Execution API 完整实现
- ✅ 单元测试覆盖率 > 80%

#### 阶段2：数据库设计和实现（2 天）

**任务：**
1. 编写数据库迁移脚本
2. 实现 DatabaseStore CRUD 操作
3. 实现事务处理和并发控制
4. 编写数据库集成测试

**交付物：**
- ✅ 数据库表结构（workflow_definitions, workflow_schedules, workflow_webhooks）
- ✅ 迁移脚本
- ✅ DatabaseStore 完整实现
- ✅ 集成测试

#### 阶段3：实现 Story 1-10 和 1-11（按原计划）

**Story 1-10（5 天）：**
1. Schedule API Handler 实现
2. Temporal Schedules 集成
3. 参数绑定和覆盖机制
4. 单元测试和集成测试

**Story 1-11（5 天）：**
1. Webhook API Handler 实现
2. Webhook 触发端点实现
3. Secret 验证机制
4. 单元测试和集成测试

**交付物：**
- ✅ Schedule 管理 API
- ✅ Webhook 管理和触发 API
- ✅ 完整测试覆盖

#### 阶段4：Epic 6 验证（0.5 天）

**任务：**
1. 验证模板 API 与新的工作流定义 API 的集成
2. 测试模板到定义的转换流程
3. 确认模板和定义的清晰边界

**交付物：**
- ✅ 模板与定义集成测试通过
- ✅ 确认架构分离符合预期

#### 阶段5：测试和文档更新（2 天）

**任务：**
1. 端到端测试场景
2. API 文档更新（OpenAPI 规范）
3. 快速开始指南更新
4. 架构文档更新
5. CLI 工具文档更新

**交付物：**
- ✅ 完整 API 测试套件
- ✅ 更新后的文档
- ✅ 快速开始指南

### 5.2 时间线

```
Week 1:
  Day 1-3: 阶段1 - 重构 Story 1-9 (DefinitionStore)
  Day 4-5: 阶段2 - 数据库设计和实现

Week 2:
  Day 1-3: 阶段3 - Story 1-10 (Schedule API)
  Day 4-5: 阶段3 - Story 1-11 (Webhook API) [部分]

Week 3:
  Day 1-2: 阶段3 - Story 1-11 (Webhook API) [完成]
  Day 3上午: 阶段4 - Epic 6 验证 (0.5 天)
  Day 3下午-5: 阶段5 - 测试和文档更新
```

**总时间：约 13 个工作日（2.5 周）**

### 5.3 依赖关系

```
阶段1 (Story 1-9 重构)
    ↓
阶段2 (数据库实现)
    ↓
阶段3 (Story 1-10, 1-11)
    ↓
阶段4 (Story 6-4 调整)
    ↓
阶段5 (测试和文档)
```

**关键依赖：**
- Story 1-10 和 1-11 **依赖** Story 1-9 重构完成
- Story 6-4 调整 **依赖** Story 1-9 Definition API 实现
- 所有阶段 **依赖** 数据库表结构完成

---

## 6. 风险评估与缓解

### 6.1 技术风险

#### 风险1：数据库迁移复杂性

**风险等级：** 低  
**描述：** 新增数据库表，需要迁移脚本和版本管理

**缓解措施：**
- ✅ 使用成熟的迁移工具（如 golang-migrate）
- ✅ 提供回滚脚本
- ✅ 在测试环境充分验证
- ✅ 复用 Temporal 已有的 PostgreSQL 实例

#### 风险2：API 向后兼容性

**风险等级：** 低  
**描述：** Story 1-9 API 重构可能影响已有集成

**缓解措施：**
- ✅ 保留 `POST /v1/executions` 支持一次性执行
- ✅ Story 1-9 目前处于内部使用阶段，无外部依赖
- ✅ 提供详细的迁移指南

#### 风险3：Story 1-9 重构引入 Bug

**风险等级：** 低  
**描述：** 重构可能引入新的 Bug

**缓解措施：**
- ✅ 保留现有单元测试
- ✅ 增加集成测试覆盖
- ✅ Code Review 严格把关
- ✅ 在测试环境充分验证

### 6.2 进度风险

#### 风险1：估时不准确

**风险等级：** 中  
**描述：** 实际开发时间可能超出估计

**缓解措施：**
- ✅ 预留 20% 缓冲时间（15 天计划 = 12 天估计 + 3 天缓冲）
- ✅ 每日 Stand-up 跟踪进度
- ✅ 及时识别阻塞问题并升级

#### 风险2：并行开发冲突

**风险等级：** 低  
**描述：** 多个阶段可能存在代码冲突

**缓解措施：**
- ✅ 严格按阶段顺序执行（依赖关系明确）
- ✅ 使用 Feature Branch 开发
- ✅ 频繁集成到主分支

### 6.3 团队风险

#### 风险1：团队士气影响

**风险等级：** 低  
**描述：** Story 1-9 重构可能影响团队士气

**缓解措施：**
- ✅ 强调是**架构改进**，而非失败
- ✅ 清晰沟通长期价值
- ✅ 认可团队已完成的工作（DSL Parser、Temporal 集成等大部分代码可复用）

---

## 7. 成功标准

### 7.1 功能验收

- [ ] **Definition API** 完整实现并测试通过
- [ ] **Execution API** 完整实现并测试通过
- [ ] **DefinitionStore** 支持文件系统和数据库存储
- [ ] **Schedule API** 可以引用定义并成功触发执行
- [ ] **Webhook API** 可以引用定义并接收触发请求
- [ ] **三层参数覆盖** 机制验证正确
- [ ] **DSL 参数化增强** 验证 runs-on、timeout、matrix 支持变量
- [ ] **Epic 6 验证** 模板 API 与定义 API 协作正常

### 7.2 质量标准

- [ ] **单元测试覆盖率** > 80%
- [ ] **集成测试** 覆盖所有 API 端点
- [ ] **端到端测试** 通过 2 个场景（定时部署 + Webhook 触发）
- [ ] **代码审查** 所有变更通过 Code Review
- [ ] **性能基准** API 响应时间 < 500ms

### 7.3 文档标准

- [ ] **OpenAPI 规范** 完整更新
- [ ] **快速开始指南** 包含新 API 示例
- [ ] **API 参考文档** 完整覆盖所有端点
- [ ] **架构文档** 更新 DefinitionStore 设计
- [ ] **ADR-0009** 已存在且完整

---

## 8. 交接计划

### 8.1 角色职责

| 角色 | 职责 | 具体任务 |
|------|------|---------|
| **开发团队** | 实施重构和新功能 | • 重构 Story 1-9 API<br>• 实现 DefinitionStore<br>• 实现 Story 1-10/1-11<br>• 调整 Story 6-4<br>• 更新测试用例 |
| **架构师** | 审查架构一致性 | • 审查 DefinitionStore 设计<br>• 审查 API 设计一致性<br>• 审查数据库表设计 |
| **产品经理** | 更新文档和验收标准 | • 更新 PRD 需求<br>• 更新 Epic 1 和 Epic 6 描述<br>• 更新验收标准 |
| **技术写作** | 更新技术文档 | • 更新 API 参考文档<br>• 更新快速开始指南<br>• 更新架构文档 |

### 8.2 里程碑检查点

| 里程碑 | 检查内容 | 负责人 |
|--------|---------|--------|
| **M1: Story 1-9 重构完成** | • DefinitionStore 实现<br>• Definition/Execution API 分离<br>• 单元测试通过 | 开发团队 + 架构师 |
| **M2: 数据库实现完成** | • 表结构创建<br>• DatabaseStore 实现<br>• 迁移脚本验证 | 开发团队 |
| **M3: Story 1-10/1-11 完成** | • Schedule/Webhook API 实现<br>• 集成测试通过<br>• 参数覆盖机制验证 | 开发团队 |
| **M4: 文档更新完成** | • OpenAPI 规范更新<br>• 快速开始指南更新<br>• 架构文档更新 | 技术写作 + 产品经理 |

### 8.3 移交清单

**开发团队 → 测试团队：**
- [ ] 完整的代码变更（Feature Branch）
- [ ] 单元测试和集成测试
- [ ] API 测试脚本
- [ ] 测试环境部署脚本

**开发团队 → 产品团队：**
- [ ] 功能演示（Demo）
- [ ] 验收标准检查清单
- [ ] 已知问题清单（如有）

**产品团队 → 用户（内部）：**
- [ ] 更新后的 API 文档
- [ ] 迁移指南（如需要）
- [ ] 快速开始指南

---

## 9. 下一步行动

### 9.1 立即行动（本周）

1. **产品经理**：
   - [ ] 获得本提案的批准
   - [ ] 更新 Epic 1 和 Epic 6 的描述和验收标准
   - [ ] 更新 PRD 需求清单

2. **开发团队**：
   - [ ] 创建 Feature Branch: `feature/definition-execution-separation`
   - [ ] 设计 DefinitionStore 接口详细规范
   - [ ] 开始阶段1：Story 1-9 重构

3. **架构师**：
   - [ ] 审查 DefinitionStore 接口设计
   - [ ] 审查数据库表设计

### 9.2 短期行动（下周）

1. **开发团队**：
   - [ ] 完成阶段1：Story 1-9 重构
   - [ ] 完成阶段2：数据库实现
   - [ ] 开始阶段3：Story 1-10 实现

2. **技术写作**：
   - [ ] 开始更新 OpenAPI 规范草稿

### 9.3 中期行动（第3周）

1. **开发团队**：
   - [ ] 完成阶段3：Story 1-10 和 1-11
   - [ ] 完成阶段4：Story 6-4 调整
   - [ ] 完成阶段5：测试和文档更新

2. **测试团队**：
   - [ ] 执行完整的端到端测试
   - [ ] 验证向后兼容性

3. **产品团队**：
   - [ ] 验收所有变更
   - [ ] 批准发布

---

## 10. 附录

### 10.1 参考文档

- [ADR-0009: 工作流定义与执行分离架构](docs/adr/0009-workflow-definition-execution-separation.md)
- [PRD: Waterflow 产品需求文档](docs/prd.md)
- [Architecture: Waterflow 架构设计文档](docs/architecture.md)
- [Epic 1: 核心工作流引擎基础](docs/epics.md#epic-1)
- [Epic 6: 工作流模板库](docs/epics.md#epic-6)

### 10.2 API 变更对比

#### 原 API（Story 1-9 已实现）

```
POST   /v1/workflows        # 提交并执行
GET    /v1/workflows/:id    # 查询状态
DELETE /v1/workflows/:id    # 取消执行
```

#### 新 API（Definition/Execution 分离）

**Definition Management:**
```
POST   /v1/workflows                    # 创建定义
GET    /v1/workflows                    # 列表定义
GET    /v1/workflows/{name}             # 获取详情
PUT    /v1/workflows/{name}             # 更新定义
DELETE /v1/workflows/{name}             # 删除定义
```

**Execution Management:**
```
POST   /v1/workflows/{name}/run         # 基于定义执行
POST   /v1/executions                   # 一次性执行
GET    /v1/executions                   # 列表执行
GET    /v1/executions/{id}              # 查询状态
POST   /v1/executions/{id}/cancel       # 取消执行
POST   /v1/executions/{id}/retry        # 重试执行
GET    /v1/executions/{id}/logs         # 获取日志
```

**Schedule Management (Story 1-10):**
```
POST   /v1/workflows/{name}/schedules
GET    /v1/workflows/{name}/schedules
GET    /v1/workflows/{name}/schedules/{id}
PUT    /v1/workflows/{name}/schedules/{id}
DELETE /v1/workflows/{name}/schedules/{id}
POST   /v1/workflows/{name}/schedules/{id}/pause
POST   /v1/workflows/{name}/schedules/{id}/resume
```

**Webhook Management (Story 1-11):**
```
POST   /v1/workflows/{name}/webhooks
GET    /v1/workflows/{name}/webhooks
DELETE /v1/workflows/{name}/webhooks/{id}
POST   /v1/webhooks/{id}/trigger
```

**Template API (Story 6-4):**
```
GET /v1/templates           # 独立的模板 API（文件系统存储）
GET /v1/templates/{name}    # 与 /v1/workflows 清晰分离
```

### 10.3 变更影响矩阵

| 组件 | 变更类型 | 影响级别 | 工作量（天） |
|------|---------|---------|-------------|
| Story 1-9 API | 重构 | ⚠️ 高 | 5 |
| DefinitionStore | 新增 | ⚠️ 高 | 3 |
| 数据库表 | 新增 | ⚠️ 中 | 2 |
| Story 1-10 | 新实现 | ✅ 中 | 5 |
| Story 1-11 | 新实现 | ✅ 中 | 5 |
| Story 6-4 | 验证 | ✅ 低 | 0.5 |
| DSL Parser | 增强 | ✅ 低 | 包含在 Story 1-9 |
| 测试用例 | 更新 | ⚠️ 中 | 2 |
| 文档 | 更新 | ⚠️ 中 | 2 |
| **总计** | | | **~19.5 天** |

---

## 11. 批准签字

### 11.1 提案批准

**产品经理签字：** John (PM Agent)  日期：2026-02-03

**项目负责人签字：** Websoft9  日期：2026-02-03

**批准意见：** 提案已批准，同意按照选项1（直接调整）方案实施。工作量评估合理，技术风险可控，长期收益明显。

### 11.2 实施批准

**实施开始日期：** 2026-02-03  
**预计完成日期：** 2026-02-20（2.5 周）

**批准执行：** ✅ 已批准立即开始阶段1实施

**重要结论：** Epic 6 (Story 6-4) 已按正确架构实现，模板和工作流定义是两个独立的 API，无需调整。总工作量优化至 19.5 天。

---

**提案版本：** 1.0  
**最后更新：** 2026-02-03  
**状态：** ✅ 已批准 - 实施中
