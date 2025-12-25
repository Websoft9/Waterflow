# Story 2.9: Agent Docker 镜像

Status: ready-for-dev

## Story

As a **运维工程师**,  
I want **Agent 提供标准 Docker 镜像**,  
so that **快速部署和扩容 Agent 节点**。

## Context

这是 **Epic 2: 分布式 Agent 系统**的第九个 Story。前面的 Stories 已实现 Agent Worker、健康监控等核心功能,现在需要将 Agent 打包为 Docker 镜像,支持容器化部署。

**前置依赖:**
- Story 2.1 (Agent Worker) - Agent 核心逻辑已完成
- Story 2.7 (健康监控) - Agent 心跳机制已实现
- Story 1.10 (Docker Compose) - Docker 部署经验已积累

**业务价值:**
- 🚀 **快速部署** - `docker run` 一行命令启动 Agent
- 📦 **统一环境** - 消除"本地可以运行,生产环境不行"问题
- 🔄 **版本控制** - 镜像标签管理多个 Agent 版本
- ☁️ **云原生** - 支持 Kubernetes、Docker Swarm 等容器编排

**技术目标:**
- 镜像大小 < 100MB (多阶段构建)
- 启动时间 < 5 秒
- 支持环境变量配置
- 支持 Plugin 挂载

## Acceptance Criteria

### AC1: 多阶段 Dockerfile 构建

**Given** Agent 源代码  
**When** 执行 `docker build`  
**Then** 生成小于 100MB 的镜像

**Dockerfile** (`build/Dockerfile.agent`):
```dockerfile
# ========================================
# Stage 1: Build Stage
# ========================================
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/

# Build agent binary
# CGO_ENABLED=0 for static binary
# -ldflags "-s -w" to reduce size
ARG VERSION=dev
ARG COMMIT=unknown
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT}" \
    -o agent \
    ./cmd/agent

# ========================================
# Stage 2: Runtime Stage
# ========================================
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    && update-ca-certificates

# Create non-root user
RUN addgroup -g 1000 waterflow && \
    adduser -D -u 1000 -G waterflow waterflow

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/agent /app/agent

# Copy default config template
COPY config.agent.example.yaml /app/config.example.yaml

# Create directories
RUN mkdir -p /app/plugins /app/logs /app/config && \
    chown -R waterflow:waterflow /app

# Switch to non-root user
USER waterflow

# Expose metrics port (optional)
EXPOSE 9090

# Health check (using process check, no extra dependencies)
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD pgrep -x agent > /dev/null || exit 1

# Environment variables
ENV TEMPORAL_SERVER_URL="temporal:7233" \
    TASK_QUEUES="" \
    LOG_LEVEL="info" \
    METRICS_PORT="9090"

# Entry point
ENTRYPOINT ["/app/agent"]
CMD ["--config", "/app/config/config.yaml"]
```

**构建优化说明:**
1. **多阶段构建** - 分离编译和运行时环境
2. **静态链接** - `CGO_ENABLED=0` 避免 libc 依赖
3. **Strip 符号表** - `-ldflags "-s -w"` 减少 30% 体积
4. **Alpine 基础镜像** - 仅 ~5MB
5. **非 root 用户** - 提升安全性

**预期镜像大小:**
```
golang:1.21-alpine (builder): ~300MB (不计入最终镜像)
alpine:3.19 (runtime):        ~5MB
agent binary:                 ~15MB
ca-certificates:              ~1MB
总计:                         ~21MB ✅
```

### AC2: Docker Compose 集成

**Given** docker-compose.yaml 已配置 Server  
**When** 添加 Agent 服务  
**Then** 支持快速启动多个 Agent

