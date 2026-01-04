# Story 5.4 CLI Status 命令验证报告

**验证日期:** 2026-01-04  
**验证者:** Senior Quality Validator  
**Story 文件:** `/data/Waterflow/docs/sprint-artifacts/5-4-cli-status-command.md`  
**Epic:** Epic 5 - 客户端工具和 SDK  
**验证清单:** `.bmad/bmm/workflows/4-implementation/create-story/checklist.md`

---

## 1. 执行摘要

**总体评级:** 6.5/10  
**是否建议批准:** ⚠️ **有条件批准** - 必须修复 4 个关键缺陷

**关键发现:**
- ✅ **优秀:** Epic 上下文完整,AC 定义详细,用户体验设计优秀
- ✅ **优秀:** Story 5-1/5-3 复用说明清晰,输出格式统一
- ✅ **优秀:** 表格和颜色设计符合 CLI 最佳实践
- 🔴 **严重:** 依赖 Story 5-3 未实现的 GetWorkflowStatus() 方法
- 🔴 **严重:** 状态字段名映射错误 - Story 1.9 无 `conclusion` 字段
- 🔴 **严重:** Task 3 假设 GetWorkflowStatus() 已存在于 Story 5-3
- ⚠️ **中等:** tablewriter 库未实际使用,应删除或实现
- ⚠️ **轻微:** Watch 模式清屏逻辑在非 TTY 环境可能导致混乱

**灾难性缺陷 (必须修复):**
1. **依赖假设错误** - Story 5-3 未实现 GetWorkflowStatus(),本 Story 无法复用
2. **API 字段映射错误** - Story 1.9 响应无 `conclusion` 字段,仅 status 字段
3. **架构要求未实现** - architecture.md 要求使用 tablewriter,但代码未使用
4. **状态值映射不明确** - Story 1.9 有 6 个状态值,本 Story 仅映射 5 个

---

## 2. 源文档分析

### 2.1 Epic 5 上下文覆盖度: ⭐⭐⭐⭐⭐ (5/5)

✅ **覆盖完整:**
- Epic 5 Story 5.4 定义: "查询工作流状态,了解执行进度"
- AC 精准匹配 Epic 要求:
  - AC1: 基础状态查询
  - AC2: 显示执行进度 (Jobs/Steps)
  - AC3: 持续监控 (--watch)
  - AC4: 格式化输出 (text/json/yaml)
  - AC5: 错误处理
  - AC6: 状态符号和颜色
- Context 清晰说明业务价值和技术定位

✅ **前置依赖明确:**
- Story 5.1 (CLI 框架) - ✅ done
- Story 5.3 (submit 命令) - ✅ done (但有误解,见 3.1)
- Story 1.9 (REST API) - ✅ done
- Story 1.8 (Temporal SDK) - ✅ done

### 2.2 Story 5-1 HTTP Client 依赖检查: ⭐⭐⭐ (3/5) ⚠️ **部分问题**

✅ **复用说明清晰:**
- Task 3 明确扩展 `pkg/client/client.go`
- 实现 GetWorkflowStatus() 方法
- 复用 HTTP 客户端、配置管理、输出格式化

⚠️ **关键误解:**

**问题 1: Story 5-1 没有实现 GetWorkflowStatus()**

Story 5-1 Task 6 仅实现了基础 HTTP Client:
```go
// Story 5-1 Task 6 实际实现
type Client struct {
    baseURL    string
    apiKey     string
    timeout    time.Duration
    debug      bool
    httpClient *http.Client
}

func New(baseURL, apiKey string, timeout time.Duration, debug bool) *Client {
    return &Client{
        baseURL:    baseURL,
        apiKey:     apiKey,
        timeout:    timeout,
        debug:      debug,
        httpClient: &http.Client{Timeout: timeout},
    }
}

// 仅实现了 parseError() 辅助方法,没有任何 API 调用方法
```

**问题 2: Story 5-3 也未实现 GetWorkflowStatus()**

Story 5-3 实际实现:
- SubmitWorkflow() - POST /v1/workflows
- parseError() - 错误解析

**Story 5-3 的 AC3/AC4 (--wait/--follow) 也依赖未实现的方法:**
```go
// Story 5-3 AC3 代码中使用了未实现的方法
status, err := c.GetWorkflowStatus(workflowID)
logs, err := c.GetWorkflowLogs(workflowID, true)
```

**结论:** Story 5-3 的 --wait/--follow 也是"待实现"状态,当前应标记为 ready-for-dev 而非 done。

