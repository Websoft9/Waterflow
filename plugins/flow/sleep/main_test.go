package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSleepNode_Name verifies the node name
func TestSleepNode_Name(t *testing.T) {
	node := &SleepNode{}
	assert.Equal(t, "flow/sleep", node.Name())
}

// TestSleepNode_Version verifies the node version
func TestSleepNode_Version(t *testing.T) {
	node := &SleepNode{}
	assert.Equal(t, "v1", node.Version())
}

// TestSleepNode_Metadata verifies the node metadata
func TestSleepNode_Metadata(t *testing.T) {
	node := &SleepNode{}
	metadata := node.Metadata()

	assert.Equal(t, "flow", metadata.Category)
	assert.Contains(t, metadata.Description, "Sleep")
	assert.NotEmpty(t, metadata.InputSchema)
	assert.NotEmpty(t, metadata.OutputSchema)

	// Verify duration parameter
	durationParam, ok := metadata.InputSchema["duration"]
	require.True(t, ok, "duration parameter should exist")
	assert.Equal(t, "string", durationParam.Type)
	assert.True(t, durationParam.Required)
	assert.NotEmpty(t, durationParam.Pattern)
}

// TestSleepNode_Params verifies the params method
func TestSleepNode_Params(t *testing.T) {
	node := &SleepNode{}
	params := node.Params()

	assert.NotEmpty(t, params)
	assert.Contains(t, params, "duration")
}

// TestSleepNode_Execute_BasicSleep tests basic sleep functionality
func TestSleepNode_Execute_BasicSleep(t *testing.T) {
	node := &SleepNode{}

	inputs := map[string]interface{}{
		"duration": "1s", // 1 second (minimum allowed)
	}

	start := time.Now()
	result, err := node.Execute(context.Background(), inputs)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Outputs["completed"].(bool))
	assert.InDelta(t, 1000, elapsed.Milliseconds(), 50) // ±50ms tolerance
	assert.Greater(t, result.Duration.Milliseconds(), int64(0))
	assert.NotEmpty(t, result.Logs)
}

// TestSleepNode_Execute_Cancelled tests context cancellation
func TestSleepNode_Execute_Cancelled(t *testing.T) {
	node := &SleepNode{}

	inputs := map[string]interface{}{
		"duration": "10s", // Long sleep
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	result, err := node.Execute(ctx, inputs)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Outputs["completed"].(bool)) // Not completed
	assert.Less(t, elapsed.Milliseconds(), int64(200))  // Quick return
	assert.Contains(t, result.Logs[len(result.Logs)-1], "cancelled")
}

// TestParseDuration_Seconds tests second-based durations
func TestParseDuration_Seconds(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"30s", 30 * time.Second},
		{"60s", 60 * time.Second},
		{"1s", 1 * time.Second},
		{"30", 30 * time.Second}, // Pure number defaults to seconds
		{"1", 1 * time.Second},
		{"120", 120 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d, err := parseDuration(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, d)
		})
	}
}

// TestParseDuration_Minutes tests minute-based durations
func TestParseDuration_Minutes(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"1m", 1 * time.Minute},
		{"5m", 5 * time.Minute},
		{"30m", 30 * time.Minute},
		{"60m", 60 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d, err := parseDuration(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, d)
		})
	}
}

// TestParseDuration_Hours tests hour-based durations
func TestParseDuration_Hours(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"1h", 1 * time.Hour},
		{"2h", 2 * time.Hour},
		{"24h", 24 * time.Hour},
		{"12h", 12 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d, err := parseDuration(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, d)
		})
	}
}

// TestParseDuration_Combined tests combined duration formats
func TestParseDuration_Combined(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"1h30m", 90 * time.Minute},
		{"2h15m", 135 * time.Minute},
		{"30m45s", 30*time.Minute + 45*time.Second},
		{"1h30m45s", 1*time.Hour + 30*time.Minute + 45*time.Second},
		{"2h0m30s", 2*time.Hour + 30*time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d, err := parseDuration(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, d)
		})
	}
}

// TestParseDuration_Invalid tests invalid duration formats
func TestParseDuration_Invalid(t *testing.T) {
	tests := []string{
		"",
		"abc",
		"1x",
		"30 s", // Space not allowed
		"s30",  // Wrong order
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := parseDuration(input)
			assert.Error(t, err)
		})
	}
}

