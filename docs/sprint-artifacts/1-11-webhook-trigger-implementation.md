# Story 1.11: Webhook Trigger 实现

Status: not-started

## Story

As a **工作流用户**,  
I want **通过 Webhook 触发工作流**,  
so that **工作流可以响应外部事件（如 Git Push、第三方通知等）自动执行**。

## Context

这是 Epic 1 的第十一个 Story，实现基于 HTTP Webhook 的事件驱动工作流触发功能。在 Story 1.10 的基础上，本 Story 提供独立的 Webhook 管理 API，允许外部系统通过 HTTP POST 触发工作流。

**设计理念（API 驱动）:**
- 工作流定义（YAML）与触发配置（Webhook）分离
- 同一工作流可配置多个不同的 Webhook（不同过滤规则）
- Webhook 独立管理：注册/启用/禁用/删除

**前置依赖:**
- Story 1.1 (Server 框架、日志系统) 已完成
- Story 1.2 (REST API 框架、错误处理) 已完成
- Story 1.3 (YAML 解析、Workflow 数据结构) 已完成
- Story 1.8 (Temporal SDK 集成) 已完成
- Story 1.9 (工作流管理 API) 已完成
- Story 1.10 (Schedule API) 已完成

**Epic 背景:**  
根据 [mvp-schedule-service-plan.md](./mvp-schedule-service-plan.md)，Waterflow 支持两种触发方式：
1. **定时触发** (Story 1.10): 基于 cron 表达式自动执行
2. **事件触发** (本 Story): 基于 HTTP Webhook 响应外部事件

**业务价值:**
- API 驱动架构：工作流定义与触发配置分离，更灵活
- 同一工作流可配置多个 Webhook（不同过滤规则）
- 支持事件驱动的自动化场景（CI/CD、监控告警、第三方集成）
- 提供安全的 Webhook 验证机制（签名验证）
- 支持 Webhook 过滤（分支、标签、路径等）
- 为 GitOps 集成奠定基础

**技术价值:**
- 复用 Schedule 管理框架，统一触发器管理
- 基于 Temporal 的可靠执行，无需担心 Webhook 丢失
- 支持异步处理，快速响应 Webhook 请求
- 完善的审计日志

## Acceptance Criteria

### AC1: 注册 Webhook Trigger

**Given** 用户已有工作流定义  
**When** 调用 POST /v1/triggers 配置 Webhook 触发  
**Then** 注册 Webhook Trigger 并返回详情

**请求示例:**
```bash
POST /v1/triggers
Content-Type: application/json

{
  "name": "deploy-on-push",
  "workflow_name": "Deploy",
  "type": "webhook",
  "filters": {
    "branches": ["main"],
    "paths": ["src/**"]
  },
  "enabled": true,
  "secret": "my-webhook-secret"
}
```

**响应示例:**
```json
{
  "trigger_id": "deploy-on-push",
  "workflow_name": "Deploy",
  "trigger_type": "webhook",
  "webhook_url": "https://waterflow.example.com/api/v1/webhooks/deploy-on-push",
  "webhook_secret": "my-webhook-secret",
  "filters": {
    "branches": ["main"]
  },
  "status": "enabled",
  "created_at": "2026-01-27T10:00:00Z"
}
```

**And** 生成唯一的 Webhook URL  
**And** 存储 Webhook Secret（用于签名验证）  
**And** 返回 201 Created 状态码

**错误处理:**
- Workflow 不存在 → 404 Not Found ("workflow 'Deploy' not found")
- Trigger ID 已存在 → 409 Conflict
- Secret 格式无效 → 400 Bad Request ("secret must be at least 16 characters")
- 过滤规则格式错误 → 400 Bad Request

### AC2: 接收 Webhook 请求

**Given** Webhook Trigger 已注册  
**When** 外部系统发送 POST 请求到 Webhook URL  
**Then** 验证请求并触发 Workflow

