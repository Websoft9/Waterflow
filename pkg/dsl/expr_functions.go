package dsl

import (
	"encoding/json"
	"fmt"
	"strings"
)

// GetBuiltinFunctions returns all built-in functions for expression evaluation
func GetBuiltinFunctions() map[string]interface{} {
	return map[string]interface{}{
		// String functions
		"len":        builtinLen,
		"upper":      strings.ToUpper,
		"lower":      strings.ToLower,
		"trim":       strings.TrimSpace,
		"split":      strings.Split,
		"join":       strings.Join,
		"format":     builtinFormat,
		"contains":   strings.Contains,
		"startsWith": strings.HasPrefix,
		"endsWith":   strings.HasSuffix,

		// JSON functions
		"toJSON":   builtinToJSON,
		"fromJSON": builtinFromJSON,

		// Condition functions
		"always": builtinAlways,
	}
}

// builtinLen returns length of string or array
func builtinLen(v interface{}) (int, error) {
	switch val := v.(type) {
	case string:
		return len(val), nil
	case []interface{}:
		return len(val), nil
	case []string:
		return len(val), nil
	default:
		return 0, fmt.Errorf("len() expects string or array, got %T", v)
	}
}

// builtinFormat formats a template string with arguments
func builtinFormat(template string, args ...interface{}) string {
	result := template
	for i, arg := range args {
		placeholder := fmt.Sprintf("{%d}", i)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", arg))
	}
	return result
}

// builtinToJSON converts a value to JSON string
func builtinToJSON(v interface{}) (string, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("toJSON error: %w", err)
	}
	return string(bytes), nil
}

// builtinFromJSON parses a JSON string
func builtinFromJSON(s string) (interface{}, error) {
	var result interface{}
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil, fmt.Errorf("fromJSON error: %w", err)
	}
	return result, nil
}

// builtinAlways always returns true
func builtinAlways() bool {
	return true
}

// Story 1.5: 条件函数工厂 (需要Job状态上下文)

// MakeSuccessFunc creates a success() function with job status context
func MakeSuccessFunc(jobStatus string) func() bool {
	return func() bool {
		return jobStatus == "success" || jobStatus == ""
	}
}

// MakeFailureFunc creates a failure() function with job status context
func MakeFailureFunc(jobStatus string) func() bool {
	return func() bool {
		return jobStatus == "failure"
	}
}

// MakeCancelledFunc creates a cancelled() function with job status context
func MakeCancelledFunc(jobStatus string) func() bool {
	return func() bool {
		return jobStatus == "cancelled"
	}
}

// ContextualFunctions returns functions that require runtime context
// These are implemented separately and injected at evaluation time
type ContextualFunctions struct {
	JobStatus string
}

// Success returns true if job status is success
func (c *ContextualFunctions) Success() bool {
	return c.JobStatus == "success"
}

// Failure returns true if job status is failure
func (c *ContextualFunctions) Failure() bool {
	return c.JobStatus == "failure"
}

// Cancelled returns true if job status is cancelled
func (c *ContextualFunctions) Cancelled() bool {
	return c.JobStatus == "cancelled"
}
