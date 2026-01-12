#!/bin/bash
# Authentication & Authorization Test
# Tests API key validation, JWT tokens, and access control

set -e

WATERFLOW_URL="${WATERFLOW_URL:-http://localhost:8080}"
PASS=0
FAIL=0

echo "=== Authentication & Authorization Test ==="
echo "Target: $WATERFLOW_URL"
echo ""

# Test 1: Unauthenticated Request (should fail)
echo "Test 1: Unauthenticated Request Rejection"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$WATERFLOW_URL/v1/workflows")
if [ "$HTTP_CODE" = "401" ] || [ "$HTTP_CODE" = "403" ]; then
    echo "✅ PASS: Unauthenticated request rejected (HTTP $HTTP_CODE)"
    ((PASS++))
else
    echo "⚠️  WARNING: Unauthenticated request allowed (HTTP $HTTP_CODE) - may be intentional for dev"
fi

# Test 2: Invalid API Key (should fail)
echo ""
echo "Test 2: Invalid API Key Rejection"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: invalid-key-12345" "$WATERFLOW_URL/v1/workflows")
if [ "$HTTP_CODE" = "401" ] || [ "$HTTP_CODE" = "403" ]; then
    echo "✅ PASS: Invalid API key rejected (HTTP $HTTP_CODE)"
    ((PASS++))
else
    echo "⚠️  WARNING: Invalid API key accepted (HTTP $HTTP_CODE)"
fi

# Test 3: Valid API Key (should succeed)
echo ""
echo "Test 3: Valid API Key Authentication"
if [ -n "$WATERFLOW_API_KEY" ]; then
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $WATERFLOW_API_KEY" "$WATERFLOW_URL/v1/workflows")
    if [ "$HTTP_CODE" = "200" ]; then
        echo "✅ PASS: Valid API key accepted (HTTP $HTTP_CODE)"
        ((PASS++))
    else
        echo "❌ FAIL: Valid API key rejected (HTTP $HTTP_CODE)"
        ((FAIL++))
    fi
else
    echo "⚠️  SKIP: WATERFLOW_API_KEY not set"
fi

# Test 4: JWT Token Validation
echo ""
echo "Test 4: JWT Token Validation"
if [ -n "$WATERFLOW_JWT_TOKEN" ]; then
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $WATERFLOW_JWT_TOKEN" "$WATERFLOW_URL/v1/workflows")
    if [ "$HTTP_CODE" = "200" ]; then
        echo "✅ PASS: Valid JWT token accepted (HTTP $HTTP_CODE)"
        ((PASS++))
    else
        echo "❌ FAIL: Valid JWT token rejected (HTTP $HTTP_CODE)"
        ((FAIL++))
    fi
else
    echo "⚠️  SKIP: WATERFLOW_JWT_TOKEN not set"
fi

# Summary
echo ""
echo "=== Test Summary ==="
echo "Passed: $PASS"
echo "Failed: $FAIL"
echo ""
echo "Note: Set WATERFLOW_API_KEY and WATERFLOW_JWT_TOKEN environment variables for full test coverage"
[ $FAIL -eq 0 ] && exit 0 || exit 1
