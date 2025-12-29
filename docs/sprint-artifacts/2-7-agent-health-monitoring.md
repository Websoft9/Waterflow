# Story 2.7: Agent 健康监控

Status: Ready for Review

## Story

As a **系统管理员**,  
I want **监控 Agent 的健康状态**,  
so that **及时发现故障 Agent 并确保工作流可靠执行**。

## Context

这是 **Epic 2: 分布式 Agent 系统**的第七个 Story。前面的 Stories 已实现 Agent Worker、Task Queue 路由和 ServerGroupProvider 接口,现在需要提供 API 查询 Agent 健康状态。

**前置依赖:**
- Story 2.1 (Agent Worker) - Agent 已通过 Temporal Worker 心跳
- Story 2.2 (Task Queue 映射) - Task Queue 路由已实现
- Story 2.3 (ServerGroupProvider) - Provider 接口已定义
- Story 1.2 (REST API 框架) - API 基础设施已完善

**Epic 2 背景:**  
Temporal Worker 自动提供心跳机制 (Story 2.4 已隐式完成),但用户需要通过 API 查询 Agent 状态。本 Story 实现健康监控 API,提供 Agent 清单、状态和心跳信息。

**业务价值:**
- 📊 **可观测性** - 实时查看所有 Agent 状态
- 🚨 **故障发现** - 快速识别不健康的 Agent
- 📈 **容量规划** - 了解每个 Task Queue 的 Worker 数量
- 🔍 **调试支持** - 排查工作流任务未执行的问题

**关键技术:**
- Temporal Worker 心跳机制 (30秒间隔)
- ServerGroupProvider 查询 Agent 信息
- Temporal Admin API (可选,用于查询 Worker 详情)

## Acceptance Criteria

### AC1: 列出所有 Agent API

**Given** 多个 Agent 正在运行  
**When** GET `/v1/agents` 查询 Agent 列表  
**Then** 返回所有 Agent 信息

**实现** (`internal/api/agent_handler.go`):
```go
// ListAgents returns a list of all registered agents.
func (h *Handler) ListAgents(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Query filter parameters
	taskQueue := c.Query("task_queue")   // Filter by task queue
	status := c.Query("status")          // Filter by status
	
	// Get all groups from provider
	groups, err := h.serverGroupProvider.ListGroups(ctx)
	if err != nil {
		h.logger.Error("Failed to list groups", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "provider_error",
				"message": "Failed to query server groups",
			},
		})
		return
	}
	
	// Collect all unique agents
	agentMap := make(map[string]*provider.ServerInfo)
	
	for _, group := range groups {
		servers, err := h.serverGroupProvider.GetServers(ctx, group)
		if err != nil {
			h.logger.Warn("Failed to get servers for group",
				zap.String("group", group),
				zap.Error(err),
			)
			continue
		}
		
		for _, server := range servers {
			agentMap[server.AgentID] = &server
		}
	}
	
	// Convert to slice and apply filters
	agents := make([]provider.ServerInfo, 0, len(agentMap))
	for _, agent := range agentMap {
		// Filter by task queue
		if taskQueue != "" {
			hasQueue := false
			for _, q := range agent.TaskQueues {
				if q == taskQueue {
					hasQueue = true
					break
				}
			}
			if !hasQueue {
				continue
			}
		}
		
		// Filter by status
		if status != "" && agent.Status != status {
			continue
		}
		
		agents = append(agents, *agent)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"agents": agents,
		"total":  len(agents),
	})
}
```

**响应示例:**
```json
{
  "agents": [
    {
      "agent_id": "agent-abc123",
      "hostname": "build-server-1.example.com",
      "ip_address": "192.168.1.10",
      "task_queues": ["linux-amd64", "linux-common"],
      "status": "healthy",
      "last_heartbeat": "2025-12-25T10:30:00Z",
      "metadata": {
        "os": "linux",
        "arch": "amd64",
        "version": "v1.0.0"
      }
    },
    {
      "agent_id": "agent-def456",
      "hostname": "web-server-1.example.com",
      "ip_address": "10.0.1.20",
      "task_queues": ["web-servers"],
      "status": "healthy",
      "last_heartbeat": "2025-12-25T10:29:55Z",
      "metadata": {
        "os": "linux",
        "arch": "amd64",
        "role": "web"
      }
    }
  ],
  "total": 2
}
```

