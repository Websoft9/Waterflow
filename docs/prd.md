---
stepsCompleted: [1, 2, 3, 4, 5, 6, 7, 8, 9]
inputDocuments: []
documentCounts:
  briefs: 0
  research: 0
  brainstorming: 0
  projectDocs: 0
workflowType: 'prd'
lastStep: 9
project_name: 'Waterflow'
user_name: 'Websoft9'
date: '2025-12-16'
revision: 'Architecture design complete - Event Sourcing, Single-node execution, Plugin system'
version: '1.2'
---

# Product Requirements Document - Waterflow

**产品:** Waterflow - 声明式工作流编排引擎  
**文档类型:** 产品需求文档 (PRD)  
**作者:** Websoft9 产品团队  
**日期:** 2025-12-16  
**版本:** 1.2  
**状态:** 架构设计完成

## 执行摘要

### 产品概述

Waterflow 是一个声明式工作流编排引擎，提供企业级分布式任务执行能力。通过 YAML DSL 定义工作流，结合生产级执行引擎，实现跨服务器的可靠任务编排。

**产品形态:** 声明式工作流编排服务（基于 Temporal，提供 DSL 和 Agent 模式）  
**技术架构:** Waterflow Server + 分布式 Agent + REST API/SDK  
**目标用户:** 平台开发者、运维工程师、DevOps 工程师  
**集成方式:** REST API（主要）、多语言 SDK（辅助）、CLI 工具（开发测试）

### 问题陈述

**市场空白:**
- **传统工作流引擎** 需要编写大量代码，学习曲线陡峭
- **CI/CD 平台** 专为部署管道设计，无法处理通用业务流程
- **低代码工具** 缺乏版本控制和代码化管理能力
- **脚本自动化** 不提供持久化、重试、状态管理等企业级能力

**用户痛点:**
- 运维团队需要跨多台服务器的任务协调，但现有工具难以实现
- 长时运行的任务需要可靠性保证，进程重启不能影响执行
- 复杂的业务流程需要简单的定义方式，但不想牺牲灵活性

### 解决方案架构

**核心价值主张:**  
Waterflow 是基于 Temporal 构建的声明式工作流编排服务，通过 YAML DSL 和分布式 Agent 模式，让用户以简单的方式实现跨服务器的可靠任务编排，无需了解底层 Temporal 的复杂性。

**技术能力:**

| 能力 | 实现方式 | 技术价值 |
|------|----------|----------|
| **持久化执行** | Event Sourcing 状态管理 (Temporal Event History) | 工作流状态 100% 持久化,进程重启后完全恢复 |
| **声明式 DSL** | YAML DSL + 表达式系统 (${{ }}) | 低门槛,支持版本控制和动态表达式 |
| **分布式调度** | Task Queue 直接映射 (runs-on → queue) | 跨服务器任务编排,Temporal 原生负载均衡 |
| **可扩展性** | 插件化节点系统 (Go Plugin .so 文件) | 热加载自定义节点,无需重启 Agent |
| **可观测性** | Event History + 结构化日志 | 完整执行历史追踪,支持时间旅行查询 |
| **精确控制** | 单节点执行模式 (每个 Step = 1 Activity) | 独立超时/重试配置,精确故障定位 |

**架构设计:**

```
┌─────────────────────────────────────────────────────────┐
│  用户应用/CLI (任何语言)                                 │
│  └─ REST API / SDK 客户端                               │
└─────────────────────────────────────────────────────────┘
                      ↓ HTTP/HTTPS
┌─────────────────────────────────────────────────────────┐
│  Waterflow Server (无状态服务)                          │
│  ├─ REST API 服务                                       │
│  ├─ DSL 解析器 (YAML → Workflow 参数)                  │
│  ├─ 表达式引擎 (${{ }} 求值)                           │
│  ├─ Schema 验证器                                       │
│  ├─ 集成接口                                            │
│  │  ├─ ServerGroupProvider (服务器组查询)              │
│  │  ├─ SecretProvider (运行时密钥注入)                 │
│  │  ├─ EventHandler (工作流事件通知)                   │
│  │  └─ LogHandler (日志输出)                           │
│  └─ Temporal Client (提交 Workflow)                    │
└─────────────────────────────────────────────────────────┘
                      ↓ gRPC
┌─────────────────────────────────────────────────────────┐
│  Temporal Server (工作流引擎)                           │
│  ├─ Event Sourcing 状态管理 (Event History 存储)       │
│  ├─ Task Queue 路由 (runs-on 直接映射)                 │
│  ├─ 持久化执行 + 自动重试                               │
│  └─ Worker 健康检查                                     │
└─────────────────────────────────────────────────────────┘
                      ↓ gRPC (Task Queue)
┌─────────────────────────────────────────────────────────┐
│  Waterflow Agents (分布式 Worker)                       │
│  ├─ Plugin Manager (扫描 .so 文件,热加载)              │
│  ├─ Node Registry (注册节点实现)                        │
│  ├─ Activity Executor (执行单节点任务)                  │
│  ├─ Heartbeat Reporter (心跳上报)                       │
│  └─ 节点实现 (.so 插件)                                 │
│     ├─ checkout.so, run.so, docker-exec.so...          │
│     └─ 用户自定义节点 (custom-*.so)                     │
└─────────────────────────────────────────────────────────┘
                      ↓ SSH/Docker/HTTP
┌─────────────────────────────────────────────────────────┐
│  Target Infrastructure (目标基础设施)                   │
│  └─ Linux Servers, Docker Containers, APIs              │
└─────────────────────────────────────────────────────────┘
```

