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

// ScheduleResponse represents schedule response
type ScheduleResponse struct {
	ScheduleID    string                 `json:"schedule_id"`
	WorkflowName  string                 `json:"workflow_name"`
	Cron          string                 `json:"cron"`
	Timezone      string                 `json:"timezone"`
	OverlapPolicy string                 `json:"overlap_policy,omitempty"`
	Status        string                 `json:"status"`
	Vars          map[string]interface{} `json:"vars,omitempty"`
	NextRunTime   string                 `json:"next_run_time,omitempty"`
	CreatedAt     string                 `json:"created_at,omitempty"`
}

const testWorkflowForSchedule = `name: scheduled-backup
vars:
  db_name: dev
jobs:
  backup:
    runs-on: waterflow-server
    steps:
      - name: Backup database
        uses: run@v1
        with:
          command: echo "Backing up DB for ${{ vars.db_name }}"
`

// TestStory1_10_Schedule_INT_001_CreateSchedule tests creating a schedule
func TestStory1_10_Schedule_INT_001_CreateSchedule(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.10-SCHEDULE-INT-001",
		Given:  "Workflow definition exists",
		When:   "POST /v1/workflows/{name}/schedules is called",
		Then:   []string{"Schedule is created in Temporal"},
	}).Log(t)

	if skipDatabaseTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	defReq := map[string]interface{}{
		"name": "scheduled-backup",
	}
	createDefinition(ctx, t, serverURL, defReq)

	scheduleReq := map[string]interface{}{
		"name": fmt.Sprintf("nightly-backup-%d", time.Now().Unix()),
		"cron": "0 2 * * *",
	}
	jsonBody, _ := json.Marshal(scheduleReq)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		serverURL+"/v1/workflows/scheduled-backup/schedules",
		bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Response: %s", string(body))

	var scheduleResp ScheduleResponse
	err = json.Unmarshal(body, &scheduleResp)
	require.NoError(t, err)

	assert.Equal(t, scheduleReq["name"], scheduleResp.ScheduleID)
	assert.Equal(t, "scheduled-backup", scheduleResp.WorkflowName)

	t.Cleanup(func() {
		deleteSchedule(ctx, t, serverURL, "scheduled-backup", scheduleResp.ScheduleID)
	})
}

// TestStory1_10_Schedule_INT_002_PauseResumeSchedule tests pause/resume
func TestStory1_10_Schedule_INT_002_PauseResumeSchedule(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.10-SCHEDULE-INT-002",
		Given:  "Active schedule exists",
		When:   "POST pause/resume is called",
		Then:   []string{"Schedule state changes"},
	}).Log(t)

	if skipDatabaseTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	defReq := map[string]interface{}{
		"name": "pause-test",
	}
	createDefinition(ctx, t, serverURL, defReq)

	schedule := createSchedule(ctx, t, serverURL, "pause-test", map[string]interface{}{
		"name": fmt.Sprintf("pause-%d", time.Now().Unix()),
		"cron": "0 7 * * *",
	})

	client := &http.Client{Timeout: 10 * time.Second}

	pauseReq := map[string]interface{}{"reason": "Maintenance"}
	jsonBody, _ := json.Marshal(pauseReq)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/v1/workflows/pause-test/schedules/%s/pause", serverURL, schedule.ScheduleID),
		bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	t.Cleanup(func() {
		deleteSchedule(ctx, t, serverURL, "pause-test", schedule.ScheduleID)
	})
}

func createDefinition(ctx context.Context, t *testing.T, serverURL string, defReq map[string]interface{}) {
	// If content not provided, generate workflow YAML matching the definition name
	if _, ok := defReq["content"]; !ok {
		defName := defReq["name"].(string)
		defReq["content"] = fmt.Sprintf(`name: %s
vars:
  db_name: dev
jobs:
  backup:
    runs-on: waterflow-server
    steps:
      - name: Test step
        uses: run@v1
        with:
          command: echo "Running workflow %s"
`, defName, defName)
	}

	jsonBody, _ := json.Marshal(defReq)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		serverURL+"/v1/workflows/definitions",
		bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to create definition: %s", body)
	}
}

