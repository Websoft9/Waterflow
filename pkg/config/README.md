# pkg/config

配置管理包,支持多种配置源和自动验证。

## 功能

- 支持 YAML 配置文件
- 支持环境变量 (WATERFLOW_* 前缀)
- 支持命令行参数
- 配置优先级: 命令行 > 环境变量 > 配置文件 > 默认值
- 自动配置验证

## 使用示例

```go
import "github.com/Websoft9/waterflow/pkg/config"

// 加载配置
cfg, err := config.Load("config.yaml")
if err != nil {
    log.Fatal(err)
}

// 使用配置
fmt.Printf("Server running on %s:%d\n", cfg.Server.Host, cfg.Server.Port)
```
