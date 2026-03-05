# ADR-0008: Temporal 作为内部服务容器

**状态:** 已采纳  
**日期:** 2025-12-29  
**决策者:** Websoft9 团队  
**前置 ADR:** ADR-0007 (已废弃)

> **📋 实施修订 (2026-03-05, Story 1-12)**  
> 本 ADR 决策方向不变（Temporal 作为内部服务，不对外暴露端口），但实际部署镜像已演进：
> - ~~`temporalio/auto-setup:1.22.0`~~ → **`temporalio/server:1.29.2`**
> - 新增 **`temporalio/admin-tools:1.29.1`** 作为 init 容器初始化 DB schema
> - 新增 **`temporal-create-namespace`** init 容器创建默认命名空间
> - 服务启动顺序：postgresql → temporal-admin-tools → temporal → temporal-create-namespace → waterflow
> - 详见 [deployments/docker-compose.yaml](../../deployments/docker-compose.yaml)

---

## 背景

ADR-0007 提出将 Temporal 内嵌到 Waterflow 容器中（使用 supervisord 管理多进程），以实现对用户透明的目标。在实际实施过程中，我们发现该方案存在以下问题：

### supervisord 方案的问题

1. **违反容器最佳实践**
   - 一个容器运行多个进程（Waterflow + Temporal）
   - 违反 "一个容器一个关注点" 的原则

2. **实施复杂度高**
   - supervisord 配置复杂，命令参数容易出错
   - Temporal Server 需要环境变量配置，与命令行参数混用导致配置混乱
   - 镜像构建需要从官方镜像提取 Temporal 二进制

3. **故障耦合**
   - 两个进程在同一容器中，一个崩溃可能影响另一个
   - 健康检查需要同时检查两个进程，逻辑复杂

4. **调试困难**
   - 两个服务的日志混在一起
   - 无法独立重启某个服务
   - 资源使用情况难以分析

5. **维护成本高**
   - Temporal 升级需要重新构建 Waterflow 镜像
   - 无法独立调整 Temporal 的资源限制

---

## 决策

**采用独立容器架构，Temporal 作为内部服务，通过 Docker 内部网络通信，不对外暴露端口。**

### 核心要素

1. **Temporal 独立容器**
   - 使用官方 `temporalio/auto-setup:1.22.0` 镜像
   - 在 Docker Compose 中作为独立服务
   - 与 Waterflow Server、Agent 同属 `waterflow-network`

2. **不对外暴露端口**
   - Temporal 的 7233 端口**不映射到宿主机**
   - 只能通过 Docker 内部网络访问（`temporal:7233`）
   - 用户无需、也无法直接访问 Temporal

3. **内部网络通信**
   - Waterflow Server → `temporal:7233`（gRPC）
   - Agent → `temporal:7233`（gRPC Worker）
   - 所有通信在 Docker bridge 网络内完成

4. **用户视角透明**
   - 用户只看到 Waterflow 的 8080 端口
   - Temporal 是"黑盒"，对用户完全透明
   - Agent 配置中连接的 `temporal:7233` 逻辑上是 "Waterflow 内部服务"

---

## 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                  Docker Network: waterflow-network          │
│                                                             │
│  ┌──────────────┐         ┌──────────────┐                 │
│  │ PostgreSQL   │◄────────│  Temporal    │                 │
│  │   :5432      │         │   :7233      │ ◄──┐            │
│  └──────────────┘         └──────────────┘    │            │
│                                  ▲             │            │
│                                  │ gRPC        │ gRPC       │
│                                  │             │            │
│                      ┌───────────┴──────┐  ┌──┴────────┐   │
│                      │  Waterflow       │  │  Agent    │   │
│                      │  Server          │  │  Worker   │   │
│                      │  :8080           │  │           │   │
│                      └───────┬──────────┘  └───────────┘   │
│                              │                             │
└──────────────────────────────┼─────────────────────────────┘
                               │
                               │ HTTP :8080 (对外暴露)
                               ▼
                           🌐 用户
