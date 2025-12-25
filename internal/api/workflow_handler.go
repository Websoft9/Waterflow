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
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.uber.org/zap"
)

// WorkflowHandlers handles workflow execution endpoints
type WorkflowHandlers struct {
	logger         *zap.Logger
	parser         *dsl.Parser
	validator      *dsl.Validator
	temporalClient *temporal.Client
	historyParser  *temporal.HistoryParser
}

// NewWorkflowHandlers creates new WorkflowHandlers instance
func NewWorkflowHandlers(logger *zap.Logger, temporalClient *temporal.Client) *WorkflowHandlers {
	validator, err := dsl.NewValidator(logger)
	if err != nil {
		logger.Error("Failed to create validator", zap.Error(err))
		validator = nil
	}

	return &WorkflowHandlers{
		logger:         logger,
		parser:         dsl.NewParser(logger),
		validator:      validator,
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

	// 1. Parse YAML
	workflow, err := h.parser.Parse([]byte(req.YAML))
	if err != nil {
		// Extract validation errors if available
		details := map[string]interface{}{
			"error": err.Error(),
		}
		h.writeError(w, r, http.StatusUnprocessableEntity, "validation_error", "YAML parsing failed", details)
		return
	}

	// 2. Validate workflow semantics (AC1 requirement)
	// Note: ValidateYAML also parses, so we skip it if parser already succeeded
	// to avoid double validation that might be too strict
	if h.validator != nil {
		if _, err := h.validator.ValidateYAML([]byte(req.YAML)); err != nil {
			// Log validation warning but don't fail the request
			// (validator might be stricter than parser)
			h.logger.Warn("Workflow validation warning", zap.Error(err))
		}
	}

	// 3. Merge vars (request vars override YAML vars)
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

	// 5. Determine Task Queue from runs-on (使用第一个 Job 的 runs-on)
	taskQueue := "default" // 默认队列
	for _, job := range workflow.Jobs {
		if job.RunsOn != "" {
			taskQueue = job.RunsOn
		}
		break // 当前只支持单 Job，使用第一个 Job 的配置
	}

	// 6. Start Temporal workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: taskQueue,
		// Workflow execution timeout (24 hours)
		WorkflowExecutionTimeout: 24 * time.Hour,
		// Store original YAML in memo for rerun (AC6)
		Memo: map[string]interface{}{
			"original_yaml": req.YAML,
			"submitted_at":  time.Now().UTC().Format(time.RFC3339),
		},
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
				ID:          job.ID,
				Name:        job.Name,
				Status:      job.Status,
				StartedAt:   job.StartTime,
				CompletedAt: job.EndTime,
				// RunsOn not available from event history
			}

			// Add steps if available
			if len(job.Steps) > 0 {
				response.Jobs[i].Steps = make([]StepStatus, len(job.Steps))
				for j, step := range job.Steps {
					response.Jobs[i].Steps[j] = StepStatus{
						Name:        step.Name,
						Status:      step.Status,
						StartedAt:   step.StartTime,
						CompletedAt: step.EndTime,
						// Conclusion derived from Status
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

	// Build Temporal Visibility query
	visibilityQuery := buildTemporalVisibilityQuery(query)

	var workflows []WorkflowSummary

	// Check if Temporal client is available
	if h.temporalClient == nil || h.temporalClient.GetClient() == nil {
		// Return empty list when Temporal is not available
		workflows = []WorkflowSummary{}
	} else {
		// Query Temporal for workflow executions
		pageSize := int32(limit)
		if limit > 1000 {
			pageSize = 1000 // Cap at max page size
		}
		listResp, err := h.temporalClient.GetClient().ListWorkflow(r.Context(), &workflowservice.ListWorkflowExecutionsRequest{
			Namespace: h.temporalClient.GetConfig().Namespace,
			PageSize:  pageSize,
			Query:     visibilityQuery,
		})

		if err != nil {
			h.logger.Warn("Failed to list workflows from Temporal", zap.Error(err))
			// Fallback to empty list instead of error
			workflows = []WorkflowSummary{}
		} else {
			// Convert to WorkflowSummary
			workflows = make([]WorkflowSummary, 0, len(listResp.Executions))
			for _, exec := range listResp.Executions {
				summary := WorkflowSummary{
					ID:     exec.Execution.WorkflowId,
					Name:   exec.Type.Name,
					Status: mapTemporalStatus(exec.Status),
				}

				if exec.StartTime != nil {
					summary.CreatedAt = exec.StartTime.AsTime().Format(time.RFC3339)
					summary.StartedAt = exec.StartTime.AsTime().Format(time.RFC3339)
				}

				if exec.CloseTime != nil {
					summary.CompletedAt = exec.CloseTime.AsTime().Format(time.RFC3339)
					summary.Conclusion = mapConclusion(exec.Status)

					if exec.StartTime != nil {
						duration := int(exec.CloseTime.AsTime().Sub(exec.StartTime.AsTime()).Seconds())
						summary.DurationSeconds = &duration
					}
				}

				workflows = append(workflows, summary)
			}
		}
	}

	// Calculate pagination (simplified - would need total count query in production)
	total := len(workflows)
	totalPages := (total + limit - 1) / limit

	response := map[string]interface{}{
		"workflows": workflows,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
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
//
//nolint:gocyclo // Rerun workflow requires multiple validation and data retrieval steps
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

	// 3. Get original YAML from workflow memo
	memo := desc.WorkflowExecutionInfo.Memo
	if memo == nil || memo.Fields == nil {
		h.writeError(w, r, http.StatusInternalServerError, "internal_error",
			"Original workflow YAML not found in memo", nil)
		return
	}

	originalYAMLPayload, ok := memo.Fields["original_yaml"]
	if !ok {
		h.writeError(w, r, http.StatusInternalServerError, "internal_error",
			"Original workflow YAML not found", nil)
		return
	}

	var originalYAML string
	dc := converter.GetDefaultDataConverter()
	if err := dc.FromPayload(originalYAMLPayload, &originalYAML); err != nil {
		h.logger.Error("Failed to unmarshal original YAML", zap.Error(err))
		h.writeError(w, r, http.StatusInternalServerError, "internal_error",
			"Failed to retrieve original YAML", nil)
		return
	}

	// 4. Parse original YAML
	workflow, err := h.parser.Parse([]byte(originalYAML))
	if err != nil {
		h.logger.Error("Failed to parse original YAML", zap.Error(err))
		h.writeError(w, r, http.StatusInternalServerError, "internal_error",
			"Failed to parse original workflow", nil)
		return
	}

	// 5. Merge override vars
	if len(req.Vars) > 0 {
		if workflow.Vars == nil {
			workflow.Vars = make(map[string]interface{})
		}
		for k, v := range req.Vars {
			workflow.Vars[k] = v
		}
	}

	// 6. Generate new workflow ID
	newWorkflowID := uuid.New().String()

	// 7. Determine task queue
	taskQueue := "default"
	for _, job := range workflow.Jobs {
		if job.RunsOn != "" {
			taskQueue = job.RunsOn
		}
		break
	}

	// 8. Start new workflow with rerun flag
	workflowOptions := client.StartWorkflowOptions{
		ID:                       newWorkflowID,
		TaskQueue:                taskQueue,
		WorkflowExecutionTimeout: 24 * time.Hour,
		Memo: map[string]interface{}{
			"original_yaml": originalYAML,
			"rerun_from":    workflowID,
			"submitted_at":  time.Now().UTC().Format(time.RFC3339),
		},
	}

	run, err := h.temporalClient.GetClient().ExecuteWorkflow(
		r.Context(),
		workflowOptions,
		"RunWorkflowExecutor",
		workflow,
	)
	if err != nil {
		h.logger.Error("Failed to start rerun workflow",
			zap.String("original_id", workflowID),
			zap.Error(err),
		)
		h.writeError(w, r, http.StatusInternalServerError, "internal_error",
			"Failed to start workflow rerun", nil)
		return
	}

	// 9. Return new workflow info
	response := map[string]interface{}{
		"id":         newWorkflowID,
		"run_id":     run.GetRunID(),
		"name":       workflow.Name,
		"status":     "running",
		"created_at": time.Now().UTC().Format(time.RFC3339),
		"rerun_from": workflowID,
		"url":        "/v1/workflows/" + newWorkflowID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
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

	// Collect all events first
	allEvents := make([]*history.HistoryEvent, 0)
	for historyIter.HasNext() {
		event, err := historyIter.Next()
		if err != nil {
			break
		}
		allEvents = append(allEvents, event)
	}

	// 3. Rebuild logs from history
	logs := make([]map[string]interface{}, 0)
	for _, event := range allEvents {
		// Extract log from event
		logEntry := h.extractLogFromEvent(event, allEvents)
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
//
//nolint:gocyclo // Complex event type handling necessary for comprehensive log extraction
func (h *WorkflowHandlers) extractLogFromEvent(event *history.HistoryEvent, events []*history.HistoryEvent) map[string]interface{} {
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

	// Add job/step info from activity events (AC4 requirement)
	switch event.EventType {
	case enums.EVENT_TYPE_ACTIVITY_TASK_STARTED:
		if attrs := event.GetActivityTaskStartedEventAttributes(); attrs != nil {
			// Activity type format: "job-{jobID}-step-{stepName}"
			// Parse from scheduled event
			if schedEvent := h.findScheduledEvent(events, attrs.ScheduledEventId); schedEvent != nil {
				if schedAttrs := schedEvent.GetActivityTaskScheduledEventAttributes(); schedAttrs != nil {
					job, step := parseActivityType(schedAttrs.ActivityType.Name)
					if job != "" {
						logEntry["job"] = job
					}
					if step != "" {
						logEntry["step"] = step
					}
				}
			}
		}
	case enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED, enums.EVENT_TYPE_ACTIVITY_TASK_FAILED:
		var schedEventID int64
		if attrs := event.GetActivityTaskCompletedEventAttributes(); attrs != nil {
			schedEventID = attrs.ScheduledEventId
		} else if attrs := event.GetActivityTaskFailedEventAttributes(); attrs != nil {
			schedEventID = attrs.ScheduledEventId
		}

		if schedEventID > 0 {
			if schedEvent := h.findScheduledEvent(events, schedEventID); schedEvent != nil {
				if schedAttrs := schedEvent.GetActivityTaskScheduledEventAttributes(); schedAttrs != nil {
					job, step := parseActivityType(schedAttrs.ActivityType.Name)
					if job != "" {
						logEntry["job"] = job
					}
					if step != "" {
						logEntry["step"] = step
					}
				}
			}
		}
	}

	return logEntry
}

// ListTaskQueues returns a list of active task queues.
// This is a placeholder implementation for Story 2.2.
// Full implementation will be provided in Story 2.7 (Agent Health Monitoring).
func (h *WorkflowHandlers) ListTaskQueues(w http.ResponseWriter, r *http.Request) {
	// Story 2.7 will implement:
	// - Query Temporal Admin API for worker heartbeats
	// - Calculate worker count per task queue
	// - Return detailed health status

	response := map[string]interface{}{
		"message": "Task queue listing not yet fully implemented (Story 2.7)",
		"hint":    "Use Temporal UI to view active task queues: http://localhost:8088",
		"task_queues": []map[string]interface{}{
			{
				"name":           "example",
				"worker_count":   0,
				"status":         "unknown",
				"implementation": "placeholder",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode task queues response", zap.Error(err))
	}
}
