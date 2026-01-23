// Package testutil provides test utilities for the Waterflow project.
// This file contains BDD (Behavior-Driven Development) documentation helpers.
package testutil

import (
	"fmt"
	"strings"
	"testing"
)

// BDDSpec represents a BDD-style test specification.
// Use this to document test behavior in a structured way.
//
// Example:
//
//	func TestWorkflowExecution(t *testing.T) {
//	    spec := testutil.BDD{
//	        TestID:      "WF-EXEC-001",
//	        Given:       "a valid workflow definition",
//	        When:        "the workflow is submitted",
//	        Then:        []string{"workflow starts successfully", "status is 'running'"},
//	        AcceptanceCriteria: []string{
//	            "AC1: Workflow ID is returned",
//	            "AC2: Start time is recorded",
//	        },
//	    }
//	    spec.Log(t)
//
//	    // test implementation...
//	}
type BDDSpec struct {
	// TestID is the unique identifier for this test (e.g., "WF-EXEC-001")
	TestID string

	// Given describes the initial context or preconditions
	Given string

	// When describes the action being tested
	When string

	// Then describes the expected outcomes (can be multiple)
	Then []string

	// AcceptanceCriteria lists the specific acceptance criteria being verified
	AcceptanceCriteria []string
}

// Log outputs the BDD specification to the test log.
func (b *BDDSpec) Log(t *testing.T) {
	t.Helper()

	var sb strings.Builder
	sb.WriteString("\n")

	if b.TestID != "" {
		sb.WriteString(fmt.Sprintf("Test ID: %s\n", b.TestID))
	}

	sb.WriteString(fmt.Sprintf("Given: %s\n", b.Given))
	sb.WriteString(fmt.Sprintf("When:  %s\n", b.When))

	if len(b.Then) == 1 {
		sb.WriteString(fmt.Sprintf("Then:  %s\n", b.Then[0]))
	} else {
		sb.WriteString("Then:\n")
		for _, then := range b.Then {
			sb.WriteString(fmt.Sprintf("  - %s\n", then))
		}
	}

	if len(b.AcceptanceCriteria) > 0 {
		sb.WriteString("Acceptance Criteria:\n")
		for _, ac := range b.AcceptanceCriteria {
			sb.WriteString(fmt.Sprintf("  - %s\n", ac))
		}
	}

	t.Log(sb.String())
}

// String returns the BDD specification as a formatted string.
func (b *BDDSpec) String() string {
	var sb strings.Builder

	if b.TestID != "" {
		sb.WriteString(fmt.Sprintf("Test ID: %s\n", b.TestID))
	}

	sb.WriteString(fmt.Sprintf("Given: %s\n", b.Given))
	sb.WriteString(fmt.Sprintf("When:  %s\n", b.When))

	if len(b.Then) == 1 {
		sb.WriteString(fmt.Sprintf("Then:  %s\n", b.Then[0]))
	} else {
		sb.WriteString("Then:\n")
		for _, then := range b.Then {
			sb.WriteString(fmt.Sprintf("  - %s\n", then))
		}
	}

	return sb.String()
}

// TestIDFormat defines the standard format for test IDs.
// Format: {Epic}.{Story}-{TestType}-{Sequence}
//
// Examples:
//   - 1.3-E2E-001  : Epic 1, Story 3, E2E test #1
//   - 2.1-UNIT-005 : Epic 2, Story 1, Unit test #5
//   - NFR-PERF-001 : Non-functional requirement, Performance test #1
//
// Test Types:
//   - UNIT: Unit tests
//   - INT:  Integration tests
//   - E2E:  End-to-end tests
//   - PERF: Performance tests
//   - SEC:  Security tests
//   - STRESS: Stress tests
const TestIDFormat = "{Epic}.{Story}-{TestType}-{Sequence}"

// GenerateTestID creates a test ID following the standard format.
//
// Usage:
//
//	testID := testutil.GenerateTestID("1", "3", "E2E", 1)
//	// Returns: "1.3-E2E-001"
func GenerateTestID(epic, story, testType string, sequence int) string {
	if story == "" {
		// For NFR or standalone epics
		return fmt.Sprintf("%s-%s-%03d", epic, testType, sequence)
	}
	return fmt.Sprintf("%s.%s-%s-%03d", epic, story, testType, sequence)
}

// Common test type constants for consistency.
const (
	TestTypeUnit   = "UNIT"
	TestTypeInt    = "INT"
	TestTypeE2E    = "E2E"
	TestTypePerf   = "PERF"
	TestTypeSec    = "SEC"
	TestTypeStress = "STRESS"
)

// Common epic prefixes.
const (
	EpicNFR      = "NFR"    // Non-functional requirements
	EpicWorkflow = "WF"     // Workflow related
	EpicAgent    = "AGENT"  // Agent related
	EpicPlugin   = "PLUGIN" // Plugin related
	EpicAPI      = "API"    // API related
)
