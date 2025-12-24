package dsl

import (
	"fmt"
	"regexp"
	"strings"
)

// Expression pattern: ${{ ... }}
var exprPattern = regexp.MustCompile(`\$\{\{(.+?)\}\}`)

// ExpressionReplacer replaces expressions in strings
type ExpressionReplacer struct {
	engine   *Engine
	maxDepth int
}

// NewExpressionReplacer creates a new expression replacer
func NewExpressionReplacer(engine *Engine) *ExpressionReplacer {
	return &ExpressionReplacer{
		engine:   engine,
		maxDepth: 10, // Maximum nesting depth
	}
}

// Replace replaces all expressions in a string
func (r *ExpressionReplacer) Replace(input string, ctx *EvalContext) (string, error) {
	var lastErr error

	result := exprPattern.ReplaceAllStringFunc(input, func(match string) string {
		// Extract expression content (remove ${{ and }})
		expression := strings.TrimSpace(match[3 : len(match)-2])

		// Evaluate
		value, err := r.engine.Evaluate(expression, ctx)
		if err != nil {
			lastErr = err
			return match // Keep original
		}

		// Convert to string
		return fmt.Sprintf("%v", value)
	})

	if lastErr != nil {
		return "", lastErr
	}

	return result, nil
}

// ReplaceInMap recursively replaces expressions in a map
func (r *ExpressionReplacer) ReplaceInMap(m map[string]interface{}, ctx *EvalContext) (map[string]interface{}, error) {
	return r.replaceInMapWithDepth(m, ctx, 0)
}

// replaceInMapWithDepth recursively replaces expressions with depth tracking
func (r *ExpressionReplacer) replaceInMapWithDepth(m map[string]interface{}, ctx *EvalContext, depth int) (map[string]interface{}, error) {
	// Check nesting depth limit
	if depth >= r.maxDepth {
		return nil, NewExpressionError(
			"",
			fmt.Sprintf("expression nesting too deep: %d levels (max %d)", depth, r.maxDepth),
			"depth_error",
		)
	}

	result := make(map[string]interface{})

	for k, v := range m {
		switch val := v.(type) {
		case string:
			replaced, err := r.Replace(val, ctx)
			if err != nil {
				return nil, fmt.Errorf("replace in key '%s': %w", k, err)
			}
			result[k] = replaced

		case map[string]interface{}:
			replaced, err := r.replaceInMapWithDepth(val, ctx, depth+1)
			if err != nil {
				return nil, fmt.Errorf("replace in key '%s': %w", k, err)
			}
			result[k] = replaced

		case []interface{}:
			replaced, err := r.replaceInArrayWithDepth(val, ctx, depth+1)
			if err != nil {
				return nil, fmt.Errorf("replace in key '%s': %w", k, err)
			}
			result[k] = replaced

		default:
			result[k] = v
		}
	}

	return result, nil
}

// ReplaceInArray recursively replaces expressions in an array
func (r *ExpressionReplacer) ReplaceInArray(arr []interface{}, ctx *EvalContext) ([]interface{}, error) {
	return r.replaceInArrayWithDepth(arr, ctx, 0)
}

// replaceInArrayWithDepth recursively replaces expressions with depth tracking
func (r *ExpressionReplacer) replaceInArrayWithDepth(arr []interface{}, ctx *EvalContext, depth int) ([]interface{}, error) {
	// Check nesting depth limit
	if depth >= r.maxDepth {
		return nil, NewExpressionError(
			"",
			fmt.Sprintf("expression nesting too deep: %d levels (max %d)", depth, r.maxDepth),
			"depth_error",
		)
	}

	result := make([]interface{}, len(arr))

	for i, v := range arr {
		switch val := v.(type) {
		case string:
			replaced, err := r.Replace(val, ctx)
			if err != nil {
				return nil, fmt.Errorf("replace in index %d: %w", i, err)
			}
			result[i] = replaced

		case map[string]interface{}:
			replaced, err := r.replaceInMapWithDepth(val, ctx, depth+1)
			if err != nil {
				return nil, fmt.Errorf("replace in index %d: %w", i, err)
			}
			result[i] = replaced

		case []interface{}:
			replaced, err := r.replaceInArrayWithDepth(val, ctx, depth+1)
			if err != nil {
				return nil, fmt.Errorf("replace in index %d: %w", i, err)
			}
			result[i] = replaced

		default:
			result[i] = v
		}
	}

	return result, nil
}

// EvaluateTyped evaluates an expression and preserves its type
func (r *ExpressionReplacer) EvaluateTyped(expression string, ctx *EvalContext) (interface{}, error) {
	value, err := r.engine.Evaluate(expression, ctx)
	if err != nil {
		return nil, err
	}

	// Return value with original type preserved
	return value, nil
}
