# Waterflow 性能分析指南 (pprof)

本指南介绍如何使用 Go pprof 工具对 Waterflow Server 和 Agent 进行性能分析和内存泄漏排查。

## 目录

- [概述](#概述)
- [pprof 端点配置](#pprof-端点配置)
- [采集 Profiles](#采集-profiles)
- [分析 Profiles](#分析-profiles)
- [内存泄漏排查](#内存泄漏排查)
- [Goroutine 泄漏排查](#goroutine-泄漏排查)
- [CPU 性能分析](#cpu-性能分析)
- [最佳实践](#最佳实践)

## 概述

Waterflow 的 Server 和 Agent 内置了 Go pprof 端点,用于实时性能分析和问题诊断。

### 支持的 Profile 类型

- **heap**: 内存分配情况
- **goroutine**: Goroutine 数量和堆栈
- **threadcreate**: 线程创建堆栈
- **block**: 阻塞分析
- **mutex**: 互斥锁竞争
- **profile**: CPU profile (30秒采样)

## pprof 端点配置

### Server

**端点**: `http://localhost:6060/debug/pprof/`

Server 在 `cmd/server/main.go` 中自动启动 pprof 服务器:

```go
import _ "net/http/pprof"

// 在单独的端口启动 pprof
go func() {
    pprofAddr := ":6060"
    logger.Log.Info("Starting pprof server", zap.String("address", pprofAddr))
    if err := http.ListenAndServe(pprofAddr, nil); err != nil {
        logger.Log.Warn("pprof server failed", zap.Error(err))
    }
}()
```

### Agent

**端点**: `http://localhost:6061/debug/pprof/`

Agent 在 `cmd/agent/main.go` 中自动启动 pprof 服务器 (端口 6061):

```go
import _ "net/http/pprof"

go func() {
    pprofAddr := ":6061"
    logger.Log.Info("Starting pprof server", zap.String("address", pprofAddr))
    if err := http.ListenAndServe(pprofAddr, nil); err != nil {
        logger.Log.Warn("pprof server failed", zap.Error(err))
    }
}()
```

### 安全注意事项

⚠️ **生产环境**: pprof 端点仅监听 localhost,不暴露到外网。如需远程访问,使用 SSH 端口转发:

```bash
ssh -L 6060:localhost:6060 user@production-server
```

## 采集 Profiles

### 方式 1: Web 界面

浏览器访问:

- Server: http://localhost:6060/debug/pprof/
- Agent: http://localhost:6061/debug/pprof/

可用的 profiles:

- `/debug/pprof/heap` - 内存堆分析
- `/debug/pprof/goroutine` - Goroutine 列表
- `/debug/pprof/profile` - CPU profile (30秒)
- `/debug/pprof/block` - 阻塞分析
- `/debug/pprof/mutex` - 互斥锁分析

### 方式 2: 命令行采集

```bash
# 内存 heap profile
curl http://localhost:6060/debug/pprof/heap > heap.prof

# Goroutine profile
curl http://localhost:6060/debug/pprof/goroutine > goroutine.prof

# CPU profile (30秒采样)
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof

# 阻塞 profile
curl http://localhost:6060/debug/pprof/block > block.prof
```

### 方式 3: 交互式分析

直接连接到运行中的进程:

```bash
# 实时内存分析
go tool pprof http://localhost:6060/debug/pprof/heap

# 实时 Goroutine 分析
go tool pprof http://localhost:6060/debug/pprof/goroutine

# CPU profiling (30秒)
go tool pprof http://localhost:6060/debug/pprof/profile
```

## 分析 Profiles

### Web UI 分析

最直观的方式 - 使用 Web UI:

```bash
# 启动 Web UI (自动打开浏览器)
go tool pprof -http=:8080 heap.prof

# 或指定远程地址
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/heap
```

**Web UI 功能**:
- **Top**: 内存/CPU 占用 Top 函数
- **Graph**: 调用关系图
- **Flame Graph**: 火焰图 (推荐)
- **Source**: 源代码级分析

### 命令行分析

进入 pprof 交互式命令行:

```bash
go tool pprof heap.prof
```

**常用命令**:

```
(pprof) top          # 显示 Top 20 内存/CPU 占用
(pprof) top20        # 显示 Top 20
(pprof) list main    # 显示 main 包的详细信息
(pprof) web          # 生成调用图 (需要 graphviz)
(pprof) png          # 导出 PNG 图片
(pprof) pdf          # 导出 PDF 报告
(pprof) help         # 帮助
```

### 对比分析

对比两个 profile,找出变化:

```bash
# 采集基线
curl http://localhost:6060/debug/pprof/heap > heap-before.prof

# 运行负载测试
./test/stress/resource_leak_test.sh

# 采集对比
curl http://localhost:6060/debug/pprof/heap > heap-after.prof

# 对比分析 (显示增量)
go tool pprof -base heap-before.prof heap-after.prof
```

## 内存泄漏排查

### 完整诊断流程

#### 1. 采集基线 heap profile

```bash
curl http://localhost:6060/debug/pprof/heap > heap-baseline.prof
```

#### 2. 运行负载测试

```bash
# 运行 1 小时负载
./test/stress/resource_leak_test.sh
```

#### 3. 采集泄漏后的 profile

```bash
curl http://localhost:6060/debug/pprof/heap > heap-leak.prof
```

#### 4. 对比分析

```bash
go tool pprof -base heap-baseline.prof heap-leak.prof
```

在 pprof 中:

```
(pprof) top
# 查看内存增长最多的函数

(pprof) list <function_name>
# 查看具体代码行
```

#### 5. 使用 Web UI 火焰图

```bash
go tool pprof -http=:8080 -base heap-baseline.prof heap-leak.prof
```

点击 "Flame Graph" 查看内存分配火焰图。

### heap profile 参数

**inuse_space**: 当前使用的内存 (默认)
```bash
go tool pprof -inuse_space heap.prof
```

**alloc_space**: 累计分配的内存 (包括已释放)
```bash
go tool pprof -alloc_space heap.prof
```

**inuse_objects**: 当前对象数量
```bash
go tool pprof -inuse_objects heap.prof
```

**alloc_objects**: 累计分配的对象数量
```bash
go tool pprof -alloc_objects heap.prof
```

## Goroutine 泄漏排查

### 检测 Goroutine 泄漏

#### 1. 查看 Goroutine 数量

```bash
curl http://localhost:6060/debug/pprof/goroutine?debug=1
```

输出示例:
```
goroutine profile: total 1523
142 @ 0x... 0x... 0x...
#   0x...   runtime.gopark+0x...
#   0x...   ...
```

#### 2. 分析 Goroutine 堆栈

```bash
# 采集 Goroutine profile
curl http://localhost:6060/debug/pprof/goroutine > goroutine.prof

# 交互式分析
go tool pprof goroutine.prof
```

在 pprof 中:

```
(pprof) top
# 查看创建 Goroutine 最多的位置

(pprof) list <function_name>
# 查看具体代码
```

#### 3. Web UI 分析

```bash
go tool pprof -http=:8080 goroutine.prof
```

查看调用图,找出 Goroutine 创建点。

### 常见 Goroutine 泄漏场景

#### 1. 未关闭的 Channel

```go
// ❌ 泄漏示例
func badExample() {
    ch := make(chan int)
    go func() {
        <-ch  // 永远阻塞
    }()
}

// ✅ 正确示例
func goodExample() {
    ch := make(chan int)
    go func() {
        select {
        case <-ch:
        case <-time.After(5 * time.Second):
            return
        }
    }()
}
```

#### 2. 无限循环

```go
// ❌ 泄漏示例
func badExample() {
    go func() {
        for {
            // 无退出条件
        }
    }()
}

// ✅ 正确示例
func goodExample(ctx context.Context) {
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            default:
                // work
            }
        }
    }()
}
```

## CPU 性能分析

### 采集 CPU Profile

```bash
# 采集 30 秒 CPU profile
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof

# 分析
go tool pprof cpu.prof
```

### 实时 CPU 分析

```bash
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

等待 30 秒后进入交互式界面:

```
(pprof) top
# 查看 CPU 占用最高的函数

(pprof) web
# 生成调用图
```

### 火焰图分析

```bash
go tool pprof -http=:8080 cpu.prof
```

点击 "Flame Graph" 查看 CPU 火焰图,快速定位性能瓶颈。

## 最佳实践

### 1. 定期采集基线

建立性能基线,用于对比:

```bash
# 每周采集基线
mkdir -p profiles/baseline-$(date +%Y%m%d)
curl http://localhost:6060/debug/pprof/heap > profiles/baseline-$(date +%Y%m%d)/heap.prof
curl http://localhost:6060/debug/pprof/goroutine > profiles/baseline-$(date +%Y%m%d)/goroutine.prof
```

### 2. 生产环境采集

生产环境应谨慎采集 profile:

- **heap**: 影响小,可随时采集
- **goroutine**: 影响小,可随时采集
- **profile (CPU)**: 有一定开销,建议在低峰期采集
- **block/mutex**: 需提前启用,有性能影响

### 3. 自动化监控

集成到监控系统:

```bash
#!/bin/bash
# scripts/monitor_goroutines.sh

while true; do
    GOROUTINES=$(curl -s http://localhost:6060/debug/pprof/goroutine?debug=1 | head -1 | awk '{print $4}')
    echo "$(date +%s) $GOROUTINES" >> goroutines.log
    
    if [ $GOROUTINES -gt 10000 ]; then
        echo "⚠️ Goroutine leak detected: $GOROUTINES"
        curl http://localhost:6060/debug/pprof/goroutine > goroutine-leak-$(date +%Y%m%d-%H%M%S).prof
    fi
    
    sleep 60
done
```

### 4. 结合压力测试

压力测试时同时采集 profiles:

```bash
# 开始采集
curl http://localhost:6060/debug/pprof/heap > heap-before.prof

# 运行压力测试
./test/stress/concurrent_workflows_test.sh

# 结束采集
curl http://localhost:6060/debug/pprof/heap > heap-after.prof

# 对比分析
go tool pprof -base heap-before.prof heap-after.prof
```

## 参考资料

- [Go pprof 官方文档](https://pkg.go.dev/net/http/pprof)
- [Profiling Go Programs](https://go.dev/blog/pprof)
- [Flame Graphs](http://www.brendangregg.com/flamegraphs.html)
- [压力测试指南](./stress-testing.md)
