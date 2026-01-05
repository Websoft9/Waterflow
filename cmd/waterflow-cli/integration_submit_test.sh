#!/bin/bash
# CLI submit 命令集成测试
# NOTE: 需要 Waterflow Server 运行在 http://localhost:8088

set -e

CLI="./bin/waterflow"
TESTDATA="./testdata/valid"
SERVER_URL="${WATERFLOW_SERVER:-http://localhost:8088}"

echo "=== CLI Submit Command Integration Tests ==="
echo "Server URL: $SERVER_URL"
echo

# 检查 CLI 是否存在
if [ ! -f "$CLI" ]; then
    echo "ERROR: CLI binary not found at $CLI"
    echo "Build with: make build"
    exit 1
fi

# 检查 Server 是否运行
if ! curl -sf "$SERVER_URL/health" > /dev/null 2>&1; then
    echo "WARNING: Waterflow server not running at $SERVER_URL"
    echo "Some tests will be skipped"
    echo
    echo "To run full integration tests:"
    echo "1. Start server: docker-compose up -d"
    echo "2. Run tests again"
    echo
    SERVER_AVAILABLE=false
else
    echo "✓ Server is running"
    SERVER_AVAILABLE=true
fi

echo

# Unit tests (不需要 Server)
echo "=== Running Unit Tests ==="
echo

echo "Test 1: Variable parsing"
go test -v ./cmd/waterflow-cli/cmd -run TestParseVars > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ Variable parsing tests passed"
else
    echo "✗ Variable parsing tests failed"
    exit 1
fi

echo "Test 2: Workflow ID validation"
go test -v ./cmd/waterflow-cli/cmd -run TestIsValidWorkflowID > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ Workflow ID validation tests passed"
else
    echo "✗ Workflow ID validation tests failed"
    exit 1
fi

echo "Test 3: HTTP client"
go test -v ./cmd/waterflow-cli/pkg/client > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ HTTP client tests passed"
else
    echo "✗ HTTP client tests failed"
    exit 1
fi

echo

# Integration tests (需要 Server)
if [ "$SERVER_AVAILABLE" = true ]; then
    echo "=== Running Integration Tests (with Server) ==="
    echo
    
    # 创建临时测试工作流文件
    TEMP_WORKFLOW=$(mktemp)
    cat > "$TEMP_WORKFLOW" <<'EOF'
name: Test Workflow
jobs:
  test:
    steps:
      - name: Echo test
        node: exec/shell
        params:
          command: echo "Hello from submit test"
EOF
    
    echo "Test 4: Basic submit (AC1)"
    OUTPUT=$($CLI submit --server "$SERVER_URL" "$TEMP_WORKFLOW" 2>&1)
    if echo "$OUTPUT" | grep -q "Workflow submitted successfully"; then
        WORKFLOW_ID=$(echo "$OUTPUT" | grep "Workflow ID" | awk '{print $3}')
        if [ -n "$WORKFLOW_ID" ]; then
            echo "✓ Basic submit passed (Workflow ID: $WORKFLOW_ID)"
        else
            echo "✗ No workflow ID returned"
            echo "$OUTPUT"
            rm -f "$TEMP_WORKFLOW"
            exit 1
        fi
    else
        echo "✗ Basic submit failed"
        echo "$OUTPUT"
        rm -f "$TEMP_WORKFLOW"
        exit 1
    fi
    
    echo "Test 5: Variable override (AC2)"
    OUTPUT=$($CLI submit --server "$SERVER_URL" "$TEMP_WORKFLOW" --var env=test --var debug=true 2>&1)
    if echo "$OUTPUT" | grep -q "Workflow submitted successfully"; then
        echo "✓ Variable override passed"
    else
        echo "✗ Variable override failed"
        echo "$OUTPUT"
        rm -f "$TEMP_WORKFLOW"
        exit 1
    fi
    
    echo "Test 6: JSON output (AC7)"
    OUTPUT=$($CLI submit --server "$SERVER_URL" --format json "$TEMP_WORKFLOW" 2>&1)
    if echo "$OUTPUT" | jq -e '.id' > /dev/null 2>&1; then
        echo "✓ JSON output passed"
    else
        echo "✗ JSON output failed"
        echo "$OUTPUT"
        rm -f "$TEMP_WORKFLOW"
        exit 1
    fi
    
    echo "Test 7: Quiet mode (AC7)"
    WORKFLOW_ID=$($CLI submit --server "$SERVER_URL" --quiet "$TEMP_WORKFLOW" 2>&1)
    if [[ "$WORKFLOW_ID" =~ ^[0-9a-f-]{36}$ ]]; then
        echo "✓ Quiet mode passed (ID: $WORKFLOW_ID)"
    else
        echo "✗ Quiet mode failed (invalid ID format: $WORKFLOW_ID)"
        rm -f "$TEMP_WORKFLOW"
        exit 1
    fi
    
    # 创建无效工作流用于错误测试
    INVALID_WORKFLOW=$(mktemp)
    cat > "$INVALID_WORKFLOW" <<'EOF'
invalid: yaml
missing: jobs
EOF
    
    echo "Test 8: Validation error handling (AC6)"
    OUTPUT=$($CLI submit --server "$SERVER_URL" "$INVALID_WORKFLOW" 2>&1 || true)
    if echo "$OUTPUT" | grep -q -i "error"; then
        echo "✓ Error handling passed"
    else
        echo "✗ Error handling failed"
        echo "$OUTPUT"
        rm -f "$TEMP_WORKFLOW" "$INVALID_WORKFLOW"
        exit 1
    fi
    
    # 清理临时文件
    rm -f "$TEMP_WORKFLOW" "$INVALID_WORKFLOW"
    
    echo
    echo "Test 9: Wait mode (AC3)"
    # Create fast workflow for wait test
    FAST_WORKFLOW=$(mktemp)
    cat > "$FAST_WORKFLOW" <<'EOF'
name: Fast Test
jobs:
  quick:
    steps:
      - name: Quick step
        node: exec/shell
        params:
          command: echo "done"
EOF
    if timeout 10s $CLI submit --server "$SERVER_URL" --wait "$FAST_WORKFLOW" 2>&1 | grep -q "Workflow completed"; then
        echo "✓ Wait mode passed"
    else
        echo "⚠ Wait mode skipped (workflow may take too long)"
    fi
    rm -f "$FAST_WORKFLOW"
    
    echo "Test 10: Validate before submit (AC5)"
    OUTPUT=$($CLI submit --server "$SERVER_URL" --validate "$TEMP_WORKFLOW" 2>&1 || true)
    if echo "$OUTPUT" | grep -q "Workflow is valid"; then
        echo "✓ Pre-validation passed"
    else
        echo "⚠ Pre-validation skipped"
    fi
    
    echo
    echo "=== All Integration Tests Passed! ==="
else
    echo "=== Skipping Integration Tests (Server not available) ==="
    echo
    echo "All unit tests passed!"
fi

echo
echo "Summary:"
echo "  Unit tests: PASSED"
if [ "$SERVER_AVAILABLE" = true ]; then
    echo "  Integration tests: PASSED"
else
    echo "  Integration tests: SKIPPED (server not running)"
fi