### 2.3 Story 1.9 API 集成检查: ⭐⭐ (2/5) 🔴 **严重缺陷**

**🔴 致命错误 1 - conclusion 字段不存在:**

**Story 5-4 AC1 中的假设:**
```bash
Status:      completed
Conclusion:  success  # ← 这个字段不存在!
```

**Story 1.9 AC2 实际 API 响应:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "run_id": "temporal-run-id-123",
  "name": "Deploy App",
  "status": "running",  // ← 只有 status 字段!
  "created_at": "2025-12-18T10:30:45Z",
  "started_at": "2025-12-18T10:30:46Z",
  "completed_at": null,
  "duration_seconds": null,
  "vars": {...},
  "jobs": [...]
}
```

**Story 1.9 AC2 status 字段定义:**
```
status 字段取值:
- `pending` - 工作流已提交但未开始
- `running` - 正在执行
- `completed` - 已完成 (成功)
- `failed` - 已完成 (失败)
- `cancelled` - 已取消
- `timeout` - 已超时
```

**❌ 没有 conclusion 字段!**

**Story 1.9 说明:**
> **And** conclusion 字段取值 (仅 status=completed 时):

这是 Story 1.9 的设计意图,但**未在 AC2 响应示例中体现**,且 Task 2 代码实现也**未包含 conclusion 字段**:

```go
// Story 1.9 Task 2 实际实现
type WorkflowStatusResponse struct {
    ID             string                 `json:"id"`
    RunID          string                 `json:"run_id"`
    Name           string                 `json:"name"`
    Status         string                 `json:"status"`
    // ❌ 没有 Conclusion 字段!
    CreatedAt      string                 `json:"created_at"`
    // ...
}
```

**根本原因:** Story 1.9 在 AC2 描述中提到 conclusion,但代码实现未包含。

**🔴 致命错误 2 - 状态值映射不完整:**

**Story 5-4 AC6 定义:**
```go
"running"    → "→" (蓝色)
"completed"  → "✓" (绿色)
"failed"     → "✗" (红色)
"cancelled"  → "⊗" (黄色)
"pending"    → "○" (灰色)
// ❌ 缺少 "timeout" 状态!
```

**Story 1.9 AC2 定义:**
```
- `pending`
- `running`
- `completed`
- `failed`
- `cancelled`
- `timeout`  // ← Story 5-4 未映射此状态
```

### 2.4 Architecture tablewriter 检查: ⭐⭐ (2/5) ⚠️ **未实现**

**architecture.md #9.1 明确要求:**
```go
github.com/olekukonko/tablewriter // 表格输出
```

**Story 5-4 实际使用:**
- Task 5 使用手动格式化 (fmt.Printf)
- Task 9 添加 `github.com/fatih/color` (颜色库)
- **未使用 tablewriter 库**

**问题:**
1. **架构要求未实现** - tablewriter 用于表格输出 (如 kubectl get)
2. **手动格式化风险** - 对齐、列宽控制容易出错
3. **依赖冗余** - 既声明 tablewriter,又不使用

**可能的误解:**
- Story 编写者认为 Jobs/Steps 层级显示不需要表格
- 但 architecture.md 要求表格输出,应至少在 compact 模式使用

**建议:**
- **方案 1:** 使用 tablewriter 实现 Jobs 表格 (推荐)
- **方案 2:** 从 architecture.md 删除 tablewriter 依赖

### 2.5 与 Story 5-3 的一致性: ⭐⭐⭐ (3/5) ⚠️ **依赖假设错误**

✅ **输出格式一致:**
- 复用 Story 5-1 的 output.Formatter
- text/json/yaml 格式统一
- --quiet 模式模式一致

✅ **错误处理模式一致:**
- formatStatusError() 格式与 Story 5-3 类似
- ExitError{Code: 1} 模式统一

⚠️ **依赖假设错误:**

**Story 5-4 Developer Context 说明:**
```markdown
**复用现有组件:**
- Story 5.1: HTTP 客户端、配置管理、输出格式化
- Story 5.3: GetWorkflowStatus() 方法 (已在 submit --wait 中实现)  ← ❌ 错误!
- Story 1.9: `GET /v1/workflows/{id}` API
```

**Story 5-3 实际情况:**
- Story 5-3 AC3 (--wait) 代码**使用了** GetWorkflowStatus()
- 但 Story 5-3 **未实现** GetWorkflowStatus()
- Story 5-3 应该标记为 ready-for-dev,而非 done

**结论:** Story 5-4 依赖的 GetWorkflowStatus() 需要在本 Story 实现,不能假设 Story 5-3 已实现。

---

## 3. 灾难性缺陷识别 (CRITICAL)

### 3.1 依赖假设错误 🔴 **CRITICAL - 必须修复**

**缺陷 ID:** DISASTER-1  
**严重程度:** 🔴 Critical - 阻塞实现  
**影响范围:** Task 3, Developer Context

**问题描述:**

Story 5-4 假设 GetWorkflowStatus() 方法在 Story 5-3 中实现:

```markdown
### Developer Context
**复用现有组件:**
- Story 5.3: GetWorkflowStatus() 方法 (已在 submit --wait 中实现)
```

**实际情况:**

**1. Story 5-1 实现:**
```go
// pkg/client/client.go (Story 5-1 Task 6)
type Client struct {
    baseURL    string
    apiKey     string
    timeout    time.Duration
    debug      bool
    httpClient *http.Client
}

