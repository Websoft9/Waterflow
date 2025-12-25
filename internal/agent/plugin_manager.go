package agent

import (
	"fmt"

	"go.uber.org/zap"
)

// Node represents a workflow node plugin interface.
// Full implementation will be in Epic 4 (Story 4.1).
type Node interface {
	Execute(ctx interface{}, params map[string]interface{}) (interface{}, error)
}

// PluginManager manages node plugins (.so files).
// Story 2.1: Stub implementation - full loading in Epic 4.
type PluginManager struct {
	pluginDir string
	logger    *zap.Logger
}

// NewPluginManager creates a new plugin manager.
func NewPluginManager(dir string, logger *zap.Logger) *PluginManager {
	return &PluginManager{
		pluginDir: dir,
		logger:    logger,
	}
}

// LoadPlugins loads all .so plugins from plugin directory.
// Story 2.1: Stub - returns nil (no plugins loaded yet).
func (pm *PluginManager) LoadPlugins() error {
	pm.logger.Info("Plugin loading not yet implemented (Epic 4)")
	return nil
}

// GetNode retrieves a node plugin by type.
// Story 2.1: Stub - returns error (plugins not loaded).
func (pm *PluginManager) GetNode(nodeType string) (Node, error) {
	return nil, fmt.Errorf("plugins not loaded yet (Epic 4)")
}
