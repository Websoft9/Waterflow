# Story 1.13: 条件执行支持

Status: ready-for-dev

## Story

As a **工作流用户**,  
I want **条件化执行 Step**,  
So that **根据运行时状态决定是否执行**。

## Acceptance Criteria

**Given** Step 配置了 `if` 条件  
**When** 工作流执行到该 Step  
**Then** 求值 if 表达式  
**And** 表达式为 true 时执行 Step  
**And** 表达式为 false 时跳过 Step  
**And** 支持引用前序 Step 的输出  
**And** 条件求值失败中止工作流

## Technical Context

### Epic Context

**Epic 1: 核心工作流引擎基础**

本 Story 是 Epic 1 的第 13 个 Story,实现工作流的条件执行能力,让用户可以根据运行时状态动态决定是否执行某个 Step。

**前置依赖:**
- ✅ Story 1.11: 变量系统 - 表达式引擎框架
- ✅ Story 1.12: 表达式求值引擎 - 运算符和函数支持
- ✅ Story 1.6: 基础工作流执行引擎 - Temporal Workflow 执行

**本 Story 实现范围:**
- ✅ 解析 Step 的 `if` 字段
- ✅ 在 Step 执行前求值条件表达式
- ✅ 基于求值结果决定是否执行 Step
- ✅ 条件为 false 时跳过 Step (标记为 skipped)
- ✅ 条件求值失败时中止工作流
- ✅ 支持引用 `job.status`, `workflow.id` 等上下文

**后续 Story 依赖:**
- Story 1.14: Step 输出引用 - 扩展条件表达式可以引用前序 Step 的输出

### Architecture Requirements

#### 1. YAML DSL 扩展: Step 的 `if` 字段

根据 ADR-0004 YAML DSL 语法设计,Step 需要支持 `if` 字段:

```yaml
jobs:
  deploy:
    runs-on: production-servers
    steps:
      # 简单条件
      - name: Deploy to Production
        if: ${{ vars.env == 'production' }}
        uses: deploy@v1
      
      # 基于 Job 状态
      - name: Notify on Success
        if: ${{ job.status == 'success' }}
        uses: notify@v1
      
      # 复杂逻辑表达式
      - name: Deploy with Validation
        if: ${{ vars.validated && vars.approved && !vars.skipDeploy }}
        uses: deploy@v1
      
      # 引用前序 Step 输出 (Story 1.14 扩展)
      - name: Use Build Output
        if: ${{ steps.build.outputs.exitCode == 0 }}
        uses: deploy@v1
```

**数据结构扩展:**

```go
// pkg/dsl/workflow.go
type Step struct {
    Name           string                 `yaml:"name"`
    Uses           string                 `yaml:"uses"`
    With           map[string]interface{} `yaml:"with"`
    If             string                 `yaml:"if,omitempty"`  // 新增条件字段
    TimeoutMinutes int                    `yaml:"timeout-minutes,omitempty"`
    Retry          *RetryConfig           `yaml:"retry,omitempty"`
}
```

#### 2. Temporal Workflow 集成: 条件判断逻辑

基于 Story 1.6 的单节点执行模式 (ADR-0002),每个 Step = 1 个 Temporal Activity。条件判断需要在 Workflow 函数中实现,**在调用 Activity 之前**求值条件。

**工作流执行流程:**

```
解析 YAML → 提交到 Temporal
    ↓
Temporal Workflow 开始
    ↓
遍历 Jobs
    ↓
遍历 Steps
    ↓
检查 Step.If 条件 ← 本 Story 添加
    ├─ 条件为空 → 执行 Step (调用 Activity)
    ├─ 条件为 true → 执行 Step (调用 Activity)
    ├─ 条件为 false → 跳过 Step (标记 skipped)
    └─ 条件求值失败 → 中止 Workflow (返回错误)
```

**关键决策:** 条件求值在 **Workflow 函数内部** 进行,不是在 Server 端提前求值,因为:
1. Job 状态 (`job.status`) 是运行时动态的
2. Step 输出 (`steps.*.outputs`) 在执行过程中产生
3. Workflow 函数可以访问完整的运行时上下文

#### 3. 表达式上下文扩展

本 Story 需要扩展表达式上下文,支持运行时动态值:

```go
// pkg/expr/context.go
func BuildWorkflowContext(workflow *Workflow, job *Job, executedSteps map[string]StepResult) map[string]interface{} {
    return map[string]interface{}{
        // 变量上下文 (Story 1.11)
        "vars": workflow.Vars,
        
        // Workflow 上下文 (运行时)
        "workflow": map[string]interface{}{
            "id":   workflow.ID,
            "name": workflow.Name,
        },
        
        // Job 上下文 (运行时)
        "job": map[string]interface{}{
            "id":     job.ID,
            "status": job.Status,  // success/failure/running
        },
        
        // Steps 上下文 (Story 1.14 扩展)
        "steps": executedSteps,
        
        // 环境变量上下文 (可选)
        "env": os.Environ(),
    }
}
```

