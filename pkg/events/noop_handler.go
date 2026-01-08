package events

import "context"

// NoOpEventHandler is a no-operation event handler that does nothing.
// This is the default handler when no custom handler is configured.
type NoOpEventHandler struct{}

// NewNoOpEventHandler creates a new no-op event handler.
func NewNoOpEventHandler() *NoOpEventHandler {
	return &NoOpEventHandler{}
}

// OnWorkflowStart does nothing.
func (h *NoOpEventHandler) OnWorkflowStart(ctx context.Context, event *WorkflowStartEvent) error {
	return nil
}

// OnWorkflowComplete does nothing.
func (h *NoOpEventHandler) OnWorkflowComplete(ctx context.Context, event *WorkflowCompleteEvent) error {
	return nil
}

// OnWorkflowFailed does nothing.
func (h *NoOpEventHandler) OnWorkflowFailed(ctx context.Context, event *WorkflowFailedEvent) error {
	return nil
}
