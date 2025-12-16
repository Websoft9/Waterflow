# Story 1.5: 工作流提交 API

Status: drafted

## Story

As a **工作流用户**,  
I want **通过 REST API 提交工作流**,  
So that **触发工作流执行**。

## Acceptance Criteria

**Given** REST API 服务和 Temporal 集成已完成  
**When** POST `/v1/workflows` 请求带有 YAML 内容  
**Then** 返回工作流 ID 和提交状态  
**And** 工作流 ID 唯一且可追踪  
**And** 请求格式错误返回 400 和详细错误信息  
**And** YAML 验证失败返回 422 和语法错误位置  
**And** 响应时间 <500ms  
**And** 工作流提交到 Temporal 执行队列

## Technical Context

### Architecture Constraints

根据 [docs/architecture.md](docs/architecture.md) §3.1.1 REST API Handler设计:

1. **核心职责**
   - 接收HTTP POST请求 (YAML工作流定义)
   - 调用YAML解析器验证语法 (Story 1.3)
   - 生成唯一WorkflowID
   - 调用Temporal Client提交工作流 (Story 1.4)
   - 返回WorkflowID和RunID给客户端

2. **请求/响应格式**

**请求:**
```json
POST /v1/workflows
Content-Type: application/json

{
  "workflow": "name: Deploy App\non: push\njobs:\n  deploy:\n    runs-on: linux-amd64\n    steps:\n      - name: Deploy\n        uses: run@v1\n        with:\n          command: echo 'Deploying...'"
}
```

**成功响应 (201 Created):**
```json
{
  "workflow_id": "wf-20251216-abc123",
  "run_id": "temporal-generated-uuid",
  "status": "running",
  "submitted_at": "2025-12-16T10:30:00Z"
}
```

**错误响应 (400 Bad Request):**
```json
{
  "type": "https://waterflow.io/errors/bad-request",
  "title": "Bad Request",
  "status": 400,
  "detail": "Missing 'workflow' field in request body"
}
```

**验证错误 (422 Unprocessable Entity):**
```json
{
  "type": "https://waterflow.io/errors/validation-failed",
  "title": "Validation Failed",
  "status": 422,
  "errors": [
    {
      "field": "name",
      "message": "Field 'name' is required"
    }
  ]
}
```

3. **非功能性需求**
   - 响应时间 p95 < 500ms
   - WorkflowID必须全局唯一
   - 幂等性: 相同WorkflowID重复提交应拒绝

### Dependencies

**前置Story:**
- ✅ Story 1.1: Waterflow Server框架搭建
- ✅ Story 1.2: REST API服务框架
  - 使用: HTTP路由、中间件、错误响应格式
- ✅ Story 1.3: YAML DSL解析器
  - 使用: `parser.Parse()` 验证YAML语法
- ✅ Story 1.4: Temporal SDK集成
  - 使用: `temporalClient.ExecuteWorkflow()` 提交工作流

**后续Story依赖本Story:**
- Story 1.6: 工作流执行引擎 - 处理提交后的工作流实际执行
- Story 1.7: 状态查询API - 根据WorkflowID查询状态

### Technology Stack

**核心技术复用:**

1. **Gin框架** (Story 1.2) - HTTP请求处理
2. **YAML解析器** (Story 1.3) - 验证工作流定义
3. **Temporal Client** (Story 1.4) - 提交到工作流引擎

**WorkflowID生成策略:**

```go
// 方案1: UUID v4 (推荐)
import "github.com/google/uuid"

workflowID := fmt.Sprintf("wf-%s", uuid.New().String())
// 示例: wf-550e8400-e29b-41d4-a716-446655440000

// 方案2: 时间戳 + 随机后缀
import "crypto/rand"

timestamp := time.Now().Format("20060102-150405")
randomSuffix := generateRandomString(8)
workflowID := fmt.Sprintf("wf-%s-%s", timestamp, randomSuffix)
// 示例: wf-20251216-103045-a7b3c9d2
```

**Temporal ExecuteWorkflow API:**

```go
import (
    "go.temporal.io/sdk/client"
)

workflowOptions := client.StartWorkflowOptions{
    ID:                 workflowID,        // 用户生成的唯一ID
    TaskQueue:          "default",         // 后续Story将从DSL提取
    WorkflowRunTimeout: 1 * time.Hour,     // 工作流超时
}

// 提交工作流 (Story 1.6将实现WorkflowFunc)
we, err := temporalClient.ExecuteWorkflow(ctx, workflowOptions, "WorkflowFunc", workflowDef)
if err != nil {
    return err
}

// we.GetID() - WorkflowID
// we.GetRunID() - Temporal生成的RunID
```

