---
template: single-server-deployment
version: 1.0.0
waterflow_version: ">= 0.1.0"
last_updated: 2026-01-07
author: Waterflow Team
category: deployment
difficulty: beginner
---

# 单服务器部署模板 (Single Server Deployment)

## 概述

单服务器部署模板提供了将应用程序部署到单台服务器的完整工作流,包括代码拉取、Docker 镜像构建、容器启动、健康检查和失败回滚功能。这是最适合小型应用、MVP 产品和开发/测试环境的部署方案。

**模板特点:**
- ✅ **自动化部署流程** - 从代码到运行容器的全流程自动化
- ✅ **失败自动回滚** - 健康检查失败时自动回滚到旧版本
- ✅ **支持增量更新** - 自动检测是否首次部署
- ✅ **灵活的健康检查** - 支持 HTTP 健康检查和重试策略
- ✅ **参数化配置** - 通过变量轻松定制

**工作流程:**
1. Clone/Pull Git 仓库代码
2. 构建 Docker 镜像
3. 备份旧版本容器 ID
4. 停止旧版本容器
5. 启动新版本容器
6. 执行健康检查 (支持重试)
7. 失败时自动回滚到旧版本

---

## 前置条件

在使用此模板之前,请确保满足以下条件:

### 1. Waterflow 环境

- ✅ Waterflow Server 已部署并运行
- ✅ Waterflow Agent 已安装在目标服务器
- ✅ Agent 连接到 Server 并注册 Task Queue (例如: `production-server`)

**验证 Agent 状态:**
```bash
# 在目标服务器上检查 Agent
systemctl status waterflow-agent

# 查看 Agent 日志
journalctl -u waterflow-agent -f
```

### 2. 目标服务器依赖

需要在目标服务器上预装以下工具:

