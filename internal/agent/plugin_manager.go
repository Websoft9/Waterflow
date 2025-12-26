package agent

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

// Node represents a workflow node plugin interface.
// Full implementation will be in Epic 4 (Story 4.1).
type Node interface {
	Execute(ctx interface{}, params map[string]interface{}) (interface{}, error)
}

// PluginManager manages node plugins (.so files).
// Story 2.9: Basic plugin scanning and validation
// Epic 4: Full plugin loading with reflection
type PluginManager struct {
	pluginDir string
	logger    *zap.Logger
	plugins   map[string]string // nodeType -> plugin path
}

// NewPluginManager creates a new plugin manager.
func NewPluginManager(dir string, logger *zap.Logger) *PluginManager {
	return &PluginManager{
		pluginDir: dir,
		logger:    logger,
		plugins:   make(map[string]string),
	}
}

// LoadPlugins scans and validates .so plugins from plugin directory.
// Story 2.9: Scan plugins for Docker volume mounting support
// Epic 4: Full dynamic loading implementation
func (pm *PluginManager) LoadPlugins() error {
	// Check if plugin directory exists
	if _, err := os.Stat(pm.pluginDir); os.IsNotExist(err) {
		pm.logger.Info("Plugin directory not found, skipping plugin loading",
			zap.String("dir", pm.pluginDir))
		return nil
	}

	// Scan for .so files
	files, err := filepath.Glob(filepath.Join(pm.pluginDir, "*.so"))
	if err != nil {
		return fmt.Errorf("failed to scan plugins: %w", err)
	}

	if len(files) == 0 {
		pm.logger.Info("No plugins found in directory",
			zap.String("dir", pm.pluginDir))
		return nil
	}

	pm.logger.Info("Scanning plugins", zap.Int("count", len(files)))

	// Validate each plugin file
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			pm.logger.Warn("Failed to stat plugin file",
				zap.String("file", file),
				zap.Error(err))
			continue
		}

		// Basic validation: file exists and has reasonable size
		if info.Size() == 0 {
			pm.logger.Warn("Plugin file is empty, skipping",
				zap.String("file", file))
			continue
		}

		// TODO Epic 4: Load plugin with plugin.Open() and extract Node interface
		pm.logger.Info("Plugin file found (loading deferred to Epic 4)",
			zap.String("file", filepath.Base(file)),
			zap.Int64("size_bytes", info.Size()))

		// Store plugin path for future loading
		pm.plugins[filepath.Base(file)] = file
	}

	pm.logger.Info("Plugin scan complete",
		zap.Int("valid_plugins", len(pm.plugins)))

	return nil
}

// GetNode retrieves a node plugin by type.
// Story 2.1: Stub - returns error (plugins not loaded).
func (pm *PluginManager) GetNode(nodeType string) (Node, error) {
	return nil, fmt.Errorf("plugins not loaded yet (Epic 4)")
}
