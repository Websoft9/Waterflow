package node

import (
	"fmt"
	"sync"
)

// ParamSpec defines the specification for a node parameter.
// It supports JSON Schema-style validation with type checking,
// required fields, defaults, regex patterns, enums, and numeric ranges.
//
// Story 1.3: Base fields (Type, Required, Description, Default, Pattern)
// Story 3.1: Extended with Enum, MinValue, MaxValue for enhanced validation
type ParamSpec struct {
	// Type specifies the parameter data type.
	// Valid values: "string", "int", "float", "bool", "object", "array"
	Type string

	// Required indicates whether this parameter must be provided.
	Required bool

	// Description provides human-readable documentation for this parameter.
	Description string

	// Default specifies the value to use if parameter is not provided.
	Default interface{}

	// Pattern is a regular expression that string values must match.
	Pattern string

	// Enum restricts values to a specific set of allowed options.
	// Example: []interface{}{"debug", "info", "warn", "error"}
	// Added in Story 3.1
	Enum []interface{}

	// MinValue specifies the minimum allowed value for numeric types.
	// Only applies when Type is "int" or "float".
	// Added in Story 3.1
	MinValue *float64

	// MaxValue specifies the maximum allowed value for numeric types.
	// Only applies when Type is "int" or "float".
	// Added in Story 3.1
	MaxValue *float64
}

// Registry 节点注册表
type Registry struct {
	mu    sync.RWMutex
	nodes map[string]Node
}

// NewRegistry 创建注册表
func NewRegistry() *Registry {
	return &Registry{
		nodes: make(map[string]Node),
	}
}

// Register 注册节点
func (r *Registry) Register(node Node) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s@%s", node.Name(), node.Version())
	if _, exists := r.nodes[key]; exists {
		return fmt.Errorf("node %s already registered", key)
	}

	r.nodes[key] = node
	return nil
}

// Get 获取节点
func (r *Registry) Get(name string) (Node, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	node, exists := r.nodes[name]
	if !exists {
		return nil, fmt.Errorf("node %s not found", name)
	}

	return node, nil
}

// List 列出所有节点
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.nodes))
	for name := range r.nodes {
		names = append(names, name)
	}
	return names
}
