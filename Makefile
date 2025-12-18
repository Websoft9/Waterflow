.PHONY: build test coverage lint fmt run docker-build clean help

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS := -ldflags "\
	-X main.Version=$(VERSION) \
	-X main.Commit=$(COMMIT) \
	-X main.BuildTime=$(BUILD_TIME)"

# Build directories
BIN_DIR := bin
BUILD_DIR := build

# Binary name
BINARY_NAME := server

## help: Display this help message
help:
	@echo "Waterflow Server - Build Targets"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' Makefile | sed 's/^## /  /'

## build: Compile server binary with version information
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) cmd/server/main.go
	@echo "Build complete: $(BIN_DIR)/$(BINARY_NAME)"
	@echo "Version: $(VERSION), Commit: $(COMMIT), Build Time: $(BUILD_TIME)"

## test: Run all tests
test:
	@echo "Running tests..."
	go test -v -race ./...

## coverage: Generate test coverage report
coverage:
	@echo "Generating coverage report..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'

## lint: Run code linters
lint:
	@echo "Running linters..."
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "golangci-lint not found. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	gofmt -s -w .

## run: Run server locally
run: build
	@echo "Starting server..."
	./$(BIN_DIR)/$(BINARY_NAME)

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t waterflow:$(VERSION) .
	docker tag waterflow:$(VERSION) waterflow:latest
	@echo "Docker image built: waterflow:$(VERSION)"

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BIN_DIR) $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "Clean complete"

## tidy: Tidy and verify module dependencies
tidy:
	@echo "Tidying module dependencies..."
	go mod tidy
	go mod verify

## install-tools: Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed successfully"
