#!/bin/bash
# Integration test for Template API endpoints
# Tests GET /v1/templates and GET /v1/templates/{name}

set -e

SERVER_URL="${SERVER_URL:-http://localhost:8080}"
PASSED=0
FAILED=0

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
pass() {
    echo -e "${GREEN}✓${NC} $1"
    ((PASSED++))
}

fail() {
    echo -e "${RED}✗${NC} $1"
    ((FAILED++))
}

info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

# Test 1: List all templates
info "Test 1: GET /v1/templates (list all templates)"
response=$(curl -s -w "\n%{http_code}" "${SERVER_URL}/v1/templates")
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" = "200" ]; then
    count=$(echo "$body" | jq -r '.count')
    if [ "$count" = "3" ]; then
        pass "List all templates: 3 templates returned"
    else
        fail "List all templates: Expected 3 templates, got $count"
    fi
else
    fail "List all templates: Expected HTTP 200, got $http_code"
fi

# Test 2: Filter by category=deployment
info "Test 2: GET /v1/templates?category=deployment"
response=$(curl -s -w "\n%{http_code}" "${SERVER_URL}/v1/templates?category=deployment")
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" = "200" ]; then
    count=$(echo "$body" | jq -r '.count')
    if [ "$count" = "2" ]; then
        pass "Filter by deployment: 2 templates returned"
    else
        fail "Filter by deployment: Expected 2 templates, got $count"
    fi
else
    fail "Filter by deployment: Expected HTTP 200, got $http_code"
fi

# Test 3: Filter by category=monitoring
info "Test 3: GET /v1/templates?category=monitoring"
response=$(curl -s -w "\n%{http_code}" "${SERVER_URL}/v1/templates?category=monitoring")
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" = "200" ]; then
    count=$(echo "$body" | jq -r '.count')
    if [ "$count" = "1" ]; then
        pass "Filter by monitoring: 1 template returned"
    else
        fail "Filter by monitoring: Expected 1 template, got $count"
    fi
else
    fail "Filter by monitoring: Expected HTTP 200, got $http_code"
fi

# Test 4: Get specific template with content
info "Test 4: GET /v1/templates/single-server-deployment (with content)"
response=$(curl -s -w "\n%{http_code}" "${SERVER_URL}/v1/templates/single-server-deployment")
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" = "200" ]; then
    name=$(echo "$body" | jq -r '.name')
    content=$(echo "$body" | jq -r '.content')
    
    if [ "$name" = "single-server-deployment" ] && [ "$content" != "null" ] && [ -n "$content" ]; then
        pass "Get template with content: Content included"
    else
        fail "Get template with content: Content missing or invalid"
    fi
else
    fail "Get template with content: Expected HTTP 200, got $http_code"
fi

# Test 5: Get template without content
info "Test 5: GET /v1/templates/single-server-deployment?content=false"
response=$(curl -s -w "\n%{http_code}" "${SERVER_URL}/v1/templates/single-server-deployment?content=false")
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" = "200" ]; then
    content=$(echo "$body" | jq -r '.content // empty')
    
    if [ -z "$content" ] || [ "$content" = "null" ]; then
        pass "Get template without content: Content omitted"
    else
        fail "Get template without content: Content should be omitted"
    fi
else
    fail "Get template without content: Expected HTTP 200, got $http_code"
fi

# Test 6: Get non-existent template (404)
info "Test 6: GET /v1/templates/invalid-template (expect 404)"
response=$(curl -s -w "\n%{http_code}" "${SERVER_URL}/v1/templates/invalid-template")
http_code=$(echo "$response" | tail -n1)

if [ "$http_code" = "404" ]; then
    pass "Non-existent template: Returns 404"
else
    fail "Non-existent template: Expected HTTP 404, got $http_code"
fi

# Test 7: Verify template structure
info "Test 7: Verify template response structure"
response=$(curl -s "${SERVER_URL}/v1/templates/multi-server-health-check")
name=$(echo "$response" | jq -r '.name')
description=$(echo "$response" | jq -r '.description')
category=$(echo "$response" | jq -r '.category')
params=$(echo "$response" | jq '.parameters | length')

if [ "$name" = "multi-server-health-check" ] && [ -n "$description" ] && [ "$category" = "monitoring" ] && [ "$params" -gt 0 ]; then
    pass "Template structure: All required fields present"
else
    fail "Template structure: Missing required fields (name: $name, category: $category, params: $params)"
fi

# Test 8: Verify parameters structure
info "Test 8: Verify parameter structure"
response=$(curl -s "${SERVER_URL}/v1/templates/distributed-stack-deployment")
param_name=$(echo "$response" | jq -r '.parameters[0].name')
param_type=$(echo "$response" | jq -r '.parameters[0].type')
param_required=$(echo "$response" | jq -r '.parameters[0].required')
param_desc=$(echo "$response" | jq -r '.parameters[0].description')

if [ -n "$param_name" ] && [ -n "$param_type" ] && [ -n "$param_desc" ]; then
    pass "Parameter structure: All fields present (name: $param_name, type: $param_type)"
else
    fail "Parameter structure: Missing fields"
fi

# Test 9: Performance test - concurrent requests
info "Test 9: Concurrent requests (10 requests)"
start_time=$(date +%s%N)
for i in {1..10}; do
    curl -s "${SERVER_URL}/v1/templates" > /dev/null &
done
wait
end_time=$(date +%s%N)
duration=$((($end_time - $start_time) / 1000000)) # Convert to milliseconds

if [ $duration -lt 500 ]; then
    pass "Performance: 10 concurrent requests completed in ${duration}ms"
else
    fail "Performance: 10 requests took ${duration}ms (expected <500ms)"
fi

# Test 10: Verify YAML content is valid
info "Test 10: Verify YAML content validity"
response=$(curl -s "${SERVER_URL}/v1/templates/single-server-deployment")
content=$(echo "$response" | jq -r '.content')

if echo "$content" | grep -q "name:" && echo "$content" | grep -q "jobs:"; then
    pass "YAML content: Valid workflow YAML"
else
    fail "YAML content: Invalid or incomplete YAML"
fi

# Summary
echo ""
echo "================================"
echo "Test Results:"
echo "  Passed: ${GREEN}$PASSED${NC}"
echo "  Failed: ${RED}$FAILED${NC}"
echo "================================"

if [ $FAILED -gt 0 ]; then
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi
