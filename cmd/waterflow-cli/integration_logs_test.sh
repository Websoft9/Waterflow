#!/bin/bash
# Integration tests for waterflow logs command (Story 5.5)
# Tests AC1-AC7 from sprint plan

set -e  # Exit on error

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Server config
SERVER_URL="${WATERFLOW_SERVER:-http://localhost:8080}"
CLI_BINARY="./bin/waterflow"

# Use existing workflow ID from status tests
TEST_WORKFLOW_ID="${1:-073e7d14-81aa-48f7-bbe2-208d290525a7}"

print_test() {
    echo -e "\n${YELLOW}TEST $1${NC}: $2"
}

pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    ((TESTS_PASSED++))
}

fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    ((TESTS_FAILED++))
}

run_test() {
    ((TESTS_RUN++))
}

cleanup() {
    echo -e "\n${YELLOW}Cleanup...${NC}"
    # No cleanup needed for logs command
}

trap cleanup EXIT

echo "================================================================"
echo "Waterflow CLI - Logs Command Integration Tests (Story 5.5)"
echo "================================================================"
echo "Using workflow ID: $TEST_WORKFLOW_ID"

# Verify CLI binary exists
if [ ! -f "$CLI_BINARY" ]; then
    echo -e "${RED}Error: CLI binary not found at $CLI_BINARY${NC}"
    echo "Run: make build-cli"
    exit 1
fi

# Verify server is running
if ! curl -sf "$SERVER_URL/health" > /dev/null 2>&1; then
    echo -e "${RED}Error: Waterflow server not responding at $SERVER_URL${NC}"
    echo "Start server: docker compose up -d"
    exit 1
fi

echo -e "${GREEN}✓ Server is running at $SERVER_URL${NC}"

#-------------------------------------------------------------------
# AC1: Query workflow logs with default settings
#-------------------------------------------------------------------
print_test "AC1" "Query workflow logs (default: last 100 lines, text format)"
run_test

LOGS_OUTPUT=$(mktemp)
if $CLI_BINARY logs "$TEST_WORKFLOW_ID" > "$LOGS_OUTPUT" 2>&1; then
    if grep -q "Workflow" "$LOGS_OUTPUT"; then
        pass "Logs retrieved successfully"
    else
        fail "Logs output missing expected content"
        cat "$LOGS_OUTPUT"
    fi
else
    fail "Failed to retrieve logs"
    cat "$LOGS_OUTPUT"
fi
rm "$LOGS_OUTPUT"

#-------------------------------------------------------------------
# AC2: Query logs with --tail parameter
#-------------------------------------------------------------------
print_test "AC2" "Query logs with --tail 5"
run_test

LOGS_OUTPUT=$(mktemp)
if $CLI_BINARY logs --tail 5 "$TEST_WORKFLOW_ID" > "$LOGS_OUTPUT" 2>&1; then
    LINE_COUNT=$(wc -l < "$LOGS_OUTPUT")
    if [ "$LINE_COUNT" -le 5 ]; then
        pass "Tail limit applied (got $LINE_COUNT lines)"
    else
        fail "Tail limit not respected (got $LINE_COUNT lines, expected ≤5)"
    fi
else
    fail "Failed to retrieve logs with --tail"
    cat "$LOGS_OUTPUT"
fi
rm "$LOGS_OUTPUT"

#-------------------------------------------------------------------
# AC3: Filter logs by level
#-------------------------------------------------------------------
print_test "AC3" "Filter logs by level (--level info)"
run_test

LOGS_OUTPUT=$(mktemp)
if $CLI_BINARY logs --level info "$TEST_WORKFLOW_ID" > "$LOGS_OUTPUT" 2>&1; then
    if grep -q "INFO" "$LOGS_OUTPUT" || grep -q "No logs found" "$LOGS_OUTPUT"; then
        pass "Level filter applied"
    else
        fail "Level filter output unexpected"
        cat "$LOGS_OUTPUT"
    fi
else
    fail "Failed to filter logs by level"
    cat "$LOGS_OUTPUT"
fi
rm "$LOGS_OUTPUT"

#-------------------------------------------------------------------
# AC4: Filter logs by job
#-------------------------------------------------------------------
print_test "AC4" "Filter logs by job (--job nonexistent)"
run_test

LOGS_OUTPUT=$(mktemp)
if $CLI_BINARY logs --job nonexistent "$TEST_WORKFLOW_ID" > "$LOGS_OUTPUT" 2>&1; then
    if grep -q "No logs found" "$LOGS_OUTPUT"; then
        pass "Job filter applied (no logs for nonexistent job)"
    else
        fail "Job filter output unexpected"
        cat "$LOGS_OUTPUT"
    fi
