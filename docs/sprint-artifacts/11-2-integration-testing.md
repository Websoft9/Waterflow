# Story 11.2: 集成测试

**Status:** code-complete

## Story

As a **质量工程师**,
I want **端到端集成测试**,
So that **验证系统整体功能**。

## Acceptance Criteria

### AC1: 测试工作流提交到执行完成全流程

**Given** 完整系统部署 (Server + Temporal + Agent)  
**When** 提交 YAML 工作流  
**Then** 工作流成功提交并返回 workflow_id  
**And** 工作流状态从 pending → running → completed  
**And** 可以查询工作流日志  
**And** 工作流输出正确

### AC2: 测试 Agent 注册和任务执行

**Given** Agent 连接到 Temporal  
**When** Agent 启动  
**Then** Agent 注册到指定 Task Queue  
**And** Agent 接收并执行任务  
**And** 任务结果正确返回给 Server  
**And** Agent 心跳正常

### AC3: 测试多节点工作流

**Given** 工作流包含多个 Step  
**When** 执行多节点工作流  
**Then** 每个 Step 按顺序执行  
**And** Step 之间可以传递数据 (outputs)  
**And** 条件执行 (if) 正确生效  
**And** Matrix 并行执行正确

### AC4: 测试故障重试机制

**Given** 工作流配置重试策略  
**When** Step 执行失败  
**Then** 自动重试指定次数  
**And** 重试间隔遵循配置  
**And** 最终失败正确标记状态  
**And** 重试日志正确记录

### AC5: 集成测试可在 CI 中运行

**Given** GitHub Actions CI 环境  
**When** 执行集成测试  
**Then** 测试自动运行无需人工干预  
**And** 测试结果输出清晰  
**And** 失败测试阻止 PR 合并  
**And** 测试执行时间 < 10 分钟

### AC6: 测试环境自动启停

**Given** Docker Compose 测试环境  
**When** 运行集成测试  
**Then** 自动启动测试环境 (docker compose up)  
**And** 等待服务健康检查通过  
**And** 执行测试用例  
**And** 自动清理环境 (docker compose down)

## Tasks / Subtasks

### Task 1: 集成测试基础设施搭建 (AC: #5, #6)

- [x] 1.1 创建 test/integration/ 目录结构规范
- [x] 1.2 创建 docker-compose.test.yaml (轻量级测试环境)
- [x] 1.3 创建 scripts/run-integration-tests.sh 脚本
- [x] 1.4 实现测试环境自动启动和健康检查等待
- [x] 1.5 实现测试环境自动清理
- [x] 1.6 添加 Makefile target: `make integration-test`

### Task 2: 工作流端到端测试 (AC: #1)

- [x] 2.1 创建 test/integration/workflow_e2e_test.go
- [x] 2.2 测试简单工作流提交和执行
- [x] 2.3 测试工作流状态查询 API
- [x] 2.4 测试工作流日志查询 API
- [x] 2.5 测试工作流取消 API
- [x] 2.6 测试工作流重新运行 API
- [x] 2.7 添加工作流超时测试

### Task 3: Agent 集成测试 (AC: #2)

- [x] 3.1 创建 test/integration/agent_test.go
- [x] 3.2 测试 Agent 启动和 Task Queue 注册
- [x] 3.3 测试 Agent 接收并执行简单任务
- [x] 3.4 测试 Agent 心跳机制
- [x] 3.5 测试 Agent 优雅关闭
- [x] 3.6 测试多 Agent 负载均衡

### Task 4: 多节点工作流测试 (AC: #3)

- [x] 4.1 创建 test/integration/multi_step_test.go
- [x] 4.2 测试顺序执行多 Step 工作流
- [x] 4.3 测试 Step 输出传递 (steps.*.outputs)
- [x] 4.4 测试条件执行 (if 表达式)
- [x] 4.5 测试 Matrix 策略并行执行
- [x] 4.6 测试 Job 依赖 (needs)
- [x] 4.7 测试 continue-on-error 行为

### Task 5: 故障重试测试 (AC: #4)

- [x] 5.1 创建 test/integration/retry_test.go
- [x] 5.2 测试默认重试策略 (3 次重试)
- [x] 5.3 测试自定义重试配置
- [x] 5.4 测试指数退避间隔
- [x] 5.5 测试永久性错误不重试
- [x] 5.6 测试超时后重试
- [ ] 5.7 测试 Temporal UI 重试状态显示

### Task 6: 核心节点集成测试 (AC: #3)

- [x] 6.1 审计现有节点集成测试
- [x] 6.2 完善 shell 节点集成测试 (已完善，无需修改)
- [x] 6.3 完善 script 节点集成测试 (已完善，无需修改)
- [x] 6.4 完善 http/request 节点集成测试 (已完善，无需修改)
- [x] 6.5 完善 file/transfer 节点集成测试 (已存在)
- [x] 6.6 完善 docker/exec 节点集成测试 (已完善，312行)
- [x] 6.7 完善 docker/compose 节点集成测试 (已存在)

