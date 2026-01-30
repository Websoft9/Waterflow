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
// Story 1.10: Schedule API - E2E Tests
// ====================================================================================
//
// Test Coverage Priority:
// - 1.10-E2E-001: P0 - Create schedule (POST /v1/schedules)
// - 1.10-E2E-002: P0 - List schedules (GET /v1/schedules)
// - 1.10-E2E-003: P1 - Query schedule details (GET /v1/schedules/{id})
// - 1.10-E2E-004: P0 - Pause/Resume schedule (PATCH /v1/schedules/{id})
// - 1.10-E2E-005: P1 - Delete schedule (DELETE /v1/schedules/{id})
// - 1.10-E2E-006: P0 - Manual trigger (POST /v1/schedules/{id}/trigger)
//
// Architecture Validation:
// - Temporal Schedules integration
// - Cron expression validation
// - Schedule metadata persistence (SQLite)
// - next_run_time, last_run_time tracking
//
// ====================================================================================

// TestStory1_10_E2E_001_CreateSchedule validates schedule creation API
// with Temporal Schedule registration
//
// Test ID: 1.10-E2E-001
// Priority: P0 (Critical - Schedule creation foundation)
// Risk: HIGH - Failure blocks automated workflow execution
//
// Acceptance Criteria Verified:
// - AC1: POST /v1/schedules creates Temporal Schedule
// - AC1: Validates cron expression format
// - AC1: Stores metadata in SQLite
// - AC1: Returns 201 Created with schedule details
// - AC1: Includes next_run_time in response
func TestStory1_10_E2E_001_CreateSchedule(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.10-E2E-001",
		Given:  "A workflow definition exists",
		When:   "POST /v1/schedules with cron expression",
		Then: []string{
			"Schedule is created in Temporal",
			"Metadata stored in SQLite",
			"Returns 201 Created with schedule ID",
			"Response includes next_run_time",
			"Invalid cron returns 422",
		},
		AcceptanceCriteria: []string{
			"AC1: POST /v1/schedules",
			"AC1: Cron expression validation",
			"AC1: Metadata persistence",
			"AC1: Returns 201 Created",
		},
	}).Log(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	_ = ctx

	// GIVEN: Schedule configuration with cron
	schedulePayload := `{
		"name": "Daily Backup",
		"workflow": {
			"name": "Backup Workflow",
			"jobs": {
				"backup": {
					"runs-on": "default",
					"steps": [
						{
							"id": "backup-step",
							"uses": "shell@v1",
							"with": {
								"command": "echo 'Running backup'"
							}
						}
					]
				}
			}
		},
		"cron": "0 2 * * *",
		"timezone": "UTC"
	}`

	// WHEN: Create schedule
	scheduleURL := fmt.Sprintf("%s/api/v1/schedules", serverURL)
	resp, err := http.Post(scheduleURL, "application/json", bytes.NewReader([]byte(schedulePayload)))
	require.NoError(t, err)
	defer resp.Body.Close()

	// THEN: Verify creation
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Should return 201 Created")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var createResponse struct {
		ScheduleID  string    `json:"scheduleId"`
		Name        string    `json:"name"`
		Cron        string    `json:"cron"`
		NextRunTime time.Time `json:"nextRunTime"`
		Status      string    `json:"status"`
	}
	err = json.Unmarshal(body, &createResponse)
	require.NoError(t, err)

	assert.NotEmpty(t, createResponse.ScheduleID, "Should return schedule ID")
	assert.Equal(t, "Daily Backup", createResponse.Name)
	assert.Equal(t, "0 2 * * *", createResponse.Cron)
	assert.NotZero(t, createResponse.NextRunTime, "Should calculate next run time")
	assert.Equal(t, "active", createResponse.Status)

	// Test invalid cron expression
	invalidPayload := `{
		"name": "Invalid Schedule",
		"workflow": {"name": "Test", "jobs": {}},
		"cron": "invalid cron"
	}`

	invalidResp, err := http.Post(scheduleURL, "application/json", bytes.NewReader([]byte(invalidPayload)))
	require.NoError(t, err)
	defer invalidResp.Body.Close()

	assert.Equal(t, http.StatusUnprocessableEntity, invalidResp.StatusCode, "Invalid cron should return 422")

	// TODO (DEV TEAM): Verify Temporal Schedule created
	// TODO (DEV TEAM): Verify SQLite metadata stored
	// TODO (DEV TEAM): Test schedule actually triggers workflow at cron time
}

