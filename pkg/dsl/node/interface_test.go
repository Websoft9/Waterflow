package node

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockNode is a test helper implementing the full Node interface.
type MockNode struct {
	NameValue     string
	VersionValue  string
	ParamsValue   map[string]ParamSpec
	MetadataValue NodeMetadata
	ExecuteFunc   func(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error)
}

func (m *MockNode) Name() string {
	return m.NameValue
}

func (m *MockNode) Version() string {
	return m.VersionValue
}

func (m *MockNode) Params() map[string]ParamSpec {
	return m.ParamsValue
}

func (m *MockNode) Execute(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, inputs)
	}
	result := NewNodeResult()
	result.SetOutput("default", "executed")
	return result, nil
}

func (m *MockNode) Metadata() NodeMetadata {
	return m.MetadataValue
}

func TestNode_Interface(t *testing.T) {
	node := &MockNode{
		NameValue:    "flow/test",
		VersionValue: "v1",
		ParamsValue: map[string]ParamSpec{
			"message": {Type: "string", Required: true},
		},
		MetadataValue: NodeMetadata{
			Description: "Test node",
			Category:    "flow",
			InputSchema: map[string]ParamSpec{
				"message": {Type: "string", Required: true},
			},
			OutputSchema: map[string]interface{}{
				"result": "string",
			},
		},
	}

	assert.Equal(t, "flow/test", node.Name())
	assert.Equal(t, "v1", node.Version())
	assert.NotNil(t, node.Params())
	assert.NotNil(t, node.Metadata())
}

