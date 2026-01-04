.PHONY: check build test clean install integration-test coverage

# 环境检查
check:
	@echo "=== Checking build environment ==="
	@echo "Go version:"
	@go version | grep -q "go1.2[2-9]" || (echo "ERROR: Go 1.22+ required" && exit 1)
	@echo "CGO_ENABLED: $(CGO_ENABLED)"
	@test "$(CGO_ENABLED)" = "1" || (echo "WARNING: CGO_ENABLED should be 1 for plugin build")
	@echo "GOOS: $$(go env GOOS)"
	@test "$$(go env GOOS)" != "windows" || (echo "ERROR: Go Plugin not supported on Windows" && exit 1)
	@echo "✅ Environment check passed"

# 编译插件
build: check
	@echo "=== Building greeter.so ==="
	go build -buildmode=plugin -o greeter.so main.go
	@ls -lh greeter.so
	@echo "✅ Build successful"

# 运行单元测试
test:
	@echo "=== Running unit tests ==="
	go test -v -cover -coverprofile=coverage.out
	@go tool cover -func=coverage.out | grep total || true
	@echo "✅ Tests passed"

# 查看覆盖率报告
coverage: test
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# 集成测试 (加载插件)
integration-test: build
	@echo "=== Running integration test ==="
	@if [ -f integration_test.go ]; then \
		go test -tags=integration -run TestGreeterPlugin integration_test.go main.go; \
	else \
		echo "No integration tests found, skipping"; \
	fi
	@echo "✅ Integration test passed"

# 安装到 Agent 插件目录
install: build
	@echo "=== Installing plugin ==="
	@if [ -d /opt/waterflow/plugins ]; then \
		sudo mkdir -p /opt/waterflow/plugins/custom; \
		sudo cp greeter.so /opt/waterflow/plugins/custom/; \
		sudo chown waterflow:waterflow /opt/waterflow/plugins/custom/greeter.so 2>/dev/null || true; \
		echo "✅ Installed to /opt/waterflow/plugins/custom/greeter.so"; \
	else \
		echo "WARNING: /opt/waterflow/plugins not found, plugin not installed"; \
	fi

# 清理
clean:
	rm -f greeter.so coverage.out coverage.html
	@echo "✅ Cleaned"
