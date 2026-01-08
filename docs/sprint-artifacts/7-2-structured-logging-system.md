# Story 7.2: 结构化日志系统

Status: ready-for-dev

## Story

As a **系统管理员**,  
I want **结构化的日志输出**,  
So that **日志易于解析和查询**。

## Context

这是 Epic 7 (生产级可靠性) 的**第二个 Story**,实现**结构化日志系统**。该 Story 在 Story 7.1 (类型化错误处理) 的基础上,增强系统的可观测性,确保所有日志都是结构化的、可查询的、且包含完整的上下文信息。

**前置依赖:**
- ✅ Story 7.1 - 类型化错误处理 (错误日志集成)
- ✅ Epic 1-6 - 现有代码已使用 zap.Logger (需要规范化)
- ✅ pkg/logger 包已存在 (基础实现,需要增强)

**Epic 背景:**  
Epic 7 专注于**生产级可靠性**。本 Story 建立在 Story 7.1 的错误处理之上,提供:
- **统一的日志接口** - 所有模块使用一致的日志 API
- **结构化日志输出** - JSON 格式,包含完整上下文
- **日志级别控制** - 支持 debug/info/warn/error 级别配置
- **敏感信息脱敏** - 自动过滤密码、Token 等敏感数据
- **高性能日志** - 性能关键路径使用 Zap 零分配日志

**业务价值:**
- 🎯 **问题定位效率** - 结构化日志支持快速查询和过滤
- 🎯 **系统可观测性** - 完整的上下文信息 (workflow_id, job_id, step_name)
- 🎯 **安全合规** - 敏感信息自动脱敏
- 🎯 **集成便利** - JSON 格式易于集成 ELK/Loki/CloudWatch
- 🎯 **性能优化** - Zap 高性能日志减少日志开销

**日志架构设计:**
```
┌────────────────────────────────────────────────────────────┐
│                 Application Code                           │
│  (Server, Agent, Workflows, Nodes)                         │
└────────────────┬───────────────────────────────────────────┘
                 │ 使用 logger.Log 或注入的 *zap.Logger
                 ↓
┌────────────────────────────────────────────────────────────┐
│              pkg/logger (Enhanced)                         │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │ Global Log   │  │ NewLogger()  │  │ WithContext()   │  │
│  │ (Singleton)  │  │ (Factory)    │  │ (Contextual)    │  │
│  └──────────────┘  └──────────────┘  └─────────────────┘  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Sanitization (敏感信息脱敏)                           │  │
│  │ - password, token, api_key, secret                   │  │
│  └──────────────────────────────────────────────────────┘  │
└────────────────┬───────────────────────────────────────────┘
                 │ zap.Logger
                 ↓
┌────────────────────────────────────────────────────────────┐
│              go.uber.org/zap                               │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │ JSON Encoder │  │ Console      │  │ File/Stdout     │  │
│  │ (Production) │  │ (Development)│  │ (Output)        │  │
│  └──────────────┘  └──────────────┘  └─────────────────┘  │
└────────────────────────────────────────────────────────────┘
```

**日志结构标准:**
```json
{
  "timestamp": "2026-01-07T10:30:15.123Z",
  "level": "info",
  "logger": "server",
  "msg": "Workflow executed successfully",
  "workflow_id": "wf-abc-123",
  "job_id": "build",
  "step_name": "deploy",
  "duration_ms": 1250,
  "status": "completed",
  "caller": "workflow/executor.go:142"
}
```

**关键字段说明:**
- `timestamp` - ISO 8601 格式时间戳
- `level` - 日志级别 (debug/info/warn/error)
- `logger` - 日志来源组件 (server/agent/workflow/node)
- `msg` - 人类可读的消息
- `workflow_id` - 工作流 ID (如果在工作流上下文中)
- `job_id` - Job ID (如果在 Job 上下文中)
- `step_name` - Step 名称 (如果在 Step 上下文中)
- `node_type` - 节点类型 (如果在节点执行中)
- `caller` - 调用位置 (文件:行号)
- 其他业务相关字段

**与现有代码的关系:**
- ✅ `pkg/logger/logger.go` - 已存在基础实现 (需要增强)
- ✅ Server/Agent main.go - 已使用 logger.Init
- ✅ 各模块 - 已注入 *zap.Logger (需要规范化)
- 🆕 敏感信息脱敏 - 新增功能
- 🆕 上下文日志 - WithContext 模式
- 🆕 日志规范 - 统一字段命名

**本 Story 的范围 (MVP):**
- ✅ 增强 pkg/logger 包 - 敏感信息脱敏、上下文日志
- ✅ 标准化日志字段 - 统一命名规范
- ✅ 日志级别动态配置 - 支持运行时调整
- ✅ 规范化现有日志调用 - 统一格式和字段
- ✅ 文档和最佳实践 - 日志使用指南
- ❌ LogHandler 接口 - 留待 Story 7.7
- ❌ 日志聚合和分析 - Post-MVP

**典型日志场景:**

**场景 1: 工作流执行日志**
```go
logger.Info("Workflow started",
    zap.String("workflow_id", workflowID),
    zap.String("user", username),
)

logger.Info("Step executing",
    zap.String("workflow_id", workflowID),
    zap.String("job_id", jobID),
    zap.String("step_name", stepName),
    zap.String("node_type", nodeType),
)

logger.Info("Workflow completed",
    zap.String("workflow_id", workflowID),
    zap.Duration("duration", duration),
    zap.String("status", "success"),
)
```