**本 Story 实现范围:**
- ✅ `vars` - 已由 Story 1.11 实现
- ✅ `workflow.id`, `workflow.name` - 本 Story 添加
- ✅ `job.id`, `job.status` - 本 Story 添加
- ⏭️ `steps.*` - Story 1.14 实现

#### 4. ADR-0002: 单节点执行模式

每个 Step 映射为 1 个 Temporal Activity 调用。条件判断逻辑:

```go
// internal/workflow/workflow.go (Temporal Workflow 函数)
func WaterflowWorkflow(ctx workflow.Context, wf *Workflow) error {
    for _, job := range wf.Jobs {
        for _, step := range job.Steps {
            // 构建表达式上下文 (包含运行时状态)
            exprContext := BuildWorkflowContext(wf, job, executedSteps)
            
            // 求值 if 条件
            if step.If != "" {
                shouldExecute, err := evaluateCondition(ctx, step.If, exprContext)
                if err != nil {
                    return fmt.Errorf("failed to evaluate condition for step %s: %w", step.Name, err)
                }
                
                if !shouldExecute {
                    // 跳过 Step,记录日志
                    workflow.GetLogger(ctx).Info("Step skipped", "step", step.Name, "condition", step.If)
                    continue  // 不调用 Activity
                }
            }
            
            // 执行 Step (调用 Activity)
            err := workflow.ExecuteActivity(ctx, ExecuteNodeActivity, step).Get(ctx, &result)
            if err != nil {
                return err
            }
            
            // 更新 Job 状态
            job.Status = "success"
        }
    }
    
    return nil
}
```

### Previous Story Learnings

#### Story 1.6: 基础工作流执行引擎

**已实现的 Temporal Workflow 框架:**

```go
// internal/workflow/workflow.go
func WaterflowWorkflow(ctx workflow.Context, wf *WorkflowDefinition) error {
    // 遍历 Jobs 和 Steps
    // 每个 Step 调用 Activity
}
```

**本 Story 扩展点:**
- 在调用 `ExecuteActivity` 之前添加条件判断逻辑
- 记录 skipped Steps 到 Workflow 历史

**文件位置:**
- `internal/workflow/workflow.go` - Temporal Workflow 函数
- `internal/workflow/activities.go` - Activity 定义

#### Story 1.11 & 1.12: 表达式引擎

**已实现的表达式求值能力:**
- ✅ 变量引用: `vars.name`
- ✅ 运算符: `==`, `!=`, `&&`, `||`, `!`
- ✅ 函数: `contains()`, `startsWith()`, 等

**本 Story 复用:**
- 直接使用 `expr.Engine.Evaluate()` 求值 `if` 条件
- 扩展上下文添加 `workflow` 和 `job` 对象

**关键洞察:**
表达式引擎已完全就绪,本 Story 只需:
1. 扩展上下文构建器
2. 在 Workflow 函数中调用表达式引擎
3. 处理求值结果 (true/false)

### Implementation Approach

#### Phase 1: YAML DSL 扩展

**1. 扩展 Step 数据结构**

```go
// pkg/dsl/workflow.go
type Step struct {
    Name           string                 `yaml:"name"`
    Uses           string                 `yaml:"uses"`
    With           map[string]interface{} `yaml:"with"`
    If             string                 `yaml:"if,omitempty"`  // 新增
    TimeoutMinutes int                    `yaml:"timeout-minutes,omitempty"`
    Retry          *RetryConfig           `yaml:"retry,omitempty"`
    ContinueOnError bool                  `yaml:"continue-on-error,omitempty"`
}
```

**2. YAML 解析验证**

```go
// pkg/dsl/parser.go
func ParseWorkflow(data []byte) (*Workflow, error) {
    var wf Workflow
    if err := yaml.Unmarshal(data, &wf); err != nil {
        return nil, err
    }
    
    // 验证 if 字段 (可选)
    for jobName, job := range wf.Jobs {
        for i, step := range job.Steps {
            if step.If != "" {
                // 基础语法检查: 是否包含 ${{ }}
                if !strings.Contains(step.If, "${{") {
                    return nil, fmt.Errorf("job %s step %d: 'if' must be an expression wrapped in ${{ }}", jobName, i)
                }
            }
        }
    }
    
    return &wf, nil
}
```

#### Phase 2: 表达式上下文扩展

**1. 扩展上下文构建器**

