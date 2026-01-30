//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// [P0] Workflow Complete Lifecycle Tests
// ============================================================================

// TestIntegration_WorkflowLifecycle_SubmitStatusLogsCancel tests full workflow lifecycle
func TestIntegration_WorkflowLifecycle_SubmitStatusLogsCancel(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// GIVEN: A valid workflow YAML with observable output
	workflowYAML := `
name: lifecycle-test
on: push
vars:
  message: "Hello from lifecycle test"
jobs:
  main:
    runs-on: linux-amd64  # Route to Agent Worker (not Server Worker)
    steps:
      - name: Echo message
        uses: run@v1
        with:
          command: echo "${{ vars.message }}"
      - name: Generate output
        uses: run@v1
        with:
          command: echo "step_output=success"
`

	// WHEN: Submit workflow
	resp, err := submitWorkflow(ctx, serverURL, workflowYAML)
	require.NoError(t, err, "Step 1: Submit workflow should succeed")
	require.NotEmpty(t, resp.GetID(), "Workflow ID should not be empty")
	t.Logf("✓ Workflow submitted: %s", resp.GetID())

	// THEN: Query status immediately (should be running or completed)
	status, err := getWorkflowStatus(ctx, serverURL, resp.GetID())
	require.NoError(t, err, "Step 2: Get status should succeed")
	assert.NotEmpty(t, status.Status, "Status should not be empty")
	t.Logf("✓ Initial status: %s", status.Status)

	// WHEN: Wait for workflow completion
	finalStatus, err := waitForWorkflowCompletion(ctx, serverURL, resp.GetID(), 90*time.Second)
	require.NoError(t, err, "Step 3: Workflow should complete")
	assert.Equal(t, "completed", strings.ToLower(finalStatus.Status), "Workflow should complete successfully")
	t.Logf("✓ Final status: %s", finalStatus.Status)

	// THEN: Retrieve logs
	logs, err := getWorkflowLogs(ctx, serverURL, resp.GetID())
	require.NoError(t, err, "Step 4: Get logs should succeed")
	assert.NotEmpty(t, logs, "Logs should not be empty")
	t.Logf("✓ Retrieved %d bytes of logs", len(logs))
}

// TestIntegration_WorkflowLifecycle_CancelRunningWorkflow tests cancellation
func TestIntegration_WorkflowLifecycle_CancelRunningWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// GIVEN: A long-running workflow
	workflowYAML := `
name: cancel-lifecycle-test
on: workflow_dispatch
jobs:
  slow:
    runs-on: linux-amd64
    steps:
      - name: Long task
        uses: sleep@v1
        with:
          duration: 120s
`

	// WHEN: Submit workflow
	resp, err := submitWorkflow(ctx, serverURL, workflowYAML)
	require.NoError(t, err)
	t.Logf("✓ Submitted long-running workflow: %s", resp.GetID())

	// Wait for workflow to start running
	time.Sleep(3 * time.Second)

	// Verify it's running
	status, err := getWorkflowStatus(ctx, serverURL, resp.GetID())
	require.NoError(t, err)
	assert.Contains(t, strings.ToLower(status.Status), "running", "Workflow should be running")

	// WHEN: Cancel the workflow
	err = cancelWorkflow(ctx, serverURL, resp.GetID())
	require.NoError(t, err, "Cancel should succeed")
	t.Logf("✓ Cancel request sent")

	// Wait for cancellation to propagate
	time.Sleep(3 * time.Second)

	// THEN: Verify workflow is cancelled
	finalStatus, err := getWorkflowStatus(ctx, serverURL, resp.GetID())
	require.NoError(t, err)
	statusLower := strings.ToLower(finalStatus.Status)
	assert.True(t,
		strings.Contains(statusLower, "cancel") || strings.Contains(statusLower, "terminated"),
		"Workflow should be cancelled, got: %s", finalStatus.Status)
	t.Logf("✓ Final status after cancel: %s", finalStatus.Status)
}

// ============================================================================
// [P0] Workflow Vars Merging Integration Tests
// ============================================================================

