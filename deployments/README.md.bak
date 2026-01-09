# deployments

部署配置文件目录。

## 架构说明

Waterflow 采用**独立容器 + 内部网络**架构（详见 [ADR-0008](../docs/adr/0008-temporal-as-internal-service.md)）：

```
┌─────────────────── Docker Network: waterflow-network ────────────────┐
│                                                                      │
│  PostgreSQL  →  Temporal (内部服务)  →  Waterflow  →  Agent         │
│    :5432          :7233 (不对外)        :8080 (对外)    Worker       │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

**关键特性：**
- ✅ Temporal 作为独立容器运行（符合容器最佳实践）
- ✅ Temporal 端口**不对外暴露**（对用户透明）
- ✅ 所有通信通过 Docker 内部网络完成
- ✅ 用户只需访问 Waterflow 的 8080 端口

## 文件列表

- **docker-compose.yaml** - Docker Compose 编排配置
  - PostgreSQL 数据库（内部服务）
  - Temporal Server（内部服务，不暴露端口）
  - Waterflow Server（对外暴露 :8080）
  - Agent Worker（示例）
- **.env.example** - 环境变量配置示例

## 环境

- **development** - 本地开发环境
- **staging** - 预发布环境 (未来)
- **production** - 生产环境 (未来)

## 快速启动

```bash
cd deployments

# 启动所有服务
docker compose up -d

# 验证服务状态（所有服务应为 healthy）
docker ps

# 查看日志
docker compose logs -f

# 访问 Waterflow API
curl http://localhost:8080/health
```

### 验证 Temporal 内部服务

```bash
# 验证 Temporal 未暴露到宿主机（应无输出）
docker port waterflow-temporal

# 验证内部网络连通性（应成功）
docker exec waterflow-agent-linux-1 nc -zv temporal 7233
```

## 详细文档

- **部署指南：** [../docs/deployment.md](../docs/deployment.md)
- **架构决策：** [../docs/adr/0008-temporal-as-internal-service.md](../docs/adr/0008-temporal-as-internal-service.md)
- **快速入门：** [../docs/quick-start.md](../docs/quick-start.md)
