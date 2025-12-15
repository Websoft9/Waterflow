---
stepsCompleted: [1, 2]
inputDocuments:
  - /data/Waterflow/docs/prd.md
workflowType: 'architecture'
lastStep: 2
project_name: 'Waterflow'
user_name: 'Websoft9'
date: '2025-12-13'
---

# Architecture Decision Document

_This document builds collaboratively through step-by-step discovery. Sections are appended as we work through each architectural decision together._

---

## 项目背景分析

### 1. 产品定位

Waterflow 是一个**声明式工作流编排服务**,提供企业级的分布式任务执行能力。通过 YAML DSL 定义工作流,基于 Temporal 提供生产级的持久化执行,通过分布式 Agent 实现跨服务器的任务编排。

**核心架构:** Waterflow Server (REST API) → Temporal (内部运行时) → 分布式 Agent

#### 1.1 设计目标

为 Temporal 提供声明式 DSL 和分布式 Agent 模式,降低工作流编排门槛。用户通过 REST API 提交 YAML 工作流,Waterflow Server 将其转换为 Temporal Workflow 执行,Agent 在目标服务器上完成具体任务。用户无需了解 Temporal 复杂性。

#### 1.2 典型使用场景

- **应用部署**: 跨多服务器编排容器化应用部署流程
- **批量运维**: 对服务器组执行批量巡检、配置更新
- **DevOps 自动化**: 声明式定义 CI/CD 流水线,替代复杂脚本
- **定时任务**: 可靠的分布式定时任务调度

#### 1.3 技术定位对比

| 对比维度 | Waterflow | Temporal | GitHub Actions |
|---------|-----------|----------|----------------|
| **抽象层级** | 高 (YAML DSL) | 中 (SDK) | 高 (YAML) |
| **用户界面** | REST API + CLI | Web UI + SDK | Web UI + API |
| **部署模式** | 独立服务 | 独立服务 | 云服务 |
| **工作流定义** | YAML | 代码 (Go/Java/Python) | YAML |
| **执行环境** | 分布式 Agent | Worker Pool | GitHub Runners |
| **目标场景** | 通用工作流编排 | 企业级工作流引擎 | CI/CD 专用 |

#### 1.4 系统边界

**Waterflow 提供:**
- YAML DSL 解析与验证
- REST API (工作流提交、查询、控制)
- 工作流到 Temporal Workflow 的转换
- 分布式 Agent 任务执行
- 节点执行逻辑和任务路由
- 执行状态和日志输出
- CLI 工具 (开发测试)
- 多语言 SDK (可选,便于集成)

**Temporal 负责 (内部实现细节):**
- 工作流状态持久化
- 分布式任务调度
- 自动重试和容错
- 执行历史存储

**用户应用负责:**
- 通过 REST API/SDK 集成 Waterflow
- 管理服务器凭证和连接信息
- 构建业务层 UI (可选)
- 实现业务级权限控制 (可选)

### 2. 功能需求

#### 2.1 Waterflow Server

提供 REST API 服务,处理工作流提交和管理:
- **工作流提交**: 接收 YAML 定义,解析并提交到 Temporal
- **状态查询**: 查询工作流执行状态和进度
- **日志获取**: 实时或历史日志流
- **生命周期管理**: 取消、重试、暂停工作流
- **DSL 验证**: YAML 语法验证和错误提示

#### 2.2 DSL 引擎

解析 YAML 工作流定义,转换为 Temporal Workflow:
- YAML 语法解析和验证 (jobs, steps, runs-on, with)
- 变量引用系统 (`${{ vars.name }}`)
- 条件执行 (`if: ${{ condition }}`)
- 循环控制 (`for-each`)
- DSL 到 Temporal Workflow 代码的动态生成
- 语法错误定位和提示

#### 2.3 分布式 Agent

Agent 作为 Temporal Worker 在目标服务器上执行任务:
- 连接到 Temporal Server
- 按服务器组 (Server Group) 注册
- 接收并执行工作流任务
- 上报执行状态和日志
- 健康检查和自动重连