**场景 2: 错误日志 (集成 Story 7.1)**
```go
logger.Error("Step execution failed",
    zap.String("workflow_id", workflowID),
    zap.String("step_name", stepName),
    zap.String("error_type", err.ErrorType()),
    zap.Bool("retryable", err.IsRetryable()),
    zap.Error(err),
)
```

**场景 3: 敏感信息脱敏**
```go
// 自动脱敏
logger.Info("User authenticated",
    zap.String("username", "admin"),
    zap.String("password", "secret123"),  // 输出: ***REDACTED***
    zap.String("api_key", "sk-xxx"),      // 输出: ***REDACTED***
)
```

## Acceptance Criteria

### AC1: 增强 pkg/logger 包 - 敏感信息脱敏

**Given** 日志可能包含敏感信息 (密码、Token、API Key)  
**When** 输出日志时  
**Then** 自动检测并脱敏敏感字段  
**And** 敏感字段名称 (不区分大小写):
- `password`
- `token`
- `api_key`
- `secret`
- `private_key`
- `access_token`
- `refresh_token`

**And** 脱敏后输出: `***REDACTED***`  
**And** 支持自定义敏感字段列表

**And** 自定义敏感字段配置方式:
1. **配置文件** - `config.yaml` 中 `log.sensitive_fields: ["custom_secret"]`
2. **环境变量** - `WATERFLOW_LOG_SENSITIVE_FIELDS=custom_secret,api_token`
3. **API 配置** - 初始化时传入: `InitWithConfig(LogConfig{SensitiveFields: []string{...}})`

**Implementation Notes:**
```go
// pkg/logger/sanitizer.go
package logger

import (
    "strings"
    "go.uber.org/zap/zapcore"
)

var (
    // DefaultSensitiveFields 默认敏感字段列表
    DefaultSensitiveFields = []string{
        "password",
        "token",
        "api_key",
        "secret",
        "private_key",
        "access_token",
        "refresh_token",
        "authorization",
        "cookie",
    }
)

// SanitizingEncoder 包装 zapcore.Encoder,自动脱敏敏感字段
type SanitizingEncoder struct {
    zapcore.Encoder
    sensitiveFields map[string]bool
}

// NewSanitizingEncoder 创建脱敏编码器
func NewSanitizingEncoder(encoder zapcore.Encoder, sensitiveFields []string) zapcore.Encoder {
    fieldMap := make(map[string]bool)
    for _, field := range sensitiveFields {
        fieldMap[strings.ToLower(field)] = true
    }
    
    return &SanitizingEncoder{
        Encoder:         encoder,
        sensitiveFields: fieldMap,
    }
}

// AddString 添加字符串字段,自动脱敏
func (e *SanitizingEncoder) AddString(key string, value string) {
    if e.isSensitive(key) {
        e.Encoder.AddString(key, "***REDACTED***")
    } else {
        e.Encoder.AddString(key, value)
    }
}

// AddByteString 添加字节字符串字段,自动脱敏
func (e *SanitizingEncoder) AddByteString(key string, value []byte) {
    if e.isSensitive(key) {
        e.Encoder.AddByteString(key, []byte("***REDACTED***"))
    } else {
        e.Encoder.AddByteString(key, value)
    }
}

func (e *SanitizingEncoder) isSensitive(key string) bool {
    return e.sensitiveFields[strings.ToLower(key)]
}

// Clone 复制编码器
func (e *SanitizingEncoder) Clone() zapcore.Encoder {
    return &SanitizingEncoder{
        Encoder:         e.Encoder.Clone(),
        sensitiveFields: e.sensitiveFields,
    }
}

// 注意: SanitizingEncoder 需要实现以下 zapcore.Encoder 接口方法:
// - AddArray, AddObject, AddBinary, AddBool, AddComplex64, AddComplex128
// - AddDuration, AddFloat32, AddFloat64, AddInt, AddInt8, AddInt16, AddInt32, AddInt64
// - AddReflected, AddUint, AddUint8, AddUint16, AddUint32, AddUint64, AddUintptr
// - AddTime, OpenNamespace
// 
// 这些方法直接委托给底层 Encoder,不需要脱敏处理:
// func (e *SanitizingEncoder) AddBool(key string, value bool) {
//     e.Encoder.AddBool(key, value)
// }
// ... (其他方法类似)
```

**集成到 logger.Init:**
```go
// pkg/logger/logger.go

// LogConfig 日志配置
type LogConfig struct {
    Level           string   // 日志级别
    Format          string   // 格式: json/text
    SensitiveFields []string // 自定义敏感字段 (追加到默认列表)
}

// Init 使用默认配置初始化日志
func Init(level string, format string) error {
    return InitWithConfig(LogConfig{
        Level:  level,
        Format: format,
    })
}

// InitWithConfig 使用完整配置初始化日志
func InitWithConfig(cfg LogConfig) error {
    var cfg zap.Config
    
    if format == "json" {
        cfg = zap.NewProductionConfig()
    } else {
        cfg = zap.NewDevelopmentConfig()
    }
    
    // ... (其他配置)
    
    logger, err := cfg.Build(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
        // 包装 Core,使用脱敏编码器
        encoder := NewSanitizingEncoder(
            zapcore.NewJSONEncoder(cfg.EncoderConfig),
            DefaultSensitiveFields,
        )
        return zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), cfg.Level.Level())
    }))
    
    if err != nil {
        return err
    }
    
    Log = logger
    return nil
}
```

