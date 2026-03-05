//go:build integration

package integration

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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

// TriggerResponse represents a webhook trigger API response
type TriggerResponse struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	WorkflowName  string                 `json:"workflow_name"`
	Type          string                 `json:"type"`
	WebhookURL    string                 `json:"webhook_url"`
	WebhookSecret string                 `json:"webhook_secret,omitempty"` // Only on Create
	Status        string                 `json:"status"`
	TotalTriggers int                    `json:"total_triggers"`
	CreatedAt     string                 `json:"created_at"`
	Filters       map[string]interface{} `json:"filters,omitempty"`
}

// TriggerListResponse represents a list of triggers
type TriggerListResponse struct {
	Triggers []TriggerResponse `json:"triggers"`
	Total    int               `json:"total"`
}

// WebhookFireResponse represents webhook trigger execution response
type WebhookFireResponse struct {
	TriggerID  string `json:"trigger_id"`
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

const testWorkflowForWebhook = `name: webhook-test-workflow
vars:
  env: dev
jobs:
  build:
    runs-on: waterflow-server
    steps:
      - name: Echo webhook received
        uses: run@v1
        with:
          command: echo "Webhook triggered for ${{ vars.env }}"
`

// computeHMACSignature computes the X-Hub-Signature-256 for a payload
func computeHMACSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// createTrigger creates a webhook trigger via API and returns the response
func createTrigger(ctx context.Context, t *testing.T, serverURL, workflowName string, req map[string]interface{}) *TriggerResponse {
	t.Helper()
	jsonBody, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/v1/workflows/%s/triggers", serverURL, workflowName),
		bytes.NewBuffer(jsonBody))
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		t.Fatalf("Failed to create trigger (status %d): %s", resp.StatusCode, body)
	}

	var result TriggerResponse
	require.NoError(t, json.Unmarshal(body, &result))
	return &result
}

// deleteTrigger deletes a webhook trigger via API
func deleteTrigger(ctx context.Context, t *testing.T, serverURL, workflowName, triggerID string) {
	t.Helper()
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/v1/workflows/%s/triggers/%s", serverURL, workflowName, triggerID), nil)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	resp.Body.Close()
}

// ====================================================================================
// Story 1.11: Webhook API - Integration Tests
// ====================================================================================

// TestStory1_11_INT_001_CreateWebhook verifies webhook creation via REST API
// Test ID: 1.11-INT-001
func TestStory1_11_INT_001_CreateWebhook(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.11-INT-001",
		Given:  "Webhook configuration with workflow reference",
		When:   "POST /v1/workflows/{name}/triggers is called",
		Then: []string{
			"Webhook trigger is created and persisted",
			"Unique webhook URL is generated",
			"webhook_secret is returned in Create response only",
			"Status is 201 Created",
		},
		AcceptanceCriteria: []string{"AC1: Register Webhook Trigger"},
	}).Log(t)

	if skipDatabaseTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	// Setup: create workflow definition
	createDefinition(ctx, t, serverURL, map[string]interface{}{"name": "webhook-test-workflow", "content": testWorkflowForWebhook})

	triggerName := fmt.Sprintf("push-deploy-%d", time.Now().UnixNano())
	trigResp := createTrigger(ctx, t, serverURL, "webhook-test-workflow", map[string]interface{}{
		"name":    triggerName,
		"type":    "webhook",
		"enabled": true,
		"secret":  "my-webhook-secret-key-1234",
		"filters": map[string]interface{}{
			"branches": []string{"main"},
		},
	})

	assert.NotEmpty(t, trigResp.ID, "trigger ID should be set")
	assert.NotEmpty(t, trigResp.WebhookURL, "webhook_url should be set")
	assert.NotEmpty(t, trigResp.WebhookSecret, "webhook_secret should be returned on Create")
	assert.Equal(t, "webhook-test-workflow", trigResp.WorkflowName)
	assert.Equal(t, "enabled", trigResp.Status)
	assert.Contains(t, trigResp.WebhookURL, trigResp.ID, "webhook_url must contain trigger id")

	t.Cleanup(func() {
		deleteTrigger(ctx, t, serverURL, "webhook-test-workflow", trigResp.ID)
	})
}

