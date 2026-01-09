# Waterflow Monitoring Stack

Complete monitoring solution for Waterflow using Prometheus and Grafana.

## Quick Start

```bash
# Start monitoring stack
cd deployments/monitoring
docker-compose up -d

# Access Grafana
open http://localhost:3000
# Default credentials: admin/admin

# Access Prometheus
open http://localhost:9090
```

## Architecture

```
┌──────────────────┐
│ Waterflow Server │ :8080/metrics
└────────┬─────────┘
         │
         │ (scrape every 15s)
         ↓
┌──────────────────┐
│   Prometheus     │ :9090
└────────┬─────────┘
         │
         │ (query)
         ↓
┌──────────────────┐
│    Grafana       │ :3000
└──────────────────┘
```

## Metrics Overview

### Workflow Metrics

- `waterflow_workflows_total{status}` - Total workflows by status
- `waterflow_workflows_running` - Currently running workflows
- `waterflow_workflow_duration_seconds` - Workflow execution duration

### Agent Metrics

- `waterflow_agents_connected` - Connected agents count
- `waterflow_agents_healthy` - Healthy agents count
- `waterflow_agent_tasks_total{agent_id,status}` - Tasks per agent

### Node Metrics

- `waterflow_node_executions_total{node_type,status}` - Node executions
- `waterflow_node_execution_duration_seconds{node_type}` - Node duration

### API Metrics

- `waterflow_http_requests_total{method,path,status}` - HTTP requests
- `waterflow_http_request_duration_seconds{method,path}` - Request latency

### System Metrics (Go Runtime)

- `go_goroutines` - Goroutine count
- `go_memstats_heap_alloc_bytes` - Heap memory
- `process_cpu_seconds_total` - CPU usage
- `process_resident_memory_bytes` - RSS memory

## Useful PromQL Queries

### Workflow Success Rate

```promql
rate(waterflow_workflows_total{status="completed"}[5m]) 
/ 
(rate(waterflow_workflows_total{status="completed"}[5m]) + rate(waterflow_workflows_total{status="failed"}[5m]))
* 100
```

### Workflow P95 Latency

```promql
histogram_quantile(0.95, rate(waterflow_workflow_duration_seconds_bucket[5m]))
```

### API P99 Latency

```promql
histogram_quantile(0.99, rate(waterflow_http_request_duration_seconds_bucket[5m]))
```

### Workflow Throughput

```promql
rate(waterflow_workflows_total[5m])
```

### Agent Health Ratio

```promql
waterflow_agents_healthy / waterflow_agents_connected
```

## Grafana Dashboards

### Waterflow Overview

Located at: `grafana/dashboards/waterflow-overview.json`

**Panels:**
- Workflow Success Rate (Stat)
- Running Workflows (Stat)
- Connected/Healthy Agents (Stats)
- Workflow Submission Rate (Graph)
- API Request Rate (Graph)
- Workflow Duration P50/P95/P99 (Graph)
- API Latency P50/P95/P99 (Graph)
- Node Executions by Type (Graph)
- Agent Task Distribution (Graph)
- Go Goroutines (Graph)
- Memory Usage (Graph)

### Importing Dashboards

**Option 1: Auto-provisioned (Default)**
- Dashboard is auto-loaded from `grafana/dashboards/`
- Updates automatically on restart

**Option 2: Manual Import**
1. Open Grafana → Dashboards → Import
2. Upload `grafana/dashboards/waterflow-overview.json`
3. Select Prometheus datasource

## Configuration

### Prometheus

Edit `prometheus/prometheus.yml` to configure:

```yaml
scrape_configs:
  - job_name: 'waterflow-server'
    static_configs:
      - targets: ['host.docker.internal:8080']
    scrape_interval: 15s
```

### Grafana

**Change admin password:**

```yaml
# docker-compose.yml
environment:
  - GF_SECURITY_ADMIN_PASSWORD=your-secure-password
```

**Enable anonymous access:**

```yaml
environment:
  - GF_AUTH_ANONYMOUS_ENABLED=true
  - GF_AUTH_ANONYMOUS_ORG_NAME=Main Org.
```

## Data Retention

### Prometheus

Default: 15 days

Change in `docker-compose.yml`:

```yaml
command:
  - '--storage.tsdb.retention.time=30d'  # 30 days
```

### Grafana

Grafana stores only metadata; actual data is in Prometheus.

## Alerting (Post-MVP)

Example alert rules (create `prometheus/alerts.yml`):

```yaml
groups:
  - name: waterflow_alerts
    interval: 30s
    rules:
      - alert: HighFailureRate
        expr: rate(waterflow_workflows_total{status="failed"}[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High workflow failure rate"

      - alert: NoConnectedAgents
        expr: waterflow_agents_connected == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "No agents connected"
```

## Troubleshooting

### Metrics not showing in Grafana

1. Check Prometheus targets: http://localhost:9090/targets
2. Verify Waterflow `/metrics` endpoint: `curl http://localhost:8080/metrics`
3. Check Grafana datasource: Configuration → Data Sources

### Prometheus cannot scrape Server

**Using host.docker.internal:**

```yaml
# For Linux, add extra_hosts to docker-compose.yml
extra_hosts:
  - "host.docker.internal:host-gateway"
```

**Using Docker network:**

```yaml
# Run Waterflow Server in same network
networks:
  - waterflow-monitoring
```

### Dashboard panels show "No Data"

1. Check time range (top-right)
2. Verify metrics exist: Explore → Prometheus → `waterflow_workflows_total`
3. Check PromQL syntax in panel query

