#!/bin/bash
# Integration Test Runner Script
# Usage: ./scripts/run-integration-tests.sh [options]
#
# Options:
#   --build      Rebuild Docker images before testing
#   --no-cleanup Don't cleanup environment after tests
#   --verbose    Enable verbose output
#   --timeout    Test timeout in minutes (default: 10)

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COMPOSE_FILE="${PROJECT_ROOT}/deployments/docker-compose.test.yaml"
TEST_TIMEOUT=${TEST_TIMEOUT:-10m}
WATERFLOW_TEST_URL=${WATERFLOW_TEST_URL:-http://localhost:18080}

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Options
BUILD_IMAGES=false
CLEANUP=true
VERBOSE=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --build)
            BUILD_IMAGES=true
            shift
            ;;
        --no-cleanup)
            CLEANUP=false
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --timeout)
            TEST_TIMEOUT="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Cleanup function
cleanup() {
    if [ "$CLEANUP" = true ]; then
        log_info "Cleaning up test environment..."
        docker compose -f "${COMPOSE_FILE}" down -v --remove-orphans 2>/dev/null || true
        docker network rm waterflow-test-network 2>/dev/null || true
    else
        log_warning "Skipping cleanup (--no-cleanup specified)"
    fi
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    
    if ! command -v docker compose &> /dev/null; then
        log_error "Docker Compose is not installed"
        exit 1
    fi
    
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Build images if requested
build_images() {
    if [ "$BUILD_IMAGES" = true ]; then
        log_info "Building Docker images..."
        docker compose -f "${COMPOSE_FILE}" build
        log_success "Docker images built"
    fi
}

# Start test environment
start_environment() {
    log_info "Starting test environment..."
    
    # Clean any existing containers
    docker compose -f "${COMPOSE_FILE}" down -v --remove-orphans 2>/dev/null || true
    
    # Start services
    docker compose -f "${COMPOSE_FILE}" up -d
    
    log_info "Waiting for services to be healthy..."
}

# Wait for services to be ready
wait_for_services() {
    local max_wait=120
    local waited=0
    
    log_info "Waiting for PostgreSQL..."
    while ! docker compose -f "${COMPOSE_FILE}" exec -T postgresql pg_isready -U temporal &>/dev/null; do
        sleep 2
        waited=$((waited + 2))
        if [ $waited -ge $max_wait ]; then
            log_error "PostgreSQL failed to start within ${max_wait}s"
            docker compose -f "${COMPOSE_FILE}" logs postgresql
            exit 1
        fi
    done
    log_success "PostgreSQL is ready"
    
    log_info "Waiting for Temporal..."
    waited=0
    while ! docker compose -f "${COMPOSE_FILE}" exec -T temporal sh -c 'nc -z localhost 7233' &>/dev/null; do
        sleep 3
        waited=$((waited + 3))
        if [ $waited -ge $max_wait ]; then
            log_error "Temporal failed to start within ${max_wait}s"
            docker compose -f "${COMPOSE_FILE}" logs temporal
            exit 1
        fi
    done
    log_success "Temporal is ready"
    
    log_info "Waiting for Waterflow Server..."
    waited=0
    while ! curl -sf "${WATERFLOW_TEST_URL}/health" &>/dev/null; do
        sleep 2
        waited=$((waited + 2))
        if [ $waited -ge $max_wait ]; then
            log_error "Waterflow Server failed to start within ${max_wait}s"
            docker compose -f "${COMPOSE_FILE}" logs waterflow-server
            exit 1
        fi
    done
    log_success "Waterflow Server is ready"
    
    log_info "Waiting for Waterflow Agent..."
    sleep 5  # Give agent time to register
    log_success "Waterflow Agent should be ready"
    
    log_success "All services are healthy!"
}

# Run integration tests
run_tests() {
    log_info "Running integration tests (timeout: ${TEST_TIMEOUT})..."
    
    cd "${PROJECT_ROOT}"
    
    # Set test environment variables
    export WATERFLOW_TEST_URL="${WATERFLOW_TEST_URL}"
    export SERVER_URL="${WATERFLOW_TEST_URL}"  # Legacy support
    export WATERFLOW_AGENT_URL="${WATERFLOW_AGENT_URL:-http://localhost:18081}"
    export INTEGRATION_TEST=true
    
    # Run tests with build tags
    local test_args="-v -tags integration -timeout ${TEST_TIMEOUT}"
    if [ "$VERBOSE" = true ]; then
        test_args="${test_args} -v"
    fi
    
    if go test ${test_args} ./test/integration/...; then
        log_success "All integration tests passed!"
        return 0
    else
        log_error "Some integration tests failed"
        return 1
    fi
}

# Show service logs on failure
show_logs_on_failure() {
    log_warning "Showing service logs for debugging..."
    echo "=== PostgreSQL Logs ==="
    docker compose -f "${COMPOSE_FILE}" logs --tail=50 postgresql 2>/dev/null || true
    echo ""
    echo "=== Temporal Logs ==="
    docker compose -f "${COMPOSE_FILE}" logs --tail=50 temporal 2>/dev/null || true
    echo ""
    echo "=== Waterflow Server Logs ==="
    docker compose -f "${COMPOSE_FILE}" logs --tail=100 waterflow-server 2>/dev/null || true
    echo ""
    echo "=== Waterflow Agent Logs ==="
    docker compose -f "${COMPOSE_FILE}" logs --tail=100 waterflow-agent 2>/dev/null || true
}

# Main execution
main() {
    log_info "=========================================="
    log_info "  Waterflow Integration Test Runner"
    log_info "=========================================="
    echo ""
    
    check_prerequisites
    build_images
    start_environment
    wait_for_services
    
    echo ""
    log_info "=========================================="
    log_info "  Running Tests"
    log_info "=========================================="
    echo ""
    
    if run_tests; then
        echo ""
        log_success "=========================================="
        log_success "  All Integration Tests Passed!"
        log_success "=========================================="
        exit 0
    else
        echo ""
        show_logs_on_failure
        log_error "=========================================="
        log_error "  Integration Tests Failed!"
        log_error "=========================================="
        exit 1
    fi
}

main "$@"
