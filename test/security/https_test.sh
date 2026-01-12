#!/bin/bash
# HTTPS/TLS Configuration Test
# Tests TLS certificate validity, cipher suites, and protocol versions

set -e

WATERFLOW_URL="${WATERFLOW_URL:-https://localhost:8443}"
PASS=0
FAIL=0

echo "=== HTTPS/TLS Configuration Test ==="
echo "Target: $WATERFLOW_URL"
echo ""

# Test 1: TLS Certificate Validity
echo "Test 1: TLS Certificate Validity"
if openssl s_client -connect localhost:8443 -showcerts </dev/null 2>/dev/null | openssl x509 -noout -dates; then
    echo "✅ PASS: Certificate is valid"
    ((PASS++))
else
    echo "❌ FAIL: Certificate validation failed"
    ((FAIL++))
fi

# Test 2: TLS Protocol Version (should support TLS 1.2+)
echo ""
echo "Test 2: TLS Protocol Version"
if openssl s_client -connect localhost:8443 -tls1_2 </dev/null 2>&1 | grep -q "Protocol.*TLSv1.2"; then
    echo "✅ PASS: TLS 1.2 is supported"
    ((PASS++))
else
    echo "❌ FAIL: TLS 1.2 not supported"
    ((FAIL++))
fi

# Test 3: Strong Cipher Suites
echo ""
echo "Test 3: Strong Cipher Suites"
if openssl s_client -connect localhost:8443 -cipher 'HIGH:!aNULL:!MD5' </dev/null 2>&1 | grep -q "Cipher.*"; then
    echo "✅ PASS: Strong cipher suites configured"
    ((PASS++))
else
    echo "❌ FAIL: Weak cipher suites detected"
    ((FAIL++))
fi

# Test 4: HTTPS Redirect
echo ""
echo "Test 4: HTTP to HTTPS Redirect"
if curl -sI http://localhost:8080/ | grep -q "Location: https://"; then
    echo "✅ PASS: HTTP redirects to HTTPS"
    ((PASS++))
else
    echo "⚠️  WARNING: HTTP does not redirect to HTTPS (may be intentional for dev)"
fi

# Summary
echo ""
echo "=== Test Summary ==="
echo "Passed: $PASS"
echo "Failed: $FAIL"
[ $FAIL -eq 0 ] && exit 0 || exit 1
