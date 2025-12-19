package dsl

import (
	"fmt"
)

// ConditionEvaluator evaluates conditional expressions
type ConditionEvaluator struct {
	engine *Engine
}

// NewConditionEvaluator creates a new condition evaluator
func NewConditionEvaluator(engine *Engine) *ConditionEvaluator {
	return &ConditionEvaluator{engine: engine}
}

// Evaluate evaluates an if condition expression
func (e *ConditionEvaluator) Evaluate(condition string, ctx *EvalContext) (bool, error) {
	if condition == "" {
		return true, nil // No condition means always execute
	}

	// Evaluate expression
	result, err := e.engine.Evaluate(condition, ctx)
	if err != nil {
		return false, fmt.Errorf("evaluate if condition: %w", err)
	}

	// Type check (must be bool)
	boolResult, ok := result.(bool)
	if !ok {
		return false, NewExpressionError(
			condition,
			fmt.Sprintf("if expression must return bool, got %T: %v", result, result),
			"type_error",
		)
	}

	return boolResult, nil
}