#### 2.4 节点系统

可插拔的任务执行单元,MVP 提供 10 个核心节点:

**控制流节点:**
- 条件判断 (if/else)
- 循环迭代 (for-each)
- 延迟等待 (sleep)

**运维基础节点:**
- Shell 命令执行
- 文件传输 (上传/下载)
- HTTP 请求
- 环境变量设置

**Docker 管理节点:**
- Docker 命令执行
- Docker Compose Up
- Docker Compose Down

**扩展机制:**
- 节点接口规范 (Go interface)
- 参数 Schema 验证 (JSON Schema)
- 宿主应用可注册自定义节点

#### 2.5 REST API

**核心端点:**
- `POST /v1/workflows` - 提交工作流
- `GET /v1/workflows/{id}` - 查询状态
- `POST /v1/workflows/{id}/cancel` - 取消执行
- `GET /v1/workflows/{id}/logs` - 获取日志流
- `POST /v1/workflows/{id}/retry` - 重试失败工作流
- `POST /v1/validate` - 验证 YAML 语法
- `GET /v1/nodes` - 列出可用节点
- `GET /v1/agents` - 列出 Agent 状态

**认证策略:**
- MVP: 简单 API Key 认证
- Post-MVP: JWT + RBAC

**API 规范:**
- OpenAPI 3.0 文档自动生成
- RESTful 设计原则
- 统一错误格式 (RFC 7807)

#### 2.6 多语言 SDK (可选)

为便于集成,提供 SDK 封装 REST API:
- **Go SDK**: 原生实现,类型安全
- **Python SDK**: 适用 Python 生态集成
- **Node.js SDK**: 适用 JavaScript/TypeScript 应用

SDK 示例 (Go):
```go
client := waterflow.NewClient("http://waterflow-server:8080", apiKey)

// 提交工作流
workflowID, err := client.SubmitWorkflow(ctx, yamlContent, 
  waterflow.WithVariables(vars),
)

// 查询状态
status, err := client.GetWorkflowStatus(ctx, workflowID)
```

#### 2.7 CLI 工具

用于开发测试和快速验证:
- `waterflow validate <file>` - 验证 YAML 语法
- `waterflow submit <file>` - 提交工作流
- `waterflow status <workflow-id>` - 查询状态
- `waterflow logs <workflow-id>` - 查看日志
- `waterflow cancel <workflow-id>` - 取消执行
- `waterflow node list` - 列出可用节点
- `waterflow agent list` - 列出 Agent 状态

#### 2.8 工作流模板

提供预定义工作流模板,加速常见场景开发。

**API 接口:**
```bash
GET /v1/templates - 列出所有模板
GET /v1/templates/{name} - 获取模板 YAML
```

**MVP 内置模板:**
- 单服务器应用部署
- 多服务器批量巡检  
- 分布式应用部署 (WordPress + MySQL)
- Docker Compose 部署
- 文件同步任务

**用户自定义模板:**
- 用户应用可上传自定义模板
- 模板市场 (社区贡献)

### 3. 非功能性需求

#### 3.1 部署简单性

**快速启动:**
- Docker Compose 一键部署 (Waterflow Server + Temporal + Agents)
- Kubernetes Helm Chart 生产部署
- 预构建 Docker 镜像

**配置简单:**
- 环境变量配置 (Temporal 地址、端口、日志级别)
- 可选配置文件 (YAML/TOML)
- 默认配置开箱即用

**最小依赖:**
- 仅依赖 Temporal Server (可与 Waterflow 一起部署)
- 无需额外数据库 (Temporal 自带持久化)
- 轻量级运行时

#### 3.2 性能

- Server 启动时间 < 5 秒
- YAML 解析 (1000 行) < 100ms
- 工作流提交 API 响应 < 500ms
- 单工作流支持 100+ Agent 并发执行
- Temporal Worker Pool 并发处理
- Agent 内存占用 < 50MB (空闲)

#### 3.3 可靠性

