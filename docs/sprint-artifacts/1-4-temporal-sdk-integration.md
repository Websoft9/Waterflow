# Story 1.4: Temporal SDK 集成

Status: drafted

## Story

As a **系统架构师**,  
I want **集成 Temporal Go SDK**,  
So that **可以利用 Temporal 的持久化执行能力**。

## Acceptance Criteria

**Given** Temporal Server 已部署并可访问  
**When** Waterflow Server 启动时  
**Then** 成功连接到 Temporal Server  
**And** 创建 Temporal Client 实例  
**And** 注册 Waterflow Namespace  
**And** 连接失败时记录错误并重试  
**And** 配置连接参数 (host, port, namespace) 可通过配置文件设置

## Technical Context

### Architecture Constraints

根据 [docs/architecture.md](docs/architecture.md) §3.1.3 Temporal Client设计:

1. **核心职责**
   - 连接到Temporal Server (gRPC)
   - 提供Workflow提交接口 (ExecuteWorkflow)
   - 查询Workflow状态 (DescribeWorkflowExecution)
   - 取消Workflow (CancelWorkflow)
   - 管理连接池和重试逻辑

2. **关键设计约束** (参考 ADR-0001)
   - Temporal作为底层工作流引擎 (Event Sourcing模式)
   - 必须使用Temporal Go SDK v1.22+
   - 连接到Temporal Frontend Service (默认7233端口)
   - 使用独立Namespace隔离 (推荐: "waterflow")

3. **非功能性需求**
   - 启动时连接失败应重试 (最多3次,间隔5秒)
   - 连接成功后定期健康检查
   - 配置参数外部化 (config.yaml或环境变量)

### Dependencies

**前置Story:**
- ✅ Story 1.1: Waterflow Server框架搭建
- ✅ Story 1.2: REST API服务框架
  - 使用: `/ready` 端点需要检查Temporal连接状态
- ✅ Story 1.3: YAML DSL解析器

**后续Story依赖本Story:**
- Story 1.5: 工作流提交API - 使用Temporal Client提交Workflow
- Story 1.6: 工作流执行引擎 - 定义Temporal Workflow实现
- Story 1.7: 状态查询API - 使用Temporal Client查询状态

**外部依赖:**
- Temporal Server (需要预先部署)
  - Frontend Service: gRPC端口7233
  - 持久化存储: PostgreSQL/MySQL/Cassandra
  - (MVP可使用Docker Compose快速部署)

### Technology Stack

**Temporal Go SDK: v1.22+**

选择理由 (参考ADR-0001):
- **官方推荐:** Temporal官方维护的Go客户端
- **功能完整:** 支持Workflow/Activity/Query/Signal
- **生产验证:** Uber等公司大规模使用
- **持续更新:** 活跃的开发和社区支持

```bash
go get go.temporal.io/sdk@latest
```

**核心SDK组件:**

1. **Client** - 主要客户端接口
   ```go
   import "go.temporal.io/sdk/client"
   
   c, err := client.Dial(client.Options{
       HostPort:  "localhost:7233",
       Namespace: "waterflow",
   })
   ```

2. **Workflow API** - 提交和管理Workflow
   ```go
   workflowOptions := client.StartWorkflowOptions{
       ID:        "workflow-123",
       TaskQueue: "linux-amd64",
   }
   
   we, err := c.ExecuteWorkflow(ctx, workflowOptions, WorkflowFunc, input)
   ```

3. **Query API** - 查询Workflow状态
   ```go
   describe, err := c.DescribeWorkflowExecution(ctx, workflowID, runID)
   status := describe.WorkflowExecutionInfo.Status
   ```

**配置管理:**

基于Story 1.1的Viper配置,扩展Temporal配置段:

```yaml
temporal:
  host_port: "localhost:7233"
  namespace: "waterflow"
  connection_timeout: 10s
  retry:
    max_attempts: 3
    initial_interval: 5s
  tls:
    enabled: false
    # cert_file: /path/to/cert.pem (生产环境)
    # key_file: /path/to/key.pem
```

**环境变量覆盖:**
- `WATERFLOW_TEMPORAL_HOST_PORT` → temporal.host_port
- `WATERFLOW_TEMPORAL_NAMESPACE` → temporal.namespace

### Temporal Architecture Overview

**Temporal Server组件:**

