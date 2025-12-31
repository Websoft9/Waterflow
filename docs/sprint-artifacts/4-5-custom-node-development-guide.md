# Story 4.5: 自定义节点开发指南

Status: 🔵 **ready-for-dev**

## Story

As a **开发者**,  
I want **完整的自定义节点开发文档**,  
So that **快速创建节点插件**。

## Context

这是 Epic 4 (节点扩展系统) 的第五个 Story,也是该 Epic 的**最后一个 Story**。在前置 Story 4.1-4.4 的基础上,提供一份 **全面、深入、实用的自定义节点开发指南**,确保第三方开发者能够快速上手并创建高质量的节点插件。

**前置依赖:**
- ✅ Story 3.1 (节点接口设计) - Node 接口定义、NodeResult、NodeMetadata、ParamSpec
- ✅ Story 4.1 (Plugin Manager) - PluginManager 加载机制、NodeRegistry
- ✅ Story 4.2 (参数 Schema 验证) - ValidateInputs() 运行时验证、JSON Schema
- ✅ Story 4.3 (重试策略配置) - Temporal RetryPolicy 集成
- ✅ Story 4.4 (开发示例) - Greeter 节点端到端示例

**Epic 背景:**  
Epic 4 专注于 **节点扩展能力**,使 Waterflow 成为可扩展的工作流平台。本 Story 作为 Epic 的收官之作,通过完整的开发指南,将前面 4 个 Story 的技术实现转化为开发者友好的文档,降低第三方节点开发门槛,促进节点生态发展。

**业务价值:**
- 🎯 **知识转移** - 将内部技术知识系统化,让外部开发者快速理解节点系统
- 🎯 **降低门槛** - 通过清晰的文档和示例,30 分钟内从零到创建首个节点
- 🎯 **质量保证** - 提供最佳实践和测试指南,确保第三方节点质量
- 🎯 **生态发展** - 促进社区贡献,建立 Waterflow 节点插件生态

**文档定位:**
- **不是** API 参考文档 (已有 ADR-0003、Story 3.1 文档)
- **不是** 快速入门 (已有 examples/plugins/template/)
- **是** 完整开发指南,涵盖从概念到部署的全流程
- **是** 最佳实践手册,包含常见问题和解决方案

## Acceptance Criteria

### AC1: 核心概念章节

**Given** 开发者初次接触 Waterflow 节点系统  
**When** 阅读开发指南的"核心概念"章节  
**Then** 说明节点接口的每个方法 (Name, Version, Params, Execute, Metadata)  
**And** 解释 ParamSpec 的所有字段及其作用 (Type, Required, Default, Pattern, Enum, MinValue, MaxValue)  
**And** 说明 NodeResult 结构和输出机制 (Outputs, Logs, Duration, Metadata)  
**And** 说明 NodeMetadata 的用途 (Description, Category, InputSchema, OutputSchema)  
**And** 解释节点分类系统 (exec/docker/http/file/flow/custom)  
**And** 说明节点版本管理策略 (语义化版本 v1, v1.2.3)  
**And** 说明 Go Plugin 机制和 Register() 函数  
**And** 包含架构图展示节点在系统中的位置  
**And** 引用相关 ADR 文档 (ADR-0003: 插件化节点系统)

### AC2: 分层示例章节

**Given** 开发者需要学习节点开发  
**When** 阅读"示例节点"章节  
**Then** 提供 3 个不同复杂度的示例节点:  

**示例 1: 简单节点 (Hello World)**
- 功能: echo 节点,原样返回输入
- 复杂度: <50 LOC
- 学习重点: 基础接口实现、参数验证、输出返回
- 参数: 1 个必需参数 (message: string)
- 输出: 1 个输出 (echo: string)

**示例 2: 中等复杂度节点 (Greeter)**
- 功能: 多语言问候语生成 (引用 Story 4.4 实现)
- 复杂度: <150 LOC
- 学习重点: 枚举验证、默认值、多输出、时间处理
- 参数: 3 个参数 (name: string, language: enum, time_of_day: enum)
- 输出: 3 个输出 (greeting, language, time_of_day)

**示例 3: 生产级节点 (HTTP Request)**
- 功能: HTTP 客户端 (参考 Story 3.5 实现)
- 复杂度: ~300 LOC
- 学习重点: 上下文取消、超时处理、错误分类、重试策略、复杂参数
- 参数: 8+ 参数 (url, method, headers, body, timeout, retry, ...)
- 输出: 5+ 输出 (status_code, body, headers, duration, ...)

**And** 每个示例包含:
- 完整源代码 (可直接编译)
- 参数定义和验证策略
- 执行逻辑实现
- 单元测试代码
- 使用示例 (YAML)

### AC3: 参数验证章节

**Given** 开发者需要定义节点参数  
**When** 阅读"参数定义和验证"章节  
**Then** 说明如何定义输入输出 Schema  
**And** 说明每种参数类型的验证规则 (string, int, float, bool, object, array)  
**And** 说明高级验证技术:
- Pattern - 正则表达式验证 (示例: email, URL, 文件路径)
- Enum - 枚举值验证 (示例: method: [GET, POST, PUT])
- MinValue/MaxValue - 数值范围 (示例: timeout: 1-3600)
- Required/Default - 必需性和默认值
**And** 说明如何使用 ValidateInputs() 函数  
**And** 提供 5+ 参数验证示例:
- 邮箱格式验证
- URL 格式验证
- 端口范围验证 (1-65535)
- 文件路径格式验证
- HTTP 方法枚举验证
**And** 说明自定义验证逻辑实现方法

### AC4: 错误处理和日志章节

