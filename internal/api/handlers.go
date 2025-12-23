package api

import (
	"encoding/json"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Handlers contains HTTP handler functions
type Handlers struct {
	logger    *zap.Logger
	validator *dsl.Validator
	version   string
	commit    string
	buildTime string
}

// NewHandlers creates new Handlers instance
func NewHandlers(logger *zap.Logger, version, commit, buildTime string) *Handlers {
	validator, err := dsl.NewValidator(logger)
	if err != nil {
		logger.Fatal("Failed to create DSL validator", zap.Error(err))
	}

	return &Handlers{
		logger:    logger,
		validator: validator,
		version:   version,
		commit:    commit,
		buildTime: buildTime,
	}
}

// ErrorResponse represents RFC 7807 Problem Details format
type ErrorResponse struct {
	Type     string       `json:"type"`
	Title    string       `json:"title"`
	Status   int          `json:"status"`
	Detail   string       `json:"detail,omitempty"`
	Instance string       `json:"instance,omitempty"`
	Errors   []FieldError `json:"errors,omitempty"`
}

// FieldError represents a field-level error
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Health handles GET /health endpoint
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode health response", zap.Error(err))
	}
}

// Ready handles GET /ready endpoint
func (h *Handlers) Ready(w http.ResponseWriter, r *http.Request) {
	// Note: Actual Temporal health check is implemented via router dependency injection
	// This handler is kept for backward compatibility but should not be called directly
	// Use NewRouter with temporalClient to get proper readiness checks
	response := map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"checks":    map[string]string{"temporal": "not_configured"},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode ready response", zap.Error(err))
	}
}

// Version handles GET /version endpoint
func (h *Handlers) Version(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"version":    h.version,
		"commit":     h.commit,
		"build_time": h.buildTime,
		"go_version": runtime.Version(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode version response", zap.Error(err))
	}
}

// Metrics handles GET /metrics endpoint
func (h *Handlers) Metrics(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}

// NotFound handles 404 errors with RFC 7807 format
func (h *Handlers) NotFound(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, r, http.StatusNotFound, "Not Found", "The requested resource was not found")
}

// MethodNotAllowed handles 405 errors with RFC 7807 format
func (h *Handlers) MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "The request method is not allowed for this resource")
}

// writeError writes RFC 7807 error response
func (h *Handlers) writeError(w http.ResponseWriter, r *http.Request, status int, title, detail string) {
	errResp := ErrorResponse{
		Type:     "about:blank",
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: r.URL.Path,
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		h.logger.Error("Failed to encode error response", zap.Error(err))
	}
}

// ValidateWorkflow handles POST /v1/workflows/validate endpoint
func (h *Handlers) ValidateWorkflow(w http.ResponseWriter, r *http.Request) {
	// Limit request body size to 10MB (防止 DoS 攻击)
	const maxBodySize = 10 * 1024 * 1024 // 10MB
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	// Read YAML content
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, r, http.StatusBadRequest, "Invalid Request", "Request body too large or failed to read")
		return
	}
	defer func() { _ = r.Body.Close() }()

	if len(body) == 0 {
		h.writeError(w, r, http.StatusBadRequest, "Invalid Request", "Request body is empty")
		return
	}

	// Validate YAML
	workflow, err := h.validator.ValidateYAML(body)
	if err != nil {
		// Return validation error
		if validationErr, ok := err.(*dsl.ValidationError); ok {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(validationErr.ToHTTPError())
			return
		}

		// Other error
		h.writeError(w, r, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	// Validation success, return parsed workflow
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":    true,
		"workflow": workflow,
	})
}

// RenderWorkflow handles POST /v1/workflows/render endpoint
func (h *Handlers) RenderWorkflow(w http.ResponseWriter, r *http.Request) {
	// Limit request body size to 10MB
	const maxBodySize = 10 * 1024 * 1024 // 10MB
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	// Read YAML content
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, r, http.StatusBadRequest, "Invalid Request", "Request body too large or failed to read")
		return
	}
	defer func() { _ = r.Body.Close() }()

	if len(body) == 0 {
		h.writeError(w, r, http.StatusBadRequest, "Invalid Request", "Request body is empty")
		return
	}

	// Parse workflow
	parser := dsl.NewParser(h.logger)
	workflow, err := parser.Parse(body)
	if err != nil {
		h.writeError(w, r, http.StatusBadRequest, "Parse Error", "Failed to parse workflow YAML")
		return
	}

	// Render workflow (evaluate expressions)
	renderer := dsl.NewWorkflowRenderer()
	renderedWorkflow, err := renderer.RenderWorkflow(workflow)
	if err != nil {
		h.writeError(w, r, http.StatusBadRequest, "Render Error", err.Error())
		return
	}

	// Return rendered workflow
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"workflow": renderedWorkflow,
	})
}

// GetWorkflowSchema handles GET /schema/workflow.json endpoint
func (h *Handlers) GetWorkflowSchema(w http.ResponseWriter, r *http.Request) {
	// Get embedded schema from validator
	schemaJSON, err := h.validator.GetSchemaJSON()
	if err != nil {
		h.writeError(w, r, http.StatusInternalServerError, "Internal Server Error", "Failed to load schema")
		return
	}

	// Return schema JSON
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Allow CORS for schema access
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(schemaJSON)
}
