# 配置说明

Waterflow 支持通过配置文件、环境变量和命令行参数进行灵活配置,适应不同的部署环境。

## 📋 配置方式和优先级

Waterflow 支持以下配置方式,按优先级从高到低排列:

1. **命令行参数** - 运行时指定,优先级最高
2. **环境变量** - 云原生部署首选 (Docker/Kubernetes)
3. **配置文件** - 结构化配置,适合复杂场景
4. **默认值** - 内置默认值,适合开发环境

**示例**: 如果在 `config.yaml` 中设置 `server.port: 8080`,同时设置环境变量 `WATERFLOW_SERVER_PORT=9090`,最终生效的值是 `9090`(环境变量优先级更高)。

## 📁 配置文件结构

```
Waterflow/
├── examples/configs/               # 配置文件模板目录
│   ├── config.example.yaml         # ✅ Server 配置模板 (带完整注释)
│   ├── config.agent.example.yaml   # ✅ Agent 配置模板
│   ├── config-dev.yaml             # ✅ 开发环境配置示例
│   ├── config-prod.yaml            # ✅ 生产环境配置示例
│   ├── config-minimal.yaml         # ✅ 最小化配置示例
│   └── server-groups.example.yaml  # ✅ Server Groups 配置模板
├── config.yaml                     # ❌ 实际配置 (不提交到Git)
└── .gitignore                      # 已忽略 config.yaml
```

**配置文件说明：**
- **模板文件**（`examples/configs/*.example.yaml`）：包含完整注释，提交到 Git
- **环境配置示例**（`config-dev.yaml`, `config-prod.yaml`, `config-minimal.yaml`）：不同环境的配置示例
- **实际配置**（`config.yaml`）：本地环境特定配置，不提交到 Git

**支持的配置文件格式：**
- ✅ YAML 格式（`.yaml`, `.yml`）
- ✅ TOML 格式（`.toml`）

**格式对比示例：**

<table>
<tr>
<td>YAML 格式 (config.yaml)</td>
<td>TOML 格式 (config.toml)</td>
</tr>
<tr>
<td>

```yaml
server:
  host: "0.0.0.0"
  port: 8080

log:
  level: "info"
  format: "json"

temporal:
  host: "localhost:7233"
  namespace: "waterflow"
  task_queue: "waterflow-server"
```

</td>
<td>

```toml
[server]
host = "0.0.0.0"
port = 8080

[log]
level = "info"
format = "json"

[temporal]
host = "localhost:7233"
namespace = "waterflow"
task_queue = "waterflow-server"
```

</td>
</tr>
</table>

**格式选择建议：**
- **YAML**：更简洁，支持注释，适合层级较深的配置（推荐）
- **TOML**：语法严格，类型安全，适合类型敏感的配置
- 两种格式功能完全等价，选择您团队熟悉的即可

**注意**：配置文件格式通过文件扩展名自动识别，无需额外指定。

## 🚀 快速开始

### 1. 创建配置文件

**方式一：使用环境配置示例**

```bash
# 开发环境
cp examples/configs/config-dev.yaml config.yaml

# 生产环境
cp examples/configs/config-prod.yaml config.yaml

# 最小化配置
cp examples/configs/config-minimal.yaml config.yaml
```

**方式二：使用通用模板**

```bash
# 复制模板文件
cp examples/configs/config.example.yaml config.yaml

# 根据本地环境修改
vim config.yaml
```

### 2. 使用环境变量（推荐生产环境）

```bash
# 通过环境变量覆盖配置
export WATERFLOW_SERVER_PORT=9090
export WATERFLOW_LOG_LEVEL=info
export WATERFLOW_TEMPORAL_HOST=temporal:7233

# 运行服务
./bin/server
```

## 配置优先级

配置来源按以下优先级从高到低：

1. **命令行参数** - `--port`, `--log-level` 等
2. **环境变量** - `WATERFLOW_*` 前缀
3. **配置文件** - `config.yaml`
4. **默认值**

## 配置文件

默认配置文件为 `config.yaml`，可通过 `--config` 参数指定其他路径。

**注意事项：**
- `config.yaml` 包含本地环境特定配置，每个开发者的配置可能不同
- 该文件已在 `.gitignore` 中，不会被提交到 Git
- 团队共享配置更新：修改 `examples/configs/config.example.yaml` 并提交

### 配置文件位置