**Webhook 请求示例 (GitHub 格式):**
```bash
POST /api/v1/webhooks/deploy-on-push
Content-Type: application/json
X-Hub-Signature-256: sha256=abc123...
X-GitHub-Event: push

{
  "ref": "refs/heads/main",
  "before": "abc123",
  "after": "def456",
  "repository": {
    "name": "my-app",
    "full_name": "myorg/my-app"
  },
  "pusher": {
    "name": "developer"
  },
  "commits": [
    {
      "id": "def456",
      "message": "Update deployment script",
      "author": {
        "name": "developer"
      }
    }
  ]
}
```

**Webhook 响应:**
```json
{
  "trigger_id": "deploy-on-push",
  "workflow_id": "deploy-on-push-1706356800",
  "run_id": "abc123...",
  "status": "triggered",
  "message": "Workflow triggered successfully"
}
```

**And** 验证 HMAC 签名（X-Hub-Signature-256）  
**And** 应用过滤规则（分支、路径等）  
**And** 异步启动 Temporal Workflow  
**And** 快速响应（< 100ms）避免 Webhook 超时

**签名验证逻辑:**
```go
func verifySignature(payload []byte, signature string, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expectedMAC := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte("sha256="+expectedMAC))
}
```

**错误处理:**
- Trigger 不存在 → 404 Not Found
- 签名验证失败 → 401 Unauthorized
- 不匹配过滤规则 → 200 OK (ignored)
- Payload 格式错误 → 400 Bad Request

### AC3: Webhook 过滤规则

**Given** Webhook Trigger 配置了过滤规则  
**When** 收到 Webhook 请求  
**Then** 仅触发匹配规则的 Workflow

**支持的过滤规则:**

**1. 分支过滤:**
```yaml
name: Deploy
on:
  push:
    branches:
      - main
      - release/*  # 通配符支持
    branches-ignore:
      - develop
```

**2. 标签过滤:**
```yaml
name: Release
on:
  push:
    tags:
      - v*.*.*  # 仅版本标签
```

**3. 路径过滤:**
```yaml
name: Build Docs
on:
  push:
    paths:
      - docs/**
      - README.md
```

**4. 事件类型过滤:**
```yaml
name: PR Checker
on:
  push:
    types:
      - opened
      - synchronize
```

**And** 支持通配符匹配（*, **)  
**And** 支持排除规则（-ignore 后缀）  
**And** 多个条件 AND 关系（全部匹配才触发）  
**And** 不匹配时返回 200 OK (避免 Webhook 重试)

**过滤示例:**
```bash
# 匹配：main 分支 + docs 路径变更
POST /api/v1/webhooks/deploy-on-push
{
  "ref": "refs/heads/main",
  "commits": [{"modified": ["docs/README.md"]}]
}
→ 200 OK, Workflow Triggered

# 不匹配：develop 分支（在 ignore 列表）
POST /api/v1/webhooks/deploy-on-push
{
  "ref": "refs/heads/develop"
}
→ 200 OK, Ignored (not triggered)
```

### AC4: 列出所有 Webhook Triggers

**Given** 系统中有多个已注册的 Webhook Triggers  
**When** 调用 GET /v1/triggers  
**Then** 返回 Trigger 列表（分页）

**请求示例:**
```bash
GET /v1/triggers?type=webhook&status=enabled&limit=20

# 支持过滤参数
GET /v1/triggers?workflow_name=Deploy
GET /v1/triggers?type=webhook
GET /v1/triggers?status=disabled
```

