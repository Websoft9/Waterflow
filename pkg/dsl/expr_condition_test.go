package dsl

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConditionEvaluator_Evaluate(t *testing.T) {
	engine := NewEngine(1 * time.Second)
	evaluator := NewConditionEvaluator(engine)

	// Create base context with functions

	tests := []struct {
		name      string
		condition string
		ctxUpdate func(*EvalContext)
		want      bool
		wantErr   bool
	}{
		{
			name:      "empty condition",
			condition: "",
			ctxUpdate: func(ctx *EvalContext) {},
			want:      true,
			wantErr:   false,
		},
		{
			name:      "simple true",
			condition: "true",
			ctxUpdate: func(ctx *EvalContext) {},
			want:      true,
			wantErr:   false,
		},
		{
			name:      "simple false",
			condition: "false",
			ctxUpdate: func(ctx *EvalContext) {},
			want:      false,
			wantErr:   false,
		},
		{
			name:      "comparison",
			condition: "vars.env == \"production\"",
			ctxUpdate: func(ctx *EvalContext) {
				ctx.Vars = map[string]interface{}{"env": "production"}
			},
			want:    true,
			wantErr: false,
		},
		{
			name:      "logical and",
			condition: "vars.env == \"production\" && job.status == \"success\"",
			ctxUpdate: func(ctx *EvalContext) {
				ctx.Vars = map[string]interface{}{"env": "production"}
				ctx.Job = map[string]interface{}{"status": "success"}
			},
			want:    true,
			wantErr: false,
		},
		{
			name:      "logical or",
			condition: "vars.env == \"production\" || vars.env == \"staging\"",
			ctxUpdate: func(ctx *EvalContext) {
				ctx.Vars = map[string]interface{}{"env": "staging"}
			},
			want:    true,
			wantErr: false,
		},
		{
			name:      "function call",
			condition: "always()",
			ctxUpdate: func(ctx *EvalContext) {},
			want:      true,
			wantErr:   false,
		},
		{
			name:      "non-bool return type",
			condition: "\"string\"",
			ctxUpdate: func(ctx *EvalContext) {},
			want:      false,
			wantErr:   true,
		},
		{
			name:      "non-bool number",
			condition: "123",
			ctxUpdate: func(ctx *EvalContext) {},
			want:      false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := mockContextBuilder()
			tt.ctxUpdate(ctx)

			got, err := evaluator.Evaluate(tt.condition, ctx)
			if tt.wantErr {
				assert.Error(t, err)
				if exprErr, ok := err.(*ExpressionError); ok {
					assert.Equal(t, "type_error", exprErr.Type)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestConditionEvaluator_ComplexConditions(t *testing.T) {
	engine := NewEngine(1 * time.Second)
	evaluator := NewConditionEvaluator(engine)

	ctx := &EvalContext{
		Vars: map[string]interface{}{
			"env":        "production",
			"notify_all": true,
		},
		Job: map[string]interface{}{
			"status": "success",
		},
	}

	tests := []struct {
		name      string
		condition string
		want      bool
		wantErr   bool
	}{
		{
			name:      "complex and/or",
			condition: `job.status == "success" && (vars.env == "production" || vars.notify_all)`,
			want:      true,
			wantErr:   false,
		},
		{
			name:      "negation",
			condition: `!(job.status == "failure")`,
			want:      true,
			wantErr:   false,
		},
		{
			name:      "multiple conditions",
			condition: `vars.env == "production" && job.status == "success" && vars.notify_all`,
			want:      true,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluator.Evaluate(tt.condition, ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
