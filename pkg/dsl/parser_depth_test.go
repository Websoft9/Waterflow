package dsl_test

import (
	"strings"
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestParser_MaxDepthCheck(t *testing.T) {
	parser := dsl.NewParser(zap.NewNop())

	// 构造一个嵌套深度为 25 层的 YAML (超过限制 20 层)
	var builder strings.Builder
	builder.WriteString("name: Deep Nesting Test\njobs:\n  test:\n    runs-on: default\n    steps:\n      - uses: run@v1\n        with:\n")

	// 创建深层嵌套
	indent := "          "
	for i := 0; i < 25; i++ {
		builder.WriteString(indent)
		builder.WriteString("level")
		builder.WriteString(strings.Repeat(string(rune('0'+i%10)), 1))
		builder.WriteString(":\n")
		indent += "  "
	}
	builder.WriteString(indent)
	builder.WriteString("value: deep\n")

	content := []byte(builder.String())

	_, err := parser.Parse(content)
	require.Error(t, err, "Should reject YAML with depth > 20")

	valErr, ok := err.(*dsl.ValidationError)
	require.True(t, ok)
	assert.Equal(t, "yaml_syntax_error", valErr.Type)
	assert.Contains(t, valErr.Errors[0].Error, "nesting depth")
	assert.Contains(t, valErr.Errors[0].Error, "exceeds limit")
}

func TestParser_AcceptableDepth(t *testing.T) {
	parser := dsl.NewParser(zap.NewNop())

	// 构造一个嵌套深度为 15 层的 YAML (在限制内)
	var builder strings.Builder
	builder.WriteString("name: Acceptable Depth\njobs:\n  test:\n    runs-on: default\n    steps:\n      - uses: run@v1\n        with:\n")

	indent := "          "
	for i := 0; i < 10; i++ {
		builder.WriteString(indent)
		builder.WriteString("level")
		builder.WriteString(strings.Repeat(string(rune('0'+i)), 1))
		builder.WriteString(":\n")
		indent += "  "
	}
	builder.WriteString(indent)
	builder.WriteString("value: acceptable\n")

	content := []byte(builder.String())

	workflow, err := parser.Parse(content)
	require.NoError(t, err, "Should accept YAML with depth <= 20")
	assert.NotNil(t, workflow)
	assert.Equal(t, "Acceptable Depth", workflow.Name)
}
