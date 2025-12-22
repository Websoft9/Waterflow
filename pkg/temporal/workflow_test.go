package temporal

import (
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/stretchr/testify/assert"
)

func TestBuildEvalContext(t *testing.T) {
	wf := &dsl.Workflow{
		Name: "test-workflow",
		Vars: map[string]interface{}{
			"version": "1.0.0",
		},
		Env: map[string]string{
			"GLOBAL_VAR": "global",
		},
	}

	job := &dsl.Job{
		Name:   "test-job",
		RunsOn: "test-queue",
		Env: map[string]string{
			"JOB_VAR": "job",
		},
	}

	ctx := buildEvalContext(wf, job, nil)

	assert.NotNil(t, ctx)
	assert.Equal(t, "test-workflow", ctx.Workflow["name"])
	assert.Equal(t, "test-job", ctx.Job["name"])
	assert.Equal(t, "1.0.0", ctx.Vars["version"])
	assert.Equal(t, "global", ctx.Env["GLOBAL_VAR"])
	assert.Equal(t, "job", ctx.Env["JOB_VAR"]) // Job env should override
}
