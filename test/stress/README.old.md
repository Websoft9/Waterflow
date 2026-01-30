# Stress Testing and Fault Tolerance

压力测试和容错验证测试套件，用于验证 Waterflow 在高负载和故障场景下的可靠性。

## 测试概述

本测试套件验证以下 Production Reliability 要求：

- **AC1**: 1000+ 并发工作流 (成功率 > 99%)
- **AC2**: Server 崩溃后自动恢复 (Event Sourcing)
- **AC3**: 无资源泄漏 (内存/Goroutine/连接)
- **AC4**: Event Sourcing 零状态丢失
- **AC5**: 超时和重试策略正确执行

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
docker-compose up -d temporal

# 2. 启动 Waterflow Server
./bin/server

# 3. 启动 Agent (可选，用于分布式测试)
./bin/agent
```

## 快速开始

### 运行所有压力测试

```bash
# 完整测试套件 (需要约 30 分钟)
make stress-test

# 或者单独运行
./test/stress/run_all_tests.sh
```

### 运行单个测试

```bash
# 1. 并发工作流测试 (AC1)
./test/stress/concurrent_workflows_test.sh

# 2. Server 崩溃恢复测试 (AC2)
./test/stress/server_crash_recovery_test.sh

# 3. 资源泄漏检测 (AC3)
./test/stress/resource_leak_test.sh

# 4. 超时重试验证 (AC5)
./test/stress/timeout_retry_test.sh
```

### 运行 Go 测试

```bash
# 并发测试
go test -v ./test/stress -run TestConcurrentWorkflows

# Benchmark
go test -bench=. ./test/stress
```

## 测试详解

### 1. 并发工作流测试 (AC1)

**目标**: 验证系统在 1000+ 并发工作流下的稳定性

**脚本**: `test/stress/concurrent_workflows_test.sh`

**配置环境变量**:

```bash
export CONCURRENT_WORKFLOWS=1000  # 并发数
export WORKFLOW_FILE=examples/hello-world.yaml
export SERVER_URL=http://localhost:8080
```

**验证标准**:

- 成功率 >= 99%
- CPU < 80%
- Memory < 2GB
- Connections < 1000
- 平均执行时间 < 10s

**示例输出**:

```
=== Concurrent Workflows Stress Test Results ===
Total Workflows: 1000
Success: 998
Failed: 2
Success Rate: 99.80%
Duration: 45s
Throughput: 22.22 workflows/sec

Resource Usage:
  Max CPU: 62.5%
  Max Memory: 1024MB
  Max Connections: 850

✅ Success rate 99.80% meets target
✅ CPU usage 62.5% within limits
✅ Memory usage 1024 MB within limits
```

### 2. Server 崩溃恢复测试 (AC2)

**目标**: 验证 Event Sourcing 容错能力

**脚本**: `test/stress/server_crash_recovery_test.sh`

**测试流程**:

1. 启动 Server
2. 提交 10 个长时运行工作流
3. 等待工作流开始执行
4. Kill -9 强制终止 Server
5. 重启 Server
6. 验证所有工作流恢复并继续执行

**验证标准**:

- 所有工作流恢复
- 恢复时间 < 10s
- 无状态丢失

**示例输出**:

```
=== Recovery Results ===
Submitted Workflows: 10
Recovered: 10
Lost: 0
Recovery Time: 7s
Recovery Rate: 100%

✅ All workflows recovered successfully
✅ Recovery time 7s within 10s limit
✅ No state loss detected (Event Sourcing working)
```

### 3. 资源泄漏检测 (AC3)

**目标**: 验证长时间运行无资源泄漏

**脚本**: `test/stress/resource_leak_test.sh`

**测试流程**:

1. 启动 Server 和资源监控
2. 持续 10 分钟提交工作流
3. 分析资源使用趋势

**验证标准**:

- 内存增长 < 50%
- Goroutine 数量稳定 (增长 < 100%)
- 连接数稳定 (增长 < 100)

**示例输出**:

```
=== Resource Usage Analysis ===
Memory:
  Initial (avg): 512MB
  Final (avg): 628MB
  Growth: 22.66%
  Max: 680MB
  ✅ No memory leak detected

Goroutines:
  Initial (avg): 45
  Final (avg): 52
  Growth: 15.56%
  ✅ No goroutine leak detected

Connections:
  Initial (avg): 25
  Final (avg): 32
  Growth: 7
  ✅ No connection leak detected
```

### 4. 超时重试验证 (AC5)

**目标**: 验证超时和重试策略正确执行

**脚本**: `test/stress/timeout_retry_test.sh`

**测试场景**:

1. **NonRetryableError**: 应快速失败，无重试
2. **Retryable Error**: 应按策略重试 (3次，2s backoff)
3. **Timeout**: 应在指定时间后终止

**验证标准**:

- NonRetryableError 在 10s 内失败
- Retryable error 重试 3 次
- Timeout 正确触发

**示例输出**:

```
Test 1: NonRetryableError handling
✅ Workflow failed in 3s
✅ Failed quickly (no unnecessary retries)

Test 2: Retryable error handling
Workflow failed after 12s
✅ Retried 3 times as expected

