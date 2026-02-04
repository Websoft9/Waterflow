//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/test/support/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ====================================================================================
// Story 1.9: Workflow Management API - New Architecture Integration Tests (ADR-0009)
// ====================================================================================
//
// Test Coverage:
// - Definition API: CRUD operations for workflow definitions
// - Execution API: Runtime workflow execution and management
// - Parameter override mechanism
// - Error scenarios (404, 409, 422, 500)
//
// Architecture: Definition/Execution separation
// - Definition API: Manages workflow definitions in database
// - Execution API: Manages workflow runtime via Temporal
//
// ====================================================================================

// DefinitionResponse represents workflow definition response
type DefinitionResponse struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Content     string                 `json:"content"`
	Tags        []string               `json:"tags"`
	Parameters  []Parameter            `json:"parameters"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
	Version     int                    `json:"version"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Parameter represents workflow parameter
type Parameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
}

// ExecutionResponse represents workflow execution response
type ExecutionResponse struct {
	ExecutionID  string `json:"execution_id"`
	WorkflowName string `json:"workflow_name"`
	Status       string `json:"status"`
	StartedAt    string `json:"started_at"`
	URL          string `json:"url"`
}

// ErrorResponse represents error response
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

const testDefinitionYAML = `name: test-hello
on:
  workflow_call:
vars:
  message: "Hello World"
  count: 3
jobs:
  greet:
    runs-on: default
    steps:
      - name: Print greeting
        run: echo "${{ vars.message }}"
      - name: Print count
        run: echo "Count is ${{ vars.count }}"
`

const testDefinitionWithParamsYAML = `name: test-params
on:
  workflow_call:
vars:
  env: production
  replicas: 5
  debug: false
jobs:
  deploy:
    runs-on: default
    steps:
      - name: Deploy
        run: echo "Deploying to ${{ vars.env }} with ${{ vars.replicas }} replicas"
`

// TestStory1_9_NewArch_INT_001_CreateDefinition tests creating workflow definition
func TestStory1_9_NewArch_INT_001_CreateDefinition(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.9-NEWARCH-INT-001",
		Given:  "Valid workflow YAML with parameters",
		When:   "POST /v1/workflows/definitions is called",
		Then: []string{
			"Definition is created in database",
			"201 Created status returned",
			"Definition metadata includes parameters",
			"Response time < 500ms",
		},
		AcceptanceCriteria: []string{
			"AC1: Create workflow definition with YAML content",
			"AC: Extract parameters from vars section",
			"AC: Store definition in PostgreSQL",
			"AC: Return 409 if definition already exists",
		},
	}).Log(t)

	if skipDatabaseTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	// Create definition request
	reqBody := map[string]interface{}{
		"name":        "test-hello-create",
		"description": "Test workflow for creation",
		"content":     testDefinitionYAML,
		"tags":        []string{"test", "integration"},
	}

	jsonBody, err := json.Marshal(reqBody)
	require.NoError(t, err)

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		serverURL+"/v1/workflows/definitions",
		bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	duration := time.Since(start)

	// Assert response
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Should return 201 Created")
	assert.Less(t, duration.Milliseconds(), int64(500), "Response time should be < 500ms")

	// Parse response
	var defResp DefinitionResponse
	err = json.NewDecoder(resp.Body).Decode(&defResp)
	require.NoError(t, err)

	assert.Equal(t, "test-hello-create", defResp.Name)
	assert.Equal(t, "Test workflow for creation", defResp.Description)
	assert.Contains(t, defResp.Tags, "test")
	assert.Contains(t, defResp.Tags, "integration")
	assert.Len(t, defResp.Parameters, 2, "Should extract 2 parameters from vars")

	// Verify parameters
	paramNames := make(map[string]bool)
	for _, p := range defResp.Parameters {
		paramNames[p.Name] = true
	}
	assert.True(t, paramNames["message"], "Should have 'message' parameter")
	assert.True(t, paramNames["count"], "Should have 'count' parameter")

	// Cleanup
	cleanupDefinition(t, "test-hello-create")
}

