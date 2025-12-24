package dsl

// ErrorClassifier 错误分类器
type ErrorClassifier struct {
	nonRetryableErrors map[string]bool
}

// NewErrorClassifier 创建错误分类器
func NewErrorClassifier() *ErrorClassifier {
	classifier := &ErrorClassifier{
		nonRetryableErrors: make(map[string]bool),
	}

	// 初始化永久性错误类型
	classifier.registerNonRetryableErrors()

	return classifier
}

// registerNonRetryableErrors 注册永久性错误类型
func (c *ErrorClassifier) registerNonRetryableErrors() {
	nonRetryable := []string{
		"validation_error",    // YAML 解析错误
		"schema_error",        // Schema 验证错误
		"not_found",           // 404 资源不存在
		"permission_denied",   // 403 权限拒绝
		"invalid_argument",    // 400 参数错误
		"node_not_registered", // 节点未注册
		"syntax_error",        // 语法错误
		"type_error",          // 类型错误
		"configuration_error", // 配置错误
	}

	for _, errType := range nonRetryable {
		c.nonRetryableErrors[errType] = true
	}
}

// IsRetryable 判断错误是否可重试
func (c *ErrorClassifier) IsRetryable(errType string) bool {
	// 如果在永久性错误列表中，则不可重试
	if c.nonRetryableErrors[errType] {
		return false
	}

	// 默认所有其他错误都可重试
	return true
}

// ClassifyError 分类错误并返回错误类型
// 临时性错误: network_timeout, connection_refused, service_unavailable, internal_error, deadline_exceeded
// 永久性错误: validation_error, schema_error, not_found, permission_denied, invalid_argument
// 使用更精确的匹配策略,避免误判
func (c *ErrorClassifier) ClassifyError(err error) string {
	if err == nil {
		return ""
	}

	errMsg := toLower(err.Error())

	// 优先匹配永久性错误(更具体的模式)
	if matchExact(errMsg, []string{"schema error", "invalid schema", "schema validation", "schema"}) {
		return "schema_error"
	}
	if matchExact(errMsg, []string{"validation error", "validation failed", "yaml parse", "unmarshal error", "parse error"}) {
		return "validation_error"
	}
	if matchExact(errMsg, []string{"not found", "404", "no such", "does not exist"}) {
		return "not_found"
	}
	if matchExact(errMsg, []string{"permission denied", "403", "forbidden", "access denied"}) {
		return "permission_denied"
	}
	if matchExact(errMsg, []string{"invalid argument", "400", "bad request"}) {
		return "invalid_argument"
	}
	if matchExact(errMsg, []string{"node not registered", "unknown node"}) {
		return "node_not_registered"
	}

	// 临时性错误
	if matchExact(errMsg, []string{"context deadline exceeded", "deadline exceeded", "timed out", "timeout"}) {
		return "deadline_exceeded"
	}
	if matchExact(errMsg, []string{"connection refused", "dial tcp", "dial"}) {
		return "connection_refused"
	}
	if matchExact(errMsg, []string{"503", "service unavailable", "unavailable"}) {
		return "service_unavailable"
	}
	if matchExact(errMsg, []string{"500", "internal server error", "internal"}) {
		return "internal_error"
	}

	return "unknown_error"
}

// matchExact 精确匹配错误消息中的关键词
func matchExact(msg string, patterns []string) bool {
	for _, pattern := range patterns {
		if indexOf(msg, pattern) >= 0 {
			return true
		}
	}
	return false
}

// toLower 转换为小写 (简化版)
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			result[i] = s[i] + 32
		} else {
			result[i] = s[i]
		}
	}
	return string(result)
}

// indexOf 查找子字符串位置
func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
