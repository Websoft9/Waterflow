---
date: '2025-12-16'
author: 'Architecture Team'
status: 'Agent Architecture Specification'
---

# Waterflow Agent 架构设计

## 1. 核心定位

Agent 是 Temporal Worker 的业务封装,负责监听任务队列并执行工作流节点。

**职责:**
- 监听 Temporal Task Queue 拉取任务
- 执行 Workflow 和 Activity
- 路由节点到对应执行器
- 管理插件生命周期

**非职责:**
- 任务队列管理 (由 Temporal Server 负责)
- 调度决策 (由 Temporal Server 负责)
- 状态持久化 (由 Temporal Server 负责)

## 2. 架构设计

### 2.1 分层架构

```
Temporal Worker
  ↓
ExecuteNodeActivity (单一 Activity 定义)
  ↓
NodeRegistry (节点注册表)
  ↓
├─ Core Plugins (/opt/waterflow/plugins/core/)
└─ Custom Plugins (/opt/waterflow/plugins/custom/)
```

**设计要点:**
- 只注册 1 个 Activity 定义 (`ExecuteNodeActivity`)
- Workflow 循环调用该 Activity N 次(每个节点一次)
- 每次调用独立配置超时和重试策略
- NodeRegistry 动态管理所有节点执行器

### 2.2 进程模型

```
main goroutine
  ├─ PluginManager (fsnotify 监控插件目录)
  └─ Temporal Worker
       ├─ Poller Goroutines (10-20个) → Long Polling
       └─ Executor Goroutines (50-100个) → 并发执行
```

### 2.3 核心组件

**Temporal Worker**
- 配置: `MaxConcurrentActivityExecutionSize = 100`
- Long Polling 监听 Task Queue

**ExecuteNodeActivity**
```go
worker.RegisterActivity(&ExecuteNodeActivity{
    nodeRegistry: registry,
})
```

**NodeRegistry**
```go
type NodeRegistry struct {
    executors map[string]NodeExecutor
    mu        sync.RWMutex
}

func (r *NodeRegistry) Register(name string, executor NodeExecutor) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.executors[name] = executor
}

func (r *NodeRegistry) Get(name string) NodeExecutor {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.executors[name]
}
```

**PluginManager**
- 启动时扫描 `/opt/waterflow/plugins/` 加载所有 .so 文件
- 调用插件的 `Register()` 函数自动注册到 NodeRegistry
- 运行时监控文件变化自动热加载
- 自动上报元数据到 Server

```go
// 加载插件流程
func (pm *PluginManager) loadPlugin(path string) error {
    // 1. 加载 .so 文件
    p, err := plugin.Open(path)
    if err != nil {
        return err
    }
    
    // 2. 查找 Register 函数
    registerFunc, err := p.Lookup("Register")
    if err != nil {
        return err
    }
    
    // 3. 调用 Register 注册到 NodeRegistry
    register := registerFunc.(func(*NodeRegistry) error)
    if err := register(pm.nodeRegistry); err != nil {
        return err
    }
    
    // 4. 上报元数据到 Server
    pm.reportMetadata()
    
    return nil
}
```

## 3. 执行流程

### 3.1 Workflow 编排

```go
func JobWorkflow(ctx workflow.Context, job Job) error {
    execCtx := &ExecutionContext{
        Outputs: make(map[string]interface{}),
    }
    
    for i, step := range job.Steps {
        // 从 DSL 构建 ActivityOptions
        opts := workflow.ActivityOptions{
            StartToCloseTimeout: step.Timeout,
            HeartbeatTimeout:    step.HeartbeatTimeout,
            RetryPolicy: &temporal.RetryPolicy{
                MaximumAttempts:    step.Retry.MaxAttempts,
                InitialInterval:    step.Retry.InitialInterval,
                MaximumInterval:    step.Retry.MaximumInterval,
                BackoffCoefficient: step.Retry.BackoffCoefficient,
            },
        }
        ctx = workflow.WithActivityOptions(ctx, opts)
        
        // 调用 Activity 执行单个节点
        var result NodeResult
        err := workflow.ExecuteActivity(
            ctx,
            "ExecuteNodeActivity",
            step,
            execCtx,
        ).Get(ctx, &result)
        
        if err != nil {
            return fmt.Errorf("step %d (%s) failed: %w", i, step.Uses, err)
        }
        
        // 保存输出
        if step.ID != "" {
            execCtx.Outputs[step.ID] = result.Output
        }
    }
    
    return nil
}
```

### 3.2 Activity 实现

