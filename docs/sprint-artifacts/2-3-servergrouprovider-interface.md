# Story 2.3: ServerGroupProvider 接口实现

Status: ready-for-dev

## Story

As a **系统架构师**,  
I want **实现 ServerGroupProvider 接口支持外部 CMDB 集成**,  
so that **可以从企业现有的 CMDB/Ansible Inventory 等系统获取服务器组信息**。

## Context

这是 **Epic 2: 分布式 Agent 系统**的第三个 Story。Story 2.1 和 2.2 已实现基础的 Agent Worker 和 Task Queue 路由,现在需要提供可扩展的接口,支持从外部系统获取服务器组信息。

**前置依赖:**
- Story 2.1 (Agent Worker 基础框架) - Agent 已能注册到 Task Queue
- Story 2.2 (Task Queue 直接映射) - runs-on 路由机制已实现
- Story 1.2 (配置管理) - 配置系统已完善

**Epic 2 背景:**  
虽然 Task Queue 直接映射提供了零配置路由,但企业用户通常已有 CMDB 系统维护服务器清单。ServerGroupProvider 接口让用户可以从现有系统查询服务器组信息,无需手动维护。

**业务价值:**
- 🔌 **可扩展性** - 支持集成任意 CMDB 系统 (Ansible, Terraform, 自研)
- 📋 **统一管理** - 服务器清单由单一数据源维护
- 🔄 **动态发现** - Server 可查询当前可用的 Agent 和服务器组
- 🛡️ **企业就绪** - 满足企业级用户的集成需求

**设计原则:**
- 接口简单 (≤3 个方法)
- 提供默认实现 (内存、配置文件)
- 可选集成 (不影响核心功能)

## Acceptance Criteria

### AC1: ServerGroupProvider 接口定义

**Given** 需要查询服务器组和 Agent 信息  
**When** 定义 ServerGroupProvider 接口  
**Then** 创建 `pkg/provider/server_group.go`:

```go
package provider

import (
	"context"
	"time"
)

// ServerInfo represents information about a single agent/server.
type ServerInfo struct {
	// AgentID is the unique identifier of the agent worker.
	AgentID string `json:"agent_id"`
	
	// Hostname is the server's hostname.
	Hostname string `json:"hostname"`
	
	// IPAddress is the server's IP address.
	IPAddress string `json:"ip_address,omitempty"`
	
	// TaskQueues is the list of task queues this agent polls.
	TaskQueues []string `json:"task_queues"`
	
	// Status indicates the agent's health status.
	// Values: "healthy", "unhealthy", "unknown"
	Status string `json:"status"`
	
	// LastHeartbeat is the timestamp of the last heartbeat.
	LastHeartbeat time.Time `json:"last_heartbeat"`
	
	// Metadata contains additional server attributes (OS, arch, tags, etc.)
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ServerGroupProvider defines the interface for querying server groups.
// Implementations can integrate with CMDB systems, Ansible inventories,
// configuration files, or other sources.
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
```

**接口设计要点:**
- **简单易实现** - 只有 3 个方法
- **上下文感知** - 所有方法接受 `context.Context` 支持超时和取消
- **灵活元数据** - `Metadata` 字段支持自定义属性
- **无依赖** - 接口不依赖 Temporal 或其他外部库

### AC2: 内存实现 (InMemoryProvider)

**Given** 简单部署场景或测试环境  
**When** 无外部 CMDB 系统  
**Then** 提供内存实现 `pkg/provider/memory_provider.go`:

