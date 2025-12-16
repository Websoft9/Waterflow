# Story 1.3: YAML DSL 解析器

Status: drafted

## Story

As a **工作流用户**,  
I want **提交 YAML 格式的工作流定义**,  
So that **系统可以解析并验证工作流语法**。

## Acceptance Criteria

**Given** 一个符合规范的 YAML 工作流文件  
**When** 通过 API 提交工作流内容  
**Then** 系统成功解析 YAML 结构 (name, jobs, steps)  
**And** 验证必需字段存在 (job name, runs-on, steps)  
**And** 语法错误返回具体错误位置 (行号、字段名)  
**And** 支持基本字段: jobs, steps, runs-on, uses, with  
**And** 解析结果转换为内部数据结构

## Technical Context

### Architecture Constraints

根据 [docs/architecture.md](docs/architecture.md) §3.1.2 YAML Parser设计:

1. **核心职责**
   - YAML语法解析 (使用 `gopkg.in/yaml.v3`)
   - Schema验证 (必需字段、类型检查、约束验证)
   - 数据结构转换 (YAML → Go struct)
   - 错误诊断 (语法错误定位到行号)

2. **输入/输出**
   - **输入:** YAML字符串 (工作流定义)
   - **输出:** `WorkflowDefinition` Go结构体 或 验证错误列表

3. **关键设计约束** (参考 ADR-0004)
   - 语法必须与GitHub Actions兼容
   - 支持核心字段: `name`, `on`, `jobs`, `steps`, `uses`, `with`, `runs-on`
   - MVP不支持: matrix策略, container, services, outputs传递

### Dependencies

**前置Story:**
- ✅ Story 1.1: Waterflow Server框架搭建
- ✅ Story 1.2: REST API服务框架
  - 使用: HTTP端点接收YAML内容
  - 集成: 解析器作为中间件/服务层

**后续Story依赖本Story:**
- Story 1.5: 工作流提交API - 需要解析器验证YAML
- Story 1.6: 工作流执行引擎 - 使用解析后的结构体

### Technology Stack

**YAML解析库: gopkg.in/yaml.v3**

选择理由:
- **官方推荐:** Go生态标准YAML库
- **详细错误:** 提供行号、列号定位
- **灵活性:** 支持自定义UnmarshalYAML
- **性能:** 纯Go实现,性能优异

```bash
go get gopkg.in/yaml.v3
```

**Schema验证库: go-playground/validator/v10**

用于结构体字段验证:
```go
type Job struct {
    RunsOn  string `yaml:"runs-on" validate:"required"`
    Steps   []Step `yaml:"steps" validate:"required,min=1,dive"`
}
```

验证规则:
- `required` - 必需字段
- `min=1` - 数组至少1个元素
- `dive` - 递归验证数组元素
- `oneof` - 枚举值验证

```bash
go get github.com/go-playground/validator/v10
```

### YAML DSL Specification (基于ADR-0004)

**最小可用工作流示例:**

```yaml
name: Build and Test

on: push  # MVP简化版,仅支持字符串

jobs:
  build:
    runs-on: linux-amd64  # Task Queue名称
    timeout-minutes: 30
    
    steps:
      - name: Checkout Code
        uses: checkout@v1
        with:
          repository: ${{ workflow.repository }}
      
      - name: Run Tests
        uses: run@v1
        with:
          command: go test ./...
        timeout-minutes: 10
```

**核心字段规范:**

1. **Workflow级别**
   - `name` (必需): string - 工作流名称
   - `on` (必需): string - 触发事件 (MVP仅支持 "push")
   - `jobs` (必需): map[string]Job - 任务列表

2. **Job级别**
   - `runs-on` (必需): string - Task Queue名称 (如 "linux-amd64", "windows-x64")
   - `steps` (必需): []Step - 步骤列表 (至少1个)
   - `timeout-minutes` (可选): int - 超时时间 (默认60分钟)
   - `needs` (可选): []string - 依赖的Job ID (MVP暂不实现)

3. **Step级别**
   - `name` (必需): string - 步骤名称
   - `uses` (必需): string - 节点类型 (格式: `node-name@version`)
   - `with` (可选): map[string]interface{} - 节点参数
   - `timeout-minutes` (可选): int - 步骤超时 (默认10分钟)
   - `if` (可选): string - 条件表达式 (MVP暂不实现)

**验证规则:**