| 工具 | 用途 | 安装命令 (Ubuntu/Debian) |
|------|------|-------------------------|
| **Git** | 克隆代码仓库 | `sudo apt install git` |
| **Docker** | 构建和运行容器 | [Docker 官方安装指南](https://docs.docker.com/engine/install/) |
| **curl** | 健康检查 (可选) | `sudo apt install curl` |

**验证依赖:**
```bash
# 验证 Git
git --version
# 预期输出: git version 2.x.x

# 验证 Docker
docker --version
# 预期输出: Docker version 24.x.x

# 验证 Docker 服务运行
docker ps
# 预期输出: CONTAINER ID  IMAGE  ...
```

### 3. 网络和权限

- ✅ 目标服务器可访问 Git 仓库 (公开仓库或已配置 SSH/Token)
- ✅ Docker 服务运行且 Agent 用户有权限执行 Docker 命令
- ✅ 应用端口未被占用 (默认 3000,可配置)

**配置 Docker 权限:**
```bash
# 将 Agent 运行用户添加到 docker 组
sudo usermod -aG docker waterflow

# 重启 Agent 服务
sudo systemctl restart waterflow-agent
```

**配置私有仓库访问 (可选):**
```bash
# SSH 方式
ssh-keygen -t ed25519 -C "deploy-key"
# 将 ~/.ssh/id_ed25519.pub 添加到 GitHub Deploy Keys

# HTTPS Token 方式
git config --global credential.helper store
echo "https://username:token@github.com" > ~/.git-credentials
```

### 4. 应用要求

- ✅ 应用仓库根目录包含 `Dockerfile`
- ✅ 应用提供健康检查端点 (推荐 `/health` 路径)
- ✅ Dockerfile `EXPOSE` 指令声明端口 (或使用固定端口)

**示例 Dockerfile:**
```dockerfile
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .
EXPOSE 8080
CMD ["npm", "start"]
```

**示例健康检查端点 (Node.js Express):**
```javascript
app.get('/health', (req, res) => {
  res.status(200).json({ status: 'healthy' });
});
```

---

## 参数说明

### 必需参数

必须在 `vars` 部分配置以下参数:

| 参数 | 说明 | 示例 |
|------|------|------|
| `repo_url` | Git 仓库地址 (HTTPS 或 SSH) | `https://github.com/user/app.git` |
| `app_name` | 应用名称,用于容器命名 | `my-app` |

### 可选参数 (含默认值)

以下参数有默认值,可根据需要覆盖:

| 参数 | 默认值 | 说明 | 示例 |
|------|--------|------|------|
| `app_port` | `3000` | 应用监听端口 (宿主机映射端口) | `8080` |
| `branch` | `main` | Git 分支名称 | `develop` |
| `deploy_path` | `/opt/{app_name}` | 代码部署目录 | `/opt/my-app` |
| `health_check_url` | `http://localhost:{app_port}/health` | 健康检查 URL | `http://localhost:8080/api/health` |
| `health_check_retries` | `5` | 健康检查最大重试次数 | `10` |
| `health_check_delay` | `10` | 健康检查重试延迟 (秒) | `5` |

### runs-on 参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `runs-on` | `production-server` | Agent 的 Task Queue 名称,决定工作流在哪台服务器执行 |

**修改 runs-on:**
```yaml
jobs:
  deploy:
    runs-on: my-custom-queue  # 修改为你的 Agent Task Queue
```

---

## 使用示例

### 示例 1: 部署 Node.js Express 应用

**场景:** 部署简单的 Express.js Web 应用到生产服务器

**配置文件: `my-express-app.yaml`**

```yaml
name: Deploy Express App to Production
on: push

vars:
  # 必需参数
  repo_url: "https://github.com/myorg/express-api.git"
  app_name: "express-api"
  
  # 可选参数
  app_port: "3000"
  branch: "main"
  health_check_url: "http://localhost:3000/health"

jobs:
  deploy:
    runs-on: production-server
    steps:
      # ... (模板步骤保持不变)
```

**提交工作流:**
```bash
# 使用 CLI
waterflow submit my-express-app.yaml

# 查看执行状态
waterflow status <workflow-id>

# 实时查看日志
waterflow logs -f <workflow-id>
```

**验证部署:**
```bash
# SSH 到目标服务器
ssh user@production-server

# 检查容器运行状态
docker ps | grep express-api

# 测试健康检查
curl http://localhost:3000/health
# 预期输出: {"status":"healthy"}
```

---

### 示例 2: 部署 Python Flask 应用到开发环境

**场景:** 部署 Flask API 到开发服务器,使用 `develop` 分支

**配置文件: `flask-dev-deployment.yaml`**

```yaml
name: Deploy Flask App to Dev
on: push

vars:
  # 必需参数
  repo_url: "https://github.com/myorg/flask-backend.git"
  app_name: "flask-backend-dev"
  
  # 可选参数 (自定义)
  app_port: "5000"
  branch: "develop"
  deploy_path: "/opt/flask-dev"
  health_check_url: "http://localhost:5000/api/health"
  health_check_retries: 3
  health_check_delay: 5

jobs:
  deploy:
    runs-on: dev-server  # 开发环境 Agent
    steps:
      # ... (模板步骤)
```

**Dockerfile 示例 (Flask):**

```dockerfile
FROM python:3.11-slim
WORKDIR /app

# 安装依赖
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# 复制应用代码
COPY . .

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/api/health || exit 1

# 启动应用
CMD ["gunicorn", "--bind", "0.0.0.0:8080", "app:app"]
```

**Flask 健康检查端点:**

```python
from flask import Flask, jsonify
app = Flask(__name__)

@app.route('/api/health', methods=['GET'])
def health():
    return jsonify({"status": "healthy", "service": "flask-backend"}), 200
```

**提交和验证:**

```bash
# 提交工作流
waterflow submit flask-dev-deployment.yaml

# 验证部署
curl http://dev-server:5000/api/health
```

---

### 示例 3: 部署 Go 微服务,自定义健康检查

**场景:** 部署 Go 编写的微服务,健康检查路径为 `/healthz`,需要更长的启动时间

**配置文件: `go-service-deployment.yaml`**

```yaml
name: Deploy Go Microservice
on: push

vars:
  repo_url: "https://github.com/myorg/user-service.git"
  app_name: "user-service"
  app_port: "8080"
  branch: "release/v2.0"
  health_check_url: "http://localhost:8080/healthz"
  health_check_retries: 10  # Go 服务启动较慢
  health_check_delay: 15    # 每次重试间隔 15 秒

jobs:
  deploy:
    runs-on: microservices-cluster
    steps:
      # ... (模板步骤)
```

**Go 健康检查示例:**

```go
package main

import (
    "encoding/json"
    "net/http"
)

type HealthResponse struct {
    Status  string `json:"status"`
    Version string `json:"version"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(HealthResponse{
        Status:  "healthy",
        Version: "2.0.0",
    })
}