**架构关键特征:**

1. **Server 完全无状态** - 所有工作流状态存储在 Temporal Event History
2. **单节点执行模式** - 每个 Step = 1 个 Temporal Activity 调用
3. **插件化节点系统** - 所有节点编译为 .so 文件,支持热加载
4. **Task Queue 直接映射** - runs-on 字段直接映射到 Task Queue 名称
5. **Event Sourcing** - Temporal 通过 Event History 重建完整执行状态

### 竞争差异化

**市场定位:**

| 类别 | 代表产品 | Waterflow 优势 |
|------|----------|----------------|
| **传统工作流引擎** | Temporal、Cadence | YAML DSL 降低使用门槛 |
| **CI/CD 平台** | GitHub Actions、GitLab CI | 通用工作流编排，不限于部署 |
| **低代码工具** | n8n、Zapier | 支持 Git、代码化管理 |
| **脚本工具** | Ansible、Shell | 持久化、自动重试、状态管理 |

**核心优势:**

1. **Event Sourcing 架构 + 单节点执行模式**
   - 所有执行状态存储在 Temporal Event History,100% 持久化
   - Server 完全无状态,崩溃后可从 Temporal 重建状态
   - 每个 Step 独立配置超时/重试,精确故障控制
   - 支持时间旅行查询和完整审计追踪

2. **插件化节点系统 + 热加载**
   - 所有节点编译为 Go Plugin (.so 文件)
   - Agent 启动时自动扫描和加载插件
   - 支持运行时热加载新节点,无需重启
   - 用户可在 <50 LOC 内实现自定义节点

3. **Task Queue 直接映射 + 分布式 Agent**
   - runs-on 字段直接映射到 Temporal Task Queue
   - Temporal 原生负载均衡,零额外配置
   - Agent 支持多个 Task Queue,灵活部署
   - 真正的多服务器任务编排,故障隔离

4. **声明式 DSL + 表达式系统**
   - 简单的 YAML 语法,隐藏 Temporal 复杂性
   - ${{ }} 表达式系统,GitHub Actions 兼容
   - 支持 Git 版本控制和 Code Review
   - 安全沙箱执行环境

**关键亮点:**  
用 YAML 配置实现企业级工作流（数小时长任务、自动重试、状态持久化），无需编写复杂代码。

## 项目分类

**技术类型:** 工作流引擎 / 开发者工具  
**领域:** DevOps 运维自动化 / 分布式任务编排  
**复杂度:** Medium - 涉及分布式执行、状态管理、容错机制  
**项目上下文:** Greenfield - 全新项目  
**主要用户:** 运维工程师、DevOps 工程师  
**次要用户:** 开发者（SDK 集成）

---

## 成功标准

### 用户成功指标

**工作流用户成功（主要用户）:**

| 目标 | 关键结果 | 目标值 | 衡量方式 |
|------|----------|---------|----------|
| **快速上手** | 首次工作流执行时间 | ≤30分钟 | 用户测试 |
| **易用性** | 创建复杂工作流的时间 | ≤1小时 | 任务完成率 |
| **自助服务** | 仅通过文档完成任务率 | ≥80% | 支持工单量 |

**开发者成功（SDK 集成）:**

| 目标 | 关键结果 | 目标值 | 衡量方式 |
|------|----------|---------|----------|
| **集成简单** | SDK 集成代码行数 | ≤100 LOC | 代码分析 |
| **模板覆盖率** | 内置节点可解决的任务 | ≥80% | 节点使用统计 |
| **问题排查** | 用户能独立调试失败 | ≥70% | 支持升级率 |

**用户体验里程碑:**
1. **验证:** 执行示例工作流并查看执行日志  
2. **创建:** 编写第一个工作流并成功运行
3. **部署:** 将 Agent 部署到远程服务器
4. **精通:** 创建自定义节点

### 业务成功指标

**MVP 验证目标:**

**目标 1: 产品可用性**
- **KR1:** 完成 3+ 真实场景的工作流验证
- **KR2:** ≥2 个生产环境部署
- **KR3:** 用户 NPS 评分 ≥40

**目标 2: 技术可行性**  
- **KR1:** 生产环境系统运行时间 ≥99.5%
- **KR2:** 零工作流状态损坏事件
- **KR3:** P95 工作流启动延迟 ≤500ms

**目标 3: 生态基础**
- **KR1:** 实现并文档化 10 个核心节点  
- **KR2:** ≥3 个可复用的工作流模板
- **KR3:** 自定义节点开发指南经用户验证

**领先指标（MVP 阶段）:**
- 每周活跃用户数
- 工作流执行总量
- 节点使用分布
- 文档访问量 vs 支持请求比率

**滞后指标（Post-MVP）:**
- 用户留存率（12个月）  
- 工作流执行可靠性（成功率）
- 生产问题平均解决时间

### 技术成功标准

**架构质量属性:**