```go
// Workflow结构体
type WorkflowDefinition struct {
    Name string           `yaml:"name" validate:"required,min=1"`
    On   string           `yaml:"on" validate:"required,oneof=push schedule webhook"`
    Jobs map[string]Job   `yaml:"jobs" validate:"required,min=1,dive"`
}

// Job结构体
type Job struct {
    RunsOn         string `yaml:"runs-on" validate:"required"`
    Steps          []Step `yaml:"steps" validate:"required,min=1,dive"`
    TimeoutMinutes int    `yaml:"timeout-minutes" validate:"omitempty,min=1,max=1440"`
}

// Step结构体
type Step struct {
    Name           string                 `yaml:"name" validate:"required"`
    Uses           string                 `yaml:"uses" validate:"required,node_format"`
    With           map[string]interface{} `yaml:"with" validate:"omitempty"`
    TimeoutMinutes int                    `yaml:"timeout-minutes" validate:"omitempty,min=1,max=120"`
}
```

**自定义验证器:**

```go
// 验证 uses 字段格式: node-name@version
func validateNodeFormat(fl validator.FieldLevel) bool {
    uses := fl.Field().String()
    // 正则: ^[a-z][a-z0-9-]*@v[0-9]+$
    // 示例: checkout@v1, run@v1, notify@v2
    matched, _ := regexp.MatchString(`^[a-z][a-z0-9-]*@v[0-9]+$`, uses)
    return matched
}
```

### Project Structure Updates

基于Story 1.1-1.2的结构,本Story新增:

```
internal/
├── parser/
│   ├── parser.go           # 解析器主逻辑 (新建)
│   ├── validator.go        # Schema验证器 (新建)
│   ├── types.go            # WorkflowDefinition结构体 (新建)
│   ├── errors.go           # 错误类型定义 (新建)
│   └── parser_test.go      # 解析器单元测试 (新建)
├── server/handlers/
│   └── validate.go         # 验证端点handler (新建)

pkg/
└── models/
    └── workflow.go         # 工作流数据模型 (新建,可能与internal/parser/types.go合并)

test/
└── testdata/
    └── workflows/          # 测试用YAML文件 (新建)
        ├── valid_minimal.yaml
        ├── valid_complex.yaml
        ├── invalid_missing_name.yaml
        ├── invalid_empty_steps.yaml
        └── invalid_bad_uses_format.yaml
```

## Tasks / Subtasks

### Task 1: 定义工作流数据结构 (AC: 解析结果转换为内部数据结构)

- [ ] 1.1 创建`internal/parser/types.go`
  ```go
  package parser
  
  type WorkflowDefinition struct {
      Name string           `yaml:"name"`
      On   string           `yaml:"on"`
      Jobs map[string]Job   `yaml:"jobs"`
  }
  
  type Job struct {
      RunsOn         string                 `yaml:"runs-on"`
      Steps          []Step                 `yaml:"steps"`
      TimeoutMinutes int                    `yaml:"timeout-minutes,omitempty"`
  }
  
  type Step struct {
      Name           string                 `yaml:"name"`
      Uses           string                 `yaml:"uses"`
      With           map[string]interface{} `yaml:"with,omitempty"`
      TimeoutMinutes int                    `yaml:"timeout-minutes,omitempty"`
  }
  ```

- [ ] 1.2 添加验证标签
  ```go
  type WorkflowDefinition struct {
      Name string           `yaml:"name" validate:"required,min=1"`
      On   string           `yaml:"on" validate:"required,oneof=push schedule webhook"`
      Jobs map[string]Job   `yaml:"jobs" validate:"required,min=1,dive,keys,job_name,endkeys"`
  }
  
  type Job struct {
      RunsOn         string `yaml:"runs-on" validate:"required"`
      Steps          []Step `yaml:"steps" validate:"required,min=1,dive"`
      TimeoutMinutes int    `yaml:"timeout-minutes" validate:"omitempty,min=1,max=1440"`
  }
  
  type Step struct {
      Name           string                 `yaml:"name" validate:"required"`
      Uses           string                 `yaml:"uses" validate:"required,node_format"`
      With           map[string]interface{} `yaml:"with"`
      TimeoutMinutes int                    `yaml:"timeout-minutes" validate:"omitempty,min=1,max=120"`
  }
  ```

- [ ] 1.3 测试YAML反序列化
  ```bash
  # 验证YAML → Go struct映射
  go test -v ./internal/parser -run TestUnmarshalYAML
  ```

### Task 2: 实现YAML解析器 (AC: 系统成功解析YAML结构)

