# Story 6.4: 模板 API 端点

Status: ready-for-dev

## Story

As a **开发者**,  
I want **通过 API 访问工作流模板**,  
So that **程序化使用模板**。

## Context

这是 Epic 6 (工作流模板库) 的**第四个 Story**,实现**模板 API 端点**。该 Story 为前三个 Story 创建的工作流模板提供 REST API 访问接口,支持程序化获取模板列表、模板内容和参数说明。

**前置依赖:**
- ✅ Story 1.2 - REST API 服务框架 (路由、Handler、错误处理)
- ✅ Story 6.1 - 单服务器部署模板 (模板文件)
- ✅ Story 6.2 - 多服务器健康检查模板 (模板文件)
- ✅ Story 6.3 - 分布式栈部署模板 (模板文件)

**Epic 背景:**  
Epic 6 专注于**工作流模板库**。前三个 Story 创建了 3 个高质量模板,本 Story 提供 API 访问接口,使得:
- SDK/CLI 可以程序化获取模板
- Web UI 可以展示模板列表 (Post-MVP)
- 用户可以通过 API 浏览和使用模板
- 自动化工具可以集成模板

**业务价值:**
- 🎯 **程序化访问** - SDK/CLI 可直接获取模板
- 🎯 **模板发现** - API 列出所有可用模板
- 🎯 **参数说明** - API 提供参数元数据
- 🎯 **易于集成** - 标准 REST API,任何语言可用

**API 设计原则:**
1. **RESTful** - 符合 REST 规范
2. **元数据丰富** - 提供模板描述、参数说明
3. **向后兼容** - 支持未来扩展 (tags, categories)
4. **性能优化** - 模板列表缓存,避免重复读取

**API 范围 (MVP):**
- ✅ GET /v1/templates - 列出所有模板
- ✅ GET /v1/templates/{name} - 获取单个模板内容
- ❌ POST /v1/templates - 上传自定义模板 (Post-MVP)
- ❌ GET /v1/templates/{name}/versions - 模板版本管理 (Post-MVP)

**与现有 API 的关系:**
```
/v1/workflows         # Story 1.9: 工作流管理 API
/v1/workflows/validate # Story 1.2: 验证 API
/v1/nodes             # Story 5.6: 节点列表 API
/v1/templates         # 本 Story: 模板 API (新增)
```

**模板元数据设计:**

每个模板包含:
- `name` - 模板名称 (唯一标识)
- `description` - 模板描述
- `category` - 类别 (deployment, monitoring, automation)
- `parameters` - 参数列表 (name, type, required, default, description)
- `tags` - 标签 (optional, post-mvp)
- `examples` - 使用示例 (optional)

**实现策略:**

**选项 A: 从 YAML 注释解析元数据** (推荐,灵活)
- 模板 YAML 顶部包含结构化注释
- API 读取 YAML 文件,解析注释提取元数据
- 优点:模板和元数据在同一文件,易于维护
- 缺点:需要解析逻辑

**选项 B: 独立元数据文件** (简单,但分散)
- 创建 `templates.json` 或 `template-metadata.json`
- 包含所有模板的元数据
- 优点:简单,易于查询
- 缺点:模板和元数据分离,易不同步

**选项 C: 代码硬编码** (不推荐)
- 在 API Handler 中硬编码模板列表
- 不灵活,难以维护

MVP 采用**选项 B** (独立元数据文件),简单快速。Post-MVP 可考虑选项 A。

**文档结构:**
```
examples/workflows/
  ├── single-server-deployment.yaml
  ├── multi-server-health-check.yaml
  ├── distributed-stack-deployment.yaml
  └── templates-metadata.json          # 本 Story: 模板元数据
  
internal/api/
  ├── template_handler.go               # 本 Story: 模板 API Handler
  └── template_handler_test.go          # 本 Story: 单元测试
```

## Acceptance Criteria

### AC1: GET /v1/templates - 列出所有模板

**Given** 3 个内置模板已创建 (Story 6.1-6.3)  
**When** 调用 `GET /v1/templates` API  
**Then** 返回 HTTP 200 和 JSON 响应:
```json
{
  "templates": [
    {
      "name": "single-server-deployment",
      "description": "Deploy application to a single server",
      "category": "deployment",
      "parameters": [...]
    },
    {
      "name": "multi-server-health-check",
      "description": "Parallel health check across multiple servers",
      "category": "monitoring"
    },
    {
      "name": "distributed-stack-deployment",
      "description": "Deploy multi-tier application stack",
      "category": "deployment"
    }
  ],
  "count": 3
}
```