```
┌──────────────────────────────────────────────────────┐
│                Temporal Server                       │
│                                                      │
│  ┌──────────────┐    ┌──────────────┐              │
│  │ Frontend     │───→│ History      │              │
│  │ (gRPC 7233)  │    │ (Event Store)│              │
│  └──────────────┘    └──────────────┘              │
│         ↑                    ↓                      │
│         │            ┌──────────────┐              │
│         │            │ Matching     │              │
│         │            │ (Task Queue) │              │
│         │            └──────────────┘              │
│         │                    ↓                      │
│  ┌──────────────┐    ┌──────────────┐              │
│  │ Worker       │───→│ PostgreSQL   │              │
│  │ (Internal)   │    │ (Persistence)│              │
│  └──────────────┘    └──────────────┘              │
└──────────────────────────────────────────────────────┘
        ↑
        │ gRPC
        │
┌───────────────────┐
│ Waterflow Client  │
│ (本Story实现)      │
└───────────────────┘
```

**关键概念:**

1. **Namespace** - 逻辑隔离单元
   - 类似K8s的Namespace
   - 推荐为Waterflow创建独立namespace: "waterflow"
   - 命令: `tctl namespace register waterflow`

2. **Task Queue** - 任务路由机制
   - 对应DSL中的`runs-on`字段
   - Worker注册到特定Task Queue (如 "linux-amd64")
   - Workflow提交时指定Task Queue

3. **Workflow Execution** - 工作流实例
   - 每次提交创建一个Execution
   - WorkflowID: 用户指定或自动生成
   - RunID: Temporal自动生成的唯一标识

### Project Structure Updates

基于Story 1.1-1.3的结构,本Story新增:

```
internal/
├── temporal/
│   ├── client.go           # Temporal客户端封装 (新建)
│   ├── options.go          # Client配置选项 (新建)
│   ├── health.go           # 健康检查逻辑 (新建)
│   └── client_test.go      # 客户端单元测试 (新建)
├── config/
│   └── config.go           # 配置结构扩展 (修改 - 添加TemporalConfig)
└── server/handlers/
    └── health.go           # 修改 - 集成Temporal健康检查

cmd/server/
└── main.go                 # 修改 - 初始化Temporal Client

deployments/
├── config.yaml             # 修改 - 添加temporal配置段
└── docker-compose.yaml     # 新建 - Temporal本地部署 (可选)
```

## Tasks / Subtasks

### Task 1: 添加Temporal SDK依赖 (AC: 创建Temporal Client实例)

- [ ] 1.1 安装Temporal Go SDK
  ```bash
  cd /data/Waterflow
  go get go.temporal.io/sdk@v1.22.0
  go mod tidy
  ```

- [ ] 1.2 验证依赖安装
  ```bash
  go list -m go.temporal.io/sdk
  # 期望输出: go.temporal.io/sdk v1.22.0
  ```

- [ ] 1.3 创建测试连接程序
  ```go
  // test/temporal_connection_test.go (临时)
  package main
  
  import (
      "context"
      "log"
      "go.temporal.io/sdk/client"
  )
  
  func main() {
      c, err := client.Dial(client.Options{
          HostPort: "localhost:7233",
      })
      if err != nil {
          log.Fatal("Failed to connect:", err)
      }
      defer c.Close()
      log.Println("✅ Temporal connection successful")
  }
  ```

### Task 2: 扩展配置结构 (AC: 配置连接参数可通过配置文件设置)

- [ ] 2.1 扩展`internal/config/config.go`
  ```go
  type Config struct {
      Server   ServerConfig   `mapstructure:"server"`
      Log      LogConfig      `mapstructure:"log"`
      Temporal TemporalConfig `mapstructure:"temporal"` // 新增
  }
  
  type TemporalConfig struct {
      HostPort          string        `mapstructure:"host_port"`
      Namespace         string        `mapstructure:"namespace"`
      ConnectionTimeout time.Duration `mapstructure:"connection_timeout"`
      Retry             RetryConfig   `mapstructure:"retry"`
      TLS               TLSConfig     `mapstructure:"tls"`
  }
  
  type RetryConfig struct {
      MaxAttempts     int           `mapstructure:"max_attempts"`
      InitialInterval time.Duration `mapstructure:"initial_interval"`
  }
  
  type TLSConfig struct {
      Enabled  bool   `mapstructure:"enabled"`
      CertFile string `mapstructure:"cert_file"`
      KeyFile  string `mapstructure:"key_file"`
  }
  ```

- [ ] 2.2 添加配置验证
  ```go
  func (cfg *TemporalConfig) Validate() error {
      if cfg.HostPort == "" {
          return fmt.Errorf("temporal.host_port is required")
      }
      if cfg.Namespace == "" {
          return fmt.Errorf("temporal.namespace is required")
      }
      if cfg.Retry.MaxAttempts < 1 {
          return fmt.Errorf("temporal.retry.max_attempts must be >= 1")
      }
      return nil
  }
  ```

