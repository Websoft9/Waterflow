package events_test

import (
	"context"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/events"
	"github.com/stretchr/testify/assert"
)

func TestNoOpEventHandler(t *testing.T) {
	handler := events.NewNoOpEventHandler()
	ctx := context.Background()

	err := handler.OnWorkflowStart(ctx, &events.WorkflowStartEvent{
		EventType:    "workflow.started",
		WorkflowID:   "test-1",
		Timestamp:    time.Now(),
		WorkflowName: "test-workflow",
	})
	assert.NoError(t, err)

	err = handler.OnWorkflowComplete(ctx, &events.WorkflowCompleteEvent{
		EventType:  "workflow.completed",
		WorkflowID: "test-1",
		Timestamp:  time.Now(),
	})
	assert.NoError(t, err)

	err = handler.OnWorkflowFailed(ctx, &events.WorkflowFailedEvent{
		EventType:  "workflow.failed",
		WorkflowID: "test-1",
		Timestamp:  time.Now(),
	})
	assert.NoError(t, err)
}
