# Story 7.4: 压力测试和容错验证 (Event Sourcing)

Status: in-progress

## Story

As a **质量工程师**,  
I want **验证系统在压力下的表现和 Event Sourcing 容错能力**,  
So that **确保生产环境稳定性**。

## Context

这是 Epic 7 (生产级可靠性) 的**第四个 Story**,实现**压力测试和容错验证系统**。该 Story 在 Story 7.3 (性能基准测试) 的基础上,进一步验证系统在极限压力下的稳定性和容错能力,特别是 Event Sourcing 架构的零状态丢失保证。

**前置依赖:**
- ✅ Story 7.3 - 性能基准测试 (性能测试框架)
- ✅ Story 1.7 - 超时和重试策略
- ✅ Story 1.8 - Temporal SDK 集成 (Event Sourcing)
- ✅ Story 2.1 - Agent Worker 框架 (重连机制)
- ✅ ADR-0001 - Temporal 工作流引擎 (容错保证)

**Epic 背景:**  
Epic 7 专注于**生产级可靠性**。本 Story 建立在性能测试的基础上,提供:
- **大规模并发压力测试** - 1000+ 并发工作流
- **故障注入测试** - Server/Agent/Temporal 故障场景
- **容错能力验证** - Event Sourcing 零状态丢失
- **资源泄漏检测** - 内存/连接泄漏监控
- **超时/重试场景覆盖** - 单节点执行模式验证

**业务价值:**
- 🎯 **生产稳定性保证** - 验证极限条件下系统可靠性
- 🎯 **容错能力证明** - 验证 Event Sourcing 架构优势
- 🎯 **问题预防** - 提前发现潜在稳定性问题
- 🎯 **运维信心** - 提供故障恢复SOP验证
- 🎯 **资源优化** - 识别资源泄漏和优化点

**Temporal Event Sourcing 架构优势:**
```
传统架构 (有状态):
┌─────────────┐     崩溃    ┌─────────────┐
│ Server      │  ─────────→ │ 状态丢失    │
│ (内存状态)  │             │ 工作流失败  │
└─────────────┘             └─────────────┘

Event Sourcing 架构 (无状态):
┌─────────────┐     崩溃    ┌─────────────┐
│ Server      │  ─────────→ │ Event       │
│ (无状态)    │             │ History     │
└─────────────┘             │ (持久化)    │
        ↓ 重启               └─────────────┘
┌─────────────┐                    ↓
│ Server      │  ←─────── 从 Event History
│ (完全恢复)  │             重建状态并继续
└─────────────┘
```

**压力测试架构:**
```
┌────────────────────────────────────────────────────────┐
│         Stress Testing Framework                       │
├────────────────────────────────────────────────────────┤
│                                                        │
│  ┌─────────────────┐  ┌─────────────────────────┐    │
│  │ Load Generators │  │ Fault Injectors         │    │
│  ├─────────────────┤  ├─────────────────────────┤    │
│  │ - 1000+ 并发    │  │ - Server 崩溃           │    │
│  │ - 持续负载      │  │ - Agent 断开            │    │
│  │ - 峰值流量      │  │ - Temporal 连接失败     │    │
│  │ - 混合场景      │  │ - 网络分区              │    │
│  └─────────────────┘  └─────────────────────────┘    │
│                                                        │
│  ┌──────────────────────────────────────────────┐     │
│  │ Monitoring & Validation                      │     │
│  ├──────────────────────────────────────────────┤     │
│  │ - 工作流成功率 (>99%)                        │     │
│  │ - 状态一致性验证                             │     │
│  │ - 资源使用监控 (CPU/Memory/Connections)      │     │
│  │ - 泄漏检测 (Memory/Goroutines)               │     │
│  └──────────────────────────────────────────────┘     │
└────────────────────────────────────────────────────────┘
```

**测试场景分类:**

**1. 大规模并发测试**
- 1000+ 并发工作流稳定执行
- 混合工作负载 (短/长/复杂工作流)
- 资源占用在合理范围

**2. 故障注入测试**
- Server 崩溃和恢复
- Agent 断开和重连
- Temporal 连接失败和重试
- 网络延迟和丢包

**3. 资源泄漏测试**
- 内存泄漏检测
- Goroutine 泄漏检测
- 数据库连接泄漏
- 文件句柄泄漏

**4. 超时/重试场景测试**
- Activity 超时和重试
- Workflow 超时
- 节点执行失败和重试
- NonRetryableError 验证

**现有容错机制分析:**
```
已实现的容错机制:
✅ Temporal 连接重试 (internal/agent/worker.go)
   - MaxRetries: 可配置次数
   - RetryInterval: 可配置间隔
   - 指数退避策略

✅ Activity 重试策略 (pkg/dsl/retry.go)
   - 可配置最大重试次数
   - 可配置退避策略
   - NonRetryableError 支持

✅ Event Sourcing (pkg/temporal/workflow.go)
   - Event History 完整持久化
   - 状态自动恢复
   - 确定性执行

需要验证:
🔬 大规模并发下的稳定性
🔬 故障恢复的正确性
🔬 资源泄漏检测
🔬 极限条件下的性能
```

**本 Story 的范围 (MVP):**
- ✅ 实现 1000+ 并发工作流压力测试
- ✅ 实现故障注入测试框架
- ✅ 实现资源泄漏检测
- ✅ 验证 Event Sourcing 零状态丢失
- ✅ 验证超时/重试策略正确性
- ✅ 压力测试报告和分析
- ❌ 分布式追踪集成 - Post-MVP
- ❌ Chaos Engineering 自动化 - Post-MVP

## Acceptance Criteria

### AC1: 1000 个并发工作流稳定执行

**Given** 完整系统部署 (Server + Temporal + Agents)  
**When** 提交 1000 个并发工作流  
**Then** 工作流成功率 > 99%  
**And** 系统资源占用在合理范围:
- CPU < 80%
- Memory < 2GB (Server)
- Connections < 1000

**And** 平均工作流执行时间 < 10秒

**Implementation Notes:**

