#!/bin/bash
# Resource leak detection test
# Tests AC3: No memory/goroutine/connection leaks

set -e

SERVER_BIN="${SERVER_BIN:-./bin/server}"
SERVER_URL="${SERVER_URL:-http://localhost:8080}"
WORKFLOW_FILE="${WORKFLOW_FILE:-examples/hello-world.yaml}"

echo "=== Resource Leak Detection Test ==="
echo "Server: $SERVER_BIN"
echo "Duration: 10 minutes of continuous operation"
echo ""

# 结果目录
RESULTS_DIR="test/stress/results/leak-detection-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$RESULTS_DIR"

# 1. 启动 Server
echo "Starting server..."
$SERVER_BIN > "$RESULTS_DIR/server.log" 2>&1 &
SERVER_PID=$!
echo "Server PID: $SERVER_PID"
sleep 5

if ! ps -p $SERVER_PID > /dev/null; then
    echo "❌ Server failed to start"
    cat "$RESULTS_DIR/server.log"
    exit 1
fi

# 2. 启动资源监控
echo "Starting resource monitoring..."
./scripts/monitor_resources.sh "$RESULTS_DIR" &
MONITOR_PID=$!

# 3. 持续 10 分钟提交工作流
echo "Submitting workflows continuously for 10 minutes..."
START_TIME=$(date +%s)
DURATION=600  # 10 minutes
SUBMITTED=0

while [ $(($(date +%s) - START_TIME)) -lt $DURATION ]; do
    curl -s -X POST "$SERVER_URL/v1/workflows" \
        -H "Content-Type: application/yaml" \
        --data-binary "@$WORKFLOW_FILE" \
        > /dev/null 2>&1 || true
    
    ((SUBMITTED++))
    
    # 每 60 秒报告进度
    ELAPSED=$(($(date +%s) - START_TIME))
    if [ $((ELAPSED % 60)) -eq 0 ]; then
        echo "[$ELAPSED s] Submitted $SUBMITTED workflows"
    fi
    
    sleep 1
done

echo "Submitted $SUBMITTED workflows total"

# 停止资源监控
kill $MONITOR_PID 2>/dev/null || true
wait $MONITOR_PID 2>/dev/null || true

# 4. 分析资源使用趋势
echo ""
echo "=== Resource Usage Analysis ==="

# 内存泄漏检测
if [ -f "$RESULTS_DIR/memory.log" ]; then
    INITIAL_MEM=$(head -10 "$RESULTS_DIR/memory.log" | awk '{sum+=$1} END {print sum/NR}')
    FINAL_MEM=$(tail -10 "$RESULTS_DIR/memory.log" | awk '{sum+=$1} END {print sum/NR}')
    MEM_GROWTH=$(echo "scale=2; ($FINAL_MEM - $INITIAL_MEM) / $INITIAL_MEM * 100" | bc)
    MAX_MEM=$(sort -n "$RESULTS_DIR/memory.log" | tail -1)
    
    echo "Memory:"
    echo "  Initial (avg): ${INITIAL_MEM}MB"
    echo "  Final (avg): ${FINAL_MEM}MB"
    echo "  Growth: ${MEM_GROWTH}%"
    echo "  Max: ${MAX_MEM}MB"
    
    # 判断内存泄漏 (10分钟内增长不应超过50%)
    if (( $(echo "$MEM_GROWTH > 50" | bc -l) )); then
        echo "  ⚠️  Memory leak suspected (growth ${MEM_GROWTH}%)"
        LEAK_DETECTED=1
    else
        echo "  ✅ No memory leak detected"
    fi
else
    echo "Memory data not available"
fi

# Goroutine 泄漏检测 (如果有 pprof 数据)
if [ -f "$RESULTS_DIR/goroutines.log" ]; then
    INITIAL_GOROUTINES=$(head -10 "$RESULTS_DIR/goroutines.log" | awk '{sum+=$1} END {print sum/NR}')
    FINAL_GOROUTINES=$(tail -10 "$RESULTS_DIR/goroutines.log" | awk '{sum+=$1} END {print sum/NR}')
    GOROUTINE_GROWTH=$(echo "scale=2; ($FINAL_GOROUTINES - $INITIAL_GOROUTINES) / $INITIAL_GOROUTINES * 100" | bc)
    
    echo ""
    echo "Goroutines:"
    echo "  Initial (avg): ${INITIAL_GOROUTINES}"
    echo "  Final (avg): ${FINAL_GOROUTINES}"
    echo "  Growth: ${GOROUTINE_GROWTH}%"
    
    if (( $(echo "$GOROUTINE_GROWTH > 100" | bc -l) )); then
        echo "  ⚠️  Goroutine leak suspected (growth ${GOROUTINE_GROWTH}%)"
        LEAK_DETECTED=1
    else
        echo "  ✅ No goroutine leak detected"
    fi
