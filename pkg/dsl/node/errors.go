package node

import "fmt"

// PluginNotFoundError indicates that the plugin file does not exist.
type PluginNotFoundError struct {
	Path string
}

func (e *PluginNotFoundError) Error() string {
	return fmt.Sprintf("plugin not found: %s", e.Path)
}

// PluginLoadError indicates that the .so file could not be loaded.
type PluginLoadError struct {
	Path string
	Err  error
}

func (e *PluginLoadError) Error() string {
	return fmt.Sprintf("failed to load plugin %s: %v", e.Path, e.Err)
}

// PluginVersionMismatchError indicates Go version mismatch.
type PluginVersionMismatchError struct {
	Expected string
	Actual   string
}

func (e *PluginVersionMismatchError) Error() string {
	return fmt.Sprintf("Go version mismatch: expected %s, got %s", e.Expected, e.Actual)
}

// RegisterFunctionNotFoundError indicates Register function was not found in plugin.
type RegisterFunctionNotFoundError struct {
	PluginPath string
}

func (e *RegisterFunctionNotFoundError) Error() string {
	return fmt.Sprintf("Register function not found in plugin: %s", e.PluginPath)
}

// RegisterFunctionSignatureError indicates incorrect Register function signature.
type RegisterFunctionSignatureError struct {
	PluginPath string
	Expected   string
	Actual     string
}

func (e *RegisterFunctionSignatureError) Error() string {
	return fmt.Sprintf("invalid Register function signature in %s: expected %s, got %s",
		e.PluginPath, e.Expected, e.Actual)
}

// InvalidNodeError indicates the returned Node does not conform to the interface.
type InvalidNodeError struct {
	PluginPath string
	Reason     string
}

func (e *InvalidNodeError) Error() string {
	return fmt.Sprintf("invalid node from plugin %s: %s", e.PluginPath, e.Reason)
}

// NodeValidationError indicates node validation failed with details.
type NodeValidationError struct {
	NodeName string
	Errors   []string
}

func (e *NodeValidationError) Error() string {
	return fmt.Sprintf("node validation failed for %s: %v", e.NodeName, e.Errors)
}
