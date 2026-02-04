package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Websoft9/waterflow/pkg/audit"
	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/Websoft9/waterflow/pkg/events"
	"github.com/Websoft9/waterflow/pkg/metrics"
	"github.com/Websoft9/waterflow/pkg/temporal"
	"github.com/Websoft9/waterflow/pkg/workflow"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// ExecutionHandlers handles workflow execution endpoints
type ExecutionHandlers struct {
	logger          *zap.Logger
	parser          *dsl.Parser
	validator       *dsl.Validator
	temporalClient  *temporal.Client
	defStore        workflow.DefinitionStore
	historyParser   *temporal.HistoryParser
	workflowTracker *metrics.WorkflowTracker
	eventDispatcher *events.EventDispatcher
	workflowMonitor *events.WorkflowMonitor
	auditLogger     audit.AuditLogger
}

// NewExecutionHandlers creates new ExecutionHandlers instance
func NewExecutionHandlers(logger *zap.Logger, temporalClient *temporal.Client, defStore workflow.DefinitionStore, eventDispatcher *events.EventDispatcher) *ExecutionHandlers {
	validator, err := dsl.NewValidator(logger)
	if err != nil {
		logger.Error("Failed to create validator", zap.Error(err))
		validator = nil
	}

	return &ExecutionHandlers{
		logger:          logger,
		parser:          dsl.NewParser(logger),
		validator:       validator,
		temporalClient:  temporalClient,
		defStore:        defStore,
		historyParser:   temporal.NewHistoryParser(),
		workflowTracker: metrics.NewWorkflowTracker(),
		eventDispatcher: eventDispatcher,
		workflowMonitor: events.NewWorkflowMonitor(temporalClient, eventDispatcher, logger),
		auditLogger:     nil,
	}
}

// SetAuditLogger sets the audit logger
func (h *ExecutionHandlers) SetAuditLogger(logger audit.AuditLogger) {
	h.auditLogger = logger
}

// ExecuteWorkflowDefinitionRequest represents execute definition request
type ExecuteWorkflowDefinitionRequest struct {
	Vars map[string]interface{} `json:"vars,omitempty"`
}

// ExecuteWorkflowDefinitionResponse represents execute response
type ExecuteWorkflowDefinitionResponse struct {
	ExecutionID  string `json:"execution_id"`
	WorkflowName string `json:"workflow_name"`
	Status       string `json:"status"`
	StartedAt    string `json:"started_at"`
	URL          string `json:"url"`
}

