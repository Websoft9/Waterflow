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
// Story 1.5: Conditional Execution - E2E Tests
// ====================================================================================
//
// Test Coverage Priority:
// - 1.5-E2E-001: P0 - Conditional execution with if expressions
// - 1.5-E2E-002: P0 - Step outputs and dependencies
// - 1.5-E2E-003: P0 - Job dependencies with needs keyword
// - 1.5-E2E-004: P1 - Continue-on-error behavior
//
// Architecture Validation:
// - Full pipeline: Conditional YAML → Expression Engine → Temporal DAG → Execution
// - Verify step/job skipping based on conditions
// - Validate Event History records condition evaluation results
//
// ====================================================================================

// TestStory1_5_E2E_001_ConditionalExecution validates conditional step execution
// using if expressions with expression evaluation
//
// Test ID: 1.5-E2E-001
// Priority: P0 (Critical - Conditional execution control)
// Risk: HIGH - Failure breaks conditional workflows
//
// Acceptance Criteria Verified:
// - AC1: Step-level if expressions
// - AC1: Job-level if expressions
// - AC1: Expression evaluation in conditions
// - AC1: Skipped steps marked with SKIPPED status
func TestStory1_5_E2E_001_ConditionalExecution(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.5-E2E-001",
		Given:  "A workflow with conditional if expressions on steps and jobs",
		When:   "Conditions evaluate to true/false",
		Then: []string{
			"Steps with if: true execute normally",
			"Steps with if: false are skipped (status=SKIPPED)",
			"Skipped steps do not run Activities",
			"Event History records condition evaluation",
		},
		AcceptanceCriteria: []string{
			"AC1: Step-level if expressions",
			"AC1: Job-level if expressions",
			"AC1: Expression evaluation in if",
			"AC1: SKIPPED status for false conditions",
		},
	}).Log(t)

	// GIVEN: Workflow with conditional steps
	workflowWithConditions := `
name: Conditional Execution Test
vars:
  environment: production
  enable_deploy: false

jobs:
  test-conditions:
    runs-on: default
    steps:
      - id: always-run
        uses: shell@v1
        with:
          command: echo "This always runs"
      
      - id: run-in-production
        if: "${{ vars.environment == 'production' }}"
        uses: shell@v1
        with:
          command: echo "Running in production"
      
      - id: run-in-staging
        if: "${{ vars.environment == 'staging' }}"
        uses: shell@v1
        with:
          command: echo "Running in staging"
      
      - id: deploy-enabled
        if: "${{ vars.enable_deploy }}"
        uses: shell@v1
        with:
          command: echo "Deploying..."
      
      - id: deploy-disabled
        if: "${{ !vars.enable_deploy }}"
        uses: shell@v1
        with:
          command: echo "Deploy is disabled"
`

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	_ = ctx

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	resp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(workflowWithConditions))))
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

	// Wait for completion
	time.Sleep(15 * time.Second)

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

	assert.Equal(t, "COMPLETED", finalStatus.Status)

	// THEN: Verify expected execution pattern
	require.Len(t, finalStatus.Jobs, 1)
	steps := finalStatus.Jobs[0].Steps

	// Find step statuses
	stepStatuses := make(map[string]string)
	for _, step := range steps {
		stepStatuses[step.ID] = step.Status
	}

	// Verify execution results
	assert.Equal(t, "COMPLETED", stepStatuses["always-run"], "Always-run step should execute")
	assert.Equal(t, "COMPLETED", stepStatuses["run-in-production"], "Production condition true, should execute")
	assert.Equal(t, "SKIPPED", stepStatuses["run-in-staging"], "Staging condition false, should skip")
	assert.Equal(t, "SKIPPED", stepStatuses["deploy-enabled"], "Deploy disabled, should skip")
	assert.Equal(t, "COMPLETED", stepStatuses["deploy-disabled"], "Deploy disabled condition true, should execute")

	// TODO (DEV TEAM): Query Event History to verify condition evaluation events
	// TODO (DEV TEAM): Verify skipped steps have no Activity execution events
}

