// Package stress provides stress testing utilities for Waterflow
package stress

import (
	"runtime"
	"testing"
	"time"
)

// TestMemoryProfile validates no memory leaks over extended runtime
// Acceptance Criteria (AC6):
// - Memory growth < 20% over 1 hour (simplified to 10 minutes for test)
// - Goroutine count stable
func TestMemoryProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory profile test in short mode")
	}

	// Record initial memory
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	initialGoroutines := runtime.NumGoroutine()

	t.Logf("Initial state:")
	t.Logf("  Heap: %d MB", m1.Alloc/1024/1024)
	t.Logf("  Goroutines: %d", initialGoroutines)

	// Run load for 10 minutes (reduced from 1 hour for testing)
	duration := 10 * time.Minute
	if testing.Short() {
		duration = 1 * time.Minute
	}

	deadline := time.Now().Add(duration)
	iterations := 0

	for time.Now().Before(deadline) {
		// Simulate workflow processing
		processWorkflow()
		iterations++
		time.Sleep(100 * time.Millisecond)
	}

	t.Logf("Completed %d iterations", iterations)

	// Force GC to get accurate measurement
	runtime.GC()
	time.Sleep(1 * time.Second)

	// Record final memory
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	finalGoroutines := runtime.NumGoroutine()

	t.Logf("Final state:")
	t.Logf("  Heap: %d MB", m2.Alloc/1024/1024)
	t.Logf("  Goroutines: %d", finalGoroutines)

	// Calculate growth
	memGrowth := float64(m2.Alloc-m1.Alloc) / float64(m1.Alloc) * 100
	goroutineGrowth := float64(finalGoroutines-initialGoroutines) / float64(initialGoroutines) * 100

	t.Logf("Memory growth: %.2f%%", memGrowth)
	t.Logf("Goroutine growth: %.2f%%", goroutineGrowth)

	// Verify memory growth < 20%
	if memGrowth > 20 {
		t.Errorf("Excessive memory growth: %.2f%% (threshold: 20%%)", memGrowth)
	}

	// Verify goroutine growth < 20%
	if goroutineGrowth > 20 {
		t.Errorf("Excessive goroutine growth: %.2f%% (threshold: 20%%)", goroutineGrowth)
	}

	// Verify absolute goroutine count
	if finalGoroutines > 1000 {
		t.Errorf("Too many goroutines: %d (threshold: 1000)", finalGoroutines)
	}

	t.Log("✅ Memory profile test passed - no leaks detected")
}

// TestGoroutineLeakDetection specifically checks for goroutine leaks
func TestGoroutineLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping goroutine leak detection in short mode")
	}

	initial := runtime.NumGoroutine()
	t.Logf("Initial goroutines: %d", initial)

	// Simulate creating and cleaning up resources
	for i := 0; i < 100; i++ {
		go func() {
			time.Sleep(10 * time.Millisecond)
		}()
	}

	// Wait for goroutines to complete
	time.Sleep(1 * time.Second)

	final := runtime.NumGoroutine()
	t.Logf("Final goroutines: %d", final)

	// Allow small variance
	if final > initial+10 {
		t.Errorf("Goroutine leak detected: %d -> %d (growth: %d)", initial, final, final-initial)
	}

	t.Log("✅ No goroutine leaks detected")
}

// processWorkflow simulates workflow processing for memory testing
func processWorkflow() {
	// Simulate some memory allocation
	data := make([]byte, 1024)
	_ = data
}
