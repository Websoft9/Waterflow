// Package events provides workflow lifecycle event handling interfaces
package events

import (
	"context"
	"time"
)

// EventHandler defines the interface for handling workflow lifecycle events.
type EventHandler interface {
	OnWorkflowStart(ctx context.Context, event *WorkflowStartEvent) error
	OnWorkflowComplete(ctx context.Context, event *WorkflowCompleteEvent) error
	OnWorkflowFailed(ctx context.Context, event *WorkflowFailedEvent) error
}

// WorkflowStartEvent represents a workflow start event.
type WorkflowStartEvent struct {
	EventType    string                 `json:"event_type"`
	WorkflowID   string                 `json:"workflow_id"`
	Timestamp    time.Time              `json:"timestamp"`
	WorkflowName string                 `json:"workflow_name,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// WorkflowCompleteEvent represents a workflow completion event.
type WorkflowCompleteEvent struct {
	EventType       string          `json:"event_type"`
	WorkflowID      string          `json:"workflow_id"`
	Timestamp       time.Time       `json:"timestamp"`
	StartTime       time.Time       `json:"start_time"`
	DurationSeconds float64         `json:"duration_seconds"`
	Result          *WorkflowResult `json:"result,omitempty"`
}

// WorkflowResult contains workflow execution result details.
type WorkflowResult struct {
	Status     string `json:"status"`
	JobsCount  int    `json:"jobs_count,omitempty"`
	StepsCount int    `json:"steps_count,omitempty"`
}

// WorkflowFailedEvent represents a workflow failure event.
type WorkflowFailedEvent struct {
	EventType       string         `json:"event_type"`
	WorkflowID      string         `json:"workflow_id"`
	Timestamp       time.Time      `json:"timestamp"`
	StartTime       time.Time      `json:"start_time"`
	DurationSeconds float64        `json:"duration_seconds"`
	Error           *WorkflowError `json:"error,omitempty"`
}

// WorkflowError contains workflow failure details.
type WorkflowError struct {
	Message string                 `json:"message"`
	Type    string                 `json:"type,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}
