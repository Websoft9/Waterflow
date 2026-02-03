# ADR-0007: Workflow 存储与触发器架构设计

**状态:** 🟡 提议中  
**日期:** 2026-02-02  
**决策者:** 架构团队  
**相关 Story:** Story 1.10 (Schedule API), Story 1.11 (Webhook Trigger)

## 背景

Waterflow 当前面临关键的架构决策：
1. **YAML Workflow 定义应该如何存储？**（Git / 文件系统 / 数据库）
2. **触发器（Schedule/Webhook）与 Workflow 的关系？**（一体化 vs 分离）
3. **是否需要恢复 YAML 中的 `on` 字段？**（Story 1.3 已删除）
4. **Server-Agent 分布式架构下，YAML 如何分发到 100 台服务器？**

这些决策直接影响：
- GitOps 支持能力
- 多租户隔离方案
- 用户体验和学习成本
- 分布式执行的正确性和效率
- 定时任务的统一触发

## 关键架构澄清：Server-Agent 分布式模式

### Waterflow 分布式架构核心原则

**重要:** Waterflow 采用 Server-Agent 分布式架构，YAML 存储和执行模式与传统 CI/CD 工具有本质区别：

```
┌────────────────────────────────────────────────────┐
│         Server 端（中心节点）                       │
│  ┌──────────────────────────────────────────┐     │
│  │  YAML 存储（单一数据源）                  │     │
│  │  - Git 仓库 / 文件系统 / 数据库           │     │
│  │  - 只存储在 Server，Agent 永不访问        │     │
│  └──────────────────┬───────────────────────┘     │
│                     │                             │
│  ┌──────────────────▼───────────────────────┐     │
│  │  Temporal Server                         │     │
│  │  - Workflow 在 Server 端解析和执行        │     │
│  │  - Activity 路由到 Agent 端执行           │     │
│  │  - Schedule 统一时间触发                  │     │
│  └──────────────────┬───────────────────────┘     │
└─────────────────────┼──────────────────────────────┘
                      │ gRPC (Task Queue)
           ┌──────────┼──────────┐
           │          │          │
      ┌────▼───┐ ┌────▼───┐ ┌───▼────┐
      │Agent 1 │ │Agent 2 │ │Agent100│
      │linux-  │ │linux-  │ │linux-  │
      │amd64   │ │amd64   │ │amd64   │
      └────────┘ └────────┘ └────────┘
       只执行      只执行      只执行
      Activity   Activity   Activity
```

**核心设计决策：**

1. **YAML 永远只存储在 Server 端**
   - Git 仓库: Server clone 到本地
   - 文件系统: Server /opt/waterflow/workflows/
   - 数据库: Server SQLite/PostgreSQL
   - ❌ Agent **永远不需要**访问 YAML 文件

2. **Workflow 在 Server 端解析和执行**
   ```go
   // Server 端 Temporal Worker
   func RunWorkflowExecutor(ctx workflow.Context, wf *dsl.Workflow) error {
       // ✅ Workflow 逻辑在 Server 端执行
       // ✅ YAML 已在 Server 端解析为 wf 对象
       for _, job := range wf.Jobs {
           // 只有 Activity 才分发到 Agent
           activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
               TaskQueue: job.RunsOn,  // "linux-amd64" → 路由到 Agent
           })
           workflow.ExecuteActivity(activityCtx, "ExecuteStepActivity", step)
       }
   }
   ```

3. **Agent 只是执行器（Worker），只接收 Activity 参数**
   ```go
   // Agent 端 Temporal Worker
   func (a *Activities) ExecuteStepActivity(ctx context.Context, step *dsl.Step) error {
       // ✅ Agent 只接收已解析的 Step 参数
       // ✅ 不需要访问 YAML 文件
       return a.nodeRegistry.Execute(step.Uses, step.With)
   }
   ```

