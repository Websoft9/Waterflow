package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

// PluginManager manages node plugins (.so files).
// Story 2.9: Basic plugin scanning and validation
// Story 3.1: Uses unified pkg/dsl/node.Node interface
// Story 4.1: Full plugin loading with dynamic registration
type PluginManager struct {
	pluginDir   string
	registry    *node.Registry
	loader      node.PluginLoader
	logger      *zap.Logger
	timersMutex sync.Mutex // Protects debounceTimers map
}

// NewPluginManager creates a new plugin manager with real plugin loader.
func NewPluginManager(dir string, registry *node.Registry, logger *zap.Logger) *PluginManager {
	return &PluginManager{
		pluginDir: dir,
		registry:  registry,
		loader:    node.NewRealPluginLoader(),
		logger:    logger,
	}
}

// NewPluginManagerWithLoader creates a new plugin manager with custom loader (for testing).
func NewPluginManagerWithLoader(dir string, registry *node.Registry, loader node.PluginLoader, logger *zap.Logger) *PluginManager {
	return &PluginManager{
		pluginDir: dir,
		registry:  registry,
		loader:    loader,
		logger:    logger,
	}
}

// LoadPlugins scans and loads all .so plugins from plugin directory.
// Story 2.9: Scan plugins for Docker volume mounting support
// Story 4.1: Full dynamic loading with node registration
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

	pm.logger.Info("Loading plugins", zap.Int("count", len(files)))

	successCount := 0
	failureCount := 0

	// Load each plugin file
	for _, file := range files {
		if err := pm.LoadPlugin(file); err != nil {
			pm.logger.Warn("Failed to load plugin",
				zap.String("file", filepath.Base(file)),
				zap.Error(err))
			failureCount++
			continue
		}
		successCount++
	}

	pm.logger.Info("Plugin loading complete",
		zap.Int("success", successCount),
		zap.Int("failure", failureCount),
		zap.Int("total", len(files)))

	return nil
}

// LoadPlugin loads a single plugin file and registers the node.
// This method supports both initial loading and hot-reloading.
func (pm *PluginManager) LoadPlugin(path string) error {
	// Validate file exists and has content
	info, err := os.Stat(path)
	if err != nil {
		return &node.PluginNotFoundError{Path: path}
	}

	if info.Size() == 0 {
		return &node.PluginLoadError{
			Path: path,
			Err:  fmt.Errorf("plugin file is empty"),
		}
	}

	// Open the plugin
	p, err := pm.loader.Open(path)
	if err != nil {
		return &node.PluginLoadError{
			Path: path,
			Err:  err,
		}
	}

	// Lookup the Register function
	symbol, err := pm.loader.Lookup(p, "Register")
	if err != nil {
		return &node.RegisterFunctionNotFoundError{
			PluginPath: path,
		}
	}

	// Type assert to Register function signature
	registerFunc, ok := symbol.(func() node.Node)
	if !ok {
		return &node.RegisterFunctionSignatureError{
			PluginPath: path,
			Expected:   "func() node.Node",
			Actual:     fmt.Sprintf("%T", symbol),
		}
	}

	// Call Register to get the node instance
	nodeInstance := registerFunc()

	// Validate the node
	if err := node.ValidateNode(nodeInstance); err != nil {
		return &node.InvalidNodeError{
			PluginPath: path,
			Reason:     err.Error(),
		}
	}

	// Register the node in the registry
	if err := pm.registry.Register(nodeInstance); err != nil {
		// If node already registered, update it (hot-reload)
		var alreadyRegistered *node.NodeAlreadyRegisteredError
		if errors.As(err, &alreadyRegistered) {
			if updateErr := pm.registry.Update(nodeInstance); updateErr != nil {
				return fmt.Errorf("failed to update node during hot-reload: %w", updateErr)
			}
			pm.logger.Info("Plugin hot-reloaded (updated)",
				zap.String("plugin", filepath.Base(path)),
				zap.String("node", nodeInstance.Name()),
				zap.String("version", nodeInstance.Version()))
			return nil
		}
		return err
	}

	pm.logger.Info("Plugin loaded successfully",
		zap.String("plugin", filepath.Base(path)),
		zap.String("node", nodeInstance.Name()),
		zap.String("version", nodeInstance.Version()))

	return nil
}

// GetNode retrieves a node plugin by type.
// Story 2.1: Stub - returns error (plugins not loaded).
// Story 3.1: Returns pkg/dsl/node.Node interface
// Story 4.1: Delegates to NodeRegistry
func (pm *PluginManager) GetNode(nodeType string) (node.Node, error) {
	return pm.registry.Get(nodeType)
}

// WatchPlugins monitors the plugin directory for changes and hot-reloads plugins.
// This method blocks until the context is cancelled.
// It uses fsnotify to detect CREATE and WRITE events on .so files.
func (pm *PluginManager) WatchPlugins(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			pm.logger.Error("Failed to close watcher", zap.Error(err))
		}
	}()

	if err := watcher.Add(pm.pluginDir); err != nil {
		return fmt.Errorf("failed to watch directory %s: %w", pm.pluginDir, err)
	}

	pm.logger.Info("Started watching plugin directory for hot-reload",
		zap.String("dir", pm.pluginDir))

	// Debounce timer map to avoid multiple reloads for the same file
	debounceTimers := make(map[string]*time.Timer)

	for {
		select {
		case <-ctx.Done():
			pm.logger.Info("Stopped watching plugin directory")
			return nil

		case event := <-watcher.Events:
			// Only process .so files
			if !strings.HasSuffix(event.Name, ".so") {
				continue
			}

			// Only handle CREATE and WRITE events
			if event.Op&fsnotify.Create == fsnotify.Create ||
				event.Op&fsnotify.Write == fsnotify.Write {

				// Cancel previous debounce timer if exists
				pm.timersMutex.Lock()
				if timer, exists := debounceTimers[event.Name]; exists {
					timer.Stop()
				}

				// Set new debounce timer (500ms delay)
				debounceTimers[event.Name] = time.AfterFunc(500*time.Millisecond, func() {
					pm.logger.Info("Detected plugin change, hot-reloading",
						zap.String("file", filepath.Base(event.Name)),
						zap.String("operation", event.Op.String()))

					if err := pm.LoadPlugin(event.Name); err != nil {
						pm.logger.Warn("Failed to hot-reload plugin",
							zap.String("file", filepath.Base(event.Name)),
							zap.Error(err))
					} else {
						pm.logger.Info("Plugin hot-reloaded successfully",
							zap.String("file", filepath.Base(event.Name)))
					}

					pm.timersMutex.Lock()
					delete(debounceTimers, event.Name)
					pm.timersMutex.Unlock()
				})
				pm.timersMutex.Unlock()
			}

		case err := <-watcher.Errors:
			pm.logger.Error("Watcher error", zap.Error(err))
		}
	}
}
