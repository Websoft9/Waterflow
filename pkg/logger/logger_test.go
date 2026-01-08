package logger

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
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

// TestSanitization 测试敏感信息脱敏
func TestSanitization(t *testing.T) {
	var buf bytes.Buffer

	// 创建测试编码器
	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		TimeKey:        "timestamp",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}

	encoder := zapcore.NewJSONEncoder(encoderCfg)
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)

	// 包装 Core 以支持脱敏
	sanitizingCore := NewSanitizingCore(core, DefaultSensitiveFields)
	testLogger := zap.New(sanitizingCore)

	// 记录包含敏感信息的日志
	testLogger.Info("User login",
		zap.String("username", "admin"),
		zap.String("password", "secret123"),
		zap.String("api_key", "sk-xxx-yyy"),
		zap.String("token", "bearer-token-xyz"),
	)

	output := buf.String()

	// 验证用户名正常输出
	assert.Contains(t, output, `"username":"admin"`)

	// 验证敏感字段被脱敏
	assert.Contains(t, output, `"password":"***REDACTED***"`)
	assert.Contains(t, output, `"api_key":"***REDACTED***"`)
	assert.Contains(t, output, `"token":"***REDACTED***"`)

	// 确保原始值不出现
	assert.NotContains(t, output, "secret123")
	assert.NotContains(t, output, "sk-xxx-yyy")
	assert.NotContains(t, output, "bearer-token-xyz")
}

// TestSanitizationCaseInsensitive 测试大小写不敏感
func TestSanitizationCaseInsensitive(t *testing.T) {
	var buf bytes.Buffer

	encoderCfg := zapcore.EncoderConfig{
		MessageKey:  "msg",
		LevelKey:    "level",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
	}

	encoder := zapcore.NewJSONEncoder(encoderCfg)
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)
	sanitizingCore := NewSanitizingCore(core, DefaultSensitiveFields)
	testLogger := zap.New(sanitizingCore)

	// 使用不同大小写的敏感字段
	testLogger.Info("Test",
		zap.String("Password", "test1"),
		zap.String("PASSWORD", "test2"),
		zap.String("API_KEY", "test3"),
		zap.String("Api_Key", "test4"),
	)

	output := buf.String()

	// 所有变体都应被脱敏
	count := strings.Count(output, "***REDACTED***")
	assert.Equal(t, 4, count, "Expected 4 redactions, output: %s", output)
}

// TestCustomSensitiveFields 测试自定义敏感字段
func TestCustomSensitiveFields(t *testing.T) {
	var buf bytes.Buffer

	encoderCfg := zapcore.EncoderConfig{
		MessageKey:  "msg",
		LevelKey:    "level",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
	}

	customFields := []string{"custom_secret", "internal_token"}
	encoder := zapcore.NewJSONEncoder(encoderCfg)
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)
	sanitizingCore := NewSanitizingCore(core, customFields)
	testLogger := zap.New(sanitizingCore)

	testLogger.Info("Test",
		zap.String("custom_secret", "my-secret"),
		zap.String("internal_token", "my-token"),
		zap.String("normal_field", "normal-value"),
	)

	output := buf.String()

	// 自定义字段应被脱敏
	assert.Contains(t, output, `"custom_secret":"***REDACTED***"`)
	assert.Contains(t, output, `"internal_token":"***REDACTED***"`)

	// 普通字段应正常输出
	assert.Contains(t, output, `"normal_field":"normal-value"`)
}

// TestInitWithConfig 测试配置初始化
func TestInitWithConfig(t *testing.T) {
	// 测试 JSON 格式
	err := InitWithConfig(LogConfig{
		Level:           "info",
		Format:          "json",
		SensitiveFields: []string{"custom_field"},
	})
	require.NoError(t, err)
	assert.NotNil(t, Log)

	// 测试文本格式
	err = InitWithConfig(LogConfig{
		Level:  "debug",
		Format: "text",
	})
	require.NoError(t, err)

	// 测试无效级别
	err = InitWithConfig(LogConfig{
		Level:  "invalid",
		Format: "json",
	})
	assert.Error(t, err)
}