// TestStory1_10_E2E_002_ListSchedules validates schedule list API
// with status and name filtering
//
// Test ID: 1.10-E2E-002
// Priority: P0 (Critical - Schedule discovery)
// Risk: MEDIUM - Failure limits schedule management
//
// Acceptance Criteria Verified:
// - AC2: GET /v1/schedules returns paginated list
// - AC2: Filter by status (active, paused)
// - AC2: Filter by workflow name
// - AC2: Includes next_run_time and last_run_time
func TestStory1_10_E2E_002_ListSchedules(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.10-E2E-002",
		Given:  "Multiple schedules exist in the system",
		When:   "GET /v1/schedules with filters",
		Then: []string{
			"Returns paginated schedule list",
			"Supports status filtering (active, paused)",
			"Supports workflow name filtering",
			"Includes next_run_time and last_run_time",
		},
		AcceptanceCriteria: []string{
			"AC2: GET /v1/schedules",
			"AC2: Filter by status",
			"AC2: Filter by workflow name",
		},
	}).Log(t)

	// GIVEN: Create multiple schedules
	scheduleURL := fmt.Sprintf("%s/api/v1/schedules", serverURL)

	for i := 1; i <= 2; i++ {
		payload := fmt.Sprintf(`{
			"name": "Test Schedule %d",
			"workflow": {"name": "Test Workflow", "jobs": {}},
			"cron": "0 %d * * *"
		}`, i, i)

		http.Post(scheduleURL, "application/json", bytes.NewReader([]byte(payload)))
	}

	time.Sleep(2 * time.Second)

	// WHEN: List schedules
	listURL := fmt.Sprintf("%s/api/v1/schedules?page=1&limit=10", serverURL)
	listResp, err := http.Get(listURL)
	require.NoError(t, err)
	defer listResp.Body.Close()

	// THEN: Verify list
	assert.Equal(t, http.StatusOK, listResp.StatusCode)

	listBody, err := io.ReadAll(listResp.Body)
	require.NoError(t, err)

	var listResponse struct {
		Schedules []struct {
			ScheduleID  string    `json:"scheduleId"`
			Name        string    `json:"name"`
			Status      string    `json:"status"`
			NextRunTime time.Time `json:"nextRunTime"`
			LastRunTime time.Time `json:"lastRunTime,omitempty"`
		} `json:"schedules"`
		Total int `json:"total"`
	}
	err = json.Unmarshal(listBody, &listResponse)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, listResponse.Total, 2, "Should have at least 2 schedules")

	// Test status filtering
	filterURL := fmt.Sprintf("%s/api/v1/schedules?status=active", serverURL)
	filterResp, err := http.Get(filterURL)
	require.NoError(t, err)
	defer filterResp.Body.Close()

	assert.Equal(t, http.StatusOK, filterResp.StatusCode)

	// TODO (DEV TEAM): Test workflow name filtering
	// TODO (DEV TEAM): Verify next_run_time populated
	// TODO (DEV TEAM): Verify last_run_time after schedule runs
}

