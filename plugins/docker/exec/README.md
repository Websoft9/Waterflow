# Docker Exec Node (docker/exec)

Execute Docker CLI commands on Waterflow agent servers.

## Overview

The `docker/exec` node provides a universal interface to execute Docker CLI commands. It supports all Docker subcommands (run, ps, stop, rm, images, pull, exec, logs, etc.) and handles error classification for Temporal retry logic.

## Prerequisites

**CRITICAL:** The agent server must have Docker installed and properly configured:

1. **Docker Installation**: Docker Engine must be installed
2. **User Permissions**: Agent user must have Docker socket access
3. **Docker Daemon**: Docker daemon must be running

### Permission Setup

```bash
# Add agent user to docker group
sudo usermod -aG docker waterflow-agent

# Re-login or use newgrp to activate group
newgrp docker

# Verify permissions (should work without sudo)
docker ps
```

### Verify Docker Availability

```bash
# Check Docker installation
which docker

# Check daemon status
systemctl status docker

# Test permissions
docker info
```

## Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `command` | string | Yes | - | Docker subcommand (e.g., 'run', 'ps', 'stop') |
| `args` | array | No | `[]` | Command arguments |
| `timeout` | string | No | `"5m"` | Command timeout (e.g., '30s', '5m', '10m') |
| `docker_host` | string | No | `"unix:///var/run/docker.sock"` | Docker daemon socket |

## Outputs

| Output | Type | Description |
|--------|------|-------------|
| `exit_code` | int | Command exit code (0 = success) |
| `stdout` | string | Standard output |
| `stderr` | string | Standard error output |
| `elapsed_ms` | int | Execution time in milliseconds |
| `command_line` | string | Full command executed |

## Error Classification

The node automatically classifies errors for Temporal retry behavior:

### Permanent Errors (Non-Retriable)
- Docker not installed
- Docker daemon not running
- Permission denied
- Container not found
- Invalid command syntax

### Temporary Errors (Retriable)
- Image not found (may need pull)
- Network errors
- Timeout errors

## Usage Examples

### Example 1: Pull an Image

```yaml
steps:
  - name: Pull nginx
    uses: docker/exec@v1
    with:
      command: pull
      args: ["nginx:latest"]
      timeout: 10m
```

### Example 2: Run a Container

```yaml
steps:
  - name: Start nginx container
    uses: docker/exec@v1
    with:
      command: run
      args:
        - "-d"
        - "--name"
        - "web-server"
        - "-p"
        - "8080:80"
        - "nginx:latest"
```

### Example 3: List Containers

```yaml
steps:
  - name: List running containers
    uses: docker/exec@v1
    with:
      command: ps
      args: ["--filter", "status=running"]
```

### Example 4: Execute Command in Container

```yaml
steps:
  - name: Check nginx version
    uses: docker/exec@v1
    with:
      command: exec
      args: ["web-server", "nginx", "-v"]
```

### Example 5: View Container Logs

```yaml
steps:
  - name: Get logs
    uses: docker/exec@v1
    with:
      command: logs
      args: ["--tail", "100", "web-server"]
```

### Example 6: Stop Container

```yaml
steps:
  - name: Stop container
    uses: docker/exec@v1
    with:
      command: stop
      args: ["web-server"]
```

### Example 7: Remove Container

```yaml
steps:
  - name: Remove container
    uses: docker/exec@v1
    with:
      command: rm
      args: ["-f", "web-server"]
```

### Example 8: Container Lifecycle Management

```yaml
name: Container Deployment
steps:
  - name: Pull image
    uses: docker/exec@v1
    with:
      command: pull
      args: ["myapp:v1.0"]
      timeout: 10m

  - name: Stop old container
    uses: docker/exec@v1
    with:
      command: stop
      args: ["myapp"]
    continue-on-error: true

  - name: Remove old container
    uses: docker/exec@v1
    with:
      command: rm
      args: ["myapp"]
    continue-on-error: true

  - name: Start new container
    uses: docker/exec@v1
    with:
      command: run
      args:
        - "-d"
        - "--name"
        - "myapp"
        - "--restart"
        - "unless-stopped"
        - "-p"
        - "3000:3000"
        - "-e"
        - "NODE_ENV=production"
        - "myapp:v1.0"

  - name: Verify container
    uses: docker/exec@v1
    with:
      command: ps
      args: ["--filter", "name=myapp", "--filter", "status=running"]
```

### Example 9: Multi-Container with Network

```yaml
name: Deploy App with Database
steps:
  - name: Create network
    uses: docker/exec@v1
    with:
      command: network
      args: ["create", "app-network"]
    continue-on-error: true

  - name: Start database
    uses: docker/exec@v1
    with:
      command: run
      args:
        - "-d"
        - "--name"
        - "app-db"
        - "--network"
        - "app-network"
        - "-e"
        - "POSTGRES_PASSWORD=secret"
        - "postgres:15"

  - name: Start application
    uses: docker/exec@v1
    with:
      command: run
      args:
        - "-d"
        - "--name"
        - "app-server"
        - "--network"
        - "app-network"
        - "-p"
        - "8080:8080"
        - "-e"
        - "DATABASE_URL=postgresql://postgres:secret@app-db:5432/mydb"
        - "myapp:latest"
```

