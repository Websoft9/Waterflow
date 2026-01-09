# Waterflow 部署指南

## Docker 单容器部署

### 快速开始

最小配置启动 Waterflow Server:

```bash
docker run -d \
  --name waterflow-server \
  -p 8080:8080 \
  -e WATERFLOW_TEMPORAL_HOST=temporal.example.com:7233 \
  waterflow/server:latest
```

### 环境变量配置

| 环境变量 | 说明 | 默认值 | 必需 |
|---------|------|--------|------|
| `WATERFLOW_TEMPORAL_HOST` | Temporal Server gRPC 地址 | localhost:7233 | ✅ |
| `WATERFLOW_SERVER_PORT` | HTTP API 监听端口 | 8080 | ❌ |
| `WATERFLOW_SERVER_API_KEY` | API 认证密钥 (未设置则不启用) | - | ❌ |
| `WATERFLOW_LOG_LEVEL` | 日志级别 (debug/info/warn/error) | info | ❌ |
| `WATERFLOW_SERVER_METRICS_PORT` | Prometheus 指标端口 | 9090 | ❌ |
| `WATERFLOW_SERVER_TLS_CERT_FILE` | TLS 证书文件路径 (可选 HTTPS) | - | ❌ |
| `WATERFLOW_SERVER_TLS_KEY_FILE` | TLS 密钥文件路径 (可选 HTTPS) | - | ❌ |

配置优先级: **环境变量 > 配置文件 > 默认值**

### 常用部署场景

#### 场景 1: 开发环境 (本地测试)

```bash
docker run -d \
  --name waterflow-server \
  -p 8080:8080 \
  -e WATERFLOW_TEMPORAL_HOST=localhost:7233 \
  -e WATERFLOW_LOG_LEVEL=debug \
  waterflow/server:latest
```

#### 场景 2: 生产环境 (启用认证 + 指标)

```bash
docker run -d \
  --name waterflow-server \
  -p 8080:8080 \
  -p 9090:9090 \
  -e WATERFLOW_TEMPORAL_HOST=temporal-prod.internal:7233 \
  -e WATERFLOW_SERVER_API_KEY=${API_SECRET} \
  -e WATERFLOW_LOG_LEVEL=warn \
  -e WATERFLOW_SERVER_METRICS_PORT=9090 \
  waterflow/server:latest
```

#### 场景 3: 使用配置文件

```bash
docker run -d \
  --name waterflow-server \
  -p 8080:8080 \
  -v /opt/waterflow/config.yaml:/etc/waterflow/config.yaml:ro \
  -e WATERFLOW_TEMPORAL_HOST=temporal:7233 \
  waterflow/server:latest
```

### 健康检查

```bash
# 检查容器健康状态
docker inspect waterflow-server --format='{{.State.Health.Status}}'
# 预期输出: healthy

# 手动测试健康端点
curl http://localhost:8080/health
# 预期输出: {"status":"healthy","timestamp":"..."}
```

### 资源要求

| 资源 | 最小值 | 推荐值 |
|------|--------|--------|
| CPU | 0.5 core | 2 cores |
| 内存 | 128MB | 512MB |
| 磁盘 | 100MB | 500MB |

### 镜像版本说明

- `latest` - 最新稳定版本 (main 分支)
- `develop` - 开发版本 (可能不稳定)
- `v1.0.0` - 特定版本号
- `v1.0` - 主次版本 (自动跟随补丁版本)

### 故障排查

#### 容器启动失败

```bash
# 查看容器日志
docker logs waterflow-server

# 常见错误:
# - "temporal.host is required" → 未设置 WATERFLOW_TEMPORAL_HOST
# - "connection refused" → Temporal Server 不可达
```

#### 健康检查失败

```bash
# 进入容器检查
docker exec -it waterflow-server sh

# 手动测试健康端点
curl localhost:8080/health
```

---

## 使用 Docker Compose 快速部署

### 前置要求

- Docker Engine 20.10+
- Docker Compose 2.0+
- 最少 2GB 可用内存
- 端口 5432, 7233, 8080, 8088 未被占用

### 快速启动

1. **（可选）配置环境变量**

```bash
# 进入 deployments 目录
cd deployments

# 复制环境变量模板
cp .env.example .env

# 编辑 .env 文件，修改敏感信息（如数据库密码）
vi .env
```

2. **启动服务栈**

```bash
# 启动所有服务
docker-compose up -d
```

服务启动顺序：PostgreSQL → Temporal Server → Temporal UI + Waterflow

3. **查看服务状态**

```bash
docker-compose ps
```

所有服务应显示 `healthy` 状态。

