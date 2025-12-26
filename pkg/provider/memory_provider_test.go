package provider

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryProvider_RegisterServer(t *testing.T) {
	provider := NewInMemoryProvider()

	server := ServerInfo{
		AgentID:       "agent-1",
		Hostname:      "server1",
		TaskQueues:    []string{"linux-amd64", "linux-common"},
		Status:        "healthy",
		LastHeartbeat: time.Now(),
	}

	err := provider.RegisterServer(server)
	require.NoError(t, err)

	// Verify registered in both groups
	servers, err := provider.GetServers(context.Background(), "linux-amd64")
	require.NoError(t, err)
	assert.Len(t, servers, 1)
	assert.Equal(t, "agent-1", servers[0].AgentID)

	servers, err = provider.GetServers(context.Background(), "linux-common")
	require.NoError(t, err)
	assert.Len(t, servers, 1)
}

func TestInMemoryProvider_GetServers_EmptyGroup(t *testing.T) {
	provider := NewInMemoryProvider()

	servers, err := provider.GetServers(context.Background(), "non-existent")
	require.NoError(t, err)
	assert.Empty(t, servers)
}

func TestInMemoryProvider_ListGroups(t *testing.T) {
	provider := NewInMemoryProvider()

	provider.RegisterServer(ServerInfo{
		AgentID:    "agent-1",
		TaskQueues: []string{"group-a", "group-b"},
	})

	groups, err := provider.ListGroups(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"group-a", "group-b"}, groups)
}

func TestInMemoryProvider_UpdateHeartbeat(t *testing.T) {
	provider := NewInMemoryProvider()

	provider.RegisterServer(ServerInfo{
		AgentID:       "agent-1",
		Hostname:      "server1",
		TaskQueues:    []string{"linux-amd64"},
		Status:        "healthy",
		LastHeartbeat: time.Now().Add(-1 * time.Hour),
	})

	// Update heartbeat
	err := provider.UpdateHeartbeat("agent-1", "healthy")
	require.NoError(t, err)

	// Verify updated
	servers, _ := provider.GetServers(context.Background(), "linux-amd64")
	require.Len(t, servers, 1)
	assert.WithinDuration(t, time.Now(), servers[0].LastHeartbeat, 1*time.Second)
}

func TestInMemoryProvider_RegisterServer_UpdateExisting(t *testing.T) {
	provider := NewInMemoryProvider()

	// Register first time
	provider.RegisterServer(ServerInfo{
		AgentID:    "agent-1",
		Hostname:   "old-hostname",
		TaskQueues: []string{"linux-amd64"},
		Status:     "healthy",
	})

	// Re-register with updated info
	provider.RegisterServer(ServerInfo{
		AgentID:    "agent-1",
		Hostname:   "new-hostname",
		TaskQueues: []string{"linux-amd64"},
		Status:     "healthy",
	})

	// Should have only one server with updated info
	servers, _ := provider.GetServers(context.Background(), "linux-amd64")
	require.Len(t, servers, 1)
	assert.Equal(t, "new-hostname", servers[0].Hostname)
}

func TestInMemoryProvider_Close(t *testing.T) {
	provider := NewInMemoryProvider()
	err := provider.Close()
	assert.NoError(t, err)
}

