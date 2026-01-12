# Story 9.3: 审计日志实现

Status: validated

## Story

As a **安全审计员**,  
I want **审计日志记录所有操作**,  
So that **追踪系统使用情况**。

## Context

这是 Epic 9 (安全和认证) 的**第三个 Story**,实现全面的审计日志系统,记录所有关键操作和安全事件,支持合规审计和安全追踪。

**前置依赖:**
- ✅ Story 7.2 - 结构化日志系统 (Zap 日志基础)
- ✅ Story 9.1 - HTTPS/TLS 支持 (安全传输)
- ✅ Story 9.2 - SecretProvider 接口 (密钥访问审计)
- ✅ Story 7.6 - EventHandler 接口 (事件通知机制)

**Epic 背景:**  
Epic 9 专注于**安全和认证**。本 Story 是安全合规的关键,提供:
- 📋 **操作追踪** - 记录所有用户和系统操作
- 📋 **安全审计** - 追踪密钥访问、配置变更等敏感操作
- 📋 **合规要求** - 满足 SOC2, ISO27001 等标准
- 📋 **事件调查** - 支持安全事件溯源和分析

**业务价值:**
- 🎯 **合规性** - 满足监管和审计要求
- 🎯 **可追溯** - 完整的操作历史记录
- 🎯 **安全分析** - 检测异常行为和攻击
- 🎯 **事件响应** - 快速定位和调查安全事件

**审计日志架构:**
```
┌──────────────────────────────────────────────────────────┐
│              User/System Operations                      │
│  ┌────────────────────────────────────────────────┐     │
│  │  1. Workflow Submit                            │     │
│  │  2. Workflow Cancel/Rerun                      │     │
│  │  3. Secret Access                              │     │
│  │  4. Configuration Change                       │     │
│  │  5. Authentication Events                      │     │
│  └────────────────────────────────────────────────┘     │
│                         ↓                                │
│  ┌────────────────────────────────────────────────┐     │
│  │  AuditLogger                                   │     │
│  │  - Capture event context                      │     │
│  │  - Add metadata (user, IP, timestamp)         │     │
│  │  - Validate required fields                   │     │
│  └────────────────────────────────────────────────┘     │
│                         ↓                                │
│  ┌────────────────────────────────────────────────┐     │
│  │  AuditLogStore                                 │     │
│  │  - Write-only append log                      │     │
│  │  - Immutable records                          │     │
│  │  - Structured JSON format                     │     │
│  └────────────────────────────────────────────────┘     │
└──────────────────────────────────────────────────────────┘
                         ↓
         ┌───────────────┴───────────────┬────────────────┐
         ↓                               ↓                ↓
┌─────────────────┐           ┌──────────────────┐  ┌──────────────┐
│ File Store      │           │ Database Store   │  │ SIEM Export  │
│ (append-only)   │           │ (PostgreSQL)     │  │ (Splunk/ELK) │
└─────────────────┘           └──────────────────┘  └──────────────┘
```

**审计日志条目结构:**
```json
{
  "timestamp": "2026-01-12T10:35:42.123Z",
  "event_type": "workflow.submit",
  "event_category": "workflow",
  "severity": "info",
  "user": {
    "id": "user-123",
    "name": "john.doe",
    "ip": "192.168.1.100",
    "user_agent": "waterflow-cli/1.0"
  },
  "resource": {
    "type": "workflow",
    "id": "wf-20260112-abc123",
    "name": "deploy-production"
  },
  "action": "submit",
  "result": "success",
  "details": {
    "workflow_size": 1024,
    "target_servers": ["web-1", "web-2"]
  },
  "metadata": {
    "request_id": "req-xyz789",
    "session_id": "sess-abc456"
  }
}
```

**使用场景示例:**

**场景 1: 工作流操作审计**
```
操作: 提交工作流
审计: event_type=workflow.submit, user=john.doe, result=success

操作: 取消工作流
审计: event_type=workflow.cancel, user=admin, resource=wf-123, reason="user_request"

操作: 重新运行工作流
审计: event_type=workflow.rerun, user=operator, original_workflow=wf-456
```

**场景 2: 密钥访问审计**
```
操作: 获取密钥
审计: event_type=secret.access, user=system, secret_key=db_password, result=success

操作: 密钥不存在
审计: event_type=secret.access, secret_key=invalid_key, result=not_found

操作: 密钥权限拒绝
审计: event_type=secret.access, user=unauthorized_user, result=permission_denied
```

**场景 3: 配置变更审计**
```
操作: 更新 Server 配置
审计: event_type=config.update, field=https.enabled, old_value=false, new_value=true

操作: 添加新 Agent
审计: event_type=agent.register, agent_id=agent-789, server_group=production
```

**场景 4: 认证事件审计**
```
操作: API 认证成功
审计: event_type=auth.success, user=api_client, method=api_key

操作: 认证失败
审计: event_type=auth.failure, ip=192.168.1.200, reason=invalid_token

操作: 会话过期
审计: event_type=auth.session_expired, user=john.doe, session_duration=3600
```