### 完整配置示例

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: "30s"
  write_timeout: "30s"
  shutdown_timeout: "30s"

log:
  level: "info"
  format: "json"
  output: "stdout"

temporal:
  host: "localhost:7233"
  namespace: "waterflow"
  task_queue: "waterflow-server"
```

## 配置项说明

### Server 配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `server.host` | string | "0.0.0.0" | 监听地址，0.0.0.0 表示所有接口 |
| `server.port` | int | 8080 | HTTP 端口（1-65535） |
| `server.read_timeout` | duration | "30s" | 读取请求超时时间（≥1s） |
| `server.write_timeout` | duration | "30s" | 写入响应超时时间（≥1s） |
| `server.shutdown_timeout` | duration | "30s" | 优雅关闭超时时间（≥1s） |

### Log 配置

| 配置项 | 类型 | 默认值 | 可选值 | 说明 |
|--------|------|--------|--------|------|
| `log.level` | string | "info" | debug, info, warn, error | 日志级别 |
| `log.format` | string | "json" | json, text | 日志格式 |
| `log.output` | string | "stdout" | stdout, stderr, 文件路径 | 日志输出 |

**环境建议：**
- **开发环境**: `level=debug`, `format=text`
- **生产环境**: `level=info`, `format=json`

### Temporal 配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `temporal.host` | string | "localhost:7233" | Temporal Server 地址 |
| `temporal.namespace` | string | "waterflow" | Temporal Namespace |
| `temporal.task_queue` | string | "waterflow-server" | Task Queue 名称 |

### Health Check 配置 (Story 8-4)

Waterflow Server 提供健康检查端点用于容器编排和监控系统。

**端点说明：**
- **`/health`**: Liveness probe - 检查进程是否存活（快速响应 < 100ms）
- **`/ready`**: Readiness probe - 检查依赖服务状态（Temporal、数据库）

**配置项：**

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `server.health.timeout` | duration | "2s" | 整体健康检查超时时间 |
| `server.health.temporal_timeout` | duration | "2s" | Temporal 连接检查超时 |
| `server.health.db_timeout` | duration | "1s" | 数据库 Ping 超时 |

**环境变量：**
```bash
WATERFLOW_SERVER_HEALTH_TIMEOUT=2s
WATERFLOW_SERVER_HEALTH_TEMPORAL_TIMEOUT=2s
WATERFLOW_SERVER_HEALTH_DB_TIMEOUT=1s
```

**配置示例：**
```yaml
server:
  health:
    timeout: 2s              # 整体健康检查超时
    temporal_timeout: 2s     # Temporal 检查超时
    db_timeout: 1s           # 数据库检查超时
```

**响应示例：**

所有依赖健康时（HTTP 200）：
```json
{
  "status": "ready",
  "timestamp": "2026-01-09T10:30:00Z",
  "checks": {
    "temporal": "ok"
  }
}
```

依赖不可用时（HTTP 503）：
```json
{
  "status": "not_ready",
  "timestamp": "2026-01-09T10:30:05Z",
  "checks": {
    "temporal": "connection refused: dial tcp [::1]:7233: connect: connection refused"
  }
}
```

**Docker Health Check：**

Server Dockerfile 已配置健康检查：
```dockerfile
HEALTHCHECK --interval=10s --timeout=5s --start-period=10s --retries=10 \
    CMD curl -f http://localhost:8080/health || exit 1
```

**Kubernetes Readiness Probe：**

```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 10   # 等待 Temporal 连接建立
  periodSeconds: 5          # 每 5 秒检查一次
  timeoutSeconds: 2         # 单次检查超时
  failureThreshold: 3       # 连续失败 3 次标记为 Not Ready
```

**最佳实践：**
- Liveness probe 使用 `/health`（快速检查）
- Readiness probe 使用 `/ready`（依赖检查）
- 设置合理的 `start-period`/`initialDelaySeconds`（10-20 秒）
- 健康检查不需要认证（容器平台需要访问）

### Agent Health Check 配置 (Story 8-4)

Agent 通过 Prometheus metrics 端点提供健康检查：

**健康检查端点：**
- **`/metrics`**: 暴露在 `9090` 端口（可通过 `WATERFLOW_AGENT_METRICS_PORT` 配置）

**Docker Health Check：**

Agent Dockerfile 配置使用 metrics 端点进行健康检查：
```dockerfile
HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=5 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9090/metrics || exit 1
```

**说明：**
- Agent 每 15 秒检查一次（比 Server 稍慢，因为 Agent 启动较慢）
- 启动后等待 20 秒才开始检查（Agent 需要连接 Temporal 并加载插件）
- 连续失败 5 次后标记为 unhealthy
- 使用 `wget` 而非 `curl`（Alpine 镜像默认包含 wget）

**Kubernetes Liveness Probe：**

```yaml
livenessProbe:
  httpGet:
    path: /metrics
    port: 9090
  initialDelaySeconds: 20   # 等待 Agent 启动并连接 Temporal
  periodSeconds: 15         # 每 15 秒检查一次
  timeoutSeconds: 5         # 单次检查超时
  failureThreshold: 5       # 连续失败 5 次重启 Pod
