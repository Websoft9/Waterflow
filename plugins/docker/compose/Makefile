.PHONY: build test clean install check-compose integration-test

PLUGIN_NAME = compose.so

check-compose:
	@(docker compose version > /dev/null 2>&1 || docker-compose --version > /dev/null 2>&1) || \
		(echo "Error: Docker Compose not installed" && exit 1)

build:
	@go version | grep -q "go1.2[2-9]" || (echo "Error: Go 1.22+ required" && exit 1)
	CGO_ENABLED=1 go build -buildmode=plugin -o $(PLUGIN_NAME) main.go

test:
	go test -v -short -race -coverprofile=coverage.out
	go tool cover -func=coverage.out

integration-test: build check-compose
	go test -v -race -tags=integration

clean:
	rm -f $(PLUGIN_NAME) coverage.out

install: build
	mkdir -p /opt/waterflow/plugins
	cp $(PLUGIN_NAME) /opt/waterflow/plugins/

.DEFAULT_GOAL := build
