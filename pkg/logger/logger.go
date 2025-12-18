// Package logger provides structured logging functionality using zap.
package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log is the global logger instance.
var Log *zap.Logger

// Init initializes the global logger with the specified level and format.
// Level should be one of: debug, info, warn, error
// Format should be one of: json, text
func Init(level string, format string) error {
	var cfg zap.Config

	// Select config based on format
	if format == "json" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Parse and set log level
	zapLevel, err := parseLevel(level)
	if err != nil {
		return err
	}
	cfg.Level = zap.NewAtomicLevelAt(zapLevel)

	// Build logger
	logger, err := cfg.Build()
	if err != nil {
		return fmt.Errorf("failed to build logger: %w", err)
	}

	Log = logger
	return nil
}

// parseLevel converts string level to zapcore.Level.
func parseLevel(level string) (zapcore.Level, error) {
	switch level {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("invalid log level: %s", level)
	}
}

// Sync flushes any buffered log entries.
// Should be called before application exit.
func Sync() error {
	if Log != nil {
		return Log.Sync()
	}
	return nil
}