| 质量属性 | 要求 | 验收标准 |
|----------|------|----------|
| **持久性** | 工作流状态在进程故障后幸存 | 崩溃测试中 100% 状态恢复 |
| **可靠性** | 自动故障处理 | 每个节点可配置重试策略 |
| **可观测性** | 执行可追溯性 | 每次工作流运行的完整事件日志 |
| **可扩展性** | 多服务器协调 | 支持 ≥100 个并发 Agent 连接 |
| **可扩展性** | 自定义节点开发 | 插件 API 每个节点 <50 LOC |
| **可移植性** | 跨平台部署 | 支持 Linux、macOS、Windows |

**系统能力:**

1. **长时运行执行**
   - 工作流可执行数小时/数天而不丢失状态
   - 进程重启不中断工作流进度
   - 状态变更的一次性执行语义

2. **容错能力**
   - 指数退避的自动重试
   - 每步骤可配置超时和重试策略
   - 部分故障隔离（单服务器故障不中止整个工作流）

3. **分布式协调**  
   - 跨服务器任务依赖
   - Agent 健康监控和任务路由
   - 多个服务器组上的并发执行

### 验收测试场景

**场景 1: 多服务器健康检查**  
在 3 台服务器上并发执行脚本，实时显示进度，结果聚合到单一报告。

**场景 2: 分布式应用部署**  
在不同服务器组部署 Web 应用和数据库，按依赖顺序执行，健康检查验证，失败自动重试和回滚

---

## 产品范围

### 系统边界

**Waterflow 核心能力:**

| 组件 | 功能 | 接口 |
|------|------|------|
| **REST API 服务** | HTTP API 提供工作流管理 | HTTP/JSON (OpenAPI 3.0) |
| **DSL 引擎** | YAML 工作流解析与验证 | 内部 API |
| **工作流运行时** | 持久化工作流执行 (Temporal) | 内部集成 |
| **节点注册表** | 内置节点 + 自定义节点管理 | 插件 API |
| **Agent 调度** | 任务分发到目标服务器 | gRPC/内部协议 |
| **执行监控** | 结构化日志和事件输出 | EventHandler、LogHandler |

**依赖注入接口:**

| 接口 | 职责 | 必需性 | 典型实现示例 |
|------|------|--------|--------|
| **ServerGroupProvider** | 提供服务器组和 Agent 清单 | 必需 | CMDB、Ansible Inventory、静态配置文件 |
| **SecretProvider** | 提供工作流所需密钥（运行时注入） | 必需 | HashiCorp Vault、AWS KMS、环境变量 |
| **EventHandler** | 接收工作流事件（开始、完成、失败） | 可选 | Webhook、消息队列、日志系统 |
| **LogHandler** | 接收工作流执行日志 | 可选 | ELK Stack、Loki、CloudWatch |

**接口设计原则:**
- 接口简单：每个接口 ≤3 个方法
- 职责单一：每个接口只做一件事
- 易于实现：提供默认实现和示例代码

**职责边界:**

**Waterflow 核心职责（引擎能力）:**
- ✅ 工作流解析和执行
- ✅ 分布式任务调度
- ✅ 状态持久化和恢复 (Event Sourcing 模式,存储于 Temporal Event History)
- ✅ 节点编排和重试 (单节点执行模式: 每个 Step = 1 Activity)
- ✅ Agent 生命周期管理
- ✅ 插件化节点系统 (所有节点为 .so 插件,支持热加载)

**外部系统职责（通过接口集成）:**
- ❌ 服务器清单管理 → 通过 ServerGroupProvider 接口获取
- ❌ 密钥存储和管理 → 通过 SecretProvider 接口获取
- ❌ 用户认证和授权 → 不在 Waterflow 职责范围
- ❌ Agent 自动部署 → 用户使用 Ansible/Terraform 等工具
- ❌ 日志存储和展示 → 通过 LogHandler 接口输出

**设计原则:**
Waterflow 采用**服务化架构**:
- **完整服务**: Waterflow Server 提供完整的工作流编排能力
- **封装 Temporal**: Temporal 作为底层运行时,用户无需感知
- **DSL 抽象**: YAML DSL 隐藏 Temporal 复杂性,降低使用门槛
- **Agent 模式**: 分布式 Agent 实现跨服务器任务编排
- **接口集成**: 服务器组、密钥等通过接口集成外部系统
- **职责专注**: 专注工作流编排核心能力,不重复造轮子

**集成示例:**

```go
// 实现 ServerGroupProvider 接口
type MyServerGroupProvider struct {
    db *sql.DB
}

func (p *MyServerGroupProvider) GetServers(groupName string) ([]ServerInfo, error) {
    return queryServers(p.db, groupName)
}

// 初始化 Waterflow
engine := waterflow.New(waterflow.Config{
    ServerGroupProvider: &MyServerGroupProvider{db: myDB},
})
```

### MVP 定义

**MVP 目标:**  
验证 Waterflow SDK 能够被轻松嵌入到应用程序中，通过 YAML 工作流实现跨服务器的可靠任务编排，并提供生产级执行保证。

**MVP 策略: Server + API 优先**
- **主要交付:** Waterflow Server + REST API（核心服务）
- **辅助工具:** CLI（开发测试）、Go SDK（便捷集成）
- **内部依赖:** Temporal Server（打包部署或独立部署）

