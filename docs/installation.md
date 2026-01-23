# Installation Guide

This guide covers all the ways to install Waterflow on your system.

## Quick Install

### One-Line Installation (Recommended)

```bash
# Install CLI (default)
curl -fsSL https://raw.githubusercontent.com/Websoft9/waterflow/main/scripts/install.sh | bash

# Install specific component
curl -fsSL https://raw.githubusercontent.com/Websoft9/waterflow/main/scripts/install.sh | bash -s -- -c server

# Install specific version
curl -fsSL https://raw.githubusercontent.com/Websoft9/waterflow/main/scripts/install.sh | bash -s -- -v v0.1.0

# Install all components
curl -fsSL https://raw.githubusercontent.com/Websoft9/waterflow/main/scripts/install.sh | bash -s -- -c all
```

### Installation Script Options

| Option | Description | Default |
|--------|-------------|---------|
| `-v, --version` | Version to install | `latest` |
| `-c, --component` | Component: `cli`, `server`, `agent`, `all` | `cli` |
| `-d, --dir` | Installation directory | `/usr/local/bin` |
| `-h, --help` | Show help message | - |

## Installation Methods

### 1. Docker (Recommended for Production)

The easiest way to run Waterflow in production.

#### Pull Images

```bash
# Server
docker pull websoft9/waterflow-server:latest

# Agent
docker pull websoft9/waterflow-agent:latest
```

#### Docker Compose (Full Stack)

```bash
# Clone repository
git clone https://github.com/Websoft9/waterflow.git
cd waterflow

# Start all services
docker compose -f deployments/docker-compose.yaml up -d

# Check status
docker compose -f deployments/docker-compose.yaml ps
```

#### Available Images

| Image | Registry | Platforms |
|-------|----------|-----------|
| `websoft9/waterflow-server` | Docker Hub | linux/amd64, linux/arm64 |
| `websoft9/waterflow-agent` | Docker Hub | linux/amd64, linux/arm64 |
| `ghcr.io/websoft9/waterflow-server` | GHCR | linux/amd64, linux/arm64 |
| `ghcr.io/websoft9/waterflow-agent` | GHCR | linux/amd64, linux/arm64 |

#### Image Tags

| Tag | Description |
|-----|-------------|
| `latest` | Latest stable release |
| `v1.0.0` | Specific version |
| `v1.0` | Latest patch of v1.0.x |
| `v1` | Latest minor of v1.x.x |
| `develop` | Development branch |

### 2. Binary Download

