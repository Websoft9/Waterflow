#!/bin/bash
# Docker Security Configuration Test
# Tests container security options, user privileges, and resource limits

set -e

PASS=0
FAIL=0

echo "=== Docker Security Configuration Test ==="
echo ""

# Test 1: Non-Root User
echo "Test 1: Containers Running as Non-Root User"
for container in waterflow-server waterflow-agent-linux-1; do
    if docker inspect "$container" --format '{{.Config.User}}' 2>/dev/null | grep -q "waterflow"; then
        echo "✅ PASS: $container runs as non-root user"
        ((PASS++))
    else
        echo "❌ FAIL: $container runs as root"
        ((FAIL++))
    fi
done

# Test 2: Read-Only Filesystem
echo ""
echo "Test 2: Read-Only Filesystem Configuration"
for container in waterflow-server waterflow-agent-linux-1; do
    if docker inspect "$container" --format '{{.HostConfig.ReadonlyRootfs}}' 2>/dev/null | grep -q "true"; then
        echo "✅ PASS: $container has read-only filesystem"
        ((PASS++))
    else
        echo "⚠️  WARNING: $container filesystem is writable (may need tmpfs volumes)"
    fi
done

# Test 3: No New Privileges
echo ""
echo "Test 3: no-new-privileges Security Option"
for container in waterflow-server waterflow-agent-linux-1 waterflow-postgresql waterflow-temporal; do
    if docker inspect "$container" --format '{{.HostConfig.SecurityOpt}}' 2>/dev/null | grep -q "no-new-privileges"; then
        echo "✅ PASS: $container has no-new-privileges"
        ((PASS++))
    else
        echo "❌ FAIL: $container missing no-new-privileges"
        ((FAIL++))
    fi
done

# Test 4: Capability Drop
echo ""
echo "Test 4: Capabilities Dropped"
for container in waterflow-server waterflow-agent-linux-1; do
    if docker inspect "$container" --format '{{.HostConfig.CapDrop}}' 2>/dev/null | grep -q "ALL"; then
        echo "✅ PASS: $container dropped all capabilities"
        ((PASS++))
    else
        echo "❌ FAIL: $container has unnecessary capabilities"
        ((FAIL++))
    fi
done

# Test 5: Resource Limits
echo ""
echo "Test 5: Resource Limits Configured"
for container in waterflow-server waterflow-agent-linux-1 waterflow-postgresql waterflow-temporal; do
    MEMORY_LIMIT=$(docker inspect "$container" --format '{{.HostConfig.Memory}}' 2>/dev/null)
    CPU_LIMIT=$(docker inspect "$container" --format '{{.HostConfig.NanoCpus}}' 2>/dev/null)
    
    if [ "$MEMORY_LIMIT" != "0" ] || [ "$CPU_LIMIT" != "0" ]; then
        echo "✅ PASS: $container has resource limits (Memory: $MEMORY_LIMIT, CPU: $CPU_LIMIT)"
        ((PASS++))
    else
        echo "⚠️  WARNING: $container has no resource limits"
    fi
done

# Test 6: Network Isolation
echo ""
echo "Test 6: Network Isolation"
NETWORKS=$(docker network ls --filter "name=waterflow" --format "{{.Name}}")
if echo "$NETWORKS" | grep -q "waterflow-network"; then
    echo "✅ PASS: Custom network configured for isolation"
    ((PASS++))
else
    echo "❌ FAIL: No custom network for isolation"
    ((FAIL++))
fi

# Summary
echo ""
echo "=== Test Summary ==="
echo "Passed: $PASS"
echo "Failed: $FAIL"
echo ""
echo "Note: Run 'docker-compose up -d' before running this test"
[ $FAIL -eq 0 ] && exit 0 || exit 1
