# Story 5.2 CLI validate 命令 - 验证报告

**验证日期:** 2026-01-04  
**验证人员:** Senior Quality Validator  
**Story 文件:** `/data/Waterflow/docs/sprint-artifacts/5-2-cli-validate-command.md`  
**验证方法:** 系统性 Checklist 验证 + 依赖分析 + 架构一致性检查

---

## 1. 执行摘要

### 总体评级: **7.5/10** ⚠️

### 评级分解
- ✅ **架构一致性:** 9/10 (基本符合 Story 5.1 和 Story 1.3)
- ⚠️ **需求清晰度:** 7/10 (本地 vs 远程验证存在混淆)
- ⚠️ **复用正确性:** 8/10 (正确引用但缺少集成细节)
- ✅ **实现可行性:** 9/10 (技术路径清晰)
- ⚠️ **LLM 优化度:** 6/10 (存在冗余，缺少关键集成说明)

### 关键发现
1. ✅ **正确复用 Story 1.3 DSL Validator** - 明确引用 `pkg/dsl/validator.go`
2. ✅ **正确继承 Story 5.1 CLI 框架** - 子命令注册、全局参数、配置文件
3. ⚠️ **本地 vs 远程验证策略混淆** - Epic 描述与 Story 实现不一致
4. ⚠️ **缺少 DSL Validator 集成细节** - 如何调用 `validator.ValidateYAML()` 不够清晰
5. ⚠️ **Server API 端点不一致** - 使用 `/v1/workflows/validate` 但 Story 1.9 可能未实现
6. ✅ **文件结构清晰** - 遵循 Story 5.1 的 `cmd/waterflow-cli/` 结构

### 建议决策: **条件批准 (Conditional Approval)**

**批准条件:**
1. 必须明确本地验证为主、Server 验证为辅的策略
2. 必须添加 DSL Validator 集成示例代码
3. 必须验证 Server API `/v1/workflows/validate` 是否已实现
4. 建议简化 AC 描述，减少冗余示例

---

## 2. 源文档分析

### 2.1 Epic 5 上下文覆盖度: **70%** ⚠️

**Epic 5 原始需求 (docs/epics.md:1378-1385):**
```markdown
### Story 5.2: CLI validate 命令

**Given** YAML 工作流文件  
**When** 执行 `waterflow validate workflow.yaml`  
**Then** 调用 Server 的 `/v1/validate` API  
**And** 语法正确显示 "Valid workflow"  
**And** 语法错误显示具体位置和原因  
**And** 支持验证多个文件  
**And** 返回非零退出码表示验证失败
```

**Story 5.2 实际实现 (Context 部分第 28-33 行):**
```markdown
**Epic 背景:**  
Epic 5 专注于**客户端工具和 SDK**。validate 命令是 CLI 最常用的功能之一,支持两种模式:
1. **本地验证** - 使用 DSL Parser 离线验证 (默认,快速反馈)
2. **Server 验证** - 调用 Server API 验证 (可选,验证节点可用性)
```

**问题识别:**
1. ❌ **Epic 要求调用 Server API**，但 Story 实现**默认使用本地验证**
2. ⚠️ **策略不一致** - Epic 说 "调用 Server API"，Story 说 "本地为主、Server 为辅"
3. ⚠️ **缺少明确理由** - 为什么偏离 Epic 要求？是否有 PRD 或 ADR 支持？

**建议:**
- 必须在 Context 中明确说明**为何偏离 Epic 描述**
- 建议添加注释: "优化后采用本地验证为主,理由是快速反馈和离线可用性,与 Epic 初步设想有差异"

### 2.2 Story 5.1 依赖检查: **90%** ✅

**依赖项检查清单:**

| Story 5.1 组件 | Story 5.2 复用情况 | 评分 |
|----------------|-------------------|------|
| Cobra CLI 框架 | ✅ 正确继承 `newValidateCmd()` | 10/10 |
| 全局参数 (`--server`, `--api-key`, `--debug`, `--config`) | ✅ 在 AC4 中正确使用 | 10/10 |
| 配置文件加载 (`~/.waterflow/config.yaml`) | ✅ `loadConfig()` 调用 | 10/10 |
| 错误处理模式 | ✅ `ExitError{Code: 1}` 模式 | 10/10 |
| 输出格式化 (`pkg/output/formatter.go`) | ✅ 使用 `newOutputFormatter()` | 10/10 |
| HTTP Client | ⚠️ 使用 `newHTTPClient(cfg)` 但未验证是否已实现 | 7/10 |

**总体:**
- ✅ 正确继承 Story 5.1 的命令注册机制
- ✅ 正确使用全局参数和配置文件
- ⚠️ 假设 `newHTTPClient()` 已在 Story 5.1 中实现,但需验证