```go
// pkg/expr/context.go
package expr

import (
    "github.com/websoft9/waterflow/pkg/dsl"
)

// WorkflowRuntimeContext 工作流运行时上下文
type WorkflowRuntimeContext struct {
    Workflow      *dsl.Workflow
    Job           *dsl.Job
    ExecutedSteps map[string]StepOutput
}

// StepOutput Step 执行输出
type StepOutput struct {
    Outputs map[string]interface{}
    Status  string  // success/failure
}

// BuildRuntimeContext 构建包含运行时状态的表达式上下文
func BuildRuntimeContext(rtCtx *WorkflowRuntimeContext) map[string]interface{} {
    ctx := make(map[string]interface{})
    
    // 变量上下文 (来自 YAML vars 定义)
    if rtCtx.Workflow.Vars != nil {
        ctx["vars"] = rtCtx.Workflow.Vars
    }
    
    // Workflow 上下文 (运行时)
    ctx["workflow"] = map[string]interface{}{
        "id":   rtCtx.Workflow.ID,
        "name": rtCtx.Workflow.Name,
    }
    
    // Job 上下文 (运行时)
    ctx["job"] = map[string]interface{}{
        "id":     rtCtx.Job.ID,
        "status": rtCtx.Job.Status,
    }
    
    // Steps 上下文 (Story 1.14 扩展)
    ctx["steps"] = rtCtx.ExecutedSteps
    
    return ctx
}
```

**2. 辅助函数: 状态检查函数**

```go
// pkg/expr/context.go

// RegisterStatusFunctions 注册状态检查函数 (GHA 兼容)
func RegisterStatusFunctions(env map[string]interface{}, jobStatus string) {
    // success() - 所有前置步骤成功
    env["success"] = func() bool {
        return jobStatus == "success" || jobStatus == "running"
    }
    
    // failure() - 任一前置步骤失败
    env["failure"] = func() bool {
        return jobStatus == "failure"
    }
    
    // always() - 总是执行
    env["always"] = func() bool {
        return true
    }
    
    // cancelled() - 工作流被取消
    env["cancelled"] = func() bool {
        return jobStatus == "cancelled"
    }
}
```

#### Phase 3: Workflow 函数集成

**1. 条件求值函数**

```go
// internal/workflow/conditional.go
package workflow

import (
    "context"
    "fmt"
    
    "github.com/websoft9/waterflow/pkg/expr"
)

// EvaluateStepCondition 求值 Step 的 if 条件
func EvaluateStepCondition(ifCondition string, rtCtx *expr.WorkflowRuntimeContext) (bool, error) {
    if ifCondition == "" {
        return true, nil  // 无条件,总是执行
    }
    
    // 构建表达式上下文
    exprContext := expr.BuildRuntimeContext(rtCtx)
    
    // 注册状态函数
    expr.RegisterStatusFunctions(exprContext, rtCtx.Job.Status)
    
    // 创建表达式引擎
    engine := expr.NewEngine(exprContext)
    
    // 提取表达式内容 (去除 ${{ }})
    expression := extractExpression(ifCondition)
    
    // 求值
    result, err := engine.Evaluate(context.Background(), expression)
    if err != nil {
        return false, fmt.Errorf("failed to evaluate condition '%s': %w", ifCondition, err)
    }
    
    // 类型检查: 必须是 bool
    boolResult, ok := result.(bool)
    if !ok {
        return false, fmt.Errorf("condition must evaluate to boolean, got %T: %v", result, result)
    }
    
    return boolResult, nil
}

// extractExpression 从 ${{ expression }} 中提取表达式
func extractExpression(ifCondition string) string {
    // 去除 ${{ 和 }}
    expr := strings.TrimSpace(ifCondition)
    expr = strings.TrimPrefix(expr, "${{")
    expr = strings.TrimSuffix(expr, "}}")
    return strings.TrimSpace(expr)
}
```

**2. 修改 Workflow 函数**