4. **查看日志**

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f waterflow
docker-compose logs -f temporal
```

### 验证部署

#### 1. 健康检查

```bash
# Waterflow 健康检查
curl http://localhost:8080/health

# 预期输出
{"status":"ok"}
```

#### 2. Temporal UI

访问 http://localhost:8088 查看 Temporal Web UI

#### 3. 提交测试工作流

```bash
curl -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{"yaml":"name: test-workflow\non: push\njobs:\n  test:\n    steps:\n      - name: Hello\n        run: echo Hello\n"}'
```

预期输出：
```json
{
  "id": "ae4ee6a3-6ad9-4ed1-a793-072e8061f8a7",
  "run_id": "8adbd563-0060-4df2-bc4c-fbd0a46f3276",
  "name": "test-workflow",
  "status": "running",
  "created_at": "2025-12-22T03:37:11Z",
  "url": "/v1/workflows/ae4ee6a3-6ad9-4ed1-a793-072e8061f8a7"
}
```

#### 4. 查询工作流状态

```bash
curl http://localhost:8080/v1/workflows/{workflow-id}
```

### 环境变量配置

Waterflow 使用**两层配置架构**：

#### 1. docker-compose.yaml（容器层）
通过 `.env` 文件或环境变量配置容器：

```bash
cd deployments

# 创建 .env 文件（推荐用于生产环境）
cp .env.example .env

# 编辑 .env 修改敏感信息
vi .env
```

示例 .env 内容：
```bash
POSTGRES_PASSWORD=your_secure_password
WATERFLOW_LOG_LEVEL=debug
```

#### 2. config.yaml（应用层）
容器内的应用配置文件，提供默认值。

#### 配置优先级（高到低）

Waterflow 配置解析遵循以下优先级顺序:
1. **命令行参数** - 运行时通过 `--port`, `--log-level` 等指定 (仅适用于非Docker部署)
2. **环境变量** - `WATERFLOW_*` 前缀,推荐用于容器部署
3. **配置文件** - `config.yaml` 或 `config.toml` (可选)
4. **默认值** - 内置的开发友好默认值

**示例**: 如果 `config.yaml` 中设置 `server.port: 8080`,同时设置环境变量 `WATERFLOW_SERVER_PORT=9090`,最终生效的是 `9090`。

#### 配置方式

**方式一: 纯环境变量 (推荐 - Docker/Kubernetes)**

所有配置通过环境变量提供,无需配置文件:

```bash
# .env 文件
WATERFLOW_SERVER_PORT=8080
WATERFLOW_LOG_LEVEL=info
WATERFLOW_LOG_FORMAT=json
WATERFLOW_TEMPORAL_HOST=temporal:7233
WATERFLOW_TEMPORAL_NAMESPACE=default
```

**方式二: 配置文件 + 环境变量覆盖**

使用基础配置文件,通过环境变量覆盖特定值:

```yaml
# config.yaml (基础配置)
server:
  port: 8080
log:
  level: info
temporal:
  host: temporal:7233
```

```bash
# 环境变量覆盖日志级别
export WATERFLOW_LOG_LEVEL=debug
```

#### 完整配置参考

**所有可用的环境变量和配置选项请参考** **[配置参考文档](configuration.md)**,包括:
- Server/Agent 所有配置项的环境变量映射表
- 配置验证规则和错误信息说明
- 开发/测试/生产环境配置示例
- 故障排查指南

#### 可配置项 (常用)

**数据库配置 (.env)**:
```bash
POSTGRES_USER=temporal          # 数据库用户名
POSTGRES_PASSWORD=temporal      # 数据库密码(生产环境请修改)
POSTGRES_DB=temporal            # 数据库名称
```

**Waterflow 配置 (.env 或环境变量)**:
```bash
# Server 配置
WATERFLOW_SERVER_HOST=0.0.0.0
WATERFLOW_SERVER_PORT=8080
WATERFLOW_SERVER_READ_TIMEOUT=30s
WATERFLOW_SERVER_WRITE_TIMEOUT=30s

# 日志配置
WATERFLOW_LOG_LEVEL=info          # debug|info|warn|error
WATERFLOW_LOG_FORMAT=json         # json|text

# Temporal 配置
WATERFLOW_TEMPORAL_HOST=temporal:7233
WATERFLOW_TEMPORAL_NAMESPACE=default
WATERFLOW_TEMPORAL_TASK_QUEUE=waterflow-server

