# Waterflow 快速开始指南

最快 10 分钟部署完整的 Waterflow + Temporal 工作流编排环境。

## ⚡ 一键部署

### 前置要求

**硬件要求:**

| 资源 | 最低要求 | 推荐配置 |
|------|---------|----------|
| CPU | 1 核 | 2 核+ |
| 内存 | 2GB | 4GB+ |
| 磁盘 | 20GB | 50GB+ |

**软件依赖:**

| 软件 | 最低版本 | 验证命令 |
|------|---------|----------|
| Docker Engine | 20.10+ | `docker --version` |
| Docker Compose | 2.0+ | `docker compose version` |

**环境检查:**

```bash
# 验证 Docker 安装和版本
docker --version
# 预期输出: Docker version 20.10.x 或更高

docker compose version
# 预期输出: Docker Compose version v2.x.x 或更高

# 检查可用内存（Linux/macOS）
free -h | grep Mem
# 确保至少有 2GB 可用内存
```

### 快速启动

```bash
# 克隆仓库
git clone https://github.com/websoft9/waterflow.git
cd waterflow

# 启动所有服务
cd deployments
docker compose up -d
```

等待 2-3 分钟，所有服务启动完成。

### 验证部署

```bash
# 检查服务状态
docker compose ps
```

**预期输出** - 应看到5个运行中的容器：

| 服务 | 状态 | 端口 |
|------|------|------|
| waterflow-server | Up | 8080 |
| waterflow-agent | Up | - |
| temporal | Up | 7233, 8233 |
| temporal-ui | Up | 8088 |
| postgresql | Up | 5432 |

```bash
# 验证 Waterflow 健康检查
curl http://localhost:8080/health
```

**预期输出:**
```json
{"status":"ok"}
```

```bash
# 验证 Temporal 连接
curl http://localhost:8080/ready
```

**预期输出:**
```json
{"status":"ready","checks":{"temporal":"ok"}}
```

## 🎯 提交第一个工作流

### 使用示例工作流

```bash
# 提交 Hello World 示例
curl -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/json" \
  -d "{\"yaml\": \"$(cat ../examples/hello-world.yaml | sed 's/"/\\"/g' | tr '\n' ' ')\"}"
```

### 查看执行结果

**方法 1: API 查询**
```bash
# 列出所有工作流
curl http://localhost:8080/v1/workflows

# 查询特定工作流（替换 {id} 为实际 ID）
curl http://localhost:8080/v1/workflows/{id}
```

**预期响应示例:**
```json
{
  "id": "wf_abc123def456",
  "status": "completed",
  "created_at": "2026-01-15T10:30:00Z",
  "completed_at": "2026-01-15T10:30:05Z",
  "jobs": [
    {
      "name": "hello",
      "status": "completed",
      "steps": [
        {"name": "greet", "status": "completed", "outputs": {"message": "Hello, World!"}}
      ]
    }
  ]
}
```

**查看工作流日志:**
```bash
# 查询工作流日志（替换 {id} 为实际 ID）
curl http://localhost:8080/v1/workflows/{id}/logs

# 或使用命令行工具
./scripts/logs.sh waterflow
```

**方法 2: Temporal UI**

访问 http://localhost:8088 查看可视化执行流程。

## 📦 服务端口

| 服务 | 端口 | 说明 |
|------|------|------|
| Waterflow API | 8080 | REST API 服务 |
| Swagger UI | 8080/docs | **API 交互式文档** ⭐ |
| Temporal gRPC | 7233 | Temporal 客户端连接 |
| Temporal HTTP | 8233 | Temporal HTTP API |
| Temporal UI | 8088 | Web 控制台 |
| PostgreSQL | 5432 | 数据库（内部） |

**访问 API 文档:** http://localhost:8080/docs - 查看完整的 REST API 规范并在线测试

## 🛠️ 常用命令

```bash
# 查看日志
./scripts/logs.sh waterflow
./scripts/logs.sh temporal

# 重启服务
cd deployments
docker compose restart waterflow

# 停止服务
docker compose down

# 清理环境（删除所有数据）
./scripts/cleanup.sh
```

## 🆘 常见问题排查

### ❌ Docker 未安装或版本过低

**现象:** 执行 `docker --version` 失败或版本低于 20.10

**解决方案:**
```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# macOS
brew install docker docker-compose

# 验证安装
docker --version
docker compose version
```

### ❌ 端口冲突 (8080/8088 已被占用)

**现象:** `docker compose up` 报错 "bind: address already in use"

