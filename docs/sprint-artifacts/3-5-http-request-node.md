# Story 3.5: HTTP 请求节点 (http/request)

**状态:** done  
**Epic:** 3 - 核心节点插件库  
**Story ID:** 3.5  
**创建日期:** 2025-12-30  
**完成日期:** 2025-12-30  
**代码审查:** 已完成 (2025-12-30)

---

## 核心设计决策 (必读!)

**关键架构对齐:**
- **包路径**: 使用 `github.com/websoft9/waterflow/pkg/dsl/node` (NOT `pkg/node/`)
- **接口实现**: 必须实现 5 个方法: Name, Version, Params, Execute, Metadata
- **返回类型**: Execute 返回 `*node.NodeResult` (NOT `map[string]interface{}`)
- **NodeResult 结构**: `{Outputs: map, Logs: []string, Duration: time.Duration, Metadata: map}`

**HTTP 节点特性:**
- **分类**: http (网络通信) - 与 exec/flow 类节点不同
- **核心功能**: HTTP/HTTPS 请求 + Temporal 错误分类 + 自动 JSON 处理
- **安全**: 默认验证 SSL 证书，支持自定义 headers (Authorization)
- **错误分类**: 4xx 永久错误(不重试)，5xx/网络错误 临时错误(可重试)
- **响应限制**: 建议 < 10MB，大文件使用专用 file/transfer 节点

**Temporal 集成:**
- 使用 `temporal.NewApplicationError` 创建错误
- 4xx 错误标记 `NonRetryable`
- 5xx/网络错误默认可重试
- 超时错误自动分类为临时错误

**依赖关系:**
- 继承 Story 3.1 的 Node 接口定义
- 参考 Story 3.2 的错误分类模式
- 使用 Go 标准库 net/http (无外部 HTTP 库)
- 集成 Temporal SDK 错误类型

---

## Story

As a **工作流用户**,  
I want **发送 HTTP 请求**,  
So that **与外部 API 集成**。

---

## Acceptance Criteria

**AC1: HTTP 方法支持**  
**Given** 工作流需要调用 HTTP API  
**When** Step 使用 `http/request` 节点  
**Then** 支持 GET、POST、PUT、DELETE、PATCH 方法  
**And** 方法参数大小写不敏感  
**And** 默认方法为 GET  

**AC2: 请求配置**  
**Given** 需要配置 HTTP 请求  
**When** 配置请求参数  
**Then** 支持参数: url (必填)、method、headers、body、timeout  
**And** headers 支持多个键值对  
**And** body 支持字符串或 JSON 对象  
**And** timeout 默认 30 秒，最大 5 分钟  
**And** 自动设置 User-Agent: Waterflow/1.0  

**AC3: 响应处理**  
**Given** HTTP 请求已发送  
**When** 接收到响应  
**Then** 返回 status_code、headers、body  
**And** 2xx 状态码视为成功  
**And** 4xx/5xx 状态码返回错误  
**And** body 自动解析 JSON (Content-Type: application/json)  
**And** 其他 Content-Type 返回原始字符串  
**And** 超时返回 Temporal 可重试错误  

**AC4: 安全和错误处理**  
**Given** 请求可能失败  
**When** 处理错误场景  
**Then** 网络错误、DNS 错误返回可重试错误  
**And** 4xx 客户端错误返回永久错误  
**And** 5xx 服务器错误返回可重试错误  
**And** 证书验证失败返回永久错误  
**And** 支持 HTTPS  

---

## Tasks / Subtasks

### Task 1: 创建 HTTP 请求节点插件目录结构 (AC1, AC2)
- [x] 创建 `plugins/http/request/` 目录
- [x] 创建 `plugins/http/request/main.go` - 插件主文件
- [x] 创建 `plugins/http/request/main_test.go` - 单元测试
- [x] 创建 `plugins/http/request/Makefile` - 编译脚本
- [x] 创建 `plugins/http/request/README.md` - 节点文档

### Task 2: 实现 HTTP 请求节点接口 (AC1, AC2)
- [x] 定义 HTTPRequestNode 结构体
  - [x] 实现 `Name() string` - 返回 "http/request"
  - [x] 实现 `Version() string` - 返回 "v1"
  - [x] 实现 `Metadata() node.NodeMetadata` - 返回节点元数据
  - [x] 实现 `Execute(ctx, inputs) (outputs, error)` - 执行 HTTP 请求
- [x] 定义输入参数 Schema (AC2)
  - [x] url (string, required) - 请求 URL
  - [x] method (string, optional, default: GET) - HTTP 方法
  - [x] headers (map[string]string, optional) - 请求头
  - [x] body (string/object, optional) - 请求体
  - [x] timeout (string, optional, default: "30s") - 超时时间
  - [x] verify_ssl (bool, optional, default: true) - 验证 SSL 证书