## Production Deployment

### Persistent Storage

Volumes are defined in `docker-compose.yml`:

- `prometheus-data` - Prometheus TSDB
- `grafana-data` - Grafana dashboards and settings

### Backup

```bash
# Backup Prometheus data
docker run --rm -v waterflow_prometheus-data:/data -v $(pwd):/backup \
  alpine tar czf /backup/prometheus-backup.tar.gz /data

# Backup Grafana data
docker run --rm -v waterflow_grafana-data:/data -v $(pwd):/backup \
  alpine tar czf /backup/grafana-backup.tar.gz /data
```

### High Availability

For production, consider:

- **Prometheus**: Use Thanos or Cortex for long-term storage
- **Grafana**: Run multiple instances behind a load balancer
- **Alertmanager**: Add for notifications (Slack, Email, PagerDuty)

## Resource Requirements

**Minimal:**
- CPU: 0.5 cores
- Memory: 1GB
- Disk: 10GB (for 15 days retention)

**Recommended:**
- CPU: 2 cores
- Memory: 4GB
- Disk: 50GB (for 30+ days retention)

## Security

### Authentication

Default credentials: `admin/admin` (change immediately!)

### Network Security

Recommended firewall rules:

```
# Allow only from trusted networks
Grafana  (3000/tcp): 10.0.0.0/8
Prometheus (9090/tcp): 127.0.0.1 (localhost only)
```

### TLS/HTTPS

Add reverse proxy (Nginx/Traefik) for HTTPS:

```nginx
server {
  listen 443 ssl;
  server_name grafana.example.com;
  
  location / {
    proxy_pass http://localhost:3000;
  }
}
```

## Integration with Deployment

### Docker Compose Deployment

监控栈已集成到主 `docker-compose.yaml` 中，包含：

- **Prometheus**: 自动抓取 Waterflow Server 和 Agent 指标
- **Grafana**: 预配置数据源和仪表板
- **自动发现**: 服务通过 Docker 网络自动连接

启动命令：
```bash
cd /opt/waterflow/deployments
docker-compose up -d
```

所有服务（Waterflow、Temporal、Prometheus、Grafana）将一起启动。

### 独立监控部署

如果需要独立运行监控栈：

```bash
cd /opt/waterflow/deployments/monitoring
docker-compose up -d
```

**注意**: 需要配置 `prometheus/prometheus.yml` 中的目标地址。

### Grafana Dashboard 导入

监控栈已包含预配置的 Waterflow 仪表板：

**自动导入**（推荐）：
- 仪表板位于 `grafana/dashboards/waterflow-overview.json`
- Grafana 启动时自动加载（通过 provisioning）
- 无需手动导入

**手动导入**：
1. 访问 Grafana: http://localhost:3000
2. 导航到 **Dashboards** → **Import**
3. 上传 `grafana/dashboards/waterflow-overview.json`
4. 选择 **Prometheus** 数据源
5. 点击 **Import**

### AlertManager 配置

生产环境建议配置告警通知：

**1. 创建 AlertManager 配置**
```yaml
# prometheus/alertmanager.yml
global:
  resolve_timeout: 5m
  slack_api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'

route:
  group_by: ['alertname', 'severity']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h
  receiver: 'slack-notifications'

receivers:
  - name: 'slack-notifications'
    slack_configs:
      - channel: '#waterflow-alerts'
        title: 'Waterflow Alert'
        text: '{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'
```

**2. 启用 AlertManager**
```yaml
# docker-compose.yml 添加 AlertManager 服务
alertmanager:
  image: prom/alertmanager:latest
  ports:
    - "9093:9093"
  volumes:
    - ./prometheus/alertmanager.yml:/etc/alertmanager/alertmanager.yml
  command:
    - '--config.file=/etc/alertmanager/alertmanager.yml'
```

**3. 配置告警规则**

告警规则文件已在 README 的 "Alerting (Post-MVP)" 部分定义，包括：
- **HighFailureRate**: 工作流失败率过高
- **NoConnectedAgents**: 无 Agent 连接

更多告警规则示例参见 `prometheus/alerts.yml`（需要创建）。

### 监控最佳实践

**1. 设置合理的数据保留期**
```yaml
# docker-compose.yml
command:
  - '--storage.tsdb.retention.time=30d'  # 生产环境建议 30 天
```

**2. 定期备份监控数据**
```bash
# 使用 scripts/backup-configs.sh 备份 Prometheus 配置
# 定期备份 Prometheus 数据卷
docker run --rm -v waterflow_prometheus-data:/data -v /backup:/backup \
  alpine tar czf /backup/prometheus-$(date +%Y%m%d).tar.gz /data
```

**3. 监控资源使用**

监控栈本身的资源消耗：
- **Prometheus**: ~500MB 内存（15 天保留）
- **Grafana**: ~200MB 内存
- **磁盘**: ~1GB/天（取决于指标数量）

**4. 配置告警通知渠道**

支持的通知方式：
- Slack
- Email
- PagerDuty
- Webhook（自定义集成）

参见 Prometheus AlertManager 文档配置通知渠道。

## References

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [PromQL Cheat Sheet](https://promlabs.com/promql-cheat-sheet/)
- [Grafana Dashboard Best Practices](https://grafana.com/docs/grafana/latest/best-practices/)
- [Waterflow Deployment Guide](../../docs/deployment.md)
- [Waterflow Configuration Guide](../../docs/configuration.md)
