package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Websoft9/waterflow/internal/api/handlers"
	"github.com/Websoft9/waterflow/internal/server/schedule"
	"github.com/Websoft9/waterflow/pkg/audit"
	"github.com/Websoft9/waterflow/pkg/config"
	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/Websoft9/waterflow/pkg/events"
	"github.com/Websoft9/waterflow/pkg/metrics"
	"github.com/Websoft9/waterflow/pkg/middleware"
	"github.com/Websoft9/waterflow/pkg/temporal"
	"github.com/Websoft9/waterflow/pkg/trigger"
	"github.com/Websoft9/waterflow/pkg/workflow"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NewRouter creates and configures HTTP router with all endpoints
// db parameter is optional - if provided, database health check will be included in /ready endpoint
// cfg parameter is optional - if provided, uses configured health check timeouts; otherwise uses defaults
func NewRouter(logger *zap.Logger, temporalClient *temporal.Client, eventDispatcher *events.EventDispatcher, version, commit, buildTime string) http.Handler {
	return NewRouterWithDB(logger, temporalClient, eventDispatcher, nil, nil, version, commit, buildTime, nil, nil)
}

// NewRouterWithDB creates router with optional database health check support and configurable timeouts
// gormDB parameter is optional - if provided, workflow definition management endpoints will be registered
func NewRouterWithDB(logger *zap.Logger, temporalClient *temporal.Client, eventDispatcher *events.EventDispatcher, db *sql.DB, cfg *config.Config, version, commit, buildTime string, auditLogger audit.AuditLogger, nodeRegistry *node.Registry) http.Handler {
	return NewRouterWithGORM(logger, temporalClient, eventDispatcher, db, nil, cfg, version, commit, buildTime, auditLogger, nodeRegistry)
}

