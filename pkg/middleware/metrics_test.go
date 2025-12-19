package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Websoft9/waterflow/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMetricsMiddleware(t *testing.T) {
	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Wrap with metrics middleware
	metricsHandler := Metrics(handler)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Reset metrics
	metrics.HTTPRequestsTotal.Reset()
	metrics.HTTPRequestDuration.Reset()

	// Execute request
	metricsHandler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify metrics with correct label order (method, path, status)
	counterValue := testutil.ToFloat64(metrics.HTTPRequestsTotal.WithLabelValues("GET", "/test", "200"))
	assert.Equal(t, float64(1), counterValue)

	// Verify histogram has recorded
	histogramCount := testutil.CollectAndCount(metrics.HTTPRequestDuration)
	assert.Greater(t, histogramCount, 0)
}

func TestMetricsMiddlewareWithDifferentStatus(t *testing.T) {
	// Create test handler that returns 404
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	metricsHandler := Metrics(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	w := httptest.NewRecorder()

	// Reset metrics
	metrics.HTTPRequestsTotal.Reset()

	metricsHandler.ServeHTTP(w, req)

	// Verify metrics with correct labels (method, path, status)
	counterValue := testutil.ToFloat64(metrics.HTTPRequestsTotal.WithLabelValues("POST", "/api/test", "404"))
	assert.Equal(t, float64(1), counterValue)
}

func TestMetricsMiddlewareIntegration(t *testing.T) {
	// Test that metrics are exported in Prometheus format
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	metricsHandler := Metrics(handler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	metrics.HTTPRequestsTotal.Reset()
	metrics.HTTPRequestDuration.Reset()

	metricsHandler.ServeHTTP(w, req)

	// Verify metric name and value
	expected := `
		# HELP waterflow_http_requests_total Total number of HTTP requests
		# TYPE waterflow_http_requests_total counter
		waterflow_http_requests_total{method="GET",path="/health",status="200"} 1
	`
	err := testutil.CollectAndCompare(metrics.HTTPRequestsTotal, strings.NewReader(expected))
	assert.NoError(t, err)
}