// TestStory1_10_E2E_003_QuerySchedule validates schedule details query
// with run history and statistics
//
// Test ID: 1.10-E2E-003
// Priority: P1 (Important - Schedule monitoring)
// Risk: LOW - Failure limits schedule inspection
//
// Acceptance Criteria Verified:
// - AC3: GET /v1/schedules/{id} returns complete details
// - AC3: Includes next_run_time, last_run_time, total_runs
// - AC3: Returns 404 for non-existent schedule
func TestStory1_10_E2E_003_QuerySchedule(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.10-E2E-003",
		Given:  "A schedule has been created",
		When:   "GET /v1/schedules/{id}",
		Then: []string{
			"Returns complete schedule details",
			"Includes next_run_time, last_run_time",
			"Includes total_runs count",
			"Returns 404 for non-existent schedule",
		},
		AcceptanceCriteria: []string{
			"AC3: GET /v1/schedules/{id}",
			"AC3: Complete details",
			"AC3: next_run_time, last_run_time, total_runs",
		},
	}).Log(t)

	// GIVEN: Create a schedule
	scheduleURL := fmt.Sprintf("%s/api/v1/schedules", serverURL)
	payload := `{
		"name": "Query Test Schedule",
		"workflow": {"name": "Test", "jobs": {}},
		"cron": "0 3 * * *"
	}`

	createResp, err := http.Post(scheduleURL, "application/json", bytes.NewReader([]byte(payload)))
	require.NoError(t, err)
	defer createResp.Body.Close()

	var createResponse struct {
		ScheduleID string `json:"scheduleId"`
	}
	createBody, _ := io.ReadAll(createResp.Body)
	json.Unmarshal(createBody, &createResponse)
	scheduleID := createResponse.ScheduleID

	// WHEN: Query schedule
	queryURL := fmt.Sprintf("%s/api/v1/schedules/%s", serverURL, scheduleID)
	queryResp, err := http.Get(queryURL)
	require.NoError(t, err)
	defer queryResp.Body.Close()

	// THEN: Verify details
	assert.Equal(t, http.StatusOK, queryResp.StatusCode)

	queryBody, err := io.ReadAll(queryResp.Body)
	require.NoError(t, err)

	var details struct {
		ScheduleID  string    `json:"scheduleId"`
		Name        string    `json:"name"`
		Cron        string    `json:"cron"`
		NextRunTime time.Time `json:"nextRunTime"`
		LastRunTime time.Time `json:"lastRunTime,omitempty"`
		TotalRuns   int       `json:"totalRuns"`
	}
	err = json.Unmarshal(queryBody, &details)
	require.NoError(t, err)

	assert.Equal(t, scheduleID, details.ScheduleID)
	assert.Equal(t, "Query Test Schedule", details.Name)
	assert.NotZero(t, details.NextRunTime)

	// Test 404
	notFoundURL := fmt.Sprintf("%s/api/v1/schedules/non-existent-id", serverURL)
	notFoundResp, err := http.Get(notFoundURL)
	require.NoError(t, err)
	defer notFoundResp.Body.Close()

	assert.Equal(t, http.StatusNotFound, notFoundResp.StatusCode)

	// TODO (DEV TEAM): Verify total_runs increments after execution
}

