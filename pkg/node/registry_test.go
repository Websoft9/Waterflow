package node

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockNode 测试用节点
type MockNode struct {
	name    string
	version string
	params  map[string]ParamSpec
}

func (n *MockNode) Name() string    { return n.name }
func (n *MockNode) Version() string { return n.version }
func (n *MockNode) Params() map[string]ParamSpec {
	return n.params
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.nodes)
	assert.Empty(t, registry.List())
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	node := &MockNode{
		name:    "test",
		version: "v1",
		params:  map[string]ParamSpec{},
	}

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

	node := &MockNode{
		name:    "duplicate",
		version: "v1",
		params:  map[string]ParamSpec{},
	}

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

	node := &MockNode{
		name:    "checkout",
		version: "v1",
		params: map[string]ParamSpec{
			"repository": {Type: "string", Required: true},
		},
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
		{name: "checkout", version: "v1", params: map[string]ParamSpec{}},
		{name: "run", version: "v1", params: map[string]ParamSpec{}},
		{name: "notify", version: "v2", params: map[string]ParamSpec{}},
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
	baseNode := &MockNode{name: "base", version: "v1", params: map[string]ParamSpec{}}
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
			node := &MockNode{
				name:    "node",
				version: "v" + string(rune(idx)),
				params:  map[string]ParamSpec{},
			}
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
	v1 := &MockNode{name: "checkout", version: "v1", params: map[string]ParamSpec{}}
	v2 := &MockNode{name: "checkout", version: "v2", params: map[string]ParamSpec{}}

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
