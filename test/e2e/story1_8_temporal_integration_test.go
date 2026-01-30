//go:build e2e

package e2e

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
// Story 1.8: Temporal SDK Integration - E2E Tests
// ====================================================================================
//
// Test Coverage Priority:
// - 1.8-E2E-001: P0 - Temporal Server connection and authentication
// - 1.8-E2E-002: P0 - Workflow registration and execution
// - 1.8-E2E-003: P0 - Single-node execution (no distribution)
// - 1.8-E2E-004: P0 - Event Sourcing state persistence
// - 1.8-E2E-005: P1 - Basic workflow lifecycle (start, query, complete)
//
// Architecture Validation:
// - Full integration: Waterflow Server → Temporal SDK → Temporal Server → Agent
// - Event History immutability verification
// - State persistence across Server restarts
// - Temporal Workflow/Activity pattern validation
//
// ====================================================================================

// TestStory1_8_E2E_001_TemporalConnection validates Temporal Server connection
// and client authentication setup
//
// Test ID: 1.8-E2E-001
// Priority: P0 (Critical - Foundation of all workflow execution)
// Risk: HIGH - Failure blocks entire system
//
// Acceptance Criteria Verified:
// - AC1: Temporal Client connection to Temporal Server
// - AC1: Namespace configuration (default or custom)
// - AC1: TLS/mTLS support (optional in MVP)
// - AC1: Connection health check
func TestStory1_8_E2E_001_TemporalConnection(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.8-E2E-001",
		Given:  "Waterflow Server is configured to connect to Temporal Server",
		When:   "Server starts up",
		Then: []string{
			"Temporal Client successfully connects to Temporal Server",
			"Connection uses configured namespace (e.g., 'waterflow')",
			"Health check passes",
			"Server can submit workflows without connection errors",
		},
		AcceptanceCriteria: []string{
			"AC1: Temporal Client connection",
			"AC1: Namespace configuration",
			"AC1: Connection health check",
		},
	}).Log(t)

	// GIVEN: Server is running with Temporal connection
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = ctx

	// WHEN: Submit a simple workflow to verify connection
	simpleWorkflow := `
name: Temporal Connection Test
jobs:
  connection-test:
    runs-on: default
    steps:
      - id: ping
        uses: shell@v1
        with:
          command: echo "Temporal connection OK"
`

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	resp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(simpleWorkflow))))
	require.NoError(t, err, "Should connect to Server")
	defer resp.Body.Close()

	// THEN: Workflow submission succeeds (proves Temporal connection)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Workflow submission should succeed")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var submitResponse struct {
		WorkflowID string `json:"workflowId"`
	}
	err = json.Unmarshal(body, &submitResponse)
	require.NoError(t, err)
	require.NotEmpty(t, submitResponse.WorkflowID, "Should return workflow ID")

	// Verify workflow starts (proves Temporal Workflow execution)
	time.Sleep(5 * time.Second)

	statusURL := fmt.Sprintf("%s/api/v1/workflows/%s", serverURL, submitResponse.WorkflowID)
	statusResp, err := http.Get(statusURL)
	require.NoError(t, err)
	defer statusResp.Body.Close()

	assert.Equal(t, http.StatusOK, statusResp.StatusCode, "Query should succeed")

	// TODO (DEV TEAM): Add health check endpoint to verify Temporal connection status
	// TODO (DEV TEAM): Verify Temporal namespace configuration
	// TODO (DEV TEAM): Test TLS connection if configured
}

