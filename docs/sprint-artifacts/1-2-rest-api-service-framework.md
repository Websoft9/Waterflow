# Story 1.2: REST API 服务框架

Status: drafted

## Story

As a **开发者**,  
I want **实现 REST API 服务框架**,  
so that **可以通过 HTTP 接口接收工作流请求**。

## Acceptance Criteria

**Given** Waterflow Server 框架已搭建  
**When** 启动 Server 进程  
**Then** HTTP 服务监听在配置的端口 (默认 8080)  
**And** 提供健康检查端点 `GET /health` 返回 200  
**And** 提供就绪检查端点 `GET /ready` 返回服务状态  
**And** 支持优雅关闭 (SIGTERM)  
**And** 配置通过环境变量或 YAML 文件加载  
**And** 结构化日志输出到 stdout

## Technical Context

### Architecture Constraints

根据 [docs/architecture.md](docs/architecture.md) §3.1.1 REST API Handler设计:

1. **核心职责**
   - 处理HTTP请求 (提交工作流、查询状态、获取日志)
   - 请求参数验证
   - API认证 (API Key/JWT) - 本Story暂不实现,Epic 10处理
   - 错误响应格式化 (RFC 7807)

2. **关键端点** (本Story实现基础框架)
   - `GET /health` - 健康检查 (始终返回200)
   - `GET /ready` - 就绪检查 (检查依赖服务状态)
   - `GET /v1/` - API版本信息
   - 后续Stories将添加: `/v1/workflows`, `/v1/validate` 等

3. **非功能性需求** (参考 PRD NFR1-NFR5)
   - 响应时间: p95 < 500ms
   - 并发支持: 100+ req/s
   - 优雅关闭: 最多等待30秒完成进行中的请求
   - 日志: JSON格式,包含request_id, trace_id

### Dependencies

**前置Story:**
- ✅ Story 1.1: Waterflow Server框架搭建 (已完成)
  - 依赖: 项目结构, go.mod, Gin依赖已安装
  - 使用: `internal/config`, `internal/logger` 模块

**后续Story依赖本Story:**
- Story 1.3-1.9: 所有REST API端点都基于本Story建立的框架

### Technology Stack

**Web框架: Gin v1.9+**

选择理由 (参考Story 1.1决策):
- 高性能: 比标准库net/http快40倍
- 中间件丰富: CORS, 日志, Recovery, 限流等
- 路由分组: 便于API版本管理 (`/v1/`, `/v2/`)
- 社区活跃: 70k+ GitHub stars

**核心中间件:**

1. **Recovery Middleware** - panic恢复
   ```go
   gin.Recovery() // 内置,捕获panic并返回500
   ```

2. **Logger Middleware** - 请求日志
   ```go
   // 自定义,集成Zap结构化日志
   // 记录: method, path, status, latency, client_ip
   ```

3. **CORS Middleware** - 跨域支持
   ```go
   github.com/gin-contrib/cors
   // 配置: 允许的origin, methods, headers
   ```

4. **RequestID Middleware** - 请求追踪
   ```go
   github.com/gin-contrib/requestid
   // 生成UUID,注入到context和响应头
   ```

**配置管理: 基于Story 1.1的Viper**

```yaml
server:
  port: 8080
  host: 0.0.0.0
  mode: release  # gin mode: debug, release, test
  shutdown_timeout: 30s
  cors:
    enabled: true
    allowed_origins: ["*"]
log:
  level: info
  format: json
```

**环境变量覆盖:**
- `WATERFLOW_SERVER_PORT` → server.port
- `WATERFLOW_SERVER_HOST` → server.host
- `WATERFLOW_LOG_LEVEL` → log.level

### Project Structure Updates

基于Story 1.1创建的结构,本Story新增/修改:

```
internal/
├── server/
│   ├── server.go           # HTTP服务器主逻辑 (新建)
│   ├── router.go           # 路由配置 (新建)
│   ├── middleware/         # 中间件目录 (新建)
│   │   ├── logger.go       # 日志中间件
│   │   ├── requestid.go    # RequestID中间件
│   │   └── recovery.go     # Recovery中间件 (可选,Gin内置)
│   └── handlers/           # HTTP处理器目录 (新建)
│       ├── health.go       # 健康检查handler
│       └── version.go      # 版本信息handler
├── config/
│   └── config.go           # 配置结构扩展 (修改)
└── logger/
    └── logger.go           # Logger初始化 (修改)

cmd/server/
└── main.go                 # 主入口,启动HTTP服务器 (修改)

api/
└── openapi.yaml            # OpenAPI规范 (新建/更新)
```