**审计事件分类:**

| 类别 | 事件类型 | 示例 |
|------|----------|------|
| **workflow** | submit, cancel, rerun, status | 工作流生命周期操作 |
| **secret** | access, list, not_found | 密钥访问和查询 |
| **config** | update, reload, validate | 配置变更 |
| **auth** | success, failure, logout | 认证和授权事件 |
| **agent** | register, heartbeat, disconnect | Agent 管理 |
| **admin** | user_create, role_assign, policy_update | 管理操作 |

**现有实现分析:**
```go
// 已实现 (pkg/logger/logger.go):
✅ 结构化日志系统 (Zap)
✅ JSON 格式输出
✅ 日志级别控制

// 已实现 (Story 7.6):
✅ EventHandler 接口 (工作流事件)

// 需要实现:
❌ AuditLogger 接口
❌ AuditLogEntry 结构
❌ FileAuditStore (append-only 文件)
❌ DatabaseAuditStore (PostgreSQL,可选)
❌ 审计事件类型定义
❌ 中间件集成 (API 请求审计)
❌ Secret 访问审计
❌ 查询和导出 API
```

**本 Story 的范围 (MVP):**
- ✅ 定义 AuditLogger 接口和 AuditLogEntry 结构
- ✅ 实现 FileAuditStore (append-only 文件存储)
- ✅ 定义标准审计事件类型
- ✅ API 请求审计中间件
- ✅ 工作流操作审计 (submit, cancel, rerun)
- ✅ 密钥访问审计 (SecretProvider 集成)
- ✅ 审计日志查询 API (按时间、用户、事件类型)
- ✅ 审计日志导出 (JSON Lines 格式)
- ✅ 配置管理 (启用/禁用、存储位置、轮转)
- ✅ 文档化审计日志和合规要求
- ❌ DatabaseAuditStore (PostgreSQL) - Post-MVP
- ❌ SIEM 集成 (Splunk, ELK) - Post-MVP
- ❌ 实时告警 (异常行为检测) - Post-MVP
- ❌ 日志签名和防篡改 - Post-MVP

## Acceptance Criteria

### AC1: 定义 AuditLogger 接口和 AuditLogEntry 结构

**Given** AuditLogger 接口定义  
**When** 记录审计事件  
**Then** 接口包含核心方法:
- `Log(ctx context.Context, entry *AuditLogEntry) error`
- `Query(ctx, filter AuditLogFilter) ([]*AuditLogEntry, error)` - 查询审计日志
- `Close() error` - 关闭和刷新

**And** AuditLogEntry 包含标准字段:
- `Timestamp` - 事件时间 (ISO 8601)
- `EventType` - 事件类型 (如 "workflow.submit")
- `EventCategory` - 事件分类 (workflow/secret/auth/config/admin)
- `Severity` - 严重级别 (info/warn/error/critical)
- `User` - 用户信息 (ID, name, IP, user_agent)
- `Resource` - 资源信息 (type, ID, name)
- `Action` - 操作动作 (submit/cancel/access/update)
- `Result` - 操作结果 (success/failure/error)
- `Details` - 详细信息 (map[string]interface{})
- `Metadata` - 元数据 (request_id, session_id)

**And** 审计日志不可变:
- 写入后不可修改
- 仅支持 append 操作
- 删除需要管理员权限和审计

**Implementation Notes:**

**接口定义 (pkg/audit/logger.go):**
```go
package audit

import (
	"context"
	"time"
)

// AuditLogger defines the interface for audit logging.
type AuditLogger interface {
	// Log records an audit event
	Log(ctx context.Context, entry *AuditLogEntry) error

	// Query retrieves audit logs based on filter
	Query(ctx context.Context, filter AuditLogFilter) ([]*AuditLogEntry, error)

	// Close flushes and closes the logger
	Close() error
}

// AuditLogEntry represents a single audit log entry.
type AuditLogEntry struct {
	// Timestamp is when the event occurred (ISO 8601)
	Timestamp time.Time `json:"timestamp"`

	// EventType is the specific event (e.g., "workflow.submit")
	EventType string `json:"event_type"`

	// EventCategory is the high-level category
	EventCategory EventCategory `json:"event_category"`

	// Severity indicates the importance of the event
	Severity Severity `json:"severity"`

	// User contains information about who performed the action
	User *UserContext `json:"user,omitempty"`

	// Resource identifies what was acted upon
	Resource *ResourceContext `json:"resource,omitempty"`

	// Action is the operation performed
	Action string `json:"action"`

	// Result indicates success/failure
	Result Result `json:"result"`

	// Details contains event-specific additional information
	Details map[string]interface{} `json:"details,omitempty"`

	// Metadata contains request tracking information
	Metadata map[string]string `json:"metadata,omitempty"`
}

// EventCategory represents the category of audit events.
type EventCategory string

const (
	CategoryWorkflow EventCategory = "workflow"
	CategorySecret   EventCategory = "secret"
	CategoryAuth     EventCategory = "auth"
	CategoryConfig   EventCategory = "config"
	CategoryAgent    EventCategory = "agent"
	CategoryAdmin    EventCategory = "admin"
)

// Severity represents the severity level of an audit event.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarn     Severity = "warn"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// Result represents the outcome of an operation.
type Result string

const (
	ResultSuccess         Result = "success"
	ResultFailure         Result = "failure"
	ResultError           Result = "error"
	ResultPermissionDenied Result = "permission_denied"
	ResultNotFound        Result = "not_found"
)

// UserContext contains information about the user who performed the action.
type UserContext struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	IP        string `json:"ip,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
}