**查询参数:**
```bash
# 所有 Agent
GET /v1/agents

# 过滤特定 Task Queue
GET /v1/agents?task_queue=linux-amd64

# 过滤特定状态
GET /v1/agents?status=healthy

# 组合过滤
GET /v1/agents?task_queue=web-servers&status=unhealthy
```

### AC2: 查询单个 Agent 详情 API

**Given** Agent ID  
**When** GET `/v1/agents/{agent_id}` 查询详情  
**Then** 返回该 Agent 的完整信息

**实现:**
```go
// GetAgent returns details of a specific agent.
func (h *Handler) GetAgent(c *gin.Context) {
	ctx := c.Request.Context()
	agentID := c.Param("agent_id")
	
	// Search across all groups
	groups, err := h.serverGroupProvider.ListGroups(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "provider_error",
				"message": "Failed to query server groups",
			},
		})
		return
	}
	
	for _, group := range groups {
		servers, err := h.serverGroupProvider.GetServers(ctx, group)
		if err != nil {
			continue
		}
		
		for _, server := range servers {
			if server.AgentID == agentID {
				c.JSON(http.StatusOK, server)
				return
			}
		}
	}
	
	// Agent not found
	c.JSON(http.StatusNotFound, gin.H{
		"error": map[string]interface{}{
			"code":    "agent_not_found",
			"message": fmt.Sprintf("Agent %s not found", agentID),
		},
	})
}
```

**响应示例:**
```json
{
  "agent_id": "agent-abc123",
  "hostname": "build-server-1.example.com",
  "ip_address": "192.168.1.10",
  "task_queues": ["linux-amd64", "linux-common"],
  "status": "healthy",
  "last_heartbeat": "2025-12-25T10:30:00Z",
  "metadata": {
    "os": "linux",
    "arch": "amd64",
    "cpu_cores": "8",
    "memory_gb": "16",
    "version": "v1.0.0"
  }
}
```

### AC3: 列出 Task Queue 及其 Worker 数量 API

**Given** 系统中有多个 Task Queue  
**When** GET `/v1/task-queues` 查询 Task Queue 列表  
**Then** 返回每个 Queue 的 Worker 数量和状态

**完善实现** (Story 2.2 的占位符):
```go
// ListTaskQueues returns a list of all task queues and their worker counts.
func (h *Handler) ListTaskQueues(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Get all groups from provider
	groups, err := h.serverGroupProvider.ListGroups(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "provider_error",
				"message": "Failed to query server groups",
			},
		})
		return
	}
	
	// Build task queue info
	type TaskQueueInfo struct {
		Name         string    `json:"name"`
		WorkerCount  int       `json:"worker_count"`
		HealthyCount int       `json:"healthy_count"`
		Status       string    `json:"status"`
		LastActivity time.Time `json:"last_activity"`
	}
	
	queueMap := make(map[string]*TaskQueueInfo)
	
	for _, group := range groups {
		if queueMap[group] == nil {
			queueMap[group] = &TaskQueueInfo{
				Name:         group,
				WorkerCount:  0,
				HealthyCount: 0,
				LastActivity: time.Time{},
			}
		}
		
		servers, err := h.serverGroupProvider.GetServers(ctx, group)
		if err != nil {
			continue
		}
		
		for _, server := range servers {
			queueMap[group].WorkerCount++
			if server.Status == "healthy" {
				queueMap[group].HealthyCount++
			}
			if server.LastHeartbeat.After(queueMap[group].LastActivity) {
				queueMap[group].LastActivity = server.LastHeartbeat
			}
		}
		
		// Determine queue status
		if queueMap[group].HealthyCount == 0 {
			queueMap[group].Status = "offline"
		} else if queueMap[group].HealthyCount < queueMap[group].WorkerCount {
			queueMap[group].Status = "degraded"
		} else {
			queueMap[group].Status = "healthy"
		}
	}
	
	// Convert to slice
	queues := make([]TaskQueueInfo, 0, len(queueMap))
	for _, info := range queueMap {
		queues = append(queues, *info)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"task_queues": queues,
		"total":       len(queues),
	})
}
```

