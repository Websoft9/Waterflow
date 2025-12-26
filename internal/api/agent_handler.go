package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Websoft9/waterflow/pkg/provider"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// AgentHandlers contains agent-related HTTP handlers
type AgentHandlers struct {
	logger              *zap.Logger
	serverGroupProvider provider.ServerGroupProvider
}

// NewAgentHandlers creates new AgentHandlers instance
func NewAgentHandlers(logger *zap.Logger, sgProvider provider.ServerGroupProvider) *AgentHandlers {
	return &AgentHandlers{
		logger:              logger,
		serverGroupProvider: sgProvider,
	}
}

// RegisterAgentRequest represents the agent registration payload.
type RegisterAgentRequest struct {
	AgentID    string            `json:"agent_id"`
	Hostname   string            `json:"hostname"`
	IPAddress  string            `json:"ip_address,omitempty"`
	TaskQueues []string          `json:"task_queues"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// RegisterAgent handles POST /v1/agents/register
func (h *AgentHandlers) RegisterAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "invalid_request",
				"message": "Invalid JSON payload: " + err.Error(),
			},
		})
		return
	}

	// Validate required fields
	if req.AgentID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "invalid_request",
				"message": "agent_id is required",
			},
		})
		return
	}
	if req.Hostname == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "invalid_request",
				"message": "hostname is required",
			},
		})
		return
	}
	if len(req.TaskQueues) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "invalid_request",
				"message": "task_queues must contain at least one queue",
			},
		})
		return
	}

	// Create ServerInfo
	serverInfo := provider.ServerInfo{
		AgentID:       req.AgentID,
		Hostname:      req.Hostname,
		IPAddress:     req.IPAddress,
		TaskQueues:    req.TaskQueues,
		Status:        "healthy",
		LastHeartbeat: time.Now(),
		Metadata:      req.Metadata,
	}

	// Register to provider (only works with InMemoryProvider)
	if memProvider, ok := h.serverGroupProvider.(*provider.InMemoryProvider); ok {
		if err := memProvider.RegisterServer(serverInfo); err != nil {
			h.logger.Error("Failed to register agent", zap.Error(err))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "registration_failed",
					"message": "Failed to register agent",
				},
			})
			return
		}
	} else {
		// FileProvider doesn't support dynamic registration
		h.logger.Warn("Agent registration not supported for current provider type")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "not_supported",
				"message": "Agent registration only supported with in-memory provider",
			},
		})
		return
	}

	h.logger.Info("Agent registered",
		zap.String("agent_id", req.AgentID),
		zap.String("hostname", req.Hostname),
		zap.Strings("task_queues", req.TaskQueues),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "Agent registered successfully",
		"agent_id": req.AgentID,
	})
}

// ListAgents returns a list of all registered agents.
func (h *AgentHandlers) ListAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Query filter parameters
	taskQueue := r.URL.Query().Get("task_queue")
	status := r.URL.Query().Get("status")

	// Get all groups from provider
	groups, err := h.serverGroupProvider.ListGroups(ctx)
	if err != nil {
		h.logger.Error("Failed to list groups", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "provider_error",
				"message": "Failed to query server groups",
			},
		})
		return
	}

	// Collect all unique agents
	agentMap := make(map[string]*provider.ServerInfo)

	for _, group := range groups {
		servers, err := h.serverGroupProvider.GetServers(ctx, group)
		if err != nil {
			h.logger.Warn("Failed to get servers for group",
				zap.String("group", group),
				zap.Error(err),
			)
			continue
		}

		for _, server := range servers {
			agentMap[server.AgentID] = &server
		}
	}

	// Convert to slice and apply filters
	agents := make([]provider.ServerInfo, 0, len(agentMap))
	for _, agent := range agentMap {
		// Filter by task queue
		if taskQueue != "" {
			hasQueue := false
			for _, q := range agent.TaskQueues {
				if q == taskQueue {
					hasQueue = true
					break
				}
			}
			if !hasQueue {
				continue
			}
		}

		// Filter by status
		if status != "" && agent.Status != status {
			continue
		}

		agents = append(agents, *agent)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agents": agents,
		"total":  len(agents),
	})
}

// GetAgent returns details of a specific agent.
func (h *AgentHandlers) GetAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Extract agent_id from URL path using mux.Vars
	vars := mux.Vars(r)
	agentID := vars["agent_id"]

	if agentID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "invalid_request",
				"message": "agent_id is required",
			},
		})
		return
	}

	// Search across all groups
	groups, err := h.serverGroupProvider.ListGroups(ctx)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "provider_error",
				"message": "Failed to query server groups",
			},
		})
		return
	}

	for _, group := range groups {
		servers, err := h.serverGroupProvider.GetServers(ctx, group)
		if err != nil {
			continue
		}

		for _, server := range servers {
			if server.AgentID == agentID {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(server)
				return
			}
		}
	}

	// Agent not found
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    "agent_not_found",
			"message": fmt.Sprintf("Agent %s not found", agentID),
		},
	})
}

// TaskQueueInfo represents information about a task queue.
type TaskQueueInfo struct {
	Name         string    `json:"name"`
	WorkerCount  int       `json:"worker_count"`
	HealthyCount int       `json:"healthy_count"`
	Status       string    `json:"status"`
	LastActivity time.Time `json:"last_activity"`
}

// ListTaskQueues returns a list of all task queues and their worker counts.
func (h *AgentHandlers) ListTaskQueues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Get all groups from provider
	groups, err := h.serverGroupProvider.ListGroups(ctx)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "provider_error",
				"message": "Failed to query server groups",
			},
		})
		return
	}

	// Build task queue info
	queueMap := make(map[string]*TaskQueueInfo)

	for _, group := range groups {
		if queueMap[group] == nil {
			queueMap[group] = &TaskQueueInfo{
				Name:         group,
				WorkerCount:  0,
				HealthyCount: 0,
				LastActivity: time.Time{},
			}
		}

		servers, err := h.serverGroupProvider.GetServers(ctx, group)
		if err != nil {
			continue
		}

		for _, server := range servers {
			queueMap[group].WorkerCount++
			if server.Status == "healthy" {
				queueMap[group].HealthyCount++
			}
			if server.LastHeartbeat.After(queueMap[group].LastActivity) {
				queueMap[group].LastActivity = server.LastHeartbeat
			}
		}

		// Determine queue status
		if queueMap[group].WorkerCount == 0 {
			queueMap[group].Status = "no_workers" // Never had any workers
		} else if queueMap[group].HealthyCount == 0 {
			queueMap[group].Status = "offline" // Had workers but all unhealthy
		} else if queueMap[group].HealthyCount < queueMap[group].WorkerCount {
			queueMap[group].Status = "degraded" // Some workers unhealthy
		} else {
			queueMap[group].Status = "healthy" // All workers healthy
		}
	}

	// Convert to slice
	queues := make([]TaskQueueInfo, 0, len(queueMap))
	for _, info := range queueMap {
		queues = append(queues, *info)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"task_queues": queues,
		"total":       len(queues),
	})
}

// UpdateAgentHeartbeat updates an agent's heartbeat.
// TODO(security): Add agent authentication (API key, mTLS, or JWT) to prevent
// unauthorized heartbeat spoofing. Currently any client can update any agent's status.
func (h *AgentHandlers) UpdateAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AgentID string `json:"agent_id"`
		Status  string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	// Validate required fields
	if req.AgentID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "invalid_request",
				"message": "agent_id is required",
			},
		})
		return
	}

	if req.Status == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "invalid_request",
				"message": "status is required",
			},
		})
		return
	}

	// Validate status value
	validStatuses := map[string]bool{"healthy": true, "unhealthy": true, "unknown": true}
	if !validStatuses[req.Status] {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "invalid_request",
				"message": "status must be one of: healthy, unhealthy, unknown",
			},
		})
		return
	}

	// Update heartbeat (only works with InMemoryProvider)
	if memProvider, ok := h.serverGroupProvider.(*provider.InMemoryProvider); ok {
		if err := memProvider.UpdateHeartbeat(req.AgentID, req.Status); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "update_failed",
					"message": "Failed to update heartbeat",
				},
			})
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Heartbeat updated",
	})
}

// GetAgentsSummary returns aggregated agent health statistics.
func (h *AgentHandlers) GetAgentsSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	groups, err := h.serverGroupProvider.ListGroups(ctx)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "provider_error",
				"message": "Failed to query server groups",
			},
		})
		return
	}

	summary := struct {
		TotalAgents     int `json:"total_agents"`
		HealthyAgents   int `json:"healthy_agents"`
		UnhealthyAgents int `json:"unhealthy_agents"`
		TotalQueues     int `json:"total_queues"`
		OfflineQueues   int `json:"offline_queues"`
	}{}

	agentMap := make(map[string]*provider.ServerInfo)
	queueStatus := make(map[string]bool) // queue -> has healthy worker

	for _, group := range groups {
		servers, err := h.serverGroupProvider.GetServers(ctx, group)
		if err != nil {
			continue
		}

		hasHealthy := false
		for _, server := range servers {
			agentMap[server.AgentID] = &server
			if server.Status == "healthy" {
				hasHealthy = true
			}
		}
		queueStatus[group] = hasHealthy
	}

	summary.TotalAgents = len(agentMap)
	summary.TotalQueues = len(queueStatus)

	for _, agent := range agentMap {
		if agent.Status == "healthy" {
			summary.HealthyAgents++
		} else {
			summary.UnhealthyAgents++
		}
	}

	for _, hasHealthy := range queueStatus {
		if !hasHealthy {
			summary.OfflineQueues++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(summary)
}
