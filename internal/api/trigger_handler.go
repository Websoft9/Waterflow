package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/Websoft9/waterflow/pkg/trigger"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// TriggerHandlers handles Trigger management endpoints
type TriggerHandlers struct {
	logger  *zap.Logger
	manager *trigger.Manager
}

// NewTriggerHandlers creates new TriggerHandlers instance
func NewTriggerHandlers(logger *zap.Logger, manager *trigger.Manager) *TriggerHandlers {
	return &TriggerHandlers{
		logger:  logger,
		manager: manager,
	}
}

// CreateTrigger handles POST /v1/workflows/{name}/triggers
func (h *TriggerHandlers) CreateTrigger(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowName := vars["name"]

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read request body", zap.Error(err))
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse request
	var req trigger.CreateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Error("Failed to parse request", zap.Error(err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Override workflow name from URL
	req.WorkflowName = workflowName

	// Set default type if not specified
	if req.Type == "" {
		req.Type = trigger.TypeWebhook
	}

	// Create trigger
	trig, err := h.manager.Create(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create trigger",
			zap.String("workflow_name", workflowName),
			zap.Error(err),
		)

		// HIGH-2 fix: use errors.Is instead of string matching
		switch {
		case errors.Is(err, trigger.ErrWorkflowNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, trigger.ErrAlreadyExists):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	// MEDIUM-1 fix: expose secret only on Create via TriggerResponse
	response := trig.ToResponse(true)

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}

// ListTriggers handles GET /v1/workflows/{name}/triggers
func (h *TriggerHandlers) ListTriggers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowName := vars["name"]

	// Parse query parameters
	filter := &trigger.Filter{
		WorkflowName: workflowName,
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = trigger.Status(status)
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	} else {
		filter.Limit = 20 // Default limit
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	// List triggers
	triggers, total, err := h.manager.List(r.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list triggers",
			zap.String("workflow_name", workflowName),
			zap.Error(err),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build response (no secrets in list)
	triggerResponses := make([]*trigger.TriggerResponse, len(triggers))
	for i, t := range triggers {
		triggerResponses[i] = t.ToResponse(false)
	}
	response := map[string]interface{}{
		"triggers": triggerResponses,
		"total":    total,
		"limit":    filter.Limit,
		"offset":   filter.Offset,
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}

// GetTrigger handles GET /v1/workflows/{name}/triggers/{trigger_id}
func (h *TriggerHandlers) GetTrigger(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	triggerID := vars["trigger_id"]

	// Get trigger
	trig, err := h.manager.Get(r.Context(), triggerID)
	if err != nil {
		h.logger.Error("Failed to get trigger",
			zap.String("trigger_id", triggerID),
			zap.Error(err),
		)
		http.Error(w, "Trigger not found", http.StatusNotFound)
		return
	}

	// Return response (no secret exposed in Get)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(trig.ToResponse(false)); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}

// UpdateTrigger handles PATCH /v1/workflows/{name}/triggers/{trigger_id}
func (h *TriggerHandlers) UpdateTrigger(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	triggerID := vars["trigger_id"]

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read request body", zap.Error(err))
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse request
	var req trigger.UpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Error("Failed to parse request", zap.Error(err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Update trigger
	trig, err := h.manager.Update(r.Context(), triggerID, &req)
	if err != nil {
		h.logger.Error("Failed to update trigger",
			zap.String("trigger_id", triggerID),
			zap.Error(err),
		)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return response (no secret exposed in Update)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(trig.ToResponse(false)); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}

// DeleteTrigger handles DELETE /v1/workflows/{name}/triggers/{trigger_id}
func (h *TriggerHandlers) DeleteTrigger(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	triggerID := vars["trigger_id"]

	// Delete trigger
	if err := h.manager.Delete(r.Context(), triggerID); err != nil {
		h.logger.Error("Failed to delete trigger",
			zap.String("trigger_id", triggerID),
			zap.Error(err),
		)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// EnableTrigger handles POST /v1/workflows/{name}/triggers/{trigger_id}/enable
func (h *TriggerHandlers) EnableTrigger(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	triggerID := vars["trigger_id"]

	// Enable trigger
	trig, err := h.manager.Enable(r.Context(), triggerID)
	if err != nil {
		h.logger.Error("Failed to enable trigger",
			zap.String("trigger_id", triggerID),
			zap.Error(err),
		)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Return response (no secret exposed in Enable)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(trig.ToResponse(false)); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}

// DisableTrigger handles POST /v1/workflows/{name}/triggers/{trigger_id}/disable
func (h *TriggerHandlers) DisableTrigger(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	triggerID := vars["trigger_id"]

	// Read request body (optional reason)
	var req struct {
		Reason string `json:"reason"`
	}
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err == nil {
			json.Unmarshal(body, &req)
		}
		r.Body.Close()
	}

	// Disable trigger
	trig, err := h.manager.Disable(r.Context(), triggerID, req.Reason)
	if err != nil {
		h.logger.Error("Failed to disable trigger",
			zap.String("trigger_id", triggerID),
			zap.Error(err),
		)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Return response (no secret exposed in Disable)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(trig.ToResponse(false)); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}

// GetTriggerLogs handles GET /v1/workflows/{name}/triggers/{trigger_id}/logs
func (h *TriggerHandlers) GetTriggerLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	triggerID := vars["trigger_id"]

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	logs, err := h.manager.GetWebhookLogs(r.Context(), triggerID, limit)
	if err != nil {
		h.logger.Error("Failed to get webhook logs",
			zap.String("trigger_id", triggerID),
			zap.Error(err),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"trigger_id": triggerID,
		"logs":       logs,
		"total":      len(logs),
	}); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}
