package dsl_test

import (
	"os"
	"strings"
	"testing"
	"time"

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
	// 20 jobs, 200 steps, ~2000 lines (验证 AC7 性能要求 <700ms)
	content, err := os.ReadFile("../../testdata/benchmark/large.yaml")
	if err != nil {
		b.Fatal(err)
	}

	validator, err := dsl.NewValidator(zap.NewNop())
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	// 记录时间验证是否满足 AC7 要求
	start := time.Now()
	for i := 0; i < b.N; i++ {
		_, _ = validator.ValidateYAML(content)
	}
	elapsed := time.Since(start)

	// 单次验证时间应 <700ms (AC7 要求)
	avgTime := elapsed / time.Duration(b.N)
	if avgTime > 700*time.Millisecond {
		b.Errorf("Large workflow validation too slow: %v (expected <700ms per AC7)", avgTime)
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

// BenchmarkValidate1000LineWorkflow tests parsing performance for 1000+ line YAML
// AC2: 1000 行 YAML < 100ms
func BenchmarkValidate1000LineWorkflow(b *testing.B) {
	// xlarge.yaml: 1522 lines, 20 jobs, 200 steps
	content, err := os.ReadFile("../../testdata/benchmark/xlarge.yaml")
	if err != nil {
		b.Fatal(err)
	}

	lineCount := len(strings.Split(string(content), "\n"))
	if lineCount < 1000 {
		b.Fatalf("Test file should have ~1000 lines, got %d", lineCount)
	}

	b.Logf("Testing with %d lines", lineCount)

	validator, err := dsl.NewValidator(zap.NewNop())
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	var totalTime time.Duration
	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, err := validator.ValidateYAML(content)
		if err != nil {
			b.Fatal(err)
		}
		totalTime += time.Since(start)
	}

	// Verify performance target
	avgTime := totalTime / time.Duration(b.N)

	if avgTime > 100*time.Millisecond {
		b.Logf("WARNING: Average parse time %v exceeds 100ms target", avgTime)
	}

	b.Logf("Average time: %v (target < 100ms)", avgTime)
}

// BenchmarkValidateMemory tests memory allocation during validation
// AC2: Memory allocation < 10MB
func BenchmarkValidateMemory(b *testing.B) {
	content, err := os.ReadFile("../../testdata/benchmark/large.yaml")
	if err != nil {
		b.Fatal(err)
	}

	validator, err := dsl.NewValidator(zap.NewNop())
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = validator.ValidateYAML(content)
	}

	// Note: memory metrics are reported automatically by -benchmem flag
	// Look for B/op (bytes per operation) in the output
	// Target: < 10MB (10485760 bytes) per operation
}

// BenchmarkParse1000Lines tests pure YAML parsing performance
func BenchmarkParse1000Lines(b *testing.B) {
	content, err := os.ReadFile("../../testdata/benchmark/xlarge.yaml")
	if err != nil {
		b.Fatal(err)
	}

	parser := dsl.NewParser(zap.NewNop())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(content)
	}
}