// ❌ 没有任何 API 方法,仅有基础结构
```

**2. Story 5-3 使用但未实现:**
```go
// Story 5-3 AC3 代码示例
status, err := c.GetWorkflowStatus(workflowID)  // ← 调用但未实现!
```

**3. Story 5-4 Task 3 假设已存在:**
```markdown
### Task 3: HTTP 客户端 GetWorkflowStatus 方法 (AC1)
- [ ] 扩展 `cmd/waterflow-cli/pkg/client/client.go`
```

**根本原因:**
- Story 5-1: 临时 HTTP Client,仅基础结构
- Story 5-3: 使用 GetWorkflowStatus(),但标记为"待实现"
- Story 5-4: 假设 Story 5-3 已实现,导致循环依赖

**影响:**
- LLM 开发者会困惑 GetWorkflowStatus() 是否已实现
- 可能重复实现或遗漏实现
- Story 5-3 应标记为 ready-for-dev 而非 done

**修复方案:**

**方案 1 (推荐): 明确 Story 5-4 实现 GetWorkflowStatus()**
```markdown
### Developer Context

**HTTP Client 方法实现责任:**
- Story 5-1: 基础 HTTP Client 结构 (New, parseError)
- Story 5-3: SubmitWorkflow() 方法 (POST /v1/workflows)
- **Story 5-4: GetWorkflowStatus() 方法 (GET /v1/workflows/{id})** ← 本 Story 实现
- Story 5-5: GetWorkflowLogs() 方法 (GET /v1/workflows/{id}/logs)

**注意:**
- Story 5-3 的 --wait/--follow 功能当前**未完全实现**
- GetWorkflowStatus() 和 GetWorkflowLogs() 方法由后续 Story 提供
- Story 5-3 应重新评估状态 (ready-for-dev vs done)
```

### 3.2 API 响应字段映射错误 🔴 **CRITICAL - 必须修复**

**缺陷 ID:** DISASTER-2  
**严重程度:** 🔴 Critical - 实现错误  
**影响范围:** AC1, AC2, Task 3, Task 5

**问题描述:**

Story 5-4 假设 API 响应包含 `conclusion` 字段:

**AC1 示例:**
```bash
Status:      completed
Conclusion:  success  # ← Story 1.9 API 没有此字段!
```

**Task 3 代码:**
```go
type WorkflowStatus struct {
    ID              string `json:"id"`
    Status          string `json:"status"`
    Conclusion      string `json:"conclusion,omitempty"`  // ← 不存在!
    // ...
}
```

**Story 1.9 AC2 实际响应:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "running",  // ← 只有 status 字段
  // ❌ 没有 conclusion 字段
  "jobs": [...]
}
```

**Story 1.9 status 字段说明:**
```
- `completed` - 已完成 (成功)
- `failed` - 已完成 (失败)
```

**结论:** status 已包含成功/失败信息,无需 conclusion 字段。

**修复方案:**

**方案 1 (推荐): 删除 conclusion 字段,仅使用 status**

**修改 AC1:**
```bash
# 运行中工作流
Status:      running
Created:     2026-01-04T10:30:45Z
Started:     2026-01-04T10:30:46Z
Duration:    2m 15s

# 成功工作流
Status:      completed  (绿色)
Completed:   2026-01-04T10:05:30Z
Duration:    5m 29s

# 失败工作流
Status:      failed  (红色)
Completed:   2026-01-04T09:02:15Z
Duration:    2m 14s
View logs:   waterflow logs def-789
```