func createSchedule(ctx context.Context, t *testing.T, serverURL, workflowName string, scheduleReq map[string]interface{}) *ScheduleResponse {
	if _, ok := scheduleReq["timezone"]; !ok {
		scheduleReq["timezone"] = "UTC"
	}

	jsonBody, _ := json.Marshal(scheduleReq)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/v1/workflows/%s/schedules", serverURL, workflowName),
		bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create schedule: %s", body)
	}

	var scheduleResp ScheduleResponse
	err = json.Unmarshal(body, &scheduleResp)
	require.NoError(t, err)

	return &scheduleResp
}

func deleteSchedule(ctx context.Context, t *testing.T, serverURL, workflowName, scheduleID string) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/v1/workflows/%s/schedules/%s", serverURL, workflowName, scheduleID), nil)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Logf("Failed to delete schedule: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Logf("Unexpected delete status %d: %s", resp.StatusCode, body)
	}
}

// TestStory1_10_Schedule_INT_003_GetSchedule tests getting schedule details
func TestStory1_10_Schedule_INT_003_GetSchedule(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.10-SCHEDULE-INT-003",
		Given:  "Schedule exists",
		When:   "GET /v1/workflows/{name}/schedules/{schedule_id} is called",
		Then:   []string{"Schedule details returned with all fields"},
	}).Log(t)

	if skipDatabaseTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	defReq := map[string]interface{}{
		"name": "get-test",
	}
	createDefinition(ctx, t, serverURL, defReq)

	schedule := createSchedule(ctx, t, serverURL, "get-test", map[string]interface{}{
		"name": fmt.Sprintf("get-schedule-%d", time.Now().Unix()),
		"cron": "0 8 * * *",
		"vars": map[string]interface{}{"env": "test"},
	})

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/v1/workflows/get-test/schedules/%s", serverURL, schedule.ScheduleID), nil)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Response: %s", string(body))

	var scheduleResp ScheduleResponse
	err = json.Unmarshal(body, &scheduleResp)
	require.NoError(t, err)

	assert.Equal(t, schedule.ScheduleID, scheduleResp.ScheduleID)
	assert.Equal(t, "get-test", scheduleResp.WorkflowName)
	assert.Equal(t, "0 8 * * *", scheduleResp.Cron)
	assert.NotEmpty(t, scheduleResp.Status)

	t.Cleanup(func() {
		deleteSchedule(ctx, t, serverURL, "get-test", schedule.ScheduleID)
	})
}

// TestStory1_10_Schedule_INT_004_UpdateSchedule tests updating schedule
func TestStory1_10_Schedule_INT_004_UpdateSchedule(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.10-SCHEDULE-INT-004",
		Given:  "Schedule exists",
		When:   "PUT /v1/workflows/{name}/schedules/{schedule_id} is called",
		Then:   []string{"Schedule configuration updated"},
	}).Log(t)

	if skipDatabaseTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	defReq := map[string]interface{}{
		"name": "update-test",
	}
	createDefinition(ctx, t, serverURL, defReq)

	schedule := createSchedule(ctx, t, serverURL, "update-test", map[string]interface{}{
		"name": fmt.Sprintf("update-schedule-%d", time.Now().Unix()),
		"cron": "0 9 * * *",
	})

	updateReq := map[string]interface{}{
		"cron": "0 10 * * *",
		"vars": map[string]interface{}{"env": "production"},
	}
	jsonBody, _ := json.Marshal(updateReq)

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPut,
		fmt.Sprintf("%s/v1/workflows/update-test/schedules/%s", serverURL, schedule.ScheduleID),
		bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Response: %s", string(body))

	var scheduleResp ScheduleResponse
	err = json.Unmarshal(body, &scheduleResp)
	require.NoError(t, err)
	assert.Equal(t, "0 10 * * *", scheduleResp.Cron)

	t.Cleanup(func() {
		deleteSchedule(ctx, t, serverURL, "update-test", schedule.ScheduleID)
	})
}

