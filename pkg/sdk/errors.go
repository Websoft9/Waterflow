package sdk

import "fmt"

// ServerError represents an error returned from the Waterflow server.
//
// It includes the HTTP status code, error code, message, and optional details.
// Use the helper functions IsNotFound and IsValidationError to check for
// specific error types.
//
// Example:
//
//	if serverErr, ok := err.(*ServerError); ok {
//	    fmt.Printf("Status: %d, Code: %s\n", serverErr.StatusCode, serverErr.Code)
//	}
type ServerError struct {
	StatusCode int
	Code       string
	Message    string
	Details    map[string]interface{}
}

func (e *ServerError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("server error (HTTP %d): %s - %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("server error (HTTP %d): %s", e.StatusCode, e.Message)
}

// IsNotFound checks if the error is a 404 Not Found error.
//
// Returns true if the error is a ServerError with HTTP status code 404.
//
// Example:
//
//	status, err := client.GetWorkflowStatus(ctx, workflowID)
//	if err != nil {
//	    if IsNotFound(err) {
//	        fmt.Println("Workflow not found")
//	        return
//	    }
//	    return err
//	}
func IsNotFound(err error) bool {
	if serverErr, ok := err.(*ServerError); ok {
		return serverErr.StatusCode == 404
	}
	return false
}

// IsValidationError checks if the error is a validation error (400 or 422).
//
// Returns true if the error is a ServerError with HTTP status code 400 or 422,
// indicating that the workflow YAML failed validation.
//
// Example:
//
//	resp, err := client.SubmitWorkflow(ctx, req)
//	if err != nil {
//	    if IsValidationError(err) {
//	        fmt.Println("Invalid workflow YAML")
//	        if serverErr, ok := err.(*ServerError); ok {
//	            fmt.Printf("Details: %v\n", serverErr.Details)
//	        }
//	        return
//	    }
//	}
func IsValidationError(err error) bool {
	if serverErr, ok := err.(*ServerError); ok {
		return serverErr.StatusCode == 400 || serverErr.StatusCode == 422
	}
	return false
}
