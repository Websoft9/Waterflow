#!/bin/bash
# Concurrent workflows stress test
# Tests AC1: 1000+ concurrent workflows with >99% success rate

set -e

CONCURRENT_WORKFLOWS="${CONCURRENT_WORKFLOWS:-1000}"
WORKFLOW_FILE="${WORKFLOW_FILE:-examples/hello-world.yaml}"
SERVER_URL="${SERVER_URL:-http://localhost:8080}"

echo "=== Concurrent Workflows Stress Test ==="
echo "Concurrent Workflows: $CONCURRENT_WORKFLOWS"
echo "Workflow File: $WORKFLOW_FILE"
echo "Server URL: $SERVER_URL"
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

# 使用 xargs 并发提交 (不依赖 GNU parallel)
seq 1 $CONCURRENT_WORKFLOWS | xargs -P 100 -I {} sh -c \
    "curl -s -X POST $SERVER_URL/v1/workflows \
     -H 'Content-Type: application/yaml' \
     --data-binary @$WORKFLOW_FILE \
     -o $RESULTS_DIR/response-{}.json \
     -w '%{http_code}' > $RESULTS_DIR/status-{}.txt"

SUBMIT_END_TIME=$(date +%s)
SUBMIT_DURATION=$((SUBMIT_END_TIME - START_TIME))

echo "All workflows submitted in ${SUBMIT_DURATION}s"
echo ""

# 等待所有工作流完成 (最多 5 分钟)
echo "Waiting for workflows to complete..."
TIMEOUT=300
ELAPSED=0

while [ $ELAPSED -lt $TIMEOUT ]; do
    # 统计状态
    HTTP_200=$(grep -c "200" $RESULTS_DIR/status-*.txt 2>/dev/null || echo "0")
    
    if [ "$HTTP_200" -eq "$CONCURRENT_WORKFLOWS" ]; then
        echo "All workflows submitted successfully!"
        break
    fi
    
    echo "[$ELAPSED s] HTTP 200: $HTTP_200 / $CONCURRENT_WORKFLOWS"
    sleep 10
    ELAPSED=$((ELAPSED + 10))
done

# 停止资源监控
kill $MONITOR_PID 2>/dev/null || true
wait $MONITOR_PID 2>/dev/null || true

END_TIME=$(date +%s)
TOTAL_DURATION=$((END_TIME - START_TIME))

# 分析结果
echo ""
echo "=== Results ==="
SUCCESS=$(grep -c "200" $RESULTS_DIR/status-*.txt 2>/dev/null || echo "0")
FAILED=$((CONCURRENT_WORKFLOWS - SUCCESS))
SUCCESS_RATE=$(echo "scale=2; $SUCCESS * 100 / $CONCURRENT_WORKFLOWS" | bc)

echo "Total Duration: ${TOTAL_DURATION}s"
echo "Success: $SUCCESS"
echo "Failed: $FAILED"
echo "Success Rate: ${SUCCESS_RATE}%"
echo "Throughput: $(echo "scale=2; $CONCURRENT_WORKFLOWS / $TOTAL_DURATION" | bc) workflows/sec"

# 资源使用分析
echo ""
echo "=== Resource Usage ==="
if [ -f "$RESULTS_DIR/cpu.log" ]; then
    MAX_CPU=$(sort -n $RESULTS_DIR/cpu.log | tail -1)
    echo "Max CPU: ${MAX_CPU}%"
else
    echo "CPU data not available"
fi

if [ -f "$RESULTS_DIR/memory.log" ]; then
    MAX_MEM=$(sort -n $RESULTS_DIR/memory.log | tail -1)
    echo "Max Memory: ${MAX_MEM}MB"
else
    echo "Memory data not available"
fi

if [ -f "$RESULTS_DIR/connections.log" ]; then
    MAX_CONNS=$(sort -n $RESULTS_DIR/connections.log | tail -1)
    echo "Max Connections: ${MAX_CONNS}"
else
    echo "Connection data not available"
fi

# 验证 AC1 目标
echo ""
echo "=== Validation ==="
PASSED=0