- 工作流执行失败支持重试
- Agent 离线不影响系统运行
- 基于 Temporal 的容错机制
- 健康检查和任务路由

#### 3.4 可维护性

- 统一 Go 技术栈
- 节点开发周期 < 2 小时
- 完善的 SDK 文档和示例
- 单元测试覆盖率 > 80%
- 集成测试工具包

#### 3.5 可扩展性

- 节点接口向后兼容
- 宿主应用可注册自定义节点
- DSL 语法版本化
- 支持未来多语言节点 (gRPC)

#### 3.6 安全性

**凭证管理:**
- Waterflow Server 不存储任何凭证
- SSH 密钥/密码由用户应用管理
- 节点执行时凭证通过 API 实时传入

**通信安全:**
- Agent 通过 Temporal 通信 (mTLS)
- REST API 支持 HTTPS/TLS
- 可选 API Key/JWT 认证

**多租户隔离:**
- API 支持租户标识透传
- Temporal Namespace 物理隔离
- 隔离逻辑由用户应用实现

**审计:**
- 结构化审计事件输出
- 工作流执行历史跟踪
- API 调用日志

#### 3.7 文档完善性

**REST API 文档:**
- OpenAPI 3.0 规范
- 自动生成 API 文档
- 多语言客户端示例 (curl, Python, Go, Node.js)
- 错误码参考

**DSL 语法文档:**
- YAML 语法参考
- 表达式系统说明
- 内置节点参考
- 完整示例工作流

**部署指南:**
- Docker Compose 快速启动
- Kubernetes 生产部署
- Agent 配置指南
- 故障排查

**开发指南:**
- 自定义节点开发
- SDK 集成示例
- 最佳实践

### 4. 项目规模评估

#### 4.1 技术复杂度

- **分布式系统**: Server-Agent 架构,跨服务器执行 → 中高
- **领域复杂度**: DevOps 工作流编排,多技术栈 → 中
- **技术栈深度**: Temporal 深度集成,分布式工作流模式 → 中高
- **接口设计**: SDK 公共 API 需向后兼容 → 高

#### 4.2 MVP 范围

**核心交付物:**
1. **Waterflow Server** (REST API 服务)
2. **Agent** (分布式任务执行器)
3. **DSL 引擎** (YAML 解析和验证)
4. **10 个内置节点**
5. **CLI 工具** (开发测试)
6. **5 个工作流模板**
7. **Docker Compose 部署方案**
8. **REST API 文档** (OpenAPI 3.0)
9. **DSL 语法文档**

**明确排除:**
- Web UI (由用户应用自建)
- 用户认证授权系统 (简单 API Key 认证)
- 复杂 Agent 自动部署
- 工作流可视化编辑器
- 内置监控面板
- 多语言 SDK (Post-MVP)

**可接受的技术快捷方式:**
- 简单轮询负载均衡
- 基于文件的服务器组配置
- 基础 CLI 输出
- 同步工作流提交
- 简单 API Key 认证

#### 4.3 团队与周期

- 团队规模: 2-3 名 Go 后端工程师
- 开发周期: 3-4 个月 (MVP)
- 关键技能: 
  - Go 服务开发 (REST API, 分布式系统)
  - Temporal 深度理解 (Workflow/Activity 模式)
  - DSL 设计和解析
  - Docker/Kubernetes 部署

#### 4.4 关键技术决策

**必须一次做对 (无法后期修改):**
- REST API 路由和端点设计
- DSL 语法规范 (YAML 结构/字段)
- 节点接口设计
- 服务器组抽象
- Temporal Workflow 模式
- 数据模型 (Workflow/Job/Step 状态)

**可后期优化:**
- CLI 命令结构
- 负载均衡算法
- 性能优化
- 监控指标粒度
- 日志格式细节

### 5. 技术约束与依赖

#### 5.1 核心技术栈

**Temporal 工作流引擎:**
- 版本: Temporal Server (v1.20+) + Temporal Go SDK (v1.25+)
- 约束: 深度理解 Workflow/Activity 模式
- 风险缓解: 团队完成 Temporal 官方教程,开发 PoC 验证
- 架构影响:
  - Waterflow Server 作为 Temporal Client 提交工作流
  - Agent 设计为 Temporal Worker
  - 工作流状态依赖 Temporal 持久化

