# Log Handlers 指南

## 概述

LogHandler 接口允许收集工作流执行日志并发送到日志聚合系统：
- 集中化日志管理
- 支持自定义日志处理逻辑
- 批量缓冲优化性能
- 文件轮转支持

## 日志结构

LogEntry 包含以下字段：

```json
{
  "timestamp": "2026-01-08T11:28:33.277Z",
  "level": "INFO",
  "workflow_id": "wf-abc123",
  "run_id": "temporal-run-id",
  "job_id": "deploy",
  "step_id": "step-1",
  "node_type": "exec/shell",
  "message": "Executing command",
  "metadata": {
    "exit_code": 0,
    "duration_ms": 1234
  }
}
```

## 内置实现

### StdoutLogHandler

输出 JSON 日志到标准输出,适合容器环境：

```go
handler := logs.NewStdoutLogHandler(logs.StdoutConfig{
    BufferSize: 100,  // 缓冲100条后批量输出
    Pretty:     false, // false=单行JSON,true=格式化
})
defer handler.Close()
```

**特性:**
- 批量缓冲减少 I/O
- JSON 格式便于解析
- 容器环境友好（stdout → Docker/K8s）

### FileLogHandler

写入文件并自动轮转：

```go
handler, err := logs.NewFileLogHandler(logs.FileConfig{
    Path:       "/var/log/waterflow/workflow.jsonl",
    BufferSize: 100,    // 缓冲100条
    MaxSizeMB:  100,    // 100MB轮转
})
defer handler.Close()
```

**特性:**
- 批量缓冲写入
- 自动日志轮转（超过 MaxSizeMB）
- 旧日志保存为 `.old` 文件

## 使用示例

### 记录工作流日志

```go
import (
    "context"
    "time"
    "github.com/Websoft9/waterflow/pkg/logs"
)

handler := logs.NewStdoutLogHandler(logs.StdoutConfig{
    BufferSize: 50,
})
defer handler.Close()

// 工作流开始
handler.OnLog(context.Background(), &logs.LogEntry{
    Timestamp:  time.Now(),
    Level:      logs.LogLevelInfo,
    WorkflowID: workflowID,
    Message:    "Workflow started",
})

// 步骤执行
handler.OnLog(context.Background(), &logs.LogEntry{
    Timestamp:  time.Now(),
    Level:      logs.LogLevelInfo,
    WorkflowID: workflowID,
    JobID:      "build",
    StepID:     "compile",
    NodeType:   "exec/shell",
    Message:    "Compiling application",
    Metadata: map[string]interface{}{
        "command": "go build",
    },
})

// 错误日志
handler.OnLog(context.Background(), &logs.LogEntry{
    Timestamp:  time.Now(),
    Level:      logs.LogLevelError,
    WorkflowID: workflowID,
    JobID:      "build",
    Message:    "Build failed",
    Metadata: map[string]interface{}{
        "exit_code": 1,
        "error":     "compilation error",
    },
})
```

### 容器环境（Docker/Kubernetes）

使用 StdoutLogHandler,配合日志采集器：

**docker-compose.yml:**
```yaml
services:
  waterflow:
    image: waterflow/server:latest
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

**Kubernetes + Fluentd:**
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: waterflow-server
spec:
  containers:
  - name: server
    image: waterflow/server:latest
    # Fluentd 自动采集 stdout
```

### 本地开发（文件日志）

```bash
# 配置
export WATERFLOW_LOG_HANDLER=file
export WATERFLOW_LOG_FILE=/tmp/waterflow.jsonl

# 运行
./bin/server

# 查看日志
tail -f /tmp/waterflow.jsonl | jq .
```

## 集成企业日志系统

### ELK Stack（Elasticsearch + Logstash + Kibana）

**Filebeat 配置:**
```yaml
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /var/log/waterflow/*.jsonl
  json.keys_under_root: true
  json.add_error_key: true

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "waterflow-logs-%{+yyyy.MM.dd}"
```

### Grafana Loki

**Promtail 配置:**
```yaml
scrape_configs:
- job_name: waterflow
  static_configs:
  - targets:
      - localhost
    labels:
      job: waterflow
      __path__: /var/log/waterflow/*.jsonl
  pipeline_stages:
  - json:
      expressions:
        level: level
        workflow_id: workflow_id
```

## 性能优化

### 批量缓冲

减少 I/O 调用次数：

```go
// 推荐: BufferSize 50-200
handler := logs.NewStdoutLogHandler(logs.StdoutConfig{
    BufferSize: 100, // 累积100条后一次性输出
})
```

### 异步写入

LogHandler 调用应该在独立 goroutine：

```go
go func() {
    for logEntry := range logChannel {
        handler.OnLog(context.Background(), logEntry)
    }
}()
```

## 故障排查

### 日志丢失

**原因:** 程序崩溃时缓冲未刷新

**解决:** 确保调用 `Close()`:

```go
defer handler.Close() // 刷新缓冲并关闭
```

### 文件权限错误

```
failed to open log file: permission denied
```

**解决:**
```bash
mkdir -p /var/log/waterflow
chmod 755 /var/log/waterflow
```

### 日志文件过大

**问题:** 磁盘空间耗尽

**解决:** 设置合理的 MaxSizeMB：

```go
handler, _ := logs.NewFileLogHandler(logs.FileConfig{
    Path:      "/var/log/waterflow/workflow.jsonl",
    MaxSizeMB: 50, // 50MB 轮转
})
```

## 自定义 LogHandler

实现 `logs.LogHandler` 接口：

```go
type CustomLogHandler struct {
    // your fields
}

func (h *CustomLogHandler) OnLog(ctx context.Context, entry *logs.LogEntry) error {
    // 处理日志逻辑
    // 例如: 发送到 Kafka、写入数据库等
    return nil
}

func (h *CustomLogHandler) Close() error {
    // 清理资源
    return nil
}
```

## 最佳实践

1. **容器环境用 StdoutLogHandler** - 日志采集器统一处理
2. **本地开发用 FileLogHandler** - 方便查看历史日志
3. **设置合理的 BufferSize** - 平衡性能和内存占用
4. **始终调用 Close()** - 确保日志完整性
5. **结构化 Metadata** - 便于日志查询和分析
6. **日志分级** - 合理使用 DEBUG/INFO/WARN/ERROR
