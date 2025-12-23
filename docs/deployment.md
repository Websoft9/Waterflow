# Waterflow 部署指南

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
1. **环境变量** (`WATERFLOW_*`) - docker-compose.yaml 或 .env 设置
2. **config.yaml** - 容器内默认配置
3. **代码默认值** - viper 内置

#### 可配置项

**数据库配置（.env）：**
```bash
POSTGRES_USER=temporal          # 数据库用户名
POSTGRES_PASSWORD=temporal      # 数据库密码（生产环境请修改）
POSTGRES_DB=temporal            # 数据库名称
```

**Waterflow 配置（.env 或环境变量）：**

**Waterflow 配置（.env 或环境变量）：**
```bash
# 服务配置
WATERFLOW_SERVER_HOST=0.0.0.0
WATERFLOW_SERVER_PORT=8080

# 日志配置
WATERFLOW_LOG_LEVEL=info          # debug|info|warn|error
WATERFLOW_LOG_FORMAT=json         # json|text

# Temporal 配置
WATERFLOW_TEMPORAL_HOST=temporal:7233
WATERFLOW_TEMPORAL_NAMESPACE=default
WATERFLOW_TEMPORAL_TASKQUEUE=waterflow-server
```

**注意：** 环境变量会自动覆盖 config.yaml 中的配置。

#### 示例：修改日志级别

**方法 1：使用 .env 文件（推荐）**
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