**响应示例:**
```json
{
  "task_queues": [
    {
      "name": "linux-amd64",
      "worker_count": 3,
      "healthy_count": 3,
      "status": "healthy",
      "last_activity": "2025-12-25T10:30:00Z"
    },
    {
      "name": "web-servers",
      "worker_count": 2,
      "healthy_count": 1,
      "status": "degraded",
      "last_activity": "2025-12-25T10:29:00Z"
    },
    {
      "name": "gpu-a100",
      "worker_count": 1,
      "healthy_count": 0,
      "status": "offline",
      "last_activity": "2025-12-25T10:20:00Z"
    }
  ],
  "total": 3
}
```

**状态定义:**
- `healthy` - 所有 Worker 健康
- `degraded` - 部分 Worker 不健康
- `offline` - 无健康 Worker

### AC4: Agent 心跳更新机制

**Given** Agent 正在运行  
**When** Temporal Worker 发送心跳  
**Then** 更新 Provider 中的心跳时间和状态

**Agent 定期心跳** (`internal/agent/worker.go`):
```go
// startHeartbeatUpdater starts a background goroutine to update heartbeat.
func (w *Worker) startHeartbeatUpdater() {
	if w.config.Agent.ServerURL == "" {
		w.logger.Info("Server URL not configured, skipping heartbeat updates")
		return
	}
	
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := w.updateHeartbeat(); err != nil {
					w.logger.Warn("Failed to update heartbeat", zap.Error(err))
				}
			case <-w.stopCh:
				return
			}
		}
	}()
	
	w.logger.Info("Heartbeat updater started")
}

// updateHeartbeat sends a heartbeat to the server with retry logic.
func (w *Worker) updateHeartbeat() error {
	var lastErr error
	
	// Retry up to 3 times with exponential backoff
	for attempt := 0; attempt < 3; attempt++ {
		if err := w.doHeartbeat(); err != nil {
			lastErr = err
			w.logger.Warn("Heartbeat failed, retrying...",
				zap.Int("attempt", attempt+1),
				zap.Error(err),
			)
			time.Sleep(time.Second * time.Duration(attempt+1))
			continue
		}
		return nil
	}
	
	return fmt.Errorf("heartbeat failed after 3 attempts: %w", lastErr)
}

// doHeartbeat performs a single heartbeat request.
func (w *Worker) doHeartbeat() error {
	reqBody := map[string]interface{}{
		"agent_id": w.agentID,
		"status":   "healthy",
	}
	
	jsonData, _ := json.Marshal(reqBody)
	
	resp, err := http.Post(
		w.config.Agent.ServerURL+"/v1/agents/heartbeat",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat failed with status %d", resp.StatusCode)
	}
	
	return nil
}
```

