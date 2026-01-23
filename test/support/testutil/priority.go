// Package testutil provides testing utilities for Waterflow tests
// This file contains test priority helpers and build tag documentation
package testutil

/*
Test Priority Classification System for Waterflow

PRIORITY LEVELS:
  - P0 (Critical): Core functionality, must pass for any deployment
  - P1 (High): Important features, should pass for production
  - P2 (Medium): Secondary features, nice to pass
  - P3 (Low): Edge cases, can be deferred

USAGE WITH BUILD TAGS:

Create test files with build tags:

  //go:build p0
  // +build p0

  package mytest

  func TestCriticalPath(t *testing.T) {
      // P0 critical test
  }

Run tests by priority:

  # Run only P0 (critical) tests
  go test -tags=p0 ./...

  # Run P0 and P1 tests
  go test -tags='p0 p1' ./...

  # Run all tests (default, no tags)
  go test ./...

ALTERNATIVE: Using naming conventions:

  func TestP0_CriticalPath(t *testing.T) {}
  func TestP1_ImportantFeature(t *testing.T) {}

Run with name filter:
  go test -run="TestP0_" ./...
  go test -run="TestP(0|1)_" ./...

ALTERNATIVE: Using environment variables:

  TEST_PRIORITY=p0 go test ./...
*/

import (
	"os"
	"strings"
	"testing"
)

// TestPriority represents test priority level
type TestPriority int

const (
	// P0 Critical - Core functionality, must pass for any deployment
	P0 TestPriority = iota
	// P1 High - Important features, should pass for production
	P1
	// P2 Medium - Secondary features, nice to pass
	P2
	// P3 Low - Edge cases, can be deferred
	P3
)

// String returns the string representation of the priority
func (p TestPriority) String() string {
	switch p {
	case P0:
		return "P0"
	case P1:
		return "P1"
	case P2:
		return "P2"
	case P3:
		return "P3"
	default:
		return "Unknown"
	}
}

// SkipIfNotPriority skips the test if TEST_PRIORITY env var doesn't match
// Usage: testutil.SkipIfNotPriority(t, testutil.P0)
func SkipIfNotPriority(t *testing.T, priority TestPriority) {
	t.Helper()

	envPriority := os.Getenv("TEST_PRIORITY")
	if envPriority == "" || envPriority == "all" {
		// No priority filter, run all tests
		return
	}

	// Parse priority from env var (e.g., "p0", "p0,p1", "P0")
	envPriority = strings.ToLower(envPriority)
	priorities := strings.Split(envPriority, ",")

	targetPriority := strings.ToLower(priority.String())
	for _, p := range priorities {
		if strings.TrimSpace(p) == targetPriority {
			return
		}
	}

	t.Skipf("Skipping %s test - TEST_PRIORITY=%s", priority, envPriority)
}

// RequirePriority returns true if the test should run based on TEST_PRIORITY
func RequirePriority(priority TestPriority) bool {
	envPriority := os.Getenv("TEST_PRIORITY")
	if envPriority == "" || envPriority == "all" {
		return true
	}

	envPriority = strings.ToLower(envPriority)
	priorities := strings.Split(envPriority, ",")

	targetPriority := strings.ToLower(priority.String())
	for _, p := range priorities {
		if strings.TrimSpace(p) == targetPriority {
			return true
		}
	}

	return false
}

// IsCriticalTest marks a test as P0 Critical
// Skip if TEST_PRIORITY is set and doesn't include P0
func IsCriticalTest(t *testing.T) {
	t.Helper()
	SkipIfNotPriority(t, P0)
}

// IsHighPriorityTest marks a test as P1 High
func IsHighPriorityTest(t *testing.T) {
	t.Helper()
	SkipIfNotPriority(t, P1)
}

// IsMediumPriorityTest marks a test as P2 Medium
func IsMediumPriorityTest(t *testing.T) {
	t.Helper()
	SkipIfNotPriority(t, P2)
}

// IsLowPriorityTest marks a test as P3 Low
func IsLowPriorityTest(t *testing.T) {
	t.Helper()
	SkipIfNotPriority(t, P3)
}

// TestCategory represents test category for organization
type TestCategory string

const (
	// CategoryUnit for unit tests
	CategoryUnit TestCategory = "unit"
	// CategoryIntegration for integration tests
	CategoryIntegration TestCategory = "integration"
	// CategoryE2E for end-to-end tests
	CategoryE2E TestCategory = "e2e"
	// CategoryPerformance for performance tests
	CategoryPerformance TestCategory = "performance"
	// CategoryStress for stress tests
	CategoryStress TestCategory = "stress"
)

// SkipIfNotCategory skips if TEST_CATEGORY doesn't match
func SkipIfNotCategory(t *testing.T, category TestCategory) {
	t.Helper()

	envCategory := os.Getenv("TEST_CATEGORY")
	if envCategory == "" || envCategory == "all" {
		return
	}

	envCategory = strings.ToLower(envCategory)
	if envCategory != string(category) {
		t.Skipf("Skipping %s test - TEST_CATEGORY=%s", category, envCategory)
	}
}
