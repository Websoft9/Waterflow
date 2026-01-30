package dsl

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerializableEvalContext_JSONSerialization(t *testing.T) {
	// Create a full EvalContext with functions
	ctx := &EvalContext{
		Workflow: map[string]interface{}{"name": "test-workflow"},
		Job:      map[string]interface{}{"status": "success"},
		Steps:    map[string]interface{}{"step1": map[string]interface{}{"output": "value"}},
		Vars:     map[string]interface{}{"env": "production"},
		Env:      map[string]string{"PATH": "/usr/bin"},
		Matrix:   map[string]interface{}{"os": "linux", "version": "1.0"},
		Runner:   map[string]interface{}{"os": "linux"},
		Inputs:   map[string]interface{}{"input1": "value1"},
		Secrets:  map[string]string{"api_key": "secret123"},
		Needs:    map[string]interface{}{"job1": map[string]interface{}{"outputs": map[string]interface{}{}}},
	}

	// Register functions (these should NOT be serialized)
	funcs := GetBuiltinFunctions()
	ctx.Len = funcs["len"].(func(interface{}) (int, error))
	ctx.Upper = funcs["upper"].(func(string) string)

	// Convert to serializable version
	serializable := ctx.ToSerializable()

	// Test JSON serialization (should succeed)
	jsonData, err := json.Marshal(serializable)
	require.NoError(t, err, "SerializableEvalContext should be JSON serializable")
	assert.NotEmpty(t, jsonData)

	// Deserialize back
	var deserialized SerializableEvalContext
	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)

	// Verify data is preserved
	assert.Equal(t, "test-workflow", deserialized.Workflow["name"])
	assert.Equal(t, "success", deserialized.Job["status"])
	assert.Equal(t, "production", deserialized.Vars["env"])
	assert.Equal(t, "/usr/bin", deserialized.Env["PATH"])
	assert.Equal(t, "linux", deserialized.Matrix["os"])
}

func TestSerializableEvalContext_RoundTrip(t *testing.T) {
	// Original context with data
	original := &EvalContext{
		Workflow: map[string]interface{}{"name": "workflow1", "id": "w1"},
		Job:      map[string]interface{}{"status": "running", "id": "j1"},
		Steps:    map[string]interface{}{"setup": map[string]interface{}{"result": "ok"}},
		Vars:     map[string]interface{}{"version": "1.2.3"},
		Env:      map[string]string{"ENV": "prod"},
		Matrix:   map[string]interface{}{"platform": "linux"},
		Needs:    map[string]interface{}{},
	}

	// Convert to serializable and back
	serializable := original.ToSerializable()
	restored := serializable.ToEvalContext()

	// Verify data is preserved
	assert.Equal(t, original.Workflow, restored.Workflow)
	assert.Equal(t, original.Job, restored.Job)
	assert.Equal(t, original.Steps, restored.Steps)
	assert.Equal(t, original.Vars, restored.Vars)
	assert.Equal(t, original.Env, restored.Env)
	assert.Equal(t, original.Matrix, restored.Matrix)

	// Verify functions are re-registered
	assert.NotNil(t, restored.Len, "Len function should be registered")
	assert.NotNil(t, restored.Upper, "Upper function should be registered")
	assert.NotNil(t, restored.Format, "Format function should be registered")
	assert.NotNil(t, restored.Success, "Success function should be registered")
}

func TestSerializableEvalContext_ContextFunctions(t *testing.T) {
	serializable := &SerializableEvalContext{
		Job: map[string]interface{}{"status": "success"},
	}

	// Restore to EvalContext
	ctx := serializable.ToEvalContext()

	// Test that context-dependent functions work
	assert.NotNil(t, ctx.Success)
	assert.True(t, ctx.Success(), "Success should return true when status is 'success'")

	// Test static functions
	assert.NotNil(t, ctx.Always)
	assert.True(t, ctx.Always(), "Always should always return true")
}

func TestEvalContext_CannotSerialize(t *testing.T) {
	// Create EvalContext with functions
	ctx := &EvalContext{
		Workflow: map[string]interface{}{"name": "test"},
	}
	funcs := GetBuiltinFunctions()
	ctx.Len = funcs["len"].(func(interface{}) (int, error))

	// Attempting to serialize EvalContext directly should fail
	_, err := json.Marshal(ctx)
	assert.Error(t, err, "EvalContext with functions should NOT be JSON serializable")
	assert.Contains(t, err.Error(), "unsupported type", "Error should mention unsupported type")
}