- [ ] 2.3 更新`deployments/config.yaml`
  ```yaml
  server:
    port: 8080
    host: 0.0.0.0
    mode: release
    shutdown_timeout: 30s
  
  log:
    level: info
    format: json
  
  temporal:
    host_port: "localhost:7233"
    namespace: "waterflow"
    connection_timeout: 10s
    retry:
      max_attempts: 3
      initial_interval: 5s
    tls:
      enabled: false
  ```

### Task 3: 实现Temporal客户端封装 (AC: 成功连接到Temporal Server)

- [ ] 3.1 创建`internal/temporal/client.go`
  ```go
  package temporal
  
  import (
      "context"
      "fmt"
      "time"
      "go.temporal.io/sdk/client"
      "go.uber.org/zap"
      "waterflow/internal/config"
  )
  
  type Client struct {
      client    client.Client
      config    *config.TemporalConfig
      logger    *zap.Logger
      connected bool
  }
  
  // New 创建Temporal客户端 (带重试)
  func New(cfg *config.TemporalConfig, logger *zap.Logger) (*Client, error) {
      if err := cfg.Validate(); err != nil {
          return nil, fmt.Errorf("invalid config: %w", err)
      }
      
      tc := &Client{
          config: cfg,
          logger: logger,
      }
      
      // 重试连接
      var lastErr error
      for attempt := 1; attempt <= cfg.Retry.MaxAttempts; attempt++ {
          logger.Info("Connecting to Temporal",
              zap.String("host_port", cfg.HostPort),
              zap.String("namespace", cfg.Namespace),
              zap.Int("attempt", attempt),
          )
          
          c, err := tc.dial()
          if err == nil {
              tc.client = c
              tc.connected = true
              logger.Info("✅ Temporal connection successful")
              return tc, nil
          }
          
          lastErr = err
          logger.Warn("Temporal connection failed",
              zap.Error(err),
              zap.Int("attempt", attempt),
          )
          
          if attempt < cfg.Retry.MaxAttempts {
              time.Sleep(cfg.Retry.InitialInterval)
          }
      }
      
      return nil, fmt.Errorf("failed to connect after %d attempts: %w",
          cfg.Retry.MaxAttempts, lastErr)
  }
  
  // dial 建立Temporal连接
  func (tc *Client) dial() (client.Client, error) {
      ctx, cancel := context.WithTimeout(context.Background(), tc.config.ConnectionTimeout)
      defer cancel()
      
      options := client.Options{
          HostPort:  tc.config.HostPort,
          Namespace: tc.config.Namespace,
          Logger:    NewTemporalLogger(tc.logger), // 集成Zap
      }
      
      // TLS配置 (可选)
      if tc.config.TLS.Enabled {
          // TODO: Story 10.x 实现TLS配置
      }
      
      return client.DialContext(ctx, options)
  }
  
  // GetClient 返回底层Temporal客户端
  func (tc *Client) GetClient() client.Client {
      return tc.client
  }
  
  // Close 关闭连接
  func (tc *Client) Close() {
      if tc.client != nil {
          tc.client.Close()
          tc.logger.Info("Temporal connection closed")
      }
  }
  
  // IsConnected 检查连接状态
  func (tc *Client) IsConnected() bool {
      return tc.connected && tc.client != nil
  }
  ```

- [ ] 3.2 创建`internal/temporal/options.go`
  ```go
  package temporal
  
  import (
      "go.temporal.io/sdk/log"
      "go.uber.org/zap"
  )
  
  // TemporalLogger 适配Zap到Temporal Logger接口
  type TemporalLogger struct {
      logger *zap.Logger
  }
  
  func NewTemporalLogger(logger *zap.Logger) log.Logger {
      return &TemporalLogger{logger: logger}
  }
  
  func (tl *TemporalLogger) Debug(msg string, keyvals ...interface{}) {
      tl.logger.Sugar().Debugw(msg, keyvals...)
  }
  
  func (tl *TemporalLogger) Info(msg string, keyvals ...interface{}) {
      tl.logger.Sugar().Infow(msg, keyvals...)
  }
  
  func (tl *TemporalLogger) Warn(msg string, keyvals ...interface{}) {
      tl.logger.Sugar().Warnw(msg, keyvals...)
  }
  
  func (tl *TemporalLogger) Error(msg string, keyvals ...interface{}) {
      tl.logger.Sugar().Errorw(msg, keyvals...)
  }
  ```

### Task 4: 实现健康检查集成 (AC: 连接失败时记录错误并重试)