```

### 端口暴露情况

| 服务 | 容器端口 | 宿主机端口 | 用途 | 暴露给用户 |
|------|---------|----------|------|----------|
| PostgreSQL | 5432 | - | 内部数据库 | ❌ 否 |
| **Temporal** | **7233** | **-** | **内部引擎** | **❌ 否** |
| Waterflow | 8080 | 8080 | HTTP API | ✅ 是 |
| Agent | 9090 | - | 健康检查 | ❌ 否 |

---

## 实现细节

### docker-compose.yaml 配置

```yaml
services:
  # PostgreSQL - 内部数据库
  postgresql:
    image: postgres:15-alpine
    networks:
      - waterflow-network
    # 不暴露端口

  # Temporal - 内部服务（关键：无 ports 配置）
  temporal:
    image: temporalio/auto-setup:1.22.0
    depends_on:
      postgresql:
        condition: service_healthy
    environment:
      DB: postgresql
      POSTGRES_SEEDS: postgresql
      # ... 其他配置
    networks:
      - waterflow-network
    # ⚠️ 没有 ports 配置 - 不对外暴露

  # Waterflow Server - 对外服务
  waterflow:
    image: waterflow:latest
    depends_on:
      temporal:
        condition: service_healthy
    environment:
      WATERFLOW_TEMPORAL_HOST: temporal:7233  # 内部网络地址
    ports:
      - "8080:8080"  # 只暴露 Waterflow API
    networks:
      - waterflow-network

  # Agent - 内部 Worker
  agent:
    image: waterflow/agent:latest
    environment:
      TEMPORAL_SERVER_URL: temporal:7233  # 内部网络地址
    networks:
      - waterflow-network
    # 不暴露端口
```

### Agent 配置示例

```yaml
# config.agent.yaml
temporal:
  address: temporal:7233  # Docker 内部网络地址
  namespace: default

agent:
  task_queues:
    - linux-amd64
    - linux-common
```

**用户视角：**
- 用户不需要理解 "temporal:7233" 是什么
- 文档中表述为 "连接到 Waterflow 系统"
- 实际上是通过内部网络连接 Temporal

---

## 理由

### 1. 符合容器最佳实践

**一个容器一个关注点：**
- PostgreSQL 容器：数据持久化
- Temporal 容器：工作流引擎
- Waterflow 容器：API 和 DSL 解析
- Agent 容器：任务执行

**单一进程优势：**
- 日志清晰：`docker logs temporal` vs `docker logs waterflow`
- 健康检查独立：各自有明确的健康状态
- 资源隔离：可为 Temporal 设置独立的 CPU/内存限制
- 故障隔离：Temporal 重启不影响 Waterflow

### 2. 保持对用户透明

虽然 Temporal 是独立容器，但通过**不暴露端口**实现了对用户的透明性：

| 方面 | 嵌入方案 | 独立容器+内部网络 | 独立容器+暴露端口 |
|------|---------|-----------------|----------------|
| 用户可见性 | ✅ 不可见 | ✅ 不可见（端口不暴露） | ❌ 可见（7233 端口） |
| 容器数量 | 3 个 | 4 个 | 4 个 |
| 实施复杂度 | ❌ 高（supervisord） | ✅ 低（官方镜像） | ✅ 低（官方镜像） |
| 维护性 | ❌ 差（多进程） | ✅ 好（单进程） | ✅ 好（单进程） |

**结论：** 独立容器+内部网络方案在保持透明性的同时，提供了更好的实施和维护体验。

### 3. 配置简单，开箱即用

**使用官方镜像：**
```dockerfile
# ❌ 嵌入方案需要：
FROM temporalio/auto-setup:1.22.0 AS temporal-source
COPY --from=temporal-source /usr/local/bin/temporal-server /app/
# + supervisord 配置 + 环境变量映射...

# ✅ 独立容器方案：
services:
  temporal:
    image: temporalio/auto-setup:1.22.0  # 直接使用