// ExecuteWorkflowDefinition handles POST /v1/workflows/{name}/run (AC6)
func (h *ExecutionHandlers) ExecuteWorkflowDefinition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowName := vars["name"]

	if workflowName == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Workflow name is required", nil)
		return
	}

	// Parse request
	var req ExecuteWorkflowDefinitionRequest
	if r.Body != nil {
		body, _ := io.ReadAll(r.Body)
		if len(body) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON format", nil)
				return
			}
		}
		defer r.Body.Close()
	}

	// 1. Load workflow definition from database
	def, err := h.defStore.Get(r.Context(), workflowName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, r, http.StatusNotFound, "not_found", fmt.Sprintf("Workflow definition '%s' not found", workflowName), nil)
			return
		}
		h.logger.Error("Failed to get workflow definition", zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to load workflow definition", nil)
		return
	}

	// 2. Parse YAML content
	wf, err := h.parser.Parse([]byte(def.Content))
	if err != nil {
		h.logger.Error("Failed to parse workflow YAML", zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to parse workflow definition", nil)
		return
	}

	// 3. Merge vars (priority: YAML vars < execution vars)
	if req.Vars != nil {
		if wf.Vars == nil {
			wf.Vars = make(map[string]interface{})
		}
		for k, v := range req.Vars {
			wf.Vars[k] = v
		}
	}

	// 4. Generate execution ID: {workflow_name}-{uuid} (full UUID for safety)
	executionID := workflowName + "-" + uuid.New().String()

	// 5. Determine task queue from runs-on
	taskQueue := "default"
	for _, job := range wf.Jobs {
		if job.RunsOn != "" {
			taskQueue = job.RunsOn
			break
		}
	}

	// 6. Submit to Temporal
	if h.temporalClient == nil || h.temporalClient.GetClient() == nil {
		h.logger.Error("Temporal client is not available")
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Workflow execution service unavailable", nil)
		return
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:                       executionID,
		TaskQueue:                taskQueue,
		WorkflowExecutionTimeout: 24 * time.Hour,
		Memo: map[string]interface{}{
			"workflow_name":  workflowName,
			"definition_run": true,
		},
	}

	run, err := h.temporalClient.GetClient().ExecuteWorkflow(
		r.Context(),
		workflowOptions,
		"RunWorkflowExecutor",
		wf,
	)
	if err != nil {
		h.logger.Error("Failed to start workflow execution",
			zap.String("execution_id", executionID),
			zap.Error(err),
		)

		if h.auditLogger != nil {
			_ = h.auditLogger.Log(r.Context(), audit.NewAuditLogEntry(audit.EventWorkflowSubmit, audit.CategoryWorkflow).
				WithResource(&audit.ResourceContext{Type: "workflow", ID: executionID, Name: workflowName}).
				WithAction("execute_definition").
				WithResult(audit.ResultError).
				WithDetail("error", err.Error()))
		}

		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to start workflow execution", nil)
		return
	}

	h.logger.Info("Workflow execution started from definition",
		zap.String("execution_id", executionID),
		zap.String("workflow_name", workflowName),
		zap.String("run_id", run.GetRunID()),
	)

	// Track metrics
	h.workflowTracker.TrackSubmission(executionID, true)

	// Dispatch events
	startTime := time.Now()
	if h.eventDispatcher != nil {
		h.eventDispatcher.DispatchWorkflowStart(&events.WorkflowStartEvent{
			EventType:    "workflow.started",
			WorkflowID:   executionID,
			Timestamp:    startTime,
			WorkflowName: wf.Name,
			Metadata:     map[string]interface{}{"source": "definition", "workflow_name": workflowName},
		})
	}

	if h.workflowMonitor != nil {
		h.workflowMonitor.MonitorWorkflow(executionID, startTime)
	}

	// Audit log
	if h.auditLogger != nil {
		_ = h.auditLogger.Log(r.Context(), audit.NewAuditLogEntry(audit.EventWorkflowSubmit, audit.CategoryWorkflow).
			WithResource(&audit.ResourceContext{Type: "workflow", ID: executionID, Name: workflowName}).
			WithAction("execute_definition").
			WithResult(audit.ResultSuccess).
			WithDetail("run_id", run.GetRunID()))
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(ExecuteWorkflowDefinitionResponse{
		ExecutionID:  executionID,
		WorkflowName: workflowName,
		Status:       "running",
		StartedAt:    startTime.Format(time.RFC3339),
		URL:          "/v1/executions/" + executionID,
	})
}

// DirectExecuteWorkflowRequest represents direct execution request
type DirectExecuteWorkflowRequest struct {
	Workflow string                 `json:"workflow"`
	Vars     map[string]interface{} `json:"vars,omitempty"`
}

// DirectExecuteWorkflowResponse represents direct execution response
type DirectExecuteWorkflowResponse struct {
	ExecutionID  string `json:"execution_id"`
	WorkflowName string `json:"workflow_name"`
	Status       string `json:"status"`
	StartedAt    string `json:"started_at"`
	URL          string `json:"url"`
}

// DirectExecuteWorkflow handles POST /v1/executions (AC7)
func (h *ExecutionHandlers) DirectExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	// Parse request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Failed to read request body", nil)
		return
	}
	defer r.Body.Close()

	var req DirectExecuteWorkflowRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON format", nil)
		return
	}

	if req.Workflow == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Workflow YAML is required", nil)
		return
	}

	// Parse YAML
	wf, err := h.parser.Parse([]byte(req.Workflow))
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, "validation_error", "YAML validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Validate workflow
	if h.validator != nil {
		if _, err := h.validator.ValidateYAML([]byte(req.Workflow)); err != nil {
			writeError(w, r, http.StatusUnprocessableEntity, "validation_error", "YAML validation failed", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}
	}

	// Merge vars
	if req.Vars != nil {
		if wf.Vars == nil {
			wf.Vars = make(map[string]interface{})
		}
		for k, v := range req.Vars {
			wf.Vars[k] = v
		}
	}

	// Generate execution ID (full UUID for safety)
	executionID := wf.Name + "-" + uuid.New().String()

	// Determine task queue
	taskQueue := "default"
	for _, job := range wf.Jobs {
		if job.RunsOn != "" {
			taskQueue = job.RunsOn
			break
		}
	}

	// Submit to Temporal
	if h.temporalClient == nil || h.temporalClient.GetClient() == nil {
		h.logger.Error("Temporal client is not available")
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Workflow execution service unavailable", nil)
		return
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:                       executionID,
		TaskQueue:                taskQueue,
		WorkflowExecutionTimeout: 24 * time.Hour,
		Memo: map[string]interface{}{
			"direct_execution": true,
		},
	}

	_, err = h.temporalClient.GetClient().ExecuteWorkflow(
		r.Context(),
		workflowOptions,
		"RunWorkflowExecutor",
		wf,
	)
	if err != nil {
		h.logger.Error("Failed to start direct execution",
			zap.String("execution_id", executionID),
			zap.Error(err),
		)
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to start workflow execution", nil)
		return
	}

	h.logger.Info("Direct workflow execution started",
		zap.String("execution_id", executionID),
		zap.String("workflow_name", wf.Name),
	)

	// Track and dispatch
	h.workflowTracker.TrackSubmission(executionID, true)
	startTime := time.Now()

	if h.eventDispatcher != nil {
		h.eventDispatcher.DispatchWorkflowStart(&events.WorkflowStartEvent{
			EventType:    "workflow.started",
			WorkflowID:   executionID,
			Timestamp:    startTime,
			WorkflowName: wf.Name,
			Metadata:     map[string]interface{}{"source": "direct"},
		})
	}

	if h.workflowMonitor != nil {
		h.workflowMonitor.MonitorWorkflow(executionID, startTime)
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(DirectExecuteWorkflowResponse{
		ExecutionID:  executionID,
		WorkflowName: wf.Name,
		Status:       "running",
		StartedAt:    startTime.Format(time.RFC3339),
		URL:          "/v1/executions/" + executionID,
	})
}

// ExecutionStatusResponse represents execution status response
type ExecutionStatusResponse struct {
	ExecutionID     string                 `json:"execution_id"`
	WorkflowName    string                 `json:"workflow_name"`
	Status          string                 `json:"status"`
	CreatedAt       string                 `json:"created_at"`
	StartedAt       string                 `json:"started_at,omitempty"`
	CompletedAt     string                 `json:"completed_at,omitempty"`
	DurationSeconds *int                   `json:"duration_seconds,omitempty"`
	Vars            map[string]interface{} `json:"vars,omitempty"`
	Jobs            []JobStatus            `json:"jobs,omitempty"`
}

// GetExecutionStatus handles GET /v1/executions/{id} (AC8)
func (h *ExecutionHandlers) GetExecutionStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	executionID := vars["id"]

	if executionID == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Execution ID is required", nil)
		return
	}

	// Check Temporal client availability
	if h.temporalClient == nil || h.temporalClient.GetClient() == nil {
		h.logger.Error("Temporal client is not available")
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Execution service unavailable", nil)
		return
	}

	// Query from Temporal
	if h.temporalClient == nil || h.temporalClient.GetClient() == nil {
		h.logger.Error("Temporal client is not available")
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Workflow execution service unavailable", nil)
		return
	}

	desc, err := h.temporalClient.GetClient().DescribeWorkflowExecution(r.Context(), executionID, "")
	if err != nil {
		h.logger.Error("Failed to describe execution", zap.String("execution_id", executionID), zap.Error(err))
		writeError(w, r, http.StatusNotFound, "not_found", "Execution not found", nil)
		return
	}

	info := desc.WorkflowExecutionInfo
	status := mapTemporalStatus(info.Status)

	// Build response
	response := ExecutionStatusResponse{
		ExecutionID:  executionID,
		WorkflowName: info.Type.Name,
		Status:       status,
	}

	if info.StartTime != nil {
		response.CreatedAt = info.StartTime.AsTime().Format(time.RFC3339)
		response.StartedAt = info.StartTime.AsTime().Format(time.RFC3339)
	}

	if info.CloseTime != nil {
		response.CompletedAt = info.CloseTime.AsTime().Format(time.RFC3339)
		if info.StartTime != nil {
			duration := int(info.CloseTime.AsTime().Sub(info.StartTime.AsTime()).Seconds())
			response.DurationSeconds = &duration
		}
	}

	// Parse jobs from history
	historyIter := h.temporalClient.GetClient().GetWorkflowHistory(r.Context(), executionID, info.Execution.RunId, false, 0)
	var events []*history.HistoryEvent
	for historyIter.HasNext() {
		event, err := historyIter.Next()
		if err != nil {
			break
		}
		events = append(events, event)
	}

	jobs := h.historyParser.ParseJobsFromHistory(events)
	if len(jobs) > 0 {
		response.Jobs = make([]JobStatus, len(jobs))
		for i, job := range jobs {
			response.Jobs[i] = JobStatus{
				ID:          job.ID,
				Name:        job.Name,
				Status:      job.Status,
				StartedAt:   job.StartTime,
				CompletedAt: job.EndTime,
			}

			if len(job.Steps) > 0 {
				response.Jobs[i].Steps = make([]StepStatus, len(job.Steps))
				for j, step := range job.Steps {
					response.Jobs[i].Steps[j] = StepStatus{
						Name:        step.Name,
						Status:      step.Status,
						StartedAt:   step.StartTime,
						CompletedAt: step.EndTime,
					}
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ListExecutionsResponse represents list executions response
type ListExecutionsResponse struct {
	Executions []ExecutionSummary `json:"executions"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
}

// ExecutionSummary represents execution summary
type ExecutionSummary struct {
	ExecutionID     string `json:"execution_id"`
	WorkflowName    string `json:"workflow_name"`
	Status          string `json:"status"`
	Conclusion      string `json:"conclusion,omitempty"`
	StartedAt       string `json:"started_at"`
	CompletedAt     string `json:"completed_at,omitempty"`
	DurationSeconds *int   `json:"duration_seconds,omitempty"`
}

// ListExecutions handles GET /v1/executions (AC9)
func (h *ExecutionHandlers) ListExecutions(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()
	workflowName := query.Get("workflow_name")
	statusFilter := query.Get("status")

	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Build Temporal Visibility query
	var queryParts []string
	if workflowName != "" {
		queryParts = append(queryParts, fmt.Sprintf("WorkflowType = '%s'", workflowName))
	}
	if statusFilter != "" {
		statuses := strings.Split(statusFilter, ",")
		var statusConds []string
		for _, s := range statuses {
			statusConds = append(statusConds, fmt.Sprintf("ExecutionStatus = '%s'", strings.ToUpper(s)))
		}
		if len(statusConds) > 0 {
			queryParts = append(queryParts, "("+strings.Join(statusConds, " OR ")+")")
		}
	}

	visibilityQuery := strings.Join(queryParts, " AND ")

	// Check Temporal client availability
	if h.temporalClient == nil || h.temporalClient.GetClient() == nil {
		h.logger.Error("Temporal client is not available")
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Execution service unavailable", nil)
		return
	}

	// Query Temporal
	listResp, err := h.temporalClient.GetClient().ListWorkflow(r.Context(), &workflowservice.ListWorkflowExecutionsRequest{
		Namespace: h.temporalClient.GetConfig().Namespace,
		PageSize:  int32(limit),
		Query:     visibilityQuery,
	})

	var executions []ExecutionSummary
	if err != nil {
		h.logger.Warn("Failed to list executions", zap.Error(err))
		executions = []ExecutionSummary{}
	} else {
		executions = make([]ExecutionSummary, 0, len(listResp.GetExecutions()))
		for _, exec := range listResp.GetExecutions() {
			summary := ExecutionSummary{
				ExecutionID:  exec.Execution.WorkflowId,
				WorkflowName: exec.Type.Name,
				Status:       mapTemporalStatus(exec.Status),
			}

			if exec.StartTime != nil {
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

			executions = append(executions, summary)
		}
	}

	total := len(executions)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ListExecutionsResponse{
		Executions: executions,
		Total:      total,
		Page:       page,
		Limit:      limit,
	})
}

// GetExecutionLogs handles GET /v1/executions/{id}/logs (AC10)
func (h *ExecutionHandlers) GetExecutionLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	executionID := vars["id"]

	if executionID == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Execution ID is required", nil)
		return
	}

	// Check Temporal client availability
	if h.temporalClient == nil || h.temporalClient.GetClient() == nil {
		h.logger.Error("Temporal client is not available")
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Execution service unavailable", nil)
		return
	}

	// Parse filters
	query := r.URL.Query()
	level := query.Get("level")
	job := query.Get("job")
	step := query.Get("step")
	tail, _ := strconv.Atoi(query.Get("tail"))
	if tail < 1 || tail > 1000 {
		tail = 100
	}

	// Check execution exists
	desc, err := h.temporalClient.GetClient().DescribeWorkflowExecution(r.Context(), executionID, "")
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Execution not found", nil)
		return
	}

	// Get event history
	historyIter := h.temporalClient.GetClient().GetWorkflowHistory(
		r.Context(),
		executionID,
		desc.WorkflowExecutionInfo.Execution.RunId,
		false,
		0,
	)

	var events []*history.HistoryEvent
	for historyIter.HasNext() {
		event, err := historyIter.Next()
		if err != nil {
			break
		}
		events = append(events, event)
	}

	// Rebuild logs
	logs := make([]map[string]interface{}, 0)
	for _, event := range events {
		logEntry := extractLogFromEvent(event, events)
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

	// Apply tail
	if len(logs) > tail {
		logs = logs[len(logs)-tail:]
	}

	// Return JSON Lines format
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)
	for _, log := range logs {
		_ = enc.Encode(log)
	}
}

// CancelExecution handles POST /v1/executions/{id}/cancel (AC11)
func (h *ExecutionHandlers) CancelExecution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	executionID := vars["id"]

	if executionID == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Execution ID is required", nil)
		return
	}

	// Check Temporal client availability
	if h.temporalClient == nil || h.temporalClient.GetClient() == nil {
		h.logger.Error("Temporal client is not available")
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Execution service unavailable", nil)
		return
	}

	// Check execution exists and status
	desc, err := h.temporalClient.GetClient().DescribeWorkflowExecution(r.Context(), executionID, "")
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Execution not found", nil)
		return
	}

	status := mapTemporalStatus(desc.WorkflowExecutionInfo.Status)
	if status != "running" {
		writeError(w, r, http.StatusConflict, "conflict", "Cannot cancel non-running execution", map[string]interface{}{
			"execution_id":   executionID,
			"current_status": status,
		})
		return
	}

	// Cancel
	err = h.temporalClient.GetClient().CancelWorkflow(r.Context(), executionID, "")
	if err != nil {
		h.logger.Error("Failed to cancel execution", zap.String("execution_id", executionID), zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to cancel execution", nil)
		return
	}

	h.logger.Info("Execution cancellation requested", zap.String("execution_id", executionID))

	// Audit log
	if h.auditLogger != nil {
		_ = h.auditLogger.Log(r.Context(), audit.NewAuditLogEntry(audit.EventWorkflowCancel, audit.CategoryWorkflow).
			WithResource(&audit.ResourceContext{Type: "workflow", ID: executionID}).
			WithAction("cancel").
			WithResult(audit.ResultSuccess))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"execution_id": executionID,
		"status":       "cancelling",
		"message":      "Execution cancellation requested",
	})
}

// TerminateExecutionRequest represents terminate request
type TerminateExecutionRequest struct {
	Reason string `json:"reason,omitempty"`
}

// TerminateExecution handles POST /v1/executions/{id}/terminate (AC12)
func (h *ExecutionHandlers) TerminateExecution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	executionID := vars["id"]

	if executionID == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Execution ID is required", nil)
		return
	}

	// Check Temporal client availability
	if h.temporalClient == nil || h.temporalClient.GetClient() == nil {
		h.logger.Error("Temporal client is not available")
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Execution service unavailable", nil)
		return
	}

	// Parse reason
	var req TerminateExecutionRequest
	if r.Body != nil {
		body, _ := io.ReadAll(r.Body)
		if len(body) > 0 {
			_ = json.Unmarshal(body, &req)
		}
		defer r.Body.Close()
	}

	// Terminate
	err := h.temporalClient.GetClient().TerminateWorkflow(r.Context(), executionID, "", req.Reason)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, r, http.StatusNotFound, "not_found", "Execution not found", nil)
			return
		}
		h.logger.Error("Failed to terminate execution", zap.String("execution_id", executionID), zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to terminate execution", nil)
		return
	}

	h.logger.Info("Execution terminated", zap.String("execution_id", executionID), zap.String("reason", req.Reason))

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions

func extractLogFromEvent(event *history.HistoryEvent, events []*history.HistoryEvent) map[string]interface{} {
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

	return map[string]interface{}{
		"timestamp": timestamp,
		"level":     level,
		"message":   message,
	}
}
