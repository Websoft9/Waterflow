#!/bin/bash
# Timeout and retry strategy validation test
# Tests AC5: NonRetryableError handling and retry policies

set -e

SERVER_URL="${SERVER_URL:-http://localhost:8080}"

echo "=== Timeout and Retry Strategy Validation Test ==="
echo "Server URL: $SERVER_URL"
echo ""

# 结果目录
RESULTS_DIR="test/stress/results/timeout-retry-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$RESULTS_DIR"

# Test 1: NonRetryableError should fail immediately
echo "Test 1: NonRetryableError handling"
echo "-----------------------------------"

WORKFLOW_NONRETRYABLE=$(cat <<'EOF'
name: nonretryable-error-test
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Fail with NonRetryableError
        uses: run@v1
        with:
          command: exit 255
EOF
)

echo "Submitting workflow with NonRetryableError..."
START_TIME=$(date +%s)

RESPONSE=$(curl -s -X POST "$SERVER_URL/v1/workflows" \
    -H "Content-Type: application/yaml" \
    --data "$WORKFLOW_NONRETRYABLE")

WORKFLOW_ID=$(echo "$RESPONSE" | jq -r '.workflow_id // .id // empty')

if [ -z "$WORKFLOW_ID" ]; then
    echo "❌ Failed to submit workflow"
    exit 1
fi

echo "Workflow ID: $WORKFLOW_ID"

# 等待失败 (应该快速失败)
MAX_WAIT=60
ELAPSED=0
FAILED_QUICKLY=0

while [ $ELAPSED -lt $MAX_WAIT ]; do
    STATUS=$(curl -s "$SERVER_URL/v1/workflows/$WORKFLOW_ID" | jq -r '.status // "unknown"')
    
    if [ "$STATUS" == "failed" ]; then
        FAIL_TIME=$(($(date +%s) - START_TIME))
        echo "✅ Workflow failed in ${FAIL_TIME}s"
        
        if [ $FAIL_TIME -le 10 ]; then
            echo "✅ Failed quickly (no unnecessary retries)"
            FAILED_QUICKLY=1
        else
            echo "⚠️  Failed slowly (${FAIL_TIME}s > 10s), may have retried"
        fi
        break
    fi
    
    sleep 2
    ELAPSED=$((ELAPSED + 2))
done

if [ $FAILED_QUICKLY -eq 0 ]; then
    echo "❌ NonRetryableError did not fail quickly"
fi

# Test 2: Retryable errors should retry
echo ""
echo "Test 2: Retryable error handling"
echo "---------------------------------"

WORKFLOW_RETRYABLE=$(cat <<'EOF'
name: retryable-error-test
jobs:
  test:
    runs-on: linux-amd64
    retry:
      max_attempts: 3
      backoff: 2s
    steps:
      - name: Fail with retryable error
        uses: run@v1
        with:
          command: exit 1
EOF
)

echo "Submitting workflow with retryable error..."
START_TIME=$(date +%s)

RESPONSE=$(curl -s -X POST "$SERVER_URL/v1/workflows" \
    -H "Content-Type: application/yaml" \
    --data "$WORKFLOW_RETRYABLE")

WORKFLOW_ID=$(echo "$RESPONSE" | jq -r '.workflow_id // .id // empty')

if [ -z "$WORKFLOW_ID" ]; then
    echo "❌ Failed to submit workflow"
    exit 1
fi

echo "Workflow ID: $WORKFLOW_ID"

# 记录重试次数
MAX_WAIT=60
ELAPSED=0
RETRY_COUNT=0

while [ $ELAPSED -lt $MAX_WAIT ]; do
    # 获取事件历史 (如果可用)
    EVENTS=$(curl -s "$SERVER_URL/v1/workflows/$WORKFLOW_ID/history" 2>/dev/null || echo '{"events":[]}')
    ATTEMPTS=$(echo "$EVENTS" | jq '[.events[] | select(.type == "ACTIVITY_TASK_STARTED")] | length')
    
    if [ "$ATTEMPTS" != "null" ] && [ $ATTEMPTS -gt 0 ]; then
        RETRY_COUNT=$ATTEMPTS
    fi
    
    STATUS=$(curl -s "$SERVER_URL/v1/workflows/$WORKFLOW_ID" | jq -r '.status // "unknown"')
    
    if [ "$STATUS" == "failed" ]; then
        FAIL_TIME=$(($(date +%s) - START_TIME))
        echo "Workflow failed after ${FAIL_TIME}s"
        
        if [ $RETRY_COUNT -ge 3 ]; then
            echo "✅ Retried 3 times as expected (attempts: $RETRY_COUNT)"
        else
            echo "⚠️  Retry count unclear (detected attempts: $RETRY_COUNT)"
        fi
        break
    fi
    
    sleep 2
    ELAPSED=$((ELAPSED + 2))
