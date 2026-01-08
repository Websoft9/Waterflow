package metrics

import (
	"sync"
)

// AgentTracker tracks agent connection and health metrics
type AgentTracker struct {
	mu              sync.RWMutex
	connectedAgents map[string]bool // agentID -> is healthy
}

// NewAgentTracker creates a new agent tracker
func NewAgentTracker() *AgentTracker {
	return &AgentTracker{
		connectedAgents: make(map[string]bool),
	}
}

// TrackConnection records an agent connection
func (at *AgentTracker) TrackConnection(agentID string) {
	at.mu.Lock()
	defer at.mu.Unlock()

	if _, exists := at.connectedAgents[agentID]; !exists {
		at.connectedAgents[agentID] = true
		AgentsConnected.Inc()
		AgentsHealthy.Inc()
	}
}

// TrackDisconnection records an agent disconnection
func (at *AgentTracker) TrackDisconnection(agentID string) {
	at.mu.Lock()
	defer at.mu.Unlock()

	if healthy, exists := at.connectedAgents[agentID]; exists {
		delete(at.connectedAgents, agentID)
		AgentsConnected.Dec()

		if healthy {
			AgentsHealthy.Dec()
		}
	}
}

// TrackHealthChange records agent health status change
func (at *AgentTracker) TrackHealthChange(agentID string, healthy bool) {
	at.mu.Lock()
	defer at.mu.Unlock()

	oldHealthy, exists := at.connectedAgents[agentID]
	if !exists {
		return
	}

	if oldHealthy != healthy {
		at.connectedAgents[agentID] = healthy

		if healthy {
			AgentsHealthy.Inc()
		} else {
			AgentsHealthy.Dec()
		}
	}
}

// TrackTaskExecution records task execution by agent
func (at *AgentTracker) TrackTaskExecution(agentID string, success bool) {
	status := "completed"
	if !success {
		status = "failed"
	}

	AgentTasksTotal.WithLabelValues(agentID, status).Inc()
}

// GetConnectedCount returns number of connected agents
func (at *AgentTracker) GetConnectedCount() int {
	at.mu.RLock()
	defer at.mu.RUnlock()
	return len(at.connectedAgents)
}

// GetHealthyCount returns number of healthy agents
func (at *AgentTracker) GetHealthyCount() int {
	at.mu.RLock()
	defer at.mu.RUnlock()

	count := 0
	for _, healthy := range at.connectedAgents {
		if healthy {
			count++
		}
	}
	return count
}
