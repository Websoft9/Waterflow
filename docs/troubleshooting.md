# Waterflow 故障排查手册

本文档提供 Waterflow 常见问题的诊断和解决方法。

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

# 等待 Temporal 完全启动（约 30 秒）
docker-compose logs -f temporal
```

**原因 2: 配置地址错误**
```bash
# Docker Compose 环境应使用服务名
WATERFLOW_TEMPORAL_HOST=temporal:7233  # 正确
# WATERFLOW_TEMPORAL_HOST=localhost:7233  # 错误（容器内无法访问）
```

**原因 3: 网络问题**
```bash
# 检查 Docker 网络
docker network ls
docker network inspect waterflow-network

# 重建网络
docker-compose down
docker-compose up -d
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

## 诊断工具

### 日志查看

**Docker Compose 环境**
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

### 健康检查

```bash
# Liveness 探针（进程存活）
curl http://localhost:8080/health

# Readiness 探针（依赖服务状态）
curl http://localhost:8080/ready

# 版本信息
curl http://localhost:8080/version
```

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
```

---

## 获取支持

### 自助资源

- **文档**: [deployment.md](deployment.md), [configuration.md](configuration.md)
- **GitHub Issues**: https://github.com/websoft9/waterflow/issues
- **示例配置**: `examples/configs/`

### 提交 Issue 时请提供

1. **环境信息**
   - Waterflow 版本 (`/version` 输出)
   - Docker/Kubernetes 版本
   - 操作系统版本

2. **错误日志**
   ```bash
   docker-compose logs --tail=100 waterflow
   ```

3. **配置文件**（移除敏感信息）
   ```bash
   cat deployments/.env | grep -v PASSWORD
   ```

4. **复现步骤**
   - 详细的操作步骤
   - 预期行为 vs 实际行为

### 紧急支持

- **安全漏洞**: security@websoft9.com
- **生产故障**: 参考企业支持合同
