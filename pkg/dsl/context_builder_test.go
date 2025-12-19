package dsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextBuilder(t *testing.T) {
	workflow := &Workflow{
		Name: "Test Workflow",
		Vars: map[string]interface{}{
			"env":     "production",
			"version": "v1.2.3",
		},
		Env: map[string]string{
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

	builder := NewContextBuilder(workflow).
		WithJob(job).
		WithRunner(map[string]interface{}{
			"os":   "linux",
			"arch": "amd64",
		}).
		WithInputs(map[string]interface{}{
			"branch": "main",
		}).
		WithSecrets(map[string]string{
			"api_key": "secret123",
		})

	ctx := builder.Build()

	// Check workflow context
	assert.Equal(t, "Test Workflow", ctx.Workflow["name"])

	// Check vars
	assert.Equal(t, "production", ctx.Vars["env"])
	assert.Equal(t, "v1.2.3", ctx.Vars["version"])

	// Check env merging (job overrides workflow)
	assert.Equal(t, "debug", ctx.Env["LOG_LEVEL"])
	assert.Equal(t, "/tmp", ctx.Env["BUILD_DIR"])

	// Check job context
	assert.Equal(t, "build", ctx.Job["id"])
	assert.Equal(t, "build", ctx.Job["name"])

	// Check runner
	assert.Equal(t, "linux", ctx.Runner["os"])
	assert.Equal(t, "amd64", ctx.Runner["arch"])

	// Check inputs
	assert.Equal(t, "main", ctx.Inputs["branch"])

	// Check secrets
	assert.Equal(t, "secret123", ctx.Secrets["api_key"])

	// Check functions are registered
	assert.NotNil(t, ctx.Len)
	assert.NotNil(t, ctx.Upper)
	assert.NotNil(t, ctx.Format)
}
