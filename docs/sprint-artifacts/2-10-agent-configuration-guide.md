# Story 2.10: Agent 配置与部署指南

Status: ready-for-dev

## Story

As a **运维工程师**,  
I want **完整的 Agent 配置和部署文档**,  
so that **快速上手 Agent 部署和故障排查**。

## Context

这是 **Epic 2: 分布式 Agent 系统**的第十个也是最后一个 Story。前面的 Stories 已实现 Agent Worker、Docker 镜像等所有功能,现在需要提供全面的配置指南和部署最佳实践。

**前置依赖:**
- Story 2.1 (Agent Worker) - Agent 核心功能
- Story 2.7 (健康监控) - Agent 监控 API
- Story 2.9 (Docker 镜像) - 容器化部署

**业务价值:**
- 📖 **降低学习曲线** - 5 分钟快速上手
- 🛠️ **标准化部署** - 统一的部署模式
- 🔍 **快速排障** - 常见问题解决方案
- 📋 **最佳实践** - 生产环境配置建议

**文档范围:**
1. 配置文件详解
2. 多种部署方式 (Docker, systemd, Kubernetes)
3. 监控和日志配置
4. 常见问题排查
5. 安全加固建议

## Acceptance Criteria

### AC1: Agent 配置文件完整示例

**Given** 用户需要配置 Agent  
**When** 参考配置文件模板  
**Then** 包含所有配置项和注释

**配置文件** (`config.agent.example.yaml`):
```yaml
# ==============================================
# Waterflow Agent Configuration
# ==============================================
# 
# 配置优先级: 环境变量 > 配置文件 > 默认值
# 
# 快速开始:
#   1. 复制此文件为 config.yaml
#   2. 修改 temporal.server_url 和 agent.task_queues
#   3. 启动 Agent: ./agent --config config.yaml
# ==============================================

# Agent 基本配置
agent:
  # Agent 唯一标识符 (建议使用主机名或自动生成)
  # 环境变量: AGENT_ID
  # 默认: agent-<hostname>-<timestamp>
  id: "agent-build-server-1"
  
  # Agent 监听的 Task Queue 列表 (必填)
  # 环境变量: TASK_QUEUES (逗号分隔)
  # 示例: TASK_QUEUES=linux-amd64,gpu-a100
  task_queues:
    - "linux-amd64"
    - "linux-common"
  
  # Waterflow Server URL (用于心跳上报和注册)
  # 环境变量: SERVER_URL
  # 留空则不上报心跳
  server_url: "http://localhost:8080"
  
  # Agent 元数据 (可选,用于 ServerGroupProvider 查询)
  metadata:
    os: "linux"
    arch: "amd64"
    cpu_cores: "16"
    memory_gb: "32"
    gpu: "NVIDIA A100"
    region: "us-west-1"
    datacenter: "dc1"

# Temporal 连接配置
temporal:
  # Temporal Server 地址 (必填)
  # 环境变量: TEMPORAL_SERVER_URL
  server_url: "localhost:7233"
  
  # Temporal Namespace
  # 默认: default
  namespace: "default"
  
  # Worker 配置
  worker:
    # 并发执行的 Activity 数量
    # 建议设置为 CPU 核心数
    max_concurrent_activities: 10
    
    # 并发执行的 Workflow 数量
    max_concurrent_workflows: 10
    
    # Task Queue 长轮询超时时间
    task_queue_poll_timeout: "60s"
  
  # 连接超时配置
  connection:
    # 连接超时
    dial_timeout: "5s"
    
    # 保活间隔
    keep_alive_time: "30s"
    
    # 保活超时
    keep_alive_timeout: "15s"

# 日志配置
logger:
  # 日志级别: debug, info, warn, error
  # 环境变量: LOG_LEVEL
  level: "info"
  
  # 日志格式: json, console
  format: "console"
  
  # 日志输出: stdout, stderr, file
  output: "stdout"
  
  # 日志文件配置 (仅当 output=file 时生效)
  file:
    path: "/var/log/waterflow/agent.log"
    max_size: 100  # MB
    max_backups: 7
    max_age: 30  # days
    compress: true

# Metrics 配置
metrics:
  # 是否启用 Metrics
  enabled: true
  
  # Metrics HTTP 端口
  # 环境变量: METRICS_PORT
  port: "9090"
  
  # Metrics 路径
  path: "/metrics"

# Plugin 配置
plugins:
  # Plugin 目录
  directory: "/opt/waterflow/plugins"
  
  # 是否自动加载所有 .so 文件
  auto_load: true
  
  # 显式指定要加载的 Plugin (可选)
  enabled:
    - "my-custom-plugin.so"

# 安全配置
security:
  # TLS 配置 (连接 Temporal Server)
  tls:
    enabled: false
    cert_file: "/etc/waterflow/certs/client.crt"
    key_file: "/etc/waterflow/certs/client.key"
    ca_file: "/etc/waterflow/certs/ca.crt"
    
    # 跳过证书验证 (仅用于开发环境)
    insecure_skip_verify: false
  
  # mTLS 客户端认证
  mtls:
    enabled: false
    client_cert: "/etc/waterflow/certs/agent.crt"
    client_key: "/etc/waterflow/certs/agent.key"

# 高级配置
advanced:
  # 优雅关闭超时时间
  graceful_shutdown_timeout: "30s"
  
  # 心跳上报间隔
  heartbeat_interval: "30s"
  
  # Activity 心跳超时 (Temporal)
  activity_heartbeat_timeout: "10s"
  
  # 重试策略
  retry:
    initial_interval: "1s"
    backoff_coefficient: 2.0
    maximum_interval: "60s"
    maximum_attempts: 5
```

**验证配置文件:**
```bash
# 检查配置文件语法
./agent --config config.yaml --validate

# 打印解析后的配置
./agent --config config.yaml --print-config
```

