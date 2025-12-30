.PHONY: build test clean install integration-test

PLUGIN_NAME = transfer.so

build:
	@echo "Checking Go version..."
	@go version | grep -q "go1.2[2-9]" || (echo "Error: Go 1.22+ required" && exit 1)
	@echo "Building plugin..."
	CGO_ENABLED=1 go build -buildmode=plugin -o $(PLUGIN_NAME) main.go
	@echo "✓ Plugin built: $(PLUGIN_NAME)"
	@ls -lh $(PLUGIN_NAME)

test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out
	@echo "\n=== Coverage Report ==="
	go tool cover -func=coverage.out
	@echo "✓ Tests passed"

integration-test: build
	@echo "Starting SFTP test server..."
	@docker run -d --name sftp-test -p 2222:22 \
		-v /tmp/sftp-upload:/home/testuser/upload \
		atmoz/sftp testuser:testpass:::upload || true
	@sleep 2
	@echo "Running integration tests..."
	go test -v -tags=integration -timeout 5m
	@echo "Stopping SFTP test server..."
	@docker rm -f sftp-test || true
	@echo "✓ Integration tests passed"

clean:
	rm -f $(PLUGIN_NAME) coverage.out
	@echo "✓ Cleaned"

install: build
	@echo "Installing plugin..."
	mkdir -p /opt/waterflow/plugins
	cp $(PLUGIN_NAME) /opt/waterflow/plugins/
	@echo "✓ Installed to /opt/waterflow/plugins/$(PLUGIN_NAME)"

.DEFAULT_GOAL := build