- [ ] 4.1 创建`internal/temporal/health.go`
  ```go
  package temporal
  
  import (
      "context"
      "time"
      "go.temporal.io/sdk/client"
  )
  
  // HealthCheck 检查Temporal连接健康状态
  func (tc *Client) HealthCheck(ctx context.Context) error {
      if !tc.IsConnected() {
          return fmt.Errorf("temporal client not connected")
      }
      
      // 使用DescribeNamespace验证连接
      ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
      defer cancel()
      
      _, err := tc.client.DescribeNamespace(ctx, tc.config.Namespace)
      if err != nil {
          tc.connected = false
          return fmt.Errorf("temporal health check failed: %w", err)
      }
      
      return nil
  }
  ```

- [ ] 4.2 更新`internal/server/handlers/health.go`
  ```go
  package handlers
  
  import (
      "context"
      "net/http"
      "time"
      "github.com/gin-gonic/gin"
      "waterflow/internal/temporal"
  )
  
  type HealthHandler struct {
      temporalClient *temporal.Client
  }
  
  func NewHealthHandler(tc *temporal.Client) *HealthHandler {
      return &HealthHandler{temporalClient: tc}
  }
  
  // HealthCheck - GET /health (始终返回200)
  func (h *HealthHandler) HealthCheck(c *gin.Context) {
      c.JSON(http.StatusOK, gin.H{
          "status":    "healthy",
          "timestamp": time.Now().Unix(),
      })
  }
  
  // ReadinessCheck - GET /ready (检查依赖服务)
  func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
      ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
      defer cancel()
      
      // 检查Temporal连接
      temporalStatus := "ready"
      if err := h.temporalClient.HealthCheck(ctx); err != nil {
          temporalStatus = "unhealthy"
          c.JSON(http.StatusServiceUnavailable, gin.H{
              "status": "not_ready",
              "checks": gin.H{
                  "temporal": temporalStatus,
                  "error":    err.Error(),
              },
          })
          return
      }
      
      c.JSON(http.StatusOK, gin.H{
          "status": "ready",
          "checks": gin.H{
              "temporal": temporalStatus,
          },
      })
  }
  ```

### Task 5: 集成到主入口 (AC: Waterflow Server启动时成功连接)

- [ ] 5.1 更新`cmd/server/main.go`
  ```go
  package main
  
  import (
      "log"
      "os"
      "os/signal"
      "syscall"
      
      "waterflow/internal/config"
      "waterflow/internal/logger"
      "waterflow/internal/server"
      "waterflow/internal/temporal"
      
      "go.uber.org/zap"
  )
  
  func main() {
      // 1. 加载配置
      cfg, err := config.Load()
      if err != nil {
          log.Fatal("Failed to load config:", err)
      }
      
      // 2. 初始化Logger
      zapLogger := logger.New(cfg.Log)
      defer zapLogger.Sync()
      
      // 3. 连接Temporal (带重试)
      temporalClient, err := temporal.New(&cfg.Temporal, zapLogger)
      if err != nil {
          zapLogger.Fatal("Failed to connect to Temporal", zap.Error(err))
      }
      defer temporalClient.Close()
      
      // 4. 创建HTTP Server
      srv := server.New(cfg, zapLogger, temporalClient)
      
      // 5. 启动服务器
      go func() {
          if err := srv.Start(); err != nil {
              zapLogger.Fatal("Server failed", zap.Error(err))
          }
      }()
      
      // 6. 优雅关闭
      quit := make(chan os.Signal, 1)
      signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
      <-quit
      
      zapLogger.Info("Shutting down server...")
      srv.Shutdown()
      zapLogger.Info("Server exited")
  }
  ```

- [ ] 5.2 更新`internal/server/server.go`
  ```go
  type Server struct {
      config         *config.Config
      logger         *zap.Logger
      router         *gin.Engine
      httpServer     *http.Server
      temporalClient *temporal.Client  // 新增
  }
  
  func New(cfg *config.Config, logger *zap.Logger, tc *temporal.Client) *Server {
      router := SetupRouter(logger, tc)  // 传递temporal client
      
      return &Server{
          config:         cfg,
          logger:         logger,
          temporalClient: tc,
          router:         router,
          httpServer: &http.Server{
              Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
              Handler: router,
          },
      }
  }
  ```

