// Package server implements the HTTP server.
package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/Websoft9/waterflow/internal/api"
	"github.com/Websoft9/waterflow/pkg/audit"
	"github.com/Websoft9/waterflow/pkg/config"
	"github.com/Websoft9/waterflow/pkg/events"
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
	// agentMonitor periodically updates agent metrics
	agentMonitor *AgentMonitor
	// eventDispatcher dispatches workflow lifecycle events
	eventDispatcher *events.EventDispatcher
	// auditLogger records audit logs (Story 9-3)
	auditLogger audit.AuditLogger
}

// New creates a new Server instance.
func New(cfg *config.Config, logger *zap.Logger, version, commit, buildTime string) *Server {
	// Initialize server without Temporal first (async connection in Start method)
	// This allows HTTP server to start immediately for health checks

	// Initialize EventHandler based on configuration
	var eventHandler events.EventHandler
	switch cfg.Events.HandlerType {
	case "webhook":
		if cfg.Events.Webhook.URL != "" {
			eventHandler = events.NewWebhookEventHandler(events.WebhookConfig{
				URL:     cfg.Events.Webhook.URL,
				Headers: cfg.Events.Webhook.Headers,
				Timeout: cfg.Events.Webhook.Timeout,
			})
			logger.Info("Webhook event handler initialized",
				zap.String("url", cfg.Events.Webhook.URL),
			)
		} else {
			logger.Warn("Webhook handler configured but URL is empty, using noop")
			eventHandler = events.NewNoOpEventHandler()
		}
	case "noop", "":
		eventHandler = events.NewNoOpEventHandler()
		logger.Info("NoOp event handler initialized")
	default:
		logger.Warn("Unknown event handler type, using noop",
			zap.String("type", cfg.Events.HandlerType),
		)
		eventHandler = events.NewNoOpEventHandler()
	}

	// Create EventDispatcher
	eventDispatcher := events.NewEventDispatcher(events.EventDispatcherConfig{
		Handler: eventHandler,
		Logger:  logger,
		Timeout: 10 * time.Second,
	})

	return &Server{
		config:          cfg,
		logger:          logger,
		version:         version,
		commit:          commit,
		buildTime:       buildTime,
		temporalClient:  nil, // Will be initialized asynchronously in Start()
		eventDispatcher: eventDispatcher,
		auditLogger:     nil, // Will be initialized in Start()
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	// Initialize audit logging (Story 9-3)
	if s.config.Audit.Enabled {
		// Set default audit path if not configured
		auditPath := s.config.Audit.File.Path
		if auditPath == "" {
			auditPath = "/var/log/waterflow/audit"
		}

		auditConfig := &audit.FileAuditStoreConfig{
			Path:       auditPath,
			MaxSize:    s.config.Audit.File.MaxSize,
			MaxAge:     s.config.Audit.File.MaxAge,
			MaxBackups: s.config.Audit.File.MaxBackups,
			Compress:   s.config.Audit.File.Compress,
		}

		auditLogger, err := audit.NewFileAuditStore(auditConfig)
		if err != nil {
			s.logger.Warn("Failed to initialize audit logger",
				zap.Error(err),
				zap.String("path", auditPath),
			)
		} else {
			s.auditLogger = auditLogger
			s.logger.Info("Audit logging initialized",
				zap.String("path", auditPath),
				zap.Int("max_size_mb", auditConfig.MaxSize),
				zap.Int("max_age_days", auditConfig.MaxAge),
			)
		}
	} else {
		s.logger.Info("Audit logging disabled")
	}

	// Initialize Temporal client asynchronously to avoid blocking HTTP server startup
	// This allows health checks to pass while Temporal connection is being established
	if s.config.Temporal.Host != "" {
		go func() {
			s.logger.Info("Initializing Temporal client asynchronously",
				zap.String("temporal_host", s.config.Temporal.Host),
			)
			temporalClient, err := temporal.NewClient(&s.config.Temporal, s.logger)
			if err != nil {
				s.logger.Warn("Failed to connect to Temporal, workflow API will be disabled",
					zap.Error(err),
					zap.String("temporal_host", s.config.Temporal.Host),
				)
			} else {
				s.temporalClient = temporalClient
				s.logger.Info("Temporal client connected successfully",
					zap.String("temporal_host", s.config.Temporal.Host),
					zap.String("namespace", s.config.Temporal.Namespace),
				)

				// Start AgentMonitor after Temporal is available
				s.agentMonitor = NewAgentMonitor(s.temporalClient, s.logger, 30*time.Second)
				s.agentMonitor.Start()
			}
		}()
	} else {
		s.logger.Info("Temporal not configured, workflow API will be disabled")
	}

	// Start AgentMonitor if Temporal is available
	// Note: Removed from here as it's now in the async goroutine above

	// Create router with all API endpoints (Story 8-4 AC6: pass config for health check timeouts)
	router := api.NewRouterWithDB(s.logger, s.temporalClient, s.eventDispatcher, nil, s.config, s.version, s.commit, s.buildTime, s.auditLogger)

	// Apply middleware chain: RequestID -> Logger -> Recovery -> Metrics -> Audit -> CORS -> Version -> Router
	// Order follows AC7: RequestID first for tracing, Logger for request logging,
	// Recovery to catch panics, Metrics for monitoring, Audit for compliance, CORS for security, Version for info
	var handler http.Handler = router

	// Add audit middleware if audit logger is initialized (Story 9-3 AC3)
	if s.auditLogger != nil {
		handler = middleware.AuditMiddleware(s.auditLogger, s.logger)(handler)
	}

	handler = middleware.RequestID(
		middleware.Logger(s.logger)(
			middleware.Recovery(s.logger)(
				middleware.Metrics(
					middleware.CORS(
						middleware.Version(s.version)(handler),
					),
				),
			),
		),
	)

	// Determine if HTTPS is enabled (Story 9-1)
	if s.config.Server.IsTLSEnabled() {
		return s.startHTTPS(handler)
	}

	// Start HTTP server only
	return s.startHTTP(handler)
}

// startHTTP starts the HTTP server (without TLS).
func (s *Server) startHTTP(handler http.Handler) error {
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

// startHTTPS starts the HTTPS server with TLS (Story 9-1).
func (s *Server) startHTTPS(handler http.Handler) error {
	// Get TLS certificate and key files
	certFile := s.config.Server.GetTLSCertFile()
	keyFile := s.config.Server.GetTLSKeyFile()

	// Build TLS configuration
	tlsConfig := s.config.Server.HTTPS.BuildTLSConfig()

	// Create HTTPS server
	httpsPort := s.config.Server.HTTPS.Port
	if httpsPort == 0 {
		httpsPort = 8443 // Default HTTPS port
	}

	httpsServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Server.Host, httpsPort),
		Handler:      handler,
		TLSConfig:    tlsConfig,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
	}

	s.httpServer = httpsServer // Store for shutdown

	s.logger.Info("HTTPS server starting",
		zap.String("host", s.config.Server.Host),
		zap.Int("port", httpsPort),
		zap.String("cert_file", certFile),
		zap.Uint16("min_tls_version", tlsConfig.MinVersion),
	)

	// Check if HTTP redirect is enabled (Story 9-1 AC4)
	if s.config.Server.HTTP.Enabled {
		go s.startHTTPRedirect(httpsPort)
	}

	// Start HTTPS server (blocking)
	if err := httpsServer.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTPS server failed: %w", err)
	}

	return nil
}