func main() {
    http.HandleFunc("/healthz", healthHandler)
    http.ListenAndServe(":8080", nil)
}
```

---

## 定制指南

### 1. 修改构建方式

如果不使用 Docker,可以修改构建步骤:

**使用构建脚本:**

```yaml
jobs:
  deploy:
    steps:
      # 替换 Docker 构建步骤
      - name: Build with custom script
        uses: exec/script@v1
        with:
          script_path: ${{ vars.deploy_path }}/build.sh
        timeout-minutes: 10
      
      # 替换容器启动步骤
      - name: Start application process
        uses: exec/shell@v1
        with:
          command: |
            cd ${{ vars.deploy_path }}
            # 使用 PM2 启动 Node.js 应用
            pm2 restart ${{ vars.app_name }} || pm2 start server.js --name ${{ vars.app_name }}
```

**build.sh 示例:**

```bash
#!/bin/bash
set -e

echo "Building application..."
cd "$(dirname "$0")"

# Node.js 项目
npm ci
npm run build

# 或 Go 项目
# go build -o app ./cmd/server

echo "Build completed successfully"
```

### 2. 添加环境变量

在容器启动时注入环境变量:

```yaml
- name: Start new version container
  uses: exec/shell@v1
  with:
    command: |
      docker run -d --name ${{ vars.app_name }} \
        -p ${{ vars.app_port }}:8080 \
        -e DATABASE_URL="postgresql://user:pass@db:5432/mydb" \
        -e REDIS_HOST="redis.example.com" \
        -e ENV="production" \
        ${{ vars.app_name }}:${{ vars.branch }}
```

**或使用 .env 文件:**

```yaml
- name: Load environment variables
  uses: exec/shell@v1
  with:
    command: |
      docker run -d --name ${{ vars.app_name }} \
        -p ${{ vars.app_port }}:8080 \
        --env-file ${{ vars.deploy_path }}/.env \
        ${{ vars.app_name }}:${{ vars.branch }}
```

### 3. 添加数据库迁移步骤

在启动应用前执行数据库迁移:

```yaml
jobs:
  deploy:
    steps:
      # ... (前面的步骤)
      
      - name: Run database migrations
        uses: exec/shell@v1
        with:
          command: |
            cd ${{ vars.deploy_path }}
            # Node.js (Sequelize)
            npm run migrate
            
            # 或 Python (Alembic)
            # alembic upgrade head
            
            # 或 Go (golang-migrate)
            # migrate -path ./migrations -database "postgres://..." up
        timeout-minutes: 5
      
      # ... (继续启动容器步骤)
```

### 4. 添加 Slack 通知

在部署成功或失败后发送通知:

```yaml
jobs:
  deploy:
    steps:
      # ... (所有部署步骤)
      
      - name: Notify deployment success
        if: ${{ success() }}
        uses: http/request@v1
        with:
          url: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
          method: POST
          body: |
            {
              "text": "✅ Deployment succeeded: ${{ vars.app_name }} on ${{ vars.branch }}"
            }
      
      - name: Notify deployment failure
        if: ${{ failure() }}
        uses: http/request@v1
        with:
          url: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
          method: POST
          body: |
            {
              "text": "❌ Deployment failed: ${{ vars.app_name }} on ${{ vars.branch }}"
            }
```

### 5. 添加日志收集

将容器日志输出到外部日志系统:

```yaml
- name: Start new version container
  uses: exec/shell@v1
  with:
    command: |
      docker run -d --name ${{ vars.app_name }} \
        -p ${{ vars.app_port }}:8080 \
        --log-driver=syslog \
        --log-opt syslog-address=udp://logs.example.com:514 \
        --log-opt tag="${{ vars.app_name }}" \
        ${{ vars.app_name }}:${{ vars.branch }}
```

### 6. 添加资源限制

限制容器 CPU 和内存使用:

```yaml
- name: Start new version container
  uses: exec/shell@v1
  with:
    command: |
      docker run -d --name ${{ vars.app_name }} \
        -p ${{ vars.app_port }}:8080 \
        --memory="512m" \
        --memory-swap="1g" \
        --cpus="1.0" \
        ${{ vars.app_name }}:${{ vars.branch }}
