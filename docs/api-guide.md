# Waterflow API 使用指南

本指南介绍如何使用 Waterflow REST API 管理工作流。

## 快速开始

### 访问 API 文档

启动 Waterflow Server 后，访问交互式 API 文档：

```
http://localhost:8080/docs
```

Swagger UI 提供：
- ✅ 所有 API 端点列表和详细说明
- ✅ 在线测试功能（Try it out）
- ✅ 请求/响应示例
- ✅ 参数验证和类型说明

### OpenAPI 规范文件

获取完整的 OpenAPI 3.0 规范：

```bash
# YAML 格式
curl http://localhost:8080/api/openapi.yaml

# 或直接访问文件
cat api/openapi.yaml
```

## 核心 API 使用

### 1. 提交工作流

```bash
# 基础示例
curl -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "yaml": "name: hello-world\njobs:\n  greet:\n    runs-on: default\n    steps:\n      - uses: shell@v1\n        with:\n          command: echo\n          args: [\"Hello, World!\"]"
  }'

# 使用变量
curl -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "yaml": "name: deploy-app\njobs:\n  deploy:\n    runs-on: web-servers\n    steps:\n      - uses: shell@v1\n        with:\n          command: docker\n          args: [\"pull\", \"${{ vars.image }}\"]",
    "vars": {
      "image": "nginx:latest"
    }
  }'

# 响应示例
{
  "id": "wf_20260115_abc123",
  "run_id": "def456ghi789",
  "name": "hello-world",
  "status": "pending",
  "created_at": "2026-01-15T10:30:00Z",
  "url": "http://localhost:8088/namespaces/default/workflows/wf_20260115_abc123/def456ghi789"
}
```

### 2. 查询工作流状态

```bash
# 查询单个工作流
curl http://localhost:8080/v1/workflows/wf_20260115_abc123

# 响应示例
{
  "id": "wf_20260115_abc123",
  "name": "hello-world",
  "status": "completed",
  "created_at": "2026-01-15T10:30:00Z",
  "completed_at": "2026-01-15T10:30:05Z",
  "jobs": [
    {
      "name": "greet",
      "status": "completed",
      "steps": [
        {
          "name": "Say Hello",
          "status": "completed",
          "outputs": {
            "stdout": "Hello, World!\n"
          }
        }
      ]
    }
  ]
}
```

### 3. 列出工作流

```bash
# 列出所有工作流（分页）
curl "http://localhost:8080/v1/workflows?page=1&limit=20"

# 按状态过滤
curl "http://localhost:8080/v1/workflows?status=running"

# 按名称搜索
curl "http://localhost:8080/v1/workflows?name=deploy"

# 按时间范围过滤
curl "http://localhost:8080/v1/workflows?created_after=2026-01-15T00:00:00Z&created_before=2026-01-16T00:00:00Z"
```

### 4. 获取工作流日志

```bash
# 获取所有日志（NDJSON 格式）
curl http://localhost:8080/v1/workflows/wf_20260115_abc123/logs

# 按日志级别过滤
curl "http://localhost:8080/v1/workflows/wf_20260115_abc123/logs?level=error"

# 按 Job 过滤
curl "http://localhost:8080/v1/workflows/wf_20260115_abc123/logs?job=deploy"

# 只返回最后 100 条日志
curl "http://localhost:8080/v1/workflows/wf_20260115_abc123/logs?tail=100"

# 多条件组合
curl "http://localhost:8080/v1/workflows/wf_20260115_abc123/logs?job=deploy&step=build&level=info"
```

### 5. 取消工作流

```bash
curl -X POST http://localhost:8080/v1/workflows/wf_20260115_abc123/cancel

# 响应
{
  "message": "Workflow cancellation requested"
}
```

### 6. 重新运行工作流

```bash
curl -X POST http://localhost:8080/v1/workflows/wf_20260115_abc123/rerun

# 响应（新工作流 ID）
{
  "id": "wf_20260115_xyz789",
  "run_id": "new_run_id",
  "name": "hello-world",
  "status": "pending",
  "created_at": "2026-01-15T11:00:00Z"
}
```

## 工作流验证

在提交前验证 YAML 语法：

```bash
curl -X POST http://localhost:8080/v1/validate \
  -H "Content-Type: application/json" \
  -d '{
    "yaml": "name: test\njobs:\n  test:\n    runs-on: default\n    steps:\n      - uses: shell@v1\n        with:\n          command: echo test"
  }'

# 验证成功
{
  "valid": true,
  "message": "Validation successful"
}

# 验证失败（422 错误）
{
  "type": "validation_failed",
  "title": "Validation Failed",
  "status": 422,
  "detail": "Workflow validation failed",
  "errors": [
    {
      "field": "jobs.build.steps[0].uses",
      "message": "Unknown node: unknown@v1"
    }
  ]
}
```

## 模板管理

### 列出可用模板

```bash
curl http://localhost:8080/v1/templates

# 响应
{
  "templates": [
    {
      "name": "single-server-deployment",
      "title": "单服务器部署",
      "description": "将应用部署到单台服务器",
      "category": "deployment",
      "tags": ["deployment", "docker"]
    }
  ]
}
```

### 获取模板内容

```bash
curl http://localhost:8080/v1/templates/single-server-deployment

# 直接使用模板提交工作流
TEMPLATE=$(curl -s http://localhost:8080/v1/templates/single-server-deployment | jq -r '.yaml')
curl -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/json" \
  -d "{\"yaml\": \"$TEMPLATE\", \"vars\": {\"repo_url\": \"https://github.com/user/app.git\"}}"
```

