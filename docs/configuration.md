# 配置说明

Waterflow 支持通过配置文件、环境变量和命令行参数进行配置。

## 配置优先级

配置来源按以下优先级从高到低：

1. **命令行参数** - `--port`, `--log-level` 等
2. **环境变量** - `WATERFLOW_*` 前缀
3. **配置文件** - `config.yaml`
4. **默认值**

## 配置文件

默认配置文件为 `config.yaml`，可通过 `--config` 参数指定其他路径。

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

## 环境变量

所有配置项都可以通过环境变量覆盖。环境变量命名规则：

```
WATERFLOW_ + 配置路径（用下划线分隔层级）
```

### 示例

| 配置路径 | 环境变量 |
|----------|----------|
| `server.port` | `WATERFLOW_SERVER_PORT` |
| `server.host` | `WATERFLOW_SERVER_HOST` |
| `log.level` | `WATERFLOW_LOG_LEVEL` |
| `log.format` | `WATERFLOW_LOG_FORMAT` |
| `temporal.host` | `WATERFLOW_TEMPORAL_HOST` |

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

启动时会自动验证配置，包括：

- 端口范围检查（1-65535）
- 日志级别有效性（debug/info/warn/error）
- 超时时间最小值（≥1s）
- 必需字段检查

配置无效时会显示详细错误信息并退出：

```
Failed to load config: invalid configuration: server.port must be between 1 and 65535, got 70000
```

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
- 复制 `config.example.yaml` 为 `config.yaml`
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
