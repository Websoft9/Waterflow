# Waterflow 故障排查手册

本文档提供 Waterflow 常见问题的诊断和解决方法，帮助用户快速定位和解决生产环境问题。

## 目录
- [常见问题](#常见问题)
  - [服务无法启动](#服务无法启动)
  - [Temporal 连接失败](#temporal-连接失败)
  - [数据库连接失败](#数据库连接失败)
  - [工作流提交失败](#工作流提交失败)
  - [工作流执行失败](#工作流执行失败)
  - [工作流执行卡住](#工作流执行卡住)
  - [Agent 无法连接](#agent-无法连接)
  - [日志查询失败](#日志查询失败)
  - [Docker 部署问题](#docker-部署问题)
  - [性能问题](#性能问题)
- [日志分析](#日志分析)
  - [Server 日志查看和分析](#server-日志查看和分析)
  - [Agent 日志查看和分析](#agent-日志查看和分析)
- [工作流调试](#工作流调试)
  - [工作流执行失败调试流程](#工作流执行失败调试流程)
  - [调试技巧和最佳实践](#调试技巧和最佳实践)
- [诊断工具](#诊断工具)
  - [健康检查端点](#健康检查端点)
  - [日志查看命令](#日志查看命令)
  - [资源监控](#资源监控)
  - [网络诊断](#网络诊断)
- [诊断命令速查表](#诊断命令速查表)
- [获取支持](#获取支持)

---

## 常见问题

### 服务无法启动

#### 症状
- Docker 容器启动后立即退出
- Systemd 服务启动失败
- 健康检查一直返回错误

#### 诊断步骤

**1. 检查端口占用**
```bash
# 检查 Waterflow 端口
netstat -tuln | grep 8080

# 检查 Temporal 端口
netstat -tuln | grep 7233

# 检查 PostgreSQL 端口
netstat -tuln | grep 5432
```

**2. 查看错误日志**
```bash
# Docker Compose 环境
cd deployments
docker-compose logs waterflow

# Kubernetes 环境
kubectl logs -l app=waterflow-server --tail=100

# Systemd 环境
sudo journalctl -u waterflow-server -n 100 --no-pager
```

**3. 检查配置文件**
```bash
# 验证配置文件语法
./bin/server --config /etc/waterflow/config.yaml --validate
```

#### 常见原因和解决方案

**原因 1: 端口被占用**
```bash
# 查找占用进程
sudo lsof -i :8080

# 停止占用进程或修改配置使用其他端口
vi deployments/.env
# 修改 WATERFLOW_SERVER_PORT=9090
```

**原因 2: 配置文件错误**
```bash
# 检查 YAML 语法
yamllint /etc/waterflow/config.yaml

# 使用示例配置
cp examples/configs/config.example.yaml /etc/waterflow/config.yaml
```

**原因 3: 权限问题**
```bash
# 检查文件权限
ls -l /etc/waterflow/config.yaml

# 修复权限
sudo chown waterflow:waterflow /etc/waterflow/config.yaml
sudo chmod 600 /etc/waterflow/config.yaml
```

---

### Temporal 连接失败

#### 症状
- `/ready` 端点返回 503
- 日志显示 "temporal: connection refused"
- 工作流提交失败

#### 诊断步骤

**1. 检查 Temporal 服务状态**
```bash
# Docker Compose
docker-compose ps temporal

# 查看 Temporal 日志
docker-compose logs temporal
```

**2. 验证网络连通性**
```bash
# 从 Waterflow 容器测试连接
docker-compose exec waterflow nc -zv temporal 7233

# 从宿主机测试（如果 Temporal 暴露端口）
nc -zv localhost 7233
```

**3. 检查配置**
```bash
# 查看 Temporal 配置
docker-compose exec waterflow env | grep TEMPORAL
```

#### 常见原因和解决方案

**原因 1: Temporal 服务未启动**
```bash
# 启动 Temporal
cd deployments
docker-compose up -d temporal

# Kubernetes 环境
kubectl get pods -l app=temporal

# 等待 Temporal 完全启动（约 30 秒）
docker-compose logs -f temporal
```

**原因 2: 配置地址错误**
```bash
# Docker Compose 环境必须使用服务名
WATERFLOW_TEMPORAL_HOST=temporal:7233  # ✅ 正确
# WATERFLOW_TEMPORAL_HOST=localhost:7233  # ❌ 错误（容器内无法访问宿主机 localhost）

# Kubernetes 环境使用 Service DNS
WATERFLOW_TEMPORAL_HOST=temporal.default.svc.cluster.local:7233  # ✅ 正确

# 二进制部署可以使用 localhost
WATERFLOW_TEMPORAL_HOST=localhost:7233  # ✅ 正确（直接部署）
```

**原因 3: 网络问题**
```bash
# 检查 Docker 网络
docker network ls
docker network inspect waterflow-network

# Kubernetes 检查网络策略
kubectl get networkpolicies

# 重建网络（Docker Compose）
docker-compose down
docker-compose up -d
```

**原因 4: 连接重试配置不当**
```yaml
# config.yaml 检查重试配置
temporal:
  max_retries: 5          # 最大重试次数
  retry_interval: 2s      # 重试间隔
  connection_timeout: 10s # 连接超时
```

---

### 数据库连接失败

#### 症状
- Temporal 启动失败
- 日志显示 "database connection refused"
- PostgreSQL 健康检查失败

#### 诊断步骤

**1. 检查 PostgreSQL 状态**
```bash
docker-compose ps postgresql
docker-compose logs postgresql
```

**2. 验证数据库连接**
```bash
# 从 Temporal 容器测试
docker-compose exec temporal nc -zv postgresql 5432

# 直接连接数据库
docker-compose exec postgresql psql -U temporal -d temporal -c "SELECT 1"
```

#### 解决方案

**重启 PostgreSQL**
```bash
cd deployments
docker-compose restart postgresql

# 等待健康检查通过
docker-compose ps postgresql
```

**检查数据卷**
```bash
# 查看数据卷
docker volume ls | grep waterflow

# 检查数据卷完整性
docker volume inspect deployments_postgresql-data
```

---

### 工作流提交失败

#### 症状
- POST `/v1/workflows` 返回 4xx/5xx
- 工作流无法创建
- YAML 验证失败

#### 诊断步骤

**1. 验证 YAML 语法**
```bash
# 使用验证端点
curl -X POST http://localhost:8080/v1/workflows/validate \
  -H "Content-Type: application/json" \
  -d @workflow.json
```

**2. 检查错误响应**
```bash
# 查看详细错误信息
curl -v -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/json" \
  -d @workflow.json
```

#### 常见原因和解决方案

**原因 1: YAML 语法错误**
- 检查缩进（必须使用空格，不能使用 Tab）
- 验证必需字段（name, on, jobs）
- 使用在线 YAML 验证工具

**原因 2: 节点不存在**
```bash
# 查看可用节点
curl http://localhost:8080/v1/nodes

# 使用正确的节点名称
```

**原因 3: Task Queue 不匹配**
- 确保工作流的 `runs-on` 值与 Agent 的 `task_queues` 匹配

---

### 工作流执行失败

#### 症状
- 工作流状态为 "failed"
- Step 执行报错
- 节点返回错误信息

#### 诊断步骤

**1. 查看工作流状态和错误**
```bash
# 查询工作流详情
curl http://localhost:8080/v1/workflows/{workflow-id}

# 查看错误字段
curl http://localhost:8080/v1/workflows/{workflow-id} | jq '.error'
```

**2. 查看工作流日志**
```bash
# 获取完整日志
curl http://localhost:8080/v1/workflows/{workflow-id}/logs

# 只看错误日志
curl http://localhost:8080/v1/workflows/{workflow-id}/logs?level=error

# 过滤特定 Job
curl http://localhost:8080/v1/workflows/{workflow-id}/logs?job=deploy
```

**3. 在 Temporal UI 查看执行历史**
```bash
# 打开 Temporal UI
open http://localhost:8088

# 搜索 workflowID，查看：
# - Event History（完整执行历史）
# - Activity 失败详情（错误消息、堆栈跟踪）
# - Retry 状态和次数
```

#### 常见原因和解决方案

**原因 1: 节点执行错误**
```bash
# 检查节点参数
# 示例：shell 命令失败
{
  "error": "Activity failed: exit status 1",
  "step": "deploy",
  "node": "exec/shell@v1"
}

# 解决方法：
# 1. 检查命令是否正确
# 2. 验证参数和环境变量
# 3. 检查目标服务器状态
```

**原因 2: 参数验证失败**
```json
{
  "type": "validation_error",
  "title": "Parameter Validation Failed",
  "detail": "required parameter 'command' is missing"
}

// 解决方法：补充必需参数
```

**原因 3: 权限不足**
```bash
# SSH 权限问题
# 错误: "Permission denied (publickey)"

# 解决方法：
# 1. 验证 SSH 密钥配置
ssh-copy-id user@target-host

# 2. 检查文件权限
chmod 600 ~/.ssh/id_rsa

# 3. 使用 with.private_key 参数传递密钥
```

**原因 4: 超时配置不当**
```yaml
# 检查超时配置
jobs:
  deploy:
    timeout-minutes: 30  # Job 级超时
    steps:
      - timeout-minutes: 5  # Step 级超时

# 确保 Step 超时 < Job 超时
```

**原因 5: Matrix 组合数超限**
```yaml
# 错误：Matrix combinations 300 exceed limit 256
strategy:
  matrix:
    server: [1..100]  # 100 个
    env: [dev, staging, prod]  # 3 个
    # 100 × 3 = 300 > 256 ❌

# 解决方法：减少组合数或拆分为多个 Job
```

**原因 6: 表达式求值失败**
```yaml
# 错误：Undefined variable: vars.undefined_var
steps:
  - with:
      command: echo
      args: ["${{ vars.undefined_var }}"]  # ❌ 未定义

# 解决方法：在 vars 中定义变量
vars:
  undefined_var: "value"
```

---

### 工作流执行卡住

#### 症状
- 工作流状态一直是 "running"
- 步骤长时间无进展
- Agent 日志无输出

#### 诊断步骤

**1. 检查工作流状态**
```bash
# 查看工作流详情
curl http://localhost:8080/v1/workflows/{workflow-id}

# 在 Temporal UI 查看
open http://localhost:8088
```

**2. 检查 Agent 状态**
```bash
# Docker Compose
docker-compose ps agent-linux-1
docker-compose logs agent-linux-1

# 查看 Agent 连接状态
curl http://localhost:9090/metrics | grep agent
```

**3. 检查 Task Queue**
```bash
# 在 Temporal UI 查看 Task Queue 深度
open http://localhost:8088/namespaces/default/task-queues
```

#### 解决方案

**重启 Agent**
```bash
cd deployments
docker-compose restart agent-linux-1
```

**取消卡住的工作流**
```bash
curl -X POST http://localhost:8080/v1/workflows/{workflow-id}/cancel
```

---

### Agent 无法连接

#### 症状
- Agent 日志显示 "Failed to connect to Temporal"
- Task Queue 无 Worker 可用
- 工作流提交后无 Agent 执行

#### 诊断步骤

**1. 检查 Agent 状态**
```bash
# Docker Compose
docker-compose ps agent-linux-1
docker-compose logs agent-linux-1

# Kubernetes
kubectl get pods -l app=waterflow-agent
kubectl logs -l app=waterflow-agent --tail=100

# 二进制部署
systemctl status waterflow-agent
journalctl -u waterflow-agent -n 100
```

**2. 验证 Temporal 连接配置**
```bash
# 检查 Agent 配置
docker-compose exec agent-linux-1 env | grep TEMPORAL

# Kubernetes
kubectl exec waterflow-agent-0 -- env | grep WATERFLOW_TEMPORAL
```

**3. 检查网络连通性**
```bash
# 从 Agent 容器测试
docker-compose exec agent-linux-1 nc -zv temporal 7233

# Kubernetes
kubectl exec waterflow-agent-0 -- nc -zv temporal.default.svc.cluster.local 7233
```

#### 常见原因和解决方案

**原因 1: Task Queue 配置不匹配**
```yaml
# 工作流 YAML
jobs:
  deploy:
    runs-on: web-servers  # Task Queue 名称

# Agent 配置必须匹配
# config.yaml
agent:
  task_queues:
    - web-servers  # ✅ 匹配

# 解决方法：确保 runs-on 与 Agent task_queues 一致
```

**原因 2: Temporal 地址配置错误**
```bash
# Docker Compose 环境使用服务名
WATERFLOW_TEMPORAL_HOST=temporal:7233  # ✅ 正确

# Kubernetes 使用 Service DNS
WATERFLOW_TEMPORAL_HOST=temporal.default.svc.cluster.local:7233  # ✅ 正确

# 常见错误：
# WATERFLOW_TEMPORAL_HOST=localhost:7233  # ❌ 容器内无法访问
```

**原因 3: 网络防火墙阻止连接**
```bash
# 检查防火墙规则
sudo iptables -L -n | grep 7233

# Kubernetes 检查网络策略
kubectl get networkpolicies

# 临时禁用防火墙测试
sudo systemctl stop firewalld
```

**原因 4: 插件加载失败**
```bash
# Agent 日志错误：
# "Plugin load failed: /plugins/exec-shell.so: cannot open shared object file"

# 解决方法：
# 1. 检查插件目录完整性
ls -lh /path/to/plugins/

# 2. 重新构建 Agent 镜像
docker-compose build agent-linux-1

# 3. 验证插件版本兼容性
```

---

### 日志查询失败

#### 症状
- `/v1/workflows/{id}/logs` 返回 404 或空结果
- 日志延迟严重
- 日志内容不完整

#### 诊断步骤

**1. 检查日志存储**
```bash
# Docker Compose（如果配置了卷挂载）
docker-compose exec waterflow ls -lh /var/log/waterflow/

# 检查磁盘空间
df -h

# 查看 Docker 日志大小
docker inspect waterflow-server | jq '.[0].HostConfig.LogConfig'
```

**2. 检查 LogHandler 配置**
```yaml
# config.yaml
log:
  handler: stdout  # stdout, file, elk, loki
  file:
    path: /var/log/waterflow/
    max_size: 100  # MB
```

**3. 验证工作流是否生成日志**
```bash
# 查看 Temporal UI Event History
open http://localhost:8088

# 检查 Activity 日志输出
docker-compose logs agent-linux-1 | grep workflow_id
```

#### 常见原因和解决方案

**原因 1: 日志存储空间不足**
```bash
# 检查磁盘使用
df -h

# 清理旧日志
find /var/log/waterflow/ -name "*.log" -mtime +7 -delete

# 增加磁盘空间（Docker 卷）
docker volume ls
```

**原因 2: 日志格式错误**
```bash
# 检查日志是否为有效 JSON
tail -n 1 /var/log/waterflow/workflow.log | jq .

# 如果解析失败，检查日志输出配置
```

**原因 3: LogHandler 集成问题**
```bash
# ELK/Loki 连接失败
# 错误：connection refused to elasticsearch:9200

# 解决方法：
# 1. 验证外部日志系统可用性
curl http://elasticsearch:9200

# 2. 检查认证配置
# 3. 降级为 stdout handler
```

---

### Docker 部署问题

#### 症状
- 容器无法启动
- 镜像拉取失败
- 卷挂载错误

#### 诊断步骤

**1. 检查容器状态**
```bash
docker-compose ps
docker-compose logs waterflow
```

**2. 检查镜像**
```bash
# 列出本地镜像
docker images | grep waterflow

# 拉取最新镜像
docker-compose pull
```

**3. 检查卷挂载**
```bash
# 查看卷
docker volume ls | grep waterflow

# 检查卷详情
docker volume inspect deployments_waterflow-data
```

#### 常见原因和解决方案

**原因 1: 镜像拉取失败**
```bash
# 错误：Error response from daemon: pull access denied

# 解决方法：
# 1. 检查网络连接
curl -I https://ghcr.io

# 2. 配置镜像加速器
# /etc/docker/daemon.json
{
  "registry-mirrors": ["https://mirror.ccs.tencentyun.com"]
}

# 3. 手动拉取镜像
docker pull ghcr.io/websoft9/waterflow:latest
```

**原因 2: 卷挂载权限错误**
```bash
# 错误：permission denied

# 解决方法：
# 1. 检查目录权限
ls -ld ./data/

# 2. 修复权限
sudo chown -R 1000:1000 ./data/

# 3. 使用绝对路径
# docker-compose.yaml
volumes:
  - /absolute/path/to/data:/data
```

**原因 3: 端口冲突**
```bash
# 错误：bind: address already in use

# 解决方法：
# 1. 查找占用进程
sudo lsof -i :8080

# 2. 修改端口映射
# .env
WATERFLOW_SERVER_PORT=9090
```

**原因 4: 网络配置错误**
```bash
# 检查 Docker 网络
docker network ls
docker network inspect waterflow-network

# 重建网络
docker-compose down
docker network prune
docker-compose up -d
```

---

### 性能问题

#### 症状
- 工作流执行缓慢
- 内存占用高
- CPU 使用率高

#### 诊断步骤

**1. 查看资源使用**
```bash
# Docker 容器资源
docker stats

# 系统资源
top
free -h
df -h
```

**2. 检查工作流并发数**
```bash
# 查看运行中的工作流数量
curl http://localhost:8080/v1/workflows?status=running
```

**3. 分析慢查询**
```bash
# 查看 PostgreSQL 慢查询
docker-compose exec postgresql psql -U temporal -d temporal \
  -c "SELECT query, calls, total_time FROM pg_stat_statements ORDER BY total_time DESC LIMIT 10"
```

#### 解决方案

**增加资源限制**
```yaml
# docker-compose.yaml
services:
  waterflow:
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 4G
```

**调整并发配置**
```yaml
# config.yaml
agent:
  max_concurrent_activities: 10  # 根据 CPU 核数调整
```

**优化数据库**
```bash
# 清理历史数据
docker-compose exec postgresql psql -U temporal -d temporal \
  -c "DELETE FROM events WHERE created_at < NOW() - INTERVAL '30 days'"
```

---

## 日志分析

### Server 日志查看和分析

#### 日志查看命令

**Docker Compose 环境**
```bash
# 查看所有日志
docker-compose logs waterflow

# 最近 100 行
docker-compose logs --tail=100 waterflow

# 实时跟踪
docker-compose logs -f waterflow

# 时间范围过滤
docker-compose logs --since 2026-01-15T10:00:00 waterflow
docker-compose logs --since 1h waterflow  # 最近 1 小时
```

**Systemd 环境**
```bash
# 最近 100 行
journalctl -u waterflow-server -n 100 --no-pager

# 实时跟踪
journalctl -u waterflow-server -f

# 时间范围
journalctl -u waterflow-server --since "2026-01-15 10:00:00"
journalctl -u waterflow-server --since "1 hour ago"

# 禁用分页器（便于 grep）
journalctl -u waterflow-server --no-pager
```

**Kubernetes 环境**
```bash
# 查看 Pod 日志
kubectl logs -l app=waterflow-server

# 最近 100 行
kubectl logs -l app=waterflow-server --tail=100

# 实时跟踪
kubectl logs -f waterflow-server-0

# 时间范围（最近 1 小时）
kubectl logs --since=1h waterflow-server-0

# 所有副本日志
kubectl logs -l app=waterflow-server --all-containers=true
```

#### 结构化日志字段说明

Waterflow 使用 JSON 格式结构化日志，包含以下标准字段：

| 字段 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `timestamp` | string | ISO 8601 时间戳 | `2026-01-15T10:30:45.123Z` |
| `level` | string | 日志级别 | `debug`, `info`, `warn`, `error` |
| `workflow_id` | string | 工作流 ID（如果关联） | `wf-deploy-prod-20260115` |
| `run_id` | string | Temporal Run ID | `abc123...` |
| `component` | string | 组件名称 | `api`, `executor`, `validator` |
| `message` | string | 日志消息 | `Workflow started successfully` |
| `error` | object | 错误详情（仅 error 级别） | `{type, title, detail}` |
| `duration` | number | 操作耗时（毫秒） | `1234` |
| `user_id` | string | 用户标识（如果有认证） | `admin` |

**示例日志：**
```json
{
  "timestamp": "2026-01-15T10:30:45.123Z",
  "level": "error",
  "workflow_id": "wf-deploy-prod-20260115",
  "component": "executor",
  "message": "Step execution failed",
  "error": {
    "type": "node_execution_error",
    "title": "Shell Command Failed",
    "detail": "exit status 1",
    "step": "deploy"
  },
  "duration": 1234
}
```

#### 日志级别过滤

```bash
# grep 过滤（Docker/Systemd）
docker-compose logs waterflow | grep '"level":"error"'
journalctl -u waterflow-server --no-pager | grep '"level":"error"'

# jq 过滤（推荐）
docker-compose logs waterflow | jq 'select(.level=="error")'
kubectl logs waterflow-server-0 | jq 'select(.level=="error")'

# 多级别过滤
jq 'select(.level=="error" or .level=="warn")'

# 统计错误数量
grep '"level":"error"' | wc -l
```

#### 常见错误日志模式和诊断

| 错误模式 | 问题类型 | 诊断方法 |
|----------|----------|----------|
| `"connection refused"` | Temporal 连接失败 | 检查 Temporal 服务状态: `docker-compose ps temporal` |
| `"validation_error"` | YAML 语法错误 | 运行本地验证: `waterflow validate workflow.yaml` |
| `"timeout exceeded"` | 超时问题 | 检查 `timeout-minutes` 配置和网络延迟 |
| `"permission denied"` | 权限不足 | 检查文件权限、SSH 密钥、API 认证 |
| `"node not found"` | 节点不存在 | 查看可用节点: `curl /v1/nodes` |
| `"task queue not found"` | Task Queue 不匹配 | 检查 `runs-on` 与 Agent `task_queues` 配置 |
| `"database connection failed"` | 数据库问题 | 检查 PostgreSQL 状态: `docker-compose ps postgresql` |
| `"out of memory"` | 内存不足 | 检查资源使用: `docker stats`, 增加内存限制 |

#### 日志查询示例

**查询特定工作流日志：**
```bash
# grep 方式
docker-compose logs waterflow | grep '"workflow_id":"wf-deploy-prod"'

# jq 方式
docker-compose logs waterflow | jq 'select(.workflow_id=="wf-deploy-prod")'
```

**查询 API 错误日志：**
```bash
docker-compose logs waterflow | \
  jq 'select(.component=="api" and .level=="error")'
```

**统计错误类型分布：**
```bash
docker-compose logs waterflow | \
  jq -r 'select(.level=="error") | .error.type' | \
  sort | uniq -c | sort -rn
```

**查询慢请求（>1 秒）：**
```bash
docker-compose logs waterflow | \
  jq 'select(.duration > 1000)'
```

**按时间统计错误数量：**
```bash
journalctl -u waterflow-server --since "1 hour ago" --no-pager | \
  grep '"level":"error"' | \
  jq -r '.timestamp' | \
  cut -d'T' -f1 | \
  uniq -c
```

---

### Agent 日志查看和分析

#### 日志查看命令

**Docker Compose 环境**
```bash
# 查看 Agent 日志
docker-compose logs agent-linux-1

# 实时跟踪
docker-compose logs -f agent-linux-1

# 最近 100 行
docker-compose logs --tail=100 agent-linux-1

# 时间范围
docker-compose logs --since 1h agent-linux-1
```

**二进制部署**
```bash
# Agent 日志输出到 stdout（JSON 格式）
# 重定向到文件
./agent > agent.log 2>&1

# 实时查看
tail -f agent.log

# Systemd 部署
journalctl -u waterflow-agent -f
journalctl -u waterflow-agent -n 100 --no-pager
```

**Kubernetes 环境**
```bash
# 查看 Agent Pod 日志
kubectl logs -l app=waterflow-agent --tail=100

# 实时跟踪
kubectl logs -f waterflow-agent-0

# 所有 Agent 实例
kubectl logs -l app=waterflow-agent --all-containers=true
```

#### 启用 Debug 日志

**环境变量方式：**
```bash
# Docker Compose
# .env 文件
WATERFLOW_LOG_LEVEL=debug

# Kubernetes
# deployment.yaml
env:
  - name: WATERFLOW_LOG_LEVEL
    value: debug

# 二进制部署
export WATERFLOW_LOG_LEVEL=debug
./agent
```

**配置文件方式：**
```yaml
# config.yaml
log:
  level: debug  # debug, info, warn, error
  format: json
```

#### 常见 Agent 错误模式

| 错误消息 | 问题类型 | 解决方法 |
|----------|----------|----------|
| `"Failed to connect to Temporal"` | Temporal 连接失败 | 检查 Temporal 地址配置、网络连通性 |
| `"Task queue not found"` | Task Queue 不匹配 | 确保工作流 `runs-on` 与 Agent `task_queues` 一致 |
| `"Plugin load failed"` | 节点插件加载失败 | 检查 `.so` 文件完整性、版本兼容性 |
| `"Activity execution failed"` | 节点执行错误 | 查看错误详情、检查参数和目标服务器状态 |
| `"Heartbeat timeout"` | Agent 心跳超时 | 检查网络连接、Temporal 服务状态 |
| `"Worker stopped"` | Worker 异常停止 | 查看完整错误堆栈、检查系统资源 |
| `"Context deadline exceeded"` | Activity 超时 | 检查 `timeout-minutes` 配置、网络延迟 |

#### 关联工作流执行

Agent 日志包含以下关联字段：

```json
{
  "timestamp": "2026-01-15T10:30:45Z",
  "level": "info",
  "workflow_id": "wf-deploy-prod-20260115",
  "run_id": "abc123...",
  "job_id": "deploy",
  "step_id": "deploy-container",
  "activity_id": "activity-123",
  "message": "Step execution started"
}
```

**过滤特定工作流日志：**
```bash
# 按 workflow_id 过滤
docker-compose logs agent-linux-1 | grep '"workflow_id":"wf-deploy-prod"'

# 按 job_id 过滤
docker-compose logs agent-linux-1 | jq 'select(.job_id=="deploy")'

# 按 step_id 过滤
docker-compose logs agent-linux-1 | jq 'select(.step_id=="deploy-container")'
```

**追踪完整执行链路：**
```bash
# 使用 run_id 追踪从提交到完成的所有日志
RUN_ID="abc123..."

# Server 日志（工作流提交）
docker-compose logs waterflow | jq "select(.run_id==\"$RUN_ID\")"

# Agent 日志（Step 执行）
docker-compose logs agent-linux-1 | jq "select(.run_id==\"$RUN_ID\")"

# Temporal UI（完整 Event History）
open "http://localhost:8088/namespaces/default/workflows/$RUN_ID"
```

---

## 工作流调试

### 工作流执行失败调试流程

当工作流执行失败时，按照以下 7 步系统化流程定位问题：

#### Step 1: 查看工作流状态

```bash
# 查询工作流详情
curl http://localhost:8080/v1/workflows/{workflow-id}

# 检查关键字段：
# - status: running, completed, failed, cancelled
# - current_job: 当前执行的 Job
# - error: 错误详情（如果失败）
```

**示例响应：**
```json
{
  "id": "wf-deploy-prod-20260115",
  "name": "deploy-production",
  "status": "failed",
  "error": {
    "type": "node_execution_error",
    "title": "Shell Command Failed",
    "detail": "exit status 1",
    "step": "deploy-container"
  }
}
```

#### Step 2: 查看工作流日志

```bash
# 获取完整日志
curl http://localhost:8080/v1/workflows/{workflow-id}/logs

# 只看错误日志
curl http://localhost:8080/v1/workflows/{workflow-id}/logs?level=error

# 过滤特定 Job
curl http://localhost:8080/v1/workflows/{workflow-id}/logs?job=deploy

# 过滤特定 Step
curl http://localhost:8080/v1/workflows/{workflow-id}/logs?step=deploy-container
```

#### Step 3: 在 Temporal UI 查看执行历史

```bash
# 打开 Temporal UI
open http://localhost:8088

# 操作步骤：
# 1. 在搜索框输入 workflow-id
# 2. 点击进入工作流详情页
# 3. 查看 Event History（完整执行历史）
# 4. 定位失败的 Activity
# 5. 查看错误消息和堆栈跟踪
# 6. 检查 Retry 状态和次数
```

**Temporal UI 关键信息：**
- **Event History**: 所有事件时间线（WorkflowStarted → ActivityScheduled → ActivityFailed）
- **Activity Input**: 节点输入参数
- **Activity Output/Error**: 节点输出或错误详情
- **Retry Attempt**: 当前重试次数
- **Stack Trace**: 完整堆栈跟踪

#### Step 4: 定位失败的 Job/Step

从日志和 Temporal UI 中提取：

```json
{
  "job_id": "deploy",
  "step_id": "deploy-container",
  "node_type": "exec/shell@v1",
  "error": {
    "type": "node_execution_error",
    "detail": "exit status 1",
    "command": "docker run nginx:latest",
    "stderr": "docker: Error response from daemon: Conflict."
  }
}
```

**分析检查清单：**
- ✅ 哪个 Job 失败？
- ✅ 哪个 Step 失败？
- ✅ 使用什么节点？
- ✅ 错误类型是什么？
- ✅ 错误详情和建议？

#### Step 5: 检查节点参数和配置

```yaml
# 检查 YAML 配置
steps:
  - id: deploy-container
    name: Deploy Container
    uses: exec/shell@v1
    with:
      command: docker
      args:
        - run
        - -d
        - --name=myapp
        - nginx:latest
    timeout-minutes: 5
    retry-strategy:
      max-attempts: 3
```

**验证检查清单：**
- ✅ `with` 参数类型和值是否正确？
- ✅ 必需参数是否完整？
- ✅ 表达式 `${{ }}` 是否求值成功？
- ✅ 环境变量是否正确传递？
- ✅ 超时配置是否合理？

**本地验证 YAML：**
```bash
waterflow validate workflow.yaml
```

#### Step 6: 检查 Agent 状态和日志

```bash
# 查询 Agent 列表
curl http://localhost:8080/v1/agents

# 检查 Agent 是否在线
# 示例响应：
[
  {
    "id": "agent-linux-1",
    "task_queues": ["web-servers"],
    "status": "healthy",
    "last_heartbeat": "2026-01-15T10:30:45Z"
  }
]

# 检查 Task Queue 匹配
# 工作流: runs-on: web-servers
# Agent: task_queues: ["web-servers"]  ✅ 匹配

# 查看 Agent 日志
docker-compose logs agent-linux-1 | grep '{workflow-id}'
```

#### Step 7: 检查权限和网络连通性

**SSH 密钥检查：**
```bash
# 测试 SSH 连接
ssh -i ~/.ssh/id_rsa user@target-host

# 复制公钥到目标服务器
ssh-copy-id user@target-host

# 验证无密码登录
ssh user@target-host "echo 'Connected successfully'"
```

**防火墙检查：**
```bash
# 检查防火墙规则
sudo iptables -L -n

# 检查端口可达性
nc -zv target-host 22
telnet target-host 22

# 临时禁用防火墙测试
sudo systemctl stop firewalld
```

**网络连通性检查：**
```bash
# Ping 测试
ping target-host

# 路由检查
traceroute target-host

# DNS 解析
nslookup target-host
```

---

### 调试技巧和最佳实践

#### 1. 本地验证 YAML

```bash
# 使用 CLI 验证 YAML 语法
waterflow validate workflow.yaml

# 检查输出：
# ✅ Validation successful
# ❌ Validation failed: syntax error at line 10
```

#### 2. 隔离失败步骤

使用 `if` 条件暂时禁用其他 Step：

```yaml
jobs:
  deploy:
    steps:
      - name: Step 1
        if: ${{ false }}  # 暂时禁用
        uses: shell@v1
      
      - name: Step 2（调试中）
        uses: shell@v1  # 只执行这个 Step
      
      - name: Step 3
        if: ${{ false }}  # 暂时禁用
        uses: shell@v1
```

#### 3. 允许失败继续

使用 `continue-on-error: true` 允许某些 Step 失败：

```yaml
steps:
  - name: Optional Cleanup
    continue-on-error: true
    uses: shell@v1
    with:
      command: rm
      args: ["-rf", "/tmp/cache"]
```

#### 4. 添加 Debug 输出

```yaml
steps:
  - name: Debug Variables
    uses: exec/shell@v1
    with:
      command: echo
      args:
        - "vars.environment=${{ vars.environment }}"
        - "matrix.server=${{ matrix.server }}"
        - "steps.build.outputs.tag=${{ steps.build.outputs.tag }}"
  
  - name: Debug Environment
    uses: exec/shell@v1
    with:
      command: env
    env:
      DEBUG: "true"
```

#### 5. 减少 Matrix 组合调试

```yaml
# 调试阶段使用少量组合
strategy:
  matrix:
    server: [web-1]  # 只测试一个
    # server: [web-1, web-2, web-3]  # 生产环境恢复
```

#### 6. 增加超时时间

临时提高 `timeout-minutes` 以排除超时问题：

```yaml
steps:
  - timeout-minutes: 60  # 临时提高到 60 分钟调试
    uses: shell@v1
```

#### 7. 查看 Temporal UI 完整历史

```bash
# Temporal UI 是最强大的调试工具
open http://localhost:8088

# 可以看到：
# - 完整 Event History（每个事件的时间戳）
# - Activity Input/Output（实际传递的参数）
# - Retry 状态（重试次数、间隔、原因）
# - Stack Trace（完整错误堆栈）
```

#### 8. 使用表达式调试技巧

```yaml
# 测试表达式求值
steps:
  - name: Test Expression
    uses: shell@v1
    with:
      command: echo
      args:
        - "${{ toJSON(vars) }}"  # 输出所有变量
        - "${{ toJSON(matrix) }}"  # 输出 Matrix 变量
```

#### 9. 逐步增量调试

1. 先运行最简单的工作流（单 Job 单 Step）
2. 确认基础功能正常
3. 逐步添加复杂功能（Matrix、条件、依赖）
4. 每次只改一个变量

#### 10. 保留失败工作流用于分析

```bash
# 不要立即删除失败的工作流
# 可以重新运行（Rerun）
curl -X POST http://localhost:8080/v1/workflows/{workflow-id}/rerun

# 在 Temporal UI 查看历史记录
# 即使工作流完成，Event History 仍可查看
```

---

---

## 诊断工具

### 健康检查端点

Waterflow Server 提供四个健康检查端点用于监控和诊断：

#### `/health` - Liveness 探针

**用途**: 检查进程是否存活（不依赖外部服务，快速响应）

```bash
curl http://localhost:8080/health
```

**响应示例：**
```json
{
  "status": "healthy"
}
```

**状态码：**
- `200 OK`: 服务运行中
- `503 Service Unavailable`: 服务不可用（进程即将退出）

**使用场景：**
- Docker `HEALTHCHECK` 指令
- Kubernetes `livenessProbe`
- 监控系统存活检查

**Docker Compose 配置示例：**
```yaml
services:
  waterflow:
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 40s
```

**Kubernetes 配置示例：**
```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 3
  failureThreshold: 3
```

---

#### `/ready` - Readiness 探针

**用途**: 检查依赖服务是否就绪（Temporal 连接正常）

```bash
curl http://localhost:8080/ready
```

**响应示例（就绪）：**
```json
{
  "status": "ready",
  "checks": {
    "temporal": {
      "status": "healthy",
      "latency_ms": 12
    },
    "database": {
      "status": "healthy"
    }
  },
  "uptime_seconds": 3600
}
```

**响应示例（未就绪）：**
```json
{
  "status": "not_ready",
  "checks": {
    "temporal": {
      "status": "unhealthy",
      "error": "connection refused"
    }
  }
}
```

**状态码：**
- `200 OK`: 所有依赖就绪，可以接受流量
- `503 Service Unavailable`: 依赖未就绪，不应接受流量

**使用场景：**
- Kubernetes `readinessProbe`（控制流量路由）
- 负载均衡器健康检查
- 部署验证（等待服务完全启动）

**Kubernetes 配置示例：**
```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 5
  timeoutSeconds: 3
  successThreshold: 1
  failureThreshold: 3
```

---

#### `/version` - 版本信息

**用途**: 获取服务版本、Git commit、构建时间

```bash
curl http://localhost:8080/version
```

**响应示例：**
```json
{
  "version": "v1.0.0",
  "commit": "abc123def456",
  "build_time": "2026-01-15T10:30:45Z",
  "go_version": "go1.21.5"
}
```

**使用场景：**
- 验证部署版本
- 回滚验证（确认版本正确）
- 故障排查（报告版本信息）

---

#### `/metrics` - Prometheus 指标

**用途**: 导出 Prometheus 格式指标

```bash
curl http://localhost:8080/metrics
```

**响应示例（部分）：**
```prometheus
# HELP waterflow_workflows_total Total number of workflows
# TYPE waterflow_workflows_total counter
waterflow_workflows_total{status="completed"} 42
waterflow_workflows_total{status="failed"} 3

# HELP waterflow_workflow_duration_seconds Workflow execution duration
# TYPE waterflow_workflow_duration_seconds histogram
waterflow_workflow_duration_seconds_bucket{le="1"} 10
waterflow_workflow_duration_seconds_bucket{le="5"} 25
waterflow_workflow_duration_seconds_sum 120.5
waterflow_workflow_duration_seconds_count 42

# HELP waterflow_api_request_duration_seconds API request latency
# TYPE waterflow_api_request_duration_seconds histogram
waterflow_api_request_duration_seconds_bucket{method="POST",path="/v1/workflows",le="0.1"} 50

# HELP go_memstats_alloc_bytes Memory allocated
# TYPE go_memstats_alloc_bytes gauge
go_memstats_alloc_bytes 12345678
```

**关键指标类别：**

1. **工作流执行指标**
   - `waterflow_workflows_total`: 工作流总数（按状态分组）
   - `waterflow_workflow_duration_seconds`: 工作流执行耗时
   - `waterflow_jobs_total`: Job 总数
   - `waterflow_steps_total`: Step 总数

2. **API 请求指标**
   - `waterflow_api_request_duration_seconds`: API 请求延迟
   - `waterflow_api_requests_total`: API 请求总数
   - `waterflow_api_errors_total`: API 错误总数

3. **系统资源指标**
   - `go_memstats_alloc_bytes`: 内存分配
   - `go_goroutines`: Goroutine 数量
   - `process_cpu_seconds_total`: CPU 使用时间

**Prometheus 配置示例：**
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'waterflow'
    static_configs:
      - targets: ['waterflow:8080']
    metrics_path: /metrics
    scrape_interval: 15s
```

**Grafana Dashboard 查询示例：**
```promql
# 工作流成功率
rate(waterflow_workflows_total{status="completed"}[5m]) / 
rate(waterflow_workflows_total[5m]) * 100

# API P95 延迟
histogram_quantile(0.95, 
  rate(waterflow_api_request_duration_seconds_bucket[5m]))

# 内存使用趋势
go_memstats_alloc_bytes / 1024 / 1024
```

---

### 日志查看命令

#### Docker Compose 环境
```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务最近 100 行日志
docker-compose logs --tail=100 waterflow

# 查看特定时间段日志
docker-compose logs --since 2026-01-09T10:00:00 waterflow
```

**Systemd 环境**
```bash
# 实时查看日志
sudo journalctl -u waterflow-server -f

# 查看最近 100 行日志
sudo journalctl -u waterflow-server -n 100 --no-pager

# 查看特定时间段日志
sudo journalctl -u waterflow-server --since "2026-01-09 10:00:00"
```

**Kubernetes 环境**
```bash
# 查看 Pod 日志
kubectl logs -l app=waterflow-server

# 实时跟踪
kubectl logs -f waterflow-server-0

# 最近 100 行
kubectl logs --tail=100 waterflow-server-0

# 所有副本日志
kubectl logs -l app=waterflow-server --all-containers=true
```

---

### 健康检查

**基础健康检查：**

**基础健康检查：**
```bash
# Liveness 探针（进程存活）
curl http://localhost:8080/health

# Readiness 探针（依赖服务状态）
curl http://localhost:8080/ready

# 版本信息
curl http://localhost:8080/version

# Prometheus 指标
curl http://localhost:8080/metrics
```

**详细说明见**: [健康检查端点](#健康检查端点)

---

### 资源监控

**Docker 环境**
```bash
# 实时资源使用
docker stats

# 特定容器资源
docker stats waterflow-server

# 容器详细信息
docker inspect waterflow-server
```

**系统资源**
```bash
# CPU 和内存
top
htop  # 如果已安装

# 内存详情
free -h

# 磁盘使用
df -h

# 磁盘 I/O
iostat -x 1
```

**Kubernetes 环境**
```bash
# Pod 资源使用
kubectl top pod waterflow-server-0

# 所有 Pod 资源
kubectl top pods -l app=waterflow

# 节点资源
kubectl top nodes
```

---

### 网络诊断

```bash
# 检查端口监听
netstat -tuln | grep -E '8080|7233|5432'

# 测试端口连通性
nc -zv localhost 8080

# 检查 DNS 解析（Docker 环境）
docker-compose exec waterflow nslookup temporal

# 检查路由
docker-compose exec waterflow traceroute temporal

# Kubernetes 环境
kubectl exec waterflow-server-0 -- nc -zv temporal 7233
kubectl exec waterflow-server-0 -- nslookup temporal
```

---

## 诊断命令速查表

快速诊断问题的常用命令清单，按类别分组。

### 服务状态检查

| 命令 | 说明 | 环境 |
|------|------|------|
| `docker-compose ps` | 检查容器状态 | Docker Compose |
| `systemctl status waterflow-server` | 检查 Systemd 服务状态 | Systemd |
| `kubectl get pods -l app=waterflow` | 检查 Kubernetes Pods 状态 | Kubernetes |
| `curl http://localhost:8080/health` | Liveness 探针（进程存活） | 所有 |
| `curl http://localhost:8080/ready` | Readiness 探针（依赖服务） | 所有 |
| `curl http://localhost:8080/version` | 版本信息 | 所有 |

**典型输出：**
```bash
$ docker-compose ps
NAME                    STATUS      PORTS
waterflow               Up 2 hours  0.0.0.0:8080->8080/tcp
temporal                Up 2 hours  0.0.0.0:7233->7233/tcp
postgresql              Up 2 hours  5432/tcp
agent-linux-1           Up 2 hours

$ curl http://localhost:8080/health
{"status":"healthy"}
```

---

### 日志查看

| 命令 | 说明 | 环境 |
|------|------|------|
| `docker-compose logs -f waterflow` | 实时日志（Server） | Docker Compose |
| `docker-compose logs -f agent-linux-1` | 实时日志（Agent） | Docker Compose |
| `journalctl -u waterflow-server -n 100` | 最近 100 行（Server） | Systemd |
| `kubectl logs -f waterflow-server-0` | 实时日志（Kubernetes） | Kubernetes |
| `grep '"level":"error"'` | 过滤错误日志 | 所有 |
| `jq 'select(.level=="error")'` | JSON 过滤错误 | 所有 |

**典型输出：**
```json
{
  "timestamp": "2026-01-15T10:30:45Z",
  "level": "error",
  "workflow_id": "wf-deploy-prod",
  "component": "executor",
  "message": "Step execution failed",
  "error": {
    "type": "node_execution_error",
    "detail": "exit status 1"
  }
}
```

---

### 网络诊断

| 命令 | 说明 | 典型输出 |
|------|------|----------|
| `netstat -tuln \| grep 8080` | 检查端口监听 | `tcp 0.0.0.0:8080 LISTEN` |
| `nc -zv localhost 7233` | Temporal 端口连通性 | `Connection to localhost 7233 port [tcp/*] succeeded!` |
| `nslookup temporal` | DNS 解析（Docker） | `Name: temporal / Address: 172.18.0.3` |
| `traceroute temporal` | 路由检查 | `1  temporal (172.18.0.3)  0.123 ms` |
| `curl -v http://localhost:8080/health` | HTTP 连通性详情 | `< HTTP/1.1 200 OK` |

**使用示例：**
```bash
# 检查 Waterflow 是否监听端口
$ netstat -tuln | grep 8080
tcp6       0      0 :::8080                 :::*                    LISTEN

# 测试 Temporal 连接
$ nc -zv localhost 7233
Connection to localhost 7233 port [tcp/*] succeeded!
```

---

### 资源监控

| 命令 | 说明 | 关键指标 |
|------|------|----------|
| `docker stats` | 容器资源使用（实时） | CPU%, MEM%, NET I/O |
| `docker stats waterflow-server` | 特定容器资源 | CPU%, MEM USAGE/LIMIT |
| `top` 或 `htop` | 系统资源使用 | %CPU, %MEM, PID |
| `free -h` | 内存使用 | total, used, free, available |
| `df -h` | 磁盘使用 | Size, Used, Avail, Use% |
| `iostat -x 1` | 磁盘 I/O | r/s, w/s, %util |
| `kubectl top pod` | Pod 资源（Kubernetes） | CPU(cores), MEMORY(bytes) |

**典型输出：**
```bash
$ docker stats waterflow-server --no-stream
CONTAINER         CPU %   MEM USAGE / LIMIT   NET I/O         BLOCK I/O
waterflow-server  2.5%    128MiB / 2GiB      1.2MB / 856kB   12MB / 8MB

$ free -h
              total        used        free      shared  buff/cache   available
Mem:           15Gi        8.2Gi       2.3Gi       234Mi        5.1Gi        6.8Gi
Swap:          2.0Gi          0B       2.0Gi
```

---

### 工作流调试

| 命令 | 说明 | 用途 |
|------|------|------|
| `waterflow validate workflow.yaml` | 本地 YAML 验证 | 提交前语法检查 |
| `curl GET /v1/workflows/{id}` | 查询工作流状态 | 查看执行状态和错误 |
| `curl GET /v1/workflows/{id}/logs` | 查询工作流日志 | 查看详细日志 |
| `curl GET /v1/workflows/{id}/logs?level=error` | 只看错误日志 | 快速定位错误 |
| `curl POST /v1/workflows/{id}/cancel` | 取消工作流 | 停止执行中的工作流 |
| `curl GET /v1/nodes` | 查看可用节点 | 验证节点存在性 |
| `open http://localhost:8088` | 打开 Temporal UI | 查看完整执行历史 |

**使用示例：**
```bash
# 验证 YAML
$ waterflow validate deploy.yaml
✅ Validation successful

# 查询工作流状态
$ curl http://localhost:8080/v1/workflows/wf-deploy-prod | jq .
{
  "id": "wf-deploy-prod",
  "status": "failed",
  "error": {
    "type": "node_execution_error",
    "detail": "exit status 1"
  }
}

# 只看错误日志
$ curl "http://localhost:8080/v1/workflows/wf-deploy-prod/logs?level=error"
```

---

### Temporal 诊断

| 命令 | 说明 | 用途 |
|------|------|------|
| `docker-compose logs temporal` | Temporal 日志 | 诊断 Temporal 服务问题 |
| `docker-compose ps temporal` | Temporal 容器状态 | 检查服务是否运行 |
| `curl http://localhost:8088/api/v1/namespaces` | Temporal API 测试 | 验证 API 可用性 |
| `open http://localhost:8088` | Temporal UI | 查看工作流执行历史 |
| `kubectl get pods -l app=temporal` | Temporal Pods（Kubernetes） | K8s 环境状态检查 |

**Temporal UI 使用：**
1. 打开 http://localhost:8088
2. 选择 Namespace: `default`
3. 搜索 Workflow ID
4. 查看 Event History（完整执行时间线）
5. 检查 Activity 失败详情

---

### 数据库诊断

| 命令 | 说明 | 典型输出 |
|------|------|----------|
| `docker-compose ps postgresql` | PostgreSQL 状态 | `Up 2 hours` |
| `docker-compose logs postgresql` | PostgreSQL 日志 | 启动和错误日志 |
| `docker-compose exec postgresql psql -U temporal -c "SELECT 1"` | 连接测试 | `?column? 1` |
| `docker volume ls \| grep postgres` | 数据卷列表 | `deployments_postgresql-data` |

**使用示例：**
```bash
# 测试数据库连接
$ docker-compose exec postgresql psql -U temporal -c "SELECT 1"
 ?column? 
----------
        1
(1 row)

# 检查数据库大小
$ docker-compose exec postgresql psql -U temporal -c "\l+"
```

---

### 常见问题快速诊断流程

#### 问题：服务无法启动
```bash
1. docker-compose ps                      # 检查容器状态
2. docker-compose logs waterflow          # 查看错误日志
3. netstat -tuln | grep 8080              # 检查端口占用
4. curl http://localhost:8080/health      # 测试健康检查
```

#### 问题：Temporal 连接失败
```bash
1. docker-compose ps temporal             # Temporal 是否运行
2. nc -zv localhost 7233                  # 端口连通性
3. curl http://localhost:8080/ready       # 检查 Readiness
4. docker-compose logs temporal           # 查看 Temporal 日志
```

#### 问题：工作流执行失败
```bash
1. curl /v1/workflows/{id}                # 查看工作流状态
2. curl /v1/workflows/{id}/logs?level=error  # 查看错误日志
3. open http://localhost:8088             # Temporal UI 查看详情
4. docker-compose logs agent-linux-1      # 查看 Agent 日志
```

#### 问题：性能问题
```bash
1. docker stats                           # 容器资源使用
2. top                                    # 系统资源使用
3. curl /metrics                          # Prometheus 指标
4. docker-compose logs | grep timeout     # 查找超时日志
```

---

## 获取支持

## 获取支持

### 自助资源

- **完整文档**: 
  - [部署指南](deployment.md)
  - [配置参考](configuration.md)
  - [快速开始](quick-start.md)
  - [YAML DSL 语法参考](yaml-dsl-reference.md)
  - [API 文档](http://localhost:8080/docs)
  
- **GitHub**:
  - Issues: https://github.com/websoft9/waterflow/issues
  - Discussions: https://github.com/websoft9/waterflow/discussions
  - 示例配置: `examples/configs/`
  - 工作流模板: `examples/workflows/`

- **相关 Story 参考**:
  - [Story 7.1 - 类型化错误处理](sprint-artifacts/7-1-typed-error-handling.md)
  - [Story 7.2 - 结构化日志系统](sprint-artifacts/7-2-structured-logging.md)
  - [Story 8.5 - 部署文档](sprint-artifacts/8-5-deployment-documentation.md)

### 提交 Issue 时请提供

为了快速解决问题，请在 GitHub Issue 中提供以下信息：

**1. 环境信息**
```bash
# Waterflow 版本
curl http://localhost:8080/version

# Docker 版本
docker --version
docker-compose --version

# Kubernetes 版本（如适用）
kubectl version --short

# 操作系统
uname -a
```

**2. 完整错误日志**
```bash
# Server 日志（最近 200 行）
docker-compose logs --tail=200 waterflow > server.log

# Agent 日志（最近 200 行）
docker-compose logs --tail=200 agent-linux-1 > agent.log

# Temporal 日志（如果相关）
docker-compose logs --tail=200 temporal > temporal.log
```

**3. 配置文件**（移除敏感信息）
```bash
# 环境变量配置
cat deployments/.env | grep -v -E 'PASSWORD|SECRET|KEY'

# 工作流 YAML
cat workflow.yaml
```

**4. 复现步骤**
```markdown
### 复现步骤
1. 执行命令: `docker-compose up -d`
2. 提交工作流: `curl -X POST /v1/workflows -d @workflow.json`
3. 观察到错误: 工作流状态变为 failed

### 预期行为
工作流应该成功执行

### 实际行为
工作流在 Step "deploy" 失败，错误消息: "exit status 1"

### 相关日志
```json
{
  "level": "error",
  "message": "Step execution failed",
  "error": {...}
}
```
```

**5. 已尝试的解决方法**
```markdown
- ✅ 已检查 Temporal 连接状态
- ✅ 已验证 YAML 语法
- ❌ 问题仍然存在
```

### 紧急支持

- **安全漏洞**: security@websoft9.com
- **生产故障**: 参考企业支持合同
- **社区支持**: GitHub Discussions

### 贡献文档改进

如果您在使用过程中发现文档可以改进的地方：

1. Fork 仓库: https://github.com/websoft9/waterflow
2. 编辑文档: `docs/troubleshooting.md`
3. 提交 Pull Request

---

**文档版本**: 1.1.0  
**更新日期**: 2026-01-15  
**维护**: Websoft9 Team
