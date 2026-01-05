package sdk

import (
	"time"
)

// SubmitWorkflowRequest contains parameters for submitting a workflow.
//
// The YAML field is required and should contain valid Waterflow workflow definition.
// The Vars field is optional and can be used to override workflow variables.
//
// Example:
//
//	req := &SubmitWorkflowRequest{
//	    YAML: yamlContent,
//	    Vars: map[string]interface{}{
//	        "environment": "production",
//	        "replicas": 3,
//	    },
//	}
type SubmitWorkflowRequest struct {
	YAML string                 `json:"yaml"` // Workflow YAML content
	Vars map[string]interface{} `json:"vars,omitempty"`
}

// SubmitWorkflowResponse contains submission result
type SubmitWorkflowResponse struct {
	ID        string    `json:"id"`
	RunID     string    `json:"run_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	URL       string    `json:"url"`
}

// WorkflowStatus represents the detailed execution status of a workflow.
//
// The Jobs field is an array of JobStatus and should be iterated using range:
//
//	for _, job := range status.Jobs {
//	    fmt.Printf("Job %s: %s\n", job.Name, job.Status)
//	    for _, step := range job.Steps {
//	        fmt.Printf("  Step %s: %s\n", step.Name, step.Status)
//	    }
//	}
//
// DurationSeconds is a pointer type and will be nil if the workflow hasn't completed.
type WorkflowStatus struct {
	ID              string                 `json:"id"`
	RunID           string                 `json:"run_id"`
	Name            string                 `json:"name"`
	Status          string                 `json:"status"`
	CreatedAt       string                 `json:"created_at"`
	StartedAt       string                 `json:"started_at,omitempty"`
	CompletedAt     string                 `json:"completed_at,omitempty"`
	DurationSeconds *int                   `json:"duration_seconds,omitempty"` // Nullable, nil if not completed
	Vars            map[string]interface{} `json:"vars,omitempty"`
	Jobs            []JobStatus            `json:"jobs,omitempty"` // Array of job statuses, not a map
	Error           string                 `json:"error,omitempty"`
}

// JobStatus represents the execution status of a single job within a workflow.
//
// The Steps field is an array of StepStatus and should be iterated using range:
//
//	for _, step := range job.Steps {
//	    fmt.Printf("Step %s: %s\n", step.Name, step.Status)
//	}
type JobStatus struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Status      string       `json:"status"`
	StartedAt   string       `json:"started_at,omitempty"`
	CompletedAt string       `json:"completed_at,omitempty"`
	RunsOn      string       `json:"runs_on,omitempty"`
	Steps       []StepStatus `json:"steps,omitempty"` // Array of step statuses, not a map
}

// StepStatus represents the execution status of a single step within a job.
//
// The Conclusion field is only set when Status is "completed" and indicates
// the outcome: "success", "failure", "cancelled", or "timeout".
type StepStatus struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Conclusion  string `json:"conclusion,omitempty"` // Only set when Status is "completed": success/failure/cancelled/timeout
	StartedAt   string `json:"started_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
}

// ListWorkflowsRequest contains list query parameters
type ListWorkflowsRequest struct {
	Page   int    // Page number (default 1)
	Limit  int    // Items per page (default 20)
	Status string // Status filter
	Name   string // Name search
}

// ListWorkflowsResponse contains paginated workflow list
type ListWorkflowsResponse struct {
	Workflows  []WorkflowSummary `json:"workflows"`
	Pagination PaginationInfo    `json:"pagination"`
}

// WorkflowSummary represents a workflow in list view
type WorkflowSummary struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	DurationSeconds *int       `json:"duration_seconds,omitempty"`
}

// PaginationInfo contains pagination metadata
type PaginationInfo struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// GetLogsRequest contains parameters for retrieving workflow logs.
//
// Example:
//
//	req := &GetLogsRequest{
//	    WorkflowID: "wf-abc123",
//	    Level:      "error,warn",
//	    Job:        "deploy",
//	    Tail:       100,
//	}
type GetLogsRequest struct {
	WorkflowID string // Workflow ID to retrieve logs for (required)
	Level      string // Filter by level (e.g., "error,warn")
	Job        string // Filter by job name
	Step       string // Filter by step name
	Tail       int    // Only return last N lines (0 = all logs)
}

// RerunWorkflowRequest contains parameters for rerunning a workflow.
//
// The Vars field allows overriding variables from the original workflow execution.
//
// Example:
//
//	req := &RerunWorkflowRequest{
//	    WorkflowID: originalWorkflowID,
//	    Vars: map[string]interface{}{
//	        "environment": "staging",
//	        "timeout":     600,
//	    },
//	}
type RerunWorkflowRequest struct {
	WorkflowID string                 // Original workflow ID (required)
	Vars       map[string]interface{} // Variable overrides (optional)
}

// LogEntry represents a single log entry from workflow execution.
//
// Logs are returned in chronological order. The Job and Step fields
// are populated when the log is associated with a specific job or step.
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Job       string `json:"job,omitempty"`
	Step      string `json:"step,omitempty"`
	Message   string `json:"message"`
	Error     string `json:"error,omitempty"`
}
