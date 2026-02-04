package temporal

import (
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestNewActivities(t *testing.T) {
	logger := zaptest.NewLogger(t)
	registry := node.NewRegistry()
	activities := NewActivities(logger, registry)

	assert.NotNil(t, activities)
	assert.NotNil(t, activities.logger)
	assert.NotNil(t, activities.nodeRegistry)
	assert.NotNil(t, activities.nodeTracker)
}

func TestActivities_Structure(t *testing.T) {
	logger := zap.NewNop()
	registry := node.NewRegistry()

	t.Run("activities_has_required_fields", func(t *testing.T) {
		activities := NewActivities(logger, registry)
		
		// Verify all fields are initialized
		assert.NotNil(t, activities.logger, "logger should be initialized")
		assert.NotNil(t, activities.nodeRegistry, "nodeRegistry should be initialized")
		assert.NotNil(t, activities.nodeTracker, "nodeTracker should be initialized")
	})
}

func TestExecuteStepInput_Structure(t *testing.T) {
	t.Run("input_serialization", func(t *testing.T) {
		// Test that ExecuteStepInput only contains serializable types
		input := ExecuteStepInput{
			WorkflowName: "test-workflow",
			JobName:      "test-job",
			StepName:     "test-step",
			StepUses:     "exec/shell@v1",
			StepWith: map[string]interface{}{
				"command": "echo hello",
			},
			StepEnv: map[string]string{
				"FOO": "bar",
			},
			StepIf: "success()",
			Context: &dsl.SerializableEvalContext{
				Workflow: map[string]interface{}{"name": "test"},
				Job:      map[string]interface{}{"name": "job1"},
				Steps:    map[string]interface{}{},
				Vars:     map[string]interface{}{"key": "value"},
				Env:      map[string]string{"ENV": "test"},
				Matrix:   map[string]interface{}{},
				Runner:   map[string]interface{}{},
				Inputs:   map[string]interface{}{},
				Secrets:  map[string]string{},
				Needs:    map[string]interface{}{},
			},
		}

		// Verify all fields are set
		assert.Equal(t, "test-workflow", input.WorkflowName)
		assert.Equal(t, "test-job", input.JobName)
		assert.Equal(t, "test-step", input.StepName)
		assert.Equal(t, "exec/shell@v1", input.StepUses)
		assert.NotNil(t, input.StepWith)
		assert.NotNil(t, input.StepEnv)
		assert.Equal(t, "success()", input.StepIf)
		assert.NotNil(t, input.Context)
	})
}

func TestStepResult_Structure(t *testing.T) {
	t.Run("result_status_types", func(t *testing.T) {
		// Test different status types
		statuses := []string{"success", "failure", "skipped", "timeout"}
		
		for _, status := range statuses {
			result := &StepResult{
				Status:     status,
				Outputs:    map[string]string{"output1": "value1"},
				Error:      "",
				DurationMs: 1234,
			}
			
			assert.Equal(t, status, result.Status)
			assert.NotNil(t, result.Outputs)
			assert.GreaterOrEqual(t, result.DurationMs, int64(0))
		}
	})
	
	t.Run("result_with_error", func(t *testing.T) {
		result := &StepResult{
			Status:     "failure",
			Outputs:    map[string]string{},
			Error:      "execution failed",
			DurationMs: 5678,
		}
		
		assert.Equal(t, "failure", result.Status)
		assert.NotEmpty(t, result.Error)
		assert.Equal(t, "execution failed", result.Error)
	})
}

func TestSerializableEvalContext_Reconstruction(t *testing.T) {
	t.Run("context_round_trip", func(t *testing.T) {
		// Create serializable context
		serializable := &dsl.SerializableEvalContext{
			Workflow: map[string]interface{}{"name": "test-wf"},
			Job:      map[string]interface{}{"name": "test-job", "status": "success"},
			Steps:    map[string]interface{}{},
			Vars:     map[string]interface{}{"var1": "value1"},
			Env:      map[string]string{"ENV1": "envvalue1"},
			Matrix:   map[string]interface{}{"os": "linux"},
			Runner:   map[string]interface{}{"name": "runner1"},
			Inputs:   map[string]interface{}{"input1": "inputvalue1"},
			Secrets:  map[string]string{"SECRET1": "secretvalue1"},
			Needs:    map[string]interface{}{"job1": map[string]interface{}{"result": "success"}},
		}

		// Convert to EvalContext
		evalCtx := serializable.ToEvalContext()

		// Verify all data fields are preserved
		assert.Equal(t, "test-wf", evalCtx.Workflow["name"])
		assert.Equal(t, "test-job", evalCtx.Job["name"])
		assert.Equal(t, "value1", evalCtx.Vars["var1"])
		assert.Equal(t, "envvalue1", evalCtx.Env["ENV1"])
		assert.Equal(t, "linux", evalCtx.Matrix["os"])

		// Verify functions are reconstructed
		assert.NotNil(t, evalCtx.Len)
		assert.NotNil(t, evalCtx.Upper)
		assert.NotNil(t, evalCtx.Lower)
		assert.NotNil(t, evalCtx.Success)
		assert.NotNil(t, evalCtx.Failure)
	})
}

// Note: ExecuteStepActivity error handling tests (NonRetryableError) are covered
// in workflow_test.go TestToTemporalRetryPolicy_NonRetryableErrorTypes which validates
// the NonRetryableErrorTypes list configuration.
//
// Full integration tests with actual node execution require a real Temporal server
// and are located in test/integration/.
