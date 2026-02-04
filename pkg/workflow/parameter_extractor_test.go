package workflow

import (
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/stretchr/testify/assert"
)

func TestParameterExtractor_ExtractParameters(t *testing.T) {
	extractor := NewParameterExtractor()

	tests := []struct {
		name     string
		workflow *dsl.Workflow
		expected []Parameter
	}{
		{
			name: "extract from vars",
			workflow: &dsl.Workflow{
				Name: "test",
				Vars: map[string]interface{}{
					"env":     "production",
					"version": "1.0.0",
					"count":   3,
				},
			},
			expected: []Parameter{
				{Name: "env", Type: "string", Default: "production"},
				{Name: "version", Type: "string", Default: "1.0.0"},
				{Name: "count", Type: "number", Default: 3},
			},
		},
		{
			name: "extract from expressions",
			workflow: &dsl.Workflow{
				Name: "deploy",
				Jobs: map[string]*dsl.Job{
					"deploy": {
						RunsOn: "${{ vars.agent }}",
						Steps: []*dsl.Step{
							{
								Name: "Deploy",
								Uses: "deploy@v1",
								With: map[string]interface{}{
									"env": "${{ vars.environment }}",
								},
							},
						},
					},
				},
			},
			expected: []Parameter{
				{Name: "agent", Type: "string"},
				{Name: "environment", Type: "string"},
			},
		},
		{
			name: "mixed vars and expressions",
			workflow: &dsl.Workflow{
				Name: "test",
				Vars: map[string]interface{}{
					"defined_var": "value",
				},
				Jobs: map[string]*dsl.Job{
					"test": {
						RunsOn: "${{ vars.agent }}",
					},
				},
			},
			expected: []Parameter{
				{Name: "defined_var", Type: "string", Default: "value"},
				{Name: "agent", Type: "string"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := extractor.ExtractParameters(tt.workflow)

			// Convert to map for easier comparison
			paramMap := make(map[string]Parameter)
			for _, p := range params {
				paramMap[p.Name] = p
			}

			expectedMap := make(map[string]Parameter)
			for _, p := range tt.expected {
				expectedMap[p.Name] = p
			}

			assert.Equal(t, len(expectedMap), len(paramMap), "parameter count mismatch")

			for name, expected := range expectedMap {
				actual, ok := paramMap[name]
				assert.True(t, ok, "parameter %s not found", name)
				assert.Equal(t, expected.Type, actual.Type, "type mismatch for %s", name)
				if expected.Default != nil {
					assert.Equal(t, expected.Default, actual.Default, "default mismatch for %s", name)
				}
			}
		})
	}
}

func TestInferType(t *testing.T) {
	tests := []struct {
		value    interface{}
		expected string
	}{
		{value: "string", expected: "string"},
		{value: 123, expected: "number"},
		{value: 45.67, expected: "number"},
		{value: true, expected: "boolean"},
		{value: []interface{}{1, 2, 3}, expected: "array"},
		{value: map[string]interface{}{"key": "value"}, expected: "object"},
		{value: nil, expected: "string"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := inferType(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeVarName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "${{ vars.env }}", expected: "env"},
		{input: "${{vars.version}}", expected: "version"},
		{input: "  ${{ vars.name }}  ", expected: "name"},
		{input: "simple_name", expected: "simple_name"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeVarName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
