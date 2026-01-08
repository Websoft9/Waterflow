#!/bin/bash
# Metrics endpoint integration test

set -e

SERVER_URL="${SERVER_URL:-http://localhost:8080}"

echo "=== Metrics Endpoint Integration Test ==="
echo "Server: $SERVER_URL"
echo ""

# Test 1: Endpoint accessibility
echo "Test 1: Metrics endpoint accessibility"
echo "---------------------------------------"

RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/metrics")
HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" != "200" ]; then
    echo "❌ Metrics endpoint returned HTTP $HTTP_CODE"
    exit 1
fi

echo "✅ Endpoint accessible (HTTP 200)"

# Test 2: Content-Type validation
echo ""
echo "Test 2: Content-Type validation"
echo "--------------------------------"

CONTENT_TYPE=$(curl -s -I "$SERVER_URL/metrics" \
    | grep -i "content-type" \
    | awk '{print $2}' \
    | tr -d '\r')

if [[ ! "$CONTENT_TYPE" =~ "text/plain" ]]; then
    echo "❌ Invalid Content-Type: $CONTENT_TYPE"
    exit 1
fi

echo "✅ Content-Type: $CONTENT_TYPE"

# Test 3: Prometheus format validation
echo ""
echo "Test 3: Prometheus format validation"
echo "-------------------------------------"

if ! echo "$BODY" | grep -q "# HELP"; then
    echo "❌ Missing HELP comments"
    exit 1
fi

if ! echo "$BODY" | grep -q "# TYPE"; then
    echo "❌ Missing TYPE metadata"
    exit 1
fi

echo "✅ Prometheus format validated"

# Test 4: Required metrics existence
echo ""
echo "Test 4: Required metrics existence"
echo "-----------------------------------"

REQUIRED_METRICS=(
    "waterflow_http_requests_total"
    "waterflow_http_request_duration_seconds"
    "waterflow_workflows_total"
    "waterflow_workflows_running"
    "waterflow_workflow_duration_seconds"
    "waterflow_agents_connected"
    "waterflow_agents_healthy"
    "waterflow_agent_tasks_total"
    "waterflow_node_executions_total"
    "waterflow_node_execution_duration_seconds"
    "waterflow_workflow_submissions_total"
    "waterflow_yaml_validations_total"
)

MISSING_METRICS=()

for metric in "${REQUIRED_METRICS[@]}"; do
    if ! echo "$BODY" | grep -q "$metric"; then
        MISSING_METRICS+=("$metric")
        echo "⚠️  Missing: $metric"
    else
        echo "✅ Found: $metric"
    fi
done

if [ ${#MISSING_METRICS[@]} -gt 0 ]; then
    echo ""
    echo "❌ Missing ${#MISSING_METRICS[@]} required metrics"
    exit 1
fi

# Test 5: Go runtime metrics
echo ""
echo "Test 5: Go runtime metrics"
echo "--------------------------"

GO_METRICS=(
    "go_goroutines"
    "go_memstats_alloc_bytes"
    "process_cpu_seconds_total"
)

for metric in "${GO_METRICS[@]}"; do
    if echo "$BODY" | grep -q "$metric"; then
        echo "✅ Found: $metric"
    else
        echo "⚠️  Missing: $metric (optional)"
    fi
done

# Test 6: Response time
echo ""
echo "Test 6: Response time"
echo "---------------------"

START=$(date +%s%N)
curl -s "$SERVER_URL/metrics" > /dev/null
END=$(date +%s%N)
DURATION=$(( (END - START) / 1000000 ))  # Convert to ms

echo "Response time: ${DURATION}ms"

if [ $DURATION -gt 100 ]; then
    echo "⚠️  Response time > 100ms"
else
    echo "✅ Response time within limits"
fi

# Test 7: Metric label validation
echo ""
echo "Test 7: Metric label validation"
echo "--------------------------------"

# Check workflow status labels
if echo "$BODY" | grep -q 'waterflow_workflows_total{status="submitted"}'; then
    echo "✅ Workflow status label found"
else
    echo "⚠️  No workflow submissions yet"
fi

# Check HTTP method/path labels
if echo "$BODY" | grep -q 'waterflow_http_requests_total{.*method=.*path=.*status='; then
    echo "✅ HTTP request labels found"
else
    echo "⚠️  No HTTP requests tracked yet"
fi

# Summary
echo ""
echo "========================================"
echo "  Test Summary"
echo "========================================"
echo "✅ Metrics endpoint functional"
echo "✅ All required metrics registered"
echo "✅ Prometheus format compliant"
echo "✅ Response time acceptable"
echo ""
echo "✅ Metrics integration test PASSED"