// TestStory1_11_INT_002_ListWebhooks verifies listing triggers via REST API
// Test ID: 1.11-INT-002
func TestStory1_11_INT_002_ListWebhooks(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.11-INT-002",
		Given:  "Multiple webhook triggers exist",
		When:   "GET /v1/workflows/{name}/triggers is called",
		Then: []string{
			"All triggers are returned",
			"Pagination works",
			"Secret is NOT returned in list",
		},
		AcceptanceCriteria: []string{"AC4: List Webhook Triggers"},
	}).Log(t)

	if skipDatabaseTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	createDefinition(ctx, t, serverURL, map[string]interface{}{"name": "webhook-test-workflow", "content": testWorkflowForWebhook})

	name1 := fmt.Sprintf("trigger-list-a-%d", time.Now().UnixNano())
	name2 := fmt.Sprintf("trigger-list-b-%d", time.Now().UnixNano())
	t1 := createTrigger(ctx, t, serverURL, "webhook-test-workflow", map[string]interface{}{
		"name": name1, "type": "webhook", "enabled": true,
	})
	t2 := createTrigger(ctx, t, serverURL, "webhook-test-workflow", map[string]interface{}{
		"name": name2, "type": "webhook", "enabled": true,
	})

	// List triggers
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		serverURL+"/v1/workflows/webhook-test-workflow/triggers?limit=50", nil)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var listResp TriggerListResponse
	body, _ := io.ReadAll(resp.Body)
	require.NoError(t, json.Unmarshal(body, &listResp))

	assert.GreaterOrEqual(t, listResp.Total, 2, "at least 2 triggers should be returned")

	// Verify secret NOT in list responses
	for _, tr := range listResp.Triggers {
		assert.Empty(t, tr.WebhookSecret, "secret must NOT be returned in list")
	}

	t.Cleanup(func() {
		deleteTrigger(ctx, t, serverURL, "webhook-test-workflow", t1.ID)
		deleteTrigger(ctx, t, serverURL, "webhook-test-workflow", t2.ID)
	})
}

// TestStory1_11_INT_003_DeleteWebhook verifies trigger deletion via REST API
// Test ID: 1.11-INT-003
func TestStory1_11_INT_003_DeleteWebhook(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.11-INT-003",
		Given:  "Existing webhook trigger",
		When:   "DELETE /v1/workflows/{name}/triggers/{id} is called",
		Then: []string{
			"Trigger is removed — 204 No Content",
			"Subsequent GET returns 404",
		},
		AcceptanceCriteria: []string{"AC8: Delete Webhook Trigger"},
	}).Log(t)

	if skipDatabaseTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	createDefinition(ctx, t, serverURL, map[string]interface{}{"name": "webhook-test-workflow", "content": testWorkflowForWebhook})

	trig := createTrigger(ctx, t, serverURL, "webhook-test-workflow", map[string]interface{}{
		"name": fmt.Sprintf("trigger-del-%d", time.Now().UnixNano()), "type": "webhook", "enabled": true,
	})

	// Delete
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/v1/workflows/webhook-test-workflow/triggers/%s", serverURL, trig.ID), nil)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Confirm 404 on Get
	getReq, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/v1/workflows/webhook-test-workflow/triggers/%s", serverURL, trig.ID), nil)
	getResp, err := client.Do(getReq)
	require.NoError(t, err)
	getResp.Body.Close()
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
}