- [x] 定义输出结构 (AC3)
  - [x] status_code (int) - HTTP 状态码
  - [x] headers (map[string][]string) - 响应头
  - [x] body (string/object) - 响应体 (自动解析 JSON)
  - [x] content_type (string) - Content-Type
  - [x] content_length (int) - Content-Length
  - [x] elapsed_ms (int) - 请求耗时 (毫秒)
- [x] 实现 Register() 函数

### Task 3: 实现 HTTP 客户端逻辑 (AC1, AC2, AC3)
- [x] 创建 HTTP Client
  - [x] 使用 `http.Client` 配置 timeout
  - [x] 支持 context 取消
  - [x] 配置 TLS (verify_ssl)
  - [x] 设置默认 User-Agent
- [x] 构建请求
  - [x] 验证 URL 格式
  - [x] 规范化 method (转大写)
  - [x] 设置 headers
  - [x] 序列化 body (JSON 对象 → string)
  - [x] 创建 `http.Request`
- [x] 发送请求
  - [x] 调用 `client.Do(req)`
  - [x] 捕获网络错误
  - [x] 记录开始时间
- [x] 处理响应 (AC3)
  - [x] 读取 response body
  - [x] 根据 Content-Type 解析 body
  - [x] 提取 status_code、headers
  - [x] 计算 elapsed_ms
  - [x] 关闭 response.Body

### Task 4: 实现错误分类逻辑 (AC4)
### Task 4: 实现错误分类 (AC4)
- [x] 创建 `classifyError(statusCode, err)` 函数
  - [x] 网络错误: TemporaryError (可重试)
  - [x] DNS 错误: TemporaryError (可重试)
  - [x] Timeout: TemporaryError (可重试)
  - [x] 4xx 状态码: PermanentError (不可重试)
  - [x] 5xx 状态码: TemporaryError (可重试)
  - [x] TLS 证书错误: PermanentError (不可重试)
- [x] 集成 Temporal 错误类型
  - [x] 使用 `temporal.NewApplicationError`
  - [x] 设置 `NonRetryable` 标志

### Task 5: 实现请求体处理 (AC2)
- [x] 支持 string body
  - [x] 直接使用原始字符串
  - [x] 如未设置 Content-Type，默认 text/plain
- [x] 支持 JSON object body
  - [x] 检测 map[string]interface{} 类型
  - [x] 序列化为 JSON
  - [x] 自动设置 Content-Type: application/json
- [x] 参数验证
  - [x] GET/DELETE 请求不应有 body (警告)
  - [x] POST/PUT/PATCH 通常需要 body

### Task 6: 编写单元测试
- [x] 测试基本 GET 请求
  - [x] 模拟 HTTP 服务器 (httptest)
  - [x] 验证 URL、method、headers
  - [x] 验证响应解析
- [x] 测试 POST 请求
  - [x] JSON body 序列化
  - [x] Content-Type 自动设置
  - [x] 验证 body 接收
- [x] 测试所有 HTTP 方法
  - [x] GET, POST, PUT, DELETE, PATCH
  - [x] 方法大小写不敏感
- [x] 测试 headers 设置
  - [x] 自定义 headers
  - [x] User-Agent 默认值
  - [x] Authorization header
- [x] 测试响应解析
  - [x] JSON 自动解析
  - [x] 非 JSON 返回字符串
  - [x] 空 body 处理
- [x] 测试状态码处理
  - [x] 2xx 成功
  - [x] 4xx 客户端错误
  - [x] 5xx 服务器错误
- [x] 测试超时
  - [x] 设置短超时
  - [x] 模拟慢响应
  - [x] 验证超时错误分类
- [x] 测试 context 取消
  - [x] 取消正在进行的请求
  - [x] 验证立即返回
- [x] 测试错误分类
  - [x] 网络错误 → TemporaryError
  - [x] 4xx → PermanentError
  - [x] 5xx → TemporaryError
- [x] 测试 SSL 验证
  - [x] verify_ssl=true (默认)
  - [x] verify_ssl=false
- [x] 测试覆盖率目标 >90% (实际: 92.3%)

### Task 7: 实现 Makefile 和编译脚本
- [x] 创建 Makefile 目标
  - [x] `make build` - 编译插件为 request.so
  - [x] `make test` - 运行单元测试
  - [x] `make clean` - 清理构建产物
  - [x] `make install` - 安装到插件目录
- [x] 添加依赖检查
  - [x] 检查 Go 版本 >= 1.22
  - [x] 检查 CGO_ENABLED=1

