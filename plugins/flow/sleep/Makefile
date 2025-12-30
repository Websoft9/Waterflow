.PHONY: build test clean install coverage integration-test

PLUGIN_NAME = sleep.so
GO_VERSION_MIN = 1.22

build:
	@echo "Checking Go version..."
	@go version | grep -qE "go1\.(2[2-9]|[3-9][0-9])" || (echo "Error: Go $(GO_VERSION_MIN)+ required" && exit 1)
	@echo "Building plugin..."
	CGO_ENABLED=1 go build -buildmode=plugin -o $(PLUGIN_NAME) main.go
	@echo "✓ Plugin built: $(PLUGIN_NAME)"
	@ls -lh $(PLUGIN_NAME)

test:
	@echo "Running unit tests..."
	go test -v -race -coverprofile=coverage.out
	@echo ""
	@echo "=== Coverage Report ==="
	@go tool cover -func=coverage.out | tail -1
	@echo "✓ Tests passed"

coverage: test
	@echo ""
	@echo "=== Detailed Coverage Report ==="
	@go tool cover -func=coverage.out
	@echo ""
	@echo "Opening coverage in browser..."
	@go tool cover -html=coverage.out

integration-test: build
	@echo "Running integration tests..."
	@go test -v -tags=integration ./...
	@echo "✓ Integration tests passed"

clean:
	rm -f $(PLUGIN_NAME) coverage.out
	@echo "✓ Cleaned"

install: build
	@echo "Installing plugin..."
	@mkdir -p /opt/waterflow/plugins
	@cp $(PLUGIN_NAME) /opt/waterflow/plugins/
	@echo "✓ Installed to /opt/waterflow/plugins/$(PLUGIN_NAME)"

.DEFAULT_GOAL := build
