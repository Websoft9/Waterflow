# Waterflow 测试工具使用指南

本指南介绍如何使用 `test/support` 目录下的测试辅助工具。

## 目录结构

```
test/support/
├── apitest/      # API 测试辅助
├── factories/    # 测试数据工厂
├── mocks/        # Mock 实现
└── testutil/     # 通用测试工具
```

## 1. 测试环境 (testutil)

### TestEnvironment

创建标准测试环境，包含 Logger、Context 和自动清理：

```go
import "github.com/Websoft9/waterflow/test/support/testutil"

func TestExample(t *testing.T) {
    env := testutil.NewTestEnvironment(t)
    
    // 使用 env.Logger 进行日志
    env.Logger.Info("test started")
    
    // 使用 env.Context 进行超时控制
    result, err := someFunc(env.Context)
    
    // 添加自定义清理
    env.AddCleanup(func() {
        // 清理资源
    })
}
```

### 辅助函数

```go
// 跳过短模式测试
testutil.SkipIfShort(t)

// 跳过 CI 环境
testutil.SkipIfCI(t)

// 创建临时目录（自动清理）
dir := testutil.TempDir(t)

// 创建临时文件
path := testutil.TempFile(t, "test.yaml", "content")

// 创建 Mock HTTP Server
server := testutil.MockServer(t, func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"ok"}`))
})

// 断言条件最终成立
testutil.AssertEventually(t, func() bool {
    return someCondition()
}, 5*time.Second, 100*time.Millisecond, "condition not met")

// 生成唯一 ID
id := testutil.GenerateUniqueID("workflow")
```

## 2. 测试数据工厂 (factories)

### WorkflowFactory

创建测试用的 Workflow YAML：

```go
import "github.com/Websoft9/waterflow/test/support/factories"

func TestWorkflow(t *testing.T) {
    // 简单 Hello World 工作流
    yaml := factories.HelloWorldWorkflow()
    
    // 自定义工作流
    yaml := factories.NewWorkflowFactory().
        WithName("my-test-workflow").
        WithTrigger("manual").
        AddSimpleJob("build", "linux-amd64", "Building...").
        AddSimpleJob("test", "linux-amd64", "Testing...").
        Build()
}
```

### StepFactory

创建不同类型的步骤：

```go
sf := factories.NewStepFactory()

// Shell 命令步骤
step := sf.ShellStep("echo hello")

// HTTP 请求步骤
step := sf.HTTPStep("GET", "https://api.example.com/status")

// 延迟步骤
step := sf.SleepStep("5s")

// Docker 执行步骤
step := sf.DockerStep("alpine:latest", "echo hello")
```

### ConfigFactory

创建测试配置：

```go
cf := factories.NewConfigFactory()

serverCfg := cf.ServerConfig(8080)
agentCfg := cf.AgentConfig([]string{"linux-amd64"})
temporalCfg := cf.TemporalConfig("localhost:7233")
```

### UserFactory

创建测试用户：

```go
uf := factories.NewUserFactory()
user1 := uf.Create() // user-1, test1@example.com
user2 := uf.Create() // user-2, test2@example.com
```

## 3. Mock 实现 (mocks)

### HTTPClient Mock

模拟 HTTP 请求：

```go
import "github.com/Websoft9/waterflow/test/support/mocks"

func TestHTTPClient(t *testing.T) {
    client := mocks.NewHTTPClient()
    
    // 添加预期响应
    client.AddResponse(http.StatusOK, `{"status":"success"}`)
    client.AddResponse(http.StatusNotFound, `{"error":"not found"}`)
    
    // 添加错误响应
    client.AddError(errors.New("connection refused"))
    
    // 设置默认响应
    client.SetDefaultResponse(http.StatusOK, `{}`)
    
    // 使用 client.Do(req) 发送请求
    
    // 验证请求
    assert.Equal(t, 3, client.RequestCount())
    lastReq := client.LastRequest()
    assert.Equal(t, "GET", lastReq.Method)
    
    // 重置
    client.Reset()
}
```

### PluginLoader Mock

模拟插件加载：

```go
func TestPluginLoader(t *testing.T) {
    loader := mocks.NewPluginLoader()
    
    // 注册 Mock 节点
    mockNode := mocks.NewMockNode("shell", "v1").
        WithExecuteFunc(func(ctx context.Context, params map[string]interface{}) (*node.Result, error) {
            return &node.Result{
                Status: "success",
                Outputs: map[string]interface{}{
                    "stdout": "hello world",
                },
            }, nil
        })
    
    loader.RegisterNode("shell@v1", mockNode)
    
    // 使用
    n, err := loader.GetNode("shell@v1")
    require.NoError(t, err)
    
    result, err := n.Execute(ctx, params)
    require.NoError(t, err)
}
```

### MockNode

创建可配置的 Mock 节点：

```go
// 基本使用
mockNode := mocks.NewMockNode("http", "v1")

// 自定义执行逻辑
mockNode.WithExecuteFunc(func(ctx context.Context, params map[string]interface{}) (*node.Result, error) {
    // 自定义逻辑
    return &node.Result{Status: "success"}, nil
})

// 设置验证错误
mockNode.WithValidateError(errors.New("invalid params"))
```

## 4. API 测试辅助 (apitest)

参见 `test/support/apitest/` 目录中的实现。

## 5. 最佳实践

### 测试隔离

```go
func TestIsolated(t *testing.T) {
    t.Parallel() // 启用并行测试
    
    env := testutil.NewTestEnvironment(t)
    // 每个测试使用独立的环境
}
```

### 表驱动测试

```go
func TestTableDriven(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "hello", "HELLO", false},
        {"empty input", "", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Transform(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### 使用 require vs assert

```go
func TestAssertions(t *testing.T) {
    // require: 失败立即停止测试
    result, err := SomeFunc()
    require.NoError(t, err, "setup must succeed")
    require.NotNil(t, result)
    
    // assert: 失败继续执行
    assert.Equal(t, "expected", result.Value)
    assert.Contains(t, result.Message, "success")
}
```

## 6. 运行测试

```bash
# 运行所有测试
go test ./...

# 运行短模式（跳过集成测试）
go test ./... -short

# 运行特定包
go test ./internal/agent/...

# 运行带覆盖率
go test ./... -cover -coverprofile=coverage.out

# 查看覆盖率报告
go tool cover -html=coverage.out
```