```

---

## 故障排查

### 常见问题

#### 1. Git Clone 失败

**错误信息:**
```
fatal: could not read Username for 'https://github.com': terminal prompts disabled
```

**原因:** 私有仓库需要认证

**解决方案:**

**方式 A: 使用 SSH (推荐)**
```bash
# 在目标服务器生成 SSH 密钥
ssh-keygen -t ed25519 -C "deploy@server"

# 查看公钥
cat ~/.ssh/id_ed25519.pub

# 在 GitHub/GitLab 添加为 Deploy Key
# GitHub: Settings > Deploy keys > Add deploy key
```

修改 `repo_url` 为 SSH 格式:
```yaml
vars:
  repo_url: "git@github.com:myorg/myapp.git"
```

**方式 B: 使用 Personal Access Token**
```bash
# 在 GitHub 创建 Token: Settings > Developer settings > Personal access tokens

# 在仓库 URL 中嵌入 token (不安全,仅用于测试)
```

```yaml
vars:
  repo_url: "https://username:ghp_xxxxxxxxxxxxx@github.com/myorg/myapp.git"
```

---

#### 2. Docker 构建失败

**错误信息:**
```
ERROR [internal] load metadata for docker.io/library/node:18
```

**原因:** 无法拉取基础镜像 (网络问题或 Docker Hub 限流)

**解决方案:**

**方式 A: 使用镜像加速器**
```bash
# 配置 Docker 镜像加速 (中国大陆)
sudo tee /etc/docker/daemon.json <<-'EOF'
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com"
  ]
}
EOF

sudo systemctl restart docker
```

**方式 B: 提前拉取基础镜像**
```bash
docker pull node:18-alpine
docker pull python:3.11-slim
```

**方式 C: 使用本地基础镜像**

修改 Dockerfile:
```dockerfile
# 使用已存在的镜像
FROM node:18-alpine AS base
```

---

#### 3. 健康检查超时

**错误信息:**
```
Step "Health check new version" failed after 5 retries
```

**原因:** 应用启动时间过长或健康检查端点未就绪

**解决方案:**

**方式 A: 增加重试次数和延迟**
```yaml
vars:
  health_check_retries: 10
  health_check_delay: 20
```

**方式 B: 检查健康检查端点**
```bash
# 手动测试健康检查
docker exec -it my-app curl http://localhost:8080/health

# 查看应用日志
docker logs my-app
```

**方式 C: 临时禁用健康检查 (调试用)**
```yaml
- name: Health check new version
  uses: http/request@v1
  with:
    url: ${{ vars.health_check_url }}
    method: GET
  continue-on-error: true  # 添加此行,即使失败也继续
```

---

#### 4. 端口已被占用

**错误信息:**
```
docker: Error response from daemon: driver failed programming external connectivity on endpoint my-app: Bind for 0.0.0.0:3000 failed: port is already allocated.
```

**原因:** 旧容器未正确停止或其他进程占用端口

**解决方案:**

**方式 A: 强制清理旧容器**
```bash
# 停止所有同名容器
docker stop my-app || true
docker rm my-app || true

# 查找占用端口的进程
lsof -i :3000
kill -9 <PID>
```

**方式 B: 修改端口映射**
```yaml
vars:
  app_port: "3001"  # 使用不同端口
```

**方式 C: 添加清理步骤到工作流**
```yaml
- name: Force cleanup old container
  uses: exec/shell@v1
  with:
    command: |
      docker stop ${{ vars.app_name }} 2>/dev/null || true
      docker rm -f ${{ vars.app_name }} 2>/dev/null || true
```

---

#### 5. 回滚失败

**错误信息:**
```
ERROR: No backup file found, cannot rollback
```

**原因:** 首次部署失败,没有旧版本可回滚

**解决方案:**

这是预期行为。对于首次部署失败:

1. 检查部署日志定位问题:
   ```bash
   waterflow logs <workflow-id>
   ```

2. 修复问题后重新提交工作流

3. 如需保留失败的容器用于调试:
   ```yaml
   - name: Rollback on health check failure
     if: ${{ failure() }}
     uses: exec/shell@v1
     with:
       command: |
         # 注释掉清理命令,保留失败容器
         # docker stop ${{ vars.app_name }} || true
         # docker rm ${{ vars.app_name }} || true
         
         echo "Container preserved for debugging"
         echo "Run: docker logs ${{ vars.app_name }}"
         exit 1
   ```

---

#### 6. Docker 权限不足

**错误信息:**
```
permission denied while trying to connect to the Docker daemon socket
```

**原因:** Agent 运行用户无权执行 Docker 命令

**解决方案:**

```bash
# 将 waterflow 用户添加到 docker 组
sudo usermod -aG docker waterflow

