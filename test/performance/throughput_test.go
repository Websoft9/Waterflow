package performance

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/sdk"
)

// Note: These tests use relative paths and must be run from the project root or test/performance directory
// Example: go test -v ./test/performance -run=TestWorkflowThroughput

// TestWorkflowThroughput tests workflow submission throughput
// AC3: Throughput > 100 workflows/sec
// Error rate < 1%
func TestWorkflowThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping throughput test in short mode")
	}

	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	client, err := sdk.NewClient(&sdk.ClientConfig{
		ServerURL: serverURL,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Load workflow
	workflowYAML, err := os.ReadFile("../../examples/hello-world.yaml")
	if err != nil {
		t.Fatal(err)
	}

	duration := 30 * time.Second
	targetRate := 100 // workflows/sec

	t.Logf("Testing throughput against %s", serverURL)
	t.Logf("Duration: %v", duration)
	t.Logf("Target: %d workflows/sec", targetRate)

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var (
		submitted atomic.Int64
		errors    atomic.Int64
	)

	// Concurrent workers
	concurrency := 50
	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					_, err := client.SubmitWorkflow(context.Background(), &sdk.SubmitWorkflowRequest{
						YAML: string(workflowYAML),
					})

					if err != nil {
						errors.Add(1)
					} else {
						submitted.Add(1)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	elapsed := time.Since(start)

	submittedCount := submitted.Load()
	errorCount := errors.Load()
	total := submittedCount + errorCount

	actualRate := float64(submittedCount) / elapsed.Seconds()
	errorRate := float64(errorCount) / float64(total) * 100

	t.Logf("=== Results ===")
	t.Logf("Submitted: %d workflows", submittedCount)
	t.Logf("Errors: %d", errorCount)
	t.Logf("Total: %d", total)
	t.Logf("Elapsed: %v", elapsed)
	t.Logf("Actual Rate: %.2f workflows/sec", actualRate)
	t.Logf("Error Rate: %.2f%%", errorRate)

	// Validate AC3
	if actualRate < float64(targetRate) {
		t.Errorf("❌ Throughput %.2f/sec below target %d/sec", actualRate, targetRate)
	} else {
		t.Logf("✅ Throughput target met (%.2f >= %d/sec)", actualRate, targetRate)
	}

	if errorRate > 1.0 {
		t.Errorf("❌ Error rate %.2f%% above threshold 1%%", errorRate)
	} else {
		t.Logf("✅ Error rate acceptable (%.2f%% < 1%%)", errorRate)
	}
}

// BenchmarkWorkflowSubmit benchmarks single workflow submission
func BenchmarkWorkflowSubmit(b *testing.B) {
	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	client, err := sdk.NewClient(&sdk.ClientConfig{
		ServerURL: serverURL,
	})
	if err != nil {
		b.Fatal(err)
	}

	workflowYAML, err := os.ReadFile("../../examples/hello-world.yaml")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.SubmitWorkflow(context.Background(), &sdk.SubmitWorkflowRequest{
			YAML: string(workflowYAML),
		})
		if err != nil {
			b.Logf("Error on iteration %d: %v", i, err)
			// Continue instead of failing to measure error rate
		}
	}
}

// BenchmarkConcurrentSubmit benchmarks concurrent workflow submission
func BenchmarkConcurrentSubmit(b *testing.B) {
	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	client, err := sdk.NewClient(&sdk.ClientConfig{
		ServerURL: serverURL,
	})
	if err != nil {
		b.Fatal(err)
	}

	workflowYAML, err := os.ReadFile("../../examples/hello-world.yaml")
	if err != nil {
		b.Fatal(err)
	}

	concurrency := 10

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.SubmitWorkflow(context.Background(), &sdk.SubmitWorkflowRequest{
				YAML: string(workflowYAML),
			})
			if err != nil {
				b.Logf("Error: %v", err)
			}
		}
	})

	b.Logf("Concurrency: %d", concurrency)
}

// TestAPILatency measures API response times
// AC1: P99 < 500ms
func TestAPILatency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping latency test in short mode")
	}

	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	client, err := sdk.NewClient(&sdk.ClientConfig{
		ServerURL: serverURL,
	})
	if err != nil {
		t.Fatal(err)
	}

	workflowYAML, err := os.ReadFile("../../examples/hello-world.yaml")
	if err != nil {
		t.Fatal(err)
	}

	// Measure submission latency
	samples := 100
	latencies := make([]time.Duration, samples)

	for i := 0; i < samples; i++ {
		start := time.Now()
		_, err := client.SubmitWorkflow(context.Background(), &sdk.SubmitWorkflowRequest{
			YAML: string(workflowYAML),
		})
		latencies[i] = time.Since(start)

		if err != nil {
			t.Logf("Submission error on iteration %d: %v (continuing)", i, err)
			// Don't fail - just log errors
		}
	}

	// Calculate percentiles
	p50, p95, p99 := calculatePercentiles(latencies)

	t.Logf("=== Workflow Submission API Latency ===")
	t.Logf("Samples: %d", samples)
	t.Logf("P50: %v", p50)
	t.Logf("P95: %v", p95)
	t.Logf("P99: %v", p99)

	// Validate AC1
	if p99 > 500*time.Millisecond {
		t.Errorf("❌ P99 latency %v exceeds 500ms target", p99)
	} else {
		t.Logf("✅ P99 latency target met (%v < 500ms)", p99)
	}

	if p50 > 200*time.Millisecond {
		t.Logf("⚠️  P50 latency %v exceeds 200ms target", p50)
	} else {
		t.Logf("✅ P50 latency target met (%v < 200ms)", p50)
	}
}

// calculatePercentiles calculates P50, P95, P99 from latency samples
func calculatePercentiles(latencies []time.Duration) (p50, p95, p99 time.Duration) {
	if len(latencies) == 0 {
		return 0, 0, 0
	}

	// Sort latencies
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Calculate indices
	idx50 := int(float64(len(sorted)) * 0.50)
	idx95 := int(float64(len(sorted)) * 0.95)
	idx99 := int(float64(len(sorted)) * 0.99)

	if idx50 >= len(sorted) {
		idx50 = len(sorted) - 1
	}
	if idx95 >= len(sorted) {
		idx95 = len(sorted) - 1
	}
	if idx99 >= len(sorted) {
		idx99 = len(sorted) - 1
	}

	return sorted[idx50], sorted[idx95], sorted[idx99]
}