**修改 Task 3:**
```go
type WorkflowStatus struct {
    ID              string `json:"id"`
    RunID           string `json:"run_id"`
    Name            string `json:"name"`
    Status          string `json:"status"`  // ← 仅 status 字段
    // ❌ 删除 Conclusion 字段
    CreatedAt       string `json:"created_at"`
    // ...
}
```

**修改 Task 5:**
```go
func (f *Formatter) printStatusText(status *client.WorkflowStatus, compact bool) error {
    // 状态
    statusColor := f.getStatusColor(status.Status)
    fmt.Printf("Status:      %s\n", statusColor.Sprint(status.Status))
    
    // ❌ 删除 Conclusion 显示
    // if status.Conclusion != "" {
    //     fmt.Printf("Conclusion:  %s\n", ...)
    // }
    
    // ...
}
```

### 3.3 状态值映射不完整 🔴 **CRITICAL - 必须修复**

**缺陷 ID:** DISASTER-3  
**严重程度:** 🔴 Critical - 功能遗漏  
**影响范围:** AC6, Task 5

**问题描述:**

**Story 1.9 AC2 定义 6 个状态:**
```
- `pending`
- `running`
- `completed`
- `failed`
- `cancelled`
- `timeout`  // ← Story 5-4 未映射!
```

**Story 5-4 AC6 仅映射 5 个:**
```go
"running"    → "→" (蓝色)
"completed"  → "✓" (绿色)
"failed"     → "✗" (红色)
"cancelled"  → "⊗" (黄色)
"pending"    → "○" (灰色)
// ❌ 缺少 "timeout" 状态映射!
```

**影响:**
- timeout 状态显示为 "unknown" 或默认符号
- 用户无法区分 timeout 和其他状态

**修复方案:**

**添加 timeout 状态映射:**

**修改 AC6:**
```markdown
### AC6: 状态符号和颜色

**状态符号定义:**
- `✓` - 成功 (completed,绿色)
- `→` - 正在执行 (running,蓝色)
- `✗` - 失败 (failed,红色)
- `○` - 待执行 (pending,灰色)
- `⊗` - 已取消 (cancelled,黄色)
- `⏱` - 超时 (timeout,黄色)  ← 新增
```

**修改 Task 5:**
```go
func (f *Formatter) getStatusSymbol(status string) string {
    if f.noColor {
        switch status {
        case "completed":
            return "[✓]"
        case "running":
            return "[→]"
        case "failed":
            return "[✗]"
        case "cancelled":
            return "[⊗]"
        case "timeout":
            return "[⏱]"  // ← 新增
        default:
            return "[○]"
        }
    }
    
    switch status {
    case "completed":
        return color.GreenString("✓")
    case "running":
        return color.BlueString("→")
    case "failed":
        return color.RedString("✗")
    case "cancelled":
        return color.YellowString("⊗")
    case "timeout":
        return color.YellowString("⏱")  // ← 新增
    default:
        return color.New(color.FgHiBlack).Sprint("○")
    }
}
```

### 3.4 tablewriter 依赖未使用 ⚠️ **MEDIUM - 应当修复**

**缺陷 ID:** DISASTER-4  
**严重程度:** ⚠️ Medium - 架构不一致  
**影响范围:** architecture.md, Task 5, Task 9

**问题描述:**

**architecture.md #9.1 要求:**
```go
github.com/olekukonko/tablewriter // 表格输出
```

**Story 5-4 实际实现:**
- Task 5: 手动格式化 (fmt.Printf)
- Task 9: 添加 color 库
- ❌ 未使用 tablewriter

**问题:**
1. **架构要求违反** - 声明但不使用
2. **表格输出质量** - 手动格式化难以对齐
3. **依赖冗余** - go.mod 包含但未导入

**参考: kubectl get pods 输出**
```
NAME                     READY   STATUS    RESTARTS   AGE
nginx-6799fc88d8-abc     1/1     Running   0          5m
redis-74c8f9d8c9-def     1/1     Running   2          10m
```

**修复方案:**

**方案 1 (推荐): 使用 tablewriter 实现 compact 模式**

```markdown
### AC2: 显示执行进度

**紧凑模式 (表格输出):**
```bash
$ waterflow status --compact 550e8400-e29b-41d4-a716-446655440000
Workflow: Multi-Step Deployment
ID:       550e8400-e29b-41d4-a716-446655440000
Status:   running

