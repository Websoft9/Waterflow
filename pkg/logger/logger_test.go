package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name  string
		level string
		want  zapcore.Level
	}{
		{"debug level", "debug", zapcore.DebugLevel},
		{"info level", "info", zapcore.InfoLevel},
		{"warn level", "warn", zapcore.WarnLevel},
		{"error level", "error", zapcore.ErrorLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Init(tt.level, "json")
			require.NoError(t, err)
			assert.NotNil(t, Log)
		})
	}
}

func TestJSONFormat(t *testing.T) {
	err := Init("info", "json")
	require.NoError(t, err)
	assert.NotNil(t, Log)
}

func TestTextFormat(t *testing.T) {
	err := Init("info", "text")
	require.NoError(t, err)
	assert.NotNil(t, Log)
}

func TestContextFields(t *testing.T) {
	err := Init("info", "json")
	require.NoError(t, err)

	// Test logging with context fields
	Log.Info("test message",
		zap.String("component", "test"),
		zap.Int("port", 8080),
	)
	// If no panic, test passes
}

func TestInvalidLevel(t *testing.T) {
	err := Init("invalid", "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid log level")
}

func TestSync(t *testing.T) {
	err := Init("info", "json")
	require.NoError(t, err)

	// Should not panic
	_ = Sync() // Ignore return value in test
	// Sync may return error on stdout/stderr, which is expected
	// We just verify it doesn't panic
}