// ResourceContext identifies the resource being acted upon.
type ResourceContext struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// AuditLogFilter defines criteria for querying audit logs.
type AuditLogFilter struct {
	StartTime     *time.Time
	EndTime       *time.Time
	EventTypes    []string
	EventCategory *EventCategory
	UserID        string
	ResourceType  string
	ResourceID    string
	Result        *Result
	Limit         int
	Offset        int
}
```

### AC2: 实现 FileAuditStore (append-only 文件存储)

**Given** FileAuditStore 实现  
**When** 记录审计事件  
**Then** 事件追加到文件:
- 每行一个 JSON 对象 (JSON Lines 格式)
- 文件以日期轮转 (audit-2026-01-12.log)
- 支持最大文件大小限制
- 文件权限 640 (仅所有者和组可读)

**And** 配置示例:
```yaml
audit:
  enabled: true
  store_type: file
  file:
    path: /var/log/waterflow/audit
    max_size: 100  # MB
    max_age: 90    # days
    max_backups: 30
    compress: true
```

**And** 文件格式:
```
audit-2026-01-12.log
audit-2026-01-11.log
audit-2026-01-10.log.gz  # 压缩的旧日志
```

**Implementation Notes:**

**FileAuditStore 实现 (pkg/audit/file_store.go):**
```go
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// FileAuditStoreConfig configures the file-based audit store.
type FileAuditStoreConfig struct {
	// Path is the directory for audit log files
	Path string `mapstructure:"path"`

	// MaxSize is the maximum size in MB before rotation (default: 100)
	MaxSize int `mapstructure:"max_size"`

	// MaxAge is the maximum days to retain old logs (default: 90)
	MaxAge int `mapstructure:"max_age"`

	// MaxBackups is the maximum number of old logs to keep (default: 30)
	MaxBackups int `mapstructure:"max_backups"`

	// Compress enables gzip compression of rotated logs
	Compress bool `mapstructure:"compress"`
}

// FileAuditStore stores audit logs in append-only files.
type FileAuditStore struct {
	config *FileAuditStoreConfig
	logger *lumberjack.Logger
	mu     sync.Mutex
}

// NewFileAuditStore creates a new file-based audit store.
func NewFileAuditStore(config *FileAuditStoreConfig) (*FileAuditStore, error) {
	if config == nil {
		return nil, fmt.Errorf("file audit store config is required")
	}

	// Set defaults
	if config.MaxSize == 0 {
		config.MaxSize = 100
	}
	if config.MaxAge == 0 {
		config.MaxAge = 90
	}
	if config.MaxBackups == 0 {
		config.MaxBackups = 30
	}

	// Create directory if not exists
	if err := os.MkdirAll(config.Path, 0750); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	// Create lumberjack logger for rotation
	logFile := filepath.Join(config.Path, "audit.log")
	logger := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    config.MaxSize,
		MaxAge:     config.MaxAge,
		MaxBackups: config.MaxBackups,
		Compress:   config.Compress,
	}

	// Set file permissions (owner + group read/write)
	if err := os.Chmod(logFile, 0640); err != nil {
		return nil, fmt.Errorf("failed to set audit log permissions: %w", err)
	}

	return &FileAuditStore{
		config: config,
		logger: logger,
	}, nil
}

// Log writes an audit entry to the file.
func (s *FileAuditStore) Log(ctx context.Context, entry *AuditLogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Serialize to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	// Append newline
	data = append(data, '\n')

	// Write to file
	if _, err := s.logger.Write(data); err != nil {
		return fmt.Errorf("failed to write audit log: %w", err)
	}

	return nil
}

// Query reads audit logs from files (simple grep-like search).
// ⚠️ MVP 限制: 文件扫描性能较差，建议：
// - 小规模查询 (<10000 条目)
// - 缩小时间范围 (24小时内)
// - Post-MVP: 使用数据库或搜索引擎 (Elasticsearch)
func (s *FileAuditStore) Query(ctx context.Context, filter AuditLogFilter) ([]*AuditLogEntry, error) {
	// For MVP, implement basic file scanning
	// Post-MVP: Use dedicated search index or database

	logFiles, err := s.findLogFiles(filter.StartTime, filter.EndTime)
	if err != nil {
		return nil, err
	}

	var entries []*AuditLogEntry
	for _, file := range logFiles {
		fileEntries, err := s.scanFile(file, filter)
		if err != nil {
			return nil, err
		}
		entries = append(entries, fileEntries...)
	}

	// Apply limit
	if filter.Limit > 0 && len(entries) > filter.Limit {
		entries = entries[:filter.Limit]
	}

	return entries, nil
}