```

**配置 Metrics 端口：**
```bash
# 环境变量
export WATERFLOW_AGENT_METRICS_PORT=9091

# 或配置文件
agent:
  metrics_port: "9091"
```

## 环境变量

所有配置项都可以通过环境变量覆盖。环境变量命名规则：

```
WATERFLOW_ + 配置路径（用下划线分隔层级）
```

### 示例

**Server 环境变量:**

| 配置路径 | 环境变量 | 类型 | 默认值 |
|----------|----------|------|-------|
| `server.port` | `WATERFLOW_SERVER_PORT` | int | 8080 |
| `server.host` | `WATERFLOW_SERVER_HOST` | string | 0.0.0.0 |
| `server.read_timeout` | `WATERFLOW_SERVER_READ_TIMEOUT` | duration | 30s |
| `server.write_timeout` | `WATERFLOW_SERVER_WRITE_TIMEOUT` | duration | 30s |
| `server.shutdown_timeout` | `WATERFLOW_SERVER_SHUTDOWN_TIMEOUT` | duration | 30s |
| `server.metrics_port` | `WATERFLOW_SERVER_METRICS_PORT` | int | 9090 |
| `server.api_key` | `WATERFLOW_SERVER_API_KEY` | string | "" |
| `server.tls_cert_file` | `WATERFLOW_SERVER_TLS_CERT_FILE` | string | "" |
| `server.tls_key_file` | `WATERFLOW_SERVER_TLS_KEY_FILE` | string | "" |
| `log.level` | `WATERFLOW_LOG_LEVEL` | string | info |
| `log.format` | `WATERFLOW_LOG_FORMAT` | string | json |
| `log.output` | `WATERFLOW_LOG_OUTPUT` | string | stdout |
| `temporal.host` | `WATERFLOW_TEMPORAL_HOST` | string | localhost:7233 |
| `temporal.namespace` | `WATERFLOW_TEMPORAL_NAMESPACE` | string | waterflow |
| `temporal.task_queue` | `WATERFLOW_TEMPORAL_TASK_QUEUE` | string | waterflow-server |
| `temporal.connection_timeout` | `WATERFLOW_TEMPORAL_CONNECTION_TIMEOUT` | duration | 10s |
| `temporal.max_retries` | `WATERFLOW_TEMPORAL_MAX_RETRIES` | int | 10 |
| `temporal.retry_interval` | `WATERFLOW_TEMPORAL_RETRY_INTERVAL` | duration | 5s |
| `events.handler_type` | `WATERFLOW_EVENTS_HANDLER_TYPE` | string | noop |
| `events.webhook.url` | `WATERFLOW_EVENTS_WEBHOOK_URL` | string | "" |
| `events.webhook.timeout` | `WATERFLOW_EVENTS_WEBHOOK_TIMEOUT` | duration | 5s |

**Agent 环境变量:**

| 配置路径 | 环境变量 | 类型 | 默认值 | 说明 |
|----------|----------|------|-------|------|
| `agent.task_queues` | `WATERFLOW_AGENT_TASK_QUEUES`<br>或 `TASK_QUEUES`(兼容) | string[] | [] | **必需**,逗号分隔 |
| `agent.id` | `WATERFLOW_AGENT_ID` | string | auto | Agent实例ID |
| `agent.plugin_dir` | `WATERFLOW_AGENT_PLUGIN_DIR` | string | /opt/waterflow/plugins | 插件目录 |
| `agent.metrics_port` | `WATERFLOW_AGENT_METRICS_PORT` | string | "" | Prometheus端口 |
| `agent.shutdown_timeout` | `WATERFLOW_AGENT_SHUTDOWN_TIMEOUT` | duration | 30s | 关闭超时 |
| `temporal.host` | `WATERFLOW_TEMPORAL_HOST` | string | localhost:7233 | Temporal地址 |
| `temporal.namespace` | `WATERFLOW_TEMPORAL_NAMESPACE` | string | default | Temporal命名空间 |
| `temporal.connection_timeout` | `WATERFLOW_TEMPORAL_CONNECTION_TIMEOUT` | duration | 10s | 连接超时 |
| `temporal.max_retries` | `WATERFLOW_TEMPORAL_MAX_RETRIES` | int | 10 | 最大重试次数 |
| `temporal.retry_interval` | `WATERFLOW_TEMPORAL_RETRY_INTERVAL` | duration | 5s | 重试间隔 |
| `log.level` | `WATERFLOW_LOG_LEVEL` | string | info | 日志级别 |
| `log.format` | `WATERFLOW_LOG_FORMAT` | string | json | 日志格式 |
| `log.output` | `WATERFLOW_LOG_OUTPUT` | string | stdout | 日志输出 |

**注意**:
- Agent的 `task_queues` 支持两种环境变量格式,新代码推荐使用 `WATERFLOW_AGENT_TASK_QUEUES`
- `TASK_QUEUES` 是为了向后兼容早期版本
- Task queues 使用逗号分隔: `linux-amd64,linux-common,gpu-a100`

**Agent 环境变量使用示例:**

```bash
# 1. 基本配置 (必需)
export WATERFLOW_AGENT_TASK_QUEUES=linux-amd64,linux-common
export WATERFLOW_TEMPORAL_HOST=temporal:7233
export WATERFLOW_TEMPORAL_NAMESPACE=waterflow
./bin/agent

