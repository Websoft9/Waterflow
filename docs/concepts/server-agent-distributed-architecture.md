# Server-Agent 分布式架构详解

**文档说明:** 本文档详细解释 Waterflow 的 Server-Agent 分布式架构，特别是 YAML 存储、工作流执行、定时任务触发等关键机制。

---

## 1. 架构概览

### 1.1 拓扑结构

```
┌──────────────────────────────────────────────────────────┐
│                   Waterflow Server                        │
│                    (中心调度节点)                          │
│                                                           │
│  ┌─────────────────────────────────────────────────┐     │
│  │  YAML 工作流存储 (单一数据源)                     │     │
│  │  ├─ Git 仓库 (推荐)                               │     │
│  │  ├─ 文件系统 (/opt/waterflow/workflows/)         │     │
│  │  └─ 数据库 (SQLite/PostgreSQL)                   │     │
│  └──────────────────────┬──────────────────────────┘     │
│                         │                                │
│  ┌──────────────────────▼──────────────────────────┐     │
│  │  Temporal Server (工作流引擎)                     │     │
│  │  ├─ Workflow 编排 (Server 端执行)                │     │
│  │  ├─ Schedule 管理 (统一定时触发)                  │     │
│  │  ├─ Task Queue 路由 (分发任务到 Agent)            │     │
│  │  └─ 重试和容错 (自动处理失败)                     │     │
│  └──────────────────────┬──────────────────────────┘     │
│                         │                                │
└─────────────────────────┼─────────────────────────────────┘
                          │
                   gRPC (Task Queue)
                          │
          ┌───────────────┼───────────────┐
          │               │               │
     ┌────▼────┐     ┌────▼────┐    ┌────▼────┐
     │ Agent 1 │     │ Agent 2 │    │Agent 100│
     │ web-01  │     │ web-02  │    │ web-100 │
     │         │     │         │    │         │
     │ Task Queue:   Task Queue:    Task Queue:│
     │ linux-amd64   linux-amd64    linux-amd64│
     └─────────┘     └─────────┘    └─────────┘
       执行           执行           执行
      Activity      Activity       Activity
```

### 1.2 职责划分

| 组件 | 职责 | 存储 YAML | 解析 YAML | 执行 Workflow | 执行 Activity |
|------|------|----------|----------|--------------|--------------|
| **Waterflow Server** | 中心调度 | ✅ 是 | ✅ 是 | ❌ 否（Temporal Worker 执行） | ❌ 否（分发到 Agent） |
| **Temporal Server** | 工作流引擎 | ❌ 否 | ❌ 否 | ✅ 是（调度编排） | ❌ 否（路由到 Agent） |
| **Agent Worker** | 执行节点 | ❌ 否 | ❌ 否 | ❌ 否 | ✅ 是（本地执行） |

---

## 2. YAML 存储策略

### 2.1 存储方案对比

| 方案 | 存储位置 | 优势 | 劣势 | 推荐场景 |
|------|---------|------|------|---------|
| **Git 仓库** | GitHub/GitLab | ✅ 版本控制<br>✅ Code Review<br>✅ GitOps | ⚠️ 需 Git 基础设施<br>⚠️ Webhook 同步 | DevOps 团队<br>企业级生产环境 |
| **文件系统** | /opt/waterflow/workflows/ | ✅ 简单直接<br>✅ 零依赖<br>✅ 高性能 | ⚠️ 无版本控制<br>⚠️ 多节点同步困难 | 小团队<br>单机部署 |
| **数据库** | SQLite/PostgreSQL | ✅ 集中管理<br>✅ 多租户隔离<br>✅ 丰富查询 | ⚠️ 架构复杂<br>⚠️ 失去 GitOps | SaaS 平台<br>多租户场景 |

### 2.2 推荐方案：Git 仓库存储

#### 工作流

```bash
# 1. 创建 Git 仓库
mkdir waterflow-workflows && cd waterflow-workflows
git init

# 2. 创建 YAML 工作流
cat > deploy.yaml <<EOF
name: Deploy Application
on:
  schedule:
    - cron: "0 2 * * *"
      timezone: "UTC"
jobs:
  deploy:
    runs-on: linux-amd64
    steps:
      - name: Deploy
        uses: shell@v1
        with:
          cmd: docker pull nginx && docker restart nginx
EOF

# 3. 提交到 Git
git add deploy.yaml
git commit -m "Add deployment workflow"
git push origin main

# 4. Server 自动同步
# Waterflow Server 检测到 Webhook 触发
# → git pull 拉取最新代码
# → 解析 on.schedule 字段
# → 自动创建 Temporal Schedule
```

