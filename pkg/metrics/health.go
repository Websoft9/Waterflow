package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Health check related metrics (Story 8-4 Task 5)
var (
	// HealthCheckDuration tracks the duration of health check requests
	HealthCheckDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "waterflow_health_check_duration_seconds",
			Help:    "Duration of health check requests in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		},
		[]string{"endpoint", "status"},
	)

	// DependencyHealthStatus indicates the health status of dependencies
	// 1 = healthy, 0 = unhealthy
	DependencyHealthStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "waterflow_dependency_health_status",
			Help: "Health status of dependencies (1=healthy, 0=unhealthy)",
		},
		[]string{"dependency"},
	)

	// HealthCheckTotal counts the total number of health checks
	HealthCheckTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "waterflow_health_check_total",
			Help: "Total number of health check requests",
		},
		[]string{"endpoint", "status"},
	)

	// ReadinessStatus indicates if the service is ready (1) or not (0)
	ReadinessStatus = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "waterflow_readiness_status",
			Help: "Current readiness status (1=ready, 0=not ready)",
		},
	)
)

// RecordHealthCheck records a health check operation
func RecordHealthCheck(endpoint string, status string, duration float64) {
	HealthCheckDuration.WithLabelValues(endpoint, status).Observe(duration)
	HealthCheckTotal.WithLabelValues(endpoint, status).Inc()
}

// UpdateDependencyHealth updates the health status of a dependency
func UpdateDependencyHealth(dependency string, healthy bool) {
	if healthy {
		DependencyHealthStatus.WithLabelValues(dependency).Set(1)
	} else {
		DependencyHealthStatus.WithLabelValues(dependency).Set(0)
	}
}

// UpdateReadinessStatus updates the overall readiness status
func UpdateReadinessStatus(ready bool) {
	if ready {
		ReadinessStatus.Set(1)
	} else {
		ReadinessStatus.Set(0)
	}
}