```go
package provider

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// InMemoryProvider is a simple in-memory implementation of ServerGroupProvider.
// Useful for testing and simple deployments without external CMDB.
type InMemoryProvider struct {
	mu      sync.RWMutex
	groups  map[string][]ServerInfo // groupName -> servers
}

// NewInMemoryProvider creates a new in-memory provider.
func NewInMemoryProvider() *InMemoryProvider {
	return &InMemoryProvider{
		groups: make(map[string][]ServerInfo),
	}
}

// RegisterServer registers a server to one or more groups.
// This is typically called when an agent starts up.
func (p *InMemoryProvider) RegisterServer(server ServerInfo) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	for _, queue := range server.TaskQueues {
		if p.groups[queue] == nil {
			p.groups[queue] = []ServerInfo{}
		}
		
		// Check if server already registered (by AgentID)
		found := false
		for i, existing := range p.groups[queue] {
			if existing.AgentID == server.AgentID {
				// Update existing entry
				p.groups[queue][i] = server
				found = true
				break
			}
		}
		
		if !found {
			p.groups[queue] = append(p.groups[queue], server)
		}
	}
	
	return nil
}

// GetServers returns all servers in the specified group.
func (p *InMemoryProvider) GetServers(ctx context.Context, groupName string) ([]ServerInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	servers, ok := p.groups[groupName]
	if !ok {
		return []ServerInfo{}, nil // Empty list, not an error
	}
	
	// Return a copy to prevent external modification
	result := make([]ServerInfo, len(servers))
	copy(result, servers)
	
	return result, nil
}

// ListGroups returns all available group names.
func (p *InMemoryProvider) ListGroups(ctx context.Context) ([]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	groups := make([]string, 0, len(p.groups))
	for name := range p.groups {
		groups = append(groups, name)
	}
	
	return groups, nil
}

// Close is a no-op for in-memory provider.
func (p *InMemoryProvider) Close() error {
	return nil
}

// UpdateHeartbeat updates the last heartbeat time for a server.
func (p *InMemoryProvider) UpdateHeartbeat(agentID string, status string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	now := time.Now()
	
	for _, servers := range p.groups {
		for i, server := range servers {
			if server.AgentID == agentID {
				servers[i].Status = status
				servers[i].LastHeartbeat = now
			}
		}
	}
	
	return nil
}
```

**使用示例:**
```go
// 创建 provider
provider := provider.NewInMemoryProvider()

// 注册 Agent (Agent 启动时调用)
provider.RegisterServer(provider.ServerInfo{
	AgentID:       "agent-123",
	Hostname:      "server1.example.com",
	IPAddress:     "192.168.1.10",
	TaskQueues:    []string{"linux-amd64", "linux-common"},
	Status:        "healthy",
	LastHeartbeat: time.Now(),
	Metadata: map[string]string{
		"os":   "linux",
		"arch": "amd64",
	},
})

// 查询服务器组
servers, _ := provider.GetServers(context.Background(), "linux-amd64")
// Returns: [{agent-123, server1.example.com, ...}]
```

### AC3: 配置文件实现 (FileProvider)

**Given** 静态服务器清单  
**When** 服务器组定义不经常变化  
**Then** 提供配置文件实现 `pkg/provider/file_provider.go`:

```go
package provider

import (
	"context"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// FileProvider loads server groups from a YAML configuration file.
type FileProvider struct {
	filePath string
	groups   map[string][]ServerInfo
}

// ServerGroupConfig represents the YAML structure for server groups.
type ServerGroupConfig struct {
	Groups map[string]GroupConfig `yaml:"groups"`
}

type GroupConfig struct {
	Servers []ServerConfig `yaml:"servers"`
}

type ServerConfig struct {
	AgentID    string            `yaml:"agent_id"`
	Hostname   string            `yaml:"hostname"`
	IPAddress  string            `yaml:"ip_address,omitempty"`
	TaskQueues []string          `yaml:"task_queues"`
	Metadata   map[string]string `yaml:"metadata,omitempty"`
}

// NewFileProvider creates a provider from a YAML file.
func NewFileProvider(filePath string) (*FileProvider, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	var config ServerGroupConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	
	// Convert to internal format
	groups := make(map[string][]ServerInfo)
	for groupName, groupConfig := range config.Groups {
		servers := make([]ServerInfo, 0, len(groupConfig.Servers))
		for _, sc := range groupConfig.Servers {
			servers = append(servers, ServerInfo{
				AgentID:       sc.AgentID,
				Hostname:      sc.Hostname,
				IPAddress:     sc.IPAddress,
				TaskQueues:    sc.TaskQueues,
				Status:        "unknown", // File doesn't track real-time status
				LastHeartbeat: time.Time{},
				Metadata:      sc.Metadata,
			})
		}
		groups[groupName] = servers
	}
	
	return &FileProvider{
		filePath: filePath,
		groups:   groups,
	}, nil
}

// GetServers returns servers in the specified group.
func (p *FileProvider) GetServers(ctx context.Context, groupName string) ([]ServerInfo, error) {
	servers, ok := p.groups[groupName]
	if !ok {
		return []ServerInfo{}, nil
	}
	
	// Return a copy
	result := make([]ServerInfo, len(servers))
	copy(result, servers)
	return result, nil
}

// ListGroups returns all group names.
func (p *FileProvider) ListGroups(ctx context.Context) ([]string, error) {
	groups := make([]string, 0, len(p.groups))
	for name := range p.groups {
		groups = append(groups, name)
	}
	return groups, nil
}

// Close is a no-op for file provider.
func (p *FileProvider) Close() error {
	return nil
}
```

