package dsl_test

import (
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowRenderer_RenderWorkflow(t *testing.T) {
	renderer := dsl.NewWorkflowRenderer()

	workflow := &dsl.Workflow{
		Name: "Test Workflow",
		On:   "push",
		Vars: map[string]interface{}{
			"app":     "myapp",
			"version": "v1.2.3",
			"env":     "production",
		},
		Env: map[string]string{
			"APP_NAME":    "${{ vars.app }}",
			"APP_VERSION": "${{ vars.version }}",
			"LOG_LEVEL":   "info",
		},
		Jobs: map[string]*dsl.Job{
			"build": {
				Name:   "build",
				RunsOn: "ubuntu-latest",
				Env: map[string]string{
					"LOG_LEVEL": "debug", // overrides workflow
					"BUILD_ENV": "${{ vars.env }}",
				},
				Steps: []*dsl.Step{
					{
						Name: "Checkout",
						Uses: "checkout@v1",
						With: map[string]interface{}{
							"repository": "${{ vars.app }}",
						},
					},
					{
						Name: "Build",
						Uses: "run@v1",
						With: map[string]interface{}{
							"command": "echo Building ${{ vars.app }} v${{ vars.version }}",
						},
						Env: map[string]string{
							"BUILD_ENV": "staging", // overrides job
						},
					},
				},
			},
		},
	}

	rendered, err := renderer.RenderWorkflow(workflow)
	require.NoError(t, err)
	require.NotNil(t, rendered)

	// Check workflow-level env rendering
	assert.Equal(t, "myapp", rendered.Env["APP_NAME"])
	assert.Equal(t, "v1.2.3", rendered.Env["APP_VERSION"])
	assert.Equal(t, "info", rendered.Env["LOG_LEVEL"])

	// Check job exists
	buildJob, exists := rendered.Jobs["build"]
	require.True(t, exists)

	// Check job-level env rendering and merging
	assert.Equal(t, "debug", buildJob.Env["LOG_LEVEL"])      // job overrides workflow
	assert.Equal(t, "production", buildJob.Env["BUILD_ENV"]) // rendered expression

	// Check steps
	require.Len(t, buildJob.Steps, 2)

	// Check first step
	checkoutStep := buildJob.Steps[0]
	assert.Equal(t, "Checkout", checkoutStep.Name)
	assert.Equal(t, "myapp", checkoutStep.With["repository"])

	// Check second step
	buildStep := buildJob.Steps[1]
	assert.Equal(t, "Build", buildStep.Name)
	assert.Equal(t, "echo Building myapp vv1.2.3", buildStep.With["command"])
	assert.Equal(t, "staging", buildStep.Env["BUILD_ENV"]) // step overrides job
}

func TestWorkflowRenderer_RenderStep_IfCondition(t *testing.T) {
	renderer := dsl.NewWorkflowRenderer()

	workflow := &dsl.Workflow{
		Name: "Test",
		Vars: map[string]interface{}{
			"env": "production",
		},
	}

	job := &dsl.Job{
		Name:   "deploy",
		RunsOn: "ubuntu-latest",
	}

	tests := []struct {
		name         string
		step         *dsl.Step
		shouldRender bool
		description  string
	}{
		{
			name: "step with true condition",
			step: &dsl.Step{
				Name: "Deploy",
				Uses: "deploy@v1",
				If:   `vars.env == "production"`,
			},
			shouldRender: true,
			description:  "should render when condition is true",
		},
		{
			name: "step with false condition",
			step: &dsl.Step{
				Name: "Deploy",
				Uses: "deploy@v1",
				If:   `vars.env == "staging"`,
			},
			shouldRender: false,
			description:  "should skip when condition is false",
		},
		{
			name: "step without condition",
			step: &dsl.Step{
				Name: "Always Run",
				Uses: "run@v1",
			},
			shouldRender: true,
			description:  "should render when no condition",
		},
		{
			name: "step with always function",
			step: &dsl.Step{
				Name: "Cleanup",
				Uses: "cleanup@v1",
				If:   `always()`,
			},
			shouldRender: true,
			description:  "should render with always() function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := dsl.NewContextBuilder(workflow).WithJob(job).Build()
			rendered, err := renderer.RenderStep(workflow, job, tt.step, ctx)
			require.NoError(t, err)

			if tt.shouldRender {
				assert.NotNil(t, rendered, tt.description)
			} else {
				assert.Nil(t, rendered, tt.description)
			}
		})
	}
}

