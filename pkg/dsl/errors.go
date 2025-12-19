package dsl

import (
	"encoding/json"
	"fmt"
)

// ValidationError 验证错误
type ValidationError struct {
	Type   string       `json:"type"`   // yaml_syntax_error, schema_validation_error, semantic_validation_error
	Detail string       `json:"detail"` // 错误描述
	Errors []FieldError `json:"errors"` // 具体字段错误列表
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s (%d errors)", e.Type, e.Detail, len(e.Errors))
}

// FieldError 字段错误
type FieldError struct {
	Line       int         `json:"line,omitempty"`       // 行号 (从 1 开始)
	Column     int         `json:"column,omitempty"`     // 列号 (从 1 开始)
	Field      string      `json:"field"`                // 字段路径 (如 "jobs.build.runs-on")
	Error      string      `json:"error"`                // 错误描述
	Value      interface{} `json:"value,omitempty"`      // 错误的值
	Snippet    string      `json:"snippet,omitempty"`    // 代码片段 (含上下文)
	Suggestion string      `json:"suggestion,omitempty"` // 修复建议
}

// ToHTTPError 转换为 HTTP 错误响应 (RFC 7807)
func (e *ValidationError) ToHTTPError() map[string]interface{} {
	return map[string]interface{}{
		"type":   "about:blank",
		"title":  "Workflow Validation Failed",
		"status": 400,
		"detail": e.Detail,
		"errors": e.Errors,
	}
}

// ToJSON 转换为 JSON 字符串
func (e *ValidationError) ToJSON() (string, error) {
	data, err := json.MarshalIndent(e.ToHTTPError(), "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
