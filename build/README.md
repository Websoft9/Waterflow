# Waterflow Docker 构建文件

本目录包含 Waterflow 各组件的 Docker 镜像构建文件。

## 📁 文件说明

### Dockerfile.server
**用途：** Waterflow Server 的 Docker 镜像构建文件

**构建命令：**
```bash
# 在项目根目录执行
docker build -f build/Dockerfile.server -t waterflow:latest .

# 或使用 docker-compose
cd deployments
docker compose build waterflow
```

**生成镜像：**
- 镜像名称：`waterflow:latest`
- 基础镜像：`golang:1.24-alpine` (构建阶段) + `alpine:3.19` (运行阶段)
- 暴露端口：`8080` (HTTP API)

**主要功能：**
- YAML DSL 解析和验证
- Workflow 管理 API
- Temporal 客户端连接

---

### Dockerfile.agent
**用途：** Waterflow Agent 的 Docker 镜像构建文件

**构建命令：**
```bash
# 在项目根目录执行
docker build -f build/Dockerfile.agent -t waterflow/agent:latest .

# 或使用 docker-compose
cd deployments
docker compose build agent-linux-1
```

**生成镜像：**
- 镜像名称：`waterflow/agent:latest`
- 基础镜像：`golang:1.24-alpine` (构建阶段) + `alpine:3.19` (运行阶段)
- 暴露端口：`9090` (健康检查)

**主要功能：**
- Temporal Worker 注册
- 任务执行 (shell、docker 等)
- 插件系统支持

---

## 🔧 构建优化

### 多阶段构建
所有 Dockerfile 都使用多阶段构建：
- **Stage 1 (builder):** 编译 Go 二进制文件
- **Stage 2 (runtime):** 精简的运行时镜像

**优点：**
- 镜像体积小 (Server ~50MB, Agent ~50MB)
- 不包含编译工具，更安全
- 构建缓存优化，加快重复构建

### Go Proxy 配置
所有 Dockerfile 都配置了 GOPROXY 加速下载：
```dockerfile
ENV GOPROXY=https://goproxy.cn,direct
```

适用于中国大陆网络环境，可根据需要调整。

---

## 📊 镜像大小对比

| 镜像 | 大小 | 说明 |
|------|------|------|
| waterflow:latest | ~50MB | Server 镜像 (Alpine 基础) |
| waterflow/agent:latest | ~50MB | Agent 镜像 (Alpine 基础) |

---

## 🚀 快速构建所有镜像

```bash
# 使用 docker-compose 一次性构建所有镜像
cd deployments
docker compose build

# 验证镜像
docker images | grep waterflow
```

---

## 🔍 常见问题

### Q: 为什么有两个 Dockerfile？
A: Server 和 Agent 是独立组件，有不同的依赖和配置需求。

### Q: 为什么放在 build/ 目录？
A: 集中管理所有构建文件，保持项目根目录整洁。命名清晰（`Dockerfile.server` vs `Dockerfile.agent`）。

### Q: 如何自定义构建参数？
A: 使用 `--build-arg`：
```bash
docker build -f build/Dockerfile.server \
  --build-arg VERSION=1.0.0 \
  --build-arg COMMIT=$(git rev-parse HEAD) \
  -t waterflow:1.0.0 .
```

### Q: 构建失败怎么办？
A: 检查以下几点：
1. 确保在项目根目录执行构建命令
2. 检查网络连接（Go 模块下载）
3. 验证 Go 版本要求（go.mod 中定义）
4. 查看构建日志中的具体错误信息

---

## 📚 相关文档

- [部署文档](../docs/deployment.md)
- [Docker Compose 配置](../deployments/README.md)
- [配置指南](../docs/configuration.md)
- [快速开始](../docs/quick-start.md)
