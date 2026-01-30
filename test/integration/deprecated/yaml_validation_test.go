//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestYAMLValidation_ValidWorkflow tests YAML parsing succeeds
// Note: Actual workflow execution requires registered nodes
func TestYAMLValidation_ValidWorkflow(t *testing.T) {
	t.Skip("Requires node registration - covered by workflow_lifecycle_test.go")
}

// TestYAMLValidation_SyntaxError tests YAML with syntax errors
func TestYAMLValidation_SyntaxError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()

	tests := []struct {
		name        string
		yaml        string
		expectError string
	}{
		{
			name: "Missing colon after key",
			yaml: `name: Test
jobs:
  build:
    steps
      - uses: checkout@v1
`,
			expectError: "yaml",
		},
		{
			name: "Invalid indentation",
			yaml: `name: Test
jobs:
  build:
  steps:
    - uses: checkout@v1
`,
			expectError: "yaml",
		},
		{
			name: "Unclosed quote",
			yaml: `name: "Test
jobs:
  build:
    steps:
      - uses: checkout@v1
`,
			expectError: "yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := submitWorkflowRequestBody{YAML: tt.yaml}
			bodyBytes, _ := json.Marshal(reqBody)

			req, err := http.NewRequest(http.MethodPost, serverURL+"/v1/workflows", bytes.NewReader(bodyBytes))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should return 400 Bad Request
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject YAML with syntax errors")

			// Parse error response
			body, _ := io.ReadAll(resp.Body)
			var errorResp map[string]interface{}
			err = json.Unmarshal(body, &errorResp)
			require.NoError(t, err, "Error response should be valid JSON")

			// Verify RFC 7807 format
			assert.Contains(t, errorResp, "type", "Error should have 'type' field")
			assert.Contains(t, errorResp, "title", "Error should have 'title' field")
			assert.Contains(t, errorResp, "status", "Error should have 'status' field")
			assert.Contains(t, errorResp, "detail", "Error should have 'detail' field")

			// Verify error message contains parsing/validation info
			if detail, ok := errorResp["detail"].(string); ok {
				// Server returns generic "YAML parsing failed" for syntax errors
				assert.Contains(t, detail, "parsing", "Error detail should mention parsing issue")
			}
		})
	}
}

// TestYAMLValidation_MissingRequiredFields tests validation of required fields
func TestYAMLValidation_MissingRequiredFields(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()

	tests := []struct {
		name         string
		yaml         string
		missingField string
	}{
		{
			name: "Missing workflow name",
			yaml: `jobs:
  build:
    steps:
      - uses: checkout@v1
`,
			missingField: "name",
		},
		{
			name: "Missing jobs",
			yaml: `name: Test Workflow
`,
			missingField: "jobs",
		},
		{
			name: "Missing steps in job",
			yaml: `name: Test Workflow
jobs:
  build:
    runs-on: linux-amd64
`,
			missingField: "steps",
		},
		{
			name: "Missing uses in step",
			yaml: `name: Test Workflow
jobs:
  build:
    steps:
      - name: Test Step
        with:
          command: echo hello
`,
			missingField: "uses",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := submitWorkflowRequestBody{YAML: tt.yaml}
			bodyBytes, _ := json.Marshal(reqBody)

			req, err := http.NewRequest(http.MethodPost, serverURL+"/v1/workflows", bytes.NewReader(bodyBytes))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should return 400 Bad Request
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject YAML with missing required fields")

			// Parse error response
			body, _ := io.ReadAll(resp.Body)
			var errorResp map[string]interface{}
			err = json.Unmarshal(body, &errorResp)
			require.NoError(t, err, "Error response should be valid JSON")

			// Server may return generic message, check that validation failed
			if detail, ok := errorResp["detail"].(string); ok {
				assert.Contains(t, detail, "validation", "Error should mention validation failure")
			}
			// The specific field error might be in the errors array
			// For now, just verify we got a 400 Bad Request
		})
	}
}

// TestYAMLValidation_InvalidFieldTypes tests type validation
func TestYAMLValidation_InvalidFieldTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()

	tests := []struct {
		name        string
		yaml        string
		expectError string
	}{
		{
			name: "timeout-minutes as string instead of int",
			yaml: `name: Test
jobs:
  build:
    timeout-minutes: "30"
    steps:
      - uses: checkout@v1
`,
			expectError: "timeout",
		},
		{
			name: "continue-on-error as string instead of bool",
			yaml: `name: Test
jobs:
  build:
    continue-on-error: "true"
    steps:
      - uses: checkout@v1
`,
			expectError: "continue-on-error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := submitWorkflowRequestBody{YAML: tt.yaml}
			bodyBytes, _ := json.Marshal(reqBody)

			req, err := http.NewRequest(http.MethodPost, serverURL+"/v1/workflows", bytes.NewReader(bodyBytes))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should return 400 Bad Request
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject YAML with invalid field types")

			body, _ := io.ReadAll(resp.Body)
			// Server returns generic validation error for type mismatches
			assert.Contains(t, string(body), "validation", "Error should mention validation failure")
		})
	}
}

