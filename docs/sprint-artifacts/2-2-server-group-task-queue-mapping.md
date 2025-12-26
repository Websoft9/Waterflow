# Story 2.2: 服务器组概念和 Task Queue 直接映射

Status: done

## Story

As a **系统架构师**,  
I want **实现服务器组 (Server Group) 的概念和 Task Queue 直接映射机制**,  
so that **工作流可以通过 runs-on 字段直接路由任务到特定服务器组执行,实现零配置路由**。

## Context

这是 **Epic 2: 分布式 Agent 系统**的第二个 Story。Story 2.1 已完成 Agent Worker 基础框架,现在需要实现核心路由机制:如何将工作流任务路由到正确的 Agent 执行。

**前置依赖:**
- Story 2.1 (Agent Worker 基础框架) - Agent 已能连接 Temporal 并轮询 Task Queue
- Story 1.8 (Temporal SDK 集成) - Server 已能提交工作流到 Temporal
- Story 1.3 (YAML DSL 解析) - `runs-on` 字段已解析

**Epic 2 背景:**  
运维工程师可以在多台服务器上部署 Agent,工作流通过 Task Queue 直接映射机制将任务分发到特定服务器组执行。本 Story 实现这一核心路由机制,无需额外配置或调度器。

**业务价值:**
- 🎯 零配置路由 - `runs-on` 值直接映射到 Task Queue,无需维护映射表
- 📡 动态扩展 - 新增服务器组无需修改 Server 代码或配置
- ⚖️ 自动负载均衡 - Temporal 原生负载均衡在同组内分发任务
- 🔄 灵活分组 - 支持 Agent 注册到多个 Queue (如通用 + 专用)

**关键架构决策:**
- [ADR-0006: Task Queue 路由机制](../adr/0006-task-queue-routing.md) - runs-on 直接映射策略
- 服务器组是逻辑概念,无需物理实体
- Temporal Task Queue 即服务器组标识

## Acceptance Criteria

### AC1: runs-on 字段直接映射到 Task Queue

**Given** 工作流 YAML 定义了 `runs-on` 字段  
**When** Server 解析并启动工作流  
**Then** 将 `runs-on` 值直接作为 Temporal Task Queue 名称

**示例 YAML:**
```yaml
name: Multi-Server Deploy

jobs:
  build:
    runs-on: linux-amd64        # Task Queue: "linux-amd64"
    steps:
      - name: Build App
        uses: shell@v1
        with:
          command: make build
  
  deploy-web:
    runs-on: web-servers        # Task Queue: "web-servers"
    needs: [build]
    steps:
      - name: Deploy to Web
        uses: deploy@v1
  
  deploy-db:
    runs-on: db-servers         # Task Queue: "db-servers"
    needs: [build]
    steps:
      - name: Deploy Database
        uses: deploy@v1
```

**Server 实现** (扩展 `pkg/temporal/client.go`):
```go
// SubmitWorkflow submits a workflow to Temporal with proper task queue routing.
func (c *Client) SubmitWorkflow(ctx context.Context, workflow *dsl.Workflow) (*WorkflowRun, error) {
	workflowID := uuid.New().String()
	
	// Start the main workflow orchestrator
	workflowOptions := client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                c.config.TaskQueue, // Server's task queue
		WorkflowExecutionTimeout: 24 * time.Hour,
	}
	
	run, err := c.client.ExecuteWorkflow(ctx, workflowOptions, "RunWorkflowExecutor", workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to start workflow: %w", err)
	}
	
	c.logger.Info("Workflow submitted",
		zap.String("workflow_id", workflowID),
		zap.String("workflow_name", workflow.Name),
		zap.String("run_id", run.GetRunID()),
	)
	
	return &WorkflowRun{
		ID:    workflowID,
		RunID: run.GetRunID(),
	}, nil
}
```

**Workflow 执行器** (扩展 `pkg/temporal/workflow.go`):
```go
// RunWorkflowExecutor orchestrates the entire workflow execution.
func RunWorkflowExecutor(ctx workflow.Context, wf *dsl.Workflow) error {
	logger := workflow.GetLogger(ctx)
	
	// Build job dependency graph
	graph, err := buildJobGraph(wf.Jobs)
	if err != nil {
		return fmt.Errorf("failed to build job graph: %w", err)
	}
	
	// Execute jobs based on dependency order
	for _, level := range graph.TopologicalOrder() {
		var futures []workflow.Future
		
		for _, jobID := range level {
			job := wf.Jobs[jobID]
			
			// CRITICAL: Use job's runs-on as Task Queue
			// This routes the job to agents polling that queue
			childWorkflowOptions := workflow.ChildWorkflowOptions{
				WorkflowID:            fmt.Sprintf("%s-job-%s", workflow.GetInfo(ctx).WorkflowExecution.ID, jobID),
				TaskQueue:             job.RunsOn, // Direct mapping!
				WorkflowExecutionTimeout: time.Duration(job.TimeoutMinutes) * time.Minute,
			}
			childCtx := workflow.WithChildOptions(ctx, childWorkflowOptions)
			
			// Start child workflow for job execution
			future := workflow.ExecuteChildWorkflow(childCtx, RunJobWorkflow, job)
			futures = append(futures, future)
			
			logger.Info("Job started",
				"job_id", jobID,
				"runs_on", job.RunsOn,
				"task_queue", job.RunsOn, // Same value
			)
		}
		
		// Wait for all jobs in this level to complete
		for i, future := range futures {
			if err := future.Get(ctx, nil); err != nil {
				return fmt.Errorf("job %d failed: %w", i, err)
			}
		}
	}
	
	return nil
}
```