**理由:**
1. 用户无需了解 Temporal，Waterflow 提供完整解决方案
2. REST API 支持多语言集成（Python, Node.js, Go, Java 等）
3. 降低用户部署和维护成本（Waterflow 管理 Temporal）
4. 更符合工作流服务的产品定位（对标 Airflow, Jenkins）

**必须具备（核心交付物）:**

**1. Waterflow Server（核心服务）**
- **核心组件:**
  - REST API 服务（HTTP/JSON）
  - DSL 解析引擎（YAML → Temporal Workflow）
  - 节点注册表（内置 + 自定义节点）
  - Temporal SDK 集成层
  - 配置管理（服务器组、密钥、事件处理）

- **主要 API 端点:**
  ```
  POST   /v1/workflows          # 提交工作流
  GET    /v1/workflows/:id      # 查询状态
  DELETE /v1/workflows/:id      # 取消执行
  GET    /v1/workflows/:id/logs # 获取日志
  POST   /v1/validate           # 验证 YAML
  GET    /v1/nodes              # 列出节点
  ```

- **部署方式:**
  - Docker Compose（一键部署，内置 Temporal）
  - 二进制独立部署

- **集成示例:**
  ```bash
  # 提交工作流
  curl -X POST http://waterflow:8080/v1/workflows \
    -H "Content-Type: application/json" \
    -d '{"workflow": "<yaml-content>"}'
  ```

**2. Go SDK（便捷集成）**
- **核心功能:** 封装 REST API 调用，提供 Go 语言惯用接口
- **主要 API:**
  ```go
  client := waterflow.NewClient("http://waterflow:8080")
  workflowID, _ := client.Submit(ctx, yamlContent)
  status, _ := client.Query(ctx, workflowID)
  ```
- **设计原则:**
  - 薄客户端（仅封装 HTTP 调用）
  - 上下文感知（context.Context）
  - 错误处理友好

**3. Agent 执行器**
- **部署方式:** Docker 容器
- **核心功能:**
  - 自动注册到服务器组
  - 执行工作流节点任务
  - 健康状态上报
  - 网络故障自动重连
- **交付:** Docker 镜像 + 安装脚本

**4. 节点库（10 个内置节点）**

*控制流（3 个节点）:*
- `flow/condition` - 条件分支（if/else）
- `flow/loop` - 集合迭代
- `flow/sleep` - 定时延迟

*操作（4 个节点）:*
- `exec/shell` - Shell 命令执行
- `exec/script` - 脚本文件执行
- `file/transfer` - 文件上传/下载
- `http/request` - HTTP 客户端

*Docker 管理（3 个节点）:*
- `docker/exec` - Docker CLI 命令
- `docker/compose-up` - 启动 Docker Compose 栈
- `docker/compose-down` - 停止 Docker Compose 栈

**5. CLI 工具（开发辅助）**
- `waterflow validate <workflow.yaml>` - 语法验证（调用 API）
- `waterflow submit <workflow.yaml>` - 提交工作流
- `waterflow status <workflow-id>` - 查询状态
- `waterflow logs <workflow-id>` - 查看日志
- `waterflow nodes list` - 查看可用节点

**6. 工作流模板（3 个示例）**
- 单服务器应用部署
- 多服务器健康检查
- 分布式栈部署（应用 + 数据库）

**7. 文档**
- 快速开始指南（30 分钟入门）
- Waterflow Server 部署指南
- REST API 参考（OpenAPI 规范）
- Go SDK API 参考
- DSL 语法规范
- 表达式系统文档 (${{ }} 语法)
- 节点参考（全部 10 个节点）
- 自定义节点开发指南 (插件 SDK)
- 架构决策记录 (ADR 目录)
  - ADR-0001: 使用 Temporal 作为工作流引擎
  - ADR-0002: 单节点执行模式
  - ADR-0003: 插件化节点系统
  - ADR-0004: YAML DSL 语法设计
  - ADR-0005: 表达式系统语法
  - ADR-0006: Task Queue 路由机制
- 核心架构概念
  - Event Sourcing 状态管理
  - 单节点执行模式
  - 插件化节点系统
  - Task Queue 直接映射
- 工作流模板示例

**应该有（Post-MVP 优先级）:**
- **多语言 SDK:** Python SDK、Node.js SDK（封装 REST API）
- **Web UI:** 工作流执行监控、日志查看、节点管理
- **高级 DSL 功能:** 变量、表达式、子工作流
- **扩展节点库:** 10+ 个额外节点（云提供商、数据库操作）
- **可观测性:** Prometheus metrics 导出、分布式追踪

**可以有（未来考虑）:**
- 可视化工作流编辑器
- 节点市场
- 多语言节点支持（Python、JavaScript）
- 社区贡献的工作流模板库

**不会有（MVP 明确排除）:**
- ✗ 内置服务器清单管理（通过 ServerGroupProvider 接口集成外部系统）
- ✗ 内置密钥存储系统（通过 SecretProvider 接口集成 Vault/KMS）
- ✗ 用户认证和授权（通过反向代理或 API Gateway 实现）
- ✗ Agent 自动部署工具（用户使用 Ansible/Terraform）
- ✗ 内置监控仪表板（通过 EventHandler 集成 Prometheus/Grafana）
- ✗ 可视化工作流编辑器（Post-MVP）
- ✗ Webhook/cron 触发器（Post-MVP）