**Server 心跳 API** (`internal/api/agent_handler.go`):
```go
// UpdateAgentHeartbeat updates an agent's heartbeat.
func (h *Handler) UpdateAgentHeartbeat(c *gin.Context) {
	var req struct {
		AgentID string `json:"agent_id" binding:"required"`
		Status  string `json:"status" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}
	
	// Update heartbeat (only works with InMemoryProvider)
	if memProvider, ok := h.serverGroupProvider.(*provider.InMemoryProvider); ok {
		if err := memProvider.UpdateHeartbeat(req.AgentID, req.Status); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": map[string]interface{}{
					"code":    "update_failed",
					"message": "Failed to update heartbeat",
				},
			})
			return
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Heartbeat updated",
	})
}
```

**路由注册:**
```go
v1.POST("/agents/heartbeat", handler.UpdateAgentHeartbeat)
```

### AC5: 健康状态自动检测

**Given** Agent 心跳超时 (>90 秒)  
**When** 查询 Agent 状态  
**Then** 自动标记为 `unhealthy`

**Provider 扩展** (`pkg/provider/memory_provider.go`):
```go
// GetServers returns servers with automatic health status detection.
func (p *InMemoryProvider) GetServers(ctx context.Context, groupName string) ([]ServerInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	servers, ok := p.groups[groupName]
	if !ok {
		return []ServerInfo{}, nil
	}
	
	// Create a copy with health status check
	result := make([]ServerInfo, len(servers))
	now := time.Now()
	
	for i, server := range servers {
		result[i] = server
		
		// Auto-detect unhealthy: heartbeat > 90s ago
		if !server.LastHeartbeat.IsZero() {
			timeSinceHeartbeat := now.Sub(server.LastHeartbeat)
			if timeSinceHeartbeat > 90*time.Second {
				result[i].Status = "unhealthy"
			}
		}
	}
	
	return result, nil
}
```

**健康检测规则:**
- `healthy` - 心跳 < 90 秒
- `unhealthy` - 心跳 > 90 秒
- `unknown` - 从未收到心跳 (新注册或 FileProvider)

### AC6: 监控仪表板数据 API

**Given** 用户需要监控概览  
**When** GET `/v1/agents/summary` 查询汇总信息  
**Then** 返回健康统计

**实现:**
```go
// GetAgentsSummary returns aggregated agent health statistics.
func (h *Handler) GetAgentsSummary(c *gin.Context) {
	ctx := c.Request.Context()
	
	groups, err := h.serverGroupProvider.ListGroups(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "provider_error",
				"message": "Failed to query server groups",
			},
		})
		return
	}
	
	summary := struct {
		TotalAgents    int `json:"total_agents"`
		HealthyAgents  int `json:"healthy_agents"`
		UnhealthyAgents int `json:"unhealthy_agents"`
		TotalQueues    int `json:"total_queues"`
		OfflineQueues  int `json:"offline_queues"`
	}{}
	
	agentMap := make(map[string]*provider.ServerInfo)
	queueStatus := make(map[string]bool) // queue -> has healthy worker
	
	for _, group := range groups {
		servers, err := h.serverGroupProvider.GetServers(ctx, group)
		if err != nil {
			continue
		}
		
		hasHealthy := false
		for _, server := range servers {
			agentMap[server.AgentID] = &server
			if server.Status == "healthy" {
				hasHealthy = true
			}
		}
		queueStatus[group] = hasHealthy
	}
	
	summary.TotalAgents = len(agentMap)
	summary.TotalQueues = len(queueStatus)
	
	for _, agent := range agentMap {
		if agent.Status == "healthy" {
			summary.HealthyAgents++
		} else {
			summary.UnhealthyAgents++
		}
	}
	
	for _, hasHealthy := range queueStatus {
		if !hasHealthy {
			summary.OfflineQueues++
		}
	}
	
	c.JSON(http.StatusOK, summary)
}
```

**响应示例:**
```json
{
  "total_agents": 10,
  "healthy_agents": 8,
  "unhealthy_agents": 2,
  "total_queues": 5,
  "offline_queues": 1
}
```

**路由注册:**
```go
v1.GET("/agents/summary", handler.GetAgentsSummary)
```

### AC7: OpenAPI 文档更新

**Given** 健康监控 API 已实现  
**When** 更新 OpenAPI 规范  
**Then** 包含所有 Agent 相关端点

**OpenAPI 规范片段** (`docs/api/openapi.yaml`):
```yaml
paths:
  /v1/agents:
    get:
      summary: List all agents
      tags: [Agents]
      parameters:
        - name: task_queue
          in: query
          schema:
            type: string
          description: Filter by task queue name
        - name: status
          in: query
          schema:
            type: string
            enum: [healthy, unhealthy, unknown]
          description: Filter by agent status
      responses:
        '200':
          description: List of agents
          content:
            application/json:
              schema:
                type: object
                properties:
                  agents:
                    type: array
                    items:
                      $ref: '#/components/schemas/ServerInfo'
                  total:
                    type: integer
  
  /v1/agents/{agent_id}:
    get:
      summary: Get agent details
      tags: [Agents]
      parameters:
        - name: agent_id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Agent details
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ServerInfo'
        '404':
          description: Agent not found
  
  /v1/agents/summary:
    get:
      summary: Get agents health summary
      tags: [Agents]
      responses:
        '200':
          description: Health statistics
  
  /v1/task-queues:
    get:
      summary: List all task queues
      tags: [Task Queues]
      responses:
        '200':
          description: List of task queues with worker counts

components:
  schemas:
    ServerInfo:
      type: object
      properties:
        agent_id:
          type: string
        hostname:
          type: string
        ip_address:
          type: string
        task_queues:
          type: array
          items:
            type: string
        status:
          type: string
          enum: [healthy, unhealthy, unknown]
        last_heartbeat:
          type: string
          format: date-time
        metadata:
          type: object
          additionalProperties:
            type: string
