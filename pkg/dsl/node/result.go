package node

import "time"

// NodeResult represents the structured return value from a node execution.
// It contains outputs, logs, execution duration, and optional metadata.
type NodeResult struct {
	// Outputs contains the data produced by the node execution.
	// Keys are output parameter names, values are their corresponding data.
	Outputs map[string]interface{}

	// Logs contains execution log messages generated during node execution.
	Logs []string

	// Duration records how long the node took to execute.
	Duration time.Duration

	// Metadata contains additional information about the execution,
	// such as resource usage, debug info, or custom metrics.
	Metadata map[string]interface{}
}

// NewNodeResult creates a new NodeResult with initialized maps.
func NewNodeResult() *NodeResult {
	return &NodeResult{
		Outputs:  make(map[string]interface{}),
		Logs:     make([]string, 0),
		Metadata: make(map[string]interface{}),
	}
}

// AddLog appends a log message to the result.
func (r *NodeResult) AddLog(msg string) {
	r.Logs = append(r.Logs, msg)
}

// SetOutput sets an output parameter value.
func (r *NodeResult) SetOutput(key string, value interface{}) {
	r.Outputs[key] = value
}

// SetMetadata sets a metadata value.
func (r *NodeResult) SetMetadata(key string, value interface{}) {
	r.Metadata[key] = value
}
