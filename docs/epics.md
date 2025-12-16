---
stepsCompleted: [1, 2, 3, 4]
inputDocuments:
  - /data/Waterflow/docs/prd.md
  - /data/Waterflow/docs/architecture.md
workflowType: 'epics-and-stories'
lastStep: 4
project_name: 'Waterflow'
user_name: 'Websoft9'
date: '2025-12-15'
status: 'complete'
---

# Waterflow - Epic Breakdown

## Overview

本文档将 Waterflow 的 PRD 和架构需求分解为可执行的 Epic 和 User Stories。

## Requirements Inventory

### Functional Requirements (功能需求)

从 PRD 提取的核心功能需求:

**FR1:** 提供 YAML DSL 解析和验证能力,支持工作流定义 (jobs, steps, runs-on, with, if 条件)

**FR2:** 通过 REST API 提供工作流管理能力 (提交、查询状态、取消执行、获取日志、验证 YAML)

**FR3:** 支持分布式 Agent 部署在目标服务器上执行任务

**FR4:** 实现服务器组 (Server Group) 概念,支持工作流目标特定服务器组执行

**FR5:** 提供 10 个核心节点 (编译为 .so 插件,随 Agent 镜像发布):
  - 控制流: condition (if/else), loop (for-each), sleep
  - 操作: shell, script, file/transfer, http/request
  - Docker: docker/exec, docker/compose-up, docker/compose-down

**FR6:** 支持自定义节点开发和热加载 (通过 Go Plugin 机制,自动注册到 NodeRegistry)

**FR7:** 通过 Temporal 实现工作流状态持久化,支持进程重启后恢复

**FR8:** 支持节点级别的重试策略配置 (超时、重试次数、指数退避)

**FR9:** 提供实时工作流执行状态跟踪和结构化日志输出

**FR10:** 提供 CLI 工具用于 YAML 验证、工作流提交、状态查询、日志查看

**FR11:** 提供 Go SDK 封装 REST API,简化 Go 语言集成

**FR12:** 支持工作流模板库 (至少 3 个内置模板: 单服务器部署、多服务器健康检查、分布式栈部署)

**FR13:** 支持并行执行多服务器任务

**FR14:** Agent 健康监控和自动故障检测

**FR15:** 支持变量引用系统 `${{ vars.name }}`

**FR16:** 支持条件执行 `if: ${{ condition }}`

### Non-Functional Requirements (非功能需求)

**NFR1: 部署简单性**
- Docker Compose 一键部署 (Waterflow Server + Temporal + Agents) ≤10 分钟
- 预构建 Docker 镜像

**NFR2: 性能**
- Server 启动时间 < 5 秒
- YAML 解析 (1000 行) < 100ms
- 工作流提交 API 响应 < 500ms
- 支持 ≥100 个并发 Agent 连接
- Agent 内存占用 < 50MB (空闲)

**NFR3: 可靠性**
- 工作流状态在进程故障后 100% 恢复 (依赖 Temporal)
- Agent 连续丢失 3 次心跳后标记为不健康
- 支持节点级别重试策略

**NFR4: 可观测性**
- 结构化日志输出 (JSON 格式)
- 完整的工作流执行历史追踪
- API 调用日志记录

**NFR5: 可扩展性**
- 节点接口向后兼容
- 支持自定义节点开发 (插件 API 每个节点 <50 LOC)
- DSL 语法版本化

**NFR6: 安全性**
- 零凭证存储 (SSH 密钥/密码由用户应用管理)
- Agent 通过 Temporal 通信 (mTLS)
- REST API 支持 HTTPS/TLS
- 可选 API Key/JWT 认证

**NFR7: 文档完善性**
- OpenAPI 3.0 REST API 规范
- 快速开始指南 (30 分钟从部署到首个工作流)
- 完整的 YAML DSL 语法参考
- 10 个内置节点文档
- 自定义节点开发指南

**NFR8: 跨平台支持**
- Server: Linux (Docker 部署)
- Agent: Linux 服务器、容器环境
- CLI: Linux/MacOS/Windows (Go 交叉编译)

### Additional Requirements (架构和实现需求)

从架构文档提取的技术实现需求:

**AR1: 技术栈选型**
- 语言: Go 1.21+
- 工作流运行时: Temporal Server (v1.20+) + Temporal Go SDK (v1.25+)
- HTTP 框架: Gin/Echo (REST API 服务)
- CLI 框架: Cobra
- 配置管理: Viper
- 日志: Zap/Logrus
- 容器化: Docker

**AR2: 架构约束**
- 必须使用 Temporal SDK (Server ↔ Temporal ↔ Agent)
- 工作流状态完全依赖 Temporal 持久化 (Event Sourcing 模式)
- 所有执行状态存储在 Temporal Event History,Server 无状态
- 单节点执行模式: 每个 Step 映射为 1 个 Temporal Activity 调用 (ADR-0002)
- Agent 注册信息可存入内存/Redis/etcd (MVP 使用内存)
- 配置通过环境变量 + 配置文件 (YAML/TOML)

**AR3: 接口设计**
- ServerGroupProvider: 提供服务器组和 Agent 清单
- SecretProvider: 提供工作流所需密钥 (运行时注入)
- EventHandler: 接收工作流事件 (可选)
- LogHandler: 接收工作流执行日志 (可选)
- 每个接口 ≤3 个方法,提供默认实现

**AR4: 数据模型**
- Workflow: 工作流定义和执行状态 (状态存储在 Temporal Event History)
- Job: 工作流中的作业单元
- Step: 作业中的步骤 (每个 Step = 1 个 Activity 调用)
- Node: 可执行的节点类型 (以 .so 插件形式实现)
- ServerGroup: 服务器组逻辑集合 (直接映射到 Task Queue)
- Agent: Worker 进程实例 (加载插件并执行 Activity)

**AR5: 部署方案**
- Docker Compose: 开发和生产部署
- 二进制独立部署 (Server + Agent)

**AR6: 错误处理**
- Go 标准错误包装和类型化错误
- DSL 验证器提供精确错误位置 (行号、字段)
- REST API 返回 RFC 7807 Problem Details
- 节点执行错误包含上下文 (服务器、步骤、参数)

**AR7: 测试策略**
- 单元测试覆盖率 > 80%
- 每个节点的集成测试
- 端到端验收测试 (2 个场景: 健康检查 + 分布式部署)
- 性能基准测试

**AR8: 文档结构**
- getting-started/ (快速开始、安装、首个工作流)
- guides/ (Server 部署、Agent 设置、自定义节点、故障排查)
- reference/ (REST API、DSL 语法、Go SDK、CLI、节点参考、配置)
- concepts/ (架构、执行模型、服务器组、节点系统)
- examples/ (工作流模板示例)

**AR9: CI/CD**
- GitHub Actions 自动化构建
- golangci-lint, gosec 代码质量检查
- Docker 镜像自动构建和推送
- 二进制发布到 GitHub Releases

**AR10: 分发方式**
- Docker Hub / GitHub Container Registry (镜像)
- GitHub Releases (二进制: Linux, macOS, Windows)
- Go modules: `go get github.com/websoft9/waterflow`

### FR Coverage Map