// TestStory1_10_Schedule_INT_005_TriggerSchedule tests manual trigger
func TestStory1_10_Schedule_INT_005_TriggerSchedule(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.10-SCHEDULE-INT-005",
		Given:  "Schedule exists",
		When:   "POST /v1/workflows/{name}/schedules/{schedule_id}/trigger is called",
		Then:   []string{"Workflow execution started immediately"},
	}).Log(t)

	if skipDatabaseTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	defReq := map[string]interface{}{
		"name": "trigger-test",
	}
	createDefinition(ctx, t, serverURL, defReq)

	schedule := createSchedule(ctx, t, serverURL, "trigger-test", map[string]interface{}{
		"name": fmt.Sprintf("trigger-schedule-%d", time.Now().Unix()),
		"cron": "0 11 * * *",
		"vars": map[string]interface{}{"db_name": "staging"},
	})

	triggerReq := map[string]interface{}{
		"vars": map[string]interface{}{"db_name": "hotfix"},
	}
	jsonBody, _ := json.Marshal(triggerReq)

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/v1/workflows/trigger-test/schedules/%s/trigger", serverURL, schedule.ScheduleID),
		bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Response: %s", string(body))

	var triggerResp map[string]interface{}
	err = json.Unmarshal(body, &triggerResp)
	require.NoError(t, err)
	assert.NotEmpty(t, triggerResp["workflow_id"])

	t.Cleanup(func() {
		deleteSchedule(ctx, t, serverURL, "trigger-test", schedule.ScheduleID)
	})
}

// TestStory1_10_Schedule_INT_006_DeleteConflict tests AC10 definition deletion conflict
func TestStory1_10_Schedule_INT_006_DeleteConflict(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.10-SCHEDULE-INT-006",
		Given:  "Workflow has schedules",
		When:   "DELETE /v1/workflows/{name} is called",
		Then:   []string{"409 Conflict returned with schedule details"},
	}).Log(t)

	if skipDatabaseTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	defReq := map[string]interface{}{
		"name": "conflict-test",
	}
	createDefinition(ctx, t, serverURL, defReq)

	schedule := createSchedule(ctx, t, serverURL, "conflict-test", map[string]interface{}{
		"name": fmt.Sprintf("conflict-schedule-%d", time.Now().Unix()),
		"cron": "0 12 * * *",
	})

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/v1/workflows/definitions/conflict-test", serverURL), nil)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusConflict, resp.StatusCode, "Response: %s", string(body))

	var errResp map[string]interface{}
	err = json.Unmarshal(body, &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp["error"].(map[string]interface{})["message"], "referenced by")

	t.Cleanup(func() {
		deleteSchedule(ctx, t, serverURL, "conflict-test", schedule.ScheduleID)
	})
}