### Task 8: 编写节点文档
- [x] 创建 README.md
  - [x] 节点描述和使用场景
  - [x] 参数详细说明
  - [x] HTTP 方法说明
  - [x] Headers 和 Body 格式
  - [x] 状态码处理规则
  - [x] 至少 6 个使用示例
    - [x] GET 请求
    - [x] POST JSON 数据
    - [x] 自定义 Headers (Authorization)
    - [x] 错误处理
    - [x] 超时配置
    - [x] HTTPS 请求
  - [x] 常见 API 集成场景
- [x] 添加 YAML 示例

### Task 9: 集成测试
- [x] 创建 `plugins/http/request/integration_test.go`
- [x] 测试插件加载
  - [x] 编译为 .so 文件
  - [x] 使用 plugin.Open 加载
  - [x] 调用 Register 获取节点
- [x] 测试真实 HTTP 请求
  - [x] 使用 httptest.Server
  - [x] 验证完整请求-响应流程
- [x] 测试与 NodeRegistry 集成

---

## Developer Context

### 架构背景

**依赖 Stories:**
- **Story 3.1: 节点接口设计** - Node 接口定义
- **Story 3.2: Shell 命令执行节点** - 错误分类参考
- **Story 3.4: 延迟等待节点** - Timeout 处理参考

**节点定位:**
- **分类**: http (网络通信)
- **用途**: 与外部 HTTP/REST API 集成，实现服务间通信
- **特点**: 支持常见 HTTP 方法，自动 JSON 处理，错误分类

**典型使用场景:**
1. **调用 REST API**: 获取数据、提交表单、触发操作
2. **Webhook 通知**: 发送部署完成通知到 Slack/钉钉
3. **健康检查**: 检查服务 HTTP 端点可用性
4. **数据同步**: 从外部系统拉取配置或状态
5. **CMDB 集成**: 查询/更新资产信息

### 技术栈和依赖

**Go 标准库:**
```go
import (
    "context"
    "net/http"        // HTTP 客户端
    "net/url"         // URL 解析和验证
    "time"            // 超时控制
    "io"              // 读取响应体
    "bytes"           // 请求体缓冲
    "encoding/json"   // JSON 序列化/反序列化
    "strings"         // 字符串处理
    "fmt"             // 格式化
    "crypto/tls"      // TLS 配置
)
```

**项目依赖:**
```go
import (
    "github.com/websoft9/waterflow/pkg/dsl/node"  // Story 3.1 的 Node 接口
    "go.temporal.io/sdk/temporal"  // Temporal 错误类型
)
```

**无外部 HTTP 库** - 使用 Go 标准 net/http

### 核心实现参考