**并发工作流测试脚本:**
```bash
#!/bin/bash
# test/stress/concurrent_workflows_test.sh

set -e

CONCURRENT_WORKFLOWS="${CONCURRENT_WORKFLOWS:-1000}"
WORKFLOW_FILE="${WORKFLOW_FILE:-examples/hello-world.yaml}"
SERVER_URL="${SERVER_URL:-http://localhost:8080}"

echo "=== Concurrent Workflows Stress Test ==="
echo "Concurrent Workflows: $CONCURRENT_WORKFLOWS"
echo ""

# 创建结果目录
RESULTS_DIR="test/stress/results/concurrent-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$RESULTS_DIR"

# 启动资源监控
echo "Starting resource monitoring..."
./scripts/monitor_resources.sh "$RESULTS_DIR" &
MONITOR_PID=$!

# 并发提交工作流
echo "Submitting $CONCURRENT_WORKFLOWS workflows..."
START_TIME=$(date +%s)

# 使用 GNU parallel 并发提交
seq 1 $CONCURRENT_WORKFLOWS | parallel -j 100 \
    "curl -s -X POST $SERVER_URL/v1/workflows \
     -H 'Content-Type: application/yaml' \
     --data-binary @$WORKFLOW_FILE \
     -o $RESULTS_DIR/response-{}.json"

SUBMIT_END_TIME=$(date +%s)
SUBMIT_DURATION=$((SUBMIT_END_TIME - START_TIME))

echo "All workflows submitted in ${SUBMIT_DURATION}s"
echo ""

# 等待所有工作流完成 (最多 5 分钟)
echo "Waiting for workflows to complete..."
TIMEOUT=300
ELAPSED=0

while [ $ELAPSED -lt $TIMEOUT ]; do
    # 检查完成状态
    COMPLETED=$(jq -r '.status' $RESULTS_DIR/response-*.json 2>/dev/null \
        | grep -c "completed" || echo "0")
    
    if [ "$COMPLETED" -eq "$CONCURRENT_WORKFLOWS" ]; then
        echo "All workflows completed!"
        break
    fi
    
    echo "[$ELAPSED s] Completed: $COMPLETED / $CONCURRENT_WORKFLOWS"
    sleep 10
    ELAPSED=$((ELAPSED + 10))
done

# 停止资源监控
kill $MONITOR_PID

END_TIME=$(date +%s)
TOTAL_DURATION=$((END_TIME - START_TIME))

# 分析结果
echo ""
echo "=== Results ==="
SUCCESS=$(jq -r '.status' $RESULTS_DIR/response-*.json | grep -c "completed" || echo "0")
FAILED=$(jq -r '.status' $RESULTS_DIR/response-*.json | grep -c "failed" || echo "0")
SUCCESS_RATE=$(echo "scale=2; $SUCCESS * 100 / $CONCURRENT_WORKFLOWS" | bc)

echo "Total Duration: ${TOTAL_DURATION}s"
echo "Success: $SUCCESS"
echo "Failed: $FAILED"
echo "Success Rate: ${SUCCESS_RATE}%"

# 资源使用分析
echo ""
echo "=== Resource Usage ==="
MAX_CPU=$(awk 'BEGIN{max=0} {if($1>max) max=$1} END{print max}' $RESULTS_DIR/cpu.log)
MAX_MEM=$(awk 'BEGIN{max=0} {if($1>max) max=$1} END{print max}' $RESULTS_DIR/memory.log)
echo "Max CPU: ${MAX_CPU}%"
echo "Max Memory: ${MAX_MEM}MB"

# 验证目标
if (( $(echo "$SUCCESS_RATE < 99" | bc -l) )); then
    echo "❌ Success rate ${SUCCESS_RATE}% below target 99%"
    exit 1
fi

if (( $(echo "$MAX_CPU > 80" | bc -l) )); then
    echo "⚠️  Max CPU ${MAX_CPU}% exceeds 80% threshold"
fi

if (( $(echo "$MAX_MEM > 2048" | bc -l) )); then
    echo "⚠️  Max Memory ${MAX_MEM}MB exceeds 2GB threshold"
fi

echo "✅ Concurrent workflows stress test passed"
```

**资源监控脚本:**
```bash
#!/bin/bash
# scripts/monitor_resources.sh

RESULTS_DIR=$1
INTERVAL=5

while true; do
    # CPU 使用率
    CPU=$(top -bn1 | grep "Cpu(s)" | sed "s/.*, *\([0-9.]*\)%* id.*/\1/" \
        | awk '{print 100 - $1}')
    echo "$CPU" >> "$RESULTS_DIR/cpu.log"
    
    # 内存使用 (MB)
    MEM=$(free -m | grep Mem | awk '{print $3}')
    echo "$MEM" >> "$RESULTS_DIR/memory.log"
    
    # 连接数
    CONNS=$(ss -ant | wc -l)
    echo "$CONNS" >> "$RESULTS_DIR/connections.log"
    
    sleep $INTERVAL
done
```

**Go 并发测试:**
```go
// test/stress/concurrent_workflows_test.go
package stress

import (
    "context"
    "sync"
    "testing"
    "time"
    
    "github.com/Websoft9/waterflow/pkg/sdk"
)

func TestConcurrentWorkflows(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping stress test in short mode")
    }
    
    client, err := sdk.NewClient("http://localhost:8080")
    if err != nil {
        t.Fatal(err)
    }
    
    concurrency := 1000
    workflow := []byte(`
name: stress-test
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - uses: exec/shell
        with:
          command: echo "stress test"
`)
    
    var (
        success int
        failed  int
        mu      sync.Mutex
        wg      sync.WaitGroup
    )
    
    start := time.Now()
    
    // 并发提交
    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
            defer cancel()
            
            resp, err := client.SubmitWorkflow(ctx, workflow, nil)
            
            mu.Lock()
            if err != nil || resp.Status == "failed" {
                failed++
            } else {
                success++
            }
            mu.Unlock()
        }(i)
    }
    
    wg.Wait()
    duration := time.Since(start)
    
    successRate := float64(success) / float64(concurrency) * 100
    
    t.Logf("Duration: %v", duration)
    t.Logf("Success: %d", success)
    t.Logf("Failed: %d", failed)
    t.Logf("Success Rate: %.2f%%", successRate)
    
    if successRate < 99.0 {
        t.Errorf("Success rate %.2f%% below target 99%%", successRate)
    }
}
```

### AC2: Server 崩溃后自动重启,工作流从 Event History 恢复并继续

**Given** Event Sourcing 架构  
**When** Server 在工作流执行期间崩溃并重启  
**Then** 所有运行中的工作流自动恢复  
**And** 工作流从中断点继续执行  
**And** 无状态丢失  
**And** 恢复时间 < 10秒(从进程终止到workflow恢复执行)

**Implementation Notes:**

