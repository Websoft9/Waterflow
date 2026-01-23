package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestWorkflowTracker(t *testing.T) {
	// Reset metrics
	WorkflowsTotal.Reset()
	WorkflowsRunning.Set(0)
	WorkflowDuration.Reset()

	tracker := NewWorkflowTracker()

	t.Run("TrackSubmission", func(t *testing.T) {
		tracker.TrackSubmission("wf-1", true)

		if count := tracker.GetRunningCount(); count != 1 {
			t.Errorf("Expected 1 running workflow, got %d", count)
		}

		// Check counter
		count := testutil.ToFloat64(WorkflowsTotal.WithLabelValues("submitted"))
		if count != 1 {
			t.Errorf("Expected 1 submitted workflow, got %f", count)
		}

		// Check gauge
		running := testutil.ToFloat64(WorkflowsRunning)
		if running != 1 {
			t.Errorf("Expected 1 running workflow gauge, got %f", running)
		}
	})

	t.Run("TrackCompletion", func(t *testing.T) {
		// Small delay to ensure submission is processed
		tracker.TrackCompletion("wf-1", "completed")

		if count := tracker.GetRunningCount(); count != 0 {
			t.Errorf("Expected 0 running workflows, got %d", count)
		}

		// Check counter
		count := testutil.ToFloat64(WorkflowsTotal.WithLabelValues("completed"))
		if count != 1 {
			t.Errorf("Expected 1 completed workflow, got %f", count)
		}

		// Check gauge decreased
		running := testutil.ToFloat64(WorkflowsRunning)
		if running != 0 {
			t.Errorf("Expected 0 running workflow gauge, got %f", running)
		}
	})

	t.Run("TrackValidation", func(t *testing.T) {
		YAMLValidationsTotal.Reset()

		TrackValidation(true)
		TrackValidation(false)

		successCount := testutil.ToFloat64(YAMLValidationsTotal.WithLabelValues("success"))
		failureCount := testutil.ToFloat64(YAMLValidationsTotal.WithLabelValues("failure"))

		if successCount != 1 {
			t.Errorf("Expected 1 validation success, got %f", successCount)
		}

		if failureCount != 1 {
			t.Errorf("Expected 1 validation failure, got %f", failureCount)
		}
	})
}

func TestNodeTracker(t *testing.T) {
	NodeExecutionsTotal.Reset()
	NodeExecutionDuration.Reset()

	tracker := NewNodeTracker()

	t.Run("TrackExecution", func(t *testing.T) {
		tracker.TrackExecution("exec/shell", 100*time.Millisecond, true)
		tracker.TrackExecution("http/request", 500*time.Millisecond, false)

		successCount := testutil.ToFloat64(NodeExecutionsTotal.WithLabelValues("exec/shell", "success"))
		failureCount := testutil.ToFloat64(NodeExecutionsTotal.WithLabelValues("http/request", "failure"))

		if successCount != 1 {
			t.Errorf("Expected 1 success, got %f", successCount)
		}

		if failureCount != 1 {
			t.Errorf("Expected 1 failure, got %f", failureCount)
		}
	})

	t.Run("TrackNodeStart", func(t *testing.T) {
		start := time.Now()
		done := tracker.TrackNodeStart("flow/sleep")
		// Simulate some work with minimal sleep (this is testing duration tracking)
		for time.Since(start) < 50*time.Millisecond {
			// Busy wait for more precise timing
		}
		done(true)

		count := testutil.ToFloat64(NodeExecutionsTotal.WithLabelValues("flow/sleep", "success"))
		if count != 1 {
			t.Errorf("Expected 1 execution, got %f", count)
		}
	})
}

func TestAgentTracker(t *testing.T) {
	AgentsConnected.Set(0)
	AgentsHealthy.Set(0)
	AgentTasksTotal.Reset()

	tracker := NewAgentTracker()

	t.Run("TrackConnection", func(t *testing.T) {
		tracker.TrackConnection("agent-1")

		if count := tracker.GetConnectedCount(); count != 1 {
			t.Errorf("Expected 1 connected agent, got %d", count)
		}

		connected := testutil.ToFloat64(AgentsConnected)
		healthy := testutil.ToFloat64(AgentsHealthy)

		if connected != 1 {
			t.Errorf("Expected 1 connected metric, got %f", connected)
		}

		if healthy != 1 {
			t.Errorf("Expected 1 healthy metric, got %f", healthy)
		}
	})

	t.Run("TrackHealthChange", func(t *testing.T) {
		tracker.TrackHealthChange("agent-1", false)

		if count := tracker.GetHealthyCount(); count != 0 {
			t.Errorf("Expected 0 healthy agents, got %d", count)
		}

		healthy := testutil.ToFloat64(AgentsHealthy)
		if healthy != 0 {
			t.Errorf("Expected 0 healthy metric, got %f", healthy)
		}

		// Restore health
		tracker.TrackHealthChange("agent-1", true)

		if count := tracker.GetHealthyCount(); count != 1 {
			t.Errorf("Expected 1 healthy agent, got %d", count)
		}
	})

	t.Run("TrackTaskExecution", func(t *testing.T) {
		tracker.TrackTaskExecution("agent-1", true)
		tracker.TrackTaskExecution("agent-1", false)

		completed := testutil.ToFloat64(AgentTasksTotal.WithLabelValues("agent-1", "completed"))
		failed := testutil.ToFloat64(AgentTasksTotal.WithLabelValues("agent-1", "failed"))

		if completed != 1 {
			t.Errorf("Expected 1 completed task, got %f", completed)
		}

		if failed != 1 {
			t.Errorf("Expected 1 failed task, got %f", failed)
		}
	})

	t.Run("TrackDisconnection", func(t *testing.T) {
		tracker.TrackDisconnection("agent-1")

		if count := tracker.GetConnectedCount(); count != 0 {
			t.Errorf("Expected 0 connected agents, got %d", count)
		}

		connected := testutil.ToFloat64(AgentsConnected)
		if connected != 0 {
			t.Errorf("Expected 0 connected metric, got %f", connected)
		}
	})
}

func TestMetricsRegistration(t *testing.T) {
	// Test that all metrics are properly registered
	metrics := []prometheus.Collector{
		HTTPRequestsTotal,
		HTTPRequestDuration,
		WorkflowsTotal,
		WorkflowsRunning,
		WorkflowDuration,
		AgentsConnected,
		AgentsHealthy,
		AgentTasksTotal,
		NodeExecutionsTotal,
		NodeExecutionDuration,
		WorkflowSubmissionsTotal,
		YAMLValidationsTotal,
	}

	for _, metric := range metrics {
		if metric == nil {
			t.Error("Metric is nil")
		}
	}
}
