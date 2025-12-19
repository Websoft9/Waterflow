package dsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOutputParser_ParseLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected map[string]string
	}{
		{
			name: "valid set-output",
			line: "::set-output name=version::v1.2.3",
			expected: map[string]string{
				"version": "v1.2.3",
			},
		},
		{
			name: "set-output with spaces",
			line: "::set-output name=commit :: a1b2c3d4 ",
			expected: map[string]string{
				"commit": "a1b2c3d4",
			},
		},
		{
			name: "set-output with complex value",
			line: "::set-output name=artifact::app-v1.2.3.tar.gz",
			expected: map[string]string{
				"artifact": "app-v1.2.3.tar.gz",
			},
		},
		{
			name:     "invalid format",
			line:     "normal output line",
			expected: map[string]string{},
		},
		{
			name:     "partial match",
			line:     "::set-output version::v1.0",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewOutputParser()
			parser.ParseLine(tt.line)
			assert.Equal(t, tt.expected, parser.GetOutputs())
		})
	}
}

func TestOutputParser_ParseOutput(t *testing.T) {
	output := `
Starting build...
::set-output name=version::v1.2.3
Build successful
::set-output name=commit::a1b2c3d4
::set-output name=artifact::app-v1.2.3.tar.gz
Build completed
`

	parser := NewOutputParser()
	outputs := parser.ParseOutput(output)

	expected := map[string]string{
		"version":  "v1.2.3",
		"commit":   "a1b2c3d4",
		"artifact": "app-v1.2.3.tar.gz",
	}

	assert.Equal(t, expected, outputs)
}

func TestOutputParser_MultipleValues(t *testing.T) {
	parser := NewOutputParser()

	parser.ParseLine("::set-output name=key1::value1")
	parser.ParseLine("::set-output name=key2::value2")
	parser.ParseLine("normal output")
	parser.ParseLine("::set-output name=key3::value3")

	expected := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	assert.Equal(t, expected, parser.GetOutputs())
}

func TestOutputParser_OverwriteValue(t *testing.T) {
	parser := NewOutputParser()

	parser.ParseLine("::set-output name=version::v1.0.0")
	parser.ParseLine("::set-output name=version::v2.0.0") // overwrite

	expected := map[string]string{
		"version": "v2.0.0",
	}

	assert.Equal(t, expected, parser.GetOutputs())
}