- [ ] 5.3 更新`internal/server/router.go`
  ```go
  func SetupRouter(logger *zap.Logger, tc *temporal.Client) *gin.Engine {
      router := gin.New()
      
      // ... 中间件配置 ...
      
      // 健康检查 (需要Temporal Client)
      healthHandler := handlers.NewHealthHandler(tc)
      router.GET("/health", healthHandler.HealthCheck)
      router.GET("/ready", healthHandler.ReadinessCheck)
      
      // API v1
      v1 := router.Group("/v1")
      {
          v1.GET("/", handlers.APIVersionInfo)
          v1.POST("/validate", handlers.NewValidateHandler().Validate)
          // Story 1.5将添加: v1.POST("/workflows", ...)
      }
      
      return router
  }
  ```

### Task 6: 注册Temporal Namespace (AC: 注册Waterflow Namespace)

- [ ] 6.1 创建Namespace注册脚本
  ```bash
  # scripts/setup_temporal_namespace.sh
  #!/bin/bash
  
  TEMPORAL_HOST=${TEMPORAL_HOST:-localhost:7233}
  NAMESPACE=${NAMESPACE:-waterflow}
  
  echo "Registering Temporal namespace: $NAMESPACE"
  
  tctl --address $TEMPORAL_HOST \
       namespace register \
       --namespace $NAMESPACE \
       --description "Waterflow workflow orchestration namespace" \
       --retention 7
  
  if [ $? -eq 0 ]; then
      echo "✅ Namespace registered successfully"
  else
      echo "❌ Failed to register namespace"
      exit 1
  fi
  ```

- [ ] 6.2 使其可执行
  ```bash
  chmod +x scripts/setup_temporal_namespace.sh
  ```

- [ ] 6.3 添加到文档
  ```markdown
  # docs/deployment.md
  
  ## Temporal Setup
  
  1. Start Temporal Server (Docker Compose):
     ```bash
     cd deployments
     docker-compose up -d temporal
     ```
  
  2. Register namespace:
     ```bash
     ./scripts/setup_temporal_namespace.sh
     ```
  
  3. Verify:
     ```bash
     tctl namespace describe waterflow
     ```
  ```

### Task 7: 添加单元测试 (代码质量保障)

- [ ] 7.1 创建`internal/temporal/client_test.go`
  ```go
  package temporal
  
  import (
      "testing"
      "time"
      "github.com/stretchr/testify/assert"
      "go.uber.org/zap"
      "waterflow/internal/config"
  )
  
  func TestNew_Success(t *testing.T) {
      // 需要运行中的Temporal Server
      if testing.Short() {
          t.Skip("Skipping integration test")
      }
      
      cfg := &config.TemporalConfig{
          HostPort:          "localhost:7233",
          Namespace:         "default",
          ConnectionTimeout: 10 * time.Second,
          Retry: config.RetryConfig{
              MaxAttempts:     3,
              InitialInterval: 1 * time.Second,
          },
      }
      
      logger := zap.NewNop()
      
      client, err := New(cfg, logger)
      assert.NoError(t, err)
      assert.NotNil(t, client)
      assert.True(t, client.IsConnected())
      
      defer client.Close()
  }
  
  func TestNew_InvalidConfig(t *testing.T) {
      cfg := &config.TemporalConfig{
          HostPort:  "", // 无效
          Namespace: "waterflow",
      }
      
      logger := zap.NewNop()
      
      _, err := New(cfg, logger)
      assert.Error(t, err)
      assert.Contains(t, err.Error(), "host_port")
  }
  
  func TestNew_ConnectionFailed(t *testing.T) {
      cfg := &config.TemporalConfig{
          HostPort:          "invalid-host:9999",
          Namespace:         "waterflow",
          ConnectionTimeout: 2 * time.Second,
          Retry: config.RetryConfig{
              MaxAttempts:     2,
              InitialInterval: 1 * time.Second,
          },
      }
      
      logger := zap.NewNop()
      
      _, err := New(cfg, logger)
      assert.Error(t, err)
      assert.Contains(t, err.Error(), "failed to connect")
  }
  ```

- [ ] 7.2 创建`internal/temporal/health_test.go`
  ```go
  func TestHealthCheck_Success(t *testing.T) {
      if testing.Short() {
          t.Skip("Skipping integration test")
      }
      
      cfg := &config.TemporalConfig{
          HostPort:          "localhost:7233",
          Namespace:         "default",
          ConnectionTimeout: 10 * time.Second,
          Retry: config.RetryConfig{
              MaxAttempts:     3,
              InitialInterval: 1 * time.Second,
          },
      }
      
      client, _ := New(cfg, zap.NewNop())
      defer client.Close()
      
      ctx := context.Background()
      err := client.HealthCheck(ctx)
      assert.NoError(t, err)
  }
  ```

- [ ] 7.3 运行测试
  ```bash
  # 单元测试 (不需要Temporal)
  go test -short ./internal/temporal
  
  # 集成测试 (需要Temporal Server运行)
  docker-compose -f deployments/docker-compose.yaml up -d temporal
  go test ./internal/temporal
  ```