**验证:**
- YAML 中 `runs-on: linux-amd64` → Temporal Task Queue: `linux-amd64`
- YAML 中 `runs-on: web-servers` → Temporal Task Queue: `web-servers`
- 无需配置文件维护 Queue 映射表
- Server 日志记录每个 Job 的路由信息

### AC2: Agent 注册到多个 Task Queue

**Given** Agent 配置了多个 Task Queue  
**When** Agent 启动时  
**Then** 为每个 Queue 创建独立的 Worker 并开始轮询

**Agent 配置** (已在 Story 2.1 实现):
```yaml
# config.agent.example.yaml
agent:
  task_queues:
    - linux-amd64      # 主要队列:Linux AMD64 任务
    - linux-common     # 通用队列:所有 Linux 任务
    - gpu-a100         # 专用队列:GPU 任务(如果有 GPU)
```

**Agent Worker 实现** (已在 Story 2.1 实现 `internal/agent/worker.go`):
```go
// Start starts workers for all configured task queues.
func (w *Worker) Start() error {
	// Create and start a worker for each task queue
	for _, taskQueue := range w.config.Agent.TaskQueues {
		workerInstance := worker.New(w.temporalClient.GetClient(), taskQueue, worker.Options{
			MaxConcurrentActivityExecutionSize:     100,
			MaxConcurrentWorkflowTaskExecutionSize: 50,
		})

		// Register workflows and activities
		workerInstance.RegisterWorkflow(temporal.RunJobWorkflow)
		activities := &temporal.Activities{
			PluginManager: w.pluginManager,
			Logger:        w.logger,
		}
		workerInstance.RegisterActivity(activities.ExecuteStepActivity)

		w.workers = append(w.workers, workerInstance)

		w.logger.Info("Registered worker for task queue",
			zap.String("task_queue", taskQueue),
		)

		// Start worker in background
		go func(queue string, wk worker.Worker) {
			w.logger.Info("Starting worker polling", zap.String("task_queue", queue))
			if err := wk.Run(worker.InterruptCh()); err != nil {
				w.logger.Error("Worker stopped with error",
					zap.String("task_queue", queue),
					zap.Error(err),
				)
			}
		}(taskQueue, workerInstance)
	}

	return nil
}
```

**验证:**
```bash
# 启动 Agent
bin/agent --task-queues linux-amd64,linux-common,gpu-a100

# 日志输出:
# {"level":"info","message":"Registered worker for task queue","task_queue":"linux-amd64"}
# {"level":"info","message":"Registered worker for task queue","task_queue":"linux-common"}
# {"level":"info","message":"Registered worker for task queue","task_queue":"gpu-a100"}
# {"level":"info","message":"Starting worker polling","task_queue":"linux-amd64"}
# {"level":"info","message":"Starting worker polling","task_queue":"linux-common"}
# {"level":"info","message":"Starting worker polling","task_queue":"gpu-a100"}
```

**行为:**
- Agent 同时轮询 3 个 Task Queue
- 任何一个 Queue 有任务到达,Agent 都会执行
- 不同 Queue 的任务可以并发执行 (取决于 `MaxConcurrentActivityExecutionSize`)

### AC3: Temporal 原生负载均衡验证

**Given** 多个 Agent 注册到同一个 Task Queue  
**When** 多个任务提交到该 Queue  
**Then** Temporal 自动在 Agent 之间分发任务 (轮询)

**测试场景:**
```bash
# 启动 3 个 Agent,都注册到 "linux-amd64" Queue
# Terminal 1
bin/agent --task-queues linux-amd64

# Terminal 2
bin/agent --task-queues linux-amd64

# Terminal 3
bin/agent --task-queues linux-amd64
```

**提交工作流** (10 个 Jobs,都 runs-on: linux-amd64):
```yaml
name: Load Balancing Test

jobs:
  job-1:
    runs-on: linux-amd64
    steps:
      - uses: shell@v1
        with:
          command: echo "Job 1"
  
  job-2:
    runs-on: linux-amd64
    steps:
      - uses: shell@v1
        with:
          command: echo "Job 2"
  
  # ... job-3 to job-10
```

**预期行为:**
- 10 个 Jobs 分配到 3 个 Agent 执行
- 分配大致均衡 (Agent1: 3-4 个, Agent2: 3-4 个, Agent3: 3-4 个)
- Temporal 使用轮询策略分发任务
- 繁忙的 Agent 不会接收新任务直到空闲

