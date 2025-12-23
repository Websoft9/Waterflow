package dsl_test

import (
	"os"
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupParser() *dsl.Parser {
	logger, _ := zap.NewDevelopment()
	return dsl.NewParser(logger)
}

func TestParser_Parse_ValidYAML(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		expected func(*testing.T, *dsl.Workflow)
	}{
		{
			name: "simple workflow",
			file: "../../testdata/valid/simple.yaml",
			expected: func(t *testing.T, wf *dsl.Workflow) {
				assert.Equal(t, "Build and Test", wf.Name)
				assert.Equal(t, "push", wf.On)
				assert.Len(t, wf.Jobs, 1)

				build, exists := wf.Jobs["build"]
				require.True(t, exists, "job 'build' should exist")
				assert.Equal(t, "linux-amd64", build.RunsOn)
				assert.Equal(t, 30, build.TimeoutMinutes)
				assert.Len(t, build.Steps, 2)

				// Check first step
				step0 := build.Steps[0]
				assert.Equal(t, "Checkout Code", step0.Name)
				assert.Equal(t, "checkout@v1", step0.Uses)
				assert.Equal(t, "https://github.com/websoft9/waterflow", step0.With["repository"])

				// Check second step
				step1 := build.Steps[1]
				assert.Equal(t, "Run Tests", step1.Name)
				assert.Equal(t, "run@v1", step1.Uses)
				assert.Equal(t, "go test ./...", step1.With["command"])
			},
		},
		{
			name: "multi-job workflow",
			file: "../../testdata/valid/multi-job.yaml",
			expected: func(t *testing.T, wf *dsl.Workflow) {
				assert.Equal(t, "CI/CD Pipeline", wf.Name)
				assert.Len(t, wf.Jobs, 3)

				// Check environment variables
				assert.Equal(t, "1.21", wf.Env["GO_VERSION"])
				assert.Equal(t, "localhost", wf.Env["DB_HOST"])

				// Check job dependencies
				test := wf.Jobs["test"]
				require.NotNil(t, test)
				assert.Equal(t, []string{"build"}, test.Needs)

				deploy := wf.Jobs["deploy"]
				require.NotNil(t, deploy)
				assert.Equal(t, []string{"test"}, deploy.Needs)
			},
		},
	}

	parser := setupParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := os.ReadFile(tt.file)
			require.NoError(t, err)

			workflow, err := parser.Parse(content)
			require.NoError(t, err)
			require.NotNil(t, workflow)

			tt.expected(t, workflow)
		})
	}
}

func TestParser_Parse_InvalidYAML(t *testing.T) {
	tests := []struct {
		name          string
		file          string
		expectedError string
	}{
		{
			name:          "syntax error",
			file:          "../../testdata/invalid/syntax-error.yaml",
			expectedError: "yaml_syntax_error",
		},
	}

	parser := setupParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := os.ReadFile(tt.file)
			require.NoError(t, err)

			workflow, err := parser.Parse(content)
			assert.Nil(t, workflow)
			assert.Error(t, err)

			validationErr, ok := err.(*dsl.ValidationError)
			require.True(t, ok, "error should be ValidationError")
			assert.Equal(t, tt.expectedError, validationErr.Type)
			assert.NotEmpty(t, validationErr.Errors)

			// Check error details
			fieldErr := validationErr.Errors[0]
			assert.Greater(t, fieldErr.Line, 0, "line number should be > 0")
			assert.NotEmpty(t, fieldErr.Error)
		})
	}
}

func TestParser_ExtractLineNumbers(t *testing.T) {
	parser := setupParser()

	content, err := os.ReadFile("../../testdata/valid/simple.yaml")
	require.NoError(t, err)

	workflow, err := parser.Parse(content)
	require.NoError(t, err)

	// Check line map
	assert.NotNil(t, workflow.LineMap)
	assert.Greater(t, len(workflow.LineMap), 0)

	// Verify specific line numbers
	assert.Greater(t, workflow.LineMap["name"], 0)
	assert.Greater(t, workflow.LineMap["jobs"], 0)
	assert.Greater(t, workflow.LineMap["jobs.build"], 0)

	// Verify job line number
	build := workflow.Jobs["build"]
	assert.Greater(t, build.LineNum, 0)

	// Verify step line numbers
	for i, step := range build.Steps {
		assert.Greater(t, step.LineNum, 0, "step %d should have line number", i)
		assert.Equal(t, i, step.Index, "step index should be set")
	}
}

func TestParser_ErrorCodeSnippet(t *testing.T) {
	parser := setupParser()

	content, err := os.ReadFile("../../testdata/invalid/syntax-error.yaml")
	require.NoError(t, err)

	_, err = parser.Parse(content)
	require.Error(t, err)

	validationErr, ok := err.(*dsl.ValidationError)
	require.True(t, ok)
	require.NotEmpty(t, validationErr.Errors)

	fieldErr := validationErr.Errors[0]
	assert.NotEmpty(t, fieldErr.Snippet, "should include code snippet")
	assert.NotEmpty(t, fieldErr.Suggestion, "should include suggestion")

	// Snippet should contain the error line
	assert.Contains(t, fieldErr.Snippet, "→", "should mark error line")
}

func TestParser_InternalFields(t *testing.T) {
	parser := setupParser()

	content, err := os.ReadFile("../../testdata/valid/simple.yaml")
	require.NoError(t, err)

	workflow, err := parser.Parse(content)
	require.NoError(t, err)

	// Check job names are populated
	for jobName, job := range workflow.Jobs {
		assert.Equal(t, jobName, job.Name, "job name should match key")
	}

	// Check step indices are populated
	build := workflow.Jobs["build"]
	for i, step := range build.Steps {
		assert.Equal(t, i, step.Index, "step index should be sequential")
	}
}

func TestParser_RunsOnDefault(t *testing.T) {
	parser := setupParser()

	// YAML without runs-on
	yamlContent := `
name: Test Workflow
on: push
jobs:
  test:
    steps:
      - uses: run@v1
        with:
          command: echo test
`

	workflow, err := parser.Parse([]byte(yamlContent))
	require.NoError(t, err)

	// Check default value is set
	testJob := workflow.Jobs["test"]
	assert.Equal(t, "default", testJob.RunsOn, "runs-on should default to 'default'")
}