**Given** 开发者需要处理错误和记录日志  
**When** 阅读"错误处理和日志"章节  
**Then** 说明如何处理错误和返回错误信息  
**And** 说明 Temporal 错误分类 (ApplicationError vs TemporaryError)  
**And** 说明如何区分永久错误和临时错误:
- 永久错误: 参数验证失败、资源不存在、权限不足 (不应重试)
- 临时错误: 网络超时、服务不可用、速率限制 (可以重试)
**And** 说明如何使用 ApplicationFailure 标记非重试错误  
**And** 说明日志记录最佳实践:
- 使用 NodeResult.AddLog() 记录结构化日志
- 记录关键执行步骤 (开始、进度、完成)
- 记录重要参数值 (脱敏处理敏感数据)
- 记录错误上下文 (便于调试)
**And** 提供错误处理示例代码 (参数错误、网络超时、权限不足)  
**And** 说明如何记录日志并在 Temporal UI 中查看

### AC5: 测试最佳实践章节

**Given** 开发者需要测试节点  
**When** 阅读"测试"章节  
**Then** 说明节点测试最佳实践  
**And** 说明单元测试编写方法:
- 测试每个接口方法 (Name, Version, Params, Metadata)
- 测试 Execute 正常路径和异常路径
- 测试参数验证 (必需参数缺失、类型错误、格式错误)
- 测试上下文取消 (ctx.Done())
**And** 说明如何使用表驱动测试 (table-driven tests)  
**And** 提供测试用例设计指南:
- 正常输入测试 (happy path)
- 边界值测试 (最小/最大值)
- 异常输入测试 (空值、nil、错误类型)
- 并发安全测试 (goroutine 并发调用)
**And** 说明集成测试方法 (使用 PluginManager 加载插件)  
**And** 说明测试覆盖率目标 (>80%)  
**And** 提供完整测试代码示例 (含 table-driven tests)

### AC6: 编译和部署章节

**Given** 开发者需要构建和部署节点插件  
**When** 阅读"编译和部署"章节  
**Then** 说明节点打包和分发方式  
**And** 说明编译前置要求:
- Go 版本必须与 Agent 匹配 (使用 `go version` 验证)
- CGO 必须启用 (`export CGO_ENABLED=1`)
- 平台限制 (仅支持 Linux/macOS)
**And** 说明编译命令: `go build -buildmode=plugin -o node.so main.go`  
**And** 说明编译验证步骤:
- 检查 .so 文件大小 (>100KB)
- 使用 `file` 命令验证文件类型
- 使用 `nm` 命令检查 Register 符号导出
**And** 说明插件部署流程:
1. 复制 .so 文件到 Agent 的 `/opt/waterflow/plugins/<category>/` 目录
2. 检查文件权限 (可读可执行)
3. Agent 自动热加载或重启 Agent
4. 验证插件加载成功 (查看日志)
**And** 说明热加载机制 (fsnotify 监控插件目录)  
**And** 说明版本管理策略 (文件名包含版本: `http-request-v1.so`)  
**And** 提供 Makefile 模板简化编译流程

## Dev Notes

### 项目结构信息

**文档位置:**
- 主文档: `/data/Waterflow/docs/guides/node-development.md`
- 节点参考文档: `/data/Waterflow/docs/nodes/`
- ADR 文档: `/data/Waterflow/docs/adr/0003-plugin-based-node-system.md`

**示例代码位置:**
- 开发模板: `/data/Waterflow/examples/plugins/template/`
- Echo 示例: `/data/Waterflow/examples/plugins/echo/`
- Greeter 示例: `/data/Waterflow/examples/plugins/greeter/` (Story 4.4)

**相关实现代码:**
- 节点接口: `/data/Waterflow/pkg/dsl/node/interface.go`
- 参数验证: `/data/Waterflow/pkg/dsl/node/validator.go`
- Plugin Manager: `/data/Waterflow/internal/agent/plugin_manager.go`

### 架构约束和设计决策

**核心架构决策 (ADR-0003):**
- ✅ **插件化系统** - 所有节点(含内置节点)均为 Go Plugin (.so 文件)
- ✅ **热加载支持** - fsnotify 监控插件目录,自动加载新插件
- ✅ **统一接口** - 所有节点实现 `pkg/dsl/node.Node` 接口
- ✅ **版本隔离** - 节点版本独立于 Agent 版本

**技术限制:**
- ⚠️ **Go 版本匹配** - Plugin 编译和 Agent 必须使用完全相同的 Go 版本
- ⚠️ **CGO 要求** - 必须启用 CGO (`CGO_ENABLED=1`)
- ⚠️ **平台限制** - 仅支持 Linux/macOS (Windows 不支持 Go Plugin)

**接口设计 (Story 3.1):**
```go
// pkg/dsl/node/interface.go
type Node interface {
    Name() string          // 格式: category/name (如 exec/shell)
    Version() string       // 格式: vX.Y.Z 或 vX (如 v1, v1.2.3)
    Params() map[string]ParamSpec  // 参数定义
    Execute(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error)
    Metadata() NodeMetadata
}
```

**参数验证 (Story 4.2):**
```go
// 参数规格
type ParamSpec struct {
    Type        string        // string, int, float, bool, object, array
    Required    bool          // 是否必需
    Description string        // 参数说明
    Default     interface{}   // 默认值
    Pattern     string        // 正则表达式验证
    Enum        []interface{} // 枚举值验证
    MinValue    *float64      // 最小值 (数值类型)
    MaxValue    *float64      // 最大值 (数值类型)
}

// 运行时验证
func ValidateInputs(inputs map[string]interface{}, params map[string]ParamSpec) error
```

**错误处理 (Story 3.5, 4.3):**
- **永久错误 (不重试)**: 使用 `temporal.NewApplicationFailure()` 标记
  - 参数验证失败
  - 资源不存在 (404)
  - 权限不足 (401, 403)
  - 客户端错误 (4xx)
  - TLS 证书错误
- **临时错误 (可重试)**: 返回普通 error
  - 网络超时
  - 服务不可用 (503)
  - 服务器错误 (5xx)
  - DNS 解析失败

### 前置 Story 完成情况

| Story | 状态 | 相关内容 |
|-------|------|----------|
| 3.1 | ✅ done | 节点接口设计、NodeResult、NodeMetadata、ParamSpec |
| 4.1 | ✅ done | Plugin Manager、NodeRegistry、插件加载机制、热加载 |
| 4.2 | ✅ done | ValidateInputs() 函数、JSON Schema 验证、错误消息 |
| 4.3 | ✅ done | Temporal RetryPolicy 集成、错误分类、重试策略 |
| 4.4 | ✅ done | Greeter 节点示例、编译脚本、单元测试、集成测试 |

