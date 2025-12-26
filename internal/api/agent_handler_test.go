package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Websoft9/waterflow/pkg/provider"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAgentHandler_RegisterAgent(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	memProvider := provider.NewInMemoryProvider()
	handler := NewAgentHandlers(logger, memProvider)

	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "valid registration",
			payload: map[string]interface{}{
				"agent_id":    "agent-001",
				"hostname":    "test-server.example.com",
				"ip_address":  "192.168.1.10",
				"task_queues": []string{"linux-amd64", "linux-common"},
				"metadata": map[string]string{
					"os":   "linux",
					"arch": "amd64",
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "Agent registered successfully", body["message"])
				assert.Equal(t, "agent-001", body["agent_id"])
			},
		},
		{
			name: "missing agent_id",
			payload: map[string]interface{}{
				"hostname":    "test-server.example.com",
				"task_queues": []string{"linux-amd64"},
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				errorObj := body["error"].(map[string]interface{})
				assert.Equal(t, "invalid_request", errorObj["code"])
			},
		},
		{
			name: "missing hostname",
			payload: map[string]interface{}{
				"agent_id":    "agent-001",
				"task_queues": []string{"linux-amd64"},
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				errorObj := body["error"].(map[string]interface{})
				assert.Equal(t, "invalid_request", errorObj["code"])
			},
		},
		{
			name: "empty task_queues",
			payload: map[string]interface{}{
				"agent_id":    "agent-001",
				"hostname":    "test-server.example.com",
				"task_queues": []string{},
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				errorObj := body["error"].(map[string]interface{})
				assert.Equal(t, "invalid_request", errorObj["code"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/v1/agents/register", bytes.NewReader(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.RegisterAgent(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestAgentHandler_RegisterAgent_FileProvider(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Create temporary YAML file
	tmpFile := t.TempDir() + "/servers.yaml"
	fileProvider, err := provider.NewFileProvider(tmpFile)
	if err != nil {
		// File doesn't exist, which is expected - just create empty provider
		fileProvider = &provider.FileProvider{}
	}

	handler := NewAgentHandlers(logger, fileProvider)

	payload := map[string]interface{}{
		"agent_id":    "agent-001",
		"hostname":    "test-server.example.com",
		"task_queues": []string{"linux-amd64"},
	}

	payloadBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/v1/agents/register", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.RegisterAgent(w, req)

	// Should fail with file provider
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "not_supported", errorObj["code"])
}

func TestAgentHandler_ListAgents(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	memProvider := provider.NewInMemoryProvider()

	// Setup test data
	agent1 := provider.ServerInfo{
		AgentID:    "agent-1",
		Hostname:   "server-1",
		IPAddress:  "192.168.1.10",
		TaskQueues: []string{"linux-amd64", "linux-common"},
		Status:     "healthy",
		Metadata:   map[string]string{"os": "linux"},
	}

	agent2 := provider.ServerInfo{
		AgentID:    "agent-2",
		Hostname:   "server-2",
		IPAddress:  "192.168.1.20",
		TaskQueues: []string{"web-servers"},
		Status:     "unhealthy",
		Metadata:   map[string]string{"os": "linux"},
	}

	memProvider.RegisterServer(agent1)
	memProvider.RegisterServer(agent2)

	handlers := NewAgentHandlers(logger, memProvider)

	tests := []struct {
		name           string
		query          string
		expectedCount  int
		expectedStatus int
	}{
		{
			name:           "list all agents",
			query:          "",
			expectedCount:  2,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "filter by task queue",
			query:          "?task_queue=linux-amd64",
			expectedCount:  1,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "filter by status healthy",
			query:          "?status=healthy",
			expectedCount:  1,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "filter by status unhealthy",
			query:          "?status=unhealthy",
			expectedCount:  1,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "filter by non-existent task queue",
			query:          "?task_queue=non-existent",
			expectedCount:  0,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "combined filters",
			query:          "?task_queue=linux-amd64&status=healthy",
			expectedCount:  1,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/agents"+tt.query, nil)
			w := httptest.NewRecorder()

			handlers.ListAgents(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			total, ok := resp["total"].(float64)
			require.True(t, ok, "response missing 'total' field")

			assert.Equal(t, tt.expectedCount, int(total))
		})
	}
}

func TestAgentHandler_ListAgentsMethodNotAllowed(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	memProvider := provider.NewInMemoryProvider()
	handlers := NewAgentHandlers(logger, memProvider)

	req := httptest.NewRequest(http.MethodPost, "/v1/agents", nil)
	w := httptest.NewRecorder()

	handlers.ListAgents(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestAgentHandler_GetAgent(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	memProvider := provider.NewInMemoryProvider()

	// Setup test data
	agent1 := provider.ServerInfo{
		AgentID:    "agent-123",
		Hostname:   "server-1",
		IPAddress:  "192.168.1.10",
		TaskQueues: []string{"linux-amd64"},
		Status:     "healthy",
		Metadata:   map[string]string{"os": "linux", "version": "v1.0.0"},
	}

	memProvider.RegisterServer(agent1)
	handlers := NewAgentHandlers(logger, memProvider)

	// Need to use gorilla/mux router for path parameters
	router := mux.NewRouter()
	router.HandleFunc("/v1/agents/{agent_id}", handlers.GetAgent).Methods(http.MethodGet)

	t.Run("get existing agent", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/agents/agent-123", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp provider.ServerInfo
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, "agent-123", resp.AgentID)
		assert.Equal(t, "server-1", resp.Hostname)
		assert.Equal(t, "192.168.1.10", resp.IPAddress)
	})

	t.Run("get non-existent agent", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/agents/agent-999", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		errorObj := resp["error"].(map[string]interface{})
		assert.Equal(t, "agent_not_found", errorObj["code"])
		assert.Contains(t, errorObj["message"], "agent-999")
	})

	t.Run("empty agent_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/agents/", nil)
		w := httptest.NewRecorder()

		// This won't match the route, should return 404 from router
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/agents/agent-123", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestAgentHandler_ListTaskQueues(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	memProvider := provider.NewInMemoryProvider()

	// Setup test data with different statuses
	agent1 := provider.ServerInfo{
		AgentID:    "agent-1",
		Hostname:   "server-1",
		TaskQueues: []string{"linux-amd64", "linux-common"},
		Status:     "healthy",
	}

	agent2 := provider.ServerInfo{
		AgentID:    "agent-2",
		Hostname:   "server-2",
		TaskQueues: []string{"linux-amd64"},
		Status:     "healthy",
	}

	agent3 := provider.ServerInfo{
		AgentID:    "agent-3",
		Hostname:   "server-3",
		TaskQueues: []string{"web-servers"},
		Status:     "unhealthy",
	}

	memProvider.RegisterServer(agent1)
	memProvider.RegisterServer(agent2)
	memProvider.RegisterServer(agent3)

	handlers := NewAgentHandlers(logger, memProvider)

	t.Run("list all task queues", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/task-queues", nil)
		w := httptest.NewRecorder()

		handlers.ListTaskQueues(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		total, ok := resp["total"].(float64)
		require.True(t, ok)
		assert.Equal(t, 3, int(total)) // linux-amd64, linux-common, web-servers

		queues, ok := resp["task_queues"].([]interface{})
		require.True(t, ok)

		// Verify queue statuses
		queueMap := make(map[string]map[string]interface{})
		for _, q := range queues {
			queue := q.(map[string]interface{})
			name := queue["name"].(string)
			queueMap[name] = queue
		}

		// linux-amd64 should be healthy (2 workers, both healthy)
		linuxQueue := queueMap["linux-amd64"]
		assert.Equal(t, "healthy", linuxQueue["status"])
		assert.Equal(t, float64(2), linuxQueue["worker_count"])
		assert.Equal(t, float64(2), linuxQueue["healthy_count"])

		// web-servers should be offline (1 worker, 0 healthy)
		webQueue := queueMap["web-servers"]
		assert.Equal(t, "offline", webQueue["status"])
		assert.Equal(t, float64(1), webQueue["worker_count"])
		assert.Equal(t, float64(0), webQueue["healthy_count"])
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/task-queues", nil)
		w := httptest.NewRecorder()

		handlers.ListTaskQueues(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestAgentHandler_ListTaskQueues_DegradedStatus(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	memProvider := provider.NewInMemoryProvider()

	// Setup degraded queue (some workers healthy, some not)
	agent1 := provider.ServerInfo{
		AgentID:    "agent-1",
		Hostname:   "server-1",
		TaskQueues: []string{"mixed-queue"},
		Status:     "healthy",
	}

	agent2 := provider.ServerInfo{
		AgentID:    "agent-2",
		Hostname:   "server-2",
		TaskQueues: []string{"mixed-queue"},
		Status:     "unhealthy",
	}

	memProvider.RegisterServer(agent1)
	memProvider.RegisterServer(agent2)

	handlers := NewAgentHandlers(logger, memProvider)

	req := httptest.NewRequest(http.MethodGet, "/v1/task-queues", nil)
	w := httptest.NewRecorder()

	handlers.ListTaskQueues(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	queues := resp["task_queues"].([]interface{})
	queue := queues[0].(map[string]interface{})

	assert.Equal(t, "degraded", queue["status"])
	assert.Equal(t, float64(2), queue["worker_count"])
	assert.Equal(t, float64(1), queue["healthy_count"])
}

func TestAgentHandler_UpdateAgentHeartbeat(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	memProvider := provider.NewInMemoryProvider()

	// Register an agent first
	agent := provider.ServerInfo{
		AgentID:    "agent-test",
		Hostname:   "test-server",
		TaskQueues: []string{"test-queue"},
		Status:     "healthy",
	}
	memProvider.RegisterServer(agent)

	handlers := NewAgentHandlers(logger, memProvider)

	t.Run("valid heartbeat update", func(t *testing.T) {
		payload := map[string]interface{}{
			"agent_id": "agent-test",
			"status":   "healthy",
		}
		payloadBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/v1/agents/heartbeat", bytes.NewReader(payloadBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handlers.UpdateAgentHeartbeat(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, "Heartbeat updated", resp["message"])
	})

	t.Run("missing agent_id", func(t *testing.T) {
		payload := map[string]interface{}{
			"status": "healthy",
		}
		payloadBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/v1/agents/heartbeat", bytes.NewReader(payloadBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handlers.UpdateAgentHeartbeat(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing status", func(t *testing.T) {
		payload := map[string]interface{}{
			"agent_id": "agent-test",
		}
		payloadBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/v1/agents/heartbeat", bytes.NewReader(payloadBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handlers.UpdateAgentHeartbeat(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid status value", func(t *testing.T) {
		payload := map[string]interface{}{
			"agent_id": "agent-test",
			"status":   "invalid-status",
		}
		payloadBytes, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/v1/agents/heartbeat", bytes.NewReader(payloadBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handlers.UpdateAgentHeartbeat(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		errorObj := resp["error"].(map[string]interface{})
		assert.Contains(t, errorObj["message"], "healthy, unhealthy, unknown")
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/agents/heartbeat", nil)
		w := httptest.NewRecorder()

		handlers.UpdateAgentHeartbeat(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestAgentHandler_GetAgentsSummary(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	memProvider := provider.NewInMemoryProvider()

	// Setup test data with mixed statuses
	agent1 := provider.ServerInfo{
		AgentID:    "agent-1",
		Hostname:   "server-1",
		TaskQueues: []string{"linux-amd64", "linux-common"},
		Status:     "healthy",
	}

	agent2 := provider.ServerInfo{
		AgentID:    "agent-2",
		Hostname:   "server-2",
		TaskQueues: []string{"linux-amd64"},
		Status:     "healthy",
	}

	agent3 := provider.ServerInfo{
		AgentID:    "agent-3",
		Hostname:   "server-3",
		TaskQueues: []string{"web-servers"},
		Status:     "unhealthy",
	}

	agent4 := provider.ServerInfo{
		AgentID:    "agent-4",
		Hostname:   "server-4",
		TaskQueues: []string{"gpu-queue"},
		Status:     "unhealthy",
	}

	memProvider.RegisterServer(agent1)
	memProvider.RegisterServer(agent2)
	memProvider.RegisterServer(agent3)
	memProvider.RegisterServer(agent4)

	handlers := NewAgentHandlers(logger, memProvider)

	t.Run("get summary statistics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/agents/summary", nil)
		w := httptest.NewRecorder()

		handlers.GetAgentsSummary(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var summary map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &summary)
		require.NoError(t, err)

		assert.Equal(t, float64(4), summary["total_agents"])
		assert.Equal(t, float64(2), summary["healthy_agents"])
		assert.Equal(t, float64(2), summary["unhealthy_agents"])
		assert.Equal(t, float64(4), summary["total_queues"])   // linux-amd64, linux-common, web-servers, gpu-queue
		assert.Equal(t, float64(2), summary["offline_queues"]) // web-servers, gpu-queue (no healthy workers)
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/agents/summary", nil)
		w := httptest.NewRecorder()

		handlers.GetAgentsSummary(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

// TestAgentHandler_ConcurrentRegistration tests concurrent agent registrations
func TestAgentHandler_ConcurrentRegistration(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	memProvider := provider.NewInMemoryProvider()
	handler := NewAgentHandlers(logger, memProvider)

	// Register 10 agents concurrently
	numAgents := 10
	done := make(chan bool, numAgents)

	for i := 0; i < numAgents; i++ {
		go func(agentID int) {
			payload := map[string]interface{}{
				"agent_id":    fmt.Sprintf("agent-%d", agentID),
				"hostname":    fmt.Sprintf("server-%d", agentID),
				"task_queues": []string{"test-queue"},
			}

			payloadBytes, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/v1/agents/register", bytes.NewReader(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.RegisterAgent(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numAgents; i++ {
		<-done
	}

	// Verify all agents registered
	req := httptest.NewRequest(http.MethodGet, "/v1/agents", nil)
	w := httptest.NewRecorder()
	handler.ListAgents(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	total := int(resp["total"].(float64))
	assert.Equal(t, numAgents, total)
}

// TestAgentHandler_ConcurrentHeartbeats tests concurrent heartbeat updates
func TestAgentHandler_ConcurrentHeartbeats(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	memProvider := provider.NewInMemoryProvider()

	// Register test agent
	agent := provider.ServerInfo{
		AgentID:    "agent-test",
		Hostname:   "test-server",
		TaskQueues: []string{"test-queue"},
		Status:     "healthy",
	}
	memProvider.RegisterServer(agent)

	handler := NewAgentHandlers(logger, memProvider)

	// Send 100 concurrent heartbeats
	numRequests := 100
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			payload := map[string]interface{}{
				"agent_id": "agent-test",
				"status":   "healthy",
			}
			payloadBytes, _ := json.Marshal(payload)

			req := httptest.NewRequest(http.MethodPost, "/v1/agents/heartbeat", bytes.NewReader(payloadBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.UpdateAgentHeartbeat(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
			done <- true
		}()
	}

	// Wait for all requests
	for i := 0; i < numRequests; i++ {
		<-done
	}

	// Verify agent still healthy
	ctx := context.Background()
	servers, err := memProvider.GetServers(ctx, "test-queue")
	require.NoError(t, err)
	require.Len(t, servers, 1)
	assert.Equal(t, "healthy", servers[0].Status)
}
