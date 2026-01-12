# Audit Logging Package

This package provides audit logging functionality for Waterflow, enabling security monitoring and compliance tracking.

## Features

- **Immutable Audit Logs**: Append-only file storage for tamper-proof audit trails
- **Structured Events**: JSON Lines format for easy parsing and analysis
- **File Rotation**: Automatic log rotation with compression using lumberjack
- **Event Categories**: Workflow, secret, auth, config, agent, and admin events
- **Query Support**: Filter logs by time range, event type, user, resource, and result
- **Secret Provider Wrapper**: Automatic audit logging for all secret access operations
- **Workflow Integration**: Audit logging for workflow submit, cancel, and rerun operations

## Quick Start

### 1. Create File Audit Store

```go
import "github.com/Websoft9/waterflow/pkg/audit"

// Configure file-based audit logging
config := &audit.FileAuditStoreConfig{
    Path:       "/var/log/waterflow/audit",
    MaxSize:    100,  // 100 MB per file
    MaxAge:     90,   // Keep logs for 90 days
    MaxBackups: 30,   // Keep up to 30 old log files
    Compress:   true, // Compress rotated files
}

auditLogger, err := audit.NewFileAuditStore(config)
if err != nil {
    log.Fatal(err)
}
defer auditLogger.Close()
```

### 2. Log Audit Events

```go
// Log workflow submission
entry := audit.NewAuditLogEntry(audit.EventWorkflowSubmit, audit.CategoryWorkflow).
    WithUser(&audit.UserContext{
        ID:   "user123",
        Name: "Alice",
        IP:   "192.168.1.100",
    }).
    WithResource(&audit.ResourceContext{
        Type: "workflow",
        ID:   "wf-abc123",
        Name: "deploy-production",
    }).
    WithAction("submit").
    WithResult(audit.ResultSuccess).
    WithDetail("task_queue", "default").
    WithDetail("run_id", "run-xyz789")

err := auditLogger.Log(context.Background(), entry)
```

### 3. Query Audit Logs

```go
// Query recent workflow events
filter := audit.AuditLogFilter{
    EventCategory: &audit.CategoryWorkflow,
    Limit:         100,
}

logs, err := auditLogger.Query(context.Background(), filter)
if err != nil {
    log.Fatal(err)
}

for _, entry := range logs {
    fmt.Printf("[%s] %s: %s (%s)\n",
        entry.Timestamp.Format(time.RFC3339),
        entry.EventType,
        entry.Action,
        entry.Result,
    )
}
```

## Event Types

### Workflow Events
- `workflow.submit` - Workflow submitted for execution
- `workflow.cancel` - Workflow cancellation requested
- `workflow.rerun` - Workflow rerun initiated
- `workflow.status` - Workflow status queried
- `workflow.list` - Workflow list queried

### Secret Events
- `secret.access` - Secret retrieved
- `secret.list` - Multiple secrets retrieved
- `secret.not_found` - Secret not found

### Auth Events
- `auth.success` - Authentication successful
- `auth.failure` - Authentication failed
- `auth.logout` - User logged out
- `auth.session_expired` - Session expired

### Config Events
- `config.update` - Configuration updated
- `config.reload` - Configuration reloaded
- `config.validate` - Configuration validated

### Agent Events
- `agent.register` - Agent registered
- `agent.heartbeat` - Agent heartbeat
- `agent.disconnect` - Agent disconnected

### Admin Events
- `admin.user_create` - User created
- `admin.role_assign` - Role assigned
- `admin.policy_update` - Policy updated

## Audit Secret Provider

Wrap any `SecretProvider` to automatically log all secret access:

```go
// Create base provider (e.g., Vault)
vaultProvider, err := secrets.NewVaultSecretProvider(config)

// Wrap with audit logging
auditedProvider := audit.NewAuditSecretProvider(vaultProvider, auditLogger)

// All secret access is now automatically audited
secret, err := auditedProvider.GetSecret(ctx, "api_key")
```

## Workflow Handler Integration

Integrate audit logging into workflow handlers:

```go
// Create workflow handlers
handlers := api.NewWorkflowHandlers(logger, temporalClient, eventDispatcher)

// Set audit logger
handlers.SetAuditLogger(auditLogger)

// All workflow operations (submit, cancel, rerun) are now audited automatically
```

## Log Format

Audit logs are stored in JSON Lines format (one JSON object per line):

```json
{"timestamp":"2026-01-12T10:30:00Z","event_type":"workflow.submit","event_category":"workflow","severity":"info","user":{"id":"user123","name":"Alice","ip":"192.168.1.100"},"resource":{"type":"workflow","id":"wf-abc123","name":"deploy-production"},"action":"submit","result":"success","details":{"task_queue":"default","run_id":"run-xyz789"},"metadata":{}}
```

## Query Filters

Filter audit logs by multiple criteria:

```go
filter := audit.AuditLogFilter{
    StartTime:     &startTime,
    EndTime:       &endTime,
    EventTypes:    []string{audit.EventWorkflowSubmit, audit.EventWorkflowCancel},
    EventCategory: &audit.CategoryWorkflow,
    UserID:        "user123",
    ResourceType:  "workflow",
    ResourceID:    "wf-abc123",
    Result:        &audit.ResultSuccess,
    Limit:         100,
    Offset:        0,
}

logs, err := auditLogger.Query(ctx, filter)
```

## Compliance Features

- **Immutability**: Append-only logs prevent tampering
- **Retention**: Configurable retention policies (MaxAge, MaxBackups)
- **Compression**: Automatic compression of old logs to save space
- **Structured Format**: JSON Lines for easy parsing and SIEM integration
- **Complete Context**: User, resource, action, result, and metadata
- **Timestamp Precision**: UTC timestamps with ISO 8601 format

## Best Practices

1. **Always check errors**: Audit logging errors should be logged but not block operations
2. **Use structured details**: Add relevant context in `details` map
3. **Set appropriate severity**: Use `SeverityWarn` for suspicious events, `SeverityError` for failures
4. **Sanitize sensitive data**: Never log secret values, only secret keys/names
5. **Query efficiently**: Use specific filters and limits to avoid scanning large files
6. **Monitor storage**: Set up alerts for audit log storage capacity

## Testing

Run audit package tests:

```bash
go test ./pkg/audit/... -v
```

All 15 test suites should pass:
- FileAuditStore creation and configuration
- Log writing and query operations
- Filter matching logic
- Builder pattern fluent API
- Secret provider auditing
- Event types and categories

## Architecture

```
pkg/audit/
├── logger.go           # Core interfaces and types
├── builder.go          # Fluent API builder
├── file_store.go       # File-based storage implementation
├── secret_provider.go  # Secret provider audit wrapper
└── *_test.go          # Comprehensive test suite
```

## Dependencies

- `gopkg.in/natefinch/lumberjack.v2` - Log rotation
- `github.com/Websoft9/waterflow/pkg/secrets` - Secret provider interface

## Future Enhancements

- Database backend for better query performance
- Real-time audit event streaming
- Audit log encryption at rest
- Integration with external SIEM systems
- Compliance report generation
