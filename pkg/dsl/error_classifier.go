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
func (c *ErrorClassifier) ClassifyError(err error) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()

	// 简单的字符串匹配分类 (实际项目中可能使用更复杂的错误类型检测)
	// 优先检查更具体的错误类型
	switch {
	case contains(errMsg, "schema", "invalid schema"):
		return "schema_error"
	case contains(errMsg, "validation", "parse", "unmarshal"):
		return "validation_error"
	case contains(errMsg, "not found", "404"):
		return "not_found"
	case contains(errMsg, "permission denied", "403", "forbidden"):
		return "permission_denied"
	case contains(errMsg, "invalid argument", "400", "bad request"):
		return "invalid_argument"
	case contains(errMsg, "node not registered", "unknown node"):
		return "node_not_registered"
	case contains(errMsg, "timeout", "timed out", "deadline exceeded", "context deadline"):
		return "deadline_exceeded"
	case contains(errMsg, "connection refused", "dial"):
		return "connection_refused"
	case contains(errMsg, "503", "service unavailable"):
		return "service_unavailable"
	case contains(errMsg, "500", "internal"):
		return "internal_error"
	default:
		return "unknown_error"
	}
}

// contains 检查错误消息是否包含任一关键词
func contains(msg string, keywords ...string) bool {
	msgLower := toLower(msg)
	for _, keyword := range keywords {
		if indexOf(msgLower, toLower(keyword)) >= 0 {
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
