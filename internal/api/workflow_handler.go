package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Websoft9/waterflow/pkg/audit"
	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/Websoft9/waterflow/pkg/errors"
	"github.com/Websoft9/waterflow/pkg/events"
	"github.com/Websoft9/waterflow/pkg/metrics"
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
	logger          *zap.Logger
	parser          *dsl.Parser
	validator       *dsl.Validator
	temporalClient  *temporal.Client
	historyParser   *temporal.HistoryParser
	workflowTracker *metrics.WorkflowTracker
	eventDispatcher *events.EventDispatcher
	workflowMonitor *events.WorkflowMonitor
	auditLogger     audit.AuditLogger
}

// NewWorkflowHandlers creates new WorkflowHandlers instance
func NewWorkflowHandlers(logger *zap.Logger, temporalClient *temporal.Client, eventDispatcher *events.EventDispatcher) *WorkflowHandlers {
	validator, err := dsl.NewValidator(logger)
	if err != nil {
		logger.Error("Failed to create validator", zap.Error(err))
		validator = nil
	}

	return &WorkflowHandlers{
		logger:          logger,
		parser:          dsl.NewParser(logger),
		validator:       validator,
		temporalClient:  temporalClient,
		historyParser:   temporal.NewHistoryParser(),
		workflowTracker: metrics.NewWorkflowTracker(),
		eventDispatcher: eventDispatcher,
		workflowMonitor: events.NewWorkflowMonitor(temporalClient, eventDispatcher, logger),
		auditLogger:     nil, // Will be set by SetAuditLogger if needed
	}
}

