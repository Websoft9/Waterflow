package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Websoft9/waterflow/internal/server/schedule"
	"github.com/gorilla/mux"
	enumspb "go.temporal.io/api/enums/v1"
	"go.uber.org/zap"
)

// ScheduleHandlers handles Schedule endpoints
type ScheduleHandlers struct {
	logger  *zap.Logger
	manager *schedule.Manager
}

// NewScheduleHandlers creates new ScheduleHandlers instance
func NewScheduleHandlers(logger *zap.Logger, manager *schedule.Manager) *ScheduleHandlers {
	return &ScheduleHandlers{
		logger:  logger,
		manager: manager,
	}
}

// CreateScheduleRequest represents create schedule request (maps to schedule.CreateRequest)
type CreateScheduleRequest struct {
	Name          string                 `json:"name"`
	WorkflowName  string                 `json:"workflow_name"`
	Cron          string                 `json:"cron"`
	Timezone      string                 `json:"timezone,omitempty"`
	OverlapPolicy string                 `json:"overlap_policy,omitempty"`
	Vars          map[string]interface{} `json:"vars,omitempty"`
	Paused        bool                   `json:"paused,omitempty"`
}

// UpdateScheduleRequest represents update schedule request (maps to schedule.UpdateRequest)
type UpdateScheduleRequest struct {
	Cron          string                 `json:"cron,omitempty"`
	Timezone      string                 `json:"timezone,omitempty"`
	OverlapPolicy string                 `json:"overlap_policy,omitempty"`
	Vars          map[string]interface{} `json:"vars,omitempty"`
}

// TriggerScheduleRequest represents trigger schedule request
type TriggerScheduleRequest struct {
	OverlapPolicy string                 `json:"overlap_policy,omitempty"`
	Vars          map[string]interface{} `json:"vars,omitempty"`
}

// PauseScheduleRequest represents pause schedule request
type PauseScheduleRequest struct {
	Reason string `json:"reason,omitempty"`
}

// ResumeScheduleRequest represents resume schedule request
type ResumeScheduleRequest struct {
	Reason string `json:"reason,omitempty"`
}

// ScheduleResponse represents schedule response
type ScheduleResponse struct {
	ScheduleID    string                 `json:"schedule_id"`
	WorkflowName  string                 `json:"workflow_name"`
	Cron          string                 `json:"cron"`
	Timezone      string                 `json:"timezone"`
	OverlapPolicy string                 `json:"overlap_policy,omitempty"`
	Status        string                 `json:"status"`
	Vars          map[string]interface{} `json:"vars,omitempty"`
	NextRunTime   string                 `json:"next_run_time,omitempty"`
	LastRunTime   string                 `json:"last_run_time,omitempty"`
	LastRunID     string                 `json:"last_run_id,omitempty"`
	LastRunStatus string                 `json:"last_run_status,omitempty"`
	TotalRuns     int                    `json:"total_runs,omitempty"`
	CreatedAt     string                 `json:"created_at,omitempty"`
	UpdatedAt     string                 `json:"updated_at,omitempty"`
	Note          string                 `json:"note,omitempty"`
}

// ListSchedulesResponse represents list schedules response
type ListSchedulesResponse struct {
	Schedules []ScheduleResponse `json:"schedules"`
	Total     int                `json:"total"`
	Page      int                `json:"page"`
	Limit     int                `json:"limit"`
}

// TriggerScheduleResponse represents trigger schedule response
type TriggerScheduleResponse struct {
	ScheduleID  string `json:"schedule_id"`
	WorkflowID  string `json:"workflow_id"`
	TriggeredAt string `json:"triggered_at"`
	Status      string `json:"status"`
}

