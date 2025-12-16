# ADR-0002: 单节点执行模式

**状态:** ✅ 已采纳  
**日期:** 2025-12-15  
**决策者:** 架构团队  

## 背景

在工作流执行中,每个节点(Step)需要独立的超时和重试配置。例如:

```yaml
jobs:
  build:
    steps:
      - name: Checkout
        timeout-minutes: 5
        retry:
          attempts: 3
      
      - name: Build
        timeout-minutes: 30
        retry:
          attempts: 1
```

需要决定如何映射到 Temporal 的执行模型:
1. **批处理模式** - 一个 Job 内所有 Steps 在一个 Activity 中串行执行
2. **单节点模式** - 每个 Step 映射为一个独立的 Activity 调用

## 决策

采用 **单节点执行模式**:每个 Step 映射为一个独立的 Activity 调用。

## 理由

### 核心优势:

1. **细粒度超时控制**

```go
// Workflow 中为每个 Step 设置独立超时
for _, step := range job.Steps {
    ctx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
        StartToCloseTimeout: time.Duration(step.TimeoutMinutes) * time.Minute,
        RetryPolicy: &temporal.RetryPolicy{
            MaximumAttempts: step.Retry.Attempts,
        },
    })
    
    err := workflow.ExecuteActivity(ctx, "ExecuteNode", step).Get(ctx, nil)
}
```

2. **独立的重试策略**
   - 每个 Step 可以有不同的重试次数和退避策略
   - 失败的 Step 不影响其他 Step 的执行

3. **完整的可观测性**
   - Temporal UI 显示每个 Step 的执行状态
   - Event History 记录每个 Activity 的启动/完成/失败
   - 方便定位问题节点

4. **灵活的容错**
   - 单个 Step 失败可以单独重试
   - 支持 Step 级别的 continue-on-error

### 与批处理模式对比:

| 维度 | 单节点模式 | 批处理模式 | 决策 |
|------|------------|------------|------|
| **超时粒度** | 每个 Step 独立 | 整个 Job 共享 | ✅ 单节点 |
| **重试粒度** | 每个 Step 独立 | 整个 Job 重试 | ✅ 单节点 |
| **可观测性** | 每个 Step 可见 | 只能看到 Job | ✅ 单节点 |
| **并发执行** | 自然支持 | 需要复杂实现 | ✅ 单节点 |
| **失败恢复** | 从失败 Step 重试 | 整个 Job 重新开始 | ✅ 单节点 |

## 后果

### 正面影响:

✅ **精确控制** - 每个 Step 的超时/重试完全独立  
✅ **清晰状态** - Temporal UI 展示每个 Step 的执行状态  
✅ **易于调试** - 快速定位失败的具体 Step  
✅ **自然并发** - 多个 Step 可并行执行(future)  

### 负面影响:

⚠️ **Activity 数量多** - 一个 Job 10 个 Steps = 10 个 Activity 调用  
⚠️ **Event History 长** - 每个 Activity 产生多个 Event  

### 性能评估:

基于 Temporal 官方性能数据:
- 单个 Workflow 可支持 10,000+ Activity 调用
- Event History 可存储百万级事件
- 对于常见工作流(< 100 Steps),性能完全不是问题

## 实现示例

### Workflow 中调用单个节点:

```go
func RunJobWorkflow(ctx workflow.Context, job *Job) error {
    for _, step := range job.Steps {
        // 每个 Step 独立的 Activity 选项
        activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
            StartToCloseTimeout: time.Duration(step.TimeoutMinutes) * time.Minute,
            RetryPolicy: &temporal.RetryPolicy{
                MaximumAttempts:    step.Retry.Attempts,
                InitialInterval:    time.Second,
                BackoffCoefficient: 2.0,
            },
        })
        
        var result NodeResult
        err := workflow.ExecuteActivity(activityCtx, "ExecuteNode", ExecuteNodeInput{
            NodeType: step.Uses,
            Args:     step.With,
        }).Get(activityCtx, &result)
        
        if err != nil && !step.ContinueOnError {
            return err
        }
    }
    return nil
}
```

### Agent 中执行节点:

```go
func (w *Worker) ExecuteNode(ctx context.Context, input ExecuteNodeInput) (NodeResult, error) {
    // 从 PluginManager 加载节点实现
    node, err := w.pluginManager.GetNode(input.NodeType)
    if err != nil {
        return nil, err
    }
    
    // 执行单个节点
    return node.Execute(ctx, input.Args)
}
```

## 替代方案

### 方案 A: 批处理模式 (被拒绝)

所有 Steps 在一个 Activity 中执行:

```go
func ExecuteBatchSteps(ctx context.Context, steps []Step) error {
    for _, step := range steps {
        // 所有 Steps 共享 Activity 超时
        err := executeStep(step)
        if err != nil {
            return err // 整个批次失败
        }
    }
    return nil
}
```

**被拒绝原因:**
- ❌ 无法为单个 Step 设置超时
- ❌ 一个 Step 失败导致所有 Step 重新执行
- ❌ 无法在 Temporal UI 看到单个 Step 状态

### 方案 B: 混合模式 (考虑但未采纳)

快速步骤批处理,慢速步骤独立:

```yaml
steps:
  - batch: [checkout, setup-env]  # 批处理
  - name: Build                   # 独立 Activity
    timeout-minutes: 30
```

**未采纳原因:**
- 增加复杂度,用户需要理解批处理概念
- 配置不一致,部分 Steps 支持超时,部分不支持
- 收益不明显(Activity 开销很小)

## 参考资料

- [Temporal Activity 最佳实践](https://docs.temporal.io/docs/go/activities/)
- [架构优化总结](../analysis/architecture-optimization-summary.md)
