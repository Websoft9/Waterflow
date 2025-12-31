package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"go.uber.org/zap"
)

func TestPluginManager_LoadPlugins_NoDirectory(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	registry := node.NewRegistry()
	pm := NewPluginManager("/nonexistent/path", registry, logger)

	err := pm.LoadPlugins()
	if err != nil {
		t.Errorf("LoadPlugins should not fail when directory doesn't exist, got: %v", err)
	}
}

func TestPluginManager_LoadPlugins_EmptyDirectory(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "waterflow-plugins-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	logger, _ := zap.NewDevelopment()
	registry := node.NewRegistry()
	pm := NewPluginManager(tmpDir, registry, logger)

	err = pm.LoadPlugins()
	if err != nil {
		t.Errorf("LoadPlugins failed: %v", err)
	}

	// Registry should be empty
	if len(registry.List()) != 0 {
		t.Errorf("Expected 0 plugins, got %d", len(registry.List()))
	}
}

func TestPluginManager_LoadPlugins_WithPlugins(t *testing.T) {
	// This test is now covered by TestPluginManager_LoadPlugins_WithRegistry_Success
	// Skip for now as it requires mock loader to properly test
	t.Skip("Replaced by mock-based tests")
}

func TestPluginManager_LoadPlugins_EmptyFile(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "waterflow-plugins-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create empty .so file
	emptyPlugin := filepath.Join(tmpDir, "empty.so")
	if err := os.WriteFile(emptyPlugin, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	// Create valid .so file (will fail to load without proper mock)
	validPlugin := filepath.Join(tmpDir, "valid.so")
	if err := os.WriteFile(validPlugin, []byte("valid content"), 0600); err != nil {
		t.Fatal(err)
	}

	logger, _ := zap.NewDevelopment()
	registry := node.NewRegistry()
	pm := NewPluginManager(tmpDir, registry, logger)

	err = pm.LoadPlugins()
	if err != nil {
		t.Errorf("LoadPlugins failed: %v", err)
	}

	// Both files should fail to load (no proper .so), but empty one should be caught earlier
	// Registry should be empty
	if len(registry.List()) != 0 {
		t.Errorf("Expected 0 plugins, got %d", len(registry.List()))
	}
}

func TestPluginManager_LoadPlugins_MixedFiles(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "waterflow-plugins-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create .so files
	soFile := filepath.Join(tmpDir, "plugin.so")
	if err := os.WriteFile(soFile, []byte("plugin"), 0600); err != nil {
		t.Fatal(err)
	}

	// Create non-.so files (should be ignored)
	txtFile := filepath.Join(tmpDir, "readme.txt")
	if err := os.WriteFile(txtFile, []byte("readme"), 0600); err != nil {
		t.Fatal(err)
	}

	goFile := filepath.Join(tmpDir, "source.go")
	if err := os.WriteFile(goFile, []byte("package main"), 0600); err != nil {
		t.Fatal(err)
	}

	logger, _ := zap.NewDevelopment()
	registry := node.NewRegistry()
	pm := NewPluginManager(tmpDir, registry, logger)

	err = pm.LoadPlugins()
	if err != nil {
		t.Errorf("LoadPlugins failed: %v", err)
	}

	// Only .so file should be attempted (will fail without mock)
	// Non-.so files should not be scanned
	// Registry should be empty as real plugin loading will fail
	if len(registry.List()) != 0 {
		t.Errorf("Expected 0 plugins, got %d", len(registry.List()))
	}
}

// Mock implementations for testing

// mockNode implements node.Node interface for testing
type mockNode struct {
	name    string
	version string
}

func (m *mockNode) Name() string    { return m.name }
func (m *mockNode) Version() string { return m.version }
func (m *mockNode) Params() map[string]node.ParamSpec {
	return map[string]node.ParamSpec{
		"test": {Type: "string", Required: true},
	}
}
func (m *mockNode) Execute(_ context.Context, _ map[string]interface{}) (*node.NodeResult, error) {
	return node.NewNodeResult(), nil
}
func (m *mockNode) Metadata() node.NodeMetadata {
	return node.NodeMetadata{
		Category:     "flow",
		Description:  "Mock node",
		InputSchema:  map[string]node.ParamSpec{},
		OutputSchema: map[string]interface{}{},
	}
}

// mockPluginLoader implements node.PluginLoader for testing
type mockPluginLoader struct {
	openFunc   func(path string) (*plugin.Plugin, error)
	lookupFunc func(p *plugin.Plugin, symbol string) (plugin.Symbol, error)
}

func (m *mockPluginLoader) Open(path string) (*plugin.Plugin, error) {
	if m.openFunc != nil {
		return m.openFunc(path)
	}
	return nil, fmt.Errorf("mock not configured")
}

func (m *mockPluginLoader) Lookup(p *plugin.Plugin, symbol string) (plugin.Symbol, error) {
	if m.lookupFunc != nil {
		return m.lookupFunc(p, symbol)
	}
	return nil, fmt.Errorf("mock not configured")
}

// Test cases for upgraded PluginManager with dynamic loading

func TestPluginManager_LoadPlugins_WithRegistry_Success(t *testing.T) {
	// Create temporary directory with dummy .so file
	tmpDir, err := os.MkdirTemp("", "waterflow-plugins-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	pluginPath := filepath.Join(tmpDir, "test.so")
	if err := os.WriteFile(pluginPath, []byte("dummy"), 0600); err != nil {
		t.Fatal(err)
	}

	// Setup mock loader that returns a valid node
	testNode := &mockNode{name: "test/mock", version: "v1"}
	mockLoader := &mockPluginLoader{
		openFunc: func(path string) (*plugin.Plugin, error) {
			return &plugin.Plugin{}, nil // Return dummy plugin
		},
		lookupFunc: func(p *plugin.Plugin, symbol string) (plugin.Symbol, error) {
			// Return a Register function that returns our test node
			registerFunc := func() node.Node {
				return testNode
			}
			return plugin.Symbol(registerFunc), nil
		},
	}

	logger, _ := zap.NewDevelopment()
	registry := node.NewRegistry()
	pm := NewPluginManagerWithLoader(tmpDir, registry, mockLoader, logger)

	// Should now succeed with proper mock
	err = pm.LoadPlugins()
	if err != nil {
		t.Errorf("LoadPlugins failed: %v", err)
	}

	// Node should be registered
	if len(registry.List()) != 1 {
		t.Errorf("Expected 1 node registered, got %d", len(registry.List()))
	}

	// Verify node can be retrieved
	retrievedNode, err := pm.GetNode("test/mock@v1")
	if err != nil {
		t.Errorf("GetNode failed: %v", err)
	}
	if retrievedNode.Name() != "test/mock" {
		t.Errorf("Expected node name 'test/mock', got '%s'", retrievedNode.Name())
	}
}

func TestPluginManager_LoadPlugins_PluginOpenError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "waterflow-plugins-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	pluginPath := filepath.Join(tmpDir, "bad.so")
	if err := os.WriteFile(pluginPath, []byte("dummy"), 0600); err != nil {
		t.Fatal(err)
	}

	// Mock loader that fails to open plugin
	mockLoader := &mockPluginLoader{
		openFunc: func(path string) (*plugin.Plugin, error) {
			return nil, fmt.Errorf("plugin version mismatch")
		},
	}

	logger, _ := zap.NewDevelopment()
	registry := node.NewRegistry()
	pm := NewPluginManagerWithLoader(tmpDir, registry, mockLoader, logger)

	// Should not return error (non-fatal), but log warning
	err = pm.LoadPlugins()
	if err != nil {
		t.Errorf("LoadPlugins should not fail on plugin open error, got: %v", err)
	}

	// Registry should be empty
	if len(registry.List()) != 0 {
		t.Errorf("Expected 0 registered nodes, got %d", len(registry.List()))
	}
}

func TestPluginManager_LoadPlugins_RegisterFunctionNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "waterflow-plugins-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	pluginPath := filepath.Join(tmpDir, "noreg.so")
	if err := os.WriteFile(pluginPath, []byte("dummy"), 0600); err != nil {
		t.Fatal(err)
	}

	// Mock loader that can open but has no Register function
	mockLoader := &mockPluginLoader{
		openFunc: func(path string) (*plugin.Plugin, error) {
			return &plugin.Plugin{}, nil
		},
		lookupFunc: func(p *plugin.Plugin, symbol string) (plugin.Symbol, error) {
			return nil, fmt.Errorf("symbol not found: %s", symbol)
		},
	}

	logger, _ := zap.NewDevelopment()
	registry := node.NewRegistry()
	pm := NewPluginManagerWithLoader(tmpDir, registry, mockLoader, logger)

	err = pm.LoadPlugins()
	if err != nil {
		t.Errorf("LoadPlugins should not fail on missing Register function, got: %v", err)
	}

	// Registry should be empty
	if len(registry.List()) != 0 {
		t.Errorf("Expected 0 registered nodes, got %d", len(registry.List()))
	}
}

func TestPluginManager_GetNode_Success(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	registry := node.NewRegistry()

	// Register a test node
	testNode := &mockNode{name: "test/mock", version: "v1"}
	if err := registry.Register(testNode); err != nil {
		t.Fatal(err)
	}

	pm := NewPluginManager("/tmp", registry, logger)

	// GetNode should now work
	retrievedNode, err := pm.GetNode("test/mock@v1")
	if err != nil {
		t.Errorf("GetNode failed: %v", err)
	}

	if retrievedNode.Name() != "test/mock" {
		t.Errorf("Expected node name 'test/mock', got '%s'", retrievedNode.Name())
	}
}
