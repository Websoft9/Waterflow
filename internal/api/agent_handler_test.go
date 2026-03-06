package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/temporal"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ---------------------------------------------------------------------------
// determineAgentStatus — pure function, no Temporal client needed
// ---------------------------------------------------------------------------

func TestDetermineAgentStatus_Unavailable_NoPollers(t *testing.T) {
	info := &temporal.TaskQueueInfo{Pollers: 0, HealthyPollers: 0}
	assert.Equal(t, "unavailable", determineAgentStatus(info))
}

func TestDetermineAgentStatus_Unavailable_ZeroHealthy(t *testing.T) {
	info := &temporal.TaskQueueInfo{Pollers: 3, HealthyPollers: 0}
	assert.Equal(t, "unavailable", determineAgentStatus(info))
}

func TestDetermineAgentStatus_Degraded(t *testing.T) {
	tests := []struct {
		pollers        int
		healthyPollers int
	}{
		{pollers: 4, healthyPollers: 1},  // 25% healthy → degraded
		{pollers: 3, healthyPollers: 1},  // 33% healthy → degraded (1*2 < 3)
		{pollers: 10, healthyPollers: 4}, // 40% healthy → degraded (4*2=8 < 10)
	}
	for _, tc := range tests {
		info := &temporal.TaskQueueInfo{Pollers: tc.pollers, HealthyPollers: tc.healthyPollers}
		assert.Equal(t, "degraded", determineAgentStatus(info),
			"pollers=%d healthyPollers=%d", tc.pollers, tc.healthyPollers)
	}
}

func TestDetermineAgentStatus_Healthy(t *testing.T) {
	tests := []struct {
		pollers        int
		healthyPollers int
	}{
		{pollers: 2, healthyPollers: 1}, // exactly 50%: 1*2 == 2, NOT less than → healthy
		{pollers: 3, healthyPollers: 2}, // 66% healthy
		{pollers: 4, healthyPollers: 2}, // exactly 50%
		{pollers: 1, healthyPollers: 1}, // single healthy worker
	}
	for _, tc := range tests {
		info := &temporal.TaskQueueInfo{Pollers: tc.pollers, HealthyPollers: tc.healthyPollers}
		assert.Equal(t, "healthy", determineAgentStatus(info),
			"pollers=%d healthyPollers=%d", tc.pollers, tc.healthyPollers)
	}
}

// ---------------------------------------------------------------------------
// NewAgentHandlers — construction
// ---------------------------------------------------------------------------

func TestNewAgentHandlers(t *testing.T) {
	logger := zaptest.NewLogger(t)
	h := NewAgentHandlers(logger, nil)
	require.NotNil(t, h)
	assert.Nil(t, h.temporalClient)
}

// ---------------------------------------------------------------------------
// GetAgentStatus — nil client guard (AC2 error path)
// ---------------------------------------------------------------------------

func TestGetAgentStatus_NoTemporalClient(t *testing.T) {
	logger := zaptest.NewLogger(t)
	h := NewAgentHandlers(logger, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/agents/linux-amd64", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "linux-amd64"})
	w := httptest.NewRecorder()

	h.GetAgentStatus(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestGetAgentStatus_EmptyName(t *testing.T) {
	logger := zaptest.NewLogger(t)
	h := NewAgentHandlers(logger, nil)

	// Even with a nil client, an empty name should return 400 — but nil guard fires first.
	// Test the actual empty-name validation by checking the handler logic order:
	// nil-client guard → 503; empty name (after client check) → 400.
	// With nil client we get 503.
	req := httptest.NewRequest(http.MethodGet, "/v1/agents/", nil)
	req = mux.SetURLVars(req, map[string]string{"name": ""})
	w := httptest.NewRecorder()

	h.GetAgentStatus(w, req)

	// nil client fires before name validation — 503 expected
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// ListAgents — nil client guard (AC1 error path)
// ---------------------------------------------------------------------------

func TestListAgents_NoTemporalClient(t *testing.T) {
	logger := zaptest.NewLogger(t)
	h := NewAgentHandlers(logger, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/agents", nil)
	w := httptest.NewRecorder()

	h.ListAgents(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// ListTaskQueues — nil client guard (AC3 error path)
// ---------------------------------------------------------------------------

func TestListTaskQueues_NoTemporalClient_AgentHandlers(t *testing.T) {
	logger := zaptest.NewLogger(t)
	h := NewAgentHandlers(logger, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/task-queues", nil)
	w := httptest.NewRecorder()

	h.ListTaskQueues(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// GetAgentsSummary — nil client guard (AC6 error path)
// ---------------------------------------------------------------------------

func TestGetAgentsSummary_NoTemporalClient(t *testing.T) {
	logger := zaptest.NewLogger(t)
	h := NewAgentHandlers(logger, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/agents/summary", nil)
	w := httptest.NewRecorder()

	h.GetAgentsSummary(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// Response struct marshaling — verify JSON field names match OpenAPI spec
// ---------------------------------------------------------------------------

func TestAgentResponse_JSONFields(t *testing.T) {
	resp := AgentResponse{
		Name:           "linux-amd64",
		Pollers:        3,
		HealthyPollers: 2,
		TaskBacklog:    5,
		Status:         "degraded",
		LastUpdateTime: time.Time{},
	}
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Equal(t, "linux-amd64", m["name"])
	assert.Equal(t, float64(3), m["pollers"])
	assert.Equal(t, float64(2), m["healthy_pollers"])
	assert.Equal(t, float64(5), m["task_backlog"])
	assert.Equal(t, "degraded", m["status"])
	assert.Contains(t, m, "last_update_time")
}

func TestListAgentsResponse_JSONFields(t *testing.T) {
	resp := ListAgentsResponse{
		Agents:     []AgentResponse{{Name: "q1", Status: "healthy"}},
		TotalCount: 1,
		Timestamp:  time.Now(),
	}
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Contains(t, m, "agents")
	assert.Equal(t, float64(1), m["total_count"])
	assert.Contains(t, m, "timestamp")
}

func TestAgentsSummaryResponse_JSONFields(t *testing.T) {
	resp := AgentsSummaryResponse{
		TotalQueues:       5,
		HealthyQueues:     3,
		DegradedQueues:    1,
		UnavailableQueues: 1,
		TotalPollers:      12,
		HealthyPollers:    10,
		Timestamp:         time.Now(),
	}
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Equal(t, float64(5), m["total_queues"])
	assert.Equal(t, float64(3), m["healthy_queues"])
	assert.Equal(t, float64(1), m["degraded_queues"])
	assert.Equal(t, float64(1), m["unavailable_queues"])
	assert.Equal(t, float64(12), m["total_pollers"])
	assert.Equal(t, float64(10), m["healthy_pollers"])
}