## Tasks / Subtasks

### Task 1: 实现HTTP Server核心逻辑 (AC: HTTP服务监听端口)

- [ ] 1.1 创建`internal/server/server.go`
  ```go
  type Server struct {
      config *config.Config
      logger *zap.Logger
      router *gin.Engine
      httpServer *http.Server
  }
  
  func New(cfg *config.Config, logger *zap.Logger) *Server
  func (s *Server) Start() error
  func (s *Server) Shutdown(ctx context.Context) error
  ```

- [ ] 1.2 实现服务器启动逻辑
  - 设置Gin模式 (根据config.server.mode)
  - 创建http.Server实例
  - 配置监听地址 (config.server.host:port)
  - 启动goroutine监听HTTP请求
  - 返回error如果端口被占用

- [ ] 1.3 验证服务器启动
  ```bash
  go run ./cmd/server
  # 输出: Server listening on 0.0.0.0:8080
  curl http://localhost:8080/
  # 期望: 404 (路由尚未配置)
  ```

### Task 2: 实现优雅关闭机制 (AC: 支持优雅关闭)

- [ ] 2.1 在`main.go`中捕获信号
  ```go
  quit := make(chan os.Signal, 1)
  signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
  <-quit
  logger.Info("Shutting down server...")
  ```

- [ ] 2.2 实现Shutdown方法
  ```go
  ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
  defer cancel()
  if err := srv.Shutdown(ctx); err != nil {
      logger.Fatal("Server forced to shutdown", zap.Error(err))
  }
  ```

- [ ] 2.3 测试优雅关闭
  ```bash
  # Terminal 1: 启动服务器
  go run ./cmd/server
  
  # Terminal 2: 发送SIGTERM
  kill -TERM <pid>
  
  # 期望: 日志显示"Shutting down server..."
  # 等待进行中请求完成(最多30秒)
  ```

### Task 3: 配置中间件链 (AC: 结构化日志输出)

- [ ] 3.1 创建`internal/server/middleware/logger.go`
  - 使用Zap记录每个请求
  - 字段: timestamp, method, path, status, latency, client_ip, request_id
  - 格式: JSON
  - 示例:
    ```json
    {
      "level": "info",
      "ts": "2025-12-16T10:30:45.123Z",
      "method": "GET",
      "path": "/health",
      "status": 200,
      "latency_ms": 1.2,
      "client_ip": "192.168.1.1",
      "request_id": "uuid-123"
    }
    ```

- [ ] 3.2 创建`internal/server/middleware/requestid.go`
  - 使用`github.com/gin-contrib/requestid`
  - 或自定义: 生成UUID,存入context
  - 添加响应头: `X-Request-ID`

- [ ] 3.3 配置CORS中间件
  ```go
  import "github.com/gin-contrib/cors"
  
  router.Use(cors.New(cors.Config{
      AllowOrigins: config.Server.CORS.AllowedOrigins,
      AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
      AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
  }))
  ```

- [ ] 3.4 在router.go中应用中间件
  ```go
  router.Use(gin.Recovery())
  router.Use(middleware.RequestID())
  router.Use(middleware.Logger(logger))
  router.Use(cors.New(...))
  ```

### Task 4: 实现健康检查和就绪检查端点 (AC: /health和/ready端点)

- [ ] 4.1 创建`internal/server/handlers/health.go`
  ```go
  // GET /health - 始终返回200 OK
  func HealthCheck(c *gin.Context) {
      c.JSON(http.StatusOK, gin.H{
          "status": "healthy",
          "timestamp": time.Now().Unix(),
      })
  }
  ```

- [ ] 4.2 创建就绪检查handler
  ```go
  // GET /ready - 检查依赖服务状态
  func ReadinessCheck(c *gin.Context) {
      // 本Story暂时返回OK
      // Story 1.4添加Temporal连接检查
      c.JSON(http.StatusOK, gin.H{
          "status": "ready",
          "checks": gin.H{
              "temporal": "not_configured", // 待Story 1.4实现
          },
      })
  }
  ```

- [ ] 4.3 在router.go注册端点
  ```go
  router.GET("/health", handlers.HealthCheck)
  router.GET("/ready", handlers.ReadinessCheck)
  router.GET("/", handlers.VersionInfo) // 可选
  ```

- [ ] 4.4 测试端点
  ```bash
  curl http://localhost:8080/health
  # {"status":"healthy","timestamp":1702728645}
  
  curl http://localhost:8080/ready
  # {"status":"ready","checks":{"temporal":"not_configured"}}
  ```

### Task 5: 实现API版本路由分组 (为后续Stories准备)

