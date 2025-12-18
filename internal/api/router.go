package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// NewRouter creates and configures HTTP router with all endpoints
func NewRouter(logger *zap.Logger) http.Handler {
	router := mux.NewRouter()

	// Register handlers
	h := NewHandlers(logger)

	router.HandleFunc("/health", h.Health).Methods(http.MethodGet)
	router.HandleFunc("/ready", h.Ready).Methods(http.MethodGet)
	router.HandleFunc("/version", h.Version).Methods(http.MethodGet)
	router.HandleFunc("/metrics", h.Metrics).Methods(http.MethodGet)

	// Custom error handlers
	router.NotFoundHandler = http.HandlerFunc(h.NotFound)
	router.MethodNotAllowedHandler = http.HandlerFunc(h.MethodNotAllowed)

	return router
}