# Agent 配置 (仅Agent使用)
WATERFLOW_AGENT_TASK_QUEUES=linux-amd64,linux-common  # 必需
WATERFLOW_AGENT_PLUGIN_DIR=/opt/waterflow/plugins
```

**说明**: 
- 配置文件完全可选,可以完全依靠环境变量运行
- 环境变量命名规则: `WATERFLOW_` + 配置路径(用下划线替换点)
- 完整列表见 [配置参考文档](configuration.md)

#### 示例: 修改日志级别

**方法 1: 使用 .env 文件 (推荐)**
```bash
cd deployments
echo "WATERFLOW_LOG_LEVEL=debug" >> .env
docker-compose up -d
```

**方法 2：直接设置环境变量**
```bash
cd deployments
WATERFLOW_LOG_LEVEL=debug docker-compose up -d
```

**方法 3：修改 docker-compose.yaml**

### 常见问题排查

#### 服务无法启动

```bash
# 检查端口占用
netstat -tuln | grep -E '5432|7233|8080|8088'

# 查看详细错误日志
cd deployments
docker-compose logs waterflow
```

#### Temporal 连接失败

```bash
cd deployments
# 检查 Temporal 服务状态
docker-compose ps temporal

# 验证 Temporal 健康状态
docker-compose exec temporal temporal operator cluster health
```

#### 数据库连接问题

```bash
cd deployments
# 检查 PostgreSQL 容器
docker-compose ps postgresql

# 查看数据库日志
docker-compose logs postgresql

# 验证数据库连接
docker-compose exec postgresql psql -U temporal -d temporal -c "SELECT 1"
```

### 停止和清理

```bash
cd deployments
# 停止所有服务
docker-compose down

# 停止服务并删除数据卷（警告：删除所有数据）
docker-compose down -v

# 重建服务
docker-compose up -d --build
```

### 生产环境建议

1. **数据持久化**：确保 PostgreSQL 数据卷挂载到宿主机可靠存储
2. **资源限制**：在 docker-compose.yaml 中配置 `resources` 限制
3. **日志管理**：配置日志驱动和日志轮转
4. **监控告警**：集成 Prometheus + Grafana 监控指标
5. **安全加固**：
   - 使用 secrets 管理敏感信息
   - 配置防火墙规则
   - 启用 TLS/SSL
   - 定期更新镜像

### 架构说明

```
┌─────────────┐     ┌──────────────┐
│  Waterflow  │────>│   Temporal   │
│   :8080     │     │    :7233     │
└─────────────┘     └──────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │ PostgreSQL   │
                    │    :5432     │
                    └──────────────┘
```

- **Waterflow**：工作流 REST API 服务
- **Temporal**：工作流引擎和任务编排
- **PostgreSQL**：Temporal 持久化存储
- **Temporal UI**：工作流可视化界面

### 下一步

- 查看 [API 文档](../api/README.md) 了解完整 REST API
- 阅读 [配置文档](configuration.md) 了解高级配置
- 参考 [开发文档](development.md) 进行本地开发
- 查看 [故障排查文档](troubleshooting.md) 解决常见问题

---

## 生产环境部署

### 资源规划

#### 最小配置（单机部署，工作流 < 100/天）
- **CPU**: 2 核
- **内存**: 4GB
- **磁盘**: 20GB SSD
- **网络**: 100Mbps

#### 推荐配置（中等负载，工作流 < 1000/天）
- **CPU**: 4 核
- **内存**: 8GB
- **磁盘**: 50GB SSD
- **网络**: 1Gbps

#### 高性能配置（高负载，工作流 > 1000/天）
- **CPU**: 8+ 核
- **内存**: 16GB+
- **磁盘**: 100GB+ NVMe SSD
- **网络**: 10Gbps

### 生产环境配置示例

#### docker-compose.prod.yaml

```yaml
services:
  waterflow:
    image: waterflow:latest
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G
    restart: always
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

### 安全加固

1. **修改默认密码**: 更改 PostgreSQL 密码
2. **限制端口暴露**: Temporal 和 PostgreSQL 不对外开放
3. **配置防火墙**: 只开放 8080 端口
4. **启用 TLS**: 生产环境必须启用 HTTPS
5. **定期更新**: 及时应用安全补丁

### 监控集成

参考 [监控配置文档](../deployments/monitoring/README.md) 集成 Prometheus + Grafana。

---

## 二进制独立部署

### 方式 1: 下载预编译二进制

```bash
# 下载最新版本
VERSION=v1.0.0
wget https://github.com/websoft9/waterflow/releases/download/${VERSION}/waterflow-server-linux-amd64
wget https://github.com/websoft9/waterflow/releases/download/${VERSION}/waterflow-agent-linux-amd64

# 安装到系统目录
sudo mkdir -p /opt/waterflow/bin
sudo mv waterflow-server-linux-amd64 /opt/waterflow/bin/server
sudo mv waterflow-agent-linux-amd64 /opt/waterflow/bin/agent
sudo chmod +x /opt/waterflow/bin/*
```

