package node

import (
	"fmt"
	"sync"
)

// Node 节点接口
type Node interface {
	Name() string
	Version() string
	Params() map[string]ParamSpec
}

// ParamSpec 参数规范
type ParamSpec struct {
	Type        string
	Required    bool
	Description string
	Default     interface{}
	Pattern     string
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