### AC2: systemd 服务单元文件

**Given** 裸机服务器需要部署 Agent  
**When** 使用 systemd 管理 Agent  
**Then** 支持开机自启和故障重启

**systemd Service 文件** (`deployments/systemd/waterflow-agent.service`):
```ini
[Unit]
Description=Waterflow Agent - Distributed Workflow Execution Agent
Documentation=https://github.com/yourusername/waterflow
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=waterflow
Group=waterflow

# 工作目录
WorkingDirectory=/opt/waterflow

# Agent 可执行文件
ExecStart=/opt/waterflow/bin/agent --config /etc/waterflow/agent.yaml

# 环境变量 (可选,配置文件优先)
Environment="LOG_LEVEL=info"
Environment="METRICS_PORT=9090"
EnvironmentFile=-/etc/waterflow/agent.env

# 重启策略
Restart=on-failure
RestartSec=5s
StartLimitInterval=60s
StartLimitBurst=3

# 超时配置
TimeoutStartSec=30s
TimeoutStopSec=30s

# 资源限制
LimitNOFILE=65536
LimitNPROC=4096

# 安全加固
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/waterflow /opt/waterflow/data

# 日志
StandardOutput=journal
StandardError=journal
SyslogIdentifier=waterflow-agent

[Install]
WantedBy=multi-user.target
```

**部署脚本** (`scripts/install-agent.sh`):
```bash
#!/bin/bash
set -e

echo "Installing Waterflow Agent..."

# 1. 创建用户和组
if ! id -u waterflow &>/dev/null; then
    echo "Creating waterflow user..."
    sudo useradd -r -s /bin/false -d /opt/waterflow waterflow
fi

# 2. 创建目录结构
echo "Creating directories..."
sudo mkdir -p /opt/waterflow/{bin,plugins,data}
sudo mkdir -p /etc/waterflow
sudo mkdir -p /var/log/waterflow

# 3. 复制二进制文件
echo "Installing agent binary..."
sudo cp bin/agent /opt/waterflow/bin/
sudo chmod +x /opt/waterflow/bin/agent

# 4. 复制配置文件
if [ ! -f /etc/waterflow/agent.yaml ]; then
    echo "Installing default config..."
    sudo cp config.agent.example.yaml /etc/waterflow/agent.yaml
    echo "⚠️  Please edit /etc/waterflow/agent.yaml to configure Task Queues"
fi

# 5. 设置权限
sudo chown -R waterflow:waterflow /opt/waterflow
sudo chown -R waterflow:waterflow /var/log/waterflow
sudo chmod 640 /etc/waterflow/agent.yaml

# 6. 安装 systemd service
echo "Installing systemd service..."
sudo cp deployments/systemd/waterflow-agent.service /etc/systemd/system/
sudo systemctl daemon-reload

# 7. 启用并启动服务
echo "Starting Waterflow Agent..."
sudo systemctl enable waterflow-agent
sudo systemctl start waterflow-agent

# 8. 检查状态
sleep 2
sudo systemctl status waterflow-agent --no-pager

echo ""
echo "✅ Waterflow Agent installed successfully!"
echo ""
echo "Next steps:"
echo "  1. Edit config: sudo nano /etc/waterflow/agent.yaml"
echo "  2. Restart service: sudo systemctl restart waterflow-agent"
echo "  3. View logs: sudo journalctl -u waterflow-agent -f"
```

**使用示例:**
```bash
# 安装 Agent
sudo ./scripts/install-agent.sh

# 编辑配置
sudo nano /etc/waterflow/agent.yaml

# 重启服务
sudo systemctl restart waterflow-agent

# 查看状态
sudo systemctl status waterflow-agent

# 查看日志
sudo journalctl -u waterflow-agent -f

# 停止服务
sudo systemctl stop waterflow-agent

# 禁用开机自启
sudo systemctl disable waterflow-agent
```

### AC3: Docker 部署快速开始指南

**Given** 用户有 Docker 环境  
**When** 参考部署指南  
**Then** 5 分钟内启动 Agent

**快速开始文档** (`docs/guides/agent-quickstart.md`):
```markdown
# Agent 快速开始指南

## 前置条件

- Docker 20.10+ 或 Podman 3.0+
- 可访问的 Temporal Server (或使用 Docker Compose 启动)

## 方式 1: 单个 Agent 容器 (最简单)

### 1. 启动 Agent

```bash
docker run -d \
  --name waterflow-agent \
  -e TEMPORAL_SERVER_URL=temporal.example.com:7233 \
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
      TEMPORAL_SERVER_URL: temporal:7233
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

# ✓ 心跳正常上报 (如果配置了 SERVER_URL)
curl http://localhost:8080/v1/agents | jq '.total'
# 预期: 1 或更多
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

# 3. 检查心跳
if curl -s http://localhost:8080/v1/agents | jq -e '.total > 0' > /dev/null; then
    echo "✅ Agent 已注册"
else
    echo "⚠️  Agent 未注册 (可能未配置 SERVER_URL)"
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

## 方式 3: Kubernetes (生产环境)

### 1. 创建 Deployment

```bash
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: waterflow-agent
spec:
  replicas: 3
  selector:
    matchLabels:
      app: waterflow-agent
  template:
    metadata:
      labels:
        app: waterflow-agent
    spec:
      containers:
      - name: agent
        image: waterflow/agent:latest
        env:
        - name: TEMPORAL_SERVER_URL
          value: "temporal.default.svc.cluster.local:7233"
        - name: TASK_QUEUES
          value: "linux-amd64"
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
EOF
```

### 2. 验证部署

```bash
# 查看 Pods
kubectl get pods -l app=waterflow-agent

# 查看日志
kubectl logs -l app=waterflow-agent --tail=50
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

- [配置文件详解](./agent-configuration.md)
- [Plugin 开发指南](./plugin-development.md)
- [生产环境部署最佳实践](./production-deployment.md)
```

