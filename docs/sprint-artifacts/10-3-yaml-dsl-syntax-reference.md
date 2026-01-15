# Story 10.3: YAML DSL 语法参考

**Epic**: Epic 10 - 完整文档体系  
**Status**: ready-for-dev  
**Created**: 2025-01-15  
**Dependencies**: 
- ✅ Story 1.3 - YAML DSL 解析和验证 (DSL 核心结构定义)
- ✅ Story 1.4 - 表达式引擎和变量系统 (表达式语法实现)
- ✅ Story 1.5 - 条件执行和控制流 (if 条件、Job依赖、输出引用)
- ✅ Story 1.6 - Matrix 并行执行策略 (Matrix 语法实现)
- ✅ Story 1.7 - 超时和重试策略 (超时/重试配置)

---

## User Story

As a **工作流用户**,  
I want **完整的 YAML DSL 语法文档 (基于 ADR-0004 和 ADR-0005)**,  
So that **编写正确的工作流并理解所有语法特性**。

---

## Acceptance Criteria

### AC1: 顶层结构和工作流定义

**Given** 用户开始编写工作流文件  
**When** 查阅语法文档顶层结构部分  
**Then** 文档清晰说明所有顶层字段 (name, on, vars, env, jobs)  
**And** 说明每个字段的类型、必需性、默认值  
**And** 说明 `name` 字段的命名规范和约束  
**And** 说明 `on` 触发器语法 (push/schedule/webhook 等)  
**And** 说明 `vars` 全局变量定义语法 (支持嵌套对象、数组)  
**And** 说明 `env` 环境变量语法 (支持表达式求值)  
**And** 提供完整的顶层结构示例  
**And** 交叉引用 ADR-0004 (YAML DSL 设计决策)

### AC2: Job 结构和配置

**Given** 用户编写 Job 定义  
**When** 查阅 Job 语法文档  
**Then** 文档完整说明 Job 所有字段:  
**And** `runs-on` - Task Queue 直接映射机制 (ADR-0006),说明如何路由任务  
**And** `needs` - Job 依赖关系,说明并行/串行执行逻辑  
**And** `if` - Job 级条件执行语法  
**And** `timeout-minutes` - Job 超时配置 (默认 360 分钟,ADR-0002 单节点执行模式)  
**And** `strategy.matrix` - Matrix 并行策略语法和展开规则  
**And** `strategy.max-parallel` - 并发控制配置  
**And** `strategy.fail-fast` - 失败策略 (默认 true)  
**And** `continue-on-error` - 失败容忍语法  
**And** `outputs` - Job 输出定义语法  
**And** `env` - Job 级环境变量 (继承和覆盖规则)  
**And** `steps` - Step 列表语法  
**And** 提供 Job 完整示例 (普通 Job、Matrix Job、依赖 Job)  
**And** 说明 runs-on 如何映射到 Temporal Task Queue (零配置路由)

### AC3: Step 结构和节点调用

**Given** 用户编写 Step 定义  
**When** 查阅 Step 语法文档  
**Then** 文档完整说明 Step 所有字段:  
**And** `id` - Step 标识符,用于输出引用  
**And** `name` - Step 显示名称 (可包含表达式)  
**And** `uses` - 节点调用语法 `<node>@<version>` (如 exec/shell@v1)  
**And** `with` - 节点参数传递语法 (支持表达式求值)  
**And** `if` - Step 级条件执行语法  
**And** `timeout-minutes` - Step 超时配置 (单节点执行模式,ADR-0002)  
**And** `retry-strategy` - 重试策略语法 (max-attempts, initial-interval, backoff-coefficient, max-interval)  
**And** `continue-on-error` - Step 失败容忍  
**And** `env` - Step 级环境变量 (最高优先级)  
**And** 提供 Step 完整示例 (shell 执行、Docker 管理、HTTP 请求)  
**And** 说明每个 Step = 1 个 Temporal Activity 调用 (ADR-0002)

### AC4: 变量引用和表达式语法

