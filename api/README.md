# api

API 定义和规范文件。

## 内容

- OpenAPI/Swagger 规范
- Protocol Buffers 定义 (如果使用 gRPC)
- API 文档

## Template API (Story 6.4)

### GET /v1/templates

列出所有可用的工作流模板。

**查询参数:**
- `category` (可选): 按类别过滤 (deployment, monitoring, automation)

**示例请求:**
```bash
# 获取所有模板
curl http://localhost:8080/v1/templates

# 按类别过滤
curl http://localhost:8080/v1/templates?category=deployment
```

**响应示例:**
```json
{
  "templates": [
    {
      "name": "single-server-deployment",
      "display_name": "Single Server Deployment",
      "description": "Deploy application to a single server with build, health check, and rollback support",
      "category": "deployment",
      "version": "1.0.0",
      "author": "Waterflow Team",
      "parameters": [
        {
          "name": "repo_url",
          "type": "string",
          "required": true,
          "description": "Git repository URL",
          "example": "https://github.com/user/myapp.git"
        }
      ],
      "examples": [
        {
          "name": "Node.js Application",
          "description": "Deploy a Node.js Express application",
          "vars": {
            "repo_url": "https://github.com/user/nodejs-app.git",
            "app_name": "nodejs-api",
            "app_port": 3000
          }
        }
      ]
    }
  ],
  "count": 3
}
```

### GET /v1/templates/{name}

获取特定模板的详细信息和内容。

**路径参数:**
- `name`: 模板名称 (例如: single-server-deployment)

**查询参数:**
- `content` (可选): 是否包含YAML内容 (默认: true, 设为false仅返回元数据)

**示例请求:**
```bash
# 获取完整模板 (含YAML内容)
curl http://localhost:8080/v1/templates/single-server-deployment

# 仅获取元数据
curl http://localhost:8080/v1/templates/single-server-deployment?content=false
```

**响应示例:**
```json
{
  "name": "single-server-deployment",
  "display_name": "Single Server Deployment",
  "description": "Deploy application to a single server with build, health check, and rollback support",
  "category": "deployment",
  "version": "1.0.0",
  "parameters": [...],
  "examples": [...],
  "content": "name: Single Server Deployment\non: push\nvars:\n  repo_url: \"https://github.com/user/app.git\"\n  app_name: \"my-app\"\n..."
}
```

**错误响应:**
- `404 Not Found`: 模板不存在
- `400 Bad Request`: 无效的模板名称
- `413 Payload Too Large`: 模板文件超过1MB
- `500 Internal Server Error`: 服务器内部错误

所有错误响应遵循 RFC 7807 Problem Details 格式:

```json
{
  "type": "https://waterflow.io/errors/template-not-found",
  "title": "Template Not Found",
  "status": 404,
  "detail": "Template 'invalid-template' does not exist",
  "instance": "/v1/templates/invalid-template"
}
```
- `400 Bad Request`: 无效的模板名称
- `413 Payload Too Large`: 模板文件超过1MB

## 未来计划

Story 1.2 将添加 REST API 的 OpenAPI 规范。
