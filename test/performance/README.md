# Performance Testing

性能基准测试框架，用于验证 Waterflow 系统满足 PRD 性能指标 (NFR2)。

## ⚠️ 前置条件

**无需额外环境:**
- ✅ Go 基准测试 (pkg/dsl/*_bench_test.go) - 可直接运行

**需要 Server + Temporal 环境:**
- ⚠️ API 性能测试 (throughput_test.go)
- ⚠️ 吞吐量测试 (TestWorkflowThroughput)
- ⚠️ 延迟测试 (TestAPILatency)
- ⚠️ 并发连接测试 (TestConcurrentAgentConnections)

**启动环境:**
```bash
# 1. 启动 Temporal (使用 Docker Compose)
docker-compose -f deployments/docker-compose.yaml up -d temporal

# 2. 启动 Waterflow Server
./bin/server

# 3. 等待服务就绪
curl http://localhost:8080/health
```

## 性能目标 (PRD NFR2)

### API 性能
- ✅ P50 响应时间 < 200ms
- ✅ P99 响应时间 < 500ms  
- ✅ 工作流提交吞吐量 > 100/秒

### DSL 解析性能
- ✅ YAML 解析 (1000行) < 100ms (实际: ~16ms)
- ✅ 内存分配 < 10MB (实际: ~2.9MB)

### Agent 性能
- ✅ 空闲内存 < 50MB
- ✅ Event History 查询延迟 < 100ms

### Server 性能
- ✅ 启动时间 < 5 秒
- ✅ 支持 ≥100 个并发 Agent 连接

## 目录结构

```
test/performance/
├── README.md                      # 本文档
├── throughput_test.go             # API 性能测试 (AC1, AC3)
├── agent_memory_test.go           # Agent 内存测试 (AC4)
├── concurrent_agents_test.go      # 并发 Agent 连接测试 (AC5)
├── event_history_latency_test.go  # Event History 查询延迟 (AC6)
├── server_startup_test.go         # Server 启动时间测试 (AC9)
└── server_stateless_test.go       # Server 无状态性能测试 (AC8)
```

## 快速开始

### 1. 运行 Go 基准测试

```bash
# DSL 解析性能 (AC2)
go test -bench='BenchmarkValidate(1000|Memory)' -benchmem ./pkg/dsl -run=^$

# 所有 DSL 基准测试
go test -bench=. -benchmem ./pkg/dsl -run=^$
```

### 2. 运行 API 延迟测试 (AC1)

```bash
# 启动 Server
./bin/server &

# 运行 API 延迟测试
SERVER_URL=http://localhost:8080 go test -v ./test/performance \
    -run=TestAPILatency -timeout=5m

# 期望: P50 <200ms, P99 <500ms
```

### 3. 运行吞吐量测试 (AC3)

```bash
# Go 测试 (需要 Server 运行)
SERVER_URL=http://localhost:8080 go test -v ./test/performance \
    -run=TestWorkflowThroughput -timeout=5m

# 期望: >100 workflows/sec, <1% error rate
```

### 4. 运行 Agent 内存测试 (AC4)

```bash
# 启动 Agent
./bin/agent &
AGENT_PID=$!

# 运行内存测试
AGENT_PID=$AGENT_PID go test -v ./test/performance \
    -run=TestAgentMemoryUsage -timeout=2m

# 期望: RSS <50MB, Heap <30MB
```

### 5. 运行并发 Agent 连接测试 (AC5)

```bash
# Server 必须运行
SERVER_URL=http://localhost:8080 go test -v ./test/performance \
    -run=TestConcurrentAgentConnections -timeout=5m

# 期望: ≥100 agents connected, success rate >95%
```

### 6. 运行 Event History 延迟测试 (AC6)

```bash
# Temporal 和 Server 必须运行
TEMPORAL_HOST=localhost:7233 \
SERVER_URL=http://localhost:8080 \
go test -v ./test/performance \
    -run=TestEventHistoryQueryLatency -timeout=5m

# 期望: P99 <100ms
```

### 7. 运行 Server 启动时间测试 (AC9)

```bash
# 确保 Server 已编译
make build-server

# 运行启动时间测试
SERVER_BINARY=./bin/server go test -v ./test/performance \
    -run=TestServerStartupTime -timeout=30s

# 期望: <5 秒
```

### 8. 运行 Server 无状态性能测试 (AC8)

```bash
# Server 必须运行
SERVER_BINARY=./bin/server go test -v ./test/performance \
    -run=TestServerStatelessPerformance -timeout=10m

# 期望: 性能退化 <10%
```

## 测试命名规范

### 测试函数

- **格式:** `TestPerf_{Feature}_{Metric}` 或 `Test{Feature}{Metric}`
- **示例:** 
  - `TestAPILatency` (AC1)
  - `TestWorkflowThroughput` (AC3)
  - `TestAgentMemoryUsage` (AC4)
  - `TestConcurrentAgentConnections` (AC5)

### Benchmark 函数

- **格式:** `Benchmark{Feature}_{Scenario}`
- **示例:** 
  - `BenchmarkWorkflowSubmit_100Concurrent`
  - `BenchmarkServerStartup`

## PRD AC 追溯表

| AC编号 | PRD 需求 | 测试文件 | 状态 |
|--------|---------|---------|------|
| AC1 | P50 < 200ms, P99 < 500ms | `throughput_test.go::TestAPILatency` | ✅ 已实现 |
| AC2 | YAML 1000行 < 100ms | `pkg/dsl/validator_bench_test.go` | ✅ 已实现 |
| AC3 | 吞吐量 > 100/秒 | `throughput_test.go::TestWorkflowThroughput` | ✅ 已实现 |
| AC4 | Agent 空闲内存 < 50MB | `agent_memory_test.go::TestAgentMemoryUsage` | ✅ 已补充 |
| AC5 | 支持 100+ 并发 Agent | `concurrent_agents_test.go::TestConcurrentAgentConnections` | ✅ 已补充 |
| AC6 | Event History < 100ms | `event_history_latency_test.go::TestEventHistoryQueryLatency` | ✅ 已补充 |
| AC7 | 可重复基准测试 | 所有测试可通过 `go test` 重复运行 | ✅ 已实现 |
| AC8 | Server 无状态性能 | `server_stateless_test.go::TestServerStatelessPerformance` | ✅ 已补充 |
| AC9 | Server 启动 < 5秒 | `server_startup_test.go::TestServerStartupTime` | ✅ 已补充 |

## 参考

- [PRD 性能需求](../../docs/prd.md#目标-2-技术可行性)
- [Story 7.3: 性能基准测试](../../docs/sprint-artifacts/7-3-performance-benchmarking.md)
- [测试标准文档](../../docs/test-review.md)