**And** 响应包含所有可用模板  
**And** 每个模板包含 name, description, category  
**And** 支持可选查询参数: `?category=deployment` (过滤)

**Implementation Notes:**
- 读取 `examples/workflows/templates-metadata.json`
- 加载到内存缓存 (启动时或首次请求)
- Handler: `GET /v1/templates`
- 支持 category 过滤 (可选,MVP 建议支持)

### AC2: GET /v1/templates/{name} - 获取单个模板

**Given** 模板 "single-server-deployment" 存在  
**When** 调用 `GET /v1/templates/single-server-deployment`  
**Then** 返回 HTTP 200 和 JSON 响应:
```json
{
  "name": "single-server-deployment",
  "description": "Deploy application to a single server with health check and rollback",
  "category": "deployment",
  "parameters": [
    {
      "name": "repo_url",
      "type": "string",
      "required": true,
      "description": "Git repository URL"
    },
    {
      "name": "app_name",
      "type": "string",
      "required": true,
      "description": "Application name for container/process"
    },
    {
      "name": "app_port",
      "type": "integer",
      "required": false,
      "default": "3000",
      "description": "Application listening port"
    }
  ],
  "content": "<YAML content as string>",
  "examples": [
    {
      "name": "Node.js Application",
      "description": "Deploy Node.js Express app",
      "vars": {
        "repo_url": "https://github.com/user/nodejs-app.git",
        "app_name": "my-nodejs-app",
        "app_port": 3000
      }
    }
  ]
}
```

**And** 响应包含完整的模板元数据  
**And** `content` 字段包含原始 YAML 内容:  
- 使用 JSON 字符串编码 (自动转义引号、换行符)  
- 保持原始缩进和格式  
- 如果 YAML 文件超过 1MB,返回 413 Payload Too Large  
**And** `parameters` 列表包含所有参数说明  
**And** 支持可选查询参数 `?content=false`:  
- content=true (默认): 返回完整 YAML 内容  
- content=false: 仅返回元数据,跳过 YAML 读取 (性能优化)  
**And** 如果模板不存在,返回 HTTP 404

**Implementation Notes:**
- 路径参数: `{name}` (如 "single-server-deployment")
- 读取元数据: `templates-metadata.json`
- 读取 YAML: `examples/workflows/{name}.yaml` (如 content=true)
- YAML 内容通过 Go `json.Marshal` 自动处理转义
- 文件大小检查: 最大 1MB (防止响应过大)
- 合并返回完整响应
- 404 错误: 模板不存在时
- 413 错误: YAML 文件过大时

### AC3: 参数类型和验证

**Given** 模板参数定义  
**When** 返回参数元数据  
**Then** 支持以下参数类型:
- `string` - 字符串
- `integer` - 整数
- `boolean` - 布尔值
- `array` - 数组 (如 servers 列表)
- `object` - 对象 (可选,Post-MVP)

**And** 每个参数包含:
- `name` - 参数名称
- `type` - 参数类型
- `required` - 是否必需 (boolean)
- `default` - 默认值 (可选)
- `description` - 参数说明
- `example` - 示例值 (可选)

**Implementation Notes:**
- 在 `templates-metadata.json` 中定义参数
- 参数类型遵循 JSON Schema 规范
- 验证元数据文件格式 (启动时)

### AC4: 错误处理和状态码

**Given** API 实现  
**When** 发生各种错误情况  
**Then** 返回正确的 HTTP 状态码和错误消息:

| 场景 | 状态码 | 错误消息 |
|------|--------|----------|
| 模板不存在 | 404 | Template not found: {name} |
| 无效的 category 过滤 | 400 | Invalid category: {value} |
| YAML 文件过大 (>1MB) | 413 | Template file too large |
| 模板文件读取失败 | 500 | Failed to read template |
| 元数据文件损坏 | 500 | Template metadata corrupted |

**And** 错误响应使用 RFC 7807 Problem Details 格式:
```json
{
  "type": "https://waterflow.io/errors/template-not-found",
  "title": "Template Not Found",
  "status": 404,
  "detail": "Template 'invalid-template' does not exist",
  "instance": "/v1/templates/invalid-template"
}
```

**Implementation Notes:**
- 复用现有错误处理机制 (Story 1.2)
- 使用 RFC 7807 格式
- 记录错误日志