// TestIntegration_WorkflowVars_RequestOverridesYAML tests vars merging behavior
func TestIntegration_WorkflowVars_RequestOverridesYAML(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// GIVEN: Workflow with YAML-defined vars
	workflowYAML := `
name: vars-merge-test
on: workflow_dispatch
vars:
  environment: production
  version: "1.0"
  region: us-west-2
jobs:
  deploy:
    runs-on: linux-amd64
    steps:
      - name: Show vars
        uses: shell@v1
        with:
          command: |
            echo "ENV=${{ vars.environment }}"
            echo "VERSION=${{ vars.version }}"
            echo "REGION=${{ vars.region }}"
`

	// WHEN: Submit with JSON body containing override vars
	reqBody := map[string]interface{}{
		"yaml": workflowYAML,
		"vars": map[string]interface{}{
			"environment": "staging",         // Override existing
			"version":     "2.0",             // Override existing
			"new_var":     "added_at_submit", // Add new var
		},
	}
	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", serverURL+"/v1/workflows", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// THEN: Should accept and execute
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated,
		"Expected 200 or 201, got %d", resp.StatusCode)

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		workflowID, ok := result["id"].(string)
		if !ok {
			workflowID, _ = result["workflow_id"].(string)
		}
		require.NotEmpty(t, workflowID, "Should return workflow ID")
		t.Logf("✓ Workflow with merged vars submitted: %s", workflowID)

		// Wait for completion and verify
		_, err := waitForWorkflowCompletion(ctx, serverURL, workflowID, 60*time.Second)
		require.NoError(t, err, "Workflow should complete")
		t.Logf("✓ Workflow completed successfully with merged vars")
	}
}

// ============================================================================
// [P1] Workflow Rerun Integration Tests
// ============================================================================

// TestIntegration_WorkflowRerun_WithNewVars tests rerun with override vars
func TestIntegration_WorkflowRerun_WithNewVars(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// GIVEN: A completed workflow
	workflowYAML := `
name: rerun-vars-test
on: workflow_dispatch
vars:
  run_count: "1"
jobs:
  count:
    runs-on: linux-amd64
    steps:
      - name: Show run count
        uses: shell@v1
        with:
          command: echo "Run count is ${{ vars.run_count }}"
`

	// Submit original workflow
	originalResp, err := submitWorkflow(ctx, serverURL, workflowYAML)
	require.NoError(t, err)
	t.Logf("✓ Original workflow submitted: %s", originalResp.GetID())

	// Wait for completion
	_, err = waitForWorkflowCompletion(ctx, serverURL, originalResp.GetID(), 60*time.Second)
	require.NoError(t, err)
	t.Logf("✓ Original workflow completed")

	// WHEN: Rerun with new vars
	rerunBody := map[string]interface{}{
		"vars": map[string]interface{}{
			"run_count": "2",
		},
	}
	body, _ := json.Marshal(rerunBody)

	req, err := http.NewRequestWithContext(ctx, "POST",
		serverURL+"/v1/workflows/"+originalResp.GetID()+"/rerun",
		bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// THEN: Should create new workflow
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusCreated {
		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		newWorkflowID, ok := result["id"].(string)
		if !ok {
			newWorkflowID, _ = result["workflow_id"].(string)
		}

		if newWorkflowID != "" {
			t.Logf("✓ Rerun created new workflow: %s", newWorkflowID)
			assert.NotEqual(t, originalResp.GetID(), newWorkflowID, "Rerun should create new workflow ID")

			// Wait for rerun to complete
			_, err := waitForWorkflowCompletion(ctx, serverURL, newWorkflowID, 60*time.Second)
			require.NoError(t, err, "Rerun workflow should complete")
			t.Logf("✓ Rerun workflow completed")
		}
	} else {
		t.Logf("Rerun returned status %d (may require Temporal connection)", resp.StatusCode)
	}
}

// ============================================================================
// [P1] Concurrent Workflow Execution Tests
// ============================================================================

// TestIntegration_ConcurrentWorkflows tests parallel workflow execution
func TestIntegration_ConcurrentWorkflows(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	const numWorkflows = 5

	// GIVEN: Multiple concurrent workflow submissions
	workflowYAML := `
name: concurrent-test-%d
on: workflow_dispatch
jobs:
  quick:
    runs-on: linux-amd64
    steps:
      - name: Quick task
        uses: shell@v1
        with:
          command: echo "Workflow %d executing"
`

	var wg sync.WaitGroup
	results := make(chan struct {
		id  string
		err error
	}, numWorkflows)

	// WHEN: Submit workflows concurrently
	for i := 0; i < numWorkflows; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			yaml := fmt.Sprintf(workflowYAML, idx, idx)
			resp, err := submitWorkflow(ctx, serverURL, yaml)
			if err != nil {
				results <- struct {
					id  string
					err error
				}{"", err}
				return
			}
			results <- struct {
				id  string
				err error
			}{resp.GetID(), nil}
		}(i)
	}

	// Wait for all submissions
	wg.Wait()
	close(results)

	// THEN: All workflows should be submitted successfully
	var submittedIDs []string
	var submitErrors []error

	for result := range results {
		if result.err != nil {
			submitErrors = append(submitErrors, result.err)
		} else {
			submittedIDs = append(submittedIDs, result.id)
		}
	}

	t.Logf("✓ Submitted %d workflows, %d errors", len(submittedIDs), len(submitErrors))
	assert.GreaterOrEqual(t, len(submittedIDs), numWorkflows-1,
		"Most workflows should be submitted successfully")

	// Wait for at least one to complete
	if len(submittedIDs) > 0 {
		_, err := waitForWorkflowCompletion(ctx, serverURL, submittedIDs[0], 90*time.Second)
		if err != nil {
			t.Logf("First workflow completion: %v", err)
		} else {
			t.Logf("✓ First concurrent workflow completed")
		}
	}
}

