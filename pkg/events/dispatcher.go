package events

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// EventDispatcher dispatches workflow events to handlers asynchronously.
type EventDispatcher struct {
	handler EventHandler
	logger  *zap.Logger
	timeout time.Duration
}

// EventDispatcherConfig contains event dispatcher configuration.
type EventDispatcherConfig struct {
	// Handler is the event handler to use
	Handler EventHandler

	// Logger for error logging (optional)
	Logger *zap.Logger

	// Timeout for event handler execution (default: 10s)
	Timeout time.Duration
}

// NewEventDispatcher creates a new event dispatcher.
func NewEventDispatcher(config EventDispatcherConfig) *EventDispatcher {
	handler := config.Handler
	if handler == nil {
		handler = NewNoOpEventHandler()
	}

	logger := config.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &EventDispatcher{
		handler: handler,
		logger:  logger,
		timeout: timeout,
	}
}

// DispatchWorkflowStart dispatches a workflow start event asynchronously.
func (d *EventDispatcher) DispatchWorkflowStart(event *WorkflowStartEvent) {
	go d.handleEvent(func(ctx context.Context) error {
		return d.handler.OnWorkflowStart(ctx, event)
	}, "workflow.started", event.WorkflowID)
}

// DispatchWorkflowComplete dispatches a workflow complete event asynchronously.
func (d *EventDispatcher) DispatchWorkflowComplete(event *WorkflowCompleteEvent) {
	go d.handleEvent(func(ctx context.Context) error {
		return d.handler.OnWorkflowComplete(ctx, event)
	}, "workflow.completed", event.WorkflowID)
}

// DispatchWorkflowFailed dispatches a workflow failed event asynchronously.
func (d *EventDispatcher) DispatchWorkflowFailed(event *WorkflowFailedEvent) {
	go d.handleEvent(func(ctx context.Context) error {
		return d.handler.OnWorkflowFailed(ctx, event)
	}, "workflow.failed", event.WorkflowID)
}

// handleEvent executes the event handler with timeout.
func (d *EventDispatcher) handleEvent(fn func(context.Context) error, eventType, workflowID string) {
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()

	if err := fn(ctx); err != nil {
		d.logger.Error("Event handler failed",
			zap.String("event_type", eventType),
			zap.String("workflow_id", workflowID),
			zap.Error(err),
		)
	}
}