**MVP 成功标准:**
- [ ] **≤10 分钟**完成 Waterflow Server 部署（Docker Compose）
- [ ] 首个工作流在 **≤30 分钟**内运行成功（通过 API 或 CLI）
- [ ] 两个验收测试场景均通过（健康检查 + 分布式部署）
- [ ] REST API 符合 OpenAPI 3.0 规范，支持 Swagger UI
- [ ] 依赖注入接口设计简洁（每个接口 ≤3 个方法）
- [ ] 文档支持自助服务入门（无需人工支持）
- [ ] **Event Sourcing 验证:** 系统处理进程崩溃时无工作流状态丢失
- [ ] **单节点执行验证:** 每个 Step 独立配置超时/重试,Temporal UI 清晰展示
- [ ] **插件系统验证:** 自定义节点热加载成功,无需重启 Agent
- [ ] **Task Queue 验证:** runs-on 直接映射到 Task Queue,跨服务器路由正确
- [ ] **核心指标:** 部署到提交首个工作流 <15 分钟

### 产品路线图

**阶段 1: MVP（第 1-3 月）**
- Waterflow Server (核心服务 + REST API)
- Temporal Server 集成和管理
- 覆盖基本操作的 10 个内置节点
- 通过 Docker 部署 Agent
- Go SDK (便捷客户端封装)
- CLI 工具（开发测试）
- 自助服务入门文档 + API 参考
- Docker Compose 一键部署方案

**阶段 2: 生产就绪（第 4-6 月）**
- 多语言 SDK (Python, Node.js, Java)
- Web UI (工作流监控、日志查看)
- 扩展节点库（20+ 个节点）:
  - 云提供商集成（AWS EC2、S3）
  - 数据库操作（MySQL、PostgreSQL、Redis）
- 高级 DSL 功能（变量、表达式、子工作流）
- 性能优化（并行执行、缓存）
- 工作流模板库（10+ 个生产模板）
- 监控集成（Prometheus 指标导出）

**阶段 3: 生态增长（第 7-12 月）**
- 带全面文档的节点开发 SDK
- 社区贡献流程（GitHub 工作流）
- 多语言节点支持（通过 gRPC 的 Python、JavaScript）
- 工作流调试工具（断点、单步执行）
- 迁移工具（脚本转工作流转换器）
- 视频教程和认证计划

**阶段 4: 企业特性（第 13-18 月）**
- 可视化工作流编辑器（基于 Web）
- 带验证/认证的节点市场
- 高级调度（cron、webhook 触发器）
- 工作流版本控制和回滚
- A/B 测试和金丝雀部署支持
- 企业支持级别和 SLA
- 合规认证（SOC2、ISO27001）

**长期愿景（18 个月+）:**
- AI 辅助的自然语言工作流生成
- 跨平台工作流可移植性（导入/导出标准）
- 联邦工作流执行（多区域协调）
- 行业特定模板库（医疗、金融、零售）
- 与平台供应商的战略合作伙伴关系

### 产品愿景

**北极星指标:**  
每月跨所有集成平台的生产工作流执行数量

**3 年愿景:**
Waterflow 成为运维自动化、DevOps 工具和分布式系统管理领域的标准工作流编排服务。

**成功指标:**
- ≥ 50 个生产环境部署实例
- ≥ 100 个社区贡献的市场节点
- ≥ 100 万次/月的工作流执行（所有实例总和）
- 作为 YAML DSL + Temporal 的最佳实践获得行业认可

**战略目标:**
1. **普及部署:** 成为需要多服务器任务编排的首选服务
2. **繁荣生态:** 培育拥有 >100 个贡献者的活跃节点开发者社区
3. **生产级别:** 实现企业级可靠性，工作流状态损坏率 <0.01%
4. **跨领域采用:** 从 DevOps 扩展到通用业务流程自动化

---

## 用户旅程

### 旅程 1：运维工程师 - 从脚本到工作流

**痛点:** 运维工程师管理 80 台服务器，维护十几个 Bash 脚本，部署时需要在多个终端窗口切换，脚本失败需要手动清理重来。

**解决:** 使用 Waterflow YAML 定义部署流程，10 分钟完成 Server 和 Agent 部署。系统自动协调多服务器执行，失败自动重试，执行日志清晰可查。

**成果:** 将所有运维任务转换为工作流模板，新员工通过 YAML 文件即可理解和执行部署任务。

### 旅程 2：DevOps 团队负责人 - 效率提升

**痛点:** DevOps 团队为 30+ 个微服务提供支持，每个微服务部署流程不同，团队成员容易出错。Bash 脚本超过 500 行且分散在不同仓库。

**解决:** 用 150 行 YAML 实现全栈应用部署流程，包含数据库迁移、并行部署、健康检查和自动回滚。创建工作流模板库供团队复用。

**成果:** 部署失败率从 15% 降到 3%，平均部署时间从 45 分钟降到 15 分钟，新员工上手时间从 2 周降到 2 天。

### 核心能力需求

**工作流定义:** YAML DSL、服务器组、并行执行、条件循环、自动重试  
**节点生态:** Docker、Shell、文件、HTTP、健康检查  
**可观测性:** 实时进度、步骤日志、执行历史  
**协作复用:** 模板库、参数化、知识共享

---

## 技术选型

**核心技术:**

