#!/bin/bash
# API Performance Benchmark using curl (no external deps required)
# For more accurate results, use hey or vegeta (see install_tools.sh)

set -e

SERVER_URL="${SERVER_URL:-http://localhost:8080}"
WORKFLOW_FILE="${WORKFLOW_FILE:-examples/hello-world.yaml}"
REQUESTS="${REQUESTS:-100}"
CONCURRENCY="${CONCURRENCY:-10}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

mkdir -p results

echo "=== API Performance Benchmark ==="
echo "Server: $SERVER_URL"
echo "Requests: $REQUESTS"
echo "Concurrency: $CONCURRENCY"
echo ""

# Check if server is running
if ! curl -sf "$SERVER_URL/health" > /dev/null 2>&1; then
    echo -e "${RED}❌ Server is not running at $SERVER_URL${NC}"
    echo "Please start the server first:"
    echo "  ./bin/server"
    exit 1
fi

echo "✓ Server is running"
echo ""

# Function to measure latency
measure_latency() {
    local url=$1
    local method=${2:-GET}
    local body=$3
    local label=$4
    
    echo "--- $label ---"
    
    local total_time=0
    local count=0
    local min_time=999999
    local max_time=0
    
    # Warm-up
    if [ "$method" = "POST" ] && [ -n "$body" ]; then
        curl -s -X POST "$url" \
            -H "Content-Type: application/yaml" \
            --data-binary "@$body" > /dev/null 2>&1
    else
        curl -s "$url" > /dev/null 2>&1
    fi
    
    # Measure latency
    for i in $(seq 1 $REQUESTS); do
        if [ "$method" = "POST" ] && [ -n "$body" ]; then
            time_taken=$(curl -s -w '%{time_total}\n' -o /dev/null -X POST "$url" \
                -H "Content-Type: application/yaml" \
                --data-binary "@$body")
        else
            time_taken=$(curl -s -w '%{time_total}\n' -o /dev/null "$url")
        fi
        
        # Convert to milliseconds
        time_ms=$(echo "$time_taken * 1000" | bc)
        total_time=$(echo "$total_time + $time_ms" | bc)
        count=$((count + 1))
        
        # Track min/max
        if (( $(echo "$time_ms < $min_time" | bc -l) )); then
            min_time=$time_ms
        fi
        if (( $(echo "$time_ms > $max_time" | bc -l) )); then
            max_time=$time_ms
        fi
        
        # Progress
        if (( i % 10 == 0 )); then
            echo -n "."
        fi
    done
    
    echo ""
    
    # Calculate average
    avg_time=$(echo "scale=2; $total_time / $count" | bc)
    
    echo "Requests: $count"
    echo "Min: ${min_time}ms"
    echo "Max: ${max_time}ms"
    echo "Avg: ${avg_time}ms"
    
    # Store results
    echo "$label,$count,$min_time,$avg_time,$max_time" >> results/api_bench.csv
    
    echo ""
}

# Initialize CSV
echo "Endpoint,Requests,Min(ms),Avg(ms),Max(ms)" > results/api_bench.csv

# Test 1: POST /v1/validate (验证 YAML)
measure_latency "$SERVER_URL/v1/validate" "POST" "$WORKFLOW_FILE" "POST /v1/validate"

# Test 2: POST /v1/workflows (提交工作流)
echo "--- POST /v1/workflows ---"
workflow_id=""
for i in $(seq 1 5); do
    result=$(curl -s -X POST "$SERVER_URL/v1/workflows" \
        -H "Content-Type: application/yaml" \
        --data-binary "@$WORKFLOW_FILE")
    
    if echo "$result" | jq -e '.workflow_id' > /dev/null 2>&1; then
        workflow_id=$(echo "$result" | jq -r '.workflow_id')
        echo "✓ Workflow submitted: $workflow_id"
        break
    fi
    
    echo "  Attempt $i failed, retrying..."
    sleep 1
done

if [ -z "$workflow_id" ]; then
    echo -e "${YELLOW}⚠ Could not submit workflow, skipping GET tests${NC}"
else
    # Test 3: GET /v1/workflows/{id} (查询状态)
    measure_latency "$SERVER_URL/v1/workflows/$workflow_id" "GET" "" "GET /v1/workflows/{id}"
    
    # Test 4: GET /v1/workflows/{id}/logs (获取日志) - if available
    if curl -sf "$SERVER_URL/v1/workflows/$workflow_id/logs" > /dev/null 2>&1; then
        measure_latency "$SERVER_URL/v1/workflows/$workflow_id/logs" "GET" "" "GET /v1/workflows/{id}/logs"
    fi
fi

echo "=== Benchmark Complete ==="
echo "Results saved to results/api_bench.csv"
echo ""

# Display summary
echo "=== Summary ==="
column -t -s, results/api_bench.csv

# Validate against targets (AC1)
echo ""
echo "=== Target Validation (AC1) ==="

validate_avg=$(awk -F',' 'NR==2 {print $4}' results/api_bench.csv)

if [ -n "$validate_avg" ]; then
    if (( $(echo "$validate_avg > 500" | bc -l) )); then
        echo -e "${RED}❌ POST /v1/validate avg ${validate_avg}ms exceeds 500ms target${NC}"
    else
        echo -e "${GREEN}✅ POST /v1/validate avg ${validate_avg}ms meets target (<500ms)${NC}"
    fi
fi

echo ""
echo "Note: For more accurate percentile measurements (P50, P95, P99),"
echo "install and use hey or vegeta (see test/performance/install_tools.sh)"
