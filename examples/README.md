# Waterflow 示例

本目录包含 Waterflow 的示例工作流和配置文件。

## 🎨 工作流模板库 (推荐)

**访问完整模板库:** [工作流模板库文档](../docs/templates/README.md)

Waterflow 提供生产就绪的工作流模板,包含详细文档、使用示例和故障排查指南:

| 模板 | 说明 | 完整文档 |
|------|------|----------|
| **单服务器部署** | 将应用部署到单台服务器,含构建、健康检查、回滚 | [查看文档](../docs/templates/single-server-deployment.md) |
| **多服务器健康检查** | 并行检查多台服务器 CPU、内存、磁盘状态 | [查看文档](../docs/templates/multi-server-health-check.md) |
| **分布式栈部署** | 部署多层应用栈,支持服务依赖管理 | [查看文档](../docs/templates/distributed-stack-deployment.md) |

**快速使用模板:**

```bash
# 1. 浏览可用模板
ls examples/workflows/*.yaml

# 2. 复制模板
cp examples/workflows/single-server-deployment.yaml my-deployment.yaml

# 3. 修改配置
vim my-deployment.yaml  # 编辑 vars 部分

# 4. 提交工作流
waterflow submit my-deployment.yaml
```

**完整模板文档,包含:**
- ✅ 详细的参数说明和配置选项
- ✅ 至少 2 个真实使用示例
- ✅ 定制指南和扩展方法
- ✅ 常见问题故障排查
- ✅ 最佳实践建议

📖 **[访问工作流模板库 →](../docs/templates/README.md)**

---

## 📁 目录结构

```
examples/
├── configs/                    # 配置文件模板
│   ├── config.example.yaml          # Server 配置模板
│   ├── config.agent.example.yaml    # Agent 配置模板
│   └── server-groups.example.yaml   # Server Groups 配置模板
├── workflows/                  # 工作流模板
│   ├── templates-metadata.json        # 模板元数据 (API 数据源)
│   ├── single-server-deployment.yaml  # 单服务器部署模板 (生产级)
│   ├── multi-server-health-check.yaml # 多服务器健康检查模板
│   ├── distributed-stack-deployment.yaml # 分布式栈部署模板
│   ├── shell-examples.yaml            # Shell 节点示例
│   ├── script-examples.yaml           # Script 节点示例
│   ├── docker-exec-examples.yaml      # Docker 节点示例
│   └── ...                            # 其他节点示例
├── hello-world.yaml            # 基础工作流示例
├── multi-step.yaml
├── matrix.yaml
├── multi-server.yaml
├── providers/                  # Provider 插件示例
└── README.md
```

## 🔧 配置文件模板

### Server 配置

```bash
# 复制模板创建配置文件
cp examples/configs/config.example.yaml config.yaml

# 编辑配置
vim config.yaml
```

详细配置说明参见：[配置指南](../docs/configuration.md)

### Agent 配置

```bash
# 复制模板
cp examples/configs/config.agent.example.yaml config.agent.yaml

# 编辑配置
vim config.agent.yaml
```

详细说明参见：[docs/guides/agent-README.md](../docs/guides/agent-README.md)

## 📋 工作流示例

### 1. hello-world.yaml - 基础示例
最简单的 Waterflow 工作流，演示：
- 基本 YAML 语法
- 变量使用 (`vars`)
- 表达式插值 (`${{ }}`)
- 工作流手动触发 (`workflow_dispatch`)

**运行示例：**
```bash
curl -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/json" \
  -d @- <<EOF
{
  "yaml": "$(cat examples/hello-world.yaml | sed 's/"/\\"/g' | tr '\n' ' ')"
}
EOF
```

### 2. multi-step.yaml - 多步骤工作流
演示多个步骤和任务依赖：
- 多个 jobs
- 任务依赖 (`needs`)
- 多步骤执行
- 顺序编排

### 3. matrix.yaml - 矩阵并行执行
演示矩阵策略并行执行：
- Matrix 策略 (`strategy.matrix`)
- 并行任务执行
- 矩阵变量引用

## 📚 基础示例库

### workflows/ 目录 - 节点功能示例

基础示例展示每个核心节点的用法和常见场景。适合新用户学习 Waterflow DSL 语法和节点特性。

#### 可用基础示例

| 示例文件 | 节点类型 | 学习难度 | 说明 |
|---------|---------|---------|------|
| [shell-examples.yaml](workflows/shell-examples.yaml) | `exec/shell` | 入门 | Shell 命令执行、环境变量、工作目录配置 |
| [script-examples.yaml](workflows/script-examples.yaml) | `exec/script` | 入门 | Bash/Python 内联脚本、参数传递 |
| [file-transfer-examples.yaml](workflows/file-transfer-examples.yaml) | `file/transfer` | 入门 | SCP 文件上传/下载、SSH 认证 |
| [sleep-examples.yaml](workflows/sleep-examples.yaml) | `flow/sleep` | 入门 | 延迟等待、服务就绪检查、重试间隔 |
| [docker-exec-examples.yaml](workflows/docker-exec-examples.yaml) | `docker/exec` | 中级 | Docker 拉取、运行、exec、构建、清理 |
| [docker-compose-examples.yaml](workflows/docker-compose-examples.yaml) | `docker/compose` | 中级 | Docker Compose 多容器编排 |
| [database-backup.yaml](workflows/database-backup.yaml) | 多节点 | 中级 | PostgreSQL/MySQL 自动备份、条件执行 |
| [ci-cd-pipeline.yaml](workflows/ci-cd-pipeline.yaml) | 多节点 | 高级 | 完整 CI/CD 流水线、Job 依赖、条件部署 |
| [webhook-notification.yaml](workflows/webhook-notification.yaml) | `http/request` | 入门 | Slack/钉钉/企业微信通知、HTTP POST |

#### 学习路径建议

**🌱 入门级 (< 30 分钟)**  
学习基本节点和 DSL 语法：
1. [shell-examples.yaml](workflows/shell-examples.yaml) - 命令执行基础
2. [script-examples.yaml](workflows/script-examples.yaml) - 脚本编写
3. [sleep-examples.yaml](workflows/sleep-examples.yaml) - 流程控制
4. [webhook-notification.yaml](workflows/webhook-notification.yaml) - HTTP 请求

**🌿 中级 (1-2 小时)**  
学习文件操作和容器编排：
5. [file-transfer-examples.yaml](workflows/file-transfer-examples.yaml) - 文件传输
6. [docker-exec-examples.yaml](workflows/docker-exec-examples.yaml) - Docker 操作
7. [docker-compose-examples.yaml](workflows/docker-compose-examples.yaml) - 容器编排
8. [database-backup.yaml](workflows/database-backup.yaml) - 数据备份场景

**🌳 高级 (2-3 小时)**  
学习复杂编排和生产级模板：
9. [ci-cd-pipeline.yaml](workflows/ci-cd-pipeline.yaml) - CI/CD 集成
10. [single-server-deployment.yaml](workflows/single-server-deployment.yaml) - 单服务器部署
11. [multi-server-health-check.yaml](workflows/multi-server-health-check.yaml) - 多服务器监控
12. [distributed-stack-deployment.yaml](workflows/distributed-stack-deployment.yaml) - 分布式部署

#### 按场景分类索引

**部署场景:**
- [single-server-deployment.yaml](workflows/single-server-deployment.yaml) - 单服务器应用部署
- [distributed-stack-deployment.yaml](workflows/distributed-stack-deployment.yaml) - 多层应用栈部署
- [ci-cd-pipeline.yaml](workflows/ci-cd-pipeline.yaml) - CI/CD 自动化部署

**监控场景:**
- [multi-server-health-check.yaml](workflows/multi-server-health-check.yaml) - 服务器健康检查
- [sleep-examples.yaml](workflows/sleep-examples.yaml) - 服务就绪等待

