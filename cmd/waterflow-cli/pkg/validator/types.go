package validator

import (
	"encoding/json"
	"time"
)

// Validator is the interface for workflow validation
type Validator interface {
	Validate(filepath string) (*ValidationResult, error)
}

// ValidationResult represents the validation result for a single file
type ValidationResult struct {
	File           string            `json:"file" yaml:"file"`
	Valid          bool              `json:"valid" yaml:"valid"`
	WorkflowName   string            `json:"workflow_name,omitempty" yaml:"workflow_name,omitempty"`
	JobsCount      int               `json:"jobs,omitempty" yaml:"jobs,omitempty"`
	StepsCount     int               `json:"steps,omitempty" yaml:"steps,omitempty"`
	Remote         bool              `json:"remote,omitempty" yaml:"remote,omitempty"`
	Errors         []ValidationError `json:"errors,omitempty" yaml:"errors,omitempty"`
	ValidationTime time.Duration     `json:"-" yaml:"-"`
}

// MarshalJSON customizes JSON encoding to convert ValidationTime to milliseconds
func (r *ValidationResult) MarshalJSON() ([]byte, error) {
	type Alias ValidationResult
	return json.Marshal(&struct {
		*Alias
		ValidationTimeMS int64 `json:"validation_time_ms"`
	}{
		Alias:            (*Alias)(r),
		ValidationTimeMS: r.ValidationTime.Milliseconds(),
	})
}

// ValidationTimeMS returns validation time in milliseconds
func (r *ValidationResult) ValidationTimeMS() int64 {
	return r.ValidationTime.Milliseconds()
}

// ValidationError represents a single validation error
type ValidationError struct {
	Type       string `json:"type" yaml:"type"`
	Field      string `json:"field,omitempty" yaml:"field,omitempty"`
	Line       int    `json:"line,omitempty" yaml:"line,omitempty"`
	Message    string `json:"message" yaml:"message"`
	Suggestion string `json:"suggestion,omitempty" yaml:"suggestion,omitempty"`
}

// ValidationResults represents the validation results for multiple files
type ValidationResults struct {
	Total  int                 `json:"total" yaml:"total"`
	Passed int                 `json:"passed" yaml:"passed"`
	Failed int                 `json:"failed" yaml:"failed"`
	Files  []*ValidationResult `json:"files" yaml:"files"`
}

// HasErrors returns true if any file validation failed
func (r *ValidationResults) HasErrors() bool {
	return r.Failed > 0
}