### Task 7: API 集成测试 (AC: #1)

- [x] 7.1 创建 test/integration/api_test.go
- [x] 7.2 测试 POST /v1/workflows (提交)
- [x] 7.3 测试 GET /v1/workflows/{id} (查询)
- [x] 7.4 测试 GET /v1/workflows (列表)
- [x] 7.5 测试 GET /v1/workflows/{id}/logs (日志)
- [x] 7.6 测试 POST /v1/workflows/{id}/cancel (取消)
- [x] 7.7 测试 POST /v1/workflows/{id}/rerun (重新运行)
- [x] 7.8 测试 POST /v1/validate (验证)
- [x] 7.9 测试 GET /v1/nodes (节点列表)

### Task 8: 测试数据和 Fixtures (AC: #1, #3)

- [x] 8.1 创建 test/integration/testdata/ 目录
- [x] 8.2 创建简单工作流 YAML fixtures
- [x] 8.3 创建多 Step 工作流 YAML fixtures
- [x] 8.4 创建 Matrix 工作流 YAML fixtures
- [x] 8.5 创建故障重试工作流 YAML fixtures
- [x] 8.6 创建条件执行工作流 YAML fixtures

### Task 9: CI/CD 集成 (AC: #5)

- [x] 9.1 更新 .github/workflows/ 添加集成测试 job
- [x] 9.2 配置 Docker Compose 服务启动
- [x] 9.3 添加测试超时控制 (10 分钟)
- [x] 9.4 配置测试结果报告
- [x] 9.5 配置失败时保存日志 artifact

### Task 10: 测试报告和文档 (AC: #5)

- [x] 10.1 添加测试报告生成 (gotestsum)
- [x] 10.2 创建 test/integration/README.md
- [x] 10.3 文档化测试环境要求
- [x] 10.4 文档化如何本地运行集成测试
- [x] 10.5 添加故障排查指南

## Dev Notes

### 现有集成测试资产分析

**test/integration/ 目录 (已有):**
- `health_check_test.go` - 健康检查端点测试
- `parameter_validation_test.go` - 参数验证测试
- `metrics_test.sh` - Metrics 端点 shell 测试

**CLI 集成测试脚本 (cmd/waterflow-cli/):**
- `integration_test.sh` - 基础 CLI 命令测试
- `integration_validate_test.sh` - validate 命令测试
- `integration_submit_test.sh` - submit 命令测试
- `integration_status_test.sh` - status 命令测试
- `integration_logs_test.sh` - logs 命令测试
- `integration_node_list_test.sh` - nodes 命令测试

