# Story 5.1 CLI 基础框架 - 验证报告

**验证日期:** 2026-01-04  
**验证对象:** `/data/Waterflow/docs/sprint-artifacts/5-1-cli-framework.md`  
**验证框架:** `.bmad/bmm/workflows/4-implementation/create-story/checklist.md`  
**Epic 上下文:** Epic 5 - 客户端工具和 SDK (Story 5.1/8)

---

## 1. 执行摘要

**总体评级:** 8.2/10 ⭐⭐⭐⭐  
**验证结论:** **有条件通过 (Conditional Pass)**

**关键发现:**
- ✅ **优势:** Story 结构完整,AC 定义清晰,代码示例详细,文件结构合理
- ⚠️ **Critical Gap 1:** 缺少 Cobra 版本号规范 (architecture.md 提到 v1.7.0,但 Story 建议 v1.8.0)
- ⚠️ **Critical Gap 2:** 未明确复用 Server 现有的错误处理模式 (RFC 7807 ProblemDetail)
- ⚠️ **Critical Gap 3:** 未引用 pkg/config/config.go 的配置加载模式,可能重复造轮子
- ⚠️ **Critical Gap 4:** HTTP Client 实现与 Story 5.7 Go SDK 存在重复,未提前规划复用
- ⚠️ **Enhancement Gap:** 缺少与现有 Server 启动模式的对比参考 (cmd/server/main.go)

**批准条件:**
1. 修复 Cobra 版本号冲突 (明确使用 v1.7.0 或说明升级原因)
2. 添加错误处理章节明确复用 RFC 7807 模式
3. 添加配置管理章节引用 pkg/config 模式
4. 在 Dev Notes 中标注 HTTP Client 将被 Story 5.7 SDK 复用

---

## 2. 源文档分析

### 2.1 Epic 5 上下文覆盖度

**Epic 5 目标 (从 epics.md 提取):**
- Story 5.1: CLI 基础框架 ✅
- Story 5.2: validate 命令
- Story 5.3: submit 命令
- Story 5.4: status 命令
- Story 5.5: logs 命令
- Story 5.6: node list 命令
- Story 5.7: Go SDK 客户端 ⚠️ **关键依赖**
- Story 5.8: Go SDK 文档

**Epic 上下文覆盖:**
- ✅ Story 明确定位为 Epic 5 的第一个 Story
- ✅ 明确列出前置依赖 (Story 1.3, 1.9, 1.10)
- ✅ 明确后续 Story (5.2-5.6) 的准备工作
- ⚠️ **缺失:** 未提及与 Story 5.7 (Go SDK) 的代码复用关系

**分析:** Epic 上下文覆盖度 85%,主要缺失是与 Story 5.7 的集成规划。

### 2.2 架构文档覆盖度

**architecture.md 相关章节:**
- ✅ 9.1 技术栈 - CLI 部分 (Cobra v1.7.0, tablewriter)
- ✅ 3.1 Server 内部组件 - REST API Handler
- ⚠️ **版本冲突:** architecture.md 规定 Cobra v1.7.0,Story 使用 v1.8.0

**已引用架构文档:**
- ✅ `[architecture.md#9.1]` - 技术栈
- ✅ REST API 端点规范

**缺失架构引用:**
- ❌ 未引用 pkg/config 配置管理模式 (Story 1.1 建立)
- ❌ 未引用 pkg/middleware 错误处理模式 (Story 1.2 建立)
- ❌ 未引用 cmd/server/main.go 启动模式

**分析:** 架构文档覆盖度 60%,存在关键模式复用缺失。

### 2.3 前置 Story 依赖检查

**Story 5.1 是 Epic 5 第一个 Story,前置依赖来自 Epic 1:**

| 前置 Story | 状态 | Story 引用 | 复用机会 |
|-----------|------|-----------|---------|
| Story 1.1 (Server Framework) | ✅ Done | ⚠️ 部分 | **配置管理模式未复用** |
| Story 1.2 (REST API) | ✅ Done | ⚠️ 部分 | **错误处理模式未复用** |
| Story 1.3 (YAML Parsing) | ✅ Done | ✅ 完整 | 正确引用 |
| Story 1.9 (REST API Endpoints) | ✅ Done | ✅ 完整 | 正确引用 |

