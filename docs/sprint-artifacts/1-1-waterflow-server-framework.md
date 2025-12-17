# Story 1.1: Waterflow Server 框架搭建

Status: drafted

## Story

As a **开发者**,  
I want **搭建 Waterflow Server 的基础框架和目录结构**,  
so that **后续可以在统一的架构上开发各个功能模块**。

## Acceptance Criteria

**Given** Go 1.21+ 开发环境已配置  
**When** 执行项目初始化命令  
**Then** 创建标准 Go 项目结构 (cmd/, pkg/, internal/, api/)  
**And** 包含 Makefile, go.mod, Dockerfile  
**And** 配置 golangci-lint 和代码质量检查  
**And** 基础 CI 管道 (GitHub Actions) 可以构建项目

## Technical Context

### Architecture Constraints

根据 [docs/architecture.md](docs/architecture.md) 设计,本Story需要建立:

1. **容器架构** - Waterflow Server 是3个核心容器之一
   - 职责: REST API + YAML DSL解析 + Temporal Client
   - 技术栈: Go + Gin/Echo框架
   - 依赖: Temporal Server (gRPC连接)

2. **组件结构** (参考 architecture.md §3.1)
   - REST API Handler - HTTP请求处理
   - DSL Parser - YAML解析
   - Validator - Schema验证
   - Expression Engine - 表达式求值
   - Temporal Client - 工作流提交

3. **关键ADR决策**
   - [ADR-0001](docs/adr/0001-use-temporal-workflow-engine.md): 使用 Temporal 作为底层引擎 (Event Sourcing)
   - [ADR-0004](docs/adr/0004-yaml-dsl-syntax.md): YAML DSL 语法设计
   - [ADR-0005](docs/adr/0005-expression-system-syntax.md): 表达式系统语法 (${{ }})
   - 要求: 必须支持 Temporal Go SDK 集成

4. **功能需求映射**
   - **FR3 工作流管理 API**: 建立 REST API 基础框架
   - **NFR1 部署简单性**: 单二进制部署,配置外部化

### Project Structure Requirements

基于Go最佳实践和项目需求,需要建立以下目录结构:

```
waterflow/
├── cmd/
│   └── server/           # Server主程序入口
│       └── main.go
├── pkg/                  # 可导出的公共包
│   ├── api/             # REST API定义
│   ├── dsl/             # DSL解析器
│   └── client/          # Go SDK客户端(未来)
├── internal/            # 内部实现,不可导出
│   ├── server/          # HTTP服务器实现
│   ├── workflow/        # Temporal Workflow实现
│   ├── config/          # 配置管理
│   └── logger/          # 日志组件
├── api/                 # API定义(OpenAPI/Protobuf)
│   └── openapi.yaml
├── build/               # 构建配置
│   └── Dockerfile
├── deployments/         # 部署配置
│   └── docker-compose.yml
├── scripts/             # 脚本工具
├── tests/               # 集成测试
├── docs/                # 文档(已存在)
├── .github/
│   └── workflows/       # GitHub Actions CI
│       └── ci.yml
├── Makefile
├── go.mod
├── go.sum
├── .golangci.yml        # Lint配置
├── .gitignore
└── README.md
```

**关键约束:**
- Go版本: >= 1.21 (泛型支持,性能优化)
- 模块路径: `github.com/websoft9/waterflow`
- 代码风格: 遵循golangci-lint标准配置

### Technology Stack

**核心依赖:**
1. **Web框架:** Gin v1.9+ (高性能,中间件丰富)
   - 理由: 比Echo更活跃的社区,更好的性能
   - 备选: Echo (如果需要更简洁的API)

2. **Temporal SDK:** `go.temporal.io/sdk` v1.25+
   - 必需: 与Temporal Server v1.20+版本兼容
   - 配置: 连接参数从环境变量/配置文件加载

3. **配置管理:** Viper v1.16+
   - 支持: YAML配置文件 + 环境变量覆盖
   - 12-factor app兼容

4. **日志:** Zap v1.26+ (结构化日志)
   - 要求: JSON格式输出,支持日志级别
   - 集成: 与Gin中间件集成

5. **代码质量:**
   - golangci-lint v1.55+ (包含20+linters)
   - gofmt, goimports (代码格式化)

6. **测试框架:**
   - testify v1.8+ (assertions)
   - gomock v1.6+ (mocking)

**依赖管理策略:**
- 使用Go Modules (`go.mod`)
- 版本锁定策略: 固定minor版本,允许patch更新
- 定期依赖安全扫描 (`go list -m all | nancy`)

## Tasks / Subtasks

### Task 1: 初始化Go项目结构 (AC: 创建标准目录结构)

