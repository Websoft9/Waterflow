# Agent 监控集成指南

## Prometheus 集成

### 1. 配置 Prometheus

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'waterflow-agents'
    static_configs:
      - targets:
        - 'agent-1:9090'
        - 'agent-2:9090'
    metrics_path: '/metrics'
    
    # 服务发现 (Kubernetes)
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - waterflow
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        action: keep
        regex: waterflow-agent
```

### 2. Grafana Dashboard

导入预制 Dashboard: [Waterflow Agent Dashboard (ID: 12345)](https://grafana.com/dashboards/12345)

或手动创建:

**Panel 1: Agent 数量**
```promql
count(up{job="waterflow-agents"} == 1)
```

**Panel 2: Activity 执行率**
```promql
rate(temporal_activity_execution_total[5m])
```

**Panel 3: Activity 失败率**
```promql
rate(temporal_activity_execution_failed_total[5m]) /
rate(temporal_activity_execution_total[5m]) * 100
```

**Panel 4: Task Queue 轮询延迟**
```promql
histogram_quantile(0.99, rate(temporal_task_queue_poll_latency_seconds_bucket[5m]))
```

### 3. 关键指标说明

#### Agent 健康指标

| 指标 | 类型 | 说明 |
|------|------|------|
| `up` | Gauge | Agent 是否在线 (1=在线, 0=离线) |
| `waterflow_agent_uptime_seconds` | Counter | Agent 运行时长 (秒) |
| `waterflow_agent_version_info` | Info | Agent 版本信息 |

#### Temporal Worker 指标

| 指标 | 类型 | 说明 |
|------|------|------|
| `temporal_worker_task_queue_poll_requests_total` | Counter | Task Queue 轮询请求总数 |
| `temporal_activity_execution_total` | Counter | Activity 执行总数 |
| `temporal_activity_execution_failed_total` | Counter | Activity 执行失败总数 |
| `temporal_activity_execution_duration_seconds` | Histogram | Activity 执行时长分布 |
| `temporal_task_queue_poll_latency_seconds` | Histogram | Task Queue 轮询延迟 |

#### 系统资源指标

| 指标 | 类型 | 说明 |
|------|------|------|
| `process_cpu_seconds_total` | Counter | CPU 使用时间 |
| `process_resident_memory_bytes` | Gauge | 内存使用量 |
| `go_goroutines` | Gauge | Goroutine 数量 |
| `go_threads` | Gauge | 线程数量 |

### 4. 告警规则示例

```yaml
# alerts.yml
groups:
  - name: waterflow_agent_alerts
    rules:
      # Agent 离线告警
      - alert: AgentDown
        expr: up{job="waterflow-agents"} == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Agent {{ $labels.instance }} is down"
          description: "Agent has been down for more than 2 minutes"
      
      # Activity 高失败率告警
      - alert: HighActivityFailureRate
        expr: |
          rate(temporal_activity_execution_failed_total[5m]) /
          rate(temporal_activity_execution_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High activity failure rate on {{ $labels.instance }}"
          description: "Activity failure rate is {{ $value | humanizePercentage }}"
      
      # 高延迟告警
      - alert: HighTaskQueuePollLatency
        expr: |
          histogram_quantile(0.99,
            rate(temporal_task_queue_poll_latency_seconds_bucket[5m])
          ) > 5
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High task queue poll latency on {{ $labels.instance }}"
          description: "P99 latency is {{ $value }}s"
      
      # 内存使用告警
      - alert: HighMemoryUsage
        expr: process_resident_memory_bytes{job="waterflow-agents"} > 1073741824
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage on {{ $labels.instance }}"
          description: "Memory usage is {{ $value | humanize }}B (>1GB)"
      
      # Goroutine 泄漏告警
      - alert: GoroutineLeak
        expr: go_goroutines{job="waterflow-agents"} > 10000
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Possible goroutine leak on {{ $labels.instance }}"
          description: "Goroutine count is {{ $value }}"
```

## Datadog 集成

### 1. Datadog Agent 配置

```yaml
# datadog-agent.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: datadog-checks
data:
  prometheus.yaml: |
    instances:
      - prometheus_url: http://agent:9090/metrics
        namespace: waterflow
        metrics:
          - temporal_*
          - waterflow_*
          - process_*
          - go_*
```

### 2. 自动发现 (Kubernetes)

```yaml
# waterflow-agent-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: waterflow-agent
spec:
  template:
    metadata:
      annotations:
        ad.datadoghq.com/agent.check_names: '["openmetrics"]'
        ad.datadoghq.com/agent.init_configs: '[{}]'
        ad.datadoghq.com/agent.instances: |
          [
            {
              "prometheus_url": "http://%%host%%:9090/metrics",
              "namespace": "waterflow",
              "metrics": ["temporal_*", "waterflow_*"]
            }
          ]
```

## CloudWatch 集成 (AWS)

### 1. CloudWatch Agent 配置

```json
{
  "metrics": {
    "namespace": "Waterflow/Agent",
    "metrics_collected": {
      "prometheus": {
        "prometheus_config_path": "/opt/aws/amazon-cloudwatch-agent/etc/prometheus.yaml",
        "emf_processor": {
          "metric_declaration": [
            {
              "source_labels": ["job"],
              "label_matcher": "^waterflow-agents$",
              "dimensions": [["instance"]],
              "metric_selectors": [
                "^temporal_activity_execution_total$",
                "^temporal_activity_execution_failed_total$"
              ]
            }
          ]
        }
      }
    }
  }
}
```

## Loki 日志集成

### 1. Promtail 配置

```yaml
# promtail-config.yaml
clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: waterflow-agents
    docker_sd_configs:
      - host: unix:///var/run/docker.sock
        filters:
          - name: label
            values: ["com.docker.compose.service=agent"]
    relabel_configs:
      - source_labels: ['__meta_docker_container_name']
        target_label: 'container'
```

### 2. LogQL 查询示例

```logql
# 查看所有 Agent 日志
{container="waterflow-agent"}

# 查看错误日志
{container="waterflow-agent"} |= "ERROR"

# 统计 Activity 失败数
count_over_time({container="waterflow-agent"} |= "Activity failed" [5m])
```

## ELK Stack 集成

### 1. Filebeat 配置

```yaml
# filebeat.yml
filebeat.inputs:
  - type: container
    paths:
      - '/var/lib/docker/containers/*/*.log'
    processors:
      - add_docker_metadata:
          host: "unix:///var/run/docker.sock"
      - decode_json_fields:
          fields: ["message"]
          target: "json"
          overwrite_keys: true

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "waterflow-agent-%{+yyyy.MM.dd}"

setup.kibana:
  host: "kibana:5601"
```

### 2. Kibana Dashboard

**可视化示例:**
- Activity 执行趋势 (折线图)
- 错误日志分类 (饼图)
- Agent 状态分布 (地图)
- 任务执行时长分布 (直方图)

## 健康检查端点

Agent 提供以下健康检查端点:

### 1. Liveness Probe

```bash
# 检查 Agent 进程是否存活
curl http://agent:9090/health/live
# 响应: {"status":"ok"}
```

**Kubernetes 配置:**
```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 9090
  initialDelaySeconds: 10
  periodSeconds: 10
```

### 2. Readiness Probe

```bash
# 检查 Agent 是否准备好接收任务
curl http://agent:9090/health/ready
# 响应: {"status":"ready","temporal_connected":true}
```

**Kubernetes 配置:**
```yaml
readinessProbe:
  httpGet:
    path: /health/ready
    port: 9090
  initialDelaySeconds: 5
  periodSeconds: 5
```

## 自定义监控

### 1. 自定义指标

Agent 支持通过环境变量添加自定义标签:

```yaml
environment:
  - AGENT_LABEL_DATACENTER=us-west-1
  - AGENT_LABEL_TEAM=platform
  - AGENT_LABEL_ENV=production
```

这些标签会自动附加到所有 Prometheus 指标:

```
temporal_activity_execution_total{datacenter="us-west-1",team="platform",env="production"} 42
```

### 2. 自定义告警通知

**Slack 通知:**
```yaml
# alertmanager.yml
receivers:
  - name: 'slack-notifications'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL'
        channel: '#waterflow-alerts'
        text: |
          {{ range .Alerts }}
          *Alert:* {{ .Labels.alertname }}
          *Severity:* {{ .Labels.severity }}
          *Agent:* {{ .Labels.instance }}
          *Description:* {{ .Annotations.description }}
          {{ end }}
```

**PagerDuty 集成:**
```yaml
receivers:
  - name: 'pagerduty-critical'
    pagerduty_configs:
      - service_key: 'YOUR_PAGERDUTY_SERVICE_KEY'
        description: '{{ .GroupLabels.alertname }}: {{ .CommonAnnotations.summary }}'
```

## 监控最佳实践

### 1. 指标保留策略

```yaml
# prometheus.yml
global:
  external_labels:
    cluster: 'production'
    region: 'us-west-1'

storage:
  tsdb:
    retention.time: 30d  # 保留 30 天
    retention.size: 100GB  # 或 100GB
```

### 2. 关键指标告警阈值建议

| 指标 | 告警阈值 | 持续时间 | 严重级别 |
|------|---------|---------|---------|
| Agent Down | `up == 0` | 2min | Critical |
| Activity 失败率 | `> 10%` | 5min | Warning |
| Activity 失败率 | `> 50%` | 5min | Critical |
| 轮询延迟 P99 | `> 5s` | 10min | Warning |
| 内存使用 | `> 1GB` | 5min | Warning |
| Goroutine 数量 | `> 10000` | 10min | Warning |

### 3. Dashboard 设计建议

**Overview Dashboard (总览):**
- Agent 总数 / 在线数量
- Activity 执行速率 (QPS)
- 平均执行时长
- 失败率趋势

**Detail Dashboard (详细):**
- 按 Task Queue 分组的 Agent 数量
- 各 Agent 的资源使用情况
- Activity 执行时长分布
- 错误日志流

**Troubleshooting Dashboard (排障):**
- 慢查询 (P95 > 10s)
- 错误率异常 Agent
- 资源异常 Agent
- 网络连接状态

## 下一步

- [配置最佳实践](./agent-best-practices.md) - 优化配置以改善监控指标
- [故障排查手册](./agent-troubleshooting.md) - 根据监控告警定位问题
- [快速开始指南](./agent-quickstart.md) - 部署监控堆栈