| 组件 | 技术 | 理由 |
|-----------|------------|---------------|
| **语言** | Go | 性能、并发、跨平台 |
| **工作流运行时** | Temporal | 生产验证的持久执行 |
| **容器化** | Docker | 便携 Agent 部署 |
| **API 风格** | RESTful + OpenAPI 3.0 | 标准化、多语言支持 |

**交付方式:**
- Waterflow Server (Docker 镜像)
- Waterflow Agent (Docker 镜像)
- CLI 工具 (二进制)
- Go SDK (Go 模块)
- REST API (OpenAPI 规范)

**部署支持:**
- 一体化部署: Docker Compose
- 生产部署: Docker Compose
- 平台: Linux, macOS (开发), Windows (WSL2)

---

## 文档要求

**文档框架 (Divio 系统):**

| 类型 | 目的 | 受众 |
|------|---------|----------|
| **教程** | 学习导向、动手实践 | 新用户 |
| **操作指南** | 问题导向、实用性 | 中级用户 |
| **参考** | 信息导向、精确 | 所有用户 |
| **解释** | 理解导向、概念 | 高级用户 |

**MVP 文档交付物:**
- 快速开始指南 (30分钟部署到首个工作流)
- Server 部署指南
- Agent 配置指南
- YAML DSL 语法参考
- 节点参考文档 (10个内置节点)
- REST API 规范 (OpenAPI 3.0)
- 工作流示例模板库

---

## MVP 策略与开发方法
| **解释** | 理解导向、概念 | 高级用户 | 概念文章 |

**MVP 文档结构:**

```
docs/
├── getting-started/
│   ├── quick-start.md              # 30 分钟教程 (部署 Server + 首个工作流)
│   ├── installation.md             # Server 和 Agent 部署
│   └── first-workflow.md           # "Hello World" 示例
├── guides/
│   ├── server-deployment.md       # Server 部署指南
│   ├── agent-setup.md              # Agent 配置指南
│   ├── custom-nodes.md             # 构建自定义节点
│   ├── multi-server-workflows.md   # 高级模式
│   └── troubleshooting.md          # 常见问题和解决方案
├── reference/
│   ├── rest-api.md                 # REST API 规范 (OpenAPI 3.0)
│   ├── dsl-syntax.md               # 完整 YAML 规范
│   ├── go-sdk.md                   # Go SDK API 文档
│   ├── cli-reference.md            # 所有 CLI 命令
│   ├── nodes/
│   │   ├── control-flow.md         # 条件、循环、睡眠
│   │   ├── operations.md           # Shell、脚本、文件、HTTP
│   │   └── docker.md               # Docker 节点
│   └── configuration.md            # 所有配置选项
├── concepts/
│   ├── architecture.md             # C4 Model 系统架构
│   ├── execution-model.md          # Event Sourcing + 单节点执行
│   ├── server-groups.md            # Task Queue 直接映射
│   └── node-system.md              # 插件化节点系统 (Go Plugin)
├── adr/                            # Architecture Decision Records
│   ├── README.md                   # ADR 索引
│   ├── 0001-use-temporal-workflow-engine.md
│   ├── 0002-single-node-execution-pattern.md
│   ├── 0003-plugin-based-node-system.md
│   ├── 0004-yaml-dsl-syntax.md
│   ├── 0005-expression-system-syntax.md
│   └── 0006-task-queue-routing.md
├── examples/
│   ├── single-server-deploy.yaml
│   ├── multi-server-health-check.yaml
│   ├── distributed-stack.yaml
│   └── README.md                   # 示例目录
└── contributing/
    ├── development-setup.md
    ├── node-contribution.md
    └── code-of-conduct.md
```
│   ├── cli-reference.md            # 所有 CLI 命令
│   ├── nodes/
│   │   ├── control-flow.md         # 条件、循环、睡眠
│   │   ├── operations.md           # Shell、脚本、文件、HTTP
│   │   └── docker.md               # Docker 节点
│   └── configuration.md            # 所有配置选项
├── concepts/
│   ├── architecture.md             # 系统架构
│   ├── execution-model.md          # 工作流执行原理
│   ├── server-groups.md            # 服务器组概念
│   └── node-system.md              # 节点架构
├── examples/
│   ├── single-server-deploy.yaml
│   ├── multi-server-health-check.yaml
│   ├── distributed-stack.yaml
│   └── README.md                   # 示例目录
└── contributing/
    ├── development-setup.md
    ├── node-contribution.md
    └── code-of-conduct.md
