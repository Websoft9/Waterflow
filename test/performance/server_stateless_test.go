//go:build performance
// +build performance

package performance

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/sdk"
)

// TestServerStatelessPerformance validates Server performance is consistent across restarts
// PRD AC8: 验证 Server 无状态不影响性能
// Story: 7.3-PERF-008
func TestServerStatelessPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping server stateless performance test in short mode")
	}

	serverBinary := os.Getenv("SERVER_BINARY")
	if serverBinary == "" {
		serverBinary = "./bin/server"
	}

	serverURL := "http://localhost:8080"

	// Phase 1: Measure baseline performance
	t.Log("=== Phase 1: Baseline Performance ===")

	if !isServerHealthy(serverURL) {
		t.Fatal("Server is not running. Please start server before running this test.")
	}

	baselineP99, baselineThroughput := measureServerPerformance(t, serverURL, "baseline")

	// Phase 2: Restart server
	t.Log("=== Phase 2: Restarting Server ===")

	if err := restartServer(t, serverBinary); err != nil {
		t.Fatalf("Failed to restart server: %v", err)
	}

	time.Sleep(5 * time.Second) // Wait for server to stabilize

	// Phase 3: Measure post-restart performance
	t.Log("=== Phase 3: Post-Restart Performance ===")

	if !isServerHealthy(serverURL) {
		t.Fatal("Server not healthy after restart")
	}

	postRestartP99, postRestartThroughput := measureServerPerformance(t, serverURL, "post-restart")

	// Phase 4: Compare performance
	t.Log("=== Phase 4: Performance Comparison ===")

	latencyChange := ((postRestartP99 - baselineP99) / baselineP99) * 100
	throughputChange := ((postRestartThroughput - baselineThroughput) / baselineThroughput) * 100

	t.Logf("Baseline P99: %v", baselineP99)
	t.Logf("Post-Restart P99: %v", postRestartP99)
	t.Logf("Latency Change: %.2f%%", latencyChange)
	t.Logf("")
	t.Logf("Baseline Throughput: %.2f req/s", baselineThroughput)
	t.Logf("Post-Restart Throughput: %.2f req/s", postRestartThroughput)
	t.Logf("Throughput Change: %.2f%%", throughputChange)

	// Validate AC8: Performance degradation < 10%
	threshold := 10.0

	if latencyChange > threshold {
		t.Errorf("❌ Latency degradation %.2f%% exceeds %v%% threshold", latencyChange, threshold)
	} else {
		t.Logf("✅ Latency consistent after restart (%.2f%% change)", latencyChange)
	}

	if throughputChange < -threshold {
		t.Errorf("❌ Throughput degradation %.2f%% exceeds %v%% threshold", throughputChange, threshold)
	} else {
		t.Logf("✅ Throughput consistent after restart (%.2f%% change)", throughputChange)
	}
}

// measureServerPerformance measures P99 latency and throughput
func measureServerPerformance(t *testing.T, serverURL, phase string) (time.Duration, float64) {
	client, err := sdk.NewClient(&sdk.ClientConfig{
		ServerURL: serverURL,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	workflowYAML, err := os.ReadFile("../../examples/hello-world.yaml")
	if err != nil {
		t.Fatalf("Failed to read workflow: %v", err)
	}

	// Measure latency
	samples := 50
	latencies := make([]time.Duration, samples)

	start := time.Now()

	for i := 0; i < samples; i++ {
		reqStart := time.Now()

		_, err := client.SubmitWorkflow(context.Background(), &sdk.SubmitWorkflowRequest{
			YAML: string(workflowYAML),
		})

		latency := time.Since(reqStart)
		latencies[i] = latency

		if err != nil {
			t.Logf("[%s] Request %d failed: %v", phase, i, err)
		}

		time.Sleep(20 * time.Millisecond) // Throttle
	}

	elapsed := time.Since(start)

	// Calculate metrics
	_, _, p99 := calculatePercentiles(latencies)
	throughput := float64(samples) / elapsed.Seconds()

	t.Logf("[%s] P99: %v, Throughput: %.2f req/s", phase, p99, throughput)

	return p99, throughput
}

// restartServer restarts the server process
func restartServer(t *testing.T, serverBinary string) error {
	t.Log("Killing existing server...")

	// Kill existing server
	if err := exec.Command("pkill", "-f", serverBinary).Run(); err != nil {
		t.Logf("pkill returned error (may be normal): %v", err)
	}

	time.Sleep(3 * time.Second)

	t.Log("Starting new server instance...")

	// Start new server
	cmd := exec.Command(serverBinary)
	cmd.Env = append(os.Environ(),
		"WATERFLOW_TEMPORAL_HOST=localhost:7233",
		"WATERFLOW_LOG_LEVEL=warn",
	)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	t.Logf("Server started with PID: %d", cmd.Process.Pid)

	// Wait for health
	maxWait := 15 * time.Second
	checkInterval := 500 * time.Millisecond
	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		if isServerHealthy("http://localhost:8080") {
			t.Log("Server is healthy")
			return nil
		}

		time.Sleep(checkInterval)
	}

	return fmt.Errorf("server did not become healthy within %v", maxWait)
}

// isServerHealthy checks if server health endpoint responds
func isServerHealthy(serverURL string) bool {
	resp, err := http.Get(serverURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}