**Given** 用户需要动态引用变量  
**When** 查阅表达式语法文档  
**Then** 文档完整说明 `${{ expression }}` 语法 (ADR-0005)  
**And** 说明所有上下文变量:  
  - `vars.*` - 全局变量引用 (workflow.vars)  
  - `env.*` - 环境变量引用  
  - `matrix.*` - Matrix 变量引用 (仅 Matrix Job 可用)  
  - `steps.<id>.outputs.*` - Step 输出引用  
  - `needs.<job>.outputs.*` - Job 依赖输出引用  
  - `workflow.*` - 工作流元数据 (name, id)  
  - `job.*` - Job 元数据 (name, status)  
  - `runner.*` - Agent 信息 (os, arch)  
  - `secrets.*` - 密钥引用 (通过 SecretProvider 接口获取)  
**And** 说明表达式运算符:  
  - 算术: `+`, `-`, `*`, `/`, `%`, `**`  
  - 比较: `==`, `!=`, `>`, `<`, `>=`, `<=`  
  - 逻辑: `&&`, `||`, `!`  
  - 字符串: `contains`, `startsWith`, `endsWith`  
**And** 说明内置函数:  
  - `len(array)` - 数组长度  
  - `upper(str)`, `lower(str)`, `trim(str)` - 字符串处理  
  - `split(str, sep)`, `join(array, sep)` - 分割/连接  
  - `format(template, args...)` - 格式化  
  - `toJSON(obj)`, `fromJSON(str)` - JSON 转换  
  - `success()`, `failure()`, `cancelled()`, `always()` - 条件函数  
**And** 提供表达式完整示例 (所有运算符和函数)  
**And** 说明表达式求值规则和优先级  
**And** 说明表达式沙箱限制 (无文件/网络访问)

### AC5: Matrix 并行策略语法

**Given** 用户需要并行执行多个相似任务  
**When** 查阅 Matrix 语法文档  
**Then** 文档完整说明 Matrix 策略:  
**And** `strategy.matrix` - Matrix 维度定义语法 (map[string][]interface{})  
**And** 笛卡尔积展开规则 (多维组合逻辑)  
**And** Matrix 组合数限制 (最大 256 个组合)  
**And** `max-parallel` - 并发控制 (默认无限制,全并行)  
**And** `fail-fast` - 失败策略 (默认 true,任一失败立即取消其他)  
**And** Matrix 变量引用语法 `${{ matrix.dimension }}`  
**And** Matrix 实例独立追踪 (每个实例 = 1 个 Activity)  
**And** 提供 Matrix 完整示例:  
  - 单维 Matrix (服务器列表)  
  - 多维 Matrix (OS x 版本)  
  - max-parallel 限流示例  
  - fail-fast 策略对比  
**And** 说明 MVP 暂不支持 include/exclude (提示替代方案)  
**And** 交叉引用 Story 1.6 实现详情

### AC6: 超时和重试策略配置

**Given** 用户需要配置超时和重试  
**When** 查阅超时/重试语法文档  
**Then** 文档完整说明超时配置 (ADR-0002 单节点执行模式):  
**And** Job 级 `timeout-minutes` 语法 (默认 360 分钟)  
**And** Step 级 `timeout-minutes` 语法 (继承 Job 超时或独立配置)  
**And** 超时时 Temporal 自动终止 Activity  
**And** 文档完整说明重试策略配置:  
**And** `retry-strategy.max-attempts` - 最大重试次数 (默认 3)  
**And** `retry-strategy.initial-interval` - 首次重试间隔 (默认 1s)  
**And** `retry-strategy.backoff-coefficient` - 退避系数 (默认 2.0,指数退避)  
**And** `retry-strategy.max-interval` - 最大重试间隔 (默认 60s)  
**And** 说明每个 Step 独立配置超时/重试 (单节点执行模式优势)  
**And** 提供超时/重试完整示例:  
  - HTTP 请求重试 (临时故障恢复)  
  - Docker 部署超时配置  
  - 分布式任务重试策略  
**And** 说明不可重试错误 (参数错误、404 等)  
**And** 交叉引用 Story 1.7 实现详情

### AC7: 完整工作流示例和最佳实践

