package api

import (
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/Websoft9/waterflow/pkg/dsl/node/builtin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestNewNodeHandlers_WithRegistry tests NodeHandlers with a real registry
func TestNewNodeHandlers_WithRegistry(t *testing.T) {
	logger := zap.NewNop()
	registry := node.NewRegistry()
	err := registry.Register(&builtin.CheckoutNode{})
	require.NoError(t, err)

	handlers := NewNodeHandlers(logger, registry)

	assert.NotNil(t, handlers)
	assert.NotNil(t, handlers.nodeRegistry, "NodeHandlers should store registry reference")
	assert.Equal(t, registry, handlers.nodeRegistry, "Should use provided registry")
}

// TestNewNodeHandlers_WithNilRegistry tests graceful degradation with nil registry
func TestNewNodeHandlers_WithNilRegistry(t *testing.T) {
	logger := zap.NewNop()

	handlers := NewNodeHandlers(logger, nil)

	assert.NotNil(t, handlers)
	assert.NotNil(t, handlers.nodeRegistry, "Should create empty registry when nil is provided")

	// Empty registry should have no nodes
	nodes := handlers.nodeRegistry.List()
	assert.Equal(t, 0, len(nodes), "Nil registry should result in empty registry")
}

// TestNewNodeHandlers_WithEmptyRegistry tests behavior with empty registry
func TestNewNodeHandlers_WithEmptyRegistry(t *testing.T) {
	logger := zap.NewNop()
	registry := node.NewRegistry() // Empty registry

	handlers := NewNodeHandlers(logger, registry)

	assert.NotNil(t, handlers)
	assert.NotNil(t, handlers.nodeRegistry)

	nodes := handlers.nodeRegistry.List()
	assert.Equal(t, 0, len(nodes), "Empty registry should have no nodes")
}
