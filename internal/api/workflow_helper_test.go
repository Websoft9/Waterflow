package api

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildTemporalVisibilityQuery_Empty(t *testing.T) {
	query := url.Values{}
	result := buildTemporalVisibilityQuery(query)
	assert.Empty(t, result, "empty query should return empty string")
}

func TestBuildTemporalVisibilityQuery_StatusFilter(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		contains string
	}{
		{"running", "running", "ExecutionStatus = 'Running'"},
		{"completed", "completed", "ExecutionStatus = 'Completed'"},
		{"failed", "failed", "ExecutionStatus = 'Failed'"},
		{"cancelled", "cancelled", "ExecutionStatus = 'Canceled'"},
		{"timeout", "timeout", "ExecutionStatus = 'TimedOut'"},
		{"terminated", "terminated", "ExecutionStatus = 'Terminated'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := url.Values{}
			query.Set("status", tt.status)
			result := buildTemporalVisibilityQuery(query)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestBuildTemporalVisibilityQuery_MultipleStatuses(t *testing.T) {
	query := url.Values{}
	query.Set("status", "running,completed")
	result := buildTemporalVisibilityQuery(query)
	assert.Contains(t, result, "ExecutionStatus = 'Running'")
	assert.Contains(t, result, "ExecutionStatus = 'Completed'")
	assert.Contains(t, result, " OR ")
}

func TestBuildTemporalVisibilityQuery_InvalidStatus(t *testing.T) {
	query := url.Values{}
	query.Set("status", "invalid-status")
	result := buildTemporalVisibilityQuery(query)
	assert.Empty(t, result, "invalid status should be ignored")
}

func TestBuildTemporalVisibilityQuery_NameFilter(t *testing.T) {
	query := url.Values{}
	query.Set("name", "test-workflow")
	result := buildTemporalVisibilityQuery(query)
	assert.Contains(t, result, "WorkflowType = 'test-workflow'")
}

func TestBuildTemporalVisibilityQuery_TimeFilters(t *testing.T) {
	query := url.Values{}
	query.Set("created_after", "2024-01-01T00:00:00Z")
	query.Set("created_before", "2024-12-31T23:59:59Z")
	result := buildTemporalVisibilityQuery(query)
	assert.Contains(t, result, "StartTime > '2024-01-01T00:00:00Z'")
	assert.Contains(t, result, "StartTime < '2024-12-31T23:59:59Z'")
}

func TestBuildTemporalVisibilityQuery_CombinedFilters(t *testing.T) {
	query := url.Values{}
	query.Set("status", "running")
	query.Set("name", "deploy")
	result := buildTemporalVisibilityQuery(query)
	assert.Contains(t, result, " AND ")
	assert.Contains(t, result, "ExecutionStatus")
	assert.Contains(t, result, "WorkflowType")
}

func TestMapToTemporalVisibilityStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"running", "Running"},
		{"completed", "Completed"},
		{"failed", "Failed"},
		{"cancelled", "Canceled"},
		{"timeout", "TimedOut"},
		{"terminated", "Terminated"},
		{"unknown", ""},
		{"invalid", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapToTemporalVisibilityStatus(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseActivityType(t *testing.T) {
	tests := []struct {
		name         string
		activityType string
		expectedJob  string
		expectedStep string
	}{
		{
			name:         "valid job-step format",
			activityType: "job-build-step-compile",
			expectedJob:  "build",
			expectedStep: "compile",
		},
		{
			name:         "step name with dashes",
			activityType: "job-deploy-step-run-tests-unit",
			expectedJob:  "deploy",
			expectedStep: "run-tests-unit",
		},
		{
			name:         "workflow executor",
			activityType: "RunWorkflowExecutor",
			expectedJob:  "",
			expectedStep: "",
		},
		{
			name:         "empty activity type",
			activityType: "",
			expectedJob:  "",
			expectedStep: "",
		},
		{
			name:         "fallback for unknown format",
			activityType: "custom-activity",
			expectedJob:  "",
			expectedStep: "custom-activity",
		},
		{
			name:         "partial job-step format",
			activityType: "job-only",
			expectedJob:  "",
			expectedStep: "job-only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job, step := parseActivityType(tt.activityType)
			assert.Equal(t, tt.expectedJob, job)
			assert.Equal(t, tt.expectedStep, step)
		})
	}
}

func TestWorkflowSummary_Structure(t *testing.T) {
	summary := WorkflowSummary{
		ID:        "wf-123",
		Name:      "test-workflow",
		Status:    "running",
		CreatedAt: "2024-01-01T00:00:00Z",
		StartedAt: "2024-01-01T00:00:01Z",
	}

	assert.Equal(t, "wf-123", summary.ID)
	assert.Equal(t, "test-workflow", summary.Name)
	assert.Equal(t, "running", summary.Status)
	assert.Empty(t, summary.Conclusion)
	assert.Nil(t, summary.DurationSeconds)
}

func TestWorkflowSummary_WithConclusion(t *testing.T) {
	duration := 120
	summary := WorkflowSummary{
		ID:              "wf-456",
		Name:            "completed-workflow",
		Status:          "completed",
		Conclusion:      "success",
		CreatedAt:       "2024-01-01T00:00:00Z",
		StartedAt:       "2024-01-01T00:00:01Z",
		CompletedAt:     "2024-01-01T00:02:01Z",
		DurationSeconds: &duration,
	}

	assert.Equal(t, "completed", summary.Status)
	assert.Equal(t, "success", summary.Conclusion)
	assert.NotNil(t, summary.DurationSeconds)
	assert.Equal(t, 120, *summary.DurationSeconds)
}