**响应示例:**
```json
{
  "triggers": [
    {
      "id": "deploy-on-push",
      "workflow_name": "Deploy",
      "type": "webhook",
      "webhook_url": "https://waterflow.example.com/api/v1/webhooks/deploy-on-push",
      "filters": {
        "branches": ["main"]
      },
      "status": "enabled",
      "total_triggers": 120,
      "last_triggered_at": "2026-01-27T09:00:00Z",
      "created_at": "2026-01-01T10:00:00Z"
    },
    {
      "id": "notify-on-release",
      "workflow_name": "Release Notification",
      "type": "webhook",
      "webhook_url": "https://waterflow.example.com/api/v1/webhooks/notify-on-release",
      "filters": {
        "tags": ["v*.*.*"]
      },
      "status": "enabled",
      "total_triggers": 15
    }
  ],
  "total": 2,
  "limit": 20,
  "offset": 0
}
```

**And** 支持分页  
**And** 支持按类型/状态/工作流名称过滤  
**And** 包含统计信息（触发次数、最后触发时间）

### AC5: 查询 Webhook Trigger 详情

**Given** Webhook Trigger 已注册  
**When** 调用 GET /v1/triggers/:id  
**Then** 返回完整详情

**请求示例:**
```bash
GET /v1/triggers/deploy-on-push
```

**响应示例:**
```json
{
  "id": "deploy-on-push",
  "workflow_name": "Deploy",
  "workflow_yaml": "name: Deploy\non:\n  push:...",
  "type": "webhook",
  "webhook_url": "https://waterflow.example.com/api/v1/webhooks/deploy-on-push",
  "webhook_secret": "my-webhook-secret",
  "filters": {
    "branches": ["main"],
    "paths": ["src/**"]
  },
  "status": "enabled",
  "stats": {
    "total_triggers": 120,
    "successful_triggers": 118,
    "failed_triggers": 2,
    "last_triggered_at": "2026-01-27T09:00:00Z",
    "last_workflow_id": "deploy-on-push-1706356800"
  },
  "recent_triggers": [
    {
      "timestamp": "2026-01-27T09:00:00Z",
      "source_ip": "192.168.1.100",
      "event_type": "push",
      "ref": "refs/heads/main",
      "workflow_id": "deploy-on-push-1706356800",
      "status": "success"
    },
    {
      "timestamp": "2026-01-26T15:30:00Z",
      "source_ip": "192.168.1.100",
      "event_type": "push",
      "ref": "refs/heads/develop",
      "status": "ignored",
      "reason": "Branch not in filter"
    }
  ],
  "created_at": "2026-01-01T10:00:00Z",
  "updated_at": "2026-01-27T09:00:00Z"
}
```

**And** 包含完整的 YAML 定义  
**And** 包含 Webhook Secret（敏感信息，仅管理员可见）  
**And** 包含详细统计信息  
**And** 包含最近触发记录（最多 20 条）

### AC6: 禁用和启用 Webhook Trigger

**Given** Webhook Trigger 处于 enabled 状态  
**When** 调用 POST /v1/triggers/:id/disable  
**Then** Trigger 被禁用，不再响应 Webhook

**禁用请求:**
```bash
POST /v1/triggers/deploy-on-push/disable
Content-Type: application/json

{
  "reason": "Maintenance"
}
```

**禁用响应:**
```json
{
  "trigger_id": "deploy-on-push",
  "status": "disabled",
  "disabled_at": "2026-01-27T10:00:00Z",
  "reason": "Maintenance"
}
```

**启用请求:**
```bash
POST /v1/triggers/deploy-on-push/enable
```

**启用响应:**
```json
{
  "trigger_id": "deploy-on-push",
  "status": "enabled",
  "enabled_at": "2026-01-27T12:00:00Z"
}
```

**And** 禁用后的 Webhook 请求返回 200 OK (ignored)  
**And** 不影响已运行的 Workflow  
**And** 启用后立即生效

### AC7: 更新 Webhook Trigger 配置

**Given** Webhook Trigger 已注册  
**When** 调用 PATCH /v1/triggers/:id  
**Then** 更新 Trigger 配置

**请求示例:**
```bash
PATCH /v1/triggers/deploy-on-push
Content-Type: application/json

{
  "workflow_yaml": "name: Deploy\non:\n  push:\n    branches:\n      - main\n      - staging\njobs:...",
  "secret": "new-secret-key"
}
```