#### Server 端配置

```yaml
# /etc/waterflow/server.yaml
workflows:
  storage:
    type: git
    git:
      repository: https://github.com/org/waterflow-workflows.git
      branch: main
      local_path: /opt/waterflow/workflows
      sync_interval: 5m  # 每 5 分钟自动 pull
      webhook_secret: "your-secret"  # GitHub Webhook 验证

temporal:
  host: localhost:7233
  namespace: default
```

#### Agent 端配置（不需要访问 Git）

```yaml
# /etc/waterflow/agent.yaml
temporal:
  host: waterflow-server.example.com:7233  # 连接 Server 端 Temporal
  namespace: default

agent:
  task_queues:
    - linux-amd64  # 监听此队列，接收 Server 分发的任务
  max_concurrent: 10
```

---

## 3. 工作流执行流程

### 3.1 完整执行链路

#### 场景：在 100 台 Web 服务器上部署应用

```yaml
# deploy.yaml
name: Deploy to Production
on:
  schedule:
    - cron: "0 2 * * *"
jobs:
  deploy:
    runs-on: linux-amd64
    steps:
      - name: Pull Docker Image
        uses: shell@v1
        with:
          cmd: docker pull myapp:latest
      - name: Restart Container
        uses: shell@v1
        with:
          cmd: docker restart myapp
```

#### Step 1: Server 端加载 YAML

```go
// Waterflow Server 启动时
func (s *Server) Start() error {
    // 1. 克隆 Git 仓库
    git.Clone("https://github.com/org/workflows.git", "/opt/waterflow/workflows")
    
    // 2. 扫描所有 YAML 文件
    files := filepath.Glob("/opt/waterflow/workflows/*.yaml")
    
    // 3. 解析每个 YAML
    for _, file := range files {
        yamlContent := os.ReadFile(file)
        wf := dsl.Parse(yamlContent)
        
        // 4. 检查 on.schedule 字段
        if wf.On != nil && wf.On.Schedule != nil {
            // 5. 自动创建 Temporal Schedule
            s.createSchedule(wf)
        }
    }
}

func (s *Server) createSchedule(wf *dsl.Workflow) {
    scheduleClient.Create(ctx, &client.ScheduleOptions{
        ID: fmt.Sprintf("%s-auto", wf.Name),
        Spec: &client.ScheduleSpec{
            CronExpressions: []string{"0 2 * * *"},
            Timezone:        "UTC",  // ✅ 统一时区
        },
        Action: &client.ScheduleWorkflowAction{
            Workflow:  RunWorkflowExecutor,
            Args:      []interface{}{wf},  // ✅ 解析后的 Workflow 对象
            TaskQueue: "default",  // Workflow 在 Server 端执行
        },
    })
}
```

#### Step 2: Temporal Schedule 触发

```go
// UTC 02:00:00 时 Temporal 自动触发
┌─ Temporal Server ─────────────────────────┐
│                                           │
│ Schedule ID: "Deploy to Production-auto"  │
│ Cron: "0 2 * * *"                         │
│ Timezone: UTC                             │
│                                           │
│ ✅ 触发时间: 2026-02-02 02:00:00 UTC      │
│ ✅ 只触发一次（分布式锁）                  │
│                                           │
│ 创建 Workflow Execution:                  │
│   RunWorkflowExecutor(wf)                │
│                                           │
└───────────────────────────────────────────┘
```

#### Step 3: Server 端执行 Workflow 编排

