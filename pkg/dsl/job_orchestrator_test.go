package dsl

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockJobExecutor for testing
type MockJobExecutor struct {
	executedJobs []string
}

func (m *MockJobExecutor) Execute(ctx context.Context, job *Job, evalCtx *EvalContext) (*JobResult, error) {
	m.executedJobs = append(m.executedJobs, job.Name)

	// Compute outputs based on job.Outputs expressions
	outputs := make(map[string]string)
	if len(job.Outputs) > 0 {
		engine := NewEngine(1 * time.Second)
		for name, expr := range job.Outputs {
			// Evaluate output expression
			value, err := engine.Evaluate(expr, evalCtx)
			if err != nil {
				outputs[name] = fmt.Sprintf("error: %v", err)
			} else {
				outputs[name] = fmt.Sprintf("%v", value)
			}
		}
	}

	return &JobResult{
		JobID:      job.Name,
		Status:     "completed",
		Conclusion: "success",
		Outputs:    outputs,
	}, nil
}

func TestJobOrchestrator_SimpleExecution(t *testing.T) {
	workflow := &Workflow{
		Name: "Test workflow",
		Jobs: map[string]*Job{
			"build": {
				Name: "build",
				Steps: []*Step{
					{Name: "Build app"},
				},
			},
		},
	}

	executor := &MockJobExecutor{}
	orch := NewJobOrchestrator(workflow, executor)

	err := orch.Execute(context.Background(), workflow)
	assert.NoError(t, err)

	assert.Equal(t, []string{"build"}, executor.executedJobs)
	result := orch.GetResult("build")
	assert.NotNil(t, result)
	assert.Equal(t, "success", result.Conclusion)
}

func TestJobOrchestrator_WithDependencies(t *testing.T) {
	workflow := &Workflow{
		Name: "Test workflow",
		Jobs: map[string]*Job{
			"build": {
				Name: "build",
				Steps: []*Step{
					{Name: "Build app"},
				},
				Outputs: map[string]string{
					"version": "'1.0.0'",
				},
			},
			"test": {
				Name:  "test",
				Needs: []string{"build"},
				Steps: []*Step{
					{Name: "Run tests"},
				},
			},
			"deploy": {
				Name:  "deploy",
				Needs: []string{"test"},
				Steps: []*Step{
					{Name: "Deploy app"},
				},
			},
		},
	}

	executor := &MockJobExecutor{}
	orch := NewJobOrchestrator(workflow, executor)

	err := orch.Execute(context.Background(), workflow)
	assert.NoError(t, err)

	assert.Len(t, executor.executedJobs, 3)
	// Build should be first
	assert.Equal(t, "build", executor.executedJobs[0])
	// Test should be second
	assert.Equal(t, "test", executor.executedJobs[1])
	// Deploy should be last
	assert.Equal(t, "deploy", executor.executedJobs[2])
}

func TestJobOrchestrator_JobOutputs(t *testing.T) {
	workflow := &Workflow{
		Name: "Test workflow",
		Vars: map[string]interface{}{
			"app_name": "myapp",
		},
		Jobs: map[string]*Job{
			"build": {
				Name: "build",
				Steps: []*Step{
					{Name: "Build app"},
				},
				Outputs: map[string]string{
					"version": "'1.2.3'",
					"image":   "vars.app_name + ':latest'",
				},
			},
			"deploy": {
				Name:  "deploy",
				Needs: []string{"build"},
				Steps: []*Step{
					{Name: "Deploy app"},
				},
			},
		},
	}

	executor := &MockJobExecutor{}
	orch := NewJobOrchestrator(workflow, executor)

	err := orch.Execute(context.Background(), workflow)
	assert.NoError(t, err)

	buildResult := orch.GetResult("build")
	assert.NotNil(t, buildResult)
	assert.Equal(t, "1.2.3", buildResult.Outputs["version"])
	assert.Equal(t, "myapp:latest", buildResult.Outputs["image"])
}

func TestJobOrchestrator_IfConditionSkip(t *testing.T) {
	workflow := &Workflow{
		Name: "Test workflow",
		Vars: map[string]interface{}{
			"env": "development",
		},
		Jobs: map[string]*Job{
			"build": {
				Name: "build",
				Steps: []*Step{
					{Name: "Build app"},
				},
			},
			"deploy": {
				Name:  "deploy",
				Needs: []string{"build"},
				If:    "vars.env == 'production'",
				Steps: []*Step{
					{Name: "Deploy to production"},
				},
			},
		},
	}

	executor := &MockJobExecutor{}
	orch := NewJobOrchestrator(workflow, executor)

	err := orch.Execute(context.Background(), workflow)
	assert.NoError(t, err)

	// Only build should execute
	assert.Equal(t, []string{"build"}, executor.executedJobs)

	deployResult := orch.GetResult("deploy")
	assert.NotNil(t, deployResult)
	assert.Equal(t, "skipped", deployResult.Conclusion)
}

func TestJobOrchestrator_NeedsOutputsInContext(t *testing.T) {
	workflow := &Workflow{
		Name: "Test workflow",
		Jobs: map[string]*Job{
			"build": {
				Name: "build",
				Steps: []*Step{
					{Name: "Build app"},
				},
				Outputs: map[string]string{
					"version": "'2.0.0'",
				},
			},
			"deploy": {
				Name:  "deploy",
				Needs: []string{"build"},
				Steps: []*Step{
					{Name: "Deploy app"},
				},
				Outputs: map[string]string{
					"deployed_version": "needs.build.outputs.version",
				},
			},
		},
	}

	executor := &MockJobExecutor{}
	orch := NewJobOrchestrator(workflow, executor)

	err := orch.Execute(context.Background(), workflow)
	assert.NoError(t, err)

	deployResult := orch.GetResult("deploy")
	assert.NotNil(t, deployResult)
	assert.Equal(t, "2.0.0", deployResult.Outputs["deployed_version"])
}