- [ ] 2.1 创建`internal/parser/parser.go`
  ```go
  package parser
  
  import (
      "fmt"
      "gopkg.in/yaml.v3"
  )
  
  type Parser struct {
      validator *validator.Validate
  }
  
  func New() *Parser {
      v := validator.New()
      // 注册自定义验证器
      v.RegisterValidation("node_format", validateNodeFormat)
      v.RegisterValidation("job_name", validateJobName)
      
      return &Parser{validator: v}
  }
  
  // Parse 解析YAML字符串为WorkflowDefinition
  func (p *Parser) Parse(yamlContent string) (*WorkflowDefinition, error) {
      var wf WorkflowDefinition
      
      // 1. YAML解析
      if err := yaml.Unmarshal([]byte(yamlContent), &wf); err != nil {
          return nil, NewParseError(err)
      }
      
      // 2. Schema验证
      if err := p.validator.Struct(&wf); err != nil {
          return nil, NewValidationError(err)
      }
      
      return &wf, nil
  }
  ```

- [ ] 2.2 实现ParseFile方法 (可选)
  ```go
  func (p *Parser) ParseFile(filePath string) (*WorkflowDefinition, error) {
      content, err := os.ReadFile(filePath)
      if err != nil {
          return nil, fmt.Errorf("failed to read file: %w", err)
      }
      return p.Parse(string(content))
  }
  ```

- [ ] 2.3 测试基本解析功能
  ```go
  func TestParseValidWorkflow(t *testing.T) {
      p := New()
      yaml := `
  name: Test Workflow
  on: push
  jobs:
    build:
      runs-on: linux-amd64
      steps:
        - name: Checkout
          uses: checkout@v1
  `
      wf, err := p.Parse(yaml)
      assert.NoError(t, err)
      assert.Equal(t, "Test Workflow", wf.Name)
      assert.Len(t, wf.Jobs, 1)
  }
  ```

### Task 3: 实现Schema验证器 (AC: 验证必需字段存在)

- [ ] 3.1 创建`internal/parser/validator.go`
  ```go
  package parser
  
  import (
      "fmt"
      "regexp"
      "github.com/go-playground/validator/v10"
  )
  
  // validateNodeFormat 验证 uses 字段格式
  func validateNodeFormat(fl validator.FieldLevel) bool {
      uses := fl.Field().String()
      // 格式: node-name@version (例如 checkout@v1)
      matched, _ := regexp.MatchString(`^[a-z][a-z0-9-]*@v[0-9]+$`, uses)
      return matched
  }
  
  // validateJobName 验证 Job ID格式
  func validateJobName(fl validator.FieldLevel) bool {
      name := fl.Field().String()
      // 格式: 小写字母开头,允许字母数字和连字符
      matched, _ := regexp.MatchString(`^[a-z][a-z0-9-]*$`, name)
      return matched
  }
  ```

- [ ] 3.2 实现验证错误友好化
  ```go
  // 将 validator.ValidationErrors 转换为用户友好消息
  func FormatValidationError(err error) []string {
      var messages []string
      
      if validationErrs, ok := err.(validator.ValidationErrors); ok {
          for _, e := range validationErrs {
              msg := formatFieldError(e)
              messages = append(messages, msg)
          }
      }
      
      return messages
  }
  
  func formatFieldError(e validator.FieldError) string {
      switch e.Tag() {
      case "required":
          return fmt.Sprintf("Field '%s' is required", e.Field())
      case "min":
          return fmt.Sprintf("Field '%s' must have at least %s items", e.Field(), e.Param())
      case "node_format":
          return fmt.Sprintf("Field '%s' must match format 'name@version' (got: %s)", e.Field(), e.Value())
      default:
          return fmt.Sprintf("Field '%s' validation failed: %s", e.Field(), e.Tag())
      }
  }
  ```

- [ ] 3.3 测试验证规则
  ```go
  func TestValidation_MissingName(t *testing.T) {
      p := New()
      yaml := `
  on: push
  jobs:
    build:
      runs-on: linux
      steps:
        - name: Test
          uses: run@v1
  `
      _, err := p.Parse(yaml)
      assert.Error(t, err)
      assert.Contains(t, err.Error(), "name")
  }
  
  func TestValidation_InvalidUsesFormat(t *testing.T) {
      p := New()
      yaml := `
  name: Test
  on: push
  jobs:
    build:
      runs-on: linux
      steps:
        - name: Test
          uses: INVALID_FORMAT  # 应该是 node@v1
  `
      _, err := p.Parse(yaml)
      assert.Error(t, err)
      assert.Contains(t, err.Error(), "node_format")
  }
  ```

### Task 4: 实现错误诊断系统 (AC: 语法错误返回具体错误位置)

