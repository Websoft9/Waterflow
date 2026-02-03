# Waterflow REST API 完整清单

**版本:** v1  
**最后更新:** 2026-02-03  
**总计:** 32 个 API 端点

> **架构变更说明 (ADR-0009):**  
> 自 2026-02-03 起，API 采用 **Definition/Execution 分离架构**：
> - `/v1/workflows` - 工作流定义管理 (CRUD)
> - `/v1/executions` - 工作流执行管理
> - Schedule/Webhook 挂载到 `/v1/workflows/{name}/` 下

---

## 📊 API 统计

| 类别 | 端点数量 | Story |
|------|---------|-------|
| **基础设施** | 4 | Story 1.2 |
| **工作流定义** | 5 | Story 1.9 |
| **工作流执行** | 7 | Story 1.9 |
| **定时调度** | 6 | Story 1.10 |
| **Webhook 触发** | 5 | Story 1.11 |
| **模板管理** | 2 | Story 6.4 |
| **节点管理** | 1 | Story 5.6 |
| **YAML 验证** | 1 | Story 5.2 |
| **审计日志** | 1 | Story 9.3 |
| **总计** | **32** | - |

---

## 1️⃣ 基础设施 API (4 个) - Story 1.2

### 1.1 健康检查
```
GET /health
```
- **用途**: 服务基础健康检查
- **返回**: `{"status": "healthy"}`
- **依赖**: 无 (不检查外部服务)

### 1.2 就绪检查
```
GET /ready
```
- **用途**: 服务就绪检查 (包含依赖)
- **返回**: `{"status": "ready"}` 或 503
- **依赖**: Temporal 连接状态、数据库连接

### 1.3 Prometheus 监控
```
GET /metrics
```
- **用途**: Prometheus 格式监控指标
- **返回**: 文本格式指标
- **指标**: HTTP 请求、工作流计数、Go runtime

### 1.4 版本信息
```
GET /version
```
- **用途**: 服务版本信息
- **返回**: `{"version": "1.0.0", "commit": "abc123", "build_time": "..."}`

---

## 2️⃣ 工作流定义 API (5 个) - Story 1.9

> **说明**: 工作流定义是持久化存储的 YAML 工作流，可被 Schedule/Webhook 引用。

### 2.1 创建工作流定义
```
POST /v1/workflows
Content-Type: application/json

{
  "name": "deploy-app",
  "description": "Deploy application to servers",
  "content": "name: deploy-app\njobs:...",
  "category": "deployment"
}
```
- **用途**: 创建新的工作流定义
- **返回**: 201 Created
```json
{
  "name": "deploy-app",
  "namespace": "user",
  "description": "Deploy application to servers",
  "category": "deployment",
  "created_at": "2026-02-03T10:00:00Z"
}
```
- **验证**: YAML 语法和语义验证
- **存储**: 用户定义存储在数据库

### 2.2 列出工作流定义
```
GET /v1/workflows
GET /v1/workflows?category=deployment&page=1&limit=20
```
- **用途**: 分页列出工作流定义
- **过滤参数**:
  - `category`: deployment, monitoring, automation 等
  - `page`, `limit`: 分页参数
- **返回**:
```json
{
  "workflows": [
    {
      "name": "deploy-app",
      "description": "...",
      "category": "deployment",
      "created_at": "2026-02-03T10:00:00Z",
      "updated_at": "2026-02-03T10:00:00Z"
    }
  ],
  "total": 15
}
```

### 2.3 获取工作流定义详情
```
GET /v1/workflows/{name}
```
- **用途**: 获取单个工作流定义 (含 YAML 内容)
- **返回**:
```json
{
  "name": "deploy-app",
  "description": "Deploy application to servers",
  "category": "deployment",
  "content": "name: deploy-app\nvars:\n  env: prod\njobs:...",
  "parameters": [
    {"name": "env", "type": "string", "default": "prod"}
  ],
  "created_at": "2026-02-03T10:00:00Z",
  "updated_at": "2026-02-03T10:00:00Z"
}
```
- **404**: 工作流定义不存在

