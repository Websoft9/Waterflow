package middleware

import (
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// extractClientIP extracts the real client IP address from request.
// Checks X-Forwarded-For and X-Real-IP headers for reverse proxy scenarios.
func extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fallback to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		return ip[:idx]
	}
	return ip
}

// responseWriter wraps http.ResponseWriter to capture status and size
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

// Write captures the response size
func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// Logger middleware logs HTTP requests and responses
func Logger(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer
			rw := &responseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			// Process request
			next.ServeHTTP(rw, r)

			// Log request
			duration := time.Since(start)

			// Extract client IP (supports reverse proxy)
			ip := extractClientIP(r)

			logger.Info("HTTP request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("query", r.URL.RawQuery),
				zap.Int("status", rw.status),
				zap.Int("size", rw.size),
				zap.Duration("duration", duration),
				zap.String("ip", ip),
				zap.String("user_agent", r.UserAgent()),
				zap.String("request_id", GetRequestID(r.Context())),
			)
		})
	}
}