// TestStory1_11_INT_004_WebhookTrigger verifies that POST to webhook URL triggers Temporal workflow
// Test ID: 1.11-INT-004
func TestStory1_11_INT_004_WebhookTrigger(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.11-INT-004",
		Given:  "Enabled webhook trigger with HMAC secret",
		When:   "POST /api/v1/webhooks/{id}/trigger with valid signature",
		Then: []string{
			"Webhook returns 200 OK with status=triggered",
			"workflow_id and run_id are returned",
			"Temporal workflow is started",
		},
		AcceptanceCriteria: []string{"AC2: Receive Webhook Request"},
	}).Log(t)

	if skipDatabaseTests(t) || skipTemporalTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	createDefinition(ctx, t, serverURL, map[string]interface{}{"name": "webhook-test-workflow", "content": testWorkflowForWebhook})

	const secret = "integration-test-secret-1234"
	trig := createTrigger(ctx, t, serverURL, "webhook-test-workflow", map[string]interface{}{
		"name": fmt.Sprintf("webhook-fire-%d", time.Now().UnixNano()), "type": "webhook",
		"enabled": true, "secret": secret,
	})
	require.NotEmpty(t, trig.ID)

	// Build push payload (GitHub-style)
	payload := []byte(`{
		"ref": "refs/heads/main",
		"commits": [{"id": "abc123", "message": "fix: deploy", "author": {"name": "dev"}}],
		"repository": {"name": "my-app"}
	}`)

	sig := computeHMACSignature(payload, secret)

	fireReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/api/v1/webhooks/%s/trigger", serverURL, trig.ID),
		bytes.NewBuffer(payload))
	fireReq.Header.Set("Content-Type", "application/json")
	fireReq.Header.Set("X-Hub-Signature-256", sig)
	fireReq.Header.Set("X-GitHub-Event", "push")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(fireReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Response: %s", string(body))

	var fireResp WebhookFireResponse
	require.NoError(t, json.Unmarshal(body, &fireResp))
	assert.Equal(t, "triggered", fireResp.Status)
	assert.NotEmpty(t, fireResp.WorkflowID, "workflow_id must be returned")
	assert.NotEmpty(t, fireResp.RunID, "run_id must be non-empty after CRIT-1 fix")
	assert.Contains(t, fireResp.WorkflowID, trig.ID, "workflow_id should contain trigger id")

	t.Cleanup(func() {
		deleteTrigger(ctx, t, serverURL, "webhook-test-workflow", trig.ID)
	})
}

// TestStory1_11_INT_005_WebhookPayloadVars verifies vars are accessible in workflow
// Test ID: 1.11-INT-005
func TestStory1_11_INT_005_WebhookInvalidSignature(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.11-INT-005",
		Given:  "Enabled webhook trigger with HMAC secret",
		When:   "POST to webhook URL with invalid signature",
		Then:   []string{"401 Unauthorized is returned"},
		AcceptanceCriteria: []string{"AC2: Signature Verification"},
	}).Log(t)

	if skipDatabaseTests(t) {
		return
	}

	ctx := context.Background()
	serverURL := getServerURL()

	createDefinition(ctx, t, serverURL, map[string]interface{}{"name": "webhook-test-workflow", "content": testWorkflowForWebhook})

	trig := createTrigger(ctx, t, serverURL, "webhook-test-workflow", map[string]interface{}{
		"name": fmt.Sprintf("webhook-sig-%d", time.Now().UnixNano()), "type": "webhook",
		"enabled": true, "secret": "correct-secret-key-1234",
	})

	payload := []byte(`{"ref": "refs/heads/main"}`)

	fireReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/api/v1/webhooks/%s/trigger", serverURL, trig.ID),
		bytes.NewBuffer(payload))
	fireReq.Header.Set("Content-Type", "application/json")
	fireReq.Header.Set("X-Hub-Signature-256", "sha256=invalid-signature")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(fireReq)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	t.Cleanup(func() {
		deleteTrigger(ctx, t, serverURL, "webhook-test-workflow", trig.ID)
	})
}