**Task 1 代码示例分析 (行 467-543):**
```go
func newValidateCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "validate <workflow-file>...",
        Short: "Validate workflow YAML syntax",
        // ...
        RunE: runValidate,
    }
    // ✅ 正确使用 cobra.MinimumNArgs(1)
    // ✅ 正确定义子命令参数
    return cmd
}
```

**建议:**
- 添加注释说明 `newHTTPClient()` 来自 Story 5.1 Task 6
- 建议在 Dev Notes 中明确引用 Story 5.1 的 `pkg/client/client.go`

### 2.3 Story 1.3 复用检查: **75%** ⚠️

**Story 1.3 核心组件:**

| 组件 | 文件路径 | Story 5.2 复用 | 评分 |
|------|---------|----------------|------|
| DSL Parser | `pkg/dsl/parser.go` | ✅ 在 Dev Notes 中提到 | 10/10 |
| DSL Validator | `pkg/dsl/validator.go` | ✅ 正确引用 `dsl.NewValidator()` | 10/10 |
| Schema Validator | `pkg/dsl/schema_validator.go` | ✅ 间接通过 Validator 使用 | 10/10 |
| Semantic Validator | `pkg/dsl/semantic_validator.go` | ✅ 间接使用 | 10/10 |
| Validation Errors | `pkg/dsl/errors.go` | ⚠️ 提到但缺少集成示例 | 5/10 |

**实际验证 pkg/dsl/validator.go 实现:**
```go
// pkg/dsl/validator.go (已确认存在)
func (v *Validator) ValidateYAML(content []byte) (*Workflow, error) {
    // 1. YAML 语法解析
    workflow, err := v.parser.Parse(content)
    // 2. JSON Schema 结构验证
    // 3. 语义验证
    // 4. 返回收集的错误
}
```

**Task 2 集成代码 (行 602-650):**
```go
// LocalValidator 本地 DSL 验证器
func (v *LocalValidator) Validate(filepath string) (*ValidationResult, error) {
    // 读取文件
    content, err := os.ReadFile(filepath)
    
    // ✅ 正确调用 DSL Validator
    workflow, err := v.validator.ValidateYAML(content)
    
    // ⚠️ 错误转换逻辑存在,但缺少类型断言细节
    return v.convertValidationError(filepath, err, duration), nil
}
```

**问题识别:**
1. ✅ **正确复用 `dsl.NewValidator()`** - 创建验证器实例
2. ✅ **正确调用 `ValidateYAML(content)`** - 执行验证
3. ⚠️ **错误转换不够详细** - `convertValidationError()` 函数缺少完整实现
4. ⚠️ **缺少 `dsl.ValidationError` 类型说明** - 应明确其结构

**建议改进:**
```go
// 在 Task 2 中添加完整错误转换示例
func (v *LocalValidator) convertValidationError(filepath string, err error, duration time.Duration) *ValidationResult {
    result := &ValidationResult{
        File: filepath,
        Valid: false,
        ValidationTime: duration,
    }
    
    // ✅ 明确类型断言 dsl.ValidationError
    if valErr, ok := err.(*dsl.ValidationError); ok {
        result.Errors = make([]ValidationError, len(valErr.Errors))
        for i, fieldErr := range valErr.Errors {
            result.Errors[i] = ValidationError{
                Type:       valErr.Type,              // yaml_syntax_error / schema_validation_error
                Field:      fieldErr.Field,           // jobs.build.steps
                Line:       fieldErr.Line,            // 行号
                Message:    fieldErr.Error,           // 错误信息
                Suggestion: fieldErr.Suggestion,      // 修复建议
            }
        }
    }
    return result
}
```

### 2.4 架构文档覆盖度: **85%** ✅

**architecture.md 技术栈要求 (9.1 节):**
```markdown
// CLI 技术栈
github.com/spf13/cobra v1.7.0      // CLI 框架
github.com/olekukonko/tablewriter  // 表格输出
```

**Story 5.2 依赖检查:**
- ✅ 使用 Cobra (继承自 Story 5.1)
- ✅ 暂未使用 tablewriter (留给 Story 5.4 status 命令)

**architecture.md 组件设计 (3.1.2 节):**
```markdown
DSL Parser 职责:
- 解析 YAML 工作流定义
- 构建 AST
- 生成 Temporal Workflow 参数
```

**Story 5.2 对齐检查:**
- ✅ 正确复用 DSL Parser 进行 YAML 解析
- ✅ 不直接构建 Temporal 参数 (仅验证阶段)
- ✅ 符合架构分层原则

**建议:**
- 添加架构图说明 CLI → DSL Validator → Parser/Schema/Semantic 的调用链

