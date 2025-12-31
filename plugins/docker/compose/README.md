# Docker Compose Node

**Category:** docker  
**Version:** v1  
**Plugin:** compose.so

## Overview

The Docker Compose node provides complete lifecycle management for Docker Compose stacks through unified `up` and `down` operations. It automatically detects Docker Compose availability (new `docker compose` or legacy `docker-compose`), validates Compose files, and provides Temporal-compatible error classification.

## Features

- ✅ **Unified Operations**: Single node for up/down via `action` parameter
- ✅ **Auto-detection**: Supports both `docker compose` (v2+) and `docker-compose` (v1.x)
- ✅ **File Validation**: Pre-execution Compose file existence checks
- ✅ **Temporal Integration**: Retryable vs non-retryable error classification
- ✅ **Rich Outputs**: Returns container lists, services, stdout/stderr
- ✅ **Environment Injection**: Custom environment variables and working directory
- ✅ **Timeout Control**: Configurable operation timeouts (default: 10m)

## Prerequisites

- Docker Compose installed on the agent server:
  - New version: `docker compose` (Docker CLI plugin, v2.0+)
  - Old version: `docker-compose` (standalone binary, v1.x)
- Valid `docker-compose.yml` file

## Parameters

### Common Parameters

| Parameter      | Type   | Required | Default              | Description                        |
|----------------|--------|----------|----------------------|------------------------------------|
| `action`       | string | Yes      | -                    | Operation: `"up"` or `"down"`      |
| `file`         | string | No       | `docker-compose.yml` | Compose file path                  |
| `project_name` | string | No       | -                    | Project name (`-p` flag)           |
| `workdir`      | string | No       | -                    | Working directory                  |
| `env`          | object | No       | -                    | Environment variables              |
| `timeout`      | string | No       | `"10m"`              | Operation timeout                  |

### Up-Specific Parameters

| Parameter        | Type | Required | Default | Description                              |
|------------------|------|----------|---------|------------------------------------------|
| `detach`         | bool | No       | `true`  | Detached mode (`-d`)                     |
| `build`          | bool | No       | `false` | Build images before starting (`--build`) |
| `force_recreate` | bool | No       | `false` | Force recreate containers                |

### Down-Specific Parameters

| Parameter        | Type   | Required | Default | Description                          |
|------------------|--------|----------|---------|--------------------------------------|
| `volumes`        | bool   | No       | `false` | Remove volumes (`-v`)                |
| `rmi`            | string | No       | `""`    | Remove images: `"local"` or `"all"`  |
| `remove_orphans` | bool   | No       | `true`  | Remove orphan containers             |

## Outputs

| Field       | Type     | Description                 |
|-------------|----------|-----------------------------|
| `action`    | string   | Executed action (up/down)   |
| `containers`| array    | List of container names     |
| `services`  | array    | List of service names       |
| `exit_code` | int      | Command exit code           |
| `stdout`    | string   | Standard output             |
| `stderr`    | string   | Standard error              |
| `elapsed_ms`| int      | Execution time (ms)         |

## Error Classification

| Error Type              | Retryable | Cause                                    |
|-------------------------|-----------|------------------------------------------|
| `ComposeFileNotFound`   | ❌         | Compose file doesn't exist               |
| `ComposeConfigError`    | ❌         | YAML syntax or config errors             |
| `PortConflict`          | ❌         | Port already allocated                   |
| `TimeoutError`          | ✅         | Operation timeout (network/build delays) |
| `NetworkError`          | ✅         | Network connection issues                |
| `ImagePullError`        | ✅         | Image pull failures (temporary network)  |
| `DockerComposeNotInstalled` | ❌     | Compose not found on agent server        |

## Usage Examples

### 1. Basic Up/Down Lifecycle

```yaml
jobs:
  deploy:
    steps:
      - name: Deploy stack
        uses: docker/compose@v1
        with:
          action: up
          file: docker-compose.yml
        id: deploy
      
      - name: Cleanup on failure
        uses: docker/compose@v1
        with:
          action: down
          volumes: true
        if: failure()
```

### 2. Up with Build and Force Recreate

```yaml
- name: Deploy with fresh build
  uses: docker/compose@v1
  with:
    action: up
    file: docker-compose.yml
    project_name: myapp
    build: true
    force_recreate: true
    timeout: 15m
```

### 3. Down with Volume and Image Cleanup

```yaml
- name: Complete cleanup
  uses: docker/compose@v1
  with:
    action: down
    project_name: myapp
    volumes: true
    rmi: all
    remove_orphans: true
```

### 4. Multi-Environment Deployment

