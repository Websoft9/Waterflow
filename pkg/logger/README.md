# pkg/logger

基于 zap 的结构化日志系统。

## 功能

- 结构化 JSON 日志输出
- 可配置的日志级别
- 高性能 (>1M logs/sec)
- 上下文字段支持

## 使用示例

```go
import "github.com/Websoft9/waterflow/pkg/logger"

// 初始化日志
if err := logger.Init("info", "json"); err != nil {
    panic(err)
}

// 记录日志
logger.Log.Info("server started",
    zap.String("component", "server"),
    zap.Int("port", 8080),
)
```
