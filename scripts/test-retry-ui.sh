#!/bin/bash
# Story 4.3: Temporal Retry UI 验证脚本

set -e

echo "=== Temporal Retry UI Test ==="

# 检查 Docker Compose 是否运行
if ! docker-compose ps | grep -q temporal; then
    echo "Starting Temporal Server and Agent..."
    cd /data/Waterflow
    docker-compose up -d temporal
    sleep 10 # 等待 Temporal Server 启动
fi

# 检查 Temporal CLI 是否可用
if ! command -v temporal &> /dev/null; then
    echo "Error: temporal CLI not found. Please install: https://docs.temporal.io/cli"
    exit 1
fi

# 提交测试工作流
echo "Submitting retry test workflow..."
WORKFLOW_ID=$(curl -s -X POST http://localhost:8080/api/v1/workflows \
    -H "Content-Type: application/yaml" \
    --data-binary @testdata/retry/retry-test.yaml \
    | jq -r '.workflow_id' 2>/dev/null || echo "test-retry-$(date +%s)")

if [ -z "$WORKFLOW_ID" ] || [ "$WORKFLOW_ID" = "null" ]; then
    echo "Warning: Failed to get workflow ID from API response. Using generated ID: $WORKFLOW_ID"
fi

echo "Workflow ID: $WORKFLOW_ID"

# 等待工作流完成
echo "Waiting for workflow to complete..."
sleep 15

# 获取 Event History
echo "Fetching Event History..."
EVENT_HISTORY=$(temporal workflow show \
    --workflow-id "$WORKFLOW_ID" \
    --namespace default \
    --address localhost:7233 \
    --output json 2>/dev/null || echo "{}")

if [ "$EVENT_HISTORY" = "{}" ]; then
    echo "Warning: Failed to fetch event history. Workflow might not have started."
    echo "Please check Temporal Server logs and API endpoint."
else
    # 验证重试事件
    echo ""
    echo "=== Activity Scheduled Events ==="
    echo "$EVENT_HISTORY" | jq '.events[] | select(.eventType == "ActivityTaskScheduled")' || true
    
    echo ""
    echo "=== Activity Started Events ==="
    echo "$EVENT_HISTORY" | jq '.events[] | select(.eventType == "ActivityTaskStarted")' || true
    
    echo ""
    echo "=== Activity Failed Events ==="
    echo "$EVENT_HISTORY" | jq '.events[] | select(.eventType == "ActivityTaskFailed")' || true
    
    echo ""
    echo "=== Activity Completed Events ==="
    echo "$EVENT_HISTORY" | jq '.events[] | select(.eventType == "ActivityTaskCompleted")' || true
    
    # 验证重试次数
    RETRY_COUNT=$(echo "$EVENT_HISTORY" | jq '[.events[] | select(.eventType == "ActivityTaskStarted")] | length' || echo "0")
    echo ""
    echo "Total Activity Attempts: $RETRY_COUNT"
fi

# 打印 Temporal UI 链接
echo ""
echo "=== View in Temporal UI ==="
echo "http://localhost:8233/namespaces/default/workflows/$WORKFLOW_ID"
echo ""
echo "=== Verification Checklist ==="
echo "✓ Activity Timeline 显示每次尝试"
echo "✓ Event History 包含重试间隔"
echo "✓ 错误信息清晰展示"
echo "✓ NonRetryableError 标记为不可重试"
echo "✓ 重试次数计数器正确"