- [ ] 5.1 创建`internal/server/router.go`
  ```go
  func SetupRouter(logger *zap.Logger) *gin.Engine {
      router := gin.New()
      
      // 全局中间件
      router.Use(gin.Recovery())
      router.Use(middleware.RequestID())
      router.Use(middleware.Logger(logger))
      
      // 根路径
      router.GET("/health", handlers.HealthCheck)
      router.GET("/ready", handlers.ReadinessCheck)
      
      // API v1路由组
      v1 := router.Group("/v1")
      {
          v1.GET("/", handlers.APIVersionInfo)
          // Story 1.5将添加: v1.POST("/workflows", ...)
          // Story 1.7将添加: v1.GET("/workflows/:id", ...)
      }
      
      return router
  }
  ```

- [ ] 5.2 实现版本信息handler
  ```go
  func APIVersionInfo(c *gin.Context) {
      c.JSON(http.StatusOK, gin.H{
          "version": "v1",
          "build": os.Getenv("BUILD_VERSION"), // CI时注入
          "endpoints": []string{
              "POST /v1/workflows (TODO)",
              "GET /v1/workflows/:id (TODO)",
          },
      })
  }
  ```

### Task 6: 更新配置和主入口 (AC: 配置通过环境变量/YAML加载)

- [ ] 6.1 扩展`internal/config/config.go`
  ```go
  type Config struct {
      Server ServerConfig `mapstructure:"server"`
      Log    LogConfig    `mapstructure:"log"`
  }
  
  type ServerConfig struct {
      Port            int           `mapstructure:"port"`
      Host            string        `mapstructure:"host"`
      Mode            string        `mapstructure:"mode"` // debug/release/test
      ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
      CORS            CORSConfig    `mapstructure:"cors"`
  }
  
  type CORSConfig struct {
      Enabled        bool     `mapstructure:"enabled"`
      AllowedOrigins []string `mapstructure:"allowed_origins"`
  }
  ```

- [ ] 6.2 更新默认配置文件`deployments/config.yaml`
  ```yaml
  server:
    port: 8080
    host: 0.0.0.0
    mode: release
    shutdown_timeout: 30s
    cors:
      enabled: true
      allowed_origins: ["*"]
  
  log:
    level: info
    format: json
  ```

- [ ] 6.3 更新`cmd/server/main.go`
  ```go
  func main() {
      // 1. 加载配置
      cfg, err := config.Load()
      if err != nil {
          log.Fatal("Failed to load config:", err)
      }
      
      // 2. 初始化Logger
      logger := logger.New(cfg.Log)
      defer logger.Sync()
      
      // 3. 创建HTTP Server
      srv := server.New(cfg, logger)
      
      // 4. 启动服务器(goroutine)
      go func() {
          if err := srv.Start(); err != nil && err != http.ErrServerClosed {
              logger.Fatal("Server failed", zap.Error(err))
          }
      }()
      
      // 5. 优雅关闭
      quit := make(chan os.Signal, 1)
      signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
      <-quit
      
      ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
      defer cancel()
      
      if err := srv.Shutdown(ctx); err != nil {
          logger.Fatal("Forced shutdown", zap.Error(err))
      }
      logger.Info("Server exited")
  }
  ```

### Task 7: 添加单元测试 (确保代码质量)

- [ ] 7.1 创建`internal/server/server_test.go`
  - 测试服务器启动和关闭
  - 测试端口冲突处理
  - 使用mock config和logger

- [ ] 7.2 创建`internal/server/handlers/health_test.go`
  ```go
  func TestHealthCheck(t *testing.T) {
      router := gin.New()
      router.GET("/health", HealthCheck)
      
      req := httptest.NewRequest("GET", "/health", nil)
      w := httptest.NewRecorder()
      router.ServeHTTP(w, req)
      
      assert.Equal(t, 200, w.Code)
      assert.Contains(t, w.Body.String(), "healthy")
  }
  ```

- [ ] 7.3 测试中间件功能
  - RequestID注入测试
  - Logger中间件日志输出测试
  - CORS头部验证测试

- [ ] 7.4 运行测试并验证覆盖率
  ```bash
  make test
  # 期望: 所有测试通过
  # 覆盖率: internal/server目录 >70%
  ```

### Task 8: 更新OpenAPI规范 (API文档)

- [ ] 8.1 创建/更新`api/openapi.yaml`
  ```yaml
  openapi: 3.0.0
  info:
    title: Waterflow API
    version: 1.0.0
  servers:
    - url: http://localhost:8080
  paths:
    /health:
      get:
        summary: Health check
        responses:
          '200':
            description: Service is healthy
    /ready:
      get:
        summary: Readiness check
        responses:
          '200':
            description: Service is ready
    /v1/:
      get:
        summary: API version information
  ```

