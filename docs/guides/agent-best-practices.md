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

### Activity 心跳超时配置

> **注意：** Agent 不再需要配置 `heartbeat_interval`（已在 ADR-0008 中移除）。  
> 以下配置指的是 Temporal Activity 的心跳超时设置。

```yaml
advanced:
  # Activity 心跳超时 - Temporal 内部检测机制
  activity_heartbeat_timeout: "10s"  # 短超时，快速检测 Activity 失败
  activity_heartbeat_timeout: "30s"  # ✅ 推荐默认值
  activity_heartbeat_timeout: "60s"  # 长超时，适用于稳定环境
```

**说明:**
- Activity 心跳由 Temporal 自动管理
- Agent 通过 Temporal Worker 自动注册和连接

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

# Docker Compose 多实例部署
docker compose up -d --scale agent-linux=5
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

### 自动重启策略

```ini
# systemd
[Service]
Restart=on-failure      # 仅在失败时重启
RestartSec=5s           # 重启前等待 5 秒
StartLimitBurst=3       # 1 分钟内最多重启 3 次
StartLimitIntervalSec=60s
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

## 10. 升级策略

### 滚动升级 (Zero Downtime)

```bash
# Docker Compose
docker-compose up -d --no-deps --build agent
```

### 回滚

```bash
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
curl https://api.github.com/repos/Websoft9/Waterflow/releases/latest | jq '.body'

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

### systemd 服务升级

```bash
# 1. 下载新二进制
wget https://github.com/Websoft9/Waterflow/releases/download/v1.1.0/agent-linux-amd64
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
