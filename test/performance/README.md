# Performance Testing

性能基准测试框架,用于验证 Waterflow 系统满足 PRD 性能指标 (NFR2)。

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

## 快速开始

### 1. 运行 Go 基准测试

```bash
# DSL 解析性能 (AC2)
go test -bench='BenchmarkValidate(1000|Memory)' -benchmem ./pkg/dsl -run=^$

# 所有 DSL 基准测试
go test -bench=. -benchmem ./pkg/dsl -run=^$
```

### 2. 运行 API 负载测试

**使用 curl (无需额外工具):**

```bash
# 启动 Server
./bin/server &

# 运行 API 基准测试
./test/performance/api_benchmark.sh
```

**使用专业工具 (推荐):**

```bash
# 安装 hey 和 vegeta
./test/performance/install_tools.sh

# 使用 vegeta 运行精确测试
./test/performance/vegeta_benchmark.sh
```

### 3. 运行吞吐量测试 (AC3)

```bash
# Go 测试 (需要 Server 运行)
SERVER_URL=http://localhost:8080 go test -v ./test/performance \
    -run=TestWorkflowThroughput -timeout=5m

# 期望: >100 workflows/sec, <1% error rate
```

### 4. 运行 API 延迟测试 (AC1)

```bash
SERVER_URL=http://localhost:8080 go test -v ./test/performance \
    -run=TestAPILatency -timeout=5m

# 期望: P50 <200ms, P99 <500ms
```

## 测试结构

```
test/performance/
├── api_benchmark.sh        # API 负载测试脚本 (curl)
├── install_tools.sh        # 安装 hey/vegeta
├── throughput_test.go      # 吞吐量和延迟测试 (Go)
├── results/                # 测试结果输出
└── baseline/               # 性能基线数据

pkg/dsl/*_bench_test.go     # DSL 组件基准测试
testdata/benchmark/         # 测试 YAML 文件
├── small.yaml              # ~27 lines
├── medium.yaml             # ~164 lines
├── large.yaml              # ~907 lines
└── xlarge.yaml             # ~1000+ lines
```

## 基准测试分类

### Level 1: 组件级 (Go Benchmark)
- **DSL 解析**: `pkg/dsl/validator_bench_test.go`
- **表达式引擎**: `pkg/dsl/expr_engine_bench_test.go`
- **Matrix 扩展**: `pkg/dsl/matrix_bench_test.go`

**运行:**
```bash
go test -bench=. -benchmem ./pkg/dsl
```

### Level 2: API 级 (HTTP Load Testing)
- POST /v1/workflows (提交工作流)
- GET /v1/workflows/{id} (查询状态)
- POST /v1/validate (验证 YAML)

**运行:**
```bash
./test/performance/api_benchmark.sh
```

### Level 3: 集成级 (End-to-End)
- 完整工作流执行
- 并发提交测试
- 吞吐量验证

**运行:**
```bash
go test -v ./test/performance -timeout=5m
```

## 性能基线

当前性能基线 (2026-01-08):

```json
{
  "dsl": {
    "parse_1000_lines_ns": 16133896,
    "parse_1000_lines_ms": 16,
    "memory_bytes": 2943986,
    "memory_mb": 2.9,
    "allocs": 41773
  },
  "api": {
    "submit_workflow_avg_ms": "TBD",
    "get_status_avg_ms": "TBD",
    "validate_avg_ms": "TBD"
  }
}
```

## 环境变量

- `SERVER_URL` - Server 地址 (默认: http://localhost:8080)
- `REQUESTS` - 请求数量 (默认: 100)
- `CONCURRENCY` - 并发数 (默认: 10)
- `WORKFLOW_FILE` - 测试工作流文件 (默认: examples/hello-world.yaml)

## 性能优化建议

### 已知优化
1. **Zap Logger** - 零分配日志 (Story 7.2)
2. **DSL 缓存** - 避免重复解析
3. **连接池** - Temporal gRPC 连接复用

### 潜在优化 (Post-MVP)
1. **响应缓存** - GET /v1/workflows/{id}
2. **YAML 解析优化** - 使用更快的 YAML 库
3. **并发控制** - 限制并发 Temporal 调用

## CI/CD 集成

性能测试已集成到 CI/CD 流程:

```bash
# Makefile 目标
make benchmark           # 运行所有基准测试
make benchmark-baseline  # 建立性能基线
make benchmark-compare   # 对比性能回归
```

## 故障排查

### Server 未运行
```bash
❌ Server is not running at http://localhost:8080
```
**解决:** 启动 Server: `./bin/server`

### 性能未达标
```bash
❌ P99 latency 650ms exceeds 500ms target
```
**检查:**
1. 系统负载 (`top`, `htop`)
2. Temporal Server 状态
3. 网络延迟

### 吞吐量低
```bash
❌ Throughput 80/sec below target 100/sec
```
**优化:**
1. 增加并发数
2. 检查资源限制 (CPU, Memory)
3. 调优 Temporal 配置

## 参考

- [PRD NFR2: 性能要求](../../docs/prd.md#nfr2-性能)
- [Story 7.3: 性能基准测试](../../docs/sprint-artifacts/7-3-performance-benchmarking.md)
- [Go Benchmark 文档](https://pkg.go.dev/testing#hdr-Benchmarks)

---

**创建日期:** 2026-01-08  
**维护者:** DevOps Team