**响应示例:**
```json
{
  "trigger_id": "deploy-on-push",
  "workflow_name": "Deploy",
  "filters": {
    "branches": ["main", "staging"]
  },
  "webhook_secret": "new-secret-key",
  "updated_at": "2026-01-27T10:00:00Z"
}
```

**And** 更新会立即生效  
**And** Secret 更新后需要通知外部系统  
**And** 已运行的 Workflow 不受影响

### AC8: 删除 Webhook Trigger

**Given** Webhook Trigger 已注册  
**When** 调用 DELETE /v1/triggers/:id  
**Then** Trigger 被删除

**请求示例:**
```bash
DELETE /v1/triggers/deploy-on-push
```

**响应:**
```
204 No Content
```

**And** Webhook URL 失效（返回 404）  
**And** 本地元数据被删除  
**And** 已运行的 Workflow 不受影响  
**And** 删除后无法恢复（需要重新创建）

### AC9: Webhook 触发历史查询

**Given** 需要查看 Webhook 触发历史  
**When** 调用工作流列表 API 并按触发源过滤  
**Then** 通过统一的工作流查询获取触发记录

**查询示例:**
```bash
# 查询特定 Webhook Trigger 触发的所有工作流
curl "http://localhost:8080/v1/workflows?trigger_type=webhook&trigger_source=deploy-on-push&limit=50"

# 查询特定时间段的触发记录
curl "http://localhost:8080/v1/workflows?trigger_type=webhook&start_time=2026-01-27T00:00:00Z&end_time=2026-01-27T23:59:59Z"
```

**响应示例:**
```json
{
  "workflows": [
    {
      "id": "deploy-on-push-1706356800",
      "name": "deploy-production",
      "status": "running",
      "metadata": {
        "trigger_type": "webhook",
        "trigger_source": "deploy-on-push",
        "trigger_event": {
          "event_type": "push",
          "branch": "main",
          "commit": "abc123",
          "author": "john@example.com",
          "source_ip": "192.168.1.100",
          "timestamp": "2026-01-27T09:00:00Z"
        }
      },
      "created_at": "2026-01-27T09:00:01Z"
    }
  ],
  "total": 42,
  "page": 1,
  "per_page": 50
}
```

**And** 工作流元数据包含:
- `trigger_type` - 触发类型 (webhook, schedule, manual)
- `trigger_source` - Trigger ID
- `trigger_event` - Webhook payload 摘要（前 1KB）
- 来源 IP 地址
- 签名验证结果
- 过滤匹配结果

**And** 避免数据冗余，保持单一数据源 (工作流记录)  
**And** 与 Schedule API 设计对称 (Schedule 也不提供独立 logs)

**设计说明:**  
不提供独立的 `GET /v1/triggers/{id}/logs` API，原因：  
- Webhook 日志本质是"哪些请求触发了哪些工作流"  
- 这些信息已在工作流元数据中 (Event Sourcing)  
- 通过工作流 API 查询，保持数据一致性  
- 减少 API 数量，简化系统设计


## Tasks / Subtasks

### Task 1: Webhook Trigger Manager 核心逻辑
- [ ] 创建 pkg/trigger/types.go - Trigger 数据结构
- [ ] 创建 pkg/trigger/manager.go - Trigger 管理器
  - [ ] Create() - 创建 Webhook Trigger
  - [ ] List() - 列出 Triggers
  - [ ] Get() - 查询详情
  - [ ] Enable() - 启用
  - [ ] Disable() - 禁用
  - [ ] Update() - 更新配置
  - [ ] Delete() - 删除
- [ ] 创建 pkg/trigger/filter.go - 过滤规则引擎
  - [ ] MatchBranch() - 分支匹配
  - [ ] MatchTag() - 标签匹配
  - [ ] MatchPath() - 路径匹配
  - [ ] MatchEventType() - 事件类型匹配

