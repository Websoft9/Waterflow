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

// BenchmarkExecuteFromDefinition benchmarks executing workflow from definition
func BenchmarkExecuteFromDefinition(b *testing.B) {
	logger := zap.NewNop()
	mockStore := new(MockDefinitionStore)
	handler := NewExecutionHandlers(logger, nil, mockStore, nil)

	// Mock definition
	def := &workflow.WorkflowDefinition{
		Name:    "benchmark-workflow",
		Content: testWorkflowYAML,
	}
	mockStore.On("Get", nil, "benchmark-workflow").Return(def, nil).Maybe()

	reqBody := ExecuteWorkflowDefinitionRequest{
		Vars: map[string]interface{}{
			"env": "staging",
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/benchmark-workflow/run", bytes.NewBuffer(bodyBytes))
		w := httptest.NewRecorder()
		handler.ExecuteWorkflowDefinition(w, req)
	}
}

// BenchmarkDirectExecute benchmarks direct workflow execution
func BenchmarkDirectExecute(b *testing.B) {
	logger := zap.NewNop()
	handler := NewExecutionHandlers(logger, nil, nil, nil)

	reqBody := DirectExecuteWorkflowRequest{
		Workflow: testWorkflowYAML,
		Vars: map[string]interface{}{
			"env": "staging",
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/executions", bytes.NewBuffer(bodyBytes))
		w := httptest.NewRecorder()
		handler.DirectExecuteWorkflow(w, req)
	}
}

const testWorkflowYAML = `name: benchmark-test
on:
  workflow_call:
vars:
  env: production
  replicas: 3
jobs:
  deploy:
    runs-on: default
    steps:
      - name: Deploy
        run: echo "deploying"`