## Tasks / Subtasks

### Task 1: 创建模板元数据文件

- [ ] 1.1 定义元数据 JSON Schema
  - 定义模板元数据结构
  - 定义参数元数据结构
  - 包含示例数据
  
- [ ] 1.2 创建 templates-metadata.json
  - 创建 `examples/workflows/templates-metadata.json`
  - 添加 3 个模板的元数据 (Story 6.1-6.3)
  - 包含完整的参数定义
  - 添加使用示例

### Task 2: 实现模板 API Handler (AC1, AC2, AC4)

- [ ] 2.1 创建 template_handler.go
  - 定义数据结构: Template, TemplateParameter, TemplateExample
  - 实现 ListTemplates Handler (GET /v1/templates)
  - 实现 GetTemplate Handler (GET /v1/templates/{name})
  
- [ ] 2.2 实现元数据加载逻辑
  - 读取 templates-metadata.json
  - 解析 JSON 到结构体
  - 使用 sync.Once 确保元数据只加载一次
  - 使用 sync.RWMutex 保护缓存读写 (并发安全)
  - 测试并发请求的缓存访问 (go test -race)
  - 错误处理: 文件不存在、JSON 格式错误
  
- [ ] 2.3 实现模板内容读取
  - 检查 `?content=false` 查询参数,决定是否读取 YAML
  - 读取 YAML 文件: `examples/workflows/{name}.yaml`
  - 检查文件大小 (最大 1MB),超过返回 413 错误
  - 返回原始 YAML 内容作为 JSON 字符串 (自动转义)
  - 错误处理: 文件不存在、读取失败、文件过大
  
- [ ] 2.4 实现过滤功能 (AC1)
  - 支持 `?category=deployment` 查询参数
  - 过滤模板列表
  - 忽略大小写
  
- [ ] 2.5 实现错误处理
  - 404: 模板不存在
  - 400: 无效参数
  - 500: 服务器错误
  - 使用 RFC 7807 格式

### Task 3: 注册路由和集成

- [ ] 3.1 更新 router.go
  - 添加 GET /v1/templates 路由
  - 添加 GET /v1/templates/{name} 路由
  - 注册 Handler
  
- [ ] 3.2 更新 server.go (如需要)
  - 初始化模板服务
  - 配置模板目录路径

### Task 4: 单元测试和集成测试

- [ ] 4.1 创建 template_handler_test.go
  - 测试 ListTemplates (空列表、多个模板)
  - 测试 GetTemplate (存在、不存在)
  - 测试参数过滤 (category)
  - 测试错误情况 (404, 500)
  
- [ ] 4.2 创建测试数据
  - 创建测试用模板元数据
  - 创建测试用 YAML 文件
  
- [ ] 4.3 集成测试
  - 启动 Server
  - 调用 API: `curl http://localhost:8080/v1/templates`
  - 验证响应 JSON
  - 调用 API: `curl http://localhost:8080/v1/templates/single-server-deployment`
  - 验证 YAML 内容
  - 测试 `?content=false` 参数 (仅返回元数据)
  - 测试 `?category=deployment` 过滤功能
  
- [ ] 4.4 元数据一致性验证
  - 读取每个模板的 YAML 文件
  - 解析 vars 部分,提取实际参数列表
  - 对比元数据 JSON 中的 parameters 列表
  - 验证参数名称、类型、默认值一致性
  - 失败时生成详细不一致报告
  
- [ ] 4.5 性能和并发测试
  - 响应时间测试: ListTemplates < 50ms
  - 响应时间测试: GetTemplate < 100ms
  - 并发测试: 100 个并发请求访问 /v1/templates
  - 竞态检测: go test -race (验证缓存线程安全)
  - 大文件测试: 创建 500KB YAML,验证响应时间
  - 文件大小限制测试: 创建 2MB YAML,验证 413 错误

### Task 5: 文档更新

- [ ] 5.1 更新 API 文档
  - 添加 /v1/templates 端点说明
  - 添加请求/响应示例
  - 更新 OpenAPI 规范 (如存在)
  
- [ ] 5.2 更新 README
  - 添加模板 API 使用示例
  - curl 示例
  - Go SDK 示例 (如适用)

## Dev Notes

### Architecture Alignment

**REST API 框架 (Story 1.2):**
- ✅ 复用现有 Router (gorilla/mux)
- ✅ 复用错误处理 (RFC 7807)
- ✅ 复用中间件 (RequestID, Logger, CORS)

