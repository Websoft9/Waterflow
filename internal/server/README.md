# internal/server

HTTP Server 实现。

## 职责

- HTTP Server 生命周期管理
- 路由注册
- 健康检查端点
- 优雅关闭处理

## 使用示例

```go
import "github.com/Websoft9/waterflow/internal/server"

srv := server.New(config)
if err := srv.Start(); err != nil {
    log.Fatal(err)
}
```
