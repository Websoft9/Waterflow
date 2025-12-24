package dsl

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestConditionalExecution_Integration tests complete if condition flow
func TestConditionalExecution_Integration(t *testing.T) {
	// Create workflow with pre-evaluated conditions (after expression replacement)
	workflow := &Workflow{
		Name: "Conditional Test",
		Vars: map[string]interface{}{
			"env":           "production",
			"should_deploy": true,
		},
		Jobs: map[string]*Job{
			"check": {
				Name: "check",
				Steps: []*Step{
					{
						Name: "Always Run",
						Uses: "echo@v1",
					},
					{
						Name: "Deploy to Production",
						Uses: "deploy@v1",
						If:   "vars.env == 'production'", // Pre-evaluated expression
					},
					{
						Name: "Deploy to Staging",
						Uses: "deploy@v1",
						If:   "vars.env == 'staging'",
					},
					{
						Name: "Conditional Deploy",
						Uses: "deploy@v1",
						If:   "vars.should_deploy == true",
					},
				},
			},
		},
	}

	// Build context
	ctx := NewContextBuilder(workflow).Build()

	// Execute steps with if conditions
	executor := NewStepExecutor()
	job := workflow.Jobs["check"]

	for _, step := range job.Steps {
		result, err := executor.Execute(context.Background(), step, ctx)
		require.NoError(t, err)

		// Verify conditional execution
		if step.Name == "Deploy to Production" {
			assert.Equal(t, "completed", result.Status)
			assert.Equal(t, "success", result.Conclusion) // Should execute (env == production)
		} else if step.Name == "Deploy to Staging" {
			assert.Equal(t, "completed", result.Status)
			assert.Equal(t, "skipped", result.Conclusion) // Should skip (env != staging)
		}
	}
}

// TestJobDependencies_Integration tests job orchestration with needs
func TestJobDependencies_Integration(t *testing.T) {
	// Load workflow
	yamlContent, err := os.ReadFile("../../testdata/workflows/dependencies.yaml")
	require.NoError(t, err)

	parser := NewParser(zap.NewNop())
	workflow, err := parser.Parse(yamlContent)
	require.NoError(t, err)

	// Create orchestrator with mock executor
	executor := &MockJobExecutor{}
	orch := NewJobOrchestrator(workflow, executor)

	// Execute workflow
	err = orch.Execute(context.Background(), workflow)
	require.NoError(t, err)

	// Verify execution order
	assert.Contains(t, executor.executedJobs[:2], "build")
	assert.Contains(t, executor.executedJobs[:2], "test")
	assert.Equal(t, "deploy", executor.executedJobs[2]) // Deploy should be last

	// Verify deploy job received build outputs
	deployResult := orch.GetResult("deploy")
	assert.NotNil(t, deployResult)
	assert.Equal(t, "success", deployResult.Conclusion)
}

// TestContinueOnError_Integration tests continue-on-error behavior
func TestContinueOnError_Integration(t *testing.T) {
	// Load workflow
	yamlContent, err := os.ReadFile("../../testdata/workflows/continue-on-error.yaml")
	require.NoError(t, err)

	parser := NewParser(zap.NewNop())
	workflow, err := parser.Parse(yamlContent)
	require.NoError(t, err)

	// Verify continue-on-error is set
	optionalJob := workflow.Jobs["optional_checks"]
	assert.True(t, optionalJob.ContinueOnError)

	optionalStep := optionalJob.Steps[0]
	assert.True(t, optionalStep.ContinueOnError)
}

// TestConditionFunctions_Integration tests success(), failure(), always()
func TestConditionFunctions_Integration(t *testing.T) {
	workflow := &Workflow{
		Name: "Condition Functions Test",
		Jobs: map[string]*Job{
			"notify": {
				Name: "notify",
				Steps: []*Step{
					{
						Name: "Notify Success",
						Uses: "notify@v1",
						If:   "success()", // Pre-evaluated expression
					},
					{
						Name: "Notify Failure",
						Uses: "notify@v1",
						If:   "failure()",
					},
					{
						Name: "Always Cleanup",
						Uses: "cleanup@v1",
						If:   "always()",
					},
				},
			},
		},
	}

	// Build context with job status = success
	ctx := NewContextBuilder(workflow).Build()
	ctx.UpdateJobStatus("success")

	executor := NewStepExecutor()
	job := workflow.Jobs["notify"]

	for _, step := range job.Steps {
		result, err := executor.Execute(context.Background(), step, ctx)
		require.NoError(t, err)

		if step.Name == "Notify Success" {
			assert.Equal(t, "success", result.Conclusion) // Should execute
		} else if step.Name == "Notify Failure" {
			assert.Equal(t, "skipped", result.Conclusion) // Should skip
		} else if step.Name == "Always Cleanup" {
			assert.Equal(t, "success", result.Conclusion) // Should always execute
		}
	}
}

// TestStepOutputsReferencing_Integration tests step output referencing
func TestStepOutputsReferencing_Integration(t *testing.T) {
	workflow := &Workflow{
		Name: "Step Outputs Test",
		Vars: make(map[string]interface{}),
		Jobs: map[string]*Job{
			"build": {
				Name: "build",
				Steps: []*Step{
					{
						ID:   "checkout",
						Name: "Checkout",
						Uses: "checkout@v1",
					},
					{
						ID:   "build",
						Name: "Build",
						Uses: "build@v1",
					},
				},
			},
		},
	}

	ctx := NewContextBuilder(workflow).Build()
	executor := NewStepExecutor()

	// Execute checkout step
	checkoutStep := workflow.Jobs["build"].Steps[0]
	_, err := executor.Execute(context.Background(), checkoutStep, ctx)
	require.NoError(t, err)

	// Verify step outputs are accessible
	outputManager := executor.GetOutputManager()
	assert.NotNil(t, outputManager)

	// Context should have steps.checkout.outputs
	stepsCtx := outputManager.ToContext()
	assert.Contains(t, stepsCtx, "checkout")
}

// TestJobOutputComputation_Integration tests job output expressions
func TestJobOutputComputation_Integration(t *testing.T) {
	workflow := &Workflow{
		Name: "Job Outputs Test",
		Jobs: map[string]*Job{
			"build": {
				Name: "build",
				Steps: []*Step{
					{
						ID:   "build_step",
						Name: "Build",
						Uses: "build@v1",
					},
				},
				Outputs: map[string]string{
					"version": "${{ steps.build_step.outputs.version }}",
					"commit":  "${{ steps.build_step.outputs.commit }}",
				},
			},
		},
	}

	// Create computer
	computer := NewJobOutputComputer()

	// Build context with step outputs
	ctx := NewContextBuilder(workflow).Build()
	outputManager := NewStepsOutputManager()
	outputManager.Set("build_step", map[string]interface{}{
		"version": "v1.2.3",
		"commit":  "abc123",
	})
	ctx.Steps = outputManager.ToContext()

	// Compute job outputs
	job := workflow.Jobs["build"]
	outputs, err := computer.Compute(job, ctx)
	require.NoError(t, err)

	assert.Equal(t, "v1.2.3", outputs["version"])
	assert.Equal(t, "abc123", outputs["commit"])
}
