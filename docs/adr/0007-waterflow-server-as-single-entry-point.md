# ADR-0007: Waterflow 内嵌 Temporal 并简化 Agent 架构

**状态:** 已废弃（Superseded by 实际实施方案）  
**日期:** 2025-12-26  
**决策者:** Websoft9 团队  
**更新日期:** 2025-12-29（架构调整：独立容器方案）

## 2025-12-29 更新：架构调整

经过实际实施验证，我们调整了架构决策：

**最终采用方案：Temporal 作为独立容器 + 内部网络通信**

### 调整理由

1. **容器最佳实践**：一个容器运行一个进程，避免 supervisord 多进程管理复杂性
2. **故障隔离**：Temporal 和 Waterflow 进程独立，互不影响
3. **易于维护**：独立日志、健康检查、资源限制
4. **配置简单**：直接使用官方 temporalio/auto-setup 镜像，无需自行构建
5. **实施验证**：supervisord 方案在实际部署中遇到配置和稳定性问题

### 最终架构

```yaml
services:
  postgresql:     # 数据持久化
  temporal:       # 独立容器，不对外暴露端口
  waterflow:      # Waterflow Server，连接 temporal:7233
  agent:          # Agent Worker，连接 temporal:7233
```

**关键点：**
- Temporal 端口**不对外暴露**（无 ports 配置）
- 所有通信通过 Docker 内部网络（waterflow-network）
- 用户视角：Temporal 仍然是内部实现细节，对用户透明
- 技术实现：符合容器化最佳实践

### 与原 ADR 的关系

**保留的目标：**
- ✅ Temporal 对用户透明（不暴露端口）
- ✅ 用户只需配置 Waterflow 相关参数
- ✅ Agent 配置简化

**调整的实现：**
- ❌ 不使用内嵌方式（supervisord）
- ✅ 使用独立容器 + 内部网络

---

## 原始 ADR 内容（供参考）

---

## 背景

### 当前设计的问题

在当前的实现中（Epic 2），Agent 需要配置两个不同的服务器地址：

```yaml
# Agent 配置
temporal:
  address: "temporal:7233"    # ❌ Temporal Server 地址（直连）
  namespace: "default"

agent:
  server_url: "http://waterflow:8080"  # Waterflow Server 地址（可选）
  task_queues: ["linux-amd64"]
```

**问题 1: 架构封装性差**
- Temporal 是 Waterflow 的内部实现细节，不应暴露给用户
- Agent 直接连接 Temporal Server，绕过了 Waterflow Server
- 破坏了系统的抽象层次

**问题 2: 用户配置复杂**
- 用户需要了解 Temporal 的存在
- 需要配置两个不同的地址和端口
- 需要理解 Temporal 的 namespace、task queue 等概念

**问题 3: 可维护性和可扩展性受限**
- 如果将来更换底层引擎（从 Temporal 迁移到其他引擎），所有 Agent 配置都需要修改
- 无法在 Waterflow 层面统一管理 Agent 连接（如 TLS、认证、限流）
- Temporal 版本升级可能影响所有 Agent

**问题 4: 职责混乱**
- Agent 注册到 Waterflow Server（监控功能），但任务执行却从 Temporal 获取
- Waterflow Server 只能看到注册信息，无法控制任务分发
- 两个连接的生命周期管理独立，可能导致不一致状态

### 用户期望

用户的视角中：
- **只有 Waterflow**，没有 Temporal
- Temporal 应该是 Waterflow 的内部组件（类似 PostgreSQL 是 Waterflow 的数据库）
- Agent 应该只需要配置一个地址：Waterflow Server

---

## 决策

**将 Temporal Server 内嵌到 Waterflow 容器中，Agent 只连接 Waterflow（通过内嵌的 Temporal），取消独立的 Agent 注册功能。**

### 核心决策

1. **Temporal 内嵌化**
   - Temporal Server 作为 Waterflow 容器内的进程运行
   - 使用 supervisord 管理多进程（Waterflow Server + Temporal Server）
   - Temporal 的 7233 端口对外暴露（Agent 连接），但逻辑上属于 Waterflow

2. **Agent 连接简化**
   - Agent 配置中只需要 Waterflow 地址（如 `waterflow:7233`）
   - Agent 不需要区分 Temporal 和 Waterflow（对用户透明）
   - 完全复用 Temporal Worker SDK（零性能损耗）