```go
// Server 端 Temporal Worker 执行
func RunWorkflowExecutor(ctx workflow.Context, wf *dsl.Workflow) error {
    // wf 对象已包含完整的 YAML 定义
    // Agent 永远不需要访问 YAML 文件
    
    for _, job := range wf.Jobs {
        // job.RunsOn = "linux-amd64"
        
        for _, step := range job.Steps {
            // 创建 Activity 并路由到 Agent
            activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
                TaskQueue:           job.RunsOn,  // "linux-amd64"
                StartToCloseTimeout: 10 * time.Minute,
                RetryPolicy: &temporal.RetryPolicy{
                    MaximumAttempts: 3,  // ✅ 失败自动重试 3 次
                },
            })
            
            // 执行 Activity（在 Agent 端）
            var result StepResult
            err := workflow.ExecuteActivity(activityCtx, "ExecuteStepActivity", step).Get(ctx, &result)
            if err != nil {
                return fmt.Errorf("step %s failed: %w", step.Name, err)
            }
        }
    }
    
    return nil
}
```

#### Step 4: Temporal 路由任务到 Agent

```
┌─ Temporal Server ──────────────────────────────────┐
│                                                    │
│ Task Queue: "linux-amd64"                          │
│ ├─ Task 1 (Pull Docker Image) → 等待分配           │
│ ├─ Task 2 (Pull Docker Image) → 等待分配           │
│ ├─ ...                                             │
│ └─ Task 100 (Pull Docker Image) → 等待分配         │
│                                                    │
│ ✅ Temporal 自动负载均衡                            │
│ ✅ 每个任务只分配一次                               │
│ ✅ Agent 失败自动重新分配                           │
│                                                    │
└────────────────────────────────────────────────────┘
         │
         ├─────────────┬─────────────┐
         ↓             ↓             ↓
    Agent 1        Agent 2       Agent 100
    (web-01)       (web-02)      (web-100)
    
    轮询获取任务 → 先到先得
```

#### Step 5: Agent 执行 Activity

```go
// Agent 端 Temporal Worker
func (a *Activities) ExecuteStepActivity(ctx context.Context, step *dsl.Step) (*StepResult, error) {
    // ✅ step 参数已包含所有必要信息
    // ✅ 不需要访问 YAML 文件
    
    // step.Name = "Pull Docker Image"
    // step.Uses = "shell@v1"
    // step.With = {cmd: "docker pull myapp:latest"}
    
    // 执行节点
    result, err := a.nodeRegistry.Execute(step.Uses, step.With)
    if err != nil {
        return nil, fmt.Errorf("execute node failed: %w", err)
    }
    
    return &StepResult{
        Status: "success",
        Output: result.Stdout,
    }, nil
}
```

---

## 4. 定时任务统一触发

### 4.1 问题：100 台服务器时间同步

**错误设计（GitHub Actions 模式）:**
```yaml
# ❌ 错误：每台 Agent 独立判断触发时间
每台服务器:
  - 读取本地 YAML 文件
  - 检查 on.schedule 字段
  - 使用本地时钟判断是否到达触发时间
  - 独立执行工作流

问题:
  ❌ 100 台服务器时钟不同步 → 触发时间不一致
  ❌ 每台服务器都执行一次 → 重复执行 100 次
  ❌ 无法保证幂等性
  ❌ 失败重试逻辑分散
```

**正确设计（Waterflow 模式）:**
```yaml
# ✅ 正确：Server 端统一触发，分发到 Agent

Server 端:
  - Temporal Schedule 在 Server 端创建
  - UTC 02:00:00 时 Temporal 触发一次
  - 创建一个 Workflow Execution
  - Workflow 编排 100 个 Activity 任务
  - 分发到 Task Queue "linux-amd64"

Temporal Server:
  - 负载均衡到 100 台 Agent
  - 每个 Agent 接收到 1 个 Activity 任务
  - 失败自动重试，重新分配给其他 Agent

结果:
  ✅ 统一触发时间（Temporal Server 时钟）
  ✅ 只创建一个 Workflow Execution
  ✅ 100 个 Activity 并行执行
  ✅ 失败自动重试
```

### 4.2 Schedule 创建流程

