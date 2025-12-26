package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/internal/api"
	"github.com/Websoft9/waterflow/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestAgentRegistration_Integration tests the end-to-end agent registration flow.
func TestAgentRegistration_Integration(t *testing.T) {
	// Create logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create in-memory provider
	sgProvider := provider.NewInMemoryProvider()

	// Create router with agent handler
	router := api.NewRouter(logger, nil, sgProvider, "test", "test", "test")

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Simulate agent registration
	reqBody := map[string]interface{}{
		"agent_id":    "test-agent-001",
		"hostname":    "integration-test-server",
		"ip_address":  "192.168.1.100",
		"task_queues": []string{"linux-amd64", "integration-test"},
		"metadata": map[string]string{
			"os":   "linux",
			"arch": "amd64",
		},
	}

	jsonData, _ := json.Marshal(reqBody)
	resp, err := http.Post(
		server.URL+"/v1/agents/register",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify registration succeeded
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify agent is in provider
	ctx := context.Background()
	servers, err := sgProvider.GetServers(ctx, "linux-amd64")
	require.NoError(t, err)
	require.Len(t, servers, 1)

	assert.Equal(t, "test-agent-001", servers[0].AgentID)
	assert.Equal(t, "integration-test-server", servers[0].Hostname)
	assert.Equal(t, "192.168.1.100", servers[0].IPAddress)
	assert.Contains(t, servers[0].TaskQueues, "linux-amd64")
	assert.Equal(t, "healthy", servers[0].Status)
	assert.WithinDuration(t, time.Now(), servers[0].LastHeartbeat, 5*time.Second)
}

// TestMultipleAgentRegistration tests multiple agents registering to different queues.
func TestMultipleAgentRegistration(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	sgProvider := provider.NewInMemoryProvider()
	router := api.NewRouter(logger, nil, sgProvider, "test", "test", "test")
	server := httptest.NewServer(router)
	defer server.Close()

	// Register multiple agents
	agents := []struct {
		id     string
		queues []string
	}{
		{"agent-001", []string{"linux-amd64"}},
		{"agent-002", []string{"linux-arm64"}},
		{"agent-003", []string{"linux-amd64", "linux-arm64"}},
	}

	for _, agent := range agents {
		reqBody := map[string]interface{}{
			"agent_id":    agent.id,
			"hostname":    "test-server",
			"task_queues": agent.queues,
		}
		jsonData, _ := json.Marshal(reqBody)
		resp, err := http.Post(
			server.URL+"/v1/agents/register",
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	}

	// Verify groups
	ctx := context.Background()
	groups, err := sgProvider.ListGroups(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"linux-amd64", "linux-arm64"}, groups)

	// Verify linux-amd64 has 2 agents
	servers, err := sgProvider.GetServers(ctx, "linux-amd64")
	require.NoError(t, err)
	assert.Len(t, servers, 2)
}