# 2. 完整配置
export WATERFLOW_AGENT_TASK_QUEUES=linux-amd64,gpu-a100
export WATERFLOW_AGENT_ID=agent-prod-01
export WATERFLOW_AGENT_PLUGIN_DIR=/opt/waterflow/plugins
export WATERFLOW_AGENT_METRICS_PORT=9090
export WATERFLOW_TEMPORAL_HOST=temporal.prod.internal:7233
export WATERFLOW_TEMPORAL_NAMESPACE=waterflow-prod
export WATERFLOW_LOG_LEVEL=info
export WATERFLOW_LOG_FORMAT=json
./bin/agent

# 3. 向后兼容的写法 (legacy)
export TASK_QUEUES=linux-amd64,linux-common
export WATERFLOW_TEMPORAL_HOST=localhost:7233
./bin/agent
```

**Docker 部署示例:**

```bash
# Server 容器
docker run -d \
  -e WATERFLOW_SERVER_PORT=9090 \
  -e WATERFLOW_LOG_LEVEL=debug \
  -e WATERFLOW_TEMPORAL_HOST=temporal:7233 \
  -p 9090:9090 \
  waterflow:latest

# Agent 容器
docker run -d \
  -e WATERFLOW_AGENT_TASK_QUEUES=linux-amd64,linux-common \
  -e WATERFLOW_TEMPORAL_HOST=temporal:7233 \
  -e WATERFLOW_TEMPORAL_NAMESPACE=waterflow \
  -v /opt/waterflow/plugins:/opt/waterflow/plugins:ro \
  waterflow-agent:latest
```

**Kubernetes 部署示例:**

```yaml
# Agent Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: waterflow-agent
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: agent
        image: waterflow-agent:latest
        env:
        # 从 ConfigMap 读取
        - name: WATERFLOW_AGENT_TASK_QUEUES
          valueFrom:
            configMapKeyRef:
              name: waterflow-config
              key: agent.task.queues
        - name: WATERFLOW_TEMPORAL_HOST
          valueFrom:
            configMapKeyRef:
              name: waterflow-config
              key: temporal.host
        # 从 Secret 读取敏感信息
        - name: WATERFLOW_SERVER_API_KEY
          valueFrom:
            secretKeyRef:
              name: waterflow-secrets
              key: api-key
---
# ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: waterflow-config
data:
  agent.task.queues: "linux-amd64,linux-common"
  temporal.host: "temporal.default.svc.cluster.local:7233"
```

### 使用示例

```bash
# 单个环境变量
export WATERFLOW_SERVER_PORT=9090
./bin/server

