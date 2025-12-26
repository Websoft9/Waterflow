# Provider Package

ServerGroupProvider 接口及实现,用于管理 Agent 服务器组信息。

## 概述

`provider` 包提供了可扩展的接口,用于查询和管理服务器组信息。支持从多种数据源获取 Agent 服务器清单:

- **InMemoryProvider** - 内存存储,支持动态注册
- **FileProvider** - YAML 文件配置
- **自定义 Provider** - 集成外部 CMDB 系统

## 接口定义

```go
type ServerGroupProvider interface {
	GetServers(ctx context.Context, groupName string) ([]ServerInfo, error)
	ListGroups(ctx context.Context) ([]string, error)
	Close() error
}
```

## 使用示例

### InMemoryProvider

```go
provider := provider.NewInMemoryProvider()

// Agent 注册
provider.RegisterServer(provider.ServerInfo{
	AgentID:    "agent-001",
	Hostname:   "server1.example.com",
	TaskQueues: []string{"linux-amd64", "linux-common"},
	Status:     "healthy",
})

// 查询服务器组
servers, _ := provider.GetServers(context.Background(), "linux-amd64")
```

### FileProvider

```go
provider, err := provider.NewFileProvider("/etc/waterflow/server-groups.yaml")
if err != nil {
	log.Fatal(err)
}
defer provider.Close()

servers, _ := provider.GetServers(context.Background(), "web-servers")
```

## 配置

在 `config.yaml` 中配置 Provider:

```yaml
server:
  server_group_provider: memory  # 或 "file"
  server_group_file: /etc/waterflow/server-groups.yaml  # 仅 provider=file 时需要
```

## 测试

```bash
# 运行测试
go test -v ./pkg/provider/

# 查看覆盖率
go test -cover ./pkg/provider/

# 性能测试
go test -bench=. ./pkg/provider/
```

## 扩展开发

参考 [CMDB 集成指南](../../docs/guides/cmdb-integration.md) 开发自定义 Provider。

## 相关 Story

- Story 2.3: ServerGroupProvider 接口实现
- Story 2.7: Agent 健康监控 (将使用此接口)
