# Stress Testing and Fault Tolerance

压力测试和容错验证测试套件，用于验证 Waterflow 在高负载和故障场景下的可靠性。

## 测试概述

本测试套件验证 **PRD 技术成功标准** 中的架构质量属性：

### PRD 追溯

| 测试文件 | PRD 追溯 | 质量属性 | 验收标准 |
|---------|---------|---------|---------|
| [concurrent_workflows_test.go](concurrent_workflows_test.go) | [PRD#L238](../../docs/prd.md#L238) | 可扩展性 | 支持 ≥100 个并发 Agent 连接 (测试 1000+) |
| [event_sourcing_test.go](event_sourcing_test.go) | [PRD#L235](../../docs/prd.md#L235) | 持久性 | 崩溃测试中 100% 状态恢复 |
| [event_history_integrity_test.go](event_history_integrity_test.go) | [PRD#L235](../../docs/prd.md#L235) | 持久性 | Event History 完整性验证 |
| [server_crash_recovery_test.go](server_crash_recovery_test.go) | [PRD#L235](../../docs/prd.md#L235) | 持久性 | Server 进程崩溃后状态恢复 |
| [memory_profile_test.go](memory_profile_test.go) | [PRD#L236](../../docs/prd.md#L236) | 可靠性 | 内存使用分析和泄漏检测 |
| [resource_leak_test.go](resource_leak_test.go) | [PRD#L236](../../docs/prd.md#L236) | 可靠性 | Goroutine/连接泄漏检测 |
| [timeout_retry_test.go](timeout_retry_test.go) | [PRD#L249](../../docs/prd.md#L249) | 容错能力 | 超时和重试策略验证 |

## 目录结构

```
test/stress/
├── README.md                          # 本文档
├── concurrent_workflows_test.go       # 并发工作流压力测试 (PRD#L238)
├── event_sourcing_test.go             # Event Sourcing 持久化 (PRD#L235)
├── event_history_integrity_test.go    # Event History 完整性 (PRD#L235)
├── server_crash_recovery_test.go      # Server 崩溃恢复 (PRD#L235)
├── memory_profile_test.go             # 内存分析 (PRD#L236)
├── resource_leak_test.go              # 资源泄漏检测 (PRD#L236)
└── timeout_retry_test.go              # 超时重试验证 (PRD#L249)
```

**重构说明**: 已删除所有 Shell 脚本，统一使用 Go 测试框架（符合 BMad 测试金字塔标准）

## 前置条件

### 系统要求

- Linux 系统 (bash, top, free, ss/netstat)
- Go 1.21+ (用于 Go 测试)
- jq (JSON 解析)
- bc (计算)
- curl (API 调用)

### 服务依赖

测试需要完整的 Waterflow 运行环境：

```bash
# 1. 启动 Temporal
docker-compose -f deployments/docker-compose.yaml up -d temporal

# 2. 启动 Waterflow Server
./bin/server

# 3. 启动 Agent (可选，用于分布式测试)
./bin/agent
```

## 快速开始

### 运行所有压力测试

```bash
# 完整测试套件 (Go 测试标准)
go test -tags=stress -v ./test/stress/...

# 或使用 Makefile
make stress-test
```

### 运行单个测试

```bash
# 1. 并发工作流测试 (PRD#L238)
go test -v ./test/stress -run TestConcurrentWorkflows

# 2. Server 崩溃恢复测试 (PRD#L235)
go test -v ./test/stress -run TestServerCrashRecovery

# 3. 资源泄漏检测 (PRD#L236)
go test -v ./test/stress -run TestResourceLeak

# 4. 超时重试验证 (PRD#L249)
go test -v ./test/stress -run TestTimeoutRetry

# 5. Event Sourcing 验证 (PRD#L235)
go test -v ./test/stress -run TestEventSourcing

# 6. Event History 完整性 (PRD#L235)
go test -v ./test/stress -run TestEventHistoryIntegrity
```

### 性能基准测试

```bash
# 运行所有 Benchmark
go test -bench=. -benchmem ./test/stress/...

# 内存分析
go test -v ./test/stress -run TestMemoryProfile
```
PRD#L238)

**PRD 追溯**: [prd.md#L238](../../docs/prd.md#L238) - 可扩展性: 支持 ≥100 个并发 Agent 连接

**测试文件**: [concurrent_workflows_test.go](concurrent_workflows_test.go)

**验证标准**:
- 并发数: 1000+ 工作流
- 成功率: >= 99%
- CPU 使用: < 80%
- 内存使用: < 2GB
- 平均执行时间: < 10s

**环境变量**:
```bash
export CONCURRENT_WORKFLOWS=1000
export WATERFLOW_SERVER_URL=http://localhost:8080
```

### 2. Server 崩溃恢复 (PRD#L235)

**PRD 追溯**: [prd.md#L235](../../docs/prd.md#L235) - 持久性: 崩溃测试中 100% 状态恢复

**测试文件**: [server_crash_recovery_test.go](server_crash_recovery_test.go)

**验证点**:
- Server 进程 SIGKILL 后重启
- Event History 完整无损 (100% 恢复)
- 工作流状态正确恢复
- 无数据丢失

### 3. 资源泄漏检测 (PRD#L236)

**PRD 追溯**: [prd.md#L236](../../docs/prd.md#L236) - 可靠性: 自动故障处理

**测试文件**: [resource_leak_test.go](resource_leak_test.go)

**监控指标**:
- 内存增长: < 50MB/100 workflows
- Goroutine 增长: < 20 个
- 数据库连接: 正常释放
- 文件描述符: 无泄漏

### 4. Event Sourcing 持久化 (PRD#L235)

**PRD 追溯**: [prd.md#L235](../../docs/prd.md#L235) - 持久性: 工作流状态在进程故障后幸存

**测试文件**: [event_sourcing_test.go](event_sourcing_test.go), [event_history_integrity_test.go](event_history_integrity_test.go)

**验证点**:
- 所有状态变更记录到 Event History
- Event History 可重放
- Temporal 持久化正确
- 零状态丢失

### 5. 超时重试策略 (PRD#L249)

**PRD 追溯**: [prd.md#L249](../../docs/prd.md#L249) - 容错能力: 每步骤可配置超时和重试策略

**测试文件**: [timeout_retry_test.go](timeout_retry_test.go)

**验证点**:
- 超时策略正确触发
- 重试次数符合配置
- 指数退避正确实现 (1s, 2s, 4s, 8s...)
- 最大重试次数限制生效
- 超时策略正确执行
- 重试次数符合配置
- 指数退避正确实现

## 环境变量配置

```bash
# 并发工作流测试
export CONCURRENT_WORKFLOWS=1000
export WATERFLOW_SERVER_URL=http://localhost:8080

# 资源泄漏测试
export LEAK_THRESHOLD_MB=50
export GOROUTINE_THRESHOLD=20

# 超时重试测试
export RETRY_ATTEMPTS=3
export TIMEOUT_SECONDS=30
```

## BMad 测试金字塔对齐

### 测试层级定位

- **层级**: Stress (系统级非功能测试)
- **追溯**: PRD 技术成功标准 (非 Story AC)
- **工具**: Go 测试框架 + Benchmark
- **运行**: `go test -tags=stress`

### 与其他测试层的关系

| 测试层 | 范围 | 追溯 | 工具 |
|-------|------|------|------|
| Unit | 函数/方法 | 实现代码 | Go test |
| Integration | 组件交互 | Story AC | Go test (Epic目录) |
| E2E | 端到端流程 | Story 场景 | Go test |
| **Stress** | **系统级负载/容错** | **PRD 质量属性** | **Go test + Benchmark** |
| Acceptance | PRD 场景 | PRD 验收场景 | Go test |

### Tea 追溯标准

**所有测试必须追溯到 PRD**:
- ✅ 并发测试 → [PRD#L238](../../docs/prd.md#L238)
- ✅ 崩溃恢复 → [PRD#L235](../../docs/prd.md#L235)
- ✅ 资源泄漏 → [PRD#L236](../../docs/prd.md#L236)
- ✅ 超时重试 → [PRD#L249](../../docs/prd.md#L249)

**不依赖 Story/Epic** (Stress 是系统级验证)

## 测试命名规范

### 测试函数

- **格式**: `TestStress_{Feature}_{Scenario}` 或 `Test{Feature}` (简洁)
- **示例**: `TestConcurrentWorkflows`, `TestServerCrashRecovery`
- **注释**: 必须包含 PRD 追溯注释

**示例**:
```go
// TestConcurrentWorkflows tests system behavior under 1000+ concurrent workflows
// 追溯: PRD#L238 - 可扩展性: 支持 ≥100 个并发 Agent 连接
func TestConcurrentWorkflows(t *testing.T) { ... }
```

### Benchmark 函数

- **格式**: `Benchmark{Feature}`
- **示例**: `BenchmarkConcurrentSubmit`

## 参考

- [PRD 可靠性需求](../../docs/prd.md#nfr3-可靠性)
- [测试标准文档](../../docs/test-review.md)
