#!/bin/bash
# CLI validate command integration tests

set -e

CLI="./bin/waterflow"
TESTDATA="./testdata"

echo "=== CLI validate Command Integration Tests ==="
echo

# Test 1: Valid workflow (AC1)
echo "Test 1: Valid workflow validation"
OUTPUT=$($CLI validate $TESTDATA/valid/simple.yaml)
if ! echo "$OUTPUT" | grep -q "✓ Workflow is valid"; then
    echo "FAIL: Should show workflow is valid"
    exit 1
fi
if ! echo "$OUTPUT" | grep -q "Build and Test"; then
    echo "FAIL: Should show workflow name"
    exit 1
fi
echo "PASS"
echo

# Test 2: Syntax error detection (AC2)
echo "Test 2: Syntax error detection"
if $CLI validate $TESTDATA/invalid/syntax-error.yaml 2>&1 | grep -q "yaml_syntax_error"; then
    echo "PASS"
else
    echo "FAIL: Should report yaml_syntax_error"
    exit 1
fi
echo

# Test 3: Multiple files validation (AC3)
echo "Test 3: Multiple files validation"
OUTPUT=$($CLI validate $TESTDATA/valid/simple.yaml $TESTDATA/invalid/syntax-error.yaml 2>&1 || true)
if ! echo "$OUTPUT" | grep -q "Summary:"; then
    echo "FAIL: Should show summary for multiple files"
    exit 1
fi
if ! echo "$OUTPUT" | grep -q "Total:   2"; then
    echo "FAIL: Should show total count"
    exit 1
fi
echo "PASS"
echo

# Test 4: Recursive directory scan (AC3)
echo "Test 4: Recursive directory validation"
OUTPUT=$($CLI validate --recursive $TESTDATA/valid/ 2>&1)
if ! echo "$OUTPUT" | grep -q "Validating"; then
    echo "FAIL: Should show validating message"
    exit 1
fi
if ! echo "$OUTPUT" | grep -q "Summary:"; then
    echo "FAIL: Should show summary"
    exit 1
fi
echo "PASS"
echo

# Test 5: JSON output format (AC5)
echo "Test 5: JSON output format"
OUTPUT=$($CLI validate --format json $TESTDATA/valid/simple.yaml)
if ! echo "$OUTPUT" | jq -e '.valid == true' > /dev/null 2>&1; then
    echo "FAIL: JSON output should have valid=true"
    exit 1
fi
if ! echo "$OUTPUT" | jq -e '.workflow_name == "Build and Test"' > /dev/null 2>&1; then
    echo "FAIL: JSON should contain workflow name"
    exit 1
fi
if ! echo "$OUTPUT" | jq -e 'has("validation_time_ms")' > /dev/null 2>&1; then
    echo "FAIL: JSON should contain validation_time_ms field"
    exit 1
fi
echo "PASS"
echo

# Test 6: YAML output format (AC5)
echo "Test 6: YAML output format"
OUTPUT=$($CLI validate --format yaml $TESTDATA/valid/simple.yaml)
if ! echo "$OUTPUT" | grep -q "valid: true"; then
    echo "FAIL: YAML output should have valid: true"
    exit 1
fi
echo "PASS"
echo

# Test 7: File not found error (AC6)
echo "Test 7: File not found error handling"
if $CLI validate nonexistent.yaml 2>&1 | grep -q "file not found"; then
    echo "PASS"
else
    echo "FAIL: Should report file not found"
    exit 1
fi
echo

# Test 8: Empty file error (AC6)
echo "Test 8: Empty file error handling"
TMPFILE=$(mktemp --suffix=.yaml)
if $CLI validate $TMPFILE 2>&1 | grep -q "Empty YAML file"; then
    echo "PASS"
else
    echo "FAIL: Should report empty file error"
    rm -f $TMPFILE
    exit 1
fi
rm -f $TMPFILE
echo

# Test 9: Verbose mode (AC1)
echo "Test 9: Verbose output mode"
OUTPUT=$($CLI validate --verbose $TESTDATA/valid/simple.yaml)
if ! echo "$OUTPUT" | grep -q "Workflow Details:"; then
    echo "FAIL: Verbose mode should show detailed information"
    exit 1
fi
if ! echo "$OUTPUT" | grep -q "Validation passed in"; then
    echo "FAIL: Verbose mode should show validation time"
    exit 1
fi
echo "PASS"
echo

# Test 10: Exit code on validation failure
echo "Test 10: Exit code validation"
if $CLI validate $TESTDATA/invalid/syntax-error.yaml > /dev/null 2>&1; then
    echo "FAIL: Should exit with non-zero code on validation failure"
    exit 1
else
    echo "PASS"
fi
echo

echo "==================================="
echo "All 10 integration tests passed! ✓"
echo "==================================="