### Request Validation Strategy

**多层验证:**

1. **HTTP层验证** (Gin binding)
   - Content-Type检查
   - JSON格式验证
   - 必需字段检查

2. **YAML语法验证** (Story 1.3 Parser)
   - YAML格式正确性
   - Schema验证 (必需字段、类型)
   - 自定义规则 (uses格式、job名称等)

3. **业务逻辑验证**
   - WorkflowID唯一性检查 (可选,Temporal自动处理)
   - 工作流大小限制 (防止DoS)

**验证流程:**

```
Request → Gin Binding → YAML Parser → Temporal Client
           ↓              ↓              ↓
         400 Bad       422 Validation   500 Internal
         Request       Failed           Error
```

### Project Structure Updates

基于Story 1.1-1.4的结构,本Story新增:

```
internal/
├── server/handlers/
│   ├── workflow.go         # 工作流提交handler (新建)
│   └── workflow_test.go    # handler单元测试 (新建)
├── service/
│   ├── workflow_service.go # 工作流服务层 (新建)
│   └── workflow_service_test.go (新建)
├── models/
│   ├── request.go          # 请求/响应模型 (新建)
│   └── response.go         (新建)

api/
└── openapi.yaml            # 更新 - 添加POST /v1/workflows定义
```

**职责分层:**

- **Handler层** (`handlers/workflow.go`) - HTTP请求处理、参数绑定
- **Service层** (`service/workflow_service.go`) - 业务逻辑、调用Parser和Temporal
- **Model层** (`models/`) - 数据结构定义

## Tasks / Subtasks

### Task 1: 定义请求/响应数据模型 (AC: 返回工作流ID和提交状态)

- [ ] 1.1 创建`internal/models/request.go`
  ```go
  package models
  
  import "time"
  
  // SubmitWorkflowRequest 提交工作流请求
  type SubmitWorkflowRequest struct {
      Workflow string `json:"workflow" binding:"required"` // YAML内容
  }
  
  // SubmitWorkflowResponse 提交工作流响应
  type SubmitWorkflowResponse struct {
      WorkflowID  string    `json:"workflow_id"`
      RunID       string    `json:"run_id"`
      Status      string    `json:"status"`       // running, failed
      SubmittedAt time.Time `json:"submitted_at"`
  }
  ```

- [ ] 1.2 创建`internal/models/response.go`
  ```go
  package models
  
  // ErrorResponse RFC 7807错误响应
  type ErrorResponse struct {
      Type   string        `json:"type"`
      Title  string        `json:"title"`
      Status int           `json:"status"`
      Detail string        `json:"detail,omitempty"`
      Errors []FieldError  `json:"errors,omitempty"`
  }
  
  type FieldError struct {
      Field   string `json:"field"`
      Message string `json:"message"`
  }
  
  // NewBadRequestError 创建400错误
  func NewBadRequestError(detail string) *ErrorResponse {
      return &ErrorResponse{
          Type:   "https://waterflow.io/errors/bad-request",
          Title:  "Bad Request",
          Status: 400,
          Detail: detail,
      }
  }
  
  // NewValidationError 创建422错误
  func NewValidationError(errors []FieldError) *ErrorResponse {
      return &ErrorResponse{
          Type:   "https://waterflow.io/errors/validation-failed",
          Title:  "Validation Failed",
          Status: 422,
          Errors: errors,
      }
  }
  ```

### Task 2: 实现WorkflowID生成器 (AC: 工作流ID唯一且可追踪)

- [ ] 2.1 创建`internal/service/workflow_service.go`
  ```go
  package service
  
  import (
      "context"
      "fmt"
      "time"
      
      "github.com/google/uuid"
      "go.uber.org/zap"
      
      "waterflow/internal/parser"
      "waterflow/internal/temporal"
  )
  
  type WorkflowService struct {
      parser         *parser.Parser
      temporalClient *temporal.Client
      logger         *zap.Logger
  }
  
  func NewWorkflowService(p *parser.Parser, tc *temporal.Client, logger *zap.Logger) *WorkflowService {
      return &WorkflowService{
          parser:         p,
          temporalClient: tc,
          logger:         logger,
      }
  }
  
  // GenerateWorkflowID 生成唯一WorkflowID
  func (ws *WorkflowService) GenerateWorkflowID() string {
      return fmt.Sprintf("wf-%s", uuid.New().String())
  }
  ```

- [ ] 2.2 添加WorkflowID验证
  ```go
  // ValidateWorkflowID 验证WorkflowID格式
  func ValidateWorkflowID(id string) error {
      if len(id) < 5 || !strings.HasPrefix(id, "wf-") {
          return fmt.Errorf("invalid workflow ID format")
      }
      return nil
  }
  ```