### AC4: 配置最佳实践文档

**Given** 用户需要生产环境配置建议  
**When** 参考最佳实践文档  
**Then** 获得安全、高性能的配置

**最佳实践文档** (`docs/guides/agent-best-practices.md`):
````markdown
# Agent 配置最佳实践

## 1. Task Queue 规划

### 按资源类型分组

```yaml
# ✅ 推荐: 细粒度分组
agent:
  task_queues:
    - "linux-amd64-high-cpu"    # 8+ CPU 核心
    - "linux-amd64-high-memory" # 16GB+ 内存
    - "gpu-nvidia-a100"         # NVIDIA A100 GPU

# ❌ 不推荐: 单个通用队列
agent:
  task_queues:
    - "default"  # 无法区分资源需求
```

**原因:** 细粒度分组允许工作流精确选择所需资源。

### 队列命名规范

```
格式: <OS>-<ARCH>-<FEATURE>-<REGION>

示例:
- linux-amd64-gpu-us-west-1
- darwin-arm64-build-office
- windows-amd64-test-qa
```

## 2. 性能调优

### Worker 并发配置

```yaml
temporal:
  worker:
    # CPU 密集型任务
    max_concurrent_activities: 4  # = CPU 核心数

    # I/O 密集型任务
    max_concurrent_activities: 20  # = CPU 核心数 * 2~3

    # 混合任务
    max_concurrent_activities: 10  # = CPU 核心数 * 1.5
```

**测试方法:**
```bash
# 启动 Agent 并观察 CPU 使用率
htop

# CPU 使用率 < 50% → 增加并发数
# CPU 使用率 > 90% → 减少并发数
```

### 心跳间隔优化

```yaml
advanced:
  # 短心跳间隔 (10s) - 快速故障检测
  heartbeat_interval: "10s"  # 适用于关键任务
  
  # 中等心跳间隔 (30s) - 平衡性能和检测速度
  heartbeat_interval: "30s"  # ✅ 推荐默认值
  
  # 长心跳间隔 (60s) - 减少网络开销
  heartbeat_interval: "60s"  # 适用于稳定环境
```

**权衡:**
- 短间隔 → 快速发现故障 Agent,但增加网络流量
- 长间隔 → 减少开销,但故障检测延迟

## 3. 日志管理

### 生产环境日志配置

```yaml
logger:
  level: "info"  # ✅ 生产环境
  # level: "debug"  # ❌ 仅用于调试,会产生大量日志

  format: "json"  # ✅ 便于日志分析工具解析
  # format: "console"  # ❌ 仅用于开发环境

  output: "file"  # ✅ 持久化日志
  
  file:
    path: "/var/log/waterflow/agent.log"
    max_size: 100  # 每个文件 100MB
    max_backups: 7  # 保留 7 个备份
    max_age: 30    # 保留 30 天
    compress: true  # 压缩旧日志
```

**日志轮转策略:**
- 每天生成 ~500MB 日志 → `max_size: 100, max_backups: 5`
- 每天生成 ~100MB 日志 → `max_size: 100, max_backups: 7`

### 集成日志聚合系统

```bash
# 方案 1: Fluentd
docker run -d \
  --log-driver=fluentd \
  --log-opt fluentd-address=fluentd:24224 \
  waterflow/agent:latest

# 方案 2: Loki
docker run -d \
  --log-driver=loki \
  --log-opt loki-url=http://loki:3100/loki/api/v1/push \
  waterflow/agent:latest

# 方案 3: CloudWatch (AWS)
docker run -d \
  --log-driver=awslogs \
  --log-opt awslogs-group=/waterflow/agent \
  waterflow/agent:latest
```

## 4. 安全加固

### TLS 加密通信

```yaml
security:
  tls:
    enabled: true
    cert_file: "/etc/waterflow/certs/client.crt"
    key_file: "/etc/waterflow/certs/client.key"
    ca_file: "/etc/waterflow/certs/ca.crt"
    insecure_skip_verify: false  # ✅ 生产环境必须为 false
```

**生成证书:**
```bash
# 1. 生成 CA
openssl genrsa -out ca.key 4096
openssl req -new -x509 -days 3650 -key ca.key -out ca.crt

# 2. 生成 Agent 证书
openssl genrsa -out agent.key 4096
openssl req -new -key agent.key -out agent.csr
openssl x509 -req -days 365 -in agent.csr -CA ca.crt -CAkey ca.key -out agent.crt
```

### 文件权限限制

```bash
# 配置文件仅 waterflow 用户可读
sudo chmod 640 /etc/waterflow/agent.yaml
sudo chown waterflow:waterflow /etc/waterflow/agent.yaml

# 证书文件仅 waterflow 用户可读
sudo chmod 600 /etc/waterflow/certs/*.key
sudo chown waterflow:waterflow /etc/waterflow/certs/*
```

### systemd 安全选项

```ini
[Service]
# ✅ 推荐的安全选项
NoNewPrivileges=true        # 禁止提权
PrivateTmp=true             # 隔离 /tmp
ProtectSystem=strict        # 只读文件系统
ProtectHome=true            # 隔离 /home
ReadWritePaths=/var/log/waterflow  # 仅允许写日志

# ❌ 不要使用
# User=root                 # 避免以 root 运行
# PermissionsStartOnly=true # 已废弃
```

## 5. 高可用部署

### 多 Agent 冗余

```bash
# 每个 Task Queue 至少 2 个 Agent
docker run -d --name agent-1 -e TASK_QUEUES=linux-amd64 waterflow/agent:latest
docker run -d --name agent-2 -e TASK_QUEUES=linux-amd64 waterflow/agent:latest

# ✅ 好处:
# - 故障自动转移
# - 负载均衡
# - 滚动升级

# Kubernetes HPA 自动扩缩容
kubectl autoscale deployment waterflow-agent --min=2 --max=10 --cpu-percent=70
```