#### HTTP 请求节点结构
```go
// plugins/http/request/main.go
package main

import (
    "bytes"
    "context"
    "crypto/tls"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
    "time"
    
    "go.temporal.io/sdk/temporal"
    "github.com/websoft9/waterflow/pkg/dsl/node"
)

// HTTPRequestNode 实现 HTTP 请求节点
type HTTPRequestNode struct{}

// Name 返回节点名称
func (n *HTTPRequestNode) Name() string {
    return "http/request"
}

// Version 返回节点版本
func (n *HTTPRequestNode) Version() string {
    return "v1"
}

// Params 返回参数规格
func (n *HTTPRequestNode) Params() map[string]node.ParamSpec {
    return n.Metadata().InputSchema
}

// Metadata 返回节点元数据
func (n *HTTPRequestNode) Metadata() node.NodeMetadata {
    return node.NodeMetadata{
        Description: "Send HTTP requests to external APIs",
        Category:    "http",
        InputSchema: map[string]node.ParamSpec{
            "url": {
                Type:        "string",
                Required:    true,
                Description: "Target URL",
            },
            "method": {
                Type:        "string",
                Required:    false,
                Default:     "GET",
                Description: "HTTP method (GET, POST, PUT, DELETE, PATCH)",
            },
            "headers": {
                Type:        "object",
                Required:    false,
                Description: "Request headers (key-value pairs)",
            },
            "body": {
                Type:        "string|object",
                Required:    false,
                Description: "Request body (string or JSON object)",
            },
            "timeout": {
                Type:        "string",
                Required:    false,
                Default:     "30s",
                Description: "Request timeout (e.g., '30s', '1m')",
            },
            "verify_ssl": {
                Type:        "bool",
                Required:    false,
                Default:     true,
                Description: "Verify SSL certificates",
            },
        },
        OutputSchema: map[string]interface{}{
            "status_code":    "int",
            "headers":        "object",
            "body":           "string|object",
            "content_type":   "string",
            "content_length": "int",
            "elapsed_ms":     "int",
        },
    }
}

// Execute 执行 HTTP 请求
func (n *HTTPRequestNode) Execute(
    ctx context.Context,
    inputs map[string]interface{},
) (*node.NodeResult, error) {
    startTime := time.Now()
    
    // 1. 解析参数
    urlStr, ok := inputs["url"].(string)
    if !ok || urlStr == "" {
        return nil, temporal.NewApplicationError("url is required", "InvalidParameter")
    }
    
    // 验证 URL 格式
    parsedURL, err := url.Parse(urlStr)
    if err != nil {
        return nil, temporal.NewApplicationError(
            fmt.Sprintf("invalid url: %v", err),
            "InvalidParameter",
        )
    }
    if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
        return nil, temporal.NewApplicationError(
            "url must use http or https scheme",
            "InvalidParameter",
        )
    }
    
    // 2. 获取 HTTP 方法
    method := "GET"
    if m, ok := inputs["method"].(string); ok && m != "" {
        method = strings.ToUpper(m)
    }
    
    // 验证方法
    validMethods := map[string]bool{
        "GET": true, "POST": true, "PUT": true,
        "DELETE": true, "PATCH": true, "HEAD": true,
        "OPTIONS": true,
    }
    if !validMethods[method] {
        return nil, temporal.NewApplicationError(
            fmt.Sprintf("invalid HTTP method: %s", method),
            "InvalidParameter",
        )
    }
    
    // 3. 准备请求体
    var bodyReader io.Reader
    var contentType string
    
    if body, ok := inputs["body"]; ok && body != nil {
        switch v := body.(type) {
        case string:
            bodyReader = strings.NewReader(v)
            contentType = "text/plain"
        case map[string]interface{}:
            // JSON 对象
            jsonData, err := json.Marshal(v)
            if err != nil {
                return nil, temporal.NewApplicationError(
                    fmt.Sprintf("failed to marshal body: %v", err),
                    "InvalidParameter",
                )
            }
            bodyReader = bytes.NewReader(jsonData)
            contentType = "application/json"
        default:
            return nil, temporal.NewApplicationError(
                "body must be string or object",
                "InvalidParameter",
            )
        }
    }
    
    // 4. 创建请求
    req, err := http.NewRequestWithContext(ctx, method, urlStr, bodyReader)
    if err != nil {
        return nil, temporal.NewApplicationError(
            fmt.Sprintf("failed to create request: %v", err),
            "RequestCreationFailed",
        )
    }
    
    // 5. 设置 Headers
    req.Header.Set("User-Agent", "Waterflow/1.0")
    if contentType != "" {
        req.Header.Set("Content-Type", contentType)
    }
    
    if headers, ok := inputs["headers"].(map[string]interface{}); ok {
        for k, v := range headers {
            if strVal, ok := v.(string); ok {
                req.Header.Set(k, strVal)
            }
        }
    }
    
    // 6. 解析超时
    timeout := 30 * time.Second
    if timeoutStr, ok := inputs["timeout"].(string); ok && timeoutStr != "" {
        if d, err := time.ParseDuration(timeoutStr); err == nil {
            timeout = d
        }
    }
    
    // 验证超时范围
    if timeout > 5*time.Minute {
        return nil, temporal.NewApplicationError(
            "timeout must not exceed 5 minutes",
            "InvalidParameter",
        )
    }
    
    // 7. 配置 HTTP Client
    verifySsl := true
    if v, ok := inputs["verify_ssl"].(bool); ok {
        verifySsl = v
    }
    
    client := &http.Client{
        Timeout: timeout,
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{
                InsecureSkipVerify: !verifySsl,
            },
        },
    }
    
    // 8. 发送请求
    resp, err := client.Do(req)
    if err != nil {
        return nil, classifyHTTPError(err, 0)
    }
    defer resp.Body.Close()
    
    // 9. 读取响应体
    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, temporal.NewApplicationError(
            fmt.Sprintf("failed to read response: %v", err),
            "ResponseReadFailed",
        )
    }
    
    elapsed := time.Since(startTime)
    
    // 10. 解析响应体
    var bodyOutput interface{} = string(respBody)
    respContentType := resp.Header.Get("Content-Type")
    
    if strings.Contains(respContentType, "application/json") {
        var jsonBody interface{}
        if err := json.Unmarshal(respBody, &jsonBody); err == nil {
            bodyOutput = jsonBody
        }
    }
    
    // 11. 检查状态码
    if resp.StatusCode >= 400 {
        return nil, classifyHTTPError(
            fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody)),
            resp.StatusCode,
        )
    }
    
    // 12. 返回结果
    return &node.NodeResult{
        Outputs: map[string]interface{}{
            "status_code":    resp.StatusCode,
            "headers":        resp.Header,
            "body":           bodyOutput,
            "content_type":   respContentType,
            "content_length": len(respBody),
            "elapsed_ms":     elapsed.Milliseconds(),
        },
        Logs: []string{
            fmt.Sprintf("HTTP %s %s", method, urlStr),
            fmt.Sprintf("Response: %d in %dms", resp.StatusCode, elapsed.Milliseconds()),
        },
        Duration: elapsed,
    }, nil
}

// classifyHTTPError 分类 HTTP 错误
// 4xx: PermanentError (不可重试)
// 5xx: TemporaryError (可重试)
// 网络/超时: TemporaryError (可重试)
func classifyHTTPError(err error, statusCode int) error {
    errMsg := err.Error()
    
    // 4xx 客户端错误 - 永久错误
    if statusCode >= 400 && statusCode < 500 {
        return temporal.NewApplicationError(
            errMsg,
            "HTTPClientError",
            temporal.WithNonRetryable(),
        )
    }
    
    // 5xx 服务器错误 - 临时错误 (可重试)
    if statusCode >= 500 {
        return temporal.NewApplicationError(
            errMsg,
            "HTTPServerError",
        )
    }
    
    // TLS 证书错误 - 永久错误
    if strings.Contains(errMsg, "certificate") ||
       strings.Contains(errMsg, "tls") {
        return temporal.NewApplicationError(
            errMsg,
            "TLSError",
            temporal.WithNonRetryable(),
        )
    }
    
    // 网络错误、超时 - 临时错误 (可重试)
    return temporal.NewApplicationError(
        errMsg,
        "NetworkError",
    )
}

// Register 是 Go Plugin 导出的注册函数
func Register() node.Node {
    return &HTTPRequestNode{}
}
```