### 现有文档资源

**已有文档 (需要参考和引用):**
1. `/data/Waterflow/docs/guides/node-development.md` (416 行)
   - 现有的节点开发指南 (较简略)
   - 包含快速开始、接口说明、编译步骤
   - **本 Story 需要大幅扩展和完善此文档**

2. `/data/Waterflow/docs/adr/0003-plugin-based-node-system.md` (595 行)
   - 插件化节点系统的架构决策
   - 包含接口定义、实现示例、加载机制
   - **本 Story 应引用此 ADR,不重复内容**

3. `/data/Waterflow/docs/nodes/README.md` (306 行)
   - 核心节点库概览
   - 节点使用指南 (用户视角)
   - **本 Story 从开发者视角编写,可参考其组织结构**

4. `/data/Waterflow/docs/nodes/exec/shell.md` (456 行)
   - exec/shell 节点参考文档
   - 包含参数说明、返回值、使用示例、常见错误
   - **本 Story 可参考其文档风格,用于"示例 3: 生产级节点"**

**示例代码资源:**
1. `examples/plugins/template/` - 节点开发模板 (Story 3.1)
2. `examples/plugins/echo/` - Echo 示例节点 (Story 3.1)
3. `examples/plugins/greeter/` - Greeter 示例节点 (Story 4.4, 已完成)

### 现有文档基线分析

> **重要:** 本章节分析现有文档的内容和结构，明确扩展策略。

**文件:** `/data/Waterflow/docs/guides/node-development.md` (416 行)

**当前章节结构分析:**

```bash
# 读取现有文档结构
grep "^#" /data/Waterflow/docs/guides/node-development.md

# 预期输出示例:
# # Custom Node Development Guide
# ## Quick Start
# ## Node Interface
# ## Build and Deploy
# ## FAQ
```

**保留部分 (✅ 已完成，无需修改):**
- 文档头部和简介 (L1-30) - 保留原有的快速开始部分
- Quick Start 章节 (L31-120) - 简短的快速入门，方便快速上手
- 基础概念说明 - 如果已有，保留核心部分

**扩展部分 (📝 需要大幅扩展):**
- Node Interface 章节 → 扩展为 AC1 "核心概念章节" (详细说明每个接口方法)
- 可能缺少完整的示例 → 添加 AC2 "分层示例章节" (3 个完整示例)
- 可能缺少参数验证详细说明 → 添加 AC3 "参数验证章节" (5+ 验证示例)
- 可能缺少错误处理说明 → 添加 AC4 "错误处理和日志章节" (错误分类、日志最佳实践)
- 可能缺少测试指南 → 添加 AC5 "测试最佳实践章节" (单元测试、表驱动测试)
- Build and Deploy → 扩展为 AC6 "编译和部署章节" (详细的编译验证、部署流程)

**新增部分 (➕ 全新章节):**
- Best Practices 章节 - 代码组织、性能优化、安全考虑
- Advanced Topics 章节 - 上下文取消、超时处理、复杂参数
- Troubleshooting 章节 - 常见问题、调试技巧、错误排查

**文档更新策略:**

| 策略 | 说明 | 优点 | 缺点 |
|------|------|------|------|
| **扩展现有文档** | 在现有 416 行基础上扩展到 2000+ 行 | 保持连续性，保留现有引用 | 可能导致结构混乱 |
| **重写全部** | 完全重写，废弃旧版本 | 结构清晰，质量统一 | 破坏现有引用链接 |
| **版本化** | 保留旧版本，创建 v2 文档 | 向后兼容 | 维护两份文档 |

**推荐策略:** **扩展现有文档 + 版本控制**

**实施步骤:**

1. **备份现有文档:**
   ```bash
   cd /data/Waterflow/docs/guides
   cp node-development.md node-development-v1-backup.md
   ```

2. **保留核心部分 (L1-120):**
   - 文档标题和简介
   - Quick Start 章节（快速入门，30 分钟上手）
   - 基础接口说明

3. **扩展和新增章节 (L121-2000+):**
   - 在 Quick Start 后添加详细章节
   - 每个 AC 对应一个独立章节
   - 添加 Best Practices 和 Troubleshooting

4. **更新内部链接:**
   - 确保所有引用指向正确章节
   - 更新目录索引

5. **添加版本信息:**
   ```markdown
   # Custom Node Development Guide
   
   **版本:** 2.0 (Epic 4 完成版)  
   **上次更新:** 2025-12-31  
   **前置 Stories:** 3.1, 4.1, 4.2, 4.3, 4.4
   
   > 本指南基于 Epic 4 (Node Extension System) 完成后的最新实现。  
   > 如需查看旧版本，请参考 [v1 备份](node-development-v1-backup.md)。
   ```

**文档结构建议:**