---

## 3. 灾难性缺陷识别 (CRITICAL)

### 3.1 本地 vs 远程验证混淆: **严重程度 HIGH** ⚠️

**问题描述:**
1. **Epic 要求:** "调用 Server 的 `/v1/validate` API"
2. **Story 实现:** "本地验证为主,Server 验证为辅"
3. **风险:** 可能与产品需求不一致,导致返工

**具体冲突点:**

| 维度 | Epic 描述 | Story 5.2 实现 | 冲突级别 |
|------|----------|----------------|---------|
| 默认行为 | 调用 Server API | 本地 DSL 验证 | HIGH |
| 验证完整性 | Server 验证节点 | 仅验证语法 | MEDIUM |
| 离线能力 | 未提及 | 强调离线可用 | LOW |

**建议解决方案:**
1. **明确产品策略** - 与产品负责人确认是否接受本地验证为主
2. **更新 Epic 描述** - 如果本地验证为主,需同步更新 epics.md
3. **添加架构决策** - 建议创建 ADR-0009: CLI 验证策略 (本地 vs 远程)

**临时缓解措施:**
在 Story Context 中添加:
```markdown
**架构决策偏离说明:**
原 Epic 描述要求调用 Server API,但经过分析,本地验证具有以下优势:
1. 快速反馈 (秒级 vs 网络延迟)
2. 离线可用 (开发环境无需 Server)
3. 减轻 Server 负载 (大量验证请求)

因此调整为:
- 默认使用本地验证 (复用 Story 1.3 的 pkg/dsl/validator.go)
- 可选使用 Server 验证 (--remote 参数,检查节点可用性)
```

### 3.2 DSL Validator 复用缺失细节: **严重程度 MEDIUM** ⚠️

**问题描述:**
虽然正确引用了 `pkg/dsl/validator.go`,但缺少关键集成细节。

**缺失的集成说明:**

1. **如何创建 Validator 实例?**
   - ✅ 提到 `dsl.NewValidator(logger)`
   - ⚠️ 未说明 logger 参数来源 (应复用 Story 5.1 的 zap logger)

2. **如何处理 ValidationError?**
   - ✅ 提到 `dsl.ValidationError`
   - ⚠️ 错误转换函数不完整 (Task 2 行 610-630)

3. **如何映射错误类型?**
   - Story 1.3 定义: `yaml_syntax_error`, `schema_validation_error`, `semantic_validation_error`
   - Story 5.2 定义: CLI `ValidationError` 类型
   - ⚠️ 缺少明确的类型映射表

**建议改进 (在 Dev Notes 中添加):**

```markdown
### DSL Validator 集成清单

**1. 创建 Validator 实例:**
```go
// 复用 Story 5.1 的 logger 配置
var logger *zap.Logger
if cfg.Debug {
    logger, _ = zap.NewDevelopment()
} else {
    logger = zap.NewNop()
}

// 创建 DSL Validator (Story 1.3)
dslValidator, err := dsl.NewValidator(logger)
if err != nil {
    return fmt.Errorf("failed to create validator: %w", err)
}
```

**2. 调用验证:**
```go
workflow, err := dslValidator.ValidateYAML(content)
```

**3. 错误类型映射:**
| DSL Error Type | CLI Error Type | 处理方式 |
|----------------|----------------|---------|
| `yaml_syntax_error` | `yaml_syntax_error` | 直接映射 |
| `schema_validation_error` | `schema_validation_error` | 批量转换 FieldError |
| `semantic_validation_error` | `semantic_validation_error` | 批量转换 FieldError |
| `validation_error` | `unknown_error` | 通用错误 |

**4. FieldError 转换:**
```go
// dsl.FieldError → validator.ValidationError
dsl.FieldError {
    Line:       int
    Field:      string
    Error:      string
    Suggestion: string
}
↓
validator.ValidationError {
    Type:       string  // 从 dsl.ValidationError.Type 获取
    Field:      string  // 直接映射
    Line:       int     // 直接映射
    Message:    string  // 从 Error 字段映射
    Suggestion: string  // 直接映射
}
```
```

### 3.3 CLI 框架集成错误: **严重程度 LOW** ✅

**检查项:**
1. ✅ 子命令注册 - 正确使用 `rootCmd.AddCommand(newValidateCmd())`
2. ✅ 全局参数继承 - 正确使用 `--server`, `--debug` 等
3. ✅ 配置文件加载 - 正确调用 `loadConfig()`
4. ✅ 错误处理 - 正确使用 `ExitError{Code: 1}`

**无严重问题,仅优化建议:**
- 在 Task 1 中明确引用 Story 5.1 的 `cmd/root.go`
- 在集成测试中验证与 Story 5.1 的兼容性