**工作流验证 API (Story 1.2):**
- 参考: `ValidateWorkflow` 和 `RenderWorkflow` Handler
- 类似模式: 读取 YAML,返回 JSON

**节点列表 API (Story 5.6):**
- 参考: `ListNodes` Handler
- 类似模式: 列出可用资源,返回元数据

**文件结构:**
```
internal/api/
  ├── router.go                    # 更新:添加模板路由
  ├── template_handler.go          # 新建:模板 API Handler
  └── template_handler_test.go     # 新建:单元测试

examples/workflows/
  ├── single-server-deployment.yaml
  ├── multi-server-health-check.yaml
  ├── distributed-stack-deployment.yaml
  └── templates-metadata.json      # 新建:模板元数据
```

### 关键技术决策

**1. 模板元数据结构**

**templates-metadata.json:**
```json
{
  "templates": [
    {
      "name": "single-server-deployment",
      "display_name": "Single Server Deployment",
      "description": "Deploy application to a single server with build, health check, and rollback support",
      "category": "deployment",
      "version": "1.0.0",
      "author": "Waterflow Team",
      "parameters": [
        {
          "name": "repo_url",
          "type": "string",
          "required": true,
          "description": "Git repository URL",
          "example": "https://github.com/user/myapp.git"
        },
        {
          "name": "app_name",
          "type": "string",
          "required": true,
          "description": "Application name for container/process naming",
          "example": "my-app"
        },
        {
          "name": "app_port",
          "type": "integer",
          "required": false,
          "default": "3000",
          "description": "Application listening port",
          "example": "3000"
        },
        {
          "name": "branch",
          "type": "string",
          "required": false,
          "default": "main",
          "description": "Git branch to deploy",
          "example": "main"
        }
      ],
      "examples": [
        {
          "name": "Node.js Application",
          "description": "Deploy a Node.js Express application",
          "vars": {
            "repo_url": "https://github.com/user/nodejs-app.git",
            "app_name": "nodejs-api",
            "app_port": 3000
          }
        }
      ]
    },
    {
      "name": "multi-server-health-check",
      "display_name": "Multi-Server Health Check",
      "description": "Parallel health check across multiple servers with metric collection and reporting",
      "category": "monitoring",
      "version": "1.0.0",
      "parameters": [
        {
          "name": "servers",
          "type": "array",
          "items": "string",
          "required": true,
          "description": "List of server names to check",
          "example": ["web-1", "web-2", "db-1"]
        },
        {
          "name": "cpu_threshold",
          "type": "integer",
          "required": false,
          "default": "80",
          "description": "CPU usage warning threshold (%)",
          "example": "80"
        }
      ]
    },
    {
      "name": "distributed-stack-deployment",
      "display_name": "Distributed Stack Deployment",
      "description": "Deploy multi-tier application stack with dependency management (Database + Application)",
      "category": "deployment",
      "version": "1.0.0",
      "parameters": [
        {
          "name": "db_version",
          "type": "string",
          "required": false,
          "default": "postgres:14",
          "description": "PostgreSQL Docker image version"
        },
        {
          "name": "app_image",
          "type": "string",
          "required": true,
          "description": "Application Docker image name"
        }
      ]
    }
  ]
}
```

**2. Go 数据结构**

```go
// Template represents a workflow template
type Template struct {
    Name        string              `json:"name"`
    DisplayName string              `json:"display_name"`
    Description string              `json:"description"`
    Category    string              `json:"category"`
    Version     string              `json:"version,omitempty"`
    Author      string              `json:"author,omitempty"`
    Parameters  []TemplateParameter `json:"parameters"`
    Examples    []TemplateExample   `json:"examples,omitempty"`
}

// TemplateParameter represents a template parameter
type TemplateParameter struct {
    Name        string      `json:"name"`
    Type        string      `json:"type"` // string, integer, boolean, array, object
    Items       string      `json:"items,omitempty"` // for array type
    Required    bool        `json:"required"`
    Default     string      `json:"default,omitempty"`
    Description string      `json:"description"`
    Example     interface{} `json:"example,omitempty"`
}

// TemplateExample represents a usage example
type TemplateExample struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Vars        map[string]interface{} `json:"vars"`
}

// TemplateListResponse for GET /v1/templates
type TemplateListResponse struct {
    Templates []Template `json:"templates"`
    Count     int        `json:"count"`
}

// TemplateDetailResponse for GET /v1/templates/{name}
type TemplateDetailResponse struct {
    Template
    Content string `json:"content,omitempty"` // YAML content as string (omitted if content=false)
}

// TemplateHandlers holds handler dependencies
type TemplateHandlers struct {
    logger          *logger.Logger
    cachedTemplates []Template
    cacheMutex      sync.RWMutex  // Protects cachedTemplates
    cacheOnce       sync.Once     // Ensures metadata loaded only once
    cacheLoadErr    error         // Stores cache load error
}
```