```markdown
# Custom Node Development Guide

**版本:** 2.0 (Epic 4 完成版)  
**上次更新:** 2025-12-31

## Table of Contents
- [Overview](#overview)
- [Quick Start](#quick-start) (保留自 v1)
- [Core Concepts](#core-concepts) (AC1, 新增)
- [Node Examples](#node-examples) (AC2, 新增)
  - [Example 1: Echo Node (Simple)](#example-1-echo-node)
  - [Example 2: Greeter Node (Intermediate)](#example-2-greeter-node)
  - [Example 3: HTTP Request (Production)](#example-3-http-request)
- [Parameter Validation](#parameter-validation) (AC3, 新增)
- [Error Handling and Logging](#error-handling) (AC4, 新增)
- [Testing Best Practices](#testing) (AC5, 新增)
- [Build and Deploy](#build-and-deploy) (AC6, 扩展自 v1)
- [Best Practices](#best-practices) (新增)
- [Troubleshooting](#troubleshooting) (新增)
- [FAQ](#faq) (扩展自 v1)
- [References](#references) (新增)
- [Changelog](#changelog) (新增)

## Overview
<!-- 保留现有简介，轻微调整 -->

## Quick Start
<!-- 保留现有内容 (L31-120)，确保 30 分钟快速上手 -->

## Core Concepts (AC1)
<!-- 全新内容：详细说明接口、ParamSpec、NodeResult、NodeMetadata -->

## Node Examples (AC2)
<!-- 全新内容：3 个完整示例，从简单到复杂 -->

## Parameter Validation (AC3)
<!-- 全新内容：参数类型、验证规则、5+ 示例 -->

## Error Handling and Logging (AC4)
<!-- 全新内容：错误分类、Temporal 集成、日志最佳实践 -->

## Testing Best Practices (AC5)
<!-- 全新内容：单元测试、表驱动测试、集成测试 -->

## Build and Deploy (AC6)
<!-- 扩展现有 "Build and Deploy" 章节，添加详细验证步骤 -->

## Best Practices
<!-- 全新内容：代码组织、性能优化、安全考虑 -->

## Troubleshooting
<!-- 全新内容：常见问题、调试技巧、错误排查决策树 -->

## FAQ
<!-- 保留现有内容，添加新问题 -->

## References
<!-- 全新内容：ADR、Story、外部文档链接 -->

## Changelog
<!-- 记录从 v1 到 v2 的主要变更 -->
```

**验收标准:**

完成后，运行以下验证：

```bash
# 1. 文档长度验证
wc -l /data/Waterflow/docs/guides/node-development.md
# 预期: >2000 行

# 2. 章节完整性验证
grep "^##" /data/Waterflow/docs/guides/node-development.md | wc -l
# 预期: ≥12 个章节

# 3. 代码示例验证
grep -c "^```go" /data/Waterflow/docs/guides/node-development.md
# 预期: ≥20 个 Go 代码块

grep -c "^```yaml" /data/Waterflow/docs/guides/node-development.md
# 预期: ≥10 个 YAML 代码块

# 4. 链接有效性验证
grep -o '\[.*\](.*\.md)' /data/Waterflow/docs/guides/node-development.md | \
  while read link; do
    file=$(echo $link | sed 's/.*((.*))/ \1/')
    if [ ! -f "/data/Waterflow/docs/$file" ]; then
      echo "❌ 无效链接: $link"
    fi
  done
# 预期: 无输出（所有链接有效）

# 5. 章节引用完整性
for ac in "AC1" "AC2" "AC3" "AC4" "AC5" "AC6"; do
  grep -q "$ac" /data/Waterflow/docs/guides/node-development.md || echo "❌ 缺少 $ac 对应章节"
done
# 预期: 无输出（所有 AC 都有对应章节）
```

**文档更新记录:**

| 版本 | 日期 | 主要变更 |
|------|------|----------|
| 1.0 | 2025-11-15 | 初始版本，基础接口说明和快速开始 |
| 2.0 | 2025-12-31 | Epic 4 完成版，添加 6 个 AC 章节，3 个完整示例，2000+ 行完整指南 |

### 文档结构建议

基于 AC 要求和现有文档资源,建议的文档结构:

```markdown
# Custom Node Development Guide

## Overview
- Waterflow 节点系统简介
- 本指南的目标读者和用途
- 前置知识要求 (Go 基础、接口概念)

## Core Concepts (AC1)
- 节点接口详解 (Node interface)
- 参数系统 (ParamSpec)
- 执行结果 (NodeResult)
- 元数据系统 (NodeMetadata)
- 节点分类和版本
- Go Plugin 机制
- 架构图 (节点在系统中的位置)

## Node Examples (AC2)
- 示例 1: Hello World (Echo Node)
- 示例 2: Intermediate (Greeter Node)
- 示例 3: Production-Grade (HTTP Request)

## Parameter Definition and Validation (AC3)
- 参数类型系统
- 高级验证技术 (Pattern, Enum, Range)
- ValidateInputs() 使用
- 自定义验证逻辑

## Error Handling and Logging (AC4)
- 错误分类 (永久 vs 临时)
- Temporal 错误处理
- 日志记录最佳实践
- 调试技巧

## Testing (AC5)
- 单元测试
- 表驱动测试
- 集成测试
- 测试覆盖率

## Build and Deployment (AC6)
- 编译前置要求
- 编译步骤
- 验证和调试
- 部署流程
- 热加载机制
- 版本管理

## Best Practices
- 代码组织
- 性能优化
- 安全考虑
- 常见陷阱

## FAQ
- Go 版本不匹配
- CGO 未启用
- 插件加载失败
- 调试困难

## References
- ADR-0003: 插件化节点系统
- Story 3.1: 节点接口设计
- 核心节点库文档
```

### 引用的代码实现

**1. 节点接口 (pkg/dsl/node/interface.go):**
```go
type Node interface {
    Name() string
    Version() string
    Params() map[string]ParamSpec
    Execute(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error)
    Metadata() NodeMetadata
}
```

**2. ParamSpec 结构 (pkg/dsl/node/registry.go):**
```go
type ParamSpec struct {
    Type        string
    Required    bool
    Description string
    Default     interface{}
    Pattern     string
    Enum        []interface{}
    MinValue    *float64
    MaxValue    *float64
}
```

**3. NodeResult 结构 (pkg/dsl/node/result.go):**
```go
type NodeResult struct {
    Outputs  map[string]interface{}
    Logs     []string
    Duration time.Duration
    Metadata map[string]interface{}
}
```

**4. 参数验证 (pkg/dsl/node/validator.go):**
```go
func ValidateInputs(inputs map[string]interface{}, params map[string]ParamSpec) error
```

### 前置 Story 的测试覆盖率

根据 sprint-status.yaml 中的完成记录:

| Story | 测试覆盖率 | 备注 |
|-------|------------|------|
| 3.1 | 92.2% | 节点接口设计 |
| 3.2 | 95.2% | Shell 命令执行节点 |
| 3.3 | 86.2% | 脚本文件执行节点 |
| 3.4 | 97.2% | Sleep 延迟节点 |
| 3.5 | 92.3% | HTTP 请求节点 |
| 3.6 | >90% | 文件传输节点 (完整模式) |
| 3.7 | >90% | Docker Exec 节点 (完整模式) |
| 3.8 | 43.5% | Docker Compose 节点 |
| 4.4 | (待实现) | Greeter 节点示例,目标 >80% |

**本 Story 需要:**
- 引用这些高覆盖率的测试代码作为最佳实践示例
- 说明如何达到 80%+ 覆盖率

### 关键技术细节

**1. Go Plugin 编译命令:**
```bash
go build -buildmode=plugin -o mynode.so main.go
```

**2. Go 版本验证:**
```bash
go version  # 必须与 Agent 完全一致
```

**3. CGO 启用:**
```bash
export CGO_ENABLED=1
```

**4. 插件部署路径:**
```
/opt/waterflow/plugins/
├── exec/
│   ├── shell-v1.so
│   └── script-v1.so
├── docker/
│   ├── exec-v1.so
│   └── compose-v1.so
├── http/
│   └── request-v1.so
├── file/
│   └── transfer-v1.so
├── flow/
│   └── sleep-v1.so
└── custom/
    ├── greeter-v1.so
    └── mynode-v1.so