### AC2: 上下文日志 - WithContext 模式

**Given** 日志需要携带上下文信息 (workflow_id, job_id, step_name)  
**When** 创建带上下文的 logger  
**Then** 提供 `WithContext` 方法创建子 logger  
**And** 子 logger 自动附加上下文字段  
**And** 支持链式添加多个上下文

**Implementation Notes:**
```go
// pkg/logger/context.go
package logger

import "go.uber.org/zap"

// WithWorkflowContext 创建带工作流上下文的 logger
func WithWorkflowContext(workflowID string) *zap.Logger {
    return Log.With(zap.String("workflow_id", workflowID))
}

// WithJobContext 创建带 Job 上下文的 logger
func WithJobContext(workflowID, jobID string) *zap.Logger {
    return Log.With(
        zap.String("workflow_id", workflowID),
        zap.String("job_id", jobID),
    )
}

// WithStepContext 创建带 Step 上下文的 logger
func WithStepContext(workflowID, jobID, stepName string) *zap.Logger {
    return Log.With(
        zap.String("workflow_id", workflowID),
        zap.String("job_id", jobID),
        zap.String("step_name", stepName),
    )
}

// WithNodeContext 创建带节点上下文的 logger
func WithNodeContext(workflowID, jobID, stepName, nodeType string) *zap.Logger {
    return Log.With(
        zap.String("workflow_id", workflowID),
        zap.String("job_id", jobID),
        zap.String("step_name", stepName),
        zap.String("node_type", nodeType),
    )
}

// WithFields 创建带任意字段的 logger
func WithFields(fields ...zap.Field) *zap.Logger {
    return Log.With(fields...)
}
```

**使用示例:**
```go
// Workflow 执行器
type Executor struct {
    logger *zap.Logger
}

func NewExecutor(workflowID string) *Executor {
    return &Executor{
        logger: logger.WithWorkflowContext(workflowID),
    }
}

func (e *Executor) ExecuteStep(jobID, stepName, nodeType string) error {
    stepLogger := e.logger.With(
        zap.String("job_id", jobID),
        zap.String("step_name", stepName),
        zap.String("node_type", nodeType),
    )
    
    stepLogger.Info("Step started")
    // ...执行逻辑...
    stepLogger.Info("Step completed", zap.Duration("duration", duration))
    
    return nil
}
```

### AC3: 日志级别动态配置

**Given** 系统运行中需要调整日志级别  
**When** 修改配置或环境变量  
**Then** 支持运行时调整日志级别 (不需要重启)  
**And** 提供 HTTP 端点修改日志级别  
**And** 日志级别: debug, info, warn, error  
**And** /admin 端点需要认证保护 (Bearer Token 或 API Key)  
**And** 记录日志级别修改的审计日志 (谁、何时、修改为什么级别)

**Implementation Notes:**
```go
// pkg/logger/logger.go
import "go.uber.org/zap/zapcore"

var (
    Log         *zap.Logger
    atomicLevel zap.AtomicLevel  // 支持动态调整
)

func Init(level string, format string) error {
    var cfg zap.Config
    
    if format == "json" {
        cfg = zap.NewProductionConfig()
    var zapCfg zap.Config
    
    if cfg.Format == "json" {
        zapCfg = zap.NewProductionConfig()
    } else {
        zapCfg = zap.NewDevelopmentConfig()
    }
    
    // 解析日志级别
    zapLevel, err := parseLevel(cfg.Level)
    if err != nil {
        return err
    }
    
    // 使用 AtomicLevel 支持动态调整
    atomicLevel = zap.NewAtomicLevelAt(zapLevel)
    zapCfg.Level = atomicLevel
    
    // 合并默认和自定义敏感字段
    sensitiveFields := append([]string{}, DefaultSensitiveFields...)
    sensitiveFields = append(sensitiveFields, cfg.SensitiveFields...)
    
    logger, err := zapCfg.Build(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
        encoder := NewSanitizingEncoder(
            zapcore.NewJSONEncoder(zapCfg.EncoderConfig),
            sensitiveFields,
        )
        return zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapCfg.Level.Level())
    }))
    if err != nil {
        return err
    }
    
    Log = logger
    return nil
}

// SetLevel 动态设置日志级别
func SetLevel(level string) error {
    zapLevel, err := parseLevel(level)
    if err != nil {
        return err
    }
    
    atomicLevel.SetLevel(zapLevel)
    Log.Info("Log level changed", zap.String("new_level", level))
    
    return nil
}

// GetLevel 获取当前日志级别
func GetLevel() string {
    switch atomicLevel.Level() {
    case zapcore.DebugLevel:
        return "debug"
    case zapcore.InfoLevel:
        return "info"
    case zapcore.WarnLevel:
        return "warn"
    case zapcore.ErrorLevel:
        return "error"
    default:
        return "unknown"
    }
}
```

**HTTP 端点 (在 Server 中实现):**
```go
// internal/server/handlers/admin.go
import "github.com/websoft9/waterflow/pkg/middleware"

func (h *AdminHandler) SetLogLevel(w http.ResponseWriter, r *http.Request) {
    // 权限检查已通过中间件完成 (见路由注册)
    
    var req struct {
        Level string `json:"level"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    if err := logger.SetLevel(req.Level); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // 审计日志: 记录谁修改了日志级别
    user := r.Context().Value("user").(string) // 从认证中间件获取
    logger.Log.Info("Log level changed",
        zap.String("level", req.Level),
        zap.String("changed_by", user),
        zap.String("remote_addr", r.RemoteAddr),
    )
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "level": req.Level,
        "message": "Log level updated",
    })
}