### Task 3: 实现工作流提交服务层 (AC: 工作流提交到Temporal执行队列)

- [ ] 3.1 实现`SubmitWorkflow`方法
  ```go
  package service
  
  import (
      "context"
      "fmt"
      "time"
      
      "go.temporal.io/sdk/client"
      "go.uber.org/zap"
      
      "waterflow/internal/models"
      "waterflow/internal/parser"
  )
  
  // SubmitWorkflow 提交工作流到Temporal
  func (ws *WorkflowService) SubmitWorkflow(ctx context.Context, yamlContent string) (*models.SubmitWorkflowResponse, error) {
      // 1. 解析YAML
      wf, err := ws.parser.Parse(yamlContent)
      if err != nil {
          ws.logger.Error("Failed to parse workflow",
              zap.Error(err),
          )
          return nil, fmt.Errorf("parse error: %w", err)
      }
      
      // 2. 生成WorkflowID
      workflowID := ws.GenerateWorkflowID()
      
      // 3. 提交到Temporal
      workflowOptions := client.StartWorkflowOptions{
          ID:                 workflowID,
          TaskQueue:          "default", // MVP使用默认队列,Story 1.6将从DSL提取
          WorkflowRunTimeout: 1 * time.Hour,
      }
      
      // 注意: WorkflowFunc将在Story 1.6实现
      // 现在传递解析后的WorkflowDefinition
      we, err := ws.temporalClient.GetClient().ExecuteWorkflow(
          ctx,
          workflowOptions,
          "SimpleWorkflow", // 临时workflow名称,Story 1.6替换
          wf,
      )
      if err != nil {
          ws.logger.Error("Failed to submit workflow to Temporal",
              zap.Error(err),
              zap.String("workflow_id", workflowID),
          )
          return nil, fmt.Errorf("temporal submission error: %w", err)
      }
      
      // 4. 返回响应
      response := &models.SubmitWorkflowResponse{
          WorkflowID:  we.GetID(),
          RunID:       we.GetRunID(),
          Status:      "running",
          SubmittedAt: time.Now(),
      }
      
      ws.logger.Info("Workflow submitted successfully",
          zap.String("workflow_id", response.WorkflowID),
          zap.String("run_id", response.RunID),
      )
      
      return response, nil
  }
  ```

- [ ] 3.2 添加错误分类
  ```go
  // 区分不同类型的错误
  type WorkflowError struct {
      Type    string // "parse_error", "temporal_error", "validation_error"
      Message string
      Err     error
  }
  
  func (e *WorkflowError) Error() string {
      return fmt.Sprintf("%s: %s", e.Type, e.Message)
  }
  
  func (e *WorkflowError) Unwrap() error {
      return e.Err
  }
  ```

### Task 4: 实现HTTP Handler (AC: POST /v1/workflows请求处理)

- [ ] 4.1 创建`internal/server/handlers/workflow.go`
  ```go
  package handlers
  
  import (
      "net/http"
      "time"
      
      "github.com/gin-gonic/gin"
      "go.uber.org/zap"
      
      "waterflow/internal/models"
      "waterflow/internal/parser"
      "waterflow/internal/service"
  )
  
  type WorkflowHandler struct {
      workflowService *service.WorkflowService
      logger          *zap.Logger
  }
  
  func NewWorkflowHandler(ws *service.WorkflowService, logger *zap.Logger) *WorkflowHandler {
      return &WorkflowHandler{
          workflowService: ws,
          logger:          logger,
      }
  }
  
  // SubmitWorkflow - POST /v1/workflows
  func (h *WorkflowHandler) SubmitWorkflow(c *gin.Context) {
      var req models.SubmitWorkflowRequest
      
      // 1. 绑定请求体
      if err := c.ShouldBindJSON(&req); err != nil {
          c.JSON(http.StatusBadRequest, models.NewBadRequestError(
              "Invalid request format: " + err.Error(),
          ))
          return
      }
      
      // 2. 验证YAML大小限制
      const maxYAMLSize = 1 << 20 // 1MB
      if len(req.Workflow) > maxYAMLSize {
          c.JSON(http.StatusBadRequest, models.NewBadRequestError(
              fmt.Sprintf("Workflow YAML exceeds maximum size of %d bytes", maxYAMLSize),
          ))
          return
      }
      
      // 3. 提交工作流
      ctx := c.Request.Context()
      resp, err := h.workflowService.SubmitWorkflow(ctx, req.Workflow)
      
      if err != nil {
          h.handleSubmitError(c, err)
          return
      }
      
      // 4. 返回成功响应
      c.JSON(http.StatusCreated, resp)
  }
  
  // handleSubmitError 处理提交错误
  func (h *WorkflowHandler) handleSubmitError(c *gin.Context, err error) {
      // 根据错误类型返回不同HTTP状态码
      switch e := err.(type) {
      case *parser.ValidationError:
          // YAML验证错误 → 422
          errors := make([]models.FieldError, len(e.Fields))
          for i, f := range e.Fields {
              errors[i] = models.FieldError{
                  Field:   f.Field,
                  Message: f.Message,
              }
          }
          c.JSON(http.StatusUnprocessableEntity, models.NewValidationError(errors))
          
      case *parser.ParseError:
          // YAML语法错误 → 422
          c.JSON(http.StatusUnprocessableEntity, &models.ErrorResponse{
              Type:   "https://waterflow.io/errors/parse-error",
              Title:  "YAML Parse Error",
              Status: 422,
              Detail: e.Message,
          })
          
      default:
          // 其他错误 → 500
          h.logger.Error("Internal error during workflow submission",
              zap.Error(err),
          )
          c.JSON(http.StatusInternalServerError, &models.ErrorResponse{
              Type:   "https://waterflow.io/errors/internal-error",
              Title:  "Internal Server Error",
              Status: 500,
              Detail: "Failed to submit workflow",
          })
      }
  }
  ```