```go
// Server 端代码
func (s *Server) OnWorkflowUploaded(wf *dsl.Workflow) error {
    // 1. 检查 on.schedule 字段
    if wf.On != nil && wf.On.Schedule != nil {
        for idx, sched := range wf.On.Schedule {
            // 2. 创建 Temporal Schedule
            scheduleID := fmt.Sprintf("%s-auto-%d", wf.Name, idx)
            
            _, err := s.scheduleClient.Create(ctx, client.ScheduleOptions{
                ID: scheduleID,
                
                Spec: client.ScheduleSpec{
                    CronExpressions: []string{sched.Cron},
                    Timezone:        sched.Timezone,  // ✅ 统一时区
                },
                
                Action: &client.ScheduleWorkflowAction{
                    Workflow:  RunWorkflowExecutor,
                    Args:      []interface{}{wf},  // ✅ 完整的 Workflow 对象
                    TaskQueue: "default",  // Workflow 在 Server 端执行
                    
                    WorkflowExecutionTimeout: 24 * time.Hour,
                },
                
                Policy: &client.SchedulePolicy{
                    Overlap: client.ScheduleOverlapPolicySkip,  // ✅ 防止重复执行
                },
            })
            
            if err != nil {
                return fmt.Errorf("failed to create schedule: %w", err)
            }
            
            log.Info("Schedule created",
                "workflow", wf.Name,
                "schedule_id", scheduleID,
                "cron", sched.Cron,
                "timezone", sched.Timezone,
            )
        }
    }
    
    return nil
}
```

---

## 5. 部分失败处理

### 5.1 场景：100 台服务器中 10 台失败

```yaml
场景描述:
  - 部署任务分发到 100 台服务器
  - 90 台成功
  - 10 台失败（网络超时、磁盘满等）

Waterflow 处理策略:
```

#### 策略 1: 自动重试（默认）

```go
// Workflow 配置
activityOptions := workflow.ActivityOptions{
    TaskQueue:           "linux-amd64",
    StartToCloseTimeout: 10 * time.Minute,
    
    RetryPolicy: &temporal.RetryPolicy{
        InitialInterval:    1 * time.Second,
        BackoffCoefficient: 2.0,
        MaximumInterval:    1 * time.Minute,
        MaximumAttempts:    3,  // ✅ 自动重试 3 次
    },
}

执行流程:
1. Agent 5 执行失败 → 上报 Failed
2. Temporal 1 秒后重试 → 分配给 Agent 5 或其他空闲 Agent
3. 第 2 次失败 → 2 秒后重试
4. 第 3 次失败 → 4 秒后重试
5. 3 次都失败 → 标记为永久失败
```

#### 策略 2: 忽略失败（continue-on-error）

```yaml
# YAML 配置
jobs:
  deploy:
    runs-on: linux-amd64
    steps:
      - name: Deploy
        uses: shell@v1
        with:
          cmd: docker pull nginx
        continue-on-error: true  ◄─── 失败不影响其他 Agent
```

```go
// Workflow 实现
for _, step := range job.Steps {
    err := workflow.ExecuteActivity(activityCtx, "ExecuteStepActivity", step).Get(ctx, &result)
    if err != nil {
        if step.ContinueOnError {
            logger.Warn("Step failed but continuing", "step", step.Name, "error", err)
            continue  // ✅ 忽略失败，继续执行
        }
        return err  // ❌ 失败终止 Workflow
    }
}
```

#### 策略 3: 部分失败报告

```go
// 收集所有 Agent 执行结果
type DeploymentResult struct {
    TotalAgents     int
    SuccessAgents   int
    FailedAgents    int
    FailedHosts     []string
}

func RunWorkflowExecutor(ctx workflow.Context, wf *dsl.Workflow) error {
    result := &DeploymentResult{
        TotalAgents: 100,
    }
    
    for _, step := range job.Steps {
        err := workflow.ExecuteActivity(activityCtx, "ExecuteStepActivity", step).Get(ctx, &stepResult)
        if err != nil {
            result.FailedAgents++
            result.FailedHosts = append(result.FailedHosts, stepResult.Hostname)
        } else {
            result.SuccessAgents++
        }
    }
    
    // 返回详细报告
    logger.Info("Deployment completed",
        "total", result.TotalAgents,
        "success", result.SuccessAgents,
        "failed", result.FailedAgents,
        "failed_hosts", result.FailedHosts,
    )
    
    return nil
}
```

---

## 6. 负载均衡和并发控制

### 6.1 Temporal Task Queue 负载均衡

```yaml
机制:
  - 100 个 Activity 任务进入 Task Queue "linux-amd64"
  - 100 台 Agent 同时轮询此队列
  - 先到先得（先轮询到的 Agent 获取任务）
  - Temporal 保证每个任务只分配一次

配置:
  - Agent 并发度: max_concurrent: 10
  - 每个 Agent 最多同时执行 10 个 Activity
  - 100 台 Agent → 最大并发 1000 个任务
```

