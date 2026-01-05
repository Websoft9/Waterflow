#!/bin/bash
# Simple integration tests for waterflow logs command (Story 5.5)

set +e # Don't exit on error, we want to see all test results

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

TESTS_RUN=0
TESTS_PASSED=0
CLI="./bin/waterflow"
WF_ID="${1:-073e7d14-81aa-48f7-bbe2-208d290525a7}"

echo "================================================================"
echo "Waterflow CLI - Logs Command Tests (Story 5.5)"
echo "Workflow ID: $WF_ID"
echo "================================================================"

# AC1: Query logs with default settings
echo -e "\n${YELLOW}AC1: Query logs (default)${NC}"
((TESTS_RUN++))
if $CLI logs "$WF_ID" 2>&1 | grep -q "Workflow"; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}"
fi

# AC2: Query logs with --tail
echo -e "\n${YELLOW}AC2: Query logs with --tail 5${NC}"
((TESTS_RUN++))
OUTPUT=$($CLI logs --tail 5 "$WF_ID" 2>&1)
LINE_COUNT=$(echo "$OUTPUT" | wc -l)
if [ "$LINE_COUNT" -le 6 ]; then  # 5 lines + possible empty line
    echo -e "${GREEN}✓ PASS${NC} (got $LINE_COUNT lines)"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} (got $LINE_COUNT lines)"
fi

# AC3: Filter logs by level
echo -e "\n${YELLOW}AC3: Filter logs by level (--level info)${NC}"
((TESTS_RUN++))
if $CLI logs --level info "$WF_ID" 2>&1 | grep -E "INFO|No logs found" > /dev/null; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}"
fi

# AC4: Filter logs by job
echo -e "\n${YELLOW}AC4: Filter logs by nonexistent job${NC}"
((TESTS_RUN++))
if $CLI logs --job nonexistent "$WF_ID" 2>&1 | grep -q "No logs found"; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}"
fi

# AC5: Output in JSON format
echo -e "\n${YELLOW}AC5: Output logs in JSON format${NC}"
((TESTS_RUN++))
if $CLI logs --format json "$WF_ID" 2>&1 | grep -q '"timestamp"'; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}"
fi

# AC6: Output without timestamps
echo -e "\n${YELLOW}AC6: Output without timestamps${NC}"
((TESTS_RUN++))
OUTPUT=$($CLI logs --no-timestamps "$WF_ID" 2>&1)
if ! echo "$OUTPUT" | grep -q "\[20[0-9][0-9]-"; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}"
fi

# AC7: Output without colors
echo -e "\n${YELLOW}AC7: Output without colors${NC}"
((TESTS_RUN++))
OUTPUT=$($CLI logs --no-color "$WF_ID" 2>&1)
if ! echo "$OUTPUT" | grep -q $'\033\['; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}"
fi

# Extra: Compact format
echo -e "\n${YELLOW}EXTRA: Compact format${NC}"
((TESTS_RUN++))
OUTPUT=$($CLI logs --format compact "$WF_ID" 2>&1)
if echo "$OUTPUT" | grep -q "Workflow" && ! echo "$OUTPUT" | grep -q "\[20[0-9][0-9]-"; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}"
fi

# Extra: Invalid workflow ID
echo -e "\n${YELLOW}EXTRA: Invalid workflow ID${NC}"
((TESTS_RUN++))
if $CLI logs "invalid-id-12345" 2>&1 | grep -E "Workflow not found|404" > /dev/null; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC}"
fi

# Summary
TESTS_FAILED=$((TESTS_RUN - TESTS_PASSED))
echo ""
echo "================================================================"
echo "Test Summary"
echo "================================================================"
echo "Total:  $TESTS_RUN"
echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Failed: ${RED}$TESTS_FAILED${NC}"
echo "================================================================"

if [ "$TESTS_FAILED" -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed!${NC}"
    exit 1
fi