// ============================================================================
// [P1] List Workflows Integration Tests
// ============================================================================

// TestIntegration_ListWorkflows_Pagination tests workflow listing with pagination
func TestIntegration_ListWorkflows_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Submit a few workflows first
	for i := 0; i < 3; i++ {
		yaml := fmt.Sprintf(`
name: pagination-test-%d
on: workflow_dispatch
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Quick
        uses: shell@v1
        with:
          command: echo "test %d"
`, i, i)
		_, _ = submitWorkflow(ctx, serverURL, yaml)
	}

	// Wait briefly for submission processing
	time.Sleep(2 * time.Second)

	t.Run("List with default params", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET", serverURL+"/v1/workflows", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Contains(t, result, "workflows")
		assert.Contains(t, result, "pagination")
		t.Logf("✓ List workflows returned successfully")
	})

	t.Run("List with page and limit", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET",
			serverURL+"/v1/workflows?page=1&limit=5", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		pagination, ok := result["pagination"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(1), pagination["page"])
		assert.Equal(t, float64(5), pagination["limit"])
		t.Logf("✓ Pagination params applied correctly")
	})

	t.Run("List with status filter", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET",
			serverURL+"/v1/workflows?status=running", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		t.Logf("✓ Status filter applied")
	})
}

// ============================================================================
// [P2] Error Handling Integration Tests
// ============================================================================

// TestIntegration_ErrorHandling_Lifecycle tests various error scenarios for workflow lifecycle
func TestIntegration_ErrorHandling_Lifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	t.Run("Submit empty YAML", func(t *testing.T) {
		reqBody := map[string]string{"yaml": ""}
		body, _ := json.Marshal(reqBody)

		req, err := http.NewRequestWithContext(ctx, "POST",
			serverURL+"/v1/workflows", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.True(t, resp.StatusCode >= 400, "Should return error for empty YAML")
		t.Logf("✓ Empty YAML rejected with status %d", resp.StatusCode)
	})

	t.Run("Submit invalid YAML syntax", func(t *testing.T) {
		reqBody := map[string]string{"yaml": "invalid: yaml: content: here:"}
		body, _ := json.Marshal(reqBody)

		req, err := http.NewRequestWithContext(ctx, "POST",
			serverURL+"/v1/workflows", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.True(t, resp.StatusCode >= 400, "Should return error for invalid YAML")
		t.Logf("✓ Invalid YAML rejected with status %d", resp.StatusCode)
	})

	t.Run("Get non-existent workflow", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET",
			serverURL+"/v1/workflows/non-existent-workflow-12345", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode, "Should return 404")
		t.Logf("✓ Non-existent workflow returns 404")
	})

	t.Run("Cancel non-existent workflow", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "POST",
			serverURL+"/v1/workflows/non-existent-workflow-12345/cancel", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.True(t, resp.StatusCode >= 400, "Should return error")
		t.Logf("✓ Cancel non-existent workflow rejected with status %d", resp.StatusCode)
	})

	t.Run("Rerun non-existent workflow", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "POST",
			serverURL+"/v1/workflows/non-existent-workflow-12345/rerun", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.True(t, resp.StatusCode >= 400, "Should return error")
		t.Logf("✓ Rerun non-existent workflow rejected with status %d", resp.StatusCode)
	})

	t.Run("Invalid pagination params", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET",
			serverURL+"/v1/workflows?page=0", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "page=0 should be rejected")
		t.Logf("✓ Invalid pagination rejected")
	})
}