**验证方式:**
1. 检查 Temporal UI - 显示 3 个 Worker 在 `linux-amd64` Queue
2. 查看 Agent 日志 - 每个 Agent 执行不同的 Jobs
3. Temporal Metrics - `temporal_worker_task_queue_poll_succeed` 指标

**日志示例** (Agent 1):
```json
{"level":"info","message":"Executing job","job_id":"job-1","task_queue":"linux-amd64"}
{"level":"info","message":"Executing job","job_id":"job-4","task_queue":"linux-amd64"}
{"level":"info","message":"Executing job","job_id":"job-7","task_queue":"linux-amd64"}
```

**日志示例** (Agent 2):
```json
{"level":"info","message":"Executing job","job_id":"job-2","task_queue":"linux-amd64"}
{"level":"info","message":"Executing job","job_id":"job-5","task_queue":"linux-amd64"}
{"level":"info","message":"Executing job","job_id":"job-9","task_queue":"linux-amd64"}
```

### AC4: 服务器组命名规范和验证

**Given** 用户定义 `runs-on` 字段  
**When** Server 验证 YAML  
**Then** 验证 Task Queue 名称符合 Temporal 要求

**命名规则** (ADR-0006):
- 只能包含字母、数字和连字符 (`-`)
- 不能包含下划线 (`_`)、空格或特殊字符
- 长度 < 256 字符
- 必须以字母或数字开头和结尾

**实现** (扩展 `pkg/dsl/validator.go`):
```go
// ValidateWorkflow validates the entire workflow structure.
func (v *Validator) ValidateWorkflow(workflow *Workflow) error {
	var errors []ValidationError
	
	// Validate jobs
	for jobID, job := range workflow.Jobs {
		// Validate runs-on field
		if job.RunsOn == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("jobs.%s.runs-on", jobID),
				Message: "runs-on is required",
			})
		} else {
			if err := validateTaskQueueName(job.RunsOn); err != nil {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("jobs.%s.runs-on", jobID),
					Message: err.Error(),
					Suggestion: "Use only alphanumeric characters and hyphens (e.g., 'linux-amd64', 'web-servers')",
				})
			}
		}
		
		// ... other validations
	}
	
	if len(errors) > 0 {
		return &WorkflowValidationError{Errors: errors}
	}
	return nil
}

// validateTaskQueueName validates Task Queue naming per ADR-0006.
func validateTaskQueueName(name string) error {
	// Temporal requirement: alphanumeric and hyphens, length < 256
	re := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)
	if !re.MatchString(name) {
		return fmt.Errorf("invalid task queue name: must contain only alphanumeric and hyphens")
	}
	if len(name) > 255 {
		return fmt.Errorf("task queue name too long (max 255 characters)")
	}
	return nil
}
```

**验证测试:**
```go
func TestValidateTaskQueueName(t *testing.T) {
	tests := []struct {
		name    string
		queue   string
		wantErr bool
	}{
		{"valid: alphanumeric", "linux-amd64", false},
		{"valid: with hyphens", "web-servers-prod", false},
		{"valid: numbers", "gpu-a100", false},
		{"invalid: underscore", "linux_amd64", true},
		{"invalid: space", "web servers", true},
		{"invalid: special char", "linux@amd64", true},
		{"invalid: starts with hyphen", "-linux", true},
		{"invalid: ends with hyphen", "linux-", true},
		{"invalid: too long", strings.Repeat("a", 256), true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTaskQueueName(tt.queue)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
```

**用户体验:**
```bash
# 提交工作流
POST /v1/workflows

# 错误响应 (422 Unprocessable Entity)
{
  "error": {
    "code": "validation_error",
    "message": "YAML validation failed",
    "details": {
      "errors": [
        {
          "field": "jobs.build.runs-on",
          "line": 8,
          "message": "invalid task queue name: must contain only alphanumeric and hyphens",
          "current_value": "linux_amd64",
          "suggestion": "Use only alphanumeric characters and hyphens (e.g., 'linux-amd64', 'web-servers')"
        }
      ]
    }
  }
}
```

### AC5: 服务器组推荐命名约定

**Given** 用户需要定义 `runs-on` 值  
**When** 查阅文档  
**Then** 提供清晰的命名约定指南

**推荐命名模式** (文档):

| 分类 | 命名模式 | 示例 |
|------|----------|------|
| **操作系统 + 架构** | `{os}-{arch}` | `linux-amd64`, `linux-arm64`, `macos-arm64`, `windows-x64` |
| **硬件特性** | `{feature}-{model}` | `gpu-a100`, `gpu-v100`, `high-memory`, `nvme-storage` |
| **环境/用途** | `{env}` 或 `{purpose}` | `production`, `staging`, `build-servers`, `web-servers`, `db-servers` |
| **地理位置** | `{region}-{zone}` | `us-west-1`, `eu-central-1`, `asia-east-1` |
| **自定义** | `{custom-name}` | `my-custom-group`, `special-hardware` |