**Trigger Manager 实现示例:**
```go
// pkg/trigger/manager.go
package trigger

import (
    "context"
    "crypto/hmac"
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "time"
    
    "github.com/Websoft9/waterflow/pkg/dsl"
    "github.com/Websoft9/waterflow/pkg/temporal"
    "go.uber.org/zap"
)

type Manager struct {
    temporalClient *temporal.Client
    storage        *Storage
    logger         *zap.Logger
}

func NewManager(temporalClient *temporal.Client, storage *Storage, logger *zap.Logger) *Manager {
    return &Manager{
        temporalClient: temporalClient,
        storage:        storage,
        logger:         logger,
    }
}

// Create 创建新的 Webhook Trigger
func (m *Manager) Create(ctx context.Context, req *CreateTriggerRequest) (*Trigger, error) {
    // 1. 验证 Workflow 是否存在
    workflow, err := m.workflowStore.GetByName(req.WorkflowName)
    if err != nil {
        return nil, fmt.Errorf("workflow '%s' not found: %w", req.WorkflowName, err)
    }
    
    // 2. 验证过滤规则格式
    if err := validateFilters(req.Filters); err != nil {
        return nil, fmt.Errorf("invalid filters: %w", err)
    }
    
    // 3. 生成 Webhook Secret（如果未提供）
    secret := req.Secret
    if secret == "" {
        secret = generateSecret()
    }
    if len(secret) < 16 {
        return nil, fmt.Errorf("secret must be at least 16 characters")
    }
    
    // 4. 创建 Trigger
    trigger := &Trigger{
        ID:           req.Name,
        WorkflowName: req.WorkflowName,
        Type:         "webhook",
        WebhookURL:   fmt.Sprintf("/api/v1/webhooks/%s", req.Name),
        Secret:       secret,
        Filters:      req.Filters,
        Status:       ternary(req.Enabled, "enabled", "disabled"),
        CreatedAt:    time.Now(),
    }
    
    // 5. 存储到 SQLite
    if err := m.storage.Save(trigger); err != nil {
        return nil, fmt.Errorf("failed to save trigger: %w", err)
    }
    
    m.logger.Info("Webhook trigger created",
        zap.String("trigger_id", req.Name),
        zap.String("status", trigger.Status),
    )
    
    return trigger, nil
}

// HandleWebhook 处理 Webhook 请求
func (m *Manager) HandleWebhook(ctx context.Context, triggerID string, payload []byte, signature string, headers map[string]string) (*WebhookResponse, error) {
    // 1. 查询 Trigger
    trigger, err := m.storage.Get(triggerID)
    if err != nil {
        return nil, fmt.Errorf("trigger not found: %w", err)
    }
    
    // 2. 检查是否启用
    if trigger.Status != "enabled" {
        m.logWebhookEvent(triggerID, headers, false, "Trigger disabled")
        return &WebhookResponse{Status: "ignored", Message: "Trigger is disabled"}, nil
    }
    
    // 3. 验证签名
    if !verifySignature(payload, signature, trigger.Secret) {
        m.logWebhookEvent(triggerID, headers, false, "Signature verification failed")
        return nil, fmt.Errorf("invalid signature")
    }
    
    // 4. 解析 Payload
    event, err := parseWebhookPayload(payload, headers)
    if err != nil {
        return nil, fmt.Errorf("invalid payload: %w", err)
    }
    
    // 5. 应用过滤规则
    matched, reason := m.applyFilters(trigger.Filters, event)
    if !matched {
        m.logWebhookEvent(triggerID, headers, false, reason)
        return &WebhookResponse{Status: "ignored", Message: reason}, nil
    }
    
    // 6. 获取 Workflow 定义
    workflow, err := m.workflowStore.GetByName(trigger.WorkflowName)
    if err != nil {
        return nil, fmt.Errorf("workflow not found: %w", err)
    }
    
    // 7. 触发 Temporal Workflow（异步）
    workflowID := fmt.Sprintf("%s-%d", triggerID, time.Now().Unix())
    
    go func() {
        _, err := m.temporalClient.GetClient().ExecuteWorkflow(
            context.Background(),
            temporal.WorkflowOptions{
                ID:        workflowID,
                TaskQueue: "default",
            },
            temporal.RunWorkflowExecutor,
            workflow,
        )
        
        if err != nil {
            m.logger.Error("Failed to trigger workflow",
                zap.String("trigger_id", triggerID),
                zap.String("workflow_id", workflowID),
                zap.Error(err),
            )
        }
    }()
    
    // 8. 记录审计日志
    m.logWebhookEvent(triggerID, headers, true, "Workflow triggered")
    
    // 9. 更新统计信息
    m.storage.IncrementTriggerCount(triggerID)
    
    return &WebhookResponse{
        TriggerID:  triggerID,
        WorkflowID: workflowID,
        Status:     "triggered",
        Message:    "Workflow triggered successfully",
    }, nil
}

// 验证签名
func verifySignature(payload []byte, signature string, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expectedMAC := hex.EncodeToString(mac.Sum(nil))
    expectedSignature := "sha256=" + expectedMAC
    return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// 生成随机 Secret
func generateSecret() string {
    b := make([]byte, 32)
    rand.Read(b)
    return hex.EncodeToString(b)
}
```

