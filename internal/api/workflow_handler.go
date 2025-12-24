package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/Websoft9/waterflow/pkg/temporal"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/history/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// WorkflowHandlers handles workflow execution endpoints
type WorkflowHandlers struct {
	logger         *zap.Logger
	parser         *dsl.Parser
	temporalClient *temporal.Client
	historyParser  *temporal.HistoryParser
}

// NewWorkflowHandlers creates new WorkflowHandlers instance
func NewWorkflowHandlers(logger *zap.Logger, temporalClient *temporal.Client) *WorkflowHandlers {
	return &WorkflowHandlers{
		logger:         logger,
		parser:         dsl.NewParser(logger),
		temporalClient: temporalClient,
		historyParser:  temporal.NewHistoryParser(),
	}
}

// SubmitWorkflowRequest represents workflow submission request
type SubmitWorkflowRequest struct {
	YAML string                 `json:"yaml"`
	Vars map[string]interface{} `json:"vars,omitempty"`
}

// SubmitWorkflowResponse represents workflow submission response
type SubmitWorkflowResponse struct {
	ID        string `json:"id"`
	RunID     string `json:"run_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	URL       string `json:"url"`
}

// SubmitWorkflow handles POST /v1/workflows endpoint
func (h *WorkflowHandlers) SubmitWorkflow(w http.ResponseWriter, r *http.Request) {
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, r, http.StatusBadRequest, "invalid_request", "Failed to read request body", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	defer func() { _ = r.Body.Close() }()

	// Parse JSON request
	var req SubmitWorkflowRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON format", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	if req.YAML == "" {
		h.writeError(w, r, http.StatusBadRequest, "invalid_request", "Request body is required", map[string]interface{}{
			"field":  "yaml",
			"reason": "missing required field",
		})
		return
	}

	// 1. Parse and validate YAML
	workflow, err := h.parser.Parse([]byte(req.YAML))
	if err != nil {
		// Extract validation errors if available
		details := map[string]interface{}{
			"error": err.Error(),
		}
		h.writeError(w, r, http.StatusUnprocessableEntity, "validation_error", "YAML validation failed", details)
		return
	}

	// 2. Merge vars (request vars override YAML vars)
	if len(req.Vars) > 0 {
		if workflow.Vars == nil {
			workflow.Vars = make(map[string]interface{})
		}
		for k, v := range req.Vars {
			workflow.Vars[k] = v
		}
	}

	// 3. Generate workflow ID (UUID v4)
	workflowID := uuid.New().String()

	// 4. Determine Task Queue from runs-on (使用第一个 Job 的 runs-on)
	taskQueue := "default" // 默认队列
	for _, job := range workflow.Jobs {
		if job.RunsOn != "" {
			taskQueue = job.RunsOn
		}
		break // 当前只支持单 Job，使用第一个 Job 的配置
	}

	// 5. Start Temporal workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: taskQueue,
		// Workflow execution timeout (24 hours)
		WorkflowExecutionTimeout: 24 * time.Hour,
	}

	run, err := h.temporalClient.GetClient().ExecuteWorkflow(
		r.Context(),
		workflowOptions,
		"RunWorkflowExecutor",
		workflow,
	)
	if err != nil {
		h.logger.Error("Failed to start workflow",
			zap.String("workflow_id", workflowID),
			zap.Error(err),
		)
		h.writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to start workflow execution", nil)
		return
	}

	h.logger.Info("Workflow started successfully",
		zap.String("workflow_id", workflowID),
		zap.String("run_id", run.GetRunID()),
		zap.String("workflow_name", workflow.Name),
		zap.String("task_queue", taskQueue),
	)

	// Return workflow info with AC1 format
	createdAt := time.Now().UTC().Format(time.RFC3339)
	response := SubmitWorkflowResponse{
		ID:        workflowID,
		RunID:     run.GetRunID(),
		Name:      workflow.Name,
		Status:    "running",
		CreatedAt: createdAt,
		URL:       "/v1/workflows/" + workflowID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

// WorkflowStatus represents workflow execution status (AC2)
type WorkflowStatus struct {
	ID              string                 `json:"id"`
	RunID           string                 `json:"run_id"`
	Name            string                 `json:"name"`
	Status          string                 `json:"status"`
	Conclusion      string                 `json:"conclusion,omitempty"`
	CreatedAt       string                 `json:"created_at"`
	StartedAt       string                 `json:"started_at,omitempty"`
	CompletedAt     string                 `json:"completed_at,omitempty"`
	DurationSeconds *int                   `json:"duration_seconds,omitempty"`
	Vars            map[string]interface{} `json:"vars,omitempty"`
	Jobs            []JobStatus            `json:"jobs,omitempty"`
}

// JobStatus represents job execution status (AC2)
type JobStatus struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Status      string       `json:"status"`
	StartedAt   string       `json:"started_at,omitempty"`
	CompletedAt string       `json:"completed_at,omitempty"`
	RunsOn      string       `json:"runs_on,omitempty"`
	Steps       []StepStatus `json:"steps,omitempty"`
}

// StepStatus represents step execution status (AC2)
type StepStatus struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Conclusion  string `json:"conclusion,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
}

