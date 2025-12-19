package dsl

import "sync"

// WorkflowState tracks the execution state of a workflow.
// It is safe for concurrent use.
type WorkflowState struct {
	mu         sync.RWMutex
	WorkflowID string
	Status     string // running, completed, failed, cancelled
	JobStates  map[string]*JobState
}

// JobState tracks the state of a job
type JobState struct {
	JobID      string
	Status     string // pending, running, completed, skipped, failed
	Conclusion string // success, failure, skipped, cancelled
	Outputs    map[string]string
	StepStates []*StepState
}

// StepState tracks the state of a step
type StepState struct {
	StepID      string
	Name        string
	Status      string // pending, running, completed, skipped
	Conclusion  string // success, failure, skipped
	Outputs     map[string]string
	Error       string
	ContinuedOn bool
}

// NewWorkflowState creates a new workflow state tracker
func NewWorkflowState(workflowID string) *WorkflowState {
	return &WorkflowState{
		WorkflowID: workflowID,
		Status:     "running",
		JobStates:  make(map[string]*JobState),
	}
}

// UpdateJobState updates the state of a job
func (ws *WorkflowState) UpdateJobState(jobID string, status, conclusion string, outputs map[string]string) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.JobStates[jobID] == nil {
		ws.JobStates[jobID] = &JobState{
			JobID:      jobID,
			StepStates: make([]*StepState, 0),
		}
	}

	ws.JobStates[jobID].Status = status
	ws.JobStates[jobID].Conclusion = conclusion
	ws.JobStates[jobID].Outputs = outputs
}

// AddStepState adds a step state to a job
func (ws *WorkflowState) AddStepState(jobID string, stepState *StepState) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.JobStates[jobID] == nil {
		ws.JobStates[jobID] = &JobState{
			JobID:      jobID,
			StepStates: make([]*StepState, 0),
		}
	}

	ws.JobStates[jobID].StepStates = append(ws.JobStates[jobID].StepStates, stepState)
}

// GetJobState returns the state of a job
func (ws *WorkflowState) GetJobState(jobID string) *JobState {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.JobStates[jobID]
}

// MarkCompleted marks the workflow as completed
func (ws *WorkflowState) MarkCompleted() {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.Status = "completed"
}

// MarkFailed marks the workflow as failed
func (ws *WorkflowState) MarkFailed() {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.Status = "failed"
}

// MarkCancelled marks the workflow as cancelled
func (ws *WorkflowState) MarkCancelled() {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.Status = "cancelled"
}