### 2.4 更新工作流定义
```
PUT /v1/workflows/{name}
Content-Type: application/json

{
  "description": "Updated description",
  "content": "name: deploy-app\njobs:..."
}
```
- **用途**: 更新工作流定义
- **返回**: 200 OK + 更新后的定义
- **验证**: YAML 语法和语义验证

### 2.5 删除工作流定义
```
DELETE /v1/workflows/{name}
```
- **用途**: 删除工作流定义
- **返回**: 204 No Content
- **冲突检查**: 存在关联的 Schedule/Webhook 时返回 409

---

## 3️⃣ 工作流执行 API (7 个) - Story 1.9

> **说明**: 工作流执行是一次具体的运行实例。

### 3.1 执行工作流定义
```
POST /v1/workflows/{name}/run
Content-Type: application/json

{
  "namespace": "user",
  "vars": {
    "env": "production",
    "version": "1.2.3"
  }
}
```
- **用途**: 执行已保存的工作流定义
- **返回**: 202 Accepted
```json
{
  "execution_id": "deploy-app-abc123",
  "workflow_name": "deploy-app",
  "status": "running",
  "started_at": "2026-02-03T10:00:00Z"
}
```
- **参数覆盖**: `vars` 覆盖 YAML 中的默认值
- **404**: 工作流定义不存在

### 3.2 直接执行工作流 (一次性)
```
POST /v1/executions
Content-Type: application/json

{
  "workflow": "name: one-time-task\njobs:...",
  "vars": {"key": "value"},
  "dry_run": false
}
```
- **用途**: 直接执行 YAML (不存储定义)
- **返回**: 202 Accepted
```json
{
  "execution_id": "one-time-task-xyz789",
  "status": "running"
}
```
- **场景**: 临时任务、测试、调试
- **dry_run**: 仅验证不执行

### 3.3 查询执行详情
```
GET /v1/executions/{id}
GET /v1/executions/{id}?include=events
```
- **用途**: 查询单个执行的状态
- **返回**: 完整执行信息 (状态、进度、Job/Step 详情)
- **特性**: 可选包含原始 Event History

### 3.4 列出执行
```
GET /v1/executions
GET /v1/executions?workflow_name=deploy-app&status=running
GET /v1/executions?start_time_from=2026-01-01T00:00:00Z
```
- **用途**: 分页列出工作流执行
- **过滤**: workflow_name, status, 时间范围, trigger_type
- **返回**: 执行列表 + 分页信息

### 3.5 获取执行日志
```
GET /v1/executions/{id}/logs
GET /v1/executions/{id}/logs?level=error,warn&job=deploy&stream=true
```
- **用途**: 查询执行日志
- **返回**: JSON Lines 格式日志
- **特性**: 支持过滤、实时流 (SSE)

### 3.6 取消执行
```
POST /v1/executions/{id}/cancel
```
- **用途**: 取消运行中的执行
- **返回**: 202 Accepted
- **行为**: 优雅停止，传播到所有 Activity

### 3.7 强制终止执行
```
POST /v1/executions/{id}/terminate
Content-Type: application/json

{
  "reason": "Resource cleanup required"
}
```
- **用途**: 强制终止运行中或卡住的执行
- **返回**: 204 No Content
- **行为**: 立即终止，不执行清理逻辑

---

## 4️⃣ 定时调度 API (6 个) - Story 1.10

> **说明**: Schedule 挂载在工作流定义下，引用而非复制 YAML。

