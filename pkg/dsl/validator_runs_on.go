package dsl

import (
	"fmt"
	"strings"
)

// ValidateRunsOn validates runs-on field for Task Queue compatibility
// Per ADR-0006: Task Queue names must be alphanumeric with hyphens only.
// Delegates core naming validation to ValidateTaskQueueName to avoid duplication (M1 修复).
func ValidateRunsOn(runsOn string) error {
	if runsOn == "" {
		return fmt.Errorf("runs-on cannot be empty")
	}

	// Allow expressions (will be rendered at execution time)
	// Expressions can contain any characters, validation happens after rendering
	if containsExpression(runsOn) {
		return nil // Expression syntax will be validated separately
	}

	// Delegate to the canonical ValidateTaskQueueName for naming rules
	return ValidateTaskQueueName(runsOn)
}

// containsExpression checks if a string contains expression syntax
// Returns true if the string contains ${{ ... }} pattern
func containsExpression(s string) bool {
	return strings.Contains(s, "${{") && strings.Contains(s, "}}")
}
