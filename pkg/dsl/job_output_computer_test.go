package dsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobOutputComputer_Compute(t *testing.T) {
	computer := NewJobOutputComputer()

	job := &Job{
		Name: "build",
		Outputs: map[string]string{
			"version": "${{ steps.build_step.outputs.version }}",
			"commit":  "${{ steps.build_step.outputs.commit }}",
			"static":  "static-value",
		},
	}

	// Create context with step outputs
	outputManager := NewStepsOutputManager()
	outputManager.Set("build_step", map[string]interface{}{
		"version": "v1.2.3",
		"commit":  "abc123def456",
	})

	workflow := &Workflow{
		Name: "test",
		Vars: make(map[string]interface{}),
		Jobs: map[string]*Job{"build": job},
	}

	ctx := NewContextBuilder(workflow).Build()
	ctx.Steps = outputManager.ToContext()

	// Compute outputs
	outputs, err := computer.Compute(job, ctx)
	require.NoError(t, err)

	assert.Equal(t, "v1.2.3", outputs["version"])
	assert.Equal(t, "abc123def456", outputs["commit"])
	assert.Equal(t, "static-value", outputs["static"])
}

func TestJobOutputComputer_WithExpressions(t *testing.T) {
	computer := NewJobOutputComputer()

	job := &Job{
		Name: "build",
		Outputs: map[string]string{
			"full_version": "${{ format('{0}-{1}', steps.build_step.outputs.version, steps.build_step.outputs.commit) }}",
			"upper_name":   "${{ upper('build') }}",
		},
	}

	// Create context
	outputManager := NewStepsOutputManager()
	outputManager.Set("build_step", map[string]interface{}{
		"version": "v1.0.0",
		"commit":  "abc123",
	})

	workflow := &Workflow{
		Name: "test",
		Vars: make(map[string]interface{}),
		Jobs: map[string]*Job{"build": job},
	}

	ctx := NewContextBuilder(workflow).Build()
	ctx.Steps = outputManager.ToContext()

	// Compute outputs
	outputs, err := computer.Compute(job, ctx)
	require.NoError(t, err)

	assert.Equal(t, "v1.0.0-abc123", outputs["full_version"])
	assert.Equal(t, "BUILD", outputs["upper_name"])
}

func TestJobOutputComputer_EmptyOutputs(t *testing.T) {
	computer := NewJobOutputComputer()

	job := &Job{
		Name:    "test",
		Outputs: map[string]string{},
	}

	workflow := &Workflow{
		Name: "test",
		Vars: make(map[string]interface{}),
		Jobs: map[string]*Job{"test": job},
	}
	ctx := NewContextBuilder(workflow).Build()

	outputs, err := computer.Compute(job, ctx)
	require.NoError(t, err)
	assert.Empty(t, outputs)
}

func TestJobOutputComputer_InvalidExpression(t *testing.T) {
	computer := NewJobOutputComputer()

	job := &Job{
		Name: "build",
		Outputs: map[string]string{
			"invalid": "${{ steps.nonexistent.outputs.value }}",
		},
	}

	workflow := &Workflow{
		Name: "test",
		Vars: make(map[string]interface{}),
		Jobs: map[string]*Job{"build": job},
	}
	ctx := NewContextBuilder(workflow).Build()

	_, err := computer.Compute(job, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compute job output")
}