### Task 8: 创建Docker Compose部署文件 (可选,本地开发)

- [ ] 8.1 创建`deployments/docker-compose.yaml`
  ```yaml
  version: "3.8"
  
  services:
    temporal:
      image: temporalio/auto-setup:1.22.0
      ports:
        - "7233:7233"  # Frontend gRPC
        - "8233:8233"  # Web UI
      environment:
        - DB=postgresql
        - DB_PORT=5432
        - POSTGRES_USER=temporal
        - POSTGRES_PWD=temporal
        - POSTGRES_SEEDS=postgres
        - DYNAMIC_CONFIG_FILE_PATH=config/dynamicconfig/development-sql.yaml
      volumes:
        - ./temporal-dynamicconfig:/etc/temporal/config/dynamicconfig
      depends_on:
        - postgres
      networks:
        - waterflow-net
    
    postgres:
      image: postgres:14-alpine
      environment:
        POSTGRES_USER: temporal
        POSTGRES_PASSWORD: temporal
        POSTGRES_DB: temporal
      ports:
        - "5432:5432"
      volumes:
        - temporal-postgres-data:/var/lib/postgresql/data
      networks:
        - waterflow-net
  
  volumes:
    temporal-postgres-data:
  
  networks:
    waterflow-net:
      driver: bridge
  ```

- [ ] 8.2 创建启动脚本
  ```bash
  # scripts/start_dev_env.sh
  #!/bin/bash
  
  echo "Starting Temporal development environment..."
  
  cd deployments
  docker-compose up -d
  
  echo "Waiting for Temporal to be ready..."
  sleep 10
  
  # 注册namespace
  cd ..
  ./scripts/setup_temporal_namespace.sh
  
  echo "✅ Development environment ready"
  echo "   Temporal UI: http://localhost:8233"
  echo "   Temporal gRPC: localhost:7233"
  ```

- [ ] 8.3 添加到Makefile
  ```makefile
  .PHONY: dev-env
  dev-env:
  	@echo "Starting development environment..."
  	@./scripts/start_dev_env.sh
  
  .PHONY: dev-env-stop
  dev-env-stop:
  	@echo "Stopping development environment..."
  	@cd deployments && docker-compose down
  ```

## Dev Notes

### Critical Implementation Guidelines

**1. 连接重试策略**

```go
// ❌ 错误: 无限重试可能导致启动挂起
for {
    c, err := client.Dial(options)
    if err == nil {
        return c, nil
    }
    time.Sleep(5 * time.Second)
}

// ✅ 正确: 限制重试次数
for attempt := 1; attempt <= maxAttempts; attempt++ {
    c, err := client.Dial(options)
    if err == nil {
        return c, nil
    }
    if attempt < maxAttempts {
        time.Sleep(initialInterval)
    }
}
return nil, fmt.Errorf("connection failed after %d attempts", maxAttempts)
```

**2. Context超时控制**

```go
// ✅ 使用Context控制连接超时
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

c, err := client.DialContext(ctx, options)
if err != nil {
    // 区分超时错误和其他错误
    if errors.Is(err, context.DeadlineExceeded) {
        return fmt.Errorf("connection timeout after 10s")
    }
    return err
}
```

**3. 日志集成最佳实践**

```go
// Temporal SDK使用自己的Logger接口,需要适配Zap
type TemporalLogger struct {
    logger *zap.Logger
}

func (tl *TemporalLogger) Debug(msg string, keyvals ...interface{}) {
    // 将Temporal的key-value格式转换为Zap
    tl.logger.Sugar().Debugw(msg, keyvals...)
}

// 使用:
options := client.Options{
    Logger: NewTemporalLogger(zapLogger),
}
```

**4. 健康检查实现**

```go
// ✅ 使用DescribeNamespace验证连接
func (tc *Client) HealthCheck(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
    defer cancel()
    
    _, err := tc.client.DescribeNamespace(ctx, tc.config.Namespace)
    return err
}

// ❌ 避免: 仅检查client是否为nil (不验证实际连接)
func (tc *Client) HealthCheck() error {
    if tc.client == nil {
        return fmt.Errorf("client is nil")
    }
    return nil  // 不准确!
}
```

**5. 优雅关闭**

```go
// 在main.go中确保关闭Temporal连接
defer temporalClient.Close()

// Client.Close()实现:
func (tc *Client) Close() {
    if tc.client != nil {
        tc.client.Close()
        tc.connected = false
        tc.logger.Info("Temporal connection closed")
    }
}
```