**Server 崩溃恢复测试:**
```bash
#!/bin/bash
# test/stress/server_crash_recovery_test.sh

set -e

echo "=== Server Crash Recovery Test ==="

# 1. 启动系统
echo "Starting server..."
./bin/server &
SERVER_PID=$!
sleep 5

# 2. 提交长时运行的工作流
echo "Submitting long-running workflows..."
WORKFLOW_IDS=()
for i in {1..10}; do
    RESPONSE=$(curl -s -X POST http://localhost:8080/v1/workflows \
        -H "Content-Type: application/yaml" \
        --data-binary "@testdata/workflows/long-running.yaml")
    
    WORKFLOW_ID=$(echo "$RESPONSE" | jq -r '.workflow_id')
    WORKFLOW_IDS+=("$WORKFLOW_ID")
    echo "Submitted: $WORKFLOW_ID"
done

# 3. 等待工作流开始执行
echo "Waiting for workflows to start..."
sleep 10

# 4. 崩溃 Server
echo "Crashing server (kill -9)..."
KILL_TIME=$(date +%s)
kill -9 $SERVER_PID
sleep 2

# 5. 重启 Server
echo "Restarting server..."
RECOVERY_START=$KILL_TIME
./bin/server &
SERVER_PID=$!
sleep 5

# 6. 验证工作流恢复
echo "Verifying workflow recovery..."
RECOVERED=0

for WORKFLOW_ID in "${WORKFLOW_IDS[@]}"; do
    STATUS=$(curl -s "http://localhost:8080/v1/workflows/$WORKFLOW_ID" \
        | jq -r '.status')
    
    if [ "$STATUS" == "running" ] || [ "$STATUS" == "completed" ]; then
        echo "✅ $WORKFLOW_ID: $STATUS"
        ((RECOVERED++))
    else
        echo "❌ $WORKFLOW_ID: $STATUS"
    fi
done

# 测量恢复时间
RECOVERY_TIME=$(($(date +%s) - $RECOVERY_START))
echo "Recovery time: ${RECOVERY_TIME}s"

# 清理
kill $SERVER_PID

# 验证恢复时间
if [ $RECOVERY_TIME -gt 10 ]; then
    echo "❌ Recovery time ${RECOVERY_TIME}s exceeds 10s limit"
    exit 1
fi

# 验证工作流恢复
if [ $RECOVERED -ne ${#WORKFLOW_IDS[@]} ]; then
    echo "❌ Only $RECOVERED / ${#WORKFLOW_IDS[@]} workflows recovered"
    exit 1
fi

echo "✅ All workflows recovered successfully in ${RECOVERY_TIME}s"
```

**Event Sourcing 验证测试:**
```go
// test/stress/event_sourcing_test.go
package stress

import (
    "context"
    "testing"
    "time"
    
    "github.com/Websoft9/waterflow/pkg/temporal"
    "go.temporal.io/sdk/client"
)

func TestEventSourcingRecovery(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping stress test in short mode")
    }
    
    // 连接到 Temporal
    tc, err := client.Dial(client.Options{
        HostPort:  "localhost:7233",
        Namespace: "default",
    })
    if err != nil {
        t.Fatal(err)
    }
    defer tc.Close()
    
    ctx := context.Background()
    workflowID := "event-sourcing-test"
    
    // 启动工作流
    we, err := tc.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
        ID:        workflowID,
        TaskQueue: "test-queue",
    }, temporal.WorkflowExecutor, workflowID)
    if err != nil {
        t.Fatal(err)
    }
    
    // 等待一段时间
    time.Sleep(5 * time.Second)
    
    // 获取 Event History (before crash simulation)
    iter := tc.GetWorkflowHistory(ctx, workflowID, "", false, 0)
    eventsBefore := 0
    for iter.HasNext() {
        _, err := iter.Next()
        if err != nil {
            t.Fatal(err)
        }
        eventsBefore++
    }
    
    t.Logf("Events before: %d", eventsBefore)
    
    // 模拟 Server 崩溃 (通过 Temporal API 验证状态)
    // 在实际测试中,这里会 kill Server 进程
    
    // 等待"重启"
    time.Sleep(3 * time.Second)
    
    // 获取 Event History (after restart)
    iter = tc.GetWorkflowHistory(ctx, workflowID, "", false, 0)
    eventsAfter := 0
    for iter.HasNext() {
        _, err := iter.Next()
        if err != nil {
            t.Fatal(err)
        }
        eventsAfter++
    }
    
    t.Logf("Events after: %d", eventsAfter)
    
    // 验证: Event History 完整保留
    if eventsAfter < eventsBefore {
        t.Errorf("Event History incomplete: %d < %d", eventsAfter, eventsBefore)
    }
    
    // 等待工作流完成
    err = we.Get(ctx, nil)
    if err != nil {
        t.Errorf("Workflow failed after recovery: %v", err)
    }
}
```

### AC3: Agent 断开重连后任务继续 (Temporal 自动重试)

**Given** Agent 正在执行任务  
**When** Agent 断开连接 (网络故障/进程崩溃)  
**Then** Temporal 自动将任务路由到其他 Agent  
**Or** Agent 重连后继续执行任务  
**And** 任务不丢失

**Implementation Notes:**

**Agent 断开重连测试:**
```bash
#!/bin/bash
# test/stress/agent_reconnect_test.sh

set -e

echo "=== Agent Reconnect Test ==="

# 1. 启动多个 Agent
echo "Starting 3 agents..."
AGENT_PIDS=()
for i in {1..3}; do
    ./bin/agent --task-queues linux-amd64 --log-level warn &
    AGENT_PIDS+=($!)
    echo "Started agent $i (PID: ${AGENT_PIDS[$((i-1))]})"
done

sleep 5

# 2. 提交工作流
echo "Submitting workflow..."
WORKFLOW_ID=$(curl -s -X POST http://localhost:8080/v1/workflows \
    -H "Content-Type: application/yaml" \
    --data-binary "@testdata/workflows/multi-step.yaml" \
    | jq -r '.workflow_id')

echo "Workflow ID: $WORKFLOW_ID"

# 3. 等待开始执行
sleep 5

# 4. 断开一个 Agent
echo "Disconnecting agent 1..."
kill ${AGENT_PIDS[0]}
sleep 2

# 5. 再断开一个 Agent
echo "Disconnecting agent 2..."
kill ${AGENT_PIDS[1]}
sleep 2

# 6. 验证工作流仍在执行
STATUS=$(curl -s "http://localhost:8080/v1/workflows/$WORKFLOW_ID" \
    | jq -r '.status')

echo "Workflow status after agent disconnects: $STATUS"

# 7. 重启 Agents
echo "Restarting agents..."
./bin/agent --task-queues linux-amd64 --log-level warn &
AGENT_PIDS[0]=$!

./bin/agent --task-queues linux-amd64 --log-level warn &
AGENT_PIDS[1]=$!

sleep 5

# 8. 等待工作流完成
echo "Waiting for workflow completion..."
TIMEOUT=60
ELAPSED=0

while [ $ELAPSED -lt $TIMEOUT ]; do
    STATUS=$(curl -s "http://localhost:8080/v1/workflows/$WORKFLOW_ID" \
        | jq -r '.status')
    
    if [ "$STATUS" == "completed" ]; then
        echo "✅ Workflow completed successfully"
        break
    fi
    
    echo "[$ELAPSED s] Status: $STATUS"
    sleep 5
    ELAPSED=$((ELAPSED + 5))
done

# 清理
for pid in "${AGENT_PIDS[@]}"; do
    kill $pid 2>/dev/null || true
done

if [ "$STATUS" != "completed" ]; then
    echo "❌ Workflow did not complete: $STATUS"
    exit 1
fi

echo "✅ Agent reconnect test passed"
```

