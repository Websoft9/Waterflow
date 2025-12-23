// Package metrics provides Prometheus metrics for monitoring
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPRequestsTotal is the total number of HTTP requests
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "waterflow_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration is the HTTP request duration in seconds
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "waterflow_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// WorkflowsTotal is the total number of workflows submitted
	WorkflowsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "waterflow_workflows_total",
			Help: "Total number of workflows submitted",
		},
		[]string{"status"}, // completed, failed, running
	)
)
