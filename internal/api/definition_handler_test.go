package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/workflow"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockDefinitionStore is a mock implementation of DefinitionStore
type MockDefinitionStore struct {
	mock.Mock
}

func (m *MockDefinitionStore) Create(ctx context.Context, def *workflow.WorkflowDefinition) error {
	args := m.Called(ctx, def)
	return args.Error(0)
}

func (m *MockDefinitionStore) Get(ctx context.Context, name string) (*workflow.WorkflowDefinition, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.WorkflowDefinition), args.Error(1)
}

func (m *MockDefinitionStore) List(ctx context.Context, filter workflow.DefinitionFilter) ([]*workflow.WorkflowDefinition, int, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*workflow.WorkflowDefinition), args.Int(1), args.Error(2)
}

func (m *MockDefinitionStore) Update(ctx context.Context, def *workflow.WorkflowDefinition) error {
	args := m.Called(ctx, def)
	return args.Error(0)
}

func (m *MockDefinitionStore) Delete(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockDefinitionStore) Exists(ctx context.Context, name string) (bool, error) {
	args := m.Called(ctx, name)
	return args.Bool(0), args.Error(1)
}

func TestCreateWorkflowDefinition_Success(t *testing.T) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewDefinitionHandlers(logger, mockStore, nil)
	// Disable semantic validator for test simplicity
	handler.validator = nil
	handler.validator = nil

	// Mock store behavior
	mockStore.On("Create", mock.Anything, mock.AnythingOfType("*workflow.WorkflowDefinition")).
		Run(func(args mock.Arguments) {
			def := args.Get(1).(*workflow.WorkflowDefinition)
			def.CreatedAt = time.Now()
			def.UpdatedAt = time.Now()
		}).
		Return(nil)

	reqBody := CreateWorkflowDefinitionRequest{
		Name:        "test-workflow",
		Description: "Test workflow",
		Category:    "testing",
		Content: `name: test-workflow
on:
  workflow_call:
vars:
  env: production
jobs:
  test:
    runs-on: default
    steps:
      - name: Echo
        uses: exec/shell@v1
        with:
          command: echo "hello"`,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/definitions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateWorkflowDefinition(w, req)

	if w.Code != http.StatusCreated {
		t.Logf("Response body: %s", w.Body.String())
	}
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp CreateWorkflowDefinitionResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "test-workflow", resp.Name)
	assert.Equal(t, "Test workflow", resp.Description)
	assert.Equal(t, "testing", resp.Category)
	assert.NotEmpty(t, resp.CreatedAt)

	mockStore.AssertExpectations(t)
}