# 多个环境变量
export WATERFLOW_SERVER_PORT=9090
export WATERFLOW_LOG_LEVEL=debug
export WATERFLOW_LOG_FORMAT=text
./bin/server
```

### Docker 环境变量

```bash
docker run -e WATERFLOW_SERVER_PORT=9090 \
           -e WATERFLOW_LOG_LEVEL=debug \
           -p 9090:9090 \
           waterflow:latest
```

## 配置文件路径

### 默认路径

Waterflow Server 和 Agent 使用不同的默认配置文件路径：

| 组件 | 默认路径 | 说明 |
|------|---------|------|
| **Server** | `/etc/waterflow/config.yaml` | 系统级配置目录 (裸机部署) |
| **Agent** | `/app/config/config.yaml` | 应用配置目录 (容器部署) |

**重要说明**:
- 上述默认路径仅为**占位符**,实际部署中很少使用
- **推荐做法**: 使用 `--config` 参数显式指定配置文件路径
- **Docker 部署**: 推荐使用环境变量,无需挂载配置文件
- **裸机部署**: 可将配置文件放置在 `/etc/waterflow/` 或自定义路径

### 指定配置文件

```bash
# 使用 --config 参数指定路径
./bin/server --config /path/to/config.yaml
./bin/agent --config /opt/waterflow/agent-config.yaml

# 使用环境变量 CONFIG_PATH (仅 Agent 支持)
export CONFIG_PATH=/opt/waterflow/config.yaml
./bin/agent

# Docker 挂载配置文件
docker run -d \
  -v /etc/waterflow/config.yaml:/app/config.yaml:ro \
  waterflow:latest --config /app/config.yaml
```

### 配置文件不存在时的行为

如果指定的配置文件不存在,Waterflow 会:
1. 输出警告信息: `Warning: config file ... not found`
2. 使用默认值和环境变量继续启动
3. 不会报错退出

这种设计允许**完全依赖环境变量进行配置**(云原生最佳实践)。

## 配置文件路径

### 默认路径说明

Waterflow Server 和 Agent 使用不同的默认配置文件路径：

| 组件 | 默认路径 | 说明 |
|------|---------|------|
| **Server** | `/etc/waterflow/config.yaml` | 系统级配置目录 (裸机部署) |
| **Agent** | `/app/config/config.yaml` | 应用配置目录 (容器部署) |

**重要说明**:
- 上述默认路径仅为**占位符**,实际部署中很少直接使用
- **推荐做法**: 使用 `--config` 参数显式指定配置文件路径
- **Docker 部署**: 推荐使用环境变量,无需挂载配置文件
- **裸机部署**: 可将配置文件放置在 `/etc/waterflow/` 或自定义路径

### 指定配置文件

```bash
# 使用 --config 参数指定路径
./bin/server --config /path/to/config.yaml
./bin/agent --config /opt/waterflow/agent-config.yaml

# 使用环境变量 CONFIG_PATH (仅 Agent 支持)
export CONFIG_PATH=/opt/waterflow/config.yaml
./bin/agent

# Docker 挂载配置文件
docker run -d \
  -v /etc/waterflow/config.yaml:/app/config.yaml:ro \
  waterflow:latest --config /app/config.yaml
```

### 配置文件不存在时的行为

如果指定的配置文件不存在,Waterflow 会:
1. 输出警告信息: `Warning: config file ... not found`
2. 使用默认值和环境变量继续启动
3. 不会报错退出

这种设计允许**完全依赖环境变量进行配置**(云原生最佳实践)。

## 命令行参数

支持的命令行参数：

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--config` | string | "config.yaml" | 配置文件路径 |
| `--port` | int | 0 (使用配置文件) | HTTP 端口 |
| `--log-level` | string | "" (使用配置文件) | 日志级别 |
| `--version` | bool | false | 显示版本信息 |

### 使用示例

```bash
# 指定配置文件
./bin/server --config /etc/waterflow/config.yaml

# 覆盖端口
./bin/server --port 9090

# 覆盖日志级别
./bin/server --log-level debug

# 组合使用
./bin/server --config prod.yaml --port 9090 --log-level info

# 显示版本
./bin/server --version
```

## 配置验证

启动时会自动验证配置,确保所有值符合要求。验证失败时会显示详细错误信息并立即退出(exit code 1)。

### Server 配置验证规则

