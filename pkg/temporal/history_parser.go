package temporal

import (
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/history/v1"
)

// HistoryParser parses Temporal event history to extract workflow/job/step status
type HistoryParser struct{}

// NewHistoryParser creates a new HistoryParser instance
func NewHistoryParser() *HistoryParser {
	return &HistoryParser{}
}

// JobStatus represents the status of a job in workflow execution
type JobStatus struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Status      string       `json:"status"` // pending, running, completed, failed
	StartTime   string       `json:"start_time,omitempty"`
	EndTime     string       `json:"end_time,omitempty"`
	CurrentStep string       `json:"current_step,omitempty"`
	Steps       []StepStatus `json:"steps,omitempty"`
}

// StepStatus represents the status of a step in job execution
type StepStatus struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Status    string `json:"status"` // pending, running, completed, failed, skipped
	StartTime string `json:"start_time,omitempty"`
	EndTime   string `json:"end_time,omitempty"`
	Error     string `json:"error,omitempty"`
}

// ParseJobsFromHistory extracts job and step status from workflow event history
func (p *HistoryParser) ParseJobsFromHistory(events []*history.HistoryEvent) []JobStatus {
	jobs := make([]JobStatus, 0)
	currentSteps := make(map[string]*StepStatus)        // activity_id -> StepStatus
	eventCache := make(map[int64]*history.HistoryEvent) // event_id -> HistoryEvent

	// First pass: cache all events by ID
	for _, event := range events {
		eventCache[event.EventId] = event
	}

	// Second pass: process events
	for _, event := range events {
		switch event.EventType {
		case enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED:
			// Activity (Step) scheduled
			attrs := event.GetActivityTaskScheduledEventAttributes()
			stepStatus := &StepStatus{
				Name:   attrs.ActivityType.Name,
				Status: "pending",
			}
			currentSteps[attrs.ActivityId] = stepStatus

		case enums.EVENT_TYPE_ACTIVITY_TASK_STARTED:
			// Activity (Step) started
			attrs := event.GetActivityTaskStartedEventAttributes()
			// Find scheduled event to get activity ID
			if scheduledEvent, ok := eventCache[attrs.ScheduledEventId]; ok {
				scheduledAttrs := scheduledEvent.GetActivityTaskScheduledEventAttributes()
				if step, ok := currentSteps[scheduledAttrs.ActivityId]; ok {
					step.Status = "running"
					if event.EventTime != nil {
						step.StartTime = event.EventTime.AsTime().Format("2006-01-02T15:04:05Z07:00")
					}
				}
			}

		case enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
			// Activity (Step) completed
			attrs := event.GetActivityTaskCompletedEventAttributes()
			if scheduledEvent, ok := eventCache[attrs.ScheduledEventId]; ok {
				scheduledAttrs := scheduledEvent.GetActivityTaskScheduledEventAttributes()
				if step, ok := currentSteps[scheduledAttrs.ActivityId]; ok {
					step.Status = "completed"
					if event.EventTime != nil {
						step.EndTime = event.EventTime.AsTime().Format("2006-01-02T15:04:05Z07:00")
					}
				}
			}

		case enums.EVENT_TYPE_ACTIVITY_TASK_FAILED:
			// Activity (Step) failed
			attrs := event.GetActivityTaskFailedEventAttributes()
			if scheduledEvent, ok := eventCache[attrs.ScheduledEventId]; ok {
				scheduledAttrs := scheduledEvent.GetActivityTaskScheduledEventAttributes()
				if step, ok := currentSteps[scheduledAttrs.ActivityId]; ok {
					step.Status = "failed"
					if event.EventTime != nil {
						step.EndTime = event.EventTime.AsTime().Format("2006-01-02T15:04:05Z07:00")
					}
					if attrs.Failure != nil {
						step.Error = attrs.Failure.Message
					}
				}
			}

		case enums.EVENT_TYPE_ACTIVITY_TASK_TIMED_OUT:
			// Activity (Step) timed out
			attrs := event.GetActivityTaskTimedOutEventAttributes()
			if scheduledEvent, ok := eventCache[attrs.ScheduledEventId]; ok {
				scheduledAttrs := scheduledEvent.GetActivityTaskScheduledEventAttributes()
				if step, ok := currentSteps[scheduledAttrs.ActivityId]; ok {
					step.Status = "timeout"
					if event.EventTime != nil {
						step.EndTime = event.EventTime.AsTime().Format("2006-01-02T15:04:05Z07:00")
					}
					step.Error = "Activity execution timed out"
				}
			}
		}
	}

	// Aggregate steps into jobs
	// Note: This is a simplified implementation. In reality, we would need to
	// parse child workflow events to properly group steps by job.
	// For now, we create a single "default" job containing all steps.
	if len(currentSteps) > 0 {
		steps := make([]StepStatus, 0, len(currentSteps))
		for _, step := range currentSteps {
			steps = append(steps, *step)
		}

		job := JobStatus{
			ID:     "default",
			Name:   "default",
			Status: "running",
			Steps:  steps,
		}

		// Determine job status from steps
		allCompleted := true
		anyFailed := false
		for _, step := range steps {
			if step.Status == "failed" || step.Status == "timeout" {
				anyFailed = true
			}
			if step.Status != "completed" {
				allCompleted = false
			}
		}

		if anyFailed {
			job.Status = "failed"
		} else if allCompleted {
			job.Status = "completed"
		}

		jobs = append(jobs, job)
	}

	return jobs
}
