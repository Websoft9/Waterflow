package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewValidationError tests ValidationError construction
func TestNewValidationError(t *testing.T) {
	errors := []FieldError{
		{
			Line:       10,
			Field:      "jobs.build.runs-on",
			Error:      "required",
			Suggestion: "add runs-on: <task-queue-name>",
		},
		{
			Line:  15,
			Field: "jobs.build.steps[0].run",
			Error: "invalid syntax",
		},
	}

	err := NewValidationError("Found 2 validation errors", errors)

	assert.NotNil(t, err)
	assert.Equal(t, "validation_error", err.ErrorType())
	assert.False(t, err.IsRetryable())
	assert.Len(t, err.Errors, 2)
	assert.Equal(t, "jobs.build.runs-on", err.Errors[0].Field)
	assert.Equal(t, 10, err.Errors[0].Line)
}

// TestValidationError_ToRFC7807 tests RFC 7807 conversion
func TestValidationError_ToRFC7807(t *testing.T) {
	fieldErrors := []FieldError{
		{
			Field: "jobs.build.runs-on",
			Error: "required",
		},
	}

	err := NewValidationError("Validation failed", fieldErrors)
	rfc7807 := err.ToRFC7807()

	assert.Equal(t, "about:blank", rfc7807["type"])
	assert.Equal(t, "Workflow Validation Failed", rfc7807["title"])
	assert.Equal(t, 400, rfc7807["status"])
	assert.Equal(t, "Validation failed", rfc7807["detail"])
	assert.NotNil(t, rfc7807["errors"])
}

// TestNewSchemaError tests SchemaError construction
func TestNewSchemaError(t *testing.T) {
	err := NewSchemaError("jobs.build.runs-on", "string", "null")

	assert.NotNil(t, err)
	assert.Equal(t, "schema_error", err.ErrorType())
	assert.False(t, err.IsRetryable())
	assert.Equal(t, "jobs.build.runs-on", err.Field)
	assert.Equal(t, "string", err.Expected)
	assert.Equal(t, "null", err.Actual)
	assert.Contains(t, err.Error(), "jobs.build.runs-on")
}

// TestSchemaError_Context tests schema error context
func TestSchemaError_Context(t *testing.T) {
	err := NewSchemaError("jobs.build.timeout", "string or number", "boolean")

	ctx := err.ErrorContext()
	assert.Equal(t, "jobs.build.timeout", ctx["field"])
	assert.Equal(t, "string or number", ctx["expected"])
	assert.Equal(t, "boolean", ctx["actual"])
}

// TestValidationError_ToJSON tests JSON serialization
func TestValidationError_ToJSON(t *testing.T) {
	errors := []FieldError{
		{
			Line:  10,
			Field: "jobs.build.runs-on",
			Error: "required",
		},
	}

	err := NewValidationError("Validation failed", errors)
	data, jsonErr := err.ToJSON()

	require.NoError(t, jsonErr)
	require.NotEmpty(t, data)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, "validation_error")
	assert.Contains(t, jsonStr, "Validation failed")
}

// TestFieldError_Structure tests FieldError structure
func TestFieldError_Structure(t *testing.T) {
	fe := FieldError{
		Line:       42,
		Column:     10,
		Field:      "jobs.test.steps[2].uses",
		Error:      "unknown node type",
		Value:      "custom/unknown",
		Snippet:    "  - uses: custom/unknown",
		Suggestion: "check available node types with 'waterflow node list'",
	}

	assert.Equal(t, 42, fe.Line)
	assert.Equal(t, 10, fe.Column)
	assert.Equal(t, "jobs.test.steps[2].uses", fe.Field)
	assert.Equal(t, "unknown node type", fe.Error)
	assert.Equal(t, "custom/unknown", fe.Value)
	assert.NotEmpty(t, fe.Snippet)
	assert.NotEmpty(t, fe.Suggestion)
}
