# Logger Package

`pkg/logger` 提供基于 Zap 的结构化日志功能,支持敏感信息脱敏、上下文日志和动态级别配置。

## 特性

- ✅ **结构化日志** - JSON 格式,易于解析和查询
- ✅ **敏感信息脱敏** - 自动过滤 password, token, api_key 等敏感字段
- ✅ **上下文日志** - 支持 workflow_id, job_id, step_name 等上下文
- ✅ **日志级别动态配置** - 运行时调整日志级别
- ✅ **高性能** - 基于 Zap,支持零分配日志
- ✅ **标准字段命名** - 统一的字段命名规范

## 快速开始

### 初始化

```go
import "github.com/websoft9/waterflow/pkg/logger"

// 使用默认配置
err := logger.Init("info", "json")
if err != nil {
    panic(err)
}
defer logger.Sync()
```

### 敏感信息脱敏

默认敏感字段: password, token, api_key, secret, private_key, access_token, refresh_token, authorization, cookie

```go
logger.Log.Info("User authenticated",
    zap.String("username", "admin"),
    zap.String("password", "secret123"),  // 输出: ***REDACTED***
)
```

### 上下文日志

```go
workflowLogger := logger.WithWorkflowContext("wf-123")
workflowLogger.Info("Workflow started")
```

### 标准字段

```go
logger.Log.Info("Step executed",
    logger.WorkflowID("wf-123"),
    logger.JobID("build"),
    logger.StepName("compile"),
)
```

### 动态日志级别

```go
logger.SetLevel("debug")
```

详细文档请参考 [完整文档](../../docs/guides/logging-best-practices.md)