```

**5. 热加载机制 (Story 4.1):**
- PluginManager 使用 fsnotify 监控 `/opt/waterflow/plugins/` 目录
- 检测到新 .so 文件时自动加载并注册
- 插件加载失败记录错误但不影响 Agent 启动

### 参考资料链接

**外部文档:**
- [Go Plugin Package](https://pkg.go.dev/plugin)
- [Temporal Go SDK](https://docs.temporal.io/dev-guide/go)
- [JSON Schema](https://json-schema.org/)

**项目内部文档:**
- [Architecture](../architecture.md) - 系统架构
- [ADR-0003: Plugin-Based Node System](../adr/0003-plugin-based-node-system.md)
- [ADR-0002: Single Node Execution Pattern](../adr/0002-single-node-execution-pattern.md)
- [Node Reference Documentation](../nodes/README.md)

### 文档质量要求

**完整性:**
- ✅ 覆盖所有 6 个 AC
- ✅ 包含 3 个完整示例 (简单、中等、复杂)
- ✅ 包含所有接口方法的详细说明
- ✅ 包含错误处理、日志、测试、编译、部署的完整流程

**正确性:**
- ✅ 所有代码示例可直接编译运行
- ✅ 所有 Go 版本、CGO、平台要求准确无误
- ✅ 所有文件路径和命令正确
- ✅ 引用现有文档和代码时保持一致性

**实用性:**
- ✅ 提供可复制粘贴的代码示例
- ✅ 提供 Makefile 模板简化编译
- ✅ 提供常见问题和解决方案
- ✅ 提供调试技巧和最佳实践

**可读性:**
- ✅ 清晰的章节结构
- ✅ 适当的代码高亮
- ✅ 详细的注释说明
- ✅ 表格、列表、图表辅助理解

**一致性:**
- ✅ 与现有节点文档风格一致 (参考 docs/nodes/exec/shell.md)
- ✅ 与现有开发指南风格一致 (参考 docs/guides/node-development.md)
- ✅ 术语使用与 ADR 和架构文档一致

### 开发策略

**文档组织:**
1. **扩展现有文档** - 在 `/data/Waterflow/docs/guides/node-development.md` 基础上扩展
   - 保留现有的快速开始部分 (前 200 行左右)
   - 添加详细的章节内容 (AC1-AC6)
   - 确保向后兼容,不破坏现有引用

2. **复用现有示例** - 引用已完成的示例节点
   - Echo 节点 (examples/plugins/echo/) - 示例 1
   - Greeter 节点 (examples/plugins/greeter/) - 示例 2
   - HTTP Request 节点 (plugins/http/) - 示例 3

3. **引用架构文档** - 避免重复内容
   - 插件机制 → 引用 ADR-0003
   - 执行模式 → 引用 ADR-0002
   - 接口设计 → 引用 Story 3.1 文档

**质量保证:**
1. 所有代码示例必须可编译运行
2. 所有命令必须在 Linux 环境验证
3. 所有文件路径必须与实际项目结构一致
4. 所有引用链接必须有效

**文档长度估算:**
- 基于 AC1-AC6 要求: ~1500-2000 行 Markdown
- 包含 3 个完整示例代码: ~500 行
- 包含测试代码示例: ~200 行
- 总计: ~2200-2700 行

### Project Context Reference

**技术栈:**
- Go 1.22.0+
- Temporal Go SDK v1.38.0
- Go Plugin 机制
- JSON Schema 验证

**代码规范:**
- Gofmt 格式化
- Golint 静态检查
- 测试覆盖率 >80%
- Godoc 注释

**文档规范:**
- Markdown 格式
- 中英文混排 (技术术语英文,说明中文)
- 代码块使用语法高亮
- 表格辅助说明

## Dev Agent Record

### Context Reference

[待 Dev Agent 添加]

### Agent Model Used

[待 Dev Agent 添加]

### Debug Log References

[待 Dev Agent 添加]

### Completion Notes List

[待 Dev Agent 添加]

### File List

预计修改的文件:
- `/data/Waterflow/docs/guides/node-development.md` (主要扩展此文件)

可能需要创建的补充文件:
- 无 (所有内容整合到主文档中)

可能需要引用的现有文件:
- `/data/Waterflow/docs/adr/0003-plugin-based-node-system.md`
- `/data/Waterflow/docs/nodes/README.md`
- `/data/Waterflow/docs/nodes/exec/shell.md`
- `/data/Waterflow/examples/plugins/template/`
- `/data/Waterflow/examples/plugins/echo/`
- `/data/Waterflow/examples/plugins/greeter/` (Story 4.4)

---

**Story 创建完成!** ✅

**下一步 (Dev Agent 执行时):**
1. 阅读并分析现有的 `docs/guides/node-development.md` (416 行)
2. 阅读 Story 4.4 的 Greeter 节点实现 (作为示例 2)
3. 阅读 Story 3.5 的 HTTP Request 节点实现 (作为示例 3)
4. 按照 AC1-AC6 的要求扩展和完善文档
5. 确保所有代码示例可编译运行
6. 验证所有链接和引用有效
7. 运行文档质量检查 (拼写、格式、一致性)

**质量门控:**
- ✅ 所有 6 个 AC 完全达成
- ✅ 3 个示例节点代码完整且可运行
- ✅ 文档长度 >2000 行
- ✅ 所有接口方法有详细说明
- ✅ 所有编译、测试、部署步骤清晰
- ✅ 常见问题和最佳实践完善

---

## 验收标准清单

> **重要:** 完成文档开发后的质量验证清单

### 文档完整性验证

**必需章节 (12 个):**
- [ ] ✅ Overview (文档概览)
- [ ] ✅ Quick Start (保留自现有文档)
- [ ] ✅ Core Concepts (AC1 - 核心概念)
- [ ] ✅ Node Examples (AC2 - 3 个分层示例)
- [ ] ✅ Parameter Validation (AC3 - 参数验证)
- [ ] ✅ Error Handling and Logging (AC4 - 错误处理和日志)
- [ ] ✅ Testing Best Practices (AC5 - 测试最佳实践)
- [ ] ✅ Build and Deploy (AC6 - 编译和部署)
- [ ] ✅ Best Practices (最佳实践)
- [ ] ✅ Troubleshooting (故障排除)
- [ ] ✅ FAQ (常见问题)
- [ ] ✅ References (参考资料)

### 内容质量验证

**文档长度要求:**
- [ ] ✅ 总行数 >2000 行
- [ ] ✅ Core Concepts 章节 >300 行
- [ ] ✅ Node Examples 章节 >500 行 (3 个完整示例)
- [ ] ✅ Parameter Validation 章节 >200 行
- [ ] ✅ Error Handling 章节 >200 行
- [ ] ✅ Testing 章节 >250 行
- [ ] ✅ Build and Deploy 章节 >200 行

**代码示例数量要求:**
- [ ] ✅ Go 代码块 ≥20 个
- [ ] ✅ YAML 使用示例 ≥10 个
- [ ] ✅ Bash 命令示例 ≥15 个
- [ ] ✅ 完整可运行节点示例 ≥3 个
- [ ] ✅ 测试代码示例 ≥5 个

**AC1: 核心概念章节验证**
- [ ] ✅ Node 接口 5 个方法详细说明 (Name, Version, Params, Execute, Metadata)
- [ ] ✅ ParamSpec 所有字段说明 (Type, Required, Default, Pattern, Enum, MinValue, MaxValue)
- [ ] ✅ NodeResult 结构说明 (Outputs, Logs, Duration, Metadata)
- [ ] ✅ NodeMetadata 用途说明
- [ ] ✅ 节点分类系统说明 (exec/docker/http/file/flow/custom)
- [ ] ✅ 节点版本管理策略说明 (v1, v1.2.3)
- [ ] ✅ Go Plugin 机制和 Register() 函数说明
- [ ] ✅ 架构图展示节点在系统中的位置
- [ ] ✅ 引用 ADR-0003 (插件化节点系统)

**AC2: 分层示例章节验证**

示例 1: Echo 节点 (简单)
- [ ] ✅ 完整源代码 (<50 LOC)
- [ ] ✅ 可直接编译运行
- [ ] ✅ 参数定义 (1 个必需参数: message)
- [ ] ✅ 输出定义 (1 个输出: echo)
- [ ] ✅ 单元测试代码
- [ ] ✅ YAML 使用示例

示例 2: Greeter 节点 (中等)
- [ ] ✅ 引用 Story 4.4 完整实现
- [ ] ✅ 参数定义 (3 个参数: name, language, time_of_day)
- [ ] ✅ 枚举验证示例
- [ ] ✅ 默认值处理
- [ ] ✅ 多输出示例 (3 个输出)
- [ ] ✅ 时间处理逻辑
- [ ] ✅ 单元测试代码

示例 3: HTTP Request 节点 (生产级)
- [ ] ✅ 完整源代码 (~300 LOC)
- [ ] ✅ 参数定义 (8+ 参数)
- [ ] ✅ 上下文取消处理
- [ ] ✅ 超时处理
- [ ] ✅ 错误分类 (永久 vs 临时)
- [ ] ✅ 重试策略集成
- [ ] ✅ 复杂参数处理 (headers, body)
- [ ] ✅ 单元测试代码

**AC3: 参数验证章节验证**
- [ ] ✅ 输入输出 Schema 定义方法
- [ ] ✅ 6 种参数类型验证规则 (string, int, float, bool, object, array)
- [ ] ✅ Pattern 验证 (正则表达式)
- [ ] ✅ Enum 验证 (枚举值)
- [ ] ✅ MinValue/MaxValue 验证 (数值范围)
- [ ] ✅ Required/Default 验证
- [ ] ✅ ValidateInputs() 函数使用说明
- [ ] ✅ 5+ 参数验证示例:
  - [ ] 邮箱格式验证
  - [ ] URL 格式验证
  - [ ] 端口范围验证 (1-65535)
  - [ ] 文件路径格式验证
  - [ ] HTTP 方法枚举验证
- [ ] ✅ 自定义验证逻辑实现方法

**AC4: 错误处理和日志章节验证**
- [ ] ✅ 错误处理方法说明
- [ ] ✅ Temporal 错误分类 (ApplicationError vs TemporaryError)
- [ ] ✅ 永久错误和临时错误区分:
  - [ ] 永久错误类型列表 (参数验证、资源不存在、权限不足)
  - [ ] 临时错误类型列表 (网络超时、服务不可用、速率限制)
- [ ] ✅ ApplicationFailure 使用说明
- [ ] ✅ 日志记录最佳实践:
  - [ ] NodeResult.AddLog() 使用
  - [ ] 记录关键执行步骤
  - [ ] 参数值记录 (脱敏处理)
  - [ ] 错误上下文记录
- [ ] ✅ 3+ 错误处理示例:
  - [ ] 参数验证错误示例
  - [ ] 网络超时错误示例
  - [ ] 权限不足错误示例
- [ ] ✅ Temporal UI 日志查看说明

**AC5: 测试最佳实践章节验证**
- [ ] ✅ 节点测试最佳实践说明
- [ ] ✅ 单元测试编写方法:
  - [ ] 测试每个接口方法
  - [ ] 测试 Execute 正常路径
  - [ ] 测试 Execute 异常路径
  - [ ] 测试参数验证
  - [ ] 测试上下文取消
- [ ] ✅ 表驱动测试 (table-driven tests) 说明和示例
- [ ] ✅ 测试用例设计指南:
  - [ ] 正常输入测试 (happy path)
  - [ ] 边界值测试
  - [ ] 异常输入测试
  - [ ] 并发安全测试
- [ ] ✅ 集成测试方法 (PluginManager 加载)
- [ ] ✅ 测试覆盖率目标说明 (>80%)
- [ ] ✅ 完整测试代码示例

**AC6: 编译和部署章节验证**
- [ ] ✅ 节点打包和分发方式说明
- [ ] ✅ 编译前置要求:
  - [ ] Go 版本匹配要求 (与 Agent 一致)
  - [ ] CGO 启用要求 (CGO_ENABLED=1)
  - [ ] 平台限制说明 (Linux/macOS only)
- [ ] ✅ 编译命令: `go build -buildmode=plugin -o node.so main.go`
- [ ] ✅ 编译验证步骤:
  - [ ] 检查 .so 文件大小
  - [ ] 使用 `file` 命令验证文件类型
  - [ ] 使用 `nm` 命令检查 Register 符号
- [ ] ✅ 插件部署流程 (4 个步骤):
  - [ ] 复制 .so 文件到插件目录
  - [ ] 检查文件权限
  - [ ] Agent 热加载或重启
  - [ ] 验证插件加载成功
- [ ] ✅ 热加载机制说明 (fsnotify)
- [ ] ✅ 版本管理策略 (文件名包含版本)
- [ ] ✅ Makefile 模板

### 技术准确性验证

**Go 版本和平台:**
- [ ] ✅ Go 版本要求明确 (1.22.0+)
- [ ] ✅ Go 版本匹配要求详细说明（插件与 Agent 必须一致）
- [ ] ✅ CGO 要求明确 (CGO_ENABLED=1)
- [ ] ✅ 平台限制明确 (Linux/macOS，Windows 不支持)
- [ ] ✅ 提供版本验证命令

**文件路径正确性:**
- [ ] ✅ 所有示例代码路径存在或标注为示例
- [ ] ✅ 插件部署路径正确 (/opt/waterflow/plugins/)
- [ ] ✅ 引用的文档路径有效
- [ ] ✅ 示例节点目录结构正确

**命令正确性:**
- [ ] ✅ 所有编译命令可执行
- [ ] ✅ 所有验证命令有效
- [ ] ✅ 所有部署命令正确
- [ ] ✅ 所有测试命令可运行

### 引用和链接验证

**ADR 引用:**
- [ ] ✅ ADR-0003 (插件化节点系统) 正确引用
- [ ] ✅ ADR-0002 (单节点执行模式) 正确引用
- [ ] ✅ ADR 链接有效

**Story 引用:**
- [ ] ✅ Story 3.1 (节点接口设计) 引用
- [ ] ✅ Story 4.1 (Plugin Manager) 引用
- [ ] ✅ Story 4.2 (参数验证) 引用
- [ ] ✅ Story 4.3 (重试策略) 引用
- [ ] ✅ Story 4.4 (Greeter 示例) 引用和链接有效

**示例代码引用:**
- [ ] ✅ examples/plugins/template/ 引用
- [ ] ✅ examples/plugins/echo/ 引用
- [ ] ✅ examples/plugins/greeter/ 引用
- [ ] ✅ 示例代码路径正确

**外部文档引用:**
- [ ] ✅ Go Plugin 官方文档链接有效
- [ ] ✅ Temporal Go SDK 文档链接有效
- [ ] ✅ JSON Schema 文档链接有效
- [ ] ✅ 所有外部链接可访问

### 一致性验证

**术语一致性:**
- [ ] ✅ 节点 (Node) vs 插件 (Plugin) 术语一致使用
- [ ] ✅ ParamSpec vs Schema 术语统一
- [ ] ✅ 错误分类术语与 Story 4.3 一致
- [ ] ✅ 专有名词首次出现有解释

**代码风格一致性:**
- [ ] ✅ 所有 Go 代码使用 gofmt 格式化
- [ ] ✅ 所有代码注释风格一致
- [ ] ✅ 所有命名规范统一 (驼峰、下划线)
- [ ] ✅ 所有错误处理模式一致

**文档风格一致性:**
- [ ] ✅ 中英文混排规范一致
- [ ] ✅ 代码块语法高亮正确 (```go, ```yaml, ```bash)
- [ ] ✅ 表格格式统一
- [ ] ✅ 列表格式统一 (缩进、符号)
- [ ] ✅ 标题层级正确 (H1, H2, H3)