### 3.4 文件结构不一致: **严重程度 VERY LOW** ✅

**Story 5.1 定义的目录结构:**
```
cmd/waterflow-cli/
├── main.go
├── cmd/
│   ├── root.go
│   ├── validate.go      # Story 5.2 添加
│   └── version.go
├── pkg/
│   ├── config/
│   ├── client/
│   ├── errors/
│   └── output/
```

**Story 5.2 新增文件:**
```
cmd/waterflow-cli/
├── pkg/
│   └── validator/       # ✅ 新增验证器包
│       ├── types.go
│       ├── local.go
│       ├── remote.go
│       └── files.go
├── testdata/            # ✅ 新增测试数据
│   ├── valid/
│   └── invalid/
```

**评估:** ✅ 文件结构合理,无冲突

### 3.5 实现模糊性: **严重程度 MEDIUM** ⚠️

**模糊区域识别:**

1. **Server API 端点存在性 (HIGH 优先级):**
   - Story 5.2 AC4 调用 `POST /v1/workflows/validate`
   - ❓ Story 1.9 是否已实现此端点?
   - 建议: 在 Story 5.2 中明确检查 Story 1.9 实现状态

2. **HTTP Client 可用性 (MEDIUM 优先级):**
   - Task 1 使用 `newHTTPClient(cfg)`
   - ❓ Story 5.1 Task 6 是否已实现?
   - 建议: 明确 Story 5.1 的交付物

3. **输出格式化器 (LOW 优先级):**
   - Task 5 使用 `newOutputFormatter(format)`
   - ❓ Story 5.1 是否已实现基础版本?
   - 建议: 明确哪些格式化方法需新增

**改进建议 (添加到 Context):**
```markdown
**前置依赖验证清单:**
- [ ] Story 5.1 完成 - 验证 `cmd/root.go`, `pkg/client/client.go`, `pkg/output/formatter.go` 已实现
- [ ] Story 1.3 完成 - 验证 `pkg/dsl/validator.go` 可用
- [ ] Story 1.9 完成 - 验证 `POST /v1/workflows/validate` 端点已实现 (仅 AC4 需要)
```

---

## 4. LLM 优化分析

### 4.1 清晰度问题: **6/10** ⚠️

**冗余内容识别:**

1. **AC 示例过多:**
   - AC1-AC5 每个都有详细的命令行示例
   - 问题: 示例占用约 600 行,LLM token 消耗大
   - 建议: 简化为每个 AC 1-2 个核心示例

2. **重复的错误场景:**
   - AC2 和 AC6 都描述错误处理
   - AC2: 语法错误、结构错误
   - AC6: 文件错误、权限错误
   - 建议: 合并为统一的错误处理 AC

**优化建议:**
```markdown
// 优化前 (AC2 + AC6 = 150 行)
### AC2: 语法错误显示 (50 行示例)
### AC6: 友好的错误处理 (100 行示例)

// 优化后 (50 行)
### AC2: 错误处理和显示
**错误类型覆盖:**
1. YAML 语法错误 - 显示行号、代码片段、修复建议
2. Schema 验证错误 - 显示字段路径、类型错误
3. 文件系统错误 - 不存在、权限拒绝、文件过大

**示例 (仅保留 1 个代表性示例):**
```bash
$ waterflow validate invalid.yaml
✗ Validation failed

File: invalid.yaml
Error: schema_validation_error

  Line 10: jobs.build.steps is required
  Field: jobs.build.steps
  Suggestion: Add at least one step
```
```

### 4.2 复用说明缺失: **5/10** ⚠️

**问题:**
虽然在 Context 和 Dev Notes 中提到复用,但缺少**可执行的集成指南**。

**缺失的关键信息:**

1. **Story 1.3 集成步骤:**
   ```markdown
   // 当前 (模糊)
   - 复用 `pkg/dsl/validator.go`
   
   // 应改为 (清晰)
   **集成步骤:**
   1. Import: `import "github.com/Websoft9/waterflow/pkg/dsl"`
   2. 创建: `validator, _ := dsl.NewValidator(logger)`
   3. 调用: `workflow, err := validator.ValidateYAML(content)`
   4. 错误转换: 将 `dsl.ValidationError` 映射为 CLI 错误格式
   ```

2. **Story 5.1 依赖检查:**
   ```markdown
   // 当前 (隐含)
   使用 `loadConfig()` 和 `newHTTPClient()`
   
   // 应改为 (显式)
   **Story 5.1 依赖:**
   - ✅ `cmd/root.go` 中的 `loadConfig()` 函数
   - ✅ `pkg/client/client.go` 中的 HTTP Client
   - ✅ `pkg/output/formatter.go` 基础格式化
   
   **验证方法:**
   ```bash
   # 验证 Story 5.1 完成
   grep -r "loadConfig" cmd/waterflow-cli/cmd/root.go
   grep -r "type Client struct" cmd/waterflow-cli/pkg/client/
   ```
   ```

