.PHONY: check build test clean

check:
	@echo "Checking build environment..."
	@go version
	@echo "CGO_ENABLED=$(CGO_ENABLED)"

build: check
	@echo "Building echo.so..."
	go build -buildmode=plugin -o echo.so main.go
	@ls -lh echo.so
	@echo "Build successful"

test:
	go test -v -cover

clean:
	rm -f echo.so
