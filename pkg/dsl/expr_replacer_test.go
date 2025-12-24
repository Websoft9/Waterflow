package dsl

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpressionReplacer_Replace(t *testing.T) {
	engine := NewEngine(1 * time.Second)
	replacer := NewExpressionReplacer(engine)

	tests := []struct {
		name      string
		input     string
		ctxUpdate func(*EvalContext)
		want      string
		wantErr   bool
	}{
		{
			name:  "simple variable",
			input: "Version: ${{ vars.version }}",
			ctxUpdate: func(ctx *EvalContext) {
				ctx.Vars = map[string]interface{}{"version": "v1.2.3"}
			},
			want:    "Version: v1.2.3",
			wantErr: false,
		},
		{
			name:  "multiple expressions",
			input: "${{ vars.app }} v${{ vars.version }}",
			ctxUpdate: func(ctx *EvalContext) {
				ctx.Vars = map[string]interface{}{
					"app":     "MyApp",
					"version": "1.0.0",
				}
			},
			want:    "MyApp v1.0.0",
			wantErr: false,
		},
		{
			name:  "expression with function",
			input: "Name: ${{ upper(workflow.name) }}",
			ctxUpdate: func(ctx *EvalContext) {
				ctx.Workflow = map[string]interface{}{"name": "build"}
			},
			want:    "Name: BUILD",
			wantErr: false,
		},
		{
			name:      "no expressions",
			input:     "plain text",
			ctxUpdate: func(ctx *EvalContext) {},
			want:      "plain text",
			wantErr:   false,
		},
		{
			name:      "expression with arithmetic",
			input:     "Result: ${{ 5 + 3 }}",
			ctxUpdate: func(ctx *EvalContext) {},
			want:      "Result: 8",
			wantErr:   false,
		},
		{
			name:  "expression with comparison",
			input: "Status: ${{ vars.env == \"production\" }}",
			ctxUpdate: func(ctx *EvalContext) {
				ctx.Vars = map[string]interface{}{"env": "production"}
			},
			want:    "Status: true",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := mockContextBuilder()
			tt.ctxUpdate(ctx)

			got, err := replacer.Replace(tt.input, ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestExpressionReplacer_ReplaceInMap(t *testing.T) {
	engine := NewEngine(1 * time.Second)
	replacer := NewExpressionReplacer(engine)

	ctx := mockContextBuilder()
	ctx.Vars = map[string]interface{}{
		"repo":   "my-repo",
		"branch": "main",
		"commit": "abc123",
	}

	input := map[string]interface{}{
		"repository": "${{ vars.repo }}",
		"branch":     "${{ vars.branch }}",
		"commit":     "${{ vars.commit }}",
		"static":     "value",
		"number":     42,
	}

	got, err := replacer.ReplaceInMap(input, ctx)
	require.NoError(t, err)

	want := map[string]interface{}{
		"repository": "my-repo",
		"branch":     "main",
		"commit":     "abc123",
		"static":     "value",
		"number":     42,
	}

	assert.Equal(t, want, got)
}

func TestExpressionReplacer_ReplaceInMap_Nested(t *testing.T) {
	engine := NewEngine(1 * time.Second)
	replacer := NewExpressionReplacer(engine)

	ctx := mockContextBuilder()
	ctx.Vars = map[string]interface{}{
		"version": "v1.0.0",
	}

	input := map[string]interface{}{
		"config": map[string]interface{}{
			"app": map[string]interface{}{
				"version": "${{ vars.version }}",
			},
		},
	}

	got, err := replacer.ReplaceInMap(input, ctx)
	require.NoError(t, err)

	want := map[string]interface{}{
		"config": map[string]interface{}{
			"app": map[string]interface{}{
				"version": "v1.0.0",
			},
		},
	}

	assert.Equal(t, want, got)
}

func TestExpressionReplacer_ReplaceInArray(t *testing.T) {
	engine := NewEngine(1 * time.Second)
	replacer := NewExpressionReplacer(engine)

	ctx := mockContextBuilder()
	ctx.Vars = map[string]interface{}{
		"env": "production",
	}

	input := []interface{}{
		"${{ vars.env }}",
		"static",
		42,
	}

	got, err := replacer.ReplaceInArray(input, ctx)
	require.NoError(t, err)

	want := []interface{}{
		"production",
		"static",
		42,
	}

	assert.Equal(t, want, got)
}

func TestExpressionReplacer_EvaluateTyped(t *testing.T) {
	engine := NewEngine(1 * time.Second)
	replacer := NewExpressionReplacer(engine)

	ctx := mockContextBuilder()

	tests := []struct {
		name       string
		expression string
		want       interface{}
		wantType   string
		wantErr    bool
	}{
		{
			name:       "int type preserved",
			expression: "5 + 3",
			want:       8,
			wantType:   "int",
			wantErr:    false,
		},
		{
			name:       "bool type preserved",
			expression: "true",
			want:       true,
			wantType:   "bool",
			wantErr:    false,
		},
		{
			name:       "string type preserved",
			expression: `"hello"`,
			want:       "hello",
			wantType:   "string",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := replacer.EvaluateTyped(tt.expression, ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestExpressionReplacer_ErrorHandling(t *testing.T) {
	engine := NewEngine(1 * time.Second)
	replacer := NewExpressionReplacer(engine)

	ctx := mockContextBuilder()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "invalid expression",
			input:   "${{ 1 + }}",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := replacer.Replace(tt.input, ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExpressionReplacer_ReplaceInArray_DeepNesting(t *testing.T) {
	engine := NewEngine(1 * time.Second)
	replacer := NewExpressionReplacer(engine)

	ctx := mockContextBuilder()
	ctx.Vars = map[string]interface{}{
		"env":     "prod",
		"version": "v2.0.0",
	}

	// 深度嵌套数组，包含 map 和数组
	input := []interface{}{
		"${{ vars.env }}",
		map[string]interface{}{
			"version": "${{ vars.version }}",
			"tags":    []interface{}{"${{ vars.env }}", "stable"},
		},
		[]interface{}{
			"${{ vars.env }}",
			map[string]interface{}{
				"nested": "${{ vars.version }}",
			},
		},
	}

	got, err := replacer.ReplaceInArray(input, ctx)
	require.NoError(t, err)

	want := []interface{}{
		"prod",
		map[string]interface{}{
			"version": "v2.0.0",
			"tags":    []interface{}{"prod", "stable"},
		},
		[]interface{}{
			"prod",
			map[string]interface{}{
				"nested": "v2.0.0",
			},
		},
	}

	assert.Equal(t, want, got)
}

func TestExpressionReplacer_ReplaceInMap_DeepNesting(t *testing.T) {
	engine := NewEngine(1 * time.Second)
	replacer := NewExpressionReplacer(engine)

	ctx := mockContextBuilder()
	ctx.Vars = map[string]interface{}{
		"host": "localhost",
		"port": "3306",
	}

	// 深度嵌套 map，包含数组
	input := map[string]interface{}{
		"database": map[string]interface{}{
			"primary": map[string]interface{}{
				"host": "${{ vars.host }}",
				"port": "${{ vars.port }}",
			},
			"replicas": []interface{}{
				map[string]interface{}{
					"host": "${{ vars.host }}",
				},
			},
		},
	}

	got, err := replacer.ReplaceInMap(input, ctx)
	require.NoError(t, err)

	want := map[string]interface{}{
		"database": map[string]interface{}{
			"primary": map[string]interface{}{
				"host": "localhost",
				"port": "3306",
			},
			"replicas": []interface{}{
				map[string]interface{}{
					"host": "localhost",
				},
			},
		},
	}

	assert.Equal(t, want, got)
}

func TestExpressionReplacer_ReplaceInArray_ErrorPropagation(t *testing.T) {
	engine := NewEngine(1 * time.Second)
	replacer := NewExpressionReplacer(engine)

	ctx := mockContextBuilder()

	// 数组中包含无效表达式
	input := []interface{}{
		"${{ 1 + }}",
		"valid",
	}

	_, err := replacer.ReplaceInArray(input, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "index 0")
}

func TestExpressionReplacer_ReplaceInMap_ErrorPropagation(t *testing.T) {
	engine := NewEngine(1 * time.Second)
	replacer := NewExpressionReplacer(engine)

	ctx := mockContextBuilder()

	// Map 中包含无效表达式
	input := map[string]interface{}{
		"invalid": "${{ 1 + }}",
		"valid":   "value",
	}

	_, err := replacer.ReplaceInMap(input, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestExpressionReplacer_EvaluateTyped_ErrorCases(t *testing.T) {
	engine := NewEngine(1 * time.Second)
	replacer := NewExpressionReplacer(engine)

	ctx := mockContextBuilder()

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := replacer.EvaluateTyped(tt.expression, ctx)
			assert.Error(t, err)
		})
	}
}

func TestExpressionReplacer_NestingDepthLimit(t *testing.T) {
	engine := NewEngine(1 * time.Second)
	replacer := NewExpressionReplacer(engine)

	ctx := mockContextBuilder()

	// 创建一个超过10层嵌套的结构
	deepMap := make(map[string]interface{})
	current := deepMap
	for i := 0; i < 12; i++ {
		nested := make(map[string]interface{})
		current["level"] = nested
		current = nested
	}
	current["value"] = "too deep"

	// 应该因为嵌套深度超过限制而失败
	_, err := replacer.ReplaceInMap(deepMap, ctx)
	require.Error(t, err)

	// 检查错误信息包含深度相关内容
	assert.Contains(t, err.Error(), "nesting too deep")
}

func TestExpressionReplacer_NestingWithinLimit(t *testing.T) {
	engine := NewEngine(1 * time.Second)
	replacer := NewExpressionReplacer(engine)

	ctx := mockContextBuilder()
	ctx.Vars = map[string]interface{}{
		"value": "test",
	}

	// 创建一个接近但不超过10层嵌套的结构
	deepMap := make(map[string]interface{})
	current := deepMap
	for i := 0; i < 8; i++ {
		nested := make(map[string]interface{})
		current["level"] = nested
		current = nested
	}
	current["value"] = "${{ vars.value }}"

	// 应该成功
	result, err := replacer.ReplaceInMap(deepMap, ctx)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// 验证表达式被正确替换（深度遍历检查）
	checkDeep := result
	for i := 0; i < 8; i++ {
		next, ok := checkDeep["level"].(map[string]interface{})
		require.True(t, ok, "expected nested map at level %d", i)
		checkDeep = next
	}
	assert.Equal(t, "test", checkDeep["value"])
}

func TestExpressionReplacer_ArrayNestingDepthLimit(t *testing.T) {
	engine := NewEngine(1 * time.Second)
	replacer := NewExpressionReplacer(engine)

	ctx := mockContextBuilder()

	// 创建一个超过10层嵌套的数组
	var deepArray interface{} = "value"
	for i := 0; i < 12; i++ {
		deepArray = []interface{}{deepArray}
	}

	// 应该因为嵌套深度超过限制而失败
	_, err := replacer.ReplaceInArray(deepArray.([]interface{}), ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nesting too deep")
}
