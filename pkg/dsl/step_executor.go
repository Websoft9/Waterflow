package dsl

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Common errors
var (
	ErrConditionEvalFailed = errors.New("condition evaluation failed")
	ErrStepExecutionFailed = errors.New("step execution failed")
)

// Default timeout for expression evaluation
const DefaultExpressionTimeout = 1 * time.Second

// StepResult represents the result of step execution
type StepResult struct {
	Status      string            // completed, skipped
	Conclusion  string            // success, failure, skipped
	Error       string            // error message if failed
	Outputs     map[string]string // step outputs
	ContinuedOn bool              // true if continued despite error
}

// StepExecutor executes workflow steps with if condition support
type StepExecutor struct {
	outputParser  *OutputParser
	condEvaluator *ConditionEvaluator
	outputManager *StepsOutputManager
}

// NewStepExecutor creates a new step executor
func NewStepExecutor() *StepExecutor {
	engine := NewEngine(DefaultExpressionTimeout)
	return &StepExecutor{
		outputParser:  NewOutputParser(),
		condEvaluator: NewConditionEvaluator(engine),
		outputManager: NewStepsOutputManager(),
	}
}

// Execute executes a step with if condition evaluation
func (e *StepExecutor) Execute(ctx context.Context, step *Step, evalCtx *EvalContext) (*StepResult, error) {
	// 1. Evaluate if condition
	if step.If != "" {
		shouldRun, err := e.condEvaluator.Evaluate(step.If, evalCtx)
		if err != nil {
			return nil, fmt.Errorf("evaluate step if condition: %w", err)
		}

		if !shouldRun {
			return &StepResult{
				Status:     "completed",
				Conclusion: "skipped",
			}, nil
		}
	}

	// 2. Execute step (mock execution for now - will integrate with node executor)
	// TODO: Integrate with actual node executor in future
	output := fmt.Sprintf("Mock execution of step: %s\n", step.Name)

	// Simulate step execution
	if step.Uses != "" {
		output += "::set-output name=status::success\n"
	}

	// 3. Parse outputs
	outputs := e.outputParser.ParseOutput(output)

	// 5. Store step outputs if step has ID
	if step.ID != "" {
		outputsInterface := make(map[string]interface{})
		for k, v := range outputs {
			outputsInterface[k] = v
		}
		e.outputManager.Update(step.ID, outputsInterface)

		// Update eval context
		evalCtx.Steps = e.outputManager.ToContext()
	}

	// 5. Return result
	conclusion := "success"

	return &StepResult{
		Status:     "completed",
		Conclusion: conclusion,
		Outputs:    outputs,
	}, nil
}

// GetOutputManager returns the output manager
func (e *StepExecutor) GetOutputManager() *StepsOutputManager {
	return e.outputManager
}