Download pre-built binaries from [GitHub Releases](https://github.com/Websoft9/waterflow/releases).

#### Linux (amd64)

```bash
# Download
VERSION=v0.1.0
curl -LO https://github.com/Websoft9/waterflow/releases/download/${VERSION}/waterflow-cli-linux-amd64

# Install
chmod +x waterflow-cli-linux-amd64
sudo mv waterflow-cli-linux-amd64 /usr/local/bin/waterflow

# Verify
waterflow version
```

#### Linux (arm64)

```bash
VERSION=v0.1.0
curl -LO https://github.com/Websoft9/waterflow/releases/download/${VERSION}/waterflow-cli-linux-arm64
chmod +x waterflow-cli-linux-arm64
sudo mv waterflow-cli-linux-arm64 /usr/local/bin/waterflow
```

#### macOS (Intel)

```bash
VERSION=v0.1.0
curl -LO https://github.com/Websoft9/waterflow/releases/download/${VERSION}/waterflow-cli-darwin-amd64
chmod +x waterflow-cli-darwin-amd64
sudo mv waterflow-cli-darwin-amd64 /usr/local/bin/waterflow
```

#### macOS (Apple Silicon)

```bash
VERSION=v0.1.0
curl -LO https://github.com/Websoft9/waterflow/releases/download/${VERSION}/waterflow-cli-darwin-arm64
chmod +x waterflow-cli-darwin-arm64
sudo mv waterflow-cli-darwin-arm64 /usr/local/bin/waterflow
```

#### Windows

Download from GitHub Releases and add to PATH:
- `waterflow-cli-windows-amd64.exe`
- `waterflow-server-windows-amd64.exe`
- `waterflow-agent-windows-amd64.exe`

### 3. Go Install

If you have Go 1.21+ installed:

```bash
# Install CLI
go install github.com/Websoft9/waterflow/cmd/waterflow-cli@latest

# Install Server
go install github.com/Websoft9/waterflow/cmd/server@latest

# Install Agent
go install github.com/Websoft9/waterflow/cmd/agent@latest
```

### 4. Build from Source

```bash
# Clone repository
git clone https://github.com/Websoft9/waterflow.git
cd waterflow

# Build all binaries
make build-all

# Or build individually
make build        # Server
make build-agent  # Agent
make cli          # CLI

# Binaries are in ./bin/
ls -la bin/
```

## Verifying Installation

### Check Version

```bash
# CLI
waterflow version

# Server
./bin/server --version

# Agent
./bin/agent --version
```

### Verify Checksums

```bash
# Download checksums
VERSION=v0.1.0
curl -LO https://github.com/Websoft9/waterflow/releases/download/${VERSION}/checksums.txt

# Verify
sha256sum -c checksums.txt
```

## Component Overview

| Component | Purpose | Required |
|-----------|---------|----------|
| Server | Workflow orchestration, API | Yes |
| Agent | Task execution on target servers | Yes (at least 1) |
| CLI | Command-line management tool | Optional |

## System Requirements

### Minimum Requirements

| Component | CPU | Memory | Disk |
|-----------|-----|--------|------|
| Server | 1 core | 512MB | 100MB |
| Agent | 1 core | 256MB | 50MB |

### Recommended for Production

| Component | CPU | Memory | Disk |
|-----------|-----|--------|------|
| Server | 2+ cores | 2GB+ | 10GB+ |
| Agent | 2+ cores | 1GB+ | 5GB+ |

### Dependencies

- **Temporal**: Workflow execution backend (included in Docker Compose)
- **PostgreSQL**: Temporal persistence (included in Docker Compose)

## Configuration

### Server Configuration

Create `/etc/waterflow/server.yaml`:

```yaml
server:
  port: 8080
  host: "0.0.0.0"

temporal:
  host: "localhost:7233"
  namespace: "default"

database:
  host: "localhost"
  port: 5432
```

### Agent Configuration

Create `/etc/waterflow/agent.yaml`:

```yaml
agent:
  server_url: "http://localhost:8080"
  task_queue: "linux-amd64"
  poll_interval: "1s"

temporal:
  host: "localhost:7233"
```

See [Configuration Guide](configuration.md) for complete options.

## Uninstallation

### Remove Binaries

```bash
# CLI
sudo rm /usr/local/bin/waterflow

# Server
sudo rm /usr/local/bin/waterflow-server

# Agent
sudo rm /usr/local/bin/waterflow-agent
```

### Remove Docker

```bash
# Stop and remove containers
docker compose -f deployments/docker-compose.yaml down -v

# Remove images
docker rmi websoft9/waterflow-server websoft9/waterflow-agent
```

### Remove Configuration

```bash
sudo rm -rf /etc/waterflow
```

## Troubleshooting

### Common Issues

#### "Permission denied" during installation

```bash
# Use sudo or install to user directory
./install.sh -d ~/.local/bin
```

#### Binary not found after installation

```bash
# Add to PATH
export PATH=$PATH:/usr/local/bin
# Add to ~/.bashrc or ~/.zshrc for persistence
```

#### Connection refused

```bash
# Check if server is running
curl http://localhost:8080/health

# Check Docker containers
docker ps | grep waterflow
```

### Getting Help

- [Documentation](https://github.com/Websoft9/waterflow/tree/main/docs)
- [Troubleshooting Guide](troubleshooting.md)
- [GitHub Issues](https://github.com/Websoft9/waterflow/issues)

## Next Steps

1. [Quick Start Guide](quick-start.md) - Run your first workflow
2. [YAML DSL Reference](yaml-dsl-reference.md) - Learn the workflow syntax
3. [Deployment Guide](deployment.md) - Production deployment
