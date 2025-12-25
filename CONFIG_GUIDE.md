# Waterflow 配置文件指南

## 配置文件说明

### 文件结构

```
waterflow/
├── config.example.yaml   # ✅ 配置模板 (带完整注释,提交到Git)
└── config.yaml          # ⚠️ 本地配置 (已添加到.gitignore,不提交)
```

### 使用方法

#### 1. 首次使用

```bash
# 复制示例配置文件
cp config.example.yaml config.yaml

# 根据本地环境修改 config.yaml
vim config.yaml
```

#### 2. 本地开发配置

`config.yaml` 用于本地开发,可以自定义：
- 修改端口号
- 调整日志级别
- 配置 Temporal 连接地址

**注意:** `config.yaml` 已在 `.gitignore` 中,不会被提交。

#### 3. 生产环境配置

**推荐使用环境变量覆盖配置:**

```bash
# 通过环境变量覆盖
export WATERFLOW_SERVER_PORT=9090
export WATERFLOW_LOG_LEVEL=debug
export WATERFLOW_TEMPORAL_HOST=temporal.prod.com:7233

# 运行服务
./bin/server
```

**环境变量命名规则:**
- 前缀: `WATERFLOW_`
- 格式: `WATERFLOW_<配置路径>`(用下划线分隔层级)

**示例映射:**
```yaml
server:
  port: 8080              # WATERFLOW_SERVER_PORT
  host: "0.0.0.0"         # WATERFLOW_SERVER_HOST

log:
  level: "info"           # WATERFLOW_LOG_LEVEL
  format: "json"          # WATERFLOW_LOG_FORMAT

temporal:
  host: "localhost:7233"  # WATERFLOW_TEMPORAL_HOST
  namespace: "default"    # WATERFLOW_TEMPORAL_NAMESPACE
```

#### 4. Docker 部署

```bash
docker run \
  -e WATERFLOW_SERVER_PORT=9090 \
  -e WATERFLOW_LOG_LEVEL=debug \
  -e WATERFLOW_TEMPORAL_HOST=temporal:7233 \
  waterflow:latest
```

或使用 docker-compose:

```yaml
services:
  waterflow:
    image: waterflow:latest
    environment:
      - WATERFLOW_SERVER_PORT=9090
      - WATERFLOW_LOG_LEVEL=info
      - WATERFLOW_TEMPORAL_HOST=temporal:7233
```

## 配置项说明

详细配置说明请查看 `config.example.yaml` 中的注释。

## 常见问题

**Q: 为什么 config.yaml 不提交到 Git?**
A: `config.yaml` 包含本地环境特定的配置,每个开发者的配置可能不同。提交会造成冲突。

**Q: 如何在团队中共享配置更新?**
A: 更新 `config.example.yaml` 并提交,团队成员拉取后手动同步到自己的 `config.yaml`。

**Q: 生产环境如何管理配置?**
A: 推荐使用环境变量或 K8s ConfigMap/Secret 管理,不使用配置文件。
