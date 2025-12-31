package node

import "plugin"

// PluginLoader abstracts plugin loading for testability.
// This interface allows mocking plugin.Open() and Lookup() in tests,
// avoiding the need for real .so files and enabling error scenario simulation.
type PluginLoader interface {
	// Open loads a plugin from the given path.
	// Returns the loaded plugin or an error if loading fails.
	Open(path string) (*plugin.Plugin, error)

	// Lookup finds a symbol in the plugin.
	// Returns the symbol or an error if not found.
	Lookup(p *plugin.Plugin, symbolName string) (plugin.Symbol, error)
}

// RealPluginLoader implements PluginLoader using Go's plugin package.
// This is the production implementation used by the PluginManager.
type RealPluginLoader struct{}

// NewRealPluginLoader creates a new RealPluginLoader instance.
func NewRealPluginLoader() *RealPluginLoader {
	return &RealPluginLoader{}
}

// Open loads a plugin using the standard plugin.Open function.
func (l *RealPluginLoader) Open(path string) (*plugin.Plugin, error) {
	return plugin.Open(path)
}

// Lookup finds a symbol in the plugin using the Plugin.Lookup method.
func (l *RealPluginLoader) Lookup(p *plugin.Plugin, symbolName string) (plugin.Symbol, error) {
	return p.Lookup(symbolName)
}