### 测试策略

#### 单元测试示例
```go
// plugins/http/request/main_test.go
package main

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestHTTPRequestNode_Name(t *testing.T) {
    node := &HTTPRequestNode{}
    assert.Equal(t, "http/request", node.Name())
}

func TestHTTPRequestNode_Version(t *testing.T) {
    node := &HTTPRequestNode{}
    assert.Equal(t, "v1", node.Version())
}

func TestHTTPRequestNode_Execute_GET(t *testing.T) {
    // 创建测试服务器
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "GET", r.Method)
        assert.Equal(t, "Waterflow/1.0", r.Header.Get("User-Agent"))
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{
            "message": "success",
        })
    }))
    defer server.Close()
    
    node := &HTTPRequestNode{}
    inputs := map[string]interface{}{
        "url":    server.URL,
        "method": "GET",
    }
    
    result, err := node.Execute(context.Background(), inputs)
    
    require.NoError(t, err)
    assert.Equal(t, 200, result.Outputs["status_code"])
    
    body := result.Outputs["body"].(map[string]interface{})
    assert.Equal(t, "success", body["message"])
    assert.Greater(t, result.Duration.Milliseconds(), int64(0))
}

func TestHTTPRequestNode_Execute_POST_JSON(t *testing.T) {
    // 创建测试服务器
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "POST", r.Method)
        assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
        
        var body map[string]interface{}
        json.NewDecoder(r.Body).Decode(&body)
        assert.Equal(t, "test", body["key"])
        
        w.WriteHeader(http.StatusCreated)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "created",
        })
    }))
    defer server.Close()
    
    node := &HTTPRequestNode{}
    inputs := map[string]interface{}{
        "url":    server.URL,
        "method": "POST",
        "body": map[string]interface{}{
            "key": "test",
        },
    }
    
    result, err := node.Execute(context.Background(), inputs)
    
    require.NoError(t, err)
    assert.Equal(t, 201, result.Outputs["status_code"])
}

func TestHTTPRequestNode_Execute_CustomHeaders(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
        assert.Equal(t, "application/json", r.Header.Get("Accept"))
        
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()
    
    node := &HTTPRequestNode{}
    inputs := map[string]interface{}{
        "url": server.URL,
        "headers": map[string]interface{}{
            "Authorization": "Bearer token123",
            "Accept":        "application/json",
        },
    }
    
    outputs, err := node.Execute(context.Background(), inputs)
    
    require.NoError(t, err)
    assert.Equal(t, 200, outputs["status_code"])
}

func TestHTTPRequestNode_Execute_Timeout(t *testing.T) {
    // 创建慢响应服务器
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        time.Sleep(2 * time.Second)
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()
    
    node := &HTTPRequestNode{}
    inputs := map[string]interface{}{
        "url":     server.URL,
        "timeout": "100ms",
    }
    
    _, err := node.Execute(context.Background(), inputs)
    
    require.Error(t, err)
    assert.Contains(t, err.Error(), "timeout")
}

func TestHTTPRequestNode_Execute_4xxError(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusNotFound)
        w.Write([]byte("Not Found"))
    }))
    defer server.Close()
    
    node := &HTTPRequestNode{}
    inputs := map[string]interface{}{
        "url": server.URL,
    }
    
    _, err := node.Execute(context.Background(), inputs)
    
    require.Error(t, err)
    assert.Contains(t, err.Error(), "404")
    
    // 验证是永久错误 (不可重试)
    // Temporal error 应该标记为 NonRetryable
}

func TestHTTPRequestNode_Execute_5xxError(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusInternalServerError)
        w.Write([]byte("Server Error"))
    }))
    defer server.Close()
    
    node := &HTTPRequestNode{}
    inputs := map[string]interface{}{
        "url": server.URL,
    }
    
    _, err := node.Execute(context.Background(), inputs)
    
    require.Error(t, err)
    assert.Contains(t, err.Error(), "500")
    
    // 5xx 应该是可重试错误
}

func TestHTTPRequestNode_Execute_MethodCaseInsensitive(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "POST", r.Method)
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()
    
    node := &HTTPRequestNode{}
    
    tests := []string{"post", "Post", "POST", "PoSt"}
    for _, method := range tests {
        inputs := map[string]interface{}{
            "url":    server.URL,
            "method": method,
        }
        
        _, err := node.Execute(context.Background(), inputs)
        require.NoError(t, err)
    }
}

func TestHTTPRequestNode_Execute_InvalidURL(t *testing.T) {
    node := &HTTPRequestNode{}
    
    tests := []struct {
        name string
        url  string
    }{
        {"missing scheme", "example.com"},
        {"invalid scheme", "ftp://example.com"},
        {"malformed", "http://[invalid"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            inputs := map[string]interface{}{
                "url": tt.url,
            }
            
            _, err := node.Execute(context.Background(), inputs)
            require.Error(t, err)
        })
    }
}

// ... 更多测试用例见完整测试文件
// 覆盖: String body, All HTTP methods, Context cancellation, Response parsing (JSON/text)
```