**备份场景:**
- [database-backup.yaml](workflows/database-backup.yaml) - 数据库自动备份

**测试场景:**
- [ci-cd-pipeline.yaml](workflows/ci-cd-pipeline.yaml) - 自动化测试流水线

**通知场景:**
- [webhook-notification.yaml](workflows/webhook-notification.yaml) - 多平台消息推送

#### 快速体验

```bash
# 1. 运行 Shell 命令示例
waterflow submit examples/workflows/shell-examples.yaml

# 2. 测试文件传输（需配置 SSH）
waterflow submit examples/workflows/file-transfer-examples.yaml

# 3. 发送 Webhook 通知
waterflow submit examples/workflows/webhook-notification.yaml

# 4. 执行数据库备份
waterflow submit examples/workflows/database-backup.yaml
```

#### 获取示例元数据

```bash
# 查看示例元数据 JSON
cat examples/workflows/examples-metadata.json

# 按类别查询（如果 API 支持）
curl http://localhost:8080/v1/examples?category=basic

# 按学习难度查询
curl http://localhost:8080/v1/examples?complexity=beginner
```

---

## 🎯 生产级模板

### workflows/ 目录 - 生产就绪的工作流模板

`workflows/` 文件夹包含可直接用于生产的、参数化的工作流模板。这些模板展示了 Waterflow 的最佳实践,并包含完整的错误处理和回滚逻辑。

**📖 完整模板文档:** [工作流模板库](../docs/templates/README.md)

每个模板都有详细的独立文档页面,包含:
- 详细参数说明
- 多个真实使用示例
- 定制指南
- 故障排查
- 最佳实践

### 可用模板概览

#### 1. 单服务器部署 (single-server-deployment.yaml)

**适用场景:** 小型 Web 应用、MVP 产品、开发/测试环境

**功能特性:**
- ✅ Git 代码拉取/更新
- ✅ Docker 镜像构建
- ✅ 优雅停止旧版本
- ✅ 启动新版本
- ✅ HTTP 健康检查（带重试）
- ✅ 失败自动回滚

**📖 完整文档:** [单服务器部署模板](../docs/templates/single-server-deployment.md)

**快速使用:**
```bash
cp examples/workflows/single-server-deployment.yaml my-app.yaml
# 编辑 vars.repo_url 和 vars.app_name
waterflow submit my-app.yaml
```

---

#### 2. 多服务器健康检查 (multi-server-health-check.yaml)

**适用场景:** 服务器巡检、部署前验证、容量规划

**功能特性:**
- ✅ SSH 远程执行,无需 Agent
- ✅ Matrix 策略并行检查
- ✅ 收集 CPU、内存、磁盘指标
- ✅ Python 生成 Markdown 报告
- ✅ 跨平台兼容 (Ubuntu、CentOS 等)

**📖 完整文档:** [多服务器健康检查模板](../docs/templates/multi-server-health-check.md)

**快速使用:**
```bash
# 配置 SSH 免密登录
ssh-copy-id user@server1

cp examples/workflows/multi-server-health-check.yaml health-check.yaml
# 编辑 jobs.health-check.strategy.matrix.server
waterflow submit health-check.yaml
```

---

#### 3. 分布式栈部署 (distributed-stack-deployment.yaml)

**适用场景:** Web 应用 + 数据库、微服务架构、多层应用

**功能特性:**
- ✅ Job 依赖管理 (数据库先启动)
- ✅ 跨服务器部署编排
- ✅ 数据库初始化和健康检查
- ✅ 应用与数据库连接验证
- ✅ 失败回滚选项

**📖 完整文档:** [分布式栈部署模板](../docs/templates/distributed-stack-deployment.md)

**快速使用:**
```bash
cp examples/workflows/distributed-stack-deployment.yaml my-stack.yaml
# 编辑数据库和应用配置
waterflow submit my-stack.yaml
```

