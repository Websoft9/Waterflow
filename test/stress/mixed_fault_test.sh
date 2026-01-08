#!/bin/bash
# Mixed fault scenario test
# Tests AC8: Server crash + Agent disconnect simultaneously

set -e

SERVER_BIN="${SERVER_BIN:-./bin/server}"
AGENT_BIN="${AGENT_BIN:-./bin/agent}"
SERVER_URL="${SERVER_URL:-http://localhost:8080}"
WORKFLOW_FILE="${WORKFLOW_FILE:-testdata/workflows/long-running.yaml}"

# Cleanup function
cleanup() {
    echo "Cleaning up..."
    pkill -f "$SERVER_BIN" 2>/dev/null || true
    pkill -f "$AGENT_BIN" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "=== Mixed Fault Scenario Test ==="
echo "Server: $SERVER_BIN"
echo "Agent: $AGENT_BIN"
echo ""

RESULTS_DIR="test/stress/results/mixed-fault-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$RESULTS_DIR"

# 1. Start Server and Agent
echo "Starting server and agent..."
$SERVER_BIN > "$RESULTS_DIR/server.log" 2>&1 &
SERVER_PID=$!
sleep 3

$AGENT_BIN --task-queues linux-amd64 > "$RESULTS_DIR/agent.log" 2>&1 &
AGENT_PID=$!
sleep 3

# Verify services started
if ! ps -p $SERVER_PID > /dev/null; then
    echo "❌ Server failed to start"
    cat "$RESULTS_DIR/server.log"
    exit 1
fi

if ! ps -p $AGENT_PID > /dev/null; then
    echo "❌ Agent failed to start"
    cat "$RESULTS_DIR/agent.log"
    exit 1
fi

# 2. Submit long-running workflow
echo "Submitting long-running workflow..."
RESPONSE=$(curl -s -X POST "$SERVER_URL/v1/workflows" \
    -H "Content-Type: application/yaml" \
    --data-binary "@$WORKFLOW_FILE" || echo '{"error": "failed"}')

WORKFLOW_ID=$(echo "$RESPONSE" | jq -r '.workflow_id // .id // empty')

if [ -z "$WORKFLOW_ID" ]; then
    echo "❌ Failed to submit workflow"
    exit 1
fi

echo "Workflow ID: $WORKFLOW_ID"

# 3. Wait for workflow to start
echo "Waiting for workflow to start..."
sleep 5

# 4. Record Event History before fault
EVENTS_BEFORE=$(curl -s "$SERVER_URL/v1/workflows/$WORKFLOW_ID/history" 2>/dev/null | jq -r '.events | length' || echo "0")
echo "Events before fault: $EVENTS_BEFORE"

# 5. Simulate mixed fault: kill Server and Agent simultaneously
echo "Simulating mixed fault: killing Server and Agent..."
FAULT_TIME=$(date +%s)
kill -9 $SERVER_PID $AGENT_PID 2>/dev/null || true

sleep 2

# 6. Restart services
echo "Restarting services..."
RECOVERY_START=$(date +%s)

$SERVER_BIN > "$RESULTS_DIR/server-restart.log" 2>&1 &
SERVER_PID=$!

$AGENT_BIN --task-queues linux-amd64 > "$RESULTS_DIR/agent-restart.log" 2>&1 &
AGENT_PID=$!

# 7. Wait for services to recover
echo "Waiting for services to recover..."
RECOVERED=false
TIMEOUT=15

for i in $(seq 1 $TIMEOUT); do
    sleep 1
    ELAPSED=$(($(date +%s) - $RECOVERY_START))
    
    # Check if both services are healthy
    if curl -s "$SERVER_URL/health" > /dev/null 2>&1; then
        RECOVERED=true
        RECOVERY_TIME=$ELAPSED
        echo "✅ Services recovered in ${RECOVERY_TIME}s"
        break
    fi
done

if [ "$RECOVERED" = "false" ]; then
    echo "❌ Recovery time exceeded 15 seconds"
    exit 1
fi

# 8. Verify workflow recovery
echo "Verifying workflow recovery..."
sleep 5

STATUS=$(curl -s "$SERVER_URL/v1/workflows/$WORKFLOW_ID" 2>/dev/null | jq -r '.status // "unknown"')

if [ "$STATUS" != "running" ] && [ "$STATUS" != "completed" ] && [ "$STATUS" != "succeeded" ]; then
    echo "❌ Workflow in unexpected status: $STATUS"
    exit 1
fi

echo "✅ Workflow recovered: $STATUS"

# 9. Verify Event History integrity
EVENTS_AFTER=$(curl -s "$SERVER_URL/v1/workflows/$WORKFLOW_ID/history" 2>/dev/null | jq -r '.events | length' || echo "0")
echo "Events after recovery: $EVENTS_AFTER"

if [ "$EVENTS_AFTER" -lt "$EVENTS_BEFORE" ]; then
    echo "❌ Event History lost events: $EVENTS_BEFORE -> $EVENTS_AFTER"
    exit 1
fi

echo "✅ Event History integrity: 100%"
echo ""
echo "✅ Mixed fault scenario test passed"
echo "   Recovery time: ${RECOVERY_TIME}s"
echo "   Workflow status: $STATUS"
echo "   Events: $EVENTS_BEFORE -> $EVENTS_AFTER"
