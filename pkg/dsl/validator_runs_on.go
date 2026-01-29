package dsl

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateRunsOn validates runs-on field for Task Queue compatibility
// Per ADR-0006: Task Queue names must be alphanumeric with hyphens only
func ValidateRunsOn(runsOn string) error {
	if runsOn == "" {
		return fmt.Errorf("runs-on cannot be empty")
	}

	// Check length first
	if len(runsOn) > 255 {
		return fmt.Errorf("task queue name too long: %d characters (max 255)", len(runsOn))
	}

	// Allow expressions (will be rendered at execution time)
	// Expressions can contain any characters, validation happens after rendering
	if containsExpression(runsOn) {
		return nil // Expression syntax will be validated separately
	}

	// Temporal Task Queue naming requirements:
	// - Only alphanumeric and hyphens
	// - Must start and end with alphanumeric
	re := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)
	if !re.MatchString(runsOn) {
		return fmt.Errorf("invalid task queue name '%s': must contain only alphanumeric characters and hyphens, and start/end with alphanumeric", runsOn)
	}

	return nil
}

// containsExpression checks if a string contains expression syntax
// Returns true if the string contains ${{ ... }} pattern
func containsExpression(s string) bool {
	return strings.Contains(s, "${{") && strings.Contains(s, "}}")
}