### Task 5: 注册路由端点 (集成到Router)

- [ ] 5.1 更新`internal/server/router.go`
  ```go
  func SetupRouter(logger *zap.Logger, tc *temporal.Client) *gin.Engine {
      router := gin.New()
      
      // ... 中间件配置 ...
      
      // 健康检查
      healthHandler := handlers.NewHealthHandler(tc)
      router.GET("/health", healthHandler.HealthCheck)
      router.GET("/ready", healthHandler.ReadinessCheck)
      
      // API v1
      v1 := router.Group("/v1")
      {
          v1.GET("/", handlers.APIVersionInfo)
          
          // Story 1.3: 验证端点
          validateHandler := handlers.NewValidateHandler()
          v1.POST("/validate", validateHandler.Validate)
          
          // Story 1.5: 工作流提交端点 (新增)
          parserInstance := parser.New()
          workflowService := service.NewWorkflowService(parserInstance, tc, logger)
          workflowHandler := handlers.NewWorkflowHandler(workflowService, logger)
          v1.POST("/workflows", workflowHandler.SubmitWorkflow)
      }
      
      return router
  }
  ```

- [ ] 5.2 优化依赖注入
  ```go
  // 在server.go中统一创建service
  type Server struct {
      config          *config.Config
      logger          *zap.Logger
      router          *gin.Engine
      httpServer      *http.Server
      temporalClient  *temporal.Client
      workflowService *service.WorkflowService // 新增
  }
  
  func New(cfg *config.Config, logger *zap.Logger, tc *temporal.Client) *Server {
      // 创建Parser
      parserInstance := parser.New()
      
      // 创建WorkflowService
      workflowService := service.NewWorkflowService(parserInstance, tc, logger)
      
      // 创建Router
      router := SetupRouter(logger, tc, workflowService)
      
      return &Server{
          config:          cfg,
          logger:          logger,
          temporalClient:  tc,
          workflowService: workflowService,
          router:          router,
          httpServer: &http.Server{
              Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
              Handler: router,
          },
      }
  }
  ```

### Task 6: 添加请求超时控制 (AC: 响应时间<500ms)

- [ ] 6.1 添加Context超时中间件
  ```go
  // internal/server/middleware/timeout.go
  package middleware
  
  import (
      "context"
      "time"
      
      "github.com/gin-gonic/gin"
  )
  
  // RequestTimeout 设置请求超时
  func RequestTimeout(timeout time.Duration) gin.HandlerFunc {
      return func(c *gin.Context) {
          ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
          defer cancel()
          
          c.Request = c.Request.WithContext(ctx)
          c.Next()
      }
  }
  ```

- [ ] 6.2 在router中应用超时
  ```go
  v1.Use(middleware.RequestTimeout(3 * time.Second))
  v1.POST("/workflows", workflowHandler.SubmitWorkflow)
  ```

- [ ] 6.3 在Service层检查Context
  ```go
  func (ws *WorkflowService) SubmitWorkflow(ctx context.Context, yamlContent string) (*models.SubmitWorkflowResponse, error) {
      // 检查Context是否已取消
      select {
      case <-ctx.Done():
          return nil, fmt.Errorf("request timeout: %w", ctx.Err())
      default:
      }
      
      // ... 正常逻辑 ...
  }
  ```