### YAML 工作流示例

```yaml
# examples/workflows/http-examples.yaml
name: HTTP Request Node Examples

jobs:
  common-scenarios:
    name: Common HTTP Scenarios
    runs-on: linux-amd64
    steps:
      # 基础 GET 请求
      - name: Simple GET request
        uses: http/request@v1
        with:
          url: https://api.github.com/repos/waterflow/waterflow
          method: GET
        id: github-info
      
      # GET with headers
      - name: GET with Authorization
        uses: http/request@v1
        with:
          url: https://api.example.com/data
          headers:
            Authorization: Bearer ${{ secrets.API_TOKEN }}
      
      # POST JSON
      - name: POST JSON data
        uses: http/request@v1
        with:
          url: https://api.example.com/users
          method: POST
          body:
            name: John Doe
            email: john@example.com
      
      # Webhook 通知
      - name: Send Slack notification
        uses: http/request@v1
        with:
          url: ${{ secrets.SLACK_WEBHOOK_URL }}
          method: POST
          body:
            text: "Deployment completed!"
      
      # 健康检查
      - name: Health check
        uses: http/request@v1
        with:
          url: https://api.example.com/health
          timeout: 5s
        continue-on-error: true
      
      # API 重试
      - name: Call with retry
        uses: http/request@v1
        with:
          url: https://api.example.com/data
          timeout: 10s
        retry:
          max_attempts: 3
          backoff_coefficient: 2.0
      
      # 多步骤 API 集成
      - name: Authenticate
        uses: http/request@v1
        with:
          url: https://auth.example.com/oauth/token
          method: POST
          body:
            grant_type: client_credentials
            client_id: ${{ secrets.CLIENT_ID }}
        id: auth
      
      - name: Use token
        uses: http/request@v1
        with:
          url: https://api.example.com/users/me
          headers:
            Authorization: Bearer ${{ steps.auth.outputs.body.access_token }}
```

**关键使用场景**: REST API调用、Webhook通知、健康检查、CMDB集成、OAuth认证流程、服务间通信

### 性能和资源限制

**关键限制 - 响应体大小**:
- **建议上限**: < 10MB
- **原因**: 响应体完全加载到内存,大文件会导致内存压力
- **大文件场景**: 使用专用 file/transfer 节点(支持流式下载)
- **JSON 解析**: 自动处理,但大 JSON (>1MB) 建议预先评估

**HTTP Client 配置**:
- **每请求独立 Client**: 支持每请求不同的 timeout/SSL 配置
- **连接复用**: Go http.Transport 自动管理 Keep-Alive
- **HTTP/2**: 自动启用(HTTPS)

**超时配置**:
- 默认 30s,范围 1s-5m
- API 调用建议 10-30s,健康检查 5s