**扩展 docker-compose.yaml** (`deployments/docker-compose.yaml`):
```yaml
services:
  # ========================================
  # Waterflow Server
  # ========================================
  server:
    build:
      context: ..
      dockerfile: Dockerfile
    image: waterflow/server:latest
    container_name: waterflow-server
    ports:
      - "8080:8080"
    environment:
      - TEMPORAL_SERVER_URL=temporal:7233
      - LOG_LEVEL=info
    depends_on:
      temporal:
        condition: service_healthy
    networks:
      - waterflow-net

  # ========================================
  # Waterflow Agents
  # ========================================
  agent-linux-1:
    build:
      context: ..
      dockerfile: build/Dockerfile.agent
    image: waterflow/agent:latest
    container_name: waterflow-agent-linux-1
    environment:
      - TEMPORAL_SERVER_URL=temporal:7233
      - TASK_QUEUES=linux-amd64,linux-common
      - LOG_LEVEL=info
      - AGENT_ID=agent-linux-1
      - SERVER_URL=http://server:8080  # 用于心跳上报
    volumes:
      - ./agent-config.yaml:/app/config/config.yaml:ro
      - agent-plugins:/app/plugins:ro
    depends_on:
      - server
    networks:
      - waterflow-net
    restart: unless-stopped

  agent-linux-2:
    image: waterflow/agent:latest
    container_name: waterflow-agent-linux-2
    environment:
      - TEMPORAL_SERVER_URL=temporal:7233
      - TASK_QUEUES=linux-amd64,linux-common
      - LOG_LEVEL=info
      - AGENT_ID=agent-linux-2
      - SERVER_URL=http://server:8080
    volumes:
      - ./agent-config.yaml:/app/config/config.yaml:ro
      - agent-plugins:/app/plugins:ro
    depends_on:
      - server
    networks:
      - waterflow-net
    restart: unless-stopped

  agent-web:
    image: waterflow/agent:latest
    container_name: waterflow-agent-web
    environment:
      - TEMPORAL_SERVER_URL=temporal:7233
      - TASK_QUEUES=web-servers
      - LOG_LEVEL=info
      - AGENT_ID=agent-web-1
      - SERVER_URL=http://server:8080
    volumes:
      - ./agent-config.yaml:/app/config/config.yaml:ro
      - agent-plugins:/app/plugins:ro
    depends_on:
      - server
    networks:
      - waterflow-net
    restart: unless-stopped

  # ========================================
  # Temporal Dependencies (unchanged)
  # ========================================
  temporal:
    # ... (existing config)

  postgresql:
    # ... (existing config)

volumes:
  agent-plugins:
    driver: local

networks:
  waterflow-net:
    driver: bridge
```

**使用示例:**
```bash
# 启动全部服务 (Server + 3 Agents)
docker-compose up -d

# 仅启动 Server
docker-compose up -d server temporal postgresql

# 扩容 Agent (运行 5 个 linux-amd64 Worker)
docker-compose up -d --scale agent-linux-1=5

# 查看 Agent 日志
docker-compose logs -f agent-linux-1

# 停止某个 Agent
docker-compose stop agent-web
```

### AC3: 环境变量配置支持

**Given** Agent 镜像已构建  
**When** 通过环境变量配置参数  
**Then** 无需挂载配置文件即可启动

**Agent 配置加载逻辑** (`cmd/agent/main.go`):
```go
func loadConfig() (*config.AgentConfig, error) {
	cfg := &config.AgentConfig{}
	
	// 1. Load from config file if exists
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "/app/config/config.yaml"
	}
	
	if _, err := os.Stat(configPath); err == nil {
		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		if err := viper.Unmarshal(cfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}
	
	// 2. Override with environment variables
	// TEMPORAL_SERVER_URL
	if url := os.Getenv("TEMPORAL_SERVER_URL"); url != "" {
		cfg.Temporal.ServerURL = url
	}
	
	// TASK_QUEUES (comma-separated)
	if queues := os.Getenv("TASK_QUEUES"); queues != "" {
		cfg.Agent.TaskQueues = strings.Split(queues, ",")
	}
	
	// AGENT_ID
	if agentID := os.Getenv("AGENT_ID"); agentID != "" {
		cfg.Agent.ID = agentID
	} else {
		// Auto-generate ID
		hostname, _ := os.Hostname()
		cfg.Agent.ID = fmt.Sprintf("agent-%s-%d", hostname, time.Now().Unix())
	}
	
	// SERVER_URL
	if serverURL := os.Getenv("SERVER_URL"); serverURL != "" {
		cfg.Agent.ServerURL = serverURL
	}
	
	// LOG_LEVEL
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		cfg.Logger.Level = logLevel
	}
	
	// METRICS_PORT
	if port := os.Getenv("METRICS_PORT"); port != "" {
		cfg.Metrics.Port = port
	}
	
	// 3. Validate required fields
	if cfg.Temporal.ServerURL == "" {
		return nil, errors.New("TEMPORAL_SERVER_URL is required")
	}
	
	if len(cfg.Agent.TaskQueues) == 0 {
		return nil, errors.New("TASK_QUEUES is required")
	}
	
	return cfg, nil
}
```

**纯环境变量启动:**
```bash
docker run -d \
  --name my-agent \
  -e TEMPORAL_SERVER_URL=temporal.example.com:7233 \
  -e TASK_QUEUES=linux-amd64,gpu-a100 \
  -e AGENT_ID=my-custom-agent \
  -e LOG_LEVEL=debug \
  waterflow/agent:latest
```