func TestWorkflowRenderer_EnvMerging(t *testing.T) {
	renderer := dsl.NewWorkflowRenderer()

	workflow := &dsl.Workflow{
		Name: "Test",
		Vars: map[string]interface{}{
			"version": "v1.0.0",
		},
		Env: map[string]string{
			"LEVEL_1": "workflow",
			"LEVEL_2": "workflow",
			"LEVEL_3": "workflow",
			"VERSION": "${{ vars.version }}",
		},
	}

	job := &dsl.Job{
		Name:   "test",
		RunsOn: "ubuntu-latest",
		Env: map[string]string{
			"LEVEL_2": "job", // overrides workflow
			"LEVEL_3": "job",
			"JOB_VAR": "job-only",
		},
	}

	step := &dsl.Step{
		Name: "test-step",
		Uses: "run@v1",
		Env: map[string]string{
			"LEVEL_3":  "step", // overrides job
			"STEP_VAR": "step-only",
		},
	}

	ctx := dsl.NewContextBuilder(workflow).WithJob(job).Build()
	rendered, err := renderer.RenderStep(workflow, job, step, ctx)
	require.NoError(t, err)
	require.NotNil(t, rendered)

	// Check env merging (step > job > workflow)
	assert.Equal(t, "workflow", rendered.Env["LEVEL_1"])   // workflow only
	assert.Equal(t, "job", rendered.Env["LEVEL_2"])        // job overrides workflow
	assert.Equal(t, "step", rendered.Env["LEVEL_3"])       // step overrides job
	assert.Equal(t, "job-only", rendered.Env["JOB_VAR"])   // job only
	assert.Equal(t, "step-only", rendered.Env["STEP_VAR"]) // step only
	assert.Equal(t, "v1.0.0", rendered.Env["VERSION"])     // expression rendered
}

func TestWorkflowRenderer_ComplexExpressions(t *testing.T) {
	renderer := dsl.NewWorkflowRenderer()

	workflow := &dsl.Workflow{
		Name: "Complex Test",
		Vars: map[string]interface{}{
			"image": "nginx",
			"tag":   "1.21",
			"port":  8080,
		},
	}

	job := &dsl.Job{
		Name:   "deploy",
		RunsOn: "ubuntu-latest",
		Steps: []*dsl.Step{
			{
				Name: "Deploy Container",
				Uses: "docker@v1",
				With: map[string]interface{}{
					"image":   "${{ format(\"{0}:{1}\", vars.image, vars.tag) }}",
					"port":    "${{ vars.port }}",
					"name":    "${{ upper(vars.image) }}",
					"command": "echo Running ${{ vars.image }} on port ${{ vars.port }}",
				},
			},
		},
	}

	ctx := dsl.NewContextBuilder(workflow).WithJob(job).Build()
	rendered, err := renderer.RenderStep(workflow, job, job.Steps[0], ctx)
	require.NoError(t, err)
	require.NotNil(t, rendered)

	// Check complex expression rendering
	assert.Equal(t, "nginx:1.21", rendered.With["image"])
	assert.Equal(t, "8080", rendered.With["port"])
	assert.Equal(t, "NGINX", rendered.With["name"])
	assert.Equal(t, "echo Running nginx on port 8080", rendered.With["command"])
}

func TestWorkflowRenderer_ErrorHandling(t *testing.T) {
	renderer := dsl.NewWorkflowRenderer()

	tests := []struct {
		name        string
		workflow    *dsl.Workflow
		expectError bool
		errorMsg    string
	}{
		{
			name: "invalid expression in env",
			workflow: &dsl.Workflow{
				Name: "Test",
				Env: map[string]string{
					"INVALID": "${{ 1 + }}",
				},
				Jobs: map[string]*dsl.Job{
					"test": {
						Name:   "test",
						RunsOn: "ubuntu-latest",
						Steps:  []*dsl.Step{},
					},
				},
			},
			expectError: true,
			errorMsg:    "render workflow env",
		},
		{
			name: "invalid if condition type",
			workflow: &dsl.Workflow{
				Name: "Test",
				Jobs: map[string]*dsl.Job{
					"test": {
						Name:   "test",
						RunsOn: "ubuntu-latest",
						Steps: []*dsl.Step{
							{
								Name: "Invalid If",
								Uses: "run@v1",
								If:   `"string"`, // should be bool
							},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "evaluate if condition",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := renderer.RenderWorkflow(tt.workflow)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
