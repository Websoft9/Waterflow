package dsl

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockStepExecutor extends StepExecutor for testing error scenarios
type MockStepExecutor struct {
	*StepExecutor
	shouldFail bool
}

func NewMockStepExecutor(shouldFail bool) *MockStepExecutor {
	return &MockStepExecutor{
		StepExecutor: NewStepExecutor(),
		shouldFail:   shouldFail,
	}
}

func (e *MockStepExecutor) Execute(ctx context.Context, step *Step, evalCtx *EvalContext) (*StepResult, error) {
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

	// 2. Execute step with mock error injection
	output := fmt.Sprintf("Mock execution of step: %s\n", step.Name)
	var executionError error

	if e.shouldFail {
		executionError = fmt.Errorf("mock step failure")
	} else if step.Uses != "" {
		output += "::set-output name=status::success\n"
	}

	// 3. Handle execution error with continue-on-error
	if executionError != nil {
		if step.ContinueOnError {
			// Continue despite error
			outputs := make(map[string]string)
			return &StepResult{
				Status:      "completed",
				Conclusion:  "failure",
				Error:       executionError.Error(),
				Outputs:     outputs,
				ContinuedOn: true,
			}, nil
		}
		// Fail the step
		return nil, executionError
	}

	// 4. Parse outputs
	parser := NewOutputParser()
	outputs := parser.ParseOutput(output)

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

	// 6. Return result
	conclusion := "success"

	return &StepResult{
		Status:     "completed",
		Conclusion: conclusion,
		Outputs:    outputs,
	}, nil
}

func TestContinueOnError_ContinuesOnFailure(t *testing.T) {
	executor := NewMockStepExecutor(true) // inject failure

	workflow := &Workflow{
		Name: "Test",
	}

	step := &Step{
		Name:            "Flaky test",
		Uses:            "test@v1",
		ContinueOnError: true,
	}

	evalCtx := NewContextBuilder(workflow).Build()
	result, err := executor.Execute(context.Background(), step, evalCtx)

	assert.NoError(t, err)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, "failure", result.Conclusion)
	assert.Equal(t, "mock step failure", result.Error)
	assert.True(t, result.ContinuedOn)
}

func TestContinueOnError_FailsWithoutFlag(t *testing.T) {
	executor := NewMockStepExecutor(true) // inject failure

	workflow := &Workflow{
		Name: "Test",
	}

	step := &Step{
		Name:            "Critical step",
		Uses:            "deploy@v1",
		ContinueOnError: false,
	}

	evalCtx := NewContextBuilder(workflow).Build()
	result, err := executor.Execute(context.Background(), step, evalCtx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "mock step failure")
}

func TestContinueOnError_SuccessCase(t *testing.T) {
	executor := NewMockStepExecutor(false) // no failure

	workflow := &Workflow{
		Name: "Test",
	}

	step := &Step{
		Name:            "Normal step",
		Uses:            "build@v1",
		ContinueOnError: true, // flag is set but not needed
	}

	evalCtx := NewContextBuilder(workflow).Build()
	result, err := executor.Execute(context.Background(), step, evalCtx)

	assert.NoError(t, err)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, "success", result.Conclusion)
	assert.False(t, result.ContinuedOn)
}