**环境变量覆盖优先级:**
```
环境变量 > 配置文件 > 默认值
```

### AC4: Plugin 挂载支持

**Given** 用户有自定义 Plugin (.so 文件)  
**When** 挂载到 `/app/plugins` 目录  
**Then** Agent 自动加载 Plugin

**Plugin 加载逻辑** (`internal/agent/worker.go`):
```go
func (w *Worker) loadPlugins() error {
	pluginDir := "/app/plugins"
	
	// Check if plugin directory exists
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		w.logger.Info("Plugin directory not found, skipping plugin loading")
		return nil
	}
	
	// Scan .so files
	files, err := filepath.Glob(filepath.Join(pluginDir, "*.so"))
	if err != nil {
		return fmt.Errorf("failed to scan plugins: %w", err)
	}
	
	w.logger.Info("Loading plugins", zap.Int("count", len(files)))
	
	for _, file := range files {
		if err := w.loadPlugin(file); err != nil {
			w.logger.Warn("Failed to load plugin",
				zap.String("file", file),
				zap.Error(err),
			)
		} else {
			w.logger.Info("Plugin loaded successfully", zap.String("file", file))
		}
	}
	
	return nil
}
```

**Docker 挂载 Plugin:**
```bash
# 方式1: 挂载单个 Plugin
docker run -d \
  -v /path/to/my-plugin.so:/app/plugins/my-plugin.so:ro \
  -e TASK_QUEUES=custom-queue \
  waterflow/agent:latest

# 方式2: 挂载整个 Plugin 目录
docker run -d \
  -v /opt/waterflow/plugins:/app/plugins:ro \
  -e TASK_QUEUES=linux-amd64 \
  waterflow/agent:latest

# 方式3: 使用 Docker Volume
docker volume create waterflow-plugins
docker run -d \
  -v waterflow-plugins:/app/plugins:ro \
  waterflow/agent:latest
```

### AC5: 镜像版本管理和发布

**Given** Agent 代码已更新  
**When** 执行 CI/CD 流程  
**Then** 自动构建并推送镜像到 Registry

**Makefile** (`Makefile`):
```makefile
# Image configuration
DOCKER_REGISTRY ?= docker.io
DOCKER_REPO ?= waterflow
IMAGE_NAME_SERVER = $(DOCKER_REGISTRY)/$(DOCKER_REPO)/server
IMAGE_NAME_AGENT = $(DOCKER_REGISTRY)/$(DOCKER_REPO)/agent

# Version
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build tags
TAG_VERSION = $(VERSION)
TAG_LATEST = latest

# ========================================
# Agent Docker Image
# ========================================

.PHONY: docker-agent
docker-agent:
	@echo "Building Agent Docker image..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		-f build/Dockerfile.agent \
		-t $(IMAGE_NAME_AGENT):$(TAG_VERSION) \
		-t $(IMAGE_NAME_AGENT):$(TAG_LATEST) \
		.
	@echo "Agent image built: $(IMAGE_NAME_AGENT):$(TAG_VERSION)"

.PHONY: docker-agent-push
docker-agent-push: docker-agent
	@echo "Pushing Agent image..."
	docker push $(IMAGE_NAME_AGENT):$(TAG_VERSION)
	docker push $(IMAGE_NAME_AGENT):$(TAG_LATEST)
	@echo "Agent image pushed"

.PHONY: docker-agent-run
docker-agent-run:
	docker run --rm -it \
		-e TEMPORAL_SERVER_URL=host.docker.internal:7233 \
		-e TASK_QUEUES=linux-amd64 \
		-e LOG_LEVEL=debug \
		$(IMAGE_NAME_AGENT):$(TAG_LATEST)

# ========================================
# All Images
# ========================================

.PHONY: docker-all
docker-all: docker-server docker-agent

.PHONY: docker-push
docker-push: docker-server-push docker-agent-push
```

**使用示例:**
```bash
# 构建 Agent 镜像 (自动打标签 latest 和版本号)
make docker-agent

# 构建并推送到 Registry
make docker-agent-push

# 指定版本号构建
VERSION=v1.2.0 make docker-agent

# 快速测试 Agent 镜像
make docker-agent-run
```

**镜像标签策略:**
- `latest` - 最新开发版本
- `v1.2.0` - 语义化版本号 (release)
- `v1.2.0-rc1` - 候选版本 (pre-release)
- `dev-abc123` - 开发分支 (commit SHA)

### AC6: Kubernetes 部署支持

**Given** Kubernetes 集群  
**When** 部署 Agent Deployment  
**Then** 支持自动扩缩容

