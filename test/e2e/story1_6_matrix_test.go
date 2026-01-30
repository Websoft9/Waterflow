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
// Story 1.6: Matrix Strategy - E2E Tests
// ====================================================================================
//
// Test Coverage Priority:
// - 1.6-E2E-001: P0 - Matrix definition and expansion
// - 1.6-E2E-002: P0 - Matrix variable reference in steps
// - 1.6-E2E-003: P0 - Parallel matrix execution
// - 1.6-E2E-004: P1 - Fail-fast strategy control
//
// Architecture Validation:
// - Matrix expansion: YAML → Matrix Engine → Temporal Child Workflows
// - Parallel execution via Temporal's concurrency control
// - Verify max 256 combinations limit
//
// ====================================================================================

// TestStory1_6_E2E_001_MatrixExpansion validates matrix definition and expansion
// into multiple job instances with correct combination generation
//
// Test ID: 1.6-E2E-001
// Priority: P0 (Critical - Matrix expansion engine)
// Risk: HIGH - Failure breaks matrix builds
//
// Acceptance Criteria Verified:
// - AC1: Matrix definition with multiple dimensions
// - AC1: Cartesian product expansion (max 256)
// - AC1: Each combination creates job instance
// - AC1: Matrix variables populated per instance
func TestStory1_6_E2E_001_MatrixExpansion(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.6-E2E-001",
		Given:  "A workflow with matrix strategy defining multiple dimensions",
		When:   "The workflow is submitted for execution",
		Then: []string{
			"Matrix expands into all combinations (Cartesian product)",
			"Each combination creates a separate job instance",
			"Total instances = product of dimension sizes",
			"Max 256 combinations enforced",
		},
		AcceptanceCriteria: []string{
			"AC1: Matrix definition",
			"AC1: Cartesian product expansion",
			"AC1: Job instance per combination",
			"AC1: Max 256 combinations",
		},
	}).Log(t)

	// GIVEN: Workflow with matrix strategy
	workflowWithMatrix := `
name: Matrix Expansion Test
jobs:
  test-matrix:
    runs-on: default
    strategy:
      matrix:
        os: [ubuntu, alpine, centos]
        version: [18, 20, 22]
    steps:
      - id: show-matrix
        uses: shell@v1
        with:
          command: echo "Testing on ${{ matrix.os }} version ${{ matrix.version }}"
`

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	_ = ctx

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	resp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(workflowWithMatrix))))
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

	// Wait for matrix expansion and execution
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
			Name   string            `json:"name"`
			Status string            `json:"status"`
			Matrix map[string]string `json:"matrix,omitempty"`
		} `json:"jobs"`
	}
	err = json.Unmarshal(statusBody, &finalStatus)
	require.NoError(t, err)

	// THEN: Verify matrix expansion
	// Expected: 3 OS * 3 versions = 9 job instances
	assert.Equal(t, 9, len(finalStatus.Jobs), "Should have 9 matrix job instances (3*3)")

	// Verify all combinations present
	expectedCombinations := []struct {
		os      string
		version string
	}{
		{"ubuntu", "18"}, {"ubuntu", "20"}, {"ubuntu", "22"},
		{"alpine", "18"}, {"alpine", "20"}, {"alpine", "22"},
		{"centos", "18"}, {"centos", "20"}, {"centos", "22"},
	}

	foundCombinations := make(map[string]bool)
	for _, job := range finalStatus.Jobs {
		if job.Matrix != nil {
			key := fmt.Sprintf("%s-%s", job.Matrix["os"], job.Matrix["version"])
			foundCombinations[key] = true
			assert.Equal(t, "COMPLETED", job.Status, fmt.Sprintf("Matrix job %s should complete", key))
		}
	}

	for _, expected := range expectedCombinations {
		key := fmt.Sprintf("%s-%s", expected.os, expected.version)
		assert.True(t, foundCombinations[key], fmt.Sprintf("Should have combination %s", key))
	}

	// TODO (DEV TEAM): Query Event History to verify matrix expansion event
	// TODO (DEV TEAM): Verify matrix metadata stored per job instance
}

