# Waterflow Error Handling System

统一的类型化错误处理系统,支持 RFC 7807 Problem Details for HTTP APIs 和 Temporal 重试策略。

## Overview

`pkg/errors` 包提供了 Waterflow 项目中所有错误的统一接口和实现。它基于以下原则设计:

- **类型化** - 每种错误场景有专用的类型
- **上下文丰富** - 错误包含 workflow_id, job_id, step_name 等上下文
- **可序列化** - 所有错误可转换为 JSON (支持 API 响应和日志记录)
- **错误链完整** - 使用 Go 1.13+ 的 `errors.Unwrap()` 保留底层错误
- **分类明确** - 区分可重试 (临时性) 和不可重试 (永久性) 错误
- **RFC 7807 兼容** - REST API 错误响应遵循标准格式

## Quick Start

### 基本使用

```go
import "github.com/Websoft9/waterflow/pkg/errors"

// 创建验证错误
err := errors.NewValidationError("YAML syntax error", []errors.FieldError{
    {
        Line:       10,
        Field:      "jobs.build.runs-on",
        Error:      "required",
        Suggestion: "add runs-on: <task-queue-name>",
    },
})

// 创建节点执行错误
err := errors.NewNodeExecutionError("http/request", "api-call", 
    fmt.Errorf("connection timeout"))

// 创建工作流未找到错误
err := errors.NewWorkflowNotFoundError("wf-123")
```

### 错误包装和上下文

```go
// 包装错误并添加上下文
wrapped := errors.WrapWithContext(originalErr, "Step execution failed", map[string]interface{}{
    "workflow_id": "wf-123",
    "job_id":      "job-456",
    "step_name":   "deploy",
})

// 检查错误类型和可重试性
if wrapped.IsRetryable() {
    // 自动重试
} else {
    // 立即失败
}
```

### RFC 7807 响应

```go
// HTTP Handler 中使用
func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error) {
    rfc7807 := errors.ToRFC7807(err, r.URL.Path)
    
    w.Header().Set("Content-Type", "application/problem+json")
    w.WriteHeader(rfc7807.Status)
    
    json.NewEncoder(w).Encode(rfc7807)
}

// 响应示例:
// {
//   "type": "validation_error",
//   "title": "Validation Failed",
//   "status": 400,
//   "detail": "Found 3 validation errors in workflow definition",
//   "instance": "/v1/workflows/abc-123",
//   "errors": [...]
// }
```

## Error Types

### WaterflowError Interface

所有 Waterflow 错误都实现这个接口:

```go
type WaterflowError interface {
    error
    ErrorType() string                      // 错误类型 (validation_error, node_error 等)
    ErrorMessage() string                   // 人类可读的错误信息
    ErrorContext() map[string]interface{}   // 错误上下文
    IsRetryable() bool                      // 是否可重试
    ToJSON() ([]byte, error)                // 序列化为 JSON
}
```

### Workflow Errors

- **WorkflowNotFoundError** - 工作流不存在 (non-retryable)
- **WorkflowExecutionError** - 工作流执行失败 (retryability depends on cause)
- **WorkflowTimeoutError** - 工作流超时 (retryable)
- **WorkflowCancelledError** - 工作流被取消 (non-retryable)

### Validation Errors

- **ValidationError** - YAML/Schema 验证失败 (non-retryable)
- **SchemaError** - Schema 类型不匹配 (non-retryable)

### Node Errors

- **NodeExecutionError** - 节点执行失败 (retryability depends on cause)
- **NodeNotFoundError** - 节点类型未注册 (non-retryable)
- **NodeParameterError** - 节点参数错误 (non-retryable)

## Error Classification

### Permanent Errors (Non-Retryable)

这些错误不应该重试,因为它们是永久性的:

- `validation_error` - YAML 语法或 Schema 验证错误
- `schema_error` - Schema 验证失败
- `not_found` - 资源不存在 (404)
- `permission_denied` - 权限拒绝 (403)
- `invalid_argument` - 参数错误 (400)
- `node_not_registered` - 节点未注册
- `plugin_load_error` - 插件加载失败
- `cancelled` - 用户取消

### Temporary Errors (Retryable)

这些错误是临时性的,应该重试:

