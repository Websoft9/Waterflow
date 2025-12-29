// Package server implements the HTTP server.
package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Websoft9/waterflow/internal/api"
	"github.com/Websoft9/waterflow/pkg/config"
	"github.com/Websoft9/waterflow/pkg/middleware"
	"github.com/Websoft9/waterflow/pkg/temporal"
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
	// version is the server version.
	version string
	// commit is the git commit hash.
	commit string
	// buildTime is the build timestamp.
	buildTime string
	// temporalClient is the Temporal workflow engine client
	temporalClient *temporal.Client
}

// New creates a new Server instance.
func New(cfg *config.Config, logger *zap.Logger, version, commit, buildTime string) *Server {
	// Initialize Temporal client
	var temporalClient *temporal.Client
	if cfg.Temporal.Host != "" {
		var err error
		temporalClient, err = temporal.NewClient(&cfg.Temporal, logger)
		if err != nil {
			logger.Warn("Failed to connect to Temporal, workflow API will be disabled",
				zap.Error(err),
				zap.String("temporal_host", cfg.Temporal.Host),
			)
		} else {
			logger.Info("Connected to Temporal successfully",
				zap.String("temporal_host", cfg.Temporal.Host),
				zap.String("namespace", cfg.Temporal.Namespace),
			)
		}
	} else {
		logger.Info("Temporal not configured, workflow API will be disabled")
	}

	return &Server{
		config:         cfg,
		logger:         logger,
		version:        version,
		commit:         commit,
		buildTime:      buildTime,
		temporalClient: temporalClient,
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	// Create router with all API endpoints
	router := api.NewRouter(s.logger, s.temporalClient, s.version, s.commit, s.buildTime)

	// Apply middleware chain: RequestID -> Logger -> Recovery -> Metrics -> CORS -> Version -> Router
	// Order follows AC7: RequestID first for tracing, Logger for request logging,
	// Recovery to catch panics, Metrics for monitoring, CORS for security, Version for info
	handler := middleware.RequestID(
		middleware.Logger(s.logger)(
			middleware.Recovery(s.logger)(
				middleware.Metrics(
					middleware.CORS(
						middleware.Version(s.version)(router),
					),
				),
			),
		),
	)

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port),
		Handler:      handler,
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

	// Close Temporal client if connected
	if s.temporalClient != nil {
		s.temporalClient.Close()
	}

	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}

	return nil
}
