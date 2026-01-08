# Waterflow 日志最佳实践指南

本文档提供 Waterflow 项目中使用结构化日志的最佳实践指南。

## 目录

- [日志级别使用指南](#日志级别使用指南)
- [结构化日志最佳实践](#结构化日志最佳实践)
- [上下文日志模式](#上下文日志模式)
- [敏感信息处理](#敏感信息处理)
- [性能优化技巧](#性能优化技巧)
- [安全最佳实践](#安全最佳实践)

## 日志级别使用指南

### Debug 级别
用于开发调试信息，生产环境默认关闭。

**何时使用:**
- 变量值追踪
- 函数调用流程
- 中间计算结果
- 性能分析数据

**示例:**
```go
logger.Debug("Processing workflow step",
    logger.WorkflowID(workflowID),
    logger.StepName(stepName),
    zap.Any("input_params", params),
)
```

### Info 级别
用于记录重要的业务事件和操作。

**何时使用:**
- 工作流启动/完成
- HTTP 请求处理
- 系统状态变更
- 配置加载

**示例:**
```go
logger.Info("Workflow started",
    logger.WorkflowID(workflowID),
    zap.String("user", username),
    zap.Int("total_steps", len(steps)),
)
```

### Warn 级别
用于记录潜在问题，但系统仍可继续运行。

**何时使用:**
- 降级功能使用
- 接近资源限制
- 重试操作
- 配置缺失使用默认值

**示例:**
```go
logger.Warn("Temporal connection failed, using fallback",
    zap.Error(err),
    zap.String("fallback_mode", "local"),
)
```

### Error 级别
用于记录需要人工关注的错误。

**何时使用:**
- 请求处理失败
- 外部服务调用失败
- 数据验证失败
- 系统异常

**示例:**
```go
logger.Error("Workflow execution failed",
    logger.WorkflowID(workflowID),
    logger.ErrorType(err.ErrorType()),
    logger.Retryable(err.IsRetryable()),
    zap.Error(err),
)
```

## 结构化日志最佳实践

### ✅ 使用结构化字段，不使用字符串拼接

**好的做法:**
```go
logger.Info("Request processed",
    zap.String("method", method),
    zap.String("path", path),
    zap.Int("status", statusCode),
    zap.Duration("duration", duration),
)
```

**不好的做法:**
```go
// ❌ 不要这样做
logger.Info(fmt.Sprintf("Request %s %s returned %d in %v", method, path, statusCode, duration))
```

**原因:**
- 结构化字段易于解析和查询
- 避免字符串拼接开销
- 类型安全，防止格式错误

### ✅ 使用标准字段名常量

**好的做法:**
```go
logger.Info("Step completed",
    logger.WorkflowID(workflowID),
    logger.JobID(jobID),
    logger.StepName(stepName),
    zap.Duration(logger.FieldDuration, duration),
)
```

**不好的做法:**
```go
// ❌ 字段名不一致
logger.Info("Step completed",
    zap.String("wf_id", workflowID),     // 应该用 workflow_id
    zap.String("job", jobID),            // 应该用 job_id
    zap.String("step", stepName),        // 应该用 step_name
)
```

### ✅ 避免记录大对象

**好的做法:**
```go
logger.Info("Workflow submitted",
    logger.WorkflowID(workflowID),
    zap.Int("step_count", len(workflow.Jobs)),
    zap.String("workflow_name", workflow.Name),
)
```

**不好的做法:**
```go
// ❌ 记录整个工作流对象（可能很大）
logger.Info("Workflow submitted",
    zap.Any("workflow", workflow),  // 可能包含大量数据
)
```

## 上下文日志模式

### 为工作流执行创建上下文 Logger

**推荐模式:**
```go
type Executor struct {
    logger *zap.Logger
}

func NewExecutor(workflowID string) *Executor {
    return &Executor{
        logger: logger.WithWorkflowContext(workflowID),
    }
}

func (e *Executor) ExecuteStep(jobID, stepName string) error {
    stepLogger := e.logger.With(
        logger.JobID(jobID),
        logger.StepName(stepName),
    )
    
    stepLogger.Info("Step started")
    // ... 执行逻辑 ...
    stepLogger.Info("Step completed", zap.Duration("duration", duration))
    
    return nil
}
```

**优点:**
- 自动附加上下文字段
- 避免重复传递参数
- 日志自动关联到工作流

### 链式创建子 Logger

```go
// 工作流级别
wfLogger := logger.WithWorkflowContext(workflowID)

// Job 级别
jobLogger := wfLogger.With(logger.JobID(jobID))

// Step 级别
stepLogger := jobLogger.With(logger.StepName(stepName))

stepLogger.Info("Executing")  // 自动包含 workflow_id, job_id, step_name
```

## 敏感信息处理

### 自动脱敏字段

系统自动脱敏以下字段（不区分大小写）：
- `password`
- `token`
- `api_key`
- `secret`
- `private_key`
- `access_token`
- `refresh_token`
- `authorization`
- `cookie`

**示例:**
```go
logger.Info("User authenticated",
    zap.String("username", "admin"),
    zap.String("password", "secret123"),  // 输出: ***REDACTED***
)
```

### 不要记录敏感业务数据

```go
// ❌ 不好的做法
logger.Info("Payment processed",
    zap.String("credit_card", cardNumber),   // 永远不要记录
    zap.String("cvv", cvv),                  // 永远不要记录
)

// ✅ 好的做法
logger.Info("Payment processed",
    zap.String("payment_id", paymentID),
    zap.String("card_last4", last4Digits),   // 只记录后4位
    zap.String("payment_method", "card"),
)
```

### 配置自定义敏感字段

**方法1: 配置文件**
```yaml
log:
  level: info
  format: json
  sensitive_fields:
    - custom_secret
    - internal_token
```

**方法2: 环境变量**
```bash
export WATERFLOW_LOG_SENSITIVE_FIELDS=custom_secret,internal_token
```

## 性能优化技巧

### 使用 CheckedEntry 避免无效日志构造

当日志级别未启用时，避免昂贵的操作：

```go
// ✅ 使用 CheckedEntry
if ce := logger.Check(zap.DebugLevel, "Expensive debug log"); ce != nil {
    // 只有在 debug 级别启用时才执行
    expensiveData := computeExpensiveData()
    ce.Write(zap.Any("data", expensiveData))
}

// ❌ 不好的做法
logger.Debug("Expensive debug log",
    zap.Any("data", computeExpensiveData()),  // 即使debug未启用也会执行
)
```

### 避免在高频路径使用 Debug 日志

```go
// ❌ 不好的做法（每次循环都记录）
for _, item := range items {
    logger.Debug("Processing item", zap.String("id", item.ID))
    process(item)
}

// ✅ 好的做法（只记录摘要）
logger.Debug("Processing items", zap.Int("count", len(items)))
for _, item := range items {
    process(item)
}
logger.Debug("Items processed")
```

### 使用零分配日志 API

```go
// ✅ 零分配（使用 zap.Field）
logger.Info("Request processed",
    zap.String("method", "GET"),
    zap.Int("status", 200),
)

// ❌ 有分配（使用 Sugar API）
sugar := logger.Sugar()
sugar.Infof("Request %s returned %d", "GET", 200)
```

## 安全最佳实践

### 防止日志注入攻击

**不直接将用户输入拼接到消息字段:**

```go
// ❌ 危险：用户输入可能包含换行符等特殊字符
userInput := r.FormValue("name")
logger.Info(fmt.Sprintf("User input: %s", userInput))

// ✅ 安全：使用结构化字段自动转义
logger.Info("User input received",
    zap.String("input", userInput),  // Zap 自动转义特殊字符
)
```

### Admin 端点安全

访问 `/admin/log-level` 端点需要认证：

```bash
# ❌ 未认证请求会被拒绝
curl http://localhost:8080/admin/log-level

# ✅ 使用 Bearer Token
curl -H "Authorization: Bearer your-token" \
     http://localhost:8080/admin/log-level

# ✅ 使用 API Key
curl -H "X-API-Key: your-api-key" \
     http://localhost:8080/admin/log-level
```

### 审计日志

所有日志级别修改都会记录审计日志：

```json
{
  "level": "info",
  "msg": "Log level changed via API",
  "level": "debug",
  "changed_by": "admin",
  "remote_addr": "192.168.1.100:54321",
  "user_agent": "curl/7.68.0"
}
```

### 不记录完整请求/响应体

```go
// ❌ 不好的做法（可能包含敏感数据或导致日志膨胀）
logger.Info("Request received",
    zap.ByteString("body", requestBody),  // 可能很大
)

// ✅ 好的做法（只记录关键信息）
logger.Info("Request received",
    zap.String("method", r.Method),
    zap.String("path", r.URL.Path),
    zap.Int("content_length", int(r.ContentLength)),
)
```

## 常见反模式

### ❌ 反模式1：过度记录

```go
// 不要在每个函数入口/出口都记录
func ProcessData(data string) error {
    logger.Debug("ProcessData called")  // ❌ 过度
    // ... 处理逻辑 ...
    logger.Debug("ProcessData finished")  // ❌ 过度
    return nil
}
```

### ❌ 反模式2：在循环中记录详细信息

```go
// 不要在大循环中记录每次迭代
for i := 0; i < 1000000; i++ {
    logger.Debug("Processing", zap.Int("index", i))  // ❌ 性能问题
}
```

### ❌ 反模式3：使用 panic 级别日志

```go
// Waterflow 不使用 Panic 或 Fatal 级别
logger.Fatal("Critical error")  // ❌ 会导致程序退出
logger.Panic("Something wrong")  // ❌ 会导致 panic
```

**正确做法:**
```go
// 使用 Error 并返回错误
logger.Error("Critical error", zap.Error(err))
return err
```

## 日志查询示例

使用 `jq` 查询 JSON 日志：

```bash
# 按 workflow_id 查询
jq 'select(.workflow_id == "wf-abc-123")' < server.log

# 按级别过滤
jq 'select(.level == "error")' < server.log

# 统计错误类型
jq -r 'select(.level == "error") | .error_type' < server.log | sort | uniq -c

# 计算平均执行时间
jq -s 'map(select(.msg == "Step completed") | .duration_ms) | add/length' < server.log
```

## 总结

**核心原则:**
1. ✅ **结构化字段** - 不使用字符串拼接
2. ✅ **标准字段名** - 使用常量确保一致性
3. ✅ **敏感信息脱敏** - 依赖自动脱敏或不记录
4. ✅ **上下文日志** - 使用 WithContext 模式
5. ✅ **性能优化** - CheckedEntry + 零分配 API
6. ✅ **安全防护** - 防止日志注入，保护 Admin 端点
7. ✅ **适度记录** - 避免过度记录和大对象

遵循这些最佳实践，可以确保日志系统既高效又安全。
