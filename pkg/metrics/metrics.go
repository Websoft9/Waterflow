// Package metrics provides Prometheus metrics for monitoring
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// === HTTP Metrics ===

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

	// === Workflow Metrics ===

	// WorkflowsTotal is the total number of workflows by status
	WorkflowsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "waterflow_workflows_total",
			Help: "Total number of workflows by status",
		},
		[]string{"status"}, // submitted, completed, failed, cancelled
	)

	// WorkflowsRunning tracks currently running workflows
	WorkflowsRunning = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "waterflow_workflows_running",
			Help: "Number of currently running workflows",
		},
	)

	// WorkflowDuration tracks workflow execution duration distribution
	WorkflowDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "waterflow_workflow_duration_seconds",
			Help: "Workflow execution duration in seconds",
			// Buckets optimized for workflow duration: 1s to 1 hour
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600},
		},
		[]string{"status"}, // completed, failed
	)

	// === Agent Metrics ===

	// AgentsConnected tracks number of connected agents
	AgentsConnected = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "waterflow_agents_connected",
			Help: "Number of currently connected agents",
		},
	)

	// AgentsHealthy tracks number of healthy agents
	AgentsHealthy = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "waterflow_agents_healthy",
			Help: "Number of healthy agents",
		},
	)

	// AgentTasksTotal tracks total tasks executed by agents
	AgentTasksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "waterflow_agent_tasks_total",
			Help: "Total number of tasks executed by agents",
		},
		[]string{"task_queue", "status"}, // task_queue: linux-amd64, windows-amd64; status: completed, failed
	)

	// === Node Execution Metrics ===

	// NodeExecutionsTotal tracks total node executions by type
	NodeExecutionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "waterflow_node_executions_total",
			Help: "Total number of node executions by type and status",
		},
		[]string{"node_type", "status"}, // exec/shell, http/request, etc. | success, failure
	)

	// NodeExecutionDuration tracks node execution duration
	NodeExecutionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "waterflow_node_execution_duration_seconds",
			Help: "Node execution duration in seconds",
			// Buckets optimized for node execution: 10ms to 10 minutes
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 5, 10, 30, 60, 300, 600},
		},
		[]string{"node_type"},
	)

	// === API Business Metrics ===

	// WorkflowSubmissionsTotal tracks workflow submissions
	WorkflowSubmissionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "waterflow_workflow_submissions_total",
			Help: "Total number of workflow submissions",
		},
		[]string{"result"}, // success, validation_error, server_error
	)

	// YAMLValidationsTotal tracks YAML validation results
	YAMLValidationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "waterflow_yaml_validations_total",
			Help: "Total number of YAML validations",
		},
		[]string{"result"}, // success, failure
	)

	// === System Metrics (Go runtime) ===

	// These are auto-registered by prometheus/client_golang
	// - go_goroutines
	// - go_memstats_alloc_bytes
	// - go_memstats_heap_alloc_bytes
	// - process_cpu_seconds_total
	// - process_resident_memory_bytes
)
