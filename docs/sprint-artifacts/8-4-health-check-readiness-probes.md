# Story 8.4: Health Check and Readiness Probes (健康检查和就绪探针)

**Created:** 2025-12-19  
**Status:** ready-for-dev  
**Epic:** Epic 8 - Deployment and Operations  
**Assignee:** TBD  
**Story Points:** 5

---

## Context

健康检查和就绪探针是云原生应用的重要组成部分，用于让容器编排平台（如 Docker、Kubernetes）了解服务状态：

- **Liveness Probe (`/health`)**: 检测进程是否存活，失败时容器会被重启
- **Readiness Probe (`/ready`)**: 检测服务是否准备好接收流量，失败时从负载均衡器移除

**Current State:**
- ✅ `/health` 端点已实现 - 基础健康检查（返回 "healthy" + 时间戳）
- ✅ `/ready` 端点已实现 - 包含 Temporal 连接检查
- ✅ Dockerfile.server 已配置 HEALTHCHECK（使用 `/health`，间隔 10s）
- ❌ Dockerfile.agent 缺少 HEALTHCHECK 配置
- ❌ `/ready` 端点缺少数据库连接检查
- ❌ 缺少详细的健康检查文档
- ❌ 未实现健康检查指标采集

**Implementation Files:**
- `/internal/api/handlers.go` - Health() 和 Ready() 基础实现
- `/internal/api/router.go` - 路由配置，包含 Temporal 健康检查逻辑
- `/build/Dockerfile.server` - Server HEALTHCHECK 已配置
- `/build/Dockerfile.agent` - 缺少 HEALTHCHECK

**Dependencies:**
- Story 8-3 (Configuration Management) - 健康检查超时配置
- Story 8-2 (Docker Compose) - 容器健康检查集成

---

## User Story

**As a** 系统管理员 / DevOps 工程师,  
**I want** 健康检查和就绪探针端点,  
**So that** 监控系统可以自动检测服务状态并触发重启或流量切换。

---

## Acceptance Criteria

### AC1: `/health` 端点检查进程存活状态

**Given** Server 或 Agent 正在运行  
**When** 监控系统请求 `GET /health`  
**Then** 返回 HTTP 200 状态码  
**And** 响应体包含 `{"status":"healthy","timestamp":"2025-12-19T10:30:00Z"}`  
**And** Content-Type 为 `application/json`  
**And** 响应时间 < 100ms（快速检查）

**Implementation Notes:**
- ✅ Server 已实现 - `/internal/api/handlers.go:54-64`
- ❌ Agent 需要实现类似端点（或复用 metrics 端口）
- Health 端点只检查进程存活，不依赖外部服务

**Test Cases:**
```bash
# Server health check
curl http://localhost:8080/health
# Expected: {"status":"healthy","timestamp":"..."}

# Docker health check
docker inspect waterflow-server --format='{{.State.Health.Status}}'
# Expected: healthy
```

---

### AC2: `/ready` 端点检查服务就绪状态

**Given** Server 启动并连接依赖服务  
**When** 负载均衡器请求 `GET /ready`  
**Then** 检查 Temporal 连接状态  
**And** 检查数据库连接状态（如果配置）  
**And** 所有检查通过时返回 HTTP 200  
**And** 任一检查失败时返回 HTTP 503 Service Unavailable  
**And** 响应体包含详细检查结果

**Example Response (All Ready):**
```json
{
  "status": "ready",
  "timestamp": "2025-12-19T10:30:00Z",
  "checks": {
    "temporal": "ok",
    "database": "ok"
  }
}
```

**Example Response (Not Ready):**
```json
{
  "status": "not_ready",
  "timestamp": "2025-12-19T10:30:05Z",
  "checks": {
    "temporal": "connection refused: dial tcp [::1]:7233: connect: connection refused",
    "database": "ok"
  }
}
```

**Implementation Notes:**
- ✅ Temporal 检查已实现 - `/internal/api/router.go:30-62`
- ❌ 需添加数据库连接检查（使用 `db.Ping(ctx)`）
- ❌ 需支持可配置的检查超时（默认 2 秒）

---

### AC3: 服务不可用时返回 503 状态码

**Given** Server 已启动但依赖服务不可用  
**When** 请求 `/ready` 端点  
**Then** 返回 HTTP 503 Service Unavailable  
**And** 响应体说明具体失败原因（如 "temporal: connection timeout"）  
**And** 容器编排器将停止向该实例转发流量

