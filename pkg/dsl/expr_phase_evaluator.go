package dsl

import (
	"time"
)

// PhaseEvaluator 分阶段表达式求值器
// 根据 ADR-0009，表达式求值分为两个阶段：
// 1. 定义解析阶段（工作流启动时）- 求值结构性字段
// 2. Step 执行阶段（每个 Step 执行前）- 求值运行时字段
type PhaseEvaluator struct {
	engine *Engine
}

// NewPhaseEvaluator 创建分阶段求值器
func NewPhaseEvaluator(timeout time.Duration) *PhaseEvaluator {
	return &PhaseEvaluator{
		engine: NewEngine(timeout),
	}
}

// BuildDefinitionContext 构建定义解析阶段上下文
// 此阶段仅包含 vars，不包含运行时变量（workflow, job, steps 等）
//
// 使用场景：
// - 求值 runs-on 表达式
// - 求值 timeout-minutes 表达式
// - 求值 strategy.matrix 表达式
func (e *PhaseEvaluator) BuildDefinitionContext(vars map[string]interface{}) *EvalContext {
	// 获取内置函数
	funcs := GetBuiltinFunctions()

	ctx := &EvalContext{
		Vars: vars,
		Env:  make(map[string]string),

		// 注册内置函数
		Len:        funcs["len"].(func(interface{}) (int, error)),
		Upper:      funcs["upper"].(func(string) string),
		Lower:      funcs["lower"].(func(string) string),
		Trim:       funcs["trim"].(func(string) string),
		Split:      funcs["split"].(func(string, string) []string),
		Join:       funcs["join"].(func([]string, string) string),
		Format:     funcs["format"].(func(string, ...interface{}) string),
		Contains:   funcs["contains"].(func(string, string) bool),
		StartsWith: funcs["startsWith"].(func(string, string) bool),
		EndsWith:   funcs["endsWith"].(func(string, string) bool),
		ToJSON:     funcs["toJSON"].(func(interface{}) (string, error)),
		FromJSON:   funcs["fromJSON"].(func(string) (interface{}, error)),
		Always:     funcs["always"].(func() bool),

		// 条件函数在定义阶段不可用（返回 false）
		Success:   func() bool { return false },
		Failure:   func() bool { return false },
		Cancelled: func() bool { return false },
	}

	// 不包含以下运行时上下文：
	// - Workflow: 工作流信息在执行时才确定
	// - Job: Job 状态在执行时才确定
	// - Steps: Step 输出在执行时才有
	// - Matrix: Matrix 变量在 Job 实例化时确定
	// - Runner: Runner 信息由 Agent 提供
	// - Inputs: 触发器输入在执行时提供
	// - Secrets: 密钥在执行时访问
	// - Needs: Job 依赖输出在执行时确定

	return ctx
}

// BuildStepContext 构建 Step 执行阶段上下文
// 此阶段包含完整上下文，可访问所有运行时变量
//
// 使用场景：
// - 求值 step.name 表达式
// - 求值 step.if 条件表达式
// - 求值 step.with 参数表达式
// - 求值 step.env 环境变量表达式
func (e *PhaseEvaluator) BuildStepContext(
	vars map[string]interface{},
	workflowInfo map[string]interface{},
	jobInfo map[string]interface{},
	stepOutputs map[string]interface{},
	matrixVars map[string]interface{},
	env map[string]string,
	runnerInfo map[string]interface{},
	inputs map[string]interface{},
	needsOutputs map[string]interface{},
) *EvalContext {
	jobStatus := "success" // 默认状态
	if jobInfo != nil {
		if status, ok := jobInfo["status"].(string); ok {
			jobStatus = status
		}
	}

	// 获取内置函数
	funcs := GetBuiltinFunctions()

	ctx := &EvalContext{
		Vars:     vars,
		Workflow: workflowInfo,
		Job:      jobInfo,
		Steps:    stepOutputs,
		Matrix:   matrixVars,
		Env:      env,
		Runner:   runnerInfo,
		Inputs:   inputs,
		Needs:    needsOutputs,
		Secrets:  make(map[string]string), // 静态密钥（向后兼容）

		// 注册内置函数
		Len:        funcs["len"].(func(interface{}) (int, error)),
		Upper:      funcs["upper"].(func(string) string),
		Lower:      funcs["lower"].(func(string) string),
		Trim:       funcs["trim"].(func(string) string),
		Split:      funcs["split"].(func(string, string) []string),
		Join:       funcs["join"].(func([]string, string) string),
		Format:     funcs["format"].(func(string, ...interface{}) string),
		Contains:   funcs["contains"].(func(string, string) bool),
		StartsWith: funcs["startsWith"].(func(string, string) bool),
		EndsWith:   funcs["endsWith"].(func(string, string) bool),
		ToJSON:     funcs["toJSON"].(func(interface{}) (string, error)),
		FromJSON:   funcs["fromJSON"].(func(string) (interface{}, error)),
		Always:     funcs["always"].(func() bool),

		// 条件函数（依赖 Job 状态）
		Success:   MakeSuccessFunc(jobStatus),
		Failure:   MakeFailureFunc(jobStatus),
		Cancelled: MakeCancelledFunc(jobStatus),
	}

	return ctx
}

// Evaluate 求值表达式（使用内部引擎）
func (e *PhaseEvaluator) Evaluate(expression string, ctx *EvalContext) (interface{}, error) {
	return e.engine.Evaluate(expression, ctx)
}

// GetEngine 获取底层表达式引擎
func (e *PhaseEvaluator) GetEngine() *Engine {
	return e.engine
}