**配置文件示例** (`server-groups.yaml`):
```yaml
groups:
  linux-amd64:
    servers:
      - agent_id: agent-001
        hostname: build-server-1.example.com
        ip_address: 192.168.1.10
        task_queues:
          - linux-amd64
          - linux-common
        metadata:
          os: linux
          arch: amd64
          datacenter: us-west
      
      - agent_id: agent-002
        hostname: build-server-2.example.com
        ip_address: 192.168.1.11
        task_queues:
          - linux-amd64
          - linux-common
        metadata:
          os: linux
          arch: amd64
          datacenter: us-east
  
  web-servers:
    servers:
      - agent_id: agent-web-1
        hostname: web-1.example.com
        ip_address: 10.0.1.20
        task_queues:
          - web-servers
        metadata:
          role: web
          environment: production
      
      - agent_id: agent-web-2
        hostname: web-2.example.com
        ip_address: 10.0.1.21
        task_queues:
          - web-servers
        metadata:
          role: web
          environment: production
```

**使用:**
```go
provider, err := provider.NewFileProvider("/etc/waterflow/server-groups.yaml")
if err != nil {
	log.Fatal(err)
}

servers, _ := provider.GetServers(context.Background(), "web-servers")
// Returns: [{agent-web-1, ...}, {agent-web-2, ...}]
```

### AC4: Server 集成 Provider

**Given** ServerGroupProvider 已实现  
**When** Server 启动时  
**Then** 注入 Provider 到 Server 实例

**扩展配置** (`pkg/config/config.go`):
```go
// ServerConfig represents server-specific configuration.
type ServerConfig struct {
	// ... existing fields
	
	// ServerGroupProvider specifies the provider type.
	// Options: "memory" (default), "file", "custom"
	ServerGroupProvider string `mapstructure:"server_group_provider"`
	
	// ServerGroupFile is the path to server groups YAML (if provider=file)
	ServerGroupFile string `mapstructure:"server_group_file"`
}
```

**配置示例** (`config.yaml`):
```yaml
server:
  host: "0.0.0.0"
  port: 8080
  server_group_provider: "file"  # or "memory"
  server_group_file: "/etc/waterflow/server-groups.yaml"
```

**Server 初始化** (`internal/server/server.go`):
```go
type Server struct {
	config             *config.Config
	logger             *zap.Logger
	router             *gin.Engine
	temporalClient     *temporal.Client
	serverGroupProvider provider.ServerGroupProvider // New field
	// ... other fields
}

func New(cfg *config.Config, logger *zap.Logger, version, commit, buildTime string) *Server {
	// ... existing initialization
	
	// Initialize ServerGroupProvider
	var sgProvider provider.ServerGroupProvider
	var err error
	
	switch cfg.Server.ServerGroupProvider {
	case "file":
		if cfg.Server.ServerGroupFile == "" {
			logger.Fatal("server_group_file must be specified when provider=file")
		}
		sgProvider, err = provider.NewFileProvider(cfg.Server.ServerGroupFile)
		if err != nil {
			logger.Fatal("Failed to create file provider", zap.Error(err))
		}
		logger.Info("Using file-based server group provider",
			zap.String("file", cfg.Server.ServerGroupFile),
		)
	
	case "memory":
		fallthrough
	default:
		sgProvider = provider.NewInMemoryProvider()
		logger.Info("Using in-memory server group provider")
	}
	
	return &Server{
		config:             cfg,
		logger:             logger,
		serverGroupProvider: sgProvider,
		// ... other fields
	}
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	// ... existing shutdown logic
	
	// Close provider
	if err := s.serverGroupProvider.Close(); err != nil {
		s.logger.Warn("Failed to close server group provider", zap.Error(err))
	}
	
	return nil
}
```

### AC5: Agent 自动注册到 Provider