func (h *AdminHandler) GetLogLevel(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "level": logger.GetLevel(),
    })
}
```

**路由注册 (带认证中间件):**
```go
// internal/server/router.go
import "github.com/websoft9/waterflow/pkg/middleware"

// Admin 路由需要认证
adminRouter := r.PathPrefix("/admin").Subrouter()
adminRouter.Use(middleware.RequireAuth) // Bearer Token 或 API Key 认证
adminRouter.HandleFunc("/log-level", adminHandler.GetLogLevel).Methods("GET")
adminRouter.HandleFunc("/log-level", adminHandler.SetLogLevel).Methods("PUT")
```

### AC4: 标准化日志字段命名

**Given** 日志需要统一的字段命名规范  
**When** 输出日志时  
**Then** 使用以下标准字段名:

**通用字段:**
- `timestamp` - 时间戳 (自动添加)
- `level` - 日志级别 (自动添加)
- `logger` - 日志来源组件
- `msg` - 消息
- `caller` - 调用位置 (可选)

**工作流相关:**
- `workflow_id` - 工作流 ID
- `workflow_name` - 工作流名称
- `workflow_status` - 工作流状态 (running/completed/failed)
- `job_id` - Job ID
- `step_name` - Step 名称
- `node_type` - 节点类型

**性能相关:**
- `duration` - 执行时长 (使用 zap.Duration)
- `duration_ms` - 毫秒时长 (使用 zap.Int64)

**错误相关 (集成 Story 7.1):**
- `error` - 错误对象 (使用 zap.Error)
- `error_type` - 错误类型
- `retryable` - 是否可重试

**系统相关:**
- `component` - 组件名称 (server/agent/worker)
- `version` - 版本号
- `host` - 主机名
- `pid` - 进程 ID

**And** 提供常量定义:
```go
// pkg/logger/fields.go
package logger

// 标准字段名常量
const (
    // 工作流字段
    FieldWorkflowID     = "workflow_id"
    FieldWorkflowName   = "workflow_name"
    FieldWorkflowStatus = "workflow_status"
    FieldJobID          = "job_id"
    FieldStepName       = "step_name"
    FieldNodeType       = "node_type"
    
    // 性能字段
    FieldDuration   = "duration"
    FieldDurationMS = "duration_ms"
    
    // 错误字段
    FieldErrorType = "error_type"
    FieldRetryable = "retryable"
    
    // 系统字段
    FieldComponent = "component"
    FieldVersion   = "version"
    FieldHost      = "host"
    FieldPID       = "pid"
)

// 辅助函数
func WorkflowID(id string) zap.Field {
    return zap.String(FieldWorkflowID, id)
}

func JobID(id string) zap.Field {
    return zap.String(FieldJobID, id)
}

func StepName(name string) zap.Field {
    return zap.String(FieldStepName, name)
}

func NodeType(nodeType string) zap.Field {
    return zap.String(FieldNodeType, nodeType)
}

func ErrorType(errType string) zap.Field {
    return zap.String(FieldErrorType, errType)
}

func Retryable(retryable bool) zap.Field {
    return zap.Bool(FieldRetryable, retryable)
}
```

**使用示例:**
```go
logger.Info("Step executed",
    logger.WorkflowID(workflowID),
    logger.JobID(jobID),
    logger.StepName(stepName),
    logger.NodeType(nodeType),
    zap.Duration(logger.FieldDuration, duration),
)
```

### AC5: 规范化现有日志调用

**Given** 现有代码已使用 logger.Log 和注入的 *zap.Logger  
**When** 审查和更新现有日志调用  
**Then** 确保所有日志调用符合规范:
- ✅ 使用结构化字段 (不使用 fmt.Sprintf)
- ✅ 使用标准字段名
- ✅ 包含完整上下文
- ✅ 敏感信息不直接记录

**需要更新的模块:**
1. **cmd/server/main.go** - 启动日志
2. **cmd/agent/main.go** - 启动日志
3. **internal/server/** - API Handler 日志
4. **internal/agent/** - Worker 日志
5. **pkg/temporal/** - Workflow/Activity 日志

**Before (不规范):**
```go
logger.Info(fmt.Sprintf("Workflow %s started by user %s", workflowID, user))
```

**After (规范化):**
```go
logger.Info("Workflow started",
    logger.WorkflowID(workflowID),
    zap.String("user", user),
)
```

**Before (敏感信息):**
```go
logger.Info("SSH connection",
    zap.String("password", password),
)
```

**After (脱敏):**
```go
logger.Info("SSH connection",
    zap.String("host", host),
    zap.String("user", user),
    // password 自动脱敏,或不记录
)
```

### AC6: 性能关键路径优化

**Given** 日志可能影响性能关键路径  
**When** 在高频路径记录日志  
**Then** 使用 Zap 的零分配日志  
**And** 避免在 debug 级别下的昂贵操作  
**And** 使用 CheckedEntry 避免无效日志构造

**Implementation Notes:**
```go
// 使用 CheckedEntry 避免无效日志构造
if ce := logger.Check(zap.DebugLevel, "Expensive debug log"); ce != nil {
    // 只有在 debug 级别启用时才执行昂贵操作
    expensiveData := computeExpensiveData()
    ce.Write(zap.Any("data", expensiveData))
}