**3. Handler 实现**

```go
// ListTemplates handles GET /v1/templates
func (h *TemplateHandlers) ListTemplates(w http.ResponseWriter, r *http.Request) {
    // Parse query parameters
    category := r.URL.Query().Get("category")
    
    // Get templates from metadata
    templates, err := h.getTemplateMetadata()
    if err != nil {
        h.sendError(w, r, http.StatusInternalServerError, "Failed to load templates", err)
        return
    }
    
    // Filter by category if specified
    if category != "" {
        templates = filterByCategory(templates, category)
    }
    
    // Build response
    response := TemplateListResponse{
        Templates: templates,
        Count:     len(templates),
    }
    
    h.sendJSON(w, http.StatusOK, response)
}

// GetTemplate handles GET /v1/templates/{name}
func (h *TemplateHandlers) GetTemplate(w http.ResponseWriter, r *http.Request) {
    // Get template name from path
    vars := mux.Vars(r)
    name := vars["name"]
    
    // Get template metadata
    metadata, err := h.getTemplateByName(name)
    if err != nil {
        if errors.Is(err, ErrTemplateNotFound) {
            h.sendError(w, r, http.StatusNotFound, fmt.Sprintf("Template not found: %s", name), err)
            return
        }
        h.sendError(w, r, http.StatusInternalServerError, "Failed to load template", err)
        return
    }
    
    // Build response
    response := TemplateDetailResponse{
        Template: *metadata,
    }
    
    // Read YAML content if requested (default: true)
    includeContent := r.URL.Query().Get("content") != "false"
    if includeContent {
        content, err := h.readTemplateContent(name)
        if err != nil {
            if errors.Is(err, ErrTemplateFileTooLarge) {
                h.sendError(w, r, http.StatusRequestEntityTooLarge, "Template file too large (max 1MB)", err)
                return
            }
            h.sendError(w, r, http.StatusInternalServerError, "Failed to read template content", err)
            return
        }
        response.Content = content
    }
    
    h.sendJSON(w, http.StatusOK, response)
}

// getTemplateMetadata loads templates-metadata.json with concurrency-safe caching
func (h *TemplateHandlers) getTemplateMetadata() ([]Template, error) {
    // Load metadata only once using sync.Once
    h.cacheOnce.Do(func() {
        // Read metadata file
        data, err := os.ReadFile("examples/workflows/templates-metadata.json")
        if err != nil {
            h.cacheLoadErr = fmt.Errorf("read metadata file: %w", err)
            return
        }
        
        var metadata struct {
            Templates []Template `json:"templates"`
        }
        if err := json.Unmarshal(data, &metadata); err != nil {
            h.cacheLoadErr = fmt.Errorf("parse metadata: %w", err)
            return
        }
        
        // Cache templates (protected by sync.Once, no mutex needed here)
        h.cachedTemplates = metadata.Templates
    })
    
    // Return any error from loading
    if h.cacheLoadErr != nil {
        return nil, h.cacheLoadErr
    }
    
    // Read-lock for safe concurrent access
    h.cacheMutex.RLock()
    defer h.cacheMutex.RUnlock()
    
    return h.cachedTemplates, nil
}

// readTemplateContent reads YAML file with size limit
func (h *TemplateHandlers) readTemplateContent(name string) (string, error) {
    path := fmt.Sprintf("examples/workflows/%s.yaml", name)
    
    // Check file size before reading (max 1MB)
    fileInfo, err := os.Stat(path)
    if err != nil {
        return "", fmt.Errorf("stat template file: %w", err)
    }
    
    const maxFileSize = 1 * 1024 * 1024 // 1MB
    if fileInfo.Size() > maxFileSize {
        return "", ErrTemplateFileTooLarge
    }
    
    data, err := os.ReadFile(path)
    if err != nil {
        return "", fmt.Errorf("read template file: %w", err)
    }
    return string(data), nil
}
```

