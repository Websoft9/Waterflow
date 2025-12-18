package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// ContextKey is the type for context keys
type ContextKey string

// RequestIDKey is the context key for request ID
const RequestIDKey ContextKey = "request_id"

// RequestID middleware generates or extracts request ID
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to get existing request ID from header
		requestID := r.Header.Get("X-Request-ID")

		// Generate new ID if not provided
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add to context
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

		// Set response header
		w.Header().Set("X-Request-ID", requestID)

		// Call next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}