- [ ] 4.1 创建`internal/parser/errors.go`
  ```go
  package parser
  
  import (
      "fmt"
      "gopkg.in/yaml.v3"
  )
  
  // ParseError YAML语法错误
  type ParseError struct {
      Line    int
      Column  int
      Message string
  }
  
  func (e *ParseError) Error() string {
      return fmt.Sprintf("YAML syntax error at line %d, column %d: %s", 
          e.Line, e.Column, e.Message)
  }
  
  // NewParseError 从yaml.v3错误提取行号
  func NewParseError(err error) error {
      if yamlErr, ok := err.(*yaml.TypeError); ok {
          // yaml.TypeError包含详细错误信息
          return &ParseError{
              Message: yamlErr.Error(),
          }
      }
      
      // 尝试从错误消息解析行号
      // yaml.v3 错误格式: "yaml: line 5: ..."
      return &ParseError{
          Message: err.Error(),
      }
  }
  
  // ValidationError Schema验证错误
  type ValidationError struct {
      Fields []FieldError
  }
  
  type FieldError struct {
      Field   string
      Message string
  }
  
  func (e *ValidationError) Error() string {
      if len(e.Fields) == 1 {
          return fmt.Sprintf("Validation error: %s", e.Fields[0].Message)
      }
      return fmt.Sprintf("Validation failed with %d errors", len(e.Fields))
  }
  
  // NewValidationError 转换validator错误
  func NewValidationError(err error) error {
      validationErrs, ok := err.(validator.ValidationErrors)
      if !ok {
          return err
      }
      
      fields := make([]FieldError, 0, len(validationErrs))
      for _, e := range validationErrs {
          fields = append(fields, FieldError{
              Field:   e.Field(),
              Message: formatFieldError(e),
          })
      }
      
      return &ValidationError{Fields: fields}
  }
  ```

- [ ] 4.2 实现JSON格式错误响应
  ```go
  // 用于HTTP响应的错误格式
  type ErrorResponse struct {
      Type    string       `json:"type"`
      Title   string       `json:"title"`
      Status  int          `json:"status"`
      Errors  []ErrorDetail `json:"errors,omitempty"`
  }
  
  type ErrorDetail struct {
      Field   string `json:"field,omitempty"`
      Line    int    `json:"line,omitempty"`
      Column  int    `json:"column,omitempty"`
      Message string `json:"message"`
  }
  ```

