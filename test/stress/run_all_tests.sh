#!/bin/bash
# Run all stress tests sequentially
# Usage: ./test/stress/run_all_tests.sh

set -e

echo "========================================"
echo "  Waterflow Stress Test Suite"
echo "========================================"
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 结果目录
RESULTS_ROOT="test/stress/results/suite-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$RESULTS_ROOT"

# 测试结果统计
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 运行测试函数
run_test() {
    local test_name=$1
    local test_script=$2
    
    ((TOTAL_TESTS++))
    
    echo ""
    echo "================================================"
    echo "Running: $test_name"
    echo "================================================"
    
    if bash "$test_script" > "$RESULTS_ROOT/${test_name}.log" 2>&1; then
        echo -e "${GREEN}✅ PASSED${NC}: $test_name"
        ((PASSED_TESTS++))
    else
        echo -e "${RED}❌ FAILED${NC}: $test_name"
        echo "   Log: $RESULTS_ROOT/${test_name}.log"
        ((FAILED_TESTS++))
    fi
}

# 检查前置条件
echo "Checking prerequisites..."

if ! command -v jq > /dev/null; then
    echo -e "${RED}Error: jq not found${NC}"
    exit 1
fi

if ! command -v bc > /dev/null; then
    echo -e "${RED}Error: bc not found${NC}"
    exit 1
fi

if [ ! -f "./bin/server" ]; then
    echo -e "${YELLOW}Warning: ./bin/server not found, building...${NC}"
    make build
fi

# 检查 Server 是否运行
if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo -e "${YELLOW}Warning: Server not running on http://localhost:8080${NC}"
    echo "Please start the server before running stress tests:"
    echo "  ./bin/server"
    echo ""
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

echo "Prerequisites OK"
echo ""

# 运行测试套件
echo "Starting test suite..."
START_TIME=$(date +%s)

# Test 1: 并发工作流测试 (AC1)
run_test "concurrent-workflows" "./test/stress/concurrent_workflows_test.sh"

# Test 2: Server 崩溃恢复测试 (AC2)
# 注意: 这个测试会重启 Server，如果 Server 由其他进程管理则跳过
if [ -n "$SKIP_CRASH_TEST" ]; then
    echo -e "${YELLOW}⊘ SKIPPED${NC}: server-crash-recovery (SKIP_CRASH_TEST set)"
else
    run_test "server-crash-recovery" "./test/stress/server_crash_recovery_test.sh"
fi

# Test 3: 资源泄漏检测 (AC3)
run_test "resource-leak-detection" "./test/stress/resource_leak_test.sh"

# Test 4: 超时重试验证 (AC5)
run_test "timeout-retry-validation" "./test/stress/timeout_retry_test.sh"

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# 生成总结报告
echo ""
echo "========================================"
echo "  Test Suite Summary"
echo "========================================"
echo "Total Tests: $TOTAL_TESTS"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"
echo "Duration: ${DURATION}s"
echo ""
echo "Results saved to: $RESULTS_ROOT"

# 创建汇总报告
cat > "$RESULTS_ROOT/summary.txt" <<EOF
=== Waterflow Stress Test Suite Summary ===
Date: $(date)
Duration: ${DURATION}s

Results:
  Total Tests: $TOTAL_TESTS
  Passed: $PASSED_TESTS
  Failed: $FAILED_TESTS
  Success Rate: $(echo "scale=2; $PASSED_TESTS * 100 / $TOTAL_TESTS" | bc)%

Tests:
  1. Concurrent Workflows (AC1): $(grep -q "concurrent-workflows" <<< "$PASSED_TESTS" && echo "PASS" || echo "FAIL")
  2. Server Crash Recovery (AC2): $([ -z "$SKIP_CRASH_TEST" ] && echo "RAN" || echo "SKIPPED")
  3. Resource Leak Detection (AC3): $(grep -q "resource-leak" <<< "$PASSED_TESTS" && echo "PASS" || echo "FAIL")
  4. Timeout/Retry Validation (AC5): $(grep -q "timeout-retry" <<< "$PASSED_TESTS" && echo "PASS" || echo "FAIL")

Logs:
$(ls -1 $RESULTS_ROOT/*.log)
EOF

cat "$RESULTS_ROOT/summary.txt"

# 返回状态
if [ $FAILED_TESTS -gt 0 ]; then
    echo ""
    echo -e "${RED}❌ Test suite FAILED${NC}"
    exit 1
else
    echo ""
    echo -e "${GREEN}✅ Test suite PASSED${NC}"
    exit 0
fi