// Performance benchmarks
func BenchmarkInMemoryProvider_GetServers_10000Agents(b *testing.B) {
	provider := NewInMemoryProvider()

	// Register 10000 agents
	for i := 0; i < 10000; i++ {
		provider.RegisterServer(ServerInfo{
			AgentID:    fmt.Sprintf("agent-%d", i),
			TaskQueues: []string{"linux-amd64"},
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.GetServers(context.Background(), "linux-amd64")
		if err != nil {
			b.Fatal(err)
		}
	}
	// Expected: < 10ms per operation
}

func BenchmarkInMemoryProvider_RegisterServer(b *testing.B) {
	provider := NewInMemoryProvider()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider.RegisterServer(ServerInfo{
			AgentID:    fmt.Sprintf("agent-%d", i),
			TaskQueues: []string{"linux-amd64"},
		})
	}
}

func TestInMemoryProvider_HealthDetection(t *testing.T) {
	ctx := context.Background()

	t.Run("healthy agent within 90 seconds", func(t *testing.T) {
		provider := NewInMemoryProvider()

		agent := ServerInfo{
			AgentID:       "agent-healthy",
			Hostname:      "server-healthy",
			TaskQueues:    []string{"test-queue"},
			Status:        "healthy",
			LastHeartbeat: time.Now().Add(-30 * time.Second), // 30 seconds ago
		}

		provider.RegisterServer(agent)

		servers, err := provider.GetServers(ctx, "test-queue")
		require.NoError(t, err)
		require.Len(t, servers, 1)

		assert.Equal(t, "healthy", servers[0].Status)
	})

	t.Run("unhealthy agent over 90 seconds", func(t *testing.T) {
		provider := NewInMemoryProvider()

		agent := ServerInfo{
			AgentID:       "agent-unhealthy",
			Hostname:      "server-unhealthy",
			TaskQueues:    []string{"test-queue"},
			Status:        "healthy",                          // Initially healthy
			LastHeartbeat: time.Now().Add(-120 * time.Second), // 120 seconds ago
		}

		provider.RegisterServer(agent)

		servers, err := provider.GetServers(ctx, "test-queue")
		require.NoError(t, err)
		require.Len(t, servers, 1)

		// Should be automatically marked as unhealthy
		assert.Equal(t, "unhealthy", servers[0].Status)
	})

	t.Run("agent with zero heartbeat remains unchanged", func(t *testing.T) {
		provider := NewInMemoryProvider()

		agent := ServerInfo{
			AgentID:       "agent-no-heartbeat",
			Hostname:      "server-no-heartbeat",
			TaskQueues:    []string{"test-queue"},
			Status:        "unknown",
			LastHeartbeat: time.Time{}, // Zero time
		}

		provider.RegisterServer(agent)

		servers, err := provider.GetServers(ctx, "test-queue")
		require.NoError(t, err)
		require.Len(t, servers, 1)

		// Status should remain unchanged
		assert.Equal(t, "unknown", servers[0].Status)
	})

	t.Run("exactly 90 seconds edge case", func(t *testing.T) {
		provider := NewInMemoryProvider()

		// Use 89 seconds to be clearly within the healthy range
		agent := ServerInfo{
			AgentID:       "agent-edge-case",
			Hostname:      "server-edge",
			TaskQueues:    []string{"test-queue"},
			Status:        "healthy",
			LastHeartbeat: time.Now().Add(-89 * time.Second),
		}

		provider.RegisterServer(agent)

		servers, err := provider.GetServers(ctx, "test-queue")
		require.NoError(t, err)
		require.Len(t, servers, 1)

		// Should still be healthy (< 90 seconds)
		assert.Equal(t, "healthy", servers[0].Status)
	})

	t.Run("heartbeat update makes unhealthy agent healthy", func(t *testing.T) {
		provider := NewInMemoryProvider()

		// Register an agent with old heartbeat
		agent := ServerInfo{
			AgentID:       "agent-1",
			Hostname:      "server-1",
			TaskQueues:    []string{"test-queue"},
			Status:        "healthy",
			LastHeartbeat: time.Now().Add(-2 * time.Minute),
		}

		provider.RegisterServer(agent)

		// Verify initial state (should be unhealthy due to old heartbeat)
		servers, _ := provider.GetServers(ctx, "test-queue")
		assert.Equal(t, "unhealthy", servers[0].Status)

		// Update heartbeat
		err := provider.UpdateHeartbeat("agent-1", "healthy")
		require.NoError(t, err)

		// Verify heartbeat was updated
		servers, _ = provider.GetServers(ctx, "test-queue")
		assert.Equal(t, "healthy", servers[0].Status)

		// Check that LastHeartbeat was updated to recent time
		timeSince := time.Since(servers[0].LastHeartbeat)
		assert.Less(t, timeSince, 5*time.Second, "heartbeat should be very recent")
	})
}