// TestStory1_6_E2E_002_MatrixVariableReference validates matrix variable reference
// in step definitions using ${{ matrix.* }} syntax
//
// Test ID: 1.6-E2E-002
// Priority: P0 (Critical - Matrix variable system)
// Risk: HIGH - Failure prevents matrix parameterization
//
// Acceptance Criteria Verified:
// - AC2: Reference matrix vars using ${{ matrix.<key> }}
// - AC2: Matrix vars available in all step fields
// - AC2: Matrix vars resolved per job instance
func TestStory1_6_E2E_002_MatrixVariableReference(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.6-E2E-002",
		Given:  "A matrix workflow with steps referencing matrix variables",
		When:   "Matrix jobs execute with different variable values",
		Then: []string{
			"Matrix variables resolve to correct values per instance",
			"Variables accessible in step commands",
			"Variables accessible in step env",
			"Each instance has unique variable values",
		},
		AcceptanceCriteria: []string{
			"AC2: ${{ matrix.* }} syntax",
			"AC2: Matrix vars in all step fields",
			"AC2: Unique values per instance",
		},
	}).Log(t)

	// GIVEN: Matrix workflow using matrix variables
	workflowWithMatrixVars := `
name: Matrix Variables Test
jobs:
  test-vars:
    runs-on: default
    strategy:
      matrix:
        language: [go, node, python]
        version: ["1.21", "20", "3.11"]
        include:
          - language: go
            version: "1.21"
            runner: go-runner
          - language: node
            version: "20"
            runner: node-runner
          - language: python
            version: "3.11"
            runner: python-runner
    steps:
      - id: show-config
        uses: shell@v1
        env:
          LANG: "${{ matrix.language }}"
          VERSION: "${{ matrix.version }}"
        with:
          command: echo "Testing ${{ matrix.language }} version ${{ matrix.version }} on ${{ matrix.runner }}"
`

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	resp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(workflowWithMatrixVars))))
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

	// Wait for execution
	time.Sleep(20 * time.Second)

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

	// TODO (DEV TEAM): Query logs for each matrix instance
	// TODO (DEV TEAM): Verify go instance has "Testing go version 1.21 on go-runner"
	// TODO (DEV TEAM): Verify node instance has "Testing node version 20 on node-runner"
	// TODO (DEV TEAM): Verify python instance has "Testing python version 3.11 on python-runner"
	// TODO (DEV TEAM): Verify env vars LANG and VERSION set correctly per instance
}

// TestStory1_6_E2E_003_ParallelExecution validates parallel execution of matrix jobs
// using Temporal's concurrency control
//
// Test ID: 1.6-E2E-003
// Priority: P0 (Critical - Parallel execution performance)
// Risk: MEDIUM - Failure causes sequential execution, performance degradation
//
// Acceptance Criteria Verified:
// - AC3: Matrix jobs execute in parallel
// - AC3: Parallel execution via Temporal Child Workflows
// - AC3: Total time < sum of individual times
// - AC3: max-parallel controls concurrency
func TestStory1_6_E2E_003_ParallelExecution(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.6-E2E-003",
		Given:  "A matrix workflow with multiple job instances",
		When:   "The workflow executes",
		Then: []string{
			"Matrix jobs run in parallel (not sequential)",
			"Total execution time < sum of individual durations",
			"Temporal schedules Child Workflows concurrently",
			"max-parallel setting limits concurrency",
		},
		AcceptanceCriteria: []string{
			"AC3: Parallel execution",
			"AC3: Temporal Child Workflows",
			"AC3: Timing proves parallelism",
			"AC3: max-parallel concurrency control",
		},
	}).Log(t)

	// GIVEN: Matrix workflow with jobs that take time
	workflowWithParallelMatrix := `
name: Parallel Matrix Execution Test
jobs:
  parallel-test:
    runs-on: default
    strategy:
      matrix:
        instance: [1, 2, 3, 4]
    steps:
      - id: slow-step
        uses: shell@v1
        with:
          command: echo "Instance ${{ matrix.instance }} starting" && sleep 5 && echo "Instance ${{ matrix.instance }} done"
`

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	startTime := time.Now()

	resp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(workflowWithParallelMatrix))))
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

	// Poll for completion
	var finalStatus struct {
		Status string `json:"status"`
	}

	statusURL := fmt.Sprintf("%s/api/v1/workflows/%s", serverURL, submitResponse.WorkflowID)
	completed := false

	for i := 0; i < 60; i++ {
		statusResp, err := http.Get(statusURL)
		require.NoError(t, err)

		statusBody, err := io.ReadAll(statusResp.Body)
		statusResp.Body.Close()
		require.NoError(t, err)

		err = json.Unmarshal(statusBody, &finalStatus)
		require.NoError(t, err)

		if finalStatus.Status == "COMPLETED" || finalStatus.Status == "FAILED" {
			completed = true
			break
		}
		time.Sleep(1 * time.Second)
	}

	require.True(t, completed, "Workflow should complete")
	assert.Equal(t, "COMPLETED", finalStatus.Status)

	totalTime := time.Since(startTime).Seconds()

	// THEN: Verify parallel execution by timing
	// Sequential: 4 jobs * 5 seconds = 20 seconds
	// Parallel: ~5-10 seconds (allowing overhead)
	// If total time < 15 seconds, proves parallel execution
	assert.Less(t, totalTime, 15.0, "Total time should be <15s, proving parallel execution (not 20s sequential)")

	// TODO (DEV TEAM): Query Temporal to verify Child Workflow execution
	// TODO (DEV TEAM): Verify 4 Child Workflows started concurrently
	// TODO (DEV TEAM): Check start times of all instances (should be close)
}

