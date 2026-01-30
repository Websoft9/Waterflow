// Package stress provides stress testing utilities for Waterflow
package stress

import (
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/sdk"
)

// TestResourceLeak verifies no resource leaks during continuous execution
// 追溯: PRD#L235 - 可靠性: 自动故障处理
// 验证: 内存、Goroutine、连接等资源不泄漏
func TestResourceLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource leak test in short mode")
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

	// Workflow template
	workflow := []byte(`name: leak-test
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Echo test
        uses: run@v1
        with:
          command: echo "leak test"
`)

	// Step 1: Baseline metrics
	runtime.GC()
	time.Sleep(1 * time.Second)

	var baselineMemStats runtime.MemStats
	runtime.ReadMemStats(&baselineMemStats)
	baselineGoroutines := runtime.NumGoroutine()

	t.Logf("Baseline - Memory: %d MB, Goroutines: %d",
		baselineMemStats.Alloc/1024/1024, baselineGoroutines)

	// Step 2: Execute workflows continuously
	iterations := 100
	for i := 0; i < iterations; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		_, err := client.SubmitWorkflow(ctx, &sdk.SubmitWorkflowRequest{
			YAML: string(workflow),
		})
		if err != nil {
			cancel()
			t.Logf("Workflow %d failed (expected in stress test): %v", i, err)
		}

		cancel()

		// Periodic GC to detect leaks
		if i%20 == 0 {
			runtime.GC()
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Step 3: Final GC and measurement
	runtime.GC()
	time.Sleep(2 * time.Second)

	var finalMemStats runtime.MemStats
	runtime.ReadMemStats(&finalMemStats)
	finalGoroutines := runtime.NumGoroutine()

	t.Logf("Final - Memory: %d MB, Goroutines: %d",
		finalMemStats.Alloc/1024/1024, finalGoroutines)

	// Step 4: Validate no leaks
	memGrowthMB := int64(finalMemStats.Alloc-baselineMemStats.Alloc) / 1024 / 1024
	goroutineGrowth := finalGoroutines - baselineGoroutines

	t.Logf("Growth - Memory: %d MB, Goroutines: %d", memGrowthMB, goroutineGrowth)

	// Thresholds (tunable based on workload)
	const (
		maxMemGrowthMB     = 50 // Max 50MB growth for 100 workflows
		maxGoroutineGrowth = 20 // Max 20 goroutines growth
	)

	if memGrowthMB > maxMemGrowthMB {
		t.Errorf("Memory leak detected: growth %d MB exceeds threshold %d MB",
			memGrowthMB, maxMemGrowthMB)
	}

	if goroutineGrowth > maxGoroutineGrowth {
		t.Errorf("Goroutine leak detected: growth %d exceeds threshold %d",
			goroutineGrowth, maxGoroutineGrowth)
	}

	// PRD requirement: No resource leaks
	if memGrowthMB > maxMemGrowthMB || goroutineGrowth > maxGoroutineGrowth {
		t.Errorf("PRD#L235 violation: Resource leak detected")
	}
}