### Task 2: Filter Engine 实现
- [ ] 创建 pkg/trigger/filter.go
- [ ] 实现分支匹配（支持通配符）
- [ ] 实现标签匹配
- [ ] 实现路径匹配
- [ ] 实现事件类型匹配

**Filter Engine 示例:**
```go
// pkg/trigger/filter.go
package trigger

import (
    "path/filepath"
    "strings"
)

type FilterEngine struct{}

// MatchBranch 分支过滤
func (f *FilterEngine) MatchBranch(ref string, filters *PushConfig) bool {
    branch := strings.TrimPrefix(ref, "refs/heads/")
    
    // 检查 branches-ignore
    for _, pattern := range filters.BranchesIgnore {
        if matchPattern(pattern, branch) {
            return false
        }
    }
    
    // 检查 branches
    if len(filters.Branches) == 0 {
        return true  // 未配置则全部匹配
    }
    
    for _, pattern := range filters.Branches {
        if matchPattern(pattern, branch) {
            return true
        }
    }
    
    return false
}

// MatchPath 路径过滤
func (f *FilterEngine) MatchPath(changedFiles []string, filters *PushConfig) bool {
    if len(filters.Paths) == 0 {
        return true  // 未配置则全部匹配
    }
    
    for _, file := range changedFiles {
        for _, pattern := range filters.Paths {
            if matchPathPattern(pattern, file) {
                return true
            }
        }
    }
    
    return false
}

// matchPattern 支持通配符的模式匹配
func matchPattern(pattern, value string) bool {
    matched, _ := filepath.Match(pattern, value)
    return matched
}
```

### Task 3: SQLite 存储实现
- [ ] 扩展 internal/storage/trigger_store.go
- [ ] 设计 triggers 表结构
- [ ] 设计 webhook_logs 表结构
- [ ] 实现 CRUD 操作
- [ ] 实现审计日志查询

