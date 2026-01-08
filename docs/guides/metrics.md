# Waterflow Metrics Guide

Complete guide to Waterflow Prometheus metrics and monitoring.

## Overview

Waterflow exposes Prometheus metrics at `GET /metrics` endpoint. All metrics use the `waterflow_` prefix.

## Metric Types

### Counter
Monotonically increasing value (resets on restart).  
**Use case**: Total requests, total workflows, errors

### Gauge
Current value that can increase or decrease.  
**Use case**: Running workflows, connected agents, queue size

### Histogram
Distribution of values with configurable buckets.  
**Use case**: Request latency, workflow duration

## Available Metrics

### HTTP Metrics

#### waterflow_http_requests_total

**Type**: Counter  
**Labels**: `method`, `path`, `status`  
**Description**: Total HTTP requests

**Example**:
```promql
# Request rate by endpoint
rate(waterflow_http_requests_total[5m])

# Error rate
rate(waterflow_http_requests_total{status=~"5.."}[5m])
```

#### waterflow_http_request_duration_seconds

**Type**: Histogram  
**Labels**: `method`, `path`  
**Buckets**: [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]  
**Description**: HTTP request latency

**Example**:
```promql
# P95 latency
histogram_quantile(0.95, rate(waterflow_http_request_duration_seconds_bucket[5m]))

# Average latency by endpoint
rate(waterflow_http_request_duration_seconds_sum[5m]) 
/ 
rate(waterflow_http_request_duration_seconds_count[5m])
```

### Workflow Metrics

#### waterflow_workflows_total

**Type**: Counter  
**Labels**: `status` (submitted, completed, failed, cancelled)  
**Description**: Total workflows by status

**Example**:
```promql
# Success rate (last 5 minutes)
rate(waterflow_workflows_total{status="completed"}[5m]) 
/ 
(rate(waterflow_workflows_total{status="completed"}[5m]) + rate(waterflow_workflows_total{status="failed"}[5m]))
* 100

# Failure rate
rate(waterflow_workflows_total{status="failed"}[5m])
```

#### waterflow_workflows_running

**Type**: Gauge  
**Description**: Currently running workflows

**Example**:
```promql
# Current running workflows
waterflow_workflows_running

# Max running workflows (last hour)
max_over_time(waterflow_workflows_running[1h])
```

#### waterflow_workflow_duration_seconds

**Type**: Histogram  
**Labels**: `status` (completed, failed)  
**Buckets**: [1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600]  
**Description**: Workflow execution duration

**Example**:
```promql
# P50, P95, P99 latency
histogram_quantile(0.50, rate(waterflow_workflow_duration_seconds_bucket[5m]))
histogram_quantile(0.95, rate(waterflow_workflow_duration_seconds_bucket[5m]))
histogram_quantile(0.99, rate(waterflow_workflow_duration_seconds_bucket[5m]))

# Average duration by status
rate(waterflow_workflow_duration_seconds_sum{status="completed"}[5m])
/
rate(waterflow_workflow_duration_seconds_count{status="completed"}[5m])
```

### Agent Metrics

#### waterflow_agents_connected

**Type**: Gauge  
**Description**: Number of connected agents

**Example**:
```promql
# Current connected agents
waterflow_agents_connected

# Agent connectivity over time
waterflow_agents_connected[1h]
```

#### waterflow_agents_healthy

**Type**: Gauge  
**Description**: Number of healthy agents

**Example**:
```promql
# Healthy ratio
waterflow_agents_healthy / waterflow_agents_connected

# Unhealthy agents
waterflow_agents_connected - waterflow_agents_healthy
```

#### waterflow_agent_tasks_total

**Type**: Counter  
**Labels**: `agent_id`, `status` (completed, failed)  
**Description**: Tasks executed by agents

**Example**:
```promql
# Task rate per agent
rate(waterflow_agent_tasks_total[5m])

# Failed task rate for specific agent
rate(waterflow_agent_tasks_total{agent_id="agent-1",status="failed"}[5m])

# Top 5 busiest agents
topk(5, rate(waterflow_agent_tasks_total[5m]))
```

### Node Execution Metrics

#### waterflow_node_executions_total

**Type**: Counter  
**Labels**: `node_type`, `status` (success, failure)  
**Description**: Node executions by type

**Example**:
```promql
# Execution rate by node type
rate(waterflow_node_executions_total[5m])

# Failure rate for exec/shell nodes
rate(waterflow_node_executions_total{node_type="exec/shell",status="failure"}[5m])

# Most used node types
topk(5, rate(waterflow_node_executions_total[5m]))
```

#### waterflow_node_execution_duration_seconds

**Type**: Histogram  
**Labels**: `node_type`  
**Buckets**: [0.01, 0.05, 0.1, 0.5, 1, 5, 10, 30, 60, 300, 600]  
**Description**: Node execution duration