**FR1** → Epic 1 (YAML DSL 解析和验证)  
**FR2** → Epic 1 (REST API 服务)  
**FR3** → Epic 2 (分布式 Agent 部署)  
**FR4** → Epic 2 (服务器组概念)  
**FR5** → Epic 3 (10 个内置节点)  
**FR6** → Epic 4 (自定义节点扩展)  
**FR7** → Epic 1 (持久化执行)  
**FR8** → Epic 4, Epic 8 (重试策略配置)  
**FR9** → Epic 1, Epic 8 (状态跟踪和日志)  
**FR10** → Epic 6 (CLI 工具)  
**FR11** → Epic 6 (Go SDK)  
**FR12** → Epic 7 (工作流模板库)  
**FR13** → Epic 2 (并行执行)  
**FR14** → Epic 2 (健康监控)  
**FR15** → Epic 2, Epic 5 (ServerGroupProvider接口 + 变量引用系统)  
**FR16** → Epic 5 (条件执行)  
**FR17** → Epic 8 (EventHandler接口) ✨ **新增**  
**FR18** → Epic 8 (LogHandler接口) ✨ **新增**  
**FR19** → Epic 9 (Docker镜像打包 - Server和Agent) ✨ **已完善**  
**FR20** → Epic 1, Epic 9 (Docker Compose部署)  
**FR21** → Epic 11 (文档结构)  

**NFR1** → Epic 9 (部署简单性)  
**NFR2** → Epic 8 (性能)  
**NFR3** → Epic 8 (可靠性)  
**NFR4** → Epic 8 (可观测性)  
**NFR5** → Epic 4 (可扩展性)  
**NFR6** → Epic 10 (安全性)  
**NFR7** → Epic 11 (文档完善性)  
**NFR8** → Epic 9 (跨平台支持)  

**AR1-AR10** → 分布在 Epic 1, 4, 8, 9, 10, 11, 12 (架构和实现需求)

## Epic List

### Epic 1: 核心工作流引擎基础
开发者可以部署 Waterflow Server,通过 Temporal 执行基本的 YAML 工作流定义,查看执行状态和日志

**FRs covered:** FR1, FR2, FR7, FR9

### Epic 2: 分布式 Agent 系统
运维工程师可以在多台服务器上部署 Agent,工作流可以将任务分发到特定服务器组执行,实现跨服务器编排。支持通过 ServerGroupProvider 接口集成外部 CMDB 系统。

**FRs covered:** FR3, FR4, FR13, FR14, FR15 (ServerGroupProvider接口)

### Epic 3: 核心节点插件库
用户可以使用 10 个核心节点构建实用的工作流,覆盖控制流、Shell 操作、文件传输、HTTP 请求、Docker 管理等常见场景。所有节点都编译为 .so 插件,通过 Agent 启动时自动加载 (ADR-0003 插件化节点系统)。

**FRs covered:** FR5

### Epic 4: 节点扩展系统

开发者可以创建自定义节点扩展 Waterflow 能力,通过简单的插件 API (<50 LOC) 注册新节点类型,支持热加载无需重启 (ADR-0003 插件化节点系统)

**FRs covered:** FR6, FR8, NFR5

### Epic 5: 高级 DSL 功能
用户可以使用变量引用、条件执行等高级 DSL 功能,编写更灵活和可复用的工作流

**FRs covered:** FR15, FR16

### Epic 6: 客户端工具和 SDK
开发者可以使用 CLI 工具快速验证和测试工作流,或通过 Go SDK 将 Waterflow 集成到 Go 应用中

**FRs covered:** FR10, FR11

### Epic 7: 工作流模板库
用户可以从预定义模板快速开始,了解 Waterflow 的最佳实践和常见模式

**FRs covered:** FR12

### Epic 8: 生产级可靠性
Waterflow 在生产环境中稳定运行,支持故障恢复、性能优化、完善的错误处理。提供 EventHandler 和 LogHandler 接口用于集成外部监控和日志系统。

**FRs covered:** FR8, FR17 (EventHandler), FR18 (LogHandler), NFR2, NFR3, NFR4

### Epic 9: 部署和运维
用户可以通过 Docker Compose 快速部署开发和生产环境。提供 Waterflow Server 和 Agent 的 Docker 镜像,支持一键部署完整栈。

**FRs covered:** FR19 (Docker镜像 - Server和Agent), FR20 (Docker Compose), NFR1, NFR8

### Epic 10: 安全和认证
Waterflow 支持 API 认证,保护敏感凭证,提供安全的通信机制

**FRs covered:** NFR6

### Epic 11: 完整文档体系
用户可以通过完善的文档自助完成从入门到高级使用的全部流程,无需人工支持

**FRs covered:** NFR7

### Epic 12: 质量保证和发布
Waterflow 通过全面测试验证,提供稳定的发布版本和多种分发渠道

**FRs covered:** AR7, AR9, AR10

---

## Epic 1: 核心工作流引擎基础

开发者可以部署 Waterflow Server,通过 Temporal 执行基本的 YAML 工作流定义,查看执行状态和日志

### Story 1.1: Waterflow Server 框架搭建

As a **开发者**,  
I want **搭建 Waterflow Server 的基础框架和目录结构**,  
So that **后续可以在统一的架构上开发各个功能模块**。

**Acceptance Criteria:**

**Given** Go 1.21+ 开发环境已配置  
**When** 执行项目初始化命令  
**Then** 创建标准 Go 项目结构 (cmd/, pkg/, internal/, api/)  
**And** 包含 Makefile, go.mod, Dockerfile  
**And** 配置 golangci-lint 和代码质量检查  
**And** 基础 CI 管道 (GitHub Actions) 可以构建项目

### Story 1.2: REST API 服务框架

As a **开发者**,  
I want **实现 REST API 服务框架**,  
So that **可以通过 HTTP 接口接收工作流请求**。

**Acceptance Criteria:**

**Given** Waterflow Server 框架已搭建  
**When** 启动 Server 进程  
**Then** HTTP 服务监听在配置的端口 (默认 8080)  
**And** 提供健康检查端点 `GET /health` 返回 200  
**And** 提供就绪检查端点 `GET /ready` 返回服务状态  
**And** 支持优雅关闭 (SIGTERM)  
**And** 配置通过环境变量或 YAML 文件加载  
**And** 结构化日志输出到 stdout

### Story 1.3: YAML DSL 解析器

As a **工作流用户**,  
I want **提交 YAML 格式的工作流定义**,  
So that **系统可以解析并验证工作流语法**。

**Acceptance Criteria:**

**Given** 一个符合规范的 YAML 工作流文件  
**When** 通过 API 提交工作流内容  
**Then** 系统成功解析 YAML 结构 (name, jobs, steps)  
**And** 验证必需字段存在 (job name, runs-on, steps)  
**And** 语法错误返回具体错误位置 (行号、字段名)  
**And** 支持基本字段: jobs, steps, runs-on, uses, with  
**And** 解析结果转换为内部数据结构

### Story 1.4: Temporal SDK 集成

As a **系统架构师**,  
I want **集成 Temporal Go SDK**,  
So that **可以利用 Temporal 的持久化执行能力**。

**Acceptance Criteria:**

**Given** Temporal Server 已部署并可访问  
**When** Waterflow Server 启动时  
**Then** 成功连接到 Temporal Server  
**And** 创建 Temporal Client 实例  
**And** 注册 Waterflow Namespace  
**And** 连接失败时记录错误并重试  
**And** 配置连接参数 (host, port, namespace) 可通过配置文件设置

### Story 1.5: 工作流提交 API

As a **工作流用户**,  
I want **通过 REST API 提交工作流**,  
So that **触发工作流执行**。

**Acceptance Criteria:**

**Given** REST API 服务和 Temporal 集成已完成  
**When** POST `/v1/workflows` 请求带有 YAML 内容  
**Then** 返回工作流 ID 和提交状态  
**And** 工作流 ID 唯一且可追踪  
**And** 请求格式错误返回 400 和详细错误信息  
**And** YAML 验证失败返回 422 和语法错误位置  
**And** 响应时间 <500ms  
**And** 工作流提交到 Temporal 执行队列

