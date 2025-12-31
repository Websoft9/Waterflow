# http/request@v1

**分类**: http (网络通信)  
**版本**: v1  
**状态**: stable

## 概述

http/request 节点用于发送 HTTP/HTTPS 请求，与外部 REST API 集成。支持常见 HTTP 方法（GET、POST、PUT、DELETE、PATCH），自动处理 JSON，集成 Temporal 错误分类。

## 使用场景

- **REST API 调用**: 获取数据、提交表单、触发操作
- **Webhook 通知**: 发送部署完成通知到 Slack/钉钉
- **健康检查**: 检查服务 HTTP 端点可用性
- **数据同步**: 从外部系统拉取配置或状态
- **CMDB 集成**: 查询/更新资产信息

## 参数

| 参数名 | 类型 | 必需 | 默认值 | 描述 |
|--------|------|------|--------|------|
| url | string | ✅ | - | 请求 URL |
| method | string | ❌ | GET | HTTP 方法 (GET/POST/PUT/DELETE/PATCH) |
| headers | object | ❌ | {} | 请求头 |
| body | string/object | ❌ | - | 请求体 |
| timeout | string | ❌ | 30s | 请求超时 |
| verify_ssl | bool | ❌ | true | 验证 SSL 证书 |

### 参数详细说明

**url** (string, required)
- 目标 URL（必须包含协议 http:// 或 https://）
- 示例: `"https://api.example.com/users"`

**method** (string, optional, default: GET)
- HTTP 方法，大小写不敏感
- 支持: GET, POST, PUT, DELETE, PATCH
- 示例: `"POST"`, `"get"`

**headers** (object, optional)
- 自定义请求头
- 常用: Authorization, Content-Type, User-Agent
- 自动添加: User-Agent: Waterflow/1.0
- 示例: `{"Authorization": "Bearer token123"}`

**body** (string/object, optional)
- 请求体内容
- string: 直接发送
- object: 自动 JSON 序列化，并设置 Content-Type: application/json
- 示例: `{"name": "John", "age": 30}`

**timeout** (string, optional, default: 30s)
- 请求超时时间（最大 5 分钟）
- 格式: "30s", "1m", "2m"
- 示例: `"60s"`

**verify_ssl** (bool, optional, default: true)
- 是否验证 SSL 证书
- 生产环境建议 true
- 开发环境可设为 false（自签名证书）
- 示例: `false`

## 返回值

| 字段名 | 类型 | 描述 |
|--------|------|------|
| status_code | int | HTTP 状态码 |
| headers | object | 响应头 |
| body | string/object | 响应体（JSON自动解析） |
| content_type | string | Content-Type |
| elapsed_ms | int | 请求耗时 (毫秒) |

## 使用示例

### 示例 1: GET 请求

```yaml
steps:
  - name: Get user info
    uses: http/request@v1
    with:
      url: https://api.example.com/users/123
      method: GET
    id: user
  
  - name: Print user name
    uses: exec/shell@v1
    with:
      command: echo
      args: ["User: ${{ steps.user.outputs.body.name }}"]
```

### 示例 2: POST JSON 数据

```yaml
steps:
  - name: Create user
    uses: http/request@v1
    with:
      url: https://api.example.com/users
      method: POST
      headers:
        Authorization: Bearer ${{ secrets.API_TOKEN }}
      body:
        name: John Doe
        email: john@example.com
        role: admin
    id: create
  
  - name: Print created user ID
    uses: exec/shell@v1
    with:
      command: echo
      args: ["Created user ID: ${{ steps.create.outputs.body.id }}"]
```

### 示例 3: 自定义 Headers

```yaml
steps:
  - name: Call API with auth
    uses: http/request@v1
    with:
      url: https://api.example.com/data
      method: GET
      headers:
        Authorization: Bearer ${{ secrets.API_TOKEN }}
        X-Request-ID: req-12345
        Accept: application/json
```

### 示例 4: Webhook 通知

```yaml
steps:
  - name: Deploy application
    uses: docker/compose@v1
    with:
      action: up
  
  - name: Send Slack notification
    uses: http/request@v1
    with:
      url: ${{ secrets.SLACK_WEBHOOK_URL }}
      method: POST
      body:
        text: "Deployment completed successfully!"
        channel: "#deployments"
```

### 示例 5: 健康检查

```yaml
steps:
  - name: Check service health
    uses: http/request@v1
    with:
      url: http://localhost:8080/health
      method: GET
      timeout: 10s
    retry:
      max_attempts: 5
      initial_interval: 2s
```