### Task 7: 添加单元测试和集成测试 (代码质量保障)

- [ ] 7.1 创建`internal/service/workflow_service_test.go`
  ```go
  package service
  
  import (
      "context"
      "testing"
      
      "github.com/stretchr/testify/assert"
      "github.com/stretchr/testify/mock"
      "go.uber.org/zap"
      
      "waterflow/internal/parser"
  )
  
  func TestGenerateWorkflowID(t *testing.T) {
      ws := &WorkflowService{}
      
      id1 := ws.GenerateWorkflowID()
      id2 := ws.GenerateWorkflowID()
      
      assert.NotEqual(t, id1, id2, "WorkflowIDs should be unique")
      assert.Contains(t, id1, "wf-")
  }
  
  func TestSubmitWorkflow_Success(t *testing.T) {
      // Mock Parser
      mockParser := &parser.Parser{}
      
      // Mock Temporal Client
      // (需要创建mock,或使用真实Temporal进行集成测试)
      
      ws := NewWorkflowService(mockParser, nil, zap.NewNop())
      
      yamlContent := `
  name: Test Workflow
  on: push
  jobs:
    test:
      runs-on: linux
      steps:
        - name: Test
          uses: run@v1
  `
      
      ctx := context.Background()
      resp, err := ws.SubmitWorkflow(ctx, yamlContent)
      
      assert.NoError(t, err)
      assert.NotEmpty(t, resp.WorkflowID)
      assert.Equal(t, "running", resp.Status)
  }
  
  func TestSubmitWorkflow_ParseError(t *testing.T) {
      mockParser := &parser.Parser{}
      ws := NewWorkflowService(mockParser, nil, zap.NewNop())
      
      invalidYAML := `invalid yaml content`
      
      ctx := context.Background()
      _, err := ws.SubmitWorkflow(ctx, invalidYAML)
      
      assert.Error(t, err)
      assert.Contains(t, err.Error(), "parse error")
  }
  ```

- [ ] 7.2 创建`internal/server/handlers/workflow_test.go`
  ```go
  package handlers
  
  import (
      "bytes"
      "encoding/json"
      "net/http"
      "net/http/httptest"
      "testing"
      
      "github.com/gin-gonic/gin"
      "github.com/stretchr/testify/assert"
      "go.uber.org/zap"
      
      "waterflow/internal/models"
  )
  
  func TestSubmitWorkflow_Success(t *testing.T) {
      // 设置测试环境
      gin.SetMode(gin.TestMode)
      
      // Mock WorkflowService
      // ... (需要mock实现)
      
      handler := NewWorkflowHandler(mockService, zap.NewNop())
      
      router := gin.New()
      router.POST("/workflows", handler.SubmitWorkflow)
      
      // 准备请求
      reqBody := models.SubmitWorkflowRequest{
          Workflow: "name: Test\non: push\njobs:\n  build:\n    runs-on: linux\n    steps:\n      - name: Test\n        uses: run@v1",
      }
      bodyBytes, _ := json.Marshal(reqBody)
      
      req := httptest.NewRequest("POST", "/workflows", bytes.NewReader(bodyBytes))
      req.Header.Set("Content-Type", "application/json")
      
      w := httptest.NewRecorder()
      router.ServeHTTP(w, req)
      
      // 验证响应
      assert.Equal(t, http.StatusCreated, w.Code)
      
      var resp models.SubmitWorkflowResponse
      json.Unmarshal(w.Body.Bytes(), &resp)
      assert.NotEmpty(t, resp.WorkflowID)
      assert.Equal(t, "running", resp.Status)
  }
  
  func TestSubmitWorkflow_MissingWorkflowField(t *testing.T) {
      handler := NewWorkflowHandler(nil, zap.NewNop())
      
      router := gin.New()
      router.POST("/workflows", handler.SubmitWorkflow)
      
      req := httptest.NewRequest("POST", "/workflows", bytes.NewReader([]byte("{}")))
      req.Header.Set("Content-Type", "application/json")
      
      w := httptest.NewRecorder()
      router.ServeHTTP(w, req)
      
      assert.Equal(t, http.StatusBadRequest, w.Code)
  }
  
  func TestSubmitWorkflow_InvalidYAML(t *testing.T) {
      // Mock service返回ParseError
      // ...
      
      router := gin.New()
      router.POST("/workflows", handler.SubmitWorkflow)
      
      reqBody := models.SubmitWorkflowRequest{
          Workflow: "invalid: yaml: content:",
      }
      bodyBytes, _ := json.Marshal(reqBody)
      
      req := httptest.NewRequest("POST", "/workflows", bytes.NewReader(bodyBytes))
      req.Header.Set("Content-Type", "application/json")
      
      w := httptest.NewRecorder()
      router.ServeHTTP(w, req)
      
      assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
  }
  ```

