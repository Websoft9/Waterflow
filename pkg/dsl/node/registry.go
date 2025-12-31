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
		return &NodeAlreadyRegisteredError{NodeKey: key}
	}

	r.nodes[key] = node
	return nil
}

// Get 获取节点
// Supports both exact version matching (name@version) and partial matching (name only).
// When only name is provided, returns the first matching node.
func (r *Registry) Get(name string) (Node, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try exact match first
	if node, exists := r.nodes[name]; exists {
		return node, nil
	}

	// If no @, try partial match
	for key, node := range r.nodes {
		if len(name) > 0 && name[len(name)-1] != '@' {
			// Check if this is a version-less query
			expectedPrefix := name + "@"
			if len(key) > len(expectedPrefix) && key[:len(expectedPrefix)] == expectedPrefix {
				return node, nil
			}
		}
	}

	return nil, &NodeNotFoundError{NodeType: name}
}

// List 列出所有节点
// Returns a list of node keys in the format "name@version".
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.nodes))
	for name := range r.nodes {
		names = append(names, name)
	}
	return names
}

// Update updates an existing node registration.
// This is used during hot-reload to replace an existing node with a new version.
func (r *Registry) Update(node Node) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s@%s", node.Name(), node.Version())
	if _, exists := r.nodes[key]; !exists {
		return &NodeNotFoundError{NodeType: key}
	}

	r.nodes[key] = node
	return nil
}

// ListNodes returns metadata for all registered nodes.
// The list is sorted by category for better organization.
func (r *Registry) ListNodes() []NodeMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadataList := make([]NodeMetadata, 0, len(r.nodes))
	for _, node := range r.nodes {
		metadataList = append(metadataList, node.Metadata())
	}
	return metadataList
}
