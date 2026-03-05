package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/Websoft9/waterflow/internal/server/schedule"
	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/Websoft9/waterflow/pkg/workflow"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// DefinitionHandlers handles workflow definition endpoints
type DefinitionHandlers struct {
	logger          *zap.Logger
	parser          *dsl.Parser
	validator       *dsl.Validator
	defStore        workflow.DefinitionStore
	paramExtractor  *workflow.ParameterExtractor
	scheduleManager *schedule.Manager
}

// NewDefinitionHandlers creates new DefinitionHandlers instance
func NewDefinitionHandlers(logger *zap.Logger, defStore workflow.DefinitionStore, scheduleManager *schedule.Manager) *DefinitionHandlers {
	validator, err := dsl.NewValidator(logger)
	if err != nil {
		logger.Error("Failed to create validator", zap.Error(err))
		validator = nil
	}

	return &DefinitionHandlers{
		logger:          logger,
		parser:          dsl.NewParser(logger),
		validator:       validator,
		defStore:        defStore,
		paramExtractor:  workflow.NewParameterExtractor(),
		scheduleManager: scheduleManager,
	}
}

// CreateWorkflowDefinitionRequest represents create definition request
type CreateWorkflowDefinitionRequest struct {
	Name        string                 `json:"name"`
	DisplayName string                 `json:"display_name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Category    string                 `json:"category,omitempty"`
	Tags        map[string]interface{} `json:"tags,omitempty"`
	Content     string                 `json:"content"`
}

// CreateWorkflowDefinitionResponse represents create definition response
type CreateWorkflowDefinitionResponse struct {
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Category    string               `json:"category,omitempty"`
	Parameters  []workflow.Parameter `json:"parameters,omitempty"`
	ContentHash string               `json:"content_hash,omitempty"`
	CreatedAt   string               `json:"created_at"`
	UpdatedAt   string               `json:"updated_at"`
}

// CreateWorkflowDefinition handles POST /v1/workflows
func (h *DefinitionHandlers) CreateWorkflowDefinition(w http.ResponseWriter, r *http.Request) {
	// Parse request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Failed to read request body", nil)
		return
	}
	defer r.Body.Close()

	var req CreateWorkflowDefinitionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON format", nil)
		return
	}

	// Validate required fields
	if req.Name == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Request body is required", map[string]interface{}{
			"field":  "name",
			"reason": "missing required field",
		})
		return
	}

	if req.Content == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Request body is required", map[string]interface{}{
			"field":  "content",
			"reason": "missing required field",
		})
		return
	}

	// Parse and validate YAML
	wf, err := h.parser.Parse([]byte(req.Content))
	if err != nil {
		// Extract detailed errors from ValidationError
		if validationErr, ok := err.(*dsl.ValidationError); ok {
			writeError(w, r, http.StatusUnprocessableEntity, "validation_error", "YAML validation failed", map[string]interface{}{
				"errors": validationErr.Errors,
			})
		} else {
			writeError(w, r, http.StatusUnprocessableEntity, "validation_error", "YAML validation failed", map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"error": err.Error(),
					},
				},
			})
		}
		return
	}

	// Validate workflow (semantic validation)
	if h.validator != nil {
		if _, err := h.validator.ValidateYAML([]byte(req.Content)); err != nil {
			// Extract detailed errors from ValidationError
			if validationErr, ok := err.(*dsl.ValidationError); ok {
				writeError(w, r, http.StatusUnprocessableEntity, "validation_error", "YAML validation failed", map[string]interface{}{
					"errors": validationErr.Errors,
				})
			} else {
				writeError(w, r, http.StatusUnprocessableEntity, "validation_error", "YAML validation failed", map[string]interface{}{
					"errors": []map[string]interface{}{
						{
							"error": err.Error(),
						},
					},
				})
			}
			return
		}
	}

	// Extract parameters
	params := h.paramExtractor.ExtractParameters(wf)

	// Create definition
	def := &workflow.WorkflowDefinition{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Category:    req.Category,
		TagsData:    req.Tags,
		ParamsData:  params,
		Content:     req.Content,
	}

	err = h.defStore.Create(r.Context(), def)
	if err != nil {
		if contains(err.Error(), "already exists") {
			writeError(w, r, http.StatusConflict, "conflict", err.Error(), nil)
			return
		}
		h.logger.Error("Failed to create workflow definition", zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to create workflow definition", nil)
		return
	}

	h.logger.Info("Workflow definition created",
		zap.String("name", def.Name),
		zap.String("category", def.Category),
		zap.Int("parameters", len(def.ParamsData)),
	)

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreateWorkflowDefinitionResponse{
		Name:        def.Name,
		Description: def.Description,
		Category:    def.Category,
		Parameters:  def.ParamsData,
		CreatedAt:   def.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   def.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// ListWorkflowDefinitionsResponse represents list response
type ListWorkflowDefinitionsResponse struct {
	Workflows []WorkflowDefinitionSummary `json:"workflows"`
	Total     int                         `json:"total"`
	Page      int                         `json:"page"`
	Limit     int                         `json:"limit"`
}

// WorkflowDefinitionSummary represents a workflow definition summary
type WorkflowDefinitionSummary struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ListWorkflowDefinitions handles GET /v1/workflows
func (h *DefinitionHandlers) ListWorkflowDefinitions(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()
	category := query.Get("category")
	namePrefix := query.Get("name_prefix")

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

	// Query definitions
	defs, total, err := h.defStore.List(r.Context(), workflow.DefinitionFilter{
		Category:   category,
		NamePrefix: namePrefix,
		Page:       page,
		Limit:      limit,
	})
	if err != nil {
		h.logger.Error("Failed to list workflow definitions", zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to list workflow definitions", nil)
		return
	}

	// Build response
	workflows := make([]WorkflowDefinitionSummary, len(defs))
	for i, def := range defs {
		workflows[i] = WorkflowDefinitionSummary{
			Name:        def.Name,
			Description: def.Description,
			Category:    def.Category,
			CreatedAt:   def.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   def.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ListWorkflowDefinitionsResponse{
		Workflows: workflows,
		Total:     total,
		Page:      page,
		Limit:     limit,
	})
}

// GetWorkflowDefinitionResponse represents get definition response
type GetWorkflowDefinitionResponse struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Category    string                 `json:"category,omitempty"`
	Content     string                 `json:"content"`
	Parameters  []workflow.Parameter   `json:"parameters,omitempty"`
	Tags        map[string]interface{} `json:"tags,omitempty"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}

