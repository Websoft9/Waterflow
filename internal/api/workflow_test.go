package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Websoft9/waterflow/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupRouter() http.Handler {
	logger, _ := zap.NewDevelopment()
	return api.NewRouter(logger, "v1.0.0", "abc123", "2025-12-19")
}

func TestValidateWorkflow_Success(t *testing.T) {
	router := setupRouter()

	yamlContent := `
name: Test Workflow
on: push
jobs:
  build:
    runs-on: linux-amd64
    steps:
      - uses: checkout@v1
        with:
          repository: https://github.com/websoft9/waterflow
      - uses: run@v1
        with:
          command: go test ./...
`

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/validate", bytes.NewBufferString(yamlContent))
	req.Header.Set("Content-Type", "application/x-yaml")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.True(t, response["valid"].(bool))
	assert.NotNil(t, response["workflow"])
}

func TestValidateWorkflow_SyntaxError(t *testing.T) {
	router := setupRouter()

	yamlContent := `
name: Test
on: push
jobs:
  build
    runs-on: linux-amd64
`

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/validate", bytes.NewBufferString(yamlContent))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "problem+json")

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "Workflow Validation Failed", response["title"])
	assert.NotNil(t, response["errors"])
}

func TestValidateWorkflow_SemanticError(t *testing.T) {
	router := setupRouter()

	yamlContent := `
name: Test
on: push
jobs:
  build:
    runs-on: linux-amd64
    steps:
      - uses: nonexistent@v1
`

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/validate", bytes.NewBufferString(yamlContent))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "Workflow Validation Failed", response["title"])
	errors := response["errors"].([]interface{})
	assert.NotEmpty(t, errors)
}

func TestValidateWorkflow_EmptyBody(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/validate", bytes.NewBufferString(""))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRenderWorkflow_Success(t *testing.T) {
	router := setupRouter()

	yamlContent := `
name: Test Workflow
env:
  VERSION: "1.0.0"
  STAGE: production
on: push
jobs:
  build:
    runs-on: linux-amd64
    steps:
      - uses: run@v1
        with:
          command: echo "Version ${{ env.VERSION }} in ${{ env.STAGE }}"
`

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/render", bytes.NewBufferString(yamlContent))
	req.Header.Set("Content-Type", "application/x-yaml")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.NotNil(t, response["workflow"])
}

func TestRenderWorkflow_ParseError(t *testing.T) {
	router := setupRouter()

	yamlContent := `
name: Test
invalid yaml
`

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/render", bytes.NewBufferString(yamlContent))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRenderWorkflow_EmptyBody(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/render", bytes.NewBufferString(""))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