- [ ] 7.3 运行测试
  ```bash
  # 单元测试
  go test -v ./internal/service
  go test -v ./internal/server/handlers
  
  # 覆盖率
  go test -cover ./internal/service ./internal/server/handlers
  # 期望: >75%
  ```

### Task 8: 更新OpenAPI文档 (API文档化)

- [ ] 8.1 更新`api/openapi.yaml`
  ```yaml
  openapi: 3.0.0
  info:
    title: Waterflow API
    version: 1.0.0
    description: Declarative workflow orchestration engine
  
  servers:
    - url: http://localhost:8080
      description: Development server
  
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
          '503':
            description: Service is not ready
    
    /v1/validate:
      post:
        summary: Validate workflow YAML
        requestBody:
          required: true
          content:
            application/json:
              schema:
                type: object
                required:
                  - workflow
                properties:
                  workflow:
                    type: string
                    description: YAML workflow definition
        responses:
          '200':
            description: Validation successful
          '422':
            description: Validation failed
    
    /v1/workflows:
      post:
        summary: Submit workflow for execution
        requestBody:
          required: true
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/SubmitWorkflowRequest'
        responses:
          '201':
            description: Workflow submitted successfully
            content:
              application/json:
                schema:
                  $ref: '#/components/schemas/SubmitWorkflowResponse'
          '400':
            description: Bad request
            content:
              application/json:
                schema:
                  $ref: '#/components/schemas/ErrorResponse'
          '422':
            description: Validation failed
            content:
              application/json:
                schema:
                  $ref: '#/components/schemas/ValidationErrorResponse'
          '500':
            description: Internal server error
  
  components:
    schemas:
      SubmitWorkflowRequest:
        type: object
        required:
          - workflow
        properties:
          workflow:
            type: string
            description: YAML workflow definition
            example: |
              name: Deploy Application
              on: push
              jobs:
                deploy:
                  runs-on: linux-amd64
                  steps:
                    - name: Deploy
                      uses: run@v1
                      with:
                        command: echo "Deploying..."
      
      SubmitWorkflowResponse:
        type: object
        properties:
          workflow_id:
            type: string
            example: wf-550e8400-e29b-41d4-a716-446655440000
          run_id:
            type: string
            example: temporal-uuid-12345
          status:
            type: string
            enum: [running, failed]
            example: running
          submitted_at:
            type: string
            format: date-time
            example: 2025-12-16T10:30:00Z
      
      ErrorResponse:
        type: object
        properties:
          type:
            type: string
            example: https://waterflow.io/errors/bad-request
          title:
            type: string
            example: Bad Request
          status:
            type: integer
            example: 400
          detail:
            type: string
            example: Missing 'workflow' field in request body
      
      ValidationErrorResponse:
        type: object
        properties:
          type:
            type: string
            example: https://waterflow.io/errors/validation-failed
          title:
            type: string
            example: Validation Failed
          status:
            type: integer
            example: 422
          errors:
            type: array
            items:
              type: object
              properties:
                field:
                  type: string
                  example: name
                message:
                  type: string
                  example: Field 'name' is required
  ```

## Dev Notes

### Critical Implementation Guidelines

**1. 错误处理优先级**

```go
// ✅ 正确: 区分错误类型,返回合适的HTTP状态码
func (h *WorkflowHandler) handleSubmitError(c *gin.Context, err error) {
    switch e := err.(type) {
    case *parser.ValidationError:
        c.JSON(422, ...) // 验证错误
    case *parser.ParseError:
        c.JSON(422, ...) // 语法错误
    default:
        c.JSON(500, ...) // 内部错误
    }
}

// ❌ 错误: 所有错误返回500
func (h *WorkflowHandler) handleSubmitError(c *gin.Context, err error) {
    c.JSON(500, gin.H{"error": err.Error()})
}
```

**2. WorkflowID唯一性**

```go
// ✅ 使用UUID保证全局唯一
import "github.com/google/uuid"

workflowID := fmt.Sprintf("wf-%s", uuid.New().String())

// ❌ 避免: 时间戳可能冲突
workflowID := fmt.Sprintf("wf-%d", time.Now().Unix())
```

**3. Context传递**

```go
// ✅ 使用gin.Context中的Request.Context()
ctx := c.Request.Context()
resp, err := h.workflowService.SubmitWorkflow(ctx, req.Workflow)

// ❌ 创建新Context会丢失超时等信息
ctx := context.Background()
```

**4. 请求大小限制**

