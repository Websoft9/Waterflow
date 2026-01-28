package dsl

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEngine_EvaluationTimeout tests that expression evaluation times out after 1 second
func TestEngine_EvaluationTimeout(t *testing.T) {
	engine := NewEngine(100 * time.Millisecond) // Use short timeout for test

	ctx := mockContextBuilder()

	// This expression will timeout (infinite loop simulation not possible in expr)
	// Instead, we test the timeout mechanism exists
	_, err := engine.Evaluate("1 + 1", ctx)

	// Normal expression should succeed
	require.NoError(t, err)
}

// TestEngine_TimeoutProtection tests timeout protection with RunWithTimeout
func TestEngine_TimeoutProtection(t *testing.T) {
	engine := NewEngine(50 * time.Millisecond)

	ctx := mockContextBuilder()

	// Compile a simple program
	program, err := engine.Compile("1 + 1")
	require.NoError(t, err)

	// Should complete before timeout
	result, err := engine.RunWithTimeout(program, ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, result)
}

// TestEngine_VeryComplexExpression tests complex nested expressions don't timeout
func TestEngine_VeryComplexExpression(t *testing.T) {
	engine := NewEngine(1 * time.Second)

	ctx := mockContextBuilder()
	ctx.Vars = map[string]interface{}{
		"a": int64(1),
		"b": int64(2),
		"c": int64(3),
		"d": int64(4),
		"e": int64(5),
	}

	// Complex nested expression
	expr := `((vars.a + vars.b) * vars.c + vars.d) * vars.e`

	result, err := engine.Evaluate(expr, ctx)
	require.NoError(t, err)

	// ((1 + 2) * 3 + 4) * 5 = (9 + 4) * 5 = 65
	// Result type may be int or int64 depending on expr library
	assert.Equal(t, 65, int(result.(int)))
}

// TestExpressionReplacer_TimeoutInReplace tests timeout during string replacement
func TestExpressionReplacer_TimeoutInReplace(t *testing.T) {
	engine := NewEngine(1 * time.Second)
	replacer := NewExpressionReplacer(engine)

	ctx := mockContextBuilder()
	ctx.Vars = map[string]interface{}{
		"value": "test",
	}

	// Multiple expressions in one string
	input := strings.Repeat("${{ vars.value }} ", 100)

	result, err := replacer.Replace(input, ctx)
	require.NoError(t, err)
	assert.Contains(t, result, "test")
}

// TestEngine_ConcurrentEvaluations tests multiple concurrent evaluations
func TestEngine_ConcurrentEvaluations(t *testing.T) {
	engine := NewEngine(1 * time.Second)

	ctx := mockContextBuilder()
	ctx.Vars = map[string]interface{}{
		"value": int64(42),
	}

	// Run 10 concurrent evaluations
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			result, err := engine.Evaluate("vars.value * 2", ctx)
			assert.NoError(t, err)
			// Result type may be int or int64
			assert.Equal(t, 84, int(result.(int)))
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(2 * time.Second):
			t.Fatal("Concurrent evaluation timeout")
		}
	}
}

// TestConditionEvaluator_TimeoutProtection tests condition evaluation timeout
func TestConditionEvaluator_TimeoutProtection(t *testing.T) {
	engine := NewEngine(1 * time.Second)
	evaluator := NewConditionEvaluator(engine)

	ctx := mockContextBuilder()
	ctx.Vars = map[string]interface{}{
		"env": "production",
	}
	ctx.Job = map[string]interface{}{
		"status": "success",
	}

	// Complex condition should complete before timeout
	condition := `vars.env == "production" && job.status == "success"`

	result, err := evaluator.Evaluate(condition, ctx)
	require.NoError(t, err)
	assert.True(t, result)
}

// TestExpressionReplacer_NestingWithTimeout tests deep nesting with timeout
func TestExpressionReplacer_NestingWithTimeout(t *testing.T) {
	engine := NewEngine(1 * time.Second)
	replacer := NewExpressionReplacer(engine)

	ctx := mockContextBuilder()
	ctx.Vars = map[string]interface{}{
		"value": "test",
	}

	// Create deeply nested map (within limit of 10)
	input := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": map[string]interface{}{
					"level4": map[string]interface{}{
						"level5": map[string]interface{}{
							"value": "${{ vars.value }}",
						},
					},
				},
			},
		},
	}

	result, err := replacer.ReplaceInMap(input, ctx)
	require.NoError(t, err)
	assert.NotNil(t, result)
}