3. **取消 Agent 注册机制**
   - 删除 Agent 到 Waterflow Server 的 HTTP 注册（POST /v1/agents/register）
   - 删除 Agent 心跳机制（PUT /v1/agents/heartbeat）
   - 删除 ServerGroupProvider 接口及其实现
   - Temporal Worker 的连接本身就是"注册"

4. **可观测性通过 Temporal 原生 API**
   - `/v1/agents` API 改为查询 Temporal Worker 列表
   - 用户可直接使用 Temporal UI 查看 Agent 状态
   - 单一数据源，无同步延迟

### 新的架构

```
┌──────────────────────────────────────┐
│     Waterflow Container              │
│                                      │
│  ┌────────────────────────────┐     │
│  │  Waterflow Server          │     │
│  │  (HTTP API :8080)          │     │
│  └───────────┬────────────────┘     │
│              │ localhost            │
│  ┌───────────▼────────────────┐     │
│  │  Temporal Server           │     │
│  │  (:7233 exposed)           │     │  ← 内嵌但端口暴露
│  └────────────────────────────┘     │
│                                      │
└──────────────────┬───────────────────┘
                   │
              gRPC │:7233
                   │
              ┌────┴─────┐
              │  Agent   │
              │ (Worker) │  ← 只连接 7233，不需要注册
              └──────────┘
```

### Agent 配置（最终形态）

```yaml
# config.agent.yaml
server:
  url: "waterflow:7233"  # 🎯 只需要一个地址！逻辑上是 Waterflow，技术上是内嵌的 Temporal
  namespace: "default"

agent:
  task_queues:
    - linux-amd64
    - linux-common

# ❌ 不再需要：
# - TEMPORAL_SERVER_URL（已内嵌）
# - SERVER_URL（不需要注册）
# - 心跳配置（Temporal Worker SDK 自动处理）
```

---

## 理由

### 1. **架构纯粹性：Temporal 是实现细节**

Temporal 应该是 Waterflow 的内部组件，就像 PostgreSQL 是数据库一样，对用户完全透明。

```
用户视角:
  Waterflow = 完整的工作流系统
  
技术实现:
  Waterflow = DSL Parser + API + Temporal (内嵌)
```

**类比：**
- PostgreSQL 在 Waterflow 容器外（因为是数据层）
- Temporal 应该在 Waterflow 容器内（因为是引擎层）

### 2. **Agent 注册功能是重复设计**

当前 Epic 2 实现的 Agent 注册机制与 Temporal Worker 机制完全重复：

| 功能 | Temporal Worker | Waterflow Agent 注册 | 重复？ |
|------|----------------|---------------------|--------|
| Worker 上线通知 | ✅ gRPC Connect | ✅ POST /register | ✅ 重复 |
| 心跳机制 | ✅ 30s 自动心跳 | ✅ 30s HTTP 心跳 | ✅ 重复 |
| 健康检测 | ✅ 3 次失败下线 | ✅ 3 次失败 unhealthy | ✅ 重复 |
| Worker 列表 | ✅ ListWorkers API | ✅ ServerGroupProvider | ✅ 重复 |
| Task Queue 映射 | ✅ 原生支持 | ✅ runs-on 映射 | ✅ 重复 |
| 负载均衡 | ✅ 原生支持 | ❌ 不支持 | Temporal 更强 |

**问题：**
- Agent 维护两个连接（Temporal gRPC + Waterflow HTTP）
- 两份心跳（浪费网络和 CPU）
- 两份数据源可能不一致（同步延迟）
- ServerGroupProvider 查询的数据不用于任务分发（仅供查询）

**解决：**
- 删除 Waterflow 的 Agent 注册机制
- 完全依赖 Temporal Worker 机制
- `/v1/agents` API 直接查询 Temporal

### 3. **用户体验提升**

**配置简化：**

```yaml
# 当前（Epic 2）：需要两个地址
TEMPORAL_SERVER_URL: temporal:7233
SERVER_URL: http://waterflow:8080

# 优化后：只需要一个地址
SERVER_URL: waterflow:7233
```

**部署简化：**

```yaml
# 当前：3 个容器
services:
  - postgresql
  - temporal        # ← 独立容器
  - waterflow
  - agent

# 优化后：2 个容器
services:
  - postgresql
  - waterflow      # ← 内嵌 Temporal
  - agent
```