// 使用 zap.Field 避免字符串拼接
logger.Info("Request processed",
    zap.Int("status_code", 200),
    zap.Duration("duration", duration),
    zap.String("method", method),
    zap.String("path", path),
)

// 避免:
// logger.Info(fmt.Sprintf("Request %s %s took %v", method, path, duration))
```

**基准测试:**
```go
// pkg/logger/logger_bench_test.go
func BenchmarkStructuredLog(b *testing.B) {
    logger, _ := zap.NewProduction()
    defer logger.Sync()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        logger.Info("Benchmark message",
            zap.String("key1", "value1"),
            zap.Int("key2", 42),
        )
    }
}

func BenchmarkFormattedLog(b *testing.B) {
    logger, _ := zap.NewProduction()
    defer logger.Sync()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        logger.Sugar().Infof("Benchmark message: %s, %d", "value1", 42)
    }
}
```

### AC7: 日志使用文档和最佳实践

**Given** 开发者需要了解日志规范  
**When** 编写新代码时  
**Then** 提供完整的日志使用文档  
**And** 包含最佳实践指南  
**And** 提供常见场景示例

**文档结构:**
- `pkg/logger/README.md` - 日志包使用指南
- `docs/guides/logging-best-practices.md` - 最佳实践
- 代码注释和 GoDoc

**最佳实践:**
1. **使用结构化字段,不使用字符串拼接**
2. **使用标准字段名常量**
3. **不记录敏感信息 (或依赖自动脱敏)**
4. **为工作流上下文创建子 logger**
5. **在性能关键路径使用 CheckedEntry**
6. **Error 级别用于需要人工干预的错误**
7. **Info 级别用于重要的业务事件**
8. **Debug 级别用于开发调试信息**
9. **不直接记录用户输入,防止日志注入攻击** - 使用结构化字段自动转义
10. **不记录完整请求/响应体** - 可能包含敏感数据或导致日志膨胀

## Tasks / Subtasks

### Task 1: 实现敏感信息脱敏 (AC1)

- [ ] 1.1 创建 `pkg/logger/sanitizer.go`
  - SanitizingEncoder 结构
  - DefaultSensitiveFields 列表
  - AddString/AddByteString 方法重写

- [ ] 1.2 集成脱敏编码器到 Init
  - 修改 logger.Init 使用 zap.WrapCore
  - 包装 JSON/Console Encoder
  - 应用 DefaultSensitiveFields

- [ ] 1.3 支持自定义敏感字段
  - 定义 LogConfig 结构体
  - 实现 InitWithConfig 函数
  - 配置文件支持 log.sensitive_fields
  - 环境变量支持 WATERFLOW_LOG_SENSITIVE_FIELDS
  - 合并默认和自定义字段
  - 实现所有必需的 Encoder 接口方法 (委托模式)

- [ ] 1.4 编写脱敏测试
  - 默认敏感字段测试
  - 自定义字段测试
  - 大小写不敏感测试
  - 性能基准测试

### Task 2: 实现上下文日志 (AC2)

- [ ] 2.1 创建 `pkg/logger/context.go`
  - WithWorkflowContext
  - WithJobContext
  - WithStepContext
  - WithNodeContext
  - WithFields

- [ ] 2.2 更新 Workflow 执行器使用上下文 logger
  - Executor 注入 logger
  - 每个 Step 创建 stepLogger
  - Activity 使用上下文 logger

- [ ] 2.3 更新 Agent Worker 使用上下文 logger
  - Worker 注入 logger
  - PluginManager 使用上下文 logger

- [ ] 2.4 编写上下文日志测试
  - WithContext 方法测试
  - 字段继承测试
  - 链式调用测试

### Task 3: 实现日志级别动态配置 (AC3)

- [ ] 3.1 修改 logger.Init 使用 AtomicLevel
  - 声明全局 atomicLevel 变量
  - Init 中使用 zap.NewAtomicLevelAt
  - 保留 atomicLevel 引用

- [ ] 3.2 实现 SetLevel/GetLevel 函数
  - SetLevel 解析和验证级别
  - 调用 atomicLevel.SetLevel
  - GetLevel 返回当前级别

- [ ] 3.3 实现 Admin HTTP 端点
  - 创建 internal/server/handlers/admin.go
  - SetLogLevel PUT /admin/log-level (带认证检查)
  - GetLogLevel GET /admin/log-level (带认证检查)
  - 添加审计日志 (记录修改者、时间、新级别)
  - 注册路由并应用 middleware.RequireAuth 中间件

- [ ] 3.4 编写动态级别测试
  - SetLevel 功能测试
  - HTTP 端点集成测试
  - 并发安全测试

### Task 4: 标准化字段命名 (AC4)

- [ ] 4.1 创建 `pkg/logger/fields.go`
  - 字段名常量定义
  - 辅助函数 (WorkflowID, JobID, StepName 等)

- [ ] 4.2 更新 encoder 配置
  - 时间格式: ISO 8601
  - 级别字段: level
  - 消息字段: msg
  - Caller 字段: caller

- [ ] 4.3 编写字段使用示例
  - 工作流日志示例
  - 错误日志示例
  - 性能日志示例

- [ ] 4.4 编写字段命名测试
  - 常量值验证
  - 辅助函数输出验证

### Task 5: 规范化现有日志调用 (AC5)

- [ ] 5.1 审查 cmd/server/main.go
  - 替换 fmt.Sprintf
  - 使用标准字段
  - 添加缺失上下文

- [ ] 5.2 审查 cmd/agent/main.go
  - 替换 fmt.Sprintf
  - 使用标准字段
  - 添加缺失上下文

- [ ] 5.3 审查 internal/server/ 日志
  - API Handler 日志规范化
  - 错误日志集成 Story 7.1
  - 添加请求上下文

- [ ] 5.4 审查 internal/agent/ 日志
  - Worker 日志规范化
  - PluginManager 日志规范化
  - 添加 agent_id 上下文

- [ ] 5.5 审查 pkg/temporal/ 日志
  - Workflow 日志规范化
  - Activity 日志规范化
  - 添加 workflow_id 上下文

- [ ] 5.6 验证所有模块
  - 编译通过
  - 测试通过
  - golangci-lint 通过

### Task 6: 性能优化和基准测试 (AC6)

- [ ] 6.1 创建 `pkg/logger/logger_bench_test.go`
  - BenchmarkStructuredLog
  - BenchmarkFormattedLog
  - BenchmarkCheckedEntry
  - BenchmarkSanitization

- [ ] 6.2 优化高频日志路径
  - 识别性能关键路径
  - 使用 CheckedEntry
  - 避免昂贵操作

- [ ] 6.3 运行基准测试
  - 建立性能基线
  - 对比优化前后
  - 确保零分配

- [ ] 6.4 文档性能最佳实践
  - 何时使用 CheckedEntry
  - 避免的反模式

### Task 7: 文档和最佳实践 (AC7)

- [ ] 7.1 编写 `pkg/logger/README.md`
  - 包概述
  - 快速开始
  - API 参考
  - 配置说明

- [ ] 7.2 编写 `docs/guides/logging-best-practices.md`
  - 日志级别使用指南
  - 结构化日志最佳实践
  - 上下文日志模式
  - 敏感信息处理
  - 性能优化技巧
  - 日志注入攻击防护 (不直接记录用户输入)
  - /admin 端点安全配置 (认证、审计)

- [ ] 7.3 更新代码注释和 GoDoc
  - 所有公开函数添加注释
  - 示例代码
  - 使用说明

- [ ] 7.4 创建日志示例
  - 工作流执行日志
  - 错误处理日志
  - API 请求日志
  - Agent 日志

### Task 8: 集成测试和验证

- [ ] 8.1 端到端日志测试
  - 提交工作流,验证日志输出
  - 检查日志结构和字段
  - 验证敏感信息脱敏

- [ ] 8.2 日志查询测试
  - 使用 jq 查询 JSON 日志
  - 按 workflow_id 过滤
  - 按 level 过滤

- [ ] 8.3 性能测试
  - 高并发日志输出
  - CPU 和内存占用
  - 日志吞吐量

- [ ] 8.4 文档审查
  - README 完整性
  - 最佳实践准确性
  - 示例代码可运行

## Dev Notes

### Architecture Alignment

**日志架构 (本 Story):**
- ✅ 结构化日志 - JSON 格式,易于解析
- ✅ 敏感信息脱敏 - 自动过滤密码、Token
- ✅ 上下文日志 - workflow_id, job_id, step_name
- ✅ 日志级别动态配置 - 运行时调整
- ✅ 高性能日志 - Zap 零分配

**与 Story 7.1 集成 (错误处理):**
- ✅ 错误日志包含 error_type
- ✅ 错误日志包含 retryable 标志
- ✅ 使用 zap.Error 记录错误对象

**与 Story 7.7 准备 (LogHandler 接口):**
- ✅ 结构化日志为 LogHandler 奠定基础
- ✅ 标准字段命名便于 LogHandler 解析
- ❌ LogHandler 接口在 Story 7.7 实现

**与系统各模块集成:**
- ✅ Server - API 请求日志,工作流提交日志
- ✅ Agent - Worker 日志,插件加载日志
- ✅ Workflow - 执行日志,步骤日志
- ✅ Temporal - Activity 日志,错误日志

### Project Structure

```
pkg/
├── logger/
│   ├── logger.go           # ✅ 已存在,需要增强
│   ├── sanitizer.go        # 🆕 敏感信息脱敏
│   ├── context.go          # 🆕 上下文日志
│   ├── fields.go           # 🆕 标准字段常量
│   ├── logger_test.go      # ✅ 已存在,需要扩展
│   ├── logger_bench_test.go # 🆕 性能基准测试
│   └── README.md           # 🆕 使用指南

