# Waterflow 验收测试 (Acceptance Tests)

验收测试是 BMad 4 层测试金字塔的**最顶层**，从**用户视角**验证系统是否满足 **PRD 定义的业务场景**。

## 测试定位

| 层级 | 范围 | 验证内容 | 来源 |
|------|------|---------|------|
| Unit | 单个函数/类 | 代码逻辑正确性 | 代码实现 |
| Integration | 组件交互 | Story AC 验证 | epics.md Story AC |
| E2E | 完整技术管道 | Story 端到端流程 | epics.md Story 流程 |
| **Acceptance** | **PRD 业务场景** | **用户需求满足度** | **prd.md 使用场景** |

**核心特点**：
- ✅ 验证 **PRD 定义的真实业务场景**（而非技术实现）
- ✅ 从 **用户角度** 验证功能可用性
- ✅ 测试 **端到端业务流程**（包括多个 Story 组合）
- ✅ 使用 **真实环境和真实数据**
- ✅ 数量最少，执行时间最长

## 测试场景 (来自 PRD)

根据 [prd.md](../../docs/prd.md) 第 261-264 行定义，Waterflow 有 **2 个核心验收场景**：

### 场景 1: 多服务器健康检查

**PRD 定义 (prd.md#L261):**  
> 在 3 台服务器上并发执行脚本，实时显示进度，结果聚合到单一报告。

**业务价值：** 验证分布式任务执行和结果聚合能力

**测试文件:** [health_check_test.go](health_check_test.go)

**验证点：**
- ✅ 在 3 台服务器上并发执行脚本
- ✅ 实时显示执行进度
- ✅ 收集 CPU、内存、磁盘使用率
- ✅ 结果聚合到单一健康报告
- ✅ 测试在 5 分钟内完成

**涉及功能（Story 组合）：**
- Story 1.1: Server 框架
- Story 1.2: REST API
- Story 1.3: YAML DSL
- Story 1.8: Temporal 集成

---

### 场景 2: 分布式应用部署

**PRD 定义 (prd.md#L264):**  
> 在不同服务器组部署 Web 应用和数据库，按依赖顺序执行，健康检查验证，失败自动重试和回滚

**业务价值：** 验证复杂工作流编排和可靠性保证

**测试文件:** [distributed_deploy_test.go](distributed_deploy_test.go)

**验证点：**
- ✅ 在不同服务器组部署 Web 应用和数据库
- ✅ 按依赖顺序执行 (Database → Application)
- ✅ 执行健康检查验证部署成功
- ✅ 失败时自动重试
- ✅ 支持失败回滚机制
- ✅ 测试在 10 分钟内完成

**涉及功能（Story 组合）：**
- Story 1.3: YAML DSL (依赖关系)
- Story 1.5: 条件执行
- Story 1.6: Matrix 策略
- Story 1.7: 超时和重试
- Story 1.8: Temporal 集成
- Story 1.9: 工作流管理 API

---

## 目录结构

```
test/acceptance/
├── README.md                         # 本文档
├── health_check_test.go              # 场景 1: 多服务器健康检查
├── distributed_deploy_test.go        # 场景 2: 分布式应用部署
├── helpers.go                        # 测试辅助函数
├── report.go                         # 测试报告生成器
├── docker-compose.acceptance.yaml    # 验收测试环境配置
└── testdata/
    ├── workflows/
    │   ├── health-check.yaml         # 健康检查工作流定义
    │   └── distributed-deploy.yaml   # 分布式部署工作流定义
    └── mock-app/                     # 模拟应用
        ├── Dockerfile
        └── main.go
```

## BMad Tea 规范符合性

### 测试来源追溯

| 测试场景 | PRD 定义 | 验收标准 | 测试文件 |
|---------|---------|---------|---------|
| 场景 1: 多服务器健康检查 | [prd.md#L261](../../docs/prd.md) | PRD 第 261 行 | [health_check_test.go](health_check_test.go) |
| 场景 2: 分布式应用部署 | [prd.md#L264](../../docs/prd.md) | PRD 第 264 行 | [distributed_deploy_test.go](distributed_deploy_test.go) |

### 测试命名规范

**格式:** `TestAcceptance_{Scenario}`

- `TestAcceptance_HealthCheck` - 场景 1 健康检查
- `TestAcceptance_DistributedDeploy` - 场景 2 分布式部署
- `TestAcceptance_DeployRollback` - 场景 2 回滚机制验证

### 测试编写原则

1. **从用户视角** - 测试业务流程而非技术实现
2. **完整场景覆盖** - 验证 PRD 定义的所有验收标准
3. **真实环境** - 使用完整的 Waterflow 栈（Server + Temporal + Agent）
4. **可重复执行** - 每次测试独立，清理测试数据
5. **清晰断言** - 验证业务结果（如"部署成功"）而非技术细节

### Build Tags

```go
//go:build acceptance
```

使用 build tag 隔离验收测试，避免与其他测试层级混淆。



## 运行测试

### 快速运行

```bash
# 在项目根目录运行
cd /data/Waterflow

# 运行所有验收测试
make acceptance-test

# 或使用脚本
./scripts/run-acceptance-tests.sh
```

### 手动运行

```bash
# 1. 启动验收测试环境
cd test/acceptance
docker compose -f docker-compose.acceptance.yaml up -d

# 2. 等待服务就绪（约 30 秒）
# Server: http://localhost:18080/health
# Temporal: localhost:17233

# 3. 运行测试
cd /data/Waterflow
SERVER_URL=http://localhost:18080 \
TEMPORAL_HOST=localhost:17233 \
go test -v -tags=acceptance ./test/acceptance/...

# 4. 清理环境
cd test/acceptance
docker compose -f docker-compose.acceptance.yaml down -v
```

### 运行单个场景

```bash
# 场景 1: 健康检查
go test -v -tags=acceptance ./test/acceptance -run TestAcceptance_HealthCheck

# 场景 2: 分布式部署
go test -v -tags=acceptance ./test/acceptance -run TestAcceptance_DistributedDeploy
```

## 环境要求

### 依赖服务

- **Docker** 和 **Docker Compose** - 启动测试环境
- **Temporal Server** - 工作流引擎（通过 docker-compose 启动）
- **PostgreSQL** - Event Sourcing 存储
- **Waterflow Server** - REST API 服务
- **Waterflow Agent** - 分布式执行节点（3 个实例）

### 端口配置

| 服务 | 端口 | 用途 |
|------|------|------|
| Waterflow Server | 18080 | HTTP API |
| Temporal Server | 17233 | gRPC |
| Temporal UI | 18233 | Web UI（调试用）|
| PostgreSQL | 15432 | 数据库 |

### 资源要求

- CPU: 至少 4 核
- 内存: 至少 4GB
- 磁盘: 至少 2GB 可用空间

## 测试报告

测试完成后会生成详细报告：

```bash
# 报告位置
test/acceptance/reports/acceptance-report-{timestamp}.json

# 报告内容
{
  "scenario": "health_check",
  "status": "passed",
  "duration": "125.3s",
  "assertions": [
    {"name": "concurrent_execution", "passed": true},
    {"name": "progress_tracking", "passed": true},
    {"name": "result_aggregation", "passed": true}
  ]
}
```

## 故障排查

### 常见问题

**1. Temporal 连接失败**
```bash
# 检查 Temporal 是否就绪
curl http://localhost:18233/
docker logs acceptance-temporal-1
```

**2. Agent 未注册**
```bash
# 检查 Agent 日志
docker logs acceptance-agent-1
docker logs acceptance-agent-2
docker logs acceptance-agent-3

# 验证 Agent 心跳
curl http://localhost:18080/v1/nodes
```

**3. 工作流执行超时**
```bash
# 增加超时时间
export ACCEPTANCE_TIMEOUT=600s

# 查看 Temporal UI
open http://localhost:18233
```

## 添加新场景

当 PRD 新增业务场景时：

1. **在 PRD 中定义场景** - 更新 `docs/prd.md`
2. **创建工作流 YAML** - 添加到 `testdata/workflows/`
3. **编写测试代码** - 创建 `{scenario}_test.go`
4. **更新本 README** - 添加场景说明

### 测试模板

```go
//go:build acceptance

package acceptance

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

// TestAcceptance_YourScenario tests PRD Scenario X: Your scenario description
// PRD Reference: prd.md#LXX
func TestAcceptance_YourScenario(t *testing.T) {
    // 1. Setup environment
    env := setupAcceptanceEnvironment(t)
    defer env.Cleanup()

    // 2. Submit workflow
    workflowID := submitWorkflow(t, env, "testdata/workflows/your-scenario.yaml")

    // 3. Verify business outcome
    assert.Eventually(t, func() bool {
        // Check business result
        return workflowCompleted(env, workflowID)
    }, 5*time.Minute, 5*time.Second)

    // 4. Generate report
    generateReport(t, "your_scenario", workflowID)
}
```

## 参考

- [PRD 使用场景](../../docs/prd.md#场景)
- [测试标准文档](../../docs/test-review.md)
- [BMad 测试规范](../../docs/bmad-testing-standards.md)
SERVER_URL=http://localhost:18080 go test -v -tags acceptance ./test/acceptance/epic1/...

# 4. 运行特定场景
SERVER_URL=http://localhost:18080 go test -v -tags acceptance ./test/acceptance/epic1/ -run TestAcceptance_HealthCheck

# 5. 清理环境
cd test/acceptance/epic1
docker compose -f docker-compose.acceptance.yaml down -v
```

### 调试模式

```bash
# 保留环境不清理
./scripts/run-acceptance-tests.sh --epic 1 --keep-env

# 查看 Server 日志
docker compose -f test/acceptance/epic1/docker-compose.acceptance.yaml logs -f waterflow-server

# 查看 Agent 日志
docker compose -f test/acceptance/epic1/docker-compose.acceptance.yaml logs -f agent-web-1
```

## 测试环境架构

```
┌─────────────────────────────────────────────────────────────────┐
│              Epic 1 Acceptance Test Environment                │
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

### Agent 配置

| Agent ID | Hostname | Task Queue | 用途 |
|----------|----------|------------|------|
| agent-web-1 | web-server-1 | linux-amd64 | Web 应用部署 |
| agent-web-2 | web-server-2 | linux-amd64 | Web 应用部署 |
| agent-db-1 | db-server-1 | db-server | 数据库部署 |

## 验证检查项

运行测试前，请确保：

- ✅ Docker 和 Docker Compose 已安装
- ✅ 端口未被占用: 18080 (Server), 17233 (Temporal), 5432 (PostgreSQL)
- ✅ 有足够的系统资源（推荐: 4GB+ RAM）
- ✅ 网络连接正常（需要拉取 Docker 镜像）

## 测试报告

测试完成后，可以在以下位置查看报告：

```
epic1/reports/
├── acceptance-report-{timestamp}.md   # Markdown 格式报告
└── test-output.txt                    # 原始测试输出
```

## 故障排查

### Server 启动失败
```bash
# 检查 Server 日志
docker compose -f test/acceptance/epic1/docker-compose.acceptance.yaml logs waterflow-server

# 验证端口未被占用
lsof -i :18080
```

### Agent 连接失败
```bash
# 检查 Agent 日志
docker compose -f test/acceptance/epic1/docker-compose.acceptance.yaml logs agent-web-1

# 验证 Temporal 连接
docker compose -f test/acceptance/epic1/docker-compose.acceptance.yaml exec agent-web-1 nc -zv temporal 7233
```

### 工作流执行超时
```bash
# 查看 Temporal UI (如已启用)
# http://localhost:8088

# 检查工作流状态
curl http://localhost:18080/api/v1/workflows
```

## 参考文档

- [验收测试总览](../README.md)
- [Epic 1 定义](../../../docs/epics.md#epic-1)
- [集成测试](../../integration/epic1/README.md)
- [E2E 测试](../../e2e/README.md)
