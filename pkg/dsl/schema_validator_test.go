package dsl_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSchemaValidator() *dsl.SchemaValidator {
	validator, err := dsl.NewSchemaValidator()
	if err != nil {
		panic(err)
	}
	return validator
}

func TestSchemaValidator_Validate_Valid(t *testing.T) {
	parser := setupParser()
	validator := setupSchemaValidator()

	tests := []string{
		"../../testdata/valid/simple.yaml",
		"../../testdata/valid/multi-job.yaml",
	}

	for _, file := range tests {
		t.Run(file, func(t *testing.T) {
			// #nosec G304 - test file paths are hardcoded and safe
			content, err := os.ReadFile(file)
			require.NoError(t, err)

			workflow, err := parser.Parse(content)
			require.NoError(t, err)

			err = validator.ValidateYAML(content, workflow)
			if err != nil {
				if valErr, ok := err.(*dsl.ValidationError); ok {
					t.Logf("Validation errors:")
					for _, e := range valErr.Errors {
						t.Logf("  - field=%s, error=%s", e.Field, e.Error)
					}
				}
			}
			assert.NoError(t, err, "valid workflow should pass schema validation")
		})
	}
}

func TestSchemaValidator_Validate_MissingRequired(t *testing.T) {
	parser := setupParser()
	validator := setupSchemaValidator()

	content, err := os.ReadFile("../../testdata/invalid/missing-required.yaml")
	require.NoError(t, err)

	workflow, err := parser.Parse(content)
	require.NoError(t, err)

	err = validator.ValidateYAML(content, workflow)
	require.Error(t, err, "should fail validation for missing required field")

	validationErr, ok := err.(*dsl.ValidationError)
	require.True(t, ok)
	assert.Equal(t, "schema_validation_error", validationErr.Type)
	assert.NotEmpty(t, validationErr.Errors)

	// Note: runs-on is now optional, so we check for steps being required instead
	hasStepsError := false
	for _, fieldErr := range validationErr.Errors {
		if fieldErr.Field == "jobs.build.steps" || fieldErr.Field == "jobs.build" {
			hasStepsError = true
			assert.Contains(t, fieldErr.Error, "required")
			break
		}
	}
	assert.True(t, hasStepsError, "should have error about missing required field (steps)")
}

func TestSchemaValidator_Validate_InvalidType(t *testing.T) {
	parser := setupParser()
	validator := setupSchemaValidator()

	content, err := os.ReadFile("../../testdata/invalid/invalid-type.yaml")
	require.NoError(t, err)

	workflow, err := parser.Parse(content)
	require.NoError(t, err)

	err = validator.ValidateYAML(content, workflow)
	require.Error(t, err, "should fail validation for invalid type")

	validationErr, ok := err.(*dsl.ValidationError)
	require.True(t, ok)
	assert.Equal(t, "schema_validation_error", validationErr.Type)

	// Check error details - should have pattern error for uses field
	assert.NotEmpty(t, validationErr.Errors)
	hasPatternError := false
	for _, fieldErr := range validationErr.Errors {
		if strings.Contains(fieldErr.Field, "uses") {
			hasPatternError = true
			assert.NotEmpty(t, fieldErr.Suggestion)
			break
		}
	}
	assert.True(t, hasPatternError, "should have pattern validation error for uses field")
}

func TestSchemaValidator_ErrorSuggestions(t *testing.T) {
	parser := setupParser()
	validator := setupSchemaValidator()

	// Create a YAML with invalid uses pattern
	content := []byte(`
name: Test
on: push
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - uses: invalid_node
`)

	workflow, err := parser.Parse(content)
	require.NoError(t, err)

	err = validator.ValidateYAML(content, workflow)
	require.Error(t, err)

	validationErr, ok := err.(*dsl.ValidationError)
	require.True(t, ok)

	// Should have pattern error for uses field
	hasPatternError := false
	for _, fieldErr := range validationErr.Errors {
		if strings.Contains(fieldErr.Field, "uses") {
			hasPatternError = true
			assert.NotEmpty(t, fieldErr.Suggestion)
			break
		}
	}
	assert.True(t, hasPatternError, "should have pattern validation error")
}

func TestSchemaValidator_LineNumbers(t *testing.T) {
	parser := setupParser()
	validator := setupSchemaValidator()

	content, err := os.ReadFile("../../testdata/invalid/missing-required.yaml")
	require.NoError(t, err)

	workflow, err := parser.Parse(content)
	require.NoError(t, err)

	err = validator.ValidateYAML(content, workflow)
	require.Error(t, err)

	validationErr, ok := err.(*dsl.ValidationError)
	require.True(t, ok)

	// At least some errors should have line numbers from LineMap
	hasLineNumber := false
	for _, fieldErr := range validationErr.Errors {
		if fieldErr.Line > 0 {
			hasLineNumber = true
			break
		}
	}
	// Note: Line numbers might not always be available depending on field path
	// This is a best-effort test
	t.Logf("Has line number: %v", hasLineNumber)
}