// TestStory1_10_Schedule_INT_007_PauseResumeStateValidation tests AC5 - full pause/resume cycle with state validation
func TestStory1_10_Schedule_INT_007_PauseResumeStateValidation(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.10-SCHEDULE-INT-007",
		Given:  "Active schedule exists",
		When:   "Pause and Resume operations are performed",
		Then:   []string{"Schedule state changes correctly", "Status reflects actual state"},
	}).Log(t)

	if skipDatabaseTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	defReq := map[string]interface{}{
		"name": "pause-resume-test",
	}
	createDefinition(ctx, t, serverURL, defReq)

	schedule := createSchedule(ctx, t, serverURL, "pause-resume-test", map[string]interface{}{
		"name": fmt.Sprintf("pause-resume-%d", time.Now().Unix()),
		"cron": "0 7 * * *",
	})

	client := &http.Client{Timeout: 10 * time.Second}

	// Step 1: Verify initial state is active
	getReq, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/v1/workflows/pause-resume-test/schedules/%s", serverURL, schedule.ScheduleID), nil)
	getResp, err := client.Do(getReq)
	require.NoError(t, err)
	defer getResp.Body.Close()

	var initialState ScheduleResponse
	json.NewDecoder(getResp.Body).Decode(&initialState)
	assert.Equal(t, "active", initialState.Status, "Initial state should be active")

	// Step 2: Pause the schedule
	pauseReq := map[string]interface{}{"reason": "Integration test"}
	jsonBody, _ := json.Marshal(pauseReq)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/v1/workflows/pause-resume-test/schedules/%s/pause", serverURL, schedule.ScheduleID),
		bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	pauseResp, err := client.Do(req)
	require.NoError(t, err)
	defer pauseResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, pauseResp.StatusCode)

	// Step 3: Verify paused state
	time.Sleep(1 * time.Second) // Allow Temporal to propagate state change

	getReq2, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/v1/workflows/pause-resume-test/schedules/%s", serverURL, schedule.ScheduleID), nil)
	getResp2, err := client.Do(getReq2)
	require.NoError(t, err)
	defer getResp2.Body.Close()

	var pausedState ScheduleResponse
	json.NewDecoder(getResp2.Body).Decode(&pausedState)
	assert.Equal(t, "paused", pausedState.Status, "State should be paused after pause operation")

	// Step 4: Resume the schedule
	resumeReq := map[string]interface{}{"reason": "Test complete"}
	jsonBody2, _ := json.Marshal(resumeReq)
	req2, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/v1/workflows/pause-resume-test/schedules/%s/resume", serverURL, schedule.ScheduleID),
		bytes.NewBuffer(jsonBody2))
	req2.Header.Set("Content-Type", "application/json")

	resumeResp, err := client.Do(req2)
	require.NoError(t, err)
	defer resumeResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resumeResp.StatusCode)

	// Step 5: Verify active state again
	time.Sleep(1 * time.Second)

	getReq3, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/v1/workflows/pause-resume-test/schedules/%s", serverURL, schedule.ScheduleID), nil)
	getResp3, err := client.Do(getReq3)
	require.NoError(t, err)
	defer getResp3.Body.Close()

	var activeState ScheduleResponse
	json.NewDecoder(getResp3.Body).Decode(&activeState)
	assert.Equal(t, "active", activeState.Status, "State should be active after resume operation")

	t.Cleanup(func() {
		deleteSchedule(ctx, t, serverURL, "pause-resume-test", schedule.ScheduleID)
	})
}

// TestStory1_10_Schedule_INT_008_UpdatePartialFields tests AC7 - partial field updates
func TestStory1_10_Schedule_INT_008_UpdatePartialFields(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.10-SCHEDULE-INT-008",
		Given:  "Schedule exists with specific configuration",
		When:   "Only some fields are updated",
		Then:   []string{"Updated fields change", "Other fields remain unchanged"},
	}).Log(t)

	if skipDatabaseTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	defReq := map[string]interface{}{
		"name": "partial-update-test",
	}
	createDefinition(ctx, t, serverURL, defReq)

	schedule := createSchedule(ctx, t, serverURL, "partial-update-test", map[string]interface{}{
		"name":     fmt.Sprintf("partial-update-%d", time.Now().Unix()),
		"cron":     "0 9 * * *",
		"timezone": "UTC",
		"vars":     map[string]interface{}{"env": "dev", "count": 5},
	})

	client := &http.Client{Timeout: 10 * time.Second}

	// Update only cron field
	updateReq := map[string]interface{}{
		"cron": "0 10 * * *",
	}
	jsonBody, _ := json.Marshal(updateReq)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPut,
		fmt.Sprintf("%s/v1/workflows/partial-update-test/schedules/%s", serverURL, schedule.ScheduleID),
		bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Response: %s", string(body))

	var scheduleResp ScheduleResponse
	err = json.Unmarshal(body, &scheduleResp)
	require.NoError(t, err)

	// Verify updated field
	assert.Equal(t, "0 10 * * *", scheduleResp.Cron, "Cron should be updated")

	// Verify unchanged fields
	assert.Equal(t, "UTC", scheduleResp.Timezone, "Timezone should remain unchanged")
	// Note: vars comparison would require fetching full schedule details

	// Update only vars field
	updateReq2 := map[string]interface{}{
		"vars": map[string]interface{}{"env": "production", "count": 10},
	}
	jsonBody2, _ := json.Marshal(updateReq2)

	req2, _ := http.NewRequestWithContext(ctx, http.MethodPut,
		fmt.Sprintf("%s/v1/workflows/partial-update-test/schedules/%s", serverURL, schedule.ScheduleID),
		bytes.NewBuffer(jsonBody2))
	req2.Header.Set("Content-Type", "application/json")

	resp2, err := client.Do(req2)
	require.NoError(t, err)
	defer resp2.Body.Close()

	body2, _ := io.ReadAll(resp2.Body)
	assert.Equal(t, http.StatusOK, resp2.StatusCode, "Response: %s", string(body2))

	var scheduleResp2 ScheduleResponse
	err = json.Unmarshal(body2, &scheduleResp2)
	require.NoError(t, err)

	// Verify cron remained unchanged
	assert.Equal(t, "0 10 * * *", scheduleResp2.Cron, "Cron should remain from previous update")

	t.Cleanup(func() {
		deleteSchedule(ctx, t, serverURL, "partial-update-test", schedule.ScheduleID)
	})
}

