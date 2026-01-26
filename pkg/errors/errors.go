// Package errors provides a unified typed error handling system for Waterflow.
// It implements RFC 7807 Problem Details for HTTP APIs and supports error classification
// for Temporal retry strategies.
package errors

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
)

// WaterflowError is the unified interface for all Waterflow errors.
// It extends the standard error interface with type information, context,
// retryability classification, and JSON serialization.
type WaterflowError interface {
	error
	// ErrorType returns the error type (validation_error, node_error, etc.)
	ErrorType() string
	// ErrorMessage returns the human-readable error message
	ErrorMessage() string
	// ErrorContext returns the error context (workflow_id, job_id, step_name, etc.)
	ErrorContext() map[string]interface{}
	// IsRetryable returns whether this error should be retried
	IsRetryable() bool
	// ToJSON serializes the error to JSON format
	ToJSON() ([]byte, error)
}

// BaseError is the foundation implementation for all error types.
// It provides common functionality like error wrapping, context management,
// and optional stack trace collection.
type BaseError struct {
	// Type is the error type (validation_error, node_error, etc.)
	Type string
	// Message is the human-readable error message
	Message string
	// Context holds additional error context (workflow_id, job_id, etc.)
	Context map[string]interface{}
	// Cause is the underlying error (supports errors.Unwrap)
	Cause error
	// Retryable indicates whether this error should be retried
	Retryable bool
	// StackTrace holds the stack trace (only collected in debug mode or explicit call)
	StackTrace string
}

// Error implements the error interface.
// It formats the error message with optional cause chain.
func (e *BaseError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying error, supporting errors.Is() and errors.As().
func (e *BaseError) Unwrap() error {
	return e.Cause
}

// ErrorType implements WaterflowError interface.
func (e *BaseError) ErrorType() string {
	return e.Type
}

// ErrorMessage implements WaterflowError interface.
func (e *BaseError) ErrorMessage() string {
	return e.Message
}

// ErrorContext implements WaterflowError interface.
func (e *BaseError) ErrorContext() map[string]interface{} {
	return e.Context
}

// IsRetryable implements WaterflowError interface.
func (e *BaseError) IsRetryable() bool {
	return e.Retryable
}

// ToJSON implements WaterflowError interface.
// It serializes the error to JSON format with sanitized context.
func (e *BaseError) ToJSON() ([]byte, error) {
	data := map[string]interface{}{
		"type":    e.Type,
		"message": e.Message,
		"context": sanitizeContext(e.Context),
	}
	if e.Cause != nil {
		data["cause"] = e.Cause.Error()
	}
	if e.StackTrace != "" {
		data["stack_trace"] = e.StackTrace
	}
	return json.Marshal(data)
}

// WithContext adds context information (chainable).
// It creates the context map if it doesn't exist and adds the key-value pair.
func (e *BaseError) WithContext(key string, value interface{}) *BaseError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithCause wraps an underlying error (chainable).
// It sets the Cause field for error chain support.
func (e *BaseError) WithCause(cause error) *BaseError {
	e.Cause = cause
	return e
}

// WithStackTrace collects and adds stack trace (chainable).
// PERFORMANCE NOTE: Stack trace collection has non-trivial overhead.
// Only call this in debug mode or when explicitly needed for troubleshooting.
// The exact overhead depends on call depth and will be benchmarked in future versions.
func (e *BaseError) WithStackTrace() *BaseError {
	e.StackTrace = captureStackTrace()
	return e
}

// captureStackTrace captures the current call stack.
// It uses runtime.Caller to collect stack frames, skipping the first 3 frames
// (captureStackTrace -> WithStackTrace -> error constructor).
func captureStackTrace() string {
	const maxDepth = 32
	var frames []string

	for i := 3; i < maxDepth; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		var funcName string
		if fn != nil {
			funcName = fn.Name()
		} else {
			funcName = "unknown"
		}

		frames = append(frames, fmt.Sprintf("%s\n\t%s:%d", funcName, file, line))
	}

	return strings.Join(frames, "\n")
}

