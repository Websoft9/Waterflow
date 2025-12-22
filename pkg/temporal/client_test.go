package temporal

import (
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestNewClient_Success(t *testing.T) {
	// Note: This test requires a running Temporal server at localhost:7233
	// Skip if Temporal is not available
	t.Skip("Requires running Temporal server - run manually")

	logger := zaptest.NewLogger(t)
	cfg := &config.TemporalConfig{
		Host:              "localhost:7233",
		Namespace:         "default",
		TaskQueue:         "test-queue",
		ConnectionTimeout: 10 * time.Second,
		MaxRetries:        3,
		RetryInterval:     time.Second,
	}

	client, err := NewClient(cfg, logger)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, client.GetClient())
	assert.Equal(t, cfg, client.GetConfig())

	client.Close()
}

func TestNewClient_ConnectionFailure(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.TemporalConfig{
		Host:              "localhost:9999", // Invalid port
		Namespace:         "default",
		TaskQueue:         "test-queue",
		ConnectionTimeout: time.Second,
		MaxRetries:        2,
		RetryInterval:     100 * time.Millisecond,
	}

	client, err := NewClient(cfg, logger)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to connect to Temporal after 2 attempts")
}

func TestTemporalLogger(t *testing.T) {
	logger := zaptest.NewLogger(t)
	tLogger := newTemporalLogger(logger)

	// Test logging methods don't panic
	tLogger.Debug("debug message", "key1", "value1")
	tLogger.Info("info message", "key2", "value2")
	tLogger.Warn("warn message", "key3", "value3")
	tLogger.Error("error message", "key4", "value4")
}

func TestConvertToZapFields(t *testing.T) {
	tests := []struct {
		name     string
		keyvals  []interface{}
		expected int
	}{
		{
			name:     "empty",
			keyvals:  []interface{}{},
			expected: 0,
		},
		{
			name:     "single pair",
			keyvals:  []interface{}{"key", "value"},
			expected: 1,
		},
		{
			name:     "multiple pairs",
			keyvals:  []interface{}{"key1", "value1", "key2", "value2"},
			expected: 2,
		},
		{
			name:     "odd number of elements",
			keyvals:  []interface{}{"key1", "value1", "key2"},
			expected: 1,
		},
		{
			name:     "non-string key",
			keyvals:  []interface{}{123, "value", "key2", "value2"},
			expected: 1, // Skip non-string key
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := convertToZapFields(tt.keyvals)
			assert.Len(t, fields, tt.expected)
		})
	}
}