**Failure Scenarios:**
- Temporal 服务未启动 → `temporal: connection refused`
- Temporal 网络不可达 → `temporal: i/o timeout`
- 数据库连接池耗尽 → `database: too many connections`
- 依赖服务响应超时 → `temporal: context deadline exceeded`

**Implementation Notes:**
- ✅ HTTP 503 逻辑已实现 - `/internal/api/router.go:53-54`
- 需确保所有检查都设置超时，避免阻塞

---

### AC4: Docker HEALTHCHECK 配置使用 `/health` 端点

**Given** Dockerfile 中配置 HEALTHCHECK  
**When** 容器启动  
**Then** Docker 每隔 10 秒执行健康检查  
**And** 使用 `curl -f http://localhost:8080/health` 命令  
**And** 失败 10 次后标记容器为 unhealthy  
**And** 启动后等待 10 秒才开始首次检查（`--start-period`）

**Current Configuration (Server):**
```dockerfile
HEALTHCHECK --interval=10s --timeout=5s --start-period=10s --retries=10 \
    CMD curl -f http://localhost:8080/health || exit 1
```

**Implementation Notes:**
- ✅ Server Dockerfile 已配置 - `/build/Dockerfile.server:60-61`
- ❌ Agent Dockerfile 缺少 HEALTHCHECK 配置
- 需确保 `curl` 工具已安装在容器镜像中

**Agent HEALTHCHECK Requirements:**
```dockerfile
# Agent should check metrics port (9090) or implement /health endpoint
HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=5 \
    CMD curl -f http://localhost:9090/metrics || exit 1
```

---

### AC5: Readiness 检查通过 `/ready` 端点验证

**Given** Kubernetes Deployment 配置 readinessProbe  
**When** Pod 启动  
**Then** Kubelet 调用 `/ready` 端点检查就绪状态  
**And** 返回 200 时 Pod 标记为 Ready，加入 Service endpoints  
**And** 返回 503 时 Pod 从 Service endpoints 移除  
**And** 支持配置 `initialDelaySeconds`, `periodSeconds`, `failureThreshold`

**Kubernetes Readiness Probe Example:**
```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 8080
    scheme: HTTP
  initialDelaySeconds: 10  # 等待 Temporal 连接建立
  periodSeconds: 5         # 每 5 秒检查一次
  timeoutSeconds: 2        # 单次检查超时
  successThreshold: 1      # 连续成功 1 次标记为 Ready
  failureThreshold: 3      # 连续失败 3 次标记为 Not Ready
```

**Implementation Notes:**
- `/ready` 端点需支持快速响应（< 2 秒）
- 应避免在健康检查中执行复杂逻辑（如查询大量数据）
- 数据库 Ping 应使用连接池已有连接，而非创建新连接

---

### AC6: 健康检查支持配置化超时和重试

**Given** `config.yaml` 配置健康检查参数  
**When** Server 启动  
**Then** 使用配置的超时时间执行依赖检查  
**And** 支持配置各依赖服务的超时时间  
**And** 超时配置应合理（默认 2s，最大 5s）

**Configuration Example:**
```yaml
server:
  health:
    timeout: 2s           # 整体健康检查超时
    temporal_timeout: 2s  # Temporal 连接检查超时
    db_timeout: 1s        # 数据库 Ping 超时
```

**Implementation Notes:**
- 需在 `pkg/config/config.go` 添加 `HealthConfig` 结构
- Temporal 健康检查使用 `context.WithTimeout`
- 数据库 Ping 使用 `db.PingContext(ctx)`

---

## Implementation Tasks

### Task 1: 增强 Server `/ready` 端点支持数据库检查

**Files:**
- `/internal/api/router.go` - 添加数据库健康检查逻辑
- `/pkg/config/config.go` - 添加健康检查配置

**Changes:**
1. 在 `NewRouter()` 接受 `db *sql.DB` 参数
2. 在 `/ready` handler 中添加数据库 Ping 检查：
   ```go
   if db != nil {
       ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
       defer cancel()
       if err := db.PingContext(ctx); err != nil {
           checks["database"] = err.Error()
           allReady = false
       } else {
           checks["database"] = "ok"
       }
   }
   ```