4. **Schedule 统一在 Server 端触发**
   ```go
   // Server 端创建 Schedule
   scheduleClient.Create(ctx, &client.ScheduleOptions{
       ID: "daily-backup",
       Spec: &client.ScheduleSpec{
           CronExpressions: []string{"0 2 * * *"},
           Timezone:        "UTC",  // ✅ 统一时区
       },
       Action: &client.ScheduleWorkflowAction{
           Workflow:  RunWorkflowExecutor,
           Args:      []interface{}{wf},  // ✅ 解析后的 Workflow 对象
           TaskQueue: "default",  // Workflow 在 Server 端执行
       },
   })
   ```

5. **100 台服务器场景处理**
   - ✅ Temporal Task Queue 自动负载均衡
   - ✅ 所有 Agent 监听同一个 Task Queue ("linux-amd64")
   - ✅ Temporal 保证任务只执行一次（分布式锁）
   - ✅ 失败自动重试，重新分配给其他 Agent
   - ✅ 定时任务只在 Server 端触发一次，然后分发到 100 台 Agent

**关键区别对比：**

| 维度 | GitHub Actions | Waterflow |
|------|---------------|-----------|
| **YAML 存储** | Git 仓库 | Server 端（Git/文件系统/数据库） |
| **Workflow 执行** | Runner 下载 YAML 执行 | Server 端解析，Agent 只执行 Activity |
| **定时触发** | 每个 Runner 独立判断 | Server 统一触发，分发到 Agent |
| **100 台服务器** | 每台独立下载 YAML | Server 解析一次，分发任务到 100 台 |
| **时间同步** | 每台服务器独立时钟 | Temporal Server 统一时间 |
| **失败重试** | Runner 独立重试逻辑 | Temporal 集中管理重试 |

---

## 行业最佳实践分析

### 主流产品对比

| 产品 | YAML 存储 | 触发器定义 | 架构模式 | 动态管理 |
|------|----------|-----------|---------|---------|
| **GitHub Actions** | Git 仓库 | YAML 内 `on` 字段 | 一体化 | ❌ 无 API |
| **GitLab CI** | Git 仓库 | YAML 内 `rules` | 一体化 | ⚠️ 有限 |
| **Argo Workflows** | K8s CRD (etcd) | 分离 (CronWorkflow) | 分离式 | ✅ 完整 API |
| **Airflow** | 代码文件 (dags/) | 代码内定义 | 一体化 | ⚠️ 需重启 |
| **Temporal** | 编译代码 | 分离 (Schedule API) | 分离式 | ✅ 完整 API |

### 关键发现

1. **CI/CD 工具（GitHub Actions/GitLab CI）**
   - 采用一体化设计（YAML 包含触发器）
   - 强依赖 Git 仓库存储
   - 适合单一 CI/CD 场景
   - 缺乏运行时动态管理能力

2. **工作流引擎（Argo/Temporal）**
   - 采用分离式设计（Workflow + Trigger 独立）
   - 支持完整的 REST API 动态管理
   - 适合复杂企业工作流场景
   - 更高的灵活性和扩展性

## 决策

采用 **「混合模式」架构**：
- ✅ **支持 YAML `on` 字段声明式定义**（借鉴 GitHub Actions）
- ✅ **同时提供 REST API 动态管理**（借鉴 Argo/Temporal）
- ✅ **推荐 Git 仓库存储 YAML**（支持多种存储方式）
- ✅ **Schedule 只存储引用，不存储 YAML**（避免数据冗余）

## 理由

### 核心优势

#### 1. **最佳的用户体验**

```yaml
# 方案 A: 声明式（GitHub Actions 风格，简单直观）
name: Backup Database
on:
  schedule:
    - cron: "0 2 * * *"
jobs:
  backup:
    runs-on: default
    steps:
      - name: Backup
        uses: shell@v1
        with:
          cmd: pg_dump mydb > backup.sql

# 方案 B: 命令式（API 驱动，灵活强大）
# YAML 只定义业务逻辑
name: Backup Database
jobs:
  backup:
    runs-on: default
    steps:
      - name: Backup
        uses: shell@v1

# 然后通过 API 动态创建触发器
POST /v1/schedules
{
  "workflowName": "Backup Database",
  "cron": "0 2 * * *"
}

# Waterflow 支持两种方式！
```