cmd/
├── server/main.go          # 🔧 需要规范化日志
├── agent/main.go           # 🔧 需要规范化日志

internal/
├── server/
│   ├── handlers/
│   │   ├── admin.go        # 🆕 日志级别管理端点
│   │   └── ...             # 🔧 规范化日志
│   └── router.go           # 🔧 注册 admin 路由
├── agent/
│   ├── worker.go           # 🔧 规范化日志
│   └── plugin_manager.go   # 🔧 规范化日志

docs/
└── guides/
    └── logging-best-practices.md  # 🆕 最佳实践文档
```

### 关键技术决策

**1. Zap vs Logrus**
- ✅ 选择: Zap
- 理由:
  - 性能优异 (零分配)
  - 结构化日志原生支持
  - 社区活跃,维护良好
- 已实现: pkg/logger 已使用 Zap

**2. 全局 logger vs 依赖注入**
- ✅ 选择: 混合模式
  - 全局 logger.Log 用于简单场景
  - 注入 *zap.Logger 用于复杂组件
- 理由:
  - 全局 logger 简化使用
  - 注入 logger 支持测试和上下文

**3. 脱敏实现方式**
- ✅ 选择: 包装 zapcore.Encoder
- 理由:
  - 透明,无需修改调用代码
  - 高效,在编码层脱敏
  - 可配置敏感字段列表
- 替代方案:
  - ❌ 在日志调用前手动脱敏 (容易遗漏)
  - ❌ 后处理日志文件 (性能差,不实时)

**4. 日志级别动态配置**
- ✅ 选择: zap.AtomicLevel + HTTP 端点
- 理由:
  - 无需重启调整级别
  - 支持远程管理
  - 线程安全
- 实现: 
  - AtomicLevel 支持并发安全修改
  - HTTP PUT /admin/log-level

**5. 上下文日志模式**
- ✅ 选择: logger.With 创建子 logger
- 理由:
  - 自动附加上下文字段
  - 避免重复传递参数
  - 符合 Zap 最佳实践
- 示例:
  ```go
  stepLogger := logger.With(
      zap.String("workflow_id", workflowID),
      zap.String("step_name", stepName),
  )
  stepLogger.Info("Step started")  // 自动包含 workflow_id, step_name
  ```

### 测试策略

**单元测试覆盖率目标: >85%**

**测试分类:**
1. **脱敏测试** - 验证敏感字段自动脱敏
2. **上下文测试** - 验证上下文字段正确附加
3. **级别配置测试** - 验证动态级别调整
4. **字段命名测试** - 验证标准字段常量
5. **性能基准测试** - 验证零分配和性能

**测试示例:**
```go
func TestSanitization(t *testing.T) {
    // 初始化带脱敏的 logger
    logger, _ := zap.NewDevelopment(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
        encoder := NewSanitizingEncoder(
            zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()),
            DefaultSensitiveFields,
        )
        return zapcore.NewCore(encoder, zapcore.AddSync(&testBuffer), zapcore.DebugLevel)
    }))
    
    logger.Info("User login",
        zap.String("username", "admin"),
        zap.String("password", "secret123"),
    )
    
    output := testBuffer.String()
    assert.Contains(t, output, `"username":"admin"`)
    assert.Contains(t, output, `"password":"***REDACTED***"`)
    assert.NotContains(t, output, "secret123")
}