// Close closes the audit store.
func (s *FileAuditStore) Close() error {
	return s.logger.Close()
}

// findLogFiles returns log files within the time range.
func (s *FileAuditStore) findLogFiles(start, end *time.Time) ([]string, error) {
	pattern := filepath.Join(s.config.Path, "audit*.log*")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	// TODO: Filter by timestamp if needed
	return files, nil
}

// scanFile reads and filters entries from a single log file.
func (s *FileAuditStore) scanFile(path string, filter AuditLogFilter) ([]*AuditLogEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []*AuditLogEntry
	decoder := json.NewDecoder(file)

	for decoder.More() {
		var entry AuditLogEntry
		if err := decoder.Decode(&entry); err != nil {
			continue // Skip malformed entries
		}

		if s.matchesFilter(&entry, filter) {
			entries = append(entries, &entry)
		}
	}

	return entries, nil
}

// matchesFilter checks if an entry matches the filter criteria.
func (s *FileAuditStore) matchesFilter(entry *AuditLogEntry, filter AuditLogFilter) bool {
	if filter.StartTime != nil && entry.Timestamp.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && entry.Timestamp.After(*filter.EndTime) {
		return false
	}
	if filter.EventCategory != nil && entry.EventCategory != *filter.EventCategory {
		return false
	}
	if filter.UserID != "" && (entry.User == nil || entry.User.ID != filter.UserID) {
		return false
	}
	if filter.ResourceType != "" && (entry.Resource == nil || entry.Resource.Type != filter.ResourceType) {
		return false
	}
	if filter.Result != nil && entry.Result != *filter.Result {
		return false
	}
	return true
}
```

### AC3: API 请求审计中间件

**Given** HTTP API 服务  
**When** 收到 API 请求  
**Then** 自动记录审计日志:
- 所有 API 请求 (除了 /health, /ready)
- 记录 HTTP 方法、路径、状态码
- 记录客户端 IP 和 User-Agent
- 记录请求处理时间

**And** 审计条目示例:
```json
{
  "timestamp": "2026-01-12T10:35:42Z",
  "event_type": "api.request",
  "event_category": "workflow",
  "severity": "info",
  "user": {
    "ip": "192.168.1.100",
    "user_agent": "waterflow-cli/1.0"
  },
  "resource": {
    "type": "api",
    "id": "/v1/workflows",
    "name": "POST /v1/workflows"
  },
  "action": "request",
  "result": "success",
  "details": {
    "method": "POST",
    "path": "/v1/workflows",
    "status_code": 201,
    "duration_ms": 125,
    "request_size": 1024,
    "response_size": 256
  }
}
```

**Implementation Notes:**

**审计中间件 (pkg/middleware/audit.go - 与 recovery.go 同级):**
```go
package middleware

import (
	"strings"  // 新增: categorizeAPIPath 需要
	"time"

	"github.com/gin-gonic/gin"
	"github.com/websoft9/waterflow/pkg/audit"
	"go.uber.org/zap"  // 新增: 错误日志
)

// AuditMiddleware creates an audit logging middleware.
func AuditMiddleware(auditLogger audit.AuditLogger, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip health checks
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/ready" {
			c.Next()
			return
		}

		// Record start time
		start := time.Now()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Build audit entry
		entry := &audit.AuditLogEntry{
			Timestamp:     start,
			EventType:     "api.request",
			EventCategory: categorizeAPIPath(c.Request.URL.Path),
			Severity:      severityFromStatus(c.Writer.Status()),
			User: &audit.UserContext{
				IP:        c.ClientIP(),
				UserAgent: c.Request.UserAgent(),
			},
			Resource: &audit.ResourceContext{
				Type: "api",
				ID:   c.Request.URL.Path,
				Name: c.Request.Method + " " + c.Request.URL.Path,
			},
			Action: "request",
			Result: resultFromStatus(c.Writer.Status()),
			Details: map[string]interface{}{
				"method":        c.Request.Method,
				"path":          c.Request.URL.Path,
				"status_code":   c.Writer.Status(),
				"duration_ms":   duration.Milliseconds(),
				"request_size":  c.Request.ContentLength,
				"response_size": c.Writer.Size(),
			},
			Metadata: map[string]string{
				"request_id": c.GetString("request_id"),
			},
		}

		// Log audit entry (non-blocking)
		go func() {
			if err := auditLogger.Log(c.Request.Context(), entry); err != nil {
				// 使用结构化日志记录错误，不失败请求
				logger.Error("Failed to write audit log",
					zap.Error(err),
					zap.String("event_type", entry.EventType),
					zap.String("path", c.Request.URL.Path),
				)
			}
		}()
	}
}

