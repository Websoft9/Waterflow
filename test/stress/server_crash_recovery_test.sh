#!/bin/bash
# Server crash recovery test
# Tests AC2: Server crash recovery with Event Sourcing

set -e

SERVER_BIN="${SERVER_BIN:-./bin/server}"
SERVER_URL="${SERVER_URL:-http://localhost:8080}"
WORKFLOW_FILE="${WORKFLOW_FILE:-testdata/workflows/long-running.yaml}"

echo "=== Server Crash Recovery Test ==="
echo "Server: $SERVER_BIN"
echo "URL: $SERVER_URL"
echo ""

# 结果目录
RESULTS_DIR="test/stress/results/crash-recovery-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$RESULTS_DIR"

# 1. 启动 Server
echo "Starting server..."
$SERVER_BIN > "$RESULTS_DIR/server.log" 2>&1 &
SERVER_PID=$!
echo "Server PID: $SERVER_PID"

# 等待启动
sleep 5
if ! ps -p $SERVER_PID > /dev/null; then
    echo "❌ Server failed to start"
    cat "$RESULTS_DIR/server.log"
    exit 1
fi

# 2. 提交长时运行的工作流
echo "Submitting long-running workflows..."
WORKFLOW_IDS=()

for i in {1..10}; do
    RESPONSE=$(curl -s -X POST "$SERVER_URL/v1/workflows" \
        -H "Content-Type: application/yaml" \
        --data-binary "@$WORKFLOW_FILE" || echo '{"error": "failed"}')
    
    WORKFLOW_ID=$(echo "$RESPONSE" | jq -r '.workflow_id // .id // empty')
    
    if [ -z "$WORKFLOW_ID" ]; then
        echo "⚠️  Failed to submit workflow $i"
    else
        WORKFLOW_IDS+=("$WORKFLOW_ID")
        echo "Submitted: $WORKFLOW_ID"
    fi
done

SUBMITTED_COUNT=${#WORKFLOW_IDS[@]}
echo "Successfully submitted: $SUBMITTED_COUNT workflows"

if [ $SUBMITTED_COUNT -eq 0 ]; then
    echo "❌ No workflows submitted successfully"
    kill $SERVER_PID 2>/dev/null || true
    exit 1
fi

# 3. 等待工作流开始执行
echo "Waiting for workflows to start executing..."
sleep 10

# 4. 记录崩溃前状态
echo "Recording pre-crash state..."
for WORKFLOW_ID in "${WORKFLOW_IDS[@]}"; do
    STATUS=$(curl -s "$SERVER_URL/v1/workflows/$WORKFLOW_ID" | jq -r '.status // "unknown"')
    echo "$WORKFLOW_ID,$STATUS" >> "$RESULTS_DIR/pre-crash-status.csv"
done

# 5. 崩溃 Server (kill -9)
echo ""
echo "Crashing server with kill -9..."
CRASH_TIME=$(date +%s)
kill -9 $SERVER_PID 2>/dev/null || true
sleep 2

# 6. 重启 Server
echo "Restarting server..."
RECOVERY_START=$CRASH_TIME
$SERVER_BIN > "$RESULTS_DIR/server-recovered.log" 2>&1 &
SERVER_PID=$!
echo "New Server PID: $SERVER_PID"

# 等待启动
sleep 5
if ! ps -p $SERVER_PID > /dev/null; then
    echo "❌ Server failed to restart"
    cat "$RESULTS_DIR/server-recovered.log"
    exit 1
fi

RECOVERY_END=$(date +%s)
RECOVERY_TIME=$((RECOVERY_END - RECOVERY_START))

echo "Server restarted in ${RECOVERY_TIME}s"

# 7. 验证工作流恢复
echo ""
echo "Verifying workflow recovery..."
RECOVERED=0
LOST=0

for WORKFLOW_ID in "${WORKFLOW_IDS[@]}"; do
    # 等待最多 30 秒让工作流恢复
    MAX_WAIT=30
    ELAPSED=0
    RECOVERED_STATUS=""
    
    while [ $ELAPSED -lt $MAX_WAIT ]; do
        STATUS=$(curl -s "$SERVER_URL/v1/workflows/$WORKFLOW_ID" 2>/dev/null | jq -r '.status // empty')
        
        if [ -n "$STATUS" ] && [ "$STATUS" != "null" ]; then
            RECOVERED_STATUS=$STATUS
            break
        fi
        
        sleep 2
        ELAPSED=$((ELAPSED + 2))
    done
    
    if [ -n "$RECOVERED_STATUS" ]; then
        echo "✅ $WORKFLOW_ID: $RECOVERED_STATUS"
        echo "$WORKFLOW_ID,$RECOVERED_STATUS" >> "$RESULTS_DIR/post-crash-status.csv"
        ((RECOVERED++))
    else
        echo "❌ $WORKFLOW_ID: NOT FOUND"
        echo "$WORKFLOW_ID,lost" >> "$RESULTS_DIR/post-crash-status.csv"
        ((LOST++))
    fi
done

# 清理
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

# 结果分析
echo ""
echo "=== Recovery Results ==="
echo "Submitted Workflows: $SUBMITTED_COUNT"
echo "Recovered: $RECOVERED"
echo "Lost: $LOST"
echo "Recovery Time: ${RECOVERY_TIME}s"
echo "Recovery Rate: $(echo "scale=2; $RECOVERED * 100 / $SUBMITTED_COUNT" | bc)%"

# 保存报告
cat > "$RESULTS_DIR/report.txt" <<EOF
=== Server Crash Recovery Test Report ===
Date: $(date)

Configuration:
  - Server: $SERVER_BIN
  - URL: $SERVER_URL
  - Workflow: $WORKFLOW_FILE

Results:
  - Submitted Workflows: $SUBMITTED_COUNT
  - Recovered: $RECOVERED
  - Lost: $LOST
  - Recovery Time: ${RECOVERY_TIME}s
  - Recovery Rate: $(echo "scale=2; $RECOVERED * 100 / $SUBMITTED_COUNT" | bc)%

Validation (AC2):
  - All workflows recovered: $([ $LOST -eq 0 ] && echo "PASS" || echo "FAIL")
  - Recovery time < 10s: $([ $RECOVERY_TIME -lt 10 ] && echo "PASS" || echo "FAIL")
  - No state loss: $([ $RECOVERED -eq $SUBMITTED_COUNT ] && echo "PASS" || echo "FAIL")

Pre-crash status:
$(cat "$RESULTS_DIR/pre-crash-status.csv")

Post-crash status:
$(cat "$RESULTS_DIR/post-crash-status.csv")
EOF

cat "$RESULTS_DIR/report.txt"

# 验证 AC2 目标
echo ""
echo "=== Validation ==="

if [ $RECOVERY_TIME -ge 10 ]; then
    echo "❌ Recovery time ${RECOVERY_TIME}s exceeds 10s limit (AC2)"
    exit 1
fi

if [ $LOST -ne 0 ]; then
    echo "❌ $LOST workflows lost (state loss detected, AC2 failed)"
    exit 1
fi

if [ $RECOVERED -ne $SUBMITTED_COUNT ]; then
    echo "❌ Only $RECOVERED / $SUBMITTED_COUNT workflows recovered"
    exit 1
fi

echo "✅ All workflows recovered successfully"
echo "✅ Recovery time ${RECOVERY_TIME}s within 10s limit"
echo "✅ No state loss detected (Event Sourcing working)"
echo ""
echo "✅ Server crash recovery test PASSED"