#### 2. **GitOps 最佳实践**

```bash
# 用户工作流
git clone https://github.com/org/workflows.git
cd workflows
vim backup.yaml  # 编辑工作流
git commit -m "Update backup schedule"
git push

# Waterflow Server 自动同步
# 1. 检测到 Git 仓库变更（Webhook）
# 2. 自动 git pull 拉取最新代码
# 3. 解析 on 字段，自动创建/更新 Schedule
# 4. 无需重启，热更新生效
```

#### 3. **灵活的存储策略**

| 存储方式 | 适用场景 | 优势 | 实现复杂度 |
|---------|---------|------|-----------|
| **Git 仓库（推荐）** | DevOps 团队、企业级 | 版本控制、Code Review、GitOps | 🟡 中等 |
| **文件系统** | 小团队、快速部署 | 简单直接、零依赖 | 🟢 简单 |
| **数据库** | SaaS 平台、多租户 | 集中管理、租户隔离 | 🔴 复杂 |

#### 4. **避免数据冗余**

```go
// ❌ 错误设计：Schedule 存储完整 YAML
type Schedule struct {
    ID       string
    Cron     string
    YAMLContent string  // 冗余！与 Git 仓库数据重复
}

// ✅ 正确设计：Schedule 只存储引用
type Schedule struct {
    ID           string
    WorkflowName string  // 引用，运行时动态加载 YAML
    Cron         string
    Timezone     string
}

// 运行时流程
func RunWorkflowExecutor(ctx workflow.Context, workflowName string) error {
    // 1. 根据 name 查找 YAML 文件
    yamlPath := filepath.Join("/opt/waterflow/workflows", workflowName + ".yaml")
    
    // 2. 实时读取（单一数据源）
    yamlContent, _ := os.ReadFile(yamlPath)
    
    // 3. 解析并执行
    wf, _ := dsl.Parse(yamlContent)
    return executeWorkflow(ctx, wf)
}
```

### 与其他方案对比

#### 方案 A: 完全一体化（GitHub Actions 风格）
```yaml
优势:
  ✅ 用户熟悉，学习成本低
  ✅ YAML 文件即完整定义

劣势:
  ❌ 缺乏运行时动态管理
  ❌ 无法一个 Workflow 配置多个触发器
  ❌ 修改 Schedule 必须编辑 YAML + Git 提交
  ❌ 不适合复杂企业场景

决策: ❌ 拒绝（灵活性不足）
```

#### 方案 B: 完全分离（当前 Story 1.3 设计）
```yaml
优势:
  ✅ 职责清晰
  ✅ 完整的 API 支持
  ✅ 符合 Temporal 原生模型

劣势:
  ❌ 学习成本高（需理解两套概念）
  ❌ 配置繁琐（先创建 Workflow，再创建 Schedule）
  ❌ 失去 GitOps 声明式优势

决策: ❌ 拒绝（用户体验差）
```

#### 方案 C: 混合模式（本 ADR 推荐）
```yaml
优势:
  ✅ 声明式 + 命令式双模式
  ✅ 既简单（on 字段）又灵活（API）
  ✅ 支持 GitOps 最佳实践
  ✅ 兼顾用户体验和企业需求

劣势:
  ⚠️ 实现复杂度稍高（需支持两种模式）
  ⚠️ 需处理 on 字段与 API 的冲突

决策: ✅ 采纳（最佳平衡）
```

## 后果

### 正面影响

✅ **降低学习成本**
- 熟悉 GitHub Actions 的用户可以直接使用 `on` 字段
- 无需理解 Workflow 和 Schedule 的分离概念

