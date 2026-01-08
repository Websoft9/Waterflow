// Package logger provides structured logging functionality using zap.
package logger

import (
	"strings"

	"go.uber.org/zap/zapcore"
)

var (
	// DefaultSensitiveFields 默认敏感字段列表
	DefaultSensitiveFields = []string{
		"password",
		"token",
		"api_key",
		"secret",
		"private_key",
		"access_token",
		"refresh_token",
		"authorization",
		"cookie",
	}
)

// SanitizingCore 包装 zapcore.Core,在添加字段时自动脱敏敏感字段
type SanitizingCore struct {
	zapcore.Core
	sensitiveFields map[string]bool
}

// NewSanitizingCore 创建脱敏 Core
func NewSanitizingCore(core zapcore.Core, sensitiveFields []string) zapcore.Core {
	fieldMap := make(map[string]bool)
	for _, field := range sensitiveFields {
		fieldMap[strings.ToLower(field)] = true
	}

	return &SanitizingCore{
		Core:            core,
		sensitiveFields: fieldMap,
	}
}

// isSensitive 检查字段是否敏感 (不区分大小写)
func (c *SanitizingCore) isSensitive(key string) bool {
	return c.sensitiveFields[strings.ToLower(key)]
}

// With 添加字段,创建新的 Core
func (c *SanitizingCore) With(fields []zapcore.Field) zapcore.Core {
	sanitizedFields := c.sanitizeFields(fields)
	return &SanitizingCore{
		Core:            c.Core.With(sanitizedFields),
		sensitiveFields: c.sensitiveFields,
	}
}

// Check 检查是否启用指定级别
func (c *SanitizingCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

// Write 写入日志条目,脱敏字段
func (c *SanitizingCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	sanitizedFields := c.sanitizeFields(fields)
	return c.Core.Write(entry, sanitizedFields)
}

// sanitizeFields 脱敏字段数组 (性能优化版)
func (c *SanitizingCore) sanitizeFields(fields []zapcore.Field) []zapcore.Field {
	// 快速检查：如果没有敏感字段，直接返回原数组
	hasSensitive := false
	for i := range fields {
		if c.isSensitive(fields[i].Key) {
			hasSensitive = true
			break
		}
	}

	if !hasSensitive {
		return fields
	}

	// 有敏感字段时才创建新数组
	sanitized := make([]zapcore.Field, len(fields))
	for i, field := range fields {
		if c.isSensitive(field.Key) {
			sanitized[i] = zapcore.Field{
				Key:    field.Key,
				Type:   zapcore.StringType,
				String: "***REDACTED***",
			}
		} else {
			sanitized[i] = field
		}
	}
	return sanitized
}
