package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/Websoft9/waterflow/pkg/audit"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// AuditHandler handles audit log query endpoints.
type AuditHandler struct {
	logger      *zap.Logger
	auditLogger audit.AuditLogger
}

// NewAuditHandler creates a new audit handler.
func NewAuditHandler(logger *zap.Logger, auditLogger audit.AuditLogger) *AuditHandler {
	return &AuditHandler{
		logger:      logger,
		auditLogger: auditLogger,
	}
}

// QueryLogsRequest represents the query parameters for audit logs.
type QueryLogsRequest struct {
	StartTime     *time.Time           `json:"start_time,omitempty"`
	EndTime       *time.Time           `json:"end_time,omitempty"`
	EventTypes    []string             `json:"event_types,omitempty"`
	EventCategory *audit.EventCategory `json:"event_category,omitempty"`
	UserID        string               `json:"user_id,omitempty"`
	ResourceType  string               `json:"resource_type,omitempty"`
	ResourceID    string               `json:"resource_id,omitempty"`
	Result        *audit.Result        `json:"result,omitempty"`
	Limit         int                  `json:"limit,omitempty"`
	Offset        int                  `json:"offset,omitempty"`
}

// QueryLogsResponse represents the response for audit log queries.
type QueryLogsResponse struct {
	Total   int                    `json:"total"`
	Limit   int                    `json:"limit"`
	Offset  int                    `json:"offset"`
	Entries []*audit.AuditLogEntry `json:"entries"`
}

// QueryLogs handles GET /v1/audit/logs endpoint.
//
// Supports filtering by:
// - start_time, end_time: Time range (RFC3339 format)
// - event_type: Event type filter (comma-separated)
// - event_category: Event category (workflow, secret, auth, etc.)
// - user_id: User ID filter
// - resource_type, resource_id: Resource filters
// - result: Result filter (success, failure, etc.)
// - limit, offset: Pagination
func (h *AuditHandler) QueryLogs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	filter := audit.AuditLogFilter{}

	// Parse time range
	if startTimeStr := r.URL.Query().Get("start_time"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filter.StartTime = &t
		}
	}

	if endTimeStr := r.URL.Query().Get("end_time"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filter.EndTime = &t
		}
	}

	// Parse event types (comma-separated)
	if eventTypes := r.URL.Query().Get("event_type"); eventTypes != "" {
		filter.EventTypes = []string{eventTypes}
	}

	// Parse event category
	if categoryStr := r.URL.Query().Get("event_category"); categoryStr != "" {
		category := audit.EventCategory(categoryStr)
		filter.EventCategory = &category
	}

	// Parse user ID
	filter.UserID = r.URL.Query().Get("user_id")

	// Parse resource filters
	filter.ResourceType = r.URL.Query().Get("resource_type")
	filter.ResourceID = r.URL.Query().Get("resource_id")

	// Parse result
	if resultStr := r.URL.Query().Get("result"); resultStr != "" {
		result := audit.Result(resultStr)
		filter.Result = &result
	}

	// Parse pagination
	filter.Limit = parseIntParam(r, "limit", 100)
	filter.Offset = parseIntParam(r, "offset", 0)

	// Query audit logs
	entries, err := h.auditLogger.Query(r.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to query audit logs", zap.Error(err))
		h.writeError(w, r, http.StatusInternalServerError, "internal_error", "Failed to query audit logs", nil)
		return
	}

	// Build response
	response := QueryLogsResponse{
		Total:   len(entries),
		Limit:   filter.Limit,
		Offset:  filter.Offset,
		Entries: entries,
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}

// RegisterRoutes registers audit API routes.
func (h *AuditHandler) RegisterRoutes(router *mux.Router) {
	// Audit logs query
	router.HandleFunc("/v1/audit/logs", h.QueryLogs).Methods("GET")
}

// parseIntParam parses an integer query parameter with a default value.
func parseIntParam(r *http.Request, name string, defaultValue int) int {
	if valueStr := r.URL.Query().Get(name); valueStr != "" {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}

// writeError writes an error response.
func (h *AuditHandler) writeError(w http.ResponseWriter, r *http.Request, statusCode int, errorCode string, message string, details map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    errorCode,
			"message": message,
		},
	}

	if details != nil {
		response["error"].(map[string]interface{})["details"] = details
	}

	_ = json.NewEncoder(w).Encode(response)
}
