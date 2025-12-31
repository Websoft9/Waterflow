package node

import (
	"context"
	"fmt"
	"testing"
)

// BenchmarkValidateInputs_10Params benchmarks ValidateInputs with 10 parameters
func BenchmarkValidateInputs_10Params(b *testing.B) {
	specs := make(map[string]ParamSpec)
	inputs := make(map[string]interface{})

	for i := 0; i < 10; i++ {
		paramName := fmt.Sprintf("param%d", i)
		specs[paramName] = ParamSpec{Type: "string", Required: false}
		inputs[paramName] = "value"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateInputs(inputs, specs)
	}
}

// BenchmarkValidateInputs_TypeValidation benchmarks type validation
func BenchmarkValidateInputs_TypeValidation(b *testing.B) {
	specs := map[string]ParamSpec{
		"string_param": {Type: "string", Required: true},
		"int_param":    {Type: "int", Required: true},
		"float_param":  {Type: "float", Required: true},
		"bool_param":   {Type: "bool", Required: true},
		"object_param": {Type: "object", Required: true},
		"array_param":  {Type: "array", Required: true},
	}

	inputs := map[string]interface{}{
		"string_param": "test",
		"int_param":    42,
		"float_param":  3.14,
		"bool_param":   true,
		"object_param": map[string]interface{}{"key": "value"},
		"array_param":  []interface{}{1, 2, 3},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateInputs(inputs, specs)
	}
}

// BenchmarkValidateInputs_PatternValidation benchmarks pattern validation with caching
func BenchmarkValidateInputs_PatternValidation(b *testing.B) {
	specs := map[string]ParamSpec{
		"email": {
			Type:    "string",
			Pattern: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
		},
		"username": {
			Type:    "string",
			Pattern: `^[a-z0-9]+$`,
		},
	}

	inputs := map[string]interface{}{
		"email":    "test@example.com",
		"username": "user123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateInputs(inputs, specs)
	}
}

// BenchmarkValidateInputs_EnumValidation benchmarks enum validation
func BenchmarkValidateInputs_EnumValidation(b *testing.B) {
	specs := map[string]ParamSpec{
		"level": {
			Type: "string",
			Enum: []interface{}{"debug", "info", "warn", "error"},
		},
	}

	inputs := map[string]interface{}{
		"level": "info",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateInputs(inputs, specs)
	}
}

// BenchmarkValidateInputs_RangeValidation benchmarks numeric range validation
func BenchmarkValidateInputs_RangeValidation(b *testing.B) {
	minVal := 1.0
	maxVal := 100.0

	specs := map[string]ParamSpec{
		"timeout": {
			Type:     "int",
			MinValue: &minVal,
			MaxValue: &maxVal,
		},
	}

	inputs := map[string]interface{}{
		"timeout": 50,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateInputs(inputs, specs)
	}
}

// Original benchmarks below

func BenchmarkNodeExecute(b *testing.B) {
	node := &MockNode{
		NameValue:    "flow/benchmark",
		VersionValue: "v1",
		ParamsValue:  map[string]ParamSpec{},
		MetadataValue: NodeMetadata{
			Description:  "Benchmark node",
			Category:     "flow",
			InputSchema:  map[string]ParamSpec{},
			OutputSchema: map[string]interface{}{},
		},
		ExecuteFunc: func(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error) {
			result := NewNodeResult()
			result.SetOutput("value", 42)
			return result, nil
		},
	}

	ctx := context.Background()
	inputs := map[string]interface{}{
		"value": 42,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = node.Execute(ctx, inputs)
	}
}

func BenchmarkParamValidation(b *testing.B) {
	specs := map[string]ParamSpec{
		"command": {
			Type:        "string",
			Required:    true,
			Description: "Command to execute",
		},
		"timeout": {
			Type:     "int",
			Required: false,
			Default:  60,
		},
		"log_level": {
			Type:     "string",
			Required: false,
			Enum:     []interface{}{"debug", "info", "warn", "error"},
		},
	}

	inputs := map[string]interface{}{
		"command":   "echo hello",
		"timeout":   30,
		"log_level": "info",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateInputs(inputs, specs)
	}
}

func BenchmarkMetadata(b *testing.B) {
	node := &MockNode{
		NameValue:    "exec/shell",
		VersionValue: "v1",
		ParamsValue:  map[string]ParamSpec{},
		MetadataValue: NodeMetadata{
			Description: "Shell command execution node",
			Category:    "exec",
			InputSchema: map[string]ParamSpec{
				"command": {Type: "string", Required: true},
				"timeout": {Type: "int", Required: false},
				"env":     {Type: "object", Required: false},
			},
			OutputSchema: map[string]interface{}{
				"stdout":    "string",
				"exit_code": "int",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = node.Metadata()
	}
}

func BenchmarkValidateNode(b *testing.B) {
	node := &MockNode{
		NameValue:    "exec/shell",
		VersionValue: "v1.2.3",
		ParamsValue:  map[string]ParamSpec{},
		MetadataValue: NodeMetadata{
			Description: "Shell command execution",
			Category:    "exec",
			InputSchema: map[string]ParamSpec{
				"command": {Type: "string", Required: true},
			},
			OutputSchema: map[string]interface{}{
				"stdout": "string",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateNode(node)
	}
}

func BenchmarkNodeResult_Operations(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := NewNodeResult()
		result.SetOutput("key1", "value1")
		result.SetOutput("key2", 42)
		result.AddLog("log message")
		result.SetMetadata("meta", "data")
	}
}

func BenchmarkRegistry_Concurrent(b *testing.B) {
	registry := NewRegistry()

	// Pre-populate registry
	for i := 0; i < 100; i++ {
		node := &MockNode{
			NameValue:    "node" + string(rune(i)),
			VersionValue: "v1",
			ParamsValue:  map[string]ParamSpec{},
			MetadataValue: NodeMetadata{
				Description:  "Test",
				Category:     "flow",
				InputSchema:  map[string]ParamSpec{},
				OutputSchema: map[string]interface{}{},
			},
		}
		_ = registry.Register(node)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = registry.Get("node50@v1")
		}
	})
}