### 可执行性验证

**代码可编译性验证:**

```bash
# 提取并编译示例 1: Echo 节点
mkdir -p /tmp/test-echo
# 从文档提取代码到 /tmp/test-echo/main.go
cd /tmp/test-echo
go mod init github.com/Websoft9/waterflow/examples/plugins/echo
go mod tidy
go build -buildmode=plugin -o echo.so main.go
# 预期: 编译成功，生成 echo.so

# 提取并编译示例 3: HTTP Request 节点
mkdir -p /tmp/test-http
# 从文档提取代码到 /tmp/test-http/main.go
cd /tmp/test-http
go mod init github.com/Websoft9/waterflow/examples/plugins/http
go mod tidy
go build -buildmode=plugin -o http.so main.go
# 预期: 编译成功，生成 http.so
```

**测试代码可运行性验证:**

```bash
# 提取测试代码并运行
cd /tmp/test-echo
# 从文档提取测试代码到 main_test.go
go test -v
# 预期: 所有测试通过

cd /tmp/test-http
go test -v
# 预期: 所有测试通过
```

**命令可执行性验证:**

```bash
# 验证所有 bash 命令示例
# 提取所有 ```bash 代码块
# 在安全的测试环境执行
# 预期: 无语法错误，命令存在
```

### 用户验收测试

**开发者视角验证 (UAT):**
- [ ] ✅ 新手开发者可在 30 分钟内创建首个节点（使用 Quick Start）
- [ ] ✅ 有经验开发者可在 2 小时内创建生产级节点
- [ ] ✅ 所有常见问题都有解决方案（FAQ 章节）
- [ ] ✅ 文档可独立使用，无需额外查找资料
- [ ] ✅ 代码示例可直接复制粘贴使用

**文档可读性验证:**
- [ ] ✅ 章节结构清晰，逻辑连贯
- [ ] ✅ 目录完整，易于导航
- [ ] ✅ 代码示例有详细注释
- [ ] ✅ 复杂概念有图表或示意图辅助
- [ ] ✅ 术语在首次出现时有清晰解释
- [ ] ✅ 每个章节有明确的学习目标

**文档实用性验证:**
- [ ] ✅ 提供可复制的代码示例
- [ ] ✅ 提供 Makefile 模板简化编译
- [ ] ✅ 提供常见错误和解决方案
- [ ] ✅ 提供调试技巧和最佳实践
- [ ] ✅ 提供性能优化建议

### 自动化验证脚本

**创建验证脚本 (scripts/validate-node-guide.sh):**

```bash
#!/bin/bash
# 验证节点开发指南文档质量