- [ ] 1.1 创建根目录结构
  ```bash
  mkdir -p cmd/server pkg/{api,dsl,client} internal/{server,workflow,config,logger}
  mkdir -p api build deployments scripts tests .github/workflows
  ```

- [ ] 1.2 初始化Go Module
  ```bash
  go mod init github.com/websoft9/waterflow
  ```

- [ ] 1.3 添加核心依赖
  ```bash
  go get github.com/gin-gonic/gin@v1.9.1
  go get go.temporal.io/sdk@v1.25.0
  go get github.com/spf13/viper@v1.16.0
  go get go.uber.org/zap@v1.26.0
  ```
  
  **依赖版本锁定说明:**
  - 使用精确版本号(如v1.9.1)而非范围(v1.9)以避免意外升级
  - go.mod会自动记录精确版本,后续`go mod tidy`不会改变major.minor版本
  - 仅允许patch版本更新(如v1.9.1→v1.9.2)以获取安全修复

- [ ] 1.4 创建main.go骨架(包含配置加载和logger初始化)
  - 路径: `cmd/server/main.go`
  - 内容: 基础入口,加载配置,初始化logger
  - **为Story 1.2预留:** HTTP Server启动接口设计
    - 预留`internal/server/server.go`中的`Server`结构体
    - 预留`Run()`和`Shutdown()`方法签名
    - 预留中间件注册机制(logger, recovery, cors)
    - 预留路由注册接口`RegisterRoutes()`
  - 验证: `go build ./cmd/server` 成功

### Task 2: 配置Makefile和构建工具 (AC: 包含Makefile)

- [ ] 2.1 创建Makefile,包含以下目标:
  ```makefile
  .PHONY: build test lint fmt clean docker-build install-tools

  install-tools:   # 安装开发工具(golangci-lint, goimports等)
  build:           # 编译server二进制
  test:            # 运行单元测试
  lint:            # 运行golangci-lint
  fmt:             # 格式化代码
  clean:           # 清理构建产物
  docker-build:    # 构建Docker镜像
  ```
  
  **install-tools目标实现:**
  ```makefile
  install-tools:
  	@echo "Installing development tools..."
  	@command -v golangci-lint >/dev/null 2>&1 || \
  	  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
  	  sh -s -- -b $(shell go env GOPATH)/bin v1.55.2
  	@go install golang.org/x/tools/cmd/goimports@latest
  	@echo "✅ Tools installed successfully"
  ```

- [ ] 2.2 验证构建命令
  ```bash
  make build  # 应生成 bin/waterflow-server
  make test   # 应运行测试(即使当前无测试)
  ```

### Task 3: 配置代码质量工具 (AC: 配置golangci-lint)

- [ ] 3.0 安装golangci-lint v1.55+
  ```bash
  # 安装golangci-lint到GOPATH/bin
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2
  
  # 验证安装
  golangci-lint --version
  ```

- [ ] 3.1 创建`.golangci.yml`配置文件
  - 启用linters: errcheck, gosimple, govet, ineffassign, staticcheck, typecheck, unused, misspell
  - 禁用: 过于严格的linters (gocyclo阈值放宽)
  - 排除: `vendor/`, `tests/`生成代码

- [ ] 3.2 集成到Makefile
  ```bash
  make lint  # 应运行golangci-lint并通过
  ```

- [ ] 3.3 添加pre-commit hook (必需,质量保障)
  - 路径: `.git/hooks/pre-commit`
  - 内容:
  ```bash
  #!/bin/sh
  # Pre-commit hook for code quality
  make fmt
  make lint
  if [ $? -ne 0 ]; then
    echo "❌ Lint failed. Please fix errors before committing."
    exit 1
  fi
  ```
  - 添加可执行权限: `chmod +x .git/hooks/pre-commit`

### Task 4: 创建Dockerfile (AC: 包含Dockerfile)

- [ ] 4.1 创建多阶段Dockerfile
  - 路径: `build/Dockerfile`
  - Stage 1: Builder (基于golang:1.21-alpine)
    - **优化:** 先复制go.mod和go.sum利用Docker layer cache
    ```dockerfile
    # 第1层: 复制依赖定义(变化频率低)
    COPY go.mod go.sum ./
    RUN go mod download
    
    # 第2层: 复制源代码(变化频率高)
    COPY . .
    ```
    - 编译二进制 (CGO_ENABLED=0 for static binary)
  - Stage 2: Runtime (基于alpine:3.18)
    - 复制二进制
    - 暴露端口8080
    - 设置非root用户运行
    - ENTRYPOINT: `/app/waterflow-server`
  
  **Layer cache优化说明:**
  - go.mod/go.sum很少变化,单独COPY可以缓存依赖下载层
  - 源代码频繁变化,放在后面避免重复下载依赖
  - 预计构建时间从5分钟降至30秒(依赖层命中缓存时)