```go
// internal/workflow/workflow.go
package workflow

import (
    "fmt"
    
    "go.temporal.io/sdk/workflow"
    
    "github.com/websoft9/waterflow/pkg/dsl"
    "github.com/websoft9/waterflow/pkg/expr"
)

// WaterflowWorkflow Temporal Workflow 函数
func WaterflowWorkflow(ctx workflow.Context, wf *dsl.Workflow) error {
    logger := workflow.GetLogger(ctx)
    
    // 遍历 Jobs
    for jobName, job := range wf.Jobs {
        logger.Info("Starting job", "job", jobName)
        
        // 初始化 Job 状态
        job.Status = "running"
        job.ID = fmt.Sprintf("%s-%s", wf.ID, jobName)
        
        // 跟踪已执行的 Steps (用于 steps.* 上下文)
        executedSteps := make(map[string]expr.StepOutput)
        
        // 遍历 Steps
        for i, step := range job.Steps {
            logger.Info("Processing step", "step", step.Name, "index", i)
            
            // 构建运行时上下文
            rtCtx := &expr.WorkflowRuntimeContext{
                Workflow:      wf,
                Job:           &job,
                ExecutedSteps: executedSteps,
            }
            
            // 求值 if 条件
            if step.If != "" {
                shouldExecute, err := EvaluateStepCondition(step.If, rtCtx)
                if err != nil {
                    logger.Error("Condition evaluation failed", "step", step.Name, "condition", step.If, "error", err)
                    job.Status = "failure"
                    return fmt.Errorf("failed to evaluate condition for step '%s': %w", step.Name, err)
                }
                
                if !shouldExecute {
                    logger.Info("Step skipped due to condition", "step", step.Name, "condition", step.If)
                    
                    // 记录 skipped Step
                    executedSteps[step.Name] = expr.StepOutput{
                        Status: "skipped",
                    }
                    
                    continue  // 跳过此 Step,不调用 Activity
                }
            }
            
            // 执行 Step (调用 Activity)
            logger.Info("Executing step", "step", step.Name, "node", step.Uses)
            
            var activityResult ActivityResult
            activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
                StartToCloseTimeout: getStepTimeout(step),
                RetryPolicy:         getStepRetryPolicy(step),
            })
            
            err := workflow.ExecuteActivity(activityCtx, ExecuteNodeActivity, step).Get(ctx, &activityResult)
            
            if err != nil {
                logger.Error("Step execution failed", "step", step.Name, "error", err)
                
                // 记录失败的 Step
                executedSteps[step.Name] = expr.StepOutput{
                    Status: "failure",
                }
                
                // 检查是否继续
                if !step.ContinueOnError {
                    job.Status = "failure"
                    return fmt.Errorf("step '%s' failed: %w", step.Name, err)
                }
            } else {
                logger.Info("Step completed successfully", "step", step.Name)
                
                // 记录成功的 Step (包含输出)
                executedSteps[step.Name] = expr.StepOutput{
                    Status:  "success",
                    Outputs: activityResult.Outputs,
                }
            }
        }
        
        // Job 完成
        job.Status = "success"
        logger.Info("Job completed", "job", jobName)
    }
    
    return nil
}

// ActivityResult Activity 执行结果
type ActivityResult struct {
    Outputs map[string]interface{}
}

// getStepTimeout 获取 Step 超时配置
func getStepTimeout(step dsl.Step) time.Duration {
    if step.TimeoutMinutes > 0 {
        return time.Duration(step.TimeoutMinutes) * time.Minute
    }
    return 30 * time.Minute  // 默认 30 分钟
}

// getStepRetryPolicy 获取 Step 重试策略
func getStepRetryPolicy(step dsl.Step) *temporal.RetryPolicy {
    if step.Retry != nil {
        return &temporal.RetryPolicy{
            MaximumAttempts: step.Retry.MaxAttempts,
            // ... 其他重试配置
        }
    }
    return nil
}
```

#### Phase 4: 测试

**1. 单元测试: 条件求值**