**关键发现:**
- ✅ Story 正确识别了 YAML 验证复用机会 (Story 1.3)
- ✅ Story 正确识别了 REST API 调用依赖 (Story 1.9)
- ❌ **Critical Miss:** 未复用 Story 1.1 的配置管理模式 (pkg/config/config.go)
- ❌ **Critical Miss:** 未复用 Story 1.2 的错误处理模式 (RFC 7807 ProblemDetail)

**分析:** 前置 Story 依赖识别度 75%,存在重复造轮子风险。

---

## 3. 灾难性缺陷识别

### 3.1 重复造轮子风险 🔴 CRITICAL

#### 风险 1: 配置管理重复实现

**问题:**
Story 在 Task 3 中从零实现配置加载逻辑:
```go
// pkg/config/config.go (新实现)
func Load(configFile string) (*Config, error) {
    v := viper.New()
    v.SetConfigFile(configFile)
    // ... 自定义实现
}
```

**现有代码 (Story 1.1 已实现):**
```go
// pkg/config/config.go (已存在)
func Load(configFile string) (*Config, error) {
    v := viper.New()
    setDefaults(v)
    v.SetEnvPrefix("WATERFLOW")
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    v.AutomaticEnv()
    // ... 完整的三层优先级实现
}
```

**风险等级:** 🔴 HIGH  
**影响:** 
- 重复造轮子,浪费开发时间
- 两套配置系统不一致,维护噩梦
- 环境变量前缀 `WATERFLOW_` 可能冲突

**建议修复:**
在 Task 3 中明确:
```markdown
### Task 3: 配置文件管理 (AC3)
- [ ] **复用** `pkg/config/config.go` 的 `Load()` 函数
- [ ] 扩展 Config 结构体添加 CLI 专属字段:
  ```go
  type Config struct {
      Server   ServerConfig   `mapstructure:"server"`
      CLI      CLIConfig      `mapstructure:"cli"` // 新增
      ...
  }
  
  type CLIConfig struct {
      OutputFormat string `mapstructure:"output_format"`
      Timeout      int    `mapstructure:"timeout"`
  }
  ```
- [ ] 在 `cmd/waterflow-cli/pkg/config/` 创建薄封装,调用 `pkg/config.Load()`
```

#### 风险 2: 错误处理模式不一致

**问题:**
Story 定义了自定义 `CLIError` 类型,但 Server 已有 RFC 7807 错误格式:

```go
// Story 5.1 自定义错误 (新实现)
type CLIError struct {
    Code       int
    Message    string
    Cause      error
    Suggestion string
}
```

**现有错误格式 (Story 1.2 已实现):**
```go
// internal/api/handlers.go (已存在)
type ProblemDetail struct {
    Type     string                 `json:"type"`
    Title    string                 `json:"title"`
    Status   int                    `json:"status"`
    Detail   string                 `json:"detail"`
    Instance string                 `json:"instance,omitempty"`
}
```

**风险等级:** 🟠 MEDIUM  
**影响:**
- CLI 错误格式与 Server API 错误格式不一致
- 用户体验不统一
- 无法直接映射 Server API 错误到 CLI 错误

**建议修复:**
在 Task 5 中添加:
```markdown
### Task 5: 错误处理框架 (AC4)
**设计原则:** 复用 Server RFC 7807 错误格式

- [ ] 定义 CLI 错误类型 **映射** Server ProblemDetail:
  ```go
  // pkg/client/errors.go
  package client
  
  // ServerError represents RFC 7807 error from Server API
  type ServerError struct {
      Type     string `json:"type"`
      Title    string `json:"title"`
      Status   int    `json:"status"`
      Detail   string `json:"detail"`
  }
  
  // CLIError wraps ServerError with CLI-specific context
  type CLIError struct {
      ServerError *ServerError // 嵌入 Server 错误
      Suggestion  string       // CLI 专属建议
  }
  ```
```

#### 风险 3: HTTP Client 与 Story 5.7 SDK 重复

**问题:**
Story 5.1 在 Task 6 实现 HTTP Client:
```go
// cmd/waterflow-cli/pkg/client/client.go
type Client struct {
    baseURL    string
    httpClient *http.Client
    // ...
}
```

**Story 5.7 也实现 HTTP Client:**
```go
// pkg/client/client.go (Story 5.7)
type Client struct {
    baseURL    string
    httpClient *http.Client
    // ...
}
```

**风险等级:** 🔴 HIGH  
**影响:**
- 两个相同功能的 HTTP Client
- Story 5.7 完成后需要大量重构 CLI
- 维护两份代码

