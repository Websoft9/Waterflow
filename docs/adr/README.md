# Architecture Decision Records (ADR)

本目录包含 Waterflow 项目的所有架构决策记录。

## ADR 格式

每个 ADR 遵循以下结构:

```markdown
# ADR-NNNN: 决策标题

**状态:** [提议中/已采纳/已废弃/已替代]
**日期:** YYYY-MM-DD
**决策者:** 团队/个人

## 背景
描述需要决策的问题和上下文

## 决策
明确的决策陈述

## 理由
为什么做出这个决策

## 后果
决策的正面和负面影响

## 替代方案
考虑过但未采纳的其他方案
```

## ADR 索引

| 编号 | 标题 | 状态 | 日期 |
|------|------|------|------|
| [0001](0001-use-temporal-workflow-engine.md) | 使用 Temporal 作为工作流引擎 | ✅ 已采纳 | 2025-12-13 |
| [0002](0002-single-node-execution-pattern.md) | 单节点执行模式 | ✅ 已采纳 | 2025-12-16 |
| [0003](0003-plugin-based-node-system.md) | 基于插件的节点系统 | ✅ 已采纳 | 2025-12-16 |
| [0004](0004-yaml-dsl-syntax.md) | YAML DSL 语法设计 | ✅ 已采纳 | 2025-12-13 |
| [0005](0005-expression-system-syntax.md) | 表达式系统语法 | ✅ 已采纳 | 2025-12-13 |
| [0006](0006-task-queue-routing.md) | Task Queue 路由机制 | ✅ 已采纳 | 2025-12-15 |

## 修订历史

- 2025-12-16: 创建 ADR 目录,记录核心架构决策