// startHTTPRedirect starts an HTTP server that redirects to HTTPS (Story 9-1 AC4).
func (s *Server) startHTTPRedirect(httpsPort int) {
	httpPort := s.config.Server.HTTP.Port
	if httpPort == 0 {
		httpPort = 8080 // Default HTTP port
	}

	// Create redirect handler
	var redirectHandler http.Handler
	if s.config.Server.HTTP.RedirectToHTTPS {
		// HTTP → HTTPS redirect
		redirectHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract host without port (supports IPv4 and IPv6)
			host, _, err := net.SplitHostPort(r.Host)
			if err != nil {
				// No port in host, use as-is
				host = r.Host
			}

			// Construct HTTPS URL
			httpsURL := fmt.Sprintf("https://%s:%d%s", host, httpsPort, r.RequestURI)
			http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
		})

		s.logger.Info("HTTP → HTTPS redirect enabled",
			zap.Int("http_port", httpPort),
			zap.Int("https_port", httpsPort),
		)
	} else {
		// Normal HTTP server (no redirect)
		// Use the same handler as HTTPS
		redirectHandler = s.httpServer.Handler

		s.logger.Info("HTTP server starting (parallel with HTTPS)",
			zap.Int("http_port", httpPort),
		)
	}

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Server.Host, httpPort),
		Handler:      redirectHandler,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
	}

	// Start HTTP server (non-blocking)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP redirect server failed", zap.Error(err))
		}
	}()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("HTTP server shutting down")

	// Stop AgentMonitor
	if s.agentMonitor != nil {
		s.agentMonitor.Stop()
	}

	// Close Temporal client if connected
	if s.temporalClient != nil {
		s.temporalClient.Close()
	}

	// Close audit logger (Story 9-3)
	if s.auditLogger != nil {
		if err := s.auditLogger.Close(); err != nil {
			s.logger.Warn("Failed to close audit logger", zap.Error(err))
		}
	}

	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}

	return nil
}
