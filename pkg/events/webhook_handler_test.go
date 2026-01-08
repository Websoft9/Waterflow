package events_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/events"
	"github.com/stretchr/testify/assert"
)

func TestWebhookEventHandler_Success(t *testing.T) {
	// Create test server
	received := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create handler
	handler := events.NewWebhookEventHandler(events.WebhookConfig{
		URL: server.URL,
	})

	// Send event
	err := handler.OnWorkflowStart(context.Background(), &events.WorkflowStartEvent{
		EventType:    "workflow.started",
		WorkflowID:   "test-1",
		Timestamp:    time.Now(),
		WorkflowName: "test-workflow",
	})

	assert.NoError(t, err)
	assert.True(t, received, "Server should have received the request")
}

func TestWebhookEventHandler_CustomHeaders(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-value", r.Header.Get("X-Custom-Header"))
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create handler with custom headers
	handler := events.NewWebhookEventHandler(events.WebhookConfig{
		URL: server.URL,
		Headers: map[string]string{
			"X-Custom-Header": "test-value",
			"Authorization":   "Bearer token123",
		},
	})

	// Send event
	err := handler.OnWorkflowStart(context.Background(), &events.WorkflowStartEvent{
		EventType:  "workflow.started",
		WorkflowID: "test-2",
		Timestamp:  time.Now(),
	})

	assert.NoError(t, err)
}

func TestWebhookEventHandler_Timeout(t *testing.T) {
	// Create slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second) // Longer than timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create handler with short timeout
	handler := events.NewWebhookEventHandler(events.WebhookConfig{
		URL:     server.URL,
		Timeout: 100 * time.Millisecond,
	})

	// Send event (should timeout)
	err := handler.OnWorkflowStart(context.Background(), &events.WorkflowStartEvent{
		EventType:  "workflow.started",
		WorkflowID: "test-timeout",
		Timestamp:  time.Now(),
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestWebhookEventHandler_ServerError(t *testing.T) {
	// Create server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create handler
	handler := events.NewWebhookEventHandler(events.WebhookConfig{
		URL: server.URL,
	})

	// Send event
	err := handler.OnWorkflowStart(context.Background(), &events.WorkflowStartEvent{
		EventType:  "workflow.started",
		WorkflowID: "test-error",
		Timestamp:  time.Now(),
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook returned status 500")
}

func TestWebhookEventHandler_EmptyURL(t *testing.T) {
	// Create handler with empty URL
	handler := events.NewWebhookEventHandler(events.WebhookConfig{
		URL: "",
	})

	// Send event (should fail)
	err := handler.OnWorkflowStart(context.Background(), &events.WorkflowStartEvent{
		EventType:  "workflow.started",
		WorkflowID: "test-empty-url",
		Timestamp:  time.Now(),
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook URL is not configured")
}

func TestWebhookEventHandler_AllEventTypes(t *testing.T) {
	// Create test server
	eventCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		eventCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create handler
	handler := events.NewWebhookEventHandler(events.WebhookConfig{
		URL: server.URL,
	})

	// Test all event types
	ctx := context.Background()

	// Workflow started
	err := handler.OnWorkflowStart(ctx, &events.WorkflowStartEvent{
		EventType:  "workflow.started",
		WorkflowID: "test-3",
		Timestamp:  time.Now(),
	})
	assert.NoError(t, err)

	// Workflow completed
	err = handler.OnWorkflowComplete(ctx, &events.WorkflowCompleteEvent{
		EventType:       "workflow.completed",
		WorkflowID:      "test-3",
		Timestamp:       time.Now(),
		StartTime:       time.Now().Add(-5 * time.Minute),
		DurationSeconds: 300,
	})
	assert.NoError(t, err)

	// Workflow failed
	err = handler.OnWorkflowFailed(ctx, &events.WorkflowFailedEvent{
		EventType:       "workflow.failed",
		WorkflowID:      "test-4",
		Timestamp:       time.Now(),
		StartTime:       time.Now().Add(-2 * time.Minute),
		DurationSeconds: 120,
		Error: &events.WorkflowError{
			Message: "Test error",
			Type:    "TestError",
		},
	})
	assert.NoError(t, err)

	assert.Equal(t, 3, eventCount, "Should have received 3 events")
}
