package node

// NodeMetadata provides comprehensive information about a node.
// It includes description, category, and schemas for inputs and outputs.
type NodeMetadata struct {
	// Description provides a human-readable explanation of what the node does.
	Description string

	// Category classifies the node into a functional group.
	// Valid categories: exec, docker, http, file, flow
	Category string

	// InputSchema defines the parameters this node accepts.
	// Keys are parameter names, values are their specifications.
	InputSchema map[string]ParamSpec

	// OutputSchema defines the structure of data this node produces.
	// Keys are output names, values describe their types and formats.
	OutputSchema map[string]interface{}
}

// NewNodeMetadata creates a new NodeMetadata with initialized maps.
func NewNodeMetadata(description, category string) *NodeMetadata {
	return &NodeMetadata{
		Description:  description,
		Category:     category,
		InputSchema:  make(map[string]ParamSpec),
		OutputSchema: make(map[string]interface{}),
	}
}

// AddInputParam adds an input parameter to the schema.
func (m *NodeMetadata) AddInputParam(name string, spec ParamSpec) {
	m.InputSchema[name] = spec
}

// AddOutputParam adds an output parameter to the schema.
func (m *NodeMetadata) AddOutputParam(name string, schema interface{}) {
	m.OutputSchema[name] = schema
}
