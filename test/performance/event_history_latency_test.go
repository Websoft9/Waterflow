//go:build performance
// +build performance

package performance

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/sdk"
	"go.temporal.io/sdk/client"
)

// TestEventHistoryQueryLatency validates Temporal Event History query performance
// PRD AC6: Event History 查询延迟 < 100ms
// Story: 7.3-PERF-006
func TestEventHistoryQueryLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Event History latency test in short mode")
	}

	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "localhost:7233"
	}

	namespace := os.Getenv("TEMPORAL_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	// Connect to Temporal
	temporalClient, err := client.Dial(client.Options{
		HostPort:  temporalHost,
		Namespace: namespace,
	})
	if err != nil {
		t.Fatalf("Failed to connect to Temporal at %s: %v", temporalHost, err)
	}
	defer temporalClient.Close()

	// Submit a test workflow to create Event History
	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	waterflowClient, err := sdk.NewClient(&sdk.ClientConfig{
		ServerURL: serverURL,
	})
	if err != nil {
		t.Fatalf("Failed to create Waterflow client: %v", err)
	}

	// Load and submit test workflow
	workflowYAML, err := os.ReadFile("../../examples/hello-world.yaml")
	if err != nil {
		t.Fatalf("Failed to read workflow file: %v", err)
	}

	resp, err := waterflowClient.SubmitWorkflow(context.Background(), &sdk.SubmitWorkflowRequest{
		YAML: string(workflowYAML),
	})
	if err != nil {
		t.Fatalf("Failed to submit workflow: %v", err)
	}

	workflowID := resp.WorkflowID
	runID := resp.RunID
	t.Logf("Submitted test workflow: %s (run: %s)", workflowID, runID)

	// Wait for workflow to complete
	time.Sleep(10 * time.Second)

	// Measure Event History query latency
	samples := 50
	latencies := make([]time.Duration, samples)

	t.Logf("Querying Event History %d times...", samples)

	for i := 0; i < samples; i++ {
		start := time.Now()

		iter := temporalClient.GetWorkflowHistory(
			context.Background(),
			workflowID,
			runID,
			false, // waitForNewEvent
			0,     // historyEventFilterType
		)

		// Consume all events
		eventCount := 0
		for iter.HasNext() {
			_, err := iter.Next()
			if err != nil {
				t.Logf("Error reading event %d: %v", eventCount, err)
				break
			}
			eventCount++
		}

		latency := time.Since(start)
		latencies[i] = latency

		t.Logf("[%02d/%02d] Query latency: %v (%d events)", i+1, samples, latency, eventCount)

		time.Sleep(100 * time.Millisecond)
	}

	// Calculate statistics
	p50, p95, p99 := calculatePercentiles(latencies)
	avg := calculateAverage(latencies)

	t.Logf("=== Event History Query Latency ===")
	t.Logf("Samples: %d", samples)
	t.Logf("Average: %v", avg)
	t.Logf("P50: %v", p50)
	t.Logf("P95: %v", p95)
	t.Logf("P99: %v", p99)

	// Validate AC6
	if p99 > 100*time.Millisecond {
		t.Errorf("❌ P99 latency %v exceeds 100ms threshold", p99)
	} else {
		t.Logf("✅ Event History query latency within limit (P99: %v < 100ms)", p99)
	}

	if p50 > 50*time.Millisecond {
		t.Logf("⚠️  P50 latency %v exceeds 50ms target", p50)
	} else {
		t.Logf("✅ P50 latency excellent (%v < 50ms)", p50)
	}
}

// calculateAverage computes average duration
func calculateAverage(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	var sum time.Duration
	for _, d := range latencies {
		sum += d
	}

	return sum / time.Duration(len(latencies))
}