```go
// internal/workflow/conditional_test.go
func TestEvaluateStepCondition(t *testing.T) {
    tests := []struct {
        name        string
        ifCondition string
        rtCtx       *expr.WorkflowRuntimeContext
        want        bool
        wantErr     bool
    }{
        {
            name:        "empty condition - always execute",
            ifCondition: "",
            rtCtx:       &expr.WorkflowRuntimeContext{},
            want:        true,
            wantErr:     false,
        },
        {
            name:        "simple true condition",
            ifCondition: "${{ vars.enabled }}",
            rtCtx: &expr.WorkflowRuntimeContext{
                Workflow: &dsl.Workflow{
                    Vars: map[string]interface{}{"enabled": true},
                },
                Job: &dsl.Job{Status: "running"},
            },
            want:    true,
            wantErr: false,
        },
        {
            name:        "simple false condition",
            ifCondition: "${{ vars.enabled }}",
            rtCtx: &expr.WorkflowRuntimeContext{
                Workflow: &dsl.Workflow{
                    Vars: map[string]interface{}{"enabled": false},
                },
                Job: &dsl.Job{Status: "running"},
            },
            want:    false,
            wantErr: false,
        },
        {
            name:        "job status check",
            ifCondition: "${{ job.status == 'success' }}",
            rtCtx: &expr.WorkflowRuntimeContext{
                Workflow: &dsl.Workflow{},
                Job:      &dsl.Job{Status: "success"},
            },
            want:    true,
            wantErr: false,
        },
        {
            name:        "complex condition",
            ifCondition: "${{ vars.env == 'production' && job.status == 'running' }}",
            rtCtx: &expr.WorkflowRuntimeContext{
                Workflow: &dsl.Workflow{
                    Vars: map[string]interface{}{"env": "production"},
                },
                Job: &dsl.Job{Status: "running"},
            },
            want:    true,
            wantErr: false,
        },
        {
            name:        "status function - success()",
            ifCondition: "${{ success() }}",
            rtCtx: &expr.WorkflowRuntimeContext{
                Workflow: &dsl.Workflow{},
                Job:      &dsl.Job{Status: "success"},
            },
            want:    true,
            wantErr: false,
        },
        {
            name:        "status function - failure()",
            ifCondition: "${{ failure() }}",
            rtCtx: &expr.WorkflowRuntimeContext{
                Workflow: &dsl.Workflow{},
                Job:      &dsl.Job{Status: "failure"},
            },
            want:    true,
            wantErr: false,
        },
        {
            name:        "invalid expression",
            ifCondition: "${{ vars.undefined }}",
            rtCtx: &expr.WorkflowRuntimeContext{
                Workflow: &dsl.Workflow{Vars: map[string]interface{}{}},
                Job:      &dsl.Job{Status: "running"},
            },
            want:    false,
            wantErr: true,
        },
        {
            name:        "non-boolean result",
            ifCondition: "${{ vars.count }}",
            rtCtx: &expr.WorkflowRuntimeContext{
                Workflow: &dsl.Workflow{
                    Vars: map[string]interface{}{"count": 10},
                },
                Job: &dsl.Job{Status: "running"},
            },
            want:    false,
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := EvaluateStepCondition(tt.ifCondition, tt.rtCtx)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("EvaluateStepCondition() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if !tt.wantErr && got != tt.want {
                t.Errorf("EvaluateStepCondition() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

**2. 集成测试: 完整工作流**

```go
// test/integration/conditional_execution_test.go
func TestConditionalExecution_E2E(t *testing.T) {
    yamlContent := `
name: Conditional Test
vars:
  deploy: true
  skipTests: false
  env: production

jobs:
  build:
    runs-on: test-runner
    steps:
      # 总是执行
      - name: Checkout
        uses: checkout@v1
      
      # 条件执行
      - name: Run Tests
        if: ${{ !vars.skipTests }}
        uses: test@v1
      
      # 基于环境
      - name: Deploy to Production
        if: ${{ vars.deploy && vars.env == 'production' }}
        uses: deploy@v1
      
      # 基于 Job 状态
      - name: Notify on Success
        if: ${{ success() }}
        uses: notify@v1
`
    
    // 解析工作流
    workflow, err := dsl.ParseWorkflow([]byte(yamlContent))
    require.NoError(t, err)
    
    // 验证 if 字段解析
    buildJob := workflow.Jobs["build"]
    assert.Empty(t, buildJob.Steps[0].If)  // Checkout - 无条件
    assert.Equal(t, "${{ !vars.skipTests }}", buildJob.Steps[1].If)
    assert.Equal(t, "${{ vars.deploy && vars.env == 'production' }}", buildJob.Steps[2].If)
    assert.Equal(t, "${{ success() }}", buildJob.Steps[3].If)
    
    // 模拟运行时上下文
    rtCtx := &expr.WorkflowRuntimeContext{
        Workflow:      workflow,
        Job:           &buildJob,
        ExecutedSteps: make(map[string]expr.StepOutput),
    }
    rtCtx.Job.Status = "running"
    
    // 测试条件求值
    // Step 1: Run Tests - should execute (!false = true)
    shouldExecute, err := workflow.EvaluateStepCondition(buildJob.Steps[1].If, rtCtx)
    require.NoError(t, err)
    assert.True(t, shouldExecute)
    
    // Step 2: Deploy - should execute (true && 'production' == 'production')
    shouldExecute, err = workflow.EvaluateStepCondition(buildJob.Steps[2].If, rtCtx)
    require.NoError(t, err)
    assert.True(t, shouldExecute)
    
    // 修改 Job 状态为 success
    rtCtx.Job.Status = "success"
    
    // Step 3: Notify - should execute (success() = true)
    shouldExecute, err = workflow.EvaluateStepCondition(buildJob.Steps[3].If, rtCtx)
    require.NoError(t, err)
    assert.True(t, shouldExecute)
}
```

**3. Temporal Workflow 测试**

```go
// internal/workflow/workflow_test.go
func TestWaterflowWorkflow_ConditionalSteps(t *testing.T) {
    testSuite := &testsuite.WorkflowTestSuite{}
    env := testSuite.NewTestWorkflowEnvironment()
    
    // 注册 Activities
    env.RegisterActivity(ExecuteNodeActivity)
    
    // 准备测试工作流
    wf := &dsl.Workflow{
        ID:   "test-wf-1",
        Name: "Conditional Test",
        Vars: map[string]interface{}{
            "runTests": true,
            "deploy":   false,
        },
        Jobs: map[string]dsl.Job{
            "test": {
                Steps: []dsl.Step{
                    {
                        Name: "Always Run",
                        Uses: "echo@v1",
                    },
                    {
                        Name: "Conditional Run",
                        Uses: "test@v1",
                        If:   "${{ vars.runTests }}",
                    },
                    {
                        Name: "Conditional Skip",
                        Uses: "deploy@v1",
                        If:   "${{ vars.deploy }}",
                    },
                },
            },
        },
    }
    
    // Mock Activity 响应
    env.OnActivity(ExecuteNodeActivity, mock.Anything, mock.Anything).Return(&ActivityResult{}, nil)
    
    // 执行 Workflow
    env.ExecuteWorkflow(WaterflowWorkflow, wf)
    
    // 验证结果
    require.True(t, env.IsWorkflowCompleted())
    require.NoError(t, env.GetWorkflowError())
    
    // 验证 Activity 调用次数
    // 应该调用 2 次: "Always Run" 和 "Conditional Run"
    // "Conditional Skip" 应该被跳过
    env.AssertNumberOfCalls(t, "ExecuteNodeActivity", 2)
}
```

### Error Handling

#### 1. 条件求值失败处理

```go
// 条件求值失败应该中止工作流,不是跳过 Step
func EvaluateStepCondition(ifCondition string, rtCtx *expr.WorkflowRuntimeContext) (bool, error) {
    // ...
    result, err := engine.Evaluate(context.Background(), expression)
    if err != nil {
        // 返回错误,中止工作流
        return false, &ConditionalError{
            Step:      rtCtx.CurrentStep,
            Condition: ifCondition,
            Cause:     err,
        }
    }
    // ...
}

