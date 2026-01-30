# Waterflow 测试框架

本目录包含 Waterflow 项目的完整测试套件，按照 BMad 标准组织。

## 目录结构

```
test/
├── README.md           # 本文件
├── support/            # 测试支持包 (跨 Epic 共享)
│   ├── testutil/       # 通用测试工具
│   ├── factories/      # 测试数据工厂
│   ├── mocks/          # Mock 实现
│   └── apitest/        # API 测试客户端
├── integration/        # 集成测试 (组件交互)
│   ├── epic1/          # Epic 1: 核心工作流引擎 (67 tests)
│   └── epic9/          # Epic 9: 安全和合规 (9 tests)
├── e2e/                # 端到端测试 (完整管道)
│   ├── story1_*.go     # Story 级 E2E 测试 (44 tests)
│   ├── common_test.go
│   ├── docker-compose.e2e.yaml
│   └── testdata/
├── acceptance/         # 验收测试 (PRD 场景)
│   ├── *_test.go       # 场景测试 (2 scenarios)
│   ├── helpers.go
│   ├── docker-compose.acceptance.yaml
│   └── testdata/
├── performance/        # 性能测试 (系统级)
│   ├── throughput_test.go
│   └── server_startup_test.go
└── stress/             # 压力测试和容错验证 (系统级)
    ├── *.go            # Go 压力测试
    └── *.sh            # Shell 脚本测试
```

## BMad 测试组织标准

### 何时使用 Epic 组织

**✅ 集成测试 (Integration)** - 按 Epic 组织
- 原因：每个 Epic 引入不同的组件和交互逻辑
- 数量大，需要清晰的边界
- 结构：`test/integration/epic{N}/`

**❌ 其他测试类型** - 平铺组织
- E2E/Acceptance/Performance/Stress/Security 验证系统级特性
- 测试数量少，平铺结构更简洁
- 无需额外的目录层级

### 测试层级 (4-Tier)

按照 BMad 标准，Waterflow 使用 4 层测试金字塔：

| 层级 | 目录 | 范围 | 环境 | 速度 | 数量 |
|------|------|------|------|------|------|
| **1. Unit** | `pkg/*/` | 单个函数/类 | 内存 | 快 | 多 |
| **2. Integration** | `test/integration/` | 组件交互 | 部分真实 | 中 | 中 |
| **3. E2E** | `test/e2e/` | 完整技术管道 | 完整真实 | 慢 | 少 |
| **4. Acceptance** | `test/acceptance/` | PRD 业务场景 | 生产级 | 最慢 | 最少 |

#### 单元测试 (Unit)
- **位置**: `pkg/*/`, `internal/*/`
- **文件**: `*_test.go`
- **标签**: 无
- **示例**: `pkg/dsl/parser_test.go`

#### 集成测试 (Integration)
- **位置**: `test/integration/epic{N}/`
- **文件**: `story{E}_{S}_test.go`
- **标签**: `//go:build integration`
- **命名**: `TestStory{E}_{S}_INT_{Seq}_{Description}`
- **示例**: `test/integration/epic1/story1_1_test.go`

#### E2E 测试 (End-to-End)
- **位置**: `test/e2e/epic{N}/`
- **文件**: `story{E}_{S}_{feature}_test.go`
- **标签**: `//go:build e2e`
- **命名**: `TestStory{E}_{S}_E2E_{Seq}_{Description}`
- **示例**: `test/e2e/epic1/story1_3_yaml_dsl_test.go`

#### 验收测试 (Acceptance)
- **位置**: `test/acceptance/epic{N}/`
- **文件**: `{scenario}_test.go`
- **标签**: `//go:build acceptance`
- **命名**: `TestAcceptance_{Scenario}`
- **示例**: `test/acceptance/epic1/health_check_test.go`

### 特殊测试类型

#### 性能测试 (Performance)
- **位置**: `test/performance/epic{N}/`
- **文件**: `*_test.go`, `*.sh`
- **用途**: 验证 NFR2 性能指标
- **示例**: 吞吐量、延迟、资源使用

#### 压力测试 (Stress)
- **位置**: `test/stress/`
- **文件**: `*.go`, `*.sh`
- **用途**: 验证系统可靠性和容错能力
- **示例**: 并发、崩溃恢复、资源泄漏