// GetWorkflowStatus handles GET /v1/workflows/{id} endpoint (AC2)
func (h *WorkflowHandlers) GetWorkflowStatus(w http.ResponseWriter, r *http.Request) {
	// Extract workflow ID from path using mux
	vars := mux.Vars(r)
	workflowID := vars["id"]

	if workflowID == "" {
		h.writeError(w, r, http.StatusBadRequest, "invalid_request", "Workflow ID is required", nil)
		return
	}

	// Query workflow status from Temporal
	desc, err := h.temporalClient.GetClient().DescribeWorkflowExecution(r.Context(), workflowID, "")
	if err != nil {
		h.logger.Error("Failed to describe workflow",
			zap.String("workflow_id", workflowID),
			zap.Error(err),
		)
		h.writeError(w, r, http.StatusNotFound, "not_found", "Workflow not found", map[string]interface{}{
			"workflow_id": workflowID,
		})
		return
	}

	info := desc.WorkflowExecutionInfo

	// Map Temporal status to our status
	status := mapTemporalStatus(info.Status)
	conclusion := mapConclusion(info.Status)

	// Build response
	response := WorkflowStatus{
		ID:         workflowID,
		RunID:      info.Execution.RunId,
		Name:       info.Type.Name,
		Status:     status,
		Conclusion: conclusion,
	}

	// Add timestamps
	if info.StartTime != nil {
		response.CreatedAt = info.StartTime.AsTime().Format("2006-01-02T15:04:05Z07:00")
		response.StartedAt = info.StartTime.AsTime().Format("2006-01-02T15:04:05Z07:00")
	}

	if info.CloseTime != nil {
		response.CompletedAt = info.CloseTime.AsTime().Format("2006-01-02T15:04:05Z07:00")

		// Calculate duration
		if info.StartTime != nil {
			duration := int(info.CloseTime.AsTime().Sub(info.StartTime.AsTime()).Seconds())
			response.DurationSeconds = &duration
		}
	}

	// Parse jobs from event history
	historyIter := h.temporalClient.GetClient().GetWorkflowHistory(
		r.Context(),
		workflowID,
		info.Execution.RunId,
		false,
		0,
	)

	// Collect all events
	var events []*history.HistoryEvent
	for historyIter.HasNext() {
		event, err := historyIter.Next()
		if err != nil {
			h.logger.Warn("Failed to read history event", zap.Error(err))
			break
		}
		events = append(events, event)
	}

	// Parse jobs from events
	jobs := h.historyParser.ParseJobsFromHistory(events)
	if len(jobs) > 0 {
		// Convert to API JobStatus format (AC2)
		response.Jobs = make([]JobStatus, len(jobs))
		for i, job := range jobs {
			response.Jobs[i] = JobStatus{
				ID:     job.ID,
				Name:   job.Name,
				Status: job.Status,
				// RunsOn: job.RunsOn,  // Not available from history parser
			}

			// Add steps if available
			if len(job.Steps) > 0 {
				response.Jobs[i].Steps = make([]StepStatus, len(job.Steps))
				for j, step := range job.Steps {
					response.Jobs[i].Steps[j] = StepStatus{
						Name:   step.Name,
						Status: step.Status,
					}
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// mapConclusion maps Temporal status to conclusion (AC2)
func mapConclusion(status enums.WorkflowExecutionStatus) string {
	switch status {
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		return "success"
	case enums.WORKFLOW_EXECUTION_STATUS_FAILED:
		return "failure"
	case enums.WORKFLOW_EXECUTION_STATUS_CANCELED:
		return "cancelled"
	case enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
		return "timeout"
	default:
		return ""
	}
}

// mapTemporalStatus maps Temporal workflow status to our status strings
func mapTemporalStatus(status enums.WorkflowExecutionStatus) string {
	switch status {
	case enums.WORKFLOW_EXECUTION_STATUS_RUNNING:
		return "running"
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		return "completed"
	case enums.WORKFLOW_EXECUTION_STATUS_FAILED:
		return "failed"
	case enums.WORKFLOW_EXECUTION_STATUS_CANCELED:
		return "cancelled"
	case enums.WORKFLOW_EXECUTION_STATUS_TERMINATED:
		return "terminated"
	case enums.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW:
		return "running"
	case enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
		return "timeout"
	default:
		return "unknown"
	}
}

// writeError writes unified error response (AC7 format)
func (h *WorkflowHandlers) writeError(w http.ResponseWriter, r *http.Request, status int, code string, message string, details interface{}) {
	errResp := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}

	if details != nil {
		errResp["error"].(map[string]interface{})["details"] = details
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		h.logger.Error("Failed to encode error response", zap.Error(err))
	}
}

// ListWorkflows handles GET /v1/workflows endpoint (AC3)
func (h *WorkflowHandlers) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()
	page := parseIntParam(query.Get("page"), 1)
	limit := parseIntParam(query.Get("limit"), 20)

	// Validate parameters
	if page < 1 {
		h.writeError(w, r, http.StatusBadRequest, "invalid_parameter", "Invalid query parameter", map[string]interface{}{
			"field":  "page",
			"value":  page,
			"reason": "page must be >= 1",
		})
		return
	}

	if limit < 1 || limit > 100 {
		h.writeError(w, r, http.StatusBadRequest, "invalid_parameter", "Invalid query parameter", map[string]interface{}{
			"field":  "limit",
			"value":  limit,
			"reason": "limit must be between 1 and 100",
		})
		return
	}

	// Note: Temporal ListWorkflow requires Visibility API
	// For MVP, return empty list with pagination info
	// Full implementation would query Temporal's Visibility API

	response := map[string]interface{}{
		"workflows": []interface{}{},
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       0,
			"total_pages": 0,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// CancelWorkflow handles POST /v1/workflows/{id}/cancel endpoint (AC5)
func (h *WorkflowHandlers) CancelWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	if workflowID == "" {
		h.writeError(w, r, http.StatusBadRequest, "invalid_request", "Workflow ID is required", nil)
		return
	}

	// 1. Check workflow exists and get status
	desc, err := h.temporalClient.GetClient().DescribeWorkflowExecution(r.Context(), workflowID, "")
	if err != nil {
		h.writeError(w, r, http.StatusNotFound, "not_found", "Workflow not found", map[string]interface{}{
			"workflow_id": workflowID,
		})
		return
	}

	// 2. Check status (only running workflows can be cancelled)
	status := mapTemporalStatus(desc.WorkflowExecutionInfo.Status)
	if status != "running" {
		h.writeError(w, r, http.StatusConflict, "conflict", "Cannot cancel completed workflow", map[string]interface{}{
			"workflow_id":    workflowID,
			"current_status": status,
		})
		return
	}

	// 3. Send cancel signal to Temporal
	err = h.temporalClient.GetClient().CancelWorkflow(r.Context(), workflowID, "")
	if err != nil {
		h.logger.Error("Failed to cancel workflow",
			zap.String("workflow_id", workflowID),
			zap.Error(err),
		)
		h.writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to cancel workflow", nil)
		return
	}

	h.logger.Info("Workflow cancellation requested",
		zap.String("workflow_id", workflowID),
	)

	// 4. Return 202 Accepted
	response := map[string]interface{}{
		"id":      workflowID,
		"status":  "cancelling",
		"message": "Workflow cancellation requested",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(response)
}

// RerunWorkflow handles POST /v1/workflows/{id}/rerun endpoint (AC6)
func (h *WorkflowHandlers) RerunWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	if workflowID == "" {
		h.writeError(w, r, http.StatusBadRequest, "invalid_request", "Workflow ID is required", nil)
		return
	}

	// Parse rerun request
	var req struct {
		Vars map[string]interface{} `json:"vars,omitempty"`
	}
	if r.Body != nil {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)
		defer func() { _ = r.Body.Close() }()
	}

	// 1. Get original workflow
	desc, err := h.temporalClient.GetClient().DescribeWorkflowExecution(r.Context(), workflowID, "")
	if err != nil {
		h.writeError(w, r, http.StatusNotFound, "not_found", "Workflow not found", map[string]interface{}{
			"workflow_id": workflowID,
		})
		return
	}

	// 2. Check status (only completed workflows can be rerun)
	status := mapTemporalStatus(desc.WorkflowExecutionInfo.Status)
	if status == "running" {
		h.writeError(w, r, http.StatusConflict, "conflict", "Cannot rerun running workflow", map[string]interface{}{
			"workflow_id":    workflowID,
			"current_status": status,
		})
		return
	}

	// 3. Note: In production, would fetch original workflow YAML from storage
	// For MVP, return error indicating feature not fully implemented
	h.writeError(w, r, http.StatusNotImplemented, "not_implemented",
		"Workflow rerun requires workflow storage (planned for future release)", nil)
}

// parseIntParam parses integer query parameter with default value
func parseIntParam(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	var val int
	if _, err := fmt.Sscanf(s, "%d", &val); err != nil {
		return defaultVal
	}
	return val
}

// GetWorkflowLogs handles GET /v1/workflows/{id}/logs endpoint (AC4)
//
//nolint:gocyclo // Log filtering and processing requires multiple steps
func (h *WorkflowHandlers) GetWorkflowLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	if workflowID == "" {
		h.writeError(w, r, http.StatusBadRequest, "invalid_request", "Workflow ID is required", nil)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	level := query.Get("level")
	job := query.Get("job")
	step := query.Get("step")
	tail := parseIntParam(query.Get("tail"), 100)

	// Validate tail parameter
	if tail < 1 || tail > 1000 {
		h.writeError(w, r, http.StatusBadRequest, "invalid_parameter", "Invalid query parameter", map[string]interface{}{
			"field":  "tail",
			"value":  tail,
			"reason": "tail must be between 1 and 1000",
		})
		return
	}

	// 1. Check workflow exists
	desc, err := h.temporalClient.GetClient().DescribeWorkflowExecution(r.Context(), workflowID, "")
	if err != nil {
		h.writeError(w, r, http.StatusNotFound, "not_found", "Workflow not found", map[string]interface{}{
			"workflow_id": workflowID,
		})
		return
	}

	// 2. Get event history
	historyIter := h.temporalClient.GetClient().GetWorkflowHistory(
		r.Context(),
		workflowID,
		desc.WorkflowExecutionInfo.Execution.RunId,
		false,
		0,
	)

	// 3. Rebuild logs from history
	logs := make([]map[string]interface{}, 0)
	for historyIter.HasNext() {
		event, err := historyIter.Next()
		if err != nil {
			break
		}

		// Extract log from event
		logEntry := h.extractLogFromEvent(event)
		if logEntry != nil {
			// Apply filters
			if level != "" && logEntry["level"] != level {
				continue
			}
			if job != "" && logEntry["job"] != job {
				continue
			}
			if step != "" && logEntry["step"] != step {
				continue
			}

			logs = append(logs, logEntry)
		}
	}

	// 4. Apply tail limit
	if len(logs) > tail {
		logs = logs[len(logs)-tail:]
	}

	// 5. Return JSON Lines format
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)
	for _, log := range logs {
		_ = enc.Encode(log)
	}
}

// extractLogFromEvent extracts log entry from Temporal event
func (h *WorkflowHandlers) extractLogFromEvent(event *history.HistoryEvent) map[string]interface{} {
	if event == nil || event.EventTime == nil {
		return nil
	}

	timestamp := event.EventTime.AsTime().Format(time.RFC3339)
	level := "info"
	var message string

	switch event.EventType {
	case enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED:
		message = "Workflow started"
	case enums.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED:
		message = "Workflow completed successfully"
	case enums.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED:
		level = "error"
		attrs := event.GetWorkflowExecutionFailedEventAttributes()
		if attrs != nil && attrs.Failure != nil {
			message = "Workflow failed: " + attrs.Failure.Message
		} else {
			message = "Workflow failed"
		}
	case enums.EVENT_TYPE_ACTIVITY_TASK_STARTED:
		message = "Step started"
	case enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
		message = "Step completed"
	case enums.EVENT_TYPE_ACTIVITY_TASK_FAILED:
		level = "error"
		attrs := event.GetActivityTaskFailedEventAttributes()
		if attrs != nil && attrs.Failure != nil {
			message = "Step failed: " + attrs.Failure.Message
		} else {
			message = "Step failed"
		}
	default:
		return nil
	}

	if message == "" {
		return nil
	}

	logEntry := map[string]interface{}{
		"timestamp": timestamp,
		"level":     level,
		"message":   message,
	}

	// Add job/step info if available
	// Note: Would need to parse from activity attributes in production
	// For now, this is simplified

	return logEntry
}
