// Package handlers provides HTTP handler implementations.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Websoft9/waterflow/pkg/logger"
	"go.uber.org/zap"
)

// AdminHandler handles administrative operations.
type AdminHandler struct {
	logger *zap.Logger
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(log *zap.Logger) *AdminHandler {
	return &AdminHandler{
		logger: log,
	}
}

// SetLogLevelRequest represents the request body for setting log level.
type SetLogLevelRequest struct {
	Level string `json:"level"`
}

// LogLevelResponse represents the log level response.
type LogLevelResponse struct {
	Level   string `json:"level"`
	Message string `json:"message,omitempty"`
}

// SetLogLevel handles PUT /admin/log-level
func (h *AdminHandler) SetLogLevel(w http.ResponseWriter, r *http.Request) {
	var req SetLogLevelRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := logger.SetLevel(req.Level); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := "admin"
	h.logger.Info("Log level changed via API",
		zap.String("level", req.Level),
		zap.String("changed_by", user),
		zap.String("remote_addr", r.RemoteAddr),
		zap.String("user_agent", r.UserAgent()),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(LogLevelResponse{
		Level:   req.Level,
		Message: "Log level updated successfully",
	})
}

// GetLogLevel handles GET /admin/log-level
func (h *AdminHandler) GetLogLevel(w http.ResponseWriter, r *http.Request) {
	currentLevel := logger.GetLevel()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(LogLevelResponse{
		Level: currentLevel,
	})
}
