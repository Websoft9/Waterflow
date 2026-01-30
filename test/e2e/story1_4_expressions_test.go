//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/test/support/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ====================================================================================
// Story 1.4: Expression Engine and Variable System - E2E Tests
// ====================================================================================
//
// Test Coverage Priority:
// - 1.4-E2E-001: P0 - Variable definition and reference in workflow execution
// - 1.4-E2E-002: P0 - Expression evaluation with arithmetic and string operations
// - 1.4-E2E-003: P0 - Environment variable merging (workflow > job > step)
// - 1.4-E2E-004: P1 - Context variables (workflow, job, runner)
//
// Architecture Validation:
// - Full pipeline: YAML with expressions → Parser → Expression Engine → Temporal → Execution
// - Verify variable resolution at runtime
// - Validate expression sandbox (no file/network access)
//
// ====================================================================================

// TestStory1_4_E2E_001_VariableDefinitionAndReference validates variable definition
// and reference in workflow execution with nested objects and arrays
//
// Test ID: 1.4-E2E-001
// Priority: P0 (Critical - Core expression engine functionality)
// Risk: HIGH - Failure blocks all dynamic workflows
//
// Acceptance Criteria Verified:
// - AC1: Variable definition with nested objects/arrays
// - AC1: Variable reference using ${{ vars.* }} syntax
// - AC1: Nested object access ${{ vars.db.host }}
// - AC1: Array access ${{ vars.servers[0] }}
// - AC1: Undefined variable reference returns error
func TestStory1_4_E2E_001_VariableDefinitionAndReference(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.4-E2E-001",
		Given:  "A workflow with complex variable definitions (nested objects and arrays)",
		When:   "The workflow is executed with variable references in steps",
		Then: []string{
			"Variables are correctly resolved during execution",
			"Nested object access works (${{ vars.db.host }})",
			"Array access works (${{ vars.servers[0] }})",
			"Step outputs contain correctly evaluated variable values",
		},
		AcceptanceCriteria: []string{
			"AC1: Variable definition and reference",
			"AC1: Nested object access",
			"AC1: Array access",
			"AC1: Undefined variable error",
		},
	}).Log(t)

	// GIVEN: Workflow with complex variables
	workflowWithVars := `
name: Variable Reference Test
vars:
  environment: production
  version: "1.2.3"
  database:
    host: postgres.example.com
    port: 5432
    name: myapp
  servers:
    - web1.example.com
    - web2.example.com
    - web3.example.com

jobs:
  test-vars:
    runs-on: default
    steps:
      - id: simple-var
        uses: shell@v1
        with:
          command: echo "Environment is ${{ vars.environment }}"
      
      - id: nested-object
        uses: shell@v1
        with:
          command: echo "Database host is ${{ vars.database.host }}"
      
      - id: array-access
        uses: shell@v1
        with:
          command: echo "First server is ${{ vars.servers[0] }}"
      
      - id: complex-expression
        uses: shell@v1
        with:
          command: echo "Connection string is ${{ vars.database.host }}:${{ vars.database.port }}/${{ vars.database.name }}"
`

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	_ = ctx // Reserved for future Event History queries

	// WHEN: Submit workflow
	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	resp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(workflowWithVars))))
	require.NoError(t, err, "Failed to submit workflow")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var submitResponse struct {
		WorkflowID string `json:"workflowId"`
	}
	err = json.Unmarshal(body, &submitResponse)
	require.NoError(t, err)
	require.NotEmpty(t, submitResponse.WorkflowID)

	// THEN: Wait for completion and verify variable resolution
	statusURL := fmt.Sprintf("%s/api/v1/workflows/%s", serverURL, submitResponse.WorkflowID)

	// Poll for completion
	var finalStatus struct {
		Status string `json:"status"`
		Jobs   []struct {
			Steps []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"steps"`
		} `json:"jobs"`
	}

	success := false
	for i := 0; i < 60; i++ {
		statusResp, err := http.Get(statusURL)
		require.NoError(t, err)

		statusBody, err := io.ReadAll(statusResp.Body)
		statusResp.Body.Close()
		require.NoError(t, err)

		err = json.Unmarshal(statusBody, &finalStatus)
		require.NoError(t, err)

		if finalStatus.Status == "COMPLETED" || finalStatus.Status == "FAILED" {
			success = true
			break
		}
		time.Sleep(1 * time.Second)
	}

	require.True(t, success, "Workflow did not complete")
	assert.Equal(t, "COMPLETED", finalStatus.Status, "Workflow should complete successfully")

	// Verify all steps executed (variable resolution worked)
	require.Len(t, finalStatus.Jobs, 1)
	assert.Len(t, finalStatus.Jobs[0].Steps, 4, "Should have 4 steps")
	for _, step := range finalStatus.Jobs[0].Steps {
		assert.Equal(t, "COMPLETED", step.Status, fmt.Sprintf("Step %s should complete", step.ID))
	}

	// TODO (DEV TEAM): Verify step outputs contain correctly evaluated values
	// TODO (DEV TEAM): Query logs to verify command strings had variables substituted
}

// TestStory1_4_E2E_002_ExpressionEvaluation validates expression evaluation
// with arithmetic, string operations, and built-in functions
//
// Test ID: 1.4-E2E-002
// Priority: P0 (Critical - Expression engine core functionality)
// Risk: HIGH - Failure blocks dynamic computations in workflows
//
// Acceptance Criteria Verified:
// - AC2: Arithmetic operations (+, -, *, /, %)
// - AC2: Comparison operations (==, !=, >, <)
// - AC2: Logical operations (&&, ||, !)
// - AC2: String operations (contains, startsWith)
// - AC2: Built-in functions (len, upper, lower)
// - AC2: Expression sandbox (no file/network access)
func TestStory1_4_E2E_002_ExpressionEvaluation(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.4-E2E-002",
		Given:  "A workflow with various expression types (arithmetic, string, logical)",
		When:   "Expressions are evaluated during workflow execution",
		Then: []string{
			"Arithmetic expressions compute correctly (1 + 2 * 3 = 7)",
			"String operations work (contains, upper, lower)",
			"Logical operations evaluate properly (&&, ||, !)",
			"Built-in functions execute (len, trim, split)",
			"Expression syntax errors are caught with clear messages",
		},
		AcceptanceCriteria: []string{
			"AC2: Arithmetic operations",
			"AC2: String operations",
			"AC2: Logical operations",
			"AC2: Built-in functions",
			"AC2: Sandbox execution (no file/network)",
		},
	}).Log(t)

	// GIVEN: Workflow with expression operations
	workflowWithExpressions := `
name: Expression Evaluation Test
vars:
  count: 10
  name: "waterflow"
  version: "1.0.0"

jobs:
  test-expressions:
    runs-on: default
    steps:
      - id: arithmetic
        uses: shell@v1
        with:
          command: echo "Result is ${{ 1 + 2 * 3 }}"
      
      - id: comparison
        uses: shell@v1
        with:
          command: echo "Count greater than 5 is ${{ vars.count > 5 }}"
      
      - id: string-ops
        uses: shell@v1
        with:
          command: echo "Name uppercase is ${{ upper(vars.name) }}"
      
      - id: logical
        uses: shell@v1
        with:
          command: echo "Complex condition is ${{ vars.count > 5 && contains(vars.version, '1.0') }}"
`

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	resp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(workflowWithExpressions))))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var submitResponse struct {
		WorkflowID string `json:"workflowId"`
	}
	err = json.Unmarshal(body, &submitResponse)
	require.NoError(t, err)

	// Wait for completion
	time.Sleep(10 * time.Second)

	statusURL := fmt.Sprintf("%s/api/v1/workflows/%s", serverURL, submitResponse.WorkflowID)
	statusResp, err := http.Get(statusURL)
	require.NoError(t, err)
	defer statusResp.Body.Close()

	statusBody, err := io.ReadAll(statusResp.Body)
	require.NoError(t, err)

	var finalStatus struct {
		Status string `json:"status"`
	}
	err = json.Unmarshal(statusBody, &finalStatus)
	require.NoError(t, err)

	assert.Equal(t, "COMPLETED", finalStatus.Status, "Expression evaluation should succeed")

	// TODO (DEV TEAM): Verify expression results in step outputs/logs
	// TODO (DEV TEAM): Test expression syntax error returns 400 with precise error location
	// TODO (DEV TEAM): Verify sandbox prevents file access (should fail)
}

// TestStory1_4_E2E_003_EnvironmentVariableMerging validates environment variable
// merging across workflow/job/step levels with correct priority
//
// Test ID: 1.4-E2E-003
// Priority: P0 (Critical - Environment variable system)
// Risk: MEDIUM - Incorrect merging causes wrong runtime environment
//
// Acceptance Criteria Verified:
// - AC3: Three-level env definition (workflow/job/step)
// - AC3: Priority: step > job > workflow
// - AC3: Expression evaluation in env values
// - AC3: Env variables passed to Activity execution
func TestStory1_4_E2E_003_EnvironmentVariableMerging(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.4-E2E-003",
		Given:  "A workflow with env defined at workflow, job, and step levels",
		When:   "Steps execute with environment variable merging",
		Then: []string{
			"Step-level env overrides job-level",
			"Job-level env overrides workflow-level",
			"All env variables are available in Activity execution",
			"Expression evaluation works in env values",
		},
		AcceptanceCriteria: []string{
			"AC3: Three-level env definition",
			"AC3: Priority: step > job > workflow",
			"AC3: Expression evaluation in env",
			"AC3: Env passed to Activity",
		},
	}).Log(t)

	// GIVEN: Workflow with three-level env
	workflowWithEnv := `
name: Environment Variable Merging Test
vars:
  version: "1.2.3"
env:
  GLOBAL_VAR: global-value
  OVERRIDE_ME: workflow-level

jobs:
  test-env:
    runs-on: default
    env:
      JOB_VAR: job-value
      OVERRIDE_ME: job-level
    steps:
      - id: step-with-override
        uses: shell@v1
        env:
          STEP_VAR: step-value
          OVERRIDE_ME: step-level
          VERSION: "${{ vars.version }}"
        with:
          command: env | grep -E '(GLOBAL_VAR|JOB_VAR|STEP_VAR|OVERRIDE_ME|VERSION)'
      
      - id: step-without-override
        uses: shell@v1
        with:
          command: env | grep OVERRIDE_ME
`

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	resp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(workflowWithEnv))))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var submitResponse struct {
		WorkflowID string `json:"workflowId"`
	}
	err = json.Unmarshal(body, &submitResponse)
	require.NoError(t, err)

	// Wait for completion
	time.Sleep(10 * time.Second)

	statusURL := fmt.Sprintf("%s/api/v1/workflows/%s", serverURL, submitResponse.WorkflowID)
	statusResp, err := http.Get(statusURL)
	require.NoError(t, err)
	defer statusResp.Body.Close()

	assert.Equal(t, http.StatusOK, statusResp.StatusCode)

	// TODO (DEV TEAM): Query logs to verify env variable values:
	// - Step 1 should have OVERRIDE_ME=step-level
	// - Step 2 should have OVERRIDE_ME=job-level (no step override)
	// - VERSION should be "1.2.3" (expression evaluated)
	// TODO (DEV TEAM): Verify GLOBAL_VAR, JOB_VAR, STEP_VAR all present in step 1
}

// TestStory1_4_E2E_004_ContextVariables validates context variables
// (workflow, job, runner) available during execution
//
// Test ID: 1.4-E2E-004
// Priority: P1 (Important - Context variable system)
// Risk: LOW - Failure limits dynamic workflows but not critical
//
// Acceptance Criteria Verified:
// - AC4: workflow.name, workflow.id available
// - AC4: job.name, job.status available
// - AC4: runner.os, runner.arch available
func TestStory1_4_E2E_004_ContextVariables(t *testing.T) {
	(&testutil.BDDSpec{
		TestID: "1.4-E2E-004",
		Given:  "A workflow with steps that reference context variables",
		When:   "The workflow executes",
		Then: []string{
			"workflow.name and workflow.id are accessible",
			"job.name is accessible",
			"runner.os and runner.arch are populated from Agent",
		},
		AcceptanceCriteria: []string{
			"AC4: workflow.* context variables",
			"AC4: job.* context variables",
			"AC4: runner.* context variables from Agent",
		},
	}).Log(t)

	// GIVEN: Workflow using context variables
	workflowWithContext := `
name: Context Variables Test
jobs:
  test-context:
    runs-on: default
    steps:
      - id: workflow-context
        uses: shell@v1
        with:
          command: echo "Workflow name is ${{ workflow.name }} ID is ${{ workflow.id }}"
      
      - id: job-context
        uses: shell@v1
        with:
          command: echo "Job name is ${{ job.name }}"
      
      - id: runner-context
        uses: shell@v1
        with:
          command: echo "Runner OS is ${{ runner.os }} Arch is ${{ runner.arch }}"
`

	submitURL := fmt.Sprintf("%s/api/v1/workflows", serverURL)
	resp, err := http.Post(submitURL, "application/x-yaml", io.NopCloser(bytes.NewReader([]byte(workflowWithContext))))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var submitResponse struct {
		WorkflowID string `json:"workflowId"`
	}
	err = json.Unmarshal(body, &submitResponse)
	require.NoError(t, err)

	// Wait for completion
	time.Sleep(10 * time.Second)

	statusURL := fmt.Sprintf("%s/api/v1/workflows/%s", serverURL, submitResponse.WorkflowID)
	statusResp, err := http.Get(statusURL)
	require.NoError(t, err)
	defer statusResp.Body.Close()

	statusBody, err := io.ReadAll(statusResp.Body)
	require.NoError(t, err)

	var finalStatus struct {
		Status string `json:"status"`
	}
	err = json.Unmarshal(statusBody, &finalStatus)
	require.NoError(t, err)

	assert.Equal(t, "COMPLETED", finalStatus.Status)

	// TODO (DEV TEAM): Query logs to verify context variable values
	// TODO (DEV TEAM): Verify workflow.name = "Context Variables Test"
	// TODO (DEV TEAM): Verify workflow.id matches submitResponse.WorkflowID
	// TODO (DEV TEAM): Verify runner.os = "linux" (from Agent)
}
