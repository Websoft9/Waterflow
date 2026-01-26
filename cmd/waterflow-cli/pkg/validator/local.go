package validator

import (
	"fmt"
	"os"
	"time"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"go.uber.org/zap"
)

// LocalValidator validates workflows using local DSL parser
type LocalValidator struct {
	validator *dsl.Validator
	logger    *zap.Logger
}

// NewLocalValidator creates a new local validator
func NewLocalValidator(validator *dsl.Validator, logger *zap.Logger) *LocalValidator {
	return &LocalValidator{
		validator: validator,
		logger:    logger,
	}
}

// Validate validates a workflow file using local DSL parser
func (v *LocalValidator) Validate(filepath string) (*ValidationResult, error) {
	start := time.Now()

	// Read file
	content, err := os.ReadFile(filepath) //nolint:gosec // File path from user input
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", filepath)
		}
		if os.IsPermission(err) {
			return nil, fmt.Errorf("permission denied: %s", filepath)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Check file size limit (10MB)
	const maxSize = 10 * 1024 * 1024
	if len(content) > maxSize {
		return &ValidationResult{
			File:  filepath,
			Valid: false,
			Errors: []ValidationError{{
				Type:       "file_error",
				Message:    fmt.Sprintf("File size %d bytes exceeds limit %d bytes", len(content), maxSize),
				Suggestion: "Split large workflow into multiple smaller workflows",
			}},
			ValidationTime: time.Since(start),
		}, nil
	}

	// Check empty file
	if len(content) == 0 {
		return &ValidationResult{
			File:  filepath,
			Valid: false,
			Errors: []ValidationError{{
				Type:       "file_error",
				Message:    "Empty YAML file",
				Suggestion: "Add workflow definition to the file",
			}},
			ValidationTime: time.Since(start),
		}, nil
	}

	// Validate using DSL Validator
	workflow, err := v.validator.ValidateYAML(content)

	duration := time.Since(start)

	if err != nil {
		return v.convertValidationError(filepath, err, duration), nil
	}

	// Validation successful
	return &ValidationResult{
		File:           filepath,
		Valid:          true,
		WorkflowName:   workflow.Name,
		JobsCount:      len(workflow.Jobs),
		StepsCount:     v.countSteps(workflow),
		ValidationTime: duration,
	}, nil
}

// convertValidationError converts DSL validation error to CLI format
func (v *LocalValidator) convertValidationError(filepath string, err error, duration time.Duration) *ValidationResult {
	result := &ValidationResult{
		File:           filepath,
		Valid:          false,
		ValidationTime: duration,
	}

	// Convert DSL ValidationError
	if valErr, ok := err.(*dsl.ValidationError); ok {
		result.Errors = make([]ValidationError, len(valErr.Errors))
		for i, fieldErr := range valErr.Errors {
			result.Errors[i] = ValidationError{
				Type:       valErr.Type,
				Field:      fieldErr.Field,
				Line:       fieldErr.Line,
				Message:    fieldErr.Error,
				Suggestion: fieldErr.Suggestion,
			}
		}
	} else {
		// Unknown error type
		result.Errors = []ValidationError{{
			Type:    "unknown_error",
			Message: err.Error(),
		}}
	}

	return result
}

// countSteps counts total steps across all jobs in the workflow
func (v *LocalValidator) countSteps(workflow *dsl.Workflow) int {
	count := 0
	for _, job := range workflow.Jobs {
		count += len(job.Steps)
	}
	return count
}
