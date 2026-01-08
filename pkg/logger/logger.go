// Package logger provides structured logging functionality using zap.
package logger

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log is the global logger instance.
var Log *zap.Logger

// atomicLevel 支持动态调整日志级别
var atomicLevel zap.AtomicLevel

// LogConfig 日志配置
type LogConfig struct {
	Level           string   // 日志级别: debug, info, warn, error
	Format          string   // 格式: json, text
	SensitiveFields []string // 自定义敏感字段 (追加到默认列表)
}

// Init initializes the global logger with the specified level and format.
// Level should be one of: debug, info, warn, error
// Format should be one of: json, text
func Init(level string, format string) error {
	return InitWithConfig(LogConfig{
		Level:  level,
		Format: format,
	})
}

// InitWithConfig 使用完整配置初始化日志
func InitWithConfig(cfg LogConfig) error {
	var zapCfg zap.Config

	// Select config based on format
	if cfg.Format == "json" {
		zapCfg = zap.NewProductionConfig()
		// Configure encoder for ISO 8601 timestamp format (AC4)
		zapCfg.EncoderConfig.TimeKey = "timestamp"
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		zapCfg.EncoderConfig.LevelKey = "level"
		zapCfg.EncoderConfig.MessageKey = "msg"
		zapCfg.EncoderConfig.CallerKey = "caller"
	} else {
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		// Use ISO 8601 for text format too
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// Parse and set log level
	zapLevel, err := parseLevel(cfg.Level)
	if err != nil {
		return err
	}

	// 使用 AtomicLevel 支持动态调整
	atomicLevel = zap.NewAtomicLevelAt(zapLevel)
	zapCfg.Level = atomicLevel

	// 合并默认和自定义敏感字段
	sensitiveFields := append([]string{}, DefaultSensitiveFields...)
	sensitiveFields = append(sensitiveFields, cfg.SensitiveFields...)

	// 从环境变量加载额外的敏感字段
	if envFields := os.Getenv("WATERFLOW_LOG_SENSITIVE_FIELDS"); envFields != "" {
		extraFields := strings.Split(envFields, ",")
		for _, field := range extraFields {
			field = strings.TrimSpace(field)
			if field != "" {
				sensitiveFields = append(sensitiveFields, field)
			}
		}
	}

	// Build logger with sanitizing core
	logger, err := zapCfg.Build(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		// 包装 Core 以支持脱敏
		return NewSanitizingCore(core, sensitiveFields)
	}))

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

// SetLevel 动态设置日志级别
func SetLevel(level string) error {
	zapLevel, err := parseLevel(level)
	if err != nil {
		return err
	}

	atomicLevel.SetLevel(zapLevel)
	Log.Info("Log level changed", zap.String("new_level", level))

	return nil
}

// GetLevel 获取当前日志级别
func GetLevel() string {
	switch atomicLevel.Level() {
	case zapcore.DebugLevel:
		return "debug"
	case zapcore.InfoLevel:
		return "info"
	case zapcore.WarnLevel:
		return "warn"
	case zapcore.ErrorLevel:
		return "error"
	default:
		return "unknown"
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