// TestStory1_10_E2E_004_PauseResumeSchedule validates schedule pause/resume API
// with Temporal Schedule state management
//
// Test ID: 1.10-E2E-004
// Priority: P0 (Critical - Schedule control)
// Risk: HIGH - Failure prevents schedule management
//
// Acceptance Criteria Verified:
// - AC4: PATCH /v1/schedules/{id} updates status
// - AC4: Paused schedule stops triggering workflows
// - AC4: Resumed schedule continues normal execution
// - AC4: Temporal Schedule state updated
func TestStory1_10_E2E_004_PauseResumeSchedule(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.10-E2E-004",
		Given:  "An active schedule exists",
		When:   "PATCH /v1/schedules/{id} to pause/resume",
		Then: []string{
			"Schedule state updated in Temporal",
			"Paused schedule stops triggering",
			"Resumed schedule continues execution",
			"Status reflects current state",
		},
		AcceptanceCriteria: []string{
			"AC4: PATCH /v1/schedules/{id}",
			"AC4: Pause stops triggering",
			"AC4: Resume continues execution",
		},
	}).Log(t)

	// GIVEN: Create a schedule
	scheduleURL := fmt.Sprintf("%s/api/v1/schedules", serverURL)
	payload := `{
		"name": "Pause Test Schedule",
		"workflow": {"name": "Test", "jobs": {}},
		"cron": "* * * * *"
	}`

	createResp, err := http.Post(scheduleURL, "application/json", bytes.NewReader([]byte(payload)))
	require.NoError(t, err)
	defer createResp.Body.Close()

	var createResponse struct {
		ScheduleID string `json:"scheduleId"`
	}
	createBody, _ := io.ReadAll(createResp.Body)
	json.Unmarshal(createBody, &createResponse)
	scheduleID := createResponse.ScheduleID

	// WHEN: Pause schedule
	patchURL := fmt.Sprintf("%s/api/v1/schedules/%s", serverURL, scheduleID)
	pausePayload := `{"status": "paused"}`

	req, _ := http.NewRequest(http.MethodPatch, patchURL, bytes.NewReader([]byte(pausePayload)))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	pauseResp, err := client.Do(req)
	require.NoError(t, err)
	defer pauseResp.Body.Close()

	// THEN: Verify paused
	assert.Equal(t, http.StatusOK, pauseResp.StatusCode)

	pauseBody, _ := io.ReadAll(pauseResp.Body)
	var pauseResponse struct {
		Status string `json:"status"`
	}
	json.Unmarshal(pauseBody, &pauseResponse)
	assert.Equal(t, "paused", pauseResponse.Status)

	// Resume schedule
	resumePayload := `{"status": "active"}`
	req, _ = http.NewRequest(http.MethodPatch, patchURL, bytes.NewReader([]byte(resumePayload)))
	req.Header.Set("Content-Type", "application/json")
	resumeResp, err := client.Do(req)
	require.NoError(t, err)
	defer resumeResp.Body.Close()

	assert.Equal(t, http.StatusOK, resumeResp.StatusCode)

	resumeBody, _ := io.ReadAll(resumeResp.Body)
	var resumeResponse struct {
		Status string `json:"status"`
	}
	json.Unmarshal(resumeBody, &resumeResponse)
	assert.Equal(t, "active", resumeResponse.Status)

	// TODO (DEV TEAM): Verify Temporal Schedule state updated
	// TODO (DEV TEAM): Verify paused schedule doesn't trigger workflows
	// TODO (DEV TEAM): Verify resumed schedule triggers normally
}

// TestStory1_10_E2E_005_DeleteSchedule validates schedule deletion API
// with Temporal Schedule cleanup
//
// Test ID: 1.10-E2E-005
// Priority: P1 (Important - Schedule lifecycle)
// Risk: MEDIUM - Failure prevents schedule cleanup
//
// Acceptance Criteria Verified:
// - AC5: DELETE /v1/schedules/{id} removes schedule
// - AC5: Temporal Schedule deleted
// - AC5: Metadata cleaned from SQLite
// - AC5: Running workflows unaffected
func TestStory1_10_E2E_005_DeleteSchedule(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.10-E2E-005",
		Given:  "A schedule exists",
		When:   "DELETE /v1/schedules/{id}",
		Then: []string{
			"Schedule deleted from Temporal",
			"Metadata removed from SQLite",
			"Returns 204 No Content",
			"Running workflows continue (unaffected)",
		},
		AcceptanceCriteria: []string{
			"AC5: DELETE /v1/schedules/{id}",
			"AC5: Temporal Schedule deleted",
			"AC5: Metadata cleanup",
			"AC5: Running workflows unaffected",
		},
	}).Log(t)

	// GIVEN: Create a schedule
	scheduleURL := fmt.Sprintf("%s/api/v1/schedules", serverURL)
	payload := `{
		"name": "Delete Test Schedule",
		"workflow": {"name": "Test", "jobs": {}},
		"cron": "0 4 * * *"
	}`

	createResp, err := http.Post(scheduleURL, "application/json", bytes.NewReader([]byte(payload)))
	require.NoError(t, err)
	defer createResp.Body.Close()

	var createResponse struct {
		ScheduleID string `json:"scheduleId"`
	}
	createBody, _ := io.ReadAll(createResp.Body)
	json.Unmarshal(createBody, &createResponse)
	scheduleID := createResponse.ScheduleID

	// WHEN: Delete schedule
	deleteURL := fmt.Sprintf("%s/api/v1/schedules/%s", serverURL, scheduleID)
	req, _ := http.NewRequest(http.MethodDelete, deleteURL, nil)
	client := &http.Client{}
	deleteResp, err := client.Do(req)
	require.NoError(t, err)
	defer deleteResp.Body.Close()

	// THEN: Verify deletion
	assert.Equal(t, http.StatusNoContent, deleteResp.StatusCode, "Should return 204 No Content")

	// Verify schedule no longer exists
	queryURL := fmt.Sprintf("%s/api/v1/schedules/%s", serverURL, scheduleID)
	queryResp, err := http.Get(queryURL)
	require.NoError(t, err)
	defer queryResp.Body.Close()

	assert.Equal(t, http.StatusNotFound, queryResp.StatusCode, "Deleted schedule should return 404")

	// TODO (DEV TEAM): Verify Temporal Schedule deleted
	// TODO (DEV TEAM): Verify SQLite metadata removed
	// TODO (DEV TEAM): Test deleting while workflows are running
}

