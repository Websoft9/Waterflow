# Story 7.5: Prometheus 指标导出

Status: ready-for-dev

## Story

As a **系统管理员**,  
I want **导出 Prometheus 指标**,  
So that **监控系统运行状态和性能**。

## Context

这是 Epic 7 (生产级可靠性) 的**第五个 Story**,实现**完善的 Prometheus 指标导出系统**。该 Story 在现有基础指标上进行扩展,添加工作流生命周期、Agent 健康、节点执行等关键业务指标,并提供 Grafana Dashboard 模板用于可视化监控。

**前置依赖:**
- ✅ Story 1.2 - REST API 服务框架 (已有基础 `/metrics` 端点)
- ✅ Story 1.8 - Temporal SDK 集成 (工作流状态查询)
- ✅ Story 2.1 - Agent Worker 框架 (Agent 连接管理)
- ✅ 现有实现 - pkg/metrics 包 (HTTP 请求指标)
- ✅ 现有实现 - pkg/middleware/metrics.go (指标中间件)

**Epic 背景:**  
Epic 7 专注于**生产级可靠性**。本 Story 提供全面的可观测性支持,使运维团队能够:
- 📊 **实时监控系统健康** - 工作流执行状态、成功率、延迟
- 📊 **资源使用分析** - API 请求、Agent 连接、节点执行时长
- 📊 **故障快速定位** - 失败率、错误类型分布
- 📊 **容量规划** - 并发工作流数、吞吐量趋势
- 📊 **SLO 验证** - P50/P95/P99 延迟监控

**业务价值:**
- 🎯 **主动监控** - 实时掌握系统状态,问题预警
- 🎯 **数据驱动** - 基于指标优化系统性能
- 🎯 **故障快速恢复** - 可视化仪表板加速问题定位
- 🎯 **生产就绪** - 符合企业级监控标准
- 🎯 **成本优化** - 识别资源瓶颈和优化机会

**现有指标分析:**
```go
// 已实现 (pkg/metrics/metrics.go):
✅ waterflow_http_requests_total        // HTTP 请求计数器
   Labels: method, path, status
   
✅ waterflow_http_request_duration_seconds  // HTTP 请求延迟直方图
   Labels: method, path
   Buckets: [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
   
✅ waterflow_workflows_total            // 工作流计数器 (已定义未使用)
   Labels: status

✅ /metrics 端点已暴露 (internal/api/handlers.go)
   - promhttp.Handler() 集成
   - GET /metrics 路由注册
```

**需要扩展的指标:**

**1. 工作流生命周期指标**
- 执行中的工作流数量 (Gauge)
- 工作流执行时长分布 (Histogram)
- 各状态工作流计数 (Counter)

**2. Agent 健康指标**
- 已连接 Agent 数量 (Gauge)
- 健康 Agent 数量 (Gauge)
- Agent 任务执行计数 (Counter)

**3. 节点执行指标**
- 节点执行时长分布 (Histogram)
- 各类型节点调用计数 (Counter)
- 节点失败率 (Counter)

**4. API 业务指标**
- 工作流提交速率 (Counter)
- YAML 验证成功/失败率 (Counter)

**Prometheus 指标类型选择:**

**Counter (累加计数器):**
- 工作流提交总数
- HTTP 请求总数
- 节点执行总数
- 失败总数

**Gauge (瞬时值):**
- 执行中工作流数
- 已连接 Agent 数
- 队列长度

**Histogram (分布统计):**
- HTTP 请求延迟
- 工作流执行时长
- 节点执行时长

**监控架构:**
```
┌────────────────────────────────────────────────────────────┐
│                   Waterflow Server                         │
│                                                            │
│  ┌──────────────────────────────────────────────────┐     │
│  │  Metrics Collectors                              │     │
│  ├──────────────────────────────────────────────────┤     │
│  │  - HTTP Middleware  → Request metrics            │     │
│  │  - Workflow Tracker → Lifecycle metrics          │     │
│  │  - Agent Monitor    → Health metrics             │     │
│  │  - Node Interceptor → Execution metrics          │     │
│  └──────────────────────────────────────────────────┘     │
│                         ↓                                  │
│  ┌──────────────────────────────────────────────────┐     │
│  │  Prometheus Registry (pkg/metrics)               │     │
│  └──────────────────────────────────────────────────┘     │
│                         ↓                                  │
│              GET /metrics (promhttp)                       │
└────────────────────────────────────────────────────────────┘
                         ↓
┌────────────────────────────────────────────────────────────┐
│              Prometheus Server (Pull Model)                │
│  - Scrape interval: 15s                                    │
│  - Retention: 15 days                                      │
│  - PromQL queries                                          │
└────────────────────────────────────────────────────────────┘
                         ↓
┌────────────────────────────────────────────────────────────┐
│                  Grafana Dashboard                         │
│  - Workflow Overview (成功率/吞吐量/延迟)                  │
│  - Agent Health (连接数/任务分布)                         │
│  - API Performance (P50/P95/P99)                           │
│  - System Resources (Go runtime metrics)                   │
└────────────────────────────────────────────────────────────┘
```

**本 Story 的范围 (MVP):**
- ✅ 扩展 pkg/metrics 包添加工作流/Agent/节点指标
- ✅ 实现指标收集器 (Workflow/Agent/Node Tracker)
- ✅ 集成到现有 /metrics 端点
- ✅ 提供 Grafana Dashboard JSON 模板
- ✅ 添加 Prometheus + Grafana Docker Compose 配置
- ✅ 文档化所有指标定义和使用方法
- ❌ 告警规则 (AlertManager) - Post-MVP
- ❌ 自定义指标导出器接口 - Post-MVP

## Acceptance Criteria

### AC1: /metrics 端点导出 Prometheus 格式指标