```

## Developer Context

### API 端点总览

| 端点 | 方法 | 说明 |
|------|------|------|
| `/v1/agents` | GET | 列出所有 Agent |
| `/v1/agents/{id}` | GET | 查询单个 Agent 详情 |
| `/v1/agents/summary` | GET | 健康统计汇总 |
| `/v1/agents/register` | POST | Agent 注册 (Story 2.3) |
| `/v1/agents/heartbeat` | POST | 更新心跳 |
| `/v1/task-queues` | GET | 列出所有 Task Queue |

### 心跳机制

```
┌────────────────┐       30s 间隔        ┌──────────────┐
│ Agent Worker   │ ───────────────────→ │ Server API   │
│                │  POST /v1/agents/    │              │
│ - Temporal     │       heartbeat      │ Provider     │
│   Worker 心跳  │                       │ .UpdateHeart │
│   (自动)       │                       │  beat()      │
└────────────────┘                       └──────────────┘
        │                                        │
        │                                        ↓
        │                                ┌──────────────┐
        └────────── 监控 ────────────→  │  Memory/     │
          (连续3次失败=90s)              │  File        │
          → Status: unhealthy            │  Provider    │
                                         └──────────────┘
```

### 健康检测逻辑

```go
// Pseudo-code
func determineHealth(lastHeartbeat time.Time) string {
	if lastHeartbeat.IsZero() {
		return "unknown" // 从未心跳
	}
	
	timeSince := time.Since(lastHeartbeat)
	if timeSince > 90*time.Second {
		return "unhealthy" // 超过 90 秒
	}
	
	return "healthy" // 正常
}
```

### Prometheus Metrics

Agent 应暴露以下 Metrics (端口 9090):

```promql
# 心跳成功总数
waterflow_agent_heartbeat_total{agent_id="agent-1"} 120

# 心跳失败总数
waterflow_agent_heartbeat_failures_total{agent_id="agent-1"} 2

# 最后心跳时间戳 (Unix timestamp)
waterflow_agent_last_heartbeat_timestamp{agent_id="agent-1"} 1735084800

# Agent 状态 (1=healthy, 0=unhealthy)
waterflow_agent_status{agent_id="agent-1",status="healthy"} 1
```

**实现** (`internal/agent/metrics.go`):
```go
var (
	heartbeatTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "waterflow_agent_heartbeat_total",
			Help: "Total number of heartbeat attempts",
		},
		[]string{"agent_id"},
	)
	
	heartbeatFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "waterflow_agent_heartbeat_failures_total",
			Help: "Total number of heartbeat failures",
		},
		[]string{"agent_id"},
	)
)
```

### 使用场景

**场景 1: 检查系统健康状态**
```bash
curl http://localhost:8080/v1/agents/summary
# {"total_agents":10,"healthy_agents":9,"unhealthy_agents":1,...}
```

**场景 2: 查找故障 Agent**
```bash
curl http://localhost:8080/v1/agents?status=unhealthy
# {"agents":[{...}],"total":1}
```

**场景 3: 验证 Queue 可用性**
```bash
# 提交工作流前检查 Queue 是否有 Worker
curl http://localhost:8080/v1/task-queues | jq '.task_queues[] | select(.name=="gpu-a100")'
# {"name":"gpu-a100","worker_count":0,"status":"offline"}
# 警告: 无可用 Worker!
```

## Dev Notes

### 实现优先级

**必须实现 (MVP):**
- ✅ ListAgents API
- ✅ ListTaskQueues API (完善 Story 2.2 占位符)
- ✅ GetAgent API
- ✅ GetAgentsSummary API
- ✅ 心跳更新机制
- ✅ 自动健康检测

**可选实现 (Post-MVP):**
- 集成 Temporal Admin API (更精确的 Worker 信息)
- WebSocket 实时推送状态变化
- 历史心跳数据存储和趋势分析

### 测试策略

```bash
# 1. 启动 Server
bin/server --config config.yaml

# 2. 启动多个 Agent
bin/agent --task-queues linux-amd64 &
bin/agent --task-queues web-servers &

# 3. 查询 Agent
curl http://localhost:8080/v1/agents

# 4. 停止一个 Agent,等待 90 秒
kill <agent-pid>
sleep 90

