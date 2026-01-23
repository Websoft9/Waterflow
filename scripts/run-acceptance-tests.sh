#!/bin/bash
# Acceptance Test Runner for Waterflow
# Runs PRD-defined acceptance scenarios with full environment setup
# Usage: ./scripts/run-acceptance-tests.sh [--keep-env] [--scenario <name>]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COMPOSE_FILE="test/acceptance/docker-compose.acceptance.yaml"
REPORT_DIR="test/acceptance/reports"
TIMEOUT_HEALTH=180
TIMEOUT_TESTS=900  # 15 minutes for acceptance tests

# Parse arguments
KEEP_ENV=false
SCENARIO=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --keep-env)
            KEEP_ENV=true
            shift
            ;;
        --scenario)
            SCENARIO="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

cleanup() {
    if [ "$KEEP_ENV" = false ]; then
        log_info "Cleaning up acceptance test environment..."
        docker compose -f "$COMPOSE_FILE" down -v --remove-orphans 2>/dev/null || true
        log_success "Environment cleaned up"
    else
        log_warn "Keeping environment running (--keep-env specified)"
    fi
}

wait_for_service() {
    local service_name=$1
    local health_url=$2
    local timeout=$3
    local elapsed=0
    
    log_info "Waiting for $service_name to be healthy..."
    
    while [ $elapsed -lt $timeout ]; do
        if curl -sf "$health_url" > /dev/null 2>&1; then
            log_success "$service_name is healthy"
            return 0
        fi
        sleep 3
        elapsed=$((elapsed + 3))
        echo -n "."
    done
    
    echo ""
    log_error "$service_name failed to become healthy within ${timeout}s"
    return 1
}

generate_report() {
    local status=$1
    local start_time=$2
    local end_time=$3
    local report_file="$REPORT_DIR/acceptance-report-$(date +%Y%m%d-%H%M%S).md"
    
    mkdir -p "$REPORT_DIR"
    
    local duration=$((end_time - start_time))
    local minutes=$((duration / 60))
    local seconds=$((duration % 60))
    
    cat > "$report_file" << EOF
# Waterflow Acceptance Test Report

**Date:** $(date -u '+%Y-%m-%d %H:%M:%S UTC')
**Version:** $(git describe --tags --always 2>/dev/null || echo "dev")
**Environment:** Docker Compose (Acceptance)
**Duration:** ${minutes}m ${seconds}s

## Summary

| Metric | Value |
|--------|-------|
| Overall Status | $([ "$status" = "0" ] && echo "✅ PASSED" || echo "❌ FAILED") |
| Total Duration | ${minutes}m ${seconds}s |
| Environment | Docker Compose |

## Test Output

\`\`\`
$(cat "$REPORT_DIR/test-output.txt" 2>/dev/null || echo "No output captured")
\`\`\`

## Environment Details

- Server URL: http://localhost:18080
- Temporal: localhost:17233
- Agents: 3 (web-1, web-2, db-1)

## Logs

Logs are available in:
- Server: \`docker compose -f $COMPOSE_FILE logs waterflow-server\`
- Agents: \`docker compose -f $COMPOSE_FILE logs agent-web-1 agent-web-2 agent-db-1\`
EOF

    log_info "Report generated: $report_file"
    echo "$report_file"
}

# Main execution
main() {
    log_info "=========================================="
    log_info "Waterflow Acceptance Test Runner"
    log_info "=========================================="
    
    # Record start time
    START_TIME=$(date +%s)
    
    # Setup trap for cleanup
    trap cleanup EXIT
    
    # Check prerequisites
    log_info "Checking prerequisites..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    
    if ! docker compose version &> /dev/null; then
        log_error "Docker Compose is not installed"
        exit 1
    fi
    
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
    
    # Start environment
    log_info "Starting acceptance test environment..."
    docker compose -f "$COMPOSE_FILE" up -d --build
    
    # Wait for services
    log_info "Waiting for services to be healthy..."
    
    # Wait for Temporal (via direct health check)
    local temporal_healthy=false
    for i in {1..60}; do
        if docker compose -f "$COMPOSE_FILE" exec -T temporal sh -c 'nc -z localhost 7233' 2>/dev/null; then
            temporal_healthy=true
            log_success "Temporal is healthy"
            break
        fi
        sleep 3
        echo -n "."
    done
    
    if [ "$temporal_healthy" = false ]; then
        log_error "Temporal failed to become healthy"
        exit 1
    fi
    
    # Wait for Server
    if ! wait_for_service "Waterflow Server" "http://localhost:18080/health" $TIMEOUT_HEALTH; then
        log_error "Server health check failed"
        docker compose -f "$COMPOSE_FILE" logs waterflow-server
        exit 1
    fi
    
    # Wait for Agents to register (give them time to connect)
    log_info "Waiting for agents to register..."
    sleep 10
    log_success "Agents should be registered"
    
    # Create report directory
    mkdir -p "$REPORT_DIR"
    
    # Run acceptance tests
    log_info "Running acceptance tests..."
    
    TEST_ARGS="-v -tags acceptance -timeout ${TIMEOUT_TESTS}s"
    if [ -n "$SCENARIO" ]; then
        TEST_ARGS="$TEST_ARGS -run $SCENARIO"
    fi
    
    set +e
    SERVER_URL=http://localhost:18080 \
    TEMPORAL_HOST=localhost:17233 \
    go test $TEST_ARGS ./test/acceptance/... 2>&1 | tee "$REPORT_DIR/test-output.txt"
    TEST_EXIT_CODE=${PIPESTATUS[0]}
    set -e
    
    # Record end time
    END_TIME=$(date +%s)
    
    # Generate report
    REPORT_FILE=$(generate_report $TEST_EXIT_CODE $START_TIME $END_TIME)
    
    # Summary
    echo ""
    log_info "=========================================="
    if [ $TEST_EXIT_CODE -eq 0 ]; then
        log_success "All acceptance tests PASSED"
    else
        log_error "Some acceptance tests FAILED"
    fi
    log_info "Report: $REPORT_FILE"
    log_info "=========================================="
    
    exit $TEST_EXIT_CODE
}

main "$@"