**Given** Server 运行中  
**When** 访问 `GET /metrics`  
**Then** 返回 Prometheus text exposition 格式  
**And** Content-Type 为 `text/plain; version=0.0.4; charset=utf-8`  
**And** 包含所有注册的指标  
**And** 响应时间 < 100ms

**Implementation Notes:**

**现有实现已完成:**
```go
// internal/api/handlers.go (Line 109)
func (h *Handlers) Metrics(w http.ResponseWriter, r *http.Request) {
    promhttp.Handler().ServeHTTP(w, r)
}

// internal/api/router.go (Line 66)
router.HandleFunc("/metrics", h.Metrics).Methods(http.MethodGet)
```

**验证脚本:**
```bash
#!/bin/bash
# test/integration/metrics_endpoint_test.sh

set -e

echo "=== Metrics Endpoint Test ==="

# 1. 测试端点可访问性
echo "Testing /metrics endpoint..."
RESPONSE=$(curl -s -w "\n%{http_code}" http://localhost:8080/metrics)
HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" != "200" ]; then
    echo "❌ Metrics endpoint returned $HTTP_CODE"
    exit 1
fi

echo "✅ Endpoint accessible (HTTP 200)"

# 2. 验证 Content-Type
CONTENT_TYPE=$(curl -s -I http://localhost:8080/metrics \
    | grep -i "content-type" \
    | awk '{print $2}' \
    | tr -d '\r')

if [[ ! "$CONTENT_TYPE" =~ "text/plain" ]]; then
    echo "❌ Invalid Content-Type: $CONTENT_TYPE"
    exit 1
fi

echo "✅ Content-Type correct: $CONTENT_TYPE"

# 3. 验证 Prometheus 格式
if ! echo "$BODY" | grep -q "# HELP"; then
    echo "❌ Missing HELP comments"
    exit 1
fi

if ! echo "$BODY" | grep -q "# TYPE"; then
    echo "❌ Missing TYPE metadata"
    exit 1
fi

echo "✅ Prometheus format validated"

# 4. 验证响应时间
START=$(date +%s%N)
curl -s http://localhost:8080/metrics > /dev/null
END=$(date +%s%N)
DURATION=$(( (END - START) / 1000000 ))  # Convert to ms

echo "Response time: ${DURATION}ms"

if [ $DURATION -gt 100 ]; then
    echo "⚠️  Response time > 100ms"
fi

echo "✅ Metrics endpoint test passed"
```

### AC2: 工作流指标 - 提交数、执行中数量、成功/失败率

**Given** 工作流正在执行  
**When** 查询 /metrics  
**Then** 包含以下指标:
- `waterflow_workflows_total{status="submitted|completed|failed"}` (Counter)
- `waterflow_workflows_running` (Gauge)
- `waterflow_workflow_duration_seconds` (Histogram)

**Implementation Notes:**

**扩展 pkg/metrics/metrics.go:**
```go
// pkg/metrics/metrics.go

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// === HTTP Metrics (已有) ===
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "waterflow_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "waterflow_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// === Workflow Metrics (扩展) ===
	
	// WorkflowsTotal tracks total workflow submissions by status
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
)
```

**集成到 Workflow Handlers:**
```go
// internal/api/workflow_handlers.go

func (h *WorkflowHandlers) SubmitWorkflow(w http.ResponseWriter, r *http.Request) {
	// ... existing validation code ...

	// Submit to Temporal
	workflowID, err := h.temporalClient.SubmitWorkflow(ctx, yamlContent, variables)
	if err != nil {
		// ... error handling ...
		metrics.WorkflowsTotal.WithLabelValues("failed").Inc()
		return
	}

	// Track successful submission
	metrics.WorkflowsTotal.WithLabelValues("submitted").Inc()
	metrics.WorkflowsRunning.Inc()

	// ... return response ...
}
```

**Workflow 完成时更新指标:**
```go
// pkg/temporal/workflow.go

import (
	"time"
	"github.com/Websoft9/waterflow/pkg/metrics"
)

func WorkflowExecutor(ctx workflow.Context, workflowID string) error {
	startTime := time.Now()
	
	// ... existing workflow logic ...

	// On completion
	duration := time.Since(startTime).Seconds()
	metrics.WorkflowsRunning.Dec()
	
	if err != nil {
		metrics.WorkflowsTotal.WithLabelValues("failed").Inc()
		metrics.WorkflowDuration.WithLabelValues("failed").Observe(duration)
		return err
	}
	
	metrics.WorkflowsTotal.WithLabelValues("completed").Inc()
	metrics.WorkflowDuration.WithLabelValues("completed").Observe(duration)
	return nil
}
```

**PromQL 查询示例:**
```promql
# 工作流成功率 (最近 5 分钟)
rate(waterflow_workflows_total{status="completed"}[5m]) 
/ 
(rate(waterflow_workflows_total{status="completed"}[5m]) + rate(waterflow_workflows_total{status="failed"}[5m]))

# 执行中的工作流数
waterflow_workflows_running

# 工作流执行时长 P95
histogram_quantile(0.95, rate(waterflow_workflow_duration_seconds_bucket[5m]))
```

### AC3: API 请求延迟直方图 (已实现,验证完善)

**Given** HTTP 请求到达  
**When** 中间件记录延迟  
**Then** `waterflow_http_request_duration_seconds` 直方图更新  
**And** 标签包含 method 和 path  
**And** 桶覆盖 5ms 到 10s 范围

**Implementation Notes:**

**现有实现已完成:**
```go
// pkg/middleware/metrics.go (已实现)
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		status := fmt.Sprintf("%d", wrapped.statusCode)

		// Record metrics
		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	})
}
```

