package errors

import "net/http"

// RFC7807Response represents RFC 7807 Problem Details for HTTP APIs.
// See: https://datatracker.ietf.org/doc/html/rfc7807
type RFC7807Response struct {
	// Type is a URI reference that identifies the problem type
	Type string `json:"type"`
	// Title is a short, human-readable summary of the problem type
	Title string `json:"title"`
	// Status is the HTTP status code
	Status int `json:"status"`
	// Detail is a human-readable explanation specific to this occurrence
	Detail string `json:"detail"`
	// Instance is a URI reference that identifies the specific occurrence
	Instance string `json:"instance,omitempty"`
	// Errors contains additional validation errors (optional)
	Errors interface{} `json:"errors,omitempty"`
	// Metadata contains additional error context (optional)
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ToRFC7807 converts a WaterflowError to RFC 7807 format.
// It automatically maps error types to HTTP status codes and titles.
func ToRFC7807(err error, instance string) *RFC7807Response {
	if wfErr, ok := err.(WaterflowError); ok {
		status := errorTypeToHTTPStatus(wfErr.ErrorType())

		resp := &RFC7807Response{
			Type:     wfErr.ErrorType(),
			Title:    errorTypeToTitle(wfErr.ErrorType()),
			Status:   status,
			Detail:   wfErr.ErrorMessage(),
			Instance: instance,
			Metadata: sanitizeContext(wfErr.ErrorContext()),
		}

		// Special handling for ValidationError
		if valErr, ok := err.(*ValidationError); ok {
			resp.Errors = valErr.Errors
		}

		return resp
	}

	// Unknown error defaults to 500 Internal Server Error
	return &RFC7807Response{
		Type:     "internal_error",
		Title:    "Internal Server Error",
		Status:   500,
		Detail:   err.Error(),
		Instance: instance,
	}
}

// errorTypeToHTTPStatus maps error types to HTTP status codes.
func errorTypeToHTTPStatus(errType string) int {
	switch errType {
	case "validation_error", "schema_error", "invalid_argument":
		return http.StatusBadRequest // 400
	case "not_found":
		return http.StatusNotFound // 404
	case "permission_denied":
		return http.StatusForbidden // 403
	case "deadline_exceeded":
		return http.StatusRequestTimeout // 408
	case "service_unavailable", "connection_refused":
		return http.StatusServiceUnavailable // 503
	case "cancelled":
		return 499 // Client Closed Request (nginx convention)
	default:
		return http.StatusInternalServerError // 500
	}
}

// errorTypeToTitle maps error types to human-readable titles.
func errorTypeToTitle(errType string) string {
	titles := map[string]string{
		"validation_error":    "Validation Failed",
		"schema_error":        "Schema Validation Failed",
		"not_found":           "Resource Not Found",
		"permission_denied":   "Permission Denied",
		"invalid_argument":    "Invalid Argument",
		"node_not_registered": "Node Not Registered",
		"deadline_exceeded":   "Request Timeout",
		"service_unavailable": "Service Unavailable",
		"connection_refused":  "Service Unavailable",
		"internal_error":      "Internal Server Error",
		"cancelled":           "Request Cancelled",
		"unknown_error":       "Unknown Error",
	}

	if title, ok := titles[errType]; ok {
		return title
	}
	return "Error"
}