func TestContextLogger(t *testing.T) {
    logger, _ := zap.NewDevelopment()
    
    workflowLogger := logger.With(zap.String("workflow_id", "wf-123"))
    
    workflowLogger.Info("Workflow started")
    // 验证输出包含 workflow_id
}

func BenchmarkStructuredLog(b *testing.B) {
    logger, _ := zap.NewProduction()
    defer logger.Sync()
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        logger.Info("Benchmark",
            zap.String("key1", "value1"),
            zap.Int("key2", 42),
        )
    }
    
    // 期望: 0 allocs/op
}
```

### 性能考虑

**日志性能要求:**
- ✅ 结构化日志零分配
- ✅ 脱敏开销 <25% (目标 1500ns vs 1200ns 基线)
- ✅ 上下文 logger 创建 <100ns

**优化策略:**
1. **使用 Zap 的零分配 API** - zap.String, zap.Int 等
2. **脱敏在编码层** - 避免字符串复制
3. **CheckedEntry 避免无效日志** - debug 级别检查
4. **缓冲输出** - 减少系统调用

**基准测试目标:**
```
BenchmarkStructuredLog-8      1000000      1200 ns/op      0 allocs/op
BenchmarkWithContext-8       10000000       120 ns/op      0 allocs/op
BenchmarkSanitization-8       800000      1500 ns/op      0 allocs/op  (脱敏开销 ~25%)
```

**注意:** 脱敏开销约 25% (1500ns vs 1200ns),在可接受范围内,因为安全性更重要。

### 安全考虑

**敏感信息保护:**
- ✅ 自动脱敏常见敏感字段
- ✅ 支持自定义敏感字段列表
- ✅ 不区分大小写匹配

**访问控制:**
- ✅ /admin 端点需要认证 (Bearer Token 或 API Key)
- ✅ 使用 middleware.RequireAuth 中间件保护
- ✅ 审计日志记录所有级别修改操作

**日志注入防护:**
- ✅ 使用结构化字段自动转义特殊字符
- ✅ 不直接记录用户输入到消息字段
- ✅ 验证和清理日志字段值
- ⚠️ 示例:
  ```go
  // ❌ 错误: 直接记录用户输入
  logger.Info(fmt.Sprintf("User input: %s", userInput))
  
  // ✅ 正确: 使用结构化字段
  logger.Info("User input received",
      zap.String("input", userInput), // Zap 自动转义
  )
  ```

**默认脱敏字段:**
- password
- token
- api_key
- secret
- private_key
- access_token
- refresh_token
- authorization
- cookie

**最佳实践:**
1. **不记录完整的请求/响应体** - 可能包含敏感数据
2. **记录用户名,不记录密码**
3. **记录错误类型,不记录敏感详情**
4. **使用 workflow_id 关联日志,不记录业务敏感字段**
5. **使用结构化字段防止日志注入** - Zap 自动转义特殊字符
6. **不直接将用户输入拼接到 msg 字段** - 使用独立字段

### 向后兼容性

**兼容性保证:**
- ✅ 现有 logger.Log 调用继续工作
- ✅ 现有注入的 *zap.Logger 继续工作
- ✅ 新增功能可选使用

**迁移建议:**
1. **新代码** - 直接使用标准字段和上下文 logger
2. **现有代码** - 逐步迁移,不破坏现有功能
3. **测试** - 确保所有测试通过

### 可观测性增强

**日志查询示例:**

**1. 按 workflow_id 查询:**
```bash
jq 'select(.workflow_id == "wf-abc-123")' < server.log
```

**2. 按级别过滤:**
```bash
jq 'select(.level == "error")' < server.log
```

**3. 统计错误类型:**
```bash
jq -r 'select(.level == "error") | .error_type' < server.log | sort | uniq -c
```

**4. 计算平均执行时间:**
```bash
jq -s 'map(select(.msg == "Step completed") | .duration_ms) | add/length' < server.log
```

**集成到日志系统:**
- ✅ ELK Stack - Filebeat → Elasticsearch → Kibana
- ✅ Loki - Promtail → Loki → Grafana
- ✅ CloudWatch Logs - awslogs driver
- ✅ Splunk - Splunk Forwarder

### 未来扩展

**Post-MVP 增强:**
1. **日志采样** - 高频日志采样 (1/100)
2. **日志聚合** - 批量发送,减少网络开销
3. **Trace ID** - 集成 OpenTelemetry Trace
4. **日志压缩** - gzip 压缩日志文件
5. **日志轮转** - 基于大小或时间轮转

**扩展点 (Story 7.7 准备):**
```go
type LogHandler interface {
    OnLog(ctx context.Context, entry LogEntry) error
}