- [ ] 8.2 集成Swagger UI (可选)
  - 使用`github.com/swaggo/gin-swagger`
  - 访问: http://localhost:8080/swagger/index.html

## Dev Notes

### Critical Implementation Guidelines

**1. 错误处理规范**

遵循RFC 7807 Problem Details标准:

```go
type ProblemDetail struct {
    Type     string      `json:"type"`
    Title    string      `json:"title"`
    Status   int         `json:"status"`
    Detail   string      `json:"detail,omitempty"`
    Instance string      `json:"instance,omitempty"`
    TraceID  string      `json:"trace_id,omitempty"`
}

// 示例: 404错误
c.JSON(404, ProblemDetail{
    Type:     "https://waterflow.io/errors/not-found",
    Title:    "Resource Not Found",
    Status:   404,
    Detail:   "Workflow with ID xyz not found",
    Instance: "/v1/workflows/xyz",
    TraceID:  requestID,
})
```

**2. 日志规范 - 基于Story 1.1的Zap**

```go
// 请求日志
logger.Info("Request processed",
    zap.String("method", c.Request.Method),
    zap.String("path", c.Request.URL.Path),
    zap.Int("status", c.Writer.Status()),
    zap.Duration("latency", latency),
    zap.String("client_ip", c.ClientIP()),
    zap.String("request_id", requestID),
)

// 错误日志
logger.Error("Failed to process request",
    zap.Error(err),
    zap.String("request_id", requestID),
    zap.String("path", c.Request.URL.Path),
)
```

**3. Context传递 - 为Temporal集成准备**

```go
// 将Gin Context转换为标准context.Context
func GinContextToContext(c *gin.Context) context.Context {
    ctx := c.Request.Context()
    // 注入request_id用于分布式追踪
    ctx = context.WithValue(ctx, "request_id", c.GetString("request_id"))
    return ctx
}

// Story 1.4将使用此context调用Temporal
```

**4. 配置验证 - 启动时检查**

```go
func (cfg *ServerConfig) Validate() error {
    if cfg.Port < 1024 || cfg.Port > 65535 {
        return fmt.Errorf("invalid port: %d", cfg.Port)
    }
    if cfg.Mode != "debug" && cfg.Mode != "release" && cfg.Mode != "test" {
        return fmt.Errorf("invalid mode: %s", cfg.Mode)
    }
    return nil
}
```

**5. 性能考虑**

- **路由性能:** Gin使用基数树,查找复杂度O(log n)
- **中间件开销:** 每个中间件增加约0.1ms延迟,合理控制数量
- **JSON序列化:** 使用`encoding/json`标准库,大数据考虑`jsoniter`
- **并发:** Gin默认无并发限制,生产环境建议添加限流中间件

**6. 安全最佳实践 (本Story基础)**

```go
// 1. 禁用Debug模式(生产)
gin.SetMode(gin.ReleaseMode)

// 2. 设置请求大小限制
router.Use(func(c *gin.Context) {
    c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20) // 10MB
    c.Next()
})

// 3. 设置超时
srv := &http.Server{
    Addr:         cfg.Server.Addr(),
    Handler:      router,
    ReadTimeout:  10 * time.Second,
    WriteTimeout: 10 * time.Second,
    IdleTimeout:  60 * time.Second,
}
```

### Integration with Story 1.1

**复用Story 1.1成果:**

1. ✅ **项目结构** - 直接在internal/server/下创建新文件
2. ✅ **Gin依赖** - go.mod已包含github.com/gin-gonic/gin
3. ✅ **Logger模块** - internal/logger/logger.go已实现
4. ✅ **Config模块** - internal/config/config.go扩展即可

**需要修改Story 1.1文件:**

- `cmd/server/main.go` - 从简单入口改为完整HTTP服务器
- `internal/config/config.go` - 添加ServerConfig结构
- `deployments/config.yaml` - 添加server配置段

### Testing Strategy

**单元测试:**
- `server_test.go` - 服务器启停测试
- `health_test.go` - 健康检查端点
- `middleware_test.go` - 中间件功能

**集成测试 (可选):**
```go
func TestServerIntegration(t *testing.T) {
    // 1. 启动测试服务器
    srv := setupTestServer()
    go srv.Start()
    defer srv.Shutdown(context.Background())
    
    // 2. 等待服务器就绪
    time.Sleep(100 * time.Millisecond)
    
    // 3. 测试HTTP请求
    resp, err := http.Get("http://localhost:8080/health")
    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
}
```