```go
// ✅ 限制YAML大小防止DoS
const maxYAMLSize = 1 << 20 // 1MB
if len(req.Workflow) > maxYAMLSize {
    return error
}

// 也可以在Gin层设置
router.Use(func(c *gin.Context) {
    c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)
    c.Next()
})
```

**5. 幂等性考虑**

```go
// Temporal自动处理WorkflowID唯一性
// 相同ID重复提交会返回错误
we, err := client.ExecuteWorkflow(ctx, options, workflowFunc, args)
if err != nil {
    // 检查是否为重复ID错误
    if strings.Contains(err.Error(), "workflow execution already started") {
        return &models.ErrorResponse{
            Status: 409, // Conflict
            Detail: "Workflow with this ID already exists",
        }
    }
}
```

**6. 日志记录最佳实践**

```go
// ✅ 记录关键字段
ws.logger.Info("Workflow submitted",
    zap.String("workflow_id", workflowID),
    zap.String("run_id", runID),
    zap.String("workflow_name", wf.Name),
    zap.Int("job_count", len(wf.Jobs)),
)

// ❌ 避免: 记录敏感信息或大量数据
ws.logger.Info("Workflow submitted", zap.String("yaml", yamlContent)) // 太大!
```

**7. 性能优化**

```go
// ✅ 并发安全的Parser实例复用
type WorkflowService struct {
    parser *parser.Parser // 创建一次,重复使用
}

// ❌ 每次请求创建Parser
func (ws *WorkflowService) SubmitWorkflow(...) {
    p := parser.New() // 浪费资源
}
```

### Integration with Previous Stories

**与Story 1.3 YAML解析器集成:**

```go
// Story 1.3提供的解析器
parser := parser.New()
wf, err := parser.Parse(yamlContent)

// Story 1.5使用解析结果
if err != nil {
    // 返回422错误
    return models.NewValidationError(...)
}
```

**与Story 1.4 Temporal Client集成:**

```go
// Story 1.4提供的Client
temporalClient.GetClient().ExecuteWorkflow(ctx, options, workflowFunc, args)

// Story 1.5提交工作流
we, err := temporalClient.GetClient().ExecuteWorkflow(...)
workflowID := we.GetID()
runID := we.GetRunID()
```

**为Story 1.6准备:**

```go
// Story 1.5传递解析后的WorkflowDefinition
we, err := client.ExecuteWorkflow(ctx, options, "WorkflowFunc", wf)

// Story 1.6将实现真实的WorkflowFunc
func WorkflowFunc(ctx workflow.Context, wf *parser.WorkflowDefinition) error {
    // 执行jobs和steps
}
```

### Testing Strategy

**单元测试覆盖:**

| 组件 | 测试场景 |
|------|---------|
| WorkflowService | WorkflowID生成、YAML解析、Temporal提交 |
| WorkflowHandler | 请求绑定、错误处理、响应格式 |
| Models | 数据序列化/反序列化 |

**集成测试:**

```bash
# 1. 启动Temporal
make dev-env

# 2. 启动Waterflow Server
go run ./cmd/server

# 3. 提交工作流
curl -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "workflow": "name: Test\non: push\njobs:\n  build:\n    runs-on: linux\n    steps:\n      - name: Build\n        uses: run@v1\n        with:\n          command: echo Building"
  }'

# 期望响应:
# {
#   "workflow_id": "wf-xxx",
#   "run_id": "yyy",
#   "status": "running",
#   "submitted_at": "2025-12-16T10:30:00Z"
# }

# 4. 在Temporal UI查看
# http://localhost:8233
```

**性能测试:**

```bash
# 使用Apache Bench
ab -n 100 -c 10 -p workflow.json -T application/json \
   http://localhost:8080/v1/workflows

# 期望: p95 < 500ms
```

### References

**架构设计:**
- [docs/architecture.md §3.1.1](docs/architecture.md) - REST API Handler设计
- [docs/architecture.md §2.2](docs/architecture.md) - 工作流提交流程