3. 添加健康检查配置结构：
   ```go
   type HealthConfig struct {
       Timeout         time.Duration `mapstructure:"timeout"`
       TemporalTimeout time.Duration `mapstructure:"temporal_timeout"`
       DatabaseTimeout time.Duration `mapstructure:"db_timeout"`
   }
   ```
4. 使用配置的超时时间替换硬编码的 2s

**Acceptance:**
- `/ready` 返回包含 `database` 状态的响应
- 数据库不可用时返回 HTTP 503
- 超时时间可通过配置文件调整

---

### Task 2: 为 Agent 添加健康检查端点

**Files:**
- `/build/Dockerfile.agent` - 添加 HEALTHCHECK 指令
- Agent HTTP server 实现（如果存在 metrics 端口）

**Changes:**
1. 在 Dockerfile.agent 添加 HEALTHCHECK（第 70 行后）：
   ```dockerfile
   # Health check using metrics endpoint
   HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=5 \
       CMD wget --no-verbose --tries=1 --spider http://localhost:9090/metrics || exit 1
   ```
   
2. 确保 Agent 暴露 metrics 端点（或添加专用 `/health` 端点）

3. 如需添加 `/health` 端点，参考 Server 实现创建简单 HTTP handler

**Acceptance:**
- `docker inspect waterflow-agent` 显示 Health 状态
- Agent 容器故障时自动重启

---

### Task 3: 添加健康检查测试用例

**Files:**
- `/internal/api/router_ready_test.go` - 扩展现有测试
- `/test/integration/health_check_test.go` - 新增集成测试

**Test Coverage:**

**Unit Tests (router_ready_test.go):**
```go
func TestReadyEndpoint_DatabaseCheck(t *testing.T) {
    // Test 1: Database healthy
    // Test 2: Database connection failed
    // Test 3: Database timeout
}

func TestReadyEndpoint_MultipleChecks(t *testing.T) {
    // Test all checks passing
    // Test partial failure (temporal ok, db fail)
}
```

**Integration Tests (health_check_test.go):**
```go
func TestHealthCheckIntegration(t *testing.T) {
    // Test 1: Start server, verify /health returns 200
    // Test 2: Stop Temporal, verify /ready returns 503
    // Test 3: Restart Temporal, verify /ready returns 200
}
```

**Acceptance:**
- Unit test coverage > 90% for health check handlers
- Integration tests validate end-to-end Docker health check behavior

---

### Task 4: 添加健康检查配置文档

**Files:**
- `/docs/configuration.md` - 添加健康检查配置章节
- `/examples/configs/config.example.yaml` - 添加配置示例
- `/deployments/docker-compose.yaml` - 添加 healthcheck 配置示例

**Documentation Updates:**

**configuration.md:**
```markdown
## Health Check Configuration

Server health check endpoints are available for monitoring:

- **`/health`**: Liveness probe - checks if process is alive
- **`/ready`**: Readiness probe - checks dependencies (Temporal, database)

### Configuration

```yaml
server:
  health:
    timeout: 2s           # Overall health check timeout
    temporal_timeout: 2s  # Temporal connection check timeout
    db_timeout: 1s        # Database ping timeout
```

### Docker Health Checks

Server Dockerfile includes health check:
```dockerfile
HEALTHCHECK --interval=10s --timeout=5s --start-period=10s --retries=10 \
    CMD curl -f http://localhost:8080/health || exit 1
```

### Kubernetes Readiness Probes

Example Deployment configuration:
```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 5
  timeoutSeconds: 2
```
```

**Acceptance:**
- Documentation explains all health check endpoints
- Example configurations provided for Docker and Kubernetes
- Troubleshooting guide for common health check failures

---

### Task 5: 添加健康检查指标采集

**Files:**
- `/pkg/metrics/metrics.go` - 添加健康检查相关 metrics
- `/internal/api/router.go` - 记录健康检查结果

**Metrics to Add:**
```go
var (
    healthCheckDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "waterflow_health_check_duration_seconds",
            Help:    "Duration of health check requests",
            Buckets: prometheus.DefBuckets,
        },
        []string{"endpoint", "status"},
    )
    
    dependencyHealthStatus = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "waterflow_dependency_health_status",
            Help: "Health status of dependencies (1=healthy, 0=unhealthy)",
        },
        []string{"dependency"},
    )
)
```

