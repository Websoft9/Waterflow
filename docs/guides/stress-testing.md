# Waterflow 压力测试指南

本指南介绍如何运行 Waterflow 的压力测试套件,验证系统在高负载和故障场景下的稳定性。

## 目录

- [概述](#概述)
- [前置条件](#前置条件)
- [快速开始](#快速开始)
- [测试场景](#测试场景)
- [性能分析与 pprof](#性能分析与-pprof)
- [故障排查](#故障排查)
- [CI/CD 集成](#cicd-集成)

## 概述

压力测试套件验证以下关键能力:

- **并发处理**: 1000+ 并发工作流,成功率 >99%
- **故障恢复**: Server 崩溃后自动恢复,Event Sourcing 零状态丢失
- **Agent 重连**: Agent 断开后自动重连,任务继续执行
- **资源泄漏**: 内存/Goroutine 泄漏检测,长时间运行稳定性
- **超时重试**: 各种超时/重试场景正确处理
- **混合故障**: Server + Agent 同时故障的恢复能力

## 前置条件

### 系统要求

- **操作系统**: Linux (推荐 Ubuntu 20.04+)
- **Go**: 1.21+
- **工具**: jq, bc, curl

### 环境准备

```bash
# 安装依赖
sudo apt-get update
sudo apt-get install -y jq bc curl

# 构建二进制文件
make build build-agent

# 启动 Temporal (Docker)
docker-compose -f deployments/docker-compose.yaml up -d temporal
```

## 快速开始

### 运行完整测试套件

```bash
# 运行所有压力测试
make stress-test

# 或直接运行脚本
chmod +x test/stress/*.sh
./test/stress/run_all_tests.sh
```

### 运行快速测试

```bash
# 减少并发数,快速验证
make stress-test-quick

# 或设置环境变量
CONCURRENT_WORKFLOWS=100 ./test/stress/concurrent_workflows_test.sh
```

### 运行单个测试

```bash
# 并发工作流测试
./test/stress/concurrent_workflows_test.sh

# Server 崩溃恢复
./test/stress/server_crash_recovery_test.sh

# 资源泄漏检测
./test/stress/resource_leak_test.sh

# 超时重试场景
./test/stress/timeout_retry_test.sh

# Agent 重连测试
./test/stress/agent_reconnect_test.sh

# 混合故障场景
./test/stress/mixed_fault_test.sh
```

## 测试场景

### 1. 并发工作流压力测试

**目标**: 验证系统在 1000+ 并发下的稳定性

**指标**:
- 成功率 >99%
- CPU <80%
- Memory <2GB
- 平均响应时间 <10s

**运行**:
```bash
CONCURRENT_WORKFLOWS=1000 ./test/stress/concurrent_workflows_test.sh
```

**结果分析**:
```bash
# 查看结果
cat test/stress/results/concurrent-*/summary.txt

# CPU/内存趋势
cat test/stress/results/concurrent-*/cpu.log
cat test/stress/results/concurrent-*/memory.log
```

### 2. Server 崩溃恢复测试

**目标**: 验证 Event Sourcing 架构的零状态丢失

**场景**:
1. 启动 Server
2. 提交 10 个长时运行工作流
3. kill -9 Server (模拟崩溃)
4. 重启 Server
5. 验证所有工作流恢复

**预期**:
- 恢复时间 <10s
- 所有工作流继续执行
- Event History 完整

**运行**:
```bash
./test/stress/server_crash_recovery_test.sh
```

### 3. 资源泄漏检测

**目标**: 验证长时间运行无内存/Goroutine 泄漏

**指标**:
- 内存增长 <20%/hour
- Goroutine 增长 <20%/hour

**运行**:
```bash
# 1 小时负载测试
./test/stress/resource_leak_test.sh

# Go 内存分析
go test -v ./test/stress -run TestMemoryProfile
```

### 4. 混合故障场景

**目标**: 验证 Server + Agent 同时故障的恢复能力

**场景**:
1. 同时 kill Server 和 Agent
2. 重启服务
3. 验证工作流恢复和 Event History 完整性

**指标**:
- 恢复时间 <15s
- 工作流继续执行
- Event History 无丢失

**运行**:
```bash
./test/stress/mixed_fault_test.sh
```

## 性能分析与 pprof

### pprof 端点

Server 和 Agent 默认启用 pprof 端点:

- **Server**: `http://localhost:6060/debug/pprof/`
- **Agent**: `http://localhost:6061/debug/pprof/`

### 内存分析

```bash
# 实时内存分析
go tool pprof http://localhost:6060/debug/pprof/heap

# 保存 heap profile
curl http://localhost:6060/debug/pprof/heap > heap.prof
go tool pprof -http=:8080 heap.prof

# 查看内存分配
go tool pprof -alloc_space http://localhost:6060/debug/pprof/heap
```

### Goroutine 分析

```bash
# Goroutine 泄漏检测
curl http://localhost:6060/debug/pprof/goroutine > goroutine.prof
go tool pprof goroutine.prof

# 实时查看 Goroutine
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

### CPU Profiling

```bash
# 30 秒 CPU profile
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof -http=:8080 cpu.prof
```

### 内存泄漏诊断流程

1. **采集基线**:
   ```bash
   curl http://localhost:6060/debug/pprof/heap > heap-before.prof
   ```

2. **运行负载测试**:
   ```bash
   ./test/stress/resource_leak_test.sh
   ```

3. **采集对比**:
   ```bash
   curl http://localhost:6060/debug/pprof/heap > heap-after.prof
   go tool pprof -base heap-before.prof heap-after.prof
   ```

4. **分析增长**:
   ```
   (pprof) top
   (pprof) list <function_name>
   ```

## 故障排查

### 常见问题

#### 1. 测试超时

**症状**: 工作流长时间未完成

**排查**:
```bash
# 检查 Temporal 连接
curl http://localhost:7233/api/v1/health

# 检查 Agent 日志
tail -f test/stress/results/*/agent.log

# 检查 Server 日志
tail -f test/stress/results/*/server.log
```

#### 2. 成功率低于 99%

**症状**: `Success Rate: 95%`

**排查**:
```bash
# 分析失败原因
grep "failed" test/stress/results/*/response-*.json | jq .

# 检查资源限制
ulimit -a

# 增加文件描述符
ulimit -n 65536
```

#### 3. 内存增长过快

**症状**: `Memory growth: 35%`

**排查**:
```bash
# pprof 内存分析
go tool pprof -alloc_space http://localhost:6060/debug/pprof/heap

# 查看 top 内存分配
(pprof) top20

# 检查 Goroutine 泄漏
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

## CI/CD 集成

### GitHub Actions

压力测试在 nightly build 中自动运行:

```yaml
# .github/workflows/stress-test.yml
on:
  schedule:
    - cron: '0 2 * * *'  # 每天凌晨 2 点
```

### 手动触发

```bash
# GitHub Actions UI
Actions -> Stress Tests -> Run workflow

# 或通过 gh CLI
gh workflow run stress-test.yml
```

## 最佳实践

### 1. 测试环境隔离

- 使用专用测试环境
- 资源配置接近生产环境
- 避免与开发环境混用

### 2. 性能基线

定期运行并记录基线:

```bash
# 保存基线报告
./test/stress/run_all_tests.sh > baseline-$(date +%Y%m%d).txt
```

### 3. 持续监控

- 每日 nightly 运行
- 失败自动告警
- 趋势分析

### 4. 故障注入

逐步增加故障复杂度:

1. 单点故障 (Server 或 Agent)
2. 混合故障 (Server + Agent)
3. 网络分区 (Post-MVP)
4. 资源耗尽 (Post-MVP)

## 参考资料

- [Event Sourcing 架构文档](../architecture.md)
- [Temporal 测试指南](https://docs.temporal.io/develop/go/testing-suite)
- [Go pprof 文档](https://golang.org/pkg/net/http/pprof/)
- [性能基准测试 (Story 7.3)](../sprint-artifacts/7-3-performance-benchmarking.md)