**Kubernetes Deployment** (`deployments/k8s/agent-deployment.yaml`):
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: waterflow-agent-linux
  namespace: waterflow
  labels:
    app: waterflow-agent
    queue: linux-amd64
spec:
  replicas: 3  # 初始 3 个副本
  selector:
    matchLabels:
      app: waterflow-agent
      queue: linux-amd64
  template:
    metadata:
      labels:
        app: waterflow-agent
        queue: linux-amd64
    spec:
      containers:
      - name: agent
        image: waterflow/agent:latest
        imagePullPolicy: Always
        env:
        - name: TEMPORAL_SERVER_URL
          value: "temporal.waterflow.svc.cluster.local:7233"
        - name: TASK_QUEUES
          value: "linux-amd64,linux-common"
        - name: LOG_LEVEL
          value: "info"
        - name: AGENT_ID
          valueFrom:
            fieldRef:
              fieldPath: metadata.name  # Pod 名称作为 Agent ID
        - name: SERVER_URL
          value: "http://waterflow-server.waterflow.svc.cluster.local:8080"
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
        livenessProbe:
          httpGet:
            path: /health
            port: 9090
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 9090
          initialDelaySeconds: 5
          periodSeconds: 10
        volumeMounts:
        - name: plugins
          mountPath: /app/plugins
          readOnly: true
      volumes:
      - name: plugins
        configMap:
          name: waterflow-plugins
          optional: true
      restartPolicy: Always

---
# HorizontalPodAutoscaler for auto-scaling
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: waterflow-agent-linux-hpa
  namespace: waterflow
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: waterflow-agent-linux
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Pods
        value: 1
        periodSeconds: 60
```

**部署命令:**
```bash
# 创建 namespace
kubectl create namespace waterflow

# 部署 Agent
kubectl apply -f deployments/k8s/agent-deployment.yaml

# 查看 Agent Pods
kubectl get pods -n waterflow -l app=waterflow-agent

# 查看 HPA 状态
kubectl get hpa -n waterflow

# 手动扩容
kubectl scale deployment waterflow-agent-linux --replicas=5 -n waterflow
```

### AC7: 镜像安全扫描

**Given** Docker 镜像已构建  
**When** 执行安全扫描  
**Then** 符合安全基线标准

**安全基线:**
- ✅ CRITICAL 漏洞 = 0
- ✅ HIGH 漏洞 ≤ 3 (有文档化的例外清单)
- ⚠️ MEDIUM 漏洞 ≤ 10
- 📋 维护漏洞例外清单 (`.trivyignore`)

**集成 Trivy 扫描** (`.github/workflows/docker.yml`):
```yaml
name: Docker Build and Scan

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  build-and-scan:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Build Agent Image
      run: |
        docker build \
          -f build/Dockerfile.agent \
          -t waterflow/agent:test \
          .
    
    - name: Run Trivy vulnerability scanner
      uses: aquasecurity/trivy-action@master
      with:
        image-ref: 'waterflow/agent:test'
        format: 'sarif'
        output: 'trivy-results.sarif'
        severity: 'CRITICAL,HIGH'
    
    - name: Upload Trivy results to GitHub Security
      uses: github/codeql-action/upload-sarif@v2
      with:
        sarif_file: 'trivy-results.sarif'
    
    - name: Fail on high vulnerabilities
      run: |
        docker run --rm \
          -v /var/run/docker.sock:/var/run/docker.sock \
          aquasec/trivy:latest \
          image --exit-code 1 --severity HIGH,CRITICAL \
          waterflow/agent:test
```

**本地扫描:**
```bash
# 安装 Trivy
brew install trivy  # macOS
apt-get install trivy  # Ubuntu

# 扫描镜像
trivy image waterflow/agent:latest

# 仅显示高危漏洞
trivy image --severity HIGH,CRITICAL waterflow/agent:latest

# 使用忽略文件
trivy image --ignorefile .trivyignore waterflow/agent:latest
```

**漏洞例外清单** (`.trivyignore`):
```
# CVE-2023-xxxxx - Alpine base image issue, no fix available
# Severity: HIGH
# Reason: Does not affect our use case (network isolated)
# Review Date: 2025-12-25
CVE-2023-xxxxx

# CVE-2024-yyyyy - OpenSSL vulnerability
# Severity: MEDIUM
# Reason: Fixed in next Alpine release, low risk
# Review Date: 2025-12-25
CVE-2024-yyyyy
```

## Developer Context

### 镜像构建流程

```
┌──────────────┐
│ Source Code  │
└──────┬───────┘
       ↓
