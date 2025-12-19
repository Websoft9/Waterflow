package dsl

import (
	"context"
	"fmt"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// Engine is the expression evaluation engine
type Engine struct {
	timeout time.Duration
}

// NewEngine creates a new expression engine with the given timeout
func NewEngine(timeout time.Duration) *Engine {
	return &Engine{
		timeout: timeout,
	}
}

// Compile compiles an expression for later execution (can be cached)
func (e *Engine) Compile(expression string) (*vm.Program, error) {
	// Build options
	options := []expr.Option{
		expr.Env(EvalContext{}),
		expr.AllowUndefinedVariables(),
	}

	program, err := expr.Compile(expression, options...)
	if err != nil {
		return nil, e.wrapError(err, expression)
	}

	return program, nil
}

// Evaluate evaluates an expression with the given context
func (e *Engine) Evaluate(expression string, ctx *EvalContext) (interface{}, error) {
	// Compile expression
	program, err := e.Compile(expression)
	if err != nil {
		return nil, err
	}

	// Execute with timeout protection
	return e.RunWithTimeout(program, ctx)
}

// RunWithTimeout runs a compiled program with timeout protection
func (e *Engine) RunWithTimeout(program *vm.Program, ctx *EvalContext) (interface{}, error) {
	// Create timeout context
	evalCtx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	// Channel to receive result
	type result struct {
		value interface{}
		err   error
	}
	done := make(chan result, 1)

	// Run evaluation in goroutine
	go func() {
		value, err := expr.Run(program, ctx)
		done <- result{value: value, err: err}
	}()

	// Wait for result or timeout
	select {
	case res := <-done:
		if res.err != nil {
			return nil, e.wrapError(res.err, "")
		}
		return res.value, nil
	case <-evalCtx.Done():
		return nil, NewExpressionError(
			"",
			fmt.Sprintf("expression evaluation timeout (>%v)", e.timeout),
			"timeout_error",
		)
	}
}

// wrapError wraps an error into ExpressionError
func (e *Engine) wrapError(err error, expression string) error {
	if exprErr, ok := err.(*ExpressionError); ok {
		return exprErr
	}

	return NewExpressionError(
		expression,
		err.Error(),
		"expression_evaluation_error",
	)
}
