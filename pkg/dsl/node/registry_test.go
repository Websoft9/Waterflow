package node

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a simple mock node for testing
func createMockNode(name, version string) *MockNode {
	return &MockNode{
		NameValue:    name,
		VersionValue: version,
		ParamsValue:  map[string]ParamSpec{},
		MetadataValue: NodeMetadata{
			Description:  "Test node",
			Category:     "flow",
			InputSchema:  map[string]ParamSpec{},
			OutputSchema: map[string]interface{}{},
		},
		ExecuteFunc: func(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error) {
			return NewNodeResult(), nil
		},
	}
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.nodes)
	assert.Empty(t, registry.List())
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	node := createMockNode("test", "v1")

	// 成功注册
	err := registry.Register(node)
	assert.NoError(t, err)

	// 验证已注册
	got, err := registry.Get("test@v1")
	assert.NoError(t, err)
	assert.Equal(t, node, got)
}

func TestRegistry_Register_Duplicate(t *testing.T) {
	registry := NewRegistry()

	node := createMockNode("duplicate", "v1")

	// 首次注册成功
	err := registry.Register(node)
	require.NoError(t, err)

	// 重复注册失败
	err = registry.Register(node)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	node := createMockNode("checkout", "v1")
	node.ParamsValue = map[string]ParamSpec{
		"repository": {Type: "string", Required: true},
	}

	_ = registry.Register(node)

	// 成功获取
	got, err := registry.Get("checkout@v1")
	assert.NoError(t, err)
	assert.Equal(t, "checkout", got.Name())
	assert.Equal(t, "v1", got.Version())

	// 不存在的节点
	_, err = registry.Get("nonexistent@v1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	// 空注册表
	assert.Empty(t, registry.List())

	// 注册多个节点
	nodes := []*MockNode{
		createMockNode("checkout", "v1"),
		createMockNode("run", "v1"),
		createMockNode("notify", "v2"),
	}

	for _, node := range nodes {
		_ = registry.Register(node)
	}

	// 验证列表
	list := registry.List()
	assert.Len(t, list, 3)
	assert.Contains(t, list, "checkout@v1")
	assert.Contains(t, list, "run@v1")
	assert.Contains(t, list, "notify@v2")
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewRegistry()

	// 注册初始节点
	baseNode := createMockNode("base", "v1")
	_ = registry.Register(baseNode)

	var wg sync.WaitGroup
	concurrency := 100

	// 并发读
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			_, _ = registry.Get("base@v1")
			_ = registry.List()
		}()
	}

	// 并发写（不同节点）
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		idx := i
		go func() {
			defer wg.Done()
			node := createMockNode("node", "v"+string(rune(idx)))
			_ = registry.Register(node)
		}()
	}

	wg.Wait()

	// 验证基础节点仍然可访问
	got, err := registry.Get("base@v1")
	assert.NoError(t, err)
	assert.Equal(t, "base", got.Name())
}

func TestRegistry_MultipleVersions(t *testing.T) {
	registry := NewRegistry()

	// 注册同名节点的不同版本
	v1 := createMockNode("checkout", "v1")
	v2 := createMockNode("checkout", "v2")

	err := registry.Register(v1)
	require.NoError(t, err)

	err = registry.Register(v2)
	require.NoError(t, err)

	// 验证两个版本都可访问
	gotV1, err := registry.Get("checkout@v1")
	assert.NoError(t, err)
	assert.Equal(t, "v1", gotV1.Version())

	gotV2, err := registry.Get("checkout@v2")
	assert.NoError(t, err)
	assert.Equal(t, "v2", gotV2.Version())

	// 列表包含两个版本
	list := registry.List()
	assert.Len(t, list, 2)
	assert.Contains(t, list, "checkout@v1")
	assert.Contains(t, list, "checkout@v2")
}

// Test custom error types
func TestRegistry_ErrorTypes(t *testing.T) {
	registry := NewRegistry()

	node := createMockNode("test", "v1")
	_ = registry.Register(node)

	// Test NodeAlreadyRegisteredError
	err := registry.Register(node)
	assert.Error(t, err)
	var alreadyRegistered *NodeAlreadyRegisteredError
	assert.ErrorAs(t, err, &alreadyRegistered)
	if alreadyRegistered != nil {
		assert.Equal(t, "test@v1", alreadyRegistered.NodeKey)
	}

	// Test NodeNotFoundError
	_, err = registry.Get("nonexistent@v1")
	assert.Error(t, err)
	var notFound *NodeNotFoundError
	assert.ErrorAs(t, err, &notFound)
	if notFound != nil {
		assert.Equal(t, "nonexistent@v1", notFound.NodeType)
	}
}

// Test Update method for hot-reload
func TestRegistry_Update(t *testing.T) {
	registry := NewRegistry()

	// Register initial node
	node1 := createMockNode("test", "v1")
	err := registry.Register(node1)
	require.NoError(t, err)

	// Update the node with a new instance
	node2 := createMockNode("test", "v1")
	node2.MetadataValue.Description = "Updated description"
	err = registry.Update(node2)
	assert.NoError(t, err)

	// Verify the node was updated
	retrieved, err := registry.Get("test@v1")
	require.NoError(t, err)
	assert.Equal(t, "Updated description", retrieved.Metadata().Description)

	// Test updating non-existent node
	nonExistentNode := createMockNode("nonexistent", "v1")
	err = registry.Update(nonExistentNode)
	assert.Error(t, err)
	var notFound *NodeNotFoundError
	assert.ErrorAs(t, err, &notFound)
}
