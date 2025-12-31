package dsl

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestToTemporalRetryPolicy_NonRetryableErrorTypes tests that all 7 error types are present.
func TestToTemporalRetryPolicy_NonRetryableErrorTypes(t *testing.T) {
	policy := &ResolvedRetryPolicy{
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
	assert.Len(t, temporalPolicy.NonRetryableErrorTypes, 7,
		"NonRetryableErrorTypes list must have exactly 7 items")
}

// TestToTemporalRetryPolicy_BasicFields tests basic retry policy conversion.
func TestToTemporalRetryPolicy_BasicFields(t *testing.T) {
	policy := &ResolvedRetryPolicy{
		MaxAttempts:        5,
		InitialInterval:    2 * time.Second,
		BackoffCoefficient: 1.5,
		MaxInterval:        30 * time.Second,
	}

	temporalPolicy := policy.ToTemporalRetryPolicy()

	assert.Equal(t, int32(5), temporalPolicy.MaximumAttempts)
	assert.Equal(t, 2*time.Second, temporalPolicy.InitialInterval)
	assert.Equal(t, 1.5, temporalPolicy.BackoffCoefficient)
	assert.Equal(t, 30*time.Second, temporalPolicy.MaximumInterval)
}
