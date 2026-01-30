//go:build performance
// +build performance

package performance

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestServerStartupTime validates Server startup performance
// PRD AC9: Server 启动时间 < 5 秒
// Story: 7.3-PERF-009
func TestServerStartupTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping server startup test in short mode")
	}

	serverBinary := os.Getenv("SERVER_BINARY")
	if serverBinary == "" {
		serverBinary = "./bin/server"
	}

	// Verify binary exists
	if _, err := os.Stat(serverBinary); os.IsNotExist(err) {
		t.Fatalf("Server binary not found at %s. Build it first: make build-server", serverBinary)
	}

	t.Logf("Testing startup time for: %s", serverBinary)

	// Kill any existing server instances
	exec.Command("pkill", "-f", serverBinary).Run()
	time.Sleep(2 * time.Second)

	// Start server and measure time
	start := time.Now()

	cmd := exec.Command(serverBinary)
	cmd.Env = append(os.Environ(),
		"WATERFLOW_TEMPORAL_HOST=localhost:7233",
		"WATERFLOW_TEMPORAL_NAMESPACE=default",
		"WATERFLOW_LOG_LEVEL=warn",
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	serverPID := cmd.Process.Pid
	t.Logf("Server started with PID: %d", serverPID)

	// Ensure cleanup
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	// Poll health endpoint
	healthURL := "http://localhost:8080/health"
	maxWait := 15 * time.Second
	checkInterval := 100 * time.Millisecond

	var startupTime time.Duration
	ready := false

	ctx, cancel := context.WithTimeout(context.Background(), maxWait)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("❌ Server failed to become healthy within %v", maxWait)

		case <-time.After(checkInterval):
			resp, err := http.Get(healthURL)
			if err == nil && resp.StatusCode == 200 {
				resp.Body.Close()
				startupTime = time.Since(start)
				ready = true

				t.Logf("✅ Server became healthy")
				goto HealthCheckComplete
			}

			if err != nil {
				t.Logf("Health check attempt failed: %v (retrying...)", err)
			} else {
				resp.Body.Close()
			}
		}
	}

HealthCheckComplete:
	if !ready {
		t.Fatal("Server health check failed")
	}

	t.Logf("=== Server Startup Performance ===")
	t.Logf("Startup Time: %v", startupTime)
	t.Logf("PID: %d", serverPID)

	// Validate AC9
	threshold := 5 * time.Second
	if startupTime > threshold {
		t.Errorf("❌ Startup time %v exceeds %v threshold", startupTime, threshold)
	} else {
		t.Logf("✅ Startup time within limit (%v < %v)", startupTime, threshold)
	}

	// Additional validation: verify server is fully functional
	t.Log("Verifying server functionality...")

	// Test health endpoint
	resp, err := http.Get(healthURL)
	if err != nil || resp.StatusCode != 200 {
		t.Errorf("Server health endpoint not responding correctly")
	} else {
		resp.Body.Close()
		t.Logf("✅ Health endpoint responding")
	}

	// Test metrics endpoint (optional)
	metricsResp, err := http.Get("http://localhost:8080/metrics")
	if err == nil {
		metricsResp.Body.Close()
		t.Logf("✅ Metrics endpoint available")
	}
}

// BenchmarkServerStartup benchmarks server startup time
func BenchmarkServerStartup(b *testing.B) {
	serverBinary := os.Getenv("SERVER_BINARY")
	if serverBinary == "" {
		serverBinary = "./bin/server"
	}

	for i := 0; i < b.N; i++ {
		// Kill existing instances
		exec.Command("pkill", "-f", serverBinary).Run()
		time.Sleep(1 * time.Second)

		start := time.Now()

		cmd := exec.Command(serverBinary)
		cmd.Env = append(os.Environ(), "WATERFLOW_LOG_LEVEL=error")

		if err := cmd.Start(); err != nil {
			b.Fatalf("Failed to start server: %v", err)
		}

		// Wait for health
		ready := false
		timeout := time.After(10 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)

	WaitLoop:
		for {
			select {
			case <-timeout:
				b.Fatal("Server startup timeout")
			case <-ticker.C:
				resp, err := http.Get("http://localhost:8080/health")
				if err == nil && resp.StatusCode == 200 {
					resp.Body.Close()
					ready = true
					break WaitLoop
				}
			}
		}

		ticker.Stop()

		if ready {
			elapsed := time.Since(start)
			b.Logf("Startup %d: %v", i+1, elapsed)
		}

		// Cleanup
		cmd.Process.Kill()
		cmd.Wait()
		time.Sleep(1 * time.Second)
	}
}
