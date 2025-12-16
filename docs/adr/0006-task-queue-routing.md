# ADR-0006: Task Queue 路由机制

**状态:** ✅ 已采纳  
**日期:** 2025-12-15  
**决策者:** 架构团队  

## 背景

Waterflow 需要支持服务器组(Agent Group),将任务路由到特定的 Agent 执行:

```yaml
jobs:
  build-linux:
    runs-on: linux-amd64  # 需要路由到 Linux AMD64 服务器组
  
  build-mac:
    runs-on: macos-arm64  # 需要路由到 macOS ARM64 服务器组
```

Temporal 提供了 Task Queue 机制实现任务路由。需要决定:
1. **Queue 命名规则** - 如何从 `runs-on` 映射到 Task Queue
2. **Queue 管理** - 静态预定义 vs 动态创建
3. **Worker 注册** - Agent 如何声明支持的 Queue

## 决策

采用 **直接映射策略**: `runs-on` 值直接作为 Temporal Task Queue 名称。

## 理由

### 核心优势:

1. **简单直观**
   - 无需额外映射表
   - 配置即文档
   - 易于理解和调试

2. **灵活性**
   - 用户可以自定义任意 Queue 名称
   - 支持动态添加新服务器组
   - 无需修改 Server 配置

3. **Temporal 原生支持**
   - Task Queue 是 Temporal 的一等公民
   - 自动负载均衡
   - 内置健康检查

4. **零配置**
   - Server 无需维护 Queue 列表
   - Agent 启动时声明支持的 Queue
   - 动态发现,自动路由

### 与其他方案对比:

| 方案 | 优点 | 缺点 | 决策 |
|------|------|------|------|
| **直接映射** | 简单,灵活,零配置 | Queue 名称暴露给用户 | ✅ 选择 |
| 标签匹配 | 表达力强 | 复杂,需要调度器 | ❌ |
| 预定义 Queue | 可控 | 不灵活,需要预配置 | ❌ |

## 后果

### 正面影响:

✅ **简单实现** - 无需额外的路由逻辑  
✅ **动态扩展** - 新增服务器组无需修改 Server  
✅ **灵活配置** - 用户可以自定义 Queue 命名  
✅ **负载均衡** - Temporal 自动分发任务到 Worker  

### 负面影响:

⚠️ **Queue 名称管理** - 用户需要知道可用的 Queue 名称  
⚠️ **无效 Queue** - 如果 `runs-on` 指定的 Queue 没有 Worker,任务会一直等待  

### 风险缓解:

- **Queue 发现 API**: Server 提供 API 查询当前活跃的 Queue
- **超时保护**: Job 级别超时,避免无限等待
- **告警机制**: 任务等待超过阈值时告警

## 实现示例

### DSL 中指定 Queue:

```yaml
jobs:
  build-linux:
    runs-on: linux-amd64      # Task Queue: linux-amd64
    steps:
      - uses: checkout@v1
  
  build-windows:
    runs-on: windows-x64      # Task Queue: windows-x64
    steps:
      - uses: checkout@v1
  
  gpu-training:
    runs-on: gpu-a100         # Task Queue: gpu-a100
    steps:
      - uses: run@v1
```

### Server 中启动 Workflow:

```go
// Server 启动 Job Workflow 时指定 Task Queue
func (s *Server) StartJob(ctx context.Context, job *Job) error {
    workflowOptions := client.StartWorkflowOptions{
        ID:        fmt.Sprintf("job-%s", job.ID),
        TaskQueue: job.RunsOn,  // 直接使用 runs-on 值
    }
    
    _, err := s.temporalClient.ExecuteWorkflow(ctx, workflowOptions, "RunJobWorkflow", job)
    return err
}
```

### Agent 中注册 Worker:

```go
// Agent 启动时声明支持的 Task Queue
func (w *Worker) Start(taskQueues []string) error {
    for _, queue := range taskQueues {
        worker := worker.New(w.temporalClient, queue, worker.Options{})
        
        // 注册 Workflow
        worker.RegisterWorkflow(RunJobWorkflow)
        
        // 注册 Activity
        worker.RegisterActivity(w.ExecuteNode)
        
        // 启动 Worker(非阻塞)
        go worker.Run(worker.InterruptCh())
    }
    
    return nil
}
```

### Agent 配置文件:

```yaml
# /etc/waterflow/agent.yaml
server: https://waterflow.example.com
task-queues:
  - linux-amd64      # 支持 Linux AMD64 任务
  - linux-common     # 支持通用 Linux 任务
  - gpu-a100         # 支持 GPU 任务(如果有 GPU)
```

### Agent 启动:

```bash
# 启动 Agent,指定支持的 Queue
waterflow-agent start \
  --task-queue linux-amd64 \
  --task-queue linux-common \
  --task-queue gpu-a100
```

## Queue 命名规范

### 推荐命名:

```yaml
# 操作系统 + 架构
runs-on: linux-amd64
runs-on: linux-arm64
runs-on: macos-arm64
runs-on: windows-x64

# 功能特性
runs-on: gpu-a100
runs-on: gpu-v100
runs-on: high-memory

# 环境
runs-on: production
runs-on: staging

# 自定义
runs-on: my-custom-group
```