fi

# 连接泄漏检测
if [ -f "$RESULTS_DIR/connections.log" ]; then
    INITIAL_CONNS=$(head -10 "$RESULTS_DIR/connections.log" | awk '{sum+=$1} END {print sum/NR}')
    FINAL_CONNS=$(tail -10 "$RESULTS_DIR/connections.log" | awk '{sum+=$1} END {print sum/NR}')
    CONN_GROWTH=$(echo "scale=2; $FINAL_CONNS - $INITIAL_CONNS" | bc)
    MAX_CONNS=$(sort -n "$RESULTS_DIR/connections.log" | tail -1)
    
    echo ""
    echo "Connections:"
    echo "  Initial (avg): ${INITIAL_CONNS}"
    echo "  Final (avg): ${FINAL_CONNS}"
    echo "  Growth: ${CONN_GROWTH}"
    echo "  Max: ${MAX_CONNS}"
    
    if (( $(echo "$CONN_GROWTH > 100" | bc -l) )); then
        echo "  ⚠️  Connection leak suspected (growth ${CONN_GROWTH})"
        LEAK_DETECTED=1
    else
        echo "  ✅ No connection leak detected"
    fi
fi

# 清理
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

# 保存报告
cat > "$RESULTS_DIR/report.txt" <<EOF
=== Resource Leak Detection Test Report ===
Date: $(date)

Configuration:
  - Server: $SERVER_BIN
  - Duration: 10 minutes
  - Workflows Submitted: $SUBMITTED

Memory Analysis:
  - Initial: ${INITIAL_MEM}MB
  - Final: ${FINAL_MEM}MB
  - Growth: ${MEM_GROWTH}%
  - Max: ${MAX_MEM}MB
  - Status: $([ -z "$LEAK_DETECTED" ] && echo "No leak" || echo "Leak suspected")

Goroutine Analysis:
$([ -f "$RESULTS_DIR/goroutines.log" ] && cat <<GOROUTINES
  - Initial: ${INITIAL_GOROUTINES}
  - Final: ${FINAL_GOROUTINES}
  - Growth: ${GOROUTINE_GROWTH}%
GOROUTINES
|| echo "  - Data not available")

Connection Analysis:
  - Initial: ${INITIAL_CONNS}
  - Final: ${FINAL_CONNS}
  - Growth: ${CONN_GROWTH}
  - Max: ${MAX_CONNS}

Validation (AC3):
  - Memory growth < 50%: $([ $(echo "$MEM_GROWTH < 50" | bc -l) -eq 1 ] && echo "PASS" || echo "FAIL")
  - Goroutine stable: $([ -z "$GOROUTINE_GROWTH" ] && echo "N/A" || ([ $(echo "$GOROUTINE_GROWTH < 100" | bc -l) -eq 1 ] && echo "PASS" || echo "FAIL"))
  - Connection stable: $([ $(echo "$CONN_GROWTH < 100" | bc -l) -eq 1 ] && echo "PASS" || echo "FAIL")

Overall: $([ -z "$LEAK_DETECTED" ] && echo "PASS" || echo "FAIL")
EOF

cat "$RESULTS_DIR/report.txt"

# 验证
echo ""
echo "=== Validation ==="

if [ -n "$LEAK_DETECTED" ]; then
    echo "❌ Resource leaks detected (AC3 failed)"
    echo ""
    echo "Recommendations:"
    echo "1. Run pprof to identify leak sources:"
    echo "   go tool pprof http://localhost:8080/debug/pprof/heap"
    echo "2. Check for unclosed database connections"
    echo "3. Check for goroutine leaks with:"
    echo "   go tool pprof http://localhost:8080/debug/pprof/goroutine"
    exit 1
fi

echo "✅ No resource leaks detected"
echo "✅ Memory growth ${MEM_GROWTH}% within acceptable range"
echo "✅ System stable over 10-minute stress period"
echo ""
echo "✅ Resource leak detection test PASSED"