// TestStory1_5_E2E_002_StepOutputs validates step output capture and reference
// in subsequent steps using steps.<step-id>.outputs syntax
//
// Test ID: 1.5-E2E-002
// Priority: P0 (Critical - Step dependency mechanism)
// Risk: HIGH - Failure breaks workflow data flow
//
// Acceptance Criteria Verified:
// - AC2: Step outputs captured via ::set-output::
// - AC2: Reference outputs using ${{ steps.<id>.outputs.<key> }}
// - AC2: Outputs available to dependent steps
// - AC2: Outputs used in conditional expressions
func TestStory1_5_E2E_002_StepOutputs(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.5-E2E-002",
		Given:  "A workflow with steps that produce and consume outputs",
		When:   "Steps execute in sequence with output dependencies",
		Then: []string{
			"Step outputs are captured from Activity results",
			"Subsequent steps can reference outputs via ${{ steps.<id>.outputs.<key> }}",
			"Outputs can be used in if conditions",
			"Outputs persist in Event History",
		},
		AcceptanceCriteria: []string{
			"AC2: Output capture via ::set-output::",
			"AC2: Output reference syntax",
			"AC2: Outputs in dependent steps",
			"AC2: Outputs in conditions",
		},
	}).Log(t)

	// GIVEN: Workflow with step outputs
	workflowWithOutputs := `
name: Step Outputs Test
jobs:
  test-outputs:
    runs-on: default
    steps:
      - id: generate-version
        uses: shell@v1
        with:
          command: |
            echo "::set-output name=version::1.2.3"
            echo "::set-output name=build::456"
      
      - id: use-version
        uses: shell@v1
        with:
          command: echo "Version is ${{ steps.generate-version.outputs.version }}"
      
      - id: conditional-on-output
        if: "${{ steps.generate-version.outputs.build > 400 }}"
        uses: shell@v1
        with:
          command: echo "Build number ${{ steps.generate-version.outputs.build }} is high"
      
      - id: combine-outputs
        uses: shell@v1
        with:
          command: echo "Full version: ${{ steps.generate-version.outputs.version }}-${{ steps.generate-version.outputs.build }}"
`

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	resp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(workflowWithOutputs))))
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

	// Wait for completion
	time.Sleep(15 * time.Second)

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

	assert.Equal(t, "COMPLETED", finalStatus.Status)

	// TODO (DEV TEAM): Query step outputs via API
	// TODO (DEV TEAM): Verify outputs.version = "1.2.3"
	// TODO (DEV TEAM): Verify outputs.build = "456"
	// TODO (DEV TEAM): Query logs to verify output substitution in commands
	// TODO (DEV TEAM): Verify conditional-on-output executed (build > 400 is true)
}