**组合命名:**
```yaml
# 组合多个维度
runs-on: linux-amd64-gpu-a100-us-west    # OS + Arch + Hardware + Region

# 简洁优先
runs-on: gpu-servers                     # 简单清晰

# 环境隔离
runs-on: prod-web-servers                # 生产环境 Web 服务器
runs-on: staging-web-servers             # 测试环境 Web 服务器
```

**反例 (不推荐):**
```yaml
# ❌ 包含下划线
runs-on: linux_amd64

# ❌ 包含特殊字符
runs-on: linux@amd64

# ❌ 包含空格
runs-on: linux amd64

# ❌ 过长且复杂
runs-on: linux-ubuntu-22-04-amd64-with-docker-and-gpu-nvidia-a100-in-us-west-1-zone-a
```

**最佳实践:**
1. 保持简洁 (通常 2-4 个单词)
2. 使用连字符分隔
3. 小写字母 (虽然支持大写,但小写更规范)
4. 见名知意 (他人能理解服务器组用途)
5. 一致性 (项目内统一命名风格)

**文档位置:**
- `docs/guides/server-groups.md` - 服务器组命名指南
- `README.md` - 快速开始中包含示例
- API 错误提示 - 验证失败时提供建议

### AC6: 不存在 Queue 的错误处理

**Given** 工作流指定了 `runs-on: special-hardware`  
**When** 没有 Agent 注册到 `special-hardware` Queue  
**Then** Job 等待直到超时或 Agent 上线

**场景 1: Job 级超时**
```yaml
jobs:
  build:
    runs-on: non-existent-queue
    timeout-minutes: 10          # 10 分钟超时
    steps:
      - uses: shell@v1
```

**行为:**
- Job 进入等待状态 (Temporal Workflow 等待 Activity)
- 10 分钟后超时,Job 标记为 `timeout`
- 工作流失败,错误信息: "Job 'build' timed out waiting for worker"

**Temporal UI 显示:**
```
Workflow Status: Failed
└─ Job: build
   Status: Timeout
   Task Queue: non-existent-queue
   Scheduled Time: 2025-12-25 10:00:00
   Timeout Time: 2025-12-25 10:10:00
   Error: Activity timeout (no worker available)
```

**场景 2: Agent 延迟上线**
```yaml
jobs:
  deploy:
    runs-on: special-hardware
    timeout-minutes: 60
```

**时间线:**
- 10:00 - Job 提交,进入 `special-hardware` Queue
- 10:05 - 仍在等待 (无 Agent)
- 10:10 - Agent 启动并注册到 `special-hardware` Queue
- 10:10 - Job 立即分发到 Agent 并开始执行
- 10:15 - Job 完成

**日志:**
```json
// 10:00 - Server
{"level":"info","message":"Job submitted","job_id":"deploy","task_queue":"special-hardware"}

// 10:05 - Server (Temporal 内部等待,无日志)

// 10:10 - Agent
{"level":"info","message":"Worker started","task_queue":"special-hardware"}
{"level":"info","message":"Received job","job_id":"deploy","task_queue":"special-hardware"}

// 10:15 - Agent
{"level":"info","message":"Job completed","job_id":"deploy","duration_seconds":300}
```

**最佳实践:**
- 提前启动 Agent (在提交工作流前)
- 使用合理的超时时间 (考虑 Agent 启动时间)
- 监控 Task Queue 状态 (通过 Temporal UI)

### AC7: 服务器组状态查询 API (可选,为 Story 2.7 准备)

**Given** 用户想知道哪些 Task Queue 有可用 Agent  
**When** 调用状态查询 API  
**Then** 返回所有活跃的 Task Queue 和 Worker 数量

**API 端点** (本 Story 实现基础,Story 2.7 完善):
```
GET /v1/task-queues
```

**响应示例:**
```json
{
  "task_queues": [
    {
      "name": "linux-amd64",
      "worker_count": 3,
      "last_heartbeat": "2025-12-25T10:30:00Z",
      "status": "healthy"
    },
    {
      "name": "linux-common",
      "worker_count": 5,
      "last_heartbeat": "2025-12-25T10:30:00Z",
      "status": "healthy"
    },
    {
      "name": "gpu-a100",
      "worker_count": 1,
      "last_heartbeat": "2025-12-25T10:29:45Z",
      "status": "healthy"
    }
  ]
}
```

**实现** (基础版本,`internal/api/task_queue_handler.go`):
```go
// ListTaskQueues returns a list of active task queues.
// This is a basic implementation that queries Temporal for worker status.
func (h *Handler) ListTaskQueues(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Query Temporal for task queue statistics
	// Note: Temporal SDK doesn't provide a direct API for this in MVP
	// This is a placeholder for Story 2.7 (Agent Health Monitoring)
	
	// For now, return a simple response based on known agents
	// Full implementation in Story 2.7 will query Temporal Admin API
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Task queue listing not yet implemented (Story 2.7)",
		"hint":    "Use Temporal UI to view active task queues",
	})
}
```

**路由注册** (`internal/api/router.go`):
```go
func SetupRouter(handler *Handler) *gin.Engine {
	router := gin.Default()
	
	v1 := router.Group("/v1")
	{
		// ... existing routes
		
		// Task queue management (Story 2.2 placeholder, Story 2.7 full implementation)
		v1.GET("/task-queues", handler.ListTaskQueues)
	}
	
	return router
}
```

