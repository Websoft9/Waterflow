// Package logger provides structured logging functionality using zap.
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