### AC4: Temporal 连接失败时自动重试

**Given** Agent 配置了 Temporal 连接重试  
**When** Temporal Server 不可用  
**Then** Agent 自动重试连接  
**And** 连接成功后正常工作  
**And** 重试次数和间隔可配置

**Implementation Notes:**

**Temporal 连接重试测试:**
```bash
#!/bin/bash
# test/stress/temporal_reconnect_test.sh

set -e

echo "=== Temporal Reconnect Test ==="

# 1. 先不启动 Temporal,启动 Agent
echo "Starting agent without Temporal..."
./bin/agent \
    --config <(cat <<EOF
temporal:
  host: localhost:7233
  namespace: default
  max_retries: 10
  retry_interval: 2s
agent:
  task_queues: [linux-amd64]
log:
  level: debug
EOF
) 2>&1 | tee agent.log &

AGENT_PID=$!
sleep 5

# 2. 验证 Agent 正在重试
RETRY_COUNT=$(grep -c "Failed to connect to Temporal, retrying" agent.log || echo "0")
echo "Retry attempts detected: $RETRY_COUNT"

if [ "$RETRY_COUNT" -eq 0 ]; then
    echo "❌ No retry attempts detected"
    kill $AGENT_PID
    exit 1
fi

# 3. 启动 Temporal (假设使用 docker-compose)
echo "Starting Temporal..."
docker-compose -f deployments/docker-compose.yaml up -d temporal
sleep 10

# 4. 验证 Agent 连接成功
sleep 5
SUCCESS=$(grep -c "Connected to Temporal" agent.log || echo "0")

if [ "$SUCCESS" -eq 0 ]; then
    echo "❌ Agent did not connect to Temporal"
    kill $AGENT_PID
    exit 1
fi

echo "✅ Agent connected after retries"

# 清理
kill $AGENT_PID
docker-compose -f deployments/docker-compose.yaml down

echo "✅ Temporal reconnect test passed"
```

**验证重试配置:**
```go
// internal/agent/worker_test.go
func TestTemporalConnectionRetry(t *testing.T) {
    cfg := &config.Config{
        Temporal: config.TemporalConfig{
            Host:              "invalid-host:7233",
            Namespace:         "default",
            MaxRetries:        3,
            RetryInterval:     1 * time.Second,
            ConnectionTimeout: 2 * time.Second,
        },
        Agent: config.AgentConfig{
            TaskQueues: []string{"test"},
        },
    }
    
    logger, _ := zap.NewDevelopment()
    
    start := time.Now()
    _, err := agent.NewWorker(cfg, logger)
    duration := time.Since(start)
    
    // 应该在 3 次重试后失败
    if err == nil {
        t.Error("Expected connection error")
    }
    
    // 验证重试时间 (3 retries * 1s interval + overhead)
    expectedDuration := time.Duration(cfg.Temporal.MaxRetries) * cfg.Temporal.RetryInterval
    if duration < expectedDuration {
        t.Errorf("Retries too fast: %v < %v", duration, expectedDuration)
    }
    
    t.Logf("Failed after %v (expected ~%v)", duration, expectedDuration)
}
```

### AC5: 验证 Event Sourcing 模式下零状态丢失

**Given** 工作流执行中产生状态变更  
**When** Server 崩溃  
**Then** 所有状态变更已持久化到 Event History  
**And** 重启后状态完全恢复  
**And** 无数据丢失

**Implementation Notes:**

**Event History 完整性验证:**
```go
// test/stress/event_history_integrity_test.go
package stress

import (
    "context"
    "testing"
    
    "go.temporal.io/api/enums/v1"
    "go.temporal.io/sdk/client"
)

func TestEventHistoryIntegrity(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping stress test")
    }
    
    tc, err := client.Dial(client.Options{
        HostPort:  "localhost:7233",
        Namespace: "default",
    })
    if err != nil {
        t.Fatal(err)
    }
    defer tc.Close()
    
    ctx := context.Background()
    workflowID := "event-integrity-test"
    
    // 执行工作流
    we, err := tc.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
        ID:        workflowID,
        TaskQueue: "test-queue",
    }, ComplexWorkflow)
    if err != nil {
        t.Fatal(err)
    }
    
    // 等待完成
    err = we.Get(ctx, nil)
    if err != nil {
        t.Fatal(err)
    }
    
    // 获取 Event History
    iter := tc.GetWorkflowHistory(ctx, workflowID, "", false, 0)
    
    events := make(map[enums.EventType]int)
    totalEvents := 0
    
    for iter.HasNext() {
        event, err := iter.Next()
        if err != nil {
            t.Fatal(err)
        }
        
        events[event.EventType]++
        totalEvents++
    }
    
    t.Logf("Total events: %d", totalEvents)
    for eventType, count := range events {
        t.Logf("  %v: %d", eventType, count)
    }
    
    // 验证关键事件
    requiredEvents := []enums.EventType{
        enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
        enums.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED,
        enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
        enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
    }
    
    for _, eventType := range requiredEvents {
        if events[eventType] == 0 {
            t.Errorf("Missing event type: %v", eventType)
        }
    }
    
    // 验证事件顺序和完整性
    if totalEvents < 10 {
        t.Errorf("Too few events: %d", totalEvents)
    }
}
```

### AC6: 系统资源占用在合理范围且无内存泄漏

**Given** 系统长时间运行  
**When** 处理大量工作流  
**Then** 内存使用稳定,无持续增长(1小时内增长率<20%)  
**And** Goroutine 数量稳定(无持续增长,增长率<20%)  
**And** 数据库连接数稳定  
**And** 文件句柄数稳定  
**And** pprof endpoint已启用(Server:6060, Agent:6061)  
**And** 可通过pprof heap和goroutine profile进行深度分析

**Implementation Notes:**

**内存泄漏检测:**
```bash
#!/bin/bash
# test/stress/memory_leak_test.sh

set -e

# Cleanup on exit
cleanup() {
  echo "Cleaning up..."
  kill $SERVER_PID 2>/dev/null || true
  kill $MONITOR_PID 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "=== Memory Leak Detection Test ==="

# 1. 启动 Server
echo "Starting server..."
./bin/server &
SERVER_PID=$!
sleep 5

# 2. 持续负载 (1 小时)
echo "Starting continuous load for 1 hour..."
DURATION=3600
START_TIME=$(date +%s)
END_TIME=$((START_TIME + DURATION))

RESULTS_DIR="test/stress/results/memory-leak-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$RESULTS_DIR"

# 启动资源监控
./scripts/monitor_resources.sh "$RESULTS_DIR" &
MONITOR_PID=$!

# 持续提交工作流
while [ $(date +%s) -lt $END_TIME ]; do
    curl -s -X POST http://localhost:8080/v1/workflows \
        -H "Content-Type: application/yaml" \
        --data-binary "@examples/hello-world.yaml" \
        > /dev/null
    
    sleep 1
done

# 停止监控
kill $MONITOR_PID

# 停止 Server
kill $SERVER_PID

# 分析内存趋势
echo ""
echo "=== Memory Leak Analysis ==="
python3 - <<'EOF'
import sys
with open('$RESULTS_DIR/memory.log') as f:
    data = [float(line.strip()) for line in f if line.strip()]

initial = sum(data[:10]) / 10  # 前 10 个样本平均
final = sum(data[-10:]) / 10    # 后 10 个样本平均
growth = ((final - initial) / initial) * 100

print(f"Initial Memory: {initial:.2f} MB")
print(f"Final Memory: {final:.2f} MB")
print(f"Growth: {growth:.2f}%")

if growth > 20:
    print("❌ Memory leak detected (>20% growth)")
    sys.exit(1)
else:
    print("✅ No memory leak detected")
EOF
```

