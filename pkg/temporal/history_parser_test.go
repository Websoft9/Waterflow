package temporal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/history/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNewHistoryParser(t *testing.T) {
	parser := NewHistoryParser()
	assert.NotNil(t, parser)
}

func TestParseJobsFromHistory_EmptyEvents(t *testing.T) {
	parser := NewHistoryParser()

	// Test nil slice
	jobs := parser.ParseJobsFromHistory(nil)
	assert.Empty(t, jobs)

	// Test empty slice
	jobs = parser.ParseJobsFromHistory([]*history.HistoryEvent{})
	assert.Empty(t, jobs)

	// Test slice with nil elements
	jobs = parser.ParseJobsFromHistory([]*history.HistoryEvent{nil, nil})
	assert.Empty(t, jobs)
}

func TestParseJobsFromHistory_ActivityScheduled(t *testing.T) {
	parser := NewHistoryParser()

	events := []*history.HistoryEvent{
		{
			EventId:   1,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
			Attributes: &history.HistoryEvent_ActivityTaskScheduledEventAttributes{
				ActivityTaskScheduledEventAttributes: &history.ActivityTaskScheduledEventAttributes{
					ActivityId: "activity-1",
					ActivityType: &commonpb.ActivityType{
						Name: "ExecuteStepActivity",
					},
				},
			},
		},
	}

	jobs := parser.ParseJobsFromHistory(events)
	assert.NotNil(t, jobs)
	assert.Len(t, jobs, 1)
	assert.Equal(t, "default", jobs[0].Name)
	assert.Len(t, jobs[0].Steps, 1)
	assert.Equal(t, "pending", jobs[0].Steps[0].Status)
}

func TestParseJobsFromHistory_ActivityStarted(t *testing.T) {
	parser := NewHistoryParser()

	events := []*history.HistoryEvent{
		{
			EventId:   1,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
			Attributes: &history.HistoryEvent_ActivityTaskScheduledEventAttributes{
				ActivityTaskScheduledEventAttributes: &history.ActivityTaskScheduledEventAttributes{
					ActivityId: "activity-1",
					ActivityType: &commonpb.ActivityType{
						Name: "ExecuteStepActivity",
					},
				},
			},
		},
		{
			EventId:   2,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_STARTED,
			EventTime: timestamppb.Now(),
			Attributes: &history.HistoryEvent_ActivityTaskStartedEventAttributes{
				ActivityTaskStartedEventAttributes: &history.ActivityTaskStartedEventAttributes{
					ScheduledEventId: 1,
				},
			},
		},
	}

	jobs := parser.ParseJobsFromHistory(events)
	assert.NotNil(t, jobs)
	assert.Len(t, jobs, 1)
	assert.Len(t, jobs[0].Steps, 1)
	assert.Equal(t, "running", jobs[0].Steps[0].Status)
}

func TestParseJobsFromHistory_ActivityCompleted(t *testing.T) {
	parser := NewHistoryParser()

	events := []*history.HistoryEvent{
		{
			EventId:   1,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
			Attributes: &history.HistoryEvent_ActivityTaskScheduledEventAttributes{
				ActivityTaskScheduledEventAttributes: &history.ActivityTaskScheduledEventAttributes{
					ActivityId: "activity-1",
					ActivityType: &commonpb.ActivityType{
						Name: "ExecuteStepActivity",
					},
				},
			},
		},
		{
			EventId:   2,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_STARTED,
			EventTime: timestamppb.Now(),
			Attributes: &history.HistoryEvent_ActivityTaskStartedEventAttributes{
				ActivityTaskStartedEventAttributes: &history.ActivityTaskStartedEventAttributes{
					ScheduledEventId: 1,
				},
			},
		},
		{
			EventId:   3,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
			EventTime: timestamppb.Now(),
			Attributes: &history.HistoryEvent_ActivityTaskCompletedEventAttributes{
				ActivityTaskCompletedEventAttributes: &history.ActivityTaskCompletedEventAttributes{
					ScheduledEventId: 1,
				},
			},
		},
	}

	jobs := parser.ParseJobsFromHistory(events)
	assert.NotNil(t, jobs)
	assert.Len(t, jobs, 1)
	assert.Equal(t, "completed", jobs[0].Status)
	assert.Len(t, jobs[0].Steps, 1)
	assert.Equal(t, "completed", jobs[0].Steps[0].Status)
}

func TestParseJobsFromHistory_ActivityFailed(t *testing.T) {
	parser := NewHistoryParser()

	events := []*history.HistoryEvent{
		{
			EventId:   1,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
			Attributes: &history.HistoryEvent_ActivityTaskScheduledEventAttributes{
				ActivityTaskScheduledEventAttributes: &history.ActivityTaskScheduledEventAttributes{
					ActivityId: "activity-1",
					ActivityType: &commonpb.ActivityType{
						Name: "ExecuteStepActivity",
					},
				},
			},
		},
		{
			EventId:   2,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_FAILED,
			EventTime: timestamppb.Now(),
			Attributes: &history.HistoryEvent_ActivityTaskFailedEventAttributes{
				ActivityTaskFailedEventAttributes: &history.ActivityTaskFailedEventAttributes{
					ScheduledEventId: 1,
				},
			},
		},
	}

	jobs := parser.ParseJobsFromHistory(events)
	assert.NotNil(t, jobs)
	assert.Len(t, jobs, 1)
	assert.Equal(t, "failed", jobs[0].Status)
}

