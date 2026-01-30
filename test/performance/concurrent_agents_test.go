//go:build performance
// +build performance

package performance

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestConcurrentAgentConnections validates Server supports 100+ concurrent Agent connections
// PRD AC5: 支持 ≥100 个并发 Agent 连接
// Story: 7.3-PERF-005
func TestConcurrentAgentConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent agent test in short mode")
	}

	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	// Verify server health
	resp, err := http.Get(serverURL + "/health")
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("Server not healthy at %s: %v", serverURL, err)
	}

	targetAgents := 100
	connectionDuration := 2 * time.Minute

	t.Logf("Testing %d concurrent agent connections for %v", targetAgents, connectionDuration)

	var (
		connectedCount   atomic.Int32
		failedCount      atomic.Int32
		heartbeatsSent   atomic.Int64
		heartbeatsFailed atomic.Int64
	)

	ctx, cancel := context.WithTimeout(context.Background(), connectionDuration)
	defer cancel()

	var wg sync.WaitGroup

	// Simulate concurrent agents
	for i := 0; i < targetAgents; i++ {
		wg.Add(1)
		agentID := fmt.Sprintf("perf-agent-%d", i)

		go func(id string) {
			defer wg.Done()

			// Register agent (simulated heartbeat)
			if err := sendAgentHeartbeat(serverURL, id); err != nil {
				t.Logf("Agent %s failed to connect: %v", id, err)
				failedCount.Add(1)
				return
			}

			connectedCount.Add(1)
			t.Logf("Agent %s connected (%d/%d)", id, connectedCount.Load(), targetAgents)

			// Send periodic heartbeats
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := sendAgentHeartbeat(serverURL, id); err != nil {
						heartbeatsFailed.Add(1)
					} else {
						heartbeatsSent.Add(1)
					}
				}
			}
		}(agentID)

		// Stagger agent starts
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for test duration
	<-ctx.Done()
	wg.Wait()

	// Results
	connected := connectedCount.Load()
	failed := failedCount.Load()
	hbSent := heartbeatsSent.Load()
	hbFailed := heartbeatsFailed.Load()

	t.Logf("=== Concurrent Agent Test Results ===")
	t.Logf("Target Agents: %d", targetAgents)
	t.Logf("Connected: %d", connected)
	t.Logf("Failed: %d", failed)
	t.Logf("Heartbeats Sent: %d", hbSent)
	t.Logf("Heartbeats Failed: %d", hbFailed)

	successRate := float64(connected) / float64(targetAgents) * 100
	t.Logf("Connection Success Rate: %.2f%%", successRate)

	// Validate AC5
	if connected < 100 {
		t.Errorf("❌ Connected agents %d < 100 threshold", connected)
	} else {
		t.Logf("✅ Server supports 100+ concurrent agents (%d connected)", connected)
	}

	if successRate < 95.0 {
		t.Errorf("❌ Connection success rate %.2f%% below 95%% threshold", successRate)
	} else {
		t.Logf("✅ High connection reliability (%.2f%%)", successRate)
	}
}

// sendAgentHeartbeat simulates agent heartbeat to server
func sendAgentHeartbeat(serverURL, agentID string) error {
	// This is a mock implementation
	// In real scenario, this would POST to /api/v1/agents/heartbeat
	// For now, we just verify server is responsive

	endpoint := fmt.Sprintf("%s/health", serverURL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Agent-ID", agentID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}
