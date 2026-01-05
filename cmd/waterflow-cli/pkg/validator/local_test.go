package validator

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLocalValidator_Validate_Success(t *testing.T) {
	// Create validator
	logger := zap.NewNop()
	dslValidator, err := dsl.NewValidator(logger)
	require.NoError(t, err)

	v := NewLocalValidator(dslValidator, logger)

	// Create temp valid workflow file (using registered checkout node)
	content := `name: Test Workflow
on: push
jobs:
  test:
    runs-on: default
    steps:
      - uses: checkout@v1
        with:
          repository: https://github.com/test/repo
`
	tmpFile := createTempYAML(t, content)
	defer os.Remove(tmpFile)

	// Validate
	result, err := v.Validate(tmpFile)

	// Assert
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, tmpFile, result.File)
	assert.Equal(t, "Test Workflow", result.WorkflowName)
	assert.Equal(t, 1, result.JobsCount)
	assert.Equal(t, 1, result.StepsCount)
	assert.Empty(t, result.Errors)
	assert.Greater(t, result.ValidationTime, time.Duration(0))
}

func TestLocalValidator_Validate_SyntaxError(t *testing.T) {
	logger := zap.NewNop()
	dslValidator, err := dsl.NewValidator(logger)
	require.NoError(t, err)

	v := NewLocalValidator(dslValidator, logger)

	// Create temp file with syntax error
	content := `name: Invalid
on: push
jobs:
  build
    runs-on: linux
`
	tmpFile := createTempYAML(t, content)
	defer os.Remove(tmpFile)

	// Validate
	result, err := v.Validate(tmpFile)

	// Assert
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
	assert.Equal(t, "yaml_syntax_error", result.Errors[0].Type)
}

func TestLocalValidator_Validate_FileNotFound(t *testing.T) {
	logger := zap.NewNop()
	dslValidator, err := dsl.NewValidator(logger)
	require.NoError(t, err)

	v := NewLocalValidator(dslValidator, logger)

	// Validate non-existent file
	result, err := v.Validate("nonexistent.yaml")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "file not found")
}

func TestLocalValidator_Validate_EmptyFile(t *testing.T) {
	logger := zap.NewNop()
	dslValidator, err := dsl.NewValidator(logger)
	require.NoError(t, err)

	v := NewLocalValidator(dslValidator, logger)

	// Create empty file
	tmpFile := createTempYAML(t, "")
	defer os.Remove(tmpFile)

	// Validate
	result, err := v.Validate(tmpFile)

	// Assert
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, "file_error", result.Errors[0].Type)
	assert.Contains(t, result.Errors[0].Message, "Empty YAML file")
	assert.Contains(t, result.Errors[0].Suggestion, "Add workflow definition")
}

func TestLocalValidator_Validate_FileTooLarge(t *testing.T) {
	logger := zap.NewNop()
	dslValidator, err := dsl.NewValidator(logger)
	require.NoError(t, err)

	v := NewLocalValidator(dslValidator, logger)

	// Create large file (11MB)
	tmpFile := filepath.Join(t.TempDir(), "large.yaml")
	largeContent := make([]byte, 11*1024*1024)
	err = os.WriteFile(tmpFile, largeContent, 0644)
	require.NoError(t, err)
	defer os.Remove(tmpFile)

	// Validate
	result, err := v.Validate(tmpFile)

	// Assert
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, "file_error", result.Errors[0].Type)
	assert.Contains(t, result.Errors[0].Message, "exceeds limit")
	assert.Contains(t, result.Errors[0].Suggestion, "Split large workflow")
}

func TestLocalValidator_Validate_SchemaValidationError(t *testing.T) {
	logger := zap.NewNop()
	dslValidator, err := dsl.NewValidator(logger)
	require.NoError(t, err)

	v := NewLocalValidator(dslValidator, logger)

	// Create file with schema errors
	content := `name: Missing Required
on: push
jobs:
  build:
    runs-on: linux
`
	tmpFile := createTempYAML(t, content)
	defer os.Remove(tmpFile)

	// Validate
	result, err := v.Validate(tmpFile)

	// Assert
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

func TestLocalValidator_CountSteps(t *testing.T) {
	logger := zap.NewNop()
	dslValidator, err := dsl.NewValidator(logger)
	require.NoError(t, err)

	v := NewLocalValidator(dslValidator, logger)

	tests := []struct {
		name     string
		workflow *dsl.Workflow
		expected int
	}{
		{
			name: "single job single step",
			workflow: &dsl.Workflow{
				Jobs: map[string]*dsl.Job{
					"test": {Steps: []*dsl.Step{{}}},
				},
			},
			expected: 1,
		},
		{
			name: "single job multiple steps",
			workflow: &dsl.Workflow{
				Jobs: map[string]*dsl.Job{
					"test": {Steps: []*dsl.Step{{}, {}, {}}},
				},
			},
			expected: 3,
		},
		{
			name: "multiple jobs multiple steps",
			workflow: &dsl.Workflow{
				Jobs: map[string]*dsl.Job{
					"build": {Steps: []*dsl.Step{{}, {}}},
					"test":  {Steps: []*dsl.Step{{}, {}, {}}},
				},
			},
			expected: 5,
		},
		{
			name: "no steps",
			workflow: &dsl.Workflow{
				Jobs: map[string]*dsl.Job{
					"test": {Steps: []*dsl.Step{}},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := v.countSteps(tt.workflow)
			assert.Equal(t, tt.expected, count)
		})
	}
}

func TestLocalValidator_ConvertValidationError(t *testing.T) {
	logger := zap.NewNop()
	dslValidator, err := dsl.NewValidator(logger)
	require.NoError(t, err)

	v := NewLocalValidator(dslValidator, logger)

	tests := []struct {
		name     string
		inputErr error
		wantType string
		wantLen  int
	}{
		{
			name: "DSL ValidationError",
			inputErr: &dsl.ValidationError{
				Type: "schema_validation_error",
				Errors: []dsl.FieldError{
					{
						Field:      "jobs.test.steps",
						Line:       10,
						Error:      "required field missing",
						Suggestion: "Add steps",
					},
				},
			},
			wantType: "schema_validation_error",
			wantLen:  1,
		},
		{
			name:     "unknown error",
			inputErr: assert.AnError,
			wantType: "unknown_error",
			wantLen:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.convertValidationError("test.yaml", tt.inputErr, 100*time.Millisecond)

			assert.False(t, result.Valid)
			assert.Equal(t, "test.yaml", result.File)
			assert.Len(t, result.Errors, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, tt.wantType, result.Errors[0].Type)
			}
			assert.Equal(t, 100*time.Millisecond, result.ValidationTime)
		})
	}
}

// createTempYAML creates a temporary YAML file for testing
func createTempYAML(t *testing.T, content string) string {
	tmpFile := filepath.Join(t.TempDir(), "test.yaml")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)
	return tmpFile
}
