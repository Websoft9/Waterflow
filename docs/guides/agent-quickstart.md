# Agent 快速开始指南

## 前置条件

- Docker 20.10+ 或 Podman 3.0+
- 可访问的 Temporal Server (或使用 Docker Compose 启动)

## 方式 1: 单个 Agent 容器 (最简单)

### 1. 启动 Agent

```bash
docker run -d \
  --name waterflow-agent \
  -e TEMPORAL_HOST=temporal.example.com:7233 \
  -e TASK_QUEUES=linux-amd64 \
  -e AGENT_ID=my-first-agent \
  -e LOG_LEVEL=info \
  waterflow/agent:latest
```

### 2. 查看日志

```bash
docker logs -f waterflow-agent
```

### 3. 检查状态

```bash
# Agent 应该显示 "Worker started successfully"
docker logs waterflow-agent 2>&1 | grep "started"
```

## 方式 2: Docker Compose (推荐)

### 1. 创建配置文件

创建 `docker-compose.yaml`:

```yaml
version: '3.8'

services:
  # Temporal Server (可选,如果已有 Temporal 集群可跳过)
  temporal:
    image: temporalio/auto-setup:latest
    ports:
      - "7233:7233"
    environment:
      - DB=postgresql
      - DB_PORT=5432
      - POSTGRES_USER=temporal
      - POSTGRES_PWD=temporal
      - POSTGRES_SEEDS=postgresql
    depends_on:
      - postgresql

  postgresql:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: temporal
      POSTGRES_PASSWORD: temporal

  # Waterflow Agent
  agent:
    image: waterflow/agent:latest
    environment:
      TEMPORAL_HOST: temporal:7233
      TASK_QUEUES: linux-amd64,linux-common
      LOG_LEVEL: info
    depends_on:
      - temporal
    restart: unless-stopped
```

### 2. 启动服务

```bash
docker-compose up -d
```

### 3. 验证 Agent 运行

**验证清单:**

```bash
# ✓ Agent 容器运行中
docker ps | grep agent
# 预期: STATUS = Up

# ✓ Agent 日志无错误
docker-compose logs agent | grep -i error
# 预期: 无输出或仅 WARN

# ✓ Temporal 连接成功
docker-compose logs agent | grep "Worker started successfully"
# 预期: [INFO] Worker started successfully

# ✓ Task Queue 注册成功
docker-compose logs agent | grep "Polling task queues"
# 预期: Polling task queues: [linux-amd64 linux-common]
```

**完整验证脚本** (`scripts/verify-agent.sh`):
```bash
#!/bin/bash
set -e

echo "🔍 验证 Agent 安装..."

# 1. 检查容器状态
if docker ps | grep -q waterflow-agent; then
    echo "✅ Agent 容器运行中"
else
    echo "❌ Agent 容器未运行"
    exit 1
fi

# 2. 检查日志
if docker logs waterflow-agent 2>&1 | grep -q "Worker started successfully"; then
    echo "✅ Worker 启动成功"
else
    echo "❌ Worker 启动失败"
    exit 1
fi

# 3. 检查 Worker 注册
if docker logs waterflow-agent 2>&1 | grep -q "Polling task queues"; then
    echo "✅ Worker 已连接到 Temporal"
else
    echo "❌ Worker 未连接"
    exit 1
fi

echo "🎉 验证完成!"
```

### 原验证步骤

```bash
# 查看 Agent 日志
docker-compose logs -f agent

# 应该看到:
# [INFO] Agent started successfully
# [INFO] Polling task queues: [linux-amd64 linux-common]
```

## 配置 Task Queue

Agent 通过 `TASK_QUEUES` 环境变量指定监听的队列:

```bash
# 单个队列
-e TASK_QUEUES=linux-amd64

# 多个队列 (逗号分隔)
-e TASK_QUEUES=linux-amd64,gpu-a100,web-servers
```

**Task Queue 命名建议:**
- 按 OS/Arch: `linux-amd64`, `darwin-arm64`
- 按功能: `gpu-workers`, `web-servers`, `build-agents`
- 按地域: `us-west-1`, `eu-central-1`

## 监控 Agent 状态

### 方式 1: Waterflow Server API

```bash
# 列出所有 Agent
curl http://localhost:8080/v1/agents

# 查看健康摘要
curl http://localhost:8080/v1/agents/summary
```

### 方式 2: Prometheus Metrics

```bash
# Agent 暴露 Metrics 端口 (默认 9090)
curl http://agent-ip:9090/metrics

# 关键指标:
# - temporal_worker_task_queue_poll_requests_total
# - temporal_activity_execution_total
# - temporal_activity_execution_failed_total
```

## 故障排查

### Agent 无法连接 Temporal

**症状:** 日志显示 `failed to create Temporal client`

**解决:**
```bash
# 1. 检查 Temporal Server 是否可达
telnet temporal.example.com 7233

# 2. 检查环境变量
docker exec waterflow-agent env | grep TEMPORAL

# 3. 检查网络 (Docker)
docker exec waterflow-agent ping temporal

# 4. 查看详细日志
docker logs waterflow-agent 2>&1 | grep -i error
```

### Agent 未接收到任务

**症状:** 提交工作流后 Agent 无响应

**解决:**
```bash
# 1. 检查 Task Queue 配置
docker logs waterflow-agent | grep "Polling task queues"

# 2. 验证工作流的 runs-on 是否匹配
# Workflow YAML:
#   runs-on: linux-amd64  ← 必须与 Agent 的 TASK_QUEUES 匹配

# 3. 查询 Task Queue 状态
curl http://localhost:8080/v1/task-queues | jq '.task_queues[] | select(.name=="linux-amd64")'
```

### Agent 频繁重启

**症状:** `docker ps` 显示 Agent 不断重启

**解决:**
```bash
# 1. 查看退出原因
docker logs waterflow-agent --tail=100

# 2. 检查资源限制 (OOM?)
docker stats waterflow-agent

# 3. 增加内存限制
docker run -d \
  --memory=512m \
  --memory-swap=1g \
  -e TASK_QUEUES=linux-amd64 \
  waterflow/agent:latest
```

## 下一步

- [配置文件详解](../../docs/sprint-artifacts/2-10-agent-configuration-guide.md)
- [生产环境部署最佳实践](./agent-best-practices.md)
- [故障排查手册](./agent-troubleshooting.md)