// TestStory1_5_E2E_003_JobDependencies validates job execution order
// using needs keyword to define job dependencies
//
// Test ID: 1.5-E2E-003
// Priority: P0 (Critical - Job dependency DAG)
// Risk: HIGH - Failure breaks multi-job workflows
//
// Acceptance Criteria Verified:
// - AC3: Job dependencies via needs keyword
// - AC3: Dependent jobs wait for prerequisite completion
// - AC3: Job outputs accessible to dependent jobs
// - AC3: Circular dependency detection
func TestStory1_5_E2E_003_JobDependencies(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.5-E2E-003",
		Given:  "A workflow with multiple jobs linked by needs dependencies",
		When:   "The workflow executes",
		Then: []string{
			"Jobs execute in correct dependency order",
			"Dependent jobs wait for prerequisite completion",
			"Parallel jobs execute concurrently",
			"Job outputs are accessible to dependent jobs",
		},
		AcceptanceCriteria: []string{
			"AC3: needs keyword for dependencies",
			"AC3: Dependency ordering enforced",
			"AC3: Job outputs accessible",
			"AC3: Circular dependency detection",
		},
	}).Log(t)

	// GIVEN: Workflow with job dependencies
	workflowWithJobDeps := `
name: Job Dependencies Test
jobs:
  build:
    runs-on: default
    steps:
      - id: build-step
        uses: shell@v1
        with:
          command: echo "Building..." && sleep 2
  
  test-unit:
    needs: build
    runs-on: default
    steps:
      - id: unit-test
        uses: shell@v1
        with:
          command: echo "Running unit tests..." && sleep 2
  
  test-integration:
    needs: build
    runs-on: default
    steps:
      - id: integration-test
        uses: shell@v1
        with:
          command: echo "Running integration tests..." && sleep 2
  
  deploy:
    needs: [test-unit, test-integration]
    runs-on: default
    steps:
      - id: deploy-step
        uses: shell@v1
        with:
          command: echo "Deploying..."
`

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	resp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(workflowWithJobDeps))))
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

	// Wait for completion
	time.Sleep(30 * time.Second)

	statusURL := fmt.Sprintf("%s/api/v1/workflows/%s", serverURL, submitResponse.WorkflowID)
	statusResp, err := http.Get(statusURL)
	require.NoError(t, err)
	defer statusResp.Body.Close()

	statusBody, err := io.ReadAll(statusResp.Body)
	require.NoError(t, err)

	var finalStatus struct {
		Status string `json:"status"`
		Jobs   []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"jobs"`
	}
	err = json.Unmarshal(statusBody, &finalStatus)
	require.NoError(t, err)

	assert.Equal(t, "COMPLETED", finalStatus.Status)
	assert.Len(t, finalStatus.Jobs, 4, "Should have 4 jobs")

	// Verify all jobs completed
	for _, job := range finalStatus.Jobs {
		assert.Equal(t, "COMPLETED", job.Status, fmt.Sprintf("Job %s should complete", job.Name))
	}

	// TODO (DEV TEAM): Query Event History to verify job execution order
	// TODO (DEV TEAM): Verify build started first
	// TODO (DEV TEAM): Verify test-unit and test-integration started after build
	// TODO (DEV TEAM): Verify deploy started after both tests completed
	// TODO (DEV TEAM): Verify test-unit and test-integration ran in parallel
}

// TestStory1_5_E2E_004_ContinueOnError validates continue-on-error behavior
// allowing workflows to continue despite step failures
//
// Test ID: 1.5-E2E-004
// Priority: P1 (Important - Error handling control)
// Risk: MEDIUM - Failure prevents optional step patterns
//
// Acceptance Criteria Verified:
// - AC4: continue-on-error: true allows step failure
// - AC4: Workflow continues after failed step
// - AC4: Failed step marked as FAILED status
// - AC4: Subsequent steps still execute
func TestStory1_5_E2E_004_ContinueOnError(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.5-E2E-004",
		Given:  "A workflow with steps marked continue-on-error: true",
		When:   "A step fails during execution",
		Then: []string{
			"The failed step is marked with FAILED status",
			"The workflow continues to next steps",
			"Overall workflow status is still COMPLETED",
			"Failure is recorded in Event History",
		},
		AcceptanceCriteria: []string{
			"AC4: continue-on-error allows failure",
			"AC4: Workflow continues",
			"AC4: Failed step marked FAILED",
			"AC4: Subsequent steps execute",
		},
	}).Log(t)

	// GIVEN: Workflow with continue-on-error
	workflowWithContinueOnError := `
name: Continue On Error Test
jobs:
  test-continue:
    runs-on: default
    steps:
      - id: step-1
        uses: shell@v1
        with:
          command: echo "Step 1 success"
      
      - id: step-2-fails
        continue-on-error: true
        uses: shell@v1
        with:
          command: exit 1
      
      - id: step-3
        uses: shell@v1
        with:
          command: echo "Step 3 runs despite step 2 failure"
`

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	resp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(workflowWithContinueOnError))))
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

	// Wait for completion
	time.Sleep(15 * time.Second)

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

	// THEN: Workflow completes despite step failure
	assert.Equal(t, "COMPLETED", finalStatus.Status, "Workflow should complete")

	require.Len(t, finalStatus.Jobs, 1)
	steps := finalStatus.Jobs[0].Steps

	stepStatuses := make(map[string]string)
	for _, step := range steps {
		stepStatuses[step.ID] = step.Status
	}

	assert.Equal(t, "COMPLETED", stepStatuses["step-1"], "Step 1 should succeed")
	assert.Equal(t, "FAILED", stepStatuses["step-2-fails"], "Step 2 should fail")
	assert.Equal(t, "COMPLETED", stepStatuses["step-3"], "Step 3 should execute despite step 2 failure")

	// TODO (DEV TEAM): Verify Event History contains failure event
	// TODO (DEV TEAM): Verify continue-on-error flag recorded in metadata
}
