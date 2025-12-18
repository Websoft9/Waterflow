package api

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Handlers contains HTTP handler functions
type Handlers struct {
	logger *zap.Logger
}

// NewHandlers creates new Handlers instance
func NewHandlers(logger *zap.Logger) *Handlers {
	return &Handlers{
		logger: logger,
	}
}

// ErrorResponse represents RFC 7807 Problem Details format
type ErrorResponse struct {
	Type     string                 `json:"type"`
	Title    string                 `json:"title"`
	Status   int                    `json:"status"`
	Detail   string                 `json:"detail,omitempty"`
	Instance string                 `json:"instance,omitempty"`
	Extra    map[string]interface{} `json:"extra,omitempty"`
}

// Health handles GET /health endpoint
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "ok",
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
	// TODO: Check Temporal connection when integrated (Story 1-8)
	response := map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
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
		"version":    "dev",
		"commit":     "unknown",
		"build_time": "unknown",
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