type LogEntry struct {
    Timestamp  time.Time
    Level      string
    Logger     string
    Message    string
    Fields     map[string]interface{}
    WorkflowID string
    JobID      string
    StepName   string
}
```

## Completion Criteria

**Story 完成标准:**

1. ✅ **所有 AC 完成**
   - AC1-AC7 所有验收标准通过
   - 单元测试覆盖率 > 85%

2. ✅ **代码质量**
   - golangci-lint 无错误
   - 所有测试通过
   - 基准测试达标 (零分配)

3. ✅ **文档完整**
   - pkg/logger/README.md 完整
   - logging-best-practices.md 完整
   - 代码注释和 GoDoc 完整

4. ✅ **集成验证**
   - 所有模块日志规范化
   - 敏感信息脱敏验证
   - 动态级别调整验证
   - /admin 端点认证验证
   - 审计日志记录验证

5. ✅ **性能达标**
   - 结构化日志零分配
   - 脱敏开销 <5%
   - 基准测试通过

**验收测试场景:**

**场景 1: 敏感信息脱敏**
```bash
# 提交包含密码的工作流
curl -X POST http://localhost:8080/v1/workflows \
  -H "Content-Type: application/yaml" \
  -d @workflow-with-password.yaml

# 检查日志
cat server.log | jq 'select(.password)'
# 期望: "password": "***REDACTED***"
```

**场景 2: 上下文日志**
```bash
# 提交工作流
curl -X POST http://localhost:8080/v1/workflows \
  -d @hello-world.yaml

# 检查日志包含 workflow_id
cat server.log | jq 'select(.workflow_id)'
# 期望: 所有相关日志包含相同的 workflow_id
```

**场景 3: 动态级别调整 (带认证)**
```bash
# 获取当前日志级别 (需要认证)
curl http://localhost:8080/admin/log-level \
  -H "Authorization: Bearer <token>"
# 期望: {"level": "info"}

# 未认证访问 - 应该失败
curl http://localhost:8080/admin/log-level
# 期望: 401 Unauthorized

# 调整到 debug 级别 (需要认证)
curl -X PUT http://localhost:8080/admin/log-level \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"level": "debug"}'

# 验证审计日志
cat server.log | jq 'select(.msg == "Log level changed")'
# 期望: 包含 changed_by, level, remote_addr

# 验证 debug 日志出现
tail -f server.log | jq 'select(.level == "debug")'
```

**场景 4: 日志查询**
```bash
# 查询某个工作流的所有日志
jq 'select(.workflow_id == "wf-xxx")' < server.log

# 统计错误类型分布
jq -r 'select(.level == "error") | .error_type' < server.log | sort | uniq -c

# 计算平均步骤执行时间
jq -s 'map(select(.msg == "Step completed") | .duration_ms) | add/length' < server.log
```

## References

**现有代码参考:**
- [pkg/logger/logger.go](../../pkg/logger/logger.go) - 基础日志实现
- [cmd/server/main.go](../../cmd/server/main.go) - Server 启动日志
- [cmd/agent/main.go](../../cmd/agent/main.go) - Agent 启动日志
- [internal/agent/plugin_manager.go](../../internal/agent/plugin_manager.go) - 插件管理日志
- [internal/agent/worker.go](../../internal/agent/worker.go) - Worker 日志

**相关 Story:**
- Story 7.1 - 类型化错误处理 (错误日志集成)
- Story 7.7 - LogHandler 接口 (日志处理扩展点)
- Story 7.5 - Prometheus 指标 (监控集成)

**外部参考:**
- [Uber Zap](https://github.com/uber-go/zap) - 官方文档
- [Best Practices for Application Logging](https://www.dataset.com/blog/the-10-commandments-of-logging/)
- [Structured Logging](https://www.karllhughes.com/posts/structured-logging)
- [12-Factor App - Logs](https://12factor.net/logs)

---

**创建日期:** 2026-01-07  
**创建者:** SM Agent (Bob)  
**Epic:** 7 - 生产级可靠性  
**依赖:** Story 7.1 完成  
**预估点数:** 8 points (中等复杂度,影响面广)  
**优先级:** High (Epic 7 的核心 Story)
