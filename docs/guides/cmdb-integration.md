# CMDB 集成指南

> ⚠️ **历史文档警告** (更新于 2025-12-29)  
> 本文档描述的 Agent 注册/心跳机制已废弃（ADR-0007 之前的架构）。  
> **当前架构：** Agent 通过 Temporal Worker 自动连接，无需注册 API。  
> **CMDB 集成：** 现在通过 `server-groups.yaml` 文件进行服务器组映射。  
> **参考文档：** [ADR-0008](../adr/0008-temporal-as-internal-service.md) | [Server Groups 指南](./server-groups.md)

## 概述

Waterflow 通过 `ServerGroupProvider` 接口支持集成外部 CMDB 系统,让您可以从现有的配置管理数据库、Ansible Inventory 或其他服务器清单系统获取服务器组信息。

## 内置 Provider

### 1. InMemoryProvider (默认)

**适用场景:** 测试环境、小规模部署、动态 Agent 注册

**配置:**
```yaml
server:
  server_group_provider: memory  # 可省略,这是默认值
```

**特点:**
- ✅ Agent 启动时通过 API 动态注册
- ✅ 支持心跳更新
- ✅ 零配置,开箱即用
- ⚠️ 信息存储在内存中
- ⚠️ Server 重启后数据丢失

**Agent 注册示例:**
```bash
curl -X POST http://localhost:8080/v1/agents/register \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "agent-prod-web-1",
    "hostname": "web-server-1.example.com",
    "ip_address": "192.168.1.10",
    "task_queues": ["web-servers", "linux-amd64"],
    "metadata": {
      "datacenter": "us-west",
      "role": "web",
      "environment": "production"
    }
  }'
```

### 2. FileProvider

**适用场景:** 静态服务器清单、版本控制的配置

**配置:**
```yaml
server:
  server_group_provider: file
  server_group_file: /etc/waterflow/server-groups.yaml
```

**特点:**
- ✅ YAML 配置文件定义服务器组
- ✅ 支持版本控制 (Git)
- ✅ 适合服务器组不经常变化的场景
- ⚠️ 不支持动态注册
- ⚠️ 需要重启 Server 以重新加载配置

**配置文件示例:** (`/etc/waterflow/server-groups.yaml`)
```yaml
groups:
  web-servers:
    servers:
      - agent_id: agent-web-1
        hostname: web-1.example.com
        ip_address: 10.0.1.20
        task_queues:
          - web-servers
          - linux-amd64
        metadata:
          datacenter: us-west
          role: web
          environment: production
      
      - agent_id: agent-web-2
        hostname: web-2.example.com
        ip_address: 10.0.1.21
        task_queues:
          - web-servers
          - linux-amd64
        metadata:
          datacenter: us-east
          role: web
          environment: production
  
  database-servers:
    servers:
      - agent_id: agent-db-1
        hostname: postgres-primary.example.com
        ip_address: 10.0.2.10
        task_queues:
          - database-servers
          - linux-amd64
        metadata:
          role: database
          db_type: postgresql
```

## 自定义 Provider 开发

### 接口定义

实现 `ServerGroupProvider` 接口的 3 个方法:

```go
package provider

import (
	"context"
	"time"
)

type ServerGroupProvider interface {
	// GetServers returns a list of servers in the specified group.
	// Returns empty list if group doesn't exist or has no servers.
	GetServers(ctx context.Context, groupName string) ([]ServerInfo, error)

	// ListGroups returns all available server group names.
	// This is used for discovery and validation.
	ListGroups(ctx context.Context) ([]string, error)

	// Close releases any resources held by the provider.
	Close() error
}

type ServerInfo struct {
	AgentID       string            `json:"agent_id"`
	Hostname      string            `json:"hostname"`
	IPAddress     string            `json:"ip_address,omitempty"`
	TaskQueues    []string          `json:"task_queues"`
	Status        string            `json:"status"` // "healthy", "unhealthy", "unknown"
	LastHeartbeat time.Time         `json:"last_heartbeat"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}
```

### 示例 1: Ansible Inventory 集成

参考实现: `examples/providers/ansible_provider.go`

```go
type AnsibleInventoryProvider struct {
	inventoryPath string
}

func NewAnsibleInventoryProvider(inventoryPath string) *AnsibleInventoryProvider {
	return &AnsibleInventoryProvider{
		inventoryPath: inventoryPath,
	}
}