**Go 语言生态:**
- 版本: Go 1.21+
- Server 核心依赖: 
  - Temporal Go SDK
  - Gin/Echo (REST API 框架)
  - Viper (配置管理)
  - Zap/Logrus (日志)
- CLI 依赖: Cobra (CLI 框架)
- 原则: 最小化依赖,优先使用标准库

**Docker 容器技术:**
- Waterflow Server Docker 镜像
- Agent Docker 镜像 (推荐部署方式)
- Docker 管理节点需访问 Docker Socket
- Docker Compose 一键部署

**数据存储:**
- 工作流状态: Temporal 持久化 (PostgreSQL/MySQL/Cassandra)
- Server 配置: 环境变量/配置文件 (YAML/TOML)
- Agent 注册信息: 内存 (MVP) / Redis/etcd (Post-MVP)

#### 5.2 平台约束

- **Server**: Linux (Docker/Kubernetes 部署)
- **Agent**: Linux 服务器、容器环境
- **CLI**: Linux/MacOS/Windows (Go 交叉编译)
- **网络**: Agent 需连接 Temporal Server (端口 7233)
- **部署**: Docker Compose / Kubernetes Helm Chart

#### 5.3 架构约束

**通信协议:**
- 必须使用 Temporal SDK (Server ↔ Temporal ↔ Agent)
- 不可自定义通信协议
- REST API 使用 HTTP/HTTPS

**状态管理:**
- 完全依赖 Temporal 持久化
- Waterflow 不实现独立状态存储
- Agent 注册信息可存入内存/Redis/etcd

**配置方式:**
- Server: 环境变量 + 配置文件 (YAML/TOML)
- Agent: 环境变量 + 配置文件
- CLI: 命令行参数 + 配置文件 (可选)

**日志输出:**
- 结构化日志 (JSON 格式)
- 支持多种日志库 (Zap/Logrus/slog)
- 输出到 stdout/stderr 或文件

#### 5.4 可扩展性边界

- 节点接口向后兼容,支持未来多语言节点
- DSL 语法 v1.0 版本锁定后保持兼容
- Go SDK API 遵循语义化版本
- REST API 使用 `/v1/` 前缀版本化
- 宿主应用可注册自定义节点

### 6. 横切关注点

#### 6.1 分布式协调与容错

利用 Temporal 的 Workflow 持久化和自动重试机制:
- **任务分发**: Agent 作为 Temporal Worker,任务分发由 Temporal 管理
- **健康检查**: 确保只向在线 Agent 分发任务
- **确定性要求**: 工作流定义必须是确定性的 (Temporal 要求)
- **重试策略**: 节点执行封装为 Temporal Activity,支持超时和重试配置
- **状态恢复**: Server 重启后,运行中的工作流自动恢复

**架构优化 (基于 Temporal 深度分析):**

**优化 1: 解释器模式 (确定性保证)**
```go
// ❌ 错误: 在 Workflow 中解析 DSL (非确定性)
func WaterflowWorkflow(ctx workflow.Context, yamlContent string) error {
    dsl := parseYAML(yamlContent) // 修改解析器会破坏确定性!
}

// ✅ 正确: DSL 解析在 Server 端
// Server 端 (可迭代修改)
func (s *Server) SubmitWorkflow(yamlContent string) error {
    dsl, err := s.parser.Parse(yamlContent) // 解析在这里
    return s.temporalClient.ExecuteWorkflow(ctx, options, WaterflowWorkflow, dsl)
}

// Workflow (确定性,代码固定)
func WaterflowWorkflow(ctx workflow.Context, dsl WorkflowDSL) error {
    // 接收已解析对象,确定性保证
}
```