// CreateSchedule handles POST /v1/workflows/{name}/schedules
func (h *ScheduleHandlers) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	// Extract workflow name from URL
	vars := mux.Vars(r)
	workflowName := vars["name"]

	// Parse request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Failed to read request body", nil)
		return
	}
	defer r.Body.Close()

	var req CreateScheduleRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON format", nil)
		return
	}

	// Validate required fields
	if req.Name == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "schedule name is required", map[string]interface{}{
			"field":  "name",
			"reason": "missing required field",
		})
		return
	}

	if req.Cron == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "cron expression is required", map[string]interface{}{
			"field":  "cron",
			"reason": "missing required field",
		})
		return
	}

	// Set WorkflowName from URL path
	req.WorkflowName = workflowName

	// Set defaults
	if req.Timezone == "" {
		req.Timezone = "UTC"
	}
	if req.OverlapPolicy == "" {
		req.OverlapPolicy = "skip"
	}

	// Convert to manager request
	createReq := &schedule.CreateRequest{
		Name:          req.Name,
		WorkflowName:  req.WorkflowName,
		Cron:          req.Cron,
		Timezone:      req.Timezone,
		OverlapPolicy: req.OverlapPolicy,
		Vars:          req.Vars,
		Paused:        req.Paused,
	}

	// Call manager
	sch, err := h.manager.Create(r.Context(), createReq)
	if err != nil {
		if contains(err.Error(), "workflow not found") {
			writeError(w, r, http.StatusNotFound, "workflow_not_found", err.Error(), nil)
			return
		}
		if contains(err.Error(), "already exists") {
			writeError(w, r, http.StatusConflict, "schedule_exists", err.Error(), nil)
			return
		}
		if contains(err.Error(), "invalid cron") {
			writeError(w, r, http.StatusBadRequest, "invalid_cron", err.Error(), nil)
			return
		}
		if contains(err.Error(), "limit exceeded") {
			writeError(w, r, http.StatusTooManyRequests, "schedule_limit_exceeded", err.Error(), nil)
			return
		}
		h.logger.Error("Failed to create schedule", zap.Error(err), zap.String("schedule_name", req.Name))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to create schedule", nil)
		return
	}

	h.logger.Info("Schedule created",
		zap.String("schedule_id", sch.ID),
		zap.String("workflow_name", sch.WorkflowName),
		zap.String("cron", sch.Cron),
	)

	// Return response
	resp := convertToScheduleResponse(sch)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// ListSchedulesByWorkflow handles GET /v1/workflows/{name}/schedules
func (h *ScheduleHandlers) ListSchedulesByWorkflow(w http.ResponseWriter, r *http.Request) {
	// Extract workflow name from URL
	vars := mux.Vars(r)
	workflowName := vars["name"]

	// Call manager
	schedules, err := h.manager.ListByWorkflow(r.Context(), workflowName)
	if err != nil {
		h.logger.Error("Failed to list schedules by workflow",
			zap.Error(err),
			zap.String("workflow_name", workflowName),
		)
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to list schedules", nil)
		return
	}

	// Convert to response
	resp := ListSchedulesResponse{
		Schedules: make([]ScheduleResponse, len(schedules)),
		Total:     len(schedules),
		Page:      1,
		Limit:     len(schedules),
	}

	for i, sch := range schedules {
		resp.Schedules[i] = convertToScheduleResponse(sch)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// ListAllSchedules handles GET /v1/schedules
func (h *ScheduleHandlers) ListAllSchedules(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()
	status := query.Get("status")
	workflowName := query.Get("workflow_name")

	page, err := strconv.Atoi(query.Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(query.Get("limit"))
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		writeError(w, r, http.StatusBadRequest, "invalid_parameter", "Limit must be <= 100", nil)
		return
	}

	offset := (page - 1) * limit

	// Build filter
	filter := &schedule.Filter{
		Status:       status,
		WorkflowName: workflowName,
		Limit:        limit,
		Offset:       offset,
	}

	// Call manager
	schedules, total, err := h.manager.List(r.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list schedules", zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to list schedules", nil)
		return
	}

	// Convert to response
	resp := ListSchedulesResponse{
		Schedules: make([]ScheduleResponse, len(schedules)),
		Total:     total,
		Page:      page,
		Limit:     limit,
	}

	for i, sch := range schedules {
		resp.Schedules[i] = convertToScheduleResponse(sch)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// GetSchedule handles GET /v1/workflows/{name}/schedules/{schedule_id}
func (h *ScheduleHandlers) GetSchedule(w http.ResponseWriter, r *http.Request) {
	// Extract parameters from URL
	vars := mux.Vars(r)
	scheduleID := vars["schedule_id"]

	// Call manager
	sch, err := h.manager.Get(r.Context(), scheduleID)
	if err != nil {
		if contains(err.Error(), "not found") {
			writeError(w, r, http.StatusNotFound, "schedule_not_found", err.Error(), nil)
			return
		}
		h.logger.Error("Failed to get schedule", zap.Error(err), zap.String("schedule_id", scheduleID))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to get schedule", nil)
		return
	}

	// Return response
	resp := convertToScheduleResponse(sch)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// UpdateSchedule handles PUT /v1/workflows/{name}/schedules/{schedule_id}
func (h *ScheduleHandlers) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	// Extract parameters from URL
	vars := mux.Vars(r)
	scheduleID := vars["schedule_id"]

	// Parse request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Failed to read request body", nil)
		return
	}
	defer r.Body.Close()

	var req UpdateScheduleRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON format", nil)
		return
	}

	// Validate at least one field is provided
	if req.Cron == "" && req.Timezone == "" && req.OverlapPolicy == "" && req.Vars == nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "At least one field must be provided for update", nil)
		return
	}

	// Convert to manager request
	updateReq := &schedule.UpdateRequest{
		Cron:          req.Cron,
		Timezone:      req.Timezone,
		OverlapPolicy: req.OverlapPolicy,
		Vars:          req.Vars,
	}

	// Call manager
	sch, err := h.manager.Update(r.Context(), scheduleID, updateReq)
	if err != nil {
		if contains(err.Error(), "not found") {
			writeError(w, r, http.StatusNotFound, "schedule_not_found", err.Error(), nil)
			return
		}
		if contains(err.Error(), "invalid cron") {
			writeError(w, r, http.StatusBadRequest, "invalid_cron", err.Error(), nil)
			return
		}
		h.logger.Error("Failed to update schedule", zap.Error(err), zap.String("schedule_id", scheduleID))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to update schedule", nil)
		return
	}

	h.logger.Info("Schedule updated",
		zap.String("schedule_id", scheduleID),
	)

	// Return response
	resp := convertToScheduleResponse(sch)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// DeleteSchedule handles DELETE /v1/workflows/{name}/schedules/{schedule_id}