// GetWorkflowDefinition handles GET /v1/workflows/{name}
func (h *DefinitionHandlers) GetWorkflowDefinition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	h.logger.Debug("Get workflow definition request", zap.String("name", name))

	if name == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Workflow name is required", nil)
		return
	}

	// Get definition
	def, err := h.defStore.Get(r.Context(), name)
	if err != nil {
		if contains(err.Error(), "not found") {
			writeError(w, r, http.StatusNotFound, "not_found", err.Error(), nil)
			return
		}
		h.logger.Error("Failed to get workflow definition", zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to get workflow definition", nil)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(GetWorkflowDefinitionResponse{
		Name:        def.Name,
		Description: def.Description,
		Category:    def.Category,
		Content:     def.Content,
		Parameters:  def.ParamsData,
		Tags:        def.TagsData,
		CreatedAt:   def.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   def.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// UpdateWorkflowDefinitionRequest represents update definition request
type UpdateWorkflowDefinitionRequest struct {
	Description string                 `json:"description,omitempty"`
	Category    string                 `json:"category,omitempty"`
	Tags        map[string]interface{} `json:"tags,omitempty"`
	Content     string                 `json:"content,omitempty"`
}

// UpdateWorkflowDefinition handles PUT /v1/workflows/{name}
func (h *DefinitionHandlers) UpdateWorkflowDefinition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	if name == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Workflow name is required", nil)
		return
	}

	// Parse request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Failed to read request body", nil)
		return
	}
	defer r.Body.Close()

	var req UpdateWorkflowDefinitionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON format", nil)
		return
	}

	// Get existing definition
	def, err := h.defStore.Get(r.Context(), name)
	if err != nil {
		if contains(err.Error(), "not found") {
			writeError(w, r, http.StatusNotFound, "not_found", err.Error(), nil)
			return
		}
		h.logger.Error("Failed to get workflow definition", zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to get workflow definition", nil)
		return
	}

	// Update fields
	if req.Description != "" {
		def.Description = req.Description
	}
	if req.Category != "" {
		def.Category = req.Category
	}
	if req.Tags != nil {
		def.TagsData = req.Tags
	}
	if req.Content != "" {
		// Validate new content
		wf, err := h.parser.Parse([]byte(req.Content))
		if err != nil {
			writeError(w, r, http.StatusUnprocessableEntity, "validation_error", "YAML validation failed", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}

		if h.validator != nil {
			if _, err := h.validator.ValidateYAML([]byte(req.Content)); err != nil {
				writeError(w, r, http.StatusUnprocessableEntity, "validation_error", "YAML validation failed", map[string]interface{}{
					"error": err.Error(),
				})
				return
			}
		}

		// Re-extract parameters
		def.ParamsData = h.paramExtractor.ExtractParameters(wf)
		def.Content = req.Content
	}

	// Update in store
	err = h.defStore.Update(r.Context(), def)
	if err != nil {
		if contains(err.Error(), "not found") {
			writeError(w, r, http.StatusNotFound, "not_found", err.Error(), nil)
			return
		}
		h.logger.Error("Failed to update workflow definition", zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to update workflow definition", nil)
		return
	}

	h.logger.Info("Workflow definition updated",
		zap.String("name", def.Name),
		zap.String("category", def.Category),
		zap.Int("parameters", len(def.ParamsData)),
	)

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(GetWorkflowDefinitionResponse{
		Name:        def.Name,
		Description: def.Description,
		Category:    def.Category,
		Content:     def.Content,
		Parameters:  def.ParamsData,
		Tags:        def.TagsData,
		CreatedAt:   def.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   def.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// DeleteWorkflowDefinition handles DELETE /v1/workflows/{name}
func (h *DefinitionHandlers) DeleteWorkflowDefinition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	h.logger.Info("Delete workflow definition request", zap.String("name", name))

	if name == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Workflow name is required", nil)
		return
	}

	// Check if definition is referenced by schedules (AC10 - Story 1.10)
	if h.scheduleManager != nil {
		schedules, err := h.scheduleManager.ListByWorkflow(r.Context(), name)
		if err != nil {
			h.logger.Warn("Failed to check schedule references", zap.Error(err))
			// Continue with deletion even if check fails
		} else if len(schedules) > 0 {
			// Build schedule details for error response
			scheduleDetails := make([]map[string]interface{}, 0, len(schedules))
			for _, sched := range schedules {
				scheduleDetails = append(scheduleDetails, map[string]interface{}{
					"schedule_id": sched.ID,
					"cron":        sched.Cron,
					"status":      sched.Status,
				})
			}

			writeError(w, r, http.StatusConflict, "conflict",
				fmt.Sprintf("Cannot delete workflow '%s', referenced by %d active schedules", name, len(schedules)),
				map[string]interface{}{
					"schedules":  scheduleDetails,
					"suggestion": "Delete all schedules before deleting the workflow definition",
				})
			return
		}
	}

	// Delete definition
	err := h.defStore.Delete(r.Context(), name)
	if err != nil {
		if contains(err.Error(), "not found") {
			writeError(w, r, http.StatusNotFound, "not_found", err.Error(), nil)
			return
		}
		h.logger.Error("Failed to delete workflow definition", zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to delete workflow definition", nil)
		return
	}

	// Return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAny(s, substr))
}

func containsAny(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