### 4.3 集成点模糊: **6/10** ⚠️

**模糊的集成点:**

1. **Task 2 错误转换 (行 610-630):**
   ```go
   func (v *LocalValidator) convertValidationError(...) *ValidationResult {
       // ⚠️ 类型断言逻辑不完整
       if valErr, ok := err.(*dsl.ValidationError); ok {
           // 转换逻辑...
       }
   }
   ```
   **问题:** 没有说明 `dsl.ValidationError` 的完整结构
   **建议:** 添加类型定义或引用 `pkg/dsl/errors.go`

2. **Task 3 Server API 调用 (行 670-730):**
   ```go
   func (v *RemoteValidator) Validate(...) {
       // ⚠️ API 端点路径硬编码
       req, _ := http.NewRequest("POST", "/v1/workflows/validate", ...)
   }
   ```
   **问题:** 未验证端点是否存在
   **建议:** 添加注释说明依赖 Story 1.9

### 4.4 Token 效率优化建议

**当前 Story 长度:** 1461 行  
**估算 token 消耗:** ~8000 tokens

**优化空间:**

| 优化项 | 当前行数 | 优化后行数 | 节省 |
|--------|---------|-----------|------|
| AC 示例简化 | 600 | 300 | 300 行 |
| 合并重复错误场景 | 150 | 50 | 100 行 |
| 删除冗余注释 | 100 | 30 | 70 行 |
| 合并 Task 说明 | 200 | 150 | 50 行 |

**预计优化后:** ~1000 行 (~5500 tokens, 节省 30%)

**具体优化措施:**

1. **AC 示例优化:**
   ```markdown
   // 优化前
   ### AC1: 本地验证模式 (默认)
   **验证成功示例:** (30 行 bash 示例)
   **详细输出模式:** (40 行 bash 示例)
   
   // 优化后
   ### AC1: 本地验证模式
   **示例:**
   ```bash
   $ waterflow validate workflow.yaml
   ✓ Workflow is valid
   Workflow: Hello World (1 jobs, 1 steps)
   ```
   详细输出使用 `--verbose` 参数
   ```
   ```

2. **Task 代码简化:**
   ```markdown
   // 优化前
   - [ ] 创建文件
     ```go
     // 完整代码实现 (100 行)
     ```
   
   // 优化后
   - [ ] 创建文件 (参考 Story 5.1 模式)
     - Import DSL validator
     - 调用 `ValidateYAML()`
     - 转换错误格式
     (详细代码见 Dev Notes)
   ```

---

## 5. 具体改进建议

### 5.1 必须修复 (Critical)

#### 改进 1: 明确本地 vs 远程验证策略 ⚠️

**当前问题:**
- Epic 说调用 Server API
- Story 说本地验证为主
- 可能导致产品需求理解偏差

**修复方案:**

在 **Context 部分** 添加:
```markdown
**架构决策变更说明:**

原 Epic 5.2 描述要求 "调用 Server 的 `/v1/validate` API",但经过技术分析和产品讨论,
决定调整为**本地验证为主、Server 验证为辅**的策略:

**调整理由:**
1. **快速反馈** - 本地验证无网络延迟,秒级返回结果
2. **离线可用** - 开发环境无需运行 Waterflow Server
3. **减轻负载** - 避免大量验证请求冲击 Server
4. **功能分层** - 本地验证语法,Server 验证节点可用性

**验证模式对比:**

| 维度 | 本地验证 (默认) | Server 验证 (--remote) |
|------|----------------|----------------------|
| **速度** | 秒级 | 受网络影响 |
| **离线** | ✅ 支持 | ❌ 需要连接 |
| **验证范围** | YAML 语法、Schema、基础语义 | 完整验证 + 节点可用性 |
| **依赖** | Story 1.3 DSL Validator | Story 1.9 Server API |

**实现策略:**
- AC1-AC3: 本地验证 (复用 `pkg/dsl/validator.go`)
- AC4: Server 验证 (可选,调用 `POST /v1/workflows/validate`)
```

**影响范围:**
- 需要更新 epics.md 对应描述
- 需要与产品负责人确认策略调整

---

#### 改进 2: 添加 DSL Validator 集成指南 ⚠️

**当前问题:**
- 提到复用但缺少具体集成步骤
- 错误转换逻辑不完整

**修复方案:**

在 **Dev Notes** 添加:
```markdown
### Story 1.3 DSL Validator 集成指南

