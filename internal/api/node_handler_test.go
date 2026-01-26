package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/Websoft9/waterflow/pkg/dsl/node/builtin"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNodeHandlers_ListNodes(t *testing.T) {
	logger := zap.NewNop()
	registry := node.NewRegistry()
	// Register some test nodes
	_ = registry.Register(&builtin.CheckoutNode{})
	_ = registry.Register(&builtin.RunNode{})
	h := NewNodeHandlers(logger, registry)

	t.Run("list all nodes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/nodes", nil)
		w := httptest.NewRecorder()

		h.ListNodes(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.NotNil(t, response["nodes"])
		assert.NotNil(t, response["total"])

		nodes := response["nodes"].([]interface{})
		assert.Greater(t, len(nodes), 0)
	})

	t.Run("filter by category", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/nodes?category=exec", nil)
		w := httptest.NewRecorder()

		h.ListNodes(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		nodes := response["nodes"].([]interface{})
		for _, n := range nodes {
			node := n.(map[string]interface{})
			assert.Equal(t, "exec", node["category"])
		}
	})

	t.Run("filter by search term", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/nodes?search=checkout", nil)
		w := httptest.NewRecorder()

		h.ListNodes(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		nodes := response["nodes"].([]interface{})
		assert.Greater(t, len(nodes), 0)
		// At least one should contain "checkout" in name
		found := false
		for _, n := range nodes {
			node := n.(map[string]interface{})
			if name, ok := node["name"].(string); ok {
				if strings.Contains(name, "checkout") {
					found = true
					break
				}
			}
		}
		assert.True(t, found, "Expected at least one node containing 'checkout' in name")
	})

	t.Run("filter by category and search", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/nodes?category=exec&search=run", nil)
		w := httptest.NewRecorder()

		h.ListNodes(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		nodes := response["nodes"].([]interface{})
		for _, n := range nodes {
			node := n.(map[string]interface{})
			assert.Equal(t, "exec", node["category"])
		}
	})

	t.Run("no results for invalid category", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/nodes?category=nonexistent", nil)
		w := httptest.NewRecorder()

		h.ListNodes(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		nodes := response["nodes"].([]interface{})
		assert.Equal(t, 0, len(nodes))
	})
}

func TestNodeHandlers_GetNode(t *testing.T) {
	logger := zap.NewNop()
	registry := node.NewRegistry()
	// Register test nodes
	_ = registry.Register(&builtin.CheckoutNode{})
	_ = registry.Register(&builtin.RunNode{})
	h := NewNodeHandlers(logger, registry)

	t.Run("get existing node by name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/nodes/checkout@v1", nil)
		w := httptest.NewRecorder()

		// Need to set mux vars
		req = mux.SetURLVars(req, map[string]string{"name": "checkout@v1"})

		h.GetNode(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "checkout@v1", response["name"])
		assert.Equal(t, "v1", response["version"])
		assert.Equal(t, "exec", response["category"])
		assert.NotNil(t, response["input_schema"])
		assert.NotNil(t, response["output_schema"])
	})

	t.Run("get existing node with version", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/nodes/run@v1", nil)
		w := httptest.NewRecorder()

		req = mux.SetURLVars(req, map[string]string{"name": "run@v1"})

		h.GetNode(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "run@v1", response["name"])
	})

	t.Run("node not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/nodes/nonexistent", nil)
		w := httptest.NewRecorder()

		req = mux.SetURLVars(req, map[string]string{"name": "nonexistent"})

		h.GetNode(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))
	})

	t.Run("empty node name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/nodes/", nil)
		w := httptest.NewRecorder()

		req = mux.SetURLVars(req, map[string]string{"name": ""})

		h.GetNode(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))
	})
}

// Removed TestGetKnownNodes and TestFilterNodeList - now covered by TestNodeHandlers_GetKnownNodes_*