if (( $(echo "$SUCCESS_RATE >= 99" | bc -l) )); then
    echo "✅ Success rate ${SUCCESS_RATE}% meets target (>= 99%)"
    PASSED=$((PASSED + 1))
else
    echo "❌ Success rate ${SUCCESS_RATE}% below target 99%"
fi

if [ -n "$MAX_CPU" ] && (( $(echo "$MAX_CPU <= 80" | bc -l) )); then
    echo "✅ Max CPU ${MAX_CPU}% within limits (<= 80%)"
    PASSED=$((PASSED + 1))
elif [ -n "$MAX_CPU" ]; then
    echo "⚠️  Max CPU ${MAX_CPU}% exceeds 80% threshold"
fi

if [ -n "$MAX_MEM" ] && (( $(echo "$MAX_MEM <= 2048" | bc -l) )); then
    echo "✅ Max Memory ${MAX_MEM}MB within limits (<= 2GB)"
    PASSED=$((PASSED + 1))
elif [ -n "$MAX_MEM" ]; then
    echo "⚠️  Max Memory ${MAX_MEM}MB exceeds 2GB threshold"
fi

if [ -n "$MAX_CONNS" ] && [ "$MAX_CONNS" -le 1000 ]; then
    echo "✅ Max Connections ${MAX_CONNS} within limits (<= 1000)"
    PASSED=$((PASSED + 1))
elif [ -n "$MAX_CONNS" ]; then
    echo "⚠️  Max Connections ${MAX_CONNS} exceeds 1000 threshold"
fi

AVG_DURATION=$(echo "scale=2; $TOTAL_DURATION / $CONCURRENT_WORKFLOWS" | bc)
if (( $(echo "$AVG_DURATION <= 10" | bc -l) )); then
    echo "✅ Average execution time ${AVG_DURATION}s within limits (<= 10s)"
    PASSED=$((PASSED + 1))
else
    echo "⚠️  Average execution time ${AVG_DURATION}s exceeds 10s"
fi

# 保存报告
cat > "$RESULTS_DIR/report.txt" <<EOF
=== Concurrent Workflows Stress Test Report ===
Date: $(date)
Configuration:
  - Concurrent Workflows: $CONCURRENT_WORKFLOWS
  - Workflow File: $WORKFLOW_FILE
  - Server URL: $SERVER_URL

Results:
  - Total Duration: ${TOTAL_DURATION}s
  - Success: $SUCCESS
  - Failed: $FAILED
  - Success Rate: ${SUCCESS_RATE}%
  - Throughput: $(echo "scale=2; $CONCURRENT_WORKFLOWS / $TOTAL_DURATION" | bc) workflows/sec

Resource Usage:
  - Max CPU: ${MAX_CPU}%
  - Max Memory: ${MAX_MEM}MB
  - Max Connections: ${MAX_CONNS}

Validation:
  - Success Rate >= 99%: $([ $(echo "$SUCCESS_RATE >= 99" | bc -l) -eq 1 ] && echo "PASS" || echo "FAIL")
  - CPU <= 80%: $([ -n "$MAX_CPU" ] && [ $(echo "$MAX_CPU <= 80" | bc -l) -eq 1 ] && echo "PASS" || echo "N/A")
  - Memory <= 2GB: $([ -n "$MAX_MEM" ] && [ $(echo "$MAX_MEM <= 2048" | bc -l) -eq 1 ] && echo "PASS" || echo "N/A")
  - Connections <= 1000: $([ -n "$MAX_CONNS" ] && [ $MAX_CONNS -le 1000 ] && echo "PASS" || echo "N/A")
  - Avg Time <= 10s: $([ $(echo "$AVG_DURATION <= 10" | bc -l) -eq 1 ] && echo "PASS" || echo "FAIL")

Overall: $PASSED / 5 checks passed
EOF

cat "$RESULTS_DIR/report.txt"

# 最终判定
if [ "$PASSED" -lt 2 ]; then
    echo ""
    echo "❌ Concurrent workflows stress test FAILED"
    exit 1
fi

echo ""
echo "✅ Concurrent workflows stress test PASSED"
