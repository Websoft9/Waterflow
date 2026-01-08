package metrics

import (
	"sync"
	"time"
)

// WorkflowTracker tracks workflow lifecycle metrics
type WorkflowTracker struct {
	mu               sync.RWMutex
	runningWorkflows map[string]time.Time // workflowID -> start time
}

// NewWorkflowTracker creates a new workflow tracker
func NewWorkflowTracker() *WorkflowTracker {
	return &WorkflowTracker{
		runningWorkflows: make(map[string]time.Time),
	}
}

// TrackSubmission records a workflow submission
func (wt *WorkflowTracker) TrackSubmission(workflowID string, success bool) {
	if success {
		WorkflowsTotal.WithLabelValues("submitted").Inc()
		WorkflowSubmissionsTotal.WithLabelValues("success").Inc()

		wt.mu.Lock()
		wt.runningWorkflows[workflowID] = time.Now()
		wt.mu.Unlock()

		WorkflowsRunning.Inc()
	} else {
		WorkflowSubmissionsTotal.WithLabelValues("validation_error").Inc()
	}
}

// TrackCompletion records a workflow completion
func (wt *WorkflowTracker) TrackCompletion(workflowID string, status string) {
	wt.mu.Lock()
	startTime, exists := wt.runningWorkflows[workflowID]
	if exists {
		delete(wt.runningWorkflows, workflowID)
	}
	wt.mu.Unlock()

	if !exists {
		// Workflow not tracked (possibly started before tracker initialization)
		return
	}

	// Update metrics
	WorkflowsRunning.Dec()
	WorkflowsTotal.WithLabelValues(status).Inc()

	// Record duration
	duration := time.Since(startTime).Seconds()
	WorkflowDuration.WithLabelValues(status).Observe(duration)
}

// GetRunningCount returns the number of running workflows
func (wt *WorkflowTracker) GetRunningCount() int {
	wt.mu.RLock()
	defer wt.mu.RUnlock()
	return len(wt.runningWorkflows)
}

// TrackValidation records YAML validation result
func TrackValidation(success bool) {
	if success {
		YAMLValidationsTotal.WithLabelValues("success").Inc()
	} else {
		YAMLValidationsTotal.WithLabelValues("failure").Inc()
	}
}
