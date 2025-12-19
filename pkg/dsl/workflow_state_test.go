package dsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWorkflowState(t *testing.T) {
	state := NewWorkflowState("wf-123")

	assert.Equal(t, "wf-123", state.WorkflowID)
	assert.Equal(t, "running", state.Status)
	assert.NotNil(t, state.JobStates)
	assert.Empty(t, state.JobStates)
}

func TestWorkflowState_UpdateJobState(t *testing.T) {
	state := NewWorkflowState("wf-123")

	outputs := map[string]string{
		"version": "1.0.0",
		"image":   "myapp:latest",
	}

	state.UpdateJobState("build", "completed", "success", outputs)

	jobState := state.GetJobState("build")
	assert.NotNil(t, jobState)
	assert.Equal(t, "build", jobState.JobID)
	assert.Equal(t, "completed", jobState.Status)
	assert.Equal(t, "success", jobState.Conclusion)
	assert.Equal(t, outputs, jobState.Outputs)
}

func TestWorkflowState_AddStepState(t *testing.T) {
	state := NewWorkflowState("wf-123")

	stepState := &StepState{
		StepID:     "step-1",
		Name:       "Build",
		Status:     "completed",
		Conclusion: "success",
		Outputs:    map[string]string{"artifact": "app.zip"},
	}

	state.AddStepState("build", stepState)

	jobState := state.GetJobState("build")
	assert.NotNil(t, jobState)
	assert.Len(t, jobState.StepStates, 1)
	assert.Equal(t, stepState, jobState.StepStates[0])
}

func TestWorkflowState_MarkCompleted(t *testing.T) {
	state := NewWorkflowState("wf-123")
	state.MarkCompleted()

	assert.Equal(t, "completed", state.Status)
}

func TestWorkflowState_MarkFailed(t *testing.T) {
	state := NewWorkflowState("wf-123")
	state.MarkFailed()

	assert.Equal(t, "failed", state.Status)
}

func TestWorkflowState_MarkCancelled(t *testing.T) {
	state := NewWorkflowState("wf-123")
	state.MarkCancelled()

	assert.Equal(t, "cancelled", state.Status)
}

func TestWorkflowState_GetJobStateNotExists(t *testing.T) {
	state := NewWorkflowState("wf-123")

	jobState := state.GetJobState("nonexistent")
	assert.Nil(t, jobState)
}

func TestWorkflowState_ConcurrentAccess(t *testing.T) {
	state := NewWorkflowState("wf-123")

	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func() {
			state.UpdateJobState("job-1", "running", "pending", nil)
			done <- true
		}()
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			_ = state.GetJobState("job-1")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Should not panic and job should exist
	jobState := state.GetJobState("job-1")
	assert.NotNil(t, jobState)
}

func TestWorkflowState_MultipleJobs(t *testing.T) {
	state := NewWorkflowState("wf-123")

	state.UpdateJobState("build", "completed", "success", map[string]string{"version": "1.0.0"})
	state.UpdateJobState("test", "running", "pending", nil)
	state.UpdateJobState("deploy", "pending", "pending", nil)

	assert.Len(t, state.JobStates, 3)
	assert.Equal(t, "completed", state.GetJobState("build").Status)
	assert.Equal(t, "running", state.GetJobState("test").Status)
	assert.Equal(t, "pending", state.GetJobState("deploy").Status)
}

func TestWorkflowState_AddMultipleSteps(t *testing.T) {
	state := NewWorkflowState("wf-123")

	steps := []*StepState{
		{StepID: "step-1", Name: "Checkout", Status: "completed", Conclusion: "success"},
		{StepID: "step-2", Name: "Build", Status: "completed", Conclusion: "success"},
		{StepID: "step-3", Name: "Test", Status: "running", Conclusion: "pending"},
	}

	for _, step := range steps {
		state.AddStepState("build", step)
	}

	jobState := state.GetJobState("build")
	assert.NotNil(t, jobState)
	assert.Len(t, jobState.StepStates, 3)

	for i, step := range steps {
		assert.Equal(t, step, jobState.StepStates[i])
	}
}
