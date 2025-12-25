package api

import (
	"fmt"
	"net/url"
	"strings"

	"go.temporal.io/api/history/v1"
)

// WorkflowSummary represents a workflow in list response (AC3)
type WorkflowSummary struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	Conclusion      string `json:"conclusion,omitempty"`
	CreatedAt       string `json:"created_at"`
	StartedAt       string `json:"started_at,omitempty"`
	CompletedAt     string `json:"completed_at,omitempty"`
	DurationSeconds *int   `json:"duration_seconds,omitempty"`
}

// buildTemporalVisibilityQuery builds Temporal visibility query from HTTP query parameters
func buildTemporalVisibilityQuery(query url.Values) string {
	conditions := []string{}

	// Status filter
	if statusList := query.Get("status"); statusList != "" {
		statuses := strings.Split(statusList, ",")
		if len(statuses) > 0 {
			statusConditions := []string{}
			for _, status := range statuses {
				status = strings.TrimSpace(status)
				// Map our status to Temporal status
				temporalStatus := mapToTemporalVisibilityStatus(status)
				if temporalStatus != "" {
					statusConditions = append(statusConditions, fmt.Sprintf("ExecutionStatus = '%s'", temporalStatus))
				}
			}
			if len(statusConditions) > 0 {
				conditions = append(conditions, "("+strings.Join(statusConditions, " OR ")+")")
			}
		}
	}

	// Name filter (fuzzy search)
	if name := query.Get("name"); name != "" {
		// Temporal uses LIKE syntax
		conditions = append(conditions, fmt.Sprintf("WorkflowType = '%s'", name))
	}

	// Time range filters
	if createdAfter := query.Get("created_after"); createdAfter != "" {
		conditions = append(conditions, fmt.Sprintf("StartTime > '%s'", createdAfter))
	}
	if createdBefore := query.Get("created_before"); createdBefore != "" {
		conditions = append(conditions, fmt.Sprintf("StartTime < '%s'", createdBefore))
	}

	if len(conditions) == 0 {
		return "" // Empty query returns all workflows
	}

	return strings.Join(conditions, " AND ")
}

// mapToTemporalVisibilityStatus maps our API status to Temporal visibility status
func mapToTemporalVisibilityStatus(status string) string {
	switch status {
	case "running":
		return "Running"
	case "completed":
		return "Completed"
	case "failed":
		return "Failed"
	case "cancelled":
		return "Canceled"
	case "timeout":
		return "TimedOut"
	case "terminated":
		return "Terminated"
	default:
		return ""
	}
}

// parseActivityType parses job and step from activity type name
// Format: "job-{jobID}-step-{stepName}" or "RunWorkflowExecutor"
func parseActivityType(activityType string) (job string, step string) {
	// Handle workflow activity (not job/step)
	if activityType == "RunWorkflowExecutor" || activityType == "" {
		return "", ""
	}

	// Parse job-{id}-step-{name} format
	parts := strings.Split(activityType, "-")
	if len(parts) >= 4 && parts[0] == "job" && parts[2] == "step" {
		job = parts[1]
		step = strings.Join(parts[3:], "-") // Handle step names with dashes
		return job, step
	}

	// Fallback: use full activity type as step name
	return "", activityType
}

// findScheduledEvent finds the scheduled event by ID from event history
func (h *WorkflowHandlers) findScheduledEvent(events []*history.HistoryEvent, scheduledEventID int64) *history.HistoryEvent {
	for _, event := range events {
		if event.EventId == scheduledEventID {
			return event
		}
	}
	return nil
}