✅ **GitOps 天然支持**
- YAML 存储在 Git 仓库，版本控制、Code Review 开箱即用
- Webhook 自动同步，无需手动触发

✅ **企业级灵活性**
- 支持运行时 API 动态管理（超越 GitHub Actions）
- 一个 Workflow 可以有多个独立 Schedule

✅ **避免数据冗余**
- Schedule 只存储引用，YAML 是单一数据源
- 更新 YAML 自动影响所有相关 Schedule

### 负面影响

⚠️ **实现复杂度增加**
- 需要支持 Git 仓库同步机制
- 需要处理 `on` 字段解析和自动创建 Schedule
- 需要解决 YAML 变更时 Schedule 的同步问题

⚠️ **潜在冲突场景**
```yaml
# 场景：用户既在 YAML 定义 on.schedule，又通过 API 创建 Schedule
name: Backup
on:
  schedule:
    - cron: "0 2 * * *"  # YAML 声明

# 同时用户调用 API
POST /v1/schedules
{
  "workflowName": "Backup",
  "cron": "0 14 * * *"
}

# 期望结果：两个 Schedule 共存
# - backup-auto (来自 YAML)
# - backup-afternoon (来自 API)

# 实现策略：
# 1. YAML on 字段生成的 Schedule ID 固定格式：{workflowName}-auto
# 2. API 创建的 Schedule ID 由用户指定或自动生成
# 3. 互不冲突，可以共存
```

### 风险缓解

**风险 1: Git 仓库同步失败**
```go
// 缓解策略：本地缓存 + 降级机制
func SyncWorkflows() error {
    if err := gitPull(); err != nil {
        log.Warn("Git sync failed, using cached workflows")
        return nil  // 不中断服务
    }
    return reloadWorkflows()
}
```

**风险 2: YAML 变更导致 Schedule 不一致**
```go
// 缓解策略：自动同步 Schedule
func OnYAMLChanged(workflowName string) {
    // 1. 重新解析 YAML
    wf := parseYAML(workflowName)
    
    // 2. 对比 on 字段变更
    if wf.On.Schedule != nil {
        // 3. 更新或创建 auto Schedule
        scheduleClient.Update(ctx, workflowName+"-auto", wf.On.Schedule)
    } else {
        // 4. 删除 auto Schedule
        scheduleClient.Delete(ctx, workflowName+"-auto")
    }
}
```

**风险 3: 多节点 Server 的 Git 同步一致性**
```go
// 缓解策略：Webhook 通知 + 分布式锁
func HandleGitWebhook(w http.ResponseWriter, r *http.Request) {
    // 1. 获取分布式锁（避免并发 pull）
    lock := acquireLock("git-sync")
    defer lock.Release()
    
    // 2. 拉取最新代码
    gitPull()
    
    // 3. 通知其他节点（通过 Redis Pub/Sub）
    redis.Publish("workflow-reload", "backup.yaml")
}
```

## 实施计划

### Phase 1: 恢复 `on` 字段支持（Story 1.10 修订）

```yaml
# 1. 恢复 pkg/dsl/workflow.go 的 On 字段
type Workflow struct {
    Name string
    On   *TriggerConfig  // 重新添加
    Vars map[string]interface{}
    Jobs map[string]*Job
}

type TriggerConfig struct {
    Schedule []ScheduleConfig
    Webhook  []WebhookConfig
}

# 2. 更新 Story 1.3 的 YAML Parser
# 3. 更新相关测试用例
```

### Phase 2: Git 仓库存储支持（新 Story）

```go
// 1. 实现 Git 同步服务
type GitWorkflowStore struct {
    RepoURL    string
    LocalPath  string
    SyncPeriod time.Duration
}

func (s *GitWorkflowStore) Sync() error {
    // git clone or git pull
}

// 2. Webhook 监听器
func (s *GitWorkflowStore) HandleWebhook(payload GitHubWebhookPayload) {
    // 检测 workflows/ 目录变更
    // 触发同步
}
```