type ConditionalError struct {
    Step      string
    Condition string
    Cause     error
}

func (e *ConditionalError) Error() string {
    return fmt.Sprintf("failed to evaluate condition for step '%s': %s\n  condition: %s\n  cause: %v",
        e.Step, e.Condition, e.Cause)
}
```

#### 2. 用户友好错误信息

```
Error: Failed to evaluate condition for step 'Deploy to Production'
  Condition: ${{ vars.environment == 'prod' }}
  Cause: Variable 'vars.environment' is undefined
  Suggestion: Define 'environment' in the 'vars' section or check spelling
  
  Example:
  vars:
    environment: production
```

### Performance Considerations

#### 1. 表达式缓存 (可选)

如果同一个条件在多个 Steps 中重复使用,可以缓存求值结果:

```go
type ConditionCache struct {
    cache map[string]bool
    mu    sync.RWMutex
}

func (c *ConditionCache) Evaluate(condition string, evaluator func() (bool, error)) (bool, error) {
    c.mu.RLock()
    if result, cached := c.cache[condition]; cached {
        c.mu.RUnlock()
        return result, nil
    }
    c.mu.RUnlock()
    
    result, err := evaluator()
    if err != nil {
        return false, err
    }
    
    c.mu.Lock()
    c.cache[condition] = result
    c.mu.Unlock()
    
    return result, nil
}
```

**Note:** MVP 阶段可以跳过缓存,性能已足够。

### Documentation Requirements

#### 1. 用户文档: 条件执行语法

```markdown
## Conditional Execution

使用 `if` 字段条件化执行 Steps。

### 基础语法

\`\`\`yaml
steps:
  - name: Step Name
    if: ${{ expression }}
    uses: node@v1
\`\`\`

### 示例

#### 基于变量

\`\`\`yaml
vars:
  deploy: true
  env: production

steps:
  - name: Deploy to Production
    if: ${{ vars.deploy && vars.env == 'production' }}
    uses: deploy@v1
\`\`\`

#### 基于 Job 状态

\`\`\`yaml
steps:
  - name: Notify on Success
    if: ${{ job.status == 'success' }}
    uses: notify@v1
\`\`\`

#### 使用状态函数

\`\`\`yaml
steps:
  - name: Cleanup on Failure
    if: ${{ failure() }}
    uses: cleanup@v1
  
  - name: Always Notify
    if: ${{ always() }}
    uses: notify@v1
\`\`\`

### 可用上下文

- `vars.*` - 工作流变量
- `workflow.id` - 工作流 ID
- `workflow.name` - 工作流名称
- `job.id` - Job ID
- `job.status` - Job 状态 (success/failure/running)

### 状态函数

- `success()` - 所有前置步骤成功
- `failure()` - 任一前置步骤失败
- `always()` - 总是执行
- `cancelled()` - 工作流被取消
```

### Integration with Story 1.14

本 Story 实现后,Story 1.14 只需扩展上下文添加 `steps.*`:

```go
// Story 1.14 将扩展
func BuildRuntimeContext(rtCtx *WorkflowRuntimeContext) map[string]interface{} {
    ctx := BuildRuntimeContext(rtCtx)  // 复用本 Story 的上下文
    
    // 添加 steps 上下文
    ctx["steps"] = rtCtx.ExecutedSteps
    
    return ctx
}
```

然后用户就可以写:

```yaml
- name: Deploy
  if: ${{ steps.build.outputs.success == true }}
  uses: deploy@v1
```

### Acceptance Criteria Verification

✅ **AC1:** 求值 if 表达式
- 实现: `EvaluateStepCondition()` 函数

✅ **AC2:** 表达式为 true 时执行 Step
- 实现: Workflow 函数调用 Activity

✅ **AC3:** 表达式为 false 时跳过 Step
- 实现: `continue` 跳过 Activity 调用,记录 skipped

✅ **AC4:** 支持引用前序 Step 的输出
- 实现: 上下文包含 `executedSteps`,Story 1.14 扩展

✅ **AC5:** 条件求值失败中止工作流
- 实现: 返回错误,Workflow 终止

## Tasks / Subtasks

### Phase 0: 依赖验证 (AC: All)

