package dsl

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DefinitionRenderer 定义阶段渲染器
// 根据 ADR-0009，在工作流启动时求值结构性字段：
// - runs-on: Job 路由到哪个 Task Queue
// - timeout-minutes: Job/Step 超时配置
// - strategy.matrix: Matrix 并行维度
type DefinitionRenderer struct {
	phaseEvaluator *PhaseEvaluator
	replacer       *ExpressionReplacer
	varsMerger     *VarsMerger
}

// NewDefinitionRenderer 创建定义阶段渲染器
func NewDefinitionRenderer() *DefinitionRenderer {
	engine := NewEngine(1 * time.Second)
	phaseEvaluator := NewPhaseEvaluator(1 * time.Second)

	return &DefinitionRenderer{
		phaseEvaluator: phaseEvaluator,
		replacer:       NewExpressionReplacer(engine),
		varsMerger:     NewVarsMerger(),
	}
}

// RenderDefinitionPhase 执行定义解析阶段渲染
// 求值所有结构性字段的表达式，确保执行计划在运行前确定
//
// 参数:
//   - workflow: 原始工作流定义
//   - triggerVars: 触发器绑定的参数 (Schedule/Webhook)
//   - executionVars: 执行时传入的参数 (API 调用)
//
// 返回:
//   - 渲染后的工作流定义（结构性字段已求值）
func (r *DefinitionRenderer) RenderDefinitionPhase(
	workflow *Workflow,
	triggerVars map[string]interface{},
	executionVars map[string]interface{},
) (*Workflow, error) {
	// 1. 三层参数合并
	mergedVars := r.varsMerger.Merge(workflow.Vars, triggerVars, executionVars)

	// 2. 构建定义阶段上下文（仅包含 vars）
	ctx := r.phaseEvaluator.BuildDefinitionContext(mergedVars)

	// 3. 遍历所有 Job，求值结构性字段
	for jobName, job := range workflow.Jobs {
		if err := r.renderJobDefinitionFields(job, ctx); err != nil {
			return nil, fmt.Errorf("render job '%s' definition fields: %w", jobName, err)
		}
	}

	// 4. 更新工作流 vars 为合并后的值
	workflow.Vars = mergedVars

	return workflow, nil
}

// renderJobDefinitionFields 渲染 Job 的结构性字段
func (r *DefinitionRenderer) renderJobDefinitionFields(job *Job, ctx *EvalContext) error {
	// 1. 求值 runs-on 表达式
	if err := r.renderRunsOn(job, ctx); err != nil {
		return fmt.Errorf("render runs-on: %w", err)
	}

	// 2. 求值 timeout-minutes 表达式
	if err := r.renderTimeout(job, ctx); err != nil {
		return fmt.Errorf("render timeout-minutes: %w", err)
	}

	// 3. 求值 strategy.matrix 表达式
	if job.Strategy != nil {
		if err := r.renderMatrix(job.Strategy, ctx); err != nil {
			return fmt.Errorf("render strategy.matrix: %w", err)
		}
	}

	return nil
}

// renderRunsOn 求值 runs-on 字段
func (r *DefinitionRenderer) renderRunsOn(job *Job, ctx *EvalContext) error {
	if job.RunsOn == "" {
		return nil
	}

	// 检查是否包含表达式
	if !strings.Contains(job.RunsOn, "${{") {
		return nil // 非表达式，无需求值
	}

	// 保存原始表达式
	job.RunsOnExpr = job.RunsOn

	// 求值表达式
	rendered, err := r.replacer.Replace(job.RunsOn, ctx)
	if err != nil {
		return fmt.Errorf("evaluate runs-on expression '%s': %w", job.RunsOn, err)
	}

	// 更新字段
	job.RunsOn = rendered
	return nil
}

// renderTimeout 求值 timeout-minutes 字段
// 支持两种格式：
// 1. 直接整数: timeout-minutes: 30
// 2. 表达式: timeout-minutes: ${{ vars.timeout }}
func (r *DefinitionRenderer) renderTimeout(job *Job, ctx *EvalContext) error {
	// timeout-minutes 在 YAML 解析时已转为 int
	// 如果需要支持表达式，需要在 Parser 中特殊处理

	// 临时实现：检查 Job 是否有 timeout 相关的自定义字段
	// 完整实现需要修改 Parser 以保留原始表达式

	return nil // MVP: 暂不支持 timeout-minutes 表达式
}

// renderMatrix 求值 strategy.matrix 字段
func (r *DefinitionRenderer) renderMatrix(strategy *Strategy, ctx *EvalContext) error {
	if strategy.Matrix == nil {
		return nil
	}

	renderedMatrix := make(map[string][]interface{})

	for key, values := range strategy.Matrix {
		renderedValues := make([]interface{}, 0, len(values))

		for _, value := range values {
			// 检查值是否为字符串表达式
			if strValue, ok := value.(string); ok && strings.Contains(strValue, "${{") {
				// 保存原始表达式
				if strategy.MatrixExpr == nil {
					strategy.MatrixExpr = make(map[string]string)
				}
				strategy.MatrixExpr[key] = strValue

				// 表达式可能返回数组或单个值
				// 注意: replacer.Replace 返回 string，需要实际求值
				evaluated, err := r.phaseEvaluator.Evaluate(strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strValue, "${{"), "}}")), ctx)
				if err != nil {
					return fmt.Errorf("evaluate matrix.%s expression '%s': %w", key, strValue, err)
				}

				if arrayResult, ok := evaluated.([]interface{}); ok {
					renderedValues = append(renderedValues, arrayResult...)
				} else {
					renderedValues = append(renderedValues, evaluated)
				}
			} else {
				// 非表达式，直接使用原值
				renderedValues = append(renderedValues, value)
			}
		}

		renderedMatrix[key] = renderedValues
	}

	// 更新 Matrix
	strategy.Matrix = renderedMatrix
	return nil
}

// EvaluateMatrixExpression 求值 Matrix 数组表达式
// 示例: ${{ vars.servers }} → ["web1", "web2"]
func (r *DefinitionRenderer) EvaluateMatrixExpression(expression string, ctx *EvalContext) ([]interface{}, error) {
	// 去掉 ${{ 和 }}
	expr := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(expression, "${{"), "}}"))

	// 求值
	result, err := r.phaseEvaluator.Evaluate(expr, ctx)
	if err != nil {
		return nil, err
	}

	// 转换为数组
	switch v := result.(type) {
	case []interface{}:
		return v, nil
	case []string:
		arr := make([]interface{}, len(v))
		for i, s := range v {
			arr[i] = s
		}
		return arr, nil
	default:
		// 单个值包装为数组
		return []interface{}{v}, nil
	}
}

// ParseTimeout 解析超时字符串（支持 duration 格式）
// 示例: "30m", "1h", "90s"
func ParseTimeout(timeout string) (int, error) {
	duration, err := time.ParseDuration(timeout)
	if err != nil {
		// 尝试解析为纯数字（分钟）
		minutes, err := strconv.Atoi(timeout)
		if err != nil {
			return 0, fmt.Errorf("invalid timeout format '%s': %w", timeout, err)
		}
		return minutes, nil
	}

	return int(duration.Minutes()), nil
}