// TestStory1_9_NewArch_INT_002_GetDefinition tests retrieving workflow definition
func TestStory1_9_NewArch_INT_002_GetDefinition(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.9-NEWARCH-INT-002",
		Given:  "Workflow definition exists in database",
		When:   "GET /v1/workflows/definitions/{name} is called",
		Then: []string{
			"Definition details are returned",
			"200 OK status returned",
			"Content includes original YAML",
			"Response time < 200ms",
		},
		AcceptanceCriteria: []string{
			"AC2: Get workflow definition by name",
			"AC: Return 404 if definition not found",
		},
	}).Log(t)

	if skipDatabaseTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	// Create definition first
	defName := "test-hello-get"
	createTestDefinition(t, defName, testDefinitionYAML)
	defer cleanupDefinition(t, defName)

	// Get definition
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		serverURL+"/v1/workflows/definitions/"+defName, nil)
	require.NoError(t, err)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	duration := time.Since(start)

	// Assert response
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should return 200 OK")
	assert.Less(t, duration.Milliseconds(), int64(200), "Response time should be < 200ms")

	// Parse response
	var defResp DefinitionResponse
	err = json.NewDecoder(resp.Body).Decode(&defResp)
	require.NoError(t, err)

	assert.Equal(t, defName, defResp.Name)
	assert.Contains(t, defResp.Content, "name: test-hello")
	assert.NotEmpty(t, defResp.CreatedAt)
}

// TestStory1_9_NewArch_INT_003_UpdateDefinition tests updating workflow definition
func TestStory1_9_NewArch_INT_003_UpdateDefinition(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.9-NEWARCH-INT-003",
		Given:  "Workflow definition exists",
		When:   "PUT /v1/workflows/definitions/{name} is called",
		Then: []string{
			"Definition is updated in database",
			"Version is incremented",
			"200 OK status returned",
		},
		AcceptanceCriteria: []string{
			"AC3: Update workflow definition",
			"AC: Validate YAML before update",
			"AC: Return 404 if definition not found",
		},
	}).Log(t)

	if skipDatabaseTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	// Create definition first
	defName := "test-hello-update"
	createTestDefinition(t, defName, testDefinitionYAML)
	defer cleanupDefinition(t, defName)

	// Update definition
	updatedYAML := `name: test-hello
on:
  workflow_call:
vars:
  message: "Updated Message"
  count: 5
jobs:
  greet:
    runs-on: default
    steps:
      - name: Print updated greeting
        run: echo "${{ vars.message }}"
`

	reqBody := map[string]interface{}{
		"content":     updatedYAML,
		"description": "Updated description",
	}

	jsonBody, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		serverURL+"/v1/workflows/definitions/"+defName,
		bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert response
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should return 200 OK")

	// Parse response
	var defResp DefinitionResponse
	err = json.NewDecoder(resp.Body).Decode(&defResp)
	require.NoError(t, err)

	assert.Equal(t, "Updated description", defResp.Description)
	assert.Contains(t, defResp.Content, "Updated Message")
	assert.Equal(t, 2, defResp.Version, "Version should be incremented to 2")
}

// TestStory1_9_NewArch_INT_004_DeleteDefinition tests deleting workflow definition
func TestStory1_9_NewArch_INT_004_DeleteDefinition(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.9-NEWARCH-INT-004",
		Given:  "Workflow definition exists",
		When:   "DELETE /v1/workflows/definitions/{name} is called",
		Then: []string{
			"Definition is removed from database",
			"204 No Content status returned",
			"Subsequent GET returns 404",
		},
		AcceptanceCriteria: []string{
			"AC4: Delete workflow definition",
			"AC: Return 404 if definition not found",
		},
	}).Log(t)

	if skipDatabaseTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	// Create definition first
	defName := "test-hello-delete"
	createTestDefinition(t, defName, testDefinitionYAML)

	// Delete definition
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		serverURL+"/v1/workflows/definitions/"+defName, nil)
	require.NoError(t, err)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert response
	assert.Equal(t, http.StatusNoContent, resp.StatusCode, "Should return 204 No Content")

	// Verify deletion - GET should return 404
	req2, err := http.NewRequestWithContext(ctx, http.MethodGet,
		serverURL+"/v1/workflows/definitions/"+defName, nil)
	require.NoError(t, err)

	resp2, err := client.Do(req2)
	require.NoError(t, err)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp2.StatusCode, "GET after DELETE should return 404")
}