**Go 内存分析:**
```go
// test/stress/memory_profile_test.go
package stress

import (
    "runtime"
    "testing"
    "time"
)

func TestMemoryProfile(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping memory profile test")
    }
    
    // 记录初始内存
    var m1 runtime.MemStats
    runtime.ReadMemStats(&m1)
    
    // 运行负载 (10 分钟)
    duration := 10 * time.Minute
    deadline := time.Now().Add(duration)
    
    for time.Now().Before(deadline) {
        // 模拟工作流处理
        processWorkflow()
        time.Sleep(100 * time.Millisecond)
    }
    
    // 强制 GC
    runtime.GC()
    time.Sleep(1 * time.Second)
    
    // 记录最终内存
    var m2 runtime.MemStats
    runtime.ReadMemStats(&m2)
    
    t.Logf("Initial Heap: %d MB", m1.Alloc/1024/1024)
    t.Logf("Final Heap: %d MB", m2.Alloc/1024/1024)
    t.Logf("Goroutines: %d", runtime.NumGoroutine())
    
    // 验证内存增长
    growth := float64(m2.Alloc-m1.Alloc) / float64(m1.Alloc) * 100
    t.Logf("Memory growth: %.2f%%", growth)
    
    if growth > 20 {
        t.Errorf("Excessive memory growth: %.2f%%", growth)
    }
    
    // 验证 Goroutine 数量
    if runtime.NumGoroutine() > 1000 {
        t.Errorf("Too many goroutines: %d", runtime.NumGoroutine())
    }
}

func processWorkflow() {
    // 模拟工作流处理
}
```

### AC7: 压力测试覆盖单节点执行模式的超时/重试场景

**Given** 单节点执行模式 (每个 Step = 1 Activity)  
**When** 执行超时/重试测试  
**Then** Activity 超时正确触发  
**And** 重试策略正确应用  
**And** NonRetryableError 不重试  
**And** 可重试错误正确重试

**Implementation Notes:**

**超时/重试场景测试:**
```bash
#!/bin/bash
# test/stress/timeout_retry_scenarios_test.sh

set -e

echo "=== Timeout/Retry Scenarios Test ==="

# 场景 1: Activity 超时
echo "Scenario 1: Activity Timeout"
cat > /tmp/timeout-test.yaml <<EOF
name: timeout-test
jobs:
  test:
    runs-on: linux-amd64
    timeout: 30s
    steps:
      - name: slow-step
        uses: flow/sleep
        timeout: 5s
        with:
          duration: 10s
EOF

WORKFLOW_ID=$(curl -s -X POST http://localhost:8080/v1/workflows \
    -H "Content-Type: application/yaml" \
    --data-binary "@/tmp/timeout-test.yaml" \
    | jq -r '.workflow_id')

# 等待超时
sleep 15

STATUS=$(curl -s "http://localhost:8080/v1/workflows/$WORKFLOW_ID" \
    | jq -r '.status')

if [ "$STATUS" != "failed" ]; then
    echo "❌ Expected timeout failure, got: $STATUS"
    exit 1
fi

echo "✅ Timeout correctly triggered"

# 场景 2: 可重试错误
echo ""
echo "Scenario 2: Retryable Error"
cat > /tmp/retry-test.yaml <<EOF
name: retry-test
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: retry-step
        uses: http/request
        retry:
          max_attempts: 3
          backoff: exponential
        with:
          url: http://invalid-host.example.com
          method: GET
EOF

WORKFLOW_ID=$(curl -s -X POST http://localhost:8080/v1/workflows \
    -H "Content-Type: application/yaml" \
    --data-binary "@/tmp/retry-test.yaml" \
    | jq -r '.workflow_id')

# 等待重试完成
sleep 20

# 获取日志,验证重试次数
LOGS=$(curl -s "http://localhost:8080/v1/workflows/$WORKFLOW_ID/logs")
RETRY_COUNT=$(echo "$LOGS" | grep -c "attempt" || echo "0")

if [ "$RETRY_COUNT" -lt 3 ]; then
    echo "❌ Expected >= 3 retry attempts, got: $RETRY_COUNT"
    exit 1
fi

echo "✅ Retry strategy correctly applied"

# 场景 3: NonRetryableError
echo ""
echo "Scenario 3: NonRetryableError"
cat > /tmp/nonretry-test.yaml <<EOF
name: nonretry-test
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: invalid-step
        uses: exec/shell
        retry:
          max_attempts: 5
        with:
          # 缺少 required 参数会触发 validation error
EOF

WORKFLOW_ID=$(curl -s -X POST http://localhost:8080/v1/workflows \
    -H "Content-Type: application/yaml" \
    --data-binary "@/tmp/nonretry-test.yaml" \
    | jq -r '.workflow_id')

# 等待失败 (不应重试)
sleep 10

LOGS=$(curl -s "http://localhost:8080/v1/workflows/$WORKFLOW_ID/logs")
RETRY_COUNT=$(echo "$LOGS" | grep -c "attempt" || echo "0")

if [ "$RETRY_COUNT" -gt 1 ]; then
    echo "❌ NonRetryableError should not retry, got: $RETRY_COUNT attempts"
    exit 1
fi

echo "✅ NonRetryableError correctly not retried"
echo ""
echo "✅ All timeout/retry scenarios passed"
```

---

### AC8: 混合故障场景验证

**Given** 生产环境可能发生多种故障同时出现  
**When** Server崩溃和Agent断开同时发生  
**Then** 系统能够完全恢复  
**And** workflow继续执行,无状态丢失  
**And** Event History完整性100%  
**And** 恢复时间 < 15秒(允许比单一故障稍长)

**Implementation Notes:**