```yaml
- name: Deploy to dev
  uses: docker/compose@v1
  with:
    action: up
    file: docker-compose.dev.yml
    project_name: myapp-dev
    env:
      ENV: development
      DEBUG: "true"

- name: Deploy to staging
  uses: docker/compose@v1
  with:
    action: up
    file: docker-compose.staging.yml
    project_name: myapp-staging
    env:
      ENV: staging
```

### 5. Database Backup Flow

```yaml
- name: Start database
  uses: docker/compose@v1
  with:
    action: up
    file: docker-compose.db.yml
    project_name: db-backup

- name: Wait for database
  uses: flow/sleep@v1
  with:
    duration: 30s

- name: Backup database
  uses: docker/exec@v1
  with:
    command: exec
    args: ["db-backup-postgres-1", "pg_dump", "-U", "postgres", "mydb"]

- name: Cleanup
  uses: docker/compose@v1
  with:
    action: down
    project_name: db-backup
    volumes: true
```

### 6. Blue-Green Deployment

```yaml
- name: Deploy green environment
  uses: docker/compose@v1
  with:
    action: up
    file: docker-compose.yml
    project_name: myapp-green
    env:
      PORT: "8081"
      ENV_COLOR: green

- name: Test green environment
  uses: http/request@v1
  with:
    url: http://localhost:8081/health
    timeout: 30s

- name: Switch traffic
  uses: exec/shell@v1
  with:
    command: nginx -s reload

- name: Stop blue environment
  uses: docker/compose@v1
  with:
    action: down
    project_name: myapp-blue
```

### 7. Complete Lifecycle with Tests

```yaml
- name: Stop old stack
  uses: docker/compose@v1
  with:
    action: down
    project_name: myapp
    volumes: true
    rmi: local
  continue-on-error: true

- name: Deploy new stack
  uses: docker/compose@v1
  with:
    action: up
    project_name: myapp
    build: true
    force_recreate: true

- name: Smoke tests
  uses: exec/shell@v1
  with:
    command: curl -f http://localhost:8080/health
  retry:
    max_attempts: 5
    initial_interval: 2s

- name: Cleanup on failure
  uses: docker/compose@v1
  with:
    action: down
    project_name: myapp
    volumes: true
  if: failure()
```

## Example Compose File

```yaml
version: '3.8'

services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
    depends_on:
      - api
    networks:
      - app-network

  api:
    build: ./api
    ports:
      - "3000:3000"
    environment:
      - DATABASE_URL=postgresql://postgres:password@db:5432/myapp
      - REDIS_URL=redis://cache:6379
    depends_on:
      - db
      - cache
    networks:
      - app-network

  db:
    image: postgres:15
    environment:
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=myapp
    volumes:
      - db-data:/var/lib/postgresql/data
    networks:
      - app-network

  cache:
    image: redis:7
    networks:
      - app-network

networks:
  app-network:
    driver: bridge

volumes:
  db-data:
```

## Timeout Recommendations

- **Simple stacks (2-3 services)**: 2-5m
- **Complex stacks (5+ services)**: 10-15m
- **Stacks with builds**: 15-30m
- **Default**: 10m

## Troubleshooting

### Compose Not Found

**Error:** `docker-compose not found - please install Docker Compose`

**Solution:**
```bash
# New version (recommended)
sudo apt-get install docker-compose-plugin

# Legacy version
sudo curl -L "https://github.com/docker/compose/releases/download/v1.29.2/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

### File Not Found

**Error:** `compose file not found: /path/to/docker-compose.yml`

**Solution:**
- Check `file` parameter path (relative to `workdir` if set)
- Verify file exists on agent server
- Use absolute path if relative path fails

### Port Conflicts

**Error:** `port 8080 is already allocated`

**Solution:**
- Stop conflicting containers
- Change port mapping in Compose file
- Use different `project_name` for isolation

### Build Timeout

**Error:** `context deadline exceeded`

**Solution:**
- Increase `timeout` parameter (e.g., `"30m"`)
- Optimize Dockerfile (multi-stage builds, layer caching)
- Pre-pull base images

## Performance Tips

1. **Use `detach: true`** (default) for faster returns
2. **Enable `build: true`** only when needed
3. **Set appropriate timeouts** based on stack complexity
4. **Use `remove_orphans: true`** to avoid container clutter
5. **Leverage Docker layer caching** for faster builds

## See Also

- [Docker Compose CLI Reference](https://docs.docker.com/compose/reference/)
- [Compose File Specification](https://docs.docker.com/compose/compose-file/)
- [Story 3.7: Docker Exec Node](../../../docs/sprint-artifacts/3-7-docker-exec-node.md)
- [Story 3.1: Node Interface](../../../docs/sprint-artifacts/3-1-node-interface-design.md)