### Story 1.6: 基础工作流执行引擎

As a **系统**,  
I want **将解析的 YAML 工作流转换为 Temporal Workflow 执行**,  
So that **工作流可以持久化运行**。

**Acceptance Criteria:**

**Given** 工作流已通过 API 提交  
**When** 工作流开始执行  
**Then** 创建 Temporal Workflow 实例  
**And** 工作流状态持久化到 Temporal  
**And** 支持单 Job、单 Step 的简单工作流  
**And** Step 执行结果记录到 Temporal Event History  
**And** 工作流执行失败时状态正确记录

### Story 1.7: 工作流状态查询 API

As a **工作流用户**,  
I want **查询工作流的执行状态**,  
So that **了解工作流进度和结果**。

**Acceptance Criteria:**

**Given** 工作流已提交并执行  
**When** GET `/v1/workflows/{id}` 查询工作流  
**Then** 返回工作流状态 (running, completed, failed)  
**And** 返回执行进度 (当前 Job/Step)  
**And** 返回开始时间和持续时间  
**And** 工作流不存在返回 404  
**And** 响应时间 <200ms

### Story 1.8: 工作流日志输出

As a **工作流用户**,  
I want **获取工作流执行日志**,  
So that **调试失败原因和验证执行过程**。

**Acceptance Criteria:**

**Given** 工作流正在执行或已完成  
**When** GET `/v1/workflows/{id}/logs` 请求日志  
**Then** 返回结构化日志 (JSON 格式)  
**And** 日志包含时间戳、级别、Job/Step 信息、消息  
**And** 支持日志级别过滤 (info, warn, error)  
**And** 支持实时日志流 (SSE 或 WebSocket)  
**And** 历史日志从 Temporal Event History 获取

### Story 1.9: 工作流取消 API

As a **工作流用户**,  
I want **取消正在运行的工作流**,  
So that **停止不需要的执行释放资源**。

**Acceptance Criteria:**

**Given** 工作流正在运行  
**When** POST `/v1/workflows/{id}/cancel` 请求取消  
**Then** 工作流标记为 cancelled 状态  
**And** Temporal Workflow 收到取消信号  
**And** 正在执行的 Step 优雅停止  
**And** 取消已完成的工作流返回 409  
**And** 取消成功返回 202

### Story 1.10: Docker Compose 部署方案

As a **开发者**,  
I want **通过 Docker Compose 一键部署 Waterflow + Temporal**,  
So that **快速搭建开发环境**。

**Acceptance Criteria:**

