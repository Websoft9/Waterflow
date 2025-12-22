package api

import (
	"net/http"

	"github.com/Websoft9/waterflow/pkg/temporal"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// NewRouter creates and configures HTTP router with all endpoints
func NewRouter(logger *zap.Logger, temporalClient *temporal.Client, version, commit, buildTime string) http.Handler {
	router := mux.NewRouter()

	// Register basic handlers
	h := NewHandlers(logger, version, commit, buildTime)

	router.HandleFunc("/health", h.Health).Methods(http.MethodGet)
	router.HandleFunc("/ready", h.Ready).Methods(http.MethodGet)
	router.HandleFunc("/version", h.Version).Methods(http.MethodGet)
	router.HandleFunc("/metrics", h.Metrics).Methods(http.MethodGet)

	// V1 API endpoints (utility endpoints)
	router.HandleFunc("/v1/workflows/validate", h.ValidateWorkflow).Methods(http.MethodPost)
	router.HandleFunc("/v1/workflows/render", h.RenderWorkflow).Methods(http.MethodPost)

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
	}

	// Custom error handlers
	router.NotFoundHandler = http.HandlerFunc(h.NotFound)
	router.MethodNotAllowedHandler = http.HandlerFunc(h.MethodNotAllowed)

	return router
}