**解决方案:**
```bash
# 方法 1: 查找占用端口的进程并停止
sudo lsof -i :8080
sudo kill -9 <PID>

# 方法 2: 修改 Waterflow 端口
cd deployments
echo "WATERFLOW_SERVER_PORT=9090" >> .env
echo "TEMPORAL_UI_PORT=9088" >> .env
docker compose up -d
```

### ❌ 内存不足 (Temporal 启动失败)

**现象:** Temporal 容器反复重启，日志显示 "OOMKilled"

**解决方案:**
```bash
# 检查可用内存
free -h

# 增加 Docker 内存限制（Docker Desktop）
# Settings → Resources → Memory → 设置为 4GB+

# 或使用轻量级配置
cd deployments
docker compose -f docker-compose-minimal.yaml up -d
```

### ❌ 网络连接问题 (无法访问 Docker Hub)

**现象:** 镜像拉取失败 "error pulling image"

**解决方案:**
```bash
# 配置 Docker 镜像加速（中国大陆用户）
sudo mkdir -p /etc/docker
sudo tee /etc/docker/daemon.json <<-'EOF'
{
  "registry-mirrors": ["https://mirror.gcr.io"]
}
EOF
sudo systemctl daemon-reload
sudo systemctl restart docker
```

### ❌ 工作流提交失败 (YAML 语法错误)

**现象:** API 返回 400 错误 "invalid YAML syntax"

**解决方案:**
```bash
# 先验证 YAML 语法
curl -X POST http://localhost:8080/v1/workflows/validate \
  -H "Content-Type: application/json" \
  -d "{\"yaml\": \"$(cat your-workflow.yaml)\"}" | jq

# 查看详细错误信息
# 常见问题: 缩进错误、冒号后缺少空格、引号不匹配
```