**Given** 安装了 Docker 和 Docker Compose  
**When** 执行 `docker-compose up`  
**Then** 启动 Temporal Server (含 PostgreSQL)  
**And** 启动 Waterflow Server 并连接到 Temporal  
**And** 所有服务健康检查通过  
**And** Waterflow API 可访问 (http://localhost:8080)  
**And** 提供 README 说明部署步骤  
**And** 部署时间 <10 分钟

---

## Epic 2: 分布式 Agent 系统

运维工程师可以在多台服务器上部署 Agent,工作流可以将任务分发到特定服务器组执行,实现跨服务器编排

### Story 2.1: Agent Worker 基础框架

As a **开发者**,  
I want **创建 Agent Worker 的基础框架**,  
So that **Agent 可以作为 Temporal Worker 运行**。

**Acceptance Criteria:**

**Given** Go 开发环境和 Temporal SDK  
**When** 构建 Agent 二进制  
**Then** Agent 可以独立启动  
**And** 通过配置连接到 Temporal Server  
**And** 注册为 Temporal Worker  
**And** 配置 Worker Task Queue 名称  
**And** 支持优雅关闭  
**And** 日志输出到 stdout

### Story 2.2: 服务器组概念和 ServerGroupProvider 接口实现

As a **系统架构师**,  
I want **实现服务器组 (Server Group) 的概念和 ServerGroupProvider 接口**,  
So that **工作流可以目标特定服务器组执行任务,并支持与外部 CMDB 集成**。

**Acceptance Criteria:**

**Given** Agent 和 Server 已实现  
**When** Agent 启动时指定服务器组名称  
**Then** Agent 注册到对应的 Task Queue (以组名命名)  
**And** 定义 ServerGroupProvider 接口包含方法:  
**And** GetServers(ctx, groupName) ([]ServerInfo, error) - 查询服务器组中的 Agent 清单  
**And** ServerInfo 包含: agentID, hostname, ip, status, lastHeartbeat  
**And** 提供默认内存实现 (InMemoryServerGroupProvider)  
**And** 提供配置文件实现 (从 YAML/JSON 加载服务器组定义)  
**And** Server 配置支持注入自定义 ServerGroupProvider  
**And** Server 维护服务器组和 Agent 的映射关系  
**And** 工作流中 `runs-on` 字段指定服务器组  
**And** 任务路由到正确的服务器组  
**And** 支持多个 Agent 属于同一个组  
**And** 提供 CMDB 集成示例 (如何实现自定义 Provider)  
**And** 文档说明如何实现自定义 ServerGroupProvider

### Story 2.3: Agent 注册和心跳

As a **运维工程师**,  
I want **Agent 自动注册并维持心跳**,  
So that **Server 知道哪些 Agent 在线可用**。

**Acceptance Criteria:**

**Given** Agent 已启动并连接到 Temporal  
**When** Agent 开始运行  
**Then** Agent 向 Server 注册 (服务器组、主机名、IP)  
**And** 每 30 秒发送心跳  
**And** 心跳包含 Agent 状态 (空闲/繁忙)  
**And** Server 记录 Agent 最后心跳时间  
**And** 连续 3 次心跳失败标记 Agent 为 unhealthy  
**And** Agent 重连后自动恢复 healthy 状态

### Story 2.4: 任务分发到 Agent

As a **工作流用户**,  
I want **工作流任务分发到指定服务器组的 Agent**,  
So that **任务在正确的服务器上执行**。

**Acceptance Criteria:**

**Given** 工作流定义了 `runs-on: web-servers`  
**When** 工作流执行到该 Job  
**Then** 任务发送到 `web-servers` Task Queue  
**And** 该组的任意 Agent 接收任务  
**And** 无可用 Agent 时任务排队等待  
**And** Agent 执行完成后返回结果  
**And** 执行失败时记录错误信息

### Story 2.5: 并行执行多服务器任务

As a **工作流用户**,  
I want **工作流在多个服务器上并行执行任务**,  
So that **提高执行效率**。

**Acceptance Criteria:**

**Given** 工作流有多个 Job 分别目标不同服务器组  
**When** 工作流执行  
**Then** 多个 Job 并行启动  
**And** 每个 Job 路由到对应的服务器组  
**And** Job 独立执行互不阻塞  
**And** 所有 Job 完成后工作流标记为完成  
**And** 任一 Job 失败不影响其他 Job (除非定义依赖)

### Story 2.6: Agent 健康监控

As a **系统管理员**,  
I want **监控 Agent 的健康状态**,  
So that **及时发现故障 Agent**。

**Acceptance Criteria:**

**Given** 多个 Agent 在运行  
**When** 查询 Agent 状态  
**Then** API 返回所有 Agent 列表  
**And** 每个 Agent 显示状态 (healthy/unhealthy)  
**And** 显示最后心跳时间  
**And** 显示当前任务数  
**And** Unhealthy Agent 不接收新任务  
**And** 提供 `GET /v1/agents` 端点

### Story 2.7: 简单负载均衡

As a **系统**,  
I want **在服务器组内均衡分配任务**,  
So that **避免单个 Agent 过载**。

**Acceptance Criteria:**

**Given** 服务器组有多个 Agent  
**When** 多个任务分发到该组  
**Then** 任务轮询分配给不同 Agent  
**And** 繁忙 Agent 不接收新任务直到空闲  
**And** Temporal 自动处理任务队列分发  
**And** Agent 处理完一个任务后才接收下一个

### Story 2.8: Agent Docker 镜像

As a **运维工程师**,  
I want **使用 Docker 部署 Agent**,  
So that **简化 Agent 的安装和管理**。

**Acceptance Criteria:**

**Given** Agent 代码已完成  
**When** 构建 Docker 镜像  
**Then** 生成轻量级 Agent 镜像 (<100MB)  
**And** 通过环境变量配置 Temporal 地址和服务器组  
**And** 支持挂载 Docker Socket (用于 Docker 节点)  
**And** 提供 docker run 示例命令  
**And** 镜像推送到 Docker Hub  
**And** 空闲时内存占用 <50MB

### Story 2.9: Agent 配置指南

As a **运维工程师**,  
I want **清晰的 Agent 配置文档**,  
So that **快速在服务器上部署 Agent**。

**Acceptance Criteria:**

**Given** Agent Docker 镜像可用  
**When** 阅读 Agent 配置指南  
**Then** 文档说明所有配置参数  
**And** 提供 Docker 和二进制两种部署方式  
**And** 包含服务器组配置示例  
**And** 说明网络要求 (Temporal 端口 7233)  
**And** 提供故障排查指南  
**And** 提供 systemd service 文件模板

---

## Epic 3: 内置节点库

用户可以使用 10 个内置节点构建实用的工作流,覆盖控制流、Shell 操作、文件传输、HTTP 请求、Docker 管理等常见场景

### Story 3.1: 节点接口设计

As a **开发者**,  
I want **设计统一的节点接口**,  
So that **所有节点遵循一致的实现标准**。

**Acceptance Criteria:**

**Given** 节点系统架构设计  
**When** 定义节点接口  
**Then** 接口包含 Execute(ctx, inputs) (outputs, error) 方法  
**And** 节点元数据包含 name, version, description, input_schema, output_schema  
**And** 支持 JSON Schema 参数验证  
**And** 节点可独立测试  
**And** 提供节点开发模板

### Story 3.2: Shell 命令执行节点

As a **工作流用户**,  
I want **在 Agent 上执行 Shell 命令**,  
So that **运行系统命令和脚本**。

**Acceptance Criteria:**

**Given** Agent 在目标服务器运行  
**When** 工作流 Step 使用 `exec/shell` 节点  
**Then** 在 Agent 服务器执行指定命令  
**And** 支持参数: command, args, env, workdir, timeout  
**And** 捕获 stdout 和 stderr  
**And** 返回退出码  
**And** 超时自动终止进程  
**And** 命令执行失败抛出错误

### Story 3.3: 条件判断节点

As a **工作流用户**,  
I want **根据条件执行不同分支**,  
So that **实现工作流逻辑控制**。

**Acceptance Criteria:**

**Given** 工作流需要条件分支  
**When** Step 使用 `flow/condition` 节点  
**Then** 支持 if 表达式求值  
**And** 表达式结果为 true 执行 then 分支  
**And** 表达式结果为 false 执行 else 分支 (可选)  
**And** 支持比较运算符 (==, !=, >, <, >=, <=)  
**And** 支持逻辑运算符 (&&, ||, !)  
**And** 变量引用正确解析

### Story 3.4: 循环迭代节点

As a **工作流用户**,  
I want **对列表进行迭代执行**,  
So that **批量处理多个项目**。

**Acceptance Criteria:**

**Given** 工作流需要循环处理  
**When** Step 使用 `flow/loop` 节点  
**Then** 支持 for-each 迭代列表  
**And** 每次迭代设置当前项变量  
**And** 支持嵌套 Steps 在循环内执行  
**And** 任一迭代失败可配置是否中断  
**And** 返回所有迭代结果数组

### Story 3.5: 延迟等待节点

As a **工作流用户**,  
I want **在步骤间添加延迟**,  
So that **等待外部系统准备就绪**。

**Acceptance Criteria:**

**Given** 工作流需要等待  
**When** Step 使用 `flow/sleep` 节点  
**Then** 支持秒、分钟、小时为单位的延迟  
**And** 延迟期间工作流状态持久化  
**And** 延迟可被取消  
**And** 支持参数: duration (如 "30s", "5m", "1h")

### Story 3.6: HTTP 请求节点

As a **工作流用户**,  
I want **发送 HTTP 请求**,  
So that **与外部 API 集成**。

**Acceptance Criteria:**

**Given** 工作流需要调用 HTTP API  
**When** Step 使用 `http/request` 节点  
**Then** 支持 GET, POST, PUT, DELETE, PATCH 方法  
**And** 支持自定义 Headers  
**And** 支持 JSON 和 Form 请求体  
**And** 返回响应状态码、Headers、Body  
**And** 支持超时配置  
**And** 4xx/5xx 状态码可配置是否抛出错误

### Story 3.7: 文件传输节点

As a **工作流用户**,  
I want **在服务器间传输文件**,  
So that **分发配置文件或收集日志**。

**Acceptance Criteria:**

**Given** 需要文件上传或下载  
**When** Step 使用 `file/transfer` 节点  
**Then** 支持上传文件到 Agent 服务器  
**And** 支持从 Agent 下载文件  
**And** 支持 SCP/SFTP 协议  
**And** 支持文件权限设置  
**And** 传输进度可追踪  
**And** 传输失败自动重试

### Story 3.8: Docker 命令执行节点

As a **工作流用户**,  
I want **在 Agent 上执行 Docker 命令**,  
So that **管理容器和镜像**。

**Acceptance Criteria:**

**Given** Agent 可以访问 Docker Socket  
**When** Step 使用 `docker/exec` 节点  
**Then** 支持任意 Docker CLI 命令  
**And** 支持参数: command, args  
**And** 捕获命令输出  
**And** Docker 未安装时返回明确错误  
**And** 支持常用命令: run, ps, stop, rm, images, pull

### Story 3.9: Docker Compose Up 节点

As a **工作流用户**,  
I want **启动 Docker Compose 栈**,  
So that **部署多容器应用**。

**Acceptance Criteria:**

**Given** Agent 服务器有 docker-compose 文件  
**When** Step 使用 `docker/compose-up` 节点  
**Then** 执行 docker-compose up  
**And** 支持参数: file, project_name, detach, build  
**And** 等待所有服务启动完成  
**And** 捕获启动日志  
**And** 健康检查验证服务可用

### Story 3.10: Docker Compose Down 节点

As a **工作流用户**,  
I want **停止 Docker Compose 栈**,  
So that **清理部署的应用**。

**Acceptance Criteria:**

**Given** Docker Compose 栈正在运行  
**When** Step 使用 `docker/compose-down` 节点  
**Then** 执行 docker-compose down  
**And** 支持参数: file, project_name, volumes, rmi  
**And** 等待所有容器停止  
**And** 可选删除 volumes 和镜像  
**And** 捕获停止日志

### Story 3.11: 节点参考文档

As a **工作流用户**,  
I want **每个节点的完整参考文档**,  
So that **了解如何使用节点**。

**Acceptance Criteria:**

**Given** 10 个节点已实现  
**When** 查阅节点文档  
**Then** 每个节点有独立文档页面  
**And** 文档包含描述、参数列表、返回值、示例  
**And** 参数说明包含类型、必需性、默认值  
**And** 至少 2 个实际使用示例  
**And** 说明常见错误和解决方法

---

## Epic 4: 节点扩展系统

开发者可以创建自定义节点扩展 Waterflow 能力,通过简单的插件 API (<50 LOC) 注册新节点类型

### Story 4.1: NodeRegistry 和插件加载器实现

As a **开发者**,  
I want **实现节点注册中心和插件加载器**,  
So that **管理所有节点插件**。

**Acceptance Criteria:**

**Given** 节点接口已定义  
**When** Agent 启动时  
**Then** 初始化 NodeRegistry  
**And** 扫描 `/opt/waterflow/plugins/` 目录加载所有 .so 文件  
**And** 调用插件的 Register() 函数自动注册  
**And** 节点按 name 唯一标识  
**And** 提供 ListNodes() 方法查询可用节点  
**And** 提供 GetNode(name) 方法获取节点实例  
**And** 支持运行时热加载新插件 (fsnotify)

### Story 4.2: 节点参数 Schema 验证

As a **系统**,  
I want **验证节点输入参数符合 Schema**,  
So that **避免运行时参数错误**。

**Acceptance Criteria:**

**Given** 节点定义了 input_schema (JSON Schema)  
**When** 工作流执行到该节点  
**Then** 验证输入参数符合 Schema  
**And** 参数类型错误返回明确信息  
**And** 必需参数缺失时报错  
**And** 参数范围验证 (min, max, enum)  
**And** 验证失败中止工作流执行

### Story 4.3: 节点重试策略配置

As a **工作流用户**,  
I want **配置节点的重试策略**,  
So that **临时故障可以自动恢复**。

**Acceptance Criteria:**

**Given** Step 可能因网络等原因失败  
**When** Step 配置了重试策略  
**Then** 支持配置: max_attempts, initial_interval, max_interval, backoff_coefficient  
**And** 失败后按指数退避重试  
**And** 达到最大次数后标记为失败  
**And** 重试日志记录每次尝试  
**And** 不可重试的错误 (如参数错误) 不重试

### Story 4.4: 自定义节点插件开发示例

As a **开发者**,  
I want **创建自己的节点插件**,  
So that **扩展 Waterflow 能力**。

**Acceptance Criteria:**

**Given** 节点插件 API 文档  
**When** 按照示例创建自定义节点  
**Then** 实现 NodeExecutor 接口 (Execute + Metadata)  
**And** 定义节点元数据和 Schema  
**And** 实现 Register() 函数注册到 NodeRegistry  
**And** 编译为 .so 文件: `go build -buildmode=plugin`  
**And** 复制到 `/opt/waterflow/plugins/custom/` 目录  
**And** Agent 自动检测并加载该插件  
**And** 节点在工作流中立即可用  
**And** 示例包含完整测试代码

### Story 4.5: 自定义节点开发指南

As a **开发者**,  
I want **完整的自定义节点开发文档**,  
So that **快速创建节点插件**。

**Acceptance Criteria:**

**Given** 自定义节点 API  
**When** 阅读开发指南  
**Then** 说明节点接口的每个方法  
**And** 提供 3 个不同复杂度的示例节点  
**And** 说明如何定义输入输出 Schema  
**And** 说明如何处理错误和日志  
**And** 提供节点测试最佳实践  
**And** 说明节点打包和分发方式

---

## Epic 5: 高级 DSL 功能

用户可以使用变量引用、条件执行等高级 DSL 功能,编写更灵活和可复用的工作流

### Story 5.1: 变量系统实现

As a **工作流用户**,  
I want **在工作流中定义和使用变量**,  
So that **复用值和参数化工作流**。

**Acceptance Criteria:**

**Given** YAML DSL 解析器  
**When** 工作流定义变量 `vars: {env: production}`  
**Then** 支持通过 `${{ vars.env }}` 引用变量  
**And** 变量替换在执行前完成  
**And** 未定义变量引用时报错  
**And** 支持嵌套对象访问 `${{ vars.db.host }}`  
**And** 支持数组索引 `${{ vars.servers[0] }}`

### Story 5.2: 表达式求值引擎

As a **系统**,  
I want **求值工作流中的表达式**,  
So that **支持动态计算**。

**Acceptance Criteria:**

**Given** 工作流包含表达式  
**When** 表达式求值  
**Then** 支持算术运算 (+, -, *, /, %)  
**And** 支持比较运算 (==, !=, >, <, >=, <=)  
**And** 支持逻辑运算 (&&, ||, !)  
**And** 支持字符串操作 (concat, contains, startsWith, endsWith)  
**And** 支持函数调用 (len, upper, lower, trim)  
**And** 语法错误返回明确位置

### Story 5.3: 条件执行支持

As a **工作流用户**,  
I want **条件化执行 Step**,  
So that **根据运行时状态决定是否执行**。

**Acceptance Criteria:**

**Given** Step 配置了 `if` 条件  
**When** 工作流执行到该 Step  
**Then** 求值 if 表达式  
**And** 表达式为 true 时执行 Step  
**And** 表达式为 false 时跳过 Step  
**And** 支持引用前序 Step 的输出  
**And** 条件求值失败中止工作流

### Story 5.4: Step 输出引用

As a **工作流用户**,  
I want **引用前序 Step 的输出**,  
So that **Step 之间传递数据**。

**Acceptance Criteria:**

**Given** Step 执行完成并有输出  
**When** 后续 Step 引用该输出  
**Then** 支持 `${{ steps.step_id.outputs.key }}` 语法  
**And** 输出值正确传递  
**And** Step 不存在时报错  
**And** 输出字段不存在时报错  
**And** 支持链式引用多个 Step 输出

---

## Epic 6: 客户端工具和 SDK

开发者可以使用 CLI 工具快速验证和测试工作流,或通过 Go SDK 将 Waterflow 集成到 Go 应用中

### Story 6.1: CLI 基础框架

As a **开发者**,  
I want **构建 CLI 工具基础框架**,  
So that **提供命令行接口**。

**Acceptance Criteria:**

**Given** Cobra CLI 框架  
**When** 构建 waterflow CLI  
**Then** 提供根命令和子命令结构  
**And** 支持全局参数 (--server, --api-key, --debug)  
**And** 提供 help 和 version 命令  
**And** 配置文件支持 (~/.waterflow/config.yaml)  
**And** 输出友好的错误信息

### Story 6.2: CLI validate 命令

As a **工作流用户**,  
I want **验证 YAML 工作流语法**,  
So that **提交前发现错误**。

**Acceptance Criteria:**

**Given** YAML 工作流文件  
**When** 执行 `waterflow validate workflow.yaml`  
**Then** 调用 Server 的 `/v1/validate` API  
**And** 语法正确显示 "Valid workflow"  
**And** 语法错误显示具体位置和原因  
**And** 支持验证多个文件  
**And** 返回非零退出码表示验证失败

### Story 6.3: CLI submit 命令

As a **工作流用户**,  
I want **通过 CLI 提交工作流**,  
So that **快速触发执行**。

**Acceptance Criteria:**

**Given** 有效的 YAML 工作流文件  
**When** 执行 `waterflow submit workflow.yaml`  
**Then** 提交工作流到 Server  
**And** 返回工作流 ID  
**And** 支持 `--wait` 参数等待完成  
**And** 支持 `--follow` 参数实时显示日志  
**And** 提交失败显示错误详情

### Story 6.4: CLI status 命令

As a **工作流用户**,  
I want **查询工作流状态**,  
So that **了解执行进度**。

**Acceptance Criteria:**

**Given** 工作流 ID  
**When** 执行 `waterflow status <workflow-id>`  
**Then** 显示工作流状态 (running/completed/failed)  
**And** 显示当前执行进度  
**And** 显示执行时间  
**And** 支持 `--watch` 参数持续监控  
**And** 以表格或 JSON 格式输出

### Story 6.5: CLI logs 命令

As a **工作流用户**,  
I want **查看工作流日志**,  
So that **调试执行问题**。

**Acceptance Criteria:**

**Given** 工作流 ID  
**When** 执行 `waterflow logs <workflow-id>`  
**Then** 显示工作流执行日志  
**And** 支持 `--follow` 参数实时跟踪  
**And** 支持 `--level` 参数过滤日志级别  
**And** 支持 `--step` 参数只显示特定 Step 日志  
**And** 日志带颜色高亮

### Story 6.6: CLI node list 命令

As a **工作流用户**,  
I want **列出所有可用节点**,  
So that **了解可以使用的节点类型**。

**Acceptance Criteria:**

**Given** Server 运行中  
**When** 执行 `waterflow node list`  
**Then** 显示所有注册的节点  
**And** 显示节点名称和描述  
**And** 支持 `--detail` 参数显示完整 Schema  
**And** 支持按类别分组 (控制流/操作/Docker)

### Story 6.7: Go SDK 客户端

As a **Go 开发者**,  
I want **使用 Go SDK 集成 Waterflow**,  
So that **在 Go 应用中编排工作流**。

**Acceptance Criteria:**

**Given** REST API 完整实现  
**When** 使用 Go SDK  
**Then** 提供 Client 结构体封装 API 调用  
**And** 支持 SubmitWorkflow, GetStatus, GetLogs, Cancel 方法  
**And** 使用 context.Context 支持超时和取消  
**And** 返回类型化的错误  
**And** 提供完整 GoDoc 文档  
**And** 提供使用示例代码

### Story 6.8: Go SDK 文档

As a **Go 开发者**,  
I want **Go SDK 的 API 文档**,  
So that **了解如何使用 SDK**。

**Acceptance Criteria:**

**Given** Go SDK 实现  
**When** 查阅 SDK 文档  
**Then** 提供 pkg.go.dev 兼容的文档  
**And** 每个公开方法有注释说明  
**And** 包含快速开始示例  
**And** 包含错误处理最佳实践  
**And** 说明如何配置客户端 (Server URL, API Key)

---

## Epic 7: 工作流模板库

用户可以从预定义模板快速开始,了解 Waterflow 的最佳实践和常见模式

### Story 7.1: 单服务器部署模板

As a **工作流用户**,  
I want **单服务器应用部署模板**,  
So that **快速部署简单应用**。

**Acceptance Criteria:**

**Given** 需要部署单服务器应用  
**When** 使用模板  
**Then** 模板包含: 拉取代码、构建、停止旧版本、启动新版本、健康检查  
**And** 模板参数化 (repo_url, app_name, port)  
**And** 包含完整使用说明  
**And** 包含失败回滚逻辑

### Story 7.2: 多服务器健康检查模板

As a **运维工程师**,  
I want **多服务器批量健康检查模板**,  
So that **定期巡检服务器状态**。

**Acceptance Criteria:**

**Given** 需要检查多台服务器  
**When** 使用模板  
**Then** 模板包含: 并行执行、CPU/内存/磁盘检查、生成报告  
**And** 模板参数化 (server_groups, thresholds)  
**And** 结果聚合到单一报告  
**And** 异常服务器告警

### Story 7.3: 分布式栈部署模板

As a **工作流用户**,  
I want **分布式应用栈部署模板**,  
So that **部署多层架构应用**。

**Acceptance Criteria:**

**Given** 需要部署 Web + DB 分布式应用  
**When** 使用模板  
**Then** 模板包含: 数据库部署、应用部署、依赖顺序控制  
**And** 先部署数据库并健康检查  
**And** 数据库就绪后部署应用  
**And** 参数化 (db_version, app_version, config)  
**And** 包含完整示例和说明

### Story 7.4: 模板 API 端点

As a **开发者**,  
I want **通过 API 访问工作流模板**,  
So that **程序化使用模板**。

**Acceptance Criteria:**

**Given** 内置模板已创建  
**When** 调用 `GET /v1/templates` API  
**Then** 返回所有可用模板列表  
**And** 每个模板包含 name, description, parameters  
**And** `GET /v1/templates/{name}` 返回模板 YAML 内容  
**And** 支持参数说明和示例值

### Story 7.5: 模板文档和示例

As a **工作流用户**,  
I want **模板使用文档**,  
So that **理解如何使用和定制模板**。

**Acceptance Criteria:**

**Given** 3 个内置模板  
**When** 查阅模板文档  
**Then** 每个模板有独立文档页面  
**And** 说明模板用途和适用场景  
**And** 列出所有参数及其说明  
**And** 提供完整使用示例  
**And** 说明如何定制模板

---

## Epic 8: 生产级可靠性

Waterflow 在生产环境中稳定运行,支持故障恢复、性能优化、完善的错误处理

### Story 8.1: 类型化错误处理

As a **开发者**,  
I want **统一的错误处理机制**,  
So that **错误清晰且易于调试**。

**Acceptance Criteria:**

**Given** 系统各模块代码  
**When** 发生错误  
**Then** 使用类型化错误 (ErrInvalidYAML, ErrNodeNotFound, ErrWorkflowTimeout)  
**And** 错误包含上下文信息 (workflow_id, step_name, node_type)  
**And** 错误可序列化为 JSON  
**And** REST API 返回 RFC 7807 Problem Details 格式  
**And** 错误链完整保留

### Story 8.2: 结构化日志系统

As a **系统管理员**,  
I want **结构化的日志输出**,  
So that **日志易于解析和查询**。

**Acceptance Criteria:**

**Given** Server 和 Agent 运行  
**When** 系统运行和处理请求  
**Then** 日志输出 JSON 格式  
**And** 日志包含: timestamp, level, workflow_id, component, message  
**And** 支持日志级别配置 (debug, info, warn, error)  
**And** 敏感信息自动脱敏 (密码、Token)  
**And** 性能关键路径使用 Zap 高性能日志

### Story 8.3: 性能基准测试

As a **开发者**,  
I want **建立性能基准**,  
So that **验证性能指标达标**。

**Acceptance Criteria:**

**Given** Server 和 Agent 实现  
**When** 执行基准测试  
**Then** API 响应时间 P50 < 200ms, P99 < 500ms  
**And** YAML 解析 (1000行) < 100ms  
**And** 支持 100+ 并发 Agent 连接  
**And** 工作流提交吞吐量 > 100/秒  
**And** Agent 空闲内存 < 50MB  
**And** 基准测试可重复运行

### Story 8.4: 压力测试和容错验证

As a **质量工程师**,  
I want **验证系统在压力下的表现**,  
So that **确保生产环境稳定性**。

**Acceptance Criteria:**

**Given** 完整系统部署  
**When** 执行压力测试  
**Then** 1000 个并发工作流稳定执行  
**And** Server 崩溃后自动重启,工作流继续  
**And** Agent 断开重连后任务继续  
**And** Temporal 连接失败时自动重试  
**And** 系统资源占用在合理范围  
**And** 无内存泄漏

### Story 8.5: Prometheus 指标导出

As a **系统管理员**,  
I want **导出 Prometheus 指标**,  
So that **监控系统运行状态**。

**Acceptance Criteria:**

**Given** Server 运行中  
**When** 访问 `/metrics` 端点  
**Then** 导出 Prometheus 格式指标  
**And** 指标包含: 工作流提交数、执行中数量、成功/失败率  
**And** 指标包含: API 请求延迟直方图  
**And** 指标包含: Agent 连接数、健康数量  
**And** 指标包含: 节点执行时长  
**And** 提供 Grafana Dashboard 模板

### Story 8.6: EventHandler 接口实现

As a **系统集成者**,  
I want **通过 EventHandler 接口接收工作流生命周期事件**,  
So that **集成外部监控和通知系统**。

**Acceptance Criteria:**

**Given** Waterflow Server 运行中  
**When** 工作流生命周期事件发生  
**Then** 定义 EventHandler 接口包含三个方法:  
**And** OnWorkflowStart(ctx, workflowID, metadata) - 工作流开始时调用  
**And** OnWorkflowComplete(ctx, workflowID, result) - 工作流成功完成时调用  
**And** OnWorkflowFailed(ctx, workflowID, error) - 工作流失败时调用  
**And** 提供默认 Webhook 实现 (POST JSON 到配置的 URL)  
**And** 提供空实现 (NoOpEventHandler) 作为默认值  
**And** Server 配置支持注入自定义 EventHandler  
**And** 事件发送失败不影响工作流执行  
**And** 提供集成示例: Slack 通知、Prometheus Pushgateway  
**And** 文档说明如何实现自定义 EventHandler

### Story 8.7: LogHandler 接口实现

As a **系统集成者**,  
I want **通过 LogHandler 接口接收工作流执行日志**,  
So that **集成企业日志系统**。

**Acceptance Criteria:**

**Given** Waterflow Server 和 Agent 运行中  
**When** 工作流执行产生日志  
**Then** 定义 LogHandler 接口包含一个方法:  
**And** OnLog(ctx, entry LogEntry) - 接收日志条目  
**And** LogEntry 包含: timestamp, level, workflowID, stepID, nodeType, message  
**And** 提供默认 Stdout 实现 (输出到标准输出)  
**And** 提供 File 实现 (写入日志文件,支持轮转)  
**And** Server 配置支持注入自定义 LogHandler  
**And** 日志发送失败记录到错误日志但不中断执行  
**And** 支持批量发送优化性能 (可选)  
**And** 提供集成示例: Loki、CloudWatch Logs  
**And** 文档说明如何实现自定义 LogHandler

---

## Epic 9: 部署和运维

用户可以通过 Docker Compose 快速部署开发和生产环境

### Story 9.0: Waterflow Server Docker 镜像构建

As a **开发者**,  
I want **构建和发布 Waterflow Server Docker 镜像**,  
So that **简化 Server 部署和分发**。

**Acceptance Criteria:**

**Given** Server 代码已完成  
**When** 构建 Docker 镜像  
**Then** 创建多阶段 Dockerfile (builder + runtime)  
**And** 使用轻量级基础镜像 (alpine 或 distroless)  
**And** 镜像大小 < 50MB (压缩后)  
**And** 支持环境变量配置所有参数  
**And** 暴露端口 8080 (HTTP API)  
**And** 健康检查配置 (HEALTHCHECK 指令)  
**And** 非 root 用户运行提升安全性  
**And** CI/CD 自动构建和推送到 Docker Hub 和 GHCR  
**And** 镜像标签策略: latest, 版本号 (v1.0.0), commit SHA  
**And** 提供 docker run 示例命令  
**And** 启动时间 < 5 秒  
**And** 空闲内存占用 < 100MB

### Story 9.1: Docker Compose 完善

As a **开发者**,  
I want **完善的 Docker Compose 配置**,  
So that **一键启动完整环境**。

**Acceptance Criteria:**

**Given** Docker Compose 文件  
**When** 执行 `docker-compose up`  
**Then** 启动 Temporal Server (PostgreSQL后端)  
**And** 启动 Waterflow Server  
**And** 启动至少 1 个 Agent (示例)  
**And** 配置持久化 volume (数据库数据)  
**And** 提供环境变量配置说明  
**And** 总启动时间 < 10 分钟

### Story 9.2: 配置管理

As a **系统管理员**,  
I want **灵活的配置管理**,  
So that **适配不同环境**。

**Acceptance Criteria:**

**Given** Server 和 Agent 部署  
**When** 配置系统  
**Then** 支持环境变量配置所有参数  
**And** 支持 YAML/TOML 配置文件  
**And** 环境变量优先级高于配置文件  
**And** 提供默认配置适合开发环境  
**And** 配置项文档完整 (类型、默认值、说明)  
**And** 配置错误启动时报告清晰错误

### Story 9.3: 健康检查和就绪探针

As a **系统管理员**,  
I want **健康检查端点**,  
So that **监控服务可用性**。

**Acceptance Criteria:**

**Given** Server 运行中  
**When** 请求健康检查  
**Then** `/health` 端点检查进程存活  
**And** `/ready` 端点检查 Temporal 连接、数据库连接  
**And** 服务不可用时返回 503  
**And** Docker Healthcheck 使用 `/health` 端点  
**And** Readiness 检查通过 `/ready` 端点验证

### Story 9.5: 部署文档

As a **系统管理员**,  
I want **完整的部署文档**,  
So that **正确部署和配置系统**。

**Acceptance Criteria:**

**Given** 部署工具和配置  
**When** 查阅部署文档  
**Then** 说明 Docker Compose 部署步骤  
**And** 说明二进制独立部署步骤  
**And** 列出所有配置参数和说明  
**And** 提供生产环境最佳实践  
**And** 说明如何备份和恢复  
**And** 说明如何升级版本

---

## Epic 10: 安全和认证

Waterflow 支持 API 认证,保护敏感凭证,提供安全的通信机制

### Story 10.1: API Key 认证

As a **系统管理员**,  
I want **API Key 认证机制**,  
So that **保护 API 访问**。

**Acceptance Criteria:**

**Given** Server 配置启用认证  
**When** 客户端调用 API  
**Then** 请求必须包含 `Authorization: Bearer <api-key>` Header  
**And** 无效 API Key 返回 401  
**And** API Key 通过配置文件或环境变量设置  
**And** 支持多个 API Key (用于不同应用)  
**And** 健康检查端点不需要认证

### Story 10.2: HTTPS/TLS 支持

As a **系统管理员**,  
I want **HTTPS 加密通信**,  
So that **保护传输中的数据**。

**Acceptance Criteria:**

**Given** Server 配置 TLS  
**When** 客户端连接  
**Then** 支持 HTTPS 协议  
**And** 配置 TLS 证书和密钥  
**And** 支持自签名证书 (开发环境)  
**And** 强制最低 TLS 1.2  
**And** HTTP 自动重定向到 HTTPS (可选)

### Story 10.3: SecretProvider 接口

As a **开发者**,  
I want **SecretProvider 接口**,  
So that **工作流安全获取密钥**。

**Acceptance Criteria:**

**Given** 工作流需要访问密钥 (如 SSH 密码)  
**When** 节点请求密钥  
**Then** 通过 SecretProvider 接口获取  
**And** 接口定义: GetSecret(ctx, key) (value, error)  
**And** Server 不存储密钥  
**And** 提供默认实现 (环境变量)  
**And** 提供 Vault 集成示例  
**And** 密钥不记录到日志

### Story 10.4: 审计日志

As a **安全审计员**,  
I want **审计日志记录所有操作**,  
So that **追踪系统使用情况**。

**Acceptance Criteria:**

**Given** Server 运行并处理请求  
**When** 用户执行操作  
**Then** 记录审计日志: 时间、用户、操作、资源、结果  
**And** 审计日志包含: 工作流提交、取消、查询  
**And** 审计日志独立于应用日志  
**And** 审计日志不可篡改 (append-only)  
**And** 支持导出审计日志

### Story 10.5: 安全最佳实践文档

As a **系统管理员**,  
I want **安全配置指南**,  
So that **安全地部署和运维**。

**Acceptance Criteria:**

**Given** 安全功能实现  
**When** 查阅安全文档  
**Then** 说明如何启用 API 认证  
**And** 说明如何配置 HTTPS/TLS  
**And** 说明如何集成密钥管理服务  
**And** 说明最小权限原则 (Agent 权限)  
**And** 说明网络安全配置 (防火墙规则)  
**And** 提供安全检查清单

---

## Epic 11: 完整文档体系

用户可以通过完善的文档自助完成从入门到高级使用的全部流程,无需人工支持

### Story 11.1: 快速开始指南

As a **新用户**,  
I want **30 分钟快速开始教程**,  
So that **快速体验 Waterflow**。

**Acceptance Criteria:**

**Given** 新用户没有 Waterflow 经验  
**When** 按照快速开始指南操作  
**Then** 30 分钟内完成: 部署 Server、部署 Agent、运行首个工作流  
**And** 每一步有清晰的命令示例  
**And** 包含预期输出和验证步骤  
**And** 包含常见问题解决方案  
**And** 引导用户到下一步学习资源

### Story 11.2: REST API 规范文档

As a **集成开发者**,  
I want **OpenAPI 3.0 REST API 文档**,  
So that **了解所有 API 端点**。

**Acceptance Criteria:**

**Given** REST API 实现  
**When** 访问 API 文档  
**Then** 提供 OpenAPI 3.0 规范文件  
**And** Swagger UI 交互式文档可访问  
**And** 每个端点有请求/响应示例  
**And** 说明所有参数和返回值  
**And** 说明错误码和错误响应格式  
**And** 提供 curl 示例

### Story 11.3: YAML DSL 语法参考

As a **工作流用户**,  
I want **完整的 YAML DSL 语法文档**,  
So that **编写正确的工作流**。

**Acceptance Criteria:**

**Given** DSL 语法定义  
**When** 查阅语法文档  
**Then** 说明所有顶层字段 (name, vars, jobs)  
**And** 说明 Job 结构 (runs-on, steps, timeout)  
**And** 说明 Step 结构 (uses, with, if, retry)  
**And** 说明变量引用语法  
**And** 说明表达式语法和函数  
**And** 提供完整工作流示例  
**And** 说明常见模式和最佳实践

### Story 11.4: 故障排查指南

As a **用户**,  
I want **故障排查文档**,  
So that **自助解决问题**。

**Acceptance Criteria:**

**Given** 用户遇到问题  
**When** 查阅故障排查指南  
**Then** 列出常见问题和解决方案  
**And** 说明如何查看 Server 日志  
**And** 说明如何查看 Agent 日志  
**And** 说明如何调试工作流失败  
**And** 说明如何排查 Temporal 连接问题  
**And** 提供诊断命令清单

### Story 11.5: 工作流示例库

As a **工作流用户**,  
I want **丰富的工作流示例**,  
So that **学习最佳实践**。

**Acceptance Criteria:**

**Given** 各种使用场景  
**When** 查阅示例库  
**Then** 提供至少 10 个不同场景的示例  
**And** 每个示例有完整 YAML 和说明  
**And** 示例包含: 部署、健康检查、备份、测试、通知  
**And** 示例展示不同节点用法  
**And** 示例展示高级 DSL 功能

---

## Epic 12: 质量保证和发布

Waterflow 通过全面测试验证,提供稳定的发布版本和多种分发渠道

### Story 12.1: 单元测试框架

As a **开发者**,  
I want **完善的单元测试**,  
So that **验证代码正确性**。

**Acceptance Criteria:**

**Given** 所有模块代码  
**When** 执行测试  
**Then** 单元测试覆盖率 > 80%  
**And** 每个节点有独立测试  
**And** DSL 解析器有完整测试  
**And** API Handler 有测试  
**And** 使用 Testify 断言库  
**And** 测试在 CI 中自动运行

### Story 12.2: 集成测试

As a **质量工程师**,  
I want **端到端集成测试**,  
So that **验证系统整体功能**。

**Acceptance Criteria:**

**Given** 完整系统部署  
**When** 执行集成测试  
**Then** 测试工作流提交到执行完成全流程  
**And** 测试 Agent 注册和任务执行  
**And** 测试多节点工作流  
**And** 测试故障重试机制  
**And** 集成测试可在 CI 中运行  
**And** 测试环境自动启停

### Story 12.3: 验收测试场景

As a **产品经理**,  
I want **验收测试覆盖关键场景**,  
So that **确保 MVP 目标达成**。

**Acceptance Criteria:**

**Given** PRD 定义的验收场景  
**When** 执行验收测试  
**Then** 场景1: 多服务器健康检查工作流通过  
**And** 场景2: 分布式应用部署工作流通过  
**And** 每个场景有自动化测试脚本  
**And** 测试结果生成报告  
**And** 所有验收测试通过才能发布

### Story 12.4: GitHub Actions CI/CD

As a **开发者**,  
I want **自动化 CI/CD 流程**,  
So that **保证代码质量和自动发布**。

**Acceptance Criteria:**

**Given** GitHub Actions 配置  
**When** 提交代码或创建 Release  
**Then** 自动运行: lint、单元测试、集成测试  
**And** 构建 Docker 镜像并推送到 Registry  
**And** 编译多平台二进制 (Linux/MacOS/Windows)  
**And** 创建 GitHub Release 附带二进制  
**And** 代码质量检查失败时阻止合并  
**And** Tag 推送时自动发布版本

### Story 12.5: 发布和分发

As a **用户**,  
I want **多种方式获取 Waterflow**,  
So that **选择最适合的安装方式**。

**Acceptance Criteria:**

**Given** 新版本发布  
**When** 用户安装 Waterflow  
**Then** Docker Hub 提供最新镜像  
**And** GitHub Releases 提供二进制下载  
**And** Go modules 可通过 `go get` 安装  
**And** 提供版本号和 Changelog  
**And** 提供 checksum 文件验证完整性  
**And** 所有分发渠道版本同步

---

## 总结

**共 12 个 Epics, 80 User Stories**

所有需求已完整分解为可执行的 Stories,每个 Story 都包含清晰的验收标准。Stories 按 Epic 组织,体现用户价值和技术实现的平衡。

### 最近更新 (2025-12-16)

**补充的Stories (基于实施就绪性评估):**

1. **Epic 2 - Story 2.2**: 增强ServerGroupProvider接口实现的验收标准
   - 明确接口定义和默认实现
   - 添加CMDB集成示例

2. **Epic 8 - Story 8.6**: EventHandler接口实现 ✨ **新增**
   - 支持工作流生命周期事件集成
   - 提供Webhook默认实现和集成示例

3. **Epic 8 - Story 8.7**: LogHandler接口实现 ✨ **新增**
   - 支持日志系统集成
   - 提供Stdout和File默认实现

4. **Epic 9 - Story 9.0**: Waterflow Server Docker镜像构建 ✨ **新增**
   - 完善Docker镜像构建和发布流程
   - 补充Server镜像(原本只有Agent镜像)

**FR覆盖率更新:**
- 原覆盖率: 85.7% (18/21)
- 当前覆盖率: 100% (21/21) ✅
- 补充的FR: FR15 (ServerGroupProvider), FR17 (EventHandler), FR18 (LogHandler), FR19 (Server镜像)

**Story总数更新:**
- Epic 2: 9 → 9 Stories (Story 2.2增强)
- Epic 8: 5 → 7 Stories (新增8.6, 8.7)
- Epic 9: 4 → 5 Stories (新增9.0)
- **总计: 76 → 80 Stories**
