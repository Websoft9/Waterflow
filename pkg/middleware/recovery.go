package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"

	"go.uber.org/zap"
)

// Recovery middleware recovers from panics and logs the error
func Recovery(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log panic with stack trace
					stack := string(debug.Stack())
					logger.Error("HTTP panic recovered",
						zap.String("error", fmt.Sprintf("%v", err)),
						zap.String("stack", stack),
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
						zap.String("request_id", GetRequestID(r.Context())),
					)

					// Return RFC 7807 error response
					errorResponse := map[string]interface{}{
						"type":     "about:blank",
						"title":    "Internal Server Error",
						"status":   http.StatusInternalServerError,
						"detail":   "An unexpected error occurred",
						"instance": r.URL.Path,
					}

					w.Header().Set("Content-Type", "application/problem+json")
					w.WriteHeader(http.StatusInternalServerError)

					if encodeErr := json.NewEncoder(w).Encode(errorResponse); encodeErr != nil {
						logger.Error("Failed to encode error response", zap.Error(encodeErr))
					}
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
