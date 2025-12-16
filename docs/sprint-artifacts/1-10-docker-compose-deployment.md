# Story 1.10: Docker Compose 部署方案

Status: drafted

## Story

As a **开发者**,  
I want **通过 Docker Compose 一键部署 Waterflow + Temporal**,  
So that **快速搭建开发环境**。

## Acceptance Criteria

**Given** 安装了 Docker 和 Docker Compose  
**When** 执行 `docker-compose up`  
**Then** 启动 Temporal Server (含 PostgreSQL)  
**And** 启动 Waterflow Server 并连接到 Temporal  
**And** 所有服务健康检查通过  
**And** Waterflow API 可访问 (http://localhost:8080)  
**And** 提供 README 说明部署步骤  
**And** 部署时间 <10 分钟

## Technical Context

### Architecture Constraints

根据 [docs/architecture.md](docs/architecture.md) §5.2 Docker Compose 配置设计:

1. **服务架构**

```
┌─────────────────────────────────────────────────────────┐
│                  Docker Compose Host                    │
│                                                         │
│  ┌────────────────┐         ┌────────────────────┐     │
│  │ Waterflow      │         │ Temporal Server    │     │
│  │ Server         │────────→│                    │     │
│  │ :8080          │  gRPC   │ :7233              │     │
│  └────────────────┘         └─────────┬──────────┘     │
│         ↑                              │                │
│         │ HTTP                         │                │
│         │                              ↓                │
│         │                   ┌────────────────────┐     │
│         │                   │ PostgreSQL         │     │
│         │                   │ (Temporal DB)      │     │
│         │                   └────────────────────┘     │
│         │                                              │
│  ┌────────────────┐                                    │
│  │ Temporal UI    │                                    │
│  │ :8088          │ (可选)                              │
│  └────────────────┘                                    │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

2. **服务依赖关系**

```
PostgreSQL (DB)
    ↓
Temporal Server (Workflow Engine)
    ↓
Waterflow Server (REST API)
```

3. **端口映射**

| 服务 | 容器端口 | 主机端口 | 用途 |
|-----|---------|---------|------|
| PostgreSQL | 5432 | - | 内部数据库 (不暴露) |
| Temporal Server | 7233 | 7233 | gRPC (Waterflow 连接) |
| Temporal UI | 8088 | 8088 | Web 管理界面 (可选) |
| Waterflow Server | 8080 | 8080 | REST API |

### Dependencies

**前置 Story:**
- ✅ Story 1.1: Waterflow Server 框架搭建
  - 使用: Server 二进制/Docker 镜像
- ✅ Story 1.2: REST API 服务框架
  - 使用: HTTP Server 配置
- ✅ Story 1.4: Temporal SDK 集成
  - 使用: Temporal Client 连接配置

**后续 Story 依赖本 Story:**
- Epic 2-12 的所有 Story - 基于此部署方案进行开发测试

### Technology Stack

**Docker Compose:**

```yaml
version: '3.8'

services:
  # PostgreSQL - Temporal 数据库
  postgresql:
    image: postgres:14-alpine
    environment:
      POSTGRES_PASSWORD: temporal
      POSTGRES_USER: temporal
      POSTGRES_DB: temporal
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U temporal"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Temporal Server - 工作流引擎
  temporal:
    image: temporalio/auto-setup:1.22.4
    depends_on:
      postgresql:
        condition: service_healthy
    environment:
      - DB=postgresql
      - DB_PORT=5432
      - POSTGRES_USER=temporal
      - POSTGRES_PWD=temporal
      - POSTGRES_SEEDS=postgresql
      - DYNAMIC_CONFIG_FILE_PATH=config/dynamicconfig/development-sql.yaml
    ports:
      - "7233:7233"
    healthcheck:
      test: ["CMD", "tctl", "--address", "temporal:7233", "cluster", "health"]
      interval: 10s
      timeout: 5s
      retries: 10

  # Temporal UI (可选)
  temporal-ui:
    image: temporalio/ui:2.21.3
    depends_on:
      - temporal
    environment:
      - TEMPORAL_ADDRESS=temporal:7233
      - TEMPORAL_CORS_ORIGINS=http://localhost:3000
    ports:
      - "8088:8088"

  # Waterflow Server - REST API
  waterflow-server:
    build:
      context: .
      dockerfile: Dockerfile
    depends_on:
      temporal:
        condition: service_healthy
    environment:
      - TEMPORAL_HOST=temporal:7233
      - SERVER_PORT=8080
      - LOG_LEVEL=info
      - API_KEY=${API_KEY:-waterflow-dev-key}
    ports:
      - "8080:8080"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 5
    volumes:
      - ./config:/app/config

volumes:
  postgres_data:
    driver: local
```

**Dockerfile (Waterflow Server):**

```dockerfile
# Multi-stage build
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o waterflow-server ./cmd/server

# Final stage
FROM alpine:3.18

# Install ca-certificates and curl for healthcheck
RUN apk --no-cache add ca-certificates curl

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/waterflow-server .

# Create non-root user
RUN addgroup -S waterflow && adduser -S waterflow -G waterflow
USER waterflow

EXPOSE 8080

CMD ["./waterflow-server"]
```

**环境变量配置 (.env):**

```bash
# Temporal Configuration
TEMPORAL_HOST=temporal:7233
TEMPORAL_NAMESPACE=default

# Server Configuration
SERVER_PORT=8080
LOG_LEVEL=info

# Authentication
API_KEY=waterflow-dev-key

# Optional: Database (for future use)
# DATABASE_URL=postgres://user:pass@localhost:5432/waterflow
```

### Project Structure Updates

本 Story 在项目根目录新增:

```
/data/Waterflow/
├── docker-compose.yml           # Docker Compose 配置 (新建)
├── docker-compose.dev.yml       # 开发环境覆盖配置 (新建)
├── Dockerfile                   # Waterflow Server 镜像 (新建)
├── .env.example                 # 环境变量模板 (新建)
├── .dockerignore                # Docker 忽略文件 (新建)
├── Makefile                     # 构建和部署命令 (新建)
├── deployments/
│   ├── docker/
│   │   └── README.md            # Docker 部署文档 (新建)
│   └── kubernetes/              # (未来扩展)
│       └── README.md
└── scripts/
    ├── wait-for-it.sh           # 服务等待脚本 (新建)
    └── init-dev-env.sh          # 开发环境初始化 (新建)
```

## Tasks / Subtasks

### Task 1: 创建 Dockerfile (AC: Waterflow Server 镜像)

- [ ] 1.1 创建 `Dockerfile`
  ```dockerfile
  # Multi-stage build for minimal image size
  FROM golang:1.21-alpine AS builder
  
  LABEL maintainer="Websoft9 <help@websoft9.com>"
  
  WORKDIR /build
  
  # Install build dependencies
  RUN apk add --no-cache git
  
  # Copy go mod files
  COPY go.mod go.sum ./
  RUN go mod download
  
  # Copy source code
  COPY . .
  
  # Build binary
  RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
      -ldflags="-w -s" \
      -o waterflow-server \
      ./cmd/server
  
  # Final stage - minimal runtime image
  FROM alpine:3.18
  
  # Install runtime dependencies
  RUN apk --no-cache add \
      ca-certificates \
      curl \
      tzdata
  
  WORKDIR /app
  
  # Copy binary from builder
  COPY --from=builder /build/waterflow-server .
  
  # Create directories
  RUN mkdir -p /app/config /app/logs
  
  # Create non-root user
  RUN addgroup -S waterflow && \
      adduser -S waterflow -G waterflow && \
      chown -R waterflow:waterflow /app
  
  USER waterflow
  
  EXPOSE 8080
  
  HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
      CMD curl -f http://localhost:8080/health || exit 1
  
  CMD ["./waterflow-server"]
  ```

- [ ] 1.2 创建 `.dockerignore`
  ```
  # Git files
  .git
  .gitignore
  
  # Documentation
  *.md
  docs/
  
  # Build artifacts
  bin/
  dist/
  *.exe
  *.dll
  *.so
  *.dylib
  
  # Test files
  *_test.go
  test/
  coverage.out
  
  # Development files
  .vscode/
  .idea/
  *.swp
  *.swo
  
  # Docker files
  Dockerfile*
  docker-compose*.yml
  .dockerignore
  
  # Environment
  .env
  .env.local
  
  # Temporary files
  tmp/
  *.log
  ```

### Task 2: 创建 docker-compose.yml (AC: 一键启动所有服务)

- [ ] 2.1 创建 `docker-compose.yml`
  ```yaml
  version: '3.8'
  
  services:
    # PostgreSQL - Temporal 数据库
    postgresql:
      container_name: waterflow-postgres
      image: postgres:14-alpine
      environment:
        POSTGRES_PASSWORD: temporal
        POSTGRES_USER: temporal
        POSTGRES_DB: temporal
      volumes:
        - postgres_data:/var/lib/postgresql/data
      networks:
        - waterflow-network
      healthcheck:
        test: ["CMD-SHELL", "pg_isready -U temporal"]
        interval: 10s
        timeout: 5s
        retries: 5
      restart: unless-stopped
  
    # Temporal Server - 工作流引擎
    temporal:
      container_name: waterflow-temporal
      image: temporalio/auto-setup:1.22.4
      depends_on:
        postgresql:
          condition: service_healthy
      environment:
        - DB=postgresql
        - DB_PORT=5432
        - POSTGRES_USER=temporal
        - POSTGRES_PWD=temporal
        - POSTGRES_SEEDS=postgresql
        - DYNAMIC_CONFIG_FILE_PATH=config/dynamicconfig/development-sql.yaml
      ports:
        - "7233:7233"
      networks:
        - waterflow-network
      healthcheck:
        test: ["CMD", "tctl", "--address", "temporal:7233", "cluster", "health"]
        interval: 10s
        timeout: 5s
        retries: 10
      restart: unless-stopped
  
    # Temporal UI - Web 管理界面
    temporal-ui:
      container_name: waterflow-temporal-ui
      image: temporalio/ui:2.21.3
      depends_on:
        - temporal
      environment:
        - TEMPORAL_ADDRESS=temporal:7233
        - TEMPORAL_CORS_ORIGINS=http://localhost:3000
      ports:
        - "8088:8088"
      networks:
        - waterflow-network
      restart: unless-stopped
  
    # Waterflow Server - REST API
    waterflow-server:
      container_name: waterflow-server
      build:
        context: .
        dockerfile: Dockerfile
      depends_on:
        temporal:
          condition: service_healthy
      environment:
        - TEMPORAL_HOST=temporal:7233
        - TEMPORAL_NAMESPACE=default
        - SERVER_PORT=8080
        - LOG_LEVEL=info
        - API_KEY=${API_KEY:-waterflow-dev-key}
      ports:
        - "8080:8080"
      networks:
        - waterflow-network
      healthcheck:
        test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
        interval: 10s
        timeout: 5s
        retries: 5
      restart: unless-stopped
      volumes:
        - ./config:/app/config:ro
  
  networks:
    waterflow-network:
      driver: bridge
  
  volumes:
    postgres_data:
      driver: local
  ```

- [ ] 2.2 创建 `docker-compose.dev.yml` (开发环境覆盖)
  ```yaml
  version: '3.8'
  
  services:
    waterflow-server:
      build:
        context: .
        dockerfile: Dockerfile
        target: builder  # 使用 builder stage 进行热重载
      command: go run ./cmd/server
      environment:
        - LOG_LEVEL=debug
        - GIN_MODE=debug
      volumes:
        - .:/build  # 挂载源码支持热重载
      ports:
        - "8080:8080"
        - "2345:2345"  # Delve 调试端口
  
    postgresql:
      ports:
        - "5432:5432"  # 暴露端口用于本地连接
  
    temporal:
      environment:
        - LOG_LEVEL=debug
  ```

### Task 3: 创建 Makefile (AC: 简化命令操作)

- [ ] 3.1 创建 `Makefile`
  ```makefile
  .PHONY: help build run stop clean test dev-up dev-down logs
  
  # Variables
  DOCKER_COMPOSE := docker-compose
  DOCKER_COMPOSE_DEV := docker-compose -f docker-compose.yml -f docker-compose.dev.yml
  
  ## help: Display this help message
  help:
  	@echo "Waterflow - Docker Compose Commands"
  	@echo ""
  	@echo "Usage: make [target]"
  	@echo ""
  	@echo "Targets:"
  	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
  
  ## build: Build Waterflow Server Docker image
  build:
  	$(DOCKER_COMPOSE) build waterflow-server
  
  ## up: Start all services in production mode
  up:
  	$(DOCKER_COMPOSE) up -d
  	@echo "✅ Waterflow is starting..."
  	@echo "   Waterflow API: http://localhost:8080"
  	@echo "   Temporal UI:   http://localhost:8088"
  	@echo ""
  	@echo "Run 'make logs' to see logs"
  	@echo "Run 'make health' to check service health"
  
  ## dev-up: Start all services in development mode
  dev-up:
  	$(DOCKER_COMPOSE_DEV) up -d
  	@echo "✅ Development environment started"
  	@echo "   Source code is mounted for hot reload"
  
  ## down: Stop all services
  down:
  	$(DOCKER_COMPOSE) down
  
  ## stop: Stop all services without removing containers
  stop:
  	$(DOCKER_COMPOSE) stop
  
  ## restart: Restart all services
  restart: down up
  
  ## logs: Tail logs from all services
  logs:
  	$(DOCKER_COMPOSE) logs -f
  
  ## logs-server: Tail logs from Waterflow Server
  logs-server:
  	$(DOCKER_COMPOSE) logs -f waterflow-server
  
  ## health: Check health status of all services
  health:
  	@echo "Checking service health..."
  	@echo ""
  	@echo "PostgreSQL:"
  	@docker exec waterflow-postgres pg_isready -U temporal || echo "❌ Not ready"
  	@echo ""
  	@echo "Temporal Server:"
  	@curl -s http://localhost:7233/health || echo "❌ Not ready"
  	@echo ""
  	@echo "Waterflow Server:"
  	@curl -s http://localhost:8080/health || echo "❌ Not ready"
  
  ## clean: Remove all containers, volumes, and images
  clean: down
  	$(DOCKER_COMPOSE) down -v --remove-orphans
  	docker rmi waterflow-waterflow-server || true
  	@echo "✅ Cleaned up all resources"
  
  ## test: Run integration tests
  test:
  	@echo "Running integration tests..."
  	@./scripts/integration-test.sh
  
  ## init: Initialize development environment
  init:
  	@echo "Initializing Waterflow development environment..."
  	@cp .env.example .env
  	@echo "✅ .env file created (edit as needed)"
  	@echo ""
  	@echo "Next steps:"
  	@echo "  1. Edit .env file with your configuration"
  	@echo "  2. Run 'make up' to start services"
  
  ## ps: List running containers
  ps:
  	$(DOCKER_COMPOSE) ps
  
  ## exec-server: Open shell in Waterflow Server container
  exec-server:
  	docker exec -it waterflow-server sh
  
  ## exec-temporal: Open shell in Temporal container
  exec-temporal:
  	docker exec -it waterflow-temporal sh
  ```

### Task 4: 创建环境变量模板 (AC: 配置说明)

- [ ] 4.1 创建 `.env.example`
  ```bash
  # Waterflow Server Configuration
  
  # Temporal Connection
  TEMPORAL_HOST=temporal:7233
  TEMPORAL_NAMESPACE=default
  
  # Server Settings
  SERVER_PORT=8080
  LOG_LEVEL=info
  
  # Authentication
  # WARNING: Change this in production!
  API_KEY=waterflow-dev-key
  
  # Optional: Enable Gin debug mode (development only)
  # GIN_MODE=debug
  
  # Optional: Custom configuration file
  # CONFIG_FILE=/app/config/config.yaml
  ```

### Task 5: 创建部署文档 (AC: README 说明)

- [ ] 5.1 创建 `deployments/docker/README.md`
  ```markdown
  # Waterflow Docker Compose 部署指南
  
  本指南介绍如何使用 Docker Compose 快速部署 Waterflow 开发环境。
  
  ## 前置要求
  
  - Docker 20.10+
  - Docker Compose 2.0+
  - 可用内存 >= 4GB
  - 可用磁盘 >= 10GB
  
  ## 快速启动
  
  ### 1. 克隆仓库
  
  ```bash
  git clone https://github.com/Websoft9/Waterflow.git
  cd Waterflow
  ```
  
  ### 2. 初始化配置
  
  ```bash
  make init
  ```
  
  这会创建 `.env` 文件，根据需要编辑配置。
  
  ### 3. 启动服务
  
  ```bash
  make up
  ```
  
  首次启动需要下载镜像，大约需要 3-5 分钟。
  
  ### 4. 验证部署
  
  ```bash
  # 检查服务健康状态
  make health
  
  # 查看服务日志
  make logs
  ```
  
  **访问服务:**
  - Waterflow API: http://localhost:8080
  - Temporal UI: http://localhost:8088
  - API 健康检查: http://localhost:8080/health
  
  ## 服务架构
  
  ```
  ┌──────────────────────────────────────────┐
  │  Docker Compose 环境                     │
  │                                          │
  │  ┌────────────┐      ┌───────────────┐  │
  │  │ Waterflow  │─────→│ Temporal      │  │
  │  │ Server     │ gRPC │ Server        │  │
  │  │ :8080      │      │ :7233         │  │
  │  └────────────┘      └───────┬───────┘  │
  │                              │          │
  │                              ↓          │
  │                      ┌───────────────┐  │
  │                      │ PostgreSQL    │  │
  │                      │ :5432         │  │
  │                      └───────────────┘  │
  │                                          │
  │  ┌────────────┐                         │
  │  │ Temporal   │                         │
  │  │ UI :8088   │                         │
  │  └────────────┘                         │
  └──────────────────────────────────────────┘
  ```
  
  ## 常用命令
  
  ```bash
  # 启动服务
  make up
  
  # 停止服务
  make down
  
  # 查看日志
  make logs
  
  # 仅查看 Waterflow Server 日志
  make logs-server
  
  # 检查服务健康
  make health
  
  # 重启服务
  make restart
  
  # 清理所有数据 (包括数据库)
  make clean
  ```
  
  ## 开发模式
  
  开发模式支持代码热重载:
  
  ```bash
  # 启动开发环境
  make dev-up
  
  # 修改代码会自动重新编译
  # PostgreSQL 端口暴露到主机 :5432
  ```
  
  ## 测试 API
  
  ### 1. 健康检查
  
  ```bash
  curl http://localhost:8080/health
  ```
  
  预期响应:
  ```json
  {
    "status": "healthy",
    "temporal": {
      "connected": true,
      "namespace": "default"
    }
  }
  ```
  
  ### 2. 提交工作流
  
  ```bash
  curl -X POST http://localhost:8080/v1/workflows \
    -H "Content-Type: application/json" \
    -H "X-API-Key: waterflow-dev-key" \
    -d '{
      "workflow": "name: Test\non: push\njobs:\n  build:\n    runs-on: linux\n    steps:\n      - name: Hello\n        uses: run@v1"
    }'
  ```
  
  ### 3. 查询工作流状态
  
  ```bash
  curl http://localhost:8080/v1/workflows/{workflow_id}
  ```
  
  ## 故障排查
  
  ### 服务启动失败
  
  ```bash
  # 查看详细日志
  docker-compose logs waterflow-server
  
  # 检查 Temporal 连接
  docker exec waterflow-server curl temporal:7233
  ```
  
  ### 端口冲突
  
  如果端口 8080 或 7233 已被占用,修改 `docker-compose.yml`:
  
  ```yaml
  services:
    waterflow-server:
      ports:
        - "8081:8080"  # 改为其他端口
  ```
  
  ### 清理并重新开始
  
  ```bash
  make clean
  make up
  ```
  
  ## 生产环境部署
  
  **警告:** 默认配置仅适用于开发环境,生产部署需要:
  
  1. **修改默认密码**
     ```bash
     # .env 文件
     API_KEY=<强密码>
     POSTGRES_PASSWORD=<强密码>
     ```
  
  2. **启用 HTTPS**
     - 使用 Nginx/Traefik 作为反向代理
     - 配置 SSL 证书
  
  3. **持久化数据**
     - 确保 PostgreSQL 数据卷在宿主机上
     - 定期备份数据库
  
  4. **资源限制**
     ```yaml
     services:
       waterflow-server:
         deploy:
           resources:
             limits:
               cpus: '2'
               memory: 2G
     ```
  
  5. **日志管理**
     - 配置日志轮转
     - 集成日志收集系统 (ELK/Loki)
  
  ## 下一步
  
  - 📖 阅读 [API 文档](../../docs/api.md)
  - 📖 学习 [YAML DSL 语法](../../docs/dsl.md)
  - 🚀 查看 [示例工作流](../../examples/)
  
  ## 常见问题
  
  **Q: Temporal UI 无法访问?**  
  A: 确保 8088 端口未被占用,检查 `docker-compose logs temporal-ui`
  
  **Q: Waterflow Server 连接 Temporal 失败?**  
  A: 等待 Temporal 完全启动 (约 30 秒),检查健康状态 `make health`
  
  **Q: 如何重置所有数据?**  
  A: 运行 `make clean`,这会删除所有容器和数据卷
  ```

### Task 6: 创建辅助脚本 (AC: 自动化工具)

- [ ] 6.1 创建 `scripts/wait-for-it.sh`
  ```bash
  #!/usr/bin/env bash
  # wait-for-it.sh - Wait for service to be ready
  
  set -e
  
  host="$1"
  port="$2"
  timeout="${3:-30}"
  
  echo "Waiting for $host:$port..."
  
  for i in $(seq $timeout); do
      if nc -z "$host" "$port" > /dev/null 2>&1; then
          echo "$host:$port is available"
          exit 0
      fi
      echo "Waiting... ($i/$timeout)"
      sleep 1
  done
  
  echo "Timeout waiting for $host:$port"
  exit 1
  ```

- [ ] 6.2 创建 `scripts/init-dev-env.sh`
  ```bash
  #!/usr/bin/env bash
  # Initialize development environment
  
  set -e
  
  echo "🚀 Initializing Waterflow development environment..."
  
  # Check prerequisites
  if ! command -v docker &> /dev/null; then
      echo "❌ Docker is not installed"
      exit 1
  fi
  
  if ! command -v docker-compose &> /dev/null; then
      echo "❌ Docker Compose is not installed"
      exit 1
  fi
  
  # Create .env if not exists
  if [ ! -f .env ]; then
      echo "📝 Creating .env file..."
      cp .env.example .env
      echo "✅ .env file created"
  else
      echo "⚠️  .env file already exists, skipping..."
  fi
  
  # Create necessary directories
  mkdir -p config logs
  
  echo ""
  echo "✅ Initialization complete!"
  echo ""
  echo "Next steps:"
  echo "  1. Edit .env file if needed"
  echo "  2. Run 'make up' to start services"
  echo "  3. Visit http://localhost:8080/health to verify"
  ```

- [ ] 6.3 设置脚本权限
  ```bash
  chmod +x scripts/*.sh
  ```

### Task 7: 更新项目 README (AC: 部署说明)

- [ ] 7.1 更新 `README.md` 添加快速启动部分
  ```markdown
  ## 🚀 快速开始
  
  ### 使用 Docker Compose (推荐)
  
  最快的方式体验 Waterflow:
  
  ```bash
  # 1. 克隆仓库
  git clone https://github.com/Websoft9/Waterflow.git
  cd Waterflow
  
  # 2. 启动服务
  make up
  
  # 3. 验证部署
  curl http://localhost:8080/health
  ```
  
  **访问服务:**
  - Waterflow API: http://localhost:8080
  - Temporal UI: http://localhost:8088
  
  详细部署文档请参考 [Docker 部署指南](deployments/docker/README.md)
  
  ### 手动编译
  
  ```bash
  # 安装依赖
  go mod download
  
  # 构建
  make build
  
  # 运行 (需要先启动 Temporal)
  ./bin/waterflow-server
  ```
  ```

### Task 8: 集成测试 (AC: 部署验证)

- [ ] 8.1 创建 `scripts/integration-test.sh`
  ```bash
  #!/usr/bin/env bash
  # Integration test for Docker Compose deployment
  
  set -e
  
  echo "=== Waterflow Docker Compose Integration Test ==="
  
  # Colors
  GREEN='\033[0;32m'
  RED='\033[0;31m'
  NC='\033[0m' # No Color
  
  # Test variables
  BASE_URL="http://localhost:8080"
  TEMPORAL_UI="http://localhost:8088"
  MAX_RETRIES=30
  
  # Function to check service health
  check_service() {
      local url=$1
      local name=$2
      local retries=0
      
      echo "Checking $name..."
      
      while [ $retries -lt $MAX_RETRIES ]; do
          if curl -sf "$url" > /dev/null 2>&1; then
              echo -e "${GREEN}✅ $name is healthy${NC}"
              return 0
          fi
          echo "Waiting for $name... ($((retries+1))/$MAX_RETRIES)"
          sleep 2
          retries=$((retries+1))
      done
      
      echo -e "${RED}❌ $name failed to start${NC}"
      return 1
  }
  
  # 1. Start services
  echo "Starting Docker Compose services..."
  docker-compose up -d
  
  # 2. Wait for PostgreSQL
  echo "Waiting for PostgreSQL..."
  sleep 5
  
  # 3. Check Temporal Server
  check_service "http://localhost:7233/health" "Temporal Server" || exit 1
  
  # 4. Check Waterflow Server
  check_service "$BASE_URL/health" "Waterflow Server" || exit 1
  
  # 5. Test API endpoints
  echo ""
  echo "Testing API endpoints..."
  
  # Health check
  HEALTH=$(curl -s $BASE_URL/health)
  if echo "$HEALTH" | grep -q "healthy"; then
      echo -e "${GREEN}✅ Health check passed${NC}"
  else
      echo -e "${RED}❌ Health check failed${NC}"
      echo "Response: $HEALTH"
      exit 1
  fi
  
  # Validate endpoint (without workflow submission test)
  echo "Testing validate endpoint..."
  VALIDATE_RESPONSE=$(curl -s -X POST $BASE_URL/v1/validate \
      -H "Content-Type: application/json" \
      -d '{"workflow":"name: Test\non: push\njobs:\n  build:\n    runs-on: linux\n    steps:\n      - name: Test"}')
  
  if echo "$VALIDATE_RESPONSE" | grep -q "valid"; then
      echo -e "${GREEN}✅ Validate endpoint working${NC}"
  else
      echo -e "${RED}❌ Validate endpoint failed${NC}"
      echo "Response: $VALIDATE_RESPONSE"
  fi
  
  # 6. Check Temporal UI
  if curl -sf $TEMPORAL_UI > /dev/null 2>&1; then
      echo -e "${GREEN}✅ Temporal UI accessible${NC}"
  else
      echo -e "${RED}⚠️  Temporal UI not accessible (non-critical)${NC}"
  fi
  
  # 7. Check logs for errors
  echo ""
  echo "Checking for errors in logs..."
  ERRORS=$(docker-compose logs waterflow-server 2>&1 | grep -i "error" || true)
  if [ -z "$ERRORS" ]; then
      echo -e "${GREEN}✅ No errors in Waterflow Server logs${NC}"
  else
      echo -e "${RED}⚠️  Found errors in logs:${NC}"
      echo "$ERRORS"
  fi
  
  # Summary
  echo ""
  echo "=== Test Summary ==="
  echo -e "${GREEN}✅ All core services are running${NC}"
  echo ""
  echo "Services:"
  echo "  - Waterflow API: $BASE_URL"
  echo "  - Temporal UI:   $TEMPORAL_UI"
  echo ""
  echo "Run 'make logs' to view logs"
  echo "Run 'make down' to stop services"
  ```

- [ ] 8.2 设置测试脚本权限
  ```bash
  chmod +x scripts/integration-test.sh
  ```

### Task 9: 性能优化和最佳实践

- [ ] 9.1 更新 Dockerfile 添加多阶段构建优化
  ```dockerfile
  # 已在 Task 1.1 中实现
  # 添加构建缓存优化注释
  
  # Tips for faster builds:
  # 1. 使用 BuildKit: DOCKER_BUILDKIT=1 docker build .
  # 2. 缓存 go mod: go.mod 和 go.sum 单独 COPY
  # 3. 最小化层数: 合并 RUN 命令
  # 4. .dockerignore: 排除不必要文件
  ```

- [ ] 9.2 添加 Docker Compose 资源限制 (可选)
  ```yaml
  # docker-compose.yml 添加资源限制
  services:
    waterflow-server:
      deploy:
        resources:
          limits:
            cpus: '2'
            memory: 2G
          reservations:
            cpus: '0.5'
            memory: 512M
  ```

## Dev Notes

### Critical Implementation Guidelines

**1. 健康检查顺序 - 确保依赖服务先启动**

```yaml
# ✅ 正确: 使用 depends_on 和 healthcheck
services:
  waterflow-server:
    depends_on:
      temporal:
        condition: service_healthy

# ❌ 错误: 不等待依赖服务
services:
  waterflow-server:
    depends_on:
      - temporal  # 仅等待容器创建,不等待服务就绪
```

**2. 环境变量优先级 - .env 文件 vs 命令行**

```bash
# ✅ 正确: .env 文件作为默认值
# docker-compose.yml
environment:
  - API_KEY=${API_KEY:-default-key}

# 命令行覆盖
API_KEY=custom docker-compose up

# ❌ 错误: 硬编码敏感信息
environment:
  - API_KEY=hardcoded-secret
```

**3. 数据持久化 - 使用命名卷**

```yaml
# ✅ 正确: 命名卷持久化数据
volumes:
  postgres_data:
    driver: local

# ❌ 错误: 匿名卷,重启后数据丢失
volumes:
  - /var/lib/postgresql/data
```

**4. 网络隔离 - 自定义网络**

```yaml
# ✅ 正确: 自定义网络隔离服务
networks:
  waterflow-network:
    driver: bridge

# ❌ 错误: 使用默认网络,可能与其他容器冲突
```

**5. 镜像构建优化 - 分层缓存**

```dockerfile
# ✅ 正确: 先复制依赖文件
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build

# ❌ 错误: 一次复制所有文件
COPY . .
RUN go mod download && go build  # 代码改动导致重新下载依赖
```

**6. 容器日志管理 - 防止磁盘占满**

```yaml
# ✅ 正确: 限制日志大小
logging:
  driver: "json-file"
  options:
    max-size: "10m"
    max-file: "3"

# ❌ 错误: 无限制日志
# (默认行为,可能占满磁盘)
```

### Integration with Previous Stories

**与 Story 1.1 Server 框架集成:**

```dockerfile
# Dockerfile 构建 Story 1.1 创建的 cmd/server
RUN go build -o waterflow-server ./cmd/server
```

**与 Story 1.2 REST API 集成:**

```yaml
# docker-compose.yml 暴露 API 端口
ports:
  - "8080:8080"

# 健康检查使用 /health 端点
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
```

**与 Story 1.4 Temporal 集成:**

```yaml
# docker-compose.yml 配置 Temporal 连接
environment:
  - TEMPORAL_HOST=temporal:7233

# 确保 Temporal 先启动
depends_on:
  temporal:
    condition: service_healthy
```

**为 Epic 2-12 准备:**

```yaml
# 未来可扩展 Agent 服务
services:
  waterflow-agent:
    image: waterflow/agent:latest
    environment:
      - TEMPORAL_HOST=temporal:7233
      - TASK_QUEUES=linux-amd64
```

### Testing Strategy

**本地测试:**

```bash
# 1. 构建并启动
make up

# 2. 等待服务就绪
make health

# 3. 运行集成测试
make test

# 4. 查看日志
make logs

# 5. 清理
make clean
```

**CI/CD 测试:**

```yaml
# .github/workflows/docker-test.yml
name: Docker Compose Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Start services
        run: make up
      
      - name: Run integration tests
        run: make test
      
      - name: Stop services
        run: make down
```

### Performance Considerations

**1. 镜像大小优化**

```dockerfile
# 使用 alpine 基础镜像
FROM alpine:3.18  # ~5MB

# vs
FROM ubuntu:22.04  # ~77MB

# 最终镜像大小:
# - Builder stage: ~500MB (仅构建时)
# - Final image: ~15MB (Waterflow binary + alpine)
```

**2. 启动时间优化**

```yaml
# 并行启动不依赖的服务
# PostgreSQL 和 Temporal UI 可并行
# Temporal 依赖 PostgreSQL
# Waterflow 依赖 Temporal

# 预期启动时间:
# - PostgreSQL: 5-10s
# - Temporal: 20-30s
# - Waterflow: 5s
# 总计: ~35-45s
```

**3. 资源使用**

```
服务资源占用 (典型):
- PostgreSQL: 50MB RAM
- Temporal:   200MB RAM
- Waterflow:  30MB RAM
─────────────────────────
总计:         ~280MB RAM
```

### Production Deployment Checklist

**安全加固:**

- [ ] 修改所有默认密码
- [ ] 启用 TLS/HTTPS
- [ ] 限制网络访问 (防火墙)
- [ ] 使用 secrets 管理敏感信息
- [ ] 定期更新镜像

**高可用性:**

- [ ] 数据库备份策略
- [ ] 多副本部署 (Kubernetes)
- [ ] 负载均衡
- [ ] 健康检查和自动重启
- [ ] 日志聚合和监控

**性能调优:**

- [ ] 调整资源限制
- [ ] 启用 PostgreSQL 连接池
- [ ] Temporal Worker 并发配置
- [ ] Nginx 反向代理缓存

### References

**架构设计:**
- [docs/architecture.md §5](docs/architecture.md) - Deployment View

**技术文档:**
- [Docker Compose 文档](https://docs.docker.com/compose/)
- [Temporal Docker 部署](https://docs.temporal.io/self-hosted-guide/docker-compose)
- [Multi-stage Builds](https://docs.docker.com/build/building/multi-stage/)

**项目上下文:**
- [docs/epics.md Epic 1](docs/epics.md) - 所有前置 Story 已完成

### Dependency Graph

```
所有 Story 1.1-1.9 ──┐
                     ↓
Story 1.10 (Docker Compose 部署) ← 当前 Story
    ↓
    └→ Epic 2-12 所有开发工作 - 基于此环境进行开发测试
```

## Dev Agent Record

### Context Reference

**Source Documents Analyzed:**
1. [docs/epics.md](docs/epics.md) (lines 428-445) - Story 1.10 需求定义
2. [docs/architecture.md](docs/architecture.md) (§5.1, §5.2) - Docker Compose 配置设计
3. [README.md](README.md) - 项目概览

**Previous Stories:**
- Story 1.1-1.9: 全部 drafted (Epic 1 完整实现链)

### Agent Model Used

Claude 3.5 Sonnet (BMM Scrum Master Agent - Bob)

### Estimated Effort

**开发时间:** 6-8 小时  
**复杂度:** 中等

**时间分解:**
- Dockerfile 编写: 1 小时
- docker-compose.yml 配置: 1.5 小时
- Makefile 创建: 1 小时
- 部署文档编写: 1.5 小时
- 辅助脚本: 1 小时
- 集成测试: 1 小时
- 调试和优化: 1 小时

**技能要求:**
- Docker 多阶段构建
- Docker Compose 编排
- Shell 脚本
- 服务健康检查
- 网络和数据卷管理

### Debug Log References

<!-- Will be populated during implementation -->

### Completion Notes List

<!-- Developer 填写完成时的笔记 -->

### File List

**预期创建文件清单:**

```
新建文件 (~10 个):
├── Dockerfile                              # Waterflow Server 镜像
├── .dockerignore                           # Docker 忽略文件
├── docker-compose.yml                      # 生产环境配置
├── docker-compose.dev.yml                  # 开发环境覆盖
├── .env.example                            # 环境变量模板
├── Makefile                                # 构建和部署命令
├── deployments/docker/README.md            # 部署文档
├── scripts/
│   ├── wait-for-it.sh                      # 服务等待脚本
│   ├── init-dev-env.sh                     # 环境初始化
│   └── integration-test.sh                 # 集成测试

修改文件 (~1 个):
└── README.md                                # 添加快速启动部分
```

**关键代码片段:**

**docker-compose.yml (核心):**
```yaml
version: '3.8'

services:
  postgresql:
    image: postgres:14-alpine
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U temporal"]

  temporal:
    image: temporalio/auto-setup:1.22.4
    depends_on:
      postgresql:
        condition: service_healthy

  waterflow-server:
    build: .
    depends_on:
      temporal:
        condition: service_healthy
    ports:
      - "8080:8080"
```

**Dockerfile (多阶段构建):**
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o waterflow-server ./cmd/server

FROM alpine:3.18
COPY --from=builder /build/waterflow-server .
CMD ["./waterflow-server"]
```

**Makefile (便捷命令):**
```makefile
up:
	docker-compose up -d

health:
	curl http://localhost:8080/health

clean:
	docker-compose down -v
```

---

**Story Ready for Development** ✅

开发者可基于此 Story,实现 Waterflow 的 Docker Compose 一键部署方案。
本 Story 完成后,用户可在 10 分钟内搭建完整的开发环境。

**Epic 1 完成!** 🎉
所有 10 个 Story 已全部 drafted,总工时估算: 69-91 小时。