// NewRouterWithGORM creates router with GORM database for workflow definitions
func NewRouterWithGORM(logger *zap.Logger, temporalClient *temporal.Client, eventDispatcher *events.EventDispatcher, db *sql.DB, gormDB *gorm.DB, cfg *config.Config, version, commit, buildTime string, auditLogger audit.AuditLogger, nodeRegistry *node.Registry) http.Handler {
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

	// API Documentation endpoints (Story 10-2)
	router.HandleFunc("/docs", ServeSwaggerUI).Methods(http.MethodGet)
	router.HandleFunc("/api/openapi.yaml", ServeOpenAPISpec).Methods(http.MethodGet)

	// V1 API endpoints (utility endpoints)
	router.HandleFunc("/v1/workflows/validate", h.ValidateWorkflow).Methods(http.MethodPost)
	router.HandleFunc("/v1/workflows/render", h.RenderWorkflow).Methods(http.MethodPost)

	// Schema endpoint (Story 1.3 - AC6)
	router.HandleFunc("/schema/workflow.json", h.GetWorkflowSchema).Methods(http.MethodGet)

	// Workflow management endpoints
	if temporalClient != nil {
		// New Architecture: Definition API + Execution API (Story 1-9 ADR-0009)
		// IMPORTANT: Register these specific routes BEFORE legacy wildcard routes
		// to prevent /v1/workflows/{id} from intercepting /v1/workflows/definitions
		if gormDB != nil {
			// Initialize Definition Store
			defStore := workflow.NewDatabaseDefinitionStore(gormDB)

			// Schedule Manager (for AC10 conflict detection)
			scheduleManager := schedule.NewManager(temporalClient.GetClient(), defStore, logger)

			// Definition API: Workflow definition management
			defHandlers := NewDefinitionHandlers(logger, defStore, scheduleManager)
			router.HandleFunc("/v1/workflows/definitions", defHandlers.CreateWorkflowDefinition).Methods(http.MethodPost)
			router.HandleFunc("/v1/workflows/definitions", defHandlers.ListWorkflowDefinitions).Methods(http.MethodGet)
			router.HandleFunc("/v1/workflows/definitions/{name}", defHandlers.GetWorkflowDefinition).Methods(http.MethodGet)
			router.HandleFunc("/v1/workflows/definitions/{name}", defHandlers.UpdateWorkflowDefinition).Methods(http.MethodPut)
			router.HandleFunc("/v1/workflows/definitions/{name}", defHandlers.DeleteWorkflowDefinition).Methods(http.MethodDelete)

			// Execution API: Workflow runtime operations
			execHandlers := NewExecutionHandlers(logger, temporalClient, defStore, eventDispatcher)
			if auditLogger != nil {
				execHandlers.SetAuditLogger(auditLogger)
			}

			// Execute from definition
			router.HandleFunc("/v1/workflows/{name}/run", execHandlers.ExecuteWorkflowDefinition).Methods(http.MethodPost)

			// Direct execution
			router.HandleFunc("/v1/executions", execHandlers.DirectExecuteWorkflow).Methods(http.MethodPost)
			router.HandleFunc("/v1/executions", execHandlers.ListExecutions).Methods(http.MethodGet)
			router.HandleFunc("/v1/executions/{id}", execHandlers.GetExecutionStatus).Methods(http.MethodGet)
			router.HandleFunc("/v1/executions/{id}/logs", execHandlers.GetExecutionLogs).Methods(http.MethodGet)
			router.HandleFunc("/v1/executions/{id}/cancel", execHandlers.CancelExecution).Methods(http.MethodPost)
			router.HandleFunc("/v1/executions/{id}/terminate", execHandlers.TerminateExecution).Methods(http.MethodPost)

			// Schedule API: Workflow schedule management (Story 1-10)
			scheduleHandlers := NewScheduleHandlers(logger, scheduleManager)

			// Create schedule for workflow
			router.HandleFunc("/v1/workflows/{name}/schedules", scheduleHandlers.CreateSchedule).Methods(http.MethodPost)

			// List schedules for workflow
			router.HandleFunc("/v1/workflows/{name}/schedules", scheduleHandlers.ListSchedulesByWorkflow).Methods(http.MethodGet)

			// Get schedule details
			router.HandleFunc("/v1/workflows/{name}/schedules/{schedule_id}", scheduleHandlers.GetSchedule).Methods(http.MethodGet)

			// Update schedule
			router.HandleFunc("/v1/workflows/{name}/schedules/{schedule_id}", scheduleHandlers.UpdateSchedule).Methods(http.MethodPut)

			// Delete schedule
			router.HandleFunc("/v1/workflows/{name}/schedules/{schedule_id}", scheduleHandlers.DeleteSchedule).Methods(http.MethodDelete)

			// Pause schedule
			router.HandleFunc("/v1/workflows/{name}/schedules/{schedule_id}/pause", scheduleHandlers.PauseSchedule).Methods(http.MethodPost)

			// Resume schedule
			router.HandleFunc("/v1/workflows/{name}/schedules/{schedule_id}/resume", scheduleHandlers.ResumeSchedule).Methods(http.MethodPost)

			// Trigger schedule manually
			router.HandleFunc("/v1/workflows/{name}/schedules/{schedule_id}/trigger", scheduleHandlers.TriggerSchedule).Methods(http.MethodPost)

			// List all schedules (global query)
			router.HandleFunc("/v1/schedules", scheduleHandlers.ListAllSchedules).Methods(http.MethodGet)

			// Webhook Trigger API: Webhook trigger management (Story 1-11)
			triggerStorage, err := trigger.NewGORMStorage(gormDB)
			if err != nil {
				logger.Warn("Failed to initialize trigger storage",
					zap.Error(err),
				)
			} else {
				// Initialize Trigger Manager
				baseURL := fmt.Sprintf("http://%s:%d", cfg.Server.Host, cfg.Server.Port)
				if cfg.Server.Host == "" || cfg.Server.Host == "0.0.0.0" {
					baseURL = fmt.Sprintf("http://localhost:%d", cfg.Server.Port)
				}
				triggerManager := trigger.NewManager(
					temporalClient.GetClient(),
					defStore,
					triggerStorage,
					logger,
					baseURL,
				)

				// Initialize Trigger Handlers
				triggerHandlers := NewTriggerHandlers(logger, triggerManager)
				webhookHandlers := NewWebhookHandlers(logger, triggerManager)

				// Trigger Management API
				router.HandleFunc("/v1/workflows/{name}/triggers", triggerHandlers.CreateTrigger).Methods(http.MethodPost)
				router.HandleFunc("/v1/workflows/{name}/triggers", triggerHandlers.ListTriggers).Methods(http.MethodGet)
				router.HandleFunc("/v1/workflows/{name}/triggers/{trigger_id}", triggerHandlers.GetTrigger).Methods(http.MethodGet)
				router.HandleFunc("/v1/workflows/{name}/triggers/{trigger_id}", triggerHandlers.UpdateTrigger).Methods(http.MethodPatch)
				router.HandleFunc("/v1/workflows/{name}/triggers/{trigger_id}", triggerHandlers.DeleteTrigger).Methods(http.MethodDelete)
				router.HandleFunc("/v1/workflows/{name}/triggers/{trigger_id}/enable", triggerHandlers.EnableTrigger).Methods(http.MethodPost)
				router.HandleFunc("/v1/workflows/{name}/triggers/{trigger_id}/disable", triggerHandlers.DisableTrigger).Methods(http.MethodPost)

				// Webhook logs
				router.HandleFunc("/v1/workflows/{name}/triggers/{trigger_id}/logs", triggerHandlers.GetTriggerLogs).Methods(http.MethodGet)

				// Webhook Trigger Endpoint (public, for external webhooks)
				router.HandleFunc("/api/v1/webhooks/{trigger_id}/trigger", webhookHandlers.HandleWebhook).Methods(http.MethodPost)

				logger.Info("Webhook Trigger API initialized",
					zap.String("base_url", baseURL),
				)
			}
		}

		// Legacy Workflow Execution Endpoints (Story 1.9 - AC1-AC6)
		// Registered AFTER specific routes to prevent wildcard path conflicts
		wh := NewWorkflowHandlers(logger, temporalClient, eventDispatcher)

		// Set audit logger if provided (Story 9-3)
		if auditLogger != nil {
			wh.SetAuditLogger(auditLogger)
		}

		// Legacy endpoints - maintained for backward compatibility
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

		// AC8: Terminate workflow
		router.HandleFunc("/v1/workflows/{id}/terminate", wh.TerminateWorkflow).Methods(http.MethodPost)

		// AC6: Rerun workflow
		router.HandleFunc("/v1/workflows/{id}/rerun", wh.RerunWorkflow).Methods(http.MethodPost)

		// Agent discovery and health monitoring endpoints (Story 1.9 AC9, Story 2.7)
		ah := NewAgentHandlers(logger, temporalClient)
		router.HandleFunc("/v1/agents", ah.ListAgents).Methods(http.MethodGet)
		// /summary must be registered before /{name} to avoid being swallowed by the param route
		router.HandleFunc("/v1/agents/summary", ah.GetAgentsSummary).Methods(http.MethodGet)
		router.HandleFunc("/v1/agents/{name}", ah.GetAgentStatus).Methods(http.MethodGet)

		// Task Queue discovery endpoint (Story 2.7 AC3: real Temporal-backed implementation)
		router.HandleFunc("/v1/task-queues", ah.ListTaskQueues).Methods(http.MethodGet)
	}

	// Audit log endpoints (Story 9-3 AC6)
	if auditLogger != nil {
		auditHandler := NewAuditHandler(logger, auditLogger)
		auditHandler.RegisterRoutes(router)
	}

	// Node management endpoints (Story 5.6, Tech Debt: Node Handler Registry)
	nh := NewNodeHandlers(logger, nodeRegistry)
	router.HandleFunc("/v1/nodes", nh.ListNodes).Methods(http.MethodGet)
	router.HandleFunc("/v1/nodes/{name}", nh.GetNode).Methods(http.MethodGet)

	// Template management endpoints (Story 6.4)
	th := NewTemplateHandlers(logger)
	router.HandleFunc("/v1/templates", th.ListTemplates).Methods(http.MethodGet)
	router.HandleFunc("/v1/templates/{name}", th.GetTemplate).Methods(http.MethodGet)

	// Admin endpoints (Story 7.2 - AC3)
	// Note: Authentication middleware removed as Waterflow serves as a component,
	// authentication is handled by the parent application
	adminRouter := router.PathPrefix("/admin").Subrouter()
	adminH := handlers.NewAdminHandler(logger)
	adminRouter.HandleFunc("/log-level", adminH.GetLogLevel).Methods(http.MethodGet)
	adminRouter.HandleFunc("/log-level", adminH.SetLogLevel).Methods(http.MethodPut)

	// Custom error handlers
	router.NotFoundHandler = http.HandlerFunc(h.NotFound)
	router.MethodNotAllowedHandler = http.HandlerFunc(h.MethodNotAllowed)

	return router
}