// TestSleepNode_Execute_ValidationError tests validation errors
func TestSleepNode_Execute_ValidationError(t *testing.T) {
	node := &SleepNode{}

	tests := []struct {
		name   string
		inputs map[string]interface{}
		errMsg string
	}{
		{
			name:   "missing duration",
			inputs: map[string]interface{}{},
			errMsg: "duration is required",
		},
		{
			name:   "empty duration",
			inputs: map[string]interface{}{"duration": ""},
			errMsg: "duration is required",
		},
		{
			name:   "invalid format",
			inputs: map[string]interface{}{"duration": "abc"},
			errMsg: "invalid duration",
		},
		{
			name:   "too short",
			inputs: map[string]interface{}{"duration": "0s"},
			errMsg: "at least 1 second",
		},
		{
			name:   "negative duration",
			inputs: map[string]interface{}{"duration": "-1s"},
			errMsg: "at least 1 second",
		},
		{
			name:   "too long",
			inputs: map[string]interface{}{"duration": "25h"},
			errMsg: "not exceed 24 hours",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := node.Execute(context.Background(), tt.inputs)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

// TestSleepNode_Execute_OutputFields tests all output fields
func TestSleepNode_Execute_OutputFields(t *testing.T) {
	node := &SleepNode{}

	inputs := map[string]interface{}{
		"duration": "1s",
	}

	result, err := node.Execute(context.Background(), inputs)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check all output fields exist
	assert.Contains(t, result.Outputs, "duration_seconds")
	assert.Contains(t, result.Outputs, "completed")
	assert.Contains(t, result.Outputs, "elapsed_ms")

	// Validate values
	assert.Equal(t, 1, result.Outputs["duration_seconds"].(int))
	assert.True(t, result.Outputs["completed"].(bool))
	assert.Greater(t, result.Outputs["elapsed_ms"].(int64), int64(900)) // At least 900ms
}

// TestSleepNode_ConcurrentSafe tests concurrent execution safety
func TestSleepNode_ConcurrentSafe(t *testing.T) {
	node := &SleepNode{}

	// Run multiple goroutines concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			inputs := map[string]interface{}{
				"duration": "1s",
			}
			result, err := node.Execute(context.Background(), inputs)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// OK
		case <-time.After(15 * time.Second):
			t.Fatal("timeout waiting for concurrent executions")
		}
	}
}

// TestSleepNode_DifferentDurations tests various duration values
func TestSleepNode_DifferentDurations(t *testing.T) {
	node := &SleepNode{}

	tests := []struct {
		duration string
		minMS    int64
		maxMS    int64
	}{
		{"1s", 980, 1100},
		{"2s", 1980, 2100},
		{"3s", 2980, 3100},
	}

	for _, tt := range tests {
		t.Run(tt.duration, func(t *testing.T) {
			inputs := map[string]interface{}{
				"duration": tt.duration,
			}

			start := time.Now()
			result, err := node.Execute(context.Background(), inputs)
			elapsed := time.Since(start)

			require.NoError(t, err)
			assert.True(t, result.Outputs["completed"].(bool))
			assert.GreaterOrEqual(t, elapsed.Milliseconds(), tt.minMS)
			assert.LessOrEqual(t, elapsed.Milliseconds(), tt.maxMS)
		})
	}
}

// TestSleepNode_Logs tests that logs are properly generated
func TestSleepNode_Logs(t *testing.T) {
	node := &SleepNode{}

	t.Run("completed logs", func(t *testing.T) {
		inputs := map[string]interface{}{"duration": "1s"}
		result, err := node.Execute(context.Background(), inputs)

		require.NoError(t, err)
		assert.Len(t, result.Logs, 2)
		assert.Contains(t, result.Logs[0], "Sleep started")
		assert.Contains(t, result.Logs[1], "completed")
	})

	t.Run("cancelled logs", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		inputs := map[string]interface{}{"duration": "10s"}
		result, err := node.Execute(ctx, inputs)

		require.NoError(t, err)
		assert.Len(t, result.Logs, 2)
		assert.Contains(t, result.Logs[0], "Sleep started")
		assert.Contains(t, result.Logs[1], "cancelled")
	})
}

// TestRegister tests the Register function
func TestRegister(t *testing.T) {
	node := Register()
	require.NotNil(t, node)
	assert.Equal(t, "flow/sleep", node.Name())
	assert.Equal(t, "v1", node.Version())
}

// TestParseDuration_EdgeCases tests additional edge cases for full coverage
func TestParseDuration_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"valid milliseconds", "100ms", 100 * time.Millisecond, false},
		{"valid microseconds", "500us", 500 * time.Microsecond, false},
		{"valid nanoseconds", "1000ns", 1000 * time.Nanosecond, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := parseDuration(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, d)
			}
		})
	}
}