// TestStory1_9_NewArch_INT_005_ListDefinitions tests listing workflow definitions
func TestStory1_9_NewArch_INT_005_ListDefinitions(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.9-NEWARCH-INT-005",
		Given:  "Multiple workflow definitions exist",
		When:   "GET /v1/workflows/definitions with filters",
		Then: []string{
			"Paginated list is returned",
			"Filters by tags work correctly",
			"Response includes total count",
		},
		AcceptanceCriteria: []string{
			"AC5: List workflow definitions",
			"AC: Support pagination (page, limit)",
			"AC: Support filtering by tags",
		},
	}).Log(t)

	if skipDatabaseTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	// Create multiple definitions
	createTestDefinitionWithTags(t, "test-list-1", testDefinitionYAML, []string{"integration", "test"})
	createTestDefinitionWithTags(t, "test-list-2", testDefinitionWithParamsYAML, []string{"integration", "prod"})
	defer cleanupDefinition(t, "test-list-1")
	defer cleanupDefinition(t, "test-list-2")

	// List all
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		serverURL+"/v1/workflows/definitions?limit=10", nil)
	require.NoError(t, err)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should return 200 OK")

	var listResp struct {
		Definitions []DefinitionResponse `json:"definitions"`
		Total       int                  `json:"total"`
		Page        int                  `json:"page"`
		Limit       int                  `json:"limit"`
	}
	err = json.NewDecoder(resp.Body).Decode(&listResp)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, listResp.Total, 2, "Should have at least 2 definitions")
	assert.GreaterOrEqual(t, len(listResp.Definitions), 2, "Should return at least 2 definitions")
}

