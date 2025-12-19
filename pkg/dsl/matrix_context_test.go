package dsl

import (
	"testing"

	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
)

// TestEvalContext_Matrix 测试 Matrix 上下文
func TestEvalContext_Matrix(t *testing.T) {
	workflow := &Workflow{
		Name: "test",
		Vars: map[string]interface{}{
			"version": "1.0",
		},
	}

	job := &Job{
		Name: "deploy",
	}

	matrix := map[string]interface{}{
		"server": "web1",
		"env":    "prod",
		"port":   8080,
	}

	ctx := NewContextBuilder(workflow).
		WithJob(job).
		WithMatrix(matrix).
		Build()

	assert.NotNil(t, ctx.Matrix)
	assert.Equal(t, "web1", ctx.Matrix["server"])
	assert.Equal(t, "prod", ctx.Matrix["env"])
	assert.Equal(t, 8080, ctx.Matrix["port"])
}

// TestEvalContext_MatrixInExpression 测试在表达式中使用 Matrix
func TestEvalContext_MatrixInExpression(t *testing.T) {
	workflow := &Workflow{
		Name: "test",
	}

	matrix := map[string]interface{}{
		"server": "web1",
		"env":    "prod",
		"port":   8080,
	}

	evalCtx := NewContextBuilder(workflow).
		WithMatrix(matrix).
		Build()

	tests := []struct {
		name string
		expr string
		want interface{}
	}{
		{
			name: "access matrix.server",
			expr: "matrix.server",
			want: "web1",
		},
		{
			name: "access matrix.env",
			expr: "matrix.env",
			want: "prod",
		},
		{
			name: "access matrix.port",
			expr: "matrix.port",
			want: 8080,
		},
		{
			name: "string interpolation",
			expr: `"Deploying to " + matrix.server + ":" + string(matrix.port)`,
			want: "Deploying to web1:8080",
		},
		{
			name: "conditional with matrix",
			expr: `matrix.env == "prod"`,
			want: true,
		},
		{
			name: "conditional with matrix (false)",
			expr: `matrix.env == "staging"`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, err := expr.Compile(tt.expr, expr.Env(evalCtx))
			assert.NoError(t, err)

			result, err := expr.Run(program, evalCtx)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

// TestEvalContext_NoMatrix 测试无 Matrix 的上下文
func TestEvalContext_NoMatrix(t *testing.T) {
	workflow := &Workflow{
		Name: "test",
	}

	ctx := NewContextBuilder(workflow).
		Build()

	// Matrix 应该为 nil
	assert.Nil(t, ctx.Matrix)
}

// TestEvalContext_MatrixWithDifferentTypes 测试不同类型的 Matrix 值
func TestEvalContext_MatrixWithDifferentTypes(t *testing.T) {
	workflow := &Workflow{
		Name: "test",
	}

	matrix := map[string]interface{}{
		"version": 1.20,
		"os":      "ubuntu",
		"enabled": true,
	}

	evalCtx := NewContextBuilder(workflow).
		WithMatrix(matrix).
		Build()

	tests := []struct {
		name string
		expr string
		want interface{}
	}{
		{
			name: "float value",
			expr: "matrix.version",
			want: 1.20,
		},
		{
			name: "string value",
			expr: "matrix.os",
			want: "ubuntu",
		},
		{
			name: "bool value",
			expr: "matrix.enabled",
			want: true,
		},
		{
			name: "comparison with float",
			expr: "matrix.version > 1.19",
			want: true,
		},
		{
			name: "comparison with bool",
			expr: "matrix.enabled == true",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, err := expr.Compile(tt.expr, expr.Env(evalCtx))
			assert.NoError(t, err)

			result, err := expr.Run(program, evalCtx)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}