**优化 2: 批量执行 Activity (性能优化)**
```go
// ❌ 问题: 每个 step 一个 Activity → 1000 steps = 1000 Events
for _, step := range job.Steps {
    workflow.ExecuteActivity(ctx, ExecuteStepActivity, step)
}

// ✅ 优化: 一个 job 所有 steps 打包成一个 Activity
func JobWorkflow(ctx workflow.Context, job JobDSL) error {
    return workflow.ExecuteActivity(ctx, ExecuteJobActivity, job.Steps).Get(ctx, nil)
}

func ExecuteJobActivity(ctx context.Context, steps []StepDSL) error {
    for i, step := range steps {
        executor := nodeRegistry.Get(step.Uses)
        executor.Execute(ctx, step.With)
        // 心跳上报进度
        activity.RecordHeartbeat(ctx, map[string]interface{}{
            "progress": float64(i+1) / float64(len(steps)) * 100,
        })
    }
    return nil
}
```

**优化 3: Task Queue 路由 (runs-on 映射)**
```go
// runs-on: web-servers → TaskQueue: "server-group-web"
for _, job := range dsl.Jobs {
    workflow.ExecuteChildWorkflow(
        workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
            TaskQueue: job.RunsOn, // 直接映射到队列名
        }),
        JobWorkflow,
        job,
    )
}

// Agent 启动时注册到特定队列
func main() {
    serverGroup := os.Getenv("SERVER_GROUP") // "web-servers"
    worker := worker.New(temporalClient, serverGroup, worker.Options{})
    worker.Run(worker.InterruptCh())
}
```

#### 6.2 状态持久化与可观测性

依赖 Temporal 的 Event History 实现状态管理:
- CLI/SDK 通过 Temporal Client 查询工作流状态和日志
- 日志分级: 工作流级、Job 级、Step 级
- 支持实时日志流 (tail -f 模式)
- Post-MVP 考虑独立日志聚合

**Event Sourcing 架构优势:**

```
Temporal 持久化机制:

1. WorkflowExecutionStarted Event (工作流开始)
   ↓
2. ChildWorkflowExecutionStarted (Job 开始)
   ↓
3. ActivityTaskScheduled Event (Activity 调度)
   ↓
4. ActivityTaskCompleted Event (Activity 完成,包含结果)
   ↓
5. ChildWorkflowExecutionCompleted (Job 完成)
   ↓
6. WorkflowExecutionCompleted Event (工作流完成)

关键特性:
✅ 状态重建: 通过重放 Event History 恢复完整状态
✅ 崩溃恢复: Server/Worker 崩溃后,从 Event 继续执行
✅ 时间旅行: 可查看任意时间点的工作流状态
✅ 审计追溯: 完整 Event 链,所有操作可追溯
```

**Waterflow Server 无状态设计:**
```go
// Server 不存储任何状态,仅作为 API Gateway
func (s *Server) GetWorkflowStatus(workflowID string) (*WorkflowStatus, error) {
    // 直接查询 Temporal
    execution := s.temporalClient.GetWorkflow(ctx, workflowID, "")
    
    // 从 Event History 重建状态
    var status WorkflowStatus
    err := execution.Get(ctx, &status)
    
    return &status, err
}

// Server 崩溃不影响工作流执行
// 重启后仍可查询所有运行中的工作流
```

#### 6.3 可插拔节点架构

节点通过接口注册和发现:
- 节点接口: `Execute(ctx, inputs) (outputs, error)`
- 节点元数据: name, version, description, input_schema, output_schema
- 参数 Schema 验证使用 JSON Schema
- MVP 内置节点编译到二进制,Post-MVP 支持插件加载

#### 6.4 配置管理

SDK 零配置文件设计:
- 配置通过构造函数和 Option 模式传入
- 示例: `waterflow.NewEngine(client, waterflow.WithLogger(log))`
- CLI/REST API Server 可选使用 Viper 读取配置
- 配置模型统一,传递方式灵活

#### 6.5 安全与权限