---

### 模板选择指南

| 需求 | 推荐模板 |
|------|----------|
| 部署单个应用到一台服务器 | 单服务器部署 |
| 检查多台服务器健康状态 | 多服务器健康检查 |
| 部署应用和数据库 | 分布式栈部署 |
| 部署微服务到多台服务器 | 分布式栈部署 + 定制 |
| 定期监控服务器 | 多服务器健康检查 + Cron |

### 通过 API 获取模板

```bash
# 列出所有可用模板
curl http://localhost:8080/v1/templates

# 获取特定模板详情
curl http://localhost:8080/v1/templates/single-server-deployment

# 仅获取元数据 (不含 YAML 内容)
curl http://localhost:8080/v1/templates/single-server-deployment?content=false

# 按类别过滤
curl http://localhost:8080/v1/templates?category=deployment
```

### 与 CI/CD 集成示例

**GitHub Actions:**

```yaml
# .github/workflows/deploy.yml
name: Deploy to Production

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Submit deployment to Waterflow
        run: |
          curl -X POST http://waterflow-server:8080/v1/workflows \
            -H "Content-Type: application/json" \
            -d '{
              "workflow_file": "single-server-deployment",
              "vars": {
                "repo_url": "${{ github.repositoryUrl }}",
                "app_name": "my-app",
                "branch": "${{ github.ref_name }}"
              }
            }'
```

**GitLab CI:**

```yaml
# .gitlab-ci.yml
deploy:
  stage: deploy
  script:
    - |
      curl -X POST http://waterflow-server:8080/v1/workflows \
        -H "Content-Type: application/json" \
        -d @deployment-config.json
  only:
    - main
```

---

#### 2. multi-server-health-check.yaml - 多服务器健康检查

**适用场景:**
- 定期服务器巡检和监控
- 批量健康状态检查
- 资源使用监控
- 问题服务器快速定位

**功能特性:**
- ✅ Matrix 并行执行 (同时检查多台服务器)
- ✅ SSH 远程执行 (无需在每台服务器部署 Agent)
- ✅ CPU/内存/磁盘使用率检查
- ✅ 跨平台命令兼容 (带回退方案)
- ✅ 异常自动识别和告警
- ✅ 统一 Markdown 报告生成

**快速开始:**
```bash
# 1. 配置 SSH 免密登录
ssh-copy-id root@server1
ssh-copy-id root@server2
ssh-copy-id root@server3

# 2. 复制模板
cp examples/workflows/multi-server-health-check.yaml my-health-check.yaml

# 3. 编辑配置（修改 vars 部分）
vim my-health-check.yaml
# 必需修改:
#   servers: ["server1", "server2", "server3"]

# 4. 提交工作流
waterflow submit my-health-check.yaml

# 5. 查看报告
cat /tmp/health_report.md
```

**参数说明:**

| 参数 | 必需 | 默认值 | 说明 |
|------|------|--------|------|
| `servers` | ✅ | - | 服务器列表 (主机名或IP) |
| `ssh_user` | ⚙️ | root | SSH 用户名 |
| `cpu_threshold` | ⚙️ | 80 | CPU 使用率告警阈值 (%) |
| `memory_threshold` | ⚙️ | 85 | 内存使用率告警阈值 (%) |
| `disk_threshold` | ⚙️ | 90 | 磁盘使用率严重阈值 (%) |
| `report_path` | ⚙️ | /tmp/health_report.md | 报告保存路径 |

**使用示例:**

```yaml
# 示例 1: 检查 3 台 Web 服务器
vars:
  servers: ["web-1.example.com", "web-2.example.com", "web-3.example.com"]
  ssh_user: "ubuntu"

# 示例 2: 检查混合环境 (Web + DB + Cache)
vars:
  servers: ["web-1", "db-1", "cache-1"]
  cpu_threshold: 70
  memory_threshold: 80
  disk_threshold: 85

# 示例 3: 生产环境监控
vars:
  servers: ["prod-web-1", "prod-web-2", "prod-db-1"]
  cpu_threshold: 75
  memory_threshold: 80
  report_path: "/var/waterflow/reports/health_report.md"
```