Jobs:
+--------+-----------+----------+
| NAME   | STATUS    | DURATION |
+--------+-----------+----------+
| build  | completed | 2m 10s   |
| deploy | running   | 1m 10s   |
| verify | pending   | -        |
+--------+-----------+----------+
```

**修改 Task 5:**
```go
import "github.com/olekukonko/tablewriter"

func (f *Formatter) printStatusText(status *client.WorkflowStatus, compact bool) error {
    // ... 基本信息 ...
    
    // Jobs 表格 (compact 模式)
    if compact && len(status.Jobs) > 0 {
        table := tablewriter.NewWriter(os.Stdout)
        table.SetHeader([]string{"Name", "Status", "Duration"})
        table.SetBorder(true)
        
        for _, job := range status.Jobs {
            duration := calculateJobDuration(job)
            table.Append([]string{
                job.Name,
                f.colorizeStatus(job.Status),
                duration,
            })
        }
        table.Render()
    } else {
        // 层级显示 (非 compact 模式,使用现有代码)
        f.printJobsHierarchy(status.Jobs)
    }
}
```

**方案 2: 从 architecture.md 删除 tablewriter**
- 如果不使用表格输出,从架构文档删除依赖
- 不推荐,因为表格输出是 CLI 最佳实践

---

## 4. LLM 优化分析

### 4.1 清晰度问题

**问题 1: GetWorkflowStatus() 实现责任不明确**

**当前描述:**
```markdown
**复用现有组件:**
- Story 5.3: GetWorkflowStatus() 方法 (已在 submit --wait 中实现)
```

**优化建议:**
```markdown
**HTTP Client 方法实现责任分配:**

| Story | 方法 | 端点 | 状态 |
|-------|------|------|------|
| 5-1 | New(), parseError() | - | ✅ done |
| 5-3 | SubmitWorkflow() | POST /v1/workflows | ✅ done |
| **5-4** | **GetWorkflowStatus()** | GET /v1/workflows/{id} | **本 Story 实现** |
| 5-5 | GetWorkflowLogs() | GET /v1/workflows/{id}/logs | 待实现 |

**注意:** Story 5-3 的 --wait/--follow 依赖 Story 5-4/5-5 的方法,当前为占位实现。
```

**Token 节省:** 5 句 → 1 表格

**问题 2: API 字段映射说明冗长**

**当前描述:**
```markdown
**And** status 字段取值:
- `pending` - 工作流已提交但未开始
- `running` - 正在执行
- `completed` - 已完成 (成功)
- `failed` - 已完成 (失败)
- `cancelled` - 已取消
- `timeout` - 已超时

**And** conclusion 字段取值 (仅 status=completed 时):
- `success` - 成功
- `failure` - 失败
- `cancelled` - 取消
- `timeout` - 超时
```

**优化建议:**
```markdown
**状态字段 (status):** pending/running/completed/failed/cancelled/timeout

**注意:** Story 1.9 API 无 conclusion 字段,status 已包含成功/失败信息。
```

**Token 节省:** 15 行 → 3 行

### 4.2 实现细节缺失

**问题 1: Watch 模式退出条件不明确**

**当前代码:**
```go
// isTerminalStatus 判断是否为终止状态
func isTerminalStatus(status string) bool {
    return status == "completed" || status == "failed" || status == "cancelled"
}
```

**问题:** timeout 状态是否退出?

**优化建议:**
```go
// isTerminalStatus 判断工作流是否已终止
// 终止状态: completed, failed, cancelled, timeout
func isTerminalStatus(status string) bool {
    terminal := map[string]bool{
        "completed": true,
        "failed":    true,
        "cancelled": true,
        "timeout":   true,
    }
    return terminal[status]
}
```

**问题 2: 清屏逻辑未考虑非 TTY 环境**

**当前代码:**
```go
func clearScreen() {
    fmt.Print("\033[H\033[2J")  // ← 非 TTY 环境显示乱码
}
```

**优化建议:**
```go
func clearScreen() {
    if !isTerminal() {
        fmt.Println("\n---")  // 非 TTY 使用分隔符
        return
    }
    fmt.Print("\033[H\033[2J")
}
```

### 4.3 集成点模糊

**问题: Story 1.9 API 集成细节不足**

**当前描述:**
```markdown
**依赖 Server API:**
- `GET /v1/workflows/{id}` - 查询工作流状态
```

**优化建议:**
```markdown
**API 集成参考 (Story 1.9 AC2):**

