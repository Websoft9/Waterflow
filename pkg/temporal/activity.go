package temporal

import (
	"context"
	"fmt"
	"time"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"go.temporal.io/sdk/activity"
	"go.uber.org/zap"
)

// Activities holds all workflow activities.
type Activities struct {
	logger *zap.Logger
}

// NewActivities creates a new Activities instance.
func NewActivities(logger *zap.Logger) *Activities {
	return &Activities{
		logger: logger,
	}
}

// ExecuteStepInput is the input parameter for ExecuteStepActivity.
type ExecuteStepInput struct {
	Workflow *dsl.Workflow
	Job      *dsl.Job
	Step     *dsl.Step
	Context  *dsl.EvalContext
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
	logger.Info("Executing step", "name", input.Step.Name, "uses", input.Step.Uses)

	startTime := time.Now()

	// 1. Check if condition (using Story 1.5 ConditionEvaluator)
	if input.Step.If != "" {
		engine := dsl.NewEngine(5 * time.Second)
		condEval := dsl.NewConditionEvaluator(engine)
		shouldRun, err := condEval.Evaluate(input.Step.If, input.Context)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate if condition: %w", err)
		}

		if !shouldRun {
			logger.Info("Step skipped due to if condition", "name", input.Step.Name)
			return &StepResult{
				Status:     "skipped",
				DurationMs: time.Since(startTime).Milliseconds(),
			}, nil
		}
	}

	// 2. Render step (replace expressions - using Story 1.4 WorkflowRenderer)
	renderer := dsl.NewWorkflowRenderer()
	renderedStep, err := renderer.RenderStep(input.Workflow, input.Job, input.Step, input.Context)
	if err != nil {
		return nil, fmt.Errorf("failed to render step: %w", err)
	}

	// 3. Execute node
	// TODO(Story 1.1): Integrate with NodeExecutor when Story 1.1 is completed
	// Current implementation is a placeholder for testing Temporal integration.
	// Expected integration:
	//   nodeExecutor := executor.NewNodeExecutor(a.nodeRegistry)
	//   nodeResult, err := nodeExecutor.Execute(ctx, renderedStep)
	//   if err != nil { return error }
	//   outputs = nodeResult.Outputs
	logger.Info("Node execution placeholder (awaiting Story 1.1 NodeExecutor)", "uses", renderedStep.Uses)

	// Simulate execution
	outputs := make(map[string]string)
	outputs["result"] = "success"

	// Record heartbeat (in production, this should be called periodically during long operations)
	activity.RecordHeartbeat(ctx, map[string]interface{}{
		"step":     input.Step.Name,
		"progress": "completed",
	})

	duration := time.Since(startTime)

	logger.Info("Step completed", "name", input.Step.Name, "duration_ms", duration.Milliseconds())

	return &StepResult{
		Status:     "success",
		Outputs:    outputs,
		DurationMs: duration.Milliseconds(),
	}, nil
}