```go
type ExecuteNodeActivity struct {
    nodeRegistry *NodeRegistry
}

func (a *ExecuteNodeActivity) Execute(
    ctx context.Context,
    step Step,
    execCtx *ExecutionContext,
) (*NodeResult, error) {
    // 查找执行器
    executor := a.nodeRegistry.Get(step.Uses)
    if executor == nil {
        return nil, fmt.Errorf("unknown node: %s", step.Uses)
    }
    
    // 执行节点
    result, err := executor.Execute(ctx, step.With, execCtx)
    if err != nil {
        return nil, err
    }
    
    // 上报心跳
    activity.RecordHeartbeat(ctx, map[string]interface{}{
        "node":   step.Uses,
        "status": "completed",
    })
    
    return result, nil
}
```

### 3.3 DSL 配置映射

```yaml
steps:
  - id: build
    uses: docker/build
    timeout: 10m              # → StartToCloseTimeout
    heartbeat_timeout: 1m     # → HeartbeatTimeout
    retry:
      max_attempts: 3         # → MaximumAttempts
      initial_interval: 1s    # → InitialInterval
      maximum_interval: 30s   # → MaximumInterval
      backoff_coefficient: 2  # → BackoffCoefficient
```

## 4. 插件系统

### 4.1 插件开发

所有节点都是插件,不区分核心或第三方。

**开发步骤:**

1. 实现节点接口

```go
package main

import (
    "context"
    "waterflow/pkg/executor"
)

type SlackNotifyExecutor struct{}

func (e *SlackNotifyExecutor) Execute(
    ctx context.Context,
    params map[string]interface{},
    execCtx *executor.ExecutionContext,
) (*executor.NodeResult, error) {
    webhookURL := params["webhook_url"].(string)
    message := params["message"].(string)
    
    err := sendSlackMessage(webhookURL, message)
    if err != nil {
        return nil, err
    }
    
    return &executor.NodeResult{
        Output:   map[string]interface{}{"status": "sent"},
        ExitCode: 0,
    }, nil
}

func (e *SlackNotifyExecutor) Metadata() executor.NodeMetadata {
    return executor.NodeMetadata{
        Name:        "slack/notify",
        Version:     "1.0.0",
        Description: "Send notification to Slack channel",
        Parameters: map[string]executor.ParameterSpec{
            "webhook_url": {
                Type:        "string",
                Required:    true,
                Description: "Slack webhook URL",
            },
            "message": {
                Type:        "string",
                Required:    true,
                Description: "Message content",
            },
        },
        Outputs: map[string]executor.OutputSpec{
            "status": {
                Type:        "string",
                Description: "Send status",
            },
        },
    }
}

func Register(registry *executor.NodeRegistry) error {
    registry.Register("slack/notify", &SlackNotifyExecutor{})
    return nil
}
```

2. 编译插件

```bash
go build -buildmode=plugin -o slack-notify.so
```

3. 部署插件

```bash
# 复制到插件目录
cp slack-notify.so /opt/waterflow/plugins/custom/

# Agent 自动检测并加载 (< 1秒)
# DSL 立即可用
```

### 4.2 元数据驱动

Agent 加载插件时自动提取元数据上报到 Server:

```
Agent 加载插件
  ↓
提取 Metadata (name, parameters, outputs)
  ↓
gRPC 上报到 Server
  ↓
Server 更新 NodeSchemaRegistry
  ↓
DSL 验证器自动识别该节点
```

DSL 自动可用,无需手动开发 Schema:

```yaml
# workflow.yaml
steps:
  - uses: slack/notify  # 立即可用
    with:
      webhook_url: ${{ secrets.SLACK_WEBHOOK }}
      message: "Deploy completed!"
```

### 4.3 热加载机制

```go
func (pm *PluginManager) watchPlugins() {
    watcher, _ := fsnotify.NewWatcher()
    watcher.Add("/opt/waterflow/plugins/")
    
    for {
        select {
        case event := <-watcher.Events:
            if event.Op&fsnotify.Create == fsnotify.Create ||
               event.Op&fsnotify.Write == fsnotify.Write {
                if strings.HasSuffix(event.Name, ".so") {
                    pm.loadPlugin(event.Name)
                }
            }
        }
    }
}
```

**特性:**
- 自动检测文件变化
- 正在执行的任务继续使用旧执行器
- 新任务使用新执行器
- 零停机更新
- 无需额外端口或服务

**限制:**
- Go Plugin 无法真正卸载
- 建议定期重启 Agent (如每周一次)

## 5. GitHub Actions 兼容

### 5.1 实现策略

支持 GitHub Actions 生态,重点支持 JavaScript Actions (占 80%+)。

**技术方案:**
- JavaScript Actions: 直接用 Node.js 执行
- Docker Actions: 使用宿主机 Docker (挂载 socket)
- Composite Actions: 递归执行子步骤

### 5.2 Actions Runtime