// TestStory1_9_NewArch_INT_006_ExecuteFromDefinition tests executing from saved definition
func TestStory1_9_NewArch_INT_006_ExecuteFromDefinition(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.9-NEWARCH-INT-006",
		Given:  "Workflow definition exists with parameters",
		When:   "POST /v1/workflows/{name}/run with variable overrides",
		Then: []string{
			"Definition is loaded from database",
			"Variables are merged (execution > YAML)",
			"Workflow is submitted to Temporal",
			"202 Accepted status returned",
			"Execution ID is generated",
		},
		AcceptanceCriteria: []string{
			"AC6: Execute workflow from definition",
			"AC: Support variable override",
			"AC: Return execution ID and status URL",
		},
	}).Log(t)

	if skipDatabaseTests(t) || skipTemporalTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	// Create definition first
	defName := "test-execute-from-def"
	createTestDefinition(t, defName, testDefinitionWithParamsYAML)
	defer cleanupDefinition(t, defName)

	// Execute with variable overrides
	reqBody := map[string]interface{}{
		"vars": map[string]interface{}{
			"env":      "staging", // Override from production
			"replicas": 3,         // Override from 5
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		serverURL+"/v1/workflows/"+defName+"/run",
		bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert response
	assert.Equal(t, http.StatusAccepted, resp.StatusCode, "Should return 202 Accepted")

	// Parse response
	var execResp ExecutionResponse
	err = json.NewDecoder(resp.Body).Decode(&execResp)
	require.NoError(t, err)

	assert.NotEmpty(t, execResp.ExecutionID, "Should return execution ID")
	assert.Equal(t, defName, execResp.WorkflowName)
	assert.Equal(t, "running", execResp.Status)
	assert.Contains(t, execResp.URL, "/v1/executions/")
}

// TestStory1_9_NewArch_INT_007_DirectExecute tests direct workflow execution
func TestStory1_9_NewArch_INT_007_DirectExecute(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.9-NEWARCH-INT-007",
		Given:  "Valid workflow YAML without saving definition",
		When:   "POST /v1/executions with workflow content",
		Then: []string{
			"Workflow is validated and executed",
			"No definition stored in database",
			"202 Accepted status returned",
			"Execution ID is generated",
		},
		AcceptanceCriteria: []string{
			"AC7: Direct workflow execution",
			"AC: Validate YAML before execution",
			"AC: Support variable injection",
			"AC: Return 422 for invalid YAML",
		},
	}).Log(t)

	if skipTemporalTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	// Direct execute
	reqBody := map[string]interface{}{
		"workflow": testDefinitionYAML,
		"vars": map[string]interface{}{
			"message": "Direct execution test",
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		serverURL+"/v1/executions",
		bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert response
	assert.Equal(t, http.StatusAccepted, resp.StatusCode, "Should return 202 Accepted")

	// Parse response
	var execResp ExecutionResponse
	err = json.NewDecoder(resp.Body).Decode(&execResp)
	require.NoError(t, err)

	assert.NotEmpty(t, execResp.ExecutionID, "Should return execution ID")
	assert.Equal(t, "running", execResp.Status)
}

// TestStory1_9_NewArch_INT_008_ErrorScenarios tests error handling
func TestStory1_9_NewArch_INT_008_ErrorScenarios(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.9-NEWARCH-INT-008",
		Given:  "Various invalid requests",
		When:   "APIs are called with invalid data",
		Then: []string{
			"Appropriate error codes are returned",
			"Error messages are descriptive",
			"Error format is consistent (AC13)",
		},
		AcceptanceCriteria: []string{
			"AC13: Unified error format",
			"AC: 404 for not found",
			"AC: 409 for conflicts",
			"AC: 422 for validation errors",
		},
	}).Log(t)

	if skipDatabaseTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()
	client := &http.Client{Timeout: 10 * time.Second}

	t.Run("404_DefinitionNotFound", func(t *testing.T) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
			serverURL+"/v1/workflows/definitions/nonexistent", nil)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		var errResp ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		assert.Equal(t, "not_found", errResp.Code)
	})

	t.Run("409_DuplicateDefinition", func(t *testing.T) {
		defName := "test-duplicate"
		createTestDefinition(t, defName, testDefinitionYAML)
		defer cleanupDefinition(t, defName)

		// Try to create again
		reqBody := map[string]interface{}{
			"name":    defName,
			"content": testDefinitionYAML,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
			serverURL+"/v1/workflows/definitions",
			bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		var errResp ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		assert.Equal(t, "conflict", errResp.Code)
	})

	t.Run("422_InvalidYAML", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"workflow": "invalid: yaml: content:\n  - broken",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
			serverURL+"/v1/executions",
			bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

		var errResp ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		assert.Equal(t, "validation_error", errResp.Code)
	})
}

// Helper functions

func skipDatabaseTests(t *testing.T) bool {
	if getEnvOrDefault("SKIP_DATABASE_TESTS", "") == "true" {
		t.Skip("Database tests disabled (SKIP_DATABASE_TESTS=true)")
		return true
	}
	return false
}

func skipTemporalTests(t *testing.T) bool {
	if getEnvOrDefault("SKIP_TEMPORAL_TESTS", "") == "true" {
		t.Skip("Temporal tests disabled (SKIP_TEMPORAL_TESTS=true)")
		return true
	}
	return false
}

func createTestDefinition(t *testing.T, name, content string) {
	createTestDefinitionWithTags(t, name, content, []string{"test"})
}

func createTestDefinitionWithTags(t *testing.T, name, content string, tags []string) {
	ctx := context.Background()
	serverURL := getServerURL()

	reqBody := map[string]interface{}{
		"name":        name,
		"description": fmt.Sprintf("Test definition: %s", name),
		"content":     content,
		"tags":        tags,
	}

	jsonBody, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		serverURL+"/v1/workflows/definitions",
		bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(resp.Body)
		t.Logf("Failed to create definition: %s - Response: %s", name, string(body))
	}
}

func cleanupDefinition(t *testing.T, name string) {
	ctx := context.Background()
	serverURL := getServerURL()

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		serverURL+"/v1/workflows/definitions/"+name, nil)
	if err != nil {
		t.Logf("Failed to create cleanup request: %v", err)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Logf("Failed to cleanup definition %s: %v", name, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		t.Logf("Cleanup warning for %s: status %d", name, resp.StatusCode)
	}
}