**请求:**
```
GET /v1/workflows/{id}
Authorization: Bearer {api_key}
```

**响应 (200 OK):**
```json
{
  "id": "uuid",
  "status": "running|completed|failed|cancelled|timeout|pending",
  "jobs": [{"id": "job1", "status": "...", "steps": [...]}]
}
```

**错误响应:**
- 404: Workflow not found
- 401: Invalid API key
- 500: Server error
```

---

## 5. 具体改进建议

### 5.1 必须修复 (Critical)

**修复 1: 明确 GetWorkflowStatus() 实现责任**

**位置:** Developer Context

**原文:**
```markdown
**复用现有组件:**
- Story 5.3: GetWorkflowStatus() 方法 (已在 submit --wait 中实现)
```

**修改为:**
```markdown
**HTTP Client 方法实现责任:**

| Story | 方法 | 说明 |
|-------|------|------|
| 5-1 | 基础结构 | New(), parseError() |
| 5-3 | SubmitWorkflow() | POST /v1/workflows |
| **5-4** | **GetWorkflowStatus()** | **GET /v1/workflows/{id} (本 Story 实现)** |
| 5-5 | GetWorkflowLogs() | GET /v1/workflows/{id}/logs |

**重要说明:**
- Story 5-3 的 --wait/--follow 功能依赖本 Story 的 GetWorkflowStatus()
- Story 5-3 应在 Story 5-4/5-5 完成后重新测试
```

**修复 2: 删除 conclusion 字段,仅使用 status**

**位置:** AC1, Task 3, Task 5

**AC1 修改:**
```bash
# 删除所有 Conclusion 行
Status:      completed  (绿色)
# Conclusion:  success  ← 删除

Status:      failed  (红色)
# Conclusion:  failure  ← 删除
```

**Task 3 修改:**
```go
type WorkflowStatus struct {
    ID              string `json:"id"`
    Status          string `json:"status"`
    // ❌ 删除以下行:
    // Conclusion      string `json:"conclusion,omitempty"`
    // ...
}
```

**Task 5 修改:**
```go
func (f *Formatter) printStatusText(...) {
    fmt.Printf("Status:      %s\n", statusColor.Sprint(status.Status))
    
    // ❌ 删除以下代码块:
    // if status.Conclusion != "" {
    //     conclusionColor := f.getConclusionColor(status.Conclusion)
    //     fmt.Printf("Conclusion:  %s\n", conclusionColor.Sprint(status.Conclusion))
    // }
}

// ❌ 删除 getConclusionColor() 函数
```

**修复 3: 添加 timeout 状态映射**

**位置:** AC6, Task 5

**AC6 添加:**
```markdown
**状态符号定义:**
- `✓` - completed (绿色)
- `→` - running (蓝色)
- `✗` - failed (红色)
- `○` - pending (灰色)
- `⊗` - cancelled (黄色)
- `⏱` - timeout (黄色)  ← 新增
```

**Task 5 添加:**
```go
case "timeout":
    return color.YellowString("⏱")  // 在 getStatusSymbol() 中添加
```

**修复 4: 实现 tablewriter 或删除依赖**

**方案 A (推荐): 实现 compact 模式表格**

**AC2 添加示例:**
```bash
$ waterflow status --compact <id>
Workflow: Multi-Step Deployment
Status:   running

Jobs:
+--------+-----------+----------+
| NAME   | STATUS    | DURATION |
+--------+-----------+----------+
| build  | completed | 2m 10s   |
| deploy | running   | 1m 10s   |
| verify | pending   | -        |
+--------+-----------+----------+
```

**Task 5 添加表格实现:**
```go
import "github.com/olekukonko/tablewriter"

if compact {
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"Name", "Status", "Duration"})
    for _, job := range status.Jobs {
        table.Append([]string{job.Name, job.Status, ...})
    }
    table.Render()
}
```

**方案 B: 删除 tablewriter 依赖**
- 从 architecture.md 删除 tablewriter
- 从 Task 9 删除 tablewriter
- 保持当前手动格式化

### 5.2 应当改进 (Should)

**改进 1: 优化 Watch 模式清屏逻辑**

**位置:** Task 4

**当前代码:**
```go
func clearScreen() {
    fmt.Print("\033[H\033[2J")
}
```

**修改为:**
```go
func clearScreen() {
    if !isTerminal() {
        fmt.Println("\n--- Status Update ---")
        return
    }
    fmt.Print("\033[H\033[2J")
}
```

**改进 2: 添加 API 集成详细说明**

**位置:** Developer Context

**添加:**
```markdown
### API 集成细节 (Story 1.9 AC2)

