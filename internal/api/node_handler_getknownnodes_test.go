package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"net/http"

	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/Websoft9/waterflow/pkg/dsl/node/builtin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestNodeHandlers_GetKnownNodes_DynamicLoading tests that getKnownNodes uses registry
func TestNodeHandlers_GetKnownNodes_DynamicLoading(t *testing.T) {
	logger := zap.NewNop()
	registry := node.NewRegistry()

	// Register builtin nodes
	err := registry.Register(&builtin.CheckoutNode{})
	require.NoError(t, err)
	err = registry.Register(&builtin.RunNode{})
	require.NoError(t, err)

	handlers := NewNodeHandlers(logger, registry)

	// Make HTTP request
	req := httptest.NewRequest(http.MethodGet, "/v1/nodes", nil)
	w := httptest.NewRecorder()

	handlers.ListNodes(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	nodes := response["nodes"].([]interface{})
	assert.Equal(t, 2, len(nodes), "Should return 2 registered nodes")

	// Verify node structure matches expected format
	firstNode := nodes[0].(map[string]interface{})
	assert.Contains(t, firstNode, "name", "Node should have name field")
	assert.Contains(t, firstNode, "category", "Node should have category field")
	assert.Contains(t, firstNode, "description", "Node should have description field")
	assert.Contains(t, firstNode, "input_schema", "Node should have input_schema field")
	assert.Contains(t, firstNode, "output_schema", "Node should have output_schema field")

	// Verify name format is "name@version"
	name := firstNode["name"].(string)
	assert.Contains(t, name, "@", "Node name should include version (name@version)")
}

// TestNodeHandlers_GetKnownNodes_EmptyRegistry tests behavior with empty registry
func TestNodeHandlers_GetKnownNodes_EmptyRegistry(t *testing.T) {
	logger := zap.NewNop()
	registry := node.NewRegistry() // Empty registry

	handlers := NewNodeHandlers(logger, registry)

	req := httptest.NewRequest(http.MethodGet, "/v1/nodes", nil)
	w := httptest.NewRecorder()

	handlers.ListNodes(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Should not crash with empty registry")

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	nodes := response["nodes"].([]interface{})
	assert.Equal(t, 0, len(nodes), "Empty registry should return empty array")
	assert.Equal(t, float64(0), response["total"], "Total should be 0")
}

// TestNodeHandlers_GetKnownNodes_ResponseFormat tests backward compatibility of response format
func TestNodeHandlers_GetKnownNodes_ResponseFormat(t *testing.T) {
	logger := zap.NewNop()
	registry := node.NewRegistry()

	// Register a node
	err := registry.Register(&builtin.CheckoutNode{})
	require.NoError(t, err)

	handlers := NewNodeHandlers(logger, registry)

	req := httptest.NewRequest(http.MethodGet, "/v1/nodes", nil)
	w := httptest.NewRecorder()

	handlers.ListNodes(w, req)

	var response map[string]interface{}
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	// Verify top-level structure
	assert.Contains(t, response, "nodes")
	assert.Contains(t, response, "total")

	nodes := response["nodes"].([]interface{})
	assert.Greater(t, len(nodes), 0)

	// Verify each node has required fields (backward compatibility)
	for _, n := range nodes {
		node := n.(map[string]interface{})
		assert.NotEmpty(t, node["name"], "Node must have name")
		assert.NotEmpty(t, node["category"], "Node must have category")
		assert.NotEmpty(t, node["description"], "Node must have description")
		assert.NotNil(t, node["input_schema"], "Node must have input_schema")
		assert.NotNil(t, node["output_schema"], "Node must have output_schema")
	}
}