### 4.1 创建 Schedule
```
POST /v1/workflows/{name}/schedules
Content-Type: application/json

{
  "schedule_id": "nightly-deploy",
  "cron": "0 2 * * *",
  "timezone": "Asia/Shanghai",
  "vars": {
    "env": "production"
  },
  "overlap_policy": "skip",
  "enabled": true
}
```
- **用途**: 为工作流定义创建定时调度
- **返回**: 201 Created
```json
{
  "schedule_id": "nightly-deploy",
  "workflow_name": "deploy-app",
  "cron": "0 2 * * *",
  "timezone": "Asia/Shanghai",
  "vars": {"env": "production"},
  "next_run_time": "2026-02-04T02:00:00+08:00",
  "enabled": true
}
```
- **vars**: 执行时覆盖 YAML 默认值
- **404**: 工作流定义不存在

### 4.2 列出 Schedules
```
GET /v1/workflows/{name}/schedules
```
- **用途**: 列出工作流的所有 Schedules
- **返回**: Schedule 列表

### 4.3 查询 Schedule 详情
```
GET /v1/workflows/{name}/schedules/{schedule_id}
```
- **用途**: 查询单个 Schedule
- **返回**: 完整 Schedule 信息 + 最近执行历史

### 4.4 更新 Schedule
```
PUT /v1/workflows/{name}/schedules/{schedule_id}
Content-Type: application/json

{
  "cron": "0 3 * * *",
  "vars": {"env": "staging"},
  "enabled": false
}
```
- **用途**: 更新 Schedule 配置
- **返回**: 200 OK + 更新后的 Schedule

### 4.5 删除 Schedule
```
DELETE /v1/workflows/{name}/schedules/{schedule_id}
```
- **用途**: 删除 Schedule
- **返回**: 204 No Content
- **行为**: 不影响正在运行的执行

### 4.6 手动触发 Schedule
```
POST /v1/workflows/{name}/schedules/{schedule_id}/trigger
Content-Type: application/json

{
  "vars": {"version": "1.2.4"}
}
```
- **用途**: 立即触发一次执行
- **返回**: 202 Accepted + execution_id
- **vars**: 可选，覆盖 Schedule 绑定的 vars

---

## 5️⃣ Webhook 触发 API (5 个) - Story 1.11

> **说明**: Webhook 挂载在工作流定义下。

### 5.1 创建 Webhook
```
POST /v1/workflows/{name}/webhooks
Content-Type: application/json

{
  "webhook_id": "github-push",
  "secret": "my-webhook-secret",
  "vars": {
    "branch": "main"
  },
  "enabled": true
}
```
- **用途**: 为工作流定义创建 Webhook 触发器
- **返回**: 201 Created
```json
{
  "webhook_id": "github-push",
  "workflow_name": "deploy-app",
  "trigger_url": "https://waterflow.example.com/api/v1/webhooks/github-push/trigger",
  "vars": {"branch": "main"},
  "enabled": true
}
```

### 5.2 列出 Webhooks
```
GET /v1/workflows/{name}/webhooks
```
- **用途**: 列出工作流的所有 Webhooks
- **返回**: Webhook 列表

### 5.3 查询 Webhook 详情
```
GET /v1/workflows/{name}/webhooks/{webhook_id}
```
- **用途**: 查询单个 Webhook
- **返回**: 完整 Webhook 信息

### 5.4 删除 Webhook
```
DELETE /v1/workflows/{name}/webhooks/{webhook_id}
```
- **用途**: 删除 Webhook
- **返回**: 204 No Content
- **行为**: trigger_url 立即失效

### 5.5 Webhook 触发端点
```
POST /api/v1/webhooks/{webhook_id}/trigger
Content-Type: application/json
X-Hub-Signature-256: sha256=...

{
  "ref": "refs/heads/main",
  "vars": {"commit": "abc123"}
}
```
- **用途**: 接收外部 Webhook 事件并触发执行
- **返回**: 202 Accepted + execution_id
- **安全**: HMAC 签名验证
- **vars**: payload 中的 vars 作为执行时覆盖

---

## 6️⃣ 模板管理 API (2 个) - Story 6.4