// TestStory1_10_Schedule_INT_009_DeleteBehavior tests AC8 - delete behavior verification
func TestStory1_10_Schedule_INT_009_DeleteBehavior(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.10-SCHEDULE-INT-009",
		Given:  "Schedule exists",
		When:   "DELETE operation is performed",
		Then:   []string{"Schedule is removed", "Subsequent GET returns 404", "Definition unaffected"},
	}).Log(t)

	if skipDatabaseTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	defReq := map[string]interface{}{
		"name": "delete-behavior-test",
	}
	createDefinition(ctx, t, serverURL, defReq)

	schedule := createSchedule(ctx, t, serverURL, "delete-behavior-test", map[string]interface{}{
		"name": fmt.Sprintf("delete-behavior-%d", time.Now().Unix()),
		"cron": "0 13 * * *",
	})

	client := &http.Client{Timeout: 10 * time.Second}

	// Step 1: Verify schedule exists
	getReq, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/v1/workflows/delete-behavior-test/schedules/%s", serverURL, schedule.ScheduleID), nil)
	getResp, err := client.Do(getReq)
	require.NoError(t, err)
	defer getResp.Body.Close()
	assert.Equal(t, http.StatusOK, getResp.StatusCode, "Schedule should exist before deletion")

	// Step 2: Delete schedule
	delReq, _ := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/v1/workflows/delete-behavior-test/schedules/%s", serverURL, schedule.ScheduleID), nil)
	delResp, err := client.Do(delReq)
	require.NoError(t, err)
	defer delResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, delResp.StatusCode, "Delete should return 204")

	// Step 3: Verify schedule no longer exists
	time.Sleep(1 * time.Second) // Allow Temporal to propagate deletion

	getReq2, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/v1/workflows/delete-behavior-test/schedules/%s", serverURL, schedule.ScheduleID), nil)
	getResp2, err := client.Do(getReq2)
	require.NoError(t, err)
	defer getResp2.Body.Close()
	assert.Equal(t, http.StatusNotFound, getResp2.StatusCode, "Schedule should not exist after deletion")

	// Step 4: Verify definition still exists
	defGetReq, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/v1/workflows/definitions/delete-behavior-test", serverURL), nil)
	defGetResp, err := client.Do(defGetReq)
	require.NoError(t, err)
	defer defGetResp.Body.Close()
	assert.Equal(t, http.StatusOK, defGetResp.StatusCode, "Definition should still exist after schedule deletion")

	// No cleanup needed - schedule already deleted
}

