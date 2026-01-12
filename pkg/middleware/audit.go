package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/Websoft9/waterflow/pkg/audit"
	"go.uber.org/zap"
)

// auditResponseWriter wraps http.ResponseWriter to capture status code and size
type auditResponseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *auditResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *auditResponseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// AuditMiddleware creates an audit logging middleware for API requests.
//
// This middleware records all API requests (except health checks) to the audit log
// with details including HTTP method, path, status code, client IP, and duration.
func AuditMiddleware(auditLogger audit.AuditLogger, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip health checks and metrics endpoints
			path := r.URL.Path
			if path == "/health" || path == "/ready" || path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			// Record start time
			startTime := time.Now()

			// Wrap response writer to capture status and size
			rw := &auditResponseWriter{ResponseWriter: w, statusCode: 0, size: 0}

			// Process request
			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(startTime)

			// Get client IP
			clientIP := r.RemoteAddr
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				clientIP = strings.Split(forwarded, ",")[0]
			} else if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
				clientIP = realIP
			}

			// Build audit entry
			entry := audit.NewAuditLogEntry(audit.EventAPIRequest, categorizeAPIPath(path)).
				WithUser(&audit.UserContext{
					IP:        clientIP,
					UserAgent: r.UserAgent(),
				}).
				WithResource(&audit.ResourceContext{
					Type: "api",
					ID:   path,
					Name: r.Method + " " + path,
				}).
				WithAction("request").
				WithResult(resultFromStatus(rw.statusCode)).
				WithSeverity(severityFromStatus(rw.statusCode)).
				WithDetail("method", r.Method).
				WithDetail("path", path).
				WithDetail("status_code", rw.statusCode).
				WithDetail("duration_ms", duration.Milliseconds()).
				WithDetail("request_size", r.ContentLength).
				WithDetail("response_size", rw.size)

			// Add request ID if available
			if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
				entry.WithMetadata("request_id", requestID)
			}

			// Log audit entry asynchronously (non-blocking)
			go func() {
				if err := auditLogger.Log(r.Context(), entry); err != nil {
					// Use structured logging for errors, don't fail the request
					logger.Error("Failed to write audit log",
						zap.Error(err),
						zap.String("event_type", entry.EventType),
						zap.String("path", path),
					)
				}
			}()
		})
	}
}

// categorizeAPIPath determines the event category based on the API path.
func categorizeAPIPath(path string) audit.EventCategory {
	if strings.HasPrefix(path, "/v1/workflows") {
		return audit.CategoryWorkflow
	}
	if strings.HasPrefix(path, "/v1/audit") {
		return audit.CategoryAdmin
	}
	if strings.HasPrefix(path, "/v1/agents") {
		return audit.CategoryAgent
	}
	// Default category
	return audit.CategoryWorkflow
}

// severityFromStatus determines severity level from HTTP status code.
func severityFromStatus(code int) audit.Severity {
	if code >= 500 {
		return audit.SeverityError
	}
	if code >= 400 {
		return audit.SeverityWarn
	}
	return audit.SeverityInfo
}

// resultFromStatus determines result from HTTP status code.
func resultFromStatus(code int) audit.Result {
	if code >= 200 && code < 300 {
		return audit.ResultSuccess
	}
	if code == 404 {
		return audit.ResultNotFound
	}
	if code == 403 || code == 401 {
		return audit.ResultPermissionDenied
	}
	return audit.ResultError
}