**Given** Agent 启动并连接到 Server  
**When** Agent 成功连接到 Temporal  
**Then** Agent 向 Server 注册自己的信息

**注意:** 本 AC 需要 Agent → Server 通信,简化实现中可以:
1. **方案 A (推荐):** Agent 启动时通过 HTTP 调用 Server 注册 API
2. **方案 B:** Server 通过 Temporal Admin API 查询 Worker 信息

**实现方案 A** (Agent 注册 API):

**Server 端 API** (`internal/api/agent_handler.go`):
```go
package api

import (
	"net/http"
	"time"

	"github.com/Websoft9/waterflow/pkg/provider"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RegisterAgentRequest represents the agent registration payload.
type RegisterAgentRequest struct {
	AgentID    string            `json:"agent_id" binding:"required"`
	Hostname   string            `json:"hostname" binding:"required"`
	IPAddress  string            `json:"ip_address"`
	TaskQueues []string          `json:"task_queues" binding:"required,min=1"`
	Metadata   map[string]string `json:"metadata"`
}

// RegisterAgent handles agent registration.
func (h *Handler) RegisterAgent(c *gin.Context) {
	var req RegisterAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}
	
	// Create ServerInfo
	serverInfo := provider.ServerInfo{
		AgentID:       req.AgentID,
		Hostname:      req.Hostname,
		IPAddress:     req.IPAddress,
		TaskQueues:    req.TaskQueues,
		Status:        "healthy",
		LastHeartbeat: time.Now(),
		Metadata:      req.Metadata,
	}
	
	// Register to provider (only works with InMemoryProvider)
	if memProvider, ok := h.serverGroupProvider.(*provider.InMemoryProvider); ok {
		if err := memProvider.RegisterServer(serverInfo); err != nil {
			h.logger.Error("Failed to register agent", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": map[string]interface{}{
					"code":    "registration_failed",
					"message": "Failed to register agent",
				},
			})
			return
		}
	}
	
	h.logger.Info("Agent registered",
		zap.String("agent_id", req.AgentID),
		zap.String("hostname", req.Hostname),
		zap.Strings("task_queues", req.TaskQueues),
	)
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Agent registered successfully",
		"agent_id": req.AgentID,
	})
}
```

**路由注册** (`internal/api/router.go`):
```go
v1 := router.Group("/v1")
{
	// ... existing routes
	
	// Agent management
	v1.POST("/agents/register", handler.RegisterAgent)
	v1.GET("/agents", handler.ListAgents) // Story 2.7
}
```

**Agent 端调用** (`internal/agent/worker.go`):
```go
// Start starts the agent worker.
func (w *Worker) Start() error {
	// ... existing worker startup
	
	// Register to server (if configured)
	if w.config.Agent.ServerURL != "" {
		if err := w.registerToServer(); err != nil {
			w.logger.Warn("Failed to register to server", zap.Error(err))
			// Don't fail startup - registration is optional
		}
	}
	
	return nil
}

// registerToServer registers this agent to the Waterflow server.
func (w *Worker) registerToServer() error {
	hostname, _ := os.Hostname()
	agentID := fmt.Sprintf("agent-%s", uuid.New().String()[:8])
	
	reqBody := map[string]interface{}{
		"agent_id":    agentID,
		"hostname":    hostname,
		"ip_address":  getLocalIP(),
		"task_queues": w.config.Agent.TaskQueues,
		"metadata": map[string]string{
			"os":      runtime.GOOS,
			"arch":    runtime.GOARCH,
			"version": w.version,
		},
	}
	
	jsonData, _ := json.Marshal(reqBody)
	
	resp, err := http.Post(
		w.config.Agent.ServerURL+"/v1/agents/register",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to send registration: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed with status %d", resp.StatusCode)
	}
	
	w.logger.Info("Registered to server",
		zap.String("agent_id", agentID),
		zap.String("server_url", w.config.Agent.ServerURL),
	)
	
	return nil
}

func getLocalIP() string {
	// Implementation to get local IP address
	// ... (simplified for brevity)
	return ""
}
```

**Agent 配置扩展** (`pkg/config/config.go`):
```go
type AgentConfig struct {
	// ... existing fields
	
	// ServerURL is the Waterflow server URL (for registration)
	// Example: "http://localhost:8080"
	// Optional: If empty, agent won't register
	ServerURL string `mapstructure:"server_url"`
}
```

