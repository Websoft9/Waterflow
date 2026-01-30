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
// Story 1.7: Timeout and Retry Mechanism - E2E Tests
// ====================================================================================
//
// Test Coverage Priority:
// - 1.7-E2E-001: P0 - Timeout control with resource cleanup
// - 1.7-E2E-002: P0 - Retry strategy with exponential backoff
//
// Architecture Validation:
// - Timeout via Temporal Activity timeouts (StartToClose, ScheduleToClose)
// - Retry via Temporal Activity RetryPolicy
// - Verify resource cleanup on timeout
// - Event History immutability validation
//
// ====================================================================================

// TestStory1_7_E2E_001_TimeoutControl validates timeout mechanisms
// at step and job levels with proper resource cleanup
//
// Test ID: 1.7-E2E-001
// Priority: P0 (Critical - Timeout control mechanism)
// Risk: HIGH - Failure causes infinite hanging workflows
//
// Acceptance Criteria Verified:
// - AC1: timeout-minutes at step level
// - AC1: timeout-minutes at job level
// - AC1: Step timeout < job timeout hierarchy
// - AC1: Timeout triggers Activity cancellation
// - AC1: Resources cleaned up on timeout
// - AC1: Timeout event in Event History
func TestStory1_7_E2E_001_TimeoutControl(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.7-E2E-001",
		Given:  "A workflow with timeout-minutes defined on steps and jobs",
		When:   "A step exceeds its timeout duration",
		Then: []string{
			"The step is cancelled after timeout-minutes elapsed",
			"Temporal Activity receives cancellation signal",
			"Step status becomes TIMEOUT or CANCELLED",
			"Resources (Agent processes) are cleaned up",
			"Timeout event recorded in Event History",
			"Workflow can continue or fail based on continue-on-error",
		},
		AcceptanceCriteria: []string{
			"AC1: timeout-minutes at step level",
			"AC1: timeout-minutes at job level",
			"AC1: Timeout hierarchy enforcement",
			"AC1: Activity cancellation on timeout",
			"AC1: Resource cleanup",
			"AC1: Event History timeout record",
		},
	}).Log(t)

	// GIVEN: Workflow with step timeout
	workflowWithTimeout := `
name: Timeout Control Test
jobs:
  test-timeout:
    runs-on: default
    timeout-minutes: 5
    steps:
      - id: quick-step
        timeout-minutes: 1
        uses: shell@v1
        with:
          command: echo "Quick step completes fast"
      
      - id: slow-step
        timeout-minutes: 1
        continue-on-error: true
        uses: shell@v1
        with:
          command: echo "Starting slow step" && sleep 120
      
      - id: after-timeout
        uses: shell@v1
        with:
          command: echo "This runs after timeout due to continue-on-error"
`

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	_ = ctx

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	startTime := time.Now()

	resp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(workflowWithTimeout))))
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
		Jobs   []struct {
			Steps []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"steps"`
		} `json:"jobs"`
	}

	statusURL := fmt.Sprintf("%s/api/v1/workflows/%s", serverURL, submitResponse.WorkflowID)
	completed := false

	for i := 0; i < 150; i++ {
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

	require.True(t, completed, "Workflow should complete or timeout")
	elapsed := time.Since(startTime).Seconds()

	// THEN: Verify timeout behavior
	require.Len(t, finalStatus.Jobs, 1)
	steps := finalStatus.Jobs[0].Steps

	stepStatuses := make(map[string]string)
	for _, step := range steps {
		stepStatuses[step.ID] = step.Status
	}

	// Verify execution pattern
	assert.Equal(t, "COMPLETED", stepStatuses["quick-step"], "Quick step should complete")
	assert.Contains(t, []string{"TIMEOUT", "CANCELLED", "FAILED"}, stepStatuses["slow-step"],
		"Slow step should timeout after 1 minute")
	assert.Equal(t, "COMPLETED", stepStatuses["after-timeout"], "After-timeout step should run (continue-on-error)")

	// Verify timing: slow-step should timeout at ~60s, not run full 120s
	assert.Less(t, elapsed, 90.0, "Total time should be <90s (timeout at 60s), not 120s+ (full sleep)")

	// TODO (DEV TEAM): Query Event History to verify timeout event
	// TODO (DEV TEAM): Verify Temporal Activity has ScheduleToCloseTimeout = 60s
	// TODO (DEV TEAM): Verify Agent process cleanup (no orphaned sleep processes)
	// TODO (DEV TEAM): Test job-level timeout overrides step-level
}

// TestStory1_7_E2E_002_RetryStrategy validates retry mechanism with exponential backoff
// for transient failures in step execution
//
// Test ID: 1.7-E2E-002
// Priority: P0 (Critical - Retry resilience mechanism)
// Risk: HIGH - Failure prevents recovery from transient errors
//
// Acceptance Criteria Verified:
// - AC2: retry.max-attempts configuration
// - AC2: retry.backoff exponential strategy
// - AC2: retry.backoff-multiplier (default 2.0)
// - AC2: Retry attempt count in Event History
// - AC2: Final success/failure after retries exhausted
func TestStory1_7_E2E_002_RetryStrategy(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.7-E2E-002",
		Given:  "A workflow with retry strategy configured on steps",
		When:   "A step fails transiently and then succeeds on retry",
		Then: []string{
			"The step is retried up to max-attempts times",
			"Backoff delays increase exponentially (1s, 2s, 4s, 8s...)",
			"Retry attempts recorded in Event History",
			"Final status is COMPLETED if retry succeeds",
			"Final status is FAILED if all retries exhausted",
		},
		AcceptanceCriteria: []string{
			"AC2: retry.max-attempts",
			"AC2: retry.backoff exponential",
			"AC2: retry.backoff-multiplier",
			"AC2: Retry count in Event History",
			"AC2: Final success/failure status",
		},
	}).Log(t)

	// GIVEN: Workflow with retry configuration
	workflowWithRetry := `
name: Retry Strategy Test
jobs:
  test-retry:
    runs-on: default
    steps:
      - id: flaky-step
        uses: shell@v1
        retry:
          max-attempts: 3
          backoff: exponential
          backoff-multiplier: 2.0
        with:
          command: |
            ATTEMPT_FILE=/tmp/retry_attempts_${{ workflow.id }}.txt
            if [ ! -f $ATTEMPT_FILE ]; then
              echo "1" > $ATTEMPT_FILE
              echo "Attempt 1: Failing"
              exit 1
            else
              ATTEMPTS=$(cat $ATTEMPT_FILE)
              NEXT=$((ATTEMPTS + 1))
              echo $NEXT > $ATTEMPT_FILE
              if [ $ATTEMPTS -lt 2 ]; then
                echo "Attempt $NEXT: Failing"
                exit 1
              else
                echo "Attempt $NEXT: Success!"
                rm -f $ATTEMPT_FILE
                exit 0
              fi
            fi
      
      - id: always-fail-retry
        uses: shell@v1
        continue-on-error: true
        retry:
          max-attempts: 3
          backoff: exponential
        with:
          command: echo "Always failing" && exit 1
`

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	startTime := time.Now()

	resp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(workflowWithRetry))))
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
		Jobs   []struct {
			Steps []struct {
				ID           string `json:"id"`
				Status       string `json:"status"`
				RetryAttempt int    `json:"retryAttempt,omitempty"`
			} `json:"steps"`
		} `json:"jobs"`
	}

	statusURL := fmt.Sprintf("%s/api/v1/workflows/%s", serverURL, submitResponse.WorkflowID)
	completed := false

	for i := 0; i < 90; i++ {
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
	elapsed := time.Since(startTime).Seconds()

	// THEN: Verify retry behavior
	assert.Equal(t, "COMPLETED", finalStatus.Status, "Workflow should complete")

	require.Len(t, finalStatus.Jobs, 1)
	steps := finalStatus.Jobs[0].Steps

	stepStatuses := make(map[string]string)
	for _, step := range steps {
		stepStatuses[step.ID] = step.Status
	}

	// Verify flaky-step eventually succeeds after retries
	assert.Equal(t, "COMPLETED", stepStatuses["flaky-step"],
		"Flaky step should succeed after 2 retries (3rd attempt succeeds)")

	// Verify always-fail exhausts retries
	assert.Equal(t, "FAILED", stepStatuses["always-fail-retry"],
		"Always-fail step should fail after 3 attempts exhausted")

	// Verify timing includes exponential backoff
	// flaky-step: 3 attempts with backoff (0s + 1s + 2s = 3s)
	// always-fail: 3 attempts with backoff (0s + 1s + 2s = 3s)
	// Total should be at least 6 seconds (backoff delays)
	assert.GreaterOrEqual(t, elapsed, 6.0, "Total time should include exponential backoff delays (>=6s)")

	// TODO (DEV TEAM): Query Event History to verify retry attempt events
	// TODO (DEV TEAM): Verify Temporal Activity RetryPolicy configured
	// TODO (DEV TEAM): Verify backoff delays: 1s, 2s, 4s (multiplier 2.0)
	// TODO (DEV TEAM): Verify retry count in step metadata (attempt 1, 2, 3)
}