**请求:**
```http
GET /v1/workflows/{id}
Authorization: Bearer {api_key}
```

**响应字段映射:**
- `status`: running/completed/failed/cancelled/timeout/pending
- `jobs[].status`: Job 状态
- `jobs[].steps[].status`: Step 状态

**错误处理:**
- 404 → "Workflow not found"
- 401 → "Invalid API key"
- 500 → "Server error"
```

**改进 3: 明确轮询间隔和性能影响**

**位置:** Dev Notes

**添加:**
```markdown
### 性能考虑

**轮询间隔:**
- 默认: 2 秒 (balance 实时性和服务器负载)
- 最小: 1 秒 (避免过度轮询)
- 推荐: 2-5 秒

**服务器负载:**
- 单个 watch 命令: 0.5 req/s
- 100 并发 watch: 50 req/s
- 建议: Server 限流 100 req/s/IP

**后续优化 (Story 6+):**
- Server-Sent Events (SSE) 推送
- WebSocket 实时更新
- 减少 Server 轮询压力
```

### 5.3 可选优化 (Optional)

**优化 1: 添加状态过渡说明**

**位置:** Dev Notes

**添加:**
```markdown
### 工作流状态转换

```
pending → running → completed
                 ↘ failed
                 ↘ cancelled
                 ↘ timeout
```

**终止状态:** completed, failed, cancelled, timeout (watch 模式退出)
**非终止状态:** pending, running (watch 模式继续轮询)
```

**优化 2: 简化 AC 示例代码**

**当前 AC1:**
```bash
$ waterflow status 550e8400-e29b-41d4-a716-446655440000
Workflow: Deploy Application
ID:       550e8400-e29b-41d4-a716-446655440000
Run ID:   temporal-run-id-123
Status:      running
Conclusion:  -
Created:     2026-01-04T10:30:45Z
Started:     2026-01-04T10:30:46Z
Duration:    2m 15s
$ echo $?
0
```

**优化为:**
```bash
$ waterflow status <workflow-id>
Workflow: Deploy Application
Status:   running
Duration: 2m 15s
Created:  2026-01-04T10:30:45Z

View logs: waterflow logs <workflow-id>
```

**Token 节省:** 12 行 → 6 行

**优化 3: 合并重复的错误处理示例**

**当前 AC5:** 6 个错误场景,每个 10 行

**优化为:**
```markdown
### AC5: 友好的错误处理

**错误类型和建议:**

| 错误 | 退出码 | 建议 |
|------|--------|------|
| 工作流不存在 | 1 | Check ID or use 'waterflow list' |
| Server 连接失败 | 1 | Check server URL and connectivity |
| API 认证失败 | 1 | Set API key via --api-key or env |
| Server 内部错误 | 1 | Check server logs |
| 无效 ID 格式 | 2 | Expected UUID v4 format |

**详细示例见 Task 7**
```

**Token 节省:** 60 行 → 10 行

---

## 6. 最终评级

### 综合评分

| 维度 | 评分 | 说明 |
|------|------|------|
| Epic 上下文覆盖 | 5/5 | 完整覆盖 Epic 5 要求 |
| 前置依赖检查 | 2/5 | 🔴 依赖假设错误 (GetWorkflowStatus) |
| API 集成正确性 | 2/5 | 🔴 conclusion 字段不存在 |
| 架构一致性 | 3/5 | ⚠️ tablewriter 未实现 |
| 代码清晰度 | 4/5 | Task 定义清晰,但有误解 |
| 错误处理完整性 | 4/5 | 覆盖全面,缺少 timeout 映射 |
| 用户体验设计 | 5/5 | 优秀的 CLI UX,符号/颜色设计佳 |
| LLM 优化 | 3/5 | 存在冗余,可优化 token 效率 |

**总体评分:** 6.5/10

### 批准建议

**⚠️ 有条件批准 - 必须修复以下缺陷:**

**Critical (必须修复):**
1. ✅ **修复 1:** 明确 GetWorkflowStatus() 由本 Story 实现,不依赖 Story 5-3
2. ✅ **修复 2:** 删除所有 conclusion 字段引用,仅使用 status 字段
3. ✅ **修复 3:** 添加 timeout 状态映射 (`⏱` 黄色)
4. ✅ **修复 4:** 实现 tablewriter 表格 (compact 模式) 或删除依赖

**Should (强烈建议):**
5. 优化 Watch 模式清屏逻辑 (非 TTY 环境)
6. 添加 API 集成详细说明

**批准条件:**
- 完成所有 4 个 Critical 修复
- 更新 Story 5-3 状态说明 (依赖 5-4/5-5)
- 验证 Story 1.9 API 响应格式一致性

### 修复后重新评估

**修复后预估评分:** 8.5/10

**优势:**
- Epic 上下文覆盖完整
- 用户体验设计优秀
- 错误处理友好
- 与 Story 5-1/5-2 复用清晰

**改进领域:**
- API 集成文档可更详细
- LLM token 效率可优化
- 性能影响说明可增强

---

## 附录: 关键证据

### A. Story 5-1 HTTP Client 实际实现

**Story 5-1 Task 6 完整代码:**
```go
// cmd/waterflow-cli/pkg/client/client.go
package client

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type Client struct {
    baseURL    string
    apiKey     string
    timeout    time.Duration
    debug      bool
    httpClient *http.Client
}

