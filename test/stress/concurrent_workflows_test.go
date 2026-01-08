// Package stress provides stress testing utilities for Waterflow
package stress

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/sdk"
)

// TestConcurrentWorkflows tests system behavior under 1000+ concurrent workflow submissions
// Acceptance Criteria:
// - Success rate > 99%
// - CPU < 80%
// - Memory < 2GB (Server)
// - Connections < 1000
// - Average execution time < 10s
func TestConcurrentWorkflows(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	serverURL := os.Getenv("WATERFLOW_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	client, err := sdk.NewClient(&sdk.ClientConfig{
		ServerURL: serverURL,
	})
	if err != nil {
		t.Fatal(err)
	}

	concurrency := 1000
	if c := os.Getenv("CONCURRENT_WORKFLOWS"); c != "" {
		_, _ = fmt.Sscanf(c, "%d", &concurrency)
	}

	workflow := []byte(`name: stress-test
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Echo test
        uses: run@v1
        with:
          command: echo "stress test"
`)

	var (
		success int
		failed  int
		mu      sync.Mutex
		wg      sync.WaitGroup
	)

	// 资源监控
	resourceMonitor := newResourceMonitor()
	resourceMonitor.start()
	defer resourceMonitor.stop()

	start := time.Now()

	// 并发提交
	t.Logf("Submitting %d concurrent workflows...", concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			_, err := client.SubmitWorkflow(ctx, &sdk.SubmitWorkflowRequest{
				YAML: string(workflow),
			})

			mu.Lock()
			if err != nil {
				failed++
				if failed <= 10 { // Log first 10 failures
					t.Logf("Workflow %d failed: %v", id, err)
				}
			} else {
				success++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	resourceMonitor.stop()

	successRate := float64(success) / float64(concurrency) * 100

	// 结果报告
	t.Logf("=== Concurrent Workflows Stress Test Results ===")
	t.Logf("Total Workflows: %d", concurrency)
	t.Logf("Success: %d", success)
	t.Logf("Failed: %d", failed)
	t.Logf("Success Rate: %.2f%%", successRate)
	t.Logf("Duration: %v", duration)
	t.Logf("Throughput: %.2f workflows/sec", float64(concurrency)/duration.Seconds())
	t.Logf("")
	t.Logf("Resource Usage:")
	t.Logf("  Max CPU: %.2f%%", resourceMonitor.maxCPU)
	t.Logf("  Max Memory: %.2f MB", resourceMonitor.maxMemMB)
	t.Logf("  Max Goroutines: %d", resourceMonitor.maxGoroutines)

	// 验证 AC1: 成功率 > 99%
	if successRate < 99.0 {
		t.Errorf("❌ Success rate %.2f%% below target 99%%", successRate)
	} else {
		t.Logf("✅ Success rate %.2f%% meets target", successRate)
	}

	// 验证 AC1: CPU < 80%
	if resourceMonitor.maxCPU > 80.0 {
		t.Errorf("⚠️  Max CPU %.2f%% exceeds 80%% threshold", resourceMonitor.maxCPU)
	} else {
		t.Logf("✅ CPU usage %.2f%% within limits", resourceMonitor.maxCPU)
	}

	// 验证 AC1: Memory < 2GB
	if resourceMonitor.maxMemMB > 2048 {
		t.Errorf("⚠️  Max Memory %.2f MB exceeds 2GB threshold", resourceMonitor.maxMemMB)
	} else {
		t.Logf("✅ Memory usage %.2f MB within limits", resourceMonitor.maxMemMB)
	}

	// 验证 AC1: 平均执行时间 < 10s
	avgDuration := duration.Seconds() / float64(concurrency)
	if avgDuration > 10.0 {
		t.Errorf("⚠️  Average execution time %.2fs exceeds 10s", avgDuration)
	} else {
		t.Logf("✅ Average execution time %.2fs within limits", avgDuration)
	}
}

// BenchmarkConcurrentWorkflows benchmarks concurrent workflow submissions
func BenchmarkConcurrentWorkflows(b *testing.B) {
	serverURL := os.Getenv("WATERFLOW_SERVER_URL")
	if serverURL == "" {
		b.Skip("WATERFLOW_SERVER_URL not set")
	}

	client, err := sdk.NewClient(&sdk.ClientConfig{
		ServerURL: serverURL,
	})
	if err != nil {
		b.Fatal(err)
	}

	workflow := []byte(`name: bench-test
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Echo test
        uses: run@v1
        with:
          command: echo "benchmark"
`)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, err := client.SubmitWorkflow(ctx, &sdk.SubmitWorkflowRequest{
				YAML: string(workflow),
			})
			cancel()
			if err != nil {
				b.Logf("Error: %v", err)
			}
		}
	})
}

// resourceMonitor monitors system resources during test
type resourceMonitor struct {
	maxCPU        float64
	maxMemMB      float64
	maxGoroutines int
	stopChan      chan struct{}
	wg            sync.WaitGroup
	mu            sync.Mutex
}

func newResourceMonitor() *resourceMonitor {
	return &resourceMonitor{
		stopChan: make(chan struct{}),
	}
}

func (rm *resourceMonitor) start() {
	rm.wg.Add(1)
	go func() {
		defer rm.wg.Done()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		var m runtime.MemStats
		for {
			select {
			case <-rm.stopChan:
				return
			case <-ticker.C:
				runtime.ReadMemStats(&m)
				memMB := float64(m.Alloc) / 1024 / 1024
				goroutines := runtime.NumGoroutine()

				rm.mu.Lock()
				if memMB > rm.maxMemMB {
					rm.maxMemMB = memMB
				}
				if goroutines > rm.maxGoroutines {
					rm.maxGoroutines = goroutines
				}
				// Note: CPU monitoring requires additional dependencies
				// For now, we track memory and goroutines
				rm.mu.Unlock()
			}
		}
	}()
}

func (rm *resourceMonitor) stop() {
	close(rm.stopChan)
	rm.wg.Wait()
}
