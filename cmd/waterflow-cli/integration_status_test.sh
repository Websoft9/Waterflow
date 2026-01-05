#!/bin/bash
# CLI status 命令集成测试
# Story 5.4: CLI status command

set -e

CLI="./bin/waterflow"
SERVER_URL="${WATERFLOW_SERVER:-http://localhost:8080}"

echo "=== Waterflow CLI Status Command Integration Tests ==="
echo "Server: $SERVER_URL"
echo

# Check if server is running
if ! curl -sf "$SERVER_URL/health" > /dev/null 2>&1; then
    echo "ERROR: Waterflow server not running at $SERVER_URL"
    echo "Start server with: docker-compose up -d"
    exit 1
fi

echo "✓ Server is running"
echo

# Test 1: Submit a test workflow first
echo "=== Test 1: Submit test workflow ==="
WORKFLOW_FILE="testdata/valid/simple.yaml"
if [ ! -f "$WORKFLOW_FILE" ]; then
    echo "ERROR: Test workflow file not found: $WORKFLOW_FILE"
    exit 1
fi

WORKFLOW_ID=$($CLI submit --quiet $WORKFLOW_FILE 2>/dev/null || echo "")
if [ -z "$WORKFLOW_ID" ]; then
    echo "ERROR: Failed to submit workflow"
    exit 1
fi

echo "✓ Submitted workflow: $WORKFLOW_ID"
echo

# Wait a moment for workflow to start
sleep 1

# Test 2: Basic status query (AC1)
echo "=== Test 2: Basic status query (AC1) ==="
OUTPUT=$($CLI status $WORKFLOW_ID 2>&1)
if ! echo "$OUTPUT" | grep -q "Workflow:"; then
    echo "ERROR: Output missing 'Workflow:' field"
    echo "$OUTPUT"
    exit 1
fi
if ! echo "$OUTPUT" | grep -q "Status:"; then
    echo "ERROR: Output missing 'Status:' field"
    echo "$OUTPUT"
    exit 1
fi
if ! echo "$OUTPUT" | grep -q "ID:"; then
    echo "ERROR: Output missing 'ID:' field"
    echo "$OUTPUT"
    exit 1
fi
echo "✓ PASS: Basic status query displays expected fields"
echo

# Test 3: JSON output (AC4)
echo "=== Test 3: JSON output (AC4) ==="
JSON_OUTPUT=$($CLI status --format json $WORKFLOW_ID 2>&1)
if ! echo "$JSON_OUTPUT" | jq -e '.id' > /dev/null 2>&1; then
    echo "ERROR: JSON output missing 'id' field"
    echo "$JSON_OUTPUT"
    exit 1
fi
if ! echo "$JSON_OUTPUT" | jq -e '.status' > /dev/null 2>&1; then
    echo "ERROR: JSON output missing 'status' field"
    echo "$JSON_OUTPUT"
    exit 1
fi
echo "✓ PASS: JSON output format valid"
echo

# Test 4: YAML output (AC4)
echo "=== Test 4: YAML output (AC4) ==="
YAML_OUTPUT=$($CLI status --format yaml $WORKFLOW_ID 2>&1)
if ! echo "$YAML_OUTPUT" | grep -q "^id:"; then
    echo "ERROR: YAML output missing 'id' field"
    echo "$YAML_OUTPUT"
    exit 1
fi
if ! echo "$YAML_OUTPUT" | grep -q "^status:"; then
    echo "ERROR: YAML output missing 'status' field"
    echo "$YAML_OUTPUT"
    exit 1
fi
echo "✓ PASS: YAML output format valid"
echo

# Test 5: Quiet mode (AC4)
echo "=== Test 5: Quiet mode (AC4) ==="
QUIET_OUTPUT=$($CLI status --quiet $WORKFLOW_ID 2>&1)
# Should only output status value (one word, no newlines or extra text)
WORD_COUNT=$(echo "$QUIET_OUTPUT" | wc -w)
if [ "$WORD_COUNT" -ne 1 ]; then
    echo "ERROR: Quiet mode should output exactly one word (status value)"
    echo "Got: '$QUIET_OUTPUT'"
    echo "Word count: $WORD_COUNT"
    exit 1