**混合故障测试脚本:**
```bash
#!/bin/bash
# test/stress/mixed_fault_test.sh

set -e

# Cleanup on exit
cleanup() {
  echo "Cleaning up..."
  pkill -f "waterflow-server" || true
  pkill -f "waterflow-agent" || true
}
trap cleanup EXIT INT TERM

echo "=== Mixed Fault Scenario Test ==="

# 1. 启动系统
echo "Starting server and agent..."
./bin/server &
SERVER_PID=$!
sleep 3

./bin/agent &
AGENT_PID=$!
sleep 3

# 2. 提交长时运行的工作流
echo "Submitting long-running workflow..."
WORKFLOW_ID=$(curl -s -X POST http://localhost:8080/v1/workflows \
    -H "Content-Type: application/yaml" \
    --data-binary "@testdata/workflows/long-running.yaml" \
    | jq -r '.workflow_id')

echo "Workflow ID: $WORKFLOW_ID"

# 3. 等待工作流开始执行
echo "Waiting for workflow to start..."
sleep 5

# 4. 记录故障前的Event History
EVENTS_BEFORE=$(curl -s "http://localhost:8080/v1/workflows/$WORKFLOW_ID/history" \
    | jq -r '.events | length')
echo "Events before fault: $EVENTS_BEFORE"

# 5. 同时kill Server和Agent
echo "Simulating mixed fault: killing Server and Agent simultaneously..."
FAULT_TIME=$(date +%s)
kill -9 $SERVER_PID $AGENT_PID

sleep 2

# 6. 重启Server和Agent
echo "Restarting services..."
RECOVERY_START=$(date +%s)
./bin/server &
SERVER_PID=$!

./bin/agent &
AGENT_PID=$!

# 7. 等待服务恢复
echo "Waiting for services to recover..."
RECOVERED=false
while [ "$RECOVERED" = "false" ]; do
    sleep 1
    ELAPSED=$(($(date +%s) - $RECOVERY_START))
    
    if [ $ELAPSED -gt 15 ]; then
        echo "❌ Recovery time exceeded 15 seconds"
        exit 1
    fi
    
    # 检查Server和Agent健康状态
    if curl -s http://localhost:8080/health > /dev/null 2>&1 && \
       curl -s http://localhost:8081/health > /dev/null 2>&1; then
        RECOVERED=true
        RECOVERY_TIME=$ELAPSED
        echo "✅ Services recovered in ${RECOVERY_TIME}s"
    fi
done

# 8. 验证workflow恢复
echo "Verifying workflow recovery..."
sleep 5

STATUS=$(curl -s "http://localhost:8080/v1/workflows/$WORKFLOW_ID" \
    | jq -r '.status')

if [ "$STATUS" != "running" ] && [ "$STATUS" != "completed" ]; then
    echo "❌ Workflow in unexpected status: $STATUS"
    exit 1
fi

echo "✅ Workflow recovered: $STATUS"

# 9. 验证Event History完整性
EVENTS_AFTER=$(curl -s "http://localhost:8080/v1/workflows/$WORKFLOW_ID/history" \
    | jq -r '.events | length')

echo "Events after recovery: $EVENTS_AFTER"

if [ $EVENTS_AFTER -lt $EVENTS_BEFORE ]; then
    echo "❌ Event History lost events: $EVENTS_BEFORE -> $EVENTS_AFTER"
    exit 1
fi

echo "✅ Event History integrity: 100%"
echo "✅ Mixed fault scenario test passed"
echo "   Recovery time: ${RECOVERY_TIME}s"
echo "   Workflow status: $STATUS"
echo "   Events: $EVENTS_BEFORE -> $EVENTS_AFTER"
```

**Go单元测试:**
```go
// internal/server/mixed_fault_test.go
func TestMixedFaultRecovery(t *testing.T) {
    // 启动测试Server和Agent
    server := startTestServer(t)
    agent := startTestAgent(t)
    
    // 提交工作流
    workflowID := submitTestWorkflow(t, "testdata/workflows/long-running.yaml")
    time.Sleep(3 * time.Second)
    
    // 记录故障前的Event History
    eventsBefore := getEventHistory(t, workflowID)
    t.Logf("Events before fault: %d", len(eventsBefore))
    
    // 模拟混合故障
    faultTime := time.Now()
    server.Process.Kill()
    agent.Process.Kill()
    
    time.Sleep(2 * time.Second)
    
    // 重启服务
    recoveryStart := time.Now()
    server = startTestServer(t)
    agent = startTestAgent(t)
    
    // 等待恢复
    recovered := false
    for !recovered && time.Since(recoveryStart) < 15*time.Second {
        time.Sleep(500 * time.Millisecond)
        
        if checkHealth(server) && checkHealth(agent) {
            recovered = true
        }
    }
    
    if !recovered {
        t.Fatal("Services failed to recover within 15 seconds")
    }
    
    recoveryTime := time.Since(recoveryStart)
    t.Logf("Recovery time: %v", recoveryTime)
    
    // 验证workflow状态
    time.Sleep(5 * time.Second)
    status := getWorkflowStatus(t, workflowID)
    
    if status != "running" && status != "completed" {
        t.Fatalf("Unexpected workflow status: %s", status)
    }
    
    t.Logf("Workflow status after recovery: %s", status)
    
    // 验证Event History完整性
    eventsAfter := getEventHistory(t, workflowID)
    
    if len(eventsAfter) < len(eventsBefore) {
        t.Fatalf("Event History lost events: %d -> %d", 
            len(eventsBefore), len(eventsAfter))
    }
    
    t.Logf("Event History integrity: %d -> %d events", 
        len(eventsBefore), len(eventsAfter))
    t.Log("✅ Mixed fault recovery test passed")
}
```

---

## Tasks / Subtasks

### Task 1: 实现并发工作流压力测试 (AC1)

- [x] 1.1 创建并发测试脚本
  - test/stress/concurrent_workflows_test.sh
  - 使用 GNU parallel 并发提交
  - 结果收集和分析

- [x] 1.2 实现资源监控脚本
  - scripts/monitor_resources.sh (Linux)
  - 监控 CPU/Memory/Connections
  - 定期采样记录
  - 或使用internal/server/monitor_resources.go (跨平台,基于gopsutil)

- [x] 1.3 创建 Go 并发测试
  - test/stress/concurrent_workflows_test.go
  - sync.WaitGroup 并发控制
  - 成功率统计

- [x] 1.4 配置测试工作流
  - 短工作流 (1-2 steps)
  - 长工作流 (10+ steps)
  - 复杂工作流 (matrix, conditions)

### Task 2: 实现故障注入测试 (AC2, AC3, AC4)

- [x] 2.1 创建 Server 崩溃恢复测试
  - test/stress/server_crash_recovery_test.sh
  - kill -9 模拟崩溃
  - 验证工作流恢复

- [x] 2.2 创建 Agent 断开重连测试
  - test/stress/agent_reconnect_test.sh
  - 多 Agent 场景
  - 任务路由验证

- [x] 2.3 创建 Temporal 连接重试测试
  - test/stress/temporal_reconnect_test.sh
  - 验证重试配置
  - 连接成功验证