**报告示例:**

```markdown
# 🏥 Multi-Server Health Check Report

**Generated:** 2026-01-07 10:30:45
**Total Servers:** 3

## ⚙️ Configured Thresholds

- CPU Warning: > 80%
- Memory Warning: > 85%
- Disk Critical: > 90%

## 📊 Server Health Status

| Server | CPU (%) | Memory (%) | Disk (%) | Status |
|--------|---------|------------|----------|--------|
| web-1  | 45.2    | 62.1       | 75.3     | ✅ OK     |
| web-2  | 88.3    | 78.5       | 45.2     | ⚠️ WARN  |
| db-1   | 38.5    | 55.3       | 92.1     | 🚨 CRITICAL |

## 📈 Summary

- **Total Servers:** 3
- ✅ **Healthy:** 1
- ⚠️ **Warnings:** 1
- 🚨 **Critical:** 1
- ❌ **Failed:** 0
- 📊 **Health Score:** 33.3%

## ⚠️ Issues Detected

- **web-2**: CPU usage high (88.3%, threshold: 80%)
- **db-1**: ⚠️ **CRITICAL** - Disk usage high (92.1%, threshold: 90%)
```

**定时健康检查 (Cron 集成):**

```bash
# 添加到 crontab - 每小时执行一次
0 * * * * /usr/local/bin/waterflow submit /etc/waterflow/workflows/multi-server-health-check.yaml >> /var/log/waterflow-health.log 2>&1

# 每天凌晨 2 点执行
0 2 * * * /usr/local/bin/waterflow submit /etc/waterflow/workflows/multi-server-health-check.yaml

# 每 15 分钟执行一次
*/15 * * * * /usr/local/bin/waterflow submit /etc/waterflow/workflows/multi-server-health-check.yaml
```

**前置条件:**
- SSH 免密登录配置完成
- Python 3 已安装 (用于报告生成)
- 目标服务器为 Linux 系统
- Waterflow Server 和 Agent 正在运行

**详细文档:** 查看模板文件顶部注释获取完整参数说明、架构设计和故障排查指南。

---

#### 3. distributed-stack-deployment.yaml - 分布式应用栈部署

**适用场景:**
- 多层架构应用部署 (Web + Database)
- 微服务栈部署
- 服务依赖编排
- 跨服务器协调部署

**功能特性:**
- ✅ Job 依赖编排 (Database → Application)
- ✅ 跨服务器部署 (db-server + app-server)
- ✅ 健康检查验证 (PostgreSQL + API)
- ✅ 网络连接测试
- ✅ 数据库初始化 (可选 SQL 脚本)
- ✅ 失败自动回滚 (可选)
- ✅ 完全参数化配置

**架构流程:**
```
Step 1: Deploy Database (db-server)
   ├─ Pull PostgreSQL image
   ├─ Start PostgreSQL container
   ├─ Wait for database ready (pg_isready)
   └─ Initialize schema (optional)
       ↓
Step 2: Deploy Application (app-server)
   ├─ Test network connectivity to database
   ├─ Pull application image
   ├─ Configure DATABASE_URL
   ├─ Start application container
   └─ Health check API endpoint
       ↓
Step 3: Rollback on Failure (optional)
   ├─ Stop application container
   └─ Stop database container
```

**快速开始:**
```bash
# 1. 确保两台服务器已部署 Waterflow Agent
waterflow node list | grep -E "(db-server|app-server)"

# 2. 复制模板
cp examples/workflows/distributed-stack-deployment.yaml my-stack.yaml

# 3. 编辑配置（修改 vars 部分）
vim my-stack.yaml
# 必需修改:
#   db_server: "your-db-server"     # Task Queue 名称
#   app_server: "your-app-server"   # Task Queue 名称
#   app_image: "your/app-image"     # 应用镜像
#   db_password: "your-password"    # 数据库密码

# 4. 提交工作流
waterflow submit my-stack.yaml

# 5. 查看状态
waterflow status <workflow-id>

# 6. 验证部署
curl http://app-server:3000/health
```

