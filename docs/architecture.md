---
stepsCompleted: [1, 2, 3, 4, 5, 6, 7, 8]
inputDocuments:
  - /data/Waterflow/docs/prd.md
  - /data/Waterflow/docs/adr/0001-use-temporal-workflow-engine.md
  - /data/Waterflow/docs/adr/0002-single-node-execution-pattern.md
  - /data/Waterflow/docs/adr/0003-plugin-based-node-system.md
  - /data/Waterflow/docs/adr/0004-yaml-dsl-syntax.md
  - /data/Waterflow/docs/adr/0005-expression-system-syntax.md
  - /data/Waterflow/docs/adr/0006-task-queue-routing.md
workflowType: 'architecture'
lastStep: 8
project_name: 'Waterflow'
user_name: 'Websoft9'
date: '2025-12-16'
status: 'complete'
version: '1.0'
---

# Waterflow 架构设计文档

**版本:** 1.0  
**日期:** 2025-12-16  
**状态:** Architecture Design Complete

---

## 文档说明

本文档采用 [C4 Model](https://c4model.com/) 组织架构视图,从不同抽象层次描述 Waterflow 系统架构:
- **Context View (系统上下文)** - Waterflow 在整体生态中的定位
- **Container View (容器视图)** - 核心组件及其交互
- **Component View (组件视图)** - 各容器内部结构
- **Data Flow (数据流)** - 典型场景的执行流程
- **Deployment View (部署视图)** - 运行时部署架构
- **Quality Attributes (质量属性)** - 非功能性需求

核心架构决策记录在 [Architecture Decision Records (ADR)](adr/README.md) 中。

---

## 1. Context View (系统上下文)

### 1.1 系统定位

Waterflow 是一个**声明式工作流编排服务**,为 [Temporal](https://temporal.io) 提供 YAML DSL 接口和分布式 Agent 执行模式。

**定位:** 简化分布式工作流编排,让用户通过 YAML 定义工作流,无需编写 Temporal SDK 代码。

**核心价值:**
- 声明式 DSL → 降低编程门槛
- 分布式 Agent → 跨服务器任务编排  
- 生产级可靠性 → 基于 Temporal 的持久化执行

### 1.2 系统边界

```
┌─────────────────────────────────────────────────────────────┐
│                      User Application                       │
│  (业务系统通过 REST API/SDK 集成 Waterflow)                  │
└──────────────┬──────────────────────────────────────────────┘
               │ REST API
               ↓
┌──────────────────────────────────────────────────────────────┐
│                        Waterflow                             │
│  ┌──────────┐    ┌──────────┐    ┌────────────────────┐     │
│  │  Server  │───→│ Temporal │←───│ Distributed Agents │     │
│  │(REST API)│    │ (Engine) │    │  (Task Executors)  │     │
│  └──────────┘    └──────────┘    └────────────────────┘     │
└──────────────────────────────────────────────────────────────┘
               │
               ↓ 执行任务
┌──────────────────────────────────────────────────────────────┐
│              Target Infrastructure                           │
│  (Linux Servers, Docker Containers)                          │
└──────────────────────────────────────────────────────────────┘
```

**Waterflow 负责:**
- YAML DSL 解析与验证
- REST API 服务 (工作流提交、状态查询、日志获取)
- DSL → Temporal Workflow 转换
- 分布式 Agent 任务执行
- 插件化节点系统

**Temporal 负责 (内部依赖):**
- 工作流状态持久化 (Event Sourcing)
- 分布式任务调度 (Task Queue)
- 容错与自动重试
- 执行历史存储

**用户应用负责:**
- 集成 Waterflow REST API/SDK
- 业务层权限控制
- UI 界面(可选)
- 服务器凭证管理

### 1.3 典型使用场景

| 场景 | 示例 |
|------|------|
| **应用部署** | 跨多服务器编排容器化应用部署 |
| **批量运维** | 对服务器组执行批量巡检、配置更新 |
| **CI/CD 流水线** | 声明式定义构建、测试、部署流程 |
| **定时任务** | 可靠的分布式定时任务调度 |
| **数据处理** | 跨多节点的 ETL 数据处理流程 |

### 1.4 技术定位对比

| 对比维度 | Waterflow | Temporal | GitHub Actions |
|---------|-----------|----------|----------------|
| **抽象层级** | 高 (YAML DSL) | 中 (SDK 编程) | 高 (YAML) |
| **用户界面** | REST API + CLI | Web UI + SDK | Web UI + API |
| **部署模式** | 自托管 | 自托管/云服务 | 云服务 |
| **工作流定义** | YAML | 代码 | YAML |
| **执行环境** | 分布式 Agent | Worker Pool | GitHub Runners |
| **目标场景** | 通用工作流编排 | 企业级工作流引擎 | CI/CD 专用 |

---

## 2. Container View (容器视图)

### 2.1 核心容器

Waterflow 系统由 3 个核心容器组成:

```
┌────────────────┐         ┌──────────────────┐
│ User           │         │ Waterflow Server │
│ Application    │────────→│ (REST API)       │
│                │ HTTPS   │                  │
└────────────────┘         └─────────┬────────┘
                                     │ gRPC
                                     ↓
                           ┌──────────────────┐
                           │ Temporal Server  │
                           │ (Workflow Engine)│
                           └─────────┬────────┘
                                     │ gRPC
                                     ↓
                           ┌──────────────────┐
                           │ Agent Workers    │
                           │ (Task Executors) │
                           └──────────────────┘
```

#### Container 1: Waterflow Server

**技术:** Go + gorilla/mux  
**职责:**
- 提供 REST API 端点
- 解析和验证 YAML DSL
- 转换 DSL 为 Temporal Workflow 参数
- 作为 Temporal Client 提交工作流
- 查询工作流状态和日志

**关键决策:** [ADR-0004: YAML DSL 语法设计](adr/0004-yaml-dsl-syntax.md)

#### Container 2: Temporal Server

**技术:** Temporal (Go)  
**职责:**
- 工作流状态持久化 (Event Sourcing)
- 任务调度和路由 (Task Queue)
- 容错与自动重试
- 执行历史存储
- Worker 健康检查

**关键决策:** [ADR-0001: 使用 Temporal 作为工作流引擎](adr/0001-use-temporal-workflow-engine.md)

**外部依赖:** PostgreSQL/MySQL/Cassandra (持久化存储)

#### Container 3: Agent Workers

**技术:** Go + Temporal Worker SDK  
**职责:**
- 作为 Temporal Worker 连接到 Temporal Server
- 注册到特定 Task Queue (基于服务器组)
- 加载插件化节点 (.so 文件)
- 执行节点任务
- 上报心跳和日志

**关键决策:**
- [ADR-0002: 单节点执行模式](adr/0002-single-node-execution-pattern.md)
- [ADR-0003: 插件化节点系统](adr/0003-plugin-based-node-system.md)
- [ADR-0006: Task Queue 路由机制](adr/0006-task-queue-routing.md)

### 2.2 容器交互

#### 工作流提交流程

```
User App ──POST /v1/workflows──→ Server ──ExecuteWorkflow──→ Temporal
                                            (DSL → Workflow)
```

#### 任务执行流程

```
Temporal ──Task Queue──→ Agent ──Execute Node──→ Target Server
         (runs-on)       (Plugin)              (Docker/SSH)
```

#### 状态查询流程

```
User App ──GET /v1/workflows/{id}──→ Server ──GetWorkflow──→ Temporal
                                              (Query Event History)
```

---

## 3. Component View (组件视图)

### 3.1 Server 内部组件

```
┌─────────────────────────────────────────────────────────┐
│                  Waterflow Server                       │
│                                                         │
│  ┌────────────┐    ┌────────────┐    ┌──────────────┐  │
│  │ REST API   │───→│ DSL Parser │───→│ Temporal     │  │
│  │ Handler    │    │ (YAML)     │    │ Client       │  │
│  └────────────┘    └────────────┘    └──────────────┘  │
│         │                 │                            │
│         ↓                 ↓                            │
│  ┌────────────┐    ┌────────────┐                     │
│  │ Validator  │    │ Expression │                     │
│  │ (Schema)   │    │ Engine     │                     │
│  └────────────┘    └────────────┘                     │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

#### 3.1.1 REST API Handler

**职责:**
- 处理 HTTP 请求 (提交工作流、查询状态、获取日志)
- 请求参数验证
- API 认证 (API Key/JWT)
- 错误响应格式化 (RFC 7807)

**关键端点:**
- `POST /v1/workflows` - 提交工作流
- `GET /v1/workflows/{id}` - 查询状态
- `POST /v1/workflows/{id}/cancel` - 取消执行
- `GET /v1/workflows/{id}/logs` - 获取日志
- `POST /v1/validate` - 验证 YAML 语法

#### 3.1.2 DSL Parser

**职责:**
- 解析 YAML 工作流定义
- 构建 AST (Abstract Syntax Tree)
- 生成 Temporal Workflow 参数

**实现:**
```go
type WorkflowDSL struct {
    Name string
    On   map[string]interface{}
    Jobs map[string]JobDSL
}

type JobDSL struct {
    RunsOn  string
    Timeout int
    Steps   []StepDSL
}

type StepDSL struct {
    Name    string
    Uses    string
    With    map[string]interface{}
    Timeout int
    Retry   *RetryConfig
}
```

**关键决策:** [ADR-0004: YAML DSL 语法设计](adr/0004-yaml-dsl-syntax.md)

#### 3.1.3 Expression Engine

**职责:**
- 解析表达式 `${{ expression }}`
- 求值上下文变量
- 支持运算符和内置函数

**示例:**
```yaml
steps:
  - name: Use Output
    with:
      commit: ${{ steps.checkout.outputs.commit }}
  
  - name: Conditional Step
    if: ${{ job.status == 'success' }}
```

**关键决策:** [ADR-0005: 表达式系统语法](adr/0005-expression-system-syntax.md)

**实现库:** [antonmedv/expr](https://github.com/antonmedv/expr) (MVP 阶段)

#### 3.1.4 Validator

**职责:**
- JSON Schema 验证
- 语法错误定位 (行号、字段)
- 语义验证 (节点是否存在、参数类型)

#### 3.1.5 Temporal Client

**职责:**
- 提交 Workflow 到 Temporal Server
- 查询 Workflow 状态
- 取消/重试 Workflow

**连接管理:**
```go
type Server struct {
    temporalClient client.Client // 连接池复用
}

func NewServer(temporalAddr string) (*Server, error) {
    client, err := client.NewClient(client.Options{
        HostPort: temporalAddr,
    })
    return &Server{temporalClient: client}, nil
}
```

### 3.2 Agent 内部组件

```
┌─────────────────────────────────────────────────────────┐
│                     Agent Worker                        │
│                                                         │
│  ┌────────────┐    ┌────────────┐    ┌──────────────┐  │
│  │ Temporal   │───→│ Plugin     │───→│ Node         │  │
│  │ Worker     │    │ Manager    │    │ Executors    │  │
│  └────────────┘    └────────────┘    └──────────────┘  │
│         │                 │                 │          │
│         ↓                 ↓                 ↓          │
│  ┌────────────┐    ┌────────────┐    ┌──────────────┐  │
│  │ Workflow   │    │ Node       │    │ .so Plugins  │  │
│  │ Handlers   │    │ Registry   │    │ (/plugins/)  │  │
│  └────────────┘    └────────────┘    └──────────────┘  │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

#### 3.2.1 Temporal Worker

**职责:**
- 连接到 Temporal Server
- 注册到 Task Queue (基于 `runs-on` 配置)
- 轮询任务并执行
- 上报心跳和执行状态

**Task Queue 路由:**
```go
func (w *Worker) Start(taskQueues []string) error {
    for _, queue := range taskQueues {
        worker := worker.New(w.temporalClient, queue, worker.Options{})
        worker.RegisterWorkflow(RunJobWorkflow)
        worker.RegisterActivity(w.ExecuteNode)
        go worker.Run(worker.InterruptCh())
    }
    return nil
}
```

**关键决策:** [ADR-0006: Task Queue 路由机制](adr/0006-task-queue-routing.md)

#### 3.2.2 Plugin Manager

**职责:**
- 扫描插件目录 (`/opt/waterflow/plugins/`)
- 加载 .so 文件
- 调用插件的 `Register()` 函数
- 热加载支持 (fsnotify 监控)

**加载流程:**
```go
func (pm *PluginManager) LoadPlugin(path string) error {
    // 1. 加载 .so 文件
    p, err := plugin.Open(path)
    
    // 2. 查找 Register 函数
    symbol, err := p.Lookup("Register")
    
    // 3. 调用 Register 注册节点
    register := symbol.(func() node.Node)
    nodeInstance := register()
    
    // 4. 注册到 NodeRegistry
    pm.nodeRegistry.Register(nodeInstance)
    
    return nil
}
```

**关键决策:** [ADR-0003: 插件化节点系统](adr/0003-plugin-based-node-system.md)

#### 3.2.3 Node Registry

**职责:**
- 存储已注册的节点实现
- 根据 `uses` 字段查找节点
- 提供节点元数据

**接口:**
```go
type NodeRegistry interface {
    Register(node Node) error
    Get(nodeType string) (Node, error)
    List() []NodeMetadata
}

type Node interface {
    Execute(ctx context.Context, args map[string]interface{}) (NodeResult, error)
}
```

#### 3.2.4 Workflow Handlers

**职责:**
- 实现 Temporal Workflow 函数
- 编排 Job 执行
- 管理执行上下文

**单节点执行模式:**
```go
func RunJobWorkflow(ctx workflow.Context, job JobDSL) error {
    execCtx := &ExecutionContext{
        Outputs: make(map[string]interface{}),
    }
    
    for _, step := range job.Steps {
        // 每个 Step 独立配置超时/重试
        opts := workflow.ActivityOptions{
            StartToCloseTimeout: time.Duration(step.Timeout) * time.Minute,
            RetryPolicy: &temporal.RetryPolicy{
                MaximumAttempts: step.Retry.Attempts,
            },
        }
        ctx = workflow.WithActivityOptions(ctx, opts)
        
        // 一个 Step = 一个 Activity 调用
        var result NodeResult
        err := workflow.ExecuteActivity(ctx, "ExecuteNode", step, execCtx).Get(ctx, &result)
        
        if err != nil && !step.ContinueOnError {
            return err
        }
        
        execCtx.Outputs[step.ID] = result.Output
    }
    return nil
}
```

**关键决策:** [ADR-0002: 单节点执行模式](adr/0002-single-node-execution-pattern.md)

#### 3.2.5 Node Executors

**职责:**
- 执行具体节点逻辑
- 与目标基础设施交互 (Docker/SSH/HTTP)
- 返回执行结果

**插件示例:**
```go
// plugins/checkout/main.go
package main

import (
    "context"
    "waterflow/pkg/node"
)

type CheckoutNode struct{}

func (n *CheckoutNode) Execute(ctx context.Context, args map[string]interface{}) (node.NodeResult, error) {
    repo := args["repository"].(string)
    // Git checkout 逻辑
    return node.NodeResult{
        Outputs: map[string]string{"commit": "abc123"},
    }, nil
}

func Register() node.Node {
    return &CheckoutNode{}
}
```

### 3.3 数据模型

#### 3.3.1 DSL 模型

```go
type WorkflowDSL struct {
    Name    string
    On      TriggerConfig
    Jobs    map[string]JobDSL
    Env     map[string]string
}

type JobDSL struct {
    RunsOn         string
    TimeoutMinutes int
    Needs          []string
    Steps          []StepDSL
    Env            map[string]string
}

type StepDSL struct {
    ID             string
    Name           string
    Uses           string
    With           map[string]interface{}
    TimeoutMinutes int
    Retry          *RetryConfig
    If             string
    ContinueOnError bool
}

type RetryConfig struct {
    Attempts         int
    InitialInterval  string
    BackoffCoefficient float64
}
```

#### 3.3.2 执行上下文

```go
type ExecutionContext struct {
    WorkflowID string
    Outputs    map[string]interface{}
    Env        map[string]string
}
```

#### 3.3.3 节点结果

```go
type NodeResult struct {
    Output map[string]string
    Logs   []string
    Error  error
}
```

---

## 4. Data Flow (数据流)

### 4.1 工作流提交与执行流程

```
┌──────────┐     ┌────────┐     ┌──────────┐     ┌───────┐
│User App  │────→│ Server │────→│ Temporal │────→│ Agent │
└──────────┘     └────────┘     └──────────┘     └───────┘
     │                │               │               │
     │ 1. POST        │               │               │
     │  /v1/workflows │               │               │
     │  (YAML)        │               │               │
     ├───────────────→│               │               │
     │                │ 2. Parse DSL  │               │
     │                │    Validate   │               │
     │                │               │               │
     │                │ 3. Execute    │               │
     │                │    Workflow   │               │
     │                ├──────────────→│               │
     │                │               │ 4. Schedule   │
     │                │               │    Activity   │
     │                │               │   (Task Queue)│
     │                │               ├──────────────→│
     │                │               │               │ 5. Load Plugin
     │                │               │               │    Execute Node
     │                │               │               │
     │                │               │ 6. Complete   │
     │                │               │←──────────────│
     │                │               │               │
     │ 7. Response    │               │               │
     │   (workflow_id)│               │               │
     │←───────────────│               │               │
```

**详细步骤:**

1. **User App → Server**: 提交 YAML 工作流定义
2. **Server**: 解析 YAML,生成 WorkflowDSL 对象,验证语法
3. **Server → Temporal**: 调用 `ExecuteWorkflow()`,传入 WorkflowDSL
4. **Temporal**: 根据 `runs-on` 字段路由任务到对应 Task Queue
5. **Agent**: Worker 轮询 Task Queue,获取任务,加载节点插件并执行
6. **Agent → Temporal**: 上报执行结果和日志
7. **Server → User App**: 返回 `workflow_id`

### 4.2 状态查询流程

```
┌──────────┐     ┌────────┐     ┌──────────┐
│User App  │────→│ Server │────→│ Temporal │
└──────────┘     └────────┘     └──────────┘
     │                │               │
     │ GET /v1/       │               │
     │ workflows/{id} │               │
     ├───────────────→│               │
     │                │ GetWorkflow() │
     │                ├──────────────→│
     │                │               │ Query Event
     │                │               │ History
     │                │               │
     │                │ WorkflowStatus│
     │                │←──────────────│
     │ Response       │               │
     │ (status, logs) │               │
     │←───────────────│               │
```

**Event Sourcing 优势:**
- Temporal 通过 Event History 重建完整状态
- Server 无状态,崩溃后仍可查询所有工作流
- 时间旅行:可查看任意时间点的状态

### 4.3 单节点执行流程

基于 [ADR-0002](adr/0002-single-node-execution-pattern.md),每个 Step 映射为一个 Activity 调用:

```
Workflow Context
  │
  ├─ Job 1
  │   ├─ Step 1 ──→ Activity Call (独立超时 5min,重试 3次)
  │   ├─ Step 2 ──→ Activity Call (独立超时 30min,重试 1次)
  │   └─ Step 3 ──→ Activity Call (独立超时 10min,重试 2次)
  │
  └─ Job 2
      ├─ Step 1 ──→ Activity Call
      └─ Step 2 ──→ Activity Call
```

**优势:**
- 每个 Step 独立配置超时和重试
- Temporal UI 清晰展示每个 Step 状态
- 失败的 Step 可单独重试
- 自然支持并发执行 (future)

### 4.4 Task Queue 路由流程

基于 [ADR-0006](adr/0006-task-queue-routing.md),`runs-on` 直接映射到 Task Queue:

```
DSL:
  jobs:
    build-linux:
      runs-on: linux-amd64  ──→ Task Queue: "linux-amd64"
    
    build-mac:
      runs-on: macos-arm64  ──→ Task Queue: "macos-arm64"

Agent 配置:
  Agent-1: task-queues: [linux-amd64, linux-common]
  Agent-2: task-queues: [macos-arm64, mac-common]
  Agent-3: task-queues: [linux-amd64, gpu-a100]

路由结果:
  build-linux  → Agent-1 或 Agent-3 (负载均衡)
  build-mac    → Agent-2
```

### 4.5 插件加载流程

基于 [ADR-0003](adr/0003-plugin-based-node-system.md),所有节点都是插件:

```
Agent 启动
  │
  ├─ 扫描 /opt/waterflow/plugins/
  │   ├─ checkout.so
  │   ├─ run.so
  │   └─ custom-deploy.so
  │
  ├─ 加载每个 .so 文件
  │   ├─ plugin.Open("checkout.so")
  │   ├─ p.Lookup("Register")
  │   └─ register() → Node 实例
  │
  └─ 注册到 NodeRegistry
      └─ nodeRegistry.Register("checkout@v1", nodeInstance)

执行时:
  DSL: uses: checkout@v1
    ↓
  NodeRegistry.Get("checkout@v1")
    ↓
  node.Execute(ctx, args)
```

**热加载:**
```
fsnotify 监控 /opt/waterflow/plugins/
  │
  ├─ 检测到 new-node.so 添加
  ├─ 自动加载并注册
  └─ 新任务立即可用
```

---
## 5. Deployment View (部署视图)

### 5.1 MVP 部署架构

```
┌─────────────────────────────────────────────────────────┐
│                  Docker Compose Host                    │
│                                                         │
│  ┌────────────────┐         ┌────────────────────┐     │
│  │ Waterflow      │         │ Temporal Server    │     │
│  │ Server         │────────→│                    │     │
│  │ :8080          │  gRPC   │ :7233              │     │
│  └────────────────┘         └─────────┬──────────┘     │
│         ↑                              │                │
│         │ HTTPS                        │                │
│         │                              ↓                │
│  ┌────────────────┐         ┌────────────────────┐     │
│  │ Reverse Proxy  │         │ PostgreSQL         │     │
│  │ (Traefik/Nginx)│         │ (Temporal DB)      │     │
│  └────────────────┘         └────────────────────┘     │
│                                                         │
└─────────────────────────────────────────────────────────┘
                    │
                    │ gRPC
                    ↓
┌─────────────────────────────────────────────────────────┐
│              Target Servers (Agent Deployment)          │
│                                                         │
│  Server-1:           Server-2:           Server-3:      │
│  ┌──────────┐        ┌──────────┐       ┌──────────┐   │
│  │  Agent   │        │  Agent   │       │  Agent   │   │
│  │ (Docker) │        │ (Docker) │       │ (Binary) │   │
│  └──────────┘        └──────────┘       └──────────┘   │
│  Task Queue:         Task Queue:        Task Queue:    │
│  linux-amd64         linux-amd64        gpu-a100        │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### 5.2 Docker Compose 配置

```yaml
version: '3.8'

services:
  postgresql:
    image: postgres:14
    environment:
      POSTGRES_PASSWORD: temporal
      POSTGRES_USER: temporal
    volumes:
      - postgres_data:/var/lib/postgresql/data

  temporal:
    image: temporalio/auto-setup:1.22.0
    depends_on:
      - postgresql
    environment:
      - DB=postgresql
      - DB_PORT=5432
      - POSTGRES_USER=temporal
      - POSTGRES_PWD=temporal
      - POSTGRES_SEEDS=postgresql
    ports:
      - "7233:7233"

  waterflow-server:
    image: waterflow/server:latest
    depends_on:
      - temporal
    environment:
      - TEMPORAL_HOST=temporal:7233
      - PORT=8080
      - API_KEY=${API_KEY}
    ports:
      - "8080:8080"

  waterflow-agent:
    image: waterflow/agent:latest
    depends_on:
      - temporal
    environment:
      - TEMPORAL_HOST=temporal:7233
      - TASK_QUEUES=linux-amd64,linux-common
      - PLUGIN_DIR=/opt/waterflow/plugins
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./plugins:/opt/waterflow/plugins

volumes:
  postgres_data:
```

### 5.3 Agent 部署模式

#### 模式 1: Docker 容器 (推荐)

```bash
docker run -d \
  --name waterflow-agent \
  -e TEMPORAL_HOST=temporal.example.com:7233 \
  -e TASK_QUEUES=linux-amd64,gpu-a100 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /opt/waterflow/plugins:/opt/waterflow/plugins \
  waterflow/agent:latest
```

#### 模式 2: 二进制部署

```bash
# 下载 Agent 二进制
wget https://github.com/waterflow/releases/agent-linux-amd64

# 配置
cat > /etc/waterflow/agent.yaml <<EOF
temporal:
  host: temporal.example.com:7233
task-queues:
  - linux-amd64
  - linux-common
plugin-dir: /opt/waterflow/plugins
EOF

# 启动 Systemd 服务
systemctl start waterflow-agent
```

### 5.4 网络拓扑

```
Internet
    │
    ↓
┌─────────────────┐
│ Load Balancer   │
│ (HTTPS :443)    │
└────────┬────────┘
         │
         ↓
┌─────────────────┐
│ Waterflow Server│
│ (HTTP :8080)    │
└────────┬────────┘
         │ gRPC :7233
         ↓
┌─────────────────┐         ┌──────────────┐
│ Temporal Server │────────→│ PostgreSQL   │
│                 │  :5432  │              │
└────────┬────────┘         └──────────────┘
         │ gRPC :7233
         ↓
┌──────────────────────────┐
│ Agent Workers (Firewall) │
│ (outbound only)          │
└──────────────────────────┘
```

**安全要点:**
- Agent 仅需 **outbound** 连接到 Temporal (7233)
- 无需在 Agent 服务器开放端口
- Temporal 使用 mTLS 加密通信

---

## 6. Quality Attributes (质量属性)

### 6.1 可靠性 (Reliability)

**容错机制:**
- 基于 Temporal 的 Event Sourcing 实现状态持久化
- Server/Agent 崩溃后自动恢复执行
- 节点执行失败自动重试 (可配置策略)
- Agent 离线不影响系统运行

**重试策略:**
```yaml
steps:
  - name: Deploy
    uses: deploy@v1
    timeout-minutes: 30
    retry:
      attempts: 3
      initial-interval: 1s
      backoff-coefficient: 2.0
```

**故障隔离:**
- 单个节点失败不影响其他节点
- `continue-on-error: true` 支持失败继续
- Job 级别隔离 (一个 Job 失败不影响其他 Job)

### 6.2 性能 (Performance)

**目标指标:**
- Server 启动时间 < 5秒
- YAML 解析 (1000行) < 100ms
- 工作流提交 API 响应 < 500ms
- 单工作流支持 100+ Agent 并发执行
- Agent 内存占用 < 50MB (空闲)

**优化策略:**
- DSL 解析结果缓存
- Temporal Worker Pool 并发处理
- gRPC 连接池复用
- Continue-As-New 处理超大工作流

**扩展性:**
- Server 水平扩展 (无状态设计)
- Agent 按需添加
- Temporal 集群扩展

### 6.3 安全性 (Security)

**凭证管理:**
- Waterflow 零凭证存储原则
- 节点执行时凭证由用户应用实时传入
- 支持外部密钥管理系统集成 (Vault)

**通信安全:**
- Agent ↔ Temporal: mTLS 加密
- User ↔ Server: HTTPS/TLS
- API 认证: API Key (MVP) / JWT (Post-MVP)

**多租户隔离:**
- API 支持 `tenant_id` 透传
- Temporal Namespace 物理隔离
- 隔离逻辑由用户应用实现

**审计:**
- Temporal Event History 记录完整执行链路
- 结构化审计日志输出
- API 调用日志

### 6.4 可维护性 (Maintainability)

**代码质量:**
- 统一 Go 技术栈
- 单元测试覆盖率 > 80%
- 集成测试覆盖核心场景
- Go 标准错误处理

**文档:**
- REST API 文档 (OpenAPI 3.0)
- DSL 语法文档
- 节点开发指南
- 部署运维文档

**可观测性:**
- 结构化日志 (JSON)
- Prometheus 指标导出
- Temporal UI 查看执行历史
- 错误链路追踪

### 6.5 可扩展性 (Extensibility)

**插件化节点:**
- 所有节点都是插件 (.so 文件)
- 节点接口向后兼容
- 热加载支持
- 第三方可开发自定义节点

**DSL 版本化:**
- 语法版本 v1.0 锁定后保持兼容
- 新增功能通过可选字段扩展
- 弃用功能保留兼容期

**API 版本化:**
- REST API 使用 `/v1/` 前缀
- 语义化版本管理
- 向后兼容原则

### 6.6 易用性 (Usability)

**低学习成本:**
- GitHub Actions 风格语法 (用户熟悉)
- 详细的错误提示 (行号、字段定位)
- 丰富的示例和模板

**开发体验:**
- CLI 工具快速验证
- 本地 Docker Compose 一键启动
- 节点开发周期 < 2小时

**集成便利:**
- REST API + SDK 支持
- OpenAPI 文档自动生成
- 多语言客户端示例

### 6.7 部署简单性 (Deployability)

**快速启动:**
- Docker Compose 一键部署
- 预构建 Docker 镜像

**配置简单:**
- 环境变量配置
- 默认配置开箱即用
- 可选配置文件 (YAML)

**最小依赖:**
- 仅依赖 Temporal Server
- 无需额外数据库
- 轻量级运行时

---

## 7. 关键技术决策

所有核心架构决策记录在 [ADR](adr/README.md) 中:

| ADR | 决策 | 理由 |
|-----|------|------|
| [ADR-0001](adr/0001-use-temporal-workflow-engine.md) | 使用 Temporal 作为工作流引擎 | 成熟稳定,持久化执行,生产就绪 |
| [ADR-0002](adr/0002-single-node-execution-pattern.md) | 单节点执行模式 | 独立超时/重试,精确控制,易调试 |
| [ADR-0003](adr/0003-plugin-based-node-system.md) | 插件化节点系统 | 可扩展,热加载,统一机制 |
| [ADR-0004](adr/0004-yaml-dsl-syntax.md) | YAML DSL 语法 | 用户熟悉,可读性好,生态成熟 |
| [ADR-0005](adr/0005-expression-system-syntax.md) | 表达式系统语法 | GitHub Actions 兼容,安全沙箱 |
| [ADR-0006](adr/0006-task-queue-routing.md) | Task Queue 路由机制 | 简单直观,零配置,灵活扩展 |

---

## 8. MVP 范围

### 8.1 核心交付物

1. **Waterflow Server** - REST API 服务
2. **Waterflow Agent** - 分布式任务执行器
3. **DSL Engine** - YAML 解析和表达式引擎
4. **10 个内置节点** - 基础执行能力
5. **CLI 工具** - 开发测试
6. **Docker Compose 部署** - 快速启动
7. **REST API 文档** - OpenAPI 3.0
8. **DSL 语法文档** - 用户参考

### 8.2 内置节点列表

| 节点 | 功能 | 优先级 |
|------|------|--------|
| `checkout@v1` | Git 仓库检出 | P0 |
| `run@v1` | Shell 命令执行 | P0 |
| `docker@v1` | Docker 命令执行 | P0 |
| `http@v1` | HTTP 请求 | P1 |
| `file-transfer@v1` | 文件传输 | P1 |
| `sleep@v1` | 延迟等待 | P1 |
| `if@v1` | 条件判断 | P1 |
| `for-each@v1` | 循环迭代 | P2 |
| `docker-compose@v1` | Docker Compose | P2 |
| `cache@v1` | 缓存管理 | P2 |

### 8.3 明确排除

**不在 MVP 范围:**
- Web UI (由用户应用自建)
- 复杂用户认证系统 (简单 API Key 即可)
- 工作流可视化编辑器
- 内置监控面板
- 多语言 SDK (Post-MVP)
- Agent 自动部署工具
- 复杂负载均衡算法

### 8.4 技术快捷方式 (可接受)

- 简单 API Key 认证 (无 RBAC)
- 基于环境变量的配置 (无动态配置)
- 同步工作流提交 (无异步队列)
- 简单轮询负载均衡 (无智能调度)
- 基础 CLI 输出 (无 TUI 界面)

---

## 9. 技术栈

### 9.1 核心依赖

**Waterflow Server:**
```go
require (
    go.temporal.io/sdk v1.25.0              // Temporal Go SDK
    github.com/gorilla/mux v1.8.1           // HTTP 路由器
    github.com/spf13/viper v1.21.0          // 配置管理
    go.uber.org/zap v1.27.1                 // 结构化日志
    github.com/prometheus/client_golang v1.23.2 // Prometheus 指标
    github.com/google/uuid v1.6.0           // UUID 生成
    gopkg.in/yaml.v3 v3.0.1                 // YAML 解析
    github.com/antonmedv/expr v1.15.0       // 表达式引擎
)
```

**Waterflow Agent:**
```go
require (
    go.temporal.io/sdk v1.25.0          // Temporal Worker SDK
    go.uber.org/zap v1.27.1             // 日志
    github.com/docker/docker v24.0.0    // Docker SDK
    github.com/fsnotify/fsnotify v1.9.0 // 文件监控(热加载)
)
```

**CLI:**
```go
require (
    github.com/spf13/cobra v1.7.0     // CLI 框架
    github.com/olekukonko/tablewriter // 表格输出
)
```

### 9.2 外部依赖

- **Temporal Server** v1.22+ (工作流引擎)
- **PostgreSQL** 14+ (Temporal 持久化)
- **Docker** 20.10+ (可选,容器节点)

---

## 10. 风险与缓解

### 10.1 技术风险

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| Temporal 学习曲线陡峭 | 高 | 中 | 团队完成官方教程,构建 PoC 验证 |
| Go Plugin 跨平台限制 | 中 | 高 | Windows 使用内置编译 fallback |
| DSL 语法设计缺陷 | 高 | 中 | 参考 GitHub Actions 成熟语法 |
| 性能不满足大规模场景 | 中 | 低 | 使用 Continue-As-New,分批处理 |

### 10.2 架构风险

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| Temporal 单点依赖 | 高 | Temporal 支持集群部署,HA 架构 |
| 插件沙箱隔离不足 | 中 | 限制插件能力,未来考虑 WASM |
| Event History 过大 | 低 | Continue-As-New 分批处理 |

---

## 11. 后续演进方向

### Phase 1: MVP (3-4 个月)
- ✅ 核心功能实现
- ✅ Docker Compose 部署
- ✅ 10 个内置节点

### Phase 2: 生产就绪 (2-3 个月)
- Prometheus 监控
- 多语言 SDK (Python, Node.js)
- 高级节点 (Ansible)

### Phase 3: 企业特性 (3-6 个月)
- Web UI (工作流编辑、监控)
- RBAC 权限系统
- 审计日志
- 工作流市场

### Phase 4: 高级能力 (持续)
- 工作流可视化编辑器
- AI 辅助工作流生成
- 多云编排能力
- WASM 插件支持

---

## 12. 参考资料

- [PRD: 产品需求文档](prd.md)
- [Epics: 功能分解](epics.md)
- [ADR: 架构决策记录](adr/README.md)
- [Temporal 官方文档](https://docs.temporal.io)
- [GitHub Actions 语法参考](https://docs.github.com/en/actions/reference/workflow-syntax-for-github-actions)
- [C4 Model](https://c4model.com/)