func New(baseURL, apiKey string, timeout time.Duration, debug bool) *Client {
    return &Client{
        baseURL:    baseURL,
        apiKey:     apiKey,
        timeout:    timeout,
        debug:      debug,
        httpClient: &http.Client{Timeout: timeout},
    }
}

type ServerError struct {
    StatusCode int
    Code       string
    Message    string
    Details    map[string]interface{}
}

func (e *ServerError) Error() string {
    return fmt.Sprintf("server error: %d %s - %s", e.StatusCode, e.Code, e.Message)
}

func (c *Client) parseError(resp *http.Response) error {
    var apiErr struct {
        Error struct {
            Code    string                 `json:"code"`
            Message string                 `json:"message"`
            Details map[string]interface{} `json:"details"`
        } `json:"error"`
    }
    
    json.NewDecoder(resp.Body).Decode(&apiErr)
    
    return &ServerError{
        StatusCode: resp.StatusCode,
        Code:       apiErr.Error.Code,
        Message:    apiErr.Error.Message,
        Details:    apiErr.Error.Details,
    }
}

// ❌ 没有任何 API 调用方法 (SubmitWorkflow, GetWorkflowStatus 等)
```

### B. Story 1.9 AC2 实际响应格式

**Story 1.9 AC2 完整响应:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "run_id": "temporal-run-id-123",
  "name": "Deploy App",
  "status": "running",
  "created_at": "2025-12-18T10:30:45Z",
  "started_at": "2025-12-18T10:30:46Z",
  "completed_at": null,
  "duration_seconds": null,
  "vars": {
    "env": "production"
  },
  "jobs": [
    {
      "id": "deploy",
      "name": "deploy",
      "status": "running",
      "started_at": "2025-12-18T10:30:46Z",
      "completed_at": null,
      "runs_on": "linux-amd64",
      "steps": [
        {
          "name": "Deploy",
          "status": "running",
          "started_at": "2025-12-18T10:30:47Z",
          "completed_at": null,
          "conclusion": null
        }
      ]
    }
  ]
}
```

**❌ 顶层没有 conclusion 字段!**

**Story 1.9 AC2 Task 2 代码:**
```go
type WorkflowStatusResponse struct {
    ID             string                 `json:"id"`
    RunID          string                 `json:"run_id"`
    Name           string                 `json:"name"`
    Status         string                 `json:"status"`
    // ❌ 没有 Conclusion 字段
    CreatedAt      string                 `json:"created_at"`
    StartedAt      string                 `json:"started_at,omitempty"`
    CompletedAt    string                 `json:"completed_at,omitempty"`
    DurationSeconds *int                   `json:"duration_seconds,omitempty"`
    Vars           map[string]interface{} `json:"vars"`
    Jobs           []JobStatus            `json:"jobs"`
}
```

### C. architecture.md tablewriter 要求

**docs/architecture.md 行 1167:**
```go
// CLI 技术栈
github.com/spf13/cobra v1.7.0      // CLI 框架
github.com/olekukonko/tablewriter  // 表格输出
```

---

**验证完成时间:** 2026-01-04  
**验证者签名:** Senior Quality Validator  
**建议:** 有条件批准 - 修复 4 个 Critical 缺陷后可进入开发