## 节点查询

### 列出可用节点

```bash
curl http://localhost:8080/v1/nodes

# 响应
{
  "nodes": [
    {
      "name": "shell@v1",
      "category": "exec",
      "description": "执行 Shell 命令"
    },
    {
      "name": "http@v1",
      "category": "http",
      "description": "发送 HTTP 请求"
    }
  ]
}
```

## 审计日志

查询系统审计事件：

```bash
# 列出所有审计日志
curl "http://localhost:8080/v1/audit/logs?page=1&limit=50"

# 按事件类型过滤
curl "http://localhost:8080/v1/audit/logs?event_type=workflow.submit"

# 按时间范围过滤
curl "http://localhost:8080/v1/audit/logs?start_time=2026-01-15T00:00:00Z&end_time=2026-01-16T00:00:00Z"
```

## 健康检查和监控

```bash
# 基础健康检查
curl http://localhost:8080/health
# {"status":"ok"}

# 就绪检查（包含 Temporal 连接状态）
curl http://localhost:8080/ready
# {"status":"ready","checks":{"temporal":"ok"}}

# Prometheus 指标
curl http://localhost:8080/metrics

# 版本信息
curl http://localhost:8080/version
# {"version":"1.0.0","commit":"abc123","build_time":"2026-01-15T10:00:00Z"}
```

## 错误处理

Waterflow 遵循 RFC 7807 Problem Details 标准返回错误：

### 400 Bad Request - 参数错误

```json
{
  "type": "invalid_request",
  "title": "Invalid Request",
  "status": 400,
  "detail": "Missing required parameter: yaml",
  "instance": "/v1/workflows"
}
```

### 404 Not Found - 资源不存在

```json
{
  "type": "not_found",
  "title": "Not Found",
  "status": 404,
  "detail": "Workflow not found",
  "instance": "/v1/workflows/wf_invalid"
}
```

### 409 Conflict - 资源冲突

```json
{
  "type": "conflict",
  "title": "Conflict",
  "status": 409,
  "detail": "Workflow already cancelled",
  "instance": "/v1/workflows/wf_20260115_abc123/cancel"
}
```

### 422 Unprocessable Entity - 验证失败

```json
{
  "type": "validation_failed",
  "title": "Validation Failed",
  "status": 422,
  "detail": "Workflow validation failed",
  "errors": [
    {
      "field": "jobs.build.steps[0].uses",
      "message": "Unknown node: unknown@v1"
    }
  ]
}
```

### 500 Internal Server Error - 服务器错误

```json
{
  "type": "internal_error",
  "title": "Internal Server Error",
  "status": 500,
  "detail": "An unexpected error occurred",
  "instance": "/v1/workflows"
}
```

## 认证（可选）

### Bearer Token 认证

```bash
curl -X POST http://localhost:8080/v1/workflows \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"yaml": "..."}'
```

### API Key 认证

```bash
curl -X POST http://localhost:8080/v1/workflows \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"yaml": "..."}'
```

**注意**: 开发环境默认不需要认证。生产环境请参考 [安全配置指南](guides/security-guide.md)。

## 代码生成

使用 OpenAPI 规范生成客户端代码：

### Go 客户端

```bash
# 使用 openapi-generator
docker run --rm -v ${PWD}:/local openapitools/openapi-generator-cli generate \
  -i /local/api/openapi.yaml \
  -g go \
  -o /local/clients/go-generated

# 或使用 Waterflow 官方 Go SDK
go get github.com/Websoft9/waterflow/pkg/sdk
```

### Python 客户端

```bash
docker run --rm -v ${PWD}:/local openapitools/openapi-generator-cli generate \
  -i /local/api/openapi.yaml \
  -g python \
  -o /local/clients/python
```

### JavaScript/TypeScript 客户端

```bash
docker run --rm -v ${PWD}:/local openapitools/openapi-generator-cli generate \
  -i /local/api/openapi.yaml \
  -g typescript-axios \
  -o /local/clients/typescript
```

## 分页和过滤最佳实践

### 分页参数

- `page`: 页码（从 1 开始，默认 1）
- `limit`: 每页数量（1-100，默认 20）

### 过滤参数

工作流列表支持以下过滤条件：
- `status`: 按状态过滤（pending, running, completed, failed, cancelled）
- `name`: 按名称模糊搜索
- `created_after`: 创建时间起始（ISO 8601 格式）
- `created_before`: 创建时间结束（ISO 8601 格式）

### 响应元数据

所有分页接口返回 `meta` 字段：

```json
{
  "meta": {
    "total": 42,
    "page": 1,
    "limit": 20,
    "total_pages": 3
  }
}
```

## 下一步

- 📖 [YAML DSL 语法参考](yaml-dsl-reference.md) - 学习完整的工作流语法
- 🎨 [工作流模板库](templates/README.md) - 使用生产就绪的模板
- 🔒 [安全配置指南](guides/security-guide.md) - 生产环境安全部署
- 💻 [Go SDK 文档](../pkg/sdk/README.md) - 使用官方 SDK

## 相关资源

- [OpenAPI 规范文件](../api/openapi.yaml)
- [Swagger UI](http://localhost:8080/docs)
- [错误处理文档](../pkg/errors/README.md)
- [审计日志文档](../pkg/audit/README.md)
