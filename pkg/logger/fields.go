// Package logger provides structured logging functionality using zap.
package logger

import "go.uber.org/zap"

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

// 辅助函数 - 提供类型安全的字段构造

// WorkflowID 创建 workflow_id 字段
func WorkflowID(id string) zap.Field {
	return zap.String(FieldWorkflowID, id)
}

// WorkflowName 创建 workflow_name 字段
func WorkflowName(name string) zap.Field {
	return zap.String(FieldWorkflowName, name)
}

// WorkflowStatus 创建 workflow_status 字段
func WorkflowStatus(status string) zap.Field {
	return zap.String(FieldWorkflowStatus, status)
}

// JobID 创建 job_id 字段
func JobID(id string) zap.Field {
	return zap.String(FieldJobID, id)
}

// StepName 创建 step_name 字段
func StepName(name string) zap.Field {
	return zap.String(FieldStepName, name)
}

// NodeType 创建 node_type 字段
func NodeType(nodeType string) zap.Field {
	return zap.String(FieldNodeType, nodeType)
}

// ErrorType 创建 error_type 字段
func ErrorType(errType string) zap.Field {
	return zap.String(FieldErrorType, errType)
}

// Retryable 创建 retryable 字段
func Retryable(retryable bool) zap.Field {
	return zap.Bool(FieldRetryable, retryable)
}

// Component 创建 component 字段
func Component(component string) zap.Field {
	return zap.String(FieldComponent, component)
}

// Version 创建 version 字段
func Version(version string) zap.Field {
	return zap.String(FieldVersion, version)
}

// Host 创建 host 字段
func Host(host string) zap.Field {
	return zap.String(FieldHost, host)
}

// PID 创建 pid 字段
func PID(pid int) zap.Field {
	return zap.Int(FieldPID, pid)
}
