package dsl

import (
	"errors"
	"fmt"
)

// 预定义错误类型,用于测试和基准测试

// ErrValidation 创建验证错误
func ErrValidation(msg string) error {
	return fmt.Errorf("validation_error: %s", msg)
}

// ErrSchema 创建Schema错误
func ErrSchema(msg string) error {
	return fmt.Errorf("schema_error: %s", msg)
}

// ErrNotFound 创建资源不存在错误
func ErrNotFound(msg string) error {
	return fmt.Errorf("not_found: %s", msg)
}

// ErrPermissionDenied 创建权限拒绝错误
func ErrPermissionDenied(msg string) error {
	return fmt.Errorf("permission_denied: %s", msg)
}

// ErrInvalidArgument 创建参数错误
func ErrInvalidArgument(msg string) error {
	return fmt.Errorf("invalid_argument: %s", msg)
}

// ErrNetworkTimeout 创建网络超时错误
func ErrNetworkTimeout(msg string) error {
	return fmt.Errorf("network timeout: %s", msg)
}

// ErrConnectionRefused 创建连接拒绝错误
func ErrConnectionRefused(msg string) error {
	return fmt.Errorf("connection refused: %s", msg)
}

// ErrServiceUnavailable 创建服务不可用错误
func ErrServiceUnavailable(msg string) error {
	return errors.New("503 service unavailable: " + msg)
}