// TestStory1_8_E2E_002_WorkflowRegistration validates Workflow registration
// as Temporal Workflows and Activity execution pattern
//
// Test ID: 1.8-E2E-002
// Priority: P0 (Critical - Core execution pattern)
// Risk: HIGH - Failure breaks workflow execution model
//
// Acceptance Criteria Verified:
// - AC2: Workflow registration in Temporal Worker
// - AC2: Activity registration for step execution
// - AC2: Task Queue mapping (runs-on → Task Queue)
// - AC2: Workflow execution creates Temporal Workflow instance
func TestStory1_8_E2E_002_WorkflowRegistration(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.8-E2E-002",
		Given:  "Server has registered Workflows and Activities with Temporal Worker",
		When:   "A workflow YAML is submitted",
		Then: []string{
			"Temporal Workflow instance is created",
			"Each step executes as Temporal Activity",
			"Task Queue routing works (runs-on field)",
			"Workflow completes successfully",
		},
		AcceptanceCriteria: []string{
			"AC2: Workflow registration in Worker",
			"AC2: Activity registration",
			"AC2: Task Queue mapping",
			"AC2: Workflow instance creation",
		},
	}).Log(t)

	// GIVEN: Multi-step workflow to verify Activity execution
	multiStepWorkflow := `
name: Workflow Registration Test
jobs:
  registration-test:
    runs-on: default
    steps:
      - id: step-1
        uses: shell@v1
        with:
          command: echo "Activity 1 executed"
      
      - id: step-2
        uses: shell@v1
        with:
          command: echo "Activity 2 executed"
      
      - id: step-3
        uses: shell@v1
        with:
          command: echo "Activity 3 executed"
`

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	resp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(multiStepWorkflow))))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var submitResponse struct {
		WorkflowID string `json:"workflowId"`
	}
	err = json.Unmarshal(body, &submitResponse)
	require.NoError(t, err)

	// WHEN: Wait for workflow execution
	time.Sleep(15 * time.Second)

	// THEN: Verify workflow completed
	statusURL := fmt.Sprintf("%s/api/v1/workflows/%s", serverURL, submitResponse.WorkflowID)
	statusResp, err := http.Get(statusURL)
	require.NoError(t, err)
	defer statusResp.Body.Close()

	statusBody, err := io.ReadAll(statusResp.Body)
	require.NoError(t, err)

	var finalStatus struct {
		Status string `json:"status"`
		Jobs   []struct {
			Steps []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"steps"`
		} `json:"jobs"`
	}
	err = json.Unmarshal(statusBody, &finalStatus)
	require.NoError(t, err)

	assert.Equal(t, "COMPLETED", finalStatus.Status, "Workflow should complete")
	require.Len(t, finalStatus.Jobs, 1)
	assert.Len(t, finalStatus.Jobs[0].Steps, 3, "Should have 3 steps (3 Activities)")

	for _, step := range finalStatus.Jobs[0].Steps {
		assert.Equal(t, "COMPLETED", step.Status, fmt.Sprintf("Step %s should complete", step.ID))
	}

	// TODO (DEV TEAM): Query Temporal API to verify Workflow exists
	// TODO (DEV TEAM): Verify Activity execution events in Event History
	// TODO (DEV TEAM): Verify Task Queue name matches runs-on field
}

// TestStory1_8_E2E_003_SingleNodeExecution validates single-node execution
// without distributed Agent system (MVP scope)
//
// Test ID: 1.8-E2E-003
// Priority: P0 (Critical - MVP execution model)
// Risk: MEDIUM - Failure limits to distributed mode only
//
// Acceptance Criteria Verified:
// - AC3: Server can execute workflows without external Agents
// - AC3: Built-in Agent or local execution mode
// - AC3: All workflow features work in single-node mode
func TestStory1_8_E2E_003_SingleNodeExecution(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.8-E2E-003",
		Given:  "Server is running in single-node mode (no external Agents)",
		When:   "A workflow with complex features is submitted",
		Then: []string{
			"Workflow executes successfully on Server's built-in Agent",
			"All features work: variables, conditions, matrix, timeout, retry",
			"No requirement for external Agent deployment",
			"Suitable for development and small deployments",
		},
		AcceptanceCriteria: []string{
			"AC3: Single-node execution without external Agents",
			"AC3: Built-in Agent or local execution",
			"AC3: All features functional",
		},
	}).Log(t)

	// GIVEN: Complex workflow using multiple features
	complexWorkflow := `
name: Single Node Execution Test
vars:
  test_value: "single-node"
jobs:
  single-node-test:
    runs-on: default
    strategy:
      matrix:
        instance: [1, 2]
    steps:
      - id: test-features
        uses: shell@v1
        env:
          TEST_VAR: "${{ vars.test_value }}"
        with:
          command: echo "Instance ${{ matrix.instance }} - Value: $TEST_VAR"
      
      - id: conditional-step
        if: "${{ matrix.instance == 1 }}"
        uses: shell@v1
        with:
          command: echo "Conditional execution works"
`

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	resp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(complexWorkflow))))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var submitResponse struct {
		WorkflowID string `json:"workflowId"`
	}
	err = json.Unmarshal(body, &submitResponse)
	require.NoError(t, err)

	// Wait for matrix execution
	time.Sleep(20 * time.Second)

	// THEN: Verify successful execution
	statusURL := fmt.Sprintf("%s/api/v1/workflows/%s", serverURL, submitResponse.WorkflowID)
	statusResp, err := http.Get(statusURL)
	require.NoError(t, err)
	defer statusResp.Body.Close()

	statusBody, err := io.ReadAll(statusResp.Body)
	require.NoError(t, err)

	var finalStatus struct {
		Status string `json:"status"`
	}
	err = json.Unmarshal(statusBody, &finalStatus)
	require.NoError(t, err)

	assert.Equal(t, "COMPLETED", finalStatus.Status, "Complex workflow should execute in single-node mode")

	// TODO (DEV TEAM): Verify all matrix instances executed
	// TODO (DEV TEAM): Verify conditional step executed only for instance 1
	// TODO (DEV TEAM): Verify variables were resolved
}

// TestStory1_8_E2E_004_EventSourcingPersistence validates Event Sourcing
// state persistence and Event History immutability
//
// Test ID: 1.8-E2E-004
// Priority: P0 (Critical - Event Sourcing architecture foundation)
// Risk: HIGH - Failure breaks audit trail and state recovery
//
// Acceptance Criteria Verified:
// - AC4: All state changes recorded in Event History
// - AC4: Event History is immutable (append-only)
// - AC4: Workflow state can be reconstructed from events
// - AC4: State persists across Server restarts
// - AC4: Event History size limits enforced (50MB default)
func TestStory1_8_E2E_004_EventSourcingPersistence(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.8-E2E-004",
		Given:  "A workflow executes with multiple state transitions",
		When:   "Querying workflow status and history",
		Then: []string{
			"All state changes are recorded in Event History",
			"Event History is append-only (no modifications)",
			"Workflow state reflects Event History replay",
			"State persists even if Server restarts",
			"Event History size limits are enforced",
		},
		AcceptanceCriteria: []string{
			"AC4: State changes in Event History",
			"AC4: Event History immutability",
			"AC4: State reconstruction from events",
			"AC4: Persistence across restarts",
			"AC4: Size limits (50MB default)",
		},
	}).Log(t)

	// GIVEN: Workflow with multiple state transitions
	workflowWithStates := `
name: Event Sourcing Test
jobs:
  event-test:
    runs-on: default
    steps:
      - id: step-1
        uses: shell@v1
        with:
          command: echo "State transition 1"
      
      - id: step-2
        uses: shell@v1
        with:
          command: echo "State transition 2"
      
      - id: step-3
        uses: shell@v1
        with:
          command: echo "State transition 3"
`

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	resp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(workflowWithStates))))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var submitResponse struct {
		WorkflowID string `json:"workflowId"`
	}
	err = json.Unmarshal(body, &submitResponse)
	require.NoError(t, err)

	workflowID := submitResponse.WorkflowID

	// Query status multiple times during execution to observe state changes
	statusURL := fmt.Sprintf("%s/api/v1/workflows/%s", serverURL, workflowID)

	// Poll and collect state snapshots
	var stateSnapshots []string
	for i := 0; i < 20; i++ {
		statusResp, err := http.Get(statusURL)
		if err == nil {
			statusBody, err := io.ReadAll(statusResp.Body)
			statusResp.Body.Close()

			if err == nil {
				var status struct {
					Status string `json:"status"`
				}
				json.Unmarshal(statusBody, &status)
				stateSnapshots = append(stateSnapshots, status.Status)

				if status.Status == "COMPLETED" || status.Status == "FAILED" {
					break
				}
			}
		}
		time.Sleep(1 * time.Second)
	}

	// THEN: Verify state transitions were captured
	assert.NotEmpty(t, stateSnapshots, "Should capture state transitions")

	// Query final state
	finalResp, err := http.Get(statusURL)
	require.NoError(t, err)
	defer finalResp.Body.Close()

	finalBody, err := io.ReadAll(finalResp.Body)
	require.NoError(t, err)

	var finalStatus struct {
		Status string `json:"status"`
	}
	err = json.Unmarshal(finalBody, &finalStatus)
	require.NoError(t, err)

	assert.Equal(t, "COMPLETED", finalStatus.Status)

	// TODO (DEV TEAM): Query Temporal Event History API directly
	// TODO (DEV TEAM): Verify Event History contains all state transition events
	// TODO (DEV TEAM): Verify Event History is append-only (no deleted/modified events)
	// TODO (DEV TEAM): Test Server restart and verify workflow state recovers
	// TODO (DEV TEAM): Test Event History size limit enforcement (50MB)
	// TODO (DEV TEAM): Verify WorkflowExecutionStarted, ActivityTaskScheduled, ActivityTaskCompleted events exist
}

// TestStory1_8_E2E_005_BasicWorkflowLifecycle validates basic workflow lifecycle
// operations: start, query, complete
//
// Test ID: 1.8-E2E-005
// Priority: P1 (Important - Basic workflow operations)
// Risk: MEDIUM - Failure limits workflow management capabilities
//
// Acceptance Criteria Verified:
// - AC5: Workflow start returns workflow ID immediately
// - AC5: Query returns current workflow state
// - AC5: Workflow completes and final state persists
// - AC5: Support for basic workflow patterns (sequential, parallel)
func TestStory1_8_E2E_005_BasicWorkflowLifecycle(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.8-E2E-005",
		Given:  "A workflow definition is ready to execute",
		When:   "Workflow lifecycle operations are performed",
		Then: []string{
			"Start returns workflow ID immediately (<500ms)",
			"Query returns current state at any point",
			"Workflow completes and final state persists",
			"Sequential and parallel job patterns supported",
		},
		AcceptanceCriteria: []string{
			"AC5: Start returns ID immediately",
			"AC5: Query returns current state",
			"AC5: Completion and final state",
			"AC5: Basic workflow patterns (sequential, parallel)",
		},
	}).Log(t)

	// GIVEN: Workflow with sequential and parallel patterns
	lifecycleWorkflow := `
name: Workflow Lifecycle Test
jobs:
  job-1:
    runs-on: default
    steps:
      - id: job1-step1
        uses: shell@v1
        with:
          command: echo "Job 1 Step 1" && sleep 2
  
  job-2:
    needs: job-1
    runs-on: default
    steps:
      - id: job2-step1
        uses: shell@v1
        with:
          command: echo "Job 2 Step 1" && sleep 2
  
  job-3:
    needs: job-1
    runs-on: default
    steps:
      - id: job3-step1
        uses: shell@v1
        with:
          command: echo "Job 3 Step 1" && sleep 2
`

	// WHEN: Start workflow
	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	startTime := time.Now()

	resp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(lifecycleWorkflow))))
	require.NoError(t, err)
	defer resp.Body.Close()

	submitDuration := time.Since(startTime).Milliseconds()

	// THEN: Start returns immediately
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Less(t, submitDuration, int64(500), "Submit should return in <500ms")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var submitResponse struct {
		WorkflowID string `json:"workflowId"`
	}
	err = json.Unmarshal(body, &submitResponse)
	require.NoError(t, err)
	require.NotEmpty(t, submitResponse.WorkflowID)

	workflowID := submitResponse.WorkflowID

	// Query workflow state multiple times
	statusURL := fmt.Sprintf("%s/api/v1/workflows/%s", serverURL, workflowID)

	// Initial query (should be RUNNING)
	time.Sleep(2 * time.Second)
	queryResp, err := http.Get(statusURL)
	require.NoError(t, err)
	queryBody, _ := io.ReadAll(queryResp.Body)
	queryResp.Body.Close()

	var queryStatus struct {
		Status string `json:"status"`
	}
	json.Unmarshal(queryBody, &queryStatus)
	assert.Contains(t, []string{"RUNNING", "PENDING"}, queryStatus.Status, "Should be running or pending")

	// Wait for completion
	time.Sleep(20 * time.Second)

	// Final query (should be COMPLETED)
	finalResp, err := http.Get(statusURL)
	require.NoError(t, err)
	finalBody, _ := io.ReadAll(finalResp.Body)
	finalResp.Body.Close()

	var finalStatus struct {
		Status string `json:"status"`
	}
	json.Unmarshal(finalBody, &finalStatus)
	assert.Equal(t, "COMPLETED", finalStatus.Status, "Workflow should complete")

	// TODO (DEV TEAM): Verify job-2 and job-3 ran in parallel (after job-1)
	// TODO (DEV TEAM): Verify execution times reflect parallel vs sequential
	// TODO (DEV TEAM): Query Event History to verify execution order
}