### AC6: CMDB 集成示例和文档

**Given** 企业用户有自定义 CMDB  
**When** 需要集成 Waterflow  
**Then** 提供集成示例和文档

**创建集成示例** (`examples/providers/ansible_provider.go`):
```go
package examples

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/Websoft9/waterflow/pkg/provider"
)

// AnsibleInventoryProvider integrates with Ansible inventory.
// This is an example implementation showing how to create custom providers.
type AnsibleInventoryProvider struct {
	inventoryPath string
}

// NewAnsibleInventoryProvider creates a provider from Ansible inventory.
func NewAnsibleInventoryProvider(inventoryPath string) *AnsibleInventoryProvider {
	return &AnsibleInventoryProvider{
		inventoryPath: inventoryPath,
	}
}

// GetServers queries Ansible inventory for a specific group.
func (p *AnsibleInventoryProvider) GetServers(ctx context.Context, groupName string) ([]provider.ServerInfo, error) {
	// Execute: ansible-inventory -i <inventory> --list --export
	cmd := exec.CommandContext(ctx, "ansible-inventory",
		"-i", p.inventoryPath,
		"--list",
		"--export",
	)
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query ansible inventory: %w", err)
	}
	
	// Parse JSON output
	var inventory map[string]interface{}
	if err := json.Unmarshal(output, &inventory); err != nil {
		return nil, fmt.Errorf("failed to parse inventory: %w", err)
	}
	
	// Extract hosts from group
	groupData, ok := inventory[groupName].(map[string]interface{})
	if !ok {
		return []provider.ServerInfo{}, nil
	}
	
	hosts, ok := groupData["hosts"].([]interface{})
	if !ok {
		return []provider.ServerInfo{}, nil
	}
	
	// Convert to ServerInfo
	servers := make([]provider.ServerInfo, 0, len(hosts))
	for _, host := range hosts {
		hostname := host.(string)
		servers = append(servers, provider.ServerInfo{
			AgentID:    fmt.Sprintf("ansible-%s", hostname),
			Hostname:   hostname,
			TaskQueues: []string{groupName},
			Status:     "unknown",
			Metadata: map[string]string{
				"source": "ansible-inventory",
			},
		})
	}
	
	return servers, nil
}

// ListGroups returns all groups in Ansible inventory.
func (p *AnsibleInventoryProvider) ListGroups(ctx context.Context) ([]string, error) {
	// Similar implementation using ansible-inventory --graph
	// ... (simplified for example)
	return []string{}, nil
}

// Close releases resources.
func (p *AnsibleInventoryProvider) Close() error {
	return nil
}
```

**集成文档** (`docs/guides/cmdb-integration.md`):
```markdown
# CMDB 集成指南

## 概述

Waterflow 通过 `ServerGroupProvider` 接口支持集成外部 CMDB 系统。

## 内置 Provider

### 1. InMemoryProvider (默认)

适用场景: 测试、小规模部署

配置:
\`\`\`yaml
server:
  server_group_provider: memory
\`\`\`

特点:
- Agent 启动时通过 API 注册
- 信息存储在内存中
- Server 重启后丢失

### 2. FileProvider

适用场景: 静态服务器清单

配置:
\`\`\`yaml
server:
  server_group_provider: file
  server_group_file: /etc/waterflow/server-groups.yaml
\`\`\`

特点:
- YAML 配置文件定义服务器组
- 适合服务器组不经常变化的场景

## 自定义 Provider

### 接口定义

实现 `ServerGroupProvider` 接口:
\`\`\`go
type ServerGroupProvider interface {
	GetServers(ctx context.Context, groupName string) ([]ServerInfo, error)
	ListGroups(ctx context.Context) ([]string, error)
	Close() error
}
\`\`\`

### 示例: Ansible Inventory 集成

参考 `examples/providers/ansible_provider.go`

### 示例: 数据库集成

\`\`\`go
type DatabaseProvider struct {
	db *sql.DB
}

func (p *DatabaseProvider) GetServers(ctx context.Context, groupName string) ([]ServerInfo, error) {
	query := "SELECT agent_id, hostname, ip_address FROM servers WHERE group_name = ?"
	// ... execute query and return results
}
\`\`\`

## 最佳实践

1. **缓存查询结果** - CMDB 查询可能较慢,考虑缓存
2. **错误处理** - 优雅处理 CMDB 不可用的情况
3. **超时控制** - 使用 context 实现超时
4. **日志记录** - 记录所有 CMDB 交互
```