**参数说明:**

**数据库配置:**

| 参数 | 必需 | 默认值 | 说明 |
|------|------|--------|------|
| `db_server` | ✅ | db-server | 数据库服务器 Task Queue 名称 |
| `db_version` | ⚙️ | postgres:14 | PostgreSQL 版本 |
| `db_port` | ⚙️ | 5432 | PostgreSQL 端口 |
| `db_name` | ⚙️ | myapp | 数据库名称 |
| `db_user` | ⚙️ | appuser | 数据库用户 |
| `db_password` | ✅ | changeme | 数据库密码 (生产环境使用 Secret!) |
| `db_init_script` | ⚙️ | "" | 数据库初始化 SQL 脚本路径 |
| `db_data_volume` | ⚙️ | postgres_data | 数据卷名称 |

**应用配置:**

| 参数 | 必需 | 默认值 | 说明 |
|------|------|--------|------|
| `app_server` | ✅ | app-server | 应用服务器 Task Queue 名称 |
| `app_image` | ✅ | myapp/backend | 应用 Docker 镜像 |
| `app_version` | ⚙️ | latest | 应用版本/标签 |
| `app_port` | ⚙️ | 3000 | 应用 HTTP 端口 |
| `app_container_name` | ⚙️ | myapp | 应用容器名称 |
| `app_health_endpoint` | ⚙️ | /health | 健康检查端点 |

**部署配置:**

| 参数 | 必需 | 默认值 | 说明 |
|------|------|--------|------|
| `deploy_environment` | ⚙️ | dev | 环境标识 (dev/staging/prod) |
| `deploy_timeout` | ⚙️ | 300 | 最大部署时间 (秒) |
| `health_check_retries` | ⚙️ | 10 | 健康检查重试次数 |
| `health_check_delay` | ⚙️ | 5s | 健康检查重试间隔 |
| `rollback_on_failure` | ⚙️ | false | 失败时是否回滚数据库 |

**使用示例:**

```yaml
# 示例 1: Node.js Express + PostgreSQL
vars:
  db_server: "prod-db-1"
  db_name: "expressapp"
  db_password: "your-secure-password"
  app_server: "prod-web-1"
  app_image: "myorg/express-api"
  app_version: "v1.2.3"

# 示例 2: Python Flask + PostgreSQL with schema initialization
vars:
  db_server: "dev-db"
  db_init_script: "/opt/schema/init.sql"  # 创建表和初始数据
  app_server: "dev-app"
  app_image: "myorg/flask-api"
  app_health_endpoint: "/api/health"

# 示例 3: Production deployment with rollback enabled
vars:
  deploy_environment: "prod"
  db_server: "prod-db-primary"
  db_password: "your-production-password"
  app_server: "prod-app-1"
  app_image: "myorg/backend"
  app_version: "v2.0.0"
  rollback_on_failure: true
  health_check_retries: 20
```

**数据库初始化:**

创建 SQL 脚本 (`/opt/init.sql`):

```sql
-- Create tables
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample data
INSERT INTO users (username, email) VALUES
    ('alice', 'alice@example.com'),
    ('bob', 'bob@example.com')
ON CONFLICT (username) DO NOTHING;

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO appuser;
```

配置模板使用该脚本:
```yaml
vars:
  db_init_script: "/opt/init.sql"
```

**故障排查:**

