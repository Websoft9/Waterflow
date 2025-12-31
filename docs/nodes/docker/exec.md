# docker/exec@v1

**分类**: docker (容器管理)  
**版本**: v1  
**状态**: stable

## 概述

docker/exec 节点用于在 Agent 服务器上执行 Docker CLI 命令。支持所有 Docker 子命令（run, ps, stop, rm, images, pull, exec, logs等），提供通用的 Docker 命令包装器。

**前置条件**: Agent 服务器必须安装 Docker，且 Agent 用户有 Docker 权限。

## 使用场景

- **容器管理**: 启动、停止、删除容器
- **镜像管理**: 拉取、列出、删除镜像
- **容器监控**: 查看日志、执行命令、检查状态
- **部署验证**: 检查容器健康状态
- **清理任务**: 删除停止的容器、清理无用镜像

## 参数

| 参数名 | 类型 | 必需 | 默认值 | 描述 |
|--------|------|------|--------|------|
| command | string | ✅ | - | Docker 子命令 (如 "run", "ps") |
| args | array | ❌ | [] | 命令参数 |
| timeout | string | ❌ | 5m | 命令超时 |
| docker_host | string | ❌ | - | Docker daemon 地址 |

### 参数详细说明

**command** (string, required)
- Docker 子命令
- 示例: "run", "ps", "stop", "rm", "images", "pull", "exec", "logs"

**args** (array, optional)
- 命令参数列表
- 按顺序传递给 docker 命令
- 示例: `["-d", "--name", "myapp", "nginx:latest"]`

**timeout** (string, optional, default: 5m)
- 命令执行超时
- 镜像拉取建议 10m+
- 示例: `"10m"`

**docker_host** (string, optional)
- Docker daemon 地址
- 默认: unix:///var/run/docker.sock
- 示例: `"tcp://192.168.1.100:2375"`

## 返回值

| 字段名 | 类型 | 描述 |
|--------|------|------|
| exit_code | int | 退出码 |
| stdout | string | 标准输出 |
| stderr | string | 标准错误 |
| elapsed_ms | int | 执行耗时 (毫秒) |
| command_line | string | 完整命令行 |

## Docker 环境要求

### 1. 安装 Docker

```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com | sh

# CentOS/RHEL
sudo yum install -y docker-ce
```

### 2. 配置用户权限

```bash
# 添加用户到 docker 组
sudo usermod -aG docker waterflow-agent

# 重新登录或执行
newgrp docker

# 验证权限
docker ps  # 应该无需 sudo
```

### 3. 启动 Docker

```bash
sudo systemctl start docker
sudo systemctl enable docker
```

## 使用示例

### 示例 1: 运行容器

```yaml
steps:
  - name: Run nginx container
    uses: docker/exec@v1
    with:
      command: run
      args:
        - "-d"
        - "--name"
        - "web"
        - "-p"
        - "80:80"
        - "nginx:latest"
```

### 示例 2: 列出容器

```yaml
steps:
  - name: List all containers
    uses: docker/exec@v1
    with:
      command: ps
      args: ["-a"]
    id: containers
  
  - name: Print container list
    uses: exec/shell@v1
    with:
      command: echo
      args: ["Containers: ${{ steps.containers.outputs.stdout }}"]
```

### 示例 3: 停止和删除容器

```yaml
steps:
  - name: Stop container
    uses: docker/exec@v1
    with:
      command: stop
      args: ["web"]
  
  - name: Remove container
    uses: docker/exec@v1
    with:
      command: rm
      args: ["web"]
```

### 示例 4: 拉取镜像

```yaml
steps:
  - name: Pull image with retry
    uses: docker/exec@v1
    with:
      command: pull
      args: ["nginx:latest"]
      timeout: 10m
    retry:
      max_attempts: 3
      initial_interval: 5s
```

### 示例 5: 查看容器日志

```yaml
steps:
  - name: Get container logs
    uses: docker/exec@v1
    with:
      command: logs
      args: ["--tail", "100", "web"]
    id: logs
  
  - name: Print logs
    uses: exec/shell@v1
    with:
      command: echo
      args: ["${{ steps.logs.outputs.stdout }}"]
```

### 示例 6: 执行容器内命令

```yaml
steps:
  - name: Execute command in container
    uses: docker/exec@v1
    with:
      command: exec
      args: ["web", "nginx", "-t"]
```

### 示例 7: 查看容器信息

```yaml
steps:
  - name: Inspect container
    uses: docker/exec@v1
    with:
      command: inspect
      args: ["web"]
    id: inspect
```

### 示例 8: 列出镜像

```yaml
steps:
  - name: List images
    uses: docker/exec@v1
    with:
      command: images
      args: ["--format", "{{.Repository}}:{{.Tag}}"]
```