**概念简化：**
- 用户不需要了解 Temporal
- 不需要理解 "Worker 注册" vs "Agent 注册"
- 不需要配置 namespace、task queue（由 Waterflow 管理）

### 4. **性能和可靠性**

**内嵌 Temporal 的优势：**
- ✅ 容器内通信（localhost，无网络延迟）
- ✅ 减少容器数量（资源占用更小）
- ✅ 简化网络拓扑（无需暴露 Temporal 端口给用户）
- ✅ 统一生命周期（Waterflow 启动即包含 Temporal）

**取消 Agent 注册的优势：**
- ✅ 减少网络请求（无 HTTP 注册和心跳）
- ✅ 单一数据源（Temporal 是唯一真相）
- ✅ 无同步延迟（不存在数据不一致）
- ✅ 减少故障点（注册失败不再是问题）

### 5. **对标业界最佳实践**

**对比其他工作流系统：**

| 系统 | Agent 如何连接 | 引擎暴露给用户 |
|------|--------------|--------------|
| **Temporal** | 直连 Temporal Server | ✅ 是 |
| **Airflow** | 直连 Scheduler | ✅ 是 |
| **Argo Workflows** | 直连 K8s API | ✅ 是 |
| **Jenkins** | 连接 Jenkins Master | ❌ 否（Master 是统一入口）|
| **GitLab CI** | 连接 GitLab Server | ❌ 否（Server 是统一入口）|
| **Waterflow（当前）** | 直连 Temporal | ✅ 是（暴露） |
| **Waterflow（优化后）** | 连接 Waterflow | ❌ 否（内嵌） |

**Waterflow 的定位：**
- 类似 Jenkins/GitLab CI（统一入口，内部引擎对用户透明）
- 而非 Temporal/Airflow（引擎直接暴露给用户）

---

## 后果

### 正面影响

✅ **架构清晰**
- Temporal 成为真正的内部实现（黑盒）
- 用户只需要理解 Waterflow
- 清晰的分层架构

✅ **配置简化**
- Agent 配置减少 50%（一个地址 vs 两个地址）
- 部署容器减少（3个 → 2个核心容器）
- 文档复杂度降低

✅ **性能提升**
- 容器内通信（localhost，零网络延迟）
- 减少 HTTP 请求（无注册和心跳）
- 单一数据源（无同步开销）

✅ **代码简化**
- 删除 ~800 行代码（Agent 注册 + ServerGroupProvider）
- 减少维护成本
- 减少 bug 风险

✅ **可靠性提升**
- 减少故障点（无注册失败问题）
- 单一数据源（Temporal 是唯一真相）
- 无数据不一致风险

### 负面影响

⚠️ **Epic 2 部分功能需要重构**
- Story 2.3（ServerGroupProvider）需要删除
- Story 2.4（Agent 注册）需要删除
- Story 2.7（健康监控 API）需要改为查询 Temporal
- 影响范围：~800 行代码，3 个 Story

⚠️ **失去自定义调度扩展点**
- 完全依赖 Temporal 的调度策略
- 无法实现自定义负载均衡
- 如果未来需要高级调度，需要其他方案

⚠️ **提交前无法验证 Agent 可用性**
- 用户提交 `runs-on: xxx` 时
- Waterflow 无法立即告知"没有可用 Agent"
- 任务会在 Temporal 中排队等待（Temporal 原生行为）

⚠️ **Docker 镜像构建复杂度增加**
- 需要将 Temporal Server 二进制打包到 Waterflow 镜像
- 需要 supervisord 管理多进程
- 镜像大小增加（但可优化）

### 风险缓解

**Epic 2 重构风险：**
- 影响的功能均为"可选功能"（注册失败不影响任务执行）
- 重构范围明确（3 个 Story）
- 可以在 Epic 3 后统一重构

**扩展性风险：**
- 未来如需自定义调度，可在 Waterflow Server 层实现
- 提交时检查 runs-on 可通过查询 Temporal Worker 列表实现
- 保持架构灵活性

**性能风险：**
- supervisord 开销极小（<1% CPU）
- 容器内通信比网络通信更快
- 经过压力测试验证

---

## 实现方案

### 技术选型：容器内多进程 + supervisord

**架构：**