**Usage Example:**
```go
// In /ready handler
start := time.Now()
defer func() {
    duration := time.Since(start).Seconds()
    healthCheckDuration.WithLabelValues("ready", statusLabel).Observe(duration)
}()

// Record dependency status
if temporalHealthy {
    dependencyHealthStatus.WithLabelValues("temporal").Set(1)
} else {
    dependencyHealthStatus.WithLabelValues("temporal").Set(0)
}
```

**Acceptance:**
- Prometheus metrics track health check latency
- Grafana dashboard shows dependency health status over time
- Alerts can be configured for prolonged unhealthy state

---

## Development Notes

### Architecture Insights

1. **Health vs Ready Distinction:**
   - **`/health`** 检查本地状态（进程存活），应 always 快速响应
   - **`/ready`** 检查依赖服务，可能较慢或失败，用于流量管理

2. **Temporal Health Check Implementation:**
   - 当前使用 `temporalClient.CheckHealth(ctx)` 方法
   - 实现位于 `/pkg/temporal/client.go`（需确认是否存在此方法）
   - 如未实现，可使用简单的 `DescribeNamespace()` 调用验证连接

3. **Database Connection Check:**
   - 使用 `db.PingContext(ctx)` 验证连接池状态
   - 避免创建新连接（影响性能）
   - 设置合理超时（1-2 秒）避免阻塞健康检查

4. **Agent Health Check Strategy:**
   - Agent 可能没有 HTTP API 服务
   - 选项 1: 复用 metrics 端口（9090）检查 `/metrics` 端点
   - 选项 2: 添加轻量级 HTTP 服务专用于健康检查
   - 选项 3: 使用进程信号检查（不推荐，不符合容器最佳实践）

### Edge Cases & Error Handling

1. **启动阶段 (`--start-period`):**
   - Server 需等待 Temporal 连接建立（约 5-10 秒）
   - 使用 `--start-period=10s` 避免启动期间健康检查失败
   - Kubernetes 使用 `initialDelaySeconds` 实现类似功能

2. **超时处理:**
   - 所有外部调用必须设置超时（使用 `context.WithTimeout`）
   - 避免健康检查阻塞导致容器假死
   - 超时时间应小于 Docker HEALTHCHECK `--timeout` 参数

3. **并发健康检查:**
   - 多个监控系统可能同时调用 `/ready`
   - 需确保依赖检查（如 DB Ping）线程安全
   - 考虑添加缓存机制（如 5 秒内返回缓存结果）避免频繁检查

4. **服务降级场景:**
   - 某些依赖可选（如 metrics, tracing）不应影响 Readiness
   - 只有核心依赖（Temporal, Database）失败才返回 503
   - 考虑添加 `critical` 标记区分必需和可选依赖

### Performance Considerations

1. **响应时间要求:**
   - `/health` 应在 < 100ms 内响应（只检查本地状态）
   - `/ready` 应在 < 2s 内响应（包含网络调用）
   - 超时配置应合理（默认 2s，可配置 1-5s）

2. **资源消耗:**
   - 健康检查频率高（10s 一次）需避免重量级操作
   - 数据库 Ping 应使用现有连接，避免创建新连接
   - 考虑添加结果缓存减少检查频率

3. **Metrics 影响:**
   - 每次健康检查都记录 metrics 会产生大量时序数据
   - 考虑只记录失败的检查或聚合统计

### Testing Strategy

1. **Unit Tests:**
   - Mock Temporal client 和数据库连接
   - 测试各种失败场景（连接拒绝、超时、部分失败）
   - 验证响应格式和 HTTP 状态码

2. **Integration Tests:**
   - 使用 Docker Compose 启动完整环境
   - 测试健康检查在真实依赖下的行为
   - 验证容器重启逻辑（stop Temporal，观察 health 状态变化）

3. **Load Tests:**
   - 模拟高频健康检查（每秒 100 次）验证性能
   - 确保健康检查不会成为性能瓶颈

### Related Stories & Dependencies

- **Story 8-2 (Docker Compose):** 需要集成健康检查配置到 docker-compose.yaml
- **Story 8-3 (Configuration Management):** 健康检查超时参数通过配置管理
- **Story 1-9 (REST API):** 健康检查端点是 API 的一部分
- **Epic 9 (Security):** 考虑健康检查端点是否需要认证（通常不需要）