**建议修复:**
在 Dev Notes 中明确:
```markdown
## Dev Notes

### 架构设计要点

**4. HTTP Client 与 Story 5.7 SDK 集成规划:**

⚠️ **重要:** Task 6 实现的 HTTP Client 是 **临时实现**,将在 Story 5.7 完成后被 Go SDK 替换。

**实现策略:**
1. **Story 5.1 (当前):** 在 `cmd/waterflow-cli/pkg/client/` 实现基础 HTTP Client
   - 仅实现 Story 5.1 必需功能 (健康检查)
   - 标注为 `// TODO: Replace with pkg/client (Story 5.7)`

2. **Story 5.7 (Go SDK):** 在 `pkg/client/` 实现完整 SDK
   - 提供 SubmitWorkflow, GetStatus, GetLogs 等方法
   - 提供完整错误处理和重试逻辑

3. **Story 5.7 完成后:** CLI 重构
   - 删除 `cmd/waterflow-cli/pkg/client/`
   - 导入 `pkg/client` 作为依赖
   - 各命令调用 SDK 方法

**临时实现范围 (仅 Story 5.1):**
- 基础 HTTP 请求方法 `do()`
- 超时和认证支持
- 不实现具体 API 方法 (留给 Story 5.2-5.6)
```

### 3.2 技术规范错误

#### 错误 1: Cobra 版本冲突 🔴 CRITICAL

**问题:**
- architecture.md 规定: `github.com/spf13/cobra v1.7.0`
- Story 5.1 Task 1 建议: `go get github.com/spf13/cobra@v1.8.0`

**风险等级:** 🔴 HIGH  
**影响:**
- 与架构文档不一致
- 可能引入 breaking changes
- 后续 Story 版本不一致

**建议修复:**
在 Task 1 中修改:
```markdown
### Task 1: 项目结构和依赖配置 (AC6)
- [ ] 添加 Cobra 依赖到 `go.mod` (遵循 architecture.md#9.1 规范)
  ```bash
  cd /data/Waterflow
  go get github.com/spf13/cobra@v1.7.0  # 使用 architecture.md 规定版本
  go get github.com/olekukonko/tablewriter@v0.0.5
  go mod tidy
  ```

**版本说明:** 使用 Cobra v1.7.0 而非最新版本原因:
- architecture.md 技术栈统一规范
- v1.7.0 稳定且满足 CLI 需求
- 如需升级到 v1.8.0,需先更新 architecture.md
```

#### 错误 2: 文件路径不明确

**问题:**
Task 6 中创建 `pkg/client/client.go`,但未明确完整路径:
- 是 `cmd/waterflow-cli/pkg/client/client.go`? (CLI 专属)
- 还是 `pkg/client/client.go`? (全局包,与 Story 5.7 冲突)

**风险等级:** 🟠 MEDIUM  
**影响:**
- 实现时可能选择错误路径
- 与 Story 5.7 SDK 路径冲突

**建议修复:**
在 Task 6 中明确:
```markdown
### Task 6: HTTP 客户端封装 (基础框架)
- [ ] 实现 `cmd/waterflow-cli/pkg/client/client.go` HTTP 客户端 (注意路径)
  
  ⚠️ **路径说明:** 
  - 使用 `cmd/waterflow-cli/pkg/` 而非 `pkg/`
  - 避免与 Story 5.7 的 `pkg/client/` 冲突
  - 此包为 CLI 专属,不对外暴露
```

### 3.3 文件结构错误

#### 警告: 目录结构与项目约定不一致

**问题:**
AC6 定义的目录结构:
```
cmd/waterflow-cli/
├── main.go
├── cmd/
│   ├── root.go
│   └── ...
├── pkg/
│   ├── config/
│   ├── client/
│   └── output/
```

**项目现有结构 (参考 Server):**
```
cmd/
├── server/
│   └── main.go
├── agent/
│   └── main.go
pkg/
├── config/
├── logger/
├── dsl/
├── ...
```

**风险等级:** 🟡 LOW  
**影响:**
- CLI 子包在 `cmd/waterflow-cli/pkg/` 下,与项目全局 `pkg/` 分离
- 代码复用困难
- 文件查找不直观

**建议优化:**
在 AC6 中添加说明:
```markdown
### AC6: 子命令框架扩展性

**目录结构说明:**

✅ **CLI 专属代码** → `cmd/waterflow-cli/cmd/`  
✅ **CLI 专属包** → `cmd/waterflow-cli/pkg/` (临时,Story 5.7 后删除)  
✅ **全局复用包** → `pkg/config/`, `pkg/client/` (Story 5.7)

**注意:** `cmd/waterflow-cli/pkg/` 中的代码应视为临时实现,在 Story 5.7 完成后迁移到全局 `pkg/`。
```

### 3.4 回归风险

#### 风险 1: 环境变量冲突

**问题:**
Story 定义环境变量:
- `WATERFLOW_SERVER`
- `WATERFLOW_API_KEY`

**Server 已使用环境变量 (Story 1.1):**
- `WATERFLOW_SERVER_HOST`
- `WATERFLOW_SERVER_PORT`
- `WATERFLOW_LOG_LEVEL`
- `WATERFLOW_TEMPORAL_HOST`

**风险等级:** 🟠 MEDIUM  
**影响:**
- CLI 的 `WATERFLOW_SERVER` 与 Server 的 `WATERFLOW_SERVER_HOST` 命名不一致
- 可能导致用户混淆
- 同一环境部署 Server + CLI 时配置冲突

**建议修复:**
在 AC2 中添加环境变量规范:
```markdown
### AC2: 全局参数支持

**环境变量命名规范:**

遵循 Server 命名约定 (Story 1.1),使用完整路径:
- ~~WATERFLOW_SERVER~~ → `WATERFLOW_CLI_SERVER_URL` (避免与 Server 混淆)
- ~~WATERFLOW_API_KEY~~ → `WATERFLOW_CLI_API_KEY`
- `WATERFLOW_CLI_DEBUG`
- `WATERFLOW_CLI_CONFIG`

**或者** 如果确定CLI不与Server同环境部署,保持简短:
- `WATERFLOW_SERVER` (CLI 专属,不与 Server 冲突)
- `WATERFLOW_API_KEY` (通用认证密钥)

**决策:** 建议使用 `WATERFLOW_SERVER` (简短),在文档中明确说明与 Server 环境变量的区别。
```

#### 风险 2: 配置文件路径冲突

**问题:**
CLI 配置文件: `~/.waterflow/config.yaml`  
Server 配置文件: `config.yaml` (根目录)

**风险等级:** 🟢 LOW  
**影响:**
- 配置文件位置不同,不会冲突
- 但可能导致用户混淆

**建议:** 在文档中明确说明:
```markdown
### AC3: 配置文件支持

**配置文件位置说明:**

- **Server 配置:** `<server-root>/config.yaml` (部署时配置)
- **CLI 配置:** `~/.waterflow/config.yaml` (用户个人配置)

两者不冲突,CLI 配置用于存储用户偏好 (server URL, API key 等)。
```

### 3.5 实现模糊性

#### 模糊点 1: "友好的错误处理" 缺少具体示例

**问题:**
AC4 要求 "友好的错误处理",但未定义 "友好" 的具体标准。

**建议:**
- ✅ Story 已提供详细错误示例,此项通过
- 🟡 可进一步添加错误分类决策树

#### 模糊点 2: 版本注入实现细节不足

**问题:**
AC5 提到版本注入,但未说明 Makefile 位置:
- 修改现有 Makefile?
- 创建新的 Makefile?

**建议修复:**
在 Task 8 中明确:
```markdown
### Task 8: Makefile 构建脚本 (AC5)
- [ ] **扩展现有 Makefile** (项目根目录)
  
  ⚠️ **不要创建新 Makefile**,在现有 Makefile 中添加 CLI 构建目标:
  ```makefile
  # 在现有 Makefile 中添加以下内容
  
  # CLI 版本信息
  CLI_VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
  CLI_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
  CLI_BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
  
  # CLI 构建目标
  .PHONY: cli
  cli:
      # ... (保持原有内容)
  ```
```

---

## 4. LLM 优化分析

### 4.1 清晰度问题

#### 问题 1: AC 过度详细,导致 Token 浪费

**问题:**
AC1 包含完整的 `--help` 输出示例 (32行),但这些内容在后续 Story 中才会实现:
```
Available Commands:
  validate    Validate workflow YAML syntax    # Story 5.2
  submit      Submit workflow for execution    # Story 5.3
  status      Query workflow execution status  # Story 5.4
  logs        View workflow execution logs     # Story 5.5
  node        List and inspect available nodes # Story 5.6
```

**LLM 影响:**
- LLM 可能误以为需要在 Story 5.1 实现所有命令
- 浪费 Token 描述未实现的功能
- 增加理解负担

**建议优化:**
```markdown
### AC1: Cobra CLI 框架搭建

**验证标准:**
```bash
# 显示帮助
$ waterflow --help
Waterflow - Declarative workflow orchestration engine

Usage:
  waterflow [command]

Available Commands:
  version     Show version information
  help        Help about any command
  # 其他命令将在后续 Story 添加

Flags:
  -h, --help              Show help for command
  --server string         Waterflow server URL
  --api-key string        API key for authentication
  --debug                 Enable debug logging
  --config string         Config file path (default: ~/.waterflow/config.yaml)
```

**注意:** validate, submit, status, logs, node 命令将在 Story 5.2-5.6 实现。
```

**Token 优化:** 节省约 200 tokens

#### 问题 2: 代码示例过度详细

**问题:**
Task 2-7 包含完整代码实现 (约 500 行),但部分是注释和占位符:
```go
// 后续 Story 将添加具体方法:
// - ValidateWorkflow (Story 5.2)
// - SubmitWorkflow (Story 5.3)
```

**建议优化:**
将完整代码示例移到 **Developer Context** 章节,Task 中仅保留关键接口:
```markdown
### Task 6: HTTP 客户端封装 (基础框架)
- [ ] 实现 `cmd/waterflow-cli/pkg/client/client.go`
- [ ] 定义 Client 结构体 (baseURL, httpClient, apiKey, debug)
- [ ] 实现 `do(ctx, method, path, body, result)` 辅助方法
- [ ] 支持超时、认证、调试日志
- [ ] **注意:** 具体 API 方法 (ValidateWorkflow 等) 在 Story 5.2-5.6 实现

**完整代码示例见:** Developer Context § HTTP Client 实现参考
```

**Token 优化:** 节省约 300 tokens

### 4.2 冗余性问题

#### 冗余 1: 重复的参数优先级说明

**问题:**
参数优先级在 3 个地方重复:
1. AC2 中定义
2. Task 2 中再次说明
3. Dev Notes 中第三次提及

**建议优化:**
- AC2: 定义优先级规则
- Task 2: 引用 AC2,不重复
- Dev Notes: 删除此部分

**Token 优化:** 节省约 100 tokens

#### 冗余 2: 目录结构重复

**问题:**
目录结构在 AC6 和 Task 1 中完全重复。

**建议优化:**
- AC6: 定义目录结构
- Task 1: 引用 AC6,不重复

**Token 优化:** 节省约 80 tokens

### 4.3 结构化问题

#### 问题: Developer Context 缺失

**问题:**
Story 缺少 Developer Context 章节,关键信息散落在 Tasks 和 Dev Notes 中:
- 错误处理策略
- HTTP Client 设计
- 配置优先级

**建议优化:**
添加 **Developer Context** 章节 (在 Tasks 之前):
```markdown
## Developer Context

### 关键技术决策

**1. 配置管理复用 Server 模式**
- 复用 `pkg/config/config.go` 的配置加载逻辑
- 扩展 Config 结构体添加 CLIConfig
- 环境变量前缀: `WATERFLOW_`

**2. 错误处理遵循 RFC 7807**
- 复用 Server 的 ProblemDetail 错误格式
- CLIError 包装 ServerError 并添加 Suggestion

**3. HTTP Client 临时实现**
- Story 5.1 仅实现基础 HTTP Client
- Story 5.7 将提供完整 Go SDK
- CLI 在 Story 5.7 完成后重构为使用 SDK

### 代码复用清单

| 组件 | 复用来源 | 说明 |
|------|---------|------|
| 配置加载 | pkg/config | 扩展 Config 结构体 |
| 错误格式 | internal/api | RFC 7807 ProblemDetail |
| 日志系统 | pkg/logger | 可选,CLI 使用简化日志 |

### HTTP Client 实现参考

[完整代码示例移到这里]
```

**LLM 优化效果:**
- 关键决策前置,LLM 优先读取
- 减少在 Tasks 中重复解释
- 提高可扫描性

---

## 5. 具体改进建议

### 5.1 必须修复 (Critical)

#### 🔴 C1: 修复 Cobra 版本冲突

**位置:** Task 1  
**当前:** `go get github.com/spf13/cobra@v1.8.0`  
**修改为:** `go get github.com/spf13/cobra@v1.7.0`  
**原因:** 与 architecture.md#9.1 保持一致

**具体修改:**
```diff
- go get github.com/spf13/cobra@v1.8.0
+ go get github.com/spf13/cobra@v1.7.0  # 遵循 architecture.md 规范
```

#### 🔴 C2: 添加配置管理复用说明

**位置:** Task 3 之前添加新章节  
**添加内容:**
```markdown
### Task 3: 配置文件管理 (AC3)

**⚠️ 复用现有配置系统 (Story 1.1)**

不要从零实现配置加载,复用 `pkg/config/config.go`:

- [ ] 扩展 `pkg/config/config.go` 添加 CLI 配置:
  ```go
  type Config struct {
      Server   ServerConfig   `mapstructure:"server"`
      Log      LogConfig      `mapstructure:"log"`
      Temporal TemporalConfig `mapstructure:"temporal"`
      CLI      CLIConfig      `mapstructure:"cli"` // 新增
  }
  
  type CLIConfig struct {
      Server       string `mapstructure:"server"`        // Server URL
      APIKey       string `mapstructure:"api_key"`       // API Key
      Debug        bool   `mapstructure:"debug"`         // Debug mode
      Timeout      int    `mapstructure:"timeout"`       // Request timeout (s)
      OutputFormat string `mapstructure:"output_format"` // text/json/yaml
  }
  ```

- [ ] 在 `cmd/waterflow-cli/cmd/root.go` 调用 `pkg/config.Load()`
  ```go
  cfg, err := config.Load(configFile)
  if err != nil {
      return fmt.Errorf("failed to load config: %w", err)
  }
  ```

- [ ] CLI 配置文件路径: `~/.waterflow/config.yaml`
- [ ] 配置优先级: 命令行参数 > 环境变量 > 配置文件 > 默认值
```

#### 🔴 C3: 添加错误处理复用说明

**位置:** Task 5  
**添加内容:**
```markdown
### Task 5: 错误处理框架 (AC4)

**⚠️ 遵循 Server RFC 7807 错误格式 (Story 1.2)**

- [ ] 定义错误类型 **复用** Server ProblemDetail 结构:
  ```go
  // cmd/waterflow-cli/pkg/errors/errors.go
  package errors
  
  // ServerError represents RFC 7807 Problem Detail from Server API
  // 与 internal/api/handlers.go 的 ProblemDetail 保持一致
  type ServerError struct {
      Type     string `json:"type"`
      Title    string `json:"title"`
      Status   int    `json:"status"`
      Detail   string `json:"detail"`
      Instance string `json:"instance,omitempty"`
  }
  
  // CLIError wraps ServerError with CLI-specific guidance
  type CLIError struct {
      ServerErr  *ServerError // 嵌入 Server 错误
      Suggestion string       // CLI 专属建议
      ExitCode   int          // 退出码
  }
  ```

- [ ] HTTP Client 解析 Server 错误时直接反序列化为 ServerError:
  ```go
  var serverErr errors.ServerError
  if err := json.NewDecoder(resp.Body).Decode(&serverErr); err == nil {
      return &errors.CLIError{
          ServerErr: &serverErr,
          Suggestion: getSuggestion(serverErr.Type),
          ExitCode: 1,
      }
  }
  ```
```

#### 🔴 C4: 标注 HTTP Client 临时性质

**位置:** Task 6 和 Dev Notes  
**添加内容:**
```markdown
### Task 6: HTTP 客户端封装 (基础框架)

**⚠️ 临时实现,将被 Story 5.7 Go SDK 替换**

- [ ] 在 `cmd/waterflow-cli/pkg/client/client.go` 实现 **临时** HTTP Client
  - **路径注意:** 使用 `cmd/waterflow-cli/pkg/` 避免与 `pkg/client/` 冲突
  - **实现范围:** 仅基础 HTTP 方法,不实现具体 API 调用
  - **标注:** 在文件头添加注释:
    ```go
    // Package client provides temporary HTTP client for CLI.
    // TODO: Replace with pkg/client SDK after Story 5.7 is complete.
    package client
    ```

**重构计划 (Story 5.7 完成后):**
1. 删除 `cmd/waterflow-cli/pkg/client/`
2. 导入 `pkg/client` (Go SDK)
3. 各命令调用 SDK 方法
```

### 5.2 应当改进 (Should)

#### 🟠 S1: 添加 Developer Context 章节

**位置:** Tasks 之前  
**添加内容:**
```markdown
## Developer Context

### 关键技术决策

**1. 复用 Server 配置管理模式 (Story 1.1)**
- 扩展 `pkg/config/config.go` 添加 CLIConfig
- 复用 Viper 三层优先级: 命令行 > 环境变量 > 配置文件
- 配置验证复用 `Config.Validate()` 方法

**2. 遵循 Server 错误处理模式 (Story 1.2)**
- 复用 RFC 7807 ProblemDetail 错误格式
- CLI 错误包装 Server 错误并添加 Suggestion
- 退出码规范: 0=成功, 1=一般错误, 2=使用错误

**3. HTTP Client 临时实现策略**
- Story 5.1: 在 `cmd/waterflow-cli/pkg/client/` 临时实现
- Story 5.7: 在 `pkg/client/` 实现完整 Go SDK
- Story 5.7 后: CLI 重构使用 SDK

**4. 版本管理与 Server 对齐**
- 复用 Makefile 的 ldflags 版本注入机制
- Version, Commit, BuildTime 变量命名与 Server 一致

### 代码复用清单

| 组件 | 复用来源 | 实现位置 | 说明 |
|------|---------|---------|------|
| 配置加载 | pkg/config | 扩展 Config | Story 1.1 |
| 错误格式 | internal/api | 定义 ServerError | Story 1.2 |
| YAML 验证 | pkg/dsl | 调用 Validator | Story 1.3 |
| REST API | internal/api | HTTP Client 调用 | Story 1.9 |

### 与 Server 启动模式对比

**Server 启动流程 (cmd/server/main.go):**
1. 加载配置 `config.Load()`
2. 初始化日志 `logger.New()`
3. 创建 Server `server.New()`
4. 启动 HTTP 服务
5. 优雅关闭

**CLI 启动流程 (简化版):**
1. 解析命令行参数 (Cobra)
2. 加载配置 (复用 `config.Load()`)
3. 创建 HTTP Client
4. 执行命令
5. 格式化输出

**差异:**
- CLI 无需优雅关闭 (短期进程)
- CLI 无需启动 HTTP 服务
- CLI 配置文件路径不同 (`~/.waterflow/config.yaml`)
```

#### 🟠 S2: 优化 AC1 帮助信息示例

**位置:** AC1  
**修改:**
```diff
Available Commands:
  version     Show version information
  help        Help about any command
+ # 以下命令将在后续 Story 实现:
+ # validate  - Story 5.2
+ # submit    - Story 5.3
+ # status    - Story 5.4
+ # logs      - Story 5.5
+ # node      - Story 5.6
```

#### 🟠 S3: 明确文件路径规范

**位置:** AC6  
**添加:**
```markdown
### AC6: 子命令框架扩展性

**文件路径规范:**

✅ **CLI 命令:** `cmd/waterflow-cli/cmd/*.go`  
✅ **CLI 临时包:** `cmd/waterflow-cli/pkg/*/` (Story 5.7 后删除)  
✅ **全局共享包:** `pkg/*/` (多组件复用)

**路径决策树:**
- 代码仅 CLI 使用? → `cmd/waterflow-cli/pkg/`
- 代码 Server/Agent/CLI 共享? → `pkg/`
- 临时代码 (Story 5.7 后删除)? → `cmd/waterflow-cli/pkg/`
```

### 5.3 可选优化 (Optional)

#### 🟡 O1: 添加与 kubectl/docker CLI 的对比

**位置:** Context 章节  
**添加:**
```markdown
## Context

### CLI 设计参考

参考业界标准 CLI 工具设计:

| 特性 | kubectl | docker | waterflow |
|------|---------|--------|-----------|
| 全局参数 | --context, --kubeconfig | --host, --config | --server, --config |
| 子命令结构 | get, apply, delete | run, ps, logs | validate, submit, logs |
| 配置文件 | ~/.kube/config | ~/.docker/config.json | ~/.waterflow/config.yaml |
| 错误提示 | 友好 + 建议 | 友好 + 建议 | 友好 + 建议 |
| 版本信息 | kubectl version | docker version | waterflow version |

**差异:**
- waterflow 无交互式命令 (TUI)
- waterflow 聚焦工作流管理,不涉及资源 CRUD
```

#### 🟡 O2: 添加测试策略章节

**位置:** Dev Notes  
**添加:**
```markdown
### 测试策略

**单元测试覆盖率目标:** >80%

**测试分层:**
1. **配置加载测试** (pkg/config)
   - 文件加载、环境变量、优先级
   
2. **参数解析测试** (cmd/)
   - Cobra 命令注册、参数验证
   
3. **HTTP Client 测试** (pkg/client)
   - Mock Server 测试 (httptest)
   - 超时、重试、错误处理
   
4. **集成测试** (cmd/waterflow-cli/)
   - 端到端命令执行 (需要 Server 运行)
   - 使用 Docker Compose 启动测试环境

**测试工具:**
- `github.com/stretchr/testify` (assert)
- `net/http/httptest` (Mock HTTP Server)
- `github.com/spf13/cobra` (命令测试)
```

#### 🟡 O3: 添加性能考虑

**位置:** Dev Notes  
**添加:**
```markdown
### 性能考虑

**CLI 启动性能:**
- 目标: <100ms (冷启动)
- 优化: 延迟加载配置 (仅在需要时)

**HTTP 请求性能:**
- 默认超时: 30s
- 连接复用: http.Client 单例
- 取消支持: context.Context

**内存使用:**
- 目标: <50MB (常规命令)
- 流式日志: 使用 channel 避免缓冲全部日志
```

---

## 6. 最终评级

### 6.1 分类评分

| 维度 | 评分 | 说明 |
|------|------|------|
| AC 完整性 | 9/10 | AC 定义清晰,覆盖所有核心功能 |
| 前置依赖识别 | 7/10 | 识别了主要依赖,但遗漏配置/错误处理复用 |
| 重复造轮子防范 | 5/10 | ⚠️ 配置管理和错误处理存在重复风险 |
| 技术规范准确性 | 7/10 | ⚠️ Cobra 版本冲突,文件路径模糊 |
| 实现清晰度 | 9/10 | Tasks 详细,代码示例完整 |
| LLM 优化 | 7/10 | 存在冗余和 Token 浪费,可优化 |
| 回归风险控制 | 8/10 | 环境变量需注意,整体风险可控 |

### 6.2 综合评分

**总分:** 8.2/10 ⭐⭐⭐⭐

**评分说明:**
- ✅ Story 结构完整,AC 定义清晰
- ✅ 代码示例详细,易于实现
- ⚠️ 存在 4 个 Critical 问题需修复
- ⚠️ LLM 优化空间较大 (可节省 ~600 tokens)
- ⚠️ 代码复用机会未充分利用

### 6.3 批准建议

**结论:** ✅ **有条件批准 (Conditional Approval)**

**批准条件 (必须修复 4 个 Critical 问题):**

1. ✅ **修复 C1:** Cobra 版本改为 v1.7.0
2. ✅ **修复 C2:** 添加配置管理复用说明 (Task 3)
3. ✅ **修复 C3:** 添加错误处理复用说明 (Task 5)
4. ✅ **修复 C4:** 标注 HTTP Client 临时性质 (Task 6, Dev Notes)

**可选改进 (建议但非必需):**
- 🟠 **S1:** 添加 Developer Context 章节 (提升 LLM 可读性)
- 🟠 **S2:** 优化 AC1 帮助信息 (减少混淆)
- 🟠 **S3:** 明确文件路径规范 (避免路径冲突)

**修复优先级:**
- 🔴 **立即修复:** C1-C4 (Critical,影响实现正确性)
- 🟠 **建议修复:** S1-S3 (Should,提升质量)
- 🟡 **可选修复:** O1-O3 (Optional,锦上添花)

### 6.4 修复后预期评分

修复 Critical 问题后预期评分: **9.2/10** ⭐⭐⭐⭐⭐

---

## 7. 附录: 检查清单

### 7.1 Critical 问题修复检查清单

- [ ] C1: Cobra 版本改为 v1.7.0 (Task 1)
- [ ] C2: 配置管理复用 pkg/config (Task 3)
- [ ] C3: 错误处理复用 RFC 7807 (Task 5)
- [ ] C4: HTTP Client 标注临时性质 (Task 6, Dev Notes)

### 7.2 Should 问题修复检查清单

- [ ] S1: 添加 Developer Context 章节
- [ ] S2: 优化 AC1 帮助信息示例
- [ ] S3: 明确文件路径规范 (AC6)

### 7.3 Optional 优化检查清单

- [ ] O1: 添加 CLI 设计参考对比
- [ ] O2: 添加测试策略章节
- [ ] O3: 添加性能考虑章节

---

**验证完成时间:** 2026-01-04  
**验证工具版本:** BMM Checklist v4.0  
**下一步:** 修复 Critical 问题后重新提交验证
