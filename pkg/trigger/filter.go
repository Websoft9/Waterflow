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

	// Check path filters (including paths_ignore)
	if (len(filters.Paths) > 0 || len(filters.PathsIgnore) > 0) && len(event.ChangedFiles) > 0 {
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
	// Check tags-ignore first (exclusion has priority)
	for _, pattern := range filters.TagsIgnore {
		if matchPattern(pattern, tag) {
			return false
		}
	}

	// If no tag filter specified, accept all (except ignored)
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
	if len(filters.Paths) == 0 && len(filters.PathsIgnore) == 0 {
		return true
	}

	for _, file := range changedFiles {
		// Check paths_ignore first — if matched, skip this file
		ignored := false
		for _, pattern := range filters.PathsIgnore {
			if matchPathPattern(pattern, file) {
				ignored = true
				break
			}
		}
		if ignored {
			continue
		}

		// If no paths filter, non-ignored file is a match
		if len(filters.Paths) == 0 {
			return true
		}

		// Check paths filter
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
// Supports ** for zero-or-more directory segments and * for within a segment
func matchPathPattern(pattern, path string) bool {
	// Normalize separators
	pattern = filepath.ToSlash(pattern)
	path = filepath.ToSlash(path)

	if !strings.Contains(pattern, "**") {
		matched, err := filepath.Match(pattern, path)
		if err != nil {
			return pattern == path
		}
		return matched
	}

	// Split both pattern and path into segments and use recursive matching
	patParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")
	return matchSegments(patParts, pathParts)
}

// matchSegments recursively matches pattern segments against path segments.
// ** consumes zero or more path segments.
func matchSegments(patParts, pathParts []string) bool {
	for len(patParts) > 0 {
		if patParts[0] == "**" {
			// ** matches zero or more path segments — try all possibilities
			for i := 0; i <= len(pathParts); i++ {
				if matchSegments(patParts[1:], pathParts[i:]) {
					return true
				}
			}
			return false
		}

		if len(pathParts) == 0 {
			return false
		}

		// Match single segment (may contain *)
		matched, err := filepath.Match(patParts[0], pathParts[0])
		if err != nil || !matched {
			return false
		}

		patParts = patParts[1:]
		pathParts = pathParts[1:]
	}

	return len(pathParts) == 0
}