| 问题 | 可能原因 | 解决方案 |
|------|----------|----------|
| 数据库未就绪 | 启动时间长 | 增加 `health_check_retries` (如 20) |
| 网络连接失败 | 防火墙/网络配置 | 检查防火墙规则,允许 app-server → db-server:5432 |
| 应用健康检查失败 | DATABASE_URL 错误 | 检查应用日志: `docker logs myapp` |
| 回滚未触发 | 配置未启用 | 设置 `rollback_on_failure: true` |
| 初始化脚本失败 | SQL 语法错误 | 验证 SQL 脚本,检查数据库权限 |

**Docker Compose 集成:**

也可以使用 Docker Compose 部署 (参考 `examples/configs/db-compose.yml`):

```bash
# 在 db-server 上运行
docker-compose -f examples/configs/db-compose.yml up -d

# 或在 workflow 中使用 docker/compose@v1 节点
- name: Deploy database
  uses: docker/compose@v1
  with:
    file: examples/configs/db-compose.yml
    action: up
    args: -d
```

**网络配置要求:**

1. **服务器间通信:**
   - app-server 必须能访问 db-server:5432
   - 配置防火墙规则或使用同一网络

2. **DATABASE_URL 格式:**
   ```
   postgres://username:password@host:port/database
   # 示例:
   postgres://appuser:changeme@db-server:5432/myapp
   ```

3. **主机名解析:**
   - 确保 `db-server` 主机名能解析 (DNS 或 /etc/hosts)
   - 或使用 IP 地址替代主机名

**前置条件:**
- Waterflow Agents 部署在 db-server 和 app-server
- Docker 安装在两台服务器
- PostgreSQL 客户端工具 (psql, pg_isready) 安装在 db-server
- 网络连接: app-server → db-server:5432
- (可选) 数据库初始化脚本准备

**详细文档:** 查看模板文件顶部注释获取完整架构说明、参数详情和故障排查指南。

---

#### 其他模板（即将推出）

- **blue-green-deployment.yaml** - 蓝绿部署模板
- **canary-deployment.yaml** - 金丝雀部署模板

**适用场景:**
- 定期服务器巡检和监控
- 批量健康状态检查
- 资源使用监控
- 问题服务器快速定位

**功能特性:**
- ✅ Matrix 并行执行 (同时检查多台服务器)
- ✅ SSH 远程执行 (无需在每台服务器部署 Agent)
- ✅ CPU/内存/磁盘使用率检查
- ✅ 跨平台命令兼容 (带回退方案)
- ✅ 异常自动识别和告警
- ✅ 统一 Markdown 报告生成

**快速开始:**
```bash
# 1. 配置 SSH 免密登录
ssh-copy-id root@server1
ssh-copy-id root@server2
ssh-copy-id root@server3

# 2. 复制模板
cp examples/workflows/multi-server-health-check.yaml my-health-check.yaml

# 3. 编辑配置（修改 vars 部分）
vim my-health-check.yaml
# 必需修改:
#   servers: ["server1", "server2", "server3"]

# 4. 提交工作流
waterflow submit my-health-check.yaml

# 5. 查看报告
cat /tmp/health_report.md
```

**参数说明:**

| 参数 | 必需 | 默认值 | 说明 |
|------|------|--------|------|
| `servers` | ✅ | - | 服务器列表 (主机名或IP) |
| `ssh_user` | ⚙️ | root | SSH 用户名 |
| `cpu_threshold` | ⚙️ | 80 | CPU 使用率告警阈值 (%) |
| `memory_threshold` | ⚙️ | 85 | 内存使用率告警阈值 (%) |
| `disk_threshold` | ⚙️ | 90 | 磁盘使用率严重阈值 (%) |
| `report_path` | ⚙️ | /tmp/health_report.md | 报告保存路径 |

**使用示例:**