- [ ] 4.3 测试错误定位准确性
  ```go
  func TestParseError_LineNumber(t *testing.T) {
      p := New()
      yaml := `
  name: Test
  on: push
  jobs:
    build:
      runs-on: [invalid yaml structure
  `
      _, err := p.Parse(yaml)
      assert.Error(t, err)
      
      parseErr, ok := err.(*ParseError)
      assert.True(t, ok)
      assert.Greater(t, parseErr.Line, 0)
  }
  ```

### Task 5: 集成REST API端点 (AC: 通过API提交工作流内容)

- [ ] 5.1 创建`internal/server/handlers/validate.go`
  ```go
  package handlers
  
  import (
      "net/http"
      "github.com/gin-gonic/gin"
      "waterflow/internal/parser"
  )
  
  type ValidateHandler struct {
      parser *parser.Parser
  }
  
  func NewValidateHandler() *ValidateHandler {
      return &ValidateHandler{
          parser: parser.New(),
      }
  }
  
  // POST /v1/validate
  func (h *ValidateHandler) Validate(c *gin.Context) {
      // 1. 读取请求体
      var req struct {
          Workflow string `json:"workflow" binding:"required"`
      }
      
      if err := c.ShouldBindJSON(&req); err != nil {
          c.JSON(http.StatusBadRequest, gin.H{
              "error": "Missing 'workflow' field",
          })
          return
      }
      
      // 2. 解析YAML
      wf, err := h.parser.Parse(req.Workflow)
      if err != nil {
          // 返回验证错误
          c.JSON(http.StatusBadRequest, formatParseError(err))
          return
      }
      
      // 3. 返回解析结果摘要
      c.JSON(http.StatusOK, gin.H{
          "valid": true,
          "summary": gin.H{
              "name":     wf.Name,
              "on":       wf.On,
              "job_count": len(wf.Jobs),
          },
      })
  }
  
  func formatParseError(err error) gin.H {
      // 根据错误类型返回不同格式
      switch e := err.(type) {
      case *parser.ParseError:
          return gin.H{
              "valid": false,
              "error": e.Message,
              "line":  e.Line,
          }
      case *parser.ValidationError:
          return gin.H{
              "valid":  false,
              "errors": e.Fields,
          }
      default:
          return gin.H{
              "valid": false,
              "error": err.Error(),
          }
      }
  }
  ```

- [ ] 5.2 在`internal/server/router.go`注册端点
  ```go
  func SetupRouter(logger *zap.Logger) *gin.Engine {
      // ... 现有代码 ...
      
      v1 := router.Group("/v1")
      {
          // Story 1.2的端点
          v1.GET("/", handlers.APIVersionInfo)
          
          // Story 1.3新增: 验证端点
          validateHandler := handlers.NewValidateHandler()
          v1.POST("/validate", validateHandler.Validate)
      }
      
      return router
  }
  ```

- [ ] 5.3 测试API端点
  ```bash
  # 测试有效YAML
  curl -X POST http://localhost:8080/v1/validate \
    -H "Content-Type: application/json" \
    -d '{
      "workflow": "name: Test\non: push\njobs:\n  build:\n    runs-on: linux\n    steps:\n      - name: Test\n        uses: run@v1"
    }'
  
  # 期望响应:
  # {"valid":true,"summary":{"name":"Test","on":"push","job_count":1}}
  
  # 测试无效YAML (缺少name)
  curl -X POST http://localhost:8080/v1/validate \
    -H "Content-Type: application/json" \
    -d '{
      "workflow": "on: push\njobs:\n  build:\n    runs-on: linux\n    steps:\n      - name: Test\n        uses: run@v1"
    }'
  
  # 期望响应:
  # {"valid":false,"errors":[{"field":"Name","message":"Field 'Name' is required"}]}
  ```

### Task 6: 创建测试数据集 (确保覆盖各种场景)

- [ ] 6.1 创建`test/testdata/workflows/valid_minimal.yaml`
  ```yaml
  name: Minimal Workflow
  on: push
  jobs:
    test:
      runs-on: linux-amd64
      steps:
        - name: Run Test
          uses: run@v1
          with:
            command: echo "Hello"
  ```

- [ ] 6.2 创建`test/testdata/workflows/valid_complex.yaml`
  ```yaml
  name: Complex Workflow
  on: push
  jobs:
    build:
      runs-on: linux-amd64
      timeout-minutes: 30
      steps:
        - name: Checkout
          uses: checkout@v1
          with:
            repository: websoft9/waterflow
        
        - name: Build
          uses: run@v1
          with:
            command: make build
          timeout-minutes: 10
    
    test:
      runs-on: linux-amd64
      steps:
        - name: Run Tests
          uses: run@v1
          with:
            command: go test ./...
  ```

- [ ] 6.3 创建无效测试用例
  ```yaml
  # invalid_missing_name.yaml
  on: push
  jobs:
    build:
      runs-on: linux
      steps:
        - name: Test
          uses: run@v1
  
  # invalid_empty_steps.yaml
  name: Empty Steps
  on: push
  jobs:
    build:
      runs-on: linux
      steps: []  # 空数组,应该报错
  
  # invalid_bad_uses_format.yaml
  name: Bad Uses
  on: push
  jobs:
    build:
      runs-on: linux
      steps:
        - name: Test
          uses: InvalidFormat  # 缺少@version
  ```

- [ ] 6.4 使用testdata编写表驱动测试
  ```go
  func TestParser_Testdata(t *testing.T) {
      p := New()
      
      testCases := []struct {
          file      string
          shouldErr bool
      }{
          {"test/testdata/workflows/valid_minimal.yaml", false},
          {"test/testdata/workflows/valid_complex.yaml", false},
          {"test/testdata/workflows/invalid_missing_name.yaml", true},
          {"test/testdata/workflows/invalid_empty_steps.yaml", true},
          {"test/testdata/workflows/invalid_bad_uses_format.yaml", true},
      }
      
      for _, tc := range testCases {
          t.Run(tc.file, func(t *testing.T) {
              _, err := p.ParseFile(tc.file)
              if tc.shouldErr {
                  assert.Error(t, err)
              } else {
                  assert.NoError(t, err)
              }
          })
      }
  }
  ```

### Task 7: 添加单元测试和集成测试 (代码质量保障)

- [ ] 7.1 创建`internal/parser/parser_test.go`
  - 测试Parse方法各种输入
  - 测试ParseFile方法
  - 测试错误格式化

- [ ] 7.2 创建`internal/server/handlers/validate_test.go`
  ```go
  func TestValidateHandler_Success(t *testing.T) {
      router := gin.New()
      handler := NewValidateHandler()
      router.POST("/validate", handler.Validate)
      
      yaml := `name: Test
  on: push
  jobs:
    build:
      runs-on: linux
      steps:
        - name: Test
          uses: run@v1`
      
      req := httptest.NewRequest("POST", "/validate", strings.NewReader(
          fmt.Sprintf(`{"workflow":"%s"}`, escapeJSON(yaml)),
      ))
      req.Header.Set("Content-Type", "application/json")
      
      w := httptest.NewRecorder()
      router.ServeHTTP(w, req)
      
      assert.Equal(t, 200, w.Code)
      assert.Contains(t, w.Body.String(), `"valid":true`)
  }
  
  func TestValidateHandler_InvalidYAML(t *testing.T) {
      router := gin.New()
      handler := NewValidateHandler()
      router.POST("/validate", handler.Validate)
      
      yaml := `on: push`  // 缺少name
      
      req := httptest.NewRequest("POST", "/validate", strings.NewReader(
          fmt.Sprintf(`{"workflow":"%s"}`, escapeJSON(yaml)),
      ))
      req.Header.Set("Content-Type", "application/json")
      
      w := httptest.NewRecorder()
      router.ServeHTTP(w, req)
      
      assert.Equal(t, 400, w.Code)
      assert.Contains(t, w.Body.String(), `"valid":false`)
  }
  ```

- [ ] 7.3 运行测试并验证覆盖率
  ```bash
  make test
  # 期望: 所有测试通过
  # 覆盖率: internal/parser >85%
  
  go test -v ./internal/parser -coverprofile=coverage.out
  go tool cover -html=coverage.out
  ```

### Task 8: 添加性能基准测试 (可选)

- [ ] 8.1 创建`internal/parser/parser_bench_test.go`
  ```go
  func BenchmarkParser_Parse(b *testing.B) {
      p := New()
      yaml := loadTestWorkflow("valid_complex.yaml")
      
      b.ResetTimer()
      for i := 0; i < b.N; i++ {
          _, _ = p.Parse(yaml)
      }
  }
  
  func BenchmarkParser_ParseLarge(b *testing.B) {
      p := New()
      // 生成大型工作流 (100个jobs)
      yaml := generateLargeWorkflow(100)
      
      b.ResetTimer()
      for i := 0; i < b.N; i++ {
          _, _ = p.Parse(yaml)
      }
  }
  ```

- [ ] 8.2 运行基准测试
  ```bash
  go test -bench=. -benchmem ./internal/parser
  # 期望性能指标:
  # - 小型工作流 (<10 jobs): <1ms
  # - 中型工作流 (10-50 jobs): <5ms
  # - 大型工作流 (>100 jobs): <20ms
  ```

## Dev Notes

### Critical Implementation Guidelines

**1. YAML陷阱注意事项**

```yaml
# ❌ 错误: on会被解析为布尔值true
on: yes

# ✅ 正确: 使用引号
on: "yes"

# ❌ 错误: 版本号1.0会被解析为浮点数
version: 1.0

# ✅ 正确: 使用引号
version: "1.0"
```

在Go侧处理:
```go
// 使用yaml.v3的严格模式
decoder := yaml.NewDecoder(reader)
decoder.KnownFields(true)  // 未知字段报错
```

**2. 验证器性能优化**

```go
// ❌ 低效: 每次Parse创建新validator
func (p *Parser) Parse(yaml string) (*WorkflowDefinition, error) {
    v := validator.New()  // 慢!
    v.RegisterValidation("node_format", validateNodeFormat)
    // ...
}

// ✅ 高效: 在New()时创建一次
func New() *Parser {
    v := validator.New()
    v.RegisterValidation("node_format", validateNodeFormat)
    return &Parser{validator: v}
}
```

**3. 错误消息国际化 (可选)**

```go
// 支持中英文错误消息
type ErrorMessages struct {
    Lang string
}

func (em *ErrorMessages) Format(err validator.FieldError) string {
    if em.Lang == "zh" {
        return formatZH(err)
    }
    return formatEN(err)
}
```

**4. 内存优化 - 避免重复解析**

```go
// 缓存解析结果 (如果需要多次访问)
type CachingParser struct {
    parser *Parser
    cache  map[string]*WorkflowDefinition
    mu     sync.RWMutex
}

func (cp *CachingParser) Parse(yaml string) (*WorkflowDefinition, error) {
    hash := computeHash(yaml)
    
    cp.mu.RLock()
    if cached, ok := cp.cache[hash]; ok {
        cp.mu.RUnlock()
        return cached, nil
    }
    cp.mu.RUnlock()
    
    wf, err := cp.parser.Parse(yaml)
    if err != nil {
        return nil, err
    }
    
    cp.mu.Lock()
    cp.cache[hash] = wf
    cp.mu.Unlock()
    
    return wf, nil
}
```

**5. 安全性考虑**

```go
// 限制YAML文件大小 (防止DoS)
const MaxYAMLSize = 1 << 20  // 1MB

func (p *Parser) Parse(yamlContent string) (*WorkflowDefinition, error) {
    if len(yamlContent) > MaxYAMLSize {
        return nil, fmt.Errorf("YAML content exceeds max size of %d bytes", MaxYAMLSize)
    }
    // ... 正常解析
}

// 限制嵌套深度
decoder := yaml.NewDecoder(reader)
decoder.KnownFields(true)
// yaml.v3暂不支持深度限制,手动检查
```

**6. 测试策略 - 模糊测试 (Fuzzing)**

```go
func FuzzParser(f *testing.F) {
    p := New()
    
    // 添加种子语料
    f.Add("name: Test\non: push\njobs:\n  build:\n    runs-on: linux\n    steps:\n      - name: Test\n        uses: run@v1")
    
    f.Fuzz(func(t *testing.T, yamlContent string) {
          // 不应该panic
          _, _ = p.Parse(yamlContent)
      })
}
```

运行模糊测试:
```bash
go test -fuzz=FuzzParser -fuzztime=30s ./internal/parser
```

### Integration with Previous Stories

**与Story 1.2 REST API集成:**

```go
// Story 1.2提供的HTTP框架
router := gin.New()

// Story 1.3新增验证端点
v1 := router.Group("/v1")
v1.POST("/validate", validateHandler.Validate)

// Story 1.5将复用解析器
v1.POST("/workflows", workflowHandler.Submit)  // 内部调用parser.Parse()
```

**复用Story 1.1-1.2成果:**

1. ✅ **项目结构** - 在internal/parser/下创建
2. ✅ **Makefile** - 添加`make test-parser`目标
3. ✅ **Logger** - 解析错误记录到日志
4. ✅ **Config** - 可配置最大YAML大小

### Testing Strategy

**单元测试矩阵:**

| 测试类别 | 测试用例数 | 覆盖场景 |
|---------|-----------|---------|
| 有效YAML | 5+ | 最小/复杂/边界情况 |
| 缺失字段 | 5+ | name/on/jobs/runs-on/steps |
| 格式错误 | 5+ | uses格式/job-id格式/语法错误 |
| 边界值 | 3+ | 超时上下限/空数组/大型工作流 |
| 错误定位 | 3+ | 行号准确性 |

**集成测试:**

```bash
# 启动服务器
go run ./cmd/server &

# 测试验证端点
./test/integration/test_validate_api.sh

# 停止服务器
kill %1
```

**性能要求:**

- 小型工作流 (<10 jobs): <1ms
- 中型工作流 (10-50 jobs): <5ms
- 大型工作流 (>100 jobs): <20ms
- 内存: 每个WorkflowDefinition <1MB

### References

**架构设计:**
- [docs/architecture.md §3.1.2](docs/architecture.md) - YAML Parser职责
- [docs/adr/0004-yaml-dsl-syntax.md](docs/adr/0004-yaml-dsl-syntax.md) - YAML语法规范

**技术文档:**
- [gopkg.in/yaml.v3 Documentation](https://pkg.go.dev/gopkg.in/yaml.v3)
- [go-playground/validator Documentation](https://github.com/go-playground/validator)
- [GitHub Actions Workflow Syntax](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions)

**项目上下文:**
- [docs/epics.md Story 1.1-1.2](docs/epics.md) - 前置Stories
- [docs/epics.md Story 1.5](docs/epics.md) - 工作流提交API (使用解析器)

**Go规范:**
- [Effective Go: Errors](https://go.dev/doc/effective_go#errors)
- [Uber Go Style Guide: Error Handling](https://github.com/uber-go/guide/blob/master/style.md#error-handling)

### Dependency Graph

```
Story 1.1 (框架)
    ↓
Story 1.2 (REST API)
    ↓
Story 1.3 (YAML解析器) ← 当前Story
    ↓
    ├→ Story 1.5 (工作流提交API) - 使用Parse()验证YAML
    ├→ Story 1.6 (执行引擎) - 使用WorkflowDefinition
    └→ Story 2.x (表达式系统) - 解析${{}}语法
```

## Dev Agent Record

### Context Reference

**Source Documents Analyzed:**
1. [docs/epics.md](docs/epics.md) (lines 314-327) - Story 1.3需求定义
2. [docs/architecture.md](docs/architecture.md) (§3.1.2) - YAML Parser架构设计
3. [docs/adr/0004-yaml-dsl-syntax.md](docs/adr/0004-yaml-dsl-syntax.md) - YAML语法规范

**Previous Stories:**
- Story 1.1: 项目框架 (已完成)
- Story 1.2: REST API框架 (已完成)

### Agent Model Used

Claude 3.5 Sonnet (BMM Scrum Master Agent - Bob)

### Estimated Effort

**开发时间:** 8-10小时  
**复杂度:** 中等

**时间分解:**
- 数据结构定义: 1小时
- YAML解析器实现: 2小时
- Schema验证器实现: 2小时
- 错误诊断系统: 1.5小时
- REST API集成: 1小时
- 测试数据集创建: 0.5小时
- 单元测试编写: 2小时
- 性能基准测试: 1小时

**技能要求:**
- Go语言进阶 (结构体标签、接口、错误处理)
- YAML语法理解
- 正则表达式
- 单元测试和模糊测试

### Debug Log References

<!-- Will be populated during implementation -->

### Completion Notes List

<!-- Developer填写完成时的笔记 -->

### File List

**预期创建/修改的文件清单:**

```
新建文件 (~15个):
├── internal/parser/
│   ├── parser.go                   # 解析器核心逻辑
│   ├── validator.go                # 自定义验证器
│   ├── types.go                    # WorkflowDefinition结构体
│   ├── errors.go                   # 错误类型定义
│   ├── parser_test.go              # 单元测试
│   └── parser_bench_test.go        # 性能基准测试
├── internal/server/handlers/
│   ├── validate.go                 # 验证端点handler
│   └── validate_test.go            # 端点测试
├── test/testdata/workflows/
│   ├── valid_minimal.yaml
│   ├── valid_complex.yaml
│   ├── invalid_missing_name.yaml
│   ├── invalid_empty_steps.yaml
│   └── invalid_bad_uses_format.yaml

修改文件 (~2个):
├── internal/server/router.go       # 注册/v1/validate端点
└── go.mod                          # 添加yaml.v3和validator依赖
```

**关键代码片段:**

**parser.go (核心):**
```go
package parser

import (
    "gopkg.in/yaml.v3"
    "github.com/go-playground/validator/v10"
)

type Parser struct {
    validator *validator.Validate
}

func New() *Parser {
    v := validator.New()
    v.RegisterValidation("node_format", validateNodeFormat)
    v.RegisterValidation("job_name", validateJobName)
    return &Parser{validator: v}
}

func (p *Parser) Parse(yamlContent string) (*WorkflowDefinition, error) {
    var wf WorkflowDefinition
    
    // 1. YAML解析
    if err := yaml.Unmarshal([]byte(yamlContent), &wf); err != nil {
        return nil, NewParseError(err)
    }
    
    // 2. Schema验证
    if err := p.validator.Struct(&wf); err != nil {
        return nil, NewValidationError(err)
    }
    
    return &wf, nil
}
```

**types.go (数据结构):**
```go
package parser

type WorkflowDefinition struct {
    Name string           `yaml:"name" validate:"required,min=1"`
    On   string           `yaml:"on" validate:"required,oneof=push schedule webhook"`
    Jobs map[string]Job   `yaml:"jobs" validate:"required,min=1,dive"`
}

type Job struct {
    RunsOn         string `yaml:"runs-on" validate:"required"`
    Steps          []Step `yaml:"steps" validate:"required,min=1,dive"`
    TimeoutMinutes int    `yaml:"timeout-minutes" validate:"omitempty,min=1,max=1440"`
}

type Step struct {
    Name           string                 `yaml:"name" validate:"required"`
    Uses           string                 `yaml:"uses" validate:"required,node_format"`
    With           map[string]interface{} `yaml:"with"`
    TimeoutMinutes int                    `yaml:"timeout-minutes" validate:"omitempty,min=1,max=120"`
}
```

**validate.go (API端点):**
```go
package handlers

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "waterflow/internal/parser"
)

type ValidateHandler struct {
    parser *parser.Parser
}

func NewValidateHandler() *ValidateHandler {
    return &ValidateHandler{parser: parser.New()}
}

func (h *ValidateHandler) Validate(c *gin.Context) {
    var req struct {
        Workflow string `json:"workflow" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'workflow' field"})
        return
    }
    
    wf, err := h.parser.Parse(req.Workflow)
    if err != nil {
        c.JSON(http.StatusBadRequest, formatParseError(err))
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "valid": true,
        "summary": gin.H{
            "name":      wf.Name,
            "on":        wf.On,
            "job_count": len(wf.Jobs),
        },
    })
}
```

---

**Story Ready for Development** ✅

开发者可基于Story 1.1-1.2的成果,实现YAML解析器和验证端点。
解析器将在Story 1.5(工作流提交API)中被复用。
