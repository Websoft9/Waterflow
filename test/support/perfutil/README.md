# Performance Testing Utilities

本包提供性能测试和压力测试所需的通用工具。

## 使用场景

- `test/performance/` - 性能基准测试
- `test/stress/` - 压力测试和容错验证

## 提供的工具

### 1. 内存泄漏检测

```go
import "github.com/Websoft9/waterflow/test/support/perfutil"

func TestResourceLeak(t *testing.T) {
    // 捕获基线
    baseline := perfutil.CaptureMemoryBaseline(t)
    
    // 运行测试逻辑
    for i := 0; i < 100; i++ {
        // 执行操作...
    }
    
    // 验证无泄漏 (最大增长 50MB)
    perfutil.AssertNoMemoryLeak(t, baseline, 50)
}
```

### 2. Goroutine 泄漏检测

```go
func TestGoroutineLeak(t *testing.T) {
    tracker := perfutil.StartGoroutineTracking(t)
    
    // 运行测试逻辑
    for i := 0; i < 100; i++ {
        go doSomething()
    }
    
    // 验证无泄漏 (最大增长 20 个 goroutine)
    tracker.AssertNoGoroutineLeak(t, 20)
}
```

### 3. 崩溃恢复测试

```go
import "os/exec"

func TestServerCrashRecovery(t *testing.T) {
    // 启动服务器进程
    cmd := exec.Command("./bin/server")
    cmd.Start()
    
    simulator := perfutil.NewCrashSimulator(t, cmd)
    simulator.WaitForStartup(5 * time.Second)
    
    // 获取崩溃前的事件数
    eventsBefore := perfutil.VerifyEventHistory(t, temporalClient, workflowID)
    
    // 模拟崩溃
    simulator.SimulateCrash()
    
    // 重启服务器...
    
    // 验证状态恢复 (PRD#L235: 100% 恢复)
    perfutil.AssertStateRecovery(t, temporalClient, workflowID, eventsBefore)
    perfutil.AssertWorkflowRecoverable(t, temporalClient, workflowID)
}
```

### 4. 性能基准测试

```go
func BenchmarkConcurrent(b *testing.B) {
    perfutil.BenchmarkConcurrent(b, 100, func() {
        // 并发执行的操作
        client.SubmitWorkflow(ctx, req)
    })
}
```

### 5. 执行时间断言

```go
func TestResponseTime(t *testing.T) {
    perfutil.AssertDurationWithin(t, 5*time.Second, func() {
        // 应该在 5 秒内完成
        client.SubmitWorkflow(ctx, req)
    })
}
```

## API 参考

### 内存管理

- `CaptureMemoryBaseline(t) *MemStats` - 捕获内存基线
- `GetCurrentMemStats(t) *MemStats` - 获取当前内存统计
- `MemoryGrowthMB(baseline, current) int64` - 计算内存增长
- `AssertNoMemoryLeak(t, baseline, maxGrowthMB)` - 断言无内存泄漏

### Goroutine 管理

- `StartGoroutineTracking(t) *GoroutineTracker` - 开始跟踪 goroutine
- `tracker.AssertNoGoroutineLeak(t, maxGrowth)` - 断言无 goroutine 泄漏

### 崩溃恢复

- `NewCrashSimulator(t, cmd) *CrashSimulator` - 创建崩溃模拟器
- `simulator.WaitForStartup(duration)` - 等待进程启动
- `simulator.SimulateCrash() error` - 模拟崩溃 (SIGKILL)
- `VerifyEventHistory(t, client, workflowID) int` - 验证事件历史
- `AssertStateRecovery(t, client, workflowID, expectedEvents)` - 断言状态恢复
- `AssertWorkflowRecoverable(t, client, workflowID)` - 断言工作流可恢复

### 性能测量

- `BenchmarkConcurrent(b, concurrency, fn)` - 并发基准测试
- `MeasureDuration(t, name, fn) time.Duration` - 测量执行时间
- `AssertDurationWithin(t, timeout, fn)` - 断言执行时间在限制内

## 最佳实践

### 内存泄漏测试

```go
func TestMemoryLeak(t *testing.T) {
    baseline := perfutil.CaptureMemoryBaseline(t)
    
    // 执行多次操作
    iterations := 100
    for i := 0; i < iterations; i++ {
        // 执行操作...
        
        // 定期 GC 辅助检测
        if i%20 == 0 {
            runtime.GC()
            time.Sleep(100 * time.Millisecond)
        }
    }
    
    // 最终验证
    perfutil.AssertNoMemoryLeak(t, baseline, 50)
}
```

### 崩溃恢复测试

```go
func TestCrashRecovery(t *testing.T) {
    // 1. 启动并等待就绪
    cmd := exec.Command("./bin/server")
    cmd.Start()
    simulator := perfutil.NewCrashSimulator(t, cmd)
    simulator.WaitForStartup(5 * time.Second)
    
    // 2. 运行工作流并记录状态
    workflowID := "test-crash"
    client.ExecuteWorkflow(ctx, options, workflowID)
    eventsBefore := perfutil.VerifyEventHistory(t, tc, workflowID)
    
    // 3. 模拟崩溃
    simulator.SimulateCrash()
    time.Sleep(3 * time.Second)
    
    // 4. 重启服务器
    cmd = exec.Command("./bin/server")
    cmd.Start()
    defer cmd.Process.Kill()
    time.Sleep(5 * time.Second)
    
    // 5. 验证恢复
    perfutil.AssertStateRecovery(t, tc, workflowID, eventsBefore)
}
```

## PRD 追溯

所有工具直接支持 PRD 技术成功标准验证：

- **PRD#L235** (持久性) - `AssertStateRecovery` 验证 100% 状态恢复
- **PRD#L236** (可靠性) - `AssertNoMemoryLeak`, `AssertNoGoroutineLeak` 验证资源管理
- **PRD#L238** (可扩展性) - `BenchmarkConcurrent` 验证并发性能
- **PRD#L249** (容错能力) - `AssertDurationWithin` 验证超时策略
