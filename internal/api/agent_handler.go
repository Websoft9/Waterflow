package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Websoft9/waterflow/pkg/temporal"
	"github.com/gorilla/mux"
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

// TaskQueueResponse represents a task queue status
type TaskQueueResponse struct {
	Name           string    `json:"name"`
	Pollers        int       `json:"pollers"`
	HealthyPollers int       `json:"healthy_pollers"`
	TaskBacklog    int64     `json:"task_backlog"`
	Status         string    `json:"status"`
	LastUpdateTime time.Time `json:"last_update_time"`
}

// ListTaskQueuesResponse represents the task queues list response
type ListTaskQueuesResponse struct {
	TaskQueues []TaskQueueResponse `json:"task_queues"`
	TotalCount int                 `json:"total_count"`
	Timestamp  time.Time           `json:"timestamp"`
}

// AgentsSummaryResponse represents aggregated agent health statistics
type AgentsSummaryResponse struct {
	TotalQueues       int       `json:"total_queues"`
	HealthyQueues     int       `json:"healthy_queues"`
	DegradedQueues    int       `json:"degraded_queues"`
	UnavailableQueues int       `json:"unavailable_queues"`
	TotalPollers      int       `json:"total_pollers"`
	HealthyPollers    int       `json:"healthy_pollers"`
	Timestamp         time.Time `json:"timestamp"`
}

// GetAgentStatus handles GET /v1/agents/:name
// Returns health status and worker count for a specific agent
func (h *AgentHandlers) GetAgentStatus(w http.ResponseWriter, r *http.Request) {
	if h.temporalClient == nil {
		writeError(w, r, http.StatusServiceUnavailable, "service_unavailable", "temporal client not configured", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Extract agent name from URL path (using gorilla/mux router variable)
	agentName := mux.Vars(r)["name"]
	if agentName == "" {
		h.logger.Warn("Agent name is required")
		writeError(w, r, http.StatusBadRequest, "invalid_request", "agent name is required", nil)
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
		writeError(w, r, http.StatusInternalServerError, "internal_error", "failed to query agent status",
			map[string]interface{}{"agent": agentName})
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response) //nolint:errcheck
}

// ListAgents handles GET /v1/agents
// Returns status of all known agents (discovered from Task Queues)
func (h *AgentHandlers) ListAgents(w http.ResponseWriter, r *http.Request) {
	if h.temporalClient == nil {
		writeError(w, r, http.StatusServiceUnavailable, "service_unavailable", "temporal client not configured", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	h.logger.Info("Listing agents")

	// Get all unique agents from recent workflows
	agentNames, err := h.discoverAgents(ctx)
	if err != nil {
		h.logger.Error("Failed to discover agents", zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "failed to discover agents", nil)
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
			// Include agent even if query fails (mark as unavailable, zero time signals no data)
			agents = append(agents, AgentResponse{
				Name:           agentName,
				Pollers:        0,
				HealthyPollers: 0,
				Status:         "unavailable",
				LastUpdateTime: time.Time{},
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response) //nolint:errcheck
}

// discoverAgents discovers agents from recent workflow executions via Temporal.
// It queries the task queues that have recently been used by workflow runs.
func (h *AgentHandlers) discoverAgents(ctx context.Context) ([]string, error) {
	return h.temporalClient.DiscoverTaskQueues(ctx, 50)
}

// determineAgentStatus determines agent health status
func determineAgentStatus(info *temporal.TaskQueueInfo) string {
	if info.Pollers == 0 || info.HealthyPollers == 0 {
		return "unavailable" // No workers connected
	}
	// Use multiplication to avoid integer division truncation (e.g., 1/3*2 = 0 not 0.67)
	if info.HealthyPollers*2 < info.Pollers {
		return "degraded" // Less than 50% workers healthy
	}
	return "healthy" // All workers healthy
}

// ListTaskQueues handles GET /v1/task-queues
// Returns all known task queues with real-time Temporal poller data (AC3)
func (h *AgentHandlers) ListTaskQueues(w http.ResponseWriter, r *http.Request) {
	if h.temporalClient == nil {
		writeError(w, r, http.StatusServiceUnavailable, "service_unavailable", "temporal client not configured", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	h.logger.Info("Listing task queues")

	queueNames, err := h.temporalClient.DiscoverTaskQueues(ctx, 50)
	if err != nil {
		h.logger.Error("Failed to discover task queues", zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "failed to discover task queues", nil)
		return
	}

	var queues []TaskQueueResponse
	for _, name := range queueNames {
		info, err := h.temporalClient.DescribeTaskQueue(ctx, name)
		if err != nil {
			h.logger.Warn("Failed to describe task queue",
				zap.String("queue", name),
				zap.Error(err),
			)
			queues = append(queues, TaskQueueResponse{
				Name:           name,
				Status:         "unavailable",
				LastUpdateTime: time.Time{},
			})
			continue
		}
		queues = append(queues, TaskQueueResponse{
			Name:           info.Name,
			Pollers:        info.Pollers,
			HealthyPollers: info.HealthyPollers,
			TaskBacklog:    info.TaskBacklog,
			Status:         determineAgentStatus(info),
			LastUpdateTime: info.LastUpdateTime,
		})
	}

	response := ListTaskQueuesResponse{
		TaskQueues: queues,
		TotalCount: len(queues),
		Timestamp:  time.Now(),
	}

	h.logger.Info("Task queues listed", zap.Int("count", len(queues)))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response) //nolint:errcheck
}

// GetAgentsSummary handles GET /v1/agents/summary
// Returns aggregated health statistics across all known task queues (AC6)
func (h *AgentHandlers) GetAgentsSummary(w http.ResponseWriter, r *http.Request) {
	if h.temporalClient == nil {
		writeError(w, r, http.StatusServiceUnavailable, "service_unavailable", "temporal client not configured", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	h.logger.Info("Getting agents summary")

	queueNames, err := h.temporalClient.DiscoverTaskQueues(ctx, 50)
	if err != nil {
		h.logger.Error("Failed to discover task queues for summary", zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "failed to discover task queues", nil)
		return
	}

	summary := AgentsSummaryResponse{
		TotalQueues: len(queueNames),
		Timestamp:   time.Now(),
	}

	for _, name := range queueNames {
		info, err := h.temporalClient.DescribeTaskQueue(ctx, name)
		if err != nil {
			h.logger.Warn("Failed to describe task queue for summary",
				zap.String("queue", name),
				zap.Error(err),
			)
			summary.UnavailableQueues++
			continue
		}
		switch determineAgentStatus(info) {
		case "healthy":
			summary.HealthyQueues++
		case "degraded":
			summary.DegradedQueues++
		default:
			summary.UnavailableQueues++
		}
		summary.TotalPollers += info.Pollers
		summary.HealthyPollers += info.HealthyPollers
	}

	h.logger.Info("Agents summary computed",
		zap.Int("total_queues", summary.TotalQueues),
		zap.Int("healthy", summary.HealthyQueues),
		zap.Int("degraded", summary.DegradedQueues),
		zap.Int("unavailable", summary.UnavailableQueues),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(summary) //nolint:errcheck
}