set -e

DOC_FILE="/data/Waterflow/docs/guides/node-development.md"

echo "=== 验证节点开发指南文档 ==="

# 1. 文档存在性
if [ ! -f "$DOC_FILE" ]; then
    echo "❌ 文档不存在: $DOC_FILE"
    exit 1
fi
echo "✅ 文档存在"

# 2. 文档长度
LINES=$(wc -l < "$DOC_FILE")
if [ $LINES -lt 2000 ]; then
    echo "❌ 文档行数不足: $LINES < 2000"
    exit 1
fi
echo "✅ 文档长度: $LINES 行"

# 3. 章节完整性
CHAPTERS=$(grep -c "^##" "$DOC_FILE")
if [ $CHAPTERS -lt 12 ]; then
    echo "❌ 章节数量不足: $CHAPTERS < 12"
    exit 1
fi
echo "✅ 章节数量: $CHAPTERS 个"

# 4. 代码示例数量
GO_BLOCKS=$(grep -c "^\`\`\`go" "$DOC_FILE" || true)
if [ $GO_BLOCKS -lt 20 ]; then
    echo "⚠️ Go 代码块较少: $GO_BLOCKS < 20"
else
    echo "✅ Go 代码块: $GO_BLOCKS 个"
fi

YAML_BLOCKS=$(grep -c "^\`\`\`yaml" "$DOC_FILE" || true)
if [ $YAML_BLOCKS -lt 10 ]; then
    echo "⚠️ YAML 示例较少: $YAML_BLOCKS < 10"