**节点插件集成测试 (plugins/*/):**
- `plugins/exec/shell/integration_test.go`
- `plugins/exec/script/integration_test.go`
- `plugins/http/request/integration_test.go`
- `plugins/file/transfer/integration_test.go`
- `plugins/docker/exec/integration_test.go`
- `plugins/docker/compose/integration_test.go`
- `plugins/flow/sleep/integration_test.go`

**DSL 集成测试 (pkg/dsl/):**
- `conditional_integration_test.go`
- `integration_state_test.go`
- `matrix_integration_test.go`
- `timeout_retry_integration_test.go`

### 测试环境架构

```
┌─────────────────────────────────────────────────────┐
│              Docker Compose Test Environment        │
│                                                     │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────┐ │
│  │PostgreSQL│  │ Temporal │  │ Waterflow Server  │ │
│  │  :5432   │←─│  :7233   │←─│      :8080        │ │
│  └──────────┘  └──────────┘  └───────────────────┘ │
│                      ↑                              │
│                      │                              │
│              ┌───────────────┐                      │
│              │ Waterflow     │                      │
│              │ Agent         │                      │
│              └───────────────┘                      │
│                                                     │
└─────────────────────────────────────────────────────┘
           ↑
           │ HTTP API
           │
    ┌──────────────┐
    │ Integration  │
    │ Test Runner  │
    └──────────────┘
```

### 集成测试命名规范

```go
// 文件命名
xxx_test.go           // 常规测试
xxx_integration_test.go // 集成测试 (需要外部依赖)

// 函数命名
func TestIntegration_WorkflowSubmit(t *testing.T)     // 集成测试前缀
func TestE2E_WorkflowLifecycle(t *testing.T)          // 端到端测试

// Build tags
// +build integration
package integration
```

### 测试运行方式

```bash
# 本地运行集成测试
make integration-test

# 运行特定测试
go test -v ./test/integration -run TestIntegration_Workflow

# 使用 docker compose 启动测试环境
docker compose -f deployments/docker-compose.test.yaml up -d
go test -v ./test/integration
docker compose -f deployments/docker-compose.test.yaml down

# CI 环境运行
./scripts/run-integration-tests.sh
```

### 测试超时配置

```go
func TestIntegration_WorkflowSubmit(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // 设置测试超时
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    
    // ...
}
```

### 测试数据目录结构

```
test/integration/
├── testdata/
│   ├── workflows/
│   │   ├── simple.yaml
│   │   ├── multi-step.yaml
│   │   ├── matrix.yaml
│   │   ├── retry.yaml
│   │   └── conditional.yaml
│   ├── expected/
│   │   └── workflow-output.json
│   └── invalid/
│       ├── syntax-error.yaml
│       └── validation-error.yaml
├── workflow_e2e_test.go
├── agent_test.go
├── api_test.go
├── multi_step_test.go
├── retry_test.go
└── README.md
```

### Project Structure Notes

- 集成测试使用 `// +build integration` build tag
- 集成测试不与单元测试混合运行
- 使用 `testing.Short()` 跳过耗时测试
- 测试环境使用独立的 docker-compose.test.yaml

### References

- [Source: docs/architecture.md#AR7] - 测试策略: 每个节点的集成测试
- [Source: docs/epics.md#story-11.2] - 集成测试验收标准
- [Source: test/integration/health_check_test.go] - 现有集成测试模式
- [Source: deployments/docker-compose.yaml] - 测试环境配置

---

## Dev Agent Record

### Context Reference

<!-- Path(s) to story context XML will be added here by context workflow -->

### Agent Model Used

Claude Opus 4.5 (via GitHub Copilot)

### Debug Log References

### Completion Notes List

- **Task 1 完成**: 创建了集成测试基础设施
  - `deployments/docker-compose.test.yaml` - 轻量级测试环境
  - `scripts/run-integration-tests.sh` - 测试运行脚本
  - `Makefile` 添加 `integration-test` 和 `integration-test-only` targets

- **Task 2-5, 7 完成**: 创建了完整的集成测试套件
  - `test/integration/common_test.go` - 共享工具函数
  - `test/integration/workflow_e2e_test.go` - 工作流 E2E 测试
  - `test/integration/api_test.go` - API 集成测试
  - `test/integration/agent_test.go` - Agent 集成测试
  - `test/integration/multi_step_test.go` - 多步骤工作流测试
  - `test/integration/retry_test.go` - 重试与容错测试

- **Task 8 完成**: 创建了测试数据夹具
  - 6 个工作流 YAML fixtures 覆盖各种场景

- **Task 9 完成**: CI/CD 集成
  - `.github/workflows/ci.yml` 添加 `integration-tests` job
  - 配置自动启动测试环境、超时控制、日志收集

- **Task 10 部分完成**: 文档
  - `test/integration/README.md` - 完整的集成测试文档

- **待完成**: Task 5.7 (Temporal UI 验证需手动), Task 6 (节点集成测试审计), Task 10.1 (gotestsum)

### File List

- [deployments/docker-compose.test.yaml](deployments/docker-compose.test.yaml) - 新建
- [scripts/run-integration-tests.sh](scripts/run-integration-tests.sh) - 新建
- [test/integration/README.md](test/integration/README.md) - 新建
- [test/integration/common_test.go](test/integration/common_test.go) - 新建
- [test/integration/workflow_e2e_test.go](test/integration/workflow_e2e_test.go) - 新建
- [test/integration/api_test.go](test/integration/api_test.go) - 新建
- [test/integration/agent_test.go](test/integration/agent_test.go) - 新建
- [test/integration/multi_step_test.go](test/integration/multi_step_test.go) - 新建
- [test/integration/retry_test.go](test/integration/retry_test.go) - 新建
- [test/integration/testdata/workflows/simple-echo.yaml](test/integration/testdata/workflows/simple-echo.yaml) - 新建
- [test/integration/testdata/workflows/multi-step-outputs.yaml](test/integration/testdata/workflows/multi-step-outputs.yaml) - 新建
- [test/integration/testdata/workflows/matrix.yaml](test/integration/testdata/workflows/matrix.yaml) - 新建
- [test/integration/testdata/workflows/retry.yaml](test/integration/testdata/workflows/retry.yaml) - 新建
- [test/integration/testdata/workflows/conditional.yaml](test/integration/testdata/workflows/conditional.yaml) - 新建
- [test/integration/testdata/workflows/job-dependencies.yaml](test/integration/testdata/workflows/job-dependencies.yaml) - 新建
- [test/integration/audit-node-integration-tests.md](test/integration/audit-node-integration-tests.md) - 新建
- [Makefile](Makefile) - 修改 (添加 integration-test, integration-test-only, integration-test-report targets)
- [.github/workflows/ci.yml](.github/workflows/ci.yml) - 修改 (添加 integration-tests job)