func TestCreateWorkflowDefinition_Conflict(t *testing.T) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewDefinitionHandlers(logger, mockStore, nil)
	handler.validator = nil
	handler.validator = nil

	mockStore.On("Create", mock.Anything, mock.AnythingOfType("*workflow.WorkflowDefinition")).
		Return(errors.New("workflow definition 'test-workflow' already exists"))

	reqBody := CreateWorkflowDefinitionRequest{
		Name: "test-workflow",
		Content: `name: test-workflow
on:
  workflow_call:
jobs:
  test:
    runs-on: default
    steps:
      - uses: exec/shell@v1
        with:
          command: echo "test"`,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/definitions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.CreateWorkflowDefinition(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockStore.AssertExpectations(t)
}

func TestGetWorkflowDefinition_Success(t *testing.T) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewDefinitionHandlers(logger, mockStore, nil)
	handler.validator = nil
	handler.validator = nil

	expectedDef := &workflow.WorkflowDefinition{
		Name:        "test-workflow",
		Description: "Test description",
		Category:    "testing",
		Content:     "name: test-workflow\nvars:\n  env: prod",
		ParamsData: []workflow.Parameter{
			{Name: "env", Type: "string", Default: "prod"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockStore.On("Get", mock.Anything, "test-workflow").Return(expectedDef, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/workflows/definitions/test-workflow", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-workflow"})
	w := httptest.NewRecorder()

	handler.GetWorkflowDefinition(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp GetWorkflowDefinitionResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "test-workflow", resp.Name)
	assert.Equal(t, "Test description", resp.Description)
	assert.Equal(t, "testing", resp.Category)
	assert.NotEmpty(t, resp.Content)
	assert.Len(t, resp.Parameters, 1)

	mockStore.AssertExpectations(t)
}

func TestGetWorkflowDefinition_NotFound(t *testing.T) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewDefinitionHandlers(logger, mockStore, nil)
	handler.validator = nil
	handler.validator = nil

	mockStore.On("Get", mock.Anything, "nonexistent").
		Return(nil, errors.New("workflow definition 'nonexistent' not found"))

	req := httptest.NewRequest(http.MethodGet, "/v1/workflows/definitions/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "nonexistent"})
	w := httptest.NewRecorder()

	handler.GetWorkflowDefinition(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockStore.AssertExpectations(t)
}

func TestListWorkflowDefinitions_Success(t *testing.T) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewDefinitionHandlers(logger, mockStore, nil)
	handler.validator = nil
	handler.validator = nil

	expectedDefs := []*workflow.WorkflowDefinition{
		{
			Name:        "workflow-1",
			Description: "First workflow",
			Category:    "testing",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Name:        "workflow-2",
			Description: "Second workflow",
			Category:    "deployment",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	mockStore.On("List", mock.Anything, mock.AnythingOfType("workflow.DefinitionFilter")).
		Return(expectedDefs, 2, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/workflows/definitions?page=1&limit=20", nil)
	w := httptest.NewRecorder()

	handler.ListWorkflowDefinitions(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp ListWorkflowDefinitionsResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Workflows, 2)
	assert.Equal(t, 2, resp.Total)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 20, resp.Limit)

	mockStore.AssertExpectations(t)
}

func TestUpdateWorkflowDefinition_Success(t *testing.T) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewDefinitionHandlers(logger, mockStore, nil)
	handler.validator = nil
	handler.validator = nil

	existingDef := &workflow.WorkflowDefinition{
		Name:        "test-workflow",
		Description: "Old description",
		Category:    "old-category",
		Content:     "name: test\non:\n  workflow_call:\nvars:\n  env: old\njobs:\n  test:\n    runs-on: default\n    steps:\n      - uses: exec/shell@v1\n        with:\n          command: echo",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockStore.On("Get", mock.Anything, "test-workflow").Return(existingDef, nil)
	mockStore.On("Update", mock.Anything, mock.AnythingOfType("*workflow.WorkflowDefinition")).Return(nil)

	reqBody := UpdateWorkflowDefinitionRequest{
		Description: "New description",
		Content: `name: test
on:
  workflow_call:
vars:
  env: new
jobs:
  test:
    runs-on: default
    steps:
      - uses: exec/shell@v1
        with:
          command: echo "updated"`,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/v1/workflows/definitions/test-workflow", bytes.NewBuffer(body))
	req = mux.SetURLVars(req, map[string]string{"name": "test-workflow"})
	w := httptest.NewRecorder()

	handler.UpdateWorkflowDefinition(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockStore.AssertExpectations(t)
}

func TestDeleteWorkflowDefinition_Success(t *testing.T) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewDefinitionHandlers(logger, mockStore, nil)
	handler.validator = nil
	handler.validator = nil

	mockStore.On("Delete", mock.Anything, "test-workflow").Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/v1/workflows/definitions/test-workflow", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-workflow"})
	w := httptest.NewRecorder()

	handler.DeleteWorkflowDefinition(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockStore.AssertExpectations(t)
}

func TestDeleteWorkflowDefinition_NotFound(t *testing.T) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewDefinitionHandlers(logger, mockStore, nil)
	handler.validator = nil
	handler.validator = nil

	mockStore.On("Delete", mock.Anything, "nonexistent").
		Return(errors.New("workflow definition 'nonexistent' not found"))

	req := httptest.NewRequest(http.MethodDelete, "/v1/workflows/definitions/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "nonexistent"})
	w := httptest.NewRecorder()

	handler.DeleteWorkflowDefinition(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockStore.AssertExpectations(t)
}

func TestCreateWorkflowDefinition_InvalidYAML(t *testing.T) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewDefinitionHandlers(logger, mockStore, nil)
	handler.validator = nil

	// Invalid YAML syntax
	reqBody := `{
		"name": "test-workflow",
		"content": "invalid: yaml: syntax: [[[[]"
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/definitions", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.CreateWorkflowDefinition(w, req)

	// Should return 422 for YAML parse error
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Contains(t, errResp, "error")
}

func TestCreateWorkflowDefinition_MissingName(t *testing.T) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewDefinitionHandlers(logger, mockStore, nil)
	handler.validator = nil

	reqBody := `{
		"content": "name: test\njobs:\n  test:\n    runs-on: default\n    steps:\n      - run: echo"
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/definitions", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.CreateWorkflowDefinition(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateWorkflowDefinition_MissingContent(t *testing.T) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewDefinitionHandlers(logger, mockStore, nil)
	handler.validator = nil

	reqBody := `{
		"name": "test-workflow"
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/definitions", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.CreateWorkflowDefinition(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateWorkflowDefinition_InvalidYAML(t *testing.T) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewDefinitionHandlers(logger, mockStore, nil)
	handler.validator = nil

	// Mock Get to return existing definition
	existingDef := &workflow.WorkflowDefinition{
		Name:      "test",
		Content:   "name: test\njobs:\n  test:\n    runs-on: default\n    steps:\n      - run: echo",
		CreatedAt: time.Now().Add(-1 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}
	mockStore.On("Get", mock.Anything, "test").Return(existingDef, nil)

	reqBody := `{
		"content": "invalid: yaml: [[[[]"
	}`

	req := httptest.NewRequest(http.MethodPut, "/v1/workflows/definitions/test", bytes.NewBufferString(reqBody))
	req = mux.SetURLVars(req, map[string]string{"name": "test"})
	w := httptest.NewRecorder()

	handler.UpdateWorkflowDefinition(w, req)

	// Should return 422 for YAML validation error
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
