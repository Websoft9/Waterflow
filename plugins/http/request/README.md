# HTTP/Request Node

HTTP 请求节点 - 与外部 API 集成，支持常见 HTTP 方法和自动 JSON 处理。

## 功能描述

`http/request` 节点提供完整的 HTTP 客户端功能，支持 GET、POST、PUT、DELETE、PATCH 等方法，自动处理 JSON 序列化/反序列化，并集成 Temporal 错误分类以实现智能重试。

## 使用场景

1. **REST API 调用** - 获取数据、提交表单、触发外部操作
2. **Webhook 通知** - 发送部署完成通知到 Slack/钉钉/企业微信  
3. **健康检查** - 检查服务 HTTP 端点可用性
4. **数据同步** - 从外部系统拉取配置或状态
5. **CMDB 集成** - 查询/更新资产信息

## 参数

### 输入参数

| 参数名 | 类型 | 必需 | 默认值 | 描述 |
|--------|------|------|--------|------|
| `url` | string | 是 | - | 目标 URL (http:// or https://) |
| `method` | string | 否 | GET | HTTP 方法 (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS) |
| `headers` | object | 否 | {} | 请求头 (键值对) |
| `body` | string/object | 否 | - | 请求体 (字符串或 JSON 对象) |
| `timeout` | string | 否 | 30s | 请求超时 (如 "30s", "1m", 最大 5 分钟) |
| `verify_ssl` | bool | 否 | true | 验证 SSL 证书 |

### 输出参数

| 参数名 | 类型 | 描述 |
|--------|------|------|
| `status_code` | int | HTTP 状态码 |
| `headers` | object | 响应头 |
| `body` | string/object | 响应体 (JSON 自动解析) |
| `content_type` | string | Content-Type 头值 |
| `content_length` | int | 响应体字节数 |
| `elapsed_ms` | int | 请求耗时 (毫秒) |

## 使用示例

### 示例 1: GET 请求

```yaml
name: Fetch User Data
jobs:
  get-user:
    runs-on: linux-amd64
    steps:
      - name: Get user info
        uses: http/request@v1
        with:
          url: https://api.example.com/users/123
          method: GET
```

### 示例 2: POST JSON 数据

```yaml
- name: Create user
  uses: http/request@v1
  with:
    url: https://api.example.com/users
    method: POST
    headers:
      Content-Type: application/json
    body:
      name: John Doe
      email: john@example.com
```

### 示例 3: 自定义 Headers (Authorization)

```yaml
- name: Authenticated API call
  uses: http/request@v1
  with:
    url: https://api.example.com/protected
    headers:
      Authorization: Bearer ${secrets.API_TOKEN}
      Accept: application/json
```

### 示例 4: 错误处理

```yaml
- name: API call with retry
  uses: http/request@v1
  with:
    url: https://api.example.com/data
  continue-on-error: true
  retry:
    attempts: 3
    backoff: exponential
```

### 示例 5: 超时配置

```yaml
- name: Quick health check
  uses: http/request@v1
  with:
    url: http://localhost:8080/health
    timeout: 5s
```

### 示例 6: HTTPS 禁用证书验证 (测试环境)

```yaml
- name: Call self-signed API
  uses: http/request@v1
  with:
    url: https://internal-api.local/data
    verify_ssl: false
```

## 错误处理和重试

节点自动分类 HTTP 错误：

| 错误类型 | 状态码/条件 | Temporal 行为 | 说明 |
|----------|-------------|---------------|------|
| **永久错误** | 4xx 客户端错误 | 不重试 | 请求格式错误、认证失败、资源不存在 |
| **永久错误** | TLS 证书错误 | 不重试 | 证书验证失败 |
| **临时错误** | 5xx 服务器错误 | 可重试 | 服务器内部错误、网关超时 |
| **临时错误** | 网络错误 | 可重试 | 连接超时、DNS 解析失败 |

## 限制和注意事项

### HTTP 方法

- 支持：GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS
- 方法名称大小写不敏感 (`get` = `GET`)

### JSON 自动处理

- **请求体**: `body` 为对象时自动序列化为 JSON，并设置 `Content-Type: application/json`
- **响应体**: `Content-Type: application/json` 时自动解析为对象，其他保留为字符串

### 超时限制

- 默认超时：30 秒
- 最大超时：5 分钟
- 超时后返回临时错误，工作流可重试

### 响应大小

- 建议：< 10MB
- 大文件传输请使用专用的 `file/transfer` 节点

### SSL 验证

- 默认启用证书验证
- 仅在测试环境使用 `verify_ssl: false`
- 生产环境务必启用证书验证

## 编译和测试

```bash
# 编译插件
make build

# 运行测试
make test

# 查看覆盖率
make coverage

# 集成测试
make integration-test

# 清理
make clean

# 安装
make install
```

## 技术实现

- **HTTP 客户端**: Go 标准库 `net/http`
- **JSON 处理**: `encoding/json`
- **错误分类**: Temporal SDK `temporal.NewApplicationError`
- **SSL 配置**: `crypto/tls`

## 性能

- 请求延迟：取决于目标服务器
- 内存占用：~1-2MB (小响应)
- 并发安全：是

## 许可证

与 Waterflow 项目保持一致
