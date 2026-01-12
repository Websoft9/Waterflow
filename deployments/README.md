# Waterflow Docker Compose 部署

一键部署 Waterflow 完整技术栈 (PostgreSQL + Temporal + Waterflow Server + Agent)。

## 目录
1. [架构说明](#架构说明)
2. [前置要求](#前置要求)
3. [快速启动](#快速启动)
4. [配置说明](#配置说明)
5. [验证部署](#验证部署)
6. [测试工作流](#测试工作流)
7. [服务端点](#服务端点)
8. [故障排查](#故障排查)
9. [数据持久化](#数据持久化)
10. [停止和清理](#停止和清理)

## 架构说明

**服务列表：**
- **PostgreSQL 15**: Temporal 数据库 (内部服务)
- **Temporal Server 1.22.0**: 工作流引擎 (内部服务)
- **Temporal UI 2.22.0**: Web UI 调试工具 (端口 8088)
- **Waterflow Server**: REST API 服务 (端口 8080)
- **Waterflow Agent**: 工作流执行器 (linux-amd64, linux-common)

**网络架构:**
```
┌─────────────────────────────────────────────────────┐
│ Docker Bridge Network (waterflow-network)          │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ┌──────────────┐      ┌──────────────┐           │
│  │ PostgreSQL   │◄─────│   Temporal   │           │
│  │  (内部 5432) │      │  (内部 7233) │           │
│  └──────────────┘      └──────┬───────┘           │
│                                │                    │
│                        ┌───────▼────────┐          │
│                        │   Waterflow    │          │
│                        │   (对外 8080)   │          │
│                        └───────┬────────┘          │
│                                │                    │
│                        ┌───────▼────────┐          │
│                        │     Agent      │          │
│                        │   (内部执行)    │          │
│                        └────────────────┘          │
│                                                     │
│                        ┌────────────────┐          │
│                        │  Temporal UI   │          │
│                        │  (对外 8088)    │          │
│                        └────────────────┘          │
│                                                     │
└─────────────────────────────────────────────────────┘
```

**设计原则 (ADR-0008):**
- Temporal 作为内部服务,端口不对外暴露
- Waterflow Server 是唯一对外 API 入口
- 所有服务通过 Docker 内部网络通信
- 数据持久化通过 Docker 卷管理

## 前置要求

### 软件依赖
- **Docker**: >= 20.10
- **Docker Compose**: >= 2.0 (推荐使用 Compose V2)
- **系统要求**: Linux/MacOS/Windows (WSL2)
- **内存**: >= 4GB (建议 8GB)
- **磁盘**: >= 10GB 可用空间

### 验证环境
```bash
# 检查 Docker 版本
docker --version
# 预期: Docker version 20.10.x 或更高

# 检查 Docker Compose 版本
docker compose version
# 预期: Docker Compose version v2.x.x 或更高

# 验证 Docker 运行
docker ps
# 预期: 无错误
```

## 快速启动

### 步骤 1: 克隆仓库
```bash
git clone https://github.com/websoft9/waterflow.git
cd waterflow/deployments
```

### 步骤 2: 配置环境变量 (可选)
```bash
# 复制配置模板
cp .env.example .env

# 修改配置 (可选,默认配置即可使用)
nano .env

# 示例修改:
# - 修改 Server 端口为 9090
# - 启用 API 认证
# - 调整日志级别为 debug
```

### 步骤 3: 启动服务
```bash
# 启动所有服务
docker compose up -d

# 查看启动日志
docker compose logs -f

# Ctrl+C 退出日志,服务继续运行
```

### 步骤 4: 等待服务就绪
```bash
# 等待 2-3 分钟,直到所有服务健康

# 检查服务状态
docker compose ps

# 预期输出:
# NAME                       STATUS              PORTS
# waterflow-postgresql       Up (healthy)        -
# waterflow-temporal         Up (healthy)        -
# waterflow-server           Up (healthy)        0.0.0.0:8080->8080/tcp
# waterflow-temporal-ui      Up                  0.0.0.0:8088->8088/tcp
# waterflow-agent-linux-1    Up (healthy)        -
```

## 配置说明

### 环境变量配置

Waterflow 完全支持通过环境变量配置,配置文件是可选的。所有配置通过 `.env` 文件管理。

**配置优先级**: 环境变量 > 配置文件 > 默认值

**PostgreSQL 配置:**
```bash
POSTGRES_USER=temporal          # 数据库用户名
POSTGRES_PASSWORD=temporal      # 数据库密码
POSTGRES_DB=temporal            # 数据库名称
```

**Temporal 配置:**
```bash
TEMPORAL_LOG_LEVEL=info         # 日志级别: debug, info, warn, error
```

**Waterflow Server 配置:**
```bash
WATERFLOW_SERVER_PORT=8080                  # API 端口
WATERFLOW_SERVER_READ_TIMEOUT=30s           # 读取超时
WATERFLOW_SERVER_WRITE_TIMEOUT=30s          # 写入超时
WATERFLOW_TEMPORAL_HOST=temporal:7233       # Temporal 连接地址 (内部网络)
WATERFLOW_TEMPORAL_NAMESPACE=default        # Temporal 命名空间
WATERFLOW_LOG_LEVEL=info                    # 日志级别: debug|info|warn|error
WATERFLOW_LOG_FORMAT=json                   # 日志格式: json|text
# WATERFLOW_SERVER_API_KEY=your-key         # API 认证(可选,留空不启用)
```

**Agent 配置:**
```bash
# 必需配置
WATERFLOW_AGENT_TASK_QUEUES=linux-amd64,linux-common  # Task Queue 列表 (必需)
# 或使用向后兼容的变量名
TASK_QUEUES=linux-amd64,linux-common

# 可选配置
WATERFLOW_AGENT_PLUGIN_DIR=/opt/waterflow/plugins   # 插件目录
WATERFLOW_AGENT_METRICS_PORT=9091                   # Prometheus 端口
WATERFLOW_TEMPORAL_HOST=temporal:7233               # Temporal 地址
WATERFLOW_LOG_LEVEL=info                            # 日志级别
```

**Temporal UI 配置:**
```bash
TEMPORAL_UI_PORT=8088               # UI 端口
```

**环境变量命名规则**:
- Server/Agent 配置使用 `WATERFLOW_` 前缀
- 配置路径用下划线连接: `server.port` → `WATERFLOW_SERVER_PORT`
- 完整的环境变量列表和验证规则请参考 **[配置参考文档](../docs/configuration.md)**

### 使用示例

**开发环境 (调试模式)**:
```bash
# .env 配置
WATERFLOW_LOG_LEVEL=debug
TEMPORAL_LOG_LEVEL=debug

# 重启服务
docker compose restart
```

**生产环境 (启用认证):**
```bash
# .env 配置
WATERFLOW_API_KEY=my-secret-key-12345
WATERFLOW_LOG_LEVEL=warn

# 重启服务
docker compose restart
```

**自定义端口:**
```bash
# .env 配置
WATERFLOW_SERVER_PORT=9090
TEMPORAL_UI_PORT=9088

# 重启服务
docker compose down && docker compose up -d
```

## 验证部署

### 验证 1: 健康检查
```bash
# 验证所有服务健康
docker compose ps

# 预期: postgresql, temporal, waterflow, agent 显示 healthy

# 手动检查各服务
curl http://localhost:8080/health
# 预期: {"status":"healthy","timestamp":"2026-01-09T..."}

curl http://localhost:8080/ready
# 预期: {"status":"ready","timestamp":"2026-01-09T..."}

curl -I http://localhost:8088
# 预期: HTTP/1.1 200 OK
```

### 验证 2: 内部网络连通性
```bash
# 验证 Temporal 未暴露到宿主机 (应无输出)
docker port waterflow-temporal
# 预期: 无输出 (端口未映射)

# 验证内部网络连通
docker exec waterflow-agent-linux-1 nc -zv temporal 7233
# 预期: Connection to temporal 7233 port [tcp/*] succeeded!
```

### 验证 3: 查看节点列表
```bash
# 查询可用节点
curl http://localhost:8080/v1/nodes

# 预期输出: JSON 数组,包含核心节点
# [
#   {"name":"exec/shell","version":"v1",...},
#   {"name":"exec/script","version":"v1",...},
#   {"name":"flow/sleep","version":"v1",...},
#   ...
# ]
```

## 测试工作流

### 提交测试工作流
```bash
# 进入项目根目录
cd ..

# 提交 Hello World 工作流
curl -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/yaml" \
  --data-binary @examples/hello-world.yaml

# 预期输出:
# {
#   "workflow_id": "abc123...",
#   "run_id": "def456...",
#   "status": "pending"
# }

# 保存 workflow_id 用于后续查询
WORKFLOW_ID="abc123..."  # 替换为实际 ID
```

### 查询工作流状态
```bash
# 查询状态
curl http://localhost:8080/v1/workflows/$WORKFLOW_ID

# 预期输出:
# {
#   "workflow_id": "abc123...",
#   "status": "completed",  # 或 running, failed
#   "name": "Hello World",
#   ...
# }
```

### 查看工作流日志
```bash
# 获取日志
curl http://localhost:8080/v1/workflows/$WORKFLOW_ID/logs

# 预期输出: JSON Lines 格式的日志
```

### 在 Temporal UI 查看
```bash
# 浏览器访问
open http://localhost:8088

# 导航至: Workflows → 找到你的工作流
# 查看: Event History、Input/Output、重试记录等
```

## 服务端点

启动成功后,以下端点可访问:

### Waterflow Server
- **API 地址**: http://localhost:8080
- **健康检查**: http://localhost:8080/health
- **就绪检查**: http://localhost:8080/ready
- **版本信息**: http://localhost:8080/version
- **节点列表**: http://localhost:8080/v1/nodes
- **Prometheus 指标**: http://localhost:8080/metrics

### Temporal UI
- **UI 地址**: http://localhost:8088
- **用途**: 查看工作流执行状态、Event History、Task Queue 状态
- **默认 Namespace**: default

### 内部服务 (不对外暴露)
- **PostgreSQL**: 仅内部网络访问 (temporal 使用)
- **Temporal gRPC**: 仅内部网络访问 (waterflow 和 agent 使用)

### 网络端口

**需要开放的端口:**
- **8080**: Waterflow API (必需)
- **8088**: Temporal UI (可选,仅开发环境)

**不需要开放的端口** (内部服务):
- 5432: PostgreSQL
- 7233: Temporal gRPC

## 故障排查

### 问题 0: docker compose 命令未找到
```bash
# 症状: bash: docker compose: command not found 或 docker-compose: command not found

# 解决方案 1: 安装 Docker Compose V2 (推荐)
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install docker-compose-plugin

# 验证安装
docker compose version
# 预期: Docker Compose version v2.x.x

# 解决方案 2: 使用 Docker Compose V1 (仅当 V2 不可用)
# 将所有 "docker compose" 命令改为 "docker-compose"
docker-compose up -d
```

### 问题 1: 服务启动失败
```bash
# 查看服务日志
docker compose logs waterflow
docker compose logs temporal
docker compose logs postgresql

# 常见原因:
# - 端口冲突 (8080 或 8088 已被占用)
# - 内存不足 (< 4GB)
# - Temporal 初始化超时 (等待更长时间)
```

### 问题 2: Waterflow 连接不上 Temporal
```bash
# 检查 Temporal 健康状态
docker inspect waterflow-temporal --format='{{.State.Health.Status}}'
# 预期: healthy

# 检查网络连通
docker exec waterflow-server nc -zv temporal 7233
# 预期: succeeded

# 查看 Waterflow 日志
docker compose logs waterflow | grep -i temporal
```

### 问题 3: 工作流提交失败
```bash
# 检查 Agent 是否在线
docker compose ps | grep agent
# 预期: Up 状态

# 检查 Agent 日志
docker compose logs waterflow-agent-linux-1

# 验证 Task Queue 配置
curl http://localhost:8080/v1/nodes
# 确认 Agent 的 Task Queue 与工作流 runs-on 匹配
```

### 问题 4: 端口冲突
```bash
# 检查端口占用 (Linux/MacOS)
lsof -i :8080
lsof -i :8088

# Windows (PowerShell)
# netstat -ano | findstr :8080

# 解决方案 1: 停止占用端口的程序
# 解决方案 2: 修改 .env 中的端口配置
# WATERFLOW_SERVER_PORT=9090
# TEMPORAL_UI_PORT=9088

# 重启服务
docker compose down && docker compose up -d
```

### 问题 5: Server API 从宿主机无法访问 (IPv6 监听问题)
```bash
# 症状: docker exec 容器内 curl 正常,宿主机 curl localhost:8080 失败

# 检查 Server 监听地址
docker exec waterflow-server netstat -tlnp | grep 8080
# 如果看到 ":::8080" (仅IPv6),而非 "0.0.0.0:8080" (IPv4)

# 解决方案 1: 使用 IPv6 地址访问
curl http://[::1]:8080/health

# 解决方案 2: 确保 WATERFLOW_SERVER_HOST=0.0.0.0 (推荐)
# 检查 .env 配置或 docker-compose.yaml 环境变量
# 确保设置: WATERFLOW_SERVER_HOST=0.0.0.0

# 重启服务
docker compose restart waterflow
```

### 问题 6: Temporal UI 8088 端口被占用
```bash
# 症状: Temporal UI 容器启动但无法访问

# 检查 UI 日志
docker logs waterflow-temporal-ui

# 如果看到端口冲突 ("bind: address already in use")
# Temporal UI 监听 8080,但尝试映射到宿主机 8088
# 需要修改 docker-compose.yaml 中的端口映射

# 临时解决: 修改 .env
TEMPORAL_UI_PORT=9088

# 重启 UI
docker compose restart temporal-ui
```

### 问题 7: Agent 健康检查失败但工作正常
```bash
# 症状: docker ps 显示 agent unhealthy,但 Agent 日志正常工作

# 检查 metrics 端点
docker exec waterflow-agent-linux-1 wget -O- http://localhost:9090/metrics

# 如果返回 "Connection refused",说明 Agent 未启动 metrics HTTP server
# 这是正常的 - Agent metrics 端点可能未实现或配置未启用

# 解决方案: 暂时接受 unhealthy 状态 (不影响工作流执行)
# 或修改 docker-compose.yaml 移除 Agent 健康检查
```

### 获取支持
```bash
# 查看完整日志
docker compose logs

# 导出日志到文件
docker compose logs > waterflow-logs.txt

# 提交 Issue 时附带:
# 1. Docker/Docker Compose 版本
# 2. 操作系统版本
# 3. docker compose ps 输出
# 4. 日志文件
```

## 数据持久化

### 卷列表和用途

**postgresql-data** - PostgreSQL 数据目录
- 包含: Temporal 工作流历史、状态数据
- 路径: /var/lib/postgresql/data
- 持久化: 必需 (删除将丢失所有工作流历史)

**agent-plugins** - Agent 插件目录
- 包含: 自定义节点插件 (.so 文件)
- 路径: /app/plugins
- 持久化: 可选 (方便添加自定义插件)

### 插件管理

```bash
# 查看默认插件
docker exec waterflow-agent-linux-1 ls -lh /app/plugins/

# 添加自定义插件到卷
# 1. 编译插件
go build -buildmode=plugin -o custom.so plugin.go

# 2. 复制到卷
docker cp custom.so waterflow-agent-linux-1:/app/plugins/

# 3. 重启 Agent
docker compose restart agent-linux-1

# 4. 验证加载
docker logs waterflow-agent-linux-1 | grep "Loaded plugin"
```

### 卷管理命令

```bash
# 查看卷
docker volume ls | grep waterflow

# 查看卷详情
docker volume inspect deployments_postgresql-data
docker volume inspect deployments_agent-plugins

# 备份卷 (PostgreSQL 数据)
docker run --rm \
  -v deployments_postgresql-data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/postgresql-backup.tar.gz -C /data .

# 恢复卷
docker run --rm \
  -v deployments_postgresql-data:/data \
  -v $(pwd):/backup \
  alpine tar xzf /backup/postgresql-backup.tar.gz -C /data

# 清理卷 (危险操作,删除所有数据)
docker compose down -v
```

## 停止和清理

### 停止服务 (保留数据)
```bash
# 停止所有服务
docker compose stop

# 重新启动
docker compose start
```

### 停止并删除容器 (保留数据)
```bash
docker compose down

# 再次启动 (数据卷保留)
docker compose up -d
```

### 完全清理 (删除数据)
```bash
# ⚠️ 警告: 这将删除所有数据 (包括工作流历史)
docker compose down -v

# 确认删除
docker volume ls | grep waterflow
# 预期: 无输出 (卷已删除)
```

### 清理镜像 (释放磁盘空间)
```bash
# 删除未使用的镜像
docker image prune -a

# 清理构建缓存
docker builder prune -a
```

## 启动性能

### 首次启动 (拉取镜像)
- **预计时间**: 5-10 分钟 (取决于网络速度)
- **下载大小**: ~500MB (PostgreSQL + Temporal + UI)
- **构建时间**: ~2 分钟 (Waterflow Server + Agent)

### 再次启动 (镜像已缓存)
- **预计时间**: 1-2 分钟
- **仅需时间**: 服务启动和健康检查

### 加速启动
1. **使用镜像缓存**: 首次运行后,镜像会缓存在本地
2. **预拉取镜像**: `docker compose pull`
3. **使用国内镜像源** (中国大陆用户):
   ```bash
   # 配置 Docker 镜像加速器 (阿里云/DaoCloud)
   # 详见: https://cr.console.aliyun.com/cn-hangzhou/instances/mirrors
   ```

## 更多文档

- **快速入门**: [../docs/quick-start.md](../docs/quick-start.md)
- **部署指南**: [../docs/deployment.md](../docs/deployment.md)
- **架构决策**: [../docs/adr/0008-temporal-as-internal-service.md](../docs/adr/0008-temporal-as-internal-service.md)
- **API 参考**: [../internal/api/README.md](../internal/api/README.md)

## 贡献

发现问题或有改进建议? 欢迎提交 Issue 或 Pull Request!

- **GitHub 仓库**: https://github.com/websoft9/waterflow
- **贡献指南**: [../CONTRIBUTING.md](../CONTRIBUTING.md)