func (p *AnsibleInventoryProvider) GetServers(ctx context.Context, groupName string) ([]ServerInfo, error) {
	// 执行 ansible-inventory 命令查询服务器
	cmd := exec.CommandContext(ctx, "ansible-inventory",
		"-i", p.inventoryPath,
		"--list",
		"--export",
	)
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query ansible inventory: %w", err)
	}
	
	// 解析 JSON 输出并转换为 ServerInfo
	// ... (详见完整代码)
}
```

**使用:**
```go
provider := NewAnsibleInventoryProvider("/etc/ansible/inventory")
servers, err := provider.GetServers(ctx, "web-servers")
```

### 示例 2: HTTP API 集成

从远程 CMDB API 查询服务器信息:

```go
type HTTPCMDBProvider struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewHTTPCMDBProvider(baseURL, apiKey string) *HTTPCMDBProvider {
	return &HTTPCMDBProvider{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *HTTPCMDBProvider) GetServers(ctx context.Context, groupName string) ([]ServerInfo, error) {
	url := fmt.Sprintf("%s/api/v1/server-groups/%s/servers", p.baseURL, groupName)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	// 解析响应并转换为 ServerInfo
	var servers []ServerInfo
	if err := json.NewDecoder(resp.Body).Decode(&servers); err != nil {
		return nil, err
	}
	
	return servers, nil
}
```

### 示例 3: 数据库集成

从数据库查询服务器清单:

```go
type DatabaseProvider struct {
	db *sql.DB
}

func NewDatabaseProvider(dsn string) (*DatabaseProvider, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	
	return &DatabaseProvider{db: db}, nil
}

func (p *DatabaseProvider) GetServers(ctx context.Context, groupName string) ([]ServerInfo, error) {
	query := `
		SELECT agent_id, hostname, ip_address, status
		FROM servers
		WHERE server_group = $1 AND active = true
	`
	
	rows, err := p.db.QueryContext(ctx, query, groupName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var servers []ServerInfo
	for rows.Next() {
		var s ServerInfo
		if err := rows.Scan(&s.AgentID, &s.Hostname, &s.IPAddress, &s.Status); err != nil {
			return nil, err
		}
		s.TaskQueues = []string{groupName}
		servers = append(servers, s)
	}
	
	return servers, rows.Err()
}

func (p *DatabaseProvider) Close() error {
	return p.db.Close()
}
```

## 集成到 Waterflow Server

### 方法 1: 内置 Provider (推荐用于生产)

将自定义 Provider 添加到 `internal/server/server.go`:

```go
import "your-company/waterflow-cmdb-provider"

func New(cfg *config.Config, logger *zap.Logger, version, commit, buildTime string) *Server {
	var sgProvider provider.ServerGroupProvider
	
	switch cfg.Server.ServerGroupProvider {
	case "cmdb":
		sgProvider = cmdbprovider.NewCMDBProvider(cfg.Server.CMDBAPIKey)
	case "ansible":
		sgProvider = ansibleprovider.NewAnsibleProvider(cfg.Server.AnsibleInventory)
	case "file":
		sgProvider, _ = provider.NewFileProvider(cfg.Server.ServerGroupFile)
	default:
		sgProvider = provider.NewInMemoryProvider()
	}
	
	return &Server{
		serverGroupProvider: sgProvider,
		// ...
	}
}
```

### 方法 2: 插件机制 (规划中)

未来版本将支持通过 Go Plugin 机制动态加载自定义 Provider。

## 最佳实践

### 1. 缓存查询结果

CMDB 查询可能较慢,考虑添加缓存层:

```go
type CachedProvider struct {
	underlying provider.ServerGroupProvider
	cache      *cache.Cache
	ttl        time.Duration
}

func (p *CachedProvider) GetServers(ctx context.Context, groupName string) ([]ServerInfo, error) {
	// 先查缓存
	if cached, found := p.cache.Get(groupName); found {
		return cached.([]ServerInfo), nil
	}
	
	// 缓存未命中,查询底层 Provider
	servers, err := p.underlying.GetServers(ctx, groupName)
	if err != nil {
		return nil, err
	}
	
	// 写入缓存
	p.cache.Set(groupName, servers, p.ttl)
	return servers, nil
}
```

### 2. 错误处理

优雅处理 CMDB 不可用的情况:

```go
func (p *CustomProvider) GetServers(ctx context.Context, groupName string) ([]ServerInfo, error) {
	servers, err := p.queryCMDB(ctx, groupName)
	if err != nil {
		// 记录错误
		p.logger.Error("CMDB query failed", zap.Error(err))
		
		// 返回空列表而不是错误,允许工作流继续
		// (或者使用备用数据源)
		return []ServerInfo{}, nil
	}
	return servers, nil
}
```

### 3. 超时控制

使用 context 实现超时:

```go
func (p *CustomProvider) GetServers(ctx context.Context, groupName string) ([]ServerInfo, error) {
	// 设置查询超时
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	// 使用带超时的 context 查询
	return p.queryCMDB(ctx, groupName)
}
```

### 4. 日志记录

记录所有 CMDB 交互便于排查问题:

```go
func (p *CustomProvider) GetServers(ctx context.Context, groupName string) ([]ServerInfo, error) {
	p.logger.Debug("Querying CMDB for server group",
		zap.String("group", groupName),
	)
	
	servers, err := p.queryCMDB(ctx, groupName)
	if err != nil {
		p.logger.Error("CMDB query failed",
			zap.String("group", groupName),
			zap.Error(err),
		)
		return nil, err
	}
	
	p.logger.Info("CMDB query successful",
		zap.String("group", groupName),
		zap.Int("server_count", len(servers)),
	)
	
	return servers, nil
}
```

## 故障排查

### CMDB 连接失败

**症状:** Server 启动时报错 "Failed to connect to CMDB"

**解决方法:**
1. 检查 CMDB API 地址和认证凭据
2. 验证网络连接: `curl <cmdb-url>`
3. 查看 Server 日志获取详细错误信息

### 服务器组为空

**症状:** `GetServers()` 返回空列表

**解决方法:**
1. 验证服务器组名称拼写
2. 检查 CMDB 中是否确实有该组
3. 使用 `ListGroups()` 查看所有可用组

### 性能问题

**症状:** CMDB 查询非常慢

**解决方法:**
1. 添加缓存层 (见最佳实践)
2. 优化 CMDB 查询(添加索引、减少返回字段)
3. 考虑异步预加载常用服务器组

## 参考资源

- [ServerGroupProvider 接口文档](../pkg/provider/server_group.go)
- [InMemoryProvider 实现](../pkg/provider/memory_provider.go)
- [FileProvider 实现](../pkg/provider/file_provider.go)
- [Ansible Provider 示例](../examples/providers/ansible_provider.go)
