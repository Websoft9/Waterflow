# Story 1.1: Waterflow Server 框架搭建

Status: ready-for-dev

## Story

As a **开发者**,  
I want **搭建 Waterflow Server 的基础框架和配置系统**,  
so that **后续可以在统一的架构上开发各个功能模块**。

## Context

这是 Waterflow 项目的第一个 Story,建立整个系统的基础框架。此 Story 专注于创建一个生产就绪的 Go 服务骨架,包含配置管理、日志系统和基础项目结构,为后续所有功能模块提供统一的开发基础。

**Epic 背景:** Epic 1 的目标是构建核心工作流引擎基础,使开发者能够部署 Waterflow Server 并通过 Temporal Event Sourcing 实现工作流状态 100% 持久化。

**业务价值:** 建立标准化的项目结构和工具链,确保代码质量,简化后续功能开发,支持快速迭代。

## Acceptance Criteria

### AC1: 标准 Go 项目结构
**Given** Go 1.21+ 开发环境已配置  
**When** 执行项目初始化命令  
**Then** 创建标准 Go 项目结构:
```
waterflow/
├── cmd/
│   └── server/          # Server 入口
│       └── main.go
├── pkg/                 # 可导出的公共库
│   ├── config/          # 配置管理
│   └── logger/          # 日志系统
├── internal/            # 内部私有代码
│   ├── server/          # Server 实现
│   └── ...
├── api/                 # API 定义 (OpenAPI, proto)
├── scripts/             # 构建和部署脚本
├── deployments/         # 部署配置 (Docker, k8s)
│   └── docker-compose.yml
├── test/                # 额外测试文件
├── docs/                # 项目文档
├── .github/
│   └── workflows/       # GitHub Actions CI
│       └── ci.yml
├── go.mod
├── go.sum
├── Makefile
├── .gitignore
├── .golangci.yml        # Lint 配置
├── Dockerfile
└── README.md
```

