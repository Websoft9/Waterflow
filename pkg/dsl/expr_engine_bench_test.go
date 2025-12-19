package dsl

import (
	"testing"
	"time"
)

func BenchmarkEngineEvaluate_SimpleVariable(b *testing.B) {
	engine := NewEngine(1 * time.Second)

	ctx := mockContextBuilder()
	ctx.Vars = map[string]interface{}{
		"env": "production",
	}

	expression := "vars.env"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Evaluate(expression, ctx)
	}
}

func BenchmarkEngineEvaluate_Arithmetic(b *testing.B) {
	engine := NewEngine(1 * time.Second)

	ctx := mockContextBuilder()

	expression := "1 + 2 * 3"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Evaluate(expression, ctx)
	}
}

func BenchmarkEngineEvaluate_ComplexExpression(b *testing.B) {
	engine := NewEngine(1 * time.Second)

	ctx := mockContextBuilder()
	ctx.Vars = map[string]interface{}{
		"image":   "myapp",
		"version": "v1.2.3",
	}

	expression := `format("{0}:{1}", vars.image, vars.version)`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Evaluate(expression, ctx)
	}
}

func BenchmarkEngineEvaluate_NestedFunction(b *testing.B) {
	engine := NewEngine(1 * time.Second)

	ctx := mockContextBuilder()
	ctx.Vars = map[string]interface{}{
		"env": "production",
	}

	expression := `upper(format("env: {0}", vars.env))`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Evaluate(expression, ctx)
	}
}

func BenchmarkExpressionReplacer_Replace(b *testing.B) {
	engine := NewEngine(1 * time.Second)
	replacer := NewExpressionReplacer(engine)

	ctx := mockContextBuilder()
	ctx.Vars = map[string]interface{}{
		"app":     "MyApp",
		"version": "v1.2.3",
	}

	input := "${{ vars.app }} v${{ vars.version }}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = replacer.Replace(input, ctx)
	}
}

func BenchmarkExpressionReplacer_ReplaceInMap(b *testing.B) {
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
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = replacer.ReplaceInMap(input, ctx)
	}
}

func BenchmarkConditionEvaluator_Simple(b *testing.B) {
	engine := NewEngine(1 * time.Second)
	evaluator := NewConditionEvaluator(engine)

	ctx := mockContextBuilder()
	ctx.Vars = map[string]interface{}{
		"env": "production",
	}

	condition := `vars.env == "production"`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate(condition, ctx)
	}
}

func BenchmarkConditionEvaluator_Complex(b *testing.B) {
	engine := NewEngine(1 * time.Second)
	evaluator := NewConditionEvaluator(engine)

	ctx := mockContextBuilder()
	ctx.Vars = map[string]interface{}{
		"env":        "production",
		"notify_all": true,
	}
	ctx.Job = map[string]interface{}{
		"status": "success",
	}

	condition := `job.status == "success" && (vars.env == "production" || vars.notify_all)`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate(condition, ctx)
	}
}