**Story 2.7 完善:**
- 调用 Temporal Admin API 获取 Worker 心跳
- 计算每个 Queue 的 Worker 数量
- 返回详细的健康状态

## Developer Context

### 架构概述

Task Queue 直接映射是 Waterflow 分布式系统的核心路由机制,实现了零配置的任务分发:

```
┌─────────────────────────────────────────────────────────────┐
│                     User Workflow YAML                      │
│                                                             │
│  jobs:                                                      │
│    build:                                                   │
│      runs-on: linux-amd64  ←────────┐                      │
│                                      │                      │
│    deploy-web:                       │ Direct Mapping      │
│      runs-on: web-servers  ←─────────┤ (No Config Needed)  │
│                                      │                      │
│    deploy-db:                        │                      │
│      runs-on: db-servers  ←──────────┘                      │
└─────────────────────────────────────────────────────────────┘
                      │
                      ↓ Server parses YAML
┌─────────────────────────────────────────────────────────────┐
│                   Waterflow Server                          │
│                                                             │
│  RunWorkflowExecutor(workflow):                             │
│    for job in workflow.jobs:                                │
│      taskQueue = job.runs_on  ←── Direct Assignment        │
│      ExecuteChildWorkflow(job, taskQueue)                   │
└─────────────────────────────────────────────────────────────┘
                      │
                      ↓ Submit to Temporal
┌─────────────────────────────────────────────────────────────┐
│                  Temporal Server                            │
│                                                             │
│  ┌─────────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │ Queue:          │  │ Queue:       │  │ Queue:       │  │
│  │ linux-amd64     │  │ web-servers  │  │ db-servers   │  │
│  │                 │  │              │  │              │  │
│  │ - build job     │  │ - deploy job │  │ - deploy job │  │
│  └─────────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────────┘
         │                     │                   │
         ↓ Poll               ↓ Poll              ↓ Poll
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│ Agent A       │    │ Agent B       │    │ Agent C       │
│ (Build Server)│    │ (Web Server)  │    │ (DB Server)   │
│               │    │               │    │               │
│ Queues:       │    │ Queues:       │    │ Queues:       │
│ - linux-amd64 │    │ - web-servers │    │ - db-servers  │
└───────────────┘    └───────────────┘    └───────────────┘
```

### 关键技术实现

#### 1. Workflow 编排器中的 Queue 路由

**文件:** `pkg/temporal/workflow.go`

```go
// RunWorkflowExecutor is the main workflow orchestrator.
// It executes jobs based on their dependency graph and routes each job
// to the appropriate task queue using runs-on field.
func RunWorkflowExecutor(ctx workflow.Context, wf *dsl.Workflow) error {
	logger := workflow.GetLogger(ctx)
	workflowInfo := workflow.GetInfo(ctx)
	
	logger.Info("Starting workflow execution",
		"workflow_id", workflowInfo.WorkflowExecution.ID,
		"workflow_name", wf.Name,
		"job_count", len(wf.Jobs),
	)
	
	// Build dependency graph
	graph, err := buildJobGraph(wf.Jobs)
	if err != nil {
		return fmt.Errorf("failed to build job graph: %w", err)
	}
	
	// Execute jobs level by level (topological order)
	for levelIndex, level := range graph.TopologicalOrder() {
		logger.Info("Executing job level",
			"level", levelIndex,
			"job_count", len(level),
		)
		
		var futures []workflow.Future
		
		for _, jobID := range level {
			job := wf.Jobs[jobID]
			
			// CRITICAL: Direct mapping of runs-on to Task Queue
			taskQueue := job.RunsOn
			
			// Validate task queue name (defensive check)
			if taskQueue == "" {
				return fmt.Errorf("job %s has empty runs-on field", jobID)
			}
			
			// Create child workflow options with specific task queue
			childWorkflowOptions := workflow.ChildWorkflowOptions{
				WorkflowID:               fmt.Sprintf("%s-job-%s", workflowInfo.WorkflowExecution.ID, jobID),
				TaskQueue:                taskQueue, // Routes to specific agent group
				WorkflowExecutionTimeout: time.Duration(job.TimeoutMinutes) * time.Minute,
			}
			childCtx := workflow.WithChildOptions(ctx, childWorkflowOptions)
			
			// Start job execution on target task queue
			future := workflow.ExecuteChildWorkflow(childCtx, RunJobWorkflow, job)
			futures = append(futures, future)
			
			logger.Info("Job submitted to task queue",
				"job_id", jobID,
				"task_queue", taskQueue,
				"timeout_minutes", job.TimeoutMinutes,
			)
		}
		
		// Wait for all jobs in this level to complete
		for i, future := range futures {
			jobID := level[i]
			if err := future.Get(ctx, nil); err != nil {
				logger.Error("Job failed",
					"job_id", jobID,
					"error", err,
				)
				return fmt.Errorf("job %s failed: %w", jobID, err)
			}
			logger.Info("Job completed", "job_id", jobID)
		}
	}
	
	logger.Info("Workflow execution completed successfully",
		"workflow_id", workflowInfo.WorkflowExecution.ID,
	)
	
	return nil
}

// buildJobGraph constructs a dependency graph from jobs.
func buildJobGraph(jobs map[string]*dsl.Job) (*JobGraph, error) {
	graph := &JobGraph{
		nodes: make(map[string]*JobNode),
	}
	
	// Create nodes
	for jobID, job := range jobs {
		graph.nodes[jobID] = &JobNode{
			ID:           jobID,
			Job:          job,
			Dependencies: job.Needs,
		}
	}
	
	// Validate dependencies and detect cycles
	if err := graph.Validate(); err != nil {
		return nil, err
	}
	
	return graph, nil
}
```

