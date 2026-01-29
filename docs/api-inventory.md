# Waterflow REST API 完整清单

**版本:** v1  
**最后更新:** 2026-01-28  
**总计:** 29 个 API 端点

---

## 📊 API 统计

| 类别 | 端点数量 | Story |
|------|---------|-------|
| **基础设施** | 4 | Story 1.2 |
| **工作流管理** | 7 | Story 1.9 |
| **定时调度** | 6 | Story 1.10 |
| **Webhook 触发** | 6 | Story 1.11 |
| **模板管理** | 2 | Story 6.4 |
| **节点管理** | 1 | Story 5.6 |
| **YAML 验证** | 1 | Story 5.2 |
| **审计日志** | 1 | Story 9.3 |
| **Agent 管理** | 2 | Story 1.9 |
| **总计** | **29** | - |

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
- **依赖**: Temporal 连接状态

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

## 2️⃣ 工作流管理 API (7 个) - Story 1.9

### 2.1 提交工作流
```
POST /v1/workflows
Content-Type: application/json

{
  "yaml": "name: Deploy\njobs:...",
  "dry_run": false
}
```
- **用途**: 提交新工作流
- **返回**: `{"workflow_id": "uuid", "status": "pending"}`
- **特性**: 支持 dry-run 验证

### 2.2 查询工作流详情
```
GET /v1/workflows/{id}
GET /v1/workflows/{id}?include=events
```
- **用途**: 查询单个工作流状态
- **返回**: 完整工作流信息 (状态、进度、Job/Step 详情)
- **特性**: 可选包含原始 Event History

### 2.3 列出工作流
```
GET /v1/workflows?page=1&limit=20&status=running&workflow_name=Deploy&start_time_from=2026-01-01T00:00:00Z
```
- **用途**: 分页列出工作流
- **过滤**: 状态、名称、时间范围
- **返回**: 工作流列表 + 分页信息

### 2.4 获取工作流日志
```
GET /v1/workflows/{id}/logs
GET /v1/workflows/{id}/logs?level=error,warn&job=deploy&stream=true
```
- **用途**: 查询工作流日志
- **返回**: JSON Lines 格式日志
- **特性**: 支持过滤、实时流 (SSE)

### 2.5 取消工作流
```
POST /v1/workflows/{id}/cancel
```
- **用途**: 取消运行中的工作流
- **返回**: 202 Accepted
- **行为**: 优雅停止,传播到所有 Activity

### 2.6 重新运行工作流
```
POST /v1/workflows/{id}/rerun
Content-Type: application/json

{
  "vars": {"env": "staging"},
  "skip_successful": true,
  "from_job": "deploy"
}
```
- **用途**: 重新运行已完成工作流
- **返回**: `{"workflow_id": "new-uuid"}`
- **特性**: 支持变量覆盖、跳过成功 Step、指定起点

### 2.7 强制终止工作流
```
POST /v1/workflows/{id}/terminate
Content-Type: application/json

{
  "reason": "Resource cleanup required"
}
```
- **用途**: 强制终止运行中或卡住的工作流
- **返回**: 204 No Content
- **行为**: 立即终止，不执行清理逻辑 (vs Cancel 优雅停止)
- **场景**: 工作流卡死、资源泄漏、紧急停止

---

## 3️⃣ 定时调度 API (6 个) - Story 1.10

### 3.1 创建 Schedule
```
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
- **用途**: 注册定时工作流
- **返回**: Schedule 详情 + next_run_time
- **特性**: 基于 Temporal Schedules

### 3.2 列出 Schedules
```
GET /v1/schedules?status=active&limit=20&offset=0
```
- **用途**: 分页列出 Schedules
- **过滤**: 状态、工作流名称
- **返回**: Schedule 列表 + 分页信息

### 3.3 查询 Schedule 详情
```
GET /v1/schedules/{id}
```
- **用途**: 查询单个 Schedule
- **返回**: 完整 Schedule 信息 + 执行历史

### 3.4 暂停/恢复 Schedule
```
PATCH /v1/schedules/{id}
Content-Type: application/json

{
  "paused": true
}
```
- **用途**: 暂停或恢复 Schedule
- **返回**: 更新后的 Schedule 状态

### 3.5 删除 Schedule
```
DELETE /v1/schedules/{id}
```
- **用途**: 删除 Schedule
- **返回**: 204 No Content
- **行为**: 不影响正在运行的工作流

### 3.6 手动触发 Schedule
```
POST /v1/schedules/{id}/trigger
```
- **用途**: 立即触发一次工作流执行
- **返回**: 新工作流 ID
- **行为**: 不影响正常调度

---

## 4️⃣ Webhook 触发 API (6 个) - Story 1.11

### 4.1 注册 Webhook Trigger
```
POST /v1/triggers
Content-Type: application/json

{
  "name": "deploy-on-push",
  "workflow_name": "Deploy",
  "type": "webhook",
  "filters": {
    "branches": ["main"],
    "paths": ["src/**"]
  },
  "enabled": true,
  "secret": "my-webhook-secret"
}
```
- **用途**: 注册 Webhook 触发器
- **返回**: Webhook URL + Secret
- **特性**: 支持过滤规则

### 4.2 接收 Webhook 请求
```
POST /api/v1/webhooks/{trigger_id}
Content-Type: application/json
X-Hub-Signature-256: sha256=...

