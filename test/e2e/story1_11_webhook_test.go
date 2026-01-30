//go:build e2e

package e2e

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

// ====================================================================================
// Story 1.11: Webhook Trigger - E2E Tests
// ====================================================================================
//
// Test Coverage Priority:
// - 1.11-E2E-001: P0 - Register webhook trigger (POST /v1/triggers)
// - 1.11-E2E-002: P0 - Receive webhook and trigger workflow
//
// Architecture Validation:
// - Webhook URL generation and routing
// - HMAC signature verification (X-Hub-Signature-256)
// - Async workflow execution (<100ms response)
// - Filter rules application (branch, path, event type)
//
// ====================================================================================

// TestStory1_11_E2E_001_RegisterWebhookTrigger validates webhook trigger registration API
// with unique URL generation and secret management
//
// Test ID: 1.11-E2E-001
// Priority: P0 (Critical - Webhook setup foundation)
// Risk: HIGH - Failure blocks webhook-based automation
//
// Acceptance Criteria Verified:
// - AC1: POST /v1/triggers creates webhook trigger
// - AC1: Generates unique webhook URL
// - AC1: Stores webhook secret for signature verification
// - AC1: Returns 201 Created with trigger details
// - AC1: Supports filter configuration (branch, path, etc.)
func TestStory1_11_E2E_001_RegisterWebhookTrigger(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.11-E2E-001",
		Given:  "A workflow definition exists",
		When:   "POST /v1/triggers to register webhook",
		Then: []string{
			"Webhook trigger is created",
			"Returns unique webhook URL",
			"Returns webhook secret for signature generation",
			"Returns 201 Created",
			"Filter rules are validated and stored",
		},
		AcceptanceCriteria: []string{
			"AC1: POST /v1/triggers",
			"AC1: Unique webhook URL",
			"AC1: Webhook secret storage",
			"AC1: Returns 201 Created",
		},
	}).Log(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	_ = ctx

	// GIVEN: Webhook trigger configuration
	triggerPayload := `{
		"type": "webhook",
		"name": "GitHub Push Trigger",
		"workflow": {
			"name": "CI Pipeline",
			"jobs": {
				"build": {
					"runs-on": "default",
					"steps": [
						{
							"id": "build-step",
							"uses": "shell@v1",
							"with": {
								"command": "echo 'Building from webhook'"
							}
						}
					]
				}
			}
		},
		"filters": {
			"branches": ["main", "develop"],
			"events": ["push", "pull_request"]
		}
	}`

	// WHEN: Register webhook trigger
	triggersURL := fmt.Sprintf("%s/api/v1/triggers", serverURL)
	resp, err := http.Post(triggersURL, "application/json", bytes.NewReader([]byte(triggerPayload)))
	require.NoError(t, err)
	defer resp.Body.Close()

	// THEN: Verify registration
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Should return 201 Created")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var createResponse struct {
		TriggerID  string `json:"triggerId"`
		Name       string `json:"name"`
		Type       string `json:"type"`
		WebhookURL string `json:"webhookUrl"`
		Secret     string `json:"secret"`
		Status     string `json:"status"`
	}
	err = json.Unmarshal(body, &createResponse)
	require.NoError(t, err)

	assert.NotEmpty(t, createResponse.TriggerID, "Should return trigger ID")
	assert.Equal(t, "webhook", createResponse.Type)
	assert.Equal(t, "GitHub Push Trigger", createResponse.Name)
	assert.NotEmpty(t, createResponse.WebhookURL, "Should return unique webhook URL")
	assert.NotEmpty(t, createResponse.Secret, "Should return webhook secret")
	assert.Equal(t, "enabled", createResponse.Status)

	// Verify webhook URL format
	assert.Contains(t, createResponse.WebhookURL, "/webhooks/", "Webhook URL should contain /webhooks/ path")
	assert.Contains(t, createResponse.WebhookURL, createResponse.TriggerID, "Webhook URL should contain trigger ID")

	// TODO (DEV TEAM): Verify filter rules stored correctly
	// TODO (DEV TEAM): Test invalid workflow definition returns 422
	// TODO (DEV TEAM): Verify secret is securely stored (hashed)
}