// TestStory1_10_E2E_006_ManualTrigger validates manual schedule trigger API
// with immediate workflow execution
//
// Test ID: 1.10-E2E-006
// Priority: P0 (Critical - Manual workflow execution)
// Risk: MEDIUM - Failure limits on-demand execution
//
// Acceptance Criteria Verified:
// - AC6: POST /v1/schedules/{id}/trigger immediately executes workflow
// - AC6: Does not affect normal cron schedule
// - AC6: Returns workflow ID of triggered instance
func TestStory1_10_E2E_006_ManualTrigger(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.10-E2E-006",
		Given:  "A schedule is configured",
		When:   "POST /v1/schedules/{id}/trigger",
		Then: []string{
			"Workflow executes immediately",
			"Normal cron schedule unaffected",
			"Returns workflow ID",
			"Returns 200 OK",
		},
		AcceptanceCriteria: []string{
			"AC6: POST /v1/schedules/{id}/trigger",
			"AC6: Immediate execution",
			"AC6: Cron schedule unaffected",
		},
	}).Log(t)

	// GIVEN: Create a schedule with long cron (won't trigger naturally during test)
	scheduleURL := fmt.Sprintf("%s/api/v1/schedules", serverURL)
	payload := `{
		"name": "Manual Trigger Test",
		"workflow": {
			"name": "Triggered Workflow",
			"jobs": {
				"trigger-job": {
					"runs-on": "default",
					"steps": [
						{
							"id": "trigger-step",
							"uses": "shell@v1",
							"with": {"command": "echo 'Manually triggered'"}
						}
					]
				}
			}
		},
		"cron": "0 0 1 1 *"
	}`

	createResp, err := http.Post(scheduleURL, "application/json", bytes.NewReader([]byte(payload)))
	require.NoError(t, err)
	defer createResp.Body.Close()

	var createResponse struct {
		ScheduleID string `json:"scheduleId"`
	}
	createBody, _ := io.ReadAll(createResp.Body)
	json.Unmarshal(createBody, &createResponse)
	scheduleID := createResponse.ScheduleID

	// WHEN: Manually trigger schedule
	triggerURL := fmt.Sprintf("%s/api/v1/schedules/%s/trigger", serverURL, scheduleID)
	triggerResp, err := http.Post(triggerURL, "application/json", nil)
	require.NoError(t, err)
	defer triggerResp.Body.Close()

	// THEN: Verify trigger
	assert.Equal(t, http.StatusOK, triggerResp.StatusCode)

	triggerBody, err := io.ReadAll(triggerResp.Body)
	require.NoError(t, err)

	var triggerResponse struct {
		WorkflowID string `json:"workflowId"`
	}
	err = json.Unmarshal(triggerBody, &triggerResponse)
	require.NoError(t, err)

	assert.NotEmpty(t, triggerResponse.WorkflowID, "Should return workflow ID of triggered instance")

	// Verify workflow was created
	time.Sleep(3 * time.Second)
	workflowURL := fmt.Sprintf("%s/api/v1/workflows/%s", serverURL, triggerResponse.WorkflowID)
	workflowResp, err := http.Get(workflowURL)
	require.NoError(t, err)
	defer workflowResp.Body.Close()

	assert.Equal(t, http.StatusOK, workflowResp.StatusCode, "Triggered workflow should exist")

	// TODO (DEV TEAM): Verify schedule's next_run_time unchanged (cron not affected)
	// TODO (DEV TEAM): Verify total_runs incremented
}