```dockerfile
# Waterflow Container
FROM ubuntu:22.04

# 安装 supervisord
RUN apt-get update && apt-get install -y supervisor

# 复制 Temporal Server 二进制（从官方镜像）
COPY --from=temporalio/auto-setup:1.22.0 /usr/local/bin/temporal-server /usr/local/bin/

# 复制 Waterflow Server 二进制
COPY bin/server /usr/local/bin/waterflow-server

# Supervisor 配置
COPY supervisord.conf /etc/supervisor/conf.d/waterflow.conf

# 暴露端口
EXPOSE 8080 7233

CMD ["/usr/bin/supervisord"]
```

**supervisord.conf：**

```ini
[supervisord]
nodaemon=true

[program:temporal]
command=/usr/local/bin/temporal-server start --bind 0.0.0.0:7233
priority=1
autorestart=true

[program:waterflow]
command=/usr/local/bin/waterflow-server
priority=2
depends_on=temporal
autorestart=true
```

### 代码改动范围

#### 删除代码（~800 行）

```bash
# Agent 端
internal/agent/worker.go
  - registerToServer()           # 删除
  - startHeartbeatUpdater()      # 删除
  - updateHeartbeat()            # 删除
  - doHeartbeat()                # 删除

# Server 端
internal/server/server.go
  - ServerGroupProvider 初始化  # 删除

internal/api/agent_handler.go
  - POST /v1/agents/register     # 删除
  - PUT /v1/agents/heartbeat     # 删除
  - GET /v1/agents               # 改为查询 Temporal

pkg/provider/
  - memory_provider.go           # 删除整个目录
  - file_provider.go
  - provider.go

# 配置
config.agent.yaml
  - agent.server_url             # 删除
```

#### 新增/修改代码（~200 行）

```bash
# Dockerfile
build/Dockerfile.server-with-temporal  # 新增
deployments/supervisord.conf           # 新增

# API 改动
internal/api/agent_handler.go
  - ListAgents() - 改为查询 Temporal Worker 列表

# 配置改动
pkg/config/config.go
  - 移除 Agent.ServerURL 字段
```

### 数据迁移

**配置迁移（自动）：**

```go
// pkg/config/config.go
func LoadAgentConfig(path string) (*Config, error) {
    cfg := &Config{}
    viper.ReadInConfig()
    
    // 兼容旧配置（过渡期）
    if serverURL := viper.GetString("agent.server_url"); serverURL != "" {
        log.Warn("agent.server_url is deprecated, it will be ignored")
    }
    
    // 新配置只需要 server.url
    cfg.Server.URL = viper.GetString("server.url")
    
    return cfg, nil
}
```

**Docker Compose 迁移：**

```yaml
# 旧版本（Epic 2）
services:
  temporal:
    image: temporalio/auto-setup:1.22.0
  waterflow:
    image: waterflow/server:latest
  agent:
    environment:
      TEMPORAL_SERVER_URL: temporal:7233
      SERVER_URL: http://waterflow:8080

# 新版本（ADR-0007）
services:
  waterflow:  # 内嵌 Temporal
    image: waterflow/server-with-temporal:latest
    ports:
      - "8080:8080"
      - "7233:7233"
  agent:
    environment:
      SERVER_URL: waterflow:7233  # 只需要一个地址
```

### 可观测性实现

**GET /v1/agents 实现：**

```go
// internal/api/agent_handler.go

func (h *AgentHandler) ListAgents(c *gin.Context) {
    // 查询 Temporal Worker 列表
    ctx := c.Request.Context()
    
    resp, err := h.temporalClient.WorkflowService().ListWorkers(ctx, &workflowservice.ListWorkersRequest{
        Namespace: "default",
    })
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // 转换为 Waterflow Agent 格式
    agents := make([]Agent, 0, len(resp.Workers))
    for _, w := range resp.Workers {
        agents = append(agents, Agent{
            ID:         w.Identity,
            TaskQueues: w.TaskQueues,
            Status:     w.RateLimit > 0 ? "healthy" : "unhealthy",
            LastSeen:   w.LastAccessTime.AsTime(),
        })
    }
    
    c.JSON(200, gin.H{"agents": agents})
}
```

**用户体验：**

```bash
# CLI 查询（不变）
$ waterflow agents list

ID              TASK_QUEUES          STATUS    LAST_SEEN
agent-linux-1   linux-amd64,common   healthy   2s ago
agent-linux-2   linux-amd64          healthy   5s ago

# Temporal UI（直接可用）
http://waterflow:8088/namespaces/default/workers
```