```go
type GitHubActionExecutor struct {
    actionsCache string
}

func (e *GitHubActionExecutor) Execute(
    ctx context.Context,
    params map[string]interface{},
    execCtx *executor.ExecutionContext,
) (*executor.NodeResult, error) {
    actionRef := params["action"].(string)
    inputs := params["inputs"].(map[string]interface{})
    
    // 下载并缓存 Action
    actionDir, err := e.downloadAction(actionRef)
    if err != nil {
        return nil, err
    }
    
    // 解析 action.yml
    metadata, err := e.parseActionMetadata(actionDir)
    if err != nil {
        return nil, err
    }
    
    // 根据类型执行
    switch metadata.Runs.Using {
    case "node20", "node16":
        return e.executeJavaScriptAction(ctx, actionDir, metadata, inputs, execCtx)
    case "docker":
        return e.executeDockerAction(ctx, actionDir, metadata, inputs, execCtx)
    case "composite":
        return e.executeCompositeAction(ctx, actionDir, metadata, inputs, execCtx)
    }
}

// JavaScript Actions 执行
func (e *GitHubActionExecutor) executeJavaScriptAction(...) (*NodeResult, error) {
    env := e.buildActionEnv(inputs, execCtx)
    entrypoint := filepath.Join(actionDir, metadata.Runs.Main)
    
    cmd := exec.CommandContext(ctx, "node", entrypoint)
    cmd.Env = append(os.Environ(), env...)
    cmd.Dir = actionDir
    
    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, err
    }
    
    return &executor.NodeResult{
        Output:   e.parseActionOutputs(execCtx),
        ExitCode: 0,
    }, nil
}

// Docker Actions 执行
func (e *GitHubActionExecutor) executeDockerAction(...) (*NodeResult, error) {
    env := e.buildActionEnv(inputs, execCtx)
    image := metadata.Runs.Image
    
    if image == "Dockerfile" {
        image = fmt.Sprintf("waterflow-action-%s", metadata.Name)
        buildCmd := exec.CommandContext(ctx, "docker", "build", "-t", image, actionDir)
        if err := buildCmd.Run(); err != nil {
            return nil, err
        }
    }
    
    args := []string{"run", "--rm"}
    for _, envVar := range env {
        args = append(args, "-e", envVar)
    }
    args = append(args, "-v", fmt.Sprintf("%s:%s", execCtx.Workspace, "/github/workspace"))
    args = append(args, image)
    
    cmd := exec.CommandContext(ctx, "docker", args...)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, err
    }
    
    return &executor.NodeResult{
        Output:   e.parseActionOutputs(execCtx),
        ExitCode: 0,
    }, nil
}
```

### 5.3 部署配置

Docker Actions 需要挂载宿主机 Docker socket:

```yaml
# docker-compose.yml
services:
  agent:
    image: waterflow/agent:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
```
  hostPath:
    path: /var/run/docker.sock
    type: Socket
```

## 6. 容错与重试

### 6.1 节点级重试

每个节点独立配置重试策略:

```yaml
steps:
  - uses: docker/build
    retry:
      max_attempts: 3
      initial_interval: 1s
      maximum_interval: 30s
      backoff_coefficient: 2
```

执行示例:
```
step1 (docker/build) - 成功
step2 (docker/push)  - 失败 → 重试(1s后) → 失败 → 重试(2s后) → 成功
step3 (slack/notify) - 成功
```

### 6.2 超时控制

两层超时:

```yaml
jobs:
  deploy:
    timeout: 30m          # Workflow 总超时
    steps:
      - uses: docker/build
        timeout: 10m      # 节点独立超时
      - uses: docker/push
        timeout: 5m
```

### 6.3 Agent 离线处理

- Temporal Worker 心跳超时 (默认 30秒)
- 根据节点 RetryPolicy 自动重试
- 调度到其他可用 Agent

## 7. 日志与追踪

### 7.1 Temporal 内置能力

**Event History**

Temporal 自动记录所有事件:

```
Workflow: wf-123
├─ WorkflowExecutionStarted
├─ ActivityScheduled (id: step-build)
├─ ActivityStarted
├─ ActivityHeartbeatRecorded (progress: 50%)
├─ ActivityCompleted (result: {...})
├─ ActivityScheduled (id: step-deploy)
├─ ActivityStarted
├─ ActivityCompleted
└─ WorkflowExecutionCompleted
```

**Search Attributes**

```go
workflow.UpsertSearchAttributes(ctx, map[string]interface{}{
    "CustomWorkflowName": job.Name,
    "CustomStatus":       "running",
    "CustomBranch":       job.Trigger.Branch,
    "CustomCommit":       job.Trigger.Commit,
})
```

查询:
```go
query := "CustomStatus = 'running'"
resp := temporalClient.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
    Query: query,
})
```

**Activity Heartbeat**

```go
activity.RecordHeartbeat(ctx, map[string]interface{}{
    "status":   "starting",
    "step":     step.ID,
    "node":     step.Uses,
})

