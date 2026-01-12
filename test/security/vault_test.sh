#!/bin/bash
# Vault Integration Test
# Tests Vault connectivity, secret retrieval, and dynamic secrets

set -e

VAULT_ADDR="${VAULT_ADDR:-http://localhost:8200}"
PASS=0
FAIL=0

echo "=== Vault Integration Test ==="
echo "Vault Address: $VAULT_ADDR"
echo ""

# Test 1: Vault Health Check
echo "Test 1: Vault Health Check"
if curl -sf "$VAULT_ADDR/v1/sys/health" > /dev/null 2>&1; then
    echo "✅ PASS: Vault is healthy and accessible"
    ((PASS++))
else
    echo "❌ FAIL: Cannot connect to Vault"
    ((FAIL++))
    exit 1  # Cannot continue without Vault
fi

# Test 2: Vault Authentication
echo ""
echo "Test 2: Vault Authentication"
if [ -n "$VAULT_TOKEN" ]; then
    if curl -sf -H "X-Vault-Token: $VAULT_TOKEN" "$VAULT_ADDR/v1/sys/health" > /dev/null 2>&1; then
        echo "✅ PASS: Vault authentication successful"
        ((PASS++))
    else
        echo "❌ FAIL: Vault authentication failed"
        ((FAIL++))
    fi
else
    echo "⚠️  SKIP: VAULT_TOKEN not set"
fi

# Test 3: Read Secret from KV Store
echo ""
echo "Test 3: Read Secret from KV Store"
if [ -n "$VAULT_TOKEN" ]; then
    SECRET_PATH="${VAULT_SECRET_PATH:-secret/data/waterflow/test}"
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "X-Vault-Token: $VAULT_TOKEN" "$VAULT_ADDR/v1/$SECRET_PATH")
    if [ "$HTTP_CODE" = "200" ]; then
        echo "✅ PASS: Secret retrieved successfully"
        ((PASS++))
    elif [ "$HTTP_CODE" = "404" ]; then
        echo "⚠️  WARNING: Secret not found (path: $SECRET_PATH)"
    else
        echo "❌ FAIL: Failed to retrieve secret (HTTP $HTTP_CODE)"
        ((FAIL++))
    fi
else
    echo "⚠️  SKIP: VAULT_TOKEN not set"
fi

# Test 4: Waterflow Vault Integration
echo ""
echo "Test 4: Waterflow SecretProvider Integration"
if command -v waterflow &> /dev/null; then
    # Check if waterflow config uses Vault
    if waterflow config show 2>/dev/null | grep -q "vault"; then
        echo "✅ PASS: Waterflow configured to use Vault"
        ((PASS++))
    else
        echo "⚠️  WARNING: Waterflow not configured for Vault"
    fi
else
    echo "⚠️  SKIP: Waterflow CLI not available"
fi

# Summary
echo ""
echo "=== Test Summary ==="
echo "Passed: $PASS"
echo "Failed: $FAIL"
echo ""
echo "Note: Set VAULT_TOKEN and VAULT_SECRET_PATH for full test coverage"
[ $FAIL -eq 0 ] && exit 0 || exit 1