### Security Considerations

1. **信息泄露:**
   - `/ready` 端点暴露依赖服务信息（Temporal, Database）
   - 不应返回敏感信息（如连接字符串、IP 地址）
   - 只返回通用错误消息（如 "connection refused" 而非完整错误堆栈）

2. **未认证访问:**
   - 健康检查端点通常不需要认证（容器平台需要访问）
   - 但应限制返回信息的详细程度
   - 考虑添加内部 `/healthz/verbose` 端点提供详细信息（需认证）

3. **DDoS 防护:**
   - 健康检查端点可能被滥用（高频请求）
   - 考虑添加 rate limiting（如每秒最多 10 次）
   - 或在 nginx/Ingress 层面限制

### Documentation Updates Needed

1. **README.md:**
   - 添加健康检查端点说明
   - 提供 Docker 和 Kubernetes 配置示例

2. **docs/deployment.md:**
   - 详细说明健康检查配置
   - 故障排查指南（如 "为什么容器一直重启"）

3. **docs/architecture.md:**
   - 更新架构图，标注健康检查流程
   - 说明 Liveness vs Readiness 策略

4. **API 文档:**
   - 添加 `/health` 和 `/ready` 端点到 API spec
   - 提供 OpenAPI/Swagger 定义

---

## Files to Create/Modify

### New Files
1. `/test/integration/health_check_test.go` - 集成测试

### Modified Files
1. `/internal/api/router.go` - 添加数据库健康检查
2. `/internal/api/router_ready_test.go` - 扩展测试用例
3. `/pkg/config/config.go` - 添加 HealthConfig 结构
4. `/pkg/metrics/metrics.go` - 添加健康检查 metrics
5. `/build/Dockerfile.agent` - 添加 HEALTHCHECK 指令
6. `/docs/configuration.md` - 健康检查配置文档
7. `/docs/deployment.md` - 健康检查部署指南
8. `/examples/configs/config.example.yaml` - 添加配置示例
9. `/deployments/docker-compose.yaml` - 添加健康检查配置

### Files to Review
1. `/pkg/temporal/client.go` - 确认 `CheckHealth()` 方法实现
2. `/internal/server/server.go` - 确认如何传递 DB 连接到 Router
3. `/cmd/agent/main.go` - 确认 Agent 是否有 HTTP 服务

---

## Acceptance Criteria Summary

- [x] AC1: `/health` 端点返回进程存活状态（Server 已实现）
- [ ] AC2: `/ready` 端点检查 Temporal + 数据库连接
- [ ] AC3: 依赖不可用时返回 HTTP 503
- [x] AC4: Server Docker HEALTHCHECK 已配置
- [ ] AC4: Agent Docker HEALTHCHECK 需添加
- [ ] AC5: Kubernetes Readiness Probe 支持
- [ ] AC6: 健康检查超时可配置

**Definition of Done:**
- [ ] 所有 AC 通过验收测试
- [ ] Unit test coverage ≥ 90%
- [ ] Integration tests 通过
- [ ] 健康检查文档完整
- [ ] Docker 和 K8s 配置示例添加
- [ ] Metrics 集成完成
- [ ] Code review 通过

---

## Estimated Effort

- **Story Points:** 5
- **Estimated Hours:** 12-16 hours
  - Task 1 (Database check): 3 hours
  - Task 2 (Agent health): 2 hours
  - Task 3 (Testing): 4 hours
  - Task 4 (Documentation): 2 hours
  - Task 5 (Metrics): 3 hours

---

## Notes

**Technical Debt:**
- Current `/ready` implementation in `router.go` is inline - consider refactoring to dedicated handler
- Agent health check strategy needs architectural decision (metrics port vs dedicated endpoint)

**Future Enhancements:**
- Health check result caching (避免频繁检查)
- Verbose health endpoint with detailed diagnostics (需认证)
- Startup probe support (Kubernetes 1.16+)
- Custom health check plugins (让用户扩展检查逻辑)

**References:**
- [Kubernetes Liveness, Readiness, and Startup Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
- [Docker HEALTHCHECK](https://docs.docker.com/engine/reference/builder/#healthcheck)
- [Health Check Response Format (RFC 7807)](https://tools.ietf.org/html/rfc7807)
