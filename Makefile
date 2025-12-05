# Waterflow Makefile
# Comprehensive build, test, and development tasks

.PHONY: help build test lint clean dev-setup dev run docker-build docker-run docs install release

# Default target
help: ## Show this help message
	@echo "Waterflow Development Makefile"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development setup
dev-setup: ## Set up development environment
	@echo "Setting up development environment..."
	@./scripts/dev-setup.sh

# Build targets
build: ## Build Waterflow binary
	@echo "Building Waterflow..."
	@go build -o bin/waterflow ./cmd/waterflow

build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@./scripts/build.sh

# Testing targets
test: ## Run all tests
	@echo "Running test suite..."
	@./scripts/test.sh

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	@go test -v -race -cover ./...

test-integration: ## Run integration tests only
	@echo "Running integration tests..."
	@go test -v -tags=integration ./test/integration/...

test-bench: ## Run benchmark tests
	@echo "Running benchmark tests..."
	@go test -bench=. -benchmem ./...

# Code quality targets
lint: ## Run linting
	@echo "Running linters..."
	@golangci-lint run

fmt: ## Format code
	@echo "Formatting code..."
	@gofmt -s -w .
	@goimports -w .

# Development server
dev: ## Start development server with hot reload
	@echo "Starting development server..."
	@air

# Docker targets
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t waterflow:dev .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@docker run --rm -p 8080:8080 waterflow:dev

docker-compose-up: ## Start development environment with docker-compose
	@echo "Starting development environment..."
	@cd docker && docker-compose up -d

docker-compose-down: ## Stop development environment
	@echo "Stopping development environment..."
	@cd docker && docker-compose down

docker-compose-logs: ## Show docker-compose logs
	@echo "Showing logs..."
	@cd docker && docker-compose logs -f

docker-compose-test: ## Run tests with docker-compose
	@echo "Running tests with docker-compose..."
	@cd docker && docker-compose --profile test up --abort-on-container-exit

docker-compose-prod: ## Start production environment
	@echo "Starting production environment..."
	@cd docker && docker-compose -f docker-compose.prod.yml --profile prod up -d

# Documentation
docs: ## Generate documentation
	@echo "Generating documentation..."
	@go generate ./docs/...

docs-serve: ## Serve documentation locally
	@echo "Serving documentation..."
	@cd docs && python -m http.server 8000

# Installation
install: ## Install Waterflow to system
	@echo "Installing Waterflow..."
	@go install ./cmd/waterflow

# Release targets
release: ## Create a new release
	@echo "Creating release..."
	@./scripts/release.sh

release-dry-run: ## Test release process without publishing
	@echo "Testing release process..."
	@./scripts/release.sh --dry-run

# Cleanup
clean: ## Clean build artifacts
	@echo "Cleaning up..."
	@rm -rf bin/
	@rm -rf dist/
	@rm -rf test/results/
	@rm -rf .tmp/
	@go clean ./...

clean-all: clean ## Clean all artifacts including dependencies
	@echo "Deep cleaning..."
	@go clean -cache
	@go clean -testcache
	@go clean -modcache

# Utility targets
version: ## Show version information
	@echo "Waterflow Development Build"
	@go version
	@echo "Commit: $$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
	@echo "Date: $$(date -u +%Y-%m-%dT%H:%M:%SZ)"

deps: ## Download and tidy dependencies
	@echo "Managing dependencies..."
	@go mod download
	@go mod tidy

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

# CI/CD targets
ci: lint test build ## Run CI pipeline locally

# Development workflow
workflow-init: ## Initialize development workflow
	@echo "Initializing development workflow..."
	@mkdir -p workflows/
	@mkdir -p config/
	@cp examples/hello-world.yaml workflows/
	@echo "✅ Development workflow initialized"

# Help for development
dev-help: ## Show development-specific help
	@echo "Development Workflow:"
	@echo "1. make dev-setup          # Initial setup"
	@echo "2. make dev               # Start development server"
	@echo "3. make test              # Run tests"
	@echo "4. make build             # Build binary"
	@echo "5. make docker-build      # Build Docker image"
	@echo ""
	@echo "Useful commands:"
	@echo "- make clean              # Clean artifacts"
	@echo "- make lint               # Check code quality"
	@echo "- make fmt                # Format code"
	@echo "- make version            # Show version info"