**6. Namespace注册注意事项**

```bash
# 检查namespace是否已存在
tctl namespace describe waterflow

# 如果已存在,跳过注册
if [ $? -eq 0 ]; then
    echo "Namespace already exists"
    exit 0
fi

# 注册新namespace
tctl namespace register waterflow --retention 7
```

**7. TLS配置 (生产环境)**

```go
// 本Story暂不实现TLS,预留接口
if cfg.TLS.Enabled {
    tlsConfig, err := LoadTLSConfig(cfg.TLS.CertFile, cfg.TLS.KeyFile)
    if err != nil {
        return nil, err
    }
    options.ConnectionOptions = client.ConnectionOptions{
        TLS: tlsConfig,
    }
}
```

### Integration with Previous Stories

**与Story 1.2 REST API集成:**

```go
// Story 1.2提供的/ready端点
func ReadinessCheck(c *gin.Context) {
    // Story 1.4扩展: 检查Temporal连接
    if err := temporalClient.HealthCheck(ctx); err != nil {
        c.JSON(503, gin.H{"status": "not_ready", "error": err.Error()})
        return
    }
    c.JSON(200, gin.H{"status": "ready"})
}
```

**为Story 1.5准备:**

```go
// Story 1.5将使用Temporal Client提交Workflow
func (h *WorkflowHandler) Submit(c *gin.Context) {
    // 1. 解析YAML (Story 1.3)
    wf, err := parser.Parse(yamlContent)
    
    // 2. 提交到Temporal (Story 1.4提供的Client)
    we, err := h.temporalClient.GetClient().ExecuteWorkflow(ctx, options, ...)
    
    c.JSON(200, gin.H{"workflow_id": we.GetID()})
}
```

### Testing Strategy

**单元测试 (无需Temporal Server):**

| 测试用例 | 目的 |
|---------|-----|
| TestNew_InvalidConfig | 验证配置验证逻辑 |
| TestValidate_HostPort | 测试必需字段检查 |
| TestLogger_Adaptation | 验证Zap→Temporal Logger适配 |

**集成测试 (需要Temporal Server):**

```bash
# 1. 启动Temporal
make dev-env

# 2. 运行集成测试
go test ./internal/temporal

# 3. 测试连接重试
docker-compose stop temporal
go test -run TestNew_Retry ./internal/temporal
docker-compose start temporal
```

**手动验证:**

```bash
# 1. 启动Waterflow Server
go run ./cmd/server

# 期望日志:
# INFO Connecting to Temporal host_port=localhost:7233
# INFO ✅ Temporal connection successful
# INFO Starting HTTP server addr=0.0.0.0:8080

# 2. 检查健康状态
curl http://localhost:8080/ready
# {"status":"ready","checks":{"temporal":"ready"}}

# 3. 停止Temporal,验证健康检查失败
docker-compose stop temporal
curl http://localhost:8080/ready
# {"status":"not_ready","checks":{"temporal":"unhealthy","error":"..."}}
```

### References

**架构设计:**
- [docs/architecture.md §3.1.3](docs/architecture.md) - Temporal Client组件
- [docs/adr/0001-use-temporal-workflow-engine.md](docs/adr/0001-use-temporal-workflow-engine.md) - Temporal选型决策

