package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNodeHandlers_ListNodes(t *testing.T) {
	logger := zap.NewNop()
	h := NewNodeHandlers(logger)

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
		req := httptest.NewRequest(http.MethodGet, "/v1/nodes?search=shell", nil)
		w := httptest.NewRecorder()

		h.ListNodes(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		nodes := response["nodes"].([]interface{})
		assert.Greater(t, len(nodes), 0)
		// At least one should contain "shell" in name
		found := false
		for _, n := range nodes {
			node := n.(map[string]interface{})
			if name, ok := node["name"].(string); ok {
				if contains(name, "shell") {
					found = true
					break
				}
			}
		}
		assert.True(t, found, "Expected at least one node containing 'shell' in name")
	})

	t.Run("filter by category and search", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/nodes?category=docker&search=compose", nil)
		w := httptest.NewRecorder()

		h.ListNodes(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		nodes := response["nodes"].([]interface{})
		for _, n := range nodes {
			node := n.(map[string]interface{})
			assert.Equal(t, "docker", node["category"])
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
	h := NewNodeHandlers(logger)

	t.Run("get existing node by name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/nodes/exec/shell", nil)
		w := httptest.NewRecorder()

		// Need to set mux vars
		req = mux.SetURLVars(req, map[string]string{"name": "exec/shell"})

		h.GetNode(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "exec/shell", response["name"])
		assert.Equal(t, "v1", response["version"])
		assert.Equal(t, "exec", response["category"])
		assert.NotNil(t, response["input_schema"])
		assert.NotNil(t, response["output_schema"])
	})

	t.Run("get existing node with version", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/nodes/exec/shell@v1", nil)
		w := httptest.NewRecorder()

		req = mux.SetURLVars(req, map[string]string{"name": "exec/shell@v1"})

		h.GetNode(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "exec/shell", response["name"])
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

func TestGetKnownNodes(t *testing.T) {
	nodes := getKnownNodes()

	// Verify we have expected nodes
	assert.Greater(t, len(nodes), 0)

	// Each node should have required fields
	for _, node := range nodes {
		assert.NotEmpty(t, node["name"], "Node should have name")
		assert.NotEmpty(t, node["version"], "Node should have version")
		assert.NotEmpty(t, node["category"], "Node should have category")
		assert.NotEmpty(t, node["description"], "Node should have description")
		assert.NotNil(t, node["input_schema"], "Node should have input_schema")
	}
}

func TestFilterNodeList(t *testing.T) {
	nodes := getKnownNodes()

	t.Run("no filter", func(t *testing.T) {
		filtered := filterNodeList(nodes, "", "")
		assert.Equal(t, len(nodes), len(filtered))
	})

	t.Run("filter by category", func(t *testing.T) {
		filtered := filterNodeList(nodes, "exec", "")
		for _, node := range filtered {
			assert.Equal(t, "exec", node["category"])
		}
	})

	t.Run("filter by search", func(t *testing.T) {
		filtered := filterNodeList(nodes, "", "http")
		assert.Greater(t, len(filtered), 0)
	})

	t.Run("filter by both", func(t *testing.T) {
		filtered := filterNodeList(nodes, "docker", "exec")
		for _, node := range filtered {
			assert.Equal(t, "docker", node["category"])
		}
	})

	t.Run("search case insensitive", func(t *testing.T) {
		filtered1 := filterNodeList(nodes, "", "SHELL")
		filtered2 := filterNodeList(nodes, "", "shell")
		assert.Equal(t, len(filtered1), len(filtered2))
	})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
