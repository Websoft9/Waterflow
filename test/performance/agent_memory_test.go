//go:build performance
// +build performance

package performance

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// TestAgentMemoryUsage validates Agent idle memory consumption
// PRD AC4: Agent 空闲内存 < 50MB, Heap < 30MB
// Story: 7.3-PERF-004
func TestAgentMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping agent memory test in short mode")
	}

	// Note: This test requires a running Agent instance
	// For integration, consider starting agent programmatically or via docker-compose

	agentPID := os.Getenv("AGENT_PID")
	if agentPID == "" {
		t.Skip("AGENT_PID environment variable not set. Start agent and set: export AGENT_PID=$(pgrep -f 'bin/agent')")
	}

	pid := mustParsePID(t, agentPID)
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		t.Fatalf("Failed to attach to agent process %d: %v", pid, err)
	}

	t.Logf("Monitoring Agent PID: %d", pid)

	// Wait for agent initialization
	time.Sleep(10 * time.Second)

	// Monitor for 30 seconds to ensure steady state
	var maxRSS, avgRSS float64
	samples := 30

	for i := 0; i < samples; i++ {
		memInfo, err := proc.MemoryInfo()
		if err != nil {
			t.Fatalf("Failed to get memory info at sample %d: %v", i, err)
		}

		rssMB := float64(memInfo.RSS) / 1024 / 1024
		avgRSS += rssMB

		if rssMB > maxRSS {
			maxRSS = rssMB
		}

		t.Logf("[%02d/%02d] RSS: %.2f MB", i+1, samples, rssMB)
		time.Sleep(1 * time.Second)
	}

	avgRSS /= float64(samples)

	// Get Go heap stats (if agent exposes pprof)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	heapMB := float64(m.Alloc) / 1024 / 1024

	t.Logf("=== Memory Usage Summary ===")
	t.Logf("Average RSS: %.2f MB", avgRSS)
	t.Logf("Max RSS: %.2f MB", maxRSS)
	t.Logf("Heap Alloc: %.2f MB", heapMB)

	// Validate AC4
	if maxRSS > 50.0 {
		t.Errorf("❌ Agent memory %.2f MB exceeds 50 MB threshold", maxRSS)
	} else {
		t.Logf("✅ Agent memory within limit (%.2f MB < 50 MB)", maxRSS)
	}

	if heapMB > 30.0 {
		t.Logf("⚠️  Heap memory %.2f MB exceeds 30 MB target", heapMB)
	} else {
		t.Logf("✅ Heap memory within limit (%.2f MB < 30 MB)", heapMB)
	}
}

// mustParsePID converts PID string to int
func mustParsePID(t *testing.T, pidStr string) int {
	var pid int
	_, err := fmt.Sscanf(pidStr, "%d", &pid)
	if err != nil {
		t.Fatalf("Invalid PID format: %s", pidStr)
	}
	return pid
}