func TestParseJobsFromHistory_ActivityTimedOut(t *testing.T) {
	parser := NewHistoryParser()

	events := []*history.HistoryEvent{
		{
			EventId:   1,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
			Attributes: &history.HistoryEvent_ActivityTaskScheduledEventAttributes{
				ActivityTaskScheduledEventAttributes: &history.ActivityTaskScheduledEventAttributes{
					ActivityId: "activity-1",
					ActivityType: &commonpb.ActivityType{
						Name: "ExecuteStepActivity",
					},
				},
			},
		},
		{
			EventId:   2,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_TIMED_OUT,
			EventTime: timestamppb.Now(),
			Attributes: &history.HistoryEvent_ActivityTaskTimedOutEventAttributes{
				ActivityTaskTimedOutEventAttributes: &history.ActivityTaskTimedOutEventAttributes{
					ScheduledEventId: 1,
				},
			},
		},
	}

	jobs := parser.ParseJobsFromHistory(events)
	assert.NotNil(t, jobs)
	assert.Len(t, jobs, 1)
	assert.Equal(t, "failed", jobs[0].Status)
	assert.Contains(t, jobs[0].Steps[0].Error, "timed out")
}

func TestParseJobsFromHistory_MultipleActivities(t *testing.T) {
	parser := NewHistoryParser()

	events := []*history.HistoryEvent{
		{
			EventId:   1,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
			Attributes: &history.HistoryEvent_ActivityTaskScheduledEventAttributes{
				ActivityTaskScheduledEventAttributes: &history.ActivityTaskScheduledEventAttributes{
					ActivityId: "step-1",
					ActivityType: &commonpb.ActivityType{
						Name: "ExecuteStepActivity",
					},
				},
			},
		},
		{
			EventId:   2,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
			EventTime: timestamppb.Now(),
			Attributes: &history.HistoryEvent_ActivityTaskCompletedEventAttributes{
				ActivityTaskCompletedEventAttributes: &history.ActivityTaskCompletedEventAttributes{
					ScheduledEventId: 1,
				},
			},
		},
		{
			EventId:   3,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
			Attributes: &history.HistoryEvent_ActivityTaskScheduledEventAttributes{
				ActivityTaskScheduledEventAttributes: &history.ActivityTaskScheduledEventAttributes{
					ActivityId: "step-2",
					ActivityType: &commonpb.ActivityType{
						Name: "ExecuteStepActivity",
					},
				},
			},
		},
		{
			EventId:   4,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_STARTED,
			EventTime: timestamppb.Now(),
			Attributes: &history.HistoryEvent_ActivityTaskStartedEventAttributes{
				ActivityTaskStartedEventAttributes: &history.ActivityTaskStartedEventAttributes{
					ScheduledEventId: 3,
				},
			},
		},
	}

	jobs := parser.ParseJobsFromHistory(events)
	assert.NotNil(t, jobs)
	assert.Len(t, jobs, 1)
	assert.Len(t, jobs[0].Steps, 2)
	assert.Equal(t, "running", jobs[0].Status) // One step is still running
}

func TestParseJobsFromHistory_NilAttributes(t *testing.T) {
	parser := NewHistoryParser()

	// Events with nil attributes should be handled gracefully
	events := []*history.HistoryEvent{
		{
			EventId:    1,
			EventType:  enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
			Attributes: nil, // nil attributes
		},
		{
			EventId:   2,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
			Attributes: &history.HistoryEvent_ActivityTaskScheduledEventAttributes{
				ActivityTaskScheduledEventAttributes: &history.ActivityTaskScheduledEventAttributes{
					ActivityId:   "step-1",
					ActivityType: nil, // nil activity type
				},
			},
		},
		{
			EventId:   3,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_STARTED,
			Attributes: &history.HistoryEvent_ActivityTaskStartedEventAttributes{
				ActivityTaskStartedEventAttributes: nil, // nil inner attributes
			},
		},
		{
			EventId:   4,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
			Attributes: &history.HistoryEvent_ActivityTaskCompletedEventAttributes{
				ActivityTaskCompletedEventAttributes: nil, // nil inner attributes
			},
		},
		{
			EventId:   5,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_FAILED,
			Attributes: &history.HistoryEvent_ActivityTaskFailedEventAttributes{
				ActivityTaskFailedEventAttributes: nil, // nil inner attributes
			},
		},
	}

	// Should not panic
	jobs := parser.ParseJobsFromHistory(events)
	assert.NotNil(t, jobs)
}

func TestJobStatus_Struct(t *testing.T) {
	job := JobStatus{
		ID:          "job-1",
		Name:        "build",
		Status:      "completed",
		StartTime:   "2026-01-23T10:00:00Z",
		EndTime:     "2026-01-23T10:05:00Z",
		CurrentStep: "step-3",
		Steps: []StepStatus{
			{ID: "1", Name: "checkout", Status: "completed"},
			{ID: "2", Name: "build", Status: "completed"},
			{ID: "3", Name: "test", Status: "completed"},
		},
	}

	assert.Equal(t, "job-1", job.ID)
	assert.Equal(t, "build", job.Name)
	assert.Equal(t, "completed", job.Status)
	assert.Len(t, job.Steps, 3)
}

func TestStepStatus_Struct(t *testing.T) {
	step := StepStatus{
		ID:        "step-1",
		Name:      "Run tests",
		Status:    "failed",
		StartTime: "2026-01-23T10:00:00Z",
		EndTime:   "2026-01-23T10:01:00Z",
		Error:     "test failed: exit code 1",
	}

	assert.Equal(t, "step-1", step.ID)
	assert.Equal(t, "Run tests", step.Name)
	assert.Equal(t, "failed", step.Status)
	assert.Contains(t, step.Error, "exit code 1")
}
