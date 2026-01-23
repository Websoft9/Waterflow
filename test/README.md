# Waterflow 测试框架

本目录包含 Waterflow 项目的测试支持基础设施。

## 目录结构

```
test/
├── README.md           # 本文件
├── support/            # 测试支持包
│   ├── testutil/       # 通用测试工具
│   ├── factories/      # 测试数据工厂
│   └── apitest/        # API 测试客户端
├── integration/        # 集成测试
├── performance/        # 性能测试
├── security/           # 安全测试
└── stress/             # 压力测试
```

## 测试支持包

### testutil - 通用测试工具

提供常用的测试辅助函数：

```go
import "github.com/user/waterflow/test/support/testutil"

func TestExample(t *testing.T) {
    // 短测试跳过
    testutil.SkipIfShort(t, "需要外部服务")
    
    // 创建临时目录
    dir := testutil.TempDir(t)
    
    // 创建 mock HTTP 服务器
    server := testutil.MockServer(t, func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })
    
    // 最终断言
    testutil.AssertEventually(t, func() bool {
        return someCondition
    }, 5*time.Second, 100*time.Millisecond)
}
```

**主要功能：**
- `TestEnvironment` - 管理测试环境（目录、清理等）
- `SkipIfShort(t, reason)` - 短测试模式跳过
- `SkipIfCI(t, reason)` - CI 环境跳过
- `TempDir(t)` - 创建自动清理的临时目录
- `TempFile(t, content)` - 创建临时文件
- `MockServer(t, handler)` - 创建 mock HTTP 服务器
- `AssertEventually(t, condition, timeout, interval)` - 等待条件满足
- `GenerateUniqueID()` - 生成唯一 ID

### factories - 测试数据工厂

使用工厂模式生成测试数据：

```go
import "github.com/user/waterflow/test/support/factories"

func TestWorkflow(t *testing.T) {
    // 使用流式构建器创建工作流
    workflow := factories.NewWorkflowFactory().
        WithName("test-workflow").
        WithDescription("测试工作流").
        WithEnv("ENV", "test").
        WithJob("build", factories.JobConfig{
            Plugin: "exec",
            Steps: []factories.StepConfig{
                {Name: "build", Action: "run", Params: map[string]interface{}{"cmd": "go build"}},
            },
        }).
        Build()
    
    // 或使用预定义工作流
    helloWorld := factories.HelloWorldWorkflow()
    
    // 创建测试用户
    user := factories.NewUserFactory().
        WithEmail("test@example.com").
        WithRole("admin").
        Create()
}
```

**主要功能：**
- `WorkflowFactory` - 工作流 YAML 构建器
- `HelloWorldWorkflow()` - 预定义的 Hello World 工作流
- `UserFactory` - 测试用户构建器

### apitest - API 测试客户端

流式 API 测试和 RFC 7807 错误断言：

```go
import "github.com/user/waterflow/test/support/apitest"

func TestAPIEndpoint(t *testing.T) {
    client := apitest.NewClient(t, router)
    
    // 测试成功响应
    client.Get("/api/v1/health").
        Do().
        AssertOK().
        AssertJSON()
    
    // 测试 POST 请求
    client.Post("/api/v1/workflows").
        WithJSONBody(map[string]string{"name": "test"}).
        Do().
        AssertCreated()
    
    // 测试 RFC 7807 错误响应
    client.Get("/api/v1/workflows/not-found").
        Do().
        AssertNotFound().
        AssertRFC7807().
        IsNotFound().
        HasDetail("workflow not found")
}
```

**主要功能：**
- `NewClient(t, handler)` - 创建 API 测试客户端
- `Get(path)`, `Post(path)` - HTTP 请求构建器
- `WithJSONBody(v)` - 设置 JSON 请求体
- `AssertStatus(code)`, `AssertOK()`, `AssertNotFound()` - 状态码断言
- `AssertRFC7807()` - RFC 7807 错误断言链
- `IsNotFound()`, `IsBadRequest()`, `IsMethodNotAllowed()` - 预定义错误断言

## RFC 7807 错误格式

Waterflow API 使用 RFC 7807 Problem Details 格式返回错误：

```json
{
    "type": "not_found",
    "title": "Not Found",
    "status": 404,
    "detail": "workflow not found"
}
```

**错误类型：**
| type | status | 说明 |
|------|--------|------|
| `not_found` | 404 | 资源未找到 |
| `invalid_argument` | 400 | 无效参数 |
| `method_not_allowed` | 405 | 方法不允许 |
| `payload_too_large` | 413 | 请求体过大 |
| `internal_error` | 500 | 内部错误 |
| `service_unavailable` | 503 | 服务不可用 |

## 运行测试

```bash
# 运行所有测试
make test

# 运行短测试（跳过慢测试）
go test -short ./...

# 运行特定包的测试
go test ./internal/api/...

# 运行带覆盖率的测试
go test -cover ./...

# 运行集成测试（需要 Temporal）
go test -tags=integration ./test/integration/...
```

## 测试约定

1. **单元测试** - 放在对应包的 `*_test.go` 文件中
2. **集成测试** - 放在 `test/integration/` 目录
3. **测试命名** - 使用 `Test<Function>_<Scenario>` 格式
4. **表驱动测试** - 对于多场景使用 `t.Run()` 子测试
5. **Mock 依赖** - 使用接口和依赖注入