### 跨地域部署

```yaml
# 美国西部 Agent
agent:
  id: "agent-us-west-1"
  task_queues:
    - "linux-amd64"
    - "us-west-1"  # 地域标签

# 欧洲中部 Agent
agent:
  id: "agent-eu-central-1"
  task_queues:
    - "linux-amd64"
    - "eu-central-1"
```

**工作流指定地域:**
```yaml
jobs:
  deploy-us:
    runs-on: us-west-1
    steps:
      - run: echo "Deploying to US West"
  
  deploy-eu:
    runs-on: eu-central-1
    steps:
      - run: echo "Deploying to EU Central"
```

## 6. 监控和告警

### Prometheus Metrics

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'waterflow-agents'
    static_configs:
      - targets:
        - 'agent-1:9090'
        - 'agent-2:9090'
    metrics_path: '/metrics'
    scrape_interval: 15s
```

**关键指标:**
```promql
# 活跃 Agent 数量
count(up{job="waterflow-agents"} == 1)

# Activity 执行失败率
rate(temporal_activity_execution_failed_total[5m]) /
rate(temporal_activity_execution_total[5m])

# Task Queue 轮询延迟
temporal_task_queue_poll_latency_seconds
```

### 告警规则

```yaml
# alerts.yml
groups:
- name: waterflow_agent
  rules:
  - alert: AgentDown
    expr: up{job="waterflow-agents"} == 0
    for: 2m
    labels:
      severity: critical
    annotations:
      summary: "Agent {{ $labels.instance }} is down"
  
  - alert: HighActivityFailureRate
    expr: |
      rate(temporal_activity_execution_failed_total[5m]) /
      rate(temporal_activity_execution_total[5m]) > 0.1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High activity failure rate (>10%)"
```

## 7. 资源限制

### Docker

```bash
docker run -d \
  --cpus="2.0" \           # 限制 2 个 CPU 核心
  --memory="2g" \          # 限制 2GB 内存
  --memory-swap="3g" \     # 限制 3GB 总内存
  --pids-limit=100 \       # 限制进程数
  waterflow/agent:latest
```

### Kubernetes

```yaml
resources:
  requests:
    cpu: "500m"      # 0.5 核心 (最低需求)
    memory: "512Mi"  # 512MB (最低需求)
  limits:
    cpu: "2000m"     # 2 核心 (最大使用)
    memory: "2Gi"    # 2GB (最大使用)
```

**资源规划建议:**
- CPU 密集型: `requests=1, limits=2`
- I/O 密集型: `requests=0.5, limits=1`
- 混合负载: `requests=0.5, limits=1.5`

## 8. 故障恢复

### 优雅关闭

```yaml
advanced:
  graceful_shutdown_timeout: "30s"  # 等待 30 秒完成当前任务
```

**systemd:**
```ini
[Service]
TimeoutStopSec=30s  # 与 graceful_shutdown_timeout 匹配
KillMode=mixed      # 先发送 SIGTERM,超时后 SIGKILL
```

**Kubernetes:**
```yaml
spec:
  terminationGracePeriodSeconds: 30  # 与 graceful_shutdown_timeout 匹配
```

### 自动重启策略

```ini
# systemd
[Service]
Restart=on-failure      # 仅在失败时重启
RestartSec=5s           # 重启前等待 5 秒
StartLimitBurst=3       # 1 分钟内最多重启 3 次
StartLimitIntervalSec=60s
```

```yaml
# Kubernetes
spec:
  restartPolicy: Always  # 总是重启
  # 或者使用 Pod Disruption Budget
---
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: waterflow-agent-pdb
spec:
  minAvailable: 1  # 至少保留 1 个 Pod 运行
  selector:
    matchLabels:
      app: waterflow-agent
```

## 9. 备份和灾难恢复

### 配置备份

```bash
# 自动备份脚本
#!/bin/bash
DATE=$(date +%Y%m%d)
tar -czf /backup/agent-config-$DATE.tar.gz \
  /etc/waterflow/agent.yaml \
  /etc/waterflow/certs/

# 保留最近 30 天备份
find /backup -name "agent-config-*.tar.gz" -mtime +30 -delete
```

### 多 AZ 部署 (AWS)

```yaml
# agent-deployment-us-east-1a.yaml
spec:
  nodeSelector:
    topology.kubernetes.io/zone: us-east-1a
  replicas: 2

# agent-deployment-us-east-1b.yaml
spec:
  nodeSelector:
    topology.kubernetes.io/zone: us-east-1b
  replicas: 2
```

## 10. 升级策略

### 滚动升级 (Zero Downtime)

```bash
# Docker Compose
docker-compose up -d --no-deps --build agent

# Kubernetes
kubectl set image deployment/waterflow-agent \
  agent=waterflow/agent:v1.2.0

# 监控升级进度
kubectl rollout status deployment/waterflow-agent
```

### 蓝绿部署

```bash
# 1. 部署新版本 (绿色)
kubectl apply -f agent-deployment-v2.yaml

# 2. 验证新版本正常
kubectl get pods -l version=v2

# 3. 切换流量 (更新 Service selector)
kubectl patch service waterflow-agent -p '{"spec":{"selector":{"version":"v2"}}}'

# 4. 删除旧版本 (蓝色)
kubectl delete deployment waterflow-agent-v1
```

### 回滚

```bash
# Kubernetes
kubectl rollout undo deployment/waterflow-agent

# Docker
docker run -d --name agent waterflow/agent:v1.0.0  # 使用旧版本镜像
```

## 11. Agent 升级指南

### 升级前准备

```bash
# 1. 备份当前配置
cp /etc/waterflow/agent.yaml /etc/waterflow/agent.yaml.backup.$(date +%Y%m%d)

