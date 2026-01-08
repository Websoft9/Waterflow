# Event Handlers 指南

## 概述

EventHandler 接口允许监听工作流生命周期事件：
- **workflow.started** - 工作流开始执行
- **workflow.completed** - 工作流成功完成
- **workflow.failed** - 工作流执行失败

## 内置实现

### NoOpEventHandler（默认）

不执行任何操作的事件处理器，用于禁用事件通知。

```yaml
events:
  handler_type: noop
```

### WebhookEventHandler

通过 HTTP POST 将事件发送到 Webhook 端点。

```yaml
events:
  handler_type: webhook
  webhook:
    url: https://hooks.slack.com/services/YOUR/WEBHOOK/URL
    timeout: 5s
    headers:
      X-Custom-Header: waterflow
      Authorization: Bearer YOUR_TOKEN
```

**请求格式：**

```json
{
  "event_type": "workflow.started",
  "workflow_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-01-01T12:00:00Z",
  "workflow_name": "my-workflow",
  "metadata": {
    "user": "alice"
  }
}
```

## Slack 集成示例

### 创建 Slack Incoming Webhook

1. 打开 Slack App 设置：https://api.slack.com/apps
2. 创建新 App 或选择现有 App
3. 启用 "Incoming Webhooks"
4. 添加 Webhook 到目标频道
5. 复制 Webhook URL

### 配置 Waterflow

```yaml
events:
  handler_type: webhook
  webhook:
    url: https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXX
    timeout: 10s
```

### 测试集成

提交一个简单的工作流测试：

```bash
waterflow submit examples/hello-world.yaml
```

Slack 频道将收到通知：
- "Workflow my-workflow started (550e8400-e29b-41d4-a716-446655440000)"
- "Workflow my-workflow completed successfully"

## 异步处理

所有事件处理都是**异步非阻塞**的：
- 事件分发使用独立 goroutine
- 处理超时默认 10 秒
- 错误记录到日志，不影响工作流执行

## 自定义 EventHandler

实现 `events.EventHandler` 接口：

```go
type EventHandler interface {
    OnWorkflowStart(ctx context.Context, event *WorkflowStartEvent) error
    OnWorkflowComplete(ctx context.Context, event *WorkflowCompleteEvent) error
    OnWorkflowFailed(ctx context.Context, event *WorkflowFailedEvent) error
}
```

示例：自定义日志处理器

```go
type LogEventHandler struct {
    logger *zap.Logger
}

func (h *LogEventHandler) OnWorkflowStart(ctx context.Context, event *WorkflowStartEvent) error {
    h.logger.Info("Workflow started",
        zap.String("workflow_id", event.WorkflowID),
        zap.String("workflow_name", event.WorkflowName),
    )
    return nil
}
```

## 故障排查

### Webhook 超时

增加超时时间：

```yaml
events:
  webhook:
    timeout: 30s  # 默认 5s
```

### 验证 Webhook 收到请求

使用 webhook.site 测试：

1. 访问 https://webhook.site
2. 复制唯一 URL
3. 配置为 Waterflow webhook.url
4. 提交工作流
5. 查看 webhook.site 捕获的请求

### 调试日志

启用 debug 日志查看事件分发：

```yaml
log:
  level: debug
```

查看日志：

```
DEBUG Event dispatched {"event_type": "workflow.started", "workflow_id": "..."}
ERROR Event handler failed {"event_type": "workflow.started", "error": "..."}
```

## 最佳实践

1. **幂等性**: Webhook 端点应设计为幂等（可能收到重复事件）
2. **快速响应**: Webhook 应在 5 秒内响应（避免超时）
3. **错误处理**: 事件处理失败不影响工作流执行
4. **安全**: 使用 HTTPS 和认证头保护 Webhook 端点