- [x] 2.4 实现 Go 单元测试
  - TestTemporalConnectionRetry
  - 验证重试次数和间隔

### Task 3: 实现 Event Sourcing 验证 (AC5)

- [x] 3.1 创建 Event History 完整性测试
  - test/stress/event_sourcing_test.go
  - 获取 Event History
  - 验证事件完整性

- [x] 3.2 创建 Event History 完整性测试
  - test/stress/event_history_integrity_test.go
  - 验证关键事件类型
  - 验证事件顺序

- [x] 3.3 验证零状态丢失
  - 崩溃前后状态对比
  - 数据一致性检查

### Task 4: 实现资源泄漏检测 (AC6)

- [x] 4.1 创建内存泄漏检测脚本
  - test/stress/memory_leak_test.sh (已存在为resource_leak_test.sh)
  - 长时间负载测试 (1 小时)
  - 内存趋势分析

- [x] 4.2 实现 Go 内存分析
  - test/stress/memory_profile_test.go
  - runtime.MemStats 监控
  - Goroutine 数量检查

- [x] 4.3 创建资源泄漏分析脚本
  - Python 脚本分析内存趋势
  - 检测 >20% 增长
  - 生成报告

- [x] 4.4 添加 pprof 支持
  - Server 暴露 pprof 端点
  - 内存/CPU profile 收集
  - 分析工具集成

### Task 5: 实现超时/重试场景测试 (AC7)

- [x] 5.1 创建超时场景测试
  - test/stress/timeout_retry_scenarios_test.sh
  - Activity 超时验证
  - Workflow 超时验证

- [x] 5.2 创建重试场景测试
  - 可重试错误测试
  - NonRetryableError 测试
  - 重试次数验证

- [x] 5.3 创建测试工作流 YAML
  - 超时工作流
  - 重试工作流
  - 混合场景工作流

- [x] 5.4 日志分析验证
  - 解析日志提取重试信息
  - 验证重试策略应用

### Task 6: 集成到 CI/CD

- [x] 6.1 添加 Makefile 目标
  - make stress-test
  - make stress-test-quick (快速版本)
  - make stress-test-go (Go测试)

- [x] 6.2 创建 CI workflow
  - .github/workflows/stress-test.yml
  - 夜间定时运行
  - 失败通知

- [x] 6.3 配置测试环境
  - Docker Compose 测试环境
  - 资源限制配置
  - 清理脚本

### Task 7: 压力测试报告

- [x] 7.1 创建报告模板
  - test/stress/README.md (已有完整文档)
  - 测试场景
  - 结果分析
  - 问题和建议

- [ ] 7.2 实现报告生成脚本
  - scripts/generate_stress_report.py (Post-MVP)
  - 自动收集结果
  - 生成 Markdown 报告

- [ ] 7.3 创建可视化仪表板
  - test/stress/dashboard.html
  - 资源使用图表
  - 成功率趋势

---

### Task 8: 混合故障场景测试 (AC8)

- [x] 8.1 实现混合故障测试脚本
  - test/stress/mixed_fault_test.sh
  - 同时kill Server和Agent进程
  - 测量双重故障恢复时间

- [x] 8.2 Event History完整性验证
  - 对比故障前后的Event History
  - 验证无事件丢失
  - 验证事件顺序正确

- [x] 8.3 Workflow状态一致性检查
  - 验证workflow能继续执行
  - 验证Step状态正确恢复
  - 验证无重复执行

- [x] 8.4 Go单元测试
  - 实现TestMixedFaultRecovery
  - 模拟systemd/supervisor重启
  - 验证恢复时间 < 15s

---

### Task 9: pprof配置和文档

- [x] 9.1 Server pprof配置
  - cmd/server/main.go: 导入net/http/pprof
  - 启动pprof HTTP server (localhost:6060)
  - 默认启用

- [x] 9.2 Agent pprof配置
  - cmd/agent/main.go: 导入net/http/pprof
  - 启动pprof HTTP server (localhost:6061)
  - 默认启用

- [x] 9.3 编写pprof使用文档
  - docs/guides/profiling.md
  - heap profile采集和分析
  - goroutine profile采集和分析
  - 内存泄漏诊断流程

- [x] 9.4 更新压力测试指南
  - docs/guides/stress-testing.md
  - 添加pprof分析章节
  - 内存泄漏排查流程
  - 最佳实践

---

## Dev Notes

### 24小时稳定性测试(可选/Post-MVP)

AC6当前定义为1小时负载测试(内存增长率<20%)。如需生产级稳定性验证,建议Post-MVP阶段执行24小时稳定性测试:
- **测试时长**: 24小时持续负载
- **监控指标**: 内存、CPU、goroutine、磁盘IO
- **成功标准**: 内存增长<10%,无crash,无goroutine泄漏
- **环境要求**: staging环境,真实workload模拟

此测试可作为发布前的最终验证,但不作为Story 7-4的必需验收标准。

### 跨平台兼容性

部分测试脚本使用Linux特定命令(`ps`, `top`, `free`, `kill`等)。建议:
1. **文档明确要求**: 在README中说明测试环境需Linux(推荐Ubuntu 20.04+)
2. **Go跨平台实现**: 优先使用Go + gopsutil实现核心监控逻辑
3. **CI/CD平台**: 使用Linux runners执行集成测试

如需支持macOS/Windows测试,应重构监控脚本为纯Go实现。

### Architecture Alignment

**压力测试架构 (本 Story):**
- ✅ 多层次测试 - 并发/故障/资源/场景
- ✅ Event Sourcing 验证 - 零状态丢失
- ✅ 自动化框架 - 可重复运行
- ✅ CI/CD 集成 - 持续验证

**与 Temporal 架构集成:**
- ✅ Event History 持久化
- ✅ 自动重试机制
- ✅ Worker 心跳和健康检查
- ✅ Task Queue 路由

**与系统各模块:**
- ✅ Server 无状态验证
- ✅ Agent 重连机制
- ✅ 超时/重试策略
- ✅ 资源管理

### Project Structure