**And** 目录结构遵循 [Standard Go Project Layout](https://github.com/golang-standards/project-layout)  
**And** 所有目录包含 README.md 说明用途

### AC2: 构建和质量工具
**Given** 项目结构已创建  
**When** 配置开发工具  
**Then** 创建 `Makefile` 包含以下目标:
- `make build` - 编译 server 二进制 (注入版本信息)
- `make test` - 运行所有测试
- `make coverage` - 生成测试覆盖率报告
- `make lint` - 运行代码检查
- `make fmt` - 格式化代码
- `make run` - 本地运行 server
- `make docker-build` - 构建 Docker 镜像
- `make clean` - 清理构建产物

**And** 版本信息注入 (支持 `/version` 端点):
```makefile
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

LDFLAGS := -ldflags "\
    -X main.Version=$(VERSION) \
    -X main.Commit=$(COMMIT) \
    -X main.BuildTime=$(BUILD_TIME)"

build:
	go build $(LDFLAGS) -o bin/server cmd/server/main.go
```

**And** 测试覆盖率目标:
```makefile
coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'
```

**And** 配置 `golangci-lint` 包含以下 linters:
- gofmt, goimports - 格式化
- govet - Go 静态分析
- errcheck - 错误处理检查
- staticcheck - 静态代码分析
- gosec - 安全检查
- ineffassign - 无效赋值检查
- misspell - 拼写检查
- unconvert - 不必要的类型转换
- gocyclo - 圈复杂度检查

**And** `.golangci.yml` 配置示例:
```yaml
linters:
  enable:
    - gofmt
    - goimports
    - govet
    - errcheck
    - staticcheck
    - gosec
    - ineffassign
    - misspell
    - unconvert
    - gocyclo

linters-settings:
  gocyclo:
    min-complexity: 15
  gosec:
    excludes:
      - G104  # 审计错误检查（某些情况下可接受）

run:
  timeout: 5m
  tests: true

issues:
  exclude-use-default: false
  max-issues-per-linter: 0
  max-same-issues: 0
```

**And** 创建 `.github/workflows/ci.yml` 包含:
- Go 版本: 1.21
- 步骤: checkout → setup-go → lint → test → build
- 触发条件: push, pull_request
- 测试覆盖率报告

### AC3: 配置管理系统
**Given** Server 需要支持多环境部署  
**When** Server 启动时  
**Then** 实现配置加载逻辑,支持:

**配置来源优先级 (高→低):**
1. 命令行标志 (--port, --log-level)
2. 环境变量 (WATERFLOW_* 前缀)
3. 配置文件 (--config 指定或默认 config.yaml)
4. 默认值

**环境变量映射规范:**
所有环境变量使用 `WATERFLOW_` 前缀,下划线分隔层级:
```
WATERFLOW_SERVER_HOST          → server.host
WATERFLOW_SERVER_PORT          → server.port
WATERFLOW_SERVER_READ_TIMEOUT  → server.read_timeout
WATERFLOW_SERVER_WRITE_TIMEOUT → server.write_timeout
WATERFLOW_LOG_LEVEL            → log.level
WATERFLOW_LOG_FORMAT           → log.format
WATERFLOW_LOG_OUTPUT           → log.output
WATERFLOW_TEMPORAL_HOST        → temporal.host
WATERFLOW_TEMPORAL_NAMESPACE   → temporal.namespace
WATERFLOW_TEMPORAL_TASK_QUEUE  → temporal.task_queue
```

**配置结构 (config.yaml):**
```yaml
server:
  host: "0.0.0.0"        # 监听地址
  port: 8080             # HTTP 端口
  read_timeout: 30s      # 读超时
  write_timeout: 30s     # 写超时
  shutdown_timeout: 30s  # 优雅关闭超时

log:
  level: "info"          # debug, info, warn, error
  format: "json"         # json, text
  output: "stdout"       # stdout, file path

temporal:
  host: "localhost:7233" # Temporal Server 地址
  namespace: "waterflow" # Temporal Namespace
  task_queue: "waterflow-server"
```

**配置验证:**
- 端口范围: 1-65535
- 日志级别: debug, info, warn, error
- 超时时间: ≥1s
- Temporal 地址格式: host:port
- 必需字段检查

**错误处理:**
- 配置文件不存在 → 使用默认值并警告
- 配置格式错误 → 显示详细错误信息并退出
- 配置值无效 → 显示字段路径、错误原因、有效范围

**And** 提供 `config.example.yaml` 包含所有配置项和注释

### AC4: 结构化日志系统
**Given** Server 运行时需要记录日志  
**When** Server 执行任何操作  
**Then** 实现结构化日志输出:

**日志格式 (JSON):**
```json
{
  "timestamp": "2025-12-18T10:30:45.123Z",
  "level": "info",
  "message": "server started",
  "component": "server",
  "context": {
    "port": 8080,
    "version": "0.1.0"
  }
}
```

**日志级别:**
- **DEBUG**: 详细调试信息 (函数调用、参数)
- **INFO**: 重要事件 (服务启动、配置加载)
- **WARN**: 警告信息 (配置使用默认值)
- **ERROR**: 错误信息 (启动失败、配置错误)

**日志字段:**
- timestamp: ISO 8601 格式
- level: 日志级别
- message: 日志消息
- component: 组件名称 (server, config, logger)
- context: 额外上下文 (键值对)
- error: 错误堆栈 (error 级别)

**配置控制:**
- 通过 `log.level` 配置过滤日志
- 通过 `log.format` 切换 JSON/文本格式
- 开发环境推荐: format=text, level=debug
- 生产环境推荐: format=json, level=info

**And** 日志库选择: **zap** (uber-go/zap) - 高性能、结构化

### AC5: 服务器生命周期管理
**Given** Server 配置和日志系统已实现  
**When** 实现 Server 主函数  
**Then** `cmd/server/main.go` 包含完整生命周期:

**启动流程:**
1. 解析命令行参数
2. 加载配置文件
3. 初始化日志系统
4. 验证配置
5. 创建 Server 实例
6. 注册基础 HTTP 路由 (健康检查端点)
7. 启动 HTTP 服务
8. 记录启动成功日志

**基础健康检查端点:**
```go
// GET /health - 简单存活检查
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "healthy",
    })
}

// 注册路由
http.HandleFunc("/health", s.healthHandler)
```

**Note:** 完整的 `/ready` 端点 (检查 Temporal 连接) 将在 Story 1.2 实现

**优雅关闭流程:**
1. 监听 SIGTERM/SIGINT 信号
2. 停止接收新请求
3. 等待现有请求完成 (最多 30 秒)
4. 关闭所有连接
5. 记录关闭日志
6. 退出进程

**错误处理:**
- 配置加载失败 → 记录错误并退出 (exit code 1)
- 端口占用 → 记录错误并退出 (exit code 1)
- 关闭超时 → 强制终止 (exit code 2)

**And** 提供友好的启动输出:
```
2025-12-18T10:30:45Z [INFO] Waterflow Server starting...
2025-12-18T10:30:45Z [INFO] Config loaded from: config.yaml
2025-12-18T10:30:45Z [INFO] Log level: info
2025-12-18T10:30:45Z [INFO] HTTP server listening on :8080
2025-12-18T10:30:45Z [INFO] Server started successfully
```

### AC6: Docker 支持
**Given** Server 需要容器化部署  
**When** 构建 Docker 镜像  
**Then** 创建多阶段 `Dockerfile`:

**构建阶段 (builder):**
- 基础镜像: golang:1.21-alpine
- 安装构建依赖 (make, git)
- 复制源代码
- 编译二进制 (静态链接, CGO_ENABLED=0)
- 优化: Go build cache

**运行阶段 (runtime):**
- 基础镜像: alpine:3.19
- 安装 ca-certificates (HTTPS 支持)
- 创建非 root 用户 waterflow
- 复制二进制和配置示例
- 暴露端口 8080
- 健康检查: `wget -q --spider http://localhost:8080/health`
- 入口点: `/app/server --config /etc/waterflow/config.yaml`

**镜像要求:**
- 镜像大小 < 50MB (压缩后)
- 启动时间 < 5 秒
- 非 root 用户运行
- 支持环境变量配置
- 支持多平台 (amd64/arm64) - Post-MVP 可选

**多平台构建 (可选):**
```bash
# 使用 Docker buildx 构建多平台镜像
docker buildx create --use
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t waterflow:latest \
  --push .
```

**And** 创建 `.dockerignore`:
```
.git
.github
docs
test
*.md
.gitignore
```

### AC7: 基础测试
**Given** 核心功能已实现  
**When** 编写测试  
**Then** 创建测试文件:

**配置管理测试 (pkg/config/config_test.go):**
- TestLoadConfigFromFile - 从文件加载
- TestLoadConfigFromEnv - 从环境变量加载
- TestConfigPriority - 优先级验证
- TestConfigValidation - 配置验证
- TestDefaultConfig - 默认值

**日志系统测试 (pkg/logger/logger_test.go):**
- TestLogLevels - 日志级别过滤
- TestJSONFormat - JSON 格式输出
- TestContextFields - 上下文字段

**Server 测试 (internal/server/server_test.go):**
- TestServerStart - 服务器启动
- TestServerShutdown - 优雅关闭
- TestConfigReload - 配置重载 (可选)

**测试覆盖率:** ≥80% (核心包)

**And** 测试可通过 `make test` 执行

## Tasks / Subtasks

### Task 1: 项目初始化和结构搭建 (AC1)
- [ ] 创建 GitHub 仓库 waterflow
- [ ] 初始化 Go 模块 (使用实际的 GitHub 组织/用户名):
  - 如果有 GitHub 组织: `go mod init github.com/<org>/waterflow`
  - 如果是个人仓库: `go mod init github.com/<username>/waterflow`
  - 本地开发可使用: `go mod init waterflow`
- [ ] 创建标准项目目录结构
- [ ] 编写各目录 README.md 说明用途
- [ ] 创建 .gitignore (IDE files, binaries, logs, coverage reports)

### Task 2: 构建和质量工具配置 (AC2)
- [ ] 创建 Makefile 包含所有构建目标
- [ ] 配置 .golangci.yml (启用推荐 linters)
- [ ] 创建 GitHub Actions CI 工作流
- [ ] 测试 CI 流水线 (push 触发构建)

### Task 3: 配置管理实现 (AC3)
- [ ] 定义配置结构体 (ServerConfig, LogConfig, TemporalConfig):

**完整配置结构体定义:**
```go
// pkg/config/config.go
package config

import "time"

type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Log      LogConfig      `mapstructure:"log"`
    Temporal TemporalConfig `mapstructure:"temporal"`
}

type ServerConfig struct {
    Host            string        `mapstructure:"host"`
    Port            int           `mapstructure:"port"`
    ReadTimeout     time.Duration `mapstructure:"read_timeout"`
    WriteTimeout    time.Duration `mapstructure:"write_timeout"`
    ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

type LogConfig struct {
    Level  string `mapstructure:"level"`  // debug, info, warn, error
    Format string `mapstructure:"format"` // json, text
    Output string `mapstructure:"output"` // stdout, file path
}

type TemporalConfig struct {
    Host      string `mapstructure:"host"`
    Namespace string `mapstructure:"namespace"`
    TaskQueue string `mapstructure:"task_queue"`
}
```

- [ ] 实现配置加载逻辑 (文件 + 环境变量 + 默认值)
- [ ] 实现配置验证函数
- [ ] 创建 config.example.yaml
- [ ] 编写配置管理测试

**技术选型:** [spf13/viper](https://github.com/spf13/viper) 用于配置管理
- 支持多种配置格式 (YAML, JSON, TOML)
- 支持环境变量自动映射
- 支持配置文件监听 (热重载)
- 支持默认值和配置合并

### Task 4: 日志系统实现 (AC4)
- [ ] 集成 uber-go/zap 日志库
- [ ] 实现日志初始化函数 (基于配置)
- [ ] 创建结构化日志 helper 函数
- [ ] 实现日志级别控制
- [ ] 编写日志系统测试

**日志实现示例:**
```go
// pkg/logger/logger.go
package logger

import "go.uber.org/zap"

var Log *zap.Logger

func Init(level string, format string) error {
    var cfg zap.Config
    if format == "json" {
        cfg = zap.NewProductionConfig()
    } else {
        cfg = zap.NewDevelopmentConfig()
    }
    
    cfg.Level = parseLevel(level)
    
    var err error
    Log, err = cfg.Build()
    return err
}
```

### Task 5: Server 主框架实现 (AC5)
- [ ] 实现 cmd/server/main.go 入口函数
- [ ] 实现命令行参数解析 (flag 包或 cobra)
- [ ] 实现优雅关闭处理 (signal handling)
- [ ] 创建基础 HTTP server (暂无路由)
- [ ] 添加启动和关闭日志
- [ ] 测试优雅关闭

**Server 实现示例:**
```go
// internal/server/server.go
package server

import (
    "context"
    "net/http"
    "time"
)

type Server struct {
    httpServer *http.Server
    config     *Config
}

func (s *Server) Start() error {
    s.httpServer = &http.Server{
        Addr:         fmt.Sprintf("%s:%d", s.config.Host, s.config.Port),
        ReadTimeout:  s.config.ReadTimeout,
        WriteTimeout: s.config.WriteTimeout,
    }
    
    return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
    return s.httpServer.Shutdown(ctx)
}
```

### Task 6: Docker 镜像构建 (AC6)
- [ ] 创建多阶段 Dockerfile
- [ ] 创建 .dockerignore
- [ ] 测试镜像构建: `docker build -t waterflow:dev .`
- [ ] 测试镜像运行: `docker run -p 8080:8080 waterflow:dev`
- [ ] 验证镜像大小 < 50MB
- [ ] 添加 make docker-build 目标

### Task 7: 基础测试编写 (AC7)
- [ ] 编写配置管理单元测试
- [ ] 编写日志系统单元测试
- [ ] 编写 Server 启动/关闭测试
- [ ] 运行测试: `make test`
- [ ] 验证测试覆盖率 ≥80%

### Task 8: 文档和示例
- [ ] 编写 README.md,包含以下章节:
  ```markdown
  # Waterflow
  
  ## 项目简介
  简要描述 Waterflow 是什么,核心价值
  
  ## 功能特性
  - Event Sourcing 状态管理
  - 声明式 YAML DSL
  - 分布式 Agent 执行
  - 插件化节点系统
  
  ## 快速开始
  ### 前置要求
  - Go 1.21+
  - Docker (可选)
  
  ### 安装
  ```bash
  git clone ...
  make build
  ```
  
  ### 第一个工作流
  示例代码和执行步骤
  
  ## 开发指南
  ### 克隆仓库
  ### 构建和测试
  ### 贡献指南
  
  ## 架构
  高层架构图和核心组件说明
  
  ## License
  MIT/Apache 2.0
  ```

- [ ] 创建 docs/development.md (开发环境设置)
- [ ] 创建 docs/configuration.md (配置参数说明)
- [ ] 添加代码注释 (godoc 格式)

## Technical Requirements

### Technology Stack
- **语言:** Go 1.21+
- **配置管理:** [spf13/viper](https://github.com/spf13/viper) v1.18+
- **日志库:** [uber-go/zap](https://github.com/uber-go/zap) v1.26+
  - 选择 zap 而非 Go 1.21 的 log/slog 原因:
    - **性能:** zap 提供 >1M logs/sec,比 slog 快 4-10x
    - **生产验证:** Uber 等大规模生产环境验证
    - **生态成熟:** 丰富的插件和集成
    - **结构化日志:** 原生支持结构化字段,零分配
  - 未来可考虑迁移到 slog (标准库优势)
- **HTTP 框架:** 标准库 net/http (此 Story 暂不需要 Gin/Echo)
- **测试框架:** 标准库 testing + [stretchr/testify](https://github.com/stretchr/testify) v1.8+
- **Linter:** [golangci-lint](https://github.com/golangci/golangci-lint) v1.55+

### Architecture Constraints

**Event Sourcing 架构 (ADR-0001):**
- Server 完全无状态,所有工作流状态存储在 Temporal Event History
- 此 Story 不涉及状态管理,但需要预留 Temporal 连接配置

**配置管理模式:**
- 12-Factor App 原则:配置与代码分离
- 支持环境变量注入 (容器化部署必需)
- 默认值应适合开发环境

**日志规范:**
- 结构化日志 (JSON) 便于日志聚合系统解析
- 避免敏感信息泄露 (密码、Token 自动脱敏)
- 请求 ID 追踪 (后续 Story 实现)

### Code Style and Standards

**Go 代码规范:**
- 遵循 [Effective Go](https://go.dev/doc/effective_go)
- 遵循 [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- 包名小写,不使用下划线或驼峰
- 导出函数使用驼峰命名
- 错误处理:使用 `if err != nil` 立即检查
- 返回值命名:仅在文档需要时使用

**项目结构规范:**
- `cmd/` - 可执行程序入口,保持简洁
- `pkg/` - 可被外部引用的库代码
- `internal/` - 项目私有代码,不可被外部引用
- `api/` - API 定义文件 (OpenAPI, proto)

**测试规范:**
- 测试文件命名: `xxx_test.go`
- 测试函数命名: `TestXxx`
- 使用 table-driven tests 处理多个测试用例
- Mock 外部依赖 (数据库、API)

### File Structure

```
waterflow/
├── cmd/
│   └── server/
│       └── main.go              # Server 入口,负责初始化和启动
├── pkg/
│   ├── config/
│   │   ├── config.go            # Config 结构定义和加载逻辑
│   │   └── config_test.go       # 配置测试
│   └── logger/
│       ├── logger.go            # 日志初始化和全局 logger
│       └── logger_test.go       # 日志测试
├── internal/
│   └── server/
│       ├── server.go            # Server 实现 (Start, Shutdown)
│       └── server_test.go       # Server 测试
├── .github/
│   └── workflows/
│       └── ci.yml               # GitHub Actions CI
├── deployments/
│   └── docker-compose.yml       # 开发环境 Docker Compose
├── go.mod                       # Go 模块定义
├── go.sum                       # 依赖锁定
├── Makefile                     # 构建脚本
├── Dockerfile                   # 多阶段 Docker 构建
├── .gitignore                   # Git 忽略文件
├── .golangci.yml                # Lint 配置
├── config.example.yaml          # 配置示例
└── README.md                    # 项目说明
```

### Performance Requirements

- **启动时间:** Server 启动 < 5 秒
- **内存占用:** 空闲时 < 100MB
- **配置加载:** < 100ms
- **日志性能:** zap 库提供 >1M logs/sec 性能

### Security Requirements

- **非 root 运行:** Docker 容器使用非特权用户
- **配置安全:** 敏感配置 (Temporal 连接串) 支持环境变量注入
- **日志安全:** 密码等敏感字段自动脱敏 (后续 Story 实现)

## Definition of Done

- [ ] 所有 Acceptance Criteria 验收通过
- [ ] 所有 Tasks 完成并测试通过
- [ ] 代码通过 golangci-lint 检查,无警告
- [ ] 单元测试覆盖率 ≥80%
- [ ] GitHub Actions CI 构建通过
- [ ] Docker 镜像构建成功,大小 < 50MB
- [ ] 代码已提交到 main 分支
- [ ] README.md 包含快速开始说明
- [ ] 配置示例文件完整且有注释
- [ ] Code Review 通过 (如果团队有多人)

## References

### Architecture Documents
- [Architecture - Container View](../architecture.md#2-container-view-容器视图) - Server 架构定位
- [Architecture - Component View](../architecture.md#31-server-内部组件) - Server 内部组件设计
- [ADR-0001: 使用 Temporal 作为工作流引擎](../adr/0001-use-temporal-workflow-engine.md) - Event Sourcing 架构

### PRD Requirements
- [PRD - AR1: 技术栈选型](../prd.md) - Go 1.21+, Viper, Zap, 容器化
- [PRD - NFR2: 性能](../prd.md) - Server 启动 < 5s, 内存 < 100MB

### External Resources
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout) - 项目结构参考
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) - 代码风格
- [12-Factor App](https://12factor.net/) - 配置管理原则
- [spf13/viper Documentation](https://github.com/spf13/viper) - 配置库
- [uber-go/zap Documentation](https://github.com/uber-go/zap) - 日志库

## Dev Agent Record

### Context Reference

此 Story 是项目的第一个实现,无前置 Story 依赖。

### Completion Notes

- 此 Story 完成后,项目将具备基础框架,可以开始实现 REST API (Story 1.2)
- 配置系统已预留 Temporal 连接配置,供后续集成使用
- Docker 镜像基础已建立,后续 Story 可增量添加功能
- 日志系统已标准化,所有后续组件应使用 `logger.Log` 记录日志

### File List

**预期创建的文件:**
- cmd/server/main.go (包含版本变量: Version, Commit, BuildTime)
- pkg/config/config.go (完整配置结构体)
- pkg/config/config_test.go
- pkg/logger/logger.go
- pkg/logger/logger_test.go
- internal/server/server.go (包含 /health 端点)
- internal/server/server_test.go
- .github/workflows/ci.yml
- Makefile (包含版本注入、coverage 目标)
- Dockerfile (多阶段构建,支持多平台)
- .gitignore (包含 coverage 文件)
- .golangci.yml (完整 linter 配置)
- .dockerignore
- config.example.yaml (完整环境变量注释)
- README.md (完整章节结构)
- docs/development.md
- docs/configuration.md
- go.mod
- go.sum

**构建产物 (gitignore):**
- bin/server
- coverage.out
- coverage.html

---

**Story 创建时间:** 2025-12-18  
**Story 状态:** ready-for-dev  
**预估工作量:** 3-5 天 (1 名开发者)
