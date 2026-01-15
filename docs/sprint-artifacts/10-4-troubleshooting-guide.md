# Story 10.4: 故障排查指南

**Epic**: Epic 10 - 完整文档体系  
**Status**: ready-for-dev  
**Created**: 2025-01-15  
**Dependencies**: 
- ✅ Story 1.2 - REST API 服务框架和监控 (健康检查端点)
- ✅ Story 1.9 - 工作流管理 API (错误响应格式)
- ✅ Story 2.4 - Agent 注册和心跳 (Agent 健康监控)
- ✅ Story 7.1 - 类型化错误处理 (统一错误类型)
- ✅ Story 7.2 - 结构化日志系统 (日志查询和分析)
- ✅ 现有故障排查文档: docs/troubleshooting.md (基础版本,需优化扩充)

---

## User Story

As a **用户 (开发者/运维工程师/系统管理员)**,  
I want **完整的故障排查文档和诊断工具指南**,  
So that **自助解决问题,减少停机时间,无需依赖外部支持**。

---

## Acceptance Criteria

### AC1: 常见问题和解决方案

**Given** 用户遇到常见问题  
**When** 查阅故障排查指南常见问题章节  
**Then** 文档列出至少 10 个常见问题场景:  
**And** 服务无法启动 (端口占用、配置错误、权限问题)  
**And** Temporal 连接失败 (服务未启动、配置错误、网络问题)  
**And** 数据库连接失败 (PostgreSQL 状态、连接配置、数据卷完整性)  
**And** 工作流提交失败 (YAML 语法错误、节点不存在、Task Queue 不匹配)  
**And** 工作流执行卡住 (Agent 离线、Task Queue 堵塞、超时配置不当)  
**And** 工作流失败 (节点执行错误、参数验证失败、权限不足)  
**And** Agent 无法连接 (Temporal 地址错误、网络防火墙、证书问题)  
**And** 性能问题 (资源不足、并发过高、数据库慢查询)  
**And** 日志查询失败 (日志丢失、格式错误、存储满)  
**And** Docker 部署问题 (镜像拉取失败、卷挂载错误、网络配置)  
**And** 每个问题包含完整的诊断步骤、常见原因和解决方案  
**And** 提供可执行的命令示例 (复制即用)

### AC2: Server 日志查看和分析

**Given** 用户需要查看 Server 日志  
**When** 查阅日志查看章节  
**Then** 文档说明如何查看不同环境的日志:  
**And** Docker Compose 环境: `docker-compose logs waterflow` 命令详解  
**And** Systemd 环境: `journalctl -u waterflow-server` 命令详解  
**And** Kubernetes 环境: `kubectl logs` 命令详解  
**And** 说明日志级别过滤 (debug, info, warn, error)  
**And** 说明时间范围过滤 (--since, --until)  
**And** 说明实时日志跟踪 (-f 参数)  
**And** 说明结构化日志字段解析 (JSON 格式)  
**And** 提供常见错误日志模式和诊断方法:  
  - "connection refused" → Temporal 连接问题  
  - "validation_error" → YAML 语法错误  
  - "timeout" → 超时配置或性能问题  
  - "permission denied" → 权限或认证问题  

### AC3: Agent 日志查看和分析

**Given** 用户需要调试 Agent 执行问题  
**When** 查阅 Agent 日志章节  
**Then** 文档说明如何查看 Agent 日志:  
**And** Docker Compose 环境: `docker-compose logs agent-linux-1` 命令详解  
**And** 二进制部署: Agent 日志输出位置和配置  
**And** 说明如何启用 debug 级别日志 (环境变量或配置文件)  
**And** 说明如何过滤特定工作流/Job/Step 的日志  
**And** 提供常见 Agent 错误日志模式:  
  - "Failed to connect to Temporal" → Temporal 连接失败  
  - "Task queue not found" → runs-on 配置错误  
  - "Plugin load failed" → 节点插件加载失败  
  - "Activity execution failed" → 节点执行错误  
**And** 说明如何关联 Agent 日志和工作流执行 (workflowID, runID)

### AC4: 工作流执行失败调试

**Given** 工作流执行失败  
**When** 查阅工作流调试章节  
**Then** 文档提供系统化的调试流程:  
**And** Step 1: 查看工作流状态 (`GET /v1/workflows/{id}`)  
**And** Step 2: 查看工作流日志 (`GET /v1/workflows/{id}/logs`)  
**And** Step 3: 在 Temporal UI 查看执行历史 (Event History)  
**And** Step 4: 定位失败的 Job/Step (错误消息和堆栈跟踪)  
**And** Step 5: 检查节点参数和配置 (with 参数验证)  
**And** Step 6: 检查 Agent 状态和日志 (Agent 是否在线)  
**And** Step 7: 检查权限和网络连通性 (SSH 密钥、防火墙)  
**And** 提供调试技巧:  
  - 使用 `waterflow validate` 本地验证 YAML  
  - 使用 `if` 条件隔离失败步骤  
  - 使用 `continue-on-error: true` 允许失败继续  
  - 添加 debug 输出 Step (echo 变量值)  
  - 减少 Matrix 组合数调试单个实例  