func TestNode_Execute(t *testing.T) {
	tests := []struct {
		name        string
		inputs      map[string]interface{}
		setupFunc   func() *MockNode
		wantOutputs map[string]interface{}
		wantErr     bool
	}{
		{
			name: "successful execution",
			inputs: map[string]interface{}{
				"message": "hello",
			},
			setupFunc: func() *MockNode {
				return &MockNode{
					NameValue:    "flow/echo",
					VersionValue: "v1",
					ExecuteFunc: func(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error) {
						result := NewNodeResult()
						result.SetOutput("message", inputs["message"])
						result.AddLog("echoed message")
						return result, nil
					},
				}
			},
			wantOutputs: map[string]interface{}{
				"message": "hello",
			},
			wantErr: false,
		},
		{
			name: "context cancellation",
			inputs: map[string]interface{}{
				"value": 42,
			},
			setupFunc: func() *MockNode {
				return &MockNode{
					NameValue:    "flow/test",
					VersionValue: "v1",
					ExecuteFunc: func(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error) {
						select {
						case <-ctx.Done():
							return nil, ctx.Err()
						case <-time.After(1 * time.Second):
							result := NewNodeResult()
							result.SetOutput("value", inputs["value"])
							return result, nil
						}
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := tt.setupFunc()
			ctx := context.Background()
			if tt.wantErr && tt.name == "context cancellation" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel() // Cancel immediately
			}

			result, err := node.Execute(ctx, tt.inputs)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			for key, expected := range tt.wantOutputs {
				assert.Equal(t, expected, result.Outputs[key])
			}
		})
	}
}

func TestNode_Metadata(t *testing.T) {
	metadata := NodeMetadata{
		Description: "Test node for unit testing",
		Category:    "flow",
		InputSchema: map[string]ParamSpec{
			"command": {
				Type:        "string",
				Required:    true,
				Description: "Command to execute",
			},
			"timeout": {
				Type:        "int",
				Required:    false,
				Default:     60,
				Description: "Timeout in seconds",
			},
		},
		OutputSchema: map[string]interface{}{
			"stdout":    "string",
			"exit_code": "int",
		},
	}

	node := &MockNode{
		NameValue:     "exec/shell",
		VersionValue:  "v1",
		MetadataValue: metadata,
	}

	result := node.Metadata()
	assert.Equal(t, "Test node for unit testing", result.Description)
	assert.Equal(t, "flow", result.Category)
	assert.Len(t, result.InputSchema, 2)
	assert.Len(t, result.OutputSchema, 2)
}

func TestNode_ConcurrentExecution(t *testing.T) {
	node := &MockNode{
		NameValue:    "flow/concurrent",
		VersionValue: "v1",
		ExecuteFunc: func(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error) {
			// Simulate some work
			time.Sleep(10 * time.Millisecond)
			result := NewNodeResult()
			result.SetOutput("value", inputs["value"])
			return result, nil
		},
	}

	const concurrency = 10
	results := make(chan *NodeResult, concurrency)
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			result, err := node.Execute(context.Background(), map[string]interface{}{
				"value": id,
			})
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}(i)
	}

	// Collect results
	for i := 0; i < concurrency; i++ {
		select {
		case result := <-results:
			assert.NotNil(t, result)
			assert.NotNil(t, result.Outputs["value"])
		case err := <-errors:
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestNodeResult_Methods(t *testing.T) {
	result := NewNodeResult()

	// Test SetOutput
	result.SetOutput("key1", "value1")
	result.SetOutput("key2", 42)
	assert.Equal(t, "value1", result.Outputs["key1"])
	assert.Equal(t, 42, result.Outputs["key2"])

	// Test AddLog
	result.AddLog("log message 1")
	result.AddLog("log message 2")
	assert.Len(t, result.Logs, 2)
	assert.Equal(t, "log message 1", result.Logs[0])
	assert.Equal(t, "log message 2", result.Logs[1])

	// Test SetMetadata
	result.SetMetadata("cpu_usage", 45.2)
	result.SetMetadata("memory_mb", 128)
	assert.Equal(t, 45.2, result.Metadata["cpu_usage"])
	assert.Equal(t, 128, result.Metadata["memory_mb"])

	// Test Duration
	result.Duration = 250 * time.Millisecond
	assert.Equal(t, 250*time.Millisecond, result.Duration)
}

func TestNodeMetadata_Methods(t *testing.T) {
	metadata := NewNodeMetadata("Test node", "exec")

	// Test initialization
	assert.Equal(t, "Test node", metadata.Description)
	assert.Equal(t, "exec", metadata.Category)
	assert.NotNil(t, metadata.InputSchema)
	assert.NotNil(t, metadata.OutputSchema)

	// Test AddInputParam
	metadata.AddInputParam("command", ParamSpec{
		Type:        "string",
		Required:    true,
		Description: "Shell command",
	})
	assert.Len(t, metadata.InputSchema, 1)
	assert.True(t, metadata.InputSchema["command"].Required)

	// Test AddOutputParam
	metadata.AddOutputParam("stdout", map[string]interface{}{
		"type": "string",
		"desc": "Standard output",
	})
	assert.Len(t, metadata.OutputSchema, 1)
	assert.NotNil(t, metadata.OutputSchema["stdout"])
}

func TestNode_ExecuteWithTimeout(t *testing.T) {
	node := &MockNode{
		NameValue:    "flow/slow",
		VersionValue: "v1",
		ExecuteFunc: func(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error) {
			select {
			case <-time.After(2 * time.Second):
				result := NewNodeResult()
				result.SetOutput("done", true)
				return result, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := node.Execute(ctx, map[string]interface{}{})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestNode_ExecuteReturnsAllResultFields(t *testing.T) {
	node := &MockNode{
		NameValue:    "flow/complete",
		VersionValue: "v1",
		ExecuteFunc: func(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error) {
			start := time.Now()
			time.Sleep(50 * time.Millisecond)

			result := NewNodeResult()
			result.SetOutput("status", "success")
			result.SetOutput("code", 0)
			result.AddLog("Starting execution")
			result.AddLog("Completed successfully")
			result.SetMetadata("execution_time_ms", time.Since(start).Milliseconds())
			result.Duration = time.Since(start)

			return result, nil
		},
	}

	result, err := node.Execute(context.Background(), map[string]interface{}{})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify all fields are populated
	assert.NotEmpty(t, result.Outputs)
	assert.Equal(t, "success", result.Outputs["status"])
	assert.Equal(t, 0, result.Outputs["code"])

	assert.Len(t, result.Logs, 2)
	assert.Contains(t, result.Logs[0], "Starting")
	assert.Contains(t, result.Logs[1], "Completed")

	assert.NotEmpty(t, result.Metadata)
	assert.Greater(t, result.Metadata["execution_time_ms"], int64(0))

	assert.Greater(t, result.Duration, time.Duration(0))
}
