# Waterflow 核心架构概念

本目录包含 Waterflow 核心架构概念的深度解析文档。这些文档面向高级用户、开发者和架构师，旨在帮助您深入理解 Waterflow 的设计决策、工作原理和扩展能力。

---

## 📚 概念文档导航

### 1. [Event Sourcing 执行模型](./event-sourcing-execution-model.md)

**概述:** Waterflow 如何使用 Temporal Event History 实现完整的状态持久化、崩溃恢复和时间旅行调试。

**核心内容:**
- Event Sourcing 是什么 - 定义与原理
- Temporal Event History 机制
- 与传统数据库状态存储的对比
- 完整 Event History JSON 示例
- 工作流崩溃后恢复流程

**适合读者:**
- 高级用户 - 理解系统可靠性保障
- 架构师 - 评估架构决策和权衡
- 开发者 - 调试工作流执行问题

**关键要点:**
- ✅ 零状态丢失 - Server 崩溃不影响工作流执行
- ✅ 完整审计日志 - 所有操作可追溯
- ✅ 时间旅行调试 - 查看任意时刻的执行状态
- ⚠️ Event History 体积增长 - 需要配置保留策略

---

### 2. [单节点执行模式](./single-node-execution-pattern.md)

**概述:** 每个 Step 编译为一个独立的 Temporal Activity 调用，提供精细的超时控制、重试策略和可观测性。

**核心内容:**
- 单节点执行模式定义 (1 Step = 1 Activity)
- 为什么选择单节点而非批处理模式
- YAML → Temporal Workflow 代码映射
- 单节点 vs 批处理的详细对比
- 性能影响分析 (Activity 数量 vs Event History 大小)

**适合读者:**
- 工作流开发者 - 理解 Step 的执行粒度和限制
- 性能优化人员 - 评估性能影响和优化方案
- 架构师 - 理解设计权衡和适用场景

**关键要点:**
- ✅ 精细超时控制 - 每个 Step 独立配置超时
- ✅ 独立重试策略 - 失败的 Step 单独重试
- ✅ 高可观测性 - Temporal UI 可查看每个 Step 状态
- ⚠️ Activity 数量多 - 大型工作流建议使用 Job 分批

---

### 3. [插件化节点系统](./plugin-node-system.md)

**概述:** 节点作为 Go Plugin (.so 动态库) 加载和执行，提供热加载、零核心代码修改的扩展能力。

**核心内容:**
- Go Plugin 机制介绍 (.so 动态库)
- Plugin Manager 和 NodeRegistry 架构
- 节点接口定义 (NodeExecutor interface)
- 热加载机制 (fsnotify 监控)
- 自定义节点开发完整指南

**适合读者:**
- 节点开发者 - 开发自定义节点扩展功能
- 扩展性需求用户 - 理解如何集成自定义业务逻辑
- 架构师 - 评估插件系统的设计和限制

**关键要点:**
- ✅ 语言一致性 - 插件和核心都是 Go 代码
- ✅ 热加载支持 - 新插件运行时动态加载
- ✅ 简化部署 - 插件以 .so 文件形式分发
- ⚠️ 跨平台限制 - Linux/macOS 支持，Windows 使用内置编译

---

### 4. [Task Queue 路由机制](./task-queue-routing.md)

**概述:** `runs-on` 字段直接映射到 Temporal Task Queue，实现零配置、高度灵活的跨服务器工作流编排。

**核心内容:**
- Temporal Task Queue 原理
- `runs-on` → Task Queue 直接映射设计
- 服务器组 (Server Group) 概念
- 单 Agent 多 Queue / 多 Agent 单 Queue 场景
- 动态添加新服务器组的流程

**适合读者:**
- 分布式部署用户 - 理解如何在多台服务器上执行工作流
- 运维工程师 - 配置 Agent 和 Task Queue 映射
- 架构师 - 评估路由机制的设计和适用场景

**关键要点:**
- ✅ 零配置路由 - `runs-on` 直接决定目标 Agent
- ✅ 完全灵活 - 可以使用任意 Queue 名称
- ✅ Temporal 原生负载均衡 - 多 Agent 自动负载均衡
- ⚠️ Queue 名称管理 - 拼写错误会导致工作流等待

---

### 5. [集成接口设计](./integration-interfaces.md)

**概述:** 4 个核心集成接口 (SecretProvider, EventHandler, LogHandler) 允许集成企业系统。

**核心内容:**
- Waterflow 可扩展性设计理念
- SecretProvider - 运行时密钥注入，零凭证存储
- EventHandler - 工作流生命周期事件通知
- LogHandler - 工作流执行日志集中管理
- 自定义实现示例 (Vault, Slack, Loki)

**适合读者:**
- 企业集成开发者 - 集成现有企业系统
- DevOps 工程师 - 配置和管理集成接口
- 架构师 - 评估集成能力和扩展方案

**关键要点:**
- ✅ SecretProvider - 集成 HashiCorp Vault、AWS KMS
- ✅ EventHandler - 集成 Slack、Prometheus、自定义告警
- ✅ LogHandler - 集成 Loki、CloudWatch Logs、Elasticsearch
- ✅ 编译时类型安全 - 接口在编译时检查