> **架构说明**: `/v1/templates` 是独立的模板 API，与工作流定义 API (`/v1/workflows`) 清晰分离。模板是可复用的蓝图（只读），工作流定义是可执行的实例（可读写）。

### 6.1 列出模板
```
GET /v1/templates
```
- **用途**: 获取所有系统预置工作流模板
- **存储**: 文件系统 `examples/workflows/`
- **返回**: 模板列表 (包含 name, category, description, parameters)
- **过滤**: 支持 `?category=deployment` 过滤

### 6.2 获取模板详情
```
GET /v1/templates/{name}
```
- **用途**: 获取特定模板的详情和 YAML 内容
- **返回**: 完整模板 YAML + 元数据

---

## 7️⃣ 节点管理 API (1 个) - Story 5.6

### 7.1 列出节点
```
GET /v1/nodes
GET /v1/nodes?format=yaml
```
- **用途**: 列出所有可用节点
- **返回**: 节点列表 (name, version, description)
- **格式**: JSON 或 YAML

---

## 8️⃣ YAML 验证 API (1 个) - Story 5.2

### 8.1 验证 YAML
```
POST /v1/validate
Content-Type: application/json

{
  "yaml": "name: Test\njobs:..."
}
```
- **用途**: 验证 YAML 工作流语法
- **返回**: 验证结果 + 错误详情
- **特性**: 离线验证，无需 Temporal

---

## 9️⃣ 审计日志 API (1 个) - Story 9.3

### 9.1 查询审计日志
```
GET /v1/audit?action=workflow.create&user=admin&start_time=2026-01-01T00:00:00Z
```
- **用途**: 查询系统审计日志
- **过滤**: 操作类型、用户、时间范围
- **返回**: 审计日志列表
- **操作类型**: workflow.create, workflow.delete, schedule.create, webhook.trigger 等

---

## 🚀 Post-MVP API

### 执行扩展
```
POST /v1/executions/{id}/retry           # 重试失败的执行 (P1)
POST /v1/executions/{id}/archive         # 归档执行记录 (P2)
GET  /v1/executions/{id}/events          # 原始 Event History (P2)
```

### 工作流定义扩展
```
POST /v1/workflows/{name}/fork           # Fork 定义 (P2)
GET  /v1/workflows/{name}/versions       # 版本历史 (P3)
```

---

## 📈 API 设计特点

### 统一规范
- ✅ RESTful 风格 (资源导向)
- ✅ 版本化 (/v1/)
- ✅ 统一错误格式 (RFC 7807)
- ✅ Request-ID 追踪
- ✅ CORS 支持

### 架构原则 (ADR-0009)
- ✅ **Definition/Execution 分离** - 定义持久化，执行是运行实例
- ✅ **触发器挂载** - Schedule/Webhook 属于工作流定义
- ✅ **混合存储** - system=文件系统，user=数据库
- ✅ **参数三层覆盖** - YAML默认 → 触发器绑定 → 执行时

### 性能要求
- ✅ 基础查询 < 200ms
- ✅ 列表查询 < 300ms
- ✅ 日志查询 < 500ms (中型工作流)
- ✅ 实时流式输出 (SSE)

### 安全特性
- ✅ HTTPS/TLS 支持 (Story 9.1)
- ✅ API 认证 (JWT/API Key, Story 9.3)
- ✅ 限流 (100 req/min per IP)
- ✅ HMAC 签名验证 (Webhook)
- ✅ 审计日志 (Story 9.3)

---

## 🔗 相关文档

- [ADR-0009: 工作流定义与执行分离架构](./adr/0009-workflow-definition-execution-separation.md)
- [API 完整参考](./api-guide.md) - 每个端点的详细文档
- [OpenAPI 规范](../api/openapi.yaml) - Story 10.2
- [快速开始](./quick-start.md) - API 使用示例

---

**当前状态:** 基于 ADR-0009 架构，提供 **32 个生产级 REST API** 🎉