---

## 替代方案（已否决）

### 替代方案 1: 保持当前架构（独立 Temporal 容器 + Agent 注册）

**优点：**
- Epic 2 已经实现，无需改动
- Temporal 容器独立，方便调试
- Agent 注册提供了额外的监控数据

**否决原因：**
- ❌ Temporal 对用户可见（破坏封装）
- ❌ Agent 维护两个连接（复杂且重复）
- ❌ ServerGroupProvider 不参与任务分发（无实际价值）
- ❌ 配置复杂（需要两个地址）

### 替代方案 2: gRPC 代理方案（不内嵌 Temporal）

**做法：**
- Waterflow Server 实现 Temporal gRPC 协议代理
- Temporal 仍然是独立容器
- Agent 连接 Waterflow，Waterflow 转发到 Temporal

**否决原因：**
- ❌ 实现复杂（需要代理完整 gRPC 协议）
- ❌ 性能损耗（多一跳网络）
- ❌ Temporal 仍然独立部署（用户可见）
- ✅ 内嵌方案更简单直接

### 替代方案 3: 保留 Agent 注册，但标记为"可选"

**做法：**
- 内嵌 Temporal
- 保留 Agent 注册功能（可选）
- 文档说明"注册仅用于监控"

**否决原因：**
- ❌ 保留了冗余代码（~800 行）
- ❌ 用户困惑（为什么有两个连接？）
- ❌ 维护成本（两套机制需要同步维护）
- ✅ 完全删除更清晰

---

## 相关 ADR

- [ADR-0001: 使用 Temporal 作为工作流引擎](0001-use-temporal-workflow-engine.md)
  - Temporal 作为内部实现，对用户透明
- [ADR-0006: Task Queue 路由机制](0006-task-queue-routing.md)
  - Task Queue 路由仍然有效，但路由控制在 Waterflow Server

---

## 实施计划

### 阶段 1: Epic 2 完成和文档更新（当前阶段）

**状态：** ✅ 已完成

- [x] Epic 2 全部 10 个 Story 实现完成
- [x] Docker Compose 部署验证通过
- [x] 创建 ADR-0007 文档
- [x] 明确架构改进方向

### 阶段 2: Epic 3 开发（保持现状）

**策略：** 不改动 Epic 2，保持架构稳定

- [ ] 按计划开发 Epic 3（核心节点插件库）
- [ ] Agent 注册功能保持可选状态
- [ ] 为架构重构积累经验

### 阶段 3: 架构重构实施（Epic 3 后或 Epic 12）

**预计工作量：** 3-5 天

#### Step 1: 内嵌 Temporal（1-2 天）

- [ ] 创建 `build/Dockerfile.server-with-temporal`
- [ ] 编写 `deployments/supervisord.conf`
- [ ] 构建和测试新镜像
- [ ] 验证 Agent 可正常连接

#### Step 2: 删除 Agent 注册代码（1 天）

- [ ] 删除 `internal/agent/worker.go` 中的注册逻辑
- [ ] 删除 `internal/api/agent_handler.go` 注册 API
- [ ] 删除 `pkg/provider/` 整个目录
- [ ] 删除相关配置字段

#### Step 3: 实现 Temporal 查询（1 天）

- [ ] 修改 `GET /v1/agents` 查询 Temporal Worker 列表
- [ ] 更新 API 响应格式
- [ ] 添加单元测试

#### Step 4: 文档和配置更新（1 天）

- [ ] 更新所有 Agent 文档（6 个）
- [ ] 更新 Docker Compose 配置
- [ ] 更新配置示例文件
- [ ] 编写迁移指南

### 阶段 4: 发布和迁移

- [ ] 发布新版本镜像
- [ ] 提供迁移工具
- [ ] 发布迁移公告
- [ ] 逐步下线旧架构

---

## 参考资料

- [Temporal Architecture](https://docs.temporal.io/references/architecture)
- [gRPC Proxy Pattern](https://grpc.io/docs/guides/proxy/)
- [Jenkins Agent Protocol](https://github.com/jenkinsci/remoting/blob/master/docs/protocols.md)
- [GitLab Runner Architecture](https://docs.gitlab.com/runner/development/architecture.html)

---

## 修订历史

- 2025-12-26: 初始版本（Websoft9）
