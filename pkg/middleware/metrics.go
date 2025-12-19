package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Websoft9/waterflow/pkg/metrics"
)

// Metrics middleware collects HTTP request metrics
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		rw := &responseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		// Process request
		next.ServeHTTP(rw, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		path := r.URL.Path
		method := r.Method
		status := fmt.Sprintf("%d", rw.status)

		metrics.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
		metrics.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	})
}
