package agent

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestPluginManager_LoadPlugins_NoDirectory(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	pm := NewPluginManager("/nonexistent/path", logger)

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
	pm := NewPluginManager(tmpDir, logger)

	err = pm.LoadPlugins()
	if err != nil {
		t.Errorf("LoadPlugins failed: %v", err)
	}

	if len(pm.plugins) != 0 {
		t.Errorf("Expected 0 plugins, got %d", len(pm.plugins))
	}
}

func TestPluginManager_LoadPlugins_WithPlugins(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "waterflow-plugins-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create dummy .so files
	plugins := []string{"plugin1.so", "plugin2.so", "plugin3.so"}
	for _, plugin := range plugins {
		file := filepath.Join(tmpDir, plugin)
		if err := os.WriteFile(file, []byte("dummy plugin content"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	logger, _ := zap.NewDevelopment()
	pm := NewPluginManager(tmpDir, logger)

	err = pm.LoadPlugins()
	if err != nil {
		t.Errorf("LoadPlugins failed: %v", err)
	}

	if len(pm.plugins) != 3 {
		t.Errorf("Expected 3 plugins, got %d", len(pm.plugins))
	}

	// Verify plugin paths are stored
	for _, plugin := range plugins {
		if _, exists := pm.plugins[plugin]; !exists {
			t.Errorf("Plugin %s not found in loaded plugins", plugin)
		}
	}
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

	// Create valid .so file
	validPlugin := filepath.Join(tmpDir, "valid.so")
	if err := os.WriteFile(validPlugin, []byte("valid content"), 0600); err != nil {
		t.Fatal(err)
	}

	logger, _ := zap.NewDevelopment()
	pm := NewPluginManager(tmpDir, logger)

	err = pm.LoadPlugins()
	if err != nil {
		t.Errorf("LoadPlugins failed: %v", err)
	}

	// Empty file should be skipped, only valid plugin loaded
	if len(pm.plugins) != 1 {
		t.Errorf("Expected 1 plugin (empty skipped), got %d", len(pm.plugins))
	}

	if _, exists := pm.plugins["valid.so"]; !exists {
		t.Error("Valid plugin not found")
	}

	if _, exists := pm.plugins["empty.so"]; exists {
		t.Error("Empty plugin should be skipped")
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
	pm := NewPluginManager(tmpDir, logger)

	err = pm.LoadPlugins()
	if err != nil {
		t.Errorf("LoadPlugins failed: %v", err)
	}

	// Only .so file should be loaded
	if len(pm.plugins) != 1 {
		t.Errorf("Expected 1 plugin (.so only), got %d", len(pm.plugins))
	}

	if _, exists := pm.plugins["plugin.so"]; !exists {
		t.Error("Plugin .so file not found")
	}
}
