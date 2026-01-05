package validator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidationResult_MarshalJSON(t *testing.T) {
	result := &ValidationResult{
		File:           "test.yaml",
		Valid:          true,
		WorkflowName:   "Test Workflow",
		JobsCount:      2,
		StepsCount:     5,
		ValidationTime: 150 * time.Millisecond,
	}

	// Test that time is properly converted
	assert.Equal(t, int64(150), result.ValidationTimeMS())
}

func TestValidationResults_HasErrors(t *testing.T) {
	tests := []struct {
		name     string
		results  *ValidationResults
		expected bool
	}{
		{
			name: "no errors",
			results: &ValidationResults{
				Total:  3,
				Passed: 3,
				Failed: 0,
			},
			expected: false,
		},
		{
			name: "has errors",
			results: &ValidationResults{
				Total:  3,
				Passed: 2,
				Failed: 1,
			},
			expected: true,
		},
		{
			name: "all errors",
			results: &ValidationResults{
				Total:  3,
				Passed: 0,
				Failed: 3,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.results.HasErrors())
		})
	}
}

func TestValidationError_Fields(t *testing.T) {
	err := ValidationError{
		Type:       "schema_validation_error",
		Field:      "jobs.test.steps",
		Line:       10,
		Message:    "required field missing",
		Suggestion: "Add at least one step",
	}

	assert.Equal(t, "schema_validation_error", err.Type)
	assert.Equal(t, "jobs.test.steps", err.Field)
	assert.Equal(t, 10, err.Line)
	assert.Equal(t, "required field missing", err.Message)
	assert.Equal(t, "Add at least one step", err.Suggestion)
}
