package dsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepsOutputManager(t *testing.T) {
	manager := NewStepsOutputManager()

	// Set outputs
	manager.Set("checkout", map[string]interface{}{
		"commit": "abc123",
		"branch": "main",
	})

	// Get existing output
	commit, err := manager.Get("checkout", "commit")
	require.NoError(t, err)
	assert.Equal(t, "abc123", commit)

	// Get non-existent step
	_, err = manager.Get("notexist", "commit")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found or not executed")

	// Get non-existent output key
	_, err = manager.Get("checkout", "unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "output 'unknown' not found")
	assert.Contains(t, err.Error(), "Available:")

	// ToContext
	ctx := manager.ToContext()
	assert.NotNil(t, ctx)
	assert.Contains(t, ctx, "checkout")
}

func TestExpressionError(t *testing.T) {
	err := NewExpressionError("1 + 2", "syntax error", "syntax_error")

	assert.Equal(t, "1 + 2", err.Expression)
	assert.Equal(t, "syntax error", err.Message)
	assert.Equal(t, "syntax_error", err.Type)
	assert.Contains(t, err.Error(), "syntax error")
	assert.Contains(t, err.Error(), "1 + 2")

	// With position
	errWithPos := err.WithPosition(5)
	assert.Equal(t, 5, errWithPos.Position)
	assert.Contains(t, errWithPos.Error(), "position 5")

	// With suggestion
	errWithSug := errWithPos.WithSuggestion("try using vars.version")
	assert.Equal(t, "try using vars.version", errWithSug.Suggestion)
}

func TestEvalContext_UpdateJobStatus(t *testing.T) {
	tests := []struct {
		name            string
		initialJob      map[string]interface{}
		status          string
		expectSuccess   bool
		expectFailure   bool
		expectCancelled bool
	}{
		{
			name:            "success status",
			initialJob:      nil,
			status:          "success",
			expectSuccess:   true,
			expectFailure:   false,
			expectCancelled: false,
		},
		{
			name:            "failure status",
			initialJob:      map[string]interface{}{"id": "test"},
			status:          "failure",
			expectSuccess:   false,
			expectFailure:   true,
			expectCancelled: false,
		},
		{
			name:            "cancelled status",
			initialJob:      map[string]interface{}{"id": "test"},
			status:          "cancelled",
			expectSuccess:   false,
			expectFailure:   false,
			expectCancelled: true,
		},
		{
			name:            "empty status defaults to success",
			initialJob:      nil,
			status:          "",
			expectSuccess:   true,
			expectFailure:   false,
			expectCancelled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &EvalContext{
				Job: tt.initialJob,
			}

			ctx.UpdateJobStatus(tt.status)

			// Check job status updated
			require.NotNil(t, ctx.Job)
			assert.Equal(t, tt.status, ctx.Job["status"])

			// Check condition functions
			assert.Equal(t, tt.expectSuccess, ctx.Success())
			assert.Equal(t, tt.expectFailure, ctx.Failure())
			assert.Equal(t, tt.expectCancelled, ctx.Cancelled())
		})
	}
}
