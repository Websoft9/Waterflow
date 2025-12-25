package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Websoft9/waterflow/pkg/middleware"
	"github.com/Websoft9/waterflow/pkg/temporal"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// NewRouter creates and configures HTTP router with all endpoints
func NewRouter(logger *zap.Logger, temporalClient *temporal.Client, version, commit, buildTime string) http.Handler {
	router := mux.NewRouter()

	// Apply global middleware (AC7 - Request ID and Server Version headers)
	router.Use(middleware.RequestID)
	router.Use(middleware.Version(version))

	// Register basic handlers
	h := NewHandlers(logger, version, commit, buildTime)

	router.HandleFunc("/health", h.Health).Methods(http.MethodGet)

	// Ready endpoint with Temporal health check
	if temporalClient != nil {
		router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
			checks := make(map[string]string)
			allReady := true

			// Check Temporal connection
			if err := temporalClient.CheckHealth(r.Context()); err != nil {
				checks["temporal"] = err.Error()
				allReady = false
			} else {
				checks["temporal"] = "ok"
			}

			response := map[string]interface{}{
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"checks":    checks,
			}

			w.Header().Set("Content-Type", "application/json")

			if allReady {
				response["status"] = "ready"
				w.WriteHeader(http.StatusOK)
			} else {
				response["status"] = "not_ready"
				w.WriteHeader(http.StatusServiceUnavailable)
			}

			if err := json.NewEncoder(w).Encode(response); err != nil {
				logger.Error("Failed to encode ready response", zap.Error(err))
			}
		}).Methods(http.MethodGet)
	} else {
		// Fallback when Temporal is not configured
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
		wh := NewWorkflowHandlers(logger, temporalClient)

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

		// Story 2.2 - Task queue management (placeholder for Story 2.7)
		router.HandleFunc("/v1/task-queues", wh.ListTaskQueues).Methods(http.MethodGet)
	}

	// Custom error handlers
	router.NotFoundHandler = http.HandlerFunc(h.NotFound)
	router.MethodNotAllowedHandler = http.HandlerFunc(h.MethodNotAllowed)

	return router
}