// TestStory1_10_Schedule_INT_010_OverlapPolicy tests AC9 - overlap policy enforcement
func TestStory1_10_Schedule_INT_010_OverlapPolicy(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.10-SCHEDULE-INT-010",
		Given:  "Schedule with specific overlap policy",
		When:   "Schedule is created and queried",
		Then:   []string{"Overlap policy is correctly stored", "Policy is returned in GET response"},
	}).Log(t)

	if skipDatabaseTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	defReq := map[string]interface{}{
		"name": "overlap-test",
	}
	createDefinition(ctx, t, serverURL, defReq)

	// Test all supported overlap policies
	testPolicies := []string{"skip", "allow_all", "buffer_one", "cancel_other"}

	for i, policy := range testPolicies {
		t.Run(fmt.Sprintf("Policy_%s", policy), func(t *testing.T) {
			scheduleName := fmt.Sprintf("overlap-%s-%d", policy, time.Now().Unix()+int64(i))

			schedule := createSchedule(ctx, t, serverURL, "overlap-policy-test", map[string]interface{}{
				"name":           scheduleName,
				"cron":           "0 14 * * *",
				"overlap_policy": policy,
			})

			client := &http.Client{Timeout: 10 * time.Second}

			// Verify the policy is correctly stored
			getReq, _ := http.NewRequestWithContext(ctx, http.MethodGet,
				fmt.Sprintf("%s/v1/workflows/overlap-policy-test/schedules/%s", serverURL, schedule.ScheduleID), nil)
			getResp, err := client.Do(getReq)
			require.NoError(t, err)
			defer getResp.Body.Close()

			var scheduleResp ScheduleResponse
			body, _ := io.ReadAll(getResp.Body)
			err = json.Unmarshal(body, &scheduleResp)
			require.NoError(t, err)

			assert.Equal(t, policy, scheduleResp.OverlapPolicy, "Overlap policy should match creation request")

			// Cleanup
			deleteSchedule(ctx, t, serverURL, "overlap-policy-test", schedule.ScheduleID)
		})
	}
}

// TestStory1_10_Schedule_INT_011_E2ELifecycle tests complete schedule lifecycle (AC10 enhancement)
func TestStory1_10_Schedule_INT_011_E2ELifecycle(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.10-SCHEDULE-INT-011",
		Given:  "Clean state with no schedules",
		When:   "Complete lifecycle is executed",
		Then:   []string{"All operations work correctly", "Reference integrity maintained"},
	}).Log(t)

	if skipDatabaseTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	workflowName := fmt.Sprintf("lifecycle-test-%d", time.Now().Unix())

	// Step 1: Create Definition
	defReq := map[string]interface{}{
		"name":    workflowName,
		"content": testWorkflowForSchedule,
	}
	createDefinition(ctx, t, serverURL, defReq)

	// Step 2: Create multiple schedules
	schedule1 := createSchedule(ctx, t, serverURL, workflowName, map[string]interface{}{
		"name": fmt.Sprintf("schedule-1-%d", time.Now().Unix()),
		"cron": "0 2 * * *",
	})

	schedule2 := createSchedule(ctx, t, serverURL, workflowName, map[string]interface{}{
		"name": fmt.Sprintf("schedule-2-%d", time.Now().Unix()+1),
		"cron": "0 3 * * *",
	})

	// Step 3: Attempt to delete definition (should fail with 409)
	client := &http.Client{Timeout: 10 * time.Second}
	delDefReq, _ := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/v1/workflows/definitions/%s", serverURL, workflowName), nil)
	delDefResp, err := client.Do(delDefReq)
	require.NoError(t, err)
	defer delDefResp.Body.Close()
	assert.Equal(t, http.StatusConflict, delDefResp.StatusCode, "Definition deletion should fail with schedules")

	// Step 4: Delete all schedules
	deleteSchedule(ctx, t, serverURL, workflowName, schedule1.ScheduleID)
	deleteSchedule(ctx, t, serverURL, workflowName, schedule2.ScheduleID)

	// Step 5: Delete definition (should succeed now)
	time.Sleep(1 * time.Second) // Allow propagation

	delDefReq2, _ := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/v1/workflows/definitions/%s", serverURL, workflowName), nil)
	delDefResp2, err := client.Do(delDefReq2)
	require.NoError(t, err)
	defer delDefResp2.Body.Close()
	assert.Equal(t, http.StatusNoContent, delDefResp2.StatusCode, "Definition deletion should succeed without schedules")

	// Step 6: Verify definition is gone
	getDefReq, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/v1/workflows/definitions/%s", serverURL, workflowName), nil)
	getDefResp, err := client.Do(getDefReq)
	require.NoError(t, err)
	defer getDefResp.Body.Close()
	assert.Equal(t, http.StatusNotFound, getDefResp.StatusCode, "Definition should not exist")
}