**4. 路由注册**

```go
// In router.go
func NewRouter(...) http.Handler {
    // ... existing routes ...
    
    // Template management endpoints (Story 6.4)
    th := NewTemplateHandlers(logger)
    router.HandleFunc("/v1/templates", th.ListTemplates).Methods(http.MethodGet)
    router.HandleFunc("/v1/templates/{name}", th.GetTemplate).Methods(http.MethodGet)
    
    return router
}
```

### 测试策略

**单元测试:**

```go
func TestListTemplates(t *testing.T) {
    // Test cases
    tests := []struct {
        name           string
        query          string
        expectedCount  int
        expectedStatus int
    }{
        {"All templates", "", 3, 200},
        {"Filter by category", "?category=deployment", 2, 200},
        {"Filter by monitoring", "?category=monitoring", 1, 200},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest("GET", "/v1/templates"+tt.query, nil)
            w := httptest.NewRecorder()
            
            handler.ListTemplates(w, req)
            
            assert.Equal(t, tt.expectedStatus, w.Code)
            // ... verify response body
        })
    }
}

func TestGetTemplate(t *testing.T) {
    tests := []struct {
        name           string
        templateName   string
        query          string
        expectedStatus int
        checkContent   bool
    }{
        {"Existing template with content", "single-server-deployment", "", 200, true},
        {"Existing template without content", "single-server-deployment", "?content=false", 200, false},
        {"Non-existent template", "invalid-template", "", 404, false},
        {"Large template file", "large-template", "", 413, false}, // Mock large file
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest("GET", "/v1/templates/"+tt.templateName+tt.query, nil)
            w := httptest.NewRecorder()
            
            handler.GetTemplate(w, req)
            
            assert.Equal(t, tt.expectedStatus, w.Code)
            
            if tt.expectedStatus == 200 {
                var resp TemplateDetailResponse
                json.Unmarshal(w.Body.Bytes(), &resp)
                
                if tt.checkContent {
                    assert.NotEmpty(t, resp.Content, "Content should be included")
                } else {
                    assert.Empty(t, resp.Content, "Content should be omitted")
                }
            }
        })
    }
}

func TestGetTemplateMetadataConcurrency(t *testing.T) {
    handler := NewTemplateHandlers(logger)
    
    // Run 100 concurrent requests
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _, err := handler.getTemplateMetadata()
            assert.NoError(t, err)
        }()
    }
    wg.Wait()
    
    // Verify metadata loaded only once (check logs or counter)
}
```

**集成测试:**

```bash
# Test 1: List all templates
curl http://localhost:8080/v1/templates | jq

# Expected: 3 templates with metadata

# Test 2: Get specific template with content
curl http://localhost:8080/v1/templates/single-server-deployment | jq

# Expected: Full metadata + YAML content

# Test 3: Get template metadata only (no content)
curl "http://localhost:8080/v1/templates/single-server-deployment?content=false" | jq

# Expected: Metadata only, content field omitted or empty

# Test 4: Filter by category
curl "http://localhost:8080/v1/templates?category=deployment" | jq

# Expected: 2 deployment templates

# Test 5: Non-existent template
curl -i http://localhost:8080/v1/templates/invalid

# Expected: 404 Not Found

# Test 6: Concurrent requests (performance test)
ab -n 1000 -c 100 http://localhost:8080/v1/templates

# Expected: All requests succeed, avg response time < 50ms

# Test 7: Large file test (create 2MB YAML)
curl -i http://localhost:8080/v1/templates/large-template

# Expected: 413 Payload Too Large
```

### 复用现有组件

**API 框架 (Story 1.2):**
- ✅ gorilla/mux Router
- ✅ RFC 7807 错误处理
- ✅ JSON 响应帮助函数

**中间件 (Story 1.2):**
- ✅ RequestID
- ✅ Logger
- ✅ CORS
- ✅ Version

**类似 Handler 参考:**
- ✅ ValidateWorkflow (Story 1.2) - 读取 YAML
- ✅ ListNodes (Story 5.6) - 列出资源

### Performance Considerations

**缓存策略:**
- 启动时或首次请求加载 `templates-metadata.json`
- 缓存到内存 (模板数量少,<10 个)
- YAML 内容每次读取 (或考虑缓存,如模板不常变化)

