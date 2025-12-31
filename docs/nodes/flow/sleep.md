# flow/sleep@v1

**分类**: flow (流程控制)  
**版本**: v1  
**状态**: stable

## 概述

flow/sleep 节点用于在工作流步骤间添加延迟等待。支持秒、分钟、小时为单位的延迟，延迟期间工作流状态持久化，支持通过 context 取消。

这是最简单的节点，无外部依赖，仅使用 Go 标准库。

## 使用场景

- **等待服务启动**: 启动 Docker 容器后等待服务就绪
- **API 限流控制**: API 调用之间添加延迟避免触发限流
- **重试间隔**: 失败后等待一段时间再重试
- **健康检查间隔**: 定期检查服务状态的间隔
- **部署验证**: 部署完成后等待系统稳定

## 参数

| 参数名 | 类型 | 必需 | 默认值 | 描述 |
|--------|------|------|--------|------|
| duration | string | ✅ | - | 延迟时长 (如 "30s", "5m", "1h") |

### 参数详细说明

**duration** (string, required)
- 延迟时长，支持多种格式
- 格式: 数字+单位 (s=秒, m=分钟, h=小时)
- 范围: 1秒 - 24小时
- 示例:
  - `"30s"` → 30 秒
  - `"5m"` → 5 分钟
  - `"2h"` → 2 小时
  - `"1h30m"` → 1 小时 30 分钟
  - `"60"` → 60 秒（纯数字默认为秒）

## 返回值

| 字段名 | 类型 | 描述 |
|--------|------|------|
| duration_seconds | int | 实际延迟秒数 |
| completed | bool | 是否正常完成 (true) 或被取消 (false) |
| elapsed_ms | int | 实际经过的时间 (毫秒) |

## 使用示例

### 示例 1: 等待服务启动

```yaml
steps:
  - name: Start application container
    uses: docker/exec@v1
    with:
      command: run
      args: ["-d", "--name", "myapp", "myapp:latest"]
  
  - name: Wait for application to start
    uses: flow/sleep@v1
    with:
      duration: 30s
  
  - name: Check application health
    uses: http/request@v1
    with:
      url: http://localhost:8080/health
```

### 示例 2: API 限流延迟

```yaml
steps:
  - name: Call API 1
    uses: http/request@v1
    with:
      url: https://api.example.com/resource/1
  
  - name: Wait to avoid rate limit
    uses: flow/sleep@v1
    with:
      duration: 2s
  
  - name: Call API 2
    uses: http/request@v1
    with:
      url: https://api.example.com/resource/2
```

### 示例 3: 重试间隔

```yaml
steps:
  - name: Try to connect
    uses: exec/shell@v1
    with:
      command: curl
      args: ["-f", "http://service:8080"]
    id: connect
    continue-on-error: true
  
  - name: Wait before retry
    if: steps.connect.outputs.exit_code != 0
    uses: flow/sleep@v1
    with:
      duration: 10s
  
  - name: Retry connection
    if: steps.connect.outputs.exit_code != 0
    uses: exec/shell@v1
    with:
      command: curl
      args: ["-f", "http://service:8080"]
```

### 示例 4: 定期健康检查

```yaml
steps:
  - name: Deploy application
    uses: docker/compose@v1
    with:
      action: up
  
  - name: Wait for initial startup
    uses: flow/sleep@v1
    with:
      duration: 1m
  
  - name: First health check
    uses: http/request@v1
    with:
      url: http://localhost:8080/health
  
  - name: Wait before second check
    uses: flow/sleep@v1
    with:
      duration: 30s
  
  - name: Second health check
    uses: http/request@v1
    with:
      url: http://localhost:8080/health
```

### 示例 5: 长时间延迟

