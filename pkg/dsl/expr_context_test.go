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
	err = err.WithPosition(5)
	assert.Equal(t, 5, err.Position)
	assert.Contains(t, err.Error(), "position 5")

	// With suggestion
	err = err.WithSuggestion("check syntax")
	assert.Equal(t, "check syntax", err.Suggestion)
}