{
  "ref": "refs/heads/main",
  "commits": [...]
}
```
- **用途**: 接收外部 Webhook 事件
- **返回**: 触发的工作流 ID
- **特性**: HMAC 签名验证、异步处理

### 4.3 列出 Triggers
```
GET /v1/triggers?type=webhook&status=enabled
```
- **用途**: 分页列出 Triggers
- **过滤**: 类型、状态
- **返回**: Trigger 列表

### 4.4 查询 Trigger 详情
```
GET /v1/triggers/{id}
```
- **用途**: 查询单个 Trigger
- **返回**: 完整 Trigger 配置

### 4.5 更新 Trigger
```
PATCH /v1/triggers/{id}
Content-Type: application/json

{
  "enabled": false,
  "filters": {"branches": ["main", "dev"]}
}
```
- **用途**: 更新 Trigger 配置或状态
- **返回**: 更新后的 Trigger

### 4.6 删除 Trigger
```
DELETE /v1/triggers/{id}
```
- **用途**: 删除 Trigger
- **返回**: 204 No Content
- **行为**: Webhook URL 立即失效

**查询 Webhook 触发历史:**

通过统一的工作流查询 API:
```
GET /v1/workflows?trigger_type=webhook&trigger_source={trigger_id}
```
- **设计理念**: 避免数据冗余，保持单一数据源
- **元数据**: 工作流包含 trigger_type, trigger_source, trigger_event
- **优势**: 与 Schedule 设计对称，数据一致性更好

---

## 5️⃣ 模板管理 API (2 个) - Story 6.4

### 5.1 列出模板
```
GET /v1/templates
```
- **用途**: 获取所有工作流模板
- **返回**: 模板列表 (包含元数据)

### 5.2 获取模板详情
```
GET /v1/templates/{name}
```
- **用途**: 获取单个模板的 YAML 定义
- **返回**: 完整模板 YAML + 元数据

---

## 6️⃣ 节点管理 API (1 个) - Story 5.6

### 6.1 列出节点
```
GET /v1/nodes
GET /v1/nodes?format=yaml
```
- **用途**: 列出所有可用节点
- **返回**: 节点列表 (name, version, description)
- **格式**: JSON 或 YAML

---

## 7️⃣ YAML 验证 API (1 个) - Story 5.2

### 7.1 验证 YAML
```
POST /v1/validate
Content-Type: application/json

{
  "yaml": "name: Test\njobs:..."
}
```
- **用途**: 验证 YAML 工作流语法
- **返回**: 验证结果 + 错误详情
- **特性**: 离线验证,无需 Temporal

---

## 8️⃣ 审计日志 API (1 个) - Story 9.3

### 8.1 查询审计日志
```
GET /v1/audit?action=workflow.cancel&user=admin&start_time=2026-01-01T00:00:00Z
```
- **用途**: 查询系统审计日志
- **过滤**: 操作类型、用户、时间范围
- **返回**: 审计日志列表

---

## 9️⃣ Agent 管理 API (2 个) - Story 1.9

### 9.1 列出 Agents
```
GET /v1/agents
```
- **用途**: 获取所有可用 Agents
- **返回**: 
```json
{
  "agents": [
    {
      "name": "server-01",
      "status": "healthy",
      "pollers_count": 2
    },
    {
      "name": "server-02",
      "status": "degraded",
      "pollers_count": 1
    }
  ]
}
```
- **状态说明**:
  - `healthy`: 有 ≥2 个活跃 Poller
  - `degraded`: 有 1 个活跃 Poller
  - `unavailable`: 无活跃 Poller

### 9.2 查询 Agent 状态
```
GET /v1/agents/{name}
```
- **用途**: 查询单个 Agent 的健康状态
- **返回**:
```json
{
  "name": "server-01",
  "status": "healthy",
  "pollers_count": 3,
  "backlog_count": 5
}
```
- **场景**: 
  - 提交工作流前验证 `runs-on` 对应的 Agent 是否存在
  - Matrix 场景验证所有 `server` 值对应的 Agent 是否可用
  - 防止提交到不存在的 Agent 导致工作流永久等待

---

## 🚀 Post-MVP API (4 个)

### Story 1.9 扩展
```
POST /v1/workflows/{id}/archive          # 归档工作流 (P2)
GET  /v1/workflows/{id}/events           # 原始 Event History (P2)
POST /v1/workflows/{id}/pause            # 暂停工作流 (P3)
POST /v1/workflows/{id}/resume           # 恢复工作流 (P3)
```

---

## 📈 API 设计特点

### 统一规范
- ✅ RESTful 风格 (资源导向)
- ✅ 版本化 (/v1/)
- ✅ 统一错误格式 (RFC 7807)
- ✅ Request-ID 追踪
- ✅ CORS 支持

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

- [API 完整参考](./api-guide.md) - 每个端点的详细文档
- [OpenAPI 规范](../api/openapi.yaml) - Story 10.2
- [快速开始](./quick-start.md) - API 使用示例
- [工作流管理 API 调研](./sprint-artifacts/1-9-workflow-management-api-research.md) - 技术调研

---

**当前状态:** Epic 1 完成后,Waterflow 将提供 **29 个生产级 REST API** 🎉