**Example**:
```promql
# P95 duration by node type
histogram_quantile(0.95, rate(waterflow_node_execution_duration_seconds_bucket[5m]))

# Average duration for HTTP nodes
rate(waterflow_node_execution_duration_seconds_sum{node_type="http/request"}[5m])
/
rate(waterflow_node_execution_duration_seconds_count{node_type="http/request"}[5m])
```

### API Business Metrics

#### waterflow_workflow_submissions_total

**Type**: Counter  
**Labels**: `result` (success, validation_error, server_error)  
**Description**: Workflow submission results

**Example**:
```promql
# Submission success rate
rate(waterflow_workflow_submissions_total{result="success"}[5m])
/
rate(waterflow_workflow_submissions_total[5m])
* 100

# Validation error rate
rate(waterflow_workflow_submissions_total{result="validation_error"}[5m])
```

#### waterflow_yaml_validations_total

**Type**: Counter  
**Labels**: `result` (success, failure)  
**Description**: YAML validation results

**Example**:
```promql
# Validation success rate
rate(waterflow_yaml_validations_total{result="success"}[5m])
/
rate(waterflow_yaml_validations_total[5m])
* 100
```

## Go Runtime Metrics

Automatically exported by `prometheus/client_golang`:

- `go_goroutines` - Number of goroutines
- `go_memstats_alloc_bytes` - Allocated memory
- `go_memstats_heap_alloc_bytes` - Heap allocated
- `go_memstats_heap_inuse_bytes` - Heap in use
- `process_cpu_seconds_total` - CPU time
- `process_resident_memory_bytes` - RSS memory

## Recording Rules

Pre-aggregate expensive queries:

```yaml
# prometheus/rules.yml
groups:
  - name: waterflow_recording_rules
    interval: 30s
    rules:
      - record: job:waterflow_workflow_success_rate:5m
        expr: |
          rate(waterflow_workflows_total{status="completed"}[5m])
          /
          (rate(waterflow_workflows_total{status="completed"}[5m]) + rate(waterflow_workflows_total{status="failed"}[5m]))

      - record: job:waterflow_api_p95_latency:5m
        expr: |
          histogram_quantile(0.95, rate(waterflow_http_request_duration_seconds_bucket[5m]))
```

## Alert Rules

Example alerts:

```yaml
# prometheus/alerts.yml
groups:
  - name: waterflow_alerts
    interval: 30s
    rules:
      - alert: HighWorkflowFailureRate
        expr: |
          rate(waterflow_workflows_total{status="failed"}[5m])
          /
          rate(waterflow_workflows_total[5m])
          > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High workflow failure rate ({{ $value | humanizePercentage }})"
          description: "More than 5% workflows failing in last 5 minutes"

      - alert: NoAgentsConnected
        expr: waterflow_agents_connected == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "No agents connected"

      - alert: HighAPILatency
        expr: |
          histogram_quantile(0.95, rate(waterflow_http_request_duration_seconds_bucket[5m]))
          > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "API P95 latency > 1s"

      - alert: MemoryLeakSuspected
        expr: |
          rate(go_memstats_heap_alloc_bytes[30m]) > 0
        for: 1h
        labels:
          severity: warning
        annotations:
          summary: "Memory continuously growing"
```

## Best Practices

### Label Cardinality

**Good**:
```go
// Low cardinality labels
WorkflowsTotal.WithLabelValues(status).Inc()  // 4 values
```

**Bad**:
```go
// High cardinality labels (avoid)
WorkflowsTotal.WithLabelValues(workflowID).Inc()  // Thousands of unique values
```

### Histogram Buckets

Choose buckets based on expected distribution:

```go
// For API latency (ms scale)
Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5}

// For workflow duration (seconds to minutes)
Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600}
```

### Naming Conventions

- Use `_total` suffix for counters
- Use base unit (seconds, bytes) not multiples
- Group related metrics with common prefix

## Integration Examples

### Using Trackers

```go
import "github.com/Websoft9/waterflow/pkg/metrics"

// Initialize trackers
workflowTracker := metrics.NewWorkflowTracker()
nodeTracker := metrics.NewNodeTracker()
agentTracker := metrics.NewAgentTracker()

// Track workflow submission
workflowTracker.TrackSubmission("wf-123", true)

// Track workflow completion
workflowTracker.TrackCompletion("wf-123", "completed")

// Track node execution
done := nodeTracker.TrackNodeStart("exec/shell")
// ... execute node ...
done(err == nil)

// Track agent connection
agentTracker.TrackConnection("agent-1")
agentTracker.TrackTaskExecution("agent-1", true)
```

### Custom Metrics

```go
import "github.com/prometheus/client_golang/prometheus"

var myMetric = prometheus.NewCounter(
    prometheus.CounterOpts{
        Name: "waterflow_custom_events_total",
        Help: "Custom events",
    },
)

func init() {
    prometheus.MustRegister(myMetric)
}
```

## References

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Metric Types](https://prometheus.io/docs/concepts/metric_types/)
- [Naming Best Practices](https://prometheus.io/docs/practices/naming/)
- [Histogram vs Summary](https://prometheus.io/docs/practices/histograms/)
