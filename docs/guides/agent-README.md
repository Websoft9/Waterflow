# Waterflow Agent 文档中心

## 📖 快速导航

| 阶段 | 文档 | 说明 |
|------|------|------|
| **开始** | [快速开始](./agent-quickstart.md) | 5 分钟启动第一个 Agent |
| **配置** | [配置详解](../sprint-artifacts/2-10-agent-configuration-guide.md) | 完整配置文件说明 |
| **部署** | [最佳实践](./agent-best-practices.md) | 生产环境部署建议 |
| **故障排查** | [故障排查](./agent-troubleshooting.md) | 常见问题解决方案 |
| **监控** | [监控集成](./agent-monitoring.md) | Prometheus/Grafana 集成 |

## 🚀 5 分钟快速开始

```bash
# 1. 启动 Agent (Docker)
docker run -d \
  --name waterflow-agent \
  -e TEMPORAL_SERVER_URL=temporal:7233 \
  -e TASK_QUEUES=linux-amd64 \
  waterflow/agent:latest

# 2. 验证运行
docker logs waterflow-agent

# 3. 查询状态
curl http://localhost:8080/v1/agents
```

## 📋 常见部署场景

### 场景 1: 本地开发
→ [Docker Compose 快速开始](./agent-quickstart.md#方式-2-docker-compose-推荐)

### 场景 2: 生产环境 (Docker Compose)
→ [Docker Compose 部署](../sprint-artifacts/2-9-agent-docker-image.md#ac2-docker-compose-集成)

### 场景 3: 裸机服务器
→ [systemd 部署](../sprint-artifacts/2-10-agent-configuration-guide.md#ac2-systemd-服务单元文件)

## 🔧 配置示例

### 基本配置
```yaml
agent:
  task_queues: ["linux-amd64"]
temporal:
  host: "localhost:7233"
```

### 高级配置
→ [配置最佳实践](./agent-best-practices.md#1-task-queue-规划)

## 🛠️ 常见任务

### 部署新 Agent
```bash
# 1. 下载配置模板
cp config.agent.example.yaml config.yaml

# 2. 修改配置
nano config.yaml

# 3. 启动 Agent
./agent --config config.yaml
```

### 监控 Agent 状态
```bash
# 查看所有 Agent
curl http://localhost:8080/v1/agents

# 查看特定 Task Queue 的 Agent
curl http://localhost:8080/v1/agents?task_queue=linux-amd64

# 查看 Metrics
curl http://agent:9090/metrics
```

### 故障排查
```bash
# 查看日志
docker logs -f waterflow-agent

# 检查连接
docker exec waterflow-agent ping temporal

# 验证配置
./agent --config config.yaml --validate
```

## 📚 完整文档列表

### 入门指南
- [快速开始](./agent-quickstart.md) - 5 分钟部署第一个 Agent
- [配置详解](../sprint-artifacts/2-10-agent-configuration-guide.md) - 所有配置项说明

### 部署指南
- [systemd 部署](../sprint-artifacts/2-10-agent-configuration-guide.md#ac2-systemd-服务单元文件) - 裸机服务器部署
- [Docker 部署](./agent-quickstart.md#方式-1-单个-agent-容器-最简单) - 容器化部署

###运维指南
- [最佳实践](./agent-best-practices.md) - 生产环境配置建议
- [故障排查](./agent-troubleshooting.md) - 常见问题诊断和解决
- [监控集成](./agent-monitoring.md) - Prometheus/Grafana/Datadog 集成
- [Agent 升级](./agent-best-practices.md#11-agent-升级指南) - 零停机升级流程

### 参考文档
- [配置文件模板](../../config.agent.example.yaml) - 完整配置示例
- [systemd Service](../../deployments/systemd/waterflow-agent.service) - systemd 单元文件
- [安装脚本](../../scripts/install-agent.sh) - 自动化安装
- [验证脚本](../../scripts/verify-agent.sh) - 部署验证

## ❓ 遇到问题?

### 1. 查看故障排查手册
→ [Agent 故障排查手册](./agent-troubleshooting.md)

### 2. 搜索已知问题
→ [GitHub Issues](https://github.com/Websoft9/Waterflow/issues)

### 3. 提交新问题
使用以下模板提交 Issue:

````markdown
**环境信息:**
- Agent 版本: v1.0.0
- 部署方式: Docker / systemd
- 操作系统: Ubuntu 22.04

**问题描述:**
简要描述遇到的问题

**复现步骤:**
1. 步骤 1
2. 步骤 2
3. ...

**日志输出:**
```
粘贴相关日志
```

**配置文件:**
```yaml
粘贴相关配置
```
````

## 🔗 相关资源

- [Waterflow 项目主页](https://github.com/Websoft9/Waterflow)
- [Temporal 官方文档](https://docs.temporal.io/)
- [Docker 官方文档](https://docs.docker.com/)

## 📝 文档贡献

发现文档错误或有改进建议? 欢迎提交 Pull Request!

```bash
# 1. Fork 仓库
# 2. 创建分支
git checkout -b docs/improve-agent-guide

# 3. 修改文档
nano docs/guides/agent-quickstart.md

# 4. 提交更改
git commit -m "docs: improve agent quickstart guide"

# 5. 推送并创建 PR
git push origin docs/improve-agent-guide
```

---

**上次更新:** 2025-12-26  
**文档版本:** v1.0.0
