package errors

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestToRFC7807 tests RFC 7807 conversion
func TestToRFC7807(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		instance       string
		expectedStatus int
		expectedType   string
		expectedTitle  string
	}{
		{
			name:           "validation error",
			err:            NewValidationError("YAML syntax error", []FieldError{{Field: "test", Error: "required"}}),
			instance:       "/v1/workflows/abc-123",
			expectedStatus: 400,
			expectedType:   "validation_error",
			expectedTitle:  "Validation Failed",
		},
		{
			name:           "not found error",
			err:            NewWorkflowNotFoundError("wf-123"),
			instance:       "/v1/workflows/wf-123",
			expectedStatus: 404,
			expectedType:   "not_found",
			expectedTitle:  "Resource Not Found",
		},
		{
			name:           "timeout error",
			err:            NewWorkflowTimeoutError("wf-123", 0),
			instance:       "/v1/workflows/wf-123",
			expectedStatus: 408,
			expectedType:   "deadline_exceeded",
			expectedTitle:  "Request Timeout",
		},
		{
			name:           "unknown error",
			err:            errors.New("unknown error"),
			instance:       "/v1/test",
			expectedStatus: 500,
			expectedType:   "internal_error",
			expectedTitle:  "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rfc7807 := ToRFC7807(tt.err, tt.instance)

			assert.NotNil(t, rfc7807)
			assert.Equal(t, tt.expectedStatus, rfc7807.Status)
			assert.Equal(t, tt.expectedType, rfc7807.Type)
			assert.Equal(t, tt.expectedTitle, rfc7807.Title)
			assert.Equal(t, tt.instance, rfc7807.Instance)
			assert.NotEmpty(t, rfc7807.Detail)
		})
	}
}

// TestToRFC7807_ValidationError tests validation error special handling
func TestToRFC7807_ValidationError(t *testing.T) {
	fieldErrors := []FieldError{
		{Field: "jobs.build.runs-on", Error: "required"},
		{Field: "jobs.test.steps", Error: "cannot be empty"},
	}
	err := NewValidationError("Found 2 errors", fieldErrors)

	rfc7807 := ToRFC7807(err, "/v1/workflows/test")

	assert.NotNil(t, rfc7807.Errors)
	// Errors should be the field errors array
	errorsArray, ok := rfc7807.Errors.([]FieldError)
	assert.True(t, ok)
	assert.Len(t, errorsArray, 2)
}

// TestToRFC7807_SanitizedContext tests context sanitization
func TestToRFC7807_SanitizedContext(t *testing.T) {
	err := &BaseError{
		Type:    "test_error",
		Message: "Test error",
		Context: map[string]interface{}{
			"workflow_id": "wf-123",
			"password":    "secret123",
		},
	}

	rfc7807 := ToRFC7807(err, "/v1/test")

	assert.NotNil(t, rfc7807.Metadata)
	assert.Equal(t, "wf-123", rfc7807.Metadata["workflow_id"])
	// Password should be redacted
	assert.Equal(t, "***REDACTED***", rfc7807.Metadata["password"])
}

// TestErrorTypeToHTTPStatus tests HTTP status code mapping
func TestErrorTypeToHTTPStatus(t *testing.T) {
	tests := []struct {
		errType        string
		expectedStatus int
	}{
		{"validation_error", http.StatusBadRequest},
		{"schema_error", http.StatusBadRequest},
		{"invalid_argument", http.StatusBadRequest},
		{"not_found", http.StatusNotFound},
		{"permission_denied", http.StatusForbidden},
		{"deadline_exceeded", http.StatusRequestTimeout},
		{"service_unavailable", http.StatusServiceUnavailable},
		{"connection_refused", http.StatusServiceUnavailable},
		{"cancelled", 499},
		{"internal_error", http.StatusInternalServerError},
		{"unknown_error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.errType, func(t *testing.T) {
			status := errorTypeToHTTPStatus(tt.errType)
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

// TestErrorTypeToTitle tests human-readable title mapping
func TestErrorTypeToTitle(t *testing.T) {
	tests := []struct {
		errType       string
		expectedTitle string
	}{
		{"validation_error", "Validation Failed"},
		{"schema_error", "Schema Validation Failed"},
		{"not_found", "Resource Not Found"},
		{"permission_denied", "Permission Denied"},
		{"invalid_argument", "Invalid Argument"},
		{"node_not_registered", "Node Not Registered"},
		{"deadline_exceeded", "Request Timeout"},
		{"service_unavailable", "Service Unavailable"},
		{"internal_error", "Internal Server Error"},
		{"unknown_type", "Error"},
	}

	for _, tt := range tests {
		t.Run(tt.errType, func(t *testing.T) {
			title := errorTypeToTitle(tt.errType)
			assert.Equal(t, tt.expectedTitle, title)
		})
	}
}