### 示例 6: PUT 更新数据

```yaml
steps:
  - name: Update configuration
    uses: http/request@v1
    with:
      url: https://api.example.com/config/app
      method: PUT
      headers:
        Authorization: Bearer ${{ secrets.API_TOKEN }}
      body:
        enabled: true
        max_connections: 100
        timeout: 30
```

## 常见错误

### 错误 1: 网络超时

**现象:**
```
Error: context deadline exceeded
```

**解决方法:**
1. 增加超时时间: `timeout: 60s`
2. 检查网络连接
3. 确认目标服务可访问

### 错误 2: 4xx 客户端错误

**现象:**
```
status_code: 401
body: {"error": "Unauthorized"}
```

**原因**: 认证失败、权限不足、请求参数错误

**解决方法:**
1. 检查 Authorization header
2. 验证 API Token 有效性
3. 检查请求参数格式

### 错误 3: 5xx 服务器错误

**现象:**
```
status_code: 500
body: {"error": "Internal Server Error"}
```

**解决方法:**
1. 检查目标服务日志
2. 使用重试策略（5xx 错误可重试）
3. 联系 API 服务提供方

### 错误 4: SSL 验证失败

**现象:**
```
Error: x509: certificate signed by unknown authority
```

**解决方法:**
```yaml
# 开发环境临时方案
verify_ssl: false

# 生产环境: 安装正确的 CA 证书
```

### 错误 5: JSON 解析失败

**现象:**
```
Error: invalid character '<' looking for beginning of value
```

**原因**: 响应不是 JSON 格式（如 HTML 错误页面）

**解决方法:**
检查 `content_type` 和 status_code，确认 API 返回预期格式。

## 最佳实践

### 1. 使用 Secrets 存储 Token

```yaml
# ❌ 不安全
headers:
  Authorization: Bearer hardcoded_token

# ✅ 安全
headers:
  Authorization: Bearer ${{ secrets.API_TOKEN }}
```

### 2. 设置合理超时

```yaml
# 简单 API 调用
timeout: 30s  # 默认值

# 复杂查询
timeout: 2m

# 长时间操作
timeout: 5m  # 最大值
```

### 3. 检查响应状态码

```yaml
- name: Call API
  uses: http/request@v1
  with:
    url: https://api.example.com/data
  id: api

- name: Success path
  if: steps.api.outputs.status_code == 200
  uses: exec/shell@v1
  with:
    command: echo
    args: ["Success"]

- name: Error path
  if: steps.api.outputs.status_code != 200
  uses: exec/shell@v1
  with:
    command: echo
    args: ["Failed with: ${{ steps.api.outputs.status_code }}"]
```

### 4. 使用重试处理临时错误

```yaml
- name: Call API with retry
  uses: http/request@v1
  with:
    url: https://api.example.com/data
  retry:
    max_attempts: 3
    initial_interval: 1s
    backoff_coefficient: 2.0
```

### 5. 处理大型响应

```yaml
# 响应 <10MB: 正常使用
# 响应 >10MB: 使用专用下载工具或 file/transfer 节点
```

### 6. 正确设置 Content-Type

```yaml
# JSON: 自动处理
body:
  key: value
# 自动设置 Content-Type: application/json

# Form data: 手动设置
headers:
  Content-Type: application/x-www-form-urlencoded
body: "key1=value1&key2=value2"
```

## 错误分类

节点自动分类错误以决定是否重试：

| 错误类型 | 状态码 | 可重试 | 说明 |
|----------|--------|--------|------|
| **临时错误** | 5xx | ✅ | 服务器错误，可能恢复 |
| **临时错误** | - | ✅ | 网络错误、超时、DNS错误 |
| **永久错误** | 4xx | ❌ | 客户端错误，重试无效 |
| **永久错误** | - | ❌ | TLS 证书错误 |

## 安全注意事项

1. **Token 管理**: 使用 Secrets 存储 API Token
2. **SSL 验证**: 生产环境务必启用 verify_ssl
3. **日志脱敏**: Secrets 值在日志中自动隐藏
4. **最小权限**: API Token 使用最小必需权限

## 相关节点

- [file/transfer](../file/transfer.md) - 大文件传输场景
- [exec/shell](../exec/shell.md) - 使用 curl 调用 API

## 参考文档

- [Story 3.5: HTTP 请求节点](../../sprint-artifacts/3-5-http-request-node.md)

---

**文档版本**: v1  
**最后更新**: 2025-12-31  
**状态**: 稳定