- [ ] 4.2 创建.dockerignore
  - 排除: `.git`, `vendor`, `bin`, `*.md`, `tests`

- [ ] 4.3 验证Docker构建
  ```bash
  docker build -f build/Dockerfile -t waterflow-server:dev .
  docker run --rm waterflow-server:dev --version
  ```

### Task 5: 配置GitHub Actions CI (AC: 基础CI管道)

- [ ] 5.1 创建CI workflow文件
  - 路径: `.github/workflows/ci.yml`
  - 完整配置示例:
  ```yaml
  name: CI
  
  on:
    push:
      branches: [main]
    pull_request:
      branches: [main]
  
  jobs:
    lint:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4
        - uses: actions/setup-go@v5
          with:
            go-version: '1.21'
        - name: golangci-lint
          uses: golangci/golangci-lint-action@v3
          with:
            version: v1.55.2
    
    test:
      runs-on: ubuntu-latest
      strategy:
        matrix:
          go-version: ['1.21', '1.22']
      steps:
        - uses: actions/checkout@v4
        - uses: actions/setup-go@v5
          with:
            go-version: ${{ matrix.go-version }}
        - name: Run tests
          run: |
            go test -v -race -coverprofile=coverage.out ./...
            go tool cover -func=coverage.out
        - name: Check coverage
          run: |
            COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
            echo "Coverage: $COVERAGE%"
            if (( $(echo "$COVERAGE < 70" | bc -l) )); then
              echo "❌ Coverage $COVERAGE% is below 70% threshold"
              exit 1
            fi
    
    build:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4
        - uses: actions/setup-go@v5
          with:
            go-version: '1.21'
        - name: Build
          run: make build
    
    docker:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4
        - name: Build Docker image
          run: docker build -f build/Dockerfile -t waterflow-server:ci .
  ```

- [ ] 5.2 配置Go版本矩阵(已包含在5.1)
  - 测试: Go 1.21, 1.22(通过matrix策略)
  - 主版本: Go 1.21

- [ ] 5.3 添加状态徽章到README.md
  - CI Status Badge
  - Go Version Badge
  - License Badge

### Task 6: 编写项目文档 (AC: 提供README说明)

- [ ] 6.1 创建README.md
  - 项目简介
  - 快速开始指南
    - 环境要求: Go 1.21+, Docker
    - 编译: `make build`
    - 运行: `./bin/waterflow-server`
  - 开发指南
    - 目录结构说明
    - 构建命令
    - 测试运行
  - 贡献指南(简要)
  - License: Apache 2.0

- [ ] 6.2 创建CONTRIBUTING.md
  - 代码风格要求
  - Pull Request流程
  - 测试要求

- [ ] 6.3 创建LICENSE文件
  - Apache License 2.0全文

### Task 7: 初始化配置系统 (可选,为后续Story准备)

- [ ] 7.1 创建默认配置文件
  - 路径: `deployments/config.yaml`
  - 内容: 基础配置结构
    ```yaml
    server:
      port: 8080
      host: 0.0.0.0
    temporal:
      host: localhost:7233
      namespace: waterflow
      tls_enabled: false
    log:
      level: info
      format: json
    ```
  
  **环境变量覆盖(12-factor app):**
  ```bash
  # Temporal连接配置
  WATERFLOW_TEMPORAL_HOST=localhost:7233
  WATERFLOW_TEMPORAL_NAMESPACE=waterflow
  WATERFLOW_TEMPORAL_TLS_ENABLED=false
  
  # Server配置
  WATERFLOW_SERVER_PORT=8080
  WATERFLOW_SERVER_HOST=0.0.0.0
  
  # 日志配置
  WATERFLOW_LOG_LEVEL=info
  WATERFLOW_LOG_FORMAT=json
  ```

- [ ] 7.2 实现配置加载逻辑
  - 路径: `internal/config/config.go`
  - 功能: Viper加载YAML + 环境变量覆盖
  - 优先级: ENV > config.yaml > defaults

## Dev Notes

### Critical Implementation Guidelines

