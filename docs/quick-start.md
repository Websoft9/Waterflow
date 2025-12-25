# Waterflow 快速开始指南

最快 10 分钟部署完整的 Waterflow + Temporal 工作流编排环境。

## ⚡ 一键部署

### 前置要求

- Docker Engine 20.10+
- Docker Compose 2.0+
- 最少 2GB 可用内存

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

# 验证健康检查
curl http://localhost:8080/health
```

预期输出：`{"status":"ok"}`

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

**方法 2: Temporal UI**

访问 http://localhost:8088 查看可视化执行流程。

## 📦 服务端口

| 服务 | 端口 | 说明 |
|------|------|------|
| Waterflow API | 8080 | REST API 服务 |
| Temporal gRPC | 7233 | Temporal 客户端连接 |
| Temporal HTTP | 8233 | Temporal HTTP API |
| Temporal UI | 8088 | Web 控制台 |
| PostgreSQL | 5432 | 数据库（内部） |

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

## 🔧 自定义配置

### 修改端口

编辑 `deployments/.env`:
```bash
WATERFLOW_SERVER_PORT=9090
TEMPORAL_UI_PORT=9088
```

重启服务：
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

## 📚 下一步

- 📖 [完整部署文档](deployment.md) - 详细配置和故障排查
- 🔍 [示例工作流](../examples/README.md) - 更多 YAML 示例
- 🏗️ [架构文档](architecture.md) - 系统架构设计
- 💻 [开发指南](development.md) - 本地开发环境

## 🆘 遇到问题？

查看 [部署文档 - 常见问题排查](deployment.md#常见问题排查) 章节。