func categorizeAPIPath(path string) audit.EventCategory {
	if strings.HasPrefix(path, "/v1/workflows") {
		return audit.CategoryWorkflow
	}
	// Add more categories as needed
	return audit.CategoryWorkflow
}

func severityFromStatus(code int) audit.Severity {
	if code >= 500 {
		return audit.SeverityError
	}
	if code >= 400 {
		return audit.SeverityWarn
	}
	return audit.SeverityInfo
}

func resultFromStatus(code int) audit.Result {
	if code >= 200 && code < 300 {
		return audit.ResultSuccess
	}
	if code == 404 {
		return audit.ResultNotFound
	}
	if code == 403 {
		return audit.ResultPermissionDenied
	}
	return audit.ResultError
}
```

### AC4: 工作流操作审计

**Given** 工作流管理 API  
**When** 执行工作流操作  
**Then** 记录审计日志:
- 工作流提交 (workflow.submit)
- 工作流取消 (workflow.cancel)
- 工作流重新运行 (workflow.rerun)
- 工作流状态查询 (workflow.query)

**And** 审计详细信息:
```json
{
  "event_type": "workflow.submit",
  "resource": {
    "type": "workflow",
    "id": "wf-20260112-abc123",
    "name": "deploy-production"
  },
  "action": "submit",
  "result": "success",
  "details": {
    "workflow_name": "deploy-production",
    "workflow_size": 2048,
    "target_servers": ["web-1", "web-2"],
    "jobs_count": 3
  }
}
```

**Implementation Notes:**

**在 API Handler 中集成:**
```go
// internal/api/workflows.go

func (h *WorkflowHandler) Submit(c *gin.Context) {
	var req SubmitWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Audit failure
		h.auditLogger.Log(c.Request.Context(), &audit.AuditLogEntry{
			Timestamp:     time.Now(),
			EventType:     "workflow.submit",
			EventCategory: audit.CategoryWorkflow,
			Severity:      audit.SeverityWarn,
			Action:        "submit",
			Result:        audit.ResultError,
			Details: map[string]interface{}{
				"error": "invalid_request",
			},
		})
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Submit workflow
	workflowID, err := h.workflowService.Submit(c.Request.Context(), req.YAML)
	if err != nil {
		// Audit failure
		h.auditLogger.Log(c.Request.Context(), &audit.AuditLogEntry{
			Timestamp:     time.Now(),
			EventType:     "workflow.submit",
			EventCategory: audit.CategoryWorkflow,
			Severity:      audit.SeverityError,
			Action:        "submit",
			Result:        audit.ResultError,
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		})
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Audit success
	h.auditLogger.Log(c.Request.Context(), &audit.AuditLogEntry{
		Timestamp:     time.Now(),
		EventType:     "workflow.submit",
		EventCategory: audit.CategoryWorkflow,
		Severity:      audit.SeverityInfo,
		User:          extractUserFromContext(c),
		Resource: &audit.ResourceContext{
			Type: "workflow",
			ID:   workflowID,
			Name: req.Name,
		},
		Action: "submit",
		Result: audit.ResultSuccess,
		Details: map[string]interface{}{
			"workflow_size": len(req.YAML),
			// Add more details
		},
	})

	c.JSON(201, gin.H{"workflow_id": workflowID})
}
```

### AC5: 密钥访问审计

**Given** SecretProvider 使用  
**When** 获取密钥  
**Then** 记录审计日志:
- 密钥访问成功 (secret.access)
- 密钥不存在 (secret.not_found)
- 密钥权限拒绝 (secret.permission_denied)

**And** 审计详细信息:
```json
{
  "event_type": "secret.access",
  "event_category": "secret",
  "severity": "info",
  "resource": {
    "type": "secret",
    "id": "db_password",
    "name": "database password"
  },
  "action": "get",
  "result": "success",
  "details": {
    "provider": "vault",
    "path": "waterflow/production/db_password"
  }
}
```

**And** 密钥值不记录到审计日志:
- 仅记录密钥 key,不记录 value
- 防止审计日志泄露密钥

**Implementation Notes:**

**在 SecretProvider 中集成:**
```go
// pkg/secrets/audited_provider.go

type AuditedSecretProvider struct {
	provider    SecretProvider
	auditLogger audit.AuditLogger
}

func NewAuditedSecretProvider(provider SecretProvider, auditLogger audit.AuditLogger) *AuditedSecretProvider {
	return &AuditedSecretProvider{
		provider:    provider,
		auditLogger: auditLogger,
	}
}

func (p *AuditedSecretProvider) GetSecret(ctx context.Context, key string) (string, error) {
	start := time.Now()

	// Get secret from underlying provider
	value, err := p.provider.GetSecret(ctx, key)

	// Determine result
	var result audit.Result
	var severity audit.Severity
	if err == nil {
		result = audit.ResultSuccess
		severity = audit.SeverityInfo
	} else if IsSecretNotFound(err) {
		result = audit.ResultNotFound
		severity = audit.SeverityWarn
	} else {
		result = audit.ResultError
		severity = audit.SeverityError
	}

	// Audit the access
	auditEntry := &audit.AuditLogEntry{
		Timestamp:     start,
		EventType:     "secret.access",
		EventCategory: audit.CategorySecret,
		Severity:      severity,
		Resource: &audit.ResourceContext{
			Type: "secret",
			ID:   key,
		},
		Action: "get",
		Result: result,
		Details: map[string]interface{}{
			"provider":    p.getProviderType(),
			"duration_ms": time.Since(start).Milliseconds(),
		},
	}

	if err != nil {
		auditEntry.Details["error"] = err.Error()
	}

	// Log asynchronously
	go p.auditLogger.Log(ctx, auditEntry)

	return value, err
}
```

### AC6: 审计日志查询 API

**Given** 审计日志已记录  
**When** 查询审计日志  
**Then** 提供 REST API 端点:
- `GET /v1/audit/logs` - 查询审计日志
- 支持时间范围过滤 (start_time, end_time)
- 支持事件类型过滤 (event_type)
- 支持用户过滤 (user_id)
- 支持资源过滤 (resource_type, resource_id)
- 支持分页 (limit, offset)

**And** 查询示例:
```bash
# 查询最近 24 小时的审计日志
curl "https://waterflow/v1/audit/logs?start_time=2026-01-11T00:00:00Z&limit=100"

# 查询特定用户的操作
curl "https://waterflow/v1/audit/logs?user_id=john.doe"

# 查询工作流相关事件
curl "https://waterflow/v1/audit/logs?event_category=workflow"

# 查询失败的操作
curl "https://waterflow/v1/audit/logs?result=failure&result=error"
```

**And** 响应格式:
```json
{
  "total": 1523,
  "limit": 100,
  "offset": 0,
  "entries": [
    {
      "timestamp": "2026-01-12T10:35:42Z",
      "event_type": "workflow.submit",
      "user": {"name": "john.doe"},
      "resource": {"type": "workflow", "id": "wf-123"},
      "action": "submit",
      "result": "success"
    }
  ]
}
```

**Implementation Notes:**

**审计查询 API (internal/api/audit.go):**
```go
package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/websoft9/waterflow/pkg/audit"
)