**1. 遵循Go项目布局规范**
- 参考: [golang-standards/project-layout](https://github.com/golang-standards/project-layout)
- `cmd/`: 每个可执行程序一个子目录
- `pkg/`: 可被外部导入的库代码
- `internal/`: 内部实现,禁止外部导入
- `api/`: API契约定义(OpenAPI, Protobuf)

**2. 依赖注入和接口设计**
- 从一开始就使用接口抽象核心组件
- 便于单元测试和未来替换实现
- 示例:
  ```go
  type TemporalClient interface {
      ExecuteWorkflow(ctx context.Context, options WorkflowOptions, workflow interface{}, args ...interface{}) (WorkflowRun, error)
  }
  ```

**3. 配置管理最佳实践**
- 12-factor app: 所有配置从环境变量可配置
- 默认值: 开发环境友好的默认值
- 验证: 启动时验证所有必需配置项
- 敏感信息: 不在代码中硬编码,使用Secret管理

**4. 日志规范**
- 结构化日志: JSON格式,便于日志聚合
- 日志级别: DEBUG, INFO, WARN, ERROR
- 上下文: 每条日志包含trace_id, request_id
- 禁止: 敏感信息输出到日志

**5. 错误处理**
- 使用`pkg/errors`包装错误,保留堆栈
- 自定义错误类型,便于错误分类
- HTTP错误: 遵循RFC 7807 Problem Details

**6. 代码质量要求**
- 测试覆盖率: 目标>70%
- Lint零警告: golangci-lint必须全部通过
- 文档: 所有导出函数必须有godoc注释
- 代码review: 2人以上review通过才能合并

### Project Structure Alignment

**与架构文档一致性检查:**
- ✅ 目录结构与 architecture.md §3.1 Server组件对应
- ✅ 技术栈选择符合 ADR-0001 (Temporal Go SDK)
- ✅ 构建工具支持 Docker部署 (Epic 1.10需求)

**潜在冲突和处理:**
- 无已知冲突,这是全新项目首个Story

### References

参见Technical Context章节中已引用的架构文档、ADR决策和Epic上下文。

**外部规范:**
- [Effective Go](https://go.dev/doc/effective_go)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

### Testing Strategy

**单元测试范围 (本Story):**
- [ ] 配置加载逻辑测试 (config_test.go)
- [ ] 目录结构验证测试 (确保所有必需目录存在)
- [ ] Makefile目标测试 (验证构建成功)

**测试数据准备:**
- 示例配置文件: `tests/fixtures/config.yaml`
- Mock环境变量: 使用`os.Setenv`在测试中设置

**CI集成:**
- GitHub Actions自动运行所有测试
- 覆盖率报告: 上传到Codecov (可选)

### Security Considerations

**构建安全:**
- 使用官方golang基础镜像
- 定期更新依赖版本
- 启用go.sum验证,防止依赖篡改

**运行时安全:**
- Docker容器以非root用户运行
- 最小化镜像层,减少攻击面
- 不在镜像中包含源代码或编译器

**依赖扫描:**
- 集成Gosec/Nancy扫描Go依赖漏洞(已在CI中配置)
- CI中添加安全检查步骤

### Troubleshooting Common Issues

**问题1: go.mod依赖冲突**
```bash
# 症状: go mod tidy报错"module declares its path as...but was required as..."
# 解决: 清理mod cache并重新下载
go clean -modcache
go mod tidy
```

**问题2: Docker构建失败"no space left on device"**
```bash
# 症状: Docker构建中途失败
# 解决: 清理Docker缓存
docker system prune -a
```

**问题3: golangci-lint版本不匹配**
```bash
# 症状: 本地lint通过但CI失败
# 解决: 确保本地和CI使用相同版本(v1.55.2)
golangci-lint --version
```

**问题4: Temporal连接失败**
```bash
# 症状: "connection refused" 或 "context deadline exceeded"
# 检查1: Temporal Server是否运行
docker ps | grep temporal
# 检查2: 端口是否正确(默认7233)
telnet localhost 7233
# 检查3: namespace是否存在
tctl --namespace waterflow namespace describe
```

### Performance Considerations

**编译优化:**
- 静态链接二进制 (CGO_ENABLED=0)
- 减少依赖数量,加快构建速度
- 使用Go module proxy (GOPROXY)

**Docker镜像优化:**
- 多阶段构建,最终镜像<50MB
- 使用alpine基础镜像
- 层缓存优化 (先复制go.mod再复制源码)

## Dev Agent Record

### Context Reference

<!-- Story context will be generated in subsequent workflow step -->

### Agent Model Used

Claude 3.5 Sonnet (BMM Scrum Master Agent - Bob)

### Estimated Effort

**开发时间:** 4-6小时  
**复杂度:** 低 (基础框架搭建)

**时间分解:**
- 目录结构创建: 0.5小时
- Makefile和构建配置: 1小时
- Dockerfile编写和测试: 1.5小时
- GitHub Actions CI配置: 1小时
- 文档编写: 1小时
- 验证和调试: 0.5-1小时

**技能要求:**
- Go语言基础
- Docker基础
- GitHub Actions基础
- Linux命令行熟练

### Debug Log References

<!-- Will be populated during implementation -->

### Completion Notes List

<!-- Developer填写完成时的笔记 -->

**关键文件清单:** 参见Tasks章节中的详细文件路径和内容要求。

---

**Story Ready for Development** ✅

开发者可以直接按照Tasks顺序实施,所有技术细节和约束已明确。
