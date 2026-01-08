#!/bin/bash
# Agent reconnect test
# Tests AC3: Agent disconnect/reconnect and task continuation

set -e

AGENT_BIN="${AGENT_BIN:-./bin/agent}"
SERVER_URL="${SERVER_URL:-http://localhost:8080}"
WORKFLOW_FILE="${WORKFLOW_FILE:-testdata/workflows/multi-step.yaml}"

echo "=== Agent Reconnect Test ==="
echo "Agent: $AGENT_BIN"
echo "Server: $SERVER_URL"
echo ""

RESULTS_DIR="test/stress/results/agent-reconnect-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$RESULTS_DIR"

# 1. Start multiple Agents
echo "Starting 3 agents..."
AGENT_PIDS=()
for i in {1..3}; do
    $AGENT_BIN --task-queues linux-amd64 --log-level warn > "$RESULTS_DIR/agent-$i.log" 2>&1 &
    AGENT_PIDS+=($!)
    echo "Started agent $i (PID: ${AGENT_PIDS[$((i-1))]})"
done

sleep 5

# 2. Submit workflow
echo "Submitting workflow..."
RESPONSE=$(curl -s -X POST "$SERVER_URL/v1/workflows" \
    -H "Content-Type: application/yaml" \
    --data-binary "@$WORKFLOW_FILE" || echo '{"error": "failed"}')

WORKFLOW_ID=$(echo "$RESPONSE" | jq -r '.workflow_id // .id // empty')

if [ -z "$WORKFLOW_ID" ]; then
    echo "❌ Failed to submit workflow"
    echo "Response: $RESPONSE"
    for pid in "${AGENT_PIDS[@]}"; do
        kill $pid 2>/dev/null || true
    done
    exit 1
fi

echo "Workflow ID: $WORKFLOW_ID"

# 3. Wait for workflow to start
sleep 5

# 4. Disconnect first Agent
echo "Disconnecting agent 1 (PID: ${AGENT_PIDS[0]})..."
kill ${AGENT_PIDS[0]} 2>/dev/null || true
sleep 2

# 5. Disconnect second Agent
echo "Disconnecting agent 2 (PID: ${AGENT_PIDS[1]})..."
kill ${AGENT_PIDS[1]} 2>/dev/null || true
sleep 2

# 6. Check workflow status (should still be running on agent 3)
STATUS=$(curl -s "$SERVER_URL/v1/workflows/$WORKFLOW_ID" | jq -r '.status // "unknown"')
echo "Workflow status after agent disconnects: $STATUS"

# 7. Restart Agents
echo "Restarting agents..."
$AGENT_BIN --task-queues linux-amd64 --log-level warn > "$RESULTS_DIR/agent-1-restart.log" 2>&1 &
AGENT_PIDS[0]=$!

$AGENT_BIN --task-queues linux-amd64 --log-level warn > "$RESULTS_DIR/agent-2-restart.log" 2>&1 &
AGENT_PIDS[1]=$!

sleep 5

# 8. Wait for workflow completion
echo "Waiting for workflow completion..."
TIMEOUT=60
ELAPSED=0

while [ $ELAPSED -lt $TIMEOUT ]; do
    STATUS=$(curl -s "$SERVER_URL/v1/workflows/$WORKFLOW_ID" | jq -r '.status // "unknown"')
    
    if [ "$STATUS" == "completed" ] || [ "$STATUS" == "succeeded" ]; then
        echo "✅ Workflow completed successfully"
        break
    fi
    
    if [ "$STATUS" == "failed" ]; then
        echo "❌ Workflow failed"
        break
    fi
    
    echo "[$ELAPSED s] Status: $STATUS"
    sleep 5
    ELAPSED=$((ELAPSED + 5))
done

# Cleanup
for pid in "${AGENT_PIDS[@]}"; do
    kill $pid 2>/dev/null || true
done

if [ "$STATUS" != "completed" ] && [ "$STATUS" != "succeeded" ]; then
    echo "❌ Workflow did not complete: $STATUS"
    exit 1
fi

echo "✅ Agent reconnect test passed"
echo "   Final status: $STATUS"
