package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGreeterNode_Name(t *testing.T) {
	n := &GreeterNode{}
	assert.Equal(t, "custom/greeter", n.Name())
}

func TestGreeterNode_Version(t *testing.T) {
	n := &GreeterNode{}
	assert.Equal(t, "v1", n.Version())
}

func TestGreeterNode_Params(t *testing.T) {
	n := &GreeterNode{}
	params := n.Params()

	assert.True(t, params["name"].Required)
	assert.Equal(t, "string", params["name"].Type)
	assert.False(t, params["language"].Required)
	assert.Equal(t, "en", params["language"].Default)
	assert.ElementsMatch(t, []interface{}{"en", "zh", "es", "fr"}, params["language"].Enum)
}

func TestGreeterNode_Execute_English(t *testing.T) {
	n := &GreeterNode{}
	inputs := map[string]interface{}{
		"name":        "Alice",
		"language":    "en",
		"time_of_day": "morning",
	}

	result, err := n.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Equal(t, "Good morning, Alice!", result.Outputs["greeting"])
	assert.Equal(t, "en", result.Outputs["language"])
	assert.Equal(t, "morning", result.Outputs["time_of_day"])
	assert.GreaterOrEqual(t, result.Duration.Nanoseconds(), int64(0))
}

func TestGreeterNode_Execute_Chinese(t *testing.T) {
	n := &GreeterNode{}
	inputs := map[string]interface{}{
		"name":        "小明",
		"language":    "zh",
		"time_of_day": "evening",
	}

	result, err := n.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Equal(t, "晚上好,小明!", result.Outputs["greeting"])
}

func TestGreeterNode_Execute_Spanish(t *testing.T) {
	n := &GreeterNode{}
	inputs := map[string]interface{}{
		"name":        "Maria",
		"language":    "es",
		"time_of_day": "afternoon",
	}

	result, err := n.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Equal(t, "Buenas tardes, Maria!", result.Outputs["greeting"])
}

func TestGreeterNode_Execute_AutoDetectTime(t *testing.T) {
	n := &GreeterNode{}
	inputs := map[string]interface{}{
		"name":        "Bob",
		"time_of_day": "auto",
	}

	result, err := n.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.NotEmpty(t, result.Outputs["greeting"])
	assert.Contains(t, []string{"morning", "afternoon", "evening"}, result.Outputs["time_of_day"])
}

func TestGreeterNode_Execute_MissingRequiredParam(t *testing.T) {
	n := &GreeterNode{}
	inputs := map[string]interface{}{
		"language": "en",
	}

	result, err := n.Execute(context.Background(), inputs)

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required parameter 'name'")
}

func TestGreeterNode_Execute_DefaultLanguage(t *testing.T) {
	n := &GreeterNode{}
	inputs := map[string]interface{}{
		"name": "Alice",
	}

	result, err := n.Execute(context.Background(), inputs)

	require.NoError(t, err)
	assert.Equal(t, "en", result.Outputs["language"])
	assert.NotEmpty(t, result.Outputs["greeting"])
}

func TestGreeterNode_Execute_InvalidEnum(t *testing.T) {
	n := &GreeterNode{}
	inputs := map[string]interface{}{
		"name":     "Alice",
		"language": "de",
	}

	result, err := n.Execute(context.Background(), inputs)

	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestGreeterNode_Metadata(t *testing.T) {
	n := &GreeterNode{}
	metadata := n.Metadata()

	assert.Equal(t, "Generates multi-language greetings based on time of day", metadata.Description)
	assert.Equal(t, "custom", metadata.Category)
	assert.NotEmpty(t, metadata.InputSchema)
	assert.NotEmpty(t, metadata.OutputSchema)
}

func TestRegister(t *testing.T) {
	node := Register()
	require.NotNil(t, node)
	assert.Equal(t, "custom/greeter", node.Name())
	assert.Equal(t, "v1", node.Version())
}

func TestGenerateGreeting_AllCombinations(t *testing.T) {
	testCases := []struct {
		name      string
		language  string
		timeOfDay string
		expected  string
	}{
		{"Alice", "en", "morning", "Good morning, Alice!"},
		{"Bob", "en", "afternoon", "Good afternoon, Bob!"},
		{"Charlie", "en", "evening", "Good evening, Charlie!"},
		{"小明", "zh", "morning", "早上好,小明!"},
		{"小红", "zh", "afternoon", "下午好,小红!"},
		{"小强", "zh", "evening", "晚上好,小强!"},
		{"María", "es", "morning", "Buenos días, María!"},
		{"José", "es", "afternoon", "Buenas tardes, José!"},
		{"Pedro", "es", "evening", "Buenas noches, Pedro!"},
		{"Jean", "fr", "morning", "Bonjour, Jean!"},
		{"Marie", "fr", "afternoon", "Bon après-midi, Marie!"},
		{"Pierre", "fr", "evening", "Bonsoir, Pierre!"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			actual := generateGreeting(tc.name, tc.language, tc.timeOfDay)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

// BenchmarkGreeterNode_Execute 性能基准测试
func BenchmarkGreeterNode_Execute(b *testing.B) {
	n := &GreeterNode{}
	inputs := map[string]interface{}{
		"name":        "Benchmark",
		"language":    "en",
		"time_of_day": "morning",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = n.Execute(context.Background(), inputs)
	}
}
