package dsl

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStepExecutor_IfCondition_Skip(t *testing.T) {
	executor := NewStepExecutor()

	workflow := &Workflow{
		Name: "Test",
		Vars: map[string]interface{}{
			"env": "staging",
		},
	}

	step := &Step{
		Name: "Deploy to Production",
		Uses: "deploy@v1",
		If:   "vars.env == 'production'",
	}

	evalCtx := NewContextBuilder(workflow).Build()
	result, err := executor.Execute(context.Background(), step, evalCtx)

	assert.NoError(t, err)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, "skipped", result.Conclusion)
}

func TestStepExecutor_IfCondition_Execute(t *testing.T) {
	executor := NewStepExecutor()

	workflow := &Workflow{
		Name: "Test",
		Vars: map[string]interface{}{
			"env": "production",
		},
	}

	step := &Step{
		Name: "Deploy to Production",
		Uses: "deploy@v1",
		If:   "vars.env == 'production'",
	}

	evalCtx := NewContextBuilder(workflow).Build()
	result, err := executor.Execute(context.Background(), step, evalCtx)

	assert.NoError(t, err)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, "success", result.Conclusion)
}

func TestStepExecutor_NoIfCondition(t *testing.T) {
	executor := NewStepExecutor()

	workflow := &Workflow{
		Name: "Test",
	}

	step := &Step{
		Name: "Build",
		Uses: "build@v1",
	}

	evalCtx := NewContextBuilder(workflow).Build()
	result, err := executor.Execute(context.Background(), step, evalCtx)

	assert.NoError(t, err)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, "success", result.Conclusion)
}

func TestStepExecutor_OutputParsing(t *testing.T) {
	executor := NewStepExecutor()

	workflow := &Workflow{
		Name: "Test",
	}

	step := &Step{
		ID:   "build_step",
		Name: "Build with output",
		Uses: "build@v1",
	}

	evalCtx := NewContextBuilder(workflow).Build()
	result, err := executor.Execute(context.Background(), step, evalCtx)

	assert.NoError(t, err)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, "success", result.Conclusion)
	assert.Contains(t, result.Outputs, "status")
	assert.Equal(t, "success", result.Outputs["status"])
}

func TestStepExecutor_ContextUpdate(t *testing.T) {
	executor := NewStepExecutor()

	workflow := &Workflow{
		Name: "Test",
	}

	step := &Step{
		ID:   "setup",
		Name: "Setup environment",
		Uses: "setup@v1",
	}

	evalCtx := NewContextBuilder(workflow).Build()
	result, err := executor.Execute(context.Background(), step, evalCtx)

	assert.NoError(t, err)
	assert.Equal(t, "success", result.Conclusion)

	// Verify context was updated
	outputs := executor.GetOutputManager().ToContext()
	assert.Contains(t, outputs, "setup")
}

func TestStepExecutor_AlwaysFunction(t *testing.T) {
	executor := NewStepExecutor()

	workflow := &Workflow{
		Name: "Test",
	}

	step := &Step{
		Name: "Cleanup",
		Uses: "cleanup@v1",
		If:   "always()",
	}

	evalCtx := NewContextBuilder(workflow).Build()
	result, err := executor.Execute(context.Background(), step, evalCtx)

	assert.NoError(t, err)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, "success", result.Conclusion)
}