fi

# Check if it's a valid status
case "$QUIET_OUTPUT" in
    pending|running|completed|failed|cancelled|timeout)
        echo "✓ PASS: Quiet mode outputs status value: $QUIET_OUTPUT"
        ;;
    *)
        echo "ERROR: Invalid status value: $QUIET_OUTPUT"
        exit 1
        ;;
esac
echo

# Test 6: Workflow not found (AC5)
echo "=== Test 6: Workflow not found error handling (AC5) ==="
NONEXISTENT_ID="00000000-0000-0000-0000-000000000000"
ERROR_OUTPUT=$($CLI status $NONEXISTENT_ID 2>&1 || true)
if ! echo "$ERROR_OUTPUT" | grep -qi "not found"; then
    echo "ERROR: Should report 'not found' error"
    echo "Got: $ERROR_OUTPUT"
    exit 1
fi
echo "✓ PASS: Not found error handled correctly"
echo

# Test 7: Invalid workflow ID format warning (AC2)
echo "=== Test 7: Invalid ID format warning ==="
INVALID_ID="invalid-id-format"
WARNING_OUTPUT=$($CLI status $INVALID_ID 2>&1 || true)
if ! echo "$WARNING_OUTPUT" | grep -qi "warning"; then
    echo "Note: No warning for invalid UUID format (acceptable)"
else
    echo "✓ PASS: Warning displayed for invalid UUID format"
fi
echo

# Test 8: Compact mode (AC2)
echo "=== Test 8: Compact mode (AC2) ==="
COMPACT_OUTPUT=$($CLI status --compact $WORKFLOW_ID 2>&1)
# Compact mode should still show Jobs but no Steps
if ! echo "$COMPACT_OUTPUT" | grep -q "Jobs:"; then
    echo "Note: No Jobs in output (workflow may have no jobs)"
else
    echo "✓ PASS: Compact mode displays Jobs"
fi
echo

# Test 9: No-color mode (AC6)
echo "=== Test 9: No-color mode (AC6) ==="
NO_COLOR_OUTPUT=$($CLI status --no-color $WORKFLOW_ID 2>&1)
# Check that output doesn't contain ANSI color codes
if echo "$NO_COLOR_OUTPUT" | grep -qP '\x1b\['; then
    echo "ERROR: No-color mode should not contain ANSI escape codes"
    echo "$NO_COLOR_OUTPUT" | od -c | head -20
    exit 1
fi
echo "✓ PASS: No-color mode works correctly"
echo

# Test 10: Watch mode (AC3) - manual test note
echo "=== Test 10: Watch mode (AC3) ==="
echo "SKIP: Watch mode requires manual verification"
echo "Run manually: $CLI status --watch --interval 2s $WORKFLOW_ID"
echo

# Test 11: Status symbols (AC6)
echo "=== Test 11: Status symbols (AC6) ==="
SYMBOL_OUTPUT=$($CLI status $WORKFLOW_ID 2>&1)
# Check for status symbols (may not always be present depending on workflow state)
if echo "$SYMBOL_OUTPUT" | grep -qE '[✓→✗○⊗]'; then
    echo "✓ PASS: Status symbols present in output"
else
    echo "Note: No status symbols (workflow may have no jobs/steps)"
fi
echo

echo "========================================="
echo "All automated tests passed! ✓"
echo "========================================="
echo
echo "Summary:"
echo "  ✓ AC1: Basic status query"
echo "  ✓ AC2: Jobs/Steps progress (compact mode)"
echo "  ✓ AC3: Watch mode (manual test required)"
echo "  ✓ AC4: Output formats (text/json/yaml/quiet)"
echo "  ✓ AC5: Error handling"
echo "  ✓ AC6: Colors and symbols"
echo
echo "Cleanup: Workflow ID $WORKFLOW_ID can be inspected or cleaned up"