### Example 10: Image Management

```yaml
name: Image Cleanup
steps:
  - name: List all images
    uses: docker/exec@v1
    with:
      command: images
      args: ["-a"]

  - name: Remove dangling images
    uses: docker/exec@v1
    with:
      command: image
      args: ["prune", "-f"]

  - name: Remove specific image
    uses: docker/exec@v1
    with:
      command: rmi
      args: ["old-image:v1.0"]
    continue-on-error: true
```

## Common Docker Commands

| Command | Description | Example Args |
|---------|-------------|--------------|
| `run` | Create and start container | `["-d", "--name", "app", "image:tag"]` |
| `ps` | List containers | `["--all"]`, `["--filter", "status=running"]` |
| `stop` | Stop container | `["container-name"]` |
| `start` | Start stopped container | `["container-name"]` |
| `restart` | Restart container | `["container-name"]` |
| `rm` | Remove container | `["-f", "container-name"]` |
| `images` | List images | `["--all"]` |
| `pull` | Pull image from registry | `["nginx:latest"]` |
| `rmi` | Remove image | `["image:tag"]` |
| `exec` | Execute command in container | `["container", "command", "arg"]` |
| `logs` | View container logs | `["--tail", "100", "container"]` |
| `inspect` | Display detailed info | `["container-or-image"]` |
| `network` | Manage networks | `["create", "net-name"]` |
| `volume` | Manage volumes | `["create", "vol-name"]` |

## Security Considerations

⚠️ **Docker Socket Access = Root Equivalent**

Access to the Docker socket grants root-level privileges. Follow these best practices:

1. **Principle of Least Privilege**: Only grant Docker access to necessary users
2. **Container User**: Run containers with `--user` flag to avoid root
3. **Resource Limits**: Use `--memory`, `--cpus` to prevent resource exhaustion
4. **Network Isolation**: Use custom networks to isolate containers
5. **Audit Logging**: Monitor all Docker operations
6. **Read-Only Volumes**: Use `:ro` flag for read-only mounts

### Example: Secure Container Execution

```yaml
steps:
  - name: Run container securely
    uses: docker/exec@v1
    with:
      command: run
      args:
        - "-d"
        - "--user"
        - "1000:1000"              # Non-root user
        - "--memory"
        - "512m"                    # Memory limit
        - "--cpus"
        - "1"                       # CPU limit
        - "--read-only"             # Read-only filesystem
        - "--tmpfs"
        - "/tmp:rw,noexec,nosuid"  # Writable tmp with security flags
        - "--network"
        - "isolated-net"            # Custom network
        - "--security-opt"
        - "no-new-privileges:true"  # Prevent privilege escalation
        - "myapp:latest"
```

## Timeout Recommendations

| Operation | Recommended Timeout | Notes |
|-----------|-------------------|-------|
| Fast commands (ps, version) | 30s | Quick metadata operations |
| Container start/stop | 2-5m | Depends on app startup time |
| Image pull | 10m+ | Varies by image size and network |
| Build operations | 15m+ | Depends on build complexity |
| Long-running exec | Custom | Set based on expected duration |

## Troubleshooting

### Error: Docker not found

```
Error: docker command not found
```

**Solution**: Install Docker on the agent server
```bash
# Ubuntu/Debian
sudo apt-get update && sudo apt-get install docker.io

# RHEL/CentOS
sudo yum install docker

# Start daemon
sudo systemctl start docker
sudo systemctl enable docker
```

### Error: Permission denied

```
Error: permission denied - agent user needs to be in docker group
```

**Solution**: Add user to docker group
```bash
sudo usermod -aG docker waterflow-agent
newgrp docker  # Or re-login
```

### Error: Docker daemon not running

```
Error: Docker daemon is not running
```

**Solution**: Start Docker daemon
```bash
sudo systemctl start docker
sudo systemctl status docker
```

### Error: Container not found

```
Error: No such container
```

This is a **permanent error** (non-retriable). Ensure the container exists before operating on it.

### Error: Image not found

```
Error: Unable to find image locally
```

This is a **temporary error** (retriable). The workflow will retry, or you can explicitly pull the image first.

## Performance

- **Execution**: Direct Docker CLI invocation (no wrapper overhead)
- **Latency**: Same as running `docker` command manually
- **Availability Check**: ~100ms per execution (runs `docker info`)
- **Timeout**: Configurable per operation (default: 5 minutes)

## Version History

- **v1** (2025-12-31): Initial release
  - Universal Docker command execution
  - Error classification for Temporal retry
  - Docker availability checking

## See Also

- [Docker CLI Reference](https://docs.docker.com/engine/reference/commandline/cli/)
- [Story 3.7: Docker Exec Node](../../docs/sprint-artifacts/3-7-docker-exec-node.md)
- [Story 3.1: Node Interface](../../docs/sprint-artifacts/3-1-node-interface-design.md)