| 配置项 | 验证规则 | 错误示例 |
|--------|---------|---------|
| `server.port` | 必须在 1-65535 范围内 | `invalid server.port: 99999, must be 1-65535` |
| `server.metrics_port` | 必须在 0-65535 范围内 | `invalid server.metrics_port: 70000, must be 0-65535` |
| `server.read_timeout` | 必须 >= 0 | `invalid server.read_timeout: -5s, must be >= 0` |
| `server.write_timeout` | 必须 >= 0 | `invalid server.write_timeout: -5s, must be >= 0` |
| `server.shutdown_timeout` | 必须 >= 0 | `invalid server.shutdown_timeout: -10s, must be >= 0` |
| `server.tls_cert_file` | 如果设置,则 `tls_key_file` 必需 | `server.tls_key_file is required when tls_cert_file is set` |
| `server.tls_key_file` | 如果设置,则 `tls_cert_file` 必需 | `server.tls_cert_file is required when tls_key_file is set` |
| `log.level` | 必须是: `debug`, `info`, `warn`, `error` | `invalid log.level: trace, must be one of: [debug info warn error]` |
| `log.format` | 必须是: `json`, `text` | `invalid log.format: xml, must be one of: [json text]` |
| `temporal.host` | 必须提供 | `temporal.host is required` |
| `temporal.namespace` | 必须提供 | `temporal.namespace is required` |
| `temporal.task_queue` | 必须提供 | `temporal.task_queue is required` |
| `temporal.connection_timeout` | 必须 >= 0 | `invalid temporal.connection_timeout: -5s, must be >= 0` |
| `temporal.max_retries` | 必须 >= 0 | `invalid temporal.max_retries: -1, must be >= 0` |
| `temporal.retry_interval` | 必须 >= 0 | `invalid temporal.retry_interval: -5s, must be >= 0` |
| `events.webhook.url` | 当 `handler_type=webhook` 时必需 | `events.webhook.url is required when handler_type=webhook` |

### Agent 配置验证规则

| 配置项 | 验证规则 | 错误示例 |
|--------|---------|---------|
| `agent.task_queues` | **必需**,至少包含一个队列名称 | `agent.task_queues is required, must specify at least one task queue` |
| Task Queue 名称 | 字母数字和连字符,最大255字符 | `invalid task queue name "queue_with_underscore": contains invalid character` |
| `agent.plugin_dir` | 必须提供 | `agent.plugin_dir is required` |
| `temporal.host` | 必须提供 | `temporal.host is required` |
| `temporal.namespace` | 必须提供 | `temporal.namespace is required` |
| `log.level` | 必须是: `debug`, `info`, `warn`, `error` | (同上) |
| `log.format` | 必须是: `json`, `text` | (同上) |

### 验证错误示例

#### 端口超出范围

```bash
$ WATERFLOW_SERVER_PORT=99999 ./server
Error: Failed to load config: invalid configuration: invalid server.port: 99999, must be 1-65535
  Set WATERFLOW_SERVER_PORT=8080 or update server.port in config file
```

#### 日志级别无效

```bash
$ WATERFLOW_LOG_LEVEL=trace ./server
Error: Failed to load config: invalid configuration: invalid log.level: trace, must be one of: [debug info warn error]
  Set WATERFLOW_LOG_LEVEL=info or update log.level in config file
```

#### Agent Task Queues 缺失

```bash
$ ./agent
Error: Failed to load config: invalid configuration: agent.task_queues is required, must specify at least one task queue
  Hint: Set WATERFLOW_AGENT_TASK_QUEUES environment variable or use --task-queues flag
  Example: export WATERFLOW_AGENT_TASK_QUEUES=linux-amd64,linux-common
```

#### Webhook URL 缺失

```bash
$ WATERFLOW_EVENTS_HANDLER_TYPE=webhook ./server
Error: Failed to load config: invalid configuration: events.webhook.url is required when handler_type=webhook
  Set WATERFLOW_EVENTS_WEBHOOK_URL=http://example.com/webhook or update events.webhook.url in config file
```

#### 配置文件格式错误

```yaml
# config-invalid.yaml
server:
  port: "not a number"  # 应为整数
```

```bash
$ ./server --config config-invalid.yaml
Error: Failed to load config: failed to unmarshal config: 1 error(s) decoding:
* 'Server.Port' expected type 'int', got unconvertible type 'string'
File: config-invalid.yaml
```

### 错误信息原则

所有验证错误信息遵循以下原则:

- ✅ **清晰定位** - 指出具体的配置项和错误值
- ✅ **给出建议** - 提示正确的值范围或格式
- ✅ **提供示例** - 给出正确的配置示例
- ✅ **快速失败** - 启动时立即验证,不等到运行时

### 配置值无效 (旧示例)

```
Failed to load config: invalid configuration: log.level must be one of [debug, info, warn, error], got invalid
```

**解决方案**:
- 检查配置文件中的值是否符合要求
- 参考本文档中的验证规则表
- 使用默认值或环境变量覆盖

## 最佳实践

### 开发环境

```yaml
server:
  port: 8080

log:
  level: "debug"
  format: "text"
  output: "stdout"

temporal:
  host: "localhost:7233"
```

### 生产环境

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: "60s"
  write_timeout: "60s"
  shutdown_timeout: "30s"

log:
  level: "info"
  format: "json"
  output: "/var/log/waterflow/server.log"

temporal:
  host: "temporal.production.internal:7233"
  namespace: "waterflow-prod"
```

### 容器化部署

使用环境变量注入配置，避免在镜像中硬编码：

```bash
docker run \
  -e WATERFLOW_SERVER_PORT=8080 \
  -e WATERFLOW_LOG_LEVEL=info \
  -e WATERFLOW_TEMPORAL_HOST=temporal:7233 \
  -p 8080:8080 \
  waterflow:latest
```

## 安全建议

1. **不要在配置文件中存储敏感信息**（如密码、Token）
2. **使用环境变量**注入敏感配置
3. **限制配置文件权限**：`chmod 600 config.yaml`
4. **生产环境使用专用 namespace** 避免与其他环境混用

## 故障排查

### 配置文件未找到

```
Warning: config file config.yaml not found, using defaults and environment variables
```

**解决方案**：
- 复制 `examples/configs/config.example.yaml` 为 `config.yaml`
- 使用 `--config` 指定正确路径
- 完全依赖环境变量和默认值

### 配置值无效

```
Failed to load config: invalid configuration: log.level must be one of [debug, info, warn, error], got invalid
```

**解决方案**：
- 检查配置文件中的值是否符合要求
- 参考本文档中的可选值列表
- 使用默认值或环境变量覆盖
## Step 配置参考

### retry-strategy (重试策略)

配置 Step 失败时的重试行为。(Story 4.3)

**字段:**

| 字段 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| max-attempts | integer | 否 | 3 | 最大尝试次数 (1-10) |
| initial-interval | string | 否 | "1s" | 首次重试间隔 (≥1s) |
| backoff-coefficient | float | 否 | 2.0 | 退避系数 (1.0-10.0) |
| max-interval | string | 否 | "60s" | 最大重试间隔 |

**示例:**

```yaml
steps:
  # 默认重试策略 (3次,指数退避)
  - name: Quick Task
    uses: exec/script@v1
    with:
      command: ./task.sh
  
  # 自定义重试策略 - 网络调用
  - name: API Call
    uses: http/request@v1
    retry-strategy:
      max-attempts: 10
      initial-interval: 2s
      backoff-coefficient: 1.5
      max-interval: 60s
    with:
      url: https://api.example.com/data
  
  # 禁用重试
  - name: One Shot
    uses: exec/script@v1
    retry-strategy:
      max-attempts: 1
    with:
      command: ./critical-task.sh
```

**重试算法:**

指数退避算法:
```
下次间隔 = min(
    initial-interval * (backoff-coefficient ^ attempt),
    max-interval
)
```

示例 (initial-interval=1s, backoff-coefficient=2.0):
```
尝试 1: 失败 → 等待 1s
尝试 2: 失败 → 等待 2s
尝试 3: 失败 → 等待 4s
尝试 4: 失败 → 等待 8s
```

**永久性错误 (不重试):**

以下错误类型不会重试,立即失败:
- `validation_error` - 参数验证错误
- `schema_error` - JSON Schema 验证错误
- `not_found` - 资源不存在
- `permission_denied` - 权限不足
- `invalid_argument` - 参数无效
- `node_not_registered` - 节点未注册
- `plugin_load_error` - 插件加载失败

**最佳实践:**

- 🎯 **网络调用:** 使用较高的 max-attempts (5-10次)
- 🎯 **本地脚本:** 使用默认策略 (3次)
- 🎯 **关键任务:** 禁用重试 (max-attempts: 1)
- 🎯 **长时间任务:** 增大 max-interval (避免过长等待)