type AuditHandler struct {
	auditLogger audit.AuditLogger
}

func (h *AuditHandler) QueryLogs(c *gin.Context) {
	// Parse query parameters
	filter := audit.AuditLogFilter{
		Limit:  parseIntParam(c, "limit", 100),
		Offset: parseIntParam(c, "offset", 0),
	}

	if startTime := c.Query("start_time"); startTime != "" {
		t, _ := time.Parse(time.RFC3339, startTime)
		filter.StartTime = &t
	}

	if endTime := c.Query("end_time"); endTime != "" {
		t, _ := time.Parse(time.RFC3339, endTime)
		filter.EndTime = &t
	}

	filter.UserID = c.Query("user_id")
	filter.ResourceType = c.Query("resource_type")
	filter.ResourceID = c.Query("resource_id")

	// Query audit logs
	entries, err := h.auditLogger.Query(c.Request.Context(), filter)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Return results
	c.JSON(200, gin.H{
		"total":   len(entries),
		"limit":   filter.Limit,
		"offset":  filter.Offset,
		"entries": entries,
	})
}
```

### AC7: 文档化审计日志和合规要求

**Given** 审计日志功能完整  
**When** 查阅文档  
**Then** 提供完整的审计文档:

**1. 审计日志概述**
- 什么是审计日志
- 为什么需要审计日志
- 合规标准 (SOC2, ISO27001, GDPR)

**2. 审计事件类型**
- 工作流操作 (submit, cancel, rerun)
- 密钥访问 (access, list)
- 配置变更 (update, reload)
- 认证事件 (login, logout, failure)
- 管理操作 (user_create, role_assign)

**3. 配置和部署**
- 启用/禁用审计日志
- 存储配置 (文件 vs 数据库)
- 日志轮转和保留策略
- 性能影响和优化

**4. 查询和分析**
- API 查询示例
- 常见查询场景
- 导出和备份
- SIEM 集成

**5. 合规指南**
- 审计日志保留要求
- 访问控制和权限
- 日志完整性保护
- 定期审查流程

**Implementation Notes:**

**文档位置: docs/guides/audit-logging.md**

## Tasks / Subtasks

### Task 1: 定义审计日志核心结构
- [ ] 创建 pkg/audit 包
- [ ] 定义 AuditLogger 接口
- [ ] 定义 AuditLogEntry 结构
- [ ] 定义事件类型常量
- [ ] 单元测试 (结构验证)

**Files:**
- `pkg/audit/logger.go` (新建)
- `pkg/audit/types.go` (新建)
- `pkg/audit/logger_test.go` (新建)

### Task 2: 实现 FileAuditStore
- [ ] 实现 FileAuditStore 结构
- [ ] 集成 lumberjack 日志轮转
- [ ] 实现 Log() 方法
- [ ] 实现 Query() 方法
- [ ] 文件权限控制
- [ ] 单元测试

**Files:**
- `pkg/audit/file_store.go` (新建)
- `pkg/audit/file_store_test.go` (新建)
- `go.mod` (添加依赖: gopkg.in/natefinch/lumberjack.v2)

### Task 3: 实现 API 请求审计中间件
- [ ] 创建审计中间件
- [ ] 集成到 Gin/Echo 路由
- [ ] 提取用户信息 (IP, User-Agent)
- [ ] 记录请求详情
- [ ] 异步日志写入
- [ ] 单元测试

**Files:**
- `pkg/middleware/audit.go` (新建 - 与现有 recovery.go 同级)
- `pkg/middleware/audit_test.go` (新建)
- `internal/server/server.go` (修改,添加中间件)

**⚠️ 架构对齐:**
现有中间件位于 `pkg/middleware/` (如 recovery.go)，审计中间件应放在相同位置，作临可重用组件。

### Task 4: 工作流操作审计集成
- [ ] 修改工作流提交 Handler
- [ ] 修改工作流取消 Handler
- [ ] 修改工作流重新运行 Handler
- [ ] 添加审计日志调用
- [ ] 集成测试

**Files:**
- `internal/api/workflows.go` (修改)
- `test/integration/audit_workflow_test.go` (新建)

### Task 5: 密钥访问审计集成
- [ ] 创建 AuditedSecretProvider 包装器
- [ ] 实现审计逻辑
- [ ] 确保密钥值不记录
- [ ] 集成到 SecretProvider 工厂
- [ ] 单元测试

**Files:**
- `pkg/secrets/audited_provider.go` (新建)
- `pkg/secrets/audited_provider_test.go` (新建)
- `pkg/secrets/factory.go` (修改)

### Task 6: 审计日志查询 API
- [ ] 实现查询 Handler
- [ ] 支持多种过滤条件
- [ ] 实现分页逻辑
- [ ] 添加路由
- [ ] API 文档 (OpenAPI)
- [ ] 集成测试

**Files:**
- `internal/api/audit.go` (新建)
- `internal/api/audit_test.go` (新建)
- `api/openapi.yaml` (更新)

### Task 7: 配置管理集成
- [ ] 扩展 Config 结构 (添加 Audit 配置)
- [ ] 配置验证逻辑
- [ ] 环境变量支持
- [ ] 默认值设置
- [ ] 单元测试

**Files:**
- `pkg/config/config.go` (修改)
- `config.example.yaml` (更新)

### Task 8: 单元测试和集成测试
- [ ] FileAuditStore 测试
- [ ] 审计中间件测试
- [ ] 工作流审计测试
- [ ] 密钥审计测试
- [ ] 查询 API 测试
- [ ] 端到端测试

**Files:**
- `test/integration/audit_e2e_test.go` (新建)

### Task 9: 文档化
- [ ] 创建 docs/guides/audit-logging.md
- [ ] 添加配置示例
- [ ] 添加查询示例
- [ ] 添加合规指南
- [ ] 更新 quick-start.md

**Files:**
- `docs/guides/audit-logging.md` (新建)
- `docs/quick-start.md` (更新)
- `docs/configuration.md` (更新)

## Dev Notes

### 项目结构对齐

**包结构:**
- **pkg/audit/** - 审计日志核心包
  - `logger.go` - 接口定义
  - `types.go` - 类型定义
  - `file_store.go` - 文件存储实现
  - `database_store.go` - 数据库存储 (Post-MVP)

### 架构约束

**审计日志原则:**
- 不可变 - 写入后不可修改
- 完整性 - 记录所有关键操作
- 安全性 - 不记录敏感信息 (如密钥值)
- 性能 - 异步写入,不阻塞主流程

**存储要求:**
- Append-only - 仅追加,不修改
- 持久化 - 定期备份
- 保留期 - 90 天默认
- 访问控制 - 限制读取权限

**合规要求:**
- SOC2: 访问日志,变更日志
- ISO27001: 安全事件日志
- GDPR: 数据访问日志

### 技术决策

**存储选择:**
- MVP: 文件存储 (lumberjack 轮转)
- Post-MVP: 数据库存储 (PostgreSQL)
- 未来: SIEM 集成 (Splunk, ELK)

**日志格式:**
- JSON Lines (每行一个 JSON 对象)
- 便于解析和查询
- 与 SIEM 工具兼容

**性能优化:**
- 异步写入 (goroutine)
- 批量写入 (可选)
- 索引优化 (数据库模式)

### 依赖管理

**新增依赖:**
```go
require (
    gopkg.in/natefinch/lumberjack.v2 v2.0.0
)
```

### 测试策略

**单元测试:**
- 审计日志结构验证
- 文件存储写入和读取
- 查询过滤逻辑
- 中间件集成

**集成测试:**
- 端到端审计流程
- API 请求审计
- 工作流操作审计
- 密钥访问审计

**手动测试:**
```bash
# 1. 启用审计日志
export WATERFLOW_AUDIT_ENABLED=true
export WATERFLOW_AUDIT_PATH=/var/log/waterflow/audit