┌──────────────────────────────────┐
│ Stage 1: Builder (golang:1.21)   │
│  - go mod download               │
│  - go build (static binary)      │
│  - Strip symbols (-ldflags)      │
└──────┬───────────────────────────┘
       ↓
┌──────────────────────────────────┐
│ Stage 2: Runtime (alpine:3.19)   │
│  - Copy binary only              │
│  - Add ca-certificates           │
│  - Create non-root user          │
└──────┬───────────────────────────┘
       ↓
┌──────────────┐
│ Final Image  │
│  Size: ~21MB │
└──────────────┘
```

### 环境变量清单

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `TEMPORAL_SERVER_URL` | ✅ | 无 | Temporal Server 地址 |
| `TASK_QUEUES` | ✅ | 无 | 任务队列 (逗号分隔) |
| `AGENT_ID` | ❌ | 自动生成 | Agent 唯一 ID |
| `SERVER_URL` | ❌ | 无 | Waterflow Server URL (心跳上报) |
| `LOG_LEVEL` | ❌ | `info` | 日志级别 (debug/info/warn/error) |
| `METRICS_PORT` | ❌ | `9090` | Metrics 端口 |
| `CONFIG_PATH` | ❌ | `/app/config/config.yaml` | 配置文件路径 |

### 部署场景对比

| 场景 | 部署方式 | 扩容方式 | 适用场景 |
|------|---------|---------|---------|
| **开发环境** | `docker run` | 手动启动多个容器 | 本地测试 |
| **测试环境** | Docker Compose | `--scale agent-linux=5` | 集成测试 |
| **生产环境** | Kubernetes | HPA 自动扩缩容 | 生产负载 |
| **边缘节点** | systemd + Docker | 手动管理 | 物理服务器 |

## Dev Notes

### 实现优先级

**必须实现 (MVP):**
- ✅ 多阶段 Dockerfile
- ✅ 环境变量配置
- ✅ Docker Compose 集成
- ✅ Makefile 构建脚本

**推荐实现:**
- ✅ Kubernetes Deployment 示例
- ✅ Plugin 挂载支持
- ✅ HPA 自动扩缩容

**可选实现 (Post-MVP):**
- 镜像安全扫描 (Trivy)
- 多架构镜像 (arm64, amd64)
- Helm Chart 封装

### 测试策略

```bash
# 1. 本地构建测试
make docker-agent

# 2. 验证镜像大小
docker images waterflow/agent:latest
# REPOSITORY         TAG       SIZE
# waterflow/agent    latest    21MB ✅

# 3. 测试环境变量配置
docker run --rm waterflow/agent:latest \
  -e TEMPORAL_SERVER_URL=test:7233 \
  -e TASK_QUEUES=test-queue \
  --help

# 4. 测试 Plugin 挂载
echo "test" > /tmp/test-plugin.so
docker run --rm \
  -v /tmp/test-plugin.so:/app/plugins/test.so:ro \
  waterflow/agent:latest \
  ls -la /app/plugins

# 5. Docker Compose 端到端测试
cd deployments
docker-compose up -d
docker-compose ps
docker-compose logs agent-linux-1
```

### 常见问题

**Q: 为什么使用 Alpine 而不是 Debian?**  
A: Alpine 镜像仅 5MB,大幅减少镜像大小。由于使用静态编译 (`CGO_ENABLED=0`),无需 libc 依赖。

**Q: 如何调试容器内的 Agent?**  
```bash
# 进入容器
docker exec -it waterflow-agent-linux-1 sh

# 查看进程
ps aux

# 查看日志
tail -f /app/logs/agent.log
```

**Q: 如何更新 Agent 版本?**  
```bash
# 拉取新镜像
docker pull waterflow/agent:v1.2.0

# 重启容器 (Docker Compose)
docker-compose up -d --no-deps agent-linux-1

# Kubernetes 滚动更新
kubectl set image deployment/waterflow-agent-linux \
  agent=waterflow/agent:v1.2.0 -n waterflow
```

## Dev Agent Record

### File List

**新增文件:**
- `build/Dockerfile.agent` (~80 行)
- `deployments/k8s/agent-deployment.yaml` (~120 行)
- `.github/workflows/docker.yml` (~60 行)

**修改文件:**
- `Makefile` (+60 行) - Docker 构建脚本
- `deployments/docker-compose.yaml` (+80 行) - Agent 服务配置
- `cmd/agent/main.go` (+100 行) - 环境变量配置加载
- `internal/agent/worker.go` (+50 行) - Plugin 加载逻辑

**总计:** ~550 新增/修改代码行
