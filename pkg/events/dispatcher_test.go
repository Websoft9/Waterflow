package events_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/events"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// MockEventHandler for testing
type MockEventHandler struct {
	mu             sync.Mutex
	startCalls     int
	completeCalls  int
	failedCalls    int
	lastStartEvent *events.WorkflowStartEvent
	shouldFail     bool
}

func (m *MockEventHandler) OnWorkflowStart(ctx context.Context, event *events.WorkflowStartEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startCalls++
	m.lastStartEvent = event
	if m.shouldFail {
		return assert.AnError
	}
	return nil
}

func (m *MockEventHandler) OnWorkflowComplete(ctx context.Context, event *events.WorkflowCompleteEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.completeCalls++
	if m.shouldFail {
		return assert.AnError
	}
	return nil
}

func (m *MockEventHandler) OnWorkflowFailed(ctx context.Context, event *events.WorkflowFailedEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failedCalls++
	if m.shouldFail {
		return assert.AnError
	}
	return nil
}

func (m *MockEventHandler) GetCalls() (start, complete, failed int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.startCalls, m.completeCalls, m.failedCalls
}

func TestEventDispatcher_DispatchWorkflowStart(t *testing.T) {
	mockHandler := &MockEventHandler{}
	logger, _ := zap.NewDevelopment()

	dispatcher := events.NewEventDispatcher(events.EventDispatcherConfig{
		Handler: mockHandler,
		Logger:  logger,
	})

	// Dispatch event
	dispatcher.DispatchWorkflowStart(&events.WorkflowStartEvent{
		EventType:    "workflow.started",
		WorkflowID:   "test-1",
		Timestamp:    time.Now(),
		WorkflowName: "test-workflow",
	})

	// Wait for async dispatch
	time.Sleep(100 * time.Millisecond)

	start, complete, failed := mockHandler.GetCalls()
	assert.Equal(t, 1, start, "Should have called OnWorkflowStart once")
	assert.Equal(t, 0, complete, "Should not have called OnWorkflowComplete")
	assert.Equal(t, 0, failed, "Should not have called OnWorkflowFailed")
}

func TestEventDispatcher_DispatchWorkflowComplete(t *testing.T) {
	mockHandler := &MockEventHandler{}
	logger, _ := zap.NewDevelopment()

	dispatcher := events.NewEventDispatcher(events.EventDispatcherConfig{
		Handler: mockHandler,
		Logger:  logger,
	})

	// Dispatch event
	dispatcher.DispatchWorkflowComplete(&events.WorkflowCompleteEvent{
		EventType:       "workflow.completed",
		WorkflowID:      "test-1",
		Timestamp:       time.Now(),
		StartTime:       time.Now().Add(-5 * time.Minute),
		DurationSeconds: 300,
	})

	// Wait for async dispatch
	time.Sleep(100 * time.Millisecond)

	start, complete, failed := mockHandler.GetCalls()
	assert.Equal(t, 0, start, "Should not have called OnWorkflowStart")
	assert.Equal(t, 1, complete, "Should have called OnWorkflowComplete once")
	assert.Equal(t, 0, failed, "Should not have called OnWorkflowFailed")
}

func TestEventDispatcher_DispatchWorkflowFailed(t *testing.T) {
	mockHandler := &MockEventHandler{}
	logger, _ := zap.NewDevelopment()

	dispatcher := events.NewEventDispatcher(events.EventDispatcherConfig{
		Handler: mockHandler,
		Logger:  logger,
	})

	// Dispatch event
	dispatcher.DispatchWorkflowFailed(&events.WorkflowFailedEvent{
		EventType:       "workflow.failed",
		WorkflowID:      "test-1",
		Timestamp:       time.Now(),
		StartTime:       time.Now().Add(-2 * time.Minute),
		DurationSeconds: 120,
		Error: &events.WorkflowError{
			Message: "Test error",
			Type:    "TestError",
		},
	})

	// Wait for async dispatch
	time.Sleep(100 * time.Millisecond)

	start, complete, failed := mockHandler.GetCalls()
	assert.Equal(t, 0, start, "Should not have called OnWorkflowStart")
	assert.Equal(t, 0, complete, "Should not have called OnWorkflowComplete")
	assert.Equal(t, 1, failed, "Should have called OnWorkflowFailed once")
}

func TestEventDispatcher_HandlerError(t *testing.T) {
	mockHandler := &MockEventHandler{shouldFail: true}
	logger, _ := zap.NewDevelopment()

	dispatcher := events.NewEventDispatcher(events.EventDispatcherConfig{
		Handler: mockHandler,
		Logger:  logger,
	})

	// Dispatch event (handler will fail)
	dispatcher.DispatchWorkflowStart(&events.WorkflowStartEvent{
		EventType:  "workflow.started",
		WorkflowID: "test-error",
		Timestamp:  time.Now(),
	})

	// Wait for async dispatch
	time.Sleep(100 * time.Millisecond)

	start, _, _ := mockHandler.GetCalls()
	assert.Equal(t, 1, start, "Should still call handler even if it fails")
}

func TestEventDispatcher_DefaultHandler(t *testing.T) {
	// Create dispatcher without handler (should use NoOp)
	dispatcher := events.NewEventDispatcher(events.EventDispatcherConfig{})

	// Should not panic
	assert.NotPanics(t, func() {
		dispatcher.DispatchWorkflowStart(&events.WorkflowStartEvent{
			EventType:  "workflow.started",
			WorkflowID: "test-noop",
			Timestamp:  time.Now(),
		})
	})
}

func TestEventDispatcher_Async(t *testing.T) {
	mockHandler := &MockEventHandler{}
	logger, _ := zap.NewDevelopment()

	dispatcher := events.NewEventDispatcher(events.EventDispatcherConfig{
		Handler: mockHandler,
		Logger:  logger,
	})

	// Dispatch multiple events rapidly
	for i := 0; i < 10; i++ {
		dispatcher.DispatchWorkflowStart(&events.WorkflowStartEvent{
			EventType:  "workflow.started",
			WorkflowID: "test-async",
			Timestamp:  time.Now(),
		})
	}

	// Wait for all async dispatches
	time.Sleep(200 * time.Millisecond)

	start, _, _ := mockHandler.GetCalls()
	assert.Equal(t, 10, start, "Should have dispatched all 10 events")
}
