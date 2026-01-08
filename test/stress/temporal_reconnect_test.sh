#!/bin/bash
# Temporal connection retry test
# Tests AC4: Temporal connection retry mechanism

set -e

AGENT_BIN="${AGENT_BIN:-./bin/agent}"

echo "=== Temporal Connection Retry Test ==="
echo "Agent: $AGENT_BIN"
echo ""

RESULTS_DIR="test/stress/results/temporal-retry-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$RESULTS_DIR"

# Create test config with retry settings
TEST_CONFIG="$RESULTS_DIR/config.yaml"
cat > "$TEST_CONFIG" <<EOF
temporal:
  host: localhost:7233
  namespace: default
  max_retries: 5
  retry_interval: 2s
  connection_timeout: 3s

agent:
  task_queues:
    - linux-amd64

log:
  level: debug
  format: json
EOF

# 1. Start Agent without Temporal (should retry)
echo "Starting agent without Temporal server (will retry)..."
timeout 15s $AGENT_BIN --config "$TEST_CONFIG" > "$RESULTS_DIR/agent.log" 2>&1 || true

# 2. Check retry attempts
RETRY_COUNT=$(grep -c "Failed to connect to Temporal\|Retrying connection\|Connection attempt" "$RESULTS_DIR/agent.log" || echo "0")
echo "Retry attempts detected: $RETRY_COUNT"

if [ "$RETRY_COUNT" -lt 2 ]; then
    echo "❌ Expected at least 2 retry attempts, got: $RETRY_COUNT"
    cat "$RESULTS_DIR/agent.log"
    exit 1
fi

echo "✅ Temporal connection retry mechanism verified"
echo "   Retry attempts: $RETRY_COUNT"

# Note: Full test requires actually starting Temporal server
# This test validates retry behavior when Temporal is unavailable