### AC7: 单元测试

**Given** ServerGroupProvider 实现  
**When** 运行测试  
**Then** 测试覆盖率 >80%

**测试文件** (`pkg/provider/memory_provider_test.go`):
```go
package provider

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryProvider_RegisterServer(t *testing.T) {
	provider := NewInMemoryProvider()
	
	server := ServerInfo{
		AgentID:       "agent-1",
		Hostname:      "server1",
		TaskQueues:    []string{"linux-amd64", "linux-common"},
		Status:        "healthy",
		LastHeartbeat: time.Now(),
	}
	
	err := provider.RegisterServer(server)
	require.NoError(t, err)
	
	// Verify registered in both groups
	servers, err := provider.GetServers(context.Background(), "linux-amd64")
	require.NoError(t, err)
	assert.Len(t, servers, 1)
	assert.Equal(t, "agent-1", servers[0].AgentID)
	
	servers, err = provider.GetServers(context.Background(), "linux-common")
	require.NoError(t, err)
	assert.Len(t, servers, 1)
}

func TestInMemoryProvider_GetServers_EmptyGroup(t *testing.T) {
	provider := NewInMemoryProvider()
	
	servers, err := provider.GetServers(context.Background(), "non-existent")
	require.NoError(t, err)
	assert.Empty(t, servers)
}

func TestInMemoryProvider_ListGroups(t *testing.T) {
	provider := NewInMemoryProvider()
	
	provider.RegisterServer(ServerInfo{
		AgentID:    "agent-1",
		TaskQueues: []string{"group-a", "group-b"},
	})
	
	groups, err := provider.ListGroups(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"group-a", "group-b"}, groups)
}

// Performance benchmarks
func BenchmarkInMemoryProvider_GetServers_10000Agents(b *testing.B) {
	provider := NewInMemoryProvider()
	
	// Register 10000 agents
	for i := 0; i < 10000; i++ {
		provider.RegisterServer(ServerInfo{
			AgentID:    fmt.Sprintf("agent-%d", i),
			TaskQueues: []string{"linux-amd64"},
		})
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.GetServers(context.Background(), "linux-amd64")
		if err != nil {
			b.Fatal(err)
		}
	}
	// Expected: < 10ms per operation
}

func BenchmarkFileProvider_GetServers(b *testing.B) {
	provider, _ := NewFileProvider("testdata/servers.yaml")
	defer provider.Close()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.GetServers(context.Background(), "linux-amd64")
		if err != nil {
			b.Fatal(err)
		}
	}
	// Expected: < 100ms per operation
}
```

**性能目标验证:**
```bash
# 运行性能测试
go test -bench=. -benchmem ./pkg/provider/

# 预期输出:
BenchmarkInMemoryProvider_GetServers_10000Agents-8   50000   8234 ns/op  ✅ < 10ms
BenchmarkFileProvider_GetServers-8                   10000   89456 ns/op ✅ < 100ms
```

## Developer Context

### 架构概览

```
┌──────────────────────────────────────────────────────────────┐
│                    Waterflow Server                          │
│  ┌────────────────┐          ┌──────────────────┐           │
│  │ API Handler    │─────────→│ ServerGroup      │           │
│  │ (List Agents)  │          │ Provider         │           │
│  └────────────────┘          │ (Interface)      │           │
│                               └────────┬─────────┘           │
└────────────────────────────────────────┼─────────────────────┘
                                         │
                 ┌───────────────────────┼───────────────────────┐
                 ↓                       ↓                       ↓
        ┌────────────────┐      ┌────────────────┐  ┌────────────────┐
        │ InMemory       │      │ File           │  │ Custom CMDB    │
        │ Provider       │      │ Provider       │  │ Provider       │
        │                │      │                │  │                │
        │ - HTTP API     │      │ - YAML file    │  │ - Ansible      │
        │   Registration │      │ - Auto-reload  │  │ - Terraform    │
        │ - Fast lookup  │      │ - Git-friendly │  │ - REST API     │
        └────────────────┘      └────────────────┘  └────────────────┘
```

### 架构图 (详细)