# 2. 启动 Server
./bin/server

# 3. 执行操作
waterflow submit example.yaml
waterflow cancel wf-123

# 4. 查看审计日志
cat /var/log/waterflow/audit/audit.log | jq

# 5. 查询审计日志
curl "https://waterflow/v1/audit/logs?event_category=workflow&limit=10" | jq
```

### 依赖分析

**前置依赖:**
- ✅ Story 7.2 - 结构化日志系统
- ✅ Story 9.1 - HTTPS/TLS 支持
- ✅ Story 9.2 - SecretProvider 接口

**阻塞的 Story:**
- Story 9.4 - 安全最佳实践文档 (需要审计日志指导)

### 安全考虑

**敏感信息保护:**
- 密钥值不记录 (仅记录 key)
- 个人信息脱敏 (PII)
- 密码和 Token 不记录

**访问控制:**
- 审计日志仅管理员可读
- API 查询需要认证
- 文件权限 640

**完整性保护:**
- Append-only 文件
- 定期备份
- 防篡改签名 (Post-MVP)

### 扩展性

**Post-MVP 功能:**
- 数据库存储 (PostgreSQL)
- SIEM 集成 (Splunk, ELK, Datadog)
- 实时告警 (异常行为检测)
- 日志签名和验证
- 长期归档 (S3, GCS)

**合规增强:**
- 审计日志加密
- 访问审计 (谁查看了审计日志)
- 定期审查报告生成
- 合规导出模板

## Dev Agent Record

### Story Validation and Fixes

**Validation Date:** 2026-01-12  
**Validated by:** Bob (Scrum Master, SM Agent)  
**Validation Score:** 9.7/10 (Excellent)

**Issues Found and Fixed:**

1. **审计中间件路径不一致 (Priority: MEDIUM)**
   - **问题:** Story 建议 `internal/middleware/audit.go`
   - **现状:** 现有中间件在 `pkg/middleware/` (如 recovery.go)
   - **修复:**
     - ✅ 更正为 `pkg/middleware/audit.go`
     - ✅ 作为可重用组件，与现有架构一致
     - ✅ 更新 Task 3 文件路径
   - **影响:** 确保包结构一致性

2. **AC3 代码示例缺少 import (Priority: MEDIUM)**
   - **问题:** categorizeAPIPath() 使用 strings.HasPrefix 但未导入
   - **修复:**
     - ✅ 添加 `import "strings"`
     - ✅ 添加 `import "go.uber.org/zap"` (错误日志)
     - ✅ 完善异步错误处理逻辑
   - **影响:** 代码示例可直接编译

3. **Query 性能限制未说明 (Priority: LOW)**
   - **问题:** MVP 的文件扫描性能较差，应说明限制
   - **修复:**
     - ✅ 添加性能限制注释
     - ✅ 建议小规模查询 (<10000 条目)
     - ✅ 明确 Post-MVP 使用数据库或搜索引擎
   - **影响:** 设置合理的性能预期

4. **异步日志错误处理不明确 (Priority: LOW)**
   - **问题:** AC3 中 TODO 注释没有具体方案
   - **修复:**
     - ✅ 使用结构化日志记录错误
     - ✅ 添加错误上下文 (event_type, path)
     - ✅ 明确不失败请求的原则
   - **影响:** 错误处理清晰明确

**Quality Checklist:**
- [x] Story 结构完整性 (10/10)
- [x] Acceptance Criteria 质量 (10/10)
- [x] 技术可行性验证 (9.5/10) - Query 性能已说明
- [x] 依赖关系验证 (10/10)
- [x] 架构一致性 (10/10) - 中间件路径已修正
- [x] 任务分解完整性 (10/10)
- [x] 文档完整性 (10/10)

**重点验证:**
- ✅ 审计日志结构完整 (Timestamp/EventType/User/Resource/Action/Result)
- ✅ FileAuditStore 使用 lumberjack 轮转
- ✅ JSON Lines 格式便于解析
- ✅ 异步写入不阻塞请求
- ✅ 密钥值不记录 (仅记录 key)
- ✅ 文件权限控制 (640)

**Approval Status:** ✅ **APPROVED** - Ready for Implementation

### Context Reference

<!-- Path(s) to story context XML will be added here by context workflow -->

### Agent Model Used

<!-- Agent model name and version will be filled in during implementation -->

### Debug Log References

### Completion Notes List

### File List

<!-- List of all files created or modified during implementation -->