### 命名约束:

```go
// 验证 Queue 名称
func ValidateQueueName(name string) error {
    // Temporal 要求: 字母数字和连字符,长度 < 256
    re := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)
    if !re.MatchString(name) {
        return fmt.Errorf("invalid queue name: %s", name)
    }
    if len(name) > 255 {
        return fmt.Errorf("queue name too long: %s", name)
    }
    return nil
}
```

## Queue 发现机制

### Server 提供 API 查询活跃 Queue:

```go
// API: GET /api/v1/queues
type QueueInfo struct {
    Name          string    `json:"name"`
    WorkerCount   int       `json:"worker_count"`
    PendingTasks  int       `json:"pending_tasks"`
    LastHeartbeat time.Time `json:"last_heartbeat"`
}

func (s *Server) ListQueues(ctx context.Context) ([]QueueInfo, error) {
    // 从 Temporal 获取所有 Task Queue 信息
    resp, err := s.temporalClient.DescribeTaskQueue(ctx, &workflowservice.DescribeTaskQueueRequest{
        Namespace: "default",
        TaskQueue: &taskqueuepb.TaskQueue{Name: "*"},
    })
    
    // 解析返回的 Queue 列表
    var queues []QueueInfo
    for _, pollerInfo := range resp.Pollers {
        queues = append(queues, QueueInfo{
            Name:          pollerInfo.TaskQueue,
            WorkerCount:   len(resp.Pollers),
            LastHeartbeat: pollerInfo.LastAccessTime.AsTime(),
        })
    }
    
    return queues, nil
}
```

### CLI 工具查询:

```bash
# 查看可用的 Task Queue
waterflow queue list

# 输出:
# NAME            WORKERS  PENDING  LAST_HEARTBEAT
# linux-amd64     3        5        2s ago
# linux-arm64     1        0        10s ago
# gpu-a100        2        12       1s ago
```

## 错误处理

### 场景 1: Queue 没有 Worker

```yaml
jobs:
  build:
    runs-on: non-existent-queue  # 不存在的 Queue
```

**问题:** 任务会一直等待,永不执行

**解决方案:**

1. **启动时验证** (可选):

```go
func (s *Server) ValidateWorkflow(wf *Workflow) error {
    for _, job := range wf.Jobs {
        // 检查 Queue 是否有活跃的 Worker
        hasWorker, err := s.checkQueueHasWorker(job.RunsOn)
        if err != nil {
            return err
        }
        if !hasWorker {
            log.Warnf("Queue %s has no active workers", job.RunsOn)
            // 不阻止提交,只是警告
        }
    }
    return nil
}
```

2. **Job 级别超时**:

```yaml
jobs:
  build:
    runs-on: non-existent-queue
    timeout-minutes: 30  # 30 分钟后超时失败
```

3. **告警**:

```go
// 监控任务等待时间
func (s *Server) MonitorPendingJobs() {
    for {
        jobs := s.getPendingJobs()
        for _, job := range jobs {
            if job.WaitingTime() > 10*time.Minute {
                s.alertManager.Alert(fmt.Sprintf(
                    "Job %s waiting on queue %s for %v",
                    job.ID, job.RunsOn, job.WaitingTime(),
                ))
            }
        }
        time.Sleep(1 * time.Minute)
    }
}
```

### 场景 2: Worker 突然下线

**问题:** 任务执行中 Worker 崩溃

**Temporal 自动处理:**
- Worker 心跳超时后,Temporal 自动重新调度任务到其他 Worker
- 无需额外代码

## 高级场景(未来)

### 场景 1: 标签匹配

```yaml
# 未来版本可能支持
jobs:
  build:
    runs-on:
      labels:
        os: linux
        arch: amd64
        gpu: true
```

**实现:** 需要额外的调度器,在 Server 中匹配标签和 Queue

### 场景 2: 动态 Queue 分配

```yaml
# 未来版本可能支持
jobs:
  build:
    runs-on: auto  # 自动选择可用的 Queue
```

**实现:** Server 选择负载最低的 Queue

## 替代方案

### 方案 A: 标签匹配系统 (被拒绝)

Agent 声明标签,Job 指定需求:

```yaml
# Agent 配置
labels:
  os: linux
  arch: amd64
  gpu: a100

# DSL
jobs:
  build:
    runs-on:
      os: linux
      arch: amd64
```

**被拒绝原因:**
- ❌ 需要自己实现调度器(匹配标签)
- ❌ Temporal 不支持,需要在 Server 层实现
- ❌ 复杂度高,MVP 不需要

### 方案 B: 预定义 Queue 列表 (被拒绝)

Server 配置固定的 Queue 列表:

```yaml
# Server 配置
task-queues:
  - linux-amd64
  - linux-arm64
  - windows-x64
```

**被拒绝原因:**
- ❌ 不灵活,添加新 Queue 需要修改配置
- ❌ 用户无法自定义 Queue
- ❌ 增加运维成本

## 参考资料

- [Temporal Task Queue 文档](https://docs.temporal.io/docs/concepts/task-queues/)
- [PRD: 服务器组支持](../prd.md#服务器组支持)
- [Epic 2: Server 实现](../epics.md#epic-2-server-实现)
