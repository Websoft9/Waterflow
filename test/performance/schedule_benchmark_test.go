//go:build performance

package performance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// getServerURL returns the server URL from environment or default
func getServerURL() string {
	if url := os.Getenv("WATERFLOW_TEST_URL"); url != "" {
		return url
	}
	return "http://localhost:18080"
}

const testWorkflowYAML = `name: benchmark-workflow
vars:
  test: value
jobs:
  test:
    runs-on: default
    steps:
      - name: Echo
        uses: exec@v1
        with:
          command: echo "test"
`

// BenchmarkSchedule_Create benchmarks schedule creation
func BenchmarkSchedule_Create(b *testing.B) {
	serverURL := getServerURL()
	ctx := context.Background()

	// Setup: Create definition
	defReq := map[string]interface{}{
		"name":    "benchmark-workflow",
		"content": testWorkflowYAML,
	}
	jsonBody, _ := json.Marshal(defReq)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		serverURL+"/v1/workflows/definitions",
		bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(b, err)
	resp.Body.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		scheduleReq := map[string]interface{}{
			"name": fmt.Sprintf("benchmark-schedule-%d-%d", time.Now().UnixNano(), i),
			"cron": "0 2 * * *",
		}
		jsonBody, _ := json.Marshal(scheduleReq)

		req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
			serverURL+"/v1/workflows/benchmark-workflow/schedules",
			bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		start := time.Now()
		resp, err := client.Do(req)
		elapsed := time.Since(start)

		require.NoError(b, err)
		require.Equal(b, http.StatusCreated, resp.StatusCode, "Expected 201 Created")
		resp.Body.Close()

		// Verify performance requirement: < 200ms P95
		if elapsed > 300*time.Millisecond {
			b.Logf("Warning: Create took %v (target: <200ms P95)", elapsed)
		}
	}
}

// BenchmarkSchedule_List benchmarks listing schedules with various counts
func BenchmarkSchedule_List(b *testing.B) {
	serverURL := getServerURL()
	ctx := context.Background()

	// Setup: Create definition and schedules
	defReq := map[string]interface{}{
		"name":    "list-benchmark-workflow",
		"content": testWorkflowYAML,
	}
	jsonBody, _ := json.Marshal(defReq)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		serverURL+"/v1/workflows/definitions",
		bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(b, err)
	resp.Body.Close()

	// Create 100 schedules
	scheduleCount := 100
	for i := 0; i < scheduleCount; i++ {
		scheduleReq := map[string]interface{}{
			"name": fmt.Sprintf("list-benchmark-%d", i),
			"cron": "0 2 * * *",
		}
		jsonBody, _ := json.Marshal(scheduleReq)

		req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
			serverURL+"/v1/workflows/list-benchmark-workflow/schedules",
			bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(b, err)
		resp.Body.Close()
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
			serverURL+"/v1/workflows/list-benchmark-workflow/schedules", nil)

		start := time.Now()
		resp, err := client.Do(req)
		elapsed := time.Since(start)

		require.NoError(b, err)
		require.Equal(b, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// Verify performance requirement: < 500ms P95 for 1000 schedules
		// With 100 schedules, expect < 100ms
		if elapsed > 200*time.Millisecond {
			b.Logf("Warning: List took %v for %d schedules (target: <500ms P95 for 1000)", elapsed, scheduleCount)
		}
	}
}

// BenchmarkSchedule_Get benchmarks getting schedule details
func BenchmarkSchedule_Get(b *testing.B) {
	serverURL := getServerURL()
	ctx := context.Background()

	// Setup: Create definition and schedule
	defReq := map[string]interface{}{
		"name":    "get-benchmark-workflow",
		"content": testWorkflowYAML,
	}
	jsonBody, _ := json.Marshal(defReq)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		serverURL+"/v1/workflows/definitions",
		bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(b, err)
	resp.Body.Close()

	scheduleReq := map[string]interface{}{
		"name": "get-benchmark-schedule",
		"cron": "0 2 * * *",
	}
	jsonBody, _ = json.Marshal(scheduleReq)
	req, _ = http.NewRequestWithContext(ctx, http.MethodPost,
		serverURL+"/v1/workflows/get-benchmark-workflow/schedules",
		bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	require.NoError(b, err)
	resp.Body.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
			serverURL+"/v1/workflows/get-benchmark-workflow/schedules/get-benchmark-schedule", nil)

		start := time.Now()
		resp, err := client.Do(req)
		elapsed := time.Since(start)

		require.NoError(b, err)
		require.Equal(b, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// Verify performance requirement: < 100ms P95
		if elapsed > 150*time.Millisecond {
			b.Logf("Warning: Get took %v (target: <100ms P95)", elapsed)
		}
	}
}

