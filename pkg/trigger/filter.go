package trigger

import (
	"path/filepath"
	"strings"
)

// FilterEngine handles webhook event filtering
type FilterEngine struct{}

// NewFilterEngine creates a new FilterEngine
func NewFilterEngine() *FilterEngine {
	return &FilterEngine{}
}

// Match evaluates if a webhook event matches the filter configuration
func (f *FilterEngine) Match(event *WebhookEvent, filters *FilterConfig) (bool, string) {
	if filters == nil {
		return true, ""
	}

	// Check branch filters
	if event.Branch != "" {
		if !f.MatchBranch(event.Branch, filters) {
			return false, "branch not in filter"
		}
	}

	// Check tag filters
	if event.Tag != "" {
		if !f.MatchTag(event.Tag, filters) {
			return false, "tag not in filter"
		}
	}

	// Check path filters
	if len(filters.Paths) > 0 && len(event.ChangedFiles) > 0 {
		if !f.MatchPath(event.ChangedFiles, filters) {
			return false, "no changed files match path filter"
		}
	}

	// Check event type filters
	if len(filters.EventTypes) > 0 {
		if !f.MatchEventType(event.Type, filters) {
			return false, "event type not in filter"
		}
	}

	return true, ""
}

// MatchBranch checks if a branch matches the filter rules
func (f *FilterEngine) MatchBranch(branch string, filters *FilterConfig) bool {
	// Check branches-ignore first (exclusion has priority)
	for _, pattern := range filters.BranchesIgnore {
		if matchPattern(pattern, branch) {
			return false
		}
	}

	// If no branches filter specified, accept all (except ignored)
	if len(filters.Branches) == 0 {
		return true
	}

	// Check if branch matches any allowed pattern
	for _, pattern := range filters.Branches {
		if matchPattern(pattern, branch) {
			return true
		}
	}

	return false
}

// MatchTag checks if a tag matches the filter rules
func (f *FilterEngine) MatchTag(tag string, filters *FilterConfig) bool {
	// If no tag filter specified, accept all
	if len(filters.Tags) == 0 {
		return true
	}

	// Check if tag matches any pattern
	for _, pattern := range filters.Tags {
		if matchPattern(pattern, tag) {
			return true
		}
	}

	return false
}

// MatchPath checks if any changed file matches path filters
func (f *FilterEngine) MatchPath(changedFiles []string, filters *FilterConfig) bool {
	// If no path filter specified, accept all
	if len(filters.Paths) == 0 {
		return true
	}

	// Check if any changed file matches any path pattern
	for _, file := range changedFiles {
		for _, pattern := range filters.Paths {
			if matchPathPattern(pattern, file) {
				return true
			}
		}
	}

	return false
}

// MatchEventType checks if event type matches filter
func (f *FilterEngine) MatchEventType(eventType string, filters *FilterConfig) bool {
	// If no event type filter specified, accept all
	if len(filters.EventTypes) == 0 {
		return true
	}

	// Check if event type is in the list
	for _, t := range filters.EventTypes {
		if strings.EqualFold(t, eventType) {
			return true
		}
	}

	return false
}

// matchPattern performs wildcard pattern matching for branch/tag names
// Supports * (matches any sequence) and simple string matching
func matchPattern(pattern, value string) bool {
	// Direct match
	if pattern == value {
		return true
	}

	// Wildcard match using filepath.Match (supports *, ?, [...])
	matched, err := filepath.Match(pattern, value)
	if err != nil {
		// Invalid pattern, treat as literal string comparison
		return pattern == value
	}

	return matched
}

// matchPathPattern performs path-specific pattern matching
// Supports ** for directory wildcard and * for file wildcard
func matchPathPattern(pattern, path string) bool {
	// Handle ** (match any directory depth)
	if strings.Contains(pattern, "**") {
		// Convert ** to a more specific pattern
		// e.g., "docs/**" matches "docs/", "docs/api/", "docs/api/readme.md"
		parts := strings.Split(pattern, "**")

		if len(parts) == 2 {
			prefix := parts[0]
			suffix := parts[1]

			// Check if path starts with prefix
			if prefix != "" && !strings.HasPrefix(path, strings.TrimSuffix(prefix, "/")) {
				return false
			}

			// Check if path ends with suffix (if suffix exists and is not just /)
			if suffix != "" && suffix != "/" {
				// Remove leading / from suffix for matching
				suffix = strings.TrimPrefix(suffix, "/")
				if suffix != "" {
					matched, _ := filepath.Match(suffix, filepath.Base(path))
					return matched
				}
			}

			return true
		}
	}

	// Standard wildcard match
	matched, err := filepath.Match(pattern, path)
	if err != nil {
		return pattern == path
	}

	return matched
}