- `network_timeout` - 网络超时
- `connection_refused` - 连接被拒绝
- `service_unavailable` - 服务不可用 (503)
- `internal_error` - 服务器内部错误 (500)
- `deadline_exceeded` - 超时
- `resource_exhausted` - 资源耗尽

### Error Classifier

```go
// 使用默认分类器 (未知错误不可重试)
classifier := errors.NewErrorClassifier()

// 使用自定义配置 (未知错误可重试)
classifier := errors.NewErrorClassifierWithConfig(errors.ErrorClassifierConfig{
    UnknownErrorIsRetryable: true,
})

// 分类错误
errType := classifier.ClassifyError(err)
retryable := classifier.IsRetryable(errType)

// 全局辅助函数
if errors.IsRetryableError(err) {
    // 重试逻辑
}
```

## Advanced Features

### Stack Trace Collection

堆栈跟踪默认不收集 (性能考虑),仅在以下情况启用:

```go
// 方式 1: 设置环境变量
// DEBUG=true go run main.go

// 方式 2: 明确调用
err := &errors.BaseError{...}
err.WithStackTrace()

// 方式 3: 检查调试模式 (在错误创建时自动)
if os.Getenv("DEBUG") == "true" {
    // Stack trace will be collected automatically
}
```

### Sensitive Field Sanitization

敏感字段自动脱敏:

```go
// 默认敏感字段列表
var defaultSensitiveKeys = []string{
    "password", "api_key", "token", "secret", "credential",
    "authorization", "auth", "key", "private_key", "access_token",
    "refresh_token", "session", "cookie",
}

// 自定义敏感字段
config := errors.SensitiveFieldsConfig{
    AdditionalFields: []string{"custom_key", "my_secret"},
}
sanitized := errors.SanitizeContextWithConfig(ctx, config)
```

### Error Chaining

支持 Go 1.13+ 的错误链:

```go
// 包装错误
wrapped := errors.WrapError(originalErr, "", "operation failed")

// 检查错误类型
if errors.Is(wrapped, originalErr) {
    // True
}

// 类型断言
var baseErr *errors.BaseError
if errors.As(wrapped, &baseErr) {
    // 可以访问 baseErr 的字段
}
```

## Configuration

### ErrorClassifierConfig

```go
type ErrorClassifierConfig struct {
    // UnknownErrorIsRetryable 决定未知错误是否可重试
    // 默认: false (安全策略 - 避免无限重试)
    UnknownErrorIsRetryable bool
}
```

### SensitiveFieldsConfig

```go
type SensitiveFieldsConfig struct {
    // AdditionalFields 指定额外的敏感字段名称
    AdditionalFields []string
}
```

### Stack Trace Collection

通过环境变量控制:

```bash
# 启用堆栈跟踪收集
DEBUG=true

# 或
DEBUG=1
```

## Best Practices

### 1. 创建错误时添加上下文

```go
// 好的做法
err := errors.NewNodeExecutionError("http/request", "api-call", cause).
    WithContext("url", "https://api.example.com").
    WithContext("method", "POST")

// 避免
err := errors.New("request failed")
```

### 2. 使用类型化错误而非字符串错误

```go
// 好的做法
return errors.NewValidationError("YAML syntax error", fieldErrors)

// 避免
return fmt.Errorf("YAML syntax error")
```

### 3. 在 API Handler 中使用 RFC 7807

```go
// 好的做法
rfc7807 := errors.ToRFC7807(err, r.URL.Path)
w.Header().Set("Content-Type", "application/problem+json")
json.NewEncoder(w).Encode(rfc7807)

// 避免
w.WriteHeader(500)
w.Write([]byte(err.Error()))
```

### 4. 检查可重试性而非假设

```go
// 好的做法
if errors.IsRetryableError(err) {
    // Temporal 会自动重试
    return err
} else {
    // 标记为 NonRetryable
    return temporal.NewNonRetryableApplicationError(err.Error(), "", err)
}

// 避免
// 假设所有网络错误都可重试
```

### 5. 保留错误链

```go
// 好的做法
wrapped := errors.WrapWithContext(err, "operation failed", context)

// 避免
return fmt.Errorf("operation failed: %s", err.Error()) // 丢失了原始错误
```

## Temporal Integration

### Activity 错误处理