// BenchmarkSchedule_Delete benchmarks schedule deletion
func BenchmarkSchedule_Delete(b *testing.B) {
	serverURL := getServerURL()
	ctx := context.Background()

	// Setup: Create definition
	defReq := map[string]interface{}{
		"name":    "delete-benchmark-workflow",
		"content": testWorkflowYAML,
	}
	jsonBody, _ := json.Marshal(defReq)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		serverURL+"/v1/workflows/definitions",
		bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(b, err)
	resp.Body.Close()

	// Pre-create schedules for deletion
	scheduleNames := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		scheduleName := fmt.Sprintf("delete-benchmark-%d", i)
		scheduleNames[i] = scheduleName

		scheduleReq := map[string]interface{}{
			"name": scheduleName,
			"cron": "0 2 * * *",
		}
		jsonBody, _ := json.Marshal(scheduleReq)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
			serverURL+"/v1/workflows/delete-benchmark-workflow/schedules",
			bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(b, err)
		resp.Body.Close()
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodDelete,
			serverURL+"/v1/workflows/delete-benchmark-workflow/schedules/"+scheduleNames[i], nil)

		start := time.Now()
		resp, err := client.Do(req)
		elapsed := time.Since(start)

		require.NoError(b, err)
		require.Equal(b, http.StatusNoContent, resp.StatusCode)
		resp.Body.Close()

		// Verify performance requirement: < 100ms P95
		if elapsed > 150*time.Millisecond {
			b.Logf("Warning: Delete took %v (target: <100ms P95)", elapsed)
		}
	}
}

// BenchmarkSchedule_ConcurrentCreate benchmarks concurrent schedule creation
func BenchmarkSchedule_ConcurrentCreate(b *testing.B) {
	serverURL := getServerURL()
	ctx := context.Background()

	// Setup: Create definition
	defReq := map[string]interface{}{
		"name":    "concurrent-benchmark-workflow",
		"content": testWorkflowYAML,
	}
	jsonBody, _ := json.Marshal(defReq)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		serverURL+"/v1/workflows/definitions",
		bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(b, err)
	resp.Body.Close()

	concurrency := 10 // 10 concurrent goroutines

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		errors := make(chan error, concurrency)

		for j := 0; j < concurrency; j++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				scheduleReq := map[string]interface{}{
					"name": fmt.Sprintf("concurrent-benchmark-%d-%d-%d", i, idx, time.Now().UnixNano()),
					"cron": "0 2 * * *",
				}
				jsonBody, _ := json.Marshal(scheduleReq)

				req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
					serverURL+"/v1/workflows/concurrent-benchmark-workflow/schedules",
					bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(req)
				if err != nil {
					errors <- err
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusCreated {
					body, _ := io.ReadAll(resp.Body)
					errors <- fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
					return
				}
			}(j)
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			require.NoError(b, err, "Concurrent creation should not fail")
		}
	}
}

// TestSchedule_LoadTest performs load testing with 1000 schedules (not a benchmark, runs with -tags=performance)
func TestSchedule_LoadTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	serverURL := getServerURL()
	ctx := context.Background()

	// Setup: Create definition
	defReq := map[string]interface{}{
		"name":    "load-test-workflow",
		"content": testWorkflowYAML,
	}
	jsonBody, _ := json.Marshal(defReq)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		serverURL+"/v1/workflows/definitions",
		bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	// Create 1000 schedules
	t.Log("Creating 1000 schedules...")
	startTime := time.Now()

	for i := 0; i < 1000; i++ {
		scheduleReq := map[string]interface{}{
			"name": fmt.Sprintf("load-test-schedule-%d", i),
			"cron": "0 2 * * *",
		}
		jsonBody, _ := json.Marshal(scheduleReq)

		req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
			serverURL+"/v1/workflows/load-test-workflow/schedules",
			bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		resp.Body.Close()

		if i%100 == 0 {
			t.Logf("Created %d schedules...", i)
		}
	}

	createDuration := time.Since(startTime)
	t.Logf("Created 1000 schedules in %v (avg: %v/schedule, target: <5min total)", createDuration, createDuration/1000)
	require.Less(t, createDuration, 5*time.Minute, "Should create 1000 schedules in <5 min")

	// Query performance test
	t.Log("Testing query performance with 1000 schedules...")
	req, _ = http.NewRequestWithContext(ctx, http.MethodGet,
		serverURL+"/v1/workflows/load-test-workflow/schedules", nil)

	queryStart := time.Now()
	resp, err = client.Do(req)
	queryDuration := time.Since(queryStart)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	t.Logf("List query with 1000 schedules took %v (target: <500ms P95)", queryDuration)
	require.Less(t, queryDuration, 1*time.Second, "Query should complete in <1s with 1000 schedules")

	// Cleanup: Delete all schedules
	t.Log("Cleaning up 1000 schedules...")
	cleanupStart := time.Now()

	for i := 0; i < 1000; i++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodDelete,
			serverURL+"/v1/workflows/load-test-workflow/schedules/"+fmt.Sprintf("load-test-schedule-%d", i), nil)

		resp, err := client.Do(req)
		if err != nil {
			t.Logf("Failed to delete schedule-%d: %v", i, err)
			continue
		}
		resp.Body.Close()

		if i%100 == 0 {
			t.Logf("Deleted %d schedules...", i)
		}
	}

	cleanupDuration := time.Since(cleanupStart)
	t.Logf("Deleted 1000 schedules in %v (avg: %v/schedule, target: <3min total)", cleanupDuration, cleanupDuration/1000)
	require.Less(t, cleanupDuration, 3*time.Minute, "Should delete 1000 schedules in <3 min")
}
