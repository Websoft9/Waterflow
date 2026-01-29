package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Websoft9/waterflow/pkg/temporal"
	"go.uber.org/zap"
)

// AgentHandlers handles agent discovery and status endpoints
type AgentHandlers struct {
	logger         *zap.Logger
	temporalClient *temporal.Client
}

// NewAgentHandlers creates agent handlers
func NewAgentHandlers(logger *zap.Logger, temporalClient *temporal.Client) *AgentHandlers {
	return &AgentHandlers{
		logger:         logger,
		temporalClient: temporalClient,
	}
}

// AgentResponse represents an agent status
type AgentResponse struct {
	Name           string    `json:"name"`
	Pollers        int       `json:"pollers"`
	HealthyPollers int       `json:"healthy_pollers"`
	TaskBacklog    int64     `json:"task_backlog"`
	LastUpdateTime time.Time `json:"last_update_time"`
	Status         string    `json:"status"` // "healthy", "degraded", "unavailable"
}

// ListAgentsResponse represents list agents response
type ListAgentsResponse struct {
	Agents     []AgentResponse `json:"agents"`
	TotalCount int             `json:"total_count"`
	Timestamp  time.Time       `json:"timestamp"`
}

// GetAgentStatus handles GET /v1/agents/:name
// Returns health status and worker count for a specific agent
func (h *AgentHandlers) GetAgentStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Extract agent name from URL path
	agentName := r.URL.Path[len("/v1/agents/"):]
	if agentName == "" {
		h.logger.Warn("Agent name is required")
		writeJSONError(w, http.StatusBadRequest, "agent name is required", nil)
		return
	}

	h.logger.Info("Getting agent status", zap.String("agent", agentName))

	// Query Temporal for agent status (via Task Queue)
	info, err := h.temporalClient.DescribeTaskQueue(ctx, agentName)
	if err != nil {
		h.logger.Error("Failed to describe agent",
			zap.String("agent", agentName),
			zap.Error(err),
		)
		writeJSONError(w, http.StatusInternalServerError,
			"failed to query agent status", map[string]interface{}{
				"agent": agentName,
				"error": err.Error(),
			})
		return
	}

	// Determine status
	status := determineAgentStatus(info)

	response := AgentResponse{
		Name:           info.Name,
		Pollers:        info.Pollers,
		HealthyPollers: info.HealthyPollers,
		TaskBacklog:    info.TaskBacklog,
		LastUpdateTime: info.LastUpdateTime,
		Status:         status,
	}

	h.logger.Info("Agent status retrieved",
		zap.String("agent", agentName),
		zap.String("status", status),
		zap.Int("pollers", info.Pollers),
	)

	writeJSONResponse(w, http.StatusOK, response)
}

// ListAgents handles GET /v1/agents
// Returns status of all known agents (discovered from Task Queues)
func (h *AgentHandlers) ListAgents(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	h.logger.Info("Listing agents")

	// Get all unique agents from recent workflows
	agentNames, err := h.discoverAgents(ctx)
	if err != nil {
		h.logger.Error("Failed to discover agents", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError,
			"failed to discover agents", map[string]interface{}{
				"error": err.Error(),
			})
		return
	}

	// Query status for each agent
	var agents []AgentResponse
	for _, agentName := range agentNames {
		info, err := h.temporalClient.DescribeTaskQueue(ctx, agentName)
		if err != nil {
			h.logger.Warn("Failed to describe agent",
				zap.String("agent", agentName),
				zap.Error(err),
			)
			// Include agent even if query fails (mark as unavailable)
			agents = append(agents, AgentResponse{
				Name:           agentName,
				Pollers:        0,
				HealthyPollers: 0,
				Status:         "unavailable",
				LastUpdateTime: time.Now(),
			})
			continue
		}

		status := determineAgentStatus(info)
		agents = append(agents, AgentResponse{
			Name:           info.Name,
			Pollers:        info.Pollers,
			HealthyPollers: info.HealthyPollers,
			TaskBacklog:    info.TaskBacklog,
			LastUpdateTime: info.LastUpdateTime,
			Status:         status,
		})
	}

	response := ListAgentsResponse{
		Agents:     agents,
		TotalCount: len(agents),
		Timestamp:  time.Now(),
	}

	h.logger.Info("Agents listed", zap.Int("count", len(agents)))

	writeJSONResponse(w, http.StatusOK, response)
}

// discoverAgents discovers agents from recent workflows
// This is a simplified implementation - could be enhanced with caching
func (h *AgentHandlers) discoverAgents(ctx context.Context) ([]string, error) {
	// TODO: Implement agent discovery from workflow history
	// For now, return a predefined list or query from configuration

	// Option 1: Return common agent names
	commonAgents := []string{
		"linux-amd64",
		"linux-arm64",
		"macos-arm64",
		"windows-x64",
	}

	return commonAgents, nil
}

// determineAgentStatus determines agent health status
func determineAgentStatus(info *temporal.TaskQueueInfo) string {
	if info.HealthyPollers == 0 {
		return "unavailable" // No healthy workers
	}
	if info.HealthyPollers < info.Pollers/2 {
		return "degraded" // Less than 50% workers healthy
	}
	return "healthy" // All workers healthy
}

// Helper functions

func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log error but can't change response at this point
		return
	}
}

func writeJSONError(w http.ResponseWriter, statusCode int, message string, details interface{}) {
	response := map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
		},
	}
	if details != nil {
		response["error"].(map[string]interface{})["details"] = details
	}
	writeJSONResponse(w, statusCode, response)
}