// TestStory1_11_E2E_002_ReceiveWebhookAndTrigger validates webhook reception API
// with HMAC signature verification and workflow triggering
//
// Test ID: 1.11-E2E-002
// Priority: P0 (Critical - Webhook execution)
// Risk: HIGH - Failure breaks webhook-based workflows
//
// Acceptance Criteria Verified:
// - AC2: POST to webhook URL triggers workflow
// - AC2: HMAC signature verification (X-Hub-Signature-256)
// - AC2: Filter rules applied (branch, path, event)
// - AC2: Async workflow execution
// - AC2: Response time <100ms
// - AC2: Invalid signature returns 401
func TestStory1_11_E2E_002_ReceiveWebhookAndTrigger(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.11-E2E-002",
		Given:  "A webhook trigger is registered",
		When:   "External system sends POST request to webhook URL",
		Then: []string{
			"HMAC signature is verified",
			"Filter rules are applied",
			"Workflow is triggered asynchronously",
			"Response returned in <100ms",
			"Invalid signature returns 401 Unauthorized",
			"Filtered events do not trigger workflow",
		},
		AcceptanceCriteria: []string{
			"AC2: Webhook reception",
			"AC2: HMAC signature verification",
			"AC2: Filter rules application",
			"AC2: Async workflow execution",
			"AC2: Response time <100ms",
		},
	}).Log(t)

	// GIVEN: Register webhook trigger
	triggersURL := fmt.Sprintf("%s/api/v1/triggers", serverURL)
	triggerPayload := `{
		"type": "webhook",
		"name": "Webhook Test Trigger",
		"workflow": {
			"name": "Webhook Workflow",
			"jobs": {
				"webhook-job": {
					"runs-on": "default",
					"steps": [
						{
							"id": "webhook-step",
							"uses": "shell@v1",
							"with": {
								"command": "echo 'Webhook triggered!'"
							}
						}
					]
				}
			}
		},
		"filters": {
			"branches": ["main"],
			"events": ["push"]
		}
	}`

	registerResp, err := http.Post(triggersURL, "application/json", bytes.NewReader([]byte(triggerPayload)))
	require.NoError(t, err)
	defer registerResp.Body.Close()

	var registerResponse struct {
		TriggerID  string `json:"triggerId"`
		WebhookURL string `json:"webhookUrl"`
		Secret     string `json:"secret"`
	}
	registerBody, _ := io.ReadAll(registerResp.Body)
	json.Unmarshal(registerBody, &registerResponse)

	webhookURL := registerResponse.WebhookURL
	secret := registerResponse.Secret

	// WHEN: Send webhook request with valid signature
	webhookPayload := `{
		"event": "push",
		"ref": "refs/heads/main",
		"repository": {
			"name": "test-repo"
		},
		"commits": [
			{
				"id": "abc123",
				"message": "Test commit"
			}
		]
	}`

	// Calculate HMAC signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(webhookPayload))
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	startTime := time.Now()

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader([]byte(webhookPayload)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", signature)
	req.Header.Set("X-GitHub-Event", "push")

	client := &http.Client{}
	webhookResp, err := client.Do(req)
	require.NoError(t, err)
	defer webhookResp.Body.Close()

	webhookDuration := time.Since(startTime).Milliseconds()

	// THEN: Verify webhook processing
	assert.Equal(t, http.StatusOK, webhookResp.StatusCode, "Valid webhook should return 200 OK")
	assert.Less(t, webhookDuration, int64(100), "Webhook response should be <100ms")

	webhookRespBody, err := io.ReadAll(webhookResp.Body)
	require.NoError(t, err)

	var webhookResponse struct {
		WorkflowID string `json:"workflowId"`
		Message    string `json:"message"`
	}
	err = json.Unmarshal(webhookRespBody, &webhookResponse)
	require.NoError(t, err)

	assert.NotEmpty(t, webhookResponse.WorkflowID, "Should return triggered workflow ID")

	// Verify workflow was created
	time.Sleep(3 * time.Second)
	workflowURL := fmt.Sprintf("%s/api/v1/workflows/%s", serverURL, webhookResponse.WorkflowID)
	workflowResp, err := http.Get(workflowURL)
	require.NoError(t, err)
	defer workflowResp.Body.Close()

	assert.Equal(t, http.StatusOK, workflowResp.StatusCode, "Triggered workflow should exist")

	// Test invalid signature
	invalidReq, _ := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader([]byte(webhookPayload)))
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidReq.Header.Set("X-Hub-Signature-256", "sha256=invalidsignature")

	invalidResp, err := client.Do(invalidReq)
	require.NoError(t, err)
	defer invalidResp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, invalidResp.StatusCode, "Invalid signature should return 401")

	// Test filtered event (wrong branch)
	filteredPayload := `{
		"event": "push",
		"ref": "refs/heads/feature-branch",
		"repository": {"name": "test-repo"}
	}`

	filteredMac := hmac.New(sha256.New, []byte(secret))
	filteredMac.Write([]byte(filteredPayload))
	filteredSignature := "sha256=" + hex.EncodeToString(filteredMac.Sum(nil))

	filteredReq, _ := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader([]byte(filteredPayload)))
	filteredReq.Header.Set("Content-Type", "application/json")
	filteredReq.Header.Set("X-Hub-Signature-256", filteredSignature)
	filteredReq.Header.Set("X-GitHub-Event", "push")

	filteredResp, err := client.Do(filteredReq)
	require.NoError(t, err)
	defer filteredResp.Body.Close()

	// Filtered events should still return 200 but not trigger workflow
	assert.Equal(t, http.StatusOK, filteredResp.StatusCode, "Filtered event should return 200")

	filteredRespBody, _ := io.ReadAll(filteredResp.Body)
	var filteredResponse struct {
		WorkflowID string `json:"workflowId"`
		Message    string `json:"message"`
	}
	json.Unmarshal(filteredRespBody, &filteredResponse)

	assert.Empty(t, filteredResponse.WorkflowID, "Filtered event should not trigger workflow")
	assert.Contains(t, filteredResponse.Message, "filtered", "Should indicate event was filtered")

	// TODO (DEV TEAM): Verify workflow metadata contains trigger info (trigger_type, trigger_source)
	// TODO (DEV TEAM): Test webhook payload passed to workflow as context
	// TODO (DEV TEAM): Verify filter rules for path patterns
	// TODO (DEV TEAM): Test disabled trigger returns 404
}
