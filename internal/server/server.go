// Package server implements the HTTP server.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Websoft9/waterflow/pkg/config"
	"go.uber.org/zap"
)

// Server represents the HTTP server.
type Server struct {
	// httpServer is the underlying HTTP server instance.
	httpServer *http.Server
	// config holds server configuration.
	config *config.Config
	// logger is the structured logger instance.
	logger *zap.Logger
}

// New creates a new Server instance.
func New(cfg *config.Config, logger *zap.Logger) *Server {
	return &Server{
		config: cfg,
		logger: logger,
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/health", s.healthHandler)

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port),
		Handler:      mux,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
	}

	s.logger.Info("HTTP server starting",
		zap.String("host", s.config.Server.Host),
		zap.Int("port", s.config.Server.Port),
	)

	// Start server (blocking)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("HTTP server shutting down")

	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}

	return nil
}

// healthHandler handles the /health endpoint.
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	}); err != nil {
		s.logger.Error("failed to encode health response", zap.Error(err))
	}
}
