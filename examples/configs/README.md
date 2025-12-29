# Waterflow 配置文件模板

本目录包含 Waterflow 系统的所有配置文件模板。

## 📋 文件说明

### config.example.yaml
**用途：** Waterflow Server 配置模板

**使用方法：**
```bash
# 在项目根目录
cp examples/configs/config.example.yaml config.yaml
vim config.yaml
```

**主要配置项：**
- `server`: HTTP 服务器配置（地址、端口）
- `temporal`: Temporal 连接配置
- `logger`: 日志配置
- `metrics`: 监控指标配置

**详细说明：** [配置指南](../../docs/configuration.md)

---

### config.agent.example.yaml
**用途：** Waterflow Agent 配置模板

**使用方法：**
```bash
# 在项目根目录
cp examples/configs/config.agent.example.yaml config.agent.yaml
vim config.agent.yaml
```

**主要配置项：**
- `temporal`: Temporal 连接配置
- `agent`: Agent 标识和任务队列配置
- `plugins`: 插件目录配置
- `logger`: 日志配置

**详细说明：** [docs/guides/agent-README.md](../../docs/guides/agent-README.md)

---

### server-groups.example.yaml
**用途：** Server Groups 映射配置模板

**使用方法：**
```bash
# 在项目根目录
cp examples/configs/server-groups.example.yaml server-groups.yaml
vim server-groups.yaml
```

**主要配置项：**
- Server groups 到 task queue 的映射关系
- 用于工作流中的 `runs-on` 字段

**详细说明：** [docs/guides/server-groups.md](../../docs/guides/server-groups.md)

---

## ⚙️ 环境变量

所有配置项都可以通过环境变量覆盖，格式为：`WATERFLOW_SECTION_KEY`

**示例：**
```bash
# 覆盖 Server 端口
export WATERFLOW_SERVER_PORT=9090

# 覆盖 Temporal 地址
export WATERFLOW_TEMPORAL_HOST=temporal.example.com:7233

# 覆盖日志级别
export WATERFLOW_LOG_LEVEL=debug
```

环境变量优先级 **高于** 配置文件。

---

## 🐳 Docker 部署

在 Docker 环境中，建议使用环境变量而非配置文件：

```yaml
# docker-compose.yaml
services:
  waterflow:
    image: waterflow:latest
    environment:
      WATERFLOW_SERVER_PORT: 8080
      WATERFLOW_TEMPORAL_HOST: temporal:7233
      WATERFLOW_LOG_LEVEL: info
```

详见：[docs/deployment.md](../../docs/deployment.md)

---

## 📚 相关文档

- [配置指南](../../docs/configuration.md)
- [快速开始](../../docs/quick-start.md)
- [部署文档](../../docs/deployment.md)
- [架构文档](../../docs/architecture.md)