Waterflow 最小化凭证存储原则:
- **零凭证存储**: 不存储任何 SSH 密钥、密码、Token
- **API 认证**: MVP 支持简单 API Key,Post-MVP 支持 JWT + RBAC
- **凭证注入**: 节点执行时,凭证由用户应用通过 API 实时传入
- **多租户**: API 支持传入 `tenant_id`,隔离逻辑由用户应用实现
- **Agent 权限**: 由部署者控制 (最小权限原则)
- **通信加密**: Agent 通过 Temporal 通信 (mTLS),REST API 支持 HTTPS

#### 6.6 错误处理

使用 Go 标准错误包装和类型化错误:
- 错误分类: `ErrInvalidYAML`, `ErrNodeNotFound`, `ErrWorkflowTimeout`
- DSL 验证器提供精确错误位置 (行号、字段)
- 节点执行错误包含上下文 (服务器、步骤、参数)
- REST API 返回 RFC 7807 Problem Details
- SDK 支持 Context 传递,宿主应用可控制超时和取消

#### 6.7 性能优化

MVP 阶段基础优化策略:
- DSL 解析缓存 (相同工作流不重复解析)
- Temporal Worker Pool 并发处理
- 服务器组简单轮询负载均衡
- Post-MVP 优化为智能调度
- 避免过早优化,先验证核心功能

**大规模工作流优化 (基于 Temporal 分析):**

**优化 1: Continue-As-New (超大工作流)**
```go
// 问题: 1000+ jobs 会导致 Event History 过大
// 解决: 分批处理,使用 Continue-As-New

func WaterflowWorkflow(ctx workflow.Context, dsl WorkflowDSL, startIndex int) error {
    batchSize := 100 // 每次处理 100 个 job
    endIndex := min(startIndex+batchSize, len(dsl.Jobs))
    
    // 处理当前批次
    for i := startIndex; i < endIndex; i++ {
        workflow.ExecuteChildWorkflow(ctx, JobWorkflow, dsl.Jobs[i])
    }
    
    // 超过阈值,继续下一批
    if endIndex < len(dsl.Jobs) {
        return workflow.NewContinueAsNewError(ctx, WaterflowWorkflow, dsl, endIndex)
    }
    return nil
}
```

**优化 2: 并发执行 Jobs**
```go
// 问题: 顺序执行 jobs 太慢
// 解决: 根据 needs 依赖并发执行

func WaterflowWorkflow(ctx workflow.Context, dsl WorkflowDSL) error {
    // 构建 DAG
    dag := buildDAG(dsl.Jobs)
    
    // 拓扑排序,识别可并发 jobs
    for _, level := range dag.Levels {
        // 同一层级的 jobs 并发执行
        futures := []workflow.Future{}
        for _, job := range level {
            future := workflow.ExecuteChildWorkflow(ctx, JobWorkflow, job)
            futures = append(futures, future)
        }
        
        // 等待当前层级全部完成
        for _, f := range futures {
            f.Get(ctx, nil)
        }
    }
    return nil
}
```

**优化 3: Workflow 代码缓存**
```go
// Server 端缓存已注册的 Workflow
var workflowRegistry sync.Map

func init() {
    // 只注册一次
    workflowRegistry.Store("WaterflowWorkflow", WaterflowWorkflow)
    workflowRegistry.Store("JobWorkflow", JobWorkflow)
}

// Agent Worker 启动时批量注册
func (w *AgentWorker) RegisterWorkflows() {
    w.worker.RegisterWorkflow(WaterflowWorkflow)
    w.worker.RegisterWorkflow(JobWorkflow)
    w.worker.RegisterActivity(ExecuteJobActivity)
}
```

**优化 4: gRPC 连接池复用**
```go
// Server 端复用 Temporal Client 连接
type Server struct {
    temporalClient client.Client // 单例,连接池复用
}

func NewServer(temporalAddr string) (*Server, error) {
    client, err := client.NewClient(client.Options{
        HostPort: temporalAddr,
        // 连接池配置
        ConnectionOptions: client.ConnectionOptions{
            MaxConcurrentSessionExecutionSize: 100,
        },
    })
    
    return &Server{temporalClient: client}, nil
}
```