```go
func MyActivity(ctx context.Context, input Input) (Output, error) {
    result, err := doSomething()
    if err != nil {
        // 包装错误并分类
        wrapped := errors.NewNodeExecutionError("shell", "build", err)
        
        // Temporal 会根据 IsRetryable() 决定是否重试
        if !wrapped.IsRetryable() {
            return Output{}, temporal.NewNonRetryableApplicationError(
                wrapped.Error(), wrapped.ErrorType(), wrapped,
            )
        }
        
        return Output{}, wrapped
    }
    return result, nil
}
```

### Workflow 错误处理

```go
func MyWorkflow(ctx workflow.Context, input Input) (Output, error) {
    err := workflow.ExecuteActivity(ctx, MyActivity, input).Get(ctx, &output)
    if err != nil {
        // 包装 Activity 错误
        wrapped := errors.NewWorkflowExecutionError(
            workflowID, jobID, stepName, err,
        )
        return Output{}, wrapped
    }
    return output, nil
}
```

## Migration Guide

### From pkg/dsl/errors.go

```go
// 旧代码
import "github.com/Websoft9/waterflow/pkg/dsl"

err := dsl.NewValidationError(...)

// 新代码
import "github.com/Websoft9/waterflow/pkg/errors"

err := errors.NewValidationError(...)

// 或者使用别名 (向后兼容)
import "github.com/Websoft9/waterflow/pkg/dsl"

err := dsl.ValidationError(...) // 仍然可用
```

### From node.NonRetryableError

```go
// 旧代码
type NonRetryableError interface {
    error
    IsNonRetryable() bool
}

// 新代码
type WaterflowError interface {
    error
    IsRetryable() bool // 注意: 语义相反
}

// 检查
if err, ok := err.(errors.WaterflowError); ok {
    if !err.IsRetryable() {
        // 永久性错误
    }
}
```

## Examples

### Example 1: Validation Error with Field Details

```go
fieldErrors := []errors.FieldError{
    {
        Line:       10,
        Column:     5,
        Field:      "jobs.build.runs-on",
        Error:      "required field is missing",
        Snippet:    "  build:",
        Suggestion: "add runs-on: <task-queue-name>",
    },
    {
        Line:       15,
        Field:      "jobs.test.steps[0].run",
        Error:      "invalid command syntax",
        Value:      "echo {{ invalid }}",
    },
}

err := errors.NewValidationError("Found 2 validation errors", fieldErrors)

// Convert to RFC 7807 for API response
rfc7807 := err.ToRFC7807()
// {
//   "type": "about:blank",
//   "title": "Workflow Validation Failed",
//   "status": 400,
//   "detail": "Found 2 validation errors",
//   "errors": [...]
// }
```

### Example 2: Node Execution with Retry

```go
result, err := executeNode(nodeType, inputs)
if err != nil {
    // Wrap with context
    nodeErr := errors.NewNodeExecutionError(nodeType, stepName, err)
    
    // Check retryability
    if nodeErr.IsRetryable() {
        // Temporary error - Temporal will retry
        return nil, nodeErr
    } else {
        // Permanent error - fail immediately
        return nil, temporal.NewNonRetryableApplicationError(
            nodeErr.Error(), nodeErr.ErrorType(), nodeErr,
        )
    }
}
```

### Example 3: Custom Error Classification

```go
// Create classifier with custom config
classifier := errors.NewErrorClassifierWithConfig(errors.ErrorClassifierConfig{
    UnknownErrorIsRetryable: true, // Retry unknown errors
})

// Classify error
errType := classifier.ClassifyError(err)
retryable := classifier.IsRetryable(errType)

fmt.Printf("Error type: %s, Retryable: %v\n", errType, retryable)
```

## Testing

```bash
# Run all tests
go test ./pkg/errors/... -v

# Run tests with coverage
go test ./pkg/errors/... -v -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific test
go test ./pkg/errors/... -run TestValidationError -v
```

## References

- [RFC 7807 - Problem Details for HTTP APIs](https://datatracker.ietf.org/doc/html/rfc7807)
- [Go 1.13 Error Wrapping](https://go.dev/blog/go1.13-errors)
- [Temporal Error Handling](https://docs.temporal.io/develop/go/failure-detection)
- [Effective Go - Errors](https://go.dev/doc/effective_go#errors)

## License

Copyright © 2026 Websoft9
