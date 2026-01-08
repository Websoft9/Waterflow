package events

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WebhookEventHandler sends workflow events to a webhook endpoint via HTTP POST.
type WebhookEventHandler struct {
	url     string
	headers map[string]string
	timeout time.Duration
	client  *http.Client
}

// WebhookConfig contains webhook handler configuration.
type WebhookConfig struct {
	// URL is the webhook endpoint URL
	URL string

	// Headers are additional HTTP headers to send (optional)
	Headers map[string]string

	// Timeout is the HTTP request timeout (default: 5s)
	Timeout time.Duration
}

// NewWebhookEventHandler creates a new webhook event handler.
func NewWebhookEventHandler(config WebhookConfig) *WebhookEventHandler {
	// Validate URL if provided (optional, allows graceful degradation)
	// URL validation is lenient to allow runtime error handling

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	return &WebhookEventHandler{
		url:     config.URL,
		headers: config.Headers,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// OnWorkflowStart sends workflow start event to webhook.
func (h *WebhookEventHandler) OnWorkflowStart(ctx context.Context, event *WorkflowStartEvent) error {
	return h.sendEvent(ctx, event)
}

// OnWorkflowComplete sends workflow complete event to webhook.
func (h *WebhookEventHandler) OnWorkflowComplete(ctx context.Context, event *WorkflowCompleteEvent) error {
	return h.sendEvent(ctx, event)
}

// OnWorkflowFailed sends workflow failed event to webhook.
func (h *WebhookEventHandler) OnWorkflowFailed(ctx context.Context, event *WorkflowFailedEvent) error {
	return h.sendEvent(ctx, event)
}

// sendEvent sends an event to the webhook endpoint.
func (h *WebhookEventHandler) sendEvent(ctx context.Context, event interface{}) error {
	// Check if URL is configured
	if h.url == "" {
		return fmt.Errorf("webhook URL is not configured")
	}

	// Marshal event to JSON
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range h.headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
