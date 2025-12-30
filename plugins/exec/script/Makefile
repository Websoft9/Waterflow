.PHONY: build test clean install check

PLUGIN_NAME = script.so
GO_VERSION_MIN = 1.22

check:
	@echo "Checking Go version..."
	@go version | grep -q "go1.2[2-9]\|go1.[3-9][0-9]" || (echo "Error: Go $(GO_VERSION_MIN)+ required" && exit 1)
	@echo "Checking CGO..."
	@if [ -z "$${CGO_ENABLED}" ] || [ "$${CGO_ENABLED}" != "1" ]; then \
		echo "Warning: CGO_ENABLED should be set to 1"; \
		echo "Run: export CGO_ENABLED=1"; \
	fi
	@echo "✓ Environment checks passed"

build: check
	@echo "Building plugin..."
	CGO_ENABLED=1 go build -buildmode=plugin -o $(PLUGIN_NAME) main.go
	@echo "✓ Plugin built: $(PLUGIN_NAME)"

test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out
	@echo "\n=== Coverage Report ==="
	@go tool cover -func=coverage.out | tail -1
	@echo "✓ Tests passed"

integration-test: build
	@echo "Running integration tests..."
	go test -v -tags=integration -run TestScriptPlugin
	@echo "✓ Integration tests passed"

clean:
	rm -f $(PLUGIN_NAME) coverage.out
	@echo "✓ Cleaned"

install: build
	@echo "Installing plugin..."
	@mkdir -p /opt/waterflow/plugins
	cp $(PLUGIN_NAME) /opt/waterflow/plugins/
	@echo "✓ Installed to /opt/waterflow/plugins/$(PLUGIN_NAME)"

.DEFAULT_GOAL := build