#### 2. Agent 多 Queue 轮询实现

**文件:** `internal/agent/worker.go` (已在 Story 2.1 实现)

```go
// Start creates and starts workers for all configured task queues.
func (w *Worker) Start() error {
	// Load plugins (Epic 4)
	if err := w.pluginManager.LoadPlugins(); err != nil {
		w.logger.Warn("Failed to load plugins", zap.Error(err))
	}
	
	// Create a worker for each task queue
	for _, taskQueue := range w.config.Agent.TaskQueues {
		// Create worker instance
		workerInstance := worker.New(
			w.temporalClient.GetClient(),
			taskQueue, // Each worker polls a specific queue
			worker.Options{
				MaxConcurrentActivityExecutionSize:     100,
				MaxConcurrentWorkflowTaskExecutionSize: 50,
			},
		)
		
		// Register workflows (Job executor)
		workerInstance.RegisterWorkflow(temporal.RunJobWorkflow)
		
		// Register activities (Step executor)
		activities := &temporal.Activities{
			PluginManager: w.pluginManager,
			Logger:        w.logger,
		}
		workerInstance.RegisterActivity(activities.ExecuteStepActivity)
		
		w.workers = append(w.workers, workerInstance)
		
		w.logger.Info("Worker registered for task queue",
			zap.String("task_queue", taskQueue),
		)
		
		// Start worker in background goroutine
		go func(queue string, wk worker.Worker) {
			w.logger.Info("Worker polling started",
				zap.String("task_queue", queue),
			)
			
			if err := wk.Run(worker.InterruptCh()); err != nil {
				w.logger.Error("Worker stopped with error",
					zap.String("task_queue", queue),
					zap.Error(err),
				)
			}
		}(taskQueue, workerInstance)
	}
	
	w.logger.Info("All workers started",
		zap.Int("worker_count", len(w.workers)),
		zap.Strings("task_queues", w.config.Agent.TaskQueues),
	)
	
	return nil
}
```

#### 3. 命名验证器实现

**文件:** `pkg/dsl/validator.go`

```go
// validateTaskQueueName validates Task Queue naming per ADR-0006.
// Rules:
// - Only alphanumeric characters and hyphens
// - Must start and end with alphanumeric character
// - Length < 256 characters
func validateTaskQueueName(name string) error {
	if name == "" {
		return fmt.Errorf("task queue name cannot be empty")
	}
	
	// Regex: alphanumeric start, alphanumeric/hyphen middle, alphanumeric end
	re := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)
	if !re.MatchString(name) {
		return fmt.Errorf("invalid task queue name: must contain only alphanumeric characters and hyphens, and must start/end with alphanumeric")
	}
	
	if len(name) > 255 {
		return fmt.Errorf("task queue name too long: maximum 255 characters, got %d", len(name))
	}
	
	return nil
}
```

### Temporal 负载均衡机制

Temporal 使用 **轮询 (Round-Robin)** 策略在同一 Task Queue 的多个 Worker 之间分发任务:

1. **Worker 注册:**
   - Worker 启动时向 Temporal 注册并开始轮询特定 Task Queue
   - Temporal 维护每个 Queue 的 Worker 列表

2. **任务分发:**
   - 新任务到达 Queue 时,Temporal 选择下一个空闲 Worker
   - 使用轮询算法确保均衡分发
   - 繁忙的 Worker 停止轮询,直到当前任务完成

3. **心跳机制:**
   - Worker 每 30 秒发送心跳
   - 连续 3 次心跳失败 (90 秒) → Worker 标记为 unhealthy
   - Unhealthy Worker 不接收新任务

**负载均衡示例:**

```
Task Queue: linux-amd64
├─ Worker A (Server 1) - Idle
├─ Worker B (Server 2) - Idle
└─ Worker C (Server 3) - Idle

Job 1 arrives → Assigned to Worker A
Job 2 arrives → Assigned to Worker B
Job 3 arrives → Assigned to Worker C
Job 4 arrives → Assigned to Worker A (round-robin)

Task Queue: linux-amd64
├─ Worker A (Server 1) - Busy (Job 1, Job 4)
├─ Worker B (Server 2) - Busy (Job 2)
└─ Worker C (Server 3) - Busy (Job 3)
```