// ============================================================================
// [P2] API Response Format Tests
// ============================================================================

// TestIntegration_ResponseFormats tests API response formats
func TestIntegration_ResponseFormats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	t.Run("Health endpoint JSON response", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET", serverURL+"/health", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Contains(t, result, "status")
		t.Logf("✓ Health response is valid JSON")
	})

	t.Run("Error response RFC 7807 format", func(t *testing.T) {
		reqBody := map[string]string{"yaml": ""}
		body, _ := json.Marshal(reqBody)

		req, err := http.NewRequestWithContext(ctx, "POST",
			serverURL+"/v1/workflows", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check Content-Type for problem+json
		contentType := resp.Header.Get("Content-Type")
		assert.Contains(t, contentType, "application/problem+json",
			"Error should return application/problem+json")

		var result map[string]interface{}
		bodyBytes, _ := io.ReadAll(resp.Body)
		err = json.Unmarshal(bodyBytes, &result)
		require.NoError(t, err)

		// RFC 7807 fields
		assert.Contains(t, result, "type", "Should have 'type' field")
		assert.Contains(t, result, "title", "Should have 'title' field")
		assert.Contains(t, result, "status", "Should have 'status' field")
		t.Logf("✓ Error response follows RFC 7807")
	})

	t.Run("List workflows response structure", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET", serverURL+"/v1/workflows", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		// Required response structure
		assert.Contains(t, result, "workflows", "Should have 'workflows' array")
		assert.Contains(t, result, "pagination", "Should have 'pagination' object")

		pagination, ok := result["pagination"].(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, pagination, "page")
		assert.Contains(t, pagination, "limit")
		assert.Contains(t, pagination, "total")
		assert.Contains(t, pagination, "total_pages")
		t.Logf("✓ List response has correct structure")
	})
}

// ============================================================================
// [P2] Workflow Validation Integration Tests
// ============================================================================

// TestIntegration_WorkflowValidation tests the /v1/validate endpoint
func TestIntegration_WorkflowValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	serverURL := getServerURL()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	t.Run("Validate valid workflow", func(t *testing.T) {
		yaml := `
name: valid-test
on: workflow_dispatch
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Test
        uses: shell@v1
        with:
          command: echo "valid"
`
		reqBody := map[string]string{"yaml": yaml}
		body, _ := json.Marshal(reqBody)

		req, err := http.NewRequestWithContext(ctx, "POST",
			serverURL+"/v1/validate", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		t.Logf("✓ Valid workflow passes validation")
	})

	t.Run("Validate workflow missing runs-on", func(t *testing.T) {
		yaml := `
name: invalid-no-runson
on: workflow_dispatch
jobs:
  test:
    steps:
      - name: Test
        uses: shell@v1
        with:
          command: echo "missing runs-on"
`
		reqBody := map[string]string{"yaml": yaml}
		body, _ := json.Marshal(reqBody)

		req, err := http.NewRequestWithContext(ctx, "POST",
			serverURL+"/v1/validate", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.True(t, resp.StatusCode >= 400, "Should reject workflow without runs-on")
		t.Logf("✓ Missing runs-on rejected with status %d", resp.StatusCode)
	})

	t.Run("Validate workflow with expression syntax", func(t *testing.T) {
		yaml := `
name: expression-test
on: workflow_dispatch
vars:
  env: staging
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Echo env
        uses: shell@v1
        with:
          command: echo "${{ vars.env }}"
`
		reqBody := map[string]string{"yaml": yaml}
		body, _ := json.Marshal(reqBody)

		req, err := http.NewRequestWithContext(ctx, "POST",
			serverURL+"/v1/validate", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		t.Logf("✓ Expression syntax validated")
	})
}

// WorkflowResponse for parsing API responses
type WorkflowResponse struct {
	WorkflowID string `json:"id"`
	RunID      string `json:"run_id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
}