```yaml
# 示例 1: 检查 3 台 Web 服务器
vars:
  servers: ["web-1.example.com", "web-2.example.com", "web-3.example.com"]
  ssh_user: "ubuntu"

# 示例 2: 检查混合环境 (Web + DB + Cache)
vars:
  servers: ["web-1", "db-1", "cache-1"]
  cpu_threshold: 70
  memory_threshold: 80
  disk_threshold: 85

# 示例 3: 生产环境监控
vars:
  servers: ["prod-web-1", "prod-web-2", "prod-db-1"]
  cpu_threshold: 75
  memory_threshold: 80
  report_path: "/var/waterflow/reports/health_report.md"
```

**报告示例:**

```markdown
# 🏥 Multi-Server Health Check Report

**Generated:** 2026-01-07 10:30:45
**Total Servers:** 3

## ⚙️ Configured Thresholds

- CPU Warning: > 80%
- Memory Warning: > 85%
- Disk Critical: > 90%

## 📊 Server Health Status

| Server | CPU (%) | Memory (%) | Disk (%) | Status |
|--------|---------|------------|----------|--------|
| web-1  | 45.2    | 62.1       | 75.3     | ✅ OK     |
| web-2  | 88.3    | 78.5       | 45.2     | ⚠️ WARN  |
| db-1   | 38.5    | 55.3       | 92.1     | 🚨 CRITICAL |

## 📈 Summary

- **Total Servers:** 3
- ✅ **Healthy:** 1
- ⚠️ **Warnings:** 1
- 🚨 **Critical:** 1
- ❌ **Failed:** 0
- 📊 **Health Score:** 33.3%

## ⚠️ Issues Detected

- **web-2**: CPU usage high (88.3%, threshold: 80%)
- **db-1**: ⚠️ **CRITICAL** - Disk usage high (92.1%, threshold: 90%)
```

**定时健康检查 (Cron 集成):**

```bash
# 添加到 crontab - 每小时执行一次
0 * * * * /usr/local/bin/waterflow submit /etc/waterflow/workflows/multi-server-health-check.yaml >> /var/log/waterflow-health.log 2>&1

# 每天凌晨 2 点执行
0 2 * * * /usr/local/bin/waterflow submit /etc/waterflow/workflows/multi-server-health-check.yaml

# 每 15 分钟执行一次
*/15 * * * * /usr/local/bin/waterflow submit /etc/waterflow/workflows/multi-server-health-check.yaml
```

**前置条件:**
- SSH 免密登录配置完成
- Python 3 已安装 (用于报告生成)
- 目标服务器为 Linux 系统
- Waterflow Server 和 Agent 正在运行

**详细文档:** 查看模板文件顶部注释获取完整参数说明、架构设计和故障排查指南。



## 🚀 快速测试

### 方法 1: 使用 curl 提交工作流

```bash
# 提交 hello-world 示例
curl -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/json" \
  -d "{\"yaml\": \"$(cat examples/hello-world.yaml | sed 's/"/\\"/g' | tr '\n' ' ')\"}"
```

### 方法 2: 使用测试脚本

```bash
# 运行完整部署测试（包含工作流提交）
./scripts/test-deployment.sh
```

## 📊 查看执行结果

### 1. 通过 API 查询状态

```bash
# 列出所有工作流
curl http://localhost:8080/v1/workflows

# 查询特定工作流
curl http://localhost:8080/v1/workflows/{workflow-id}
```

### 2. 通过 Temporal UI

访问 http://localhost:8088 查看工作流执行详情、历史记录和可视化流程。

## 📖 扩展阅读

- [Waterflow YAML DSL 语法](../docs/adr/0004-yaml-dsl-syntax.md)
- [表达式系统](../docs/adr/0005-expression-system-syntax.md)
- [部署指南](../docs/deployment.md)
- [开发文档](../docs/development.md)

## 💡 自定义工作流

您可以基于这些示例创建自己的工作流。基本结构：

```yaml
name: My Workflow
on:
  workflow_dispatch:  # 手动触发

vars:
  my_var: "value"

jobs:
  my_job:
    runs-on: waterflow-server
    steps:
      - name: My Step
        run: echo "Hello ${{ vars.my_var }}"
```

更多语法和功能请参考文档。