// SetAuditLogger sets the audit logger for workflow operations
func (h *WorkflowHandlers) SetAuditLogger(logger audit.AuditLogger) {
	h.auditLogger = logger
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
		h.writeErrorLegacy(w, r, http.StatusBadRequest, "invalid_request", "Failed to read request body", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	defer func() { _ = r.Body.Close() }()

	// Parse JSON request
	var req SubmitWorkflowRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.writeErrorLegacy(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON format", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	if req.YAML == "" {
		h.writeErrorLegacy(w, r, http.StatusBadRequest, "invalid_request", "Request body is required", map[string]interface{}{
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
		h.writeErrorLegacy(w, r, http.StatusUnprocessableEntity, "validation_error", "YAML parsing failed", details)
		return
	}

	// 2. Validate workflow semantics (AC1 requirement)
	// CRITICAL: Must validate runs-on field and other semantic rules
	if h.validator == nil {
		// validator 初始化失败时，拒绝请求而非跳过验证（H2 修复）
		h.logger.Error("Validator not initialized, rejecting workflow submission")
		h.writeErrorLegacy(w, r, http.StatusInternalServerError, "server_error",
			"Workflow validator not available, please contact administrator", nil)
		return
	}
	if _, err := h.validator.ValidateYAML([]byte(req.YAML)); err != nil {
		// Validation errors are CRITICAL - reject the request
		h.logger.Error("Workflow validation failed", zap.Error(err))
		h.writeErrorLegacy(w, r, http.StatusUnprocessableEntity, "validation_error",
			"Workflow validation failed", map[string]interface{}{
				"error": err.Error(),
			})
		return
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

	// 5. 父工作流 RunWorkflowExecutor 必须运行在 Server 自身的 Temporal 队列上（H1 修复）
	// 子 Job 的 Agent 路由由 workflow.go 内部根据 job.runs-on 分发，不在这里决定
	taskQueue := h.temporalClient.GetConfig().TaskQueue
	if taskQueue == "" {
		taskQueue = "waterflow-server" // 配置缺失时的安全默认值
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
		// Track submission failure
		h.workflowTracker.TrackSubmission(workflowID, false)

		// Audit workflow submission failure
		if h.auditLogger != nil {
			_ = h.auditLogger.Log(r.Context(), audit.NewAuditLogEntry(audit.EventWorkflowSubmit, audit.CategoryWorkflow).
				WithResource(&audit.ResourceContext{
					Type: "workflow",
					ID:   workflowID,
					Name: workflow.Name,
				}).
				WithAction("submit").
				WithResult(audit.ResultError).
				WithSeverity(audit.SeverityError).
				WithDetail("error", err.Error()))
		}

		h.writeErrorLegacy(w, r, http.StatusInternalServerError, "internal_error", "Failed to start workflow execution", nil)
		return
	}

	h.logger.Info("Workflow started successfully",
		zap.String("workflow_id", workflowID),
		zap.String("run_id", run.GetRunID()),
		zap.String("workflow_name", workflow.Name),
		zap.String("task_queue", taskQueue),
	)

	// Track workflow submission metrics
	h.workflowTracker.TrackSubmission(workflowID, true)

	// Dispatch workflow start event (asynchronous, non-blocking)
	startTime := time.Now()
	if h.eventDispatcher != nil {
		h.eventDispatcher.DispatchWorkflowStart(&events.WorkflowStartEvent{
			EventType:    "workflow.started",
			WorkflowID:   workflowID,
			Timestamp:    startTime,
			WorkflowName: workflow.Name,
			Metadata: map[string]interface{}{
				"source":     "api",
				"task_queue": taskQueue,
			},
		})
	}

	// Start monitoring workflow for completion/failure events
	if h.workflowMonitor != nil {
		h.workflowMonitor.MonitorWorkflow(workflowID, startTime)
	}

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

	// Audit workflow submission success
	if h.auditLogger != nil {
		_ = h.auditLogger.Log(r.Context(), audit.NewAuditLogEntry(audit.EventWorkflowSubmit, audit.CategoryWorkflow).
			WithResource(&audit.ResourceContext{
				Type: "workflow",
				ID:   workflowID,
				Name: workflow.Name,
			}).
			WithAction("submit").
			WithResult(audit.ResultSuccess).
			WithDetail("task_queue", taskQueue).
			WithDetail("run_id", run.GetRunID()))
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
		h.writeErrorLegacy(w, r, http.StatusBadRequest, "invalid_request", "Workflow ID is required", nil)
		return
	}

	// Query workflow status from Temporal
	desc, err := h.temporalClient.GetClient().DescribeWorkflowExecution(r.Context(), workflowID, "")
	if err != nil {
		h.logger.Error("Failed to describe workflow",
			zap.String("workflow_id", workflowID),
			zap.Error(err),
		)
		h.writeErrorLegacy(w, r, http.StatusNotFound, "not_found", "Workflow not found", map[string]interface{}{
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

// writeError writes RFC 7807 error response using pkg/errors
func (h *WorkflowHandlers) writeError(w http.ResponseWriter, r *http.Request, err error) {
	rfc7807 := errors.ToRFC7807(err, r.URL.Path)

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(rfc7807.Status)

	if encodeErr := json.NewEncoder(w).Encode(rfc7807); encodeErr != nil {
		h.logger.Error("Failed to encode error response", zap.Error(encodeErr))
	}
}

// writeErrorLegacy writes error response (legacy compatibility during migration)
func (h *WorkflowHandlers) writeErrorLegacy(w http.ResponseWriter, r *http.Request, status int, code string, message string, details interface{}) {
	// Create appropriate error type based on code
	var err error
	switch code {
	case "validation_error":
		err = errors.NewValidationError(message, nil)
	case "not_found":
		err = &errors.BaseError{Type: "not_found", Message: message, Retryable: false}
	case "invalid_request", "invalid_argument", "invalid_parameter":
		err = &errors.BaseError{Type: "invalid_argument", Message: message, Retryable: false}
	default:
		err = &errors.BaseError{Type: code, Message: message, Retryable: false}
	}

	if details != nil {
		if baseErr, ok := err.(*errors.BaseError); ok {
			baseErr.Context = map[string]interface{}{"details": details}
		}
	}

	h.writeError(w, r, err)
}

// ListWorkflows handles GET /v1/workflows endpoint (AC3)
func (h *WorkflowHandlers) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()
	page := parseIntParam(r, "page", 1)
	limit := parseIntParam(r, "limit", 20)

	// Validate parameters
	if page < 1 {
		h.writeErrorLegacy(w, r, http.StatusBadRequest, "invalid_parameter", "Invalid query parameter", map[string]interface{}{
			"field":  "page",
			"value":  page,
			"reason": "page must be >= 1",
		})
		return
	}

	if limit < 1 || limit > 100 {
		h.writeErrorLegacy(w, r, http.StatusBadRequest, "invalid_parameter", "Invalid query parameter", map[string]interface{}{
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
		h.writeErrorLegacy(w, r, http.StatusBadRequest, "invalid_request", "Workflow ID is required", nil)
		return
	}

	// 1. Check workflow exists and get status
	desc, err := h.temporalClient.GetClient().DescribeWorkflowExecution(r.Context(), workflowID, "")
	if err != nil {
		h.writeErrorLegacy(w, r, http.StatusNotFound, "not_found", "Workflow not found", map[string]interface{}{
			"workflow_id": workflowID,
		})
		return
	}

	// 2. Check status (only running workflows can be cancelled)
	status := mapTemporalStatus(desc.WorkflowExecutionInfo.Status)
	if status != "running" {
		h.writeErrorLegacy(w, r, http.StatusConflict, "conflict", "Cannot cancel completed workflow", map[string]interface{}{
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

		// Audit cancellation failure
		if h.auditLogger != nil {
			workflowName := ""
			if desc.WorkflowExecutionInfo != nil && desc.WorkflowExecutionInfo.Type != nil {
				workflowName = desc.WorkflowExecutionInfo.Type.Name
			}
			_ = h.auditLogger.Log(r.Context(), audit.NewAuditLogEntry(audit.EventWorkflowCancel, audit.CategoryWorkflow).
				WithResource(&audit.ResourceContext{
					Type: "workflow",
					ID:   workflowID,
					Name: workflowName,
				}).
				WithAction("cancel").
				WithResult(audit.ResultError).
				WithSeverity(audit.SeverityError).
				WithDetail("error", err.Error()))
		}

		h.writeErrorLegacy(w, r, http.StatusInternalServerError, "internal_error", "Failed to cancel workflow", nil)
		return
	}

	h.logger.Info("Workflow cancellation requested",
		zap.String("workflow_id", workflowID),
	)

	// Audit cancellation success
	if h.auditLogger != nil {
		workflowName := ""
		if desc.WorkflowExecutionInfo != nil && desc.WorkflowExecutionInfo.Type != nil {
			workflowName = desc.WorkflowExecutionInfo.Type.Name
		}
		_ = h.auditLogger.Log(r.Context(), audit.NewAuditLogEntry(audit.EventWorkflowCancel, audit.CategoryWorkflow).
			WithResource(&audit.ResourceContext{
				Type: "workflow",
				ID:   workflowID,
				Name: workflowName,
			}).
			WithAction("cancel").
			WithResult(audit.ResultSuccess))
	}

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

// TerminateWorkflowRequest represents terminate request body (AC8)
type TerminateWorkflowRequest struct {
	Reason string `json:"reason,omitempty"`
}

// TerminateWorkflow handles POST /v1/workflows/{id}/terminate endpoint (AC8)
func (h *WorkflowHandlers) TerminateWorkflow(w http.ResponseWriter, r *http.Request) {
	// Extract workflow ID from path
	vars := mux.Vars(r)
	workflowID := vars["id"]

	if workflowID == "" {
		h.writeErrorLegacy(w, r, http.StatusBadRequest, "invalid_request", "Workflow ID is required", nil)
		return
	}

	// Parse request body (reason is optional)
	var req TerminateWorkflowRequest
	if r.Body != nil {
		body, _ := io.ReadAll(r.Body)
		if len(body) > 0 {
			_ = json.Unmarshal(body, &req)
		}
		defer func() { _ = r.Body.Close() }()
	}

	// Terminate workflow via Temporal
	err := h.temporalClient.GetClient().TerminateWorkflow(
		r.Context(),
		workflowID,
		"", // runID empty = terminate current run
		req.Reason,
	)

	if err != nil {
		h.logger.Error("Failed to terminate workflow",
			zap.String("workflow_id", workflowID),
			zap.String("reason", req.Reason),
			zap.Error(err),
		)

		// Check if workflow not found
		if strings.Contains(err.Error(), "not found") {
			h.writeErrorLegacy(w, r, http.StatusNotFound, "not_found", "Workflow not found", map[string]interface{}{
				"workflow_id": workflowID,
			})
			return
		}

		h.writeErrorLegacy(w, r, http.StatusInternalServerError, "internal_error", "Failed to terminate workflow", nil)
		return
	}

	h.logger.Info("Workflow terminated successfully",
		zap.String("workflow_id", workflowID),
		zap.String("reason", req.Reason),
	)

	// AC8: Return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// RerunWorkflow handles POST /v1/workflows/{id}/rerun endpoint (AC6)
//
//nolint:gocyclo // Rerun workflow requires multiple validation and data retrieval steps
func (h *WorkflowHandlers) RerunWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	if workflowID == "" {
		h.writeErrorLegacy(w, r, http.StatusBadRequest, "invalid_request", "Workflow ID is required", nil)
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
		h.writeErrorLegacy(w, r, http.StatusNotFound, "not_found", "Workflow not found", map[string]interface{}{
			"workflow_id": workflowID,
		})
		return
	}

	// 2. Check status (only completed workflows can be rerun)
	status := mapTemporalStatus(desc.WorkflowExecutionInfo.Status)
	if status == "running" {
		h.writeErrorLegacy(w, r, http.StatusConflict, "conflict", "Cannot rerun running workflow", map[string]interface{}{
			"workflow_id":    workflowID,
			"current_status": status,
		})
		return
	}

	// 3. Get original YAML from workflow memo
	memo := desc.WorkflowExecutionInfo.Memo
	if memo == nil || memo.Fields == nil {
		h.writeErrorLegacy(w, r, http.StatusInternalServerError, "internal_error",
			"Original workflow YAML not found in memo", nil)
		return
	}

	originalYAMLPayload, ok := memo.Fields["original_yaml"]
	if !ok {
		h.writeErrorLegacy(w, r, http.StatusInternalServerError, "internal_error",
			"Original workflow YAML not found", nil)
		return
	}

	var originalYAML string
	dc := converter.GetDefaultDataConverter()
	if err := dc.FromPayload(originalYAMLPayload, &originalYAML); err != nil {
		h.logger.Error("Failed to unmarshal original YAML", zap.Error(err))
		h.writeErrorLegacy(w, r, http.StatusInternalServerError, "internal_error",
			"Failed to retrieve original YAML", nil)
		return
	}

	// 4. Parse original YAML
	workflow, err := h.parser.Parse([]byte(originalYAML))
	if err != nil {
		h.logger.Error("Failed to parse original YAML", zap.Error(err))
		h.writeErrorLegacy(w, r, http.StatusInternalServerError, "internal_error",
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

		// Audit rerun failure
		if h.auditLogger != nil {
			_ = h.auditLogger.Log(r.Context(), audit.NewAuditLogEntry(audit.EventWorkflowRerun, audit.CategoryWorkflow).
				WithResource(&audit.ResourceContext{
					Type: "workflow",
					ID:   workflowID,
					Name: workflow.Name,
				}).
				WithAction("rerun").
				WithResult(audit.ResultError).
				WithSeverity(audit.SeverityError).
				WithDetail("error", err.Error()))
		}

		h.writeErrorLegacy(w, r, http.StatusInternalServerError, "internal_error",
			"Failed to start workflow rerun", nil)
		return
	}

	h.logger.Info("Workflow rerun started",
		zap.String("original_id", workflowID),
		zap.String("new_id", newWorkflowID),
	)

	// Audit rerun success
	if h.auditLogger != nil {
		_ = h.auditLogger.Log(r.Context(), audit.NewAuditLogEntry(audit.EventWorkflowRerun, audit.CategoryWorkflow).
			WithResource(&audit.ResourceContext{
				Type: "workflow",
				ID:   newWorkflowID,
				Name: workflow.Name,
			}).
			WithAction("rerun").
			WithResult(audit.ResultSuccess).
			WithDetail("original_id", workflowID).
			WithDetail("run_id", run.GetRunID()))
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

// GetWorkflowLogs handles GET /v1/workflows/{id}/logs endpoint (AC4)
//
//nolint:gocyclo // Log filtering and processing requires multiple steps
func (h *WorkflowHandlers) GetWorkflowLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	if workflowID == "" {
		h.writeErrorLegacy(w, r, http.StatusBadRequest, "invalid_request", "Workflow ID is required", nil)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	level := query.Get("level")
	job := query.Get("job")
	step := query.Get("step")
	tail := parseIntParam(r, "tail", 100)

	// Validate tail parameter
	if tail < 1 || tail > 1000 {
		h.writeErrorLegacy(w, r, http.StatusBadRequest, "invalid_parameter", "Invalid query parameter", map[string]interface{}{
			"field":  "tail",
			"value":  tail,
			"reason": "tail must be between 1 and 1000",
		})
		return
	}

	// 1. Check workflow exists
	desc, err := h.temporalClient.GetClient().DescribeWorkflowExecution(r.Context(), workflowID, "")
	if err != nil {
		h.writeErrorLegacy(w, r, http.StatusNotFound, "not_found", "Workflow not found", map[string]interface{}{
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

// ListTaskQueues is a legacy placeholder kept for backward compatibility.
// The real implementation has moved to AgentHandlers.ListTaskQueues (Story 2.7).
// This method is no longer registered in the router.
func (h *WorkflowHandlers) ListTaskQueues(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"message":     "Task queue listing has moved to AgentHandlers (Story 2.7)",
		"hint":        "Use GET /v1/task-queues (backed by Temporal DescribeTaskQueue)",
		"task_queues": []map[string]interface{}{},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode task queues response", zap.Error(err))
	}
}