# 重启 Agent 服务
sudo systemctl restart waterflow-agent

# 验证权限
sudo -u waterflow docker ps
```

---

## 最佳实践

### 1. 使用语义化版本标签

为 Docker 镜像添加版本标签:

```yaml
- name: Build Docker image with version tag
  uses: exec/shell@v1
  with:
    command: |
      VERSION=$(git describe --tags --always)
      docker build \
        -t ${{ vars.app_name }}:${{ vars.branch }} \
        -t ${{ vars.app_name }}:$VERSION \
        ${{ vars.deploy_path }}
```

### 2. 保留镜像历史

避免删除旧镜像,保留最近 N 个版本:

```yaml
- name: Cleanup old images
  uses: exec/shell@v1
  with:
    command: |
      # 保留最近 5 个镜像,删除其他
      docker images ${{ vars.app_name }} --format "{{.ID}}" | tail -n +6 | xargs -r docker rmi
```

### 3. 使用健康检查而非固定延迟

优先使用 HTTP 健康检查,而不是 `sleep`:

```yaml
# ❌ 不推荐
- name: Wait for app to start
  uses: exec/shell@v1
  with:
    command: sleep 30

# ✅ 推荐
- name: Health check
  uses: http/request@v1
  with:
    url: ${{ vars.health_check_url }}
  retry-strategy:
    max-attempts: 10
```

### 4. 分离配置和代码

使用 Docker Secrets 或 Vault 管理敏感配置:

```yaml
- name: Start container with secrets
  uses: exec/shell@v1
  with:
    command: |
      docker run -d --name ${{ vars.app_name }} \
        -p ${{ vars.app_port }}:8080 \
        --env-file /etc/waterflow/secrets/${{ vars.app_name }}.env \
        ${{ vars.app_name }}:${{ vars.branch }}
```

### 5. 添加部署前验证

在部署前验证镜像和配置:

```yaml
- name: Validate Docker image
  uses: exec/shell@v1
  with:
    command: |
      # 验证镜像是否成功构建
      docker inspect ${{ vars.app_name }}:${{ vars.branch }} > /dev/null
      
      # 验证镜像大小 (可选)
      SIZE=$(docker images ${{ vars.app_name }}:${{ vars.branch }} --format "{{.Size}}")
      echo "Image size: $SIZE"
```

### 6. 使用多阶段构建优化镜像大小

**Dockerfile 最佳实践:**

```dockerfile
# 构建阶段
FROM node:18-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

# 运行阶段
FROM node:18-alpine
WORKDIR /app
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
EXPOSE 8080
CMD ["node", "dist/server.js"]
```

### 7. 监控和日志记录

添加监控和告警:

```yaml
- name: Setup monitoring
  uses: exec/shell@v1
  with:
    command: |
      # 启用 Prometheus metrics
      docker run -d --name ${{ vars.app_name }} \
        -p ${{ vars.app_port }}:8080 \
        -p 9090:9090 \
        -e ENABLE_METRICS=true \
        ${{ vars.app_name }}:${{ vars.branch }}
```

---

## 参考资源

- **Waterflow 文档:**
  - [快速开始](../quick-start.md)
  - [YAML DSL 语法](../yaml-dsl-syntax-reference.md)
  - [表达式系统](../expression-system.md)
  
- **节点文档:**
  - [exec/shell 节点](../nodes/exec-shell.md)
  - [http/request 节点](../nodes/http-request.md)
  
- **Docker 最佳实践:**
  - [Docker 官方文档](https://docs.docker.com/)
  - [Dockerfile 最佳实践](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/)
  
- **相关模板:**
  - [多服务器健康检查](./multi-server-health-check.md)
  - [分布式栈部署](./distributed-stack-deployment.md)

---

**上一页:** [模板库首页](./README.md)  
**下一页:** [多服务器健康检查模板](./multi-server-health-check.md)