```yaml
steps:
  - name: Start batch job
    uses: exec/shell@v1
    with:
      command: ./start-batch-job.sh
  
  - name: Wait for job completion (estimated 2 hours)
    uses: flow/sleep@v1
    with:
      duration: 2h
  
  - name: Collect results
    uses: exec/shell@v1
    with:
      command: ./collect-results.sh
```

## 常见错误

### 错误 1: 无效的时间格式

**现象:**
```
Error: invalid duration format: "abc"
```

**解决方法:**
使用正确的时间格式: `"30s"`, `"5m"`, `"1h"`

### 错误 2: 超过最大延迟时间

**现象:**
```
Error: duration exceeds maximum allowed: 24h
```

**解决方法:**
将延迟时间设置为 24 小时以内。

### 错误 3: 延迟时间过短

**现象:**
```
Error: duration is less than minimum: 1s
```

**解决方法:**
最小延迟时间为 1 秒。

### 错误 4: 延迟被取消

**现象:**
```
completed: false
elapsed_ms: 500
```

**原因:**
工作流被用户取消或 context 超时。

**解决方法:**
这是正常行为，延迟会立即返回。检查工作流取消原因。

## 最佳实践

### 1. 使用合理的延迟时间

```yaml
# 服务启动等待
duration: 30s - 1m

# API 限流
duration: 1s - 5s

# 数据库操作后等待
duration: 5s - 10s

# 长时间批处理
duration: 5m - 2h
```

### 2. 结合健康检查而非固定延迟

```yaml
# ❌ 不推荐: 固定等待可能太短或太长
- uses: flow/sleep@v1
  with:
    duration: 60s

# ✅ 推荐: 使用重试健康检查
- uses: http/request@v1
  with:
    url: http://localhost:8080/health
  retry:
    max_attempts: 10
    initial_interval: 5s
```

### 3. 在重试策略中使用

```yaml
retry:
  max_attempts: 5
  initial_interval: 2s  # 相当于每次重试前 sleep 2s
  backoff_coefficient: 2.0  # 指数退避
```

### 4. 避免过长的固定延迟

```yaml
# ❌ 不推荐: 长时间阻塞
- uses: flow/sleep@v1
  with:
    duration: 30m

# ✅ 推荐: 使用定时触发或轮询
```

### 5. 组合使用条件判断

```yaml
- name: Check if service needs restart
  uses: exec/shell@v1
  with:
    command: systemctl
    args: ["is-active", "myservice"]
  id: check
  continue-on-error: true

- name: Restart service
  if: steps.check.outputs.exit_code != 0
  uses: exec/shell@v1
  with:
    command: systemctl
    args: ["restart", "myservice"]

- name: Wait for service to start
  if: steps.check.outputs.exit_code != 0
  uses: flow/sleep@v1
  with:
    duration: 15s
```

## 时间格式参考

| 格式 | 说明 | 秒数 |
|------|------|------|
| `"1s"` | 1 秒 | 1 |
| `"30s"` | 30 秒 | 30 |
| `"1m"` | 1 分钟 | 60 |
| `"5m"` | 5 分钟 | 300 |
| `"1h"` | 1 小时 | 3600 |
| `"2h"` | 2 小时 | 7200 |
| `"1h30m"` | 1 小时 30 分钟 | 5400 |
| `"90"` | 90 秒（纯数字） | 90 |

## 性能特点

- **精度**: Go time.Sleep 精度 ±1-5ms，长延迟可忽略误差
- **资源消耗**: 极低，仅占用 goroutine 栈空间
- **可取消**: 支持 context 取消，立即返回
- **持久化**: 延迟期间工作流状态持久化到 Temporal

## 相关节点

- [http/request](../http/request.md) - 可与重试策略结合实现健康检查
- [exec/shell](../exec/shell.md) - 可在延迟前后执行检查命令

## 参考文档

- [Story 3.4: 延迟等待节点](../../sprint-artifacts/3-4-sleep-delay-node.md)

---

**文档版本**: v1  
**最后更新**: 2025-12-31  
**状态**: 稳定
