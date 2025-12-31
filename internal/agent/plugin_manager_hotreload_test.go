package agent

import (
	"context"
	"os"
	"path/filepath"
	"plugin"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"go.uber.org/zap"
)

func TestPluginManager_WatchPlugins_DetectNewFile(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "waterflow-hotreload-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Setup mock loader
	loadCount := 0
	testNode := &mockNode{name: "test/hotreload", version: "v1"}
	mockLoader := &mockPluginLoader{
		openFunc: func(path string) (*plugin.Plugin, error) {
			return &plugin.Plugin{}, nil
		},
		lookupFunc: func(p *plugin.Plugin, symbol string) (plugin.Symbol, error) {
			registerFunc := func() node.Node {
				loadCount++
				return testNode
			}
			return plugin.Symbol(registerFunc), nil
		},
	}

	logger, _ := zap.NewDevelopment()
	registry := node.NewRegistry()
	pm := NewPluginManagerWithLoader(tmpDir, registry, mockLoader, logger)

	// Start watcher in background
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		_ = pm.WatchPlugins(ctx)
	}()

	// Wait for watcher to start
	time.Sleep(200 * time.Millisecond)

	// Create a new plugin file
	pluginPath := filepath.Join(tmpDir, "newplugin.so")
	if err := os.WriteFile(pluginPath, []byte("dummy content"), 0600); err != nil {
		t.Fatal(err)
	}

	// Wait for debounce + processing
	time.Sleep(800 * time.Millisecond)

	// Verify plugin was loaded
	if loadCount < 1 {
		t.Errorf("Expected plugin to be loaded at least once, got %d loads", loadCount)
	}

	// Verify node is registered
	retrievedNode, err := registry.Get("test/hotreload@v1")
	if err != nil {
		t.Errorf("Node should be registered after hot-reload: %v", err)
	}
	if retrievedNode != nil && retrievedNode.Name() != "test/hotreload" {
		t.Errorf("Expected node name 'test/hotreload', got '%s'", retrievedNode.Name())
	}
}

func TestPluginManager_WatchPlugins_DetectFileUpdate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "waterflow-hotreload-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create initial plugin file
	pluginPath := filepath.Join(tmpDir, "update.so")
	if err := os.WriteFile(pluginPath, []byte("initial content"), 0600); err != nil {
		t.Fatal(err)
	}

	loadCount := 0
	testNode := &mockNode{name: "test/update", version: "v1"}
	mockLoader := &mockPluginLoader{
		openFunc: func(path string) (*plugin.Plugin, error) {
			return &plugin.Plugin{}, nil
		},
		lookupFunc: func(p *plugin.Plugin, symbol string) (plugin.Symbol, error) {
			registerFunc := func() node.Node {
				loadCount++
				return testNode
			}
			return plugin.Symbol(registerFunc), nil
		},
	}

	logger, _ := zap.NewDevelopment()
	registry := node.NewRegistry()
	pm := NewPluginManagerWithLoader(tmpDir, registry, mockLoader, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		_ = pm.WatchPlugins(ctx)
	}()

	time.Sleep(200 * time.Millisecond)

	// Update the file
	if err := os.WriteFile(pluginPath, []byte("updated content v2"), 0600); err != nil {
		t.Fatal(err)
	}

	time.Sleep(800 * time.Millisecond)

	// Should have been loaded at least once
	if loadCount < 1 {
		t.Errorf("Expected plugin to be reloaded, got %d loads", loadCount)
	}
}

func TestPluginManager_WatchPlugins_IgnoreNonSoFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "waterflow-hotreload-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	loadCount := 0
	mockLoader := &mockPluginLoader{
		openFunc: func(path string) (*plugin.Plugin, error) {
			loadCount++
			return &plugin.Plugin{}, nil
		},
		lookupFunc: func(p *plugin.Plugin, symbol string) (plugin.Symbol, error) {
			return nil, nil
		},
	}

	logger, _ := zap.NewDevelopment()
	registry := node.NewRegistry()
	pm := NewPluginManagerWithLoader(tmpDir, registry, mockLoader, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		_ = pm.WatchPlugins(ctx)
	}()

	time.Sleep(200 * time.Millisecond)

	// Create non-.so files
	txtFile := filepath.Join(tmpDir, "readme.txt")
	if err := os.WriteFile(txtFile, []byte("readme"), 0600); err != nil {
		t.Fatal(err)
	}

	goFile := filepath.Join(tmpDir, "source.go")
	if err := os.WriteFile(goFile, []byte("package main"), 0600); err != nil {
		t.Fatal(err)
	}

	time.Sleep(800 * time.Millisecond)

	// Should not have loaded any plugins
	if loadCount > 0 {
		t.Errorf("Expected 0 plugin loads for non-.so files, got %d", loadCount)
	}
}

func TestPluginManager_WatchPlugins_ContextCancellation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "waterflow-hotreload-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	logger, _ := zap.NewDevelopment()
	registry := node.NewRegistry()
	mockLoader := &mockPluginLoader{
		openFunc:   func(path string) (*plugin.Plugin, error) { return &plugin.Plugin{}, nil },
		lookupFunc: func(p *plugin.Plugin, symbol string) (plugin.Symbol, error) { return nil, nil },
	}
	pm := NewPluginManagerWithLoader(tmpDir, registry, mockLoader, logger)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- pm.WatchPlugins(ctx)
	}()

	time.Sleep(200 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for watcher to stop
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("WatchPlugins should return nil on context cancellation, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("WatchPlugins did not stop after context cancellation")
	}
}