Test 3: Timeout handling
✅ Workflow timed out after 11s
✅ Timed out within expected range (~10s + overhead)
```

## Go 测试

### TestConcurrentWorkflows

并发工作流测试的 Go 实现，包含资源监控。

```bash
# 运行测试
go test -v ./test/stress -run TestConcurrentWorkflows

# 配置环境变量
export WATERFLOW_SERVER_URL=http://localhost:8080
export CONCURRENT_WORKFLOWS=500

# 运行测试 (跳过 short 模式)
go test -v ./test/stress -run TestConcurrentWorkflows -timeout 30m
```

### BenchmarkConcurrentWorkflows

并发提交性能基准测试。

```bash
# 运行 benchmark
go test -bench=BenchmarkConcurrentWorkflows ./test/stress

# 配置并发度
go test -bench=. -benchtime=100x -cpu=1,4,8 ./test/stress
```

## 资源监控

### 实时监控脚本

`scripts/monitor_resources.sh` 提供实时资源监控：

```bash
# 启动监控 (5秒间隔)
./scripts/monitor_resources.sh ./results 5

# 输出文件:
# - cpu.log: CPU 使用率 (%)
# - memory.log: 内存使用 (MB)
# - connections.log: TCP 连接数
# - goroutines.log: Goroutine 数量
```

### 分析监控数据

```bash
# 最大值
MAX_CPU=$(sort -n results/cpu.log | tail -1)

# 平均值
AVG_MEM=$(awk '{sum+=$1} END {print sum/NR}' results/memory.log)

# 趋势分析
INITIAL=$(head -10 results/memory.log | awk '{sum+=$1} END {print sum/NR}')
FINAL=$(tail -10 results/memory.log | awk '{sum+=$1} END {print sum/NR}')
GROWTH=$(echo "scale=2; ($FINAL - $INITIAL) / $INITIAL * 100" | bc)
```

## 环境变量

所有测试支持以下环境变量配置：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SERVER_URL` | `http://localhost:8080` | Waterflow Server 地址 |
| `CONCURRENT_WORKFLOWS` | `1000` | 并发工作流数量 |
| `WORKFLOW_FILE` | `examples/hello-world.yaml` | 测试工作流文件 |
| `SERVER_BIN` | `./bin/server` | Server 二进制路径 |

## 故障排查

### Server 启动失败

```bash
# 检查 Temporal 连接
./bin/server --temporal-host=localhost:7233

# 检查端口占用
lsof -i :8080
```

### 测试超时

```bash
# 增加超时时间
export TEST_TIMEOUT=600  # 10 minutes

# 减少并发数
export CONCURRENT_WORKFLOWS=100
```

### 资源监控无数据

```bash
# 检查依赖
command -v top free ss jq bc

# 手动启动监控
./scripts/monitor_resources.sh ./test-results 5 &
```

### 工作流提交失败

```bash
# 验证工作流文件
./bin/waterflow validate examples/hello-world.yaml

# 测试 API 连接
curl -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/yaml" \
  --data-binary @examples/hello-world.yaml
```

## CI/CD 集成

### GitHub Actions

```yaml
name: Stress Tests

on:
  schedule:
    - cron: '0 2 * * *'  # 每天凌晨 2 点
  workflow_dispatch:

jobs:
  stress-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Build
        run: make build
      
      - name: Start Temporal
        run: docker-compose up -d temporal
      
      - name: Run Stress Tests
        run: ./test/stress/run_all_tests.sh
        timeout-minutes: 60
      
      - name: Upload Results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: stress-test-results
          path: test/stress/results/
```

## 性能基准

基于 PRD NFR2 要求，以下是预期性能基准：

| 指标 | 目标 | 验收标准 |
|------|------|----------|
| 并发工作流 | 1000+ | 成功率 > 99% |
| Server 恢复时间 | < 10s | Event Sourcing 恢复 |
| 内存泄漏 | 0 | 10分钟增长 < 50% |
| Goroutine 泄漏 | 0 | 稳定或小幅波动 |
| 连接泄漏 | 0 | 稳定在合理范围 |
| Timeout 精度 | ±2s | 超时误差 |
| Retry 策略 | 按配置 | 重试次数正确 |

## 测试报告

每个测试生成独立报告：

```
test/stress/results/
  concurrent-20260108-143052/
    report.txt         # 测试报告
    cpu.log           # CPU 使用记录
    memory.log        # 内存使用记录
    connections.log   # 连接数记录
    response-*.json   # API 响应
    status-*.txt      # HTTP 状态码
```

## 最佳实践

1. **隔离环境**: 在专用测试环境运行，避免影响生产
2. **资源预留**: 确保足够 CPU/Memory (至少 4 核 8GB)
3. **定期执行**: 每日自动化测试，及时发现回归
4. **基线对比**: 保存历史数据，对比性能趋势
5. **故障演练**: 定期执行崩溃恢复测试，验证 Event Sourcing

## 参考

- [PRD - NFR2: 可靠性](../../docs/prd.md#nfr2-可靠性)
- [ADR-0008: Temporal as Internal Service](../../docs/adr/0008-temporal-as-internal-service.md)
- [Performance Benchmarking](../../test/performance/README.md)
