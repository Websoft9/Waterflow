package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListNodes_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/nodes" {
			t.Errorf("Expected path /v1/nodes, got %s", r.URL.Path)
		}

		response := map[string]interface{}{
			"nodes": []NodeInfo{
				{
					Name:        "exec/shell",
					Version:     "v1",
					Category:    "exec",
					Description: "Execute shell commands",
					InputSchema: map[string]map[string]interface{}{
						"command": {
							"type":     "string",
							"required": true,
						},
					},
					OutputSchema: map[string]interface{}{
						"stdout": "string",
					},
				},
			},
			"total": 1,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := New(server.URL, "", 30*1000000000, false)

	nodes, err := client.ListNodes("", "")
	if err != nil {
		t.Fatalf("ListNodes() failed: %v", err)
	}

	if len(nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(nodes))
	}

	if nodes[0].Name != "exec/shell" {
		t.Errorf("Expected node name 'exec/shell', got '%s'", nodes[0].Name)
	}
}

func TestListNodes_WithCategory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameter
		category := r.URL.Query().Get("category")
		if category != "exec" {
			t.Errorf("Expected category=exec, got category=%s", category)
		}

		response := map[string]interface{}{
			"nodes": []NodeInfo{
				{
					Name:     "exec/shell",
					Version:  "v1",
					Category: "exec",
				},
			},
			"total": 1,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := New(server.URL, "", 30*1000000000, false)

	nodes, err := client.ListNodes("exec", "")
	if err != nil {
		t.Fatalf("ListNodes() with category failed: %v", err)
	}

	if len(nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(nodes))
	}
}

func TestListNodes_WithSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameter
		search := r.URL.Query().Get("search")
		if search != "shell" {
			t.Errorf("Expected search=shell, got search=%s", search)
		}

		response := map[string]interface{}{
			"nodes": []NodeInfo{
				{
					Name:    "exec/shell",
					Version: "v1",
				},
			},
			"total": 1,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := New(server.URL, "", 30*1000000000, false)

	nodes, err := client.ListNodes("", "shell")
	if err != nil {
		t.Fatalf("ListNodes() with search failed: %v", err)
	}

	if len(nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(nodes))
	}
}

func TestListNodes_WithBothFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		category := r.URL.Query().Get("category")
		search := r.URL.Query().Get("search")

		if category != "exec" || search != "shell" {
			t.Errorf("Expected category=exec&search=shell, got category=%s&search=%s", category, search)
		}

		response := map[string]interface{}{
			"nodes": []NodeInfo{},
			"total": 0,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := New(server.URL, "", 30*1000000000, false)

	nodes, err := client.ListNodes("exec", "shell")
	if err != nil {
		t.Fatalf("ListNodes() with both filters failed: %v", err)
	}

	if len(nodes) != 0 {
		t.Errorf("Expected 0 nodes, got %d", len(nodes))
	}
}

func TestListNodes_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "internal_error",
				"message": "Server error",
			},
		})
	}))
	defer server.Close()

	client := New(server.URL, "", 30*1000000000, false)

	_, err := client.ListNodes("", "")
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if serverErr, ok := err.(*ServerError); ok {
		if serverErr.StatusCode != 500 {
			t.Errorf("Expected status code 500, got %d", serverErr.StatusCode)
		}
	} else {
		t.Error("Expected ServerError type")
	}
}

func TestListNodes_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"nodes": []NodeInfo{},
			"total": 0,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := New(server.URL, "", 30*1000000000, false)

	nodes, err := client.ListNodes("", "")
	if err != nil {
		t.Fatalf("ListNodes() with empty response failed: %v", err)
	}

	if len(nodes) != 0 {
		t.Errorf("Expected 0 nodes, got %d", len(nodes))
	}
}
