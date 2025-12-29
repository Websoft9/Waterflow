.PHONY: build test coverage lint fmt run docker-build clean help

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Docker image configuration
DOCKER_REGISTRY ?= docker.io
DOCKER_REPO ?= waterflow
IMAGE_NAME_SERVER = $(DOCKER_REGISTRY)/$(DOCKER_REPO)/server
IMAGE_NAME_AGENT = $(DOCKER_REGISTRY)/$(DOCKER_REPO)/agent
TAG_VERSION = $(VERSION)
TAG_LATEST = latest

# Build flags
LDFLAGS := -ldflags "\
	-X main.Version=$(VERSION) \
	-X main.Commit=$(COMMIT) \
	-X main.BuildTime=$(BUILD_TIME)"

# Build directories
BIN_DIR := bin
BUILD_DIR := build

# Binary names
SERVER_BINARY_NAME := server
AGENT_BINARY_NAME := agent

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
	@echo "Building $(SERVER_BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(SERVER_BINARY_NAME) cmd/server/main.go
	@echo "Build complete: $(BIN_DIR)/$(SERVER_BINARY_NAME)"
	@echo "Version: $(VERSION), Commit: $(COMMIT), Build Time: $(BUILD_TIME)"

## build-agent: Compile agent binary with version information
build-agent:
	@echo "Building $(AGENT_BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(AGENT_BINARY_NAME) cmd/agent/main.go
	@echo "Build complete: $(BIN_DIR)/$(AGENT_BINARY_NAME)"
	@echo "Version: $(VERSION), Commit: $(COMMIT), Build Time: $(BUILD_TIME)"

## build-all: Compile both server and agent binaries
build-all: build build-agent

## test: Run unit tests only (skip integration tests)
test:
	@echo "Running unit tests (skipping integration tests)..."
	go test -v -race -short ./...

## test-integration: Run all tests including integration tests (requires Temporal server)
test-integration:
	@echo "Running all tests including integration tests..."
	go test -v -race ./...

## coverage: Generate test coverage report (unit tests only)
coverage:
	@echo "Generating coverage report..."
	go test -v -short -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}' || echo "Coverage report generated successfully"

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
	./$(BIN_DIR)/$(SERVER_BINARY_NAME)

## run-agent: Run agent with default config
run-agent: build-agent
	@echo "Running $(AGENT_BINARY_NAME)..."
	./$(BIN_DIR)/$(AGENT_BINARY_NAME) --config config.agent.example.yaml

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t waterflow:$(VERSION) .
	docker tag waterflow:$(VERSION) waterflow:latest
	@echo "Docker image built: waterflow:$(VERSION)"

## docker-agent: Build Agent Docker image
docker-agent:
	@echo "Building Agent Docker image..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		-f build/Dockerfile.agent \
		-t $(IMAGE_NAME_AGENT):$(TAG_VERSION) \
		-t $(IMAGE_NAME_AGENT):$(TAG_LATEST) \
		.
	@echo "Agent image built: $(IMAGE_NAME_AGENT):$(TAG_VERSION)"

## docker-agent-push: Build and push Agent image to registry
docker-agent-push: docker-agent
	@echo "Pushing Agent image..."
	docker push $(IMAGE_NAME_AGENT):$(TAG_VERSION)
	docker push $(IMAGE_NAME_AGENT):$(TAG_LATEST)
	@echo "Agent image pushed"

## docker-agent-run: Run Agent container locally for testing
docker-agent-run:
	@echo "Running Agent container..."
	docker run --rm -it \
		-e TEMPORAL_SERVER_URL=host.docker.internal:7233 \
		-e TASK_QUEUES=linux-amd64 \
		-e LOG_LEVEL=debug \
		$(IMAGE_NAME_AGENT):$(TAG_LATEST)

## docker-all: Build both Server and Agent images
docker-all: docker-build docker-agent

## docker-push: Push both Server and Agent images
docker-push: docker-build docker-agent-push
	@echo "Pushing Server image..."
	docker push waterflow:$(VERSION)
	docker push waterflow:latest

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
