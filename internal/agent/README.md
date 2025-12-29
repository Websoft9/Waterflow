# Agent 内部实现说明

## 概述

Agent 模块实现了 Waterflow 分布式系统的 Worker 端，负责在目标服务器上执行工作流任务。Agent 通过 Temporal Worker 连接到 Temporal Server，轮询指定的 Task Queue，并执行分配的 Activity。

## 架构组件

### worker.go - Temporal Worker 封装

**核心功能：**
- 连接 Temporal Server (支持重试机制)
- 为每个 Task Queue 创建独立的 Worker 实例
- 注册 Workflow 和 Activity
- 优雅关闭和资源清理

**关键类型：**
```go
type Worker struct {
    config         *config.Config
    logger         *zap.Logger
    temporalClient *temporal.Client
    workers        []worker.Worker
    pluginManager  *PluginManager
    wg             sync.WaitGroup
}
```

**生命周期：**
1. `NewWorker()` - 创建 Worker 实例并连接 Temporal
2. `Start()` - 启动所有 Task Queue 的 Worker (后台 goroutine)
3. `Shutdown()` - 停止所有 Worker 并等待任务完成 (带超时)

### plugin_manager.go - 插件管理器

**当前状态 (Story 2.1)：** Stub 实现，仅扫描和验证插件文件

**功能：**
- 扫描 `plugin_dir` 目录下的 `.so` 文件
- 验证插件文件大小和完整性
- 记录可用插件清单

**未来功能 (Epic 4 - Story 4.1)：**
- 使用 `plugin.Open()` 动态加载 `.so` 文件
- 通过反射提取 `Node` 接口实现
- 支持插件热重载 (`auto_reload_plugins`)

### config.go - Agent 配置 (未单独创建)

**说明：** Agent 配置逻辑集成在 `pkg/config/config.go` 中，通过 `LoadAgent()` 函数加载。这种设计避免了配置代码重复，因为 Agent 和 Server 共享 Temporal/Log 配置。

**配置结构：**
```go
type AgentConfig struct {
    TaskQueues      []string
    ID              string
    PluginDir       string
    AutoReloadPlugins bool
    MetricsPort     string
    ShutdownTimeout time.Duration
}
```

## Task Queue 路由机制 (ADR-0006)

**直接映射：** `runs-on: linux-amd64` → Task Queue 名称 `"linux-amd64"`

**Agent 配置示例：**
```yaml
agent:
  task_queues:
    - linux-amd64    # 接收 linux-amd64 任务
    - linux-common   # 接收通用 Linux 任务
```

**负载均衡：**
- 多个 Agent 可以监听同一个 Task Queue
- Temporal 自动分发任务到可用的 Worker
- Worker 心跳机制 (30 秒) 确保健康检测

## Workflow/Activity 注册

### Workflow 注册
```go
workerInstance.RegisterWorkflow(temporal.RunWorkflowExecutor)
```

**说明：** Agent 注册 Server 侧定义的 Workflow，用于执行完整的工作流编排。

### Activity 注册
```go
activities := temporal.NewActivities(logger)
workerInstance.RegisterActivity(activities.ExecuteStepActivity)
```

**说明：** Agent 执行单个 Step，通过 `ExecuteStepActivity` 调用节点插件。

## 并发控制

**Worker Options：**
```go
worker.Options{
    MaxConcurrentActivityExecutionSize: 100,  // 最多 100 个并发 Activity
    MaxConcurrentWorkflowTaskExecutionSize: 50,  // 最多 50 个并发 Workflow Task
    WorkerStopTimeout: cfg.Agent.ShutdownTimeout,  // 优雅关闭超时
}
```

## 优雅关闭流程

1. **接收信号** - SIGINT/SIGTERM 触发
2. **停止轮询** - 调用 `worker.Stop()` 停止接收新任务
3. **等待完成** - 使用 `sync.WaitGroup` 等待所有 Worker goroutine
4. **超时控制** - 超过 `shutdown_timeout` 强制关闭
5. **资源清理** - 关闭 Temporal 连接

## 日志规范

**启动日志：**
```json
{"level":"info","message":"Waterflow Agent starting","version":"v1.0.0"}
{"level":"info","message":"Configuration loaded","config_file":"/app/config/config.yaml"}
{"level":"info","message":"Connected to Temporal","host":"localhost:7233"}
{"level":"info","message":"Registered worker for task queue","task_queue":"linux-amd64"}
{"level":"info","message":"All workers started","worker_count":2}
```

**关闭日志：**
```json
{"level":"info","message":"Shutdown signal received"}
{"level":"info","message":"Shutting down agent workers"}
{"level":"info","message":"Stopping worker","index":0}
{"level":"info","message":"All workers stopped gracefully"}
{"level":"info","message":"Agent shutdown complete"}
```

## 测试策略

### 单元测试 (无需 Temporal Server)
- `TestPluginManager` - 插件扫描和验证
- `TestConnectToTemporal_Retry` - 连接重试逻辑
- `TestParseTaskQueues` - 队列解析和去重

### 集成测试 (需要 Temporal Server)
- `TestNewWorker_Integration` - Worker 创建和连接
- `TestWorkerShutdown_Integration` - 优雅关闭流程

**运行集成测试：**
```bash
# 启动 Temporal
docker-compose up -d temporal

# 运行集成测试
INTEGRATION_TEST=true go test -v ./internal/agent/...
```

## 依赖关系

**外部依赖：**
- `go.temporal.io/sdk` - Temporal Worker SDK
- `go.uber.org/zap` - 结构化日志

**内部依赖：**
- `pkg/temporal` - Temporal 客户端和 Activity 实现
- `pkg/config` - 配置管理
- `pkg/logger` - 日志系统

## 部署方式

### 开发环境
```bash
./bin/agent --config examples/configs/config.agent.example.yaml
```

### Docker 容器
```bash
docker run -v $(pwd)/examples/configs:/app/config waterflow/agent
```

### 生产部署 (systemd)
```bash
sudo systemctl start waterflow-agent
# 配置文件: /etc/waterflow/agent.yaml
```

## 故障排查

### Agent 无法连接 Temporal
**日志：** `Failed to connect to Temporal, retrying`
**检查：**
1. Temporal Server 是否运行
2. 配置的 `temporal.host` 是否正确
3. 网络连通性 (`telnet host port`)

### Task Queue 名称无效
**日志：** `invalid task queue name`
**原因：** Queue 名称只能包含字母、数字和连字符
**解决：** 使用 `linux-amd64` 而非 `linux_amd64`

### Worker 心跳失败
**日志：** `Heartbeat failed`
**原因：** 网络不稳定或 Temporal Server 压力过大
**解决：** 增加 `temporal.max_retries` 和检查网络