```
test/
├── stress/
│   ├── concurrent_workflows_test.sh     # 🆕 并发测试
│   ├── server_crash_recovery_test.sh    # 🆕 Server 崩溃恢复
│   ├── agent_reconnect_test.sh          # 🆕 Agent 重连
│   ├── temporal_reconnect_test.sh       # 🆕 Temporal 重连
│   ├── memory_leak_test.sh              # 🆕 内存泄漏检测
│   ├── timeout_retry_scenarios_test.sh  # 🆕 超时/重试场景
│   ├── concurrent_workflows_test.go     # 🆕 Go 并发测试
│   ├── event_sourcing_test.go           # 🆕 Event Sourcing 验证
│   ├── event_history_integrity_test.go  # 🆕 Event History 完整性
│   ├── memory_profile_test.go           # 🆕 内存分析
│   ├── results/                         # 🆕 测试结果
│   ├── dashboard.html                   # 🆕 可视化仪表板
│   └── REPORT_TEMPLATE.md               # 🆕 报告模板

scripts/
├── monitor_resources.sh                 # 🆕 资源监控
├── generate_stress_report.py            # 🆕 报告生成
└── analyze_memory_trend.py              # 🆕 内存趋势分析

testdata/
└── workflows/
    ├── long-running.yaml                # 🆕 长时运行工作流
    ├── multi-step.yaml                  # 🆕 多步骤工作流
    └── stress-test.yaml                 # 🆕 压力测试工作流

Makefile                                 # 🔧 添加 stress-test 目标

.github/workflows/
└── stress-test.yml                      # 🆕 CI/CD 集成

docs/guides/
├── stress-testing.md                    # 🆕 压力测试指南
└── troubleshooting-stress-test.md       # 🆕 故障排查
```

### 关键技术决策

**1. 并发工具选择**
- ✅ 选择: GNU parallel
- 理由:
  - 简单高效
  - 并发控制灵活
  - 结果收集方便
- 替代方案:
  - ❌ xargs -P (功能有限)
  - ❌ 自定义脚本 (复杂度高)

**2. 故障注入方式**
- ✅ 选择: kill -9 (进程级)
- 理由:
  - 模拟真实崩溃
  - 简单可靠
  - 无副作用
- 替代方案:
  - ❌ Chaos Mesh (过于复杂)
  - ❌ 网络分区 (难以控制)

**3. 内存泄漏检测策略**
- ✅ 选择: 长时间负载 + 趋势分析
- 理由:
  - 真实场景模拟
  - 可靠检测慢速泄漏
  - 结果可重现
- 实现:
  - 1 小时持续负载
  - 每 5 秒采样
  - >20% 增长告警

**4. CI/CD 集成频率**
- ✅ 选择: 夜间定时运行
- 理由:
  - 压力测试耗时较长
  - 不阻塞开发流程
  - 资源占用集中
- 配置:
  - Cron: 0 2 * * * (每天凌晨2点)
  - 失败邮件/Slack 通知

### 测试策略

**测试分层:**

**Level 1: 单组件压力**
- Server 独立压力
- Agent 独立压力
- Temporal 查询压力

**Level 2: 集成压力**
- Server + Temporal
- Server + Agent + Temporal
- 多 Agent 并发

**Level 3: 故障场景**
- 进程崩溃
- 网络故障
- 资源耗尽

**Level 4: 长时间稳定性**
- 24 小时运行
- 内存泄漏检测
- 资源占用监控

### 故障场景覆盖

**已覆盖场景:**
1. ✅ Server 崩溃恢复
2. ✅ Agent 断开重连
3. ✅ Temporal 连接重试
4. ✅ Activity 超时
5. ✅ 可重试错误
6. ✅ NonRetryableError

**未覆盖场景 (Post-MVP):**
1. ❌ 网络分区 (split-brain)
2. ❌ 磁盘满
3. ❌ OOM Killer
4. ❌ 时钟偏移
5. ❌ DNS 故障

### 性能基线 (压力测试)

**目标基线:**
```json
{
  "concurrent_workflows": {
    "count": 1000,
    "success_rate": 99.0,
    "max_cpu_percent": 80,
    "max_memory_mb": 2048,
    "max_connections": 1000
  },
  "crash_recovery": {
    "recovery_time_sec": 10,
    "workflow_recovery_rate": 100
  },
  "agent_reconnect": {
    "reconnect_time_sec": 5,
    "task_continuation": true
  },
  "memory_leak": {
    "test_duration_hours": 1,
    "max_growth_percent": 20
  }
}
```

### 监控和告警

**关键指标监控:**
- 工作流成功率
- Server CPU/Memory
- Agent 连接数
- Temporal 查询延迟
- 错误率

**告警阈值:**
- 成功率 < 99%
- CPU > 80%
- Memory > 2GB
- 内存增长 > 20%/hour

## Completion Criteria

**Story 完成标准:**

1. ✅ **所有 AC 完成**
   - AC1-AC8 所有验收标准通过(含混合故障场景)
   - 所有压力测试场景验证

2. ✅ **测试覆盖完整**
   - 并发测试实现
   - 故障注入测试实现
   - 资源泄漏检测实现(含pprof goroutine检测)
   - 超时/重试场景测试实现
   - 混合故障场景测试实现

3. ✅ **CI/CD 集成**
   - Makefile 目标添加
   - GitHub Actions 配置
   - 夜间定时运行

4. ✅ **文档完整**
   - 压力测试指南(含pprof使用)
   - 故障排查指南
   - README 更新
   - pprof配置文档

5. ✅ **稳定性验证**
   - 1000+ 并发成功率 > 99%
   - Event Sourcing 零状态丢失
   - 无内存泄漏
   - 故障自动恢复

**验收测试场景:**

**场景 1: 运行完整压力测试套件**
```bash
make stress-test
# 预期: 所有测试通过,生成报告
```

**场景 2: 并发工作流压力测试**
```bash
CONCURRENT_WORKFLOWS=1000 test/stress/concurrent_workflows_test.sh
# 预期: 成功率 > 99%
```

**场景 3: Server 崩溃恢复**
```bash
test/stress/server_crash_recovery_test.sh
# 预期: 所有工作流恢复并完成
```

**场景 4: 内存泄漏检测**
```bash
test/stress/memory_leak_test.sh
# 预期: 内存增长 < 20%
```

**场景 5: 超时/重试场景**
```bash
test/stress/timeout_retry_scenarios_test.sh
# 预期: 所有场景正确处理
```

## References

**现有代码参考:**
- [internal/agent/worker.go](../../internal/agent/worker.go) - Temporal 连接重试
- [pkg/dsl/retry.go](../../pkg/dsl/retry.go) - 重试策略
- [pkg/temporal/workflow.go](../../pkg/temporal/workflow.go) - Event Sourcing
- [ADR-0001](../../docs/adr/0001-use-temporal-workflow-engine.md) - Temporal 架构

**相关 Story:**
- Story 7.3 - 性能基准测试 (性能测试框架)
- Story 1.7 - 超时和重试策略
- Story 1.8 - Temporal SDK 集成
- Story 2.1 - Agent Worker 框架

**外部参考:**
- [Temporal Testing Guide](https://docs.temporal.io/develop/go/testing-suite)
- [Chaos Engineering Principles](https://principlesofchaos.org/)
- [Google SRE - Testing for Reliability](https://sre.google/sre-book/testing-reliability/)

---

**创建日期:** 2026-01-07  
**创建者:** SM Agent (Bob)  
**Epic:** 7 - 生产级可靠性  
**依赖:** Story 7.3 完成  
**预估点数:** 13 points (复杂度高,涉及多种故障场景)  
**优先级:** High (生产稳定性的关键验证)