else
    fail "Failed to filter logs by job"
    cat "$LOGS_OUTPUT"
fi
rm "$LOGS_OUTPUT"

#-------------------------------------------------------------------
# AC5: Output logs in JSON format
#-------------------------------------------------------------------
print_test "AC5" "Output logs in JSON format (--format json)"
run_test

LOGS_OUTPUT=$(mktemp)
if $CLI_BINARY logs --format json "$TEST_WORKFLOW_ID" > "$LOGS_OUTPUT" 2>&1; then
    if echo "$LOGS_OUTPUT" | jq empty 2>/dev/null || grep -q "\"timestamp\"" "$LOGS_OUTPUT"; then
        pass "JSON format output"
    else
        fail "JSON format invalid"
        cat "$LOGS_OUTPUT"
    fi
else
    fail "Failed to output JSON format"
    cat "$LOGS_OUTPUT"
fi
rm "$LOGS_OUTPUT"

#-------------------------------------------------------------------
# AC6: Output logs without timestamps
#-------------------------------------------------------------------
print_test "AC6" "Output logs without timestamps (--no-timestamps)"
run_test

LOGS_OUTPUT=$(mktemp)
if $CLI_BINARY logs --no-timestamps "$TEST_WORKFLOW_ID" > "$LOGS_OUTPUT" 2>&1; then
    if ! grep -q "\[20[0-9][0-9]-" "$LOGS_OUTPUT"; then
        pass "Timestamps hidden"
    else
        fail "Timestamps still present"
        cat "$LOGS_OUTPUT"
    fi
else
    fail "Failed to hide timestamps"
    cat "$LOGS_OUTPUT"
fi
rm "$LOGS_OUTPUT"

#-------------------------------------------------------------------
# AC7: Output logs without colors
#-------------------------------------------------------------------
print_test "AC7" "Output logs without colors (--no-color)"
run_test

LOGS_OUTPUT=$(mktemp)
if $CLI_BINARY logs --no-color "$TEST_WORKFLOW_ID" > "$LOGS_OUTPUT" 2>&1; then
    # Check for ANSI escape codes
    if ! grep -q $'\033\[' "$LOGS_OUTPUT"; then
        pass "No color codes in output"
    else
        fail "Color codes still present"
        cat "$LOGS_OUTPUT"
    fi
else
    fail "Failed to disable colors"
    cat "$LOGS_OUTPUT"
fi
rm "$LOGS_OUTPUT"

#-------------------------------------------------------------------
# Additional Test: Invalid workflow ID
#-------------------------------------------------------------------
print_test "EXTRA" "Query logs for invalid workflow ID"
run_test

LOGS_OUTPUT=$(mktemp)
if $CLI_BINARY logs "invalid-id-12345" > "$LOGS_OUTPUT" 2>&1; then
    fail "Should fail for invalid workflow ID"
else
    if grep -q "Workflow not found" "$LOGS_OUTPUT" || grep -q "404" "$LOGS_OUTPUT"; then
        pass "Proper error for invalid workflow ID"
    else
        fail "Unexpected error message"
        cat "$LOGS_OUTPUT"
    fi
fi
rm "$LOGS_OUTPUT"

#-------------------------------------------------------------------
# Additional Test: Compact format
#-------------------------------------------------------------------
print_test "EXTRA" "Output logs in compact format"
run_test

LOGS_OUTPUT=$(mktemp)
if $CLI_BINARY logs --format compact "$TEST_WORKFLOW_ID" > "$LOGS_OUTPUT" 2>&1; then
    # Compact format should not have timestamps
    if ! grep -q "\[20[0-9][0-9]-" "$LOGS_OUTPUT" && grep -q "Workflow" "$LOGS_OUTPUT"; then
        pass "Compact format output"
    else
        fail "Compact format unexpected"
        cat "$LOGS_OUTPUT"
    fi
else
    fail "Failed to output compact format"
    cat "$LOGS_OUTPUT"
fi
rm "$LOGS_OUTPUT"

#-------------------------------------------------------------------
# Test Summary
#-------------------------------------------------------------------
echo ""
echo "================================================================"
echo "Test Summary"
echo "================================================================"
echo "Total Tests:  $TESTS_RUN"
echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Failed: ${RED}$TESTS_FAILED${NC}"
echo "================================================================"

if [ "$TESTS_FAILED" -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
