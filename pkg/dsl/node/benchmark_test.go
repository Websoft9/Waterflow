package node

import (
	"context"
	"testing"
)

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