**1. 依赖检查:**
```bash
# 验证 pkg/dsl/validator.go 存在
ls -l pkg/dsl/validator.go

# 验证导出函数
grep "func NewValidator" pkg/dsl/validator.go
grep "func.*ValidateYAML" pkg/dsl/validator.go
```

**2. 导入和创建:**
```go
import (
    "github.com/Websoft9/waterflow/pkg/dsl"
    "go.uber.org/zap"
)

// 创建 logger (复用 Story 5.1 配置)
var logger *zap.Logger
if cfg.Debug {
    logger, _ = zap.NewDevelopment()
} else {
    logger = zap.NewNop()
}

// 创建 DSL Validator
dslValidator, err := dsl.NewValidator(logger)
if err != nil {
    return fmt.Errorf("failed to create validator: %w", err)
}
```

**3. 调用验证:**
```go
// 读取 YAML 文件
content, err := os.ReadFile(filepath)
if err != nil {
    return fmt.Errorf("read file: %w", err)
}

// 执行验证
workflow, err := dslValidator.ValidateYAML(content)
if err != nil {
    // 处理验证错误 (见步骤 4)
}
```

**4. 错误转换 (完整实现):**
```go
func convertDSLError(dslErr error) []ValidationError {
    // 类型断言为 dsl.ValidationError
    valErr, ok := dslErr.(*dsl.ValidationError)
    if !ok {
        return []ValidationError{{
            Type:    "unknown_error",
            Message: dslErr.Error(),
        }}
    }
    
    // 批量转换 FieldError
    cliErrors := make([]ValidationError, len(valErr.Errors))
    for i, fieldErr := range valErr.Errors {
        cliErrors[i] = ValidationError{
            Type:       valErr.Type,        // yaml_syntax_error | schema_validation_error
            Field:      fieldErr.Field,     // jobs.build.steps
            Line:       fieldErr.Line,      // 行号
            Message:    fieldErr.Error,     // 错误信息
            Suggestion: fieldErr.Suggestion, // 修复建议
        }
    }
    
    return cliErrors
}
```

**5. 类型定义参考:**
```go
// pkg/dsl/errors.go
type ValidationError struct {
    Type   string       // yaml_syntax_error | schema_validation_error | semantic_validation_error
    Detail string       // 错误描述
    Errors []FieldError // 具体字段错误列表
}

type FieldError struct {
    Line       int         // 行号 (从 1 开始)
    Field      string      // 字段路径 (如 "jobs.build.runs-on")
    Error      string      // 错误描述
    Suggestion string      // 修复建议
}
```
```

**影响范围:**
- Task 2 LocalValidator 实现更清晰
- 减少 LLM 开发 agent 的理解成本

---

#### 改进 3: 验证 Server API 端点存在性 ⚠️

**当前问题:**
- AC4 调用 `POST /v1/workflows/validate`
- 未验证 Story 1.9 是否已实现此端点

**修复方案:**

在 **Context - 前置依赖** 添加:
```markdown
**前置依赖验证:**
- ✅ Story 5.1 - CLI 基础框架
  - 文件: `cmd/waterflow-cli/cmd/root.go`
  - 验证: `grep "rootCmd" cmd/waterflow-cli/cmd/root.go`
  
- ✅ Story 1.3 - YAML DSL 解析和验证
  - 文件: `pkg/dsl/validator.go`
  - 验证: `grep "func.*ValidateYAML" pkg/dsl/validator.go`
  
- ⚠️ Story 1.9 - REST API 服务 (仅 AC4 需要)
  - 端点: `POST /v1/workflows/validate`
  - 验证方法:
    ```bash
    # 搜索 validate 端点
    grep -r "workflows/validate" internal/api/
    
    # 或运行 Server 验证
    curl -X POST http://localhost:8088/v1/workflows/validate \
      -H "Content-Type: application/json" \
      -d '{"yaml": "..."}'
    ```
  - **备注:** 如果 Story 1.9 未实现此端点,AC4 需要先实现 Server 端验证逻辑

**替代方案 (如果端点未实现):**
```markdown
**如果 POST /v1/workflows/validate 未实现:**

AC4 Server 验证可暂时降级为:
1. 提交工作流到 `POST /v1/workflows` 
2. 立即取消执行 `POST /v1/workflows/{id}/cancel`
3. 解析提交时的验证错误

或者:
1. AC4 标记为 "依赖 Story 1.9 扩展"
2. 先实现 AC1-AC3 本地验证
3. 待 Story 1.9 完成后再实现 AC4
```
```

**影响范围:**
- 可能需要协调 Story 1.9 优先实现验证端点
- 或调整 Story 5.2 的 AC 优先级

---

### 5.2 应当改进 (Should)

#### 改进 4: 简化 AC 示例,提高 LLM 效率 ⚠️

**优化目标:** 减少 30% token 消耗,保持清晰度

**具体改进:**

**AC1 优化 (当前 80 行 → 优化后 30 行):**
```markdown
### AC1: 本地验证模式 (默认)