**性能测试 (可选):**
```bash
# 使用hey工具
hey -n 1000 -c 10 http://localhost:8080/health
# 期望: p95 < 10ms, 成功率100%
```

### References

**架构设计:**
- [docs/architecture.md §3.1.1](docs/architecture.md) - REST API Handler职责
- [docs/architecture.md §2.1](docs/architecture.md) - Waterflow Server容器定义

**技术规范:**
- [RFC 7807: Problem Details for HTTP APIs](https://tools.ietf.org/html/rfc7807)
- [12-Factor App: Config](https://12factor.net/config)
- [Gin Documentation](https://gin-gonic.com/docs/)

**项目上下文:**
- [docs/epics.md Story 1.1](docs/epics.md) - 前置依赖Story
- [docs/epics.md Story 1.3-1.9](docs/epics.md) - 后续依赖本Story的API端点

**Go规范:**
- [Effective Go: Web Servers](https://go.dev/doc/articles/wiki/)
- [Uber Go Style Guide: Middleware](https://github.com/uber-go/guide/blob/master/style.md#middleware)

### Dependency Graph

```
Story 1.1 (框架)
    ↓
Story 1.2 (REST API框架) ← 当前Story
    ↓
    ├→ Story 1.3 (DSL解析器)
    ├→ Story 1.4 (Temporal集成)
    ├→ Story 1.5 (工作流提交API)
    ├→ Story 1.7 (状态查询API)
    └→ Story 1.8-1.9 (日志/取消API)
```

## Dev Agent Record

### Context Reference

<!-- Story context will be generated in subsequent workflow step -->

### Agent Model Used

Claude 3.5 Sonnet (BMM Scrum Master Agent - Bob)

### Estimated Effort

**开发时间:** 6-8小时  
**复杂度:** 中等

**时间分解:**
- HTTP Server核心逻辑: 1.5小时
- 中间件开发和集成: 2小时
- 健康检查端点实现: 0.5小时
- 路由配置和版本分组: 1小时
- 配置扩展和主入口更新: 1小时
- 单元测试编写: 1.5小时
- 集成测试和调试: 0.5-1小时

**技能要求:**
- Go Web开发 (Gin框架)
- HTTP协议和RESTful API
- 中间件模式理解
- 单元测试经验

### Debug Log References

<!-- Will be populated during implementation -->

### Completion Notes List

<!-- Developer填写完成时的笔记 -->

### File List

**预期创建/修改的文件清单:**

```
新建文件 (~12个):
├── internal/server/server.go
├── internal/server/router.go
├── internal/server/middleware/logger.go
├── internal/server/middleware/requestid.go
├── internal/server/handlers/health.go
├── internal/server/handlers/version.go
├── internal/server/server_test.go
├── internal/server/handlers/health_test.go
├── internal/server/middleware/logger_test.go
└── api/openapi.yaml

修改文件 (~3个):
├── cmd/server/main.go              # 添加HTTP服务器启动逻辑
├── internal/config/config.go       # 扩展ServerConfig
└── deployments/config.yaml         # 添加server配置段
```

**关键代码片段:**

**server.go:**
```go
package server

import (
    "context"
    "fmt"
    "net/http"
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

type Server struct {
    config     *config.Config
    logger     *zap.Logger
    router     *gin.Engine
    httpServer *http.Server
}

func New(cfg *config.Config, logger *zap.Logger) *Server {
    router := SetupRouter(logger)
    
    return &Server{
        config: cfg,
        logger: logger,
        router: router,
        httpServer: &http.Server{
            Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
            Handler:      router,
            ReadTimeout:  10 * time.Second,
            WriteTimeout: 10 * time.Second,
        },
    }
}

func (s *Server) Start() error {
    s.logger.Info("Starting HTTP server",
        zap.String("addr", s.httpServer.Addr),
        zap.String("mode", gin.Mode()),
    )
    return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
    s.logger.Info("Shutting down HTTP server")
    return s.httpServer.Shutdown(ctx)
}
```

**health.go:**
```go
package handlers

import (
    "net/http"
    "time"
    "github.com/gin-gonic/gin"
)

func HealthCheck(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status":    "healthy",
        "timestamp": time.Now().Unix(),
    })
}

func ReadinessCheck(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status": "ready",
        "checks": gin.H{
            "temporal": "not_configured", // Story 1.4将更新
        },
    })
}
```

---

**Story Ready for Development** ✅

开发者可基于Story 1.1的成果,按照Tasks顺序实施REST API服务框架。