**更多问题?** 查看 **[完整故障排查文档](deployment.md#常见问题排查)**

## 🔧 自定义配置

Waterflow 支持灵活的配置方式:
- **环境变量** - 推荐用于容器部署 (Docker/Kubernetes)
- **配置文件** - 推荐用于本地开发和复杂配置
- **命令行参数** - 用于临时覆盖特定值

**配置优先级**: 命令行参数 > 环境变量 > 配置文件 > 默认值

### 修改端口 (环境变量方式)

编辑 `deployments/.env`:
```bash
WATERFLOW_SERVER_PORT=9090
TEMPORAL_UI_PORT=9088
```

重启服务:
```bash
cd deployments
docker compose up -d
```

### 修改日志级别

```bash
cd deployments
echo "WATERFLOW_LOG_LEVEL=debug" >> .env
docker compose up -d
```

### 完整配置说明

详细的配置选项和验证规则请参考 **[配置参考文档](configuration.md)**,包括:
- 所有配置项的环境变量映射
- 配置验证规则和错误信息
- 不同环境的配置示例 (开发/生产/最小配置)

## 🎨 使用工作流模板

Waterflow 提供生产就绪的工作流模板,快速实现常见部署和运维场景:

### 浏览模板库

访问 **[工作流模板库](./templates/README.md)** 查看所有可用模板。

### 可用模板

| 模板 | 说明 | 适用场景 |
|------|------|----------|
| **[单服务器部署](./templates/single-server-deployment.md)** | 将应用部署到单台服务器 | Web 应用、MVP 产品 |
| **[多服务器健康检查](./templates/multi-server-health-check.md)** | 并行检查多台服务器状态 | 服务器巡检、监控 |
| **[分布式栈部署](./templates/distributed-stack-deployment.md)** | 部署多层应用栈(数据库+应用) | 生产环境、微服务 |

### 快速使用模板

```bash
# 1. 选择模板
cp examples/workflows/single-server-deployment.yaml my-app.yaml

# 2. 修改配置
vim my-app.yaml
# 编辑 vars.repo_url 和 vars.app_name

# 3. 提交工作流
curl -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/json" \
  -d "{\"yaml\": \"$(cat my-app.yaml)\"}"
```

**完整模板文档:** [工作流模板库](./templates/README.md)

## ⏰ 创建定时调度

### 基础调度示例

为工作流创建定时任务，实现自动化执行：

```bash
# 1. 创建工作流定义
curl -X POST http://localhost:8080/v1/workflows/definitions \
  -H "Content-Type: application/json" \
  -d '{
    "name": "daily-backup",
    "content": "name: daily-backup\nvars:\n  backup_dir: /tmp\njobs:\n  backup:\n    runs-on: default\n    steps:\n      - name: Run Backup\n        uses: exec@v1\n        with:\n          command: echo \"Backup to ${backup_dir}\""
  }'

# 2. 创建每日调度（凌晨 2 点执行）
curl -X POST http://localhost:8080/v1/workflows/daily-backup/schedules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "nightly-backup",
    "cron": "0 2 * * *",
    "timezone": "Asia/Shanghai",
    "vars": {
      "backup_dir": "/data/backups"
    },
    "memo": "每日凌晨 2 点自动备份"
  }'

# 3. 查看调度详情
curl http://localhost:8080/v1/workflows/daily-backup/schedules/nightly-backup
```

### 常用 Cron 表达式

| 表达式 | 说明 |
|--------|------|
| `0 * * * *` | 每小时整点 |
| `0 2 * * *` | 每天凌晨 2 点 |
| `0 9 * * 1` | 每周一上午 9 点 |
| `0 0 1 * *` | 每月 1 号零点 |
| `*/15 * * * *` | 每 15 分钟 |

### 参数覆盖机制

调度支持三层参数合并（优先级从低到高）：

```yaml
# 1. 工作流 YAML 定义的默认参数
name: backup-workflow
vars:
  backup_dir: /tmp          # 默认值
  retention_days: 7         # 默认值
```

```bash
# 2. 调度创建时绑定的参数（覆盖 YAML vars）
curl -X POST http://localhost:8080/v1/workflows/backup-workflow/schedules \
  -d '{
    "name": "prod-backup",
    "cron": "0 2 * * *",
    "vars": {
      "backup_dir": "/data/backups",  # 覆盖默认值
      "retention_days": 30             # 覆盖默认值
    }
  }'

# 3. 手动触发时提供的参数（最高优先级）
curl -X POST http://localhost:8080/v1/workflows/backup-workflow/schedules/prod-backup/trigger \
  -d '{
    "vars": {
      "backup_dir": "/mnt/emergency"  # 临时覆盖
    }
  }'
```

### 调度管理操作

```bash
# 暂停调度（系统维护）
curl -X POST http://localhost:8080/v1/workflows/daily-backup/schedules/nightly-backup/pause \
  -d '{"reason": "系统维护中"}'

# 恢复调度
curl -X POST http://localhost:8080/v1/workflows/daily-backup/schedules/nightly-backup/resume

# 更新调度（修改 Cron 表达式）
curl -X PUT http://localhost:8080/v1/workflows/daily-backup/schedules/nightly-backup \
  -d '{
    "cron": "0 3 * * *",
    "memo": "改为凌晨 3 点执行"
  }'

# 删除调度
curl -X DELETE http://localhost:8080/v1/workflows/daily-backup/schedules/nightly-backup
```

### 多调度场景

一个工作流可以创建多个调度：

```bash
# 工作日备份（周一到周五）
curl -X POST http://localhost:8080/v1/workflows/backup-workflow/schedules \
  -d '{
    "name": "weekday-backup",
    "cron": "0 2 * * 1-5",
    "vars": {"env": "production"}
  }'

# 周末备份（周六、周日）
curl -X POST http://localhost:8080/v1/workflows/backup-workflow/schedules \
  -d '{
    "name": "weekend-backup",
    "cron": "0 4 * * 0,6",
    "vars": {"env": "staging"}
  }'

# 列出所有调度
curl http://localhost:8080/v1/workflows/backup-workflow/schedules
```

**详细文档:** [Schedule API 参考](api-guide.md#schedule-api)

## 📚 下一步

- 📖 [完整部署文档](deployment.md) - 详细配置和故障排查
- 🔒 **[安全配置指南](guides/security-guide.md) - 生产环境安全最佳实践** ⭐
- 🎨 **[工作流模板库](./templates/README.md) - 生产就绪的模板** ⭐
- 🔍 [示例工作流](../examples/README.md) - 更多 YAML 示例
- 🏗️ [架构文档](architecture.md) - 系统架构设计
- 💻 [开发指南](development.md) - 本地开发环境

## 🔒 生产环境部署

**⚠️ 警告:** 上述快速启动配置仅用于开发和测试环境。生产环境部署请务必参考:

1. **[安全配置指南](guides/security-guide.md)** - HTTPS/TLS、认证授权、密钥管理、审计日志
2. **[安全检查清单](guides/security-checklist.md)** - 部署前安全验证
3. **[合规指南](guides/compliance-guide.md)** - SOC2/ISO27001/GDPR 合规要求

**快速安全加固:**
```bash
# 1. 生成 TLS 证书
./scripts/generate-tls-cert.sh production yourdomain.com

# 2. 运行安全检查
./scripts/security-check.sh

# 3. 使用安全配置启动
cp examples/configs/secure-docker-compose.yaml deployments/docker-compose.yaml
docker compose up -d
```

## 🆘 遇到问题？

查看 [部署文档 - 常见问题排查](deployment.md#常见问题排查) 章节。