# 2. 检查当前版本
./agent --version
# 输出: Waterflow Agent v1.0.0 (commit: abc123)

# 3. 查看 Release Notes
curl https://api.github.com/repos/youruser/waterflow/releases/latest | jq '.body'

# 4. 验证兼容性
# 检查配置文件是否需要更新
diff config.agent.example.yaml /etc/waterflow/agent.yaml
```

### 零停机升级 (推荐)

**步骤 1: 启动新版本 Agent**
```bash
# 启动新版本 Agent (不停止旧版本)
docker run -d \
  --name agent-v2 \
  -e TEMPORAL_SERVER_URL=temporal:7233 \
  -e TASK_QUEUES=linux-amd64 \
  waterflow/agent:v1.1.0
```

**步骤 2: 验证新版本正常**
```bash
# 检查新版本日志
docker logs agent-v2 | grep "Worker started"

# 验证新版本能接收任务
curl http://localhost:8080/v1/agents | jq '.agents[] | select(.agent_id | contains("agent-v2"))'
```

**步骤 3: 优雅停止旧版本**
```bash
# 停止旧版本 (等待当前任务完成)
docker stop -t 30 agent-v1

# 删除旧容器
docker rm agent-v1
```

### 快速升级 (允许短暂中断)

```bash
# 1. 拉取新镜像
docker pull waterflow/agent:v1.1.0

# 2. 停止旧容器
docker stop agent

# 3. 删除旧容器
docker rm agent

# 4. 启动新容器
docker run -d \
  --name agent \
  -e TEMPORAL_SERVER_URL=temporal:7233 \
  -e TASK_QUEUES=linux-amd64 \
  waterflow/agent:v1.1.0

# 总中断时间: ~10 秒
```

### Kubernetes 滚动升级

```bash
# 1. 更新镜像版本
kubectl set image deployment/waterflow-agent \
  agent=waterflow/agent:v1.1.0

# 2. 监控升级进度
kubectl rollout status deployment/waterflow-agent
# 输出: deployment "waterflow-agent" successfully rolled out

# 3. 验证新版本
kubectl get pods -l app=waterflow-agent
kubectl logs -l app=waterflow-agent --tail=20

# 4. 如果失败,立即回滚
kubectl rollout undo deployment/waterflow-agent
```

### systemd 服务升级

```bash
# 1. 下载新二进制
wget https://github.com/youruser/waterflow/releases/download/v1.1.0/agent-linux-amd64
chmod +x agent-linux-amd64

# 2. 备份旧版本
sudo mv /opt/waterflow/bin/agent /opt/waterflow/bin/agent.v1.0.0

# 3. 安装新版本
sudo mv agent-linux-amd64 /opt/waterflow/bin/agent

# 4. 重启服务
sudo systemctl restart waterflow-agent

# 5. 验证
sudo systemctl status waterflow-agent
sudo journalctl -u waterflow-agent -f
```

### 配置文件迁移

**场景: v1.1.0 新增配置字段**

```yaml
# 旧配置 (v1.0.0)
agent:
  task_queues: ["linux-amd64"]

# 新配置 (v1.1.0)
agent:
  task_queues: ["linux-amd64"]
  # 新增字段 (可选,有默认值)
  max_task_retries: 3  # 默认 3
  task_timeout: "5m"   # 默认 5 分钟
```

**迁移脚本** (`scripts/migrate-config.sh`):
```bash
#!/bin/bash
CONFIG="/etc/waterflow/agent.yaml"

# 检查是否已有新字段
if grep -q "max_task_retries" "$CONFIG"; then
    echo "配置已是最新版本"
    exit 0
fi

# 添加新字段 (使用默认值)
cat >> "$CONFIG" <<EOF

# v1.1.0 新增配置
  max_task_retries: 3
  task_timeout: "5m"
EOF

echo "✅ 配置已更新到 v1.1.0"
```

### 升级后验证

```bash
# 1. 版本确认
./agent --version
# 预期: v1.1.0

# 2. 配置验证
./agent --config /etc/waterflow/agent.yaml --validate
# 预期: Configuration is valid

# 3. 连接测试
curl http://localhost:8080/v1/agents | jq '.agents[] | .metadata.version'
# 预期: "v1.1.0"

# 4. 功能测试
# 提交测试工作流并验证执行成功
```

### 常见升级问题

**Q: 升级后 Agent 无法启动**
```bash
# 检查配置兼容性
./agent --config /etc/waterflow/agent.yaml --validate

# 如果配置不兼容,使用备份配置
cp /etc/waterflow/agent.yaml.backup.20251225 /etc/waterflow/agent.yaml

# 或回滚到旧版本
docker run -d --name agent waterflow/agent:v1.0.0
```

**Q: 升级后性能下降**
```bash
# 检查新版本的资源配置建议
cat CHANGELOG.md | grep -A5 "v1.1.0"

# 可能需要调整并发配置
max_concurrent_activities: 20  # 从 10 增加到 20
```
````

### AC5: 故障排查手册

**Given** Agent 运行异常  
**When** 参考故障排查手册  
**Then** 快速定位和解决问题

**故障排查文档** (`docs/guides/agent-troubleshooting.md`):
```markdown
# Agent 故障排查手册

## 常见问题索引