- [ ] **Task 0.1:** 创建依赖验证脚本
  - [ ] 创建 `test/verify-dependencies-story-1-13.sh`
  - [ ] 验证 Story 1.11 表达式引擎存在
    - [ ] 检查 `pkg/expr/engine.go` 存在
    - [ ] 检查 `Engine` 接口定义
    - [ ] 检查 `DefaultEngine` 实现
  - [ ] 验证 Story 1.12 表达式求值能力
    - [ ] 检查 `registerBuiltinFunctions` 存在
    - [ ] 检查运算符支持 (`==`, `&&`, `||`, `!`)
  - [ ] 验证 Story 1.6 Workflow 函数框架
    - [ ] 检查 `internal/workflow/workflow.go` 存在
    - [ ] 检查 `WaterflowWorkflow` 函数定义
  - [ ] 验证 antonmedv/expr 库可用
    - [ ] 检查 `go.mod` 包含 `github.com/antonmedv/expr`

- [ ] **Task 0.2:** 运行依赖验证
  - [ ] 执行 `./test/verify-dependencies-story-1-13.sh`
  - [ ] 确认所有依赖就绪
  - [ ] 记录验证结果

**验证脚本内容:**

```bash
#!/bin/bash
# test/verify-dependencies-story-1-13.sh

set -e

echo "=== Story 1.13 Dependency Verification ==="

# Verify Story 1.11: Expression Engine
echo "Checking Story 1.11 (Expression Engine)..."
if [ ! -f "pkg/expr/engine.go" ]; then
    echo "ERROR: pkg/expr/engine.go not found. Story 1.11 required."
    exit 1
fi

# Check Engine interface
if ! grep -q "type Engine interface" pkg/expr/engine.go; then
    echo "ERROR: Engine interface not defined in pkg/expr/engine.go"
    exit 1
fi

# Check DefaultEngine
if ! grep -q "type DefaultEngine struct" pkg/expr/engine.go; then
    echo "ERROR: DefaultEngine not implemented in pkg/expr/engine.go"
    exit 1
fi

echo "✓ Story 1.11 (Expression Engine) verified"

# Verify Story 1.12: Expression Evaluation
echo "Checking Story 1.12 (Expression Evaluation)..."
if ! grep -q "registerBuiltinFunctions" pkg/expr/*.go; then
    echo "ERROR: registerBuiltinFunctions not found. Story 1.12 required."
    exit 1
fi

echo "✓ Story 1.12 (Expression Evaluation) verified"

# Verify Story 1.6: Workflow Function
echo "Checking Story 1.6 (Workflow Execution)..."
if [ ! -f "internal/workflow/workflow.go" ]; then
    echo "ERROR: internal/workflow/workflow.go not found. Story 1.6 required."
    exit 1
fi

if ! grep -q "func WaterflowWorkflow" internal/workflow/workflow.go; then
    echo "ERROR: WaterflowWorkflow function not defined"
    exit 1
fi

echo "✓ Story 1.6 (Workflow Execution) verified"

# Verify antonmedv/expr library
echo "Checking antonmedv/expr library..."
if ! grep -q "github.com/antonmedv/expr" go.mod; then
    echo "ERROR: antonmedv/expr not in go.mod. Run: go get github.com/antonmedv/expr@v1.15.0"
    exit 1
fi

echo "✓ antonmedv/expr library verified"

echo ""
echo "=== All Dependencies Verified ✓ ==="
echo "Story 1.13 can proceed with implementation."
```

### Phase 1: YAML DSL 扩展 (AC: 1)

- [x] **Task 1.1:** 扩展 Step 数据结构
  - [x] 在 `pkg/dsl/workflow.go` 添加 `If` 字段
  - [x] 更新 YAML tags

- [x] **Task 1.2:** 更新 Parser 验证
  - [x] 验证 `if` 字段格式
  - [x] 确保包含 `${{ }}`

### Phase 2: 表达式上下文扩展 (AC: 1, 4)

- [x] **Task 2.1:** 扩展上下文构建器
  - [x] 创建 `WorkflowRuntimeContext` 结构
  - [x] 实现 `BuildRuntimeContext()` 函数
  - [x] 添加 `workflow` 上下文
  - [x] 添加 `job` 上下文
  - [x] 添加 `steps` 上下文 (为 Story 1.14 预留)

- [x] **Task 2.2:** 注册状态函数
  - [x] 实现 `success()` 函数
  - [x] 实现 `failure()` 函数
  - [x] 实现 `always()` 函数
  - [x] 实现 `cancelled()` 函数

### Phase 3: Workflow 函数集成 (AC: 2, 3, 5)

- [x] **Task 3.1:** 实现条件求值函数
  - [x] 创建 `conditional.go`
  - [x] 实现 `EvaluateStepCondition()`
  - [x] 实现 `extractExpression()` 辅助函数

- [x] **Task 3.2:** 修改 Workflow 函数
  - [x] 在 Step 执行前添加条件检查
  - [x] 实现 true → 执行 Activity
  - [x] 实现 false → 跳过 Step,记录 skipped
  - [x] 实现求值失败 → 返回错误,中止 Workflow