**Given** 用户理解所有语法特性  
**When** 查阅完整示例和最佳实践  
**Then** 文档提供至少 3 个完整的工作流示例:  
**And** 示例 1: 简单工作流 (单 Job,多 Step,变量引用)  
**And** 示例 2: 复杂工作流 (多 Job 依赖,条件执行,输出引用)  
**And** 示例 3: Matrix 并行工作流 (多服务器并行,fail-fast 策略)  
**And** 每个示例有完整注释和执行流程说明  
**And** 文档说明常见模式和最佳实践:  
  - 命名规范 (workflow、job、step、variable)  
  - 变量作用域和优先级  
  - 环境变量继承规则 (workflow → job → step)  
  - Matrix 策略使用场景  
  - 超时/重试配置建议  
  - 条件执行常见模式 (if 语法技巧)  
  - 密钥安全引用 (secrets.* + SecretProvider)  
**And** 说明如何避免常见错误:  
  - 循环依赖检测  
  - 未定义变量引用  
  - Matrix 组合数超限  
  - 超时配置不合理  
**And** 交叉引用 ADR 文档 (ADR-0004, ADR-0005, ADR-0002, ADR-0006)  
**And** 提供语法速查表 (Quick Reference)

---

## Tasks / Subtasks

### Task 1: 分析现有 DSL 实现和测试数据 (AC1-AC7)
- [x] 读取 pkg/dsl/types.go 完整结构定义 (Workflow, Job, Step, Strategy, RetryStrategy)
- [x] 读取 examples/*.yaml 示例文件 (matrix.yaml, multi-server-health-check.yaml)
- [x] 分析 Story 1.3-1.7 的实现细节和 AC 规范
- [x] 确认 ADR-0004 (YAML DSL) 和 ADR-0005 (表达式语法) 不存在,需基于代码推断设计
- [x] 分析 pkg/dsl/expr_context.go 表达式上下文定义 (所有内置变量和函数)
- [x] 分析 semantic_search 结果中的 Matrix、超时、重试语法实现
- **输出**: DSL 完整特性清单,语法约束和最佳实践

### Task 2: 创建 YAML DSL 语法参考文档 (AC1-AC3)
- [ ] 创建 `/docs/reference/dsl-syntax.md` 文档
  - [ ] 顶层结构章节 (name, on, vars, env, jobs)
    - name 字段规范 (字符限制、命名建议)
    - on 触发器语法 (push, schedule, webhook)
    - vars 全局变量定义 (支持嵌套对象、数组、表达式)
    - env 环境变量语法 (三级继承规则)
    - 完整顶层结构示例
  - [ ] Job 结构章节 (runs-on, needs, if, timeout-minutes, strategy, env, steps, outputs, continue-on-error)
    - runs-on: Task Queue 直接映射机制 (ADR-0006 零配置路由)
    - needs: Job 依赖关系和并行执行逻辑
    - if: Job 级条件执行 (表达式求值)
    - timeout-minutes: Job 超时配置 (默认 360 分钟,单节点执行模式)
    - strategy.matrix: Matrix 策略语法 (展开规则、max-parallel、fail-fast)
    - outputs: Job 输出定义 (用于 needs 引用)
    - 普通 Job、Matrix Job、依赖 Job 完整示例
  - [ ] Step 结构章节 (id, name, uses, with, if, timeout-minutes, retry-strategy, env, continue-on-error)
    - uses: 节点调用语法 `<node>@<version>`
    - with: 节点参数传递 (支持表达式)
    - retry-strategy: 重试策略配置 (max-attempts, initial-interval, backoff-coefficient, max-interval)
    - 每个 Step = 1 个 Activity 调用 (ADR-0002 单节点执行模式)
    - shell、Docker、HTTP 请求 Step 完整示例
- **输出**: 基础语法结构文档 (顶层、Job、Step)

### Task 3: 创建表达式语法参考 (AC4)
- [ ] 创建表达式语法章节 `/docs/reference/dsl-syntax.md#expressions`
  - [ ] 表达式语法基础
    - `${{ expression }}` 语法格式
    - 表达式求值规则 (运行时求值、沙箱限制)
    - 表达式可用位置 (所有字段值、with 参数、if 条件)
  - [ ] 上下文变量参考
    - `vars.*` - 全局变量引用 (嵌套访问、数组索引)
    - `env.*` - 环境变量引用
    - `matrix.*` - Matrix 变量引用 (仅 Matrix Job)
    - `steps.<id>.outputs.*` - Step 输出引用
    - `needs.<job>.outputs.*` - Job 依赖输出引用
    - `workflow.*` - 工作流元数据 (name, id)
    - `job.*` - Job 元数据 (name, status)
    - `runner.*` - Agent 信息 (os, arch)
    - `secrets.*` - 密钥引用 (通过 SecretProvider 接口)
  - [ ] 运算符参考
    - 算术运算符: `+`, `-`, `*`, `/`, `%`, `**` (幂运算)
    - 比较运算符: `==`, `!=`, `>`, `<`, `>=`, `<=`
    - 逻辑运算符: `&&`, `||`, `!`
    - 字符串运算符: `contains`, `startsWith`, `endsWith`
  - [ ] 内置函数参考
    - `len(array)` - 数组长度
    - `upper(str)`, `lower(str)`, `trim(str)` - 字符串处理
    - `split(str, sep)`, `join(array, sep)` - 分割/连接
    - `format(template, args...)` - 格式化字符串
    - `toJSON(obj)`, `fromJSON(str)` - JSON 转换
    - `success()`, `failure()`, `cancelled()`, `always()` - 条件函数
  - [ ] 表达式完整示例
    - 变量引用示例 (所有上下文变量)
    - 运算符示例 (所有运算符组合)
    - 函数调用示例 (所有内置函数)
    - 复杂表达式示例 (嵌套、多运算符)
- **输出**: 表达式语法完整参考 (ADR-0005 规范)

### Task 4: 创建 Matrix 并行策略文档 (AC5)
- [ ] 创建 Matrix 策略章节 `/docs/reference/dsl-syntax.md#matrix-strategy`
  - [ ] Matrix 基础语法
    - `strategy.matrix` 字段定义 (map[string][]interface{})
    - 笛卡尔积展开规则 (多维组合算法)
    - Matrix 组合数限制 (最大 256 个)
  - [ ] Matrix 配置参数
    - `max-parallel` - 并发控制 (默认无限制)
    - `fail-fast` - 失败策略 (默认 true)
  - [ ] Matrix 变量引用
    - `${{ matrix.dimension }}` 语法
    - Matrix 变量作用域 (仅当前实例)
  - [ ] Matrix 实例追踪
    - 每个实例 = 1 个独立 Activity
    - 独立状态追踪和日志
  - [ ] Matrix 完整示例
    - 单维 Matrix 示例 (服务器列表并行)
    - 多维 Matrix 示例 (OS x 版本组合)
    - max-parallel 限流示例
    - fail-fast 策略对比 (true vs false)
  - [ ] MVP 限制说明
    - include/exclude 暂不支持 (提示使用多个 Job 替代)
- **输出**: Matrix 策略完整文档 (基于 Story 1.6)

### Task 5: 创建超时和重试配置文档 (AC6)
- [ ] 创建超时/重试配置章节 `/docs/reference/dsl-syntax.md#timeout-and-retry`
  - [ ] 超时配置文档
    - Job 级 `timeout-minutes` (默认 360 分钟)
    - Step 级 `timeout-minutes` (继承或独立配置)
    - 超时机制 (Temporal 自动终止 Activity)
    - 超时后资源清理 (进程终止、状态记录)
  - [ ] 重试策略文档
    - `retry-strategy.max-attempts` - 最大重试次数 (默认 3)
    - `retry-strategy.initial-interval` - 首次重试间隔 (默认 1s)
    - `retry-strategy.backoff-coefficient` - 退避系数 (默认 2.0)
    - `retry-strategy.max-interval` - 最大重试间隔 (默认 60s)
    - 指数退避算法说明
    - 可重试 vs 不可重试错误 (临时故障 vs 永久错误)
  - [ ] 单节点执行模式优势
    - 每个 Step 独立配置超时/重试 (ADR-0002)
    - 细粒度控制 (HTTP 请求 vs Docker 部署不同策略)
  - [ ] 超时/重试完整示例
    - HTTP 请求重试配置 (网络临时故障)
    - Docker 部署超时配置 (长时间操作)
    - 分布式任务重试策略 (跨服务器容错)
- **输出**: 超时/重试策略完整文档 (基于 Story 1.7)

### Task 6: 创建完整工作流示例和最佳实践 (AC7)
- [ ] 创建完整示例章节 `/docs/reference/dsl-syntax.md#complete-examples`
  - [ ] 示例 1: 简单工作流
    - 单 Job,多 Step
    - 变量引用和环境变量
    - shell 执行和输出引用
    - 完整注释和执行流程说明
  - [ ] 示例 2: 复杂工作流
    - 多 Job 依赖 (needs 字段)
    - 条件执行 (if 语法)
    - Job/Step 输出引用
    - 失败容忍 (continue-on-error)
    - 完整注释和 DAG 执行图
  - [ ] 示例 3: Matrix 并行工作流
    - 多服务器并行健康检查
    - Matrix 变量引用
    - fail-fast 策略
    - 结果聚合
    - 完整注释和并行执行说明
- [ ] 创建最佳实践章节 `/docs/reference/dsl-syntax.md#best-practices`
  - [ ] 命名规范
    - workflow name 规范 (小写、连字符)
    - job/step ID 规范 (字母数字、下划线)
    - 变量命名规范 (语义化、避免保留字)
  - [ ] 变量和环境变量
    - 变量作用域 (vars vs env vs matrix)
    - 环境变量优先级 (step > job > workflow)
    - 密钥安全引用 (secrets.* + SecretProvider 接口)
  - [ ] Matrix 策略
    - 使用场景 (并行任务、跨环境测试)
    - 组合数限制建议 (避免超过 256)
    - max-parallel 设置建议 (避免资源耗尽)
  - [ ] 超时/重试
    - 超时配置建议 (Job vs Step 超时)
    - 重试策略建议 (临时故障 vs 永久错误)
    - 避免过度重试 (max-attempts 建议值)
  - [ ] 条件执行
    - if 语法常见模式 (环境判断、状态判断)
    - 避免复杂条件 (拆分为多个 Step)
- [ ] 创建常见错误章节 `/docs/reference/dsl-syntax.md#common-errors`
  - [ ] 循环依赖 (needs 字段错误)
  - [ ] 未定义变量引用 (vars/steps/needs 不存在)
  - [ ] Matrix 组合数超限 (超过 256)
  - [ ] 超时配置不合理 (过短或过长)
  - [ ] 表达式语法错误 (引号、括号不匹配)
- [ ] 创建语法速查表 `/docs/reference/dsl-syntax.md#quick-reference`
  - [ ] 所有顶层字段速查
  - [ ] 所有 Job 字段速查
  - [ ] 所有 Step 字段速查
  - [ ] 所有表达式函数速查
  - [ ] 常用模式代码片段
- **输出**: 完整示例、最佳实践、常见错误、速查表

### Task 7: 交叉引用和文档优化
- [ ] 添加 ADR 交叉引用
  - [ ] ADR-0004 (YAML DSL 设计决策) - 注意:实际不存在,基于代码推断设计理念
  - [ ] ADR-0005 (表达式语法设计) - 注意:实际不存在,基于代码推断设计理念
  - [ ] ADR-0002 (单节点执行模式) - 超时/重试独立配置
  - [ ] ADR-0006 (Task Queue 直接映射) - runs-on 路由机制
- [ ] 添加 Story 交叉引用
  - [ ] Story 1.3 - YAML DSL 解析和验证
  - [ ] Story 1.4 - 表达式引擎和变量系统
  - [ ] Story 1.5 - 条件执行和控制流
  - [ ] Story 1.6 - Matrix 并行执行策略
  - [ ] Story 1.7 - 超时和重试策略
- [ ] 添加代码示例交叉引用
  - [ ] pkg/dsl/types.go - 数据结构定义
  - [ ] examples/matrix.yaml - Matrix 示例
  - [ ] examples/workflows/multi-server-health-check.yaml - 复杂工作流示例
- [ ] 文档格式优化
  - [ ] 代码块语法高亮 (yaml, go)
  - [ ] 表格格式优化 (字段参考)
  - [ ] 目录生成 (TOC)
  - [ ] 锚点链接 (章节跳转)
- **输出**: 完整优化的 YAML DSL 语法参考文档

### Task 8: 文档验证和测试
- [ ] 语法准确性验证
  - [ ] 对比 pkg/dsl/types.go 结构定义 (所有字段完整)
  - [ ] 对比 pkg/dsl/expr_context.go 表达式上下文 (所有变量和函数)
  - [ ] 对比 examples/*.yaml 示例 (语法正确)
- [ ] 示例可执行性验证
  - [ ] 所有示例通过 YAML 语法验证
  - [ ] 所有示例通过 waterflow validate 验证
  - [ ] 所有示例可实际提交执行 (可选,需 Server 环境)
- [ ] 文档完整性检查
  - [ ] 所有 AC 覆盖 (AC1-AC7)
  - [ ] 所有字段有文档 (顶层、Job、Step)
  - [ ] 所有表达式特性有文档 (变量、运算符、函数)
  - [ ] 所有 ADR 正确引用
- **输出**: 验证通过的生产级文档

### Task 9: 更新 sprint-status.yaml 状态
- [ ] 将 Story 10-3-yaml-dsl-syntax-reference 状态从 backlog 更新为 ready-for-dev
- **输出**: sprint-status.yaml 更新完成

---

## Dev Notes

### 现有分析

**已完成的基础工作 (Epics 1-9):**
- ✅ **DSL 核心结构定义** (Story 1.3): pkg/dsl/types.go 完整实现
  - Workflow 结构: name, on, vars, env, jobs
  - Job 结构: runs-on, needs, if, timeout-minutes, strategy, env, steps, outputs, continue-on-error
  - Step 结构: id, name, uses, with, if, timeout-minutes, retry-strategy, env, continue-on-error
  - Strategy 结构: matrix, max-parallel, fail-fast, include, exclude
  - RetryStrategy 结构: max-attempts, initial-interval, backoff-coefficient, max-interval

- ✅ **表达式引擎实现** (Story 1.4): pkg/dsl/expr_context.go 完整实现
  - 上下文变量: workflow, job, steps, vars, env, matrix, runner, inputs, secrets, needs
  - 内置函数: len, upper, lower, trim, split, join, format, contains, startsWith, endsWith, toJSON, fromJSON, always
  - 条件函数: success, failure, cancelled

- ✅ **Matrix 策略实现** (Story 1.6): pkg/dsl/expander.go + pkg/dsl/executor.go
  - Matrix 展开器: 笛卡尔积算法,组合数限制 256
  - Matrix 执行器: 并发控制 (max-parallel), fail-fast 策略
  - Matrix 上下文: matrix.* 变量引用

- ✅ **超时和重试实现** (Story 1.7): pkg/dsl/types.go RetryStrategy 定义
  - Job/Step 超时配置: timeout-minutes
  - 重试策略配置: max-attempts, initial-interval, backoff-coefficient, max-interval
  - Temporal Activity 超时/重试集成

- ✅ **示例文件存在**: examples/matrix.yaml, examples/workflows/multi-server-health-check.yaml
  - matrix.yaml: 简单 Matrix 示例 (os x version)
  - multi-server-health-check.yaml: 复杂工作流示例 (Matrix 并行,变量引用,表达式)

**文档缺口 (Story 10.3 需要创建):**
- ❌ **ADR-0004 (YAML DSL 设计)** 不存在 - 需基于代码推断设计理念
- ❌ **ADR-0005 (表达式语法设计)** 不存在 - 需基于代码推断设计理念
- ❌ **完整的 YAML DSL 语法参考文档** - 本 Story 创建目标
- ✅ **快速开始文档** 已存在: docs/quick-start.md (但需要增强 DSL 语法引用)

### 优化建议

**文档结构建议:**
1. **创建独立的语法参考文档** `/docs/reference/dsl-syntax.md`
   - 不要合并到 quick-start.md (职责分离)
   - 参考文档应详尽、技术性强
   - 快速开始文档应简洁、示例驱动

2. **章节组织建议:**
   - 第 1 章: 概述 (YAML DSL 设计理念,基于 GitHub Actions 语法)
   - 第 2 章: 顶层结构 (name, on, vars, env, jobs)
   - 第 3 章: Job 结构 (所有 Job 字段)
   - 第 4 章: Step 结构 (所有 Step 字段)
   - 第 5 章: 表达式语法 (变量、运算符、函数)
   - 第 6 章: Matrix 并行策略 (完整 Matrix 语法)
   - 第 7 章: 超时和重试 (配置详解)
   - 第 8 章: 完整示例 (3 个完整工作流)
   - 第 9 章: 最佳实践 (命名、变量、Matrix、超时/重试、条件执行)
   - 第 10 章: 常见错误 (循环依赖、未定义引用、Matrix 超限)
   - 附录 A: 语法速查表
   - 附录 B: ADR 交叉引用

3. **示例驱动原则:**
   - 每个语法特性必须有完整示例
   - 示例必须可执行 (通过 waterflow validate)
   - 示例包含完整注释 (执行流程、输出结果)

4. **交叉引用策略:**
   - ADR 引用: 说明设计决策背景 (注意 ADR-0004/0005 不存在,基于代码推断)
   - Story 引用: 说明实现细节位置
   - 代码引用: 说明数据结构定义位置
   - 示例引用: 说明可运行示例文件位置

### 架构约束 (基于 ADR 和代码分析)

**YAML DSL 设计理念 (推断 ADR-0004):**
- 基于 GitHub Actions 语法设计,用户熟悉度高
- 声明式工作流定义,清晰易读
- 支持表达式系统 `${{ expression }}`,动态求值
- 支持 Matrix 并行策略,提高执行效率
- 支持超时/重试配置,增强可靠性

**表达式系统设计 (推断 ADR-0005):**
- 使用 `${{ }}` 语法,与 GitHub Actions 一致
- 基于 antonmedv/expr 引擎,安全沙箱求值
- 支持上下文变量 (vars, env, matrix, steps, needs, workflow, job, runner, secrets)
- 支持运算符 (算术、比较、逻辑、字符串)
- 支持内置函数 (字符串处理、数组操作、JSON 转换、条件判断)
- 表达式可用于所有字段值 (name, if, with 参数等)

**单节点执行模式 (ADR-0002):**
- 每个 Step = 1 个 Temporal Activity 调用
- 每个 Step 独立配置 timeout-minutes 和 retry-strategy
- 细粒度控制,不同任务不同策略
- Matrix 每个实例 = 1 个独立 Activity,独立超时/重试

**Task Queue 直接映射 (ADR-0006):**
- runs-on 字段直接映射到 Temporal Task Queue 名称
- 零配置路由,简化 Agent 注册和任务分发
- Temporal 原生负载均衡在 Task Queue 内的多个 Agent 间分发任务

### 时间估算

**Story 10.3 估算**: **16-20 小时 (2-3 工作日)**

**任务分解:**
- Task 1: 分析 DSL 实现和测试数据 - **2 小时** (已完成)
- Task 2: 创建基础语法文档 (顶层、Job、Step) - **4 小时**
- Task 3: 创建表达式语法参考 - **3 小时**
- Task 4: 创建 Matrix 策略文档 - **2 小时**
- Task 5: 创建超时/重试配置文档 - **2 小时**
- Task 6: 创建完整示例和最佳实践 - **4 小时**
- Task 7: 交叉引用和文档优化 - **2 小时**
- Task 8: 文档验证和测试 - **2 小时**
- Task 9: 更新 sprint-status.yaml - **0.5 小时**

**复杂度分析:**
- ✅ **低复杂度**: 所有实现已完成,只需文档编写
- ✅ **中等工作量**: 需要详尽的语法说明和示例
- ✅ **高质量要求**: 文档是用户直接使用的参考,必须准确完整

---

## References

### 源文档 (PRD, Architecture, Epics)
1. [PRD - 产品需求文档](../../docs/prd.md) - NFR7: 文档完善性,完整的 YAML DSL 语法参考
2. [Architecture - 架构文档](../../docs/architecture.md) - DSL 引擎设计,表达式系统,单节点执行模式
3. [Epics - Epic 10 Story 10.3](../../docs/epics.md#story-103-yaml-dsl-语法参考-adr-0004) - Story 定义和 AC 规范

### 实现参考 (Stories 1.3-1.7)
4. [Story 1.3 - YAML DSL 解析和验证](./1-3-yaml-dsl-parsing-and-validation.md) - DSL 核心结构定义,验证规则
5. [Story 1.4 - 表达式引擎和变量系统](./1-4-expression-engine-and-variables.md) - 表达式语法实现,上下文变量,内置函数
6. [Story 1.5 - 条件执行和控制流](./1-5-conditional-execution-and-control-flow.md) - if 条件语法,Job 依赖,输出引用
7. [Story 1.6 - Matrix 并行执行策略](./1-6-matrix-parallel-execution.md) - Matrix 策略语法,笛卡尔积展开,fail-fast 策略
8. [Story 1.7 - 超时和重试策略](./1-7-timeout-and-retry-strategies.md) - 超时/重试配置,单节点执行模式

### 代码实现 (DSL 核心)
9. [pkg/dsl/types.go](../../pkg/dsl/types.go) - Workflow, Job, Step, Strategy, RetryStrategy 结构定义
10. [pkg/dsl/expr_context.go](../../pkg/dsl/expr_context.go) - EvalContext 表达式上下文,所有变量和函数
11. [pkg/dsl/expander.go](../../pkg/dsl/expander.go) - Matrix 展开器,笛卡尔积算法
12. [pkg/dsl/semantic_validator.go](../../pkg/dsl/semantic_validator.go) - 语义验证规则,Matrix 验证,超时验证

### 示例文件 (实际工作流)
13. [examples/matrix.yaml](../../examples/matrix.yaml) - 简单 Matrix 示例 (os x version)
14. [examples/workflows/multi-server-health-check.yaml](../../examples/workflows/multi-server-health-check.yaml) - 复杂工作流示例 (Matrix 并行,变量引用,表达式)
15. [examples/hello-world.yaml](../../examples/hello-world.yaml) - 简单工作流示例 (单 Job,多 Step)

### 架构决策 (ADR - 注意:ADR-0004/0005 不存在,基于代码推断)
16. ADR-0004 (YAML DSL 设计) - **不存在,基于代码推断**: GitHub Actions 语法设计,声明式工作流定义
17. ADR-0005 (表达式语法设计) - **不存在,基于代码推断**: `${{ }}` 语法,antonmedv/expr 引擎,安全沙箱
18. ADR-0002 (单节点执行模式) - 每个 Step = 1 个 Activity 调用,独立超时/重试配置
19. ADR-0006 (Task Queue 直接映射) - runs-on 字段直接映射到 Task Queue,零配置路由

### 外部参考 (GitHub Actions, OpenAPI, RFC)
20. [GitHub Actions Workflow Syntax](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions) - YAML DSL 语法参考
21. [antonmedv/expr](https://github.com/antonmedv/expr) - Go 表达式引擎文档
22. [YAML 1.2 Specification](https://yaml.org/spec/1.2/spec.html) - YAML 语法规范

---

## File List

**新增文件:**
- `/docs/reference/dsl-syntax.md` - YAML DSL 语法参考文档 (本 Story 创建目标)

**修改文件:**
- `/docs/sprint-artifacts/sprint-status.yaml` - Story 10-3 状态更新 (backlog → ready-for-dev)

**参考文件:**
- `/pkg/dsl/types.go` - 数据结构定义 (Workflow, Job, Step, Strategy, RetryStrategy)
- `/pkg/dsl/expr_context.go` - 表达式上下文 (EvalContext, 变量, 函数)
- `/pkg/dsl/expander.go` - Matrix 展开器
- `/pkg/dsl/semantic_validator.go` - 语义验证器
- `/examples/matrix.yaml` - Matrix 示例
- `/examples/workflows/multi-server-health-check.yaml` - 复杂工作流示例
- `/docs/quick-start.md` - 快速开始文档 (需增强 DSL 语法引用)

---

## Change Log

**2025-01-15 - Story 创建 (100% 完整上下文)**
- ✅ 分析所有 DSL 实现代码 (pkg/dsl/*.go)
- ✅ 分析所有示例文件 (examples/*.yaml)
- ✅ 分析 Story 1.3-1.7 实现细节
- ✅ 确认 ADR-0004/0005 不存在 (需基于代码推断设计理念)
- ✅ 创建 7 个验收标准 (顶层结构、Job、Step、表达式、Matrix、超时/重试、示例)
- ✅ 创建 9 个详细任务 (66 个子任务)
- ✅ 添加 22 个参考文档链接
- ✅ 提供架构约束和时间估算
- ✅ Story 状态: ready-for-dev

---
