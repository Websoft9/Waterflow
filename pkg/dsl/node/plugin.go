package node

// RegisterFunc is the function signature that plugins must export.
// Plugins should implement a function named "Register" with this signature:
//
//	func Register() node.Node {
//	    return &MyNode{}
//	}
//
// The plugin loader will look up this symbol and call it to instantiate the node.
type RegisterFunc func() Node

// Plugin is an optional marker interface for enhanced type safety.
// Nodes implementing this interface can provide additional plugin metadata.
type Plugin interface {
	Node
	// PluginInfo returns additional plugin-specific information.
	PluginInfo() map[string]string
}