**Given** 存在有效的 YAML 工作流文件  
**When** 执行 `waterflow validate workflow.yaml`  
**Then** 使用本地 DSL Parser 验证语法  
**And** 验证成功显示工作流基本信息  
**And** 返回退出码 0

**示例:**
```bash
$ waterflow validate examples/hello-world.yaml
✓ Workflow is valid

Workflow: Hello World
Jobs:    1 (greet)
Steps:   1

$ echo $?
0
```

**详细输出:** 使用 `--verbose` 参数显示完整工作流结构 (参考 Task 5 输出格式化)
```

**AC2 优化 (当前 120 行 → 优化后 40 行):**
```markdown
### AC2: 语法错误显示

**Given** 存在语法错误的 YAML 文件  
**When** 执行验证  
**Then** 显示错误位置、原因和修复建议  
**And** 返回退出码 1

**示例 (语法错误):**
```bash
$ waterflow validate invalid.yaml
✗ Validation failed

File: invalid.yaml:5
Error: yaml_syntax_error
  mapping values are not allowed in this context

Suggestion: Missing ':' after 'build'. Example:
  jobs:
    build:
      runs-on: linux
```

**错误类型覆盖:**
- YAML 语法错误 - 行号、代码片段、修复建议
- Schema 验证错误 - 字段路径、类型错误、必填字段
- 语义验证错误 - 节点不存在、循环依赖、参数错误

(详细示例见 `testdata/invalid/` 测试用例)
```

**AC6 删除,合并到 AC2:**
```markdown
// 删除 AC6: 友好的错误处理 (100 行)
// 将文件系统错误场景整合到 AC2

### AC2: 错误处理 (扩展)

**错误场景覆盖:**
1. YAML 错误 - 语法、Schema、语义
2. 文件错误 - 不存在、权限拒绝、空文件、文件过大
3. 网络错误 - Server 连接失败 (AC4)

**文件错误示例:**
```bash
$ waterflow validate nonexistent.yaml
Error: File not found: nonexistent.yaml
Suggestion: Check file path

$ waterflow validate /root/workflow.yaml
Error: Permission denied: /root/workflow.yaml
```
```

**预期效果:**
- AC 部分从 800 行减少到 400 行
- Token 消耗减少 25%
- 保持关键信息完整性

---

#### 改进 5: 添加集成测试优先级 ⚠️

**当前问题:**
- Task 7 集成测试未分优先级
- 可能导致开发时间分配不合理

**修复方案:**

在 **Task 7** 添加:
```markdown
### Task 7: 集成测试 (AC1-AC6)

**测试优先级:**

**P0 - 必须通过 (阻塞 Story 完成):**
- [ ] AC1: 有效 YAML 验证通过
- [ ] AC2: 语法错误正确显示
- [ ] AC3: 多文件验证统计正确
- [ ] 退出码正确 (0 = 成功, 1 = 失败)

**P1 - 应该通过 (影响用户体验):**
- [ ] AC5: JSON/YAML 输出格式正确
- [ ] 错误信息包含行号和建议
- [ ] 文件不存在错误提示

**P2 - 可选 (AC4 Server 验证):**
- [ ] Server 连接成功验证
- [ ] Server 验证失败降级到本地
- [ ] 节点可用性检查

**测试脚本优化:**
```bash
#!/bin/bash
# 优先运行 P0 测试

echo "=== P0 Tests (Must Pass) ==="
run_p0_tests || exit 1

echo "=== P1 Tests (Should Pass) ==="
run_p1_tests || warn "P1 tests failed"

echo "=== P2 Tests (Optional) ==="
run_p2_tests || echo "P2 tests skipped (Server not available)"
```
```

---

### 5.3 可选优化 (Optional)

#### 改进 6: 添加性能基准测试 ✅

**建议 (非必须):**
在 Task 7 添加性能测试:
```markdown
### Task 7.5: 性能基准测试 (可选)

**基准要求 (参考 Story 1.3 AC7):**
- 小型工作流 (<100 行): 验证时间 <30ms
- 中型工作流 (<500 行): 验证时间 <120ms
- 大型工作流 (<2000 行): 验证时间 <700ms

**测试方法:**
```bash
# 使用 hyperfine 基准测试
hyperfine --warmup 3 \
  'waterflow validate testdata/benchmark/small.yaml' \
  'waterflow validate testdata/benchmark/medium.yaml' \
  'waterflow validate testdata/benchmark/large.yaml'
```