**技术文档:**
- [Temporal Go SDK Documentation](https://docs.temporal.io/dev-guide/go)
- [Temporal Client API](https://pkg.go.dev/go.temporal.io/sdk/client)
- [Temporal Namespace Management](https://docs.temporal.io/namespaces)

**项目上下文:**
- [docs/epics.md Story 1.1-1.3](docs/epics.md) - 前置Stories
- [docs/epics.md Story 1.5-1.7](docs/epics.md) - 后续依赖本Story

**部署文档:**
- [Temporal Server Setup](https://docs.temporal.io/cluster-deployment-guide)
- [Docker Compose Example](https://github.com/temporalio/docker-compose)

### Dependency Graph

```
Story 1.1 (框架)
    ↓
Story 1.2 (REST API)
    ↓
Story 1.3 (YAML解析)
    ↓
Story 1.4 (Temporal SDK集成) ← 当前Story
    ↓
    ├→ Story 1.5 (工作流提交API) - 使用Client.ExecuteWorkflow()
    ├→ Story 1.6 (执行引擎) - 定义Temporal Workflow
    ├→ Story 1.7 (状态查询API) - 使用Client.DescribeWorkflowExecution()
    └→ Story 1.9 (取消API) - 使用Client.CancelWorkflow()
```

## Dev Agent Record

### Context Reference

**Source Documents Analyzed:**
1. [docs/epics.md](docs/epics.md) (lines 327-342) - Story 1.4需求定义
2. [docs/architecture.md](docs/architecture.md) (§2.2, §3.1.3) - Temporal架构设计
3. [docs/adr/0001-use-temporal-workflow-engine.md](docs/adr/0001-use-temporal-workflow-engine.md) - Temporal选型理由

**Previous Stories:**
- Story 1.1: 项目框架 (drafted)
- Story 1.2: REST API框架 (drafted)
- Story 1.3: YAML解析器 (drafted)

### Agent Model Used

Claude 3.5 Sonnet (BMM Scrum Master Agent - Bob)

### Estimated Effort

**开发时间:** 6-8小时  
**复杂度:** 中等

**时间分解:**
- Temporal SDK集成: 1小时
- 配置结构扩展: 1小时
- 客户端封装实现: 2小时
- 健康检查集成: 1小时
- 主入口集成: 1小时
- Namespace注册脚本: 0.5小时
- 单元/集成测试: 2小时
- Docker Compose环境: 0.5小时

**技能要求:**
- Temporal基础概念 (Namespace, Client, Workflow)
- Go Context和超时控制
- 重试机制实现
- Docker Compose基础

### Debug Log References

<!-- Will be populated during implementation -->

### Completion Notes List

<!-- Developer填写完成时的笔记 -->

### File List

**预期创建/修改的文件清单:**

```
新建文件 (~10个):
├── internal/temporal/
│   ├── client.go                   # Temporal客户端封装
│   ├── options.go                  # Logger适配器
│   ├── health.go                   # 健康检查
│   ├── client_test.go              # 单元测试
│   └── health_test.go              # 健康检查测试
├── scripts/
│   ├── setup_temporal_namespace.sh # Namespace注册脚本
│   └── start_dev_env.sh            # 开发环境启动
├── deployments/
│   └── docker-compose.yaml         # Temporal本地部署

修改文件 (~5个):
├── internal/config/config.go       # 添加TemporalConfig
├── internal/server/server.go       # 集成Temporal Client
├── internal/server/router.go       # 传递Temporal Client
├── internal/server/handlers/health.go  # 更新/ready端点
├── cmd/server/main.go              # 初始化Temporal连接
└── deployments/config.yaml         # 添加temporal配置段
```

**关键代码片段:**

**client.go (核心):**
```go
package temporal

import (
    "context"
    "fmt"
    "time"
    "go.temporal.io/sdk/client"
    "go.uber.org/zap"
)

type Client struct {
    client    client.Client
    config    *config.TemporalConfig
    logger    *zap.Logger
    connected bool
}

func New(cfg *config.TemporalConfig, logger *zap.Logger) (*Client, error) {
    tc := &Client{config: cfg, logger: logger}
    
    // 重试连接
    var lastErr error
    for attempt := 1; attempt <= cfg.Retry.MaxAttempts; attempt++ {
        c, err := tc.dial()
        if err == nil {
            tc.client = c
            tc.connected = true
            logger.Info("✅ Temporal connection successful")
            return tc, nil
        }
        lastErr = err
        time.Sleep(cfg.Retry.InitialInterval)
    }
    
    return nil, fmt.Errorf("failed to connect: %w", lastErr)
}

func (tc *Client) dial() (client.Client, error) {
    ctx, cancel := context.WithTimeout(context.Background(), tc.config.ConnectionTimeout)
    defer cancel()
    
    return client.DialContext(ctx, client.Options{
        HostPort:  tc.config.HostPort,
        Namespace: tc.config.Namespace,
        Logger:    NewTemporalLogger(tc.logger),
    })
}
```

**health.go:**
```go
func (tc *Client) HealthCheck(ctx context.Context) error {
    if !tc.IsConnected() {
        return fmt.Errorf("temporal client not connected")
    }
    
    ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
    defer cancel()
    
    _, err := tc.client.DescribeNamespace(ctx, tc.config.Namespace)
    return err
}
```

**main.go集成:**
```go
func main() {
    cfg, _ := config.Load()
    logger := logger.New(cfg.Log)
    
    // 连接Temporal
    temporalClient, err := temporal.New(&cfg.Temporal, logger)
    if err != nil {
        logger.Fatal("Failed to connect to Temporal", zap.Error(err))
    }
    defer temporalClient.Close()
    
    srv := server.New(cfg, logger, temporalClient)
    srv.Start()
}
```

---

**Story Ready for Development** ✅

开发者可基于Story 1.1-1.3的成果,集成Temporal SDK实现工作流引擎连接。
本Story为Story 1.5-1.7的工作流操作API奠定基础。