**技术文档:**
- [RFC 7807: Problem Details](https://tools.ietf.org/html/rfc7807)
- [Temporal ExecuteWorkflow API](https://pkg.go.dev/go.temporal.io/sdk/client#Client.ExecuteWorkflow)
- [Gin Binding and Validation](https://gin-gonic.com/docs/examples/binding-and-validation/)

**项目上下文:**
- [docs/epics.md Story 1.1-1.4](docs/epics.md) - 前置Stories
- [docs/epics.md Story 1.6](docs/epics.md) - 工作流执行引擎 (处理提交后的执行)

**Go规范:**
- [Uber Go Style Guide: Error Handling](https://github.com/uber-go/guide/blob/master/style.md#error-handling)
- [Effective Go: Errors](https://go.dev/doc/effective_go#errors)

### Dependency Graph

```
Story 1.1 (框架)
    ↓
Story 1.2 (REST API)
    ↓
Story 1.3 (YAML解析) ──┐
    ↓                  ↓
Story 1.4 (Temporal) ──┐
    ↓                  ↓
Story 1.5 (工作流提交API) ← 当前Story
    ↓
Story 1.6 (执行引擎) - 实现真实的Temporal Workflow函数
    ↓
Story 1.7 (状态查询) - 查询提交后的工作流状态
```

## Dev Agent Record

### Context Reference

**Source Documents Analyzed:**
1. [docs/epics.md](docs/epics.md) (lines 345-360) - Story 1.5需求定义
2. [docs/architecture.md](docs/architecture.md) (§3.1.1) - REST API Handler设计
3. Story 1.3文档 - YAML解析器API
4. Story 1.4文档 - Temporal Client API

**Previous Stories:**
- Story 1.1: 项目框架 (drafted)
- Story 1.2: REST API框架 (drafted)
- Story 1.3: YAML解析器 (drafted)
- Story 1.4: Temporal SDK集成 (drafted)

### Agent Model Used

Claude 3.5 Sonnet (BMM Scrum Master Agent - Bob)

### Estimated Effort

**开发时间:** 6-8小时  
**复杂度:** 中等

**时间分解:**
- 数据模型定义: 1小时
- WorkflowID生成器: 0.5小时
- 工作流服务层: 2小时
- HTTP Handler实现: 1.5小时
- 路由集成: 0.5小时
- 超时控制: 0.5小时
- 单元测试: 2小时
- OpenAPI文档更新: 1小时

**技能要求:**
- Gin框架进阶 (错误处理、Context)
- Temporal Client API
- REST API设计最佳实践
- 错误分类和HTTP状态码

### Debug Log References

<!-- Will be populated during implementation -->

### Completion Notes List

<!-- Developer填写完成时的笔记 -->

### File List

**预期创建/修改的文件清单:**

```
新建文件 (~10个):
├── internal/models/
│   ├── request.go                  # 请求模型
│   └── response.go                 # 响应模型
├── internal/service/
│   ├── workflow_service.go         # 工作流服务层
│   └── workflow_service_test.go    # 服务层测试
├── internal/server/handlers/
│   ├── workflow.go                 # 工作流handler
│   └── workflow_test.go            # handler测试
├── internal/server/middleware/
│   └── timeout.go                  # 超时中间件

修改文件 (~3个):
├── internal/server/server.go       # 集成WorkflowService
├── internal/server/router.go       # 注册/v1/workflows端点
└── api/openapi.yaml                # 添加API文档
```

**关键代码片段:**

**workflow_service.go (核心):**
```go
package service

type WorkflowService struct {
    parser         *parser.Parser
    temporalClient *temporal.Client
    logger         *zap.Logger
}

func (ws *WorkflowService) SubmitWorkflow(ctx context.Context, yamlContent string) (*models.SubmitWorkflowResponse, error) {
    // 1. 解析YAML
    wf, err := ws.parser.Parse(yamlContent)
    if err != nil {
        return nil, fmt.Errorf("parse error: %w", err)
    }
    
    // 2. 生成WorkflowID
    workflowID := fmt.Sprintf("wf-%s", uuid.New().String())
    
    // 3. 提交到Temporal
    we, err := ws.temporalClient.GetClient().ExecuteWorkflow(
        ctx,
        client.StartWorkflowOptions{
            ID:        workflowID,
            TaskQueue: "default",
        },
        "SimpleWorkflow",
        wf,
    )
    
    // 4. 返回响应
    return &models.SubmitWorkflowResponse{
        WorkflowID:  we.GetID(),
        RunID:       we.GetRunID(),
        Status:      "running",
        SubmittedAt: time.Now(),
    }, nil
}
```

**workflow.go (Handler):**
```go
package handlers

func (h *WorkflowHandler) SubmitWorkflow(c *gin.Context) {
    var req models.SubmitWorkflowRequest
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, models.NewBadRequestError(err.Error()))
        return
    }
    
    resp, err := h.workflowService.SubmitWorkflow(c.Request.Context(), req.Workflow)
    if err != nil {
        h.handleSubmitError(c, err)
        return
    }
    
    c.JSON(201, resp)
}
```

---

**Story Ready for Development** ✅

开发者可基于Story 1.1-1.4的成果,实现工作流提交API。
本Story是Waterflow MVP的关键里程碑,首次实现端到端的工作流提交流程。