**验证测试:**
```go
// pkg/middleware/metrics_test.go

func TestMetricsMiddleware(t *testing.T) {
	// Reset metrics
	metrics.HTTPRequestsTotal.Reset()
	metrics.HTTPRequestDuration.Reset()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	metricsHandler := Metrics(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	metricsHandler.ServeHTTP(rec, req)

	// Verify counter incremented
	counter := testutil.ToFloat64(metrics.HTTPRequestsTotal.WithLabelValues("GET", "/test", "200"))
	if counter != 1.0 {
		t.Errorf("Expected counter=1.0, got %f", counter)
	}

	// Verify histogram recorded
	histogram := testutil.ToFloat64(metrics.HTTPRequestDuration.WithLabelValues("GET", "/test"))
	if histogram == 0 {
		t.Error("Expected histogram observation > 0")
	}
}
```

### AC4: Agent 连接数和健康数量指标

**Given** Agents 连接到 Server  
**When** 查询 /metrics  
**Then** 包含以下指标:
- `waterflow_agents_connected` (Gauge) - 已连接 Agent 数
- `waterflow_agents_healthy` (Gauge) - 健康 Agent 数
- `waterflow_agent_tasks_total{task_queue}` (Counter) - 各队列任务数

**Implementation Notes:**

**扩展 pkg/metrics/metrics.go:**
```go
// pkg/metrics/metrics.go

var (
	// ... existing metrics ...

	// === Agent Metrics ===
	
	// AgentsConnected tracks number of connected agents
	AgentsConnected = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "waterflow_agents_connected",
			Help: "Number of connected agents",
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
		[]string{"task_queue", "status"}, // task_queue: linux-amd64, windows-amd64; status: success, failed
	)
)
```

**Agent 监控实现:**

由于 Waterflow 使用 Temporal Worker 架构,Agent 连接信息需要通过 Temporal API 查询。