### AC5: Temporal 连接问题排查

**Given** Temporal 连接失败  
**When** 查阅 Temporal 连接排查章节  
**Then** 文档提供完整的诊断流程:  
**And** 检查 Temporal Server 状态 (docker-compose ps, kubectl get pods)  
**And** 检查网络连通性 (nc -zv, ping, telnet)  
**And** 验证配置地址 (temporal.host, temporal.namespace)  
**And** 检查 Temporal UI 可访问性 (http://localhost:8088)  
**And** 查看 Temporal 日志 (docker-compose logs temporal)  
**And** 验证 PostgreSQL 连接 (Temporal 依赖数据库)  
**And** 检查连接重试配置 (max_retries, retry_interval)  
**And** 提供常见问题解决方案:  
  - Docker 环境使用服务名 (temporal:7233 而非 localhost:7233)  
  - 检查 Docker 网络配置 (docker network inspect)  
  - 重建容器和网络 (docker-compose down && docker-compose up)  
  - 检查防火墙规则 (iptables, firewalld)  

### AC6: 健康检查端点使用

**Given** 需要监控服务健康状态  
**When** 查阅健康检查章节  
**Then** 文档完整说明所有健康检查端点:  
**And** `/health` - Liveness 探针 (进程存活检查)  
  - 返回 200: 服务运行中  
  - 返回 503: 服务不可用  
  - 不依赖外部服务 (快速响应)  
**And** `/ready` - Readiness 探针 (依赖服务检查)  
  - 返回 200: 所有依赖就绪 (Temporal 连接正常)  
  - 返回 503: 依赖未就绪 (Temporal 连接失败)  
  - 响应包含详细状态 (JSON 格式)  
**And** `/version` - 版本信息  
  - 返回版本号、Git commit、构建时间  
  - 用于验证部署版本  
**And** `/metrics` - Prometheus 指标  
  - 导出工作流执行指标、API 请求延迟、系统资源  
  - 用于监控和告警  
**And** 提供健康检查使用示例:  
  - curl 命令示例  
  - Kubernetes liveness/readiness probe 配置  
  - Docker Compose healthcheck 配置  
  - 监控系统集成 (Prometheus, Grafana)  

### AC7: 诊断命令速查表

**Given** 用户需要快速诊断问题  
**When** 查阅诊断命令速查表章节  
**Then** 文档提供分类的诊断命令清单:  
**And** 服务状态检查:  
  - `docker-compose ps` - 检查容器状态  
  - `systemctl status waterflow-server` - 检查 Systemd 服务  
  - `curl http://localhost:8080/health` - 健康检查  
  - `curl http://localhost:8080/ready` - 就绪检查  
**And** 日志查看:  
  - `docker-compose logs -f waterflow` - 实时日志  
  - `journalctl -u waterflow-server -n 100` - Systemd 日志  
  - `kubectl logs -f waterflow-server-0` - Kubernetes 日志  
**And** 网络诊断:  
  - `netstat -tuln | grep 8080` - 端口监听检查  
  - `nc -zv localhost 7233` - Temporal 端口连通性  
  - `nslookup temporal` - DNS 解析检查  
  - `traceroute temporal` - 路由检查  
**And** 资源监控:  
  - `docker stats` - 容器资源使用  
  - `top` / `htop` - 系统资源使用  
  - `free -h` - 内存使用  
  - `df -h` - 磁盘使用  
**And** 工作流调试:  
  - `waterflow validate workflow.yaml` - 本地验证  
  - `curl GET /v1/workflows/{id}` - 查询工作流状态  
  - `curl GET /v1/workflows/{id}/logs` - 查询工作流日志  
  - `curl POST /v1/workflows/{id}/cancel` - 取消工作流  
**And** 每个命令包含简短说明和典型输出示例

---

## Tasks / Subtasks

### Task 1: 分析现有故障排查文档 (AC1-AC7)
- [x] 读取 docs/troubleshooting.md 完整内容 (467 行)
- [x] 分析现有章节结构:
  - ✅ 常见问题: 服务无法启动、Temporal 连接失败、数据库连接失败、工作流提交失败、工作流执行卡住、性能问题
  - ✅ 诊断工具: 日志查看、健康检查、资源监控、网络诊断
  - ✅ 获取支持: 自助资源、提交 Issue 指南、紧急支持联系方式
- [x] 分析现有优势:
  - 问题分类清晰 (症状、诊断步骤、常见原因和解决方案)
  - 命令示例完整 (可直接复制执行)
  - 支持多种部署环境 (Docker Compose, Systemd)
- [x] 识别改进点:
  - 缺少工作流执行失败调试流程 (AC4)
  - 缺少 Agent 日志分析章节 (AC3)
  - 缺少错误日志模式匹配和诊断 (AC2-AC3)
  - 缺少诊断命令速查表 (AC7)
  - 缺少 Kubernetes 部署环境支持
  - 缺少工作流调试技巧 (validate, if 条件, debug 输出)
- **输出**: 现有文档完整分析,优化改进清单

### Task 2: 分析错误处理和日志系统实现 (AC2-AC3)
- [x] 分析 CLI 错误类型 (cmd/waterflow-cli/pkg/errors/errors.go)
  - ConfigError: 配置文件加载失败
  - ConnectionError: Server 连接失败
  - AuthError: API 认证失败
  - ValidationError: YAML 验证失败
  - UsageError: 命令使用错误
- [x] 分析 Server 错误类型 (pkg/errors/*.go)
  - ValidationError: YAML 语义验证错误
  - WorkflowError: 工作流执行错误
  - NodeError: 节点执行错误
  - HTTPError: HTTP API 错误 (RFC 7807 Problem Details)
- [x] 分析日志系统 (Story 7.2 结构化日志)
  - 结构化日志格式 (JSON)
  - 日志字段: timestamp, level, workflow_id, component, message
  - 日志级别: debug, info, warn, error
  - 敏感信息脱敏
- [x] 分析常见错误模式 (pkg/errors/classifier.go)
  - "connection refused" → 网络连接问题
  - "timeout" → 超时问题
  - "validation" → YAML 验证错误
  - "not found" → 资源不存在
  - "permission denied" → 权限问题
- **输出**: 错误类型清单,日志模式匹配规则

### Task 3: 优化现有故障排查文档 (AC1,AC5,AC6)
- [ ] 优化常见问题章节 (docs/troubleshooting.md)
  - [ ] 补充工作流失败场景
    - 节点执行错误 (参数验证、权限不足、资源不存在)
    - Matrix 组合数超限 (超过 256)
    - 超时配置不当 (Job/Step 超时)
    - 表达式求值失败 (未定义变量、语法错误)
  - [ ] 补充 Agent 无法连接场景
    - Task Queue 配置错误 (runs-on 不匹配)
    - Temporal 地址配置错误 (Docker 服务名 vs localhost)
    - 网络防火墙阻止连接
    - 插件加载失败 (.so 文件缺失或版本不兼容)
  - [ ] 补充日志查询失败场景
    - 日志存储满 (磁盘空间不足)
    - 日志格式错误 (JSON 解析失败)
    - LogHandler 集成问题 (ELK/Loki 连接失败)
  - [ ] 补充 Docker 部署问题
    - 镜像拉取失败 (网络问题、镜像不存在)
    - 卷挂载错误 (权限问题、路径不存在)
    - 网络配置错误 (端口映射、网络模式)
- [ ] 优化 Temporal 连接问题排查章节 (AC5)
  - [ ] 添加 Docker 服务名 vs localhost 说明
  - [ ] 添加 Kubernetes DNS 解析说明
  - [ ] 添加 Temporal UI 诊断方法
  - [ ] 添加连接重试配置说明 (max_retries, retry_interval)
- [ ] 优化健康检查章节 (AC6)
  - [ ] 补充 `/ready` 端点详细响应格式
  - [ ] 补充 `/metrics` 端点指标说明
  - [ ] 添加 Kubernetes probe 配置示例
  - [ ] 添加监控系统集成示例 (Prometheus, Grafana)
- **输出**: 优化后的常见问题章节 (扩充 10+ 场景)

### Task 4: 创建 Server 日志查看和分析章节 (AC2)
- [ ] 创建日志查看章节 (docs/troubleshooting.md#server-logs)
  - [ ] Docker Compose 环境
    - `docker-compose logs waterflow` - 查看所有日志
    - `docker-compose logs --tail=100 waterflow` - 最近 100 行
    - `docker-compose logs -f waterflow` - 实时跟踪
    - `docker-compose logs --since 2026-01-09T10:00:00 waterflow` - 时间范围
  - [ ] Systemd 环境
    - `journalctl -u waterflow-server -n 100` - 最近 100 行
    - `journalctl -u waterflow-server -f` - 实时跟踪
    - `journalctl -u waterflow-server --since "2026-01-09 10:00:00"` - 时间范围
    - `journalctl -u waterflow-server --no-pager` - 禁用分页
  - [ ] Kubernetes 环境
    - `kubectl logs waterflow-server-0` - 查看日志
    - `kubectl logs -f waterflow-server-0` - 实时跟踪
    - `kubectl logs --since=1h waterflow-server-0` - 最近 1 小时
    - `kubectl logs --tail=100 waterflow-server-0` - 最近 100 行
- [ ] 创建日志分析章节 (docs/troubleshooting.md#log-analysis)
  - [ ] 结构化日志字段说明
    - timestamp: 时间戳
    - level: 日志级别 (debug/info/warn/error)
    - workflow_id: 工作流 ID (关联工作流执行)
    - component: 组件名称 (api/executor/agent)
    - message: 日志消息
    - error: 错误详情 (仅 error 级别)
  - [ ] 日志级别过滤
    - `grep '"level":"error"'` - 只看错误日志
    - `jq 'select(.level=="error")' ` - 使用 jq 过滤
  - [ ] 常见错误日志模式
    - "connection refused" → Temporal 连接失败,检查 Temporal 服务状态
    - "validation_error" → YAML 语法错误,运行 `waterflow validate` 检查
    - "timeout exceeded" → 超时,检查 timeout-minutes 配置和网络延迟
    - "permission denied" → 权限不足,检查文件权限、SSH 密钥、API 认证
    - "node not found" → 节点不存在,运行 `/v1/nodes` 查看可用节点
  - [ ] 日志查询示例
    - 查询特定工作流日志: `grep '"workflow_id":"wf-123"'`
    - 查询 API 错误日志: `grep '"component":"api"' | grep '"level":"error"'`
    - 统计错误数量: `grep '"level":"error"' | wc -l`
- **输出**: Server 日志查看和分析完整章节

### Task 5: 创建 Agent 日志查看和分析章节 (AC3)
- [ ] 创建 Agent 日志查看章节 (docs/troubleshooting.md#agent-logs)
  - [ ] Docker Compose 环境
    - `docker-compose logs agent-linux-1` - 查看 Agent 日志
    - `docker-compose logs -f agent-linux-1` - 实时跟踪
    - `docker-compose logs --tail=100 agent-linux-1` - 最近 100 行
  - [ ] 二进制部署
    - Agent 日志输出到 stdout (JSON 格式)
    - 重定向到文件: `./agent > agent.log 2>&1`
    - Systemd 部署: `journalctl -u waterflow-agent -f`
  - [ ] 启用 debug 日志
    - 环境变量: `WATERFLOW_LOG_LEVEL=debug`
    - 配置文件: `log.level: debug`
- [ ] 创建 Agent 日志分析章节 (docs/troubleshooting.md#agent-log-analysis)
  - [ ] 常见 Agent 错误模式
    - "Failed to connect to Temporal" → Temporal 地址配置错误
    - "Task queue not found" → runs-on 配置与 Agent task_queues 不匹配
    - "Plugin load failed" → 节点插件加载失败,检查 .so 文件完整性
    - "Activity execution failed" → 节点执行错误,查看错误详情和参数
    - "Heartbeat timeout" → Agent 与 Temporal 失联,检查网络连接
  - [ ] 关联工作流执行
    - workflowID: 工作流唯一标识
    - runID: Temporal 运行 ID
    - jobID: Job 标识
    - stepID: Step 标识
  - [ ] 过滤特定工作流日志
    - `grep '"workflow_id":"wf-123"' agent.log`
    - `grep '"job_id":"deploy"' agent.log`
- **输出**: Agent 日志查看和分析完整章节

### Task 6: 创建工作流执行失败调试章节 (AC4)
- [ ] 创建工作流调试流程章节 (docs/troubleshooting.md#workflow-debugging)
  - [ ] Step 1: 查看工作流状态
    - `curl http://localhost:8080/v1/workflows/{id}`
    - 检查 status 字段 (running/completed/failed/cancelled)
    - 检查当前 Job/Step 进度
  - [ ] Step 2: 查看工作流日志
    - `curl http://localhost:8080/v1/workflows/{id}/logs`
    - 过滤错误日志: `?level=error`
    - 过滤特定 Job: `?job=deploy`
  - [ ] Step 3: 在 Temporal UI 查看
    - 访问 http://localhost:8088
    - 搜索 workflowID
    - 查看 Event History (完整执行历史)
    - 查看 Activity 失败详情 (错误消息、堆栈跟踪)
  - [ ] Step 4: 定位失败 Job/Step
    - 从日志中找到 job_id 和 step_id
    - 检查错误消息和建议
    - 检查节点类型和参数
  - [ ] Step 5: 检查节点参数
    - 验证 with 参数类型和值
    - 验证必需参数完整性
    - 验证表达式求值 (`${{ }}` 语法)
  - [ ] Step 6: 检查 Agent 状态
    - 查询 Agent 列表: `GET /v1/agents`
    - 检查 Agent 是否在线 (healthy)
    - 检查 Task Queue 匹配 (runs-on vs task_queues)
  - [ ] Step 7: 检查权限和网络
    - SSH 密钥是否配置 (ssh-copy-id)
    - 防火墙是否阻止连接 (iptables, firewalld)
    - 目标服务器是否可达 (ping, nc)
- [ ] 创建调试技巧章节 (docs/troubleshooting.md#debugging-tips)
  - [ ] 本地验证 YAML: `waterflow validate workflow.yaml`
  - [ ] 隔离失败步骤: 使用 `if` 条件禁用其他 Step
  - [ ] 允许失败继续: `continue-on-error: true`
  - [ ] 添加 debug 输出:
    ```yaml
    - name: Debug variables
      uses: exec/shell@v1
      with:
        command: echo
        args: ["vars.env=${{ vars.env }}, matrix.server=${{ matrix.server }}"]
    ```
  - [ ] 减少 Matrix 组合: 调试单个实例
  - [ ] 增加超时时间: 临时提高 timeout-minutes
  - [ ] 查看 Temporal UI: 完整 Event History
- **输出**: 工作流调试流程和技巧完整章节

### Task 7: 创建诊断命令速查表 (AC7)
- [ ] 创建诊断命令速查表章节 (docs/troubleshooting.md#diagnostic-commands)
  - [ ] 服务状态检查
    - `docker-compose ps` - 检查容器状态
    - `systemctl status waterflow-server` - Systemd 服务状态
    - `kubectl get pods -l app=waterflow` - Kubernetes Pods 状态
    - `curl http://localhost:8080/health` - Liveness 探针
    - `curl http://localhost:8080/ready` - Readiness 探针
    - `curl http://localhost:8080/version` - 版本信息
  - [ ] 日志查看
    - `docker-compose logs -f waterflow` - 实时日志 (Docker)
    - `journalctl -u waterflow-server -n 100` - 最近 100 行 (Systemd)
    - `kubectl logs -f waterflow-server-0` - 实时日志 (Kubernetes)
    - `grep '"level":"error"' <log-file>` - 过滤错误日志
  - [ ] 网络诊断
    - `netstat -tuln | grep 8080` - 检查端口监听
    - `nc -zv localhost 7233` - Temporal 端口连通性
    - `nslookup temporal` - DNS 解析 (Docker/Kubernetes)
    - `traceroute temporal` - 路由检查
    - `curl -v http://localhost:8080/health` - HTTP 连通性
  - [ ] 资源监控
    - `docker stats` - 容器资源使用
    - `top` 或 `htop` - 系统资源使用
    - `free -h` - 内存使用
    - `df -h` - 磁盘使用
    - `iostat -x 1` - 磁盘 I/O
  - [ ] 工作流调试
    - `waterflow validate workflow.yaml` - 本地 YAML 验证
    - `curl GET http://localhost:8080/v1/workflows/{id}` - 查询状态
    - `curl GET http://localhost:8080/v1/workflows/{id}/logs` - 查询日志
    - `curl POST http://localhost:8080/v1/workflows/{id}/cancel` - 取消工作流
    - `curl GET http://localhost:8080/v1/nodes` - 查看可用节点
  - [ ] Temporal 诊断
    - `docker-compose logs temporal` - Temporal 日志
    - `curl http://localhost:8088/api/v1/namespaces` - Temporal API
    - 打开 Temporal UI: http://localhost:8088
  - [ ] 数据库诊断
    - `docker-compose ps postgresql` - PostgreSQL 状态
    - `docker-compose exec postgresql psql -U temporal -c "SELECT 1"` - 连接测试
- [ ] 添加命令输出示例
  - 每个命令添加典型输出示例
  - 说明如何解读输出结果
  - 标注关键字段和异常值
- **输出**: 诊断命令速查表完整章节

### Task 8: 补充 Kubernetes 部署环境支持
- [ ] 在所有章节补充 Kubernetes 命令
  - [ ] 日志查看: `kubectl logs`
  - [ ] 健康检查: `kubectl get pods`, `kubectl describe pod`
  - [ ] 资源监控: `kubectl top pod`
  - [ ] 网络诊断: `kubectl exec`, `kubectl port-forward`
- [ ] 添加 Kubernetes 特定问题
  - [ ] Pod CrashLoopBackOff
  - [ ] ImagePullBackOff
  - [ ] PersistentVolumeClaim pending
  - [ ] Service 无法访问
- **输出**: Kubernetes 部署环境完整支持

### Task 9: 文档验证和优化
- [ ] 验证所有命令可执行性
  - [ ] 在 Docker Compose 环境测试所有命令
  - [ ] 在 Systemd 环境测试 (如适用)
  - [ ] 验证输出示例准确性
- [ ] 验证 AC 覆盖
  - [ ] AC1: 至少 10 个常见问题场景 ✓
  - [ ] AC2: Server 日志查看和分析 ✓
  - [ ] AC3: Agent 日志查看和分析 ✓
  - [ ] AC4: 工作流执行失败调试 ✓
  - [ ] AC5: Temporal 连接问题排查 ✓
  - [ ] AC6: 健康检查端点使用 ✓
  - [ ] AC7: 诊断命令速查表 ✓
- [ ] 文档格式优化
  - [ ] 代码块语法高亮 (bash, yaml, json)
  - [ ] 表格格式优化 (诊断命令速查表)
  - [ ] 目录生成 (TOC)
  - [ ] 锚点链接 (章节跳转)
- [ ] 交叉引用优化
  - [ ] 添加相关文档链接 (deployment.md, configuration.md, quick-start.md)
  - [ ] 添加 Story 交叉引用 (Story 1.2, 1.9, 2.4, 7.1, 7.2)
  - [ ] 添加代码实现引用 (pkg/errors/*.go, pkg/config/config.go)
- **输出**: 验证通过的生产级故障排查文档

### Task 10: 更新 sprint-status.yaml 状态
- [ ] 将 Story 10-4-troubleshooting-guide 状态从 backlog 更新为 ready-for-dev
- **输出**: sprint-status.yaml 更新完成

---

## Dev Notes

### 现有分析

**已完成的基础工作 (Epics 1-9):**
- ✅ **健康检查端点实现** (Story 1.2): /health, /ready, /version, /metrics
  - /health: Liveness 探针 (进程存活检查)
  - /ready: Readiness 探针 (Temporal 连接检查)
  - /version: 版本信息 (版本号、Git commit、构建时间)
  - /metrics: Prometheus 指标 (工作流执行、API 延迟、系统资源)

- ✅ **错误处理实现** (Story 7.1): 类型化错误系统
  - CLI 错误: ConfigError, ConnectionError, AuthError, ValidationError, UsageError
  - Server 错误: ValidationError, WorkflowError, NodeError, HTTPError (RFC 7807)
  - 错误分类器: pkg/errors/classifier.go (模式匹配)

- ✅ **结构化日志系统** (Story 7.2): JSON 格式日志
  - 日志字段: timestamp, level, workflow_id, component, message, error
  - 日志级别: debug, info, warn, error
  - 敏感信息脱敏 (密码、Token)

- ✅ **Agent 健康监控** (Story 2.4): Temporal Worker 心跳机制
  - Agent 自动发送心跳 (默认 30 秒)
  - 连续 3 次失败标记 unhealthy
  - Temporal 自动故障转移

- ✅ **工作流管理 API** (Story 1.9): 完整生命周期管理
  - POST /v1/workflows: 提交工作流
  - GET /v1/workflows/{id}: 查询工作流状态
  - GET /v1/workflows/{id}/logs: 查询工作流日志
  - POST /v1/workflows/{id}/cancel: 取消工作流
  - 统一错误格式 (RFC 7807)

**现有故障排查文档分析 (docs/troubleshooting.md):**
- ✅ **优势**:
  - 问题分类清晰 (症状、诊断步骤、常见原因和解决方案)
  - 命令示例完整 (可直接复制执行)
  - 支持多种部署环境 (Docker Compose, Systemd)
  - 涵盖主要问题: 服务启动、Temporal 连接、数据库连接、工作流提交、性能问题
  - 提供诊断工具清单: 日志查看、健康检查、资源监控、网络诊断

- ❌ **改进点** (Story 10.4 需要扩充):
  - 缺少工作流执行失败调试流程 (AC4) - **关键缺口**
  - 缺少 Agent 日志分析章节 (AC3)
  - 缺少错误日志模式匹配和诊断 (AC2-AC3)
  - 缺少诊断命令速查表 (AC7)
  - 缺少 Kubernetes 部署环境支持
  - 缺少工作流调试技巧 (validate, if 条件, debug 输出, Matrix 调试)
  - 常见问题场景不足 10 个 (AC1 要求至少 10 个)

### 优化建议

**文档扩充策略:**
1. **保留现有优势** - 不破坏已有的良好结构和示例
2. **补充缺失章节** - 工作流调试、Agent 日志、诊断速查表、Kubernetes 支持
3. **扩充常见问题** - 从 6 个扩充到 10+ 个场景
4. **优化日志分析** - 添加错误模式匹配、日志查询技巧、关联分析
5. **增强调试技巧** - Temporal UI 使用、表达式调试、Matrix 调试、本地验证

**新增章节结构:**
1. **Server 日志查看和分析** (AC2)
   - 多环境日志查看 (Docker, Systemd, Kubernetes)
   - 结构化日志字段说明
   - 日志级别过滤和时间范围过滤
   - 常见错误日志模式匹配
   - 日志查询示例和技巧

2. **Agent 日志查看和分析** (AC3)
   - Agent 日志查看 (Docker, 二进制, Systemd)
   - 启用 debug 日志
   - 常见 Agent 错误模式
   - 关联工作流执行 (workflowID, runID, jobID, stepID)
   - 过滤特定工作流/Job/Step 日志

3. **工作流执行失败调试** (AC4) - **核心新增章节**
   - 系统化调试流程 (7 步)
   - Temporal UI 使用指南
   - 节点参数验证
   - Agent 状态检查
   - 权限和网络检查
   - 调试技巧 (本地验证、隔离步骤、debug 输出、Matrix 调试)

4. **诊断命令速查表** (AC7)
   - 分类命令清单 (服务状态、日志、网络、资源、工作流、Temporal、数据库)
   - 每个命令简短说明
   - 典型输出示例
   - 可直接复制使用

**常见问题扩充清单 (AC1 要求至少 10 个):**
1. ✅ 服务无法启动 (已有)
2. ✅ Temporal 连接失败 (已有)
3. ✅ 数据库连接失败 (已有)
4. ✅ 工作流提交失败 (已有)
5. ✅ 工作流执行卡住 (已有)
6. ✅ 性能问题 (已有)
7. **新增**: 工作流执行失败 (节点错误、参数验证、权限)
8. **新增**: Agent 无法连接 (Temporal 地址、网络、Task Queue)
9. **新增**: 日志查询失败 (存储满、格式错误、LogHandler 问题)
10. **新增**: Docker 部署问题 (镜像拉取、卷挂载、网络)

### 架构约束和设计决策

**错误处理架构 (Story 7.1):**
- 类型化错误系统 (ErrInvalidYAML, ErrNodeNotFound, ErrWorkflowTimeout)
- 错误包含上下文信息 (workflow_id, step_name, node_type)
- REST API 返回 RFC 7807 Problem Details 格式
- 错误分类器模式匹配 (connection_refused, timeout, validation_error)

**日志系统架构 (Story 7.2):**
- 结构化日志 (JSON 格式)
- 日志字段标准化 (timestamp, level, workflow_id, component, message)
- 敏感信息自动脱敏
- 日志级别配置 (debug, info, warn, error)
- LogHandler 接口支持外部日志系统集成 (ELK, Loki, CloudWatch)

**健康检查架构 (Story 1.2):**
- /health: Liveness 探针 (不依赖外部服务,快速响应)
- /ready: Readiness 探针 (检查 Temporal 连接状态)
- /version: 版本信息 (Git commit, 构建时间)
- /metrics: Prometheus 指标导出

**Temporal 集成架构 (Story 1.8):**
- Event Sourcing: 工作流状态存储在 Temporal Event History
- Server 无状态: 崩溃后从 Event History 恢复
- Temporal UI: 完整执行历史查看 (http://localhost:8088)
- 连接重试机制: max_retries, retry_interval 配置

### 时间估算

**Story 10.4 估算**: **12-16 小时 (1.5-2 工作日)**

**任务分解:**
- Task 1: 分析现有故障排查文档 - **1 小时** (已完成)
- Task 2: 分析错误处理和日志系统 - **1 小时** (已完成)
- Task 3: 优化现有故障排查文档 (AC1,AC5,AC6) - **2 小时**
- Task 4: 创建 Server 日志查看和分析章节 (AC2) - **2 小时**
- Task 5: 创建 Agent 日志查看和分析章节 (AC3) - **2 小时**
- Task 6: 创建工作流执行失败调试章节 (AC4) - **3 小时** (核心章节)
- Task 7: 创建诊断命令速查表 (AC7) - **1.5 小时**
- Task 8: 补充 Kubernetes 部署环境支持 - **1.5 小时**
- Task 9: 文档验证和优化 - **2 小时**
- Task 10: 更新 sprint-status.yaml - **0.5 小时**

**复杂度分析:**
- ✅ **低-中等复杂度**: 基础文档已存在 (docs/troubleshooting.md),主要是扩充和优化
- ✅ **中等工作量**: 需要新增 4 个完整章节 (日志分析、工作流调试、诊断速查表)
- ✅ **高质量要求**: 故障排查文档直接影响用户自助解决问题能力,必须准确完整

---

## References

### 源文档 (PRD, Architecture, Epics)
1. [PRD - 产品需求文档](../../docs/prd.md) - NFR7: 文档完善性,故障排查指南
2. [Architecture - 架构文档](../../docs/architecture.md) - 健康检查、日志系统、错误处理
3. [Epics - Epic 10 Story 10.4](../../docs/epics.md#story-104-故障排查指南) - Story 定义和 AC 规范

### 实现参考 (Stories 1-9)
4. [Story 1.2 - REST API 服务框架和监控](./1-2-rest-api-framework-and-monitoring.md) - 健康检查端点
5. [Story 1.9 - 工作流管理 API](./1-9-workflow-management-api.md) - 错误响应格式,工作流查询
6. [Story 2.4 - Agent 注册和心跳](./2-4-agent-registration-and-heartbeat.md) - Agent 健康监控
7. [Story 7.1 - 类型化错误处理](./7-1-typed-error-handling.md) - 统一错误类型,RFC 7807
8. [Story 7.2 - 结构化日志系统](./7-2-structured-logging.md) - JSON 日志,日志字段,敏感信息脱敏

### 现有文档 (基础版本)
9. [docs/troubleshooting.md](../../docs/troubleshooting.md) - 现有故障排查文档 (467 行,需优化扩充)
10. [docs/deployment.md](../../docs/deployment.md) - 部署文档 (可能包含部署问题排查)
11. [docs/configuration.md](../../docs/configuration.md) - 配置文档 (配置错误排查)
12. [docs/quick-start.md](../../docs/quick-start.md) - 快速开始文档 (常见问题章节)

### 代码实现 (错误处理和日志)
13. [cmd/waterflow-cli/pkg/errors/errors.go](../../cmd/waterflow-cli/pkg/errors/errors.go) - CLI 错误类型
14. [pkg/errors/errors.go](../../pkg/errors/errors.go) - Server 错误类型
15. [pkg/errors/classifier.go](../../pkg/errors/classifier.go) - 错误分类器 (模式匹配)
16. [pkg/errors/validation.go](../../pkg/errors/validation.go) - YAML 验证错误
17. [pkg/errors/workflow.go](../../pkg/errors/workflow.go) - 工作流执行错误
18. [pkg/errors/node.go](../../pkg/errors/node.go) - 节点执行错误
19. [pkg/errors/http.go](../../pkg/errors/http.go) - HTTP API 错误 (RFC 7807)
20. [pkg/config/config.go](../../pkg/config/config.go) - 配置验证和错误消息

### 测试和示例 (错误场景)
21. [test/stress/temporal_reconnect_test.sh](../../test/stress/temporal_reconnect_test.sh) - Temporal 连接重试测试
22. [cmd/waterflow-cli/README.md](../../cmd/waterflow-cli/README.md) - CLI 错误场景示例
23. [docs/templates/*.md](../../docs/templates/) - 工作流模板故障排查章节

### 外部参考 (最佳实践)
24. [RFC 7807 - Problem Details for HTTP APIs](https://datatracker.ietf.org/doc/html/rfc7807) - 错误响应格式标准
25. [Temporal Documentation - Troubleshooting](https://docs.temporal.io/dev-guide/troubleshooting) - Temporal 故障排查参考
26. [Docker Compose Troubleshooting](https://docs.docker.com/compose/faq/) - Docker Compose 常见问题
27. [Kubernetes Troubleshooting](https://kubernetes.io/docs/tasks/debug/) - Kubernetes 调试指南

---

## File List

**修改文件:**
- `/docs/troubleshooting.md` - 现有故障排查文档 (优化扩充,本 Story 主要目标)

**新增文件:**
- 无 (所有内容更新到 docs/troubleshooting.md)

**状态更新文件:**
- `/docs/sprint-artifacts/sprint-status.yaml` - Story 10-4 状态更新 (backlog → ready-for-dev)

**参考文件:**
- `/cmd/waterflow-cli/pkg/errors/errors.go` - CLI 错误类型
- `/pkg/errors/*.go` - Server 错误类型和分类器
- `/pkg/config/config.go` - 配置验证
- `/test/stress/temporal_reconnect_test.sh` - Temporal 连接测试
- `/docs/quick-start.md` - 快速开始文档 (常见问题参考)

---

## Change Log

**2025-01-15 - Story 创建 (100% 完整上下文)**
- ✅ 分析现有故障排查文档 (docs/troubleshooting.md, 467 行)
- ✅ 分析错误处理实现 (CLI + Server 错误类型)
- ✅ 分析日志系统实现 (结构化日志,字段标准化)
- ✅ 识别文档优势和改进点 (6 个现有问题 → 10+ 个扩充)
- ✅ 创建 7 个验收标准 (常见问题、Server 日志、Agent 日志、工作流调试、Temporal 连接、健康检查、诊断速查表)
- ✅ 创建 10 个详细任务 (82 个子任务)
- ✅ 添加 27 个参考文档链接
- ✅ 提供架构约束和时间估算
- ✅ Story 状态: ready-for-dev

---