**表结构设计:**
```sql
CREATE TABLE triggers (
    id TEXT PRIMARY KEY,
    workflow_name TEXT NOT NULL,  -- 引用的工作流名称
    type TEXT NOT NULL CHECK(type IN ('webhook')),
    webhook_url TEXT,
    secret TEXT NOT NULL,
    filters TEXT,  -- JSON 格式的过滤规则
    status TEXT NOT NULL CHECK(status IN ('enabled', 'disabled')),
    total_triggers INTEGER DEFAULT 0,
    successful_triggers INTEGER DEFAULT 0,
    failed_triggers INTEGER DEFAULT 0,
    last_triggered_at DATETIME,
    last_workflow_id TEXT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME,
    disabled_at DATETIME,
    disable_reason TEXT,
    FOREIGN KEY (workflow_name) REFERENCES workflows(name) ON DELETE CASCADE
);

CREATE INDEX idx_triggers_type ON triggers(type);
CREATE INDEX idx_triggers_status ON triggers(status);
CREATE INDEX idx_triggers_workflow_name ON triggers(workflow_name);

CREATE TABLE webhook_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    trigger_id TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    source_ip TEXT,
    user_agent TEXT,
    event_type TEXT,
    signature_valid BOOLEAN,
    filter_matched BOOLEAN,
    workflow_triggered BOOLEAN,
    workflow_id TEXT,
    reason TEXT,
    processing_time_ms INTEGER,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (trigger_id) REFERENCES triggers(id) ON DELETE CASCADE
);

CREATE INDEX idx_webhook_logs_trigger_id ON webhook_logs(trigger_id);
CREATE INDEX idx_webhook_logs_timestamp ON webhook_logs(timestamp);
```

### Task 4: REST API Handler 实现
- [ ] 创建 internal/api/trigger_handler.go
- [ ] 实现所有 API 端点
  - [ ] POST /v1/triggers
  - [ ] GET /v1/triggers
  - [ ] GET /v1/triggers/:id
  - [ ] POST /v1/triggers/:id/enable
  - [ ] POST /v1/triggers/:id/disable
  - [ ] PATCH /v1/triggers/:id
  - [ ] DELETE /v1/triggers/:id
- [ ] 创建 internal/api/webhook_handler.go
  - [ ] POST /api/v1/webhooks/:trigger_id
- [ ] 集成到 Router

**说明:** ~~GET /v1/triggers/:id/logs~~ 已移除，使用工作流 API 查询触发历史

**Router 集成:**
```go
// internal/api/router.go
func NewRouter(/* ... */, triggerHandler *TriggerHandler, webhookHandler *WebhookHandler) *mux.Router {
    r := mux.NewRouter()
    
    // ... 已有路由
    
    // Trigger Management API (Story 1.10)
    r.HandleFunc("/v1/triggers", triggerHandler.CreateTrigger).Methods("POST")
    r.HandleFunc("/v1/triggers", triggerHandler.ListTriggers).Methods("GET")
    r.HandleFunc("/v1/triggers/{id}", triggerHandler.GetTrigger).Methods("GET")
    r.HandleFunc("/v1/triggers/{id}/enable", triggerHandler.EnableTrigger).Methods("POST")
    r.HandleFunc("/v1/triggers/{id}/disable", triggerHandler.DisableTrigger).Methods("POST")
    r.HandleFunc("/v1/triggers/{id}", triggerHandler.UpdateTrigger).Methods("PATCH")
    r.HandleFunc("/v1/triggers/{id}", triggerHandler.DeleteTrigger).Methods("DELETE")
    // 注意: 移除了 /triggers/{id}/logs，使用 GET /v1/workflows?trigger_source={id} 替代

    
    // Webhook Endpoint (Public API)
    r.HandleFunc("/api/v1/webhooks/{trigger_id}", webhookHandler.HandleWebhook).Methods("POST")
    
    return r
}
```

### Task 5: 单元测试
- [ ] pkg/trigger/manager_test.go - Manager 单元测试
- [ ] pkg/trigger/filter_test.go - Filter Engine 测试
- [ ] internal/storage/trigger_store_test.go - 存储测试
- [ ] internal/api/trigger_handler_test.go - API 测试
- [ ] internal/api/webhook_handler_test.go - Webhook 处理测试

### Task 6: 集成测试
- [ ] test/integration/webhook_trigger_test.go
- [ ] 测试完整的注册→Webhook 触发→查询日志流程
- [ ] 测试签名验证
- [ ] 测试过滤规则
- [ ] 测试并发 Webhook 请求