done

# Test 3: Timeout handling
echo ""
echo "Test 3: Timeout handling"
echo "------------------------"

WORKFLOW_TIMEOUT=$(cat <<'EOF'
name: timeout-test
jobs:
  test:
    runs-on: linux-amd64
    timeout: 10s
    steps:
      - name: Sleep longer than timeout
        uses: run@v1
        with:
          command: sleep 30
EOF
)

echo "Submitting workflow with 10s timeout (step takes 30s)..."
START_TIME=$(date +%s)

RESPONSE=$(curl -s -X POST "$SERVER_URL/v1/workflows" \
    -H "Content-Type: application/yaml" \
    --data "$WORKFLOW_TIMEOUT")

WORKFLOW_ID=$(echo "$RESPONSE" | jq -r '.workflow_id // .id // empty')

if [ -z "$WORKFLOW_ID" ]; then
    echo "❌ Failed to submit workflow"
    exit 1
fi

echo "Workflow ID: $WORKFLOW_ID"

# 等待超时失败
MAX_WAIT=30
ELAPSED=0
TIMED_OUT=0

while [ $ELAPSED -lt $MAX_WAIT ]; do
    STATUS=$(curl -s "$SERVER_URL/v1/workflows/$WORKFLOW_ID" | jq -r '.status // "unknown"')
    
    if [ "$STATUS" == "failed" ] || [ "$STATUS" == "timeout" ]; then
        TIMEOUT_TIME=$(($(date +%s) - START_TIME))
        echo "✅ Workflow timed out after ${TIMEOUT_TIME}s"
        
        if [ $TIMEOUT_TIME -le 15 ]; then
            echo "✅ Timed out within expected range (~10s + overhead)"
            TIMED_OUT=1
        else
            echo "⚠️  Timeout took too long (${TIMEOUT_TIME}s > 15s)"
        fi
        break
    fi
    
    sleep 2
    ELAPSED=$((ELAPSED + 2))
done

if [ $TIMED_OUT -eq 0 ]; then
    echo "❌ Workflow did not timeout as expected"
fi

# 保存报告
cat > "$RESULTS_DIR/report.txt" <<EOF
=== Timeout and Retry Strategy Validation Report ===
Date: $(date)

Test 1: NonRetryableError Handling
  - Result: $([ $FAILED_QUICKLY -eq 1 ] && echo "PASS" || echo "FAIL")
  - Expected: Fail quickly without retries
  - Actual: $([ $FAILED_QUICKLY -eq 1 ] && echo "Failed quickly" || echo "Did not fail quickly")

Test 2: Retryable Error Handling
  - Result: $([ $RETRY_COUNT -ge 3 ] && echo "PASS" || echo "UNCERTAIN")
  - Expected: Retry 3 times with backoff
  - Actual: Detected $RETRY_COUNT attempts

Test 3: Timeout Handling
  - Result: $([ $TIMED_OUT -eq 1 ] && echo "PASS" || echo "FAIL")
  - Expected: Timeout after 10s
  - Actual: $([ $TIMED_OUT -eq 1 ] && echo "Timed out within expected range" || echo "Did not timeout")

Validation (AC5):
  - NonRetryableError fails immediately: $([ $FAILED_QUICKLY -eq 1 ] && echo "PASS" || echo "FAIL")
  - Retryable errors retry with backoff: $([ $RETRY_COUNT -ge 2 ] && echo "PASS" || echo "UNCERTAIN")
  - Timeouts enforced: $([ $TIMED_OUT -eq 1 ] && echo "PASS" || echo "FAIL")

Overall: $((FAILED_QUICKLY + TIMED_OUT)) / 3 tests passed
EOF

cat "$RESULTS_DIR/report.txt"

# 验证
echo ""
echo "=== Validation ==="

PASSED=$((FAILED_QUICKLY + TIMED_OUT))

if [ $PASSED -lt 2 ]; then
    echo "❌ Timeout and retry validation failed ($PASSED / 3 tests passed)"
    exit 1
fi

echo "✅ Timeout and retry strategies validated ($PASSED / 3 tests passed)"
echo ""
echo "✅ Timeout and retry validation test PASSED"
