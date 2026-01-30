package temporal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/Websoft9/waterflow/pkg/metrics"
	"go.temporal.io/sdk/activity"
	temporal "go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
)

// Activities holds all workflow activities.
type Activities struct {
	logger       *zap.Logger
	nodeRegistry *node.Registry
	nodeTracker  *metrics.NodeTracker
}

// NewActivities creates a new Activities instance.
func NewActivities(logger *zap.Logger, nodeRegistry *node.Registry) *Activities {
	return &Activities{
		logger:       logger,
		nodeRegistry: nodeRegistry,
		nodeTracker:  metrics.NewNodeTracker(),
	}
}

// ExecuteStepInput is the input parameter for ExecuteStepActivity.
// Uses only primitive and serializable types to ensure Temporal compatibility.
type ExecuteStepInput struct {
	WorkflowName string
	JobName      string
	StepName     string
	StepUses     string
	StepWith     map[string]interface{}
	StepEnv      map[string]string
	StepIf       string
	Context      *dsl.SerializableEvalContext // 使用可序列化版本
}

// StepResult is the result returned by ExecuteStepActivity.
type StepResult struct {
	Status     string            // success, failure, skipped, timeout
	Outputs    map[string]string // Step outputs
	Error      string            // Error message (if failed)
	DurationMs int64             // Execution duration in milliseconds
}