else
    echo "✅ YAML 示例: $YAML_BLOCKS 个"
fi

# 5. AC 章节存在性
for ac in "Core Concepts" "Node Examples" "Parameter Validation" "Error Handling" "Testing" "Build and Deploy"; do
    if ! grep -q "$ac" "$DOC_FILE"; then
        echo "❌ 缺少章节: $ac"
        exit 1
    fi
done
echo "✅ 所有 AC 章节存在"

# 6. 关键术语检查
for term in "ParamSpec" "NodeResult" "ValidateInputs" "Register" "Go Plugin"; do
    if ! grep -q "$term" "$DOC_FILE"; then
        echo "⚠️ 缺少关键术语: $term"
    fi
done

echo "✅ 文档验证通过"
```

**运行验证:**

```bash
chmod +x scripts/validate-node-guide.sh
./scripts/validate-node-guide.sh

# 预期输出:
# === 验证节点开发指南文档 ===
# ✅ 文档存在
# ✅ 文档长度: 2347 行
# ✅ 章节数量: 14 个
# ✅ Go 代码块: 28 个
# ✅ YAML 示例: 13 个
# ✅ 所有 AC 章节存在
# ✅ 文档验证通过
```

### 最终检查清单

**提交前必须确认:**

- [ ] ✅ 所有 6 个 AC (AC1-AC6) 完全达成
- [ ] ✅ 文档总长度 >2000 行
- [ ] ✅ 包含 3 个完整示例节点（Echo, Greeter, HTTP Request）
- [ ] ✅ 所有代码示例已验证可编译运行
- [ ] ✅ 所有链接和引用已验证有效
- [ ] ✅ 所有命令已在 Linux 环境验证
- [ ] ✅ 文档已进行拼写检查（英文和中文）
- [ ] ✅ 文档已进行格式检查（Markdown lint）
- [ ] ✅ 文档已由至少一位第三方开发者试用并反馈
- [ ] ✅ FAQ 章节包含常见问题（至少 10 个）
- [ ] ✅ Troubleshooting 章节包含调试技巧
- [ ] ✅ Best Practices 章节包含代码组织、性能、安全建议
- [ ] ✅ 自动化验证脚本运行通过

**文档发布检查:**

- [ ] ✅ 版本号已更新 (v2.0)
- [ ] ✅ 更新日期已标注 (2025-12-31)
- [ ] ✅ Changelog 已记录主要变更
- [ ] ✅ 与旧版本的差异已在文档中说明
- [ ] ✅ 内部链接已更新（指向新章节）
- [ ] ✅ 外部引用已通知相关团队（如有必要）

---

**验收标准达成度:**

完成以上所有检查清单后，文档应达到以下质量标准：
- 完整性：100% (所有 AC 达成)
- 准确性：100% (所有代码可编译，所有命令可执行)
- 一致性：100% (术语、风格、格式统一)
- 实用性：100% (开发者可独立使用文档完成节点开发)
- 可读性：≥95% (结构清晰，示例丰富，解释详细)
