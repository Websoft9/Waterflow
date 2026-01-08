package metrics

import (
	"time"
)

// NodeTracker tracks node execution metrics
type NodeTracker struct{}

// NewNodeTracker creates a new node tracker
func NewNodeTracker() *NodeTracker {
	return &NodeTracker{}
}

// TrackExecution records a node execution
func (nt *NodeTracker) TrackExecution(nodeType string, duration time.Duration, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}

	NodeExecutionsTotal.WithLabelValues(nodeType, status).Inc()
	NodeExecutionDuration.WithLabelValues(nodeType).Observe(duration.Seconds())
}

// TrackNodeStart returns a function to be called when node completes
// Usage:
//
//	done := nodeTracker.TrackNodeStart("exec/shell")
//	defer done(err == nil)
func (nt *NodeTracker) TrackNodeStart(nodeType string) func(bool) {
	startTime := time.Now()

	return func(success bool) {
		duration := time.Since(startTime)
		nt.TrackExecution(nodeType, duration, success)
	}
}