### Phase 3: on 字段自动创建 Schedule（Story 1.10）

```go
// Server 处理流程
func OnWorkflowUploaded(wf *dsl.Workflow) error {
    // 1. 验证 YAML
    if err := validator.Validate(wf); err != nil {
        return err
    }
    
    // 2. 检查 on.schedule 字段
    if wf.On != nil && wf.On.Schedule != nil {
        for _, sched := range wf.On.Schedule {
            // 3. 自动创建 Schedule
            scheduleClient.Create(ctx, &client.ScheduleOptions{
                ID: fmt.Sprintf("%s-auto-%d", wf.Name, idx),
                Spec: &client.ScheduleSpec{
                    CronExpressions: []string{sched.Cron},
                },
                Action: &client.ScheduleWorkflowAction{
                    Workflow: temporal.RunWorkflowExecutor,
                    Args:     []interface{}{wf.Name},  // 引用 name
                    TaskQueue: "default",
                },
            })
        }
    }
    
    return nil
}
```

### Phase 4: 多存储模式支持（可选）

```go
// 配置文件支持多种存储
type WorkflowStoreConfig struct {
    Type string  // "git", "filesystem", "database"
    
    // Git 模式
    GitRepoURL    string
    GitBranch     string
    GitWebhookSecret string
    
    // 文件系统模式
    FilesystemPath string
    
    // 数据库模式
    DatabaseDSN string
}
```

## 验收标准

### AC1: 支持 YAML on 字段声明式定义

```yaml
name: Backup
on:
  schedule:
    - cron: "0 2 * * *"
      timezone: "America/New_York"
jobs:
  backup:
    runs-on: default
    steps:
      - name: Backup
        uses: shell@v1

# Server 自动创建 Schedule ID: "Backup-auto-0"
```

### AC2: Git 仓库存储支持

```bash
# 配置 Server
export WATERFLOW_WORKFLOW_STORE=git
export WATERFLOW_GIT_REPO=https://github.com/org/workflows.git

# Server 启动时自动 clone
# Webhook 触发时自动 pull
# YAML 变更时自动更新 Schedule
```

### AC3: API 动态管理（与 on 字段共存）

```bash
# YAML 中已定义 on.schedule（自动创建 backup-auto-0）
# 用户可以通过 API 创建额外的 Schedule

POST /v1/schedules
{
  "id": "backup-afternoon",
  "workflowName": "Backup",
  "cron": "0 14 * * *"
}

# 结果：同一个 Workflow 有两个 Schedule
# - backup-auto-0 (来自 YAML)
# - backup-afternoon (来自 API)
```

### AC4: Schedule 不存储 YAML，运行时动态加载

```go
// Schedule 数据结构
type Schedule struct {
    ID           string  // "backup-auto-0"
    WorkflowName string  // "Backup"（引用）
    Cron         string
}

// 运行时加载
func RunWorkflowExecutor(ctx workflow.Context, workflowName string) error {
    yamlPath := "/opt/waterflow/workflows/" + workflowName + ".yaml"
    yamlContent := os.ReadFile(yamlPath)  // 实时读取
    wf := dsl.Parse(yamlContent)
    return execute(ctx, wf)
}
```

## 参考资料

- [GitHub Actions Workflow Syntax](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions)
- [Argo Workflows Architecture](https://argoproj.github.io/argo-workflows/architecture/)
- [Temporal Schedules Documentation](https://docs.temporal.io/workflows#schedule)
- [GitOps Principles](https://opengitops.dev/)
- [Story 1.3: YAML DSL 解析和验证](../sprint-artifacts/1-3-yaml-dsl-parsing-and-validation.md)
- [Story 1.10: Schedule API 实现](../sprint-artifacts/1-10-schedule-api-implementation.md)

---

**决策状态:** 🟡 提议中 → 等待团队评审  
**下一步:** 评审通过后更新 Story 1.3 和 Story 1.10