### Task 7: 文档更新
- [ ] 更新 API 文档（OpenAPI）
- [ ] 更新用户文档（Webhook 配置指南）
- [ ] 添加 GitHub/GitLab Webhook 集成示例

## Technical Requirements

### Technology Stack
- **Temporal SDK:** go.temporal.io/sdk v1.38.0
- **存储:** SQLite 3 (Trigger 元数据 + 审计日志)
- **HTTP Router:** gorilla/mux v1.8+
- **日志:** uber-go/zap v1.26+

### Architecture Constraints

**ADR 遵循:**
- [ADR-0001: 使用 Temporal 作为工作流引擎](../adr/0001-use-temporal-workflow-engine.md)
- [ADR-0004: YAML DSL 语法设计](../adr/0004-yaml-dsl-syntax.md)

**设计原则:**
- Webhook 处理必须快速响应（< 100ms），避免触发源超时
- 使用异步模式启动 Temporal Workflow
- 签名验证使用 constant-time 比较，防御时序攻击
- 完整的审计日志，支持事后分析
- 过滤规则引擎独立，易于扩展

### Performance Requirements

**Webhook 处理性能:**
- 响应时间: < 100ms (P95)
- 签名验证: < 5ms
- 并发处理: 100+ 并发 Webhook 请求

**查询性能:**
- 列表查询: < 50ms (1000 triggers)
- 详情查询: < 20ms
- 日志查询: < 100ms (10000 logs)

### Security Requirements

- **签名验证:** 强制验证 HMAC-SHA256 签名
- **Constant-time 比较:** 防御时序攻击
- **Secret 保护:** 敏感信息仅管理员可见
- **IP 白名单:** (可选) 支持限制来源 IP
- **Rate Limiting:** 防止 Webhook 滥用（未来）

## Definition of Done

- [ ] 所有 Acceptance Criteria 验收通过
- [ ] 所有 Tasks 完成并测试通过
- [ ] 单元测试覆盖率 ≥85%
- [ ] 集成测试通过（完整流程）
- [ ] 性能测试通过（100+ 并发 Webhook）
- [ ] 安全测试通过（签名验证、时序攻击防护）
- [ ] 代码通过 golangci-lint 检查
- [ ] API 文档更新（OpenAPI）
- [ ] 用户文档更新（Webhook 集成指南）
- [ ] Code Review 通过
- [ ] GitHub/GitLab Webhook 集成验证
- [ ] 生产环境验证（至少运行 48 小时）

## References

### Architecture Documents
- [mvp-schedule-service-plan.md](./mvp-schedule-service-plan.md) - 完整实施方案
- [ADR-0001: 使用 Temporal 作为工作流引擎](../adr/0001-use-temporal-workflow-engine.md)

### PRD Requirements
- [PRD - Epic 1: 核心工作流引擎](../prd.md)
- [PRD - Webhook Trigger](../prd.md#webhook-trigger)

### Previous Stories
- [Story 1.3: YAML DSL 解析和验证](./1-3-yaml-dsl-parsing-and-validation.md) - on.push 数据结构
- [Story 1.8: Temporal SDK 集成](./1-8-temporal-sdk-integration.md) - Temporal Client
- [Story 1.9: 工作流管理 REST API](./1-9-workflow-management-api.md) - 工作流管理框架
- [Story 1.10: Schedule API 实现](./1-10-schedule-api-implementation.md) - Schedule 管理框架

### External Resources
- [GitHub Webhooks Documentation](https://docs.github.com/en/developers/webhooks-and-events/webhooks/about-webhooks)
- [GitLab Webhooks Documentation](https://docs.gitlab.com/ee/user/project/integrations/webhooks.html)
- [HMAC Signature Verification](https://www.rfc-editor.org/rfc/rfc2104)

---

**Story 创建时间:** 2026-01-27  
**预计工作量:** 2-3 天  
**优先级:** P0 (MVP 必须)  
**依赖:** Story 1.10 (Schedule API)