### 示例 9: 清理停止的容器

```yaml
steps:
  - name: Remove stopped containers
    uses: docker/exec@v1
    with:
      command: container
      args: ["prune", "-f"]
```

### 示例 10: 容器健康检查

```yaml
steps:
  - name: Check container health
    uses: docker/exec@v1
    with:
      command: inspect
      args: ["--format", "{{.State.Health.Status}}", "web"]
    id: health
  
  - name: Verify healthy
    if: steps.health.outputs.stdout == "healthy"
    uses: exec/shell@v1
    with:
      command: echo
      args: ["Container is healthy"]
```

## 常见错误

### 错误 1: Docker 未安装

**现象:**
```
Error: docker command not found
```

**解决方法:**
安装 Docker（见上文 "Docker 环境要求"）

### 错误 2: Docker Daemon 未运行

**现象:**
```
Error: Cannot connect to the Docker daemon at unix:///var/run/docker.sock
```

**解决方法:**
```bash
sudo systemctl start docker
sudo systemctl status docker
```

### 错误 3: 权限不足

**现象:**
```
Error: permission denied while trying to connect to the Docker daemon socket
```

**解决方法:**
```bash
sudo usermod -aG docker $USER
# 重新登录生效
```

### 错误 4: 容器不存在

**现象:**
```
Error: No such container: mycontainer
exit_code: 1
```

**解决方法:**
1. 检查容器名称是否正确
2. 使用 `docker ps -a` 列出所有容器

### 错误 5: 镜像不存在

**现象:**
```
Error: Unable to find image 'myimage:latest' locally
```

**解决方法:**
先拉取镜像:
```yaml
- name: Pull image
  uses: docker/exec@v1
  with:
    command: pull
    args: ["myimage:latest"]
```

### 错误 6: 端口已被占用

**现象:**
```
Error: Bind for 0.0.0.0:80 failed: port is already allocated
```

**解决方法:**
1. 停止占用端口的容器
2. 使用不同端口

## 最佳实践

### 1. 确保 Docker 环境配置

```yaml
# 执行前检查 Docker 可用性
- name: Check Docker
  uses: exec/shell@v1
  with:
    command: docker
    args: ["version"]
```

### 2. 设置合理超时

```yaml
# 简单命令
timeout: 5m  # 默认值

# 镜像拉取
timeout: 10m

# 构建镜像
timeout: 30m
```

### 3. 使用 detach 模式运行容器

```yaml
# ✅ 后台运行
command: run
args: ["-d", "--name", "app", "myapp:latest"]

# ❌ 前台运行会阻塞
command: run
args: ["--name", "app", "myapp:latest"]
```

### 4. 清理资源

```yaml
# 删除容器
- uses: docker/exec@v1
  with:
    command: rm
    args: ["-f", "mycontainer"]

# 清理停止的容器
- uses: docker/exec@v1
  with:
    command: container
    args: ["prune", "-f"]
```

### 5. 使用错误处理

```yaml
- name: Try to stop container
  uses: docker/exec@v1
  with:
    command: stop
    args: ["mycontainer"]
  continue-on-error: true  # 容器不存在也继续
```

### 6. 验证命令输出

```yaml
- name: Run container
  uses: docker/exec@v1
  with:
    command: run
    args: ["-d", "--name", "app", "myapp"]
  id: run

- name: Check if started
  if: steps.run.outputs.exit_code == 0
  uses: exec/shell@v1
  with:
    command: echo
    args: ["Container started successfully"]
```

## 安全注意事项

1. **Docker Socket 风险**: Docker Socket 访问权限等同于 root 权限
2. **容器隔离**: 使用 `--user` 运行容器，避免 root 用户
3. **资源限制**: 使用 `--memory`, `--cpus` 限制容器资源
4. **网络隔离**: 使用自定义网络隔离容器
5. **定期审计**: 记录所有 Docker 操作日志

## 错误分类

| 错误类型 | 示例 | 可重试 |
|----------|------|--------|
| **永久错误** | Docker 未安装、权限不足、容器不存在 | ❌ |
| **临时错误** | 镜像不存在、网络错误、超时 | ✅ |

## 性能特点

- **直接调用**: 直接执行 Docker CLI，无额外开销
- **性能等同**: 与手动执行 docker 命令一致
- **输出捕获**: 完整捕获 stdout/stderr

## 相关节点

- [docker/compose](compose.md) - 管理 Compose 栈（多容器应用）
- [exec/shell](../exec/shell.md) - 执行系统命令

## 参考文档

- [Story 3.7: Docker 命令执行节点](../../sprint-artifacts/3-7-docker-exec-node.md)
- [Docker Documentation](https://docs.docker.com/)

---

**文档版本**: v1  
**最后更新**: 2025-12-31  
**状态**: stable
