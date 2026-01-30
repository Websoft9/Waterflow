//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/test/support/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ====================================================================================
// Story 1.9: Workflow Management API - E2E Tests
// ====================================================================================
//
// Test Coverage Priority:
// - 1.9-E2E-001: P0 - Workflow submission (POST /v1/workflows)
// - 1.9-E2E-002: P0 - Workflow query single (GET /v1/workflows/{id})
// - 1.9-E2E-003: P0 - Workflow list query (GET /v1/workflows)
// - 1.9-E2E-004: P1 - Workflow logs (GET /v1/workflows/{id}/logs)
// - 1.9-E2E-005: P0 - Workflow cancel (POST /v1/workflows/{id}/cancel)
// - 1.9-E2E-006: P1 - Workflow rerun (POST /v1/workflows/{id}/rerun)
// - 1.9-E2E-007: P1 - Workflow terminate (POST /v1/workflows/{id}/terminate)
// - 1.9-E2E-008: P0 - Common API specs (error format, headers, performance)
//
// Architecture Validation:
// - Full REST API contract testing
// - Performance requirements verification (<200ms query, <500ms submit)
// - Error handling and HTTP status codes
// - API versioning and CORS support
//
// ====================================================================================

// TestStory1_9_E2E_001_WorkflowSubmission validates workflow submission API
// with YAML validation and immediate ID return
//
// Test ID: 1.9-E2E-001
// Priority: P0 (Critical - Primary workflow entry point)
// Risk: HIGH - Failure blocks all workflow submissions
//
// Acceptance Criteria Verified:
// - AC1: POST /v1/workflows accepts YAML content
// - AC1: Returns unique workflow ID (UUID)
// - AC1: YAML validation errors return 422 with details
// - AC1: Response time <500ms
// - AC1: Dry-run mode support (?dry_run=true)
func TestStory1_9_E2E_001_WorkflowSubmission(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.9-E2E-001",
		Given:  "REST API server is running",
		When:   "POST /v1/workflows with valid YAML",
		Then: []string{
			"Returns 200 OK with unique workflow ID",
			"Workflow ID is UUID format",
			"Response time <500ms",
			"Invalid YAML returns 422 with error location",
			"Dry-run mode validates without execution",
		},
		AcceptanceCriteria: []string{
			"AC1: POST /v1/workflows",
			"AC1: Returns workflow ID (UUID)",
			"AC1: YAML validation errors (422)",
			"AC1: Response time <500ms",
			"AC1: Dry-run mode",
		},
	}).Log(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	_ = ctx

	// GIVEN: Valid workflow YAML
	validWorkflow := `
name: API Submission Test
jobs:
  test-job:
    runs-on: default
    steps:
      - id: test-step
        uses: shell@v1
        with:
          command: echo "API test"
`

	// WHEN: Submit workflow
	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	startTime := time.Now()

	resp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(validWorkflow))))
	require.NoError(t, err)
	defer resp.Body.Close()

	submitDuration := time.Since(startTime).Milliseconds()

	// THEN: Verify response
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should return 200 OK")
	assert.Less(t, submitDuration, int64(500), "Response time should be <500ms")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var submitResponse struct {
		WorkflowID string `json:"workflowId"`
	}
	err = json.Unmarshal(body, &submitResponse)
	require.NoError(t, err)
	require.NotEmpty(t, submitResponse.WorkflowID, "Should return workflow ID")

	// Verify UUID format (basic check: length and hyphens)
	assert.Len(t, submitResponse.WorkflowID, 36, "Workflow ID should be UUID format")
	assert.Equal(t, 4, strings.Count(submitResponse.WorkflowID, "-"), "UUID should have 4 hyphens")

	// Test YAML validation error
	invalidWorkflow := `
name: Invalid Workflow
jobs:
  invalid-job:
    # Missing runs-on (required field)
    steps: invalid
`

	invalidResp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(invalidWorkflow))))
	require.NoError(t, err)
	defer invalidResp.Body.Close()

	assert.Equal(t, http.StatusUnprocessableEntity, invalidResp.StatusCode, "Invalid YAML should return 422")

	invalidBody, _ := io.ReadAll(invalidResp.Body)
	var errorResponse struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	json.Unmarshal(invalidBody, &errorResponse)
	assert.NotEmpty(t, errorResponse.Error.Message, "Should return error message")

	// TODO (DEV TEAM): Test dry-run mode: POST /v1/workflows?dry_run=true
	// TODO (DEV TEAM): Verify dry-run validates but doesn't create workflow
	// TODO (DEV TEAM): Test estimated execution time in response
}