// 执行节点...

activity.RecordHeartbeat(ctx, map[string]interface{}{
    "status":      "completed",
    "duration_ms": result.Duration,
    "exit_code":   result.ExitCode,
})
```

### 7.2 节点输出日志

Agent 将节点 stdout/stderr 写入文件:

```go
func (a *ExecuteNodeActivity) Execute(
    ctx context.Context,
    step Step,
    execCtx *ExecutionContext,
) (*NodeResult, error) {
    // 上报开始
    activity.RecordHeartbeat(ctx, HeartbeatInfo{
        Step:   step.ID,
        Status: "starting",
    })
    
    // 创建日志文件
    logFile := os.Create(fmt.Sprintf("/var/log/waterflow/%s-%s.log", 
        execCtx.WorkflowID, step.ID))
    defer logFile.Close()
    
    // 执行节点
    executor := a.nodeRegistry.Get(step.Uses)
    result, err := executor.Execute(ctx, step.With, &ExecutionContext{
        Stdout: logFile,
        Stderr: logFile,
    })
    
    // 上报完成
    activity.RecordHeartbeat(ctx, HeartbeatInfo{
        Step:       step.ID,
        Status:     "completed",
        DurationMs: result.Duration,
        ExitCode:   result.ExitCode,
    })
    
    return result, err
}
```

Server 端读取日志:

```go
func (s *Server) getStepLogs(workflowID, stepID string) ([]string, error) {
    logPath := fmt.Sprintf("/var/log/waterflow/%s-%s.log", workflowID, stepID)
    data, err := os.ReadFile(logPath)
    if err != nil {
        return nil, err
    }
    return strings.Split(string(data), "\n"), nil
}
```

### 7.3 数据流

**执行追踪:**
```
Agent 执行 → Temporal 自动记录 Event → Server 查询 Temporal API
```

**节点输出:**
```
Agent 写文件 → Server 读取文件
```

## 8. 部署方案

### 8.1 单服务器部署

```yaml
# docker-compose.yml
version: '3.8'
services:
  temporal:
    image: temporalio/auto-setup:latest
  
  waterflow-server:
    image: waterflow/server:latest
    environment:
      TEMPORAL_ADDRESS: temporal:7233
  
  waterflow-agent:
    image: waterflow/agent:latest
    environment:
      TEMPORAL_ADDRESS: temporal:7233
      SERVER_ADDRESS: waterflow-server:9090
    volumes:
      - agent-logs:/var/log/waterflow
      - /var/run/docker.sock:/var/run/docker.sock

volumes:
  agent-logs:
```

### 8.2 分布式部署

```yaml
# docker-compose.yml
services:
  waterflow-agent:
    image: waterflow/agent:latest
    environment:
      TEMPORAL_ADDRESS: temporal.internal:7233
      TASK_QUEUE: web-servers
      TAGS: web-servers,production
    volumes:
      - /opt/waterflow/plugins/custom:/opt/waterflow/plugins/custom
      - /var/run/docker.sock:/var/run/docker.sock
    deploy:
      replicas: 4
```

## 9. 配置

### 9.1 Agent 配置

```yaml
# agent.yaml
temporal:
  address: temporal:7233
  namespace: default

server:
  address: waterflow-server:9090

worker:
  task_queue: default
  max_concurrent_activities: 100
  max_concurrent_workflows: 10

plugins:
  dir: /opt/waterflow/plugins
  reload: true

logging:
  output_type: file
  output_path: /var/log/waterflow
  format: json
```

### 9.2 环境变量

```bash
TEMPORAL_ADDRESS=temporal:7233
SERVER_ADDRESS=waterflow-server:9090
TASK_QUEUE=default
PLUGIN_DIR=/opt/waterflow/plugins
LOG_OUTPUT_TYPE=file
LOG_OUTPUT_PATH=/var/log/waterflow
```

## 10. 设计原则

1. **职责单一** - Agent 专注节点执行,不做调度
2. **依赖 Temporal** - 充分利用 Temporal 能力
3. **插件化** - 所有节点都是插件
4. **热加载** - 插件动态加载无需重启
5. **独立配置** - 每个节点独立超时/重试
6. **元数据驱动** - DSL Schema 自动生成
7. **生态兼容** - 支持 GitHub Actions
8. **轻量设计** - Agent 只执行和写日志
9. **利用原生** - 使用 Temporal Event History 和 Heartbeat

---

**文档版本:** v6.0  
**最后更新:** 2025-12-16  
**状态:** 生产就绪
