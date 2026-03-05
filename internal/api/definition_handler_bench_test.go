package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Websoft9/waterflow/pkg/workflow"
	"go.uber.org/zap"
)

// BenchmarkCreateWorkflowDefinition benchmarks the create definition endpoint
func BenchmarkCreateWorkflowDefinition(b *testing.B) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewDefinitionHandlers(logger, mockStore, nil)

	// Setup mock to always succeed
	mockStore.On("Create", nil, nil).Return(nil).Maybe()

	reqBody := CreateWorkflowDefinitionRequest{
		Name:        "benchmark-workflow",
		Description: "Benchmark test",
		Category:    "testing",
		Content: `name: benchmark
vars:
  env: production
  replicas: 3
jobs:
  deploy:
    runs-on: default
    steps:
      - name: Deploy
        run: echo "deploying"`,
	}

	bodyBytes, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/definitions", bytes.NewBuffer(bodyBytes))
		w := httptest.NewRecorder()
		handler.CreateWorkflowDefinition(w, req)
	}
}

// BenchmarkListWorkflowDefinitions benchmarks the list definitions endpoint
func BenchmarkListWorkflowDefinitions(b *testing.B) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewDefinitionHandlers(logger, mockStore, nil)

	// Mock response with 20 definitions
	defs := make([]*workflow.WorkflowDefinition, 20)
	for i := 0; i < 20; i++ {
		defs[i] = &workflow.WorkflowDefinition{
			Name:        "workflow-" + string(rune(i)),
			Description: "Test workflow",
			Category:    "testing",
		}
	}

	mockStore.On("List", nil, workflow.DefinitionFilter{}).
		Return(defs, 20, nil).Maybe()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/v1/workflows/definitions", nil)
		w := httptest.NewRecorder()
		handler.ListWorkflowDefinitions(w, req)
	}
}