```go
// Agent 配置
type AgentConfig struct {
    TaskQueues     []string  // ["linux-amd64", "linux-common"]
    MaxConcurrent  int       // 10 (每个 Agent 最多并发 10 个任务)
}

// Temporal Worker 配置
workerOptions := worker.Options{
    MaxConcurrentActivityExecutionSize: 10,  // ✅ 限制并发度
}
```

### 6.2 并发控制示例

```yaml
# 场景：100 台服务器，每台最多 10 并发，总共 1000 并发

Agent 1:  [Activity 1] [Activity 2] ... [Activity 10]  ◄─── 并发 10
Agent 2:  [Activity 11] [Activity 12] ... [Activity 20] ◄─── 并发 10
...
Agent 100: [Activity 991] ... [Activity 1000]          ◄─── 并发 10

总并发: 1000

如果任务超过 1000:
  - 多余任务在 Task Queue 中等待
  - Agent 完成当前任务后获取新任务
  - 自动平衡负载
```

---

## 7. 架构优势总结

### 7.1 对比 GitHub Actions

| 特性 | GitHub Actions | Waterflow |
|------|---------------|-----------|
| **YAML 存储** | 每个 Runner 下载 | Server 端单一数据源 |
| **Workflow 执行** | Runner 本地执行 | Server 端编排，Agent 执行 Activity |
| **定时触发** | 每个 Runner 独立判断 | Server 统一触发 |
| **100 台服务器** | 100 次独立触发 | 1 次触发，分发到 100 台 |
| **时间同步** | 依赖服务器时钟 | Temporal Server 统一时钟 |
| **失败重试** | Runner 独立重试 | Temporal 集中管理重试 |
| **负载均衡** | 无 | Temporal Task Queue 自动负载均衡 |
| **幂等性** | 难以保证 | Temporal 分布式锁保证 |

### 7.2 核心优势

1. **单一数据源**
   - YAML 只存储在 Server 端
   - Agent 不需要访问 YAML 文件
   - 避免数据冗余和同步问题

2. **统一调度**
   - Schedule 在 Server 端统一触发
   - Temporal 保证只执行一次
   - 避免重复执行

3. **自动负载均衡**
   - Temporal Task Queue 自动分发任务
   - 先到先得，自动平衡负载
   - Agent 失败任务重新分配

4. **容错和重试**
   - Temporal 集中管理重试策略
   - 失败自动重试，无需人工介入
   - 支持部分失败处理

5. **可扩展性**
   - 新增 Agent 无需配置
   - 自动注册到 Task Queue
   - 立即参与任务执行

---

## 8. 常见问题

### Q1: Agent 需要访问 YAML 文件吗？

**答：不需要。** Agent 只接收已解析的 Activity 参数（`*dsl.Step`），所有 YAML 解析在 Server 端完成。

### Q2: 如何保证 100 台服务器时间一致？

**答：** Temporal Server 统一触发时间，Agent 不依赖本地时钟。所有 Schedule 使用 UTC 时区，避免时区问题。

### Q3: 部分 Agent 失败会影响整个 Workflow 吗？

**答：** 取决于配置：
- 默认：失败会重试 3 次，3 次都失败则 Workflow 失败
- `continue-on-error: true`：失败不影响其他 Agent
- 可以收集所有 Agent 结果，生成部分失败报告

### Q4: 如何动态添加新 Agent？

**答：** 
1. 在新服务器上安装 Agent
2. 配置 `task_queues: ["linux-amd64"]`
3. 启动 Agent → 自动注册到 Temporal
4. 立即开始接收任务，无需 Server 配置

### Q5: YAML 变更如何同步到 Schedule？

**答：** 
1. Git Webhook 触发 Server git pull
2. Server 重新解析 YAML
3. 更新或重新创建 Temporal Schedule
4. 下次触发时使用新配置

---

**相关文档：**
- [ADR-0007: Workflow 存储与触发器架构设计](../adr/0007-workflow-storage-and-trigger-architecture.md)
- [Task Queue 路由机制](./task-queue-routing.md)
- [Architecture 概览](../architecture.md)
