package events

import (
	"context"
	"time"

	"github.com/Websoft9/waterflow/pkg/temporal"
	"go.temporal.io/api/enums/v1"
	"go.uber.org/zap"
)

// WorkflowMonitor monitors workflow executions and dispatches lifecycle events.
type WorkflowMonitor struct {
	temporalClient  *temporal.Client
	eventDispatcher *EventDispatcher
	logger          *zap.Logger
}

// NewWorkflowMonitor creates a new workflow monitor.
func NewWorkflowMonitor(temporalClient *temporal.Client, eventDispatcher *EventDispatcher, logger *zap.Logger) *WorkflowMonitor {
	return &WorkflowMonitor{
		temporalClient:  temporalClient,
		eventDispatcher: eventDispatcher,
		logger:          logger,
	}
}

// MonitorWorkflow starts monitoring a workflow execution for completion/failure events.
// This method is non-blocking and returns immediately.
func (m *WorkflowMonitor) MonitorWorkflow(workflowID string, startTime time.Time) {
	go m.monitorWorkflowExecution(workflowID, startTime)
}

// monitorWorkflowExecution polls workflow status until completion or failure.
func (m *WorkflowMonitor) monitorWorkflowExecution(workflowID string, startTime time.Time) {
	ctx := context.Background()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(24 * time.Hour) // Max monitoring duration

	for {
		select {
		case <-ticker.C:
			status, err := m.getWorkflowStatus(ctx, workflowID)
			if err != nil {
				m.logger.Debug("Failed to get workflow status",
					zap.String("workflow_id", workflowID),
					zap.Error(err),
				)
				continue
			}

			switch status {
			case "completed":
				m.dispatchCompleteEvent(workflowID, startTime)
				return
			case "failed":
				m.dispatchFailedEvent(workflowID, startTime)
				return
			case "canceled":
				// Optionally dispatch canceled event in future
				m.logger.Debug("Workflow canceled",
					zap.String("workflow_id", workflowID),
				)
				return
			case "running":
				// Continue monitoring
				continue
			}

		case <-timeout:
			m.logger.Warn("Workflow monitoring timeout",
				zap.String("workflow_id", workflowID),
			)
			return
		}
	}
}

// getWorkflowStatus queries Temporal for workflow execution status.
func (m *WorkflowMonitor) getWorkflowStatus(ctx context.Context, workflowID string) (string, error) {
	desc, err := m.temporalClient.GetClient().DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return "", err
	}

	status := desc.GetWorkflowExecutionInfo().GetStatus()
	switch status {
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		return "completed", nil
	case enums.WORKFLOW_EXECUTION_STATUS_FAILED:
		return "failed", nil
	case enums.WORKFLOW_EXECUTION_STATUS_CANCELED:
		return "canceled", nil
	case enums.WORKFLOW_EXECUTION_STATUS_RUNNING:
		return "running", nil
	default:
		return "unknown", nil
	}
}

// dispatchCompleteEvent sends workflow completion event.
func (m *WorkflowMonitor) dispatchCompleteEvent(workflowID string, startTime time.Time) {
	duration := time.Since(startTime).Seconds()

	m.eventDispatcher.DispatchWorkflowComplete(&WorkflowCompleteEvent{
		EventType:       "workflow.completed",
		WorkflowID:      workflowID,
		Timestamp:       time.Now(),
		StartTime:       startTime,
		DurationSeconds: duration,
		Result: &WorkflowResult{
			Status: "completed",
		},
	})

	m.logger.Debug("Workflow complete event dispatched",
		zap.String("workflow_id", workflowID),
		zap.Float64("duration_seconds", duration),
	)
}

// dispatchFailedEvent sends workflow failure event.
func (m *WorkflowMonitor) dispatchFailedEvent(workflowID string, startTime time.Time) {
	duration := time.Since(startTime).Seconds()

	// Get failure details from Temporal
	ctx := context.Background()
	desc, err := m.temporalClient.GetClient().DescribeWorkflowExecution(ctx, workflowID, "")

	var errorMsg string
	var errorType string
	if err == nil && desc.GetWorkflowExecutionInfo().GetStatus() == enums.WORKFLOW_EXECUTION_STATUS_FAILED {
		// Extract failure information if available
		errorMsg = "Workflow execution failed"
		errorType = "WorkflowFailure"
	} else {
		errorMsg = "Unknown failure"
		errorType = "UnknownError"
	}

	m.eventDispatcher.DispatchWorkflowFailed(&WorkflowFailedEvent{
		EventType:       "workflow.failed",
		WorkflowID:      workflowID,
		Timestamp:       time.Now(),
		StartTime:       startTime,
		DurationSeconds: duration,
		Error: &WorkflowError{
			Message: errorMsg,
			Type:    errorType,
		},
	})

	m.logger.Debug("Workflow failed event dispatched",
		zap.String("workflow_id", workflowID),
		zap.Float64("duration_seconds", duration),
	)
}