**Temporal SDK 要求:**
- Temporal Go SDK >= v1.22.0 (支持 WorkflowService.DescribeTaskQueue)
- 使用 `workflowservice.DescribeTaskQueueRequest` 查询 Task Queue 状态
- API Reference: [Temporal API Docs](https://docs.temporal.io/references/server-api#describetaskqueue)

**实现方案:**
```go
// internal/server/agent_monitor.go (新建)

package server

import (
	"context"
	"time"

	"github.com/Websoft9/waterflow/pkg/metrics"
	"github.com/Websoft9/waterflow/pkg/temporal"
	"go.temporal.io/api/workflowservice/v1"
	"go.uber.org/zap"
)

// AgentMonitor periodically updates agent metrics
type AgentMonitor struct {
	temporalClient *temporal.Client
	logger         *zap.Logger
	interval       time.Duration
	stopCh         chan struct{}
}

// NewAgentMonitor creates a new agent monitor
func NewAgentMonitor(tc *temporal.Client, logger *zap.Logger, interval time.Duration) *AgentMonitor {
	return &AgentMonitor{
		temporalClient: tc,
		logger:         logger,
		interval:       interval,
		stopCh:         make(chan struct{}),
	}
}

// Start begins monitoring agent metrics
func (m *AgentMonitor) Start() {
	go m.run()
}

// Stop stops the monitor
func (m *AgentMonitor) Stop() {
	close(m.stopCh)
}

func (m *AgentMonitor) run() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.updateMetrics()
		case <-m.stopCh:
			return
		}
	}
}

func (m *AgentMonitor) updateMetrics() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Query Temporal for worker pollers (agents)
	// 使用 Temporal SDK 原生 API: WorkflowService.DescribeTaskQueue
	taskQueues := []string{"linux-amd64", "windows-amd64", "darwin-amd64"}
	
	totalConnected := 0
	totalHealthy := 0

	for _, taskQueue := range taskQueues {
		// 调用 Temporal WorkflowService API
		request := &workflowservice.DescribeTaskQueueRequest{
			Namespace: m.temporalClient.Namespace(),
			TaskQueue: &taskqueue.TaskQueue{
				Name: taskQueue,
				Kind: enumspb.TASK_QUEUE_KIND_NORMAL,
			},
			TaskQueueType: enumspb.TASK_QUEUE_TYPE_ACTIVITY,
		}
		
		desc, err := m.temporalClient.WorkflowService().DescribeTaskQueue(ctx, request)
		if err != nil {
			m.logger.Warn("Failed to describe task queue",
				zap.String("task_queue", taskQueue),
				zap.Error(err),
			)
			continue
		}

		// Count pollers (connected agents)
		// desc.Pollers 包含所有 Worker 的 poller 信息
		pollers := len(desc.GetPollers())
		totalConnected += pollers

		// Count healthy pollers (last seen < 30s)
		// Temporal API 返回 PollerInfo 包含 LastAccessTime (timestamppb.Timestamp)
		healthy := 0
		for _, poller := range desc.GetPollers() {
			lastAccess := poller.GetLastAccessTime().AsTime()
			if time.Since(lastAccess) < 30*time.Second {
				healthy++
			}
		}
		totalHealthy += healthy
	}

	// Update metrics
	metrics.AgentsConnected.Set(float64(totalConnected))
	metrics.AgentsHealthy.Set(float64(totalHealthy))
}
```

**实现说明:**
- ✅ 使用 Temporal Go SDK 原生 API: `WorkflowService.DescribeTaskQueue`
- ✅ 无需自定义扩展,SDK 已提供完整支持
- ✅ API 稳定,Temporal Server v1.14+ 即支持
- ✅ 返回精确的 Pollers 信息 (Identity, LastAccessTime, RatePerSecond)

**备选方案 (仅在极端情况下使用):**
如果生产环境 Temporal 版本过低 (<v1.14),可临时使用基于 Workflow 执行的估算:
```go
// 仅作为降级方案,不推荐
func estimateAgentMetrics() {
	// 统计最近 1 分钟活跃的 WorkflowID
	// 精度低,仅用于版本兼容性
	metrics.AgentsConnected.Set(float64(estimatedCount))
}
```

### AC5: 节点执行时长指标

**Given** 节点执行完成  
**When** Activity 结束  
**Then** 记录 `waterflow_node_duration_seconds{node_type}` (Histogram)  
**And** 记录 `waterflow_node_executions_total{node_type, status}` (Counter)

**Implementation Notes:**

**扩展 pkg/metrics/metrics.go:**
```go
// pkg/metrics/metrics.go

var (
	// ... existing metrics ...

	// === Node Metrics ===
	
	// NodeExecutionsTotal tracks total node executions
	NodeExecutionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "waterflow_node_executions_total",
			Help: "Total number of node executions",
		},
		[]string{"node_type", "status"}, // node_type: exec/shell, http/request; status: success, failed
	)

	// NodeDuration tracks node execution duration distribution
	NodeDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "waterflow_node_duration_seconds",
			Help: "Node execution duration in seconds",
			// Buckets optimized for node execution: 10ms to 5 minutes
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 5, 10, 30, 60, 300},
		},
		[]string{"node_type"},
	)
)
```

**集成到 Activity 执行:**
```go
// pkg/temporal/activity.go

import (
	"time"
	"github.com/Websoft9/waterflow/pkg/metrics"
)

func ExecuteNodeActivity(ctx context.Context, input NodeInput) (*NodeOutput, error) {
	startTime := time.Now()
	nodeType := input.Uses // e.g., "exec/shell", "http/request"

	// Execute node
	output, err := executeNode(ctx, input)

	// Record metrics
	duration := time.Since(startTime).Seconds()
	metrics.NodeDuration.WithLabelValues(nodeType).Observe(duration)

	if err != nil {
		metrics.NodeExecutionsTotal.WithLabelValues(nodeType, "failed").Inc()
		return nil, err
	}

	metrics.NodeExecutionsTotal.WithLabelValues(nodeType, "success").Inc()
	return output, nil
}
```

**PromQL 查询示例:**
```promql
# 各类型节点执行成功率
rate(waterflow_node_executions_total{status="success"}[5m]) 
/ 
rate(waterflow_node_executions_total[5m])

# HTTP 请求节点 P95 延迟
histogram_quantile(0.95, rate(waterflow_node_duration_seconds_bucket{node_type="http/request"}[5m]))

# 最慢的节点类型 (平均延迟)
avg(rate(waterflow_node_duration_seconds_sum[5m]) / rate(waterflow_node_duration_seconds_count[5m])) by (node_type)
```

### AC6: 提供 Grafana Dashboard 模板

**Given** Grafana 连接到 Prometheus  
**When** 导入 Dashboard JSON  
**Then** 显示完整监控面板  
**And** 包含工作流概览、API 性能、Agent 健康、节点执行统计

**Implementation Notes:**

**Dashboard 结构:**
```json
{
  "dashboard": {
    "title": "Waterflow Monitoring",
    "panels": [
      {
        "title": "Workflow Overview",
        "targets": [
          {"expr": "rate(waterflow_workflows_total[5m])"},
          {"expr": "waterflow_workflows_running"}
        ]
      },
      {
        "title": "Workflow Success Rate",
        "targets": [
          {"expr": "rate(waterflow_workflows_total{status=\"completed\"}[5m]) / (rate(waterflow_workflows_total{status=\"completed\"}[5m]) + rate(waterflow_workflows_total{status=\"failed\"}[5m]))"}
        ]
      },
      {
        "title": "API Latency (P50/P95/P99)",
        "targets": [
          {"expr": "histogram_quantile(0.50, rate(waterflow_http_request_duration_seconds_bucket[5m]))"},
          {"expr": "histogram_quantile(0.95, rate(waterflow_http_request_duration_seconds_bucket[5m]))"},
          {"expr": "histogram_quantile(0.99, rate(waterflow_http_request_duration_seconds_bucket[5m]))"}
        ]
      },
      {
        "title": "Agent Health",
        "targets": [
          {"expr": "waterflow_agents_connected"},
          {"expr": "waterflow_agents_healthy"}
        ]
      },
      {
        "title": "Node Execution Rate by Type",
        "targets": [
          {"expr": "rate(waterflow_node_executions_total[5m])"}
        ]
      }
    ]
  }
}
```

**完整 Dashboard 文件:**
```bash
# deployments/grafana/dashboards/waterflow-monitoring.json

{
  "annotations": {
    "list": [
      {
        "builtIn": 1,
        "datasource": "-- Grafana --",
        "enable": true,
        "hide": true,
        "iconColor": "rgba(0, 211, 255, 1)",
        "name": "Annotations & Alerts",
        "type": "dashboard"
      }
    ]
  },
  "editable": true,
  "gnetId": null,
  "graphTooltip": 0,
  "id": null,
  "links": [],
  "panels": [
    {
      "datasource": "Prometheus",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "palette-classic"
          },
          "custom": {
            "axisLabel": "",
            "axisPlacement": "auto",
            "barAlignment": 0,
            "drawStyle": "line",
            "fillOpacity": 10,
            "gradientMode": "none",
            "hideFrom": {
              "tooltip": false,
              "viz": false,
              "legend": false
            },
            "lineInterpolation": "linear",
            "lineWidth": 1,
            "pointSize": 5,
            "scaleDistribution": {
              "type": "linear"
            },
            "showPoints": "never",
            "spanNulls": true
          },
          "mappings": [],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "green",
                "value": null
              }
            ]
          },
          "unit": "short"
        }
      },
      "gridPos": {
        "h": 8,
        "w": 12,
        "x": 0,
        "y": 0
      },
      "id": 1,
      "options": {
        "legend": {
          "calcs": [],
          "displayMode": "list",
          "placement": "bottom"
        },
        "tooltip": {
          "mode": "single"
        }
      },
      "targets": [
        {
          "expr": "rate(waterflow_workflows_total{status=\"submitted\"}[5m])",
          "legendFormat": "Submitted",
          "refId": "A"
        },
        {
          "expr": "rate(waterflow_workflows_total{status=\"completed\"}[5m])",
          "legendFormat": "Completed",
          "refId": "B"
        },
        {
          "expr": "rate(waterflow_workflows_total{status=\"failed\"}[5m])",
          "legendFormat": "Failed",
          "refId": "C"
        }
      ],
      "title": "Workflow Rate (per second)",
      "type": "timeseries"
    },
    {
      "datasource": "Prometheus",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "thresholds"
          },
          "mappings": [],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "green",
                "value": null
              },
              {
                "color": "yellow",
                "value": 80
              },
              {
                "color": "red",
                "value": 90
              }
            ]
          },
          "unit": "percentunit"
        }
      },
      "gridPos": {
        "h": 8,
        "w": 12,
        "x": 12,
        "y": 0
      },
      "id": 2,
      "options": {
        "orientation": "auto",
        "reduceOptions": {
          "values": false,
          "calcs": [
            "lastNotNull"
          ],
          "fields": ""
        },
        "showThresholdLabels": false,
        "showThresholdMarkers": true,
        "text": {}
      },
      "targets": [
        {
          "expr": "rate(waterflow_workflows_total{status=\"completed\"}[5m]) / (rate(waterflow_workflows_total{status=\"completed\"}[5m]) + rate(waterflow_workflows_total{status=\"failed\"}[5m]))",
          "refId": "A"
        }
      ],
      "title": "Workflow Success Rate",
      "type": "gauge"
    },
    {
      "datasource": "Prometheus",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "palette-classic"
          },
          "custom": {
            "axisLabel": "Duration (s)",
            "axisPlacement": "auto",
            "barAlignment": 0,
            "drawStyle": "line",
            "fillOpacity": 10,
            "gradientMode": "none",
            "hideFrom": {
              "tooltip": false,
              "viz": false,
              "legend": false
            },
            "lineInterpolation": "linear",
            "lineWidth": 1,
            "pointSize": 5,
            "scaleDistribution": {
              "type": "linear"
            },
            "showPoints": "never",
            "spanNulls": true
          },
          "mappings": [],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "green",
                "value": null
              }
            ]
          },
          "unit": "s"
        }
      },
      "gridPos": {
        "h": 8,
        "w": 12,
        "x": 0,
        "y": 8
      },
      "id": 3,
      "options": {
        "legend": {
          "calcs": [],
          "displayMode": "list",
          "placement": "bottom"
        },
        "tooltip": {
          "mode": "single"
        }
      },
      "targets": [
        {
          "expr": "histogram_quantile(0.50, rate(waterflow_http_request_duration_seconds_bucket{path=\"/v1/workflows\"}[5m]))",
          "legendFormat": "P50",
          "refId": "A"
        },
        {
          "expr": "histogram_quantile(0.95, rate(waterflow_http_request_duration_seconds_bucket{path=\"/v1/workflows\"}[5m]))",
          "legendFormat": "P95",
          "refId": "B"
        },
        {
          "expr": "histogram_quantile(0.99, rate(waterflow_http_request_duration_seconds_bucket{path=\"/v1/workflows\"}[5m]))",
          "legendFormat": "P99",
          "refId": "C"
        }
      ],
      "title": "API Latency (Submit Workflow)",
      "type": "timeseries"
    },
    {
      "datasource": "Prometheus",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "palette-classic"
          },
          "custom": {
            "axisLabel": "",
            "axisPlacement": "auto",
            "barAlignment": 0,
            "drawStyle": "line",
            "fillOpacity": 10,
            "gradientMode": "none",
            "hideFrom": {
              "tooltip": false,
              "viz": false,
              "legend": false
            },
            "lineInterpolation": "linear",
            "lineWidth": 1,
            "pointSize": 5,
            "scaleDistribution": {
              "type": "linear"
            },
            "showPoints": "never",
            "spanNulls": true
          },
          "mappings": [],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "green",
                "value": null
              }
            ]
          },
          "unit": "short"
        }
      },
      "gridPos": {
        "h": 8,
        "w": 12,
        "x": 12,
        "y": 8
      },
      "id": 4,
      "options": {
        "legend": {
          "calcs": [],
          "displayMode": "list",
          "placement": "bottom"
        },
        "tooltip": {
          "mode": "single"
        }
      },
      "targets": [
        {
          "expr": "waterflow_agents_connected",
          "legendFormat": "Connected",
          "refId": "A"
        },
        {
          "expr": "waterflow_agents_healthy",
          "legendFormat": "Healthy",
          "refId": "B"
        }
      ],
      "title": "Agent Status",
      "type": "timeseries"
    },
    {
      "datasource": "Prometheus",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "palette-classic"
          },
          "custom": {
            "axisLabel": "",
            "axisPlacement": "auto",
            "barAlignment": 0,
            "drawStyle": "line",
            "fillOpacity": 10,
            "gradientMode": "none",
            "hideFrom": {
              "tooltip": false,
              "viz": false,
              "legend": false
            },
            "lineInterpolation": "linear",
            "lineWidth": 1,
            "pointSize": 5,
            "scaleDistribution": {
              "type": "linear"
            },
            "showPoints": "never",
            "spanNulls": true
          },
          "mappings": [],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "green",
                "value": null
              }
            ]
          },
          "unit": "ops"
        }
      },
      "gridPos": {
        "h": 8,
        "w": 24,
        "x": 0,
        "y": 16
      },
      "id": 5,
      "options": {
        "legend": {
          "calcs": [],
          "displayMode": "table",
          "placement": "right"
        },
        "tooltip": {
          "mode": "single"
        }
      },
      "targets": [
        {
          "expr": "rate(waterflow_node_executions_total{status=\"success\"}[5m])",
          "legendFormat": "{{node_type}} (success)",
          "refId": "A"
        }
      ],
      "title": "Node Execution Rate by Type",
      "type": "timeseries"
    }
  ],
  "schemaVersion": 27,
  "style": "dark",
  "tags": ["waterflow"],
  "templating": {
    "list": []
  },
  "time": {
    "from": "now-1h",
    "to": "now"
  },
  "timepicker": {},
  "timezone": "",
  "title": "Waterflow Monitoring Dashboard",
  "uid": "waterflow-monitoring",
  "version": 1
}
```

## Tasks / Subtasks

### Task 1: 扩展 pkg/metrics 包 (AC2, AC4, AC5)

- [ ] 1.1 添加工作流指标
  - WorkflowsTotal (Counter)
  - WorkflowsRunning (Gauge)
  - WorkflowDuration (Histogram)

- [ ] 1.2 添加 Agent 指标
  - AgentsConnected (Gauge)
  - AgentsHealthy (Gauge)
  - AgentTasksTotal (Counter)

- [ ] 1.3 添加节点执行指标
  - NodeExecutionsTotal (Counter)
  - NodeDuration (Histogram)

- [ ] 1.4 更新 pkg/metrics/README.md
  - 文档化所有新增指标
  - 添加标签说明和示例 PromQL

### Task 2: 集成指标收集器 (AC2, AC5)

- [ ] 2.1 Workflow 指标收集
  - SubmitWorkflow 提交时计数
  - WorkflowExecutor 完成时更新
  - GetWorkflowStatus 查询时更新运行数

- [ ] 2.2 节点执行指标收集
  - ExecuteNodeActivity 记录时长和状态
  - 错误时标记 failed 状态

- [ ] 2.3 集成 Temporal DescribeTaskQueue API
  - 验证 Temporal SDK 版本 >= v1.22.0
  - 使用 WorkflowService.DescribeTaskQueue 原生 API
  - 封装查询逻辑到 pkg/temporal/client.go
  - 实现 DescribeTaskQueue 便捷方法 (封装 request 构建)
  - 单元测试 API 调用和响应解析

- [ ] 2.4 实现 Agent Monitor
  - internal/server/agent_monitor.go
  - 定期查询 Temporal 更新 Agent 指标
  - 集成到 Server 启动流程

### Task 3: 验证 /metrics 端点 (AC1, AC3)

- [ ] 3.1 编写集成测试
  - test/integration/metrics_endpoint_test.sh
  - 验证 HTTP 200 和 Content-Type
  - 验证 Prometheus 格式

- [ ] 3.2 验证现有 HTTP 指标
  - pkg/middleware/metrics_test.go 扩展
  - 覆盖所有 API 路径

- [ ] 3.3 性能测试
  - 验证响应时间 < 100ms
  - 大量指标下的性能

### Task 4: Grafana Dashboard (AC6)

- [ ] 4.1 创建 Dashboard JSON
  - deployments/grafana/dashboards/waterflow-monitoring.json
  - 5 个核心面板 (工作流/API/Agent/节点/系统)

- [ ] 4.2 添加预配置 Grafana
  - deployments/grafana/provisioning/datasources.yaml
  - deployments/grafana/provisioning/dashboards.yaml

- [ ] 4.3 更新 Docker Compose
  - 添加 Prometheus 服务
  - 添加 Grafana 服务
  - Volume 配置

### Task 5: Prometheus 配置

- [ ] 5.1 创建 Prometheus 配置
  - deployments/prometheus/prometheus.yml
  - Scrape Waterflow Server /metrics
  - 15s 抓取间隔

- [ ] 5.2 添加告警规则 (可选)
  - deployments/prometheus/alerts.yml
  - 工作流失败率 > 5%
  - API P99 > 1s

### Task 6: 文档

- [ ] 6.1 指标参考文档
  - docs/monitoring/metrics-reference.md
  - 所有指标定义
  - PromQL 查询示例

- [ ] 6.2 监控部署指南
  - docs/monitoring/setup-guide.md
  - Prometheus + Grafana 部署
  - Dashboard 导入步骤

- [ ] 6.3 更新 README
  - 监控章节
  - Grafana Dashboard 链接

### Task 7: 单元测试

- [ ] 7.1 指标收集测试
  - pkg/metrics/metrics_test.go
  - 验证指标注册

- [ ] 7.2 集成测试
  - test/integration/metrics_collection_test.go
  - 端到端验证指标更新

### Task 8: 验收测试

- [ ] 8.1 指标准确性验证
  - 提交工作流,验证计数器
  - 验证延迟直方图桶分布

- [ ] 8.2 Grafana Dashboard 验证
  - 导入 Dashboard
  - 验证所有面板正常显示

## Dev Notes

### Architecture Alignment

**指标系统架构 (本 Story):**
- ✅ Prometheus Pull 模型 - 标准监控方案
- ✅ 中间件集成 - 自动收集 HTTP 指标
- ✅ 业务指标 - Workflow/Agent/Node 生命周期
- ✅ Grafana 可视化 - 运维友好

**与 Temporal 架构集成:**
- ✅ Event Sourcing - 指标不影响工作流状态
- ✅ Worker 查询 - 通过 DescribeTaskQueue 获取 Agent 信息
- ✅ 异步收集 - 指标收集不阻塞业务逻辑

**与可观测性栈集成:**
- ✅ Prometheus - 时序数据存储
- ✅ Grafana - 可视化仪表板
- 🔜 AlertManager - 告警 (Post-MVP)
- 🔜 Jaeger/Zipkin - 分布式追踪 (Epic 10+)

### Project Structure

```
pkg/
└── metrics/
    ├── metrics.go              # 🔧 扩展 - 新增 Workflow/Agent/Node 指标
    └── README.md               # 🔧 更新 - 新增指标文档

internal/
├── server/
│   ├── server.go               # 🔧 集成 AgentMonitor
│   └── agent_monitor.go        # 🆕 Agent 指标监控器
├── api/
│   ├── handlers.go             # ✅ /metrics 已实现
│   └── workflow_handlers.go    # 🔧 添加指标收集

pkg/
├── temporal/
│   ├── client.go               # 🔧 添加 DescribeTaskQueue
│   ├── workflow.go             # 🔧 WorkflowExecutor 添加指标
│   └── activity.go             # 🔧 ExecuteNodeActivity 添加指标
└── middleware/
    ├── metrics.go              # ✅ 已实现
    └── metrics_test.go         # ✅ 已测试

deployments/
├── prometheus/
│   ├── prometheus.yml          # 🆕 Prometheus 配置
│   └── alerts.yml              # 🆕 告警规则 (可选)
├── grafana/
│   ├── provisioning/
│   │   ├── datasources.yaml    # 🆕 数据源配置
│   │   └── dashboards.yaml     # 🆕 Dashboard 配置
│   └── dashboards/
│       └── waterflow-monitoring.json  # 🆕 监控仪表板
└── docker-compose.yaml         # 🔧 添加 Prometheus + Grafana

docs/
└── monitoring/
    ├── metrics-reference.md    # 🆕 指标参考文档
    └── setup-guide.md          # 🆕 监控部署指南

test/
└── integration/
    ├── metrics_endpoint_test.sh           # 🆕 端点测试
    └── metrics_collection_test.go         # 🆕 指标收集测试
```

### Temporal API 验证和风险评估

**DescribeTaskQueue API 验证:**

✅ **API 可用性确认:**
- Temporal Go SDK v1.22.0+ 原生支持
- API: `workflowservice.WorkflowServiceClient.DescribeTaskQueue`
- Request: `workflowservice.DescribeTaskQueueRequest`
- Response: `workflowservice.DescribeTaskQueueResponse`
  - Pollers: `[]*PollerInfo` (Identity, LastAccessTime, RatePerSecond)

✅ **Waterflow 当前环境:**
- go.mod: `go.temporal.io/sdk v1.25.0` (满足要求)
- Temporal Server 部署版本需 >= v1.14.0

**实现复杂度评估:**
- 🟢 **低复杂度** (2-4小时实现)
  - 原因: SDK 原生支持,无需自定义扩展
  - 步骤:
    1. 封装 `pkg/temporal.Client.DescribeTaskQueue()` 便捷方法
    2. 实现 `AgentMonitor.updateMetrics()` 调用封装方法
    3. 单元测试 API 调用和 mock 响应

**潜在风险和缓解措施:**

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|----------|
| Temporal Server 版本过低 | 低 | 中 | 部署前验证版本,文档说明最低要求 |
| API 调用频率限制 | 低 | 低 | 使用 30s 采样间隔,避免过度查询 |
| 网络延迟影响指标更新 | 中 | 低 | 5s timeout + 异步更新,不阻塞业务 |
| 多 Namespace 场景 | 低 | 低 | MVP 仅支持默认 Namespace |

**降级方案:**
如果生产环境不满足 Temporal Server v1.14+ 要求:
- 使用估算方案: 统计最近 1 分钟活跃 Workflow 数量
- 在 `waterflow_agents_connected` 指标添加 `source="estimated"` 标签
- 文档标注精度限制

---

### 关键技术决策

**1. 指标类型选择**
- ✅ 选择: Counter/Gauge/Histogram
- 理由:
  - Counter - 单调递增 (请求数/工作流数)
  - Gauge - 瞬时值 (运行数/连接数)
  - Histogram - 分布统计 (延迟/时长)
- 替代方案:
  - ❌ Summary (复杂度高,资源占用大)

**2. Histogram 桶设计**
- ✅ HTTP 延迟: [5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s, 10s]
  - 理由: 覆盖 API 典型延迟范围
  
- ✅ Workflow 时长: [1s, 5s, 10s, 30s, 1m, 2m, 5m, 10m, 30m, 1h]
  - 理由: 工作流通常秒级到分钟级
  
- ✅ Node 执行: [10ms, 50ms, 100ms, 500ms, 1s, 5s, 10s, 30s, 1m, 5m]
  - 理由: 节点执行快于完整工作流

**3. Agent 指标收集方式**
- ✅ 选择: Temporal WorkflowService.DescribeTaskQueue API
- 理由:
  - ✅ SDK 原生支持 (v1.22.0+),无需自定义扩展
  - ✅ 准确反映 Worker 连接状态 (Pollers 信息)
  - ✅ 包含健康检查数据 (LastAccessTime)
  - ✅ 无需额外心跳机制
  - ✅ 实现复杂度低 (2-4小时)
- 替代方案:
  - ❌ Agent 主动上报 (增加复杂度,与 Temporal 重复)
  - ❌ 估算值 (精度低,仅作降级方案)
- SDK 版本要求: go.temporal.io/sdk >= v1.22.0
- Server 版本要求: Temporal Server >= v1.14.0

**4. Grafana Dashboard 设计原则**
- ✅ 5 个核心面板:
  1. Workflow Rate - 提交/完成/失败速率
  2. Success Rate Gauge - 成功率百分比
  3. API Latency - P50/P95/P99
  4. Agent Status - 连接数/健康数
  5. Node Execution Rate - 各类型节点调用

**5. 指标标签策略**
- ✅ 最小化基数 (Cardinality)
  - ❌ 避免: workflow_id (高基数)
  - ✅ 使用: status, node_type, task_queue
- 理由: 高基数导致内存占用和查询性能问题

### 指标命名规范

遵循 Prometheus 最佳实践:
```
waterflow_<component>_<metric>_<unit>

例如:
- waterflow_workflows_total           # Counter, 无单位
- waterflow_workflows_running         # Gauge, 无单位
- waterflow_workflow_duration_seconds # Histogram, 单位后缀
- waterflow_http_requests_total       # Counter, 无单位
- waterflow_agents_connected          # Gauge, 无单位
```

**规则:**
- 小写 + 下划线分隔
- 时间单位使用 `_seconds`
- Counter 后缀 `_total`
- 不使用 `_count` (与 Histogram 冲突)

### PromQL 查询备忘

**常用查询:**
```promql
# 工作流提交速率 (每秒)
rate(waterflow_workflows_total{status="submitted"}[5m])

# 工作流成功率 (5 分钟窗口)
rate(waterflow_workflows_total{status="completed"}[5m]) 
/ 
on() group_left 
(rate(waterflow_workflows_total{status="completed"}[5m]) + rate(waterflow_workflows_total{status="failed"}[5m]))

# API P99 延迟
histogram_quantile(0.99, rate(waterflow_http_request_duration_seconds_bucket[5m]))

# 各节点类型平均执行时长
rate(waterflow_node_duration_seconds_sum[5m]) / rate(waterflow_node_duration_seconds_count[5m])

# Agent 健康率
waterflow_agents_healthy / waterflow_agents_connected
```

### Docker Compose 集成

**新增服务:**
```yaml
# deployments/docker-compose.yaml

services:
  # ... existing services ...

  prometheus:
    image: prom/prometheus:v2.45.0
    container_name: waterflow-prometheus
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus
    ports:
      - "9090:9090"
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--storage.tsdb.retention.time=15d'

  grafana:
    image: grafana/grafana:10.0.0
    container_name: waterflow-grafana
    volumes:
      - ./grafana/provisioning:/etc/grafana/provisioning
      - ./grafana/dashboards:/var/lib/grafana/dashboards
      - grafana-data:/var/lib/grafana
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_ALLOW_SIGN_UP=false
    depends_on:
      - prometheus

volumes:
  prometheus-data:
  grafana-data:
```

**Prometheus 配置:**
```yaml
# deployments/prometheus/prometheus.yml

global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'waterflow'
    static_configs:
      - targets: ['waterflow-server:8080']
        labels:
          service: 'waterflow-server'
```

## Completion Criteria

**Story 完成标准:**

1. ✅ **所有 AC 完成**
   - AC1-AC6 所有验收标准通过
   - 指标正确导出和更新
   - Temporal DescribeTaskQueue API 集成验证通过

2. ✅ **指标完整性**
   - HTTP 请求指标 (已有)
   - 工作流生命周期指标
   - Agent 健康指标
   - 节点执行指标

3. ✅ **Dashboard 可用**
   - Grafana Dashboard 导入成功
   - 所有面板正常显示
   - 数据实时更新

4. ✅ **文档完整**
   - 指标参考文档
   - 监控部署指南
   - PromQL 查询示例

5. ✅ **测试覆盖**
   - /metrics 端点测试
   - 指标收集准确性测试
   - Dashboard 集成测试

**验收测试场景:**

**场景 1: 访问 /metrics 端点**
```bash
curl http://localhost:8080/metrics
# 预期: Prometheus 格式输出,包含所有指标
```

**场景 2: 工作流指标更新**
```bash
# 1. 提交工作流
waterflow-cli submit examples/hello-world.yaml

# 2. 查询指标
curl -s http://localhost:8080/metrics | grep waterflow_workflows_total
# 预期: submitted +1

# 3. 等待完成,再次查询
curl -s http://localhost:8080/metrics | grep waterflow_workflows_total
# 预期: completed +1, running -1
```

**场景 3: Grafana Dashboard 查看**
```bash
# 1. 启动 Prometheus + Grafana
docker-compose up -d prometheus grafana

# 2. 访问 Grafana
open http://localhost:3000
# 登录: admin/admin

# 3. 导入 Dashboard
# 预期: Waterflow Monitoring Dashboard 正常显示,所有面板有数据
```

**场景 4: Agent 指标验证**
```bash
# 1. 启动 Agent
./bin/agent &

# 2. 等待 30s (AgentMonitor 采样周期)
sleep 30

# 3. 查询 Agent 指标
curl -s http://localhost:8080/metrics | grep waterflow_agents
# 预期: 
# waterflow_agents_connected 1
# waterflow_agents_healthy 1
```

**场景 5: PromQL 查询验证**
```bash
# 在 Prometheus UI (http://localhost:9090) 执行
rate(waterflow_workflows_total{status="completed"}[5m])
# 预期: 返回每秒完成工作流数

# Agent 健康率
waterflow_agents_healthy / waterflow_agents_connected
# 预期: 返回 0-1 之间的值
```

## References

**现有代码参考:**
- [pkg/metrics/metrics.go](../../pkg/metrics/metrics.go) - 现有指标定义
- [pkg/middleware/metrics.go](../../pkg/middleware/metrics.go) - HTTP 指标中间件
- [internal/api/handlers.go](../../internal/api/handlers.go) - /metrics 端点

**相关 Story:**
- Story 1.2 - REST API 服务框架 (基础 API 端点)
- Story 7.3 - 性能基准测试 (性能指标基线)
- Story 7.4 - 压力测试和容错验证 (监控压力场景)

**外部参考:**
- [Prometheus Best Practices](https://prometheus.io/docs/practices/naming/)
- [Prometheus Go Client](https://github.com/prometheus/client_golang)
- [Grafana Dashboard Best Practices](https://grafana.com/docs/grafana/latest/best-practices/)
- [RED Method Monitoring](https://www.weave.works/blog/the-red-method-key-metrics-for-microservices-architecture/)

---

**创建日期:** 2026-01-07  
**创建者:** SM Agent (Bob)  
**Epic:** 7 - 生产级可靠性  
**依赖:** Story 1.2, 1.8, 2.1 完成  
**预估点数:** 8 points (中等复杂度,扩展现有实现)  
**优先级:** High (生产监控的核心需求)