```
┌─────────────────────────────────────────────────────────────┐
│                  Waterflow Server                           │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         ServerGroupProvider Interface                 │  │
│  │                                                        │  │
│  │  • GetServers(groupName) → []ServerInfo               │  │
│  │  • ListGroups() → []string                            │  │
│  │  • Close()                                            │  │
│  └──────────────────────────────────────────────────────┘  │
│           ▲              ▲                ▲                │
│           │              │                │                │
└───────────┼──────────────┼────────────────┼────────────────┘
            │              │                │
    ┌───────┴───────┐ ┌────┴─────┐  ┌──────┴──────┐
    │  Memory       │ │  File    │  │   Custom    │
    │  Provider     │ │ Provider │  │  (Ansible,  │
    │               │ │          │  │   CMDB)     │
    │ - In-memory   │ │ - YAML   │  │ - External  │
    │   storage     │ │   file   │  │   API calls │
    └───────────────┘ └──────────┘  └─────────────┘
            ▲
            │ Register
    ┌───────┴────────┐
    │  Agent Workers │
    │                │
    │ POST /v1/      │
    │ agents/        │
    │ register       │
    └────────────────┘
```

### 设计决策

1. **接口优先** - 定义清晰的接口,易于扩展
2. **可选功能** - Provider 是可选的,不影响核心路由功能
3. **默认简单** - InMemoryProvider 作为默认,零配置
4. **企业扩展** - 通过自定义 Provider 集成 CMDB

### 使用场景

| 场景 | 推荐 Provider | 说明 |
|------|---------------|------|
| 开发/测试 | InMemoryProvider | 简单,无需配置 |
| 小规模生产 | FileProvider | 静态配置文件 |
| 企业部署 | Custom Provider | 集成现有 CMDB/Ansible |
| 动态环境 | Custom Provider | 从云 API 查询实例 |

### 与其他 Story 的关系

- Story 2.2 提供了 Task Queue 路由机制
- Story 2.3 (本 Story) 提供了服务器组信息查询
- Story 2.7 将使用 Provider 实现健康监控 API

### 可选性说明

**ServerGroupProvider 是可选的增强功能:**
- 核心路由 (Story 2.2) 不依赖 Provider
- Agent 可以直接启动,无需注册到 Server
- Provider 主要用于:
  - Server 查询可用 Agent 列表
  - 健康监控和状态查询
  - 企业 CMDB 集成

## Dev Notes

### 实现优先级

**必须实现:**
- ✅ ServerGroupProvider 接口定义
- ✅ InMemoryProvider 实现
- ✅ FileProvider 实现
- ✅ Server 集成 Provider
- ✅ 单元测试

**可选实现 (MVP 后):**
- Agent 自动注册 API (简化可先手动配置)
- CMDB 集成示例 (文档说明即可)

### 测试策略

```bash
# 单元测试
go test -v ./pkg/provider/...

# 集成测试
# 1. 启动 Server (memory provider)
bin/server --config config.yaml

# 2. 手动注册 Agent (模拟)
curl -X POST http://localhost:8080/v1/agents/register \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "agent-test",
    "hostname": "test-server",
    "task_queues": ["linux-amd64"]
  }'

# 3. 查询服务器组 (Story 2.7)
curl http://localhost:8080/v1/agents
```

## Dev Agent Record

### File List

**新增文件:**
- `pkg/provider/server_group.go` - 接口定义 (~60 行)
- `pkg/provider/memory_provider.go` - 内存实现 (~150 行)
- `pkg/provider/file_provider.go` - 文件实现 (~120 行)
- `pkg/provider/memory_provider_test.go` - 测试 (~100 行)
- `internal/api/agent_handler.go` - Agent 注册 API (~80 行)
- `examples/providers/ansible_provider.go` - Ansible 示例 (~100 行)
- `docs/guides/cmdb-integration.md` - 集成文档 (~150 行)
- `server-groups.example.yaml` - 配置示例 (~50 行)

**修改文件:**
- `pkg/config/config.go` - 扩展配置 (+20 行)
- `internal/server/server.go` - 集成 Provider (+40 行)
- `internal/api/router.go` - 注册路由 (+5 行)
- `internal/agent/worker.go` - Agent 注册逻辑 (+60 行)

**总计:** ~730 新增代码行,~125 修改行