```

**配置清晰：**
- 环境变量在 docker-compose.yaml 中统一管理
- 无需理解 supervisord 语法
- 官方镜像自带健康检查和初始化脚本

### 4. 易于调试和监控

**独立日志流：**
```bash
docker logs waterflow-temporal    # 只看 Temporal 日志
docker logs waterflow-server      # 只看 Waterflow 日志
docker logs waterflow-agent       # 只看 Agent 日志
```

**独立健康检查：**
```bash
docker ps
# 可以清楚看到每个服务的健康状态
# temporal: healthy
# waterflow: healthy
# agent: healthy
```

**资源监控：**
```bash
docker stats
# 可以独立看 Temporal 的 CPU、内存使用
# 可以单独为 Temporal 设置资源限制
```

### 5. 灵活的升级和扩展

**独立升级：**
```yaml
# 升级 Temporal 无需重建 Waterflow 镜像
temporal:
  image: temporalio/auto-setup:1.23.0  # 只改版本号
```

**未来扩展：**
- 可以轻松切换到 Temporal Cloud（修改连接地址）
- 可以部署 Temporal Cluster（多节点）
- 可以添加 Temporal UI（独立容器）

---

## 后果

### 正面影响

✅ **开发体验**
- 使用官方镜像，无需维护自定义构建
- 配置清晰，易于理解和修改
- 调试方便，日志独立

✅ **运维体验**
- 符合容器最佳实践
- 独立健康检查和监控
- 可独立升级和扩展

✅ **用户体验**
- Temporal 端口不暴露，对用户透明
- 只需关注 Waterflow 的 8080 端口
- 部署简单（`docker compose up -d`）

✅ **架构清晰**
- 每个容器职责单一
- 服务边界清晰
- 依赖关系明确（postgresql → temporal → waterflow → agent）

### 负面影响（可接受）

⚠️ **容器数量增加**
- 从 3 个容器增加到 4 个容器
- **缓解措施：** 
  - 所有容器都是必需的，无冗余
  - 资源占用合理（Temporal 官方镜像已优化）
  - 通过 docker-compose 统一管理，对用户透明

⚠️ **内部网络开销**
- 服务间通过 Docker bridge 网络通信
- **缓解措施：**
  - bridge 网络性能接近本地通信（< 1ms 延迟）
  - Temporal 与 Waterflow 的通信频率不高
  - 可使用 host 网络（如果需要极致性能）

---

## 与 ADR-0007 的关系

**ADR-0007 的目标：**
1. ✅ Temporal 对用户透明
2. ✅ Agent 配置简化
3. ✅ 统一入口点

**ADR-0008 的实现方式：**
- 通过**不暴露端口**而非**物理嵌入**来实现透明性
- 保留目标，改进实现
- 更符合容器化最佳实践

**ADR-0007 状态：** 已废弃（Superseded by ADR-0008）

---

## 验证

### 部署验证

```bash
# 启动所有服务
docker compose up -d

# 验证容器状态（所有服务应为 healthy）
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# 验证 Temporal 未暴露端口（应无输出）
docker port waterflow-temporal

# 验证 Agent 连接成功
docker logs waterflow-agent-linux-1 | grep "Connected to Temporal"
```

### 功能验证

```bash
# 1. 调用 Waterflow API
curl http://localhost:8080/health

# 2. Temporal 不可从宿主机访问（应失败）
telnet localhost 7233  # Connection refused

# 3. 内部网络可访问（在容器内）
docker exec waterflow-agent-linux-1 nc -zv temporal 7233  # 成功
```

---

## 参考

- [Docker Best Practices: One Process per Container](https://docs.docker.com/develop/dev-best-practices/)
- [Temporal Server Configuration](https://docs.temporal.io/self-hosted/how-to-set-up-temporal-server)
- [Docker Compose Networking](https://docs.docker.com/compose/networking/)
- ADR-0007: Waterflow 内嵌 Temporal 并简化 Agent 架构（已废弃）