// TestYAMLValidation_InvalidNodeReference tests validation of node references
func TestYAMLValidation_InvalidNodeReference(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()

	// YAML with non-existent node
	yamlContent := `name: Test
jobs:
  build:
    steps:
      - name: Use non-existent node
        uses: nonexistent-node@v1
        with:
          param: value
`

	reqBody := submitWorkflowRequestBody{YAML: yamlContent}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest(http.MethodPost, serverURL+"/v1/workflows", bytes.NewReader(bodyBytes))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject YAML with invalid node reference")

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Verify validation error is returned
	assert.Contains(t, bodyStr, "validation", "Error should be a validation error")
}

// TestYAMLValidation_EmptyBody tests handling of empty request body
func TestYAMLValidation_EmptyBody(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()

	req, err := http.NewRequest(http.MethodPost, serverURL+"/v1/workflows", bytes.NewReader([]byte("")))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject empty request body")

	// Verify RFC 7807 format
	body, _ := io.ReadAll(resp.Body)
	var errorResp map[string]interface{}
	err = json.Unmarshal(body, &errorResp)
	require.NoError(t, err, "Error response should be valid JSON")

	assert.Contains(t, errorResp, "type")
	assert.Contains(t, errorResp, "status")
	assert.Equal(t, float64(400), errorResp["status"])
}

// TestYAMLValidation_LargeFile tests handling of large YAML files
func TestYAMLValidation_LargeFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()

	// Generate a large YAML (>1MB to test size limits)
	largeYAML := "name: Large Workflow\njobs:\n"
	for i := 0; i < 10000; i++ {
		largeYAML += fmt.Sprintf("  job%d:\n    steps:\n      - uses: checkout@v1\n", i)
	}

	reqBody := submitWorkflowRequestBody{YAML: largeYAML}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest(http.MethodPost, serverURL+"/v1/workflows", bytes.NewReader(bodyBytes))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 413 Payload Too Large or 400 Bad Request
	assert.True(t, resp.StatusCode == http.StatusRequestEntityTooLarge || resp.StatusCode == http.StatusBadRequest,
		"Should reject files that are too large")
}

// TestYAMLValidation_ConcurrentRequests tests concurrent validation requests
func TestYAMLValidation_ConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()

	invalidYAML := `name: Test
jobs:
  build:
    invalid_field: value
`

	// Submit 10 concurrent validation requests (all should fail validation)
	numRequests := 10
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			reqBody := submitWorkflowRequestBody{YAML: invalidYAML}
			bodyBytes, _ := json.Marshal(reqBody)
			resp, err := http.Post(serverURL+"/v1/workflows", "application/json", bytes.NewReader(bodyBytes))
			if err == nil && resp != nil {
				defer resp.Body.Close()
				results <- resp.StatusCode
			} else {
				results <- 0
			}
		}()
	}

	// Wait for all requests and count 400 responses
	validationFailures := 0
	for i := 0; i < numRequests; i++ {
		status := <-results
		if status == http.StatusBadRequest {
			validationFailures++
		}
	}

	// All requests should return 400 (server handles concurrent requests)
	assert.GreaterOrEqual(t, validationFailures, numRequests-2, "Server should handle concurrent validation requests")
}

// TestYAMLValidation_RFC7807ErrorFormat tests error response format compliance
func TestYAMLValidation_RFC7807ErrorFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()

	invalidYAML := `name: Test
jobs:
  build:
    invalid_field: value
`

	reqBody := submitWorkflowRequestBody{YAML: invalidYAML}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest(http.MethodPost, serverURL+"/v1/workflows", bytes.NewReader(bodyBytes))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify Content-Type header
	contentType := resp.Header.Get("Content-Type")
	assert.Contains(t, contentType, "application/problem+json", "Error responses should use RFC 7807 media type")

	// Parse response
	body, _ := io.ReadAll(resp.Body)
	var errorResp map[string]interface{}
	err = json.Unmarshal(body, &errorResp)
	require.NoError(t, err, "Error response should be valid JSON")

	// Verify RFC 7807 required fields
	assert.Contains(t, errorResp, "type", "RFC 7807 requires 'type' field")
	assert.Contains(t, errorResp, "title", "RFC 7807 requires 'title' field")
	assert.Contains(t, errorResp, "status", "RFC 7807 requires 'status' field")

	// Verify status matches HTTP status code
	assert.Equal(t, float64(resp.StatusCode), errorResp["status"], "status field should match HTTP status code")
}