### 错误分类

| 错误类型 | 状态码 | 分类 | 可重试 | 说明 |
|---------|--------|------|--------|------|
| 客户端错误 | 400-499 | PermanentError | ❌ | 请求参数错误 |
| 服务器错误 | 500-599 | TemporaryError | ✅ | 服务端临时故障 |
| 网络错误 | - | TemporaryError | ✅ | DNS、连接失败 |
| 超时 | - | TemporaryError | ✅ | 请求超时 |
| TLS 错误 | - | PermanentError | ❌ | 证书验证失败 |

### 文件结构

```
Waterflow/
├── plugins/
│   ├── exec/
│   │   ├── shell/                    # [EXISTS] Story 3.2
│   │   └── script/                   # [EXISTS] Story 3.3
│   ├── flow/
│   │   └── sleep/                    # [EXISTS] Story 3.4
│   └── http/
│       └── request/                  # [NEW] HTTP 请求节点插件
│           ├── main.go               # [NEW] 插件实现 (~350 行)
│           ├── main_test.go          # [NEW] 单元测试 (~400 行)
│           ├── integration_test.go   # [NEW] 集成测试 (~100 行)
│           ├── Makefile              # [NEW] 编译脚本
│           └── README.md             # [NEW] 节点文档
├── pkg/
│   └── dsl/
│       └── node/
│           ├── interface.go          # [EXISTS] Story 3.1
│           └── metadata.go           # [EXISTS] Story 3.1
└── examples/
    └── workflows/
        ├── shell-examples.yaml       # [EXISTS] Story 3.2
        ├── script-examples.yaml      # [EXISTS] Story 3.3
        ├── sleep-examples.yaml       # [EXISTS] Story 3.4
        └── http-examples.yaml        # [NEW] HTTP 节点示例
```

### Makefile 示例

```makefile
# plugins/http/request/Makefile
.PHONY: build test clean install

PLUGIN_NAME = request.so

build:
	@echo "Checking Go version..."
	@go version | grep -q "go1.2[2-9]" || (echo "Error: Go 1.22+ required" && exit 1)
	@echo "Building plugin..."
	CGO_ENABLED=1 go build -buildmode=plugin -o $(PLUGIN_NAME) main.go
	@echo "✓ Plugin built: $(PLUGIN_NAME)"

test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out
	@echo "\n=== Coverage Report ==="
	go tool cover -func=coverage.out
	@echo "✓ Tests passed"

integration-test: build
	@echo "Running integration tests..."
	go test -v -tags=integration -run TestHTTPPlugin
	@echo "✓ Integration tests passed"

clean:
	rm -f $(PLUGIN_NAME) coverage.out
	@echo "✓ Cleaned"

install: build
	@echo "Installing plugin..."
	mkdir -p /opt/waterflow/plugins
	cp $(PLUGIN_NAME) /opt/waterflow/plugins/
	@echo "✓ Installed to /opt/waterflow/plugins/$(PLUGIN_NAME)"

.DEFAULT_GOAL := build
```

### 开发顺序建议

**阶段 1: 基础实现 (Day 1-2)**
1. 创建目录结构
2. 实现 HTTPRequestNode 结构体
3. 实现 GET 请求
4. 基础单元测试 (GET)

**阶段 2: 完整功能 (Day 2-3)**
1. 实现所有 HTTP 方法
2. 添加 Headers 和 Body 支持
3. 实现响应解析
4. 错误分类逻辑

**阶段 3: 测试和文档 (Day 3-4)**
1. 完善所有单元测试
2. 编写集成测试
3. 实现 Makefile
4. 编写 README 和 YAML 示例

**总估算: 3-4 工作日**

### 验收标准检查清单

- [ ] **AC1: HTTP 方法支持**
  - [ ] 支持 GET, POST, PUT, DELETE, PATCH
  - [ ] 方法大小写不敏感
  - [ ] 默认 GET

- [ ] **AC2: 请求配置**
  - [ ] url 参数必填且验证
  - [ ] headers 支持多个键值对
  - [ ] body 支持 string 和 JSON object
  - [ ] timeout 默认 30s，最大 5m
  - [ ] User-Agent 自动设置

- [ ] **AC3: 响应处理**
  - [ ] 返回 status_code, headers, body
  - [ ] 2xx 成功
  - [ ] 4xx/5xx 返回错误
  - [ ] JSON 自动解析
  - [ ] 超时可重试

- [ ] **AC4: 安全和错误处理**
  - [ ] 网络错误可重试
  - [ ] 4xx 不可重试
  - [ ] 5xx 可重试
  - [ ] HTTPS 支持
  - [ ] SSL 验证

