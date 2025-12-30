//go:build integration
// +build integration

package main

import (
	"context"
	"os"
	"os/exec"
	"plugin"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/websoft9/waterflow/pkg/dsl/node"
)

// ensurePluginBuilt checks if plugin exists, if not, builds it
func ensurePluginBuilt(t *testing.T) {
	if _, err := os.Stat("sleep.so"); os.IsNotExist(err) {
		t.Log("Plugin not found, building...")
		cmd := exec.Command("make", "build")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to build plugin: %v\nOutput: %s", err, output)
		}
		t.Log("Plugin built successfully")
	}
}

// TestSleepPlugin_Load tests loading the plugin as a .so file
func TestSleepPlugin_Load(t *testing.T) {
	// Ensure plugin is built before testing
	ensurePluginBuilt(t)

	// Load the plugin
	p, err := plugin.Open("sleep.so")
	require.NoError(t, err, "Failed to open plugin")

	// Look up the Register symbol
	symRegister, err := p.Lookup("Register")
	require.NoError(t, err, "Failed to lookup Register function")

	// Cast to the correct function type
	register, ok := symRegister.(func() node.Node)
	require.True(t, ok, "Register is not of type func() node.Node")

	// Call Register to get the node
	sleepNode := register()
	require.NotNil(t, sleepNode)

	// Verify node properties
	assert.Equal(t, "flow/sleep", sleepNode.Name())
	assert.Equal(t, "v1", sleepNode.Version())
}

// TestSleepPlugin_Execute tests executing the plugin node
func TestSleepPlugin_Execute(t *testing.T) {
	// Ensure plugin is built
	ensurePluginBuilt(t)

	// Load the plugin
	p, err := plugin.Open("sleep.so")
	require.NoError(t, err)

	symRegister, err := p.Lookup("Register")
	require.NoError(t, err)

	register := symRegister.(func() node.Node)
	sleepNode := register()

	// Test execution
	inputs := map[string]interface{}{
		"duration": "1s",
	}

	start := time.Now()
	result, err := sleepNode.Execute(context.Background(), inputs)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Outputs["completed"].(bool))
	assert.InDelta(t, 1000, elapsed.Milliseconds(), 50)
}

// TestSleepPlugin_MultipleDurations tests various durations
func TestSleepPlugin_MultipleDurations(t *testing.T) {
	// Ensure plugin is built
	ensurePluginBuilt(t)

	// Load the plugin
	p, err := plugin.Open("sleep.so")
	require.NoError(t, err)

	symRegister, err := p.Lookup("Register")
	require.NoError(t, err)

	register := symRegister.(func() node.Node)
	sleepNode := register()

	tests := []struct {
		duration string
		minMS    int64
		maxMS    int64
	}{
		{"1s", 980, 1100},
		{"2s", 1980, 2100},
	}

	for _, tt := range tests {
		t.Run(tt.duration, func(t *testing.T) {
			inputs := map[string]interface{}{
				"duration": tt.duration,
			}

			start := time.Now()
			result, err := sleepNode.Execute(context.Background(), inputs)
			elapsed := time.Since(start)

			require.NoError(t, err)
			assert.True(t, result.Outputs["completed"].(bool))
			assert.GreaterOrEqual(t, elapsed.Milliseconds(), tt.minMS)
			assert.LessOrEqual(t, elapsed.Milliseconds(), tt.maxMS)
		})
	}
}

// TestSleepPlugin_Cancellation tests context cancellation
func TestSleepPlugin_Cancellation(t *testing.T) {
	// Ensure plugin is built
	ensurePluginBuilt(t)

	// Load the plugin
	p, err := plugin.Open("sleep.so")
	require.NoError(t, err)

	symRegister, err := p.Lookup("Register")
	require.NoError(t, err)

	register := symRegister.(func() node.Node)
	sleepNode := register()

	// Test cancellation
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	inputs := map[string]interface{}{
		"duration": "10s",
	}

	start := time.Now()
	result, err := sleepNode.Execute(ctx, inputs)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.False(t, result.Outputs["completed"].(bool))
	assert.Less(t, elapsed.Milliseconds(), int64(200))
}
