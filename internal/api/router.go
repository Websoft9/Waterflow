package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Websoft9/waterflow/internal/api/handlers"
	"github.com/Websoft9/waterflow/pkg/config"
	"github.com/Websoft9/waterflow/pkg/events"
	"github.com/Websoft9/waterflow/pkg/metrics"
	"github.com/Websoft9/waterflow/pkg/middleware"
	"github.com/Websoft9/waterflow/pkg/temporal"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// NewRouter creates and configures HTTP router with all endpoints
// db parameter is optional - if provided, database health check will be included in /ready endpoint
// cfg parameter is optional - if provided, uses configured health check timeouts; otherwise uses defaults
func NewRouter(logger *zap.Logger, temporalClient *temporal.Client, eventDispatcher *events.EventDispatcher, version, commit, buildTime string) http.Handler {
	return NewRouterWithDB(logger, temporalClient, eventDispatcher, nil, nil, version, commit, buildTime)
}

// NewRouterWithDB creates router with optional database health check support and configurable timeouts
func NewRouterWithDB(logger *zap.Logger, temporalClient *temporal.Client, eventDispatcher *events.EventDispatcher, db *sql.DB, cfg *config.Config, version, commit, buildTime string) http.Handler {
	router := mux.NewRouter()

	// Apply global middleware (AC7 - Request ID and Server Version headers)
	router.Use(middleware.RequestID)
	router.Use(middleware.Version(version))

	// Register basic handlers
	h := NewHandlers(logger, version, commit, buildTime)

	router.HandleFunc("/health", h.Health).Methods(http.MethodGet)

	// Get health check timeouts from config or use defaults (Story 8-4 AC6)
	temporalTimeout := 2 * time.Second
	dbTimeout := 1 * time.Second
	if cfg != nil {
		if cfg.Server.Health.TemporalTimeout > 0 {
			temporalTimeout = cfg.Server.Health.TemporalTimeout
		}
		if cfg.Server.Health.DatabaseTimeout > 0 {
			dbTimeout = cfg.Server.Health.DatabaseTimeout
		}
	}

	// Ready endpoint with Temporal and optional database health checks (Story 8-4 AC2)
	if temporalClient != nil || db != nil {
		router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
			// Story 8-4 Task 5: Record health check metrics
			start := time.Now()
			checks := make(map[string]string)
			allReady := true

			// Check Temporal connection (Story 8-4 AC2)
			if temporalClient != nil {
				ctx, cancel := context.WithTimeout(r.Context(), temporalTimeout)
				defer cancel()

				if err := temporalClient.CheckHealth(ctx); err != nil {
					checks["temporal"] = err.Error()
					allReady = false
					metrics.UpdateDependencyHealth("temporal", false)
				} else {
					checks["temporal"] = "ok"
					metrics.UpdateDependencyHealth("temporal", true)
				}
			}

			// Check database connection if configured (Story 8-4 AC2)
			if db != nil {
				ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
				defer cancel()

				if err := db.PingContext(ctx); err != nil {
					checks["database"] = err.Error()
					allReady = false
					metrics.UpdateDependencyHealth("database", false)
				} else {
					checks["database"] = "ok"
					metrics.UpdateDependencyHealth("database", true)
				}
			}

			response := map[string]interface{}{
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"checks":    checks,
			}

			w.Header().Set("Content-Type", "application/json")

			// Story 8-4 AC3: Return 503 when dependencies are unavailable
			var statusLabel string
			if allReady {
				response["status"] = "ready"
				w.WriteHeader(http.StatusOK)
				statusLabel = "ready"
				metrics.UpdateReadinessStatus(true)
			} else {
				response["status"] = "not_ready"
				w.WriteHeader(http.StatusServiceUnavailable)
				statusLabel = "not_ready"
				metrics.UpdateReadinessStatus(false)
			}

			// Record health check duration and count
			duration := time.Since(start).Seconds()
			metrics.RecordHealthCheck("ready", statusLabel, duration)

			if err := json.NewEncoder(w).Encode(response); err != nil {
				logger.Error("Failed to encode ready response", zap.Error(err))
			}
		}).Methods(http.MethodGet)
	} else {
		// Fallback when neither Temporal nor DB is configured
		router.HandleFunc("/ready", h.Ready).Methods(http.MethodGet)
	}
	router.HandleFunc("/version", h.Version).Methods(http.MethodGet)
	router.HandleFunc("/metrics", h.Metrics).Methods(http.MethodGet)

	// V1 API endpoints (utility endpoints)
	router.HandleFunc("/v1/workflows/validate", h.ValidateWorkflow).Methods(http.MethodPost)
	router.HandleFunc("/v1/workflows/render", h.RenderWorkflow).Methods(http.MethodPost)

	// Schema endpoint (Story 1.3 - AC6)
	router.HandleFunc("/schema/workflow.json", h.GetWorkflowSchema).Methods(http.MethodGet)

	// Workflow management endpoints (Story 1.9 - AC1-AC6)
	if temporalClient != nil {
		wh := NewWorkflowHandlers(logger, temporalClient, eventDispatcher)

		// AC1: Submit workflow
		router.HandleFunc("/v1/workflows", wh.SubmitWorkflow).Methods(http.MethodPost)

		// AC2: Get workflow status
		router.HandleFunc("/v1/workflows/{id}", wh.GetWorkflowStatus).Methods(http.MethodGet)

		// AC3: List workflows
		router.HandleFunc("/v1/workflows", wh.ListWorkflows).Methods(http.MethodGet)

		// AC4: Get workflow logs
		router.HandleFunc("/v1/workflows/{id}/logs", wh.GetWorkflowLogs).Methods(http.MethodGet)

		// AC5: Cancel workflow
		router.HandleFunc("/v1/workflows/{id}/cancel", wh.CancelWorkflow).Methods(http.MethodPost)

		// AC6: Rerun workflow
		router.HandleFunc("/v1/workflows/{id}/rerun", wh.RerunWorkflow).Methods(http.MethodPost)
	}

	// Node management endpoints (Story 5.6)
	nh := NewNodeHandlers(logger)
	router.HandleFunc("/v1/nodes", nh.ListNodes).Methods(http.MethodGet)
	router.HandleFunc("/v1/nodes/{name}", nh.GetNode).Methods(http.MethodGet)

	// Template management endpoints (Story 6.4)
	th := NewTemplateHandlers(logger)
	router.HandleFunc("/v1/templates", th.ListTemplates).Methods(http.MethodGet)
	router.HandleFunc("/v1/templates/{name}", th.GetTemplate).Methods(http.MethodGet)

	// Admin endpoints (Story 7.2 - AC3)
	adminRouter := router.PathPrefix("/admin").Subrouter()
	adminRouter.Use(middleware.RequireAuth)
	ah := handlers.NewAdminHandler(logger)
	adminRouter.HandleFunc("/log-level", ah.GetLogLevel).Methods(http.MethodGet)
	adminRouter.HandleFunc("/log-level", ah.SetLogLevel).Methods(http.MethodPut)

	// Custom error handlers
	router.NotFoundHandler = http.HandlerFunc(h.NotFound)
	router.MethodNotAllowedHandler = http.HandlerFunc(h.MethodNotAllowed)

	return router
}