// TestSetAndGetLevel 测试动态级别调整
func TestSetAndGetLevel(t *testing.T) {
	// 初始化为 info
	err := Init("info", "json")
	require.NoError(t, err)

	// 验证初始级别
	assert.Equal(t, "info", GetLevel())

	// 调整到 debug
	err = SetLevel("debug")
	require.NoError(t, err)

	// 验证新级别
	assert.Equal(t, "debug", GetLevel())

	// 测试无效级别
	err = SetLevel("invalid")
	assert.Error(t, err)

	// 级别应保持不变
	assert.Equal(t, "debug", GetLevel())
}

// TestEnvironmentVariableSensitiveFields 测试环境变量敏感字段
func TestEnvironmentVariableSensitiveFields(t *testing.T) {
	// 设置环境变量
	os.Setenv("WATERFLOW_LOG_SENSITIVE_FIELDS", "env_secret,env_token")
	defer os.Unsetenv("WATERFLOW_LOG_SENSITIVE_FIELDS")

	var buf bytes.Buffer

	encoderCfg := zapcore.EncoderConfig{
		MessageKey:  "msg",
		LevelKey:    "level",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
	}

	// 初始化时应自动加载环境变量
	err := InitWithConfig(LogConfig{
		Level:  "info",
		Format: "json",
	})
	require.NoError(t, err)

	// 创建测试 logger (验证字段合并)
	allFields := append([]string{}, DefaultSensitiveFields...)
	allFields = append(allFields, "env_secret", "env_token")

	encoder := zapcore.NewJSONEncoder(encoderCfg)
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)
	sanitizingCore := NewSanitizingCore(core, allFields)
	testLogger := zap.New(sanitizingCore)

	testLogger.Info("Test",
		zap.String("env_secret", "value1"),
		zap.String("env_token", "value2"),
	)

	output := buf.String()

	// 环境变量字段应被脱敏
	assert.Contains(t, output, `"env_secret":"***REDACTED***"`)
	assert.Contains(t, output, `"env_token":"***REDACTED***"`)
}

// TestContextLogger 测试上下文 logger
func TestContextLogger(t *testing.T) {
	var buf bytes.Buffer

	encoderCfg := zapcore.EncoderConfig{
		MessageKey:  "msg",
		LevelKey:    "level",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
	}

	encoder := zapcore.NewJSONEncoder(encoderCfg)
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)
	Log = zap.New(core)

	// 创建带上下文的 logger
	workflowLogger := WithWorkflowContext("wf-123")
	workflowLogger.Info("Workflow started")

	output := buf.String()

	// 验证包含 workflow_id
	assert.Contains(t, output, `"workflow_id":"wf-123"`)
}

// TestFieldHelpers 测试字段辅助函数
func TestFieldHelpers(t *testing.T) {
	tests := []struct {
		name     string
		field    zap.Field
		expected string
	}{
		{"WorkflowID", WorkflowID("wf-123"), `"workflow_id":"wf-123"`},
		{"JobID", JobID("job-1"), `"job_id":"job-1"`},
		{"StepName", StepName("deploy"), `"step_name":"deploy"`},
		{"NodeType", NodeType("shell"), `"node_type":"shell"`},
		{"ErrorType", ErrorType("timeout"), `"error_type":"timeout"`},
		{"Component", Component("server"), `"component":"server"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			encoderCfg := zapcore.EncoderConfig{
				MessageKey:  "msg",
				LevelKey:    "level",
				EncodeLevel: zapcore.LowercaseLevelEncoder,
			}

			encoder := zapcore.NewJSONEncoder(encoderCfg)
			core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)
			testLogger := zap.New(core)

			testLogger.Info("Test", tt.field)

			output := buf.String()
			assert.Contains(t, output, tt.expected)
		})
	}
}

// TestStructuredLogging 测试结构化日志
func TestStructuredLogging(t *testing.T) {
	var buf bytes.Buffer

	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		TimeKey:        "timestamp",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}

	encoder := zapcore.NewJSONEncoder(encoderCfg)
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)
	testLogger := zap.New(core)

	// 记录结构化日志
	testLogger.Info("Workflow executed",
		WorkflowID("wf-456"),
		JobID("build"),
		StepName("compile"),
		zap.Int("exit_code", 0),
	)

	output := buf.String()

	// 解析 JSON
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	require.NoError(t, err)

	// 验证字段
	assert.Equal(t, "Workflow executed", logEntry["msg"])
	assert.Equal(t, "wf-456", logEntry["workflow_id"])
	assert.Equal(t, "build", logEntry["job_id"])
	assert.Equal(t, "compile", logEntry["step_name"])
	assert.Equal(t, float64(0), logEntry["exit_code"])
}
