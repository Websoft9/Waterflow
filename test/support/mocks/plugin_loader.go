package mocks

import (
	"context"
	"fmt"
	"sync"

	"github.com/Websoft9/waterflow/pkg/dsl/node"
)

// PluginLoader is a mock plugin loader for testing.
type PluginLoader struct {
	mu        sync.Mutex
	nodes     map[string]node.Node
	loadErr   error
	loadCalls int
}

// NewPluginLoader creates a new mock plugin loader.
func NewPluginLoader() *PluginLoader {
	return &PluginLoader{
		nodes: make(map[string]node.Node),
	}
}

// RegisterNode registers a mock node.
func (p *PluginLoader) RegisterNode(name string, n node.Node) *PluginLoader {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.nodes[name] = n
	return p
}

// SetLoadError sets an error to be returned on Load.
func (p *PluginLoader) SetLoadError(err error) *PluginLoader {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.loadErr = err
	return p
}

// Load simulates loading plugins.
func (p *PluginLoader) Load() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.loadCalls++
	return p.loadErr
}

// GetNode returns a registered mock node.
func (p *PluginLoader) GetNode(name string) (node.Node, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if n, ok := p.nodes[name]; ok {
		return n, nil
	}
	return nil, fmt.Errorf("node %q not found", name)
}

// LoadCalls returns the number of times Load was called.
func (p *PluginLoader) LoadCalls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.loadCalls
}

// MockNode is a simple mock node implementation.
type MockNode struct {
	name        string
	version     string
	executeFunc func(ctx context.Context, params map[string]interface{}) (*node.NodeResult, error)
	validateErr error
}

// NewMockNode creates a new mock node.
func NewMockNode(name, version string) *MockNode {
	return &MockNode{
		name:    name,
		version: version,
	}
}

// WithExecuteFunc sets a custom execute function.
func (n *MockNode) WithExecuteFunc(fn func(ctx context.Context, params map[string]interface{}) (*node.NodeResult, error)) *MockNode {
	n.executeFunc = fn
	return n
}

// WithValidateError sets an error to be returned on Validate.
func (n *MockNode) WithValidateError(err error) *MockNode {
	n.validateErr = err
	return n
}

// Name returns the node name.
func (n *MockNode) Name() string {
	return n.name
}

// Version returns the node version.
func (n *MockNode) Version() string {
	return n.version
}

// Params returns the parameter specifications.
func (n *MockNode) Params() map[string]node.ParamSpec {
	return map[string]node.ParamSpec{}
}

// Execute executes the mock node.
func (n *MockNode) Execute(ctx context.Context, params map[string]interface{}) (*node.NodeResult, error) {
	if n.executeFunc != nil {
		return n.executeFunc(ctx, params)
	}
	result := node.NewNodeResult()
	result.SetOutput("status", "success")
	return result, nil
}

// Metadata returns the node metadata.
func (n *MockNode) Metadata() node.NodeMetadata {
	return node.NodeMetadata{
		Description: fmt.Sprintf("Mock node: %s", n.name),
		Category:    "mock",
	}
}