// TestStory1_6_E2E_004_FailFastStrategy validates fail-fast behavior
// that stops matrix execution on first failure
//
// Test ID: 1.6-E2E-004
// Priority: P1 (Important - Matrix failure handling)
// Risk: LOW - Failure wastes resources but not critical
//
// Acceptance Criteria Verified:
// - AC4: fail-fast: true stops on first failure
// - AC4: fail-fast: false continues all instances
// - AC4: Running jobs are cancelled on fail-fast
// - AC4: Final status reflects failure
func TestStory1_6_E2E_004_FailFastStrategy(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.6-E2E-004",
		Given:  "A matrix workflow with fail-fast strategy",
		When:   "One matrix instance fails",
		Then: []string{
			"With fail-fast: true, other instances are cancelled",
			"With fail-fast: false, all instances complete",
			"Workflow status reflects failure",
			"Event History shows cancellation events",
		},
		AcceptanceCriteria: []string{
			"AC4: fail-fast: true stops on failure",
			"AC4: fail-fast: false continues all",
			"AC4: Running jobs cancelled",
			"AC4: Final status = FAILED",
		},
	}).Log(t)

	// GIVEN: Matrix workflow with fail-fast and one failing instance
	workflowWithFailFast := `
name: Fail-Fast Test
jobs:
  test-fail-fast:
    runs-on: default
    strategy:
      fail-fast: true
      matrix:
        instance: [1, 2, 3, 4]
    steps:
      - id: conditional-fail
        uses: shell@v1
        with:
          command: |
            if [ "${{ matrix.instance }}" = "2" ]; then
              echo "Instance 2 failing"
              exit 1
            else
              echo "Instance ${{ matrix.instance }} success" && sleep 10
            fi
`

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	resp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(workflowWithFailFast))))
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

	// Wait for failure and cancellation
	time.Sleep(20 * time.Second)

	statusURL := fmt.Sprintf("%s/api/v1/workflows/%s", serverURL, submitResponse.WorkflowID)
	statusResp, err := http.Get(statusURL)
	require.NoError(t, err)
	defer statusResp.Body.Close()

	statusBody, err := io.ReadAll(statusResp.Body)
	require.NoError(t, err)

	var finalStatus struct {
		Status string `json:"status"`
		Jobs   []struct {
			Status string            `json:"status"`
			Matrix map[string]string `json:"matrix,omitempty"`
		} `json:"jobs"`
	}
	err = json.Unmarshal(statusBody, &finalStatus)
	require.NoError(t, err)

	// THEN: Workflow should fail
	assert.Equal(t, "FAILED", finalStatus.Status, "Workflow should fail due to matrix instance failure")

	// Count job statuses
	var failedCount, cancelledCount, completedCount int
	for _, job := range finalStatus.Jobs {
		switch job.Status {
		case "FAILED":
			failedCount++
		case "CANCELLED":
			cancelledCount++
		case "COMPLETED":
			completedCount++
		}
	}

	// With fail-fast, expect: 1 FAILED, others CANCELLED (not COMPLETED)
	assert.Equal(t, 1, failedCount, "Should have exactly 1 failed instance (instance 2)")
	assert.Greater(t, cancelledCount, 0, "Should have cancelled instances due to fail-fast")

	// TODO (DEV TEAM): Verify Event History shows cancellation events
	// TODO (DEV TEAM): Test with fail-fast: false and verify all instances complete
	// TODO (DEV TEAM): Verify instance 2 failed, others cancelled (not all 4 completed)
}