---

## 🎯 学习路径建议

### 初级用户 (理解核心概念)
1. **Event Sourcing 执行模型** - 理解 Waterflow 的可靠性保障
2. **Task Queue 路由机制** - 理解跨服务器工作流编排

### 中级用户 (深入技术细节)
1. **单节点执行模式** - 理解性能和控制粒度的权衡
2. **插件化节点系统** - 开发自定义节点扩展功能

### 高级用户 (企业集成和扩展)
1. **集成接口设计** - 集成企业系统 (CMDB, Vault, 监控)
2. **所有概念文档** - 全面理解 Waterflow 架构

---

## 🗺️ ADR 映射表格

核心概念文档与架构决策记录 (ADR) 的映射关系:

| 概念文档 | 相关 ADR | ADR 主题 |
|---------|---------|---------|
| [Event Sourcing 执行模型](./event-sourcing-execution-model.md) | [ADR-0001](../adr/0001-use-temporal-workflow-engine.md) | 使用 Temporal 作为工作流引擎 |
| [单节点执行模式](./single-node-execution-pattern.md) | [ADR-0002](../adr/0002-single-node-execution-pattern.md) | 单节点执行模式 |
| [插件化节点系统](./plugin-node-system.md) | [ADR-0003](../adr/0003-plugin-based-node-system.md) | 插件化节点系统 |
| [Task Queue 路由机制](./task-queue-routing.md) | [ADR-0006](../adr/0006-task-queue-routing.md) | Task Queue 路由机制 |
| [集成接口设计](./integration-interfaces.md) | 多个 Story | SecretProvider, EventHandler, LogHandler |

---

## 📖 与 Architecture 文档的关系

**[Architecture Document](../architecture.md)** 提供了 Waterflow 的完整 C4 Model 架构设计，包括:
- Context View - 系统上下文和外部依赖
- Container View - 主要组件和交互
- Component View - 内部组件设计
- Deployment View - 部署架构

**Concepts 文档** 深入解析特定的架构概念和设计决策:
- Architecture 文档: **宏观架构视图** (What & How)
- Concepts 文档: **微观设计原理** (Why & Trade-offs)

**建议阅读顺序:**
1. 先阅读 [Architecture Document](../architecture.md) 了解整体架构
2. 再阅读 Concepts 文档深入理解特定概念
3. 最后阅读 [ADR 文档](../adr/) 了解架构决策的详细背景

---

## 🔗 相关文档

### 架构设计
- [Architecture Document](../architecture.md) - 完整架构设计 (C4 Model)
- [ADR 目录](../adr/) - 所有架构决策记录
- [Temporal 架构分析](../analysis/temporal-architecture-analysis.md) - Temporal 深度分析

### 用户指南
- [Quick Start Guide](../quick-start.md) - 快速开始使用 Waterflow
- [YAML DSL Syntax Reference](../yaml-dsl-reference.md) - YAML DSL 完整语法
- [自定义节点开发指南](../guides/custom-node-development.md) - 节点开发教程

### 开发者文档
- [Story 文档](../sprint-artifacts/) - 所有 Story 的实现细节
- [Epic 文档](../epics.md) - 所有 Epic 的功能规划
- [API Guide](../api-guide.md) - REST API 使用指南

---

## ❓ 常见问题

### Q1: 这些概念文档与 ADR 文档有什么区别?

**A:** 
- **ADR 文档:** 记录架构决策的过程 (背景、决策、理由、后果、替代方案)
- **Concepts 文档:** 解释架构概念的工作原理、优势、劣势、使用场景

**示例:**
- ADR-0002 记录 **为什么** 选择单节点执行模式
- [单节点执行模式](./single-node-execution-pattern.md) 解释 **如何** 工作、性能影响、适用场景

### Q2: 我应该先阅读哪个概念文档?

**A:** 推荐阅读顺序:
1. **Event Sourcing 执行模型** - Waterflow 最核心的架构概念
2. **Task Queue 路由机制** - 分布式执行的关键
3. 根据兴趣阅读其他文档

### Q3: 这些概念文档是否包含代码示例?

**A:** 是的! 每个概念文档都包含:
- 完整的代码示例 (Go 代码、YAML 工作流)
- 架构图表 (Mermaid 流程图、序列图)
- 实际使用场景和配置示例

### Q4: 我需要理解所有概念才能使用 Waterflow 吗?

**A:** 不需要! 
- **普通用户:** 阅读 [Quick Start Guide](../quick-start.md) 即可开始使用
- **高级用户:** 阅读 Concepts 文档理解深层原理
- **企业集成:** 重点阅读 [集成接口设计](./integration-interfaces.md)

---

## 📝 文档贡献

如果您在阅读过程中发现错误或有改进建议，欢迎:
- 提交 GitHub Issue: https://github.com/websoft9/waterflow/issues
- 提交 Pull Request: https://github.com/websoft9/waterflow/pulls

---

**Last Updated:** 2026-01-15  
**Document Version:** 1.0  
**Maintained by:** Waterflow Team