- [ ] **代码质量**
  - [ ] 单元测试覆盖率 >90%
  - [ ] 集成测试通过
  - [ ] 无 race condition
  - [ ] golangci-lint 无错误

- [ ] **文档完整性**
  - [ ] README.md 包含 6+ 示例
  - [ ] API 集成场景说明
  - [ ] YAML 示例覆盖常见用例

---

## References

**架构决策:**
- [ADR-0003: 插件化节点系统](../adr/0003-plugin-based-node-system.md)

**依赖 Stories:**
- [Story 3.1: 节点接口设计](./3-1-node-interface-design.md) - Node 接口定义
- [Story 3.2: Shell 命令执行节点](./3-2-shell-command-execution-node.md) - 错误分类参考

**后续 Stories:**
- [Story 3.6: 文件传输节点](../epics.md#story-36-文件传输节点)

**技术文档:**
- [Go net/http Package](https://pkg.go.dev/net/http)
- [Temporal Error Handling](https://docs.temporal.io/docs/go/error-handling)

---

## Dev Agent Record

### Context Reference
- Epic 3 定义: [docs/epics.md](../epics.md#epic-3-核心节点插件库)
- Story 3.1 接口: [docs/sprint-artifacts/3-1-node-interface-design.md](./3-1-node-interface-design.md)

### Agent Model Used
Claude Sonnet 4.5

### Completion Notes
- [x] 所有 AC 已实现
- [x] 单元测试通过 (覆盖率 92.3%, 18 tests)
- [x] 集成测试通过 (2 tests)
- [x] 插件可成功编译和加载 (request.so, 35MB)
- [x] 文档已完成
- [x] **代码审查完成**: 修复 3 个 CRITICAL 问题 (NonRetryable 错误标记)

**完成日期:** 2025-12-30  
**总测试数:** 20 (18单元 + 2集成)  
**测试覆盖率:** 92.3%  
**编译产物:** request.so (35MB)  
**测试运行时间:** 9.1s

**实现亮点:**
- ✅ 支持所有主流 HTTP 方法 (GET/POST/PUT/DELETE/PATCH)
- ✅ 自动 JSON 序列化/反序列化 (Content-Type 感知)
- ✅ Temporal 错误分类 (4xx 永久, 5xx 临时) - **已修复 NonRetryable 标记**
- ✅ SSL 验证配置
- ✅ Context 取消支持
- ✅ 完整的超时处理 (默认30s, 最大5分钟)

**代码审查修复 (2025-12-30):**
1. ✅ 修复 4xx 错误未标记为永久错误 - 使用 `NewNonRetryableApplicationError`
2. ✅ 修复 TLS 错误未标记为永久错误 - 使用 `NewNonRetryableApplicationError`
3. ✅ 添加 NonRetryable 标志验证测试 (4xx 不可重试, 5xx 可重试)
4. ✅ 修复集成测试执行问题 (函数名缩短)

### File List
**新增文件:**
- `plugins/http/request/main.go` - HTTP 请求节点实现 (331 行)
- `plugins/http/request/main_test.go` - 单元测试 (385 行, 18 tests)
- `plugins/http/request/integration_test.go` - 集成测试 (69 行, 2 tests)
- `plugins/http/request/Makefile` - 编译脚本 (build/test/clean/install targets)
- `plugins/http/request/README.md` - 节点文档 (175 行, 6 examples)
- `plugins/http/request/go.mod` - Go module 定义
- `plugins/http/request/go.sum` - 依赖锁定文件

**依赖文件:**
- `pkg/dsl/node/interface.go` - Story 3.1 Node 接口
- Temporal SDK v1.30.0 - 错误类型和上下文

### Change Log
**2025-12-30 - Story 完成**
- 实现完整 HTTP 客户端节点 (支持 GET/POST/PUT/DELETE/PATCH)
- 集成 Temporal 错误分类 (4xx 永久错误, 5xx 临时错误)
- JSON 自动序列化/反序列化
- 所有 9 个任务完成, 20 个测试通过
- 覆盖率 92.3%
- 文档包含 6 个完整示例
- 插件编译成功 (35MB)

**2025-12-30 - 代码审查修复**
- 修复 AC4 CRITICAL 违反: 4xx/TLS 错误未标记 NonRetryable
- 从 `temporal.NewApplicationError` 改为 `temporal.NewNonRetryableApplicationError`
- 添加测试验证 NonRetryable 标志 (4xx 不可重试, 5xx 可重试)
- 修复集成测试函数名 (TestHTTPPlugin_* → 可执行)
- 所有测试通过 (单元测试 18 个 + 集成测试 2 个)

---

**Story 准备完成！开发者现在拥有创建 HTTP 请求节点所需的所有上下文！** 🌐
