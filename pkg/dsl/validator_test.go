package dsl_test

import (
	"os"
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupValidator() *dsl.Validator {
	logger, _ := zap.NewDevelopment()
	validator, err := dsl.NewValidator(logger)
	if err != nil {
		panic(err)
	}
	return validator
}

func TestValidator_ValidYAML(t *testing.T) {
	validator := setupValidator()

	content, err := os.ReadFile("../../testdata/valid/simple.yaml")
	require.NoError(t, err)

	workflow, err := validator.ValidateYAML(content)
	require.NoError(t, err)
	assert.NotNil(t, workflow)
	assert.Equal(t, "Build and Test", workflow.Name)
}

func TestValidator_SyntaxError(t *testing.T) {
	validator := setupValidator()

	content, err := os.ReadFile("../../testdata/invalid/syntax-error.yaml")
	require.NoError(t, err)

	workflow, err := validator.ValidateYAML(content)
	assert.Nil(t, workflow)
	assert.Error(t, err)

	valErr, ok := err.(*dsl.ValidationError)
	require.True(t, ok)
	assert.Equal(t, "yaml_syntax_error", valErr.Type)
}

func TestValidator_SchemaError(t *testing.T) {
	validator := setupValidator()

	content, err := os.ReadFile("../../testdata/invalid/missing-required.yaml")
	require.NoError(t, err)

	workflow, err := validator.ValidateYAML(content)
	assert.Nil(t, workflow)
	assert.Error(t, err)

	valErr, ok := err.(*dsl.ValidationError)
	require.True(t, ok)
	assert.Contains(t, valErr.Type, "validation_error")
	assert.NotEmpty(t, valErr.Errors)
}

func TestValidator_SemanticError(t *testing.T) {
	validator := setupValidator()

	content := []byte(`
name: Test
on: push
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - uses: nonexistent@v1
`)

	workflow, err := validator.ValidateYAML(content)
	assert.Nil(t, workflow)
	assert.Error(t, err)

	valErr, ok := err.(*dsl.ValidationError)
	require.True(t, ok)
	assert.NotEmpty(t, valErr.Errors)
}

func TestValidator_ErrorLimit(t *testing.T) {
	validator := setupValidator()

	// Create workflow with multiple errors
	content := []byte(`
name: Test
on: push
jobs:
  test:
    steps:
      - uses: nonexistent1@v1
      - uses: nonexistent2@v1
      - uses: nonexistent3@v1
      - uses: nonexistent4@v1
      - uses: nonexistent5@v1
`)

	workflow, err := validator.ValidateYAML(content)
	assert.Nil(t, workflow)
	require.Error(t, err)

	valErr, ok := err.(*dsl.ValidationError)
	require.True(t, ok)
	// Errors should be limited to 20
	assert.LessOrEqual(t, len(valErr.Errors), 20)
}