// shouldCollectStackTrace determines whether stack trace should be collected.
// It checks DEBUG environment variable and can be extended with config support.
func shouldCollectStackTrace() bool {
	// Check DEBUG environment variable
	if debug := os.Getenv("DEBUG"); debug == "true" || debug == "1" {
		return true
	}
	// TODO: Add config file support in future (collect_stack_trace: true)
	return false
}

// WrapError wraps an error and adds context.
// If the error is already a WaterflowError, it preserves the type and retryability.
// Otherwise, it classifies the error and wraps it.
func WrapError(err error, errType, message string) *BaseError {
	if err == nil {
		return nil
	}

	// If already a WaterflowError, preserve its type and retryability
	if wfErr, ok := err.(WaterflowError); ok {
		wrapped := &BaseError{
			Type:      wfErr.ErrorType(),
			Message:   message,
			Cause:     err,
			Retryable: wfErr.IsRetryable(),
			Context:   wfErr.ErrorContext(),
		}
		// Collect stack trace if in debug mode
		if shouldCollectStackTrace() {
			_ = wrapped.WithStackTrace()
		}
		return wrapped
	}

	// Otherwise, classify and wrap
	classifier := NewErrorClassifier()
	classifiedType := classifier.ClassifyError(err)

	wrapped := &BaseError{
		Type:      classifiedType,
		Message:   message,
		Cause:     err,
		Retryable: classifier.IsRetryable(classifiedType),
	}

	// Collect stack trace if in debug mode
	if shouldCollectStackTrace() {
		_ = wrapped.WithStackTrace()
	}

	return wrapped
}

// WrapWithContext wraps an error and adds key-value context.
// It preserves error type and retryability, and merges context information.
//
// Context Merge Strategy: Existing keys are NOT overwritten. If a key already
// exists in the wrapped error's context, the new value is ignored. This preserves
// context from inner errors. Use WithContext() directly if you need to overwrite.
func WrapWithContext(err error, message string, context map[string]interface{}) *BaseError {
	wrapped := WrapError(err, "", message)
	if wrapped == nil {
		return nil
	}

	if wrapped.Context == nil {
		wrapped.Context = context
	} else {
		// Merge context (new values don't overwrite existing)
		for k, v := range context {
			if _, exists := wrapped.Context[k]; !exists {
				wrapped.Context[k] = v
			}
		}
	}

	return wrapped
}

// Default sensitive keys (can be extended via config)
var defaultSensitiveKeys = []string{
	"password", "api_key", "token", "secret", "credential",
	"authorization", "auth", "key", "private_key", "access_token",
	"refresh_token", "session", "cookie",
}

// SensitiveFieldsConfig holds configuration for sensitive field detection.
type SensitiveFieldsConfig struct {
	// AdditionalFields specifies extra field names to treat as sensitive
	AdditionalFields []string
}

// sanitizeContext removes sensitive information from context.
// It replaces sensitive field values with "***REDACTED***".
func sanitizeContext(ctx map[string]interface{}) map[string]interface{} {
	return SanitizeContextWithConfig(ctx, SensitiveFieldsConfig{})
}

// SanitizeContextWithConfig removes sensitive information with custom config.
// This exported version allows users to customize sensitive field detection.
func SanitizeContextWithConfig(ctx map[string]interface{}, config SensitiveFieldsConfig) map[string]interface{} {
	if ctx == nil {
		return nil
	}

	sanitized := make(map[string]interface{})

	// Build sensitive keys map
	sensitiveKeys := make(map[string]bool)
	for _, key := range defaultSensitiveKeys {
		sensitiveKeys[strings.ToLower(key)] = true
	}
	for _, key := range config.AdditionalFields {
		sensitiveKeys[strings.ToLower(key)] = true
	}

	// Filter context
	for k, v := range ctx {
		if sensitiveKeys[strings.ToLower(k)] {
			sanitized[k] = "***REDACTED***"
		} else {
			sanitized[k] = v
		}
	}

	return sanitized
}