func (h *ScheduleHandlers) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	// Extract parameters from URL
	vars := mux.Vars(r)
	scheduleID := vars["schedule_id"]

	// Call manager
	err := h.manager.Delete(r.Context(), scheduleID)
	if err != nil {
		if contains(err.Error(), "not found") {
			writeError(w, r, http.StatusNotFound, "schedule_not_found", err.Error(), nil)
			return
		}
		h.logger.Error("Failed to delete schedule", zap.Error(err), zap.String("schedule_id", scheduleID))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to delete schedule", nil)
		return
	}

	h.logger.Info("Schedule deleted",
		zap.String("schedule_id", scheduleID),
	)

	// Return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// PauseSchedule handles POST /v1/workflows/{name}/schedules/{schedule_id}/pause
func (h *ScheduleHandlers) PauseSchedule(w http.ResponseWriter, r *http.Request) {
	// Extract parameters from URL
	vars := mux.Vars(r)
	scheduleID := vars["schedule_id"]

	// Parse optional request body
	var req PauseScheduleRequest
	body, err := io.ReadAll(r.Body)
	if err == nil && len(body) > 0 {
		json.Unmarshal(body, &req)
	}
	defer r.Body.Close()

	if req.Reason == "" {
		req.Reason = "Paused via API"
	}

	// Call manager
	err = h.manager.Pause(r.Context(), scheduleID, req.Reason)
	if err != nil {
		if contains(err.Error(), "not found") {
			writeError(w, r, http.StatusNotFound, "schedule_not_found", err.Error(), nil)
			return
		}
		h.logger.Error("Failed to pause schedule", zap.Error(err), zap.String("schedule_id", scheduleID))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to pause schedule", nil)
		return
	}

	h.logger.Info("Schedule paused",
		zap.String("schedule_id", scheduleID),
		zap.String("reason", req.Reason),
	)

	// Return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// ResumeSchedule handles POST /v1/workflows/{name}/schedules/{schedule_id}/resume
func (h *ScheduleHandlers) ResumeSchedule(w http.ResponseWriter, r *http.Request) {
	// Extract parameters from URL
	vars := mux.Vars(r)
	scheduleID := vars["schedule_id"]

	// Parse optional request body
	var req ResumeScheduleRequest
	body, err := io.ReadAll(r.Body)
	if err == nil && len(body) > 0 {
		json.Unmarshal(body, &req)
	}
	defer r.Body.Close()

	if req.Reason == "" {
		req.Reason = "Resumed via API"
	}

	// Call manager
	err = h.manager.Resume(r.Context(), scheduleID, req.Reason)
	if err != nil {
		if contains(err.Error(), "not found") {
			writeError(w, r, http.StatusNotFound, "schedule_not_found", err.Error(), nil)
			return
		}
		h.logger.Error("Failed to resume schedule", zap.Error(err), zap.String("schedule_id", scheduleID))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to resume schedule", nil)
		return
	}

	h.logger.Info("Schedule resumed",
		zap.String("schedule_id", scheduleID),
		zap.String("reason", req.Reason),
	)

	// Return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// TriggerSchedule handles POST /v1/workflows/{name}/schedules/{schedule_id}/trigger
func (h *ScheduleHandlers) TriggerSchedule(w http.ResponseWriter, r *http.Request) {
	// Extract parameters from URL
	vars := mux.Vars(r)
	scheduleID := vars["schedule_id"]

	// Parse optional request body
	var req TriggerScheduleRequest
	body, err := io.ReadAll(r.Body)
	if err == nil && len(body) > 0 {
		json.Unmarshal(body, &req)
	}
	defer r.Body.Close()

	if req.OverlapPolicy == "" {
		req.OverlapPolicy = "allow_all"
	}

	// Parse overlap policy
	overlapPolicy := parseOverlapPolicy(req.OverlapPolicy)

	// Call manager with vars support
	workflowID, err := h.manager.Trigger(r.Context(), scheduleID, overlapPolicy, req.Vars)
	if err != nil {
		if contains(err.Error(), "not found") {
			writeError(w, r, http.StatusNotFound, "schedule_not_found", err.Error(), nil)
			return
		}
		h.logger.Error("Failed to trigger schedule", zap.Error(err), zap.String("schedule_id", scheduleID))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to trigger schedule", nil)
		return
	}

	h.logger.Info("Schedule triggered",
		zap.String("schedule_id", scheduleID),
		zap.String("workflow_id", workflowID),
	)

	// Return response
	resp := TriggerScheduleResponse{
		ScheduleID:  scheduleID,
		WorkflowID:  workflowID,
		TriggeredAt: time.Now().UTC().Format(time.RFC3339),
		Status:      "running",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// convertToScheduleResponse converts schedule.Schedule to ScheduleResponse
func convertToScheduleResponse(sch *schedule.Schedule) ScheduleResponse {
	resp := ScheduleResponse{
		ScheduleID:    sch.ID,
		WorkflowName:  sch.WorkflowName,
		Cron:          sch.Cron,
		Timezone:      sch.Timezone,
		OverlapPolicy: sch.OverlapPolicy,
		Status:        sch.Status,
		Vars:          sch.Vars,
		LastRunID:     sch.LastRunID,
		LastRunStatus: sch.LastRunStatus,
		TotalRuns:     sch.TotalRuns,
		Note:          sch.Note,
	}

	if !sch.NextRunTime.IsZero() {
		resp.NextRunTime = sch.NextRunTime.Format(time.RFC3339)
	}
	if !sch.LastRunTime.IsZero() {
		resp.LastRunTime = sch.LastRunTime.Format(time.RFC3339)
	}
	if !sch.CreatedAt.IsZero() {
		resp.CreatedAt = sch.CreatedAt.Format(time.RFC3339)
	}
	if !sch.UpdatedAt.IsZero() {
		resp.UpdatedAt = sch.UpdatedAt.Format(time.RFC3339)
	}

	return resp
}

// parseOverlapPolicy converts string to Temporal ScheduleOverlapPolicy enum
func parseOverlapPolicy(policy string) enumspb.ScheduleOverlapPolicy {
	switch policy {
	case "skip":
		return enumspb.SCHEDULE_OVERLAP_POLICY_SKIP
	case "buffer_one":
		return enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE
	case "buffer_all":
		return enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ALL
	case "cancel_other":
		return enumspb.SCHEDULE_OVERLAP_POLICY_CANCEL_OTHER
	case "terminate_other":
		return enumspb.SCHEDULE_OVERLAP_POLICY_TERMINATE_OTHER
	case "allow_all":
		return enumspb.SCHEDULE_OVERLAP_POLICY_ALLOW_ALL
	default:
		return enumspb.SCHEDULE_OVERLAP_POLICY_SKIP
	}
}
