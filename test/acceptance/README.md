# Acceptance Testing

本目录包含 Waterflow 的验收测试，用于验证 PRD 定义的关键场景。

## 目录结构

```
test/acceptance/
├── README.md                           # 本文档
├── docker-compose.acceptance.yaml      # 多 Agent 测试环境
├── helpers.go                          # 测试辅助函数
├── report.go                           # 测试报告生成器
├── scenario_health_check_test.go       # 场景1: 多服务器健康检查
├── scenario_distributed_deploy_test.go # 场景2: 分布式应用部署
├── reports/                            # 生成的测试报告
└── testdata/
    ├── workflows/
    │   ├── health-check.yaml           # 健康检查工作流
    │   └── distributed-deploy.yaml     # 分布式部署工作流
    └── mock-app/                       # 模拟应用
        ├── Dockerfile
        └── main.go
```

## PRD 验收场景

### 场景 1: 多服务器健康检查

**PRD 定义:**
> 在 3 台服务器上并发执行脚本，实时显示进度，结果聚合到单一报告

**验收标准:**
- ✅ 在 3 台服务器上并发执行脚本
- ✅ 实时显示执行进度
- ✅ 收集 CPU、内存、磁盘使用率
- ✅ 结果聚合到单一健康报告
- ✅ 测试在 5 分钟内完成

**测试文件:** `scenario_health_check_test.go`

### 场景 2: 分布式应用部署

**PRD 定义:**
> 在不同服务器组部署 Web 应用和数据库，按依赖顺序执行，健康检查验证，失败自动重试和回滚

**验收标准:**
- ✅ 在不同服务器组部署 Web 应用和数据库
- ✅ 按依赖顺序执行 (Database → Application)
- ✅ 执行健康检查验证部署成功
- ✅ 失败时自动重试
- ✅ 支持失败回滚机制
- ✅ 测试在 10 分钟内完成

**测试文件:** `scenario_distributed_deploy_test.go`

## 运行验收测试

### 方式一：使用脚本 (推荐)

```bash
# 完整运行：启动环境、运行测试、生成报告、清理
make acceptance-test

# 或直接执行脚本
./scripts/run-acceptance-tests.sh

# 保留环境用于调试
./scripts/run-acceptance-tests.sh --keep-env

# 运行特定场景
./scripts/run-acceptance-tests.sh --scenario TestAcceptance_HealthCheck
```

### 方式二：手动运行

```bash
# 1. 启动验收测试环境
docker compose -f test/acceptance/docker-compose.acceptance.yaml up -d

# 2. 等待服务就绪
# Server: http://localhost:18080/health
# Temporal: localhost:17233

# 3. 运行测试
SERVER_URL=http://localhost:18080 go test -v -tags acceptance ./test/acceptance/...

# 4. 清理环境
docker compose -f test/acceptance/docker-compose.acceptance.yaml down -v
```

### 方式三：使用 Makefile

```bash
# 完整运行
make acceptance-test

# 仅运行测试（环境已启动）
make acceptance-test-only

# 运行特定场景
make acceptance-test-scenario SCENARIO=TestAcceptance_HealthCheck
```

## 测试环境架构

```
┌─────────────────────────────────────────────────────────────────┐
│              Docker Compose Acceptance Environment              │
│                                                                 │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────────────────┐ │
│  │ PostgreSQL│  │ Temporal │  │ Waterflow Server :18080       │ │
│  │   :5432   │←─│  :17233  │←─│                               │ │
│  └──────────┘  └──────────┘  └───────────────────────────────┘ │
│                      ↑                                          │
│                      │ Task Queues                              │
│  ┌───────────────────┴───────────────────────────────────────┐ │
│  │                   Waterflow Agents                         │ │
│  │                                                            │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │ │
│  │  │  agent-1    │  │  agent-2    │  │  agent-3    │       │ │
│  │  │  web-1      │  │  web-2      │  │  db-1       │       │ │
│  │  │ linux-amd64 │  │ linux-amd64 │  │ db-server   │       │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘       │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
           ↑
           │ HTTP API
    ┌──────────────┐
    │ Acceptance   │
    │ Test Runner  │
    └──────────────┘
```

### Agent Task Queue 映射

| Agent | Hostname | Task Queue |
|-------|----------|------------|
| agent-web-1 | web-server-1 | linux-amd64 |
| agent-web-2 | web-server-2 | linux-amd64 |
| agent-db-1 | db-server-1 | db-server |

## 测试报告

测试完成后，报告生成在 `test/acceptance/reports/` 目录：

```
reports/
├── acceptance-report-20260123-143000.md
└── test-output.txt
```

### 报告格式

```markdown
# Waterflow Acceptance Test Report

**Date:** 2026-01-23 14:30:00 UTC
**Version:** v1.0.0

## Summary

| Scenario | Status | Duration |
|----------|--------|----------|
| Multi-Server Health Check | ✅ PASSED | 3m 42s |
| Distributed Stack Deployment | ✅ PASSED | 7m 15s |

**Total:** 2/2 passed (100%)

## Scenario Details
...
```

## 编写新验收场景

### 1. 添加 Build Tag

```go
//go:build acceptance

package acceptance
```

### 2. 遵循命名规范

```go
func TestAcceptance_ScenarioName(t *testing.T) {
    // ...
}
```

### 3. 使用辅助函数

```go
func TestAcceptance_NewScenario(t *testing.T) {
    serverURL := getServerURL()
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    // 提交工作流
    resp, err := submitWorkflow(ctx, serverURL, workflowYAML)
    require.NoError(t, err)

    // 等待完成
    status, err := waitForWorkflowCompletion(ctx, serverURL, resp.WorkflowID, 5*time.Minute)
    require.NoError(t, err)

    assert.True(t, isSuccessStatus(status.Status))
}
```

### 4. 添加 Fixture

将工作流 YAML 放在 `testdata/workflows/` 目录。

## 环境变量

| 变量 | 默认值 | 描述 |
|------|--------|------|
| `SERVER_URL` | `http://localhost:18080` | Waterflow Server 地址 |
| `TEMPORAL_HOST` | `localhost:17233` | Temporal 地址 |

## CI/CD 集成

验收测试作为发布门禁运行：

```yaml
# .github/workflows/release.yml
acceptance-tests:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - name: Run acceptance tests
      run: make acceptance-test
    - name: Upload report
      uses: actions/upload-artifact@v4
      with:
        name: acceptance-report
        path: test/acceptance/reports/
```

## 故障排除

### 测试超时

```bash
# 增加超时时间
go test -v -tags acceptance -timeout 20m ./test/acceptance/...
```

### Agent 未注册

```bash
# 检查 Agent 日志
docker compose -f test/acceptance/docker-compose.acceptance.yaml logs agent-web-1

# 验证 Task Queue
docker compose -f test/acceptance/docker-compose.acceptance.yaml exec temporal \
  tctl taskqueue describe --taskqueue linux-amd64
```

### 工作流执行失败

```bash
# 查看 Server 日志
docker compose -f test/acceptance/docker-compose.acceptance.yaml logs waterflow-server

# 查看工作流日志
curl http://localhost:18080/v1/workflows/{id}/logs
```

## 相关文档

- [PRD - 验收测试场景](../../docs/prd.md)
- [集成测试](../integration/README.md)
- [部署指南](../../docs/deployment.md)