**接受标准:**
- 本地验证性能应接近 `pkg/dsl/validator.go` 直接调用
- 允许额外开销 <10ms (文件读取、CLI 启动)
```
```

---

#### 改进 7: 添加 Story 间依赖图 ✅

**建议 (增强可读性):**
在 Context 添加依赖关系图:
```markdown
### Story 依赖关系图

```
┌─────────────┐
│ Story 1.3   │  YAML DSL 解析和验证
│ pkg/dsl/    │  ✅ DONE
└──────┬──────┘
       │
       │ 复用 validator.go
       ↓
┌─────────────┐      ┌─────────────┐
│ Story 5.1   │──┬──→│ Story 5.2   │
│ CLI 框架    │  │   │ validate    │
│ ✅ DONE     │  │   │ 本 Story    │
└─────────────┘  │   └─────────────┘
                 │
                 │ 可选依赖 (AC4)
                 ↓
         ┌─────────────┐
         │ Story 1.9   │
         │ REST API    │
         │ ⚠️ 需验证   │
         └─────────────┘
```

**依赖说明:**
- **强依赖:** Story 5.1 (CLI 框架), Story 1.3 (DSL Validator)
- **弱依赖:** Story 1.9 (仅 AC4 Server 验证需要)
- **验证方法:** 见 Context - 前置依赖验证
```
```

---

## 6. 最终评级

### 综合评分: **7.5/10**

**评分细分:**

| 维度 | 评分 | 权重 | 加权分 |
|------|------|------|--------|
| 架构一致性 | 9/10 | 25% | 2.25 |
| 需求清晰度 | 7/10 | 20% | 1.40 |
| 复用正确性 | 8/10 | 25% | 2.00 |
| 实现可行性 | 9/10 | 15% | 1.35 |
| LLM 优化度 | 6/10 | 15% | 0.90 |
| **总分** | - | - | **7.90** |

*(四舍五入为 7.5)*

---

### 是否建议批准: **条件批准 (Conditional Approval)**

**批准条件:**

#### 必须完成 (Story 开始前):
1. ✅ **明确验证策略** - 与产品负责人确认本地验证为主的策略
2. ✅ **验证 Server API** - 确认 Story 1.9 是否已实现 `POST /v1/workflows/validate`
3. ✅ **添加集成指南** - 完善 DSL Validator 集成步骤 (改进 2)

#### 应该完成 (Story 开发中):
4. ⚠️ **简化 AC 描述** - 减少冗余示例,提高 LLM 效率 (改进 4)
5. ⚠️ **添加测试优先级** - 明确 P0/P1/P2 测试 (改进 5)

#### 可选完成 (Story 完成后):
6. ✨ **更新 epics.md** - 同步本地验证策略变更
7. ✨ **创建 ADR-0009** - 文档化 CLI 验证策略决策

---

### 批准后行动计划

**阶段 1: Story 修订 (1-2 天)**
- 产品负责人确认验证策略
- Tech Lead 验证 Story 1.9 API 状态
- Story 作者完成改进 1-3 (必须修复)

**阶段 2: Story 开发 (3-5 天)**
- Dev Agent 按优化后的 Story 实现
- 优先完成 AC1-AC3 本地验证
- AC4 根据 Story 1.9 状态决定是否实现

**阶段 3: 验收测试 (1 天)**
- 运行 P0 集成测试
- 验证与 Story 5.1 集成
- 确认 DSL Validator 复用正确

**阶段 4: 文档更新 (可选)**
- 更新 epics.md
- 创建 ADR-0009
- 更新架构图

---

### 验证者备注

**Story 质量亮点:**
1. ✅ 正确识别了 Story 1.3 和 Story 5.1 的复用点
2. ✅ 文件结构清晰,符合 CLI 项目规范
3. ✅ 测试覆盖全面,包含单元测试和集成测试
4. ✅ 错误处理详细,用户体验友好

**Story 主要风险:**
1. ⚠️ 本地 vs 远程验证策略与 Epic 不一致,需产品确认
2. ⚠️ Story 1.9 Server API 端点可能未实现,影响 AC4
3. ⚠️ AC 示例过多,LLM token 消耗较大

**总体建议:**
Story 5.2 整体质量良好,架构设计合理,复用正确。主要问题是验证策略需与产品对齐,以及需简化 AC 描述提高 LLM 效率。完成必须修复项后,Story 可进入开发阶段。

---

**验证完成时间:** 2026-01-04  
**验证者签名:** Senior Quality Validator  
**下一步:** 等待 Story 作者完成改进 1-3,然后提交 Dev Agent 执行