- [x] **Task 3.3:** 跟踪已执行的 Steps
  - [x] 维护 `executedSteps` map
  - [x] 记录 Step 状态 (success/failure/skipped)
  - [x] 记录 Step 输出 (为 Story 1.14 准备)

### Phase 4: 错误处理 (AC: 5)

- [x] **Task 4.1:** 创建 ConditionalError 类型
  - [x] 包含 Step 名称、条件、原因
  - [x] 提供友好错误信息

- [x] **Task 4.2:** 错误场景处理
  - [x] 未定义变量
  - [x] 非布尔结果
  - [x] 语法错误

### Phase 5: 测试 (AC: All)

- [x] **Task 5.1:** 单元测试
  - [x] `conditional_test.go` - 条件求值测试
  - [x] 覆盖 true/false/错误场景
  - [x] 测试状态函数

- [x] **Task 5.2:** 集成测试
  - [x] 端到端工作流测试
  - [x] 混合条件和无条件 Steps
  - [x] 验证 skipped Steps 不调用 Activity

- [x] **Task 5.3:** Temporal Workflow 测试
  - [x] 使用 testsuite 测试框架
  - [x] Mock Activities
  - [x] 验证调用次数

- [ ] **Task 5.4:** 文档更新 (依赖 Story 11.3)
  - [ ] 添加条件执行语法文档
  - [ ] 添加示例和最佳实践
  - [ ] 更新 OpenAPI 规范

## Dev Notes

### Critical Implementation Notes

1. **条件求值位置:** 必须在 Temporal Workflow 函数内部,不是 Server 端,因为需要访问运行时状态

2. **错误优先级:** 条件求值失败应该中止工作流,不是跳过 Step,确保用户知道配置错误

3. **向后兼容:** `if` 字段是可选的,不影响现有工作流

4. **性能:** 条件求值开销很小 (< 1ms),不需要优化

5. **Temporal 历史:** Skipped Steps 会记录到 Temporal Event History,可以在 UI 中看到

### Project Structure Alignment

本 Story 遵循现有结构:
- `pkg/dsl/workflow.go` - 扩展 Step 结构
- `pkg/expr/context.go` - 扩展上下文构建器
- `internal/workflow/conditional.go` - 新增条件求值逻辑
- `internal/workflow/workflow.go` - 修改 Workflow 函数

### Testing Standards

- **单元测试覆盖率:** > 85%
- **集成测试:** 至少 2 个端到端场景
- **Temporal Workflow 测试:** 使用官方 testsuite
- **错误场景:** 覆盖所有求值失败情况

### References

- [ADR-0004: YAML DSL 语法](../adr/0004-yaml-dsl-syntax.md) - Step 结构定义
- [ADR-0005: 表达式系统语法](../adr/0005-expression-system-syntax.md) - 条件表达式语法
- [ADR-0002: 单节点执行模式](../adr/0002-single-node-execution-pattern.md) - Step = Activity
- [Story 1.6: 工作流执行引擎](1-6-basic-workflow-execution-engine.md) - Workflow 函数基础
- [Story 1.11: 变量系统](1-11-variable-system-implementation.md) - 表达式引擎基础
- [Story 1.12: 表达式求值引擎](1-12-expression-evaluation-engine.md) - 运算符和函数
- [GitHub Actions 条件语法](https://docs.github.com/en/actions/using-jobs/using-conditions-to-control-job-execution) - GHA 兼容性

## Dev Agent Record

### Context Reference

本故事已通过 BMad Method 终极上下文引擎创建,包含:
- ✅ Epic 1 完整上下文
- ✅ Story 1.6, 1.11, 1.12 前置依赖分析
- ✅ ADR-0002, 0004, 0005 架构设计
- ✅ Temporal Workflow 集成方案
- ✅ 完整的条件求值实现指南
- ✅ 运行时上下文扩展
- ✅ 测试策略和错误处理
- ✅ 与 Story 1.14 的集成准备

### Completion Notes

**Status:** ready-for-dev ✅

**创建时间:** 2025-12-17

**下一步:**
1. 开发者可直接开始实现
2. 主要工作在 Workflow 函数中添加条件检查逻辑
3. 复用 Story 1.11/1.12 的表达式引擎
4. 完成后为 Story 1.14 (Step 输出引用) 做好准备

**关键提醒:**
- 条件求值在 Temporal Workflow 函数内部
- 求值失败中止工作流,不是跳过 Step
- Skipped Steps 记录到 Temporal 历史
- 为 Story 1.14 预留 `executedSteps` 跟踪

### File List

待实现的文件清单:

**修改文件:**
- `pkg/dsl/workflow.go` - 添加 Step.If 字段
- `pkg/dsl/parser.go` - 验证 if 字段
- `pkg/expr/context.go` - 扩展运行时上下文
- `internal/workflow/workflow.go` - 添加条件检查逻辑

**新增文件:**
- `internal/workflow/conditional.go` - 条件求值函数
- `internal/workflow/conditional_test.go` - 单元测试
- `test/integration/conditional_execution_test.go` - 集成测试
