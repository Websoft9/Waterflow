package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubmitWorkflow(t *testing.T) {
	tests := []struct {
		name           string
		yamlContent    string
		vars           map[string]interface{}
		serverResponse interface{}
		statusCode     int
		wantErr        bool
		errContains    string
	}{
		{
			name:        "successful submission",
			yamlContent: "name: test\njobs:\n  test:\n    steps:\n      - run: echo test",
			vars: map[string]interface{}{
				"env":    "production",
				"region": "us-east-1",
			},
			serverResponse: &SubmitWorkflowResult{
				ID:        "550e8400-e29b-41d4-a716-446655440000",
				RunID:     "temporal-run-id-123",
				Name:      "test",
				Status:    "running",
				CreatedAt: time.Now(),
				URL:       "/v1/workflows/550e8400-e29b-41d4-a716-446655440000",
			},
			statusCode: http.StatusCreated,
			wantErr:    false,
		},
		{
			name:        "submission without variables",
			yamlContent: "name: simple\njobs:\n  build:\n    steps:\n      - run: make build",
			vars:        nil,
			serverResponse: &SubmitWorkflowResult{
				ID:        "abc-123",
				RunID:     "run-456",
				Name:      "simple",
				Status:    "running",
				CreatedAt: time.Now(),
			},
			statusCode: http.StatusCreated,
			wantErr:    false,
		},
		{
			name:        "server validation error",
			yamlContent: "invalid: yaml",
			vars:        nil,
			serverResponse: map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "validation_error",
					"message": "Workflow validation failed",
					"details": map[string]interface{}{
						"errors": []interface{}{
							map[string]interface{}{
								"field":   "jobs",
								"line":    1,
								"message": "jobs is required",
							},
						},
					},
				},
			},
			statusCode:  http.StatusUnprocessableEntity,
			wantErr:     true,
			errContains: "validation_error",
		},
		{
			name:        "unauthorized error",
			yamlContent: "name: test\njobs:\n  test:\n    steps:\n      - run: test",
			vars:        nil,
			serverResponse: map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "unauthorized",
					"message": "Invalid or missing API key",
				},
			},
			statusCode:  http.StatusUnauthorized,
			wantErr:     true,
			errContains: "unauthorized",
		},
		{
			name:        "server internal error",
			yamlContent: "name: test\njobs:\n  test:\n    steps:\n      - run: test",
			vars:        nil,
			serverResponse: map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "internal_error",
					"message": "Failed to start workflow",
				},
			},
			statusCode:  http.StatusInternalServerError,
			wantErr:     true,
			errContains: "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and path
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/v1/workflows", r.URL.Path)

				// Verify content type
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Decode request body
				var req SubmitWorkflowRequest
				err := json.NewDecoder(r.Body).Decode(&req)
				require.NoError(t, err)

				// Verify request content
				assert.Equal(t, tt.yamlContent, req.YAML)
				if tt.vars != nil {
					assert.Equal(t, tt.vars, req.Vars)
				}

				// Send response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			// Create client
			client := New(server.URL, "", 30*time.Second, false)

			// Submit workflow
			result, err := client.SubmitWorkflow(context.Background(), tt.yamlContent, tt.vars)

			// Check error
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			// Check success
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.NotEmpty(t, result.ID)
			assert.NotEmpty(t, result.RunID)
			assert.NotEmpty(t, result.Name)
			assert.NotEmpty(t, result.Status)
		})
	}
}

func TestGetWorkflowStatus(t *testing.T) {
	tests := []struct {
		name           string
		workflowID     string
		serverResponse interface{}
		statusCode     int
		wantErr        bool
		errContains    string
	}{
		{
			name:       "successful status query",
			workflowID: "550e8400-e29b-41d4-a716-446655440000",
			serverResponse: &WorkflowStatus{
				ID:     "550e8400-e29b-41d4-a716-446655440000",
				Status: "running",
				Jobs: []JobStatus{
					{
						Name:   "build",
						Status: "running",
						Steps: []StepStatus{
							{
								Name:   "compile",
								Status: "running",
							},
						},
					},
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "workflow not found",
			workflowID: "nonexistent",
			serverResponse: map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "not_found",
					"message": "Workflow not found",
				},
			},
			statusCode:  http.StatusNotFound,
			wantErr:     true,
			errContains: "not_found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and path
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Contains(t, r.URL.Path, tt.workflowID)

				// Send response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			// Create client
			client := New(server.URL, "", 30*time.Second, false)

			// Get workflow status
			status, err := client.GetWorkflowStatus(tt.workflowID)

			// Check error
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			// Check success
			require.NoError(t, err)
			require.NotNil(t, status)
			assert.Equal(t, tt.workflowID, status.ID)
			assert.NotEmpty(t, status.Status)
		})
	}
}

func TestServerError(t *testing.T) {
	err := &ServerError{
		StatusCode: 422,
		Code:       "validation_error",
		Message:    "Workflow validation failed",
		Details: map[string]interface{}{
			"errors": []interface{}{
				map[string]interface{}{
					"line":    10,
					"message": "jobs is required",
				},
			},
		},
	}

	errStr := err.Error()
	assert.Contains(t, errStr, "422")
	assert.Contains(t, errStr, "validation_error")
	assert.Contains(t, errStr, "Workflow validation failed")
}
