//go:build integration
// +build integration

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"plugin"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/websoft9/waterflow/pkg/dsl/node"
)

// ensurePluginBuilt checks if plugin exists, if not, builds it
func ensurePluginBuilt(t *testing.T) {
	if _, err := os.Stat("request.so"); os.IsNotExist(err) {
		t.Log("Plugin not found, building...")
		cmd := exec.Command("make", "build")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to build plugin: %v\nOutput: %s", err, output)
		}
		t.Log("Plugin built successfully")
	}
}

// TestHTTPPlugin_Load tests loading the plugin
func TestHTTPPlugin_Load(t *testing.T) {
	ensurePluginBuilt(t)

	p, err := plugin.Open("request.so")
	require.NoError(t, err, "Failed to open plugin")

	symRegister, err := p.Lookup("Register")
	require.NoError(t, err, "Failed to lookup Register function")

	register, ok := symRegister.(func() node.Node)
	require.True(t, ok, "Register is not of correct type")

	httpNode := register()
	require.NotNil(t, httpNode)

	assert.Equal(t, "http/request", httpNode.Name())
	assert.Equal(t, "v1", httpNode.Version())
}

// TestHTTPPlugin_Execute tests executing the plugin
func TestHTTPPlugin_Execute(t *testing.T) {
	ensurePluginBuilt(t)

	p, err := plugin.Open("request.so")
	require.NoError(t, err)

	symRegister, err := p.Lookup("Register")
	require.NoError(t, err)

	register := symRegister.(func() node.Node)
	httpNode := register()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	inputs := map[string]interface{}{
		"url": server.URL,
	}

	result, err := httpNode.Execute(context.Background(), inputs)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 200, result.Outputs["status_code"])
}
