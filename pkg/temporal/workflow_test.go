package temporal

import (
	"testing"
	"time"

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

func TestBuildEvalContext_WithMatrixInstance(t *testing.T) {
	wf := &dsl.Workflow{
		Name: "test-workflow",
	}

	job := &dsl.Job{
		Name:   "test-job",
		RunsOn: "test-queue",
	}

	instance := &dsl.MatrixInstance{
		Index: 0,
		Matrix: map[string]interface{}{
			"os":      "linux",
			"version": "1.0",
		},
	}

	ctx := buildEvalContext(wf, job, instance)

	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.Matrix)
	assert.Equal(t, "linux", ctx.Matrix["os"])
	assert.Equal(t, "1.0", ctx.Matrix["version"])
}

func TestGetMaxParallel(t *testing.T) {
	tests := []struct {
		name           string
		job            *dsl.Job
		totalInstances int
		expected       int
	}{
		{
			name:           "nil strategy",
			job:            &dsl.Job{},
			totalInstances: 10,
			expected:       10,
		},
		{
			name: "max_parallel not set",
			job: &dsl.Job{
				Strategy: &dsl.Strategy{},
			},
			totalInstances: 10,
			expected:       10,
		},
		{
			name: "max_parallel set to 0",
			job: &dsl.Job{
				Strategy: &dsl.Strategy{MaxParallel: 0},
			},
			totalInstances: 10,
			expected:       10,
		},
		{
			name: "max_parallel set to 3",
			job: &dsl.Job{
				Strategy: &dsl.Strategy{MaxParallel: 3},
			},
			totalInstances: 10,
			expected:       3,
		},
		{
			name: "max_parallel greater than total",
			job: &dsl.Job{
				Strategy: &dsl.Strategy{MaxParallel: 20},
			},
			totalInstances: 10,
			expected:       20,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := getMaxParallel(tc.job, tc.totalInstances)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetFailFast(t *testing.T) {
	falseVal := false
	trueVal := true

	tests := []struct {
		name     string
		job      *dsl.Job
		expected bool
	}{
		{
			name:     "nil strategy",
			job:      &dsl.Job{},
			expected: true,
		},
		{
			name: "fail_fast nil",
			job: &dsl.Job{
				Strategy: &dsl.Strategy{},
			},
			expected: true,
		},
		{
			name: "fail_fast true",
			job: &dsl.Job{
				Strategy: &dsl.Strategy{FailFast: &trueVal},
			},
			expected: true,
		},
		{
			name: "fail_fast false",
			job: &dsl.Job{
				Strategy: &dsl.Strategy{FailFast: &falseVal},
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := getFailFast(tc.job)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TODO: Workflow retry strategy integration tests require proper Temporal SDK
// test environment setup with activity mocking. These tests are deferred to
// integration tests (test/integration/) where we can use a real Temporal server.
//
// What was attempted:
// - TestRunJobWorkflow_CustomRetryStrategy
// - TestRunJobWorkflow_DefaultRetryStrategy
// - TestRunJobWorkflow_DisableRetry
// - TestRunJobWorkflow_MaxRetryLimit
//
// Issue: Temporal WorkflowEnvironment.OnActivity() requires activities to be
// registered first, but we cannot easily serialize/deserialize test closures.
//
// Solution: Integration tests with real Temporal server are more appropriate
// for end-to-end workflow testing. Unit tests focus on:
// - RetryPolicyResolver (pkg/dsl/retry_test.go)
// - ToTemporalRetryPolicy() NonRetryableErrorTypes (pkg/dsl/retry_nonretry_test.go)

// TestCalculateNextRetryInterval_MaxIntervalLimit tests max interval limit.
func TestCalculateNextRetryInterval_MaxIntervalLimit(t *testing.T) {
	policy := &dsl.ResolvedRetryPolicy{
		InitialInterval:    1 * time.Second,
		BackoffCoefficient: 2.0,
		MaxInterval:        10 * time.Second, // 最大间隔 10 秒
	}

	// 第 10 次重试，指数退避: 1s * 2^10 = 1024s >> 10s
	interval := policy.CalculateNextRetryInterval(10)

	// 验证被限制在 MaxInterval
	assert.Equal(t, 10*time.Second, interval,
		"Retry interval should be capped at MaxInterval")
}

// TestCalculateNextRetryInterval_FixedInterval tests fixed interval (BackoffCoefficient=1.0).
func TestCalculateNextRetryInterval_FixedInterval(t *testing.T) {
	policy := &dsl.ResolvedRetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 1.0, // 固定间隔
		MaxInterval:        60 * time.Second,
	}

	// 所有重试间隔应该相同
	for attempt := 1; attempt <= 5; attempt++ {
		interval := policy.CalculateNextRetryInterval(attempt)
		assert.Equal(t, 5*time.Second, interval,
			"Fixed interval should be constant for all attempts")
	}
}

// TestToTemporalRetryPolicy_NonRetryableErrorTypes tests NonRetryableErrorTypes list.
func TestToTemporalRetryPolicy_NonRetryableErrorTypes(t *testing.T) {
	policy := &dsl.ResolvedRetryPolicy{
		MaxAttempts:        3,
		InitialInterval:    1 * time.Second,
		BackoffCoefficient: 2.0,
		MaxInterval:        60 * time.Second,
	}

	temporalPolicy := policy.ToTemporalRetryPolicy()

	// 验证所有错误类型都在列表中
	expectedTypes := []string{
		"validation_error",
		"schema_error",
		"not_found",
		"permission_denied",
		"invalid_argument",
		"node_not_registered",
		"plugin_load_error",
	}

	assert.ElementsMatch(t, expectedTypes, temporalPolicy.NonRetryableErrorTypes,
		"NonRetryableErrorTypes list must contain all 7 error types")
}
