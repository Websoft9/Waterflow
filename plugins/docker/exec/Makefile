.PHONY: build test test-short test-integration clean install check-docker help

PLUGIN_NAME = exec.so
GO_FILES = main.go
TEST_FILES = main_test.go integration_test.go

help:
	@echo "Docker Exec Node Plugin - Makefile targets:"
	@echo "  make build              - Build plugin (.so file)"
	@echo "  make test               - Run all tests (unit + integration)"
	@echo "  make test-short         - Run unit tests only (fast)"
	@echo "  make test-integration   - Run integration tests (requires Docker)"
	@echo "  make coverage           - Generate complete coverage report"
	@echo "  make check-docker       - Verify Docker environment"
	@echo "  make clean              - Remove build artifacts"
	@echo "  make install            - Install plugin to system directory"

check-docker:
	@echo "Checking Docker environment..."
	@which docker > /dev/null || (echo "❌ Error: Docker not installed" && exit 1)
	@docker info > /dev/null 2>&1 || (echo "❌ Error: Docker daemon not running or permission denied" && exit 1)
	@echo "✅ Docker is available"

build:
	@echo "Building docker/exec plugin..."
	@go version | grep -E "go1\.(2[2-9]|[3-9][0-9])" > /dev/null || (echo "❌ Error: Go 1.22+ required" && exit 1)
	CGO_ENABLED=1 go build -buildmode=plugin -o $(PLUGIN_NAME) $(GO_FILES)
	@echo "✅ Plugin built: $(PLUGIN_NAME)"

test-short:
	@echo "Running unit tests..."
	go test -v -short -race -coverprofile=coverage.out
	@echo ""
	@echo "Coverage summary:"
	@go tool cover -func=coverage.out | grep total

test-integration: build check-docker
	@echo "Running integration tests (requires Docker)..."
	go test -v -race -tags=integration
	@echo "✅ Integration tests passed"

test: test-short test-integration
	@echo "✅ All tests passed"

coverage: check-docker
	@echo "Generating complete coverage report (unit + integration)..."
	go test -v -race -tags=integration -coverprofile=coverage-full.out
	@echo ""
	@echo "Coverage summary:"
	@go tool cover -func=coverage-full.out | grep total
	@echo ""
	@echo "Generate HTML report: go tool cover -html=coverage-full.out"

clean:
	@echo "Cleaning build artifacts..."
	rm -f $(PLUGIN_NAME) coverage.out coverage-full.out
	@echo "✅ Clean complete"

install: build
	@echo "Installing plugin to /opt/waterflow/plugins..."
	@mkdir -p /opt/waterflow/plugins
	@cp $(PLUGIN_NAME) /opt/waterflow/plugins/
	@echo "✅ Plugin installed"

.DEFAULT_GOAL := build
