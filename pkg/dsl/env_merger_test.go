package dsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvMerger(t *testing.T) {
	workflow := &Workflow{
		Name: "test",
		Vars: map[string]interface{}{
			"version": "v1.0.0",
		},
		Env: map[string]string{
			"APP_ENV":   "${{ vars.version }}",
			"LOG_LEVEL": "info",
		},
	}

	job := &Job{
		Name: "build",
		Env: map[string]string{
			"LOG_LEVEL": "debug", // overrides workflow
			"BUILD_DIR": "/tmp",
		},
	}

	step := &Step{
		Name: "compile",
		Env: map[string]string{
			"BUILD_DIR": "/opt", // overrides job
			"COMPILER":  "gcc",
		},
	}

	ctx := NewContextBuilder(workflow).Build()

	renderer := NewWorkflowRenderer()
	merged, err := renderer.envMerger.MergeStepEnv(workflow, job, step, ctx)
	require.NoError(t, err)

	// Check merging (step > job > workflow)
	assert.Equal(t, "v1.0.0", merged["APP_ENV"])  // rendered expression
	assert.Equal(t, "debug", merged["LOG_LEVEL"]) // job overrides workflow
	assert.Equal(t, "/opt", merged["BUILD_DIR"])  // step overrides job
	assert.Equal(t, "gcc", merged["COMPILER"])    // step only
}
