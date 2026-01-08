package logs_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStdoutLogHandler(t *testing.T) {
	handler := logs.NewStdoutLogHandler(logs.StdoutConfig{
		BufferSize: 2,
		Pretty:     false,
	})
	defer handler.Close()

	ctx := context.Background()

	err := handler.OnLog(ctx, &logs.LogEntry{
		Timestamp:  time.Now(),
		Level:      logs.LogLevelInfo,
		WorkflowID: "test-workflow-1",
		Message:    "Test message 1",
	})
	assert.NoError(t, err)

	err = handler.OnLog(ctx, &logs.LogEntry{
		Timestamp:  time.Now(),
		Level:      logs.LogLevelError,
		WorkflowID: "test-workflow-1",
		JobID:      "job-1",
		Message:    "Test error message",
		Metadata: map[string]interface{}{
			"exit_code": 1,
		},
	})
	assert.NoError(t, err)
}

func TestFileLogHandler(t *testing.T) {
	tmpFile := "/tmp/waterflow-test-logs.jsonl"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + ".old")

	handler, err := logs.NewFileLogHandler(logs.FileConfig{
		Path:       tmpFile,
		BufferSize: 2,
		MaxSizeMB:  1,
	})
	require.NoError(t, err)
	defer handler.Close()

	ctx := context.Background()

	err = handler.OnLog(ctx, &logs.LogEntry{
		Timestamp:  time.Now(),
		Level:      logs.LogLevelInfo,
		WorkflowID: "test-workflow-1",
		Message:    "File test message 1",
	})
	assert.NoError(t, err)

	err = handler.OnLog(ctx, &logs.LogEntry{
		Timestamp:  time.Now(),
		Level:      logs.LogLevelInfo,
		WorkflowID: "test-workflow-1",
		Message:    "File test message 2",
	})
	assert.NoError(t, err)

	err = handler.Close()
	assert.NoError(t, err)

	data, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), "File test message 1")
	assert.Contains(t, string(data), "File test message 2")
}
