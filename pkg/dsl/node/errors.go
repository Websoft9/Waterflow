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

// NodeAlreadyRegisteredError indicates a node with the same name@version is already registered.
type NodeAlreadyRegisteredError struct {
	NodeKey string
}

func (e *NodeAlreadyRegisteredError) Error() string {
	return fmt.Sprintf("node already registered: %s", e.NodeKey)
}

// NodeNotFoundError indicates the requested node was not found in the registry.
type NodeNotFoundError struct {
	NodeType string
}

func (e *NodeNotFoundError) Error() string {
	return fmt.Sprintf("node not found: %s", e.NodeType)
}

// InputValidationError indicates parameter validation failed (可能包含多个错误).
type InputValidationError struct {
	NodeName string           // 节点名称 (如 "exec/shell@v1")
	Errors   []ParameterError // 所有验证失败项
}

func (e *InputValidationError) Error() string {
	if len(e.Errors) == 1 {
		return fmt.Sprintf("parameter validation failed for %s: %s",
			e.NodeName, e.Errors[0].Message)
	}
	return fmt.Sprintf("parameter validation failed for %s: %d errors",
		e.NodeName, len(e.Errors))
}

// NonRetryable marks this error as non-retryable for Temporal.
func (e *InputValidationError) NonRetryable() bool {
	return true // 永久性错误
}

// ParameterError 表示单个参数的验证错误.
type ParameterError struct {
	ParamName string      // 参数名称
	ErrorType string      // 错误类型: Missing, TypeMismatch, PatternMismatch, EnumViolation, RangeViolation
	Expected  interface{} // 期望值 (类型/Pattern/Enum 列表/范围)
	Actual    interface{} // 实际值
	Message   string      // 人类可读的错误信息
}

// NonRetryableError 表示永久性错误（不应重试）。
// Story 4.3: 用于标记不应重试的错误类型。
type NonRetryableError struct {
	OriginalError error
	Message       string
	ErrorType     string // validation_error, schema_error, not_found, permission_denied, etc.
}

func (e *NonRetryableError) Error() string {
	return e.Message
}

func (e *NonRetryableError) Unwrap() error {
	return e.OriginalError
}