**文件 I/O:**
- 元数据文件: 一次性加载
- YAML 文件: 按需读取 (可考虑缓存)

**响应时间目标:**
- ListTemplates: < 50ms (从缓存)
- GetTemplate: < 100ms (读取 YAML)

### Security Considerations

**路径遍历防护:**
- 验证模板名称格式 (字母、数字、连字符)
- 禁止 `..` 或 `/` 字符
- 使用白名单验证 (从元数据文件)

**示例:**
```go
func validateTemplateName(name string) error {
    if matched, _ := regexp.MatchString(`^[a-zA-Z0-9-]+$`, name); !matched {
        return errors.New("invalid template name")
    }
    return nil
}
```

**敏感信息:**
- 模板文件是公开的,不包含密钥
- 参数示例不包含真实凭证

### References

**Epic 和 Story 文档:**
- [Source: docs/epics.md#Epic-6](../epics.md) - Epic 6 完整定义
- [Source: docs/sprint-artifacts/1-2-rest-api-service-framework.md](./1-2-rest-api-service-framework.md) - REST API 框架
- [Source: docs/sprint-artifacts/5-6-cli-node-list-command.md](./5-6-cli-node-list-command.md) - 节点列表 API (类似模式)
- [Source: docs/sprint-artifacts/6-1-single-server-deployment-template.md](./6-1-single-server-deployment-template.md) - 模板 1
- [Source: docs/sprint-artifacts/6-2-multi-server-health-check-template.md](./6-2-multi-server-health-check-template.md) - 模板 2
- [Source: docs/sprint-artifacts/6-3-distributed-stack-deployment-template.md](./6-3-distributed-stack-deployment-template.md) - 模板 3

**架构文档:**
- [Source: docs/architecture.md](../architecture.md) - 整体架构
- [Source: docs/prd.md](../prd.md) - 产品需求

**代码参考:**
- [Source: internal/api/router.go](../../internal/api/router.go) - 路由注册
- [Source: internal/api/handlers.go](../../internal/api/handlers.go) - Handler 模式
- [Source: internal/api/node_handlers.go](../../internal/api/node_handlers.go) - 节点 API (类似)

## Definition of Done

- [ ] 创建 `examples/workflows/templates-metadata.json` (元数据文件)
- [ ] 元数据包含 3 个模板的完整信息 (name, description, parameters, examples)
- [ ] 创建 `internal/api/template_handler.go` (AC1, AC2, AC4)
- [ ] 实现 ListTemplates Handler
- [ ] 实现 GetTemplate Handler
- [ ] 实现元数据加载和缓存逻辑
- [ ] 实现 category 过滤功能
- [ ] 实现错误处理 (404, 500, RFC 7807)
- [ ] 更新 `internal/api/router.go` 注册路由
- [ ] 创建 `internal/api/template_handler_test.go` (单元测试)
- [ ] 单元测试覆盖 ListTemplates (全部、过滤)
- [ ] 单元测试覆盖 GetTemplate (存在、不存在、content 参数)
- [ ] 单元测试覆盖错误情况 (404, 413, 500)
- [ ] 集成测试:启动 Server,调用 API,验证响应
- [ ] 一致性测试:验证元数据与 YAML 参数一致
- [ ] 性能测试:响应时间满足目标 (<50ms, <100ms)
- [ ] 并发测试:通过竞态检测 (go test -race)
- [ ] 大文件测试:验证文件大小限制 (>1MB 返回 413)
- [ ] 缓存测试:验证 sync.Once 和 sync.RWMutex 正确使用
- [ ] 文档更新:添加 API 使用示例
- [ ] 代码审查:代码质量、测试覆盖率
- [ ] 代码已提交 Git

## Dev Agent Record

### Context Reference

<!-- Story context will be added by context workflow -->

### Agent Model Used

<!-- To be filled by Dev agent -->

### Debug Log References

<!-- To be filled by Dev agent -->

### Completion Notes

<!-- To be filled by Dev agent -->

### File List

**预计创建的文件:**
- examples/workflows/templates-metadata.json (新建,约 200 行,包含完整元数据)
- internal/api/template_handler.go (新建,约 400 行,包含并发安全和文件大小检查)
- internal/api/template_handler_test.go (新建,约 300 行,包含并发和性能测试)

**预计修改的文件:**
- internal/api/router.go (更新,添加模板路由,约 +5 行)

## Change Log

- 2026-01-06: Story 创建,状态: ready-for-dev