```

**文档标准:**

1. **代码示例**
   - 每个示例必须可直接运行无需修改
   - 包含预期输出
   - 提供上下文（功能说明、使用场景）

2. **API 文档**
   - 从代码注释生成 (GoDoc)
   - 包含参数描述和返回值
   - 为每个 API 提供使用示例

3. **版本控制**
   - 文档与版本发布同步
   - 明确标记特定版本功能
   - 维护前 2 个主版本的文档

4. **搜索和导航**
   - 全文搜索功能
   - 面包屑导航
   - 相关内容链接
   - 长页面的目录

**质量指标:**
- [ ] 每个功能都有文档
- [ ] 代码示例在 CI 中测试
- [ ] 文档在 PR 中审查
- [ ] 用户反馈机制（“这有帮助吗？”）
- [ ] 文档与支持工单比例 <5:1

**工具链:**
- **Generator:** Docusaurus or VitePress
- **API Docs:** GoDoc + OpenAPI Generator
- **Diagrams:** Mermaid.js or Excalidraw
- **Hosting:** GitHub Pages or Vercel
- **Analytics:** Google Analytics or Plausible

### 技术栈

**核心技术:**

| 组件 | 技术 | 版本 | 理由 |
|-----------|------------|---------|---------------|
| **语言** | Go | 1.21+ | 性能、并发、跨平台 |
| **工作流运行时** | Temporal | 1.22+ | 生产验证的持久执行 |
| **容器化** | Docker | 20.10+ | 便携 Agent 部署 |
| **CLI 框架** | Cobra | 1.7+ | Go CLI 的行业标准 |
| **配置管理** | Viper | 1.16+ | 灵活的配置处理 |
| **HTTP 框架** | Echo/Gin | 最新 | 轻量 REST API 服务 |
| **日志** | Zap | 1.26+ | 高性能结构化日志 |
| **测试** | Testify | 1.8+ | 全面的测试断言 |

**开发工具:**
- **构建:** Go modules, Makefile
- **CI/CD:** GitHub Actions
- **代码质量:** golangci-lint, gosec
- **文档:** GoDoc, Docusaurus
- **容器化:** Docker, Docker Compose

**分发方式:**
- **二进制发布:** GitHub Releases (Linux, macOS, Windows)
- **容器镜像:** Docker Hub, GitHub Container Registry
- **包管理器:** Homebrew (macOS), apt/yum (Linux)
- **Go 模块:** `go get github.com/websoft9/waterflow`

---

## MVP 策略与开发方法

### MVP 理念

**方法:** 服务 MVP - 构建可扩展的基础，功能集最小化

**核心原则:**
1. **验证服务模型** - 证明 Waterflow Server 提供的 REST API 可用且易用
2. **展示核心价值** - 证明 YAML DSL 可以编排多服务器任务
3. **实现可扩展性** - 架构支持未来的节点生态
4. **最小化范围蔓延** - 无情删除不验证核心假设的功能

**构建-测量-学习循环:**
```
构建:   与 3 个设计伙伴共同开发 MVP
         ↓
测量: 部署时间、API 响应时间、工作流成功率、用户反馈
         ↓
学习:   哪些节点是必需的？API 设计是否直观？
         ↓