### 与其他 Story 的关系

**前置依赖:**
- ✅ Story 1.3 - `runs-on` 字段已解析
- ✅ Story 1.8 - Temporal Workflow 编排器已实现
- ✅ Story 2.1 - Agent Worker 已能轮询 Task Queue

**本 Story 完成后:**
- Story 2.3 可以实现 ServerGroupProvider (查询 Agent 清单)
- Story 2.4 心跳机制已由 Temporal 提供
- Story 2.5 任务分发已自动完成
- Story 2.7 可以实现健康监控 API

### 测试策略

#### 单元测试

```go
// pkg/dsl/validator_test.go
func TestValidateTaskQueueName(t *testing.T) {
	tests := []struct {
		name    string
		queue   string
		wantErr bool
		errMsg  string
	}{
		{"valid: simple", "linux", false, ""},
		{"valid: with hyphen", "linux-amd64", false, ""},
		{"valid: numbers", "gpu-a100", false, ""},
		{"invalid: underscore", "linux_amd64", true, "must contain only alphanumeric"},
		{"invalid: space", "web servers", true, "must contain only alphanumeric"},
		{"invalid: special char", "linux@amd64", true, "must contain only alphanumeric"},
		{"invalid: starts with hyphen", "-linux", true, "must start/end with alphanumeric"},
		{"invalid: ends with hyphen", "linux-", true, "must start/end with alphanumeric"},
		{"invalid: empty", "", true, "cannot be empty"},
		{"invalid: too long", strings.Repeat("a", 256), true, "too long"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTaskQueueName(tt.queue)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
```

#### 集成测试

```bash
# 1. 启动 Temporal
cd deployments
docker-compose up -d temporal

# 2. 启动 Server
bin/server --config config.yaml

# 3. 启动多个 Agent (不同 Queue)
bin/agent --task-queues linux-amd64 &
bin/agent --task-queues web-servers &
bin/agent --task-queues db-servers &

# 4. 提交测试工作流
curl -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/json" \
  -d @examples/multi-server.yaml

# 5. 验证
# - 检查 Temporal UI: http://localhost:8088
# - 查看每个 Queue 的 Worker 数量
# - 验证 Jobs 路由到正确的 Agent
```

#### 负载均衡测试

```bash
# 启动 3 个 Agent 到同一 Queue
bin/agent --task-queues linux-amd64 &
bin/agent --task-queues linux-amd64 &
bin/agent --task-queues linux-amd64 &

# 提交 10 个 Jobs 的工作流
# 预期: 每个 Agent 执行 3-4 个 Jobs

# 验证方式:
# 1. 查看 Agent 日志 - 每个 Agent 应该执行不同的 Jobs
# 2. Temporal UI - 显示 3 个 Worker 在 linux-amd64 Queue
# 3. Metrics - temporal_worker_task_queue_poll_succeed 指标
```

### 常见问题

#### Q1: 如何知道哪些 Task Queue 可用?

**方法 1: Temporal UI**
- 访问 http://localhost:8088
- 导航到 "Workers" 页面
- 查看所有活跃的 Task Queue 和 Worker 数量

**方法 2: API 查询** (Story 2.7 实现)
```bash
curl http://localhost:8080/v1/task-queues
```

**方法 3: 约定** (推荐)
- 团队约定标准命名 (如 `linux-amd64`, `web-servers`)
- 在文档中列出所有服务器组
- Agent 启动时记录注册的 Queue

#### Q2: Agent 可以同时属于多个服务器组吗?

**可以!** Agent 配置多个 Task Queue 即可:

```yaml
agent:
  task_queues:
    - linux-amd64      # 特定架构
    - linux-common     # 通用 Linux
    - build-servers    # 构建服务器组
```

这样 Agent 会接收以上 3 个 Queue 的任务。

#### Q3: 如果 Queue 名称拼写错误怎么办?

**现象:**
```yaml
runs-on: liunx-amd64  # 拼写错误
```

**结果:**
- Job 提交到 `liunx-amd64` Queue
- 没有 Agent 轮询该 Queue
- Job 等待直到超时

**预防:**
- 使用代码补全/模板生成 YAML
- 文档中列出标准命名
- CI/CD 中验证 Queue 名称

#### Q4: 如何动态添加新服务器组?

**步骤:**
1. 在新服务器上部署 Agent
2. 配置新的 Task Queue 名称
3. 启动 Agent
4. 在工作流中使用新的 `runs-on` 值

**示例:**
```bash
# 新服务器上
bin/agent --task-queues new-hardware-group
```

```yaml
# 工作流中
jobs:
  special-task:
    runs-on: new-hardware-group  # 立即可用
```

无需修改 Server 配置或重启!

### 下一步 (Story 2.3)

**Story 2.3: ServerGroupProvider 接口实现**

本 Story 实现了核心路由机制,Story 2.3 将增强服务器组管理:

- 定义 ServerGroupProvider 接口
- 提供内存实现 (简单)
- 提供配置文件实现 (YAML/JSON)
- 为 CMDB 集成预留接口

## Dev Notes

### 实现清单

