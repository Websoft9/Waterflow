// Package perfutil provides performance testing utilities for Waterflow.
package perfutil

import (
	"runtime"
	"testing"
	"time"
)

// MemStats captures memory statistics at a point in time.
type MemStats struct {
	AllocMB      uint64
	TotalAllocMB uint64
	NumGC        uint32
	Timestamp    time.Time
}

// CaptureMemoryBaseline captures the current memory state as a baseline.
// 追溯: 支持 test/stress 和 test/performance 测试
func CaptureMemoryBaseline(t *testing.T) *MemStats {
	t.Helper()

	// Force GC to get accurate baseline
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &MemStats{
		AllocMB:      m.Alloc / 1024 / 1024,
		TotalAllocMB: m.TotalAlloc / 1024 / 1024,
		NumGC:        m.NumGC,
		Timestamp:    time.Now(),
	}
}

// GetCurrentMemStats returns current memory statistics.
func GetCurrentMemStats(t *testing.T) *MemStats {
	t.Helper()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &MemStats{
		AllocMB:      m.Alloc / 1024 / 1024,
		TotalAllocMB: m.TotalAlloc / 1024 / 1024,
		NumGC:        m.NumGC,
		Timestamp:    time.Now(),
	}
}

// MemoryGrowthMB calculates memory growth between two snapshots.
func MemoryGrowthMB(baseline, current *MemStats) int64 {
	return int64(current.AllocMB - baseline.AllocMB)
}

// AssertNoMemoryLeak fails the test if memory growth exceeds threshold.
// 使用示例:
//
//	baseline := perfutil.CaptureMemoryBaseline(t)
//	// ... 运行测试 ...
//	perfutil.AssertNoMemoryLeak(t, baseline, 50) // Max 50MB growth
func AssertNoMemoryLeak(t *testing.T, baseline *MemStats, maxGrowthMB int64) {
	t.Helper()

	current := GetCurrentMemStats(t)
	growthMB := MemoryGrowthMB(baseline, current)

	if growthMB > maxGrowthMB {
		t.Errorf("Memory leak detected: growth %d MB exceeds threshold %d MB\n"+
			"  Baseline: %d MB at %v\n"+
			"  Current:  %d MB at %v\n"+
			"  Duration: %v",
			growthMB, maxGrowthMB,
			baseline.AllocMB, baseline.Timestamp.Format("15:04:05"),
			current.AllocMB, current.Timestamp.Format("15:04:05"),
			current.Timestamp.Sub(baseline.Timestamp))
	} else {
		t.Logf("Memory growth: %d MB (threshold: %d MB) ✓", growthMB, maxGrowthMB)
	}
}

// GoroutineTracker tracks goroutine count changes.
type GoroutineTracker struct {
	baseline  int
	startTime time.Time
}

// StartGoroutineTracking returns a new goroutine tracker.
func StartGoroutineTracking(t *testing.T) *GoroutineTracker {
	t.Helper()

	// Force GC to clean up stopped goroutines
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	return &GoroutineTracker{
		baseline:  runtime.NumGoroutine(),
		startTime: time.Now(),
	}
}

// AssertNoGoroutineLeak fails the test if goroutine count exceeds baseline + threshold.
func (gt *GoroutineTracker) AssertNoGoroutineLeak(t *testing.T, maxGrowth int) {
	t.Helper()

	// Force GC to clean up stopped goroutines
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	current := runtime.NumGoroutine()
	growth := current - gt.baseline

	if growth > maxGrowth {
		t.Errorf("Goroutine leak detected: growth %d exceeds threshold %d\n"+
			"  Baseline: %d at %v\n"+
			"  Current:  %d at %v\n"+
			"  Duration: %v",
			growth, maxGrowth,
			gt.baseline, gt.startTime.Format("15:04:05"),
			current, time.Now().Format("15:04:05"),
			time.Since(gt.startTime))
	} else {
		t.Logf("Goroutine growth: %d (threshold: %d) ✓", growth, maxGrowth)
	}
}

// BenchmarkConcurrent runs a benchmark with specified concurrency level.
// 使用示例:
//
//	perfutil.BenchmarkConcurrent(b, 100, func() {
//	    // 并发执行的操作
//	})
func BenchmarkConcurrent(b *testing.B, concurrency int, fn func()) {
	b.Helper()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			fn()
		}
	})
}

// MeasureDuration measures the duration of a function execution.
func MeasureDuration(t *testing.T, name string, fn func()) time.Duration {
	t.Helper()

	start := time.Now()
	fn()
	duration := time.Since(start)

	t.Logf("%s took %v", name, duration)
	return duration
}

// AssertDurationWithin fails if function duration exceeds timeout.
func AssertDurationWithin(t *testing.T, timeout time.Duration, fn func()) {
	t.Helper()

	start := time.Now()
	fn()
	duration := time.Since(start)

	if duration > timeout {
		t.Errorf("Operation took %v, exceeds timeout %v", duration, timeout)
	} else {
		t.Logf("Operation took %v (timeout: %v) ✓", duration, timeout)
	}
}
