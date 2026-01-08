package errors

import "fmt"

// FieldError represents a validation error for a specific field.
type FieldError struct {
	// Line is the line number where the error occurred (1-based)
	Line int `json:"line,omitempty"`
	// Column is the column number where the error occurred (1-based)
	Column int `json:"column,omitempty"`
	// Field is the name of the field that failed validation
	Field string `json:"field"`
	// Error is the validation error message
	Error string `json:"error"`
	// Value is the actual value that failed validation (optional)
	Value interface{} `json:"value,omitempty"`
	// Snippet is a code snippet showing the error context (optional)
	Snippet string `json:"snippet,omitempty"`
	// Suggestion is a hint for fixing the error (optional)
	Suggestion string `json:"suggestion,omitempty"`
}

// ValidationError represents validation failure with detailed field errors.
// This is a permanent error (non-retryable).
type ValidationError struct {
	*BaseError
	// Errors contains specific field validation errors
	Errors []FieldError
}

// NewValidationError creates a new ValidationError.
func NewValidationError(detail string, errors []FieldError) *ValidationError {
	return &ValidationError{
		BaseError: &BaseError{
			Type:      "validation_error",
			Message:   detail,
			Retryable: false,
		},
		Errors: errors,
	}
}

// ToRFC7807 converts the validation error to RFC 7807 format.
// This is used for REST API error responses.
func (e *ValidationError) ToRFC7807() map[string]interface{} {
	return map[string]interface{}{
		"type":   "about:blank",
		"title":  "Workflow Validation Failed",
		"status": 400,
		"detail": e.Message,
		"errors": e.Errors,
	}
}

// SchemaError represents a schema validation failure.
// This is a permanent error (non-retryable).
type SchemaError struct {
	*BaseError
	Field    string
	Expected string
	Actual   string
}

// NewSchemaError creates a new SchemaError.
func NewSchemaError(field, expected, actual string) *SchemaError {
	return &SchemaError{
		BaseError: &BaseError{
			Type:      "schema_error",
			Message:   fmt.Sprintf("Schema validation failed for %s", field),
			Retryable: false,
			Context: map[string]interface{}{
				"field":    field,
				"expected": expected,
				"actual":   actual,
			},
		},
		Field:    field,
		Expected: expected,
		Actual:   actual,
	}
}
