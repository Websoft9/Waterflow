package dsl_test

import (
	"os"
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/Websoft9/waterflow/pkg/dsl/node/builtin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSemanticValidator() *dsl.SemanticValidator {
	registry := node.NewRegistry()
	// 测试环境中确保注册成功，失败会导致 panic
	if err := registry.Register(&builtin.CheckoutNode{}); err != nil {
		panic(err)
	}
	if err := registry.Register(&builtin.RunNode{}); err != nil {
		panic(err)
	}
	return dsl.NewSemanticValidator(registry)
}

func TestSemanticValidator_NodeExists(t *testing.T) {
	parser := setupParser()
	validator := setupSemanticValidator()

	content, err := os.ReadFile("../../testdata/valid/simple.yaml")
	require.NoError(t, err)

	workflow, err := parser.Parse(content)
	require.NoError(t, err)

	err = validator.Validate(workflow, content)
	assert.NoError(t, err, "valid workflow should pass semantic validation")
}

func TestSemanticValidator_NodeNotFound(t *testing.T) {
	parser := setupParser()
	validator := setupSemanticValidator()

	content := []byte(`
name: Test
on: push
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - uses: nonexistent@v1
`)

	workflow, err := parser.Parse(content)
	require.NoError(t, err)

	err = validator.Validate(workflow, content)
	require.Error(t, err)

	valErr, ok := err.(*dsl.ValidationError)
	require.True(t, ok)
	assert.Equal(t, "semantic_validation_error", valErr.Type)

	assert.NotEmpty(t, valErr.Errors)
	assert.Contains(t, valErr.Errors[0].Error, "not found")
	assert.Contains(t, valErr.Errors[0].Suggestion, "Available nodes")
}

func TestSemanticValidator_MissingRequiredParam(t *testing.T) {
	parser := setupParser()
	validator := setupSemanticValidator()

	content := []byte(`
name: Test
on: push
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - uses: checkout@v1
`)

	workflow, err := parser.Parse(content)
	require.NoError(t, err)

	err = validator.Validate(workflow, content)
	require.Error(t, err)

	valErr, ok := err.(*dsl.ValidationError)
	require.True(t, ok)

	hasRepositoryError := false
	for _, e := range valErr.Errors {
		if e.Error == "missing required parameter" {
			hasRepositoryError = true
			assert.Contains(t, e.Suggestion, "repository")
			break
		}
	}
	assert.True(t, hasRepositoryError)
}

func TestSemanticValidator_JobDependencyNotFound(t *testing.T) {
	parser := setupParser()
	validator := setupSemanticValidator()

	content := []byte(`
name: Test
on: push
jobs:
  deploy:
    runs-on: linux-amd64
    needs: [nonexistent]
    steps:
      - uses: run@v1
        with:
          command: echo "deploy"
`)

	workflow, err := parser.Parse(content)
	require.NoError(t, err)

	err = validator.Validate(workflow, content)
	require.Error(t, err)

	valErr, ok := err.(*dsl.ValidationError)
	require.True(t, ok)

	hasDepError := false
	for _, e := range valErr.Errors {
		if e.Field == "jobs.deploy.needs" {
			hasDepError = true
			assert.Contains(t, e.Error, "not found")
			break
		}
	}
	assert.True(t, hasDepError)
}

func TestSemanticValidator_CyclicDependency(t *testing.T) {
	parser := setupParser()
	validator := setupSemanticValidator()

	content := []byte(`
name: Test
on: push
jobs:
  a:
    runs-on: linux-amd64
    needs: [b]
    steps:
      - uses: run@v1
        with:
          command: echo "a"
  b:
    runs-on: linux-amd64
    needs: [c]
    steps:
      - uses: run@v1
        with:
          command: echo "b"
  c:
    runs-on: linux-amd64
    needs: [a]
    steps:
      - uses: run@v1
        with:
          command: echo "c"
`)

	workflow, err := parser.Parse(content)
	require.NoError(t, err)

	err = validator.Validate(workflow, content)
	require.Error(t, err)

	valErr, ok := err.(*dsl.ValidationError)
	require.True(t, ok)

	hasCycleError := false
	for _, e := range valErr.Errors {
		if e.Field == "jobs" {
			hasCycleError = true
			assert.Contains(t, e.Error, "cyclic")
			break
		}
	}
	assert.True(t, hasCycleError)
}
