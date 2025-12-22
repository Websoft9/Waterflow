package dsl_test

import (
	"os"
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/Websoft9/waterflow/pkg/dsl/node/builtin"
	"go.uber.org/zap"
)

func BenchmarkValidateSmallWorkflow(b *testing.B) {
	// 1 job, 5 steps, ~100 lines
	content, err := os.ReadFile("../../testdata/benchmark/small.yaml")
	if err != nil {
		b.Fatal(err)
	}

	validator, err := dsl.NewValidator(zap.NewNop())
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validator.ValidateYAML(content)
	}
}

func BenchmarkValidateMediumWorkflow(b *testing.B) {
	// 5 jobs, 50 steps, ~500 lines
	content, err := os.ReadFile("../../testdata/benchmark/medium.yaml")
	if err != nil {
		b.Fatal(err)
	}

	validator, err := dsl.NewValidator(zap.NewNop())
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validator.ValidateYAML(content)
	}
}

func BenchmarkValidateLargeWorkflow(b *testing.B) {
	// 20 jobs, 200 steps, ~2000 lines
	content, err := os.ReadFile("../../testdata/benchmark/large.yaml")
	if err != nil {
		b.Fatal(err)
	}

	validator, err := dsl.NewValidator(zap.NewNop())
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validator.ValidateYAML(content)
	}
}

func BenchmarkParseOnly(b *testing.B) {
	content, err := os.ReadFile("../../testdata/benchmark/medium.yaml")
	if err != nil {
		b.Fatal(err)
	}

	parser := dsl.NewParser(zap.NewNop())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(content)
	}
}

func BenchmarkSchemaValidateOnly(b *testing.B) {
	content, err := os.ReadFile("../../testdata/benchmark/medium.yaml")
	if err != nil {
		b.Fatal(err)
	}

	parser := dsl.NewParser(zap.NewNop())
	workflow, err := parser.Parse(content)
	if err != nil {
		b.Fatal(err)
	}

	validator, err := dsl.NewSchemaValidator()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidateYAML(content, workflow)
	}
}

func BenchmarkSemanticValidateOnly(b *testing.B) {
	content, err := os.ReadFile("../../testdata/benchmark/medium.yaml")
	if err != nil {
		b.Fatal(err)
	}

	parser := dsl.NewParser(zap.NewNop())
	workflow, err := parser.Parse(content)
	if err != nil {
		b.Fatal(err)
	}

	// 直接创建语义验证器
	registry := node.NewRegistry()
	if err := registry.Register(&builtin.CheckoutNode{}); err != nil {
		b.Fatal(err)
	}
	if err := registry.Register(&builtin.RunNode{}); err != nil {
		b.Fatal(err)
	}
	validator := dsl.NewSemanticValidator(registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate(workflow, content)
	}
}