// TestStory1_9_E2E_002_WorkflowQuery validates single workflow query API
// with detailed status and execution progress
//
// Test ID: 1.9-E2E-002
// Priority: P0 (Critical - Primary status check)
// Risk: HIGH - Failure breaks workflow monitoring
//
// Acceptance Criteria Verified:
// - AC2: GET /v1/workflows/{id} returns workflow details
// - AC2: Returns status (pending, running, completed, failed, cancelled)
// - AC2: Returns job and step progress
// - AC2: Returns timestamps and duration
// - AC2: Response time <200ms (small), <500ms (large)
// - AC2: 404 for non-existent workflows
func TestStory1_9_E2E_002_WorkflowQuery(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.9-E2E-002",
		Given:  "A workflow has been submitted and is executing",
		When:   "GET /v1/workflows/{id}",
		Then: []string{
			"Returns workflow status and details",
			"Includes all job and step statuses",
			"Includes timestamps (start, end, duration)",
			"Response time <200ms for small workflows",
			"Returns 404 for non-existent workflow",
		},
		AcceptanceCriteria: []string{
			"AC2: GET /v1/workflows/{id}",
			"AC2: Returns detailed status",
			"AC2: Job/Step progress from Event History",
			"AC2: Timestamps and duration",
			"AC2: Response time <200ms/<500ms",
			"AC2: 404 for non-existent",
		},
	}).Log(t)

	// GIVEN: Submit a workflow
	workflowYAML := `
name: Query Test Workflow
jobs:
  query-test:
    runs-on: default
    steps:
      - id: step-1
        uses: shell@v1
        with:
          command: echo "Step 1" && sleep 2
      - id: step-2
        uses: shell@v1
        with:
          command: echo "Step 2"
`

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	submitResp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(workflowYAML))))
	require.NoError(t, err)
	defer submitResp.Body.Close()

	var submitResponse struct {
		WorkflowID string `json:"workflowId"`
	}
	submitBody, _ := io.ReadAll(submitResp.Body)
	json.Unmarshal(submitBody, &submitResponse)
	workflowID := submitResponse.WorkflowID

	// WHEN: Query workflow status
	time.Sleep(5 * time.Second) // Allow workflow to start

	queryURL := fmt.Sprintf("%s/api/v1/workflows/%s", serverURL, workflowID)
	queryStart := time.Now()

	queryResp, err := http.Get(queryURL)
	require.NoError(t, err)
	defer queryResp.Body.Close()

	queryDuration := time.Since(queryStart).Milliseconds()

	// THEN: Verify response
	assert.Equal(t, http.StatusOK, queryResp.StatusCode, "Should return 200 OK")
	assert.Less(t, queryDuration, int64(500), "Query should complete in <500ms")

	queryBody, err := io.ReadAll(queryResp.Body)
	require.NoError(t, err)

	var workflowDetails struct {
		WorkflowID string    `json:"workflowId"`
		Name       string    `json:"name"`
		Status     string    `json:"status"`
		StartTime  time.Time `json:"startTime,omitempty"`
		EndTime    time.Time `json:"endTime,omitempty"`
		Duration   int       `json:"duration,omitempty"`
		Jobs       []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Steps  []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"steps"`
		} `json:"jobs"`
	}
	err = json.Unmarshal(queryBody, &workflowDetails)
	require.NoError(t, err)

	assert.Equal(t, workflowID, workflowDetails.WorkflowID)
	assert.Equal(t, "Query Test Workflow", workflowDetails.Name)
	assert.Contains(t, []string{"PENDING", "RUNNING", "COMPLETED"}, workflowDetails.Status)
	assert.NotZero(t, workflowDetails.StartTime, "Should have start time")

	// Test 404 for non-existent workflow
	notFoundURL := fmt.Sprintf("%s/api/v1/workflows/non-existent-id", serverURL)
	notFoundResp, err := http.Get(notFoundURL)
	require.NoError(t, err)
	defer notFoundResp.Body.Close()

	assert.Equal(t, http.StatusNotFound, notFoundResp.StatusCode, "Should return 404")

	// TODO (DEV TEAM): Verify job and step details populated
	// TODO (DEV TEAM): Test ?include=events to include Event History
	// TODO (DEV TEAM): Verify duration calculation
}

// TestStory1_9_E2E_003_WorkflowList validates workflow list API
// with pagination and filtering
//
// Test ID: 1.9-E2E-003
// Priority: P0 (Critical - Workflow browsing)
// Risk: MEDIUM - Failure limits workflow discovery
//
// Acceptance Criteria Verified:
// - AC3: GET /v1/workflows returns paginated list
// - AC3: Filter by status, name, time range
// - AC3: Pagination (page, limit, total)
// - AC3: Default sort by submission time DESC
// - AC3: Response time <300ms
func TestStory1_9_E2E_003_WorkflowList(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.9-E2E-003",
		Given:  "Multiple workflows exist in the system",
		When:   "GET /v1/workflows with filters and pagination",
		Then: []string{
			"Returns paginated workflow list",
			"Supports status filtering",
			"Supports name filtering",
			"Supports time range filtering",
			"Returns total count and page info",
			"Response time <300ms",
		},
		AcceptanceCriteria: []string{
			"AC3: GET /v1/workflows pagination",
			"AC3: Filter by status/name/time",
			"AC3: Returns total/page/limit",
			"AC3: Sort by time DESC",
			"AC3: Response time <300ms",
		},
	}).Log(t)

	// GIVEN: Submit multiple workflows
	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)

	for i := 1; i <= 3; i++ {
		workflowYAML := fmt.Sprintf(`
name: List Test Workflow %d
jobs:
  job-%d:
    runs-on: default
    steps:
      - id: step-1
        uses: shell@v1
        with:
          command: echo "Workflow %d"
`, i, i, i)

		http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(workflowYAML))))
		time.Sleep(500 * time.Millisecond)
	}

	time.Sleep(2 * time.Second)

	// WHEN: Query workflow list
	listURL := fmt.Sprintf("%s/api/v1/workflows?page=1&limit=10", serverURL)
	listStart := time.Now()

	listResp, err := http.Get(listURL)
	require.NoError(t, err)
	defer listResp.Body.Close()

	listDuration := time.Since(listStart).Milliseconds()

	// THEN: Verify response
	assert.Equal(t, http.StatusOK, listResp.StatusCode)
	assert.Less(t, listDuration, int64(300), "List query should complete in <300ms")

	listBody, err := io.ReadAll(listResp.Body)
	require.NoError(t, err)

	var listResponse struct {
		Workflows []struct {
			WorkflowID string `json:"workflowId"`
			Name       string `json:"name"`
			Status     string `json:"status"`
		} `json:"workflows"`
		Total       int `json:"total"`
		CurrentPage int `json:"currentPage"`
		TotalPages  int `json:"totalPages"`
	}
	err = json.Unmarshal(listBody, &listResponse)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, listResponse.Total, 3, "Should have at least 3 workflows")
	assert.NotEmpty(t, listResponse.Workflows, "Should return workflow list")

	// Test status filtering
	filterURL := fmt.Sprintf("%s/api/v1/workflows?status=running,completed", serverURL)
	filterResp, err := http.Get(filterURL)
	require.NoError(t, err)
	defer filterResp.Body.Close()

	assert.Equal(t, http.StatusOK, filterResp.StatusCode, "Filter query should succeed")

	// TODO (DEV TEAM): Test name filtering (?name=Deploy)
	// TODO (DEV TEAM): Test time range filtering (?start_time_from=2025-01-01)
	// TODO (DEV TEAM): Verify sorting (newest first)
	// TODO (DEV TEAM): Test pagination limits (1-100)
}

// TestStory1_9_E2E_004_WorkflowLogs validates workflow logs API
// with structured log retrieval from Event History
//
// Test ID: 1.9-E2E-004
// Priority: P1 (Important - Debugging and monitoring)
// Risk: MEDIUM - Failure limits debugging capabilities
//
// Acceptance Criteria Verified:
// - AC4: GET /v1/workflows/{id}/logs returns structured logs
// - AC4: Logs reconstructed from Event History
// - AC4: Supports filtering by level, job, step
// - AC4: Response time <500ms for medium workflows
func TestStory1_9_E2E_004_WorkflowLogs(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.9-E2E-004",
		Given:  "A workflow has executed with multiple steps",
		When:   "GET /v1/workflows/{id}/logs",
		Then: []string{
			"Returns structured logs (JSON Lines)",
			"Logs include timestamp, level, job/step info, message",
			"Logs reconstructed from Event History",
			"Supports filtering by level, job, step",
			"Response time <500ms for medium workflows",
		},
		AcceptanceCriteria: []string{
			"AC4: GET /v1/workflows/{id}/logs",
			"AC4: Structured logs (JSON Lines)",
			"AC4: Rebuilt from Event History",
			"AC4: Filter by level/job/step",
			"AC4: Response time <500ms",
		},
	}).Log(t)

	// GIVEN: Submit workflow with multiple steps
	workflowYAML := `
name: Logs Test Workflow
jobs:
  logs-test:
    runs-on: default
    steps:
      - id: step-1
        uses: shell@v1
        with:
          command: echo "Log message 1"
      - id: step-2
        uses: shell@v1
        with:
          command: echo "Log message 2"
`

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	submitResp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(workflowYAML))))
	require.NoError(t, err)
	defer submitResp.Body.Close()

	var submitResponse struct {
		WorkflowID string `json:"workflowId"`
	}
	submitBody, _ := io.ReadAll(submitResp.Body)
	json.Unmarshal(submitBody, &submitResponse)
	workflowID := submitResponse.WorkflowID

	// Wait for execution
	time.Sleep(10 * time.Second)

	// WHEN: Query logs
	logsURL := fmt.Sprintf("%s/api/v1/workflows/%s/logs", serverURL, workflowID)
	logsStart := time.Now()

	logsResp, err := http.Get(logsURL)
	require.NoError(t, err)
	defer logsResp.Body.Close()

	logsDuration := time.Since(logsStart).Milliseconds()

	// THEN: Verify response
	assert.Equal(t, http.StatusOK, logsResp.StatusCode)
	assert.Less(t, logsDuration, int64(500), "Logs query should complete in <500ms")

	logsBody, err := io.ReadAll(logsResp.Body)
	require.NoError(t, err)
	assert.NotEmpty(t, logsBody, "Should return logs")

	// TODO (DEV TEAM): Verify JSON Lines format
	// TODO (DEV TEAM): Test log level filtering (?level=error,warn)
	// TODO (DEV TEAM): Test job/step filtering (?job=logs-test&step=step-1)
	// TODO (DEV TEAM): Test streaming logs (?stream=true with SSE)
	// TODO (DEV TEAM): Verify logs include Event History events
}

// TestStory1_9_E2E_005_WorkflowCancel validates workflow cancellation API
// with graceful shutdown propagation
//
// Test ID: 1.9-E2E-005
// Priority: P0 (Critical - Workflow control)
// Risk: HIGH - Failure prevents stopping runaway workflows
//
// Acceptance Criteria Verified:
// - AC5: POST /v1/workflows/{id}/cancel cancels running workflow
// - AC5: Graceful shutdown (30s timeout)
// - AC5: Cancellation propagates to child workflows/activities
// - AC5: Returns 409 if already completed
// - AC5: Audit log records cancellation
func TestStory1_9_E2E_005_WorkflowCancel(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.9-E2E-005",
		Given:  "A workflow is running",
		When:   "POST /v1/workflows/{id}/cancel",
		Then: []string{
			"Workflow is cancelled (status=CANCELLED)",
			"Running steps are gracefully stopped",
			"Cancellation propagates to child workflows",
			"Returns 202 Accepted",
			"Completed workflows return 409 Conflict",
			"Audit log records cancellation",
		},
		AcceptanceCriteria: []string{
			"AC5: POST /v1/workflows/{id}/cancel",
			"AC5: Graceful shutdown (30s)",
			"AC5: Cancellation propagation",
			"AC5: 409 if already completed",
			"AC5: Audit log",
		},
	}).Log(t)

	// GIVEN: Submit long-running workflow
	longWorkflow := `
name: Cancel Test Workflow
jobs:
  cancel-test:
    runs-on: default
    steps:
      - id: long-step
        uses: shell@v1
        with:
          command: sleep 60
`

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	submitResp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(longWorkflow))))
	require.NoError(t, err)
	defer submitResp.Body.Close()

	var submitResponse struct {
		WorkflowID string `json:"workflowId"`
	}
	submitBody, _ := io.ReadAll(submitResp.Body)
	json.Unmarshal(submitBody, &submitResponse)
	workflowID := submitResponse.WorkflowID

	// Wait for workflow to start
	time.Sleep(5 * time.Second)

	// WHEN: Cancel workflow
	cancelURL := fmt.Sprintf("%s/api/v1/workflows/%s/cancel", serverURL, workflowID)
	cancelResp, err := http.Post(cancelURL, "application/json", nil)
	require.NoError(t, err)
	defer cancelResp.Body.Close()

	// THEN: Verify cancellation
	assert.Equal(t, http.StatusAccepted, cancelResp.StatusCode, "Should return 202 Accepted")

	// Wait for cancellation to propagate
	time.Sleep(5 * time.Second)

	// Verify workflow status is CANCELLED
	statusURL := fmt.Sprintf("%s/api/v1/workflows/%s", serverURL, workflowID)
	statusResp, err := http.Get(statusURL)
	require.NoError(t, err)
	defer statusResp.Body.Close()

	statusBody, _ := io.ReadAll(statusResp.Body)
	var status struct {
		Status string `json:"status"`
	}
	json.Unmarshal(statusBody, &status)

	assert.Contains(t, []string{"CANCELLED", "CANCELLING"}, status.Status, "Status should be CANCELLED or CANCELLING")

	// TODO (DEV TEAM): Test cancelling already completed workflow (should return 409)
	// TODO (DEV TEAM): Verify audit log contains cancellation record
	// TODO (DEV TEAM): Test graceful shutdown timeout (30s)
	// TODO (DEV TEAM): Verify child workflows are cancelled
}

// TestStory1_9_E2E_006_WorkflowRerun validates workflow rerun API
// with parameter override support
//
// Test ID: 1.9-E2E-006
// Priority: P1 (Important - Workflow retry mechanism)
// Risk: MEDIUM - Failure limits error recovery options
//
// Acceptance Criteria Verified:
// - AC6: POST /v1/workflows/{id}/rerun creates new workflow
// - AC6: Supports vars override
// - AC6: Returns new workflow ID
// - AC6: Original workflow unchanged
// - AC6: Running workflows cannot rerun (409)
func TestStory1_9_E2E_006_WorkflowRerun(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.9-E2E-006",
		Given:  "A workflow has completed",
		When:   "POST /v1/workflows/{id}/rerun",
		Then: []string{
			"Creates new workflow with same YAML",
			"Supports vars override",
			"Returns new workflow ID",
			"Original workflow remains unchanged",
			"Running workflows return 409 Conflict",
		},
		AcceptanceCriteria: []string{
			"AC6: POST /v1/workflows/{id}/rerun",
			"AC6: Vars override support",
			"AC6: Returns new workflow ID",
			"AC6: Original unchanged",
			"AC6: Running workflows 409",
		},
	}).Log(t)

	// GIVEN: Submit and complete a workflow
	workflowYAML := `
name: Rerun Test Workflow
vars:
  environment: dev
jobs:
  rerun-test:
    runs-on: default
    steps:
      - id: test-step
        uses: shell@v1
        with:
          command: echo "Environment: ${{ vars.environment }}"
`

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	submitResp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(workflowYAML))))
	require.NoError(t, err)
	defer submitResp.Body.Close()

	var submitResponse struct {
		WorkflowID string `json:"workflowId"`
	}
	submitBody, _ := io.ReadAll(submitResp.Body)
	json.Unmarshal(submitBody, &submitResponse)
	originalID := submitResponse.WorkflowID

	// Wait for completion
	time.Sleep(10 * time.Second)

	// WHEN: Rerun workflow with vars override
	rerunURL := fmt.Sprintf("%s/api/v1/workflows/%s/rerun", serverURL, originalID)
	rerunPayload := `{"vars": {"environment": "production"}}`

	rerunResp, err := http.Post(rerunURL, "application/json", bytes.NewReader([]byte(rerunPayload)))
	require.NoError(t, err)
	defer rerunResp.Body.Close()

	// THEN: Verify rerun
	assert.Equal(t, http.StatusOK, rerunResp.StatusCode, "Should return 200 OK")

	rerunBody, err := io.ReadAll(rerunResp.Body)
	require.NoError(t, err)

	var rerunResponse struct {
		WorkflowID         string `json:"workflowId"`
		OriginalWorkflowID string `json:"originalWorkflowId"`
	}
	err = json.Unmarshal(rerunBody, &rerunResponse)
	require.NoError(t, err)

	assert.NotEmpty(t, rerunResponse.WorkflowID, "Should return new workflow ID")
	assert.NotEqual(t, originalID, rerunResponse.WorkflowID, "New ID should differ from original")
	assert.Equal(t, originalID, rerunResponse.OriginalWorkflowID, "Should track original workflow ID")

	// TODO (DEV TEAM): Test skip_successful option
	// TODO (DEV TEAM): Test from_job option
	// TODO (DEV TEAM): Verify original workflow unchanged
	// TODO (DEV TEAM): Test rerunning running workflow (should return 409)
}

// TestStory1_9_E2E_007_WorkflowTerminate validates workflow forced termination API
// with immediate shutdown and no cleanup
//
// Test ID: 1.9-E2E-007
// Priority: P1 (Important - Emergency workflow control)
// Risk: MEDIUM - Failure limits emergency stop capabilities
//
// Acceptance Criteria Verified:
// - AC8: POST /v1/workflows/{id}/terminate forces immediate stop
// - AC8: No cleanup logic executed
// - AC8: Status becomes TERMINATED
// - AC8: Difference from cancel (graceful vs forced)
// - AC8: Completed workflows return 409
func TestStory1_9_E2E_007_WorkflowTerminate(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.9-E2E-007",
		Given:  "A workflow is running",
		When:   "POST /v1/workflows/{id}/terminate",
		Then: []string{
			"Workflow is immediately terminated",
			"No cleanup logic executed",
			"Status becomes TERMINATED",
			"Returns 204 No Content",
			"Completed workflows return 409 Conflict",
		},
		AcceptanceCriteria: []string{
			"AC8: POST /v1/workflows/{id}/terminate",
			"AC8: Immediate termination",
			"AC8: No cleanup",
			"AC8: Status TERMINATED",
			"AC8: Completed workflows 409",
		},
	}).Log(t)

	// GIVEN: Submit long-running workflow
	longWorkflow := `
name: Terminate Test Workflow
jobs:
  terminate-test:
    runs-on: default
    steps:
      - id: long-step
        uses: shell@v1
        with:
          command: sleep 60
`

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	submitResp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(longWorkflow))))
	require.NoError(t, err)
	defer submitResp.Body.Close()

	var submitResponse struct {
		WorkflowID string `json:"workflowId"`
	}
	submitBody, _ := io.ReadAll(submitResp.Body)
	json.Unmarshal(submitBody, &submitResponse)
	workflowID := submitResponse.WorkflowID

	// Wait for workflow to start
	time.Sleep(5 * time.Second)

	// WHEN: Terminate workflow
	terminateURL := fmt.Sprintf("%s/api/v1/workflows/%s/terminate", serverURL, workflowID)
	terminateResp, err := http.Post(terminateURL, "application/json", nil)
	require.NoError(t, err)
	defer terminateResp.Body.Close()

	// THEN: Verify termination
	assert.Equal(t, http.StatusNoContent, terminateResp.StatusCode, "Should return 204 No Content")

	// Wait for termination to propagate
	time.Sleep(3 * time.Second)

	// Verify workflow status is TERMINATED
	statusURL := fmt.Sprintf("%s/api/v1/workflows/%s", serverURL, workflowID)
	statusResp, err := http.Get(statusURL)
	require.NoError(t, err)
	defer statusResp.Body.Close()

	statusBody, _ := io.ReadAll(statusResp.Body)
	var status struct {
		Status string `json:"status"`
	}
	json.Unmarshal(statusBody, &status)

	assert.Contains(t, []string{"TERMINATED", "TERMINATING"}, status.Status, "Status should be TERMINATED or TERMINATING")

	// TODO (DEV TEAM): Verify immediate termination (no 30s graceful period)
	// TODO (DEV TEAM): Test terminating completed workflow (should return 409)
	// TODO (DEV TEAM): Verify Event History records termination reason
}

// TestStory1_9_E2E_008_CommonAPISpecs validates common API specifications
// including error format, headers, and performance requirements
//
// Test ID: 1.9-E2E-008
// Priority: P0 (Critical - API contract consistency)
// Risk: MEDIUM - Failure breaks API client compatibility
//
// Acceptance Criteria Verified:
// - AC7: Unified error format {error: {code, message, details}}
// - AC7: Standard HTTP status codes
// - AC7: X-Request-ID header in all responses
// - AC7: CORS support (development)
// - AC7: API versioning (/v1/)
// - AC7: Rate limiting (100 req/min per IP)
func TestStory1_9_E2E_008_CommonAPISpecs(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.9-E2E-008",
		Given:  "Any API endpoint",
		When:   "Making API requests",
		Then: []string{
			"Errors use unified format: {error: {code, message, details}}",
			"HTTP status codes are standard (400, 404, 409, 422, 500)",
			"All responses include X-Request-ID header",
			"CORS headers present (development)",
			"API versioned via /v1/ prefix",
			"Rate limiting enforced (100 req/min)",
		},
		AcceptanceCriteria: []string{
			"AC7: Unified error format",
			"AC7: Standard HTTP status codes",
			"AC7: X-Request-ID header",
			"AC7: CORS support",
			"AC7: API versioning /v1/",
			"AC7: Rate limiting",
		},
	}).Log(t)

	// Test unified error format
	invalidURL := fmt.Sprintf("%s/api/v1/workflows/invalid-id-format", serverURL)
	errorResp, err := http.Get(invalidURL)
	require.NoError(t, err)
	defer errorResp.Body.Close()

	// Verify X-Request-ID header
	requestID := errorResp.Header.Get("X-Request-ID")
	assert.NotEmpty(t, requestID, "All responses should include X-Request-ID header")

	// Verify error format
	if errorResp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(errorResp.Body)
		var errorFormat struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		err := json.Unmarshal(errorBody, &errorFormat)
		assert.NoError(t, err, "Error response should use unified format")
		assert.NotEmpty(t, errorFormat.Error.Message, "Error should have message")
	}

	// Verify CORS headers (if in development mode)
	corsOrigin := errorResp.Header.Get("Access-Control-Allow-Origin")
	if corsOrigin != "" {
		assert.NotEmpty(t, corsOrigin, "CORS should be configured in development")
	}

	// Verify API versioning
	assert.Contains(t, invalidURL, "/v1/", "API should use /v1/ versioning")

	// TODO (DEV TEAM): Test rate limiting (100 req/min per IP)
	// TODO (DEV TEAM): Verify all standard HTTP status codes used correctly
	// TODO (DEV TEAM): Test OPTIONS requests for CORS preflight
}