### 方式 2: 源码编译

```bash
# 前置要求：Go 1.24+
git clone https://github.com/websoft9/waterflow.git
cd waterflow
make build

# 编译结果在 bin/ 目录
ls -lh bin/
```

### 目录结构规范

```
/opt/waterflow/
├── bin/
│   ├── server         # Server 二进制
│   └── agent          # Agent 二进制
├── plugins/           # 自定义插件目录
└── logs/              # 日志目录

/etc/waterflow/
├── config.yaml        # Server 配置
└── agent.yaml         # Agent 配置
```

### 配置文件准备

```bash
# 创建配置目录
sudo mkdir -p /etc/waterflow

# 复制配置模板
sudo cp examples/configs/config.example.yaml /etc/waterflow/config.yaml

# 编辑配置
sudo vim /etc/waterflow/config.yaml
```

### 手动启动

```bash
# 启动 Server
/opt/waterflow/bin/server --config /etc/waterflow/config.yaml

# 验证服务
curl http://localhost:8080/health
```

### Systemd 集成（推荐）

创建 Systemd 服务文件：

```bash
sudo cp deployments/systemd/waterflow-server.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable waterflow-server
sudo systemctl start waterflow-server
```

查看服务状态：

```bash
sudo systemctl status waterflow-server
sudo journalctl -u waterflow-server -f
```

---

## 备份和恢复

### 备份内容清单

#### 必须备份
1. **PostgreSQL 数据库** - Temporal 工作流状态和历史
2. **配置文件** - `/etc/waterflow/config.yaml`
3. **环境变量** - `deployments/.env`

#### 可选备份
1. **日志文件** - 用于审计和故障排查
2. **工作流定义** - YAML 文件（建议 Git 管理）

### 手动备份

```bash
# 备份数据库
./scripts/backup-database.sh

# 备份配置
./scripts/backup-configs.sh
```

### 自动化备份

使用 Cron 定时备份：

```bash
# 编辑 crontab
crontab -e

# 每天凌晨 2 点备份数据库
0 2 * * * /opt/waterflow/scripts/backup-database.sh

# 每周日凌晨 3 点备份配置
0 3 * * 0 /opt/waterflow/scripts/backup-configs.sh
```

### 数据恢复

```bash
# 恢复数据库
./scripts/restore-database.sh /backups/waterflow/waterflow_db_20260109.sql.gz

# 恢复配置
./scripts/restore-configs.sh /backups/waterflow/waterflow_configs_20260109.tar.gz
```

---

## 版本升级

### 升级前准备

1. **检查版本兼容性**
```bash
curl http://localhost:8080/version
```

2. **备份数据**
```bash
./scripts/backup-database.sh
./scripts/backup-configs.sh
```

3. **测试环境验证** - 建议先在测试环境验证升级流程

### Docker Compose 环境升级

#### 停机升级（简单，适用单实例）

```bash
cd deployments

# 停止服务
docker-compose stop waterflow

# 拉取新镜像
docker-compose pull waterflow

# 启动服务
docker-compose up -d waterflow

# 查看日志
docker-compose logs -f waterflow
```

### 二进制环境升级

```bash
# 使用升级脚本
./scripts/upgrade-binary.sh v1.2.0
```

### 回滚

如果升级失败，可以快速回滚：

```bash
# Docker Compose 回滚
cd deployments
docker-compose pull waterflow:v1.1.0
docker-compose up -d waterflow

# 二进制回滚
sudo systemctl stop waterflow-server
sudo cp /opt/waterflow/bin/server.backup /opt/waterflow/bin/server
sudo systemctl start waterflow-server
```

### 升级验证清单

- [ ] 健康检查通过 (`/health` 返回 200)
- [ ] 版本号正确 (`/version` 显示新版本)
- [ ] 依赖服务连接正常 (`/ready` 返回 200)
- [ ] 提交新工作流成功
- [ ] 查询历史工作流成功
- [ ] 无错误日志

---

## 故障排查

常见问题请参考 [故障排查文档](troubleshooting.md)。

### 快速诊断

```bash
# 检查服务健康
curl http://localhost:8080/health
curl http://localhost:8080/ready

# 查看日志
docker-compose logs -f waterflow  # Docker 环境
sudo journalctl -u waterflow-server -f  # Systemd 环境

# 检查资源
docker stats  # Docker 环境
top  # 系统资源
```
