# Integration Testing

本目录包含 Waterflow 的端到端集成测试，用于验证系统各组件在真实环境中的协作。

## 目录结构

```
test/integration/
├── README.md              # 本文档
├── workflow_e2e_test.go   # 工作流端到端测试
├── api_test.go            # API 集成测试
├── agent_test.go          # Agent 集成测试
├── multi_step_test.go     # 多步骤工作流测试
├── retry_test.go          # 重试与容错测试
├── common_test.go         # 共享测试工具函数
└── testdata/
    └── workflows/         # 测试工作流定义
        ├── simple-echo.yaml
        ├── multi-step-outputs.yaml
        ├── matrix.yaml
        ├── retry.yaml
        ├── conditional.yaml
        └── job-dependencies.yaml
```

## 运行集成测试

### 方式一：使用脚本 (推荐)

```bash
# 完整运行：启动环境、运行测试、清理
make integration-test

# 或直接执行脚本
./scripts/run-integration-tests.sh
```

### 方式二：手动运行

```bash
# 1. 启动测试环境
docker-compose -f deployments/docker-compose.test.yaml up -d

# 2. 等待服务就绪
# Server: http://localhost:18080/health
# Agent: http://localhost:18081/health

# 3. 运行测试
go test -v -tags integration ./test/integration/...

# 4. 清理环境
docker-compose -f deployments/docker-compose.test.yaml down -v
```

### 方式三：使用已有环境

```bash
# 如果环境已在运行，只运行测试
make integration-test-only

# 或指定自定义 URL
SERVER_URL=http://localhost:8080 go test -v -tags integration ./test/integration/...
```

## 测试分类

### 1. Workflow E2E 测试 (`workflow_e2e_test.go`)

验证工作流的完整生命周期：

- **提交工作流** - 验证 YAML 提交和解析
- **状态查询** - 验证状态轮询和转换
- **工作流取消** - 验证优雅取消机制
- **日志收集** - 验证步骤日志获取

### 2. API 测试 (`api_test.go`)

验证 REST API 端点：

- **YAML 验证** - `/api/v1/validate` 端点
- **可用节点** - `/api/v1/nodes` 端点
- **工作流列表** - `/api/v1/workflows` 端点
- **健康检查** - `/health` 和 `/ready` 端点
- **工作流重跑** - `/api/v1/workflows/:id/rerun`

### 3. Agent 测试 (`agent_test.go`)

验证 Agent 功能：

- **连接性** - Agent 健康检查和注册
- **命令执行** - Shell 命令在 Agent 上执行
- **环境变量** - 全局和步骤级环境变量传递
- **工作目录** - 工作目录设置
- **日志收集** - 输出日志收集
- **并发执行** - 多工作流并行执行

### 4. 多步骤测试 (`multi_step_test.go`)

验证复杂工作流：

- **顺序执行** - 步骤按序执行
- **输出传递** - 步骤间输出传递
- **条件执行** - `if` 条件控制
- **Matrix 策略** - 并行矩阵展开
- **Job 依赖** - `needs` 依赖链

### 5. 容错测试 (`retry_test.go`)

验证错误处理：

- **重试机制** - 失败重试
- **退避策略** - 指数退避
- **超时处理** - 步骤和 Job 超时
- **错误传播** - 失败状态传播
- **continue-on-error** - 错误后继续

## 环境变量

| 变量 | 默认值 | 描述 |
|------|--------|------|
| `SERVER_URL` | `http://localhost:18080` | Waterflow Server 地址 |
| `AGENT_URL` | `http://localhost:18081` | Waterflow Agent 地址 |
| `TEST_TIMEOUT` | `5m` | 测试超时时间 |

## 测试夹具

### simple-echo.yaml

最简单的工作流，单步骤输出：

```yaml
name: simple-echo
jobs:
  echo:
    steps:
      - name: Say Hello
        uses: shell@v1
        with:
          command: echo "Hello, World!"
```

### matrix.yaml

Matrix 策略测试：

```yaml
jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu, alpine]
        version: [1.20, 1.21]
    steps:
      - name: Print version
        uses: shell@v1
        with:
          command: echo "${{ matrix.os }}-${{ matrix.version }}"
```

### retry.yaml

重试配置测试：

```yaml
steps:
  - name: Flaky operation
    uses: shell@v1
    timeout-minutes: 5
    retry:
      max-attempts: 3
```

## 编写新测试

### 1. 添加 Build Tag

所有集成测试文件必须包含：

```go
//go:build integration

package integration
```

### 2. 使用共享工具函数

```go
func TestIntegration_MyFeature(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    serverURL := getServerURL()
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()

    // 提交工作流
    resp, err := submitWorkflow(ctx, serverURL, myYAML)
    require.NoError(t, err)

    // 等待完成
    status, err := waitForWorkflowCompletion(ctx, serverURL, resp.WorkflowID, 60*time.Second)
    require.NoError(t, err)

    assert.Equal(t, "completed", status.Status)
}
```

### 3. 添加测试夹具

将工作流 YAML 文件放在 `testdata/workflows/` 目录：

```go
yaml, err := loadTestWorkflow("my-fixture.yaml")
require.NoError(t, err)
```

## 故障排除

### 测试超时

```bash
# 增加超时时间
go test -v -tags integration -timeout 10m ./test/integration/...
```

### 服务未就绪

```bash
# 检查服务状态
curl http://localhost:18080/health
curl http://localhost:18081/health

# 查看日志
docker-compose -f deployments/docker-compose.test.yaml logs server
docker-compose -f deployments/docker-compose.test.yaml logs agent
```

### 数据库连接问题

```bash
# 检查 PostgreSQL
docker-compose -f deployments/docker-compose.test.yaml exec postgres pg_isready

# 重置环境
docker-compose -f deployments/docker-compose.test.yaml down -v
docker-compose -f deployments/docker-compose.test.yaml up -d
```

## CI/CD 集成

在 GitHub Actions 中使用：

```yaml
integration-tests:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.22'
    
    - name: Run integration tests
      run: make integration-test
```

## 相关文档

- [测试策略](../../docs/development.md)
- [部署指南](../../docs/deployment.md)
- [API 文档](../../docs/api-guide.md)