| 问题 | 可能原因 | 快速检查 |
|------|---------|---------|
| Agent 无法启动 | 配置错误、依赖缺失 | [→ 1.1](#11-agent-无法启动) |
| 无法连接 Temporal | 网络问题、URL 错误 | [→ 1.2](#12-无法连接-temporal) |
| Agent 未接收任务 | Queue 配置不匹配 | [→ 2.1](#21-agent-未接收任务) |
| Activity 执行失败 | Plugin 缺失、超时 | [→ 2.2](#22-activity-执行失败) |
| 内存占用过高 | 并发配置过高、内存泄漏 | [→ 3.1](#31-内存占用过高) |
| CPU 占用过高 | CPU 密集型任务 | [→ 3.2](#32-cpu-占用过高) |
| 心跳超时 | 网络延迟、服务器异常 | [→ 4.1](#41-心跳超时) |
| 日志丢失 | 日志配置错误 | [→ 5.1](#51-日志丢失) |

---

## 1. 启动问题

### 1.1 Agent 无法启动

**症状:**
```bash
$ ./agent --config config.yaml
Error: failed to load config: yaml: unmarshal errors:
  line 2: field temporal not found in type config.AgentConfig
```

**原因:** 配置文件格式错误

**解决:**
```bash
# 1. 验证配置文件语法
./agent --config config.yaml --validate

# 2. 对比示例配置
diff config.yaml config.agent.example.yaml

# 3. 使用 YAML Linter
yamllint config.yaml
```

---

### 1.2 无法连接 Temporal

**症状:**
```
[ERROR] Failed to create Temporal client: connection refused
[ERROR] Worker startup failed
```

**诊断步骤:**
```bash
# 1. 检查 Temporal Server 是否运行
telnet temporal.example.com 7233
# 或
nc -zv temporal.example.com 7233

# 2. 检查 Agent 配置
cat config.yaml | grep server_url
# 输出: server_url: "temporal.example.com:7233"

# 3. 检查 DNS 解析
nslookup temporal.example.com

# 4. 检查防火墙
sudo iptables -L -n | grep 7233

# 5. 检查 Docker 网络 (如果使用容器)
docker exec agent ping temporal
```

**常见原因:**
- ❌ `server_url: "http://localhost:7233"` (不应包含 http://)
- ✅ `server_url: "localhost:7233"` (正确格式)

---

## 2. 任务执行问题

### 2.1 Agent 未接收任务

**症状:**
```bash
# 提交工作流后,Agent 日志无任何输出
$ docker logs agent
[INFO] Worker started successfully
[INFO] Polling task queues: [linux-amd64]
# ... 没有后续日志
```

**诊断步骤:**
```bash
# 1. 检查 Agent 的 Task Queue 配置
docker logs agent | grep "Polling task queues"
# 输出: Polling task queues: [linux-amd64]

# 2. 检查工作流的 runs-on 配置
cat workflow.yaml | grep runs-on
# 输出: runs-on: linux-arm64  ← ❌ 不匹配!

# 3. 查询 Task Queue 状态
curl http://localhost:8080/v1/task-queues | jq '.task_queues[] | select(.name=="linux-amd64")'
# 输出: {"name":"linux-amd64","worker_count":0,...}  ← Worker 数量为 0!

# 4. 检查 Agent 是否已注册
curl http://localhost:8080/v1/agents?task_queue=linux-amd64
# 输出: {"agents":[],"total":0}  ← 未注册!
```

**解决:**
```bash
# 修正工作流配置
runs-on: linux-amd64  # 匹配 Agent 的 TASK_QUEUES

# 或者修改 Agent 配置
docker run -d -e TASK_QUEUES=linux-arm64,linux-amd64 waterflow/agent:latest
```

---

### 2.2 Activity 执行失败

**症状:**
```
[ERROR] Activity failed: activity type 'custom-plugin' not registered
```

**原因:** Agent 未加载自定义 Plugin

**解决:**
```bash
# 1. 检查 Plugin 目录
docker exec agent ls -la /app/plugins
# 输出: total 0  ← 目录为空!

# 2. 挂载 Plugin
docker run -d \
  -v /path/to/plugins:/app/plugins:ro \
  waterflow/agent:latest

# 3. 验证 Plugin 加载
docker logs agent | grep "Plugin loaded"
# 输出: [INFO] Plugin loaded successfully: my-plugin.so
```

---

## 3. 资源问题

### 3.1 内存占用过高

**症状:**
```bash
$ docker stats agent
CONTAINER    CPU %    MEM USAGE / LIMIT    MEM %
agent        50%      1.8GB / 2GB          90%  ← 内存接近限制!
```

**诊断:**
```bash
# 1. 检查并发配置
cat config.yaml | grep max_concurrent
# 输出: max_concurrent_activities: 50  ← 太高!

# 2. 检查是否有内存泄漏
# 观察内存使用趋势
docker stats agent --no-stream --format "table {{.MemUsage}}"

# 3. 查看 Go 运行时统计
curl http://agent:9090/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

**解决:**
```yaml
# 降低并发数
temporal:
  worker:
    max_concurrent_activities: 10  # 从 50 降低到 10

# 或增加内存限制
docker run -d --memory=4g waterflow/agent:latest
```

---

### 3.2 CPU 占用过高

**症状:**
```bash
$ top
PID   USER    %CPU  COMMAND
1234  waterflo 300%  /app/agent  ← CPU 使用率 300% (3 个核心)
```

**原因:** 可能是 CPU 密集型任务

**解决:**
```yaml
# 1. 降低并发数
temporal:
  worker:
    max_concurrent_activities: 4  # = CPU 核心数

# 2. 限制 CPU 使用
# Docker
docker run -d --cpus="2.0" waterflow/agent:latest

# Kubernetes
resources:
  limits:
    cpu: "2000m"
```

---

## 4. 网络问题

### 4.1 心跳超时

**症状:**
```
[WARN] Failed to update heartbeat: context deadline exceeded
[WARN] Agent may be marked as unhealthy
```

**诊断:**
```bash
# 1. 检查到 Server 的网络延迟
ping -c 5 waterflow-server.example.com

# 2. 测试 API 可达性
curl -w "@curl-format.txt" http://waterflow-server:8080/v1/agents/heartbeat

# curl-format.txt:
#   time_namelookup:  %{time_namelookup}\n
#   time_connect:  %{time_connect}\n
#   time_total:  %{time_total}\n

# 3. 检查防火墙规则
sudo iptables -L -n -v | grep 8080
```

**解决:**
```yaml
# 增加心跳超时时间
advanced:
  heartbeat_interval: "60s"  # 从 30s 增加到 60s

# 或配置 HTTP Proxy
environment:
  - HTTP_PROXY=http://proxy.example.com:8080
```

---

## 5. 日志和监控

### 5.1 日志丢失

**症状:**
```bash
$ docker logs agent
# 输出为空
```

**诊断:**
```bash
# 1. 检查日志配置
docker exec agent cat /app/config/config.yaml | grep -A5 logger

# 2. 检查日志文件
docker exec agent ls -lh /var/log/waterflow/
# 输出: total 0  ← 没有日志文件!

# 3. 检查日志目录权限
docker exec agent stat /var/log/waterflow/
# 输出: Access: (0755/drwxr-xr-x)  Uid: (    0/    root)  ← 权限问题!
```

**解决:**
```bash
# 修正目录权限
docker exec agent chown -R waterflow:waterflow /var/log/waterflow

# 或使用 stdout 输出
logger:
  output: "stdout"  # 不写入文件
```

---

## 6. 高级诊断

### 6.1 启用 Debug 日志

```yaml
logger:
  level: "debug"  # 启用详细日志
```

**重启 Agent:**
```bash
docker restart agent
docker logs -f agent  # 查看详细日志
```

### 6.2 使用 pprof 分析

```bash
# 1. 访问 pprof 端点
curl http://agent:9090/debug/pprof/

# 2. 生成 CPU profile
curl http://agent:9090/debug/pprof/profile?seconds=30 > cpu.prof

# 3. 分析
go tool pprof cpu.prof
> top10  # 查看 CPU 占用最高的 10 个函数
```

### 6.3 Temporal Web UI

```bash
# 访问 Temporal Web UI
open http://localhost:8080  # 默认端口

# 查看 Worker 状态:
# Workflows → Task Queues → <your-queue> → Workers
```

---

## 7. 紧急恢复流程

### 7.1 Agent 完全不响应

```bash
# 1. 强制重启
docker restart -t 0 agent  # 立即重启,不等待优雅关闭

# 2. 如果仍无响应,删除并重建
docker rm -f agent
docker run -d --name agent -e TASK_QUEUES=linux-amd64 waterflow/agent:latest

# 3. 检查工作流任务是否恢复
curl http://localhost:8080/v1/task-queues
```

### 7.2 任务积压

**症状:** 大量任务等待执行

**解决:**
```bash
# 1. 快速扩容 Agent
docker run -d --name agent-2 -e TASK_QUEUES=linux-amd64 waterflow/agent:latest
docker run -d --name agent-3 -e TASK_QUEUES=linux-amd64 waterflow/agent:latest

# Kubernetes
kubectl scale deployment waterflow-agent --replicas=10

# 2. 监控任务处理速度
watch 'curl -s http://localhost:8080/v1/task-queues | jq ".task_queues[] | select(.name==\"linux-amd64\")"'
```

---

## 8. 获取帮助

如果以上方法无法解决问题,请提供以下信息:

```bash
# 收集诊断信息
cat > diagnosis.txt <<EOF
Agent Version: $(docker exec agent /app/agent --version)
Config:
$(docker exec agent cat /app/config/config.yaml)

Recent Logs:
$(docker logs --tail=100 agent)

System Info:
$(docker exec agent uname -a)
$(docker exec agent cat /etc/os-release)

Network:
$(docker exec agent ip addr)
$(docker exec agent netstat -tuln)
EOF

# 提交 Issue: https://github.com/yourusername/waterflow/issues
```
```

### AC6: 监控集成指南

**Given** 用户需要监控 Agent  
**When** 集成 Prometheus/Grafana  
**Then** 可视化 Agent 运行状态

**监控集成文档** (`docs/guides/agent-monitoring.md`):
```markdown
# Agent 监控集成指南

## Prometheus 集成

### 1. 配置 Prometheus

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'waterflow-agents'
    static_configs:
      - targets:
        - 'agent-1:9090'
        - 'agent-2:9090'
    metrics_path: '/metrics'
    
    # 服务发现 (Kubernetes)
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - waterflow
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        action: keep
        regex: waterflow-agent
```

### 2. Grafana Dashboard

导入预制 Dashboard: [Waterflow Agent Dashboard (ID: 12345)](https://grafana.com/dashboards/12345)

或手动创建:

**Panel 1: Agent 数量**
```promql
count(up{job="waterflow-agents"} == 1)
```

**Panel 2: Activity 执行率**
```promql
rate(temporal_activity_execution_total[5m])
```

**Panel 3: Activity 失败率**
```promql
rate(temporal_activity_execution_failed_total[5m]) /
rate(temporal_activity_execution_total[5m]) * 100
```

**Panel 4: Task Queue 轮询延迟**
```promql
histogram_quantile(0.99, rate(temporal_task_queue_poll_latency_seconds_bucket[5m]))
```

## Datadog 集成

```yaml
# datadog-agent.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: datadog-checks
data:
  prometheus.yaml: |
    instances:
      - prometheus_url: http://agent:9090/metrics
        namespace: waterflow
        metrics:
          - temporal_*
```

## 告警规则

```yaml
groups:
  - name: waterflow_agent_alerts
    rules:
      - alert: AgentDown
        expr: up{job="waterflow-agents"} == 0
        for: 2m
        annotations:
          summary: "Agent {{ $labels.instance }} is down"
      
      - alert: HighFailureRate
        expr: rate(temporal_activity_execution_failed_total[5m]) > 10
        for: 5m
        annotations:
          summary: "High activity failure rate on {{ $labels.instance }}"
```
```

### AC7: README 快速链接和概述

**Given** 新用户访问文档  
**When** 查看 README  
**Then** 快速找到所需文档

**更新 README** (`docs/guides/agent-README.md`):
```markdown
# Waterflow Agent 文档中心

## 📖 快速导航

| 阶段 | 文档 | 说明 |
|------|------|------|
| **开始** | [快速开始](./agent-quickstart.md) | 5 分钟启动第一个 Agent |
| **配置** | [配置详解](../sprint-artifacts/2-10-agent-configuration-guide.md) | 完整配置文件说明 |
| **部署** | [最佳实践](./agent-best-practices.md) | 生产环境部署建议 |
| **故障排查** | [故障排查](./agent-troubleshooting.md) | 常见问题解决方案 |
| **监控** | [监控集成](./agent-monitoring.md) | Prometheus/Grafana 集成 |

## 🚀 5 分钟快速开始

```bash
# 1. 启动 Agent (Docker)
docker run -d \
  --name waterflow-agent \
  -e TEMPORAL_SERVER_URL=temporal:7233 \
  -e TASK_QUEUES=linux-amd64 \
  waterflow/agent:latest

# 2. 验证运行
docker logs waterflow-agent

# 3. 查询状态
curl http://localhost:8080/v1/agents
```

## 📋 常见部署场景

### 场景 1: 本地开发
→ [Docker Compose 快速开始](./agent-quickstart.md#方式-2-docker-compose-推荐)

### 场景 2: 生产环境 (Kubernetes)
→ [Kubernetes 部署](../sprint-artifacts/2-9-agent-docker-image.md#ac6-kubernetes-部署支持)

### 场景 3: 裸机服务器
→ [systemd 部署](../sprint-artifacts/2-10-agent-configuration-guide.md#ac2-systemd-服务单元文件)

## 🔧 配置示例

### 基本配置
```yaml
agent:
  task_queues: ["linux-amd64"]
temporal:
  server_url: "localhost:7233"
```

### 高级配置
→ [配置最佳实践](./agent-best-practices.md#1-task-queue-规划)

## ❓ 遇到问题?

1. 查看 [故障排查手册](./agent-troubleshooting.md)
2. 搜索 [GitHub Issues](https://github.com/yourusername/waterflow/issues)
3. 加入 [Slack 社区](#)
```

## Developer Context

### 文档结构

```
docs/
├── guides/
│   ├── agent-README.md           # 文档导航中心
│   ├── agent-quickstart.md       # 5 分钟快速开始
│   ├── agent-best-practices.md   # 配置最佳实践
│   ├── agent-troubleshooting.md  # 故障排查手册
│   └── agent-monitoring.md       # 监控集成指南
├── sprint-artifacts/
│   ├── 2-1-agent-worker-framework.md
│   ├── 2-9-agent-docker-image.md
│   └── 2-10-agent-configuration-guide.md  # 本 Story
└── examples/
    └── agent-configs/
        ├── basic.yaml
        ├── advanced.yaml
        ├── multi-queue.yaml
        └── production.yaml
```

### 文档使用流程

```
用户旅程:
1. agent-README.md (入口) → 选择场景
2. agent-quickstart.md (快速开始) → 5 分钟上手
3. 2-10-agent-configuration-guide.md (深入配置) → 理解所有配置项
4. agent-best-practices.md (优化配置) → 生产环境调优
5. agent-troubleshooting.md (遇到问题) → 快速解决
```

### 实现策略

**优先级 1: 核心文档 (必须)**
- ✅ `config.agent.example.yaml` - 配置模板
- ✅ `agent-quickstart.md` - 快速开始
- ✅ `deployments/systemd/waterflow-agent.service` - systemd 配置
- ✅ `scripts/install-agent.sh` - 安装脚本

**优先级 2: 指南文档 (推荐)**
- ✅ `agent-best-practices.md` - 最佳实践
- ✅ `agent-troubleshooting.md` - 故障排查
- ✅ `agent-README.md` - 文档导航

**优先级 3: 高级文档 (可选)**
- `agent-monitoring.md` - 监控集成
- `examples/agent-configs/` - 配置示例
- 视频教程、交互式文档

## Dev Notes

### 文档编写最佳实践

**DO ✅:**
- 提供可运行的示例代码
- 使用表格快速对比方案
- 包含故障现象和诊断步骤
- 从用户角度组织内容 (按场景而非功能)

**DON'T ❌:**
- 假设用户了解所有术语
- 仅提供理论说明,无实际命令
- 文档与代码不同步
- 过度使用技术术语

### 测试文档准确性

```bash
# 1. 验证所有命令可执行
grep -r '```bash' docs/guides/agent-*.md | \
  sed 's/.*```bash//; s/```$//' | \
  while read cmd; do eval "$cmd" || echo "Failed: $cmd"; done

# 2. 验证配置文件语法
./agent --config config.agent.example.yaml --validate

# 3. 验证链接有效性
npm install -g markdown-link-check
markdown-link-check docs/guides/*.md
```

## Dev Agent Record

### File List

**新增文件:**
- `config.agent.example.yaml` (~150 行)
- `deployments/systemd/waterflow-agent.service` (~50 行)
- `scripts/install-agent.sh` (~80 行)
- `docs/guides/agent-README.md` (~80 行)
- `docs/guides/agent-quickstart.md` (~300 行)
- `docs/guides/agent-best-practices.md` (~600 行)
- `docs/guides/agent-troubleshooting.md` (~500 行)
- `docs/guides/agent-monitoring.md` (~150 行)

**总计:** ~1910 新增文档行

**文档交付物:**
- 8 个 Markdown 文档
- 1 个 YAML 配置模板
- 1 个 systemd Service 文件
- 1 个 Shell 安装脚本
