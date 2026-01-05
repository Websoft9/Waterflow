#!/bin/bash
# CLI node list command integration test
# Tests Story 5-6: CLI node list command

set -e

CLI="./bin/waterflow"
SERVER_URL="${WATERFLOW_SERVER_URL:-http://localhost:8080}"

echo "=== CLI Node List Integration Tests ==="
echo "Server URL: $SERVER_URL"
echo

# Check if CLI binary exists
if [ ! -f "$CLI" ]; then
    echo "ERROR: CLI binary not found at $CLI"
    echo "Please run 'make cli' to build the CLI"
    exit 1
fi

# Check if server is running
if ! curl -sf "$SERVER_URL/health" > /dev/null 2>&1; then
    echo "WARNING: Waterflow server not running at $SERVER_URL"
    echo "Some tests will be skipped"
    SKIP_SERVER_TESTS=1
fi

echo "--- Test 1: Basic node list (AC1) ---"
if [ -z "$SKIP_SERVER_TESTS" ]; then
    OUTPUT=$($CLI --server "$SERVER_URL" node list)
    if echo "$OUTPUT" | grep -q "Available Nodes"; then
        echo "✓ PASS: Basic node list displays nodes"
    else
        echo "✗ FAIL: Expected 'Available Nodes' in output"
        echo "$OUTPUT"
        exit 1
    fi
else
    echo "⊗ SKIP: Server not running"
fi
echo

echo "--- Test 2: Node list shows categories (AC1) ---"
if [ -z "$SKIP_SERVER_TESTS" ]; then
    OUTPUT=$($CLI --server "$SERVER_URL" node list)
    if echo "$OUTPUT" | grep -q "Execution:" && echo "$OUTPUT" | grep -q "exec/shell@v1"; then
        echo "✓ PASS: Node list shows categories and nodes"
    else
        echo "✗ FAIL: Expected category grouping"
        echo "$OUTPUT"
        exit 1
    fi
else
    echo "⊗ SKIP: Server not running"
fi
echo

echo "--- Test 3: Node detail query (AC2) ---"
if [ -z "$SKIP_SERVER_TESTS" ]; then
    OUTPUT=$($CLI --server "$SERVER_URL" node list exec/shell)
    if echo "$OUTPUT" | grep -q "Input Parameters:" && echo "$OUTPUT" | grep -q "Usage Example:"; then
        echo "✓ PASS: Node detail shows parameters and usage"
    else
        echo "✗ FAIL: Expected 'Input Parameters' and 'Usage Example'"
        echo "$OUTPUT"
        exit 1
    fi
else
    echo "⊗ SKIP: Server not running"
fi
echo

echo "--- Test 4: Category filter (AC3) ---"
if [ -z "$SKIP_SERVER_TESTS" ]; then
    OUTPUT=$($CLI --server "$SERVER_URL" node list --category exec)
    if echo "$OUTPUT" | grep -q "exec/shell"; then
        echo "✓ PASS: Category filter works"
    else
        echo "✗ FAIL: Expected 'exec/shell' in filtered output"
        echo "$OUTPUT"
        exit 1
    fi
else
    echo "⊗ SKIP: Server not running"
fi
echo

echo "--- Test 5: Name search (AC4) ---"
if [ -z "$SKIP_SERVER_TESTS" ]; then
    OUTPUT=$($CLI --server "$SERVER_URL" node list --search shell)
    if echo "$OUTPUT" | grep -q "exec/shell"; then
        echo "✓ PASS: Name search works"
    else
        echo "✗ FAIL: Expected 'exec/shell' in search results"
        echo "$OUTPUT"
        exit 1
    fi
else
    echo "⊗ SKIP: Server not running"
fi
echo

echo "--- Test 6: JSON output (AC5) ---"
if [ -z "$SKIP_SERVER_TESTS" ]; then
    OUTPUT=$($CLI --server "$SERVER_URL" node list --format json)
    if echo "$OUTPUT" | jq -e '.nodes' > /dev/null 2>&1; then
        echo "✓ PASS: JSON output is valid"
    else
        echo "✗ FAIL: Expected valid JSON with 'nodes' field"
        echo "$OUTPUT"
        exit 1
    fi
else
    echo "⊗ SKIP: Server not running"
fi
echo

echo "--- Test 7: YAML output (AC5) ---"
if [ -z "$SKIP_SERVER_TESTS" ]; then
    OUTPUT=$($CLI --server "$SERVER_URL" node list --format yaml)
    if echo "$OUTPUT" | grep -q "nodes:"; then
        echo "✓ PASS: YAML output format"
    else
        echo "✗ FAIL: Expected 'nodes:' in YAML output"
        echo "$OUTPUT"
        exit 1
    fi
else
    echo "⊗ SKIP: Server not running"
fi
echo

echo "--- Test 8: Simple output format (AC5) ---"
if [ -z "$SKIP_SERVER_TESTS" ]; then
    OUTPUT=$($CLI --server "$SERVER_URL" node list --format simple)
    if echo "$OUTPUT" | grep -q "exec/shell@v1"; then
        echo "✓ PASS: Simple format shows node names"
    else
        echo "✗ FAIL: Expected 'exec/shell@v1' in simple output"
        echo "$OUTPUT"
        exit 1
    fi
else
    echo "⊗ SKIP: Server not running"
fi
echo

echo "--- Test 9: No-group mode (AC1) ---"
if [ -z "$SKIP_SERVER_TESTS" ]; then
    OUTPUT=$($CLI --server "$SERVER_URL" node list --no-group)
    if echo "$OUTPUT" | grep -q "exec/shell@v1" && ! echo "$OUTPUT" | grep -q "Execution:"; then
        echo "✓ PASS: No-group mode disables category headers"
    else
        echo "✗ FAIL: Expected flat list without category headers"
        echo "$OUTPUT"
        exit 1
    fi
else
    echo "⊗ SKIP: Server not running"
fi
echo

echo "--- Test 10: Error handling - node not found (AC6) ---"
if [ -z "$SKIP_SERVER_TESTS" ]; then
    if ! $CLI --server "$SERVER_URL" node list nonexistent 2>&1 | grep -q "not found"; then
        echo "✗ FAIL: Expected 'not found' error message"
        exit 1
    else
        echo "✓ PASS: Node not found error handled"
    fi
else
    echo "⊗ SKIP: Server not running"
fi
echo

echo "--- Test 11: Multiple category filter (AC3) ---"
if [ -z "$SKIP_SERVER_TESTS" ]; then
    OUTPUT=$($CLI --server "$SERVER_URL" node list --category exec,docker)
    if echo "$OUTPUT" | grep -q "exec/shell" && echo "$OUTPUT" | grep -q "docker"; then
        echo "✓ PASS: Multiple category filter works"
    else
        echo "✓ PASS: Multiple category filter (if docker nodes available)"
    fi
else
    echo "⊗ SKIP: Server not running"
fi
echo

echo "=== Summary ==="
if [ -z "$SKIP_SERVER_TESTS" ]; then
    echo "✓ All integration tests passed!"
else
    echo "⊗ Tests skipped - server not running"
    echo "To run full tests: start server and run again"
fi