迭代: 优化 REST API、添加关键节点、改进文档
```

### 开发阶段

**阶段 1: 基础架构（第 1-4 周）**
- [ ] Waterflow Server 框架搭建 (无状态服务)
- [ ] REST API 服务实现 (Gin/Echo)
- [ ] DSL 解析器和 Schema 验证器
- [ ] 表达式引擎实现 (${{ }} 语法)
- [ ] Temporal Client 集成 (Event Sourcing 架构)
- [ ] Workflow 定义 (单节点执行模式: 每个 Step = 1 Activity)
- [ ] Activity 实现 (ExecuteNode Activity)
- [ ] Agent Worker 基础框架
- [ ] Plugin Manager (扫描 .so 文件)
- [ ] Node Registry (节点注册与管理)
- [ ] 核心组件单元测试

**阶段 2: 核心节点插件（第 5-6 周）**
- [ ] 节点插件接口设计 (Execute, Validate, Schema)
- [ ] 实现 10 个内置节点 (编译为 .so 插件)
  - [ ] 控制流: condition, loop, sleep
  - [ ] 操作: shell, script, file/transfer, http/request
  - [ ] Docker: docker/exec, docker/compose-up, docker/compose-down
- [ ] Plugin 打包工具 (go build -buildmode=plugin)
- [ ] 热加载机制实现 (fsnotify 监控)
- [ ] 节点文档和 DSL Schema
- [ ] 每个节点的集成测试
- [ ] 错误处理和结构化日志

**阶段 3: 客户端和文档（第 7-9 周）**
- [ ] Waterflow Server Docker 镜像打包
- [ ] Waterflow Agent Docker 镜像 (含所有 .so 插件)
- [ ] Docker Compose 一键部署 (Server + Temporal + Agent)
- [ ] Go SDK 客户端 (REST API 封装)
- [ ] CLI 工具实现 (validate, submit, status, logs, nodes)
- [ ] REST API OpenAPI 3.0 文档 (Swagger UI)
- [ ] 快速开始指南 (30 分钟教程)
- [ ] 架构决策记录 (6 个 ADR 文档)
- [ ] 核心架构概念文档 (Event Sourcing, 单节点执行等)
- [ ] 自定义节点插件开发指南
- [ ] 工作流模板示例 (3 个)

**阶段 4: 集成与验证（第 10-12 周）**
- [ ] 设计伙伴部署验证（3 个生产实例）
- [ ] 端到端验收测试场景
  - [ ] 多服务器健康检查
  - [ ] 分布式应用部署
- [ ] 架构特性验证测试
  - [ ] Event Sourcing: 进程崩溃恢复测试
  - [ ] 单节点执行: 独立超时/重试验证
  - [ ] 插件系统: 热加载测试
  - [ ] Task Queue: 跨服务器路由验证
- [ ] 性能基准测试
  - [ ] API 响应时间 (<500ms)
  - [ ] 并发工作流支持 (≥100 Agent)
  - [ ] YAML 解析性能 (1000行 <100ms)
- [ ] 生产环境压力测试
- [ ] 根据反馈优化文档和 API
- [ ] Bug 修复和用户体验打磨
- [ ] 发布准备 (Docker 镜像、二进制包、文档站点)

### MVP 范围权衡

**包含（必需）:**
- ✓ 具备核心工作流能力的 Go SDK
- ✓ 10 个内置节点（控制流 + 操作 + Docker）
- ✓ 通过 Docker 部署 Agent
- ✓ 用于验证和测试的 CLI
- ✓ 自助服务入门文档

**延后（Post-MVP）:**
- ✗ REST API 服务器（直接使用 SDK）
- ✗ 高级 DSL 功能（变量、子工作流）
- ✗ 工作流监控 Web UI
- ✗ >10 个节点（根据使用数据添加）

**明确排除:**
- ✗ 带用户管理的独立服务器
- ✗ 可视化工作流编辑器
- ✗ 节点市场
- ✗ 多语言节点支持

---

## 风险管理

**技术风险:**

| 风险 | 概率 | 影响 | 缓解策略 |
|------|-------------|--------|---------------------|
| **Temporal 学习曲线比预期陡峭** | 中 | 高 | 开发前完成 Temporal 教程；构建 PoC 验证架构 |
| **DSL 设计不满足用户需求** | 中 | 高 | 早期与设计伙伴验证语法；根据反馈迭代 |
| **生产环境中的 Agent 连接问题** | 低 | 中 | 实现健壮的重试逻辑和健康监控；在多种网络环境测试 |
| **工作流执行的性能瓶颈** | 低 | 中 | 早期设定性能基准；分析和优化热点路径 |
| **节点接口对扩展太僵化** | 中 | 高 | 设计带扩展点的插件 API；锁定前用 2-3 个自定义节点验证 |

**市场风险:**

| 风险 | 概率 | 影响 | 缓解策略 |
|------|-------------|--------|---------------------|
| **平台开发者偏好构建自定义解决方案** | 中 | 高 | 通过集成时间对比展示 ROI；提供迁移指南 |
| **用户抵触从现有脚本迁移** | 高 | 中 | 提供脚本转工作流工具；强调渐进采用 |
| **竞争对手发布类似解决方案** | 低 | 中 | 专注于差异化（可嵌入 + 声明式 + 分布式）；快速行动 |

**资源风险:**

| 风险 | 概率 | 影响 | 缓解策略 |
|------|-------------|--------|---------------------|
| **团队规模不足以完成时间表** | 中 | 高 | 无情优先级排序；推迟非必要功能；考虑外部承包商 |
| **关键团队成员不可用** | 低 | 高 | 记录架构决策；关键组件的结对编程 |
| **设计伙伴带宽有限** | 中 | 中 | 自助服务文档；异步反馈渠道；定期检查 |

**不可逆决策（MVP 必须做对）:**
1. **DSL 语法结构** - 破坏性变更会影响现有工作流（第 2 周决策）
2. **节点接口设计** - 影响整个节点生态（第 3 周决策）
3. **Temporal 工作流映射** - 重构执行模型成本极高（第 1 周决策）

**可接受的技术捷径（可重构）:**  
简单轮询负载均衡、基于文件的服务器组配置、基础 CLI 输出、同步工作流提交、内存节点注册表

---

## 附录

### 术语表

**Agent:** 部署在目标服务器上执行工作流任务的 Worker 进程。

**DSL (Domain-Specific Language):** 基于 YAML 的工作流定义语法。

**Durability:** 确保工作流状态在进程重启后持久化的属性。Waterflow 采用 Event Sourcing 模式,所有执行状态存储在 Temporal Event History 中,Server 完全无状态。

**Node:** 工作流中可重用的原子操作单元（例如 shell 命令、Docker 操作）。所有节点以 Go Plugin (.so 文件) 形式实现,支持热加载和自定义扩展。

**Runtime:** 管理工作流状态和协调的底层工作流执行引擎。Waterflow 使用 Temporal 作为 Runtime,通过 Event Sourcing 实现工作流状态的完全持久化和恢复。

**Server Group:** Agent 的逻辑集合,工作流可以目标执行。通过 runs-on 字段直接映射到 Temporal Task Queue,利用 Temporal 原生负载均衡。

**Workflow:** 声明式的任务执行规范，包括依赖关系和协调逻辑。

### 参考资料

**行业标准:**
- 12-Factor App Methodology: https://12factor.net/
- Divio Documentation System: https://documentation.divio.com/
- GitHub Actions Syntax: https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions
- OpenAPI Specification: https://spec.openapis.org/oas/latest.html

**技术文档:**
- Go Documentation Standards: https://go.dev/doc/effective_go
- Temporal Documentation: https://docs.temporal.io/
- Docker Best Practices: https://docs.docker.com/develop/dev-best-practices/

**产品管理框架:**
- SMART Goals: https://en.wikipedia.org/wiki/SMART_criteria
- OKR Framework: https://www.whatmatters.com/faqs/okr-meaning-definition-example
- North Star Metric: https://amplitude.com/blog/product-north-star-metric

---

**文档状态:** 架构设计完成  
**最后更新:** 2025-12-16  
**下次评审:** Post-MVP（设计伙伴验证后）
