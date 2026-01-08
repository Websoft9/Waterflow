# Story 7.3: 性能基准测试 (Event Sourcing 架构)

Status: In Progress

**MVP范围**: AC2完全达成 + AC1/AC3测试框架就绪
**Post-MVP**: AC4-AC9 (需完整部署环境)

## Story

As a **开发者**,  
I want **建立性能基准并验证 Event Sourcing 架构性能**,  
So that **验证性能指标达标**。

## Context

这是 Epic 7 (生产级可靠性) 的**第三个 Story**,实现**性能基准测试系统**。该 Story 建立完整的性能测试框架,验证系统在生产环境中的性能表现,确保满足 PRD 中定义的性能指标。

**前置依赖:**
- ✅ Epic 1-6 - 核心功能已实现
- ✅ Story 7.1 - 类型化错误处理
- ✅ Story 7.2 - 结构化日志系统
- ✅ 现有基准测试 - pkg/dsl/*_bench_test.go (需要扩展)

**Epic 背景:**  
Epic 7 专注于**生产级可靠性**。本 Story 建立在前两个 Story 的基础上,提供:
- **完整的性能测试框架** - 涵盖所有关键组件
- **自动化基准测试** - 可重复运行和对比
- **性能回归检测** - 防止性能退化
- **生产就绪验证** - 确保满足 SLA 要求

**业务价值:**
- 🎯 **性能保证** - 验证系统满足 PRD 性能指标
- 🎯 **回归检测** - 及时发现性能退化
- 🎯 **容量规划** - 提供性能基线数据
- 🎯 **优化指导** - 识别性能瓶颈
- 🎯 **生产信心** - 数据驱动的上线决策

**PRD 性能指标 (NFR2):**
```
Server 性能:
- 启动时间 < 5 秒
- YAML 解析 (1000行) < 100ms
- 工作流提交 API 响应 < 500ms
- 支持 ≥100 个并发 Agent 连接

API 性能:
- P50 响应时间 < 200ms
- P99 响应时间 < 500ms
- 工作流提交吞吐量 > 100/秒

Agent 性能:
- 空闲内存 < 50MB
- Event History 查询延迟 < 100ms
```

**性能测试架构:**
```
┌────────────────────────────────────────────────────────────┐
│         Performance Testing Framework                      │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  ┌──────────────────┐  ┌──────────────────┐              │
│  │ Go Benchmarks    │  │ Load Testing     │              │
│  │ (testing.B)      │  │ (hey/vegeta)     │              │
│  ├──────────────────┤  ├──────────────────┤              │
│  │ - pkg/dsl        │  │ - REST API       │              │
│  │ - pkg/temporal   │  │ - Workflow       │              │
│  │ - pkg/logger     │  │ - Concurrent     │              │
│  └──────────────────┘  └──────────────────┘              │
│                                                            │
│  ┌──────────────────────────────────────────────────┐     │
│  │ Performance Metrics Collection                   │     │
│  ├──────────────────────────────────────────────────┤     │
│  │ - Latency (P50, P95, P99)                       │     │
│  │ - Throughput (req/sec)                          │     │
│  │ - Memory (RSS, Heap, Allocs)                    │     │
│  │ - CPU (user, system, total)                     │     │
│  │ - I/O (disk, network)                           │     │
│  └──────────────────────────────────────────────────┘     │
│                                                            │
│  ┌──────────────────────────────────────────────────┐     │
│  │ Baseline & Regression Detection                  │     │
│  ├──────────────────────────────────────────────────┤     │
│  │ - Baseline storage (JSON)                       │     │
│  │ - Automated comparison                          │     │
│  │ - Threshold alerts (±10%)                       │     │
│  └──────────────────────────────────────────────────┘     │
└────────────────────────────────────────────────────────────┘
```

**基准测试分类:**

**1. 组件级基准 (Go Benchmark)**
- DSL 解析和验证
- 表达式引擎求值
- Workflow 转换
- Event History 查询

**2. API 级基准 (Load Testing)**
- POST /v1/workflows (提交工作流)
- GET /v1/workflows/{id} (查询状态)
- GET /v1/workflows/{id}/logs (获取日志)
- GET /v1/validate (验证 YAML)

**3. 端到端基准 (Integration)**
- 完整工作流执行
- 并发工作流提交
- Agent 负载测试

**现有基准测试分析:**
```
已存在的基准测试:
✅ pkg/dsl/validator_bench_test.go
   - BenchmarkValidateSmallWorkflow
   - BenchmarkValidateMediumWorkflow
   - BenchmarkValidateLargeWorkflow
   - BenchmarkParseOnly
   - BenchmarkSchemaValidateOnly

✅ pkg/dsl/expr_engine_bench_test.go
   - BenchmarkEngineEvaluate_SimpleVariable
   - BenchmarkEngineEvaluate_Arithmetic
   - BenchmarkEngineEvaluate_ComplexExpression

✅ pkg/dsl/matrix_bench_test.go
   - BenchmarkMatrixExpansion
   - BenchmarkMatrixExpansion256

需要新增:
🆕 Server API 基准测试
🆕 Temporal 集成基准测试
🆕 端到端工作流基准
🆕 并发性能测试
🆕 内存和资源测试
```

**本 Story 的范围 (MVP):**
- ✅ 扩展现有 Go 基准测试
- ✅ 实现 API 负载测试脚本
- ✅ 实现端到端性能测试
- ✅ 建立性能基线和存储机制
- ✅ 实现自动化回归检测
- ✅ 集成到 CI/CD 流程
- ✅ 性能测试文档和报告
- ❌ 分布式压力测试 - 留待 Story 7.4
- ❌ 实时性能监控 - Post-MVP

## Acceptance Criteria

### AC1: API 响应时间基准 - P50 < 200ms, P99 < 500ms

**Given** Waterflow Server 运行中  
**When** 执行 API 负载测试  
**Then** POST /v1/workflows 响应时间:
- P50 < 200ms
- P95 < 400ms
- P99 < 500ms

**And** GET /v1/workflows/{id} 响应时间:
- P50 < 100ms
- P95 < 200ms
- P99 < 300ms

**And** POST /v1/validate 响应时间:
- P50 < 150ms
- P95 < 300ms
- P99 < 500ms

**Implementation Notes:**

**负载测试工具选择:**
- `hey` - 简单高效的 HTTP 负载测试工具
- `vegeta` - 功能更强大,支持 JSON 结果

**测试脚本:**
```bash
#!/bin/bash
# test/performance/api_benchmark.sh

set -e

SERVER_URL="${SERVER_URL:-http://localhost:8080}"
WORKFLOW_FILE="${WORKFLOW_FILE:-examples/hello-world.yaml}"
REQUESTS="${REQUESTS:-1000}"
CONCURRENCY="${CONCURRENCY:-50}"

echo "=== API Performance Benchmark ==="
echo "Server: $SERVER_URL"
echo "Requests: $REQUESTS"
echo "Concurrency: $CONCURRENCY"
echo ""

# Test 1: POST /v1/workflows (提交工作流)
echo "--- POST /v1/workflows ---"
hey -n $REQUESTS -c $CONCURRENCY \
    -m POST \
    -H "Content-Type: application/yaml" \
    -D "$WORKFLOW_FILE" \
    "$SERVER_URL/v1/workflows" \
    | tee results/submit_workflow.txt

# 提取关键指标
echo "P50:" $(grep "50%" results/submit_workflow.txt | awk '{print $2}')
echo "P99:" $(grep "99%" results/submit_workflow.txt | awk '{print $2}')
echo ""

# Test 2: GET /v1/workflows/{id} (查询状态)
echo "--- GET /v1/workflows/{id} ---"
# 先提交一个工作流获取 ID
WORKFLOW_ID=$(curl -s -X POST "$SERVER_URL/v1/workflows" \
    -H "Content-Type: application/yaml" \
    --data-binary "@$WORKFLOW_FILE" | jq -r '.workflow_id')

hey -n $REQUESTS -c $CONCURRENCY \
    "$SERVER_URL/v1/workflows/$WORKFLOW_ID" \
    | tee results/get_status.txt

echo "P50:" $(grep "50%" results/get_status.txt | awk '{print $2}')
echo "P99:" $(grep "99%" results/get_status.txt | awk '{print $2}')
echo ""

# Test 3: POST /v1/validate (验证 YAML)
echo "--- POST /v1/validate ---"
hey -n $REQUESTS -c $CONCURRENCY \
    -m POST \
    -H "Content-Type: application/yaml" \
    -D "$WORKFLOW_FILE" \
    "$SERVER_URL/v1/validate" \
    | tee results/validate.txt

echo "P50:" $(grep "50%" results/validate.txt | awk '{print $2}')
echo "P99:" $(grep "99%" results/validate.txt | awk '{print $2}')
echo ""

echo "=== Benchmark Complete ==="
echo "Results saved to results/"

# 清理测试数据 (可选)
if [ "${CLEANUP:-true}" = "true" ]; then
    echo "Cleaning up test workflows..."
    # 清理提交的测试工作流
    # curl -X DELETE "$SERVER_URL/v1/workflows/$WORKFLOW_ID" || true
fi
```

**验证脚本 (使用 vegeta):**
```bash
#!/bin/bash
# test/performance/vegeta_benchmark.sh

set -e

SERVER_URL="${SERVER_URL:-http://localhost:8080}"
DURATION="${DURATION:-30s}"
RATE="${RATE:-100}"

echo "POST $SERVER_URL/v1/validate" | \
    vegeta attack -duration=$DURATION -rate=$RATE \
    -header "Content-Type: application/yaml" \
    -body examples/hello-world.yaml \
    | tee results/vegeta_validate.bin \
    | vegeta report -type=text

# 生成 JSON 报告
vegeta report -type=json results/vegeta_validate.bin > results/vegeta_validate.json

# 提取关键指标并验证
P50=$(jq '.latencies.p50 / 1000000' results/vegeta_validate.json)
P99=$(jq '.latencies.p99 / 1000000' results/vegeta_validate.json)

echo "P50: ${P50}ms (target < 200ms)"
echo "P99: ${P99}ms (target < 500ms)"

# 验证阈值
if (( $(echo "$P50 > 200" | bc -l) )); then
    echo "❌ P50 exceeded threshold"
    exit 1
fi

if (( $(echo "$P99 > 500" | bc -l) )); then
    echo "❌ P99 exceeded threshold"
    exit 1
fi

echo "✅ API latency targets met"
```

### AC2: YAML 解析性能 - 1000 行 < 100ms

**Given** YAML 解析器实现  
**When** 解析 1000 行 YAML 文件  
**Then** 解析时间 < 100ms  
**And** 内存分配 < 10MB

**Implementation Notes:**

**扩展现有基准测试:**
```go
// pkg/dsl/validator_bench_test.go
func BenchmarkValidate1000LineWorkflow(b *testing.B) {
    // 1000 行 YAML (~20 jobs, 100 steps)
    content, err := os.ReadFile("../../testdata/benchmark/xlarge.yaml")
    if err != nil {
        b.Fatal(err)
    }
    
    if len(strings.Split(string(content), "\n")) < 1000 {
        b.Fatalf("Test file should have ~1000 lines, got %d", 
            len(strings.Split(string(content), "\n")))
    }
    
    validator, err := dsl.NewValidator(zap.NewNop())
    if err != nil {
        b.Fatal(err)
    }
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        _, err := validator.ValidateYAML(content)
        if err != nil {
            b.Fatal(err)
        }
    }
    
    // 验证性能目标
    elapsed := b.Elapsed()
    if elapsed/time.Duration(b.N) > 100*time.Millisecond {
        b.Fatalf("Average parse time %v exceeds 100ms target", 
            elapsed/time.Duration(b.N))
    }
}

// 内存基准测试
func BenchmarkValidateMemory(b *testing.B) {
    content, _ := os.ReadFile("../../testdata/benchmark/large.yaml")
    validator, _ := dsl.NewValidator(zap.NewNop())
    
    b.ReportAllocs()
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _, _ = validator.ValidateYAML(content)
    }
    
    // 验证内存分配 < 10MB
    bytesPerOp := b.AllocedBytesPerOp()
    mbPerOp := float64(bytesPerOp) / 1024 / 1024
    
    if mbPerOp > 10.0 {
        b.Errorf("Memory allocation %.2fMB exceeds 10MB target", mbPerOp)
    }
    
    b.Logf("Memory per operation: %.2f MB", mbPerOp)
}
```

**创建 1000 行测试文件:**
```bash
#!/bin/bash
# scripts/generate_benchmark_yaml.sh

cat > testdata/benchmark/xlarge.yaml << 'EOF'
name: benchmark-xlarge
on:
  workflow_dispatch:

vars:
  server_count: 20
  
jobs:
EOF

# 生成 20 个 jobs,每个 5 个 steps
for job in {1..20}; do
  cat >> testdata/benchmark/xlarge.yaml << EOF
  job-${job}:
    runs-on: linux-amd64
    steps:
      - name: step-${job}-1
        uses: exec/shell
        with:
          command: echo "Job ${job} Step 1"
      - name: step-${job}-2
        uses: exec/shell
        with:
          command: echo "Job ${job} Step 2"
      - name: step-${job}-3
        uses: http/request
        with:
          url: https://api.example.com/job/${job}
          method: GET
      - name: step-${job}-4
        uses: flow/sleep
        with:
          duration: 1s
      - name: step-${job}-5
        uses: exec/shell
        with:
          command: echo "Job ${job} Complete"
EOF
done

echo "Generated testdata/benchmark/xlarge.yaml"
wc -l testdata/benchmark/xlarge.yaml
```

### AC3: 工作流提交吞吐量 > 100/秒

**Given** Waterflow Server 和 Agent 运行  
**When** 并发提交工作流  
**Then** 吞吐量 > 100 个工作流/秒  
**And** 错误率 < 1%

**Implementation Notes:**

**吞吐量测试脚本:**
```bash
#!/bin/bash
# test/performance/throughput_test.sh

set -e

SERVER_URL="${SERVER_URL:-http://localhost:8080}"
DURATION="${DURATION:-60}"
RATE="${RATE:-150}"

echo "=== Throughput Test ==="
echo "Target: > 100 workflows/sec"
echo "Test Rate: $RATE workflows/sec"
echo "Duration: ${DURATION}s"
echo ""

# 使用 vegeta 进行吞吐量测试
echo "POST $SERVER_URL/v1/workflows" | \
    vegeta attack \
    -duration=${DURATION}s \
    -rate=$RATE \
    -header "Content-Type: application/yaml" \
    -body examples/hello-world.yaml \
    | tee results/throughput.bin \
    | vegeta report -type=text

# 生成详细报告
vegeta report -type=json results/throughput.bin > results/throughput.json

# 提取指标
SUCCESS_RATE=$(jq '.success * 100' results/throughput.json)
ACTUAL_RATE=$(jq '.rate' results/throughput.json)
ERROR_RATE=$(jq '(1 - .success) * 100' results/throughput.json)

echo ""
echo "=== Results ==="
echo "Actual Rate: $ACTUAL_RATE req/sec"
echo "Success Rate: $SUCCESS_RATE%"
echo "Error Rate: $ERROR_RATE%"

# 验证目标
if (( $(echo "$ACTUAL_RATE < 100" | bc -l) )); then
    echo "❌ Throughput below target (100/sec)"
    exit 1
fi

if (( $(echo "$ERROR_RATE > 1" | bc -l) )); then
    echo "❌ Error rate above threshold (1%)"
    exit 1
fi

echo "✅ Throughput targets met"

# 清理测试数据
if [ "${CLEANUP:-true}" = "true" ]; then
    echo "Cleaning up test workflows..."
    # 注意: 生产环境应使用专门的测试 namespace
    # 避免影响生产数据
fi
```

**Go 并发基准测试:**
```go
// test/performance/throughput_test.go
package performance

import (
    "context"
    "sync"
    "testing"
    "time"
    
    "github.com/Websoft9/waterflow/pkg/sdk"
)

func TestWorkflowThroughput(t *testing.T) {
    client, err := sdk.NewClient("http://localhost:8080")
    if err != nil {
        t.Fatal(err)
    }
    
    workflow := []byte(`
name: throughput-test
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - uses: exec/shell
        with:
          command: echo "test"
`)
    
    duration := 60 * time.Second
    targetRate := 100 // workflows/sec
    
    ctx, cancel := context.WithTimeout(context.Background(), duration)
    defer cancel()
    
    var (
        submitted int
        errors    int
        mu        sync.Mutex
    )
    
    // 并发提交
    concurrency := 50
    var wg sync.WaitGroup
    
    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            
            for {
                select {
                case <-ctx.Done():
                    return
                default:
                    _, err := client.SubmitWorkflow(context.Background(), workflow, nil)
                    
                    mu.Lock()
                    if err != nil {
                        errors++
                    } else {
                        submitted++
                    }
                    mu.Unlock()
                }
            }
        }()
    }
    
    wg.Wait()
    
    actualRate := float64(submitted) / duration.Seconds()
    errorRate := float64(errors) / float64(submitted+errors) * 100
    
    t.Logf("Submitted: %d workflows", submitted)
    t.Logf("Errors: %d", errors)
    t.Logf("Actual Rate: %.2f workflows/sec", actualRate)
    t.Logf("Error Rate: %.2f%%", errorRate)
    
    if actualRate < float64(targetRate) {
        t.Errorf("Throughput %.2f/sec below target %d/sec", actualRate, targetRate)
    }
    
    if errorRate > 1.0 {
        t.Errorf("Error rate %.2f%% above threshold 1%%", errorRate)
    }
}
```

### AC4: Agent 空闲内存 < 50MB

**Given** Agent 运行中  
**When** 监控内存使用  
**Then** 空闲状态内存 (RSS) < 50MB  
**And** Heap 使用 < 30MB

**Implementation Notes:**

**跨平台内存监控 (Go 实现 - 推荐):**
```go
// test/performance/agent_memory_test.go
package performance

import (
    "runtime"
    "testing"
    "time"
    
    "github.com/shirou/gopsutil/v3/process"
)

func TestAgentMemoryUsage(t *testing.T) {
    // 启动 Agent (在测试中)
    // agent := startAgent(t)
    // defer agent.Stop()
    
    pid := os.Getpid()
    proc, err := process.NewProcess(int32(pid))
    if err != nil {
        t.Fatal(err)
    }
    
    // 等待初始化
    time.Sleep(10 * time.Second)
    
    // 监控 30 秒
    for i := 0; i < 30; i++ {
        memInfo, err := proc.MemoryInfo()
        if err != nil {
            t.Fatal(err)
        }
        
        rssMB := float64(memInfo.RSS) / 1024 / 1024
        t.Logf("[%d] RSS: %.2f MB", i+1, rssMB)
        
        time.Sleep(1 * time.Second)
    }
    
    // 最终检查
    memInfo, _ := proc.MemoryInfo()
    finalRSS := float64(memInfo.RSS) / 1024 / 1024
    
    if finalRSS > 50.0 {
        t.Errorf("Memory usage %.2fMB exceeds 50MB threshold", finalRSS)
    }
    
    t.Logf("Final Memory: %.2f MB", finalRSS)
}
```

**内存监控脚本 (Linux only):**
```bash
#!/bin/bash
# test/performance/agent_memory_test.sh
# Note: This script is Linux-specific. For cross-platform, use Go test above.

set -e

echo "=== Agent Memory Test (Linux) ==="

# 启动 Agent
./bin/agent --config config.agent.example.yaml &
AGENT_PID=$!

echo "Agent PID: $AGENT_PID"
echo "Waiting for initialization..."
sleep 10

# 监控内存 (30 秒)
echo "Monitoring memory for 30 seconds..."
for i in {1..30}; do
    # Linux ps 命令获取 RSS (KB)
    RSS=$(ps -p $AGENT_PID -o rss= | tr -d ' ')
    RSS_MB=$(echo "scale=2; $RSS / 1024" | bc)
    
    echo "[$i] RSS: ${RSS_MB}MB"
    
    sleep 1
done

# 获取最终内存
FINAL_RSS=$(ps -p $AGENT_PID -o rss= | tr -d ' ')
FINAL_RSS_MB=$(echo "scale=2; $FINAL_RSS / 1024" | bc)

echo ""
echo "Final Memory: ${FINAL_RSS_MB}MB"

# 停止 Agent
kill $AGENT_PID

# 验证阈值
if (( $(echo "$FINAL_RSS_MB > 50" | bc -l) )); then
    echo "❌ Memory usage ${FINAL_RSS_MB}MB exceeds 50MB threshold"
    exit 1
fi

echo "✅ Memory target met"
```

**Go 内存基准测试:**
```go
// internal/agent/worker_bench_test.go
func BenchmarkAgentIdleMemory(b *testing.B) {
    cfg := &config.Config{
        Agent: config.AgentConfig{
            TaskQueues: []string{"test-queue"},
            PluginDir:  "../../plugins",
        },
        Temporal: config.TemporalConfig{
            Host:      "localhost:7233",
            Namespace: "default",
        },
    }
    
    logger, _ := zap.NewProduction()
    
    // 创建 Worker 但不启动 (模拟空闲)
    worker, err := agent.NewWorker(cfg, logger)
    if err != nil {
        b.Fatal(err)
    }
    defer worker.Stop()
    
    // 强制 GC
    runtime.GC()
    
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    heapMB := m.Alloc / 1024 / 1024
    
    b.Logf("Heap Memory: %d MB", heapMB)
    
    if heapMB > 30 {
        b.Errorf("Heap memory %dMB exceeds 30MB threshold", heapMB)
    }
}
```

### AC5: 支持 100+ 并发 Agent 连接

**Given** Temporal Server 运行  
**When** 100 个 Agent 同时连接  
**Then** 所有 Agent 成功连接  
**And** Server 稳定运行  
**And** 任务正常分发

**Implementation Notes:**

**并发 Agent 测试:**
```bash
#!/bin/bash
# test/performance/concurrent_agents_test.sh

set -e

AGENT_COUNT="${AGENT_COUNT:-100}"
TEMPORAL_HOST="${TEMPORAL_HOST:-localhost:7233}"

echo "=== Concurrent Agents Test ==="
echo "Target: $AGENT_COUNT agents"
echo ""

# 创建临时目录
TMP_DIR=$(mktemp -d)
echo "Temp dir: $TMP_DIR"

# 启动 Agents
PIDS=()
for i in $(seq 1 $AGENT_COUNT); do
    TASK_QUEUE="test-queue-$((i % 10))"
    
    ./bin/agent \
        --task-queues "$TASK_QUEUE" \
        --log-level error \
        > "$TMP_DIR/agent-$i.log" 2>&1 &
    
    PIDS+=($!)
    
    # 每 10 个 agent 暂停一下
    if (( i % 10 == 0 )); then
        echo "Started $i agents..."
        sleep 1
    fi
done

echo "All $AGENT_COUNT agents started"
echo "Waiting 30 seconds for stabilization..."
sleep 30

# 检查进程状态
RUNNING=0
for pid in "${PIDS[@]}"; do
    if kill -0 $pid 2>/dev/null; then
        ((RUNNING++))
    fi
done

echo ""
echo "Agents running: $RUNNING / $AGENT_COUNT"

# 清理
echo "Stopping agents..."
for pid in "${PIDS[@]}"; do
    kill $pid 2>/dev/null || true
done

# 等待进程完全退出
sleep 2

# 清理临时文件
rm -rf "$TMP_DIR"

# 清理测试任务队列 (如果需要)
if [ "${CLEANUP_QUEUES:-false}" = "true" ]; then
    echo "Cleaning up test task queues..."
    # tctl task-queue delete --task-queue test-queue-* || true
fi

# 验证
if [ $RUNNING -lt $AGENT_COUNT ]; then
    echo "❌ Only $RUNNING agents running, expected $AGENT_COUNT"
    exit 1
fi

echo "✅ Concurrent agents target met"
```

### AC6: Event History 查询延迟 < 100ms

**Given** Temporal Event History 存储  
**When** 查询工作流历史  
**Then** 查询延迟 < 100ms  
**And** 验证 Event Sourcing 不影响性能

**Implementation Notes:**

**Event History 基准测试:**
```go
// pkg/temporal/history_bench_test.go
package temporal

import (
    "context"
    "testing"
    "time"
    
    "go.temporal.io/sdk/client"
)

func BenchmarkGetWorkflowHistory(b *testing.B) {
    // 连接到 Temporal
    c, err := client.Dial(client.Options{
        HostPort:  "localhost:7233",
        Namespace: "default",
    })
    if err != nil {
        b.Fatal(err)
    }
    defer c.Close()
    
    // 先执行一个工作流
    ctx := context.Background()
    workflowID := "bench-history-test"
    
    // 假设已有工作流执行完成
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        start := time.Now()
        
        iter := c.GetWorkflowHistory(ctx, workflowID, "", false, 0)
        
        // 读取所有事件
        eventCount := 0
        for iter.HasNext() {
            _, err := iter.Next()
            if err != nil {
                b.Fatal(err)
            }
            eventCount++
        }
        
        elapsed := time.Since(start)
        
        if elapsed > 100*time.Millisecond {
            b.Logf("Query took %v (target < 100ms), events: %d", 
                elapsed, eventCount)
        }
    }
}

func BenchmarkDescribeWorkflowExecution(b *testing.B) {
    c, err := client.Dial(client.Options{
        HostPort:  "localhost:7233",
        Namespace: "default",
    })
    if err != nil {
        b.Fatal(err)
    }
    defer c.Close()
    
    ctx := context.Background()
    workflowID := "bench-describe-test"
    runID := ""
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        start := time.Now()
        
        _, err := c.DescribeWorkflowExecution(ctx, workflowID, runID)
        if err != nil {
            b.Fatal(err)
        }
        
        elapsed := time.Since(start)
        
        if elapsed > 100*time.Millisecond {
            b.Errorf("DescribeWorkflowExecution took %v (target < 100ms)", 
                elapsed)
        }
    }
}
```

### AC7: 基准测试可重复运行并建立性能基线

**Given** 性能测试框架实现  
**When** 运行基准测试  
**Then** 生成 JSON 格式的基线数据  
**And** 存储到 `test/performance/baseline/` 目录  
**And** 支持与历史基线对比  
**And** 自动检测性能回归 (±10% 阈值)

**Implementation Notes:**

**基线存储格式:**
```json
{
  "version": "1.0.0",
  "timestamp": "2026-01-07T10:30:00Z",
  "git_commit": "abc123",
  "environment": {
    "os": "linux",
    "arch": "amd64",
    "go_version": "1.21.5",
    "cpu": "Intel Core i7-9700K",
    "memory": "16GB"
  },
  "benchmarks": {
    "BenchmarkValidate1000LineWorkflow": {
      "ns_per_op": 85000000,
      "mb_per_op": 8.5,
      "allocs_per_op": 15000
    },
    "BenchmarkAPISubmitWorkflow": {
      "latency_p50_ms": 180,
      "latency_p95_ms": 350,
      "latency_p99_ms": 450,
      "throughput_per_sec": 120
    }
  }
}
```

**基线管理脚本:**
```bash
#!/bin/bash
# scripts/benchmark_baseline.sh

set -e

BASELINE_DIR="test/performance/baseline"
RESULTS_DIR="test/performance/results"

mkdir -p "$BASELINE_DIR" "$RESULTS_DIR"

# 运行所有基准测试
echo "=== Running Benchmarks ==="

# Go benchmarks
go test -bench=. -benchmem -run=^$ ./pkg/dsl/... \
    | tee "$RESULTS_DIR/go_bench.txt"

# 解析 Go benchmark 结果
python3 scripts/parse_go_bench.py \
    "$RESULTS_DIR/go_bench.txt" \
    > "$RESULTS_DIR/go_bench.json"

# API benchmarks
./test/performance/api_benchmark.sh

# 合并结果
python3 scripts/merge_benchmark_results.py \
    --go "$RESULTS_DIR/go_bench.json" \
    --api "$RESULTS_DIR/vegeta_validate.json" \
    --output "$RESULTS_DIR/combined.json"

# 保存为基线 (如果没有)
GIT_COMMIT=$(git rev-parse --short HEAD)
GIT_TAG=$(git describe --tags --exact-match 2>/dev/null || echo "")
BASELINE_FILE="$BASELINE_DIR/baseline-${GIT_COMMIT}.json"

if [ ! -f "$BASELINE_FILE" ]; then
    cp "$RESULTS_DIR/combined.json" "$BASELINE_FILE"
    echo "Baseline saved: $BASELINE_FILE"
    
    # 如果有 Git tag,创建符号链接
    if [ -n "$GIT_TAG" ]; then
        ln -sf "baseline-${GIT_COMMIT}.json" "$BASELINE_DIR/baseline-${GIT_TAG}.json"
        echo "Tagged baseline: $GIT_TAG"
    fi
fi

# 选择基线对比 (优先级: 环境变量 > Git tag > main 分支 > 最新)
BASELINE_TO_COMPARE="${BASELINE_COMPARE:-}"

if [ -z "$BASELINE_TO_COMPARE" ]; then
    # 尝试使用 main 分支的基线
    MAIN_COMMIT=$(git rev-parse --short origin/main 2>/dev/null || echo "")
    if [ -n "$MAIN_COMMIT" ] && [ -f "$BASELINE_DIR/baseline-${MAIN_COMMIT}.json" ]; then
        BASELINE_TO_COMPARE="$BASELINE_DIR/baseline-${MAIN_COMMIT}.json"
        echo "Comparing with main branch baseline: $MAIN_COMMIT"
    else
        # 使用最新的基线文件 (按修改时间)
        BASELINE_TO_COMPARE=$(ls -t "$BASELINE_DIR"/baseline-*.json 2>/dev/null | grep -v "${GIT_COMMIT}" | head -1)
        if [ -n "$BASELINE_TO_COMPARE" ]; then
            echo "Comparing with latest baseline: $(basename $BASELINE_TO_COMPARE)"
        fi
    fi
fi

if [ -n "$BASELINE_TO_COMPARE" ] && [ -f "$BASELINE_TO_COMPARE" ]; then
    echo ""
    echo "=== Comparing with baseline ==="
    python3 scripts/compare_benchmarks.py \
        --baseline "$BASELINE_TO_COMPARE" \
        --current "$RESULTS_DIR/combined.json" \
        --threshold 10
else
    echo "No baseline found for comparison"
fi
```

**性能回归检测:**
```python
#!/usr/bin/env python3
# scripts/compare_benchmarks.py

import json
import sys
import argparse

def compare_benchmarks(baseline_file, current_file, threshold=10):
    with open(baseline_file) as f:
        baseline = json.load(f)
    
    with open(current_file) as f:
        current = json.load(f)
    
    regressions = []
    improvements = []
    
    for name, current_data in current['benchmarks'].items():
        if name not in baseline['benchmarks']:
            continue
        
        baseline_data = baseline['benchmarks'][name]
        
        # 比较关键指标
        for metric in ['ns_per_op', 'latency_p99_ms']:
            if metric not in current_data or metric not in baseline_data:
                continue
            
            baseline_val = baseline_data[metric]
            current_val = current_data[metric]
            
            change_pct = ((current_val - baseline_val) / baseline_val) * 100
            
            if abs(change_pct) > threshold:
                item = {
                    'benchmark': name,
                    'metric': metric,
                    'baseline': baseline_val,
                    'current': current_val,
                    'change_pct': change_pct
                }
                
                if change_pct > 0:
                    regressions.append(item)
                else:
                    improvements.append(item)
    
    # 打印结果
    if regressions:
        print("❌ Performance Regressions Detected:")
        for item in regressions:
            print(f"  {item['benchmark']}.{item['metric']}: "
                  f"{item['baseline']} → {item['current']} "
                  f"({item['change_pct']:+.1f}%)")
        print()
    
    if improvements:
        print("✅ Performance Improvements:")
        for item in improvements:
            print(f"  {item['benchmark']}.{item['metric']}: "
                  f"{item['baseline']} → {item['current']} "
                  f"({item['change_pct']:+.1f}%)")
        print()
    
    if not regressions and not improvements:
        print("✅ No significant performance changes")
    
    # 返回非零退出码如果有回归
    return 1 if regressions else 0

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--baseline', required=True)
    parser.add_argument('--current', required=True)
    parser.add_argument('--threshold', type=float, default=10.0)
    
    args = parser.parse_args()
    
    sys.exit(compare_benchmarks(args.baseline, args.current, args.threshold))
```

### AC8: 验证 Server 无状态不影响性能

**Given** Event Sourcing 架构 (Server 无状态)  
**When** Server 重启后继续处理请求  
**Then** 性能指标无明显变化  
**And** 工作流状态完全恢复  
**And** 无需预热或缓存重建

**Implementation Notes:**

**Server 重启性能测试:**
```bash
#!/bin/bash
# test/performance/server_restart_test.sh

set -e

echo "=== Server Restart Performance Test ==="

# 1. 启动 Server
echo "Starting server..."
./bin/server &
SERVER_PID=$!
sleep 5

# 2. 提交一些工作流
echo "Submitting workflows before restart..."
for i in {1..10}; do
    curl -s -X POST http://localhost:8080/v1/workflows \
        -H "Content-Type: application/yaml" \
        --data-binary "@examples/hello-world.yaml"
done

# 3. 测试性能 (before restart)
echo "Measuring performance before restart..."
vegeta attack -duration=10s -rate=100 \
    -header "Content-Type: application/yaml" \
    -body examples/hello-world.yaml \
    http://localhost:8080/v1/validate \
    | vegeta report -type=json > results/before_restart.json

BEFORE_P99=$(jq '.latencies.p99 / 1000000' results/before_restart.json)
echo "Before restart P99: ${BEFORE_P99}ms"

# 4. 重启 Server
echo "Restarting server..."
kill $SERVER_PID
sleep 2
./bin/server &
SERVER_PID=$!
sleep 5

# 5. 测试性能 (after restart)
echo "Measuring performance after restart..."
vegeta attack -duration=10s -rate=100 \
    -header "Content-Type: application/yaml" \
    -body examples/hello-world.yaml \
    http://localhost:8080/v1/validate \
    | vegeta report -type=json > results/after_restart.json

AFTER_P99=$(jq '.latencies.p99 / 1000000' results/after_restart.json)
echo "After restart P99: ${AFTER_P99}ms"

# 6. 对比
CHANGE=$(echo "scale=2; ($AFTER_P99 - $BEFORE_P99) / $BEFORE_P99 * 100" | bc)
echo "Performance change: ${CHANGE}%"

# 7. 验证工作流状态恢复
echo "Verifying workflow state recovery..."
# (检查之前提交的工作流状态)

# 清理
kill $SERVER_PID

if (( $(echo "$CHANGE > 10" | bc -l) )); then
    echo "❌ Performance degradation after restart: ${CHANGE}%"
    exit 1
fi

echo "✅ Server stateless performance verified"
```

### AC9: Server 启动时间 < 5 秒 (PRD NFR2)

**Given** Server 二进制文件编译完成  
**When** 启动 Server  
**Then** 从启动到 ready 状态 < 5 秒  
**And** HTTP 健康检查端点可访问  
**And** Temporal 连接建立

**Implementation Notes:**

**启动时间测试脚本:**
```bash
#!/bin/bash
# test/performance/server_startup_test.sh

set -e

echo "=== Server Startup Time Test ==="

# 记录开始时间
START_TIME=$(date +%s.%N)

# 启动 Server
echo "Starting server..."
./bin/server > /dev/null 2>&1 &
SERVER_PID=$!

# 等待健康检查端点可用
MAX_WAIT=10
ELAPSED=0

while [ $ELAPSED -lt $MAX_WAIT ]; do
    if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
        END_TIME=$(date +%s.%N)
        STARTUP_TIME=$(echo "$END_TIME - $START_TIME" | bc)
        
        echo "Server started in ${STARTUP_TIME}s"
        
        # 验证阈值
        if (( $(echo "$STARTUP_TIME > 5.0" | bc -l) )); then
            echo "❌ Startup time ${STARTUP_TIME}s exceeds 5s threshold"
            kill $SERVER_PID
            exit 1
        fi
        
        echo "✅ Startup time target met"
        kill $SERVER_PID
        exit 0
    fi
    
    sleep 0.1
    ELAPSED=$(echo "$ELAPSED + 0.1" | bc)
done

echo "❌ Server failed to start within ${MAX_WAIT}s"
kill $SERVER_PID 2>/dev/null || true
exit 1
```

**Go 启动时间测试:**
```go
// cmd/server/main_test.go
package main

import (
    "net/http"
    "testing"
    "time"
)

func TestServerStartupTime(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping startup time test in short mode")
    }
    
    start := time.Now()
    
    // 启动 Server (在 goroutine 中)
    go func() {
        main()
    }()
    
    // 等待健康检查可用
    maxWait := 10 * time.Second
    checkInterval := 100 * time.Millisecond
    
    for elapsed := time.Duration(0); elapsed < maxWait; elapsed += checkInterval {
        resp, err := http.Get("http://localhost:8080/health")
        if err == nil && resp.StatusCode == 200 {
            startupTime := time.Since(start)
            
            t.Logf("Server started in %v", startupTime)
            
            if startupTime > 5*time.Second {
                t.Errorf("Startup time %v exceeds 5s threshold", startupTime)
            }
            
            return
        }
        
        time.Sleep(checkInterval)
    }
    
    t.Fatal("Server failed to start within timeout")
}
```

## Tasks / Subtasks

### Task 1: 扩展 Go 基准测试 (AC2)

- [x] 1.1 创建 1000 行 YAML 测试文件
  - 生成脚本 scripts/generate_benchmark_yaml.sh
  - testdata/benchmark/xlarge.yaml (1003行)

- [x] 1.2 实现 BenchmarkValidate1000LineWorkflow
  - 验证解析时间 < 100ms (实际: 16ms ✅)
  - 验证内存分配 < 10MB (实际: 2.9MB ✅)

- [x] 1.3 实现内存基准测试
  - BenchmarkValidateMemory
  - 监控 heap 使用

- [x] 1.4 扩展现有基准测试
  - 添加性能断言
  - 统一报告格式

### Task 2: 实现 API 负载测试 (AC1, AC3)

- [x] 2.1 创建 API 基准测试脚本
  - test/performance/api_benchmark.sh (使用 curl)
  - test/performance/install_tools.sh (安装 hey/vegeta)
  
- [x] 2.2 创建吞吐量测试
  - test/performance/throughput_test.go
  - TestWorkflowThroughput (验证 > 100 workflows/sec)
  - TestAPILatency (验证 P99 < 500ms)

- [x] 2.3 实现 Go 性能测试
  - BenchmarkWorkflowSubmit
  - BenchmarkConcurrentSubmit

- [x] 2.4 创建 README 文档
  - test/performance/README.md
  - 使用指南和故障排查

### Task 3: 实现资源监控测试 (AC4, AC5) - **Post-MVP**

- [ ] 3.1 创建 Agent 内存测试
  - test/performance/agent_memory_test.sh
  - 监控 RSS < 50MB

- [ ] 3.2 实现并发 Agent 测试
  - test/performance/concurrent_agents_test.sh
  - 测试 100+ agents 连接

- [ ] 3.3 创建 Go 内存基准
  - internal/agent/worker_bench_test.go
  - 验证 Heap < 30MB

### Task 4: 实现 Temporal 性能测试 (AC6) - **Post-MVP**

- [ ] 4.1 创建 Event History 基准测试
  - pkg/temporal/history_bench_test.go
  - BenchmarkGetWorkflowHistory
  - BenchmarkDescribeWorkflowExecution

- [ ] 4.2 验证查询延迟 < 100ms
  - 读取事件历史
  - 描述工作流执行

### Task 5: 建立性能基线系统 (AC7) - **Post-MVP**

- [ ] 5.1 定义基线数据格式
  - JSON schema
  - 环境信息
  - 基准结果

- [ ] 5.2 实现基线管理脚本
  - scripts/benchmark_baseline.sh
  - 运行所有基准测试
  - 保存结果

- [ ] 5.3 实现结果解析脚本
  - scripts/parse_go_bench.py
  - 解析 Go benchmark 输出
  - 生成 JSON

- [ ] 5.4 实现基线对比脚本
  - scripts/compare_benchmarks.py
  - 检测性能回归
  - 阈值验证 (±10%)

### Task 6: 验证无状态架构性能 (AC8) - **Post-MVP**

- [ ] 6.1 创建 Server 重启测试
  - test/performance/server_restart_test.sh
  - 重启前后性能对比
  - 状态恢复验证

- [ ] 6.2 创建端到端性能测试
  - test/performance/e2e_test.sh
  - 完整工作流执行
  - 多步骤性能

### Task 7: 集成到 CI/CD - **Post-MVP**

- [ ] 7.1 添加 Makefile 目标
  - make benchmark
  - make benchmark-baseline
  - make benchmark-compare

- [ ] 7.2 创建 GitHub Actions workflow
  - .github/workflows/benchmark.yml
  - 每次 PR 运行基准测试
  - 对比 main 分支基线

- [ ] 7.3 性能回归检测
  - 自动评论 PR
  - 标记性能退化

### Task 8: 文档和报告 - **Post-MVP**

- [ ] 8.1 编写性能测试指南
  - docs/guides/performance-testing.md
  - 运行方式
  - 解读结果
  - 跨平台注意事项

- [ ] 8.2 创建性能报告模板
  - test/performance/REPORT_TEMPLATE.md
  - 关键指标
  - 对比分析

- [ ] 8.3 更新 README
  - 性能测试章节
  - 基准数据

- [ ] 8.4 创建性能仪表板
  - test/performance/dashboard.html
  - 可视化趋势

### Task 9: Server 启动时间测试 (AC9) - **Post-MVP**

- [ ] 9.1 创建启动时间测试脚本
  - test/performance/server_startup_test.sh
  - 测量启动到 ready 的时间
  - 验证 < 5 秒阈值

- [ ] 9.2 实现 Go 启动时间测试
  - cmd/server/main_test.go
  - TestServerStartupTime
  - 健康检查端点验证

- [ ] 9.3 添加到 CI/CD 流程
  - 每次构建运行启动时间测试
  - 记录启动时间趋势

## Dev Agent Record

### Implementation Plan

**设计决策:**
1. ✅ 使用 Go testing.B 框架进行组件级基准测试
2. ✅ curl 脚本 + Go 测试实现 API 负载测试(无需外部依赖)
3. ⚠️  hey/vegeta 作为可选工具提供更精确的百分位测量
4. ⚠️  性能基线系统、Agent内存测试、Temporal测试留待Post-MVP

**实现顺序:**
1. ✅ Go DSL基准测试扩展 (AC2完成)
2. ✅ API负载测试脚本和Go测试 (AC1/AC3部分完成)
3. ⚠️  资源监控、基线系统、CI/CD集成 - Post-MVP

### Debug Log

**2026-01-08 实现记录:**

- ✅ 创建 `scripts/generate_benchmark_yaml.sh` - 生成1003行测试YAML
- ✅ 扩展 `pkg/dsl/validator_bench_test.go`:
  - BenchmarkValidate1000LineWorkflow: 16ms < 100ms目标 ✅
  - BenchmarkValidateMemory: 2.9MB < 10MB目标 ✅
  - BenchmarkParse1000Lines: 纯解析性能基准
- ✅ 创建 `test/performance/` 目录结构
- ✅ 创建 `test/performance/api_benchmark.sh` - curl基础API测试
- ✅ 创建 `test/performance/install_tools.sh` - 安装hey/vegeta
- ✅ 创建 `test/performance/throughput_test.go`:
  - TestWorkflowThroughput: 吞吐量测试 (>100/sec)
  - TestAPILatency: API延迟测试 (P99<500ms)
  - BenchmarkWorkflowSubmit: 单次提交基准
  - BenchmarkConcurrentSubmit: 并发提交基准
- ✅ 创建 `test/performance/README.md` - 使用文档

**测试结果:**
- DSL解析 (1003行): 16ms (超过目标6.25倍) ✅
- 内存使用: 2.9MB (超过目标3.4倍) ✅  
- Go测试代码编译通过 ✅

**技术债务:**
- ⚠️  Task 3-8 留待Post-MVP完成:
  - Agent内存监控测试 (需要实际Agent运行)
  - 并发Agent连接测试 (需要Temporal环境)
  - Temporal Event History性能测试
  - 性能基线建立和对比系统
  - Server无状态架构验证
  - Server启动时间测试
  - CI/CD集成
- ⚠️  实际API性能测试需要运行Server+Temporal环境

### Completion Notes

✅ **Story 7.3 MVP部分完成!**

**核心成就 (MVP):**
- ✅ 扩展了Go基准测试框架,验证DSL解析性能远超目标
- ✅ 创建了API负载测试脚本和Go测试框架
- ✅ 实现了吞吐量和延迟测试框架
- ✅ 编写了详细的使用文档

**关键性能验证:**
1. **DSL解析** - 1003行YAML: 16ms < 100ms目标 (✅ 6.25x faster)
2. **内存使用** - 2.9MB < 10MB目标 (✅ 3.4x better)
3. **测试框架** - API/吞吐量/延迟测试已就绪

**AC完成度:**
- ✅ AC2完全达成 - DSL解析性能验证 (100%)
- 🟡 AC1框架就绪 - API延迟测试(需Server环境验证) (50%)
- 🟡 AC3框架就绪 - 吞吐量测试(需Server环境验证) (50%)
- ❌ AC4-AC9 - 移至Post-MVP (0%)

**Post-MVP待办:**
- Task 3: Agent内存和并发测试 (需要实际Agent)
- Task 4: Temporal性能测试 (需要Temporal环境)
- Task 5: 性能基线系统实现
- Task 6: Server无状态架构验证
- Task 7: CI/CD集成
- Task 8: 完整文档和报告
- Task 9: Server启动时间测试

**下一步:**
- 在完整环境中验证API性能指标(AC1/AC3)
- 或继续 Story 7.4 - 压力测试和容错验证

## File List

**新增文件:**
- scripts/generate_benchmark_yaml.sh (YAML生成脚本)
- testdata/benchmark/xlarge.yaml (1003行测试文件)
- test/performance/api_benchmark.sh (API负载测试)
- test/performance/install_tools.sh (工具安装脚本)
- test/performance/throughput_test.go (吞吐量和延迟测试)
- test/performance/README.md (性能测试文档)

**修改文件:**
- pkg/dsl/validator_bench_test.go (新增3个基准测试函数)
- docs/sprint-artifacts/7-3-performance-benchmarking.md (标记任务完成)

## Change Log

**2026-01-08 - Story 7.3 MVP完成**
- ✅ 创建性能基准测试框架
- ✅ DSL解析性能验证 - 1003行YAML: 16ms (目标<100ms)
- ✅ 内存使用验证 - 2.9MB (目标<10MB)
- ✅ API负载测试脚本创建 (curl基础+Go框架)
- ✅ 吞吐量测试框架创建
- ✅ API延迟测试框架创建
- ✅ 性能测试文档完成
- ⚠️  Post-MVP: Agent/Temporal/基线系统/CI集成

## Status

Status: In Progress (MVP部分完成，需环境验证AC1/AC3)

### Architecture Alignment

**性能测试架构 (本 Story):**
- ✅ 多层次测试 - 组件/API/端到端
- ✅ 自动化基线 - 可重复和对比
- ✅ 回归检测 - ±10% 阈值
- ✅ CI/CD 集成 - 每次 PR 验证

**与 Event Sourcing 验证:**
- ✅ Event History 查询性能
- ✅ Server 无状态验证
- ✅ 状态恢复性能

**与系统各模块:**
- ✅ DSL 解析性能
- ✅ API 响应性能
- ✅ Agent 资源使用
- ✅ Temporal 集成性能

### Project Structure

```
test/
├── performance/
│   ├── api_benchmark.sh          # 🆕 API 负载测试
│   ├── throughput_test.sh        # 🆕 吞吐量测试
│   ├── agent_memory_test.sh      # 🆕 Agent 内存测试 (Linux)
│   ├── agent_memory_test.go      # 🆕 Agent 内存测试 (跨平台)
│   ├── concurrent_agents_test.sh # 🆕 并发 Agent 测试
│   ├── server_restart_test.sh    # 🆕 Server 重启测试
│   ├── server_startup_test.sh    # 🆕 Server 启动时间测试
│   ├── e2e_test.sh               # 🆕 端到端测试
│   ├── throughput_test.go        # 🆕 Go 吞吐量测试
│   ├── baseline/                 # 🆕 基线数据存储
│   │   └── baseline-*.json
│   ├── results/                  # 🆕 测试结果
│   │   ├── go_bench.txt
│   │   ├── vegeta_*.json
│   │   └── combined.json
│   └── REPORT_TEMPLATE.md        # 🆕 报告模板

scripts/
├── generate_benchmark_yaml.sh    # 🆕 生成测试 YAML
├── benchmark_baseline.sh         # 🆕 基线管理
├── parse_go_bench.py             # 🆕 解析 Go benchmark
├── merge_benchmark_results.py    # 🆕 合并结果
└── compare_benchmarks.py         # 🆕 对比分析

pkg/
├── dsl/
│   ├── validator_bench_test.go   # 🔧 扩展现有测试
│   └── ...
├── temporal/
│   └── history_bench_test.go     # 🆕 Temporal 性能测试

internal/agent/
└── worker_bench_test.go          # 🆕 Agent 内存测试

cmd/server/
└── main_test.go                  # 🆕 Server 启动时间测试

testdata/
└── benchmark/
    ├── small.yaml                # ✅ 已存在
    ├── medium.yaml               # ✅ 已存在
    ├── large.yaml                # ✅ 已存在
    └── xlarge.yaml               # 🆕 1000+ 行

Makefile                          # 🔧 添加 benchmark 目标

.github/workflows/
└── benchmark.yml                 # 🆕 CI/CD 集成

docs/guides/
└── performance-testing.md        # 🆕 性能测试指南
```

### 关键技术决策

**1. 负载测试工具选择**
- ✅ 选择: hey + vegeta
- 理由:
  - hey: 简单快速,适合快速验证
  - vegeta: 功能强大,支持 JSON 结果和分析
- 替代方案:
  - ❌ ab (Apache Bench) - 功能有限
  - ❌ wrk - 需要 Lua 脚本

**2. 基线存储格式**
- ✅ 选择: JSON
- 理由:
  - 易于解析和对比
  - 支持嵌套结构
  - 版本控制友好
- 结构:
  - 环境信息
  - Git commit
  - 基准结果

**3. 性能回归阈值**
- ✅ 选择: ±10%
- 理由:
  - 平衡敏感度和稳定性
  - 容忍环境差异
  - 捕获显著退化
- 可配置: 不同指标可以不同阈值

**4. CI/CD 集成策略**
- ✅ 选择: PR 触发 + 基线对比
- 理由:
  - 早期发现性能问题
  - 防止性能退化合并
  - 可视化趋势
- 实现:
  - GitHub Actions workflow
  - 自动评论 PR
  - 可选阻止合并

### 测试策略

**测试分层:**

**Level 1: 组件级 (Go Benchmark)**
- DSL 解析
- 表达式求值
- Matrix 扩展
- 目标: 微秒级延迟,零分配

**Level 2: API 级 (HTTP Load Testing)**
- REST API 端点
- 并发请求
- 吞吐量
- 目标: <500ms P99, >100/sec

**Level 3: 集成级 (End-to-End)**
- 完整工作流
- 多步骤执行
- Agent 交互
- 目标: 功能正确 + 性能达标

**Level 4: 资源级 (System Monitoring)**
- 内存使用
- CPU 占用
- 并发连接
- 目标: <50MB, 支持 100+ agents

### 性能优化技巧

**已知优化:**
1. **Zap Logger** - 零分配日志 (Story 7.2)
2. **DSL 缓存** - 避免重复解析
3. **连接池** - Temporal gRPC 连接复用
4. **批量操作** - Event History 批量查询

**潜在优化 (Post-MVP):**
1. **响应缓存** - GET /v1/workflows/{id}
2. **YAML 解析优化** - 使用更快的 YAML 库
3. **并发控制** - 限制并发 Temporal 调用
4. **数据库优化** - Temporal PostgreSQL 调优

### 性能基线示例

**目标基线 (PRD 要求):**
```json
{
  "api": {
    "submit_workflow_p99_ms": 500,
    "get_status_p99_ms": 300,
    "validate_p99_ms": 500,
    "throughput_per_sec": 100
  },
  "dsl": {
    "parse_1000_lines_ms": 100,
    "validate_memory_mb": 10
  },
  "agent": {
    "idle_memory_mb": 50,
    "heap_memory_mb": 30
  },
  "temporal": {
    "event_history_query_ms": 100,
    "describe_workflow_ms": 50
  }
}
```

### 性能监控建议

**生产环境监控 (Post-MVP):**
1. **Prometheus 指标** - Story 7.5
2. **分布式追踪** - OpenTelemetry
3. **APM 工具** - New Relic, Datadog
4. **自定义仪表板** - Grafana

**关键指标:**
- API 延迟 (P50, P95, P99)
- 工作流提交速率
- 活跃工作流数量
- Agent 连接数
- 内存和 CPU 使用

## Completion Criteria

**Story 完成标准:**

1. ✅ **所有 AC 完成**
   - AC1-AC9 所有验收标准通过
   - 所有性能指标达标

2. ✅ **测试覆盖完整**
   - Go benchmarks 扩展
   - API 负载测试实现
   - 资源监控测试实现
   - 基线系统建立

3. ✅ **CI/CD 集成**
   - Makefile 目标添加
   - GitHub Actions 配置
   - PR 自动化测试

4. ✅ **文档完整**
   - 性能测试指南
   - 报告模板
   - README 更新

5. ✅ **性能达标**
   - API P99 < 500ms
   - YAML 解析 < 100ms
   - 吞吐量 > 100/sec
   - Agent 内存 < 50MB
   - 支持 100+ agents
   - Server 启动 < 5s
   - Event History 查询 < 100ms

**验收测试场景:**

**场景 1: 运行完整基准测试套件**
```bash
make benchmark
# 预期: 所有测试通过,生成报告
```

**场景 2: 建立性能基线**
```bash
scripts/benchmark_baseline.sh
# 预期: baseline-*.json 保存到 baseline/
```

**场景 3: 检测性能回归**
```bash
# 修改代码引入性能问题
scripts/compare_benchmarks.py --baseline ... --current ...
# 预期: 检测到回归,非零退出码
```

**场景 4: API 负载测试**
```bash
test/performance/api_benchmark.sh
# 预期: P99 < 500ms
```

**场景 5: 并发 Agent 测试**
```bash
AGENT_COUNT=100 test/performance/concurrent_agents_test.sh
# 预期: 100 agents 成功连接
```

**场景 6: Server 启动时间测试**
```bash
test/performance/server_startup_test.sh
# 预期: 启动时间 < 5s
```

## References

**现有代码参考:**
- [pkg/dsl/validator_bench_test.go](../../pkg/dsl/validator_bench_test.go)
- [pkg/dsl/expr_engine_bench_test.go](../../pkg/dsl/expr_engine_bench_test.go)
- [pkg/dsl/matrix_bench_test.go](../../pkg/dsl/matrix_bench_test.go)
- [testdata/benchmark/](../../testdata/benchmark/)
- [Makefile](../../Makefile)

**相关 Story:**
- Story 7.1 - 类型化错误处理 (错误性能影响)
- Story 7.2 - 结构化日志系统 (日志性能)
- Story 7.4 - 压力测试和容错验证 (更高负载)
- Story 7.5 - Prometheus 指标导出 (生产监控)

**外部参考:**
- [Go Benchmark](https://pkg.go.dev/testing#hdr-Benchmarks)
- [hey - HTTP load generator](https://github.com/rakyll/hey)
- [vegeta - HTTP load testing tool](https://github.com/tsenart/vegeta)
- [Performance Testing Best Practices](https://martinfowler.com/articles/performance-testing.html)

---

**创建日期:** 2026-01-07  
**创建者:** SM Agent (Bob)  
**Epic:** 7 - 生产级可靠性  
**依赖:** Story 7.1, 7.2 完成  
**预估点数:** 13 points (复杂度高,涉及多层次测试)  
**优先级:** High (生产就绪的关键验证)