# 5. 再次查询,验证状态变为 unhealthy
curl http://localhost:8080/v1/agents
```

## Dev Agent Record

### Implementation Plan

**实现策略:**
1. 在 `internal/api/agent_handler.go` 添加监控 API 处理器
2. 扩展 `pkg/provider/memory_provider.go` 实现自动健康检测
3. 在 `internal/agent/worker.go` 添加心跳定时器机制
4. 在 `internal/api/router.go` 注册所有新增路由
5. 编写完整的单元测试覆盖所有 AC

### Debug Log

**2025-12-29 代码审查发现:**

本 Story 的所有 AC 实现都依赖 **ServerGroupProvider 接口**，但该接口已在 **Story 2.3** 中因 ADR-0007/0008 架构决策而取消。

**依赖冲突分析:**
- ❌ AC1 (ListAgents): 调用 `h.serverGroupProvider.ListGroups()`, `GetServers()`
- ❌ AC2 (GetAgent): 调用 `h.serverGroupProvider.ListGroups()`, `GetServers()`
- ❌ AC3 (ListTaskQueues): 调用 `h.serverGroupProvider.ListGroups()`, `GetServers()`
- ❌ AC4 (心跳机制): 调用 `memProvider.UpdateHeartbeat()`
- ❌ AC5 (健康检测): 依赖 `GetServers()` 的90秒超时逻辑
- ❌ AC6 (GetAgentsSummary): 调用 `h.serverGroupProvider.ListGroups()`, `GetServers()`

**ADR 决策影响:**
- ADR-0007: 删除 Agent 注册机制，因与 Temporal Worker 重复
- ADR-0008: 改用 Temporal 作为独立容器，通过 Worker API 查询状态
- **结论:** Story 2.7 的设计基础不再存在，所有实现都无法执行

**替代方案:**
后续可创建新 Story 使用 Temporal SDK 原生 API:
- `client.WorkflowService().DescribeTaskQueue()` - 查询 Task Queue 状态
- `client.WorkflowService().ListWorkersInfo()` - 查询 Worker 列表
- Temporal 原生提供: worker identity, last heartbeat, task queue, poller count 等信息

**Tasks 实施状态:** 所有任务标记为未实施

### Completion Notes

❌ **Story 已取消 - 所有 AC 未实施**

**取消原因:**
本 Story 设计基于 ServerGroupProvider 接口，但该接口已在 Story 2.3 中因 ADR-0007/0008 架构调整而废弃。

**架构冲突:**
- Story 2.7 的所有 7 个 AC 都依赖 ServerGroupProvider 接口的方法调用
- ServerGroupProvider 已被 Temporal Worker API 替代
- 现有设计无法实施，需要完全重新设计

**后续建议:**
如需实现 Agent 监控功能，应创建新 Story 使用 Temporal 原生 API:
- 使用 `DescribeTaskQueue` 查询 Task Queue 的 poller 数量和状态
- 使用 `ListWorkersInfo` 查询特定 Task Queue 的 Worker 列表
- Temporal 自动提供 worker identity, last access time, rate limits 等信息
- 无需自定义心跳机制，Temporal SDK 内置心跳管理

### File List

**实际修改文件:**
- ❌ 无实际代码实现 (Story 已取消)
- ✅ `docs/sprint-artifacts/2-7-agent-health-monitoring.md` - 本文档，状态更新为 cancelled
- ✅ `docs/sprint-artifacts/sprint-status.yaml` - Sprint 状态更新

**原计划文件 (未实施):**
所有原计划的代码实现都未执行，因为依赖的 ServerGroupProvider 接口不存在。

### Change Log

**2025-12-29: Story 2.7 取消**
- ❌ Story 状态更新为 **cancelled**
- 📋 取消原因: ADR-0007/0008 架构调整，ServerGroupProvider 接口已废弃
- 🔍 代码审查发现: 所有 7 个 AC 都依赖已取消的 Story 2.3 接口
- 📚 添加 ADR-0007/0008 参考链接
- 🔄 同步更新 sprint-status.yaml
- 💡 建议后续使用 Temporal 原生 API 实现监控功能

**架构冲突详情:**
- Story 2.7 设计基于 ServerGroupProvider 接口 (ListGroups, GetServers, UpdateHeartbeat)
- Story 2.3 (ServerGroupProvider 实现) 已于 2025-12-29 取消
- ADR-0007 决定删除 Agent 注册机制，因与 Temporal Worker 重复
- ADR-0008 采用 Temporal 独立容器架构，改用 Worker API