**注意：安全测试已整合到 Integration Epic 9**
- Epic 9 Story 测试验证 Vault、审计日志、TLS 等安全特性
- 安全是横切关注点，通过 Story AC 验证，而非独立测试目录

## 运行测试

### 快速运行

```bash
# 所有单元测试
make test

# 所有集成测试 (Epic 1)
make integration-test

# 所有 E2E 测试 (Epic 1)
make e2e-test

# 所有验收测试 (Epic 1)
make acceptance-test
```

### 按 Epic 运行

```bash
# Epic 1 集成测试 (核心工作流)
go test -tags=integration -v ./test/integration/epic1/...

# Epic 9 集成测试 (安全和合规，需要 Vault)
export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=mytoken
go test -tags=integration -v ./test/integration/epic9/...
```

### 按测试类型运行

```bash
# E2E 测试
go test -tags=e2e -v ./test/e2e/...

# 验收测试
go test -tags=acceptance -v ./test/acceptance/...

# 性能测试
cd test/performance
./api_benchmark.sh

# 压力测试
cd test/stress
./run_all_tests.sh
```

### 覆盖率

```bash
# 单元测试覆盖率
go test -cover ./...

# 集成测试覆盖率
go test -tags=integration -cover ./test/integration/epic1/...

# 生成 HTML 报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
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

### 命名规范

#### 单元测试
- **文件**: `*_test.go` (与源文件同目录)
- **函数**: `Test{Function}` 或 `Test{Type}_{Method}`
- **示例**: `TestParse`, `TestWorkflow_Validate`

#### 集成测试
- **文件**: `test/integration/epic{N}/story{E}_{S}_test.go`
- **函数**: `TestStory{E}_{S}_INT_{Seq}_{Description}`
- **Test ID**: `{E}.{S}-INT-{Seq}`
- **示例**: `TestStory1_1_INT_001_ServerHealthCheck`

#### E2E 测试
- **文件**: `test/e2e/epic{N}/story{E}_{S}_{feature}_test.go`
- **函数**: `TestStory{E}_{S}_E2E_{Seq}_{Description}`
- **Test ID**: `{E}.{S}-E2E-{Seq}`
- **示例**: `TestStory1_3_E2E_001_ComplexYAMLWorkflow`

#### 验收测试
- **文件**: `test/acceptance/epic{N}/{scenario}_test.go`
- **函数**: `TestAcceptance_{Scenario}`
- **Test ID**: `ACC-{Epic}-{Seq}`
- **示例**: `TestAcceptance_DistributedDeploy`

### Build Tags

使用 build tags 隔离不同层级的测试：

```go
//go:build integration

//go:build e2e

//go:build acceptance
```

### testdata 组织

测试数据按照 3 层组织：

1. **项目级** - `testdata/fixtures/` - 跨 Epic 共享数据
2. **测试类型级** - `test/{type}/testdata/` - 测试类型共享数据
3. **Epic 级** - `test/{type}/epic{N}/testdata/` - Epic 专属数据

参见 [testdata/README.md](../testdata/README.md) 了解详情。

## Epic 测试统计

### Epic 1 - 基础工作流引擎

| 测试类型 | 数量 | 位置 | 状态 |
|---------|------|------|------|
| Unit | ~200 | pkg/, internal/ | ✅ |
| Integration | 67 | test/integration/epic1/ | ✅ |
| E2E | 44 | test/e2e/ | ✅ |
| Acceptance | 2 | test/acceptance/ | ✅ |
| Performance | 3 | test/performance/ | ✅ |
| Stress | 8 | test/stress/ | ✅ |
| Security | 4 | test/security/ | ✅ |

## 参考文档

- [测试标准文档](../docs/test-review.md)
- [集成测试说明](integration/README.md)
- [E2E 测试说明](e2e/README.md)
- [验收测试说明](acceptance/README.md)
- [性能测试说明](performance/README.md)
- [压力测试说明](stress/README.md)
- [安全测试说明](security/README.md)
- [测试工具说明](support/README.md)
2. **集成测试** - 放在 `test/integration/` 目录
3. **测试命名** - 使用 `Test<Function>_<Scenario>` 格式
4. **表驱动测试** - 对于多场景使用 `t.Run()` 子测试
5. **Mock 依赖** - 使用接口和依赖注入
