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

	// Matrix related (Story 1.6)
	IsMatrix        bool
	MatrixInstances []*MatrixInstanceState

	// Non-Matrix job
	StepStates []*StepState
}

// MatrixInstanceState tracks the state of a Matrix instance
type MatrixInstanceState struct {
	MatrixID   string                 // deploy-0, deploy-1, etc.
	Matrix     map[string]interface{} // {server: web1, env: prod}
	Status     string                 // pending, running, completed, cancelled
	Conclusion string                 // success, failure, cancelled
	StepStates []*StepState
	Error      string
}

// StepState tracks the state of a step
type StepState struct {
	StepID      string
	Name        string
	Status      string // pending, running, completed, skipped
	Conclusion  string // success, failure, skipped, timeout (Story 1.7)
	Outputs     map[string]string
	Error       string
	ContinuedOn bool

	// Retry and timeout information (Story 1.7)
	Attempts        int    // 尝试次数 (包括首次执行)
	TimeoutMinutes  int    // 超时配置 (分钟)
	DurationSeconds int    // 实际执行时长 (秒)
	IsTimeout       bool   // 是否超时
	ErrorType       string // 错误类型 (用于重试决策)
	Retryable       bool   // 是否可重试
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

// UpdateMatrixInstanceState updates the state of a Matrix instance (Story 1.6)
func (ws *WorkflowState) UpdateMatrixInstanceState(jobID, matrixID string, matrix map[string]interface{}, status, conclusion string) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.JobStates[jobID] == nil {
		ws.JobStates[jobID] = &JobState{
			JobID:           jobID,
			IsMatrix:        true,
			MatrixInstances: make([]*MatrixInstanceState, 0),
		}
	}

	// 查找或创建 Matrix 实例
	var instanceState *MatrixInstanceState
	for _, inst := range ws.JobStates[jobID].MatrixInstances {
		if inst.MatrixID == matrixID {
			instanceState = inst
			break
		}
	}

	if instanceState == nil {
		instanceState = &MatrixInstanceState{
			MatrixID:   matrixID,
			Matrix:     matrix,
			StepStates: make([]*StepState, 0),
		}
		ws.JobStates[jobID].MatrixInstances = append(ws.JobStates[jobID].MatrixInstances, instanceState)
	}

	instanceState.Status = status
	instanceState.Conclusion = conclusion
}

// AddMatrixInstanceStepState adds a step state to a Matrix instance (Story 1.6)
func (ws *WorkflowState) AddMatrixInstanceStepState(jobID, matrixID string, stepState *StepState) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.JobStates[jobID] == nil {
		return
	}

	for _, inst := range ws.JobStates[jobID].MatrixInstances {
		if inst.MatrixID == matrixID {
			inst.StepStates = append(inst.StepStates, stepState)
			break
		}
	}
}

// GetMatrixInstanceState returns the state of a Matrix instance (Story 1.6)
func (ws *WorkflowState) GetMatrixInstanceState(jobID, matrixID string) *MatrixInstanceState {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	jobState := ws.JobStates[jobID]
	if jobState == nil {
		return nil
	}

	for _, inst := range jobState.MatrixInstances {
		if inst.MatrixID == matrixID {
			return inst
		}
	}

	return nil
}
