# Agent 故障排查手册

## 常见问题索引

| 问题 | 可能原因 | 快速检查 |
|------|---------|---------|
| Agent 无法启动 | 配置错误、依赖缺失 | [→ 1.1](#11-agent-无法启动) |
| 无法连接 Temporal | 网络问题、URL 错误 | [→ 1.2](#12-无法连接-temporal) |
| Agent 未接收任务 | Queue 配置不匹配 | [→ 2.1](#21-agent-未接收任务) |
| Activity 执行失败 | Plugin 缺失、超时 | [→ 2.2](#22-activity-执行失败) |
| 内存占用过高 | 并发配置过高、内存泄漏 | [→ 3.1](#31-内存占用过高) |
| CPU 占用过高 | CPU 密集型任务 | [→ 3.2](#32-cpu-占用过高) |
| 心跳超时 | 网络延迟、服务器异常 | [→ 4.1](#41-心跳超时) |
| 日志丢失 | 日志配置错误 | [→ 5.1](#51-日志丢失) |

---

## 1. 启动问题

### 1.1 Agent 无法启动

**症状:**
```bash
$ ./agent --config config.yaml
Error: failed to load config: yaml: unmarshal errors:
  line 2: field temporal not found in type config.AgentConfig
```

**原因:** 配置文件格式错误

**解决:**
```bash
# 1. 验证配置文件语法
./agent --config config.yaml --validate

# 2. 对比示例配置
diff config.yaml config.agent.example.yaml

# 3. 使用 YAML Linter
yamllint config.yaml
```

---

### 1.2 无法连接 Temporal

**症状:**
```
[ERROR] Failed to create Temporal client: connection refused
[ERROR] Worker startup failed
```

**诊断步骤:**
```bash
# 1. 检查 Temporal Server 是否运行
telnet temporal.example.com 7233
# 或
nc -zv temporal.example.com 7233

# 2. 检查 Agent 配置
cat config.yaml | grep "host:"
# 输出: host: "temporal.example.com:7233"

# 3. 检查 DNS 解析
nslookup temporal.example.com

# 4. 检查防火墙
sudo iptables -L -n | grep 7233

# 5. 检查 Docker 网络 (如果使用容器)
docker exec agent ping temporal
```

**常见原因:**
- ❌ `host: "http://localhost:7233"` (不应包含 http://)
- ✅ `host: "localhost:7233"` (正确格式)

---

## 2. 任务执行问题

### 2.1 Agent 未接收任务

**症状:**
```bash
# 提交工作流后,Agent 日志无任何输出
$ docker logs agent
[INFO] Worker started successfully
[INFO] Polling task queues: [linux-amd64]
# ... 没有后续日志
```

**诊断步骤:**
```bash
# 1. 检查 Agent 的 Task Queue 配置
docker logs agent | grep "Polling task queues"
# 输出: Polling task queues: [linux-amd64]

# 2. 检查工作流的 runs-on 配置
cat workflow.yaml | grep runs-on
# 输出: runs-on: linux-arm64  ← ❌ 不匹配!

# 3. 查询 Task Queue 状态
curl http://localhost:8080/v1/task-queues | jq '.task_queues[] | select(.name=="linux-amd64")'
# 输出: {"name":"linux-amd64","worker_count":0,...}  ← Worker 数量为 0!

# 4. 检查 Agent 是否已注册
curl http://localhost:8080/v1/agents?task_queue=linux-amd64
# 输出: {"agents":[],"total":0}  ← 未注册!
```

**解决:**
```bash
# 修正工作流配置
runs-on: linux-amd64  # 匹配 Agent 的 TASK_QUEUES

# 或者修改 Agent 配置
docker run -d -e TASK_QUEUES=linux-arm64,linux-amd64 waterflow/agent:latest
```

---

### 2.2 Activity 执行失败

**症状:**
```
[ERROR] Activity failed: activity type 'custom-plugin' not registered
```

**原因:** Agent 未加载自定义 Plugin

**解决:**
```bash
# 1. 检查 Plugin 目录
docker exec agent ls -la /app/plugins
# 输出: total 0  ← 目录为空!

# 2. 挂载 Plugin
docker run -d \
  -v /path/to/plugins:/app/plugins:ro \
  waterflow/agent:latest

# 3. 验证 Plugin 加载
docker logs agent | grep "Plugin loaded"
# 输出: [INFO] Plugin loaded successfully: my-plugin.so
```

---

## 3. 资源问题

### 3.1 内存占用过高

**症状:**
```bash
$ docker stats agent
CONTAINER    CPU %    MEM USAGE / LIMIT    MEM %
agent        50%      1.8GB / 2GB          90%  ← 内存接近限制!
```

**诊断:**
```bash
# 1. 检查并发配置
cat config.yaml | grep max_concurrent
# 输出: max_concurrent_activities: 50  ← 太高!

# 2. 检查是否有内存泄漏
# 观察内存使用趋势
docker stats agent --no-stream --format "table {{.MemUsage}}"

# 3. 查看 Go 运行时统计
curl http://agent:9090/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

**解决:**
```yaml
# 降低并发数
temporal:
  worker:
    max_concurrent_activities: 10  # 从 50 降低到 10

# 或增加内存限制
docker run -d --memory=4g waterflow/agent:latest
```

---

### 3.2 CPU 占用过高

**症状:**
```bash
$ top
PID   USER    %CPU  COMMAND
1234  waterflo 300%  /app/agent  ← CPU 使用率 300% (3 个核心)
```

**原因:** 可能是 CPU 密集型任务

**解决:**
```yaml
# 1. 降低并发数
temporal:
  worker:
    max_concurrent_activities: 4  # = CPU 核心数

# 2. 限制 CPU 使用 (Docker)
docker run -d --cpus="2.0" waterflow/agent:latest
```

---

## 4. 网络问题

### 4.1 心跳超时

**症状:**
```
[WARN] Failed to update heartbeat: context deadline exceeded
[WARN] Agent may be marked as unhealthy
```

**诊断:**
```bash
# 1. 检查到 Temporal 的网络延迟
ping -c 5 temporal.example.com

# 2. 测试 Temporal 连接
telnet temporal.example.com 7233

# 3. 检查防火墙规则
sudo iptables -L -n -v | grep 8080
```

**解决:**
```yaml
# 增加心跳超时时间
advanced:
  heartbeat_interval: "60s"  # 从 30s 增加到 60s

# 或配置 HTTP Proxy
environment:
  - HTTP_PROXY=http://proxy.example.com:8080
```

---

## 5. 日志和监控

### 5.1 日志丢失

**症状:**
```bash
$ docker logs agent
# 输出为空
```

**诊断:**
```bash
# 1. 检查日志配置
docker exec agent cat /app/config/config.yaml | grep -A5 logger

# 2. 检查日志文件
docker exec agent ls -lh /var/log/waterflow/
# 输出: total 0  ← 没有日志文件!

# 3. 检查日志目录权限
docker exec agent stat /var/log/waterflow/
# 输出: Access: (0755/drwxr-xr-x)  Uid: (    0/    root)  ← 权限问题!
```

**解决:**
```bash
# 修正目录权限
docker exec agent chown -R waterflow:waterflow /var/log/waterflow

# 或使用 stdout 输出
logger:
  output: "stdout"  # 不写入文件
```

---

## 6. 高级诊断

### 6.1 启用 Debug 日志

```yaml
logger:
  level: "debug"  # 启用详细日志
```

**重启 Agent:**
```bash
docker restart agent
docker logs -f agent  # 查看详细日志
```

### 6.2 使用 pprof 分析

```bash
# 1. 访问 pprof 端点
curl http://agent:9090/debug/pprof/

# 2. 生成 CPU profile
curl http://agent:9090/debug/pprof/profile?seconds=30 > cpu.prof

# 3. 分析
go tool pprof cpu.prof
> top10  # 查看 CPU 占用最高的 10 个函数
```

### 6.3 Temporal Web UI

```bash
# 访问 Temporal Web UI
open http://localhost:8080  # 默认端口

# 查看 Worker 状态:
# Workflows → Task Queues → <your-queue> → Workers
```

---

## 7. 紧急恢复流程

### 7.1 Agent 完全不响应

```bash
# 1. 强制重启
docker restart -t 0 agent  # 立即重启,不等待优雅关闭

# 2. 如果仍无响应,删除并重建
docker rm -f agent
docker run -d --name agent -e TASK_QUEUES=linux-amd64 waterflow/agent:latest

# 3. 检查工作流任务是否恢复
curl http://localhost:8080/v1/task-queues
```

### 7.2 任务积压

**症状:** 大量任务等待执行

**解决:**
```bash
# 快速扩容 Agent
docker run -d --name agent-2 -e TASK_QUEUES=linux-amd64 waterflow/agent:latest
docker run -d --name agent-3 -e TASK_QUEUES=linux-amd64 waterflow/agent:latest

# 监控任务处理速度
watch 'curl -s http://localhost:8080/v1/task-queues | jq ".task_queues[] | select(.name==\"linux-amd64\")"'
```

---

## 8. 获取帮助

如果以上方法无法解决问题,请提供以下信息:

```bash
# 收集诊断信息
cat > diagnosis.txt <<EOF
Agent Version: $(docker exec agent /app/agent --version)
Config:
$(docker exec agent cat /app/config/config.yaml)

Recent Logs:
$(docker logs --tail=100 agent)

System Info:
$(docker exec agent uname -a)
$(docker exec agent cat /etc/os-release)

Network:
$(docker exec agent ip addr)
$(docker exec agent netstat -tuln)
EOF

# 提交 Issue: https://github.com/Websoft9/Waterflow/issues
```
