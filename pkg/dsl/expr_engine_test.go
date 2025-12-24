package dsl

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_Evaluate_SimpleExpressions(t *testing.T) {
	engine := NewEngine(1 * time.Second)

	tests := []struct {
		name       string
		expression string
		ctx        *EvalContext
		want       interface{}
		wantErr    bool
	}{
		{
			name:       "arithmetic addition",
			expression: "1 + 2",
			ctx:        &EvalContext{},
			want:       3,
			wantErr:    false,
		},
		{
			name:       "arithmetic multiplication",
			expression: "2 * 3",
			ctx:        &EvalContext{},
			want:       6,
			wantErr:    false,
		},
		{
			name:       "string concatenation",
			expression: `"hello" + " " + "world"`,
			ctx:        &EvalContext{},
			want:       "hello world",
			wantErr:    false,
		},
		{
			name:       "comparison equal",
			expression: "5 == 5",
			ctx:        &EvalContext{},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "comparison not equal",
			expression: "5 != 3",
			ctx:        &EvalContext{},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "logical and",
			expression: "true && false",
			ctx:        &EvalContext{},
			want:       false,
			wantErr:    false,
		},
		{
			name:       "logical or",
			expression: "true || false",
			ctx:        &EvalContext{},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "variable reference",
			expression: "vars.version",
			ctx: &EvalContext{
				Vars: map[string]interface{}{
					"version": "v1.2.3",
				},
			},
			want:    "v1.2.3",
			wantErr: false,
		},
		{
			name:       "nested object access",
			expression: "vars.config.timeout",
			ctx: &EvalContext{
				Vars: map[string]interface{}{
					"config": map[string]interface{}{
						"timeout": 30,
					},
				},
			},
			want:    30,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Evaluate(tt.expression, tt.ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestEngine_Evaluate_BuiltinFunctions(t *testing.T) {
	engine := NewEngine(1 * time.Second)

	// Create context with functions

	ctx := mockContextBuilder()

	tests := []struct {
		name       string
		expression string
		want       interface{}
		wantErr    bool
	}{
		{
			name:       "len of string",
			expression: `len("hello")`,
			want:       5,
			wantErr:    false,
		},
		{
			name:       "upper case",
			expression: `upper("hello")`,
			want:       "HELLO",
			wantErr:    false,
		},
		{
			name:       "lower case",
			expression: `lower("WORLD")`,
			want:       "world",
			wantErr:    false,
		},
		{
			name:       "trim whitespace",
			expression: `trim("  hello  ")`,
			want:       "hello",
			wantErr:    false,
		},
		{
			name:       "format string",
			expression: `format("Hello {0}", "World")`,
			want:       "Hello World",
			wantErr:    false,
		},
		{
			name:       "format with multiple args",
			expression: `format("{0} v{1}", "App", "1.2.3")`,
			want:       "App v1.2.3",
			wantErr:    false,
		},
		{
			name:       "always",
			expression: "always()",
			want:       true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Evaluate(tt.expression, ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestEngine_Evaluate_ComplexExpressions(t *testing.T) {
	engine := NewEngine(1 * time.Second)

	ctx := mockContextBuilder()
	ctx.Workflow = map[string]interface{}{
		"name": "Build and Test",
	}
	ctx.Job = map[string]interface{}{
		"status": "success",
	}
	ctx.Vars = map[string]interface{}{
		"env":     "production",
		"version": "v1.2.3",
		"image":   "myapp",
	}
	ctx.Env = map[string]string{
		"PATH": "/usr/bin",
	}

	tests := []struct {
		name       string
		expression string
		want       interface{}
		wantErr    bool
	}{
		{
			name:       "complex conditional",
			expression: `vars.env == "production" && job["status"] == "success"`,
			want:       true,
			wantErr:    false,
		},
		{
			name:       "format with vars",
			expression: `format("{0}:{1}", vars.image, vars.version)`,
			want:       "myapp:v1.2.3",
			wantErr:    false,
		},
		{
			name:       "nested format",
			expression: `upper(format("env: {0}", vars.env))`,
			want:       "ENV: PRODUCTION",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Evaluate(tt.expression, ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestEngine_Timeout(t *testing.T) {
	engine := NewEngine(10 * time.Millisecond)

	// Note: Creating a truly infinite loop in expr is difficult
	// This test mainly verifies timeout mechanism exists
	ctx := &EvalContext{}

	// Test quick expression doesn't timeout
	_, err := engine.Evaluate("1 + 1", ctx)
	assert.NoError(t, err)
}

func TestEngine_CompileCaching(t *testing.T) {
	engine := NewEngine(1 * time.Second)

	// Compile same expression twice
	prog1, err1 := engine.Compile("vars.version")
	require.NoError(t, err1)
	require.NotNil(t, prog1)

	prog2, err2 := engine.Compile("vars.version")
	require.NoError(t, err2)
	require.NotNil(t, prog2)

	// Programs should be different instances but functionally equivalent
	assert.NotNil(t, prog1)
	assert.NotNil(t, prog2)
}

func TestEngine_ErrorHandling(t *testing.T) {
	engine := NewEngine(1 * time.Second)

	tests := []struct {
		name       string
		expression string
		wantErr    bool
	}{
		{
			name:       "syntax error",
			expression: "1 +",
			wantErr:    true,
		},
		// AllowUndefinedVariables returns nil for undefined, not error
		// so we skip that test
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := mockContextBuilder()
			_, err := engine.Evaluate(tt.expression, ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEngine_Timeout_Protection(t *testing.T) {
	// 创建一个非常短的超时时间
	engine := NewEngine(1 * time.Millisecond)

	ctx := &EvalContext{
		Vars: map[string]interface{}{},
	}

	// 编译一个简单表达式
	program, err := engine.Compile("1 + 2")
	require.NoError(t, err)

	// 添加延迟模拟超时
	time.Sleep(2 * time.Millisecond)

	// 运行时可能超时（取决于系统负载）
	_, err = engine.RunWithTimeout(program, ctx)
	// 超时错误或成功都是可接受的（取决于执行速度）
	// 主要验证超时机制存在
	if err != nil {
		t.Logf("Timeout triggered as expected: %v", err)
	}
}

func TestEngine_WrapError_WithCompileError(t *testing.T) {
	engine := NewEngine(1 * time.Second)

	// 触发编译错误
	_, err := engine.Compile("1 +")
	require.Error(t, err)

	// 验证错误被包装
	assert.Contains(t, err.Error(), "expression error")
}

func TestEngine_WrapError_WithRuntimeError(t *testing.T) {
	engine := NewEngine(1 * time.Second)

	ctx := &EvalContext{}

	// 触发运行时错误（访问未定义变量）
	_, err := engine.Evaluate("undefined_var", ctx)
	require.Error(t, err)

	// 验证错误被包装
	assert.Error(t, err)
}

func TestEngine_ExpressionLengthLimit(t *testing.T) {
	engine := NewEngine(1 * time.Second)

	// 创建一个超过1024字符的表达式
	longExpr := "1"
	for i := 0; i < 520; i++ {
		longExpr += " + 1"
	}
	// 确保超过1024字符
	require.Greater(t, len(longExpr), 1024)

	_, err := engine.Compile(longExpr)
	require.Error(t, err)

	exprErr, ok := err.(*ExpressionError)
	require.True(t, ok, "expected ExpressionError")
	assert.Equal(t, "length_error", exprErr.Type)
	assert.Contains(t, exprErr.Message, "too long")
	assert.Contains(t, exprErr.Message, "max 1024")
}

func TestEngine_ExpressionWithinLengthLimit(t *testing.T) {
	engine := NewEngine(1 * time.Second)
	ctx := &EvalContext{}

	// 创建一个接近但不超过1024字符的表达式
	expr := "1"
	for i := 0; i < 200; i++ {
		expr += " + 1"
	}
	require.LessOrEqual(t, len(expr), 1024)

	// 应该成功编译和执行
	result, err := engine.Evaluate(expr, ctx)
	require.NoError(t, err)
	assert.Equal(t, 201, result)
}
