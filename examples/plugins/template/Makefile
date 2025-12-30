.PHONY: check build test clean

GO_VERSION := $(shell go version | awk '{print $$3}')
CGO_ENABLED := $(shell go env CGO_ENABLED)
GOOS := $(shell go env GOOS)

check:
	@echo "Checking build environment..."
	@if [ "$(CGO_ENABLED)" != "1" ]; then \
		echo "ERROR: CGO_ENABLED must be 1 for plugin builds"; \
		echo "Run: export CGO_ENABLED=1"; \
		exit 1; \
	fi
	@if [ "$(GOOS)" = "windows" ]; then \
		echo "ERROR: Go plugins not supported on Windows"; \
		exit 1; \
	fi
	@echo "✓ Go version: $(GO_VERSION)"
	@echo "✓ CGO_ENABLED: $(CGO_ENABLED)"
	@echo "✓ GOOS: $(GOOS)"

build: check
	@echo "Building template.so plugin..."
	go build -buildmode=plugin -o template.so main.go
	@ls -lh template.so
	@echo "✓ Build successful"

test:
	@echo "No tests for template (copy to create real nodes)"

clean:
	rm -f template.so