// ExecuteStepActivity executes a single workflow step.
// It evaluates if conditions, renders expressions, and executes the node.
func (a *Activities) ExecuteStepActivity(ctx context.Context, input ExecuteStepInput) (*StepResult, error) {
	logger := activity.GetLogger(ctx)

	// 获取当前 Attempt 信息 (Story 4.3)
	info := activity.GetInfo(ctx)
	logger.Info("Executing step",
		"name", input.StepName,
		"uses", input.StepUses,
		"attempt", info.Attempt, // 当前尝试次数 (1-based)
	)

	startTime := time.Now()

	// Reconstruct EvalContext with functions from serializable version
	evalCtx := input.Context.ToEvalContext()

	// Reconstruct Step from serialized data
	step := &dsl.Step{
		Name: input.StepName,
		Uses: input.StepUses,
		With: input.StepWith,
		Env:  input.StepEnv,
		If:   input.StepIf,
	}

	// Reconstruct minimal Workflow and Job for rendering context
	workflow := &dsl.Workflow{
		Name: input.WorkflowName,
	}
	job := &dsl.Job{
		Name: input.JobName,
	}

	// 1. Check if condition (using Story 1.5 ConditionEvaluator)
	if step.If != "" {
		engine := dsl.NewEngine(5 * time.Second)
		condEval := dsl.NewConditionEvaluator(engine)
		shouldRun, err := condEval.Evaluate(step.If, evalCtx)
		if err != nil {
			// 条件求值错误 - 永久性错误
			return nil, temporal.NewNonRetryableApplicationError(
				fmt.Sprintf("failed to evaluate if condition: %v", err),
				"validation_error",
				err,
			)
		}

		if !shouldRun {
			logger.Info("Step skipped due to if condition", "name", step.Name)
			return &StepResult{
				Status:     "skipped",
				DurationMs: time.Since(startTime).Milliseconds(),
			}, nil
		}
	}

	// 2. Render step (replace expressions - using Story 1.4 WorkflowRenderer)
	renderer := dsl.NewWorkflowRenderer()
	renderedStep, err := renderer.RenderStep(workflow, job, step, evalCtx)
	if err != nil {
		// 表达式渲染错误 - 永久性错误
		return nil, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("failed to render step: %v", err),
			"validation_error",
			err,
		)
	}

	// 2.5. Validate parameters (Story 4.2 - AC4)
	if a.nodeRegistry != nil {
		nodeInstance, err := a.nodeRegistry.Get(renderedStep.Uses)
		if err != nil {
			// Node not found - 永久性错误
			logger.Error("Node not found", "uses", renderedStep.Uses, "error", err)
			return nil, temporal.NewNonRetryableApplicationError(
				fmt.Sprintf("node not found: %s", renderedStep.Uses),
				"node_not_registered",
				err,
			)
		}

		// Validate inputs against node's parameter specifications
		if err := node.ValidateInputs(renderedStep.With, nodeInstance.Params()); err != nil {
			// Set NodeName in validation error for better debugging
			if validationErr, ok := err.(*node.InputValidationError); ok {
				validationErr.NodeName = renderedStep.Uses
			}
			logger.Error("Parameter validation failed",
				"step", step.Name,
				"uses", renderedStep.Uses,
				"error", err)

			// 参数验证错误 - 永久性错误
			return nil, temporal.NewNonRetryableApplicationError(
				err.Error(),
				"validation_error",
				err,
			)
		}

		logger.Info("Parameter validation passed", "step", step.Name, "uses", renderedStep.Uses)

		// 3. Execute node (Story 4.3 - 真正执行节点)
		// Track node execution start
		done := a.nodeTracker.TrackNodeStart(renderedStep.Uses)
		nodeResult, err := nodeInstance.Execute(ctx, renderedStep.With)
		// Track completion
		done(err == nil)

		if err != nil {
			// 检查是否为 NonRetryableError (Story 4.3)
			var nonRetryable *node.NonRetryableError
			if errors.As(err, &nonRetryable) {
				logger.Warn("Non-retryable error detected",
					"step", step.Name,
					"error_type", nonRetryable.ErrorType,
					"message", nonRetryable.Message,
					"attempt", info.Attempt,
					"retryable", false,
				)

				// 返回 Temporal ApplicationError,标记为不可重试
				return nil, temporal.NewNonRetryableApplicationError(
					nonRetryable.Message,
					nonRetryable.ErrorType,
					err,
				)
			}

			// 其他错误 - 可重试 (网络错误、临时故障等)
			logger.Warn("Step failed, will retry according to policy",
				"step", step.Name,
				"error", err,
				"attempt", info.Attempt,
				"retryable", true,
			)
			return nil, err
		}

		// 4. 返回成功结果
		duration := time.Since(startTime)

		// 成功时记录重试信息 (如果经过重试)
		if info.Attempt > 1 {
			logger.Info("Step succeeded after retry",
				"name", step.Name,
				"attempt", info.Attempt,
				"duration_ms", duration.Milliseconds(),
			)
		} else {
			logger.Info("Step completed",
				"name", step.Name,
				"duration_ms", duration.Milliseconds(),
			)
		}

		// 转换 NodeResult.Outputs 为 map[string]string
		outputs := make(map[string]string)
		for k, v := range nodeResult.Outputs {
			outputs[k] = fmt.Sprintf("%v", v)
		}

		// Record heartbeat with detailed progress information
		activity.RecordHeartbeat(ctx, map[string]interface{}{
			"step":         step.Name,
			"uses":         step.Uses,
			"progress":     "completed",
			"duration_ms":  duration.Milliseconds(),
			"attempt":      info.Attempt,
			"outputs_size": len(outputs),
		})

		return &StepResult{
			Status:     "success",
			Outputs:    outputs,
			DurationMs: duration.Milliseconds(),
		}, nil
	}

	// Fallback: no node registry (测试模式)
	// TODO(Story 4.1): Replace with actual NodeRegistry integration
	// This placeholder is only used for testing when NodeRegistry is not available.
	// In production, nodeRegistry should always be initialized.
	logger.Warn("Node registry not available, using placeholder execution")

	// Simulate execution
	outputs := make(map[string]string)
	outputs["result"] = "success"

	// Record heartbeat with detailed progress information
	activity.RecordHeartbeat(ctx, map[string]interface{}{
		"step":        step.Name,
		"progress":    "completed",
		"duration_ms": time.Since(startTime).Milliseconds(),
		"placeholder": true, // Indicates this is a placeholder execution
	})

	duration := time.Since(startTime)

	logger.Info("Step completed", "name", step.Name, "duration_ms", duration.Milliseconds())

	return &StepResult{
		Status:     "success",
		Outputs:    outputs,
		DurationMs: duration.Milliseconds(),
	}, nil
}