**必须实现:**
- ✅ `pkg/temporal/workflow.go` - 扩展 `RunWorkflowExecutor` 使用 `runs-on` 作为 Task Queue
- ✅ `pkg/dsl/validator.go` - 添加 `validateTaskQueueName` 函数
- ✅ `pkg/dsl/validator_test.go` - 命名验证测试
- ✅ `internal/api/task_queue_handler.go` - 占位 API (Story 2.7 完善)
- ✅ `internal/api/router.go` - 注册 `/v1/task-queues` 路由
- ✅ `docs/guides/server-groups.md` - 命名约定指南

**已在 Story 2.1 实现 (无需修改):**
- Agent Worker 多 Queue 轮询
- Agent 配置支持多 Task Queue
- Temporal Worker 心跳机制

### 代码规范

- 所有新函数添加 GoDoc 注释
- 错误消息清晰且可操作
- 日志使用结构化字段
- 测试覆盖率 >80%

### 文档更新

**必须更新:**
- `docs/guides/server-groups.md` - 新建,命名约定指南
- `README.md` - 快速开始示例中使用多服务器组
- `examples/multi-server.yaml` - 多服务器部署示例

**可选更新:**
- `docs/architecture.md` - 补充 Task Queue 路由图
- `docs/quick-start.md` - 包含 Agent 部署示例

## Dev Agent Record

### Context Reference

完整的技术上下文已在 Developer Context 部分提供。

### Agent Model Used

Claude Sonnet 4.5

### Debug Log References

无调试问题。

### Completion Notes List

✅ **AC1-AC7 全部完成:**

1. **AC1: runs-on → Task Queue 直接映射** - `pkg/temporal/workflow.go` 已使用 `job.RunsOn` 作为 Task Queue
2. **AC2: Agent 多 Queue 注册** - Story 2.1 已实现,无需修改
3. **AC3: Temporal 负载均衡** - Temporal 原生支持,文档已说明
4. **AC4: Task Queue 命名验证** - `pkg/dsl/semantic_validator.go` 添加 `ValidateTaskQueueName` 函数
5. **AC5: 命名约定指南** - 创建 `docs/guides/server-groups.md`
6. **AC6: Queue 不存在处理** - Temporal 自动处理,文档已说明超时行为
7. **AC7: Task Queue API 占位** - `internal/api/workflow_handler.go` 添加 `ListTaskQueues` 方法

### File List

**新增文件:**
- `docs/guides/server-groups.md` - 服务器组命名指南 (448 行)
- `examples/multi-server.yaml` - 多服务器部署示例 (117 行)
- `pkg/dsl/task_queue_validator_test.go` - Task Queue 验证测试 (164 行)
- `test/integration/task_queue_routing_test.go` - 集成测试 (310 行,代码审查新增)

**修改文件:**
- `pkg/dsl/semantic_validator.go` - 添加 ValidateTaskQueueName + validateRunsOn (~90 行新增)
- `internal/api/workflow_handler.go` - 添加 ListTaskQueues 方法 + 强制验证 (~35 行新增/修改)
- `internal/api/router.go` - 注册 Task Queue 路由 (~2 行新增)
- `pkg/temporal/workflow.go` - 添加防御性 runs-on 检查 (~12 行新增)
- `README.md` - 更新多服务器示例 (~15 行修改)
- `docs/sprint-artifacts/sprint-status.yaml` - 状态更新
- `docs/sprint-artifacts/2-2-server-group-task-queue-mapping.md` - 本文件

**总计:** ~1020 新增代码行 (含测试), ~32 修改行

**测试结果:**
- ✅ TestValidateTaskQueueName - 19个测试全部通过
- ✅ TestSemanticValidator_ValidateRunsOn - 5个测试全部通过
- ✅ 完整测试套件 - 无回归问题

**技术亮点:**
1. 零配置路由 - runs-on 直接映射 Task Queue,无需维护映射表
2. 完善的验证 - 正则表达式验证,清晰的错误提示
3. 详细的文档 - 命名指南包含最佳实践和实际示例
4. 向后兼容 - Story 2.1 已实现的多 Queue 轮询无需修改

**已达成:**
- AC1-AC7 全部验收标准 ✅
- 代码覆盖率 >80% ✅
- 文档完整 ✅
- 测试全部通过 ✅

**代码审查修复 (2025-12-25):**
- ✅ HIGH-1: 添加集成测试验证 AC2 多 Queue 注册
- ✅ HIGH-2: 添加负载均衡集成测试验证 AC3
- ✅ HIGH-3: 强制执行 YAML 验证,拒绝无效 runs-on
- ✅ MEDIUM-1: ListTaskQueues 保持占位符(Story 2.7 实现)
- ✅ MEDIUM-2: 修正文档示例与测试用例一致性
- ✅ MEDIUM-3: 添加 AC6 超时和延迟上线测试
- ✅ MEDIUM-4: workflow.go 添加 runs-on 防御性检查
- ✅ LOW-1: 文档保持现状(448行可接受)
- ✅ LOW-2: 重命名测试用例避免误导
